package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/Krive/ServiceNow-Toolkit/pkg/servicenow"
	"github.com/Krive/ServiceNow-Toolkit/pkg/servicenow/query"
)

func main() {
	// Get credentials from environment
	instanceURL := os.Getenv("SERVICENOW_INSTANCE_URL")
	username := os.Getenv("SERVICENOW_USERNAME")
	password := os.Getenv("SERVICENOW_PASSWORD")

	if instanceURL == "" || username == "" || password == "" {
		fmt.Println("Please set SERVICENOW_INSTANCE_URL, SERVICENOW_USERNAME, and SERVICENOW_PASSWORD")
		return
	}

	// Create client
	client, err := servicenow.NewClientBasicAuth(instanceURL, username, password)
	if err != nil {
		fmt.Printf("Failed to create client: %v\n", err)
		return
	}

	// Example 1: Basic Query Builder Usage
	fmt.Println("=== Example 1: Basic Query Builder ===")
	
	// Build a query using the fluent API
	q := query.New().
		Equals("active", true).
		And().
		Contains("short_description", "test").
		OrderByDesc("sys_created_on").
		Limit(5).
		Fields("number", "short_description", "priority", "state")

	fmt.Printf("Generated query: %s\n", q.BuildQuery())
	fmt.Printf("Full query params: %+v\n\n", q.Build())

	// Example 2: Table Client with Query Builder
	fmt.Println("=== Example 2: Table Client with Query Builder ===")
	
	incidents, err := client.Table("incident").ListWithQuery(q)
	if err != nil {
		fmt.Printf("Query failed: %v\n", err)
	} else {
		fmt.Printf("Found %d incidents\n", len(incidents))
		for _, incident := range incidents {
			fmt.Printf("- %s: %s\n", incident["number"], incident["short_description"])
		}
	}
	fmt.Println()

	// Example 3: Fluent Table Query API
	fmt.Println("=== Example 3: Fluent Table Query API ===")
	
	highPriorityIncidents, err := client.Table("incident").
		Where("priority", query.OpEquals, "1").
		And().
		Equals("state", "1").
		OrderByAsc("number").
		Fields("number", "short_description", "priority").
		Limit(3).
		Execute()

	if err != nil {
		fmt.Printf("Fluent query failed: %v\n", err)
	} else {
		fmt.Printf("Found %d high priority incidents:\n", len(highPriorityIncidents))
		for _, incident := range highPriorityIncidents {
			fmt.Printf("- %s: %s (Priority: %v)\n", 
				incident["number"], 
				incident["short_description"], 
				incident["priority"])
		}
	}
	fmt.Println()

	// Example 4: Context with Timeout
	fmt.Println("=== Example 4: Query with Context Timeout ===")
	
	// Create a context with a 5-second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Execute query with context
	users, err := client.Table("sys_user").
		Where("active", query.OpEquals, true).
		OrderByAsc("name").
		Fields("name", "email", "title").
		Limit(5).
		ExecuteWithContext(ctx)

	if err != nil {
		fmt.Printf("Context query failed: %v\n", err)
	} else {
		fmt.Printf("Found %d active users:\n", len(users))
		for _, user := range users {
			fmt.Printf("- %s (%s) - %s\n", user["name"], user["email"], user["title"])
		}
	}
	fmt.Println()

	// Example 5: Complex Query with OR Logic
	fmt.Println("=== Example 5: Complex Query with OR Logic ===")
	
	criticalQuery := client.Table("incident").
		Where("priority", query.OpEquals, "1").  // Critical priority
		Or().
		Where("urgency", query.OpEquals, "1").   // OR High urgency
		And().
		Equals("active", true).                   // AND active
		OrderByDesc("sys_created_on").
		Fields("number", "short_description", "priority", "urgency", "state")

	criticalIncidents, err := criticalQuery.Execute()
	if err != nil {
		fmt.Printf("Complex query failed: %v\n", err)
	} else {
		fmt.Printf("Found %d critical/urgent incidents\n", len(criticalIncidents))
		for i, incident := range criticalIncidents {
			if i >= 3 { // Show only first 3
				break
			}
			fmt.Printf("- %s: %s (P:%v, U:%v)\n", 
				incident["number"], 
				incident["short_description"],
				incident["priority"],
				incident["urgency"])
		}
	}
	fmt.Println()

	// Example 6: Predefined Query Builders
	fmt.Println("=== Example 6: Predefined Query Builders ===")
	
	// Use predefined query builders for common cases
	recentIncidents, err := client.Table("incident").
		ListWithQuery(query.RecentRecords(7).  // Last 7 days
			Fields("number", "short_description", "sys_created_on").
			OrderByDesc("sys_created_on").
			Limit(3))

	if err != nil {
		fmt.Printf("Recent records query failed: %v\n", err)
	} else {
		fmt.Printf("Recent incidents (last 7 days): %d\n", len(recentIncidents))
		for _, incident := range recentIncidents {
			fmt.Printf("- %s: %s\n", incident["number"], incident["short_description"])
		}
	}
	fmt.Println()

	// Example 7: Count Records
	fmt.Println("=== Example 7: Count Records ===")
	
	count, err := client.Table("incident").
		Equals("active", true).
		Count()

	if err != nil {
		fmt.Printf("Count query failed: %v\n", err)
	} else {
		fmt.Printf("Total active incidents: %d\n", count)
	}
	fmt.Println()

	// Example 8: Search Across Multiple Fields
	fmt.Println("=== Example 8: Text Search ===")
	
	searchResults, err := client.Table("incident").
		ListWithQuery(query.SearchText("password", "short_description", "description").
			Fields("number", "short_description").
			Limit(3))

	if err != nil {
		fmt.Printf("Search query failed: %v\n", err)
	} else {
		fmt.Printf("Found %d incidents containing 'password':\n", len(searchResults))
		for _, incident := range searchResults {
			fmt.Printf("- %s: %s\n", incident["number"], incident["short_description"])
		}
	}

	fmt.Println("\n=== Query Builder Examples Complete ===")
}