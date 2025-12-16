package middleware

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the middleware server configuration
type Config struct {
	Server     ServerConfig     `yaml:"server"`
	PLC        PLCConfig        `yaml:"plc"`
	Middleware MiddlewareConfig `yaml:"middleware"`
	Logging    LoggingConfig    `yaml:"logging"`
}

// ServerConfig contains HTTP server configuration
type ServerConfig struct {
	Host string     `yaml:"host"`
	Port int        `yaml:"port"`
	CORS CORSConfig `yaml:"cors"`
}

// CORSConfig contains CORS configuration
type CORSConfig struct {
	Enabled          bool     `yaml:"enabled"`
	AllowedOrigins   []string `yaml:"allowed_origins"`
	AllowedMethods   []string `yaml:"allowed_methods"`
	AllowedHeaders   []string `yaml:"allowed_headers"`
	AllowCredentials bool     `yaml:"allow_credentials"`
}

// PLCConfig contains PLC connection configuration
type PLCConfig struct {
	Target         string `yaml:"target"`
	AMSNetID       string `yaml:"ams_net_id"`
	SourceNetID    string `yaml:"source_net_id"`
	AMSPort        uint16 `yaml:"ams_port"`
	TimeoutSeconds int    `yaml:"timeout_seconds"`
}

// MiddlewareConfig contains middleware-specific configuration
type MiddlewareConfig struct {
	MaxBatchSize        int `yaml:"max_batch_size"`
	MaxSubscriptions    int `yaml:"max_subscriptions"`
	WebSocketBufferSize int `yaml:"websocket_buffer_size"`
}

// LoggingConfig contains logging configuration
type LoggingConfig struct {
	Level  string `yaml:"level"`  // debug, info, warn, error
	Format string `yaml:"format"` // json, text
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Host: "0.0.0.0",
			Port: 8080,
			CORS: CORSConfig{
				Enabled:          true,
				AllowedOrigins:   []string{"*"},
				AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
				AllowedHeaders:   []string{"Content-Type", "Authorization"},
				AllowCredentials: false,
			},
		},
		PLC: PLCConfig{
			Target:         "localhost:48898",
			AMSNetID:       "10.0.10.20.1.1",
			SourceNetID:    "10.10.0.10.1.1",
			AMSPort:        851,
			TimeoutSeconds: 5,
		},
		Middleware: MiddlewareConfig{
			MaxBatchSize:        100,
			MaxSubscriptions:    1000,
			WebSocketBufferSize: 256,
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "text",
		},
	}
}

// LoadConfig loads configuration from a YAML file
func LoadConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	config := DefaultConfig()
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return config, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}

	if c.PLC.Target == "" {
		return fmt.Errorf("PLC target address is required")
	}

	if c.PLC.TimeoutSeconds < 1 {
		return fmt.Errorf("PLC timeout must be at least 1 second")
	}

	if c.Middleware.MaxBatchSize < 1 {
		return fmt.Errorf("max batch size must be at least 1")
	}

	if c.Middleware.MaxSubscriptions < 1 {
		return fmt.Errorf("max subscriptions must be at least 1")
	}

	validLogLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLogLevels[c.Logging.Level] {
		return fmt.Errorf("invalid log level: %s (must be debug, info, warn, or error)", c.Logging.Level)
	}

	validLogFormats := map[string]bool{"json": true, "text": true}
	if !validLogFormats[c.Logging.Format] {
		return fmt.Errorf("invalid log format: %s (must be json or text)", c.Logging.Format)
	}

	return nil
}

// Address returns the server address (host:port)
func (c *Config) Address() string {
	return fmt.Sprintf("%s:%d", c.Server.Host, c.Server.Port)
}

// Timeout returns the PLC timeout as a time.Duration
func (c *Config) Timeout() time.Duration {
	return time.Duration(c.PLC.TimeoutSeconds) * time.Second
}

// SaveExample saves an example configuration file
func SaveExample(filename string) error {
	config := DefaultConfig()
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
