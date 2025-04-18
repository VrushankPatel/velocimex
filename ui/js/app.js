// Velocimex Web UI Application

// Initialize WebSocket client
const wsClient = new WebSocketClient();

// UI elements
const connectionStatus = document.getElementById('connection-status');
const settingsButton = document.getElementById('settings-button');
const settingsModal = document.getElementById('settings-modal');
const closeSettingsBtn = document.getElementById('close-settings');
const settingsForm = document.getElementById('settings-form');
const orderbookSymbol = document.getElementById('orderbook-symbol');
const orderbookSpread = document.getElementById('orderbook-spread');
const asksContainer = document.getElementById('asks-container');
const bidsContainer = document.getElementById('bids-container');
const marketList = document.getElementById('market-list');
const arbitrageList = document.getElementById('arbitrage-list');
const signalsList = document.getElementById('signals-list');
const performanceChart = document.getElementById('performance-chart').getContext('2d');

// Application state
let settings = {
    displayDepth: 10,
    updateInterval: 1000,
    theme: 'light'
};

let chartInstance = null;
let currentSymbol = 'BTCUSDT';
let lastOrderbookData = null;

// Initialize the application
function init() {
    // Load settings from local storage
    loadSettings();
    
    // Set up event listeners
    settingsButton.addEventListener('click', openSettings);
    closeSettingsBtn.addEventListener('click', closeSettings);
    settingsForm.addEventListener('submit', saveSettings);
    orderbookSymbol.addEventListener('change', () => {
        currentSymbol = orderbookSymbol.value;
        subscribeToBooksAndTrades();
    });
    
    // Connect to the WebSocket server
    connectToWebSocket();
    
    // Initialize the chart
    initializeChart();
    
    // Start the UI update loop
    setInterval(updateUI, settings.updateInterval);
}

// Connect to WebSocket server
function connectToWebSocket() {
    // Set up WebSocket connection and event handlers
    wsClient.onOpen(() => {
        // Check if we're in simulation mode by fetching the system status
        fetch('/api/v1/status')
            .then(response => response.json())
            .then(data => {
                if (data.isSimulated) {
                    // Yellow icon for simulation mode
                    connectionStatus.innerHTML = `
                        <span class="w-3 h-3 bg-yellow-500 rounded-full mr-2"></span>
                        <span title="Using simulated data as real market data is not available">Simulation</span>
                    `;
                } else {
                    // Green icon for connected to real market
                    connectionStatus.innerHTML = `
                        <span class="w-3 h-3 bg-green-500 rounded-full mr-2"></span>
                        <span>Connected</span>
                    `;
                }
            })
            .catch(error => {
                console.error('Error fetching system status:', error);
                // Default to Connected display if we can't fetch the status
                connectionStatus.innerHTML = `
                    <span class="w-3 h-3 bg-green-500 rounded-full mr-2"></span>
                    <span>Connected</span>
                `;
            });
        
        // Subscribe to channels
        subscribeToBooksAndTrades();
        wsClient.subscribe('arbitrage');
        wsClient.subscribe('strategy');
    });
    
    wsClient.onClose(() => {
        connectionStatus.innerHTML = `
            <span class="w-3 h-3 bg-red-500 rounded-full mr-2"></span>
            <span>Disconnected</span>
        `;
    });
    
    wsClient.onError((error) => {
        console.error('WebSocket error:', error);
    });
    
    wsClient.onMessage((message) => {
        handleWebSocketMessage(message);
    });
    
    // Connect to the server
    wsClient.connect();
}

// Subscribe to order books and trade feeds
function subscribeToBooksAndTrades() {
    // First unsubscribe from any previous symbol
    wsClient.unsubscribe('orderbook', currentSymbol);
    
    // Subscribe to new symbol
    wsClient.subscribe('orderbook', currentSymbol);
}

// Handle incoming WebSocket messages
function handleWebSocketMessage(message) {
    console.log('Received WebSocket message:', message);
    
    if (!message || !message.channel) {
        console.error('Invalid message format:', message);
        return;
    }
    
    switch (message.channel) {
        case 'orderbook':
            console.log('Updating orderbook with data:', message.data);
            updateOrderBook(message.data);
            break;
        case 'arbitrage':
            console.log('Updating arbitrage with data:', message.data);
            updateArbitrageOpportunities(message.data);
            break;
        case 'strategy':
            console.log('Updating strategy with data:', message.data);
            updateStrategyData(message.data);
            break;
        case 'system':
            console.log('Received system message:', message);
            if (message.type === 'symbols') {
                // Update symbols in the dropdown
                updateSymbolDropdown(message.data);
            }
            break;
        default:
            console.log('Unhandled message type:', message.type, 'on channel:', message.channel);
    }
}

// Update the order book display
function updateOrderBook(data) {
    if (!data || !data.bids || !data.asks) return;
    
    lastOrderbookData = data;
    
    // Calculate max volume for volume bar scaling
    const maxBidVolume = Math.max(...data.bids.map(bid => bid.volume));
    const maxAskVolume = Math.max(...data.asks.map(ask => ask.volume));
    
    // Update asks (reversed to show highest at top)
    const asksHTML = data.asks
        .slice(0, settings.displayDepth)
        .reverse()
        .map(ask => {
            const volumePercentage = (ask.volume / maxAskVolume * 100).toFixed(0);
            return `
                <div class="orderbook-row relative">
                    <div class="ask">${formatPrice(ask.price)}</div>
                    <div>${formatVolume(ask.volume)}</div>
                    <div>${formatVolume(ask.price * ask.volume)}</div>
                    <div class="volume-bar volume-bar-ask" style="width: ${volumePercentage}%"></div>
                </div>
            `;
        })
        .join('');
    
    // Update bids
    const bidsHTML = data.bids
        .slice(0, settings.displayDepth)
        .map(bid => {
            const volumePercentage = (bid.volume / maxBidVolume * 100).toFixed(0);
            return `
                <div class="orderbook-row relative">
                    <div class="bid">${formatPrice(bid.price)}</div>
                    <div>${formatVolume(bid.volume)}</div>
                    <div>${formatVolume(bid.price * bid.volume)}</div>
                    <div class="volume-bar volume-bar-bid" style="width: ${volumePercentage}%"></div>
                </div>
            `;
        })
        .join('');
    
    // Update the containers
    asksContainer.innerHTML = asksHTML;
    bidsContainer.innerHTML = bidsHTML;
    
    // Update spread
    if (data.asks.length > 0 && data.bids.length > 0) {
        const bestAsk = data.asks[0].price;
        const bestBid = data.bids[0].price;
        const spread = bestAsk - bestBid;
        const spreadPercentage = (spread / bestAsk * 100).toFixed(2);
        
        orderbookSpread.textContent = `Spread: ${formatPrice(spread)} (${spreadPercentage}%)`;
    }
}

// Update arbitrage opportunities display
function updateArbitrageOpportunities(opportunities) {
    if (!opportunities || !opportunities.length) {
        arbitrageList.innerHTML = '<div class="text-gray-500 text-center py-8">No arbitrage opportunities found</div>';
        return;
    }
    
    const html = opportunities
        .slice(0, 5) // Show top 5 opportunities
        .map(opp => {
            const profitColor = opp.profitPercent >= 0.5 ? 'text-green-600' : opp.profitPercent >= 0.2 ? 'text-green-500' : 'text-gray-600';
            
            return `
                <div class="border rounded p-3 relative overflow-hidden">
                    <div class="font-medium flex justify-between">
                        <span>${opp.symbol}</span>
                        <span class="${profitColor} font-bold">${opp.profitPercent.toFixed(2)}%</span>
                    </div>
                    <div class="grid grid-cols-2 gap-2 mt-2">
                        <div>
                            <div class="text-xs text-gray-500">Buy at ${opp.buyExchange}</div>
                            <div class="bid font-medium">${formatPrice(opp.buyPrice)}</div>
                        </div>
                        <div>
                            <div class="text-xs text-gray-500">Sell at ${opp.sellExchange}</div>
                            <div class="ask font-medium">${formatPrice(opp.sellPrice)}</div>
                        </div>
                    </div>
                    <div class="text-xs text-gray-500 mt-1">
                        Est. Profit: ${formatCurrency(opp.estimatedProfit)} | Latency: ${opp.latencyEstimate}ms
                    </div>
                    ${opp.isValid ? 
                        `<div class="absolute bottom-0 right-0 px-2 py-1 bg-green-100 text-green-800 text-xs rounded-tl">Valid</div>` : 
                        `<div class="absolute bottom-0 right-0 px-2 py-1 bg-red-100 text-red-800 text-xs rounded-tl">Invalid</div>`
                    }
                </div>
            `;
        })
        .join('');
    
    arbitrageList.innerHTML = html;
}

// Update strategy performance data
function updateStrategyData(data) {
    if (!data) return;
    
    // Update recent signals
    if (data.recentSignals && data.recentSignals.length) {
        const signalsHtml = data.recentSignals
            .slice(0, 5)
            .map(signal => {
                const side = signal.side === 'buy' ? 'bid' : 'ask';
                
                return `
                    <div class="border rounded p-3">
                        <div class="flex justify-between">
                            <span class="font-medium">${signal.symbol} ${signal.side.toUpperCase()}</span>
                            <span class="${side}">${formatPrice(signal.price)}</span>
                        </div>
                        <div class="text-sm text-gray-600 mt-1">${signal.exchange} | Vol: ${formatVolume(signal.volume)}</div>
                        <div class="text-xs text-gray-500 mt-1">${formatTime(new Date(signal.timestamp))}</div>
                    </div>
                `;
            })
            .join('');
        
        signalsList.innerHTML = signalsHtml;
    }
    
    // Update chart data
    updatePerformanceChart(data);
}

// Initialize the performance chart
function initializeChart() {
    // Determine if dark mode is active
    const isDarkMode = settings.theme === 'dark';
    
    // Set colors based on theme
    const gridColor = isDarkMode ? 'rgba(255, 255, 255, 0.1)' : 'rgba(0, 0, 0, 0.1)';
    const textColor = isDarkMode ? '#9ca3af' : '#6b7280';
    
    const datasets = [
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
    ];
    
    const options = {
        responsive: true,
        maintainAspectRatio: false,
        scales: {
            x: {
                type: 'time',
                time: {
                    unit: 'minute'
                },
                grid: {
                    color: gridColor
                },
                ticks: {
                    color: textColor
                }
            },
            y: {
                title: {
                    display: true,
                    text: 'Profit/Loss',
                    color: textColor
                },
                grid: {
                    color: gridColor
                },
                ticks: {
                    color: textColor
                }
            }
        },
        plugins: {
            tooltip: {
                mode: 'index',
                intersect: false,
                titleColor: textColor,
                bodyColor: textColor,
                backgroundColor: isDarkMode ? 'rgba(31, 41, 55, 0.8)' : 'rgba(255, 255, 255, 0.8)',
                borderColor: isDarkMode ? 'rgba(75, 85, 99, 1)' : 'rgba(229, 231, 235, 1)'
            },
            legend: {
                position: 'top',
                labels: {
                    color: textColor
                }
            }
        }
    };
    
    chartInstance = new Chart(performanceChart, {
        type: 'line',
        data: {
            datasets: datasets
        },
        options: options
    });
}

// Update the performance chart with new data
function updatePerformanceChart(data) {
    // This is a simplified implementation - in a real app, we'd store
    // historical data and update the chart properly
    if (!chartInstance || !chartInstance.data || !chartInstance.data.datasets) return;
    
    // Add a new data point
    const dataset = chartInstance.data.datasets[0];
    if (dataset.data.length > 20) {
        dataset.data.shift(); // Remove oldest point if we have too many
    }
    
    dataset.data.push({
        x: new Date(),
        y: data.profitLoss || 0
    });
    
    chartInstance.update();
}

// Update the symbol dropdown
function updateSymbolDropdown(symbols) {
    if (!symbols || !symbols.length) return;
    
    // Clear current options
    orderbookSymbol.innerHTML = '';
    
    // Add new options
    symbols.forEach(symbol => {
        const option = document.createElement('option');
        option.value = symbol;
        option.textContent = symbol.replace(/([A-Z]+)([A-Z][a-z])/, '$1/$2');
        orderbookSymbol.appendChild(option);
    });
    
    // Set current symbol
    orderbookSymbol.value = currentSymbol;
}

// Update the market list
function updateUI() {
    // Fetch latest market data via REST API
    fetch('/api/v1/markets')
        .then(response => response.json())
        .then(data => {
            if (data && data.markets) {
                updateMarketList(data.markets);
            }
        })
        .catch(error => console.error('Error fetching market data:', error));
}

// Update the market list display
function updateMarketList(markets) {
    if (!markets || !markets.length) return;
    
    const html = markets
        .map(market => {
            return `
                <div class="flex justify-between items-center py-2 border-b">
                    <span class="font-medium">${market.symbol}</span>
                    <span>${formatPrice(market.price)}</span>
                </div>
            `;
        })
        .join('');
    
    marketList.innerHTML = html;
}

// Open settings modal
function openSettings() {
    settingsModal.classList.remove('hidden');
}

// Close settings modal
function closeSettings() {
    settingsModal.classList.add('hidden');
}

// Save settings
function saveSettings(e) {
    e.preventDefault();
    
    // Get values from form
    settings.displayDepth = parseInt(document.getElementById('display-depth').value, 10);
    settings.updateInterval = parseInt(document.getElementById('update-interval').value, 10);
    settings.theme = document.getElementById('theme-selector').value;
    
    // Save to local storage
    localStorage.setItem('velocimex_settings', JSON.stringify(settings));
    
    // Apply settings
    applySettings();
    
    // Close modal
    closeSettings();
}

// Load settings from local storage
function loadSettings() {
    const savedSettings = localStorage.getItem('velocimex_settings');
    if (savedSettings) {
        settings = JSON.parse(savedSettings);
    }
    
    // Apply to form elements
    document.getElementById('display-depth').value = settings.displayDepth;
    document.getElementById('update-interval').value = settings.updateInterval;
    document.getElementById('theme-selector').value = settings.theme;
    
    // Apply settings
    applySettings();
}

// Apply current settings
function applySettings() {
    // Apply theme
    if (settings.theme === 'dark') {
        document.body.classList.add('dark-theme');
        
        // Add border styling to panels for better visual separation in dark mode
        const panels = document.querySelectorAll('.rounded-lg.shadow-md');
        panels.forEach(panel => {
            panel.style.border = '1px solid rgba(255, 255, 255, 0.075)';
            panel.style.boxShadow = '0 4px 10px rgba(0, 0, 0, 0.25), inset 0 1px 0 rgba(255, 255, 255, 0.05)';
            panel.style.backgroundColor = '#1F2937';
        });
        
        // Improve header styling
        const headers = document.querySelectorAll('.rounded-lg.shadow-md h2');
        headers.forEach(header => {
            header.style.borderBottom = '1px solid rgba(255, 255, 255, 0.05)';
            header.style.paddingBottom = '0.75rem';
            header.style.marginBottom = '1rem';
            header.style.color = '#E5E7EB';
        });
        
        // Update chart colors for dark theme
        if (chartInstance) {
            chartInstance.options.scales.x.grid.color = 'rgba(255, 255, 255, 0.1)';
            chartInstance.options.scales.y.grid.color = 'rgba(255, 255, 255, 0.1)';
            chartInstance.options.scales.x.ticks.color = '#9ca3af';
            chartInstance.options.scales.y.ticks.color = '#9ca3af';
            chartInstance.update();
        }
        
        // Update status and styling
        console.log('Dark theme applied');
    } else {
        document.body.classList.remove('dark-theme');
        
        // Reset panel styling
        const panels = document.querySelectorAll('.rounded-lg.shadow-md');
        panels.forEach(panel => {
            panel.style.border = '';
            panel.style.boxShadow = '';
            panel.style.backgroundColor = '';
        });
        
        // Reset header styling
        const headers = document.querySelectorAll('.rounded-lg.shadow-md h2');
        headers.forEach(header => {
            header.style.borderBottom = '';
            header.style.paddingBottom = '';
            header.style.marginBottom = '';
            header.style.color = '';
        });
        
        // Update chart colors for light theme
        if (chartInstance) {
            chartInstance.options.scales.x.grid.color = 'rgba(0, 0, 0, 0.1)';
            chartInstance.options.scales.y.grid.color = 'rgba(0, 0, 0, 0.1)';
            chartInstance.options.scales.x.ticks.color = '#6b7280';
            chartInstance.options.scales.y.ticks.color = '#6b7280';
            chartInstance.update();
        }
        
        // Update status and styling
        console.log('Light theme applied');
    }
    
    // If we have orderbook data, update it with new depth
    if (lastOrderbookData) {
        updateOrderBook(lastOrderbookData);
    }
    
    // Update UI refresh rate
    clearInterval(window.uiUpdateInterval);
    window.uiUpdateInterval = setInterval(updateUI, settings.updateInterval);
}

// Format a price value
function formatPrice(price) {
    if (typeof price !== 'number') return '0.00';
    return price.toFixed(2);
}

// Format a volume value
function formatVolume(volume) {
    if (typeof volume !== 'number') return '0.00';
    
    if (volume >= 1000) {
        return (volume / 1000).toFixed(2) + 'K';
    }
    
    return volume.toFixed(2);
}

// Format a currency value
function formatCurrency(amount) {
    if (typeof amount !== 'number') return '$0.00';
    
    return '$' + amount.toFixed(2);
}

// Format a timestamp
function formatTime(timestamp) {
    if (!timestamp) return '';
    
    const hours = timestamp.getHours().toString().padStart(2, '0');
    const minutes = timestamp.getMinutes().toString().padStart(2, '0');
    const seconds = timestamp.getSeconds().toString().padStart(2, '0');
    
    return `${hours}:${minutes}:${seconds}`;
}

// Initialize the application when the DOM is loaded
document.addEventListener('DOMContentLoaded', init);