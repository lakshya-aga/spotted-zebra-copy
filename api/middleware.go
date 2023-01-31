package api

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

const (
	authorizationHeaderKey  = "authorization"
	authorizationTypeBearer = "bearer"
	Layout2                 = "2006-01-02 15:04:05"
)

// AuthMiddleware creates a gin middleware for authorization
func (server *Server) authentication(c *gin.Context) {
	authorizationHeader := c.GetHeader(authorizationHeaderKey)

	if len(authorizationHeader) == 0 {
		c.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse(errors.New("authorization header is not provided")))
		return
	}

	fields := strings.Fields(authorizationHeader)
	if len(fields) < 2 {
		c.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse(errors.New("invalid authorization header format")))
		return
	}

	authorizationType := strings.ToLower(fields[0])
	if authorizationType != authorizationTypeBearer {
		c.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse(fmt.Errorf("unsupported authorization type: %s", authorizationType)))
		return
	}

	apiKey := fields[1]

	prefix := strings.Split(apiKey, ".")[0]
	if len(prefix) != 8 {
		c.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse(errors.New("please input a valid API Key")))
		return
	}

	user, err := server.store.GetUser(c, prefix)
	if err != nil {
		if err == sql.ErrNoRows {
			c.AbortWithStatusJSON(http.StatusNotFound, errorResponse(err))
			return
		}

		c.AbortWithStatusJSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	expired, _ := time.Parse(Layout2, user.ExpiredAt)
	now, _ := time.Parse(Layout2, time.Now().Format(Layout2))
	if now.After(expired) {
		c.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse(errors.New("api key is expired")))
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Token), []byte(apiKey))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse(errors.New("please input a valid API Key")))
		return
	}
	
	c.Next()
}
