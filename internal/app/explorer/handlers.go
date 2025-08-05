package explorer

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

// Handle back navigation
func (m Model) handleBack() (tea.Model, tea.Cmd) {
	switch m.state {
	case simpleStateTableList:
		m.state = simpleStateMain
		m.loadMainMenu()
	case simpleStateTableRecords:
		m.state = simpleStateTableList
		m.currentQuery = "" // Clear any active filters when leaving table records
		m.loadTableList()
	case simpleStateRecordDetail:
		m.state = simpleStateTableRecords
		// Reload the record list (preserve any active query)
		m.loading = true
		if m.currentQuery != "" {
			return m, m.loadTableRecordsWithQueryCmd(m.currentTable, m.currentQuery)
		}
		return m, m.loadTableRecordsCmd(m.currentTable)
	case simpleStateCustomTable:
		m.state = simpleStateTableList
		m.loadTableList()
	case simpleStateQueryFilter:
		m.state = simpleStateTableRecords
		// Reload the record list (preserve any active query)
		m.loading = true
		if m.currentQuery != "" {
			return m, m.loadTableRecordsWithQueryCmd(m.currentTable, m.currentQuery)
		}
		return m, m.loadTableRecordsCmd(m.currentTable)
	case simpleStateXMLSearch:
		m.state = simpleStateRecordDetail
		m.xmlSearchQuery = ""
		m.xmlSearchResults = []int{}
		m.xmlSearchIndex = 0
	case simpleStateAdvancedFilter:
		if m.queryBuilder != nil {
			m.queryBuilder.SetActive(false)
		}
		m.state = simpleStateTableRecords
	case simpleStateFilterBrowser:
		if m.filterBrowser != nil {
			m.filterBrowser.SetActive(false)
		}
		m.state = simpleStateTableRecords
	case simpleStateColumnCustomizer:
		if m.columnCustomizer != nil {
			// Apply selected columns before going back
			m.selectedColumns = m.columnCustomizer.GetSelectedColumns()
			m.columnCustomizer.SetActive(false)
		}
		m.state = simpleStateTableRecords
		// Reload records to apply new column selection
		m.loading = true
		if m.currentQuery != "" {
			return m, m.loadTableRecordsWithQueryCmd(m.currentTable, m.currentQuery)
		}
		return m, m.loadTableRecordsCmd(m.currentTable)
	case simpleStateViewManager:
		m.state = simpleStateTableRecords
	default:
		m.state = simpleStateMain
		m.loadMainMenu()
	}
	// Update list size when state changes (header visibility may change)
	m.updateListSize()
	
	// Force a window resize message to ensure proper layout recalculation
	// This is particularly important when coming back from XML view
	var cmd tea.Cmd
	if m.state == simpleStateTableRecords {
		cmd = func() tea.Msg {
			return tea.WindowSizeMsg{Width: m.width, Height: m.height}
		}
	}
	
	return m, cmd
}

// Handle next page
func (m Model) handleNextPage() (tea.Model, tea.Cmd) {
	if m.state == simpleStateTableRecords && m.currentPage < m.totalPages-1 {
		m.currentPage++
		m.loading = true
		if m.currentQuery != "" {
			return m, m.loadTableRecordsWithQueryCmd(m.currentTable, m.currentQuery)
		}
		return m, m.loadTableRecordsCmd(m.currentTable)
	}
	return m, nil
}


// Handle previous page
func (m Model) handlePrevPage() (tea.Model, tea.Cmd) {
	if m.state == simpleStateTableRecords && m.currentPage > 0 {
		m.currentPage--
		m.loading = true
		if m.currentQuery != "" {
			return m, m.loadTableRecordsWithQueryCmd(m.currentTable, m.currentQuery)
		}
		return m, m.loadTableRecordsCmd(m.currentTable)
	}
	return m, nil
}


// Handle refresh
func (m Model) handleRefresh() (tea.Model, tea.Cmd) {
	switch m.state {
	case simpleStateTableRecords:
		m.loading = true
		if m.currentQuery != "" {
			return m, m.loadTableRecordsWithQueryCmd(m.currentTable, m.currentQuery)
		}
		return m, m.loadTableRecordsCmd(m.currentTable)
	case simpleStateTableList:
		m.loadTableList()
	case simpleStateMain:
		m.loadMainMenu()
	}
	return m, nil
}


// Handle enter key
func (m Model) handleEnter() (tea.Model, tea.Cmd) {
	switch m.state {
	case simpleStateMain:
		if item, ok := m.list.SelectedItem().(simpleItem); ok {
			switch item.id {
			case "tables":
				m.state = simpleStateTableList
				m.loadTableList()
				m.updateListSize()
			default:
				// For now, show a simple message for other features
				items := []list.Item{
					simpleItem{title: "🚧 Coming Soon", desc: fmt.Sprintf("%s feature is being implemented", item.title), id: "placeholder"},
					simpleItem{title: "← Back to Main Menu", desc: "Return to main menu", id: "back"},
				}
				m.list.SetItems(items)
			}
		}
	case simpleStateTableList:
		if item, ok := m.list.SelectedItem().(simpleItem); ok {
			if item.id == "back" {
				return m.handleBack()
			}
			if item.id == "custom_table" {
				m.state = simpleStateCustomTable
				m.updateListSize()
				return m, nil
			}
			// Skip section headers
			if item.id == "recent_header" || item.id == "popular_header" || item.id == "bookmarks_header" {
				return m, nil
			}
			m.currentTable = item.id
			m.state = simpleStateTableRecords
			m.currentPage = 0 // Reset to first page
			m.currentQuery = "" // Clear any previous filters when entering a new table
			m.loading = true  // Set loading state
			
			// Set up intelligent default columns for this table
			m.setupDefaultColumnsForTable(item.id)
			
			m.updateListSize()
			return m, m.loadTableRecordsCmd(item.id)
		}
	case simpleStateTableRecords:
		if item, ok := m.list.SelectedItem().(simpleItem); ok {
			if item.id == "back" {
				return m.handleBack()
			}
			// View record detail/XML
			return m.handleViewRecordDetail(item.id)
		}
	}
	return m, nil
}


// Handle custom table input
func (m Model) handleCustomTable() (tea.Model, tea.Cmd) {
	if m.state == simpleStateTableList {
		m.state = simpleStateCustomTable
		m.updateListSize()
		return m, nil
	}
	return m, nil
}


// Handle filter/query
func (m Model) handleFilter() (tea.Model, tea.Cmd) {
	if m.state == simpleStateTableRecords {
		m.state = simpleStateQueryFilter
		m.filterInput = m.currentQuery // Initialize filter input with current query
		m.updateListSize()
		return m, nil
	}
	return m, nil
}


// Handle view XML
func (m Model) handleViewXML() (tea.Model, tea.Cmd) {
	if m.state == simpleStateTableRecords {
		if item, ok := m.list.SelectedItem().(simpleItem); ok && item.id != "back" {
			return m.handleViewRecordDetail(item.id)
		}
	}
	return m, nil
}


// Handle view record detail
func (m Model) handleViewRecordDetail(recordID string) (tea.Model, tea.Cmd) {
	// Find the record in our current records
	for _, record := range m.records {
		if getRecordField(record, "sys_id") == recordID {
			m.selectedRecord = record
			m.state = simpleStateRecordDetail
			m.loading = true
			
			m.updateListSize()
			return m, m.loadRecordXMLCmd(recordID)
		}
	}

	// If we can't find the record, generate error
	return m, func() tea.Msg {
		return recordsErrorMsg{err: fmt.Errorf("record not found in current page: %s", recordID)}
	}
}

// Handle XML scroll
func (m Model) handleXMLScroll(direction int) (tea.Model, tea.Cmd) {
	if m.state == simpleStateRecordDetail && m.recordXML != "" {
		lines := strings.Split(m.recordXML, "\n")

		// Use same calculation as renderScrollableXML
		headerHeight := 3
		footerHeight := m.calculateHelpFooterHeight() // Use help footer height
		loadingHeight := 1 // Always reserve space for loading indicator
		contentHeight := m.height - headerHeight - footerHeight - loadingHeight
		if contentHeight < 3 {
			contentHeight = 3
		}
		xmlHeight := contentHeight - 4

		maxScroll := len(lines) - xmlHeight
		if maxScroll < 0 {
			maxScroll = 0
		}

		m.xmlScrollOffset += direction
		if m.xmlScrollOffset < 0 {
			m.xmlScrollOffset = 0
		}
		if m.xmlScrollOffset > maxScroll {
			m.xmlScrollOffset = maxScroll
		}
	}
	return m, nil
}


// Handle XML search
func (m Model) handleXMLSearch() (tea.Model, tea.Cmd) {
	if m.state == simpleStateRecordDetail && m.recordXML != "" {
		m.state = simpleStateXMLSearch
		m.xmlSearchQuery = ""
		m.xmlSearchResults = []int{}
		m.xmlSearchIndex = 0
		return m, nil
	}
	return m, nil
}


// Handle advanced query builder
func (m Model) handleAdvancedQuery() (tea.Model, tea.Cmd) {
	if m.state == simpleStateTableRecords {
		m.initializeAdvancedFiltering()
		m.initializeQueryBuilderForTable()
		if m.queryBuilder != nil {
			m.state = simpleStateAdvancedFilter
			m.queryBuilder.SetActive(true)
			m.updateListSize()
			// Send a window size message to ensure proper sizing
			return m, func() tea.Msg {
				return tea.WindowSizeMsg{Width: m.width, Height: m.height}
			}
		}
	}
	return m, nil
}


// Handle filter browser
func (m *Model) handleFilterBrowser() (*Model, tea.Cmd) {
	if m.state == simpleStateTableRecords {
		m.initializeAdvancedFiltering()
		if m.filterBrowser != nil {
			m.state = simpleStateFilterBrowser
			m.filterBrowser.SetTableName(m.currentTable)
			m.filterBrowser.SetCurrentQuery(m.currentQuery, nil) // TODO: Add conditions
			m.filterBrowser.SetActive(true)
			m.updateListSize()
		}
	}
	return m, nil
}

// Get the previous state for quit confirmation
func (m Model) getPreviousState() simpleState {
	return m.previousState
}

// Handle column customizer
func (m Model) handleColumnCustomizer() (tea.Model, tea.Cmd) {
	if m.state == simpleStateTableRecords {
		// Ensure we have table metadata loaded first
		m.initializeAdvancedFiltering()
		m.initializeQueryBuilderForTable() // This loads the actual table metadata
		m.initializeColumnCustomizer()
		if m.columnCustomizer != nil {
			m.state = simpleStateColumnCustomizer
			m.columnCustomizer.SetActive(true)
			m.columnCustomizer.SetDimensions(m.width, m.height)
			m.updateListSize()
			return m, nil
		}
	}
	return m, nil
}

// Handle save view
func (m Model) handleSaveView() (tea.Model, tea.Cmd) {
	if m.state == simpleStateTableRecords {
		// Generate a unique name based on current configuration
		timestamp := time.Now().Format("2006-01-02_15-04")
		viewName := fmt.Sprintf("%s_%s", m.currentTable, timestamp)
		
		// Add column info to description
		description := fmt.Sprintf("Auto-saved view with %d columns", len(m.selectedColumns))
		if len(m.selectedColumns) > 0 {
			description += fmt.Sprintf(": %s", strings.Join(m.selectedColumns[:min(3, len(m.selectedColumns))], ", "))
			if len(m.selectedColumns) > 3 {
				description += "..."
			}
		}
		if m.currentQuery != "" {
			description += fmt.Sprintf(" | Filter: %s", m.currentQuery)
		}
		
		m.saveViewConfiguration(viewName, description)
	}
	return m, nil
}

// Helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Handle view manager
func (m Model) handleViewManager() (tea.Model, tea.Cmd) {
	if m.state == simpleStateTableRecords {
		m.state = simpleStateViewManager
		m.viewManagerSelection = 0 // Reset selection when entering
		m.updateListSize()
		return m, nil
	}
	return m, nil
}

// Handle reset columns
func (m Model) handleResetColumns() (tea.Model, tea.Cmd) {
	if m.state == simpleStateTableRecords {
		// Reset to default columns for current table
		m.setupDefaultColumnsForTable(m.currentTable)
		m.loading = true
		m.currentPage = 0
		if m.currentQuery != "" {
			return m, m.loadTableRecordsWithQueryCmd(m.currentTable, m.currentQuery)
		}
		return m, m.loadTableRecordsCmd(m.currentTable)
	}
	return m, nil
}

// Initialize column customizer if needed
func (m *Model) initializeColumnCustomizer() {
	if m.columnCustomizer == nil {
		m.columnCustomizer = NewColumnCustomizer()
	}
	
	// Initialize with current table metadata and selected columns
	if m.tableMetadata != nil {
		m.columnCustomizer.InitializeFields(m.tableMetadata, m.selectedColumns)
	}
}

// Handle export
func (m Model) handleExport() (tea.Model, tea.Cmd) {
	if m.state == simpleStateTableRecords {
		if m.exportDialog != nil {
			m.state = simpleStateExportDialog
			m.exportDialog.SetActive(true)
			m.exportDialog.SetDimensions(m.width, m.height)
			m.updateListSize()
			return m, nil
		}
	}
	return m, nil
}

// Save view configuration
func (m *Model) saveViewConfiguration(name, description string) {
	config := &ViewConfiguration{
		Name:        name,
		TableName:   m.currentTable,
		Columns:     make([]string, len(m.selectedColumns)),
		Query:       m.currentQuery,
		Description: description,
		CreatedAt:   fmt.Sprintf("%d", time.Now().Unix()),
	}
	copy(config.Columns, m.selectedColumns)
	
	// Save to both in-memory and persistent storage
	m.viewConfigurations[name] = config
	if m.configManager != nil {
		m.configManager.SaveViewConfiguration(name, config)
	}
}

// handleSort handles column sorting
func (m Model) handleSort(direction string) (tea.Model, tea.Cmd) {
	if m.state != simpleStateTableRecords || len(m.selectedColumns) == 0 {
		return m, nil
	}
	
	// For now, sort by the first visible column
	// TODO: Add UI to select which column to sort by
	sortColumn := m.selectedColumns[0]
	
	// If already sorting by this column, toggle direction
	if m.sortColumn == sortColumn {
		if m.sortDirection == "asc" && direction == "asc" {
			direction = "desc"
		} else if m.sortDirection == "desc" && direction == "desc" {
			direction = "asc"
		}
	}
	
	m.sortColumn = sortColumn
	m.sortDirection = direction
	m.loading = true
	m.currentPage = 0 // Reset to first page when sorting
	
	// Reload data with sorting
	return m, m.loadTableRecordsWithSortCmd(m.currentTable, m.currentQuery, sortColumn, direction)
}

// handleRecentTables shows recent tables list
func (m Model) handleRecentTables() (tea.Model, tea.Cmd) {
	if m.state != simpleStateTableList && m.state != simpleStateMain {
		return m, nil
	}
	
	// TODO: Implement recent tables UI
	// For now, just switch to table list state
	m.state = simpleStateTableList
	return m, nil
}

// handleBookmarkTable bookmarks the current table
func (m Model) handleBookmarkTable() (tea.Model, tea.Cmd) {
	if m.state != simpleStateTableRecords || m.currentTable == "" {
		return m, nil
	}
	
	if m.configManager == nil {
		return m, nil
	}
	
	// Check if already bookmarked
	if m.configManager.IsBookmarked(m.currentTable) {
		// Remove bookmark
		m.configManager.RemoveBookmark(m.currentTable)
	} else {
		// Add bookmark
		displayName := m.currentTable // TODO: Get proper display name from table metadata
		m.configManager.AddBookmark(m.currentTable, displayName, "")
	}
	
	return m, nil
}

// handleShowBookmarks toggles between showing all tables and showing only bookmarked tables
func (m Model) handleShowBookmarks() (tea.Model, tea.Cmd) {
	if m.state != simpleStateTableList {
		return m, nil
	}
	
	// Toggle bookmarks view
	m.showingBookmarks = !m.showingBookmarks
	
	// Reload table list with new view
	m.loadTableList()
	
	return m, nil
}

// handleEditField enters edit mode for the currently selected field in the XML view
func (m Model) handleEditField() (tea.Model, tea.Cmd) {
	if m.state != simpleStateRecordDetail || m.selectedRecord == nil {
		return m, nil
	}
	
	// Extract editable fields from the current record
	m.editableFields = m.extractEditableFields()
	
	if len(m.editableFields) == 0 {
		return m, nil // No editable fields
	}
	
	// Use the currently selected field index
	if m.currentFieldIndex >= 0 && m.currentFieldIndex < len(m.editableFields) {
		m.editingField = m.editableFields[m.currentFieldIndex]
	} else {
		// Default to first field if index is invalid
		m.editingField = m.editableFields[0]
		m.currentFieldIndex = 0
	}
	
	// Get the current field value properly
	m.editFieldValue = m.getFieldValue(m.editingField)
	
	m.state = simpleStateEditField
	m.updateListSize()
	
	return m, nil
}

// handleNextXMLField moves to the next editable field in XML view and scrolls to it
func (m Model) handleNextXMLField() (tea.Model, tea.Cmd) {
	if m.selectedRecord == nil {
		// If no record, just scroll down
		return m.handleXMLScroll(1)
	}
	
	// Extract editable fields if not already done
	if len(m.editableFields) == 0 {
		m.editableFields = m.extractEditableFields()
	}
	
	if len(m.editableFields) == 0 {
		// If no editable fields, just scroll down
		return m.handleXMLScroll(1)
	}
	
	// Move to next field (wrap around)
	m.currentFieldIndex = (m.currentFieldIndex + 1) % len(m.editableFields)
	
	// Auto-scroll to keep selected field in view
	m.scrollToCurrentField()
	
	return m, nil
}

// handlePrevXMLField moves to the previous editable field in XML view and scrolls to it
func (m Model) handlePrevXMLField() (tea.Model, tea.Cmd) {
	if m.selectedRecord == nil {
		// If no record, just scroll up
		return m.handleXMLScroll(-1)
	}
	
	// Extract editable fields if not already done
	if len(m.editableFields) == 0 {
		m.editableFields = m.extractEditableFields()
	}
	
	if len(m.editableFields) == 0 {
		// If no editable fields, just scroll up
		return m.handleXMLScroll(-1)
	}
	
	// Move to previous field (wrap around)
	m.currentFieldIndex = (m.currentFieldIndex - 1 + len(m.editableFields)) % len(m.editableFields)
	
	// Auto-scroll to keep selected field in view
	m.scrollToCurrentField()
	
	return m, nil
}

// extractEditableFields returns ALL fields from the XML except sys_* fields
func (m Model) extractEditableFields() []string {
	if m.recordXML == "" {
		return nil
	}
	
	var fields []string
	seen := make(map[string]bool) // Prevent duplicates
	lines := strings.Split(m.recordXML, "\n")
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		// Skip comments, closing tags, and XML declarations
		if strings.HasPrefix(line, "<!--") || strings.HasPrefix(line, "</") || strings.HasPrefix(line, "<?") {
			continue
		}
		
		// Look for opening XML tags - handle multiple tags on one line if present
		if strings.Contains(line, "<") && strings.Contains(line, ">") {
			// Handle multiple tags that might be on the same line
			remaining := line
			for strings.Contains(remaining, "<") {
				// Find the start of the next tag
				tagStart := strings.Index(remaining, "<")
				if tagStart == -1 {
					break
				}
				
				// Skip closing tags and comments
				if strings.HasPrefix(remaining[tagStart:], "</") || strings.HasPrefix(remaining[tagStart:], "<!--") || strings.HasPrefix(remaining[tagStart:], "<?") {
					// Move past this tag and continue
					if nextPos := strings.Index(remaining[tagStart+1:], "<"); nextPos != -1 {
						remaining = remaining[tagStart+1+nextPos:]
					} else {
						break
					}
					continue
				}
				
				// Find the end of this tag name
				start := tagStart + 1 // Skip the <
				end := len(remaining)
				
				// Find the end of the tag name (first space, > or /)
				for i := start; i < len(remaining); i++ {
					if remaining[i] == ' ' || remaining[i] == '>' || remaining[i] == '/' {
						end = i
						break
					}
				}
				
				if end > start {
					fieldName := remaining[start:end]
					fieldName = strings.TrimSpace(fieldName)
					
					// Skip XML declaration, record wrapper, empty names, and sys_* fields
					if fieldName != "" && fieldName != "?xml" && fieldName != "record" && !strings.HasPrefix(fieldName, "sys_") {
						// Validate field name (should be valid XML name)
						if isValidFieldName(fieldName) && !seen[fieldName] {
							fields = append(fields, fieldName)
							seen[fieldName] = true
						}
					}
				}
				
				// Move past this tag and look for more
				if nextPos := strings.Index(remaining[start:], "<"); nextPos != -1 {
					remaining = remaining[start+nextPos:]
				} else {
					break
				}
			}
		}
	}
	
	// Debug: Uncomment this to see what fields are being extracted
	// fmt.Printf("DEBUG: Extracted fields: %v\n", fields)
	
	// If no fields were found, let's try a more aggressive approach
	if len(fields) == 0 {
		// Fallback: Look for any tag that might be a field
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "<") && !strings.HasPrefix(line, "</") && 
			   !strings.HasPrefix(line, "<?") && !strings.HasPrefix(line, "<!--") {
				// Try to extract any tag name for debugging
				if start := strings.Index(line, "<"); start >= 0 {
					remaining := line[start+1:]
					if end := strings.IndexAny(remaining, " />"); end > 0 {
						tagName := remaining[:end]
						if tagName != "record" && !strings.HasPrefix(tagName, "sys_") {
							if !seen[tagName] {
								fields = append(fields, tagName)
								seen[tagName] = true
							}
						}
					}
				}
			}
		}
	}
	
	return fields
}

// isValidFieldName checks if a string is a valid XML field name
func isValidFieldName(name string) bool {
	if len(name) == 0 {
		return false
	}
	
	// First character must be letter or underscore
	if !((name[0] >= 'a' && name[0] <= 'z') || (name[0] >= 'A' && name[0] <= 'Z') || name[0] == '_') {
		return false
	}
	
	// Rest can be letters, digits, underscore, hyphen, or dot
	for i := 1; i < len(name); i++ {
		c := name[i]
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '-' || c == '.') {
			return false
		}
	}
	
	return true
}

// performReferenceSearch searches for records in the reference table
func (m Model) performReferenceSearch() tea.Cmd {
	if m.client == nil || m.referenceSearchTable == "" || m.referenceSearchQuery == "" {
		return nil
	}
	
	return func() tea.Msg {
		// Create search parameters
		// Search in common display fields like name, short_description, number
		searchFields := []string{"name", "short_description", "number", "title", "label"}
		var queryParts []string
		
		for _, field := range searchFields {
			queryParts = append(queryParts, fmt.Sprintf("%sCONTAINS%s", field, m.referenceSearchQuery))
		}
		
		params := map[string]string{
			"sysparm_query":  strings.Join(queryParts, "^OR"),
			"sysparm_fields": "sys_id,name,short_description,number,title,label",
			"sysparm_limit":  "10", // Limit to 10 results for UI
		}
		
		records, err := m.client.Table(m.referenceSearchTable).List(params)
		if err != nil {
			return referenceSearchErrorMsg{err: err}
		}
		
		return referenceSearchResultsMsg{results: records}
	}
}

// getFieldValue extracts the value from XML for the given field
func (m Model) getFieldValue(fieldName string) string {
	if m.recordXML == "" {
		return ""
	}
	
	lines := strings.Split(m.recordXML, "\n")
	
	// Find the field value in XML
	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		// Handle both simple tags <field>value</field> and reference tags <field display_value="...">value</field>
		openTagStart := fmt.Sprintf("<%s", fieldName)
		closeTag := fmt.Sprintf("</%s>", fieldName)
		
		if strings.HasPrefix(line, openTagStart) && strings.HasSuffix(line, closeTag) {
			// Find where the opening tag ends
			tagEndIndex := strings.Index(line, ">")
			if tagEndIndex == -1 {
				continue
			}
			
			// Extract value between opening tag and closing tag
			start := tagEndIndex + 1
			end := len(line) - len(closeTag)
			if start < end {
				return line[start:end]
			} else if start == end {
				// Empty field case
				return ""
			}
		}
		
		// Also handle self-closing tags like <field/> or <field display_value="..." />
		if strings.HasPrefix(line, openTagStart) && (strings.HasSuffix(line, "/>") || strings.Contains(line, " />")) {
			// Self-closing tag means empty value
			return ""
		}
	}
	
	return ""
}

// scrollToCurrentField automatically scrolls to keep the current field in the middle of the screen
func (m *Model) scrollToCurrentField() {
	if m.recordXML == "" || len(m.editableFields) == 0 || m.currentFieldIndex < 0 || m.currentFieldIndex >= len(m.editableFields) {
		return
	}
	
	currentField := m.editableFields[m.currentFieldIndex]
	lines := strings.Split(m.recordXML, "\n")
	
	// Find the line containing the current field opening tag
	targetLine := -1
	for i, line := range lines {
		if strings.Contains(line, fmt.Sprintf("<%s>", currentField)) || strings.Contains(line, fmt.Sprintf("<%s ", currentField)) {
			targetLine = i
			break
		}
	}
	
	if targetLine == -1 {
		return // Field not found in XML
	}
	
	// Calculate viewport dimensions (same as renderScrollableXML)
	headerHeight := 3
	footerHeight := m.calculateHelpFooterHeight()
	loadingHeight := 1
	contentHeight := m.height - headerHeight - footerHeight - loadingHeight
	if contentHeight < 3 {
		contentHeight = 3
	}
	xmlHeight := contentHeight - 4
	if xmlHeight < 1 {
		xmlHeight = 1
	}
	
	// Calculate desired scroll position to center the field
	halfViewport := xmlHeight / 2
	desiredScroll := targetLine - halfViewport
	
	// Apply bounds checking
	maxScroll := len(lines) - xmlHeight
	if maxScroll < 0 {
		maxScroll = 0
	}
	
	// Only scroll if we're not close to top/bottom edges
	minScrollThreshold := 2 // Don't scroll if within 2 lines of top
	maxScrollThreshold := maxScroll - 2 // Don't scroll if within 2 lines of bottom
	
	if desiredScroll < minScrollThreshold {
		m.xmlScrollOffset = 0
	} else if desiredScroll > maxScrollThreshold {
		m.xmlScrollOffset = maxScroll
	} else {
		m.xmlScrollOffset = desiredScroll
	}
}

// autoSelectFieldFromSearch attempts to auto-select the field that matches the current search result
func (m *Model) autoSelectFieldFromSearch() {
	if len(m.xmlSearchResults) == 0 || m.xmlSearchIndex < 0 || m.xmlSearchIndex >= len(m.xmlSearchResults) {
		return
	}
	
	if len(m.editableFields) == 0 {
		m.editableFields = m.extractEditableFields()
	}
	
	if len(m.editableFields) == 0 {
		return
	}
	
	// Get the line number of the current search result
	targetLine := m.xmlSearchResults[m.xmlSearchIndex]
	lines := strings.Split(m.recordXML, "\n")
	
	if targetLine < 0 || targetLine >= len(lines) {
		return
	}
	
	// Get the line content
	line := strings.TrimSpace(lines[targetLine])
	
	// Try to find which field this line belongs to
	for i, fieldName := range m.editableFields {
		// Check if this line contains the field opening tag
		if strings.Contains(line, fmt.Sprintf("<%s>", fieldName)) || 
		   strings.Contains(line, fmt.Sprintf("<%s ", fieldName)) ||
		   strings.Contains(line, fmt.Sprintf("</%s>", fieldName)) {
			// Found the field, select it
			m.currentFieldIndex = i
			return
		}
	}
}