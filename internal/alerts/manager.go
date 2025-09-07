package alerts

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"velocimex/internal/logger"
)

// VelocimexAlertManager implements the AlertManager interface
type VelocimexAlertManager struct {
	rules     map[string]*AlertRule
	alerts    map[string]*Alert
	channels  map[string]AlertChannel
	ruleMutex sync.RWMutex
	alertMutex sync.RWMutex
	channelMutex sync.RWMutex
	
	logger logger.Logger
	
	ctx    context.Context
	cancel context.CancelFunc
	
	eventChan chan *AlertEvent
}

// NewAlertManager creates a new alert manager
func NewAlertManager(logger logger.Logger) *VelocimexAlertManager {
	ctx, cancel := context.WithCancel(context.Background())
	
	am := &VelocimexAlertManager{
		rules:     make(map[string]*AlertRule),
		alerts:    make(map[string]*Alert),
		channels:  make(map[string]AlertChannel),
		logger:    logger,
		ctx:       ctx,
		cancel:    cancel,
		eventChan: make(chan *AlertEvent, 1000),
	}
	
	return am
}

// AddRule adds a new alert rule
func (am *VelocimexAlertManager) AddRule(rule *AlertRule) error {
	am.ruleMutex.Lock()
	defer am.ruleMutex.Unlock()
	
	if rule.ID == "" {
		rule.ID = uuid.NewString()
	}
	
	am.rules[rule.ID] = rule
	
	if am.logger != nil {
		am.logger.Info("alert", "Added alert rule")
	}
	
	return nil
}

// RemoveRule removes an alert rule
func (am *VelocimexAlertManager) RemoveRule(ruleID string) error {
	am.ruleMutex.Lock()
	defer am.ruleMutex.Unlock()
	
	if _, exists := am.rules[ruleID]; !exists {
		return fmt.Errorf("rule %s not found", ruleID)
	}
	
	delete(am.rules, ruleID)
	
	if am.logger != nil {
		am.logger.Info("alert", "Removed alert rule")
	}
	
	return nil
}

// UpdateRule updates an existing alert rule
func (am *VelocimexAlertManager) UpdateRule(rule *AlertRule) error {
	am.ruleMutex.Lock()
	defer am.ruleMutex.Unlock()
	
	if _, exists := am.rules[rule.ID]; !exists {
		return fmt.Errorf("rule %s not found", rule.ID)
	}
	
	am.rules[rule.ID] = rule
	
	if am.logger != nil {
		am.logger.Info("alert", "Updated alert rule")
	}
	
	return nil
}

// GetRule retrieves a specific alert rule
func (am *VelocimexAlertManager) GetRule(ruleID string) (*AlertRule, error) {
	am.ruleMutex.RLock()
	defer am.ruleMutex.RUnlock()
	
	rule, exists := am.rules[ruleID]
	if !exists {
		return nil, fmt.Errorf("rule %s not found", ruleID)
	}
	
	return rule, nil
}

// GetRules returns all alert rules
func (am *VelocimexAlertManager) GetRules() []*AlertRule {
	am.ruleMutex.RLock()
	defer am.ruleMutex.RUnlock()
	
	rules := make([]*AlertRule, 0, len(am.rules))
	for _, rule := range am.rules {
		rules = append(rules, rule)
	}
	
	return rules
}

// TriggerAlert triggers an alert based on a rule
func (am *VelocimexAlertManager) TriggerAlert(rule *AlertRule, data interface{}) error {
	if !rule.Enabled {
		return nil
	}
	
	// Check cooldown (with proper locking)
	am.ruleMutex.RLock()
	lastTriggered := rule.LastTriggered
	am.ruleMutex.RUnlock()
	
	if time.Since(lastTriggered) < rule.Cooldown {
		return nil
	}
	
	// Check conditions
	if !am.evaluateConditions(rule.Conditions, data) {
		return nil
	}
	
	// Create alert
	alert := &Alert{
		ID:        uuid.New().String(),
		RuleID:    rule.ID,
		Type:      rule.Type,
		Severity:  rule.Severity,
		Title:     rule.Name,
		Message:   am.formatMessage(rule.Message, data),
		Data:      data,
		Timestamp: time.Now(),
	}
	
	// Store alert
	am.alertMutex.Lock()
	am.alerts[alert.ID] = alert
	am.alertMutex.Unlock()
	
	// Update rule last triggered time
	am.ruleMutex.Lock()
	rule.LastTriggered = time.Now()
	am.ruleMutex.Unlock()
	
	// Send to channels
	am.sendAlertToChannels(alert, rule.Channels)
	
	// Log alert
	if am.logger != nil {
		am.logger.Info("alert", "Alert triggered")
	}
	
	// Send event
	am.eventChan <- &AlertEvent{
		Type:      "alert_triggered",
		AlertID:   alert.ID,
		RuleID:    rule.ID,
		Timestamp: time.Now(),
		Data:      data,
	}
	
	return nil
}

// AcknowledgeAlert marks an alert as acknowledged
func (am *VelocimexAlertManager) AcknowledgeAlert(alertID string) error {
	am.alertMutex.Lock()
	defer am.alertMutex.Unlock()
	
	alert, exists := am.alerts[alertID]
	if !exists {
		return fmt.Errorf("alert %s not found", alertID)
	}
	
	alert.Acknowledged = true
	
	if am.logger != nil {
		am.logger.Info("alert", "Alert acknowledged")
	}
	
	return nil
}

// ResolveAlert marks an alert as resolved
func (am *VelocimexAlertManager) ResolveAlert(alertID string) error {
	am.alertMutex.Lock()
	defer am.alertMutex.Unlock()
	
	alert, exists := am.alerts[alertID]
	if !exists {
		return fmt.Errorf("alert %s not found", alertID)
	}
	
	alert.Resolved = true
	now := time.Now()
	alert.ResolvedAt = &now
	
	if am.logger != nil {
		am.logger.Info("alert", "Alert resolved")
	}
	
	return nil
}

// GetAlerts returns alerts based on filters
func (am *VelocimexAlertManager) GetAlerts(filters map[string]interface{}) ([]*Alert, error) {
	am.alertMutex.RLock()
	defer am.alertMutex.RUnlock()
	
	alerts := make([]*Alert, 0, len(am.alerts))
	for _, alert := range am.alerts {
		if am.matchesFilters(alert, filters) {
			alerts = append(alerts, alert)
		}
	}
	
	return alerts, nil
}

// GetActiveAlerts returns all active (unresolved) alerts
func (am *VelocimexAlertManager) GetActiveAlerts() ([]*Alert, error) {
	return am.GetAlerts(map[string]interface{}{
		"resolved": false,
	})
}

// RegisterChannel registers a new alert delivery channel
func (am *VelocimexAlertManager) RegisterChannel(channel AlertChannel) error {
	am.channelMutex.Lock()
	defer am.channelMutex.Unlock()
	
	am.channels[channel.Name()] = channel
	
	if am.logger != nil {
		am.logger.Info("alert", "Registered alert channel")
	}
	
	return nil
}

// RemoveChannel removes an alert delivery channel
func (am *VelocimexAlertManager) RemoveChannel(channelName string) error {
	am.channelMutex.Lock()
	defer am.channelMutex.Unlock()
	
	if _, exists := am.channels[channelName]; !exists {
		return fmt.Errorf("channel %s not found", channelName)
	}
	
	delete(am.channels, channelName)
	
	if am.logger != nil {
		am.logger.Info("alert", "Removed alert channel")
	}
	
	return nil
}

// Start starts the alert manager
func (am *VelocimexAlertManager) Start() error {
	if am.logger != nil {
		am.logger.Info("alert", "Starting alert manager")
	}
	
	// Start event processing
	go am.processEvents()
	
	return nil
}

// Stop stops the alert manager
func (am *VelocimexAlertManager) Stop() error {
	if am.logger != nil {
		am.logger.Info("alert", "Stopping alert manager")
	}
	
	am.cancel()
	
	// Wait for event processing to finish
	close(am.eventChan)
	
	return nil
}

// evaluateConditions evaluates alert conditions against data
func (am *VelocimexAlertManager) evaluateConditions(conditions []AlertCondition, data interface{}) bool {
	if len(conditions) == 0 {
		return true
	}
	
	dataMap := make(map[string]interface{})
	if data != nil {
		jsonData, _ := json.Marshal(data)
		_ = json.Unmarshal(jsonData, &dataMap)
	}
	
	for _, condition := range conditions {
		if !am.evaluateCondition(condition, dataMap) {
			return false
		}
	}
	
	return true
}

// evaluateCondition evaluates a single condition
func (am *VelocimexAlertManager) evaluateCondition(condition AlertCondition, data map[string]interface{}) bool {
	fieldValue, exists := data[condition.Field]
	if !exists {
		return false
	}
	
	// Convert value to float64 for numeric comparisons
	var numericValue float64
	var stringValue string
	
	switch v := fieldValue.(type) {
	case float64:
		numericValue = v
	case int:
		numericValue = float64(v)
	case string:
		stringValue = v
	default:
		return false
	}
	
	// Convert condition value
	var conditionValue float64
	var conditionString string
	
	switch v := condition.Value.(type) {
	case float64:
		conditionValue = v
	case int:
		conditionValue = float64(v)
	case string:
		conditionString = v
	}
	
	switch condition.Operator {
	case "gt":
		return numericValue > conditionValue
	case "lt":
		return numericValue < conditionValue
	case "eq":
		return numericValue == conditionValue || stringValue == conditionString
	case "ne":
		return numericValue != conditionValue || stringValue != conditionString
	case "contains":
		return stringValue == conditionString
	default:
		return false
	}
}

// formatMessage formats the alert message with data
func (am *VelocimexAlertManager) formatMessage(template string, data interface{}) string {
	if data == nil {
		return template
	}
	
	// Simple template formatting - in production, use a proper template engine
	dataMap := make(map[string]interface{})
	jsonData, _ := json.Marshal(data)
	_ = json.Unmarshal(jsonData, &dataMap)
	
	// Basic placeholder replacement
	for key, value := range dataMap {
		placeholder := fmt.Sprintf("{{%s}}", key)
		strValue := fmt.Sprintf("%v", value)
		template = strings.ReplaceAll(template, placeholder, strValue)
	}
	
	return template
}

// sendAlertToChannels sends an alert to registered channels
func (am *VelocimexAlertManager) sendAlertToChannels(alert *Alert, channelNames []string) {
	am.channelMutex.RLock()
	defer am.channelMutex.RUnlock()
	
	// If no specific channels specified, use all registered channels
	if len(channelNames) == 0 {
		for _, channel := range am.channels {
			go func(ch AlertChannel) {
				if err := ch.Send(alert); err != nil {
					if am.logger != nil {
						am.logger.Error("alert", "Failed to send alert to channel")
					}
				}
			}(channel)
		}
		return
	}
	
	// Send to specific channels
	for _, channelName := range channelNames {
		if channel, exists := am.channels[channelName]; exists {
			go func(ch AlertChannel) {
				if err := ch.Send(alert); err != nil {
					if am.logger != nil {
						am.logger.Error("alert", "Failed to send alert to channel")
					}
				}
			}(channel)
		}
	}
}


// matchesFilters checks if an alert matches the given filters
func (am *VelocimexAlertManager) matchesFilters(alert *Alert, filters map[string]interface{}) bool {
	for key, value := range filters {
		switch key {
		case "type":
			if alert.Type != value.(AlertType) {
				return false
			}
		case "severity":
			if alert.Severity != value.(AlertSeverity) {
				return false
			}
		case "resolved":
			if alert.Resolved != value.(bool) {
				return false
			}
		case "acknowledged":
			if alert.Acknowledged != value.(bool) {
				return false
			}
		}
	}
	return true
}

// processEvents processes alert events
func (am *VelocimexAlertManager) processEvents() {
	for {
		select {
		case _, ok := <-am.eventChan:
			if !ok {
				return
			}
			if am.logger != nil {
				am.logger.Debug("alert", "Processing alert event")
			}
		case <-am.ctx.Done():
			return
		}
	}
}
