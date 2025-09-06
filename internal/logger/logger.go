package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// VelocimexLogger implements the Logger and AuditLogger interfaces
type VelocimexLogger struct {
	config      *Config
	logger      *log.Logger
	auditLogger *log.Logger
	formatter   Formatter
	mu          sync.RWMutex
	traceID     string
	rotation    *RotatingWriter
}

// New creates a new VelocimexLogger instance
func New(config *Config) (*VelocimexLogger, error) {
	l := &VelocimexLogger{
		config: config,
	}

	// Set up main logger
	var output io.Writer
	switch config.Output {
	case "stdout":
		output = os.Stdout
	case "stderr":
		output = os.Stderr
	default:
		// File output with rotation support
		file, err := l.openLogFile(config.Output)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file: %w", err)
		}
		output = file
	}

	l.logger = log.New(output, "", 0)

	// Set up audit logger if enabled
	if config.EnableAudit {
		auditOutput, err := l.openLogFile(config.AuditFile)
		if err != nil {
			return nil, fmt.Errorf("failed to open audit file: %w", err)
		}
		l.auditLogger = log.New(auditOutput, "", 0)
	}

	return l, nil
}

func (l *VelocimexLogger) openLogFile(path string) (*os.File, error) {
	if path == "" {
		return nil, fmt.Errorf("log file path is empty")
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	return os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
}

// Debug logs a debug message
func (l *VelocimexLogger) Debug(component string, message string, fields ...map[string]interface{}) {
	l.log(DEBUG, component, message, fields...)
}

// Info logs an info message
func (l *VelocimexLogger) Info(component string, message string, fields ...map[string]interface{}) {
	l.log(INFO, component, message, fields...)
}

// Warn logs a warning message
func (l *VelocimexLogger) Warn(component string, message string, fields ...map[string]interface{}) {
	l.log(WARN, component, message, fields...)
}

// Error logs an error message
func (l *VelocimexLogger) Error(component string, message string, fields ...map[string]interface{}) {
	l.log(ERROR, component, message, fields...)
}

// Fatal logs a fatal message and exits
func (l *VelocimexLogger) Fatal(component string, message string, fields ...map[string]interface{}) {
	l.log(FATAL, component, message, fields...)
	os.Exit(1)
}

// WithTrace creates a new logger with trace ID
func (l *VelocimexLogger) WithTrace(traceID string) Logger {
	newLogger := *l
	newLogger.traceID = traceID
	return &newLogger
}

// log is the internal logging function
func (l *VelocimexLogger) log(level LogLevel, component string, message string, fields ...map[string]interface{}) {
	if level < l.config.Level {
		return
	}

	entry := LogEntry{
		Timestamp: time.Now().UTC(),
		Level:     level,
		Message:   message,
		Component: component,
		Fields:    make(map[string]interface{}),
		TraceID:   l.traceID,
	}

	// Merge fields
	if len(fields) > 0 {
		for _, field := range fields {
			for k, v := range field {
				entry.Fields[k] = v
			}
		}
	}

	var output string
	switch l.config.Format {
	case "json":
		data, err := json.Marshal(entry)
		if err != nil {
			output = fmt.Sprintf("{\"timestamp\":\"%s\",\"level\":\"%s\",\"component\":\"%s\",\"message\":\"%s\",\"error\":\"failed to marshal log entry\"}",
				entry.Timestamp.Format(time.RFC3339), level.String(), component, message)
		} else {
			output = string(data)
		}
	default: // text format
		output = fmt.Sprintf("%s [%s] %s: %s", entry.Timestamp.Format(time.RFC3339), level.String(), component, message)
		if len(entry.Fields) > 0 {
			data, _ := json.Marshal(entry.Fields)
			output += " " + string(data)
		}
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	l.logger.Println(output)
}

// LogEvent logs an audit event
func (l *VelocimexLogger) LogEvent(entry AuditEntry) {
	if !l.config.EnableAudit || l.auditLogger == nil {
		return
	}

	entry.Timestamp = time.Now().UTC()
	data, err := json.Marshal(entry)
	if err != nil {
		l.Error("audit", "failed to marshal audit entry", map[string]interface{}{
			"error": err.Error(),
			"event_type": entry.EventType,
		})
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	l.auditLogger.Println(string(data))
}

// LogTrade logs a trade execution
func (l *VelocimexLogger) LogTrade(tradeID, symbol, side, quantity, price string, metadata interface{}) {
	l.LogEvent(AuditEntry{
		EventType: TradeExecuted,
		TradeID:   tradeID,
		Symbol:    symbol,
		Side:      side,
		Quantity:  quantity,
		Price:     price,
		Metadata:  metadata,
	})
}

// LogOrder logs an order placement
func (l *VelocimexLogger) LogOrder(orderID, symbol, side, quantity, price string, orderType string, metadata interface{}) {
	l.LogEvent(AuditEntry{
		EventType: OrderPlaced,
		OrderID:   orderID,
		Symbol:    symbol,
		Side:      side,
		Quantity:  quantity,
		Price:     price,
		Metadata: map[string]interface{}{
			"order_type": orderType,
			"details":    metadata,
		},
	})
}

// LogRiskEvent logs a risk-related event
func (l *VelocimexLogger) LogRiskEvent(eventType string, symbol string, details interface{}) {
	l.LogEvent(AuditEntry{
		EventType: RiskLimitBreached,
		Symbol:    symbol,
		Metadata: map[string]interface{}{
			"event_type": eventType,
			"details":    details,
		},
	})
}

// LogStrategySignal logs a strategy signal
func (l *VelocimexLogger) LogStrategySignal(strategy string, signal string, metadata interface{}) {
	l.LogEvent(AuditEntry{
		EventType: StrategySignal,
		Strategy:  strategy,
		Metadata: map[string]interface{}{
			"signal":  signal,
			"details": metadata,
		},
	})
}

// GetTraceID returns the trace ID
func (l *VelocimexLogger) GetTraceID() string {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.traceID
}

// Flush flushes the logger
func (l *VelocimexLogger) Flush() {
	if l.rotation != nil {
		l.rotation.Sync()
	}
}

// Close closes the logger and associated files
func (l *VelocimexLogger) Close() error {
	if l.rotation != nil {
		return l.rotation.Close()
	}
	return nil
}