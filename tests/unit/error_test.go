package unit

import (
	"net/http"
	"testing"

	"github.com/Krive/ServiceNow-Toolkit/pkg/servicenow/core"
)

func TestServiceNowErrorCreation(t *testing.T) {
	err := core.NewServiceNowError(401, "Authentication failed")

	if err.StatusCode != 401 {
		t.Errorf("Expected status code 401, got %d", err.StatusCode)
	}

	if err.Message != "Authentication failed" {
		t.Errorf("Expected message 'Authentication failed', got %s", err.Message)
	}

	if err.Type != core.ErrorTypeAuthentication {
		t.Errorf("Expected error type authentication, got %s", err.Type)
	}

	if err.Retryable {
		t.Error("Expected authentication error to not be retryable")
	}
}

func TestServiceNowErrorClassification(t *testing.T) {
	tests := []struct {
		statusCode      int
		expectedType    core.ErrorType
		expectedRetryable bool
	}{
		{401, core.ErrorTypeAuthentication, false},
		{403, core.ErrorTypeAuthorization, false},
		{404, core.ErrorTypeNotFound, false},
		{429, core.ErrorTypeRateLimit, true},
		{408, core.ErrorTypeTimeout, true},
		{400, core.ErrorTypeClient, false},
		{500, core.ErrorTypeServer, true},
		{502, core.ErrorTypeServer, true},
		{999, core.ErrorTypeUnknown, false},
	}

	for _, test := range tests {
		t.Run(http.StatusText(test.statusCode), func(t *testing.T) {
			err := core.NewServiceNowError(test.statusCode, "Test error")

			if err.Type != test.expectedType {
				t.Errorf("Expected error type %s, got %s", test.expectedType, err.Type)
			}

			if err.Retryable != test.expectedRetryable {
				t.Errorf("Expected retryable %v, got %v", test.expectedRetryable, err.Retryable)
			}
		})
	}
}

func TestServiceNowErrorMethods(t *testing.T) {
	rateLimitErr := core.NewRateLimitError("Too many requests")
	
	if !rateLimitErr.IsRateLimit() {
		t.Error("Expected rate limit error to return true for IsRateLimit()")
	}

	if !rateLimitErr.IsRetryable() {
		t.Error("Expected rate limit error to be retryable")
	}

	if !rateLimitErr.IsTemporary() {
		t.Error("Expected rate limit error to be temporary")
	}

	if rateLimitErr.IsAuthError() {
		t.Error("Expected rate limit error to not be auth error")
	}

	authErr := core.NewAuthenticationError("Invalid token")
	
	if !authErr.IsAuthError() {
		t.Error("Expected authentication error to return true for IsAuthError()")
	}

	if authErr.IsRetryable() {
		t.Error("Expected authentication error to not be retryable")
	}

	if authErr.IsTemporary() {
		t.Error("Expected authentication error to not be temporary")
	}
}

func TestServiceNowErrorWithDetail(t *testing.T) {
	err := core.NewServiceNowErrorWithDetail(400, "Validation failed", "Field 'name' is required")

	expectedMsg := "ServiceNow client error [400 Bad Request]: Validation failed - Field 'name' is required"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message:\n%s\nGot:\n%s", expectedMsg, err.Error())
	}

	if err.Detail != "Field 'name' is required" {
		t.Errorf("Expected detail 'Field 'name' is required', got %s", err.Detail)
	}
}

func TestIsServiceNowError(t *testing.T) {
	snErr := core.NewValidationError("Invalid input")
	
	// Test with ServiceNow error
	extractedErr, ok := core.IsServiceNowError(snErr)
	if !ok {
		t.Error("Expected IsServiceNowError to return true for ServiceNow error")
	}

	if extractedErr != snErr {
		t.Error("Expected extracted error to be the same as original")
	}

	// Test with regular error
	regularErr := &core.ServiceNowError{}
	_, ok = core.IsServiceNowError(regularErr)
	if !ok {
		t.Error("Expected IsServiceNowError to return true for ServiceNowError type")
	}

	// Test with nil
	_, ok = core.IsServiceNowError(nil)
	if ok {
		t.Error("Expected IsServiceNowError to return false for nil")
	}
}

func TestSpecificErrorConstructors(t *testing.T) {
	// Test NewValidationError
	validationErr := core.NewValidationError("Invalid data")
	if validationErr.Type != core.ErrorTypeValidation {
		t.Errorf("Expected validation error type, got %s", validationErr.Type)
	}

	if validationErr.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status code 400, got %d", validationErr.StatusCode)
	}

	// Test NewTimeoutError
	timeoutErr := core.NewTimeoutError("Request timed out")
	if timeoutErr.Type != core.ErrorTypeTimeout {
		t.Errorf("Expected timeout error type, got %s", timeoutErr.Type)
	}

	if !timeoutErr.IsRetryable() {
		t.Error("Expected timeout error to be retryable")
	}

	// Test NewRateLimitError
	rateLimitErr := core.NewRateLimitError("Rate limit exceeded")
	if rateLimitErr.Type != core.ErrorTypeRateLimit {
		t.Errorf("Expected rate limit error type, got %s", rateLimitErr.Type)
	}

	if rateLimitErr.StatusCode != http.StatusTooManyRequests {
		t.Errorf("Expected status code 429, got %d", rateLimitErr.StatusCode)
	}
}