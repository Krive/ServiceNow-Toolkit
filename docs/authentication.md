# Authentication Guide

ServiceNow Toolkit supports multiple authentication methods to work with your ServiceNow instance. This guide covers all available authentication methods, security best practices, and troubleshooting common issues.

## Table of Contents

- [Authentication Methods](#authentication-methods)
- [SDK Authentication](#sdk-authentication)
- [CLI Authentication](#cli-authentication)
- [Security Best Practices](#security-best-practices)
- [Token Management](#token-management)
- [Troubleshooting](#troubleshooting)

## Authentication Methods

ServiceNow Toolkit supports four authentication methods:

1. **Basic Authentication** - Username and password
2. **API Key Authentication** - ServiceNow API key
3. **OAuth Client Credentials** - OAuth 2.0 client credentials flow
4. **OAuth Authorization Code** - OAuth 2.0 with refresh token

### When to Use Each Method

| Method | Use Case | Security Level | Recommended For |
|--------|----------|----------------|-----------------|
| Basic Auth | Development, testing | PP | Local development, scripts |
| API Key | Automation, CI/CD | PPP | Service accounts, automation |
| OAuth Client | Service-to-service | PPPP | Production applications |
| OAuth + Refresh | User applications | PPPPP | Interactive applications |

## SDK Authentication

### Basic Authentication

```go
package main

import (
    "log"
    "github.com/Krive/ServiceNow Toolkit/pkg/servicenow"
)

func main() {
    client, err := servicenow.NewClientBasicAuth(
        "https://yourinstance.service-now.com",
        "your.username",
        "your_password",
    )
    if err != nil {
        log.Fatal("Authentication failed:", err)
    }
    
    // Use client...
}
```

**Pros:**
- Simple to set up
- Good for development and testing
- No additional configuration required

**Cons:**
- Less secure (credentials in code/environment)
- Password changes require code updates
- Not recommended for production

### API Key Authentication

First, create an API key in ServiceNow:
1. Go to **System Definition > API Keys**
2. Click **New**
3. Fill in the details and save
4. Copy the generated API key

```go
func main() {
    client, err := servicenow.NewClientAPIKey(
        "https://yourinstance.service-now.com",
        "your_api_key_here",
    )
    if err != nil {
        log.Fatal("API key authentication failed:", err)
    }
    
    // Use client...
}
```

**Pros:**
- More secure than basic auth
- Easy to rotate keys
- Good for automation
- No password expiration issues

**Cons:**
- Requires ServiceNow configuration
- Key management needed

### OAuth Client Credentials Flow

Set up OAuth in ServiceNow:
1. Go to **System OAuth > Application Registry**
2. Click **New** > **Create an OAuth API endpoint for external clients**
3. Fill in the details and save
4. Note the Client ID and Client Secret

```go
func main() {
    client, err := servicenow.NewClientOAuth(
        "https://yourinstance.service-now.com",
        "your_client_id",
        "your_client_secret",
    )
    if err != nil {
        log.Fatal("OAuth authentication failed:", err)
    }
    
    // Use client...
}
```

**Pros:**
- Very secure
- Token-based authentication
- Automatic token refresh
- Industry standard

**Cons:**
- More complex setup
- Requires OAuth configuration in ServiceNow

### OAuth Authorization Code Flow (with Refresh Token)

This method is for applications that have already obtained a refresh token through the OAuth authorization code flow:

```go
func main() {
    client, err := servicenow.NewClientOAuthRefresh(
        "https://yourinstance.service-now.com",
        "your_client_id",
        "your_client_secret",
        "your_refresh_token",
    )
    if err != nil {
        log.Fatal("OAuth refresh authentication failed:", err)
    }
    
    // Use client...
}
```

**Pros:**
- Highest security level
- User consent-based
- Long-lived tokens
- Perfect for user applications

**Cons:**
- Most complex setup
- Requires user authorization flow
- Token management complexity

### Advanced Configuration

You can also use the unified configuration approach:

```go
import (
    "time"
    "github.com/Krive/ServiceNow Toolkit/pkg/servicenow"
    "github.com/Krive/ServiceNow Toolkit/internal/utils/retry"
    "github.com/Krive/ServiceNow Toolkit/internal/utils/ratelimit"
)

func main() {
    config := servicenow.Config{
        InstanceURL: "https://yourinstance.service-now.com",
        
        // Choose one authentication method:
        
        // Basic Auth
        Username: "your.username",
        Password: "your_password",
        
        // OR API Key
        // APIKey: "your_api_key",
        
        // OR OAuth
        // ClientID:     "your_client_id",
        // ClientSecret: "your_client_secret",
        // RefreshToken: "your_refresh_token", // Optional
        
        // Performance settings
        Timeout: 30 * time.Second,
        RetryConfig: &retry.Config{
            MaxAttempts: 3,
            BaseDelay:   1 * time.Second,
            MaxDelay:    30 * time.Second,
            Multiplier:  2.0,
            Jitter:      true,
        },
        RateLimitConfig: &ratelimit.ServiceNowLimiterConfig{
            TableRequestsPerSecond:      5.0,
            AttachmentRequestsPerSecond: 2.0,
            ImportRequestsPerSecond:     1.0,
            DefaultRequestsPerSecond:    3.0,
            TableBurst:                  10,
            AttachmentBurst:             5,
            ImportBurst:                 2,
            DefaultBurst:                6,
        },
    }
    
    client, err := servicenow.NewClient(config)
    if err != nil {
        log.Fatal("Failed to create client:", err)
    }
}
```

## CLI Authentication

The ServiceNow Toolkit CLI supports multiple ways to configure authentication:

### Method 1: Environment Variables (Recommended)

```bash
# Basic Authentication
export SERVICENOW_INSTANCE_URL="https://yourinstance.service-now.com"
export SERVICENOW_USERNAME="your.username"
export SERVICENOW_PASSWORD="your_password"

# OR API Key Authentication
export SERVICENOW_INSTANCE_URL="https://yourinstance.service-now.com"
export SERVICENOW_API_KEY="your_api_key"

# OR OAuth Authentication
export SERVICENOW_INSTANCE_URL="https://yourinstance.service-now.com"
export SERVICENOW_CLIENT_ID="your_client_id"
export SERVICENOW_CLIENT_SECRET="your_client_secret"
export SERVICENOW_REFRESH_TOKEN="your_refresh_token"  # Optional

# Test the connection
servicego table incident list --limit 1
```

### Method 2: Configuration File

Create `~/.servicego/config.yaml`:

```yaml
instance_url: "https://yourinstance.service-now.com"

# Basic Auth
username: "your.username"
password: "your_password"

# OR API Key
# api_key: "your_api_key"

# OR OAuth
# client_id: "your_client_id"
# client_secret: "your_client_secret"
# refresh_token: "your_refresh_token"

# Optional settings
timeout: 30s
rate_limit:
  table_requests_per_second: 5.0
  table_burst: 10
```

### Method 3: Interactive Setup

```bash
# Interactive authentication setup
servicego auth setup

# Test the connection
servicego auth test

# Clear stored credentials
servicego auth clear
```

### Method 4: .env File

Create a `.env` file in your current directory:

```env
SERVICENOW_INSTANCE_URL=https://yourinstance.service-now.com
SERVICENOW_USERNAME=your.username
SERVICENOW_PASSWORD=your_password
```

The CLI will automatically load this file.

### Method 5: Command Line Flags with Auth Method Selection

ServiceNow Toolkit supports an `--auth-method` flag to explicitly control authentication:

```bash
# Automatic detection (default)
servicego table incident list \
  --instance "https://yourinstance.service-now.com" \
  --api-key "your_api_key"

# Force specific auth method
servicego table incident list \
  --instance "https://yourinstance.service-now.com" \
  --username "user" --password "pass" \
  --auth-method basic

# OAuth client credentials
servicego table incident list \
  --instance "https://yourinstance.service-now.com" \
  --client-id "client_id" --client-secret "secret" \
  --auth-method oauth-client-credentials

# OAuth with refresh token
servicego table incident list \
  --instance "https://yourinstance.service-now.com" \
  --client-id "client_id" --client-secret "secret" \
  --refresh-token "token" \
  --auth-method oauth-authorization-code
```

**Available auth methods:**
- `auto` (default) - Automatically detect based on provided credentials
- `basic` - Username/password authentication
- `apikey` - API key authentication  
- `oauth-client-credentials` - OAuth client credentials flow
- `oauth-authorization-code` - OAuth with refresh token

**Auto-detection priority:**
1. API Key (if `--api-key`) - **Recommended for most use cases**
2. Basic Authentication (if `--username` and `--password`)
3. OAuth Client Credentials (if `--client-id` and `--client-secret`)
4. OAuth Authorization Code (if OAuth credentials + `--refresh-token`)

## Security Best Practices

### 1. Credential Storage

**Do:**
- Use environment variables for production
- Store credentials in secure credential managers
- Use OAuth for production applications
- Rotate API keys regularly
- Use the principle of least privilege

**Don't:**
- Hard-code credentials in source code
- Commit credentials to version control
- Share credentials between environments
- Use basic auth in production
- Store credentials in plain text files

### 2. Environment Separation

```bash
# Development
export SERVICENOW_INSTANCE_URL="https://dev-instance.service-now.com"
export SERVICENOW_API_KEY="dev_api_key"

# Production
export SERVICENOW_INSTANCE_URL="https://prod-instance.service-now.com"
export SERVICENOW_API_KEY="prod_api_key"
```

### 3. Secure Configuration Files

```bash
# Set proper permissions on config files
chmod 600 ~/.servicego/config.yaml
chmod 600 ~/.servicego/tokens/*

# Use secure directories
mkdir -p ~/.servicego/tokens
chmod 700 ~/.servicego
```

### 4. Token Security

```go
// Implement secure token storage
import (
    "encoding/json"
    "os"
    "path/filepath"
)

type TokenStorage struct {
    AccessToken  string    `json:"access_token"`
    RefreshToken string    `json:"refresh_token"`
    ExpiresAt    time.Time `json:"expires_at"`
}

func saveTokenSecurely(token *TokenStorage) error {
    homeDir, _ := os.UserHomeDir()
    tokenDir := filepath.Join(homeDir, ".servicego", "tokens")
    
    // Create directory with proper permissions
    if err := os.MkdirAll(tokenDir, 0700); err != nil {
        return err
    }
    
    tokenFile := filepath.Join(tokenDir, "current.json")
    
    // Write token with restricted permissions
    data, _ := json.Marshal(token)
    return os.WriteFile(tokenFile, data, 0600)
}
```

## Token Management

### Automatic Token Refresh

ServiceNow Toolkit automatically handles token refresh for OAuth flows:

```go
// OAuth client automatically refreshes tokens
client, err := servicenow.NewClientOAuth(instanceURL, clientID, clientSecret)

// Tokens are refreshed automatically on API calls
records, err := client.Table("incident").Execute() // Will refresh token if needed
```

### Manual Token Management

```go
import "github.com/Krive/ServiceNow Toolkit/pkg/servicenow/core"

// Get the underlying core client
coreClient := client.Core()

// Check if token is expired (for OAuth clients)
if oauthAuth, ok := coreClient.Auth.(*core.OAuthProvider); ok {
    if oauthAuth.IsTokenExpired() {
        err := oauthAuth.RefreshToken()
        if err != nil {
            log.Printf("Failed to refresh token: %v", err)
        }
    }
}
```

### Token Storage Locations

ServiceNow Toolkit stores tokens in these locations:

```
~/.servicego/
   config.yaml          # Main configuration
   tokens/
      current.json     # Current OAuth tokens
      backup.json      # Token backup
      refresh.json     # Refresh token storage
   logs/
       auth.log         # Authentication logs
```

## Troubleshooting

### Common Authentication Errors

#### 1. "Invalid user credentials"

```bash
# Basic Auth Error
Error: Authentication failed: Invalid user credentials

# Solutions:
- Verify username and password are correct
- Check if account is locked or disabled
- Ensure user has necessary roles (e.g., 'rest_service')
- Check instance URL is correct
```

#### 2. "Invalid API Key"

```bash
Error: Authentication failed: Invalid API key

# Solutions:
- Verify API key is correct and not expired
- Check API key has proper scope/roles
- Ensure API key is for the correct instance
- Regenerate API key if necessary
```

#### 3. "OAuth token expired"

```bash
Error: OAuth authentication failed: Token has expired

# Solutions:
- Token will auto-refresh on next request
- Check refresh token is still valid
- Verify OAuth application configuration
- Re-authorize if refresh token is invalid
```

#### 4. "Permission denied"

```bash
Error: Access denied for table 'incident'

# Solutions:
- Check user has required roles (e.g., 'incident_read')
- Verify table ACLs allow access
- Ensure API user has proper permissions
- Check if table exists and is accessible
```

### Debug Authentication Issues

Enable debug logging:

```go
import "log"

// Enable debug logging
log.SetFlags(log.LstdFlags | log.Lshortfile)

client, err := servicenow.NewClientBasicAuth(instanceURL, username, password)
if err != nil {
    log.Printf("Auth error details: %+v", err)
}
```

For CLI debugging:

```bash
# Enable verbose logging
servicego --verbose table incident list

# Check authentication specifically
servicego auth test --verbose

# View stored configuration
servicego auth status
```

### Testing Authentication

```go
// Test authentication by making a simple API call
func testAuthentication(client *servicenow.Client) error {
    // Try to get user info (requires minimal permissions)
    _, err := client.Table("sys_user").
        Where("sys_id", query.OpEquals, "current_user_id").
        Limit(1).
        Execute()
    
    return err
}
```

### Authentication Health Check

```bash
# CLI health check
servicego auth test

# Detailed status
servicego auth status --detailed

# Test specific operations
servicego table sys_user list --limit 1 --fields "user_name,sys_id"
```

## Advanced Authentication Scenarios

### Multi-Instance Support

```go
// Manage multiple ServiceNow instances
clients := map[string]*servicenow.Client{
    "dev": func() *servicenow.Client {
        client, _ := servicenow.NewClientAPIKey("https://dev.service-now.com", devAPIKey)
        return client
    }(),
    "prod": func() *servicenow.Client {
        client, _ := servicenow.NewClientAPIKey("https://prod.service-now.com", prodAPIKey)
        return client
    }(),
}

// Use different clients for different environments
devIncidents, _ := clients["dev"].Table("incident").Execute()
prodIncidents, _ := clients["prod"].Table("incident").Execute()
```

### Custom Authentication Provider

```go
// Implement custom authentication
type CustomAuthProvider struct {
    token string
}

func (c *CustomAuthProvider) Apply(client *resty.Client) error {
    client.SetHeader("Authorization", "Bearer "+c.token)
    return nil
}

func (c *CustomAuthProvider) Refresh() error {
    // Custom token refresh logic
    return nil
}

// Use custom auth with core client
coreClient, err := core.NewClient("https://instance.service-now.com", &CustomAuthProvider{
    token: "your_custom_token",
})
```

For more advanced authentication scenarios and enterprise integrations, please refer to the [API Reference](api-reference.md) and check the examples in the repository.