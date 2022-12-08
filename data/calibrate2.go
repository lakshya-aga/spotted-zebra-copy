package data

import (
	"fmt"
)

func GetSingleTickers(stock string) ([]string, error) {
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

