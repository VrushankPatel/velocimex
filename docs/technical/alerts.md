# Comprehensive Alert System

Velocimex implements a sophisticated alert system designed for high-frequency trading environments where immediate notification of critical events is essential for risk management and operational efficiency.

## Overview

The alert system provides:
- **Multi-channel Notifications**: Slack, Email, Webhook, Teams, Discord, SMS
- **Market Event Alerts**: Price, volume, volatility, and arbitrage monitoring
- **Strategy Signal Alerts**: Trading signal and performance monitoring
- **Rule-based Processing**: Flexible condition and action definitions
- **Template System**: Reusable alert message templates
- **Real-time Processing**: Asynchronous event processing with worker pools
- **Comprehensive Monitoring**: Alert metrics and performance tracking

## Architecture

### Core Components

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│  Alert Engine   │    │ Market Events   │    │ Strategy Signals│
│                 │    │                 │    │                 │
│ - Rule Engine   │    │ - Price Alerts  │    │ - Signal Alerts │
│ - Processors    │    │ - Volume Alerts │    │ - Performance   │
│ - Templates     │    │ - Volatility    │    │ - Risk Alerts   │
│ - Metrics       │    │ - Arbitrage     │    │ - Drawdown      │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         └───────────────────────┼───────────────────────┘
                                 │
                    ┌─────────────────┐
                    │ Alert Channels  │
                    │                 │
                    │ - Slack         │
                    │ - Email         │
                    │ - Webhook       │
                    │ - Teams         │
                    │ - Discord       │
                    │ - SMS           │
                    └─────────────────┘
```

## Configuration

### Alert Engine Configuration

```yaml
alerts:
  engine:
    enabled: true
    max_workers: 4
    queue_size: 1000
    process_timeout: 30s
    retry_attempts: 3
    retry_delay: 5s
    cooldown_period: 1m
    enable_metrics: true
    enable_templates: true
    enable_scheduling: true
    cleanup_interval: 1h
    max_alert_age: 24h
```

### Channel Configuration

```yaml
alerts:
  channels:
    slack:
      type: slack
      enabled: true
      config:
        webhook_url: "https://hooks.slack.com/services/..."
        channel: "#alerts"
        username: "Velocimex"
      priority: 1
      timeout: 10s
      retry:
        enabled: true
        max_attempts: 3
        delay: 5s
        backoff: exponential
    
    email:
      type: email
      enabled: true
      config:
        smtp_host: "smtp.gmail.com"
        smtp_port: 587
        username: "alerts@velocimex.com"
        password: "password"
        from_email: "alerts@velocimex.com"
        to_emails: ["admin@velocimex.com"]
      priority: 2
      timeout: 30s
      retry:
        enabled: true
        max_attempts: 3
        delay: 10s
        backoff: linear
```

### Market Event Configuration

```yaml
alerts:
  market_events:
    enabled: true
    check_interval: 1s
    price_thresholds:
      btc_price_high:
        symbol: "BTCUSDT"
        exchange: "binance"
        threshold: 100000.0
        operator: "above"
        channels: ["slack", "email"]
        cooldown: 5m
      
      btc_price_low:
        symbol: "BTCUSDT"
        exchange: "binance"
        threshold: 20000.0
        operator: "below"
        channels: ["slack", "email"]
        cooldown: 5m
    
    volume_thresholds:
      btc_volume_high:
        symbol: "BTCUSDT"
        exchange: "binance"
        threshold: 1000.0
        operator: "above"
        channels: ["slack"]
        cooldown: 10m
    
    volatility_thresholds:
      btc_volatility_high:
        symbol: "BTCUSDT"
        exchange: "binance"
        threshold: 0.05
        operator: "above"
        channels: ["slack", "email"]
        cooldown: 15m
    
    arbitrage_thresholds:
      btc_arbitrage:
        symbol: "BTCUSDT"
        threshold: 0.01
        channels: ["slack"]
        cooldown: 1m
```

### Strategy Signal Configuration

```yaml
alerts:
  strategy_signals:
    enabled: true
    check_interval: 1s
    signal_thresholds:
      arbitrage_signal:
        strategy: "arbitrage"
        threshold: 0.8
        operator: "above"
        channels: ["slack"]
        cooldown: 1m
    
    performance_thresholds:
      arbitrage_performance:
        strategy: "arbitrage"
        metric: "profit"
        threshold: 1000.0
        operator: "above"
        channels: ["slack", "email"]
        cooldown: 1h
    
    risk_thresholds:
      arbitrage_risk:
        strategy: "arbitrage"
        risk_type: "drawdown"
        threshold: 0.1
        operator: "above"
        channels: ["slack", "email"]
        cooldown: 5m
```

## Usage

### Basic Alert Processing

```go
// Create alert engine
config := GetDefaultAlertConfig()
engine := NewAlertEngine(config, logger)

// Register channels
slackChannel := NewSlackChannel(webhookURL, "#alerts", "Velocimex")
engine.RegisterChannel("slack", slackChannel)

emailChannel := NewEmailChannel(smtpHost, smtpPort, username, password, fromEmail, toEmails)
engine.RegisterChannel("email", emailChannel)

// Process alert event
event := &AlertEvent{
    Type:      "market_price",
    Severity:  AlertSeverityHigh,
    Source:    "market_monitor",
    Message:   "Price alert triggered for BTCUSDT",
    Metadata: map[string]interface{}{
        "symbol":    "BTCUSDT",
        "exchange":  "binance",
        "price":     50000.0,
        "threshold": 100000.0,
    },
    Timestamp: time.Now(),
}

err := engine.ProcessEvent(event)
```

### Market Event Alerts

```go
// Create market event alert system
marketAlerts := NewMarketEventAlertSystem(engine, logger)

// Add price alert rule
rule := &MarketAlertRule{
    Symbol:    "BTCUSDT",
    Exchange:  "binance",
    Type:      MarketAlertPrice,
    Condition: MarketCondition{
        Operator: "above",
        Value:    100000.0,
    },
    Threshold: 100000.0,
    Channels:  []string{"slack", "email"},
    Enabled:   true,
}

err := marketAlerts.AddMarketRule(rule)

// Process market data
marketData := map[string]interface{}{
    "price":  105000.0,
    "volume": 100.0,
    "symbol": "BTCUSDT",
}

marketAlerts.ProcessMarketData("BTCUSDT", "binance", marketData)
```

### Strategy Signal Alerts

```go
// Create strategy signal alert system
strategyAlerts := NewStrategySignalAlertSystem(engine, logger)

// Add signal alert rule
rule := &StrategyAlertRule{
    Strategy: "arbitrage",
    Type:     StrategyAlertSignal,
    Condition: StrategyCondition{
        Operator: "above",
        Value:    0.8,
    },
    Threshold: 0.8,
    Channels:  []string{"slack"},
    Enabled:   true,
}

err := strategyAlerts.AddStrategyRule(rule)

// Process trading signal
signal := map[string]interface{}{
    "strategy":   "arbitrage",
    "symbol":     "BTCUSDT",
    "side":       "BUY",
    "price":      50000.0,
    "quantity":   1.0,
    "confidence": 0.9,
}

strategyAlerts.ProcessSignal(signal)
```

### Alert Templates

```go
// Add alert template
template := &AlertTemplate{
    ID:          "price_alert",
    Name:        "Price Alert",
    Description: "Template for price alerts",
    Subject:     "Price Alert: {{symbol}} on {{exchange}}",
    Body:        "Price alert triggered for {{symbol}} on {{exchange}}. Current price: {{price}}, Threshold: {{threshold}}",
    Channels:    []string{"slack", "email"},
    Variables:   []string{"symbol", "exchange", "price", "threshold"},
}

err := engine.AddTemplate(template)
```

## Alert Types

### Market Event Alerts

#### Price Alerts
- **Above Threshold**: Alert when price exceeds specified value
- **Below Threshold**: Alert when price falls below specified value
- **Crosses Above**: Alert when price crosses above threshold
- **Crosses Below**: Alert when price crosses below threshold

#### Volume Alerts
- **High Volume**: Alert when volume exceeds normal levels
- **Low Volume**: Alert when volume falls below normal levels
- **Volume Spike**: Alert when volume increases significantly

#### Volatility Alerts
- **High Volatility**: Alert when volatility exceeds threshold
- **Low Volatility**: Alert when volatility falls below threshold
- **Volatility Spike**: Alert when volatility increases suddenly

#### Arbitrage Alerts
- **Opportunity Detected**: Alert when arbitrage opportunity exceeds threshold
- **Profit Tracking**: Alert when arbitrage profit reaches target

### Strategy Signal Alerts

#### Signal Alerts
- **Confidence Level**: Alert based on signal confidence
- **Signal Type**: Alert for specific signal types
- **Symbol Filtering**: Alert for specific trading symbols

#### Performance Alerts
- **Profit Targets**: Alert when profit reaches target
- **Loss Limits**: Alert when losses exceed limit
- **Win Rate**: Alert based on win rate performance
- **Sharpe Ratio**: Alert based on risk-adjusted returns

#### Risk Alerts
- **Drawdown**: Alert when drawdown exceeds threshold
- **Position Size**: Alert when position size exceeds limit
- **Risk Limits**: Alert when risk metrics exceed limits

## Alert Channels

### Slack Integration

```go
slackChannel := NewSlackChannel(
    "https://hooks.slack.com/services/...",
    "#alerts",
    "Velocimex",
)

// Send alert
alert := &Alert{
    ID:       "alert_123",
    Type:     "market_price",
    Severity: AlertSeverityHigh,
    Title:    "Price Alert",
    Message:  "BTCUSDT price exceeded threshold",
    Channels: []string{"slack"},
    Metadata: map[string]interface{}{
        "symbol":    "BTCUSDT",
        "price":     105000.0,
        "threshold": 100000.0,
    },
}

err := slackChannel.Send(alert)
```

### Email Integration

```go
emailChannel := NewEmailChannel(
    "smtp.gmail.com",
    587,
    "alerts@velocimex.com",
    "password",
    "alerts@velocimex.com",
    []string{"admin@velocimex.com"},
)

err := emailChannel.Send(alert)
```

### Webhook Integration

```go
webhookChannel := NewWebhookChannel(
    "https://alerts.velocimex.com/webhook",
    "POST",
    map[string]string{
        "Authorization": "Bearer token",
        "Content-Type":  "application/json",
    },
)

err := webhookChannel.Send(alert)
```

### Teams Integration

```go
teamsChannel := NewTeamsChannel(
    "https://outlook.office.com/webhook/...",
)

err := teamsChannel.Send(alert)
```

### Discord Integration

```go
discordChannel := NewDiscordChannel(
    "https://discord.com/api/webhooks/...",
    "Velocimex",
)

err := discordChannel.Send(alert)
```

### SMS Integration

```go
smsChannel := NewSMSChannel(
    "https://api.twilio.com/2010-04-01/Accounts/.../Messages.json",
    "api_key",
    "+1234567890",
    []string{"+1234567890"},
)

err := smsChannel.Send(alert)
```

## Monitoring and Metrics

### Alert Metrics

```go
// Get alert metrics
metrics := engine.GetMetrics()

fmt.Printf("Total Rules: %d\n", metrics.TotalRules)
fmt.Printf("Active Rules: %d\n", metrics.ActiveRules)
fmt.Printf("Total Alerts: %d\n", metrics.TotalAlerts)
fmt.Printf("Processed Alerts: %d\n", metrics.ProcessedAlerts)
fmt.Printf("Failed Alerts: %d\n", metrics.FailedAlerts)
fmt.Printf("Queue Size: %d\n", metrics.QueueSize)
```

### Performance Monitoring

The alert system tracks various performance metrics:

- **Processing Time**: Time taken to process alerts
- **Queue Size**: Number of pending alerts
- **Success Rate**: Percentage of successfully processed alerts
- **Channel Performance**: Performance metrics per channel
- **Rule Performance**: Performance metrics per rule

## Best Practices

### Alert Design

1. **Clear Severity Levels**: Use appropriate severity levels
2. **Concise Messages**: Keep alert messages clear and actionable
3. **Relevant Metadata**: Include only necessary context information
4. **Proper Cooldowns**: Set appropriate cooldown periods
5. **Channel Selection**: Choose appropriate channels for different alert types

### Performance Optimization

1. **Asynchronous Processing**: Use worker pools for alert processing
2. **Queue Management**: Monitor queue size and processing capacity
3. **Channel Batching**: Batch alerts when possible
4. **Template Caching**: Cache frequently used templates
5. **Metrics Collection**: Monitor system performance

### Error Handling

1. **Retry Logic**: Implement retry mechanisms for failed alerts
2. **Fallback Channels**: Use multiple channels for critical alerts
3. **Error Logging**: Log all alert processing errors
4. **Graceful Degradation**: Continue processing when channels fail
5. **Monitoring**: Set up alerts for alert system failures

## Troubleshooting

### Common Issues

1. **High Queue Size**
   - Increase worker count
   - Optimize alert processing
   - Check channel performance

2. **Failed Alerts**
   - Check channel configuration
   - Verify network connectivity
   - Review retry settings

3. **Missing Alerts**
   - Check rule conditions
   - Verify event processing
   - Review cooldown settings

4. **Slow Processing**
   - Monitor worker performance
   - Check queue bottlenecks
   - Optimize alert templates

### Debugging

```go
// Enable debug logging
config := &AlertConfig{
    Enabled: true,
    // ... other config
}

// Check alert metrics
metrics := engine.GetMetrics()
if metrics.QueueSize > 1000 {
    // Queue is getting full
}

// Check channel status
for name, channel := range channels {
    if !channel.IsEnabled() {
        // Channel is disabled
    }
}
```

## Integration

### External Systems

- **Monitoring Tools**: Prometheus, Grafana integration
- **Incident Management**: PagerDuty, OpsGenie integration
- **Communication**: Slack, Teams, Discord integration
- **Notification**: Email, SMS, Push notification integration

### APIs

- **REST API**: Alert management and configuration
- **WebSocket**: Real-time alert streaming
- **GraphQL**: Advanced alert querying
- **gRPC**: High-performance alert processing

This comprehensive alert system ensures that Velocimex maintains full visibility and immediate notification of critical events required for high-frequency trading operations.
