package alerts

import (
	"context"
	"encoding/json"
	"fmt"
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
		rule.ID = uuid.New().String()
	}
	
	am.rules[rule.ID] = rule
	
	am.logger.Info("Added alert rule", map[string]interface{}{
		"rule_id": rule.ID,
		"name":    rule.Name,
		"type":    rule.Type,
	})
	
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
	
	am.logger.Info("Removed alert rule", map[string]interface{}{
		"rule_id": ruleID,
	})
	
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
	
	am.logger.Info("Updated alert rule", map[string]interface{}{
		"rule_id": rule.ID,
		"name":    rule.Name,
	})
	
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
	
	// Check cooldown
	if time.Since(rule.LastTriggered) < rule.Cooldown {
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
	am.logger.Info("Alert triggered", map[string]interface{}{
		"alert_id": alert.ID,
		"rule_id":  rule.ID,
		"type":     rule.Type,
		"severity": rule.Severity,
	})
	
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
	
	am.logger.Info("Alert acknowledged", map[string]interface{}{
		"alert_id": alertID,
	})
	
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
	
	am.logger.Info("Alert resolved", map[string]interface{}{
		"alert_id": alertID,
	})
	
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
	
	am.logger.Info("Registered alert channel", map[string]interface{}{
		"channel_name": channel.Name(),
		"channel_type": channel.Type(),
	})
	
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
	
	am.logger.Info("Removed alert channel", map[string]interface{}{
		"channel_name": channelName,
	})
	
	return nil
}

// Start starts the alert manager
func (am *VelocimexAlertManager) Start() error {
	am.logger.Info("Starting alert manager")
	
	// Start event processing
	go am.processEvents()
	
	return nil
}

// Stop stops the alert manager
func (am *VelocimexAlertManager) Stop() error {
	am.logger.Info("Stopping alert manager")
	
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
		template = replaceAll(template, placeholder, strValue)
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
					am.logger.Error("Failed to send alert to channel", map[string]interface{}{
						"channel": ch.Name(),
						"error":   err.Error(),
					})
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
					am.logger.Error("Failed to send alert to channel", map[string]interface{}{
						"channel": ch.Name(),
						"error":   err.Error(),
					})
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
		case event, ok := <-am.eventChan:
			if !ok {
				return
			}
			
			am.logger.Debug("Processing alert event", map[string]interface{}{
				"event_type": event.Type,
				"alert_id":   event.AlertID,
			})
			
		case <-am.ctx.Done():
			return
		}
	}
}

// Helper function for string replacement
func replaceAll(s, old, new string) string {
	// Simple implementation - in production, use strings.ReplaceAll
	result := s
	for {
		if idx := indexOf(result, old); idx != -1 {
			result = result[:idx] + new + result[idx+len(old):]
		} else {
			break
		}
	}
	return result
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}