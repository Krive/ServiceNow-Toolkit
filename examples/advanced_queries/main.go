package main

import (
	"fmt"

	"github.com/Krive/ServiceNow-Toolkit/pkg/servicenow"
	"github.com/Krive/ServiceNow-Toolkit/pkg/servicenow/query"
)

func main() {
	// Create a ServiceNow client - replace with your actual credentials
	client, err := servicenow.NewClient(servicenow.Config{
		InstanceURL: "https://your-instance.service-now.com",
		Username:    "your-username",
		Password:    "your-password",
	})
	if err != nil {
		fmt.Println("Error creating client:", err)
		return
	}
	// QueryBuilder example
	qb := query.New().
		Equals("priority", 1).
		Or().
		Equals("state", "New").
		OrderByDesc("number")

	queryString := qb.BuildQuery()
	params := qb.Build()
	fmt.Printf("Built query: %s\n", queryString)

	// Use the ServiceNow client to query incidents
	incidents, err := client.Table("incident").List(params)
	if err != nil {
		fmt.Println("Error listing incidents:", err)
		return
	}
	fmt.Printf("Found %d incidents\n", len(incidents))

	// Show first few records
	for i, incident := range incidents {
		if i >= 3 { // Limit to first 3 for demo
			break
		}
		fmt.Printf("Incident %d: %s - %s\n", i+1, incident["number"], incident["short_description"])
	}

	// Another query example with date conditions
	recentQuery := query.New().
		Equals("active", true).
		And().
		GreaterThan("sys_created_on", "2024-01-01")

	recentParams := recentQuery.Build()
	recentIncidents, err := client.Table("incident").List(recentParams)
	if err != nil {
		fmt.Println("Error listing recent incidents:", err)
		return
	}
	fmt.Printf("Found %d recent active incidents\n", len(recentIncidents))
}
