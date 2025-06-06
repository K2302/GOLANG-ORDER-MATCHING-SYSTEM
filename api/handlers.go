package api

import (
	"database/sql"
	"net/http"

	"golang-order-matching/engine"
	"golang-order-matching/models"

	"github.com/gin-gonic/gin"
)

func PlaceOrder(c *gin.Context) {
	var order models.Order
	if err := c.ShouldBindJSON(&order); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payload"})
		return
	}

	if order.Quantity <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Quantity must be positive"})
		return
	}
	if order.Type == "limit" && order.Price <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Limit order must have a valid price"})
		return
	}
	if order.Type == "market" && order.Price != 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Market order should not specify price"})
		return
	}
	if order.Side != "buy" && order.Side != "sell" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Order side must be 'buy' or 'sell'"})
		return
	}
	if order.Symbol == "" {
		order.Symbol = "XYZ"
	}

	db := c.MustGet("db").(*sql.DB)
	eng := c.MustGet("engine").(*engine.Engine)

	tx, err := db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction begin failed"})
		return
	}

	order.RemainingQuantity = order.Quantity
	order.Status = "open"

	// Insert new order into DB
	res, err := tx.Exec(`
		INSERT INTO orders (symbol, type, side, price, initial_quantity, remaining_quantity, status)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		order.Symbol, order.Type, order.Side, order.Price,
		order.Quantity, order.RemainingQuantity, order.Status,
	)
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Insert order failed"})
		return
	}
	orderID, _ := res.LastInsertId()
	order.ID = orderID
	originalQty := order.Quantity

	// Perform matching
	trades, err := eng.Match(&order)
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Insert trades and update matched orders
	for _, t := range trades {
		var buyID, sellID int64
		if order.Side == "buy" {
			buyID, sellID = t.OrderID, t.MatchedOrderID
		} else {
			buyID, sellID = t.MatchedOrderID, t.OrderID
		}

		_, err := tx.Exec(`
			INSERT INTO trades (buy_order_id, sell_order_id, price, quantity, traded_at)
			VALUES (?, ?, ?, ?, ?)`,
			buyID, sellID, t.Price, t.Quantity, t.TradedAt,
		)
		if err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Insert trade failed"})
			return
		}

		matchedStatus := "open"
		if t.MatchedRemainingQty == 0 {
			matchedStatus = "filled"
		} else if t.MatchedRemainingQty < t.MatchedInitialQty {
			matchedStatus = "partially_filled"
		}

		_, err = tx.Exec(`
			UPDATE orders
			SET remaining_quantity = ?, status = ?
			WHERE id = ?`,
			t.MatchedRemainingQty, matchedStatus, t.MatchedOrderID,
		)
		if err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Update matched order failed"})
			return
		}
	}

	// Update the new order’s final status
	finalStatus := "open"
	if order.RemainingQuantity == 0 {
		finalStatus = "filled"
	} else if order.RemainingQuantity < originalQty {
		finalStatus = "partially_filled"
	}

	_, err = tx.Exec(`
		UPDATE orders
		SET remaining_quantity = ?, status = ?
		WHERE id = ?`,
		order.RemainingQuantity, finalStatus, order.ID,
	)
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Update incoming order failed"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"order_id":           orderID,
		"remaining_quantity": order.RemainingQuantity,
		"status":             finalStatus,
		"trades":             trades,
	})
}
func GetOrderByID(c *gin.Context) {
	db := c.MustGet("db").(*sql.DB)
	id := c.Param("orderId")

	var order models.Order
	err := db.QueryRow(`
		SELECT id, symbol, type, side, price, initial_quantity, remaining_quantity, status
		FROM orders
		WHERE id = ?`, id).Scan(
		&order.ID, &order.Symbol, &order.Type, &order.Side,
		&order.Price, &order.Quantity, &order.RemainingQuantity, &order.Status,
	)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve order"})
		return
	}

	c.JSON(http.StatusOK, order)
}

func CancelOrder(c *gin.Context) {
	db := c.MustGet("db").(*sql.DB)
	orderID := c.Param("orderId")

	// Check if order exists and is cancelable
	var status string
	err := db.QueryRow(`SELECT status FROM orders WHERE id = ?`, orderID).Scan(&status)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve order status"})
		return
	}

	if status != "open" && status != "partially_filled" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Only open or partially_filled orders can be canceled"})
		return
	}

	// Cancel order
	_, err = db.Exec(`UPDATE orders SET status = 'canceled' WHERE id = ?`, orderID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to cancel order"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Order canceled successfully"})
}

func GetAllOrders(c *gin.Context) {
	db := c.MustGet("db").(*sql.DB)

	rows, err := db.Query(`
    SELECT id, symbol, side, type, price, initial_quantity, remaining_quantity, status 
    FROM orders
    ORDER BY id DESC
`)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch orders"})
		return
	}
	defer rows.Close()

	var orders []gin.H
	for rows.Next() {
		var o models.Order
		err := rows.Scan(&o.ID, &o.Symbol, &o.Side, &o.Type, &o.Price, &o.Quantity, &o.RemainingQuantity, &o.Status)
		if err != nil {
			continue
		}
		orders = append(orders, gin.H{
			"id":                 o.ID,
			"symbol":             o.Symbol,
			"side":               o.Side,
			"type":               o.Type,
			"price":              o.Price,
			"initial_quantity":   o.Quantity,
			"remaining_quantity": o.RemainingQuantity,
			"status":             o.Status,
		})
	}

	c.JSON(http.StatusOK, gin.H{"orders": orders})
}

func GetOrderBook(c *gin.Context) {
	eng := c.MustGet("engine").(*engine.Engine)

	buyList := []gin.H{}
	for _, item := range *eng.BuyHeap() {
		buyList = append(buyList, gin.H{
			"id":       item.Order.ID,
			"price":    item.Order.Price,
			"quantity": item.Order.Quantity,
		})
	}

	sellList := []gin.H{}
	for _, item := range *eng.SellHeap() {
		sellList = append(sellList, gin.H{
			"id":       item.Order.ID,
			"price":    item.Order.Price,
			"quantity": item.Order.Quantity,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"buy_orders":  buyList,
		"sell_orders": sellList,
	})
}
