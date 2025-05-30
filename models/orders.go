
package models

type Order struct {
    ID                int64   `json:"id"`
    Symbol            string  `json:"symbol"`             // e.g., "AAPL"
    Type              string  `json:"type"`               // "market" or "limit"
    Side              string  `json:"side"`               // "buy" or "sell"
    Price             float64 `json:"price,omitempty"`    // Only for limit orders
    Quantity          float64 `json:"quantity"`           // Initial quantity
    RemainingQuantity float64 `json:"remaining_quantity"` // Tracked internally
    Status            string  `json:"status"`             // open/filled/canceled
}
