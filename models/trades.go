package models

type Trade struct {
    OrderID        int64   `json:"order_id"`
    MatchedOrderID int64   `json:"matched_order_id"`
    Price          float64 `json:"price"`
    Quantity       float64 `json:"quantity"`
    TradedAt       string  `json:"traded_at,omitempty"`
}
