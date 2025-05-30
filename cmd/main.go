package main

import (
	"database/sql"
	"golang-order-matching/api"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
)

func main() {
	// Connect to MySQL
	db, err := sql.Open("mysql", "root:03111985@tcp(127.0.0.1:3306)/order_matching")
	if err != nil {
		log.Fatal("Error opening DB:", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal("DB unreachable:", err)
	}
	log.Println("Connected to MySQL!")

	r := gin.Default()

	// Middleware to attach DB to all requests
	r.Use(func(c *gin.Context) {
		c.Set("db", db)
		c.Next()
	})

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	//  Register order endpoint
	r.POST("/orders", api.PlaceOrder)
    r.GET("/orderbook", api.GetOrderBook)

	r.Run(":3000") // Start server on port 3000
}
