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

	modelsMap, err := data.Calibrate(stocks)
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}

	bsk, err := mc.NewBasket(modelsMap)
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}

	mu, corrMatrix, spotref, err := data.CorrMatrix(stocks)
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

	path, err := bsk.Path(dates["mcdates"], mu, corrMatrix, []float64{1.0, 1.0, 1.0})
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}

	// get the worst of performance array
	var wop []float64
	p := make([]float64, len(spotref))
	for i := range dates["mcdates"] {
		j := 0
		for k := range spotref {
			p[j] = path[k][i]
			j++
		}
		minP := utils.MinSlice(p)
		wop = append(wop, minP)
	}

	fmt.Println(wop)
	strike := 0.80
	cpn := 0.10
	barrierCpn := 0.10
	fixCpn := 0.10
	KO := 1.05
	KI := 0.40
	maturity := 6
	freq := 1
	isEuro := false

	fcn, err := payoff.NewFCN(stocks, strike, cpn, barrierCpn, fixCpn, KO, KI, maturity, freq, isEuro, dates)
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}

	pay := fcn.Payout(wop)
	fmt.Println(pay)
}

func format(stocks []string) []string {
	sort.Strings(stocks)
	for s := 0; s < len(stocks); s++ {
		stocks[s] = strings.ToUpper(stocks[s])
	}
	return stocks
}
