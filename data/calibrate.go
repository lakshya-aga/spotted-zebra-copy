package data

import (
	"database/sql"

	"github.com/banachtech/spotted-zebra/mc"
)

/*
get the model parameter from database

args:
1. db : target database
2. date : latest date

returns:
1. map of model parameters of shortlisted stocks
2. error
*/
func getModel(db *sql.DB, date string) (map[string]mc.Model, error) {
	rows, err := db.Query(`SELECT "Ticker", "Sigma", "Alpha", "Beta", "Kappa", "Rho" FROM "ModelParameters" WHERE "Date" IN ($1)`, date)
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

/*
main handlers for calibrating model parameters

args:
1. stocks : shortlisted stocks
2. db : target database

returns:
1. map of model parameters of shortlisted stocks
2. error
*/
func Calibrate(db *sql.DB, date string) (map[string]mc.Model, error) {
	// fmt.Println("Retrieving parameters from database")

	modelsMap, err := getModel(db, date)
	if err != nil {
		return nil, err
	}
	return modelsMap, nil
}

/*
sample model parameters from shortlisted stocks

args:
1. stocks : selected stocks
2. models : map of model parameters of shortlisted stocks

returns:
1. map of model parameters of selected stocks
*/
func ModelSample(stocks []string, models map[string]mc.Model) map[string]mc.Model {
	result := map[string]mc.Model{}
	for i := 0; i < len(stocks); i++ {
		result[stocks[i]] = models[stocks[i]]
	}
	return result
}
