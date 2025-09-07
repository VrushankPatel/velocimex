package alerts

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"velocimex/internal/logger"
)

// AlertEngine provides advanced alert processing and management
type AlertEngine struct {
	config        *AlertConfig
	rules         map[string]*AlertRule
	channels      map[string]AlertChannel
	processors    map[string]AlertProcessor
	templates     map[string]*AlertTemplate
	subscriptions map[string][]string // event type -> rule IDs
	
	// Processing
	eventQueue    chan *AlertEvent
	ruleQueue     chan *AlertRule
	alertQueue    chan *Alert
	
	// State management
	mu            sync.RWMutex
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
	
	// Metrics
	metrics       *AlertMetrics
	logger        logger.Logger
}

// AlertConfig is defined in config.go

// AlertTemplate defines reusable alert templates
type AlertTemplate struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Subject     string                 `json:"subject"`
	Body        string                 `json:"body"`
	Channels    []string               `json:"channels"`
	Variables   []string               `json:"variables"`
	Metadata    map[string]interface{} `json:"metadata"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// AlertProcessor processes alerts through different channels
type AlertProcessor interface {
	Process(alert *Alert) error
	GetName() string
	IsEnabled() bool
	GetConfig() map[string]interface{}
	SetConfig(config map[string]interface{}) error
}

// AlertMetrics tracks alert system statistics
type AlertMetrics struct {
	TotalRules        int                    `json:"total_rules"`
	ActiveRules       int                    `json:"active_rules"`
	TotalAlerts       int                    `json:"total_alerts"`
	ProcessedAlerts   int                    `json:"processed_alerts"`
	FailedAlerts      int                    `json:"failed_alerts"`
	AlertsByType      map[string]int         `json:"alerts_by_type"`
	AlertsBySeverity  map[AlertSeverity]int  `json:"alerts_by_severity"`
	AlertsByChannel   map[string]int         `json:"alerts_by_channel"`
	ProcessingTime    time.Duration          `json:"processing_time"`
	LastProcessed     time.Time              `json:"last_processed"`
	QueueSize         int                    `json:"queue_size"`
	mu                sync.RWMutex
}

// NewAlertEngine creates a new alert engine
func NewAlertEngine(config *AlertConfig, logger logger.Logger) *AlertEngine {
	if config == nil {
		config = DefaultAlertConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	ae := &AlertEngine{
		config:        config,
		rules:         make(map[string]*AlertRule),
		channels:      make(map[string]AlertChannel),
		processors:    make(map[string]AlertProcessor),
		templates:     make(map[string]*AlertTemplate),
		subscriptions: make(map[string][]string),
		eventQueue:    make(chan *AlertEvent, config.QueueSize),
		ruleQueue:     make(chan *AlertRule, 100),
		alertQueue:    make(chan *Alert, config.QueueSize),
		ctx:           ctx,
		cancel:        cancel,
		metrics: &AlertMetrics{
			AlertsByType:     make(map[string]int),
			AlertsBySeverity: make(map[AlertSeverity]int),
			AlertsByChannel:  make(map[string]int),
		},
		logger: logger,
	}

	// Start workers
	for i := 0; i < config.MaxWorkers; i++ {
		ae.wg.Add(1)
		go ae.eventWorker(i)
	}

	ae.wg.Add(1)
	go ae.ruleWorker()

	ae.wg.Add(1)
	go ae.alertWorker()

	// Start cleanup worker
	if config.CleanupInterval > 0 {
		ae.wg.Add(1)
		go ae.cleanupWorker()
	}

	// Start metrics collection
	if config.EnableMetrics {
		ae.wg.Add(1)
		go ae.metricsWorker()
	}

	return ae
}

// ProcessEvent processes an alert event
func (ae *AlertEngine) ProcessEvent(event *AlertEvent) error {
	if !ae.config.Enabled {
		return nil
	}

	select {
	case ae.eventQueue <- event:
		return nil
	case <-ae.ctx.Done():
		return fmt.Errorf("alert engine is shutting down")
	default:
		return fmt.Errorf("event queue is full")
	}
}

// AddRule adds a new alert rule
func (ae *AlertEngine) AddRule(rule *AlertRule) error {
	ae.mu.Lock()
	defer ae.mu.Unlock()

	// Validate rule
	if err := ae.validateRule(rule); err != nil {
		return fmt.Errorf("invalid rule: %w", err)
	}

	// Set rule ID if not set
	if rule.ID == "" {
		rule.ID = uuid.NewString()
	}

	// Set timestamps
	now := time.Now()
	if rule.CreatedAt.IsZero() {
		rule.CreatedAt = now
	}
	rule.UpdatedAt = now

	// Add rule
	ae.rules[rule.ID] = rule

	// Update subscriptions
	ae.updateSubscriptions(rule)

	// Update metrics
	ae.updateRuleMetrics()

	ae.logger.Info("alerts", fmt.Sprintf("Added alert rule: %s", rule.Name), map[string]interface{}{
		"rule_id": rule.ID,
		"name":    rule.Name,
		"type":    rule.EventType,
	})

	return nil
}

// UpdateRule updates an existing alert rule
func (ae *AlertEngine) UpdateRule(ruleID string, rule *AlertRule) error {
	ae.mu.Lock()
	defer ae.mu.Unlock()

	existingRule, exists := ae.rules[ruleID]
	if !exists {
		return fmt.Errorf("rule %s not found", ruleID)
	}

	// Validate updated rule
	if err := ae.validateRule(rule); err != nil {
		return fmt.Errorf("invalid rule: %w", err)
	}

	// Preserve creation time
	rule.ID = ruleID
	rule.CreatedAt = existingRule.CreatedAt
	rule.UpdatedAt = time.Now()

	// Update rule
	ae.rules[ruleID] = rule

	// Update subscriptions
	ae.updateSubscriptions(rule)

	ae.logger.Info("alerts", fmt.Sprintf("Updated alert rule: %s", rule.Name), map[string]interface{}{
		"rule_id": rule.ID,
		"name":    rule.Name,
	})

	return nil
}

// RemoveRule removes an alert rule
func (ae *AlertEngine) RemoveRule(ruleID string) error {
	ae.mu.Lock()
	defer ae.mu.Unlock()

	rule, exists := ae.rules[ruleID]
	if !exists {
		return fmt.Errorf("rule %s not found", ruleID)
	}

	// Remove from subscriptions
	ae.removeFromSubscriptions(rule)

	// Remove rule
	delete(ae.rules, ruleID)

	// Update metrics
	ae.updateRuleMetrics()

	ae.logger.Info("alerts", fmt.Sprintf("Removed alert rule: %s", rule.Name), map[string]interface{}{
		"rule_id": rule.ID,
	})

	return nil
}

// RegisterChannel registers an alert channel
func (ae *AlertEngine) RegisterChannel(name string, channel AlertChannel) {
	ae.mu.Lock()
	defer ae.mu.Unlock()
	ae.channels[name] = channel
}

// RegisterProcessor registers an alert processor
func (ae *AlertEngine) RegisterProcessor(name string, processor AlertProcessor) {
	ae.mu.Lock()
	defer ae.mu.Unlock()
	ae.processors[name] = processor
}

// AddTemplate adds an alert template
func (ae *AlertEngine) AddTemplate(template *AlertTemplate) error {
	ae.mu.Lock()
	defer ae.mu.Unlock()

	if template.ID == "" {
		template.ID = uuid.NewString()
	}

	now := time.Now()
	if template.CreatedAt.IsZero() {
		template.CreatedAt = now
	}
	template.UpdatedAt = now

	ae.templates[template.ID] = template

	ae.logger.Info("alerts", fmt.Sprintf("Added alert template: %s", template.Name), map[string]interface{}{
		"template_id": template.ID,
		"name":        template.Name,
	})

	return nil
}

// GetTemplate retrieves an alert template
func (ae *AlertEngine) GetTemplate(templateID string) (*AlertTemplate, error) {
	ae.mu.RLock()
	defer ae.mu.RUnlock()

	template, exists := ae.templates[templateID]
	if !exists {
		return nil, fmt.Errorf("template %s not found", templateID)
	}

	return template, nil
}

// GetMetrics returns current alert metrics
func (ae *AlertEngine) GetMetrics() *AlertMetrics {
	ae.metrics.mu.RLock()
	defer ae.metrics.mu.RUnlock()

	// Return a copy to avoid race conditions
	return &AlertMetrics{
		TotalRules:       ae.metrics.TotalRules,
		ActiveRules:      ae.metrics.ActiveRules,
		TotalAlerts:      ae.metrics.TotalAlerts,
		ProcessedAlerts:  ae.metrics.ProcessedAlerts,
		FailedAlerts:     ae.metrics.FailedAlerts,
		AlertsByType:     copyStringIntMap(ae.metrics.AlertsByType),
		AlertsBySeverity: copySeverityIntMap(ae.metrics.AlertsBySeverity),
		AlertsByChannel:  copyStringIntMap(ae.metrics.AlertsByChannel),
		ProcessingTime:   ae.metrics.ProcessingTime,
		LastProcessed:    ae.metrics.LastProcessed,
		QueueSize:        len(ae.eventQueue),
	}
}

// Close shuts down the alert engine
func (ae *AlertEngine) Close() error {
	ae.cancel()
	close(ae.eventQueue)
	close(ae.ruleQueue)
	close(ae.alertQueue)
	ae.wg.Wait()
	return nil
}

// Worker methods

func (ae *AlertEngine) eventWorker(id int) {
	defer ae.wg.Done()

	for {
		select {
		case event, ok := <-ae.eventQueue:
			if !ok {
				return
			}
			ae.processEvent(event)
		case <-ae.ctx.Done():
			return
		}
	}
}

func (ae *AlertEngine) ruleWorker() {
	defer ae.wg.Done()

	for {
		select {
		case rule, ok := <-ae.ruleQueue:
			if !ok {
				return
			}
			ae.processRule(rule)
		case <-ae.ctx.Done():
			return
		}
	}
}

func (ae *AlertEngine) alertWorker() {
	defer ae.wg.Done()

	for {
		select {
		case alert, ok := <-ae.alertQueue:
			if !ok {
				return
			}
			ae.processAlert(alert)
		case <-ae.ctx.Done():
			return
		}
	}
}

func (ae *AlertEngine) cleanupWorker() {
	defer ae.wg.Done()

	ticker := time.NewTicker(ae.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ae.cleanup()
		case <-ae.ctx.Done():
			return
		}
	}
}

func (ae *AlertEngine) metricsWorker() {
	defer ae.wg.Done()

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ae.logMetrics()
		case <-ae.ctx.Done():
			return
		}
	}
}

// Processing methods

func (ae *AlertEngine) processEvent(event *AlertEvent) {
	start := time.Now()

	ae.mu.RLock()
	ruleIDs := ae.subscriptions[event.Type]
	ae.mu.RUnlock()

	for _, ruleID := range ruleIDs {
		ae.mu.RLock()
		rule, exists := ae.rules[ruleID]
		ae.mu.RUnlock()

		if !exists || !rule.Enabled {
			continue
		}

		// Check cooldown
		if time.Since(rule.LastTriggered) < ae.config.CooldownPeriod {
			continue
		}

		// Evaluate rule
		if ae.evaluateRule(rule, event) {
			ae.triggerRule(rule, event)
		}
	}

	ae.updateProcessingMetrics(time.Since(start))
}

func (ae *AlertEngine) processRule(rule *AlertRule) {
	// Process rule-specific logic
	ae.logger.Debug("alerts", fmt.Sprintf("Processing rule: %s", rule.Name), map[string]interface{}{
		"rule_id": rule.ID,
	})
}

func (ae *AlertEngine) processAlert(alert *Alert) {
	start := time.Now()

	// Process alert through channels
	for _, channelName := range alert.Channels {
		ae.mu.RLock()
		channel, exists := ae.channels[channelName]
		ae.mu.RUnlock()

		if !exists {
			ae.logger.Warn("alerts", fmt.Sprintf("Channel %s not found", channelName), map[string]interface{}{
				"alert_id": alert.ID,
			})
			continue
		}

		if err := channel.Send(alert); err != nil {
			ae.logger.Error("alerts", fmt.Sprintf("Failed to send alert to channel %s", channelName), map[string]interface{}{
				"alert_id": alert.ID,
				"error":    err.Error(),
			})
			ae.updateFailedAlertMetrics()
		} else {
			ae.updateProcessedAlertMetrics()
		}
	}

	ae.updateProcessingMetrics(time.Since(start))
}

func (ae *AlertEngine) evaluateRule(rule *AlertRule, event *AlertEvent) bool {
	// Evaluate conditions
	for i := range rule.Conditions {
		if !ae.evaluateCondition(&rule.Conditions[i], event) {
			return false
		}
	}
	return true
}

func (ae *AlertEngine) evaluateCondition(condition *AlertCondition, event *AlertEvent) bool {
	// Get field value
	fieldValue := ae.getFieldValue(condition.Field, event)

	// Evaluate based on operator
	switch condition.Operator {
	case "equals":
		return fieldValue == condition.Value
	case "not_equals":
		return fieldValue != condition.Value
	case "contains":
		if str, ok := fieldValue.(string); ok {
			if val, ok := condition.Value.(string); ok {
				return contains(str, val)
			}
		}
		return false
	case "greater_than":
		return compareValues(fieldValue, condition.Value, ">")
	case "less_than":
		return compareValues(fieldValue, condition.Value, "<")
	case "exists":
		return fieldValue != nil
	case "regex":
		// Implement regex matching
		return false
	default:
		return false
	}
}

func (ae *AlertEngine) getFieldValue(field string, event *AlertEvent) interface{} {
	switch field {
	case "type":
		return event.Type
	case "severity":
		return event.Severity
	case "source":
		return event.Source
	case "message":
		return event.Message
	default:
		// Check in metadata
		if event.Metadata != nil {
			return event.Metadata[field]
		}
		return nil
	}
}

func (ae *AlertEngine) triggerRule(rule *AlertRule, event *AlertEvent) {
	// Update rule trigger time
	ae.mu.Lock()
	rule.LastTriggered = time.Now()
	rule.TriggerCount++
	ae.mu.Unlock()

	// Create alert
	alert := &Alert{
		ID:        uuid.NewString(),
		RuleID:    rule.ID,
		Type:      AlertType(rule.EventType), // Convert string to AlertType
		Severity:  rule.Severity,
		Title:     rule.Name,
		Message:   ae.formatMessage(rule, event),
		Channels:  rule.Channels,
		Metadata:  ae.mergeMetadata(rule, event),
		CreatedAt: time.Now(),
		Status:    AlertStatusActive,
	}

	// Send to alert queue
	select {
	case ae.alertQueue <- alert:
		ae.updateAlertMetrics(alert)
	default:
		ae.logger.Error("alerts", "Alert queue is full", map[string]interface{}{
			"alert_id": alert.ID,
		})
	}
}

func (ae *AlertEngine) formatMessage(rule *AlertRule, event *AlertEvent) string {
	// Use template if available
	if rule.TemplateID != "" {
		ae.mu.RLock()
		template, exists := ae.templates[rule.TemplateID]
		ae.mu.RUnlock()

		if exists {
			return ae.renderTemplate(template, event)
		}
	}

	// Use rule message or default
	if rule.Message != "" {
		return rule.Message
	}

	return fmt.Sprintf("Alert: %s", event.Message)
}

func (ae *AlertEngine) renderTemplate(template *AlertTemplate, event *AlertEvent) string {
	// Simple template rendering
	// In a real implementation, this would use a proper template engine
	message := template.Body
	for _, variable := range template.Variables {
		value := ae.getFieldValue(variable, event)
		message = replaceAll(message, "{{"+variable+"}}", fmt.Sprintf("%v", value))
	}
	return message
}

func (ae *AlertEngine) mergeMetadata(rule *AlertRule, event *AlertEvent) map[string]interface{} {
	metadata := make(map[string]interface{})

	// Add event metadata
	if event.Metadata != nil {
		for k, v := range event.Metadata {
			metadata[k] = v
		}
	}

	// Add rule metadata
	if rule.Metadata != nil {
		for k, v := range rule.Metadata {
			metadata["rule_"+k] = v
		}
	}

	// Add system metadata
	metadata["rule_id"] = rule.ID
	metadata["rule_name"] = rule.Name
	metadata["event_id"] = event.ID
	metadata["triggered_at"] = time.Now()

	return metadata
}

// Helper methods

func (ae *AlertEngine) validateRule(rule *AlertRule) error {
	if rule.Name == "" {
		return fmt.Errorf("rule name is required")
	}
	if rule.EventType == "" {
		return fmt.Errorf("event type is required")
	}
	if len(rule.Conditions) == 0 {
		return fmt.Errorf("rule must have at least one condition")
	}
	if len(rule.Channels) == 0 {
		return fmt.Errorf("rule must have at least one channel")
	}
	return nil
}

func (ae *AlertEngine) updateSubscriptions(rule *AlertRule) {
	// Remove old subscriptions
	ae.removeFromSubscriptions(rule)

	// Add new subscriptions
	ae.subscriptions[rule.EventType] = append(ae.subscriptions[rule.EventType], rule.ID)
}

func (ae *AlertEngine) removeFromSubscriptions(rule *AlertRule) {
	ruleIDs := ae.subscriptions[rule.EventType]
	for i, id := range ruleIDs {
		if id == rule.ID {
			ae.subscriptions[rule.EventType] = append(ruleIDs[:i], ruleIDs[i+1:]...)
			break
		}
	}
}

func (ae *AlertEngine) updateRuleMetrics() {
	ae.metrics.mu.Lock()
	defer ae.metrics.mu.Unlock()

	ae.metrics.TotalRules = len(ae.rules)
	ae.metrics.ActiveRules = 0
	for _, rule := range ae.rules {
		if rule.Enabled {
			ae.metrics.ActiveRules++
		}
	}
}

func (ae *AlertEngine) updateAlertMetrics(alert *Alert) {
	ae.metrics.mu.Lock()
	defer ae.metrics.mu.Unlock()

	ae.metrics.TotalAlerts++
	ae.metrics.AlertsByType[string(alert.Type)]++
	ae.metrics.AlertsBySeverity[alert.Severity]++
	for _, channel := range alert.Channels {
		ae.metrics.AlertsByChannel[channel]++
	}
}

func (ae *AlertEngine) updateProcessedAlertMetrics() {
	ae.metrics.mu.Lock()
	defer ae.metrics.mu.Unlock()
	ae.metrics.ProcessedAlerts++
}

func (ae *AlertEngine) updateFailedAlertMetrics() {
	ae.metrics.mu.Lock()
	defer ae.metrics.mu.Unlock()
	ae.metrics.FailedAlerts++
}

func (ae *AlertEngine) updateProcessingMetrics(duration time.Duration) {
	ae.metrics.mu.Lock()
	defer ae.metrics.mu.Unlock()
	ae.metrics.ProcessingTime += duration
	ae.metrics.LastProcessed = time.Now()
}

func (ae *AlertEngine) cleanup() {
	// Clean up old alerts and rules
	// Implementation would depend on specific cleanup requirements
}

func (ae *AlertEngine) logMetrics() {
	metrics := ae.GetMetrics()
	ae.logger.Info("alerts", "Alert system metrics", map[string]interface{}{
		"total_rules":       metrics.TotalRules,
		"active_rules":      metrics.ActiveRules,
		"total_alerts":      metrics.TotalAlerts,
		"processed_alerts":  metrics.ProcessedAlerts,
		"failed_alerts":     metrics.FailedAlerts,
		"queue_size":        metrics.QueueSize,
	})
}

// Helper functions

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || (len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsMiddle(s, substr))))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func compareValues(a, b interface{}, op string) bool {
	// Simple comparison implementation
	// In a real implementation, this would handle different types properly
	return false
}

func replaceAll(s, old, new string) string {
	// Simple string replacement
	// In a real implementation, this would use strings.ReplaceAll
	return s
}

func copyStringIntMap(m map[string]int) map[string]int {
	result := make(map[string]int)
	for k, v := range m {
		result[k] = v
	}
	return result
}

func copySeverityIntMap(m map[AlertSeverity]int) map[AlertSeverity]int {
	result := make(map[AlertSeverity]int)
	for k, v := range m {
		result[k] = v
	}
	return result
}

// GetDefaultAlertConfig is now defined in config.go
