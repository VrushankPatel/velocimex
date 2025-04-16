package orderbook

import (
	"time"
	"velocimex/internal/normalizer"
)

// PriceLevel represents a single price level in the order book
type PriceLevel struct {
	Price  float64 `json:"price"`
	Volume float64 `json:"volume"`
}

// Update represents an order book update
type Update struct {
	Symbol    string                  `json:"symbol"`
	Timestamp time.Time               `json:"timestamp"`
	Bids      []normalizer.PriceLevel `json:"bids"`
	Asks      []normalizer.PriceLevel `json:"asks"`
}

// Convert converts a normalizer.PriceLevel slice to an orderbook.PriceLevel slice
func ConvertPriceLevels(levels []normalizer.PriceLevel) []PriceLevel {
	result := make([]PriceLevel, len(levels))
	for i, level := range levels {
		result[i] = PriceLevel{
			Price:  level.Price,
			Volume: level.Volume,
		}
	}
	return result
}
