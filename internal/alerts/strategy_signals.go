package alerts

import (
	"context"
	"fmt"
	"sync"
	"time"

	"velocimex/internal/logger"
)

// StrategySignalAlertSystem handles strategy-specific alerts
type StrategySignalAlertSystem struct {
	engine        *AlertEngine
	strategyRules map[string][]*StrategyAlertRule
	signalAlerts  map[string]*SignalAlert
	performanceAlerts map[string]*PerformanceAlert
	riskAlerts    map[string]*RiskAlert
	
	mu            sync.RWMutex
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
	logger        logger.Logger
}

// StrategyAlertRule defines a strategy-specific alert rule
type StrategyAlertRule struct {
	ID          string                 `json:"id"`
	Strategy    string                 `json:"strategy"`
	Type        StrategyAlertType      `json:"type"`
	Condition   StrategyCondition      `json:"condition"`
	Threshold   float64                `json:"threshold"`
	Timeframe   time.Duration          `json:"timeframe"`
	Enabled     bool                   `json:"enabled"`
	Channels    []string               `json:"channels"`
	Metadata    map[string]interface{} `json:"metadata"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// StrategyAlertType represents different types of strategy alerts
type StrategyAlertType string

const (
	StrategyAlertSignal     StrategyAlertType = "signal"
	StrategyAlertPerformance StrategyAlertType = "performance"
	StrategyAlertRisk       StrategyAlertType = "risk"
	StrategyAlertDrawdown   StrategyAlertType = "drawdown"
	StrategyAlertSharpe     StrategyAlertType = "sharpe"
	StrategyAlertWinRate    StrategyAlertType = "win_rate"
)

// StrategyCondition represents the condition for a strategy alert
type StrategyCondition struct {
	Operator string  `json:"operator"` // "above", "below", "crosses_above", "crosses_below", "equals"
	Value    float64 `json:"value"`
	Percent  bool    `json:"percent"` // Whether to use percentage change
}

// SignalAlert tracks signal-based alerts
type SignalAlert struct {
	Strategy      string    `json:"strategy"`
	SignalType    string    `json:"signal_type"`
	Symbol        string    `json:"symbol"`
	Side          string    `json:"side"`
	Price         float64   `json:"price"`
	Quantity      float64   `json:"quantity"`
	Confidence    float64   `json:"confidence"`
	Threshold     float64   `json:"threshold"`
	Condition     StrategyCondition `json:"condition"`
	LastCheck     time.Time `json:"last_check"`
	Triggered     bool      `json:"triggered"`
}

// PerformanceAlert tracks performance-based alerts
type PerformanceAlert struct {
	Strategy      string    `json:"strategy"`
	Metric        string    `json:"metric"`
	CurrentValue  float64   `json:"current_value"`
	PreviousValue float64   `json:"previous_value"`
	Threshold     float64   `json:"threshold"`
	Condition     StrategyCondition `json:"condition"`
	LastCheck     time.Time `json:"last_check"`
	Triggered     bool      `json:"triggered"`
}

// RiskAlert tracks risk-based alerts
type RiskAlert struct {
	Strategy      string    `json:"strategy"`
	RiskType      string    `json:"risk_type"`
	CurrentValue  float64   `json:"current_value"`
	Threshold     float64   `json:"threshold"`
	Condition     StrategyCondition `json:"condition"`
	LastCheck     time.Time `json:"last_check"`
	Triggered     bool      `json:"triggered"`
}

// NewStrategySignalAlertSystem creates a new strategy signal alert system
func NewStrategySignalAlertSystem(engine *AlertEngine, logger logger.Logger) *StrategySignalAlertSystem {
	ctx, cancel := context.WithCancel(context.Background())
	
	sas := &StrategySignalAlertSystem{
		engine:        engine,
		strategyRules: make(map[string][]*StrategyAlertRule),
		signalAlerts:  make(map[string]*SignalAlert),
		performanceAlerts: make(map[string]*PerformanceAlert),
		riskAlerts:    make(map[string]*RiskAlert),
		ctx:           ctx,
		cancel:        cancel,
		logger:        logger,
	}

	// Start monitoring workers
	sas.wg.Add(1)
	go sas.signalMonitor()
	sas.wg.Add(1)
	go sas.performanceMonitor()
	sas.wg.Add(1)
	go sas.riskMonitor()

	return sas
}

// AddStrategyRule adds a strategy-specific alert rule
func (sas *StrategySignalAlertSystem) AddStrategyRule(rule *StrategyAlertRule) error {
	sas.mu.Lock()
	defer sas.mu.Unlock()

	// Validate rule
	if err := sas.validateStrategyRule(rule); err != nil {
		return fmt.Errorf("invalid strategy rule: %w", err)
	}

	// Set rule ID if not set
	if rule.ID == "" {
		rule.ID = fmt.Sprintf("%s_%s_%d", rule.Strategy, rule.Type, time.Now().UnixNano())
	}

	// Set timestamps
	now := time.Now()
	if rule.CreatedAt.IsZero() {
		rule.CreatedAt = now
	}
	rule.UpdatedAt = now

	// Add to strategy rules
	sas.strategyRules[rule.Strategy] = append(sas.strategyRules[rule.Strategy], rule)

	// Create specific alert based on type
	switch rule.Type {
	case StrategyAlertSignal:
		sas.signalAlerts[rule.ID] = &SignalAlert{
			Strategy:   rule.Strategy,
			Threshold:  rule.Threshold,
			Condition:  rule.Condition,
			LastCheck:  now,
		}
	case StrategyAlertPerformance:
		sas.performanceAlerts[rule.ID] = &PerformanceAlert{
			Strategy:   rule.Strategy,
			Threshold:  rule.Threshold,
			Condition:  rule.Condition,
			LastCheck:  now,
		}
	case StrategyAlertRisk:
		sas.riskAlerts[rule.ID] = &RiskAlert{
			Strategy:   rule.Strategy,
			Threshold:  rule.Threshold,
			Condition:  rule.Condition,
			LastCheck:  now,
		}
	}

	sas.logger.Info("alerts", fmt.Sprintf("Added strategy rule: %s", rule.ID), map[string]interface{}{
		"strategy": rule.Strategy,
		"type":     rule.Type,
	})

	return nil
}

// ProcessSignal processes a trading signal and checks for alerts
func (sas *StrategySignalAlertSystem) ProcessSignal(signal map[string]interface{}) {
	sas.mu.RLock()
	strategy := signal["strategy"].(string)
	rules, exists := sas.strategyRules[strategy]
	sas.mu.RUnlock()

	if !exists {
		return
	}

	for _, rule := range rules {
		if !rule.Enabled || rule.Type != StrategyAlertSignal {
			continue
		}

		sas.checkSignalAlert(rule, signal)
	}
}

// ProcessPerformance processes performance data and checks for alerts
func (sas *StrategySignalAlertSystem) ProcessPerformance(strategy string, performance map[string]interface{}) {
	sas.mu.RLock()
	rules, exists := sas.strategyRules[strategy]
	sas.mu.RUnlock()

	if !exists {
		return
	}

	for _, rule := range rules {
		if !rule.Enabled || rule.Type != StrategyAlertPerformance {
			continue
		}

		sas.checkPerformanceAlert(rule, performance)
	}
}

// ProcessRisk processes risk data and checks for alerts
func (sas *StrategySignalAlertSystem) ProcessRisk(strategy string, risk map[string]interface{}) {
	sas.mu.RLock()
	rules, exists := sas.strategyRules[strategy]
	sas.mu.RUnlock()

	if !exists {
		return
	}

	for _, rule := range rules {
		if !rule.Enabled || rule.Type != StrategyAlertRisk {
			continue
		}

		sas.checkRiskAlert(rule, risk)
	}
}

// checkSignalAlert checks if a signal alert should be triggered
func (sas *StrategySignalAlertSystem) checkSignalAlert(rule *StrategyAlertRule, signal map[string]interface{}) {
	sas.mu.Lock()
	alert, exists := sas.signalAlerts[rule.ID]
	if !exists {
		sas.mu.Unlock()
		return
	}

	// Update alert with signal data
	alert.SignalType = signal["type"].(string)
	alert.Symbol = signal["symbol"].(string)
	alert.Side = signal["side"].(string)
	alert.Price = signal["price"].(float64)
	alert.Quantity = signal["quantity"].(float64)
	alert.Confidence = signal["confidence"].(float64)
	alert.LastCheck = time.Now()
	sas.mu.Unlock()

	// Check condition
	if sas.evaluateSignalCondition(alert) {
		sas.triggerSignalAlert(rule, alert, signal)
	}
}

// checkPerformanceAlert checks if a performance alert should be triggered
func (sas *StrategySignalAlertSystem) checkPerformanceAlert(rule *StrategyAlertRule, performance map[string]interface{}) {
	sas.mu.Lock()
	alert, exists := sas.performanceAlerts[rule.ID]
	if !exists {
		sas.mu.Unlock()
		return
	}

	// Update alert with performance data
	alert.PreviousValue = alert.CurrentValue
	alert.CurrentValue = performance["value"].(float64)
	alert.Metric = performance["metric"].(string)
	alert.LastCheck = time.Now()
	sas.mu.Unlock()

	// Check condition
	if sas.evaluatePerformanceCondition(alert) {
		sas.triggerPerformanceAlert(rule, alert, performance)
	}
}

// checkRiskAlert checks if a risk alert should be triggered
func (sas *StrategySignalAlertSystem) checkRiskAlert(rule *StrategyAlertRule, risk map[string]interface{}) {
	sas.mu.Lock()
	alert, exists := sas.riskAlerts[rule.ID]
	if !exists {
		sas.mu.Unlock()
		return
	}

	// Update alert with risk data
	alert.CurrentValue = risk["value"].(float64)
	alert.RiskType = risk["type"].(string)
	alert.LastCheck = time.Now()
	sas.mu.Unlock()

	// Check condition
	if sas.evaluateRiskCondition(alert) {
		sas.triggerRiskAlert(rule, alert, risk)
	}
}

// evaluateSignalCondition evaluates a signal alert condition
func (sas *StrategySignalAlertSystem) evaluateSignalCondition(alert *SignalAlert) bool {
	switch alert.Condition.Operator {
	case "above":
		return alert.Confidence > alert.Threshold
	case "below":
		return alert.Confidence < alert.Threshold
	case "equals":
		return alert.Confidence == alert.Threshold
	default:
		return false
	}
}

// evaluatePerformanceCondition evaluates a performance alert condition
func (sas *StrategySignalAlertSystem) evaluatePerformanceCondition(alert *PerformanceAlert) bool {
	switch alert.Condition.Operator {
	case "above":
		return alert.CurrentValue > alert.Threshold
	case "below":
		return alert.CurrentValue < alert.Threshold
	case "crosses_above":
		return alert.PreviousValue <= alert.Threshold && alert.CurrentValue > alert.Threshold
	case "crosses_below":
		return alert.PreviousValue >= alert.Threshold && alert.CurrentValue < alert.Threshold
	case "equals":
		return alert.CurrentValue == alert.Threshold
	default:
		return false
	}
}

// evaluateRiskCondition evaluates a risk alert condition
func (sas *StrategySignalAlertSystem) evaluateRiskCondition(alert *RiskAlert) bool {
	switch alert.Condition.Operator {
	case "above":
		return alert.CurrentValue > alert.Threshold
	case "below":
		return alert.CurrentValue < alert.Threshold
	case "equals":
		return alert.CurrentValue == alert.Threshold
	default:
		return false
	}
}

// triggerSignalAlert triggers a signal alert
func (sas *StrategySignalAlertSystem) triggerSignalAlert(rule *StrategyAlertRule, alert *SignalAlert, signal map[string]interface{}) {
	sas.mu.Lock()
	alert.Triggered = true
	sas.mu.Unlock()

	// Create alert event
	event := &AlertEvent{
		ID:        fmt.Sprintf("signal_%s_%d", rule.ID, time.Now().UnixNano()),
		Type:      "strategy_signal",
		Severity:  SeverityMedium,
		Source:    "strategy_monitor",
		Message:   fmt.Sprintf("Signal alert triggered for strategy %s", alert.Strategy),
		Metadata: map[string]interface{}{
			"strategy":     alert.Strategy,
			"signal_type":  alert.SignalType,
			"symbol":       alert.Symbol,
			"side":         alert.Side,
			"price":        alert.Price,
			"quantity":     alert.Quantity,
			"confidence":   alert.Confidence,
			"threshold":    alert.Threshold,
			"condition":    alert.Condition.Operator,
			"rule_id":      rule.ID,
		},
		Timestamp: time.Now(),
	}

	// Process event
	sas.engine.ProcessEvent(event)
}

// triggerPerformanceAlert triggers a performance alert
func (sas *StrategySignalAlertSystem) triggerPerformanceAlert(rule *StrategyAlertRule, alert *PerformanceAlert, performance map[string]interface{}) {
	sas.mu.Lock()
	alert.Triggered = true
	sas.mu.Unlock()

	// Create alert event
	event := &AlertEvent{
		ID:        fmt.Sprintf("performance_%s_%d", rule.ID, time.Now().UnixNano()),
		Type:      "strategy_performance",
		Severity:  SeverityHigh,
		Source:    "strategy_monitor",
		Message:   fmt.Sprintf("Performance alert triggered for strategy %s", alert.Strategy),
		Metadata: map[string]interface{}{
			"strategy":        alert.Strategy,
			"metric":          alert.Metric,
			"current_value":   alert.CurrentValue,
			"previous_value":  alert.PreviousValue,
			"threshold":       alert.Threshold,
			"condition":       alert.Condition.Operator,
			"rule_id":         rule.ID,
		},
		Timestamp: time.Now(),
	}

	// Process event
	sas.engine.ProcessEvent(event)
}

// triggerRiskAlert triggers a risk alert
func (sas *StrategySignalAlertSystem) triggerRiskAlert(rule *StrategyAlertRule, alert *RiskAlert, risk map[string]interface{}) {
	sas.mu.Lock()
	alert.Triggered = true
	sas.mu.Unlock()

	// Create alert event
	event := &AlertEvent{
		ID:        fmt.Sprintf("risk_%s_%d", rule.ID, time.Now().UnixNano()),
		Type:      "strategy_risk",
		Severity:  SeverityCritical,
		Source:    "strategy_monitor",
		Message:   fmt.Sprintf("Risk alert triggered for strategy %s", alert.Strategy),
		Metadata: map[string]interface{}{
			"strategy":      alert.Strategy,
			"risk_type":     alert.RiskType,
			"current_value": alert.CurrentValue,
			"threshold":     alert.Threshold,
			"condition":     alert.Condition.Operator,
			"rule_id":       rule.ID,
		},
		Timestamp: time.Now(),
	}

	// Process event
	sas.engine.ProcessEvent(event)
}

// Monitor workers

func (sas *StrategySignalAlertSystem) signalMonitor() {
	defer sas.wg.Done()
	
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			sas.checkSignalAlerts()
		case <-sas.ctx.Done():
			return
		}
	}
}

func (sas *StrategySignalAlertSystem) performanceMonitor() {
	defer sas.wg.Done()
	
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			sas.checkPerformanceAlerts()
		case <-sas.ctx.Done():
			return
		}
	}
}

func (sas *StrategySignalAlertSystem) riskMonitor() {
	defer sas.wg.Done()
	
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			sas.checkRiskAlerts()
		case <-sas.ctx.Done():
			return
		}
	}
}

// Check methods

func (sas *StrategySignalAlertSystem) checkSignalAlerts() {
	// Implementation would check signal alerts
}

func (sas *StrategySignalAlertSystem) checkPerformanceAlerts() {
	// Implementation would check performance alerts
}

func (sas *StrategySignalAlertSystem) checkRiskAlerts() {
	// Implementation would check risk alerts
}

// Helper methods

func (sas *StrategySignalAlertSystem) validateStrategyRule(rule *StrategyAlertRule) error {
	if rule.Strategy == "" {
		return fmt.Errorf("strategy is required")
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

// Close shuts down the strategy signal alert system
func (sas *StrategySignalAlertSystem) Close() error {
	sas.cancel()
	sas.wg.Wait()
	return nil
}
