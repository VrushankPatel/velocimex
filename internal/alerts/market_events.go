package alerts

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"velocimex/internal/logger"
)

// MarketEventAlertSystem handles market-specific alerts
type MarketEventAlertSystem struct {
	engine        *AlertEngine
	marketRules   map[string][]*MarketAlertRule
	priceAlerts   map[string]*PriceAlert
	volumeAlerts  map[string]*VolumeAlert
	volatilityAlerts map[string]*VolatilityAlert
	arbitrageAlerts map[string]*ArbitrageAlert
	
	mu            sync.RWMutex
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
	logger        logger.Logger
}

// MarketAlertRule defines a market-specific alert rule
type MarketAlertRule struct {
	ID          string                 `json:"id"`
	Symbol      string                 `json:"symbol"`
	Exchange    string                 `json:"exchange"`
	Type        MarketAlertType        `json:"type"`
	Condition   MarketCondition        `json:"condition"`
	Threshold   float64                `json:"threshold"`
	Timeframe   time.Duration          `json:"timeframe"`
	Enabled     bool                   `json:"enabled"`
	Channels    []string               `json:"channels"`
	Metadata    map[string]interface{} `json:"metadata"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// MarketAlertType represents different types of market alerts
type MarketAlertType string

const (
	MarketAlertPrice     MarketAlertType = "price"
	MarketAlertVolume    MarketAlertType = "volume"
	MarketAlertVolatility MarketAlertType = "volatility"
	MarketAlertArbitrage MarketAlertType = "arbitrage"
	MarketAlertSpread    MarketAlertType = "spread"
	MarketAlertLiquidity MarketAlertType = "liquidity"
)

// MarketCondition represents the condition for a market alert
type MarketCondition struct {
	Operator string  `json:"operator"` // "above", "below", "crosses_above", "crosses_below"
	Value    float64 `json:"value"`
	Percent  bool    `json:"percent"` // Whether to use percentage change
}

// PriceAlert tracks price-based alerts
type PriceAlert struct {
	Symbol      string    `json:"symbol"`
	Exchange    string    `json:"exchange"`
	CurrentPrice float64  `json:"current_price"`
	PreviousPrice float64 `json:"previous_price"`
	Threshold   float64   `json:"threshold"`
	Condition   MarketCondition `json:"condition"`
	LastCheck   time.Time `json:"last_check"`
	Triggered   bool      `json:"triggered"`
}

// VolumeAlert tracks volume-based alerts
type VolumeAlert struct {
	Symbol      string    `json:"symbol"`
	Exchange    string    `json:"exchange"`
	CurrentVolume float64 `json:"current_volume"`
	AverageVolume float64 `json:"average_volume"`
	Threshold   float64   `json:"threshold"`
	Condition   MarketCondition `json:"condition"`
	LastCheck   time.Time `json:"last_check"`
	Triggered   bool      `json:"triggered"`
}

// VolatilityAlert tracks volatility-based alerts
type VolatilityAlert struct {
	Symbol      string    `json:"symbol"`
	Exchange    string    `json:"exchange"`
	CurrentVolatility float64 `json:"current_volatility"`
	Threshold   float64   `json:"threshold"`
	Condition   MarketCondition `json:"condition"`
	LastCheck   time.Time `json:"last_check"`
	Triggered   bool      `json:"triggered"`
}

// ArbitrageAlert tracks arbitrage opportunity alerts
type ArbitrageAlert struct {
	Symbol        string    `json:"symbol"`
	BuyExchange   string    `json:"buy_exchange"`
	SellExchange  string    `json:"sell_exchange"`
	ProfitPercent float64   `json:"profit_percent"`
	Threshold     float64   `json:"threshold"`
	LastCheck     time.Time `json:"last_check"`
	Triggered     bool      `json:"triggered"`
}

// NewMarketEventAlertSystem creates a new market event alert system
func NewMarketEventAlertSystem(engine *AlertEngine, logger logger.Logger) *MarketEventAlertSystem {
	ctx, cancel := context.WithCancel(context.Background())
	
	mas := &MarketEventAlertSystem{
		engine:        engine,
		marketRules:   make(map[string][]*MarketAlertRule),
		priceAlerts:   make(map[string]*PriceAlert),
		volumeAlerts:  make(map[string]*VolumeAlert),
		volatilityAlerts: make(map[string]*VolatilityAlert),
		arbitrageAlerts: make(map[string]*ArbitrageAlert),
		ctx:           ctx,
		cancel:        cancel,
		logger:        logger,
	}

	// Start monitoring workers
	mas.wg.Add(1)
	go mas.priceMonitor()
	mas.wg.Add(1)
	go mas.volumeMonitor()
	mas.wg.Add(1)
	go mas.volatilityMonitor()
	mas.wg.Add(1)
	go mas.arbitrageMonitor()

	return mas
}

// AddMarketRule adds a market-specific alert rule
func (mas *MarketEventAlertSystem) AddMarketRule(rule *MarketAlertRule) error {
	mas.mu.Lock()
	defer mas.mu.Unlock()

	// Validate rule
	if err := mas.validateMarketRule(rule); err != nil {
		return fmt.Errorf("invalid market rule: %w", err)
	}

	// Set rule ID if not set
	if rule.ID == "" {
		rule.ID = fmt.Sprintf("%s_%s_%s_%d", rule.Symbol, rule.Exchange, rule.Type, time.Now().UnixNano())
	}

	// Set timestamps
	now := time.Now()
	if rule.CreatedAt.IsZero() {
		rule.CreatedAt = now
	}
	rule.UpdatedAt = now

	// Add to market rules
	key := fmt.Sprintf("%s_%s", rule.Symbol, rule.Exchange)
	mas.marketRules[key] = append(mas.marketRules[key], rule)

	// Create specific alert based on type
	switch rule.Type {
	case MarketAlertPrice:
		mas.priceAlerts[rule.ID] = &PriceAlert{
			Symbol:    rule.Symbol,
			Exchange:  rule.Exchange,
			Threshold: rule.Threshold,
			Condition: rule.Condition,
			LastCheck: now,
		}
	case MarketAlertVolume:
		mas.volumeAlerts[rule.ID] = &VolumeAlert{
			Symbol:    rule.Symbol,
			Exchange:  rule.Exchange,
			Threshold: rule.Threshold,
			Condition: rule.Condition,
			LastCheck: now,
		}
	case MarketAlertVolatility:
		mas.volatilityAlerts[rule.ID] = &VolatilityAlert{
			Symbol:    rule.Symbol,
			Exchange:  rule.Exchange,
			Threshold: rule.Threshold,
			Condition: rule.Condition,
			LastCheck: now,
		}
	case MarketAlertArbitrage:
		mas.arbitrageAlerts[rule.ID] = &ArbitrageAlert{
			Symbol:    rule.Symbol,
			Threshold: rule.Threshold,
			LastCheck: now,
		}
	}

	mas.logger.Info("alerts", fmt.Sprintf("Added market rule: %s", rule.ID), map[string]interface{}{
		"symbol":   rule.Symbol,
		"exchange": rule.Exchange,
		"type":     rule.Type,
	})

	return nil
}

// ProcessMarketData processes market data and checks for alerts
func (mas *MarketEventAlertSystem) ProcessMarketData(symbol, exchange string, data map[string]interface{}) {
	mas.mu.RLock()
	key := fmt.Sprintf("%s_%s", symbol, exchange)
	rules, exists := mas.marketRules[key]
	mas.mu.RUnlock()

	if !exists {
		return
	}

	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}

		switch rule.Type {
		case MarketAlertPrice:
			mas.checkPriceAlert(rule, data)
		case MarketAlertVolume:
			mas.checkVolumeAlert(rule, data)
		case MarketAlertVolatility:
			mas.checkVolatilityAlert(rule, data)
		}
	}
}

// ProcessArbitrageData processes arbitrage data and checks for alerts
func (mas *MarketEventAlertSystem) ProcessArbitrageData(arbitrageData []map[string]interface{}) {
	mas.mu.RLock()
	defer mas.mu.RUnlock()

	for _, data := range arbitrageData {
		symbol, ok := data["symbol"].(string)
		if !ok {
			continue
		}

		profitPercent, ok := data["profit_percent"].(float64)
		if !ok {
			continue
		}

		for _, alert := range mas.arbitrageAlerts {
			if alert.Symbol == symbol && !alert.Triggered && profitPercent >= alert.Threshold {
				// Create alert event
				event := &AlertEvent{
					ID:        uuid.New().String(),
					Type:      string(MarketAlertArbitrage),
					Severity:  SeverityHigh,
					Source:    "arbitrage",
					Message:   fmt.Sprintf("Arbitrage opportunity: %.4f%% profit for %s between %s and %s",
						profitPercent, alert.Symbol, data["buy_exchange"], data["sell_exchange"]),
					Metadata:  nil,
					Timestamp: time.Now(),
					Data:      data,
				}

				// Send to alert engine
				mas.engine.ProcessEvent(event)

				// Log the alert
				mas.logger.Info("market_alert", "Triggered arbitrage alert",
					map[string]interface{}{
						"symbol":         alert.Symbol,
						"buy_exchange":   data["buy_exchange"],
						"sell_exchange":  data["sell_exchange"],
						"profit_percent": profitPercent,
					})

				// Update alert state
				mas.mu.Lock()
				alert.Triggered = true
				mas.mu.Unlock()
			}
		}
	}
}

// checkPriceAlert checks if a price alert should be triggered
func (mas *MarketEventAlertSystem) checkPriceAlert(rule *MarketAlertRule, data map[string]interface{}) {
	price, ok := data["price"].(float64)
	if !ok {
		return
	}

	mas.mu.Lock()
	alert, exists := mas.priceAlerts[rule.ID]
	if !exists {
		mas.mu.Unlock()
		return
	}

	alert.PreviousPrice = alert.CurrentPrice
	alert.CurrentPrice = price
	alert.LastCheck = time.Now()
	mas.mu.Unlock()

	// Check condition
	if mas.evaluatePriceCondition(alert) {
		// Create alert event
		event := &AlertEvent{
			ID:        uuid.New().String(),
			Type:      string(MarketAlertPrice),
			Severity:  SeverityMedium,
			Source:    "market",
			Message:   fmt.Sprintf("Price alert for %s/%s: %.8f %s %.8f", 
				alert.Symbol, alert.Exchange, 
				alert.CurrentPrice, rule.Condition.Operator, rule.Threshold),
			Metadata:  rule.Metadata,
			Timestamp: time.Now(),
			Data:      data,
		}

		// Send to alert engine
		mas.engine.ProcessEvent(event)

		// Log the alert
		mas.logger.Info("market_alert", "Triggered price alert", 
			map[string]interface{}{
				"symbol":    alert.Symbol,
				"exchange":  alert.Exchange,
				"price":     alert.CurrentPrice,
				"threshold": rule.Threshold,
			})

		// Update alert state
		mas.mu.Lock()
		alert.Triggered = true
		mas.mu.Unlock()
	}
}

// checkVolumeAlert checks if a volume alert should be triggered
func (mas *MarketEventAlertSystem) checkVolumeAlert(rule *MarketAlertRule, data map[string]interface{}) {
	volume, ok := data["volume"].(float64)
	if !ok {
		return
	}

	mas.mu.Lock()
	alert, exists := mas.volumeAlerts[rule.ID]
	if !exists {
		mas.mu.Unlock()
		return
	}

	alert.CurrentVolume = volume
	alert.LastCheck = time.Now()
	mas.mu.Unlock()

	// Check condition
	if mas.evaluateVolumeCondition(alert) {
		// Create alert event
		event := &AlertEvent{
			ID:        uuid.New().String(),
			Type:      string(MarketAlertVolume),
			Severity:  SeverityMedium,
			Source:    "market",
			Message:   fmt.Sprintf("Volume alert for %s/%s: %.2f %s %.2f (avg: %.2f)",
				alert.Symbol, alert.Exchange,
				alert.CurrentVolume, rule.Condition.Operator, rule.Threshold, alert.AverageVolume),
			Metadata:  rule.Metadata,
			Timestamp: time.Now(),
			Data:      data,
		}

		// Send to alert engine
		mas.engine.ProcessEvent(event)

		// Log the alert
		mas.logger.Info("market_alert", "Triggered volume alert",
			map[string]interface{}{
				"symbol":         alert.Symbol,
				"exchange":       alert.Exchange,
				"volume":         alert.CurrentVolume,
				"average_volume": alert.AverageVolume,
				"threshold":      rule.Threshold,
			})

		// Update alert state
		mas.mu.Lock()
		alert.Triggered = true
		mas.mu.Unlock()
	}
}

// checkVolatilityAlert checks if a volatility alert should be triggered
func (mas *MarketEventAlertSystem) checkVolatilityAlert(rule *MarketAlertRule, data map[string]interface{}) {
	volatility, ok := data["volatility"].(float64)
	if !ok {
		return
	}

	mas.mu.Lock()
	alert, exists := mas.volatilityAlerts[rule.ID]
	if !exists {
		mas.mu.Unlock()
		return
	}

	alert.CurrentVolatility = volatility
	alert.LastCheck = time.Now()
	mas.mu.Unlock()

	// Check condition
	if mas.evaluateVolatilityCondition(alert) {
		// Create alert event
		event := &AlertEvent{
			ID:        uuid.New().String(),
			Type:      string(MarketAlertVolatility),
			Severity:  SeverityHigh,
			Source:    "market",
			Message:   fmt.Sprintf("Volatility alert for %s/%s: %.4f %s %.4f",
				alert.Symbol, alert.Exchange,
				alert.CurrentVolatility, rule.Condition.Operator, rule.Threshold),
			Metadata:  rule.Metadata,
			Timestamp: time.Now(),
			Data:      data,
		}

		// Send to alert engine
		mas.engine.ProcessEvent(event)

		// Log the alert
		mas.logger.Info("market_alert", "Triggered volatility alert",
			map[string]interface{}{
				"symbol":     alert.Symbol,
				"exchange":   alert.Exchange,
				"volatility": alert.CurrentVolatility,
				"threshold":  rule.Threshold,
			})

		// Update alert state
		mas.mu.Lock()
		alert.Triggered = true
		mas.mu.Unlock()
	}
}

// evaluatePriceCondition evaluates a price alert condition
func (mas *MarketEventAlertSystem) evaluatePriceCondition(alert *PriceAlert) bool {
	switch alert.Condition.Operator {
	case "above":
		return alert.CurrentPrice > alert.Threshold
	case "below":
		return alert.CurrentPrice < alert.Threshold
	case "crosses_above":
		return alert.PreviousPrice <= alert.Threshold && alert.CurrentPrice > alert.Threshold
	case "crosses_below":
		return alert.PreviousPrice >= alert.Threshold && alert.CurrentPrice < alert.Threshold
	default:
		return false
	}
}

// evaluateVolumeCondition evaluates a volume alert condition
func (mas *MarketEventAlertSystem) evaluateVolumeCondition(alert *VolumeAlert) bool {
	switch alert.Condition.Operator {
	case "above":
		return alert.CurrentVolume > alert.Threshold
	case "below":
		return alert.CurrentVolume < alert.Threshold
	case "crosses_above":
		return alert.AverageVolume <= alert.Threshold && alert.CurrentVolume > alert.Threshold
	case "crosses_below":
		return alert.AverageVolume >= alert.Threshold && alert.CurrentVolume < alert.Threshold
	default:
		return false
	}
}

// evaluateVolatilityCondition evaluates a volatility alert condition
func (mas *MarketEventAlertSystem) evaluateVolatilityCondition(alert *VolatilityAlert) bool {
	switch alert.Condition.Operator {
	case "above":
		return alert.CurrentVolatility > alert.Threshold
	case "below":
		return alert.CurrentVolatility < alert.Threshold
	}
	return false
}

// triggerVolumeAlert triggers a volume alert
func (mas *MarketEventAlertSystem) triggerVolumeAlert(rule *MarketAlertRule, alert *VolumeAlert, data map[string]interface{}) {
	mas.mu.Lock()
	alert.Triggered = true
	mas.mu.Unlock()

	// Create alert event
	event := &AlertEvent{
		ID:        uuid.New().String(),
		Type:      string(MarketAlertVolume),
		Severity:  SeverityMedium, // Use the correct severity constant
		Source:    "market",
		Message:   fmt.Sprintf("Volume alert for %s/%s: %.2f %s %.2f (avg: %.2f)", 
			alert.Symbol, alert.Exchange, 
			alert.CurrentVolume, rule.Condition.Operator, rule.Threshold, alert.AverageVolume),
		Metadata:  rule.Metadata,
		Timestamp: time.Now(),
		Data:      data,
	}

	// Send to alert engine
	mas.engine.ProcessEvent(event)

	mas.logger.Info("market_alert", "Triggered volume alert", 
		map[string]interface{}{
			"symbol":         alert.Symbol,
			"exchange":       alert.Exchange,
			"volume":         alert.CurrentVolume,
			"average_volume": alert.AverageVolume,
			"threshold":      rule.Threshold,
		})
}

// triggerVolatilityAlert triggers a volatility alert
func (mas *MarketEventAlertSystem) triggerVolatilityAlert(rule *MarketAlertRule, alert *VolatilityAlert, data map[string]interface{}) {
	mas.mu.Lock()
	alert.Triggered = true
	mas.mu.Unlock()

	// Create alert event
	event := &AlertEvent{
		ID:        uuid.New().String(),
		Type:      string(MarketAlertVolatility),
		Severity:  SeverityHigh, // Use the correct severity constant
		Source:    "market",
		Message:   fmt.Sprintf("Volatility alert for %s/%s: %.2f %s %.2f", 
			alert.Symbol, alert.Exchange, 
			alert.CurrentVolatility, rule.Condition.Operator, rule.Threshold),
		Metadata:  rule.Metadata,
		Timestamp: time.Now(),
		Data:      data,
	}

	// Process event
	mas.engine.ProcessEvent(event)

	mas.logger.Info("market_alert", "Triggered volatility alert", 
		map[string]interface{}{
			"symbol":     alert.Symbol,
			"exchange":   alert.Exchange,
			"volatility": alert.CurrentVolatility,
			"threshold":  rule.Threshold,
		})
}

// triggerArbitrageAlert triggers an arbitrage alert
func (mas *MarketEventAlertSystem) triggerArbitrageAlert(alert *ArbitrageAlert, data map[string]interface{}) {
	mas.mu.Lock()
	alert.Triggered = true
	mas.mu.Unlock()

	// Create alert event
	event := &AlertEvent{
		ID:        uuid.New().String(),
		Type:      string(MarketAlertArbitrage),
		Severity:  SeverityHigh, // Use the correct severity constant
		Source:    "arbitrage",
		Message:   fmt.Sprintf("Arbitrage opportunity: %.4f%% profit for %s between %s and %s", 
			data["profit_percent"].(float64), alert.Symbol, data["buy_exchange"], data["sell_exchange"]),
		Metadata:  nil,
		Timestamp: time.Now(),
		Data:      data,
	}

	// Send to alert engine
	mas.engine.ProcessEvent(event)
}

// Monitor workers

func (mas *MarketEventAlertSystem) priceMonitor() {
	defer mas.wg.Done()
	
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			mas.checkPriceAlerts()
		case <-mas.ctx.Done():
			return
		}
	}
}

func (mas *MarketEventAlertSystem) volumeMonitor() {
	defer mas.wg.Done()
	
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			mas.checkVolumeAlerts()
		case <-mas.ctx.Done():
			return
		}
	}
}

func (mas *MarketEventAlertSystem) volatilityMonitor() {
	defer mas.wg.Done()
	
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			mas.checkVolatilityAlerts()
		case <-mas.ctx.Done():
			return
		}
	}
}

func (mas *MarketEventAlertSystem) arbitrageMonitor() {
	defer mas.wg.Done()
	
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			mas.checkArbitrageAlerts()
		case <-mas.ctx.Done():
			return
		}
	}
}

// Check methods

func (mas *MarketEventAlertSystem) checkPriceAlerts() {
	// Implementation would check price alerts
}

func (mas *MarketEventAlertSystem) checkVolumeAlerts() {
	// Implementation would check volume alerts
}

func (mas *MarketEventAlertSystem) checkVolatilityAlerts() {
	// Implementation would check volatility alerts
}

func (mas *MarketEventAlertSystem) checkArbitrageAlerts() {
	// Implementation would check arbitrage alerts
}

// Helper methods

func (mas *MarketEventAlertSystem) validateMarketRule(rule *MarketAlertRule) error {
	if rule.Symbol == "" {
		return fmt.Errorf("symbol is required")
	}
	if rule.Exchange == "" {
		return fmt.Errorf("exchange is required")
	}
	if rule.Type == "" {
		return fmt.Errorf("type is required")
	}
	if rule.Threshold <= 0 {
		return fmt.Errorf("threshold must be positive")
	}
	if rule.Condition.Operator == "" {
		return fmt.Errorf("condition operator is required")
	}
	return nil
}

// Close shuts down the market event alert system
func (mas *MarketEventAlertSystem) Close() error {
	mas.cancel()
	mas.wg.Wait()
	return nil
}
