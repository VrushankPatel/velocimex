package feeds

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/shopspring/decimal"
	"velocimex/internal/config"
	"velocimex/internal/normalizer"
)

// CoinbaseWebSocketFeed implements WebSocket connection to Coinbase Pro
type CoinbaseWebSocketFeed struct {
	config     config.FeedConfig
	normalizer *normalizer.Normalizer
	conn       *websocket.Conn
	isConnected bool
	mu         sync.Mutex
	done       chan struct{}
	orderBookManager OrderBookManager
}

// CoinbaseMessage represents a Coinbase WebSocket message
type CoinbaseMessage struct {
	Type      string `json:"type"`
	ProductID string `json:"product_id"`
	Bids      [][]string `json:"bids,omitempty"`
	Asks      [][]string `json:"asks,omitempty"`
	Time      string `json:"time,omitempty"`
	Sequence  int64  `json:"sequence,omitempty"`
}

// NewCoinbaseWebSocketFeed creates a new Coinbase WebSocket feed
func NewCoinbaseWebSocketFeed(config config.FeedConfig, norm *normalizer.Normalizer) (*CoinbaseWebSocketFeed, error) {
	return &CoinbaseWebSocketFeed{
		config:     config,
		normalizer: norm,
		done:       make(chan struct{}),
	}, nil
}

// SetOrderBookManager sets the order book manager
func (f *CoinbaseWebSocketFeed) SetOrderBookManager(manager OrderBookManager) {
	f.orderBookManager = manager
}

// Connect establishes a connection to Coinbase WebSocket
func (f *CoinbaseWebSocketFeed) Connect() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.isConnected {
		return nil
	}

	log.Printf("Connecting to Coinbase WebSocket: %s", f.config.URL)

	conn, _, err := websocket.DefaultDialer.Dial(f.config.URL, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to Coinbase WebSocket: %v", err)
	}

	f.conn = conn
	f.isConnected = true

	// Send subscription message
	if err := f.subscribeToChannels(); err != nil {
		f.conn.Close()
		f.isConnected = false
		return fmt.Errorf("failed to subscribe to channels: %v", err)
	}

	// Start message processing
	go f.processMessages()

	log.Printf("Connected to Coinbase WebSocket feed: %s", f.config.Name)
	return nil
}

// Disconnect closes the WebSocket connection
func (f *CoinbaseWebSocketFeed) Disconnect() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if !f.isConnected {
		return nil
	}

	close(f.done)

	if f.conn != nil {
		f.conn.Close()
		f.conn = nil
	}

	f.isConnected = false
	log.Printf("Disconnected from Coinbase WebSocket feed: %s", f.config.Name)
	return nil
}

// Subscribe subscribes to market data for a symbol
func (f *CoinbaseWebSocketFeed) Subscribe(symbol string) error {
	// Coinbase subscription is handled during connection
	log.Printf("Subscribed to %s on Coinbase WebSocket feed %s", symbol, f.config.Name)
	return nil
}

// Unsubscribe unsubscribes from market data for a symbol
func (f *CoinbaseWebSocketFeed) Unsubscribe(symbol string) error {
	// Coinbase doesn't support dynamic unsubscription
	log.Printf("Unsubscribed from %s on Coinbase WebSocket feed %s", symbol, f.config.Name)
	return nil
}

// IsConnected returns whether the feed is connected
func (f *CoinbaseWebSocketFeed) IsConnected() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.isConnected
}

// subscribeToChannels sends subscription message to Coinbase
func (f *CoinbaseWebSocketFeed) subscribeToChannels() error {
	// Convert symbols to Coinbase format (e.g., BTC-USD)
	coinbaseSymbols := make([]string, 0, len(f.config.Symbols))
	for _, symbol := range f.config.Symbols {
		// Convert from BTCUSDT to BTC-USD format
		if len(symbol) >= 6 {
			base := symbol[:3]
			quote := symbol[3:]
			coinbaseSymbols = append(coinbaseSymbols, fmt.Sprintf("%s-%s", base, quote))
		}
	}

	subscription := map[string]interface{}{
		"type":       "subscribe",
		"product_ids": coinbaseSymbols,
		"channels":   []string{"level2"},
	}

	return f.conn.WriteJSON(subscription)
}

// processMessages processes incoming WebSocket messages
func (f *CoinbaseWebSocketFeed) processMessages() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Coinbase WebSocket panic recovered: %v", r)
		}
	}()

	for {
		select {
		case <-f.done:
			return
		default:
			_, message, err := f.conn.ReadMessage()
			if err != nil {
				log.Printf("Coinbase WebSocket read error: %v", err)
				f.handleDisconnection()
				return
			}

			f.handleMessage(message)
		}
	}
}

// handleMessage processes a single WebSocket message
func (f *CoinbaseWebSocketFeed) handleMessage(message []byte) {
	var msg CoinbaseMessage
	if err := json.Unmarshal(message, &msg); err != nil {
		log.Printf("Failed to unmarshal Coinbase message: %v", err)
		return
	}

	// Only process level2 updates
	if msg.Type != "l2update" && msg.Type != "snapshot" {
		return
	}

	// Convert Coinbase data to normalized format
	bids := f.convertPriceLevels(msg.Bids)
	asks := f.convertPriceLevels(msg.Asks)

	// Normalize symbol
	normalizedSymbol := f.normalizer.NormalizeSymbol("coinbase", msg.ProductID)

	// Update order book if manager is available
	if f.orderBookManager != nil {
		f.orderBookManager.UpdateOrderBook("coinbase", normalizedSymbol, bids, asks)
	}

	// Process through normalizer
	orderBookUpdate := &normalizer.OrderBookUpdate{
		Exchange:  "coinbase",
		Symbol:    normalizedSymbol,
		Bids:      bids,
		Asks:      asks,
		Timestamp: f.parseTime(msg.Time),
		Snapshot:  msg.Type == "snapshot",
	}

	f.normalizer.ProcessOrderBookUpdate(orderBookUpdate)
}

// convertPriceLevels converts Coinbase price level format to normalized format
func (f *CoinbaseWebSocketFeed) convertPriceLevels(levels [][]string) []normalizer.PriceLevel {
	result := make([]normalizer.PriceLevel, 0, len(levels))

	for _, level := range levels {
		if len(level) < 2 {
			continue
		}

		price, err := decimal.NewFromString(level[0])
		if err != nil {
			log.Printf("Failed to parse price: %v", err)
			continue
		}

		volume, err := decimal.NewFromString(level[1])
		if err != nil {
			log.Printf("Failed to parse volume: %v", err)
			continue
		}

		// Filter out zero prices and volumes
		if price.IsZero() || volume.IsZero() {
			continue
		}

		result = append(result, normalizer.PriceLevel{
			Price:  price.InexactFloat64(),
			Volume: volume.InexactFloat64(),
		})
	}

	return result
}

// parseTime parses Coinbase time format
func (f *CoinbaseWebSocketFeed) parseTime(timeStr string) time.Time {
	if timeStr == "" {
		return time.Now()
	}

	// Coinbase uses RFC3339 format
	t, err := time.Parse(time.RFC3339, timeStr)
	if err != nil {
		log.Printf("Failed to parse time: %v", err)
		return time.Now()
	}

	return t
}

// handleDisconnection handles WebSocket disconnection
func (f *CoinbaseWebSocketFeed) handleDisconnection() {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.isConnected = false
	if f.conn != nil {
		f.conn.Close()
		f.conn = nil
	}

	// Attempt to reconnect after a delay
	go func() {
		time.Sleep(5 * time.Second)
		if err := f.Connect(); err != nil {
			log.Printf("Failed to reconnect to Coinbase: %v", err)
		}
	}()
}
