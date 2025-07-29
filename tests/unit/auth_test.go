package unit

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Krive/ServiceNow-Toolkit/pkg/servicenow/core"
	"github.com/go-resty/resty/v2"
)

func TestBasicAuth(t *testing.T) {
	auth := core.NewBasicAuth("testuser", "testpass")

	client := resty.New()
	err := auth.Apply(client)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Check if Authorization header is set correctly
	headers := client.Header
	authHeader := headers.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Basic ") {
		t.Errorf("Expected Basic auth header, got: %s", authHeader)
	}

	// Test IsExpired
	if auth.IsExpired() {
		t.Error("Basic auth should never expire")
	}

	// Test Refresh (should be no-op)
	err = auth.Refresh()
	if err != nil {
		t.Errorf("Basic auth refresh should not return error, got: %v", err)
	}
}

func TestAPIKeyAuth(t *testing.T) {
	apiKey := "test-api-key-123"
	auth := core.NewAPIKeyAuth(apiKey)

	client := resty.New()
	err := auth.Apply(client)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Check if x-sn-apikey header is set correctly
	headers := client.Header
	keyHeader := headers.Get("x-sn-apikey")
	if keyHeader != apiKey {
		t.Errorf("Expected x-sn-apikey header to be %s, got: %s", apiKey, keyHeader)
	}

	// Test IsExpired
	if auth.IsExpired() {
		t.Error("API key auth should never expire")
	}

	// Test Refresh (should be no-op)
	err = auth.Refresh()
	if err != nil {
		t.Errorf("API key auth refresh should not return error, got: %v", err)
	}
}

func TestAPIKeyAuthEmptyKey(t *testing.T) {
	auth := core.NewAPIKeyAuth("")

	client := resty.New()
	err := auth.Apply(client)
	if err == nil {
		t.Error("Expected error for empty API key, got nil")
	}

	expectedMsg := "API key is required"
	if !strings.Contains(err.Error(), expectedMsg) {
		t.Errorf("Expected error message to contain '%s', got: %s", expectedMsg, err.Error())
	}
}

// MockTokenStorage implements TokenStorage for testing
type MockTokenStorage struct {
	tokens map[string]*core.OAuthToken
}

func NewMockTokenStorage() *MockTokenStorage {
	return &MockTokenStorage{
		tokens: make(map[string]*core.OAuthToken),
	}
}

func (m *MockTokenStorage) Save(key string, token *core.OAuthToken) error {
	m.tokens[key] = token
	return nil
}

func (m *MockTokenStorage) Load(key string) (*core.OAuthToken, error) {
	token, exists := m.tokens[key]
	if !exists {
		return nil, nil
	}
	return token, nil
}

func (m *MockTokenStorage) Delete(key string) error {
	delete(m.tokens, key)
	return nil
}

func TestOAuthClientCredentials(t *testing.T) {
	// Mock OAuth server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/oauth_token.do" {
			t.Errorf("Expected /oauth_token.do, got %s", r.URL.Path)
		}

		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}

		// Check form data
		err := r.ParseForm()
		if err != nil {
			t.Fatalf("Failed to parse form: %v", err)
		}

		if r.FormValue("grant_type") != "client_credentials" {
			t.Errorf("Expected grant_type=client_credentials, got %s", r.FormValue("grant_type"))
		}

		if r.FormValue("client_id") != "test-client-id" {
			t.Errorf("Expected client_id=test-client-id, got %s", r.FormValue("client_id"))
		}

		// Return mock token
		token := core.OAuthToken{
			AccessToken: "test-access-token",
			TokenType:   "Bearer",
			ExpiresIn:   3600,
			Scope:       "useraccount",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(token)
	}))
	defer server.Close()

	// Create OAuth client credentials auth with mock storage
	mockStorage := NewMockTokenStorage()
	auth := core.NewOAuthClientCredentialsWithStorage(server.URL, "test-client-id", "test-client-secret", mockStorage)

	// Test initial state
	if !auth.IsExpired() {
		t.Error("Expected auth to be expired initially")
	}

	// Apply auth (should trigger token fetch)
	client := resty.New()
	err := auth.Apply(client)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Check if Authorization header is set correctly
	headers := client.Header
	authHeader := headers.Get("Authorization")
	expectedHeader := "Bearer test-access-token"
	if authHeader != expectedHeader {
		t.Errorf("Expected Authorization header to be '%s', got: '%s'", expectedHeader, authHeader)
	}

	// Test IsExpired after successful auth
	if auth.IsExpired() {
		t.Error("Expected auth to not be expired after successful token fetch")
	}
}

func TestOAuthClientCredentialsRefresh(t *testing.T) {
	requestCount := 0
	
	// Mock OAuth server that tracks request count
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		
		token := core.OAuthToken{
			AccessToken: "test-access-token-" + string(rune(requestCount)),
			TokenType:   "Bearer",
			ExpiresIn:   1, // Very short expiry for testing
			Scope:       "useraccount",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(token)
	}))
	defer server.Close()

	auth := core.NewOAuthClientCredentials(server.URL, "test-client-id", "test-client-secret")
	client := resty.New()

	// First request should fetch token
	err := auth.Apply(client)
	if err != nil {
		t.Fatalf("Expected no error on first apply, got %v", err)
	}

	if requestCount != 1 {
		t.Errorf("Expected 1 request after first apply, got %d", requestCount)
	}

	// Wait for token to expire
	time.Sleep(2 * time.Second)

	// Second request should refresh token
	err = auth.Apply(client)
	if err != nil {
		t.Fatalf("Expected no error on second apply, got %v", err)
	}

	if requestCount != 2 {
		t.Errorf("Expected 2 requests after token refresh, got %d", requestCount)
	}
}

func TestOAuthAuthorizationCodeRefresh(t *testing.T) {
	requestCount := 0
	
	// Mock OAuth server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		
		err := r.ParseForm()
		if err != nil {
			t.Fatalf("Failed to parse form: %v", err)
		}

		if r.FormValue("grant_type") != "refresh_token" {
			t.Errorf("Expected grant_type=refresh_token, got %s", r.FormValue("grant_type"))
		}

		if r.FormValue("refresh_token") != "test-refresh-token" {
			t.Errorf("Expected refresh_token=test-refresh-token, got %s", r.FormValue("refresh_token"))
		}

		// Return new access token
		token := core.OAuthToken{
			AccessToken:  "new-access-token-" + string(rune(requestCount)),
			TokenType:    "Bearer",
			ExpiresIn:    3600,
			RefreshToken: "test-refresh-token", // Keep same refresh token
			Scope:        "useraccount",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(token)
	}))
	defer server.Close()

	auth := core.NewOAuthAuthorizationCode(server.URL, "test-client-id", "test-client-secret", "test-refresh-token")
	client := resty.New()

	// Should be expired initially (no access token)
	if !auth.IsExpired() {
		t.Error("Expected auth to be expired initially")
	}

	// Apply should trigger refresh
	err := auth.Apply(client)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if requestCount != 1 {
		t.Errorf("Expected 1 refresh request, got %d", requestCount)
	}

	// Check Authorization header
	headers := client.Header
	authHeader := headers.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer new-access-token-") {
		t.Errorf("Expected Bearer token with new access token, got: %s", authHeader)
	}

	// Should not be expired after successful refresh
	if auth.IsExpired() {
		t.Error("Expected auth to not be expired after successful refresh")
	}
}

func TestOAuthAuthorizationCodeNoRefreshToken(t *testing.T) {
	auth := core.NewOAuthAuthorizationCode("https://example.com", "client-id", "client-secret", "")
	
	err := auth.Refresh()
	if err == nil {
		t.Error("Expected error when no refresh token available")
	}

	expectedMsg := "no refresh token available"
	if !strings.Contains(err.Error(), expectedMsg) {
		t.Errorf("Expected error message to contain '%s', got: %s", expectedMsg, err.Error())
	}
}

func TestOAuthServerError(t *testing.T) {
	// Mock server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("Invalid credentials"))
	}))
	defer server.Close()

	auth := core.NewOAuthClientCredentials(server.URL, "bad-client-id", "bad-secret")
	client := resty.New()

	err := auth.Apply(client)
	if err == nil {
		t.Error("Expected error for invalid credentials")
	}

	if !strings.Contains(err.Error(), "OAuth request failed") {
		t.Errorf("Expected OAuth error message, got: %s", err.Error())
	}
}