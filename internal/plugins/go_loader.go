package plugins

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"plugin"
	"time"

	"velocimex/internal/strategy"
)

// GoLoader implements PluginLoader for Go plugins
type GoLoader struct {
	loadedPlugins map[string]*plugin.Plugin
}

// NewGoLoader creates a new Go plugin loader
func NewGoLoader() *GoLoader {
	return &GoLoader{
		loadedPlugins: make(map[string]*plugin.Plugin),
	}
}

// Load loads a Go plugin
func (gl *GoLoader) Load(path string) (strategy.Strategy, *PluginInfo, error) {
	// Check if plugin is already loaded
	if p, exists := gl.loadedPlugins[path]; exists {
		return gl.createStrategyFromPlugin(p, path)
	}
	
	// Load the plugin
	p, err := plugin.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open plugin: %v", err)
	}
	
	// Store loaded plugin
	gl.loadedPlugins[path] = p
	
	// Create strategy from plugin
	return gl.createStrategyFromPlugin(p, path)
}

// Unload unloads a Go plugin
func (gl *GoLoader) Unload(strategy strategy.Strategy) error {
	// Go plugins cannot be unloaded dynamically
	// This is a limitation of Go's plugin system
	log.Printf("Warning: Go plugins cannot be unloaded dynamically")
	return nil
}

// Validate validates a Go plugin
func (gl *GoLoader) Validate(path string) error {
	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("plugin file does not exist: %s", path)
	}
	
	// Check file extension
	if filepath.Ext(path) != ".so" {
		return fmt.Errorf("Go plugins must have .so extension")
	}
	
	// Try to open the plugin to validate it
	p, err := plugin.Open(path)
	if err != nil {
		return fmt.Errorf("plugin validation failed: %v", err)
	}
	
	// Check for required symbols
	if err := gl.validatePluginSymbols(p); err != nil {
		return fmt.Errorf("plugin symbol validation failed: %v", err)
	}
	
	return nil
}

// GetInfo returns plugin information
func (gl *GoLoader) GetInfo(path string) (*PluginInfo, error) {
	// Load plugin temporarily to get info
	p, err := plugin.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open plugin: %v", err)
	}
	
	// Get plugin info symbol
	infoSymbol, err := p.Lookup("PluginInfo")
	if err != nil {
		return nil, fmt.Errorf("plugin info symbol not found: %v", err)
	}
	
	// Cast to PluginInfo
	info, ok := infoSymbol.(*PluginInfo)
	if !ok {
		return nil, fmt.Errorf("invalid plugin info type")
	}
	
	return info, nil
}

// createStrategyFromPlugin creates a strategy from a loaded plugin
func (gl *GoLoader) createStrategyFromPlugin(p *plugin.Plugin, path string) (strategy.Strategy, *PluginInfo, error) {
	// Get plugin info
	infoSymbol, err := p.Lookup("PluginInfo")
	if err != nil {
		return nil, nil, fmt.Errorf("plugin info symbol not found: %v", err)
	}
	
	info, ok := infoSymbol.(*PluginInfo)
	if !ok {
		return nil, nil, fmt.Errorf("invalid plugin info type")
	}
	
	// Get strategy constructor symbol
	constructorSymbol, err := p.Lookup("NewStrategy")
	if err != nil {
		return nil, nil, fmt.Errorf("strategy constructor symbol not found: %v", err)
	}
	
	// Cast to constructor function
	constructor, ok := constructorSymbol.(func() strategy.Strategy)
	if !ok {
		return nil, nil, fmt.Errorf("invalid strategy constructor type")
	}
	
	// Create strategy instance
	strategyInstance := constructor()
	
	return strategyInstance, info, nil
}

// validatePluginSymbols validates that required symbols exist in the plugin
func (gl *GoLoader) validatePluginSymbols(p *plugin.Plugin) error {
	// Check for PluginInfo symbol
	if _, err := p.Lookup("PluginInfo"); err != nil {
		return fmt.Errorf("PluginInfo symbol not found: %v", err)
	}
	
	// Check for NewStrategy symbol
	if _, err := p.Lookup("NewStrategy"); err != nil {
		return fmt.Errorf("NewStrategy symbol not found: %v", err)
	}
	
	return nil
}

// MockGoLoader implements PluginLoader for testing
type MockGoLoader struct {
	strategies map[string]strategy.Strategy
	infos      map[string]*PluginInfo
}

// NewMockGoLoader creates a new mock Go loader
func NewMockGoLoader() *MockGoLoader {
	return &MockGoLoader{
		strategies: make(map[string]strategy.Strategy),
		infos:      make(map[string]*PluginInfo),
	}
}

// Load loads a mock plugin
func (mgl *MockGoLoader) Load(path string) (strategy.Strategy, *PluginInfo, error) {
	strategy, exists := mgl.strategies[path]
	if !exists {
		return nil, nil, fmt.Errorf("mock plugin not found: %s", path)
	}
	
	info, exists := mgl.infos[path]
	if !exists {
		return nil, nil, fmt.Errorf("mock plugin info not found: %s", path)
	}
	
	return strategy, info, nil
}

// Unload unloads a mock plugin
func (mgl *MockGoLoader) Unload(strategy strategy.Strategy) error {
	// Remove from mock registry
	for path, s := range mgl.strategies {
		if s == strategy {
			delete(mgl.strategies, path)
			delete(mgl.infos, path)
			break
		}
	}
	return nil
}

// Validate validates a mock plugin
func (mgl *MockGoLoader) Validate(path string) error {
	_, exists := mgl.strategies[path]
	if !exists {
		return fmt.Errorf("mock plugin not found: %s", path)
	}
	return nil
}

// GetInfo returns mock plugin information
func (mgl *MockGoLoader) GetInfo(path string) (*PluginInfo, error) {
	info, exists := mgl.infos[path]
	if !exists {
		return nil, fmt.Errorf("mock plugin info not found: %s", path)
	}
	return info, nil
}

// RegisterMockPlugin registers a mock plugin for testing
func (mgl *MockGoLoader) RegisterMockPlugin(path string, strategy strategy.Strategy, info *PluginInfo) {
	mgl.strategies[path] = strategy
	mgl.infos[path] = info
}

// PluginBuilder helps build plugins programmatically
type PluginBuilder struct {
	info     *PluginInfo
	strategy strategy.Strategy
}

// NewPluginBuilder creates a new plugin builder
func NewPluginBuilder() *PluginBuilder {
	return &PluginBuilder{
		info: &PluginInfo{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Config:    make(map[string]interface{}),
		},
	}
}

// SetID sets the plugin ID
func (pb *PluginBuilder) SetID(id string) *PluginBuilder {
	pb.info.ID = id
	return pb
}

// SetName sets the plugin name
func (pb *PluginBuilder) SetName(name string) *PluginBuilder {
	pb.info.Name = name
	return pb
}

// SetVersion sets the plugin version
func (pb *PluginBuilder) SetVersion(version string) *PluginBuilder {
	pb.info.Version = version
	return pb
}

// SetDescription sets the plugin description
func (pb *PluginBuilder) SetDescription(description string) *PluginBuilder {
	pb.info.Description = description
	return pb
}

// SetAuthor sets the plugin author
func (pb *PluginBuilder) SetAuthor(author string) *PluginBuilder {
	pb.info.Author = author
	return pb
}

// SetLicense sets the plugin license
func (pb *PluginBuilder) SetLicense(license string) *PluginBuilder {
	pb.info.License = license
	return pb
}

// SetHomepage sets the plugin homepage
func (pb *PluginBuilder) SetHomepage(homepage string) *PluginBuilder {
	pb.info.Homepage = homepage
	return pb
}

// SetRepository sets the plugin repository
func (pb *PluginBuilder) SetRepository(repository string) *PluginBuilder {
	pb.info.Repository = repository
	return pb
}

// SetTags sets the plugin tags
func (pb *PluginBuilder) SetTags(tags []string) *PluginBuilder {
	pb.info.Tags = tags
	return pb
}

// SetConfig sets the plugin configuration
func (pb *PluginBuilder) SetConfig(config map[string]interface{}) *PluginBuilder {
	pb.info.Config = config
	return pb
}

// SetStrategy sets the plugin strategy
func (pb *PluginBuilder) SetStrategy(strategy strategy.Strategy) *PluginBuilder {
	pb.strategy = strategy
	return pb
}

// Build builds the plugin
func (pb *PluginBuilder) Build() (*PluginInfo, strategy.Strategy) {
	pb.info.UpdatedAt = time.Now()
	return pb.info, pb.strategy
}
