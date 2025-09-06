package orders

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
)

// OrderStatus represents the current status of an order
type OrderStatus string

const (
	OrderStatusPending    OrderStatus = "PENDING"
	OrderStatusSubmitted  OrderStatus = "SUBMITTED"
	OrderStatusPartial    OrderStatus = "PARTIAL"
	OrderStatusFilled     OrderStatus = "FILLED"
	OrderStatusCancelled  OrderStatus = "CANCELLED"
	OrderStatusRejected   OrderStatus = "REJECTED"
	OrderStatusExpired    OrderStatus = "EXPIRED"
)

// OrderSide represents the side of an order
type OrderSide string

const (
	OrderSideBuy  OrderSide = "BUY"
	OrderSideSell OrderSide = "SELL"
)

// OrderType represents the type of order
type OrderType string

const (
	OrderTypeMarket           OrderType = "MARKET"
	OrderTypeLimit            OrderType = "LIMIT"
	OrderTypeStop             OrderType = "STOP"
	OrderTypeStopLimit        OrderType = "STOP_LIMIT"
	OrderTypeTrailingStop     OrderType = "TRAILING_STOP"
	OrderTypeTakeProfit       OrderType = "TAKE_PROFIT"
	OrderTypeTakeProfitLimit  OrderType = "TAKE_PROFIT_LIMIT"
)

// TimeInForce represents the time in force for an order
type TimeInForce string

const (
	TimeInForceGTC TimeInForce = "GTC" // Good Till Cancelled
	TimeInForceIOC TimeInForce = "IOC" // Immediate Or Cancel
	TimeInForceFOK TimeInForce = "FOK" // Fill Or Kill
	TimeInForceGTX TimeInForce = "GTX" // Good Till Crossing
)

// Order represents a trading order
type Order struct {
	ID           string          `json:"id"`
	ClientID     string          `json:"client_id"`
	Exchange     string          `json:"exchange"`
	Symbol       string          `json:"symbol"`
	Side         OrderSide       `json:"side"`
	Type         OrderType       `json:"type"`
	Quantity     decimal.Decimal `json:"quantity"`
	Price        decimal.Decimal `json:"price"`
	StopPrice    decimal.Decimal `json:"stop_price"`
	TimeInForce  TimeInForce     `json:"time_in_force"`
	Status       OrderStatus     `json:"status"`
	FilledQty    decimal.Decimal `json:"filled_qty"`
	FilledPrice  decimal.Decimal `json:"filled_price"`
	Commission   decimal.Decimal `json:"commission"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
	ExpiresAt    *time.Time      `json:"expires_at,omitempty"`
	StrategyID   string          `json:"strategy_id,omitempty"`
	StrategyName string          `json:"strategy_name,omitempty"`
	Tags         map[string]string `json:"tags,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// OrderUpdate represents an update to an order
type OrderUpdate struct {
	OrderID     string          `json:"order_id"`
	ClientID    string          `json:"client_id"`
	Status      OrderStatus     `json:"status"`
	FilledQty   decimal.Decimal `json:"filled_qty"`
	FilledPrice decimal.Decimal `json:"filled_price"`
	Commission  decimal.Decimal `json:"commission"`
	Timestamp   time.Time       `json:"timestamp"`
	Exchange    string          `json:"exchange"`
	Reason      string          `json:"reason,omitempty"`
}

// Execution represents a single trade execution
type Execution struct {
	ID        string          `json:"id"`
	OrderID   string          `json:"order_id"`
	ClientID  string          `json:"client_id"`
	Exchange  string          `json:"exchange"`
	Symbol    string          `json:"symbol"`
	Side      OrderSide       `json:"side"`
	Quantity  decimal.Decimal `json:"quantity"`
	Price     decimal.Decimal `json:"price"`
	Commission decimal.Decimal `json:"commission"`
	Timestamp time.Time       `json:"timestamp"`
	TradeID   string          `json:"trade_id"`
}

// Position represents a trading position
type Position struct {
	ID         string          `json:"id"`
	Symbol     string          `json:"symbol"`
	Exchange   string          `json:"exchange"`
	Side       OrderSide       `json:"side"`
	Quantity   decimal.Decimal `json:"quantity"`
	EntryPrice decimal.Decimal `json:"entry_price"`
	CurrentPrice decimal.Decimal `json:"current_price"`
	UnrealizedPNL decimal.Decimal `json:"unrealized_pnl"`
	RealizedPNL  decimal.Decimal `json:"realized_pnl"`
	Commission   decimal.Decimal `json:"commission"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
	StrategyID   string          `json:"strategy_id,omitempty"`
	Tags         map[string]string `json:"tags,omitempty"`
}

// OrderRequest represents a request to place an order
type OrderRequest struct {
	ClientID    string                 `json:"client_id"`
	Exchange    string                 `json:"exchange"`
	Symbol      string                 `json:"symbol"`
	Side        OrderSide              `json:"side"`
	Type        OrderType              `json:"type"`
	Quantity    decimal.Decimal        `json:"quantity"`
	Price       decimal.Decimal        `json:"price,omitempty"`
	StopPrice   decimal.Decimal        `json:"stop_price,omitempty"`
	TimeInForce TimeInForce            `json:"time_in_force,omitempty"`
	ExpiresAt   *time.Time             `json:"expires_at,omitempty"`
	StrategyID   string                 `json:"strategy_id,omitempty"`
	StrategyName string                 `json:"strategy_name,omitempty"`
	Tags         map[string]string      `json:"tags,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// RoutingDecision represents a routing decision made by the smart router
type RoutingDecision struct {
	OrderID         string            `json:"order_id"`
	Exchange        string            `json:"exchange"`
	Symbol          string            `json:"symbol"`
	Side            OrderSide         `json:"side"`
	Route           string            `json:"route"`
	Reason          string            `json:"reason"`
	ExpectedSlippage decimal.Decimal `json:"expected_slippage"`
	ExpectedFee     decimal.Decimal  `json:"expected_fee"`
	Confidence      float64          `json:"confidence"`
	Timestamp       time.Time        `json:"timestamp"`
}

// SmartRouter defines the interface for smart order routing
type SmartRouter interface {
	RouteOrder(ctx context.Context, order *OrderRequest) (*RoutingDecision, error)
	UpdateMarketData(exchange string, data interface{})
	GetBestPrice(ctx context.Context, symbol string, side OrderSide, quantity decimal.Decimal) (*RoutingDecision, error)
}

// OrderManager defines the interface for order management
type OrderManager interface {
	SubmitOrder(ctx context.Context, req *OrderRequest) (*Order, error)
	CancelOrder(ctx context.Context, orderID string) error
	GetOrder(ctx context.Context, orderID string) (*Order, error)
	GetOrders(ctx context.Context, filters map[string]interface{}) ([]*Order, error)
	GetPositions(ctx context.Context, filters map[string]interface{}) ([]*Position, error)
	GetExecutions(ctx context.Context, filters map[string]interface{}) ([]*Execution, error)
	UpdateOrderStatus(ctx context.Context, update *OrderUpdate) error
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}