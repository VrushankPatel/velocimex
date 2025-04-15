package feeds

import (
        "log"
        "sync"
        "time"

        "velocimex/internal/config"
        "velocimex/internal/normalizer"
        "velocimex/internal/simulator"
)

// StockMarketFeed implements a feed for stock market data
// It uses real market data when API keys are available,
// otherwise it falls back to simulated data
type StockMarketFeed struct {
        config      config.FeedConfig
        normalizer  *normalizer.Normalizer
        isConnected bool
        isSimulated bool
        simulator   *simulator.MarketSimulator
        updateChan  chan *simulator.MarketUpdate
        mu          sync.RWMutex
        done        chan struct{}
        symbols     map[string]bool
}

// NewStockMarketFeed creates a new stock market feed
func NewStockMarketFeed(config config.FeedConfig, norm *normalizer.Normalizer) (*StockMarketFeed, error) {
        feed := &StockMarketFeed{
                config:     config,
                normalizer: norm,
                done:       make(chan struct{}),
                symbols:    make(map[string]bool),
        }
        
        // Check if API keys are available
        if config.APIKey == "" || config.APISecret == "" {
                // No API keys, use simulation mode
                feed.isSimulated = true
                feed.simulator = simulator.NewMarketSimulator(100 * time.Millisecond)
        }
        
        return feed, nil
}

// Connect establishes a connection to the feed
func (f *StockMarketFeed) Connect() error {
        f.mu.Lock()
        defer f.mu.Unlock()
        
        if f.isConnected {
                return nil
        }
        
        if f.isSimulated {
                // Connect to the simulator
                log.Printf("Connecting to simulated %s feed (WARNING: Using simulated data)", f.config.Name)
                f.updateChan = f.simulator.Subscribe(100)
                f.simulator.Start()
        } else {
                // Connect to the real API
                log.Printf("Connecting to %s feed", f.config.Name)
                // TODO: Implement real API connection when API keys are provided
        }
        
        f.isConnected = true
        
        // Start processing messages
        go f.processMessages()
        
        return nil
}

// Disconnect closes the feed connection
func (f *StockMarketFeed) Disconnect() error {
        f.mu.Lock()
        defer f.mu.Unlock()
        
        if !f.isConnected {
                return nil
        }
        
        close(f.done)
        
        if f.isSimulated {
                // Disconnect from the simulator
                f.simulator.Unsubscribe(f.updateChan)
                f.simulator.Stop()
        } else {
                // Disconnect from the real API
                // TODO: Implement real API disconnection
        }
        
        f.isConnected = false
        
        log.Printf("Disconnected from %s feed", f.config.Name)
        
        return nil
}

// Subscribe subscribes to a symbol
func (f *StockMarketFeed) Subscribe(symbol string) error {
        f.mu.Lock()
        defer f.mu.Unlock()
        
        // Mark the symbol as subscribed
        f.symbols[symbol] = true
        
        if f.isConnected {
                if f.isSimulated {
                        // Simulator already provides data for all symbols
                        log.Printf("Subscribed to %s on %s feed (simulated)", symbol, f.config.Name)
                } else {
                        // TODO: Implement real API subscription
                        log.Printf("Subscribed to %s on %s feed", symbol, f.config.Name)
                }
        }
        
        return nil
}

// Unsubscribe unsubscribes from a symbol
func (f *StockMarketFeed) Unsubscribe(symbol string) error {
        f.mu.Lock()
        defer f.mu.Unlock()
        
        // Remove the symbol subscription
        delete(f.symbols, symbol)
        
        if f.isConnected {
                if f.isSimulated {
                        // Simulator will still provide data for all symbols, but we'll filter it
                        log.Printf("Unsubscribed from %s on %s feed (simulated)", symbol, f.config.Name)
                } else {
                        // TODO: Implement real API unsubscription
                        log.Printf("Unsubscribed from %s on %s feed", symbol, f.config.Name)
                }
        }
        
        return nil
}

// IsConnected returns whether the feed is connected
func (f *StockMarketFeed) IsConnected() bool {
        f.mu.RLock()
        defer f.mu.RUnlock()
        
        return f.isConnected
}

// IsSimulated returns whether the feed is using simulated data
func (f *StockMarketFeed) IsSimulated() bool {
        return f.isSimulated
}

// processMessages processes incoming messages from the feed
func (f *StockMarketFeed) processMessages() {
        if f.isSimulated {
                // Process messages from the simulator
                for {
                        select {
                        case <-f.done:
                                return
                        case update := <-f.updateChan:
                                // Check if we're subscribed to this symbol
                                f.mu.RLock()
                                _, subscribed := f.symbols[update.Symbol]
                                f.mu.RUnlock()
                                
                                if subscribed {
                                        // Process the update
                                        f.processSimulatedUpdate(update)
                                }
                        }
                }
        } else {
                // Process messages from the real API
                // TODO: Implement real API message processing
        }
}

// processSimulatedUpdate processes a single update from the simulator
func (f *StockMarketFeed) processSimulatedUpdate(update *simulator.MarketUpdate) {
        // This is where we would process the update and send it to the normalizer
        // The normalizer would then distribute it to interested components
        
        // For now, we'll just pass the order book update directly
        if update.OrderBook != nil {
                f.normalizer.ProcessOrderBookUpdate(update.OrderBook)
        }
}