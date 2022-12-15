# dxy-prices

Public data collector for DXY Prices based on the https://finance.yahoo.com API.
A Github Action is scheduled to update the `prices.csv` file once a day. That
CSV file can be integrated via Github's Raw Data endpoint in various ways. One
way to use the raw data is to define a Grafana CSV Data Source using the plugin
https://grafana.com/grafana/plugins/marcusolsson-csv-datasource.

![Grafana](/asset/grafana.png)
