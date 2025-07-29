# ServiceNow Toolkit API Reference

Complete reference for all ServiceNow Toolkit SDK APIs and their methods.

## Table of Contents

- [Client Creation](#client-creation)
- [Table API](#table-api)
- [Identity API](#identity-api)
- [Aggregate API](#aggregate-api)
- [Batch API](#batch-api)
- [Catalog API](#catalog-api)
- [CMDB API](#cmdb-api)
- [Attachment API](#attachment-api)
- [ImportSet API](#importset-api)
- [Query Builder](#query-builder)
- [Error Handling](#error-handling)

## Client Creation

### Basic Authentication
```go
client, err := servicenow.NewClientBasicAuth(instanceURL, username, password)
```

### API Key Authentication
```go
client, err := servicenow.NewClientAPIKey(instanceURL, apiKey)
```

### OAuth Client Credentials
```go
client, err := servicenow.NewClientOAuth(instanceURL, clientID, clientSecret)
```

### OAuth with Refresh Token
```go
client, err := servicenow.NewClientOAuthRefresh(instanceURL, clientID, clientSecret, refreshToken)
```

### Configuration Options
```go
config := servicenow.Config{
    InstanceURL: "https://instance.service-now.com",
    Username:    "user",
    Password:    "pass",
    Timeout:     30 * time.Second,
    RetryConfig: &retry.Config{
        MaxAttempts: 3,
        BaseDelay:   1 * time.Second,
    },
    RateLimitConfig: &ratelimit.ServiceNowLimiterConfig{
        TableRequestsPerSecond: 5.0,
        TableBurst:            10,
    },
}
client, err := servicenow.NewClient(config)
```

## Table API

### Methods
```go
tableClient := client.Table("tableName")

// CRUD Operations
record, err := tableClient.Get(sysID)
records, err := tableClient.GetWithContext(ctx, sysID)

newRecord, err := tableClient.Create(data)
newRecord, err := tableClient.CreateWithContext(ctx, data)

updated, err := tableClient.Update(sysID, updates)
updated, err := tableClient.UpdateWithContext(ctx, sysID, updates)

err := tableClient.Delete(sysID)
err := tableClient.DeleteWithContext(ctx, sysID)

// Query Operations
records, err := tableClient.Where("field", query.OpEquals, "value").Execute()
records, err := tableClient.Where("field", query.OpEquals, "value").ExecuteWithContext(ctx)

// Pagination
records, err := tableClient.Limit(100).Offset(50).Execute()

// Field Selection
records, err := tableClient.Fields("sys_id", "number", "short_description").Execute()

// Ordering
records, err := tableClient.OrderBy("sys_created_on").Execute()
records, err := tableClient.OrderByDesc("sys_updated_on").Execute()
```

## Identity API

### Client Creation
```go
identityClient := client.Identity()
```

### User Management
```go
// Get users
user, err := identityClient.GetUser(sysID)
user, err := identityClient.GetUserByUsername(username)
users, err := identityClient.ListUsers(&identity.UserFilter{
    Active:     &[]bool{true}[0],
    Department: "IT",
    Limit:      50,
})

// Create/Update users
newUser, err := identityClient.CreateUser(userData)
updated, err := identityClient.UpdateUser(sysID, updates)
err := identityClient.DeleteUser(sysID) // Sets active=false
```

### Role Management
```go
roleClient := identityClient.NewRoleClient()

// Get roles
role, err := roleClient.GetRole(sysID)
role, err := roleClient.GetRoleByName("admin")
roles, err := roleClient.ListRoles(&identity.RoleFilter{Active: &[]bool{true}[0]})

// Role assignments
assignment, err := roleClient.AssignRoleToUser(userSysID, roleSysID)
err := roleClient.RemoveRoleFromUser(userSysID, roleSysID)
userRoles, err := roleClient.GetUserRoles(userSysID)
roleUsers, err := roleClient.GetRoleUsers(roleSysID)
```

### Group Management
```go
groupClient := identityClient.NewGroupClient()

// Get groups
group, err := groupClient.GetGroup(sysID)
groups, err := groupClient.ListGroups(&identity.GroupFilter{Active: &[]bool{true}[0]})

// Group membership
member, err := groupClient.AddUserToGroup(userSysID, groupSysID)
err := groupClient.RemoveUserFromGroup(userSysID, groupSysID)
members, err := groupClient.GetGroupMembers(groupSysID)
userGroups, err := groupClient.GetUserGroups(userSysID)
```

### Access Control
```go
accessClient := identityClient.NewAccessClient()

// Check permissions
result, err := accessClient.CheckAccess(&identity.AccessCheckRequest{
    UserSysID: userSysID,
    Table:     "incident",
    Operation: "read",
})

// Session management
sessions, err := accessClient.GetActiveSessions()
userSessions, err := accessClient.GetUserSessions(userSysID)
err := accessClient.InvalidateUserSessions(userSysID)

// User preferences
prefs, err := accessClient.GetUserPreferences(userSysID)
pref, err := accessClient.SetUserPreference(userSysID, "theme", "dark")
err := accessClient.DeleteUserPreference(userSysID, "theme")
```

## Aggregate API

See [aggregate_api.md](aggregate_api.md) for detailed documentation.

### Basic Usage
```go
aggClient := client.Aggregate("incident")

// Simple count
count, err := aggClient.CountRecords(nil)

// Complex aggregation
result, err := aggClient.NewQuery().
    CountAll("total").
    Avg("priority", "avg_priority").
    GroupByField("state", "incident_state").
    Execute()
```

## Batch API

See [batch_api.md](batch_api.md) for detailed documentation.

### Basic Usage
```go
batchClient := client.Batch()

// Batch create
result, err := batchClient.CreateMultiple("incident", []map[string]interface{}{
    {"short_description": "Issue 1"},
    {"short_description": "Issue 2"},
})

// Mixed operations
result, err := batchClient.NewBatch().
    Create("create1", "incident", data1).
    Update("update1", "incident", sysID, updates).
    Delete("delete1", "incident", oldSysID).
    Execute()
```

## Catalog API

See [catalog_api.md](catalog_api.md) for detailed documentation.

### Basic Usage
```go
catalogClient := client.Catalog()

// Browse catalog
catalogs, err := catalogClient.ListCatalogs()
items, err := catalogClient.SearchItems("laptop")

// Order items
variables := map[string]interface{}{"cpu": "i7", "memory": "16GB"}
order, err := catalogClient.OrderNow(itemSysID, 1, variables)
```

## CMDB API

### CI Management
```go
cmdbClient := client.CMDB()

// Get Configuration Items
ci, err := cmdbClient.GetCI(sysID)
cis, err := cmdbClient.ListCIs(&cmdb.CIFilter{
    Class:  "cmdb_ci_server",
    Active: &[]bool{true}[0],
})

// Create/Update CIs
newCI, err := cmdbClient.CreateCI("cmdb_ci_server", ciData)
updated, err := cmdbClient.UpdateCI(sysID, updates)
```

### Relationship Management
```go
// Get relationships
rels, err := cmdbClient.GetCIRelationships(ciSysID)
parents, err := cmdbClient.GetCIParents(ciSysID)
children, err := cmdbClient.GetCIChildren(ciSysID)

// Create relationships
rel, err := cmdbClient.CreateRelationship(parentSysID, childSysID, relType)
```

### Class Management
```go
// Get classes
class, err := cmdbClient.GetCIClass(className)
classes, err := cmdbClient.ListCIClasses()
hierarchy, err := cmdbClient.GetClassHierarchy(className)
```

## Attachment API

### File Operations
```go
attachClient := client.Attachment()

// List attachments
attachments, err := attachClient.List("incident", recordSysID)

// Upload files
result, err := attachClient.Upload("incident", recordSysID, "/path/to/file.pdf")

// Download files
err := attachClient.Download(attachmentSysID, "/path/to/save/file.pdf")

// Delete attachments
err := attachClient.Delete(attachmentSysID)
```

## ImportSet API

### Import Operations
```go
importClient := client.ImportSet()

// Import data
result, err := importClient.ImportData("import_table", []map[string]interface{}{
    {"field1": "value1", "field2": "value2"},
    {"field1": "value3", "field2": "value4"},
})

// Check import status
status, err := importClient.GetImportStatus(importSysID)
```

## Query Builder

### Basic Queries
```go
import "github.com/Krive/ServiceNow Toolkit/pkg/servicenow/query"

// Simple conditions
q := query.New().Equals("active", true)
q := query.New().NotEquals("state", "closed")
q := query.New().Contains("short_description", "network")
q := query.New().StartsWith("number", "INC")
q := query.New().EndsWith("email", "@company.com")

// Numeric conditions
q := query.New().GreaterThan("priority", 2)
q := query.New().LessThan("urgency", 3)
q := query.New().Between("sys_created_on", startDate, endDate)

// Logical operators
q := query.New().
    Equals("active", true).
    And().
    LessThan("priority", 3).
    Or().
    Equals("urgency", 1)

// Use with table operations
records, err := client.Table("incident").
    Where(q).
    Execute()
```

### Advanced Queries
```go
// Subqueries
q := query.New().In("assigned_to", subQuery)
q := query.New().NotIn("state", []interface{}{6, 7, 8})

// Null checks
q := query.New().IsEmpty("resolved_at")
q := query.New().IsNotEmpty("assignment_group")

// Date queries
q := query.New().
    RelativeDate("sys_created_on", "today").
    And().
    RelativeDate("sys_updated_on", "last 7 days")
```

## Error Handling

### ServiceNow Errors
```go
if err != nil {
    if snErr, ok := core.IsServiceNowError(err); ok {
        switch snErr.Type {
        case core.ErrorTypeAuthentication:
            // Handle auth errors
        case core.ErrorTypeRateLimit:
            // Handle rate limiting
        case core.ErrorTypeValidation:
            // Handle validation errors
        }
        
        if snErr.IsRetryable() {
            // Retry the operation
        }
    }
}
```

### Error Types
- `ErrorTypeAuthentication` - Authentication failures
- `ErrorTypeAuthorization` - Permission denied
- `ErrorTypeRateLimit` - Rate limit exceeded
- `ErrorTypeValidation` - Invalid data
- `ErrorTypeNotFound` - Resource not found
- `ErrorTypeTimeout` - Request timeout
- `ErrorTypeNetwork` - Network issues
- `ErrorTypeServer` - Server errors
- `ErrorTypeClient` - Client errors
- `ErrorTypeUnknown` - Unknown errors

### Context and Timeouts
```go
// Set timeout for operations
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

// Use context with operations
records, err := client.Table("incident").
    Where("active", query.OpEquals, true).
    ExecuteWithContext(ctx)
```

## Rate Limiting

### Built-in Rate Limiting
```go
// Conservative settings (safe for production)
client := client.WithConservativeRateLimit()

// Aggressive settings (higher throughput)
client := client.WithAggressiveRateLimit()

// Custom settings
config := ratelimit.ServiceNowLimiterConfig{
    TableRequestsPerSecond:      5.0,
    AttachmentRequestsPerSecond: 2.0,
    ImportRequestsPerSecond:     1.0,
    DefaultRequestsPerSecond:    3.0,
    TableBurst:                  10,
    AttachmentBurst:             5,
    ImportBurst:                 2,
    DefaultBurst:                6,
}
client.SetRateLimitConfig(config)
```

## Retry Configuration

### Built-in Retry Policies
```go
// Minimal retry
client := client.WithMinimalRetry()

// Aggressive retry
client := client.WithAggressiveRetry()

// Custom retry
config := retry.Config{
    MaxAttempts: 5,
    BaseDelay:   500 * time.Millisecond,
    MaxDelay:    30 * time.Second,
    Multiplier:  2.0,
    Jitter:      true,
    RetryOn: []core.ErrorType{
        core.ErrorTypeRateLimit,
        core.ErrorTypeTimeout,
        core.ErrorTypeNetwork,
        core.ErrorTypeServer,
    },
}
client.SetRetryConfig(config)
```