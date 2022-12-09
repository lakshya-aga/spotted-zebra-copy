package main

import (
	"fmt"
	"main/data"
	"main/db"
	"main/handler"
	"main/mc"
	"os"
	"sort"
)

type bskData struct {
	Stocks     []string             `json:"stocks"`
	Models     map[string]mc.HypHyp `json:"model_parameters"`
	CorrID     map[string]int       `json:"corr_id"`
	Mean       []float64            `json:"mean"`
	CorrMatrix []float64            `json:"corr_matrix"`
	SpotPrice  map[string]float64   `json:"spot_price"`
}

const Layout = "2006-01-02"

var DefaultStocks = []string{"AAPL", "AMZN", "META", "MSFT", "TSLA", "GOOG", "NVDA", "AVGO", "QCOM", "INTC"}

func main() {
	// connect to database
	db, err := db.ConnectDB()
	if err != nil {
		panic(err)
	}
	defer db.Close()

	sort.Strings(DefaultStocks)
	allModels, allMeans, allCorr, allFixings, err := data.Initialize(DefaultStocks, db)
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}

	selectStocks := []string{"AAPL", "META", "MSFT", "ABNB"}

	filterStocks, sampleModels, sampleMu, sampleCorr, sampleFixings, err := data.Sampler(DefaultStocks, selectStocks, allModels, allMeans, allCorr, allFixings)
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}

	strike := 0.80
	cpn := 0.10
	barrierCpn := 0.20
	fixCpn := 0.20
	KO := 1.05
	KI := 0.70
	KC := 0.80
	maturity := 3
	freq := 1
	isEuro := true

	p, err := handler.Pricer(filterStocks, strike, cpn, barrierCpn, fixCpn, KO, KI, KC, maturity, freq, isEuro, sampleFixings, sampleModels, sampleMu, sampleCorr)
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
	fmt.Println(p)

}
