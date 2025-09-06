package config

import (
	"io/ioutil"
	"time"

	"gopkg.in/yaml.v2"
	
	"velocimex/internal/backtesting"
	"velocimex/internal/fix"
	"velocimex/internal/plugins"
	"velocimex/internal/risk"
	"velocimex/internal/strategy"
)

// Config contains all application configuration
type Config struct {
	Server      ServerConfig           `yaml:"server"`
	Feeds       []FeedConfig           `yaml:"feeds"`
	FIX         fix.Config             `yaml:"fix"`
	Risk        risk.RiskConfig        `yaml:"risk"`
	Backtesting backtesting.BacktestConfig `yaml:"backtesting"`
	Plugins     plugins.PluginConfig   `yaml:"plugins"`
	Strategies  StrategiesConfig       `yaml:"strategies"`
	Simulation  SimulationConfig       `yaml:"simulation"`
}

// ServerConfig contains HTTP server configuration
type ServerConfig struct {
	Host            string        `yaml:"host"`
	Port            int           `yaml:"port"`
	UIPort          int           `yaml:"uiPort"`
	ShutdownTimeout time.Duration `yaml:"shutdownTimeout"`
	ReadTimeout     time.Duration `yaml:"readTimeout"`
	WriteTimeout    time.Duration `yaml:"writeTimeout"`
	EnableCORS      bool          `yaml:"enableCORS"`
	AllowedOrigins  []string      `yaml:"allowedOrigins"`
}

// FeedConfig contains configuration for a market data feed
type FeedConfig struct {
	Name          string   `yaml:"name"`
	Type          string   `yaml:"type"` // "websocket" or "fix"
	URL           string   `yaml:"url"`
	Subscriptions []string `yaml:"subscriptions"`
	Symbols       []string `yaml:"symbols"`
	APIKey        string   `yaml:"apiKey,omitempty"`
	APISecret     string   `yaml:"apiSecret,omitempty"`
}

// StrategiesConfig contains all strategy configurations
type StrategiesConfig struct {
	Arbitrage strategy.ArbitrageConfig `yaml:"arbitrage"`
}

// SimulationConfig contains configuration for simulation and backtesting
type SimulationConfig struct {
	PaperTrading PaperTradingConfig `yaml:"paperTrading"`
}

// PaperTradingConfig contains configuration for paper trading
type PaperTradingConfig struct {
	Enabled           bool               `yaml:"enabled"`
	InitialBalance    map[string]float64 `yaml:"initialBalance"`
	LatencySimulation bool               `yaml:"latencySimulation"`
	BaseLatency       int                `yaml:"baseLatency"`
	RandomLatency     int                `yaml:"randomLatency"`
	SlippageModel     string             `yaml:"slippageModel"`
	FixedSlippage     float64            `yaml:"fixedSlippage"`
	ExchangeFees      map[string]float64 `yaml:"exchangeFees"`
}

// Load loads configuration from a file
func Load(path string) (*Config, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}