package feeds

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/shopspring/decimal"
	"velocimex/internal/config"
	"velocimex/internal/normalizer"
)

// BinanceWebSocketFeed implements WebSocket connection to Binance
type BinanceWebSocketFeed struct {
	config     config.FeedConfig
	normalizer *normalizer.Normalizer
	conn       *websocket.Conn
	isConnected bool
	mu         sync.Mutex
	done       chan struct{}
	orderBookManager OrderBookManager
}

// BinanceDepthUpdate represents Binance depth update message
type BinanceDepthUpdate struct {
	Stream string `json:"stream"`
	Data   struct {
		EventType    string     `json:"e"`
		EventTime    int64      `json:"E"`
		Symbol       string     `json:"s"`
		FirstUpdateID int64     `json:"U"`
		FinalUpdateID int64     `json:"u"`
		Bids         [][]string `json:"b"`
		Asks         [][]string `json:"a"`
	} `json:"data"`
}

// OrderBookManager interface for updating order books
type OrderBookManager interface {
	UpdateOrderBook(exchange, symbol string, bids, asks []normalizer.PriceLevel)
}

// NewBinanceWebSocketFeed creates a new Binance WebSocket feed
func NewBinanceWebSocketFeed(config config.FeedConfig, norm *normalizer.Normalizer) (*BinanceWebSocketFeed, error) {
	return &BinanceWebSocketFeed{
		config:     config,
		normalizer: norm,
		done:       make(chan struct{}),
	}, nil
}

// SetOrderBookManager sets the order book manager
func (f *BinanceWebSocketFeed) SetOrderBookManager(manager OrderBookManager) {
	f.orderBookManager = manager
}

// Connect establishes a connection to Binance WebSocket
func (f *BinanceWebSocketFeed) Connect() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.isConnected {
		return nil
	}

	// Build WebSocket URL with streams
	streams := make([]string, 0, len(f.config.Symbols))
	for _, symbol := range f.config.Symbols {
		// Convert symbol format (e.g., BTCUSDT -> btcusdt@depth)
		binanceSymbol := strings.ToLower(symbol)
		streams = append(streams, fmt.Sprintf("%s@depth", binanceSymbol))
	}

	wsURL := fmt.Sprintf("%s/stream?streams=%s", f.config.URL, strings.Join(streams, "/"))

	log.Printf("Connecting to Binance WebSocket: %s", wsURL)

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to Binance WebSocket: %v", err)
	}

	f.conn = conn
	f.isConnected = true

	// Start message processing
	go f.processMessages()

	log.Printf("Connected to Binance WebSocket feed: %s", f.config.Name)
	return nil
}

// Disconnect closes the WebSocket connection
func (f *BinanceWebSocketFeed) Disconnect() error {
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
	log.Printf("Disconnected from Binance WebSocket feed: %s", f.config.Name)
	return nil
}

// Subscribe subscribes to market data for a symbol
func (f *BinanceWebSocketFeed) Subscribe(symbol string) error {
	// Binance subscription is handled during connection
	log.Printf("Subscribed to %s on Binance WebSocket feed %s", symbol, f.config.Name)
	return nil
}

// Unsubscribe unsubscribes from market data for a symbol
func (f *BinanceWebSocketFeed) Unsubscribe(symbol string) error {
	// Binance doesn't support dynamic unsubscription
	log.Printf("Unsubscribed from %s on Binance WebSocket feed %s", symbol, f.config.Name)
	return nil
}

// IsConnected returns whether the feed is connected
func (f *BinanceWebSocketFeed) IsConnected() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.isConnected
}

// processMessages processes incoming WebSocket messages
func (f *BinanceWebSocketFeed) processMessages() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Binance WebSocket panic recovered: %v", r)
		}
	}()

	for {
		select {
		case <-f.done:
			return
		default:
			_, message, err := f.conn.ReadMessage()
			if err != nil {
				log.Printf("Binance WebSocket read error: %v", err)
				f.handleDisconnection()
				return
			}

			f.handleMessage(message)
		}
	}
}

// handleMessage processes a single WebSocket message
func (f *BinanceWebSocketFeed) handleMessage(message []byte) {
	var update BinanceDepthUpdate
	if err := json.Unmarshal(message, &update); err != nil {
		log.Printf("Failed to unmarshal Binance message: %v", err)
		return
	}

	// Convert Binance data to normalized format
	bids := f.convertPriceLevels(update.Data.Bids)
	asks := f.convertPriceLevels(update.Data.Asks)

	// Normalize symbol
	normalizedSymbol := f.normalizer.NormalizeSymbol("binance", update.Data.Symbol)

	// Update order book if manager is available
	if f.orderBookManager != nil {
		f.orderBookManager.UpdateOrderBook("binance", normalizedSymbol, bids, asks)
	}

	// Process through normalizer
	orderBookUpdate := &normalizer.OrderBookUpdate{
		Exchange:  "binance",
		Symbol:    normalizedSymbol,
		Bids:      bids,
		Asks:      asks,
		Timestamp: time.Unix(0, update.Data.EventTime*int64(time.Millisecond)),
		Snapshot:  false,
	}

	f.normalizer.ProcessOrderBookUpdate(orderBookUpdate)
}

// convertPriceLevels converts Binance price level format to normalized format
func (f *BinanceWebSocketFeed) convertPriceLevels(levels [][]string) []normalizer.PriceLevel {
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
func (f *BinanceWebSocketFeed) handleDisconnection() {
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
			log.Printf("Failed to reconnect to Binance: %v", err)
		}
	}()
}
