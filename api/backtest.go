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
	"golang.org/x/time/rate"
	"gonum.org/v1/gonum/mat"
	"gonum.org/v1/gonum/stat"
)

// // Limiter is a global rate limiter for all users
// var BacktestLimiter = rate.NewLimiter(5, 1) // 5 requests per second

var Backtestlimiters = make(map[string]*rate.Limiter)

func getBacktestLimiter(userID string) *rate.Limiter {
	limiter, ok := Pricerlimiters[userID]
	if !ok {
		// Create a new rate limiter for the user if it doesn't exist
		limiter = rate.NewLimiter(rate.Every(time.Second), 2) // 5 requests per 10 seconds
		Pricerlimiters[userID] = limiter
	}
	return limiter
}

func (server *Server) backtest(c *gin.Context) {
	var req pricerRequest
	var wg sync.WaitGroup

	prefix, exists := c.Get("prefix")
	if !exists {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"status": http.StatusBadRequest, "msg": "Authentication Error"})
		return
	}

	limiter := getBacktestLimiter(prefix.(string))

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

	result, err := server.store.GetBacktestValues(c)
	if err != nil {
		if err == sql.ErrNoRows {
			c.AbortWithStatusJSON(http.StatusNotFound, errorResponse(err))
			return
		}

		c.AbortWithStatusJSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	dates, models, fixings, means, corrMatrix := backtestConstructor(result, filterStocks)

	// prices := map[string]float64{}
	errCh := make(chan error, len(dates))
	pnlCh := make(chan float64, len(dates))
	dateCh := make(chan string, len(dates))
	// defer close(dateCh)
	// defer close(errCh)
	// defer close(pnlCh)
	var profit []float64
	for t := range dates {
		wg.Add(1)
		go func(t int) {
			defer wg.Done()
			p, err := fcnPricer(filterStocks, req, fixings[dates[t]], means[dates[t]], fixings[dates[t]], models[dates[t]], corrMatrix[dates[t]])
			if err != nil {
				errCh <- err
				pnlCh <- math.NaN()
				dateCh <- dates[t]
				return
			}

			// prices[dates[t]] = p

			payout, err := fcnPayout(dates[t], filterStocks, req, fixings[dates[t]], means[dates[t]], fixings[dates[t]], models[dates[t]], corrMatrix[dates[t]])
			if err != nil {
				errCh <- err
				pnlCh <- math.NaN()
				dateCh <- dates[t]
				return
			}

			pnl := payout - p
			if math.IsNaN(pnl) {
				errCh <- errors.New("return is NaN")
				pnlCh <- math.NaN()
				dateCh <- dates[t]
				return
			}
			errCh <- nil
			pnlCh <- pnl
			dateCh <- dates[t]
		}(t)
	}

	wg.Wait()
	close(dateCh)
	close(errCh)
	close(pnlCh)

	rollout := map[string]float64{}
	for range dates {
		err := <-errCh
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"status": http.StatusBadRequest, "msg": fmt.Sprintf("Failed compute FCN payout: %s", err)})
			return
		}
		pnl := <-pnlCh
		if !math.IsNaN(pnl) {
			profit = append(profit, pnl)
		}
		date := <-dateCh
		rollout[date] = pnl
	}

	var sortedRet []float64
	for t := range dates {
		sortedRet = append(sortedRet, rollout[dates[t]])
	}

	maxDrawDown := maxDrawDown(sortedRet)

	mean, std := stat.MeanStdDev(profit, nil)
	min, max := minmax(profit)

	c.JSON(http.StatusOK, gin.H{"mean": mean, "std": std, "min": min, "max": max, "max_drawdown": maxDrawDown})
}

func backtestConstructor(target db.GetBacktestValuesResult, filterStocks []string) ([]string, map[string]map[string]mc.Model, map[string]map[string]float64, map[string]map[string]float64, map[string]*mat.SymDense) {
	params := target.Params
	stats := target.Stats
	corr := target.Corrpair
	dates := target.Date

	models := map[string]map[string]mc.Model{}
	for i := range params {
		_, ok := models[params[i].Ticker]
		if !ok {
			models[params[i].Ticker] = map[string]mc.Model{}
		}
		models[params[i].Ticker][params[i].Date] = mc.HypHyp{Sigma: params[i].Sigma, Alpha: params[i].Alpha, Beta: params[i].Beta, Kappa: params[i].Kappa, Rho: params[i].Rho}
	}

	fixings := map[string]map[string]float64{}
	means := map[string]map[string]float64{}
	for i := range stats {
		_, ok1 := fixings[stats[i].Ticker]
		if !ok1 {
			fixings[stats[i].Ticker] = map[string]float64{}
		}
		_, ok2 := means[stats[i].Ticker]
		if !ok2 {
			means[stats[i].Ticker] = map[string]float64{}
		}
		fixings[stats[i].Ticker][stats[i].Date] = stats[i].Fixing
		means[stats[i].Ticker][stats[i].Date] = stats[i].Mean
	}

	corrpair := map[string]map[string]map[string]float64{}
	for i := range corr {
		_, ok1 := corrpair[corr[i].X0]
		if !ok1 {
			corrpair[corr[i].X0] = map[string]map[string]float64{}
		}
		_, ok2 := corrpair[corr[i].X0][corr[i].Date]
		if !ok2 {
			corrpair[corr[i].X0][corr[i].Date] = map[string]float64{}
		}
		corrpair[corr[i].X0][corr[i].Date][corr[i].X1] = corr[i].Corr
	}
	sampleModels := map[string]map[string]mc.Model{}
	sampleFixings := map[string]map[string]float64{}
	sampleMeans := map[string]map[string]float64{}
	corrs := map[string][]float64{}
	sampleCorr := map[string]*mat.SymDense{}
	for t := range dates {
		sampleModels[dates[t]] = map[string]mc.Model{}
		sampleFixings[dates[t]] = map[string]float64{}
		sampleMeans[dates[t]] = map[string]float64{}
		for i := range filterStocks {
			sampleModels[dates[t]][filterStocks[i]] = models[filterStocks[i]][dates[t]]
			sampleFixings[dates[t]][filterStocks[i]] = fixings[filterStocks[i]][dates[t]]
			sampleMeans[dates[t]][filterStocks[i]] = means[filterStocks[i]][dates[t]]
			for j := range filterStocks {
				if i == j {
					corrs[dates[t]] = append(corrs[dates[t]], 1.0)
				} else if i < j {
					corrs[dates[t]] = append(corrs[dates[t]], corrpair[filterStocks[i]][dates[t]][filterStocks[j]])
				} else {
					corrs[dates[t]] = append(corrs[dates[t]], corrpair[filterStocks[j]][dates[t]][filterStocks[i]])
				}
			}
		}
	}

	for k, v := range corrs {
		sampleCorr[k] = mat.NewSymDense(len(filterStocks), v)
	}

	return dates, sampleModels, sampleFixings, sampleMeans, sampleCorr
}

func fcnPayout(date string, stocks []string, arg pricerRequest, fixings, means, px map[string]float64, models map[string]mc.Model, corrMatrix *mat.SymDense) (float64, error) {
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

	tNow, _ := time.Parse(Layout, date)
	dates, err := util.GenerateDates(tNow, arg.Maturity, arg.Freq)
	if err != nil {
		return math.NaN(), err
	}

	n_sims := len(dates["mcdates"]) - 1
	z1 := map[string][]float64{}
	z2 := map[string][]float64{}

	for _, v := range stocks {
		z1[v] = make([]float64, n_sims)
		z2[v] = make([]float64, n_sims)
	}

	for k := 0; k < n_sims; k++ {
		r := dz1.Rand(nil)
		for i := range stocks {
			z1[stocks[i]][k] = r[i]
			z2[stocks[i]][k] = dz2.Rand()
		}
	}

	fcn := payoff.NewFCN(stocks, arg.Strike, arg.Cpn, arg.BarrierCpn, arg.FixCpn, arg.KO, arg.KI, arg.KC, arg.Maturity, arg.Freq, arg.IsEuro, dates)
	path := bsk.Path(stocks, dates["mcdates"], pxRatio, z1, z2)
	wop := wop(fixings, dates, path)
	x := fcn.Payout(wop)
	return x, nil
}

func minmax(array []float64) (float64, float64) {
	max := array[0]
	min := array[0]
	for _, value := range array {
		if max < value {
			max = value
		}
		if min > value {
			min = value
		}
	}
	return min, max
}

func maxDrawDown(array []float64) float64 {
	var rollout []float64
	x := make([]float64, len(array)+1)
	x[0] = 1.0
	for i := 0; i < len(array); i++ {
		x[i+1] = x[i] * (1.0 + array[i])
	}
	for i := 1; i < len(x); i++ {
		xt := x[i]
		xPrev := x[:i]
		_, maxPrevX := minmax(xPrev)
		r := xt/maxPrevX - 1
		rollout = append(rollout, r)
	}
	maxR, _ := minmax(rollout)
	return maxR
}
