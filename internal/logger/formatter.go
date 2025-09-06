package logger

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Formatter interface for different log formats
type Formatter interface {
	Format(entry LogEntry) ([]byte, error)
}

// JSONFormatter formats logs as JSON
type JSONFormatter struct {
	PrettyPrint bool
	IncludeTime bool
}

// NewJSONFormatter creates a new JSON formatter
func NewJSONFormatter(prettyPrint, includeTime bool) *JSONFormatter {
	return &JSONFormatter{
		PrettyPrint: prettyPrint,
		IncludeTime: includeTime,
	}
}

// Format formats a log entry as JSON
func (f *JSONFormatter) Format(entry LogEntry) ([]byte, error) {
	// Create a map for JSON output
	output := map[string]interface{}{
		"timestamp": entry.Timestamp.Format(time.RFC3339Nano),
		"level":     entry.Level.String(),
		"message":   entry.Message,
		"component": entry.Component,
	}

	// Add trace ID if present
	if entry.TraceID != "" {
		output["trace_id"] = entry.TraceID
	}

	// Add fields
	if entry.Fields != nil && len(entry.Fields) > 0 {
		for k, v := range entry.Fields {
			output[k] = v
		}
	}

	// Marshal to JSON
	if f.PrettyPrint {
		return json.MarshalIndent(output, "", "  ")
	}
	return json.Marshal(output)
}

// TextFormatter formats logs as human-readable text
type TextFormatter struct {
	IncludeTime bool
	TimeFormat  string
}

// NewTextFormatter creates a new text formatter
func NewTextFormatter(includeTime bool) *TextFormatter {
	return &TextFormatter{
		IncludeTime: includeTime,
		TimeFormat:  "2006-01-02 15:04:05.000",
	}
}

// Format formats a log entry as text
func (f *TextFormatter) Format(entry LogEntry) ([]byte, error) {
	var parts []string

	// Add timestamp
	if f.IncludeTime {
		parts = append(parts, entry.Timestamp.Format(f.TimeFormat))
	}

	// Add level
	levelStr := entry.Level.String()
	parts = append(parts, fmt.Sprintf("[%s]", levelStr))

	// Add component
	parts = append(parts, fmt.Sprintf("[%s]", entry.Component))

	// Add trace ID if present
	if entry.TraceID != "" {
		parts = append(parts, fmt.Sprintf("[trace:%s]", entry.TraceID))
	}

	// Add message
	parts = append(parts, entry.Message)

	// Add fields
	if entry.Fields != nil && len(entry.Fields) > 0 {
		var fieldParts []string
		for k, v := range entry.Fields {
			fieldParts = append(fieldParts, fmt.Sprintf("%s=%v", k, v))
		}
		if len(fieldParts) > 0 {
			parts = append(parts, fmt.Sprintf("{%s}", strings.Join(fieldParts, ", ")))
		}
	}

	return []byte(strings.Join(parts, " ") + "\n"), nil
}

// AuditFormatter formats audit entries
type AuditFormatter struct {
	PrettyPrint bool
}

// NewAuditFormatter creates a new audit formatter
func NewAuditFormatter(prettyPrint bool) *AuditFormatter {
	return &AuditFormatter{
		PrettyPrint: prettyPrint,
	}
}

// Format formats an audit entry
func (f *AuditFormatter) Format(entry AuditEntry) ([]byte, error) {
	// Create a map for JSON output
	output := map[string]interface{}{
		"timestamp":  entry.Timestamp.Format(time.RFC3339Nano),
		"event_type": string(entry.EventType),
	}

	// Add optional fields
	if entry.UserID != "" {
		output["user_id"] = entry.UserID
	}
	if entry.SessionID != "" {
		output["session_id"] = entry.SessionID
	}
	if entry.TradeID != "" {
		output["trade_id"] = entry.TradeID
	}
	if entry.OrderID != "" {
		output["order_id"] = entry.OrderID
	}
	if entry.Symbol != "" {
		output["symbol"] = entry.Symbol
	}
	if entry.Quantity != "" {
		output["quantity"] = entry.Quantity
	}
	if entry.Price != "" {
		output["price"] = entry.Price
	}
	if entry.Side != "" {
		output["side"] = entry.Side
	}
	if entry.Strategy != "" {
		output["strategy"] = entry.Strategy
	}
	if entry.BeforeState != nil {
		output["before_state"] = entry.BeforeState
	}
	if entry.AfterState != nil {
		output["after_state"] = entry.AfterState
	}
	if entry.Metadata != nil {
		output["metadata"] = entry.Metadata
	}
	if entry.IPAddress != "" {
		output["ip_address"] = entry.IPAddress
	}
	if entry.UserAgent != "" {
		output["user_agent"] = entry.UserAgent
	}

	// Marshal to JSON
	if f.PrettyPrint {
		return json.MarshalIndent(output, "", "  ")
	}
	return json.Marshal(output)
}

// LogstashFormatter formats logs for Logstash/ELK stack
type LogstashFormatter struct {
	ServiceName string
	Environment string
}

// NewLogstashFormatter creates a new Logstash formatter
func NewLogstashFormatter(serviceName, environment string) *LogstashFormatter {
	return &LogstashFormatter{
		ServiceName: serviceName,
		Environment: environment,
	}
}

// Format formats a log entry for Logstash
func (f *LogstashFormatter) Format(entry LogEntry) ([]byte, error) {
	// Create Logstash-compatible structure
	output := map[string]interface{}{
		"@timestamp": entry.Timestamp.Format(time.RFC3339Nano),
		"@version":   "1",
		"level":      entry.Level.String(),
		"message":    entry.Message,
		"component":  entry.Component,
		"service":    f.ServiceName,
		"environment": f.Environment,
	}

	// Add trace ID if present
	if entry.TraceID != "" {
		output["trace_id"] = entry.TraceID
	}

	// Add fields
	if entry.Fields != nil && len(entry.Fields) > 0 {
		for k, v := range entry.Fields {
			output[k] = v
		}
	}

	return json.Marshal(output)
}

// FluentdFormatter formats logs for Fluentd
type FluentdFormatter struct {
	Tag string
}

// NewFluentdFormatter creates a new Fluentd formatter
func NewFluentdFormatter(tag string) *FluentdFormatter {
	return &FluentdFormatter{
		Tag: tag,
	}
}

// Format formats a log entry for Fluentd
func (f *FluentdFormatter) Format(entry LogEntry) ([]byte, error) {
	// Create Fluentd-compatible structure
	output := map[string]interface{}{
		"timestamp": entry.Timestamp.Unix(),
		"level":     entry.Level.String(),
		"message":   entry.Message,
		"component": entry.Component,
		"tag":       f.Tag,
	}

	// Add trace ID if present
	if entry.TraceID != "" {
		output["trace_id"] = entry.TraceID
	}

	// Add fields
	if entry.Fields != nil && len(entry.Fields) > 0 {
		for k, v := range entry.Fields {
			output[k] = v
		}
	}

	return json.Marshal(output)
}

// CSVFormatter formats logs as CSV
type CSVFormatter struct {
	IncludeHeaders bool
	Headers        []string
}

// NewCSVFormatter creates a new CSV formatter
func NewCSVFormatter(includeHeaders bool) *CSVFormatter {
	return &CSVFormatter{
		IncludeHeaders: includeHeaders,
		Headers: []string{
			"timestamp",
			"level",
			"component",
			"message",
			"trace_id",
		},
	}
}

// Format formats a log entry as CSV
func (f *CSVFormatter) Format(entry LogEntry) ([]byte, error) {
	var parts []string

	// Add timestamp
	parts = append(parts, entry.Timestamp.Format(time.RFC3339Nano))

	// Add level
	parts = append(parts, entry.Level.String())

	// Add component
	parts = append(parts, entry.Component)

	// Add message (escape quotes)
	message := strings.ReplaceAll(entry.Message, "\"", "\"\"")
	parts = append(parts, fmt.Sprintf("\"%s\"", message))

	// Add trace ID
	parts = append(parts, entry.TraceID)

	// Add fields as additional columns
	if entry.Fields != nil && len(entry.Fields) > 0 {
		for _, v := range entry.Fields {
			parts = append(parts, fmt.Sprintf("%v", v))
		}
	}

	return []byte(strings.Join(parts, ",") + "\n"), nil
}

// GetFormatter returns a formatter based on the format string
func GetFormatter(format string, options map[string]interface{}) Formatter {
	switch strings.ToLower(format) {
	case "json":
		prettyPrint := false
		includeTime := true
		if val, ok := options["pretty_print"].(bool); ok {
			prettyPrint = val
		}
		if val, ok := options["include_time"].(bool); ok {
			includeTime = val
		}
		return NewJSONFormatter(prettyPrint, includeTime)
	case "text":
		includeTime := true
		if val, ok := options["include_time"].(bool); ok {
			includeTime = val
		}
		return NewTextFormatter(includeTime)
	case "logstash":
		serviceName := "velocimex"
		environment := "production"
		if val, ok := options["service_name"].(string); ok {
			serviceName = val
		}
		if val, ok := options["environment"].(string); ok {
			environment = val
		}
		return NewLogstashFormatter(serviceName, environment)
	case "fluentd":
		tag := "velocimex.log"
		if val, ok := options["tag"].(string); ok {
			tag = val
		}
		return NewFluentdFormatter(tag)
	case "csv":
		includeHeaders := false
		if val, ok := options["include_headers"].(bool); ok {
			includeHeaders = val
		}
		return NewCSVFormatter(includeHeaders)
	default:
		return NewTextFormatter(true)
	}
}
