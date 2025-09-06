package fix

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/quickfixgo/quickfix"
	"github.com/shopspring/decimal"
)

// Client represents a FIX protocol client
type Client struct {
	config     Config
	initiator  *quickfix.Initiator
	connected  bool
	mu         sync.RWMutex
	msgChan    chan quickfix.Message
	ctx        context.Context
	cancel     context.CancelFunc
}

// NewClient creates a new FIX client
func NewClient(config Config) *Client {
	ctx, cancel := context.WithCancel(context.Background())
	return &Client{
		config:  config,
		msgChan: make(chan quickfix.Message, 1000),
		ctx:     ctx,
		cancel:  cancel,
	}
}

// Connect establishes a FIX connection
func (c *Client) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.connected {
		return fmt.Errorf("already connected")
	}

	// Create FIX settings
	settings := quickfix.NewSettings()
	settings.GlobalSettings().Set("ConnectionType", "initiator")
	settings.GlobalSettings().Set("SocketConnectHost", c.config.Host)
	settings.GlobalSettings().Set("SocketConnectPort", fmt.Sprintf("%d", c.config.Port))
	settings.GlobalSettings().Set("StartTime", "00:00:00")
	settings.GlobalSettings().Set("EndTime", "23:59:59")
	settings.GlobalSettings().Set("UseDataDictionary", "Y")
	settings.GlobalSettings().Set("DataDictionary", "FIX44.xml")
	settings.GlobalSettings().Set("TransportDataDictionary", "FIXT11.xml")
	settings.GlobalSettings().Set("AppDataDictionary", "FIX44.xml")
	settings.GlobalSettings().Set("FileLogPath", "logs")
	settings.GlobalSettings().Set("FileStorePath", "store")

	// Session settings
	sessionID := quickfix.SessionID{
		BeginString:  c.config.BeginString,
		SenderCompID: c.config.SenderCompID,
		TargetCompID: c.config.TargetCompID,
	}

	settings.SessionSettings()[sessionID] = quickfix.NewSessionSettings()
	settings.SessionSettings()[sessionID].Set("Username", c.config.Username)
	settings.SessionSettings()[sessionID].Set("Password", c.config.Password)
	settings.SessionSettings()[sessionID].Set("HeartBtInt", fmt.Sprintf("%d", c.config.HeartBtInt))
	settings.SessionSettings()[sessionID].Set("ResetOnLogon", fmt.Sprintf("%t", c.config.ResetSeqNum))

	// Create initiator
	initiator, err := quickfix.NewInitiator(c, quickfix.NewMemoryStoreFactory(), settings, quickfix.NewScreenLogFactory())
	if err != nil {
		return fmt.Errorf("failed to create FIX initiator: %v", err)
	}

	c.initiator = initiator

	// Start the initiator
	if err := c.initiator.Start(); err != nil {
		return fmt.Errorf("failed to start FIX initiator: %v", err)
	}

	c.connected = true
	log.Printf("FIX client connected to %s:%d", c.config.Host, c.config.Port)

	return nil
}

// Disconnect closes the FIX connection
func (c *Client) Disconnect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected {
		return nil
	}

	if c.initiator != nil {
		c.initiator.Stop()
	}

	c.connected = false
	c.cancel()
	log.Printf("FIX client disconnected")

	return nil
}

// IsConnected returns the connection status
func (c *Client) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected
}

// SendOrder sends a new order via FIX protocol
func (c *Client) SendOrder(order OrderRequest) error {
	if !c.IsConnected() {
		return fmt.Errorf("not connected to FIX server")
	}

	// Create new order single message
	msg := quickfix.NewMessage()
	msg.Header.SetField(35, quickfix.FIXString("D")) // MsgType = NewOrderSingle
	msg.Header.SetField(49, quickfix.FIXString(c.config.SenderCompID)) // SenderCompID
	msg.Header.SetField(56, quickfix.FIXString(c.config.TargetCompID)) // TargetCompID
	msg.Header.SetField(34, quickfix.FIXString("1")) // MsgSeqNum (will be set by quickfix)
	msg.Header.SetField(52, quickfix.FIXString(time.Now().UTC().Format("20060102-15:04:05"))) // SendingTime

	// Order fields
	msg.Body.SetField(11, quickfix.FIXString(order.ClOrdID)) // ClOrdID
	msg.Body.SetField(55, quickfix.FIXString(order.Symbol)) // Symbol
	msg.Body.SetField(54, quickfix.FIXString(order.Side)) // Side (1=Buy, 2=Sell)
	msg.Body.SetField(38, quickfix.FIXString(order.Quantity.String())) // OrderQty
	msg.Body.SetField(40, quickfix.FIXString(order.OrderType)) // OrdType
	msg.Body.SetField(59, quickfix.FIXString(order.TimeInForce)) // TimeInForce
	msg.Body.SetField(44, quickfix.FIXString(order.Price.String())) // Price

	return quickfix.Send(msg)
}

// CancelOrder cancels an existing order
func (c *Client) CancelOrder(clOrdID string) error {
	if !c.IsConnected() {
		return fmt.Errorf("not connected to FIX server")
	}

	// Create order cancel request message
	msg := quickfix.NewMessage()
	msg.Header.SetField(35, quickfix.FIXString("F")) // MsgType = OrderCancelRequest
	msg.Header.SetField(49, quickfix.FIXString(c.config.SenderCompID)) // SenderCompID
	msg.Header.SetField(56, quickfix.FIXString(c.config.TargetCompID)) // TargetCompID
	msg.Header.SetField(34, quickfix.FIXString("1")) // MsgSeqNum
	msg.Header.SetField(52, quickfix.FIXString(time.Now().UTC().Format("20060102-15:04:05"))) // SendingTime

	// Cancel request fields
	msg.Body.SetField(11, quickfix.FIXString(clOrdID)) // OrigClOrdID
	msg.Body.SetField(41, quickfix.FIXString(clOrdID)) // ClOrdID

	return quickfix.Send(msg)
}

// OrderRequest represents a FIX order request
type OrderRequest struct {
	ClOrdID      string          `json:"cl_ord_id"`
	Symbol       string          `json:"symbol"`
	Side         string          `json:"side"` // "1" = Buy, "2" = Sell
	Quantity     decimal.Decimal `json:"quantity"`
	OrderType    string          `json:"order_type"` // "1" = Market, "2" = Limit, etc.
	Price        decimal.Decimal `json:"price"`
	TimeInForce  string          `json:"time_in_force"` // "1" = Day, "3" = IOC, etc.
}

// OnCreate implements quickfix.Application interface
func (c *Client) OnCreate(sessionID quickfix.SessionID) {
	log.Printf("FIX session created: %s", sessionID)
}

// OnLogon implements quickfix.Application interface
func (c *Client) OnLogon(sessionID quickfix.SessionID) {
	log.Printf("FIX session logged on: %s", sessionID)
}

// OnLogout implements quickfix.Application interface
func (c *Client) OnLogout(sessionID quickfix.SessionID) {
	log.Printf("FIX session logged out: %s", sessionID)
}

// ToAdmin implements quickfix.Application interface
func (c *Client) ToAdmin(message *quickfix.Message, sessionID quickfix.SessionID) {
	if c.config.LogMessages {
		log.Printf("FIX ToAdmin: %s", message.String())
	}
}

// FromAdmin implements quickfix.Application interface
func (c *Client) FromAdmin(message *quickfix.Message, sessionID quickfix.SessionID) quickfix.MessageRejectError {
	if c.config.LogMessages {
		log.Printf("FIX FromAdmin: %s", message.String())
	}
	return nil
}

// ToApp implements quickfix.Application interface
func (c *Client) ToApp(message *quickfix.Message, sessionID quickfix.SessionID) error {
	if c.config.LogMessages {
		log.Printf("FIX ToApp: %s", message.String())
	}
	return nil
}

// FromApp implements quickfix.Application interface
func (c *Client) FromApp(message *quickfix.Message, sessionID quickfix.SessionID) quickfix.MessageRejectError {
	if c.config.LogMessages {
		log.Printf("FIX FromApp: %s", message.String())
	}

	// Send message to channel for processing
	select {
	case c.msgChan <- *message:
	default:
		log.Printf("FIX message channel full, dropping message")
	}

	return nil
}
