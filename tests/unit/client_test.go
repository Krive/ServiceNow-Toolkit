package unit

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Krive/ServiceNow-Toolkit/pkg/servicenow/core"
)

func TestNewClientBasicAuth(t *testing.T) {
	client, err := core.NewClientBasicAuth("https://example.service-now.com", "testuser", "testpass")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if client == nil {
		t.Fatal("Expected client to be created")
	}

	if client.InstanceURL != "https://example.service-now.com" {
		t.Errorf("Expected InstanceURL to be 'https://example.service-now.com', got %s", client.InstanceURL)
	}

	if client.BaseURL != "https://example.service-now.com/api/now" {
		t.Errorf("Expected BaseURL to be 'https://example.service-now.com/api/now', got %s", client.BaseURL)
	}
}

func TestNewClientAPIKey(t *testing.T) {
	client, err := core.NewClientAPIKey("https://example.service-now.com", "test-api-key")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if client == nil {
		t.Fatal("Expected client to be created")
	}

	// Test that rate limiter is initialized
	rateLimiter := client.GetRateLimiter()
	if rateLimiter == nil {
		t.Error("Expected rate limiter to be initialized")
	}
}

func TestClientTimeout(t *testing.T) {
	client, err := core.NewClientBasicAuth("https://example.service-now.com", "user", "pass")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Test default timeout
	defaultTimeout := client.GetTimeout()
	if defaultTimeout.Seconds() != 30 {
		t.Errorf("Expected default timeout to be 30 seconds, got %v", defaultTimeout)
	}

	// Test setting custom timeout
	newTimeout := 60 * time.Second
	client.SetTimeout(newTimeout)
	
	updatedTimeout := client.GetTimeout()
	if updatedTimeout != newTimeout {
		t.Errorf("Expected timeout to be %v, got %v", newTimeout, updatedTimeout)
	}
}

func TestClientHandleResponse(t *testing.T) {
	client, err := core.NewClientBasicAuth("https://example.service-now.com", "user", "pass")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Test successful JSON response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"result": []map[string]interface{}{
				{"sys_id": "123", "number": "INC0000001"},
				{"sys_id": "456", "number": "INC0000002"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Make request
	resp, err := client.Client.R().Get(server.URL)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	var result map[string]interface{}
	err = client.HandleResponse(resp, nil, &result, core.FormatJSON)
	if err != nil {
		t.Fatalf("HandleResponse failed: %v", err)
	}

	// Verify response structure
	if result["result"] == nil {
		t.Error("Expected 'result' field in response")
	}

	resultArray, ok := result["result"].([]interface{})
	if !ok {
		t.Error("Expected 'result' to be an array")
	}

	if len(resultArray) != 2 {
		t.Errorf("Expected 2 results, got %d", len(resultArray))
	}
}

func TestClientHandleErrorResponse(t *testing.T) {
	client, err := core.NewClientBasicAuth("https://example.service-now.com", "user", "pass")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Test error response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		errorResponse := map[string]interface{}{
			"error": map[string]interface{}{
				"message": "Invalid credentials",
				"detail":  "The provided credentials are not valid",
			},
		}
		json.NewEncoder(w).Encode(errorResponse)
	}))
	defer server.Close()

	// Make request
	resp, err := client.Client.R().Get(server.URL)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	var result map[string]interface{}
	err = client.HandleResponse(resp, nil, &result, core.FormatJSON)
	
	// Should return ServiceNow error
	if err == nil {
		t.Error("Expected error for 401 response")
	}

	// Check if it's a ServiceNow error
	snErr, ok := core.IsServiceNowError(err)
	if !ok {
		t.Errorf("Expected ServiceNowError, got %T", err)
	}

	if snErr.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status code 401, got %d", snErr.StatusCode)
	}

	if snErr.Type != core.ErrorTypeAuthentication {
		t.Errorf("Expected authentication error type, got %s", snErr.Type)
	}
}

func TestClientRawRequest(t *testing.T) {
	// Mock ServiceNow API server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method and path
		if r.Method != "GET" {
			t.Errorf("Expected GET method, got %s", r.Method)
		}

		if r.URL.Path != "/table/incident" {
			t.Errorf("Expected /table/incident path, got %s", r.URL.Path)
		}

		// Verify query parameters
		if r.URL.Query().Get("sysparm_limit") != "5" {
			t.Errorf("Expected sysparm_limit=5, got %s", r.URL.Query().Get("sysparm_limit"))
		}

		// Verify headers
		if r.Header.Get("Accept") != "application/json" {
			t.Errorf("Expected Accept: application/json, got %s", r.Header.Get("Accept"))
		}

		// Return mock response
		response := core.Response{
			Result: []map[string]interface{}{
				{"sys_id": "123", "number": "INC0000001"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create client with server URL
	client, err := core.NewClientBasicAuth(server.URL, "user", "pass")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Make raw request
	var result core.Response
	params := map[string]string{"sysparm_limit": "5"}
	err = client.RawRequest("GET", "/table/incident", nil, params, &result)
	
	if err != nil {
		t.Fatalf("RawRequest failed: %v", err)
	}

	// Verify result
	resultArray, ok := result.Result.([]interface{})
	if !ok {
		t.Error("Expected result to be an array")
	}

	if len(resultArray) != 1 {
		t.Errorf("Expected 1 result, got %d", len(resultArray))
	}
}

func TestClientRawRootRequest(t *testing.T) {
	// Mock server for root requests (non-API endpoints)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/xmlstats.do" {
			t.Errorf("Expected /xmlstats.do path, got %s", r.URL.Path)
		}

		if r.Header.Get("Accept") != "application/xml" {
			t.Errorf("Expected Accept: application/xml, got %s", r.Header.Get("Accept"))
		}

		// Return mock XML response
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(`<stats><active_users>42</active_users></stats>`))
	}))
	defer server.Close()

	client, err := core.NewClientBasicAuth(server.URL, "user", "pass")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Make root request with XML format
	var result map[string]interface{}
	err = client.RawRootRequest("GET", "/xmlstats.do", nil, nil, &result, core.FormatXML)
	
	if err != nil {
		t.Fatalf("RawRootRequest failed: %v", err)
	}

	// For XML parsing test, we mainly verify no error occurred
	// Full XML parsing would require more complex verification
}