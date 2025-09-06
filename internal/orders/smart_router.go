package orders

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/shopspring/decimal"
	"velocimex/internal/normalizer"
	"velocimex/internal/orderbook"
)

// SmartRouterConfig holds configuration for the smart router
type SmartRouterConfig struct {
	MaxSlippage     decimal.Decimal `json:"max_slippage"`
	MaxFee          decimal.Decimal `json:"max_fee"`
	LatencyWeight   float64         `json:"latency_weight"`
	VolumeWeight    float64         `json:"volume_weight"`
	PriceWeight     float64         `json:"price_weight"`
	FeeWeight       float64         `json:"fee_weight"`
	MinConfidence   float64         `json:"min_confidence"`
	DefaultTimeout  time.Duration   `json:"default_timeout"`
}

// DefaultSmartRouterConfig returns default configuration
func DefaultSmartRouterConfig() SmartRouterConfig {
	return SmartRouterConfig{
		MaxSlippage:    decimal.NewFromFloat(0.01), // 1%
		MaxFee:         decimal.NewFromFloat(0.001), // 0.1%
		LatencyWeight:  0.2,
		VolumeWeight:   0.3,
		PriceWeight:    0.4,
		FeeWeight:      0.1,
		MinConfidence:  0.7,
		DefaultTimeout: 5 * time.Second,
	}
}

// MarketData represents current market data for an exchange
type MarketData struct {
	Exchange      string
	Symbol        string
	BidPrice      decimal.Decimal
	AskPrice      decimal.Decimal
	BidVolume     decimal.Decimal
	AskVolume     decimal.Decimal
	LastPrice     decimal.Decimal
	Volume24h     decimal.Decimal
	FeeRate       decimal.Decimal
	Latency       time.Duration
	Timestamp     time.Time
	OrderBook     *orderbook.OrderBook
}

// ExchangeRoute represents a specific route to an exchange
type ExchangeRoute struct {
	Exchange string
	Route    string
	Priority int
	Active   bool
}

// SmartRouterImpl implements the SmartRouter interface
type SmartRouterImpl struct {
	config        SmartRouterConfig
	marketData    map[string]map[string]*MarketData
	routes        map[string][]ExchangeRoute
	orderBookMgr  *orderbook.Manager
	mu            sync.RWMutex
	lastUpdate    time.Time
}

// NewSmartRouter creates a new smart router instance
func NewSmartRouter(config SmartRouterConfig, orderBookMgr *orderbook.Manager) *SmartRouterImpl {
	return &SmartRouterImpl{
		config:       config,
		marketData:   make(map[string]map[string]*MarketData),
		routes:       make(map[string][]ExchangeRoute),
		orderBookMgr: orderBookMgr,
		lastUpdate:   time.Now(),
	}
}

// RouteOrder routes an order to the best exchange based on various factors
func (sr *SmartRouterImpl) RouteOrder(ctx context.Context, order *OrderRequest) (*RoutingDecision, error) {
	sr.mu.RLock()
	defer sr.mu.RUnlock()

	// Get available routes for the symbol
	routes := sr.getAvailableRoutes(order.Symbol)
	if len(routes) == 0 {
		return nil, fmt.Errorf("no available routes for symbol %s", order.Symbol)
	}

	// Score each route based on routing criteria
	scoredRoutes := make([]*ScoredRoute, 0, len(routes))
	for _, route := range routes {
		score, err := sr.scoreRoute(order, route)
		if err != nil {
			continue // Skip routes with errors
		}
		scoredRoutes = append(scoredRoutes, score)
	}

	if len(scoredRoutes) == 0 {
		return nil, fmt.Errorf("no valid routes found")
	}

	// Sort by score (highest first)
	sort.Slice(scoredRoutes, func(i, j int) bool {
		return scoredRoutes[i].Score > scoredRoutes[j].Score
	})

	bestRoute := scoredRoutes[0]
	if bestRoute.Score < sr.config.MinConfidence {
		return nil, fmt.Errorf("no route meets minimum confidence threshold")
	}

	return &RoutingDecision{
		OrderID:          order.ClientID,
		Exchange:         bestRoute.Exchange,
		Route:            bestRoute.Route,
		Reason:           bestRoute.Reason,
		ExpectedSlippage: bestRoute.ExpectedSlippage,
		ExpectedFee:      bestRoute.ExpectedFee,
		Confidence:       bestRoute.Score,
		Timestamp:        time.Now(),
	}, nil
}

// GetBestPrice finds the best price for a given symbol and side
func (sr *SmartRouterImpl) GetBestPrice(ctx context.Context, symbol string, side OrderSide, quantity decimal.Decimal) (*RoutingDecision, error) {
	sr.mu.RLock()
	defer sr.mu.RUnlock()

	bestRoute := &RoutingDecision{
		Symbol:    symbol,
		Side:      side,
		Timestamp: time.Now(),
	}

	var bestPrice decimal.Decimal
	var bestExchange string

	for exchange, data := range sr.marketData[symbol] {
		if data == nil {
			continue
		}

		var price decimal.Decimal
		var volume decimal.Decimal

		switch side {
		case OrderSideBuy:
			price = data.AskPrice
			volume = data.AskVolume
		case OrderSideSell:
			price = data.BidPrice
			volume = data.BidVolume
		}

		if volume.LessThan(quantity) {
			continue // Skip if not enough volume
		}

		if bestPrice.IsZero() || (side == OrderSideBuy && price.LessThan(bestPrice)) ||
			(side == OrderSideSell && price.GreaterThan(bestPrice)) {
			bestPrice = price
			bestExchange = exchange
		}
	}

	if bestExchange == "" {
		return nil, fmt.Errorf("no suitable exchange found")
	}

	bestRoute.Exchange = bestExchange
	bestRoute.Reason = "best_price"
	bestRoute.Confidence = 1.0

	return bestRoute, nil
}

// UpdateMarketData updates market data for an exchange
func (sr *SmartRouterImpl) UpdateMarketData(exchange string, data interface{}) {
	sr.mu.Lock()
	defer sr.mu.Unlock()

	marketData, ok := data.(*MarketData)
	if !ok {
		return
	}

	if sr.marketData[marketData.Symbol] == nil {
		sr.marketData[marketData.Symbol] = make(map[string]*MarketData)
	}

	sr.marketData[marketData.Symbol][exchange] = marketData
	sr.lastUpdate = time.Now()
}

// ScoredRoute represents a route with a score
type ScoredRoute struct {
	Exchange        string
	Route           string
	Score           float64
	Reason          string
	ExpectedSlippage decimal.Decimal
	ExpectedFee      decimal.Decimal
}

// scoreRoute calculates a score for a route based on various factors
func (sr *SmartRouterImpl) scoreRoute(order *OrderRequest, route ExchangeRoute) (*ScoredRoute, error) {
	marketData, exists := sr.marketData[order.Symbol][route.Exchange]
	if !exists {
		return nil, fmt.Errorf("no market data for %s on %s", order.Symbol, route.Exchange)
	}

	// Calculate price impact
	priceImpact := sr.calculatePriceImpact(order, marketData)
	if priceImpact.GreaterThan(sr.config.MaxSlippage) {
		return nil, fmt.Errorf("price impact exceeds maximum allowed")
	}

	// Calculate expected fee
	expectedFee := marketData.FeeRate.Mul(order.Quantity)
	if expectedFee.GreaterThan(sr.config.MaxFee) {
		return nil, fmt.Errorf("expected fee exceeds maximum allowed")
	}

	// Calculate score components
	priceScore := sr.calculatePriceScore(order, marketData)
	volumeScore := sr.calculateVolumeScore(order, marketData)
	latencyScore := sr.calculateLatencyScore(marketData.Latency)
	feeScore := sr.calculateFeeScore(marketData.FeeRate)

	// Calculate weighted score
	score := (priceScore*sr.config.PriceWeight +
		volumeScore*sr.config.VolumeWeight +
		latencyScore*sr.config.LatencyWeight +
		feeScore*sr.config.FeeWeight)

	return &ScoredRoute{
		Exchange:         route.Exchange,
		Route:            route.Route,
		Score:            score,
		Reason:           "optimal_combination",
		ExpectedSlippage: priceImpact,
		ExpectedFee:      expectedFee,
	}, nil
}

// calculatePriceImpact calculates the expected price impact for an order
func (sr *SmartRouterImpl) calculatePriceImpact(order *OrderRequest, marketData *MarketData) decimal.Decimal {
	orderBook := marketData.OrderBook
	if orderBook == nil {
		return decimal.NewFromFloat(0.001) // Default small impact
	}

	var targetPrice decimal.Decimal
	var levels []normalizer.PriceLevel

	switch order.Side {
	case OrderSideBuy:
		targetPrice = marketData.AskPrice
		levels = orderBook.Asks
	case OrderSideSell:
		targetPrice = marketData.BidPrice
		levels = orderBook.Bids
	}

	if len(levels) == 0 {
		return decimal.NewFromFloat(0.001)
	}

	// Calculate volume-weighted average price
	remainingQty := order.Quantity
	totalCost := decimal.Zero
	volume := decimal.Zero

	for _, level := range levels {
		if remainingQty.LessThanOrEqual(decimal.Zero) {
			break
		}

		levelVolume := decimal.Min(remainingQty, decimal.NewFromFloat(level.Volume))
		totalCost = totalCost.Add(decimal.NewFromFloat(level.Price).Mul(levelVolume))
		volume = volume.Add(levelVolume)
		remainingQty = remainingQty.Sub(levelVolume)
	}

	if volume.IsZero() {
		return decimal.NewFromFloat(0.001)
	}

	vwap := totalCost.Div(volume)
	impact := vwap.Sub(targetPrice).Div(targetPrice).Abs()

	return impact
}

// calculatePriceScore calculates a price score (0-1)
func (sr *SmartRouterImpl) calculatePriceScore(order *OrderRequest, marketData *MarketData) float64 {
	var price decimal.Decimal
	switch order.Side {
	case OrderSideBuy:
		price = marketData.AskPrice
	case OrderSideSell:
		price = marketData.BidPrice
	}

	// Score based on how close the price is to the best available
	// This is a simplified scoring - in reality, you'd compare against other exchanges
	_ = price // Use price in future implementation
	return 1.0 // For now, return perfect score
}

// calculateVolumeScore calculates a volume score (0-1)
func (sr *SmartRouterImpl) calculateVolumeScore(order *OrderRequest, marketData *MarketData) float64 {
	var availableVolume decimal.Decimal
	switch order.Side {
	case OrderSideBuy:
		availableVolume = marketData.AskVolume
	case OrderSideSell:
		availableVolume = marketData.BidVolume
	}

	if availableVolume.IsZero() {
		return 0.0
	}

	// Score based on available liquidity relative to order size
	ratio := availableVolume.Div(order.Quantity)
	if ratio.GreaterThanOrEqual(decimal.NewFromInt(10)) {
		return 1.0
	}

	score, _ := ratio.Float64()
	return min(score, 1.0)
}

// calculateLatencyScore calculates a latency score (0-1)
func (sr *SmartRouterImpl) calculateLatencyScore(latency time.Duration) float64 {
	// Score decreases with latency
	// 0ms = 1.0, 100ms = 0.8, 500ms = 0.0
	latencyMs := float64(latency.Milliseconds())
	score := 1.0 - (latencyMs / 500.0)
	return max(0.0, min(score, 1.0))
}

// calculateFeeScore calculates a fee score (0-1)
func (sr *SmartRouterImpl) calculateFeeScore(feeRate decimal.Decimal) float64 {
	// Score based on fee rate (lower is better)
	feePercent, _ := feeRate.Float64()
	score := 1.0 - (feePercent * 100.0) // 0.1% = 0.9, 1% = 0.0
	return max(0.0, min(score, 1.0))
}

// getAvailableRoutes returns all available routes for a symbol
func (sr *SmartRouterImpl) getAvailableRoutes(symbol string) []ExchangeRoute {
	routes := []ExchangeRoute{
		{Exchange: "binance", Route: "spot", Priority: 1, Active: true},
		{Exchange: "coinbase", Route: "spot", Priority: 2, Active: true},
		{Exchange: "kraken", Route: "spot", Priority: 3, Active: true},
	}

	return routes
}

// GetMarketData returns current market data for a symbol
func (sr *SmartRouterImpl) GetMarketData(symbol string) map[string]*MarketData {
	sr.mu.RLock()
	defer sr.mu.RUnlock()

	data := make(map[string]*MarketData)
	if sr.marketData[symbol] != nil {
		for exchange, marketData := range sr.marketData[symbol] {
			data[exchange] = marketData
		}
	}
	return data
}

// GetLastUpdate returns the last market data update time
func (sr *SmartRouterImpl) GetLastUpdate() time.Time {
	sr.mu.RLock()
	defer sr.mu.RUnlock()
	return sr.lastUpdate
}

// Helper functions
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}