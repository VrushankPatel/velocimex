package orders

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"velocimex/internal/metrics"
)

// ManagerConfig holds configuration for the order manager
type ManagerConfig struct {
	MaxConcurrentOrders int           `json:"max_concurrent_orders"`
	OrderTimeout        time.Duration `json:"order_timeout"`
	RetryAttempts       int           `json:"retry_attempts"`
	RetryDelay          time.Duration `json:"retry_delay"`
	EnablePaperTrading  bool          `json:"enable_paper_trading"`
	DefaultSlippage     decimal.Decimal `json:"default_slippage"`
}

// DefaultManagerConfig returns default configuration
func DefaultManagerConfig() ManagerConfig {
	return ManagerConfig{
		MaxConcurrentOrders: 100,
		OrderTimeout:        30 * time.Second,
		RetryAttempts:       3,
		RetryDelay:          1 * time.Second,
		EnablePaperTrading:  false,
		DefaultSlippage:     decimal.NewFromFloat(0.001),
	}
}

// Manager implements the OrderManager interface
type Manager struct {
	config        ManagerConfig
	orders        map[string]*Order
	positions     map[string]*Position
	executions    map[string][]*Execution
	smartRouter   SmartRouter
	metrics       *metrics.Wrapper
	orderChan     chan *OrderRequest
	updateChan    chan *OrderUpdate
	cancelChan    chan string
	mu            sync.RWMutex
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
	running       bool
	lastOrderID   int64
}

// NewManager creates a new order manager instance
func NewManager(config ManagerConfig, smartRouter SmartRouter, metrics *metrics.Wrapper) *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &Manager{
		config:      config,
		orders:      make(map[string]*Order),
		positions:   make(map[string]*Position),
		executions:  make(map[string][]*Execution),
		smartRouter: smartRouter,
		metrics:     metrics,
		orderChan:   make(chan *OrderRequest, 1000),
		updateChan:  make(chan *OrderUpdate, 1000),
		cancelChan:  make(chan string, 100),
		ctx:         ctx,
		cancel:      cancel,
	}
}

// Start starts the order manager
func (m *Manager) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		return fmt.Errorf("order manager already running")
	}

	m.running = true
	m.ctx, m.cancel = context.WithCancel(ctx)

	// Start worker goroutines
	m.wg.Add(4)
	go m.orderProcessor()
	go m.updateProcessor()
	go m.positionManager()
	go m.cleanupWorker()

	if m.metrics != nil {
		m.metrics.RecordOrderEvent("manager_start", "info")
	}

	log.Println("Order manager started")
	return nil
}

// Stop stops the order manager
func (m *Manager) Stop(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return fmt.Errorf("order manager not running")
	}

	m.running = false
	m.cancel()

	// Wait for all goroutines to finish
	m.wg.Wait()

	close(m.orderChan)
	close(m.updateChan)
	close(m.cancelChan)

	if m.metrics != nil {
		m.metrics.RecordOrderEvent("manager_stop", "info")
	}

	log.Println("Order manager stopped")
	return nil
}

// SubmitOrder submits a new order
func (m *Manager) SubmitOrder(ctx context.Context, req *OrderRequest) (*Order, error) {
	if req == nil {
		return nil, fmt.Errorf("order request cannot be nil")
	}

	if req.Quantity.IsZero() || req.Quantity.IsNegative() {
		return nil, fmt.Errorf("invalid quantity")
	}

	// Generate order ID
	orderID := uuid.New().String()
	if req.ClientID == "" {
		req.ClientID = orderID
	}

	// Route the order using smart router
	routingDecision, err := m.smartRouter.RouteOrder(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to route order: %w", err)
	}

	// Create order
	order := &Order{
		ID:           orderID,
		ClientID:     req.ClientID,
		Exchange:     routingDecision.Exchange,
		Symbol:       req.Symbol,
		Side:         req.Side,
		Type:         req.Type,
		Quantity:     req.Quantity,
		Price:        req.Price,
		StopPrice:    req.StopPrice,
		TimeInForce:  req.TimeInForce,
		Status:       OrderStatusPending,
		FilledQty:    decimal.Zero,
		FilledPrice:  decimal.Zero,
		Commission:   decimal.Zero,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		ExpiresAt:    req.ExpiresAt,
		StrategyID:   req.StrategyID,
		StrategyName: req.StrategyName,
		Tags:         req.Tags,
		Metadata:     req.Metadata,
	}

	// Store order
	m.mu.Lock()
	m.orders[orderID] = order
	m.mu.Unlock()

	// Send to order processor
	select {
	case m.orderChan <- req:
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	// Record metrics
	if m.metrics != nil {
		m.metrics.RecordOrderEvent("order_submitted", "info")
		orderValue, _ := order.Quantity.Mul(order.Price).Float64()
		m.metrics.RecordOrderValue(orderValue)
	}

	return order, nil
}

// CancelOrder cancels an existing order
func (m *Manager) CancelOrder(ctx context.Context, orderID string) error {
	m.mu.RLock()
	order, exists := m.orders[orderID]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("order not found: %s", orderID)
	}

	if order.Status == OrderStatusFilled || order.Status == OrderStatusCancelled {
		return fmt.Errorf("cannot cancel order with status: %s", order.Status)
	}

	// Send to cancel channel
	select {
	case m.cancelChan <- orderID:
	case <-ctx.Done():
		return ctx.Err()
	}

	if m.metrics != nil {
		m.metrics.RecordOrderEvent("order_cancelled", "info")
	}

	return nil
}

// GetOrder retrieves an order by ID
func (m *Manager) GetOrder(ctx context.Context, orderID string) (*Order, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	order, exists := m.orders[orderID]
	if !exists {
		return nil, fmt.Errorf("order not found: %s", orderID)
	}

	return order, nil
}

// GetOrders retrieves orders with optional filters
func (m *Manager) GetOrders(ctx context.Context, filters map[string]interface{}) ([]*Order, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	orders := make([]*Order, 0, len(m.orders))
	for _, order := range m.orders {
		if m.matchesFilters(order, filters) {
			orders = append(orders, order)
		}
	}

	return orders, nil
}

// GetPositions retrieves positions with optional filters
func (m *Manager) GetPositions(ctx context.Context, filters map[string]interface{}) ([]*Position, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	positions := make([]*Position, 0, len(m.positions))
	for _, position := range m.positions {
		if m.matchesPositionFilters(position, filters) {
			positions = append(positions, position)
		}
	}

	return positions, nil
}

// GetExecutions retrieves executions with optional filters
func (m *Manager) GetExecutions(ctx context.Context, filters map[string]interface{}) ([]*Execution, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	executions := make([]*Execution, 0)
	for _, execList := range m.executions {
		for _, execution := range execList {
			if m.matchesExecutionFilters(execution, filters) {
				executions = append(executions, execution)
			}
		}
	}

	return executions, nil
}

// UpdateOrderStatus updates the status of an order
func (m *Manager) UpdateOrderStatus(ctx context.Context, update *OrderUpdate) error {
	if update == nil {
		return fmt.Errorf("order update cannot be nil")
	}

	select {
	case m.updateChan <- update:
	case <-ctx.Done():
		return ctx.Err()
	}

	return nil
}

// orderProcessor processes incoming orders
func (m *Manager) orderProcessor() {
	defer m.wg.Done()

	for {
		select {
		case req := <-m.orderChan:
			if req == nil {
				return
			}
			m.processOrder(req)
		case <-m.ctx.Done():
			return
		}
	}
}

// updateProcessor processes order updates
func (m *Manager) updateProcessor() {
	defer m.wg.Done()

	for {
		select {
		case update := <-m.updateChan:
			if update == nil {
				return
			}
			m.processUpdate(update)
		case <-m.ctx.Done():
			return
		}
	}
}

// positionManager manages positions
func (m *Manager) positionManager() {
	defer m.wg.Done()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.updatePositions()
		case <-m.ctx.Done():
			return
		}
	}
}

// cleanupWorker handles order cleanup and timeout
func (m *Manager) cleanupWorker() {
	defer m.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.cleanupExpiredOrders()
		case orderID := <-m.cancelChan:
			if orderID != "" {
				m.processCancel(orderID)
			}
		case <-m.ctx.Done():
			return
		}
	}
}

// processOrder processes a new order
func (m *Manager) processOrder(req *OrderRequest) {
	// In a real implementation, this would:
	// 1. Validate risk limits
	// 2. Submit to exchange
	// 3. Handle response
	// 4. Update order status

	// For now, simulate order processing
	m.mu.RLock()
	order, exists := m.orders[req.ClientID]
	m.mu.RUnlock()

	if !exists {
		return
	}

	// Simulate order submission
	m.mu.Lock()
	order.Status = OrderStatusSubmitted
	order.UpdatedAt = time.Now()
	m.mu.Unlock()

	// Simulate execution for paper trading
	if m.config.EnablePaperTrading {
		go m.simulateExecution(order)
	}

	if m.metrics != nil {
		m.metrics.RecordOrderEvent("order_processed", "info")
	}
}

// processUpdate processes an order update
func (m *Manager) processUpdate(update *OrderUpdate) {
	m.mu.Lock()
	defer m.mu.Unlock()

	order, exists := m.orders[update.OrderID]
	if !exists {
		return
	}

	// Update order status
	order.Status = update.Status
	order.FilledQty = update.FilledQty
	order.FilledPrice = update.FilledPrice
	order.Commission = update.Commission
	order.UpdatedAt = update.Timestamp

	// Create execution record
	if update.FilledQty.GreaterThan(decimal.Zero) {
		execution := &Execution{
			ID:        uuid.New().String(),
			OrderID:   update.OrderID,
			ClientID:  update.ClientID,
			Exchange:  update.Exchange,
			Symbol:    order.Symbol,
			Side:      order.Side,
			Quantity:  update.FilledQty,
			Price:     update.FilledPrice,
			Commission: update.Commission,
			Timestamp: update.Timestamp,
			TradeID:   update.Exchange + "_" + uuid.New().String(),
		}

		m.executions[update.OrderID] = append(m.executions[update.OrderID], execution)

		// Update position
		m.updatePositionFromExecution(execution)
	}

	if m.metrics != nil {
		m.metrics.RecordOrderEvent("order_updated", string(update.Status))
		filledQty, _ := update.FilledQty.Float64()
		m.metrics.RecordOrderFilled(filledQty)
		filledValue, _ := update.FilledQty.Mul(update.FilledPrice).Float64()
		m.metrics.RecordOrderValue(filledValue)
	}
}

// processCancel processes a cancel request
func (m *Manager) processCancel(orderID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	order, exists := m.orders[orderID]
	if !exists {
		return
	}

	if order.Status == OrderStatusFilled || order.Status == OrderStatusCancelled {
		return
	}

	order.Status = OrderStatusCancelled
	order.UpdatedAt = time.Now()

	if m.metrics != nil {
		m.metrics.RecordOrderEvent("order_cancelled", "info")
	}
}

// updatePositions updates all positions
func (m *Manager) updatePositions() {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Update unrealized P&L for all positions
	for _, position := range m.positions {
		// In a real implementation, this would fetch current prices
		// For now, we'll use the last known price
		position.UpdatedAt = time.Now()
	}

	if m.metrics != nil {
		m.metrics.RecordPositionCount(float64(len(m.positions)))
	}
}

// updatePositionFromExecution updates a position based on an execution
func (m *Manager) updatePositionFromExecution(execution *Execution) {
	positionKey := fmt.Sprintf("%s:%s", execution.Exchange, execution.Symbol)
	
	position, exists := m.positions[positionKey]
	if !exists {
		// Create new position
		position = &Position{
			ID:           uuid.New().String(),
			Symbol:       execution.Symbol,
			Exchange:     execution.Exchange,
			Side:         execution.Side,
			Quantity:     execution.Quantity,
			EntryPrice:   execution.Price,
			CurrentPrice: execution.Price,
			RealizedPNL:  decimal.Zero,
			Commission:   execution.Commission,
			CreatedAt:    execution.Timestamp,
			UpdatedAt:    execution.Timestamp,
		}
		m.positions[positionKey] = position
	} else {
		// Update existing position
		if position.Side == execution.Side {
			// Adding to position
			newQuantity := position.Quantity.Add(execution.Quantity)
			newEntryPrice := ((position.Quantity.Mul(position.EntryPrice)).Add(execution.Quantity.Mul(execution.Price))).Div(newQuantity)
			
			position.Quantity = newQuantity
			position.EntryPrice = newEntryPrice
		} else {
			// Reducing position (closing)
			if execution.Quantity.GreaterThanOrEqual(position.Quantity) {
				// Position fully closed
				realizedPNL := (execution.Price.Sub(position.EntryPrice)).Mul(position.Quantity)
				if position.Side == OrderSideSell {
					realizedPNL = realizedPNL.Neg()
				}
				
				position.RealizedPNL = position.RealizedPNL.Add(realizedPNL)
				position.Quantity = decimal.Zero
			} else {
				// Partial close
				realizedPNL := (execution.Price.Sub(position.EntryPrice)).Mul(execution.Quantity)
				if position.Side == OrderSideSell {
					realizedPNL = realizedPNL.Neg()
				}
				
				position.RealizedPNL = position.RealizedPNL.Add(realizedPNL)
				position.Quantity = position.Quantity.Sub(execution.Quantity)
			}
		}
		
		position.Commission = position.Commission.Add(execution.Commission)
		position.UpdatedAt = execution.Timestamp
	}

	if m.metrics != nil {
		positionValue, _ := position.Quantity.Mul(position.EntryPrice).Float64()
		m.metrics.RecordPositionValue(positionValue)
		realizedPNL, _ := position.RealizedPNL.Float64()
		m.metrics.RecordPositionPNL(realizedPNL)
	}
}

// simulateExecution simulates order execution for paper trading
func (m *Manager) simulateExecution(order *Order) {
	time.Sleep(100 * time.Millisecond) // Simulate network delay

	// Simulate partial or full fill
	fillRatio := decimal.NewFromFloat(0.8 + 0.2*rand.Float64()) // 80-100% fill
	filledQty := order.Quantity.Mul(fillRatio)
	
	// Simulate price with slippage
	var executionPrice decimal.Decimal
	if order.Type == OrderTypeMarket {
		executionPrice = order.Price.Mul(decimal.NewFromFloat(1.0 + 0.001*rand.Float64())) // Small slippage
	} else {
		executionPrice = order.Price
	}

	// Simulate commission
	commission := filledQty.Mul(executionPrice).Mul(decimal.NewFromFloat(0.001))

	update := &OrderUpdate{
		OrderID:     order.ID,
		ClientID:    order.ClientID,
		Status:      OrderStatusFilled,
		FilledQty:   filledQty,
		FilledPrice: executionPrice,
		Commission:  commission,
		Timestamp:   time.Now(),
		Exchange:    order.Exchange,
		Reason:      "paper_trading_simulation",
	}

	m.UpdateOrderStatus(m.ctx, update)
}

// cleanupExpiredOrders removes expired orders
func (m *Manager) cleanupExpiredOrders() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	for orderID, order := range m.orders {
		if order.ExpiresAt != nil && now.After(*order.ExpiresAt) {
			if order.Status == OrderStatusPending || order.Status == OrderStatusSubmitted {
				order.Status = OrderStatusExpired
				order.UpdatedAt = now

				log.Printf("Order %s expired", orderID)
				if m.metrics != nil {
					m.metrics.RecordOrderEvent("order_expired", "info")
				}
			}
		}
	}
}

// matchesFilters checks if an order matches the given filters
func (m *Manager) matchesFilters(order *Order, filters map[string]interface{}) bool {
	for key, value := range filters {
		switch key {
		case "symbol":
			if order.Symbol != value.(string) {
				return false
			}
		case "status":
			if order.Status != value.(OrderStatus) {
				return false
			}
		case "side":
			if order.Side != value.(OrderSide) {
				return false
			}
		case "exchange":
			if order.Exchange != value.(string) {
				return false
			}
		case "strategy_id":
			if order.StrategyID != value.(string) {
				return false
			}
		}
	}
	return true
}

// matchesPositionFilters checks if a position matches the given filters
func (m *Manager) matchesPositionFilters(position *Position, filters map[string]interface{}) bool {
	for key, value := range filters {
		switch key {
		case "symbol":
			if position.Symbol != value.(string) {
				return false
			}
		case "exchange":
			if position.Exchange != value.(string) {
				return false
			}
		case "strategy_id":
			if position.StrategyID != value.(string) {
				return false
			}
		}
	}
	return true
}

// matchesExecutionFilters checks if an execution matches the given filters
func (m *Manager) matchesExecutionFilters(execution *Execution, filters map[string]interface{}) bool {
	for key, value := range filters {
		switch key {
		case "symbol":
			if execution.Symbol != value.(string) {
				return false
			}
		case "exchange":
			if execution.Exchange != value.(string) {
				return false
			}
		case "order_id":
			if execution.OrderID != value.(string) {
				return false
			}
		}
	}
	return true
}

// GetStatistics returns order management statistics
func (m *Manager) GetStatistics() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := map[string]interface{}{
		"total_orders":     len(m.orders),
		"active_orders":    0,
		"filled_orders":    0,
		"cancelled_orders": 0,
		"total_positions":  len(m.positions),
		"total_executions": 0,
	}

	for _, order := range m.orders {
		switch order.Status {
		case OrderStatusPending, OrderStatusSubmitted, OrderStatusPartial:
			stats["active_orders"] = stats["active_orders"].(int) + 1
		case OrderStatusFilled:
			stats["filled_orders"] = stats["filled_orders"].(int) + 1
		case OrderStatusCancelled:
			stats["cancelled_orders"] = stats["cancelled_orders"].(int) + 1
		}
	}

	for _, execList := range m.executions {
		stats["total_executions"] = stats["total_executions"].(int) + len(execList)
	}

	return stats
}
