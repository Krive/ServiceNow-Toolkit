package unit

import (
	"testing"

	"github.com/Krive/ServiceNow-Toolkit/pkg/servicenow/catalog"
	"github.com/Krive/ServiceNow-Toolkit/pkg/servicenow/core"
)

func TestCatalogClient_NewCatalogClient(t *testing.T) {
	client := &core.Client{} // Mock client
	catalogClient := catalog.NewCatalogClient(client)
	
	if catalogClient == nil {
		t.Fatal("NewCatalogClient should return a non-nil CatalogClient")
	}
}

func TestCatalogClient_NewRequestTracker(t *testing.T) {
	client := &core.Client{}
	catalogClient := catalog.NewCatalogClient(client)
	
	tracker := catalogClient.NewRequestTracker()
	
	if tracker == nil {
		t.Fatal("NewRequestTracker should return a non-nil RequestTracker")
	}
}

func TestCatalogStructures(t *testing.T) {
	// Test that all catalog structures can be created
	
	testCatalog := catalog.Catalog{
		SysID:       "test_sys_id",
		Title:       "Test Catalog",
		Description: "Test Description",
		Active:      true,
		Background:  "#ffffff",
		Icon:        "test-icon",
	}
	
	if testCatalog.SysID != "test_sys_id" {
		t.Errorf("Expected SysID 'test_sys_id', got '%s'", testCatalog.SysID)
	}
	
	testCategory := catalog.Category{
		SysID:        "category_sys_id",
		Title:        "Test Category",
		Description:  "Category Description",
		Active:       true,
		CatalogSysID: "catalog_sys_id",
		ParentSysID:  "parent_sys_id",
		Icon:         "category-icon",
		Order:        1,
	}
	
	if testCategory.Title != "Test Category" {
		t.Errorf("Expected Title 'Test Category', got '%s'", testCategory.Title)
	}
	
	testItem := catalog.CatalogItem{
		SysID:            "item_sys_id",
		Name:             "Test Item",
		ShortDescription: "Test Short Description",
		Description:      "Test Description",
		Active:           true,
		CatalogSysID:     "catalog_sys_id",
		CategorySysID:    "category_sys_id",
		Price:            "10.00",
		RecurringPrice:   "5.00",
		Icon:             "item-icon",
		Picture:          "item-picture.jpg",
		Type:             "item",
		OrderGuide:       false,
	}
	
	if testItem.Name != "Test Item" {
		t.Errorf("Expected Name 'Test Item', got '%s'", testItem.Name)
	}
}

func TestCatalogVariable(t *testing.T) {
	variable := catalog.CatalogVariable{
		SysID:          "variable_sys_id",
		Name:           "test_variable",
		Question:       "Test Question?",
		Type:           "string",
		Mandatory:      true,
		Active:         true,
		DefaultValue:   "default",
		HelpText:       "Help text",
		Order:          1,
		ReadOnly:       false,
		Visible:        true,
		ChoiceTable:    "",
		ChoiceField:    "",
		ReferenceTable: "",
	}
	
	if variable.Name != "test_variable" {
		t.Errorf("Expected Name 'test_variable', got '%s'", variable.Name)
	}
	
	if !variable.Mandatory {
		t.Error("Expected variable to be mandatory")
	}
	
	choice := catalog.VariableChoice{
		Value:     "choice1",
		Text:      "Choice 1",
		Dependent: "",
		Order:     1,
	}
	
	if choice.Value != "choice1" {
		t.Errorf("Expected Value 'choice1', got '%s'", choice.Value)
	}
}

func TestCartItem(t *testing.T) {
	cartItem := catalog.CartItem{
		SysID:            "cart_item_sys_id",
		CatalogItemSysID: "catalog_item_sys_id",
		Quantity:         2,
		Price:            "20.00",
		RecurringPrice:   "10.00",
		Variables: map[string]interface{}{
			"cpu":     "intel_i7",
			"ram":     "16gb",
			"storage": "512gb",
		},
	}
	
	if cartItem.Quantity != 2 {
		t.Errorf("Expected Quantity 2, got %d", cartItem.Quantity)
	}
	
	if cartItem.Variables["cpu"] != "intel_i7" {
		t.Errorf("Expected CPU 'intel_i7', got '%v'", cartItem.Variables["cpu"])
	}
}

func TestCart(t *testing.T) {
	cart := catalog.Cart{
		Items: []catalog.CartItem{
			{
				SysID:            "item1",
				CatalogItemSysID: "catalog_item1",
				Quantity:         1,
				Price:            "10.00",
			},
			{
				SysID:            "item2",
				CatalogItemSysID: "catalog_item2",
				Quantity:         2,
				Price:            "20.00",
			},
		},
		TotalPrice: "50.00",
		Subtotal:   "40.00",
		Tax:        "10.00",
	}
	
	if len(cart.Items) != 2 {
		t.Errorf("Expected 2 cart items, got %d", len(cart.Items))
	}
	
	if cart.TotalPrice != "50.00" {
		t.Errorf("Expected TotalPrice '50.00', got '%s'", cart.TotalPrice)
	}
}

func TestAddToCartRequest(t *testing.T) {
	request := catalog.AddToCartRequest{
		CatalogItemSysID: "item_sys_id",
		Quantity:         1,
		Variables: map[string]interface{}{
			"cpu":    "intel_i5",
			"memory": "8gb",
		},
	}
	
	if request.CatalogItemSysID != "item_sys_id" {
		t.Errorf("Expected CatalogItemSysID 'item_sys_id', got '%s'", request.CatalogItemSysID)
	}
	
	if request.Quantity != 1 {
		t.Errorf("Expected Quantity 1, got %d", request.Quantity)
	}
}

func TestAddToCartResponse(t *testing.T) {
	response := catalog.AddToCartResponse{
		Success: true,
		CartID:  "cart_id_123",
		ItemID:  "item_id_456",
		Message: "Item added successfully",
		Error:   "",
	}
	
	if !response.Success {
		t.Error("Expected Success to be true")
	}
	
	if response.CartID != "cart_id_123" {
		t.Errorf("Expected CartID 'cart_id_123', got '%s'", response.CartID)
	}
}

func TestOrderResult(t *testing.T) {
	orderResult := catalog.OrderResult{
		Success:       true,
		RequestNumber: "REQ0012345",
		RequestSysID:  "request_sys_id",
		RequestItems: []catalog.RequestItem{
			{
				SysID:  "req_item_sys_id",
				Number: "RITM0012345",
				State:  "pending",
				Stage:  "request_approved",
			},
		},
		Message: "Order submitted successfully",
	}
	
	if !orderResult.Success {
		t.Error("Expected Success to be true")
	}
	
	if orderResult.RequestNumber != "REQ0012345" {
		t.Errorf("Expected RequestNumber 'REQ0012345', got '%s'", orderResult.RequestNumber)
	}
	
	if len(orderResult.RequestItems) != 1 {
		t.Errorf("Expected 1 request item, got %d", len(orderResult.RequestItems))
	}
}

func TestRequest(t *testing.T) {
	request := catalog.Request{
		SysID:               "request_sys_id",
		Number:              "REQ0012345",
		State:               "in_process",
		Stage:               "request_approved",
		RequestedBy:         "user1",
		RequestedFor:        "user2",
		OpenedBy:            "user1",
		Description:         "Test request",
		ShortDescription:    "Test",
		Justification:       "Business need",
		SpecialInstructions: "Handle with care",
		Price:               "100.00",
		Priority:            "3",
		Urgency:             "3",
		Impact:              "3",
		ApprovalState:       "approved",
	}
	
	if request.Number != "REQ0012345" {
		t.Errorf("Expected Number 'REQ0012345', got '%s'", request.Number)
	}
	
	if request.State != "in_process" {
		t.Errorf("Expected State 'in_process', got '%s'", request.State)
	}
}

func TestRequestItem(t *testing.T) {
	requestItem := catalog.RequestItem{
		SysID:            "req_item_sys_id",
		Number:           "RITM0012345",
		State:            "pending",
		Stage:            "fulfillment",
		RequestSysID:     "request_sys_id",
		CatalogItemSysID: "catalog_item_sys_id",
		Quantity:         1,
		Price:            "50.00",
		RecurringPrice:   "25.00",
		RequestedBy:      "user1",
		RequestedFor:     "user2",
		Variables: map[string]interface{}{
			"configuration": "standard",
			"priority":      "normal",
		},
		ApprovalState:       "approved",
		FulfillmentGroup:    "it_group",
		AssignedTo:          "tech1",
		BusinessService:     "email_service",
		DeliveryAddress:     "Building A, Floor 2",
		SpecialInstructions: "Install during maintenance window",
	}
	
	if requestItem.Number != "RITM0012345" {
		t.Errorf("Expected Number 'RITM0012345', got '%s'", requestItem.Number)
	}
	
	if requestItem.Quantity != 1 {
		t.Errorf("Expected Quantity 1, got %d", requestItem.Quantity)
	}
	
	if requestItem.Variables["configuration"] != "standard" {
		t.Errorf("Expected configuration 'standard', got '%v'", requestItem.Variables["configuration"])
	}
}

func TestCatalogTask(t *testing.T) {
	task := catalog.CatalogTask{
		SysID:            "task_sys_id",
		Number:           "SCTASK0012345",
		State:            "in_progress",
		RequestItemSysID: "req_item_sys_id",
		ShortDescription: "Install software",
		Description:      "Install and configure software package",
		AssignedTo:       "tech1",
		AssignmentGroup:  "software_team",
		OpenedBy:         "system",
		Priority:         "3",
		WorkNotes:        "Installation started",
		CloseNotes:       "",
	}
	
	if task.Number != "SCTASK0012345" {
		t.Errorf("Expected Number 'SCTASK0012345', got '%s'", task.Number)
	}
	
	if task.State != "in_progress" {
		t.Errorf("Expected State 'in_progress', got '%s'", task.State)
	}
}

func TestPriceEstimate(t *testing.T) {
	estimate := catalog.PriceEstimate{
		ItemSysID:      "item_sys_id",
		Quantity:       2,
		BasePrice:      25.00,
		RecurringPrice: 10.00,
		TotalPrice:     50.00,
		TotalRecurring: 20.00,
		Currency:       "USD",
		Variables: map[string]interface{}{
			"service_level": "premium",
		},
		EstimatedDate: "2024-01-15",
	}
	
	if estimate.Quantity != 2 {
		t.Errorf("Expected Quantity 2, got %d", estimate.Quantity)
	}
	
	if estimate.TotalPrice != 50.00 {
		t.Errorf("Expected TotalPrice 50.00, got %f", estimate.TotalPrice)
	}
	
	if estimate.Currency != "USD" {
		t.Errorf("Expected Currency 'USD', got '%s'", estimate.Currency)
	}
}

func TestValidationError(t *testing.T) {
	error := catalog.ValidationError{
		Variable: "cpu_type",
		Message:  "CPU type is required",
		Type:     "missing_mandatory",
	}
	
	if error.Variable != "cpu_type" {
		t.Errorf("Expected Variable 'cpu_type', got '%s'", error.Variable)
	}
	
	if error.Type != "missing_mandatory" {
		t.Errorf("Expected Type 'missing_mandatory', got '%s'", error.Type)
	}
}

func TestVariableChoice(t *testing.T) {
	choice := catalog.VariableChoice{
		Value:     "option1",
		Text:      "Option 1",
		Dependent: "parent_choice",
		Order:     1,
	}
	
	if choice.Value != "option1" {
		t.Errorf("Expected Value 'option1', got '%s'", choice.Value)
	}
	
	if choice.Text != "Option 1" {
		t.Errorf("Expected Text 'Option 1', got '%s'", choice.Text)
	}
}

func TestCatalogItemWithVariables(t *testing.T) {
	item := catalog.CatalogItem{
		SysID:            "item_sys_id",
		Name:             "Software Package",
		ShortDescription: "Software installation",
		Variables: []catalog.CatalogVariable{
			{
				SysID:     "var1_sys_id",
				Name:      "cpu_type",
				Question:  "Select CPU type",
				Type:      "choice",
				Mandatory: true,
				Active:    true,
				Choices: []catalog.VariableChoice{
					{Value: "intel_i5", Text: "Intel i5", Order: 1},
					{Value: "intel_i7", Text: "Intel i7", Order: 2},
					{Value: "amd_ryzen", Text: "AMD Ryzen", Order: 3},
				},
			},
			{
				SysID:        "var2_sys_id",
				Name:         "memory_size",
				Question:     "Select memory size",
				Type:         "string",
				Mandatory:    true,
				Active:       true,
				DefaultValue: "8gb",
			},
		},
	}
	
	if len(item.Variables) != 2 {
		t.Errorf("Expected 2 variables, got %d", len(item.Variables))
	}
	
	// Test first variable (choice type)
	cpuVar := item.Variables[0]
	if cpuVar.Name != "cpu_type" {
		t.Errorf("Expected variable name 'cpu_type', got '%s'", cpuVar.Name)
	}
	
	if len(cpuVar.Choices) != 3 {
		t.Errorf("Expected 3 choices, got %d", len(cpuVar.Choices))
	}
	
	if cpuVar.Choices[0].Value != "intel_i5" {
		t.Errorf("Expected first choice 'intel_i5', got '%s'", cpuVar.Choices[0].Value)
	}
	
	// Test second variable (string type)
	memVar := item.Variables[1]
	if memVar.Name != "memory_size" {
		t.Errorf("Expected variable name 'memory_size', got '%s'", memVar.Name)
	}
	
	if memVar.DefaultValue != "8gb" {
		t.Errorf("Expected default value '8gb', got '%s'", memVar.DefaultValue)
	}
}

func TestRequestWithItems(t *testing.T) {
	request := catalog.Request{
		SysID:  "request_sys_id",
		Number: "REQ0012345",
		State:  "in_process",
		RequestItems: []catalog.RequestItem{
			{
				SysID:            "item1_sys_id",
				Number:           "RITM0012345",
				State:            "pending",
				CatalogItemSysID: "catalog_item1",
				Quantity:         1,
			},
			{
				SysID:            "item2_sys_id",
				Number:           "RITM0012346",
				State:            "approved",
				CatalogItemSysID: "catalog_item2",
				Quantity:         2,
			},
		},
	}
	
	if len(request.RequestItems) != 2 {
		t.Errorf("Expected 2 request items, got %d", len(request.RequestItems))
	}
	
	if request.RequestItems[0].Number != "RITM0012345" {
		t.Errorf("Expected first item number 'RITM0012345', got '%s'", request.RequestItems[0].Number)
	}
	
	if request.RequestItems[1].Quantity != 2 {
		t.Errorf("Expected second item quantity 2, got %d", request.RequestItems[1].Quantity)
	}
}

func TestRequestItemWithTasks(t *testing.T) {
	requestItem := catalog.RequestItem{
		SysID:  "req_item_sys_id",
		Number: "RITM0012345",
		State:  "pending",
		Tasks: []catalog.CatalogTask{
			{
				SysID:            "task1_sys_id",
				Number:           "SCTASK0012345",
				State:            "assigned",
				ShortDescription: "Setup hardware",
				AssignedTo:       "tech1",
			},
			{
				SysID:            "task2_sys_id",
				Number:           "SCTASK0012346",
				State:            "in_progress",
				ShortDescription: "Install software",
				AssignedTo:       "tech2",
			},
		},
	}
	
	if len(requestItem.Tasks) != 2 {
		t.Errorf("Expected 2 tasks, got %d", len(requestItem.Tasks))
	}
	
	if requestItem.Tasks[0].Number != "SCTASK0012345" {
		t.Errorf("Expected first task number 'SCTASK0012345', got '%s'", requestItem.Tasks[0].Number)
	}
	
	if requestItem.Tasks[1].State != "in_progress" {
		t.Errorf("Expected second task state 'in_progress', got '%s'", requestItem.Tasks[1].State)
	}
}