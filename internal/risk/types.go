package risk

import (
	"time"

	"github.com/shopspring/decimal"
)

// RiskLevel represents the risk level of a position or portfolio
type RiskLevel string

const (
	RiskLevelLow    RiskLevel = "LOW"
	RiskLevelMedium RiskLevel = "MEDIUM"
	RiskLevelHigh   RiskLevel = "HIGH"
	RiskLevelCritical RiskLevel = "CRITICAL"
)

// RiskEvent represents a risk-related event
type RiskEvent struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Severity    RiskLevel              `json:"severity"`
	Message     string                 `json:"message"`
	Symbol      string                 `json:"symbol,omitempty"`
	Exchange    string                 `json:"exchange,omitempty"`
	Value       decimal.Decimal        `json:"value,omitempty"`
	Threshold   decimal.Decimal        `json:"threshold,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// Position represents a trading position for risk calculation
type Position struct {
	Symbol       string          `json:"symbol"`
	Exchange     string          `json:"exchange"`
	Side         string          `json:"side"` // "LONG" or "SHORT"
	Quantity     decimal.Decimal `json:"quantity"`
	EntryPrice   decimal.Decimal `json:"entry_price"`
	CurrentPrice decimal.Decimal `json:"current_price"`
	MarketValue  decimal.Decimal `json:"market_value"`
	UnrealizedPNL decimal.Decimal `json:"unrealized_pnl"`
	RealizedPNL  decimal.Decimal `json:"realized_pnl"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

// Portfolio represents the current portfolio state
type Portfolio struct {
	TotalValue     decimal.Decimal `json:"total_value"`
	CashBalance    decimal.Decimal `json:"cash_balance"`
	InvestedValue  decimal.Decimal `json:"invested_value"`
	UnrealizedPNL  decimal.Decimal `json:"unrealized_pnl"`
	RealizedPNL    decimal.Decimal `json:"realized_pnl"`
	DailyPNL       decimal.Decimal `json:"daily_pnl"`
	Positions      map[string]*Position `json:"positions"`
	LastUpdated    time.Time       `json:"last_updated"`
}

// RiskLimits represents risk management limits
type RiskLimits struct {
	MaxPositionSize     decimal.Decimal `json:"max_position_size"`
	MaxPortfolioValue   decimal.Decimal `json:"max_portfolio_value"`
	MaxDailyLoss        decimal.Decimal `json:"max_daily_loss"`
	MaxDrawdown         decimal.Decimal `json:"max_drawdown"`
	MaxConcentration    decimal.Decimal `json:"max_concentration"` // Max % in single position
	MaxLeverage         decimal.Decimal `json:"max_leverage"`
	StopLossPercentage  decimal.Decimal `json:"stop_loss_percentage"`
	TakeProfitPercentage decimal.Decimal `json:"take_profit_percentage"`
}

// RiskMetrics represents calculated risk metrics
type RiskMetrics struct {
	PortfolioValue     decimal.Decimal `json:"portfolio_value"`
	TotalExposure      decimal.Decimal `json:"total_exposure"`
	Leverage           decimal.Decimal `json:"leverage"`
	ConcentrationRisk  decimal.Decimal `json:"concentration_risk"`
	VaR95             decimal.Decimal `json:"var_95"` // Value at Risk 95%
	VaR99             decimal.Decimal `json:"var_99"` // Value at Risk 99%
	MaxDrawdown       decimal.Decimal `json:"max_drawdown"`
	SharpeRatio       decimal.Decimal `json:"sharpe_ratio"`
	SortinoRatio      decimal.Decimal `json:"sortino_ratio"`
	CalmarRatio       decimal.Decimal `json:"calmar_ratio"`
	Beta              decimal.Decimal `json:"beta"`
	Alpha             decimal.Decimal `json:"alpha"`
	Volatility        decimal.Decimal `json:"volatility"`
	LastUpdated       time.Time       `json:"last_updated"`
}

// RiskConfig represents risk management configuration
type RiskConfig struct {
	Enabled             bool            `json:"enabled"`
	UpdateInterval      time.Duration   `json:"update_interval"`
	AlertThresholds     RiskLimits      `json:"alert_thresholds"`
	AutoStopLoss        bool            `json:"auto_stop_loss"`
	AutoTakeProfit      bool            `json:"auto_take_profit"`
	MaxOpenPositions    int             `json:"max_open_positions"`
	PositionSizingMode  string          `json:"position_sizing_mode"` // "FIXED", "PERCENTAGE", "KELLY"
	DefaultPositionSize decimal.Decimal `json:"default_position_size"`
	RiskFreeRate        decimal.Decimal `json:"risk_free_rate"`
	LookbackPeriod      int             `json:"lookback_period"` // Days for historical calculations
}

// DefaultRiskConfig returns default risk management configuration
func DefaultRiskConfig() RiskConfig {
	return RiskConfig{
		Enabled:             true,
		UpdateInterval:      1 * time.Second,
		AlertThresholds: RiskLimits{
			MaxPositionSize:     decimal.NewFromFloat(10000), // $10,000 max position
			MaxPortfolioValue:   decimal.NewFromFloat(100000), // $100,000 max portfolio
			MaxDailyLoss:        decimal.NewFromFloat(5000),  // $5,000 max daily loss
			MaxDrawdown:         decimal.NewFromFloat(0.1),   // 10% max drawdown
			MaxConcentration:    decimal.NewFromFloat(0.2),   // 20% max concentration
			MaxLeverage:         decimal.NewFromFloat(2.0),   // 2x max leverage
			StopLossPercentage:  decimal.NewFromFloat(0.05),  // 5% stop loss
			TakeProfitPercentage: decimal.NewFromFloat(0.1),  // 10% take profit
		},
		AutoStopLoss:        true,
		AutoTakeProfit:      true,
		MaxOpenPositions:    10,
		PositionSizingMode:  "PERCENTAGE",
		DefaultPositionSize: decimal.NewFromFloat(0.02), // 2% of portfolio
		RiskFreeRate:        decimal.NewFromFloat(0.02), // 2% risk-free rate
		LookbackPeriod:      30, // 30 days
	}
}

// RiskManager defines the interface for risk management
type RiskManager interface {
	// Configuration
	SetConfig(config RiskConfig) error
	GetConfig() RiskConfig
	
	// Portfolio management
	UpdatePortfolio(portfolio *Portfolio) error
	GetPortfolio() *Portfolio
	GetRiskMetrics() *RiskMetrics
	
	// Position management
	AddPosition(position *Position) error
	UpdatePosition(symbol, exchange string, price decimal.Decimal) error
	RemovePosition(symbol, exchange string) error
	GetPositions() map[string]*Position
	
	// Risk checks
	CheckOrderRisk(symbol, exchange string, side string, quantity, price decimal.Decimal) (*RiskEvent, error)
	CheckPortfolioRisk() ([]*RiskEvent, error)
	CheckPositionRisk(symbol, exchange string) (*RiskEvent, error)
	
	// Risk events
	GetRiskEvents(filters map[string]interface{}) ([]*RiskEvent, error)
	SubscribeToRiskEvents(callback func(*RiskEvent)) error
	
	// Control
	Start() error
	Stop() error
	IsRunning() bool
}
