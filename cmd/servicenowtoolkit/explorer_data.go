package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

// Load main menu
func (m *simpleModel) loadMainMenu() {
	items := []list.Item{
		simpleItem{title: "📋 Table Browser", desc: "Browse and explore ServiceNow tables with filters and search", id: "tables"},
		simpleItem{title: "👥 Identity Management", desc: "Manage users, roles, and groups (Coming Soon)", id: "identity"},
		simpleItem{title: "🏗️ CMDB Explorer", desc: "Explore configuration items and relationships (Coming Soon)", id: "cmdb"},
		simpleItem{title: "🔍 Global Search", desc: "Search across multiple tables and records (Coming Soon)", id: "search"},
		simpleItem{title: "📊 Analytics", desc: "View reports and data analysis (Coming Soon)", id: "analytics"},
		simpleItem{title: "🛒 Service Catalog", desc: "Browse and request services (Coming Soon)", id: "catalog"},
	}

	if m.client == nil {
		items = append(items, simpleItem{title: "🎭 Demo Mode Active", desc: "Currently running without ServiceNow connection", id: "demo"})
	}

	m.list.SetItems(items)
	m.list.Title = "Main Menu"
}

// Load table list
func (m *simpleModel) loadTableList() {
	items := []list.Item{
		simpleItem{title: "🔧 Custom Table", desc: "Enter custom table name", id: "custom_table"},
		simpleItem{title: "incident", desc: "Service incidents", id: "incident"},
		simpleItem{title: "problem", desc: "Problem records", id: "problem"},
		simpleItem{title: "change_request", desc: "Change requests", id: "change_request"},
		simpleItem{title: "sys_user", desc: "System users", id: "sys_user"},
		simpleItem{title: "sys_user_group", desc: "User groups", id: "sys_user_group"},
		simpleItem{title: "cmdb_ci_server", desc: "Server CIs", id: "cmdb_ci_server"},
		simpleItem{title: "cmdb_ci_computer", desc: "Computer CIs", id: "cmdb_ci_computer"},
		simpleItem{title: "sc_request", desc: "Service requests", id: "sc_request"},
		simpleItem{title: "kb_knowledge", desc: "Knowledge articles", id: "kb_knowledge"},
		simpleItem{title: "← Back to Main Menu", desc: "Return to main menu", id: "back"},
	}

	m.list.SetItems(items)
	m.list.Title = "ServiceNow Tables"
}

// Load table records command - returns a command instead of directly loading
func (m *simpleModel) loadTableRecordsCmd(tableName string) tea.Cmd {
	return func() tea.Msg {
		if m.client == nil {
			// Demo mode - generate paginated mock data
			return m.loadDemoRecordsSync(tableName)
		} else {
			// Real mode - load from ServiceNow with pagination
			return m.loadRealRecordsSync(tableName)
		}
	}
}

// Load table records with query/filter command
func (m *simpleModel) loadTableRecordsWithQueryCmd(tableName, query string) tea.Cmd {
	return func() tea.Msg {
		if m.client == nil {
			// Demo mode - just return filtered demo data
			return m.loadDemoRecordsSync(tableName)
		} else {
			// Real mode - load from ServiceNow with query filter
			return m.loadRealRecordsWithQuerySync(tableName, query)
		}
	}
}

// Load real records from ServiceNow with query filter
func (m *simpleModel) loadRealRecordsWithQuerySync(tableName, query string) tea.Msg {
	offset := m.currentPage * m.pageSize

	// Build query parameters with user-provided filter
	params := map[string]string{
		"sysparm_limit":         fmt.Sprintf("%d", m.pageSize),
		"sysparm_offset":        fmt.Sprintf("%d", offset),
		"sysparm_fields":        "sys_id,number,name,title,display_name,short_description,description,state,priority,category,assigned_to,assignment_group,caller_id,user_name,email,active,department,sys_updated_on,sys_created_on",
		"sysparm_query":         query, // Use the user's query directly
		"sysparm_display_value": "all", // Get display values for reference fields
	}

	// Real ServiceNow API call with query
	records, err := m.client.Table(tableName).List(params)
	if err != nil {
		return recordsErrorMsg{err: err}
	}

	// Get actual total records using aggregate API with query
	// Proper fallback estimate based on loaded records
	total := offset + len(records)
	if len(records) == m.pageSize {
		// Got a full page, assume there might be more
		total = offset + len(records) + 1 // At least one more record exists
	}

	// Try to get exact count using aggregate API with the filter (only on first page)
	if m.currentPage == 0 {
		if aggClient := m.client.Aggregate(tableName); aggClient != nil {
			// Create a basic query builder with the raw query
			// This is a simplified approach - in production you'd want proper parsing
			aq := aggClient.NewQuery().CountAll("filtered_count")

			// Apply the query filter if it's a simple condition
			if strings.Contains(query, "=") && !strings.Contains(query, "ORDERBY") {
				// Extract the query part before any ORDERBY clause
				queryPart := query
				if idx := strings.Index(query, "ORDERBY"); idx > 0 {
					queryPart = strings.TrimSpace(query[:idx])
					// Remove trailing ^ if present
					queryPart = strings.TrimSuffix(queryPart, "^")
				}

				// For simple field=value queries, apply them
				if queryPart != "" && queryPart != "ORDERBYDESCsys_updated_on" {
					parts := strings.SplitN(queryPart, "=", 2)
					if len(parts) == 2 {
						field := strings.TrimSpace(parts[0])
						value := strings.TrimSpace(parts[1])
						aq.Equals(field, value)
					}
				}
			}

			if result, err := aq.Execute(); err == nil && result.Stats != nil {
				if countVal, ok := result.Stats["filtered_count"]; ok {
					if actualTotal := parseIntFromInterface(countVal); actualTotal > 0 {
						total = actualTotal
					}
				}
			}
			// If aggregate fails, keep the estimated total as fallback
		}
	} else {
		// On subsequent pages, use previously calculated total if it seems reasonable
		if m.totalRecords > offset+len(records) {
			total = m.totalRecords
		}
	}

	return recordsLoadedMsg{
		records: records,
		total:   total,
	}
}

// Load demo records synchronously and return message
func (m *simpleModel) loadDemoRecordsSync(tableName string) tea.Msg {
	// Simulate total records (varies by table)
	totalRecords := map[string]int{
		"incident":         347,
		"problem":          89,
		"change_request":   156,
		"sys_user":         892,
		"sys_user_group":   45,
		"cmdb_ci_server":   234,
		"cmdb_ci_computer": 567,
		"sc_request":       123,
		"kb_knowledge":     78,
	}

	total := totalRecords[tableName]
	if total == 0 {
		total = 50 // Default for unknown tables
	}

	// Generate current page records
	var records []map[string]interface{}
	startRecord := m.currentPage*m.pageSize + 1
	endRecord := startRecord + m.pageSize - 1
	if endRecord > total {
		endRecord = total
	}

	for i := startRecord; i <= endRecord; i++ {
		record := m.generateDemoRecord(tableName, i)
		records = append(records, record)
	}

	return recordsLoadedMsg{
		records: records,
		total:   total,
	}
}

// Load real records from ServiceNow synchronously
func (m *simpleModel) loadRealRecordsSync(tableName string) tea.Msg {
	// Calculate offset for current page
	offset := m.currentPage * m.pageSize

	// Build query parameters for ServiceNow API with sorting
	params := map[string]string{
		"sysparm_limit":         fmt.Sprintf("%d", m.pageSize),
		"sysparm_offset":        fmt.Sprintf("%d", offset),
		"sysparm_fields":        "sys_id,number,name,title,display_name,short_description,description,state,priority,category,assigned_to,assignment_group,caller_id,user_name,email,active,department,sys_updated_on,sys_created_on",
		"sysparm_query":         "ORDERBYDESCsys_updated_on", // Sort by sys_updated_on descending
		"sysparm_display_value": "all",                       // Get display values for reference fields
	}

	// Real ServiceNow API call
	records, err := m.client.Table(tableName).List(params)
	if err != nil {
		return recordsErrorMsg{err: err}
	}

	// Get actual total records using aggregate API
	// Proper fallback estimate based on loaded records
	total := offset + len(records)
	if len(records) == m.pageSize {
		// Got a full page, assume there might be more
		total = offset + len(records) + 1 // At least one more record exists
	}

	// Try to get exact count using aggregate API (only on first page to avoid repeated calls)
	if m.currentPage == 0 {
		if aggClient := m.client.Aggregate(tableName); aggClient != nil {
			if actualTotal, err := aggClient.CountRecords(nil); err == nil && actualTotal > 0 {
				total = actualTotal
			}
			// If aggregate fails, keep the estimated total as fallback
		}
	} else {
		// On subsequent pages, use previously calculated total if it seems reasonable
		if m.totalRecords > offset+len(records) {
			total = m.totalRecords
		}
	}

	return recordsLoadedMsg{
		records: records,
		total:   total,
	}
}

// Load record XML command
func (m *simpleModel) loadRecordXMLCmd(recordID string) tea.Cmd {
	return func() tea.Msg {
		if m.client == nil {
			// Demo mode - simple XML generation
			xml := generateXMLFromRecord(m.selectedRecord, m.currentTable, recordID)
			return recordXMLLoadedMsg{xml: xml}
		}

		// Real mode - fetch complete record for XML representation
		return m.loadCompleteRecordXML(recordID)
	}
}

// Load complete record from ServiceNow for XML display
func (m *simpleModel) loadCompleteRecordXML(recordID string) tea.Msg {
	// Fetch the complete record with all fields
	params := map[string]string{
		"sysparm_query":         fmt.Sprintf("sys_id=%s", recordID),
		"sysparm_limit":         "1",
		"sysparm_display_value": "all", // Get display values for reference fields
		// Don't limit fields - get all fields for complete XML
	}

	records, err := m.client.Table(m.currentTable).List(params)
	if err != nil {
		return recordsErrorMsg{err: fmt.Errorf("failed to load complete record: %w", err)}
	}

	if len(records) == 0 {
		return recordsErrorMsg{err: fmt.Errorf("record not found: %s", recordID)}
	}

	// Generate XML from the complete record
	xml := generateXMLFromRecord(records[0], m.currentTable, recordID)
	return recordXMLLoadedMsg{xml: xml}
}

// Process loaded records and update UI
func (m *simpleModel) processLoadedRecords(records []map[string]interface{}, total int) {
	m.totalRecords = total
	m.totalPages = (total + m.pageSize - 1) / m.pageSize

	// Store records for later access (for XML view)
	m.records = records

	// Convert records to list items
	var items []list.Item
	for i, record := range records {
		title, desc := m.formatRecordDisplay(record, i)

		items = append(items, simpleItem{
			title: title,
			desc:  desc,
			id:    getRecordField(record, "sys_id"),
		})
	}

	// Add back navigation item
	items = append(items, simpleItem{
		title: "← Back to Table List",
		desc:  "Return to table selection",
		id:    "back",
	})

	m.list.SetItems(items)
	m.updateRecordTitle(m.currentTable)
}

// Helper function to parse interface{} to int
func parseIntFromInterface(value interface{}) int {
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case string:
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return 0
}