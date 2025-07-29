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
)

// Analytics Browser states
type analyticsBrowserState int

const (
	analyticsStateMenu analyticsBrowserState = iota
	analyticsStateTableSelect
	analyticsStateMetricSelect
	analyticsStateResults
	analyticsStateCustomQuery
)

// Analytics Browser Model
type AnalyticsBrowserModel struct {
	state          analyticsBrowserState
	client         *servicenow.Client
	
	// UI Components
	list           list.Model
	table          table.Model
	textInput      textinput.Model
	
	// Data
	selectedTable  string
	selectedMetric AnalyticsMetric
	results        []AnalyticsResult
	customQuery    string
	
	// Navigation
	breadcrumb     []string
	
	// Dimensions
	width, height  int
	
	// Loading
	loading        bool
	errorMsg       string
}

// Analytics Metric represents a metric that can be computed
type AnalyticsMetric struct {
	Name        string
	Description string
	Type        string // count, sum, avg, max, min, group_by
	Field       string
	GroupBy     string
}

// Analytics Result represents the result of an analytics query
type AnalyticsResult struct {
	Label string
	Value interface{}
	Count int
}

// Analytics Menu Items
type analyticsMenuItem struct {
	title       string
	description string
	action      string
}

func (m analyticsMenuItem) Title() string       { return m.title }
func (m analyticsMenuItem) Description() string { return m.description }
func (m analyticsMenuItem) FilterValue() string { return m.title }

// NewAnalyticsBrowser creates a new analytics browser
func NewAnalyticsBrowser(client *servicenow.Client) *AnalyticsBrowserModel {
	// Initialize menu items
	items := []list.Item{
		analyticsMenuItem{
			title:       "📊 Incident Analytics",
			description: "Analyze incident patterns and metrics",
			action:      "incident",
		},
		analyticsMenuItem{
			title:       "🔧 Problem Analytics",
			description: "Problem management statistics",
			action:      "problem",
		},
		analyticsMenuItem{
			title:       "📋 Change Analytics",
			description: "Change request analysis",
			action:      "change_request",
		},
		analyticsMenuItem{
			title:       "👥 User Analytics",
			description: "User activity and statistics",
			action:      "sys_user",
		},
		analyticsMenuItem{
			title:       "🏗️  CMDB Analytics",
			description: "Configuration item metrics",
			action:      "cmdb_ci",
		},
		analyticsMenuItem{
			title:       "📈 Custom Query",
			description: "Build custom analytics queries",
			action:      "custom",
		},
		analyticsMenuItem{
			title:       "📋 All Tables",
			description: "Browse all available tables",
			action:      "tables",
		},
	}

	// Create list component
	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Analytics & Aggregation"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)

	// Initialize text input
	ti := textinput.New()
	ti.Placeholder = "Enter custom query..."
	ti.Focus()

	// Initialize table
	columns := []table.Column{
		{Title: "Metric", Width: 30},
		{Title: "Value", Width: 20},
		{Title: "Count", Width: 15},
		{Title: "Percentage", Width: 15},
	}
	
	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(15),
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

	return &AnalyticsBrowserModel{
		state:     analyticsStateMenu,
		client:    client,
		list:      l,
		table:     t,
		textInput: ti,
		breadcrumb: []string{"Analytics"},
	}
}

// Init initializes the analytics browser
func (m *AnalyticsBrowserModel) Init() tea.Cmd {
	// Initialize with static menu items - no loading needed
	return nil
}

// Update handles messages
func (m *AnalyticsBrowserModel) Update(msg tea.Msg) (*AnalyticsBrowserModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.list.SetSize(msg.Width-4, msg.Height-8)
		m.table.SetWidth(msg.Width - 4)
		m.table.SetHeight(msg.Height - 10)
		return m, nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
			return m.handleBack()
		case key.Matches(msg, key.NewBinding(key.WithKeys("q"))):
			return m, tea.Quit
		}

		switch m.state {
		case analyticsStateMenu:
			return m.updateMenu(msg)
		case analyticsStateTableSelect:
			return m.updateTableSelect(msg)
		case analyticsStateMetricSelect:
			return m.updateMetricSelect(msg)
		case analyticsStateResults:
			return m.updateResults(msg)
		case analyticsStateCustomQuery:
			return m.updateCustomQuery(msg)
		}

	case analyticsLoadCompleteMsg:
		m.loading = false
		m.results = msg.results
		m.state = analyticsStateResults
		return m.updateResultsView()

	case analyticsErrorMsg:
		m.loading = false
		m.errorMsg = string(msg)
		return m, nil
	}

	return m, cmd
}

// Handle menu updates
func (m *AnalyticsBrowserModel) updateMenu(msg tea.KeyMsg) (*AnalyticsBrowserModel, tea.Cmd) {
	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
		if item, ok := m.list.SelectedItem().(analyticsMenuItem); ok {
			return m.handleMenuSelection(item)
		}
	}
	
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// Handle table selection updates
func (m *AnalyticsBrowserModel) updateTableSelect(msg tea.KeyMsg) (*AnalyticsBrowserModel, tea.Cmd) {
	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
		return m.selectTable()
	}
	
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// Handle metric selection updates
func (m *AnalyticsBrowserModel) updateMetricSelect(msg tea.KeyMsg) (*AnalyticsBrowserModel, tea.Cmd) {
	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
		return m.executeAnalytics()
	}
	
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// Handle results updates
func (m *AnalyticsBrowserModel) updateResults(msg tea.KeyMsg) (*AnalyticsBrowserModel, tea.Cmd) {
	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

// Handle custom query updates
func (m *AnalyticsBrowserModel) updateCustomQuery(msg tea.KeyMsg) (*AnalyticsBrowserModel, tea.Cmd) {
	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
		if m.textInput.Value() != "" {
			return m.executeCustomQuery()
		}
	}
	
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

// Handle menu selection
func (m *AnalyticsBrowserModel) handleMenuSelection(item analyticsMenuItem) (*AnalyticsBrowserModel, tea.Cmd) {
	switch item.action {
	case "custom":
		m.state = analyticsStateCustomQuery
		m.breadcrumb = append(m.breadcrumb, "Custom Query")
		return m, textinput.Blink
	case "tables":
		return m.loadTableList()
	default:
		// Load specific table analytics
		m.selectedTable = item.action
		m.breadcrumb = append(m.breadcrumb, item.title)
		return m.loadTableAnalytics(item.action)
	}
}

// Handle back navigation
func (m *AnalyticsBrowserModel) handleBack() (*AnalyticsBrowserModel, tea.Cmd) {
	if len(m.breadcrumb) > 1 {
		m.breadcrumb = m.breadcrumb[:len(m.breadcrumb)-1]
		
		switch m.state {
		case analyticsStateTableSelect, analyticsStateCustomQuery:
			m.state = analyticsStateMenu
		case analyticsStateMetricSelect:
			m.state = analyticsStateTableSelect
		case analyticsStateResults:
			if m.selectedTable != "" {
				m.state = analyticsStateMetricSelect
			} else {
				m.state = analyticsStateMenu
			}
		}
	}
	return m, nil
}

// Load table list for selection
func (m *AnalyticsBrowserModel) loadTableList() (*AnalyticsBrowserModel, tea.Cmd) {
	m.state = analyticsStateTableSelect
	m.breadcrumb = append(m.breadcrumb, "Tables")
	
	// Create table selection list
	tables := []list.Item{
		analyticsMenuItem{title: "incident", description: "Incident Management", action: "incident"},
		analyticsMenuItem{title: "problem", description: "Problem Management", action: "problem"},
		analyticsMenuItem{title: "change_request", description: "Change Management", action: "change_request"},
		analyticsMenuItem{title: "sys_user", description: "User Records", action: "sys_user"},
		analyticsMenuItem{title: "sys_user_group", description: "User Groups", action: "sys_user_group"},
		analyticsMenuItem{title: "cmdb_ci_server", description: "CI Servers", action: "cmdb_ci_server"},
		analyticsMenuItem{title: "cmdb_ci_computer", description: "CI Computers", action: "cmdb_ci_computer"},
	}
	
	m.list.SetItems(tables)
	return m, nil
}

// Select a table for analytics
func (m *AnalyticsBrowserModel) selectTable() (*AnalyticsBrowserModel, tea.Cmd) {
	if item, ok := m.list.SelectedItem().(analyticsMenuItem); ok {
		m.selectedTable = item.action
		m.breadcrumb = append(m.breadcrumb, item.title)
		return m.loadTableAnalytics(item.action)
	}
	return m, nil
}

// Load analytics for a specific table
func (m *AnalyticsBrowserModel) loadTableAnalytics(tableName string) (*AnalyticsBrowserModel, tea.Cmd) {
	m.state = analyticsStateMetricSelect
	
	// Create metric selection list based on table
	metrics := m.getMetricsForTable(tableName)
	items := make([]list.Item, len(metrics))
	for i, metric := range metrics {
		items[i] = analyticsMenuItem{
			title:       metric.Name,
			description: metric.Description,
			action:      metric.Type,
		}
	}
	
	m.list.SetItems(items)
	return m, nil
}

// Get available metrics for a table
func (m *AnalyticsBrowserModel) getMetricsForTable(tableName string) []AnalyticsMetric {
	commonMetrics := []AnalyticsMetric{
		{Name: "Total Count", Description: "Total number of records", Type: "count"},
		{Name: "Records by State", Description: "Group by state field", Type: "group_by", GroupBy: "state"},
		{Name: "Records by Priority", Description: "Group by priority field", Type: "group_by", GroupBy: "priority"},
		{Name: "Created This Month", Description: "Records created in current month", Type: "date_range"},
	}
	
	switch tableName {
	case "incident":
		return append(commonMetrics, []AnalyticsMetric{
			{Name: "By Category", Description: "Group by incident category", Type: "group_by", GroupBy: "category"},
			{Name: "By Assignment Group", Description: "Group by assignment group", Type: "group_by", GroupBy: "assignment_group"},
			{Name: "Average Resolution Time", Description: "Average time to resolve", Type: "avg", Field: "resolve_time"},
		}...)
	case "sys_user":
		return []AnalyticsMetric{
			{Name: "Total Users", Description: "Total number of users", Type: "count"},
			{Name: "Active Users", Description: "Number of active users", Type: "count_active"},
			{Name: "Users by Department", Description: "Group by department", Type: "group_by", GroupBy: "department"},
			{Name: "Users by Role", Description: "Group by role", Type: "group_by", GroupBy: "sys_user_role"},
		}
	default:
		return commonMetrics
	}
}

// Execute analytics query
func (m *AnalyticsBrowserModel) executeAnalytics() (*AnalyticsBrowserModel, tea.Cmd) {
	if item, ok := m.list.SelectedItem().(analyticsMenuItem); ok {
		metric := AnalyticsMetric{
			Name: item.title,
			Type: item.action,
		}
		
		// Find the full metric details
		metrics := m.getMetricsForTable(m.selectedTable)
		for _, m := range metrics {
			if m.Name == item.title {
				metric = m
				break
			}
		}
		
		m.selectedMetric = metric
		m.loading = true
		
		return m, func() tea.Msg {
			return m.performAnalytics()
		}
	}
	return m, nil
}

// Execute custom query
func (m *AnalyticsBrowserModel) executeCustomQuery() (*AnalyticsBrowserModel, tea.Cmd) {
	m.customQuery = m.textInput.Value()
	m.loading = true
	
	return m, func() tea.Msg {
		// For demo purposes, return mock results
		return analyticsLoadCompleteMsg{
			results: []AnalyticsResult{
				{Label: "Custom Query Result", Value: "42", Count: 1},
				{Label: "Mock Data", Value: "Demo", Count: 100},
			},
		}
	}
}

// Perform analytics calculation
func (m *AnalyticsBrowserModel) performAnalytics() tea.Msg {
	if m.client == nil {
		// Demo mode - return mock data
		return m.getMockAnalyticsResults()
	}
	
	// Real implementation would use the aggregation API
	switch m.selectedMetric.Type {
	case "count":
		return m.performCountAnalytics()
	case "group_by":
		return m.performGroupByAnalytics()
	case "avg":
		return m.performAverageAnalytics()
	default:
		return analyticsErrorMsg("Analytics type not implemented yet")
	}
}

// Perform count analytics
func (m *AnalyticsBrowserModel) performCountAnalytics() tea.Msg {
	// In real implementation, this would use client.Aggregate()
	return analyticsLoadCompleteMsg{
		results: []AnalyticsResult{
			{Label: "Total Records", Value: 1234, Count: 1234},
		},
	}
}

// Perform group by analytics
func (m *AnalyticsBrowserModel) performGroupByAnalytics() tea.Msg {
	// Mock data for group by operations
	switch m.selectedTable {
	case "incident":
		return analyticsLoadCompleteMsg{
			results: []AnalyticsResult{
				{Label: "New", Value: 45, Count: 45},
				{Label: "In Progress", Value: 32, Count: 32},
				{Label: "Resolved", Value: 123, Count: 123},
				{Label: "Closed", Value: 89, Count: 89},
			},
		}
	default:
		return analyticsLoadCompleteMsg{
			results: []AnalyticsResult{
				{Label: "Category A", Value: 25, Count: 25},
				{Label: "Category B", Value: 35, Count: 35},
				{Label: "Category C", Value: 40, Count: 40},
			},
		}
	}
}

// Perform average analytics
func (m *AnalyticsBrowserModel) performAverageAnalytics() tea.Msg {
	return analyticsLoadCompleteMsg{
		results: []AnalyticsResult{
			{Label: "Average Resolution Time", Value: "2.5 days", Count: 289},
		},
	}
}

// Get mock analytics results for demo
func (m *AnalyticsBrowserModel) getMockAnalyticsResults() analyticsLoadCompleteMsg {
	switch m.selectedTable {
	case "incident":
		return analyticsLoadCompleteMsg{
			results: []AnalyticsResult{
				{Label: "High Priority", Value: 23, Count: 23},
				{Label: "Medium Priority", Value: 67, Count: 67},
				{Label: "Low Priority", Value: 145, Count: 145},
			},
		}
	case "sys_user":
		return analyticsLoadCompleteMsg{
			results: []AnalyticsResult{
				{Label: "Active Users", Value: 1234, Count: 1234},
				{Label: "Inactive Users", Value: 56, Count: 56},
				{Label: "Locked Users", Value: 12, Count: 12},
			},
		}
	default:
		return analyticsLoadCompleteMsg{
			results: []AnalyticsResult{
				{Label: "Demo Metric 1", Value: 42, Count: 42},
				{Label: "Demo Metric 2", Value: 100, Count: 100},
				{Label: "Demo Metric 3", Value: 75, Count: 75},
			},
		}
	}
}

// Update results view
func (m *AnalyticsBrowserModel) updateResultsView() (*AnalyticsBrowserModel, tea.Cmd) {
	if len(m.results) == 0 {
		return m, nil
	}
	
	// Calculate total for percentages
	var total int
	for _, result := range m.results {
		total += result.Count
	}
	
	// Create table rows
	rows := make([]table.Row, len(m.results))
	for i, result := range m.results {
		percentage := ""
		if total > 0 {
			percentage = fmt.Sprintf("%.1f%%", float64(result.Count)*100/float64(total))
		}
		
		rows[i] = table.Row{
			result.Label,
			fmt.Sprintf("%v", result.Value),
			fmt.Sprintf("%d", result.Count),
			percentage,
		}
	}
	
	m.table.SetRows(rows)
	return m, nil
}

// View renders the analytics browser
func (m *AnalyticsBrowserModel) View() string {
	if m.loading {
		return "Computing analytics..."
	}
	
	if m.errorMsg != "" {
		return ErrorStyle.Render("Error: " + m.errorMsg)
	}
	
	// Header with breadcrumb
	header := HeaderStyle.Render(
		TitleStyle.Render("Analytics & Aggregation") + " " +
		BreadcrumbStyle.Render("/ "+strings.Join(m.breadcrumb, " / ")),
	)
	
	var content string
	
	switch m.state {
	case analyticsStateMenu, analyticsStateTableSelect, analyticsStateMetricSelect:
		content = m.list.View()
		
	case analyticsStateResults:
		content = m.table.View()
		if len(m.results) > 0 {
			content += "\n\n" + InfoStyle.Render("Analytics completed successfully")
		}
		
	case analyticsStateCustomQuery:
		queryBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Padding(1, 2).
			Render(m.textInput.View())
		
		content = fmt.Sprintf(
			"Custom Analytics Query\n\n%s\n\nEnter your custom analytics query and press Enter",
			queryBox,
		)
	}
	
	// Footer with help
	footer := FooterStyle.Render("↑/↓: navigate • enter: select • esc: back • q: quit")
	
	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		ContentStyle.Render(content),
		footer,
	)
}

// Message types
type analyticsLoadCompleteMsg struct {
	results []AnalyticsResult
}

type analyticsErrorMsg string