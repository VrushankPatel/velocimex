package metrics

import (
	"time"
)

// Wrapper provides a simplified interface for metrics collection
type Wrapper struct {
	metrics *Metrics
	enabled bool
}

// NewWrapper creates a new metrics wrapper
func NewWrapper(metrics *Metrics, enabled bool) *Wrapper {
	return &Wrapper{
		metrics: metrics,
		enabled: enabled,
	}
}

// RecordMarketDataMessage records a market data message if metrics are enabled
func (w *Wrapper) RecordMarketDataMessage(exchange, symbol, msgType string) {
	if w.enabled {
		w.metrics.RecordMarketDataMessage(exchange, symbol, msgType)
	}
}

// RecordMarketDataLatency records market data latency if metrics are enabled
func (w *Wrapper) RecordMarketDataLatency(duration time.Duration) {
	if w.enabled {
		w.metrics.RecordMarketDataLatency(duration)
	}
}

// RecordOrderBookUpdate records an order book update if metrics are enabled
func (w *Wrapper) RecordOrderBookUpdate(exchange, symbol string) {
	if w.enabled {
		w.metrics.RecordOrderBookUpdate(exchange, symbol)
	}
}

// RecordOrderBookLatency records order book update latency if metrics are enabled
func (w *Wrapper) RecordOrderBookLatency(duration time.Duration) {
	if w.enabled {
		w.metrics.RecordOrderBookLatency(duration)
	}
}

// RecordStrategySignal records a strategy signal if metrics are enabled
func (w *Wrapper) RecordStrategySignal(strategy, symbol, side string) {
	if w.enabled {
		w.metrics.RecordStrategySignal(strategy, symbol, side)
	}
}

// RecordStrategyPosition records strategy position count if metrics are enabled
func (w *Wrapper) RecordStrategyPosition(strategy, symbol string, count float64) {
	if w.enabled {
		w.metrics.RecordStrategyPosition(strategy, symbol, count)
	}
}

// RecordStrategyProfitLoss records strategy profit/loss if metrics are enabled
func (w *Wrapper) RecordStrategyProfitLoss(strategy, symbol string, pnl float64) {
	if w.enabled {
		w.metrics.RecordStrategyProfitLoss(strategy, symbol, pnl)
	}
}

// RecordStrategyExecution records strategy execution duration if metrics are enabled
func (w *Wrapper) RecordStrategyExecution(strategy string, duration time.Duration) {
	if w.enabled {
		w.metrics.RecordStrategyExecution(strategy, duration)
	}
}

// RecordRiskEvent records a risk event if metrics are enabled
func (w *Wrapper) RecordRiskEvent(eventType, severity string) {
	if w.enabled {
		w.metrics.RecordRiskEvent(eventType, severity)
	}
}

// RecordPortfolioValue records portfolio value if metrics are enabled
func (w *Wrapper) RecordPortfolioValue(value float64) {
	if w.enabled {
		w.metrics.RecordPortfolioValue(value)
	}
}

// RecordPositionCount records position count if metrics are enabled
func (w *Wrapper) RecordPositionCount(count float64) {
	if w.enabled {
		w.metrics.RecordPositionCount(count)
	}
}

// RecordDailyLoss records daily loss percentage if metrics are enabled
func (w *Wrapper) RecordDailyLoss(loss float64) {
	if w.enabled {
		w.metrics.RecordDailyLoss(loss)
	}
}

// RecordAPIRequest records an API request if metrics are enabled
func (w *Wrapper) RecordAPIRequest(endpoint, method, status string) {
	if w.enabled {
		w.metrics.RecordAPIRequest(endpoint, method, status)
	}
}

// RecordAPILatency records API request latency if metrics are enabled
func (w *Wrapper) RecordAPILatency(endpoint, method string, duration time.Duration) {
	if w.enabled {
		w.metrics.RecordAPILatency(endpoint, method, duration)
	}
}

// RecordAPIError records an API error if metrics are enabled
func (w *Wrapper) RecordAPIError(endpoint, method, errorType string) {
	if w.enabled {
		w.metrics.RecordAPIError(endpoint, method, errorType)
	}
}

// RecordWebSocketConnection records WebSocket connection count if metrics are enabled
func (w *Wrapper) RecordWebSocketConnection(count int) {
	if w.enabled {
		w.metrics.RecordWebSocketConnection(count)
	}
}

// RecordWebSocketMessage records a WebSocket message if metrics are enabled
func (w *Wrapper) RecordWebSocketMessage(msgType string) {
	if w.enabled {
		w.metrics.RecordWebSocketMessage(msgType)
	}
}

// RecordOrderEvent records an order event if metrics are enabled
func (w *Wrapper) RecordOrderEvent(eventType, status string) {
	if w.enabled {
		w.metrics.RecordOrderEvent(eventType, status)
	}
}

// RecordOrderValue records order value if metrics are enabled
func (w *Wrapper) RecordOrderValue(value float64) {
	if w.enabled {
		w.metrics.RecordOrderValue(value)
	}
}

// RecordOrderFilled records order filled quantity if metrics are enabled
func (w *Wrapper) RecordOrderFilled(quantity float64) {
	if w.enabled {
		w.metrics.RecordOrderFilled(quantity)
	}
}

// RecordFeedConnection records feed connection status if metrics are enabled
func (w *Wrapper) RecordFeedConnection(feedName, status string) {
	if w.enabled {
		w.metrics.RecordFeedConnection(feedName, status)
	}
}

// UpdateUptime updates uptime metric if metrics are enabled
func (w *Wrapper) UpdateUptime() {
	if w.enabled {
		w.metrics.UpdateUptime()
	}
}