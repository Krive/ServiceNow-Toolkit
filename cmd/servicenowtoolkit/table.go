package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var tableCmd = &cobra.Command{
	Use:   "table",
	Short: "ServiceNow table operations",
	Long:  "Create, read, update, and delete records in ServiceNow tables",
}

var tableListCmd = &cobra.Command{
	Use:   "list [table_name]",
	Short: "List records from a table",
	Long:  "List records from a ServiceNow table with optional filtering and field selection",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := createClient()
		if err != nil {
			return err
		}

		tableName := args[0]
		limit, _ := cmd.Flags().GetInt("limit")
		filter, _ := cmd.Flags().GetString("filter")
		fields, _ := cmd.Flags().GetString("fields")
		orderBy, _ := cmd.Flags().GetString("order-by")
		format, _ := cmd.Flags().GetString("format")

		// Build query parameters
		params := make(map[string]string)
		
		// Apply filters
		if filter != "" {
			params["sysparm_query"] = filter
		}
		
		// Apply field selection
		if fields != "" {
			params["sysparm_fields"] = fields
		}
		
		// Apply ordering
		if orderBy != "" {
			if strings.HasSuffix(orderBy, " DESC") || strings.HasSuffix(orderBy, " desc") {
				field := strings.TrimSuffix(strings.TrimSuffix(orderBy, " DESC"), " desc")
				params["sysparm_order_by"] = field
				params["sysparm_order_direction"] = "desc"
			} else {
				params["sysparm_order_by"] = orderBy
				params["sysparm_order_direction"] = "asc"
			}
		}
		
		// Apply limit
		if limit > 0 {
			params["sysparm_limit"] = strconv.Itoa(limit)
		}
		
		// Execute query
		tableClient := client.Table(tableName)
		records, err := tableClient.List(params)
		if err != nil {
			return fmt.Errorf("failed to list records: %w", err)
		}

		return outputRecords(records, format)
	},
}

var tableGetCmd = &cobra.Command{
	Use:   "get [table_name] [sys_id]",
	Short: "Get a specific record",
	Long:  "Get a specific record from a ServiceNow table by sys_id",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := createClient()
		if err != nil {
			return err
		}

		tableName := args[0]
		sysID := args[1]
		format, _ := cmd.Flags().GetString("format")

		record, err := client.Table(tableName).Get(sysID)
		if err != nil {
			return fmt.Errorf("failed to get record: %w", err)
		}

		return outputRecord(record, format)
	},
}

var tableCreateCmd = &cobra.Command{
	Use:   "create [table_name]",
	Short: "Create a new record",
	Long:  "Create a new record in a ServiceNow table",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := createClient()
		if err != nil {
			return err
		}

		tableName := args[0]
		dataStr, _ := cmd.Flags().GetString("data")
		dataFile, _ := cmd.Flags().GetString("file")
		interactive, _ := cmd.Flags().GetBool("interactive")

		var recordData map[string]interface{}

		if interactive {
			return fmt.Errorf("interactive mode not yet implemented")
		} else if dataFile != "" {
			file, err := os.Open(dataFile)
			if err != nil {
				return fmt.Errorf("failed to open data file: %w", err)
			}
			defer file.Close()

			err = json.NewDecoder(file).Decode(&recordData)
			if err != nil {
				return fmt.Errorf("failed to parse data file: %w", err)
			}
		} else if dataStr != "" {
			err = json.Unmarshal([]byte(dataStr), &recordData)
			if err != nil {
				return fmt.Errorf("failed to parse data: %w", err)
			}
		} else {
			return fmt.Errorf("either --data, --file, or --interactive must be provided")
		}

		record, err := client.Table(tableName).Create(recordData)
		if err != nil {
			return fmt.Errorf("failed to create record: %w", err)
		}

		fmt.Printf("✅ Created record: %s\n", record["sys_id"])
		return nil
	},
}

var tableUpdateCmd = &cobra.Command{
	Use:   "update [table_name] [sys_id]",
	Short: "Update a record",
	Long:  "Update an existing record in a ServiceNow table",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := createClient()
		if err != nil {
			return err
		}

		tableName := args[0]
		sysID := args[1]
		dataStr, _ := cmd.Flags().GetString("data")
		dataFile, _ := cmd.Flags().GetString("file")

		var updates map[string]interface{}

		if dataFile != "" {
			file, err := os.Open(dataFile)
			if err != nil {
				return fmt.Errorf("failed to open data file: %w", err)
			}
			defer file.Close()

			err = json.NewDecoder(file).Decode(&updates)
			if err != nil {
				return fmt.Errorf("failed to parse data file: %w", err)
			}
		} else if dataStr != "" {
			err = json.Unmarshal([]byte(dataStr), &updates)
			if err != nil {
				return fmt.Errorf("failed to parse data: %w", err)
			}
		} else {
			return fmt.Errorf("either --data or --file must be provided")
		}

		record, err := client.Table(tableName).Update(sysID, updates)
		if err != nil {
			return fmt.Errorf("failed to update record: %w", err)
		}

		fmt.Printf("✅ Updated record: %s\n", record["sys_id"])
		return nil
	},
}

var tableDeleteCmd = &cobra.Command{
	Use:   "delete [table_name] [sys_id]",
	Short: "Delete a record",
	Long:  "Delete a record from a ServiceNow table",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := createClient()
		if err != nil {
			return err
		}

		tableName := args[0]
		sysID := args[1]
		confirm, _ := cmd.Flags().GetBool("confirm")

		if !confirm {
			fmt.Printf("⚠️  This will permanently delete record %s from table %s. Use --confirm to proceed.\n", sysID, tableName)
			return nil
		}

		err = client.Table(tableName).Delete(sysID)
		if err != nil {
			return fmt.Errorf("failed to delete record: %w", err)
		}

		fmt.Printf("✅ Deleted record: %s\n", sysID)
		return nil
	},
}

// Output functions
func outputRecords(records []map[string]interface{}, format string) error {
	switch format {
	case "json":
		return json.NewEncoder(os.Stdout).Encode(records)
	case "csv":
		return outputRecordsCSV(records)
	case "table":
		fallthrough
	default:
		return outputRecordsTable(records)
	}
}

func outputRecord(record map[string]interface{}, format string) error {
	switch format {
	case "json":
		return json.NewEncoder(os.Stdout).Encode(record)
	default:
		fmt.Println("Record Details:")
		for key, value := range record {
			fmt.Printf("  %s: %v\n", key, value)
		}
		return nil
	}
}

func outputRecordsTable(records []map[string]interface{}) error {
	if len(records) == 0 {
		fmt.Println("No records found.")
		return nil
	}

	// Get all unique keys for headers
	keys := make(map[string]bool)
	for _, record := range records {
		for key := range record {
			keys[key] = true
		}
	}

	// Convert to sorted slice
	var headers []string
	for key := range keys {
		headers = append(headers, key)
	}

	// Create table writer
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	// Print headers
	fmt.Fprintln(w, strings.Join(headers, "\t"))

	// Print records
	for _, record := range records {
		var values []string
		for _, header := range headers {
			value := ""
			if v, exists := record[header]; exists && v != nil {
				value = fmt.Sprintf("%v", v)
				// Truncate long values
				if len(value) > 50 {
					value = value[:47] + "..."
				}
			}
			values = append(values, value)
		}
		fmt.Fprintln(w, strings.Join(values, "\t"))
	}

	return w.Flush()
}

func outputRecordsCSV(records []map[string]interface{}) error {
	if len(records) == 0 {
		return nil
	}

	// Get all unique keys for headers
	keys := make(map[string]bool)
	for _, record := range records {
		for key := range record {
			keys[key] = true
		}
	}

	// Convert to sorted slice
	var headers []string
	for key := range keys {
		headers = append(headers, key)
	}

	// Print CSV headers
	fmt.Println(strings.Join(headers, ","))

	// Print CSV records
	for _, record := range records {
		var values []string
		for _, header := range headers {
			value := ""
			if v, exists := record[header]; exists && v != nil {
				value = fmt.Sprintf("%v", v)
				// Escape commas and quotes
				if strings.Contains(value, ",") || strings.Contains(value, "\"") {
					value = "\"" + strings.ReplaceAll(value, "\"", "\"\"") + "\""
				}
			}
			values = append(values, value)
		}
		fmt.Println(strings.Join(values, ","))
	}

	return nil
}

func init() {
	// List command flags
	tableListCmd.Flags().IntP("limit", "l", 10, "Limit number of results")
	tableListCmd.Flags().StringP("filter", "f", "", "Filter criteria (field=value^field2=value2)")
	tableListCmd.Flags().StringP("fields", "", "", "Comma-separated list of fields to include")
	tableListCmd.Flags().StringP("order-by", "o", "", "Order by field (append ' DESC' for descending)")
	tableListCmd.Flags().StringP("format", "", "table", "Output format (table, json, csv)")

	// Get command flags
	tableGetCmd.Flags().StringP("format", "", "table", "Output format (table, json)")

	// Create command flags
	tableCreateCmd.Flags().StringP("data", "d", "", "JSON string of record data")
	tableCreateCmd.Flags().StringP("file", "", "", "JSON file containing record data")
	tableCreateCmd.Flags().BoolP("interactive", "i", false, "Interactive mode (prompt for fields)")

	// Update command flags
	tableUpdateCmd.Flags().StringP("data", "d", "", "JSON string of updates")
	tableUpdateCmd.Flags().StringP("file", "", "", "JSON file containing updates")

	// Delete command flags
	tableDeleteCmd.Flags().BoolP("confirm", "", false, "Confirm deletion (required for safety)")

	// Add subcommands
	tableCmd.AddCommand(tableListCmd, tableGetCmd, tableCreateCmd, tableUpdateCmd, tableDeleteCmd)
	rootCmd.AddCommand(tableCmd)
}