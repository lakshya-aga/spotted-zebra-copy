package data

import (
	"database/sql"
	"errors"
	"fmt"
	"math"
	"os"
	"reflect"
	"sort"
	"strconv"
	"time"

	"gonum.org/v1/gonum/mat"
	"gonum.org/v1/gonum/stat"
)

/*
get the correlation matrix

args
stocks ([]string): slice of tickers

return
mean (map[string]float64): map of mean for each stock
corrMatrix (*mat.SymDense): correlation matrix for each stock
spotPrice (map[string]float64): spot price for each stock
error (error): error message
*/
func Statistics(stocks []string, db *sql.DB) (map[string]float64, *mat.SymDense, map[string]float64, error) {
	var err error
	today := time.Now().Format(Layout)
	update, err := updateCorr(db, today)
	if err != nil {
		panic(err)
	}

	if !update {
		var corr []float64
		for i := 0; i < len(stocks); i++ {
			for j := 0; j < len(stocks); j++ {
				if i == j {
					corr = append(corr, 1.0)
				} else if i > j {
					val, err := getCorr(db, today, stocks[j], stocks[i])
					if err != nil {
						panic(err)
					}
					corr = append(corr, val)
				} else {
					val, err := getCorr(db, today, stocks[i], stocks[j])
					if err != nil {
						panic(err)
					}
					corr = append(corr, val)
				}
			}
		}
		corrMatrix := mat.NewSymDense(len(stocks), corr)
		mu, fixings, err := getStats(db, today)
		if err != nil {
			panic(err)
		}
		return mu, corrMatrix, fixings, nil
	}

	// initialize variables
	ch := make(chan map[string]Hist, len(stocks))
	stockch := make(chan string, len(stocks))
	defer close(ch)
	// goroutines
	for i := 0; i < len(stocks); i++ {
		go func(symbol string, ch chan map[string]Hist, stockch chan string) {
			// get the daily prices for the stocks
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
	// save the prices within price map
	stockpx := map[string]map[string]Hist{}
	for i := 0; i < len(stocks); i++ {
		stockpx[<-stockch] = <-ch
	}
	// get the date 3 months before
	refDate := time.Now().AddDate(0, -3, -1).Format(Layout)
	// initialize variables
	rx := map[string][]float64{}
	var rxArr [][]float64
	mu := map[string]float64{}
	fixings := map[string]float64{}
	// calculate the returns and means, and save the spot price
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
		for c := 0; c < len(px)-1; c++ {
			rt = append(rt, math.Log(px[c]/px[c+1]))
		}
		rx[k] = rt
		mu[k] = stat.Mean(rt, nil)
		fixings[k], _ = strconv.ParseFloat(v[dateArr[0].String()].Close, 64)
		rxArr = append(rxArr, rt)
	}
	// make the slice equal length
	minLength := minLength(rxArr)

	// get the correlation matrix
	data := mat.NewDense(minLength, len(stocks), nil)
	for i := 0; i < len(stocks); i++ {
		data.SetCol(i, rx[stocks[i]][:minLength])
	}
	var corr mat.SymDense
	stat.CorrelationMatrix(&corr, data, nil)
	corrMatrix := &corr
	var corrPairs []Corr

	// get the index for the stocks position
	stocksMap := stockIndex(stocks)

	insertCorr := `insert into "CorrPairs"("Date", "X0", "X1", "Corr") values($1, $2, $3, $4)`
	insertStat := `insert into "Statistics"("Date", "Ticker", "Index", "Mean", "Fixing") values($1, $2, $3, $4, $5)`
	// save the correlation pairs
	for i := range stocks {
		for j := range stocks {
			if i < j {
				corrPairs = append(corrPairs, Corr{X1: i, X2: j, Corr: corrMatrix.At(i, j)})
				// add data to database
				_, err = db.Exec(insertCorr, time.Now().Format(Layout), stocks[i], stocks[j], corrMatrix.At(i, j))
				if err != nil {
					panic(err)
				}
			}
		}
		// add data to database
		_, err = db.Exec(insertStat, time.Now().Format(Layout), stocks[i], stocksMap[stocks[i]], mu[stocks[i]], fixings[stocks[i]])
		if err != nil {
			panic(err)
		}
	}

	// output data
	statOutput := Stat{SpotPrice: fixings, Mean: mu, Index: stocksMap, CorrPairs: corrPairs}
	err = createJson(statOutput, "statistics.json")
	if err != nil {
		return nil, nil, nil, err
	}
	return mu, corrMatrix, fixings, nil
}

/*
get the latest price (for future price evaluation use)

args
stocks ([]string): slice of tickers

return
mean (map[string]float64): map of latest price for each stock
*/
func LatestPx(stocks []string) map[string]float64 {
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

// sample the correlation matrix
func CorrSample(stocks []string, idx map[string]int, corrMatrix *mat.SymDense) *mat.SymDense {
	var corr []float64
	for i := range stocks {
		for j := range stocks {
			if i == j {
				corr = append(corr, 1.0)
			} else if i < j {
				corr = append(corr, corrMatrix.At(idx[stocks[j]], idx[stocks[i]]))
			} else {
				corr = append(corr, corrMatrix.At(idx[stocks[i]], idx[stocks[j]]))
			}
		}
	}
	sampleCorr := mat.NewSymDense(len(stocks), corr)
	return sampleCorr
}

func getStats(db *sql.DB, today string) (map[string]float64, map[string]float64, error) {
	rows, err := db.Query(`SELECT "Ticker", "Mean", "Fixing" FROM "Statistics" WHERE "Date" IN ($1)`, today)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	means := map[string]float64{}
	fixings := map[string]float64{}
	for rows.Next() {
		var ticker string
		var mean, fixing float64
		err = rows.Scan(&ticker, &mean, &fixing)
		if err != nil {
			return nil, nil, err
		}
		means[ticker] = mean
		fixings[ticker] = fixing
	}
	return means, fixings, nil
}

func getCorr(db *sql.DB, date, x0, x1 string) (float64, error) {
	var corr float64
	row := db.QueryRow(`SELECT "Corr" FROM "CorrPairs" WHERE "Date"=$1 AND "X0"=$2 AND "X1"=$3`, date, x0, x1)
	switch err := row.Scan(&corr); err {
	case sql.ErrNoRows:
		return math.NaN(), errors.New("no rows were returned")
	case nil:
		return corr, nil
	default:
		return math.NaN(), err
	}
}

func updateCorr(db *sql.DB, date string) (bool, error) {
	rows, err := db.Query(`SELECT "Date" FROM "CorrPairs" WHERE "Date" IN ($1)`, date)
	if err != nil {
		return false, err
	}
	defer rows.Close()
	var dates []string
	for rows.Next() {
		var dt string
		err = rows.Scan(&dt)
		if err != nil {
			return false, err
		}
		dates = append(dates, dt)
	}
	if len(dates) == 0 {
		return true, nil
	} else {
		return false, nil
	}
}

