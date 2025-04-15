package api

import (
        "log"
        "net/http"
        "sync"
        "time"

        "github.com/gorilla/websocket"
        "velocimex/internal/orderbook"
        "velocimex/internal/strategy"
)

// WebSocketServer handles WebSocket connections for the API
type WebSocketServer struct {
        orderBooks *orderbook.Manager
        strategies *strategy.Engine
        clients    map[*Client]bool
        broadcast  chan []byte
        register   chan *Client
        unregister chan *Client
        mu         sync.Mutex
        upgrader   websocket.Upgrader
}

// Client represents a connected WebSocket client
type Client struct {
        conn      *websocket.Conn
        server    *WebSocketServer
        send      chan []byte
        mu        sync.Mutex
        symbolSubs map[string]bool
        channelSubs map[string]bool
}

// NewWebSocketServer creates a new WebSocket server
func NewWebSocketServer(books *orderbook.Manager, strategies *strategy.Engine) *WebSocketServer {
        return &WebSocketServer{
                orderBooks: books,
                strategies: strategies,
                clients:    make(map[*Client]bool),
                broadcast:  make(chan []byte, 256),
                register:   make(chan *Client),
                unregister: make(chan *Client),
                upgrader: websocket.Upgrader{
                        ReadBufferSize:  1024,
                        WriteBufferSize: 1024,
                        CheckOrigin: func(r *http.Request) bool {
                                return true // Allow all origins for now
                        },
                },
        }
}

// ServeHTTP handles WebSocket connections
func (s *WebSocketServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
        conn, err := s.upgrader.Upgrade(w, r, nil)
        if err != nil {
                log.Printf("Failed to upgrade to WebSocket: %v", err)
                return
        }

        client := &Client{
                conn:       conn,
                server:     s,
                send:       make(chan []byte, 256),
                symbolSubs: make(map[string]bool),
                channelSubs: make(map[string]bool),
        }

        s.register <- client

        go client.readPump()
        go client.writePump()
}

// Run starts the WebSocket server
func (s *WebSocketServer) Run() {
        for {
                select {
                case client := <-s.register:
                        s.mu.Lock()
                        s.clients[client] = true
                        s.mu.Unlock()
                        log.Printf("New WebSocket client connected: %s", client.conn.RemoteAddr())

                case client := <-s.unregister:
                        s.mu.Lock()
                        if _, ok := s.clients[client]; ok {
                                delete(s.clients, client)
                                close(client.send)
                        }
                        s.mu.Unlock()
                        log.Printf("WebSocket client disconnected: %s", client.conn.RemoteAddr())

                case message := <-s.broadcast:
                        s.mu.Lock()
                        for client := range s.clients {
                                select {
                                case client.send <- message:
                                default:
                                        close(client.send)
                                        delete(s.clients, client)
                                }
                        }
                        s.mu.Unlock()
                }
        }
}

// Close closes all WebSocket connections
func (s *WebSocketServer) Close() {
        s.mu.Lock()
        for client := range s.clients {
                client.conn.Close()
        }
        s.mu.Unlock()
}

// readPump processes incoming messages from the client
func (c *Client) readPump() {
        defer func() {
                c.server.unregister <- c
                c.conn.Close()
        }()

        c.conn.SetReadLimit(4096) // 4KB
        c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
        c.conn.SetPongHandler(func(string) error {
                c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
                return nil
        })

        for {
                _, message, err := c.conn.ReadMessage()
                if err != nil {
                        if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
                                log.Printf("WebSocket error: %v", err)
                        }
                        break
                }

                // Handle message
                c.handleMessage(message)
        }
}

// writePump sends messages to the client
func (c *Client) writePump() {
        ticker := time.NewTicker(30 * time.Second)
        defer func() {
                ticker.Stop()
                c.conn.Close()
        }()

        for {
                select {
                case message, ok := <-c.send:
                        c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
                        if !ok {
                                // Channel was closed
                                c.conn.WriteMessage(websocket.CloseMessage, []byte{})
                                return
                        }

                        w, err := c.conn.NextWriter(websocket.TextMessage)
                        if err != nil {
                                return
                        }
                        w.Write(message)

                        // Add queued messages to the current websocket message
                        n := len(c.send)
                        for i := 0; i < n; i++ {
                                w.Write([]byte{'\n'})
                                w.Write(<-c.send)
                        }

                        if err := w.Close(); err != nil {
                                return
                        }

                case <-ticker.C:
                        c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
                        if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
                                return
                        }
                }
        }
}

// handleMessage processes an incoming message from the client
func (c *Client) handleMessage(msg []byte) {
    // This is a simplified implementation for demo purposes
    // In a real system, we would properly parse JSON and handle various message types
    
    // For now, just send back some sample data for demonstration
    // Let's send a sample order book update
    sampleOrderBook := `{
        "channel": "orderbook",
        "data": {
            "symbol": "BTCUSDT",
            "timestamp": "2025-04-14T16:34:42Z",
            "bids": [
                {"price": 70123.45, "volume": 2.35},
                {"price": 70120.11, "volume": 1.89},
                {"price": 70115.67, "volume": 5.21},
                {"price": 70110.22, "volume": 3.76},
                {"price": 70105.89, "volume": 7.12},
                {"price": 70100.45, "volume": 6.54},
                {"price": 70095.67, "volume": 4.32},
                {"price": 70090.22, "volume": 8.91},
                {"price": 70085.34, "volume": 3.45},
                {"price": 70080.19, "volume": 5.67}
            ],
            "asks": [
                {"price": 70125.78, "volume": 1.56},
                {"price": 70130.44, "volume": 2.78},
                {"price": 70135.89, "volume": 4.32},
                {"price": 70140.56, "volume": 3.21},
                {"price": 70145.22, "volume": 6.78},
                {"price": 70150.67, "volume": 5.43},
                {"price": 70155.34, "volume": 2.87},
                {"price": 70160.89, "volume": 4.56},
                {"price": 70165.45, "volume": 7.89},
                {"price": 70170.23, "volume": 5.43}
            ]
        }
    }`
    
    c.sendMessage([]byte(sampleOrderBook))
    
    // Also send some arbitrage opportunities
    sampleArbitrage := `{
        "channel": "arbitrage",
        "data": [
            {
                "symbol": "BTCUSDT",
                "buyExchange": "Binance",
                "sellExchange": "Coinbase",
                "buyPrice": 70110.22,
                "sellPrice": 70125.78,
                "profitPercent": 0.22,
                "estimatedProfit": 15.56,
                "latencyEstimate": 8,
                "isValid": true
            },
            {
                "symbol": "ETHUSDT",
                "buyExchange": "Kraken",
                "sellExchange": "Binance",
                "buyPrice": 3510.15,
                "sellPrice": 3518.75,
                "profitPercent": 0.25,
                "estimatedProfit": 8.60,
                "latencyEstimate": 12,
                "isValid": true
            }
        ]
    }`
    
    c.sendMessage([]byte(sampleArbitrage))
    
    // And some market data for the market list
    sampleMarketData := `{
        "channel": "system",
        "type": "symbols",
        "data": ["BTCUSDT", "ETHUSDT", "SOLUSDT", "BNBUSDT", "ADAUSDT"]
    }`
    
    c.sendMessage([]byte(sampleMarketData))
}

// sendMessage sends a message to the client
func (c *Client) sendMessage(msg []byte) {
        c.mu.Lock()
        defer c.mu.Unlock()
        
        select {
        case c.send <- msg:
        default:
                c.server.unregister <- c
                c.conn.Close()
        }
}