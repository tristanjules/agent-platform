package config

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

// Load reads and parses a TOML configuration file, applying defaults first.
// Missing fields in the TOML file will retain their default values.
func Load(path string) (*Config, error) {
	cfg := Defaults()

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file %s: %w", path, err)
	}

	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config file %s: %w", path, err)
	}

	// Resolve environment variables for sensitive values.
	cfg.resolveEnv()

	return &cfg, nil
}

// resolveEnv populates config fields from environment variables.
func (c *Config) resolveEnv() {
	// Cloud API key from environment.
	if key := os.Getenv("ANTHROPIC_API_KEY"); key != "" {
		// Store on the config for the router to access.
		// We don't put it in the struct directly to avoid accidental logging.
		cloudAPIKeyStore = key
	}

	// Ollama host override.
	if host := os.Getenv("OLLAMA_HOST"); host != "" {
		c.Inference.Local.Endpoint = host
	}
}

// cloudAPIKeyStore holds the Anthropic API key loaded from the environment.
// Kept separate from the config struct to prevent accidental serialization.
var cloudAPIKeyStore string

// CloudAPIKey returns the Anthropic API key loaded from the ANTHROPIC_API_KEY
// environment variable during config loading.
func CloudAPIKey() string {
	return cloudAPIKeyStore
}
