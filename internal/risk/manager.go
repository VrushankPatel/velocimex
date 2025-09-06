package risk

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"velocimex/internal/metrics"
)

// Manager implements the RiskManager interface
type Manager struct {
	config        RiskConfig
	portfolio     *Portfolio
	riskMetrics   *RiskMetrics
	riskEvents    []*RiskEvent
	eventCallbacks []func(*RiskEvent)
	metrics       *metrics.Wrapper
	running       bool
	mu            sync.RWMutex
	ctx           context.Context
	cancel        context.CancelFunc
}

// NewManager creates a new risk manager
func NewManager(config RiskConfig, metrics *metrics.Wrapper) *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	return &Manager{
		config:      config,
		portfolio:   &Portfolio{Positions: make(map[string]*Position)},
		riskMetrics: &RiskMetrics{},
		riskEvents:  make([]*RiskEvent, 0),
		eventCallbacks: make([]func(*RiskEvent), 0),
		metrics:     metrics,
		ctx:         ctx,
		cancel:      cancel,
	}
}

// SetConfig sets the risk management configuration
func (rm *Manager) SetConfig(config RiskConfig) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	
	rm.config = config
	return nil
}

// GetConfig returns the current configuration
func (rm *Manager) GetConfig() RiskConfig {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	return rm.config
}

// UpdatePortfolio updates the portfolio state
func (rm *Manager) UpdatePortfolio(portfolio *Portfolio) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	
	rm.portfolio = portfolio
	rm.portfolio.LastUpdated = time.Now()
	
	// Update risk metrics
	rm.calculateRiskMetrics()
	
	// Check for risk events
	go rm.checkPortfolioRisk()
	
	return nil
}

// GetPortfolio returns the current portfolio
func (rm *Manager) GetPortfolio() *Portfolio {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	return rm.portfolio
}

// GetRiskMetrics returns the current risk metrics
func (rm *Manager) GetRiskMetrics() *RiskMetrics {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	return rm.riskMetrics
}

// AddPosition adds a new position to the portfolio
func (rm *Manager) AddPosition(position *Position) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	
	key := fmt.Sprintf("%s:%s", position.Exchange, position.Symbol)
	rm.portfolio.Positions[key] = position
	
	// Update portfolio value
	rm.updatePortfolioValue()
	
	// Check position risk
	go rm.checkPositionRisk(position.Symbol, position.Exchange)
	
	return nil
}

// UpdatePosition updates an existing position
func (rm *Manager) UpdatePosition(symbol, exchange string, price decimal.Decimal) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	
	key := fmt.Sprintf("%s:%s", exchange, symbol)
	position, exists := rm.portfolio.Positions[key]
	if !exists {
		return fmt.Errorf("position not found: %s", key)
	}
	
	position.CurrentPrice = price
	position.MarketValue = position.Quantity.Mul(price)
	position.UnrealizedPNL = position.MarketValue.Sub(position.Quantity.Mul(position.EntryPrice))
	position.UpdatedAt = time.Now()
	
	// Update portfolio value
	rm.updatePortfolioValue()
	
	return nil
}

// RemovePosition removes a position from the portfolio
func (rm *Manager) RemovePosition(symbol, exchange string) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	
	key := fmt.Sprintf("%s:%s", exchange, symbol)
	delete(rm.portfolio.Positions, key)
	
	// Update portfolio value
	rm.updatePortfolioValue()
	
	return nil
}

// GetPositions returns all positions
func (rm *Manager) GetPositions() map[string]*Position {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	return rm.portfolio.Positions
}

// CheckOrderRisk checks if an order meets risk requirements
func (rm *Manager) CheckOrderRisk(symbol, exchange, side string, quantity, price decimal.Decimal) (*RiskEvent, error) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	
	orderValue := quantity.Mul(price)
	
	// Check position size limit
	if orderValue.GreaterThan(rm.config.AlertThresholds.MaxPositionSize) {
		return &RiskEvent{
			ID:        uuid.New().String(),
			Type:      "POSITION_SIZE_EXCEEDED",
			Severity:  RiskLevelHigh,
			Message:   fmt.Sprintf("Order value %s exceeds maximum position size %s", orderValue.String(), rm.config.AlertThresholds.MaxPositionSize.String()),
			Symbol:    symbol,
			Exchange:  exchange,
			Value:     orderValue,
			Threshold: rm.config.AlertThresholds.MaxPositionSize,
			Timestamp: time.Now(),
		}, nil
	}
	
	// Check portfolio value limit
	newPortfolioValue := rm.portfolio.TotalValue.Add(orderValue)
	if newPortfolioValue.GreaterThan(rm.config.AlertThresholds.MaxPortfolioValue) {
		return &RiskEvent{
			ID:        uuid.New().String(),
			Type:      "PORTFOLIO_VALUE_EXCEEDED",
			Severity:  RiskLevelHigh,
			Message:   fmt.Sprintf("Order would exceed maximum portfolio value %s", rm.config.AlertThresholds.MaxPortfolioValue.String()),
			Symbol:    symbol,
			Exchange:  exchange,
			Value:     newPortfolioValue,
			Threshold: rm.config.AlertThresholds.MaxPortfolioValue,
			Timestamp: time.Now(),
		}, nil
	}
	
	// Check concentration risk
	positionKey := fmt.Sprintf("%s:%s", exchange, symbol)
	existingPosition, exists := rm.portfolio.Positions[positionKey]
	var totalPositionValue decimal.Decimal
	if exists {
		totalPositionValue = existingPosition.MarketValue.Add(orderValue)
	} else {
		totalPositionValue = orderValue
	}
	
	concentrationRatio := totalPositionValue.Div(rm.portfolio.TotalValue)
	if concentrationRatio.GreaterThan(rm.config.AlertThresholds.MaxConcentration) {
		return &RiskEvent{
			ID:        uuid.New().String(),
			Type:      "CONCENTRATION_RISK",
			Severity:  RiskLevelMedium,
			Message:   fmt.Sprintf("Position concentration %s exceeds maximum %s", concentrationRatio.String(), rm.config.AlertThresholds.MaxConcentration.String()),
			Symbol:    symbol,
			Exchange:  exchange,
			Value:     concentrationRatio,
			Threshold: rm.config.AlertThresholds.MaxConcentration,
			Timestamp: time.Now(),
		}, nil
	}
	
	return nil, nil
}

// CheckPortfolioRisk checks the overall portfolio for risk events
func (rm *Manager) CheckPortfolioRisk() ([]*RiskEvent, error) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	
	var events []*RiskEvent
	
	// Check daily loss limit
	if rm.portfolio.DailyPNL.LessThan(rm.config.AlertThresholds.MaxDailyLoss.Neg()) {
		events = append(events, &RiskEvent{
			ID:        uuid.New().String(),
			Type:      "DAILY_LOSS_EXCEEDED",
			Severity:  RiskLevelCritical,
			Message:   fmt.Sprintf("Daily loss %s exceeds maximum %s", rm.portfolio.DailyPNL.String(), rm.config.AlertThresholds.MaxDailyLoss.String()),
			Value:     rm.portfolio.DailyPNL,
			Threshold: rm.config.AlertThresholds.MaxDailyLoss.Neg(),
			Timestamp: time.Now(),
		})
	}
	
	// Check drawdown
	if rm.riskMetrics.MaxDrawdown.GreaterThan(rm.config.AlertThresholds.MaxDrawdown) {
		events = append(events, &RiskEvent{
			ID:        uuid.New().String(),
			Type:      "DRAWDOWN_EXCEEDED",
			Severity:  RiskLevelHigh,
			Message:   fmt.Sprintf("Maximum drawdown %s exceeds limit %s", rm.riskMetrics.MaxDrawdown.String(), rm.config.AlertThresholds.MaxDrawdown.String()),
			Value:     rm.riskMetrics.MaxDrawdown,
			Threshold: rm.config.AlertThresholds.MaxDrawdown,
			Timestamp: time.Now(),
		})
	}
	
	// Check leverage
	if rm.riskMetrics.Leverage.GreaterThan(rm.config.AlertThresholds.MaxLeverage) {
		events = append(events, &RiskEvent{
			ID:        uuid.New().String(),
			Type:      "LEVERAGE_EXCEEDED",
			Severity:  RiskLevelHigh,
			Message:   fmt.Sprintf("Leverage %s exceeds maximum %s", rm.riskMetrics.Leverage.String(), rm.config.AlertThresholds.MaxLeverage.String()),
			Value:     rm.riskMetrics.Leverage,
			Threshold: rm.config.AlertThresholds.MaxLeverage,
			Timestamp: time.Now(),
		})
	}
	
	return events, nil
}

// CheckPositionRisk checks a specific position for risk events
func (rm *Manager) CheckPositionRisk(symbol, exchange string) (*RiskEvent, error) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	
	key := fmt.Sprintf("%s:%s", exchange, symbol)
	position, exists := rm.portfolio.Positions[key]
	if !exists {
		return nil, fmt.Errorf("position not found: %s", key)
	}
	
	// Check stop loss
	stopLossPrice := position.EntryPrice.Mul(decimal.NewFromFloat(1).Sub(rm.config.AlertThresholds.StopLossPercentage))
	if position.Side == "LONG" && position.CurrentPrice.LessThan(stopLossPrice) {
		return &RiskEvent{
			ID:        uuid.New().String(),
			Type:      "STOP_LOSS_TRIGGERED",
			Severity:  RiskLevelHigh,
			Message:   fmt.Sprintf("Stop loss triggered for %s at %s", symbol, position.CurrentPrice.String()),
			Symbol:    symbol,
			Exchange:  exchange,
			Value:     position.CurrentPrice,
			Threshold: stopLossPrice,
			Timestamp: time.Now(),
		}, nil
	}
	
	// Check take profit
	takeProfitPrice := position.EntryPrice.Mul(decimal.NewFromFloat(1).Add(rm.config.AlertThresholds.TakeProfitPercentage))
	if position.Side == "LONG" && position.CurrentPrice.GreaterThan(takeProfitPrice) {
		return &RiskEvent{
			ID:        uuid.New().String(),
			Type:      "TAKE_PROFIT_TRIGGERED",
			Severity:  RiskLevelLow,
			Message:   fmt.Sprintf("Take profit triggered for %s at %s", symbol, position.CurrentPrice.String()),
			Symbol:    symbol,
			Exchange:  exchange,
			Value:     position.CurrentPrice,
			Threshold: takeProfitPrice,
			Timestamp: time.Now(),
		}, nil
	}
	
	return nil, nil
}

// GetRiskEvents returns risk events with optional filtering
func (rm *Manager) GetRiskEvents(filters map[string]interface{}) ([]*RiskEvent, error) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	
	events := make([]*RiskEvent, 0)
	for _, event := range rm.riskEvents {
		if rm.matchesEventFilters(event, filters) {
			events = append(events, event)
		}
	}
	
	return events, nil
}

// SubscribeToRiskEvents subscribes to risk event notifications
func (rm *Manager) SubscribeToRiskEvents(callback func(*RiskEvent)) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	
	rm.eventCallbacks = append(rm.eventCallbacks, callback)
	return nil
}

// Start starts the risk manager
func (rm *Manager) Start() error {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	
	if rm.running {
		return fmt.Errorf("risk manager already running")
	}
	
	rm.running = true
	
	// Start risk monitoring goroutine
	go rm.riskMonitoringLoop()
	
	log.Println("Risk manager started")
	return nil
}

// Stop stops the risk manager
func (rm *Manager) Stop() error {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	
	if !rm.running {
		return nil
	}
	
	rm.running = false
	rm.cancel()
	
	log.Println("Risk manager stopped")
	return nil
}

// IsRunning returns whether the risk manager is running
func (rm *Manager) IsRunning() bool {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	return rm.running
}

// Private methods

func (rm *Manager) updatePortfolioValue() {
	rm.portfolio.TotalValue = rm.portfolio.CashBalance
	rm.portfolio.InvestedValue = decimal.Zero
	rm.portfolio.UnrealizedPNL = decimal.Zero
	
	for _, position := range rm.portfolio.Positions {
		rm.portfolio.TotalValue = rm.portfolio.TotalValue.Add(position.MarketValue)
		rm.portfolio.InvestedValue = rm.portfolio.InvestedValue.Add(position.Quantity.Mul(position.EntryPrice))
		rm.portfolio.UnrealizedPNL = rm.portfolio.UnrealizedPNL.Add(position.UnrealizedPNL)
	}
}

func (rm *Manager) calculateRiskMetrics() {
	rm.riskMetrics.PortfolioValue = rm.portfolio.TotalValue
	rm.riskMetrics.TotalExposure = rm.portfolio.InvestedValue
	rm.riskMetrics.LastUpdated = time.Now()
	
	// Calculate leverage
	if rm.portfolio.CashBalance.GreaterThan(decimal.Zero) {
		rm.riskMetrics.Leverage = rm.portfolio.InvestedValue.Div(rm.portfolio.CashBalance)
	} else {
		rm.riskMetrics.Leverage = decimal.Zero
	}
	
	// Calculate concentration risk (max position as % of portfolio)
	maxPositionValue := decimal.Zero
	for _, position := range rm.portfolio.Positions {
		if position.MarketValue.GreaterThan(maxPositionValue) {
			maxPositionValue = position.MarketValue
		}
	}
	
	if rm.portfolio.TotalValue.GreaterThan(decimal.Zero) {
		rm.riskMetrics.ConcentrationRisk = maxPositionValue.Div(rm.portfolio.TotalValue)
	} else {
		rm.riskMetrics.ConcentrationRisk = decimal.Zero
	}
	
	// Update metrics
	if rm.metrics != nil {
		rm.metrics.RecordPortfolioValue(rm.portfolio.TotalValue.InexactFloat64())
		rm.metrics.RecordPositionCount(float64(len(rm.portfolio.Positions)))
		rm.metrics.RecordDailyLoss(rm.portfolio.DailyPNL.InexactFloat64())
	}
}

func (rm *Manager) checkPortfolioRisk() {
	events, err := rm.CheckPortfolioRisk()
	if err != nil {
		log.Printf("Error checking portfolio risk: %v", err)
		return
	}
	
	for _, event := range events {
		rm.addRiskEvent(event)
	}
}

func (rm *Manager) checkPositionRisk(symbol, exchange string) {
	event, err := rm.CheckPositionRisk(symbol, exchange)
	if err != nil {
		log.Printf("Error checking position risk: %v", err)
		return
	}
	
	if event != nil {
		rm.addRiskEvent(event)
	}
}

func (rm *Manager) addRiskEvent(event *RiskEvent) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	
	rm.riskEvents = append(rm.riskEvents, event)
	
	// Keep only last 1000 events
	if len(rm.riskEvents) > 1000 {
		rm.riskEvents = rm.riskEvents[len(rm.riskEvents)-1000:]
	}
	
	// Notify callbacks
	for _, callback := range rm.eventCallbacks {
		go callback(event)
	}
	
	// Record metrics
	if rm.metrics != nil {
		rm.metrics.RecordRiskEvent(string(event.Type), string(event.Severity))
	}
}

func (rm *Manager) matchesEventFilters(event *RiskEvent, filters map[string]interface{}) bool {
	if severity, ok := filters["severity"]; ok {
		if event.Severity != RiskLevel(severity.(string)) {
			return false
		}
	}
	
	if eventType, ok := filters["type"]; ok {
		if event.Type != eventType.(string) {
			return false
		}
	}
	
	if symbol, ok := filters["symbol"]; ok {
		if event.Symbol != symbol.(string) {
			return false
		}
	}
	
	return true
}

func (rm *Manager) riskMonitoringLoop() {
	ticker := time.NewTicker(rm.config.UpdateInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			rm.calculateRiskMetrics()
			rm.checkPortfolioRisk()
		case <-rm.ctx.Done():
			return
		}
	}
}
