package data

import "main/mc"

type TickersArr struct {
	Tickers []string `json:"tickers"`
}

type TickersPara struct {
	Ticker string `json:"ticker"`
}

type Tickers struct {
	Results []TickersPara `json:"results"`
	Next    string        `json:"next_url"`
}

type TickerData struct {
	ContractType      string  `json:"contract_type"`
	ExerciseStyle     string  `json:"exercise_style"`
	ExpirationDate    string  `json:"expiration_date"`
	SharesPerContract float64 `json:"shares_per_contract"`
	StrikePrice       float64 `json:"strike_price"`
	Ticker            string  `json:"ticker"`
}

type TickerDay struct {
	Change        float64 `json:"change"`
	ChangePercent float64 `json:"change_percent"`
	Close         float64 `json:"close"`
	High          float64 `json:"high"`
	LastUpdated   int64   `json:"last_updated"`
	Low           float64 `json:"low"`
	Open          float64 `json:"open"`
	PreviousClose float64 `json:"previous_close"`
	Volume        int     `json:"volume"`
	Vwap          float64 `json:"vwap"`
}

type TickerUnderlying struct {
	ChangeToBreakEven float64 `json:"change_to_break_even"`
	LastUpdated       int64   `json:"last_updated"`
	Price             float64 `json:"price"`
	Ticker            string  `json:"ticker"`
	Timeframe         string  `json:"timeframe"`
}

type TickerResult struct {
	BreakEvenPrice    float64          `json:"break_even_price"`
	Day               TickerDay        `json:"day"`
	Details           TickerData       `json:"details"`
	ImpliedVolatility float64          `json:"implied_volatility"`
	UnderlyingAsset   TickerUnderlying `json:"underlying_asset"`
}

type TickerDetails struct {
	Results TickerResult `json:"results"`
}

type Data struct {
	K    float64 `json:"K"`
	T    float64 `json:"T"`
	Ivol float64 `json:"Ivol"`
	Name string  `json:"Name"`
}

type Hist struct {
	Close string `json:"4. close"`
}

type AlphaData struct {
	Hist map[string]Hist `json:"Time Series (Daily)"`
}

type Overview struct {
	DividentYield string `json:"DividendYield"`
}

type Model map[string]mc.HypHyp

type Corr struct {
	X1   int
	X2   int
	Corr float64
}

type Stat struct {
	SpotPrice map[string]float64 `json:"spot_price"`
	Mean      map[string]float64 `json:"mean"`
	Index     map[string]int     `json:"index"`
	CorrPairs []Corr             `json:"correlation_pairs"`
}
