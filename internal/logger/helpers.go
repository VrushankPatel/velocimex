package logger

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"time"
)

// Package-level convenience functions
var (
	Debug = GetLogger().Debug
	Info  = GetLogger().Info
	Warn  = GetLogger().Warn
	Error = GetLogger().Error
	Fatal = GetLogger().Fatal
)

// WithContext returns a logger with trace ID from context
func WithContext(ctx context.Context) *VelocimexLogger {
	traceID := GetTraceID(ctx)
	return GetLogger().WithTrace(traceID).(*VelocimexLogger)
}

// GetTraceID extracts trace ID from context
func GetTraceID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if traceID, ok := ctx.Value(traceIDKey{}).(string); ok {
		return traceID
	}
	return ""
}

// WithTraceID adds trace ID to context
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceIDKey{}, traceID)
}

type traceIDKey struct{}

// LogError is a convenience function for logging errors with stack trace
func LogError(component string, err error, fields ...map[string]interface{}) {
	if err == nil {
		return
	}

	mergedFields := make(map[string]interface{})
	if len(fields) > 0 {
		for _, field := range fields {
			for k, v := range field {
				mergedFields[k] = v
			}
		}
	}
	mergedFields["error"] = err.Error()
	mergedFields["stack_trace"] = getStackTrace()

	GetLogger().Error(component, err.Error(), mergedFields)
}

// LogMethodEntry logs method entry with parameters
func LogMethodEntry(component string, method string, params ...interface{}) {
	fields := map[string]interface{}{
		"method": method,
		"params": params,
		"action": "enter",
	}
	GetLogger().Debug(component, fmt.Sprintf("Entering %s", method), fields)
}

// LogMethodExit logs method exit with results
func LogMethodExit(component string, method string, duration time.Duration, results ...interface{}) {
	fields := map[string]interface{}{
		"method":   method,
		"duration": duration.Milliseconds(),
		"results":  results,
		"action":   "exit",
	}
	GetLogger().Debug(component, fmt.Sprintf("Exiting %s", method), fields)
}

// LogPerformance logs performance metrics
func LogPerformance(component string, operation string, duration time.Duration, metadata interface{}) {
	fields := map[string]interface{}{
		"operation": operation,
		"duration":  duration.Milliseconds(),
		"metadata":  metadata,
	}
	GetLogger().Info(component, fmt.Sprintf("Performance: %s took %v", operation, duration), fields)
}

// LogMarketData logs market data updates
func LogMarketData(component string, symbol string, dataType string, data interface{}) {
	fields := map[string]interface{}{
		"symbol":    symbol,
		"data_type": dataType,
		"data":      data,
	}
	GetLogger().Debug(component, fmt.Sprintf("Market data update for %s", symbol), fields)
}

// LogTradeExecution logs trade execution details
func LogTradeExecution(component string, tradeID string, symbol string, side string, quantity string, price string, metadata interface{}) {
	GetLogger().LogTrade(tradeID, symbol, side, quantity, price, metadata)
}

// LogOrderPlacement logs order placement details
func LogOrderPlacement(component string, orderID string, symbol string, side string, quantity string, price string, orderType string, metadata interface{}) {
	GetLogger().LogOrder(orderID, symbol, side, quantity, price, orderType, metadata)
}

// LogRiskEvent logs risk-related events
func LogRiskEvent(component string, eventType string, symbol string, details interface{}) {
	GetLogger().LogRiskEvent(eventType, symbol, details)
}

// LogStrategySignal logs strategy signals
func LogStrategySignal(component string, strategy string, signal string, metadata interface{}) {
	GetLogger().LogStrategySignal(strategy, signal, metadata)
}

// getStackTrace returns a formatted stack trace
func getStackTrace() string {
	var stack [4096]byte
	n := runtime.Stack(stack[:], false)
	lines := strings.Split(string(stack[:n]), "\n")
	
	// Skip the first few lines (runtime.Stack and this function)
	if len(lines) > 4 {
		return strings.Join(lines[4:], "\n")
	}
	return ""
}

// ComponentLogger provides a component-specific logger
type ComponentLogger struct {
	component string
	logger    *VelocimexLogger
}

// NewComponentLogger creates a new component-specific logger
func NewComponentLogger(component string) *ComponentLogger {
	return &ComponentLogger{
		component: component,
		logger:    GetLogger(),
	}
}

// WithContext returns a component logger with context
func (c *ComponentLogger) WithContext(ctx context.Context) *ComponentLogger {
	traceID := GetTraceID(ctx)
	return &ComponentLogger{
		component: c.component,
		logger:    c.logger.WithTrace(traceID).(*VelocimexLogger),
	}
}

// Debug logs a debug message
func (c *ComponentLogger) Debug(message string, fields ...map[string]interface{}) {
	c.logger.Debug(c.component, message, fields...)
}

// Info logs an info message
func (c *ComponentLogger) Info(message string, fields ...map[string]interface{}) {
	c.logger.Info(c.component, message, fields...)
}

// Warn logs a warning message
func (c *ComponentLogger) Warn(message string, fields ...map[string]interface{}) {
	c.logger.Warn(c.component, message, fields...)
}

// Error logs an error message
func (c *ComponentLogger) Error(message string, fields ...map[string]interface{}) {
	c.logger.Error(c.component, message, fields...)
}

// Fatal logs a fatal message
func (c *ComponentLogger) Fatal(message string, fields ...map[string]interface{}) {
	c.logger.Fatal(c.component, message, fields...)
}

// LogError logs an error with stack trace
func (c *ComponentLogger) LogError(err error, fields ...map[string]interface{}) {
	LogError(c.component, err, fields...)
}

// LogMethodEntry logs method entry
func (c *ComponentLogger) LogMethodEntry(method string, params ...interface{}) {
	LogMethodEntry(c.component, method, params...)
}

// LogMethodExit logs method exit
func (c *ComponentLogger) LogMethodExit(method string, duration time.Duration, results ...interface{}) {
	LogMethodExit(c.component, method, duration, results...)
}

// LogPerformance logs performance metrics
func (c *ComponentLogger) LogPerformance(operation string, duration time.Duration, metadata interface{}) {
	LogPerformance(c.component, operation, duration, metadata)
}