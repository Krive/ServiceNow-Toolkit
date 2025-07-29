package main

import (
	"fmt"
	"os"

	"github.com/Krive/ServiceNow-Toolkit/pkg/servicenow"
)

func main() {
	// Get API key from environment variable
	apiKey := os.Getenv("SERVICENOW_API_KEY")
	instanceURL := os.Getenv("SERVICENOW_INSTANCE_URL")
	
	if apiKey == "" {
		fmt.Println("SERVICENOW_API_KEY environment variable is required")
		fmt.Println("You can generate an API key in ServiceNow: System Settings > API Keys")
		return
	}
	
	if instanceURL == "" {
		fmt.Println("SERVICENOW_INSTANCE_URL environment variable is required")
		return
	}

	// Simple API key authentication
	client, err := servicenow.NewClientAPIKey(instanceURL, apiKey)
	if err != nil {
		fmt.Println("Error creating API key client:", err)
		return
	}
	
	// Test the connection by fetching some incidents
	incidents, err := client.Table("incident").List(map[string]string{"sysparm_limit": "5"})
	if err != nil {
		fmt.Println("Error fetching incidents:", err)
		return
	}
	fmt.Printf("API Key Incidents: %+v\n", incidents)

	// Demonstrate using the unified Config approach
	configClient, err := servicenow.NewClient(servicenow.Config{
		InstanceURL: instanceURL,
		APIKey:      apiKey,
	})
	if err != nil {
		fmt.Println("Error creating config-based API key client:", err)
		return
	}
	
	// Test other APIs
	users, err := configClient.Table("sys_user").List(map[string]string{
		"sysparm_limit":  "3",
		"sysparm_fields": "name,email,active",
	})
	if err != nil {
		fmt.Println("Error fetching users:", err)
		return
	}
	fmt.Printf("Config-based API Key Users: %+v\n", users)

	// Test attachments
	attachments, err := configClient.Attachment().List("incident", "some_sys_id")
	if err != nil {
		fmt.Println("Note: Attachment query failed (expected if incident doesn't exist):", err)
	} else {
		fmt.Printf("API Key Attachments: %+v\n", attachments)
	}

	fmt.Println("API key authentication successful!")
	fmt.Println("Tip: API keys are simpler than OAuth but may have different permission scopes")
}