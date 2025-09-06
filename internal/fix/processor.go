package fix

import (
	"log"
	"time"

	"github.com/quickfixgo/quickfix"
	"github.com/shopspring/decimal"
)

// MessageProcessor handles incoming FIX messages
type MessageProcessor struct {
	client    *Client
	handlers  map[string]MessageHandler
}

// MessageHandler defines the interface for handling specific FIX message types
type MessageHandler interface {
	Handle(message quickfix.Message) error
}

// NewMessageProcessor creates a new message processor
func NewMessageProcessor(client *Client) *MessageProcessor {
	processor := &MessageProcessor{
		client:   client,
		handlers: make(map[string]MessageHandler),
	}

	// Register default handlers
	processor.RegisterHandler("8", &ExecutionReportHandler{}) // ExecutionReport
	processor.RegisterHandler("9", &OrderCancelRejectHandler{}) // OrderCancelReject
	processor.RegisterHandler("3", &RejectHandler{}) // Reject

	return processor
}

// RegisterHandler registers a handler for a specific message type
func (mp *MessageProcessor) RegisterHandler(msgType string, handler MessageHandler) {
	mp.handlers[msgType] = handler
}

// ProcessMessage processes an incoming FIX message
func (mp *MessageProcessor) ProcessMessage(message quickfix.Message) error {
	msgType, err := message.Header.GetString(35) // MsgType
	if err != nil {
		return err
	}

	handler, exists := mp.handlers[msgType]
	if !exists {
		log.Printf("No handler for FIX message type: %s", msgType)
		return nil
	}

	return handler.Handle(message)
}

// Start starts processing messages from the client's message channel
func (mp *MessageProcessor) Start() {
	go func() {
		for {
			select {
			case message := <-mp.client.msgChan:
				if err := mp.ProcessMessage(message); err != nil {
					log.Printf("Error processing FIX message: %v", err)
				}
			case <-mp.client.ctx.Done():
				return
			}
		}
	}()
}

// ExecutionReportHandler handles execution report messages
type ExecutionReportHandler struct{}

func (h *ExecutionReportHandler) Handle(message quickfix.Message) error {
	// Extract execution report fields
	clOrdID, _ := message.Body.GetString(11) // ClOrdID
	orderID, _ := message.Body.GetString(37) // OrderID
	execType, _ := message.Body.GetString(150) // ExecType
	ordStatus, _ := message.Body.GetString(39) // OrdStatus
	symbol, _ := message.Body.GetString(55) // Symbol
	side, _ := message.Body.GetString(54) // Side
	leavesQty, _ := message.Body.GetString(151) // LeavesQty
	cumQty, _ := message.Body.GetString(14) // CumQty
	avgPx, _ := message.Body.GetString(6) // AvgPx
	lastQty, _ := message.Body.GetString(32) // LastQty
	lastPx, _ := message.Body.GetString(31) // LastPx

	log.Printf("Execution Report - ClOrdID: %s, OrderID: %s, ExecType: %s, Status: %s, Symbol: %s, Side: %s, LeavesQty: %s, CumQty: %s, AvgPx: %s, LastQty: %s, LastPx: %s",
		clOrdID, orderID, execType, ordStatus, symbol, side, leavesQty, cumQty, avgPx, lastQty, lastPx)

	// TODO: Update order status in order management system
	// TODO: Trigger strategy callbacks for order updates
	// TODO: Update position tracking

	return nil
}

// OrderCancelRejectHandler handles order cancel reject messages
type OrderCancelRejectHandler struct{}

func (h *OrderCancelRejectHandler) Handle(message quickfix.Message) error {
	clOrdID, _ := message.Body.GetString(11) // ClOrdID
	origClOrdID, _ := message.Body.GetString(41) // OrigClOrdID
	ordStatus, _ := message.Body.GetString(39) // OrdStatus
	cxlRejReason, _ := message.Body.GetString(102) // CxlRejReason
	text, _ := message.Body.GetString(58) // Text

	log.Printf("Order Cancel Reject - ClOrdID: %s, OrigClOrdID: %s, Status: %s, Reason: %s, Text: %s",
		clOrdID, origClOrdID, ordStatus, cxlRejReason, text)

	// TODO: Update order status in order management system
	// TODO: Notify strategy of cancel rejection

	return nil
}

// RejectHandler handles reject messages
type RejectHandler struct{}

func (h *RejectHandler) Handle(message quickfix.Message) error {
	refSeqNum, _ := message.Body.GetString(45) // RefSeqNum
	refTagID, _ := message.Body.GetString(371) // RefTagID
	refMsgType, _ := message.Body.GetString(372) // RefMsgType
	sessionRejectReason, _ := message.Body.GetString(373) // SessionRejectReason
	text, _ := message.Body.GetString(58) // Text

	log.Printf("Reject - RefSeqNum: %s, RefTagID: %s, RefMsgType: %s, Reason: %s, Text: %s",
		refSeqNum, refTagID, refMsgType, sessionRejectReason, text)

	// TODO: Handle reject appropriately
	// TODO: Log reject for debugging

	return nil
}

// OrderStatus represents the status of an order
type OrderStatus struct {
	ClOrdID     string          `json:"cl_ord_id"`
	OrderID     string          `json:"order_id"`
	Symbol      string          `json:"symbol"`
	Side        string          `json:"side"`
	Status      string          `json:"status"`
	Quantity    decimal.Decimal `json:"quantity"`
	LeavesQty   decimal.Decimal `json:"leaves_qty"`
	CumQty      decimal.Decimal `json:"cum_qty"`
	AvgPx       decimal.Decimal `json:"avg_px"`
	LastQty     decimal.Decimal `json:"last_qty"`
	LastPx      decimal.Decimal `json:"last_px"`
	Timestamp   time.Time       `json:"timestamp"`
}

// ParseExecutionReport parses an execution report message into an OrderStatus
func ParseExecutionReport(message quickfix.Message) (*OrderStatus, error) {
	status := &OrderStatus{
		Timestamp: time.Now(),
	}

	var err error

	if status.ClOrdID, err = message.Body.GetString(11); err != nil {
		return nil, err
	}
	if status.OrderID, err = message.Body.GetString(37); err != nil {
		return nil, err
	}
	if status.Symbol, err = message.Body.GetString(55); err != nil {
		return nil, err
	}
	if status.Side, err = message.Body.GetString(54); err != nil {
		return nil, err
	}
	if status.Status, err = message.Body.GetString(39); err != nil {
		return nil, err
	}

	// Parse decimal fields
	if leavesQtyStr, err := message.Body.GetString(151); err == nil {
		if leavesQty, parseErr := decimal.NewFromString(leavesQtyStr); parseErr == nil {
			status.LeavesQty = leavesQty
		}
	}
	if cumQtyStr, err := message.Body.GetString(14); err == nil {
		if cumQty, parseErr := decimal.NewFromString(cumQtyStr); parseErr == nil {
			status.CumQty = cumQty
		}
	}
	if avgPxStr, err := message.Body.GetString(6); err == nil {
		if avgPx, parseErr := decimal.NewFromString(avgPxStr); parseErr == nil {
			status.AvgPx = avgPx
		}
	}
	if lastQtyStr, err := message.Body.GetString(32); err == nil {
		if lastQty, parseErr := decimal.NewFromString(lastQtyStr); parseErr == nil {
			status.LastQty = lastQty
		}
	}
	if lastPxStr, err := message.Body.GetString(31); err == nil {
		if lastPx, parseErr := decimal.NewFromString(lastPxStr); parseErr == nil {
			status.LastPx = lastPx
		}
	}

	return status, nil
}
