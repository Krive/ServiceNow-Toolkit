package main

import (
	"fmt"

	"github.com/Krive/ServiceNow-Toolkit/internal/config"
	"github.com/Krive/ServiceNow-Toolkit/pkg/servicenow"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Println("Error loading config:", err)
		return
	}

	// OAuth with refresh token - automatically handles token refresh
	// Note: You would typically get the refresh token from an initial OAuth authorization flow
	client, err := servicenow.NewClientOAuthRefresh(
		cfg.InstanceURL, 
		cfg.ClientID, 
		cfg.ClientSecret, 
		"YOUR_REFRESH_TOKEN_HERE", // Replace with actual refresh token
	)
	if err != nil {
		fmt.Println("Error creating OAuth refresh client:", err)
		return
	}
	
	// The client will automatically refresh the access token when needed
	incidents, err := client.Table("incident").List(map[string]string{"sysparm_limit": "5"})
	if err != nil {
		fmt.Println("Error fetching incidents:", err)
		return
	}
	fmt.Printf("OAuth Refresh Token Incidents: %+v\n", incidents)

	// Demonstrate using the unified Config approach
	refreshClient, err := servicenow.NewClient(servicenow.Config{
		InstanceURL:  cfg.InstanceURL,
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RefreshToken: "YOUR_REFRESH_TOKEN_HERE", // Replace with actual refresh token
	})
	if err != nil {
		fmt.Println("Error creating config-based OAuth refresh client:", err)
		return
	}
	
	incidents, err = refreshClient.Table("incident").List(map[string]string{"sysparm_limit": "3"})
	if err != nil {
		fmt.Println("Error fetching incidents with config client:", err)
		return
	}
	fmt.Printf("Config-based OAuth Refresh Incidents: %+v\n", incidents)
	
	// Note: Tokens are automatically persisted to ~/.servicenowtoolkit/tokens/ directory
	// They will be reused across application restarts
	fmt.Println("OAuth tokens are automatically saved and will be reused on next run")
}