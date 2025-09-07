package logger

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

// LogSearchEngine provides log search and aggregation functionality
type LogSearchEngine struct {
	logDir     string
	index      *LogIndex
	mu         sync.RWMutex
	ctx        context.Context
	cancel     context.CancelFunc
	indexer    *LogIndexer
	aggregator *LogAggregator
}

// LogIndex maintains an index of log entries for fast searching
type LogIndex struct {
	entries    map[string]*LogIndexEntry
	byTime     []*LogIndexEntry
	byLevel    map[LogLevel][]*LogIndexEntry
	byComponent map[string][]*LogIndexEntry
	byTraceID  map[string][]*LogIndexEntry
	mu         sync.RWMutex
}

// LogIndexEntry represents an indexed log entry
type LogIndexEntry struct {
	ID        string    `json:"id"`
	File      string    `json:"file"`
	Line      int       `json:"line"`
	Timestamp time.Time `json:"timestamp"`
	Level     LogLevel  `json:"level"`
	Component string    `json:"component"`
	Message   string    `json:"message"`
	TraceID   string    `json:"trace_id,omitempty"`
	Size      int64     `json:"size"`
}

// LogIndexer indexes log files for searching
type LogIndexer struct {
	index     *LogIndex
	logDir    string
	interval  time.Duration
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
}

// LogAggregator provides log aggregation functionality
type LogAggregator struct {
	index *LogIndex
}

// SearchQuery represents a log search query
type SearchQuery struct {
	StartTime    *time.Time           `json:"start_time,omitempty"`
	EndTime      *time.Time           `json:"end_time,omitempty"`
	Levels       []LogLevel           `json:"levels,omitempty"`
	Components   []string             `json:"components,omitempty"`
	TraceID      string               `json:"trace_id,omitempty"`
	MessageRegex string               `json:"message_regex,omitempty"`
	Fields       map[string]interface{} `json:"fields,omitempty"`
	Limit        int                  `json:"limit,omitempty"`
	Offset       int                  `json:"offset,omitempty"`
	SortBy       string               `json:"sort_by,omitempty"` // "time", "level", "component"
	SortOrder    string               `json:"sort_order,omitempty"` // "asc", "desc"
}

// SearchResult represents the result of a log search
type SearchResult struct {
	Entries []LogIndexEntry `json:"entries"`
	Total   int             `json:"total"`
	Query   SearchQuery     `json:"query"`
	Time    time.Duration   `json:"time_ms"`
}

// AggregationQuery represents an aggregation query
type AggregationQuery struct {
	StartTime  *time.Time `json:"start_time,omitempty"`
	EndTime    *time.Time `json:"end_time,omitempty"`
	GroupBy    []string   `json:"group_by"` // "level", "component", "hour", "day"
	Aggregates []string   `json:"aggregates"` // "count", "sum", "avg"
	Field      string     `json:"field,omitempty"`
}

// AggregationResult represents the result of an aggregation
type AggregationResult struct {
	Groups []AggregationGroup `json:"groups"`
	Total  int                `json:"total"`
	Query  AggregationQuery   `json:"query"`
	Time   time.Duration      `json:"time_ms"`
}

// AggregationGroup represents a group in an aggregation result
type AggregationGroup struct {
	Key        map[string]interface{} `json:"key"`
	Count      int                    `json:"count"`
	Aggregates map[string]float64     `json:"aggregates"`
}

// NewLogSearchEngine creates a new log search engine
func NewLogSearchEngine(logDir string) *LogSearchEngine {
	ctx, cancel := context.WithCancel(context.Background())
	
	index := &LogIndex{
		entries:     make(map[string]*LogIndexEntry),
		byLevel:     make(map[LogLevel][]*LogIndexEntry),
		byComponent: make(map[string][]*LogIndexEntry),
		byTraceID:   make(map[string][]*LogIndexEntry),
	}

	indexer := &LogIndexer{
		index:    index,
		logDir:   logDir,
		interval: 1 * time.Minute,
		ctx:      ctx,
		cancel:   cancel,
	}

	aggregator := &LogAggregator{
		index: index,
	}

	lse := &LogSearchEngine{
		logDir:     logDir,
		index:      index,
		ctx:        ctx,
		cancel:     cancel,
		indexer:    indexer,
		aggregator: aggregator,
	}

	// Start indexer
	indexer.wg.Add(1)
	go indexer.run()

	return lse
}

// Search searches logs based on the given query
func (lse *LogSearchEngine) Search(ctx context.Context, query SearchQuery) (*SearchResult, error) {
	start := time.Now()
	
	lse.index.mu.RLock()
	defer lse.index.mu.RUnlock()

	var results []*LogIndexEntry

	// Filter by time range
	if query.StartTime != nil || query.EndTime != nil {
		results = lse.filterByTimeRange(lse.index.byTime, query.StartTime, query.EndTime)
	} else {
		results = make([]*LogIndexEntry, 0, len(lse.index.byTime))
		for _, entry := range lse.index.byTime {
			results = append(results, entry)
		}
	}

	// Filter by levels
	if len(query.Levels) > 0 {
		results = lse.filterByLevels(results, query.Levels)
	}

	// Filter by components
	if len(query.Components) > 0 {
		results = lse.filterByComponents(results, query.Components)
	}

	// Filter by trace ID
	if query.TraceID != "" {
		results = lse.filterByTraceID(results, query.TraceID)
	}

	// Filter by message regex
	if query.MessageRegex != "" {
		results = lse.filterByMessageRegex(results, query.MessageRegex)
	}

	// Sort results
	lse.sortResults(results, query.SortBy, query.SortOrder)

	// Apply pagination
	total := len(results)
	if query.Offset > 0 {
		if query.Offset >= len(results) {
			results = []*LogIndexEntry{}
		} else {
			results = results[query.Offset:]
		}
	}
	if query.Limit > 0 && query.Limit < len(results) {
		results = results[:query.Limit]
	}

	// Convert to slice
	entries := make([]LogIndexEntry, len(results))
	for i, entry := range results {
		entries[i] = *entry
	}

	return &SearchResult{
		Entries: entries,
		Total:   total,
		Query:   query,
		Time:    time.Since(start),
	}, nil
}

// Aggregate performs log aggregation
func (lse *LogSearchEngine) Aggregate(ctx context.Context, query AggregationQuery) (*AggregationResult, error) {
	start := time.Now()
	
	lse.index.mu.RLock()
	defer lse.index.mu.RUnlock()

	// Get entries in time range
	var entries []*LogIndexEntry
	if query.StartTime != nil || query.EndTime != nil {
		entries = lse.filterByTimeRange(lse.index.byTime, query.StartTime, query.EndTime)
	} else {
		entries = lse.index.byTime
	}

	// Group entries
	groups := lse.groupEntries(entries, query.GroupBy)

	// Calculate aggregates
	for _, group := range groups {
		lse.calculateAggregates(group, entries, query.Aggregates, query.Field)
	}

	return &AggregationResult{
		Groups: groups,
		Total:  len(entries),
		Query:  query,
		Time:   time.Since(start),
	}, nil
}

// GetLogContent retrieves the actual log content for an entry
func (lse *LogSearchEngine) GetLogContent(entry LogIndexEntry) (string, error) {
	file, err := os.Open(entry.File)
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		if lineNum == entry.Line {
			return scanner.Text(), nil
		}
	}

	return "", fmt.Errorf("line %d not found in file %s", entry.Line, entry.File)
}

// GetTraceLogs retrieves all logs for a specific trace ID
func (lse *LogSearchEngine) GetTraceLogs(traceID string) ([]LogIndexEntry, error) {
	lse.index.mu.RLock()
	defer lse.index.mu.RUnlock()

	entries, exists := lse.index.byTraceID[traceID]
	if !exists {
		return []LogIndexEntry{}, nil
	}

	result := make([]LogIndexEntry, len(entries))
	for i, entry := range entries {
		result[i] = *entry
	}

	// Sort by timestamp
	sort.Slice(result, func(i, j int) bool {
		return result[i].Timestamp.Before(result[j].Timestamp)
	})

	return result, nil
}

// Close shuts down the search engine
func (lse *LogSearchEngine) Close() error {
	lse.cancel()
	lse.indexer.wg.Wait()
	return nil
}

// Helper methods

func (lse *LogSearchEngine) filterByTimeRange(entries []*LogIndexEntry, startTime, endTime *time.Time) []*LogIndexEntry {
	var result []*LogIndexEntry

	for _, entry := range entries {
		if startTime != nil && entry.Timestamp.Before(*startTime) {
			continue
		}
		if endTime != nil && entry.Timestamp.After(*endTime) {
			continue
		}
		result = append(result, entry)
	}

	return result
}

func (lse *LogSearchEngine) filterByLevels(entries []*LogIndexEntry, levels []LogLevel) []*LogIndexEntry {
	levelMap := make(map[LogLevel]bool)
	for _, level := range levels {
		levelMap[level] = true
	}

	var result []*LogIndexEntry
	for _, entry := range entries {
		if levelMap[entry.Level] {
			result = append(result, entry)
		}
	}

	return result
}

func (lse *LogSearchEngine) filterByComponents(entries []*LogIndexEntry, components []string) []*LogIndexEntry {
	componentMap := make(map[string]bool)
	for _, component := range components {
		componentMap[component] = true
	}

	var result []*LogIndexEntry
	for _, entry := range entries {
		if componentMap[entry.Component] {
			result = append(result, entry)
		}
	}

	return result
}

func (lse *LogSearchEngine) filterByTraceID(entries []*LogIndexEntry, traceID string) []*LogIndexEntry {
	var result []*LogIndexEntry
	for _, entry := range entries {
		if entry.TraceID == traceID {
			result = append(result, entry)
		}
	}
	return result
}

func (lse *LogSearchEngine) filterByMessageRegex(entries []*LogIndexEntry, regex string) []*LogIndexEntry {
	re, err := regexp.Compile(regex)
	if err != nil {
		return entries // Return all if regex is invalid
	}

	var result []*LogIndexEntry
	for _, entry := range entries {
		if re.MatchString(entry.Message) {
			result = append(result, entry)
		}
	}
	return result
}

func (lse *LogSearchEngine) sortResults(entries []*LogIndexEntry, sortBy, sortOrder string) {
	switch sortBy {
	case "time":
		sort.Slice(entries, func(i, j int) bool {
			if sortOrder == "desc" {
				return entries[i].Timestamp.After(entries[j].Timestamp)
			}
			return entries[i].Timestamp.Before(entries[j].Timestamp)
		})
	case "level":
		sort.Slice(entries, func(i, j int) bool {
			if sortOrder == "desc" {
				return entries[i].Level > entries[j].Level
			}
			return entries[i].Level < entries[j].Level
		})
	case "component":
		sort.Slice(entries, func(i, j int) bool {
			if sortOrder == "desc" {
				return entries[i].Component > entries[j].Component
			}
			return entries[i].Component < entries[j].Component
		})
	}
}

func (lse *LogSearchEngine) groupEntries(entries []*LogIndexEntry, groupBy []string) []AggregationGroup {
	groups := make(map[string]*AggregationGroup)

	for _, entry := range entries {
		key := lse.createGroupKey(entry, groupBy)
		keyStr := lse.keyToString(key)

		if group, exists := groups[keyStr]; exists {
			group.Count++
		} else {
			groups[keyStr] = &AggregationGroup{
				Key:   key,
				Count: 1,
			}
		}
	}

	// Convert to slice
	result := make([]AggregationGroup, 0, len(groups))
	for _, group := range groups {
		result = append(result, *group)
	}

	return result
}

func (lse *LogSearchEngine) createGroupKey(entry *LogIndexEntry, groupBy []string) map[string]interface{} {
	key := make(map[string]interface{})

	for _, field := range groupBy {
		switch field {
		case "level":
			key["level"] = entry.Level.String()
		case "component":
			key["component"] = entry.Component
		case "hour":
			key["hour"] = entry.Timestamp.Hour()
		case "day":
			key["day"] = entry.Timestamp.Day()
		case "month":
			key["month"] = int(entry.Timestamp.Month())
		case "year":
			key["year"] = entry.Timestamp.Year()
		}
	}

	return key
}

func (lse *LogSearchEngine) keyToString(key map[string]interface{}) string {
	var parts []string
	for k, v := range key {
		parts = append(parts, fmt.Sprintf("%s:%v", k, v))
	}
	sort.Strings(parts)
	return strings.Join(parts, ",")
}

func (lse *LogSearchEngine) calculateAggregates(group AggregationGroup, entries []*LogIndexEntry, aggregates []string, field string) {
	group.Aggregates = make(map[string]float64)

	for _, agg := range aggregates {
		switch agg {
		case "count":
			group.Aggregates["count"] = float64(group.Count)
		case "sum":
			// Implementation would depend on the field type
			group.Aggregates["sum"] = 0
		case "avg":
			// Implementation would depend on the field type
			group.Aggregates["avg"] = 0
		}
	}
}

// LogIndexer methods

func (li *LogIndexer) run() {
	defer li.wg.Done()

	ticker := time.NewTicker(li.interval)
	defer ticker.Stop()

	// Initial index
	li.indexFiles()

	for {
		select {
		case <-ticker.C:
			li.indexFiles()
		case <-li.ctx.Done():
			return
		}
	}
}

func (li *LogIndexer) indexFiles() {
	// Walk through log directory
	err := filepath.Walk(li.logDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-log files
		if info.IsDir() || !strings.HasSuffix(path, ".log") {
			return nil
		}

		// Index the file
		li.indexFile(path)
		return nil
	})

	if err != nil {
		// Log error but don't stop indexing
		fmt.Printf("Error indexing log files: %v\n", err)
	}
}

func (li *LogIndexer) indexFile(filePath string) {
	file, err := os.Open(filePath)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Try to parse as JSON log entry
		var entry LogEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue // Skip non-JSON lines
		}

		// Create index entry
		indexEntry := &LogIndexEntry{
			ID:        fmt.Sprintf("%s:%d", filePath, lineNum),
			File:      filePath,
			Line:      lineNum,
			Timestamp: entry.Timestamp,
			Level:     entry.Level,
			Component: entry.Component,
			Message:   entry.Message,
			TraceID:   entry.TraceID,
			Size:      int64(len(line)),
		}

		// Add to index
		li.addToIndex(indexEntry)
	}
}

func (li *LogIndexer) addToIndex(entry *LogIndexEntry) {
	li.index.mu.Lock()
	defer li.index.mu.Unlock()

	// Add to main index
	li.index.entries[entry.ID] = entry

	// Add to time-sorted slice
	li.index.byTime = append(li.index.byTime, entry)

	// Add to level index
	li.index.byLevel[entry.Level] = append(li.index.byLevel[entry.Level], entry)

	// Add to component index
	li.index.byComponent[entry.Component] = append(li.index.byComponent[entry.Component], entry)

	// Add to trace ID index
	if entry.TraceID != "" {
		li.index.byTraceID[entry.TraceID] = append(li.index.byTraceID[entry.TraceID], entry)
	}
}
