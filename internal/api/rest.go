package api

import (
        "encoding/json"
        "fmt"
        "log"
        "net/http"
        "strconv"
        "strings"
        "time"

        "velocimex/internal/normalizer"
        "velocimex/internal/orderbook"
        "velocimex/internal/orders"
        "velocimex/internal/strategy"
)

// RegisterRESTHandlers registers REST API endpoints with the HTTP server
func RegisterRESTHandlers(router *http.ServeMux, bookManager *orderbook.Manager, strategyEngine *strategy.Engine, orderManager orders.OrderManager) {
        // API v1 base path
        const apiBase = "/api/v1"

        // Order book endpoints
        router.HandleFunc(apiBase+"/orderbooks", func(w http.ResponseWriter, r *http.Request) {
                handleOrderBooks(w, r, bookManager)
        })

        // Strategy endpoints
        router.HandleFunc(apiBase+"/strategies", func(w http.ResponseWriter, r *http.Request) {
                handleStrategies(w, r, strategyEngine)
        })

        // Arbitrage opportunities endpoint
        router.HandleFunc(apiBase+"/arbitrage", func(w http.ResponseWriter, r *http.Request) {
                handleArbitrage(w, r, strategyEngine)
        })

        // Market summary endpoint
        router.HandleFunc(apiBase+"/markets", func(w http.ResponseWriter, r *http.Request) {
                handleMarkets(w, r, bookManager)
        })

        // Order management endpoints
        router.HandleFunc(apiBase+"/orders", func(w http.ResponseWriter, r *http.Request) {
                handleOrders(w, r, orderManager)
        })
        
        router.HandleFunc(apiBase+"/orders/", func(w http.ResponseWriter, r *http.Request) {
                handleOrderByID(w, r, orderManager)
        })
        
        router.HandleFunc(apiBase+"/positions", func(w http.ResponseWriter, r *http.Request) {
                handlePositions(w, r, orderManager)
        })
        
        router.HandleFunc(apiBase+"/executions", func(w http.ResponseWriter, r *http.Request) {
                handleExecutions(w, r, orderManager)
        })

        // System status endpoint
        router.HandleFunc(apiBase+"/status", func(w http.ResponseWriter, r *http.Request) {
                handleSystemStatus(w, r)
        })
}

// handleOrderBooks handles requests for order book data
func handleOrderBooks(w http.ResponseWriter, r *http.Request, bookManager *orderbook.Manager) {
        switch r.Method {
        case http.MethodGet:
                // Parse query parameters
                symbol := r.URL.Query().Get("symbol")
                depthStr := r.URL.Query().Get("depth")
                depth := 10 // Default depth

                if depthStr != "" {
                        var err error
                        depth, err = strconv.Atoi(depthStr)
                        if err != nil || depth <= 0 {
                                http.Error(w, "Invalid depth parameter", http.StatusBadRequest)
                                return
                        }
                }

                // If symbol is specified, return order book for that symbol
                if symbol != "" {
                        book := bookManager.GetOrderBook(symbol)
                        if book == nil {
                                http.Error(w, "Order book not found", http.StatusNotFound)
                                return
                        }

                        bids, asks := book.GetDepth(depth)
                        response := struct {
                                Symbol    string                   `json:"symbol"`
                                Timestamp string                   `json:"timestamp"`
                                Bids      []normalizer.PriceLevel `json:"bids"`
                                Asks      []normalizer.PriceLevel `json:"asks"`
                        }{
                                Symbol:    symbol,
                                Timestamp: book.GetTimestamp().Format("2006-01-02T15:04:05.999999Z07:00"),
                                Bids:      bids,
                                Asks:      asks,
                        }

                        writeJSON(w, response)
                        return
                }

                // Otherwise, return list of available symbols
                symbols := bookManager.GetSymbols()
                writeJSON(w, map[string]interface{}{
                        "symbols": symbols,
                })

        default:
                http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        }
}

// handleStrategies handles requests for strategy data
func handleStrategies(w http.ResponseWriter, r *http.Request, strategyEngine *strategy.Engine) {
        switch r.Method {
        case http.MethodGet:
                // Check if we're requesting a specific strategy
                path := strings.TrimPrefix(r.URL.Path, "/api/v1/strategies")
                if path == "" || path == "/" {
                        // Return list of all strategies
                        results := strategyEngine.GetAllResults()
                        writeJSON(w, results)
                        return
                }

                // Extract strategy name from path
                strategyName := strings.TrimPrefix(path, "/")
                strategy, exists := strategyEngine.GetStrategy(strategyName)
                if !exists {
                        http.Error(w, "Strategy not found", http.StatusNotFound)
                        return
                }

                // Return the strategy results
                results := strategy.GetResults()
                writeJSON(w, results)

        case http.MethodPost:
                // Start/stop a strategy
                var request struct {
                        Action string `json:"action"` // "start" or "stop"
                        Name   string `json:"name"`
                }

                if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
                        http.Error(w, "Invalid request", http.StatusBadRequest)
                        return
                }

                strategy, exists := strategyEngine.GetStrategy(request.Name)
                if !exists {
                        http.Error(w, "Strategy not found", http.StatusNotFound)
                        return
                }

                switch request.Action {
                case "start":
                        if err := strategy.Start(r.Context()); err != nil {
                                http.Error(w, fmt.Sprintf("Failed to start strategy: %v", err), http.StatusInternalServerError)
                                return
                        }
                        writeJSON(w, map[string]interface{}{
                                "status":  "success",
                                "message": fmt.Sprintf("Strategy %s started", request.Name),
                        })

                case "stop":
                        if err := strategy.Stop(); err != nil {
                                http.Error(w, fmt.Sprintf("Failed to stop strategy: %v", err), http.StatusInternalServerError)
                                return
                        }
                        writeJSON(w, map[string]interface{}{
                                "status":  "success",
                                "message": fmt.Sprintf("Strategy %s stopped", request.Name),
                        })

                default:
                        http.Error(w, "Invalid action", http.StatusBadRequest)
                }

        default:
                http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        }
}

// handleArbitrage handles requests for arbitrage opportunities
func handleArbitrage(w http.ResponseWriter, r *http.Request, strategyEngine *strategy.Engine) {
        switch r.Method {
        case http.MethodGet:
                // Find the arbitrage strategy
                var arbOpportunities interface{}
                for _, s := range strategyEngine.GetAllStrategies() {
                        if arbStrategy, ok := s.(*strategy.ArbitrageStrategy); ok {
                                arbOpportunities = arbStrategy.GetOpportunities()
                                break
                        }
                }

                if arbOpportunities == nil {
                        writeJSON(w, []interface{}{})
                        return
                }

                writeJSON(w, arbOpportunities)

        default:
                http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        }
}

// handleMarkets handles requests for market summary data
func handleMarkets(w http.ResponseWriter, r *http.Request, bookManager *orderbook.Manager) {
        switch r.Method {
        case http.MethodGet:
                // Get all symbols
                symbols := bookManager.GetSymbols()
                markets := make([]map[string]interface{}, 0, len(symbols))

                // For each symbol, get the mid price and construct a market summary
                for _, symbol := range symbols {
                        book := bookManager.GetOrderBook(symbol)
                        if book == nil {
                                continue
                        }

                        bids, asks := book.GetDepth(1)
                        var midPrice float64
                        if len(bids) > 0 && len(asks) > 0 {
                                midPrice = (bids[0].Price + asks[0].Price) / 2
                        }

                        market := map[string]interface{}{
                                "symbol":    symbol,
                                "price":     midPrice,
                                "timestamp": book.GetTimestamp().Format("2006-01-02T15:04:05.999999Z07:00"),
                        }

                        markets = append(markets, market)
                }

                writeJSON(w, map[string]interface{}{
                        "markets": markets,
                })

        default:
                http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        }
}

// handleSystemStatus handles requests for system status
func handleSystemStatus(w http.ResponseWriter, r *http.Request) {
        switch r.Method {
        case http.MethodGet:
                // Check if we're in simulation mode by examining if any feed is simulated
                isSimulated := false
                
                // We should actually get this from our feed manager instance
                // But for now, since we have no API keys set up, we'll assume simulation mode
                isSimulated = true
                
                status := map[string]interface{}{
                        "status":      "running",
                        "version":     "1.0.0",
                        "timestamp":   fmt.Sprintf("%d", time.Now().Unix()),
                        "isSimulated": isSimulated,
                        "mode":        "simulation", // This will be "live" when using real APIs
                }

                writeJSON(w, status)

        default:
                http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        }
}

// handleOrders handles order management requests
func handleOrders(w http.ResponseWriter, r *http.Request, orderManager orders.OrderManager) {
        switch r.Method {
        case http.MethodGet:
                // Get all orders with optional filters
                filters := make(map[string]interface{})
                if status := r.URL.Query().Get("status"); status != "" {
                        filters["status"] = status
                }
                if exchange := r.URL.Query().Get("exchange"); exchange != "" {
                        filters["exchange"] = exchange
                }
                if symbol := r.URL.Query().Get("symbol"); symbol != "" {
                        filters["symbol"] = symbol
                }
                
                orders, err := orderManager.GetOrders(r.Context(), filters)
                if err != nil {
                        http.Error(w, fmt.Sprintf("Failed to get orders: %v", err), http.StatusInternalServerError)
                        return
                }
                
                writeJSON(w, map[string]interface{}{
                        "orders": orders,
                        "count":  len(orders),
                })
                
        case http.MethodPost:
                // Submit new order
                var req orders.OrderRequest
                if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
                        http.Error(w, "Invalid JSON", http.StatusBadRequest)
                        return
                }
                
                order, err := orderManager.SubmitOrder(r.Context(), &req)
                if err != nil {
                        http.Error(w, fmt.Sprintf("Failed to submit order: %v", err), http.StatusInternalServerError)
                        return
                }
                
                writeJSON(w, order)
                
        default:
                http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        }
}

// handleOrderByID handles requests for specific orders
func handleOrderByID(w http.ResponseWriter, r *http.Request, orderManager orders.OrderManager) {
        // Extract order ID from URL path
        path := strings.TrimPrefix(r.URL.Path, "/api/v1/orders/")
        if path == "" {
                http.Error(w, "Order ID required", http.StatusBadRequest)
                return
        }
        
        switch r.Method {
        case http.MethodGet:
                // Get specific order
                order, err := orderManager.GetOrder(r.Context(), path)
                if err != nil {
                        http.Error(w, fmt.Sprintf("Order not found: %v", err), http.StatusNotFound)
                        return
                }
                
                writeJSON(w, order)
                
        case http.MethodDelete:
                // Cancel order
                err := orderManager.CancelOrder(r.Context(), path)
                if err != nil {
                        http.Error(w, fmt.Sprintf("Failed to cancel order: %v", err), http.StatusInternalServerError)
                        return
                }
                
                writeJSON(w, map[string]string{"status": "cancelled"})
                
        default:
                http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        }
}

// handlePositions handles position management requests
func handlePositions(w http.ResponseWriter, r *http.Request, orderManager orders.OrderManager) {
        switch r.Method {
        case http.MethodGet:
                // Get all positions with optional filters
                filters := make(map[string]interface{})
                if exchange := r.URL.Query().Get("exchange"); exchange != "" {
                        filters["exchange"] = exchange
                }
                if symbol := r.URL.Query().Get("symbol"); symbol != "" {
                        filters["symbol"] = symbol
                }
                
                positions, err := orderManager.GetPositions(r.Context(), filters)
                if err != nil {
                        http.Error(w, fmt.Sprintf("Failed to get positions: %v", err), http.StatusInternalServerError)
                        return
                }
                
                writeJSON(w, map[string]interface{}{
                        "positions": positions,
                        "count":     len(positions),
                })
                
        default:
                http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        }
}

// handleExecutions handles execution history requests
func handleExecutions(w http.ResponseWriter, r *http.Request, orderManager orders.OrderManager) {
        switch r.Method {
        case http.MethodGet:
                // Get execution history with optional filters
                filters := make(map[string]interface{})
                if orderID := r.URL.Query().Get("order_id"); orderID != "" {
                        filters["order_id"] = orderID
                }
                if exchange := r.URL.Query().Get("exchange"); exchange != "" {
                        filters["exchange"] = exchange
                }
                if symbol := r.URL.Query().Get("symbol"); symbol != "" {
                        filters["symbol"] = symbol
                }
                
                executions, err := orderManager.GetExecutions(r.Context(), filters)
                if err != nil {
                        http.Error(w, fmt.Sprintf("Failed to get executions: %v", err), http.StatusInternalServerError)
                        return
                }
                
                writeJSON(w, map[string]interface{}{
                        "executions": executions,
                        "count":      len(executions),
                })
                
        default:
                http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        }
}

// writeJSON writes a JSON response
func writeJSON(w http.ResponseWriter, data interface{}) {
        w.Header().Set("Content-Type", "application/json")
        if err := json.NewEncoder(w).Encode(data); err != nil {
                log.Printf("Error encoding JSON: %v", err)
                http.Error(w, "Internal server error", http.StatusInternalServerError)
        }
}