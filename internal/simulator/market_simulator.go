package simulator

import (
	"log"
	"math"
	"math/rand"
	"sync"
	"time"

	"velocimex/internal/normalizer"
)

// StockMarketType represents different stock markets
type StockMarketType string

const (
	NASDAQ  StockMarketType = "NASDAQ"
	NYSE    StockMarketType = "NYSE"
	NSE     StockMarketType = "NSE"
	BSE     StockMarketType = "BSE"
	SP500   StockMarketType = "S&P500"
	DowJones StockMarketType = "DowJones"
)

// StockSymbol defines a stock symbol with baseline prices
type StockSymbol struct {
	Symbol       string  `json:"symbol"`
	Name         string  `json:"name"`
	BasePrice    float64 `json:"basePrice"`
	Volatility   float64 `json:"volatility"`    // Daily volatility percentage
	MarketType   StockMarketType `json:"marketType"`
	LastPrice    float64 `json:"lastPrice"`
	PercentChange float64 `json:"percentChange"`
	Volume       int64   `json:"volume"`
	High         float64 `json:"high"`
	Low          float64 `json:"low"`
	Open         float64 `json:"open"`
}

// MarketSimulator simulates market data for stocks
type MarketSimulator struct {
	symbols      map[string]*StockSymbol
	subscribers  []chan *MarketUpdate
	running      bool
	mu           sync.RWMutex
	stopChan     chan struct{}
	updateRate   time.Duration
	warningShown bool
}

// MarketUpdate represents an update from the market
type MarketUpdate struct {
	Symbol       string
	Price        float64
	Volume       int64
	Timestamp    time.Time
	OrderBook    *normalizer.OrderBookUpdate
	PercentChange float64
	MarketType   StockMarketType
}

// NewMarketSimulator creates a new market simulator
func NewMarketSimulator(updateRate time.Duration) *MarketSimulator {
	// Initialize with a random seed
	rand.Seed(time.Now().UnixNano())
	
	// Create the simulator
	sim := &MarketSimulator{
		symbols:     make(map[string]*StockSymbol),
		subscribers: make([]chan *MarketUpdate, 0),
		stopChan:    make(chan struct{}),
		updateRate:  updateRate,
	}

	// Add default stock symbols
	sim.AddStockSymbols(DefaultStockSymbols())
	
	return sim
}

// DefaultStockSymbols returns a list of default stock symbols
func DefaultStockSymbols() []*StockSymbol {
	return []*StockSymbol{
		// NASDAQ
		{Symbol: "AAPL", Name: "Apple Inc.", BasePrice: 150.0, Volatility: 2.5, MarketType: NASDAQ},
		{Symbol: "MSFT", Name: "Microsoft Corporation", BasePrice: 280.0, Volatility: 2.0, MarketType: NASDAQ},
		{Symbol: "GOOGL", Name: "Alphabet Inc.", BasePrice: 2500.0, Volatility: 2.2, MarketType: NASDAQ},
		{Symbol: "AMZN", Name: "Amazon.com Inc.", BasePrice: 3200.0, Volatility: 3.0, MarketType: NASDAQ},
		{Symbol: "TSLA", Name: "Tesla, Inc.", BasePrice: 700.0, Volatility: 5.0, MarketType: NASDAQ},
		
		// NYSE
		{Symbol: "JPM", Name: "JPMorgan Chase & Co.", BasePrice: 150.0, Volatility: 2.0, MarketType: NYSE},
		{Symbol: "BAC", Name: "Bank of America Corp.", BasePrice: 40.0, Volatility: 2.5, MarketType: NYSE},
		{Symbol: "WMT", Name: "Walmart Inc.", BasePrice: 140.0, Volatility: 1.5, MarketType: NYSE},
		{Symbol: "DIS", Name: "The Walt Disney Company", BasePrice: 180.0, Volatility: 2.8, MarketType: NYSE},
		{Symbol: "XOM", Name: "Exxon Mobil Corporation", BasePrice: 60.0, Volatility: 2.2, MarketType: NYSE},
		
		// NSE
		{Symbol: "RELIANCE", Name: "Reliance Industries Ltd.", BasePrice: 2200.0, Volatility: 2.5, MarketType: NSE},
		{Symbol: "TCS", Name: "Tata Consultancy Services Ltd.", BasePrice: 3300.0, Volatility: 2.0, MarketType: NSE},
		{Symbol: "INFY", Name: "Infosys Ltd.", BasePrice: 1600.0, Volatility: 2.2, MarketType: NSE},
		{Symbol: "HDFCBANK", Name: "HDFC Bank Ltd.", BasePrice: 1500.0, Volatility: 1.8, MarketType: NSE},
		{Symbol: "HINDUNILVR", Name: "Hindustan Unilever Ltd.", BasePrice: 2400.0, Volatility: 1.5, MarketType: NSE},
		
		// BSE
		{Symbol: "TATASTEEL.BSE", Name: "Tata Steel Ltd.", BasePrice: 1300.0, Volatility: 3.0, MarketType: BSE},
		{Symbol: "BAJFINANCE.BSE", Name: "Bajaj Finance Ltd.", BasePrice: 6500.0, Volatility: 3.5, MarketType: BSE},
		{Symbol: "ICICIBANK.BSE", Name: "ICICI Bank Ltd.", BasePrice: 700.0, Volatility: 2.2, MarketType: BSE},
		{Symbol: "SBIN.BSE", Name: "State Bank of India", BasePrice: 400.0, Volatility: 2.8, MarketType: BSE},
		{Symbol: "AXISBANK.BSE", Name: "Axis Bank Ltd.", BasePrice: 750.0, Volatility: 2.5, MarketType: BSE},
		
		// S&P 500 ETF
		{Symbol: "SPY", Name: "SPDR S&P 500 ETF Trust", BasePrice: 420.0, Volatility: 1.5, MarketType: SP500},
		
		// Dow Jones ETF
		{Symbol: "DIA", Name: "SPDR Dow Jones Industrial Average ETF", BasePrice: 350.0, Volatility: 1.3, MarketType: DowJones},
	}
}

// AddStockSymbols adds stock symbols to the simulator
func (sim *MarketSimulator) AddStockSymbols(symbols []*StockSymbol) {
	sim.mu.Lock()
	defer sim.mu.Unlock()
	
	for _, symbol := range symbols {
		// Initialize the symbol with base price
		symbol.LastPrice = symbol.BasePrice
		symbol.Open = symbol.BasePrice
		symbol.High = symbol.BasePrice
		symbol.Low = symbol.BasePrice
		symbol.Volume = 0
		symbol.PercentChange = 0.0
		
		sim.symbols[symbol.Symbol] = symbol
	}
}

// Subscribe adds a subscriber to receive market updates
func (sim *MarketSimulator) Subscribe(bufferSize int) chan *MarketUpdate {
	sim.mu.Lock()
	defer sim.mu.Unlock()
	
	ch := make(chan *MarketUpdate, bufferSize)
	sim.subscribers = append(sim.subscribers, ch)
	
	return ch
}

// Unsubscribe removes a subscriber
func (sim *MarketSimulator) Unsubscribe(ch chan *MarketUpdate) {
	sim.mu.Lock()
	defer sim.mu.Unlock()
	
	for i, sub := range sim.subscribers {
		if sub == ch {
			// Remove the subscriber
			sim.subscribers = append(sim.subscribers[:i], sim.subscribers[i+1:]...)
			close(ch)
			break
		}
	}
}

// Start begins the market simulation
func (sim *MarketSimulator) Start() {
	sim.mu.Lock()
	defer sim.mu.Unlock()
	
	if sim.running {
		return
	}
	
	sim.running = true
	sim.stopChan = make(chan struct{})
	
	// Start the simulation goroutine
	go sim.runSimulation()
	
	// Log a warning to indicate this is simulated data
	if !sim.warningShown {
		sim.warningShown = true
		SimulationWarning()
	}
}

// Stop halts the market simulation
func (sim *MarketSimulator) Stop() {
	sim.mu.Lock()
	defer sim.mu.Unlock()
	
	if !sim.running {
		return
	}
	
	close(sim.stopChan)
	sim.running = false
}

// IsRunning returns whether the simulator is running
func (sim *MarketSimulator) IsRunning() bool {
	sim.mu.RLock()
	defer sim.mu.RUnlock()
	
	return sim.running
}

// GetSymbols returns all symbols in the simulator
func (sim *MarketSimulator) GetSymbols() map[string]*StockSymbol {
	sim.mu.RLock()
	defer sim.mu.RUnlock()
	
	// Create a copy of the map
	symbols := make(map[string]*StockSymbol, len(sim.symbols))
	for k, v := range sim.symbols {
		symbols[k] = v
	}
	
	return symbols
}

// GetSymbolsForMarket returns symbols for a specific market
func (sim *MarketSimulator) GetSymbolsForMarket(market StockMarketType) []*StockSymbol {
	sim.mu.RLock()
	defer sim.mu.RUnlock()
	
	symbols := make([]*StockSymbol, 0)
	for _, symbol := range sim.symbols {
		if symbol.MarketType == market {
			symbols = append(symbols, symbol)
		}
	}
	
	return symbols
}

// runSimulation is the main simulation loop
func (sim *MarketSimulator) runSimulation() {
	ticker := time.NewTicker(sim.updateRate)
	defer ticker.Stop()
	
	for {
		select {
		case <-sim.stopChan:
			return
		case <-ticker.C:
			// Update all symbols
			sim.updatePrices()
			
			// Send updates to subscribers
			sim.publishUpdates()
		}
	}
}

// updatePrices updates all symbol prices
func (sim *MarketSimulator) updatePrices() {
	sim.mu.Lock()
	defer sim.mu.Unlock()
	
	// Get a single random factor that affects all stocks (market sentiment)
	marketSentiment := (rand.Float64() * 2) - 1 // -1.0 to +1.0
	
	for _, symbol := range sim.symbols {
		// Calculate daily volatility into per-update volatility
		// Assuming ~6.5 hours of trading per day and updates every second
		updateVolatility := symbol.Volatility / math.Sqrt(6.5 * 3600 / sim.updateRate.Seconds())
		
		// Calculate random price change with market sentiment factored in
		// 30% market sentiment, 70% stock-specific movement
		randomChange := ((0.3 * marketSentiment) + (0.7 * ((rand.Float64() * 2) - 1))) * updateVolatility / 100
		
		// Update the price
		oldPrice := symbol.LastPrice
		newPrice := oldPrice * (1 + randomChange)
		
		// Update the symbol data
		symbol.LastPrice = newPrice
		symbol.PercentChange = (newPrice - symbol.Open) / symbol.Open * 100
		
		// Update high and low
		if newPrice > symbol.High {
			symbol.High = newPrice
		}
		if newPrice < symbol.Low {
			symbol.Low = newPrice
		}
		
		// Generate random volume (higher on bigger price moves)
		volumeChange := math.Abs(randomChange) * 100
		symbol.Volume += int64(rand.Float64() * 10000 * (1 + volumeChange))
	}
}

// publishUpdates sends updates to all subscribers
func (sim *MarketSimulator) publishUpdates() {
	sim.mu.RLock()
	defer sim.mu.RUnlock()
	
	now := time.Now()
	
	// For each symbol, create an update and send it to all subscribers
	for _, symbol := range sim.symbols {
		// Create order book update
		orderBook := sim.generateOrderBook(symbol)
		
		update := &MarketUpdate{
			Symbol:        symbol.Symbol,
			Price:         symbol.LastPrice,
			Volume:        symbol.Volume,
			Timestamp:     now,
			OrderBook:     orderBook,
			PercentChange: symbol.PercentChange,
			MarketType:    symbol.MarketType,
		}
		
		// Send to all subscribers
		for _, sub := range sim.subscribers {
			select {
			case sub <- update:
				// Successfully sent
			default:
				// Channel is full, skip this update for this subscriber
			}
		}
	}
}

// generateOrderBook creates a simulated order book for a symbol
func (sim *MarketSimulator) generateOrderBook(symbol *StockSymbol) *normalizer.OrderBookUpdate {
	// Number of price levels to generate
	const numLevels = 10
	
	// Calculate the tick size (minimum price increment)
	var tickSize float64
	if symbol.LastPrice < 10 {
		tickSize = 0.01
	} else if symbol.LastPrice < 100 {
		tickSize = 0.05
	} else if symbol.LastPrice < 1000 {
		tickSize = 0.10
	} else {
		tickSize = 0.25
	}
	
	// Create bids and asks arrays
	bids := make([]normalizer.PriceLevel, numLevels)
	asks := make([]normalizer.PriceLevel, numLevels)
	
	// Calculate a random spread as a percentage of price
	// Typical spread is 0.05% to 0.2% for liquid stocks
	spreadPct := 0.05 + (rand.Float64() * 0.15)
	spreadAmount := symbol.LastPrice * spreadPct / 100
	
	// Ensure spread is at least 1 tick
	if spreadAmount < tickSize {
		spreadAmount = tickSize
	}
	
	// Calculate best bid and ask
	bestBid := symbol.LastPrice - (spreadAmount / 2)
	bestAsk := symbol.LastPrice + (spreadAmount / 2)
	
	// Round to tick size
	bestBid = math.Floor(bestBid/tickSize) * tickSize
	bestAsk = math.Ceil(bestAsk/tickSize) * tickSize
	
	// Generate bids (highest to lowest)
	accVolume := 0.0
	for i := 0; i < numLevels; i++ {
		price := bestBid - (float64(i) * tickSize)
		
		// Volume increases as we move away from the best bid
		// Also add some randomness
		volumeFactor := 1.0 + (float64(i) * 0.2) + (rand.Float64() * 0.5)
		volume := 100 * volumeFactor
		accVolume += volume
		
		bids[i] = normalizer.PriceLevel{
			Price:  price,
			Volume: volume,
		}
	}
	
	// Generate asks (lowest to highest)
	accVolume = 0.0
	for i := 0; i < numLevels; i++ {
		price := bestAsk + (float64(i) * tickSize)
		
		// Volume increases as we move away from the best ask
		// Also add some randomness
		volumeFactor := 1.0 + (float64(i) * 0.2) + (rand.Float64() * 0.5)
		volume := 100 * volumeFactor
		accVolume += volume
		
		asks[i] = normalizer.PriceLevel{
			Price:  price,
			Volume: volume,
		}
	}
	
	// Create the order book update
	orderBook := &normalizer.OrderBookUpdate{
		Exchange:  string(symbol.MarketType),
		Symbol:    symbol.Symbol,
		Bids:      bids,
		Asks:      asks,
		Timestamp: time.Now(),
		Snapshot:  true,
	}
	
	return orderBook
}

// SimulationWarning logs a warning that the data is simulated
func SimulationWarning() {
	log.Println("⚠️ WARNING: Using simulated market data. This is not real market data and should not be used for actual trading decisions.")
	log.Println("⚠️ Connect to real market data sources by providing API keys in the configuration.")
}