# ServiceNow Batch API Reference

The ServiceNow Batch API in ServiceNow Toolkit allows you to perform multiple operations in a single HTTP request, significantly improving performance for bulk operations. This API wraps ServiceNow's `/api/now/batch` endpoint.

## Quick Start

```go
import "github.com/Krive/ServiceNow Toolkit/pkg/servicenow"

client, _ := servicenow.NewClientBasicAuth(instanceURL, username, password)
batchClient := client.Batch()

// Create multiple records in one request
records := []map[string]interface{}{
    {"short_description": "Incident 1", "category": "software"},
    {"short_description": "Incident 2", "category": "hardware"},
}

result, err := batchClient.CreateMultiple("incident", records)
fmt.Printf("Created %d incidents successfully\n", result.SuccessfulRequests)
```

## BatchClient Methods

### Creating a Batch Client

```go
// Get batch client from main client
batchClient := client.Batch()
```

### Convenience Methods

These methods provide simple interfaces for common batch operations:

#### CreateMultiple
Create multiple records of the same type.

```go
records := []map[string]interface{}{
    {
        "short_description": "First incident",
        "category": "software",
        "urgency": 3,
    },
    {
        "short_description": "Second incident", 
        "category": "hardware",
        "urgency": 2,
    },
}

result, err := batchClient.CreateMultiple("incident", records)
// With context
result, err := batchClient.CreateMultipleWithContext(ctx, "incident", records)
```

#### UpdateMultiple
Update multiple records with different data.

```go
updates := map[string]map[string]interface{}{
    "sys_id_1": {
        "state": "2",
        "assigned_to": "admin",
        "comments": "Updated via batch",
    },
    "sys_id_2": {
        "state": "3", 
        "urgency": 1,
    },
}

result, err := batchClient.UpdateMultiple("incident", updates)
// With context
result, err := batchClient.UpdateMultipleWithContext(ctx, "incident", updates)
```

#### DeleteMultiple
Delete multiple records by sys_id.

```go
sysIDs := []string{
    "sys_id_1",
    "sys_id_2", 
    "sys_id_3",
}

result, err := batchClient.DeleteMultiple("incident", sysIDs)
// With context
result, err := batchClient.DeleteMultipleWithContext(ctx, "incident", sysIDs)
```

#### GetMultiple
Retrieve multiple records by sys_id.

```go
sysIDs := []string{"sys_id_1", "sys_id_2", "sys_id_3"}

result, err := batchClient.GetMultiple("incident", sysIDs)
// With context
result, err := batchClient.GetMultipleWithContext(ctx, "incident", sysIDs)
```

#### ExecuteMixed
Execute a combination of different operations in one batch.

```go
operations := batch.MixedOperations{
    Creates: []batch.CreateOperation{
        {
            ID:        "create_incident",
            TableName: "incident",
            Data: map[string]interface{}{
                "short_description": "New incident",
                "category": "software",
            },
        },
    },
    Updates: []batch.UpdateOperation{
        {
            ID:        "update_incident",
            TableName: "incident", 
            SysID:     "existing_sys_id",
            Data: map[string]interface{}{
                "state": "2",
            },
        },
    },
    Deletes: []batch.DeleteOperation{
        {
            ID:        "delete_incident",
            TableName: "incident",
            SysID:     "old_sys_id",
        },
    },
    Gets: []batch.GetOperation{
        {
            ID:        "get_incident",
            TableName: "incident",
            SysID:     "another_sys_id",
        },
    },
}

result, err := batchClient.ExecuteMixed(operations)
// With context
result, err := batchClient.ExecuteMixedWithContext(ctx, operations)
```

## BatchBuilder (Advanced Usage)

For detailed control over batch requests, use the batch builder:

```go
builder := batchClient.NewBatch()
```

### Configuration Methods

#### WithRequestID
Set a custom batch request ID for tracking.

```go
builder.WithRequestID("my_custom_batch_001")
```

#### WithEnforceOrder
Enable sequential execution of requests (default: false).

```go
builder.WithEnforceOrder(true)
```

**Note**: ServiceNow batch requests are independent by design. `enforce_order` controls execution order but doesn't allow dependent operations where one request uses results from another.

### Adding Operations

#### Create Operation
```go
builder.Create("request_id", "table_name", map[string]interface{}{
    "short_description": "New record",
    "category": "software",
})
```

#### Update Operation
```go
builder.Update("request_id", "table_name", "sys_id", map[string]interface{}{
    "state": "2",
    "assigned_to": "user_id",
})
```

#### Replace Operation (PUT)
```go
builder.Replace("request_id", "table_name", "sys_id", map[string]interface{}{
    "short_description": "Completely replaced record",
    "state": "1",
})
```

#### Delete Operation
```go
builder.Delete("request_id", "table_name", "sys_id")
```

#### Get Operation
```go
builder.Get("request_id", "/api/now/table/table_name/sys_id")
```

#### Custom Request
For operations not covered by convenience methods:

```go
customRequest := batch.RestRequest{
    ID:     "custom_operation",
    URL:    "/api/now/table/sys_user?sysparm_query=active=true&sysparm_limit=5",
    Method: batch.MethodGET,
    Headers: []batch.Header{
        {Name: "Accept", Value: "application/json"},
    },
    ExcludeResponseHeaders: true,
}
builder.AddCustomRequest(customRequest)
```

### Execution

#### Execute Methods
```go
// Basic execution
result, err := builder.Execute()

// With context
ctx := context.WithTimeout(context.Background(), 30*time.Second)
result, err := builder.ExecuteWithContext(ctx)
```

### Method Chaining
All builder methods return the builder for fluent chaining:

```go
result, err := batchClient.NewBatch().
    WithRequestID("chained_example").
    WithEnforceOrder(true).
    Create("create_1", "incident", createData).
    Update("update_1", "incident", "sys_id", updateData).
    Delete("delete_1", "incident", "old_sys_id").
    Execute()
```

## BatchResult Structure

The result provides comprehensive information about the batch execution:

```go
type BatchResult struct {
    BatchRequestID     string
    TotalRequests      int
    SuccessfulRequests int
    FailedRequests     int
    Results            map[string]*RequestResult  // Successful requests
    Errors             map[string]*RequestError   // Failed requests
}
```

### Accessing Results

#### Individual Results
```go
// Get specific result
if result, exists := batchResult.GetResult("request_id"); exists {
    fmt.Printf("Status: %d, Execution time: %v\n", 
        result.StatusCode, result.ExecutionTime)
}

// Get specific error
if error, exists := batchResult.GetError("request_id"); exists {
    fmt.Printf("Error: %s - %s\n", error.StatusText, error.ErrorDetail)
}

// Check if request succeeded
if batchResult.IsSuccess("request_id") {
    fmt.Println("Request succeeded")
}
```

#### All Results
```go
// Get all successful results
for id, result := range batchResult.GetAllSuccessful() {
    fmt.Printf("Success %s: %d\n", id, result.StatusCode)
}

// Get all errors
for id, error := range batchResult.GetAllErrors() {
    fmt.Printf("Error %s: %s\n", id, error.ErrorDetail)  
}

// Check if any requests failed
if batchResult.HasErrors() {
    fmt.Println("Some requests failed")
}
```

### RequestResult Structure

```go
type RequestResult struct {
    ID            string
    StatusCode    int
    StatusText    string
    Data          map[string]interface{} // Decoded JSON response
    ExecutionTime time.Duration
}
```

### RequestError Structure

```go
type RequestError struct {
    ID          string
    StatusCode  int
    StatusText  string
    ErrorDetail string
}
```

## Data Extraction Helpers

### ExtractRecordData
Extract record data from a successful result:

```go
record, err := batch.ExtractRecordData(requestResult)
if err != nil {
    log.Printf("Failed to extract record: %v", err)
    return
}

fmt.Printf("Record number: %s\n", record["number"])
fmt.Printf("Sys ID: %s\n", record["sys_id"])
```

### ExtractMultipleRecords
Extract records from multiple results:

```go
records, err := batch.ExtractMultipleRecords(batchResult.GetAllSuccessful())
if err != nil {
    log.Printf("Failed to extract records: %v", err)
    return
}

for _, record := range records {
    fmt.Printf("Record: %s\n", record["number"])
}
```

## HTTP Methods and Operations

### Supported HTTP Methods
- `batch.MethodGET` - Retrieve records
- `batch.MethodPOST` - Create records
- `batch.MethodPATCH` - Update records (partial)
- `batch.MethodPUT` - Replace records (full)
- `batch.MethodDELETE` - Delete records

### Operation Types

#### CREATE (POST)
Creates new records. Returns the created record with sys_id.

```go
builder.Create("create_1", "incident", map[string]interface{}{
    "short_description": "New incident",
    "urgency": 3,
})
```

#### UPDATE (PATCH)
Partially updates existing records. Only specified fields are modified.

```go
builder.Update("update_1", "incident", "sys_id_123", map[string]interface{}{
    "state": "2",
    "comments": "Updated via batch",
})
```

#### REPLACE (PUT)
Completely replaces a record. Unspecified fields may be reset to defaults.

```go
builder.Replace("replace_1", "incident", "sys_id_123", map[string]interface{}{
    "short_description": "Completely new description",
    "state": "1",
    // Other fields may be reset to defaults
})
```

#### DELETE
Removes records. No request body needed.

```go
builder.Delete("delete_1", "incident", "sys_id_123")
```

#### GET
Retrieves records. Supports query parameters in URL.

```go
// Get specific record
builder.Get("get_1", "/api/now/table/incident/sys_id_123")

// Get with query
builder.Get("get_filtered", "/api/now/table/incident?sysparm_query=active=true&sysparm_limit=10")
```

## Error Handling

### Batch-Level Errors
Complete batch failure (network issues, authentication, etc.):

```go
result, err := batchClient.CreateMultiple("incident", records)
if err != nil {
    log.Printf("Batch execution failed: %v", err)
    return
}
```

### Request-Level Errors
Individual request failures within a successful batch:

```go
if result.HasErrors() {
    fmt.Printf("%d requests failed:\n", result.FailedRequests)
    
    for id, reqErr := range result.GetAllErrors() {
        fmt.Printf("  %s: %d %s", id, reqErr.StatusCode, reqErr.StatusText)
        if reqErr.ErrorDetail != "" {
            fmt.Printf(" - %s", reqErr.ErrorDetail)
        }
        fmt.Println()
    }
}
```

### Common Error Scenarios

#### Invalid Field Names
```go
// This will fail at the request level
data := map[string]interface{}{
    "short_description": "Valid field",
    "invalid_field_name": "This field doesn't exist",
}
```

#### Missing Required Fields
```go
// Some tables require specific fields
data := map[string]interface{}{
    // Missing required fields like 'short_description' for incidents
    "category": "software",
}
```

#### Invalid References
```go
// Invalid sys_id references
data := map[string]interface{}{
    "assigned_to": "non_existent_user_id",
}
```

#### Record Not Found (GET/UPDATE/DELETE)
```go
// Trying to operate on non-existent records
builder.Get("get_missing", "/api/now/table/incident/non_existent_sys_id")
```

### Error Recovery Patterns

#### Retry Failed Operations
```go
result, err := batchClient.CreateMultiple("incident", records)
if err != nil {
    return err
}

// Retry failed operations individually
if result.HasErrors() {
    for id, reqErr := range result.GetAllErrors() {
        log.Printf("Retrying failed operation %s", id)
        // Implement retry logic based on error type
    }
}
```

#### Partial Success Handling
```go
result, err := batchClient.UpdateMultiple("incident", updates)
if err != nil {
    return err
}

fmt.Printf("Batch completed: %d/%d successful\n", 
    result.SuccessfulRequests, result.TotalRequests)

// Process successful updates
for id, reqResult := range result.GetAllSuccessful() {
    // Handle successful operations
}

// Log failed updates for investigation
for id, reqErr := range result.GetAllErrors() {
    log.Printf("Failed to update %s: %s", id, reqErr.ErrorDetail)
}
```

## Performance and Best Practices

### Batch Size Recommendations
- **Small batches (5-20 requests)**: Better error isolation, faster individual processing
- **Medium batches (20-50 requests)**: Good balance of performance and manageability
- **Large batches (50+ requests)**: Maximum throughput, but harder error handling

```go
// Process large datasets in chunks
func processBatchesInChunks(client *servicenow.Client, records []map[string]interface{}) error {
    batchClient := client.Batch()
    batchSize := 50
    
    for i := 0; i < len(records); i += batchSize {
        end := i + batchSize
        if end > len(records) {
            end = len(records)
        }
        
        batch := records[i:end]
        result, err := batchClient.CreateMultiple("incident", batch)
        if err != nil {
            return fmt.Errorf("batch %d failed: %w", i/batchSize+1, err)
        }
        
        log.Printf("Batch %d: %d successful, %d failed", 
            i/batchSize+1, result.SuccessfulRequests, result.FailedRequests)
    }
    
    return nil
}
```

### Performance Benefits
Based on ServiceNow documentation and testing:
- **56-66% throughput improvement** for bulk operations
- **Reduced network overhead**: Single authentication and connection
- **Lower transaction count**: Fewer semaphore pool transactions
- **Better resource utilization**: More efficient server processing

### Context and Timeouts
Always use context with appropriate timeouts:

```go
// Set reasonable timeout for batch operations
ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
defer cancel()

result, err := batchClient.CreateMultipleWithContext(ctx, "incident", largeRecordSet)
if err != nil {
    if ctx.Err() == context.DeadlineExceeded {
        log.Println("Batch operation timed out")
        // Handle timeout appropriately
    }
    return err
}
```

### Error Isolation
Structure batches to minimize cascading failures:

```go
// Group related operations
relatedUpdates := map[string]map[string]interface{}{
    "incident_1": {"state": "2"},
    "incident_2": {"state": "2"}, 
    "incident_3": {"state": "2"},
}

// Separate from unrelated operations
cleanupDeletes := []string{"old_sys_id_1", "old_sys_id_2"}

// Execute as separate batches
updateResult, _ := batchClient.UpdateMultiple("incident", relatedUpdates)
deleteResult, _ := batchClient.DeleteMultiple("incident", cleanupDeletes)
```

## Advanced Usage Patterns

### Audit Trail
```go
func auditAndUpdate(batchClient *batch.BatchClient, updates map[string]map[string]interface{}) error {
    // First, get current state for audit
    var sysIDs []string
    for sysID := range updates {
        sysIDs = append(sysIDs, sysID)
    }
    
    auditResult, err := batchClient.GetMultiple("incident", sysIDs)
    if err != nil {
        return err
    }
    
    // Log current state
    for id, reqResult := range auditResult.GetAllSuccessful() {
        record, _ := batch.ExtractRecordData(reqResult)
        log.Printf("Before update %s: state=%s", id, record["state"])
    }
    
    // Perform updates
    updateResult, err := batchClient.UpdateMultiple("incident", updates)
    if err != nil {
        return err
    }
    
    // Log results
    log.Printf("Updates: %d successful, %d failed", 
        updateResult.SuccessfulRequests, updateResult.FailedRequests)
    
    return nil
}
```

### Transaction-like Behavior
While batch operations don't provide true transactions, you can implement rollback patterns:

```go
func createWithRollback(batchClient *batch.BatchClient, records []map[string]interface{}) error {
    result, err := batchClient.CreateMultiple("incident", records)
    if err != nil {
        return err
    }
    
    // If any creation failed, rollback successful ones
    if result.HasErrors() {
        var createdSysIDs []string
        for _, reqResult := range result.GetAllSuccessful() {
            record, err := batch.ExtractRecordData(reqResult)
            if err == nil {
                createdSysIDs = append(createdSysIDs, record["sys_id"].(string))
            }
        }
        
        if len(createdSysIDs) > 0 {
            log.Printf("Rolling back %d successful creations due to batch errors", len(createdSysIDs))
            batchClient.DeleteMultiple("incident", createdSysIDs)
        }
        
        return fmt.Errorf("batch creation partially failed, rolled back successful operations")
    }
    
    return nil
}
```

### Progress Tracking
```go
func createWithProgress(batchClient *batch.BatchClient, records []map[string]interface{}) error {
    batchSize := 25
    total := len(records)
    processed := 0
    
    for i := 0; i < total; i += batchSize {
        end := i + batchSize
        if end > total {
            end = total
        }
        
        batch := records[i:end]
        result, err := batchClient.CreateMultiple("incident", batch)
        if err != nil {
            return err
        }
        
        processed += result.SuccessfulRequests
        progress := float64(processed) / float64(total) * 100
        
        fmt.Printf("Progress: %.1f%% (%d/%d)\n", progress, processed, total)
        
        if result.HasErrors() {
            log.Printf("Batch %d had %d errors", i/batchSize+1, result.FailedRequests)
        }
    }
    
    return nil
}
```

## ServiceNow API Mapping

The batch API maps directly to ServiceNow's `/api/now/batch` endpoint:

### Request Structure
```json
{
  "batch_request_id": "unique_identifier",
  "enforce_order": false,
  "rest_requests": [
    {
      "id": "request_1",
      "url": "/api/now/table/incident",
      "method": "POST",
      "headers": [
        {"name": "Content-Type", "value": "application/json"}
      ],
      "body": "base64_encoded_json_data",
      "exclude_response_headers": true
    }
  ]
}
```

### Response Structure
```json
{
  "batch_request_id": "unique_identifier",
  "serviced_requests": [
    {
      "id": "request_1",
      "status_code": 201,
      "status_text": "Created",
      "body": "base64_encoded_response",
      "execution_time": 150
    }
  ],
  "unserviced_requests": []
}
```

### Rate Limits
- Batch API follows the same rate limits as individual requests
- Default: 1000 requests per hour per user
- Rate limits apply to the total number of operations in all batches
- Use rate limiting features in the core client for automatic handling

## Troubleshooting

### Common Issues

#### Empty Batch Error
```
Error: batch request cannot be empty
```
**Solution**: Add at least one operation before executing the batch.

#### Base64 Encoding Issues
The SDK handles base64 encoding automatically, but if you see encoding errors:
- Ensure your data structures are JSON-serializable
- Check for circular references in your data

#### Context Timeout
```
Error: context deadline exceeded
```
**Solutions**:
- Increase timeout duration for large batches
- Reduce batch size
- Check network connectivity

#### Rate Limit Exceeded
```
Error: 429 Too Many Requests
```
**Solutions**:
- Implement exponential backoff (built into core client)
- Reduce batch frequency
- Use smaller batch sizes
- Distribute operations across time

### Debug Information
Enable debug logging to see generated requests:

```go
// This would show the actual batch request structure
log.Printf("Batch request details: %+v", batchRequest)
```

### Testing
Test batch operations thoroughly:

```go
func TestBatchOperations(t *testing.T) {
    // Test with small batches first
    testRecords := []map[string]interface{}{
        {"short_description": "Test 1"},
        {"short_description": "Test 2"},
    }
    
    result, err := batchClient.CreateMultiple("incident", testRecords)
    if err != nil {
        t.Fatalf("Batch creation failed: %v", err)
    }
    
    if result.SuccessfulRequests != len(testRecords) {
        t.Errorf("Expected %d successful requests, got %d", 
            len(testRecords), result.SuccessfulRequests)
    }
    
    // Clean up test data
    var sysIDs []string
    for _, reqResult := range result.GetAllSuccessful() {
        record, _ := batch.ExtractRecordData(reqResult)
        sysIDs = append(sysIDs, record["sys_id"].(string))
    }
    
    batchClient.DeleteMultiple("incident", sysIDs)
}
```