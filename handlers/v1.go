package handlers

import (
	"fmt"
	"net/http"
	"sort"

	"main/data"
	"main/db"
	"main/mainfuncs"
	"github.com/gin-gonic/gin"
)

type PricerDetails struct {
	Stocks     []string `json:"stocks"`
	Strike     float64  `json:"strike"`
	Cpn        float64  `json:"autocall_coupon_rate"`
	BarrierCpn float64  `json:"barrier_coupon_rate"`
	FixCpn     float64  `json:"fixed_coupon_rate"`
	KO         float64  `json:"knock_out_barrier"`
	KI         float64  `json:"knock_in_barrier"`
	KC         float64  `json:"coupon_barrier"`
	Maturity   int      `json:"maturity"`
	Freq       int      `json:"frequency"`
	IsEuro     bool     `json:"isEuro"`
}

func Pricer(c *gin.Context) {
	var details PricerDetails
	if err := c.BindJSON(&details); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"status": http.StatusBadRequest, "msg": "Error JSON binding, please check your JSON input"})
		return
	}

	// connect to database
	db, err := db.ConnectDB("localhost", 5432, "postgres", "Louis123", "testdb")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	sort.Strings(DefaultStocks)
	allModels, allMeans, allCorr, allFixings, err := data.Initialize(DefaultStocks, db)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"status": http.StatusBadRequest, "msg": fmt.Sprintf("Failed update model parameters: %s", err)})
		return
	}

	filterStocks, sampleModels, sampleMu, sampleCorr, sampleFixings, err := data.Sampler(DefaultStocks, details.Stocks, allModels, allMeans, allCorr, allFixings)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"status": http.StatusBadRequest, "msg": fmt.Sprintf("Failed retrieve parameters: %s", err)})
		return
	}

	details.Stocks = filterStocks

	p, err := mainfuncs.Pricer(filterStocks, details.Strike, details.Cpn, details.BarrierCpn, details.FixCpn, details.KO, details.KI, details.KC, details.Maturity, details.Freq, details.IsEuro, sampleFixings, sampleModels, sampleMu, sampleCorr)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"status": http.StatusBadRequest, "msg": fmt.Sprintf("Failed compute FCN price: %s", err)})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": http.StatusOK, "msg": details, "price": p})
}

func Test(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": http.StatusOK, "msg": "HI"})
}
