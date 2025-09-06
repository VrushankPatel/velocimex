package logger

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// LogMonitor monitors log patterns and triggers alerts
type LogMonitor struct {
	config     *MonitoringConfig
	rules      []MonitoringRule
	alerts     chan Alert
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	mu         sync.RWMutex
	metrics    *MonitoringMetrics
	processors map[string]AlertProcessor
}

// MonitoringConfig holds configuration for log monitoring
type MonitoringConfig struct {
	Enabled        bool          `yaml:"enabled"`
	CheckInterval  time.Duration `yaml:"check_interval"`
	BufferSize     int           `yaml:"buffer_size"`
	MaxAlerts      int           `yaml:"max_alerts"`
	AlertCooldown  time.Duration `yaml:"alert_cooldown"`
	EnableMetrics  bool          `yaml:"enable_metrics"`
	RulesFile      string        `yaml:"rules_file"`
	AlertChannels  []string      `yaml:"alert_channels"`
}

// MonitoringRule defines a rule for log monitoring
type MonitoringRule struct {
	ID          string            `yaml:"id"`
	Name        string            `yaml:"name"`
	Description string            `yaml:"description"`
	Enabled     bool              `yaml:"enabled"`
	Severity    AlertSeverity     `yaml:"severity"`
	Conditions  []RuleCondition   `yaml:"conditions"`
	Actions     []RuleAction      `yaml:"actions"`
	Cooldown    time.Duration     `yaml:"cooldown"`
	LastTriggered time.Time       `yaml:"last_triggered"`
	TriggerCount int              `yaml:"trigger_count"`
	Metadata    map[string]interface{} `yaml:"metadata"`
}

// RuleCondition defines a condition for a monitoring rule
type RuleCondition struct {
	Field    string      `yaml:"field"`
	Operator string      `yaml:"operator"` // "equals", "contains", "regex", "greater_than", "less_than", "exists"
	Value    interface{} `yaml:"value"`
	Duration time.Duration `yaml:"duration,omitempty"`
	Count    int         `yaml:"count,omitempty"`
}

// RuleAction defines an action to take when a rule is triggered
type RuleAction struct {
	Type    string                 `yaml:"type"` // "alert", "log", "webhook", "email"
	Config  map[string]interface{} `yaml:"config"`
	Enabled bool                   `yaml:"enabled"`
}

// Alert represents a triggered alert
type Alert struct {
	ID          string                 `json:"id"`
	RuleID      string                 `json:"rule_id"`
	RuleName    string                 `json:"rule_name"`
	Severity    AlertSeverity          `json:"severity"`
	Message     string                 `json:"message"`
	Timestamp   time.Time              `json:"timestamp"`
	Source      string                 `json:"source"`
	Metadata    map[string]interface{} `json:"metadata"`
	Resolved    bool                   `json:"resolved"`
	ResolvedAt  *time.Time             `json:"resolved_at,omitempty"`
}

// AlertSeverity represents the severity of an alert
type AlertSeverity int

const (
	SeverityInfo AlertSeverity = iota
	SeverityWarning
	SeverityError
	SeverityCritical
)

func (s AlertSeverity) String() string {
	switch s {
	case SeverityInfo:
		return "info"
	case SeverityWarning:
		return "warning"
	case SeverityError:
		return "error"
	case SeverityCritical:
		return "critical"
	default:
		return "unknown"
	}
}

// AlertProcessor processes alerts
type AlertProcessor interface {
	Process(alert Alert) error
	GetName() string
	IsEnabled() bool
}

// MonitoringMetrics tracks monitoring statistics
type MonitoringMetrics struct {
	TotalRules       int                    `json:"total_rules"`
	ActiveRules      int                    `json:"active_rules"`
	TriggeredRules   int                    `json:"triggered_rules"`
	TotalAlerts      int                    `json:"total_alerts"`
	AlertsBySeverity map[AlertSeverity]int  `json:"alerts_by_severity"`
	ProcessedAlerts  int                    `json:"processed_alerts"`
	FailedAlerts     int                    `json:"failed_alerts"`
	LastCheck        time.Time              `json:"last_check"`
	mu               sync.RWMutex
}

// NewLogMonitor creates a new log monitor
func NewLogMonitor(config *MonitoringConfig) *LogMonitor {
	if config == nil {
		config = GetDefaultMonitoringConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	lm := &LogMonitor{
		config:     config,
		rules:      make([]MonitoringRule, 0),
		alerts:     make(chan Alert, config.BufferSize),
		ctx:        ctx,
		cancel:     cancel,
		metrics:    &MonitoringMetrics{AlertsBySeverity: make(map[AlertSeverity]int)},
		processors: make(map[string]AlertProcessor),
	}

	// Start monitoring workers
	for i := 0; i < 2; i++ {
		lm.wg.Add(1)
		go lm.monitoringWorker(i)
	}

	// Start alert processor
	lm.wg.Add(1)
	go lm.alertProcessor()

	return lm
}

// AddRule adds a monitoring rule
func (lm *LogMonitor) AddRule(rule MonitoringRule) error {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	// Validate rule
	if err := lm.validateRule(rule); err != nil {
		return fmt.Errorf("invalid rule: %w", err)
	}

	// Check for duplicate ID
	for _, existingRule := range lm.rules {
		if existingRule.ID == rule.ID {
			return fmt.Errorf("rule with ID %s already exists", rule.ID)
		}
	}

	lm.rules = append(lm.rules, rule)
	lm.updateMetrics()
	return nil
}

// RemoveRule removes a monitoring rule
func (lm *LogMonitor) RemoveRule(ruleID string) error {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	for i, rule := range lm.rules {
		if rule.ID == ruleID {
			lm.rules = append(lm.rules[:i], lm.rules[i+1:]...)
			lm.updateMetrics()
			return nil
		}
	}

	return fmt.Errorf("rule with ID %s not found", ruleID)
}

// UpdateRule updates an existing monitoring rule
func (lm *LogMonitor) UpdateRule(ruleID string, updatedRule MonitoringRule) error {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	for i, rule := range lm.rules {
		if rule.ID == ruleID {
			// Validate updated rule
			if err := lm.validateRule(updatedRule); err != nil {
				return fmt.Errorf("invalid rule: %w", err)
			}

			lm.rules[i] = updatedRule
			lm.updateMetrics()
			return nil
		}
	}

	return fmt.Errorf("rule with ID %s not found", ruleID)
}

// RegisterProcessor registers an alert processor
func (lm *LogMonitor) RegisterProcessor(name string, processor AlertProcessor) {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	lm.processors[name] = processor
}

// ProcessLogEntry processes a log entry against monitoring rules
func (lm *LogMonitor) ProcessLogEntry(entry LogEntry) {
	if !lm.config.Enabled {
		return
	}

	lm.mu.RLock()
	rules := make([]MonitoringRule, len(lm.rules))
	copy(rules, lm.rules)
	lm.mu.RUnlock()

	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}

		// Check cooldown
		if time.Since(rule.LastTriggered) < rule.Cooldown {
			continue
		}

		// Check if rule matches
		if lm.evaluateRule(rule, entry) {
			lm.triggerRule(rule, entry)
		}
	}
}

// GetMetrics returns current monitoring metrics
func (lm *LogMonitor) GetMetrics() *MonitoringMetrics {
	lm.metrics.mu.RLock()
	defer lm.metrics.mu.RUnlock()

	// Return a copy to avoid race conditions
	return &MonitoringMetrics{
		TotalRules:       lm.metrics.TotalRules,
		ActiveRules:      lm.metrics.ActiveRules,
		TriggeredRules:   lm.metrics.TriggeredRules,
		TotalAlerts:      lm.metrics.TotalAlerts,
		AlertsBySeverity: copyAlertSeverityMap(lm.metrics.AlertsBySeverity),
		ProcessedAlerts:  lm.metrics.ProcessedAlerts,
		FailedAlerts:     lm.metrics.FailedAlerts,
		LastCheck:        lm.metrics.LastCheck,
	}
}

// Close shuts down the log monitor
func (lm *LogMonitor) Close() error {
	lm.cancel()
	close(lm.alerts)
	lm.wg.Wait()
	return nil
}

// Helper methods

func (lm *LogMonitor) validateRule(rule MonitoringRule) error {
	if rule.ID == "" {
		return fmt.Errorf("rule ID is required")
	}
	if rule.Name == "" {
		return fmt.Errorf("rule name is required")
	}
	if len(rule.Conditions) == 0 {
		return fmt.Errorf("rule must have at least one condition")
	}
	if len(rule.Actions) == 0 {
		return fmt.Errorf("rule must have at least one action")
	}

	// Validate conditions
	for i, condition := range rule.Conditions {
		if condition.Field == "" {
			return fmt.Errorf("condition %d: field is required", i)
		}
		if condition.Operator == "" {
			return fmt.Errorf("condition %d: operator is required", i)
		}
	}

	// Validate actions
	for i, action := range rule.Actions {
		if action.Type == "" {
			return fmt.Errorf("action %d: type is required", i)
		}
	}

	return nil
}

func (lm *LogMonitor) evaluateRule(rule MonitoringRule, entry LogEntry) bool {
	// All conditions must be met for the rule to trigger
	for _, condition := range rule.Conditions {
		if !lm.evaluateCondition(condition, entry) {
			return false
		}
	}
	return true
}

func (lm *LogMonitor) evaluateCondition(condition RuleCondition, entry LogEntry) bool {
	var fieldValue interface{}

	// Get field value based on condition field
	switch condition.Field {
	case "level":
		fieldValue = entry.Level.String()
	case "component":
		fieldValue = entry.Component
	case "message":
		fieldValue = entry.Message
	case "trace_id":
		fieldValue = entry.TraceID
	default:
		// Check in fields map
		if entry.Fields != nil {
			fieldValue = entry.Fields[condition.Field]
		}
	}

	// Evaluate condition
	switch condition.Operator {
	case "equals":
		return fieldValue == condition.Value
	case "contains":
		if str, ok := fieldValue.(string); ok {
			if val, ok := condition.Value.(string); ok {
				return contains(str, val)
			}
		}
		return false
	case "regex":
		// Implementation would require regex compilation
		return false
	case "greater_than":
		return compareValues(fieldValue, condition.Value, ">")
	case "less_than":
		return compareValues(fieldValue, condition.Value, "<")
	case "exists":
		return fieldValue != nil
	default:
		return false
	}
}

func (lm *LogMonitor) triggerRule(rule MonitoringRule, entry LogEntry) {
	// Update rule trigger count and time
	lm.mu.Lock()
	for i, r := range lm.rules {
		if r.ID == rule.ID {
			lm.rules[i].LastTriggered = time.Now()
			lm.rules[i].TriggerCount++
			break
		}
	}
	lm.mu.Unlock()

	// Create alert
	alert := Alert{
		ID:        fmt.Sprintf("%s-%d", rule.ID, time.Now().UnixNano()),
		RuleID:    rule.ID,
		RuleName:  rule.Name,
		Severity:  rule.Severity,
		Message:   fmt.Sprintf("Rule '%s' triggered: %s", rule.Name, rule.Description),
		Timestamp: time.Now(),
		Source:    "log_monitor",
		Metadata: map[string]interface{}{
			"log_entry": entry,
			"rule":      rule,
		},
	}

	// Send alert to channel
	select {
	case lm.alerts <- alert:
		lm.updateAlertMetrics(alert)
	default:
		// Channel is full, log error
		fmt.Printf("Alert channel is full, dropping alert: %s\n", alert.ID)
	}
}

func (lm *LogMonitor) monitoringWorker(id int) {
	defer lm.wg.Done()

	ticker := time.NewTicker(lm.config.CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			lm.performPeriodicChecks()
		case <-lm.ctx.Done():
			return
		}
	}
}

func (lm *LogMonitor) alertProcessor() {
	defer lm.wg.Done()

	for {
		select {
		case alert, ok := <-lm.alerts:
			if !ok {
				return
			}
			lm.processAlert(alert)
		case <-lm.ctx.Done():
			return
		}
	}
}

func (lm *LogMonitor) processAlert(alert Alert) {
	lm.mu.RLock()
	processors := make([]AlertProcessor, 0, len(lm.processors))
	for _, processor := range lm.processors {
		if processor.IsEnabled() {
			processors = append(processors, processor)
		}
	}
	lm.mu.RUnlock()

	// Process alert with all enabled processors
	for _, processor := range processors {
		if err := processor.Process(alert); err != nil {
			lm.updateFailedAlertMetrics()
			continue
		}
	}

	lm.updateProcessedAlertMetrics()
}

func (lm *LogMonitor) performPeriodicChecks() {
	// Update last check time
	lm.metrics.mu.Lock()
	lm.metrics.LastCheck = time.Now()
	lm.metrics.mu.Unlock()

	// Perform any periodic checks here
	// For example, check for rules that haven't triggered in a while
}

func (lm *LogMonitor) updateMetrics() {
	lm.metrics.mu.Lock()
	defer lm.metrics.mu.Unlock()

	lm.metrics.TotalRules = len(lm.rules)
	lm.metrics.ActiveRules = 0
	for _, rule := range lm.rules {
		if rule.Enabled {
			lm.metrics.ActiveRules++
		}
	}
}

func (lm *LogMonitor) updateAlertMetrics(alert Alert) {
	lm.metrics.mu.Lock()
	defer lm.metrics.mu.Unlock()

	lm.metrics.TotalAlerts++
	lm.metrics.AlertsBySeverity[alert.Severity]++
}

func (lm *LogMonitor) updateProcessedAlertMetrics() {
	lm.metrics.mu.Lock()
	defer lm.metrics.mu.Unlock()
	lm.metrics.ProcessedAlerts++
}

func (lm *LogMonitor) updateFailedAlertMetrics() {
	lm.metrics.mu.Lock()
	defer lm.metrics.mu.Unlock()
	lm.metrics.FailedAlerts++
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

func copyAlertSeverityMap(m map[AlertSeverity]int) map[AlertSeverity]int {
	result := make(map[AlertSeverity]int)
	for k, v := range m {
		result[k] = v
	}
	return result
}

// GetDefaultMonitoringConfig returns a default monitoring configuration
func GetDefaultMonitoringConfig() *MonitoringConfig {
	return &MonitoringConfig{
		Enabled:        true,
		CheckInterval:  30 * time.Second,
		BufferSize:     1000,
		MaxAlerts:      10000,
		AlertCooldown:  5 * time.Minute,
		EnableMetrics:  true,
		RulesFile:      "config/monitoring_rules.yaml",
		AlertChannels:  []string{"log", "console"},
	}
}
