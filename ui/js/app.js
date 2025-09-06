// Enhanced Velocimex Web UI Application
import { VelocimexApp } from './velocimex.js';

// Main application initialization
document.addEventListener('DOMContentLoaded', () => {
    // Initialize app
    const app = new VelocimexApp();

    // Enhanced connection status handling
    app.on('websocket:open', () => {
        console.log('WebSocket connected');
        app.toastManager.show('Connected to Velocimex', 'success');
    });

    app.on('websocket:close', () => {
        console.log('WebSocket disconnected');
        app.toastManager.show('Disconnected from Velocimex', 'error');
    });

    app.on('websocket:error', (error) => {
        console.error('WebSocket error:', error);
        app.toastManager.show(`Connection error: ${error.message}`, 'error');
    });

    app.on('websocket:status', (status) => {
        console.log('System status:', status);
        app.notificationManager.add(
            'System Status Update',
            `Mode: ${status.mode || 'Unknown'}`,
            'info'
        );
    });

    // Market data updates
    app.on('market:update', (data) => {
        console.log('Market update:', data);
    });

    app.on('orderbook:update', (data) => {
        console.log('Order book update:', data);
    });

    app.on('arbitrage:update', (data) => {
        console.log('Arbitrage update:', data);
        if (data.length > 0) {
            app.notificationManager.add(
                'Arbitrage Opportunities',
                `${data.length} new opportunities found`,
                'arbitrage'
            );
        }
    });

    app.on('signal:new', (signal) => {
        console.log('New signal:', signal);
        app.notificationManager.add(
            'New Trading Signal',
            `${signal.side} ${signal.symbol} at $${signal.price}`,
            'trade'
        );
    });

    app.on('performance:update', (data) => {
        console.log('Performance update:', data);
    });

    // Expose app to window for debugging
    window.app = app;
    window.toastManager = app.toastManager;
    window.notificationManager = app.notificationManager;

    // Add some demo data for testing
    setTimeout(() => {
        if (app.marketData.size === 0) {
            // Add demo market data
            const demoMarkets = [
                { symbol: 'BTCUSDT', exchange: 'Binance', price: 43250.50, changePercent: 2.34 },
                { symbol: 'ETHUSDT', exchange: 'Binance', price: 2650.75, changePercent: -1.23 },
                { symbol: 'SOLUSDT', exchange: 'Binance', price: 98.45, changePercent: 5.67 },
                { symbol: 'AAPL', exchange: 'NASDAQ', price: 175.25, changePercent: 0.89 },
                { symbol: 'MSFT', exchange: 'NASDAQ', price: 378.90, changePercent: -0.45 }
            ];

            demoMarkets.forEach(market => {
                app.updateMarketData(market);
            });

            // Add demo arbitrage opportunities
            const demoArbitrage = [
                {
                    symbol: 'BTCUSDT',
                    buyExchange: 'Binance',
                    sellExchange: 'Coinbase',
                    profitPercent: 0.15,
                    maxVolume: 1.5
                },
                {
                    symbol: 'ETHUSDT',
                    buyExchange: 'Kraken',
                    sellExchange: 'Binance',
                    profitPercent: 0.08,
                    maxVolume: 5.2
                }
            ];

            app.updateArbitrage(demoArbitrage);

            // Add demo signals
            const demoSignals = [
                {
                    symbol: 'BTCUSDT',
                    side: 'BUY',
                    quantity: 0.5,
                    price: 43200,
                    timestamp: new Date().toISOString()
                },
                {
                    symbol: 'ETHUSDT',
                    side: 'SELL',
                    quantity: 2.0,
                    price: 2655,
                    timestamp: new Date(Date.now() - 300000).toISOString()
                }
            ];

            demoSignals.forEach(signal => {
                app.addSignal(signal);
            });

            // Add demo performance data
            const now = new Date();
            for (let i = 0; i < 20; i++) {
                const timestamp = new Date(now.getTime() - (19 - i) * 60000);
                const pnl = Math.sin(i * 0.3) * 100 + Math.random() * 50;
                app.chartManager.addDataPoint(timestamp, pnl);
            }

            app.toastManager.show('Demo data loaded', 'info');
        }
    }, 2000);
});