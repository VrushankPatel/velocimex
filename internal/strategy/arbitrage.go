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
        Name                 string             `yaml:"name"`
        Symbols              []string           `yaml:"symbols"`
        Exchanges            []string           `yaml:"exchanges"`
        UpdateInterval       time.Duration      `yaml:"updateInterval"`
        MinimumSpread        float64            `yaml:"minimumSpread"`
        MaxSlippage          float64            `yaml:"maxSlippage"`
        MinProfitThreshold   float64            `yaml:"minProfitThreshold"`
        MaxExecutionLatency  int64              `yaml:"maxExecutionLatency"`
        SimultaneousExchanges int               `yaml:"simultaneousExchanges"`
        ExchangeFees         map[string]float64 `yaml:"exchangeFees"`
        RiskLimit            float64            `yaml:"riskLimit"`
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

// ArbitrageStrategy implements cross-exchange arbitrage
type ArbitrageStrategy struct {
        config      ArbitrageConfig
        orderBooks  *orderbook.Manager
        running     bool
        done        chan struct{}
        ctx         context.Context
        cancel      context.CancelFunc
        
        // Store current opportunities
        muOpps       sync.RWMutex
        opportunities []ArbitrageOpportunity
        
        // Track strategy results
        muResults    sync.RWMutex
        results      StrategyResults
}

// NewArbitrageStrategy creates a new arbitrage strategy
func NewArbitrageStrategy(config ArbitrageConfig) *ArbitrageStrategy {
        // Initialize with default values
        results := StrategyResults{
                Name:             config.Name,
                Running:          false,
                StartTime:        time.Time{},
                LastUpdate:       time.Time{},
                SignalsGenerated: 0,
                ProfitLoss:       0,
                RecentSignals:    make([]TradeSignal, 0),
                CurrentPositions: make([]Position, 0),
                Metrics: StrategyMetrics{
                        WinRate:        0,
                        AverageProfit:  0,
                        AverageLoss:    0,
                        ProfitFactor:   0,
                        SharpeRatio:    0,
                        DrawdownMax:    0,
                        AverageLatency: 0,
                },
        }
        
        return &ArbitrageStrategy{
                config:        config,
                done:          make(chan struct{}),
                opportunities: make([]ArbitrageOpportunity, 0),
                results:       results,
        }
}

// SetOrderBookManager sets the order book manager
func (s *ArbitrageStrategy) SetOrderBookManager(manager *orderbook.Manager) {
        s.orderBooks = manager
}

// GetName returns the name of the strategy
func (s *ArbitrageStrategy) GetName() string {
        return s.config.Name
}

// Start begins strategy execution
func (s *ArbitrageStrategy) Start(ctx context.Context) error {
        s.muResults.Lock()
        defer s.muResults.Unlock()
        
        if s.running {
                return nil // Already running
        }
        
        s.ctx, s.cancel = context.WithCancel(ctx)
        s.running = true
        s.results.Running = true
        s.results.StartTime = time.Now()
        
        // Start the strategy loop
        go s.run()
        
        log.Printf("Started %s strategy", s.config.Name)
        return nil
}

// Stop halts strategy execution
func (s *ArbitrageStrategy) Stop() error {
        s.muResults.Lock()
        defer s.muResults.Unlock()
        
        if !s.running {
                return nil // Already stopped
        }
        
        s.cancel()
        s.running = false
        s.results.Running = false
        
        log.Printf("Stopped %s strategy", s.config.Name)
        return nil
}

// IsRunning returns whether the strategy is currently running
func (s *ArbitrageStrategy) IsRunning() bool {
        s.muResults.RLock()
        defer s.muResults.RUnlock()
        return s.running
}

// GetResults returns the current strategy results
func (s *ArbitrageStrategy) GetResults() StrategyResults {
        s.muResults.RLock()
        defer s.muResults.RUnlock()
        
        // Create a copy of the results
        results := s.results
        results.LastUpdate = time.Now()
        
        return results
}

// GetOpportunities returns the current arbitrage opportunities
func (s *ArbitrageStrategy) GetOpportunities() []ArbitrageOpportunity {
        s.muOpps.RLock()
        defer s.muOpps.RUnlock()
        
        // Create a copy of the opportunities
        opps := make([]ArbitrageOpportunity, len(s.opportunities))
        copy(opps, s.opportunities)
        
        return opps
}

// run is the main strategy loop
func (s *ArbitrageStrategy) run() {
        ticker := time.NewTicker(s.config.UpdateInterval)
        defer ticker.Stop()
        
        // Main strategy loop
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
func (s *ArbitrageStrategy) updateOpportunities() {
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
        s.muOpps.Lock()
        s.opportunities = newOpps
        s.muOpps.Unlock()
        
        // Update the results
        s.muResults.Lock()
        s.results.LastUpdate = time.Now()
        s.muResults.Unlock()
}

// detectOpportunity checks for an arbitrage opportunity between two exchanges
func (s *ArbitrageStrategy) detectOpportunity(symbol, buyExchange, sellExchange string) (ArbitrageOpportunity, bool) {
        // This is a simplified implementation. In a real system, you would need to:
        // 1. Get the actual order books for the exchanges
        // 2. Calculate the exact volume you can trade at each price level
        // 3. Account for exchange fees, latency, etc.
        
        // For now, we'll simulate finding an opportunity
        
        // Get order books for both exchanges
        // In a real implementation, we would look up the actual order books from the manager
        // and use them to calculate opportunities
        
        // This is just a placeholder for demonstration
        // In a real system, we would use the actual order book data
        // to calculate opportunities based on real market prices
        opportunity := ArbitrageOpportunity{
                BuyExchange:     buyExchange,
                SellExchange:    sellExchange,
                Symbol:          symbol,
                BuyPrice:        9950.0, // These would be retrieved from actual order books
                SellPrice:       10050.0,
                MaxVolume:       1.0,
                Timestamp:       time.Now(),
                LatencyEstimate: 50, // ms
        }
        
        // Calculate fees
        buyFee := s.config.ExchangeFees[buyExchange]
        sellFee := s.config.ExchangeFees[sellExchange]
        
        // Calculate profit percentage after fees
        costBasis := opportunity.BuyPrice * (1 + buyFee)
        sellProceeds := opportunity.SellPrice * (1 - sellFee)
        profitPercent := (sellProceeds - costBasis) / costBasis * 100
        
        opportunity.ProfitPercent = profitPercent
        opportunity.EstimatedProfit = (sellProceeds - costBasis) * opportunity.MaxVolume
        
        // Check if the opportunity is valid
        opportunity.IsValid = profitPercent >= s.config.MinProfitThreshold &&
                opportunity.LatencyEstimate <= s.config.MaxExecutionLatency
        
        return opportunity, true
}

// generateSignal creates trading signals from an arbitrage opportunity
func (s *ArbitrageStrategy) generateSignal(opportunity ArbitrageOpportunity) {
        // Create buy signal
        buySignal := TradeSignal{
                Strategy:   s.config.Name,
                Symbol:     opportunity.Symbol,
                Side:       "buy",
                Price:      opportunity.BuyPrice,
                Volume:     opportunity.MaxVolume,
                Exchange:   opportunity.BuyExchange,
                Timestamp:  time.Now(),
                Confidence: calculateConfidence(opportunity),
                Reason:     fmt.Sprintf("Arbitrage opportunity with %.2f%% profit potential", opportunity.ProfitPercent),
        }
        
        // Create sell signal
        sellSignal := TradeSignal{
                Strategy:   s.config.Name,
                Symbol:     opportunity.Symbol,
                Side:       "sell",
                Price:      opportunity.SellPrice,
                Volume:     opportunity.MaxVolume,
                Exchange:   opportunity.SellExchange,
                Timestamp:  time.Now(),
                Confidence: calculateConfidence(opportunity),
                Reason:     fmt.Sprintf("Arbitrage opportunity with %.2f%% profit potential", opportunity.ProfitPercent),
        }
        
        // Update strategy results
        s.muResults.Lock()
        s.results.SignalsGenerated += 2 // One buy, one sell
        
        // Keep only the most recent signals (max 10)
        if len(s.results.RecentSignals) >= 10 {
                s.results.RecentSignals = s.results.RecentSignals[1:]
        }
        s.results.RecentSignals = append(s.results.RecentSignals, buySignal)
        
        if len(s.results.RecentSignals) >= 10 {
                s.results.RecentSignals = s.results.RecentSignals[1:]
        }
        s.results.RecentSignals = append(s.results.RecentSignals, sellSignal)
        
        // Update metrics
        // In a real system, these would be calculated based on actual performance
        s.results.Metrics.AverageLatency = float64(opportunity.LatencyEstimate)
        s.muResults.Unlock()
        
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