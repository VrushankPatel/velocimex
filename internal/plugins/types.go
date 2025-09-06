package plugins

import (
	"time"

	"velocimex/internal/orderbook"
	"velocimex/internal/strategy"
)

// PluginState represents the state of a plugin
type PluginState string

const (
	PluginStateLoaded    PluginState = "LOADED"
	PluginStateRunning   PluginState = "RUNNING"
	PluginStateStopped   PluginState = "STOPPED"
	PluginStateError     PluginState = "ERROR"
	PluginStateUnloaded  PluginState = "UNLOADED"
)

// PluginInfo represents information about a plugin
type PluginInfo struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Version     string            `json:"version"`
	Description string            `json:"description"`
	Author      string            `json:"author"`
	License     string            `json:"license"`
	Homepage    string            `json:"homepage"`
	Repository  string            `json:"repository"`
	Tags        []string          `json:"tags"`
	Config      map[string]interface{} `json:"config"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// PluginConfig represents configuration for a plugin
type PluginConfig struct {
	Enabled     bool                   `json:"enabled"`
	AutoStart   bool                   `json:"auto_start"`
	HotReload   bool                   `json:"hot_reload"`
	Sandboxed   bool                   `json:"sandboxed"`
	Timeout     time.Duration          `json:"timeout"`
	MemoryLimit int64                  `json:"memory_limit"` // bytes
	CPULimit    float64                `json:"cpu_limit"`    // percentage
	Settings    map[string]interface{} `json:"settings"`
	Permissions []string               `json:"permissions"`
}

// DefaultPluginConfig returns default plugin configuration
func DefaultPluginConfig() PluginConfig {
	return PluginConfig{
		Enabled:     true,
		AutoStart:   false,
		HotReload:   true,
		Sandboxed:   true,
		Timeout:     30 * time.Second,
		MemoryLimit: 100 * 1024 * 1024, // 100MB
		CPULimit:    50.0,               // 50%
		Settings:    make(map[string]interface{}),
		Permissions: []string{"read_market_data", "generate_signals"},
	}
}

// Plugin represents a loaded plugin
type Plugin struct {
	Info       PluginInfo   `json:"info"`
	Config     PluginConfig `json:"config"`
	State      PluginState  `json:"state"`
	Strategy   strategy.Strategy `json:"-"`
	LoadTime   time.Time    `json:"load_time"`
	StartTime  time.Time    `json:"start_time"`
	StopTime   time.Time    `json:"stop_time"`
	Error      string       `json:"error,omitempty"`
	Metrics    PluginMetrics `json:"metrics"`
}

// PluginMetrics represents performance metrics for a plugin
type PluginMetrics struct {
	ExecutionTime    time.Duration `json:"execution_time"`
	MemoryUsage      int64         `json:"memory_usage"`
	CPUUsage         float64       `json:"cpu_usage"`
	SignalsGenerated int           `json:"signals_generated"`
	ErrorsCount      int           `json:"errors_count"`
	LastExecution    time.Time     `json:"last_execution"`
	Uptime           time.Duration `json:"uptime"`
}

// PluginManager defines the interface for plugin management
type PluginManager interface {
	// Plugin lifecycle
	LoadPlugin(path string) (*Plugin, error)
	UnloadPlugin(id string) error
	StartPlugin(id string) error
	StopPlugin(id string) error
	RestartPlugin(id string) error
	
	// Plugin discovery
	DiscoverPlugins(directory string) ([]string, error)
	GetPlugin(id string) (*Plugin, error)
	GetAllPlugins() map[string]*Plugin
	
	// Plugin management
	EnablePlugin(id string) error
	DisablePlugin(id string) error
	UpdatePluginConfig(id string, config PluginConfig) error
	
	// Hot reload
	EnableHotReload() error
	DisableHotReload() error
	ReloadPlugin(id string) error
	
	// Plugin validation
	ValidatePlugin(path string) error
	GetPluginInfo(path string) (*PluginInfo, error)
	
	// Control
	Start() error
	Stop() error
	IsRunning() bool
}

// PluginLoader defines the interface for loading plugins
type PluginLoader interface {
	Load(path string) (strategy.Strategy, *PluginInfo, error)
	Unload(strategy.Strategy) error
	Validate(path string) error
	GetInfo(path string) (*PluginInfo, error)
}

// PluginWatcher defines the interface for watching plugin files
type PluginWatcher interface {
	Watch(directory string) error
	Stop() error
	OnChange(callback func(string)) error
}

// PluginSandbox defines the interface for sandboxing plugins
type PluginSandbox interface {
	CreateSandbox(config PluginConfig) (Sandbox, error)
	DestroySandbox(sandbox Sandbox) error
}

// Sandbox represents a sandboxed environment for plugins
type Sandbox interface {
	Execute(fn func() error) error
	GetMemoryUsage() int64
	GetCPUUsage() float64
	SetMemoryLimit(limit int64) error
	SetCPULimit(limit float64) error
	IsHealthy() bool
}

// PluginEvent represents an event related to plugins
type PluginEvent struct {
	Type      string      `json:"type"`
	PluginID  string      `json:"plugin_id"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data,omitempty"`
}

// PluginEventType represents types of plugin events
type PluginEventType string

const (
	PluginEventLoaded   PluginEventType = "PLUGIN_LOADED"
	PluginEventUnloaded PluginEventType = "PLUGIN_UNLOADED"
	PluginEventStarted  PluginEventType = "PLUGIN_STARTED"
	PluginEventStopped  PluginEventType = "PLUGIN_STOPPED"
	PluginEventError    PluginEventType = "PLUGIN_ERROR"
	PluginEventReloaded PluginEventType = "PLUGIN_RELOADED"
	PluginEventConfigUpdated PluginEventType = "PLUGIN_CONFIG_UPDATED"
)

// PluginRegistry represents a registry of available plugins
type PluginRegistry struct {
	Plugins map[string]*PluginInfo `json:"plugins"`
	LastUpdated time.Time          `json:"last_updated"`
}

// PluginAPI defines the API available to plugins
type PluginAPI interface {
	// Market data
	GetOrderBook(symbol, exchange string) (*orderbook.OrderBook, error)
	GetAllOrderBooks() map[string]*orderbook.OrderBook
	
	// Configuration
	GetConfig() map[string]interface{}
	SetConfig(key string, value interface{}) error
	
	// Logging
	Log(level string, message string, args ...interface{})
	
	// Metrics
	RecordMetric(name string, value float64, tags map[string]string)
	
	// Events
	EmitEvent(eventType string, data interface{})
	
	// Utilities
	GetCurrentTime() time.Time
	Sleep(duration time.Duration)
}

// PluginContext represents the context passed to plugins
type PluginContext struct {
	API     PluginAPI              `json:"-"`
	Config  map[string]interface{} `json:"config"`
	Logger  PluginLogger           `json:"-"`
	Metrics PluginMetricsCollector `json:"-"`
}

// PluginLogger defines the logging interface for plugins
type PluginLogger interface {
	Debug(message string, args ...interface{})
	Info(message string, args ...interface{})
	Warn(message string, args ...interface{})
	Error(message string, args ...interface{})
	Fatal(message string, args ...interface{})
}

// PluginMetricsCollector defines the metrics collection interface for plugins
type PluginMetricsCollector interface {
	Counter(name string, value float64, tags map[string]string)
	Gauge(name string, value float64, tags map[string]string)
	Histogram(name string, value float64, tags map[string]string)
	Timer(name string, duration time.Duration, tags map[string]string)
}

// PluginManifest represents the manifest file for a plugin
type PluginManifest struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Version     string                 `json:"version"`
	Description string                 `json:"description"`
	Author      string                 `json:"author"`
	License     string                 `json:"license"`
	Homepage    string                 `json:"homepage"`
	Repository  string                 `json:"repository"`
	Tags        []string               `json:"tags"`
	Main        string                 `json:"main"`
	Config      map[string]interface{} `json:"config"`
	Permissions []string               `json:"permissions"`
	Dependencies map[string]string     `json:"dependencies"`
	MinVersion  string                 `json:"min_version"`
	MaxVersion  string                 `json:"max_version"`
}

// PluginError represents an error from a plugin
type PluginError struct {
	PluginID  string    `json:"plugin_id"`
	Error     string    `json:"error"`
	Timestamp time.Time `json:"timestamp"`
	Severity  string    `json:"severity"`
	Context   map[string]interface{} `json:"context"`
}

// PluginHealth represents the health status of a plugin
type PluginHealth struct {
	PluginID    string        `json:"plugin_id"`
	Healthy     bool          `json:"healthy"`
	Status      string        `json:"status"`
	LastCheck   time.Time     `json:"last_check"`
	Uptime      time.Duration `json:"uptime"`
	MemoryUsage int64         `json:"memory_usage"`
	CPUUsage    float64       `json:"cpu_usage"`
	ErrorCount  int           `json:"error_count"`
	LastError   string        `json:"last_error,omitempty"`
}

// PluginDependency represents a dependency for a plugin
type PluginDependency struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Type    string `json:"type"` // "required", "optional", "conflicts"
}

// PluginTemplate represents a template for creating new plugins
type PluginTemplate struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Language    string                 `json:"language"`
	Template    map[string]interface{} `json:"template"`
	Files       map[string]string      `json:"files"`
	Config      map[string]interface{} `json:"config"`
}
