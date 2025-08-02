package explorer

import (
	"fmt"
	"sort"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

// Update method
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.updateListSize()
		return m, nil

	case tea.KeyMsg:
		// Handle column customizer state first
		if m.state == simpleStateColumnCustomizer && m.columnCustomizer != nil {
			var cmd tea.Cmd
			m.columnCustomizer, cmd = m.columnCustomizer.Update(msg)
			if !m.columnCustomizer.IsActive() {
				// Apply selected columns and go back
				m.selectedColumns = m.columnCustomizer.GetSelectedColumns()
				m.state = simpleStateTableRecords
				m.loading = true
				if m.currentQuery != "" {
					return m, m.loadTableRecordsWithQueryCmd(m.currentTable, m.currentQuery)
				}
				return m, m.loadTableRecordsCmd(m.currentTable)
			}
			return m, cmd
		}

		// Handle advanced filtering states
		if m.state == simpleStateAdvancedFilter && m.queryBuilder != nil {
			var cmd tea.Cmd
			m.queryBuilder, cmd = m.queryBuilder.Update(msg)
			if !m.queryBuilder.IsActive() {
				// Query builder was closed, apply the query
				query := m.queryBuilder.BuildQuery()
				if query != "" {
					m.currentQuery = query
					// Apply the query by reloading records with the query
					m.state = simpleStateTableRecords
					m.loading = true
					m.currentPage = 0 // Reset to first page when applying new filter
					return m, m.loadTableRecordsWithQueryCmd(m.currentTable, query)
				}
				m.state = simpleStateTableRecords
			}
			return m, cmd
		} else if m.state == simpleStateFilterBrowser && m.filterBrowser != nil {
			var cmd tea.Cmd
			m.filterBrowser, cmd = m.filterBrowser.Update(msg)
			if !m.filterBrowser.IsActive() {
				m.state = simpleStateTableRecords
				// TODO: Handle filter selection
			}
			return m, cmd
		}

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

		// Handle view manager state
		if m.state == simpleStateViewManager {
			// Get compatible configurations for navigation
			compatibleConfigs := m.getCompatibleConfigurations()
			compatibleNames := make([]string, 0, len(compatibleConfigs))
			for name := range compatibleConfigs {
				compatibleNames = append(compatibleNames, name)
			}
			sort.Strings(compatibleNames)
			
			switch {
			case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
				// Always allow escape from view manager
				m.state = simpleStateTableRecords
				return m, nil
			case key.Matches(msg, key.NewBinding(key.WithKeys("q", "ctrl+c"))):
				// Allow quit from view manager
				return m, tea.Quit
			case key.Matches(msg, key.NewBinding(key.WithKeys("up", "k"))):
				if len(compatibleNames) > 0 {
					m.viewManagerSelection = (m.viewManagerSelection - 1 + len(compatibleNames)) % len(compatibleNames)
				}
				return m, nil
			case key.Matches(msg, key.NewBinding(key.WithKeys("down", "j"))):
				if len(compatibleNames) > 0 {
					m.viewManagerSelection = (m.viewManagerSelection + 1) % len(compatibleNames)
				}
				return m, nil
			case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
				// Apply selected view configuration (only if we have configurations)
				if len(compatibleNames) > 0 && m.viewManagerSelection >= 0 && m.viewManagerSelection < len(compatibleNames) {
					selectedName := compatibleNames[m.viewManagerSelection]
					config := compatibleConfigs[selectedName]
					if config != nil {
						// Apply the configuration
						m.selectedColumns = make([]string, len(config.Columns))
						copy(m.selectedColumns, config.Columns)
						m.currentQuery = config.Query
						m.state = simpleStateTableRecords
						m.loading = true
						m.currentPage = 0
						if config.Query != "" {
							return m, m.loadTableRecordsWithQueryCmd(m.currentTable, config.Query)
						}
						return m, m.loadTableRecordsCmd(m.currentTable)
					}
				} else if len(compatibleNames) == 0 {
					// No configurations available, just go back
					m.state = simpleStateTableRecords
					return m, nil
				}
				return m, nil
			case key.Matches(msg, key.NewBinding(key.WithKeys("d"))):
				// Delete selected view configuration (only if we have configurations)
				if len(compatibleNames) > 0 && m.viewManagerSelection >= 0 && m.viewManagerSelection < len(compatibleNames) {
					selectedName := compatibleNames[m.viewManagerSelection]
					delete(m.viewConfigurations, selectedName)
					// Also delete from persistent storage
					if m.configManager != nil {
						m.configManager.DeleteViewConfiguration(selectedName)
					}
					// Adjust selection if needed after deletion
					if m.viewManagerSelection >= len(compatibleNames)-1 && len(compatibleNames) > 1 {
						m.viewManagerSelection = len(compatibleNames) - 2
					} else if len(compatibleNames) == 1 {
						// We just deleted the last item, reset selection
						m.viewManagerSelection = 0
					}
				}
				return m, nil
			}
			return m, nil
		}

		// Handle quit confirmation state
		if m.state == simpleStateQuitConfirm {
			switch {
			case key.Matches(msg, key.NewBinding(key.WithKeys("left", "h"))):
				m.quitConfirmSelection = 0 // No
				return m, nil
			case key.Matches(msg, key.NewBinding(key.WithKeys("right", "l"))):
				m.quitConfirmSelection = 1 // Yes
				return m, nil
			case key.Matches(msg, key.NewBinding(key.WithKeys("y", "Y"))):
				return m, tea.Quit
			case key.Matches(msg, key.NewBinding(key.WithKeys("n", "N"))):
				// Go back to previous state
				m.state = m.previousState
				return m, nil
			case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
				// Escape cancels quit (same as No)
				m.state = m.previousState
				return m, nil
			case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
				if m.quitConfirmSelection == 1 {
					return m, tea.Quit
				} else {
					m.state = m.previousState
					return m, nil
				}
			}
			return m, nil
		}

		// Regular hotkey handling for other states
		switch {
		case key.Matches(msg, m.keys.Quit):
			// Show quit confirmation instead of immediately quitting
			if m.state != simpleStateQuitConfirm {
				m.previousState = m.state
				m.state = simpleStateQuitConfirm
				m.quitConfirmSelection = 0 // Default to "No"
				return m, nil
			}
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
			// Only allow custom table input from table list state
			if m.state == simpleStateTableList {
				return m.handleCustomTable()
			}
		case key.Matches(msg, m.keys.Filter):
			return m.handleFilter()
		case key.Matches(msg, m.keys.ViewXML):
			return m.handleViewXML()
		case key.Matches(msg, m.keys.Search):
			if m.state == simpleStateRecordDetail {
				return m.handleXMLSearch()
			}
		case key.Matches(msg, m.keys.ColumnCustomizer):
			return m.handleColumnCustomizer()
		case key.Matches(msg, m.keys.SaveView):
			return m.handleSaveView()
		case key.Matches(msg, m.keys.ViewManager):
			return m.handleViewManager()
		case key.Matches(msg, m.keys.ResetColumns):
			return m.handleResetColumns()
		// Disabled advanced features - keeping simple filter only
		// case key.Matches(msg, key.NewBinding(key.WithKeys("a"))):
		//	if m.state == simpleStateTableRecords {
		//		return m.handleAdvancedQuery()
		//	}
		// case key.Matches(msg, key.NewBinding(key.WithKeys("b"))):
		//	if m.state == simpleStateTableRecords {
		//		return m.handleFilterBrowser()
		//	}
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