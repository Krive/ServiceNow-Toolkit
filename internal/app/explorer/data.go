package explorer

import (
	"fmt"
	"strconv"
	"strings"
	"errors"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

// Load main menu
func (m *Model) loadMainMenu() {
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
func (m *Model) loadTableList() {
	items := []list.Item{
		// Always keep custom table input at the top
		simpleItem{title: "🔧 Custom Table", desc: "Enter custom table name", id: "custom_table"},
	}

	if m.showingBookmarks {
		// Show only bookmarked tables
		if m.configManager != nil {
			bookmarks := m.configManager.GetBookmarks()
			if len(bookmarks) > 0 {
				// Add section header
				items = append(items, simpleItem{title: "━━━ Bookmarked Tables ━━━", desc: "", id: "bookmarks_header"})
				
				for _, bookmark := range bookmarks {
					title := bookmark.TableName + " ⭐"
					desc := bookmark.DisplayName
					items = append(items, simpleItem{
						title: title,
						desc:  desc,
						id:    bookmark.TableName,
					})
				}
			}
		}
		
		// Add default tables that are bookmarked
		if m.configManager != nil {
			defaultTables := []struct{ name, desc string }{
				{"incident", "Service incidents"},
				{"problem", "Problem records"},
				{"change_request", "Change requests"},
				{"sys_user", "System users"},
				{"sys_user_group", "User groups"},
				{"cmdb_ci_server", "Server CIs"},
				{"cmdb_ci_computer", "Computer CIs"},
				{"sc_request", "Service requests"},
				{"kb_knowledge", "Knowledge articles"},
			}
			
			hasBookmarkedDefaults := false
			for _, table := range defaultTables {
				if m.configManager.IsBookmarked(table.name) {
					if !hasBookmarkedDefaults {
						// Add section header only if we have bookmarked defaults
						if len(items) <= 1 { // Only custom table exists
							items = append(items, simpleItem{title: "━━━ Bookmarked Tables ━━━", desc: "", id: "bookmarks_header"})
						}
						hasBookmarkedDefaults = true
					}
					title := table.name + " ⭐"
					items = append(items, simpleItem{
						title: title,
						desc:  table.desc,
						id:    table.name,
					})
				}
			}
		}
		
		m.list.Title = "ServiceNow Tables - Bookmarks Only"
	} else {
		// Show normal view (recent tables + defaults)
		// Add recent tables from config (max 10)
		if m.configManager != nil {
			recentTables := m.configManager.GetRecentTables()
			if len(recentTables) > 0 {
				for _, recent := range recentTables {
					// Add bookmark indicator if table is bookmarked
					title := recent.TableName
					desc := recent.DisplayName
					if m.configManager.IsBookmarked(recent.TableName) {
						title += " ⭐"
					}
					items = append(items, simpleItem{
						title: title, 
						desc:  desc, 
						id:    recent.TableName,
					})
				}
			}
		}

		// Always add default popular tables (avoid duplicates with recent tables)
		defaultTables := []struct{ name, desc string }{
			{"incident", "Service incidents"},
			{"problem", "Problem records"},
			{"change_request", "Change requests"},
			{"sys_user", "System users"},
			{"sys_user_group", "User groups"},
			{"cmdb_ci_server", "Server CIs"},
			{"cmdb_ci_computer", "Computer CIs"},
			{"sc_request", "Service requests"},
			{"kb_knowledge", "Knowledge articles"},
		}
		
		// Build set of recent table names to avoid duplicates
		recentTableNames := make(map[string]bool)
		if m.configManager != nil {
			recentTables := m.configManager.GetRecentTables()
			for _, recent := range recentTables {
				recentTableNames[recent.TableName] = true
			}
		}
		
		for _, table := range defaultTables {
			// Skip if already in recent tables
			if recentTableNames[table.name] {
				continue
			}
			
			title := table.name
			if m.configManager != nil && m.configManager.IsBookmarked(table.name) {
				title += " ⭐"
			}
			items = append(items, simpleItem{
				title: title,
				desc:  table.desc,
				id:    table.name,
			})
		}
		
		m.list.Title = "ServiceNow Tables"
	}

	// Always add back button at the end
	items = append(items, simpleItem{title: "← Back to Main Menu", desc: "Return to main menu", id: "back"})

	m.list.SetItems(items)
}

// Load table records command - returns a command instead of directly loading
func (m *Model) loadTableRecordsCmd(tableName string) tea.Cmd {
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
func (m *Model) loadTableRecordsWithQueryCmd(tableName, query string) tea.Cmd {
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
func (m *Model) loadRealRecordsWithQuerySync(tableName, query string) tea.Msg {
	// Add table to recent history
	if m.configManager != nil {
		m.configManager.AddRecentTable(tableName, tableName) // TODO: Get proper display name
	}
	
	// Validate query before making API call
	if validationErr := m.validateRawQuery(query); validationErr != nil {
		return recordsErrorMsg{err: validationErr}
	}
	
	offset := m.currentPage * m.pageSize

	// Build field list based on selected columns
	fieldsList := m.buildFieldsList()
	
	// Build query parameters with user-provided filter
	params := map[string]string{
		"sysparm_limit":         fmt.Sprintf("%d", m.pageSize),
		"sysparm_offset":        fmt.Sprintf("%d", offset),
		"sysparm_fields":        fieldsList,
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
			// Extract the query part before any ORDERBY clause for counting
			queryPart := query
			if idx := strings.Index(query, "ORDERBY"); idx > 0 {
				queryPart = strings.TrimSpace(query[:idx])
				// Remove trailing ^ if present
				queryPart = strings.TrimSuffix(queryPart, "^")
			}

			// Use the new raw query method to get accurate count
			if actualTotal, err := aggClient.CountRecordsWithRawQuery(queryPart); err == nil && actualTotal > 0 {
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

// Load demo records synchronously and return message
func (m *Model) loadDemoRecordsSync(tableName string) tea.Msg {
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
func (m *Model) loadRealRecordsSync(tableName string) tea.Msg {
	// Add table to recent history
	if m.configManager != nil {
		m.configManager.AddRecentTable(tableName, tableName) // TODO: Get proper display name
	}
	
	// Calculate offset for current page
	offset := m.currentPage * m.pageSize

	// Build field list based on selected columns
	fieldsList := m.buildFieldsList()
	
	// Build query parameters for ServiceNow API with sorting
	params := map[string]string{
		"sysparm_limit":         fmt.Sprintf("%d", m.pageSize),
		"sysparm_offset":        fmt.Sprintf("%d", offset),
		"sysparm_fields":        fieldsList,
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
func (m *Model) loadRecordXMLCmd(recordID string) tea.Cmd {
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
func (m *Model) loadCompleteRecordXML(recordID string) tea.Msg {
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
func (m *Model) processLoadedRecords(records []map[string]interface{}, total int) {
	m.totalRecords = total
	m.totalPages = (total + m.pageSize - 1) / m.pageSize

	// Store records for later access (for XML view)
	m.records = records

	// Initialize advanced filtering components (including table metadata) when records are loaded
	// This ensures metadata is available for column customization
	m.initializeAdvancedFiltering()

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

// validateRawQuery performs basic validation on raw query strings
func (m *Model) validateRawQuery(query string) error {
	if strings.TrimSpace(query) == "" {
		return nil // Empty query is valid
	}
	
	// Check for basic SQL injection patterns  
	lowerQuery := strings.ToLower(query)
	sqlPatterns := []string{
		"drop table", "delete from", "update set", "insert into",
		"exec(", "execute(", "sp_", "xp_", "union select",
		"script>", "<script", "javascript:", "vbscript:",
	}
	
	for _, pattern := range sqlPatterns {
		if strings.Contains(lowerQuery, pattern) {
			return errors.New("query contains potentially dangerous patterns")
		}
	}
	
	// Check for unbalanced parentheses
	parenCount := 0
	for _, char := range query {
		if char == '(' {
			parenCount++
		} else if char == ')' {
			parenCount--
			if parenCount < 0 {
				return errors.New("query has unbalanced parentheses")
			}
		}
	}
	if parenCount != 0 {
		return errors.New("query has unbalanced parentheses")
	}
	
	// Check for consecutive operators (basic syntax check)
	if strings.Contains(query, "^^") {
		return errors.New("query contains consecutive logical operators")
	}
	
	// Check query length
	if len(query) > 8000 {
		return errors.New("query is too long (maximum 8000 characters)")
	}
	
	// Check for malformed operators
	invalidPatterns := []string{
		"=^", "^=", "!=^", "^!=", "LIKE^", "^LIKE",
	}
	
	for _, pattern := range invalidPatterns {
		if strings.Contains(query, pattern) {
			return errors.New("query contains malformed operators")
		}
	}
	
	return nil
}

// Load table records with sorting
func (m *Model) loadTableRecordsWithSortCmd(tableName, query, sortColumn, sortDirection string) tea.Cmd {
	return func() tea.Msg {
		if m.client == nil {
			// Demo mode - return sorted demo data
			return m.loadDemoRecordsSortedSync(tableName, sortColumn, sortDirection)
		} else {
			// Real mode - load from ServiceNow with sorting
			return m.loadRealRecordsWithSortSync(tableName, query, sortColumn, sortDirection)
		}
	}
}

// Load real records from ServiceNow with sorting
func (m *Model) loadRealRecordsWithSortSync(tableName, query, sortColumn, sortDirection string) tea.Msg {
	// Add table to recent history
	if m.configManager != nil {
		m.configManager.AddRecentTable(tableName, tableName) // TODO: Get proper display name
	}
	
	// Calculate offset
	offset := m.currentPage * m.pageSize

	// Build query with sorting
	orderBy := fmt.Sprintf("ORDERBY%s%s", strings.ToUpper(sortDirection), sortColumn)
	
	finalQuery := query
	if finalQuery != "" {
		finalQuery = fmt.Sprintf("%s^%s", finalQuery, orderBy)
	} else {
		finalQuery = orderBy
	}

	params := map[string]string{
		"sysparm_limit":         fmt.Sprintf("%d", m.pageSize),
		"sysparm_offset":        fmt.Sprintf("%d", offset),
		"sysparm_display_value": "all",
		"sysparm_query":         finalQuery,
	}

	// Add fields if columns are selected
	if len(m.selectedColumns) > 0 {
		fieldsList := strings.Join(m.selectedColumns, ",")
		params["sysparm_fields"] = fieldsList
	}

	records, err := m.client.Table(tableName).List(params)
	if err != nil {
		return recordsErrorMsg{err: err}
	}

	// Try to get exact count using aggregate API (only on first page to avoid repeated calls)
	total := len(records)
	if m.currentPage == 0 {
		if aggClient := m.client.Aggregate(tableName); aggClient != nil {
			// Extract the query part before any ORDERBY clause for counting
			queryPart := query
			if queryPart == "" {
				queryPart = ""
			}
			
			if actualTotal, err := aggClient.CountRecords(nil); err == nil && actualTotal > 0 {
				total = actualTotal
			}
		}
	}

	return recordsLoadedMsg{
		records: records,
		total:   total,
	}
}

// Load demo records with sorting (for demo mode)
func (m *Model) loadDemoRecordsSortedSync(tableName, sortColumn, sortDirection string) tea.Msg {
	// Use the same logic as loadDemoRecordsSync but with sorting
	total := 100 // Demo total
	
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
	
	// Simple sorting for demo data (just reverse order for desc)
	if sortDirection == "desc" {
		for i, j := 0, len(records)-1; i < j; i, j = i+1, j-1 {
			records[i], records[j] = records[j], records[i]
		}
	}
	
	return recordsLoadedMsg{
		records: records,
		total:   total,
	}
}