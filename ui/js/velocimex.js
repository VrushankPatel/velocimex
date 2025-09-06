// Enhanced Velocimex Web UI Application
import { WebSocketClient } from '../lib/websocket.js';

// Utility functions
function debounce(func, wait) {
    let timeout;
    return function(...args) {
        clearTimeout(timeout);
        timeout = setTimeout(() => func.apply(this, args), wait);
    };
}

function throttle(func, limit) {
    let inThrottle;
    return function(...args) {
        if (!inThrottle) {
            func.apply(this, args);
            inThrottle = true;
            setTimeout(() => inThrottle = false, limit);
        }
    };
}

// Toast notification system
class ToastManager {
    constructor() {
        this.container = document.getElementById('toast-container');
    }

    show(message, type = 'info', duration = 5000) {
        const toast = document.createElement('div');
        toast.className = `toast ${type} fade-in`;
        toast.innerHTML = `
            <div class="flex items-center">
                <i data-lucide="${this.getIcon(type)}" class="w-5 h-5 mr-2"></i>
                <span>${message}</span>
            </div>
        `;
        
        this.container.appendChild(toast);
        lucide.createIcons();
        
        setTimeout(() => {
            toast.style.animation = 'slideIn 0.3s ease-out reverse';
            setTimeout(() => toast.remove(), 300);
        }, duration);
    }

    getIcon(type) {
        const icons = {
            success: 'check-circle',
            error: 'x-circle',
            warning: 'alert-triangle',
            info: 'info'
        };
        return icons[type] || 'info';
    }
}

// Notification system
class NotificationManager {
    constructor() {
        this.panel = document.getElementById('notifications-panel');
        this.list = document.getElementById('notifications-list');
        this.badge = document.getElementById('notification-badge');
        this.notifications = [];
        this.maxNotifications = 50;
    }

    add(title, message, type = 'info', timestamp = new Date()) {
        const notification = {
            id: Date.now() + Math.random(),
            title,
            message,
            type,
            timestamp,
            read: false
        };

        this.notifications.unshift(notification);
        
        // Limit notifications
        if (this.notifications.length > this.maxNotifications) {
            this.notifications = this.notifications.slice(0, this.maxNotifications);
        }

        this.updateUI();
        this.updateBadge();
    }

    updateUI() {
        if (this.notifications.length === 0) {
            this.list.innerHTML = `
                <div class="text-center text-slate-500 dark:text-slate-400 text-sm py-4">
                    No notifications
                </div>
            `;
            return;
        }

        this.list.innerHTML = this.notifications.map(notification => `
            <div class="notification-item ${notification.read ? 'opacity-60' : ''}" data-id="${notification.id}">
                <i data-lucide="${this.getIcon(notification.type)}" class="notification-icon"></i>
                <div class="notification-content">
                    <div class="notification-title">${notification.title}</div>
                    <div class="notification-message">${notification.message}</div>
                    <div class="notification-time">${this.formatTime(notification.timestamp)}</div>
                </div>
            </div>
        `).join('');

        lucide.createIcons();
    }

    getIcon(type) {
        const icons = {
            success: 'check-circle',
            error: 'x-circle',
            warning: 'alert-triangle',
            info: 'info',
            trade: 'trending-up',
            arbitrage: 'zap'
        };
        return icons[type] || 'info';
    }

    formatTime(timestamp) {
        const now = new Date();
        const diff = now - timestamp;
        const minutes = Math.floor(diff / 60000);
        const hours = Math.floor(diff / 3600000);
        const days = Math.floor(diff / 86400000);

        if (minutes < 1) return 'Just now';
        if (minutes < 60) return `${minutes}m ago`;
        if (hours < 24) return `${hours}h ago`;
        return `${days}d ago`;
    }

    updateBadge() {
        const unreadCount = this.notifications.filter(n => !n.read).length;
        if (unreadCount > 0) {
            this.badge.textContent = unreadCount > 99 ? '99+' : unreadCount;
            this.badge.classList.remove('hidden');
        } else {
            this.badge.classList.add('hidden');
        }
    }

    clear() {
        this.notifications = [];
        this.updateUI();
        this.updateBadge();
    }

    markAllRead() {
        this.notifications.forEach(n => n.read = true);
        this.updateUI();
        this.updateBadge();
    }
}

// Chart manager for performance visualization
class ChartManager {
    constructor() {
        this.chart = null;
        this.data = {
            labels: [],
            datasets: [{
                label: 'P&L',
                data: [],
                borderColor: 'rgb(59, 130, 246)',
                backgroundColor: 'rgba(59, 130, 246, 0.1)',
                fill: true,
                tension: 0.4
            }]
        };
        this.init();
    }

    init() {
        const ctx = document.getElementById('performance-chart').getContext('2d');
        this.chart = new Chart(ctx, {
            type: 'line',
            data: this.data,
            options: {
                responsive: true,
                maintainAspectRatio: false,
                plugins: {
                    legend: {
                        display: false
                    },
                    tooltip: {
                        mode: 'index',
                        intersect: false,
                        backgroundColor: 'rgba(0, 0, 0, 0.8)',
                        titleColor: 'white',
                        bodyColor: 'white',
                        borderColor: 'rgba(255, 255, 255, 0.1)',
                        borderWidth: 1
                    }
                },
                scales: {
                    x: {
                        type: 'time',
                        time: {
                            unit: 'minute',
                            displayFormats: {
                                minute: 'HH:mm'
                            }
                        },
                        grid: {
                            color: 'rgba(148, 163, 184, 0.1)'
                        },
                        ticks: {
                            color: 'rgb(148, 163, 184)'
                        }
                    },
                    y: {
                        grid: {
                            color: 'rgba(148, 163, 184, 0.1)'
                        },
                        ticks: {
                            color: 'rgb(148, 163, 184)',
                            callback: function(value) {
                                return '$' + value.toFixed(2);
                            }
                        }
                    }
                },
                interaction: {
                    mode: 'nearest',
                    axis: 'x',
                    intersect: false
                },
                elements: {
                    point: {
                        radius: 0,
                        hoverRadius: 6
                    }
                }
            }
        });
    }

    updateData(newData) {
        this.data.labels = newData.labels;
        this.data.datasets[0].data = newData.values;
        this.chart.update('none');
    }

    addDataPoint(timestamp, value) {
        this.data.labels.push(timestamp);
        this.data.datasets[0].data.push(value);
        
        // Keep only last 100 points
        if (this.data.labels.length > 100) {
            this.data.labels.shift();
            this.data.datasets[0].data.shift();
        }
        
        this.chart.update('none');
    }
}

// Enhanced VelocimexApp class
export class VelocimexApp {
    constructor() {
        this.ws = null;
        this.eventHandlers = new Map();
        this.toastManager = new ToastManager();
        this.notificationManager = new NotificationManager();
        this.chartManager = new ChartManager();
        
        // UI elements
        this.connectionStatusEl = document.getElementById('connection-status');
        this.connectionIndicator = document.getElementById('connection-indicator');
        this.marketList = document.getElementById('market-list');
        this.arbitrageList = document.getElementById('arbitrage-list');
        this.asksContainer = document.getElementById('asks-container');
        this.bidsContainer = document.getElementById('bids-container');
        this.orderbookSpread = document.getElementById('orderbook-spread');
        this.marketSelect = document.getElementById('orderbook-symbol');
        this.signalsList = document.getElementById('signals-list');
        
        // Settings
        this.settings = {
            displayDepth: 10,
            updateInterval: 1000,
            theme: localStorage.getItem('theme') || 'light',
            chartType: 'line'
        };

        this.currentMarket = null;
        this.lastOrderbookData = null;
        this.marketData = new Map();
        this.arbitrageData = [];
        this.signals = [];

        this.init();
    }

    on(event, handler) {
        if (!this.eventHandlers.has(event)) {
            this.eventHandlers.set(event, new Set());
        }
        this.eventHandlers.get(event).add(handler);
    }

    emit(event, data) {
        const handlers = this.eventHandlers.get(event);
        if (handlers) {
            handlers.forEach(handler => {
                try {
                    handler(data);
                } catch (error) {
                    console.error(`Error in event handler for ${event}:`, error);
                }
            });
        }
    }

    init() {
        // Initialize WebSocket
        this.initWebSocket();
    }

    logToPage(msg) {
        // Only show debug log in development
        if (!window.location.hostname.includes('localhost') && !window.location.hostname.includes('127.0.0.1')) {
            return;
        }

        let el = document.getElementById('debug-log');
        if (!el) {
            el = document.createElement('div');
            el.id = 'debug-log';
            Object.assign(el.style, {
                position: 'fixed',
                bottom: '0',
                right: '0',
                background: 'rgba(0,0,0,0.5)',
                color: '#fff',
                fontSize: '10px',
                padding: '4px',
                zIndex: '9999',
                maxWidth: '300px',
                maxHeight: '150px',
                overflowY: 'auto',
                border: '1px solid rgba(255,255,255,0.1)',
                borderRadius: '4px 0 0 0',
                backdropFilter: 'blur(4px)',
                fontFamily: 'monospace'
            });
            document.body.appendChild(el);
        }
        
        // Keep only last 50 messages
        const lines = el.innerText.split('\n');
        if (lines.length > 50) {
            lines.shift();
        }
        lines.push(`${new Date().toLocaleTimeString()} | ${msg}`);
        el.innerText = lines.join('\n');
        el.scrollTop = el.scrollHeight;
    }

    updateConnectionStatus(isConnected, mode = null) {
        if (!this.connectionStatusEl) return;
        const [indicator, label] = this.connectionStatusEl.children;
        
        if (isConnected) {
            indicator.classList.remove('bg-gray-400', 'bg-red-500');
            indicator.classList.add('bg-green-500');
            label.textContent = `Connected${mode ? ` (${mode})` : ''}`;
        } else {
            indicator.classList.remove('bg-green-500', 'bg-gray-400');
            indicator.classList.add('bg-red-500');
            label.textContent = 'Disconnected';
        }
    }

    handleSystemStatus(status) {
        this.updateConnectionStatus(true, status.mode);
        this.emit('websocket:status', status);
    }

    subscribeToMarket(symbol) {
        if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
            console.warn('WebSocket not connected');
            return;
        }

        // Unsubscribe from previous market if any
        if (this.currentMarket && this.currentMarket !== symbol) {
            this.ws.send(JSON.stringify({
                type: 'unsubscribe',
                channel: 'orderbook',
                symbol: this.currentMarket
            }));
        }

        // Subscribe to new market
        this.ws.send(JSON.stringify({
            type: 'subscribe',
            channel: 'orderbook',
            symbol: symbol
        }));

        this.currentMarket = symbol;
    }

    initWebSocket() {
        try {
            const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
            const wsUrl = `${protocol}//${window.location.host}/ws`;
            this.logToPage('Connecting to WebSocket: ' + wsUrl);
            this.ws = new WebSocket(wsUrl);
            
            this.ws.onopen = () => {
                this.logToPage('WebSocket connected');
                this.updateConnectionStatus(true);
                this.emit('websocket:open');

                // Subscribe to system and orderbook channels
                this.ws.send(JSON.stringify({
                    type: 'subscribe',
                    channel: 'system'
                }));

                this.ws.send(JSON.stringify({
                    type: 'subscribe',
                    channel: 'orderbook'
                }));

                this.ws.send(JSON.stringify({
                    type: 'subscribe',
                    channel: 'arbitrage'
                }));

                // Load settings
                this.loadSettings();
            };

            this.ws.onclose = () => {
                this.logToPage('WebSocket disconnected');
                this.updateConnectionStatus(false);
                this.emit('websocket:close');
                setTimeout(() => this.initWebSocket(), 2000);
            };

            this.ws.onerror = (error) => {
                this.logToPage('WebSocket error: ' + error.message);
                this.updateConnectionStatus(false);
                this.emit('websocket:error', error);
            };

            this.ws.onmessage = (event) => {
                this.logToPage('WebSocket message: ' + event.data);
                try {
                    const message = JSON.parse(event.data);
                    this.handleWebSocketMessage(message);
                } catch (error) {
                    console.error('Error parsing WebSocket message:', error);
                    this.logToPage('Error parsing message: ' + error.message);
                }
            };
        } catch (error) {
            this.logToPage('Error initializing WebSocket: ' + error.message);
            this.updateConnectionStatus(false);
            setTimeout(() => this.initWebSocket(), 2000);
        }
    }

    handleWebSocketMessage(message) {
        // Validate message format
        if (!message) {
            console.warn('Received empty message');
            return;
        }

        // Check for message type or channel
        const messageType = message.type || message.channel;
        if (!messageType) {
            console.warn('Message missing type/channel:', message);
            return;
        }

        switch (messageType) {
            case 'status':
            case 'system':
                if (message.data?.type === 'symbols') {
                    this.updateMarketList(message.data.data);
                } else {
                    this.handleSystemStatus(message.data);
                }
                break;
                
            case 'orderbook':
                if (message.data && (message.data.bids || message.data.asks)) {
                    this.updateOrderBook(message.data);
                } else {
                    console.warn('Invalid orderbook data:', message.data);
                }
                break;
                
            case 'arbitrage':
                if (Array.isArray(message.data)) {
                    this.updateArbitrageOpportunities(message.data);
                } else {
                    console.warn('Invalid arbitrage data:', message.data);
                }
                break;
                
            case 'strategy':
                this.updateStrategyData(message.data);
                break;

            case 'symbols':
                if (Array.isArray(message.data)) {
                    this.updateMarketList(message.data);
                } else {
                    console.warn('Invalid symbols data:', message.data);
                }
                
            default:
                console.warn('Unhandled message type:', message.type);
        }
    }

    // Helper methods
    formatPrice(price) {
        return typeof price === 'number' ? price.toFixed(2) : '0.00';
    }

    formatVolume(volume) {
        if (typeof volume !== 'number') return '0.00';
        return volume >= 1000 ? (volume / 1000).toFixed(2) + 'K' : volume.toFixed(2);
    }

    formatCurrency(amount) {
        return typeof amount === 'number' ? '$' + amount.toFixed(2) : '$0.00';
    }

    setLoading(element, isLoading) {
        if (!element) return;

        if (isLoading) {
            if (!element.classList.contains('loading')) {
                element.classList.add('loading');
                element.setAttribute('data-original-content', element.innerHTML);
                element.innerHTML = this.loadingHTML;
            }
        } else {
            if (element.classList.contains('loading')) {
                const originalContent = element.getAttribute('data-original-content');
                if (originalContent) {
                    element.innerHTML = originalContent;
                }
                element.classList.remove('loading');
                element.removeAttribute('data-original-content');
            }
        }
    }

    updateOrderBook(data) {
        if (!data?.bids || !data?.asks) {
            console.error('[Debug] Invalid orderbook data:', data);
            return;
        }

        if (!this.asksContainer || !this.bidsContainer || !this.orderbookSpread) {
            console.error('[Debug] Missing UI elements for orderbook');
            return;
        }

        // Show loading state
        this.setLoading(this.asksContainer, true);
        this.setLoading(this.bidsContainer, true);

        console.log('[Debug] Updating orderbook:', {
            symbol: data.symbol,
            bids: data.bids.length,
            asks: data.asks.length
        });

        this.lastOrderbookData = data;

        const maxBidVolume = Math.max(...data.bids.map(bid => bid.volume));
        const maxAskVolume = Math.max(...data.asks.map(ask => ask.volume));

        // Update asks (reversed to show highest at top)
        const asksHTML = data.asks
            .slice(0, this.settings.displayDepth)
            .reverse()
            .map((ask, index) => {
                const volumePercentage = (ask.volume / maxAskVolume * 100).toFixed(0);
                return `
                    <div class="orderbook-row relative ${index === data.asks.length - 1 ? 'best-ask' : ''}">
                        <div class="ask">${this.formatPrice(ask.price)}</div>
                        <div>${this.formatVolume(ask.volume)}</div>
                        <div>${this.formatVolume(ask.price * ask.volume)}</div>
                        <div class="volume-bar volume-bar-ask" style="width: ${volumePercentage}%"></div>
                    </div>
                `;
            })
            .join('');

        // Update bids
        const bidsHTML = data.bids
            .slice(0, this.settings.displayDepth)
            .map((bid, index) => {
                const volumePercentage = (bid.volume / maxBidVolume * 100).toFixed(0);
                return `
                    <div class="orderbook-row relative ${index === 0 ? 'best-bid' : ''}">
                        <div class="bid">${this.formatPrice(bid.price)}</div>
                        <div>${this.formatVolume(bid.volume)}</div>
                        <div>${this.formatVolume(bid.price * bid.volume)}</div>
                        <div class="volume-bar volume-bar-bid" style="width: ${volumePercentage}%"></div>
                    </div>
                `;
            })
            .join('');

        // Calculate and update spread
        if (data.asks.length > 0 && data.bids.length > 0) {
            const bestAsk = data.asks[0].price;
            const bestBid = data.bids[0].price;
            const spread = bestAsk - bestBid;
            const spreadPercentage = (spread / bestAsk * 100).toFixed(4);

            this.orderbookSpread.innerHTML = `
                <span class="font-semibold">Spread:</span>
                <span class="spread-value">${this.formatPrice(spread)}</span>
                <span class="spread-percent">(${spreadPercentage}%)</span>
            `;
        }

        // Batch DOM updates
        requestAnimationFrame(() => {
            if (this.asksContainer.classList.contains('loading')) {
                this.asksContainer.classList.remove('loading');
                this.asksContainer.innerHTML = asksHTML;
            }
            
            if (this.bidsContainer.classList.contains('loading')) {
                this.bidsContainer.classList.remove('loading');
                this.bidsContainer.innerHTML = bidsHTML;
            }
        });
    }

    updateArbitrageOpportunities(opportunities) {
        if (!this.arbitrageList) return;
        
        // Show loading state
        this.setLoading(this.arbitrageList, true);
        
        // Process the update
        if (!Array.isArray(opportunities)) {
            console.error('[Debug] Invalid arbitrage opportunities data:', opportunities);
            return;
        }

        console.log('[Debug] Updating arbitrage opportunities:', opportunities.length);

        const html = opportunities
            .map(opp => {
                const profitClass = opp.profit >= 0 ? 'text-green-500' : 'text-red-500';
                return `
                    <div class="flex justify-between items-center py-2 border-b">
                        <div class="flex flex-col">
                            <span class="font-medium">${opp.symbol}</span>
                            <span class="text-xs text-gray-500">Vol: ${this.formatVolume(opp.volume)}</span>
                        </div>
                        <div class="flex flex-col items-end">
                            <span>${this.formatPrice(opp.price)}</span>
                            <span class="${profitClass} text-xs">
                                ${opp.profit > 0 ? '+' : ''}${(opp.profit * 100).toFixed(2)}%
                            </span>
                        </div>
                    </div>
                `;
            })
            .join('');

        requestAnimationFrame(() => {
            this.arbitrageList.innerHTML = html;
            this.arbitrageList.classList.remove('loading');
        });

        // Clear loading state
        this.setLoading(this.arbitrageList, false);
    }

    updateMarketList(symbols) {
        if (!Array.isArray(symbols)) {
            console.warn('Expected symbols data to be an array');
            return;
        }

        // Update market select dropdown once
        if (this.marketSelect) {
            const currentValue = this.marketSelect.value;
            this.marketSelect.innerHTML = '';

            symbols.forEach(symbol => {
                const option = document.createElement('option');
                option.value = symbol;
                option.textContent = symbol;
                this.marketSelect.appendChild(option);
            });

            // Restore previous selection or select first
            if (currentValue && symbols.includes(currentValue)) {
                this.marketSelect.value = currentValue;
            } else if (!this.currentMarket && this.marketSelect.children.length > 0) {
                this.currentMarket = this.marketSelect.value;
                this.subscribeToMarket(this.currentMarket);
            }
        }

        // Update market list display
        if (this.marketList) {
            this.setLoading(this.marketList, true);
            
            const html = symbols.map(symbol => `
                <div class="flex justify-between items-center py-2 border-b">
                    <div class="flex flex-col">
                        <span class="font-medium">${symbol}</span>
                        <span class="text-xs text-gray-500">Loading...</span>
                    </div>
                    <div class="flex flex-col items-end">
                        <span>--</span>
                        <span class="text-xs">--</span>
                    </div>
                </div>
            `).join('');

            requestAnimationFrame(() => {
                this.marketList.innerHTML = html;
                this.marketList.classList.remove('loading');
            });
        }
    }

    loadSettings() {
        const saved = localStorage.getItem('velocimex_settings');
        if (saved) {
            try {
                this.settings = { ...this.settings, ...JSON.parse(saved) };
                this.applySettings();
            } catch (e) {
                console.error('[Debug] Error loading settings:', e);
            }
        }
    }

    saveSettings(e) {
        e.preventDefault();
        
        const depthEl = document.getElementById('display-depth');
        const intervalEl = document.getElementById('update-interval');
        const themeEl = document.getElementById('theme-selector');
        
        if (depthEl && intervalEl && themeEl) {
            this.settings.displayDepth = parseInt(depthEl.value, 10);
            this.settings.updateInterval = parseInt(intervalEl.value, 10);
            this.settings.theme = themeEl.value;
            
            localStorage.setItem('velocimex_settings', JSON.stringify(this.settings));
            this.applySettings();
            
            if (this.settingsModal) {
                this.settingsModal.classList.add('hidden');
            }
        }
    }

    applySettings() {
        document.body.classList.toggle('dark-theme', this.settings.theme === 'dark');
        
        if (this.lastOrderbookData) {
            this.updateOrderBook(this.lastOrderbookData);
        }

        this.startUIUpdates();
    }

    initializeChart() {
        if (!this.performanceChart) return;
        
        // Determine if dark mode is active
        const isDarkMode = this.settings.theme === 'dark';
        const gridColor = isDarkMode ? 'rgba(255, 255, 255, 0.1)' : 'rgba(0, 0, 0, 0.1)';
        const textColor = isDarkMode ? '#9ca3af' : '#6b7280';
        
        const config = {
            type: 'line',
            data: {
                datasets: [
                    {
                        label: 'Profit/Loss',
                        data: [],
                        borderColor: 'rgb(34, 197, 94)',
                        backgroundColor: 'rgba(34, 197, 94, 0.1)',
                        fill: true,
                        tension: 0.4
                    },
                    {
                        label: 'Drawdown',
                        data: [],
                        borderColor: 'rgb(239, 68, 68)',
                        backgroundColor: 'rgba(239, 68, 68, 0.05)',
                        fill: true,
                        tension: 0.4,
                        hidden: true
                    }
                ]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                animation: false, // Disable animations for better performance
                interaction: {
                    intersect: false,
                    mode: 'index'
                },
                scales: {
                    x: {
                        type: 'time',
                        time: {
                            unit: 'minute',
                            tooltipFormat: 'HH:mm:ss'
                        },
                        grid: {
                            color: gridColor
                        },
                        ticks: {
                            color: textColor,
                            maxRotation: 0
                        }
                    },
                    y: {
                        grid: {
                            color: gridColor
                        },
                        ticks: {
                            color: textColor,
                            callback: (value) => this.formatCurrency(value)
                        }
                    }
                },
                plugins: {
                    tooltip: {
                        enabled: true,
                        mode: 'index',
                        intersect: false,
                        titleColor: textColor,
                        bodyColor: textColor,
                        backgroundColor: isDarkMode ? 'rgba(31, 41, 55, 0.8)' : 'rgba(255, 255, 255, 0.8)',
                        borderColor: isDarkMode ? 'rgba(75, 85, 99, 1)' : 'rgba(229, 231, 235, 1)',
                        callbacks: {
                            label: (context) => {
                                const value = context.parsed.y;
                                return `${context.dataset.label}: ${this.formatCurrency(value)}`;
                            }
                        }
                    },
                    legend: {
                        display: true,
                        position: 'top',
                        labels: {
                            color: textColor
                        }
                    }
                }
            }
        };
        
        // Destroy existing chart if it exists
        if (this.chartInstance) {
            this.chartInstance.destroy();
        }
        
        // Create new chart
        this.chartInstance = new Chart(this.performanceChart, config);
    }

    updateStrategyData(data) {
        if (!data) return;
        
        // Update signals list
        if (this.signalsList && data.signals?.length) {
            this.setLoading(this.signalsList, true);

            const signalsHtml = data.signals
                .slice(0, 5)
                .map(signal => {
                    const side = signal.side === 'buy' ? 'bid' : 'ask';
                    const timestamp = new Date(signal.timestamp);
                    
                    return `
                        <div class="border rounded p-3 mb-2">
                            <div class="flex justify-between">
                                <span class="font-medium">${signal.symbol} ${signal.side.toUpperCase()}</span>
                                <span class="${side}">${this.formatPrice(signal.price)}</span>
                            </div>
                            <div class="text-sm text-gray-600 mt-1">
                                ${signal.exchange} | Vol: ${this.formatVolume(signal.volume)}
                            </div>
                            <div class="text-xs text-gray-500 mt-1">
                                ${timestamp.toLocaleTimeString()}
                            </div>
                        </div>
                    `;
                })
                .join('');

            requestAnimationFrame(() => {
                this.setLoading(this.signalsList, false);
                this.signalsList.innerHTML = signalsHtml || 
                    '<div class="text-gray-500 text-center py-4">No recent signals</div>';
            });
        }

        // Update performance chart
        if (this.chartInstance && data.performance) {
            // Update profit/loss data
            const dataset = this.chartInstance.data.datasets[0];
            const drawdownDataset = this.chartInstance.data.datasets[1];
            
            if (dataset && data.performance.profitLoss !== undefined) {
                // Keep only last 100 points for performance
                if (dataset.data.length > 100) {
                    dataset.data.shift();
                }
                
                dataset.data.push({
                    x: new Date(),
                    y: data.performance.profitLoss
                });
                
                // Update dataset style based on overall performance
                const isPositive = data.performance.profitLoss >= 0;
                dataset.borderColor = isPositive ? 'rgb(34, 197, 94)' : 'rgb(239, 68, 68)';
                dataset.backgroundColor = isPositive ? 'rgba(34, 197, 94, 0.1)' : 'rgba(239, 68, 68, 0.1)';
            }
            
            // Update drawdown data
            if (drawdownDataset && data.performance.drawdown !== undefined) {
                if (drawdownDataset.data.length > 100) {
                    drawdownDataset.data.shift();
                }
                
                drawdownDataset.data.push({
                    x: new Date(),
                    y: data.performance.drawdown
                });
            }

            // Update chart without animation
            this.chartInstance.update('none');
        }
    }
}
