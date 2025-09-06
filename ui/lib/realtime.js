// Real-time Data Manager for Velocimex
// Handles WebSocket connections, data streaming, and real-time updates

export class RealtimeDataManager {
    constructor(app) {
        this.app = app;
        this.ws = null;
        this.reconnectAttempts = 0;
        this.maxReconnectAttempts = 10;
        this.reconnectDelay = 1000;
        this.heartbeatInterval = null;
        this.subscriptions = new Set();
        this.messageQueue = [];
        this.isConnected = false;
        this.connectionUrl = this.getWebSocketUrl();
    }

    // Get WebSocket URL based on current location
    getWebSocketUrl() {
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const host = window.location.host;
        return `${protocol}//${host}/ws`;
    }

    // Connect to WebSocket
    connect() {
        try {
            this.ws = new WebSocket(this.connectionUrl);
            this.setupEventListeners();
        } catch (error) {
            console.error('Failed to create WebSocket connection:', error);
            this.handleConnectionError(error);
        }
    }

    // Setup WebSocket event listeners
    setupEventListeners() {
        this.ws.onopen = () => {
            console.log('WebSocket connected');
            this.isConnected = true;
            this.reconnectAttempts = 0;
            this.app.emit('websocket:open');
            this.startHeartbeat();
            this.processMessageQueue();
            this.resubscribe();
        };

        this.ws.onclose = (event) => {
            console.log('WebSocket disconnected:', event.code, event.reason);
            this.isConnected = false;
            this.stopHeartbeat();
            this.app.emit('websocket:close', event);
            
            if (!event.wasClean && this.reconnectAttempts < this.maxReconnectAttempts) {
                this.scheduleReconnect();
            }
        };

        this.ws.onerror = (error) => {
            console.error('WebSocket error:', error);
            this.app.emit('websocket:error', error);
        };

        this.ws.onmessage = (event) => {
            this.handleMessage(event.data);
        };
    }

    // Handle incoming messages
    handleMessage(data) {
        try {
            const message = JSON.parse(data);
            this.processMessage(message);
        } catch (error) {
            console.error('Error parsing WebSocket message:', error);
        }
    }

    // Process different types of messages
    processMessage(message) {
        switch (message.type) {
            case 'market_update':
                this.app.emit('market:update', message.data);
                break;
            case 'orderbook_update':
                this.app.emit('orderbook:update', message.data);
                break;
            case 'arbitrage_update':
                this.app.emit('arbitrage:update', message.data);
                break;
            case 'signal':
                this.app.emit('signal:new', message.data);
                break;
            case 'performance_update':
                this.app.emit('performance:update', message.data);
                break;
            case 'status':
                this.app.emit('websocket:status', message.data);
                break;
            case 'error':
                console.error('Server error:', message.error);
                this.app.toastManager.show(`Server error: ${message.error}`, 'error');
                break;
            case 'pong':
                // Heartbeat response
                break;
            default:
                console.log('Unknown message type:', message.type);
        }
    }

    // Send message to server
    send(message) {
        if (this.isConnected && this.ws.readyState === WebSocket.OPEN) {
            try {
                this.ws.send(JSON.stringify(message));
            } catch (error) {
                console.error('Error sending message:', error);
                this.messageQueue.push(message);
            }
        } else {
            this.messageQueue.push(message);
        }
    }

    // Process queued messages
    processMessageQueue() {
        while (this.messageQueue.length > 0) {
            const message = this.messageQueue.shift();
            this.send(message);
        }
    }

    // Subscribe to data channels
    subscribe(channels) {
        const message = {
            type: 'subscribe',
            channels: Array.isArray(channels) ? channels : [channels]
        };
        
        channels.forEach(channel => this.subscriptions.add(channel));
        this.send(message);
    }

    // Unsubscribe from data channels
    unsubscribe(channels) {
        const message = {
            type: 'unsubscribe',
            channels: Array.isArray(channels) ? channels : [channels]
        };
        
        channels.forEach(channel => this.subscriptions.delete(channel));
        this.send(message);
    }

    // Resubscribe to all channels after reconnection
    resubscribe() {
        if (this.subscriptions.size > 0) {
            this.subscribe(Array.from(this.subscriptions));
        }
    }

    // Subscribe to specific market data
    subscribeToMarket(symbol) {
        this.send({
            type: 'subscribe_market',
            symbol: symbol
        });
    }

    // Subscribe to order book for specific symbol
    subscribeToOrderBook(symbol) {
        this.send({
            type: 'subscribe_orderbook',
            symbol: symbol
        });
    }

    // Start heartbeat to keep connection alive
    startHeartbeat() {
        this.heartbeatInterval = setInterval(() => {
            if (this.isConnected) {
                this.send({ type: 'ping' });
            }
        }, 30000); // Send ping every 30 seconds
    }

    // Stop heartbeat
    stopHeartbeat() {
        if (this.heartbeatInterval) {
            clearInterval(this.heartbeatInterval);
            this.heartbeatInterval = null;
        }
    }

    // Schedule reconnection
    scheduleReconnect() {
        this.reconnectAttempts++;
        const delay = this.reconnectDelay * Math.pow(2, this.reconnectAttempts - 1);
        
        console.log(`Scheduling reconnection attempt ${this.reconnectAttempts} in ${delay}ms`);
        
        setTimeout(() => {
            if (this.reconnectAttempts <= this.maxReconnectAttempts) {
                this.connect();
            } else {
                console.error('Max reconnection attempts reached');
                this.app.toastManager.show('Connection failed. Please refresh the page.', 'error');
            }
        }, delay);
    }

    // Handle connection errors
    handleConnectionError(error) {
        console.error('Connection error:', error);
        this.app.emit('websocket:error', error);
    }

    // Disconnect WebSocket
    disconnect() {
        this.stopHeartbeat();
        if (this.ws) {
            this.ws.close(1000, 'Client disconnecting');
            this.ws = null;
        }
        this.isConnected = false;
        this.subscriptions.clear();
        this.messageQueue = [];
    }

    // Get connection status
    getConnectionStatus() {
        return {
            connected: this.isConnected,
            readyState: this.ws ? this.ws.readyState : WebSocket.CLOSED,
            subscriptions: Array.from(this.subscriptions),
            queuedMessages: this.messageQueue.length
        };
    }

    // Request historical data
    requestHistoricalData(symbol, timeframe, limit = 100) {
        this.send({
            type: 'request_historical',
            symbol: symbol,
            timeframe: timeframe,
            limit: limit
        });
    }

    // Request market data for specific symbols
    requestMarketData(symbols) {
        this.send({
            type: 'request_market_data',
            symbols: Array.isArray(symbols) ? symbols : [symbols]
        });
    }

    // Request arbitrage opportunities
    requestArbitrageOpportunities() {
        this.send({
            type: 'request_arbitrage'
        });
    }

    // Request performance data
    requestPerformanceData() {
        this.send({
            type: 'request_performance'
        });
    }

    // Request system status
    requestSystemStatus() {
        this.send({
            type: 'request_status'
        });
    }
}

// Data stream manager for handling different types of data streams
export class DataStreamManager {
    constructor(realtimeManager) {
        this.realtimeManager = realtimeManager;
        this.streams = new Map();
        this.buffers = new Map();
        this.processors = new Map();
    }

    // Register a data stream processor
    registerProcessor(streamType, processor) {
        this.processors.set(streamType, processor);
    }

    // Start processing a data stream
    startStream(streamType, config = {}) {
        if (this.streams.has(streamType)) {
            console.warn(`Stream ${streamType} is already active`);
            return;
        }

        const stream = {
            type: streamType,
            config: config,
            active: true,
            buffer: [],
            lastUpdate: null
        };

        this.streams.set(streamType, stream);
        this.buffers.set(streamType, []);

        // Subscribe to the stream
        this.realtimeManager.subscribe(streamType);
    }

    // Stop processing a data stream
    stopStream(streamType) {
        if (this.streams.has(streamType)) {
            this.streams.get(streamType).active = false;
            this.streams.delete(streamType);
            this.buffers.delete(streamType);
            this.realtimeManager.unsubscribe(streamType);
        }
    }

    // Process incoming data
    processData(streamType, data) {
        const stream = this.streams.get(streamType);
        if (!stream || !stream.active) {
            return;
        }

        const processor = this.processors.get(streamType);
        if (processor) {
            const processedData = processor(data);
            this.addToBuffer(streamType, processedData);
        } else {
            this.addToBuffer(streamType, data);
        }

        stream.lastUpdate = Date.now();
    }

    // Add data to buffer
    addToBuffer(streamType, data) {
        const buffer = this.buffers.get(streamType);
        if (buffer) {
            buffer.push({
                data: data,
                timestamp: Date.now()
            });

            // Limit buffer size
            const maxBufferSize = 1000;
            if (buffer.length > maxBufferSize) {
                buffer.splice(0, buffer.length - maxBufferSize);
            }
        }
    }

    // Get buffered data
    getBufferedData(streamType, limit = 100) {
        const buffer = this.buffers.get(streamType);
        if (!buffer) {
            return [];
        }

        return buffer.slice(-limit);
    }

    // Get stream status
    getStreamStatus(streamType) {
        const stream = this.streams.get(streamType);
        if (!stream) {
            return null;
        }

        return {
            active: stream.active,
            lastUpdate: stream.lastUpdate,
            bufferSize: this.buffers.get(streamType)?.length || 0,
            config: stream.config
        };
    }

    // Get all active streams
    getActiveStreams() {
        return Array.from(this.streams.keys());
    }
}

// Export factory function
export function createRealtimeManager(app) {
    return new RealtimeDataManager(app);
}

export function createDataStreamManager(realtimeManager) {
    return new DataStreamManager(realtimeManager);
}
