package metrics

// Config holds metrics configuration
type Config struct {
	Enabled bool   `yaml:"enabled"`
	Port    string `yaml:"port"`
	Path    string `yaml:"path"`
}

// DefaultConfig returns default metrics configuration
func DefaultConfig() Config {
	return Config{
		Enabled: true,
		Port:    "9090",
		Path:    "/metrics",
	}
}

// Validate validates the metrics configuration
func (c *Config) Validate() error {
	if !c.Enabled {
		return nil
	}
	
	if c.Port == "" {
		c.Port = "9090"
	}
	
	if c.Path == "" {
		c.Path = "/metrics"
	}
	
	return nil
}