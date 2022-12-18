package data

import (
	"database/sql"
	"errors"
	"fmt"
	"math"
	"reflect"
	"sort"
	"strconv"
	"time"

	"gonum.org/v1/gonum/mat"
	"gonum.org/v1/gonum/stat"
)

func getHistPx(stock string) (map[string]float64, error) {
	p := map[string]float64{}
	px, err := getAlphavantage(fmt.Sprintf("https://www.alphavantage.co/query?function=TIME_SERIES_DAILY_ADJUSTED&outputsize=full&symbol=%v", stock), AlphaData{})
	if err != nil {
		return nil, err
	}
	for k, v := range px.Hist {
		p[k], _ = strconv.ParseFloat(v.Close, 64)
	}
	return p, nil
}

func getReturns(p map[string]float64, refDate string) []float64 {
	var px, rt []float64
	dateArr := dateArr(p, refDate)

	for _, t := range dateArr {
		px = append(px, p[t.String()])
	}
	for c := 0; c < len(px)-1; c++ {
		rt = append(rt, math.Log(px[c]/px[c+1]))
	}
	return rt
}

func getTickerStats(stock string) (float64, float64, []float64, error) {
	px, err := getHistPx(stock)
	if err != nil {
		return math.NaN(), math.NaN(), nil, err
	}

	// get the date 3 months before
	refDate := time.Now().AddDate(0, -3, -1).Format(Layout)
	rt := getReturns(px, refDate)
	dateArr := dateArr(px, refDate)

	mu := stat.Mean(rt, nil)
	fixings := px[dateArr[0].String()]

	return mu, fixings, rt, nil
}

func corrMatrix(rxArr [][]float64) *mat.SymDense {
	// make the slice equal length
	minLength := minLength(rxArr)

	// get the correlation matrix
	data := mat.NewDense(minLength, len(rxArr), nil)
	for i := 0; i < len(rxArr); i++ {
		data.SetCol(i, rxArr[i][:minLength])
	}
	var corr mat.SymDense
	stat.CorrelationMatrix(&corr, data, nil)
	corrMatrix := &corr
	return corrMatrix
}

func dateArr(p map[string]float64, refDate string) []reflect.Value {
	dateArr := reflect.ValueOf(p).MapKeys()
	sort.Slice(dateArr, func(i, j int) bool { return dateArr[i].String() > dateArr[j].String() })
	i := sort.Search(len(dateArr), func(i int) bool { return dateArr[i].String() < refDate })
	dateArr = dateArr[:i]
	return dateArr
}

func NewStatistics(stocks []string, db *sql.DB, date string) (map[string]float64, *mat.SymDense, map[string]float64, error) {
	var err error
	mean := map[string]float64{}
	fixings := map[string]float64{}
	var rx [][]float64
	muCh := make(chan float64, len(stocks))
	fixCh := make(chan float64, len(stocks))
	rtCh := make(chan []float64, len(stocks))
	stockCh := make(chan string, len(stocks))
	errCh := make(chan error, len(stocks))
	defer close(muCh)
	defer close(fixCh)
	defer close(rtCh)
	defer close(stockCh)
	defer close(errCh)

	for i := range stocks {
		go func(i int, muCh chan float64, fixCh chan float64, rtCh chan []float64, stockCh chan string, errCh chan error) {
			mu, fixing, rt, err := getTickerStats(stocks[i])
			muCh <- mu
			fixCh <- fixing
			rtCh <- rt
			errCh <- err
			stockCh <- stocks[i]
		}(i, muCh, fixCh, rtCh, stockCh, errCh)
	}

	bar := progressBar(len(stocks))
	for i := range stocks {
		bar.Describe(fmt.Sprintf("Computing the statistics for %v\t", stocks[i]))
		err := <-errCh
		if err != nil {
			return nil, nil, nil, err
		}
		s := <-stockCh
		mean[s] = <-muCh
		fixings[s] = <-fixCh
		rx = append(rx, <-rtCh)
		bar.Add(1)
	}
	corr := corrMatrix(rx)

	// get the index for the stocks position
	stocksMap := stockIndex(stocks)
	insertCorr := `insert into "CorrPairs"("Date", "X0", "X1", "Corr") values($1, $2, $3, $4)`
	insertStat := `insert into "Statistics"("Date", "Ticker", "Index", "Mean", "Fixing") values($1, $2, $3, $4, $5)`

	// save the correlation pairs
	for i := range stocks {
		for j := range stocks {
			if i < j {
				// add data to database
				_, err = db.Exec(insertCorr, date, stocks[i], stocks[j], corr.At(i, j))
				if err != nil {
					return nil, nil, nil, err
				}
			}
		}
		// add data to database
		_, err = db.Exec(insertStat, date, stocks[i], stocksMap[stocks[i]], mean[stocks[i]], fixings[stocks[i]])
		if err != nil {
			return nil, nil, nil, err
		}
	}
	return mean, corr, fixings, nil
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

func Statistics(stocks []string, db *sql.DB) (map[string]float64, *mat.SymDense, map[string]float64, error) {
	today := time.Now().Format(Layout)
	update, err := UpdateRequired("CorrPairs", db, today)
	if err != nil {
		return nil, nil, nil, err
	}

	if !update {
		fmt.Println("Retrieving statistics from database")
		var corr []float64
		for i := 0; i < len(stocks); i++ {
			for j := 0; j < len(stocks); j++ {
				if i == j {
					corr = append(corr, 1.0)
				} else if i > j {
					val, err := getCorr(db, today, stocks[j], stocks[i])
					if err != nil {
						return nil, nil, nil, err
					}
					corr = append(corr, val)
				} else {
					val, err := getCorr(db, today, stocks[i], stocks[j])
					if err != nil {
						return nil, nil, nil, err
					}
					corr = append(corr, val)
				}
			}
		}
		corrMatrix := mat.NewSymDense(len(stocks), corr)
		mu, fixings, err := getStats(db, today)
		if err != nil {
			return nil, nil, nil, err
		}
		return mu, corrMatrix, fixings, nil
	}

	fmt.Println("Computing new statistics")
	mean, corr, fixings, err := NewStatistics(stocks, db, today)
	if err != nil {
		return nil, nil, nil, err
	}
	fmt.Println("Saved new statistics into database")
	return mean, corr, fixings, nil
}

func StatsSample(stocks []string, idx map[string]int, corrMatrix *mat.SymDense, allmu map[string]float64, allspotref map[string]float64) ([]float64, *mat.SymDense, map[string]float64) {
	var corr []float64
	var sampleMu []float64
	sampleRef := map[string]float64{}
	for i := range stocks {
		sampleMu = append(sampleMu, allmu[stocks[i]])
		sampleRef[stocks[i]] = allspotref[stocks[i]]
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
	return sampleMu, sampleCorr, sampleRef
}

func LatestPx(stocks []string) (map[string]float64, error) {
	pxCh := make(chan map[string]float64, len(stocks))
	stockCh := make(chan string, len(stocks))
	errCh := make(chan error, len(stocks))
	defer close(pxCh)
	defer close(stockCh)
	defer close(errCh)
	for i := 0; i < len(stocks); i++ {
		go func(i int, pxCh chan map[string]float64, stockCh chan string, errCh chan error) {
			px, err := getHistPx(stocks[i])
			pxCh <- px
			stockCh <- stocks[i]
			errCh <- err
		}(i, pxCh, stockCh, errCh)
	}
	stockpx := map[string]map[string]float64{}
	for i := 0; i < len(stocks); i++ {
		err := <-errCh
		if err != nil {
			return nil, err
		}
		stockpx[<-stockCh] = <-pxCh
	}
	spotRef := map[string]float64{}
	for k, v := range stockpx {
		dateArr := reflect.ValueOf(v).MapKeys()
		sort.Slice(dateArr, func(i, j int) bool {
			return dateArr[i].String() > dateArr[j].String()
		})
		spotRef[k] = v[dateArr[0].String()]
	}
	return spotRef, nil
}
