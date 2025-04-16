package feeds

// Feed is the interface that all market data feeds must implement
type Feed interface {
	Connect() error
	Disconnect() error
	Subscribe(symbol string) error
	Unsubscribe(symbol string) error
	IsConnected() bool
}
