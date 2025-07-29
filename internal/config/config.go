package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

// Config holds centralized ServiceNow connection details
type Config struct {
	InstanceURL  string
	Username     string // For Basic Auth
	Password     string // For Basic Auth
	ClientID     string // For OAuth
	ClientSecret string // For OAuth
}

// LoadConfig loads config from environment variables, optionally from .env file
func LoadConfig() (*Config, error) {
	// Load .env file if it exists (non-fatal if missing)
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to load .env file: %w", err)
	}

	cfg := &Config{
		InstanceURL:  os.Getenv("SERVICENOW_INSTANCE_URL"),
		Username:     os.Getenv("SERVICENOW_USERNAME"),
		Password:     os.Getenv("SERVICENOW_PASSWORD"),
		ClientID:     os.Getenv("SERVICENOW_CLIENT_ID"),
		ClientSecret: os.Getenv("SERVICENOW_CLIENT_SECRET"),
	}

	// Validate required fields
	if cfg.InstanceURL == "" {
		return nil, fmt.Errorf("missing required env var: SERVICENOW_INSTANCE_URL")
	}
	// Optionally validate auth-specific fields
	if cfg.Username == "" && cfg.ClientID == "" {
		return nil, fmt.Errorf("must provide either SERVICENOW_USERNAME or SERVICENOW_CLIENT_ID for authentication")
	}

	return cfg, nil
}
