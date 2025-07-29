package main

import (
	"fmt"

	"github.com/Krive/ServiceNow-Toolkit/internal/config"
	"github.com/Krive/ServiceNow-Toolkit/pkg/servicenow/attachment"
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

	// Create a test incident to attach to
	tableClient := table.NewTableClient(client, "incident")
	newRecord := map[string]interface{}{
		"short_description": "Test for attachment",
	}
	created, err := tableClient.Create(newRecord)
	if err != nil {
		fmt.Println("Error creating record:", err)
		return
	}
	sysID := created["sys_id"].(string)

	// Attachment operations
	attachClient := attachment.NewAttachmentClient(client)

	// Upload a file (replace with your file path)
	filePath := "/Users/muellersv/Documents/Scripts/ServiceNow-Toolkit/test.txt"
	uploaded, err := attachClient.Upload("incident", sysID, filePath)
	if err != nil {
		fmt.Println("Error uploading:", err)
		return
	}
	attachID := uploaded["sys_id"].(string)
	fmt.Printf("Uploaded attachment: %+v\n", uploaded)

	// List attachments
	attachments, err := attachClient.List("incident", sysID)
	if err != nil {
		fmt.Println("Error listing:", err)
		return
	}
	fmt.Printf("Attachments: %+v\n", attachments)

	// Download
	savePath := "/Users/muellersv/Dowonloads/test.txt/testfile.txt"
	err = attachClient.Download(attachID, savePath)
	if err != nil {
		fmt.Println("Error downloading:", err)
		return
	}
	fmt.Println("Downloaded successfully to", savePath)

	// Delete
	err = attachClient.Delete(attachID)
	if err != nil {
		fmt.Println("Error deleting:", err)
		return
	}
	fmt.Println("Attachment deleted")
}
