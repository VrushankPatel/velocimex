package simulation

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"velocimex/internal/orderbook"
	"velocimex/internal/strategy"
)

// PaperTradingConfig contains configuration for paper trading
type PaperTradingConfig struct {
	InitialBalance   map[string]float64 `yaml:"initialBalance"`   // Asset -> Amount
	LatencySimulation bool              `yaml:"latencySimulation"`
	BaseLatency      int                `yaml:"baseLatency"`      // Base latency in milliseconds
	RandomLatency    int                `yaml:"randomLatency"`    // Random additional latency in milliseconds
	SlippageModel    string             `yaml:"slippageModel"`    // "none", "fixed", "proportional", "realistic"
	FixedSlippage    float64            `yaml:"fixedSlippage"`    // Fixed slippage in percentage
	ExchangeFees     map[string]float64 `yaml:"exchangeFees"`     // Exchange -> Fee percentage
}

// PaperTrader simulates trading without actual execution
type PaperTrader struct {
	config        PaperTradingConfig
	orderBooks    *orderbook.Manager
	balances      map[string]float64 // Asset -> Amount
	trades        []Trade
	muBalances    sync.RWMutex
	muTrades      sync.RWMutex
	strategies    []strategy.Strategy
	ctx           context.Context
	cancel        context.CancelFunc
	running       bool
	muRunning     sync.Mutex
}

// Trade represents a simulated trade
type Trade struct {
	Strategy   string    `json:"strategy"`
	Symbol     string    `json:"symbol"`
	Side       string    `json:"side"` // "buy" or "sell"
	Price      float64   `json:"price"`
	Volume     float64   `json:"volume"`
	Exchange   string    `json:"exchange"`
	Fee        float64   `json:"fee"`
	Timestamp  time.Time `json:"timestamp"`
	LatencyMS  int       `json:"latencyMs"`
	Slippage   float64   `json:"slippage"` // Percentage
	Successful bool      `json:"successful"`
	Reason     string    `json:"reason"`
}

// NewPaperTrader creates a new paper trading simulator
func NewPaperTrader(config PaperTradingConfig, bookManager *orderbook.Manager) *PaperTrader {
	return &PaperTrader{
		config:     config,
		orderBooks: bookManager,
		balances:   config.InitialBalance,
		trades:     make([]Trade, 0),
		strategies: make([]strategy.Strategy, 0),
	}
}

// RegisterStrategy adds a strategy to the paper trader
func (p *PaperTrader) RegisterStrategy(s strategy.Strategy) {
	p.strategies = append(p.strategies, s)
}

// Start begins paper trading simulation
func (p *PaperTrader) Start(ctx context.Context) error {
	p.muRunning.Lock()
	defer p.muRunning.Unlock()
	
	if p.running {
		return fmt.Errorf("paper trader already running")
	}
	
	p.ctx, p.cancel = context.WithCancel(ctx)
	p.running = true
	
	// Start all strategies
	for _, s := range p.strategies {
		if err := s.Start(p.ctx); err != nil {
			log.Printf("Failed to start strategy %s: %v", s.GetName(), err)
		}
	}
	
	// Start signal processing
	go p.processSignals()
	
	log.Println("Paper trading simulation started")
	return nil
}

// Stop halts paper trading simulation
func (p *PaperTrader) Stop() error {
	p.muRunning.Lock()
	defer p.muRunning.Unlock()
	
	if !p.running {
		return nil
	}
	
	p.cancel()
	p.running = false
	
	// Stop all strategies
	for _, s := range p.strategies {
		if s.IsRunning() {
			if err := s.Stop(); err != nil {
				log.Printf("Failed to stop strategy %s: %v", s.GetName(), err)
			}
		}
	}
	
	log.Println("Paper trading simulation stopped")
	return nil
}

// IsRunning returns whether the paper trader is currently running
func (p *PaperTrader) IsRunning() bool {
	p.muRunning.Lock()
	defer p.muRunning.Unlock()
	return p.running
}

// GetBalances returns the current simulated balances
func (p *PaperTrader) GetBalances() map[string]float64 {
	p.muBalances.RLock()
	defer p.muBalances.RUnlock()
	
	// Create a copy of the balances
	balances := make(map[string]float64, len(p.balances))
	for asset, amount := range p.balances {
		balances[asset] = amount
	}
	
	return balances
}

// GetTrades returns the simulated trades
func (p *PaperTrader) GetTrades(limit int) []Trade {
	p.muTrades.RLock()
	defer p.muTrades.RUnlock()
	
	if limit <= 0 || limit > len(p.trades) {
		limit = len(p.trades)
	}
	
	start := len(p.trades) - limit
	if start < 0 {
		start = 0
	}
	
	// Create a copy of the trades
	trades := make([]Trade, limit)
	copy(trades, p.trades[start:])
	
	return trades
}

// processSignals listens for signals from strategies and simulates trades
func (p *PaperTrader) processSignals() {
	// In a real implementation, we would subscribe to signals from each strategy
	// For this example, we'll poll the strategies for their results periodically
	
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	
	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			// Check each strategy for new signals
			for _, s := range p.strategies {
				results := s.GetResults()
				
				// Process any new signals
				for _, signal := range results.RecentSignals {
					// Check if we've already processed this signal
					// In a real implementation, you'd need a more robust way to track this
					if p.isSignalProcessed(signal) {
						continue
					}
					
					// Simulate the trade
					trade := p.simulateTrade(signal)
					
					// Record the trade
					p.muTrades.Lock()
					p.trades = append(p.trades, trade)
					p.muTrades.Unlock()
					
					// Update balances if the trade was successful
					if trade.Successful {
						p.updateBalances(trade)
					}
				}
			}
		}
	}
}

// isSignalProcessed checks if a signal has already been processed
func (p *PaperTrader) isSignalProcessed(signal strategy.TradeSignal) bool {
	// This is a simplified implementation
	// In a real system, you would need a more robust way to track processed signals
	
	p.muTrades.RLock()
	defer p.muTrades.RUnlock()
	
	// Check the most recent trades (up to 100)
	startIdx := len(p.trades) - 100
	if startIdx < 0 {
		startIdx = 0
	}
	
	for i := len(p.trades) - 1; i >= startIdx; i-- {
		trade := p.trades[i]
		
		// If we find a matching trade with the same timestamp, assume it's already processed
		if trade.Strategy == signal.Strategy &&
			trade.Symbol == signal.Symbol &&
			trade.Side == signal.Side &&
			trade.Exchange == signal.Exchange &&
			trade.Timestamp.Equal(signal.Timestamp) {
			return true
		}
	}
	
	return false
}

// simulateTrade simulates execution of a trade signal
func (p *PaperTrader) simulateTrade(signal strategy.TradeSignal) Trade {
	// Create a base trade record
	trade := Trade{
		Strategy:  signal.Strategy,
		Symbol:    signal.Symbol,
		Side:      signal.Side,
		Price:     signal.Price,
		Volume:    signal.Volume,
		Exchange:  signal.Exchange,
		Timestamp: time.Now(),
		Reason:    signal.Reason,
	}
	
	// Simulate latency if enabled
	if p.config.LatencySimulation {
		latency := p.simulateLatency()
		trade.LatencyMS = latency
		
		// Check if the trade would still be valid after latency
		if !p.isTradeValidAfterLatency(signal, latency) {
			trade.Successful = false
			trade.Reason = fmt.Sprintf("Trade expired due to latency (%dms)", latency)
			return trade
		}
	}
	
	// Calculate slippage
	slippage := p.calculateSlippage(signal)
	trade.Slippage = slippage
	
	// Adjust price based on slippage
	if signal.Side == "buy" {
		trade.Price = signal.Price * (1 + slippage/100)
	} else {
		trade.Price = signal.Price * (1 - slippage/100)
	}
	
	// Calculate fee
	fee := p.config.ExchangeFees[signal.Exchange]
	trade.Fee = trade.Price * trade.Volume * fee
	
	// Check if we have sufficient balance
	if !p.hasSufficientBalance(trade) {
		trade.Successful = false
		trade.Reason = "Insufficient balance"
		return trade
	}
	
	// Simulate successful trade
	trade.Successful = true
	
	return trade
}

// simulateLatency returns a simulated latency in milliseconds
func (p *PaperTrader) simulateLatency() int {
	// In a real implementation, you might use a more sophisticated model
	// For now, just return the base latency
	return p.config.BaseLatency
}

// isTradeValidAfterLatency checks if a trade would still be valid after latency
func (p *PaperTrader) isTradeValidAfterLatency(signal strategy.TradeSignal, latencyMS int) bool {
	// In a real implementation, you would check the current market price
	// and compare it to the signal price to see if the trade is still valid
	
	// For this example, always return true
	return true
}

// calculateSlippage estimates slippage for a trade
func (p *PaperTrader) calculateSlippage(signal strategy.TradeSignal) float64 {
	switch p.config.SlippageModel {
	case "none":
		return 0
	case "fixed":
		return p.config.FixedSlippage
	case "proportional":
		// Slippage proportional to volume
		return p.config.FixedSlippage * signal.Volume
	case "realistic":
		// In a real implementation, you would use the order book to calculate a realistic slippage
		// For now, just return a fixed value
		return 0.1
	default:
		return 0
	}
}

// hasSufficientBalance checks if there's enough balance for a trade
func (p *PaperTrader) hasSufficientBalance(trade Trade) bool {
	p.muBalances.RLock()
	defer p.muBalances.RUnlock()
	
	// Extract base and quote currencies from the symbol
	// For simplicity, assume symbols are in the format "BTC-USD" or "ETH/USDT"
	baseCurrency := trade.Symbol[:3]
	quoteCurrency := trade.Symbol[4:]
	
	if trade.Side == "buy" {
		// Check if we have enough of the quote currency
		cost := trade.Price * trade.Volume * (1 + p.config.ExchangeFees[trade.Exchange])
		balance, exists := p.balances[quoteCurrency]
		return exists && balance >= cost
	} else {
		// Check if we have enough of the base currency
		balance, exists := p.balances[baseCurrency]
		return exists && balance >= trade.Volume
	}
}

// updateBalances updates balances after a successful trade
func (p *PaperTrader) updateBalances(trade Trade) {
	p.muBalances.Lock()
	defer p.muBalances.Unlock()
	
	// Extract base and quote currencies from the symbol
	baseCurrency := trade.Symbol[:3]
	quoteCurrency := trade.Symbol[4:]
	
	if trade.Side == "buy" {
		// Decrease quote currency (e.g. USD)
		cost := trade.Price * trade.Volume * (1 + p.config.ExchangeFees[trade.Exchange])
		p.balances[quoteCurrency] -= cost
		
		// Increase base currency (e.g. BTC)
		if _, exists := p.balances[baseCurrency]; !exists {
			p.balances[baseCurrency] = 0
		}
		p.balances[baseCurrency] += trade.Volume
	} else {
		// Decrease base currency
		p.balances[baseCurrency] -= trade.Volume
		
		// Increase quote currency
		proceeds := trade.Price * trade.Volume * (1 - p.config.ExchangeFees[trade.Exchange])
		if _, exists := p.balances[quoteCurrency]; !exists {
			p.balances[quoteCurrency] = 0
		}
		p.balances[quoteCurrency] += proceeds
	}
}
