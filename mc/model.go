package mc

import (
	"log"
	"math"

	"gonum.org/v1/gonum/optimize"
)

// Model interface to be satisfied by option pricing model types.
type Model interface {
	//Compute a price path under model
	Path([]float64, []float64) []float64
	//Compute implied vol under the model
	IVol(float64, float64) float64
	//Get transformed model parameters. Return parameters mapped to the domain (-Inf, Inf)
	Get() []float64
	//Create a model for the given transformed parameters
	Set([]float64) Model
}

// Calibrate the given model to input data d. d is an Nx3 slice, with moneyness values in the first column, maturity in years in the second column and market implied volatility in the third column.
func Fit(m Model, d [][]float64) Model {
	par := m.Get()
	problem := optimize.Problem{
		Func: func(par []float64) float64 {
			return mse(m, par, d)
		},
	}
	res, err := optimize.Minimize(problem, par, nil, &optimize.NelderMead{})
	if err != nil {
		log.Fatal(err)
	}
	m = m.Set(res.X)
	return m
}

// Compute MSE between model implied vols and market vols.
func mse(m Model, par []float64, d [][]float64) float64 {
	m = m.Set(par)
	loss := 0.0
	v := 0.0
	for i := range d {
		v = m.IVol(d[i][0], d[i][1])
		loss += math.Pow(v-d[i][2], 2)
	}
	return loss / float64(len(d)) // added denominator
}
