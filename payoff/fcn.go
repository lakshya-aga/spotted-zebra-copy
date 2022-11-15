package payoff

import (
	"main/mc"
	"math"
	"time"

	"main/utils"
)

type FCN struct {
	Tickers       []string
	Strike        float64
	Coupon        float64
	BarrierCoupon float64
	FixedCoupon   float64
	KO            float64
	KI            float64
	Maturity      int
	CallFreq      int
	IsEuroKI      bool
	ObsDates      []time.Time
	KODates       []time.Time
	KIDates       []time.Time
}

const Layout = "2006-01-02"

func NewFCN(stocks []string, k, cpn, barCpn, fixCpn, ko, ki float64, T, freq int, isEuro bool, m map[string][]time.Time) (*FCN, error) {
	var kidates []time.Time
	if isEuro {
		kidates = []time.Time{m["mcdates"][len(m["mcdates"])-1]}
	} else {
		kidates = m["mcdates"]
	}
	f := FCN{
		Tickers:       stocks,
		Strike:        k,
		Coupon:        cpn,
		BarrierCoupon: barCpn,
		FixedCoupon:   fixCpn,
		KO:            ko,
		KI:            ki,
		Maturity:      T,
		CallFreq:      freq,
		IsEuroKI:      isEuro,
		ObsDates:      m["mcdates"],
		KODates:       m["kodates"],
		KIDates:       kidates,
	}
	return &f, nil
}

func (f *FCN) Payout(path mc.MCPath, spotref map[string]float64) float64 {
	// p holds prices at any obs t
	p := make([]float64, len(spotref))
	var j, count int
	var minP float64

	out := 1.0

	// Initialise KI flag
	isKI := false
	for i, t := range f.ObsDates {
		factor := float64(count+1) / 12.0
		// Populate slice of prices
		j = 0
		for k, v := range spotref {
			// Scale MC path values by spot reference
			p[j] = v * path[k][i]
			j++
		}
		// Compute worst of performance
		minP = utils.MinSlice(p)
		// Check for KO and redeem if required
		// Coupon dates coincide with KO dates, so pay coupon if required
		if t.Equal(f.KODates[count]) {
			out += factor * f.FixedCoupon
			if minP > f.BarrierCoupon {
				out += factor * f.BarrierCoupon
			}
			if minP > f.KO {
				out += factor * f.Coupon
				return out
			}
			count++
		}

		if !f.IsEuroKI && !isKI {
			if minP < f.KI {
				isKI = true
			}
		}
	}
	if isKI || (f.IsEuroKI && minP < f.KI) {
		out = -1.0 / f.Strike * math.Max(f.Strike-minP, 0)
	}
	return out
}
