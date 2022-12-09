package handler

import (
	"errors"
	"main/data"
	"main/mc"
	"main/payoff"
	"main/utils"
	"math"
	"time"

	"golang.org/x/exp/rand"
	"gonum.org/v1/gonum/mat"
	"gonum.org/v1/gonum/stat/distmv"
	"gonum.org/v1/gonum/stat/distuv"
)

const Layout = "2006-01-02"

func helpers(stocks []string, sampleFixings map[string]float64, sampleModels map[string]mc.Model, sampleMu []float64, sampleCorr *mat.SymDense) ([]float64, mc.Basket, *distmv.Normal, distuv.Normal, error) {
	var pxRatio []float64
	currentPx, err := data.LatestPx(stocks)
	if err != nil {
		return nil, nil, nil, distuv.Normal{}, err
	}
	for _, v := range stocks {
		pxRatio = append(pxRatio, currentPx[v]/sampleFixings[v])
	}

	bsk, err := mc.NewBasket(sampleModels)
	if err != nil {
		return nil, nil, nil, distuv.Normal{}, err
	}

	dz1, dz2, err := distributions(sampleMu, sampleCorr)
	if err != nil {
		return nil, nil, nil, distuv.Normal{}, err
	}
	return pxRatio, bsk, dz1, dz2, nil
}

func Pricer(stocks []string, k, cpn, barCpn, fixCpn, ko, ki, kc float64, T, freq int, isEuro bool, sampleFixings map[string]float64, sampleModels map[string]mc.Model, sampleMu []float64, sampleCorr *mat.SymDense) (float64, error) {
	pxRatio, bsk, dz1, dz2, err := helpers(stocks, sampleFixings, sampleModels, sampleMu, sampleCorr)
	if err != nil {
		return math.NaN(), err
	}

	tNow, _ := time.Parse(Layout, time.Now().Format(Layout))
	dates, err := utils.GenerateDates(tNow, T, freq)
	if err != nil {
		return math.NaN(), err
	}
	
	fcn, err := payoff.NewFCN(stocks, k, cpn, barCpn, fixCpn, ko, ki, kc, T, freq, isEuro, dates)
	if err != nil {
		return math.NaN(), err
	}

	nsamples := 10000
	out := 0.0
	ch := make(chan float64, nsamples)
	errCh := make(chan error, nsamples)
	defer close(ch)
	defer close(errCh)

	// Compute path payouts concurrently
	for l := 0; l < nsamples; l++ {
		go func(ch chan float64, errCh chan error) {
			path, err := bsk.Path(dates["mcdates"], pxRatio, dz1, dz2)
			if err != nil {
				ch <- math.NaN()
				errCh <- err
				return
			}
			wop := wop(sampleFixings, dates, path)
			x := fcn.Payout(wop)
			ch <- x
			errCh <- nil
		}(ch, errCh)
	}

	for l := 0; l < nsamples; l++ {
		err := <-errCh
		if err != nil {
			return math.NaN(), err
		}
		out += <-ch
	}

	price := out / float64(nsamples)
	return price, nil
}

func distributions(sampleMu []float64, sampleCorr *mat.SymDense) (*distmv.Normal, distuv.Normal, error) {
	dz1, ok := distmv.NewNormal(sampleMu, sampleCorr, rand.NewSource(uint64(time.Now().UnixNano())))
	if !ok {
		return nil, distuv.Normal{}, errors.New("invalid corr matrix")
	}

	dz2 := distuv.Normal{Mu: 0.0, Sigma: 1.0, Src: rand.NewSource(uint64(time.Now().UnixNano()))}

	return dz1, dz2, nil
}

func wop(px map[string]float64, dates map[string][]time.Time, path mc.MCPath) []float64 {
	// get the worst of performance array
	var wop []float64
	p := make([]float64, len(px))
	for i := range dates["mcdates"] {
		j := 0
		for k := range px {
			p[j] = path[k][i]
			j++
		}
		minP := utils.MinSlice(p)
		wop = append(wop, minP)
	}
	return wop
}
