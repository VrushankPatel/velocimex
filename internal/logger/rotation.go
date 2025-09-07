package logger

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// RotationStrategy defines how log files should be rotated
type RotationStrategy int

const (
	// RotationBySize rotates logs when they reach a certain size
	RotationBySize RotationStrategy = iota
	// RotationByTime rotates logs at specific time intervals
	RotationByTime
	// RotationBySizeAndTime rotates logs by both size and time
	RotationBySizeAndTime
)

// RotationConfig holds configuration for log rotation
type RotationConfig struct {
	Strategy        RotationStrategy `yaml:"strategy"`
	MaxSizeBytes    int64           `yaml:"max_size_bytes"`
	MaxAge          time.Duration   `yaml:"max_age"`
	MaxBackups      int             `yaml:"max_backups"`
	Compress        bool            `yaml:"compress"`
	LocalTime       bool            `yaml:"local_time"`
	RotateOnStartup bool            `yaml:"rotate_on_startup"`
}

// RotatingWriter implements io.Writer with log rotation
type RotatingWriter struct {
	filename    string
	config      RotationConfig
	file        *os.File
	size        int64
	mu          sync.Mutex
	rotateTime  time.Time
	nextRotate  time.Time
}

// NewRotatingWriter creates a new rotating writer
func NewRotatingWriter(filename string, config RotationConfig) (*RotatingWriter, error) {
	rw := &RotatingWriter{
		filename: filename,
		config:   config,
	}

	// Calculate next rotation time
	if config.Strategy == RotationByTime || config.Strategy == RotationBySizeAndTime {
		rw.calculateNextRotate()
	}

	// Open the current log file
	if err := rw.openFile(); err != nil {
		return nil, err
	}

	// Rotate on startup if configured
	if config.RotateOnStartup {
		if err := rw.rotate(); err != nil {
			return nil, err
		}
	}

	return rw, nil
}

// Write implements io.Writer
func (rw *RotatingWriter) Write(p []byte) (n int, err error) {
	rw.mu.Lock()
	defer rw.mu.Unlock()

	// Check if we need to rotate
	if rw.shouldRotate() {
		if err := rw.rotate(); err != nil {
			return 0, err
		}
	}

	// Write to file
	n, err = rw.file.Write(p)
	rw.size += int64(n)

	return n, err
}

// Close closes the rotating writer
func (rw *RotatingWriter) Close() error {
	rw.mu.Lock()
	defer rw.mu.Unlock()

	if rw.file != nil {
		return rw.file.Close()
	}
	return nil
}

// Sync syncs the current file
func (rw *RotatingWriter) Sync() error {
	rw.mu.Lock()
	defer rw.mu.Unlock()

	if rw.file != nil {
		return rw.file.Sync()
	}
	return nil
}

// shouldRotate checks if the log should be rotated
func (rw *RotatingWriter) shouldRotate() bool {
	switch rw.config.Strategy {
	case RotationBySize:
		return rw.size >= rw.config.MaxSizeBytes
	case RotationByTime:
		return time.Now().After(rw.nextRotate)
	case RotationBySizeAndTime:
		return rw.size >= rw.config.MaxSizeBytes || time.Now().After(rw.nextRotate)
	default:
		return false
	}
}

// rotate performs the log rotation
func (rw *RotatingWriter) rotate() error {
	// Close current file
	if rw.file != nil {
		rw.file.Close()
	}

	// Generate rotated filename
	rotatedFilename := rw.generateRotatedFilename()

	// Rename current file to rotated filename
	if err := os.Rename(rw.filename, rotatedFilename); err != nil {
		return fmt.Errorf("failed to rename log file: %w", err)
	}

	// Compress if configured
	if rw.config.Compress {
		if err := rw.compressFile(rotatedFilename); err != nil {
			return fmt.Errorf("failed to compress log file: %w", err)
		}
	}

	// Clean up old files
	if err := rw.cleanupOldFiles(); err != nil {
		return fmt.Errorf("failed to cleanup old files: %w", err)
	}

	// Open new file
	if err := rw.openFile(); err != nil {
		return err
	}

	// Reset size and calculate next rotation
	rw.size = 0
	if rw.config.Strategy == RotationByTime || rw.config.Strategy == RotationBySizeAndTime {
		rw.calculateNextRotate()
	}

	return nil
}

// openFile opens the current log file
func (rw *RotatingWriter) openFile() error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(rw.filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	// Open file in append mode
	file, err := os.OpenFile(rw.filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	rw.file = file

	// Get current file size
	if stat, err := file.Stat(); err == nil {
		rw.size = stat.Size()
	}

	return nil
}

// generateRotatedFilename generates the filename for the rotated log
func (rw *RotatingWriter) generateRotatedFilename() string {
	now := time.Now()
	if rw.config.LocalTime {
		now = now.Local()
	}

	ext := filepath.Ext(rw.filename)
	name := strings.TrimSuffix(rw.filename, ext)

	// Add timestamp to filename
	timestamp := now.Format("2006-01-02T15-04-05")
	return fmt.Sprintf("%s.%s%s", name, timestamp, ext)
}

// compressFile compresses a log file
func (rw *RotatingWriter) compressFile(filename string) error {
	// Open source file
	src, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer src.Close()

	// Create compressed file
	dst, err := os.Create(filename + ".gz")
	if err != nil {
		return err
	}
	defer dst.Close()

	// Create gzip writer
	gzWriter := gzip.NewWriter(dst)
	defer gzWriter.Close()

	// Copy and compress
	_, err = io.Copy(gzWriter, src)
	if err != nil {
		return err
	}

	// Remove original file
	return os.Remove(filename)
}

// cleanupOldFiles removes old log files based on configuration
func (rw *RotatingWriter) cleanupOldFiles() error {
	dir := filepath.Dir(rw.filename)
	baseName := filepath.Base(rw.filename)
	ext := filepath.Ext(baseName)
	name := strings.TrimSuffix(baseName, ext)

	// Read directory
	files, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	// Filter log files
	var logFiles []os.FileInfo
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		
		fileName := file.Name()
		// Check if it's a log file (original or rotated)
		if fileName == baseName || 
		   strings.HasPrefix(fileName, name+".") && 
		   (strings.HasSuffix(fileName, ext) || strings.HasSuffix(fileName, ext+".gz")) {
			if info, err := file.Info(); err == nil {
				logFiles = append(logFiles, info)
			}
		}
	}

	// Sort by modification time (oldest first)
	sort.Slice(logFiles, func(i, j int) bool {
		return logFiles[i].ModTime().Before(logFiles[j].ModTime())
	})

	// Remove files based on max backups and max age
	cutoffTime := time.Now().Add(-rw.config.MaxAge)
	removedCount := 0

	for _, file := range logFiles {
		// Skip the current log file
		if file.Name() == baseName {
			continue
		}

		shouldRemove := false

		// Check max backups
		if rw.config.MaxBackups > 0 && len(logFiles)-removedCount > rw.config.MaxBackups {
			shouldRemove = true
		}

		// Check max age
		if rw.config.MaxAge > 0 && file.ModTime().Before(cutoffTime) {
			shouldRemove = true
		}

		if shouldRemove {
			fullPath := filepath.Join(dir, file.Name())
			if err := os.Remove(fullPath); err != nil {
				return fmt.Errorf("failed to remove old log file %s: %w", fullPath, err)
			}
			removedCount++
		}
	}

	return nil
}

// calculateNextRotate calculates the next rotation time
func (rw *RotatingWriter) calculateNextRotate() {
	now := time.Now()
	if rw.config.LocalTime {
		now = now.Local()
	}

	// Calculate next rotation based on max age
	if rw.config.MaxAge > 0 {
		rw.nextRotate = now.Add(rw.config.MaxAge)
	} else {
		// Default to daily rotation
		rw.nextRotate = now.Add(24 * time.Hour)
	}
}

// LogRotationManager manages log rotation for multiple files
type LogRotationManager struct {
	writers map[string]*RotatingWriter
	config  RotationConfig
	mu      sync.RWMutex
}

// NewLogRotationManager creates a new log rotation manager
func NewLogRotationManager(config RotationConfig) *LogRotationManager {
	return &LogRotationManager{
		writers: make(map[string]*RotatingWriter),
		config:  config,
	}
}

// GetWriter returns a rotating writer for the given filename
func (lrm *LogRotationManager) GetWriter(filename string) (*RotatingWriter, error) {
	lrm.mu.Lock()
	defer lrm.mu.Unlock()

	if writer, exists := lrm.writers[filename]; exists {
		return writer, nil
	}

	writer, err := NewRotatingWriter(filename, lrm.config)
	if err != nil {
		return nil, err
	}

	lrm.writers[filename] = writer
	return writer, nil
}

// Close closes all rotating writers
func (lrm *LogRotationManager) Close() error {
	lrm.mu.Lock()
	defer lrm.mu.Unlock()

	var lastErr error
	for _, writer := range lrm.writers {
		if err := writer.Close(); err != nil {
			lastErr = err
		}
	}

	return lastErr
}

// Sync syncs all rotating writers
func (lrm *LogRotationManager) Sync() error {
	lrm.mu.RLock()
	defer lrm.mu.RUnlock()

	var lastErr error
	for _, writer := range lrm.writers {
		if err := writer.Sync(); err != nil {
			lastErr = err
		}
	}

	return lastErr
}

// GetRotationConfig returns a default rotation configuration
func GetRotationConfig() RotationConfig {
	return RotationConfig{
		Strategy:        RotationBySizeAndTime,
		MaxSizeBytes:    100 * 1024 * 1024, // 100MB
		MaxAge:          7 * 24 * time.Hour, // 7 days
		MaxBackups:      10,
		Compress:        true,
		LocalTime:       true,
		RotateOnStartup: false,
	}
}

// ParseRotationConfig parses rotation configuration from a map
func ParseRotationConfig(configMap map[string]interface{}) RotationConfig {
	config := GetRotationConfig()

	if strategy, ok := configMap["strategy"].(string); ok {
		switch strings.ToLower(strategy) {
		case "size":
			config.Strategy = RotationBySize
		case "time":
			config.Strategy = RotationByTime
		case "size_and_time":
			config.Strategy = RotationBySizeAndTime
		}
	}

	if maxSize, ok := configMap["max_size_bytes"].(int64); ok {
		config.MaxSizeBytes = maxSize
	}

	if maxAge, ok := configMap["max_age"].(string); ok {
		if duration, err := time.ParseDuration(maxAge); err == nil {
			config.MaxAge = duration
		}
	}

	if maxBackups, ok := configMap["max_backups"].(int); ok {
		config.MaxBackups = maxBackups
	}

	if compress, ok := configMap["compress"].(bool); ok {
		config.Compress = compress
	}

	if localTime, ok := configMap["local_time"].(bool); ok {
		config.LocalTime = localTime
	}

	if rotateOnStartup, ok := configMap["rotate_on_startup"].(bool); ok {
		config.RotateOnStartup = rotateOnStartup
	}

	return config
}
