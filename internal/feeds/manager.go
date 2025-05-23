package feeds

import (
        "fmt"
        "log"
        "sync"

        "velocimex/internal/config"
        "velocimex/internal/normalizer"
)

// Feed is the interface that all market data feeds must implement
type Feed interface {
        Connect() error
        Disconnect() error
        Subscribe(symbol string) error
        Unsubscribe(symbol string) error
        IsConnected() bool
}

// Manager manages multiple market data feeds
type Manager struct {
        normalizer *normalizer.Normalizer
        feeds      []Feed
        configs    []config.FeedConfig
        mu         sync.Mutex
}

// NewManager creates a new feed manager
func NewManager(normalizer *normalizer.Normalizer, configs []config.FeedConfig) *Manager {
        return &Manager{
                normalizer: normalizer,
                configs:    configs,
                feeds:      make([]Feed, 0, len(configs)),
        }
}

// Connect connects to all configured feeds
func (m *Manager) Connect() error {
        m.mu.Lock()
        defer m.mu.Unlock()

        for _, config := range m.configs {
                var feed Feed
                var err error

                // Create the appropriate feed based on the type
                switch config.Type {
                case "websocket":
                        feed, err = NewWebSocketFeed(config, m.normalizer)
                case "fix":
                        feed, err = NewFIXFeed(config, m.normalizer)
                case "stock":
                        feed, err = NewStockMarketFeed(config, m.normalizer)
                default:
                        return fmt.Errorf("unsupported feed type: %s", config.Type)
                }

                if err != nil {
                        return fmt.Errorf("failed to create feed %s: %v", config.Name, err)
                }

                // Connect to the feed
                if err := feed.Connect(); err != nil {
                        return fmt.Errorf("failed to connect to feed %s: %v", config.Name, err)
                }

                // Subscribe to symbols
                for _, symbol := range config.Symbols {
                        if err := feed.Subscribe(symbol); err != nil {
                                log.Printf("Failed to subscribe to %s on %s: %v", symbol, config.Name, err)
                        }
                }

                m.feeds = append(m.feeds, feed)
                log.Printf("Connected to feed: %s", config.Name)
        }

        return nil
}

// Disconnect disconnects from all feeds
func (m *Manager) Disconnect() {
        m.mu.Lock()
        defer m.mu.Unlock()

        for _, feed := range m.feeds {
                if feed.IsConnected() {
                        if err := feed.Disconnect(); err != nil {
                                log.Printf("Error disconnecting from feed: %v", err)
                        }
                }
        }
}

// GetConnectedFeeds returns the list of connected feeds
func (m *Manager) GetConnectedFeeds() []Feed {
        m.mu.Lock()
        defer m.mu.Unlock()

        connected := make([]Feed, 0)
        for _, feed := range m.feeds {
                if feed.IsConnected() {
                        connected = append(connected, feed)
                }
        }

        return connected
}