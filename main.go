package main

import (
	"fmt"
	"main/data"
	"main/mc"
	"main/payoff"
	"main/utils"
	"os"
	"sort"
	"strings"
	"time"
)

const Layout = "2006-01-02"

func main() {
	stocks := []string{"AAPL", "TSLA", "META"}
	stocks = format(stocks)
	spotref := make(map[string]float64)

	modelsMap, err := data.Calibrate(stocks)
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}

	for k := range modelsMap {
		spotref[k] = 1.0
	}

	bsk, err := mc.NewBasket(modelsMap)
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}

	mu, corrMatrix, err := data.CorrMatrix(stocks)
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}

	// var stk []string
	// for i := range modelsMap {
	// 	stk = append(stk, i)
	// }
	// sort.Strings(stk)

	// index := map[string]int{}
	// for i := range stk {
	// 	index[stk[i]] = i
	// }

	// var corr []float64
	// nrow, ncol := corrMatrix.Caps()
	// for r := 0; r < nrow; r++ {
	// 	for c := 0; c < ncol; c++ {
	// 		corr = append(corr, corrMatrix.At(r, c))
	// 	}
	// }

	// result := mc.BasketList{Stocks: stk, Parameter: modelsMap, Index: index, CorrMatrix: corr, Mean: mu}
	// data, err := json.MarshalIndent(result, "", " ")
	// if err != nil {
	// 	fmt.Println(err)
	// 	os.Exit(-1)
	// }
	// err = os.WriteFile("basket.json", data, 0644)
	// if err != nil {
	// 	fmt.Println(err)
	// 	os.Exit(-1)
	// }

	_, _, _ = bsk, mu, corrMatrix

	t_now, _ := time.Parse(Layout, time.Now().Format(Layout))
	dates, err := utils.GenerateDates(t_now, 6, 1)
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}

	path, err := bsk.Path(dates["mcdates"], mu, corrMatrix)
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}

	strike := 0.80
	cpn := 0.10
	barrierCpn := 0.00
	fixCpn := 0.00
	KO := 1.05
	KI := 0.70
	maturity := 6
	freq := 1
	isEuro := true

	fcn, err := payoff.NewFCN(stocks, strike, cpn, barrierCpn, fixCpn, KO, KI, maturity, freq, isEuro, dates)
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}

	pay := fcn.Payout(path, spotref)
	fmt.Println(pay)
}

func format(stocks []string) []string {
	sort.Strings(stocks)
	for s := 0; s < len(stocks); s++ {
		stocks[s] = strings.ToUpper(stocks[s])
	}
	return stocks
}
