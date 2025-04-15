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
    this.reconnectInterval = 3000; // 3 seconds
    this.subscriptions = {};
    
    // Callbacks
    this.onOpenCallbacks = [];
    this.onCloseCallbacks = [];
    this.onErrorCallbacks = [];
    this.onMessageCallbacks = [];
  }
  
  /**
   * Connect to the WebSocket server
   */
  connect() {
    if (this.socket && (this.socket.readyState === WebSocket.CONNECTING || this.socket.readyState === WebSocket.OPEN)) {
      console.log('WebSocket already connected or connecting');
      return;
    }
    
    // Determine WebSocket URL - always use the same host/port as the current page
    // This solves the issue with two servers running on different ports
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${protocol}//${window.location.host}/ws`;
    
    console.log(`Connecting to WebSocket at ${wsUrl}`);
    
    try {
      this.socket = new WebSocket(wsUrl);
    } catch (error) {
      console.error('Error creating WebSocket connection:', error);
      this.tryReconnect();
      return;
    }
    
    this.socket.onopen = (event) => {
      console.log('WebSocket connected');
      this.isConnected = true;
      this.reconnectAttempts = 0;
      
      // Resubscribe to all channels
      this.resubscribe();
      
      // Call onOpen callbacks
      this.onOpenCallbacks.forEach(callback => callback(event));
    };
    
    this.socket.onclose = (event) => {
      console.log('WebSocket disconnected', event);
      this.isConnected = false;
      
      // Call onClose callbacks
      this.onCloseCallbacks.forEach(callback => callback(event));
      
      // Try to reconnect
      this.tryReconnect();
    };
    
    this.socket.onerror = (error) => {
      console.error('WebSocket error', error);
      
      // Call onError callbacks
      this.onErrorCallbacks.forEach(callback => callback(error));
    };
    
    this.socket.onmessage = (event) => {
      // Try to parse multiple JSON messages if they were received together
      const messages = event.data.split('\n').filter(msg => msg.trim() !== '');
      
      for (const msgStr of messages) {
        let message;
        try {
          // First, try to parse as regular JSON
          message = JSON.parse(msgStr);
          console.log('Received WebSocket message:', message);
          
          // Call onMessage callbacks with the successfully parsed message
          this.onMessageCallbacks.forEach(callback => callback(message));
        } catch (error) {
          console.error('Error parsing WebSocket message:', error);
          console.log('Problematic message:', msgStr);
          
          // Try to salvage the message if it's malformed but still valid JSON with some cleanup
          try {
            const cleanedMsg = msgStr.trim()
              .replace(/,\s*}/, '}')  // Remove trailing commas before closing braces
              .replace(/,\s*]/, ']')  // Remove trailing commas before closing brackets
              .replace(/\n/g, '\\n'); // Escape newlines
              
            message = JSON.parse(cleanedMsg);
            console.log('Salvaged WebSocket message after cleanup:', message);
            
            // Call onMessage callbacks with the salvaged message
            this.onMessageCallbacks.forEach(callback => callback(message));
          } catch (secondError) {
            // If we still can't parse it, log and ignore this message
            console.error('Failed to salvage malformed message:', secondError);
          }
        }
      }
    };
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
      console.log('Maximum reconnect attempts reached');
      return;
    }
    
    this.reconnectAttempts++;
    console.log(`Attempting to reconnect (${this.reconnectAttempts}/${this.maxReconnectAttempts})...`);
    
    setTimeout(() => {
      this.connect();
    }, this.reconnectInterval);
  }
  
  /**
   * Resubscribe to all previously subscribed channels
   */
  resubscribe() {
    if (!this.isConnected) {
      return;
    }
    
    Object.keys(this.subscriptions).forEach(channel => {
      const symbols = this.subscriptions[channel];
      if (symbols.length > 0) {
        symbols.forEach(symbol => {
          this.sendSubscription(channel, symbol);
        });
      } else {
        this.sendSubscription(channel);
      }
    });
  }
  
  /**
   * Send a subscription message
   * @param {string} channel - The channel to subscribe to
   * @param {string} symbol - Optional symbol to subscribe to
   */
  sendSubscription(channel, symbol = null) {
    if (!this.isConnected) {
      return;
    }
    
    const message = {
      type: 'subscribe',
      channel: channel
    };
    
    if (symbol) {
      message.symbol = symbol;
    }
    
    this.send(message);
  }
  
  /**
   * Subscribe to a channel
   * @param {string} channel - The channel to subscribe to
   * @param {string} symbol - Optional symbol to subscribe to
   */
  subscribe(channel, symbol = null) {
    // Initialize channel if it doesn't exist
    if (!this.subscriptions[channel]) {
      this.subscriptions[channel] = [];
    }
    
    // Add symbol to subscription if provided
    if (symbol && !this.subscriptions[channel].includes(symbol)) {
      this.subscriptions[channel].push(symbol);
    }
    
    // Send subscription message if connected
    this.sendSubscription(channel, symbol);
  }
  
  /**
   * Unsubscribe from a channel
   * @param {string} channel - The channel to unsubscribe from
   * @param {string} symbol - Optional symbol to unsubscribe from
   */
  unsubscribe(channel, symbol = null) {
    if (!this.subscriptions[channel]) {
      return;
    }
    
    if (symbol) {
      // Remove symbol from subscription
      const index = this.subscriptions[channel].indexOf(symbol);
      if (index !== -1) {
        this.subscriptions[channel].splice(index, 1);
      }
    } else {
      // Remove entire channel subscription
      delete this.subscriptions[channel];
    }
    
    // Send unsubscription message if connected
    if (this.isConnected) {
      const message = {
        type: 'unsubscribe',
        channel: channel
      };
      
      if (symbol) {
        message.symbol = symbol;
      }
      
      this.send(message);
    }
  }
  
  /**
   * Send a message to the WebSocket server
   * @param {object} message - The message to send
   */
  send(message) {
    if (!this.isConnected) {
      console.warn('Cannot send message, WebSocket is not connected');
      return;
    }
    
    this.socket.send(JSON.stringify(message));
  }
  
  /**
   * Register an onOpen callback
   * @param {function} callback - The callback to register
   */
  onOpen(callback) {
    this.onOpenCallbacks.push(callback);
  }
  
  /**
   * Register an onClose callback
   * @param {function} callback - The callback to register
   */
  onClose(callback) {
    this.onCloseCallbacks.push(callback);
  }
  
  /**
   * Register an onError callback
   * @param {function} callback - The callback to register
   */
  onError(callback) {
    this.onErrorCallbacks.push(callback);
  }
  
  /**
   * Register an onMessage callback
   * @param {function} callback - The callback to register
   */
  onMessage(callback) {
    this.onMessageCallbacks.push(callback);
  }
}