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
	Enabled  bool                     `json:"enabled"`
	Channels []map[string]interface{} `json:"channels"`
	Rules    []map[string]interface{}   `json:"rules"`
	Defaults AlertDefaults            `json:"defaults"`
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
		Enabled: true,
		Channels: []map[string]interface{}{
			{
				"type":   "console",
				"name":   "console",
			},
			{
				"type":     "file",
				"name":     "file",
				"filename": "logs/alerts.jsonl",
			},
		},
		Rules: []map[string]interface{}{
			{
				"name":     "High Price Change",
				"type":     "price",
				"severity": "high",
				"conditions": []map[string]interface{}{
					{
						"field":    "change_pct",
						"operator": "gt",
						"value":    5.0,
					},
				},
				"message": "Price change {{change_pct}}% for {{symbol}}",
				"cooldown": "5m",
				"enabled":  true,
			},
			{
				"name":     "Risk Threshold",
				"type":     "risk",
				"severity": "critical",
				"conditions": []map[string]interface{}{
					{
						"field":    "risk_pct",
						"operator": "gt",
						"value":    10.0,
					},
				},
				"message": "Risk threshold exceeded: {{risk_pct}}%",
				"cooldown": "1m",
				"enabled":  true,
			},
			{
				"name":     "System Health",
				"type":     "system",
				"severity": "medium",
				"conditions": []map[string]interface{}{
					{
						"field":    "status",
						"operator": "ne",
						"value":    "healthy",
					},
				},
				"message": "System component {{component}} is {{status}}",
				"cooldown": "30s",
				"enabled":  true,
			},
			{
				"name":     "Strategy Signal",
				"type":     "strategy",
				"severity": "low",
				"conditions": []map[string]interface{}{
					{
						"field":    "signal",
						"operator": "ne",
						"value":    "",
					},
				},
				"message": "Strategy {{strategy_id}} generated signal: {{signal}}",
				"cooldown": "1m",
				"enabled":  true,
			},
		},
		Defaults: AlertDefaults{
			Cooldown:      30 * time.Second,
			MaxAlerts:     1000,
			RetentionDays: 7,
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