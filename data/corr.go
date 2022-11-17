package data

import (
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

// get the correlation matrix
func Statistics(stocks []string) (map[string]float64, *mat.SymDense, map[string]float64, error) {
	file, err := os.Stat("statistics.json")
	if err != nil {
		return nil, nil, nil, err
	}

	modTime, _ := time.Parse(Layout, file.ModTime().Format(Layout))
	tNow, _ := time.Parse(Layout, time.Now().Format(Layout))
	if modTime.Equal(tNow) {
		var corr []float64
		stat, _ := Open("statistics.json", Stat{})
		for i := 0; i < len(stat.Index); i++ {
			for j := 0; j < len(stat.Index); j++ {
				if i == j {
					corr = append(corr, 1.0)
				} else if i > j {
					for k := range stat.CorrPairs {
						if stat.CorrPairs[k].X1 == stat.Index[j] && stat.CorrPairs[k].X2 == stat.Index[i] {
							corr = append(corr, stat.CorrPairs[k].Corr)
						}
					}
				} else {
					for k := range stat.CorrPairs {
						if stat.CorrPairs[k].X1 == stat.Index[i] && stat.CorrPairs[k].X2 == stat.Index[j] {
							corr = append(corr, stat.CorrPairs[k].Corr)
						}
					}
				}
			}
		}
		corrMatrix := mat.NewSymDense(len(stat.Index), corr)
		return stat.Mean, corrMatrix, stat.SpotPrice, nil
	}

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
		for c := 0; c < len(px)-1; c++ {
			rt = append(rt, math.Log(px[c]/px[c+1]))
		}
		rx[k] = rt
		mu[k] = stat.Mean(rt, nil)
		spotRef[k], _ = strconv.ParseFloat(v[dateArr[0].String()].Close, 64)
		rxArr = append(rxArr, rt)
	}
	minLength := minLength(rxArr)

	data := mat.NewDense(minLength, len(stocks), nil)
	for i := 0; i < len(stocks); i++ {
		data.SetCol(i, rx[stocks[i]][:minLength])
	}
	var corr mat.SymDense
	stat.CorrelationMatrix(&corr, data, nil)
	corrMatrix := &corr
	var corrPairs []Corr

	stocksMap := stockIndex(stocks)

	for i, v := range stocks {
		for j, k := range stocks {
			if i < j {
				corrPairs = append(corrPairs, Corr{X1: v, X2: k, Corr: corrMatrix.At(i, j)})
			}
		}
	}

	statOutput := Stat{SpotPrice: spotRef, Mean: mu, Index: stocksMap, CorrPairs: corrPairs}
	err = createJson(statOutput, "statistics.json")
	if err != nil {
		return nil, nil, nil, err
	}
	return mu, corrMatrix, spotRef, nil
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
func stockIndex(stocks []string) map[int]string {
	result := map[int]string{}
	for i, v := range stocks {
		result[i] = v
	}
	return result
}

// get the correlation from the matrix
// func Corr(stocks []string, matrix *mat.SymDense, s1, s2 string) (float64, error) {
// 	var err error
// 	s1, s2 = strings.ToUpper(s1), strings.ToUpper(s2)
// 	stocksMap := stockIndex(stocks)
// 	idx1, ok1 := stocksMap[s1]
// 	if !ok1 {
// 		err = fmt.Errorf("cannot find %v in the ticker list", s1)
// 		return math.NaN(), err
// 	}
// 	idx2, ok2 := stocksMap[s2]
// 	if !ok2 {
// 		err = fmt.Errorf("cannot find %v in the ticker list", s2)
// 		return math.NaN(), err
// 	}
// 	return matrix.At(idx1, idx2), nil
// }
