package orders

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"velocimex/internal/metrics"
)

// MockSmartRouter is a mock implementation of SmartRouter for testing
type MockSmartRouter struct {
	RouteFunc func(ctx context.Context, req *OrderRequest) (*RoutingDecision, error)
}

func (m *MockSmartRouter) RouteOrder(ctx context.Context, req *OrderRequest) (*RoutingDecision, error) {
	if m.RouteFunc != nil {
		return m.RouteFunc(ctx, req)
	}
	return &RoutingDecision{
		Exchange: "mock_exchange",
		Price:    req.Price,
		Volume:   req.Quantity,
		Score:    1.0,
		Latency:  100 * time.Millisecond,
	}, nil
}

// TestOrderManagerInitialization tests the initialization of the order manager
func TestOrderManagerInitialization(t *testing.T) {
	config := DefaultManagerConfig()
	mockRouter := &MockSmartRouter{}
	metricsWrapper := metrics.NewWrapper(&metrics.Config{Enabled: false})

	manager := NewManager(config, mockRouter, metricsWrapper)
	assert.NotNil(t, manager)
	assert.Equal(t, config, manager.config)
	assert.NotNil(t, manager.orders)
	assert.NotNil(t, manager.positions)
	assert.NotNil(t, manager.executions)
}

// TestSubmitOrder tests order submission functionality
func TestSubmitOrder(t *testing.T) {
	config := DefaultManagerConfig()
	mockRouter := &MockSmartRouter{}
	metricsWrapper := metrics.NewWrapper(&metrics.Config{Enabled: false})

	manager := NewManager(config, mockRouter, metricsWrapper)
	ctx := context.Background()

	err := manager.Start(ctx)
	require.NoError(t, err)
	defer manager.Stop(ctx)

	req := &OrderRequest{
		Symbol:   "BTC/USD",
		Side:     OrderSideBuy,
		Type:     OrderTypeMarket,
		Quantity: decimal.NewFromFloat(1.0),
		Price:    decimal.NewFromFloat(50000.0),
	}

	order, err := manager.SubmitOrder(ctx, req)
	require.NoError(t, err)
	assert.NotEmpty(t, order.ID)
	assert.Equal(t, "BTC/USD", order.Symbol)
	assert.Equal(t, OrderSideBuy, order.Side)
	assert.Equal(t, OrderStatusPending, order.Status)
}

// TestCancelOrder tests order cancellation functionality
func TestCancelOrder(t *testing.T) {
	config := DefaultManagerConfig()
	mockRouter := &MockSmartRouter{}
	metricsWrapper := metrics.NewWrapper(&metrics.Config{Enabled: false})

	manager := NewManager(config, mockRouter, metricsWrapper)
	ctx := context.Background()

	err := manager.Start(ctx)
	require.NoError(t, err)
	defer manager.Stop(ctx)

	req := &OrderRequest{
		Symbol:   "BTC/USD",
		Side:     OrderSideBuy,
		Type:     OrderTypeLimit,
		Quantity: decimal.NewFromFloat(1.0),
		Price:    decimal.NewFromFloat(50000.0),
	}

	order, err := manager.SubmitOrder(ctx, req)
	require.NoError(t, err)

	err = manager.CancelOrder(ctx, order.ID)
	require.NoError(t, err)

	// Wait for cancellation to process
	time.Sleep(100 * time.Millisecond)

	updatedOrder, err := manager.GetOrder(ctx, order.ID)
	require.NoError(t, err)
	assert.Equal(t, OrderStatusCancelled, updatedOrder.Status)
}

// TestGetOrdersWithFilters tests order filtering functionality
func TestGetOrdersWithFilters(t *testing.T) {
	config := DefaultManagerConfig()
	mockRouter := &MockSmartRouter{}
	metricsWrapper := metrics.NewWrapper(&metrics.Config{Enabled: false})

	manager := NewManager(config, mockRouter, metricsWrapper)
	ctx := context.Background()

	err := manager.Start(ctx)
	require.NoError(t, err)
	defer manager.Stop(ctx)

	// Submit multiple orders
	orders := []*OrderRequest{
		{Symbol: "BTC/USD", Side: OrderSideBuy, Type: OrderTypeMarket, Quantity: decimal.NewFromFloat(1.0), Price: decimal.NewFromFloat(50000.0)},
		{Symbol: "ETH/USD", Side: OrderSideSell, Type: OrderTypeLimit, Quantity: decimal.NewFromFloat(10.0), Price: decimal.NewFromFloat(3000.0)},
		{Symbol: "BTC/USD", Side: OrderSideBuy, Type: OrderTypeMarket, Quantity: decimal.NewFromFloat(0.5), Price: decimal.NewFromFloat(51000.0)},
	}

	for _, req := range orders {
		_, err := manager.SubmitOrder(ctx, req)
		require.NoError(t, err)
	}

	// Test filtering by symbol
	btcOrders, err := manager.GetOrders(ctx, map[string]interface{}{"symbol": "BTC/USD"})
	require.NoError(t, err)
	assert.Len(t, btcOrders, 2)

	// Test filtering by side
	buyOrders, err := manager.GetOrders(ctx, map[string]interface{}{"side": OrderSideBuy})
	require.NoError(t, err)
	assert.Len(t, buyOrders, 2)

	// Test filtering by type
	limitOrders, err := manager.GetOrders(ctx, map[string]interface{}{"type": OrderTypeLimit})
	require.NoError(t, err)
	assert.Len(t, limitOrders, 1)
}

// TestPositionManagement tests position tracking functionality
func TestPositionManagement(t *testing.T) {
	config := DefaultManagerConfig()
	mockRouter := &MockSmartRouter{}
	metricsWrapper := metrics.NewWrapper(&metrics.Config{Enabled: false})

	manager := NewManager(config, mockRouter, metricsWrapper)
	ctx := context.Background()

	err := manager.Start(ctx)
	require.NoError(t, err)
	defer manager.Stop(ctx)

	// Submit buy order
	buyReq := &OrderRequest{
		Symbol:   "BTC/USD",
		Side:     OrderSideBuy,
		Type:     OrderTypeMarket,
		Quantity: decimal.NewFromFloat(1.0),
		Price:    decimal.NewFromFloat(50000.0),
	}

	buyOrder, err := manager.SubmitOrder(ctx, buyReq)
	require.NoError(t, err)

	// Simulate execution
	update := &OrderUpdate{
		OrderID:    buyOrder.ID,
		Status:     OrderStatusFilled,
		FilledQty:  decimal.NewFromFloat(1.0),
		FilledPrice: decimal.NewFromFloat(50000.0),
		Commission: decimal.NewFromFloat(50.0),
		Timestamp:  time.Now(),
		Exchange:   "mock_exchange",
	}

	err = manager.UpdateOrderStatus(ctx, update)
	require.NoError(t, err)

	// Verify position creation
	positions, err := manager.GetPositions(ctx, map[string]interface{}{"symbol": "BTC/USD"})
	require.NoError(t, err)
	assert.Len(t, positions, 1)
	assert.Equal(t, "BTC/USD", positions[0].Symbol)
	assert.Equal(t, OrderSideBuy, positions[0].Side)
	assert.Equal(t, decimal.NewFromFloat(1.0), positions[0].Quantity)

	// Submit sell order to close position
	sellReq := &OrderRequest{
		Symbol:   "BTC/USD",
		Side:     OrderSideSell,
		Type:     OrderTypeMarket,
		Quantity: decimal.NewFromFloat(0.5),
		Price:    decimal.NewFromFloat(51000.0),
	}

	sellOrder, err := manager.SubmitOrder(ctx, sellReq)
	require.NoError(t, err)

	// Simulate sell execution
	sellUpdate := &OrderUpdate{
		OrderID:    sellOrder.ID,
		Status:     OrderStatusFilled,
		FilledQty:  decimal.NewFromFloat(0.5),
		FilledPrice: decimal.NewFromFloat(51000.0),
		Commission: decimal.NewFromFloat(25.5),
		Timestamp:  time.Now(),
		Exchange:   "mock_exchange",
	}

	err = manager.UpdateOrderStatus(ctx, sellUpdate)
	require.NoError(t, err)

	// Verify position update
	updatedPositions, err := manager.GetPositions(ctx, map[string]interface{}{"symbol": "BTC/USD"})
	require.NoError(t, err)
	assert.Len(t, updatedPositions, 1)
	assert.Equal(t, decimal.NewFromFloat(0.5), updatedPositions[0].Quantity)
	assert.True(t, updatedPositions[0].RealizedPNL.GreaterThan(decimal.Zero))
}

// TestOrderValidation tests order validation
func TestOrderValidation(t *testing.T) {
	config := DefaultManagerConfig()
	mockRouter := &MockSmartRouter{}
	metricsWrapper := metrics.NewWrapper(&metrics.Config{Enabled: false})

	manager := NewManager(config, mockRouter, metricsWrapper)
	ctx := context.Background()

	err := manager.Start(ctx)
	require.NoError(t, err)
	defer manager.Stop(ctx)

	// Test nil order request
	_, err = manager.SubmitOrder(ctx, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "order request cannot be nil")

	// Test invalid quantity
	req := &OrderRequest{
		Symbol:   "BTC/USD",
		Side:     OrderSideBuy,
		Type:     OrderTypeMarket,
		Quantity: decimal.Zero,
		Price:    decimal.NewFromFloat(50000.0),
	}

	_, err = manager.SubmitOrder(ctx, req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid quantity")
}

// TestConcurrentOrderSubmission tests concurrent order submission
func TestConcurrentOrderSubmission(t *testing.T) {
	config := DefaultManagerConfig()
	mockRouter := &MockSmartRouter{}
	metricsWrapper := metrics.NewWrapper(&metrics.Config{Enabled: false})

	manager := NewManager(config, mockRouter, metricsWrapper)
	ctx := context.Background()

	err := manager.Start(ctx)
	require.NoError(t, err)
	defer manager.Stop(ctx)

	const numOrders = 100
	var wg sync.WaitGroup
	orders := make([]*Order, numOrders)

	for i := 0; i < numOrders; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			
			req := &OrderRequest{
				Symbol:   "BTC/USD",
				Side:     OrderSideBuy,
				Type:     OrderTypeMarket,
				Quantity: decimal.NewFromFloat(0.1),
				Price:    decimal.NewFromFloat(50000.0),
			}

			order, err := manager.SubmitOrder(ctx, req)
			assert.NoError(t, err)
			assert.NotNil(t, order)
			orders[index] = order
		}(i)
	}

	wg.Wait()

	// Verify all orders were created
	allOrders, err := manager.GetOrders(ctx, nil)
	require.NoError(t, err)
	assert.Len(t, allOrders, numOrders)
}

// TestPaperTradingMode tests paper trading functionality
func TestPaperTradingMode(t *testing.T) {
	config := DefaultManagerConfig()
	config.EnablePaperTrading = true
	mockRouter := &MockSmartRouter{}
	metricsWrapper := metrics.NewWrapper(&metrics.Config{Enabled: false})

	manager := NewManager(config, mockRouter, metricsWrapper)
	ctx := context.Background()

	err := manager.Start(ctx)
	require.NoError(t, err)
	defer manager.Stop(ctx)

	req := &OrderRequest{
		Symbol:   "BTC/USD",
		Side:     OrderSideBuy,
		Type:     OrderTypeMarket,
		Quantity: decimal.NewFromFloat(1.0),
		Price:    decimal.NewFromFloat(50000.0),
	}

	order, err := manager.SubmitOrder(ctx, req)
	require.NoError(t, err)

	// Wait for paper trading simulation
	time.Sleep(200 * time.Millisecond)

	// Verify order was filled
	updatedOrder, err := manager.GetOrder(ctx, order.ID)
	require.NoError(t, err)
	assert.Equal(t, OrderStatusFilled, updatedOrder.Status)
	assert.True(t, updatedOrder.FilledQty.GreaterThan(decimal.Zero))
}

// TestOrderTimeout tests order timeout functionality
func TestOrderTimeout(t *testing.T) {
	config := DefaultManagerConfig()
	config.OrderTimeout = 100 * time.Millisecond
	mockRouter := &MockSmartRouter{}
	metricsWrapper := metrics.NewWrapper(&metrics.Config{Enabled: false})

	manager := NewManager(config, mockRouter, metricsWrapper)
	ctx := context.Background()

	err := manager.Start(ctx)
	require.NoError(t, err)
	defer manager.Stop(ctx)

	expiresAt := time.Now().Add(50 * time.Millisecond)
	req := &OrderRequest{
		Symbol:    "BTC/USD",
		Side:      OrderSideBuy,
		Type:      OrderTypeLimit,
		Quantity:  decimal.NewFromFloat(1.0),
		Price:     decimal.NewFromFloat(50000.0),
		ExpiresAt: &expiresAt,
	}

	order, err := manager.SubmitOrder(ctx, req)
	require.NoError(t, err)

	// Wait for expiration
	time.Sleep(200 * time.Millisecond)

	// Verify order was expired
	updatedOrder, err := manager.GetOrder(ctx, order.ID)
	require.NoError(t, err)
	assert.Equal(t, OrderStatusExpired, updatedOrder.Status)
}

// TestStatistics tests statistics collection
func TestStatistics(t *testing.T) {
	config := DefaultManagerConfig()
	mockRouter := &MockSmartRouter{}
	metricsWrapper := metrics.NewWrapper(&metrics.Config{Enabled: false})

	manager := NewManager(config, mockRouter, metricsWrapper)
	ctx := context.Background()

	err := manager.Start(ctx)
	require.NoError(t, err)
	defer manager.Stop(ctx)

	// Submit orders
	orders := []*OrderRequest{
		{Symbol: "BTC/USD", Side: OrderSideBuy, Type: OrderTypeMarket, Quantity: decimal.NewFromFloat(1.0), Price: decimal.NewFromFloat(50000.0)},
		{Symbol: "ETH/USD", Side: OrderSideSell, Type: OrderTypeLimit, Quantity: decimal.NewFromFloat(10.0), Price: decimal.NewFromFloat(3000.0)},
	}

	for _, req := range orders {
		_, err := manager.SubmitOrder(ctx, req)
		require.NoError(t, err)
	}

	// Get statistics
	stats := manager.GetStatistics()
	assert.Equal(t, 2, stats["total_orders"])
	assert.Equal(t, 2, stats["active_orders"])
	assert.Equal(t, 0, stats["filled_orders"])
	assert.Equal(t, 0, stats["cancelled_orders"])
	assert.Equal(t, 0, stats["total_positions"])
	assert.Equal(t, 0, stats["total_executions"])
}

// TestSmartRouterIntegration tests smart router integration
func TestSmartRouterIntegration(t *testing.T) {
	config := DefaultManagerConfig()
	
	// Create mock router with custom behavior
	mockRouter := &MockSmartRouter{
		RouteFunc: func(ctx context.Context, req *OrderRequest) (*RoutingDecision, error) {
			return &RoutingDecision{
				Exchange: "binance",
				Price:    req.Price,
				Volume:   req.Quantity,
				Score:    0.95,
				Latency:  50 * time.Millisecond,
			}, nil
		},
	}
	
	metricsWrapper := metrics.NewWrapper(&metrics.Config{Enabled: false})

	manager := NewManager(config, mockRouter, metricsWrapper)
	ctx := context.Background()

	err := manager.Start(ctx)
	require.NoError(t, err)
	defer manager.Stop(ctx)

	req := &OrderRequest{
		Symbol:   "BTC/USD",
		Side:     OrderSideBuy,
		Type:     OrderTypeMarket,
		Quantity: decimal.NewFromFloat(1.0),
		Price:    decimal.NewFromFloat(50000.0),
	}

	order, err := manager.SubmitOrder(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, "binance", order.Exchange)
}

// TestContextCancellation tests context cancellation handling
func TestContextCancellation(t *testing.T) {
	config := DefaultManagerConfig()
	mockRouter := &MockSmartRouter{}
	metricsWrapper := metrics.NewWrapper(&metrics.Config{Enabled: false})

	manager := NewManager(config, mockRouter, metricsWrapper)
	ctx, cancel := context.WithCancel(context.Background())

	err := manager.Start(ctx)
	require.NoError(t, err)

	// Cancel context
	cancel()

	// Wait for shutdown
	time.Sleep(100 * time.Millisecond)

	// Verify manager is stopped
	assert.False(t, manager.running)
}

// TestErrorHandling tests error handling in various scenarios
func TestErrorHandling(t *testing.T) {
	config := DefaultManagerConfig()
	mockRouter := &MockSmartRouter{
		RouteFunc: func(ctx context.Context, req *OrderRequest) (*RoutingDecision, error) {
			return nil, fmt.Errorf("routing failed")
		},
	}
	metricsWrapper := metrics.NewWrapper(&metrics.Config{Enabled: false})

	manager := NewManager(config, mockRouter, metricsWrapper)
	ctx := context.Background()

	err := manager.Start(ctx)
	require.NoError(t, err)
	defer manager.Stop(ctx)

	req := &OrderRequest{
		Symbol:   "BTC/USD",
		Side:     OrderSideBuy,
		Type:     OrderTypeMarket,
		Quantity: decimal.NewFromFloat(1.0),
		Price:    decimal.NewFromFloat(50000.0),
	}

	_, err := manager.SubmitOrder(ctx, req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "routing failed")
}

// TestConcurrentAccess tests concurrent access to order manager
func TestConcurrentAccess(t *testing.T) {
	config := DefaultManagerConfig()
	mockRouter := &MockSmartRouter{}
	metricsWrapper := metrics.NewWrapper(&metrics.Config{Enabled: false})

	manager := NewManager(config, mockRouter, metricsWrapper)
	ctx := context.Background()

	err := manager.Start(ctx)
	require.NoError(t, err)
	defer manager.Stop(ctx)

	const numGoroutines = 50
	var wg sync.WaitGroup

	// Concurrent order submission
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			
			req := &OrderRequest{
				Symbol:   "BTC/USD",
				Side:     OrderSideBuy,
				Type:     OrderTypeMarket,
				Quantity: decimal.NewFromFloat(0.01),
				Price:    decimal.NewFromFloat(50000.0),
			}

			order, err := manager.SubmitOrder(ctx, req)
			assert.NoError(t, err)
			assert.NotNil(t, order)
		}(i)
	}

	// Concurrent order retrieval
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			
			orders, err := manager.GetOrders(ctx, nil)
			assert.NoError(t, err)
			assert.NotNil(t, orders)
		}(i)
	}

	// Concurrent statistics retrieval
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			
			stats := manager.GetStatistics()
			assert.NotNil(t, stats)
		}(i)
	}

	wg.Wait()
}