package data

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"main/mc"
	"os"
	"strconv"
	"time"
)

func FitPastParameters(symbol string, db *sql.DB) {
	rows, err := db.Query(`SELECT DISTINCT "Date", "Ticker", "K", "T", "Ivol" FROM "HistoricalData" WHERE "Underlying" = $1 ORDER BY "Date"`, symbol)
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
	defer rows.Close()
	data := map[string][]Data{}
	for rows.Next() {
		var date, name string
		var k, t, ivol float64
		err = rows.Scan(&date, &name, &k, &t, &ivol)
		if err != nil {
			fmt.Println(err)
			os.Exit(-1)
		}
		data[date] = append(data[date], Data{Name: name, K: k, T: t, Ivol: ivol})
	}

	ch := make(chan mc.Model, len(data))
	datesCh := make(chan string, len(data))
	errCh := make(chan error, len(data))
	defer close(ch)
	defer close(datesCh)
	defer close(errCh)

	for dt, d := range data {
		go func(date string, data []Data, ch chan mc.Model, datesCh chan string, errCh chan error) {
			var model mc.Model = mc.NewHypHyp()
			d, err := loadMktData(data)
			if err != nil {
				ch <- nil
				datesCh <- date
				errCh <- err
			}
			model = mc.Fit(model, d)
			ch <- model
			datesCh <- date
			errCh <- nil
		}(dt, d, ch, datesCh, errCh)
	}

	sqlStr := `insert into "BacktestParameters" ("Date", "Ticker", "Sigma", "Alpha", "Beta", "Kappa", "Rho") values %s`
	valchunk := map[int][]interface{}{}
	datachunk := map[int][][]interface{}{}
	c := 0
	for i := 0; i < len(data); i++ {
		dt := <-datesCh
		model := <-ch
		pars := model.Pars()
		valchunk[c] = append(valchunk[c], dt, symbol, pars[0], pars[1], pars[2], pars[3], pars[4])
		datachunk[c] = append(datachunk[c], []interface{}{dt, symbol, pars[0], pars[1], pars[2], pars[3], pars[4]})
		n := (i + 1) % 3000
		if n == 0 {
			c++
		}
	}
	for k, v := range datachunk {
		subStr := prepareQueryCreateBulk(sqlStr, v, 7)
		_, err = db.Exec(subStr, valchunk[k]...)
		if err != nil {
			panic(err)
		}
	}
	fmt.Println("saved all parameters!")
}

func GetPastContractsDetails(symbol string, db *sql.DB) {
	var option string
	var data []IvolData
	px, err := GetHistPx(symbol)
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}

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
		if contractPx.Count == 0 || len(contractPx.Results) == 0 {
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
			data = append(data, IvolData{Date: t0.Format(Layout), Name: tickers[k].Ticker, K: s, T: maturity, Ivol: ivol, Underlying: symbol})
		}
	}
	if len(data) != 0 {
		sqlStr := `insert into "HistoricalData" ("Date", "Ticker", "K", "T", "Ivol", "Underlying") values %s`
		valchunk := map[int][]interface{}{}
		datachunk := map[int][][]interface{}{}
		c := 0
		for i, v := range data {
			valchunk[c] = append(valchunk[c], v.Date, v.Name, v.K, v.T, v.Ivol, v.Underlying)
			datachunk[c] = append(datachunk[c], []interface{}{v.Date, v.Name, v.K, v.T, v.Ivol, v.Underlying})
			n := (i + 1) % 3000
			if n == 0 {
				c++
			}
		}
		for k, v := range datachunk {
			subStr := prepareQueryCreateBulk(sqlStr, v, 6)
			_, err = db.Exec(subStr, valchunk[k]...)
			if err != nil {
				panic(err)
			}
		}
	}
	fmt.Println("saved all data!")
}

func GetHistPx(stock string) (map[string]Hist, error) {
	px, err := getAlphavantage(fmt.Sprintf("https://www.alphavantage.co/query?function=TIME_SERIES_DAILY_ADJUSTED&outputsize=full&symbol=%v", stock), AlphaData{})
	if err != nil {
		err = errors.New("in GetHistPx(), http error: AlphaData{}")
		return nil, err
	}
	return px.Hist, nil
}

// func getDates(px map[string]Hist) []string {
// 	tNow, _ := time.Parse(Layout, time.Now().Format(Layout))
// 	tStart := tNow.AddDate(-2, 0, 0)
// 	tEnd := tNow.AddDate(0, 0, -1)
// 	t := tStart
// 	var dates []string
// 	for {
// 		_, ok := px[t.Format(Layout)]
// 		if ok {
// 			dates = append(dates, t.Format(Layout))
// 		}
// 		if tEnd.Sub(t) == 0 {
// 			break
// 		}
// 		t = t.AddDate(0, 0, 1)
// 	}
// 	return dates
// }

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

	return ticker.Results, nil
}

func prepareQueryCreateBulk(s string, models [][]interface{}, n int) string {
	bf := bytes.Buffer{}
	for i := range models {
		numFields := n // the number of fields you are inserting
		n := i * numFields

		bf.WriteString("(")
		for j := 0; j < numFields; j++ {
			bf.WriteString("$")
			bf.WriteString(strconv.Itoa(n + j + 1))
			bf.WriteString(", ")
		}
		bf.Truncate(bf.Len() - 2)
		bf.WriteString("), ")
	}
	bf.Truncate(bf.Len() - 2)

	return fmt.Sprintf(s, bf.String())
}
