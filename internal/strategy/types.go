package strategy

import (
	"context"
	"time"
	"velocimex/internal/orderbook"
)

// Strategy interface defines methods that all strategies must implement
type Strategy interface {
	GetName() string
	Execute() error
	GetSignals() []Signal
	GetResults() StrategyResults
	Start(ctx context.Context) error
	Stop() error
	IsRunning() bool
}

// ArbitrageStrategy interface defines methods specific to arbitrage strategies
type ArbitrageStrategy interface {
	Strategy
	GetOpportunities() []ArbitrageOpportunity
	SetOrderBookManager(manager *orderbook.Manager)
}

// Signal represents a trading signal
type Signal struct {
	Symbol    string    `json:"symbol"`
	Side      string    `json:"side"`
	Price     float64   `json:"price"`
	Volume    float64   `json:"volume"`
	Exchange  string    `json:"exchange"`
	Timestamp time.Time `json:"timestamp"`
}

// TradeSignal represents a detailed trading signal
type TradeSignal struct {
	Strategy   string    `json:"strategy"`
	Symbol     string    `json:"symbol"`
	Side       string    `json:"side"` // "buy" or "sell"
	Price      float64   `json:"price"`
	Volume     float64   `json:"volume"`
	Exchange   string    `json:"exchange"`
	Timestamp  time.Time `json:"timestamp"`
	Confidence float64   `json:"confidence"` // 0-1 scale
	Reason     string    `json:"reason"`
}

// Position represents a current trading position
type Position struct {
	Strategy   string    `json:"strategy"`
	Symbol     string    `json:"symbol"`
	Side       string    `json:"side"` // "long" or "short"
	EntryPrice float64   `json:"entryPrice"`
	Volume     float64   `json:"volume"`
	Exchange   string    `json:"exchange"`
	OpenTime   time.Time `json:"openTime"`
	PnL        float64   `json:"pnl"`
}

// Update represents a strategy update
type Update struct {
	ProfitLoss    float64  `json:"profitLoss"`
	Drawdown      float64  `json:"drawdown"`
	RecentSignals []Signal `json:"recentSignals"`
}

// StrategyMetrics represents performance metrics for a strategy
type StrategyMetrics struct {
	WinRate        float64 `json:"winRate"`
	AverageProfit  float64 `json:"averageProfit"`
	AverageLoss    float64 `json:"averageLoss"`
	ProfitFactor   float64 `json:"profitFactor"`
	SharpeRatio    float64 `json:"sharpeRatio"`
	DrawdownMax    float64 `json:"drawdownMax"`
	AverageLatency float64 `json:"averageLatency"`
}

// StrategyResults contains the current results of a strategy
type StrategyResults struct {
	Name             string          `json:"name"`
	Running          bool            `json:"running"`
	StartTime        time.Time       `json:"startTime"`
	LastUpdate       time.Time       `json:"lastUpdate"`
	SignalsGenerated int             `json:"signalsGenerated"`
	ProfitLoss       float64         `json:"profitLoss"`
	RecentSignals    []TradeSignal   `json:"recentSignals"`
	CurrentPositions []Position      `json:"currentPositions"`
	Metrics          StrategyMetrics `json:"metrics"`
}
