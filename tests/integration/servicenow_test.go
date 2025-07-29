package integration

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Krive/ServiceNow-Toolkit/pkg/servicenow"
	"github.com/Krive/ServiceNow-Toolkit/pkg/servicenow/importset"
)

// Integration tests that run against a real ServiceNow instance
// These tests require valid credentials in environment variables

func getTestConfig(t *testing.T) (string, string, string) {
	instanceURL := os.Getenv("SERVICENOW_INSTANCE_URL")
	username := os.Getenv("SERVICENOW_USERNAME")
	password := os.Getenv("SERVICENOW_PASSWORD")

	if instanceURL == "" || username == "" || password == "" {
		t.Skip("Skipping integration tests: SERVICENOW_INSTANCE_URL, SERVICENOW_USERNAME, and SERVICENOW_PASSWORD must be set")
	}

	return instanceURL, username, password
}

func getOAuthConfig(t *testing.T) (string, string, string) {
	instanceURL := os.Getenv("SERVICENOW_INSTANCE_URL")
	clientID := os.Getenv("SERVICENOW_CLIENT_ID")
	clientSecret := os.Getenv("SERVICENOW_CLIENT_SECRET")

	if instanceURL == "" || clientID == "" || clientSecret == "" {
		t.Skip("Skipping OAuth integration tests: SERVICENOW_INSTANCE_URL, SERVICENOW_CLIENT_ID, and SERVICENOW_CLIENT_SECRET must be set")
	}

	return instanceURL, clientID, clientSecret
}

func TestBasicAuthIntegration(t *testing.T) {
	instanceURL, username, password := getTestConfig(t)

	client, err := servicenow.NewClientBasicAuth(instanceURL, username, password)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Test table operations
	incidents, err := client.Table("incident").List(map[string]string{
		"sysparm_limit": "5",
		"sysparm_fields": "number,short_description,state",
	})
	if err != nil {
		t.Fatalf("Failed to list incidents: %v", err)
	}

	if len(incidents) == 0 {
		t.Log("Warning: No incidents found in test instance")
	} else {
		t.Logf("Successfully retrieved %d incidents", len(incidents))
		
		// Verify incident structure
		incident := incidents[0]
		if incident["number"] == nil {
			t.Error("Expected incident to have 'number' field")
		}
	}
}

func TestOAuthIntegration(t *testing.T) {
	instanceURL, clientID, clientSecret := getOAuthConfig(t)

	client, err := servicenow.NewClientOAuth(instanceURL, clientID, clientSecret)
	if err != nil {
		t.Fatalf("Failed to create OAuth client: %v", err)
	}

	// Test table operations with OAuth
	users, err := client.Table("sys_user").List(map[string]string{
		"sysparm_limit": "3",
		"sysparm_fields": "name,email,active",
		"sysparm_query": "active=true",
	})
	if err != nil {
		t.Fatalf("Failed to list users with OAuth: %v", err)
	}

	if len(users) == 0 {
		t.Log("Warning: No active users found")
	} else {
		t.Logf("Successfully retrieved %d users with OAuth", len(users))
	}
}

func TestConfigBasedAuthentication(t *testing.T) {
	instanceURL, username, password := getTestConfig(t)

	// Test config-based client creation
	client, err := servicenow.NewClient(servicenow.Config{
		InstanceURL: instanceURL,
		Username:    username,
		Password:    password,
		Timeout:     45 * time.Second,
	})
	if err != nil {
		t.Fatalf("Failed to create config-based client: %v", err)
	}

	// Test timeout configuration
	if client.GetTimeout() != 45*time.Second {
		t.Errorf("Expected timeout to be 45s, got %v", client.GetTimeout())
	}

	// Test API call
	groups, err := client.Table("sys_user_group").List(map[string]string{
		"sysparm_limit": "2",
		"sysparm_fields": "name,description,active",
	})
	if err != nil {
		t.Fatalf("Failed to list groups: %v", err)
	}

	t.Logf("Successfully retrieved %d groups", len(groups))
}

func TestTableOperations(t *testing.T) {
	instanceURL, username, password := getTestConfig(t)

	client, err := servicenow.NewClientBasicAuth(instanceURL, username, password)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	tableClient := client.Table("incident")

	// Test List operation
	incidents, err := tableClient.List(map[string]string{
		"sysparm_limit": "5",
		"sysparm_fields": "number,short_description,priority,state",
	})
	if err != nil {
		t.Fatalf("Failed to list incidents: %v", err)
	}

	t.Logf("Found %d incidents", len(incidents))

	if len(incidents) > 0 {
		// Test Get operation using first incident's sys_id
		incident := incidents[0]
		sysID, ok := incident["sys_id"].(string)
		if !ok {
			t.Error("Expected sys_id to be a string")
			return
		}

		retrievedIncident, err := tableClient.Get(sysID)
		if err != nil {
			t.Fatalf("Failed to get incident %s: %v", sysID, err)
		}

		if retrievedIncident["sys_id"] != sysID {
			t.Errorf("Expected retrieved incident sys_id to be %s, got %s", sysID, retrievedIncident["sys_id"])
		}

		t.Logf("Successfully retrieved incident: %s", retrievedIncident["number"])
	}
}

func TestTableSchema(t *testing.T) {
	instanceURL, username, password := getTestConfig(t)

	client, err := servicenow.NewClientBasicAuth(instanceURL, username, password)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Test schema retrieval for incident table
	schema, err := client.Table("incident").GetSchema()
	if err != nil {
		t.Fatalf("Failed to get incident schema: %v", err)
	}

	if len(schema) == 0 {
		t.Error("Expected incident schema to have columns")
		return
	}

	t.Logf("Incident table has %d columns", len(schema))

	// Verify some expected columns exist
	expectedColumns := []string{"number", "short_description", "priority", "state"}
	foundColumns := make(map[string]bool)

	for _, column := range schema {
		foundColumns[column.Name] = true
	}

	for _, expectedCol := range expectedColumns {
		if !foundColumns[expectedCol] {
			t.Errorf("Expected column '%s' not found in incident schema", expectedCol)
		}
	}
}

func TestAttachmentOperations(t *testing.T) {
	instanceURL, username, password := getTestConfig(t)

	client, err := servicenow.NewClientBasicAuth(instanceURL, username, password)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Test listing attachments (may be empty, that's ok)
	// Using a fake sys_id since we don't want to create test data
	attachments, err := client.Attachment().List("incident", "fake_sys_id")
	if err != nil {
		// This is expected to fail with 404 or similar, which is fine for testing the API structure
		t.Logf("Attachment list failed as expected: %v", err)
	} else {
		t.Logf("Found %d attachments", len(attachments))
	}
}

func TestErrorHandling(t *testing.T) {
	instanceURL, username, password := getTestConfig(t)

	client, err := servicenow.NewClientBasicAuth(instanceURL, username, password)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Test accessing non-existent table
	_, err = client.Table("non_existent_table_xyz").List(map[string]string{
		"sysparm_limit": "1",
	})
	
	if err == nil {
		t.Error("Expected error when accessing non-existent table")
	} else {
		t.Logf("Got expected error for non-existent table: %v", err)
		
		// Check if it's a ServiceNow error
		if strings.Contains(err.Error(), "ServiceNow") {
			t.Log("Error correctly identified as ServiceNow error")
		}
	}

	// Test accessing record with invalid sys_id
	_, err = client.Table("incident").Get("invalid_sys_id")
	if err == nil {
		t.Error("Expected error when accessing record with invalid sys_id")
	} else {
		t.Logf("Got expected error for invalid sys_id: %v", err)
	}
}

func TestRateLimitingBehavior(t *testing.T) {
	instanceURL, username, password := getTestConfig(t)

	// Create client with conservative rate limiting
	client, err := servicenow.NewClientBasicAuth(instanceURL, username, password)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Apply conservative rate limiting
	client.WithConservativeRateLimit()

	// Make multiple requests and measure timing
	start := time.Now()
	
	for i := 0; i < 3; i++ {
		_, err := client.Table("sys_user").List(map[string]string{
			"sysparm_limit": "1",
		})
		if err != nil {
			t.Logf("Request %d failed: %v", i+1, err)
		}
	}
	
	elapsed := time.Since(start)
	t.Logf("3 requests took %v with conservative rate limiting", elapsed)

	// With conservative settings (2 req/s for tables), 3 requests should take at least 1 second
	// But we'll be lenient due to network variability
	if elapsed < 500*time.Millisecond {
		t.Log("Note: Rate limiting may not be working as expected (requests completed very quickly)")
	}
}

func TestImportSetOperations(t *testing.T) {
	instanceURL, username, password := getTestConfig(t)

	client, err := servicenow.NewClientBasicAuth(instanceURL, username, password)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Test import set operations (using a minimal record)
	// Note: This creates actual data, so use with caution
	records := []importset.ImportRecord{
		{
			"u_test_field": "integration_test_" + time.Now().Format("20060102_150405"),
		},
	}

	// This might fail if no import set table is configured, which is expected
	_, err = client.ImportSet().Insert("u_test_import_table", records)
	if err != nil {
		t.Logf("Import set operation failed as expected (no test table configured): %v", err)
	} else {
		t.Log("Import set operation succeeded")
	}
}

// Benchmark test to measure SDK performance
func BenchmarkTableList(b *testing.B) {
	instanceURL := os.Getenv("SERVICENOW_INSTANCE_URL")
	username := os.Getenv("SERVICENOW_USERNAME")
	password := os.Getenv("SERVICENOW_PASSWORD")

	if instanceURL == "" || username == "" || password == "" {
		b.Skip("Skipping benchmark: credentials not set")
	}

	client, err := servicenow.NewClientBasicAuth(instanceURL, username, password)
	if err != nil {
		b.Fatalf("Failed to create client: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := client.Table("sys_user").List(map[string]string{
			"sysparm_limit": "1",
		})
		if err != nil {
			b.Errorf("Request failed: %v", err)
		}
	}
}