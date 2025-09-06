package logger

import (
	"os"
	"path/filepath"
)

// DefaultConfig returns the default logger configuration
func DefaultConfig() *Config {
	return &Config{
		Level:            INFO,
		Format:           "json",
		Output:           "stdout",
		EnableAudit:      true,
		AuditFile:        "logs/audit.log",
		MaxFileSizeMB:    100,
		MaxBackupFiles:   5,
		CompressBackups:  true,
		EnableTrace:      true,
		TraceHeaderName:  "X-Trace-ID",
	}
}

// LoadConfig loads logger configuration from environment variables
func LoadConfig() *Config {
	config := DefaultConfig()

	// Override from environment variables if set
	if level := os.Getenv("LOG_LEVEL"); level != "" {
		switch level {
		case "DEBUG":
			config.Level = DEBUG
		case "INFO":
			config.Level = INFO
		case "WARN":
			config.Level = WARN
		case "ERROR":
			config.Level = ERROR
		case "FATAL":
			config.Level = FATAL
		}
	}

	if format := os.Getenv("LOG_FORMAT"); format != "" {
		config.Format = format
	}

	if output := os.Getenv("LOG_OUTPUT"); output != "" {
		config.Output = output
	}

	if auditFile := os.Getenv("AUDIT_FILE"); auditFile != "" {
		config.AuditFile = auditFile
	}

	if enableAudit := os.Getenv("ENABLE_AUDIT"); enableAudit == "false" {
		config.EnableAudit = false
	}

	return config
}

// SetupLoggingDirectory ensures the logging directory exists
func SetupLoggingDirectory(logDir string) error {
	if logDir == "" {
		logDir = "logs"
	}

	// Create logs directory if it doesn't exist
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return err
	}

	return nil
}

// GetLogFilePath returns the full path for a log file
func GetLogFilePath(filename string) string {
	if filepath.IsAbs(filename) {
		return filename
	}
	return filepath.Join("logs", filename)
}

// Global logger instance
var globalLogger *VelocimexLogger

// Init initializes the global logger
func Init(config *Config) error {
	var err error
	globalLogger, err = New(config)
	return err
}

// GetLogger returns the global logger instance
func GetLogger() *VelocimexLogger {
	if globalLogger == nil {
		// Initialize with default config if not initialized
		_ = Init(DefaultConfig())
	}
	return globalLogger
}

// Shutdown gracefully shuts down the global logger
func Shutdown() error {
	if globalLogger != nil {
		return globalLogger.Close()
	}
	return nil
}