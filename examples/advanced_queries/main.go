package main

import (
	"fmt"

	"github.com/Krive/ServiceNow-Toolkit/internal/config"
	"github.com/Krive/ServiceNow-Toolkit/pkg/servicenow/core"
	"github.com/Krive/ServiceNow-Toolkit/pkg/servicenow/table"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Println("Error loading config:", err)
		return
	}

	client, err := core.NewClientBasicAuth(cfg.InstanceURL, cfg.Username, cfg.Password)
	if err != nil {
		fmt.Println("Error creating client:", err)
		return
	}
	tableClient := table.NewTableClient(client, "incident")

	// QueryBuilder + ListOptions example
	qb := tableClient.Query().
		And("priority", table.Eq, 1).
		Or("state", table.Eq, "New").
		OrderByDesc("number")

	query, err := qb.Build()
	if err != nil {
		fmt.Println("Error building query:", err)
		return
	}

	options := table.ListOptions{
		Query:        query,
		Fields:       []string{"number", "short_description", "priority", "state"},
		Limit:        5,
		DisplayValue: core.DisplayAll,
		NoCount:      true,
	}

	incidents, err := tableClient.ListOpt(options)
	if err != nil {
		fmt.Println("Error listing with options:", err)
		return
	}
	fmt.Printf("Incidents with options: %+v\n", incidents)

	// Pagination example
	pagOptions := table.ListOptions{
		Query:  "active=true",
		Fields: []string{"number"},
	}
	allIncidents, err := tableClient.Paginate(pagOptions, 10) // Fetch in pages of 10
	if err != nil {
		fmt.Println("Error paginating:", err)
		return
	}
	fmt.Printf("Total paginated incidents: %d\n", len(allIncidents))

	// Metadata: GetSchema
	schema, err := tableClient.GetSchema()
	if err != nil {
		fmt.Println("Error getting schema:", err)
		return
	}
	fmt.Printf("Schema columns: %+v\n", schema) // Now a []ColumnMetadata

	// GetKeys example
	keys, err := tableClient.GetKeys("priority=1")
	if err != nil {
		fmt.Println("Error getting keys:", err)
		return
	}
	fmt.Printf("Sys IDs for priority=1: %+v\n", keys)
}
