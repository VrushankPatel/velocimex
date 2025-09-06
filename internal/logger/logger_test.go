package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLogLevelString(t *testing.T) {
	tests := []struct {
		level    LogLevel
		expected string
	}{
		{DEBUG, "DEBUG"},
		{INFO, "INFO"},
		{WARN, "WARN"},
		{ERROR, "ERROR"},
		{FATAL, "FATAL"},
		{LogLevel(99), "UNKNOWN"},
	}

	for _, test := range tests {
		if got := test.level.String(); got != test.expected {
			t.Errorf("LogLevel.String() = %v, want %v", got, test.expected)
		}
	}
}

func TestNewLogger(t *testing.T) {
	// Test with default config
	config := &Config{
		Level:       DEBUG,
		Format:      "json",
		Output:      "stdout",
		EnableAudit: false,
	}

	logger, err := New(config)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if logger == nil {
		t.Fatal("New() returned nil logger")
	}
}

func TestLoggerWithFileOutput(t *testing.T) {
	// Create temporary directory
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test.log")

	config := &Config{
		Level:       DEBUG,
		Format:      "json",
		Output:      logFile,
		EnableAudit: false,
	}

	logger, err := New(config)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Test logging
	logger.Info("test", "test message")

	// Verify file was created
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		t.Error("Log file was not created")
	}
}

func TestLoggerWithAudit(t *testing.T) {
	// Create temporary directory
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test.log")
	auditFile := filepath.Join(tempDir, "audit.log")

	config := &Config{
		Level:       DEBUG,
		Format:      "json",
		Output:      logFile,
		EnableAudit: true,
		AuditFile:   auditFile,
	}

	logger, err := New(config)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Test audit logging
	logger.LogEvent(AuditEntry{
		EventType: TradeExecuted,
		TradeID:   "test-trade-123",
		Symbol:    "BTCUSDT",
	})

	// Verify audit file was created
	if _, err := os.Stat(auditFile); os.IsNotExist(err) {
		t.Error("Audit file was not created")
	}
}

func TestLogLevelFiltering(t *testing.T) {
	var buf bytes.Buffer

	config := &Config{
		Level:       WARN,
		Format:      "text",
		Output:      "stdout",
		EnableAudit: false,
	}

	logger := &VelocimexLogger{
		config: config,
		logger: log.New(&buf, "", 0),
	}

	// These should not appear
	logger.Debug("test", "debug message")
	logger.Info("test", "info message")

	// These should appear
	logger.Warn("test", "warn message")
	logger.Error("test", "error message")

	output := buf.String()
	if strings.Contains(output, "debug message") {
		t.Error("Debug message was logged despite WARN level")
	}
	if strings.Contains(output, "info message") {
		t.Error("Info message was logged despite WARN level")
	}
	if !strings.Contains(output, "warn message") {
		t.Error("Warn message was not logged")
	}
	if !strings.Contains(output, "error message") {
		t.Error("Error message was not logged")
	}
}

func TestJSONFormat(t *testing.T) {
	var buf bytes.Buffer

	config := &Config{
		Level:       DEBUG,
		Format:      "json",
		Output:      "stdout",
		EnableAudit: false,
	}

	logger := &VelocimexLogger{
		config: config,
		logger: log.New(&buf, "", 0),
	}

	fields := map[string]interface{}{
		"key1": "value1",
		"key2": 42,
	}

	logger.Info("test", "test message", fields)

	var entry LogEntry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("Failed to unmarshal JSON log: %v", err)
	}

	if entry.Component != "test" {
		t.Errorf("Component = %v, want test", entry.Component)
	}
	if entry.Message != "test message" {
		t.Errorf("Message = %v, want test message", entry.Message)
	}
	if entry.Level != INFO {
		t.Errorf("Level = %v, want INFO", entry.Level)
	}
	if entry.Fields["key1"] != "value1" {
		t.Errorf("Fields[key1] = %v, want value1", entry.Fields["key1"])
	}
}

func TestWithTrace(t *testing.T) {
	config := &Config{
		Level:       DEBUG,
		Format:      "json",
		Output:      "stdout",
		EnableAudit: false,
	}

	logger, _ := New(config)
	traceLogger := logger.WithTrace("test-trace-123")

	if traceLogger.(*VelocimexLogger).traceID != "test-trace-123" {
		t.Error("Trace ID was not set correctly")
	}
}

func TestAuditFunctions(t *testing.T) {
	var buf bytes.Buffer

	config := &Config{
		Level:       DEBUG,
		Format:      "json",
		Output:      "stdout",
		EnableAudit: true,
	}

	logger := &VelocimexLogger{
		config:      config,
		logger:      log.New(&buf, "", 0),
		auditLogger: log.New(&buf, "", 0),
	}

	// Test LogTrade
	logger.LogTrade("trade-123", "BTCUSDT", "BUY", "1.0", "50000", map[string]interface{}{"fee": 0.001})

	// Test LogOrder
	logger.LogOrder("order-456", "ETHUSDT", "SELL", "2.0", "3000", "LIMIT", nil)

	// Test LogRiskEvent
	logger.LogRiskEvent("position_limit_exceeded", "BTCUSDT", map[string]interface{}{"current": 100, "limit": 50})

	// Test LogStrategySignal
	logger.LogStrategySignal("arbitrage", "buy_signal", map[string]interface{}{"spread": 0.5})

	// Verify audit entries
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 4 {
		t.Errorf("Expected 4 audit entries, got %d", len(lines))
	}

	// Check that each line is valid JSON
	for i, line := range lines {
		var entry AuditEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			t.Errorf("Failed to unmarshal audit entry %d: %v", i, err)
		}
	}
}

func TestContextFunctions(t *testing.T) {
	ctx := context.Background()
	ctx = WithTraceID(ctx, "test-trace-123")

	traceID := GetTraceID(ctx)
	if traceID != "test-trace-123" {
		t.Errorf("GetTraceID = %v, want test-trace-123", traceID)
	}

	// Test WithContext
	logger := WithContext(ctx)
	if logger.GetTraceID() != "test-trace-123" {
		t.Error("Trace ID not set in logger from context")
	}
}

func TestComponentLogger(t *testing.T) {
	var buf bytes.Buffer

	config := &Config{
		Level:       DEBUG,
		Format:      "text",
		Output:      "stdout",
		EnableAudit: false,
	}

	logger := &VelocimexLogger{
		config: config,
		logger: log.New(&buf, "", 0),
	}

	// Override global logger
	globalLogger = logger

	compLogger := NewComponentLogger("test-component")
	compLogger.Info("test message")

	output := buf.String()
	if !strings.Contains(output, "test-component") {
		t.Error("Component name not found in log output")
	}
	if !strings.Contains(output, "test message") {
		t.Error("Message not found in log output")
	}
}

func TestSetupLoggingDirectory(t *testing.T) {
	tempDir := t.TempDir()
	logDir := filepath.Join(tempDir, "test_logs")

	err := SetupLoggingDirectory(logDir)
	if err != nil {
		t.Fatalf("SetupLoggingDirectory() error = %v", err)
	}

	// Verify directory was created
	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		t.Error("Logging directory was not created")
	}
}

func TestLoadConfig(t *testing.T) {
	// Save original env vars
	oldLevel := os.Getenv("LOG_LEVEL")
	oldFormat := os.Getenv("LOG_FORMAT")

	// Clean up
	defer func() {
		os.Setenv("LOG_LEVEL", oldLevel)
		os.Setenv("LOG_FORMAT", oldFormat)
	}()

	// Set test env vars
	os.Setenv("LOG_LEVEL", "DEBUG")
	os.Setenv("LOG_FORMAT", "json")

	config := LoadConfig()

	if config.Level != DEBUG {
		t.Errorf("Level = %v, want DEBUG", config.Level)
	}
	if config.Format != "json" {
		t.Errorf("Format = %v, want json", config.Format)
	}
}

func TestLogMethodEntryExit(t *testing.T) {
	var buf bytes.Buffer

	config := &Config{
		Level:       DEBUG,
		Format:      "text",
		Output:      "stdout",
		EnableAudit: false,
	}

	logger := &VelocimexLogger{
		config: config,
		logger: log.New(&buf, "", 0),
	}

	// Override global logger
	globalLogger = logger

	LogMethodEntry("test-component", "testMethod", "param1", 42)
	LogMethodExit("test-component", "testMethod", 100*time.Millisecond, "result1", true)

	output := buf.String()
	if !strings.Contains(output, "Entering testMethod") {
		t.Error("Method entry not logged")
	}
	if !strings.Contains(output, "Exiting testMethod") {
		t.Error("Method exit not logged")
	}
	if !strings.Contains(output, "param1") {
		t.Error("Parameters not logged")
	}
	if !strings.Contains(output, "result1") {
		t.Error("Results not logged")
	}
}

func TestLogPerformance(t *testing.T) {
	var buf bytes.Buffer

	config := &Config{
		Level:       INFO,
		Format:      "text",
		Output:      "stdout",
		EnableAudit: false,
	}

	logger := &VelocimexLogger{
		config: config,
		logger: log.New(&buf, "", 0),
	}

	// Override global logger
	globalLogger = logger

	LogPerformance("test-component", "database_query", 150*time.Millisecond, map[string]string{"query": "SELECT * FROM trades"})

	output := buf.String()
	if !strings.Contains(output, "Performance: database_query took 150") {
		t.Error("Performance log not found")
	}
}