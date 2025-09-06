package feeds

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"velocimex/internal/config"
	"velocimex/internal/normalizer"
)

// StockMarketFeed implements stock market data feed using various APIs
type StockMarketFeed struct {
	config     config.FeedConfig
	normalizer *normalizer.Normalizer
	isConnected bool
	mu         sync.Mutex
	done       chan struct{}
	orderBookManager OrderBookManager
	httpClient *http.Client
	apiKey     string
}

// StockQuote represents a stock quote from various exchanges
type StockQuote struct {
	Symbol    string  `json:"symbol"`
	Price     float64 `json:"price"`
	Volume    int64   `json:"volume"`
	Bid       float64 `json:"bid"`
	Ask       float64 `json:"ask"`
	High      float64 `json:"high"`
	Low       float64 `json:"low"`
	Open      float64 `json:"open"`
	PrevClose float64 `json:"prevClose"`
	Timestamp time.Time `json:"timestamp"`
	Exchange  string   `json:"exchange"`
}

// AlphaVantageResponse represents Alpha Vantage API response
type AlphaVantageResponse struct {
	GlobalQuote struct {
		Symbol           string `json:"01. symbol"`
		Open             string `json:"02. open"`
		High             string `json:"03. high"`
		Low              string `json:"04. low"`
		Price            string `json:"05. price"`
		Volume           string `json:"06. volume"`
		LatestTradingDay string `json:"07. latest trading day"`
		PreviousClose    string `json:"08. previous close"`
		Change           string `json:"09. change"`
		ChangePercent    string `json:"10. change percent"`
	} `json:"Global Quote"`
}

// YahooFinanceResponse represents Yahoo Finance API response
type YahooFinanceResponse struct {
	QuoteResponse struct {
		Result []struct {
			Symbol              string  `json:"symbol"`
			RegularMarketPrice  float64 `json:"regularMarketPrice"`
			RegularMarketVolume int64   `json:"regularMarketVolume"`
			Bid                 float64 `json:"bid"`
			Ask                 float64 `json:"ask"`
			RegularMarketHigh   float64 `json:"regularMarketHigh"`
			RegularMarketLow    float64 `json:"regularMarketLow"`
			RegularMarketOpen   float64 `json:"regularMarketOpen"`
			RegularMarketPreviousClose float64 `json:"regularMarketPreviousClose"`
			Exchange            string  `json:"fullExchangeName"`
		} `json:"result"`
	} `json:"quoteResponse"`
}

// NewStockMarketFeed creates a new stock market feed
func NewStockMarketFeed(config config.FeedConfig, norm *normalizer.Normalizer) (*StockMarketFeed, error) {
	return &StockMarketFeed{
		config:     config,
		normalizer: norm,
		done:       make(chan struct{}),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		apiKey: config.APIKey,
	}, nil
}

// SetOrderBookManager sets the order book manager
func (f *StockMarketFeed) SetOrderBookManager(manager OrderBookManager) {
	f.orderBookManager = manager
}

// Connect establishes a connection to stock market data
func (f *StockMarketFeed) Connect() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.isConnected {
		return nil
	}

	f.isConnected = true

	// Start data fetching
	go f.fetchData()

	log.Printf("Connected to stock market feed: %s", f.config.Name)
	return nil
}

// Disconnect stops the stock market data feed
func (f *StockMarketFeed) Disconnect() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if !f.isConnected {
		return nil
	}

	close(f.done)
	f.isConnected = false

	log.Printf("Disconnected from stock market feed: %s", f.config.Name)
	return nil
}

// Subscribe subscribes to market data for a symbol
func (f *StockMarketFeed) Subscribe(symbol string) error {
	log.Printf("Subscribed to %s on stock market feed %s", symbol, f.config.Name)
	return nil
}

// Unsubscribe unsubscribes from market data for a symbol
func (f *StockMarketFeed) Unsubscribe(symbol string) error {
	log.Printf("Unsubscribed from %s on stock market feed %s", symbol, f.config.Name)
	return nil
}

// IsConnected returns whether the feed is connected
func (f *StockMarketFeed) IsConnected() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.isConnected
}

// fetchData continuously fetches stock market data
func (f *StockMarketFeed) fetchData() {
	ticker := time.NewTicker(5 * time.Second) // Fetch every 5 seconds
	defer ticker.Stop()

	for {
		select {
		case <-f.done:
			return
		case <-ticker.C:
			f.fetchStockData()
		}
	}
}

// fetchStockData fetches data for all configured symbols
func (f *StockMarketFeed) fetchStockData() {
	for _, symbol := range f.config.Symbols {
		go f.fetchSymbolData(symbol)
	}
}

// fetchSymbolData fetches data for a specific symbol
func (f *StockMarketFeed) fetchSymbolData(symbol string) {
	// Try different data sources based on the exchange
	switch f.config.Name {
	case "nasdaq", "nyse":
		f.fetchFromYahooFinance(symbol)
	case "nse", "bse":
		f.fetchFromAlphaVantage(symbol)
	case "sp500", "dow":
		f.fetchIndexData(symbol)
	default:
		f.fetchFromYahooFinance(symbol)
	}
}

// fetchFromYahooFinance fetches data from Yahoo Finance API
func (f *StockMarketFeed) fetchFromYahooFinance(symbol string) {
	url := fmt.Sprintf("https://query1.finance.yahoo.com/v7/finance/quote?symbols=%s", symbol)
	
	resp, err := f.httpClient.Get(url)
	if err != nil {
		log.Printf("Failed to fetch data from Yahoo Finance for %s: %v", symbol, err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read Yahoo Finance response for %s: %v", symbol, err)
		return
	}

	var yahooResp YahooFinanceResponse
	if err := json.Unmarshal(body, &yahooResp); err != nil {
		log.Printf("Failed to parse Yahoo Finance response for %s: %v", symbol, err)
		return
	}

	if len(yahooResp.QuoteResponse.Result) > 0 {
		result := yahooResp.QuoteResponse.Result[0]
		f.processStockQuote(StockQuote{
			Symbol:    result.Symbol,
			Price:     result.RegularMarketPrice,
			Volume:    result.RegularMarketVolume,
			Bid:       result.Bid,
			Ask:       result.Ask,
			High:      result.RegularMarketHigh,
			Low:       result.RegularMarketLow,
			Open:      result.RegularMarketOpen,
			PrevClose: result.RegularMarketPreviousClose,
			Timestamp: time.Now(),
			Exchange:  f.config.Name,
		})
	}
}

// fetchFromAlphaVantage fetches data from Alpha Vantage API
func (f *StockMarketFeed) fetchFromAlphaVantage(symbol string) {
	if f.apiKey == "" {
		log.Printf("Alpha Vantage API key not provided for %s", symbol)
		return
	}

	url := fmt.Sprintf("https://www.alphavantage.co/query?function=GLOBAL_QUOTE&symbol=%s&apikey=%s", symbol, f.apiKey)
	
	resp, err := f.httpClient.Get(url)
	if err != nil {
		log.Printf("Failed to fetch data from Alpha Vantage for %s: %v", symbol, err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read Alpha Vantage response for %s: %v", symbol, err)
		return
	}

	var alphaResp AlphaVantageResponse
	if err := json.Unmarshal(body, &alphaResp); err != nil {
		log.Printf("Failed to parse Alpha Vantage response for %s: %v", symbol, err)
		return
	}

	if alphaResp.GlobalQuote.Symbol != "" {
		quote := f.parseAlphaVantageQuote(alphaResp.GlobalQuote)
		quote.Exchange = f.config.Name
		quote.Timestamp = time.Now()
		f.processStockQuote(quote)
	}
}

// fetchIndexData fetches index data (S&P 500, Dow Jones)
func (f *StockMarketFeed) fetchIndexData(symbol string) {
	// For indices, we'll use Yahoo Finance with specific symbols
	var yahooSymbol string
	switch symbol {
	case "SP500", "S&P500":
		yahooSymbol = "^GSPC"
	case "DOW", "DOWJONES":
		yahooSymbol = "^DJI"
	default:
		yahooSymbol = symbol
	}

	f.fetchFromYahooFinance(yahooSymbol)
}

// parseAlphaVantageQuote parses Alpha Vantage quote data
func (f *StockMarketFeed) parseAlphaVantageQuote(quote struct {
	Symbol           string `json:"01. symbol"`
	Open             string `json:"02. open"`
	High             string `json:"03. high"`
	Low              string `json:"04. low"`
	Price            string `json:"05. price"`
	Volume           string `json:"06. volume"`
	LatestTradingDay string `json:"07. latest trading day"`
	PreviousClose    string `json:"08. previous close"`
	Change           string `json:"09. change"`
	ChangePercent    string `json:"10. change percent"`
}) StockQuote {
	return StockQuote{
		Symbol:    quote.Symbol,
		Price:     f.parseFloat(quote.Price),
		Volume:    f.parseInt64(quote.Volume),
		High:      f.parseFloat(quote.High),
		Low:       f.parseFloat(quote.Low),
		Open:      f.parseFloat(quote.Open),
		PrevClose: f.parseFloat(quote.PreviousClose),
	}
}

// processStockQuote processes a stock quote and updates order book
func (f *StockMarketFeed) processStockQuote(quote StockQuote) {
	// Create order book data from stock quote
	bids := []normalizer.PriceLevel{}
	asks := []normalizer.PriceLevel{}

	// Add bid/ask levels if available
	if quote.Bid > 0 {
		bids = append(bids, normalizer.PriceLevel{
			Price:  quote.Bid,
			Volume: 1000, // Default volume for stock quotes
		})
	}

	if quote.Ask > 0 {
		asks = append(asks, normalizer.PriceLevel{
			Price:  quote.Ask,
			Volume: 1000, // Default volume for stock quotes
		})
	}

	// If no bid/ask, use the current price as both
	if len(bids) == 0 && len(asks) == 0 && quote.Price > 0 {
		spread := quote.Price * 0.001 // 0.1% spread
		bids = append(bids, normalizer.PriceLevel{
			Price:  quote.Price - spread/2,
			Volume: 1000,
		})
		asks = append(asks, normalizer.PriceLevel{
			Price:  quote.Price + spread/2,
			Volume: 1000,
		})
	}

	// Normalize symbol
	normalizedSymbol := f.normalizer.NormalizeSymbol(f.config.Name, quote.Symbol)

	// Update order book if manager is available
	if f.orderBookManager != nil {
		f.orderBookManager.UpdateOrderBook(f.config.Name, normalizedSymbol, bids, asks)
	}

	// Process through normalizer
	orderBookUpdate := &normalizer.OrderBookUpdate{
		Exchange:  f.config.Name,
		Symbol:    normalizedSymbol,
		Bids:      bids,
		Asks:      asks,
		Timestamp: quote.Timestamp,
		Snapshot:  true,
	}

	f.normalizer.ProcessOrderBookUpdate(orderBookUpdate)

	log.Printf("Updated %s %s: Price=%.2f, Volume=%d", f.config.Name, quote.Symbol, quote.Price, quote.Volume)
}

// parseFloat safely parses a float string
func (f *StockMarketFeed) parseFloat(s string) float64 {
	if s == "" || s == "None" {
		return 0
	}
	
	// Remove any non-numeric characters except decimal point and minus
	cleaned := strings.TrimSpace(s)
	if cleaned == "" {
		return 0
	}
	
	// Simple parsing - in production, use strconv.ParseFloat
	var result float64
	fmt.Sscanf(cleaned, "%f", &result)
	return result
}

// parseInt64 safely parses an int64 string
func (f *StockMarketFeed) parseInt64(s string) int64 {
	if s == "" || s == "None" {
		return 0
	}
	
	cleaned := strings.TrimSpace(s)
	if cleaned == "" {
		return 0
	}
	
	var result int64
	fmt.Sscanf(cleaned, "%d", &result)
	return result
}