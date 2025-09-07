package logger

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// AuditSystem provides centralized audit logging functionality
type AuditSystem struct {
	logger     AuditLogger
	config     *AuditConfig
	events     chan AuditEvent
	workers    int
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	mu         sync.RWMutex
	metrics    *AuditMetrics
	processors map[AuditEventType][]AuditProcessor
}

// AuditConfig holds configuration for the audit system
type AuditConfig struct {
	Enabled           bool          `yaml:"enabled"`
	BufferSize        int           `yaml:"buffer_size"`
	Workers           int           `yaml:"workers"`
	FlushInterval     time.Duration `yaml:"flush_interval"`
	RetentionDays     int           `yaml:"retention_days"`
	CompressOldLogs   bool          `yaml:"compress_old_logs"`
	EnableMetrics     bool          `yaml:"enable_metrics"`
	EnableAlerts      bool          `yaml:"enable_alerts"`
	AlertThresholds   map[string]int `yaml:"alert_thresholds"`
	RequiredFields    []string      `yaml:"required_fields"`
	ExcludedFields    []string      `yaml:"excluded_fields"`
}

// AuditEvent represents an audit event to be processed
type AuditEvent struct {
	ID        string                 `json:"id"`
	Type      AuditEventType         `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
	Metadata  map[string]interface{} `json:"metadata"`
	Priority  AuditPriority          `json:"priority"`
	Source    string                 `json:"source"`
	UserID    string                 `json:"user_id,omitempty"`
	SessionID string                 `json:"session_id,omitempty"`
	TraceID   string                 `json:"trace_id,omitempty"`
}

// AuditPriority represents the priority of an audit event
type AuditPriority int

const (
	PriorityLow AuditPriority = iota
	PriorityNormal
	PriorityHigh
	PriorityCritical
)

func (p AuditPriority) String() string {
	switch p {
	case PriorityLow:
		return "low"
	case PriorityNormal:
		return "normal"
	case PriorityHigh:
		return "high"
	case PriorityCritical:
		return "critical"
	default:
		return "unknown"
	}
}

// AuditProcessor defines an interface for processing audit events
type AuditProcessor interface {
	Process(event AuditEvent) error
	CanProcess(eventType AuditEventType) bool
	GetName() string
}

// AuditMetrics tracks audit system statistics
type AuditMetrics struct {
	TotalEvents     int64                    `json:"total_events"`
	EventsByType    map[AuditEventType]int64 `json:"events_by_type"`
	EventsByPriority map[AuditPriority]int64  `json:"events_by_priority"`
	ProcessedEvents int64                    `json:"processed_events"`
	FailedEvents    int64                    `json:"failed_events"`
	ProcessingTime  time.Duration            `json:"processing_time"`
	LastProcessed   time.Time                `json:"last_processed"`
	mu              sync.RWMutex
}

// NewAuditSystem creates a new audit system
func NewAuditSystem(logger AuditLogger, config *AuditConfig) *AuditSystem {
	if config == nil {
		config = GetDefaultAuditConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	as := &AuditSystem{
		logger:     logger,
		config:     config,
		events:     make(chan AuditEvent, config.BufferSize),
		workers:    config.Workers,
		ctx:        ctx,
		cancel:     cancel,
		metrics:    &AuditMetrics{EventsByType: make(map[AuditEventType]int64), EventsByPriority: make(map[AuditPriority]int64)},
		processors: make(map[AuditEventType][]AuditProcessor),
	}

	// Start workers
	for i := 0; i < as.workers; i++ {
		as.wg.Add(1)
		go as.worker(i)
	}

	// Start metrics collection
	if config.EnableMetrics {
		go as.collectMetrics()
	}

	return as
}

// LogEvent logs an audit event
func (as *AuditSystem) LogEvent(ctx context.Context, eventType AuditEventType, data map[string]interface{}, priority AuditPriority) error {
	if !as.config.Enabled {
		return nil
	}

	// Validate required fields
	if err := as.validateEvent(data); err != nil {
		return fmt.Errorf("audit event validation failed: %w", err)
	}

	// Create audit event
	event := AuditEvent{
		ID:        uuid.New().String(),
		Type:      eventType,
		Timestamp: time.Now(),
		Data:      as.sanitizeData(data),
		Metadata:  make(map[string]interface{}),
		Priority:  priority,
		Source:    "velocimex",
		TraceID:   GetTraceID(ctx),
	}

	// Add context information
	if userID := ctx.Value("user_id"); userID != nil {
		event.UserID = fmt.Sprintf("%v", userID)
	}
	if sessionID := ctx.Value("session_id"); sessionID != nil {
		event.SessionID = fmt.Sprintf("%v", sessionID)
	}

	// Add metadata
	event.Metadata["source_ip"] = ctx.Value("source_ip")
	event.Metadata["user_agent"] = ctx.Value("user_agent")
	event.Metadata["request_id"] = ctx.Value("request_id")

	// Send to processing queue
	select {
	case as.events <- event:
		as.updateMetrics(event)
		return nil
	case <-as.ctx.Done():
		return fmt.Errorf("audit system is shutting down")
	default:
		return fmt.Errorf("audit event queue is full")
	}
}

// LogTradeEvent logs a trade-related audit event
func (as *AuditSystem) LogTradeEvent(ctx context.Context, tradeID, symbol, side, quantity, price string, metadata map[string]interface{}) error {
	data := map[string]interface{}{
		"trade_id": tradeID,
		"symbol":   symbol,
		"side":     side,
		"quantity": quantity,
		"price":    price,
		"metadata": metadata,
	}

	return as.LogEvent(ctx, TradeExecuted, data, PriorityHigh)
}

// LogOrderEvent logs an order-related audit event
func (as *AuditSystem) LogOrderEvent(ctx context.Context, orderID, symbol, side, quantity, price, orderType string, metadata map[string]interface{}) error {
	data := map[string]interface{}{
		"order_id":   orderID,
		"symbol":     symbol,
		"side":       side,
		"quantity":   quantity,
		"price":      price,
		"order_type": orderType,
		"metadata":   metadata,
	}

	return as.LogEvent(ctx, OrderPlaced, data, PriorityHigh)
}

// LogRiskEvent logs a risk management audit event
func (as *AuditSystem) LogRiskEvent(ctx context.Context, eventType, symbol string, details map[string]interface{}) error {
	data := map[string]interface{}{
		"event_type": eventType,
		"symbol":     symbol,
		"details":    details,
	}

	return as.LogEvent(ctx, RiskLimitBreached, data, PriorityCritical)
}

// LogStrategyEvent logs a strategy-related audit event
func (as *AuditSystem) LogStrategyEvent(ctx context.Context, strategy, signal string, metadata map[string]interface{}) error {
	data := map[string]interface{}{
		"strategy": strategy,
		"signal":   signal,
		"metadata": metadata,
	}

	return as.LogEvent(ctx, StrategySignal, data, PriorityNormal)
}

// LogUserAction logs a user action audit event
func (as *AuditSystem) LogUserAction(ctx context.Context, action string, details map[string]interface{}) error {
	data := map[string]interface{}{
		"action":  action,
		"details": details,
	}

	return as.LogEvent(ctx, UserAction, data, PriorityNormal)
}

// LogSystemEvent logs a system-level audit event
func (as *AuditSystem) LogSystemEvent(ctx context.Context, eventType string, details map[string]interface{}) error {
	data := map[string]interface{}{
		"event_type": eventType,
		"details":    details,
	}

	return as.LogEvent(ctx, SystemError, data, PriorityHigh)
}

// RegisterProcessor registers an audit processor
func (as *AuditSystem) RegisterProcessor(eventType AuditEventType, processor AuditProcessor) {
	as.mu.Lock()
	defer as.mu.Unlock()

	as.processors[eventType] = append(as.processors[eventType], processor)
}

// GetMetrics returns current audit metrics
func (as *AuditSystem) GetMetrics() *AuditMetrics {
	as.metrics.mu.RLock()
	defer as.metrics.mu.RUnlock()

	// Return a copy to avoid race conditions
	return &AuditMetrics{
		TotalEvents:      as.metrics.TotalEvents,
		EventsByType:     copyAuditTypeMap(as.metrics.EventsByType),
		EventsByPriority: copyAuditPriorityMap(as.metrics.EventsByPriority),
		ProcessedEvents:  as.metrics.ProcessedEvents,
		FailedEvents:     as.metrics.FailedEvents,
		ProcessingTime:   as.metrics.ProcessingTime,
		LastProcessed:    as.metrics.LastProcessed,
	}
}

// Close shuts down the audit system
func (as *AuditSystem) Close() error {
	as.cancel()
	close(as.events)
	as.wg.Wait()
	return nil
}

// worker processes audit events
func (as *AuditSystem) worker(id int) {
	defer as.wg.Done()

	for {
		select {
		case event, ok := <-as.events:
			if !ok {
				return
			}
			as.processEvent(event)
		case <-as.ctx.Done():
			return
		}
	}
}

// processEvent processes a single audit event
func (as *AuditSystem) processEvent(event AuditEvent) {
	start := time.Now()

	// Log to audit logger
	entry := AuditEntry{
		Timestamp: event.Timestamp,
		EventType: event.Type,
		UserID:    event.UserID,
		SessionID: event.SessionID,
		Metadata:  event.Data,
		IPAddress: fmt.Sprintf("%v", event.Metadata["source_ip"]),
		UserAgent: fmt.Sprintf("%v", event.Metadata["user_agent"]),
	}

	as.logger.LogEvent(entry)

	// Process with registered processors
	as.mu.RLock()
	processors := as.processors[event.Type]
	as.mu.RUnlock()

	for _, processor := range processors {
		if processor.CanProcess(event.Type) {
			if err := processor.Process(event); err != nil {
				as.updateFailedMetrics()
				continue
			}
		}
	}

	// Update metrics
	as.updateProcessedMetrics(time.Since(start))
}

// validateEvent validates an audit event
func (as *AuditSystem) validateEvent(data map[string]interface{}) error {
	// Check required fields
	for _, field := range as.config.RequiredFields {
		if _, exists := data[field]; !exists {
			return fmt.Errorf("required field '%s' is missing", field)
		}
	}

	return nil
}

// sanitizeData removes excluded fields from audit data
func (as *AuditSystem) sanitizeData(data map[string]interface{}) map[string]interface{} {
	sanitized := make(map[string]interface{})
	
	for k, v := range data {
		// Skip excluded fields
		excluded := false
		for _, field := range as.config.ExcludedFields {
			if k == field {
				excluded = true
				break
			}
		}
		
		if !excluded {
			sanitized[k] = v
		}
	}
	
	return sanitized
}

// updateMetrics updates audit metrics
func (as *AuditSystem) updateMetrics(event AuditEvent) {
	as.metrics.mu.Lock()
	defer as.metrics.mu.Unlock()

	as.metrics.TotalEvents++
	as.metrics.EventsByType[event.Type]++
	as.metrics.EventsByPriority[event.Priority]++
}

// updateProcessedMetrics updates processed event metrics
func (as *AuditSystem) updateProcessedMetrics(processingTime time.Duration) {
	as.metrics.mu.Lock()
	defer as.metrics.mu.Unlock()

	as.metrics.ProcessedEvents++
	as.metrics.ProcessingTime += processingTime
	as.metrics.LastProcessed = time.Now()
}

// updateFailedMetrics updates failed event metrics
func (as *AuditSystem) updateFailedMetrics() {
	as.metrics.mu.Lock()
	defer as.metrics.mu.Unlock()

	as.metrics.FailedEvents++
}

// collectMetrics collects and logs audit metrics
func (as *AuditSystem) collectMetrics() {
	ticker := time.NewTicker(as.config.FlushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			as.logMetrics()
		case <-as.ctx.Done():
			return
		}
	}
}

// logMetrics logs audit metrics
func (as *AuditSystem) logMetrics() {
	metrics := as.GetMetrics()
	
	// Create audit entry for metrics
	entry := AuditEntry{
		Timestamp: time.Now(),
		EventType: "audit_metrics",
		Metadata: map[string]interface{}{
			"total_events":      metrics.TotalEvents,
			"events_by_type":    metrics.EventsByType,
			"events_by_priority": metrics.EventsByPriority,
			"processed_events":  metrics.ProcessedEvents,
			"failed_events":     metrics.FailedEvents,
			"processing_time":   metrics.ProcessingTime.String(),
			"last_processed":    metrics.LastProcessed,
		},
	}

	as.logger.LogEvent(entry)
}

// Helper functions

func copyAuditTypeMap(m map[AuditEventType]int64) map[AuditEventType]int64 {
	result := make(map[AuditEventType]int64)
	for k, v := range m {
		result[k] = v
	}
	return result
}

func copyAuditPriorityMap(m map[AuditPriority]int64) map[AuditPriority]int64 {
	result := make(map[AuditPriority]int64)
	for k, v := range m {
		result[k] = v
	}
	return result
}

// GetDefaultAuditConfig returns a default audit configuration
func GetDefaultAuditConfig() *AuditConfig {
	return &AuditConfig{
		Enabled:         true,
		BufferSize:      1000,
		Workers:         4,
		FlushInterval:   1 * time.Minute,
		RetentionDays:   30,
		CompressOldLogs: true,
		EnableMetrics:   true,
		EnableAlerts:    true,
		AlertThresholds: map[string]int{
			"failed_events": 10,
			"queue_size":    800,
		},
		RequiredFields: []string{"timestamp", "event_type"},
		ExcludedFields: []string{"password", "token", "secret"},
	}
}
