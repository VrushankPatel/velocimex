package fix

import (
	"time"
)

// Config holds FIX protocol configuration
type Config struct {
	// Connection settings
	Host     string        `yaml:"host"`
	Port     int           `yaml:"port"`
	Username string        `yaml:"username"`
	Password string        `yaml:"password"`
	Timeout  time.Duration `yaml:"timeout"`
	
	// FIX protocol settings
	SenderCompID   string `yaml:"sender_comp_id"`
	TargetCompID   string `yaml:"target_comp_id"`
	BeginString    string `yaml:"begin_string"`    // FIX.4.4, FIX.4.2, etc.
	HeartBtInt     int    `yaml:"heart_bt_int"`    // Heartbeat interval in seconds
	ResetSeqNum    bool   `yaml:"reset_seq_num"`   // Reset sequence numbers on logon
	
	// Trading settings
	DefaultOrderType string `yaml:"default_order_type"` // Market, Limit, Stop, etc.
	DefaultTimeInForce string `yaml:"default_time_in_force"` // Day, IOC, GTC, etc.
	
	// Security settings
	UseSSL         bool   `yaml:"use_ssl"`
	CertFile       string `yaml:"cert_file"`
	KeyFile        string `yaml:"key_file"`
	CAFile         string `yaml:"ca_file"`
	
	// Logging
	LogHeartbeats  bool   `yaml:"log_heartbeats"`
	LogMessages    bool   `yaml:"log_messages"`
	LogFile        string `yaml:"log_file"`
}

// DefaultConfig returns default FIX configuration
func DefaultConfig() Config {
	return Config{
		Host:              "localhost",
		Port:              9876,
		Username:          "velocimex",
		Password:          "password",
		Timeout:           30 * time.Second,
		SenderCompID:      "VELOCIMEX",
		TargetCompID:      "EXCHANGE",
		BeginString:       "FIX.4.4",
		HeartBtInt:        30,
		ResetSeqNum:       false,
		DefaultOrderType:  "1", // Market order
		DefaultTimeInForce: "1", // Day
		UseSSL:            false,
		LogHeartbeats:     false,
		LogMessages:       true,
		LogFile:           "fix.log",
	}
}
