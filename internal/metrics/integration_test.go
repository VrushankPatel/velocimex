package metrics

import (
	"context"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"velocimex/internal/config"
	"velocimex/internal/feeds"
	"velocimex/internal/normalizer"
	"velocimex/internal/orderbook"
	"velocimex/internal/risk"
	"velocimex/internal/strategy"
)

func TestMetricsIntegration(t *testing.T) {
	// Create metrics instance
	metrics := New()
	assert.NotNil(t, metrics)

	// Create metrics wrapper
	wrapper := NewWrapper(metrics, true)
	assert.NotNil(t, wrapper)

	// Test configuration
	cfg := config.Config{
		Metrics: config.MetricsConfig{
			Enabled: true,
			Port:    "9091",
			Path:    "/metrics",
		},
		Feeds: []config.FeedConfig{
			{
				Name: "test",
				Type: "websocket",
				URL:  "ws://localhost:8080",
				Symbols: []string{"BTCUSDT"},
			},
		},
		Risk: risk.Config{
			MaxRiskPerTrade: 0.02,
			MaxDailyLoss:    0.05,
			MaxPositions:    10,
		},
	}

	// Test metrics with feeds manager
	normalizer := normalizer.New()
	orderBookManager := orderbook.NewManager()
	feedManager := feeds.NewManager(normalizer, cfg.Feeds, wrapper)
	assert.NotNil(t, feedManager)

	// Test metrics with risk manager
	riskManager := risk.NewRiskManager(cfg.Risk, wrapper)
	assert.NotNil(t, riskManager)

	// Test metrics with strategy engine
	strategyEngine := strategy.NewEngine(orderBookManager, wrapper)
	assert.NotNil(t, strategyEngine)

	// Test metrics collection
	wrapper.RecordMarketDataMessage("binance", "BTCUSDT", "trade")
	wrapper.RecordMarketDataLatency(time.Millisecond)
	wrapper.RecordFeedConnection("binance", "connected")
	wrapper.RecordOrderBookUpdate("binance", "BTCUSDT")
	wrapper.RecordOrderBookDepth("binance", "BTCUSDT", "bid", 100.5)
	wrapper.RecordStrategySignal("arbitrage", "BTCUSDT", "buy")
	wrapper.RecordStrategyPosition("arbitrage", "BTCUSDT", 1)
	wrapper.RecordStrategyProfitLoss("arbitrage", "BTCUSDT", 100.50)
	wrapper.RecordStrategyExecution("arbitrage", time.Millisecond)
	wrapper.RecordRiskEvent("stop_loss", "high")
	wrapper.RecordPortfolioValue(10000.0)
	wrapper.RecordPositionCount(5)
	wrapper.RecordDailyLoss(2.5)
	wrapper.RecordAPIRequest("/api/v1/orders", "POST", "200")
	wrapper.RecordAPILatency("/api/v1/orders", "POST", time.Millisecond)
	wrapper.RecordAPIError("/api/v1/orders", "POST", "validation_error")
	wrapper.RecordWebSocketConnection(10)
	wrapper.RecordWebSocketMessage("order_update")

	// Test metrics registry
	registry := metrics.GetRegistry()
	assert.NotNil(t, registry)

	// Test metrics gathering
	gatherers := prometheus.Gatherers{registry}
	metricFamilies, err := gatherers.Gather()
	require.NoError(t, err)
	assert.NotEmpty(t, metricFamilies)

	// Verify specific metrics exist
	metricNames := []string{
		"velocimex_system_info",
		"velocimex_uptime_seconds",
		"velocimex_market_data_messages_total",
		"velocimex_market_data_latency_microseconds",
		"velocimex_feed_connections",
		"velocimex_order_book_depth",
		"velocimex_order_book_updates_total",
		"velocimex_order_book_latency_microseconds",
		"velocimex_strategy_signals_total",
		"velocimex_strategy_positions",
		"velocimex_strategy_profit_loss",
		"velocimex_strategy_execution_duration_microseconds",
		"velocimex_risk_events_total",
		"velocimex_portfolio_value",
		"velocimex_position_count",
		"velocimex_daily_loss_percentage",
		"velocimex_api_requests_total",
		"velocimex_api_latency_milliseconds",
		"velocimex_api_errors_total",
		"velocimex_websocket_connections",
		"velocimex_websocket_messages_total",
	}

	foundMetrics := make(map[string]bool)
	for _, mf := range metricFamilies {
		foundMetrics[*mf.Name] = true
	}

	for _, expected := range metricNames {
		assert.True(t, foundMetrics[expected], "Metric %s not found", expected)
	}

	// Test with disabled metrics
	wrapperDisabled := NewWrapper(metrics, false)
	assert.NotNil(t, wrapperDisabled)
	
	// Should not panic when metrics are disabled
	assert.NotPanics(t, func() {
		wrapperDisabled.RecordMarketDataMessage("binance", "BTCUSDT", "trade")
		wrapperDisabled.RecordStrategySignal("arbitrage", "BTCUSDT", "buy")
	})

	// Test configuration validation
	metricsConfig := Config{
		Enabled: true,
		Port:    "9091",
		Path:    "/metrics",
	}
	assert.NoError(t, metricsConfig.Validate())

	metricsConfig.Port = ""
	metricsConfig.Path = ""
	assert.NoError(t, metricsConfig.Validate())
	assert.Equal(t, "9090", metricsConfig.Port)
	assert.Equal(t, "/metrics", metricsConfig.Path)

	// Test context cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		assert.NoError(t, metrics.Start(ctx, ":9092"))
	}()

	// Allow some time for server startup
	time.Sleep(100 * time.Millisecond)
	cancel()
}