package unit

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/Krive/ServiceNow-Toolkit/pkg/servicenow/core"
	"github.com/Krive/ServiceNow-Toolkit/pkg/servicenow/table"
)

func TestTableClient_NewTableClient(t *testing.T) {
	client := &core.Client{}
	tableClient := table.NewTableClient(client, "incident")
	
	if tableClient == nil {
		t.Fatal("NewTableClient should return a non-nil TableClient")
	}
}

func TestTableClient_Get_Success(t *testing.T) {
	// Create mock response
	mockRecord := map[string]interface{}{
		"sys_id":            "incident-123",
		"number":            "INC0000001",
		"short_description": "Test incident",
		"priority":          "1",
		"state":             "1",
	}
	
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET request, got %s", r.Method)
		}
		
		if !strings.Contains(r.URL.Path, "/incident/incident-123") {
			t.Errorf("Expected path to contain '/incident/incident-123', got %s", r.URL.Path)
		}
		
		response := map[string]interface{}{
			"result": mockRecord,
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()
	
	// Create client with mock server
	restyClient := resty.New().SetHostURL(server.URL)
	client := &core.Client{
		BaseURL: server.URL,
		Client:  restyClient,
	}
	
	tableClient := table.NewTableClient(client, "incident")
	
	// Test Get
	record, err := tableClient.Get("incident-123")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	if record == nil {
		t.Fatal("Expected record to be returned")
	}
	
	if record["number"] != "INC0000001" {
		t.Errorf("Expected number 'INC0000001', got '%v'", record["number"])
	}
}

func TestTableClient_List_Success(t *testing.T) {
	// Create mock response
	mockRecords := []map[string]interface{}{
		{
			"sys_id": "incident-1",
			"number": "INC0000001",
			"state":  "1",
		},
		{
			"sys_id": "incident-2", 
			"number": "INC0000002",
			"state":  "2",
		},
	}
	
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET request, got %s", r.Method)
		}
		
		// Check query parameters
		query := r.URL.Query()
		if query.Get("sysparm_limit") != "10" {
			t.Errorf("Expected sysparm_limit=10, got %s", query.Get("sysparm_limit"))
		}
		
		response := map[string]interface{}{
			"result": mockRecords,
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()
	
	// Create client with mock server
	restyClient := resty.New().SetHostURL(server.URL)
	client := &core.Client{
		BaseURL: server.URL,
		Client:  restyClient,
	}
	
	tableClient := table.NewTableClient(client, "incident")
	
	// Test List with parameters
	params := map[string]string{
		"sysparm_limit": "10",
		"sysparm_query": "state=1",
	}
	
	records, err := tableClient.List(params)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	if len(records) != 2 {
		t.Errorf("Expected 2 records, got %d", len(records))
	}
	
	if records[0]["number"] != "INC0000001" {
		t.Errorf("Expected first record number 'INC0000001', got '%v'", records[0]["number"])
	}
}

func TestTableClient_Create_Success(t *testing.T) {
	// Create mock response
	mockRecord := map[string]interface{}{
		"sys_id":            "incident-new",
		"number":            "INC0000003",
		"short_description": "New test incident",
		"priority":          "2",
		"state":             "1",
	}
	
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		
		// Verify content type
		contentType := r.Header.Get("Content-Type")
		if !strings.Contains(contentType, "application/json") {
			t.Errorf("Expected JSON content type, got %s", contentType)
		}
		
		// Parse request body
		var requestData map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&requestData)
		if err != nil {
			t.Fatalf("Failed to decode request body: %v", err)
		}
		
		// Verify request data
		if requestData["short_description"] != "New test incident" {
			t.Errorf("Expected short_description 'New test incident', got '%v'", requestData["short_description"])
		}
		
		response := map[string]interface{}{
			"result": mockRecord,
		}
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()
	
	// Create client with mock server
	restyClient := resty.New().SetHostURL(server.URL)
	client := &core.Client{
		BaseURL: server.URL,
		Client:  restyClient,
	}
	
	tableClient := table.NewTableClient(client, "incident")
	
	// Test Create
	recordData := map[string]interface{}{
		"short_description": "New test incident",
		"priority":          "2",
	}
	
	record, err := tableClient.Create(recordData)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	if record == nil {
		t.Fatal("Expected record to be returned")
	}
	
	if record["sys_id"] != "incident-new" {
		t.Errorf("Expected sys_id 'incident-new', got '%v'", record["sys_id"])
	}
}

func TestTableClient_Update_Success(t *testing.T) {
	// Create mock response
	mockRecord := map[string]interface{}{
		"sys_id":            "incident-123",
		"number":            "INC0000001",
		"short_description": "Updated incident",
		"priority":          "1",
		"state":             "2",
	}
	
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PATCH" && r.Method != "PUT" {
			t.Errorf("Expected PATCH or PUT request, got %s", r.Method)
		}
		
		if !strings.Contains(r.URL.Path, "/incident/incident-123") {
			t.Errorf("Expected path to contain '/incident/incident-123', got %s", r.URL.Path)
		}
		
		// Parse request body
		var requestData map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&requestData)
		if err != nil {
			t.Fatalf("Failed to decode request body: %v", err)
		}
		
		// Verify update data
		if requestData["state"] != "2" {
			t.Errorf("Expected state '2', got '%v'", requestData["state"])
		}
		
		response := map[string]interface{}{
			"result": mockRecord,
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()
	
	// Create client with mock server
	restyClient := resty.New().SetHostURL(server.URL)
	client := &core.Client{
		BaseURL: server.URL,
		Client:  restyClient,
	}
	
	tableClient := table.NewTableClient(client, "incident")
	
	// Test Update
	updateData := map[string]interface{}{
		"state":             "2",
		"short_description": "Updated incident",
	}
	
	record, err := tableClient.Update("incident-123", updateData)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	if record == nil {
		t.Fatal("Expected record to be returned")
	}
	
	if record["state"] != "2" {
		t.Errorf("Expected state '2', got '%v'", record["state"])
	}
}

func TestTableClient_Delete_Success(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("Expected DELETE request, got %s", r.Method)
		}
		
		if !strings.Contains(r.URL.Path, "/incident/incident-123") {
			t.Errorf("Expected path to contain '/incident/incident-123', got %s", r.URL.Path)
		}
		
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	
	// Create client with mock server
	restyClient := resty.New().SetHostURL(server.URL)
	client := &core.Client{
		BaseURL: server.URL,
		Client:  restyClient,
	}
	
	tableClient := table.NewTableClient(client, "incident")
	
	// Test Delete
	err := tableClient.Delete("incident-123")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestTableClient_GetWithContext(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"result": map[string]interface{}{
				"sys_id": "incident-123",
				"number": "INC0000001",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()
	
	restyClient := resty.New().SetHostURL(server.URL)
	client := &core.Client{
		BaseURL: server.URL,
		Client:  restyClient,
	}
	
	tableClient := table.NewTableClient(client, "incident")
	
	// Test with context
	ctx := context.Background()
	record, err := tableClient.GetWithContext(ctx, "incident-123")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	if record == nil {
		t.Fatal("Expected record to be returned")
	}
}

func TestTableClient_ListWithContext(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"result": []map[string]interface{}{
				{
					"sys_id": "incident-1",
					"number": "INC0000001",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()
	
	restyClient := resty.New().SetHostURL(server.URL)
	client := &core.Client{
		BaseURL: server.URL,
		Client:  restyClient,
	}
	
	tableClient := table.NewTableClient(client, "incident")
	
	// Test with context
	ctx := context.Background()
	records, err := tableClient.ListWithContext(ctx, nil)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	if len(records) != 1 {
		t.Errorf("Expected 1 record, got %d", len(records))
	}
}

func TestTableClient_ErrorHandling(t *testing.T) {
	// Create mock server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		response := map[string]interface{}{
			"error": map[string]interface{}{
				"message": "Record not found",
				"detail":  "The requested record does not exist",
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()
	
	restyClient := resty.New().SetHostURL(server.URL)
	client := &core.Client{
		BaseURL: server.URL,
		Client:  restyClient,
	}
	
	tableClient := table.NewTableClient(client, "incident")
	
	// Test Get with error
	_, err := tableClient.Get("nonexistent-id")
	if err == nil {
		t.Error("Expected error for nonexistent record")
	}
}

func TestTableClient_InvalidTable(t *testing.T) {
	client := &core.Client{
		BaseURL: "http://example.com",
	}
	
	// Test with empty table name
	tableClient := table.NewTableClient(client, "")
	if tableClient == nil {
		t.Error("NewTableClient should handle empty table name gracefully")
	}
}

func TestTableListOptions_Struct(t *testing.T) {
	options := table.ListOptions{
		Query:                "state=1^priority=1",
		Fields:               []string{"sys_id", "number", "short_description"},
		Limit:                25,
		Offset:               50,
		DisplayValue:         "all",
		ExcludeReferenceLink: true,
		SuppressPaginationHeader: false,
	}
	
	if options.Query != "state=1^priority=1" {
		t.Errorf("Expected query 'state=1^priority=1', got '%s'", options.Query)
	}
	
	if len(options.Fields) != 3 {
		t.Errorf("Expected 3 fields, got %d", len(options.Fields))
	}
	
	if options.Limit != 25 {
		t.Errorf("Expected limit 25, got %d", options.Limit)
	}
	
	if !options.ExcludeReferenceLink {
		t.Error("Expected ExcludeReferenceLink to be true")
	}
}

func TestTableClient_ListOpt_Success(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify query parameters
		query := r.URL.Query()
		if query.Get("sysparm_query") != "state=1" {
			t.Errorf("Expected sysparm_query=state=1, got %s", query.Get("sysparm_query"))
		}
		
		if query.Get("sysparm_fields") != "sys_id,number" {
			t.Errorf("Expected sysparm_fields=sys_id,number, got %s", query.Get("sysparm_fields"))
		}
		
		response := map[string]interface{}{
			"result": []map[string]interface{}{
				{"sys_id": "1", "number": "INC0000001"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()
	
	restyClient := resty.New().SetHostURL(server.URL)
	client := &core.Client{
		BaseURL: server.URL,
		Client:  restyClient,
	}
	
	tableClient := table.NewTableClient(client, "incident")
	
	// Test ListOpt
	options := table.ListOptions{
		Query:  "state=1",
		Fields: []string{"sys_id", "number"},
		Limit:  10,
	}
	
	records, err := tableClient.ListOpt(options)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	if len(records) != 1 {
		t.Errorf("Expected 1 record, got %d", len(records))
	}
}