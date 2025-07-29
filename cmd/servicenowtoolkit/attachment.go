package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/Krive/ServiceNow-Toolkit/pkg/servicenow/attachment"
)

var attachmentCmd = &cobra.Command{
	Use:   "attachment",
	Short: "ServiceNow attachment operations",
	Long:  "Upload, download, list, and delete attachments in ServiceNow",
}

var attachmentListCmd = &cobra.Command{
	Use:   "list [table_name] [sys_id]",
	Short: "List attachments for a record",
	Long:  "List all attachments associated with a specific record",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := createClient()
		if err != nil {
			return err
		}

		tableName := args[0]
		sysID := args[1]
		format, _ := cmd.Flags().GetString("format")

		attachClient := attachment.NewAttachmentClient(client.Core())
		attachments, err := attachClient.List(tableName, sysID)
		if err != nil {
			return fmt.Errorf("failed to list attachments: %w", err)
		}

		return outputAttachments(attachments, format)
	},
}

var attachmentGetCmd = &cobra.Command{
	Use:   "get [attachment_id]",
	Short: "Download an attachment",
	Long:  "Download an attachment by its sys_id",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := createClient()
		if err != nil {
			return err
		}

		attachmentID := args[0]
		outputPath, _ := cmd.Flags().GetString("output")

		// Determine output file path
		if outputPath == "" {
			outputPath = fmt.Sprintf("attachment_%s", attachmentID)
		}

		attachClient := attachment.NewAttachmentClient(client.Core())
		err = attachClient.Download(attachmentID, outputPath)
		if err != nil {
			return fmt.Errorf("failed to download attachment: %w", err)
		}

		fmt.Printf("✅ Downloaded attachment: %s\n", outputPath)
		return nil
	},
}

var attachmentUploadCmd = &cobra.Command{
	Use:   "upload [table_name] [sys_id] [file_path]",
	Short: "Upload an attachment",
	Long:  "Upload a file as an attachment to a specific record",
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := createClient()
		if err != nil {
			return err
		}

		tableName := args[0]
		sysID := args[1]
		filePath := args[2]

		// Extract filename from path
		filename := filePath
		if idx := len(filePath) - 1; idx >= 0 {
			for i := idx; i >= 0; i-- {
				if filePath[i] == '/' || filePath[i] == '\\' {
					filename = filePath[i+1:]
					break
				}
			}
		}

		attachClient := attachment.NewAttachmentClient(client.Core())
		attachment, err := attachClient.Upload(tableName, sysID, filePath)
		if err != nil {
			return fmt.Errorf("failed to upload attachment: %w", err)
		}

		fmt.Printf("✅ Uploaded attachment: %s (ID: %s)\n", filename, attachment["sys_id"])
		return nil
	},
}

var attachmentDeleteCmd = &cobra.Command{
	Use:   "delete [attachment_id]",
	Short: "Delete an attachment",
	Long:  "Delete an attachment by its sys_id",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := createClient()
		if err != nil {
			return err
		}

		attachmentID := args[0]
		confirm, _ := cmd.Flags().GetBool("confirm")

		if !confirm {
			fmt.Printf("⚠️  This will permanently delete attachment %s. Use --confirm to proceed.\n", attachmentID)
			return nil
		}

		attachClient := attachment.NewAttachmentClient(client.Core())
		err = attachClient.Delete(attachmentID)
		if err != nil {
			return fmt.Errorf("failed to delete attachment: %w", err)
		}

		fmt.Printf("✅ Deleted attachment: %s\n", attachmentID)
		return nil
	},
}

// Output functions
func outputAttachments(attachments interface{}, format string) error {
	switch format {
	case "json":
		return json.NewEncoder(os.Stdout).Encode(attachments)
	default:
		fmt.Println("Attachments:")
		if data, err := json.MarshalIndent(attachments, "", "  "); err == nil {
			fmt.Println(string(data))
		}
		return nil
	}
}

func init() {
	// List command flags
	attachmentListCmd.Flags().StringP("format", "f", "table", "Output format (table, json)")

	// Get command flags
	attachmentGetCmd.Flags().StringP("output", "o", "", "Output file path (defaults to original filename)")

	// Delete command flags
	attachmentDeleteCmd.Flags().BoolP("confirm", "", false, "Confirm deletion (required for safety)")

	// Add subcommands
	attachmentCmd.AddCommand(attachmentListCmd, attachmentGetCmd, attachmentUploadCmd, attachmentDeleteCmd)
	rootCmd.AddCommand(attachmentCmd)
}
