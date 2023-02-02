package api

import (
	"database/sql"
	"errors"
	"fmt"
	"math"
	"net/http"
	"sort"
	"sync"
	"time"

	db "github.com/banachtech/spotted-zebra/db/sqlc"
	"github.com/banachtech/spotted-zebra/mc"
	"github.com/banachtech/spotted-zebra/payoff"
	"github.com/banachtech/spotted-zebra/util"
	"github.com/gin-gonic/gin"
	"golang.org/x/exp/rand"
	"golang.org/x/time/rate"
	"gonum.org/v1/gonum/mat"
	"gonum.org/v1/gonum/stat/distmv"
	"gonum.org/v1/gonum/stat/distuv"
)

type pricerRequest struct {
	Stocks     []string `json:"stocks" binding:"required"`
	Strike     float64  `json:"strike" binding:"required"`
	Cpn        float64  `json:"autocall_coupon_rate"`
	BarrierCpn float64  `json:"barrier_coupon_rate"`
	FixCpn     float64  `json:"fixed_coupon_rate"`
	KO         float64  `json:"knock_out_barrier"`
	KI         float64  `json:"knock_in_barrier"`
	KC         float64  `json:"coupon_barrier"`
	Maturity   int      `json:"maturity" binding:"required,min=1"`
	Freq       int      `json:"frequency" binding:"required,min=1"`
	IsEuro     bool     `json:"isEuro"`
}

const Layout = "2006-01-02"

var Pricerlimiters = make(map[string]*rate.Limiter)

func getPricerLimiter(userID string) *rate.Limiter {
	limiter, ok := Pricerlimiters[userID]
	if !ok {
		// Create a new rate limiter for the user if it doesn't exist
		limiter = rate.NewLimiter(rate.Every(time.Second), 5) // 5 requests per second
		Pricerlimiters[userID] = limiter
	}
	return limiter
}

func (server *Server) pricer(c *gin.Context) {
	var req pricerRequest

	prefix, exists := c.Get("prefix")
	if !exists {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"status": http.StatusBadRequest, "msg": "Authentication Error"})
		return
	}

	limiter := getPricerLimiter(prefix.(string))

	// Check if the user has exceeded the rate limit
	if !limiter.Allow() {
		c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"status": http.StatusTooManyRequests, "msg": "Too Many Requests"})
		return
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, errorResponse(err))
		return
	}
	if len(req.Stocks) == 0 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"status": http.StatusBadRequest, "msg": "Error JSON binding, please check your JSON input"})
		return
	}
	if req.Maturity < req.Freq {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"status": http.StatusBadRequest, "msg": "Maturity cannot be less than frequency"})
		return
	}
	sort.Strings(DefaultStocks)

	filterStocks, err := util.Filter(req.Stocks, DefaultStocks)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"status": http.StatusBadRequest, "msg": fmt.Sprintf("Failed filter stocks: %v", err)})
		return
	}
	req.Stocks = filterStocks

	result, err := server.store.GetValues(c)
	if err != nil {
		if err == sql.ErrNoRows {
			c.AbortWithStatusJSON(http.StatusNotFound, errorResponse(err))
			return
		}

		c.AbortWithStatusJSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	models, fixings, means, px, corrMatrix := constructor(result, filterStocks)

	p, err := fcnPricer(filterStocks, req, fixings, means, px, models, corrMatrix)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"status": http.StatusBadRequest, "msg": fmt.Sprintf("Failed compute FCN price: %s", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"price": p})
}

func constructor(target db.GetValuesResult, filterStocks []string) (map[string]mc.Model, map[string]float64, map[string]float64, map[string]float64, *mat.SymDense) {
	params := target.Params
	stats := target.Stats
	corr := target.Corrpair
	latestpx := target.LatestPrice

	models := map[string]mc.Model{}
	for i := range params {
		models[params[i].Ticker] = mc.HypHyp{Sigma: params[i].Sigma, Alpha: params[i].Alpha, Beta: params[i].Beta, Kappa: params[i].Kappa, Rho: params[i].Rho}
	}

	fixings := map[string]float64{}
	means := map[string]float64{}
	for i := range stats {
		fixings[stats[i].Ticker] = stats[i].Fixing
		means[stats[i].Ticker] = stats[i].Mean
	}

	px := map[string]float64{}
	for i := range latestpx {
		px[latestpx[i].Ticker] = latestpx[i].Fixing
	}

	corrpair := map[string]map[string]float64{}
	for i := range corr {
		_, ok := corrpair[corr[i].X0]
		if !ok {
			corrpair[corr[i].X0] = map[string]float64{}
		}
		corrpair[corr[i].X0][corr[i].X1] = corr[i].Corr
	}

	sampleModels := map[string]mc.Model{}
	sampleFixings := map[string]float64{}
	sampleMeans := map[string]float64{}
	samplePx := map[string]float64{}
	var corrs []float64

	for i := range filterStocks {
		sampleModels[filterStocks[i]] = models[filterStocks[i]]
		sampleFixings[filterStocks[i]] = fixings[filterStocks[i]]
		sampleMeans[filterStocks[i]] = means[filterStocks[i]]
		samplePx[filterStocks[i]] = px[filterStocks[i]]
		for j := range filterStocks {
			if i == j {
				corrs = append(corrs, 1.0)
			} else if i < j {
				corrs = append(corrs, corrpair[filterStocks[i]][filterStocks[j]])
			} else {
				corrs = append(corrs, corrpair[filterStocks[j]][filterStocks[i]])
			}
		}
	}

	sampleCorr := mat.NewSymDense(len(filterStocks), corrs)
	return sampleModels, sampleFixings, sampleMeans, samplePx, sampleCorr
}

func fcnPricer(stocks []string, arg pricerRequest, fixings, means, px map[string]float64, models map[string]mc.Model, corrMatrix *mat.SymDense) (float64, error) {
	var wg sync.WaitGroup
	pxRatio := map[string]float64{}
	var mu []float64
	for _, v := range stocks {
		pxRatio[v] = px[v] / fixings[v]
		mu = append(mu, means[v])
	}

	bsk := mc.NewBasket(models)

	dz1, dz2, err := distributions(mu, corrMatrix)
	if err != nil {
		return math.NaN(), err
	}

	tNow, _ := time.Parse(Layout, time.Now().Format(Layout))
	dates, err := util.GenerateDates(tNow, arg.Maturity, arg.Freq)
	if err != nil {
		return math.NaN(), err
	}

	nsamples := 10000
	n_sims := len(dates["mcdates"]) - 1
	z1 := map[int]map[string][]float64{}
	z2 := map[int]map[string][]float64{}

	for l := 0; l < nsamples; l++ {
		z1[l] = map[string][]float64{}
		z2[l] = map[string][]float64{}
		for _, v := range stocks {
			z1[l][v] = make([]float64, n_sims)
			z2[l][v] = make([]float64, n_sims)
		}
		// r := make([]float64, len(stocks))
		for k := 0; k < n_sims; k++ {
			r := dz1.Rand(nil)
			for i := range stocks {
				z1[l][stocks[i]][k] = r[i]
				z2[l][stocks[i]][k] = dz2.Rand()
			}
		}
	}

	fcn := payoff.NewFCN(stocks, arg.Strike, arg.Cpn, arg.BarrierCpn, arg.FixCpn, arg.KO, arg.KI, arg.KC, arg.Maturity, arg.Freq, arg.IsEuro, dates)

	out := 0.0
	ch := make(chan float64, nsamples)
	// defer close(ch)

	// Compute path payouts concurrently
	for l := 0; l < nsamples; l++ {
		wg.Add(1)
		go func(l int) {
			defer wg.Done()
			path := bsk.Path(stocks, dates["mcdates"], pxRatio, z1[l], z2[l])
			wop := wop(fixings, dates, path)
			x := fcn.Payout(wop)
			ch <- x
		}(l)
	}

	wg.Wait()
	close(ch)

	for l := 0; l < nsamples; l++ {
		out += <-ch
	}

	price := out / float64(nsamples)
	return price, nil
}

func distributions(sampleMu []float64, sampleCorr *mat.SymDense) (*distmv.Normal, distuv.Normal, error) {
	dz1, ok := distmv.NewNormal(sampleMu, sampleCorr, rand.NewSource(uint64(time.Now().UnixNano())))
	if !ok {
		return nil, distuv.Normal{}, errors.New("failed generate distribution")
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
		minP := util.MinSlice(p)
		wop = append(wop, minP)
	}
	return wop
}
