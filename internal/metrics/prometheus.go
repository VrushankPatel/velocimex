package metrics

import (
	"context"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics holds all Prometheus metrics
type Metrics struct {
	// System metrics
	SystemInfo *prometheus.GaugeVec
	UpTime     prometheus.Gauge
	
	// Market data metrics
	MarketDataMessages *prometheus.CounterVec
	MarketDataLatency  prometheus.Histogram
	FeedConnections    *prometheus.GaugeVec
	
	// Order book metrics
	OrderBookDepth      *prometheus.GaugeVec
	OrderBookUpdates    *prometheus.CounterVec
	OrderBookLatency    prometheus.Histogram
	
	// Order management metrics
	OrderEvents         *prometheus.CounterVec
	OrderValue          prometheus.Counter
	OrderFilled         prometheus.Counter
	
	// Strategy metrics
	StrategySignals     *prometheus.CounterVec
	StrategyPositions   *prometheus.GaugeVec
	StrategyProfitLoss  *prometheus.GaugeVec
	StrategyPerformance *prometheus.HistogramVec
	
	// Risk metrics
	RiskEvents        *prometheus.CounterVec
	PortfolioValue    prometheus.Gauge
	PositionCount     prometheus.Gauge
	DailyLoss         prometheus.Gauge
	
	// API metrics
	APIRequests   *prometheus.CounterVec
	APILatency    *prometheus.HistogramVec
	APIErrors     *prometheus.CounterVec
	
	// WebSocket metrics
	WebSocketConnections prometheus.Gauge
	WebSocketMessages    *prometheus.CounterVec
	
	// Registry
	registry *prometheus.Registry
}

// New creates a new metrics instance
func New() *Metrics {
	registry := prometheus.NewRegistry()
	
	m := &Metrics{
		registry: registry,
		
		// System metrics
		SystemInfo: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "velocimex_system_info",
				Help: "System information",
			},
			[]string{"version", "go_version"},
		),
		UpTime: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "velocimex_uptime_seconds",
				Help: "System uptime in seconds",
			},
		),
		
		// Market data metrics
		MarketDataMessages: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "velocimex_market_data_messages_total",
				Help: "Total number of market data messages received",
			},
			[]string{"exchange", "symbol", "type"},
		),
		MarketDataLatency: prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "velocimex_market_data_latency_microseconds",
				Help:    "Market data processing latency in microseconds",
				Buckets: prometheus.ExponentialBuckets(1, 2, 15),
			},
		),
		FeedConnections: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "velocimex_feed_connections",
				Help: "Number of active feed connections",
			},
			[]string{"exchange", "status"},
		),
		
		// Order book metrics
		OrderBookDepth: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "velocimex_order_book_depth",
				Help: "Current order book depth",
			},
			[]string{"exchange", "symbol", "side"},
		),
		OrderBookUpdates: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "velocimex_order_book_updates_total",
				Help: "Total number of order book updates",
			},
			[]string{"exchange", "symbol"},
		),
		OrderBookLatency: prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "velocimex_order_book_latency_microseconds",
				Help:    "Order book update latency in microseconds",
				Buckets: prometheus.ExponentialBuckets(1, 2, 15),
			},
		),
		
		// Order management metrics
		OrderEvents: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "velocimex_order_events_total",
				Help: "Total number of order events",
			},
			[]string{"event_type", "status"},
		),
		OrderValue: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "velocimex_order_value_total",
				Help: "Total value of all orders",
			},
		),
		OrderFilled: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "velocimex_order_filled_total",
				Help: "Total quantity of filled orders",
			},
		),
		
		// Strategy metrics
		StrategySignals: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "velocimex_strategy_signals_total",
				Help: "Total number of strategy signals generated",
			},
			[]string{"strategy", "symbol", "side"},
		),
		StrategyPositions: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "velocimex_strategy_positions",
				Help: "Current number of strategy positions",
			},
			[]string{"strategy", "symbol"},
		),
		StrategyProfitLoss: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "velocimex_strategy_profit_loss",
				Help: "Current profit/loss for strategies",
			},
			[]string{"strategy", "symbol"},
		),
		StrategyPerformance: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "velocimex_strategy_execution_duration_microseconds",
				Help:    "Strategy execution duration in microseconds",
				Buckets: prometheus.ExponentialBuckets(1, 2, 15),
			},
			[]string{"strategy"},
		),
		
		// Risk metrics
		RiskEvents: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "velocimex_risk_events_total",
				Help: "Total number of risk events triggered",
			},
			[]string{"type", "severity"},
		),
		PortfolioValue: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "velocimex_portfolio_value",
				Help: "Current portfolio value in USD",
			},
		),
		PositionCount: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "velocimex_position_count",
				Help: "Current number of open positions",
			},
		),
		DailyLoss: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "velocimex_daily_loss_percentage",
				Help: "Daily loss as percentage of portfolio",
			},
		),
		
		// API metrics
		APIRequests: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "velocimex_api_requests_total",
				Help: "Total number of API requests",
			},
			[]string{"endpoint", "method", "status"},
		),
		APILatency: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "velocimex_api_latency_milliseconds",
				Help:    "API request latency in milliseconds",
				Buckets: []float64{1, 5, 10, 25, 50, 100, 250, 500, 1000},
			},
			[]string{"endpoint", "method"},
		),
		APIErrors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "velocimex_api_errors_total",
				Help: "Total number of API errors",
			},
			[]string{"endpoint", "method", "error_type"},
		),
		
		// WebSocket metrics
		WebSocketConnections: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "velocimex_websocket_connections",
				Help: "Current number of WebSocket connections",
			},
		),
		WebSocketMessages: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "velocimex_websocket_messages_total",
				Help: "Total number of WebSocket messages sent",
			},
			[]string{"type"},
		),
	}
	
	// Register all metrics
	registry.MustRegister(
		m.SystemInfo,
		m.UpTime,
		m.MarketDataMessages,
		m.MarketDataLatency,
		m.FeedConnections,
		m.OrderBookDepth,
		m.OrderBookUpdates,
		m.OrderBookLatency,
		m.StrategySignals,
		m.StrategyPositions,
		m.StrategyProfitLoss,
		m.StrategyPerformance,
		m.RiskEvents,
		m.PortfolioValue,
		m.PositionCount,
		m.DailyLoss,
		m.APIRequests,
		m.APILatency,
		m.APIErrors,
		m.WebSocketConnections,
		m.WebSocketMessages,
	)
	
	// Set system info
	m.SystemInfo.WithLabelValues("1.0.0", "1.19").Set(1)
	m.UpTime.SetToCurrentTime()
	
	return m
}

// GetRegistry returns the Prometheus registry
func (m *Metrics) GetRegistry() *prometheus.Registry {
	return m.registry
}

// Start starts the metrics server
func (m *Metrics) Start(ctx context.Context, addr string) error {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{}))
	
	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}
	
	go func() {
		<-ctx.Done()
		server.Shutdown(ctx)
	}()
	
	return server.ListenAndServe()
}

// RecordMarketDataMessage records a market data message
func (m *Metrics) RecordMarketDataMessage(exchange, symbol, msgType string) {
	m.MarketDataMessages.WithLabelValues(exchange, symbol, msgType).Inc()
}

// RecordMarketDataLatency records market data processing latency
func (m *Metrics) RecordMarketDataLatency(duration time.Duration) {
	m.MarketDataLatency.Observe(float64(duration.Microseconds()))
}

// RecordFeedConnection records feed connection status
func (m *Metrics) RecordFeedConnection(exchange, status string) {
	m.FeedConnections.WithLabelValues(exchange, status).Set(1)
}

// RecordOrderBookUpdate records an order book update
func (m *Metrics) RecordOrderBookUpdate(exchange, symbol string) {
	m.OrderBookUpdates.WithLabelValues(exchange, symbol).Inc()
}

// RecordOrderBookDepth records order book depth
func (m *Metrics) RecordOrderBookDepth(exchange, symbol, side string, depth float64) {
	m.OrderBookDepth.WithLabelValues(exchange, symbol, side).Set(depth)
}

// RecordOrderBookLatency records order book update latency
func (m *Metrics) RecordOrderBookLatency(duration time.Duration) {
	m.OrderBookLatency.Observe(float64(duration.Microseconds()))
}

// RecordOrderEvent records an order event
func (m *Metrics) RecordOrderEvent(eventType, status string) {
	m.OrderEvents.WithLabelValues(eventType, status).Inc()
}

// RecordOrderValue records order value
func (m *Metrics) RecordOrderValue(value float64) {
	m.OrderValue.Add(value)
}

// RecordOrderFilled records order filled quantity
func (m *Metrics) RecordOrderFilled(quantity float64) {
	m.OrderFilled.Add(quantity)
}

// RecordPositionValue records position value
func (m *Metrics) RecordPositionValue(value float64) {
	m.PortfolioValue.Add(value)
}

// RecordPositionPNL records position PNL
func (m *Metrics) RecordPositionPNL(pnl float64) {
	m.DailyLoss.Add(pnl)
}

// RecordStrategySignal records a strategy signal
func (m *Metrics) RecordStrategySignal(strategy, symbol, side string) {
	m.StrategySignals.WithLabelValues(strategy, symbol, side).Inc()
}

// RecordStrategyPosition records strategy position count
func (m *Metrics) RecordStrategyPosition(strategy, symbol string, count float64) {
	m.StrategyPositions.WithLabelValues(strategy, symbol).Set(count)
}

// RecordStrategyProfitLoss records strategy profit/loss
func (m *Metrics) RecordStrategyProfitLoss(strategy, symbol string, pnl float64) {
	m.StrategyProfitLoss.WithLabelValues(strategy, symbol).Set(pnl)
}

// RecordStrategyExecution records strategy execution duration
func (m *Metrics) RecordStrategyExecution(strategy string, duration time.Duration) {
	m.StrategyPerformance.WithLabelValues(strategy).Observe(float64(duration.Microseconds()))
}

// RecordRiskEvent records a risk event
func (m *Metrics) RecordRiskEvent(eventType, severity string) {
	m.RiskEvents.WithLabelValues(eventType, severity).Inc()
}

// RecordPortfolioValue records portfolio value
func (m *Metrics) RecordPortfolioValue(value float64) {
	m.PortfolioValue.Set(value)
}

// RecordPositionCount records position count
func (m *Metrics) RecordPositionCount(count float64) {
	m.PositionCount.Set(count)
}

// RecordDailyLoss records daily loss percentage
func (m *Metrics) RecordDailyLoss(loss float64) {
	m.DailyLoss.Set(loss)
}

// RecordAPIRequest records an API request
func (m *Metrics) RecordAPIRequest(endpoint, method, status string) {
	m.APIRequests.WithLabelValues(endpoint, method, status).Inc()
}

// RecordAPILatency records API request latency
func (m *Metrics) RecordAPILatency(endpoint, method string, duration time.Duration) {
	m.APILatency.WithLabelValues(endpoint, method).Observe(float64(duration.Milliseconds()))
}

// RecordAPIError records an API error
func (m *Metrics) RecordAPIError(endpoint, method, errorType string) {
	m.APIErrors.WithLabelValues(endpoint, method, errorType).Inc()
}

// RecordWebSocketConnection records WebSocket connection count
func (m *Metrics) RecordWebSocketConnection(count int) {
	m.WebSocketConnections.Set(float64(count))
}

// RecordWebSocketMessage records a WebSocket message
func (m *Metrics) RecordWebSocketMessage(msgType string) {
	m.WebSocketMessages.WithLabelValues(msgType).Inc()
}

// UpdateUptime updates the uptime metric
func (m *Metrics) UpdateUptime() {
	m.UpTime.SetToCurrentTime()
}