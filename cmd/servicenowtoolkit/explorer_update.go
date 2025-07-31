package main

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

// Update method
func (m simpleModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.updateListSize()
		return m, nil

	case tea.KeyMsg:
		// Handle text input states first to avoid hotkey conflicts
		if m.state == simpleStateCustomTable {
			return m.handleCustomTableInput(msg)
		} else if m.state == simpleStateQueryFilter {
			return m.handleQueryFilterInput(msg)
		} else if m.state == simpleStateXMLSearch {
			return m.handleXMLSearchInput(msg)
		}

		// Handle search navigation when in record detail (before regular hotkeys)
		if m.state == simpleStateRecordDetail && len(m.xmlSearchResults) > 0 {
			switch {
			case key.Matches(msg, key.NewBinding(key.WithKeys("n"))):
				m.navigateSearchResult(1)
				return m, nil
			case key.Matches(msg, key.NewBinding(key.WithKeys("N"))):
				m.navigateSearchResult(-1)
				return m, nil
			}
		}

		// Regular hotkey handling for other states
		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, m.keys.Back):
			return m.handleBack()
		case key.Matches(msg, m.keys.Enter):
			return m.handleEnter()
		case key.Matches(msg, m.keys.NextPage):
			return m.handleNextPage()
		case key.Matches(msg, m.keys.PrevPage):
			return m.handlePrevPage()
		case key.Matches(msg, m.keys.Refresh):
			return m.handleRefresh()
		case key.Matches(msg, m.keys.CustomTable):
			return m.handleCustomTable()
		case key.Matches(msg, m.keys.Filter):
			return m.handleFilter()
		case key.Matches(msg, m.keys.ViewXML):
			return m.handleViewXML()
		case key.Matches(msg, m.keys.Search):
			if m.state == simpleStateRecordDetail {
				return m.handleXMLSearch()
			}
		case key.Matches(msg, m.keys.Up):
			if m.state == simpleStateRecordDetail {
				return m.handleXMLScroll(-1)
			}
		case key.Matches(msg, m.keys.Down):
			if m.state == simpleStateRecordDetail {
				return m.handleXMLScroll(1)
			}
		}

	case recordsLoadedMsg:
		m.loading = false
		m.processLoadedRecords(msg.records, msg.total)
		return m, nil

	case recordsErrorMsg:
		m.loading = false
		if m.state == simpleStateRecordDetail {
			// Error in XML loading - show error in XML view
			m.recordXML = fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<record>
    <error>Failed to load record XML: %v</error>
    <!-- Press Esc to go back -->
</record>`, msg.err)
		} else {
			// Error in record loading - show in list
			items := []list.Item{
				simpleItem{title: "❌ Error Loading Records", desc: fmt.Sprintf("Failed to load records: %v", msg.err), id: "error"},
				simpleItem{title: "← Back to Table List", desc: "Return to table list", id: "back"},
			}
			m.list.SetItems(items)
		}
		return m, nil

	case recordXMLLoadedMsg:
		m.loading = false
		m.recordXML = msg.xml
		m.xmlScrollOffset = 0 // Reset scroll when loading new XML
		return m, nil
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}