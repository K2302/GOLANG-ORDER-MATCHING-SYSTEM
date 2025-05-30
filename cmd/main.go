package main

import (
	"database/sql"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"

	"golang-order-matching/api"
	"golang-order-matching/engine"
	"golang-order-matching/models"
)

func main() {
	db, err := sql.Open("mysql", "root:your_password@tcp(127.0.0.1:3306)/order_matching")
	if err != nil {
		log.Fatal("Error opening DB:", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal("DB unreachable:", err)
	}
	log.Println("✅ Connected to MySQL!")

	// 🧠 Create matching engine
	matchEngine := engine.NewEngine()

	rows, err := db.Query(`
	SELECT id, symbol, type, side, price, remaining_quantity
	FROM orders
	WHERE status IN ('open', 'partially_filled')
`)
	if err != nil {
		log.Fatal("Failed to load order book:", err)
	}
	defer rows.Close()

	for rows.Next() {
		var o models.Order
		err := rows.Scan(&o.ID, &o.Symbol, &o.Type, &o.Side, &o.Price, &o.RemainingQuantity)
		if err != nil {
			log.Println("Skipping invalid order:", err)
			continue
		}
		matchEngine.ForceAddOrder(&o)
	}

	// Setup Gin
	r := gin.Default()

	// Inject DB and engine into context
	r.Use(func(c *gin.Context) {
		c.Set("db", db)
		c.Set("engine", matchEngine)
		c.Next()
	})

	// Routes
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	r.POST("/orders", api.PlaceOrder)
	r.GET("/orderbook", api.GetOrderBook)
	r.GET("/orders", api.GetAllOrders)
	r.GET("/orders/:orderId", api.GetOrderByID)
	r.DELETE("/orders/:orderId", api.CancelOrder)
	r.GET("/trades", api.GetTrades)
	r.Run(":3000")
}
