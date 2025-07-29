package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/Krive/ServiceNow-Toolkit/pkg/servicenow/batch"
)

var batchCmd = &cobra.Command{
	Use:   "batch",
	Short: "Batch operations for high-performance bulk processing",
	Long:  "Perform multiple operations in a single request for improved performance",
}

var batchCreateCmd = &cobra.Command{
	Use:   "create [table]",
	Short: "Create multiple records in batch",
	Long:  "Create multiple records in a single batch operation",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := createClient()
		if err != nil {
			return err
		}

		tableName := args[0]
		dataFile, _ := cmd.Flags().GetString("file")
		data, _ := cmd.Flags().GetString("data")
		format, _ := cmd.Flags().GetString("input-format")

		var records []map[string]interface{}

		if dataFile != "" {
			records, err = loadDataFromFile(dataFile, format)
			if err != nil {
				return fmt.Errorf("failed to load data from file: %w", err)
			}
		} else if data != "" {
			err = json.Unmarshal([]byte(data), &records)
			if err != nil {
				return fmt.Errorf("failed to parse data: %w", err)
			}
		} else {
			return fmt.Errorf("either --file or --data must be provided")
		}

		if len(records) == 0 {
			return fmt.Errorf("no records to create")
		}

		fmt.Printf("Creating %d records in batch...\n", len(records))

		batchClient := client.Batch()
		result, err := batchClient.CreateMultiple(tableName, records)
		if err != nil {
			return fmt.Errorf("batch create failed: %w", err)
		}

		return outputBatchResult(result, "create")
	},
}

var batchUpdateCmd = &cobra.Command{
	Use:   "update [table]",
	Short: "Update multiple records in batch",
	Long:  "Update multiple records in a single batch operation",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := createClient()
		if err != nil {
			return err
		}

		tableName := args[0]
		dataFile, _ := cmd.Flags().GetString("file")
		data, _ := cmd.Flags().GetString("data")
		format, _ := cmd.Flags().GetString("input-format")

		var updates map[string]map[string]interface{}

		if dataFile != "" {
			updates, err = loadUpdatesFromFile(dataFile, format)
			if err != nil {
				return fmt.Errorf("failed to load updates from file: %w", err)
			}
		} else if data != "" {
			err = json.Unmarshal([]byte(data), &updates)
			if err != nil {
				return fmt.Errorf("failed to parse data: %w", err)
			}
		} else {
			return fmt.Errorf("either --file or --data must be provided")
		}

		if len(updates) == 0 {
			return fmt.Errorf("no records to update")
		}

		fmt.Printf("Updating %d records in batch...\n", len(updates))

		batchClient := client.Batch()
		result, err := batchClient.UpdateMultiple(tableName, updates)
		if err != nil {
			return fmt.Errorf("batch update failed: %w", err)
		}

		return outputBatchResult(result, "update")
	},
}

var batchDeleteCmd = &cobra.Command{
	Use:   "delete [table]",
	Short: "Delete multiple records in batch",
	Long:  "Delete multiple records in a single batch operation",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := createClient()
		if err != nil {
			return err
		}

		tableName := args[0]
		sysIDs, _ := cmd.Flags().GetStringSlice("ids")
		idsFile, _ := cmd.Flags().GetString("ids-file")

		var recordIDs []string

		if idsFile != "" {
			recordIDs, err = loadIDsFromFile(idsFile)
			if err != nil {
				return fmt.Errorf("failed to load IDs from file: %w", err)
			}
		} else if len(sysIDs) > 0 {
			recordIDs = sysIDs
		} else {
			return fmt.Errorf("either --ids or --ids-file must be provided")
		}

		if len(recordIDs) == 0 {
			return fmt.Errorf("no record IDs provided")
		}

		fmt.Printf("Deleting %d records in batch...\n", len(recordIDs))

		// Confirm deletion
		confirm, _ := cmd.Flags().GetBool("confirm")
		if !confirm {
			fmt.Printf("⚠️  This will permanently delete %d records. Use --confirm to proceed.\n", len(recordIDs))
			return nil
		}

		batchClient := client.Batch()
		result, err := batchClient.DeleteMultiple(tableName, recordIDs)
		if err != nil {
			return fmt.Errorf("batch delete failed: %w", err)
		}

		return outputBatchResult(result, "delete")
	},
}

var batchMixedCmd = &cobra.Command{
	Use:   "mixed",
	Short: "Execute mixed batch operations",
	Long:  "Execute a combination of create, update, get, and delete operations in a single batch",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := createClient()
		if err != nil {
			return err
		}

		configFile, _ := cmd.Flags().GetString("config")
		if configFile == "" {
			return fmt.Errorf("--config file is required for mixed operations")
		}

		operations, err := loadMixedOperationsFromFile(configFile)
		if err != nil {
			return fmt.Errorf("failed to load operations from config: %w", err)
		}

		fmt.Printf("Executing mixed batch with %d operations...\n", 
			len(operations.Creates)+len(operations.Updates)+len(operations.Gets)+len(operations.Deletes))

		batchClient := client.Batch()
		result, err := batchClient.ExecuteMixed(operations)
		if err != nil {
			return fmt.Errorf("mixed batch execution failed: %w", err)
		}

		return outputMixedBatchResult(result)
	},
}

var batchStatusCmd = &cobra.Command{
	Use:   "status [batch_id]",
	Short: "Check the status of a batch operation",
	Long:  "Check the status and results of a previously executed batch operation",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		batchID := args[0]
		
		// This would typically query the batch status from ServiceNow
		// For now, we'll show a placeholder message
		fmt.Printf("Batch ID: %s\n", batchID)
		fmt.Println("Status: This feature requires ServiceNow batch status API integration")
		fmt.Println("Note: Individual batch operations are currently executed synchronously")
		
		return nil
	},
}

// Helper functions for file operations
func loadDataFromFile(filename, format string) ([]map[string]interface{}, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	switch format {
	case "csv":
		return loadDataFromCSV(file)
	case "json":
		fallthrough
	default:
		return loadDataFromJSON(file)
	}
}

func loadDataFromJSON(r io.Reader) ([]map[string]interface{}, error) {
	var records []map[string]interface{}
	err := json.NewDecoder(r).Decode(&records)
	return records, err
}

func loadDataFromCSV(r io.Reader) ([]map[string]interface{}, error) {
	reader := csv.NewReader(r)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	if len(records) < 1 {
		return []map[string]interface{}{}, nil
	}

	headers := records[0]
	var result []map[string]interface{}

	for _, record := range records[1:] {
		row := make(map[string]interface{})
		for i, value := range record {
			if i < len(headers) {
				// Try to parse as number
				if intVal, err := strconv.Atoi(value); err == nil {
					row[headers[i]] = intVal
				} else if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
					row[headers[i]] = floatVal
				} else if boolVal, err := strconv.ParseBool(value); err == nil {
					row[headers[i]] = boolVal
				} else {
					row[headers[i]] = value
				}
			}
		}
		result = append(result, row)
	}

	return result, nil
}

func loadUpdatesFromFile(filename, format string) (map[string]map[string]interface{}, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var updates map[string]map[string]interface{}
	err = json.NewDecoder(file).Decode(&updates)
	return updates, err
}

func loadIDsFromFile(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var ids []string
	
	// Try JSON first
	if err := json.NewDecoder(file).Decode(&ids); err == nil {
		return ids, nil
	}

	// Reset file position and try line-by-line
	file.Seek(0, 0)
	
	content, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			ids = append(ids, line)
		}
	}

	return ids, nil
}

func loadMixedOperationsFromFile(filename string) (batch.MixedOperations, error) {
	file, err := os.Open(filename)
	if err != nil {
		return batch.MixedOperations{}, err
	}
	defer file.Close()

	var operations batch.MixedOperations
	err = json.NewDecoder(file).Decode(&operations)
	return operations, err
}

// Output functions
func outputBatchResult(result interface{}, operation string) error {
	// This would need to be implemented based on the actual batch result structure
	fmt.Printf("✅ Batch %s completed\n", operation)
	
	if data, err := json.MarshalIndent(result, "", "  "); err == nil {
		fmt.Println(string(data))
	} else {
		fmt.Printf("Result: %+v\n", result)
	}

	return nil
}

func outputMixedBatchResult(result interface{}) error {
	fmt.Println("✅ Mixed batch operation completed")
	
	if data, err := json.MarshalIndent(result, "", "  "); err == nil {
		fmt.Println(string(data))
	} else {
		fmt.Printf("Result: %+v\n", result)
	}

	return nil
}

func init() {
	// Create command flags
	batchCreateCmd.Flags().StringP("file", "f", "", "JSON or CSV file containing records to create")
	batchCreateCmd.Flags().StringP("data", "d", "", "JSON string containing records to create")
	batchCreateCmd.Flags().StringP("input-format", "", "json", "Input file format (json, csv)")

	// Update command flags
	batchUpdateCmd.Flags().StringP("file", "f", "", "JSON file containing updates (format: {\"sys_id\": {\"field\": \"value\"}})")
	batchUpdateCmd.Flags().StringP("data", "d", "", "JSON string containing updates")
	batchUpdateCmd.Flags().StringP("input-format", "", "json", "Input file format (json)")

	// Delete command flags
	batchDeleteCmd.Flags().StringSliceP("ids", "i", nil, "Comma-separated list of sys_ids to delete")
	batchDeleteCmd.Flags().StringP("ids-file", "f", "", "File containing sys_ids (one per line or JSON array)")
	batchDeleteCmd.Flags().BoolP("confirm", "", false, "Confirm deletion (required for safety)")

	// Mixed command flags
	batchMixedCmd.Flags().StringP("config", "c", "", "JSON configuration file for mixed operations")

	// Add subcommands
	batchCmd.AddCommand(batchCreateCmd, batchUpdateCmd, batchDeleteCmd, batchMixedCmd, batchStatusCmd)
	rootCmd.AddCommand(batchCmd)
}