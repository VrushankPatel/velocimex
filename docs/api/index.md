# API Reference

This document provides comprehensive information about the Velocimex API, including REST endpoints, WebSocket connections, and data formats.

## Table of Contents
- [Authentication](#authentication)
- [REST API](#rest-api)
- [WebSocket API](#websocket-api)
- [Data Formats](#data-formats)
- [Error Handling](#error-handling)
- [Rate Limiting](#rate-limiting)

## Authentication

All API requests require authentication using API keys:

```bash
curl -H "X-API-Key: your_api_key" \
     -H "X-API-Secret: your_api_secret" \
     https://api.velocimex.com/v1/endpoint
```

### API Key Management
- Create new API keys in the dashboard
- Set permissions and IP restrictions
- Rotate keys regularly
- Never share API keys

## REST API

### Market Data

#### Get Market Status
```http
GET /v1/markets/status
```

Response:
```json
{
  "status": "ok",
  "data": {
    "markets": [
      {
        "exchange": "NASDAQ",
        "status": "open",
        "last_update": "2024-04-15T14:30:00Z"
      }
    ]
  }
}
```

#### Get Order Book
```http
GET /v1/markets/{exchange}/orderbook/{symbol}
```

Parameters:
- `exchange`: Exchange identifier (e.g., "NASDAQ", "BINANCE")
- `symbol`: Trading pair (e.g., "AAPL", "BTC/USDT")

### Trading

#### Place Order
```http
POST /v1/trading/orders
```

Request:
```json
{
  "exchange": "NASDAQ",
  "symbol": "AAPL",
  "side": "buy",
  "type": "limit",
  "quantity": 100,
  "price": 150.50
}
```

#### Cancel Order
```http
DELETE /v1/trading/orders/{order_id}
```

### Account Management

#### Get Account Balance
```http
GET /v1/account/balance
```

#### Get Positions
```http
GET /v1/account/positions
```

### Strategy Management

#### List Strategies
```http
GET /v1/strategies
```

#### Deploy Strategy
```http
POST /v1/strategies/{strategy_id}/deploy
```

## WebSocket API

### Market Data Stream
```javascript
const ws = new WebSocket('wss://api.velocimex.com/v1/ws/market');

ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  console.log(data);
};
```

### Order Updates
```javascript
const ws = new WebSocket('wss://api.velocimex.com/v1/ws/orders');

ws.onmessage = (event) => {
  const order = JSON.parse(event.data);
  console.log(order);
};
```

## Data Formats

### Market Data
```json
{
  "timestamp": "2024-04-15T14:30:00.123456Z",
  "exchange": "NASDAQ",
  "symbol": "AAPL",
  "price": 150.50,
  "volume": 1000,
  "bid": 150.49,
  "ask": 150.51
}
```

### Order
```json
{
  "order_id": "123456",
  "timestamp": "2024-04-15T14:30:00Z",
  "exchange": "NASDAQ",
  "symbol": "AAPL",
  "side": "buy",
  "type": "limit",
  "quantity": 100,
  "price": 150.50,
  "status": "open"
}
```

## Error Handling

### Error Response Format
```json
{
  "error": {
    "code": "INVALID_PARAMETER",
    "message": "Invalid parameter value",
    "details": {
      "parameter": "price",
      "value": "-1.00"
    }
  }
}
```

### Common Error Codes
- `AUTHENTICATION_ERROR`: Invalid API credentials
- `INVALID_PARAMETER`: Invalid request parameter
- `RATE_LIMIT_EXCEEDED`: Too many requests
- `INSUFFICIENT_BALANCE`: Not enough funds
- `MARKET_CLOSED`: Market is not open
- `INTERNAL_ERROR`: Server error

## Rate Limiting

### Limits
- REST API: 100 requests per minute
- WebSocket: 10 connections per IP
- Market Data: 1000 messages per second

### Headers
```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1620000000
```

## Best Practices

1. **Error Handling**
   - Implement proper error handling
   - Use exponential backoff
   - Log errors for debugging

2. **Rate Limiting**
   - Monitor rate limits
   - Implement request queuing
   - Use WebSocket when possible

3. **Security**
   - Use HTTPS
   - Rotate API keys
   - Validate responses

4. **Performance**
   - Use WebSocket for real-time data
   - Implement caching
   - Batch requests when possible

For more information about specific endpoints, refer to the [Technical Documentation](../technical/index.md). 