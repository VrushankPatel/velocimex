package main

import (
        "context"
        "flag"
	"fmt"
        "log"
        "net/http"
        "os"
        "os/signal"
        "syscall"
        "time"

        "velocimex/internal/api"
        "velocimex/internal/backtesting"
        "velocimex/internal/config"
        "velocimex/internal/feeds"
        "velocimex/internal/normalizer"
        "velocimex/internal/orderbook"
        "velocimex/internal/orders"
        "velocimex/internal/risk"
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
        
        // Initialize order management system
        smartRouter := orders.NewSmartRouter(orders.DefaultSmartRouterConfig(), orderBookManager)
        orderManager := orders.NewManager(orders.DefaultManagerConfig(), smartRouter, nil)
        
        // Initialize risk management system
        riskManager := risk.NewManager(cfg.Risk, nil)
        if err := riskManager.Start(); err != nil {
                log.Fatalf("Failed to start risk manager: %v", err)
        }
        
        // Initialize backtesting engine
        backtestEngine := backtesting.NewEngine()
        if err := backtestEngine.SetConfig(cfg.Backtesting); err != nil {
                log.Fatalf("Failed to configure backtesting engine: %v", err)
        }
        
        // Setup market data feeds
        feedManager := feeds.NewManager(normalizer, cfg.Feeds)
        feedManager.SetOrderBookManager(orderBookManager)
        if err := feedManager.Connect(); err != nil {
                log.Fatalf("Failed to connect to feeds: %v", err)
        }
        
        // Initialize strategy engine
        strategyEngine := strategy.NewEngine(orderBookManager)
        arbitrageStrategy := strategy.NewArbitrageStrategy(cfg.Strategies.Arbitrage)
        strategyEngine.RegisterStrategy(arbitrageStrategy)
        
        // Register strategy with backtesting engine
        if err := backtestEngine.RegisterStrategy(arbitrageStrategy); err != nil {
                log.Fatalf("Failed to register strategy with backtesting engine: %v", err)
        }
        
        // Start the HTTP and WebSocket server
        router := http.NewServeMux()
        
        // Register API endpoints
        api.RegisterRESTHandlers(router, orderBookManager, strategyEngine, orderManager, riskManager, backtestEngine)
        
        // Setup WebSocket server
        wsServer := api.NewWebSocketServer(orderBookManager, strategyEngine, orderManager, riskManager)
        router.Handle("/ws", wsServer)
        
        // Start order manager
        ctx := context.Background()
        if err := orderManager.Start(ctx); err != nil {
                log.Fatalf("Failed to start order manager: %v", err)
        }
        
        // Start WebSocket server
        go wsServer.Run()
        
        // Subscribe to orderbook manager and strategy engine updates and forward them to clients
        go func() {
            log.Println("Starting forwarding updates to WebSocket clients")
            // Use a faster ticker (200ms) for more frequent updates to improve UI responsiveness
            ticker := time.NewTicker(200 * time.Millisecond)
            defer ticker.Stop()
            
            for range ticker.C {
                // Just simulate sending some data to clients for now (test only)
                wsServer.BroadcastSampleData()
            }
        }()
        
        // Serve static files for UI
        fs := http.FileServer(http.Dir("./ui"))
        router.Handle("/", fs)

        // Start the HTTP server
        go func() {
                addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
                log.Printf("Starting HTTP server on %s", addr)
                if err := http.ListenAndServe(addr, router); err != nil {
                        log.Fatalf("HTTP server error: %v", err)
                }
        }()

        // Print UI URL
        log.Printf("Web UI available at http://%s:%d", cfg.Server.Host, cfg.Server.Port)
        
        // Setup signal handling for graceful shutdown
        sigChan := make(chan os.Signal, 1)
        signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
        
        // Block until we receive a signal
        sig := <-sigChan
        log.Printf("Received signal %v, shutting down...", sig)
        
        // Graceful shutdown
        orderManager.Stop(ctx)
        riskManager.Stop()
        backtestEngine.Stop()
        feedManager.Disconnect()
        wsServer.Close()
        
        log.Println("Shutdown complete")
}
