package data

import (
	"database/sql"

	"main/mc"
	"main/utils"

	"gonum.org/v1/gonum/mat"
)

// daily updates
func Initialize(stocks []string, db *sql.DB) (map[string]mc.Model, map[string]float64, *mat.SymDense, map[string]float64, error) {
	modelsMap, err := Calibrate(stocks, db)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	mean, corr, fixings, err := Statistics(stocks, db)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	return modelsMap, mean, corr, fixings, nil
}

// sample statistics and model parameters for selected stocks
func Sampler(defaultStocks, selectStocks []string, allModels map[string]mc.Model, allMeans map[string]float64, allCorr *mat.SymDense, allFixings map[string]float64) ([]string, map[string]mc.Model, []float64, *mat.SymDense, map[string]float64, error) {
	filterStocks, filterStocksIndex, err := utils.Filter(selectStocks, defaultStocks)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	sampleModels := ModelSample(filterStocks, allModels)
	sampleMu, sampleCorr, sampleRef := StatsSample(filterStocks, filterStocksIndex, allCorr, allMeans, allFixings)
	return filterStocks, sampleModels, sampleMu, sampleCorr, sampleRef, nil
}
