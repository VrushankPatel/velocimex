package strategy

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"velocimex/internal/orderbook"
)

// ArbitrageConfig contains configuration for the arbitrage strategy
type ArbitrageConfig struct {
	Name                  string             `yaml:"name"`
	Symbols               []string           `yaml:"symbols"`
	Exchanges             []string           `yaml:"exchanges"`
	UpdateInterval        time.Duration      `yaml:"updateInterval"`
	MinimumSpread         float64            `yaml:"minimumSpread"`
	MaxSlippage           float64            `yaml:"maxSlippage"`
	MinProfitThreshold    float64            `yaml:"minProfitThreshold"`
	MaxExecutionLatency   int64              `yaml:"maxExecutionLatency"`
	SimultaneousExchanges int                `yaml:"simultaneousExchanges"`
	ExchangeFees          map[string]float64 `yaml:"exchangeFees"`
	RiskLimit             float64            `yaml:"riskLimit"`
}

// ArbitrageOpportunity represents a potential arbitrage opportunity
type ArbitrageOpportunity struct {
	BuyExchange     string    `json:"buyExchange"`
	SellExchange    string    `json:"sellExchange"`
	Symbol          string    `json:"symbol"`
	BuyPrice        float64   `json:"buyPrice"`
	SellPrice       float64   `json:"sellPrice"`
	MaxVolume       float64   `json:"maxVolume"`
	ProfitPercent   float64   `json:"profitPercent"`
	EstimatedProfit float64   `json:"estimatedProfit"`
	Timestamp       time.Time `json:"timestamp"`
	LatencyEstimate int64     `json:"latencyEstimate"`
	IsValid         bool      `json:"isValid"`
}

// arbitrageStrategy implements the ArbitrageStrategy interface
type arbitrageStrategy struct {
	name          string
	config        ArbitrageConfig
	orderBooks    *orderbook.Manager
	opportunities []ArbitrageOpportunity
	signals       []TradeSignal
	positions     []Position
	profitLoss    float64
	metrics       StrategyMetrics
	running       bool
	startTime     time.Time
	lastUpdate    time.Time
	signalsCount  int
	mu            sync.RWMutex
	ctx           context.Context
	cancel        context.CancelFunc
}

// NewArbitrageStrategy creates a new arbitrage strategy
func NewArbitrageStrategy(config ArbitrageConfig) Strategy {
	return &arbitrageStrategy{
		name:          config.Name,
		config:        config,
		opportunities: make([]ArbitrageOpportunity, 0),
		signals:       make([]TradeSignal, 0),
		positions:     make([]Position, 0),
		metrics:       StrategyMetrics{},
	}
}

// GetName returns the strategy name
func (s *arbitrageStrategy) GetName() string {
	return s.name
}

// Execute runs the strategy
func (s *arbitrageStrategy) Execute() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return fmt.Errorf("strategy not running")
	}

	// Implementation details...
	return nil
}

// GetSignals returns recent signals as basic Signal type
func (s *arbitrageStrategy) GetSignals() []Signal {
	s.mu.RLock()
	defer s.mu.RUnlock()

	signals := make([]Signal, len(s.signals))
	for i, ts := range s.signals {
		signals[i] = Signal{
			Symbol:    ts.Symbol,
			Side:      ts.Side,
			Price:     ts.Price,
			Volume:    ts.Volume,
			Exchange:  ts.Exchange,
			Timestamp: ts.Timestamp,
		}
	}
	return signals
}

// GetResults returns the strategy results
func (s *arbitrageStrategy) GetResults() StrategyResults {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return StrategyResults{
		Name:             s.name,
		Running:          s.running,
		StartTime:        s.startTime,
		LastUpdate:       s.lastUpdate,
		SignalsGenerated: s.signalsCount,
		ProfitLoss:       s.profitLoss,
		RecentSignals:    s.signals,
		CurrentPositions: s.positions,
		Metrics:          s.metrics,
	}
}

// Start starts the strategy
func (s *arbitrageStrategy) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("strategy already running")
	}

	s.ctx, s.cancel = context.WithCancel(ctx)
	s.running = true
	s.startTime = time.Now()

	// Start the strategy loop
	go s.run()

	log.Printf("Started %s strategy", s.name)
	return nil
}

// Stop stops the strategy
func (s *arbitrageStrategy) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return fmt.Errorf("strategy not running")
	}

	if s.cancel != nil {
		s.cancel()
	}
	s.running = false
	log.Printf("Stopped %s strategy", s.name)
	return nil
}

// IsRunning returns whether the strategy is currently running
func (s *arbitrageStrategy) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// GetOpportunities returns current arbitrage opportunities
func (s *arbitrageStrategy) GetOpportunities() []ArbitrageOpportunity {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return append([]ArbitrageOpportunity{}, s.opportunities...)
}

// SetOrderBookManager sets the order book manager
func (s *arbitrageStrategy) SetOrderBookManager(manager *orderbook.Manager) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.orderBooks = manager
}

// run is the main strategy loop
func (s *arbitrageStrategy) run() {
	ticker := time.NewTicker(s.config.UpdateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.updateOpportunities()
		}
	}
}

// updateOpportunities finds arbitrage opportunities
func (s *arbitrageStrategy) updateOpportunities() {
	// Get the configured symbols and exchanges
	symbols := s.config.Symbols
	exchanges := s.config.Exchanges

	// Create a new slice to store opportunities
	newOpps := make([]ArbitrageOpportunity, 0)

	// Check each symbol
	for _, symbol := range symbols {
		// Check each exchange pair
		for i, buyExchange := range exchanges {
			for j, sellExchange := range exchanges {
				// Skip same exchange (can't arbitrage on the same exchange)
				if i == j {
					continue
				}

				// Look for an arbitrage opportunity
				opportunity, found := s.detectOpportunity(symbol, buyExchange, sellExchange)
				if found && opportunity.IsValid {
					newOpps = append(newOpps, opportunity)

					// Generate trading signals if we have a valid opportunity
					s.generateSignal(opportunity)
				}
			}
		}
	}

	// Update the opportunities
	s.mu.Lock()
	s.opportunities = newOpps
	s.lastUpdate = time.Now()
	s.mu.Unlock()
}

// detectOpportunity checks for an arbitrage opportunity between two exchanges
func (s *arbitrageStrategy) detectOpportunity(symbol, buyExchange, sellExchange string) (ArbitrageOpportunity, bool) {
	// This is a simplified implementation. In a real system, you would need to:
	// 1. Get the actual order books for the exchanges
	// 2. Calculate the exact volume you can trade at each price level
	// 3. Account for exchange fees, latency, etc.

	// For now, we'll simulate finding an opportunity
	opportunity := ArbitrageOpportunity{
		Symbol:          symbol,
		BuyExchange:     buyExchange,
		SellExchange:    sellExchange,
		BuyPrice:        9950.0, // These would be retrieved from actual order books
		SellPrice:       10050.0,
		ProfitPercent:   0,
		EstimatedProfit: 0,
		LatencyEstimate: 50, // ms
		IsValid:         false,
	}

	// Calculate fees
	buyFee := s.config.ExchangeFees[buyExchange]
	sellFee := s.config.ExchangeFees[sellExchange]

	// Calculate profit percentage after fees
	costBasis := opportunity.BuyPrice * (1 + buyFee)
	sellProceeds := opportunity.SellPrice * (1 - sellFee)
	profitPercent := (sellProceeds - costBasis) / costBasis * 100

	opportunity.ProfitPercent = profitPercent
	opportunity.EstimatedProfit = (sellProceeds - costBasis)

	// Check if the opportunity is valid
	opportunity.IsValid = profitPercent >= s.config.MinProfitThreshold &&
		opportunity.LatencyEstimate <= s.config.MaxExecutionLatency

	return opportunity, true
}

// generateSignal creates trading signals from an arbitrage opportunity
func (s *arbitrageStrategy) generateSignal(opportunity ArbitrageOpportunity) {
	// Create buy signal
	buySignal := TradeSignal{
		Strategy:   s.name,
		Symbol:     opportunity.Symbol,
		Side:       "buy",
		Price:      opportunity.BuyPrice,
		Volume:     1.0, // Fixed volume for simplicity
		Exchange:   opportunity.BuyExchange,
		Timestamp:  time.Now(),
		Confidence: calculateConfidence(opportunity),
		Reason:     fmt.Sprintf("Arbitrage opportunity with %.2f%% profit potential", opportunity.ProfitPercent),
	}

	// Create sell signal
	sellSignal := TradeSignal{
		Strategy:   s.name,
		Symbol:     opportunity.Symbol,
		Side:       "sell",
		Price:      opportunity.SellPrice,
		Volume:     1.0, // Fixed volume for simplicity
		Exchange:   opportunity.SellExchange,
		Timestamp:  time.Now(),
		Confidence: calculateConfidence(opportunity),
		Reason:     fmt.Sprintf("Arbitrage opportunity with %.2f%% profit potential", opportunity.ProfitPercent),
	}

	// Update strategy results
	s.mu.Lock()
	s.signalsCount += 2 // One buy, one sell

	// Keep only the most recent signals (max 10)
	if len(s.signals) >= 10 {
		s.signals = s.signals[1:]
	}
	s.signals = append(s.signals, buySignal)

	if len(s.signals) >= 10 {
		s.signals = s.signals[1:]
	}
	s.signals = append(s.signals, sellSignal)

	// Update metrics
	s.metrics.AverageLatency = float64(opportunity.LatencyEstimate)
	s.mu.Unlock()

	// Log the opportunity
	log.Printf("Arbitrage opportunity: Buy %s on %s at %.2f, Sell on %s at %.2f, Profit: %.2f%%",
		opportunity.Symbol, opportunity.BuyExchange, opportunity.BuyPrice,
		opportunity.SellExchange, opportunity.SellPrice, opportunity.ProfitPercent)
}

// calculateConfidence determines the confidence level of a signal
func calculateConfidence(opportunity ArbitrageOpportunity) float64 {
	// A simple confidence calculation
	// Higher profit and lower latency = higher confidence

	// Normalize profit percentage (assume max of 5%)
	normalizedProfit := opportunity.ProfitPercent / 5.0
	if normalizedProfit > 1.0 {
		normalizedProfit = 1.0
	}

	// Normalize latency (inverse, since lower latency is better)
	maxLatency := 200.0 // ms
	normalizedLatency := 1.0 - float64(opportunity.LatencyEstimate)/maxLatency
	if normalizedLatency < 0 {
		normalizedLatency = 0
	}

	// Combine factors (equal weighting)
	confidence := (normalizedProfit + normalizedLatency) / 2.0

	return confidence
}
