# spotted-zebra

A fixed coupon note pricer.

# Prerequisite
1. Create an environment file (.env) to store the API Keys in the main directory for accessing Polygon and AlphaVantage.

# API Server

Developed functions: Pricing

Target Stocks: AAPL, AMZN, META, MSFT, TSLA, GOOG, NVDA, AVGO, QCOM, INTC

Proxy: `localhost:8080`

# Authentication
You must add an Authorization header to the request with your API Key as the token in the following form
`Authorization: Bearer demo_key`

# Pricer
`POST` `/v1/pricer`

