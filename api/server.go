package api

import (
	db "github.com/banachtech/spotted-zebra/db/sqlc"
	"github.com/gin-gonic/gin"
)

var DefaultStocks = []string{"AAPL", "AMZN", "META", "MSFT", "TSLA", "GOOG", "NVDA", "AVGO", "QCOM", "INTC"}

// Server serves HTTP requests for our fcn pricer service.
type Server struct {
	store  db.Store
	router *gin.Engine
}

// NewServer creates a new HTTP server and set up routing.
func NewServer(store db.Store) *Server {
	server := &Server{store: store}

	server.setupRouter()
	return server
}

func (server *Server) setupRouter() {
	router := gin.Default()

	authRoutes := router.Group("/v1").Use(server.Authentication)
	authRoutes.POST("/pricer", server.pricer)
	authRoutes.POST("/backtest", server.backtest)
	server.router = router
}

// Start runs the HTTP server on a specific address.
func (server *Server) Start(address string) error {
	return server.router.Run(address)
}

func errorResponse(err error) gin.H {
	return gin.H{"error": err.Error()}
}
