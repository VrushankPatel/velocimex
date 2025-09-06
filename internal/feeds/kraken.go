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

// KrakenWebSocketFeed implements WebSocket connection to Kraken
type KrakenWebSocketFeed struct {
	config     config.FeedConfig
	normalizer *normalizer.Normalizer
	conn       *websocket.Conn
	isConnected bool
	mu         sync.Mutex
	done       chan struct{}
	orderBookManager OrderBookManager
}

// KrakenMessage represents a Kraken WebSocket message
type KrakenMessage struct {
	ChannelID int                    `json:"channelID,omitempty"`
	ChannelName string               `json:"channelName,omitempty"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Event     string                 `json:"event,omitempty"`
	Status    string                 `json:"status,omitempty"`
	Version   string                 `json:"version,omitempty"`
}

// KrakenOrderBookData represents Kraken order book data
type KrakenOrderBookData struct {
	Asks [][]string `json:"as"`
	Bids [][]string `json:"bs"`
}

// NewKrakenWebSocketFeed creates a new Kraken WebSocket feed
func NewKrakenWebSocketFeed(config config.FeedConfig, norm *normalizer.Normalizer) (*KrakenWebSocketFeed, error) {
	return &KrakenWebSocketFeed{
		config:     config,
		normalizer: norm,
		done:       make(chan struct{}),
	}, nil
}

// SetOrderBookManager sets the order book manager
func (f *KrakenWebSocketFeed) SetOrderBookManager(manager OrderBookManager) {
	f.orderBookManager = manager
}

// Connect establishes a connection to Kraken WebSocket
func (f *KrakenWebSocketFeed) Connect() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.isConnected {
		return nil
	}

	log.Printf("Connecting to Kraken WebSocket: %s", f.config.URL)

	conn, _, err := websocket.DefaultDialer.Dial(f.config.URL, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to Kraken WebSocket: %v", err)
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

	log.Printf("Connected to Kraken WebSocket feed: %s", f.config.Name)
	return nil
}

// Disconnect closes the WebSocket connection
func (f *KrakenWebSocketFeed) Disconnect() error {
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
	log.Printf("Disconnected from Kraken WebSocket feed: %s", f.config.Name)
	return nil
}

// Subscribe subscribes to market data for a symbol
func (f *KrakenWebSocketFeed) Subscribe(symbol string) error {
	// Kraken subscription is handled during connection
	log.Printf("Subscribed to %s on Kraken WebSocket feed %s", symbol, f.config.Name)
	return nil
}

// Unsubscribe unsubscribes from market data for a symbol
func (f *KrakenWebSocketFeed) Unsubscribe(symbol string) error {
	// Kraken doesn't support dynamic unsubscription
	log.Printf("Unsubscribed from %s on Kraken WebSocket feed %s", symbol, f.config.Name)
	return nil
}

// IsConnected returns whether the feed is connected
func (f *KrakenWebSocketFeed) IsConnected() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.isConnected
}

// subscribeToChannels sends subscription message to Kraken
func (f *KrakenWebSocketFeed) subscribeToChannels() error {
	// Convert symbols to Kraken format (e.g., XBT/USD)
	krakenSymbols := make([]string, 0, len(f.config.Symbols))
	for _, symbol := range f.config.Symbols {
		// Convert from BTCUSDT to XBT/USD format
		if len(symbol) >= 6 {
			base := symbol[:3]
			quote := symbol[3:]
			
			// Convert common symbols to Kraken format
			if base == "BTC" {
				base = "XBT"
			}
			if quote == "USDT" {
				quote = "USD"
			}
			
			krakenSymbols = append(krakenSymbols, fmt.Sprintf("%s/%s", base, quote))
		}
	}

	subscription := map[string]interface{}{
		"event": "subscribe",
		"pair":  krakenSymbols,
		"subscription": map[string]string{
			"name": "book",
		},
	}

	return f.conn.WriteJSON(subscription)
}

// processMessages processes incoming WebSocket messages
func (f *KrakenWebSocketFeed) processMessages() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Kraken WebSocket panic recovered: %v", r)
		}
	}()

	for {
		select {
		case <-f.done:
			return
		default:
			_, message, err := f.conn.ReadMessage()
			if err != nil {
				log.Printf("Kraken WebSocket read error: %v", err)
				f.handleDisconnection()
				return
			}

			f.handleMessage(message)
		}
	}
}

// handleMessage processes a single WebSocket message
func (f *KrakenWebSocketFeed) handleMessage(message []byte) {
	var msg KrakenMessage
	if err := json.Unmarshal(message, &msg); err != nil {
		log.Printf("Failed to unmarshal Kraken message: %v", err)
		return
	}

	// Handle subscription confirmation
	if msg.Event == "subscriptionStatus" {
		if msg.Status == "subscribed" {
			log.Printf("Successfully subscribed to Kraken channel: %s", msg.ChannelName)
		} else {
			log.Printf("Kraken subscription failed: %s", msg.Status)
		}
		return
	}

	// Handle order book data
	if msg.ChannelName == "book" && msg.Data != nil {
		f.handleOrderBookData(msg)
	}
}

// handleOrderBookData processes order book data from Kraken
func (f *KrakenWebSocketFeed) handleOrderBookData(msg KrakenMessage) {
	// Extract symbol from data
	symbol, ok := msg.Data["symbol"].(string)
	if !ok {
		log.Printf("No symbol found in Kraken order book data")
		return
	}

	// Convert data to order book format
	var orderBookData KrakenOrderBookData
	
	// Handle asks
	if asksData, ok := msg.Data["as"].([]interface{}); ok {
		asks := make([][]string, 0, len(asksData))
		for _, ask := range asksData {
			if askArray, ok := ask.([]interface{}); ok && len(askArray) >= 2 {
				askStr := make([]string, 2)
				askStr[0] = fmt.Sprintf("%v", askArray[0])
				askStr[1] = fmt.Sprintf("%v", askArray[1])
				asks = append(asks, askStr)
			}
		}
		orderBookData.Asks = asks
	}

	// Handle bids
	if bidsData, ok := msg.Data["bs"].([]interface{}); ok {
		bids := make([][]string, 0, len(bidsData))
		for _, bid := range bidsData {
			if bidArray, ok := bid.([]interface{}); ok && len(bidArray) >= 2 {
				bidStr := make([]string, 2)
				bidStr[0] = fmt.Sprintf("%v", bidArray[0])
				bidStr[1] = fmt.Sprintf("%v", bidArray[1])
				bids = append(bids, bidStr)
			}
		}
		orderBookData.Bids = bids
	}

	// Convert to normalized format
	bids := f.convertPriceLevels(orderBookData.Bids)
	asks := f.convertPriceLevels(orderBookData.Asks)

	// Normalize symbol
	normalizedSymbol := f.normalizer.NormalizeSymbol("kraken", symbol)

	// Update order book if manager is available
	if f.orderBookManager != nil {
		f.orderBookManager.UpdateOrderBook("kraken", normalizedSymbol, bids, asks)
	}

	// Process through normalizer
	orderBookUpdate := &normalizer.OrderBookUpdate{
		Exchange:  "kraken",
		Symbol:    normalizedSymbol,
		Bids:      bids,
		Asks:      asks,
		Timestamp: time.Now(),
		Snapshot:  false,
	}

	f.normalizer.ProcessOrderBookUpdate(orderBookUpdate)
}

// convertPriceLevels converts Kraken price level format to normalized format
func (f *KrakenWebSocketFeed) convertPriceLevels(levels [][]string) []normalizer.PriceLevel {
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

// handleDisconnection handles WebSocket disconnection
func (f *KrakenWebSocketFeed) handleDisconnection() {
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
			log.Printf("Failed to reconnect to Kraken: %v", err)
		}
	}()
}
