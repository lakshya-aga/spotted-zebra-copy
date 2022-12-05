package data

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"
)

func GetPastContractsDetails(db *sql.DB) {
	var option string
	var data []IvolData
	symbol := "AAPL"
	px, err := GetHistPx(symbol)
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}

	// dates := getDates(px)

	tickers, err := GetPastContracts(symbol)
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}

	tNow, _ := time.Parse(Layout, time.Now().Format(Layout))
	tStart := tNow.AddDate(-2, 0, 0)
	tEnd := tNow.AddDate(0, 0, -1)

	bar := progressBar(len(tickers))
	for k := range tickers {
		bar.Describe(fmt.Sprintf("Processing %v\t", tickers[k].Ticker))
		bar.Add(1)
		url := fmt.Sprintf("https://api.polygon.io/v2/aggs/ticker/%v/range/1/day/%v/%v?sort=asc&limit=5000", tickers[k].Ticker, tStart.Format(Layout), tEnd.Format(Layout))
		contractPx, err := getPolygon(url, TickerAggs{})
		if err != nil {
			fmt.Println(err)
			os.Exit(-1)
		}
		if contractPx.Status != "OK" {
			continue
		}
		if len(contractPx.Results) == 0 {
			continue
		}
		for i := 0; i < len(contractPx.Results); i++ {
			t1, _ := time.Parse(Layout, tickers[k].ExpirationDate)
			t0, _ := time.Parse(Layout, time.UnixMilli(contractPx.Results[i].T).UTC().Format(Layout))
			maturity := float64(t1.Unix()-t0.Unix()) / float64(60*60*24*365)
			underlying, _ := strconv.ParseFloat(px[t0.Format(Layout)].Close, 64)
			s := tickers[k].StrikePrice / underlying
			if (tickers[k].StrikePrice >= underlying && tickers[k].ContractType == "put") || (tickers[k].StrikePrice <= underlying && tickers[k].ContractType == "call") || maturity <= 0.0 || s < 0.5 || s > 2.0 {
				continue
			}
			if tickers[k].ContractType == "call" {
				option = "c"
			} else {
				option = "p"
			}
			r := 0.03
			// calculate implied volatility based on black-scholes model
			ivol := fit(contractPx.Results[i].C, tickers[k].StrikePrice, underlying, maturity, 0.0, r, option)
			data = append(data, IvolData{Date: t0.Format(Layout), Name: tickers[k].Ticker, K: s, T: maturity, Ivol: ivol})
			// fmt.Println(tickers[k].Ticker, contractPx.Close, k, maturity, ivol)
			// dataMap[t0.Format(Layout)] = append(dataMap[t0.Format(Layout)], Data{K: k, T: maturity, Ivol: ivol, Name: tickers[k].Ticker})
			// insertHist := `insert into "HistoricalData"("Date", "Ticker", "K", "T", "Ivol") values($1, $2, $3, $4, $5)`
			// _, err = db.Exec(insertHist, t0.Format(Layout), tickers[k].Ticker, s, maturity, ivol)
			// if err != nil {
			// 	panic(err)
			// }
		}
	}

	fmt.Println("here")
	sqlStr := `insert into HistoricalData ("Date", "Ticker", "K", "T", "Ivol") values `
	vals := []interface{}{}
	for i, row := range data {
		n := i * 5
		sqlStr += `(`
		for j := 0; j < 5; j++ {
			sqlStr += `$` + strconv.Itoa(n+j+1) + `,`
		}
		sqlStr = sqlStr[:len(sqlStr)-1] + `),`
		vals = append(vals, row.Date, row.Name, row.K, row.T, row.Ivol)
	}
	sqlStr = sqlStr[:len(sqlStr)-1]
	_, err = db.Exec(sqlStr, vals...)
	if err != nil {
		panic(err)
	}
	fmt.Println("done!")
	// dataMap := map[string][]Data{}
	// for t := range dates {
	// 	bar := progressBar(len(tickers))
	// 	for i := range tickers {
	// 		bar.Describe(fmt.Sprintf("Processing %v\t", tickers[i].Ticker))
	// 		var option string
	// 		url := fmt.Sprintf("https://api.polygon.io/v1/open-close/%v/%v", tickers[i].Ticker, dates[t])
	// 		contractPx, err := getPolygon(url, TickerLastTrade{})
	// 		if contractPx.Status != "OK" {
	// 			bar.Add(1)
	// 			continue
	// 		}
	// 		if err != nil {
	// 			bar.Add(1)
	// 			continue
	// 		}
	// 		t1, _ := time.Parse(Layout, tickers[i].ExpirationDate)
	// 		t0, _ := time.Parse(Layout, dates[t])
	// 		maturity := float64(t1.Unix()-t0.Unix()) / float64(60*60*24*365)
	// 		underlying, _ := strconv.ParseFloat(px[dates[t]].Close, 64)
	// 		k := tickers[i].StrikePrice / underlying
	// 		if (tickers[i].StrikePrice >= underlying && tickers[i].ContractType == "put") || (tickers[i].StrikePrice <= underlying && tickers[i].ContractType == "call") || maturity <= 0.0 || k < 0.5 || k > 2.0 {
	// 			bar.Add(1)
	// 			continue
	// 		}
	// 		if tickers[i].ContractType == "call" {
	// 			option = "c"
	// 		} else {
	// 			option = "p"
	// 		}
	// 		r := 0.03
	// 		// calculate implied volatility based on black-scholes model
	// 		ivol := fit(contractPx.Close, tickers[i].StrikePrice, underlying, maturity, 0.0, r, option)
	// 		// fmt.Println(tickers[i].Ticker, contractPx.Close, k, maturity, ivol)
	// 		dataMap[dates[t]] = append(dataMap[dates[t]], Data{K: k, T: maturity, Ivol: ivol, Name: tickers[i].Ticker})
	// 		insertHist := `insert into "HistoricalData"("Date", "Ticker", "K", "T", "Ivol") values($1, $2, $3, $4, $5)`
	// 		_, err = db.Exec(insertHist, dates[t], tickers[i].Ticker, k, maturity, ivol)
	// 		if err != nil {
	// 			panic(err)
	// 		}
	// 		bar.Add(1)
	// 	}
	// }
}

func GetHistPx(stock string) (map[string]Hist, error) {
	px, err := getAlphavantage(fmt.Sprintf("https://www.alphavantage.co/query?function=TIME_SERIES_DAILY_ADJUSTED&outputsize=full&symbol=%v", stock), AlphaData{})
	if err != nil {
		err = errors.New("in GetHistPx(), http error: AlphaData{}")
		return nil, err
	}
	return px.Hist, nil
}

func getDates(px map[string]Hist) []string {
	tNow, _ := time.Parse(Layout, time.Now().Format(Layout))
	tStart := tNow.AddDate(-2, 0, 0)
	tEnd := tNow.AddDate(0, 0, -1)
	t := tStart
	var dates []string
	for {
		_, ok := px[t.Format(Layout)]
		if ok {
			dates = append(dates, t.Format(Layout))
		}
		if tEnd.Sub(t) == 0 {
			break
		}
		t = t.AddDate(0, 0, 1)
	}
	return dates
}

func GetPastContracts(stock string) ([]TickersPara, error) {
	// initialize the variables
	tNow, _ := time.Parse(Layout, time.Now().Format(Layout))
	tStart := tNow.AddDate(-2, 0, 0)
	var initialUrl string // variable to store the 'next' url in the api response
	url := fmt.Sprintf("https://api.polygon.io/v3/reference/options/contracts?underlying_ticker=%v&expiration_date.gte=%v&expired=true&limit=1000", stock, tStart.Format(Layout))
	ticker, err := getPolygon(url, Tickers{})
	if err != nil {
		return nil, err
	}
	// if the contracts are more than 1000, need to access 'next' url
	initialUrl = ticker.Next
	// infinite loop until all contracts have been saved
	for {
		var extra Tickers
		if initialUrl != "" {
			extra, err = getPolygon(initialUrl, Tickers{})
			if err != nil {
				return nil, err
			}
			// append the contracts back to ticker.Results
			ticker.Results = append(ticker.Results, extra.Results...)
		}
		initialUrl = extra.Next
		if initialUrl == "" {
			break
		}
	}

	if len(ticker.Results) == 0 {
		return nil, nil
	}

	// for i := 0; i < len(ticker.Results); i++ {
	// 	tickers = append(tickers, ticker.Results[i].Ticker)
	// }

	return ticker.Results, nil
}
