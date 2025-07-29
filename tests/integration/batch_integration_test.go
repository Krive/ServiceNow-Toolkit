package integration

import (
	"context"
	"log"
	"os"
	"testing"
	"time"

	"github.com/Krive/ServiceNow-Toolkit/pkg/servicenow"
	"github.com/Krive/ServiceNow-Toolkit/pkg/servicenow/batch"
)

func setupBatchClient(t *testing.T) *servicenow.Client {
	username := os.Getenv("SN_USERNAME")
	password := os.Getenv("SN_PASSWORD")
	instanceURL := os.Getenv("SN_INSTANCE_URL")

	if username == "" || password == "" || instanceURL == "" {
		t.Skip("ServiceNow credentials not provided, skipping batch integration tests")
	}

	client, err := servicenow.NewClientBasicAuth(instanceURL, username, password)
	if err != nil {
		t.Fatalf("Failed to create ServiceNow client: %v", err)
	}

	return client
}

func TestBatchIntegration_CreateMultiple(t *testing.T) {
	client := setupBatchClient(t)
	batchClient := client.Batch()

	// Create multiple test incidents
	records := []map[string]interface{}{
		{
			"short_description": "Batch Test Incident 1",
			"description":       "First test incident created via batch API",
			"category":          "software",
			"urgency":           3,
		},
		{
			"short_description": "Batch Test Incident 2",
			"description":       "Second test incident created via batch API",
			"category":          "hardware",
			"urgency":           2,
		},
		{
			"short_description": "Batch Test Incident 3",
			"description":       "Third test incident created via batch API",
			"category":          "network",
			"urgency":           1,
		},
	}

	result, err := batchClient.CreateMultiple("incident", records)
	if err != nil {
		t.Fatalf("Failed to create multiple incidents: %v", err)
	}

	log.Printf("Batch create result: %d successful, %d failed",
		result.SuccessfulRequests, result.FailedRequests)

	// Verify all requests were successful
	if result.FailedRequests > 0 {
		t.Errorf("Expected 0 failed requests, got %d", result.FailedRequests)
		for id, reqErr := range result.GetAllErrors() {
			log.Printf("Failed request %s: %s - %s", id, reqErr.StatusText, reqErr.ErrorDetail)
		}
	}

	if result.SuccessfulRequests != len(records) {
		t.Errorf("Expected %d successful requests, got %d", len(records), result.SuccessfulRequests)
	}

	// Extract created incident sys_ids for cleanup
	var createdSysIDs []string
	for _, reqResult := range result.GetAllSuccessful() {
		record, err := batch.ExtractRecordData(reqResult)
		if err != nil {
			t.Errorf("Failed to extract record data: %v", err)
			continue
		}

		if sysID, ok := record["sys_id"]; ok {
			createdSysIDs = append(createdSysIDs, sysID.(string))
			log.Printf("Created incident: %s (sys_id: %s)", record["number"], sysID)
		}
	}

	// Cleanup: Delete the created incidents
	if len(createdSysIDs) > 0 {
		cleanupResult, err := batchClient.DeleteMultiple("incident", createdSysIDs)
		if err != nil {
			log.Printf("Warning: Failed to cleanup created incidents: %v", err)
		} else {
			log.Printf("Cleanup: Deleted %d incidents", cleanupResult.SuccessfulRequests)
		}
	}
}

func TestBatchIntegration_MixedOperations(t *testing.T) {
	client := setupBatchClient(t)
	batchClient := client.Batch()

	// Step 1: Create an incident first for testing update/delete
	createData := map[string]interface{}{
		"short_description": "Batch Mixed Test Incident",
		"description":       "Incident for testing mixed batch operations",
		"category":          "software",
	}

	createResult, err := batchClient.CreateMultiple("incident", []map[string]interface{}{createData})
	if err != nil {
		t.Fatalf("Failed to create test incident: %v", err)
	}

	if createResult.SuccessfulRequests == 0 {
		t.Fatal("Failed to create test incident for mixed operations test")
	}

	// Extract the sys_id of the created incident
	var testSysID string
	for _, reqResult := range createResult.GetAllSuccessful() {
		record, err := batch.ExtractRecordData(reqResult)
		if err != nil {
			t.Fatalf("Failed to extract created incident data: %v", err)
		}
		testSysID = record["sys_id"].(string)
		break
	}

	log.Printf("Created test incident with sys_id: %s", testSysID)

	// Step 2: Perform mixed operations
	operations := batch.MixedOperations{
		Creates: []batch.CreateOperation{
			{
				ID:        "mixed_create_1",
				TableName: "incident",
				Data: map[string]interface{}{
					"short_description": "Mixed Batch Create",
					"category":          "hardware",
				},
			},
		},
		Updates: []batch.UpdateOperation{
			{
				ID:        "mixed_update_1",
				TableName: "incident",
				SysID:     testSysID,
				Data: map[string]interface{}{
					"state":       "2",     // In Progress
					"assigned_to": "admin", // Assign to admin user
				},
			},
		},
		Gets: []batch.GetOperation{
			{
				ID:        "mixed_get_1",
				TableName: "incident",
				SysID:     testSysID,
			},
		},
	}

	mixedResult, err := batchClient.ExecuteMixed(operations)
	if err != nil {
		t.Fatalf("Failed to execute mixed operations: %v", err)
	}

	log.Printf("Mixed operations result: %d successful, %d failed",
		mixedResult.SuccessfulRequests, mixedResult.FailedRequests)

	// Verify results
	if mixedResult.HasErrors() {
		for id, reqErr := range mixedResult.GetAllErrors() {
			log.Printf("Failed mixed operation %s: %s - %s", id, reqErr.StatusText, reqErr.ErrorDetail)
		}
	}

	// Check that create operation succeeded
	if createReq, exists := mixedResult.GetResult("mixed_create_1"); exists {
		record, err := batch.ExtractRecordData(createReq)
		if err == nil {
			log.Printf("Mixed create succeeded: %s", record["number"])
		}
	}

	// Check that update operation succeeded
	if updateReq, exists := mixedResult.GetResult("mixed_update_1"); exists {
		if updateReq.StatusCode == 200 {
			log.Printf("Mixed update succeeded with status %d", updateReq.StatusCode)
		}
	}

	// Check that get operation succeeded and verify the update
	if getReq, exists := mixedResult.GetResult("mixed_get_1"); exists {
		record, err := batch.ExtractRecordData(getReq)
		if err == nil {
			log.Printf("Mixed get succeeded, state: %s", record["state"])
			if record["state"] != "2" {
				t.Errorf("Expected state '2' after update, got '%s'", record["state"])
			}
		}
	}

	// Cleanup: Delete both incidents
	var cleanupSysIDs []string
	cleanupSysIDs = append(cleanupSysIDs, testSysID)

	// Get sys_id of created incident from mixed operations
	if createReq, exists := mixedResult.GetResult("mixed_create_1"); exists {
		record, err := batch.ExtractRecordData(createReq)
		if err == nil {
			cleanupSysIDs = append(cleanupSysIDs, record["sys_id"].(string))
		}
	}

	if len(cleanupSysIDs) > 0 {
		cleanupResult, err := batchClient.DeleteMultiple("incident", cleanupSysIDs)
		if err != nil {
			log.Printf("Warning: Failed to cleanup incidents: %v", err)
		} else {
			log.Printf("Cleanup: Deleted %d incidents", cleanupResult.SuccessfulRequests)
		}
	}
}

func TestBatchIntegration_UpdateMultiple(t *testing.T) {
	client := setupBatchClient(t)
	batchClient := client.Batch()

	// Step 1: Create test incidents
	records := []map[string]interface{}{
		{
			"short_description": "Batch Update Test 1",
			"urgency":           3,
		},
		{
			"short_description": "Batch Update Test 2",
			"urgency":           3,
		},
	}

	createResult, err := batchClient.CreateMultiple("incident", records)
	if err != nil {
		t.Fatalf("Failed to create test incidents: %v", err)
	}

	if createResult.SuccessfulRequests != len(records) {
		t.Fatalf("Failed to create all test incidents")
	}

	// Extract sys_ids
	var sysIDs []string
	for _, reqResult := range createResult.GetAllSuccessful() {
		record, err := batch.ExtractRecordData(reqResult)
		if err != nil {
			t.Fatalf("Failed to extract created incident data: %v", err)
		}
		sysIDs = append(sysIDs, record["sys_id"].(string))
	}

	log.Printf("Created %d test incidents for update test", len(sysIDs))

	// Step 2: Update the incidents
	updates := make(map[string]map[string]interface{})
	for i, sysID := range sysIDs {
		updates[sysID] = map[string]interface{}{
			"urgency":  1,   // Change urgency to High
			"state":    "2", // Set to In Progress
			"comments": "Updated via batch API test " + string(rune('1'+i)),
		}
	}

	updateResult, err := batchClient.UpdateMultiple("incident", updates)
	if err != nil {
		t.Fatalf("Failed to update multiple incidents: %v", err)
	}

	log.Printf("Batch update result: %d successful, %d failed",
		updateResult.SuccessfulRequests, updateResult.FailedRequests)

	if updateResult.HasErrors() {
		for id, reqErr := range updateResult.GetAllErrors() {
			log.Printf("Failed update %s: %s - %s", id, reqErr.StatusText, reqErr.ErrorDetail)
		}
	}

	// Step 3: Verify updates by retrieving the records
	getResult, err := batchClient.GetMultiple("incident", sysIDs)
	if err != nil {
		t.Fatalf("Failed to retrieve updated incidents: %v", err)
	}

	for _, reqResult := range getResult.GetAllSuccessful() {
		record, err := batch.ExtractRecordData(reqResult)
		if err != nil {
			t.Errorf("Failed to extract retrieved record: %v", err)
			continue
		}

		// Verify updates were applied
		if record["urgency"] != "1" {
			t.Errorf("Expected urgency '1', got '%s'", record["urgency"])
		}
		if record["state"] != "2" {
			t.Errorf("Expected state '2', got '%s'", record["state"])
		}

		log.Printf("Verified update for incident %s: urgency=%s, state=%s",
			record["number"], record["urgency"], record["state"])
	}

	// Cleanup
	cleanupResult, err := batchClient.DeleteMultiple("incident", sysIDs)
	if err != nil {
		log.Printf("Warning: Failed to cleanup incidents: %v", err)
	} else {
		log.Printf("Cleanup: Deleted %d incidents", cleanupResult.SuccessfulRequests)
	}
}

func TestBatchIntegration_ContextTimeout(t *testing.T) {
	client := setupBatchClient(t)
	batchClient := client.Batch()

	// Create a context with very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	records := []map[string]interface{}{
		{"short_description": "Context timeout test"},
	}

	_, err := batchClient.CreateMultipleWithContext(ctx, "incident", records)

	// We expect this to timeout or be cancelled
	if err == nil {
		t.Log("Batch operation completed before timeout - this is possible with very fast responses")
	} else {
		log.Printf("Batch operation failed as expected due to context timeout: %v", err)
	}
}

func TestBatchIntegration_ErrorHandling(t *testing.T) {
	client := setupBatchClient(t)
	batchClient := client.Batch()

	// Create a batch with both valid and invalid operations
	operations := batch.MixedOperations{
		Creates: []batch.CreateOperation{
			{
				ID:        "valid_create",
				TableName: "incident",
				Data: map[string]interface{}{
					"short_description": "Valid incident",
					"category":          "software",
				},
			},
			{
				ID:        "invalid_create",
				TableName: "incident",
				Data: map[string]interface{}{
					"short_description":                 "Invalid incident",
					"invalid_field_that_does_not_exist": "invalid_value",
				},
			},
		},
		Gets: []batch.GetOperation{
			{
				ID:        "invalid_get",
				TableName: "incident",
				SysID:     "non_existent_sys_id_12345",
			},
		},
	}

	result, err := batchClient.ExecuteMixed(operations)
	if err != nil {
		t.Fatalf("Batch execution failed: %v", err)
	}

	log.Printf("Error handling test result: %d successful, %d failed",
		result.SuccessfulRequests, result.FailedRequests)

	// We expect some operations to succeed and some to fail
	if !result.HasErrors() {
		log.Printf("Unexpectedly, no errors occurred in the batch")
	} else {
		log.Printf("As expected, some operations failed:")
		for id, reqErr := range result.GetAllErrors() {
			log.Printf("  Failed operation %s: %d %s - %s",
				id, reqErr.StatusCode, reqErr.StatusText, reqErr.ErrorDetail)
		}
	}

	// Check if the valid create succeeded
	if validResult, exists := result.GetResult("valid_create"); exists {
		record, err := batch.ExtractRecordData(validResult)
		if err == nil {
			log.Printf("Valid create succeeded: %s", record["number"])

			// Cleanup the successfully created incident
			sysID := record["sys_id"].(string)
			cleanupResult, err := batchClient.DeleteMultiple("incident", []string{sysID})
			if err != nil {
				log.Printf("Warning: Failed to cleanup incident: %v", err)
			} else if cleanupResult.SuccessfulRequests > 0 {
				log.Printf("Cleanup: Deleted created incident")
			}
		}
	}
}

func TestBatchIntegration_OrderedExecution(t *testing.T) {
	client := setupBatchClient(t)
	batchClient := client.Batch()

	// Test ordered execution by creating an incident and then updating it
	builder := batchClient.NewBatch().
		WithEnforceOrder(true).
		WithRequestID("ordered_test_batch")

	// First, create an incident
	createData := map[string]interface{}{
		"short_description": "Ordered execution test",
		"state":             "1", // New
	}
	builder.Create("ordered_create", "incident", createData)

	// Note: In a real scenario, we would need the sys_id from the create operation
	// to update it, but ServiceNow batch API doesn't support dependent operations
	// where one request depends on the result of another within the same batch.
	// This test demonstrates the enforce_order parameter usage.

	result, err := builder.Execute()
	if err != nil {
		t.Fatalf("Ordered batch execution failed: %v", err)
	}

	log.Printf("Ordered execution test: %d successful, %d failed",
		result.SuccessfulRequests, result.FailedRequests)

	// Cleanup
	if createResult, exists := result.GetResult("ordered_create"); exists {
		record, err := batch.ExtractRecordData(createResult)
		if err == nil {
			sysID := record["sys_id"].(string)
			cleanupResult, err := batchClient.DeleteMultiple("incident", []string{sysID})
			if err != nil {
				log.Printf("Warning: Failed to cleanup incident: %v", err)
			} else {
				log.Printf("Cleanup: Deleted ordered execution test incident: %v", cleanupResult)
			}
		}
	}
}

func TestBatchIntegration_Performance(t *testing.T) {
	client := setupBatchClient(t)
	batchClient := client.Batch()

	numRecords := 10
	records := make([]map[string]interface{}, numRecords)
	for i := 0; i < numRecords; i++ {
		records[i] = map[string]interface{}{
			"short_description": "Performance test incident " + string(rune('1'+i)),
			"category":          "software",
			"urgency":           3,
		}
	}

	// Measure batch performance
	startTime := time.Now()
	result, err := batchClient.CreateMultiple("incident", records)
	batchDuration := time.Since(startTime)

	if err != nil {
		t.Fatalf("Batch performance test failed: %v", err)
	}

	log.Printf("Batch created %d incidents in %v (%v per record)",
		result.SuccessfulRequests, batchDuration, batchDuration/time.Duration(result.SuccessfulRequests))

	// Extract sys_ids for cleanup
	var createdSysIDs []string
	for _, reqResult := range result.GetAllSuccessful() {
		record, err := batch.ExtractRecordData(reqResult)
		if err == nil {
			createdSysIDs = append(createdSysIDs, record["sys_id"].(string))
		}
	}

	// Measure cleanup performance
	if len(createdSysIDs) > 0 {
		startTime = time.Now()
		cleanupResult, err := batchClient.DeleteMultiple("incident", createdSysIDs)
		cleanupDuration := time.Since(startTime)

		if err != nil {
			log.Printf("Warning: Cleanup failed: %v", err)
		} else {
			log.Printf("Batch deleted %d incidents in %v (%v per record)",
				cleanupResult.SuccessfulRequests, cleanupDuration,
				cleanupDuration/time.Duration(cleanupResult.SuccessfulRequests))
		}
	}
}
