package payoff

import (
	"testing"
	"time"

	"github.com/banachtech/spotted-zebra/util"
	"github.com/stretchr/testify/require"
)

type pricerRequest struct {
	Stocks     []string
	Strike     float64
	Cpn        float64
	BarrierCpn float64
	FixCpn     float64
	KO         float64
	KI         float64
	KC         float64
	Maturity   int
	Freq       int
	IsEuro     bool
}

func TestNewFCN(t *testing.T) {
	arg1 := pricerRequest{
		Stocks:     []string{"AAPL", "AVGO", "TSLA"},
		Strike:     0.80,
		Cpn:        0.50,
		BarrierCpn: 0.50,
		FixCpn:     0.50,
		KO:         1.05,
		KI:         0.70,
		KC:         0.80,
		Maturity:   3,
		Freq:       1,
		IsEuro:     true,
	}
	arg2 := pricerRequest{
		Stocks:     []string{"AAPL", "AVGO", "TSLA"},
		Strike:     0.80,
		Cpn:        0.50,
		BarrierCpn: 0.50,
		FixCpn:     0.50,
		KO:         1.05,
		KI:         0.70,
		KC:         0.80,
		Maturity:   3,
		Freq:       1,
		IsEuro:     false,
	}

	tNow, _ := time.Parse(Layout, "2023-01-17")

	type testCases struct {
		name string
		arg  pricerRequest
	}

	for _, test := range []testCases{
		{
			name: "IS_EURO",
			arg:  arg1,
		},
		{
			name: "NOT_EURO",
			arg:  arg2,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			dates, err := util.GenerateDates(tNow, test.arg.Maturity, test.arg.Freq)
			require.NoError(t, err)
			fcn := NewFCN(test.arg.Stocks, test.arg.Strike, test.arg.Cpn, test.arg.BarrierCpn, test.arg.FixCpn, test.arg.KO, test.arg.KI, test.arg.KC, test.arg.Maturity, test.arg.Freq, test.arg.IsEuro, dates)
			require.NotEmpty(t, fcn)
		})
	}
}