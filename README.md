# Velocimex HFT Ecosystem

Velocimex is a high-frequency trading (HFT) ecosystem written in Go with a modern React/ShadCN UI. It provides real-time market connectivity, order book management, arbitrage detection, and visualization tools for trading strategies.

![Velocimex Dashboard](https://via.placeholder.com/1200x600?text=Velocimex+HFT+Ecosystem)

## Features

- 🔌 **Market Connectivity**: Connect to real-time data sources using WebSockets and FIX protocol.
- 📊 **Order Book Engine**: Maintain top-of-book and full depth for all instruments.
- 🔄 **Arbitrage Detection**: Identify cross-exchange price differences for potential profit opportunities.
- 📈 **Strategy Simulation**: Run paper trading with realistic latency simulation.
- 🎨 **Visualization Dashboard**: Real-time order book visualization with dark/light mode toggle.
- 📱 **Responsive UI**: Modern, responsive interface built with React and Tailwind CSS.

## System Architecture

Velocimex is built with a modular architecture that separates concerns for maintainability and extensibility:

