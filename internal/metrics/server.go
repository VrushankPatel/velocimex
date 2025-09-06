package metrics

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Server represents a Prometheus metrics server
type Server struct {
	server   *http.Server
	registry *prometheus.Registry
	metrics  *Metrics
	addr     string
}

// ServerConfig represents configuration for the metrics server
type ServerConfig struct {
	Enabled     bool          `yaml:"enabled"`
	Address     string        `yaml:"address"`
	Port        int           `yaml:"port"`
	Path        string        `yaml:"path"`
	Timeout     time.Duration `yaml:"timeout"`
	EnablePprof bool          `yaml:"enable_pprof"`
}

// DefaultServerConfig returns default server configuration
func DefaultServerConfig() ServerConfig {
	return ServerConfig{
		Enabled:     true,
		Address:     "0.0.0.0",
		Port:        9090,
		Path:        "/metrics",
		Timeout:     30 * time.Second,
		EnablePprof: false,
	}
}

// NewServer creates a new Prometheus metrics server
func NewServer(config ServerConfig, metrics *Metrics) *Server {
	addr := fmt.Sprintf("%s:%d", config.Address, config.Port)
	
	mux := http.NewServeMux()
	
	// Add metrics endpoint
	mux.Handle(config.Path, promhttp.HandlerFor(metrics.GetRegistry(), promhttp.HandlerOpts{
		EnableOpenMetrics: true,
	}))
	
	// Add health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	
	// Add ready check endpoint
	mux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	
	// Add pprof endpoints if enabled
	if config.EnablePprof {
		mux.HandleFunc("/debug/pprof/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.DefaultServeMux.ServeHTTP(w, r)
		}))
	}
	
	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  config.Timeout,
		WriteTimeout: config.Timeout,
		IdleTimeout:  config.Timeout,
	}
	
	return &Server{
		server:   server,
		registry: metrics.GetRegistry(),
		metrics:  metrics,
		addr:     addr,
	}
}

// Start starts the metrics server
func (s *Server) Start(ctx context.Context) error {
	log.Printf("Starting Prometheus metrics server on %s", s.addr)
	
	go func() {
		<-ctx.Done()
		log.Println("Shutting down Prometheus metrics server...")
		
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		if err := s.server.Shutdown(shutdownCtx); err != nil {
			log.Printf("Error shutting down metrics server: %v", err)
		}
	}()
	
	return s.server.ListenAndServe()
}

// Stop stops the metrics server
func (s *Server) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	return s.server.Shutdown(ctx)
}

// GetAddress returns the server address
func (s *Server) GetAddress() string {
	return s.addr
}

// GetRegistry returns the Prometheus registry
func (s *Server) GetRegistry() *prometheus.Registry {
	return s.registry
}

// MetricsCollector represents a collector for custom metrics
type MetricsCollector struct {
	metrics map[string]prometheus.Collector
	mu      sync.RWMutex
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		metrics: make(map[string]prometheus.Collector),
	}
}

// Register registers a custom metric
func (mc *MetricsCollector) Register(name string, collector prometheus.Collector) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	if _, exists := mc.metrics[name]; exists {
		return fmt.Errorf("metric %s already registered", name)
	}
	
	mc.metrics[name] = collector
	return nil
}

// Unregister unregisters a custom metric
func (mc *MetricsCollector) Unregister(name string) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	delete(mc.metrics, name)
	return nil
}

// GetMetric returns a registered metric
func (mc *MetricsCollector) GetMetric(name string) (prometheus.Collector, bool) {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	
	collector, exists := mc.metrics[name]
	return collector, exists
}

// GetAllMetrics returns all registered metrics
func (mc *MetricsCollector) GetAllMetrics() map[string]prometheus.Collector {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	
	result := make(map[string]prometheus.Collector)
	for name, collector := range mc.metrics {
		result[name] = collector
	}
	
	return result
}

// MetricsMiddleware provides HTTP middleware for metrics collection
type MetricsMiddleware struct {
	metrics *Metrics
}

// NewMetricsMiddleware creates a new metrics middleware
func NewMetricsMiddleware(metrics *Metrics) *MetricsMiddleware {
	return &MetricsMiddleware{
		metrics: metrics,
	}
}

// Handler wraps an HTTP handler with metrics collection
func (mm *MetricsMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		// Create a response writer wrapper to capture status code
		wrapper := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		
		// Call the next handler
		next.ServeHTTP(wrapper, r)
		
		// Record metrics
		duration := time.Since(start)
		endpoint := r.URL.Path
		method := r.Method
		status := fmt.Sprintf("%d", wrapper.statusCode)
		
		mm.metrics.RecordAPIRequest(endpoint, method, status)
		mm.metrics.RecordAPILatency(endpoint, method, duration)
		
		if wrapper.statusCode >= 400 {
			mm.metrics.RecordAPIError(endpoint, method, "http_error")
		}
	})
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// MetricsExporter provides functionality to export metrics
type MetricsExporter struct {
	registry *prometheus.Registry
}

// NewMetricsExporter creates a new metrics exporter
func NewMetricsExporter(registry *prometheus.Registry) *MetricsExporter {
	return &MetricsExporter{
		registry: registry,
	}
}

// ExportToJSON exports metrics to JSON format
func (me *MetricsExporter) ExportToJSON() ([]byte, error) {
	// This would require implementing a custom JSON exporter
	// For now, we'll return a placeholder
	return []byte(`{"message": "JSON export not implemented"}`), nil
}

// ExportToCSV exports metrics to CSV format
func (me *MetricsExporter) ExportToCSV() ([]byte, error) {
	// This would require implementing a custom CSV exporter
	// For now, we'll return a placeholder
	return []byte(`metric_name,value,timestamp`), nil
}

// GetMetricsSnapshot returns a snapshot of current metrics
func (me *MetricsExporter) GetMetricsSnapshot() map[string]interface{} {
	// This would require implementing metric collection
	// For now, we'll return a placeholder
	return map[string]interface{}{
		"timestamp": time.Now(),
		"metrics":   "snapshot not implemented",
	}
}

// MetricsValidator validates metrics configuration
type MetricsValidator struct{}

// NewMetricsValidator creates a new metrics validator
func NewMetricsValidator() *MetricsValidator {
	return &MetricsValidator{}
}

// ValidateConfig validates metrics server configuration
func (mv *MetricsValidator) ValidateConfig(config ServerConfig) error {
	if config.Port < 1 || config.Port > 65535 {
		return fmt.Errorf("invalid port: %d", config.Port)
	}
	
	if config.Address == "" {
		return fmt.Errorf("address cannot be empty")
	}
	
	if config.Path == "" {
		return fmt.Errorf("path cannot be empty")
	}
	
	if config.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive")
	}
	
	return nil
}

// ValidateMetricName validates a metric name
func (mv *MetricsValidator) ValidateMetricName(name string) error {
	if name == "" {
		return fmt.Errorf("metric name cannot be empty")
	}
	
	// Add more validation rules as needed
	return nil
}
