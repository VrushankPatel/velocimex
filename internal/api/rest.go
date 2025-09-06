package api

import (
        "encoding/json"
        "fmt"
        "log"
        "net/http"
        "strconv"
        "strings"
        "time"

        "velocimex/internal/backtesting"
        "velocimex/internal/normalizer"
        "velocimex/internal/orderbook"
        "velocimex/internal/orders"
        "velocimex/internal/plugins"
        "velocimex/internal/risk"
        "velocimex/internal/strategy"
)

// RegisterRESTHandlers registers REST API endpoints with the HTTP server
func RegisterRESTHandlers(router *http.ServeMux, bookManager *orderbook.Manager, strategyEngine *strategy.Engine, orderManager orders.OrderManager, riskManager risk.RiskManager, backtestEngine backtesting.BacktestEngine, pluginManager plugins.PluginManager) {
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
        
        // Risk management endpoints
        router.HandleFunc(apiBase+"/risk/portfolio", func(w http.ResponseWriter, r *http.Request) {
                handleRiskPortfolio(w, r, riskManager)
        })
        
        router.HandleFunc(apiBase+"/risk/metrics", func(w http.ResponseWriter, r *http.Request) {
                handleRiskMetrics(w, r, riskManager)
        })
        
        router.HandleFunc(apiBase+"/risk/events", func(w http.ResponseWriter, r *http.Request) {
                handleRiskEvents(w, r, riskManager)
        })
        
        router.HandleFunc(apiBase+"/risk/positions", func(w http.ResponseWriter, r *http.Request) {
                handleRiskPositions(w, r, riskManager)
        })
        
        // Backtesting endpoints
        router.HandleFunc(apiBase+"/backtesting/run", func(w http.ResponseWriter, r *http.Request) {
                handleBacktestRun(w, r, backtestEngine)
        })
        
        router.HandleFunc(apiBase+"/backtesting/strategies", func(w http.ResponseWriter, r *http.Request) {
                handleBacktestStrategies(w, r, backtestEngine)
        })
        
        router.HandleFunc(apiBase+"/backtesting/data", func(w http.ResponseWriter, r *http.Request) {
                handleBacktestData(w, r, backtestEngine)
        })
        
        router.HandleFunc(apiBase+"/backtesting/config", func(w http.ResponseWriter, r *http.Request) {
                handleBacktestConfig(w, r, backtestEngine)
        })
        
        // Plugin management endpoints
        router.HandleFunc(apiBase+"/plugins", func(w http.ResponseWriter, r *http.Request) {
                handlePlugins(w, r, pluginManager)
        })
        
        router.HandleFunc(apiBase+"/plugins/", func(w http.ResponseWriter, r *http.Request) {
                handlePluginByID(w, r, pluginManager)
        })
        
        router.HandleFunc(apiBase+"/plugins/discover", func(w http.ResponseWriter, r *http.Request) {
                handlePluginDiscover(w, r, pluginManager)
        })
        
        router.HandleFunc(apiBase+"/plugins/health", func(w http.ResponseWriter, r *http.Request) {
                handlePluginHealth(w, r, pluginManager)
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

// handleRiskPortfolio handles risk portfolio requests
func handleRiskPortfolio(w http.ResponseWriter, r *http.Request, riskManager risk.RiskManager) {
        switch r.Method {
        case http.MethodGet:
                portfolio := riskManager.GetPortfolio()
                writeJSON(w, portfolio)
        default:
                http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        }
}

// handleRiskMetrics handles risk metrics requests
func handleRiskMetrics(w http.ResponseWriter, r *http.Request, riskManager risk.RiskManager) {
        switch r.Method {
        case http.MethodGet:
                metrics := riskManager.GetRiskMetrics()
                writeJSON(w, metrics)
        default:
                http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        }
}

// handleRiskEvents handles risk events requests
func handleRiskEvents(w http.ResponseWriter, r *http.Request, riskManager risk.RiskManager) {
        switch r.Method {
        case http.MethodGet:
                // Get risk events with optional filters
                filters := make(map[string]interface{})
                if severity := r.URL.Query().Get("severity"); severity != "" {
                        filters["severity"] = severity
                }
                if eventType := r.URL.Query().Get("type"); eventType != "" {
                        filters["type"] = eventType
                }
                if symbol := r.URL.Query().Get("symbol"); symbol != "" {
                        filters["symbol"] = symbol
                }
                
                events, err := riskManager.GetRiskEvents(filters)
                if err != nil {
                        http.Error(w, fmt.Sprintf("Failed to get risk events: %v", err), http.StatusInternalServerError)
                        return
                }
                
                writeJSON(w, map[string]interface{}{
                        "events": events,
                        "count":  len(events),
                })
        default:
                http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        }
}

// handleRiskPositions handles risk positions requests
func handleRiskPositions(w http.ResponseWriter, r *http.Request, riskManager risk.RiskManager) {
        switch r.Method {
        case http.MethodGet:
                positions := riskManager.GetPositions()
                writeJSON(w, map[string]interface{}{
                        "positions": positions,
                        "count":     len(positions),
                })
        default:
                http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        }
}

// handleBacktestRun handles backtest execution requests
func handleBacktestRun(w http.ResponseWriter, r *http.Request, backtestEngine backtesting.BacktestEngine) {
        switch r.Method {
        case http.MethodPost:
                // Parse request body for backtest parameters
                var request struct {
                        StrategyID string `json:"strategy_id,omitempty"`
                        StartDate  string `json:"start_date,omitempty"`
                        EndDate    string `json:"end_date,omitempty"`
                }
                
                if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
                        http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
                        return
                }
                
                // Update config if dates provided
                config := backtestEngine.GetConfig()
                if request.StartDate != "" {
                        if startDate, err := time.Parse("2006-01-02", request.StartDate); err == nil {
                                config.StartDate = startDate
                        }
                }
                if request.EndDate != "" {
                        if endDate, err := time.Parse("2006-01-02", request.EndDate); err == nil {
                                config.EndDate = endDate
                        }
                }
                
                if err := backtestEngine.SetConfig(config); err != nil {
                        http.Error(w, fmt.Sprintf("Failed to update config: %v", err), http.StatusInternalServerError)
                        return
                }
                
                // Run backtest
                var result *backtesting.BacktestResult
                var err error
                
                if request.StrategyID != "" {
                        result, err = backtestEngine.RunBacktestWithStrategy(request.StrategyID)
                } else {
                        result, err = backtestEngine.RunBacktest()
                }
                
                if err != nil {
                        http.Error(w, fmt.Sprintf("Backtest failed: %v", err), http.StatusInternalServerError)
                        return
                }
                
                writeJSON(w, result)
        default:
                http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        }
}

// handleBacktestStrategies handles backtest strategies requests
func handleBacktestStrategies(w http.ResponseWriter, r *http.Request, backtestEngine backtesting.BacktestEngine) {
        switch r.Method {
        case http.MethodGet:
                strategies := backtestEngine.GetRegisteredStrategies()
                strategyList := make([]map[string]interface{}, 0, len(strategies))
                
                for _, strategy := range strategies {
                        strategyList = append(strategyList, map[string]interface{}{
                                "id":   strategy.GetID(),
                                "name": strategy.GetName(),
                        })
                }
                
                writeJSON(w, map[string]interface{}{
                        "strategies": strategyList,
                        "count":      len(strategyList),
                })
        default:
                http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        }
}

// handleBacktestData handles backtest data requests
func handleBacktestData(w http.ResponseWriter, r *http.Request, backtestEngine backtesting.BacktestEngine) {
        switch r.Method {
        case http.MethodGet:
                availableData := backtestEngine.GetAvailableData()
                writeJSON(w, map[string]interface{}{
                        "available_data": availableData,
                        "symbols":       len(availableData),
                })
        default:
                http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        }
}

// handleBacktestConfig handles backtest configuration requests
func handleBacktestConfig(w http.ResponseWriter, r *http.Request, backtestEngine backtesting.BacktestEngine) {
        switch r.Method {
        case http.MethodGet:
                config := backtestEngine.GetConfig()
                writeJSON(w, config)
        case http.MethodPost:
                var config backtesting.BacktestConfig
                if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
                        http.Error(w, fmt.Sprintf("Invalid config: %v", err), http.StatusBadRequest)
                        return
                }
                
                if err := backtestEngine.SetConfig(config); err != nil {
                        http.Error(w, fmt.Sprintf("Failed to set config: %v", err), http.StatusInternalServerError)
                        return
                }
                
                writeJSON(w, map[string]string{"status": "success"})
        default:
                http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        }
}

// handlePlugins handles plugin management requests
func handlePlugins(w http.ResponseWriter, r *http.Request, pluginManager plugins.PluginManager) {
        switch r.Method {
        case http.MethodGet:
                pluginMap := pluginManager.GetAllPlugins()
                pluginList := make([]*plugins.Plugin, 0, len(pluginMap))
                
                for _, plugin := range pluginMap {
                        pluginList = append(pluginList, plugin)
                }
                
                writeJSON(w, map[string]interface{}{
                        "plugins": pluginList,
                        "count":   len(pluginList),
                })
        case http.MethodPost:
                // Load a new plugin
                var request struct {
                        Path string `json:"path"`
                }
                
                if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
                        http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
                        return
                }
                
                plugin, err := pluginManager.LoadPlugin(request.Path)
                if err != nil {
                        http.Error(w, fmt.Sprintf("Failed to load plugin: %v", err), http.StatusInternalServerError)
                        return
                }
                
                writeJSON(w, plugin)
        default:
                http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        }
}

// handlePluginByID handles individual plugin requests
func handlePluginByID(w http.ResponseWriter, r *http.Request, pluginManager plugins.PluginManager) {
        // Extract plugin ID from URL path
        path := strings.TrimPrefix(r.URL.Path, "/api/v1/plugins/")
        if path == "" {
                http.Error(w, "Plugin ID required", http.StatusBadRequest)
                return
        }
        
        switch r.Method {
        case http.MethodGet:
                plugin, err := pluginManager.GetPlugin(path)
                if err != nil {
                        http.Error(w, fmt.Sprintf("Plugin not found: %v", err), http.StatusNotFound)
                        return
                }
                
                writeJSON(w, plugin)
        case http.MethodPost:
                // Start plugin
                if err := pluginManager.StartPlugin(path); err != nil {
                        http.Error(w, fmt.Sprintf("Failed to start plugin: %v", err), http.StatusInternalServerError)
                        return
                }
                
                writeJSON(w, map[string]string{"status": "started"})
        case http.MethodDelete:
                // Stop plugin
                if err := pluginManager.StopPlugin(path); err != nil {
                        http.Error(w, fmt.Sprintf("Failed to stop plugin: %v", err), http.StatusInternalServerError)
                        return
                }
                
                writeJSON(w, map[string]string{"status": "stopped"})
        case http.MethodPut:
                // Update plugin configuration
                var config plugins.PluginConfig
                if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
                        http.Error(w, fmt.Sprintf("Invalid config: %v", err), http.StatusBadRequest)
                        return
                }
                
                if err := pluginManager.UpdatePluginConfig(path, config); err != nil {
                        http.Error(w, fmt.Sprintf("Failed to update config: %v", err), http.StatusInternalServerError)
                        return
                }
                
                writeJSON(w, map[string]string{"status": "updated"})
        default:
                http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        }
}

// handlePluginDiscover handles plugin discovery requests
func handlePluginDiscover(w http.ResponseWriter, r *http.Request, pluginManager plugins.PluginManager) {
        switch r.Method {
        case http.MethodGet:
                directory := r.URL.Query().Get("directory")
                if directory == "" {
                        directory = "plugins" // Default directory
                }
                
                plugins, err := pluginManager.DiscoverPlugins(directory)
                if err != nil {
                        http.Error(w, fmt.Sprintf("Failed to discover plugins: %v", err), http.StatusInternalServerError)
                        return
                }
                
                writeJSON(w, map[string]interface{}{
                        "plugins": plugins,
                        "count":   len(plugins),
                        "directory": directory,
                })
        default:
                http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        }
}

// handlePluginHealth handles plugin health requests
func handlePluginHealth(w http.ResponseWriter, r *http.Request, pluginManager plugins.PluginManager) {
        switch r.Method {
        case http.MethodGet:
                pluginID := r.URL.Query().Get("plugin_id")
                
                if pluginID != "" {
                        // Get health for specific plugin
                        plugin, err := pluginManager.GetPlugin(pluginID)
                        if err != nil {
                                http.Error(w, fmt.Sprintf("Plugin not found: %v", err), http.StatusNotFound)
                                return
                        }
                        
                        health := &plugins.PluginHealth{
                                PluginID:    plugin.Info.ID,
                                Healthy:     plugin.State == plugins.PluginStateRunning,
                                Status:      string(plugin.State),
                                LastCheck:   time.Now(),
                                Uptime:      time.Since(plugin.LoadTime),
                                MemoryUsage: plugin.Metrics.MemoryUsage,
                                CPUUsage:    plugin.Metrics.CPUUsage,
                                ErrorCount:  plugin.Metrics.ErrorsCount,
                                LastError:   plugin.Error,
                        }
                        
                        writeJSON(w, health)
                } else {
                        // Get health for all plugins
                        pluginMap := pluginManager.GetAllPlugins()
                        healthMap := make(map[string]*plugins.PluginHealth)
                        
                        for _, plugin := range pluginMap {
                                health := &plugins.PluginHealth{
                                        PluginID:    plugin.Info.ID,
                                        Healthy:     plugin.State == plugins.PluginStateRunning,
                                        Status:      string(plugin.State),
                                        LastCheck:   time.Now(),
                                        Uptime:      time.Since(plugin.LoadTime),
                                        MemoryUsage: plugin.Metrics.MemoryUsage,
                                        CPUUsage:    plugin.Metrics.CPUUsage,
                                        ErrorCount:  plugin.Metrics.ErrorsCount,
                                        LastError:   plugin.Error,
                                }
                                healthMap[plugin.Info.ID] = health
                        }
                        
                        writeJSON(w, map[string]interface{}{
                                "plugins": healthMap,
                                "count":   len(healthMap),
                        })
                }
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