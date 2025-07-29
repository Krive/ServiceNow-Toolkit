# Getting Started with ServiceNow Toolkit

ServiceNow Toolkit is a comprehensive Golang SDK and CLI for ServiceNow that enables both programmatic integration and terminal-based exploration of your ServiceNow instance.

## Table of Contents

- [Installation](#installation)
- [Authentication Setup](#authentication-setup)
- [Your First API Call](#your-first-api-call)
- [CLI Quick Start](#cli-quick-start)
- [Common Use Cases](#common-use-cases)
- [Next Steps](#next-steps)

## Installation

### Prerequisites
- Go 1.21 or higher
- Access to a ServiceNow instance
- Valid ServiceNow credentials (username/password, API key, or OAuth)

### Install ServiceNow Toolkit

#### As a Library
```bash
go get github.com/Krive/ServiceNow Toolkit
```

#### As a CLI Tool
```bash
# Install the CLI globally
go install github.com/Krive/ServiceNow-Toolkit/cmd/servicenowtoolkit@latest

# Or build from source
git clone https://github.com/Krive/ServiceNow-Toolkit.git
cd ServiceNow-Toolkit
go build -o servicenowtoolkit cmd/servicenowtoolkit/main.go
```

## Authentication Setup

ServiceNow Toolkit supports multiple authentication methods. Choose the one that best fits your environment.

### Method 1: Environment Variables (Recommended)

Create a `.env` file in your project root:

```env
# Basic Authentication
SERVICENOW_INSTANCE_URL=https://yourinstance.service-now.com
SERVICENOW_USERNAME=your.username
SERVICENOW_PASSWORD=your_password

# OR API Key Authentication
SERVICENOW_INSTANCE_URL=https://yourinstance.service-now.com
SERVICENOW_API_KEY=your_api_key

# OR OAuth Authentication
SERVICENOW_INSTANCE_URL=https://yourinstance.service-now.com
SERVICENOW_CLIENT_ID=your_client_id
SERVICENOW_CLIENT_SECRET=your_client_secret
SERVICENOW_REFRESH_TOKEN=your_refresh_token  # Optional, for refresh token flow
```

### Method 2: Direct Code Configuration

```go
package main

import (
    "github.com/Krive/ServiceNow Toolkit/pkg/servicenow"
)

func main() {
    // Basic Auth
    client, err := servicenow.NewClientBasicAuth(
        "https://yourinstance.service-now.com",
        "username",
        "password",
    )
    
    // API Key Auth
    client, err := servicenow.NewClientAPIKey(
        "https://yourinstance.service-now.com",
        "your_api_key",
    )
    
    // OAuth Client Credentials
    client, err := servicenow.NewClientOAuth(
        "https://yourinstance.service-now.com",
        "client_id",
        "client_secret",
    )
    
    // OAuth with Refresh Token
    client, err := servicenow.NewClientOAuthRefresh(
        "https://yourinstance.service-now.com",
        "client_id",
        "client_secret",
        "refresh_token",
    )
}
```

### Method 3: CLI Configuration

```bash
# Set up CLI authentication (creates ~/.servicenowtoolkit/config)
servicenowtoolkit auth setup

# Or use environment variables
export SERVICENOW_INSTANCE_URL="https://yourinstance.service-now.com"
export SERVICENOW_USERNAME="your.username"
export SERVICENOW_PASSWORD="your_password"
```

## Your First API Call

Let's start with a simple example that fetches some incident records:

### SDK Example

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/Krive/ServiceNow Toolkit/pkg/servicenow"
    "github.com/Krive/ServiceNow Toolkit/pkg/servicenow/query"
)

func main() {
    // Create client with your preferred authentication method
    client, err := servicenow.NewClientBasicAuth(
        "https://yourinstance.service-now.com",
        "username",
        "password",
    )
    if err != nil {
        log.Fatal("Failed to create client:", err)
    }

    // Get the last 5 active incidents
    incidents, err := client.Table("incident").
        Where("active", query.OpEquals, true).
        OrderByDesc("sys_created_on").
        Limit(5).
        Fields("number", "short_description", "state", "priority").
        Execute()
    
    if err != nil {
        log.Fatal("Failed to fetch incidents:", err)
    }

    // Display results
    fmt.Printf("Found %d active incidents:\n\n", len(incidents))
    for _, incident := range incidents {
        fmt.Printf("Number: %s\n", incident["number"])
        fmt.Printf("Description: %s\n", incident["short_description"])
        fmt.Printf("Priority: %s\n", incident["priority"])
        fmt.Printf("---\n")
    }
}
```

### CLI Example

```bash
# List recent incidents
servicenowtoolkit table incident list --limit 5 --fields "number,short_description,state" --filter "active=true"

# Get a specific incident
servicenowtoolkit table incident get INC0000123

# Create a new incident (interactive)
servicenowtoolkit table incident create --interactive
```

## CLI Quick Start

The ServiceNow Toolkit CLI provides an intuitive way to interact with ServiceNow from the command line.

### Basic Commands

```bash
# List tables
servicenowtoolkit table list

# Query records
servicenowtoolkit table incident list --limit 10
servicenowtoolkit table incident list --filter "priority=1^active=true"
servicenowtoolkit table incident list --fields "number,short_description,assigned_to"

# Get a specific record
servicenowtoolkit table incident get <sys_id>

# Create a record
servicenowtoolkit table incident create --data '{"short_description":"Test incident","category":"software"}'

# Update a record
servicenowtoolkit table incident update <sys_id> --data '{"state":"2","assigned_to":"admin"}'

# Delete a record
servicenowtoolkit table incident delete <sys_id>
```

### Working with Attachments

```bash
# List attachments for a record
servicenowtoolkit attachment list --table incident --record <sys_id>

# Upload a file
servicenowtoolkit attachment upload --table incident --record <sys_id> --file /path/to/file.pdf

# Download an attachment
servicenowtoolkit attachment download <attachment_sys_id> --output /path/to/save/file.pdf

# Delete an attachment
servicenowtoolkit attachment delete <attachment_sys_id>
```

## Common Use Cases

### 1. Incident Management

```go
// Create a new incident
newIncident := map[string]interface{}{
    "short_description": "Network connectivity issue in Building A",
    "description":       "Users in Building A cannot access internal websites",
    "category":          "network",
    "subcategory":       "connectivity",
    "urgency":           "2",
    "impact":            "2",
    "caller_id":         "user_sys_id_here",
}

incident, err := client.Table("incident").Create(newIncident)
if err != nil {
    log.Fatal("Failed to create incident:", err)
}

fmt.Printf("Created incident: %s\n", incident["number"])

// Update incident status
updates := map[string]interface{}{
    "state":       "2", // In Progress
    "assigned_to": "admin_sys_id_here",
    "work_notes":  "Investigation started",
}

updatedIncident, err := client.Table("incident").Update(incident["sys_id"].(string), updates)
```

### 2. User Management

```go
// Get identity client
identity := client.Identity()

// Find users in IT department
users, err := identity.ListUsers(&identity.UserFilter{
    Active:     &[]bool{true}[0],
    Department: "IT",
    Limit:      50,
})

for _, user := range users {
    fmt.Printf("User: %s (%s) - %s\n", user.Name, user.UserName, user.Email)
    
    // Get user's roles
    roleClient := identity.NewRoleClient()
    userRoles, err := roleClient.GetUserRoles(user.SysID)
    if err == nil {
        fmt.Printf("  Roles: %d\n", len(userRoles))
    }
}
```

### 3. Data Analysis with Aggregates

```go
// Analyze incident trends
aggClient := client.Aggregate("incident")

// Get incident counts by priority
result, err := aggClient.NewQuery().
    CountAll("incident_count").
    Avg("urgency", "avg_urgency").
    GroupByField("priority", "priority_level").
    Where("active", query.OpEquals, true).
    OrderByDesc("incident_count").
    Execute()

if err != nil {
    log.Fatal("Aggregation failed:", err)
}

fmt.Println("Incident Analysis by Priority:")
for _, row := range result.Result {
    fmt.Printf("Priority %v: %v incidents (avg urgency: %.1f)\n",
        row["priority_level"], 
        row["incident_count"], 
        row["avg_urgency"])
}
```

### 4. Bulk Operations

```go
// Create multiple incidents efficiently
batchClient := client.Batch()

incidents := []map[string]interface{}{
    {
        "short_description": "Server maintenance required",
        "category":          "hardware",
        "urgency":           3,
    },
    {
        "short_description": "Software license renewal",
        "category":          "software",
        "urgency":           4,
    },
    {
        "short_description": "Network switch replacement",
        "category":          "network",
        "urgency":           2,
    },
}

result, err := batchClient.CreateMultiple("incident", incidents)
if err != nil {
    log.Fatal("Batch creation failed:", err)
}

fmt.Printf("Successfully created %d incidents\n", result.SuccessfulRequests)
fmt.Printf("Failed requests: %d\n", result.FailedRequests)
```

### 5. Configuration Management

```go
// Work with CMDB
cmdb := client.CMDB()

// Find all Windows servers
servers, err := cmdb.ListCIs(&cmdb.CIFilter{
    Class:  "cmdb_ci_win_server",
    Active: &[]bool{true}[0],
})

for _, server := range servers {
    fmt.Printf("Server: %s (%s)\n", server.Name, server.IPAddress)
    
    // Get server relationships
    relationships, err := cmdb.GetCIRelationships(server.SysID)
    if err == nil {
        fmt.Printf("  Related CIs: %d\n", len(relationships))
    }
}
```

## Error Handling Best Practices

Always implement proper error handling:

```go
import (
    "context"
    "time"
    
    "github.com/Krive/ServiceNow Toolkit/pkg/servicenow/core"
)

// Use context with timeout
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

records, err := client.Table("incident").
    Where("active", query.OpEquals, true).
    ExecuteWithContext(ctx)

if err != nil {
    // Check if it's a ServiceNow-specific error
    if snErr, ok := core.IsServiceNowError(err); ok {
        switch snErr.Type {
        case core.ErrorTypeAuthentication:
            log.Fatal("Authentication failed - check your credentials")
        case core.ErrorTypeRateLimit:
            log.Println("Rate limit hit - retrying...")
            // Implement retry logic
        case core.ErrorTypeValidation:
            log.Printf("Validation error: %s", snErr.Message)
        default:
            log.Printf("ServiceNow error: %s", snErr.Message)
        }
    } else {
        log.Printf("General error: %s", err)
    }
    return
}
```

## Performance and Rate Limiting

Configure rate limiting and retry behavior for production use:

```go
// Conservative settings for production
client := client.
    WithConservativeRateLimit().
    WithAggressiveRetry()

// Or custom configuration
client.SetTimeout(45 * time.Second)
```

## Next Steps

Now that you have ServiceNow Toolkit set up and running, explore these advanced topics:

1. **[Authentication Guide](authentication.md)** - Detailed authentication setup
2. **[Table API](table-api.md)** - Complete table operations reference
3. **[Aggregate API](aggregate_api.md)** - Data analysis and reporting
4. **[Batch API](batch_api.md)** - High-performance bulk operations
5. **[Catalog API](catalog_api.md)** - Service catalog automation
6. **[API Reference](api-reference.md)** - Complete API documentation

### Examples Repository

Check out the `examples/` directory for more comprehensive examples:

- `examples/basic_auth/` - Basic authentication setup
- `examples/oauth/` - OAuth configuration
- `examples/query_builder/` - Advanced querying
- `examples/batch_examples.go` - Bulk operations
- `examples/aggregate_examples.go` - Data analysis
- `examples/catalog_examples.go` - Service catalog workflows

### Community and Support

- **GitHub Issues**: Report bugs and request features
- **Discussions**: Ask questions and share solutions
- **Contributing**: See [contributing.md](contributing.md) for development setup

Happy coding with ServiceNow Toolkit! =�