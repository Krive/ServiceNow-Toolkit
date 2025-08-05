package explorer

import (
	"fmt"
	"strings"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
)

// Handle custom table input
func (m Model) handleCustomTableInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC:
		return m, tea.Quit
	case tea.KeyEsc:
		m.customTableInput = "" // Clear input
		return m.handleBack()
	case tea.KeyEnter:
		if m.customTableInput != "" {
			// Load the custom table
			m.currentTable = m.customTableInput
			m.state = simpleStateTableRecords
			m.currentPage = 0
			m.currentQuery = "" // Clear any previous filters when entering a new table
			m.loading = true
			
			// Set up intelligent default columns for this table
			m.setupDefaultColumnsForTable(m.customTableInput)
			
			m.customTableInput = "" // Clear input
			return m, m.loadTableRecordsCmd(m.currentTable)
		}
	case tea.KeyBackspace:
		if len(m.customTableInput) > 0 {
			m.customTableInput = m.customTableInput[:len(m.customTableInput)-1]
		}
	case tea.KeyRunes:
		// Add typed characters to input (filter out hotkey characters when typing)
		runes := string(msg.Runes)
		if runes != "q" || len(m.customTableInput) > 0 { // Allow 'q' as part of table names
			m.customTableInput += runes
		}
	}
	return m, nil
}

// Handle XML search input
func (m Model) handleXMLSearchInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC:
		return m, tea.Quit
	case tea.KeyEsc:
		m.xmlSearchQuery = ""
		m.xmlSearchResults = []int{}
		m.xmlSearchIndex = 0
		m.state = simpleStateRecordDetail
		return m, nil
	case tea.KeyEnter:
		if m.xmlSearchQuery != "" {
			m.performXMLSearch(m.xmlSearchQuery)
			m.state = simpleStateRecordDetail
		} else {
			m.state = simpleStateRecordDetail
		}
		return m, nil
	case tea.KeyBackspace:
		if len(m.xmlSearchQuery) > 0 {
			m.xmlSearchQuery = m.xmlSearchQuery[:len(m.xmlSearchQuery)-1]
			// Update search results in real-time
			m.performXMLSearch(m.xmlSearchQuery)
		}
	case tea.KeyRunes:
		runes := string(msg.Runes)
		// Allow most characters in search
		if runes != "q" || len(m.xmlSearchQuery) > 0 {
			m.xmlSearchQuery += runes
			// Update search results in real-time
			m.performXMLSearch(m.xmlSearchQuery)
		}
	}
	return m, nil
}

// Handle query filter input
func (m Model) handleQueryFilterInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC:
		return m, tea.Quit
	case tea.KeyEsc:
		// Clear the temporary filter input and go back without applying changes
		m.filterInput = ""
		return m.handleBack()
	case tea.KeyEnter:
		// Apply the filter input to the current query
		m.currentQuery = m.filterInput
		m.filterInput = "" // Clear temporary input
		if m.currentQuery != "" {
			// Apply filter and reload records
			m.state = simpleStateTableRecords
			m.currentPage = 0
			m.loading = true
			return m, m.loadTableRecordsWithQueryCmd(m.currentTable, m.currentQuery)
		} else {
			// Clear filter and reload all records
			m.state = simpleStateTableRecords
			m.currentPage = 0
			m.loading = true
			return m, m.loadTableRecordsCmd(m.currentTable)
		}
	case tea.KeyBackspace:
		if len(m.filterInput) > 0 {
			m.filterInput = m.filterInput[:len(m.filterInput)-1]
		}
	case tea.KeyRunes:
		// Add typed characters to filter input (allow all characters for ServiceNow queries)
		m.filterInput += string(msg.Runes)
	}
	return m, nil
}

// Perform XML search
func (m *Model) performXMLSearch(query string) {
	if query == "" {
		m.xmlSearchResults = []int{}
		m.xmlSearchIndex = 0
		return
	}

	lines := strings.Split(m.recordXML, "\n")
	m.xmlSearchResults = []int{}

	// Case-insensitive search
	queryLower := strings.ToLower(query)

	for i, line := range lines {
		if strings.Contains(strings.ToLower(line), queryLower) {
			m.xmlSearchResults = append(m.xmlSearchResults, i)
		}
	}

	m.xmlSearchIndex = 0

	// Navigate to first match and auto-select the field if it matches a known field
	if len(m.xmlSearchResults) > 0 {
		m.scrollToSearchResult()
		m.autoSelectFieldFromSearch()
	}
}

// Navigate to next/previous search result
func (m *Model) navigateSearchResult(direction int) {
	if len(m.xmlSearchResults) == 0 {
		return
	}

	m.xmlSearchIndex += direction
	if m.xmlSearchIndex < 0 {
		m.xmlSearchIndex = len(m.xmlSearchResults) - 1
	}
	if m.xmlSearchIndex >= len(m.xmlSearchResults) {
		m.xmlSearchIndex = 0
	}

	m.scrollToSearchResult()
	m.autoSelectFieldFromSearch()
}

// Scroll to current search result
func (m *Model) scrollToSearchResult() {
	if len(m.xmlSearchResults) == 0 || m.xmlSearchIndex < 0 || m.xmlSearchIndex >= len(m.xmlSearchResults) {
		return
	}

	targetLine := m.xmlSearchResults[m.xmlSearchIndex]

	// Calculate viewport
	headerHeight := 3
	footerHeight := m.calculateHelpFooterHeight() // Use help footer height
	loadingHeight := 1 // Always reserve space for loading indicator
	contentHeight := m.height - headerHeight - footerHeight - loadingHeight
	if contentHeight < 3 {
		contentHeight = 3
	}
	xmlHeight := contentHeight - 4

	// Center the target line in the viewport
	m.xmlScrollOffset = targetLine - xmlHeight/2

	// Ensure scroll bounds
	lines := strings.Split(m.recordXML, "\n")
	maxScroll := len(lines) - xmlHeight
	if maxScroll < 0 {
		maxScroll = 0
	}

	if m.xmlScrollOffset < 0 {
		m.xmlScrollOffset = 0
	}
	if m.xmlScrollOffset > maxScroll {
		m.xmlScrollOffset = maxScroll
	}
}

// Handle edit field input
func (m Model) handleEditFieldInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		// Save the field and exit edit mode
		return m.handleSaveFieldEdit()
	case "esc":
		// Cancel edit and return to record detail
		m.state = simpleStateRecordDetail
		m.editingField = ""
		m.editFieldValue = ""
		m.updateListSize()
		return m, nil
	case "backspace":
		if len(m.editFieldValue) > 0 {
			m.editFieldValue = m.editFieldValue[:len(m.editFieldValue)-1]
		}
	case "ctrl+a":
		// Select all - not implemented in TUI, just clear for now
		m.editFieldValue = ""
	case "ctrl+v":
		// Paste from clipboard
		if clipboardText, err := clipboard.ReadAll(); err == nil {
			// Remove any newlines from clipboard content for single-line field
			cleanText := strings.ReplaceAll(clipboardText, "\n", "")
			cleanText = strings.ReplaceAll(cleanText, "\r", "")
			m.editFieldValue += cleanText
		}
	case "ctrl+f":
		// Reference field search - try to infer the reference table from field name
		refTable := m.inferReferenceTable(m.editingField)
		if refTable != "" {
			m.state = simpleStateReferenceSearch
			m.referenceSearchTable = refTable
			m.referenceSearchQuery = ""
			m.referenceSearchResults = nil
			m.referenceSelection = 0
			return m, nil
		}
	default:
		// Add typed character
		if len(msg.String()) == 1 {
			m.editFieldValue += msg.String()
		}
	}
	
	return m, nil
}

// Handle reference search input
func (m Model) handleReferenceSearchInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Cancel search and return to field editing
		m.state = simpleStateEditField
		m.referenceSearchQuery = ""
		m.referenceSearchResults = nil
		m.referenceSelection = 0
		return m, nil
	case "enter":
		// Select the currently highlighted reference
		if len(m.referenceSearchResults) > 0 && m.referenceSelection >= 0 && m.referenceSelection < len(m.referenceSearchResults) {
			selectedRecord := m.referenceSearchResults[m.referenceSelection]
			
			// Extract sys_id from selected record
			if sysID, ok := selectedRecord["sys_id"].(string); ok {
				m.editFieldValue = sysID
			}
			
			// Return to field editing
			m.state = simpleStateEditField
			m.referenceSearchQuery = ""
			m.referenceSearchResults = nil
			m.referenceSelection = 0
		}
		return m, nil
	case "up", "k":
		// Navigate up in search results
		if len(m.referenceSearchResults) > 0 {
			m.referenceSelection = (m.referenceSelection - 1 + len(m.referenceSearchResults)) % len(m.referenceSearchResults)
		}
		return m, nil
	case "down", "j":
		// Navigate down in search results
		if len(m.referenceSearchResults) > 0 {
			m.referenceSelection = (m.referenceSelection + 1) % len(m.referenceSearchResults)
		}
		return m, nil
	case "backspace":
		if len(m.referenceSearchQuery) > 0 {
			m.referenceSearchQuery = m.referenceSearchQuery[:len(m.referenceSearchQuery)-1]
			// Trigger search with updated query
			if len(m.referenceSearchQuery) >= 2 {
				return m, m.performReferenceSearch()
			} else {
				m.referenceSearchResults = nil
				m.referenceSelection = 0
			}
		}
		return m, nil
	default:
		// Add typed character and trigger search
		if len(msg.String()) == 1 {
			m.referenceSearchQuery += msg.String()
			// Trigger search when we have at least 2 characters
			if len(m.referenceSearchQuery) >= 2 {
				return m, m.performReferenceSearch()
			}
		}
	}
	
	return m, nil
}

// inferReferenceTable attempts to infer the reference table from field name
func (m Model) inferReferenceTable(fieldName string) string {
	fieldLower := strings.ToLower(fieldName)
	
	// Common reference field patterns
	if strings.Contains(fieldLower, "user") || fieldLower == "assigned_to" || 
	   fieldLower == "caller_id" || fieldLower == "opened_by" || fieldLower == "closed_by" {
		return "sys_user"
	}
	
	if strings.Contains(fieldLower, "group") || fieldLower == "assignment_group" {
		return "sys_user_group"
	}
	
	if strings.Contains(fieldLower, "location") {
		return "cmn_location"
	}
	
	if strings.Contains(fieldLower, "company") {
		return "core_company"
	}
	
	if strings.Contains(fieldLower, "department") {
		return "cmn_department"
	}
	
	if strings.Contains(fieldLower, "category") {
		return "sys_choice" // This is a simplification
	}
	
	// If we can't infer, return empty string to disable search
	return ""
}

// handleSaveFieldEdit saves the current field edit
func (m Model) handleSaveFieldEdit() (tea.Model, tea.Cmd) {
	if m.editingField == "" || m.recordXML == "" {
		m.state = simpleStateRecordDetail
		return m, nil
	}
	
	// Get the sys_id from XML for the update
	sysID := m.getFieldValue("sys_id")
	if sysID == "" {
		m.state = simpleStateRecordDetail
		return m, nil
	}
	
	// Store values before clearing them
	fieldName := m.editingField
	fieldValue := m.editFieldValue
	
	// Return to record detail view immediately
	m.state = simpleStateRecordDetail
	m.editingField = ""
	m.editFieldValue = ""
	m.updateListSize()
	
	// If we have a client, save to ServiceNow
	if m.client != nil {
		// Create update map with just the changed field
		updates := map[string]interface{}{
			fieldName: fieldValue,
		}
		
		// Return command to update the record
		return m, func() tea.Msg {
			_, err := m.client.Table(m.currentTable).Update(sysID, updates)
			if err != nil {
				return recordsErrorMsg{err: err}
			}
			
			// Reload the record XML after successful update
			return m.loadRecordXMLCmd(sysID)()
		}
	}
	
	// Demo mode - just update the XML locally
	lines := strings.Split(m.recordXML, "\n")
	openTagStart := fmt.Sprintf("<%s", fieldName)
	closeTag := fmt.Sprintf("</%s>", fieldName)
	
	// Find and update the field in XML
	for i, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if strings.HasPrefix(trimmedLine, openTagStart) && strings.HasSuffix(trimmedLine, closeTag) {
			// Find where the opening tag ends
			tagEndIndex := strings.Index(trimmedLine, ">")
			if tagEndIndex == -1 {
				continue
			}
			
			// Replace the content between tags, preserving any attributes
			prefix := line[:strings.Index(line, openTagStart)]
			openTagFull := trimmedLine[:tagEndIndex+1]
			lines[i] = prefix + openTagFull + fieldValue + closeTag
			break
		}
	}
	
	// Update the XML
	m.recordXML = strings.Join(lines, "\n")
	
	return m, nil
}

