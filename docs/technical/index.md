# Technical Documentation

This section provides technical details about the Velocimex HFT platform architecture, components, and implementation.

## System Architecture

Velocimex is built using a modular, event-driven architecture designed for high performance and low latency. The system is composed of several key components:

### Core Components

1. **Feed Manager**: Manages connections to various market data feeds and normalizes the incoming data.
2. **Order Book Engine**: Constructs and maintains real-time order books from market data feeds.
3. **Normalizer**: Converts exchange-specific data formats into a standardized internal format.
4. **Strategy Engine**: Executes trading strategies and generates signals based on market data.
5. **Order Manager**: Handles order submission, tracking, and execution reporting.
6. **Risk Manager**: Enforces risk constraints and limits across the system.
7. **Backtesting Engine**: Simulates trading strategies against historical data.
8. **Performance Monitor**: Tracks system performance metrics and strategy results.
9. **UI Server**: Provides the user interface for monitoring and control.
10. **API Server**: Exposes REST and WebSocket endpoints for external interaction.

### Diagram

```
+----------------+       +----------------+       +----------------+
|                |       |                |       |                |
|  Market Feeds  +------>+  Normalizer    +------>+  Order Book    |
|                |       |                |       |                |
+----------------+       +----------------+       +-------+--------+
                                                         |
                                                         v
+----------------+       +----------------+       +------+--------+
|                |       |                |       |               |
|  Order Manager |<------+  Strategy      |<------+  Signals      |
|                |       |  Engine        |       |               |
+----------------+       +----------------+       +---------------+
```

## Market Connectivity

Velocimex supports several financial markets:

1. **Cryptocurrency Markets**
   - Binance
   - Coinbase
   - Kraken

2. **Stock Markets**
   - NASDAQ
   - NYSE
   - NSE (National Stock Exchange of India)
   - BSE (Bombay Stock Exchange)
   - S&P 500 (via ETF)
   - Dow Jones (via ETF)

For each market, connections are established using official APIs. When API keys are not available, the system operates in simulation mode, generating realistic market data based on historical patterns and statistical models.

## Data Flow

1. Market data is ingested through feed connectors specific to each exchange or data provider.
2. The normalizer converts exchange-specific formats into a unified internal representation.
3. Order book engine constructs and updates full order books for each symbol.
4. Strategy engine receives order book updates and generates trading signals.
5. Order manager executes trading signals, submitting orders to exchanges.
6. Risk manager validates all orders against risk parameters before submission.
7. Performance monitor tracks execution quality and strategy performance.
8. UI displays real-time information for monitoring and control.

## Technology Stack

- **Backend**: Go (high-performance, concurrent)
- **Frontend**: React.js with ShadCN UI components (with mandatory dark mode)
- **Documentation**: MkDocs (this documentation)
- **Containerization**: Docker
- **Monitoring**: Prometheus integration
- **Database**: SQLite (default) with pluggable PostgreSQL support
- **API**: REST and WebSocket for external integration

## Deployment Options

Velocimex can be deployed in several ways:

1. **Docker Containers**: The recommended deployment method, using Docker and Docker Compose.
2. **Binary Deployment**: Direct deployment of compiled binaries (with Garble obfuscation and UPX compression).
3. **Development Mode**: Local deployment for development and testing.

## Performance Considerations

Velocimex is designed for high-performance, low-latency operation:

- **Memory Management**: Careful memory allocation with minimal garbage collection impact.
- **Lock-Free Algorithms**: Used where possible to minimize contention.
- **Parallelism**: Extensive use of Go's concurrency primitives.
- **Network Optimization**: Efficient network protocols and connection management.
- **Batch Processing**: Batching of operations where appropriate for efficiency.

## Next Steps

- [Installation Guide](installation.md)
- [Configuration Guide](configuration.md)
- [Quick Start](quick_start.md)
- [Markets Guide](markets.md)
- [Strategy Development](first_strategy.md)