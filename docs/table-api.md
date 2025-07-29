# Table API Reference

The Table API is the core of ServiceNow Toolkit, providing comprehensive access to ServiceNow tables with full CRUD operations, advanced querying, and performance optimizations.

## Table of Contents

- [Quick Start](#quick-start)
- [Basic Operations](#basic-operations)
- [Query Building](#query-building)
- [Advanced Querying](#advanced-querying)
- [Performance Optimization](#performance-optimization)
- [Error Handling](#error-handling)
- [Best Practices](#best-practices)

## Quick Start

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/Krive/ServiceNow Toolkit/pkg/servicenow"
    "github.com/Krive/ServiceNow Toolkit/pkg/servicenow/query"
)

func main() {
    // Create client
    client, err := servicenow.NewClientBasicAuth(
        "https://yourinstance.service-now.com",
        "username",
        "password",
    )
    if err != nil {
        log.Fatal(err)
    }

    // Get table client
    incidentTable := client.Table("incident")

    // Simple query
    incidents, err := incidentTable.
        Where("active", query.OpEquals, true).
        Limit(10).
        Execute()
    
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Found %d incidents\n", len(incidents))
}
```

## Basic Operations

### Creating a Table Client

```go
// Get a table client for any ServiceNow table
incidentTable := client.Table("incident")
userTable := client.Table("sys_user")
customTable := client.Table("u_custom_table")
```

### CRUD Operations

#### Create Records

```go
// Create a new incident
newIncident := map[string]interface{}{
    "short_description": "Network connectivity issue",
    "description":       "Users cannot access internal websites",
    "category":          "network",
    "subcategory":       "connectivity",
    "urgency":           "2",
    "impact":            "2",
    "caller_id":         "user_sys_id_here",
    "assignment_group":  "network_team_sys_id",
}

createdIncident, err := incidentTable.Create(newIncident)
if err != nil {
    log.Fatal("Failed to create incident:", err)
}

fmt.Printf("Created incident: %s (ID: %s)\n", 
    createdIncident["number"], 
    createdIncident["sys_id"])
```

#### Read Records

```go
// Get a specific record by sys_id
incident, err := incidentTable.Get("incident_sys_id_here")
if err != nil {
    log.Fatal("Failed to get incident:", err)
}

fmt.Printf("Incident: %s - %s\n", 
    incident["number"], 
    incident["short_description"])

// Get with context and timeout
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

incident, err = incidentTable.GetWithContext(ctx, "incident_sys_id_here")
```

#### Update Records

```go
// Update an existing incident
updates := map[string]interface{}{
    "state":         "2", // In Progress
    "assigned_to":   "admin_sys_id_here",
    "work_notes":    "Started investigation",
    "priority":      "1", // High priority
}

updatedIncident, err := incidentTable.Update("incident_sys_id_here", updates)
if err != nil {
    log.Fatal("Failed to update incident:", err)
}

fmt.Printf("Updated incident state to: %s\n", updatedIncident["state"])
```

#### Delete Records

```go
// Delete an incident (careful - this is permanent!)
err := incidentTable.Delete("incident_sys_id_here")
if err != nil {
    log.Fatal("Failed to delete incident:", err)
}

fmt.Println("Incident deleted successfully")

// Soft delete (deactivate) - recommended approach
updates := map[string]interface{}{
    "active": "false",
}
deactivated, err := incidentTable.Update("incident_sys_id_here", updates)
```

## Query Building

ServiceNow Toolkit provides a powerful query builder for complex table operations:

### Basic Queries

```go
import "github.com/Krive/ServiceNow Toolkit/pkg/servicenow/query"

// Simple equality
incidents, err := incidentTable.
    Where("active", query.OpEquals, true).
    Execute()

// Multiple conditions
incidents, err = incidentTable.
    Where("active", query.OpEquals, true).
    And().
    Where("priority", query.OpLessThan, 3).
    Execute()

// OR conditions
incidents, err = incidentTable.
    Where("priority", query.OpEquals, 1).
    Or().
    Where("urgency", query.OpEquals, 1).
    Execute()
```

### Comparison Operators

```go
// Equality
.Where("state", query.OpEquals, "1")
.Where("active", query.OpNotEquals, false)

// Numeric comparisons
.Where("priority", query.OpGreaterThan, 2)
.Where("urgency", query.OpLessThan, 3)
.Where("priority", query.OpGreaterThanOrEqual, 2)
.Where("urgency", query.OpLessThanOrEqual, 3)

// Text operations
.Where("short_description", query.OpContains, "network")
.Where("number", query.OpStartsWith, "INC")
.Where("caller_id.email", query.OpEndsWith, "@company.com")
.Where("description", query.OpNotContains, "test")

// Null checks
.Where("resolved_at", query.OpIsEmpty, nil)
.Where("assigned_to", query.OpIsNotEmpty, nil)

// List operations
.Where("state", query.OpIn, []interface{}{"1", "2", "3"})
.Where("category", query.OpNotIn, []interface{}{"test", "demo"})
```

### Date and Time Queries

```go
// Specific dates
.Where("sys_created_on", query.OpGreaterThan, "2023-01-01 00:00:00")
.Where("sys_updated_on", query.OpBetween, []interface{}{
    "2023-01-01 00:00:00", 
    "2023-12-31 23:59:59",
})

// Relative dates (ServiceNow syntax)
.Where("sys_created_on", query.OpRelativeDate, "today")
.Where("sys_updated_on", query.OpRelativeDate, "last 7 days")
.Where("due_date", query.OpRelativeDate, "next week")

// Date range queries
startDate := time.Now().AddDate(0, 0, -7) // 7 days ago
endDate := time.Now()

incidents, err := incidentTable.
    Where("sys_created_on", query.OpBetween, []interface{}{
        startDate.Format("2006-01-02 15:04:05"),
        endDate.Format("2006-01-02 15:04:05"),
    }).
    Execute()
```

### Complex Logical Operations

```go
// Complex grouping with parentheses
// (priority=1 OR urgency=1) AND active=true AND state!=6
incidents, err := incidentTable.
    OpenGroup().
        Where("priority", query.OpEquals, 1).
        Or().
        Where("urgency", query.OpEquals, 1).
    CloseGroup().
    And().
    Where("active", query.OpEquals, true).
    And().
    Where("state", query.OpNotEquals, 6).
    Execute()

// Nested conditions
// state IN (1,2,3) AND (category=hardware OR category=software)
incidents, err = incidentTable.
    Where("state", query.OpIn, []interface{}{1, 2, 3}).
    And().
    OpenGroup().
        Where("category", query.OpEquals, "hardware").
        Or().
        Where("category", query.OpEquals, "software").
    CloseGroup().
    Execute()
```

### Reference Field Queries

```go
// Query using reference fields (dot notation)
incidents, err := incidentTable.
    Where("caller_id.department", query.OpEquals, "IT").
    Where("assigned_to.user_name", query.OpStartsWith, "admin").
    Where("assignment_group.name", query.OpContains, "Network").
    Execute()

// Multiple levels of references
incidents, err = incidentTable.
    Where("caller_id.manager.department", query.OpEquals, "Engineering").
    Execute()
```

## Advanced Querying

### Field Selection

```go
// Select specific fields to reduce payload size
incidents, err := incidentTable.
    Fields("sys_id", "number", "short_description", "state", "priority").
    Execute()

// Select reference field details
incidents, err = incidentTable.
    Fields(
        "sys_id", 
        "number", 
        "short_description",
        "caller_id.name",
        "caller_id.email",
        "assigned_to.user_name",
    ).
    Execute()
```

### Sorting and Ordering

```go
// Single field ordering
incidents, err := incidentTable.
    OrderBy("sys_created_on").
    Execute()

// Descending order
incidents, err = incidentTable.
    OrderByDesc("priority").
    Execute()

// Multiple field ordering
incidents, err = incidentTable.
    OrderBy("priority").
    OrderByDesc("sys_created_on").
    Execute()

// Custom ordering
incidents, err = incidentTable.
    OrderByCustom("priority DESC, sys_created_on ASC").
    Execute()
```

### Pagination

```go
// Basic pagination
incidents, err := incidentTable.
    Limit(50).
    Offset(100).
    Execute()

// Pagination with ordering for consistent results
incidents, err = incidentTable.
    OrderBy("sys_created_on").
    Limit(25).
    Offset(0).
    Execute()

// Get next page
nextPage, err := incidentTable.
    OrderBy("sys_created_on").
    Limit(25).
    Offset(25).
    Execute()
```

### Advanced Query Patterns

#### Search Multiple Tables

```go
// Search across multiple fields
searchTerm := "network issue"
incidents, err := incidentTable.
    Where("short_description", query.OpContains, searchTerm).
    Or().
    Where("description", query.OpContains, searchTerm).
    Or().
    Where("work_notes", query.OpContains, searchTerm).
    Execute()
```

#### Dynamic Query Building

```go
func buildIncidentQuery(filters map[string]interface{}) *table.TableClient {
    query := incidentTable

    if priority, ok := filters["priority"]; ok {
        query = query.Where("priority", query.OpEquals, priority)
    }

    if state, ok := filters["state"]; ok {
        query = query.Where("state", query.OpEquals, state)
    }

    if assignedTo, ok := filters["assigned_to"]; ok {
        query = query.Where("assigned_to", query.OpEquals, assignedTo)
    }

    if active, ok := filters["active"]; ok {
        if len(query.conditions) > 0 {
            query = query.And()
        }
        query = query.Where("active", query.OpEquals, active)
    }

    return query
}

// Usage
filters := map[string]interface{}{
    "priority": 1,
    "active":   true,
    "state":    []interface{}{1, 2, 3},
}

incidents, err := buildIncidentQuery(filters).Execute()
```

#### Subqueries and References

```go
// Find incidents for users in specific groups
incidents, err := incidentTable.
    Where("caller_id.sys_user_grmember.group.name", query.OpEquals, "IT Support").
    Execute()

// Complex reference queries
incidents, err = incidentTable.
    Where("assignment_group.manager.department", query.OpEquals, "IT").
    And().
    Where("caller_id.location.name", query.OpStartsWith, "Building").
    Execute()
```

## Performance Optimization

### Field Selection Optimization

```go
// Instead of fetching all fields
incidents, err := incidentTable.Execute() // Gets all fields

// Only fetch what you need
incidents, err = incidentTable.
    Fields("sys_id", "number", "short_description", "state").
    Execute() // Much faster and less bandwidth
```

### Query Optimization

```go
// Use indexed fields for better performance
// Good: sys_id, number, active, state
incidents, err := incidentTable.
    Where("active", query.OpEquals, true).
    Where("state", query.OpIn, []interface{}{1, 2, 3}).
    Execute()

// Avoid: LIKE operations on large text fields
// Less optimal:
incidents, err = incidentTable.
    Where("description", query.OpContains, "some text").
    Execute()
```

### Pagination Best Practices

```go
// Process large datasets efficiently
const pageSize = 100
offset := 0

for {
    incidents, err := incidentTable.
        Where("active", query.OpEquals, true).
        OrderBy("sys_created_on").
        Limit(pageSize).
        Offset(offset).
        Execute()
    
    if err != nil {
        log.Fatal(err)
    }
    
    if len(incidents) == 0 {
        break // No more records
    }
    
    // Process incidents
    for _, incident := range incidents {
        fmt.Printf("Processing: %s\n", incident["number"])
    }
    
    offset += pageSize
}
```

### Batch Processing

```go
// Use batch operations for multiple records
batchClient := client.Batch()

// Instead of individual creates
for _, incidentData := range incidentDataList {
    _, err := incidentTable.Create(incidentData) // Slow
}

// Use batch create
result, err := batchClient.CreateMultiple("incident", incidentDataList)
fmt.Printf("Created %d incidents in batch\n", result.SuccessfulRequests)
```

## Error Handling

### Standard Error Handling

```go
import "github.com/Krive/ServiceNow Toolkit/pkg/servicenow/core"

incidents, err := incidentTable.Execute()
if err != nil {
    // Check if it's a ServiceNow-specific error
    if snErr, ok := core.IsServiceNowError(err); ok {
        switch snErr.Type {
        case core.ErrorTypeAuthentication:
            log.Fatal("Authentication failed - check credentials")
        case core.ErrorTypeAuthorization:
            log.Fatal("Access denied - insufficient permissions")
        case core.ErrorTypeRateLimit:
            log.Println("Rate limit hit - retrying...")
            time.Sleep(time.Second * 5)
            // Retry the operation
        case core.ErrorTypeValidation:
            log.Printf("Validation error: %s", snErr.Message)
        case core.ErrorTypeNotFound:
            log.Printf("Table or record not found: %s", snErr.Message)
        default:
            log.Printf("ServiceNow error: %s", snErr.Message)
        }
    } else {
        log.Printf("General error: %s", err)
    }
}
```

### Context and Timeout Handling

```go
import "context"

// Set operation timeout
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

incidents, err := incidentTable.
    Where("active", query.OpEquals, true).
    ExecuteWithContext(ctx)

if err != nil {
    if err == context.DeadlineExceeded {
        log.Println("Operation timed out")
    } else {
        log.Printf("Operation failed: %v", err)
    }
}
```

### Retry Logic

```go
import "time"

func executeWithRetry(operation func() ([]map[string]interface{}, error), maxRetries int) ([]map[string]interface{}, error) {
    var err error
    var result []map[string]interface{}
    
    for attempt := 0; attempt <= maxRetries; attempt++ {
        result, err = operation()
        
        if err == nil {
            return result, nil
        }
        
        // Check if error is retryable
        if snErr, ok := core.IsServiceNowError(err); ok {
            if snErr.IsRetryable() && attempt < maxRetries {
                // Exponential backoff
                delay := time.Duration(attempt+1) * time.Second
                log.Printf("Attempt %d failed, retrying in %v: %s", attempt+1, delay, err)
                time.Sleep(delay)
                continue
            }
        }
        
        break // Non-retryable error or max retries reached
    }
    
    return result, err
}

// Usage
incidents, err := executeWithRetry(func() ([]map[string]interface{}, error) {
    return incidentTable.Where("active", query.OpEquals, true).Execute()
}, 3)
```

## Best Practices

### 1. Always Use Field Selection

```go
// Good: Only fetch needed fields
incidents, err := incidentTable.
    Fields("sys_id", "number", "short_description", "state", "priority").
    Execute()

// Avoid: Fetching all fields unless necessary
incidents, err = incidentTable.Execute() // Gets all fields
```

### 2. Use Appropriate Pagination

```go
// Good: Reasonable page size
incidents, err := incidentTable.Limit(100).Execute()

// Avoid: Very large page sizes
incidents, err = incidentTable.Limit(10000).Execute() // May timeout
```

### 3. Leverage Indexed Fields

```go
// Good: Use indexed fields for filtering
incidents, err := incidentTable.
    Where("active", query.OpEquals, true).
    Where("state", query.OpIn, []interface{}{1, 2, 3}).
    Execute()

// Consider performance impact of text searches
incidents, err = incidentTable.
    Where("short_description", query.OpContains, "network").
    Limit(50). // Limit results for text searches
    Execute()
```

### 4. Implement Proper Error Handling

```go
func getActiveIncidents() ([]map[string]interface{}, error) {
    incidents, err := incidentTable.
        Where("active", query.OpEquals, true).
        Execute()
    
    if err != nil {
        if snErr, ok := core.IsServiceNowError(err); ok {
            // Log structured error information
            log.Printf("ServiceNow error: type=%s, code=%s, message=%s", 
                snErr.Type, snErr.Code, snErr.Message)
        }
        return nil, fmt.Errorf("failed to fetch incidents: %w", err)
    }
    
    return incidents, nil
}
```

### 5. Use Context for Long-Running Operations

```go
func processLargeDataset(ctx context.Context) error {
    const pageSize = 100
    offset := 0
    
    for {
        // Check if context is cancelled
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
        }
        
        incidents, err := incidentTable.
            Where("active", query.OpEquals, true).
            Limit(pageSize).
            Offset(offset).
            ExecuteWithContext(ctx)
        
        if err != nil {
            return err
        }
        
        if len(incidents) == 0 {
            break
        }
        
        // Process incidents...
        offset += pageSize
    }
    
    return nil
}
```

### 6. Structure Complex Queries

```go
// Good: Well-structured query building
func buildReportQuery(filters ReportFilters) *table.TableClient {
    query := incidentTable.
        Fields("sys_id", "number", "short_description", "state", "priority").
        Where("active", query.OpEquals, true)
    
    if filters.Priority > 0 {
        query = query.And().Where("priority", query.OpLessThanOrEqual, filters.Priority)
    }
    
    if filters.AssignmentGroup != "" {
        query = query.And().Where("assignment_group", query.OpEquals, filters.AssignmentGroup)
    }
    
    if !filters.DateRange.Start.IsZero() {
        query = query.And().Where("sys_created_on", query.OpGreaterThanOrEqual, 
            filters.DateRange.Start.Format("2006-01-02 15:04:05"))
    }
    
    return query.OrderByDesc("sys_created_on")
}
```

For more advanced Table API usage, see the [API Reference](api-reference.md) and explore the examples in the repository.