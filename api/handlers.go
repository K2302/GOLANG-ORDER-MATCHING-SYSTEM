package api

import (
    "database/sql"
    "net/http"
    "sync"

    "github.com/gin-gonic/gin"
    "golang-order-matching/engine"
    "golang-order-matching/models"
)

var once sync.Once
var matchEngine *engine.Engine

func getEngine() *engine.Engine {
    once.Do(func() {
        matchEngine = engine.NewEngine()
    })
    return matchEngine
}


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

    db, ok := c.MustGet("db").(*sql.DB)
    if !ok {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "DB connection not available"})
        return
    }

    tx, err := db.Begin()
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to begin transaction"})
        return
    }

    // Save original quantity
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

    // Match order
    trades, err := getEngine().Match(&order)
    if err != nil {
        tx.Rollback()
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    // Log trades
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
    }

    // Determine status
    finalStatus := "open"
    if order.RemainingQuantity == 0 {
        finalStatus = "filled"
    } else if order.RemainingQuantity < order.Quantity {
        finalStatus = "partially_filled"
    }

    // Update remaining_quantity and status
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

// CancelOrder handles order cancellation
func CancelOrder(c *gin.Context) {
    c.JSON(http.StatusNotImplemented, gin.H{"message": "CancelOrder not implemented"})
}

// GetOrderBook returns the current order book

func GetOrderBook(c *gin.Context) {
    e := getEngine()

    buyList := []gin.H{}
    for _, item := range *e.BuyHeap() {
        buyList = append(buyList, gin.H{
            "id":       item.Order.ID,       // <- Order instead of order
            "price":    item.Order.Price,
            "quantity": item.Order.Quantity,
        })
    }

    sellList := []gin.H{}
    for _, item := range *e.SellHeap() {
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
