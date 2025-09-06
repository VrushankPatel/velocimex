package risk

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/shopspring/decimal"
)

// AlertManager manages risk alerts and notifications
type AlertManager struct {
	alerts      map[string]*Alert
	subscribers []AlertSubscriber
	mu          sync.RWMutex
}

// Alert represents a risk alert
type Alert struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Severity    RiskLevel              `json:"severity"`
	Message     string                 `json:"message"`
	Symbol      string                 `json:"symbol,omitempty"`
	Exchange    string                 `json:"exchange,omitempty"`
	Value       decimal.Decimal        `json:"value,omitempty"`
	Threshold   decimal.Decimal        `json:"threshold,omitempty"`
	Status      AlertStatus            `json:"status"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// AlertStatus represents the status of an alert
type AlertStatus string

const (
	AlertStatusActive   AlertStatus = "ACTIVE"
	AlertStatusResolved AlertStatus = "RESOLVED"
	AlertStatusSuppressed AlertStatus = "SUPPRESSED"
)

// AlertSubscriber defines the interface for alert subscribers
type AlertSubscriber interface {
	OnAlert(alert *Alert) error
}

// AlertConfig represents alert configuration
type AlertConfig struct {
	Enabled           bool          `json:"enabled"`
	CheckInterval     time.Duration `json:"check_interval"`
	MaxAlerts         int           `json:"max_alerts"`
	SuppressionPeriod time.Duration `json:"suppression_period"`
	Channels          []string      `json:"channels"` // "email", "sms", "webhook", "log"
}

// DefaultAlertConfig returns default alert configuration
func DefaultAlertConfig() AlertConfig {
	return AlertConfig{
		Enabled:           true,
		CheckInterval:     1 * time.Second,
		MaxAlerts:         1000,
		SuppressionPeriod: 5 * time.Minute,
		Channels:          []string{"log"},
	}
}

// NewAlertManager creates a new alert manager
func NewAlertManager() *AlertManager {
	return &AlertManager{
		alerts:      make(map[string]*Alert),
		subscribers: make([]AlertSubscriber, 0),
	}
}

// CreateAlert creates a new risk alert
func (am *AlertManager) CreateAlert(alert *Alert) error {
	am.mu.Lock()
	defer am.mu.Unlock()
	
	alert.CreatedAt = time.Now()
	alert.UpdatedAt = time.Now()
	alert.Status = AlertStatusActive
	
	am.alerts[alert.ID] = alert
	
	// Notify subscribers
	for _, subscriber := range am.subscribers {
		go func(sub AlertSubscriber) {
			if err := sub.OnAlert(alert); err != nil {
				log.Printf("Error notifying alert subscriber: %v", err)
			}
		}(subscriber)
	}
	
	// Log alert
	log.Printf("Risk Alert [%s] %s: %s", alert.Severity, alert.Type, alert.Message)
	
	return nil
}

// GetAlert returns an alert by ID
func (am *AlertManager) GetAlert(id string) (*Alert, error) {
	am.mu.RLock()
	defer am.mu.RUnlock()
	
	alert, exists := am.alerts[id]
	if !exists {
		return nil, fmt.Errorf("alert not found: %s", id)
	}
	
	return alert, nil
}

// GetAlerts returns alerts with optional filtering
func (am *AlertManager) GetAlerts(filters map[string]interface{}) ([]*Alert, error) {
	am.mu.RLock()
	defer am.mu.RUnlock()
	
	var alerts []*Alert
	for _, alert := range am.alerts {
		if am.matchesAlertFilters(alert, filters) {
			alerts = append(alerts, alert)
		}
	}
	
	return alerts, nil
}

// ResolveAlert resolves an alert
func (am *AlertManager) ResolveAlert(id string) error {
	am.mu.Lock()
	defer am.mu.Unlock()
	
	alert, exists := am.alerts[id]
	if !exists {
		return fmt.Errorf("alert not found: %s", id)
	}
	
	alert.Status = AlertStatusResolved
	alert.UpdatedAt = time.Now()
	
	log.Printf("Alert resolved: %s", id)
	return nil
}

// SuppressAlert suppresses an alert
func (am *AlertManager) SuppressAlert(id string, duration time.Duration) error {
	am.mu.Lock()
	defer am.mu.Unlock()
	
	alert, exists := am.alerts[id]
	if !exists {
		return fmt.Errorf("alert not found: %s", id)
	}
	
	alert.Status = AlertStatusSuppressed
	alert.UpdatedAt = time.Now()
	
	// Auto-resolve after suppression period
	go func() {
		time.Sleep(duration)
		am.ResolveAlert(id)
	}()
	
	log.Printf("Alert suppressed for %v: %s", duration, id)
	return nil
}

// Subscribe subscribes to alert notifications
func (am *AlertManager) Subscribe(subscriber AlertSubscriber) {
	am.mu.Lock()
	defer am.mu.Unlock()
	
	am.subscribers = append(am.subscribers, subscriber)
}

// CleanupOldAlerts removes old resolved alerts
func (am *AlertManager) CleanupOldAlerts(maxAge time.Duration) {
	am.mu.Lock()
	defer am.mu.Unlock()
	
	cutoff := time.Now().Add(-maxAge)
	for id, alert := range am.alerts {
		if alert.Status == AlertStatusResolved && alert.UpdatedAt.Before(cutoff) {
			delete(am.alerts, id)
		}
	}
}

// Private methods

func (am *AlertManager) matchesAlertFilters(alert *Alert, filters map[string]interface{}) bool {
	if status, ok := filters["status"]; ok {
		if alert.Status != AlertStatus(status.(string)) {
			return false
		}
	}
	
	if severity, ok := filters["severity"]; ok {
		if alert.Severity != RiskLevel(severity.(string)) {
			return false
		}
	}
	
	if alertType, ok := filters["type"]; ok {
		if alert.Type != alertType.(string) {
			return false
		}
	}
	
	if symbol, ok := filters["symbol"]; ok {
		if alert.Symbol != symbol.(string) {
			return false
		}
	}
	
	return true
}

// LogAlertSubscriber implements AlertSubscriber for logging
type LogAlertSubscriber struct{}

// OnAlert handles alert notifications by logging them
func (las *LogAlertSubscriber) OnAlert(alert *Alert) error {
	log.Printf("RISK ALERT [%s] %s: %s", alert.Severity, alert.Type, alert.Message)
	if alert.Symbol != "" {
		log.Printf("  Symbol: %s", alert.Symbol)
	}
	if alert.Exchange != "" {
		log.Printf("  Exchange: %s", alert.Exchange)
	}
	if !alert.Value.IsZero() {
		log.Printf("  Value: %s", alert.Value.String())
	}
	if !alert.Threshold.IsZero() {
		log.Printf("  Threshold: %s", alert.Threshold.String())
	}
	return nil
}

// EmailAlertSubscriber implements AlertSubscriber for email notifications
type EmailAlertSubscriber struct {
	SMTPConfig SMTPConfig
}

// SMTPConfig represents SMTP configuration for email alerts
type SMTPConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	From     string `json:"from"`
	To       []string `json:"to"`
}

// OnAlert handles alert notifications by sending emails
func (eas *EmailAlertSubscriber) OnAlert(alert *Alert) error {
	// In a real implementation, this would send an email
	// For now, just log the email that would be sent
	log.Printf("EMAIL ALERT [%s] %s: %s", alert.Severity, alert.Type, alert.Message)
	log.Printf("  Would send email to: %v", eas.SMTPConfig.To)
	return nil
}

// WebhookAlertSubscriber implements AlertSubscriber for webhook notifications
type WebhookAlertSubscriber struct {
	URL string `json:"url"`
}

// OnAlert handles alert notifications by sending webhooks
func (was *WebhookAlertSubscriber) OnAlert(alert *Alert) error {
	// In a real implementation, this would send a webhook
	// For now, just log the webhook that would be sent
	log.Printf("WEBHOOK ALERT [%s] %s: %s", alert.Severity, alert.Type, alert.Message)
	log.Printf("  Would send webhook to: %s", was.URL)
	return nil
}

// RiskAlertBuilder helps build risk alerts
type RiskAlertBuilder struct {
	alert *Alert
}

// NewRiskAlertBuilder creates a new risk alert builder
func NewRiskAlertBuilder() *RiskAlertBuilder {
	return &RiskAlertBuilder{
		alert: &Alert{
			ID:       generateAlertID(),
			Metadata: make(map[string]interface{}),
		},
	}
}

// SetType sets the alert type
func (rab *RiskAlertBuilder) SetType(alertType string) *RiskAlertBuilder {
	rab.alert.Type = alertType
	return rab
}

// SetSeverity sets the alert severity
func (rab *RiskAlertBuilder) SetSeverity(severity RiskLevel) *RiskAlertBuilder {
	rab.alert.Severity = severity
	return rab
}

// SetMessage sets the alert message
func (rab *RiskAlertBuilder) SetMessage(message string) *RiskAlertBuilder {
	rab.alert.Message = message
	return rab
}

// SetSymbol sets the alert symbol
func (rab *RiskAlertBuilder) SetSymbol(symbol string) *RiskAlertBuilder {
	rab.alert.Symbol = symbol
	return rab
}

// SetExchange sets the alert exchange
func (rab *RiskAlertBuilder) SetExchange(exchange string) *RiskAlertBuilder {
	rab.alert.Exchange = exchange
	return rab
}

// SetValue sets the alert value
func (rab *RiskAlertBuilder) SetValue(value decimal.Decimal) *RiskAlertBuilder {
	rab.alert.Value = value
	return rab
}

// SetThreshold sets the alert threshold
func (rab *RiskAlertBuilder) SetThreshold(threshold decimal.Decimal) *RiskAlertBuilder {
	rab.alert.Threshold = threshold
	return rab
}

// SetMetadata sets alert metadata
func (rab *RiskAlertBuilder) SetMetadata(key string, value interface{}) *RiskAlertBuilder {
	rab.alert.Metadata[key] = value
	return rab
}

// Build builds the alert
func (rab *RiskAlertBuilder) Build() *Alert {
	return rab.alert
}

// generateAlertID generates a unique alert ID
func generateAlertID() string {
	return fmt.Sprintf("alert_%d", time.Now().UnixNano())
}
