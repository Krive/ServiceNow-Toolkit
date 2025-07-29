# ServiceNow Aggregate API Reference

The ServiceNow Aggregate API in ServiceNow Toolkit provides powerful data analysis capabilities, allowing you to perform statistical operations on your ServiceNow data without retrieving all records.

## Quick Start

```go
import "github.com/Krive/ServiceNow Toolkit/pkg/servicenow"

client, _ := servicenow.NewClientBasicAuth(instanceURL, username, password)
aggClient := client.Aggregate("incident")

// Count all incidents
count, err := aggClient.CountRecords(nil)
```

## AggregateClient Methods

### Creating an Aggregate Client

```go
// Get aggregate client for a specific table
aggClient := client.Aggregate("table_name")
```

### Convenience Methods

#### CountRecords
Count records with optional filtering.

```go
// Count all records
total, err := aggClient.CountRecords(nil)

// Count with filter
active, err := aggClient.CountRecords(
    query.New().Equals("active", true)
)

// With context
ctx := context.WithTimeout(context.Background(), 30*time.Second)
count, err := aggClient.CountRecordsWithContext(ctx, queryBuilder)
```

#### SumField
Sum values of a numeric field.

```go
prioritySum, err := aggClient.SumField("priority", 
    query.New().Equals("active", true)
)

// With context
prioritySum, err := aggClient.SumFieldWithContext(ctx, "priority", queryBuilder)
```

#### AvgField
Calculate average of a numeric field.

```go
avgPriority, err := aggClient.AvgField("priority", 
    query.New().Equals("active", true)
)

// With context
avgPriority, err := aggClient.AvgFieldWithContext(ctx, "priority", queryBuilder)
```

#### MinMaxField
Get both minimum and maximum values of a field.

```go
min, max, err := aggClient.MinMaxField("priority", 
    query.New().Equals("active", true)
)

// With context
min, max, err := aggClient.MinMaxFieldWithContext(ctx, "priority", queryBuilder)
```

## AggregateQuery Builder

For complex aggregations, use the query builder pattern:

```go
query := aggClient.NewQuery()
```

### Aggregate Operations

#### Count Operations
```go
// Count specific field
query.Count("field_name", "alias")

// Count all records (COUNT(*))
query.CountAll("total_count")
```

#### Mathematical Operations
```go
// Sum
query.Sum("field_name", "sum_alias")

// Average
query.Avg("field_name", "avg_alias")

// Minimum
query.Min("field_name", "min_alias")

// Maximum
query.Max("field_name", "max_alias")

// Standard Deviation
query.StdDev("field_name", "stddev_alias")

// Variance
query.Variance("field_name", "variance_alias")
```

#### Chaining Multiple Aggregates
```go
result, err := aggClient.NewQuery().
    CountAll("total").
    Sum("priority", "priority_sum").
    Avg("priority", "priority_avg").
    Min("priority", "min_priority").
    Max("priority", "max_priority").
    StdDev("priority", "priority_stddev").
    Execute()
```

### Grouping

#### GROUP BY
```go
// Group by single field
query.GroupByField("state", "state_name")

// Group by multiple fields
query.GroupByField("state", "state_name").
      GroupByField("priority", "priority_level")

// Group without alias
query.GroupByField("assignment_group", "")
```

### Filtering

#### WHERE Conditions
Use the same query builder syntax as table queries:

```go
query.Where("active", query.OpEquals, true).
      And().
      Where("priority", query.OpLessThan, 3).
      Or().
      Where("urgency", query.OpEquals, 1)
```

#### Convenience Methods
```go
// Equality
query.Equals("active", true)

// Contains
query.Contains("short_description", "network")

// Combination
query.Equals("active", true).
      And().
      Contains("description", "urgent")
```

#### HAVING Conditions
Filter aggregate results:

```go
query.CountAll("count").
      GroupByField("state", "").
      Having("COUNT(*) > 10").
      Having("AVG(priority) < 3")
```

### Sorting and Limiting

#### ORDER BY
```go
// Order by field
query.OrderBy("field_name", query.OrderAsc)
query.OrderBy("field_name", query.OrderDesc)

// Convenience methods
query.OrderByAsc("field_name")
query.OrderByDesc("field_name")

// Multiple ordering
query.OrderByDesc("count").
      OrderByAsc("state")
```

#### LIMIT and OFFSET
```go
query.Limit(50).
      Offset(100)
```

### Execution

#### Execute Methods
```go
// Basic execution
result, err := query.Execute()

// With context
ctx := context.WithTimeout(context.Background(), 30*time.Second)
result, err := query.ExecuteWithContext(ctx)
```

#### Building Parameters
Access the underlying query parameters:

```go
params := query.BuildParams()
// Returns map[string]string with ServiceNow API parameters
```

## AggregateResult Structure

The result contains two main components:

```go
type AggregateResult struct {
    Stats  map[string]interface{} // Single-row statistics
    Result []map[string]interface{} // Multi-row results (with GROUP BY)
}
```

### Accessing Results

#### Without GROUP BY
Results are in the `Stats` field:

```go
result, _ := aggClient.NewQuery().
    CountAll("total").
    Avg("priority", "avg_priority").
    Execute()

fmt.Printf("Total: %v\n", result.Stats["total"])
fmt.Printf("Average Priority: %v\n", result.Stats["avg_priority"])
```

#### With GROUP BY
Results are in the `Result` field:

```go
result, _ := aggClient.NewQuery().
    CountAll("count").
    GroupByField("state", "state_name").
    Execute()

for _, row := range result.Result {
    fmt.Printf("State %v: %v incidents\n", 
        row["state_name"], row["count"])
}
```

## Complete Examples

### Dashboard Metrics
```go
func getDashboardMetrics(client *servicenow.Client) {
    aggClient := client.Aggregate("incident")
    
    result, err := aggClient.NewQuery().
        CountAll("total_incidents").
        Count("state", "open_incidents").
        Avg("priority", "avg_priority").
        Min("sys_created_on", "earliest").
        Max("sys_created_on", "latest").
        Where("active", query.OpEquals, true).
        Execute()
    
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Dashboard Metrics:\n")
    fmt.Printf("  Total Active: %v\n", result.Stats["total_incidents"])
    fmt.Printf("  Average Priority: %v\n", result.Stats["avg_priority"])
}
```

### Performance Analysis by Department
```go
func analyzeDepartmentPerformance(client *servicenow.Client) {
    aggClient := client.Aggregate("incident")
    
    result, err := aggClient.NewQuery().
        CountAll("incident_count").
        Avg("priority", "avg_priority").
        Avg("urgency", "avg_urgency").
        GroupByField("assignment_group.department", "department").
        Where("active", query.OpEquals, true).
        Having("COUNT(*) > 5").
        OrderByDesc("incident_count").
        Limit(10).
        Execute()
    
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Println("Department Performance:")
    for _, row := range result.Result {
        fmt.Printf("  %v: %v incidents (avg priority: %v)\n",
            row["department"], row["incident_count"], row["avg_priority"])
    }
}
```

### Trend Analysis
```go
func analyzeTrends(client *servicenow.Client) {
    aggClient := client.Aggregate("incident")
    
    result, err := aggClient.NewQuery().
        CountAll("daily_count").
        Avg("priority", "daily_avg_priority").
        Where("sys_created_on", query.OpGreaterThan, "javascript:gs.daysAgoStart(30)").
        Execute()
    
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Last 30 Days:\n")
    fmt.Printf("  Incidents Created: %v\n", result.Stats["daily_count"])
    fmt.Printf("  Average Priority: %v\n", result.Stats["daily_avg_priority"])
}
```

## Advanced Usage

### Custom Aggregation with Raw Parameters
If you need functionality not covered by the builder, you can access the raw parameter building:

```go
query := aggClient.NewQuery().
    CountAll("total")

params := query.BuildParams()
params["custom_parameter"] = "custom_value"

// Execute with custom parameters
// (This would require using the core client directly)
```

### Error Handling
```go
result, err := aggClient.NewQuery().
    CountAll("total").
    Execute()

if err != nil {
    // Handle ServiceNow-specific errors
    if snErr, ok := err.(*core.ServiceNowError); ok {
        switch snErr.Type {
        case core.ErrorTypeRateLimit:
            // Handle rate limiting
        case core.ErrorTypeAuth:
            // Handle authentication errors
        default:
            // Handle other errors
        }
    }
}
```

### Context and Cancellation
```go
ctx, cancel := context.WithCancel(context.Background())

// Cancel after 10 seconds
go func() {
    time.Sleep(10 * time.Second)
    cancel()
}()

result, err := aggClient.NewQuery().
    CountAll("total").
    ExecuteWithContext(ctx)

if err == context.Canceled {
    fmt.Println("Query was cancelled")
}
```

## ServiceNow API Mapping

This aggregate API maps to ServiceNow's Stats API (`/stats/{table}`). The parameters generated include:

- `sysparm_query`: WHERE conditions
- `sysparm_sum_fields`: Aggregate functions
- `sysparm_group_by`: GROUP BY fields
- `sysparm_having`: HAVING conditions
- `sysparm_orderby`: ORDER BY clauses
- `sysparm_limit`: LIMIT
- `sysparm_offset`: OFFSET

## Best Practices

1. **Use Aliases**: Always provide meaningful aliases for aggregate fields
2. **Filter Early**: Use WHERE conditions to reduce data processing
3. **Limit Results**: Use LIMIT for large datasets
4. **Context Timeouts**: Always use context with timeouts for production code
5. **Error Handling**: Handle ServiceNow-specific errors appropriately
6. **Rate Limiting**: Configure appropriate rate limiting for your use case

## Troubleshooting

### Common Issues

1. **Empty Results**: Check your WHERE conditions and table name
2. **Permission Errors**: Ensure your user has read access to the table
3. **Invalid Aggregates**: Some fields may not support certain aggregate functions
4. **Rate Limiting**: Implement exponential backoff for rate limit errors

### Debug Information
Enable debug logging to see the generated ServiceNow API parameters:

```go
params := query.BuildParams()
fmt.Printf("Generated parameters: %+v\n", params)
```