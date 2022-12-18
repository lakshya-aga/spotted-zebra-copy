package handlers

import (
	"fmt"
	"net/http"
	"sort"
	"time"

	"main/data"
	"main/db"
	"github.com/gin-gonic/gin"
)

var DefaultStocks = []string{"AAPL", "AMZN", "META", "MSFT", "TSLA", "GOOG", "NVDA", "AVGO", "QCOM", "INTC"}

func DailyUpdates(c *gin.Context) {
	// connect to database
	db, err := db.ConnectDB("localhost", 5432, "postgres", "Louis123", "testdb")
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"status": http.StatusBadRequest, "msg": fmt.Sprintf("Failed connect to database: %s", err)})
		return
	}
	defer db.Close()

	sort.Strings(DefaultStocks)
	_, _, _, _, err = data.Initialize(DefaultStocks, db)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"status": http.StatusBadRequest, "msg": fmt.Sprintf("Failed update model parameters: %s", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": http.StatusOK,
		"msg":    fmt.Sprintf("Model Parameters updated at %s", time.Now().Format(Layout)),
	})
}
