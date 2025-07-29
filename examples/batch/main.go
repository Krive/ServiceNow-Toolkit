package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/Krive/ServiceNow-Toolkit/pkg/servicenow"
	"github.com/Krive/ServiceNow-Toolkit/pkg/servicenow/batch"
)

func main() {
	// Initialize ServiceNow client
	client, err := servicenow.NewClientBasicAuth(
		os.Getenv("SN_INSTANCE_URL"),
		os.Getenv("SN_USERNAME"),
		os.Getenv("SN_PASSWORD"),
	)
	if err != nil {
		log.Fatalf("Failed to create ServiceNow client: %v", err)
	}

	// Run examples
	fmt.Println("=== ServiceNow Batch API Examples ===\n")

	createMultipleExample(client)
	updateMultipleExample(client)
	mixedOperationsExample(client)
	customBatchBuilderExample(client)
	errorHandlingExample(client)
	performanceComparisonExample(client)
	contextTimeoutExample(client)
}

// Example 1: Create multiple records in a single batch
func createMultipleExample(client *servicenow.Client) {
	fmt.Println("1. Create Multiple Records")
	fmt.Println("--------------------------")

	batchClient := client.Batch()

	// Create multiple incidents
	incidents := []map[string]interface{}{
		{
			"short_description": "Batch created incident 1",
			"description":       "First incident created via batch API",
			"category":          "software",
			"urgency":           3,
			"impact":            3,
		},
		{
			"short_description": "Batch created incident 2",
			"description":       "Second incident created via batch API",
			"category":          "hardware",
			"urgency":           2,
			"impact":            2,
		},
		{
			"short_description": "Batch created incident 3",
			"description":       "Third incident created via batch API",
			"category":          "network",
			"urgency":           1,
			"impact":            1,
		},
	}

	result, err := batchClient.CreateMultiple("incident", incidents)
	if err != nil {
		log.Printf("Error creating multiple incidents: %v", err)
		return
	}

	fmt.Printf("Batch create result: %d successful, %d failed\n", 
		result.SuccessfulRequests, result.FailedRequests)

	// Process successful creations
	var createdSysIDs []string
	for id, reqResult := range result.GetAllSuccessful() {
		record, err := batch.ExtractRecordData(reqResult)
		if err != nil {
			log.Printf("Error extracting data for %s: %v", id, err)
			continue
		}
		
		createdSysIDs = append(createdSysIDs, record["sys_id"].(string))
		fmt.Printf("  Created: %s (sys_id: %s)\n", record["number"], record["sys_id"])
	}

	// Process any errors
	if result.HasErrors() {
		fmt.Println("Errors:")
		for id, reqErr := range result.GetAllErrors() {
			fmt.Printf("  %s: %s - %s\n", id, reqErr.StatusText, reqErr.ErrorDetail)
		}
	}

	// Cleanup: Delete created incidents
	if len(createdSysIDs) > 0 {
		cleanupResult, err := batchClient.DeleteMultiple("incident", createdSysIDs)
		if err != nil {
			log.Printf("Warning: Failed to cleanup: %v", err)
		} else {
			fmt.Printf("Cleanup: Deleted %d incidents\n", cleanupResult.SuccessfulRequests)
		}
	}
	fmt.Println()
}

// Example 2: Update multiple records
func updateMultipleExample(client *servicenow.Client) {
	fmt.Println("2. Update Multiple Records")
	fmt.Println("--------------------------")

	batchClient := client.Batch()

	// First, create some test incidents to update
	testIncidents := []map[string]interface{}{
		{
			"short_description": "Update test incident 1",
			"state":             "1", // New
			"urgency":           3,
		},
		{
			"short_description": "Update test incident 2",
			"state":             "1", // New
			"urgency":           3,
		},
	}

	createResult, err := batchClient.CreateMultiple("incident", testIncidents)
	if err != nil {
		log.Printf("Error creating test incidents: %v", err)
		return
	}

	// Extract sys_ids for updating
	var sysIDs []string
	for _, reqResult := range createResult.GetAllSuccessful() {
		record, err := batch.ExtractRecordData(reqResult)
		if err != nil {
			continue
		}
		sysIDs = append(sysIDs, record["sys_id"].(string))
	}

	if len(sysIDs) == 0 {
		fmt.Println("No incidents created for update test")
		return
	}

	fmt.Printf("Created %d test incidents for updating\n", len(sysIDs))

	// Prepare updates
	updates := make(map[string]map[string]interface{})
	for i, sysID := range sysIDs {
		updates[sysID] = map[string]interface{}{
			"state":    "2", // In Progress
			"urgency":  1,   // High
			"comments": fmt.Sprintf("Updated via batch API - Test %d", i+1),
		}
	}

	// Execute batch update
	updateResult, err := batchClient.UpdateMultiple("incident", updates)
	if err != nil {
		log.Printf("Error updating incidents: %v", err)
		return
	}

	fmt.Printf("Batch update result: %d successful, %d failed\n",
		updateResult.SuccessfulRequests, updateResult.FailedRequests)

	// Verify updates by retrieving the records
	getResult, err := batchClient.GetMultiple("incident", sysIDs)
	if err != nil {
		log.Printf("Error retrieving updated incidents: %v", err)
	} else {
		fmt.Println("Verification:")
		for _, reqResult := range getResult.GetAllSuccessful() {
			record, err := batch.ExtractRecordData(reqResult)
			if err != nil {
				continue
			}
			fmt.Printf("  %s: state=%s, urgency=%s\n", 
				record["number"], record["state"], record["urgency"])
		}
	}

	// Cleanup
	cleanupResult, err := batchClient.DeleteMultiple("incident", sysIDs)
	if err != nil {
		log.Printf("Warning: Failed to cleanup: %v", err)
	} else {
		fmt.Printf("Cleanup: Deleted %d incidents\n", cleanupResult.SuccessfulRequests)
	}
	fmt.Println()
}

// Example 3: Mixed operations in a single batch
func mixedOperationsExample(client *servicenow.Client) {
	fmt.Println("3. Mixed Operations")
	fmt.Println("-------------------")

	batchClient := client.Batch()

	// First create a test incident to update later
	testData := map[string]interface{}{
		"short_description": "Mixed operations test incident",
		"category":          "software",
	}

	createResult, err := batchClient.CreateMultiple("incident", []map[string]interface{}{testData})
	if err != nil {
		log.Printf("Error creating test incident: %v", err)
		return
	}

	var testSysID string
	for _, reqResult := range createResult.GetAllSuccessful() {
		record, err := batch.ExtractRecordData(reqResult)
		if err != nil {
			continue
		}
		testSysID = record["sys_id"].(string)
		break
	}

	if testSysID == "" {
		log.Printf("Failed to create test incident for mixed operations")
		return
	}

	fmt.Printf("Created test incident: %s\n", testSysID)

	// Define mixed operations
	operations := batch.MixedOperations{
		Creates: []batch.CreateOperation{
			{
				ID:        "mixed_create_1",
				TableName: "incident",
				Data: map[string]interface{}{
					"short_description": "Mixed batch create operation",
					"category":          "hardware",
					"urgency":           2,
				},
			},
			{
				ID:        "mixed_create_2",
				TableName: "incident",
				Data: map[string]interface{}{
					"short_description": "Another mixed batch create",
					"category":          "network",
					"urgency":           1,
				},
			},
		},
		Updates: []batch.UpdateOperation{
			{
				ID:        "mixed_update_1",
				TableName: "incident",
				SysID:     testSysID,
				Data: map[string]interface{}{
					"state":       "2", // In Progress
					"assigned_to": "admin",
					"comments":    "Updated via mixed batch operation",
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

	// Execute mixed operations
	mixedResult, err := batchClient.ExecuteMixed(operations)
	if err != nil {
		log.Printf("Error executing mixed operations: %v", err)
		return
	}

	fmt.Printf("Mixed operations result: %d successful, %d failed\n",
		mixedResult.SuccessfulRequests, mixedResult.FailedRequests)

	// Process results
	var allSysIDs []string
	allSysIDs = append(allSysIDs, testSysID) // Original test incident

	fmt.Println("Results:")
	
	// Check create operations
	for _, createOp := range operations.Creates {
		if result, exists := mixedResult.GetResult(createOp.ID); exists {
			record, err := batch.ExtractRecordData(result)
			if err == nil {
				allSysIDs = append(allSysIDs, record["sys_id"].(string))
				fmt.Printf("  Create %s: %s (sys_id: %s)\n", 
					createOp.ID, record["number"], record["sys_id"])
			}
		}
	}

	// Check update operations
	for _, updateOp := range operations.Updates {
		if result, exists := mixedResult.GetResult(updateOp.ID); exists {
			fmt.Printf("  Update %s: Status %d - %s\n", 
				updateOp.ID, result.StatusCode, result.StatusText)
		}
	}

	// Check get operations
	for _, getOp := range operations.Gets {
		if result, exists := mixedResult.GetResult(getOp.ID); exists {
			record, err := batch.ExtractRecordData(result)
			if err == nil {
				fmt.Printf("  Get %s: %s - State: %s\n", 
					getOp.ID, record["number"], record["state"])
			}
		}
	}

	// Show any errors
	if mixedResult.HasErrors() {
		fmt.Println("Errors:")
		for id, reqErr := range mixedResult.GetAllErrors() {
			fmt.Printf("  %s: %s - %s\n", id, reqErr.StatusText, reqErr.ErrorDetail)
		}
	}

	// Cleanup all created incidents
	if len(allSysIDs) > 0 {
		cleanupResult, err := batchClient.DeleteMultiple("incident", allSysIDs)
		if err != nil {
			log.Printf("Warning: Failed to cleanup: %v", err)
		} else {
			fmt.Printf("Cleanup: Deleted %d incidents\n", cleanupResult.SuccessfulRequests)
		}
	}
	fmt.Println()
}

// Example 4: Custom batch builder with detailed control
func customBatchBuilderExample(client *servicenow.Client) {
	fmt.Println("4. Custom Batch Builder")
	fmt.Println("-----------------------")

	batchClient := client.Batch()

	// Build a custom batch with detailed control
	builder := batchClient.NewBatch().
		WithRequestID("custom_batch_example_001").
		WithEnforceOrder(true) // Execute requests in order

	// Add various operations
	builder.Create("create_high_priority", "incident", map[string]interface{}{
		"short_description": "High priority batch incident",
		"urgency":           1,
		"impact":            1,
		"category":          "software",
	})

	builder.Create("create_medium_priority", "incident", map[string]interface{}{
		"short_description": "Medium priority batch incident",
		"urgency":           2,
		"impact":            2,
		"category":          "hardware",
	})

	// Add a custom request (example: get specific user)
	customRequest := batch.RestRequest{
		ID:     "get_admin_user",
		URL:    "/api/now/table/sys_user?sysparm_query=user_name=admin&sysparm_limit=1",
		Method: batch.MethodGET,
		Headers: []batch.Header{
			{Name: "Accept", Value: "application/json"},
		},
		ExcludeResponseHeaders: true,
	}
	builder.AddCustomRequest(customRequest)

	// Execute the custom batch
	result, err := builder.Execute()
	if err != nil {
		log.Printf("Error executing custom batch: %v", err)
		return
	}

	fmt.Printf("Custom batch result: %d successful, %d failed\n",
		result.SuccessfulRequests, result.FailedRequests)
	fmt.Printf("Batch ID: %s\n", result.BatchRequestID)

	// Process results
	var createdSysIDs []string
	for id, reqResult := range result.GetAllSuccessful() {
		fmt.Printf("  Request %s: Status %d, Execution time: %v\n", 
			id, reqResult.StatusCode, reqResult.ExecutionTime)

		if reqResult.Data != nil {
			if id == "get_admin_user" {
				fmt.Printf("    Admin user lookup successful\n")
			} else {
				// Extract incident data
				record, err := batch.ExtractRecordData(reqResult)
				if err == nil {
					createdSysIDs = append(createdSysIDs, record["sys_id"].(string))
					fmt.Printf("    Created incident: %s\n", record["number"])
				}
			}
		}
	}

	// Cleanup
	if len(createdSysIDs) > 0 {
		cleanupResult, err := batchClient.DeleteMultiple("incident", createdSysIDs)
		if err != nil {
			log.Printf("Warning: Failed to cleanup: %v", err)
		} else {
			fmt.Printf("Cleanup: Deleted %d incidents\n", cleanupResult.SuccessfulRequests)
		}
	}
	fmt.Println()
}

// Example 5: Error handling and partial failures
func errorHandlingExample(client *servicenow.Client) {
	fmt.Println("5. Error Handling")
	fmt.Println("------------------")

	batchClient := client.Batch()

	// Create a batch with intentionally mixed valid/invalid operations
	operations := batch.MixedOperations{
		Creates: []batch.CreateOperation{
			{
				ID:        "valid_create",
				TableName: "incident",
				Data: map[string]interface{}{
					"short_description": "Valid incident creation",
					"category":          "software",
				},
			},
			{
				ID:        "invalid_create",
				TableName: "incident",
				Data: map[string]interface{}{
					"short_description":    "Invalid incident",
					"invalid_field_name":   "this_field_does_not_exist",
					"another_invalid_field": 12345,
				},
			},
		},
		Gets: []batch.GetOperation{
			{
				ID:        "invalid_get",
				TableName: "incident",
				SysID:     "non_existent_sys_id_12345", // This sys_id doesn't exist
			},
		},
	}

	result, err := batchClient.ExecuteMixed(operations)
	if err != nil {
		log.Printf("Batch execution failed completely: %v", err)
		return
	}

	fmt.Printf("Error handling result: %d successful, %d failed\n",
		result.SuccessfulRequests, result.FailedRequests)

	// Process successful operations
	if result.SuccessfulRequests > 0 {
		fmt.Println("Successful operations:")
		var createdSysIDs []string
		for id, reqResult := range result.GetAllSuccessful() {
			fmt.Printf("  %s: %d %s (execution time: %v)\n", 
				id, reqResult.StatusCode, reqResult.StatusText, reqResult.ExecutionTime)
			
			// Extract created incident if applicable
			if reqResult.Data != nil {
				record, err := batch.ExtractRecordData(reqResult)
				if err == nil && record["sys_id"] != nil {
					createdSysIDs = append(createdSysIDs, record["sys_id"].(string))
				}
			}
		}

		// Cleanup successful creations
		if len(createdSysIDs) > 0 {
			cleanupResult, err := batchClient.DeleteMultiple("incident", createdSysIDs)
			if err != nil {
				log.Printf("Warning: Failed to cleanup: %v", err)
			} else {
				fmt.Printf("Cleanup: Deleted %d incidents\n", cleanupResult.SuccessfulRequests)
			}
		}
	}

	// Process failed operations
	if result.HasErrors() {
		fmt.Println("Failed operations:")
		for id, reqErr := range result.GetAllErrors() {
			fmt.Printf("  %s: %d %s\n", id, reqErr.StatusCode, reqErr.StatusText)
			if reqErr.ErrorDetail != "" {
				fmt.Printf("    Detail: %s\n", reqErr.ErrorDetail)
			}
		}
	}

	fmt.Println("Key takeaway: Batch operations continue processing even when some requests fail")
	fmt.Println()
}

// Example 6: Performance comparison (batch vs individual requests)
func performanceComparisonExample(client *servicenow.Client) {
	fmt.Println("6. Performance Comparison")
	fmt.Println("-------------------------")

	batchClient := client.Batch()
	tableClient := client.Table("incident")

	numRecords := 5
	testData := make([]map[string]interface{}, numRecords)
	for i := 0; i < numRecords; i++ {
		testData[i] = map[string]interface{}{
			"short_description": fmt.Sprintf("Performance test incident %d", i+1),
			"category":          "software",
			"urgency":           3,
		}
	}

	// Test 1: Batch creation
	fmt.Printf("Creating %d incidents using batch API...\n", numRecords)
	batchStart := time.Now()
	batchResult, err := batchClient.CreateMultiple("incident", testData)
	batchDuration := time.Since(batchStart)

	if err != nil {
		log.Printf("Batch creation failed: %v", err)
		return
	}

	var batchCreatedSysIDs []string
	for _, reqResult := range batchResult.GetAllSuccessful() {
		record, err := batch.ExtractRecordData(reqResult)
		if err == nil {
			batchCreatedSysIDs = append(batchCreatedSysIDs, record["sys_id"].(string))
		}
	}

	fmt.Printf("Batch API: Created %d incidents in %v (%v per record)\n",
		batchResult.SuccessfulRequests, batchDuration, 
		batchDuration/time.Duration(batchResult.SuccessfulRequests))

	// Test 2: Individual requests
	fmt.Printf("Creating %d incidents using individual requests...\n", numRecords)
	individualStart := time.Now()
	var individualCreatedSysIDs []string
	successCount := 0

	for i, data := range testData {
		record, err := tableClient.Create(data)
		if err != nil {
			log.Printf("Failed to create individual incident %d: %v", i+1, err)
			continue
		}
		individualCreatedSysIDs = append(individualCreatedSysIDs, record["sys_id"].(string))
		successCount++
	}

	individualDuration := time.Since(individualStart)
	fmt.Printf("Individual API: Created %d incidents in %v (%v per record)\n",
		successCount, individualDuration, individualDuration/time.Duration(successCount))

	// Compare performance
	if batchDuration < individualDuration {
		improvement := float64(individualDuration-batchDuration) / float64(individualDuration) * 100
		fmt.Printf("Batch API was %.1f%% faster than individual requests\n", improvement)
	} else {
		fmt.Printf("Individual requests were faster (possibly due to small dataset or network conditions)\n")
	}

	// Cleanup both sets of incidents
	allSysIDs := append(batchCreatedSysIDs, individualCreatedSysIDs...)
	if len(allSysIDs) > 0 {
		cleanupStart := time.Now()
		cleanupResult, err := batchClient.DeleteMultiple("incident", allSysIDs)
		cleanupDuration := time.Since(cleanupStart)

		if err != nil {
			log.Printf("Warning: Failed to cleanup: %v", err)
		} else {
			fmt.Printf("Batch cleanup: Deleted %d incidents in %v\n", 
				cleanupResult.SuccessfulRequests, cleanupDuration)
		}
	}
	fmt.Println()
}

// Example 7: Context timeout handling
func contextTimeoutExample(client *servicenow.Client) {
	fmt.Println("7. Context Timeout")
	fmt.Println("------------------")

	batchClient := client.Batch()

	// Create a batch operation with timeout
	testData := []map[string]interface{}{
		{
			"short_description": "Context timeout test incident",
			"category":          "software",
		},
	}

	// Test with reasonable timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Println("Executing batch with 30-second timeout...")
	result, err := batchClient.CreateMultipleWithContext(ctx, "incident", testData)
	
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			fmt.Println("Operation timed out after 30 seconds")
		} else {
			log.Printf("Operation failed: %v", err)
		}
		return
	}

	fmt.Printf("Operation completed successfully: %d successful, %d failed\n",
		result.SuccessfulRequests, result.FailedRequests)

	// Cleanup
	var createdSysIDs []string
	for _, reqResult := range result.GetAllSuccessful() {
		record, err := batch.ExtractRecordData(reqResult)
		if err == nil {
			createdSysIDs = append(createdSysIDs, record["sys_id"].(string))
		}
	}

	if len(createdSysIDs) > 0 {
		cleanupResult, err := batchClient.DeleteMultiple("incident", createdSysIDs)
		if err != nil {
			log.Printf("Warning: Failed to cleanup: %v", err)
		} else {
			fmt.Printf("Cleanup: Deleted %d incidents\n", cleanupResult.SuccessfulRequests)
		}
	}

	// Demonstrate timeout with very short timeout
	fmt.Println("Testing with very short timeout (1ms)...")
	shortCtx, shortCancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer shortCancel()

	_, err = batchClient.CreateMultipleWithContext(shortCtx, "incident", testData)
	if err != nil {
		if shortCtx.Err() == context.DeadlineExceeded {
			fmt.Println("As expected, operation timed out with 1ms timeout")
		} else {
			fmt.Printf("Operation failed with short timeout: %v\n", err)
		}
	} else {
		fmt.Println("Surprisingly, operation completed even with 1ms timeout")
	}

	fmt.Println()
}

// Utility function to demonstrate batch API usage in a real application
func realWorldUsageExample() {
	// This function shows how you might use the batch API in a real application
	// It doesn't execute, but shows the patterns

	client, _ := servicenow.NewClientBasicAuth("https://dev.service-now.com", "user", "pass")
	batchClient := client.Batch()

	// Scenario: Bulk incident migration from another system
	bulkIncidentMigration := func(incidents []map[string]interface{}) error {
		// Process in batches of 50 (ServiceNow recommendation)
		batchSize := 50
		for i := 0; i < len(incidents); i += batchSize {
			end := i + batchSize
			if end > len(incidents) {
				end = len(incidents)
			}
			
			batch := incidents[i:end]
			result, err := batchClient.CreateMultiple("incident", batch)
			if err != nil {
				return fmt.Errorf("batch %d failed: %w", i/batchSize+1, err)
			}
			
			// Log results
			log.Printf("Batch %d: %d successful, %d failed", 
				i/batchSize+1, result.SuccessfulRequests, result.FailedRequests)
			
			// Handle partial failures
			if result.HasErrors() {
				for id, reqErr := range result.GetAllErrors() {
					log.Printf("Failed to create incident %s: %s", id, reqErr.ErrorDetail)
				}
			}
		}
		return nil
	}

	// Scenario: Bulk status update based on business rules
	bulkStatusUpdate := func(incidentUpdates map[string]map[string]interface{}) error {
		result, err := batchClient.UpdateMultiple("incident", incidentUpdates)
		if err != nil {
			return fmt.Errorf("bulk update failed: %w", err)
		}
		
		log.Printf("Bulk update: %d successful, %d failed", 
			result.SuccessfulRequests, result.FailedRequests)
		
		return nil
	}

	// Scenario: Audit and cleanup old records
	auditAndCleanup := func(oldIncidentIDs []string) error {
		// First, get the records to audit
		getResult, err := batchClient.GetMultiple("incident", oldIncidentIDs)
		if err != nil {
			return fmt.Errorf("audit retrieval failed: %w", err)
		}
		
		// Process audit results
		var idsToDelete []string
		for _, reqResult := range getResult.GetAllSuccessful() {
			record, err := batch.ExtractRecordData(reqResult)
			if err != nil {
				continue
			}
			
			// Apply business logic to determine if record should be deleted
			if shouldDeleteRecord(record) {
				idsToDelete = append(idsToDelete, record["sys_id"].(string))
			}
		}
		
		// Delete the records that meet criteria
		if len(idsToDelete) > 0 {
			deleteResult, err := batchClient.DeleteMultiple("incident", idsToDelete)
			if err != nil {
				return fmt.Errorf("bulk delete failed: %w", err)
			}
			
			log.Printf("Cleanup: deleted %d records", deleteResult.SuccessfulRequests)
		}
		
		return nil
	}

	// Use the functions (this is just for demonstration)
	_, _, _ = bulkIncidentMigration, bulkStatusUpdate, auditAndCleanup
}

// Helper function for the real-world example
func shouldDeleteRecord(record map[string]interface{}) bool {
	// Example business logic: delete incidents that are resolved and older than 1 year
	state, hasState := record["state"]
	if !hasState {
		return false
	}
	
	// Check if incident is in resolved state (6 = Resolved)
	if state != "6" {
		return false
	}
	
	// In a real implementation, you would check the sys_updated_on date
	// For this example, we'll return false to be safe
	return false
}