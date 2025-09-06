package logger

import (
	"time"
)

// LogLevel represents the severity level of log messages
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
	FATAL
)

func (l LogLevel) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	case FATAL:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// LogEntry represents a structured log entry
type LogEntry struct {
	Timestamp time.Time              `json:"timestamp"`
	Level     LogLevel               `json:"level"`
	Message   string                 `json:"message"`
	Component string                 `json:"component"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
	TraceID   string                 `json:"trace_id,omitempty"`
}

// AuditEventType represents different types of audit events
type AuditEventType string

const (
	TradeExecuted     AuditEventType = "trade_executed"
	OrderPlaced       AuditEventType = "order_placed"
	OrderCancelled    AuditEventType = "order_cancelled"
	PositionOpened    AuditEventType = "position_opened"
	PositionClosed    AuditEventType = "position_closed"
	RiskLimitBreached AuditEventType = "risk_limit_breached"
	StrategySignal    AuditEventType = "strategy_signal"
	SystemError       AuditEventType = "system_error"
	UserAction        AuditEventType = "user_action"
)

// AuditEntry represents an audit trail entry
type AuditEntry struct {
	Timestamp   time.Time      `json:"timestamp"`
	EventType   AuditEventType `json:"event_type"`
	UserID      string         `json:"user_id,omitempty"`
	SessionID   string         `json:"session_id,omitempty"`
	TradeID     string         `json:"trade_id,omitempty"`
	OrderID     string         `json:"order_id,omitempty"`
	Symbol      string         `json:"symbol,omitempty"`
	Quantity    string         `json:"quantity,omitempty"`
	Price       string         `json:"price,omitempty"`
	Side        string         `json:"side,omitempty"`
	Strategy    string         `json:"strategy,omitempty"`
	BeforeState interface{}    `json:"before_state,omitempty"`
	AfterState  interface{}    `json:"after_state,omitempty"`
	Metadata    interface{}    `json:"metadata,omitempty"`
	IPAddress   string         `json:"ip_address,omitempty"`
	UserAgent   string         `json:"user_agent,omitempty"`
}

// Logger interface defines the methods for structured logging
type Logger interface {
	Debug(component string, message string, fields ...map[string]interface{})
	Info(component string, message string, fields ...map[string]interface{})
	Warn(component string, message string, fields ...map[string]interface{})
	Error(component string, message string, fields ...map[string]interface{})
	Fatal(component string, message string, fields ...map[string]interface{})
	WithTrace(traceID string) Logger
}

// AuditLogger interface defines methods for audit trail logging
type AuditLogger interface {
	LogEvent(entry AuditEntry)
	LogTrade(tradeID, symbol, side, quantity, price string, metadata interface{})
	LogOrder(orderID, symbol, side, quantity, price string, orderType string, metadata interface{})
	LogRiskEvent(eventType string, symbol string, details interface{})
	LogStrategySignal(strategy string, signal string, metadata interface{})
}

// Config holds logger configuration
type Config struct {
	Level            LogLevel `yaml:"level"`
	Format           string   `yaml:"format"` // "json" or "text"
	Output           string   `yaml:"output"` // "stdout", "stderr", or file path
	EnableAudit      bool     `yaml:"enable_audit"`
	AuditFile        string   `yaml:"audit_file"`
	MaxFileSizeMB    int      `yaml:"max_file_size_mb"`
	MaxBackupFiles   int      `yaml:"max_backup_files"`
	CompressBackups  bool     `yaml:"compress_backups"`
	EnableTrace      bool     `yaml:"enable_trace"`
	TraceHeaderName  string   `yaml:"trace_header_name"`
}