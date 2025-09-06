package plugins

import (
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Watcher implements PluginWatcher using fsnotify
type Watcher struct {
	watcher   *fsnotify.Watcher
	directory string
	callbacks []func(string)
	running   bool
	mu        sync.RWMutex
	stopChan  chan struct{}
}

// NewWatcher creates a new plugin watcher
func NewWatcher() (*Watcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	
	return &Watcher{
		watcher:   watcher,
		callbacks: make([]func(string), 0),
		stopChan:  make(chan struct{}),
	}, nil
}

// Watch starts watching a directory for changes
func (w *Watcher) Watch(directory string) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	
	if w.running {
		return nil // Already watching
	}
	
	w.directory = directory
	
	// Add directory to watcher
	if err := w.watcher.Add(directory); err != nil {
		return err
	}
	
	// Start watching goroutine
	go w.watchLoop()
	
	w.running = true
	log.Printf("Started watching directory: %s", directory)
	return nil
}

// Stop stops watching
func (w *Watcher) Stop() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	
	if !w.running {
		return nil
	}
	
	close(w.stopChan)
	w.running = false
	
	if err := w.watcher.Close(); err != nil {
		return err
	}
	
	log.Printf("Stopped watching directory: %s", w.directory)
	return nil
}

// OnChange registers a callback for file changes
func (w *Watcher) OnChange(callback func(string)) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	
	w.callbacks = append(w.callbacks, callback)
	return nil
}

// watchLoop runs the main watching loop
func (w *Watcher) watchLoop() {
	for {
		select {
		case event := <-w.watcher.Events:
			w.handleEvent(event)
		case err := <-w.watcher.Errors:
			log.Printf("Plugin watcher error: %v", err)
		case <-w.stopChan:
			return
		}
	}
}

// handleEvent handles file system events
func (w *Watcher) handleEvent(event fsnotify.Event) {
	// Only handle write events for plugin files
	if event.Op&fsnotify.Write == fsnotify.Write {
		ext := filepath.Ext(event.Name)
		if ext == ".so" || ext == ".go" {
			w.mu.RLock()
			callbacks := make([]func(string), len(w.callbacks))
			copy(callbacks, w.callbacks)
			w.mu.RUnlock()
			
			// Notify all callbacks
			for _, callback := range callbacks {
				go callback(event.Name)
			}
			
			log.Printf("Plugin file changed: %s", event.Name)
		}
	}
}

// MockWatcher implements PluginWatcher for testing
type MockWatcher struct {
	directory string
	callbacks []func(string)
	running   bool
	mu        sync.RWMutex
}

// NewMockWatcher creates a new mock watcher
func NewMockWatcher() *MockWatcher {
	return &MockWatcher{
		callbacks: make([]func(string), 0),
	}
}

// Watch starts watching a directory (mock)
func (mw *MockWatcher) Watch(directory string) error {
	mw.mu.Lock()
	defer mw.mu.Unlock()
	
	mw.directory = directory
	mw.running = true
	
	log.Printf("Mock watcher started watching: %s", directory)
	return nil
}

// Stop stops watching (mock)
func (mw *MockWatcher) Stop() error {
	mw.mu.Lock()
	defer mw.mu.Unlock()
	
	mw.running = false
	
	log.Printf("Mock watcher stopped watching: %s", mw.directory)
	return nil
}

// OnChange registers a callback for file changes (mock)
func (mw *MockWatcher) OnChange(callback func(string)) error {
	mw.mu.Lock()
	defer mw.mu.Unlock()
	
	mw.callbacks = append(mw.callbacks, callback)
	return nil
}

// SimulateChange simulates a file change for testing
func (mw *MockWatcher) SimulateChange(filename string) {
	mw.mu.RLock()
	callbacks := make([]func(string), len(mw.callbacks))
	copy(callbacks, mw.callbacks)
	mw.mu.RUnlock()
	
	for _, callback := range callbacks {
		go callback(filename)
	}
}

// PluginFileManager manages plugin files
type PluginFileManager struct {
	pluginDir string
	mu        sync.RWMutex
}

// NewPluginFileManager creates a new plugin file manager
func NewPluginFileManager(pluginDir string) *PluginFileManager {
	return &PluginFileManager{
		pluginDir: pluginDir,
	}
}

// GetPluginPath returns the full path for a plugin file
func (pfm *PluginFileManager) GetPluginPath(filename string) string {
	return filepath.Join(pfm.pluginDir, filename)
}

// ListPlugins lists all plugin files in the directory
func (pfm *PluginFileManager) ListPlugins() ([]string, error) {
	pfm.mu.RLock()
	defer pfm.mu.RUnlock()
	
	var plugins []string
	
	err := filepath.Walk(pfm.pluginDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		if info.IsDir() {
			return nil
		}
		
		ext := filepath.Ext(path)
		if ext == ".so" || ext == ".go" {
			plugins = append(plugins, path)
		}
		
		return nil
	})
	
	return plugins, err
}

// PluginExists checks if a plugin file exists
func (pfm *PluginFileManager) PluginExists(filename string) bool {
	pfm.mu.RLock()
	defer pfm.mu.RUnlock()
	
	path := pfm.GetPluginPath(filename)
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// GetPluginInfo gets information about a plugin file
func (pfm *PluginFileManager) GetPluginInfo(filename string) (os.FileInfo, error) {
	pfm.mu.RLock()
	defer pfm.mu.RUnlock()
	
	path := pfm.GetPluginPath(filename)
	return os.Stat(path)
}

// PluginRegistryManager manages a registry of available plugins
type PluginRegistryManager struct {
	plugins     map[string]*PluginInfo
	lastUpdated time.Time
	mu          sync.RWMutex
}

// NewPluginRegistryManager creates a new plugin registry manager
func NewPluginRegistryManager() *PluginRegistryManager {
	return &PluginRegistryManager{
		plugins: make(map[string]*PluginInfo),
	}
}

// Register registers a plugin in the registry
func (pr *PluginRegistryManager) Register(info *PluginInfo) {
	pr.mu.Lock()
	defer pr.mu.Unlock()
	
	pr.plugins[info.ID] = info
	pr.lastUpdated = time.Now()
}

// Unregister removes a plugin from the registry
func (pr *PluginRegistryManager) Unregister(id string) {
	pr.mu.Lock()
	defer pr.mu.Unlock()
	
	delete(pr.plugins, id)
	pr.lastUpdated = time.Now()
}

// Get returns a plugin from the registry
func (pr *PluginRegistryManager) Get(id string) (*PluginInfo, bool) {
	pr.mu.RLock()
	defer pr.mu.RUnlock()
	
	info, exists := pr.plugins[id]
	return info, exists
}

// GetAll returns all plugins in the registry
func (pr *PluginRegistryManager) GetAll() map[string]*PluginInfo {
	pr.mu.RLock()
	defer pr.mu.RUnlock()
	
	result := make(map[string]*PluginInfo)
	for id, info := range pr.plugins {
		result[id] = info
	}
	
	return result
}

// GetLastUpdated returns the last update time
func (pr *PluginRegistryManager) GetLastUpdated() time.Time {
	pr.mu.RLock()
	defer pr.mu.RUnlock()
	
	return pr.lastUpdated
}

// PluginHealthChecker checks the health of plugins
type PluginHealthChecker struct {
	plugins map[string]*PluginHealth
	mu      sync.RWMutex
}

// NewPluginHealthChecker creates a new plugin health checker
func NewPluginHealthChecker() *PluginHealthChecker {
	return &PluginHealthChecker{
		plugins: make(map[string]*PluginHealth),
	}
}

// CheckHealth checks the health of a plugin
func (phc *PluginHealthChecker) CheckHealth(plugin *Plugin) *PluginHealth {
	phc.mu.Lock()
	defer phc.mu.Unlock()
	
	health, exists := phc.plugins[plugin.Info.ID]
	if !exists {
		health = &PluginHealth{
			PluginID:  plugin.Info.ID,
			Healthy:   true,
			Status:    "unknown",
			LastCheck: time.Now(),
		}
		phc.plugins[plugin.Info.ID] = health
	}
	
	// Update health status
	health.LastCheck = time.Now()
	
	if plugin.State == PluginStateRunning {
		health.Healthy = true
		health.Status = "running"
		health.Uptime = time.Since(plugin.StartTime)
	} else if plugin.State == PluginStateError {
		health.Healthy = false
		health.Status = "error"
		health.ErrorCount++
		health.LastError = plugin.Error
	} else {
		health.Healthy = false
		health.Status = string(plugin.State)
	}
	
	// Update metrics
	health.MemoryUsage = plugin.Metrics.MemoryUsage
	health.CPUUsage = plugin.Metrics.CPUUsage
	
	return health
}

// GetHealth returns the health status of a plugin
func (phc *PluginHealthChecker) GetHealth(pluginID string) (*PluginHealth, bool) {
	phc.mu.RLock()
	defer phc.mu.RUnlock()
	
	health, exists := phc.plugins[pluginID]
	return health, exists
}

// GetAllHealth returns health status for all plugins
func (phc *PluginHealthChecker) GetAllHealth() map[string]*PluginHealth {
	phc.mu.RLock()
	defer phc.mu.RUnlock()
	
	result := make(map[string]*PluginHealth)
	for id, health := range phc.plugins {
		result[id] = health
	}
	
	return result
}
