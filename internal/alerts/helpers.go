package alerts

import (
	"context"
	"fmt"
	"sync"
	"time"

	"velocimex/internal/logger"
)

// GlobalAlertManager is the global instance of the alert manager
var (
	globalAlertManager *VelocimexAlertManager
	globalManagerMutex sync.RWMutex
)

// Init initializes the global alert manager
func Init(logger logger.Logger) error {
	globalManagerMutex.Lock()
	defer globalManagerMutex.Unlock()
	
	if globalAlertManager != nil {
		return fmt.Errorf("alert manager already initialized")
	}
	
	config, err := LoadAlertConfig(GetAlertConfigPath())
	if err != nil {
		return fmt.Errorf("failed to load alert config: %w", err)
	}
	
	manager, err := SetupAlertManager(config, logger)
	if err != nil {
		return fmt.Errorf("failed to setup alert manager: %w", err)
	}
	
	if err := manager.Start(); err != nil {
		return fmt.Errorf("failed to start alert manager: %w", err)
	}
	
	globalAlertManager = manager
	
	logger.Info("Alert system initialized", map[string]interface{}{
		"rules_count":    len(manager.GetRules()),
		"channels_count": len(manager.channels),
	})
	
	return nil
}

// Shutdown shuts down the global alert manager
func Shutdown() error {
	globalManagerMutex.Lock()
	defer globalManagerMutex.Unlock()
	
	if globalAlertManager == nil {
		return nil
	}
	
	if err := globalAlertManager.Stop(); err != nil {
		return fmt.Errorf("failed to stop alert manager: %w", err)
	}
	
	globalAlertManager = nil
	return nil
}

// GetManager returns the global alert manager instance
func GetManager() *VelocimexAlertManager {
	globalManagerMutex.RLock()
	defer globalManagerMutex.RUnlock()
	return globalAlertManager
}

// TriggerPriceAlert triggers a price-based alert
func TriggerPriceAlert(symbol string, price, previous float64) error {
	globalManagerMutex.RLock()
	defer globalManagerMutex.RUnlock()
	
	if globalAlertManager == nil {
		return fmt.Errorf("alert manager not initialized")
	}
	
	change := price - previous
	changePct := 0.0
	if previous != 0 {
		changePct = (change / previous) * 100
	}
	
	data := PriceAlertData{
		Symbol:    symbol,
		Price:     price,
		Previous:  previous,
		Change:    change,
		ChangePct: changePct,
	}
	
	// Trigger all price rules
	rules := globalAlertManager.GetRules()
	for _, rule := range rules {
		if rule.Type == AlertTypePrice {
			_ = globalAlertManager.TriggerAlert(rule, data)
		}
	}
	
	return nil
}

// TriggerRiskAlert triggers a risk-based alert
func TriggerRiskAlert(portfolioValue, riskAmount float64, positionCount int, maxDrawdown float64) error {
	globalManagerMutex.RLock()
	defer globalManagerMutex.RUnlock()
	
	if globalAlertManager == nil {
		return fmt.Errorf("alert manager not initialized")
	}
	
	riskPct := 0.0
	if portfolioValue != 0 {
		riskPct = (riskAmount / portfolioValue) * 100
	}
	
	data := RiskAlertData{
		PortfolioValue: portfolioValue,
		RiskAmount:     riskAmount,
		RiskPct:        riskPct,
		PositionCount:  positionCount,
		MaxDrawdown:    maxDrawdown,
	}
	
	// Trigger all risk rules
	rules := globalAlertManager.GetRules()
	for _, rule := range rules {
		if rule.Type == AlertTypeRisk {
			_ = globalAlertManager.TriggerAlert(rule, data)
		}
	}
	
	return nil
}

// TriggerStrategyAlert triggers a strategy-based alert
func TriggerStrategyAlert(strategyID, signal string, confidence float64, metadata interface{}) error {
	globalManagerMutex.RLock()
	defer globalManagerMutex.RUnlock()
	
	if globalAlertManager == nil {
		return fmt.Errorf("alert manager not initialized")
	}
	
	data := StrategyAlertData{
		StrategyID: strategyID,
		Signal:     signal,
		Confidence: confidence,
		Metadata:   metadata,
	}
	
	// Trigger all strategy rules
	rules := globalAlertManager.GetRules()
	for _, rule := range rules {
		if rule.Type == AlertTypeStrategy {
			_ = globalAlertManager.TriggerAlert(rule, data)
		}
	}
	
	return nil
}

// TriggerSystemAlert triggers a system-based alert
func TriggerSystemAlert(component, status, errorMsg string, uptime, memoryUsage string) error {
	globalManagerMutex.RLock()
	defer globalManagerMutex.RUnlock()
	
	if globalAlertManager == nil {
		return fmt.Errorf("alert manager not initialized")
	}
	
	data := SystemAlertData{
		Component:   component,
		Status:      status,
		Error:       errorMsg,
		Uptime:      uptime,
		MemoryUsage: memoryUsage,
	}
	
	// Trigger all system rules
	rules := globalAlertManager.GetRules()
	for _, rule := range rules {
		if rule.Type == AlertTypeSystem {
			_ = globalAlertManager.TriggerAlert(rule, data)
		}
	}
	
	return nil
}

// TriggerVolumeAlert triggers a volume-based alert
func TriggerVolumeAlert(symbol string, volume, normalVolume float64) error {
	globalManagerMutex.RLock()
	defer globalManagerMutex.RUnlock()
	
	if globalAlertManager == nil {
		return fmt.Errorf("alert manager not initialized")
	}
	
	volumeRatio := 1.0
	if normalVolume != 0 {
		volumeRatio = volume / normalVolume
	}
	
	data := map[string]interface{}{
		"symbol":        symbol,
		"volume":        volume,
		"normal_volume": normalVolume,
		"volume_ratio":  volumeRatio,
	}
	
	// Trigger all volume rules
	rules := globalAlertManager.GetRules()
	for _, rule := range rules {
		if rule.Type == AlertTypeVolume {
			_ = globalAlertManager.TriggerAlert(rule, data)
		}
	}
	
	return nil
}

// TriggerConnectivityAlert triggers a connectivity-based alert
func TriggerConnectivityAlert(component, status, errorMsg string) error {
	globalManagerMutex.RLock()
	defer globalManagerMutex.RUnlock()
	
	if globalAlertManager == nil {
		return fmt.Errorf("alert manager not initialized")
	}
	
	data := map[string]interface{}{
		"component": component,
		"status":    status,
		"error":     errorMsg,
	}
	
	// Trigger all connectivity rules
	rules := globalAlertManager.GetRules()
	for _, rule := range rules {
		if rule.Type == AlertTypeConnectivity {
			_ = globalAlertManager.TriggerAlert(rule, data)
		}
	}
	
	return nil
}

// TriggerPerformanceAlert triggers a performance-based alert
func TriggerPerformanceAlert(component, metric string, value, threshold float64) error {
	globalManagerMutex.RLock()
	defer globalManagerMutex.RUnlock()
	
	if globalAlertManager == nil {
		return fmt.Errorf("alert manager not initialized")
	}
	
	data := map[string]interface{}{
		"component": component,
		"metric":    metric,
		"value":     value,
		"threshold": threshold,
	}
	
	// Trigger all performance rules
	rules := globalAlertManager.GetRules()
	for _, rule := range rules {
		if rule.Type == AlertTypePerformance {
			_ = globalAlertManager.TriggerAlert(rule, data)
		}
	}
	
	return nil
}

// GetActiveAlerts returns all active alerts
func GetActiveAlerts() ([]*Alert, error) {
	globalManagerMutex.RLock()
	defer globalManagerMutex.RUnlock()
	
	if globalAlertManager == nil {
		return nil, fmt.Errorf("alert manager not initialized")
	}
	
	return globalAlertManager.GetActiveAlerts()
}

// GetAllAlerts returns all alerts with optional filters
func GetAllAlerts(filters map[string]interface{}) ([]*Alert, error) {
	globalManagerMutex.RLock()
	defer globalManagerMutex.RUnlock()
	
	if globalAlertManager == nil {
		return nil, fmt.Errorf("alert manager not initialized")
	}
	
	return globalAlertManager.GetAlerts(filters)
}

// AcknowledgeAlert marks an alert as acknowledged
func AcknowledgeAlert(alertID string) error {
	globalManagerMutex.RLock()
	defer globalManagerMutex.RUnlock()
	
	if globalAlertManager == nil {
		return fmt.Errorf("alert manager not initialized")
	}
	
	return globalAlertManager.AcknowledgeAlert(alertID)
}

// ResolveAlert marks an alert as resolved
func ResolveAlert(alertID string) error {
	globalManagerMutex.RLock()
	defer globalManagerMutex.RUnlock()
	
	if globalAlertManager == nil {
		return fmt.Errorf("alert manager not initialized")
	}
	
	return globalAlertManager.ResolveAlert(alertID)
}

// AddRule adds a new alert rule
func AddRule(rule *AlertRule) error {
	globalManagerMutex.RLock()
	defer globalManagerMutex.RUnlock()
	
	if globalAlertManager == nil {
		return fmt.Errorf("alert manager not initialized")
	}
	
	return globalAlertManager.AddRule(rule)
}

// RemoveRule removes an alert rule
func RemoveRule(ruleID string) error {
	globalManagerMutex.RLock()
	defer globalManagerMutex.RUnlock()
	
	if globalAlertManager == nil {
		return fmt.Errorf("alert manager not initialized")
	}
	
	return globalAlertManager.RemoveRule(ruleID)
}

// RegisterChannel registers a new alert channel
func RegisterChannel(channel AlertChannel) error {
	globalManagerMutex.RLock()
	defer globalManagerMutex.RUnlock()
	
	if globalAlertManager == nil {
		return fmt.Errorf("alert manager not initialized")
	}
	
	return globalAlertManager.RegisterChannel(channel)
}

// AlertMonitor provides a simple interface for monitoring and alerting
// It can be used as a context-based alert system
func AlertMonitor(ctx context.Context, component string) *ComponentMonitor {
	return &ComponentMonitor{
		component: component,
		ctx:       ctx,
	}
}

// ComponentMonitor provides component-specific alerting
type ComponentMonitor struct {
	component string
	ctx       context.Context
}

// Info sends an informational alert
func (m *ComponentMonitor) Info(message string, data interface{}) {
	TriggerSystemAlert(m.component, "info", message, "", "")
}

// Warn sends a warning alert
func (m *ComponentMonitor) Warn(message string, data interface{}) {
	TriggerSystemAlert(m.component, "warning", message, "", "")
}

// Error sends an error alert
func (m *ComponentMonitor) Error(message string, err error, data interface{}) {
	errorMsg := ""
	if err != nil {
		errorMsg = err.Error()
	}
	TriggerSystemAlert(m.component, "error", message, errorMsg, "")
}

// Critical sends a critical alert
func (m *ComponentMonitor) Critical(message string, err error, data interface{}) {
	errorMsg := ""
	if err != nil {
		errorMsg = err.Error()
	}
	TriggerSystemAlert(m.component, "critical", message, errorMsg, "")
}

// Performance sends a performance alert
func (m *ComponentMonitor) Performance(metric string, value, threshold float64) {
	TriggerPerformanceAlert(m.component, metric, value, threshold)
}

// Connectivity sends a connectivity alert
func (m *ComponentMonitor) Connectivity(status, errorMsg string) {
	TriggerConnectivityAlert(m.component, status, errorMsg)
}

// AlertRuleBuilder provides a fluent interface for building alert rules
func NewAlertRuleBuilder() *AlertRuleBuilder {
	return &AlertRuleBuilder{
		rule: &AlertRule{
			Enabled:  true,
			Cooldown: 30 * time.Second,
			Channels: []string{},
		},
	}
}

// AlertRuleBuilder provides a fluent interface for building alert rules
type AlertRuleBuilder struct {
	rule *AlertRule
}

func (b *AlertRuleBuilder) Name(name string) *AlertRuleBuilder {
	b.rule.Name = name
	return b
}

func (b *AlertRuleBuilder) Type(alertType AlertType) *AlertRuleBuilder {
	b.rule.Type = alertType
	return b
}

func (b *AlertRuleBuilder) Severity(severity AlertSeverity) *AlertRuleBuilder {
	b.rule.Severity = severity
	return b
}

func (b *AlertRuleBuilder) Message(message string) *AlertRuleBuilder {
	b.rule.Message = message
	return b
}

func (b *AlertRuleBuilder) Condition(field, operator string, value interface{}) *AlertRuleBuilder {
	b.rule.Conditions = append(b.rule.Conditions, AlertCondition{
		Field:    field,
		Operator: operator,
		Value:    value,
	})
	return b
}

func (b *AlertRuleBuilder) Cooldown(duration time.Duration) *AlertRuleBuilder {
	b.rule.Cooldown = duration
	return b
}

func (b *AlertRuleBuilder) Channel(channel string) *AlertRuleBuilder {
	b.rule.Channels = append(b.rule.Channels, channel)
	return b
}

func (b *AlertRuleBuilder) Build() *AlertRule {
	return b.rule
}

// Validate ensures the alert rule is valid
func (b *AlertRuleBuilder) Validate() error {
	if b.rule.Name == "" {
		return fmt.Errorf("rule name is required")
	}
	if b.rule.Type == "" {
		return fmt.Errorf("rule type is required")
	}
	if b.rule.Severity == "" {
		return fmt.Errorf("rule severity is required")
	}
	if b.rule.Message == "" {
		return fmt.Errorf("rule message is required")
	}
	return nil
}