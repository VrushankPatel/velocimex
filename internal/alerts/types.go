package alerts

import (
	"time"
)

// AlertSeverity represents the severity level of an alert
type AlertSeverity string

const (
	SeverityLow    AlertSeverity = "low"
	SeverityMedium AlertSeverity = "medium"
	SeverityHigh   AlertSeverity = "high"
	SeverityCritical AlertSeverity = "critical"
)

// AlertType represents the type of alert
type AlertType string

const (
	AlertTypePrice         AlertType = "price"
	AlertTypeVolume        AlertType = "volume"
	AlertTypeRisk          AlertType = "risk"
	AlertTypeStrategy      AlertType = "strategy"
	AlertTypeSystem        AlertType = "system"
	AlertTypeConnectivity  AlertType = "connectivity"
	AlertTypePerformance   AlertType = "performance"
)

// AlertCondition defines a condition that triggers an alert
type AlertCondition struct {
	Field     string      `json:"field"`
	Operator  string      `json:"operator"` // gt, lt, eq, ne, contains
	Value     interface{} `json:"value"`
	Threshold float64     `json:"threshold,omitempty"`
}

// AlertRule defines a rule for generating alerts
type AlertRule struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	Type        AlertType        `json:"type"`
	Severity    AlertSeverity    `json:"severity"`
	Conditions  []AlertCondition `json:"conditions"`
	Message     string           `json:"message"`
	Enabled     bool             `json:"enabled"`
	Cooldown    time.Duration    `json:"cooldown"`
	Channels    []string         `json:"channels"`
	LastTriggered time.Time      `json:"last_triggered,omitempty"`
}

// Alert represents an actual alert that has been triggered
type Alert struct {
	ID          string        `json:"id"`
	RuleID      string        `json:"rule_id"`
	Type        AlertType     `json:"type"`
	Severity    AlertSeverity `json:"severity"`
	Title       string        `json:"title"`
	Message     string        `json:"message"`
	Data        interface{}   `json:"data,omitempty"`
	Timestamp   time.Time     `json:"timestamp"`
	Acknowledged bool         `json:"acknowledged"`
	Resolved    bool          `json:"resolved"`
	ResolvedAt  *time.Time    `json:"resolved_at,omitempty"`
}

// AlertChannel represents a delivery channel for alerts
type AlertChannel interface {
	Send(alert *Alert) error
	Name() string
	Type() string
}

// AlertManager manages alert rules and notifications
type AlertManager interface {
	AddRule(rule *AlertRule) error
	RemoveRule(ruleID string) error
	UpdateRule(rule *AlertRule) error
	GetRule(ruleID string) (*AlertRule, error)
	GetRules() []*AlertRule
	
	TriggerAlert(rule *AlertRule, data interface{}) error
	AcknowledgeAlert(alertID string) error
	ResolveAlert(alertID string) error
	
	GetAlerts(filters map[string]interface{}) ([]*Alert, error)
	GetActiveAlerts() ([]*Alert, error)
	
	RegisterChannel(channel AlertChannel) error
	RemoveChannel(channelName string) error
	
	Start() error
	Stop() error
}

// AlertEvent represents an alert-related event
type AlertEvent struct {
	Type      string      `json:"type"`
	AlertID   string      `json:"alert_id,omitempty"`
	RuleID    string      `json:"rule_id,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data,omitempty"`
}

// PriceAlertData contains data for price-based alerts
type PriceAlertData struct {
	Symbol    string  `json:"symbol"`
	Price     float64 `json:"price"`
	Previous  float64 `json:"previous,omitempty"`
	Change    float64 `json:"change,omitempty"`
	ChangePct float64 `json:"change_pct,omitempty"`
}

// RiskAlertData contains data for risk-based alerts
type RiskAlertData struct {
	PortfolioValue float64 `json:"portfolio_value"`
	RiskAmount     float64 `json:"risk_amount"`
	RiskPct        float64 `json:"risk_pct"`
	PositionCount  int     `json:"position_count"`
	MaxDrawdown    float64 `json:"max_drawdown,omitempty"`
}

// StrategyAlertData contains data for strategy-based alerts
type StrategyAlertData struct {
	StrategyID string      `json:"strategy_id"`
	Signal     string      `json:"signal"`
	Confidence float64     `json:"confidence,omitempty"`
	Metadata   interface{} `json:"metadata,omitempty"`
}

// SystemAlertData contains data for system-based alerts
type SystemAlertData struct {
	Component   string `json:"component"`
	Status      string `json:"status"`
	Error       string `json:"error,omitempty"`
	Uptime      string `json:"uptime,omitempty"`
	MemoryUsage string `json:"memory_usage,omitempty"`
}