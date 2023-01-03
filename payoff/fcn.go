package payoff

import (
	"encoding/json"
	"math"
	"os"
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
	KC            float64
	Maturity      int
	CallFreq      int
	IsEuroKI      bool
	ObsDates      []time.Time
	KODates       []time.Time
	KIDates       []time.Time
}

type FCNOutput struct {
	StrikeDate    string             `json:"strike_date"`
	Tickers       []string           `json:"underlying_tickers"`
	Strike        float64            `json:"strike"`
	Maturity      int                `json:"maturity"`
	CallFreq      int                `json:"autocall_frequency"`
	IsEuro        bool               `json:"is_euro"`
	KO            float64            `json:"knock_out_barrier"`
	KI            float64            `json:"knock_in_barrier"`
	KC            float64            `json:"coupon_barrier"`
	AutoCoupon    float64            `json:"autocall_coupon_rates"`
	BarrierCoupon float64            `json:"barrier_coupon_rates"`
	FixedCoupon   float64            `json:"fixed_coupon_rates"`
	Price         float64            `json:"price"`
	FixedPrice    map[string]float64 `json:"fixings"`
}

const Layout = "2006-01-02"

func NewFCN(stocks []string, k, cpn, barCpn, fixCpn, ko, ki, kc float64, T, freq int, isEuro bool, m map[string][]time.Time) *FCN {
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
		KC:            kc,
		Maturity:      T,
		CallFreq:      freq,
		IsEuroKI:      isEuro,
		ObsDates:      m["mcdates"],
		KODates:       m["kodates"],
		KIDates:       kidates,
	}
	return &f
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
			// fmt.Printf("At %v\n", t.Format(Layout))
			out += factor * f.FixedCoupon
			// fmt.Printf("fixed coupon: %0.9f\n", factor*f.FixedCoupon)
			if path[i] > f.KC {
				out += factor * f.BarrierCoupon
				// fmt.Printf("barrier coupon: %0.9f\n", factor*f.BarrierCoupon)
			}
			if path[i] > f.KO {
				out += float64(count+1) * factor * f.Coupon
				// fmt.Printf("knock-out coupon: %0.9f\n", float64(count+1)*factor*f.Coupon)
				dt := float64(t.Unix()-f.ObsDates[0].Unix()) / float64(60*60*24*365)
				// dt := t.Sub(f.ObsDates[0]).Seconds()
				return math.Exp(-0.03*dt) * out
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
		// fmt.Printf("knock-in: %0.9f\n", -1.0/f.Strike*math.Max(f.Strike-path[T], 0))
	}
	dt := float64(f.ObsDates[T].Unix()-f.ObsDates[0].Unix()) / float64(60*60*24*365)
	return math.Exp(-0.03*dt) * out
}

func (f *FCN) Save(filename string, price float64, spotref map[string]float64) error {
	output := FCNOutput{
		StrikeDate:    time.Now().Format(Layout),
		Tickers:       f.Tickers,
		Strike:        f.Strike,
		Maturity:      f.Maturity,
		CallFreq:      f.CallFreq,
		IsEuro:        f.IsEuroKI,
		KO:            f.KO,
		KI:            f.KI,
		KC:            f.KC,
		AutoCoupon:    f.Coupon,
		BarrierCoupon: f.BarrierCoupon,
		FixedCoupon:   f.FixedCoupon,
		Price:         price,
		FixedPrice:    spotref,
	}

	data, err := json.MarshalIndent(output, "", " ")
	if err != nil {
		return err
	}
	err = os.WriteFile(filename, data, 0644)
	if err != nil {
		return err
	}
	return nil
}
