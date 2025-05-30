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
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order payload"})
		return
	}

	if order.Type == "limit" && order.Price <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Limit order must have a positive price"})
		return
	}

	if order.Type == "market" && order.Price > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Market order should not specify a price"})
		return
	}

	if order.Symbol == "" {
		order.Symbol = "XYZ"
	}

	db := c.MustGet("db").(*sql.DB)
	eng := c.MustGet("engine").(*engine.Engine)

	tx, err := db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to begin transaction"})
		return
	}

	order.RemainingQuantity = order.Quantity
	order.Status = "open"

	res, err := tx.Exec(`
		INSERT INTO orders (symbol, type, side, price, initial_quantity, remaining_quantity, status)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		order.Symbol, order.Type, order.Side, order.Price,
		order.Quantity, order.RemainingQuantity, order.Status,
	)
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to insert order"})
		return
	}

	orderID, _ := res.LastInsertId()
	order.ID = orderID

	trades, err := eng.Match(&order)
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	for _, t := range trades {
		var buyID, sellID int64
		if order.Side == "buy" {
			buyID = t.OrderID
			sellID = t.MatchedOrderID
		} else {
			buyID = t.MatchedOrderID
			sellID = t.OrderID
		}

		_, err := tx.Exec(`
			INSERT INTO trades (buy_order_id, sell_order_id, price, quantity)
			VALUES (?, ?, ?, ?)`,
			buyID, sellID, t.Price, t.Quantity,
		)
		if err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to insert trade"})
			return
		}

		// ✅ Also update matched order's status + remaining qty
		_, err = tx.Exec(`
			UPDATE orders
			SET remaining_quantity = remaining_quantity - ?, 
			    status = CASE 
				             WHEN remaining_quantity - ? = 0 THEN 'filled'
				             WHEN remaining_quantity - ? < initial_quantity THEN 'partially_filled'
				             ELSE status
				         END
			WHERE id = ?`,
			t.Quantity, t.Quantity, t.Quantity, t.MatchedOrderID,
		)
		if err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update matched order"})
			return
		}
	}

	// Capture the original quantity at the beginning
	originalQty := order.Quantity

	// ... matching logic happens ...
	trades, err := eng.Match(&order)
	// Now set RemainingQuantity to what's left after matching
	order.RemainingQuantity = order.Quantity

	// Determine final status using original quantity
	finalStatus := "open"
	if order.RemainingQuantity == 0 {
		finalStatus = "filled"
	} else if order.RemainingQuantity < originalQty {
		finalStatus = "partially_filled"
	}
	originalQty := order.Quantity // store before matching
	trades, err := eng.Match(&order)
	// ...
	order.RemainingQuantity = order.Quantity

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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update order status"})
		return
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction failed"})
		return
	}

	_, err = tx.Exec(`
		UPDATE orders
		SET remaining_quantity = ?, status = ?
		WHERE id = ?`,
		order.RemainingQuantity, finalStatus, order.ID,
	)
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update order status"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"order_id":           orderID,
		"remaining_quantity": order.RemainingQuantity,
		"status":             finalStatus,
		"trades":             trades,
	})
}

func CancelOrder(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"message": "CancelOrder not implemented"})
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
