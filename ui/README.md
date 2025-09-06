# Velocimex Enhanced UI

A modern, responsive web interface for the Velocimex High-Frequency Trading Platform built with ShadCN-inspired components and real-time data visualization.

## Features

### 🎨 Modern Design
- **ShadCN-inspired Components**: Clean, accessible UI components with consistent styling
- **Dark/Light Theme**: Seamless theme switching with system preference detection
- **Responsive Layout**: Optimized for desktop, tablet, and mobile devices
- **Smooth Animations**: Micro-interactions and transitions for better UX

### 📊 Real-time Data Visualization
- **Live Market Data**: Real-time price updates with color-coded changes
- **Interactive Order Book**: Depth visualization with best bid/ask highlighting
- **Performance Charts**: Dynamic P&L charts with Chart.js integration
- **Arbitrage Opportunities**: Real-time arbitrage detection and display

### 🔔 Smart Notifications
- **Toast Notifications**: Non-intrusive success/error/warning messages
- **Notification Panel**: Centralized notification management with read/unread states
- **Trading Alerts**: Real-time alerts for signals, arbitrage, and system events
- **Badge Counters**: Visual indicators for unread notifications

### ⚡ Performance Optimized
- **Debounced Updates**: Prevents excessive re-renders during high-frequency updates
- **Throttled Events**: Optimized event handling for smooth performance
- **Lazy Loading**: Components load only when needed
- **Efficient Re-rendering**: Smart DOM updates to minimize layout thrashing

### 🛠️ Developer Experience
- **Modular Architecture**: Clean separation of concerns with component-based design
- **TypeScript Ready**: JSDoc annotations and type hints for better IDE support
- **Hot Reloading**: Development server with live reload capabilities
- **Debug Tools**: Comprehensive logging and debugging utilities

## Architecture

### Core Components

#### VelocimexApp
The main application class that orchestrates all UI components and manages state.

```javascript
const app = new VelocimexApp();
app.on('market:update', (data) => {
    // Handle market data updates
});
```

#### ToastManager
Handles toast notifications with different types and durations.

```javascript
app.toastManager.show('Order executed successfully', 'success');
app.toastManager.show('Connection lost', 'error', 10000);
```

#### NotificationManager
Manages the notification panel with read/unread states and filtering.

```javascript
app.notificationManager.add(
    'New Trading Signal',
    'BUY BTCUSDT at $43,250',
    'trade'
);
```

#### ChartManager
Manages Chart.js instances for performance visualization.

```javascript
app.chartManager.addDataPoint(new Date(), 1250.50);
app.chartManager.updateData({ labels: [...], values: [...] });
```

### Data Flow

1. **WebSocket Connection**: Establishes real-time connection to backend
2. **Message Processing**: Parses incoming data and routes to appropriate handlers
3. **Component Updates**: UI components react to data changes and re-render
4. **User Interactions**: User actions trigger API calls and state updates

### Styling System

#### CSS Variables
The UI uses CSS custom properties for theming:

```css
:root {
  --background: 0 0% 100%;
  --foreground: 222.2 84% 4.9%;
  --primary: 221.2 83.2% 53.3%;
  /* ... more variables */
}
```

#### Component Classes
Consistent class naming following BEM methodology:

```css
.market-item { /* Base component */ }
.market-item--active { /* Modifier */ }
.market-item__symbol { /* Element */ }
```

#### Dark Mode
Automatic dark mode support with CSS custom properties:

```css
.dark {
  --background: 222.2 84% 4.9%;
  --foreground: 210 40% 98%;
  /* ... dark theme variables */
}
```

## Usage

### Basic Setup

```html
<!DOCTYPE html>
<html lang="en" class="h-full">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Velocimex - HFT Platform</title>
    <link href="https://cdn.jsdelivr.net/npm/tailwindcss@3.4.0/dist/tailwind.min.css" rel="stylesheet">
    <script src="https://cdn.jsdelivr.net/npm/chart.js@4.4.0/dist/chart.min.js"></script>
    <script src="https://unpkg.com/lucide@latest/dist/umd/lucide.js"></script>
    <link href="styles/velocimex.css" rel="stylesheet">
</head>
<body class="h-full bg-slate-50 dark:bg-slate-900">
    <!-- UI Content -->
    <script type="module" src="js/app.js"></script>
</body>
</html>
```

### Event Handling

```javascript
// Listen for market updates
app.on('market:update', (data) => {
    console.log('Market data:', data);
});

// Listen for trading signals
app.on('signal:new', (signal) => {
    console.log('New signal:', signal);
});

// Listen for connection status
app.on('websocket:open', () => {
    console.log('Connected to server');
});
```

### Customization

#### Theme Customization
```javascript
// Toggle theme
app.toggleTheme();

// Apply specific theme
app.applyTheme('dark');
```

#### Settings Management
```javascript
// Update settings
app.settings.displayDepth = 20;
app.settings.updateInterval = 500;
app.saveSettings();
```

## Components

### Market Overview
- Real-time price updates
- Color-coded price changes
- Exchange information
- Click to select market

### Order Book
- Bid/ask visualization
- Spread calculation
- Depth levels configuration
- Best price highlighting

### Arbitrage Opportunities
- Real-time opportunity detection
- Profit percentage display
- Volume information
- Exchange pair details

### Performance Charts
- P&L visualization
- Time-based data
- Interactive tooltips
- Responsive design

### Trading Signals
- Signal history
- Buy/sell indicators
- Timestamp display
- Clear functionality

### Risk Metrics
- Real-time risk calculations
- VaR display
- Sharpe ratio
- Drawdown monitoring

## Browser Support

- **Chrome**: 90+
- **Firefox**: 88+
- **Safari**: 14+
- **Edge**: 90+

## Performance Considerations

### Optimization Strategies
1. **Debounced Updates**: Prevents excessive re-renders
2. **Throttled Events**: Limits event handler frequency
3. **Virtual Scrolling**: For large data sets
4. **Lazy Loading**: Components load on demand
5. **Memory Management**: Proper cleanup of event listeners

### Monitoring
- Performance metrics tracking
- Memory usage monitoring
- Network request optimization
- Bundle size analysis

## Development

### Local Development
```bash
# Start development server
npm run dev

# Build for production
npm run build

# Run tests
npm test
```

### Code Structure
```
ui/
├── components/          # Reusable UI components
├── lib/                # Utility libraries
├── styles/             # CSS and styling
├── js/                 # JavaScript modules
├── index.html          # Main HTML file
└── README.md           # This file
```

### Contributing
1. Follow the existing code style
2. Add JSDoc comments for functions
3. Test on multiple browsers
4. Update documentation as needed

## License

This project is part of the Velocimex HFT Platform and is licensed under the same terms.
