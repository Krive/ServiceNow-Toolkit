package main

import (
	"fmt"
	"os"

	"github.com/Krive/ServiceNow-Toolkit/pkg/servicenow"
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "servicenowtoolkit",
	Short: "ServiceNow Toolkit: A comprehensive CLI for ServiceNow",
	Long: `ServiceNow Toolkit is a powerful command-line interface for ServiceNow that provides
comprehensive access to tables, identity management, aggregations, batch operations,
service catalog, CMDB, and more.

Visit https://github.com/Krive/ServiceNow-Toolkit for documentation and examples.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Load .env file if it exists
		_ = godotenv.Load()
		return nil
	},
}

var (
	// Global flags
	instanceURL string
	username    string
	password    string
	apiKey      string
	verbose     bool

	// OAuth flags
	clientID     string
	clientSecret string
	refreshToken string
	authMethod   string

	// Explorer flags
	demoMode bool
)

func init() {
	// Global persistent flags
	rootCmd.PersistentFlags().StringVar(&instanceURL, "instance", "", "ServiceNow instance URL (or set SERVICENOW_INSTANCE_URL)")
	rootCmd.PersistentFlags().StringVar(&username, "username", "", "Username for basic auth (or set SERVICENOW_USERNAME)")
	rootCmd.PersistentFlags().StringVar(&password, "password", "", "Password for basic auth (or set SERVICENOW_PASSWORD)")
	rootCmd.PersistentFlags().StringVar(&apiKey, "api-key", "", "API key for authentication (or set SERVICENOW_API_KEY)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")

	// OAuth flags
	rootCmd.PersistentFlags().StringVar(&clientID, "client-id", "", "OAuth client ID (or set SERVICENOW_CLIENT_ID)")
	rootCmd.PersistentFlags().StringVar(&clientSecret, "client-secret", "", "OAuth client secret (or set SERVICENOW_CLIENT_SECRET)")
	rootCmd.PersistentFlags().StringVar(&refreshToken, "refresh-token", "", "OAuth refresh token (or set SERVICENOW_REFRESH_TOKEN)")
	rootCmd.PersistentFlags().StringVar(&authMethod, "auth-method", "auto", "Authentication method: auto, basic, apikey, oauth-client-credentials, oauth-authorization-code")

	// Add all command groups
	// Note: Individual commands are added in their respective files via init() functions
}

// createClient creates a ServiceNow client based on provided credentials
func createClient() (*servicenow.Client, error) {
	// Get credentials from flags or environment variables
	url := getCredential(instanceURL, "SERVICENOW_INSTANCE_URL")
	user := getCredential(username, "SERVICENOW_USERNAME")
	pass := getCredential(password, "SERVICENOW_PASSWORD")
	key := getCredential(apiKey, "SERVICENOW_API_KEY")

	// OAuth credentials
	oauthClientID := getCredential(clientID, "SERVICENOW_CLIENT_ID")
	oauthClientSecret := getCredential(clientSecret, "SERVICENOW_CLIENT_SECRET")
	oauthRefreshToken := getCredential(refreshToken, "SERVICENOW_REFRESH_TOKEN")

	if url == "" {
		return nil, fmt.Errorf("ServiceNow instance URL is required (use --instance or set SERVICENOW_INSTANCE_URL)")
	}

	// Determine authentication method
	switch authMethod {
	case "basic":
		return createBasicAuthClient(url, user, pass)
	case "apikey":
		return createAPIKeyClient(url, key)
	case "oauth-client-credentials":
		return createOAuthClientCredentialsClient(url, oauthClientID, oauthClientSecret)
	case "oauth-authorization-code":
		return createOAuthAuthorizationCodeClient(url, oauthClientID, oauthClientSecret, oauthRefreshToken)
	case "auto":
		fallthrough
	default:
		// Auto-detect authentication method based on available credentials
		return autoDetectAuthMethod(url, user, pass, key, oauthClientID, oauthClientSecret, oauthRefreshToken)
	}
}

// createBasicAuthClient creates a client with basic authentication
func createBasicAuthClient(url, user, pass string) (*servicenow.Client, error) {
	if user == "" || pass == "" {
		return nil, fmt.Errorf("basic authentication requires --username and --password (or set SERVICENOW_USERNAME and SERVICENOW_PASSWORD)")
	}
	if verbose {
		fmt.Fprintf(os.Stderr, "Using basic authentication for %s (user: %s)\n", url, user)
	}
	return servicenow.NewClientBasicAuth(url, user, pass)
}

// createAPIKeyClient creates a client with API key authentication
func createAPIKeyClient(url, key string) (*servicenow.Client, error) {
	if key == "" {
		return nil, fmt.Errorf("API key authentication requires --api-key (or set SERVICENOW_API_KEY)")
	}
	if verbose {
		fmt.Fprintf(os.Stderr, "Using API key authentication for %s\n", url)
	}
	return servicenow.NewClientAPIKey(url, key)
}

// createOAuthClientCredentialsClient creates a client with OAuth client credentials flow
func createOAuthClientCredentialsClient(url, clientID, clientSecret string) (*servicenow.Client, error) {
	if clientID == "" || clientSecret == "" {
		return nil, fmt.Errorf("OAuth client credentials requires --client-id and --client-secret (or set SERVICENOW_CLIENT_ID and SERVICENOW_CLIENT_SECRET)")
	}
	if verbose {
		fmt.Fprintf(os.Stderr, "Using OAuth client credentials for %s (client: %s)\n", url, clientID)
	}
	return servicenow.NewClientOAuth(url, clientID, clientSecret)
}

// createOAuthAuthorizationCodeClient creates a client with OAuth authorization code flow
func createOAuthAuthorizationCodeClient(url, clientID, clientSecret, refreshToken string) (*servicenow.Client, error) {
	if clientID == "" || clientSecret == "" {
		return nil, fmt.Errorf("OAuth authorization code requires --client-id and --client-secret (or set SERVICENOW_CLIENT_ID and SERVICENOW_CLIENT_SECRET)")
	}
	if refreshToken == "" {
		return nil, fmt.Errorf("OAuth authorization code requires --refresh-token (or set SERVICENOW_REFRESH_TOKEN)")
	}
	if verbose {
		fmt.Fprintf(os.Stderr, "Using OAuth authorization code for %s (client: %s)\n", url, clientID)
	}
	return servicenow.NewClientOAuthRefresh(url, clientID, clientSecret, refreshToken)
}

// autoDetectAuthMethod automatically detects the best authentication method based on available credentials
func autoDetectAuthMethod(url, user, pass, key, clientID, clientSecret, refreshToken string) (*servicenow.Client, error) {
	// Priority order: API Key > Basic Auth > OAuth client credentials > OAuth authorization code

	// Check for API Key (highest priority - most common for automation)
	if key != "" {
		if verbose {
			fmt.Fprintf(os.Stderr, "Auto-detected API key authentication\n")
		}
		return createAPIKeyClient(url, key)
	}

	// Check for Basic Auth (second priority - simple and reliable)
	if user != "" && pass != "" {
		if verbose {
			fmt.Fprintf(os.Stderr, "Auto-detected basic authentication\n")
		}
		return createBasicAuthClient(url, user, pass)
	}

	// Check for OAuth client credentials (third priority)
	if clientID != "" && clientSecret != "" && refreshToken == "" {
		if verbose {
			fmt.Fprintf(os.Stderr, "Auto-detected OAuth client credentials authentication\n")
		}
		return createOAuthClientCredentialsClient(url, clientID, clientSecret)
	}

	// Check for OAuth authorization code (lowest priority - requires refresh token)
	if clientID != "" && clientSecret != "" && refreshToken != "" {
		if verbose {
			fmt.Fprintf(os.Stderr, "Auto-detected OAuth authorization code authentication\n")
		}
		return createOAuthAuthorizationCodeClient(url, clientID, clientSecret, refreshToken)
	}

	return nil, fmt.Errorf("no valid authentication credentials found. Provide one of:\n" +
		"  - API Key: --api-key (recommended)\n" +
		"  - Basic Auth: --username and --password\n" +
		"  - OAuth Client Credentials: --client-id and --client-secret\n" +
		"  - OAuth Authorization Code: --client-id, --client-secret, and --refresh-token")
}

// getCredential gets a credential from flag or environment variable
func getCredential(flagValue, envVar string) string {
	if flagValue != "" {
		return flagValue
	}
	return os.Getenv(envVar)
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}
