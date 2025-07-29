package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Krive/ServiceNow-Toolkit/pkg/servicenow"
	"github.com/Krive/ServiceNow-Toolkit/pkg/servicenow/query"
)

// CMDB Browser states
type cmdbBrowserState int

const (
	cmdbStateMenu cmdbBrowserState = iota
	cmdbStateClassList
	cmdbStateItemList
	cmdbStateItemDetail
	cmdbStateRelationships
	cmdbStateSearch
)

// CMDB Browser Model
type CMDBBrowserModel struct {
	state          cmdbBrowserState
	client         *servicenow.Client
	
	// UI Components
	list           list.Model
	table          table.Model
	textInput      textinput.Model
	
	// Data
	classes        []CMDBClass
	items          []map[string]interface{}
	currentClass   string
	selectedItem   map[string]interface{}
	relationships  []CMDBRelationship
	searchResults  []map[string]interface{}
	
	// Navigation
	breadcrumb     []string
	
	// Dimensions
	width, height  int
	
	// Loading
	loading        bool
	errorMsg       string
}

// CMDB Class represents a CI class
type CMDBClass struct {
	Name        string
	Label       string
	Table       string
	Description string
	Count       int
}

// CMDB Relationship represents a CI relationship
type CMDBRelationship struct {
	Type        string
	Direction   string
	TargetClass string
	TargetName  string
	TargetSysID string
}

// CMDB Menu Items
type cmdbMenuItem struct {
	title       string
	description string
	action      string
}

func (m cmdbMenuItem) Title() string       { return m.title }
func (m cmdbMenuItem) Description() string { return m.description }
func (m cmdbMenuItem) FilterValue() string { return m.title }

// NewCMDBBrowser creates a new CMDB browser
func NewCMDBBrowser(client *servicenow.Client) *CMDBBrowserModel {
	// Initialize menu items
	items := []list.Item{
		cmdbMenuItem{
			title:       "🖥️  Servers",
			description: "Physical and virtual servers",
			action:      "cmdb_ci_server",
		},
		cmdbMenuItem{
			title:       "💻 Computers",
			description: "Desktop and laptop computers",
			action:      "cmdb_ci_computer",
		},
		cmdbMenuItem{
			title:       "🌐 Network Equipment",
			description: "Routers, switches, firewalls",
			action:      "cmdb_ci_netgear",
		},
		cmdbMenuItem{
			title:       "📱 Applications",
			description: "Business applications",
			action:      "cmdb_ci_appl",
		},
		cmdbMenuItem{
			title:       "💾 Databases",
			description: "Database instances",
			action:      "cmdb_ci_db_instance",
		},
		cmdbMenuItem{
			title:       "☁️  Cloud Resources",
			description: "Cloud infrastructure",
			action:      "cmdb_ci_cloud_service",
		},
		cmdbMenuItem{
			title:       "🔍 Search CIs",
			description: "Search across all configuration items",
			action:      "search",
		},
		cmdbMenuItem{
			title:       "📊 Class Overview",
			description: "Browse all CI classes",
			action:      "classes",
		},
	}

	// Create list component
	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "CMDB Explorer"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)

	// Initialize text input
	ti := textinput.New()
	ti.Placeholder = "Search configuration items..."
	ti.Focus()

	// Initialize table
	columns := []table.Column{
		{Title: "Name", Width: 30},
		{Title: "Class", Width: 20},
		{Title: "State", Width: 15},
		{Title: "Environment", Width: 15},
	}
	
	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(20),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(s)

	return &CMDBBrowserModel{
		state:     cmdbStateMenu,
		client:    client,
		list:      l,
		table:     t,
		textInput: ti,
		breadcrumb: []string{"CMDB"},
	}
}

// Init initializes the CMDB browser
func (m *CMDBBrowserModel) Init() tea.Cmd {
	// Initialize with static menu items - no loading needed
	return nil
}

// Update handles messages
func (m *CMDBBrowserModel) Update(msg tea.Msg) (*CMDBBrowserModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.list.SetSize(msg.Width-4, msg.Height-8)
		m.table.SetWidth(msg.Width - 4)
		m.table.SetHeight(msg.Height - 8)
		return m, nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
			return m.handleBack()
		case key.Matches(msg, key.NewBinding(key.WithKeys("q"))):
			return m, tea.Quit
		}

		switch m.state {
		case cmdbStateMenu:
			return m.updateMenu(msg)
		case cmdbStateClassList:
			return m.updateClassList(msg)
		case cmdbStateItemList:
			return m.updateItemList(msg)
		case cmdbStateItemDetail:
			return m.updateItemDetail(msg)
		case cmdbStateRelationships:
			return m.updateRelationships(msg)
		case cmdbStateSearch:
			return m.updateSearch(msg)
		}

	case cmdbLoadCompleteMsg:
		m.loading = false
		m.items = msg.items
		if m.state == cmdbStateClassList {
			m.state = cmdbStateItemList
			return m.updateItemListView()
		}
		return m, nil

	case cmdbErrorMsg:
		m.loading = false
		m.errorMsg = string(msg)
		return m, nil
	}

	return m, cmd
}

// Handle menu updates
func (m *CMDBBrowserModel) updateMenu(msg tea.KeyMsg) (*CMDBBrowserModel, tea.Cmd) {
	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
		if item, ok := m.list.SelectedItem().(cmdbMenuItem); ok {
			return m.handleMenuSelection(item)
		}
	}
	
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// Handle class list updates
func (m *CMDBBrowserModel) updateClassList(msg tea.KeyMsg) (*CMDBBrowserModel, tea.Cmd) {
	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

// Handle item list updates
func (m *CMDBBrowserModel) updateItemList(msg tea.KeyMsg) (*CMDBBrowserModel, tea.Cmd) {
	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
		return m.viewItemDetail()
	case key.Matches(msg, key.NewBinding(key.WithKeys("r"))):
		return m.viewRelationships()
	}
	
	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

// Handle item detail updates
func (m *CMDBBrowserModel) updateItemDetail(msg tea.KeyMsg) (*CMDBBrowserModel, tea.Cmd) {
	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("r"))):
		return m.viewRelationships()
	}
	return m, nil
}

// Handle relationships updates
func (m *CMDBBrowserModel) updateRelationships(msg tea.KeyMsg) (*CMDBBrowserModel, tea.Cmd) {
	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

// Handle search updates
func (m *CMDBBrowserModel) updateSearch(msg tea.KeyMsg) (*CMDBBrowserModel, tea.Cmd) {
	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
		if m.textInput.Value() != "" {
			return m.performSearch()
		}
	}
	
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

// Handle menu selection
func (m *CMDBBrowserModel) handleMenuSelection(item cmdbMenuItem) (*CMDBBrowserModel, tea.Cmd) {
	switch item.action {
	case "search":
		m.state = cmdbStateSearch
		m.breadcrumb = append(m.breadcrumb, "Search")
		return m, textinput.Blink
	case "classes":
		return m.loadClassOverview()
	default:
		// Load specific CI class
		m.currentClass = item.action
		m.breadcrumb = append(m.breadcrumb, item.title)
		return m.loadClassItems(item.action)
	}
}

// Handle back navigation
func (m *CMDBBrowserModel) handleBack() (*CMDBBrowserModel, tea.Cmd) {
	if len(m.breadcrumb) > 1 {
		m.breadcrumb = m.breadcrumb[:len(m.breadcrumb)-1]
		
		switch m.state {
		case cmdbStateSearch, cmdbStateClassList:
			m.state = cmdbStateMenu
		case cmdbStateItemList:
			m.state = cmdbStateMenu
		case cmdbStateItemDetail, cmdbStateRelationships:
			m.state = cmdbStateItemList
			return m.updateItemListView()
		}
	}
	return m, nil
}

// Load class overview
func (m *CMDBBrowserModel) loadClassOverview() (*CMDBBrowserModel, tea.Cmd) {
	m.loading = true
	m.state = cmdbStateClassList
	m.breadcrumb = append(m.breadcrumb, "Classes")
	
	return m, func() tea.Msg {
		// Mock data for now - in real implementation, query sys_db_object
		classes := []CMDBClass{
			{Name: "cmdb_ci_server", Label: "Server", Table: "cmdb_ci_server", Description: "Physical and virtual servers", Count: 150},
			{Name: "cmdb_ci_computer", Label: "Computer", Table: "cmdb_ci_computer", Description: "Desktop and laptop computers", Count: 2500},
			{Name: "cmdb_ci_netgear", Label: "Network Gear", Table: "cmdb_ci_netgear", Description: "Network equipment", Count: 75},
			{Name: "cmdb_ci_appl", Label: "Application", Table: "cmdb_ci_appl", Description: "Business applications", Count: 200},
			{Name: "cmdb_ci_db_instance", Label: "Database", Table: "cmdb_ci_db_instance", Description: "Database instances", Count: 45},
		}
		
		items := make([]map[string]interface{}, len(classes))
		for i, class := range classes {
			items[i] = map[string]interface{}{
				"name":        class.Name,
				"label":       class.Label,
				"table":       class.Table,
				"description": class.Description,
				"count":       fmt.Sprintf("%d", class.Count),
			}
		}
		
		return cmdbLoadCompleteMsg{items: items}
	}
}

// Load items for a specific class
func (m *CMDBBrowserModel) loadClassItems(className string) (*CMDBBrowserModel, tea.Cmd) {
	m.loading = true
	m.state = cmdbStateClassList // Will transition to cmdbStateItemList on load complete
	
	return m, func() tea.Msg {
		if m.client == nil {
			// Demo mode - return mock data
			return m.getMockClassItems(className)
		}
		
		// Real implementation
		records, err := m.client.Table(className).
			Where("sys_id", query.OpIsNotEmpty, "").
			Limit(50).
			Execute()
		if err != nil {
			return cmdbErrorMsg(fmt.Sprintf("Failed to load %s: %v", className, err))
		}
		
		return cmdbLoadCompleteMsg{items: records}
	}
}

// Get mock data for demo
func (m *CMDBBrowserModel) getMockClassItems(className string) cmdbLoadCompleteMsg {
	switch className {
	case "cmdb_ci_server":
		return cmdbLoadCompleteMsg{items: []map[string]interface{}{
			{"name": "PROD-WEB-01", "class": "Linux Server", "state": "Operational", "environment": "Production"},
			{"name": "DEV-DB-02", "class": "Database Server", "state": "Operational", "environment": "Development"},
			{"name": "TEST-APP-03", "class": "Application Server", "state": "Non-Operational", "environment": "Test"},
		}}
	case "cmdb_ci_computer":
		return cmdbLoadCompleteMsg{items: []map[string]interface{}{
			{"name": "LAPTOP-001", "class": "Laptop", "state": "In Use", "environment": "Corporate"},
			{"name": "DESKTOP-045", "class": "Desktop", "state": "Available", "environment": "Corporate"},
			{"name": "WS-DEV-12", "class": "Workstation", "state": "In Use", "environment": "Development"},
		}}
	default:
		return cmdbLoadCompleteMsg{items: []map[string]interface{}{
			{"name": "Sample CI", "class": className, "state": "Unknown", "environment": "Demo"},
		}}
	}
}

// Update item list view
func (m *CMDBBrowserModel) updateItemListView() (*CMDBBrowserModel, tea.Cmd) {
	rows := make([]table.Row, len(m.items))
	for i, item := range m.items {
		name := getStringValue(item, "name")
		class := getStringValue(item, "class")
		state := getStringValue(item, "state")
		env := getStringValue(item, "environment")
		
		rows[i] = table.Row{name, class, state, env}
	}
	
	m.table.SetRows(rows)
	return m, nil
}

// View item detail
func (m *CMDBBrowserModel) viewItemDetail() (*CMDBBrowserModel, tea.Cmd) {
	if len(m.items) == 0 {
		return m, nil
	}
	
	selectedIdx := m.table.Cursor()
	if selectedIdx >= len(m.items) {
		return m, nil
	}
	
	m.selectedItem = m.items[selectedIdx]
	m.state = cmdbStateItemDetail
	
	itemName := getStringValue(m.selectedItem, "name")
	if itemName == "" {
		itemName = "Item"
	}
	m.breadcrumb = append(m.breadcrumb, itemName)
	
	return m, nil
}

// View relationships
func (m *CMDBBrowserModel) viewRelationships() (*CMDBBrowserModel, tea.Cmd) {
	m.state = cmdbStateRelationships
	m.breadcrumb = append(m.breadcrumb, "Relationships")
	
	// Mock relationships data
	m.relationships = []CMDBRelationship{
		{Type: "Runs on", Direction: "Outbound", TargetClass: "Server", TargetName: "PROD-WEB-01", TargetSysID: "123"},
		{Type: "Uses", Direction: "Outbound", TargetClass: "Database", TargetName: "PROD-DB-01", TargetSysID: "456"},
		{Type: "Hosted by", Direction: "Inbound", TargetClass: "Application", TargetName: "CRM System", TargetSysID: "789"},
	}
	
	// Update table for relationships
	columns := []table.Column{
		{Title: "Type", Width: 20},
		{Title: "Direction", Width: 15},
		{Title: "Target Class", Width: 20},
		{Title: "Target Name", Width: 25},
	}
	m.table.SetColumns(columns)
	
	rows := make([]table.Row, len(m.relationships))
	for i, rel := range m.relationships {
		rows[i] = table.Row{rel.Type, rel.Direction, rel.TargetClass, rel.TargetName}
	}
	m.table.SetRows(rows)
	
	return m, nil
}

// Perform search
func (m *CMDBBrowserModel) performSearch() (*CMDBBrowserModel, tea.Cmd) {
	searchQuery := m.textInput.Value()
	m.loading = true
	
	return m, func() tea.Msg {
		if m.client == nil {
			// Demo mode
			return cmdbLoadCompleteMsg{items: []map[string]interface{}{
				{"name": fmt.Sprintf("Search result for '%s'", searchQuery), "class": "Demo", "state": "Found", "environment": "Demo"},
			}}
		}
		
		// Real search across multiple CI tables
		tables := []string{"cmdb_ci_server", "cmdb_ci_computer", "cmdb_ci_netgear", "cmdb_ci_appl"}
		var allResults []map[string]interface{}
		
		for _, table := range tables {
			results, err := m.client.Table(table).
				Where("name", query.OpContains, searchQuery).
				Limit(10).
				Execute()
			if err == nil {
				allResults = append(allResults, results...)
			}
		}
		
		return cmdbLoadCompleteMsg{items: allResults}
	}
}

// View renders the CMDB browser
func (m *CMDBBrowserModel) View() string {
	if m.loading {
		return "Loading CMDB data..."
	}
	
	if m.errorMsg != "" {
		return ErrorStyle.Render("Error: " + m.errorMsg)
	}
	
	// Header with breadcrumb
	header := HeaderStyle.Render(
		TitleStyle.Render("CMDB Explorer") + " " +
		BreadcrumbStyle.Render("/ "+strings.Join(m.breadcrumb, " / ")),
	)
	
	var content string
	
	switch m.state {
	case cmdbStateMenu:
		content = m.list.View()
		
	case cmdbStateClassList, cmdbStateItemList:
		content = m.table.View()
		
	case cmdbStateItemDetail:
		content = m.renderItemDetail()
		
	case cmdbStateRelationships:
		content = m.table.View() + "\n\n" + 
			InfoStyle.Render("Press 'r' to view relationships, 'esc' to go back")
		
	case cmdbStateSearch:
		searchBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Padding(1, 2).
			Render(m.textInput.View())
		
		content = fmt.Sprintf(
			"Search Configuration Items\n\n%s\n\nEnter search query and press Enter",
			searchBox,
		)
		
		if len(m.searchResults) > 0 {
			content += "\n\n" + m.table.View()
		}
	}
	
	// Footer with help
	footer := FooterStyle.Render("↑/↓: navigate • enter: select • r: relationships • esc: back • q: quit")
	
	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		ContentStyle.Render(content),
		footer,
	)
}

// Render item detail
func (m *CMDBBrowserModel) renderItemDetail() string {
	if m.selectedItem == nil {
		return "No item selected"
	}
	
	var details strings.Builder
	details.WriteString("Configuration Item Details\n\n")
	
	// Basic info
	details.WriteString(lipgloss.NewStyle().Bold(true).Render("Basic Information:"))
	details.WriteString("\n")
	details.WriteString(fmt.Sprintf("Name: %s\n", getStringValue(m.selectedItem, "name")))
	details.WriteString(fmt.Sprintf("Class: %s\n", getStringValue(m.selectedItem, "class")))
	details.WriteString(fmt.Sprintf("State: %s\n", getStringValue(m.selectedItem, "state")))
	details.WriteString(fmt.Sprintf("Environment: %s\n", getStringValue(m.selectedItem, "environment")))
	
	// Additional fields
	details.WriteString("\n")
	details.WriteString(lipgloss.NewStyle().Bold(true).Render("Additional Fields:"))
	details.WriteString("\n")
	
	for key, value := range m.selectedItem {
		if key != "name" && key != "class" && key != "state" && key != "environment" {
			details.WriteString(fmt.Sprintf("%s: %v\n", key, value))
		}
	}
	
	details.WriteString("\n")
	details.WriteString(InfoStyle.Render("Press 'r' to view relationships, 'esc' to go back"))
	
	return details.String()
}

// Message types
type cmdbLoadCompleteMsg struct {
	items []map[string]interface{}
}

type cmdbErrorMsg string

// Helper function to safely get string values
func getStringValue(item map[string]interface{}, key string) string {
	if val, ok := item[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
		return fmt.Sprintf("%v", val)
	}
	return ""
}