package main

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// Handle custom table input
func (m simpleModel) handleCustomTableInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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
			m.loading = true
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
func (m simpleModel) handleXMLSearchInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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
func (m simpleModel) handleQueryFilterInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC:
		return m, tea.Quit
	case tea.KeyEsc:
		m.currentQuery = "" // Clear query
		return m.handleBack()
	case tea.KeyEnter:
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
		if len(m.currentQuery) > 0 {
			m.currentQuery = m.currentQuery[:len(m.currentQuery)-1]
		}
	case tea.KeyRunes:
		// Add typed characters to query (allow all characters for ServiceNow queries)
		m.currentQuery += string(msg.Runes)
	}
	return m, nil
}

// Perform XML search
func (m *simpleModel) performXMLSearch(query string) {
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

	// Navigate to first match
	if len(m.xmlSearchResults) > 0 {
		m.scrollToSearchResult()
	}
}

// Navigate to next/previous search result
func (m *simpleModel) navigateSearchResult(direction int) {
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
}

// Scroll to current search result
func (m *simpleModel) scrollToSearchResult() {
	if len(m.xmlSearchResults) == 0 || m.xmlSearchIndex < 0 || m.xmlSearchIndex >= len(m.xmlSearchResults) {
		return
	}

	targetLine := m.xmlSearchResults[m.xmlSearchIndex]

	// Calculate viewport
	headerHeight := 3
	footerHeight := 2
	contentHeight := m.height - headerHeight - footerHeight
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