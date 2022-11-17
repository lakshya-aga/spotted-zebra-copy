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

// get the tickers that have options trading in market
func GetTickers(stocks []string) (map[string][]string, error) {
	tickersMap := map[string][]string{}
	var tickerArr []Tickers
	var stockArr []string
	ch := make(chan Tickers, len(stocks))
	stockCh := make(chan string, len(stocks))
	defer close(ch)
	defer close(stockCh)
	start := time.Now()
	for i := 0; i < len(stocks); i++ {
		go func(symbol string, ch chan Tickers, stockCh chan string) {
			var initialUrl string
			url := fmt.Sprintf("https://api.polygon.io/v3/reference/options/contracts?underlying_ticker=%v&limit=1000", symbol)
			ticker, err := getPolygon(url, Tickers{})
			if err != nil {
				fmt.Println(err)
				os.Exit(-1)
			}
			initialUrl = ticker.Next
			for {
				var extra Tickers
				if initialUrl != "" {
					extra, err = getPolygon(initialUrl, Tickers{})
					if err != nil {
						fmt.Println(err)
						os.Exit(-1)
					}
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

	for i := 0; i < len(stocks); i++ {
		tickerArr = append(tickerArr, <-ch)
		stockArr = append(stockArr, <-stockCh)
	}

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
	sort.Strings(stockArr)
	err := createJson(map[string][]string{"tickers": stockArr}, "valid_tickers.json")
	if err != nil {
		return nil, err
	}
	return tickersMap, nil
}

// get the details of each options
func GetDetails(stocks []string) (map[string][]Data, error) {
	data, err := GetTickers(stocks)
	if err != nil {
		return nil, err
	}
	start := time.Now()
	exportMap := map[string][]Data{}
	bar := progressBar(len(data))
	for i, v := range data {
		bar.Describe(fmt.Sprintf("Processing %v\t", i))
		var detailsArr []Data
		ch := make(chan Data, len(v))
		defer close(ch)
		co, _ := getAlphavantage(fmt.Sprintf("https://www.alphavantage.co/query?function=OVERVIEW&symbol=%v", i), Overview{})
		dy, err := strconv.ParseFloat(co.DividentYield, 64)
		if err != nil {
			dy = 0.
		}
		for j := 0; j < len(v); j++ {
			go func(stock string, contract string, dy float64) {
				var option string
				url := fmt.Sprintf("https://api.polygon.io/v3/snapshot/options/%v/%v", stock, contract)
				result, err := getPolygon(url, TickerDetails{})
				if err != nil {
					fmt.Println("here")
					fmt.Println(err)
					os.Exit(-1)
				}
				expiry := result.Results.Details.ExpirationDate
				strike := result.Results.Details.StrikePrice
				underlying := result.Results.UnderlyingAsset.Price
				callPut := result.Results.Details.ContractType
				close := result.Results.Day.Close
				k := strike / underlying
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

// get the model parameters
func Calibrate(stocks []string) (map[string]mc.Model, error) {
	modelsMap := make(map[string]mc.Model)
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

	data, err := GetDetails(stocks)
	if err != nil {
		return nil, err
	}

	ch := make(chan mc.Model, len(data))
	stocksCh := make(chan string, len(data))
	defer close(ch)
	defer close(stocksCh)
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

	err = createJson(modelsMap, "parameters.json")
	if err != nil {
		return nil, err
	}
	return modelsMap, nil
}

func ModelSample(stocks []string, models map[string]mc.Model) ([]string, map[string]mc.Model) {
	result := make(map[string]mc.Model)
	for i := 0; i < len(stocks); i++ {
		_, exist := models[stocks[i]]
		if exist {
			result[stocks[i]] = models[stocks[i]]
		} else {
			stocks = append(stocks[:i], stocks[(i+1):]...)
			if i == len(result)-1 {
				break
			} else {
				i--
			}
		}
	}
	sort.Strings(stocks)
	return stocks, result
}
