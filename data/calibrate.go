package data

import (
	"fmt"
	"main/mc"
	"os"
	"sort"
	"strconv"
	"time"
)

const Layout = "2006-01-02"

/*

get the tickers that have options trading in market

args
stocks ([]string): slice of tickers

return
contractMap (map[string][]string): map of stocks' option contracts
error (error): error message

*/
func GetTickers(stocks []string) (map[string][]string, error) {
	// initialize the variables
	tickersMap := map[string][]string{}
	var tickerArr []Tickers
	var stockArr []string
	ch := make(chan Tickers, len(stocks)) // channel to store contracts
	stockCh := make(chan string, len(stocks)) // channel to store tickers
	defer close(ch)
	defer close(stockCh)

	// timing
	start := time.Now()

	// construct goroutines
	for i := 0; i < len(stocks); i++ {
		go func(symbol string, ch chan Tickers, stockCh chan string) {
			// initialize variables
			var initialUrl string // variable to store the 'next' url in the api response
			url := fmt.Sprintf("https://api.polygon.io/v3/reference/options/contracts?underlying_ticker=%v&limit=1000", symbol)
			ticker, err := getPolygon(url, Tickers{})
			if err != nil {
				fmt.Println(err)
				os.Exit(-1)
			}
			// if the contracts are more than 1000, need to access 'next' url
			initialUrl = ticker.Next
			// infinite loop until all contracts have been saved
			for {
				var extra Tickers
				if initialUrl != "" {
					extra, err = getPolygon(initialUrl, Tickers{})
					if err != nil {
						fmt.Println(err)
						os.Exit(-1)
					}
					// append the contracts back to ticker.Results
					ticker.Results = append(ticker.Results, extra.Results...)
				}
				initialUrl = extra.Next
				if initialUrl == "" {
					break
				}
			}
			ch <- ticker
			stockCh <- symbol
		}(stocks[i], ch, stockCh)
	}

	// append all results into slices
	for i := 0; i < len(stocks); i++ {
		tickerArr = append(tickerArr, <-ch)
		stockArr = append(stockArr, <-stockCh)
	}

	// clear the empty data
	for i := 0; i < len(stockArr); i++ {
		if len(tickerArr[i].Results) == 0 {
			stockArr = append(stockArr[:i], stockArr[(i+1):]...)
			tickerArr = append(tickerArr[:i], tickerArr[(i+1):]...)
			if i == len(stockArr)-1 {
				break
			} else {
				i--
			}
		} else {
			var row []string
			for j := 0; j < len(tickerArr[i].Results); j++ {
				row = append(row, tickerArr[i].Results[j].Ticker)
			}
			tickersMap[stockArr[i]] = row
		}
	}

	fmt.Printf("[%9.5fs] total available ticker(s): %v\n", time.Since(start).Seconds(), len(stockArr))
	// sort.Strings(stockArr)
	// err := createJson(map[string][]string{"tickers": stockArr}, "valid_tickers.json")
	// if err != nil {
	// 	return nil, err
	// }
	return tickersMap, nil
}

/*

get the details of each options

args
stocks ([]string): slice of tickers

return
contractMap (map[string][]Data): map of stocks' option contracts details (strike, maturity, ivol, name)
error (error): error message

*/
func GetDetails(stocks []string) (map[string][]Data, error) {
	//initialize variables
	exportMap := map[string][]Data{}

	// get the tickerMaps
	data, err := GetTickers(stocks)
	if err != nil {
		return nil, err
	}

	//timing
	start := time.Now()
	bar := progressBar(len(data)) // initialize cmd line progress bar
	for i, v := range data {
		bar.Describe(fmt.Sprintf("Processing %v\t", i))
		// initilize variables
		var detailsArr []Data
		ch := make(chan Data, len(v))
		defer close(ch)
		// get the dividend yield of equities from alphavantage
		co, _ := getAlphavantage(fmt.Sprintf("https://www.alphavantage.co/query?function=OVERVIEW&symbol=%v", i), Overview{})
		// convert string to float64
		dy, err := strconv.ParseFloat(co.DividentYield, 64)
		if err != nil {
			dy = 0.
		}
		// goroutines
		for j := 0; j < len(v); j++ {
			go func(stock string, contract string, dy float64) {
				var option string
				url := fmt.Sprintf("https://api.polygon.io/v3/snapshot/options/%v/%v", stock, contract)
				// get the contract details from polygon
				result, err := getPolygon(url, TickerDetails{})
				if err != nil {
					fmt.Println("here")
					fmt.Println(err)
					os.Exit(-1)
				}
				// initialize the values
				expiry := result.Results.Details.ExpirationDate
				strike := result.Results.Details.StrikePrice
				underlying := result.Results.UnderlyingAsset.Price
				callPut := result.Results.Details.ContractType
				close := result.Results.Day.Close
				k := strike / underlying
				// return empty data if not match requirement
				if (strike >= underlying && callPut == "put") || (strike <= underlying && callPut == "call") || close == 0. || k < 0.5 || k > 2.0 {
					ch <- Data{}
					return
				}
				if callPut == "call" {
					option = "c"
				} else {
					option = "p"
				}
				p := close
				t, err := time.Parse("2006-01-02", expiry)
				if err != nil {
					fmt.Printf("at stock %v", stock)
					fmt.Println(err)
					os.Exit(-1)
				}
				maturity := float64(t.Unix()-time.Now().Unix()) / float64(60*60*24*365)
				r := 0.03
				// calculate implied volatility based on black-scholes model
				ivol := fit(p, strike, underlying, maturity, dy, r, option)
				ch <- Data{K: k, T: maturity, Ivol: ivol, Name: contract}
			}(i, v[j], dy)
		}

		for i := 0; i < len(v); i++ {
			details := <-ch
			if details.K > 0.5 && details.K < 2.0 {
				detailsArr = append(detailsArr, details)
			}
		}
		sort.Slice(detailsArr, func(i, j int) bool { return detailsArr[i].K <= detailsArr[j].K })
		exportMap[i] = detailsArr
		bar.Add(1)
		err = createJson(map[string][]Data{"results": detailsArr}, fmt.Sprintf("storage/%v_ivol.json", i))
		if err != nil {
			return nil, err
		}
	}
	fmt.Printf("[%9.5fs] requested details from api\n", time.Since(start).Seconds())
	return exportMap, nil
}

/*

get the model parameters

args
stocks ([]string): slice of tickers

return
contractMap (map[string]mc.Model): map of models parameters for each stock
error (error): error message

*/
func Calibrate(stocks []string) (map[string]mc.Model, error) {
	modelsMap := make(map[string]mc.Model)
	// if parameters.json is updated within one day, then just read the file
	file, err := os.Stat("parameters.json")
	if err != nil {
		return nil, err
	}

	modTime, _ := time.Parse(Layout, file.ModTime().Format(Layout))
	tNow, _ := time.Parse(Layout, time.Now().Format(Layout))
	if modTime.Equal(tNow) {
		paraMaps, _ := Open("parameters.json", Model{})
		for k, v := range paraMaps {
			modelsMap[k] = v
		}
		return modelsMap, nil
	}

	// get the contract details
	data, err := GetDetails(stocks)
	if err != nil {
		return nil, err
	}

	ch := make(chan mc.Model, len(data))
	stocksCh := make(chan string, len(data))
	defer close(ch)
	defer close(stocksCh)
	// fit the contract details to the model, save the model parameters
	for k, v := range data {
		go func(stock string, data []Data, ch chan mc.Model, stocksCh chan string) {
			var model mc.Model = mc.NewHypHyp()
			d, err := loadMktData(data)
			if err != nil {
				fmt.Println(err)
			}
			model = mc.Fit(model, d)
			ch <- model
			stocksCh <- stock
		}(k, v, ch, stocksCh)
	}

	for i := 0; i < len(data); i++ {
		modelsMap[<-stocksCh] = <-ch
	}

	// generate output parameters.json data
	err = createJson(modelsMap, "parameters.json")
	if err != nil {
		return nil, err
	}
	return modelsMap, nil
}

// sample shortlisted tickers from big modelsMap
func ModelSample(stocks []string, models map[string]mc.Model) map[string]mc.Model {
	result := make(map[string]mc.Model)
	for i := 0; i < len(stocks); i++ {
		result[stocks[i]] = models[stocks[i]]
	}
	return result
}
