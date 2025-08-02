package handlers

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Krive/ServiceNow-Toolkit/internal/app/explorer"
	"github.com/Krive/ServiceNow-Toolkit/pkg/servicenow"
)

// ExplorerConfig holds configuration for the explorer
type ExplorerConfig struct {
	InstanceURL string
	Username    string
	Password    string
	APIKey      string
	AuthMethod  string
	ClientID    string
	ClientSecret string
	RefreshToken string
	DemoMode    bool
}

// RunExplorer launches the interactive ServiceNow explorer
func RunExplorer(ctx context.Context, config ExplorerConfig) error {
	var client *servicenow.Client
	var err error

	if config.DemoMode {
		client = nil // Demo mode
	} else {
		// Create ServiceNow client based on configuration
		client, err = createServiceNowClient(config)
		if err != nil {
			return fmt.Errorf("failed to create ServiceNow client: %w", err)
		}
	}

	// Create explorer model
	model := explorer.New(client)
	
	// Launch TUI program
	program := tea.NewProgram(model, tea.WithAltScreen())
	_, err = program.Run()
	return err
}

// createServiceNowClient creates a ServiceNow client from configuration
func createServiceNowClient(config ExplorerConfig) (*servicenow.Client, error) {
	if config.InstanceURL == "" {
		return nil, fmt.Errorf("instance URL is required")
	}

	// Create client based on auth method
	switch config.AuthMethod {
	case "basic":
		if config.Username == "" || config.Password == "" {
			return nil, fmt.Errorf("username and password are required for basic auth")
		}
		return servicenow.NewClientBasicAuth(config.InstanceURL, config.Username, config.Password)
	
	case "apikey":
		if config.APIKey == "" {
			return nil, fmt.Errorf("API key is required for API key auth")
		}
		return servicenow.NewClientAPIKey(config.InstanceURL, config.APIKey)
	
	case "oauth-client-credentials":
		if config.ClientID == "" || config.ClientSecret == "" {
			return nil, fmt.Errorf("client ID and client secret are required for OAuth client credentials")
		}
		return servicenow.NewClientOAuth(config.InstanceURL, config.ClientID, config.ClientSecret)
	
	default:
		// Auto-detect auth method
		if config.APIKey != "" {
			return servicenow.NewClientAPIKey(config.InstanceURL, config.APIKey)
		}
		if config.Username != "" && config.Password != "" {
			return servicenow.NewClientBasicAuth(config.InstanceURL, config.Username, config.Password)
		}
		return nil, fmt.Errorf("no valid authentication method found")
	}
}