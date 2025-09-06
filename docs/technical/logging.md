# Comprehensive Logging and Audit System

Velocimex implements a sophisticated logging and audit system designed for high-frequency trading environments where traceability, performance, and compliance are critical.

## Overview

The logging system provides:
- **Structured Logging**: JSON-formatted logs with consistent fields
- **Multi-level Logging**: DEBUG, INFO, WARN, ERROR, FATAL levels
- **Component-based Logging**: Separate loggers for different system components
- **Audit Trail**: Comprehensive audit logging for compliance
- **Log Rotation**: Automatic log file rotation and compression
- **Search and Aggregation**: Advanced log search and analytics
- **Real-time Monitoring**: Log pattern monitoring and alerting
- **Performance Tracking**: Detailed performance and latency logging

## Architecture

### Core Components

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Log Manager   │    │  Audit System   │    │ Search Engine   │
│                 │    │                 │    │                 │
│ - Multi-logger  │    │ - Event Queue   │    │ - Index Builder │
│ - Metrics       │    │ - Processors    │    │ - Query Engine  │
│ - Rotation      │    │ - Validation    │    │ - Aggregation   │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         └───────────────────────┼───────────────────────┘
                                 │
                    ┌─────────────────┐
                    │ Log Monitoring  │
                    │                 │
                    │ - Rule Engine   │
                    │ - Alert System  │
                    │ - Metrics       │
                    └─────────────────┘
```

## Configuration

### Global Configuration

```yaml
logging:
  global:
    level: INFO
    format: json
    output: logs/velocimex.log
    enable_audit: true
    audit_file: logs/audit.log
    max_file_size_mb: 100
    max_backup_files: 10
    compress_backups: true
    enable_trace: true
    trace_header_name: X-Trace-ID
```

### Component-specific Configuration

```yaml
logging:
  components:
    trading:
      level: INFO
      format: json
      output: logs/trading.log
      enable_audit: true
      max_size: 200MB
      max_age: 30d
      max_backups: 20
      compress: true
    
    risk:
      level: WARN
      format: json
      output: logs/risk.log
      enable_audit: true
      max_size: 50MB
      max_age: 90d
      max_backups: 30
      compress: true
    
    audit:
      level: INFO
      format: json
      output: logs/audit.log
      enable_audit: false
      max_size: 500MB
      max_age: 365d
      max_backups: 50
      compress: true
```

### Audit Configuration

```yaml
logging:
  audit:
    enabled: true
    buffer_size: 1000
    workers: 4
    flush_interval: 1m
    retention_days: 30
    compress_old_logs: true
    enable_metrics: true
    enable_alerts: true
    alert_thresholds:
      failed_events: 10
      queue_size: 800
    required_fields:
      - timestamp
      - event_type
    excluded_fields:
      - password
      - token
      - secret
```

## Usage

### Basic Logging

```go
// Get logger for a component
logger := logManager.GetLogger("trading")

// Log with different levels
logger.Info("trading", "Order placed successfully", map[string]interface{}{
    "order_id": "12345",
    "symbol": "BTCUSDT",
    "side": "BUY",
    "quantity": "1.0",
    "price": "50000.00",
})

logger.Error("trading", "Order execution failed", map[string]interface{}{
    "order_id": "12345",
    "error": "Insufficient balance",
    "retry_count": 3,
})
```

### Context-aware Logging

```go
// Create context with trace ID
ctx := context.WithValue(context.Background(), "trace_id", "abc123")

// Log with context
logManager.LogWithContext(ctx, logger.INFO, "trading", "Processing order", map[string]interface{}{
    "order_id": "12345",
    "user_id": "user123",
})
```

### Audit Logging

```go
// Log trade event
logManager.LogTradeEvent(ctx, "12345", "BTCUSDT", "BUY", "1.0", "50000.00", map[string]interface{}{
    "strategy": "arbitrage",
    "exchange": "binance",
})

// Log order event
logManager.LogOrderEvent(ctx, "67890", "ETHUSDT", "SELL", "5.0", "3000.00", "LIMIT", map[string]interface{}{
    "time_in_force": "GTC",
    "client_order_id": "client123",
})

// Log risk event
logManager.LogRiskEvent(ctx, "position_limit_exceeded", "BTCUSDT", map[string]interface{}{
    "current_position": "10.5",
    "max_position": "10.0",
    "excess": "0.5",
})
```

### Performance Logging

```go
// Log performance metrics
start := time.Now()
// ... perform operation ...
logManager.LogPerformance(ctx, "order_processing", time.Since(start), map[string]interface{}{
    "order_count": 100,
    "throughput": "1000 orders/sec",
})
```

## Log Formats

### JSON Format (Default)

```json
{
  "timestamp": "2025-01-27T23:47:28.123456789Z",
  "level": "INFO",
  "message": "Order placed successfully",
  "component": "trading",
  "trace_id": "abc123",
  "order_id": "12345",
  "symbol": "BTCUSDT",
  "side": "BUY",
  "quantity": "1.0",
  "price": "50000.00"
}
```

### Text Format

```
2025-01-27 23:47:28.123 [INFO] [trading] [trace:abc123] Order placed successfully {order_id=12345, symbol=BTCUSDT, side=BUY, quantity=1.0, price=50000.00}
```

### Logstash Format

```json
{
  "@timestamp": "2025-01-27T23:47:28.123456789Z",
  "@version": "1",
  "level": "INFO",
  "message": "Order placed successfully",
  "component": "trading",
  "service": "velocimex",
  "environment": "production",
  "trace_id": "abc123"
}
```

## Log Rotation

### Rotation Strategies

1. **By Size**: Rotate when file reaches maximum size
2. **By Time**: Rotate at specified time intervals
3. **By Size and Time**: Rotate when either condition is met

### Configuration

```yaml
logging:
  rotation:
    strategy: size_and_time
    max_size_bytes: 104857600  # 100MB
    max_age: 7d
    max_backups: 10
    compress: true
    local_time: true
    rotate_on_startup: false
```

### File Naming

- Current log: `trading.log`
- Rotated logs: `trading.2025-01-27T23-47-28.log`
- Compressed: `trading.2025-01-27T23-47-28.log.gz`

## Search and Analytics

### Search Queries

```go
// Search by time range
query := SearchQuery{
    StartTime: &startTime,
    EndTime:   &endTime,
    Levels:    []LogLevel{ERROR, FATAL},
    Limit:     100,
}

results, err := searchEngine.Search(ctx, query)
```

### Aggregation Queries

```go
// Aggregate by component and hour
aggQuery := AggregationQuery{
    StartTime:  &startTime,
    EndTime:    &endTime,
    GroupBy:    []string{"component", "hour"},
    Aggregates: []string{"count"},
}

results, err := searchEngine.Aggregate(ctx, aggQuery)
```

### Trace Analysis

```go
// Get all logs for a trace
traceLogs, err := searchEngine.GetTraceLogs("abc123")
```

## Monitoring and Alerting

### Monitoring Rules

```yaml
rules:
  - id: error_rate_high
    name: High Error Rate
    description: Error rate exceeds threshold
    enabled: true
    severity: warning
    conditions:
      - field: level
        operator: equals
        value: ERROR
        duration: 5m
        count: 10
    actions:
      - type: alert
        config:
          channel: slack
          message: "High error rate detected"
    cooldown: 5m

  - id: critical_errors
    name: Critical Errors
    description: Critical system errors
    enabled: true
    severity: critical
    conditions:
      - field: level
        operator: equals
        value: FATAL
    actions:
      - type: alert
        config:
          channel: email
          recipients: ["admin@velocimex.com"]
      - type: webhook
        config:
          url: "https://alerts.velocimex.com/webhook"
    cooldown: 1m
```

### Alert Processors

```go
// Register custom alert processor
processor := &SlackAlertProcessor{
    webhookURL: "https://hooks.slack.com/...",
    channel:    "#alerts",
}

monitor.RegisterProcessor("slack", processor)
```

## Performance Considerations

### Optimization Strategies

1. **Asynchronous Processing**: Audit events processed asynchronously
2. **Batched Writes**: Multiple log entries written in batches
3. **Compression**: Old log files compressed to save space
4. **Indexing**: Log entries indexed for fast searching
5. **Memory Management**: Proper cleanup of resources

### Metrics

The logging system tracks various metrics:

- Total log entries by level
- Log entries by component
- Audit events processed
- Search query performance
- Alert processing statistics
- Error rates and patterns

## Compliance and Security

### Audit Requirements

- **Immutable Logs**: Audit logs cannot be modified
- **Secure Storage**: Logs encrypted at rest
- **Access Control**: Role-based access to log data
- **Retention Policies**: Configurable retention periods
- **Compliance Reporting**: Automated compliance reports

### Data Privacy

- **PII Filtering**: Personal information automatically filtered
- **Sensitive Data**: Passwords and tokens excluded
- **Data Masking**: Sensitive fields masked in logs
- **Access Logging**: All log access logged

## Troubleshooting

### Common Issues

1. **High Disk Usage**
   - Check log rotation configuration
   - Verify compression is enabled
   - Review retention policies

2. **Slow Search Performance**
   - Check index configuration
   - Verify search query optimization
   - Consider increasing index interval

3. **Missing Logs**
   - Check log level configuration
   - Verify output file permissions
   - Review component-specific settings

4. **Alert Spam**
   - Adjust alert cooldown periods
   - Review monitoring rules
   - Check alert thresholds

### Debugging

```go
// Enable debug logging
config := &Config{
    Level: DEBUG,
    // ... other config
}

// Check log metrics
metrics := logManager.GetMetrics()
fmt.Printf("Total logs: %d\n", metrics.TotalLogs)
fmt.Printf("Errors: %d\n", metrics.Errors)
```

## Best Practices

1. **Use Structured Logging**: Always use structured fields
2. **Include Context**: Add trace IDs and user information
3. **Log at Appropriate Levels**: Use correct severity levels
4. **Avoid Sensitive Data**: Never log passwords or tokens
5. **Use Consistent Fields**: Standardize field names across components
6. **Monitor Log Health**: Set up alerts for log system issues
7. **Regular Cleanup**: Implement proper log rotation and cleanup
8. **Performance Monitoring**: Track logging performance impact

## Integration

### External Systems

- **ELK Stack**: Elasticsearch, Logstash, Kibana
- **Splunk**: Enterprise log management
- **Grafana**: Log visualization and dashboards
- **Prometheus**: Metrics collection
- **Slack/Teams**: Alert notifications
- **Email**: Critical alert notifications

### APIs

- **REST API**: Log search and retrieval
- **WebSocket**: Real-time log streaming
- **GraphQL**: Advanced querying capabilities
- **gRPC**: High-performance log ingestion

This comprehensive logging system ensures that Velocimex maintains full traceability, compliance, and observability required for high-frequency trading operations.
