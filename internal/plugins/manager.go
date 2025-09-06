package plugins

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Manager implements the PluginManager interface
type Manager struct {
	plugins      map[string]*Plugin
	loaders      map[string]PluginLoader
	watcher      PluginWatcher
	sandbox      PluginSandbox
	hotReload    bool
	running      bool
	mu           sync.RWMutex
	ctx          context.Context
	cancel       context.CancelFunc
	eventChan    chan PluginEvent
	subscribers  []func(PluginEvent)
}

// NewManager creates a new plugin manager
func NewManager() *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	return &Manager{
		plugins:     make(map[string]*Plugin),
		loaders:     make(map[string]PluginLoader),
		eventChan:   make(chan PluginEvent, 100),
		subscribers: make([]func(PluginEvent), 0),
		ctx:         ctx,
		cancel:      cancel,
	}
}

// LoadPlugin loads a plugin from the specified path
func (pm *Manager) LoadPlugin(path string) (*Plugin, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	// Validate plugin
	if err := pm.ValidatePlugin(path); err != nil {
		return nil, fmt.Errorf("plugin validation failed: %v", err)
	}
	
	// Get plugin info
	info, err := pm.GetPluginInfo(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get plugin info: %v", err)
	}
	
	// Check if plugin is already loaded
	if existing, exists := pm.plugins[info.ID]; exists {
		return existing, fmt.Errorf("plugin %s is already loaded", info.ID)
	}
	
	// Determine loader based on file extension
	ext := filepath.Ext(path)
	loader, exists := pm.loaders[ext]
	if !exists {
		return nil, fmt.Errorf("no loader found for extension %s", ext)
	}
	
	// Load the strategy
	strategy, info, err := loader.Load(path)
	if err != nil {
		return nil, fmt.Errorf("failed to load plugin: %v", err)
	}
	
	// Create plugin instance
	plugin := &Plugin{
		Info:     *info,
		Config:   DefaultPluginConfig(),
		State:    PluginStateLoaded,
		Strategy: strategy,
		LoadTime: time.Now(),
		Metrics:  PluginMetrics{},
	}
	
	pm.plugins[info.ID] = plugin
	
	// Emit event
	pm.emitEvent(PluginEvent{
		Type:      string(PluginEventLoaded),
		PluginID:  info.ID,
		Timestamp: time.Now(),
		Data:      plugin,
	})
	
	log.Printf("Plugin %s loaded successfully", info.ID)
	return plugin, nil
}

// UnloadPlugin unloads a plugin by ID
func (pm *Manager) UnloadPlugin(id string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	plugin, exists := pm.plugins[id]
	if !exists {
		return fmt.Errorf("plugin %s not found", id)
	}
	
	// Stop plugin if running
	if plugin.State == PluginStateRunning {
		if err := pm.stopPluginInternal(plugin); err != nil {
			log.Printf("Error stopping plugin %s: %v", id, err)
		}
	}
	
	// Unload strategy
	if plugin.Strategy != nil {
		ext := filepath.Ext(plugin.Info.Name)
		if loader, exists := pm.loaders[ext]; exists {
			if err := loader.Unload(plugin.Strategy); err != nil {
				log.Printf("Error unloading strategy: %v", err)
			}
		}
	}
	
	// Remove from registry
	delete(pm.plugins, id)
	
	// Emit event
	pm.emitEvent(PluginEvent{
		Type:      string(PluginEventUnloaded),
		PluginID:  id,
		Timestamp: time.Now(),
	})
	
	log.Printf("Plugin %s unloaded successfully", id)
	return nil
}

// StartPlugin starts a plugin by ID
func (pm *Manager) StartPlugin(id string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	plugin, exists := pm.plugins[id]
	if !exists {
		return fmt.Errorf("plugin %s not found", id)
	}
	
	if plugin.State == PluginStateRunning {
		return fmt.Errorf("plugin %s is already running", id)
	}
	
	if !plugin.Config.Enabled {
		return fmt.Errorf("plugin %s is disabled", id)
	}
	
	return pm.startPluginInternal(plugin)
}

// StopPlugin stops a plugin by ID
func (pm *Manager) StopPlugin(id string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	plugin, exists := pm.plugins[id]
	if !exists {
		return fmt.Errorf("plugin %s not found", id)
	}
	
	if plugin.State != PluginStateRunning {
		return fmt.Errorf("plugin %s is not running", id)
	}
	
	return pm.stopPluginInternal(plugin)
}

// RestartPlugin restarts a plugin by ID
func (pm *Manager) RestartPlugin(id string) error {
	if err := pm.StopPlugin(id); err != nil {
		return err
	}
	
	time.Sleep(100 * time.Millisecond) // Small delay
	
	return pm.StartPlugin(id)
}

// DiscoverPlugins discovers plugins in a directory
func (pm *Manager) DiscoverPlugins(directory string) ([]string, error) {
	var plugins []string
	
	err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		if info.IsDir() {
			return nil
		}
		
		// Check if file has a supported extension
		ext := filepath.Ext(path)
		if _, exists := pm.loaders[ext]; exists {
			plugins = append(plugins, path)
		}
		
		return nil
	})
	
	return plugins, err
}

// GetPlugin returns a plugin by ID
func (pm *Manager) GetPlugin(id string) (*Plugin, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	
	plugin, exists := pm.plugins[id]
	if !exists {
		return nil, fmt.Errorf("plugin %s not found", id)
	}
	
	return plugin, nil
}

// GetAllPlugins returns all loaded plugins
func (pm *Manager) GetAllPlugins() map[string]*Plugin {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	
	// Return a copy to prevent external modification
	result := make(map[string]*Plugin)
	for id, plugin := range pm.plugins {
		result[id] = plugin
	}
	
	return result
}

// EnablePlugin enables a plugin
func (pm *Manager) EnablePlugin(id string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	plugin, exists := pm.plugins[id]
	if !exists {
		return fmt.Errorf("plugin %s not found", id)
	}
	
	plugin.Config.Enabled = true
	
	// Emit event
	pm.emitEvent(PluginEvent{
		Type:      string(PluginEventConfigUpdated),
		PluginID:  id,
		Timestamp: time.Now(),
		Data:      plugin.Config,
	})
	
	return nil
}

// DisablePlugin disables a plugin
func (pm *Manager) DisablePlugin(id string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	plugin, exists := pm.plugins[id]
	if !exists {
		return fmt.Errorf("plugin %s not found", id)
	}
	
	plugin.Config.Enabled = false
	
	// Stop plugin if running
	if plugin.State == PluginStateRunning {
		if err := pm.stopPluginInternal(plugin); err != nil {
			log.Printf("Error stopping plugin %s: %v", id, err)
		}
	}
	
	// Emit event
	pm.emitEvent(PluginEvent{
		Type:      string(PluginEventConfigUpdated),
		PluginID:  id,
		Timestamp: time.Now(),
		Data:      plugin.Config,
	})
	
	return nil
}

// UpdatePluginConfig updates plugin configuration
func (pm *Manager) UpdatePluginConfig(id string, config PluginConfig) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	plugin, exists := pm.plugins[id]
	if !exists {
		return fmt.Errorf("plugin %s not found", id)
	}
	
	plugin.Config = config
	
	// Emit event
	pm.emitEvent(PluginEvent{
		Type:      string(PluginEventConfigUpdated),
		PluginID:  id,
		Timestamp: time.Now(),
		Data:      config,
	})
	
	return nil
}

// EnableHotReload enables hot reload functionality
func (pm *Manager) EnableHotReload() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	if pm.hotReload {
		return nil // Already enabled
	}
	
	pm.hotReload = true
	
	if pm.watcher != nil {
		return pm.watcher.Watch("plugins")
	}
	
	return nil
}

// DisableHotReload disables hot reload functionality
func (pm *Manager) DisableHotReload() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	if !pm.hotReload {
		return nil // Already disabled
	}
	
	pm.hotReload = false
	
	if pm.watcher != nil {
		return pm.watcher.Stop()
	}
	
	return nil
}

// ReloadPlugin reloads a plugin
func (pm *Manager) ReloadPlugin(id string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	plugin, exists := pm.plugins[id]
	if !exists {
		return fmt.Errorf("plugin %s not found", id)
	}
	
	// Stop plugin if running
	if plugin.State == PluginStateRunning {
		if err := pm.stopPluginInternal(plugin); err != nil {
			log.Printf("Error stopping plugin %s: %v", id, err)
		}
	}
	
	// Reload the plugin
	// This is a simplified implementation - in practice you'd reload from file
	plugin.LoadTime = time.Now()
	
	// Emit event
	pm.emitEvent(PluginEvent{
		Type:      string(PluginEventReloaded),
		PluginID:  id,
		Timestamp: time.Now(),
	})
	
	log.Printf("Plugin %s reloaded successfully", id)
	return nil
}

// ValidatePlugin validates a plugin
func (pm *Manager) ValidatePlugin(path string) error {
	ext := filepath.Ext(path)
	loader, exists := pm.loaders[ext]
	if !exists {
		return fmt.Errorf("no loader found for extension %s", ext)
	}
	
	return loader.Validate(path)
}

// GetPluginInfo returns plugin information
func (pm *Manager) GetPluginInfo(path string) (*PluginInfo, error) {
	ext := filepath.Ext(path)
	loader, exists := pm.loaders[ext]
	if !exists {
		return nil, fmt.Errorf("no loader found for extension %s", ext)
	}
	
	return loader.GetInfo(path)
}

// Start starts the plugin manager
func (pm *Manager) Start() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	if pm.running {
		return fmt.Errorf("plugin manager already running")
	}
	
	pm.running = true
	
	// Start event processing
	go pm.processEvents()
	
	// Auto-start enabled plugins
	for _, plugin := range pm.plugins {
		if plugin.Config.AutoStart && plugin.Config.Enabled {
			go func(p *Plugin) {
				if err := pm.startPluginInternal(p); err != nil {
					log.Printf("Failed to auto-start plugin %s: %v", p.Info.ID, err)
				}
			}(plugin)
		}
	}
	
	log.Println("Plugin manager started")
	return nil
}

// Stop stops the plugin manager
func (pm *Manager) Stop() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	if !pm.running {
		return nil
	}
	
	pm.running = false
	pm.cancel()
	
	// Stop all running plugins
	for _, plugin := range pm.plugins {
		if plugin.State == PluginStateRunning {
			if err := pm.stopPluginInternal(plugin); err != nil {
				log.Printf("Error stopping plugin %s: %v", plugin.Info.ID, err)
			}
		}
	}
	
	log.Println("Plugin manager stopped")
	return nil
}

// IsRunning returns whether the manager is running
func (pm *Manager) IsRunning() bool {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.running
}

// RegisterLoader registers a plugin loader
func (pm *Manager) RegisterLoader(extension string, loader PluginLoader) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	pm.loaders[extension] = loader
}

// SubscribeToEvents subscribes to plugin events
func (pm *Manager) SubscribeToEvents(callback func(PluginEvent)) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	pm.subscribers = append(pm.subscribers, callback)
}

// Private methods

func (pm *Manager) startPluginInternal(plugin *Plugin) error {
	if plugin.Strategy == nil {
		return fmt.Errorf("plugin strategy is nil")
	}
	
	// Start strategy
	if err := plugin.Strategy.Start(pm.ctx); err != nil {
		plugin.State = PluginStateError
		plugin.Error = err.Error()
		return fmt.Errorf("failed to start strategy: %v", err)
	}
	
	plugin.State = PluginStateRunning
	plugin.StartTime = time.Now()
	plugin.Error = ""
	
	// Emit event
	pm.emitEvent(PluginEvent{
		Type:      string(PluginEventStarted),
		PluginID:  plugin.Info.ID,
		Timestamp: time.Now(),
	})
	
	log.Printf("Plugin %s started successfully", plugin.Info.ID)
	return nil
}

func (pm *Manager) stopPluginInternal(plugin *Plugin) error {
	if plugin.Strategy == nil {
		return fmt.Errorf("plugin strategy is nil")
	}
	
	// Stop strategy
	if err := plugin.Strategy.Stop(); err != nil {
		log.Printf("Error stopping strategy: %v", err)
	}
	
	plugin.State = PluginStateStopped
	plugin.StopTime = time.Now()
	
	// Emit event
	pm.emitEvent(PluginEvent{
		Type:      string(PluginEventStopped),
		PluginID:  plugin.Info.ID,
		Timestamp: time.Now(),
	})
	
	log.Printf("Plugin %s stopped successfully", plugin.Info.ID)
	return nil
}

func (pm *Manager) emitEvent(event PluginEvent) {
	select {
	case pm.eventChan <- event:
	default:
		// Channel is full, drop event
		log.Printf("Plugin event channel full, dropping event: %s", event.Type)
	}
}

func (pm *Manager) processEvents() {
	for {
		select {
		case event := <-pm.eventChan:
			pm.mu.RLock()
			subscribers := make([]func(PluginEvent), len(pm.subscribers))
			copy(subscribers, pm.subscribers)
			pm.mu.RUnlock()
			
			for _, callback := range subscribers {
				go callback(event)
			}
		case <-pm.ctx.Done():
			return
		}
	}
}
