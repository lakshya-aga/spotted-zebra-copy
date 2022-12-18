package main

import (
	"main/handlers"
	"main/mc"
	"main/middleware"

	"github.com/gin-gonic/gin"
)

type bskData struct {
	Stocks     []string             `json:"stocks"`
	Models     map[string]mc.HypHyp `json:"model_parameters"`
	CorrID     map[string]int       `json:"corr_id"`
	Mean       []float64            `json:"mean"`
	CorrMatrix []float64            `json:"corr_matrix"`
	SpotPrice  map[string]float64   `json:"spot_price"`
}

const Layout = "2006-01-02"

var DefaultStocks = []string{"AAPL", "AMZN", "META", "MSFT", "TSLA", "GOOG", "NVDA", "AVGO", "QCOM", "INTC"}

func main() {
	r := gin.Default()

	admin := r.Group("/admin")
	{
		admin.GET("/update", handlers.DailyUpdates)
	}

	registrar := r.Group("/register")
	{
		registrar.POST("/users", handlers.Registration)
	}

	v1 := r.Group("/v1")
	v1.Use(middleware.Authentication)
	{
		v1.POST("/pricer", handlers.Pricer)
		v1.POST("/hi", handlers.Test)
		// v1.POST("/auth", middleware.Authentication)
	}

	// Listen and serve on 0.0.0.0:8080
	r.Run(":8080")
}
