// Enhanced Dashboard Component for Velocimex
// This provides a comprehensive trading dashboard with real-time updates

export class EnhancedDashboard {
    constructor(app) {
        this.app = app;
        this.components = {
            marketOverview: null,
            orderBook: null,
            arbitrage: null,
            signals: null,
            performance: null,
            riskMetrics: null
        };
        this.updateInterval = null;
        this.isVisible = true;
    }

    // Initialize all dashboard components
    init() {
        this.initMarketOverview();
        this.initOrderBook();
        this.initArbitrage();
        this.initSignals();
        this.initPerformance();
        this.initRiskMetrics();
        this.setupEventListeners();
        this.startAutoUpdate();
    }

    // Market Overview Component
    initMarketOverview() {
        this.components.marketOverview = {
            container: document.getElementById('market-list'),
            data: new Map(),
            render: (data) => {
                const markets = Array.from(data.values())
                    .sort((a, b) => b.changePercent - a.changePercent);

                this.components.marketOverview.container.innerHTML = markets.map(market => `
                    <div class="market-item ${this.app.currentMarket === market.symbol ? 'active' : ''}" 
                         data-symbol="${market.symbol}">
                        <div>
                            <div class="market-symbol">${market.symbol}</div>
                            <div class="text-xs text-slate-500 dark:text-slate-400">${market.exchange}</div>
                        </div>
                        <div class="text-right">
                            <div class="market-price">$${market.price.toFixed(2)}</div>
                            <div class="market-change ${market.changePercent >= 0 ? 'positive' : 'negative'}">
                                ${market.changePercent >= 0 ? '+' : ''}${market.changePercent.toFixed(2)}%
                            </div>
                        </div>
                    </div>
                `).join('');

                // Add click handlers
                this.components.marketOverview.container.querySelectorAll('.market-item').forEach(item => {
                    item.addEventListener('click', () => {
                        const symbol = item.dataset.symbol;
                        this.app.selectMarket(symbol);
                    });
                });
            }
        };
    }

    // Order Book Component
    initOrderBook() {
        this.components.orderBook = {
            asksContainer: document.getElementById('asks-container'),
            bidsContainer: document.getElementById('bids-container'),
            spreadElement: document.getElementById('orderbook-spread'),
            render: (data) => {
                const { asks, bids } = data;
                const depth = this.app.settings.displayDepth;

                // Render asks
                this.components.orderBook.asksContainer.innerHTML = asks.slice(0, depth).map((ask, index) => `
                    <div class="orderbook-row ${index === 0 ? 'best-ask' : ''}">
                        <div class="text-red-600 dark:text-red-400">${ask.price.toFixed(2)}</div>
                        <div>${ask.size.toFixed(4)}</div>
                        <div>${ask.total.toFixed(4)}</div>
                    </div>
                `).join('');

                // Render bids
                this.components.orderBook.bidsContainer.innerHTML = bids.slice(0, depth).map((bid, index) => `
                    <div class="orderbook-row ${index === 0 ? 'best-bid' : ''}">
                        <div class="text-green-600 dark:text-green-400">${bid.price.toFixed(2)}</div>
                        <div>${bid.size.toFixed(4)}</div>
                        <div>${bid.total.toFixed(4)}</div>
                    </div>
                `).join('');

                // Update spread
                if (asks.length > 0 && bids.length > 0) {
                    const spread = asks[0].price - bids[0].price;
                    const spreadPercent = (spread / bids[0].price) * 100;
                    this.components.orderBook.spreadElement.textContent = 
                        `Spread: $${spread.toFixed(2)} (${spreadPercent.toFixed(2)}%)`;
                }
            }
        };
    }

    // Arbitrage Component
    initArbitrage() {
        this.components.arbitrage = {
            container: document.getElementById('arbitrage-list'),
            countElement: document.getElementById('arbitrage-count'),
            data: [],
            render: (data) => {
                this.components.arbitrage.data = data;
                this.components.arbitrage.container.innerHTML = data.map(opp => `
                    <div class="arbitrage-item">
                        <div class="arbitrage-symbol">${opp.symbol}</div>
                        <div class="arbitrage-exchanges">${opp.buyExchange} → ${opp.sellExchange}</div>
                        <div class="flex justify-between items-center">
                            <div class="arbitrage-profit ${opp.profitPercent >= 0 ? 'positive' : 'negative'}">
                                ${opp.profitPercent.toFixed(2)}%
                            </div>
                            <div class="text-xs text-slate-500 dark:text-slate-400">
                                Vol: ${opp.maxVolume.toFixed(2)}
                            </div>
                        </div>
                    </div>
                `).join('');

                // Update count
                this.components.arbitrage.countElement.textContent = data.length;
            }
        };
    }

    // Signals Component
    initSignals() {
        this.components.signals = {
            container: document.getElementById('signals-list'),
            data: [],
            render: (data) => {
                this.components.signals.data = data;
                this.components.signals.container.innerHTML = data.map(signal => `
                    <div class="signal-item">
                        <div class="signal-icon ${signal.side.toLowerCase()}">
                            ${signal.side === 'BUY' ? 'B' : 'S'}
                        </div>
                        <div class="signal-content">
                            <div class="signal-symbol">${signal.symbol}</div>
                            <div class="signal-details">
                                ${signal.side} ${signal.quantity} @ $${signal.price}
                            </div>
                        </div>
                        <div class="signal-time">
                            ${this.formatTime(new Date(signal.timestamp))}
                        </div>
                    </div>
                `).join('');
            }
        };
    }

    // Performance Component
    initPerformance() {
        this.components.performance = {
            chart: this.app.chartManager,
            metrics: {
                totalPnl: document.getElementById('total-pnl'),
                winRate: document.getElementById('win-rate'),
                tradesToday: document.getElementById('trades-today'),
                activeStrategies: document.getElementById('active-strategies')
            },
            update: (data) => {
                // Update metrics
                if (data.totalPnl !== undefined) {
                    this.components.performance.metrics.totalPnl.textContent = `$${data.totalPnl.toFixed(2)}`;
                }
                if (data.winRate !== undefined) {
                    this.components.performance.metrics.winRate.textContent = `${data.winRate.toFixed(1)}%`;
                }
                if (data.tradesToday !== undefined) {
                    this.components.performance.metrics.tradesToday.textContent = data.tradesToday;
                }
                if (data.activeStrategies !== undefined) {
                    this.components.performance.metrics.activeStrategies.textContent = data.activeStrategies;
                }

                // Update chart
                if (data.timestamp && data.pnl !== undefined) {
                    this.components.performance.chart.addDataPoint(new Date(data.timestamp), data.pnl);
                }
            }
        };
    }

    // Risk Metrics Component
    initRiskMetrics() {
        this.components.riskMetrics = {
            maxDrawdown: document.getElementById('max-drawdown'),
            sharpeRatio: document.getElementById('sharpe-ratio'),
            var95: document.getElementById('var-95'),
            update: (data) => {
                if (data.maxDrawdown !== undefined) {
                    this.components.riskMetrics.maxDrawdown.textContent = `${data.maxDrawdown.toFixed(2)}%`;
                }
                if (data.sharpeRatio !== undefined) {
                    this.components.riskMetrics.sharpeRatio.textContent = data.sharpeRatio.toFixed(2);
                }
                if (data.var95 !== undefined) {
                    this.components.riskMetrics.var95.textContent = `$${data.var95.toFixed(2)}`;
                }
            }
        };
    }

    // Event Listeners
    setupEventListeners() {
        // Market selection
        this.app.on('market:update', (data) => {
            this.components.marketOverview.data.set(data.symbol, data);
            this.components.marketOverview.render(this.components.marketOverview.data);
        });

        // Order book updates
        this.app.on('orderbook:update', (data) => {
            if (data.symbol === this.app.currentMarket) {
                this.components.orderBook.render(data);
            }
        });

        // Arbitrage updates
        this.app.on('arbitrage:update', (data) => {
            this.components.arbitrage.render(data);
        });

        // Signal updates
        this.app.on('signal:new', (signal) => {
            this.components.signals.data.unshift(signal);
            if (this.components.signals.data.length > 20) {
                this.components.signals.data = this.components.signals.data.slice(0, 20);
            }
            this.components.signals.render(this.components.signals.data);
        });

        // Performance updates
        this.app.on('performance:update', (data) => {
            this.components.performance.update(data);
        });

        // Risk metrics updates
        this.app.on('performance:update', (data) => {
            this.components.riskMetrics.update(data);
        });
    }

    // Auto-update functionality
    startAutoUpdate() {
        this.updateInterval = setInterval(() => {
            if (this.isVisible) {
                this.refreshData();
            }
        }, this.app.settings.updateInterval);
    }

    stopAutoUpdate() {
        if (this.updateInterval) {
            clearInterval(this.updateInterval);
            this.updateInterval = null;
        }
    }

    // Refresh all data
    refreshData() {
        // This would typically make API calls to refresh data
        // For now, we'll just trigger a re-render of existing data
        this.components.marketOverview.render(this.components.marketOverview.data);
        this.components.arbitrage.render(this.components.arbitrage.data);
        this.components.signals.render(this.components.signals.data);
    }

    // Show/hide dashboard
    show() {
        this.isVisible = true;
        document.querySelector('main').style.display = 'block';
    }

    hide() {
        this.isVisible = false;
        document.querySelector('main').style.display = 'none';
    }

    // Utility functions
    formatTime(timestamp) {
        return new Intl.DateTimeFormat('en-US', {
            hour: '2-digit',
            minute: '2-digit',
            second: '2-digit'
        }).format(timestamp);
    }

    // Cleanup
    destroy() {
        this.stopAutoUpdate();
        // Remove event listeners if needed
    }
}

// Dashboard factory
export function createDashboard(app) {
    const dashboard = new EnhancedDashboard(app);
    dashboard.init();
    return dashboard;
}