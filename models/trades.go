package models

import "time"

type Trade struct {
    ID              int64     `json:"id"`
    OrderID         int64     `json:"order_id"`
    MatchedOrderID  int64     `json:"matched_order_id"`
    Price           float64   `json:"price"`
    Quantity        float64   `json:"quantity"`
    TradedAt        time.Time `json:"traded_at"`
}
