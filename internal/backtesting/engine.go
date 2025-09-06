package backtesting

import (
	"context"
	"fmt"
	"log"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"velocimex/internal/normalizer"
	"velocimex/internal/orderbook"
	"velocimex/internal/orders"
	"velocimex/internal/risk"
	"velocimex/internal/strategy"
)

// Engine implements the BacktestEngine interface
type Engine struct {
	config           BacktestConfig
	historicalData   map[string]map[string]*HistoricalData // symbol -> exchange -> data
	strategies       map[string]strategy.Strategy
	orderManager     orders.OrderManager
	riskManager      risk.RiskManager
	orderBookManager *orderbook.Manager
	normalizer       *normalizer.Normalizer
	
	// State
	running          bool
	paused           bool
	currentTime      time.Time
	portfolioHistory []*PortfolioSnapshot
	trades           []*BacktestTrade
	riskEvents       []*risk.RiskEvent
	
	// Synchronization
	mu               sync.RWMutex
	ctx              context.Context
	cancel           context.CancelFunc
	
	// Metrics
	totalCommission  decimal.Decimal
	totalSlippage    decimal.Decimal
	executionTimes   []time.Duration
}

// NewEngine creates a new backtesting engine
func NewEngine() *Engine {
	ctx, cancel := context.WithCancel(context.Background())
	return &Engine{
		historicalData:   make(map[string]map[string]*HistoricalData),
		strategies:       make(map[string]strategy.Strategy),
		orderBookManager: orderbook.NewManager(),
		normalizer:       normalizer.New(),
		portfolioHistory: make([]*PortfolioSnapshot, 0),
		trades:           make([]*BacktestTrade, 0),
		riskEvents:       make([]*risk.RiskEvent, 0),
		ctx:              ctx,
		cancel:           cancel,
	}
}

// SetConfig sets the backtesting configuration
func (e *Engine) SetConfig(config BacktestConfig) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	
	e.config = config
	
	// Initialize order manager with backtesting config
	smartRouter := orders.NewSmartRouter(orders.DefaultSmartRouterConfig(), e.orderBookManager)
	e.orderManager = orders.NewManager(orders.DefaultManagerConfig(), smartRouter, nil)
	
	// Initialize risk manager if enabled
	if config.RiskManagement {
		e.riskManager = risk.NewManager(config.RiskConfig, nil)
		if err := e.riskManager.Start(); err != nil {
			return fmt.Errorf("failed to start risk manager: %v", err)
		}
	}
	
	return nil
}

// GetConfig returns the current configuration
func (e *Engine) GetConfig() BacktestConfig {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.config
}

// LoadHistoricalData loads historical data for a symbol and exchange
func (e *Engine) LoadHistoricalData(symbol, exchange string, startDate, endDate time.Time) (*HistoricalData, error) {
	// In a real implementation, this would load data from a database or file
	// For now, we'll generate synthetic data
	data := e.generateSyntheticData(symbol, exchange, startDate, endDate)
	
	e.mu.Lock()
	defer e.mu.Unlock()
	
	if e.historicalData[symbol] == nil {
		e.historicalData[symbol] = make(map[string]*HistoricalData)
	}
	
	e.historicalData[symbol][exchange] = data
	return data, nil
}

// AddHistoricalData adds historical data to the engine
func (e *Engine) AddHistoricalData(data *HistoricalData) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	
	if e.historicalData[data.Symbol] == nil {
		e.historicalData[data.Symbol] = make(map[string]*HistoricalData)
	}
	
	e.historicalData[data.Symbol][data.Exchange] = data
	return nil
}

// GetAvailableData returns available historical data
func (e *Engine) GetAvailableData() map[string][]string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	
	result := make(map[string][]string)
	for symbol, exchanges := range e.historicalData {
		exchangeList := make([]string, 0, len(exchanges))
		for exchange := range exchanges {
			exchangeList = append(exchangeList, exchange)
		}
		result[symbol] = exchangeList
	}
	
	return result
}

// RegisterStrategy registers a strategy for backtesting
func (e *Engine) RegisterStrategy(strategy strategy.Strategy) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	
	e.strategies[strategy.GetID()] = strategy
	return nil
}

// GetRegisteredStrategies returns all registered strategies
func (e *Engine) GetRegisteredStrategies() []strategy.Strategy {
	e.mu.RLock()
	defer e.mu.RUnlock()
	
	strategies := make([]strategy.Strategy, 0, len(e.strategies))
	for _, s := range e.strategies {
		strategies = append(strategies, s)
	}
	
	return strategies
}

// RunBacktest runs a backtest with all registered strategies
func (e *Engine) RunBacktest() (*BacktestResult, error) {
	if len(e.strategies) == 0 {
		return nil, fmt.Errorf("no strategies registered")
	}
	
	// Run backtest for each strategy and combine results
	var combinedResult *BacktestResult
	
	for strategyID := range e.strategies {
		result, err := e.RunBacktestWithStrategy(strategyID)
		if err != nil {
			return nil, fmt.Errorf("failed to run backtest for strategy %s: %v", strategyID, err)
		}
		
		if combinedResult == nil {
			combinedResult = result
		} else {
			// Combine results (simplified - in practice you'd want more sophisticated combination)
			combinedResult.TotalTrades += result.TotalTrades
			combinedResult.WinningTrades += result.WinningTrades
			combinedResult.LosingTrades += result.LosingTrades
			combinedResult.TotalCommission = combinedResult.TotalCommission.Add(result.TotalCommission)
			combinedResult.TotalSlippage = combinedResult.TotalSlippage.Add(result.TotalSlippage)
		}
	}
	
	return combinedResult, nil
}

// RunBacktestWithStrategy runs a backtest for a specific strategy
func (e *Engine) RunBacktestWithStrategy(strategyID string) (*BacktestResult, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	
	strategy, exists := e.strategies[strategyID]
	if !exists {
		return nil, fmt.Errorf("strategy not found: %s", strategyID)
	}
	
	// Initialize backtest state
	e.running = true
	e.paused = false
	e.currentTime = e.config.StartDate
	e.portfolioHistory = make([]*PortfolioSnapshot, 0)
	e.trades = make([]*BacktestTrade, 0)
	e.riskEvents = make([]*risk.RiskEvent, 0)
	e.totalCommission = decimal.Zero
	e.totalSlippage = decimal.Zero
	e.executionTimes = make([]time.Duration, 0)
	
	// Initialize portfolio
	portfolio := &risk.Portfolio{
		TotalValue:    e.config.InitialCapital,
		CashBalance:   e.config.InitialCapital,
		InvestedValue: decimal.Zero,
		UnrealizedPNL: decimal.Zero,
		RealizedPNL:   decimal.Zero,
		DailyPNL:      decimal.Zero,
		Positions:     make(map[string]*risk.Position),
		LastUpdated:   e.currentTime,
	}
	
	if e.riskManager != nil {
		e.riskManager.UpdatePortfolio(portfolio)
	}
	
	startTime := time.Now()
	log.Printf("Starting backtest for strategy %s from %s to %s", strategyID, e.config.StartDate, e.config.EndDate)
	
	// Run the backtest
	err := e.runBacktestLoop(strategy)
	
	endTime := time.Now()
	duration := endTime.Sub(startTime)
	
	if err != nil {
		return nil, fmt.Errorf("backtest failed: %v", err)
	}
	
	// Calculate final results
	result := e.calculateBacktestResult(strategyID, duration)
	
	log.Printf("Backtest completed in %v", duration)
	return result, nil
}

// runBacktestLoop runs the main backtesting loop
func (e *Engine) runBacktestLoop(strategy strategy.Strategy) error {
	for e.currentTime.Before(e.config.EndDate) && e.running {
		if e.paused {
			time.Sleep(100 * time.Millisecond)
			continue
		}
		
		// Update market data for current time
		if err := e.updateMarketData(); err != nil {
			log.Printf("Error updating market data: %v", err)
		}
		
		// Run strategy
		if err := e.runStrategy(strategy); err != nil {
			log.Printf("Error running strategy: %v", err)
		}
		
		// Update portfolio and risk metrics
		if err := e.updatePortfolio(); err != nil {
			log.Printf("Error updating portfolio: %v", err)
		}
		
		// Take portfolio snapshot
		e.takePortfolioSnapshot()
		
		// Advance time
		e.currentTime = e.currentTime.Add(e.config.DataFrequency)
		
		// Simulate latency
		if e.config.Latency > 0 {
			time.Sleep(e.config.Latency)
		}
	}
	
	return nil
}

// updateMarketData updates market data for the current time
func (e *Engine) updateMarketData() error {
	for symbol, exchanges := range e.historicalData {
		for exchange, data := range exchanges {
			// Find data point for current time
			dataPoint := e.findDataPointForTime(data, e.currentTime)
			if dataPoint == nil {
				continue
			}
			
			// Create normalized price levels
			bids := []normalizer.PriceLevel{
				{Price: dataPoint.Bid.InexactFloat64(), Volume: dataPoint.BidSize.InexactFloat64()},
			}
			asks := []normalizer.PriceLevel{
				{Price: dataPoint.Ask.InexactFloat64(), Volume: dataPoint.AskSize.InexactFloat64()},
			}
			
			// Update order book
			e.orderBookManager.UpdateOrderBook(exchange, symbol, bids, asks)
		}
	}
	
	return nil
}

// findDataPointForTime finds the data point closest to the given time
func (e *Engine) findDataPointForTime(data *HistoricalData, targetTime time.Time) *DataPoint {
	var closest *DataPoint
	var minDiff time.Duration
	
	for _, point := range data.DataPoints {
		diff := point.Timestamp.Sub(targetTime)
		if diff < 0 {
			diff = -diff
		}
		
		if closest == nil || diff < minDiff {
			closest = point
			minDiff = diff
		}
	}
	
	return closest
}

// runStrategy runs the strategy for the current time
func (e *Engine) runStrategy(strategy strategy.Strategy) error {
	// Get current order books
	orderBooks := make(map[string]*orderbook.OrderBook)
	for symbol := range e.historicalData {
		for exchange := range e.historicalData[symbol] {
			key := fmt.Sprintf("%s:%s", exchange, symbol)
			if book := e.orderBookManager.GetOrderBook(symbol); book != nil {
				orderBooks[key] = book
			}
		}
	}
	
	// Run strategy
	signals, err := strategy.GenerateSignals(orderBooks)
	if err != nil {
		return err
	}
	
	// Execute signals
	for _, signal := range signals {
		if err := e.executeSignal(signal, strategy); err != nil {
			log.Printf("Error executing signal: %v", err)
		}
	}
	
	return nil
}

// executeSignal executes a trading signal
func (e *Engine) executeSignal(signal *strategy.Signal, strategy strategy.Strategy) error {
	// Create order request
	orderReq := &orders.OrderRequest{
		Symbol:       signal.Symbol,
		Exchange:     signal.Exchange,
		Side:         orders.OrderSide(signal.Side),
		Quantity:     signal.Quantity,
		Price:        signal.Price,
		TimeInForce:  orders.TimeInForceGTC,
		StrategyID:   strategy.GetID(),
		StrategyName: strategy.GetName(),
		Metadata:     signal.Metadata,
	}
	
	// Apply slippage
	if e.config.Slippage.GreaterThan(decimal.Zero) {
		slippageAmount := signal.Price.Mul(e.config.Slippage)
		if signal.Side == "BUY" {
			orderReq.Price = orderReq.Price.Add(slippageAmount)
		} else {
			orderReq.Price = orderReq.Price.Sub(slippageAmount)
		}
		e.totalSlippage = e.totalSlippage.Add(slippageAmount.Mul(signal.Quantity))
	}
	
	// Simulate execution time
	executionStart := time.Now()
	
	// Submit order
	_, err := e.orderManager.SubmitOrder(e.ctx, orderReq)
	if err != nil {
		return err
	}
	
	executionTime := time.Since(executionStart)
	e.executionTimes = append(e.executionTimes, executionTime)
	
	// Calculate commission
	commission := signal.Price.Mul(signal.Quantity).Mul(e.config.Commission)
	e.totalCommission = e.totalCommission.Add(commission)
	
	// Create backtest trade
	trade := &BacktestTrade{
		ID:           uuid.New().String(),
		Symbol:       signal.Symbol,
		Exchange:     signal.Exchange,
		Side:         signal.Side,
		Quantity:     signal.Quantity,
		EntryPrice:   signal.Price,
		ExitPrice:    decimal.Zero, // Will be set when position is closed
		EntryTime:    e.currentTime,
		ExitTime:     time.Time{}, // Will be set when position is closed
		Duration:     0,            // Will be calculated when position is closed
		PnL:         decimal.Zero, // Will be calculated when position is closed
		PnLPct:      decimal.Zero, // Will be calculated when position is closed
		Commission:  commission,
		Slippage:    signal.Price.Mul(signal.Quantity).Mul(e.config.Slippage),
		StrategyID:  strategy.GetID(),
		StrategyName: strategy.GetName(),
		Metadata:    signal.Metadata,
	}
	
	e.trades = append(e.trades, trade)
	
	return nil
}

// updatePortfolio updates the portfolio based on current positions
func (e *Engine) updatePortfolio() error {
	if e.riskManager == nil {
		return nil
	}
	
	// Get current portfolio
	portfolio := e.riskManager.GetPortfolio()
	
	// Update positions with current prices
	for _, position := range portfolio.Positions {
		// Find current price for position
		if data := e.historicalData[position.Symbol]; data != nil {
			if exchangeData := data[position.Exchange]; exchangeData != nil {
				dataPoint := e.findDataPointForTime(exchangeData, e.currentTime)
				if dataPoint != nil {
					e.riskManager.UpdatePosition(position.Symbol, position.Exchange, dataPoint.Close)
				}
			}
		}
	}
	
	return nil
}

// takePortfolioSnapshot takes a snapshot of the current portfolio
func (e *Engine) takePortfolioSnapshot() {
	if e.riskManager == nil {
		return
	}
	
	portfolio := e.riskManager.GetPortfolio()
	riskMetrics := e.riskManager.GetRiskMetrics()
	
	snapshot := &PortfolioSnapshot{
		Timestamp:     e.currentTime,
		TotalValue:    portfolio.TotalValue,
		CashBalance:   portfolio.CashBalance,
		InvestedValue: portfolio.InvestedValue,
		UnrealizedPNL: portfolio.UnrealizedPNL,
		RealizedPNL:   portfolio.RealizedPNL,
		DailyPNL:      portfolio.DailyPNL,
		Positions:     portfolio.Positions,
		RiskMetrics:   riskMetrics,
	}
	
	e.portfolioHistory = append(e.portfolioHistory, snapshot)
}

// calculateBacktestResult calculates the final backtest results
func (e *Engine) calculateBacktestResult(strategyID string, duration time.Duration) *BacktestResult {
	portfolio := e.riskManager.GetPortfolio()
	
	// Calculate basic metrics
	totalReturn := portfolio.TotalValue.Sub(e.config.InitialCapital)
	totalReturnPct := totalReturn.Div(e.config.InitialCapital).Mul(decimal.NewFromFloat(100))
	
	// Calculate trade metrics
	winningTrades := 0
	losingTrades := 0
	var totalPnL decimal.Decimal
	
	for _, trade := range e.trades {
		if !trade.PnL.IsZero() {
			totalPnL = totalPnL.Add(trade.PnL)
			if trade.PnL.GreaterThan(decimal.Zero) {
				winningTrades++
			} else {
				losingTrades++
			}
		}
	}
	
	winRate := decimal.Zero
	if len(e.trades) > 0 {
		winRate = decimal.NewFromInt(int64(winningTrades)).Div(decimal.NewFromInt(int64(len(e.trades))))
	}
	
	// Calculate average execution time
	avgExecutionTime := time.Duration(0)
	if len(e.executionTimes) > 0 {
		totalTime := time.Duration(0)
		for _, execTime := range e.executionTimes {
			totalTime += execTime
		}
		avgExecutionTime = totalTime / time.Duration(len(e.executionTimes))
	}
	
	// Calculate performance metrics (simplified)
	sharpeRatio := decimal.Zero
	if len(e.portfolioHistory) > 1 {
		// Calculate daily returns
		var returns []decimal.Decimal
		for i := 1; i < len(e.portfolioHistory); i++ {
			prevValue := e.portfolioHistory[i-1].TotalValue
			currValue := e.portfolioHistory[i].TotalValue
			if !prevValue.IsZero() {
				dailyReturn := currValue.Sub(prevValue).Div(prevValue)
				returns = append(returns, dailyReturn)
			}
		}
		
		// Calculate Sharpe ratio (simplified)
		if len(returns) > 0 {
			var sum decimal.Decimal
			for _, ret := range returns {
				sum = sum.Add(ret)
			}
			avgReturn := sum.Div(decimal.NewFromInt(int64(len(returns))))
			
			// Calculate standard deviation
			var variance decimal.Decimal
			for _, ret := range returns {
				diff := ret.Sub(avgReturn)
				variance = variance.Add(diff.Mul(diff))
			}
			stdDev := variance.Div(decimal.NewFromInt(int64(len(returns))))
			// Note: Sqrt() method doesn't exist in decimal package, using approximation
			stdDevFloat := stdDev.InexactFloat64()
			if stdDevFloat > 0 {
				stdDevFloat = math.Sqrt(stdDevFloat)
			}
			stdDev = decimal.NewFromFloat(stdDevFloat)
			
			if !stdDev.IsZero() {
				sharpeRatio = avgReturn.Div(stdDev)
			}
		}
	}
	
	return &BacktestResult{
		Config:           e.config,
		StartTime:        e.config.StartDate,
		EndTime:          e.config.EndDate,
		Duration:         duration,
		InitialCapital:   e.config.InitialCapital,
		FinalCapital:     portfolio.TotalValue,
		TotalReturn:      totalReturn,
		TotalReturnPct:   totalReturnPct,
		TotalTrades:      len(e.trades),
		WinningTrades:    winningTrades,
		LosingTrades:     losingTrades,
		WinRate:          winRate,
		SharpeRatio:      sharpeRatio,
		SortinoRatio:     decimal.Zero, // TODO: Implement
		CalmarRatio:      decimal.Zero, // TODO: Implement
		MaxDrawdown:      decimal.Zero, // TODO: Implement
		MaxDrawdownPct:   decimal.Zero, // TODO: Implement
		Volatility:       decimal.Zero, // TODO: Implement
		VaR95:            decimal.Zero, // TODO: Implement
		VaR99:            decimal.Zero, // TODO: Implement
		Beta:             decimal.Zero, // TODO: Implement
		Alpha:            decimal.Zero, // TODO: Implement
		TotalCommission:  e.totalCommission,
		TotalSlippage:    e.totalSlippage,
		AvgExecutionTime: avgExecutionTime,
		Trades:           e.trades,
		PortfolioHistory: e.portfolioHistory,
		RiskEvents:       e.riskEvents,
		StrategyMetrics:  make(map[string]interface{}),
	}
}

// generateSyntheticData generates synthetic historical data for testing
func (e *Engine) generateSyntheticData(symbol, exchange string, startDate, endDate time.Time) *HistoricalData {
	data := &HistoricalData{
		Symbol:     symbol,
		Exchange:   exchange,
		DataPoints: make([]*DataPoint, 0),
		StartTime:  startDate,
		EndTime:    endDate,
		Frequency:  e.config.DataFrequency,
		Metadata:   make(map[string]interface{}),
	}
	
	// Generate synthetic price data
	basePrice := decimal.NewFromFloat(50000) // Starting price
	currentPrice := basePrice
	currentTime := startDate
	
	for currentTime.Before(endDate) {
		// Generate random price movement
		change := decimal.NewFromFloat(rand.Float64()*0.02 - 0.01) // ±1% change
		currentPrice = currentPrice.Mul(decimal.NewFromFloat(1).Add(change))
		
		// Generate OHLC data
		open := currentPrice
		high := currentPrice.Mul(decimal.NewFromFloat(1.001))
		low := currentPrice.Mul(decimal.NewFromFloat(0.999))
		close := currentPrice
		
		// Generate bid/ask spread
		spread := currentPrice.Mul(decimal.NewFromFloat(0.0001)) // 0.01% spread
		bid := currentPrice.Sub(spread.Div(decimal.NewFromFloat(2)))
		ask := currentPrice.Add(spread.Div(decimal.NewFromFloat(2)))
		
		dataPoint := &DataPoint{
			Timestamp: currentTime,
			Open:      open,
			High:      high,
			Low:       low,
			Close:     close,
			Volume:    decimal.NewFromFloat(rand.Float64() * 1000),
			Bid:       bid,
			Ask:       ask,
			BidSize:   decimal.NewFromFloat(rand.Float64() * 100),
			AskSize:   decimal.NewFromFloat(rand.Float64() * 100),
			Metadata:  make(map[string]interface{}),
		}
		
		data.DataPoints = append(data.DataPoints, dataPoint)
		currentTime = currentTime.Add(e.config.DataFrequency)
	}
	
	return data
}

// AnalyzeResult analyzes backtest results
func (e *Engine) AnalyzeResult(result *BacktestResult) (*BacktestAnalysis, error) {
	// TODO: Implement comprehensive analysis
	return &BacktestAnalysis{
		Result: result,
		// Add analysis components
	}, nil
}

// GenerateReport generates a comprehensive backtest report
func (e *Engine) GenerateReport(result *BacktestResult) (*BacktestReport, error) {
	analysis, err := e.AnalyzeResult(result)
	if err != nil {
		return nil, err
	}
	
	summary := &BacktestSummary{
		Period:           fmt.Sprintf("%s to %s", result.StartTime.Format("2006-01-02"), result.EndTime.Format("2006-01-02")),
		InitialCapital:   result.InitialCapital,
		FinalCapital:     result.FinalCapital,
		TotalReturn:      result.TotalReturn,
		TotalReturnPct:   result.TotalReturnPct,
		AnnualizedReturn: decimal.Zero, // TODO: Calculate
		MaxDrawdown:      result.MaxDrawdown,
		SharpeRatio:      result.SharpeRatio,
		TotalTrades:      result.TotalTrades,
		WinRate:          result.WinRate,
		ProfitFactor:     decimal.Zero, // TODO: Calculate
		RiskAdjustedReturn: decimal.Zero, // TODO: Calculate
	}
	
	return &BacktestReport{
		Summary:         summary,
		Analysis:        analysis,
		Charts:          make(map[string]interface{}),
		Recommendations: make([]string, 0),
		GeneratedAt:     time.Now(),
		ReportVersion:   "1.0.0",
	}, nil
}

// Start starts the backtesting engine
func (e *Engine) Start() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	
	if e.running {
		return fmt.Errorf("backtesting engine already running")
	}
	
	e.running = true
	log.Println("Backtesting engine started")
	return nil
}

// Stop stops the backtesting engine
func (e *Engine) Stop() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	
	if !e.running {
		return nil
	}
	
	e.running = false
	e.cancel()
	
	if e.riskManager != nil {
		e.riskManager.Stop()
	}
	
	log.Println("Backtesting engine stopped")
	return nil
}

// IsRunning returns whether the engine is running
func (e *Engine) IsRunning() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.running
}

// Pause pauses the backtesting engine
func (e *Engine) Pause() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	
	e.paused = true
	return nil
}

// Resume resumes the backtesting engine
func (e *Engine) Resume() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	
	e.paused = false
	return nil
}
