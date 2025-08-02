package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// FilterBrowserState represents the state of the filter browser
type FilterBrowserState int

const (
	FilterBrowserStateMain FilterBrowserState = iota
	FilterBrowserStateSaved
	FilterBrowserStateHistory
	FilterBrowserStateFavorites
	FilterBrowserStateSearch
	FilterBrowserStateSave
	FilterBrowserStateEdit
)

// FilterBrowserModel handles browsing and managing saved filters
type FilterBrowserModel struct {
	state         FilterBrowserState
	width, height int
	
	// Components
	mainList      list.Model
	filterList    list.Model
	searchInput   textinput.Model
	saveNameInput textinput.Model
	saveDescInput textinput.Model
	
	// Data
	filterManager *FilterManager
	tableName     string
	currentQuery  string
	currentConditions []QueryCondition
	
	// Selection
	selectedFilter *SavedFilter
	
	// State
	isActive      bool
	errorMsg      string
	successMsg    string
	showSaveForm  bool
	saveAsFavorite bool
}

// Filter list items
type filterListItem struct {
	filter *SavedFilter
}

func (i filterListItem) Title() string {
	title := i.filter.Name
	if i.filter.IsFavorite {
		title = "⭐ " + title
	}
	return title
}

func (i filterListItem) Description() string {
	desc := i.filter.Description
	if desc == "" {
		desc = i.filter.Query
		if len(desc) > 50 {
			desc = desc[:47] + "..."
		}
	}
	
	timeStr := "Last used: " + i.filter.LastUsed.Format("Jan 2, 15:04")
	if i.filter.UseCount > 0 {
		timeStr += fmt.Sprintf(" (%d uses)", i.filter.UseCount)
	}
	
	if desc != "" {
		return desc + " • " + timeStr
	}
	return timeStr
}

func (i filterListItem) FilterValue() string {
	return i.filter.Name + " " + i.filter.Description + " " + i.filter.Query
}

// History list items
type historyListItem struct {
	history *FilterHistory
}

func (i historyListItem) Title() string {
	if len(i.history.Query) > 60 {
		return i.history.Query[:57] + "..."
	}
	return i.history.Query
}

func (i historyListItem) Description() string {
	timeStr := i.history.Timestamp.Format("Jan 2, 15:04:05")
	resultStr := ""
	if i.history.Results > 0 {
		resultStr = fmt.Sprintf(" • %d results", i.history.Results)
	}
	if i.history.Duration > 0 {
		resultStr += fmt.Sprintf(" • %v", i.history.Duration.Round(time.Millisecond))
	}
	return timeStr + resultStr
}

func (i historyListItem) FilterValue() string {
	return i.history.Query
}

// Main menu items
type mainMenuItem struct {
	title string
	desc  string
	id    string
}

func (i mainMenuItem) Title() string       { return i.title }
func (i mainMenuItem) Description() string { return i.desc }
func (i mainMenuItem) FilterValue() string { return i.title }

// NewFilterBrowserModel creates a new filter browser
func NewFilterBrowserModel(filterManager *FilterManager) *FilterBrowserModel {
	// Main menu
	mainItems := []list.Item{
		mainMenuItem{"💾 Saved Filters", "View and manage saved filters", "saved"},
		mainMenuItem{"📋 Recent History", "Browse recent filter history", "history"},
		mainMenuItem{"⭐ Favorites", "Quick access to favorite filters", "favorites"},
		mainMenuItem{"🔍 Search Filters", "Search through saved filters", "search"},
	}
	
	mainList := list.New(mainItems, list.NewDefaultDelegate(), 0, 0)
	mainList.Title = "Filter Manager"
	mainList.SetShowStatusBar(false)
	mainList.SetFilteringEnabled(false)
	
	// Filter list
	filterList := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	filterList.SetShowStatusBar(true)
	filterList.SetFilteringEnabled(true)
	
	// Search input
	searchInput := textinput.New()
	searchInput.Placeholder = "Search filters..."
	searchInput.Width = 40
	
	// Save inputs
	saveNameInput := textinput.New()
	saveNameInput.Placeholder = "Filter name..."
	saveNameInput.Width = 40
	
	saveDescInput := textinput.New()
	saveDescInput.Placeholder = "Description (optional)..."
	saveDescInput.Width = 60
	
	return &FilterBrowserModel{
		state:         FilterBrowserStateMain,
		mainList:      mainList,
		filterList:    filterList,
		searchInput:   searchInput,
		saveNameInput: saveNameInput,
		saveDescInput: saveDescInput,
		filterManager: filterManager,
	}
}

// Init initializes the filter browser
func (m *FilterBrowserModel) Init() tea.Cmd {
	return nil
}

// Update handles filter browser updates
func (m *FilterBrowserModel) Update(msg tea.Msg) (*FilterBrowserModel, tea.Cmd) {
	var cmd tea.Cmd
	
	if !m.isActive {
		return m, nil
	}
	
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.mainList.SetSize(msg.Width-4, msg.Height-10)
		m.filterList.SetSize(msg.Width-4, msg.Height-10)
		return m, nil
		
	case tea.KeyMsg:
		// Clear messages on any key
		m.errorMsg = ""
		m.successMsg = ""
		
		switch m.state {
		case FilterBrowserStateMain:
			switch {
			case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
				if item, ok := m.mainList.SelectedItem().(mainMenuItem); ok {
					switch item.id {
					case "saved":
						m.loadSavedFilters()
						m.state = FilterBrowserStateSaved
					case "history":
						m.loadHistory()
						m.state = FilterBrowserStateHistory
					case "favorites":
						m.loadFavorites()
						m.state = FilterBrowserStateFavorites
					case "search":
						m.state = FilterBrowserStateSearch
						m.searchInput.Focus()
						return m, textinput.Blink
					}
				}
			case key.Matches(msg, key.NewBinding(key.WithKeys("s"))):
				if m.currentQuery != "" {
					m.showSaveDialog()
				}
			case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
				m.isActive = false
				return m, nil
			}
			m.mainList, cmd = m.mainList.Update(msg)
			return m, cmd
			
		case FilterBrowserStateSaved, FilterBrowserStateFavorites:
			switch {
			case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
				if item, ok := m.filterList.SelectedItem().(filterListItem); ok {
					m.selectedFilter = item.filter
					// Record usage
					m.filterManager.UseFilter(item.filter.ID)
					m.isActive = false
					return m, func() tea.Msg { return FilterSelectedMsg{Filter: item.filter} }
				}
			case key.Matches(msg, key.NewBinding(key.WithKeys("d"))):
				if item, ok := m.filterList.SelectedItem().(filterListItem); ok {
					if err := m.filterManager.DeleteFilter(item.filter.ID); err != nil {
						m.errorMsg = fmt.Sprintf("Failed to delete filter: %v", err)
					} else {
						m.successMsg = "Filter deleted successfully"
						// Reload the list
						if m.state == FilterBrowserStateSaved {
							m.loadSavedFilters()
						} else {
							m.loadFavorites()
						}
					}
				}
			case key.Matches(msg, key.NewBinding(key.WithKeys("f"))):
				if item, ok := m.filterList.SelectedItem().(filterListItem); ok {
					newFavoriteStatus := !item.filter.IsFavorite
					if err := m.filterManager.UpdateFilter(item.filter.ID, item.filter.Name, 
						item.filter.Description, newFavoriteStatus, item.filter.Tags); err != nil {
						m.errorMsg = fmt.Sprintf("Failed to update filter: %v", err)
					} else {
						if newFavoriteStatus {
							m.successMsg = "Added to favorites"
						} else {
							m.successMsg = "Removed from favorites"
						}
						// Reload the list
						if m.state == FilterBrowserStateSaved {
							m.loadSavedFilters()
						} else {
							m.loadFavorites()
						}
					}
				}
			case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
				m.state = FilterBrowserStateMain
			}
			m.filterList, cmd = m.filterList.Update(msg)
			return m, cmd
			
		case FilterBrowserStateHistory:
			switch {
			case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
				if item, ok := m.filterList.SelectedItem().(historyListItem); ok {
					m.isActive = false
					return m, func() tea.Msg { 
						return FilterSelectedMsg{Query: item.history.Query, Conditions: item.history.Conditions} 
					}
				}
			case key.Matches(msg, key.NewBinding(key.WithKeys("s"))):
				if item, ok := m.filterList.SelectedItem().(historyListItem); ok {
					m.currentQuery = item.history.Query
					m.currentConditions = item.history.Conditions
					m.showSaveDialog()
				}
			case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
				m.state = FilterBrowserStateMain
			}
			m.filterList, cmd = m.filterList.Update(msg)
			return m, cmd
			
		case FilterBrowserStateSearch:
			switch {
			case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
				query := strings.TrimSpace(m.searchInput.Value())
				if query != "" {
					m.loadSearchResults(query)
					m.state = FilterBrowserStateSaved // Reuse saved state for search results
				}
			case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
				m.state = FilterBrowserStateMain
				m.searchInput.Blur()
			}
			m.searchInput, cmd = m.searchInput.Update(msg)
			return m, cmd
			
		case FilterBrowserStateSave:
			switch {
			case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
				name := strings.TrimSpace(m.saveNameInput.Value())
				if name == "" {
					m.errorMsg = "Filter name is required"
				} else {
					desc := strings.TrimSpace(m.saveDescInput.Value())
					if _, err := m.filterManager.SaveFilter(name, desc, m.tableName, 
						m.currentQuery, m.currentConditions, m.saveAsFavorite); err != nil {
						m.errorMsg = fmt.Sprintf("Failed to save filter: %v", err)
					} else {
						m.successMsg = "Filter saved successfully"
						m.hideSaveDialog()
						m.state = FilterBrowserStateMain
					}
				}
			case key.Matches(msg, key.NewBinding(key.WithKeys("tab"))):
				if m.saveNameInput.Focused() {
					m.saveNameInput.Blur()
					m.saveDescInput.Focus()
					return m, textinput.Blink
				} else {
					m.saveDescInput.Blur()
					m.saveNameInput.Focus()
					return m, textinput.Blink
				}
			case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+f"))):
				m.saveAsFavorite = !m.saveAsFavorite
			case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
				m.hideSaveDialog()
				m.state = FilterBrowserStateMain
			}
			
			if m.saveNameInput.Focused() {
				m.saveNameInput, cmd = m.saveNameInput.Update(msg)
			} else {
				m.saveDescInput, cmd = m.saveDescInput.Update(msg)
			}
			return m, cmd
		}
	}
	
	return m, nil
}

// loadSavedFilters loads saved filters for the current table
func (m *FilterBrowserModel) loadSavedFilters() {
	filters := m.filterManager.GetFiltersForTable(m.tableName)
	items := make([]list.Item, len(filters))
	for i, filter := range filters {
		items[i] = filterListItem{filter: filter}
	}
	
	m.filterList.SetItems(items)
	m.filterList.Title = fmt.Sprintf("Saved Filters - %s", m.tableName)
}

// loadHistory loads filter history
func (m *FilterBrowserModel) loadHistory() {
	history := m.filterManager.GetHistory(m.tableName, 50)
	items := make([]list.Item, len(history))
	for i, entry := range history {
		items[i] = historyListItem{history: entry}
	}
	
	m.filterList.SetItems(items)
	m.filterList.Title = fmt.Sprintf("Filter History - %s", m.tableName)
}

// loadFavorites loads favorite filters
func (m *FilterBrowserModel) loadFavorites() {
	favorites := m.filterManager.GetFavoriteFilters()
	
	// Filter by table if specified
	if m.tableName != "" {
		var filtered []*SavedFilter
		for _, fav := range favorites {
			if fav.TableName == m.tableName {
				filtered = append(filtered, fav)
			}
		}
		favorites = filtered
	}
	
	items := make([]list.Item, len(favorites))
	for i, filter := range favorites {
		items[i] = filterListItem{filter: filter}
	}
	
	m.filterList.SetItems(items)
	m.filterList.Title = "Favorite Filters"
}

// loadSearchResults loads search results
func (m *FilterBrowserModel) loadSearchResults(query string) {
	filters := m.filterManager.SearchFilters(query)
	
	// Filter by table if specified
	if m.tableName != "" {
		var filtered []*SavedFilter
		for _, filter := range filters {
			if filter.TableName == m.tableName {
				filtered = append(filtered, filter)
			}
		}
		filters = filtered
	}
	
	items := make([]list.Item, len(filters))
	for i, filter := range filters {
		items[i] = filterListItem{filter: filter}
	}
	
	m.filterList.SetItems(items)
	m.filterList.Title = fmt.Sprintf("Search Results: \"%s\"", query)
}

// showSaveDialog shows the save filter dialog
func (m *FilterBrowserModel) showSaveDialog() {
	m.showSaveForm = true
	m.state = FilterBrowserStateSave
	m.saveNameInput.SetValue("")
	m.saveDescInput.SetValue("")
	m.saveAsFavorite = false
	m.saveNameInput.Focus()
}

// hideSaveDialog hides the save filter dialog
func (m *FilterBrowserModel) hideSaveDialog() {
	m.showSaveForm = false
	m.saveNameInput.Blur()
	m.saveDescInput.Blur()
}

// SetActive sets the active state
func (m *FilterBrowserModel) SetActive(active bool) {
	m.isActive = active
	if active {
		m.state = FilterBrowserStateMain
	}
}

// IsActive returns whether the browser is active
func (m *FilterBrowserModel) IsActive() bool {
	return m.isActive
}

// SetTableName sets the current table name for filtering
func (m *FilterBrowserModel) SetTableName(tableName string) {
	m.tableName = tableName
}

// SetCurrentQuery sets the current query for saving
func (m *FilterBrowserModel) SetCurrentQuery(query string, conditions []QueryCondition) {
	m.currentQuery = query
	m.currentConditions = conditions
}

// View renders the filter browser
func (m *FilterBrowserModel) View() string {
	if !m.isActive {
		return ""
	}
	
	var content strings.Builder
	
	// Header
	content.WriteString(headerStyle.Render("Filter Manager"))
	content.WriteString("\n\n")
	
	// Messages
	if m.errorMsg != "" {
		content.WriteString(errorStyle.Render("Error: " + m.errorMsg))
		content.WriteString("\n\n")
	}
	if m.successMsg != "" {
		content.WriteString(filterBrowserSuccessStyle.Render("✓ " + m.successMsg))
		content.WriteString("\n\n")
	}
	
	// Current state view
	switch m.state {
	case FilterBrowserStateMain:
		content.WriteString(m.mainList.View())
		content.WriteString("\n\nPress 's' to save current filter, Enter to select, Esc to close")
		
	case FilterBrowserStateSaved, FilterBrowserStateFavorites:
		if len(m.filterList.Items()) == 0 {
			content.WriteString("No filters found.")
		} else {
			content.WriteString(m.filterList.View())
		}
		content.WriteString("\n\nEnter: Use filter | d: Delete | f: Toggle favorite | Esc: Back")
		
	case FilterBrowserStateHistory:
		if len(m.filterList.Items()) == 0 {
			content.WriteString("No history found.")
		} else {
			content.WriteString(m.filterList.View())
		}
		content.WriteString("\n\nEnter: Use filter | s: Save filter | Esc: Back")
		
	case FilterBrowserStateSearch:
		content.WriteString("Search Filters\n\n")
		content.WriteString("Query: " + m.searchInput.View())
		content.WriteString("\n\nPress Enter to search, Esc to cancel")
		
	case FilterBrowserStateSave:
		content.WriteString("Save Filter\n\n")
		
		nameStyle := defaultInputStyle
		descStyle := defaultInputStyle
		if m.saveNameInput.Focused() {
			nameStyle = defaultFocusedInputStyle
		} else if m.saveDescInput.Focused() {
			descStyle = defaultFocusedInputStyle
		}
		
		content.WriteString("Name: " + nameStyle.Render(m.saveNameInput.View()))
		content.WriteString("\n\n")
		content.WriteString("Description: " + descStyle.Render(m.saveDescInput.View()))
		content.WriteString("\n\n")
		
		favoriteStr := "[ ]"
		if m.saveAsFavorite {
			favoriteStr = "[✓]"
		}
		content.WriteString(fmt.Sprintf("%s Add to favorites (Ctrl+F)", favoriteStr))
		
		content.WriteString("\n\nPress Tab to switch fields, Enter to save, Esc to cancel")
	}
	
	return content.String()
}

// FilterSelectedMsg is sent when a filter is selected
type FilterSelectedMsg struct {
	Filter     *SavedFilter     `json:"filter,omitempty"`
	Query      string           `json:"query,omitempty"`
	Conditions []QueryCondition `json:"conditions,omitempty"`
}

// Local styles for filter browser
var (
	filterBrowserSuccessStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("34")).
		Bold(true)

	defaultInputStyle = lipgloss.NewStyle().
		Padding(0, 1).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240"))

	defaultFocusedInputStyle = lipgloss.NewStyle().
		Padding(0, 1).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("205"))
)