package data

import (
	"database/sql"
	"errors"
	"math"

	"gonum.org/v1/gonum/mat"
)

/*
get the statistics from database

args:
1. db : target database
2. date : latest date

returns:
1. map of mean of shortlisted stocks
2. map of fixings of shortlisted stocks
3. error
*/
func getStats(db *sql.DB, date string) (map[string]float64, map[string]float64, error) {
	rows, err := db.Query(`SELECT "Ticker", "Mean", "Fixing" FROM "Statistics" WHERE "Date" IN ($1)`, date)
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

/*
get the correlation matrix from database

args:
1. db : target database
2. date : latest date
3. x0 : stock 0
4. x1 : stock 1

returns:
1. correlation between two stocks
2. error
*/
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

/*
main handlers for computing statistics

args:
1. stocks : shortlisted stocks
2. db : target database

returns:
1. map of mean for shortlisted stocks
2. correlation matrix of shortlisted stocks
3. map of fixings for shortlisted stocks
4. error
*/
func Statistics(stocks []string, db *sql.DB, date string) (map[string]float64, *mat.SymDense, map[string]float64, error) {
	// fmt.Println("Retrieving statistics from database")

	var corr []float64
	for i := 0; i < len(stocks); i++ {
		for j := 0; j < len(stocks); j++ {
			if i == j {
				corr = append(corr, 1.0)
			} else if i > j {
				val, err := getCorr(db, date, stocks[j], stocks[i])
				if err != nil {
					return nil, nil, nil, err
				}
				corr = append(corr, val)
			} else {
				val, err := getCorr(db, date, stocks[i], stocks[j])
				if err != nil {
					return nil, nil, nil, err
				}
				corr = append(corr, val)
			}
		}
	}
	corrMatrix := mat.NewSymDense(len(stocks), corr)
	mu, fixings, err := getStats(db, date)
	if err != nil {
		return nil, nil, nil, err
	}
	return mu, corrMatrix, fixings, nil
}

/*
sample statistics from shortlisted stocks

args:
1. stocks : selected stocks
2. idx : index of stocks in the slice
3. corrMatrix : correlation matrix of shortlisted stocks
4. allmu : map of mean of shortlisted stocks
5. allFixings : map of fixings of shortlisted stocks

returns:
1. map of mean of selected stocks
2. correlation matrix of selected stocks
3. map of fixings of selected stocks
*/
func StatsSample(stocks []string, idx map[string]int, corrMatrix *mat.SymDense, allmu map[string]float64, allFixings map[string]float64) ([]float64, *mat.SymDense, map[string]float64) {
	var corr []float64
	var sampleMu []float64
	sampleRef := map[string]float64{}
	for i := range stocks {
		sampleMu = append(sampleMu, allmu[stocks[i]])
		sampleRef[stocks[i]] = allFixings[stocks[i]]
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

/*
get the latest historical price of selected stocks

args:
1. stocks : selected stocks
2. db : target database

returns:
1. map of latest price of selected stocks
2. error
*/
func LatestPx(stocks []string, db *sql.DB) (map[string]float64, error) {
	spotRef := map[string]float64{}
	for i := range stocks {
		var px float64
		row := db.QueryRow(`SELECT "Fixing" FROM "Statistics" WHERE "Ticker"=$1 ORDER BY "Date" DESC LIMIT 1`, stocks[i])
		switch err := row.Scan(&px); err {
		case sql.ErrNoRows:
			return nil, errors.New("no rows were returned")
		case nil:
			spotRef[stocks[i]] = px
		default:
			return nil, err
		}
	}
	return spotRef, nil
}
