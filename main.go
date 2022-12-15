package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/xh3b4sd/budget/v3"
	"github.com/xh3b4sd/budget/v3/pkg/breaker"
	"github.com/xh3b4sd/framer"
)

const (
	apifmt = "https://query1.finance.yahoo.com/v7/finance/download/DX-Y.NYB?period1=%d&period2=%d&interval=1d&events=history"
	dayzer = "2020-12-01T00:00:00Z"
	prifil = "prices.csv"
	reqlim = 50
)

type csvrow struct {
	Dat time.Time
	Pri float64
}

func main() {
	var err error

	var rea *os.File
	{
		rea, err = os.Open(prifil)
		if err != nil {
			log.Fatal(err)
		}
	}

	var row [][]string
	{
		row, err = csv.NewReader(rea).ReadAll()
		if err != nil {
			log.Fatal(err)
		}
	}

	{
		rea.Close()
	}

	cur := map[time.Time]float64{}
	for _, x := range row[1:] {
		cur[mustim(x[0])] = musf64(x[1])
	}

	var sta time.Time
	{
		sta = mustim(dayzer)
	}

	var end time.Time
	{
		end = time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.UTC)
	}

	var bud budget.Interface
	{
		bud = breaker.Default()
	}

	var fra *framer.Framer
	{
		fra = framer.New(framer.Config{
			Sta: sta,
			End: end,
			Len: 24 * time.Hour,
		})
	}

	var cou int
	des := map[time.Time]float64{}
	for _, x := range fra.List() {
		f64, exi := cur[x.Sta]
		if exi {
			{
				// log.Printf("setting cached prices for %s\n", x.Sta)
			}

			{
				des[x.Sta] = f64
			}
		} else if cou < reqlim {
			{
				cou++
			}

			{
				log.Printf("filling remote prices for %s\n", x.Sta)
			}

			var act func() error
			{
				act = func() error {
					var f64 float64
					{
						f64 = musapi(x.Sta)
					}

					if f64 == -1 {
						f64 = des[x.Sta.Add(-24*time.Hour)]
					}

					if f64 == 0 {
						return budget.Cancel
					}

					if cur[x.Sta] != 0 && cur[x.Sta] != f64 {
						des[x.Sta] = f64
					}

					return nil
				}
			}

			{
				err = bud.Execute(act)
				if budget.IsCancel(err) {
					break
				} else if budget.IsPassed(err) {
					break
				} else if err != nil {
					log.Fatal(err)
				}
			}

			{
				time.Sleep(200 * time.Millisecond)
			}
		}
	}

	var lis []csvrow
	for k, v := range des {
		lis = append(lis, csvrow{Dat: k, Pri: v})
	}

	{
		sort.SliceStable(lis, func(i, j int) bool { return lis[i].Dat.Before(lis[j].Dat) })
	}

	var res [][]string
	{
		res = append(res, []string{"date", "close"})
	}

	for _, x := range lis {
		res = append(res, []string{x.Dat.Format(time.RFC3339), fmt.Sprintf("%.16f", x.Pri)})
	}

	var wri *os.File
	{
		wri, err = os.OpenFile(prifil, os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.ModePerm)
		if err != nil {
			log.Fatal(err)
		}
	}

	{
		defer wri.Close()
	}

	{
		err = csv.NewWriter(wri).WriteAll(res)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func musapi(des time.Time) float64 {
	var err error

	var uni int64
	{
		uni = des.Add(12 * time.Hour).Unix()
	}

	var cli *http.Client
	{
		cli = &http.Client{Timeout: 10 * time.Second}
	}

	var res *http.Response
	{
		u := fmt.Sprintf(apifmt, uni, uni+1)

		res, err = cli.Get(u)
		if err != nil {
			log.Fatal(err)
		}
	}

	{
		defer res.Body.Close()
	}

	if res.StatusCode == http.StatusNotFound {
		return -1
	}

	var row [][]string
	{
		row, err = csv.NewReader(res.Body).ReadAll()
		if err != nil {
			log.Fatal(err)
		}
	}

	// We expect a CSV format response like shown below.
	//
	//     Date,Open,High,Low,Close,Adj Close,Volume
	//     2022-12-15,103.667999,104.405998,103.613998,104.127998,104.127998,0
	//
	if len(row) != 2 {
		return 0
	}
	if len(row[1]) != 7 {
		return 0
	}

	// Return the close price.
	return musf64(row[1][4])
}

func musf64(str string) float64 {
	f64, err := strconv.ParseFloat(str, 64)
	if err != nil {
		log.Fatal(err)
	}

	return f64
}

func mustim(str string) time.Time {
	tim, err := time.Parse(time.RFC3339, str)
	if err != nil {
		log.Fatal(err)
	}

	return tim
}
