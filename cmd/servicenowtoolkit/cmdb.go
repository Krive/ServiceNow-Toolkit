package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/Krive/ServiceNow-Toolkit/pkg/servicenow/cmdb"
)

var cmdbCmd = &cobra.Command{
	Use:   "cmdb",
	Short: "Configuration Management Database operations",
	Long:  "Manage Configuration Items (CIs), relationships, and CMDB classes",
}

// CI commands
var ciCmd = &cobra.Command{
	Use:   "ci",
	Short: "Configuration Item operations",
	Long:  "Create, read, update, and delete Configuration Items",
}

var ciListCmd = &cobra.Command{
	Use:   "list",
	Short: "List Configuration Items",
	Long:  "List Configuration Items with optional filtering by class and other criteria",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := createClient()
		if err != nil {
			return err
		}

		class, _ := cmd.Flags().GetString("class")
		name, _ := cmd.Flags().GetString("name")
		active, _ := cmd.Flags().GetBool("active")
		limit, _ := cmd.Flags().GetInt("limit")
		format, _ := cmd.Flags().GetString("format")
		fields, _ := cmd.Flags().GetString("fields")

		cmdbClient := client.CMDB()

		// Create filter
		filter := &cmdb.CIFilter{
			Limit: limit,
		}

		if class != "" {
			filter.Class = class
		}
		if name != "" {
			filter.Name = name
		}
		if cmd.Flags().Changed("active") {
			if active {
				filter.State = "1"
			} else {
				filter.State = "6"
			}
		}
		if fields != "" {
			filter.Fields = strings.Split(fields, ",")
		}

		// Get CIs
		cis, err := cmdbClient.ListCIs(filter)
		if err != nil {
			return fmt.Errorf("failed to list CIs: %w", err)
		}

		return outputCIs(cis, format)
	},
}

var ciGetCmd = &cobra.Command{
	Use:   "get [ci_id]",
	Short: "Get a specific Configuration Item",
	Long:  "Get detailed information about a specific Configuration Item",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := createClient()
		if err != nil {
			return err
		}

		ciID := args[0]
		format, _ := cmd.Flags().GetString("format")
		includeRelated, _ := cmd.Flags().GetBool("relationships")

		cmdbClient := client.CMDB()
		ci, err := cmdbClient.GetCI(ciID)
		if err != nil {
			return fmt.Errorf("failed to get CI: %w", err)
		}

		// Get relationships if requested
		var relationships interface{}
		if includeRelated {
			relClient := cmdbClient.NewRelationshipClient()
			relationships, err = relClient.GetRelationships(ciID)
			if err != nil {
				fmt.Printf("Warning: failed to get relationships: %v\n", err)
			}
		}

		return outputCIDetails(ci, relationships, format)
	},
}

var ciCreateCmd = &cobra.Command{
	Use:   "create [class]",
	Short: "Create a new Configuration Item",
	Long:  "Create a new Configuration Item of the specified class",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := createClient()
		if err != nil {
			return err
		}

		class := args[0]
		name, _ := cmd.Flags().GetString("name")
		dataStr, _ := cmd.Flags().GetString("data")
		dataFile, _ := cmd.Flags().GetString("file")

		var ciData map[string]interface{}

		if dataFile != "" {
			file, err := os.Open(dataFile)
			if err != nil {
				return fmt.Errorf("failed to open data file: %w", err)
			}
			defer file.Close()

			err = json.NewDecoder(file).Decode(&ciData)
			if err != nil {
				return fmt.Errorf("failed to parse data file: %w", err)
			}
		} else if dataStr != "" {
			err = json.Unmarshal([]byte(dataStr), &ciData)
			if err != nil {
				return fmt.Errorf("failed to parse data: %w", err)
			}
		} else {
			ciData = make(map[string]interface{})
		}

		// Set name if provided
		if name != "" {
			ciData["name"] = name
		}

		cmdbClient := client.CMDB()
		ci, err := cmdbClient.CreateCI(class, ciData)
		if err != nil {
			return fmt.Errorf("failed to create CI: %w", err)
		}

		fmt.Printf("✅ Created CI: %s (%s)\n", ci.Name, ci.SysID)
		return nil
	},
}

var ciUpdateCmd = &cobra.Command{
	Use:   "update [ci_id]",
	Short: "Update a Configuration Item",
	Long:  "Update an existing Configuration Item with new data",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := createClient()
		if err != nil {
			return err
		}

		ciID := args[0]
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

		cmdbClient := client.CMDB()
		// We need to get the CI first to determine its class
		ci, err := cmdbClient.GetCI(ciID)
		if err != nil {
			return fmt.Errorf("failed to get CI for update: %w", err)
		}
		ci, err = cmdbClient.UpdateCI(ci.SysClassName, ciID, updates)
		if err != nil {
			return fmt.Errorf("failed to update CI: %w", err)
		}

		fmt.Printf("✅ Updated CI: %s (%s)\n", ci.Name, ci.SysID)
		return nil
	},
}

// Relationship commands
var relationshipCmd = &cobra.Command{
	Use:   "relationship",
	Short: "CI relationship operations",
	Long:  "Manage relationships between Configuration Items",
}

var relationshipListCmd = &cobra.Command{
	Use:   "list [ci_id]",
	Short: "List CI relationships",
	Long:  "List all relationships for a Configuration Item",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := createClient()
		if err != nil {
			return err
		}

		ciID := args[0]
		relType, _ := cmd.Flags().GetString("type")
		format, _ := cmd.Flags().GetString("format")

		cmdbClient := client.CMDB()

		relClient := cmdbClient.NewRelationshipClient()
		
		var relationships interface{}
		if relType != "" {
			relationships, err = relClient.GetRelationshipsByType(relType)
		} else {
			relationships, err = relClient.GetRelationships(ciID)
		}

		if err != nil {
			return fmt.Errorf("failed to get relationships: %w", err)
		}

		return outputRelationships(relationships, format)
	},
}

var relationshipCreateCmd = &cobra.Command{
	Use:   "create [parent_ci_id] [child_ci_id] [relationship_type]",
	Short: "Create a relationship between CIs",
	Long:  "Create a relationship between two Configuration Items",
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := createClient()
		if err != nil {
			return err
		}

		parentID := args[0]
		childID := args[1]
		relType := args[2]

		cmdbClient := client.CMDB()
		relClient := cmdbClient.NewRelationshipClient()
		relationship, err := relClient.CreateRelationship(parentID, childID, relType)
		if err != nil {
			return fmt.Errorf("failed to create relationship: %w", err)
		}

		fmt.Printf("✅ Created relationship: %s -> %s (%s)\n", parentID, childID, relType)
		fmt.Printf("   Relationship ID: %s\n", relationship.SysID)
		return nil
	},
}

var relationshipParentsCmd = &cobra.Command{
	Use:   "parents [ci_id]",
	Short: "List parent CIs",
	Long:  "List all parent Configuration Items for a given CI",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := createClient()
		if err != nil {
			return err
		}

		ciID := args[0]
		format, _ := cmd.Flags().GetString("format")

		cmdbClient := client.CMDB()
		relClient := cmdbClient.NewRelationshipClient()
		parents, err := relClient.GetParentRelationships(ciID)
		if err != nil {
			return fmt.Errorf("failed to get parent CIs: %w", err)
		}

		return outputRelationships(parents, format)
	},
}

var relationshipChildrenCmd = &cobra.Command{
	Use:   "children [ci_id]",
	Short: "List child CIs",
	Long:  "List all child Configuration Items for a given CI",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := createClient()
		if err != nil {
			return err
		}

		ciID := args[0]
		format, _ := cmd.Flags().GetString("format")

		cmdbClient := client.CMDB()
		relClient := cmdbClient.NewRelationshipClient()
		children, err := relClient.GetChildRelationships(ciID)
		if err != nil {
			return fmt.Errorf("failed to get child CIs: %w", err)
		}

		return outputRelationships(children, format)
	},
}

// Class commands
var classCmd = &cobra.Command{
	Use:   "class",
	Short: "CMDB class operations",
	Long:  "Manage Configuration Item classes and hierarchies",
}

var classListCmd = &cobra.Command{
	Use:   "list",
	Short: "List CMDB classes",
	Long:  "List all available Configuration Item classes",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := createClient()
		if err != nil {
			return err
		}

		format, _ := cmd.Flags().GetString("format")

		cmdbClient := client.CMDB()
		classClient := cmdbClient.NewClassClient()
		classes, err := classClient.ListCIClasses()
		if err != nil {
			return fmt.Errorf("failed to list CI classes: %w", err)
		}

		return outputClasses(classes, format)
	},
}

var classGetCmd = &cobra.Command{
	Use:   "get [class_name]",
	Short: "Get class details",
	Long:  "Get detailed information about a specific CI class",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := createClient()
		if err != nil {
			return err
		}

		className := args[0]
		format, _ := cmd.Flags().GetString("format")

		cmdbClient := client.CMDB()
		classClient := cmdbClient.NewClassClient()
		class, err := classClient.GetCIClass(className)
		if err != nil {
			return fmt.Errorf("failed to get CI class: %w", err)
		}

		return outputClass(class, format)
	},
}

var classHierarchyCmd = &cobra.Command{
	Use:   "hierarchy [class_name]",
	Short: "Show class hierarchy",
	Long:  "Show the class hierarchy for a specific CI class",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := createClient()
		if err != nil {
			return err
		}

		className := args[0]
		format, _ := cmd.Flags().GetString("format")

		cmdbClient := client.CMDB()
		classClient := cmdbClient.NewClassClient()
		hierarchy, err := classClient.GetClassHierarchy(className)
		if err != nil {
			return fmt.Errorf("failed to get class hierarchy: %w", err)
		}

		return outputClassHierarchy(hierarchy, format)
	},
}

// Output functions
func outputCIs(cis []*cmdb.ConfigurationItem, format string) error {
	switch format {
	case "json":
		return json.NewEncoder(os.Stdout).Encode(cis)
	default:
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "SYS_ID\tNAME\tCLASS\tSTATE\tIP_ADDRESS\tSERIAL_NUMBER")
		for _, ci := range cis {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
				ci.SysID,
				ci.Name,
				ci.SysClassName,
				ci.State,
				ci.IPAddress,
				ci.SerialNumber,
			)
		}
		return w.Flush()
	}
}

func outputCIDetails(ci *cmdb.ConfigurationItem, relationships interface{}, format string) error {
	switch format {
	case "json":
		result := map[string]interface{}{
			"ci":            ci,
			"relationships": relationships,
		}
		return json.NewEncoder(os.Stdout).Encode(result)
	default:
		fmt.Printf("Configuration Item Details:\n")
		fmt.Printf("  SysID:         %s\n", ci.SysID)
		fmt.Printf("  Name:          %s\n", ci.Name)
		fmt.Printf("  Class:         %s\n", ci.SysClassName)
		fmt.Printf("  State:         %s\n", ci.State)
		fmt.Printf("  IP Address:    %s\n", ci.IPAddress)
		fmt.Printf("  Serial Number: %s\n", ci.SerialNumber)
		fmt.Printf("  Location:      %s\n", ci.Location)
		fmt.Printf("  Owner:         %s\n", ci.Owner)
		fmt.Printf("  Created:       %s\n", ci.CreatedOn.Format("2006-01-02 15:04:05"))

		if relationships != nil {
			fmt.Printf("\nRelationships:\n")
			if data, err := json.MarshalIndent(relationships, "  ", "  "); err == nil {
				fmt.Printf("  %s\n", string(data))
			}
		}
		return nil
	}
}

func outputRelationships(relationships interface{}, format string) error {
	switch format {
	case "json":
		return json.NewEncoder(os.Stdout).Encode(relationships)
	default:
		fmt.Println("CI Relationships:")
		if data, err := json.MarshalIndent(relationships, "", "  "); err == nil {
			fmt.Println(string(data))
		}
		return nil
	}
}

func outputClasses(classes interface{}, format string) error {
	switch format {
	case "json":
		return json.NewEncoder(os.Stdout).Encode(classes)
	default:
		fmt.Println("CMDB Classes:")
		if data, err := json.MarshalIndent(classes, "", "  "); err == nil {
			fmt.Println(string(data))
		}
		return nil
	}
}

func outputClass(class interface{}, format string) error {
	switch format {
	case "json":
		return json.NewEncoder(os.Stdout).Encode(class)
	default:
		fmt.Println("Class Details:")
		if data, err := json.MarshalIndent(class, "", "  "); err == nil {
			fmt.Println(string(data))
		}
		return nil
	}
}

func outputClassHierarchy(hierarchy interface{}, format string) error {
	switch format {
	case "json":
		return json.NewEncoder(os.Stdout).Encode(hierarchy)
	default:
		fmt.Println("Class Hierarchy:")
		if data, err := json.MarshalIndent(hierarchy, "", "  "); err == nil {
			fmt.Println(string(data))
		}
		return nil
	}
}

func init() {
	// CI list flags
	ciListCmd.Flags().StringP("class", "c", "", "Filter by CI class")
	ciListCmd.Flags().StringP("name", "n", "", "Filter by name")
	ciListCmd.Flags().BoolP("active", "a", false, "Filter by active status")
	ciListCmd.Flags().IntP("limit", "l", 10, "Limit number of results")
	ciListCmd.Flags().StringP("format", "f", "table", "Output format (table, json)")
	ciListCmd.Flags().StringP("fields", "", "", "Comma-separated list of fields to include")

	// CI get flags
	ciGetCmd.Flags().StringP("format", "f", "table", "Output format (table, json)")
	ciGetCmd.Flags().BoolP("relationships", "r", false, "Include relationships")

	// CI create flags
	ciCreateCmd.Flags().StringP("name", "n", "", "CI name")
	ciCreateCmd.Flags().StringP("data", "d", "", "JSON string of CI data")
	ciCreateCmd.Flags().StringP("file", "", "", "JSON file containing CI data")

	// CI update flags
	ciUpdateCmd.Flags().StringP("data", "d", "", "JSON string of updates")
	ciUpdateCmd.Flags().StringP("file", "", "", "JSON file containing updates")

	// Relationship list flags
	relationshipListCmd.Flags().StringP("type", "t", "", "Filter by relationship type")
	relationshipListCmd.Flags().StringP("format", "f", "table", "Output format (table, json)")

	// Relationship parents/children flags
	relationshipParentsCmd.Flags().StringP("format", "f", "table", "Output format (table, json)")
	relationshipChildrenCmd.Flags().StringP("format", "f", "table", "Output format (table, json)")

	// Class flags
	classListCmd.Flags().StringP("format", "f", "table", "Output format (table, json)")
	classGetCmd.Flags().StringP("format", "f", "table", "Output format (table, json)")
	classHierarchyCmd.Flags().StringP("format", "f", "table", "Output format (table, json)")

	// Add subcommands
	ciCmd.AddCommand(ciListCmd, ciGetCmd, ciCreateCmd, ciUpdateCmd)
	relationshipCmd.AddCommand(relationshipListCmd, relationshipCreateCmd, relationshipParentsCmd, relationshipChildrenCmd)
	classCmd.AddCommand(classListCmd, classGetCmd, classHierarchyCmd)

	cmdbCmd.AddCommand(ciCmd, relationshipCmd, classCmd)
	rootCmd.AddCommand(cmdbCmd)
}