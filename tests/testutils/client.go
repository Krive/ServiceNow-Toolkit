package testutils

import (
	"github.com/Krive/ServiceNow-Toolkit/pkg/servicenow/core"
)

// NewMockClient creates a properly initialized mock client for testing
func NewMockClient(baseURL string) (*core.Client, error) {
	// Use the proper constructor to ensure all fields are initialized correctly
	return core.NewClientBasicAuth(baseURL, "test", "test")
}