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

	// Create (POST)
	newRecord := map[string]interface{}{
		"short_description": "Test incident from ServiceNow Toolkit",
		"priority":          "3",
		"state":             "1",
	}
	created, err := tableClient.Create(newRecord)
	if err != nil {
		fmt.Println("Error creating record:", err)
		return
	}
	sysID := created["sys_id"].(string)
	fmt.Printf("Created record: %+v\n", created)

	// Get
	got, err := tableClient.Get(sysID)
	if err != nil {
		fmt.Println("Error getting record:", err)
		return
	}
	fmt.Printf("Got record: %+v\n", got)

	// Update (PATCH)
	updateData := map[string]interface{}{
		"short_description": "Updated test incident",
		"priority":          "2",
	}
	updated, err := tableClient.Update(sysID, updateData)
	if err != nil {
		fmt.Println("Error updating record (PATCH):", err)
		return
	}
	fmt.Printf("Updated record (PATCH): %+v\n", updated)

	// Put (full replace)
	putData := map[string]interface{}{
		"short_description": "Fully replaced incident",
		"priority":          "4",
		"state":             "1",
	}
	putRecord, err := tableClient.Put(sysID, putData)
	if err != nil {
		fmt.Println("Error putting record:", err)
		return
	}
	fmt.Printf("Put record: %+v\n", putRecord)

	// Delete
	err = tableClient.Delete(sysID)
	if err != nil {
		fmt.Println("Error deleting record:", err)
		return
	}
	fmt.Println("Record deleted successfully")
}
