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