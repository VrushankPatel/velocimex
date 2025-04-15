# Configuration Guide

This guide will help you set up and configure Velocimex for your trading needs.

## Table of Contents
- [Basic Configuration](#basic-configuration)
- [Exchange Configuration](#exchange-configuration)
- [Strategy Configuration](#strategy-configuration)
- [Risk Management Settings](#risk-management-settings)
- [Advanced Configuration](#advanced-configuration)

## Basic Configuration

The basic configuration file (`config.yaml`) contains essential settings for the trading bot:

```yaml
# General Settings
bot_name: "Velocimex"
version: "1.0.0"
log_level: "INFO"

# Trading Settings
base_currency: "USDT"
trading_pairs: ["BTC/USDT", "ETH/USDT"]
max_open_trades: 3
stake_currency: "USDT"
stake_amount: 100
```

## Exchange Configuration

Configure your exchange settings in the `exchange_config.yaml`:

```yaml
exchange:
  name: "binance"  # or "kraken", "coinbase", etc.
  api_key: "your_api_key"
  api_secret: "your_api_secret"
  sandbox: false
  rate_limit: true
  retries: 3
```

## Strategy Configuration

Strategy settings are defined in `strategy_config.yaml`:

```yaml
strategy:
  name: "CustomStrategy"
  timeframe: "1h"
  stop_loss: -0.05
  take_profit: 0.10
  trailing_stop: true
  trailing_stop_positive: 0.01
  indicators:
    rsi:
      period: 14
      overbought: 70
      oversold: 30
    macd:
      fast: 12
      slow: 26
      signal: 9
```

## Risk Management Settings

Configure risk management in `risk_config.yaml`:

```yaml
risk:
  max_risk_per_trade: 0.02  # 2% of portfolio
  max_daily_loss: 0.05     # 5% of portfolio
  position_sizing: "fixed"  # or "dynamic"
  leverage: 1.0
  margin_mode: "isolated"
```

## Advanced Configuration

### Database Configuration
```yaml
database:
  type: "sqlite"  # or "postgresql"
  path: "data/trades.db"
  backup: true
  backup_interval: "1d"
```

### Notification Settings
```yaml
notifications:
  telegram:
    enabled: true
    token: "your_telegram_bot_token"
    chat_id: "your_chat_id"
  email:
    enabled: false
    smtp_server: "smtp.gmail.com"
    smtp_port: 587
    username: "your_email"
    password: "your_password"
```

### Performance Monitoring
```yaml
monitoring:
  metrics:
    enabled: true
    interval: "1h"
  logging:
    level: "INFO"
    file: "logs/velocimex.log"
    max_size: "10MB"
    backup_count: 5
```

## Environment Variables

Some sensitive configurations can be set using environment variables:

```bash
export VELOCIMEX_API_KEY="your_api_key"
export VELOCIMEX_API_SECRET="your_api_secret"
export VELOCIMEX_DB_PASSWORD="your_db_password"
```

## Configuration Validation

The bot includes a configuration validator that checks your settings:

```bash
python -m velocimex validate-config
```

This will verify that all required settings are present and valid.

## Best Practices

1. Always use environment variables for sensitive information
2. Keep configuration files in version control (excluding sensitive data)
3. Use different configurations for development and production
4. Regularly backup your configuration files
5. Document any custom settings you add

## Troubleshooting

Common configuration issues and solutions:

1. **API Connection Issues**
   - Verify API keys and permissions
   - Check network connectivity
   - Ensure correct exchange selection

2. **Strategy Configuration Errors**
   - Validate indicator parameters
   - Check timeframe compatibility
   - Verify stop-loss and take-profit settings

3. **Database Connection Problems**
   - Verify database credentials
   - Check file permissions
   - Ensure sufficient disk space

For more detailed information, refer to the [Technical Documentation](../technical/index.md). 