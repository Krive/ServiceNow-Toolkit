package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/Krive/ServiceNow-Toolkit/pkg/servicenow"
	"github.com/Krive/ServiceNow-Toolkit/pkg/servicenow/catalog"
)

func main() {
	// Initialize ServiceNow client
	client, err := servicenow.NewClientBasicAuth(
		os.Getenv("SN_INSTANCE_URL"),
		os.Getenv("SN_USERNAME"),
		os.Getenv("SN_PASSWORD"),
	)
	if err != nil {
		log.Fatalf("Failed to create ServiceNow client: %v", err)
	}

	// Run examples
	fmt.Println("=== ServiceNow Service Catalog API Examples ===\n")

	browseCatalogsExample(client)
	browseCategoriesAndItemsExample(client)
	searchCatalogExample(client)
	itemDetailsAndVariablesExample(client)
	validateVariablesExample(client)
	priceEstimationExample(client)
	requestTrackingExample(client)
	contextTimeoutExample(client)
}

// Example 1: Browse available catalogs
func browseCatalogsExample(client *servicenow.Client) {
	fmt.Println("1. Browse Available Catalogs")
	fmt.Println("----------------------------")

	catalogClient := client.Catalog()

	// List all available catalogs
	catalogs, err := catalogClient.ListCatalogs()
	if err != nil {
		log.Printf("Error listing catalogs: %v", err)
		return
	}

	fmt.Printf("Found %d catalogs:\n", len(catalogs))
	for i, cat := range catalogs {
		if i >= 5 { // Show only first 5
			fmt.Printf("  ... and %d more\n", len(catalogs)-5)
			break
		}
		fmt.Printf("  %d. %s (%s)\n", i+1, cat.Title, cat.SysID)
		if cat.Description != "" {
			fmt.Printf("     Description: %s\n", cat.Description)
		}
	}

	// Get details of first catalog
	if len(catalogs) > 0 {
		catalog, err := catalogClient.GetCatalog(catalogs[0].SysID)
		if err != nil {
			log.Printf("Error getting catalog details: %v", err)
		} else {
			fmt.Printf("\nFirst catalog details:\n")
			fmt.Printf("  Title: %s\n", catalog.Title)
			fmt.Printf("  Active: %t\n", catalog.Active)
			fmt.Printf("  Background: %s\n", catalog.Background)
		}
	}
	fmt.Println()
}

// Example 2: Browse categories and items
func browseCategoriesAndItemsExample(client *servicenow.Client) {
	fmt.Println("2. Browse Categories and Items")
	fmt.Println("------------------------------")

	catalogClient := client.Catalog()

	// Get all categories
	categories, err := catalogClient.ListAllCategories()
	if err != nil {
		log.Printf("Error listing categories: %v", err)
		return
	}

	fmt.Printf("Found %d categories across all catalogs:\n", len(categories))
	for i, category := range categories {
		if i >= 3 { // Show only first 3
			fmt.Printf("  ... and %d more\n", len(categories)-3)
			break
		}
		fmt.Printf("  %d. %s (Order: %d)\n", i+1, category.Title, category.Order)
	}

	// Get items from first category
	if len(categories) > 0 {
		category := categories[0]
		items, err := catalogClient.ListItemsByCategory(category.SysID)
		if err != nil {
			log.Printf("Error listing items in category: %v", err)
		} else {
			fmt.Printf("\nItems in category '%s' (%d items):\n", category.Title, len(items))
			for i, item := range items {
				if i >= 5 { // Show only first 5
					fmt.Printf("  ... and %d more\n", len(items)-5)
					break
				}
				fmt.Printf("  %d. %s\n", i+1, item.Name)
				fmt.Printf("     Price: %s", item.Price)
				if item.RecurringPrice != "" {
					fmt.Printf(" (Recurring: %s)", item.RecurringPrice)
				}
				fmt.Println()
			}
		}
	}

	// Get order guides
	orderGuides, err := catalogClient.GetOrderGuides()
	if err != nil {
		log.Printf("Error getting order guides: %v", err)
	} else {
		fmt.Printf("\nFound %d order guides:\n", len(orderGuides))
		for i, guide := range orderGuides {
			if i >= 3 { // Show only first 3
				fmt.Printf("  ... and %d more\n", len(orderGuides)-3)
				break
			}
			fmt.Printf("  %d. %s\n", i+1, guide.Name)
		}
	}
	fmt.Println()
}

// Example 3: Search catalog items and categories
func searchCatalogExample(client *servicenow.Client) {
	fmt.Println("3. Search Catalog")
	fmt.Println("-----------------")

	catalogClient := client.Catalog()

	// Search for items
	searchTerms := []string{"laptop", "software", "phone", "license"}
	
	for _, term := range searchTerms {
		items, err := catalogClient.SearchItems(term)
		if err != nil {
			log.Printf("Error searching for '%s': %v", term, err)
			continue
		}

		fmt.Printf("Search for '%s' found %d items:\n", term, len(items))
		for i, item := range items {
			if i >= 3 { // Show only first 3 results
				fmt.Printf("  ... and %d more\n", len(items)-3)
				break
			}
			fmt.Printf("  %d. %s - %s\n", i+1, item.Name, item.Price)
		}
		fmt.Println()
	}

	// Search categories
	categories, err := catalogClient.SearchCategories("software")
	if err != nil {
		log.Printf("Error searching categories: %v", err)
	} else {
		fmt.Printf("Categories matching 'software' (%d found):\n", len(categories))
		for i, category := range categories {
			if i >= 3 {
				fmt.Printf("  ... and %d more\n", len(categories)-3)
				break
			}
			fmt.Printf("  %d. %s\n", i+1, category.Title)
		}
	}
	fmt.Println()
}

// Example 4: Get item details and variables
func itemDetailsAndVariablesExample(client *servicenow.Client) {
	fmt.Println("4. Item Details and Variables")
	fmt.Println("-----------------------------")

	catalogClient := client.Catalog()

	// Get all items and find one with variables
	allItems, err := catalogClient.ListAllItems()
	if err != nil {
		log.Printf("Error listing items: %v", err)
		return
	}

	var itemWithVariables *catalog.CatalogItem
	for _, item := range allItems {
		itemDetail, err := catalogClient.GetItemWithVariables(item.SysID)
		if err != nil {
			continue
		}
		if len(itemDetail.Variables) > 0 {
			itemWithVariables = itemDetail
			break
		}
	}

	if itemWithVariables == nil {
		fmt.Println("No items with variables found")
		return
	}

	fmt.Printf("Item: %s\n", itemWithVariables.Name)
	fmt.Printf("Description: %s\n", itemWithVariables.ShortDescription)
	fmt.Printf("Price: %s\n", itemWithVariables.Price)
	if itemWithVariables.RecurringPrice != "" {
		fmt.Printf("Recurring Price: %s\n", itemWithVariables.RecurringPrice)
	}
	fmt.Printf("Order Guide: %t\n", itemWithVariables.OrderGuide)

	fmt.Printf("\nVariables (%d total):\n", len(itemWithVariables.Variables))
	for i, variable := range itemWithVariables.Variables {
		if i >= 5 { // Show only first 5 variables
			fmt.Printf("  ... and %d more variables\n", len(itemWithVariables.Variables)-5)
			break
		}

		fmt.Printf("  %d. %s (%s)\n", i+1, variable.Question, variable.Type)
		fmt.Printf("     Name: %s\n", variable.Name)
		fmt.Printf("     Mandatory: %t\n", variable.Mandatory)
		
		if variable.DefaultValue != "" {
			fmt.Printf("     Default: %s\n", variable.DefaultValue)
		}
		
		if variable.HelpText != "" {
			fmt.Printf("     Help: %s\n", variable.HelpText)
		}

		// Show choices for choice variables
		if len(variable.Choices) > 0 {
			fmt.Printf("     Choices:\n")
			for j, choice := range variable.Choices {
				if j >= 3 { // Show only first 3 choices
					fmt.Printf("       ... and %d more choices\n", len(variable.Choices)-3)
					break
				}
				fmt.Printf("       - %s: %s\n", choice.Value, choice.Text)
			}
		}
		fmt.Println()
	}
}

// Example 5: Validate variables before ordering
func validateVariablesExample(client *servicenow.Client) {
	fmt.Println("5. Variable Validation")
	fmt.Println("----------------------")

	catalogClient := client.Catalog()

	// Find an item with variables for testing
	allItems, err := catalogClient.ListAllItems()
	if err != nil {
		log.Printf("Error listing items: %v", err)
		return
	}

	var testItem *catalog.CatalogItem
	for _, item := range allItems {
		itemDetail, err := catalogClient.GetItemWithVariables(item.SysID)
		if err != nil {
			continue
		}
		if len(itemDetail.Variables) > 0 {
			testItem = itemDetail
			break
		}
	}

	if testItem == nil {
		fmt.Println("No items with variables found for validation testing")
		return
	}

	fmt.Printf("Testing variable validation for: %s\n", testItem.Name)

	// Test 1: Validate with empty variables (should show mandatory field errors)
	fmt.Println("\nValidation Test 1: Empty variables")
	validationErrors, err := catalogClient.ValidateItemVariables(testItem.SysID, map[string]interface{}{})
	if err != nil {
		log.Printf("Validation error: %v", err)
	} else {
		fmt.Printf("Found %d validation errors:\n", len(validationErrors))
		for i, valErr := range validationErrors {
			if i >= 3 {
				fmt.Printf("  ... and %d more errors\n", len(validationErrors)-3)
				break
			}
			fmt.Printf("  %d. %s: %s (%s)\n", i+1, valErr.Variable, valErr.Message, valErr.Type)
		}
	}

	// Test 2: Validate with some valid variables
	fmt.Println("\nValidation Test 2: With valid variables")
	validVariables := make(map[string]interface{})
	
	// Fill in some mandatory variables with valid values
	for _, variable := range testItem.Variables {
		if variable.Mandatory {
			if variable.Type == "choice" && len(variable.Choices) > 0 {
				validVariables[variable.Name] = variable.Choices[0].Value
			} else if variable.DefaultValue != "" {
				validVariables[variable.Name] = variable.DefaultValue
			} else {
				// Provide sample values based on type
				switch variable.Type {
				case "string", "text":
					validVariables[variable.Name] = "test value"
				case "boolean":
					validVariables[variable.Name] = "true"
				case "integer", "numeric":
					validVariables[variable.Name] = "1"
				default:
					validVariables[variable.Name] = "sample"
				}
			}
		}
	}

	fmt.Printf("Testing with %d variables:\n", len(validVariables))
	for name, value := range validVariables {
		fmt.Printf("  %s = %v\n", name, value)
	}

	validationErrors, err = catalogClient.ValidateItemVariables(testItem.SysID, validVariables)
	if err != nil {
		log.Printf("Validation error: %v", err)
	} else {
		fmt.Printf("Found %d validation errors with valid variables\n", len(validationErrors))
		for _, valErr := range validationErrors {
			fmt.Printf("  - %s: %s\n", valErr.Variable, valErr.Message)
		}
	}
	fmt.Println()
}

// Example 6: Price estimation
func priceEstimationExample(client *servicenow.Client) {
	fmt.Println("6. Price Estimation")
	fmt.Println("-------------------")

	catalogClient := client.Catalog()

	// Get items for price estimation
	allItems, err := catalogClient.ListAllItems()
	if err != nil {
		log.Printf("Error listing items: %v", err)
		return
	}

	if len(allItems) == 0 {
		fmt.Println("No items available for price estimation")
		return
	}

	// Test price estimation for different quantities
	testItem := allItems[0]
	fmt.Printf("Price estimation for: %s\n", testItem.Name)

	quantities := []int{1, 2, 5, 10}
	for _, qty := range quantities {
		estimate, err := catalogClient.EstimatePrice(testItem.SysID, qty, map[string]interface{}{
			"service_level": "standard",
			"priority":     "normal",
		})
		
		if err != nil {
			log.Printf("Error estimating price for quantity %d: %v", qty, err)
			continue
		}

		fmt.Printf("\nQuantity %d:\n", qty)
		fmt.Printf("  Base Price: $%.2f each\n", estimate.BasePrice)
		fmt.Printf("  Total Price: $%.2f\n", estimate.TotalPrice)
		
		if estimate.RecurringPrice > 0 {
			fmt.Printf("  Recurring Price: $%.2f each\n", estimate.RecurringPrice)
			fmt.Printf("  Total Recurring: $%.2f\n", estimate.TotalRecurring)
		}
		
		fmt.Printf("  Currency: %s\n", estimate.Currency)
	}
	fmt.Println()
}

// Example 7: Request tracking
func requestTrackingExample(client *servicenow.Client) {
	fmt.Println("7. Request Tracking")
	fmt.Println("-------------------")

	catalogClient := client.Catalog()
	tracker := catalogClient.NewRequestTracker()

	// Get user's recent requests
	myRequests, err := tracker.GetMyRequests(10)
	if err != nil {
		log.Printf("Error getting user requests: %v", err)
		return
	}

	fmt.Printf("Found %d recent requests:\n", len(myRequests))
	
	if len(myRequests) == 0 {
		fmt.Println("No requests found for current user")
		return
	}

	for i, request := range myRequests {
		if i >= 5 { // Show only first 5
			fmt.Printf("  ... and %d more requests\n", len(myRequests)-5)
			break
		}

		fmt.Printf("  %d. %s - %s\n", i+1, request.Number, request.ShortDescription)
		fmt.Printf("     State: %s, Stage: %s\n", request.State, request.Stage)
		fmt.Printf("     Opened: %s\n", request.OpenedAt.Format("2006-01-02 15:04"))
		
		if request.Price != "" {
			fmt.Printf("     Price: %s\n", request.Price)
		}
		fmt.Println()
	}

	// Get detailed information for the first request
	if len(myRequests) > 0 {
		firstRequest := myRequests[0]
		fmt.Printf("Detailed tracking for request %s:\n", firstRequest.Number)

		// Get request with items
		requestWithItems, err := tracker.GetRequestWithItems(firstRequest.Number)
		if err != nil {
			log.Printf("Error getting request items: %v", err)
		} else {
			fmt.Printf("  Request has %d items:\n", len(requestWithItems.RequestItems))
			
			for i, item := range requestWithItems.RequestItems {
				if i >= 3 {
					fmt.Printf("    ... and %d more items\n", len(requestWithItems.RequestItems)-3)
					break
				}

				fmt.Printf("    Item %d: %s\n", i+1, item.Number)
				fmt.Printf("      State: %s, Stage: %s\n", item.State, item.Stage)
				fmt.Printf("      Quantity: %d\n", item.Quantity)
				
				if item.Price != "" {
					fmt.Printf("      Price: %s\n", item.Price)
				}

				// Get tasks for this item
				tasks, err := tracker.GetRequestItemTasks(item.Number)
				if err != nil {
					log.Printf("      Error getting tasks: %v", err)
				} else {
					fmt.Printf("      Tasks: %d\n", len(tasks))
					for j, task := range tasks {
						if j >= 2 {
							fmt.Printf("        ... and %d more tasks\n", len(tasks)-2)
							break
						}
						fmt.Printf("        %s: %s (%s)\n", task.Number, task.ShortDescription, task.State)
					}
				}
				fmt.Println()
			}
		}
	}

	// Get requests by state
	fmt.Println("Requests by state:")
	states := []string{"in_process", "pending_approval", "closed_complete"}
	
	for _, state := range states {
		requests, err := tracker.GetRequestsByState(state)
		if err != nil {
			log.Printf("Error getting requests in state %s: %v", state, err)
			continue
		}
		fmt.Printf("  %s: %d requests\n", state, len(requests))
	}
	fmt.Println()
}

// Example 8: Context and timeout handling
func contextTimeoutExample(client *servicenow.Client) {
	fmt.Println("8. Context and Timeout Handling")
	fmt.Println("-------------------------------")

	catalogClient := client.Catalog()

	// Example with reasonable timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Println("Listing catalogs with 30-second timeout...")
	catalogs, err := catalogClient.ListCatalogsWithContext(ctx)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			fmt.Println("Operation timed out after 30 seconds")
		} else {
			log.Printf("Operation failed: %v", err)
		}
		return
	}
	fmt.Printf("Successfully retrieved %d catalogs within timeout\n", len(catalogs))

	// Example with cancellation
	cancelCtx, cancelFunc := context.WithCancel(context.Background())
	
	// Start operation
	go func() {
		time.Sleep(100 * time.Millisecond) // Simulate delay
		fmt.Println("Cancelling operation...")
		cancelFunc()
	}()

	fmt.Println("Starting operation that will be cancelled...")
	_, err = catalogClient.ListAllItemsWithContext(cancelCtx)
	if err != nil {
		if cancelCtx.Err() == context.Canceled {
			fmt.Println("Operation was cancelled as expected")
		} else {
			log.Printf("Operation failed with different error: %v", err)
		}
	} else {
		fmt.Println("Operation completed before cancellation")
	}

	// Example with very short timeout to demonstrate timeout handling
	shortCtx, shortCancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer shortCancel()

	fmt.Println("Testing with very short timeout (1ms)...")
	_, err = catalogClient.ListCatalogsWithContext(shortCtx)
	if err != nil {
		if shortCtx.Err() == context.DeadlineExceeded {
			fmt.Println("As expected, operation timed out with 1ms timeout")
		} else {
			fmt.Printf("Unexpected error: %v", err)
		}
	} else {
		fmt.Println("Surprisingly, operation completed within 1ms")
	}
	fmt.Println()
}

// Utility function to demonstrate real-world usage patterns
func realWorldUsageExample() {
	// This function demonstrates how you might use the catalog API in a real application

	client, _ := servicenow.NewClientBasicAuth("https://dev.service-now.com", "user", "pass")
	catalogClient := client.Catalog()

	// Build a catalog browser
	buildCatalogBrowser := func() (map[string][]catalog.Category, error) {
		catalogs, err := catalogClient.ListCatalogs()
		if err != nil {
			return nil, err
		}

		catalogMap := make(map[string][]catalog.Category)
		for _, cat := range catalogs {
			categories, err := catalogClient.ListCategories(cat.SysID)
			if err != nil {
				continue
			}
			catalogMap[cat.Title] = categories
		}

		return catalogMap, nil
	}

	// Create a shopping cart manager
	createShoppingCartManager := func() *ShoppingCartManager {
		return &ShoppingCartManager{
			client: catalogClient,
			cart:   make(map[string]CartEntry),
		}
	}

	// Implement request monitoring
	monitorRequests := func(requestNumbers []string) error {
		tracker := catalogClient.NewRequestTracker()
		
		for _, reqNum := range requestNumbers {
			request, err := tracker.GetRequestWithItems(reqNum)
			if err != nil {
				log.Printf("Error tracking request %s: %v", reqNum, err)
				continue
			}

			fmt.Printf("Request %s: %s (%d items)\n", 
				request.Number, request.State, len(request.RequestItems))
			
			for _, item := range request.RequestItems {
				tasks, _ := tracker.GetRequestItemTasks(item.Number)
				fmt.Printf("  Item %s: %s (%d tasks)\n", 
					item.Number, item.State, len(tasks))
			}
		}

		return nil
	}

	// Use the functions (this is just for demonstration)
	_, _ = buildCatalogBrowser()
	_ = createShoppingCartManager()
	_ = monitorRequests([]string{"REQ0001234", "REQ0001235"})
}

// Helper structures for real-world example
type ShoppingCartManager struct {
	client *catalog.CatalogClient
	cart   map[string]CartEntry
}

type CartEntry struct {
	Item      catalog.CatalogItem
	Quantity  int
	Variables map[string]interface{}
}

func (scm *ShoppingCartManager) AddItem(itemSysID string, quantity int, variables map[string]interface{}) error {
	// Get item details
	item, err := scm.client.GetItemWithVariables(itemSysID)
	if err != nil {
		return err
	}

	// Validate variables
	validationErrors, err := scm.client.ValidateItemVariables(itemSysID, variables)
	if err != nil {
		return err
	}

	if len(validationErrors) > 0 {
		return fmt.Errorf("validation failed: %+v", validationErrors)
	}

	// Add to cart
	scm.cart[itemSysID] = CartEntry{
		Item:      *item,
		Quantity:  quantity,
		Variables: variables,
	}

	return nil
}

func (scm *ShoppingCartManager) GetTotalPrice() (float64, error) {
	var total float64
	
	for _, entry := range scm.cart {
		estimate, err := scm.client.EstimatePrice(entry.Item.SysID, entry.Quantity, entry.Variables)
		if err != nil {
			return 0, err
		}
		total += estimate.TotalPrice
	}

	return total, nil
}

func (scm *ShoppingCartManager) Checkout() (*catalog.OrderResult, error) {
	// This is a simplified example - in practice you might need to
	// add items to cart and submit them via the ServiceNow cart API
	
	if len(scm.cart) == 0 {
		return nil, fmt.Errorf("cart is empty")
	}

	// For demonstration - in a real implementation you would
	// iterate through cart items and add them to ServiceNow cart
	// then submit the order
	
	return nil, fmt.Errorf("checkout not implemented in example")
}