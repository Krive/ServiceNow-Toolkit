package integration

import (
	"context"
	"log"
	"os"
	"testing"
	"time"

	"github.com/Krive/ServiceNow-Toolkit/pkg/servicenow"
	"github.com/Krive/ServiceNow-Toolkit/pkg/servicenow/catalog"
)

func setupCatalogClient(t *testing.T) *servicenow.Client {
	username := os.Getenv("SN_USERNAME")
	password := os.Getenv("SN_PASSWORD")
	instanceURL := os.Getenv("SN_INSTANCE_URL")

	if username == "" || password == "" || instanceURL == "" {
		t.Skip("ServiceNow credentials not provided, skipping catalog integration tests")
	}

	client, err := servicenow.NewClientBasicAuth(instanceURL, username, password)
	if err != nil {
		t.Fatalf("Failed to create ServiceNow client: %v", err)
	}

	return client
}

func TestCatalogIntegration_ListCatalogs(t *testing.T) {
	client := setupCatalogClient(t)
	catalogClient := client.Catalog()

	catalogs, err := catalogClient.ListCatalogs()
	if err != nil {
		t.Fatalf("Failed to list catalogs: %v", err)
	}

	log.Printf("Found %d catalogs", len(catalogs))

	if len(catalogs) == 0 {
		t.Log("No catalogs found - this might be expected in some instances")
		return
	}

	// Test first catalog
	catalog := catalogs[0]
	if catalog.SysID == "" {
		t.Error("Catalog SysID should not be empty")
	}
	if catalog.Title == "" {
		t.Error("Catalog Title should not be empty")
	}

	log.Printf("First catalog: %s (%s)", catalog.Title, catalog.SysID)

	// Test getting specific catalog
	specificCatalog, err := catalogClient.GetCatalog(catalog.SysID)
	if err != nil {
		t.Errorf("Failed to get specific catalog: %v", err)
		return
	}

	if specificCatalog.SysID != catalog.SysID {
		t.Errorf("Expected catalog SysID %s, got %s", catalog.SysID, specificCatalog.SysID)
	}

	log.Printf("Retrieved catalog: %s", specificCatalog.Title)
}

func TestCatalogIntegration_ListCategories(t *testing.T) {
	client := setupCatalogClient(t)
	catalogClient := client.Catalog()

	// First get catalogs to find a catalog to test with
	catalogs, err := catalogClient.ListCatalogs()
	if err != nil {
		t.Fatalf("Failed to list catalogs: %v", err)
	}

	if len(catalogs) == 0 {
		t.Skip("No catalogs available for category testing")
	}

	testCatalog := catalogs[0]
	log.Printf("Testing categories for catalog: %s", testCatalog.Title)

	// Test listing categories for specific catalog
	categories, err := catalogClient.ListCategories(testCatalog.SysID)
	if err != nil {
		t.Fatalf("Failed to list categories: %v", err)
	}

	log.Printf("Found %d categories in catalog %s", len(categories), testCatalog.Title)

	if len(categories) > 0 {
		category := categories[0]
		if category.SysID == "" {
			t.Error("Category SysID should not be empty")
		}
		if category.Title == "" {
			t.Error("Category Title should not be empty")
		}

		log.Printf("First category: %s (%s)", category.Title, category.SysID)

		// Test getting specific category
		specificCategory, err := catalogClient.GetCategory(category.SysID)
		if err != nil {
			t.Errorf("Failed to get specific category: %v", err)
		} else {
			if specificCategory.SysID != category.SysID {
				t.Errorf("Expected category SysID %s, got %s", category.SysID, specificCategory.SysID)
			}
			log.Printf("Retrieved category: %s", specificCategory.Title)
		}
	}

	// Test listing all categories
	allCategories, err := catalogClient.ListAllCategories()
	if err != nil {
		t.Fatalf("Failed to list all categories: %v", err)
	}

	log.Printf("Found %d total categories across all catalogs", len(allCategories))
}

func TestCatalogIntegration_SearchCategories(t *testing.T) {
	client := setupCatalogClient(t)
	catalogClient := client.Catalog()

	// Search for categories with common terms
	searchTerms := []string{"software", "hardware", "service", "request"}
	
	for _, term := range searchTerms {
		categories, err := catalogClient.SearchCategories(term)
		if err != nil {
			t.Errorf("Failed to search categories for term '%s': %v", term, err)
			continue
		}

		log.Printf("Search for '%s' found %d categories", term, len(categories))

		// Verify search results contain the search term
		for _, category := range categories {
			if category.SysID == "" {
				t.Errorf("Category SysID should not be empty in search results")
			}
		}
	}
}

func TestCatalogIntegration_ListItems(t *testing.T) {
	client := setupCatalogClient(t)
	catalogClient := client.Catalog()

	// First get catalogs
	catalogs, err := catalogClient.ListCatalogs()
	if err != nil {
		t.Fatalf("Failed to list catalogs: %v", err)
	}

	if len(catalogs) == 0 {
		t.Skip("No catalogs available for item testing")
	}

	testCatalog := catalogs[0]
	log.Printf("Testing items for catalog: %s", testCatalog.Title)

	// Test listing items for specific catalog
	items, err := catalogClient.ListItems(testCatalog.SysID)
	if err != nil {
		t.Fatalf("Failed to list catalog items: %v", err)
	}

	log.Printf("Found %d items in catalog %s", len(items), testCatalog.Title)

	if len(items) > 0 {
		item := items[0]
		if item.SysID == "" {
			t.Error("Item SysID should not be empty")
		}
		if item.Name == "" {
			t.Error("Item Name should not be empty")
		}

		log.Printf("First item: %s (%s)", item.Name, item.SysID)

		// Test getting specific item
		specificItem, err := catalogClient.GetItem(item.SysID)
		if err != nil {
			t.Errorf("Failed to get specific item: %v", err)
		} else {
			if specificItem.SysID != item.SysID {
				t.Errorf("Expected item SysID %s, got %s", item.SysID, specificItem.SysID)
			}
			log.Printf("Retrieved item: %s", specificItem.Name)
		}

		// Test getting item with variables
		itemWithVars, err := catalogClient.GetItemWithVariables(item.SysID)
		if err != nil {
			t.Errorf("Failed to get item with variables: %v", err)
		} else {
			log.Printf("Item %s has %d variables", itemWithVars.Name, len(itemWithVars.Variables))
			
			// Log variable details
			for i, variable := range itemWithVars.Variables {
				if i >= 3 { // Log only first 3 variables
					break
				}
				log.Printf("  Variable %d: %s (%s) - Mandatory: %t", 
					i+1, variable.Question, variable.Type, variable.Mandatory)
				
				if len(variable.Choices) > 0 {
					log.Printf("    Has %d choices", len(variable.Choices))
				}
			}
		}
	}

	// Test listing all items
	allItems, err := catalogClient.ListAllItems()
	if err != nil {
		t.Fatalf("Failed to list all items: %v", err)
	}

	log.Printf("Found %d total items across all catalogs", len(allItems))
}

func TestCatalogIntegration_SearchItems(t *testing.T) {
	client := setupCatalogClient(t)
	catalogClient := client.Catalog()

	// Search for items with common terms
	searchTerms := []string{"laptop", "desktop", "software", "license", "phone"}
	
	for _, term := range searchTerms {
		items, err := catalogClient.SearchItems(term)
		if err != nil {
			t.Errorf("Failed to search items for term '%s': %v", term, err)
			continue
		}

		log.Printf("Search for '%s' found %d items", term, len(items))

		// Log first few results
		for i, item := range items {
			if i >= 3 { // Log only first 3 results
				break
			}
			log.Printf("  Result %d: %s", i+1, item.Name)
		}
	}
}

func TestCatalogIntegration_GetOrderGuides(t *testing.T) {
	client := setupCatalogClient(t)
	catalogClient := client.Catalog()

	orderGuides, err := catalogClient.GetOrderGuides()
	if err != nil {
		t.Fatalf("Failed to get order guides: %v", err)
	}

	log.Printf("Found %d order guides", len(orderGuides))

	for i, guide := range orderGuides {
		if i >= 5 { // Log only first 5 order guides
			break
		}
		log.Printf("  Order Guide %d: %s (%s)", i+1, guide.Name, guide.SysID)
	}
}

func TestCatalogIntegration_GetItemsByType(t *testing.T) {
	client := setupCatalogClient(t)
	catalogClient := client.Catalog()

	// Test different item types
	itemTypes := []string{"item", "bundle", "guide"}
	
	for _, itemType := range itemTypes {
		items, err := catalogClient.GetItemsByType(itemType)
		if err != nil {
			t.Errorf("Failed to get items by type '%s': %v", itemType, err)
			continue
		}

		log.Printf("Found %d items of type '%s'", len(items), itemType)
	}
}

func TestCatalogIntegration_ValidateVariables(t *testing.T) {
	client := setupCatalogClient(t)
	catalogClient := client.Catalog()

	// Get an item with variables for testing
	allItems, err := catalogClient.ListAllItems()
	if err != nil {
		t.Fatalf("Failed to list items for validation test: %v", err)
	}

	var testItem *catalog.CatalogItem
	for _, item := range allItems {
		itemWithVars, err := catalogClient.GetItemWithVariables(item.SysID)
		if err != nil {
			continue
		}
		if len(itemWithVars.Variables) > 0 {
			testItem = itemWithVars
			break
		}
	}

	if testItem == nil {
		t.Skip("No items with variables found for validation testing")
	}

	log.Printf("Testing variable validation for item: %s", testItem.Name)

	// Test validation with empty variables (should fail for mandatory variables)
	validationErrors, err := catalogClient.ValidateItemVariables(testItem.SysID, map[string]interface{}{})
	if err != nil {
		t.Errorf("Variable validation failed: %v", err)
		return
	}

	log.Printf("Validation with empty variables found %d errors", len(validationErrors))
	for i, valErr := range validationErrors {
		if i >= 3 { // Log only first 3 errors
			break
		}
		log.Printf("  Error %d: %s - %s", i+1, valErr.Variable, valErr.Message)
	}

	// Test validation with some variables
	testVariables := make(map[string]interface{})
	for _, variable := range testItem.Variables {
		if variable.Mandatory {
			if variable.Type == "choice" && len(variable.Choices) > 0 {
				testVariables[variable.Name] = variable.Choices[0].Value
			} else if variable.DefaultValue != "" {
				testVariables[variable.Name] = variable.DefaultValue
			} else {
				testVariables[variable.Name] = "test_value"
			}
		}
	}

	if len(testVariables) > 0 {
		validationErrors, err = catalogClient.ValidateItemVariables(testItem.SysID, testVariables)
		if err != nil {
			t.Errorf("Variable validation with test values failed: %v", err)
		} else {
			log.Printf("Validation with test variables found %d errors", len(validationErrors))
		}
	}
}

func TestCatalogIntegration_EstimatePrice(t *testing.T) {
	client := setupCatalogClient(t)
	catalogClient := client.Catalog()

	// Get an item for price estimation
	allItems, err := catalogClient.ListAllItems()
	if err != nil {
		t.Fatalf("Failed to list items for price estimation: %v", err)
	}

	if len(allItems) == 0 {
		t.Skip("No items available for price estimation")
	}

	testItem := allItems[0]
	log.Printf("Testing price estimation for item: %s", testItem.Name)

	estimate, err := catalogClient.EstimatePrice(testItem.SysID, 2, map[string]interface{}{
		"test_variable": "test_value",
	})
	if err != nil {
		t.Errorf("Failed to estimate price: %v", err)
		return
	}

	log.Printf("Price estimate for %s (qty: %d):", testItem.Name, estimate.Quantity)
	log.Printf("  Base Price: $%.2f", estimate.BasePrice)
	log.Printf("  Recurring Price: $%.2f", estimate.RecurringPrice)
	log.Printf("  Total Price: $%.2f", estimate.TotalPrice)
	log.Printf("  Total Recurring: $%.2f", estimate.TotalRecurring)
	log.Printf("  Currency: %s", estimate.Currency)

	if estimate.Quantity != 2 {
		t.Errorf("Expected quantity 2, got %d", estimate.Quantity)
	}
}

func TestCatalogIntegration_RequestTracker(t *testing.T) {
	client := setupCatalogClient(t)
	catalogClient := client.Catalog()
	tracker := catalogClient.NewRequestTracker()

	// Test getting user's requests
	myRequests, err := tracker.GetMyRequests(10)
	if err != nil {
		t.Errorf("Failed to get user requests: %v", err)
		return
	}

	log.Printf("Found %d requests for current user", len(myRequests))

	if len(myRequests) > 0 {
		request := myRequests[0]
		log.Printf("Latest request: %s (%s) - State: %s", 
			request.Number, request.ShortDescription, request.State)

		// Test getting request with items
		requestWithItems, err := tracker.GetRequestWithItems(request.Number)
		if err != nil {
			t.Errorf("Failed to get request with items: %v", err)
		} else {
			log.Printf("Request %s has %d items", 
				requestWithItems.Number, len(requestWithItems.RequestItems))
		}

		// Test getting request items
		items, err := tracker.GetRequestItems(request.Number)
		if err != nil {
			t.Errorf("Failed to get request items: %v", err)
		} else {
			log.Printf("Found %d request items for %s", len(items), request.Number)
			
			// Test getting tasks for first item if available
			if len(items) > 0 {
				item := items[0]
				log.Printf("First item: %s (%s) - State: %s", 
					item.Number, item.CatalogItemSysID, item.State)

				tasks, err := tracker.GetRequestItemTasks(item.Number)
				if err != nil {
					t.Errorf("Failed to get request item tasks: %v", err)
				} else {
					log.Printf("Request item %s has %d tasks", item.Number, len(tasks))
					
					for i, task := range tasks {
						if i >= 3 { // Log only first 3 tasks
							break
						}
						log.Printf("  Task %d: %s (%s) - State: %s", 
							i+1, task.Number, task.ShortDescription, task.State)
					}
				}

				// Test getting request item with tasks
				itemWithTasks, err := tracker.GetRequestItemWithTasks(item.Number)
				if err != nil {
					t.Errorf("Failed to get request item with tasks: %v", err)
				} else {
					log.Printf("Request item with tasks has %d tasks embedded", 
						len(itemWithTasks.Tasks))
				}
			}
		}
	}
}

func TestCatalogIntegration_RequestsByState(t *testing.T) {
	client := setupCatalogClient(t)
	catalogClient := client.Catalog()
	tracker := catalogClient.NewRequestTracker()

	// Test getting requests by different states
	states := []string{"in_process", "closed_complete", "closed_cancelled", "pending_approval"}
	
	for _, state := range states {
		requests, err := tracker.GetRequestsByState(state)
		if err != nil {
			t.Errorf("Failed to get requests by state '%s': %v", state, err)
			continue
		}

		log.Printf("Found %d requests in state '%s'", len(requests), state)
	}
}

func TestCatalogIntegration_ContextTimeout(t *testing.T) {
	client := setupCatalogClient(t)
	catalogClient := client.Catalog()

	// Test with reasonable timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	catalogs, err := catalogClient.ListCatalogsWithContext(ctx)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			t.Log("Catalog listing timed out after 30 seconds")
		} else {
			t.Errorf("Catalog listing failed: %v", err)
		}
		return
	}

	log.Printf("Context test: Found %d catalogs within timeout", len(catalogs))

	// Test with very short timeout
	shortCtx, shortCancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer shortCancel()

	_, err = catalogClient.ListCatalogsWithContext(shortCtx)
	if err != nil {
		if shortCtx.Err() == context.DeadlineExceeded {
			log.Printf("As expected, catalog listing timed out with 1ms timeout")
		} else {
			t.Errorf("Expected timeout error, got: %v", err)
		}
	} else {
		log.Printf("Surprisingly, catalog listing completed within 1ms")
	}
}

// Note: Cart and ordering integration tests are commented out because they would
// create actual requests in the ServiceNow instance. Uncomment and modify for
// testing in a development environment only.

/*
func TestCatalogIntegration_CartOperations(t *testing.T) {
	client := setupCatalogClient(t)
	catalogClient := client.Catalog()

	// WARNING: This test will create actual cart items and potentially orders
	// Only run in a development environment
	
	// Get an item to add to cart
	allItems, err := catalogClient.ListAllItems()
	if err != nil || len(allItems) == 0 {
		t.Skip("No items available for cart testing")
	}

	testItem := allItems[0]
	log.Printf("Testing cart operations with item: %s", testItem.Name)

	// Add item to cart
	cartResponse, err := catalogClient.AddToCart(testItem.SysID, 1, map[string]interface{}{})
	if err != nil {
		t.Errorf("Failed to add item to cart: %v", err)
		return
	}

	if !cartResponse.Success {
		t.Errorf("Add to cart failed: %s", cartResponse.Error)
		return
	}

	log.Printf("Successfully added item to cart: %s", cartResponse.Message)

	// Get cart contents
	cart, err := catalogClient.GetCart()
	if err != nil {
		t.Errorf("Failed to get cart: %v", err)
		return
	}

	log.Printf("Cart contains %d items, total: %s", len(cart.Items), cart.TotalPrice)

	// Clean up - clear cart to avoid creating actual orders
	err = catalogClient.ClearCart()
	if err != nil {
		t.Errorf("Failed to clear cart: %v", err)
	} else {
		log.Printf("Cart cleared successfully")
	}
}
*/