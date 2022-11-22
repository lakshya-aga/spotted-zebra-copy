package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"main/data"
	"main/mc"
	"main/utils"
	"os"
	"sort"
	"time"
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
	sort.Strings(DefaultStocks)

	selectStocks := []string{"AAPL", "META", "MSFT", "ABNB"}

	filterStocks, stocksMap, err := filter(selectStocks, DefaultStocks)
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}

	allModelsMap, err := data.Calibrate(DefaultStocks)
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}

	sampleModels := data.ModelSample(filterStocks, allModelsMap)
	fmt.Println(sampleModels)

	allmu, allcorrMatrix, allspotref, err := data.Statistics(DefaultStocks)
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
	_, _, _, _ = allmu, allcorrMatrix, allspotref, allModelsMap

	corr := data.CorrSample(filterStocks, stocksMap, allcorrMatrix)
	fmt.Println(corr)

	// strike := 0.80
	// cpn := 0.10
	// barrierCpn := 0.20
	// fixCpn := 0.20
	// KO := 1.05
	// KI := 0.70
	// KC := 0.80
	// maturity := 3
	// freq := 1
	// isEuro := false
	// stocks := []string{"AAPL", "AMZN", "META", "MSFT", "TSLA"}

	// stocks = utils.Format(stocks)

	// tNow, _ := time.Parse(Layout, time.Now().Format(Layout))
	// dates, err := utils.GenerateDates(tNow, maturity, freq)
	// if err != nil {
	// 	fmt.Println(err)
	// 	os.Exit(-1)
	// }

	// stk, modelsMap := data.ModelSample(stocks, allModelsMap)

	// bsk, err := mc.NewBasket(modelsMap)
	// if err != nil {
	// 	fmt.Println(err)
	// 	os.Exit(-1)
	// }

	// mu, corrMatrix, spotref, err := data.Statistics(stk)
	// if err != nil {
	// 	fmt.Println(err)
	// 	os.Exit(-1)
	// }

	// dz1, ok := distmv.NewNormal(mu, corrMatrix, rand.NewSource(uint64(time.Now().UnixNano())))
	// if !ok {
	// 	fmt.Println("invalid corr matrix")
	// 	os.Exit(-1)
	// }

	// dz2 := distuv.Normal{Mu: 0.0, Sigma: 1.0, Src: rand.NewSource(uint64(time.Now().UnixNano()))}

	// fcn, err := payoff.NewFCN(stk, strike, cpn, barrierCpn, fixCpn, KO, KI, KC, maturity, freq, isEuro, dates)
	// if err != nil {
	// 	fmt.Println(err)
	// 	os.Exit(-1)
	// }

	// var pxRatio []float64
	// currentPx := data.SpotPx(stk)
	// for _, v := range stk {
	// 	pxRatio = append(pxRatio, currentPx[v]/spotref[v])
	// }

	// nsamples := 10000
	// out := 0.0
	// ch := make(chan float64, nsamples)
	// defer close(ch)

	// // Compute path payouts concurrently
	// for l := 0; l < nsamples; l++ {
	// 	go func() {
	// 		path, err := bsk.Path(dates["mcdates"], pxRatio, dz1, dz2)
	// 		if err != nil {
	// 			fmt.Println(err)
	// 			os.Exit(-1)
	// 		}

	// 		wop := wop(spotref, dates, path)

	// 		x := fcn.Payout(wop)
	// 		ch <- x
	// 	}()
	// }

	// // bar := progressBar(10)
	// for l := 0; l < nsamples; l++ {
	// 	out += <-ch
	// 	// bar.Add(1)
	// }

	// price := out / float64(nsamples)
	// fmt.Println(price)

	// fcn.Save("contract.json", price, spotref)

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

func filter(selectStocks, DefaultStocks []string) ([]string, map[string]int, error) {
	stockIndex := map[string]int{}
	var stocks []string
	for i, v := range DefaultStocks {
		for j := range selectStocks {
			if v == selectStocks[j] {
				stockIndex[v] = i
				stocks = append(stocks, v)
			}
		}
	}
	if len(stocks) == 0 {
		err := errors.New("there is no available stocks")
		return nil, nil, err
	}
	return stocks, stockIndex, nil
}
