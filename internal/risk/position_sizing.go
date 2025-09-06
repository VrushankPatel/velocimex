package risk

import (
	"math"

	"github.com/shopspring/decimal"
)

// PositionSizingCalculator calculates optimal position sizes
type PositionSizingCalculator struct {
	config RiskConfig
}

// NewPositionSizingCalculator creates a new position sizing calculator
func NewPositionSizingCalculator(config RiskConfig) *PositionSizingCalculator {
	return &PositionSizingCalculator{
		config: config,
	}
}

// CalculatePositionSize calculates the optimal position size for an order
func (psc *PositionSizingCalculator) CalculatePositionSize(
	portfolio *Portfolio,
	symbol, exchange string,
	entryPrice, stopLossPrice decimal.Decimal,
	riskAmount decimal.Decimal,
) (decimal.Decimal, error) {
	
	switch psc.config.PositionSizingMode {
	case "FIXED":
		return psc.calculateFixedSize(portfolio, riskAmount)
	case "PERCENTAGE":
		return psc.calculatePercentageSize(portfolio, riskAmount)
	case "KELLY":
		return psc.calculateKellySize(portfolio, symbol, exchange, entryPrice, stopLossPrice)
	default:
		return psc.calculatePercentageSize(portfolio, riskAmount)
	}
}

// calculateFixedSize calculates position size using fixed amount
func (psc *PositionSizingCalculator) calculateFixedSize(portfolio *Portfolio, riskAmount decimal.Decimal) (decimal.Decimal, error) {
	// Use the smaller of risk amount or max position size
	if riskAmount.GreaterThan(psc.config.AlertThresholds.MaxPositionSize) {
		return psc.config.AlertThresholds.MaxPositionSize, nil
	}
	return riskAmount, nil
}

// calculatePercentageSize calculates position size as percentage of portfolio
func (psc *PositionSizingCalculator) calculatePercentageSize(portfolio *Portfolio, riskAmount decimal.Decimal) (decimal.Decimal, error) {
	// Calculate position size as percentage of portfolio value
	positionSize := portfolio.TotalValue.Mul(psc.config.DefaultPositionSize)
	
	// Ensure it doesn't exceed max position size
	if positionSize.GreaterThan(psc.config.AlertThresholds.MaxPositionSize) {
		positionSize = psc.config.AlertThresholds.MaxPositionSize
	}
	
	// Ensure it doesn't exceed available cash
	if positionSize.GreaterThan(portfolio.CashBalance) {
		positionSize = portfolio.CashBalance
	}
	
	return positionSize, nil
}

// calculateKellySize calculates position size using Kelly Criterion
func (psc *PositionSizingCalculator) calculateKellySize(
	portfolio *Portfolio,
	symbol, exchange string,
	entryPrice, stopLossPrice decimal.Decimal,
) (decimal.Decimal, error) {
	
	// Kelly Criterion: f = (bp - q) / b
	// where:
	// f = fraction of capital to bet
	// b = odds received on the wager (entry price / stop loss)
	// p = probability of winning
	// q = probability of losing (1 - p)
	
	// For simplicity, assume 50% win probability
	// In practice, this would be calculated from historical data
	p := decimal.NewFromFloat(0.5)
	q := decimal.NewFromFloat(0.5)
	
	// Calculate odds (entry price / stop loss)
	odds := entryPrice.Div(stopLossPrice)
	
	// Kelly fraction
	kellyFraction := odds.Mul(p).Sub(q).Div(odds)
	
	// Ensure Kelly fraction is positive and reasonable
	if kellyFraction.LessThanOrEqual(decimal.Zero) {
		kellyFraction = decimal.NewFromFloat(0.01) // 1% minimum
	}
	
	if kellyFraction.GreaterThan(decimal.NewFromFloat(0.25)) {
		kellyFraction = decimal.NewFromFloat(0.25) // 25% maximum
	}
	
	// Calculate position size
	positionSize := portfolio.TotalValue.Mul(kellyFraction)
	
	// Apply limits
	if positionSize.GreaterThan(psc.config.AlertThresholds.MaxPositionSize) {
		positionSize = psc.config.AlertThresholds.MaxPositionSize
	}
	
	if positionSize.GreaterThan(portfolio.CashBalance) {
		positionSize = portfolio.CashBalance
	}
	
	return positionSize, nil
}

// CalculateStopLoss calculates stop loss price based on risk percentage
func (psc *PositionSizingCalculator) CalculateStopLoss(entryPrice decimal.Decimal, side string) decimal.Decimal {
	stopLossPercentage := psc.config.AlertThresholds.StopLossPercentage
	
	if side == "LONG" {
		return entryPrice.Mul(decimal.NewFromFloat(1).Sub(stopLossPercentage))
	} else {
		return entryPrice.Mul(decimal.NewFromFloat(1).Add(stopLossPercentage))
	}
}

// CalculateTakeProfit calculates take profit price based on profit percentage
func (psc *PositionSizingCalculator) CalculateTakeProfit(entryPrice decimal.Decimal, side string) decimal.Decimal {
	takeProfitPercentage := psc.config.AlertThresholds.TakeProfitPercentage
	
	if side == "LONG" {
		return entryPrice.Mul(decimal.NewFromFloat(1).Add(takeProfitPercentage))
	} else {
		return entryPrice.Mul(decimal.NewFromFloat(1).Sub(takeProfitPercentage))
	}
}

// CalculateRiskRewardRatio calculates the risk-reward ratio for a position
func (psc *PositionSizingCalculator) CalculateRiskRewardRatio(
	entryPrice, stopLossPrice, takeProfitPrice decimal.Decimal,
) decimal.Decimal {
	
	risk := entryPrice.Sub(stopLossPrice).Abs()
	reward := takeProfitPrice.Sub(entryPrice).Abs()
	
	if risk.IsZero() {
		return decimal.Zero
	}
	
	return reward.Div(risk)
}

// CalculatePositionValue calculates the total value of a position
func (psc *PositionSizingCalculator) CalculatePositionValue(
	quantity, price decimal.Decimal,
) decimal.Decimal {
	return quantity.Mul(price)
}

// CalculateUnrealizedPNL calculates unrealized P&L for a position
func (psc *PositionSizingCalculator) CalculateUnrealizedPNL(
	quantity, entryPrice, currentPrice decimal.Decimal,
	side string,
) decimal.Decimal {
	
	if side == "LONG" {
		return quantity.Mul(currentPrice.Sub(entryPrice))
	} else {
		return quantity.Mul(entryPrice.Sub(currentPrice))
	}
}

// CalculatePortfolioMetrics calculates various portfolio risk metrics
func (psc *PositionSizingCalculator) CalculatePortfolioMetrics(portfolio *Portfolio) *PortfolioMetrics {
	metrics := &PortfolioMetrics{
		TotalValue:    portfolio.TotalValue,
		CashBalance:   portfolio.CashBalance,
		InvestedValue: portfolio.InvestedValue,
		UnrealizedPNL: portfolio.UnrealizedPNL,
		RealizedPNL:   portfolio.RealizedPNL,
		DailyPNL:      portfolio.DailyPNL,
	}
	
	// Calculate leverage
	if portfolio.CashBalance.GreaterThan(decimal.Zero) {
		metrics.Leverage = portfolio.InvestedValue.Div(portfolio.CashBalance)
	}
	
	// Calculate concentration risk
	maxPositionValue := decimal.Zero
	for _, position := range portfolio.Positions {
		if position.MarketValue.GreaterThan(maxPositionValue) {
			maxPositionValue = position.MarketValue
		}
	}
	
	if portfolio.TotalValue.GreaterThan(decimal.Zero) {
		metrics.MaxConcentration = maxPositionValue.Div(portfolio.TotalValue)
	}
	
	// Calculate diversification score (number of positions)
	metrics.DiversificationScore = decimal.NewFromInt(int64(len(portfolio.Positions)))
	
	return metrics
}

// PortfolioMetrics represents calculated portfolio metrics
type PortfolioMetrics struct {
	TotalValue           decimal.Decimal `json:"total_value"`
	CashBalance          decimal.Decimal `json:"cash_balance"`
	InvestedValue        decimal.Decimal `json:"invested_value"`
	UnrealizedPNL        decimal.Decimal `json:"unrealized_pnl"`
	RealizedPNL          decimal.Decimal `json:"realized_pnl"`
	DailyPNL             decimal.Decimal `json:"daily_pnl"`
	Leverage             decimal.Decimal `json:"leverage"`
	MaxConcentration     decimal.Decimal `json:"max_concentration"`
	DiversificationScore decimal.Decimal `json:"diversification_score"`
}

// CalculateVaR calculates Value at Risk using historical simulation
func (psc *PositionSizingCalculator) CalculateVaR(
	portfolio *Portfolio,
	confidenceLevel float64, // 0.95 for 95% VaR, 0.99 for 99% VaR
) decimal.Decimal {
	
	// This is a simplified VaR calculation
	// In practice, you would use historical returns or Monte Carlo simulation
	
	// For now, use a simple approximation based on portfolio volatility
	// Assume 2% daily volatility for simplicity
	dailyVolatility := decimal.NewFromFloat(0.02)
	
	// Calculate VaR using normal distribution approximation
	// VaR = portfolio_value * volatility * z_score
	zScore := psc.getZScore(confidenceLevel)
	portfolioValue := portfolio.TotalValue
	
	vaR := portfolioValue.Mul(dailyVolatility).Mul(zScore)
	
	return vaR.Abs() // VaR is always positive
}

// getZScore returns the z-score for a given confidence level
func (psc *PositionSizingCalculator) getZScore(confidenceLevel float64) decimal.Decimal {
	// Common z-scores for confidence levels
	switch confidenceLevel {
	case 0.90:
		return decimal.NewFromFloat(1.28)
	case 0.95:
		return decimal.NewFromFloat(1.65)
	case 0.99:
		return decimal.NewFromFloat(2.33)
	case 0.999:
		return decimal.NewFromFloat(3.09)
	default:
		// Use approximation for other confidence levels
		zScore := math.Sqrt(2) * math.Erfinv(2*confidenceLevel-1)
		return decimal.NewFromFloat(zScore)
	}
}

// CalculateSharpeRatio calculates the Sharpe ratio for the portfolio
func (psc *PositionSizingCalculator) CalculateSharpeRatio(
	portfolio *Portfolio,
	riskFreeRate decimal.Decimal,
	volatility decimal.Decimal,
) decimal.Decimal {
	
	if volatility.IsZero() {
		return decimal.Zero
	}
	
	// Sharpe ratio = (portfolio_return - risk_free_rate) / volatility
	// For simplicity, use daily P&L as return
	excessReturn := portfolio.DailyPNL.Div(portfolio.TotalValue).Sub(riskFreeRate)
	
	return excessReturn.Div(volatility)
}

// CalculateMaxDrawdown calculates the maximum drawdown from peak
func (psc *PositionSizingCalculator) CalculateMaxDrawdown(
	portfolio *Portfolio,
	peakValue decimal.Decimal,
) decimal.Decimal {
	
	if peakValue.IsZero() {
		return decimal.Zero
	}
	
	drawdown := peakValue.Sub(portfolio.TotalValue).Div(peakValue)
	
	if drawdown.LessThan(decimal.Zero) {
		return decimal.Zero
	}
	
	return drawdown
}
