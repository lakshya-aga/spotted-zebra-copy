package main

import (
	"fmt"
	"main/data"
	"main/db"
	"main/mc"
	"main/utils"
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
	// sort.Strings(DefaultStocks)

	// connect to database
	db, err := db.ConnectDB()
	if err != nil {
		panic(err)
	}
	defer db.Close()

	for i := 0; i < len(DefaultStocks); i++ {
		fmt.Printf("Handing %s\n", DefaultStocks[i])
		data.GetPastContractsDetails(DefaultStocks[i], db)
		data.FitPastParameters(DefaultStocks[i], db)
	}

	// selectStocks := []string{"AAPL", "META", "MSFT", "ABNB"}

	// filterStocks, stocksMap, err := utils.Filter(selectStocks, DefaultStocks)
	// if err != nil {
	// 	fmt.Println(err)
	// 	os.Exit(-1)
	// }

	// allModelsMap, err := data.Calibrate(DefaultStocks, db)
	// if err != nil {
	// 	fmt.Println(err)
	// 	os.Exit(-1)
	// }

	// sampleModels := data.ModelSample(filterStocks, allModelsMap)
	// fmt.Println(sampleModels)

	// allmu, allcorrMatrix, allref, err := data.Statistics(DefaultStocks, db)
	// if err != nil {
	// 	fmt.Println(err)
	// 	os.Exit(-1)
	// }
	// _, _, _, _ = allmu, allcorrMatrix, allref, allModelsMap

	// sampleMu, sampleCorr, sampleRef := data.StatsSample(filterStocks, stocksMap, allcorrMatrix, allmu, allref)
	// fmt.Println(sampleCorr)

	// strike := 0.80
	// cpn := 0.10
	// barrierCpn := 0.20
	// fixCpn := 0.20
	// KO := 1.05
	// KI := 0.70
	// KC := 0.80
	// maturity := 3
	// freq := 1
	// isEuro := true

	// stocks := utils.Format(filterStocks)

	// tNow, _ := time.Parse(Layout, time.Now().Format(Layout))
	// dates, err := utils.GenerateDates(tNow, maturity, freq)
	// if err != nil {
	// 	fmt.Println(err)
	// 	os.Exit(-1)
	// }

	// bsk, err := mc.NewBasket(sampleModels)
	// if err != nil {
	// 	fmt.Println(err)
	// 	os.Exit(-1)
	// }

	// dz1, ok := distmv.NewNormal(sampleMu, sampleCorr, rand.NewSource(uint64(time.Now().UnixNano())))
	// if !ok {
	// 	fmt.Println("invalid corr matrix")
	// 	os.Exit(-1)
	// }

	// dz2 := distuv.Normal{Mu: 0.0, Sigma: 1.0, Src: rand.NewSource(uint64(time.Now().UnixNano()))}

	// fcn, err := payoff.NewFCN(stocks, strike, cpn, barrierCpn, fixCpn, KO, KI, KC, maturity, freq, isEuro, dates)
	// if err != nil {
	// 	fmt.Println(err)
	// 	os.Exit(-1)
	// }

	// var pxRatio []float64
	// currentPx := data.LatestPx(stocks)
	// for _, v := range stocks {
	// 	pxRatio = append(pxRatio, currentPx[v]/sampleRef[v])
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

	// 		wop := wop(sampleRef, dates, path)

	// 		x := fcn.Payout(wop)
	// 		ch <- x
	// 	}()
	// }

	// for l := 0; l < nsamples; l++ {
	// 	out += <-ch
	// }

	// price := out / float64(nsamples)
	// fmt.Println(price)

	// fcn.Save("contract.json", price, spotref)

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
