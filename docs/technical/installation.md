# Installation Guide

This guide will walk you through the process of setting up Velocimex on your system.

## Prerequisites

### System Requirements

```{tab} Minimum
- CPU: 4 cores
- RAM: 16GB
- Storage: 100GB SSD
- Network: 1Gbps
```

```{tab} Recommended
- CPU: 8+ cores
- RAM: 32GB+
- Storage: 500GB NVMe SSD
- Network: 10Gbps
```

### Software Requirements

- Go 1.19 or higher
- Docker 24.0 or higher
- Node.js 18.0 or higher
- Git

## Installation Methods

### 1. Docker Installation (Recommended)

```bash
# Clone the repository
git clone https://github.com/VrushankPatel/velocimex.git
cd velocimex

# Build and run with Docker Compose
docker-compose up -d
```

### 2. Manual Installation

#### Backend Setup

```bash
# Clone the repository
git clone https://github.com/VrushankPatel/velocimex.git
cd velocimex

# Install Go dependencies
go mod download

# Build the binary
go build -o velocimex

# Run the application
./velocimex
```

#### Frontend Setup

```bash
# Navigate to the UI directory
cd ui

# Install dependencies
npm install

# Build the frontend
npm run build

# Start the development server
npm run dev
```

## Configuration

### 1. Environment Setup

Create a `.env` file in the root directory:

```ini
# API Keys
BINANCE_API_KEY=your_binance_api_key
BINANCE_SECRET_KEY=your_binance_secret_key
KRAKEN_API_KEY=your_kraken_api_key
KRAKEN_SECRET_KEY=your_kraken_secret_key

# Database Configuration
DB_TYPE=postgres
DB_HOST=localhost
DB_PORT=5432
DB_NAME=velocimex
DB_USER=velocimex
DB_PASSWORD=your_secure_password

# Server Configuration
SERVER_PORT=8080
WS_PORT=8081
LOG_LEVEL=info

# Feature Flags
ENABLE_BACKTESTING=true
ENABLE_PAPER_TRADING=true
ENABLE_LIVE_TRADING=false
```

### 2. Market Data Configuration

Edit `config.yaml` to configure market data sources:

```yaml
markets:
  - name: binance
    type: crypto
    enabled: true
    pairs:
      - BTC-USDT
      - ETH-USDT
      - SOL-USDT
    
  - name: kraken
    type: crypto
    enabled: true
    pairs:
      - XBT/USD
      - ETH/USD

  - name: nasdaq
    type: stock
    enabled: true
    symbols:
      - AAPL
      - MSFT
      - GOOGL
```

## Security Setup

### 1. SSL/TLS Configuration

Generate SSL certificates:

```bash
# Generate self-signed certificates for development
openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
  -keyout certs/private.key -out certs/certificate.crt
```

### 2. Firewall Configuration

Configure your firewall to allow these ports:

- 8080: HTTP API
- 8081: WebSocket
- 5432: PostgreSQL (if using external database)

## Verification

### 1. Health Check

```bash
curl http://localhost:8080/health
```

Expected response:
```json
{
  "status": "healthy",
  "version": "1.0.0",
  "components": {
    "database": "connected",
    "market_data": "streaming",
    "order_manager": "ready"
  }
}
```

### 2. Market Data Check

```bash
curl http://localhost:8080/api/v1/markets/status
```

### 3. UI Access

Open your browser and navigate to:
```
http://localhost:8080
```

## Troubleshooting

### Common Issues

1. **Database Connection Failed**
   ```bash
   # Check database logs
   docker logs velocimex-db
   
   # Verify database connection
   psql -h localhost -U velocimex -d velocimex
   ```

2. **Market Data Not Streaming**
   ```bash
   # Check WebSocket connection
   wscat -c ws://localhost:8081/ws
   
   # Verify API keys
   curl http://localhost:8080/api/v1/exchanges/test
   ```

3. **UI Not Loading**
   ```bash
   # Check frontend logs
   npm run dev -- --debug
   
   # Clear browser cache and reload
   ```

## Next Steps

- [Configuration Guide](configuration.md)
- [Quick Start Tutorial](quick_start.md)
- [Market Connectivity](markets.md) 