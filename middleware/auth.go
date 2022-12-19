package middleware

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"

	"main/db"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

const (
	Layout = "2006-01-02 15:04:05"
	DemoKey = "9wz024zA.UxRLCbj9V3xeSv_W"
)


// AuthMiddleware creates a gin middleware for authorization
func Authentication(c *gin.Context) {
	authorizationHeader := c.GetHeader("Authorization")

	if len(authorizationHeader) == 0 {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"status": http.StatusUnauthorized, "msg": "Authorization header is not provided"})
		return
	}

	fields := strings.Fields(authorizationHeader)
	if len(fields) < 2 {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"status": http.StatusUnauthorized, "msg": "Invalid authorization header format"})
		return
	}

	authorizationType := strings.ToLower(fields[0])
	if authorizationType != "bearer" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"status": http.StatusUnauthorized, "msg": fmt.Sprintf("unsupported authorization type %s", authorizationType)})
		return
	}

	apiKey := fields[1]
	prefix := strings.Split(apiKey, ".")[0]
	if len(prefix) != 8 {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"status": http.StatusUnauthorized, "msg": "Please input a valid API Key"})
		return
	}

	// connect to database
	db, err := db.ConnectDB("localhost", 5432, "postgres", "Louis123", "auth")
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"status": http.StatusBadRequest, "msg": fmt.Sprintf("Failed connect to database: %s", err)})
		return
	}
	defer db.Close()

	var token, expiredDate string

	err = db.QueryRow(`SELECT "token", "expired_at" FROM registrar WHERE "prefix" = $1`, prefix).Scan(&token, &expiredDate)
	if err != nil {
		if err == sql.ErrNoRows {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"status": http.StatusUnauthorized, "msg": "Please input a valid API Key"})
			return
		} else {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"status": http.StatusBadRequest, "msg": fmt.Sprintf("Failed searched in database: %s", err)})
			return
		}
	}

	expired, _ := time.Parse(Layout, expiredDate)
	now, _ := time.Parse(Layout, time.Now().Format(Layout))
	if now.After(expired) {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"status": http.StatusBadRequest, "msg": "API Key is expired"})
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(token), []byte(apiKey))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"status": http.StatusUnauthorized, "msg": "Please input a valid API Key"})
		return
	}

	c.Next()
}
