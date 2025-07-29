package unit

import (
	"testing"
	"time"

	"github.com/Krive/ServiceNow-Toolkit/pkg/servicenow/cmdb"
	"github.com/Krive/ServiceNow-Toolkit/pkg/servicenow/core"
)

func TestCMDBClient_NewCMDBClient(t *testing.T) {
	client := &core.Client{} // Mock client
	cmdbClient := cmdb.NewCMDBClient(client)
	
	if cmdbClient == nil {
		t.Fatal("NewCMDBClient should return a non-nil CMDBClient")
	}
}

func TestCMDBClient_SubClients(t *testing.T) {
	client := &core.Client{}
	cmdbClient := cmdb.NewCMDBClient(client)
	
	// Test relationship client
	relClient := cmdbClient.NewRelationshipClient()
	if relClient == nil {
		t.Error("NewRelationshipClient should return a non-nil RelationshipClient")
	}
	
	// Test class client
	classClient := cmdbClient.NewClassClient()
	if classClient == nil {
		t.Error("NewClassClient should return a non-nil ClassClient")
	}
	
	// Test identification client
	identClient := cmdbClient.NewIdentificationClient()
	if identClient == nil {
		t.Error("NewIdentificationClient should return a non-nil IdentificationClient")
	}
	
	// Test reconciliation client
	reconClient := cmdbClient.NewReconciliationClient()
	if reconClient == nil {
		t.Error("NewReconciliationClient should return a non-nil ReconciliationClient")
	}
}

func TestConfigurationItem(t *testing.T) {
	testCI := cmdb.ConfigurationItem{
		SysID:                 "ci_sys_id",
		Name:                  "Test Server",
		SysClassName:          "cmdb_ci_server",
		State:                 "installed",
		OperationalStatus:     "operational",
		Category:              "hardware",
		Subcategory:           "server",
		Environment:           "production",
		Location:              "datacenter_1",
		Owner:                 "admin",
		SupportGroup:          "server_team",
		AssignedTo:            "john_doe",
		SerialNumber:          "SN123456",
		AssetTag:              "AT789012",
		ModelID:               "model_123",
		ModelNumber:           "DL380",
		Manufacturer:          "HPE",
		Vendor:                "HPE",
		ShortDescription:      "Production web server",
		Description:           "Main production web server for the application",
		IPAddress:             "192.168.1.100",
		MacAddress:            "00:11:22:33:44:55",
		FQDN:                  "webserver01.company.com",
		DNSDomain:             "company.com",
		OSName:                "Red Hat Enterprise Linux",
		OSVersion:             "8.4",
		CPUCount:              8,
		CPUSpeed:              "2.4 GHz",
		CPUType:               "Intel Xeon",
		RAM:                   "32 GB",
		DiskSpace:             "500 GB",
		CostCenter:            "IT_001",
		BusinessService:       "Web Application",
		Application:           "E-commerce Platform",
		CreatedBy:             "discovery",
		CreatedOn:             time.Now(),
		UpdatedBy:             "admin",
		UpdatedOn:             time.Now(),
		Attributes: map[string]interface{}{
			"custom_field1": "value1",
			"custom_field2": "value2",
		},
	}
	
	// Test basic properties
	if testCI.SysID != "ci_sys_id" {
		t.Errorf("Expected SysID 'ci_sys_id', got '%s'", testCI.SysID)
	}
	
	if testCI.Name != "Test Server" {
		t.Errorf("Expected Name 'Test Server', got '%s'", testCI.Name)
	}
	
	if testCI.SysClassName != "cmdb_ci_server" {
		t.Errorf("Expected SysClassName 'cmdb_ci_server', got '%s'", testCI.SysClassName)
	}
	
	if testCI.CPUCount != 8 {
		t.Errorf("Expected CPUCount 8, got %d", testCI.CPUCount)
	}
	
	if testCI.IPAddress != "192.168.1.100" {
		t.Errorf("Expected IPAddress '192.168.1.100', got '%s'", testCI.IPAddress)
	}
	
	if len(testCI.Attributes) != 2 {
		t.Errorf("Expected 2 attributes, got %d", len(testCI.Attributes))
	}
	
	if testCI.Attributes["custom_field1"] != "value1" {
		t.Errorf("Expected custom_field1 'value1', got '%v'", testCI.Attributes["custom_field1"])
	}
}

func TestCIRelationship(t *testing.T) {
	testRel := cmdb.CIRelationship{
		SysID:             "rel_sys_id",
		Parent:            "parent_ci_sys_id",
		Child:             "child_ci_sys_id",
		Type:              "depends_on",
		ConnectionType:    "network",
		ParentDescriptor:  "Database Server",
		ChildDescriptor:   "Web Server",
		Direction:         "outbound",
		Weight:            5,
		CreatedBy:         "admin",
		CreatedOn:         time.Now(),
		UpdatedBy:         "admin",
		UpdatedOn:         time.Now(),
		Attributes: map[string]interface{}{
			"relationship_strength": "high",
			"criticality":          "critical",
		},
	}
	
	if testRel.SysID != "rel_sys_id" {
		t.Errorf("Expected SysID 'rel_sys_id', got '%s'", testRel.SysID)
	}
	
	if testRel.Type != "depends_on" {
		t.Errorf("Expected Type 'depends_on', got '%s'", testRel.Type)
	}
	
	if testRel.Weight != 5 {
		t.Errorf("Expected Weight 5, got %d", testRel.Weight)
	}
	
	if testRel.Direction != "outbound" {
		t.Errorf("Expected Direction 'outbound', got '%s'", testRel.Direction)
	}
	
	if len(testRel.Attributes) != 2 {
		t.Errorf("Expected 2 attributes, got %d", len(testRel.Attributes))
	}
}

func TestCIClass(t *testing.T) {
	testClass := cmdb.CIClass{
		SysID:       "class_sys_id",
		Name:        "cmdb_ci_server",
		Label:       "Server",
		SuperClass:  "cmdb_ci_computer",
		TableName:   "cmdb_ci_server",
		Package:     "com.glide.cmdb",
		Active:      true,
		Abstract:    false,
		Extensible:  true,
		NumberPrefix: "SRV",
		Attributes:  []string{"cpu_count", "ram", "disk_space", "os"},
		Description: "Server configuration item class",
		CreatedBy:   "system",
		CreatedOn:   time.Now(),
		UpdatedBy:   "admin",
		UpdatedOn:   time.Now(),
	}
	
	if testClass.Name != "cmdb_ci_server" {
		t.Errorf("Expected Name 'cmdb_ci_server', got '%s'", testClass.Name)
	}
	
	if testClass.Label != "Server" {
		t.Errorf("Expected Label 'Server', got '%s'", testClass.Label)
	}
	
	if testClass.SuperClass != "cmdb_ci_computer" {
		t.Errorf("Expected SuperClass 'cmdb_ci_computer', got '%s'", testClass.SuperClass)
	}
	
	if !testClass.Active {
		t.Error("Expected Active to be true")
	}
	
	if testClass.Abstract {
		t.Error("Expected Abstract to be false")
	}
	
	if !testClass.Extensible {
		t.Error("Expected Extensible to be true")
	}
	
	if len(testClass.Attributes) != 4 {
		t.Errorf("Expected 4 attributes, got %d", len(testClass.Attributes))
	}
	
	expectedAttrs := []string{"cpu_count", "ram", "disk_space", "os"}
	for i, attr := range expectedAttrs {
		if i < len(testClass.Attributes) && testClass.Attributes[i] != attr {
			t.Errorf("Expected attribute %d to be '%s', got '%s'", i, attr, testClass.Attributes[i])
		}
	}
}

func TestCIDependencyMap(t *testing.T) {
	rootCI := &cmdb.ConfigurationItem{
		SysID: "root_ci",
		Name:  "Root Server",
	}
	
	dependency1 := &cmdb.ConfigurationItem{
		SysID: "dep1",
		Name:  "Database Server",
	}
	
	dependency2 := &cmdb.ConfigurationItem{
		SysID: "dep2",
		Name:  "Storage Array",
	}
	
	dependent1 := &cmdb.ConfigurationItem{
		SysID: "dept1",
		Name:  "Web Server",
	}
	
	dependent2 := &cmdb.ConfigurationItem{
		SysID: "dept2",
		Name:  "Application Server",
	}
	
	relationship1 := &cmdb.CIRelationship{
		SysID:  "rel1",
		Parent: "dep1",
		Child:  "root_ci",
		Type:   "depends_on",
	}
	
	relationship2 := &cmdb.CIRelationship{
		SysID:  "rel2",
		Parent: "root_ci",
		Child:  "dept1",
		Type:   "supports",
	}
	
	depMap := cmdb.CIDependencyMap{
		RootCI:        rootCI,
		Dependencies:  []*cmdb.ConfigurationItem{dependency1, dependency2},
		Dependents:    []*cmdb.ConfigurationItem{dependent1, dependent2},
		Relationships: []*cmdb.CIRelationship{relationship1, relationship2},
		Depth:         3,
	}
	
	if depMap.RootCI.SysID != "root_ci" {
		t.Errorf("Expected RootCI SysID 'root_ci', got '%s'", depMap.RootCI.SysID)
	}
	
	if len(depMap.Dependencies) != 2 {
		t.Errorf("Expected 2 dependencies, got %d", len(depMap.Dependencies))
	}
	
	if len(depMap.Dependents) != 2 {
		t.Errorf("Expected 2 dependents, got %d", len(depMap.Dependents))
	}
	
	if len(depMap.Relationships) != 2 {
		t.Errorf("Expected 2 relationships, got %d", len(depMap.Relationships))
	}
	
	if depMap.Depth != 3 {
		t.Errorf("Expected Depth 3, got %d", depMap.Depth)
	}
}

func TestCIFilter(t *testing.T) {
	filter := cmdb.CIFilter{
		Class:             "cmdb_ci_server",
		State:             "installed",
		OperationalStatus: "operational",
		Environment:       "production",
		Location:          "datacenter_1",
		Owner:             "admin",
		SupportGroup:      "server_team",
		SerialNumber:      "SN123456",
		AssetTag:          "AT789012",
		IPAddress:         "192.168.1.100",
		FQDN:              "server.company.com",
		Name:              "web",
		CreatedAfter:      time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		CreatedBefore:     time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC),
		UpdatedAfter:      time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
		UpdatedBefore:     time.Date(2024, 6, 30, 23, 59, 59, 0, time.UTC),
		Limit:             100,
		Offset:            0,
		OrderBy:           "name",
		Fields:            []string{"sys_id", "name", "state", "ip_address"},
	}
	
	if filter.Class != "cmdb_ci_server" {
		t.Errorf("Expected Class 'cmdb_ci_server', got '%s'", filter.Class)
	}
	
	if filter.State != "installed" {
		t.Errorf("Expected State 'installed', got '%s'", filter.State)
	}
	
	if filter.Limit != 100 {
		t.Errorf("Expected Limit 100, got %d", filter.Limit)
	}
	
	if len(filter.Fields) != 4 {
		t.Errorf("Expected 4 fields, got %d", len(filter.Fields))
	}
	
	expectedFields := []string{"sys_id", "name", "state", "ip_address"}
	for i, field := range expectedFields {
		if i < len(filter.Fields) && filter.Fields[i] != field {
			t.Errorf("Expected field %d to be '%s', got '%s'", i, field, filter.Fields[i])
		}
	}
}

func TestCIIdentificationRequest(t *testing.T) {
	item1 := map[string]interface{}{
		"name":          "Test Server 1",
		"serial_number": "SN001",
		"ip_address":    "192.168.1.10",
	}
	
	item2 := map[string]interface{}{
		"name":          "Test Server 2",
		"serial_number": "SN002",
		"ip_address":    "192.168.1.11",
	}
	
	options := &cmdb.IdentificationOptions{
		MatchingAttributes: []string{"serial_number", "name", "ip_address"},
		Threshold:          0.8,
		Strategy:           "exact_match",
		ClassName:          "cmdb_ci_server",
	}
	
	request := cmdb.CIIdentificationRequest{
		Items:   []map[string]interface{}{item1, item2},
		Options: options,
	}
	
	if len(request.Items) != 2 {
		t.Errorf("Expected 2 items, got %d", len(request.Items))
	}
	
	if request.Options.Threshold != 0.8 {
		t.Errorf("Expected Threshold 0.8, got %f", request.Options.Threshold)
	}
	
	if request.Options.Strategy != "exact_match" {
		t.Errorf("Expected Strategy 'exact_match', got '%s'", request.Options.Strategy)
	}
	
	if len(request.Options.MatchingAttributes) != 3 {
		t.Errorf("Expected 3 matching attributes, got %d", len(request.Options.MatchingAttributes))
	}
}

func TestCIReconciliationRequest(t *testing.T) {
	item1 := map[string]interface{}{
		"name":          "Server A",
		"serial_number": "SN100",
		"state":         "installed",
	}
	
	item2 := map[string]interface{}{
		"name":          "Server B",
		"serial_number": "SN200",
		"state":         "installed",
	}
	
	options := &cmdb.ReconciliationOptions{
		CreateMissing:      true,
		UpdateExisting:     true,
		MatchingRules:      []string{"serial_number", "name"},
		ConflictResolution: "merge",
		DryRun:             false,
		ClassName:          "cmdb_ci_server",
	}
	
	request := cmdb.CIReconciliationRequest{
		Items:      []map[string]interface{}{item1, item2},
		DataSource: "discovery_tool",
		Options:    options,
	}
	
	if len(request.Items) != 2 {
		t.Errorf("Expected 2 items, got %d", len(request.Items))
	}
	
	if request.DataSource != "discovery_tool" {
		t.Errorf("Expected DataSource 'discovery_tool', got '%s'", request.DataSource)
	}
	
	if !request.Options.CreateMissing {
		t.Error("Expected CreateMissing to be true")
	}
	
	if !request.Options.UpdateExisting {
		t.Error("Expected UpdateExisting to be true")
	}
	
	if request.Options.ConflictResolution != "merge" {
		t.Errorf("Expected ConflictResolution 'merge', got '%s'", request.Options.ConflictResolution)
	}
}

func TestCIReconciliationResult(t *testing.T) {
	createdCI := &cmdb.ConfigurationItem{
		SysID: "new_ci_1",
		Name:  "New Server",
	}
	
	updatedCI := &cmdb.ConfigurationItem{
		SysID: "existing_ci_1",
		Name:  "Updated Server",
	}
	
	reconciliationError := cmdb.ReconciliationError{
		Item: map[string]interface{}{
			"name": "Failed Server",
		},
		Error:   "Validation failed",
		Code:    "VALIDATION_ERROR",
		Details: "Required field missing",
	}
	
	result := cmdb.CIReconciliationResult{
		Created:      []*cmdb.ConfigurationItem{createdCI},
		Updated:      []*cmdb.ConfigurationItem{updatedCI},
		Skipped:      []string{"skipped_item_1"},
		Errors:       []cmdb.ReconciliationError{reconciliationError},
		TotalItems:   4,
		SuccessCount: 2,
		ErrorCount:   1,
		SkippedCount: 1,
	}
	
	if len(result.Created) != 1 {
		t.Errorf("Expected 1 created CI, got %d", len(result.Created))
	}
	
	if len(result.Updated) != 1 {
		t.Errorf("Expected 1 updated CI, got %d", len(result.Updated))
	}
	
	if len(result.Skipped) != 1 {
		t.Errorf("Expected 1 skipped item, got %d", len(result.Skipped))
	}
	
	if len(result.Errors) != 1 {
		t.Errorf("Expected 1 error, got %d", len(result.Errors))
	}
	
	if result.TotalItems != 4 {
		t.Errorf("Expected TotalItems 4, got %d", result.TotalItems)
	}
	
	if result.SuccessCount != 2 {
		t.Errorf("Expected SuccessCount 2, got %d", result.SuccessCount)
	}
	
	if result.ErrorCount != 1 {
		t.Errorf("Expected ErrorCount 1, got %d", result.ErrorCount)
	}
	
	if result.SkippedCount != 1 {
		t.Errorf("Expected SkippedCount 1, got %d", result.SkippedCount)
	}
	
	// Test the reconciliation error
	if result.Errors[0].Code != "VALIDATION_ERROR" {
		t.Errorf("Expected error code 'VALIDATION_ERROR', got '%s'", result.Errors[0].Code)
	}
}

func TestCIMatch(t *testing.T) {
	inputItem := map[string]interface{}{
		"name":          "Test Server",
		"serial_number": "SN123",
		"ip_address":    "192.168.1.100",
	}
	
	matchedCI := &cmdb.ConfigurationItem{
		SysID:        "matched_ci_sys_id",
		Name:         "Test Server",
		SerialNumber: "SN123",
		IPAddress:    "192.168.1.100",
	}
	
	match := cmdb.CIMatch{
		InputItem: inputItem,
		MatchedCI: matchedCI,
		Score:     0.95,
		MatchedOn: []string{"name", "serial_number", "ip_address"},
	}
	
	if match.Score != 0.95 {
		t.Errorf("Expected Score 0.95, got %f", match.Score)
	}
	
	if len(match.MatchedOn) != 3 {
		t.Errorf("Expected 3 matched attributes, got %d", len(match.MatchedOn))
	}
	
	expectedMatchedOn := []string{"name", "serial_number", "ip_address"}
	for i, attr := range expectedMatchedOn {
		if i < len(match.MatchedOn) && match.MatchedOn[i] != attr {
			t.Errorf("Expected matched attribute %d to be '%s', got '%s'", i, attr, match.MatchedOn[i])
		}
	}
	
	if match.MatchedCI.SysID != "matched_ci_sys_id" {
		t.Errorf("Expected MatchedCI SysID 'matched_ci_sys_id', got '%s'", match.MatchedCI.SysID)
	}
	
	if inputItemName, ok := match.InputItem["name"].(string); !ok || inputItemName != "Test Server" {
		t.Errorf("Expected InputItem name 'Test Server', got '%v'", match.InputItem["name"])
	}
}