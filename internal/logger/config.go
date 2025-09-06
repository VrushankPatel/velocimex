package logger

import (
	"fmt"
	"time"
)

// LoggingConfig holds the complete logging configuration
type LoggingConfig struct {
	Global    *Config        `yaml:"global"`
	Audit     *AuditConfig   `yaml:"audit"`
	Rotation  *RotationConfig `yaml:"rotation"`
	Search    *SearchConfig  `yaml:"search"`
	Formatters map[string]FormatterConfig `yaml:"formatters"`
	Components map[string]ComponentConfig `yaml:"components"`
}

// SearchConfig holds configuration for log search functionality
type SearchConfig struct {
	Enabled     bool          `yaml:"enabled"`
	IndexDir    string        `yaml:"index_dir"`
	IndexInterval time.Duration `yaml:"index_interval"`
	MaxIndexSize int64        `yaml:"max_index_size"`
	EnableSearch bool         `yaml:"enable_search"`
	SearchPort  int           `yaml:"search_port"`
}

// ComponentConfig holds configuration for specific components
type ComponentConfig struct {
	Level      LogLevel `yaml:"level"`
	Format     string   `yaml:"format"`
	Output     string   `yaml:"output"`
	EnableAudit bool    `yaml:"enable_audit"`
	MaxSize    int64    `yaml:"max_size"`
	MaxAge     string   `yaml:"max_age"`
	MaxBackups int      `yaml:"max_backups"`
	Compress   bool     `yaml:"compress"`
}

// FormatterConfig holds configuration for log formatters
type FormatterConfig struct {
	Type         string                 `yaml:"type"`
	PrettyPrint  bool                   `yaml:"pretty_print"`
	IncludeTime  bool                   `yaml:"include_time"`
	TimeFormat   string                 `yaml:"time_format"`
	ServiceName  string                 `yaml:"service_name"`
	Environment  string                 `yaml:"environment"`
	Tag          string                 `yaml:"tag"`
	IncludeHeaders bool                 `yaml:"include_headers"`
	Options     map[string]interface{} `yaml:"options"`
}

// GetDefaultLoggingConfig returns a default logging configuration
func GetDefaultLoggingConfig() *LoggingConfig {
	return &LoggingConfig{
		Global: &Config{
			Level:            INFO,
			Format:           "json",
			Output:           "logs/velocimex.log",
			EnableAudit:      true,
			AuditFile:        "logs/audit.log",
			MaxFileSizeMB:    100,
			MaxBackupFiles:   10,
			CompressBackups:  true,
			EnableTrace:      true,
			TraceHeaderName:  "X-Trace-ID",
		},
		Audit: GetDefaultAuditConfig(),
		Rotation: &RotationConfig{
			Strategy:        RotationBySizeAndTime,
			MaxSizeBytes:    100 * 1024 * 1024, // 100MB
			MaxAge:          7 * 24 * time.Hour, // 7 days
			MaxBackups:      10,
			Compress:        true,
			LocalTime:       true,
			RotateOnStartup: false,
		},
		Search: &SearchConfig{
			Enabled:       true,
			IndexDir:      "logs/index",
			IndexInterval: 1 * time.Minute,
			MaxIndexSize:  1024 * 1024 * 1024, // 1GB
			EnableSearch:  true,
			SearchPort:    8080,
		},
		Formatters: map[string]FormatterConfig{
			"json": {
				Type:        "json",
				PrettyPrint: false,
				IncludeTime: true,
			},
			"text": {
				Type:        "text",
				IncludeTime: true,
				TimeFormat:  "2006-01-02 15:04:05.000",
			},
			"logstash": {
				Type:        "logstash",
				ServiceName: "velocimex",
				Environment: "production",
			},
		},
		Components: map[string]ComponentConfig{
			"main": {
				Level:       INFO,
				Format:      "json",
				Output:      "logs/main.log",
				EnableAudit: true,
				MaxSize:     100 * 1024 * 1024,
				MaxAge:      "7d",
				MaxBackups:  10,
				Compress:    true,
			},
			"trading": {
				Level:       INFO,
				Format:      "json",
				Output:      "logs/trading.log",
				EnableAudit: true,
				MaxSize:     200 * 1024 * 1024,
				MaxAge:      "30d",
				MaxBackups:  20,
				Compress:    true,
			},
			"risk": {
				Level:       WARN,
				Format:      "json",
				Output:      "logs/risk.log",
				EnableAudit: true,
				MaxSize:     50 * 1024 * 1024,
				MaxAge:      "90d",
				MaxBackups:  30,
				Compress:    true,
			},
			"audit": {
				Level:       INFO,
				Format:      "json",
				Output:      "logs/audit.log",
				EnableAudit: false,
				MaxSize:     500 * 1024 * 1024,
				MaxAge:      "365d",
				MaxBackups:  50,
				Compress:    true,
			},
			"performance": {
				Level:       INFO,
				Format:      "json",
				Output:      "logs/performance.log",
				EnableAudit: false,
				MaxSize:     100 * 1024 * 1024,
				MaxAge:      "30d",
				MaxBackups:  10,
				Compress:    true,
			},
			"http": {
				Level:       INFO,
				Format:      "text",
				Output:      "logs/access.log",
				EnableAudit: false,
				MaxSize:     200 * 1024 * 1024,
				MaxAge:      "7d",
				MaxBackups:  5,
				Compress:    true,
			},
		},
	}
}

// ValidateConfig validates the logging configuration
func (lc *LoggingConfig) ValidateConfig() error {
	// Validate global config
	if lc.Global == nil {
		return fmt.Errorf("global config is required")
	}

	if lc.Global.Level < DEBUG || lc.Global.Level > FATAL {
		return fmt.Errorf("invalid global log level: %d", lc.Global.Level)
	}

	if lc.Global.Format != "json" && lc.Global.Format != "text" {
		return fmt.Errorf("invalid global format: %s", lc.Global.Format)
	}

	// Validate audit config
	if lc.Audit != nil && lc.Audit.Enabled {
		if lc.Audit.BufferSize <= 0 {
			return fmt.Errorf("audit buffer size must be positive")
		}
		if lc.Audit.Workers <= 0 {
			return fmt.Errorf("audit workers must be positive")
		}
	}

	// Validate rotation config
	if lc.Rotation != nil {
		if lc.Rotation.MaxSizeBytes <= 0 {
			return fmt.Errorf("max size bytes must be positive")
		}
		if lc.Rotation.MaxBackups < 0 {
			return fmt.Errorf("max backups cannot be negative")
		}
	}

	// Validate search config
	if lc.Search != nil && lc.Search.Enabled {
		if lc.Search.IndexInterval <= 0 {
			return fmt.Errorf("index interval must be positive")
		}
		if lc.Search.MaxIndexSize <= 0 {
			return fmt.Errorf("max index size must be positive")
		}
	}

	// Validate component configs
	for name, config := range lc.Components {
		if config.Level < DEBUG || config.Level > FATAL {
			return fmt.Errorf("invalid log level for component %s: %d", name, config.Level)
		}
		if config.MaxSize <= 0 {
			return fmt.Errorf("max size must be positive for component %s", name)
		}
		if config.MaxBackups < 0 {
			return fmt.Errorf("max backups cannot be negative for component %s", name)
		}
	}

	return nil
}

// GetComponentConfig returns the configuration for a specific component
func (lc *LoggingConfig) GetComponentConfig(component string) ComponentConfig {
	if config, exists := lc.Components[component]; exists {
		return config
	}
	
	// Return default config for unknown components
	return ComponentConfig{
		Level:      lc.Global.Level,
		Format:     lc.Global.Format,
		Output:     fmt.Sprintf("logs/%s.log", component),
		EnableAudit: lc.Global.EnableAudit,
		MaxSize:    100 * 1024 * 1024,
		MaxAge:     "7d",
		MaxBackups: 10,
		Compress:   true,
	}
}

// GetFormatterConfig returns the configuration for a specific formatter
func (lc *LoggingConfig) GetFormatterConfig(formatter string) FormatterConfig {
	if config, exists := lc.Formatters[formatter]; exists {
		return config
	}
	
	// Return default config for unknown formatters
	return FormatterConfig{
		Type:        "json",
		PrettyPrint: false,
		IncludeTime: true,
	}
}

// MergeConfigs merges multiple logging configurations
func MergeConfigs(configs ...*LoggingConfig) *LoggingConfig {
	if len(configs) == 0 {
		return GetDefaultLoggingConfig()
	}

	result := configs[0]
	
	for i := 1; i < len(configs); i++ {
		config := configs[i]
		
		// Merge global config
		if config.Global != nil {
			result.Global = config.Global
		}
		
		// Merge audit config
		if config.Audit != nil {
			result.Audit = config.Audit
		}
		
		// Merge rotation config
		if config.Rotation != nil {
			result.Rotation = config.Rotation
		}
		
		// Merge search config
		if config.Search != nil {
			result.Search = config.Search
		}
		
		// Merge formatters
		if config.Formatters != nil {
			if result.Formatters == nil {
				result.Formatters = make(map[string]FormatterConfig)
			}
			for k, v := range config.Formatters {
				result.Formatters[k] = v
			}
		}
		
		// Merge components
		if config.Components != nil {
			if result.Components == nil {
				result.Components = make(map[string]ComponentConfig)
			}
			for k, v := range config.Components {
				result.Components[k] = v
			}
		}
	}
	
	return result
}

// ParseLogLevel parses a log level from string
func ParseLogLevel(level string) (LogLevel, error) {
	switch level {
	case "debug", "DEBUG":
		return DEBUG, nil
	case "info", "INFO":
		return INFO, nil
	case "warn", "WARN", "warning", "WARNING":
		return WARN, nil
	case "error", "ERROR":
		return ERROR, nil
	case "fatal", "FATAL":
		return FATAL, nil
	default:
		return INFO, fmt.Errorf("invalid log level: %s", level)
	}
}

// ParseRotationStrategy parses a rotation strategy from string
func ParseRotationStrategy(strategy string) (RotationStrategy, error) {
	switch strategy {
	case "size":
		return RotationBySize, nil
	case "time":
		return RotationByTime, nil
	case "size_and_time", "sizeandtime":
		return RotationBySizeAndTime, nil
	default:
		return RotationBySizeAndTime, fmt.Errorf("invalid rotation strategy: %s", strategy)
	}
}

// ParseDuration parses a duration string with common suffixes
func ParseDuration(duration string) (time.Duration, error) {
	// Handle common suffixes
	switch {
	case duration == "0":
		return 0, nil
	case duration == "":
		return 0, nil
	}
	
	// Try parsing as-is first
	if d, err := time.ParseDuration(duration); err == nil {
		return d, nil
	}
	
	// Try with common suffixes
	suffixes := map[string]time.Duration{
		"s": time.Second,
		"m": time.Minute,
		"h": time.Hour,
		"d": 24 * time.Hour,
		"w": 7 * 24 * time.Hour,
		"M": 30 * 24 * time.Hour,
		"y": 365 * 24 * time.Hour,
	}
	
	for suffix, multiplier := range suffixes {
		if len(duration) > len(suffix) && duration[len(duration)-len(suffix):] == suffix {
			value := duration[:len(duration)-len(suffix)]
			if d, err := time.ParseDuration(value + "s"); err == nil {
				return d * multiplier, nil
			}
		}
	}
	
	return 0, fmt.Errorf("invalid duration: %s", duration)
}