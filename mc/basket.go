package mc

import (
	"sort"
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

func (b Basket) Path(stocks []string, obsdates []time.Time, pxRatio map[string]float64, d *distmv.Normal, d2 distuv.Normal) MCPath {
	nAssets := len(b)
	n := len(obsdates) - 1

	z := map[string][]float64{}
	for k := range pxRatio {
		z[k] = make([]float64, n)
	}
	r := make([]float64, nAssets)
	for k := 0; k < n; k++ {
		r = d.Rand(r)
		for i := range stocks {
			z[stocks[i]][k] = r[i]
		}
	}
	dt := make([]float64, n)
	for i := range dt {
		dt[i] = obsdates[i+1].Sub(obsdates[i]).Hours() / (365.0 * 24.0)
	}
	x := make(map[string][]float64)
	for _, v := range b {
		x[v.Ticker] = v.Model.Path(pxRatio[v.Ticker], dt, z[v.Ticker], d2)
	}
	return x
}

// Constructor for basket
func NewBasket(modelsMap map[string]Model) Basket {
	var b Basket

	// Lookup calibrated model parameters
	for k, v := range modelsMap {
		b = append(b, Stock{Ticker: k, Model: v})
		sort.Slice(b, func(i, j int) bool { return b[i].Ticker <= b[j].Ticker })
	}
	return b
}
