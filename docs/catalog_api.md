# ServiceNow Service Catalog API Reference

The ServiceNow Service Catalog API in ServiceNow Toolkit provides comprehensive functionality for browsing catalogs, managing shopping carts, ordering items, and tracking requests. This API integrates with both ServiceNow's Service Catalog API (`/api/sn_sc/servicecatalog/`) and Table API endpoints.

## Quick Start

```go
import "github.com/Krive/ServiceNow Toolkit/pkg/servicenow"

client, _ := servicenow.NewClientBasicAuth(instanceURL, username, password)
catalogClient := client.Catalog()

// Browse catalogs
catalogs, err := catalogClient.ListCatalogs()

// Search for items
items, err := catalogClient.SearchItems("laptop")

// Get item with variables
item, err := catalogClient.GetItemWithVariables("catalog_item_sys_id")
```

## CatalogClient Methods

### Creating a Catalog Client

```go
// Get catalog client from main client
catalogClient := client.Catalog()
```

## Catalog Browsing

### Catalogs

#### ListCatalogs
Returns all available catalogs.

```go
catalogs, err := catalogClient.ListCatalogs()
// With context
catalogs, err := catalogClient.ListCatalogsWithContext(ctx)
```

#### GetCatalog
Returns a specific catalog by sys_id.

```go
catalog, err := catalogClient.GetCatalog("catalog_sys_id")
// With context
catalog, err := catalogClient.GetCatalogWithContext(ctx, "catalog_sys_id")
```

### Categories

#### ListCategories
Returns categories for a specific catalog.

```go
categories, err := catalogClient.ListCategories("catalog_sys_id")
// With context
categories, err := catalogClient.ListCategoriesWithContext(ctx, "catalog_sys_id")
```

#### ListAllCategories
Returns all active categories across all catalogs.

```go
categories, err := catalogClient.ListAllCategories()
// With context
categories, err := catalogClient.ListAllCategoriesWithContext(ctx)
```

#### GetCategory
Returns a specific category by sys_id.

```go
category, err := catalogClient.GetCategory("category_sys_id")
// With context
category, err := catalogClient.GetCategoryWithContext(ctx, "category_sys_id")
```

#### SearchCategories
Searches categories by title or description.

```go
categories, err := catalogClient.SearchCategories("software")
// With context
categories, err := catalogClient.SearchCategoriesWithContext(ctx, "software")
```

### Catalog Items

#### ListItems
Returns catalog items for a specific catalog.

```go
items, err := catalogClient.ListItems("catalog_sys_id")
// With context
items, err := catalogClient.ListItemsWithContext(ctx, "catalog_sys_id")
```

#### ListItemsByCategory
Returns catalog items for a specific category.

```go
items, err := catalogClient.ListItemsByCategory("category_sys_id")
// With context
items, err := catalogClient.ListItemsByCategoryWithContext(ctx, "category_sys_id")
```

#### ListAllItems
Returns all active catalog items.

```go
items, err := catalogClient.ListAllItems()
// With context
items, err := catalogClient.ListAllItemsWithContext(ctx)
```

#### GetItem
Returns a specific catalog item by sys_id.

```go
item, err := catalogClient.GetItem("item_sys_id")
// With context
item, err := catalogClient.GetItemWithContext(ctx, "item_sys_id")
```

#### GetItemWithVariables
Returns a catalog item with its variables.

```go
item, err := catalogClient.GetItemWithVariables("item_sys_id")
// With context
item, err := catalogClient.GetItemWithVariablesWithContext(ctx, "item_sys_id")
```

#### SearchItems
Searches catalog items by name or description.

```go
items, err := catalogClient.SearchItems("laptop")
// With context
items, err := catalogClient.SearchItemsWithContext(ctx, "laptop")
```

#### GetItemsByType
Returns catalog items of a specific type.

```go
items, err := catalogClient.GetItemsByType("item")
// With context
items, err := catalogClient.GetItemsByTypeWithContext(ctx, "item")
```

#### GetOrderGuides
Returns catalog items that are order guides.

```go
guides, err := catalogClient.GetOrderGuides()
// With context
guides, err := catalogClient.GetOrderGuidesWithContext(ctx)
```

### Item Variables

#### GetItemVariables
Returns variables for a catalog item.

```go
variables, err := catalogClient.GetItemVariables("item_sys_id")
// With context
variables, err := catalogClient.GetItemVariablesWithContext(ctx, "item_sys_id")
```

#### ValidateItemVariables
Validates variables for a catalog item.

```go
variables := map[string]interface{}{
    "cpu_type": "intel_i7",
    "memory": "16gb",
}

validationErrors, err := catalogClient.ValidateItemVariables("item_sys_id", variables)
// With context
validationErrors, err := catalogClient.ValidateItemVariablesWithContext(ctx, "item_sys_id", variables)

// Check for validation errors
if len(validationErrors) > 0 {
    for _, valErr := range validationErrors {
        fmt.Printf("Error: %s - %s (%s)\n", valErr.Variable, valErr.Message, valErr.Type)
    }
}
```

## Shopping Cart Operations

### AddToCart
Adds a catalog item to the shopping cart.

```go
variables := map[string]interface{}{
    "cpu_type": "intel_i7",
    "memory": "16gb",
}

cartResponse, err := catalogClient.AddToCart("item_sys_id", 1, variables)
// With context
cartResponse, err := catalogClient.AddToCartWithContext(ctx, "item_sys_id", 1, variables)

if cartResponse.Success {
    fmt.Printf("Added to cart: %s\n", cartResponse.Message)
}
```

### GetCart
Returns the current cart contents.

```go
cart, err := catalogClient.GetCart()
// With context
cart, err := catalogClient.GetCartWithContext(ctx)

fmt.Printf("Cart has %d items, total: %s\n", len(cart.Items), cart.TotalPrice)
```

### UpdateCartItem
Updates the quantity or variables of a cart item.

```go
updatedVariables := map[string]interface{}{
    "cpu_type": "intel_i9",
    "memory": "32gb",
}

err := catalogClient.UpdateCartItem("cart_item_sys_id", 2, updatedVariables)
// With context
err := catalogClient.UpdateCartItemWithContext(ctx, "cart_item_sys_id", 2, updatedVariables)
```

### RemoveFromCart
Removes an item from the cart.

```go
err := catalogClient.RemoveFromCart("cart_item_sys_id")
// With context
err := catalogClient.RemoveFromCartWithContext(ctx, "cart_item_sys_id")
```

### ClearCart
Removes all items from the cart.

```go
err := catalogClient.ClearCart()
// With context
err := catalogClient.ClearCartWithContext(ctx)
```

### SubmitCart
Submits the current cart as an order.

```go
orderResult, err := catalogClient.SubmitCart()
// With context
orderResult, err := catalogClient.SubmitCartWithContext(ctx)

if orderResult.Success {
    fmt.Printf("Order submitted: %s\n", orderResult.RequestNumber)
}
```

## Direct Ordering

### OrderNow
Directly orders a catalog item without using the cart.

```go
variables := map[string]interface{}{
    "cpu_type": "intel_i7",
    "memory": "16gb",
}

orderResult, err := catalogClient.OrderNow("item_sys_id", 1, variables)
// With context
orderResult, err := catalogClient.OrderNowWithContext(ctx, "item_sys_id", 1, variables)

if orderResult.Success {
    fmt.Printf("Direct order submitted: %s\n", orderResult.RequestNumber)
}
```

## Price Estimation

### EstimatePrice
Estimates the price for a catalog item with given variables.

```go
variables := map[string]interface{}{
    "service_level": "premium",
}

estimate, err := catalogClient.EstimatePrice("item_sys_id", 2, variables)
// With context
estimate, err := catalogClient.EstimatePriceWithContext(ctx, "item_sys_id", 2, variables)

fmt.Printf("Price for %d items:\n", estimate.Quantity)
fmt.Printf("  Base: $%.2f each, Total: $%.2f\n", estimate.BasePrice, estimate.TotalPrice)
fmt.Printf("  Recurring: $%.2f each, Total: $%.2f\n", estimate.RecurringPrice, estimate.TotalRecurring)
```

## Request Tracking

### Creating a Request Tracker

```go
tracker := catalogClient.NewRequestTracker()
```

### Tracking Requests

#### GetRequest
Returns a catalog request by number or sys_id.

```go
request, err := tracker.GetRequest("REQ0012345")
// With context
request, err := tracker.GetRequestWithContext(ctx, "REQ0012345")
```

#### GetRequestWithItems
Returns a request with its request items.

```go
request, err := tracker.GetRequestWithItems("REQ0012345")
// With context
request, err := tracker.GetRequestWithItemsWithContext(ctx, "REQ0012345")

fmt.Printf("Request has %d items\n", len(request.RequestItems))
```

#### GetMyRequests
Returns requests for the current user.

```go
myRequests, err := tracker.GetMyRequests(10) // Limit to 10 most recent
// With context
myRequests, err := tracker.GetMyRequestsWithContext(ctx, 10)

for _, request := range myRequests {
    fmt.Printf("Request: %s - %s (%s)\n", request.Number, request.ShortDescription, request.State)
}
```

#### GetRequestsByState
Returns requests in a specific state.

```go
requests, err := tracker.GetRequestsByState("in_process")
// With context
requests, err := tracker.GetRequestsByStateWithContext(ctx, "in_process")
```

### Tracking Request Items

#### GetRequestItems
Returns request items for a request.

```go
items, err := tracker.GetRequestItems("REQ0012345")
// With context
items, err := tracker.GetRequestItemsWithContext(ctx, "REQ0012345")
```

#### GetRequestItem
Returns a specific request item.

```go
item, err := tracker.GetRequestItem("RITM0012345")
// With context
item, err := tracker.GetRequestItemWithContext(ctx, "RITM0012345")
```

#### GetRequestItemWithTasks
Returns a request item with its tasks.

```go
item, err := tracker.GetRequestItemWithTasks("RITM0012345")
// With context
item, err := tracker.GetRequestItemWithTasksWithContext(ctx, "RITM0012345")

fmt.Printf("Request item has %d tasks\n", len(item.Tasks))
```

### Tracking Tasks

#### GetRequestItemTasks
Returns tasks for a request item.

```go
tasks, err := tracker.GetRequestItemTasks("RITM0012345")
// With context
tasks, err := tracker.GetRequestItemTasksWithContext(ctx, "RITM0012345")

for _, task := range tasks {
    fmt.Printf("Task: %s - %s (%s)\n", task.Number, task.ShortDescription, task.State)
}
```

### Progress Monitoring

#### TrackRequestProgress
Tracks the progress of a request over time.

```go
err := tracker.TrackRequestProgress("REQ0012345", func(request *catalog.Request, items []catalog.RequestItem) {
    fmt.Printf("Request %s: %s\n", request.Number, request.State)
    for _, item := range items {
        fmt.Printf("  Item %s: %s\n", item.Number, item.State)
    }
    
    // Stop tracking when request is closed
    if request.State == "closed_complete" || request.State == "closed_cancelled" {
        fmt.Printf("Request %s is now closed\n", request.Number)
    }
})

// With context
ctx, cancel := context.WithTimeout(context.Background(), 1*time.Hour)
defer cancel()

err := tracker.TrackRequestProgressWithContext(ctx, "REQ0012345", callback)
```

## Data Structures

### Catalog
```go
type Catalog struct {
    SysID       string `json:"sys_id"`
    Title       string `json:"title"`
    Description string `json:"description"`
    Active      bool   `json:"active"`
    Background  string `json:"background_color"`
    Icon        string `json:"icon"`
}
```

### Category
```go
type Category struct {
    SysID         string `json:"sys_id"`
    Title         string `json:"title"`
    Description   string `json:"description"`
    Active        bool   `json:"active"`
    CatalogSysID  string `json:"sc_catalog"`
    ParentSysID   string `json:"parent"`
    Icon          string `json:"icon"`
    Order         int    `json:"order"`
}
```

### CatalogItem
```go
type CatalogItem struct {
    SysID               string                 `json:"sys_id"`
    Name                string                 `json:"name"`
    ShortDescription    string                 `json:"short_description"`
    Description         string                 `json:"description"`
    Active              bool                   `json:"active"`
    CatalogSysID        string                 `json:"sc_catalog"`
    CategorySysID       string                 `json:"category"`
    Price               string                 `json:"price"`
    RecurringPrice      string                 `json:"recurring_price"`
    Icon                string                 `json:"icon"`
    Picture             string                 `json:"picture"`
    Type                string                 `json:"type"`
    OrderGuide          bool                   `json:"order_guide"`
    Variables           []CatalogVariable      `json:"variables,omitempty"`
    // ... additional fields
}
```

### CatalogVariable
```go
type CatalogVariable struct {
    SysID              string                 `json:"sys_id"`
    Name               string                 `json:"name"`
    Question           string                 `json:"question"`
    Type               string                 `json:"type"` // "string", "choice", "boolean", etc.
    Mandatory          bool                   `json:"mandatory"`
    Active             bool                   `json:"active"`
    DefaultValue       string                 `json:"default_value"`
    HelpText           string                 `json:"help_text"`
    Order              int                    `json:"order"`
    ReadOnly           bool                   `json:"read_only"`
    Visible            bool                   `json:"visible"`
    Choices            []VariableChoice       `json:"choices,omitempty"`
    // ... additional fields
}
```

### VariableChoice
```go
type VariableChoice struct {
    Value     string `json:"value"`
    Text      string `json:"text"`
    Dependent string `json:"dependent,omitempty"`
    Order     int    `json:"order"`
}
```

### Cart and CartItem
```go
type Cart struct {
    Items      []CartItem `json:"items"`
    TotalPrice string     `json:"total_price"`
    Subtotal   string     `json:"subtotal"`
    Tax        string     `json:"tax"`
}

type CartItem struct {
    SysID            string                 `json:"sys_id"`
    CatalogItemSysID string                 `json:"cat_item"`
    Quantity         int                    `json:"quantity"`
    Price            string                 `json:"price"`
    RecurringPrice   string                 `json:"recurring_price"`
    Variables        map[string]interface{} `json:"variables"`
}
```

### OrderResult
```go
type OrderResult struct {
    Success       bool          `json:"success"`
    RequestNumber string        `json:"request_number"`
    RequestSysID  string        `json:"request_sys_id"`
    RequestItems  []RequestItem `json:"request_items"`
    Message       string        `json:"message,omitempty"`
    Error         string        `json:"error,omitempty"`
}
```

### Request
```go
type Request struct {
    SysID             string    `json:"sys_id"`
    Number            string    `json:"number"`
    State             string    `json:"state"`
    Stage             string    `json:"stage"`
    RequestedBy       string    `json:"requested_by"`
    RequestedFor      string    `json:"requested_for"`
    OpenedBy          string    `json:"opened_by"`
    OpenedAt          time.Time `json:"opened_at"`
    Description       string    `json:"description"`
    ShortDescription  string    `json:"short_description"`
    Price             string    `json:"price"`
    Priority          string    `json:"priority"`
    ApprovalState     string    `json:"approval"`
    RequestItems      []RequestItem `json:"request_items,omitempty"`
    // ... additional fields
}
```

### RequestItem
```go
type RequestItem struct {
    SysID                 string                 `json:"sys_id"`
    Number                string                 `json:"number"`
    State                 string                 `json:"state"`
    Stage                 string                 `json:"stage"`
    RequestSysID          string                 `json:"request"`
    CatalogItemSysID      string                 `json:"cat_item"`
    Quantity              int                    `json:"quantity"`
    Price                 string                 `json:"price"`
    Variables             map[string]interface{} `json:"variables,omitempty"`
    Tasks                 []CatalogTask          `json:"tasks,omitempty"`
    ApprovalState         string                 `json:"approval"`
    FulfillmentGroup      string                 `json:"assignment_group"`
    AssignedTo            string                 `json:"assigned_to"`
    // ... additional fields
}
```

### CatalogTask
```go
type CatalogTask struct {
    SysID            string    `json:"sys_id"`
    Number           string    `json:"number"`
    State            string    `json:"state"`
    RequestItemSysID string    `json:"request_item"`
    ShortDescription string    `json:"short_description"`
    Description      string    `json:"description"`
    AssignedTo       string    `json:"assigned_to"`
    AssignmentGroup  string    `json:"assignment_group"`
    OpenedAt         time.Time `json:"opened_at"`
    DueDate          time.Time `json:"due_date"`
    Priority         string    `json:"priority"`
    // ... additional fields
}
```

### ValidationError
```go
type ValidationError struct {
    Variable string `json:"variable"`
    Message  string `json:"message"`
    Type     string `json:"type"` // "missing_mandatory", "invalid_choice", etc.
}
```

### PriceEstimate
```go
type PriceEstimate struct {
    ItemSysID      string                 `json:"item_sys_id"`
    Quantity       int                    `json:"quantity"`
    BasePrice      float64                `json:"base_price"`
    RecurringPrice float64                `json:"recurring_price"`
    TotalPrice     float64                `json:"total_price"`
    TotalRecurring float64                `json:"total_recurring"`
    Currency       string                 `json:"currency"`
    Variables      map[string]interface{} `json:"variables"`
}
```

## Variable Types and Validation

### Supported Variable Types
- **string**: Text input
- **choice**: Select from predefined options
- **boolean**: True/false checkbox
- **integer**: Numeric input
- **reference**: Reference to another table record
- **date**: Date picker
- **datetime**: Date and time picker
- **email**: Email address input
- **url**: URL input

### Variable Validation Rules
1. **Mandatory Variables**: Must be provided and non-empty
2. **Choice Variables**: Value must match one of the available choices
3. **Reference Variables**: Must point to valid record in reference table
4. **Type-specific**: Format validation based on variable type

### Example Variable Handling
```go
// Handle different variable types
variables := map[string]interface{}{
    // String variable
    "description": "Custom configuration",
    
    // Choice variable (use the value, not display text)
    "priority": "high", // not "High Priority"
    
    // Boolean variable
    "backup_required": "true", // or true/false
    
    // Integer variable
    "cpu_cores": 4, // or "4"
    
    // Reference variable (sys_id of referenced record)
    "assigned_user": "user_sys_id_here",
    
    // Date variable
    "required_date": "2024-01-15",
    
    // Email variable
    "notification_email": "user@company.com",
}
```

## Error Handling

### Catalog-Level Errors
```go
catalogs, err := catalogClient.ListCatalogs()
if err != nil {
    // Handle API errors (network, authentication, etc.)
    log.Printf("Failed to list catalogs: %v", err)
}
```

### Cart Operation Errors
```go
cartResponse, err := catalogClient.AddToCart(itemSysID, 1, variables)
if err != nil {
    log.Printf("Add to cart failed: %v", err)
} else if !cartResponse.Success {
    log.Printf("Add to cart rejected: %s", cartResponse.Error)
}
```

### Variable Validation Errors
```go
validationErrors, err := catalogClient.ValidateItemVariables(itemSysID, variables)
if err != nil {
    log.Printf("Validation check failed: %v", err)
} else if len(validationErrors) > 0 {
    for _, valErr := range validationErrors {
        switch valErr.Type {
        case "missing_mandatory":
            log.Printf("Required field missing: %s", valErr.Variable)
        case "invalid_choice":
            log.Printf("Invalid choice for %s: %s", valErr.Variable, valErr.Message)
        default:
            log.Printf("Validation error for %s: %s", valErr.Variable, valErr.Message)
        }
    }
}
```

### Order Submission Errors
```go
orderResult, err := catalogClient.OrderNow(itemSysID, 1, variables)
if err != nil {
    log.Printf("Order submission failed: %v", err)
} else if !orderResult.Success {
    log.Printf("Order rejected: %s", orderResult.Error)
} else {
    log.Printf("Order successful: %s", orderResult.RequestNumber)
}
```

## Best Practices

### Variable Management
1. **Always validate variables** before ordering
2. **Use exact choice values** from variable definitions
3. **Handle mandatory variables** properly
4. **Provide meaningful default values** where appropriate

### Cart Management
1. **Clear cart after successful orders** to avoid confusion
2. **Validate items before adding to cart**
3. **Handle cart timeouts** gracefully
4. **Update quantities and variables** as needed

### Request Tracking
1. **Use request numbers for tracking** (more user-friendly than sys_ids)
2. **Monitor request state changes** for automated workflows
3. **Track task progress** for detailed status updates
4. **Set appropriate timeouts** for progress monitoring

### Performance Optimization
1. **Cache catalog structure** when possible
2. **Use context timeouts** for long-running operations
3. **Batch related operations** when possible
4. **Implement proper retry logic** for transient failures

### Security Considerations
1. **Validate user permissions** before allowing orders
2. **Sanitize variable inputs** to prevent injection attacks
3. **Use secure authentication methods**
4. **Log audit trails** for compliance

## ServiceNow API Mapping

This service catalog implementation maps to several ServiceNow APIs:

### Service Catalog API (`/api/sn_sc/servicecatalog/`)
- Cart operations (add, update, remove, submit)
- Direct ordering
- Item details with variables

### Table API (`/api/now/table/`)
- Catalog browsing (`sc_catalog`, `sc_cat_item_category`, `sc_cat_item`)
- Request tracking (`sc_request`, `sc_req_item`, `sc_task`)
- Variable definitions (`item_option_new`, `question_choice`)

### Authentication
- Supports all ServiceNow authentication methods
- Basic auth, OAuth 2.0, API keys
- Inherits rate limiting and retry policies from core client

## Troubleshooting

### Common Issues

#### "Catalog item not found"
- Verify the sys_id is correct
- Check if the item is active
- Ensure user has access to the catalog

#### "Variable validation failed"
- Check mandatory variables are provided
- Verify choice values match exactly
- Ensure variable names are correct

#### "Cart operation failed"
- Check if user session is valid
- Verify cart isn't empty for submission
- Ensure items are still available

#### "Request not found"
- Verify request number format
- Check if user has access to the request
- Ensure request exists in the system

### Debug Tips
1. **Enable debug logging** to see API requests/responses
2. **Use ServiceNow's REST API Explorer** to test endpoints
3. **Check ServiceNow logs** for server-side errors
4. **Verify user permissions** in ServiceNow admin console

### Performance Issues
1. **Reduce batch sizes** for large operations
2. **Implement caching** for frequently accessed data
3. **Use appropriate timeouts** for network operations
4. **Monitor rate limits** and implement backoff strategies