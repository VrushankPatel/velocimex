package orderbook

import (
	"fmt"
	"sync"

	"velocimex/internal/normalizer"
)

// Manager manages multiple order books
type Manager struct {
	books map[string]*OrderBook
	mu    sync.RWMutex
}

// NewManager creates a new order book manager
func NewManager() *Manager {
	return &Manager{
		books: make(map[string]*OrderBook),
	}
}

// GetOrderBook returns the order book for a symbol
func (m *Manager) GetOrderBook(symbol string) *OrderBook {
	m.mu.RLock()
	book, ok := m.books[symbol]
	m.mu.RUnlock()
	
	if ok {
		return book
	}
	
	// Create a new order book if it doesn't exist
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Double-check in case another goroutine created it
	if book, ok := m.books[symbol]; ok {
		return book
	}
	
	book = NewOrderBook(symbol)
	m.books[symbol] = book
	return book
}

// GetSymbols returns all symbols with order books
func (m *Manager) GetSymbols() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	symbols := make([]string, 0, len(m.books))
	for symbol := range m.books {
		symbols = append(symbols, symbol)
	}
	
	return symbols
}

// GetAllOrderBooks returns all order books
func (m *Manager) GetAllOrderBooks() map[string]*OrderBook {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	// Create a copy of the map
	books := make(map[string]*OrderBook, len(m.books))
	for symbol, book := range m.books {
		books[symbol] = book
	}
	
	return books
}

// UpdateOrderBook updates an order book with new data from an exchange
func (m *Manager) UpdateOrderBook(exchange, symbol string, bids, asks []normalizer.PriceLevel) {
	// Create a composite key for exchange-specific order books
	key := fmt.Sprintf("%s:%s", exchange, symbol)
	
	book := m.GetOrderBook(key)
	book.Update(bids, asks)
}