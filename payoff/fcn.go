package payoff

import (
	"fmt"
	"math"
	"time"
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

func (f *FCN) Payout(path []float64) float64 {
	var count int
	out := 1.0
	T := len(path) - 1

	// Initialise KI flag
	isKI := false
	for i, t := range f.ObsDates {
		factor := 1 / 12.0
		// Check for KO and redeem if required
		// Coupon dates coincide with KO dates, so pay coupon if required
		if t.Equal(f.KODates[count]) {
			out += factor * f.FixedCoupon
			fmt.Printf("fixed coupon: %0.9f\n", factor*f.FixedCoupon)
			if path[i] > f.BarrierCoupon {
				out += factor * f.BarrierCoupon
				fmt.Printf("barrier coupon: %0.9f\n", factor*f.BarrierCoupon)
			}
			if path[i] > f.KO {
				out += float64(count+1) * factor * f.Coupon
				fmt.Printf("knock-out coupon: %0.9f\n", float64(count+1)*factor*f.Coupon)
				return out
			}
			count++
		}

		if !f.IsEuroKI && !isKI {
			if path[i] < f.KI {
				isKI = true
			}
		}
	}
	if isKI || (f.IsEuroKI && path[T] < f.KI) {
		out += (-1.0 / f.Strike) * math.Max(f.Strike-path[T], 0)
		fmt.Printf("knock-in: %0.9f\n", -1.0/f.Strike*math.Max(f.Strike-path[T], 0))
	}
	return out
}
