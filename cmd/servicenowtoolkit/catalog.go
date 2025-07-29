package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var catalogCmd = &cobra.Command{
	Use:   "catalog",
	Short: "Service catalog operations",
	Long:  "Browse and order from ServiceNow service catalog",
}

var catalogListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available catalogs",
	Long:  "List all available service catalogs",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := createClient()
		if err != nil {
			return err
		}

		format, _ := cmd.Flags().GetString("format")

		catalogClient := client.Catalog()
		catalogs, err := catalogClient.ListCatalogs()
		if err != nil {
			return fmt.Errorf("failed to list catalogs: %w", err)
		}

		return outputCatalogs(catalogs, format)
	},
}

var catalogItemsCmd = &cobra.Command{
	Use:   "items",
	Short: "Catalog item operations",
	Long:  "Browse, search, and order catalog items",
}

var catalogItemsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List catalog items",
	Long:  "List catalog items with optional filtering",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := createClient()
		if err != nil {
			return err
		}

		catalog, _ := cmd.Flags().GetString("catalog")
		category, _ := cmd.Flags().GetString("category")
		format, _ := cmd.Flags().GetString("format")

		catalogClient := client.Catalog()

		var items interface{}
		var itemsErr error

		if catalog != "" {
			// List items in specific catalog
			items, itemsErr = catalogClient.ListItems(catalog)
		} else if category != "" {
			// List items in specific category
			items, itemsErr = catalogClient.ListItemsByCategory(category)
		} else {
			// List all items
			items, itemsErr = catalogClient.ListAllItems()
		}

		if itemsErr != nil {
			return fmt.Errorf("failed to list catalog items: %w", itemsErr)
		}

		return outputCatalogItems(items, format)
	},
}

var catalogItemsSearchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search catalog items",
	Long:  "Search for catalog items by name or description",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := createClient()
		if err != nil {
			return err
		}

		query := args[0]
		limit, _ := cmd.Flags().GetInt("limit")
		format, _ := cmd.Flags().GetString("format")

		catalogClient := client.Catalog()
		items, err := catalogClient.SearchItems(query)
		if err != nil {
			return fmt.Errorf("failed to search catalog items: %w", err)
		}

		// Apply limit if specified
		if limit > 0 && len(items) > limit {
			items = items[:limit]
		}

		return outputCatalogItems(items, format)
	},
}

var catalogItemsGetCmd = &cobra.Command{
	Use:   "get [item_id]",
	Short: "Get catalog item details",
	Long:  "Get detailed information about a specific catalog item including variables",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := createClient()
		if err != nil {
			return err
		}

		itemID := args[0]
		format, _ := cmd.Flags().GetString("format")
		showVariables, _ := cmd.Flags().GetBool("variables")

		catalogClient := client.Catalog()

		var item interface{}
		if showVariables {
			item, err = catalogClient.GetItemWithVariables(itemID)
		} else {
			item, err = catalogClient.GetItem(itemID)
		}

		if err != nil {
			return fmt.Errorf("failed to get catalog item: %w", err)
		}

		return outputCatalogItem(item, format)
	},
}

var catalogOrderCmd = &cobra.Command{
	Use:   "order [item_id]",
	Short: "Order a catalog item",
	Long:  "Place an order for a catalog item with specified variables",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := createClient()
		if err != nil {
			return err
		}

		itemID := args[0]
		quantity, _ := cmd.Flags().GetInt("quantity")
		variablesStr, _ := cmd.Flags().GetString("variables")
		variablesFile, _ := cmd.Flags().GetString("variables-file")
		dryRun, _ := cmd.Flags().GetBool("dry-run")

		var variables map[string]interface{}

		// Load variables from file or string
		if variablesFile != "" {
			file, err := os.Open(variablesFile)
			if err != nil {
				return fmt.Errorf("failed to open variables file: %w", err)
			}
			defer file.Close()

			err = json.NewDecoder(file).Decode(&variables)
			if err != nil {
				return fmt.Errorf("failed to parse variables file: %w", err)
			}
		} else if variablesStr != "" {
			err = json.Unmarshal([]byte(variablesStr), &variables)
			if err != nil {
				return fmt.Errorf("failed to parse variables: %w", err)
			}
		} else {
			variables = make(map[string]interface{})
		}

		catalogClient := client.Catalog()

		// Validate variables first
		if len(variables) > 0 {
			validationErrors, err := catalogClient.ValidateItemVariables(itemID, variables)
			if err != nil {
				return fmt.Errorf("failed to validate variables: %w", err)
			}

			if len(validationErrors) > 0 {
				fmt.Println("❌ Variable validation errors:")
				for _, valErr := range validationErrors {
					fmt.Printf("  - %s: %s\n", valErr.Variable, valErr.Message)
				}
				return fmt.Errorf("variable validation failed")
			}
		}

		// Estimate price if requested
		if dryRun {
			estimate, err := catalogClient.EstimatePrice(itemID, quantity, variables)
			if err != nil {
				return fmt.Errorf("failed to estimate price: %w", err)
			}

			fmt.Printf("💰 Price Estimate:\n")
			fmt.Printf("  Base Price: $%.2f\n", estimate.BasePrice)
			fmt.Printf("  Recurring: $%.2f\n", estimate.RecurringPrice)
			fmt.Printf("  Total (qty %d): $%.2f\n", quantity, estimate.TotalPrice)
			fmt.Printf("  Total Recurring: $%.2f\n", estimate.TotalRecurring)
			fmt.Println("\nUse --dry-run=false to place the actual order")
			return nil
		}

		// Place the order
		fmt.Printf("🛒 Placing order for item %s (quantity: %d)...\n", itemID, quantity)
		orderResult, err := catalogClient.OrderNow(itemID, quantity, variables)
		if err != nil {
			return fmt.Errorf("failed to place order: %w", err)
		}

		fmt.Printf("✅ Order placed successfully!\n")
		fmt.Printf("   Request Number: %s\n", orderResult.RequestNumber)
		fmt.Printf("   Request ID: %s\n", orderResult.RequestSysID)
		
		return nil
	},
}

var catalogCartCmd = &cobra.Command{
	Use:   "cart",
	Short: "Shopping cart operations",
	Long:  "Manage items in your ServiceNow shopping cart",
}

var catalogCartListCmd = &cobra.Command{
	Use:   "list",
	Short: "List cart contents",
	Long:  "List all items currently in your shopping cart",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := createClient()
		if err != nil {
			return err
		}

		format, _ := cmd.Flags().GetString("format")

		catalogClient := client.Catalog()
		cart, err := catalogClient.GetCart()
		if err != nil {
			return fmt.Errorf("failed to get cart: %w", err)
		}

		return outputCart(cart, format)
	},
}

var catalogCartAddCmd = &cobra.Command{
	Use:   "add [item_id]",
	Short: "Add item to cart",
	Long:  "Add a catalog item to your shopping cart",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := createClient()
		if err != nil {
			return err
		}

		itemID := args[0]
		quantity, _ := cmd.Flags().GetInt("quantity")
		variablesStr, _ := cmd.Flags().GetString("variables")

		var variables map[string]interface{}
		if variablesStr != "" {
			err = json.Unmarshal([]byte(variablesStr), &variables)
			if err != nil {
				return fmt.Errorf("failed to parse variables: %w", err)
			}
		}

		catalogClient := client.Catalog()
		response, err := catalogClient.AddToCart(itemID, quantity, variables)
		if err != nil {
			return fmt.Errorf("failed to add item to cart: %w", err)
		}

		if response.Success {
			fmt.Printf("✅ Added item %s to cart (quantity: %d)\n", itemID, quantity)
		} else {
			fmt.Printf("❌ Failed to add item to cart: %s\n", response.Message)
		}

		return nil
	},
}

var catalogCartSubmitCmd = &cobra.Command{
	Use:   "submit",
	Short: "Submit cart for ordering",
	Long:  "Submit all items in your cart as a single order",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := createClient()
		if err != nil {
			return err
		}

		catalogClient := client.Catalog()
		
		// Show cart contents first
		cart, err := catalogClient.GetCart()
		if err != nil {
			return fmt.Errorf("failed to get cart: %w", err)
		}

		fmt.Printf("📋 Cart Summary:\n")
		fmt.Printf("   Items: %d\n", len(cart.Items))
		fmt.Printf("   Total: %s\n", cart.TotalPrice)

		// Submit cart
		orderResult, err := catalogClient.SubmitCart()
		if err != nil {
			return fmt.Errorf("failed to submit cart: %w", err)
		}

		fmt.Printf("✅ Cart submitted successfully!\n")
		fmt.Printf("   Request Number: %s\n", orderResult.RequestNumber)
		
		return nil
	},
}

var catalogRequestsCmd = &cobra.Command{
	Use:   "requests",
	Short: "Catalog request operations",
	Long:  "Track and manage catalog requests",
}

var catalogRequestsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List your catalog requests",
	Long:  "List your recent catalog requests",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := createClient()
		if err != nil {
			return err
		}

		limit, _ := cmd.Flags().GetInt("limit")
		format, _ := cmd.Flags().GetString("format")

		catalogClient := client.Catalog()
		tracker := catalogClient.NewRequestTracker()

		requests, err := tracker.GetMyRequests(limit)
		if err != nil {
			return fmt.Errorf("failed to get requests: %w", err)
		}

		return outputRequests(requests, format)
	},
}

var catalogRequestsGetCmd = &cobra.Command{
	Use:   "get [request_number]",
	Short: "Get request details",
	Long:  "Get detailed information about a specific catalog request",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := createClient()
		if err != nil {
			return err
		}

		requestNumber := args[0]
		format, _ := cmd.Flags().GetString("format")

		catalogClient := client.Catalog()
		tracker := catalogClient.NewRequestTracker()

		request, err := tracker.GetRequestWithItems(requestNumber)
		if err != nil {
			return fmt.Errorf("failed to get request: %w", err)
		}

		return outputRequest(request, format)
	},
}

// Output functions (simplified implementations)
func outputCatalogs(catalogs interface{}, format string) error {
	switch format {
	case "json":
		return json.NewEncoder(os.Stdout).Encode(catalogs)
	default:
		fmt.Println("Available Catalogs:")
		if data, err := json.MarshalIndent(catalogs, "", "  "); err == nil {
			fmt.Println(string(data))
		}
		return nil
	}
}

func outputCatalogItems(items interface{}, format string) error {
	switch format {
	case "json":
		return json.NewEncoder(os.Stdout).Encode(items)
	default:
		fmt.Println("Catalog Items:")
		if data, err := json.MarshalIndent(items, "", "  "); err == nil {
			fmt.Println(string(data))
		}
		return nil
	}
}

func outputCatalogItem(item interface{}, format string) error {
	switch format {
	case "json":
		return json.NewEncoder(os.Stdout).Encode(item)
	default:
		fmt.Println("Catalog Item Details:")
		if data, err := json.MarshalIndent(item, "", "  "); err == nil {
			fmt.Println(string(data))
		}
		return nil
	}
}

func outputCart(cart interface{}, format string) error {
	switch format {
	case "json":
		return json.NewEncoder(os.Stdout).Encode(cart)
	default:
		fmt.Println("Shopping Cart:")
		if data, err := json.MarshalIndent(cart, "", "  "); err == nil {
			fmt.Println(string(data))
		}
		return nil
	}
}

func outputRequests(requests interface{}, format string) error {
	switch format {
	case "json":
		return json.NewEncoder(os.Stdout).Encode(requests)
	default:
		fmt.Println("Catalog Requests:")
		if data, err := json.MarshalIndent(requests, "", "  "); err == nil {
			fmt.Println(string(data))
		}
		return nil
	}
}

func outputRequest(request interface{}, format string) error {
	switch format {
	case "json":
		return json.NewEncoder(os.Stdout).Encode(request)
	default:
		fmt.Println("Request Details:")
		if data, err := json.MarshalIndent(request, "", "  "); err == nil {
			fmt.Println(string(data))
		}
		return nil
	}
}

func init() {
	// Catalog list flags
	catalogListCmd.Flags().StringP("format", "f", "table", "Output format (table, json)")

	// Items list flags
	catalogItemsListCmd.Flags().StringP("catalog", "c", "", "Filter by catalog ID")
	catalogItemsListCmd.Flags().StringP("category", "", "", "Filter by category ID")
	catalogItemsListCmd.Flags().IntP("limit", "l", 10, "Limit number of results")
	catalogItemsListCmd.Flags().StringP("format", "f", "table", "Output format (table, json)")

	// Items search flags
	catalogItemsSearchCmd.Flags().IntP("limit", "l", 10, "Limit number of results")
	catalogItemsSearchCmd.Flags().StringP("format", "f", "table", "Output format (table, json)")

	// Items get flags
	catalogItemsGetCmd.Flags().StringP("format", "f", "table", "Output format (table, json)")
	catalogItemsGetCmd.Flags().BoolP("variables", "v", false, "Include item variables")

	// Order flags
	catalogOrderCmd.Flags().IntP("quantity", "q", 1, "Quantity to order")
	catalogOrderCmd.Flags().StringP("variables", "v", "", "JSON string of item variables")
	catalogOrderCmd.Flags().StringP("variables-file", "f", "", "JSON file containing item variables")
	catalogOrderCmd.Flags().BoolP("dry-run", "", true, "Show price estimate only, don't place order")

	// Cart list flags
	catalogCartListCmd.Flags().StringP("format", "f", "table", "Output format (table, json)")

	// Cart add flags
	catalogCartAddCmd.Flags().IntP("quantity", "q", 1, "Quantity to add")
	catalogCartAddCmd.Flags().StringP("variables", "v", "", "JSON string of item variables")

	// Requests list flags
	catalogRequestsListCmd.Flags().IntP("limit", "l", 10, "Limit number of results")
	catalogRequestsListCmd.Flags().StringP("format", "f", "table", "Output format (table, json)")

	// Requests get flags
	catalogRequestsGetCmd.Flags().StringP("format", "f", "table", "Output format (table, json)")

	// Add subcommands
	catalogItemsCmd.AddCommand(catalogItemsListCmd, catalogItemsSearchCmd, catalogItemsGetCmd)
	catalogCartCmd.AddCommand(catalogCartListCmd, catalogCartAddCmd, catalogCartSubmitCmd)
	catalogRequestsCmd.AddCommand(catalogRequestsListCmd, catalogRequestsGetCmd)

	catalogCmd.AddCommand(catalogListCmd, catalogItemsCmd, catalogOrderCmd, catalogCartCmd, catalogRequestsCmd)
	rootCmd.AddCommand(catalogCmd)
}