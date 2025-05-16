/**
 * WebSocketClient - A wrapper around the WebSocket API
 * Provides subscription management and automatic reconnection
 */
class WebSocketClient {
  constructor() {
    this.socket = null;
    this.isConnected = false;
    this.reconnectAttempts = 0;
    this.maxReconnectAttempts = 5;
    this.reconnectInterval = 2000; // 2 seconds
    this.subscriptions = {};
    this.debug = false;
    this.isSimulation = false;
    this.systemStatus = null;
    
    // Callbacks
    this.onOpenCallbacks = [];
    this.onCloseCallbacks = [];
    this.onErrorCallbacks = [];
    this.onMessageCallbacks = [];
  }
  
  log(...args) {
    if (this.debug) {
      console.log('[WebSocket]', ...args);
    }
  }
  
  error(...args) {
    if (this.debug) {
      console.error('[WebSocket]', ...args);
    }
  }
  
  /**
   * Connect to the WebSocket server
   */
  connect() {
    if (this.socket && (this.socket.readyState === WebSocket.CONNECTING || this.socket.readyState === WebSocket.OPEN)) {
      this.log('Already connected or connecting');
      return;
    }
    
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${protocol}//${window.location.host}/ws`;
    
    this.log(`Connecting to ${wsUrl}`);
    
    try {
      this.socket = new WebSocket(wsUrl);
      
      this.socket.onopen = () => {
        this.log('Connected');
        this.isConnected = true;
        this.reconnectAttempts = 0;
        
        // Request initial status immediately after connection
        this.send({
          type: 'subscribe',
          channel: 'system'
        });
        
        // Then resubscribe to other channels
        this.resubscribe();
        
        // Notify callbacks
        this.onOpenCallbacks.forEach(cb => cb());
      };
      
      this.socket.onclose = (event) => {
        this.log('Disconnected');
        this.isConnected = false;
        this.onCloseCallbacks.forEach(cb => cb(event));
        this.tryReconnect();
      };
      
      this.socket.onerror = (error) => {
        this.error('Connection error:', error);
        this.onErrorCallbacks.forEach(cb => cb(error));
      };
      
      this.socket.onmessage = (event) => {
        this.handleMessage(event);
      };
      
    } catch (error) {
      this.error('Failed to create connection:', error);
      this.isConnected = false;
      this.tryReconnect();
    }
  }
  
  handleMessage(event) {
    try {
      // Split multiple messages if any
      const messages = event.data.split('\n').filter(msg => msg.trim());
      
      messages.forEach(msgStr => {
        try {
          const message = JSON.parse(msgStr);
          
          // Handle system status messages
          if (message.type === 'status') {
            this.systemStatus = message.data;
            this.isSimulation = message.data.mode === 'simulation';
            this.log('System status updated:', {
              status: message.data.status,
              mode: message.data.mode,
              isSimulated: message.data.isSimulated
            });
          }

          // Call message handlers
          this.onMessageCallbacks.forEach(cb => {
            try {
              cb(message);
            } catch (handlerError) {
              this.error('Error in message handler:', handlerError);
            }
          });
        } catch (parseError) {
          this.error('Failed to parse message:', msgStr, parseError);
        }
      });
    } catch (error) {
      this.error('Error handling message:', error);
    }
  }
  
  /**
   * Disconnect from the WebSocket server
   */
  disconnect() {
    if (this.socket) {
      this.socket.close();
      this.socket = null;
      this.isConnected = false;
    }
  }
  
  /**
   * Try to reconnect to the WebSocket server
   */
  tryReconnect() {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      this.error('Max reconnection attempts reached');
      return;
    }
    
    this.reconnectAttempts++;
    this.log(`Reconnecting (${this.reconnectAttempts}/${this.maxReconnectAttempts})...`);
    
    setTimeout(() => this.connect(), this.reconnectInterval);
  }
  
  /**
   * Send a message to the WebSocket server
   * @param {object} message - The message to send
   * @returns {boolean} - Whether the send was successful
   */
  send(message) {
    if (!this.isConnected || !this.socket) {
      this.error('Cannot send, not connected');
      return false;
    }
    
    if (!message || typeof message !== 'object') {
      this.error('Invalid message format:', message);
      return false;
    }

    try {
      const msgStr = JSON.stringify(message);
      this.log('Sending:', {
        type: message.type,
        channel: message.channel,
        symbol: message.symbol,
        dataSize: msgStr.length,
        timestamp: new Date().toISOString()
      });
      this.socket.send(msgStr);
      return true;
    } catch (error) {
      this.error('Error sending message:', error);
      return false;
    }
  }
  
  /**
   * Subscribe to a channel with optional symbol filter
   * @param {string} channel - The channel to subscribe to
   * @param {string} [symbol] - Optional symbol filter
   * @returns {boolean} - Whether the subscription was successful
   */
  subscribe(channel, symbol) {
    if (!channel) {
      this.error('Channel is required for subscription');
      return false;
    }
    
    if (!this.subscriptions[channel]) {
      this.subscriptions[channel] = new Set();
    }
    
    if (symbol) {
      this.subscriptions[channel].add(symbol);
    }
    
    return this.send({
      type: 'subscribe',
      channel,
      symbol
    });
  }
  
  /**
   * Unsubscribe from a channel with optional symbol filter
   * @param {string} channel - The channel to unsubscribe from
   * @param {string} [symbol] - Optional symbol filter
   * @returns {boolean} - Whether the unsubscription was successful
   */
  unsubscribe(channel, symbol) {
    if (!channel) {
      this.error('Channel is required for unsubscription');
      return false;
    }
    
    if (this.subscriptions[channel]) {
      if (symbol) {
        this.subscriptions[channel].delete(symbol);
      } else {
        delete this.subscriptions[channel];
      }
    }
    
    return this.send({
      type: 'unsubscribe',
      channel,
      symbol
    });
  }
  
  /**
   * Resubscribe to all previously subscribed channels
   */
  resubscribe() {
    Object.entries(this.subscriptions).forEach(([channel, symbols]) => {
      if (symbols.size > 0) {
        symbols.forEach(symbol => this.subscribe(channel, symbol));
      } else {
        this.subscribe(channel);
      }
    });
  }
  
  /**
   * Register an onOpen callback
   * @param {function} callback - The callback to register
   */
  onOpen(callback) { this.onOpenCallbacks.push(callback); }
  
  /**
   * Register an onClose callback
   * @param {function} callback - The callback to register
   */
  onClose(callback) { this.onCloseCallbacks.push(callback); }
  
  /**
   * Register an onError callback
   * @param {function} callback - The callback to register
   */
  onError(callback) { this.onErrorCallbacks.push(callback); }
  
  /**
   * Register an onMessage callback
   * @param {function} callback - The callback to register
   */
  onMessage(callback) { this.onMessageCallbacks.push(callback); }
}

export { WebSocketClient };