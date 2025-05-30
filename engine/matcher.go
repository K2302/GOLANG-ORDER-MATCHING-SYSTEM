package engine

import (
	"container/heap"
	"errors"
	"sync"
	"time"

	"golang-order-matching/models"
)

type OrderItem struct {
	Order     *models.Order
	Timestamp time.Time
	Idx       int
}

// Max-Heap: higher price first, FIFO on same price
type BuyHeap []*OrderItem

func (h BuyHeap) Len() int { return len(h) }
func (h BuyHeap) Less(i, j int) bool {
	if h[i].Order.Price == h[j].Order.Price {
		return h[i].Order.ID < h[j].Order.ID // FIFO
	}
	return h[i].Order.Price > h[j].Order.Price
}
func (h BuyHeap) Swap(i, j int)       { h[i], h[j] = h[j], h[i]; h[i].Idx, h[j].Idx = i, j }
func (h *BuyHeap) Push(x interface{}) { *h = append(*h, x.(*OrderItem)) }
func (h *BuyHeap) Pop() interface{} {
	old := *h
	n := len(old)
	item := old[n-1]
	*h = old[:n-1]
	return item
}

// Min-Heap: lower price first, FIFO on same price
type SellHeap []*OrderItem

func (h SellHeap) Len() int { return len(h) }
func (h SellHeap) Less(i, j int) bool {
	if h[i].Order.Price == h[j].Order.Price {
		return h[i].Order.ID < h[j].Order.ID // FIFO
	}
	return h[i].Order.Price < h[j].Order.Price
}
func (h SellHeap) Swap(i, j int)       { h[i], h[j] = h[j], h[i]; h[i].Idx, h[j].Idx = i, j }
func (h *SellHeap) Push(x interface{}) { *h = append(*h, x.(*OrderItem)) }
func (h *SellHeap) Pop() interface{} {
	old := *h
	n := len(old)
	item := old[n-1]
	*h = old[:n-1]
	return item
}

type Engine struct {
	buyHeap  *BuyHeap
	sellHeap *SellHeap
	mu       sync.Mutex
}

func NewEngine() *Engine {
	bh := &BuyHeap{}
	sh := &SellHeap{}
	heap.Init(bh)
	heap.Init(sh)
	return &Engine{buyHeap: bh, sellHeap: sh}
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func (e *Engine) Match(o *models.Order) ([]*models.Trade, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	var trades []*models.Trade
	now := time.Now()
	remaining := o.Quantity

	switch o.Side {
	case "buy":
		for remaining > 0 && e.sellHeap.Len() > 0 {
			best := heap.Pop(e.sellHeap).(*OrderItem)

			if o.Type == "limit" && best.Order.Price > o.Price {
				heap.Push(e.sellHeap, best)
				break
			}

			matchQty := min(remaining, best.Order.Quantity)
			remaining -= matchQty
			best.Order.Quantity -= matchQty

			trades = append(trades, &models.Trade{
				OrderID:             o.ID,
				MatchedOrderID:      best.Order.ID,
				Price:               best.Order.Price,
				Quantity:            matchQty,
				TradedAt:            now,
				MatchedInitialQty:   best.Order.Quantity + matchQty,
				MatchedRemainingQty: best.Order.Quantity,
			})

			if best.Order.Quantity > 0 {
				heap.Push(e.sellHeap, best)
			}
		}

		if o.Type == "market" && len(trades) == 0 {
			return nil, errors.New("market order could not be filled — no sell orders available")
		}
		if o.Type == "limit" && remaining > 0 {
			o.Quantity = remaining
			heap.Push(e.buyHeap, &OrderItem{Order: o, Timestamp: now})
		}

	case "sell":
		for remaining > 0 && e.buyHeap.Len() > 0 {
			best := heap.Pop(e.buyHeap).(*OrderItem)

			if o.Type == "limit" && best.Order.Price < o.Price {
				heap.Push(e.buyHeap, best)
				break
			}

			matchQty := min(remaining, best.Order.Quantity)
			remaining -= matchQty
			best.Order.Quantity -= matchQty

			trades = append(trades, &models.Trade{
				OrderID:             o.ID,
				MatchedOrderID:      best.Order.ID,
				Price:               best.Order.Price,
				Quantity:            matchQty,
				TradedAt:            now,
				MatchedInitialQty:   best.Order.Quantity + matchQty,
				MatchedRemainingQty: best.Order.Quantity,
			})

			if best.Order.Quantity > 0 {
				heap.Push(e.buyHeap, best)
			}
		}

		if o.Type == "market" && len(trades) == 0 {
			return nil, errors.New("market order could not be filled — no buy orders available")
		}
		if o.Type == "limit" && remaining > 0 {
			o.Quantity = remaining
			heap.Push(e.sellHeap, &OrderItem{Order: o, Timestamp: now})
		}

	default:
		return nil, errors.New("invalid order side")
	}

	o.RemainingQuantity = remaining
	return trades, nil
}

func (e *Engine) ForceAddOrder(o *models.Order) {
	now := time.Now()
	item := &OrderItem{Order: o, Timestamp: now}
	if o.Side == "buy" {
		heap.Push(e.buyHeap, item)
	} else {
		heap.Push(e.sellHeap, item)
	}
}

func (e *Engine) BuyHeap() *BuyHeap   { return e.buyHeap }
func (e *Engine) SellHeap() *SellHeap { return e.sellHeap }
