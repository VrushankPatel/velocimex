package backtesting

import (
	"time"

	"github.com/shopspring/decimal"
	"velocimex/internal/risk"
	"velocimex/internal/strategy"
)

// BacktestConfig represents backtesting configuration
type BacktestConfig struct {
	StartDate        time.Time     `json:"start_date"`
	EndDate          time.Time     `json:"end_date"`
	InitialCapital   decimal.Decimal `json:"initial_capital"`
	Commission       decimal.Decimal `json:"commission"` // Per trade commission
	Slippage         decimal.Decimal `json:"slippage"`   // Slippage percentage
	Latency          time.Duration `json:"latency"`     // Simulated latency
	DataFrequency    time.Duration `json:"data_frequency"` // Data update frequency
	RiskManagement   bool          `json:"risk_management"`
	RiskConfig       risk.RiskConfig `json:"risk_config"`
	Symbols          []string      `json:"symbols"`
	Exchanges        []string      `json:"exchanges"`
	StrategyConfig   map[string]interface{} `json:"strategy_config"`
}

// DefaultBacktestConfig returns default backtesting configuration
func DefaultBacktestConfig() BacktestConfig {
	return BacktestConfig{
		StartDate:        time.Now().AddDate(0, -1, 0), // 1 month ago
		EndDate:          time.Now(),
		InitialCapital:   decimal.NewFromFloat(100000), // $100,000
		Commission:       decimal.NewFromFloat(0.001),   // 0.1% commission
		Slippage:         decimal.NewFromFloat(0.0005), // 0.05% slippage
		Latency:          10 * time.Millisecond,        // 10ms latency
		DataFrequency:    1 * time.Second,              // 1 second data updates
		RiskManagement:   true,
		RiskConfig:       risk.DefaultRiskConfig(),
		Symbols:          []string{"BTC/USD", "ETH/USD"},
		Exchanges:        []string{"binance", "coinbase"},
		StrategyConfig:   make(map[string]interface{}),
	}
}

// BacktestResult represents the results of a backtest
type BacktestResult struct {
	Config           BacktestConfig     `json:"config"`
	StartTime        time.Time          `json:"start_time"`
	EndTime          time.Time          `json:"end_time"`
	Duration         time.Duration      `json:"duration"`
	
	// Portfolio metrics
	InitialCapital   decimal.Decimal    `json:"initial_capital"`
	FinalCapital     decimal.Decimal    `json:"final_capital"`
	TotalReturn      decimal.Decimal    `json:"total_return"`
	TotalReturnPct   decimal.Decimal    `json:"total_return_pct"`
	
	// Trading metrics
	TotalTrades      int                `json:"total_trades"`
	WinningTrades    int                `json:"winning_trades"`
	LosingTrades     int                `json:"losing_trades"`
	WinRate          decimal.Decimal    `json:"win_rate"`
	
	// Performance metrics
	SharpeRatio      decimal.Decimal    `json:"sharpe_ratio"`
	SortinoRatio     decimal.Decimal    `json:"sortino_ratio"`
	CalmarRatio      decimal.Decimal    `json:"calmar_ratio"`
	MaxDrawdown      decimal.Decimal    `json:"max_drawdown"`
	MaxDrawdownPct   decimal.Decimal    `json:"max_drawdown_pct"`
	Volatility       decimal.Decimal    `json:"volatility"`
	
	// Risk metrics
	VaR95            decimal.Decimal    `json:"var_95"`
	VaR99            decimal.Decimal    `json:"var_99"`
	Beta             decimal.Decimal    `json:"beta"`
	Alpha            decimal.Decimal    `json:"alpha"`
	
	// Execution metrics
	TotalCommission  decimal.Decimal    `json:"total_commission"`
	TotalSlippage    decimal.Decimal    `json:"total_slippage"`
	AvgExecutionTime time.Duration      `json:"avg_execution_time"`
	
	// Detailed data
	Trades           []*BacktestTrade   `json:"trades"`
	PortfolioHistory []*PortfolioSnapshot `json:"portfolio_history"`
	RiskEvents       []*risk.RiskEvent  `json:"risk_events"`
	
	// Strategy-specific metrics
	StrategyMetrics  map[string]interface{} `json:"strategy_metrics"`
}

// BacktestTrade represents a trade executed during backtesting
type BacktestTrade struct {
	ID              string          `json:"id"`
	Symbol          string          `json:"symbol"`
	Exchange        string          `json:"exchange"`
	Side            string          `json:"side"` // "BUY" or "SELL"
	Quantity        decimal.Decimal `json:"quantity"`
	EntryPrice      decimal.Decimal `json:"entry_price"`
	ExitPrice       decimal.Decimal `json:"exit_price"`
	EntryTime       time.Time       `json:"entry_time"`
	ExitTime        time.Time       `json:"exit_time"`
	Duration        time.Duration   `json:"duration"`
	PnL             decimal.Decimal `json:"pnl"`
	PnLPct          decimal.Decimal `json:"pnl_pct"`
	Commission      decimal.Decimal `json:"commission"`
	Slippage        decimal.Decimal `json:"slippage"`
	StrategyID      string          `json:"strategy_id"`
	StrategyName    string          `json:"strategy_name"`
	Metadata        map[string]interface{} `json:"metadata"`
}

// PortfolioSnapshot represents a snapshot of the portfolio at a point in time
type PortfolioSnapshot struct {
	Timestamp       time.Time       `json:"timestamp"`
	TotalValue      decimal.Decimal `json:"total_value"`
	CashBalance     decimal.Decimal `json:"cash_balance"`
	InvestedValue   decimal.Decimal `json:"invested_value"`
	UnrealizedPNL   decimal.Decimal `json:"unrealized_pnl"`
	RealizedPNL     decimal.Decimal `json:"realized_pnl"`
	DailyPNL        decimal.Decimal `json:"daily_pnl"`
	Positions       map[string]*risk.Position `json:"positions"`
	RiskMetrics     *risk.RiskMetrics `json:"risk_metrics"`
}

// HistoricalData represents historical market data
type HistoricalData struct {
	Symbol      string                 `json:"symbol"`
	Exchange    string                 `json:"exchange"`
	DataPoints  []*DataPoint           `json:"data_points"`
	StartTime   time.Time              `json:"start_time"`
	EndTime     time.Time              `json:"end_time"`
	Frequency   time.Duration          `json:"frequency"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// DataPoint represents a single data point in historical data
type DataPoint struct {
	Timestamp   time.Time       `json:"timestamp"`
	Open        decimal.Decimal `json:"open"`
	High        decimal.Decimal `json:"high"`
	Low         decimal.Decimal `json:"low"`
	Close       decimal.Decimal `json:"close"`
	Volume      decimal.Decimal `json:"volume"`
	Bid         decimal.Decimal `json:"bid"`
	Ask         decimal.Decimal `json:"ask"`
	BidSize     decimal.Decimal `json:"bid_size"`
	AskSize     decimal.Decimal `json:"ask_size"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// BacktestEngine defines the interface for backtesting engines
type BacktestEngine interface {
	// Configuration
	SetConfig(config BacktestConfig) error
	GetConfig() BacktestConfig
	
	// Data management
	LoadHistoricalData(symbol, exchange string, startDate, endDate time.Time) (*HistoricalData, error)
	AddHistoricalData(data *HistoricalData) error
	GetAvailableData() map[string][]string // symbol -> exchanges
	
	// Strategy management
	RegisterStrategy(strategy strategy.Strategy) error
	GetRegisteredStrategies() []strategy.Strategy
	
	// Execution
	RunBacktest() (*BacktestResult, error)
	RunBacktestWithStrategy(strategyID string) (*BacktestResult, error)
	
	// Analysis
	AnalyzeResult(result *BacktestResult) (*BacktestAnalysis, error)
	GenerateReport(result *BacktestResult) (*BacktestReport, error)
	
	// Control
	Start() error
	Stop() error
	IsRunning() bool
	Pause() error
	Resume() error
}

// BacktestAnalysis represents detailed analysis of backtest results
type BacktestAnalysis struct {
	Result              *BacktestResult `json:"result"`
	
	// Performance analysis
	MonthlyReturns      map[string]decimal.Decimal `json:"monthly_returns"`
	DailyReturns        []decimal.Decimal           `json:"daily_returns"`
	ReturnDistribution  map[string]int             `json:"return_distribution"`
	
	// Risk analysis
	RiskMetrics         *risk.RiskMetrics           `json:"risk_metrics"`
	DrawdownPeriods     []*DrawdownPeriod           `json:"drawdown_periods"`
	VolatilityAnalysis  *VolatilityAnalysis         `json:"volatility_analysis"`
	
	// Trade analysis
	TradeAnalysis       *TradeAnalysis              `json:"trade_analysis"`
	SymbolAnalysis      map[string]*SymbolAnalysis  `json:"symbol_analysis"`
	ExchangeAnalysis    map[string]*ExchangeAnalysis `json:"exchange_analysis"`
	
	// Strategy analysis
	StrategyAnalysis    map[string]interface{}      `json:"strategy_analysis"`
	OptimizationSuggestions []string                `json:"optimization_suggestions"`
}

// DrawdownPeriod represents a period of drawdown
type DrawdownPeriod struct {
	StartTime   time.Time       `json:"start_time"`
	EndTime     time.Time       `json:"end_time"`
	Duration    time.Duration   `json:"duration"`
	MaxDrawdown decimal.Decimal `json:"max_drawdown"`
	RecoveryTime time.Duration `json:"recovery_time"`
}

// VolatilityAnalysis represents volatility analysis
type VolatilityAnalysis struct {
	DailyVolatility    decimal.Decimal `json:"daily_volatility"`
	MonthlyVolatility  decimal.Decimal `json:"monthly_volatility"`
	AnnualVolatility   decimal.Decimal `json:"annual_volatility"`
	VolatilityClusters []*VolatilityCluster `json:"volatility_clusters"`
}

// VolatilityCluster represents a period of high/low volatility
type VolatilityCluster struct {
	StartTime   time.Time       `json:"start_time"`
	EndTime     time.Time       `json:"end_time"`
	Volatility  decimal.Decimal `json:"volatility"`
	Type        string          `json:"type"` // "HIGH" or "LOW"
}

// TradeAnalysis represents analysis of trades
type TradeAnalysis struct {
	AvgTradeDuration    time.Duration   `json:"avg_trade_duration"`
	AvgWinSize          decimal.Decimal `json:"avg_win_size"`
	AvgLossSize         decimal.Decimal `json:"avg_loss_size"`
	ProfitFactor        decimal.Decimal `json:"profit_factor"`
	Expectancy          decimal.Decimal `json:"expectancy"`
	ConsecutiveWins     int             `json:"consecutive_wins"`
	ConsecutiveLosses   int             `json:"consecutive_losses"`
	MaxConsecutiveWins  int             `json:"max_consecutive_wins"`
	MaxConsecutiveLosses int            `json:"max_consecutive_losses"`
}

// SymbolAnalysis represents analysis for a specific symbol
type SymbolAnalysis struct {
	Symbol              string          `json:"symbol"`
	TotalTrades         int             `json:"total_trades"`
	WinningTrades       int             `json:"winning_trades"`
	LosingTrades        int             `json:"losing_trades"`
	WinRate             decimal.Decimal `json:"win_rate"`
	TotalPnL            decimal.Decimal `json:"total_pnl"`
	AvgPnL              decimal.Decimal `json:"avg_pnl"`
	MaxPnL              decimal.Decimal `json:"max_pnl"`
	MinPnL              decimal.Decimal `json:"min_pnl"`
	Volatility          decimal.Decimal `json:"volatility"`
	SharpeRatio         decimal.Decimal `json:"sharpe_ratio"`
}

// ExchangeAnalysis represents analysis for a specific exchange
type ExchangeAnalysis struct {
	Exchange            string          `json:"exchange"`
	TotalTrades         int             `json:"total_trades"`
	TotalVolume         decimal.Decimal `json:"total_volume"`
	AvgExecutionTime    time.Duration   `json:"avg_execution_time"`
	TotalCommission     decimal.Decimal `json:"total_commission"`
	TotalSlippage       decimal.Decimal `json:"total_slippage"`
	ExecutionQuality    decimal.Decimal `json:"execution_quality"`
}

// BacktestReport represents a comprehensive backtest report
type BacktestReport struct {
	Summary             *BacktestSummary `json:"summary"`
	Analysis            *BacktestAnalysis `json:"analysis"`
	Charts              map[string]interface{} `json:"charts"`
	Recommendations     []string         `json:"recommendations"`
	GeneratedAt         time.Time        `json:"generated_at"`
	ReportVersion       string           `json:"report_version"`
}

// BacktestSummary represents a summary of backtest results
type BacktestSummary struct {
	Period              string          `json:"period"`
	InitialCapital      decimal.Decimal `json:"initial_capital"`
	FinalCapital        decimal.Decimal `json:"final_capital"`
	TotalReturn         decimal.Decimal `json:"total_return"`
	TotalReturnPct      decimal.Decimal `json:"total_return_pct"`
	AnnualizedReturn    decimal.Decimal `json:"annualized_return"`
	MaxDrawdown         decimal.Decimal `json:"max_drawdown"`
	SharpeRatio         decimal.Decimal `json:"sharpe_ratio"`
	TotalTrades         int             `json:"total_trades"`
	WinRate             decimal.Decimal `json:"win_rate"`
	ProfitFactor        decimal.Decimal `json:"profit_factor"`
	RiskAdjustedReturn  decimal.Decimal `json:"risk_adjusted_return"`
}
