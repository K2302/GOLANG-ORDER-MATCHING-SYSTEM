package models

type Order struct {
	ID                int64   `json:"id"`
	Symbol            string  `json:"symbol"`
	Type              string  `json:"type"` // "limit" or "market"
	Side              string  `json:"side"` // "buy" or "sell"
	Price             float64 `json:"price,omitempty"`
	Quantity          float64 `json:"quantity"`
	RemainingQuantity float64 `json:"-"`
	Status            string  `json:"status"`
}
