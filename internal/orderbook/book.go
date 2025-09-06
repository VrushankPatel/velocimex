package orderbook

import (
	"sort"
	"sync"
	"time"

	"velocimex/internal/normalizer"
)

// OrderBook represents an order book for a symbol
type OrderBook struct {
	Symbol    string
	Timestamp time.Time
	Bids      []normalizer.PriceLevel
	Asks      []normalizer.PriceLevel
	mu        sync.RWMutex
}

// NewOrderBook creates a new order book
func NewOrderBook(symbol string) *OrderBook {
	return &OrderBook{
		Symbol:    symbol,
		Timestamp: time.Now(),
		Bids:      make([]normalizer.PriceLevel, 0),
		Asks:      make([]normalizer.PriceLevel, 0),
	}
}

// Update updates the order book with new data
func (b *OrderBook) Update(bids, asks []normalizer.PriceLevel) {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	b.Timestamp = time.Now()
	
	// Sort bids (highest first)
	sort.Slice(bids, func(i, j int) bool {
		return bids[i].Price > bids[j].Price
	})
	
	// Sort asks (lowest first)
	sort.Slice(asks, func(i, j int) bool {
		return asks[i].Price < asks[j].Price
	})
	
	b.Bids = bids
	b.Asks = asks
}

// GetDepth returns the top N levels of the order book
func (b *OrderBook) GetDepth(n int) ([]normalizer.PriceLevel, []normalizer.PriceLevel) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	
	bids := make([]normalizer.PriceLevel, 0, n)
	asks := make([]normalizer.PriceLevel, 0, n)
	
	if len(b.Bids) > 0 {
		if n >= len(b.Bids) {
			bids = append(bids, b.Bids...)
		} else {
			bids = append(bids, b.Bids[:n]...)
		}
	}
	
	if len(b.Asks) > 0 {
		if n >= len(b.Asks) {
			asks = append(asks, b.Asks...)
		} else {
			asks = append(asks, b.Asks[:n]...)
		}
	}
	
	return bids, asks
}

// GetMidPrice returns the mid price of the order book
func (b *OrderBook) GetMidPrice() float64 {
	b.mu.RLock()
	defer b.mu.RUnlock()
	
	if len(b.Bids) == 0 || len(b.Asks) == 0 {
		return 0
	}
	
	return (b.Bids[0].Price + b.Asks[0].Price) / 2
}

// GetTimestamp returns the timestamp of the last update
func (b *OrderBook) GetTimestamp() time.Time {
	b.mu.RLock()
	defer b.mu.RUnlock()
	
	return b.Timestamp
}

// GetSpread returns the spread of the order book
func (b *OrderBook) GetSpread() float64 {
	b.mu.RLock()
	defer b.mu.RUnlock()
	
	if len(b.Bids) == 0 || len(b.Asks) == 0 {
		return 0
	}
	
	return b.Asks[0].Price - b.Bids[0].Price
}

// GetSpreadPercentage returns the spread as a percentage of the mid price
func (b *OrderBook) GetSpreadPercentage() float64 {
	b.mu.RLock()
	defer b.mu.RUnlock()
	
	if len(b.Bids) == 0 || len(b.Asks) == 0 {
		return 0
	}
	
	midPrice := (b.Bids[0].Price + b.Asks[0].Price) / 2
	spread := b.Asks[0].Price - b.Bids[0].Price
	
	if midPrice == 0 {
		return 0
	}
	
	return spread / midPrice * 100
}

// GetBestBid returns the best bid price level
func (b *OrderBook) GetBestBid() *normalizer.PriceLevel {
	b.mu.RLock()
	defer b.mu.RUnlock()
	
	if len(b.Bids) == 0 {
		return nil
	}
	
	return &b.Bids[0]
}

// GetBestAsk returns the best ask price level
func (b *OrderBook) GetBestAsk() *normalizer.PriceLevel {
	b.mu.RLock()
	defer b.mu.RUnlock()
	
	if len(b.Asks) == 0 {
		return nil
	}
	
	return &b.Asks[0]
}