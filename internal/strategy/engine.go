package strategy

import (
	"sync"
	"velocimex/internal/orderbook"
)

// Engine manages trading strategies
type Engine struct {
	orderBooks  *orderbook.Manager
	strategies  map[string]Strategy
	subscribers []chan<- Update
	mu          sync.RWMutex
}

// NewEngine creates a new strategy engine
func NewEngine(bookManager *orderbook.Manager) *Engine {
	return &Engine{
		orderBooks:  bookManager,
		strategies:  make(map[string]Strategy),
		subscribers: make([]chan<- Update, 0),
	}
}

// Subscribe subscribes to strategy updates
func (e *Engine) Subscribe(ch chan<- Update) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.subscribers = append(e.subscribers, ch)
}

// GetArbitrageOpportunities returns current arbitrage opportunities
func (e *Engine) GetArbitrageOpportunities() []ArbitrageOpportunity {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var opportunities []ArbitrageOpportunity
	for _, strategy := range e.strategies {
		if arb, ok := strategy.(ArbitrageStrategy); ok {
			opportunities = append(opportunities, arb.GetOpportunities()...)
		}
	}
	return opportunities
}

// RegisterStrategy registers a new strategy
func (e *Engine) RegisterStrategy(s Strategy) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.strategies[s.GetName()] = s

	// If the strategy is an ArbitrageStrategy, set its order book manager
	if arb, ok := s.(ArbitrageStrategy); ok {
		arb.SetOrderBookManager(e.orderBooks)
	}
}

// notifySubscribers sends an update to all subscribers
func (e *Engine) notifySubscribers(update Update) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	for _, ch := range e.subscribers {
		select {
		case ch <- update:
		default:
			// Skip if subscriber's channel is full
		}
	}
}
