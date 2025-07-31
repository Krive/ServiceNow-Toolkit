package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

// Handle back navigation
func (m simpleModel) handleBack() (tea.Model, tea.Cmd) {
	switch m.state {
	case simpleStateTableList:
		m.state = simpleStateMain
		m.loadMainMenu()
	case simpleStateTableRecords:
		m.state = simpleStateTableList
		m.loadTableList()
	case simpleStateRecordDetail:
		m.state = simpleStateTableRecords
		// Reload the record list
		m.loading = true
		return m, m.loadTableRecordsCmd(m.currentTable)
	case simpleStateCustomTable, simpleStateQueryFilter:
		m.state = simpleStateTableList
		m.loadTableList()
	case simpleStateXMLSearch:
		m.state = simpleStateRecordDetail
		m.xmlSearchQuery = ""
		m.xmlSearchResults = []int{}
		m.xmlSearchIndex = 0
	default:
		m.state = simpleStateMain
		m.loadMainMenu()
	}
	// Update list size when state changes (header visibility may change)
	m.updateListSize()
	return m, nil
}

// Handle next page
func (m simpleModel) handleNextPage() (tea.Model, tea.Cmd) {
	if m.state == simpleStateTableRecords && m.currentPage < m.totalPages-1 {
		m.currentPage++
		m.loading = true
		return m, m.loadTableRecordsCmd(m.currentTable)
	}
	return m, nil
}

// Handle previous page
func (m simpleModel) handlePrevPage() (tea.Model, tea.Cmd) {
	if m.state == simpleStateTableRecords && m.currentPage > 0 {
		m.currentPage--
		m.loading = true
		return m, m.loadTableRecordsCmd(m.currentTable)
	}
	return m, nil
}

// Handle refresh
func (m simpleModel) handleRefresh() (tea.Model, tea.Cmd) {
	switch m.state {
	case simpleStateTableRecords:
		m.loading = true
		return m, m.loadTableRecordsCmd(m.currentTable)
	case simpleStateTableList:
		m.loadTableList()
	case simpleStateMain:
		m.loadMainMenu()
	}
	return m, nil
}

// Handle enter key
func (m simpleModel) handleEnter() (tea.Model, tea.Cmd) {
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
			m.loading = true  // Set loading state
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
func (m simpleModel) handleCustomTable() (tea.Model, tea.Cmd) {
	if m.state == simpleStateTableList {
		m.state = simpleStateCustomTable
		m.updateListSize()
		return m, nil
	}
	return m, nil
}

// Handle filter/query
func (m simpleModel) handleFilter() (tea.Model, tea.Cmd) {
	if m.state == simpleStateTableRecords {
		m.state = simpleStateQueryFilter
		m.updateListSize()
		return m, nil
	}
	return m, nil
}

// Handle view XML
func (m simpleModel) handleViewXML() (tea.Model, tea.Cmd) {
	if m.state == simpleStateTableRecords {
		if item, ok := m.list.SelectedItem().(simpleItem); ok && item.id != "back" {
			return m.handleViewRecordDetail(item.id)
		}
	}
	return m, nil
}

// Handle view record detail
func (m simpleModel) handleViewRecordDetail(recordID string) (tea.Model, tea.Cmd) {
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
func (m simpleModel) handleXMLScroll(direction int) (tea.Model, tea.Cmd) {
	if m.state == simpleStateRecordDetail && m.recordXML != "" {
		lines := strings.Split(m.recordXML, "\n")

		// Use same calculation as renderScrollableXML
		headerHeight := 3
		footerHeight := 2
		contentHeight := m.height - headerHeight - footerHeight
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
func (m simpleModel) handleXMLSearch() (tea.Model, tea.Cmd) {
	if m.state == simpleStateRecordDetail && m.recordXML != "" {
		m.state = simpleStateXMLSearch
		m.xmlSearchQuery = ""
		m.xmlSearchResults = []int{}
		m.xmlSearchIndex = 0
		return m, nil
	}
	return m, nil
}