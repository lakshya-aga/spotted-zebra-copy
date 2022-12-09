package data

import (
	"database/sql"
	"fmt"
	"main/mc"
	"sort"
	"strconv"
	"time"
)

func getTickers(stock string) ([]string, error) {
	// initialize variables
	var initialUrl string // variable to store the 'next' url in the api response
	url := fmt.Sprintf("https://api.polygon.io/v3/reference/options/contracts?underlying_ticker=%v&limit=1000", stock)
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
		return []string{}, nil
	}

	var tickerArr []string
	for j := range ticker.Results {
		tickerArr = append(tickerArr, ticker.Results[j].Ticker)
	}
	return tickerArr, nil
}

func getTickerDetails(stock string) ([]Data, error) {
	// get the tickerMaps
	data, err := getTickers(stock)
	if err != nil {
		return nil, err
	}

	// initialize variables
	var detailsArr []Data
	ch := make(chan Data, len(data))
	errCh := make(chan error, len(data))
	defer close(ch)
	defer close(errCh)

	// get the dividend yield of equities from alphavantage
	co, _ := getAlphavantage(fmt.Sprintf("https://www.alphavantage.co/query?function=OVERVIEW&symbol=%v", stock), Overview{})

	// convert string to float64
	dy, err := strconv.ParseFloat(co.DividentYield, 64)
	if err != nil {
		dy = 0.
	}

	// goroutines
	for j := 0; j < len(data); j++ {
		go func(stock string, contract string, dy float64, ch chan Data, errCh chan error) {
			var option string
			url := fmt.Sprintf("https://api.polygon.io/v3/snapshot/options/%v/%v", stock, contract)
			// get the contract details from polygon
			result, err := getPolygon(url, TickerDetails{})
			if err != nil {
				ch <- Data{}
				errCh <- err
			}
			// initialize the values
			expiry := result.Results.Details.ExpirationDate
			strike := result.Results.Details.StrikePrice
			underlying := result.Results.UnderlyingAsset.Price
			callPut := result.Results.Details.ContractType
			p := result.Results.Day.Close
			k := strike / underlying
			t, err := time.Parse("2006-01-02", expiry)
			if err != nil {
				ch <- Data{}
				errCh <- err
			}
			maturity := float64(t.Unix()-time.Now().Unix()) / float64(60*60*24*365)
			// return empty data if not match requirement
			if (strike >= underlying && callPut == "put") || (strike <= underlying && callPut == "call") || p == 0. || k < 0.5 || k > 2.0 || maturity <= 0. {
				ch <- Data{}
				errCh <- nil
				return
			}
			if callPut == "call" {
				option = "c"
			} else {
				option = "p"
			}
			r := 0.03
			// calculate implied volatility based on black-scholes model
			ivol := fit(p, strike, underlying, maturity, dy, r, option)
			ch <- Data{K: k, T: maturity, Ivol: ivol, Name: contract}
			errCh <- nil
		}(stock, data[j], dy, ch, errCh)
	}

	for i := 0; i < len(data); i++ {
		err := <-errCh
		if err != nil {
			return nil, err
		}
		details := <-ch
		if details.K > 0.5 && details.K < 2.0 {
			detailsArr = append(detailsArr, details)
		}
	}
	sort.Slice(detailsArr, func(i, j int) bool { return detailsArr[i].K <= detailsArr[j].K })
	err = createJson(map[string][]Data{"results": detailsArr}, fmt.Sprintf("storage/%v_ivol.json", stock))
	if err != nil {
		return nil, err
	}

	return detailsArr, nil
}

func getPara(stock string) (mc.Model, error) {
	// get the contract details
	data, err := getTickerDetails(stock)
	if err != nil {
		return nil, err
	}

	// fit the contract details to the model, save the model parameters
	var model mc.Model = mc.NewHypHyp()
	d, err := loadMktData(data)
	if err != nil {
		fmt.Println(err)
	}
	model = mc.Fit(model, d)

	return model, nil
}

func getModel(db *sql.DB, today string) (map[string]mc.Model, error) {
	rows, err := db.Query(`SELECT "Ticker", "Sigma", "Alpha", "Beta", "Kappa", "Rho" FROM "ModelParameters" WHERE "Date" IN ($1)`, today)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	modelsMap := map[string]mc.Model{}
	for rows.Next() {
		var ticker string
		var sigma, alpha, beta, kappa, rho float64
		err = rows.Scan(&ticker, &sigma, &alpha, &beta, &kappa, &rho)
		if err != nil {
			return nil, err
		}
		modelsMap[ticker] = mc.HypHyp{Sigma: sigma, Alpha: alpha, Beta: beta, Kappa: kappa, Rho: rho}
	}
	return modelsMap, nil
}

func newCalibrate(stocks []string, db *sql.DB, date string) (map[string]mc.Model, error) {
	modelsMap := map[string]mc.Model{}
	for i := range stocks {
		model, _ := getPara(stocks[i])
		modelsMap[stocks[i]] = model
	}
	insertPar := `insert into "ModelParameters"("Date", "Ticker", "Sigma", "Alpha", "Beta", "Kappa", "Rho") values($1, $2, $3, $4, $5, $6, $7)`
	for i := 0; i < len(stocks); i++ {
		pars := modelsMap[stocks[i]].Pars()
		_, err := db.Exec(insertPar, date, stocks[i], pars[0], pars[1], pars[2], pars[3], pars[4])
		if err != nil {
			panic(err)
		}
	}
	return modelsMap, nil
}

func Calibrate(stocks []string, db *sql.DB) (map[string]mc.Model, error) {
	today := time.Now().Format(Layout)
	update, err := UpdateRequired("ModelParameters", db, today)
	if err != nil {
		panic(err)
	}

	if !update {
		modelsMap, err := getModel(db, today)
		if err != nil {
			panic(err)
		}
		return modelsMap, nil
	}

	modelsMap, err := newCalibrate(stocks, db, today)
	if err != nil {
		return nil, err
	}
	return modelsMap, nil
}

func ModelSample(stocks []string, models map[string]mc.Model) map[string]mc.Model {
	result := make(map[string]mc.Model)
	for i := 0; i < len(stocks); i++ {
		result[stocks[i]] = models[stocks[i]]
	}
	return result
}
