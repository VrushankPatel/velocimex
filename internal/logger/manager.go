package logger

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// LogManager manages multiple loggers and provides centralized logging functionality
type LogManager struct {
	config        *Config
	loggers       map[string]*VelocimexLogger
	auditLogger   *VelocimexLogger
	metricsLogger *VelocimexLogger
	accessLogger  *VelocimexLogger
	mu            sync.RWMutex
	ctx           context.Context
	cancel        context.CancelFunc
	metrics       *LogMetrics
}

// LogMetrics tracks logging statistics
type LogMetrics struct {
	TotalLogs      int64            `json:"total_logs"`
	LogsByLevel    map[LogLevel]int64 `json:"logs_by_level"`
	LogsByComponent map[string]int64  `json:"logs_by_component"`
	AuditEvents    int64            `json:"audit_events"`
	Errors         int64            `json:"errors"`
	Warnings       int64            `json:"warnings"`
	mu             sync.RWMutex
}

// NewLogManager creates a new log manager with multiple specialized loggers
func NewLogManager(config *Config) (*LogManager, error) {
	ctx, cancel := context.WithCancel(context.Background())
	
	lm := &LogManager{
		config:  config,
		loggers: make(map[string]*VelocimexLogger),
		ctx:     ctx,
		cancel:  cancel,
		metrics: &LogMetrics{
			LogsByLevel:    make(map[LogLevel]int64),
			LogsByComponent: make(map[string]int64),
		},
	}

	// Create main application logger
	mainLogger, err := New(config)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create main logger: %w", err)
	}
	lm.loggers["main"] = mainLogger

	// Create audit logger
	if config.EnableAudit {
		auditConfig := *config
		auditConfig.Output = config.AuditFile
		auditConfig.Level = INFO // Audit logs are always at least INFO level
		auditLogger, err := New(&auditConfig)
		if err != nil {
			cancel()
			return nil, fmt.Errorf("failed to create audit logger: %w", err)
		}
		lm.auditLogger = auditLogger
	}

	// Create metrics logger
	metricsConfig := *config
	metricsConfig.Output = "logs/metrics.log"
	metricsConfig.Level = INFO
	metricsLogger, err := New(&metricsConfig)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create metrics logger: %w", err)
	}
	lm.metricsLogger = metricsLogger

	// Create access logger for HTTP requests
	accessConfig := *config
	accessConfig.Output = "logs/access.log"
	accessConfig.Level = INFO
	accessLogger, err := New(&accessConfig)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create access logger: %w", err)
	}
	lm.accessLogger = accessLogger

	// Start metrics collection
	go lm.collectMetrics()

	return lm, nil
}

// GetLogger returns a logger for a specific component
func (lm *LogManager) GetLogger(component string) Logger {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	if logger, exists := lm.loggers[component]; exists {
		return logger
	}

	// Create new logger for component
	componentConfig := *lm.config
	componentConfig.Output = fmt.Sprintf("logs/%s.log", strings.ToLower(component))
	componentLogger, err := New(&componentConfig)
	if err != nil {
		// Fallback to main logger
		return lm.loggers["main"]
	}

	lm.loggers[component] = componentLogger
	return componentLogger
}

// GetAuditLogger returns the audit logger
func (lm *LogManager) GetAuditLogger() AuditLogger {
	return lm.auditLogger
}

// GetMetricsLogger returns the metrics logger
func (lm *LogManager) GetMetricsLogger() Logger {
	return lm.metricsLogger
}

// GetAccessLogger returns the access logger
func (lm *LogManager) GetAccessLogger() Logger {
	return lm.accessLogger
}

// LogWithContext logs a message with context information
func (lm *LogManager) LogWithContext(ctx context.Context, level LogLevel, component, message string, fields ...map[string]interface{}) {
	// Extract trace ID from context
	traceID := GetTraceID(ctx)
	
	// Get component logger
	logger := lm.GetLogger(component)
	
	// Add trace ID to fields
	allFields := make(map[string]interface{})
	if len(fields) > 0 {
		for k, v := range fields[0] {
			allFields[k] = v
		}
	}
	allFields["trace_id"] = traceID
	allFields["goroutine_id"] = getGoroutineID()
	allFields["caller"] = getCallerInfo()

	// Log based on level
	switch level {
	case DEBUG:
		logger.Debug(component, message, allFields)
	case INFO:
		logger.Info(component, message, allFields)
	case WARN:
		logger.Warn(component, message, allFields)
	case ERROR:
		logger.Error(component, message, allFields)
	case FATAL:
		logger.Fatal(component, message, allFields)
	}

	// Update metrics
	lm.updateMetrics(level, component)
}

// LogAuditEvent logs an audit event
func (lm *LogManager) LogAuditEvent(ctx context.Context, eventType AuditEventType, details map[string]interface{}) {
	if lm.auditLogger == nil {
		return
	}

	traceID := GetTraceID(ctx)
	entry := AuditEntry{
		Timestamp: time.Now(),
		EventType: eventType,
		Metadata:  details,
	}

	// Add trace ID if available
	if traceID != "" {
		entry.Metadata = map[string]interface{}{
			"trace_id": traceID,
			"details":  details,
		}
	}

	lm.auditLogger.LogEvent(entry)
	lm.updateAuditMetrics()
}

// LogHTTPRequest logs an HTTP request
func (lm *LogManager) LogHTTPRequest(ctx context.Context, method, path, userAgent, ip string, statusCode int, duration time.Duration, size int64) {
	if lm.accessLogger == nil {
		return
	}

	fields := map[string]interface{}{
		"method":      method,
		"path":        path,
		"status_code": statusCode,
		"duration_ms": duration.Milliseconds(),
		"size_bytes":  size,
		"user_agent":  userAgent,
		"ip_address":  ip,
		"trace_id":    GetTraceID(ctx),
	}

	level := INFO
	if statusCode >= 400 {
		level = WARN
	}
	if statusCode >= 500 {
		level = ERROR
	}

	lm.LogWithContext(ctx, level, "http", fmt.Sprintf("%s %s %d", method, path, statusCode), fields)
}

// LogTradeEvent logs a trade-related event
func (lm *LogManager) LogTradeEvent(ctx context.Context, eventType string, tradeID, symbol, side, quantity, price string, metadata map[string]interface{}) {
	fields := map[string]interface{}{
		"event_type": eventType,
		"trade_id":   tradeID,
		"symbol":     symbol,
		"side":       side,
		"quantity":   quantity,
		"price":      price,
		"metadata":   metadata,
	}

	lm.LogWithContext(ctx, INFO, "trading", fmt.Sprintf("Trade %s: %s %s %s @ %s", eventType, side, quantity, symbol, price), fields)
	
	// Also log as audit event
	lm.LogAuditEvent(ctx, TradeExecuted, fields)
}

// LogOrderEvent logs an order-related event
func (lm *LogManager) LogOrderEvent(ctx context.Context, eventType string, orderID, symbol, side, quantity, price, orderType string, metadata map[string]interface{}) {
	fields := map[string]interface{}{
		"event_type": eventType,
		"order_id":   orderID,
		"symbol":     symbol,
		"side":       side,
		"quantity":   quantity,
		"price":      price,
		"order_type": orderType,
		"metadata":   metadata,
	}

	lm.LogWithContext(ctx, INFO, "orders", fmt.Sprintf("Order %s: %s %s %s @ %s", eventType, side, quantity, symbol, price), fields)
	
	// Also log as audit event
	auditEventType := OrderPlaced
	if eventType == "cancelled" {
		auditEventType = OrderCancelled
	}
	lm.LogAuditEvent(ctx, auditEventType, fields)
}

// LogRiskEvent logs a risk management event
func (lm *LogManager) LogRiskEvent(ctx context.Context, eventType, symbol string, details map[string]interface{}) {
	fields := map[string]interface{}{
		"event_type": eventType,
		"symbol":     symbol,
		"details":    details,
	}

	lm.LogWithContext(ctx, WARN, "risk", fmt.Sprintf("Risk event: %s for %s", eventType, symbol), fields)
	
	// Also log as audit event
	lm.LogAuditEvent(ctx, RiskLimitBreached, fields)
}

// LogStrategyEvent logs a strategy-related event
func (lm *LogManager) LogStrategyEvent(ctx context.Context, strategy, signal string, metadata map[string]interface{}) {
	fields := map[string]interface{}{
		"strategy": strategy,
		"signal":   signal,
		"metadata": metadata,
	}

	lm.LogWithContext(ctx, INFO, "strategy", fmt.Sprintf("Strategy %s: %s", strategy, signal), fields)
	
	// Also log as audit event
	lm.LogAuditEvent(ctx, StrategySignal, fields)
}

// LogSystemEvent logs a system-level event
func (lm *LogManager) LogSystemEvent(ctx context.Context, level LogLevel, component, message string, fields ...map[string]interface{}) {
	lm.LogWithContext(ctx, level, component, message, fields...)
}

// LogError logs an error with stack trace
func (lm *LogManager) LogError(ctx context.Context, component string, err error, message string, fields ...map[string]interface{}) {
	allFields := make(map[string]interface{})
	if len(fields) > 0 {
		for k, v := range fields[0] {
			allFields[k] = v
		}
	}
	
	allFields["error"] = err.Error()
	allFields["stack_trace"] = getStackTrace()

	lm.LogWithContext(ctx, ERROR, component, message, allFields)
}

// LogPerformance logs performance metrics
func (lm *LogManager) LogPerformance(ctx context.Context, operation string, duration time.Duration, metadata map[string]interface{}) {
	fields := map[string]interface{}{
		"operation":     operation,
		"duration_ms":   duration.Milliseconds(),
		"duration_ns":   duration.Nanoseconds(),
		"metadata":      metadata,
	}

	level := INFO
	if duration > 100*time.Millisecond {
		level = WARN
	}
	if duration > 1*time.Second {
		level = ERROR
	}

	lm.LogWithContext(ctx, level, "performance", fmt.Sprintf("Performance: %s took %v", operation, duration), fields)
}

// GetMetrics returns current logging metrics
func (lm *LogManager) GetMetrics() *LogMetrics {
	lm.metrics.mu.RLock()
	defer lm.metrics.mu.RUnlock()
	
	// Create copies of the maps to avoid race conditions
	logsByLevel := make(map[LogLevel]int64)
	for k, v := range lm.metrics.LogsByLevel {
		logsByLevel[k] = v
	}
	
	logsByComponent := make(map[string]int64)
	for k, v := range lm.metrics.LogsByComponent {
		logsByComponent[k] = v
	}
	
	// Return a copy to avoid race conditions
	return &LogMetrics{
		TotalLogs:       lm.metrics.TotalLogs,
		LogsByLevel:     logsByLevel,
		LogsByComponent: logsByComponent,
		AuditEvents:     lm.metrics.AuditEvents,
		Errors:          lm.metrics.Errors,
		Warnings:        lm.metrics.Warnings,
	}
}

// Flush flushes all loggers
func (lm *LogManager) Flush() {
	lm.mu.RLock()
	defer lm.mu.RUnlock()
	
	for _, logger := range lm.loggers {
		logger.Flush()
	}
	
	if lm.auditLogger != nil {
		lm.auditLogger.Flush()
	}
	if lm.metricsLogger != nil {
		lm.metricsLogger.Flush()
	}
	if lm.accessLogger != nil {
		lm.accessLogger.Flush()
	}
}

// Close closes all loggers
func (lm *LogManager) Close() error {
	lm.cancel()
	lm.Flush()
	
	lm.mu.Lock()
	defer lm.mu.Unlock()
	
	var lastErr error
	for _, logger := range lm.loggers {
		if err := logger.Close(); err != nil {
			lastErr = err
		}
	}
	
	if lm.auditLogger != nil {
		if err := lm.auditLogger.Close(); err != nil {
			lastErr = err
		}
	}
	if lm.metricsLogger != nil {
		if err := lm.metricsLogger.Close(); err != nil {
			lastErr = err
		}
	}
	if lm.accessLogger != nil {
		if err := lm.accessLogger.Close(); err != nil {
			lastErr = err
		}
	}
	
	return lastErr
}

// Helper functions

func (lm *LogManager) updateMetrics(level LogLevel, component string) {
	lm.metrics.mu.Lock()
	defer lm.metrics.mu.Unlock()
	
	lm.metrics.TotalLogs++
	lm.metrics.LogsByLevel[level]++
	lm.metrics.LogsByComponent[component]++
	
	if level == ERROR {
		lm.metrics.Errors++
	} else if level == WARN {
		lm.metrics.Warnings++
	}
}

func (lm *LogManager) updateAuditMetrics() {
	lm.metrics.mu.Lock()
	defer lm.metrics.mu.Unlock()
	lm.metrics.AuditEvents++
}

func (lm *LogManager) collectMetrics() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			lm.logMetrics()
		case <-lm.ctx.Done():
			return
		}
	}
}

func (lm *LogManager) logMetrics() {
	metrics := lm.GetMetrics()
	fields := map[string]interface{}{
		"total_logs":        metrics.TotalLogs,
		"logs_by_level":     metrics.LogsByLevel,
		"logs_by_component": metrics.LogsByComponent,
		"audit_events":      metrics.AuditEvents,
		"errors":            metrics.Errors,
		"warnings":          metrics.Warnings,
	}
	
	lm.LogWithContext(lm.ctx, INFO, "metrics", "Logging metrics", fields)
}

func getGoroutineID() int {
	var buf [64]byte
	n := runtime.Stack(buf[:], false)
	id := 0
	fmt.Sscanf(string(buf[:n]), "goroutine %d", &id)
	return id
}

func getCallerInfo() string {
	_, file, line, ok := runtime.Caller(3)
	if !ok {
		return "unknown"
	}
	return fmt.Sprintf("%s:%d", filepath.Base(file), line)
}


func copyMap(m interface{}) interface{} {
	switch v := m.(type) {
	case map[LogLevel]int64:
		result := make(map[LogLevel]int64)
		for k, v := range v {
			result[k] = v
		}
		return result
	case map[string]int64:
		result := make(map[string]int64)
		for k, v := range v {
			result[k] = v
		}
		return result
	default:
		return m
	}
}

func copyStringMap(m map[string]int64) map[string]int64 {
	result := make(map[string]int64)
	for k, v := range m {
		result[k] = v
	}
	return result
}
