package mc

import (
	"time"

	"gonum.org/v1/gonum/stat/distmv"
	"gonum.org/v1/gonum/stat/distuv"
)

// A stock is identified by its ticker and its calibrated model
type Stock struct {
	Ticker string
	Model  Model
}

type Basket []Stock

// Holds mc path of a basket of stocks
type MCPath map[string][]float64

type BasketList struct {
	Stocks     []string           `json:"stocks"`
	Parameter  map[string]Model   `json:"model_parameters"`
	Index      map[string]int     `json:"corr_id"`
	CorrMatrix []float64          `json:"corr_matrix"`
	Mean       []float64          `json:"mean"`
	SpotPrice  map[string]float64 `json:"spot_price"`
}

func (b Basket) Path(obsdates []time.Time, pxRatio []float64, d *distmv.Normal, d2 distuv.Normal) (MCPath, error) {
	nAssets := len(b)
	n := len(obsdates) - 1

	z := make([][]float64, nAssets)
	for i := 0; i < nAssets; i++ {
		z[i] = make([]float64, n)
	}
	r := make([]float64, nAssets)
	for k := 0; k < n; k++ {
		r = d.Rand(r)
		for i := 0; i < nAssets; i++ {
			z[i][k] = r[i]
		}
	}
	dt := make([]float64, n)
	for i := range dt {
		dt[i] = obsdates[i+1].Sub(obsdates[i]).Hours() / (365.0 * 24.0)
	}
	x := make(map[string][]float64)
	for i, v := range b {
		x[v.Ticker] = b[i].Model.Path(pxRatio[i], dt, z[i], d2)
	}
	return x, nil
}

// Constructor for basket
func NewBasket(modelsMap map[string]Model) (Basket, error) {
	var b Basket

	// Lookup calibrated model parameters
	for k, v := range modelsMap {
		//m, err = utils.GetModel(v)
		b = append(b, Stock{Ticker: k, Model: v})
	}
	return b, nil
}
