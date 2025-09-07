package alerts

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
	
	"github.com/google/uuid"
)

// AlertConfig represents the configuration for the alert system
type AlertConfig struct {
	// Basic configuration
	Enabled  bool                     `json:"enabled" yaml:"enabled"`
	Channels []map[string]interface{} `json:"channels" yaml:"channels"`
	Rules    []map[string]interface{} `json:"rules" yaml:"rules"`
	Defaults AlertDefaults           `json:"defaults" yaml:"defaults"`

	// Engine configuration
	MaxWorkers        int           `json:"max_workers" yaml:"max_workers"`
	QueueSize         int           `json:"queue_size" yaml:"queue_size"`
	ProcessTimeout    time.Duration `json:"process_timeout" yaml:"process_timeout"`
	RetryAttempts     int           `json:"retry_attempts" yaml:"retry_attempts"`
	RetryDelay        time.Duration `json:"retry_delay" yaml:"retry_delay"`
	CooldownPeriod    time.Duration `json:"cooldown_period" yaml:"cooldown_period"`
	EnableMetrics     bool          `json:"enable_metrics" yaml:"enable_metrics"`
	EnableTemplates   bool          `json:"enable_templates" yaml:"enable_templates"`
	EnableScheduling  bool          `json:"enable_scheduling" yaml:"enable_scheduling"`
	CleanupInterval   time.Duration `json:"cleanup_interval" yaml:"cleanup_interval"`
	MaxAlertAge       time.Duration `json:"max_alert_age" yaml:"max_alert_age"`
}

// AlertDefaults contains default settings for alerts
type AlertDefaults struct {
	Cooldown      time.Duration `json:"cooldown"`
	MaxAlerts     int           `json:"max_alerts"`
	RetentionDays int           `json:"retention_days"`
}

// DefaultAlertConfig returns the default alert configuration
func DefaultAlertConfig() *AlertConfig {
	return &AlertConfig{
		// Basic configuration
		Enabled:  true,
		Channels: []map[string]interface{}{
			{
				"name": "console",
				"type": "console",
				"enabled": true,
			},
			{
				"name": "email",
				"type": "email",
				"enabled": false,
				"config": map[string]interface{}{
					"smtp_host": "localhost",
					"smtp_port": 587,
				},
			},
		},
		Rules: []map[string]interface{}{
			{
				"id": "price_alert",
				"name": "Price Alert",
				"type": "price",
				"enabled": true,
				"severity": "medium",
				"conditions": []map[string]interface{}{
					{
						"field": "price",
						"operator": "gt",
						"value": 0,
					},
				},
			},
			{
				"id": "volume_alert",
				"name": "Volume Alert", 
				"type": "volume",
				"enabled": true,
				"severity": "low",
				"conditions": []map[string]interface{}{
					{
						"field": "volume",
						"operator": "gt",
						"value": 1000,
					},
				},
			},
			{
				"id": "risk_alert",
				"name": "Risk Alert",
				"type": "risk",
				"enabled": true,
				"severity": "high",
				"conditions": []map[string]interface{}{
					{
						"field": "risk_level",
						"operator": "gt",
						"value": 0.8,
					},
				},
			},
			{
				"id": "system_alert",
				"name": "System Alert",
				"type": "system",
				"enabled": true,
				"severity": "critical",
				"conditions": []map[string]interface{}{
					{
						"field": "status",
						"operator": "eq",
						"value": "error",
					},
				},
			},
		},

		// Engine configuration
		MaxWorkers:        4,
		QueueSize:         1000,
		ProcessTimeout:    30 * time.Second,
		RetryAttempts:     3,
		RetryDelay:        5 * time.Second,
		CooldownPeriod:    1 * time.Minute,
		EnableMetrics:     true,
		EnableTemplates:   true,
		EnableScheduling:  true,
		CleanupInterval:   1 * time.Hour,
		MaxAlertAge:       24 * time.Hour,

		// Default values
		Defaults: AlertDefaults{
			Cooldown:      5 * time.Minute,
			MaxAlerts:     1000,
			RetentionDays: 30,
		},
	}
}

// LoadAlertConfig loads alert configuration from file
func LoadAlertConfig(filename string) (*AlertConfig, error) {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		// Create default config file
		config := DefaultAlertConfig()
		if err := SaveAlertConfig(filename, config); err != nil {
			return nil, fmt.Errorf("failed to create default config: %w", err)
		}
		return config, nil
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config AlertConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &config, nil
}

// SaveAlertConfig saves alert configuration to file
func SaveAlertConfig(filename string, config *AlertConfig) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// SetupAlertManager creates and configures an alert manager from config
func SetupAlertManager(config *AlertConfig, logger interface{}) (*VelocimexAlertManager, error) {
	if !config.Enabled {
		return nil, fmt.Errorf("alert system is disabled")
	}

	am := NewAlertManager(nil)
	
	// Register channels
	factory := NewChannelFactory()
	for _, channelConfig := range config.Channels {
		channel, err := factory.CreateChannel(channelConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create channel: %w", err)
		}
		
		if err := am.RegisterChannel(channel); err != nil {
			return nil, fmt.Errorf("failed to register channel: %w", err)
		}
	}
	
	// Add rules
	for _, ruleConfig := range config.Rules {
		rule, err := createRuleFromConfig(ruleConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create rule: %w", err)
		}
		
		if err := am.AddRule(rule); err != nil {
			return nil, fmt.Errorf("failed to add rule: %w", err)
		}
	}
	
	return am, nil
}

// createRuleFromConfig creates an AlertRule from configuration
func createRuleFromConfig(config map[string]interface{}) (*AlertRule, error) {
	// Parse basic fields
	name, _ := config["name"].(string)
	typeStr, _ := config["type"].(string)
	severityStr, _ := config["severity"].(string)
	message, _ := config["message"].(string)
	enabled, _ := config["enabled"].(bool)
	
	// Parse conditions
	conditions := make([]AlertCondition, 0)
	if conds, ok := config["conditions"].([]interface{}); ok {
		for _, cond := range conds {
			if condMap, ok := cond.(map[string]interface{}); ok {
				condition := AlertCondition{
					Field:    getString(condMap, "field"),
					Operator: getString(condMap, "operator"),
					Value:    condMap["value"],
				}
				conditions = append(conditions, condition)
			}
		}
	}
	
	// Parse cooldown
	cooldown := 30 * time.Second
	if cooldownStr, ok := config["cooldown"].(string); ok {
		if d, err := time.ParseDuration(cooldownStr); err == nil {
			cooldown = d
		}
	}
	
	// Parse channels
	channels := make([]string, 0)
	if chans, ok := config["channels"].([]interface{}); ok {
		for _, ch := range chans {
			if str, ok := ch.(string); ok {
				channels = append(channels, str)
			}
		}
	}
	
	return &AlertRule{
		ID:         uuid.NewString(),
		Name:       name,
		Type:       AlertType(typeStr),
		Severity:   AlertSeverity(severityStr),
		Conditions: conditions,
		Message:    message,
		Enabled:    enabled,
		Cooldown:   cooldown,
		Channels:   channels,
	}, nil
}

// Helper function to safely get string from map
func getString(m map[string]interface{}, key string) string {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

// Helper function to safely get float64 from map
func getFloat64(m map[string]interface{}, key string) float64 {
	if val, ok := m[key]; ok {
		switch v := val.(type) {
		case float64:
			return v
		case int:
			return float64(v)
		case int64:
			return float64(v)
		}
	}
	return 0
}

// Helper function to safely get bool from map
func getBool(m map[string]interface{}, key string) bool {
	if val, ok := m[key]; ok {
		if b, ok := val.(bool); ok {
			return b
		}
	}
	return false
}

// CreateDefaultRules creates commonly used alert rules
func CreateDefaultRules() []*AlertRule {
	return []*AlertRule{
		{
			ID:       "high_price_change",
			Name:     "High Price Change",
			Type:     AlertTypePrice,
			Severity: SeverityHigh,
			Conditions: []AlertCondition{
				{Field: "change_pct", Operator: "gt", Value: 5.0},
			},
			Message:  "Price change {{change_pct}}% for {{symbol}}",
			Enabled:  true,
			Cooldown: 5 * time.Minute,
		},
		{
			ID:       "risk_threshold",
			Name:     "Risk Threshold",
			Type:     AlertTypeRisk,
			Severity: SeverityCritical,
			Conditions: []AlertCondition{
				{Field: "risk_pct", Operator: "gt", Value: 10.0},
			},
			Message:  "Risk threshold exceeded: {{risk_pct}}%",
			Enabled:  true,
			Cooldown: 1 * time.Minute,
		},
		{
			ID:       "volume_spike",
			Name:     "Volume Spike",
			Type:     AlertTypeVolume,
			Severity: SeverityMedium,
			Conditions: []AlertCondition{
				{Field: "volume_ratio", Operator: "gt", Value: 3.0},
			},
			Message:  "Volume spike detected: {{volume_ratio}}x normal",
			Enabled:  true,
			Cooldown: 2 * time.Minute,
		},
		{
			ID:       "connectivity_issue",
			Name:     "Connectivity Issue",
			Type:     AlertTypeConnectivity,
			Severity: SeverityHigh,
			Conditions: []AlertCondition{
				{Field: "status", Operator: "ne", Value: "connected"},
			},
			Message:  "Connectivity issue with {{component}}: {{status}}",
			Enabled:  true,
			Cooldown: 30 * time.Second,
		},
		{
			ID:       "strategy_signal",
			Name:     "Strategy Signal",
			Type:     AlertTypeStrategy,
			Severity: SeverityLow,
			Conditions: []AlertCondition{
				{Field: "signal", Operator: "ne", Value: ""},
			},
			Message:  "Strategy {{strategy_id}} generated signal: {{signal}}",
			Enabled:  true,
			Cooldown: 1 * time.Minute,
		},
	}
}

// GetAlertConfigPath returns the default path for alert configuration
func GetAlertConfigPath() string {
	return "config/alerts.json"
}