package backtesting

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/shopspring/decimal"
)

// DataProvider defines the interface for historical data providers
type DataProvider interface {
	// Data retrieval
	GetHistoricalData(symbol, exchange string, startDate, endDate time.Time) (*HistoricalData, error)
	GetAvailableSymbols() ([]string, error)
	GetAvailableExchanges(symbol string) ([]string, error)
	
	// Data management
	StoreHistoricalData(data *HistoricalData) error
	LoadHistoricalData(symbol, exchange string) (*HistoricalData, error)
	
	// Data validation
	ValidateData(data *HistoricalData) error
	CleanData(data *HistoricalData) error
}

// FileDataProvider implements DataProvider for file-based storage
type FileDataProvider struct {
	dataDir string
}

// NewFileDataProvider creates a new file-based data provider
func NewFileDataProvider(dataDir string) *FileDataProvider {
	return &FileDataProvider{
		dataDir: dataDir,
	}
}

// GetHistoricalData retrieves historical data from files
func (fdp *FileDataProvider) GetHistoricalData(symbol, exchange string, startDate, endDate time.Time) (*HistoricalData, error) {
	// Try to load from JSON file first
	if data, err := fdp.loadFromJSON(symbol, exchange); err == nil {
		return fdp.filterDataByDateRange(data, startDate, endDate), nil
	}
	
	// Try to load from CSV file
	if data, err := fdp.loadFromCSV(symbol, exchange); err == nil {
		return fdp.filterDataByDateRange(data, startDate, endDate), nil
	}
	
	return nil, fmt.Errorf("no historical data found for %s on %s", symbol, exchange)
}

// GetAvailableSymbols returns available symbols
func (fdp *FileDataProvider) GetAvailableSymbols() ([]string, error) {
	files, err := os.ReadDir(fdp.dataDir)
	if err != nil {
		return nil, err
	}
	
	symbols := make(map[string]bool)
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		
		// Extract symbol from filename (e.g., "BTC_USD_binance.json" -> "BTC/USD")
		symbol := fdp.extractSymbolFromFilename(file.Name())
		if symbol != "" {
			symbols[symbol] = true
		}
	}
	
	result := make([]string, 0, len(symbols))
	for symbol := range symbols {
		result = append(result, symbol)
	}
	
	return result, nil
}

// GetAvailableExchanges returns available exchanges for a symbol
func (fdp *FileDataProvider) GetAvailableExchanges(symbol string) ([]string, error) {
	files, err := os.ReadDir(fdp.dataDir)
	if err != nil {
		return nil, err
	}
	
	exchanges := make(map[string]bool)
	symbolPattern := fdp.symbolToFilename(symbol)
	
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		
		if fdp.filenameMatchesSymbol(file.Name(), symbolPattern) {
			exchange := fdp.extractExchangeFromFilename(file.Name())
			if exchange != "" {
				exchanges[exchange] = true
			}
		}
	}
	
	result := make([]string, 0, len(exchanges))
	for exchange := range exchanges {
		result = append(result, exchange)
	}
	
	return result, nil
}

// StoreHistoricalData stores historical data to files
func (fdp *FileDataProvider) StoreHistoricalData(data *HistoricalData) error {
	// Ensure data directory exists
	if err := os.MkdirAll(fdp.dataDir, 0755); err != nil {
		return err
	}
	
	// Store as JSON
	filename := fmt.Sprintf("%s_%s.json", fdp.symbolToFilename(data.Symbol), data.Exchange)
	filepath := fmt.Sprintf("%s/%s", fdp.dataDir, filename)
	
	file, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer file.Close()
	
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

// LoadHistoricalData loads historical data from files
func (fdp *FileDataProvider) LoadHistoricalData(symbol, exchange string) (*HistoricalData, error) {
	return fdp.loadFromJSON(symbol, exchange)
}

// ValidateData validates historical data
func (fdp *FileDataProvider) ValidateData(data *HistoricalData) error {
	if data.Symbol == "" {
		return fmt.Errorf("symbol cannot be empty")
	}
	
	if data.Exchange == "" {
		return fmt.Errorf("exchange cannot be empty")
	}
	
	if len(data.DataPoints) == 0 {
		return fmt.Errorf("no data points found")
	}
	
	// Validate data points
	for i, point := range data.DataPoints {
		if point.Timestamp.IsZero() {
			return fmt.Errorf("data point %d has zero timestamp", i)
		}
		
		if point.Close.LessThanOrEqual(decimal.Zero) {
			return fmt.Errorf("data point %d has invalid close price", i)
		}
		
		if point.Volume.LessThan(decimal.Zero) {
			return fmt.Errorf("data point %d has negative volume", i)
		}
	}
	
	return nil
}

// CleanData cleans historical data
func (fdp *FileDataProvider) CleanData(data *HistoricalData) error {
	// Remove invalid data points
	validPoints := make([]*DataPoint, 0, len(data.DataPoints))
	
	for _, point := range data.DataPoints {
		if point.Timestamp.IsZero() || point.Close.LessThanOrEqual(decimal.Zero) {
			continue
		}
		
		// Fix invalid OHLC data
		if point.Open.LessThanOrEqual(decimal.Zero) {
			point.Open = point.Close
		}
		if point.High.LessThanOrEqual(decimal.Zero) {
			point.High = point.Close
		}
		if point.Low.LessThanOrEqual(decimal.Zero) {
			point.Low = point.Close
		}
		
		// Ensure High >= Low
		if point.High.LessThan(point.Low) {
			point.High = point.Low
		}
		
		// Ensure High >= Open and High >= Close
		if point.High.LessThan(point.Open) {
			point.High = point.Open
		}
		if point.High.LessThan(point.Close) {
			point.High = point.Close
		}
		
		// Ensure Low <= Open and Low <= Close
		if point.Low.GreaterThan(point.Open) {
			point.Low = point.Open
		}
		if point.Low.GreaterThan(point.Close) {
			point.Low = point.Close
		}
		
		// Fix bid/ask data
		if point.Bid.LessThanOrEqual(decimal.Zero) {
			point.Bid = point.Close.Mul(decimal.NewFromFloat(0.999))
		}
		if point.Ask.LessThanOrEqual(decimal.Zero) {
			point.Ask = point.Close.Mul(decimal.NewFromFloat(1.001))
		}
		
		// Ensure Ask >= Bid
		if point.Ask.LessThan(point.Bid) {
			point.Ask = point.Bid.Mul(decimal.NewFromFloat(1.001))
		}
		
		validPoints = append(validPoints, point)
	}
	
	data.DataPoints = validPoints
	
	// Update start and end times
	if len(data.DataPoints) > 0 {
		data.StartTime = data.DataPoints[0].Timestamp
		data.EndTime = data.DataPoints[len(data.DataPoints)-1].Timestamp
	}
	
	return nil
}

// Private methods

func (fdp *FileDataProvider) loadFromJSON(symbol, exchange string) (*HistoricalData, error) {
	filename := fmt.Sprintf("%s_%s.json", fdp.symbolToFilename(symbol), exchange)
	filepath := fmt.Sprintf("%s/%s", fdp.dataDir, filename)
	
	file, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	
	var data HistoricalData
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&data); err != nil {
		return nil, err
	}
	
	return &data, nil
}

func (fdp *FileDataProvider) loadFromCSV(symbol, exchange string) (*HistoricalData, error) {
	filename := fmt.Sprintf("%s_%s.csv", fdp.symbolToFilename(symbol), exchange)
	filepath := fmt.Sprintf("%s/%s", fdp.dataDir, filename)
	
	file, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	
	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}
	
	if len(records) < 2 {
		return nil, fmt.Errorf("insufficient data in CSV file")
	}
	
	// Parse header
	header := records[0]
	data := &HistoricalData{
		Symbol:     symbol,
		Exchange:   exchange,
		DataPoints: make([]*DataPoint, 0, len(records)-1),
		Metadata:   make(map[string]interface{}),
	}
	
	// Parse data rows
	for i := 1; i < len(records); i++ {
		record := records[i]
		if len(record) < len(header) {
			continue
		}
		
		point, err := fdp.parseCSVRecord(header, record)
		if err != nil {
			continue // Skip invalid records
		}
		
		data.DataPoints = append(data.DataPoints, point)
	}
	
	// Set start and end times
	if len(data.DataPoints) > 0 {
		data.StartTime = data.DataPoints[0].Timestamp
		data.EndTime = data.DataPoints[len(data.DataPoints)-1].Timestamp
	}
	
	return data, nil
}

func (fdp *FileDataProvider) parseCSVRecord(header, record []string) (*DataPoint, error) {
	point := &DataPoint{
		Metadata: make(map[string]interface{}),
	}
	
	for i, field := range header {
		if i >= len(record) {
			break
		}
		
		value := record[i]
		if value == "" {
			continue
		}
		
		switch field {
		case "timestamp", "time", "date":
			if timestamp, err := time.Parse("2006-01-02 15:04:05", value); err == nil {
				point.Timestamp = timestamp
			} else if timestamp, err := time.Parse("2006-01-02", value); err == nil {
				point.Timestamp = timestamp
			}
		case "open", "o":
			if price, err := decimal.NewFromString(value); err == nil {
				point.Open = price
			}
		case "high", "h":
			if price, err := decimal.NewFromString(value); err == nil {
				point.High = price
			}
		case "low", "l":
			if price, err := decimal.NewFromString(value); err == nil {
				point.Low = price
			}
		case "close", "c":
			if price, err := decimal.NewFromString(value); err == nil {
				point.Close = price
			}
		case "volume", "v":
			if volume, err := decimal.NewFromString(value); err == nil {
				point.Volume = volume
			}
		case "bid", "b":
			if price, err := decimal.NewFromString(value); err == nil {
				point.Bid = price
			}
		case "ask", "a":
			if price, err := decimal.NewFromString(value); err == nil {
				point.Ask = price
			}
		case "bid_size", "bs":
			if size, err := decimal.NewFromString(value); err == nil {
				point.BidSize = size
			}
		case "ask_size", "as":
			if size, err := decimal.NewFromString(value); err == nil {
				point.AskSize = size
			}
		default:
			point.Metadata[field] = value
		}
	}
	
	if point.Timestamp.IsZero() {
		return nil, fmt.Errorf("invalid timestamp")
	}
	
	return point, nil
}

func (fdp *FileDataProvider) filterDataByDateRange(data *HistoricalData, startDate, endDate time.Time) *HistoricalData {
	filtered := &HistoricalData{
		Symbol:     data.Symbol,
		Exchange:   data.Exchange,
		DataPoints: make([]*DataPoint, 0),
		StartTime:  startDate,
		EndTime:    endDate,
		Frequency:  data.Frequency,
		Metadata:   data.Metadata,
	}
	
	for _, point := range data.DataPoints {
		if point.Timestamp.After(startDate) && point.Timestamp.Before(endDate) {
			filtered.DataPoints = append(filtered.DataPoints, point)
		}
	}
	
	return filtered
}

func (fdp *FileDataProvider) symbolToFilename(symbol string) string {
	// Convert "BTC/USD" to "BTC_USD"
	return fmt.Sprintf("%s", symbol)
}

func (fdp *FileDataProvider) filenameMatchesSymbol(filename, symbolPattern string) bool {
	// Check if filename contains the symbol pattern
	return len(filename) > len(symbolPattern) && filename[:len(symbolPattern)] == symbolPattern
}

func (fdp *FileDataProvider) extractSymbolFromFilename(filename string) string {
	// Extract symbol from filename like "BTC_USD_binance.json"
	parts := make([]string, 0)
	current := ""
	
	for _, char := range filename {
		if char == '_' || char == '.' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(char)
		}
	}
	
	if len(parts) >= 2 {
		return fmt.Sprintf("%s/%s", parts[0], parts[1])
	}
	
	return ""
}

func (fdp *FileDataProvider) extractExchangeFromFilename(filename string) string {
	// Extract exchange from filename like "BTC_USD_binance.json"
	parts := make([]string, 0)
	current := ""
	
	for _, char := range filename {
		if char == '_' || char == '.' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(char)
		}
	}
	
	if len(parts) >= 3 {
		return parts[2]
	}
	
	return ""
}

// MockDataProvider implements DataProvider for testing
type MockDataProvider struct {
	data map[string]map[string]*HistoricalData
}

// NewMockDataProvider creates a new mock data provider
func NewMockDataProvider() *MockDataProvider {
	return &MockDataProvider{
		data: make(map[string]map[string]*HistoricalData),
	}
}

// GetHistoricalData returns mock historical data
func (mdp *MockDataProvider) GetHistoricalData(symbol, exchange string, startDate, endDate time.Time) (*HistoricalData, error) {
	if exchanges, exists := mdp.data[symbol]; exists {
		if data, exists := exchanges[exchange]; exists {
			return data, nil
		}
	}
	
	return nil, fmt.Errorf("no data found for %s on %s", symbol, exchange)
}

// GetAvailableSymbols returns available symbols
func (mdp *MockDataProvider) GetAvailableSymbols() ([]string, error) {
	symbols := make([]string, 0, len(mdp.data))
	for symbol := range mdp.data {
		symbols = append(symbols, symbol)
	}
	return symbols, nil
}

// GetAvailableExchanges returns available exchanges for a symbol
func (mdp *MockDataProvider) GetAvailableExchanges(symbol string) ([]string, error) {
	if exchanges, exists := mdp.data[symbol]; exists {
		exchangeList := make([]string, 0, len(exchanges))
		for exchange := range exchanges {
			exchangeList = append(exchangeList, exchange)
		}
		return exchangeList, nil
	}
	return []string{}, nil
}

// StoreHistoricalData stores mock historical data
func (mdp *MockDataProvider) StoreHistoricalData(data *HistoricalData) error {
	if mdp.data[data.Symbol] == nil {
		mdp.data[data.Symbol] = make(map[string]*HistoricalData)
	}
	mdp.data[data.Symbol][data.Exchange] = data
	return nil
}

// LoadHistoricalData loads mock historical data
func (mdp *MockDataProvider) LoadHistoricalData(symbol, exchange string) (*HistoricalData, error) {
	return mdp.GetHistoricalData(symbol, exchange, time.Time{}, time.Time{})
}

// ValidateData validates mock historical data
func (mdp *MockDataProvider) ValidateData(data *HistoricalData) error {
	return nil // Mock validation always passes
}

// CleanData cleans mock historical data
func (mdp *MockDataProvider) CleanData(data *HistoricalData) error {
	return nil // Mock cleaning does nothing
}
