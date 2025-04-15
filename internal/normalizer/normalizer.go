package normalizer

import (
	"time"
)

// PriceLevel represents a price level in an order book
type PriceLevel struct {
	Price  float64 `json:"price"`
	Volume float64 `json:"volume"`
}

// Trade represents a normalized trade
type Trade struct {
	Exchange  string    `json:"exchange"`
	Symbol    string    `json:"symbol"`
	Price     float64   `json:"price"`
	Volume    float64   `json:"volume"`
	Side      string    `json:"side"` // "buy" or "sell"
	Timestamp time.Time `json:"timestamp"`
	ID        string    `json:"id"`
}

// OrderBookUpdate represents a normalized order book update
type OrderBookUpdate struct {
	Exchange  string       `json:"exchange"`
	Symbol    string       `json:"symbol"`
	Bids      []PriceLevel `json:"bids"`
	Asks      []PriceLevel `json:"asks"`
	Timestamp time.Time    `json:"timestamp"`
	Snapshot  bool         `json:"snapshot"`
}

// Normalizer normalizes market data from different exchanges
type Normalizer struct {
	// You might add exchange-specific mappings or configuration here
}

// New creates a new normalizer
func New() *Normalizer {
	return &Normalizer{}
}

// NormalizeTrade normalizes a trade from an exchange
func (n *Normalizer) NormalizeTrade(exchange, symbol string, data map[string]interface{}) *Trade {
	// This is a simplified implementation
	// In a real system, this would parse exchange-specific trade data
	// and convert it to a standard format
	
	// For now, return a placeholder trade
	return &Trade{
		Exchange:  exchange,
		Symbol:    symbol,
		Price:     0,
		Volume:    0,
		Side:      "buy",
		Timestamp: time.Now(),
		ID:        "",
	}
}

// NormalizeOrderBook normalizes an order book from an exchange
func (n *Normalizer) NormalizeOrderBook(exchange, symbol string, data map[string]interface{}) *OrderBookUpdate {
	// This is a simplified implementation
	// In a real system, this would parse exchange-specific order book data
	// and convert it to a standard format
	
	// For now, return a placeholder order book update
	return &OrderBookUpdate{
		Exchange:  exchange,
		Symbol:    symbol,
		Bids:      make([]PriceLevel, 0),
		Asks:      make([]PriceLevel, 0),
		Timestamp: time.Now(),
		Snapshot:  true,
	}
}

// NormalizeSymbol normalizes a symbol from exchange-specific to standard format
func (n *Normalizer) NormalizeSymbol(exchange, symbol string) string {
	// This is a simplified implementation
	// In a real system, this would convert exchange-specific symbols
	// to a standard format (e.g., "BTCUSD" on all exchanges)
	
	// For now, just return the input symbol
	return symbol
}