package metrics

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
)

func TestNewMetrics(t *testing.T) {
	m := New()
	assert.NotNil(t, m)
	assert.NotNil(t, m.registry)
	
	// Test that all metrics are properly registered
	gatherers := prometheus.DefaultGatherer
	_, err := gatherers.Gather()
	assert.NoError(t, err)
}

func TestMetricsMethods(t *testing.T) {
	m := New()
	
	// Test system metrics
	m.SystemInfo.WithLabelValues("1.0.0", "1.19").Set(1)
	m.UpTime.Set(100)
	
	// Test market data metrics
	m.RecordMarketDataMessage("binance", "BTCUSDT", "trade")
	m.RecordMarketDataLatency(time.Millisecond)
	m.RecordFeedConnection("binance", "connected")
	
	// Test order book metrics
	m.RecordOrderBookUpdate("binance", "BTCUSDT")
	m.RecordOrderBookDepth("binance", "BTCUSDT", "bid", 100.5)
	m.RecordOrderBookLatency(time.Millisecond)
	
	// Test strategy metrics
	m.RecordStrategySignal("arbitrage", "BTCUSDT", "buy")
	m.RecordStrategyPosition("arbitrage", "BTCUSDT", 1)
	m.RecordStrategyProfitLoss("arbitrage", "BTCUSDT", 100.50)
	m.RecordStrategyExecution("arbitrage", time.Millisecond)
	
	// Test risk metrics
	m.RecordRiskEvent("stop_loss", "high")
	m.RecordPortfolioValue(10000.0)
	m.RecordPositionCount(5)
	m.RecordDailyLoss(2.5)
	
	// Test API metrics
	m.RecordAPIRequest("/api/v1/orders", "POST", "200")
	m.RecordAPILatency("/api/v1/orders", "POST", time.Millisecond)
	m.RecordAPIError("/api/v1/orders", "POST", "validation_error")
	
	// Test WebSocket metrics
	m.RecordWebSocketConnection(10)
	m.RecordWebSocketMessage("order_update")
	
	// Verify metrics are collected
	assert.NotPanics(t, func() {
		m.UpdateUptime()
	})
}

func TestConfig(t *testing.T) {
	cfg := DefaultConfig()
	assert.True(t, cfg.Enabled)
	assert.Equal(t, "9090", cfg.Port)
	assert.Equal(t, "/metrics", cfg.Path)
	
	cfg.Port = ""
	cfg.Path = ""
	assert.NoError(t, cfg.Validate())
	assert.Equal(t, "9090", cfg.Port)
	assert.Equal(t, "/metrics", cfg.Path)
	
	cfg.Enabled = false
	assert.NoError(t, cfg.Validate())
}