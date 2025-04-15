package main

import (
        "flag"
        "fmt"
        "log"
        "net/http"
        "os"
        "os/signal"
        "syscall"

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
        
        // Serve static files for UI
        fs := http.FileServer(http.Dir("./ui"))
        router.Handle("/", fs)

        // Start HTTP servers - one for the original port in config and one for port 5000 for Replit
        go func() {
                addr := fmt.Sprintf("0.0.0.0:%d", cfg.Server.Port)
                log.Printf("Starting HTTP server at %s", addr)
                if err := http.ListenAndServe(addr, router); err != nil {
                        log.Printf("HTTP server error: %v", err)
                }
        }()
        
        // Start a second HTTP server on port 5000 for Replit
        go func() {
                addr := "0.0.0.0:5000"
                log.Printf("Starting HTTP server for Replit at %s", addr)
                if err := http.ListenAndServe(addr, router); err != nil {
                        log.Fatalf("HTTP server error on port 5000: %v", err)
                }
        }()
        
        // Print UI URLs
        log.Printf("Web UI available at http://localhost:%d and http://localhost:5000", cfg.Server.Port)
        
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
