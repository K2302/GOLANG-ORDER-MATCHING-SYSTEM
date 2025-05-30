package models

import "time"

type Trade struct {
	OrderID             int64     `json:"order_id"`
	MatchedOrderID      int64     `json:"matched_order_id"`
	Price               float64   `json:"price"`
	Quantity            float64   `json:"quantity"`
	TradedAt            time.Time `json:"traded_at"`
	MatchedInitialQty   float64   `json:"matched_initial_qty"`   // initial qty of resting order
	MatchedRemainingQty float64   `json:"matched_remaining_qty"` // resting order’s qty after this trade
}
