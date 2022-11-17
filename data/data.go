package data

import (
	"encoding/json"
	"errors"
	"fmt"
	m "main/mc"
	"math"
	"net/http"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/schollz/progressbar/v3"
	"gonum.org/v1/gonum/mat"
	"gonum.org/v1/gonum/optimize"
	"gonum.org/v1/gonum/stat"
	"gonum.org/v1/gonum/stat/distuv"
)

type TickersArr struct {
	Tickers []string `json:"tickers"`
}

type TickersPara struct {
	Ticker string `json:"ticker"`
}

type Tickers struct {
	Results []TickersPara `json:"results"`
	Next    string        `json:"next_url"`
}

type TickerData struct {
	ContractType      string  `json:"contract_type"`
	ExerciseStyle     string  `json:"exercise_style"`
	ExpirationDate    string  `json:"expiration_date"`
	SharesPerContract float64 `json:"shares_per_contract"`
	StrikePrice       float64 `json:"strike_price"`
	Ticker            string  `json:"ticker"`
}

type TickerDay struct {
	Change        float64 `json:"change"`
	ChangePercent float64 `json:"change_percent"`
	Close         float64 `json:"close"`
	High          float64 `json:"high"`
	LastUpdated   int64   `json:"last_updated"`
	Low           float64 `json:"low"`
	Open          float64 `json:"open"`
	PreviousClose float64 `json:"previous_close"`
	Volume        int     `json:"volume"`
	Vwap          float64 `json:"vwap"`
}

type TickerUnderlying struct {
	ChangeToBreakEven float64 `json:"change_to_break_even"`
	LastUpdated       int64   `json:"last_updated"`
	Price             float64 `json:"price"`
	Ticker            string  `json:"ticker"`
	Timeframe         string  `json:"timeframe"`
}

type TickerResult struct {
	BreakEvenPrice    float64          `json:"break_even_price"`
	Day               TickerDay        `json:"day"`
	Details           TickerData       `json:"details"`
	ImpliedVolatility float64          `json:"implied_volatility"`
	UnderlyingAsset   TickerUnderlying `json:"underlying_asset"`
}

type TickerDetails struct {
	Results TickerResult `json:"results"`
}

type Data struct {
	K    float64 `json:"K"`
	T    float64 `json:"T"`
	Ivol float64 `json:"Ivol"`
	Name string  `json:"Name"`
}

type Hist struct {
	Close string `json:"4. close"`
}

type AlphaData struct {
	Hist map[string]Hist `json:"Time Series (Daily)"`
}

type Overview struct {
	DividentYield string `json:"DividendYield"`
}

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
func Calibrate(stocks []string) (map[string]m.Model, error) {
	data, err := GetDetails(stocks)
	if err != nil {
		return nil, err
	}

	modelsMap := make(map[string]m.Model)
	ch := make(chan m.Model, len(data))
	stocksCh := make(chan string, len(data))
	defer close(ch)
	defer close(stocksCh)
	for k, v := range data {
		go func(stock string, data []Data, ch chan m.Model, stocksCh chan string) {
			var model m.Model = m.NewHypHyp()
			d, err := loadMktData(data)
			if err != nil {
				fmt.Println(err)
			}
			model = m.Fit(model, d)
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

// get the correlation matrix
func Statistics(stocks []string) ([]float64, *mat.SymDense, map[string]float64, error) {
	ch := make(chan map[string]Hist, len(stocks))
	stockch := make(chan string, len(stocks))
	defer close(ch)
	for i := 0; i < len(stocks); i++ {
		go func(symbol string, ch chan map[string]Hist, stockch chan string) {
			px, err := getAlphavantage(fmt.Sprintf("https://www.alphavantage.co/query?function=TIME_SERIES_DAILY_ADJUSTED&symbol=%v", symbol), AlphaData{})
			if err != nil {
				err = errors.New("in corrmatrix(), http error: AlphaData{}")
				fmt.Println(err)
				os.Exit(-1)
			}
			ch <- px.Hist
			stockch <- symbol
		}(stocks[i], ch, stockch)
	}
	stockpx := map[string]map[string]Hist{}
	for i := 0; i < len(stocks); i++ {
		stockpx[<-stockch] = <-ch
	}
	refDate := time.Now().AddDate(0, -3, -1).Format("2006-01-02")
	rx := map[string][]float64{}
	var rxArr [][]float64
	mu := map[string]float64{}
	var muArr []float64
	spotRef := map[string]float64{}
	for k, v := range stockpx {
		var px []float64
		var rt []float64
		dateArr := reflect.ValueOf(v).MapKeys()
		sort.Slice(dateArr, func(i, j int) bool {
			return dateArr[i].String() > dateArr[j].String()
		})
		i := sort.Search(len(dateArr), func(i int) bool { return dateArr[i].String() < refDate })
		dateArr = dateArr[:i]
		for _, t := range dateArr {
			p, _ := strconv.ParseFloat(v[t.String()].Close, 64)
			px = append(px, p)
		}
		total_rt := 0.0
		for c := 0; c < len(px)-1; c++ {
			total_rt += math.Log(px[c] / px[c+1])
			rt = append(rt, math.Log(px[c]/px[c+1]))
		}
		rx[k] = rt
		mu[k] = total_rt / float64(len(rt))
		spotRef[k], _ = strconv.ParseFloat(v[dateArr[0].String()].Close, 64)
		rxArr = append(rxArr, rt)
	}
	minLength := minLength(rxArr)

	data := mat.NewDense(minLength, len(stocks), nil)
	for i := 0; i < len(stocks); i++ {
		data.SetCol(i, rx[stocks[i]][:minLength])
		muArr = append(muArr, mu[stocks[i]])
	}
	var corr mat.SymDense
	stat.CorrelationMatrix(&corr, data, nil)
	corrMatrix := &corr
	return muArr, corrMatrix, spotRef, nil
}

// get the spot price
func SpotPx(stocks []string) map[string]float64 {
	ch := make(chan map[string]Hist, len(stocks))
	stockch := make(chan string, len(stocks))
	defer close(ch)
	for i := 0; i < len(stocks); i++ {
		go func(symbol string, ch chan map[string]Hist, stockch chan string) {
			px, err := getAlphavantage(fmt.Sprintf("https://www.alphavantage.co/query?function=TIME_SERIES_DAILY_ADJUSTED&symbol=%v", symbol), AlphaData{})
			if err != nil {
				err = errors.New("in SpotPx(), http error: AlphaData{}")
				fmt.Println(err)
				os.Exit(-1)
			}
			ch <- px.Hist
			stockch <- symbol
		}(stocks[i], ch, stockch)
	}
	stockpx := map[string]map[string]Hist{}
	for i := 0; i < len(stocks); i++ {
		stockpx[<-stockch] = <-ch
	}
	spotRef := map[string]float64{}
	for k, v := range stockpx {
		dateArr := reflect.ValueOf(v).MapKeys()
		sort.Slice(dateArr, func(i, j int) bool {
			return dateArr[i].String() > dateArr[j].String()
		})
		spotRef[k], _ = strconv.ParseFloat(v[dateArr[0].String()].Close, 64)
	}
	return spotRef
}

// assgin index to every stocks
func stockIndex(stocks []string) map[string]int {
	result := map[string]int{}
	for i, v := range stocks {
		result[v] = i
	}
	return result
}

// get the correlation from the matrix
func Corr(stocks []string, matrix *mat.SymDense, s1, s2 string) (float64, error) {
	var err error
	s1, s2 = strings.ToUpper(s1), strings.ToUpper(s2)
	stocksMap := stockIndex(stocks)
	idx1, ok1 := stocksMap[s1]
	if !ok1 {
		err = fmt.Errorf("cannot find %v in the ticker list", s1)
		return math.NaN(), err
	}
	idx2, ok2 := stocksMap[s2]
	if !ok2 {
		err = fmt.Errorf("cannot find %v in the ticker list", s2)
		return math.NaN(), err
	}
	return matrix.At(idx1, idx2), nil
}

// helper function to get the http request and store into struct
// input: link, and the target struct type
func getPolygon[DataType TickerDetails | Tickers | Hist](url string, target DataType) (result DataType, err error) {
	req, err := http.NewRequest("GET", url, nil)
	req.Header.Add("Authorization", `Bearer 3X8wQrb0pH9gaNJvY__sq1UohDdHfVt3`)
	if err != nil {
		return target, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println(err)
		return target, err
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&target)
	if err != nil {
		return
	}
	result = target
	return result, nil
}

// helper function to get the http request and store into struct
// input: link, and the target struct type
func getAlphavantage[DataType Overview | AlphaData](url string, target DataType) (result DataType, err error) {
	req, err := http.NewRequest("GET", url, nil)
	q := req.URL.Query()
	q.Add("apikey", "0JA1NVBAL8CHYSCH")
	req.URL.RawQuery = q.Encode()
	if err != nil {
		return target, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println(err)
		return target, err
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&target)
	if err != nil {
		return
	}
	result = target
	return result, nil
}

func bs(k, s, sigma, T, dy, r float64, option string) float64 {
	x := sigma * math.Sqrt(T)
	d1 := (math.Log(s/k) + 0.5*sigma*sigma*T) / x
	d2 := d1 - x

	N := distuv.Normal{Mu: 0.0, Sigma: 1.0}

	premium := s*math.Exp(-dy*T)*N.CDF(d1) - k*math.Exp(-r*T)*N.CDF(d2)
	if option == "p" {
		premium = -s*math.Exp(-dy*T)*N.CDF(-d1) + k*math.Exp(-r*T)*N.CDF(-d2)
	}
	return premium
}

func loss(par []float64, p, k, s, T, dy, r float64, option string) float64 {
	par[0] = math.Exp(par[0])
	loss := math.Pow(p-bs(k, s, par[0], T, dy, r, option), 2)
	return loss
}

func fit(p, k, s, T, dy, r float64, option string) float64 {
	par := []float64{math.Log(0.5)}
	problem := optimize.Problem{
		Func: func(par []float64) float64 {
			return loss(par, p, k, s, T, dy, r, option)
		},
	}
	res, err := optimize.Minimize(problem, par, nil, &optimize.NelderMead{})
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
	result := math.Exp(res.X[0])
	return result
}

// helper function to create json file
func createJson[T map[string][]Data | map[string][]string | map[string]m.Model](raw T, filename string) error {
	data, err := json.MarshalIndent(raw, "", " ")
	if err != nil {
		return err
	}
	err = os.WriteFile(filename, data, 0644)
	if err != nil {
		return err
	}
	return nil
}

func progressBar(length int) *progressbar.ProgressBar {
	bar := progressbar.NewOptions(
		length,
		progressbar.OptionSetPredictTime(false),
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionClearOnFinish(),
		progressbar.OptionUseANSICodes(true),
		progressbar.OptionShowCount(),
		progressbar.OptionSetWidth(20),
		progressbar.OptionSetVisibility(true),
		progressbar.OptionShowDescriptionAtLineEnd(),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "[green]=[reset]",
			SaucerHead:    "[green]>[reset]",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}))
	return bar
}

func minLength(data [][]float64) int {
	min := len(data[0])
	if len(data) == 1 {
		return min
	}
	for i := 1; i < len(data); i++ {
		if len(data[i]) < min {
			min = len(data[i])
		}
	}
	return min
}

// helper function to open csv file
func loadMktData(data []Data) ([][]float64, error) {
	var x [][]float64
	for i := 0; i < len(data); i++ {
		x = append(x, []float64{data[i].K, data[i].T, data[i].Ivol})
	}
	return x, nil
}
