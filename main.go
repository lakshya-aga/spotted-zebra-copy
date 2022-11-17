package main

import (
	"encoding/json"
	"fmt"
	"main/data"
	"main/mc"
	"main/payoff"
	"main/utils"
	"os"
	"sort"
	"strings"
	"time"

	"golang.org/x/exp/rand"
	"gonum.org/v1/gonum/stat/distmv"
	"gonum.org/v1/gonum/stat/distuv"
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

func main() {
	strike := 0.80
	cpn := 0.10
	barrierCpn := 0.20
	fixCpn := 0.20
	KO := 1.05
	KI := 0.70
	KC := 0.80
	maturity := 3
	freq := 1
	isEuro := false
	stocks := []string{"AAPL", "TS-LA", "META", "MSFT", "TS-LA"}

	stocks = format(stocks)

	tNow, _ := time.Parse(Layout, time.Now().Format(Layout))
	dates, err := utils.GenerateDates(tNow, maturity, freq)
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}

	modelsMap, err := data.Calibrate(stocks)
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}

	var stk []string
	for i := range modelsMap {
		stk = append(stk, i)
	}
	sort.Strings(stk)

	bsk, err := mc.NewBasket(modelsMap)
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}

	mu, corrMatrix, spotref, err := data.Statistics(stk)
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}

	dz1, ok := distmv.NewNormal(mu, corrMatrix, rand.NewSource(uint64(time.Now().UnixNano())))
	if !ok {
		fmt.Println("invalid corr matrix")
		os.Exit(-1)
	}

	dz2 := distuv.Normal{Mu: 0.0, Sigma: 1.0, Src: rand.NewSource(uint64(time.Now().UnixNano()))}

	fcn, err := payoff.NewFCN(stk, strike, cpn, barrierCpn, fixCpn, KO, KI, KC, maturity, freq, isEuro, dates)
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}

	index := map[string]int{}
	for i := range stk {
		index[stk[i]] = i
	}

	var corr []float64
	nrow, ncol := corrMatrix.Caps()
	for r := 0; r < nrow; r++ {
		for c := 0; c < ncol; c++ {
			corr = append(corr, corrMatrix.At(r, c))
		}
	}

	result := mc.BasketList{Stocks: stk, Parameter: modelsMap, Index: index, CorrMatrix: corr, Mean: mu, SpotPrice: spotref}
	ele, err := json.MarshalIndent(result, "", " ")
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
	err = os.WriteFile("basket.json", ele, 0644)
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}

	var pxRatio []float64
	currentPx := data.SpotPx(stk)
	for _, v := range stk {
		pxRatio = append(pxRatio, currentPx[v]/spotref[v])
	}

	nsamples := 10000
	out := 0.0
	ch := make(chan float64, nsamples)
	defer close(ch)

	// Compute path payouts concurrently
	for l := 0; l < nsamples; l++ {
		go func() {
			path, err := bsk.Path(dates["mcdates"], pxRatio, dz1, dz2)
			if err != nil {
				fmt.Println(err)
				os.Exit(-1)
			}

			wop := wop(spotref, dates, path)

			x := fcn.Payout(wop)
			ch <- x
		}()
	}

	// bar := progressBar(10)
	for l := 0; l < nsamples; l++ {
		out += <-ch
		// bar.Add(1)
	}

	fmt.Println(out / float64(nsamples))

	// fmt.Println(fcn)

	// bsk, err := bsk("basket.json")
	// if err != nil {
	// 	fmt.Println(err)
	// }
	// // bs, _ := mc.NewBasket(bsk.Models)
	// // fmt.Println(bs)
	// // fmt.Println(bsk.CorrID)
	// // fmt.Println(bsk.CorrMatrix)
	// corrMatrix := mat.NewSymDense(len(bsk.Stocks), bsk.CorrMatrix)

	// // _, _, _ = bsk, mu, corrMatrix

	// t_now, _ := time.Parse(Layout, time.Now().Format(Layout))
	// dates, err := utils.GenerateDates(t_now, 6, 1)
	// if err != nil {
	// 	fmt.Println(err)
	// 	os.Exit(-1)
	// }

	// dz1, ok := distmv.NewNormal(bsk.Mean, corrMatrix, rand.NewSource(uint64(time.Now().UnixNano())))
	// if !ok {
	// 	fmt.Println("invalid corr matrix")
	// 	os.Exit(-1)
	// }

	// dz2 := distuv.Normal{Mu: 0.0, Sigma: 1.0, Src: rand.NewSource(uint64(time.Now().UnixNano()))}

	// var b mc.Basket
	// for i := 0; i < len(bsk.Stocks); i++ {
	// 	b = append(b, mc.Stock{Ticker: bsk.Stocks[i], Model: bsk.Models[bsk.Stocks[i]]})
	// }

	// strike := 0.80
	// cpn := 0.10
	// barrierCpn := 0.00
	// fixCpn := 0.00
	// KO := 1.05
	// KI := 0.70
	// maturity := 3
	// freq := 1
	// isEuro := false

	// fcn, err := payoff.NewFCN(bsk.Stocks, strike, cpn, barrierCpn, fixCpn, KO, KI, maturity, freq, isEuro, dates)
	// if err != nil {
	// 	fmt.Println(err)
	// 	os.Exit(-1)
	// }

	// nsamples := 10000
	// out := 0.0
	// ch := make(chan float64, nsamples)
	// defer close(ch)

	// // Compute path payouts concurrently
	// for l := 0; l < nsamples; l++ {
	// 	go func() {
	// 		path, err := b.Path(dates["mcdates"], []float64{1.0, 1.0, 1.0}, dz1, dz2)
	// 		if err != nil {
	// 			fmt.Println(err)
	// 			os.Exit(-1)
	// 		}

	// 		wop := wop(bsk.SpotPrice, dates, path)

	// 		x := fcn.Payout(wop)
	// 		ch <- x
	// 	}()
	// }

	// // bar := progressBar(10)
	// for l := 0; l < nsamples; l++ {
	// 	out += <-ch
	// 	// bar.Add(1)
	// }

	// fmt.Println(out / float64(nsamples))
}

func format(stocks []string) []string {
	sort.Strings(stocks)
	for s := 0; s < len(stocks); s++ {
		stocks[s] = strings.ToUpper(stocks[s])
	}
	var unique []string

	for _, v := range stocks {
		skip := false
		for _, u := range unique {
			if v == u {
				skip = true
				break
			}
		}
		if !skip {
			unique = append(unique, v)
		}
	}
	return unique
}

// helper function to open tickers.json
func bsk(filename string) (bskData, error) {
	details := bskData{}
	file, err := os.ReadFile(filename)
	if err != nil {
		return bskData{}, err
	}
	err = json.Unmarshal([]byte(file), &details)
	if err != nil {
		return bskData{}, err
	}
	return details, nil
}

func wop(spotPrice map[string]float64, dates map[string][]time.Time, path mc.MCPath) []float64 {
	// get the worst of performance array
	// fmt.Println(path)
	var wop []float64
	p := make([]float64, len(spotPrice))
	for i := range dates["mcdates"] {
		j := 0
		for k := range spotPrice {
			p[j] = path[k][i]
			j++
		}
		minP := utils.MinSlice(p)
		wop = append(wop, minP)
	}
	return wop
}
