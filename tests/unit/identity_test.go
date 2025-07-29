package unit

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/Krive/ServiceNow-Toolkit/pkg/servicenow/core"
	"github.com/Krive/ServiceNow-Toolkit/pkg/servicenow/identity"
)

func TestIdentityClient_NewIdentityClient(t *testing.T) {
	client := &core.Client{}
	identityClient := identity.NewIdentityClient(client)
	
	if identityClient == nil {
		t.Fatal("NewIdentityClient should return a non-nil IdentityClient")
	}
}

func TestUser_Struct(t *testing.T) {
	user := &identity.User{
		SysID:     "test-user-id",
		UserName:  "testuser",
		FirstName: "Test",
		LastName:  "User",
		Email:     "test@example.com",
		Active:    true,
	}
	
	if user.SysID != "test-user-id" {
		t.Errorf("Expected SysID 'test-user-id', got '%s'", user.SysID)
	}
	
	if user.UserName != "testuser" {
		t.Errorf("Expected UserName 'testuser', got '%s'", user.UserName)
	}
	
	if !user.Active {
		t.Error("Expected user to be active")
	}
}

func TestRole_Struct(t *testing.T) {
	role := &identity.Role{
		SysID:       "test-role-id",
		Name:        "admin",
		Description: "Administrator role",
		Active:      true,
		CreatedOn:   time.Now(),
	}
	
	if role.Name != "admin" {
		t.Errorf("Expected role name 'admin', got '%s'", role.Name)
	}
	
	if !role.Active {
		t.Error("Expected role to be active")
	}
}

func TestGroup_Struct(t *testing.T) {
	group := &identity.Group{
		SysID:       "test-group-id",
		Name:        "test-group",
		Description: "Test Group",
		Active:      true,
		Type:        "manual",
		CreatedOn:   time.Now(),
	}
	
	if group.Name != "test-group" {
		t.Errorf("Expected group name 'test-group', got '%s'", group.Name)
	}
	
	if group.Type != "manual" {
		t.Errorf("Expected group type 'manual', got '%s'", group.Type)
	}
}

func TestUserRole_Struct(t *testing.T) {
	userRole := &identity.UserRole{
		SysID:     "test-user-role-id",
		UserSysID: "user-123",
		RoleSysID: "role-456",
		State:     "active",
		GrantedOn: time.Now(),
	}
	
	if userRole.UserSysID != "user-123" {
		t.Errorf("Expected UserSysID 'user-123', got '%s'", userRole.UserSysID)
	}
	
	if userRole.RoleSysID != "role-456" {
		t.Errorf("Expected RoleSysID 'role-456', got '%s'", userRole.RoleSysID)
	}
}

func TestGroupMember_Struct(t *testing.T) {
	member := &identity.GroupMember{
		SysID:      "test-member-id",
		GroupSysID: "group-123",
		UserSysID:  "user-456",
	}
	
	if member.GroupSysID != "group-123" {
		t.Errorf("Expected GroupSysID 'group-123', got '%s'", member.GroupSysID)
	}
	
	if member.UserSysID != "user-456" {
		t.Errorf("Expected UserSysID 'user-456', got '%s'", member.UserSysID)
	}
}

func TestUserFilter_Struct(t *testing.T) {
	filter := &identity.UserFilter{
		Email:     "test@example.com",
		Active:    &[]bool{true}[0],
		Department: "IT",
		Limit:     50,
		Fields:    []string{"sys_id", "user_name", "email"},
	}
	
	if filter.Email != "test@example.com" {
		t.Errorf("Expected Email 'test@example.com', got '%s'", filter.Email)
	}
	
	if filter.Limit != 50 {
		t.Errorf("Expected Limit 50, got %d", filter.Limit)
	}
	
	if len(filter.Fields) != 3 {
		t.Errorf("Expected 3 fields, got %d", len(filter.Fields))
	}
}

func TestRoleFilter_Struct(t *testing.T) {
	filter := &identity.RoleFilter{
		Name:   "admin",
		Active: &[]bool{true}[0],
		Limit:  25,
	}
	
	if filter.Name != "admin" {
		t.Errorf("Expected Name 'admin', got '%s'", filter.Name)
	}
	
	if filter.Limit != 25 {
		t.Errorf("Expected Limit 25, got %d", filter.Limit)
	}
}

func TestAccessCheckRequest_Struct(t *testing.T) {
	request := &identity.AccessCheckRequest{
		UserSysID:   "user-123",
		Table:       "incident",
		Operation:   "read",
		RecordSysID: "incident-456",
	}
	
	if request.UserSysID != "user-123" {
		t.Errorf("Expected UserSysID 'user-123', got '%s'", request.UserSysID)
	}
	
	if request.Operation != "read" {
		t.Errorf("Expected Operation 'read', got '%s'", request.Operation)
	}
}

func TestUserSession_Struct(t *testing.T) {
	session := &identity.UserSession{
		SysID:          "session-123",
		UserSysID:      "user-456",
		LoginTime:      time.Now(),
		LastAccessTime: time.Now(),
		IPAddress:      "192.168.1.1",
		UserAgent:      "Mozilla/5.0",
	}
	
	if session.UserSysID != "user-456" {
		t.Errorf("Expected UserSysID 'user-456', got '%s'", session.UserSysID)
	}
	
	if session.IPAddress != "192.168.1.1" {
		t.Errorf("Expected IPAddress '192.168.1.1', got '%s'", session.IPAddress)
	}
}

func TestUserPreference_Struct(t *testing.T) {
	pref := &identity.UserPreference{
		SysID:     "pref-123",
		UserSysID: "user-456",
		Name:      "timezone",
		Value:     "America/New_York",
		Type:      "string",
	}
	
	if pref.Name != "timezone" {
		t.Errorf("Expected Name 'timezone', got '%s'", pref.Name)
	}
	
	if pref.Value != "America/New_York" {
		t.Errorf("Expected Value 'America/New_York', got '%s'", pref.Value)
	}
}

// Test client methods with mock server
func TestIdentityClient_GetUser_Success(t *testing.T) {
	// Create mock user response
	mockUser := &identity.User{
		SysID:     "user-123",
		UserName:  "testuser",
		FirstName: "Test",
		LastName:  "User",
		Email:     "test@example.com",
		Active:    true,
	}
	
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET request, got %s", r.Method)
		}
		
		response := map[string]interface{}{
			"result": []interface{}{
				map[string]interface{}{
					"sys_id":     mockUser.SysID,
					"user_name":  mockUser.UserName,
					"first_name": mockUser.FirstName,
					"last_name":  mockUser.LastName,
					"email":      mockUser.Email,
					"active":     "true",
				},
			},
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
	
	identityClient := identity.NewIdentityClient(client)
	
	// Test GetUser
	user, err := identityClient.GetUser("user-123")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	if user == nil {
		t.Fatal("Expected user to be returned")
	}
	
	if user.UserName != "testuser" {
		t.Errorf("Expected username 'testuser', got '%s'", user.UserName)
	}
}

func TestIdentityClient_GetUserWithContext(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"result": []interface{}{
				map[string]interface{}{
					"sys_id":    "user-123",
					"user_name": "testuser",
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
	
	identityClient := identity.NewIdentityClient(client)
	
	// Test with context
	ctx := context.Background()
	user, err := identityClient.GetUserWithContext(ctx, "user-123")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	if user == nil {
		t.Fatal("Expected user to be returned")
	}
}

func TestIdentityClient_ListUsers_EmptyResult(t *testing.T) {
	// Create mock server returning empty result
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"result": []interface{}{},
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
	
	identityClient := identity.NewIdentityClient(client)
	
	users, err := identityClient.ListUsers(nil)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	if len(users) != 0 {
		t.Errorf("Expected 0 users, got %d", len(users))
	}
}

func TestIdentityClient_CreateUser_InvalidData(t *testing.T) {
	client := &core.Client{
		BaseURL: "http://invalid-url",
	}
	
	identityClient := identity.NewIdentityClient(client)
	
	// Test with invalid data (missing required fields)
	userData := map[string]interface{}{
		"invalid_field": "value",
	}
	
	_, err := identityClient.CreateUser(userData)
	if err == nil {
		t.Error("Expected error for invalid user data")
	}
}