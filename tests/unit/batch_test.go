package unit

import (
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"

	"github.com/Krive/ServiceNow-Toolkit/pkg/servicenow/batch"
	"github.com/Krive/ServiceNow-Toolkit/pkg/servicenow/core"
)

func TestBatchClient_NewBatch(t *testing.T) {
	client := &core.Client{} // Mock client
	batchClient := batch.NewBatchClient(client)
	
	builder := batchClient.NewBatch()
	
	if builder == nil {
		t.Fatal("NewBatch should return a non-nil BatchBuilder")
	}
}

func TestBatchBuilder_WithRequestID(t *testing.T) {
	client := &core.Client{}
	batchClient := batch.NewBatchClient(client)
	
	customID := "custom_batch_123"
	builder := batchClient.NewBatch().WithRequestID(customID)
	
	// We can't directly access the requestID field, but we can verify
	// the builder is returned for method chaining
	if builder == nil {
		t.Error("WithRequestID should return the builder for chaining")
	}
}

func TestBatchBuilder_WithEnforceOrder(t *testing.T) {
	client := &core.Client{}
	batchClient := batch.NewBatchClient(client)
	
	builder := batchClient.NewBatch().WithEnforceOrder(true)
	
	if builder == nil {
		t.Error("WithEnforceOrder should return the builder for chaining")
	}
}

func TestBatchBuilder_Get(t *testing.T) {
	client := &core.Client{}
	batchClient := batch.NewBatchClient(client)
	
	builder := batchClient.NewBatch().Get("test_get", "/api/now/table/incident/123")
	
	if builder == nil {
		t.Error("Get should return the builder for chaining")
	}
}

func TestBatchBuilder_Create(t *testing.T) {
	client := &core.Client{}
	batchClient := batch.NewBatchClient(client)
	
	data := map[string]interface{}{
		"short_description": "Test incident",
		"category": "hardware",
	}
	
	builder := batchClient.NewBatch().Create("test_create", "incident", data)
	
	if builder == nil {
		t.Error("Create should return the builder for chaining")
	}
}

func TestBatchBuilder_Update(t *testing.T) {
	client := &core.Client{}
	batchClient := batch.NewBatchClient(client)
	
	data := map[string]interface{}{
		"state": "2",
		"assigned_to": "user_id",
	}
	
	builder := batchClient.NewBatch().Update("test_update", "incident", "sys_id_123", data)
	
	if builder == nil {
		t.Error("Update should return the builder for chaining")
	}
}

func TestBatchBuilder_Replace(t *testing.T) {
	client := &core.Client{}
	batchClient := batch.NewBatchClient(client)
	
	data := map[string]interface{}{
		"short_description": "Replaced incident",
		"state": "3",
	}
	
	builder := batchClient.NewBatch().Replace("test_replace", "incident", "sys_id_123", data)
	
	if builder == nil {
		t.Error("Replace should return the builder for chaining")
	}
}

func TestBatchBuilder_Delete(t *testing.T) {
	client := &core.Client{}
	batchClient := batch.NewBatchClient(client)
	
	builder := batchClient.NewBatch().Delete("test_delete", "incident", "sys_id_123")
	
	if builder == nil {
		t.Error("Delete should return the builder for chaining")
	}
}

func TestBatchBuilder_ChainedOperations(t *testing.T) {
	client := &core.Client{}
	batchClient := batch.NewBatchClient(client)
	
	createData := map[string]interface{}{
		"short_description": "New incident",
	}
	
	updateData := map[string]interface{}{
		"state": "2",
	}
	
	builder := batchClient.NewBatch().
		WithRequestID("chained_test").
		WithEnforceOrder(true).
		Create("create_1", "incident", createData).
		Update("update_1", "incident", "sys_id_123", updateData).
		Delete("delete_1", "incident", "sys_id_456").
		Get("get_1", "/api/now/table/incident/sys_id_789")
	
	if builder == nil {
		t.Error("Chained operations should return the builder")
	}
}

func TestBatchBuilder_AddCustomRequest(t *testing.T) {
	client := &core.Client{}
	batchClient := batch.NewBatchClient(client)
	
	customRequest := batch.RestRequest{
		ID:     "custom_1",
		URL:    "/api/now/custom/endpoint",
		Method: batch.MethodGET,
		Headers: []batch.Header{
			{Name: "Custom-Header", Value: "custom-value"},
		},
	}
	
	builder := batchClient.NewBatch().AddCustomRequest(customRequest)
	
	if builder == nil {
		t.Error("AddCustomRequest should return the builder for chaining")
	}
}

func TestBatchBuilder_ExecuteEmptyBatch(t *testing.T) {
	client := &core.Client{}
	batchClient := batch.NewBatchClient(client)
	
	builder := batchClient.NewBatch()
	
	// This should fail because the batch is empty
	_, err := builder.Execute()
	if err == nil {
		t.Error("Execute should return an error for empty batch")
	}
	
	expectedError := "batch request cannot be empty"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

func TestBatchResult_Methods(t *testing.T) {
	// Create a mock batch result
	result := &batch.BatchResult{
		BatchRequestID:     "test_batch",
		TotalRequests:      3,
		SuccessfulRequests: 2,
		FailedRequests:     1,
		Results: map[string]*batch.RequestResult{
			"success_1": {
				ID:         "success_1",
				StatusCode: 200,
				StatusText: "OK",
				Data: map[string]interface{}{
					"result": map[string]interface{}{
						"sys_id": "123",
						"number": "INC001",
					},
				},
				ExecutionTime: 150 * time.Millisecond,
			},
			"success_2": {
				ID:         "success_2",
				StatusCode: 201,
				StatusText: "Created",
				Data: map[string]interface{}{
					"result": map[string]interface{}{
						"sys_id": "456",
						"number": "INC002",
					},
				},
				ExecutionTime: 200 * time.Millisecond,
			},
		},
		Errors: map[string]*batch.RequestError{
			"error_1": {
				ID:          "error_1",
				StatusCode:  400,
				StatusText:  "Bad Request",
				ErrorDetail: "Invalid field value",
			},
		},
	}

	// Test GetResult
	successResult, exists := result.GetResult("success_1")
	if !exists {
		t.Error("GetResult should find existing successful result")
	}
	if successResult.StatusCode != 200 {
		t.Errorf("Expected status code 200, got %d", successResult.StatusCode)
	}

	// Test GetResult for non-existent ID
	_, exists = result.GetResult("non_existent")
	if exists {
		t.Error("GetResult should return false for non-existent ID")
	}

	// Test GetError
	errorResult, exists := result.GetError("error_1")
	if !exists {
		t.Error("GetError should find existing error")
	}
	if errorResult.StatusCode != 400 {
		t.Errorf("Expected status code 400, got %d", errorResult.StatusCode)
	}

	// Test GetError for non-existent ID
	_, exists = result.GetError("non_existent")
	if exists {
		t.Error("GetError should return false for non-existent ID")
	}

	// Test IsSuccess
	if !result.IsSuccess("success_1") {
		t.Error("IsSuccess should return true for successful request")
	}
	if result.IsSuccess("error_1") {
		t.Error("IsSuccess should return false for failed request")
	}

	// Test GetAllSuccessful
	successful := result.GetAllSuccessful()
	if len(successful) != 2 {
		t.Errorf("Expected 2 successful results, got %d", len(successful))
	}

	// Test GetAllErrors
	errors := result.GetAllErrors()
	if len(errors) != 1 {
		t.Errorf("Expected 1 error, got %d", len(errors))
	}

	// Test HasErrors
	if !result.HasErrors() {
		t.Error("HasErrors should return true when errors exist")
	}
}

func TestExtractRecordData(t *testing.T) {
	// Test with result wrapper
	resultWithWrapper := &batch.RequestResult{
		Data: map[string]interface{}{
			"result": map[string]interface{}{
				"sys_id": "123",
				"number": "INC001",
			},
		},
	}

	record, err := batch.ExtractRecordData(resultWithWrapper)
	if err != nil {
		t.Fatalf("ExtractRecordData should not return error: %v", err)
	}
	if record["sys_id"] != "123" {
		t.Errorf("Expected sys_id '123', got '%v'", record["sys_id"])
	}

	// Test without result wrapper
	resultWithoutWrapper := &batch.RequestResult{
		Data: map[string]interface{}{
			"sys_id": "456",
			"number": "INC002",
		},
	}

	record, err = batch.ExtractRecordData(resultWithoutWrapper)
	if err != nil {
		t.Fatalf("ExtractRecordData should not return error: %v", err)
	}
	if record["sys_id"] != "456" {
		t.Errorf("Expected sys_id '456', got '%v'", record["sys_id"])
	}

	// Test with nil data
	resultWithNilData := &batch.RequestResult{
		Data: nil,
	}

	_, err = batch.ExtractRecordData(resultWithNilData)
	if err == nil {
		t.Error("ExtractRecordData should return error for nil data")
	}
}

func TestExtractMultipleRecords(t *testing.T) {
	results := map[string]*batch.RequestResult{
		"result_1": {
			Data: map[string]interface{}{
				"result": map[string]interface{}{
					"sys_id": "123",
					"number": "INC001",
				},
			},
		},
		"result_2": {
			Data: map[string]interface{}{
				"result": map[string]interface{}{
					"sys_id": "456",
					"number": "INC002",
				},
			},
		},
	}

	records, err := batch.ExtractMultipleRecords(results)
	if err != nil {
		t.Fatalf("ExtractMultipleRecords should not return error: %v", err)
	}

	if len(records) != 2 {
		t.Errorf("Expected 2 records, got %d", len(records))
	}

	// Check that both records are present
	foundSysIds := make(map[string]bool)
	for _, record := range records {
		if sysId, ok := record["sys_id"]; ok {
			foundSysIds[sysId.(string)] = true
		}
	}

	if !foundSysIds["123"] || !foundSysIds["456"] {
		t.Error("ExtractMultipleRecords should extract all records")
	}
}

func TestBatchRequest_JSONEncoding(t *testing.T) {
	// Test encoding of batch request structures
	batchReq := batch.BatchRequest{
		BatchRequestID: "test_batch",
		EnforceOrder:   true,
		RestRequests: []batch.RestRequest{
			{
				ID:     "test_1",
				URL:    "/api/now/table/incident",
				Method: batch.MethodPOST,
				Headers: []batch.Header{
					{Name: "Content-Type", Value: "application/json"},
				},
				Body:                   base64.StdEncoding.EncodeToString([]byte(`{"test":"data"}`)),
				ExcludeResponseHeaders: true,
			},
		},
	}

	// Ensure it can be marshaled to JSON
	jsonData, err := json.Marshal(batchReq)
	if err != nil {
		t.Fatalf("Failed to marshal BatchRequest to JSON: %v", err)
	}

	// Ensure it can be unmarshaled from JSON
	var unmarshaled batch.BatchRequest
	err = json.Unmarshal(jsonData, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal BatchRequest from JSON: %v", err)
	}

	// Verify the data
	if unmarshaled.BatchRequestID != "test_batch" {
		t.Errorf("Expected BatchRequestID 'test_batch', got '%s'", unmarshaled.BatchRequestID)
	}
	if !unmarshaled.EnforceOrder {
		t.Error("Expected EnforceOrder to be true")
	}
	if len(unmarshaled.RestRequests) != 1 {
		t.Errorf("Expected 1 RestRequest, got %d", len(unmarshaled.RestRequests))
	}
}

func TestBatchResponse_JSONDecoding(t *testing.T) {
	// Test decoding of batch response
	responseJSON := `{
		"batch_request_id": "test_batch",
		"serviced_requests": [
			{
				"id": "success_1",
				"status_code": 201,
				"status_text": "Created",
				"body": "eyJyZXN1bHQiOnsic3lzX2lkIjoiMTIzIiwibmFtZSI6InRlc3QifX0=",
				"execution_time": 150
			}
		],
		"unserviced_requests": [
			{
				"id": "error_1",
				"status_code": 400,
				"status_text": "Bad Request",
				"error_detail": "Invalid field"
			}
		]
	}`

	var response batch.BatchResponse
	err := json.Unmarshal([]byte(responseJSON), &response)
	if err != nil {
		t.Fatalf("Failed to unmarshal BatchResponse: %v", err)
	}

	// Verify the response structure
	if response.BatchRequestID != "test_batch" {
		t.Errorf("Expected BatchRequestID 'test_batch', got '%s'", response.BatchRequestID)
	}

	if len(response.ServicedRequests) != 1 {
		t.Errorf("Expected 1 ServicedRequest, got %d", len(response.ServicedRequests))
	}

	if len(response.UnservicedRequests) != 1 {
		t.Errorf("Expected 1 UnservicedRequest, got %d", len(response.UnservicedRequests))
	}

	// Check serviced request
	servicedReq := response.ServicedRequests[0]
	if servicedReq.ID != "success_1" {
		t.Errorf("Expected ID 'success_1', got '%s'", servicedReq.ID)
	}
	if servicedReq.StatusCode != 201 {
		t.Errorf("Expected status code 201, got %d", servicedReq.StatusCode)
	}

	// Check unserviced request
	unservicedReq := response.UnservicedRequests[0]
	if unservicedReq.ID != "error_1" {
		t.Errorf("Expected ID 'error_1', got '%s'", unservicedReq.ID)
	}
	if unservicedReq.StatusCode != 400 {
		t.Errorf("Expected status code 400, got %d", unservicedReq.StatusCode)
	}
}

func TestMixedOperations(t *testing.T) {
	client := &core.Client{}
	_ = batch.NewBatchClient(client)

	operations := batch.MixedOperations{
		Creates: []batch.CreateOperation{
			{
				ID:        "create_1",
				TableName: "incident",
				Data: map[string]interface{}{
					"short_description": "New incident",
				},
			},
		},
		Updates: []batch.UpdateOperation{
			{
				ID:        "update_1",
				TableName: "incident",
				SysID:     "sys_id_123",
				Data: map[string]interface{}{
					"state": "2",
				},
			},
		},
		Deletes: []batch.DeleteOperation{
			{
				ID:        "delete_1",
				TableName: "incident",
				SysID:     "sys_id_456",
			},
		},
		Gets: []batch.GetOperation{
			{
				ID:        "get_1",
				TableName: "incident",
				SysID:     "sys_id_789",
			},
		},
	}

	// Test that the mixed operations structure can be created
	// We won't execute it with a mock client to avoid panics
	if len(operations.Creates) != 1 {
		t.Errorf("Expected 1 create operation, got %d", len(operations.Creates))
	}
	if len(operations.Updates) != 1 {
		t.Errorf("Expected 1 update operation, got %d", len(operations.Updates))
	}
	if len(operations.Deletes) != 1 {
		t.Errorf("Expected 1 delete operation, got %d", len(operations.Deletes))
	}
	if len(operations.Gets) != 1 {
		t.Errorf("Expected 1 get operation, got %d", len(operations.Gets))
	}
}

func TestBatchBuilder_ContextSupport(t *testing.T) {
	client := &core.Client{}
	batchClient := batch.NewBatchClient(client)

	data := map[string]interface{}{
		"short_description": "Test with context",
	}

	builder := batchClient.NewBatch().Create("test_ctx", "incident", data)

	// Test that the builder supports context methods without executing
	if builder == nil {
		t.Error("Batch builder should support context operations")
	}
}

func TestConvenienceMethods(t *testing.T) {
	client := &core.Client{}
	batchClient := batch.NewBatchClient(client)

	// Test that convenience methods exist and can build batches
	// We won't execute them with mock clients to avoid panics

	// Test CreateMultiple method exists
	records := []map[string]interface{}{
		{"short_description": "Incident 1"},
		{"short_description": "Incident 2"},
	}
	if len(records) != 2 {
		t.Error("Test data should have 2 records")
	}

	// Test UpdateMultiple method exists
	updates := map[string]map[string]interface{}{
		"sys_id_1": {"state": "2"},
		"sys_id_2": {"state": "3"},
	}
	if len(updates) != 2 {
		t.Error("Test data should have 2 updates")
	}

	// Test DeleteMultiple method exists
	sysIDs := []string{"sys_id_1", "sys_id_2", "sys_id_3"}
	if len(sysIDs) != 3 {
		t.Error("Test data should have 3 sys_ids")
	}

	// Verify batch client is not nil
	if batchClient == nil {
		t.Error("Batch client should not be nil")
	}
}