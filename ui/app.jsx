// Main application component
const App = () => {
  const [activeTab, setActiveTab] = React.useState('dashboard');
  const [theme, setTheme] = React.useState('light');
  const [websocket, setWebsocket] = React.useState(null);
  const [connected, setConnected] = React.useState(false);
  const [orderBooks, setOrderBooks] = React.useState({});
  const [arbitrageOpportunities, setArbitrageOpportunities] = React.useState([]);
  const [strategies, setStrategies] = React.useState([]);
  const [symbols, setSymbols] = React.useState([]);
  const [selectedSymbol, setSelectedSymbol] = React.useState('');
  const [isSimulation, setIsSimulation] = React.useState(false);

  // Initialize WebSocket connection on component mount
  React.useEffect(() => {
    // Create WebSocket connection
    const ws = new WebSocketClient();
    ws.debug = true; // Enable debug logging
    
    // Define WebSocket event handlers
    ws.onOpen(() => {
      console.log('WebSocket connected');
      setConnected(true);
      
      // Subscribe to channels
      ws.subscribe('orderbook');
      ws.subscribe('arbitrage');
      ws.subscribe('strategy');
    });
    
    ws.onClose(() => {
      console.log('WebSocket disconnected');
      setConnected(false);
    });
    
    ws.onMessage((message) => {
      handleWebSocketMessage(message);
    });
    
    // Connect to WebSocket server
    ws.connect();
    setWebsocket(ws);
    
    // Cleanup WebSocket connection on unmount
    return () => {
      if (ws) {
        ws.disconnect();
      }
    };
  }, []);
  
  // Handle theme change
  const toggleTheme = () => {
    const newTheme = theme === 'light' ? 'dark' : 'light';
    setTheme(newTheme);
    
    // Apply theme to body
    if (newTheme === 'dark') {
      document.body.classList.add('dark');
    } else {
      document.body.classList.remove('dark');
    }
  };
  
  // Handle WebSocket messages
  const handleWebSocketMessage = (message) => {
    switch (message.type) {
      case 'status':
        setIsSimulation(message.data.mode === 'simulation');
        // Update connection status with mode
        const statusElement = document.getElementById('connection-status');
        if (statusElement) {
          const [indicator, label] = statusElement.children;
          if (connected) {
            indicator.classList.remove('bg-gray-400', 'bg-red-500');
            indicator.classList.add('bg-green-500');
            label.textContent = `Connected (${message.data.mode})`;
          } else {
            indicator.classList.remove('bg-green-500', 'bg-gray-400');
            indicator.classList.add('bg-red-500');
            label.textContent = 'Disconnected';
          }
        }
        break;

      case 'symbols':
        setSymbols(message.data);
        if (message.data.length > 0 && !selectedSymbol) {
          setSelectedSymbol(message.data[0]);
          if (websocket) {
            websocket.subscribe('orderbook', message.data[0]);
          }
        }
        break;
        
      case 'strategies':
        setStrategies(message.data);
        break;
        
      case 'snapshot':
        if (message.channel === 'orderbook') {
          setOrderBooks((prevBooks) => ({
            ...prevBooks,
            [message.symbol]: message.data
          }));
        } else if (message.channel === 'arbitrage') {
          setArbitrageOpportunities(message.data);
        }
        break;
        
      case 'update':
        if (message.channel === 'orderbook') {
          setOrderBooks((prevBooks) => ({
            ...prevBooks,
            [message.symbol]: message.data
          }));
        } else if (message.channel === 'arbitrage') {
          setArbitrageOpportunities(message.data);
        } else if (message.channel === 'strategy') {
          // Update strategy information
          setStrategies((prevStrategies) => {
            const updatedStrategies = [...prevStrategies];
            const index = updatedStrategies.findIndex(s => s.name === message.symbol);
            if (index !== -1) {
              updatedStrategies[index] = message.data;
            } else {
              updatedStrategies.push(message.data);
            }
            return updatedStrategies;
          });
        }
        break;
        
      default:
        console.log('Unknown message type:', message.type);
    }
  };
  
  // Change selected symbol
  const handleSymbolChange = (symbol) => {
    setSelectedSymbol(symbol);
    if (websocket) {
      // Unsubscribe from current symbol
      if (selectedSymbol) {
        websocket.unsubscribe('orderbook', selectedSymbol);
      }
      
      // Subscribe to new symbol
      websocket.subscribe('orderbook', symbol);
    }
  };

  // Render the appropriate content based on active tab
  const renderContent = () => {
    switch (activeTab) {
      case 'dashboard':
        return <Dashboard strategies={strategies} connected={connected} />;
      case 'orderbook':
        return (
          <div>
            <div className="mb-4">
              <label className="block text-sm font-medium mb-2">Select Symbol:</label>
              <select 
                value={selectedSymbol}
                onChange={(e) => handleSymbolChange(e.target.value)}
                className="bg-white dark:bg-gray-700 border border-gray-300 dark:border-gray-600 rounded px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500"
              >
                {symbols.map((symbol) => (
                  <option key={symbol} value={symbol}>{symbol}</option>
                ))}
              </select>
            </div>
            <OrderBook data={orderBooks[selectedSymbol]} symbol={selectedSymbol} />
          </div>
        );
      case 'arbitrage':
        return <ArbitrageOpportunities opportunities={arbitrageOpportunities} />;
      default:
        return <div>Select a tab</div>;
    }
  };

  return (
    <div className="flex flex-col h-screen">
      <Navigation 
        activeTab={activeTab}
        setActiveTab={setActiveTab}
        connected={connected}
        theme={theme}
        toggleTheme={toggleTheme}
      />
      <main className="flex-grow p-4">
        <div className="container mx-auto">
          {renderContent()}
        </div>
      </main>
      <footer className="bg-white dark:bg-gray-800 shadow py-4">
        <div className="container mx-auto text-center text-sm text-gray-500 dark:text-gray-400">
          Velocimex HFT Ecosystem &copy; {new Date().getFullYear()}
        </div>
      </footer>
    </div>
  );
};

// Render the App component to the DOM
ReactDOM.render(<App />, document.getElementById('root'));
