# spotted-zebra

A fixed coupon note pricer.

# Prerequisite

1. Create an environment file (app.env) to store the API Keys in the main directory for accessing Polygon and AlphaVantage.

# API Server

Developed functions: Pricing

Target Stocks: AAPL, AMZN, META, MSFT, TSLA, GOOG, NVDA, AVGO, QCOM, INTC

Proxy: `localhost:8080`

# Authentication

You must add an Authorization header to the request with your API Key as the token in the following form
`Authorization: Bearer demo_key`

# Pricer

`POST` `/v1/pricer`

Add request body:

```
{
  "stocks": ["AAPL", "META", "MSFT", "TSLA", "AVGO"],
  "strike" : 0.80,
  "autocall_coupon_rate" : 0.10,
  "barrier_coupon_rate" : 0.20,
  "fixed_coupon_rate" : 0.20,
  "knock_out_barrier" : 1.05,
  "knock_in_barrier" : 0.70,
  "coupon_barrier" : 0.80,
  "maturity" : 3,
  "frequency" : 1,
  "isEuro" : false
}
```

Response Object:

```
{
  "msg": {
    "stocks": ["AAPL", "AVGO", "META", "MSFT", "TSLA"],
    "strike": 0.8,
    "autocall_coupon_rate": 0.1,
    "barrier_coupon_rate": 0.2,
    "fixed_coupon_rate": 0.2,
    "knock_out_barrier": 1.05,
    "knock_in_barrier": 0.7,
    "coupon_barrier": 0.8,
    "maturity": 3,
    "frequency": 1,
    "isEuro": false
  },
  "price": 0.8852390227964861,
  "status": 200
}
```
