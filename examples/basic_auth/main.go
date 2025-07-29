package main

import (
	"fmt"

	"github.com/Krive/ServiceNow-Toolkit/internal/config"
	"github.com/Krive/ServiceNow-Toolkit/pkg/servicenow"
)

func main() {
	cfg, err := config.LoadConfig()
	fmt.Println("config:", cfg)
	if err != nil {
		fmt.Println("Error loading config:", err)
		return
	}

	// Basic Auth using new SDK entry point
	client, err := servicenow.NewClientBasicAuth(cfg.InstanceURL, cfg.Username, cfg.Password)
	fmt.Println("client:", client)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	incidents, err := client.Table("incident").List(map[string]string{"sysparm_limit": "5"})
	fmt.Println("incidents:", incidents)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Printf("Basic Auth Incidents: %+v\n", incidents)

	// OAuth using new SDK entry point
	oauthClient, err := servicenow.NewClientOAuth(cfg.InstanceURL, cfg.ClientID, cfg.ClientSecret)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	incidents, err = oauthClient.Table("incident").List(map[string]string{"sysparm_limit": "5"})
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Printf("OAuth Incidents: %+v\n", incidents)

	// Demonstrate the unified Config approach
	configClient, err := servicenow.NewClient(servicenow.Config{
		InstanceURL: cfg.InstanceURL,
		Username:    cfg.Username,
		Password:    cfg.Password,
	})
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	incidents, err = configClient.Table("incident").List(map[string]string{"sysparm_limit": "3"})
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Printf("Config-based Incidents: %+v\n", incidents)

	// Example showing all authentication methods in one place
	fmt.Println("\n=== Authentication Methods Comparison ===")
	fmt.Printf("1. Basic Auth: Username/Password (simplest, less secure)\n")
	fmt.Printf("2. OAuth Client Credentials: App-to-app auth (no user context)\n")
	fmt.Printf("3. OAuth Refresh Token: User-delegated auth (preserves user context)\n")
	fmt.Printf("4. API Key: Simple token-based auth (newest ServiceNow method)\n")
}
