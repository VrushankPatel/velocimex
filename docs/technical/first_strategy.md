# Strategy Development Guide

This guide will help you create your first trading strategy using Velocimex.

## Table of Contents
- [Strategy Basics](#strategy-basics)
- [Creating Your First Strategy](#creating-your-first-strategy)
- [Technical Indicators](#technical-indicators)
- [Backtesting](#backtesting)
- [Optimization](#optimization)
- [Live Trading](#live-trading)

## Strategy Basics

A trading strategy in Velocimex consists of several key components:

1. **Entry Signals**: Conditions that trigger buy orders
2. **Exit Signals**: Conditions that trigger sell orders
3. **Risk Management**: Stop-loss and take-profit levels
4. **Position Sizing**: How much to invest in each trade

## Creating Your First Strategy

Let's create a simple RSI-based strategy. Create a new file `strategies/my_first_strategy.py`:

```python
from velocimex.strategy import Strategy
from velocimex.indicators import RSI, MACD

class MyFirstStrategy(Strategy):
    """
    A simple RSI-based strategy that buys when RSI is oversold
    and sells when RSI is overbought.
    """
    
    def __init__(self):
        super().__init__()
        self.rsi = RSI(period=14)
        self.macd = MACD(fast=12, slow=26, signal=9)
        
    def generate_signals(self, data):
        """
        Generate trading signals based on technical indicators
        """
        # Calculate indicators
        rsi_values = self.rsi.calculate(data['close'])
        macd_line, signal_line, _ = self.macd.calculate(data['close'])
        
        # Initialize signals
        signals = pd.DataFrame(index=data.index)
        signals['signal'] = 0
        
        # Generate buy signals (RSI < 30 and MACD crossover)
        buy_condition = (rsi_values < 30) & (macd_line > signal_line)
        signals.loc[buy_condition, 'signal'] = 1
        
        # Generate sell signals (RSI > 70 and MACD crossunder)
        sell_condition = (rsi_values > 70) & (macd_line < signal_line)
        signals.loc[sell_condition, 'signal'] = -1
        
        return signals
```

## Technical Indicators

Velocimex supports various technical indicators:

### Trend Indicators
- Moving Averages (SMA, EMA)
- MACD
- ADX

### Momentum Indicators
- RSI
- Stochastic Oscillator
- CCI

### Volume Indicators
- OBV
- Volume Weighted Average Price (VWAP)

### Volatility Indicators
- Bollinger Bands
- ATR

Example of using multiple indicators:

```python
def generate_signals(self, data):
    # Calculate indicators
    sma20 = SMA(period=20).calculate(data['close'])
    sma50 = SMA(period=50).calculate(data['close'])
    rsi = RSI(period=14).calculate(data['close'])
    bb = BollingerBands(period=20).calculate(data['close'])
    
    # Generate signals
    signals = pd.DataFrame(index=data.index)
    signals['signal'] = 0
    
    # Buy when price is above both SMAs, RSI is oversold, and price is near lower BB
    buy_condition = (
        (data['close'] > sma20) & 
        (data['close'] > sma50) & 
        (rsi < 30) & 
        (data['close'] < bb['lower'])
    )
    
    signals.loc[buy_condition, 'signal'] = 1
    
    return signals
```

## Backtesting

Test your strategy using historical data:

```python
from velocimex.backtesting import Backtest

# Load historical data
data = load_historical_data('BTC/USDT', '1h', '2023-01-01', '2023-12-31')

# Initialize strategy
strategy = MyFirstStrategy()

# Run backtest
backtest = Backtest(strategy, data)
results = backtest.run()

# Analyze results
print(f"Total Return: {results['total_return']}%")
print(f"Sharpe Ratio: {results['sharpe_ratio']}")
print(f"Max Drawdown: {results['max_drawdown']}%")
```

## Optimization

Optimize your strategy parameters:

```python
from velocimex.optimization import GridSearch

# Define parameter ranges
param_grid = {
    'rsi_period': [10, 14, 20],
    'rsi_oversold': [25, 30, 35],
    'rsi_overbought': [65, 70, 75]
}

# Run optimization
optimizer = GridSearch(strategy_class=MyFirstStrategy, param_grid=param_grid)
best_params = optimizer.optimize(data)

print("Best Parameters:", best_params)
```

## Live Trading

Deploy your strategy for live trading:

```python
from velocimex.trading import LiveTrader

# Initialize live trader
trader = LiveTrader(
    strategy=MyFirstStrategy(),
    exchange='binance',
    pair='BTC/USDT',
    timeframe='1h'
)

# Start trading
trader.start()
```

## Best Practices

1. **Start Simple**
   - Begin with basic indicators
   - Test thoroughly before adding complexity

2. **Risk Management**
   - Always implement stop-loss
   - Use proper position sizing
   - Consider maximum drawdown

3. **Testing**
   - Backtest on multiple timeframes
   - Test on different market conditions
   - Validate with walk-forward analysis

4. **Documentation**
   - Document your strategy logic
   - Keep track of parameter changes
   - Record performance metrics

## Common Pitfalls

1. **Overfitting**
   - Avoid optimizing too many parameters
   - Use out-of-sample testing
   - Consider market regime changes

2. **Transaction Costs**
   - Account for fees in backtesting
   - Consider slippage
   - Factor in exchange minimums

3. **Market Conditions**
   - Test in different market regimes
   - Consider liquidity constraints
   - Account for market impact

For more advanced strategy development, refer to the [Technical Documentation](../technical/index.md). 