package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"velocimex/internal/api"
	"velocimex/internal/config"
	"velocimex/internal/feeds"
	"velocimex/internal/normalizer"
	"velocimex/internal/orderbook"
	"velocimex/internal/strategy"
)

func main() {
	// Parse command line flags
	configPath := flag.String("config", "config.yaml", "Path to configuration file")
	flag.Parse()

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize components
	normalizer := normalizer.New()
	orderBookManager := orderbook.NewManager()

	// Setup market data feeds
	feedManager := feeds.NewManager(normalizer, cfg.Feeds)
	if err := feedManager.Connect(); err != nil {
		log.Fatalf("Failed to connect to feeds: %v", err)
	}

	// Initialize strategy engine
	strategyEngine := strategy.NewEngine(orderBookManager)
	strategyEngine.RegisterStrategy(strategy.NewArbitrageStrategy(cfg.Strategies.Arbitrage))

	// Start the HTTP and WebSocket server
	router := http.NewServeMux()

	// Register API endpoints
	api.RegisterRESTHandlers(router, orderBookManager, strategyEngine)

	// Setup WebSocket server
	wsServer := api.NewWebSocketServer(orderBookManager, strategyEngine)
	router.Handle("/ws", wsServer)

	// Start WebSocket server
	go wsServer.Run()

	// Forward updates to WebSocket clients
	go func() {
		log.Println("Starting forwarding updates to WebSocket clients")

		// Create channels for receiving updates
		orderbookChan := make(chan orderbook.Update, 100)
		strategyChan := make(chan strategy.Update, 100)

		// Subscribe to updates
		orderBookManager.Subscribe(orderbookChan)
		strategyEngine.Subscribe(strategyChan)

		// Use a ticker for market data updates
		marketTicker := time.NewTicker(1 * time.Second)
		defer marketTicker.Stop()

		for {
			select {
			case update := <-orderbookChan:
				wsServer.BroadcastOrderBookUpdate(api.OrderBookUpdate{
					Symbol:    update.Symbol,
					Timestamp: update.Timestamp,
					Bids:      orderbook.ConvertPriceLevels(update.Bids),
					Asks:      orderbook.ConvertPriceLevels(update.Asks),
				})

			case update := <-strategyChan:
				signals := make([]api.StrategySignal, len(update.RecentSignals))
				for i, s := range update.RecentSignals {
					signals[i] = api.StrategySignal{
						Symbol:    s.Symbol,
						Side:      s.Side,
						Price:     s.Price,
						Volume:    s.Volume,
						Exchange:  s.Exchange,
						Timestamp: s.Timestamp,
					}
				}

				wsServer.BroadcastStrategyUpdate(api.StrategyUpdate{
					ProfitLoss:    update.ProfitLoss,
					Drawdown:      update.Drawdown,
					RecentSignals: signals,
				})

			case <-marketTicker.C:
				// Get current market data
				rawMarkets := feedManager.GetMarketData()
				markets := make([]api.MarketData, len(rawMarkets))
				for i, m := range rawMarkets {
					markets[i] = api.MarketData{
						Symbol:      m.Symbol,
						Price:       m.Price,
						PriceChange: m.PriceChange,
						Exchange:    m.Exchange,
					}
				}
				wsServer.BroadcastMarketData(markets)

				// Get arbitrage opportunities
				rawOpps := strategyEngine.GetArbitrageOpportunities()
				opportunities := make([]api.ArbitrageOpportunity, len(rawOpps))
				for i, o := range rawOpps {
					opportunities[i] = api.ArbitrageOpportunity{
						Symbol:          o.Symbol,
						BuyExchange:     o.BuyExchange,
						SellExchange:    o.SellExchange,
						BuyPrice:        o.BuyPrice,
						SellPrice:       o.SellPrice,
						ProfitPercent:   o.ProfitPercent,
						EstimatedProfit: o.EstimatedProfit,
						LatencyEstimate: o.LatencyEstimate,
						IsValid:         o.IsValid,
					}
				}
				wsServer.BroadcastArbitrageData(opportunities)
			}
		}
	}()

	// Serve static files for UI
	fs := http.FileServer(http.Dir("./ui"))
	router.Handle("/", fs)

	// Always use port 5000 for the server
	go func() {
		addr := "0.0.0.0:5000"
		log.Printf("Starting HTTP server at %s", addr)
		if err := http.ListenAndServe(addr, router); err != nil {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// Print UI URL
	log.Printf("Web UI available at http://localhost:5000")

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Block until we receive a signal
	sig := <-sigChan
	log.Printf("Received signal %v, shutting down...", sig)

	// Graceful shutdown
	feedManager.Disconnect()
	wsServer.Close()

	log.Println("Shutdown complete")
}
