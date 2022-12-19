package handlers

import (
	"fmt"
	"net/http"
	"time"

	"main/db"
	"main/utils"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type UserDetails struct {
	Email string `json:"email"`
}

const Layout = "2006-01-02 15:04:05"

func Registration(c *gin.Context) {
	var details UserDetails
	if err := c.BindJSON(&details); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"status": http.StatusBadRequest, "msg": "Error JSON binding, please check your JSON input"})
		return
	}
	if details.Email == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"status": http.StatusBadRequest, "msg": "Please enter a valid email"})
		return
	}

	// connect to database
	db, err := db.ConnectDB("localhost", 5432, "postgres", "Louis123", "auth")
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"status": http.StatusBadRequest, "msg": fmt.Sprintf("Failed connect to database: %s", err)})
		return
	}
	defer db.Close()

	prefix, token, err := utils.GenerateToken()
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"status": http.StatusBadRequest, "msg": fmt.Sprintf("Failed generate api key: %s", err)})
		return
	}
	apiKey := fmt.Sprintf("%s.%s", prefix, token)
	hashedApi, _ := bcrypt.GenerateFromPassword([]byte(apiKey), 14)

	now := time.Now()
	exp := now.AddDate(0, 6, 0)

	insertDetails := `insert into registrar ("email_address", "prefix", "token", "generated_at", "expired_at", "admin") values ($1, $2, $3, $4, $5, $6)`
	_, err = db.Exec(insertDetails, details.Email, prefix, hashedApi, now.Format(Layout), exp.Format(Layout), false)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"status": http.StatusBadRequest, "msg": fmt.Sprintf("Failed insert into database: %s", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":         http.StatusOK,
		"hashedPassword": hashedApi,
		"email":          details.Email,
		"api_key":        apiKey,
		"msg":            "insert into database successfully",
	})
}
