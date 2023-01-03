package mainfuncs

import (
	"database/sql"
	"fmt"
	"github.com/banachtech/spotted-zebra/data"

	"gonum.org/v1/gonum/stat"
)

func BackTest(defaultStocks, stocks []string, index map[string]int, k, cpn, barCpn, fixCpn, ko, ki, kc float64, T, freq int, isEuro bool, db *sql.DB) {
	dts, err := db.Query(`select distinct "Date" from "HistoricalData" order by "Date" desc`)
	if err != nil {
		panic(err)
	}
	var dates []string
	var profit []float64
	for dts.Next() {
		var date string
		err = dts.Scan(&date)
		if err != nil {
			panic(err)
		}
		dates = append(dates, date)
	}

	for t := range dates {
		modelsMap, err := data.Calibrate(db, dates[t])
		if err != nil {
			panic(err)
		}
		sampleModels := data.ModelSample(stocks, modelsMap)

		mean, corr, fixings, err := data.Statistics(defaultStocks, db, dates[t])
		if err != nil {
			panic(err)
		}
		sampleMu, sampleCorr, sampleRef := data.StatsSample(stocks, index, corr, mean, fixings)

		p, err := Pricer(stocks, k, cpn, barCpn, fixCpn, ko, ki, kc, T, freq, isEuro, sampleRef, sampleModels, sampleMu, sampleCorr, db)
		if err != nil {
			panic(err)
		}

		payout, err := Payout(dates[t], stocks, k, cpn, barCpn, fixCpn, ko, ki, kc, T, freq, isEuro, sampleRef, sampleModels, sampleMu, sampleCorr, db)
		if err != nil {
			panic(err)
		}

		pnl := payout - p
		profit = append(profit, pnl)
		fmt.Println(p, payout, payout-p)
	}
	fmt.Println(profit)
	mean, std := stat.MeanStdDev(profit, nil)
	min, max := minmax(profit)
	fmt.Println(mean, std, min, max)
}

func minmax(array []float64) (float64, float64) {
	max := array[0]
	min := array[0]
	for _, value := range array {
		if max < value {
			max = value
		}
		if min > value {
			min = value
		}
	}
	return min, max
}
