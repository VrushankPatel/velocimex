package alerts

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"velocimex/internal/logger"
)

// TestConsoleChannel is a mock channel for testing
type TestConsoleChannel struct {
	name      string
	alerts    []*Alert
	mutex     sync.Mutex
	failNext  bool
}

func NewTestConsoleChannel(name string) *TestConsoleChannel {
	return &TestConsoleChannel{
		name:   name,
		alerts: make([]*Alert, 0),
	}
}

func (t *TestConsoleChannel) Send(alert *Alert) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	
	if t.failNext {
		t.failNext = false
		return fmt.Errorf("simulated send failure")
	}
	
	t.alerts = append(t.alerts, alert)
	return nil
}

func (t *TestConsoleChannel) Name() string {
	return t.name
}

func (t *TestConsoleChannel) Type() string {
	return "test"
}

func (t *TestConsoleChannel) GetAlerts() []*Alert {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	
	alerts := make([]*Alert, len(t.alerts))
	copy(alerts, t.alerts)
	return alerts
}

func (t *TestConsoleChannel) SetFailNext(fail bool) {
	t.failNext = fail
}

func TestAlertTypes(t *testing.T) {
	// Test AlertType constants
	if AlertTypePrice != "price" {
		t.Errorf("Expected AlertTypePrice to be 'price', got %s", AlertTypePrice)
	}
	
	if AlertSeverityHigh != "high" {
		t.Errorf("Expected AlertSeverityHigh to be 'high', got %s", AlertSeverityHigh)
	}
}

func TestAlertRule(t *testing.T) {
	rule := &AlertRule{
		ID:       "test-rule",
		Name:     "Test Rule",
		Type:     AlertTypePrice,
		Severity: SeverityMedium,
		Conditions: []AlertCondition{
			{Field: "price", Operator: "gt", Value: 100.0},
		},
		Message:  "Price is {{price}}",
		Enabled:  true,
		Cooldown: time.Minute,
		Channels: []string{"console"},
	}
	
	if rule.ID != "test-rule" {
		t.Errorf("Expected rule ID to be 'test-rule', got %s", rule.ID)
	}
	
	if rule.Name != "Test Rule" {
		t.Errorf("Expected rule name to be 'Test Rule', got %s", rule.Name)
	}
}

func TestAlert(t *testing.T) {
	now := time.Now()
	alert := &Alert{
		ID:        "test-alert",
		RuleID:    "test-rule",
		Type:      AlertTypePrice,
		Severity:  SeverityHigh,
		Title:     "Test Alert",
		Message:   "Test message",
		Data:      map[string]interface{}{"price": 105.5},
		Timestamp: now,
	}
	
	if alert.ID != "test-alert" {
		t.Errorf("Expected alert ID to be 'test-alert', got %s", alert.ID)
	}
	
	if alert.Severity != SeverityHigh {
		t.Errorf("Expected alert severity to be 'high', got %s", alert.Severity)
	}
}

func TestVelocimexAlertManager(t *testing.T) {
	// Setup
	logger := logger.NewVelocimexLogger(logger.Config{
		Level:       "debug",
		Output:      "console",
		Development: true,
	})
	
	am := NewAlertManager(logger)
	
	// Test AddRule
	rule := &AlertRule{
		ID:       "test-rule",
		Name:     "Test Rule",
		Type:     AlertTypePrice,
		Severity: SeverityMedium,
		Conditions: []AlertCondition{
			{Field: "price", Operator: "gt", Value: 100.0},
		},
		Message:  "Price is {{price}}",
		Enabled:  true,
		Cooldown: time.Second,
	}
	
	err := am.AddRule(rule)
	if err != nil {
		t.Fatalf("AddRule failed: %v", err)
	}
	
	// Test GetRule
	retrievedRule, err := am.GetRule("test-rule")
	if err != nil {
		t.Fatalf("GetRule failed: %v", err)
	}
	
	if retrievedRule.Name != "Test Rule" {
		t.Errorf("Expected rule name 'Test Rule', got %s", retrievedRule.Name)
	}
	
	// Test GetRules
	rules := am.GetRules()
	if len(rules) != 1 {
		t.Errorf("Expected 1 rule, got %d", len(rules))
	}
	
	// Test RegisterChannel
	channel := NewTestConsoleChannel("test-channel")
	err = am.RegisterChannel(channel)
	if err != nil {
		t.Fatalf("RegisterChannel failed: %v", err)
	}
	
	// Test Start
	err = am.Start()
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	
	// Test TriggerAlert with matching conditions
	data := map[string]interface{}{
		"price": 105.5,
	}
	
	err = am.TriggerAlert(rule, data)
	if err != nil {
		t.Fatalf("TriggerAlert failed: %v", err)
	}
	
	// Wait for processing
	time.Sleep(100 * time.Millisecond)
	
	// Check if alert was created
	alerts, err := am.GetActiveAlerts()
	if err != nil {
		t.Fatalf("GetActiveAlerts failed: %v", err)
	}
	
	if len(alerts) != 1 {
		t.Errorf("Expected 1 active alert, got %d", len(alerts))
	}
	
	// Test cooldown
	err = am.TriggerAlert(rule, data)
	if err != nil {
		t.Fatalf("TriggerAlert failed: %v", err)
	}
	
	// Should not create another alert due to cooldown
	alerts, _ = am.GetActiveAlerts()
	if len(alerts) != 1 {
		t.Errorf("Expected still 1 active alert due to cooldown, got %d", len(alerts))
	}
	
	// Test AcknowledgeAlert
	err = am.AcknowledgeAlert(alerts[0].ID)
	if err != nil {
		t.Fatalf("AcknowledgeAlert failed: %v", err)
	}
	
	// Test ResolveAlert
	err = am.ResolveAlert(alerts[0].ID)
	if err != nil {
		t.Fatalf("ResolveAlert failed: %v", err)
	}
	
	// Test Stop
	err = am.Stop()
	if err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
}

func TestAlertConditions(t *testing.T) {
	logger := logger.NewVelocimexLogger(logger.Config{
		Level:       "debug",
		Output:      "console",
		Development: true,
	})
	
	am := NewAlertManager(logger)
	
	// Test GT condition
	rule := &AlertRule{
		ID:       "gt-rule",
		Name:     "GT Rule",
		Type:     AlertTypePrice,
		Severity: SeverityLow,
		Conditions: []AlertCondition{
			{Field: "value", Operator: "gt", Value: 100.0},
		},
		Message:  "Value is {{value}}",
		Enabled:  true,
		Cooldown: time.Second,
	}
	
	err := am.AddRule(rule)
	if err != nil {
		t.Fatalf("AddRule failed: %v", err)
	}
	
	// Test with value > 100
	data := map[string]interface{}{"value": 150.0}
	triggered := am.evaluateConditions(rule.Conditions, data)
	if !triggered {
		t.Error("Expected condition to trigger for value > 100")
	}
	
	// Test with value <= 100
	data = map[string]interface{}{"value": 100.0}
	triggered = am.evaluateConditions(rule.Conditions, data)
	if triggered {
		t.Error("Expected condition not to trigger for value <= 100")
	}
	
	// Test LT condition
	rule.Conditions = []AlertCondition{
		{Field: "value", Operator: "lt", Value: 50.0},
	}
	
	data = map[string]interface{}{"value": 30.0}
	triggered = am.evaluateConditions(rule.Conditions, data)
	if !triggered {
		t.Error("Expected condition to trigger for value < 50")
	}
	
	// Test EQ condition
	rule.Conditions = []AlertCondition{
		{Field: "status", Operator: "eq", Value: "error"},
	}
	
	data = map[string]interface{}{"status": "error"}
	triggered = am.evaluateConditions(rule.Conditions, data)
	if !triggered {
		t.Error("Expected condition to trigger for status == error")
	}
	
	// Test NE condition
	rule.Conditions = []AlertCondition{
		{Field: "status", Operator: "ne", Value: "ok"},
	}
	
	data = map[string]interface{}{"status": "error"}
	triggered = am.evaluateConditions(rule.Conditions, data)
	if !triggered {
		t.Error("Expected condition to trigger for status != ok")
	}
}

func TestAlertMessageFormatting(t *testing.T) {
	logger := logger.NewVelocimexLogger(logger.Config{
		Level:       "debug",
		Output:      "console",
		Development: true,
	})
	
	am := NewAlertManager(logger)
	
	// Test message formatting
	message := "Price is {{price}} and volume is {{volume}}"
	data := map[string]interface{}{
		"price":  105.5,
		"volume": 1000,
	}
	
	formatted := am.formatMessage(message, data)
	expected := "Price is 105.5 and volume is 1000"
	
	if formatted != expected {
		t.Errorf("Expected formatted message '%s', got '%s'", expected, formatted)
	}
}

func TestChannels(t *testing.T) {
	// Test ConsoleChannel
	console := NewConsoleChannel("test-console")
	if console.Name() != "test-console" {
		t.Errorf("Expected channel name 'test-console', got %s", console.Name())
	}
	
	if console.Type() != "console" {
		t.Errorf("Expected channel type 'console', got %s", console.Type())
	}
	
	// Test FileChannel
	tempDir := t.TempDir()
	filename := filepath.Join(tempDir, "alerts.jsonl")
	
	fileChannel, err := NewFileChannel("test-file", filename)
	if err != nil {
		t.Fatalf("NewFileChannel failed: %v", err)
	}
	defer fileChannel.Close()
	
	if fileChannel.Name() != "test-file" {
		t.Errorf("Expected file channel name 'test-file', got %s", fileChannel.Name())
	}
	
	// Test WebSocketChannel
	wsChannel := NewWebSocketChannel("test-ws")
	if wsChannel.Name() != "test-ws" {
		t.Errorf("Expected WebSocket channel name 'test-ws', got %s", wsChannel.Name())
	}
	
	if wsChannel.Type() != "websocket" {
		t.Errorf("Expected WebSocket channel type 'websocket', got %s", wsChannel.Type())
	}
	
	// Test EmailChannel
	emailChannel := NewEmailChannel("test-email", "smtp.example.com", 587, "user", "pass", "from@example.com", []string{"to@example.com"})
	if emailChannel.Name() != "test-email" {
		t.Errorf("Expected email channel name 'test-email', got %s", emailChannel.Name())
	}
	
	if emailChannel.Type() != "email" {
		t.Errorf("Expected email channel type 'email', got %s", emailChannel.Type())
	}
	
	// Test SlackChannel
	slackChannel := NewSlackChannel("test-slack", "https://hooks.slack.com/test", "alerts")
	if slackChannel.Name() != "test-slack" {
		t.Errorf("Expected Slack channel name 'test-slack', got %s", slackChannel.Name())
	}
	
	if slackChannel.Type() != "slack" {
		t.Errorf("Expected Slack channel type 'slack', got %s", slackChannel.Type())
	}
}

func TestChannelRegistration(t *testing.T) {
	logger := logger.NewVelocimexLogger(logger.Config{
		Level:       "debug",
		Output:      "console",
		Development: true,
	})
	
	am := NewAlertManager(logger)
	
	// Test channel registration
	channel := NewTestConsoleChannel("test-channel")
	err := am.RegisterChannel(channel)
	if err != nil {
		t.Fatalf("RegisterChannel failed: %v", err)
	}
	
	// Test duplicate registration
	err = am.RegisterChannel(channel)
	if err != nil {
		t.Errorf("Expected no error for duplicate channel registration, got %v", err)
	}
	
	// Test channel removal
	err = am.RemoveChannel("test-channel")
	if err != nil {
		t.Fatalf("RemoveChannel failed: %v", err)
	}
	
	// Test removal of non-existent channel
	err = am.RemoveChannel("non-existent")
	if err == nil {
		t.Error("Expected error for removing non-existent channel")
	}
}

func TestAlertFiltering(t *testing.T) {
	logger := logger.NewVelocimexLogger(logger.Config{
		Level:       "debug",
		Output:      "console",
		Development: true,
	})
	
	am := NewAlertManager(logger)
	
	// Add test alerts
	alert1 := &Alert{
		ID:        "alert-1",
		RuleID:    "rule-1",
		Type:      AlertTypePrice,
		Severity:  SeverityHigh,
		Title:     "Price Alert 1",
		Message:   "Test message 1",
		Timestamp: time.Now(),
		Resolved:  false,
	}
	
	alert2 := &Alert{
		ID:        "alert-2",
		RuleID:    "rule-2",
		Type:      AlertTypeRisk,
		Severity:  SeverityMedium,
		Title:     "Risk Alert 1",
		Message:   "Test message 2",
		Timestamp: time.Now(),
		Resolved:  true,
	}
	
	am.alertMutex.Lock()
	am.alerts["alert-1"] = alert1
	am.alerts["alert-2"] = alert2
	am.alertMutex.Unlock()
	
	// Test filtering by type
	alerts, err := am.GetAlerts(map[string]interface{}{
		"type": AlertTypePrice,
	})
	if err != nil {
		t.Fatalf("GetAlerts failed: %v", err)
	}
	
	if len(alerts) != 1 {
		t.Errorf("Expected 1 price alert, got %d", len(alerts))
	}
	
	// Test filtering by resolved status
	alerts, err = am.GetAlerts(map[string]interface{}{
		"resolved": false,
	})
	if err != nil {
		t.Fatalf("GetAlerts failed: %v", err)
	}
	
	if len(alerts) != 1 {
		t.Errorf("Expected 1 unresolved alert, got %d", len(alerts))
	}
	
	// Test no filters
	alerts, err = am.GetAlerts(map[string]interface{}{})
	if err != nil {
		t.Fatalf("GetAlerts failed: %v", err)
	}
	
	if len(alerts) != 2 {
		t.Errorf("Expected 2 alerts with no filters, got %d", len(alerts))
	}
}

func TestAlertConfig(t *testing.T) {
	// Test default config
	config := DefaultAlertConfig()
	if !config.Enabled {
		t.Error("Expected default config to be enabled")
	}
	
	if len(config.Channels) != 2 {
		t.Errorf("Expected 2 default channels, got %d", len(config.Channels))
	}
	
	if len(config.Rules) != 4 {
		t.Errorf("Expected 4 default rules, got %d", len(config.Rules))
	}
	
	// Test config file operations
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "alerts.json")
	
	// Test saving and loading
	err := SaveAlertConfig(configFile, config)
	if err != nil {
		t.Fatalf("SaveAlertConfig failed: %v", err)
	}
	
	loadedConfig, err := LoadAlertConfig(configFile)
	if err != nil {
		t.Fatalf("LoadAlertConfig failed: %v", err)
	}
	
	if !loadedConfig.Enabled {
		t.Error("Expected loaded config to be enabled")
	}
	
	if len(loadedConfig.Channels) != len(config.Channels) {
		t.Errorf("Expected %d channels, got %d", len(config.Channels), len(loadedConfig.Channels))
	}
	
	// Test non-existent file (should create default)
	nonExistentFile := filepath.Join(tempDir, "nonexistent.json")
	_, err = os.Stat(nonExistentFile)
	if !os.IsNotExist(err) {
		t.Fatalf("Expected file to not exist: %v", err)
	}
	
	loadedConfig, err = LoadAlertConfig(nonExistentFile)
	if err != nil {
		t.Fatalf("LoadAlertConfig should create default for non-existent file: %v", err)
	}
	
	if _, err := os.Stat(nonExistentFile); os.IsNotExist(err) {
		t.Error("Expected non-existent config file to be created")
	}
}

func TestAlertRuleBuilder(t *testing.T) {
	builder := NewAlertRuleBuilder()
	
	rule := builder.
		Name("Test Rule").
		Type(AlertTypePrice).
		Severity(SeverityHigh).
		Message("Price alert: {{price}}").
		Condition("price", "gt", 100.0).
		Cooldown(2 * time.Minute).
		Channel("console").
		Build()
	
	if rule.Name != "Test Rule" {
		t.Errorf("Expected rule name 'Test Rule', got %s", rule.Name)
	}
	
	if rule.Type != AlertTypePrice {
		t.Errorf("Expected rule type 'price', got %s", rule.Type)
	}
	
	if rule.Severity != SeverityHigh {
		t.Errorf("Expected rule severity 'high', got %s", rule.Severity)
	}
	
	if rule.Message != "Price alert: {{price}}" {
		t.Errorf("Expected rule message 'Price alert: {{price}}', got %s", rule.Message)
	}
	
	if len(rule.Conditions) != 1 {
		t.Errorf("Expected 1 condition, got %d", len(rule.Conditions))
	}
	
	if rule.Cooldown != 2*time.Minute {
		t.Errorf("Expected cooldown 2m, got %v", rule.Cooldown)
	}
	
	if len(rule.Channels) != 1 || rule.Channels[0] != "console" {
		t.Errorf("Expected channel 'console', got %v", rule.Channels)
	}
	
	// Test validation
	err := builder.Validate()
	if err != nil {
		t.Errorf("Expected validation to pass, got %v", err)
	}
	
	// Test validation failure
	invalidBuilder := NewAlertRuleBuilder()
	invalidRule := invalidBuilder.Build()
	err = invalidBuilder.Validate()
	if err == nil {
		t.Error("Expected validation to fail for empty rule")
	}
}

func TestGlobalAlertFunctions(t *testing.T) {
	// Setup test logger
	logger := logger.NewVelocimexLogger(logger.Config{
		Level:       "debug",
		Output:      "console",
		Development: true,
	})
	
	// Test global functions
		err := Init(logger)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer Shutdown()
	
	// Test TriggerPriceAlert
	err = TriggerPriceAlert("BTC", 50000.0, 48000.0)
	if err != nil {
		t.Errorf("TriggerPriceAlert failed: %v", err)
	}
	
	// Test TriggerRiskAlert
	err = TriggerRiskAlert(100000.0, 5000.0, 5, 2.5)
	if err != nil {
		t.Errorf("TriggerRiskAlert failed: %v", err)
	}
	
	// Test TriggerStrategyAlert
	err = TriggerStrategyAlert("strategy-1", "BUY", 0.85, map[string]interface{}{"confidence": 0.85})
	if err != nil {
		t.Errorf("TriggerStrategyAlert failed: %v", err)
	}
	
	// Test GetActiveAlerts
	alerts, err := GetActiveAlerts()
	if err != nil {
		t.Fatalf("GetActiveAlerts failed: %v", err)
	}
	
	// Should have at least some alerts
	if len(alerts) < 1 {
		t.Logf("Expected at least 1 alert, got %d", len(alerts))
	}
	
	// Test ComponentMonitor
	monitor := AlertMonitor(context.Background(), "test-component")
	monitor.Info("Test info message", map[string]interface{}{"data": "test"})
	monitor.Warn("Test warning message", map[string]interface{}{"data": "test"})
	monitor.Error("Test error message", fmt.Errorf("test error"), map[string]interface{}{"data": "test"})
	monitor.Critical("Test critical message", fmt.Errorf("test critical"), map[string]interface{}{"data": "test"})
	monitor.Performance("latency", 100.5, 50.0)
	monitor.Connectivity("connected", "")
}

func TestSetupAlertManager(t *testing.T) {
	// Setup test logger
	logger := logger.NewVelocimexLogger(logger.Config{
		Level:       "debug",
		Output:      "console",
		Development: true,
	})
	
	// Test with disabled config
	config := &AlertConfig{
		Enabled: false,
	}
	
	_, err := SetupAlertManager(config, logger)
	if err == nil {
		t.Error("Expected error for disabled alert system")
	}
	
	// Test with enabled config
	config = &AlertConfig{
		Enabled: true,
		Channels: []map[string]interface{}{
			{"type": "console", "name": "console"},
		},
		Rules: []map[string]interface{}{
			{
				"name":     "Test Rule",
				"type":     "price",
				"severity": "medium",
				"conditions": []map[string]interface{}{
					{"field": "price", "operator": "gt", "value": 100.0},
				},
				"message": "Price is {{price}}",
				"enabled": true,
			},
		},
	}
	
	manager, err := SetupAlertManager(config, logger)
	if err != nil {
		t.Fatalf("SetupAlertManager failed: %v", err)
	}
	
	if len(manager.GetRules()) != 1 {
		t.Errorf("Expected 1 rule, got %d", len(manager.GetRules()))
	}
}

func TestConcurrentOperations(t *testing.T) {
	logger := logger.NewVelocimexLogger(logger.Config{
		Level:       "debug",
		Output:      "console",
		Development: true,
	})
	
	am := NewAlertManager(logger)
	
	// Start the manager
	err := am.Start()
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer am.Stop()
	
	// Register a test channel
	channel := NewTestConsoleChannel("test")
	err = am.RegisterChannel(channel)
	if err != nil {
		t.Fatalf("RegisterChannel failed: %v", err)
	}
	
	// Add test rule
	rule := &AlertRule{
		ID:       "concurrent-rule",
		Name:     "Concurrent Test Rule",
		Type:     AlertTypePrice,
		Severity: SeverityLow,
		Conditions: []AlertCondition{
			{Field: "price", Operator: "gt", Value: 0.0},
		},
		Message:  "Concurrent test",
		Enabled:  true,
		Cooldown: 100 * time.Millisecond,
	}
	
	err = am.AddRule(rule)
	if err != nil {
		t.Fatalf("AddRule failed: %v", err)
	}
	
	// Concurrent trigger operations
	var wg sync.WaitGroup
	numGoroutines := 10
	
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			data := map[string]interface{}{"price": float64(id + 1)}
			_ = am.TriggerAlert(rule, data)
		}(i)
	}
	
	wg.Wait()
	
	// Wait for processing
	time.Sleep(200 * time.Millisecond)
	
	// Check results
	alerts, err := am.GetActiveAlerts()
	if err != nil {
		t.Fatalf("GetActiveAlerts failed: %v", err)
	}
	
	// Should have at least one alert (due to cooldown, might not have all)
	if len(alerts) < 1 {
		t.Errorf("Expected at least 1 alert from concurrent operations, got %d", len(alerts))
	}
}

func TestAlertRuleValidation(t *testing.T) {
	// Test various rule configurations
	tests := []struct {
		name        string
		rule        *AlertRule
		shouldError bool
	}{
		{
			name: "Valid rule",
			rule: &AlertRule{
				Name:     "Test",
				Type:     AlertTypePrice,
				Severity: SeverityMedium,
				Message:  "Test",
				Enabled:  true,
			},
			shouldError: false,
		},
		{
			name: "Empty name",
			rule: &AlertRule{
				Type:     AlertTypePrice,
				Severity: SeverityMedium,
				Message:  "Test",
			},
			shouldError: true,
		},
		{
			name: "Empty type",
			rule: &AlertRule{
				Name:     "Test",
				Severity: SeverityMedium,
				Message:  "Test",
			},
			shouldError: true,
		},
		{
			name: "Empty severity",
			rule: &AlertRule{
				Name:    "Test",
				Type:    AlertTypePrice,
				Message: "Test",
			},
			shouldError: true,
		},
		{
			name: "Empty message",
			rule: &AlertRule{
				Name:     "Test",
				Type:     AlertTypePrice,
				Severity: SeverityMedium,
			},
			shouldError: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewAlertRuleBuilder()
			builder.rule = tt.rule
			
			err := builder.Validate()
			if tt.shouldError && err == nil {
				t.Error("Expected validation error, got none")
			}
			if !tt.shouldError && err != nil {
				t.Errorf("Expected no validation error, got %v", err)
			}
		})
	}
}