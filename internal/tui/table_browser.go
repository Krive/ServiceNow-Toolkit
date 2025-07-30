package tui

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Krive/ServiceNow-Toolkit/pkg/servicenow"
	"github.com/Krive/ServiceNow-Toolkit/pkg/servicenow/query"
)

// Table browser states
type tableBrowserState int

const (
	tableBrowserStateTableList tableBrowserState = iota
	tableBrowserStateRecordList
	tableBrowserStateRecordDetail
	tableBrowserStateFilter
)

// Table browser model
type TableBrowserModel struct {
	client       *servicenow.Client
	state        tableBrowserState
	width, height int

	// UI Components
	tableList    list.Model
	recordTable  table.Model
	filterInput  textinput.Model

	// Data
	currentTable    string
	currentRecords  []map[string]interface{}
	currentRecord   map[string]interface{}
	availableTables []TableInfo
	columns         []table.Column
	rows            []table.Row

	// Navigation
	selectedIndex int
	
	// Filter state
	filterQuery     string
	isFiltering     bool
	filteredCount   int  // Count of records matching current filter
	
	// Loading
	loading       bool
	errorMsg      string
}

// Table information
type TableInfo struct {
	Name        string
	Label       string
	Desc        string
	RecordCount int
}

func (t TableInfo) Title() string       { return fmt.Sprintf("%s (%s)", t.Label, t.Name) }
func (t TableInfo) Description() string { 
	return fmt.Sprintf("%s - %d records", t.Desc, t.RecordCount)
}
func (t TableInfo) FilterValue() string { return t.Name + " " + t.Label + " " + t.Desc }

// Messages for table browser
type tableListLoadedMsg struct {
	tables []TableInfo
}

type recordsLoadedMsg struct {
	records []map[string]interface{}
}

type recordDetailLoadedMsg struct {
	record map[string]interface{}
}

type tableCountLoadedMsg struct {
	tableName string
	count     int
}

type filteredCountLoadedMsg struct {
	tableName string
	count     int
}

// Create new table browser
func NewTableBrowser(client *servicenow.Client) *TableBrowserModel {
	// Initialize table list
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = selectedItemStyle
	delegate.Styles.SelectedDesc = selectedItemDescStyle
	
	tableList := list.New([]list.Item{}, delegate, 0, 0)
	tableList.Title = "ServiceNow Tables"
	tableList.SetShowStatusBar(true)
	tableList.SetFilteringEnabled(true)
	tableList.Styles.Title = titleStyle

	// Initialize filter input
	filterInput := textinput.New()
	filterInput.Placeholder = "Enter filter query (e.g., state=1^priority=1)"
	filterInput.Width = 50

	// Initialize record table
	recordTable := table.New(
		table.WithColumns([]table.Column{}),
		table.WithRows([]table.Row{}),
		table.WithFocused(true),
		table.WithHeight(15),
	)
	recordTable.SetStyles(tableStyles())

	return &TableBrowserModel{
		client:      client,
		state:       tableBrowserStateTableList,
		tableList:   tableList,
		recordTable: recordTable,
		filterInput: filterInput,
	}
}

// Initialize table browser
func (m *TableBrowserModel) Init() tea.Cmd {
	return m.loadTableList()
}

// Update table browser
func (m *TableBrowserModel) Update(msg tea.Msg) (*TableBrowserModel, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.tableList.SetSize(msg.Width-4, msg.Height-10)
		m.recordTable.SetWidth(msg.Width - 4)
		m.recordTable.SetHeight(msg.Height - 15)
		return m, nil

	case tea.KeyMsg:
		switch m.state {
		case tableBrowserStateTableList:
			switch {
			case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
				if item, ok := m.tableList.SelectedItem().(TableInfo); ok {
					m.currentTable = item.Name
					m.state = tableBrowserStateRecordList
					m.filterQuery = ""     // Reset filter when entering new table
					m.filteredCount = 0    // Reset filtered count
					return m, m.loadRecords(item.Name, "")
				}
			case key.Matches(msg, key.NewBinding(key.WithKeys("r"))):
				return m, m.loadTableList()
			}
			m.tableList, cmd = m.tableList.Update(msg)
			return m, cmd

		case tableBrowserStateRecordList:
			switch {
			case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
				if len(m.rows) > 0 && m.recordTable.Cursor() < len(m.rows) {
					selectedRow := m.rows[m.recordTable.Cursor()]
					if len(selectedRow) > 0 {
						// Use sys_id (first column) to get full record
						sysID := selectedRow[0]
						m.state = tableBrowserStateRecordDetail
						return m, m.loadRecordDetail(m.currentTable, sysID)
					}
				}
			case key.Matches(msg, key.NewBinding(key.WithKeys("f"))):
				m.state = tableBrowserStateFilter
				m.isFiltering = true
				m.filterInput.Focus()
				return m, textinput.Blink
			case key.Matches(msg, key.NewBinding(key.WithKeys("r"))):
				return m, m.loadRecords(m.currentTable, m.filterQuery)
			case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
				m.state = tableBrowserStateTableList
				m.filterQuery = ""     // Reset filter when going back
				m.filteredCount = 0    // Reset filtered count
				return m, nil
			}
			m.recordTable, cmd = m.recordTable.Update(msg)
			return m, cmd

		case tableBrowserStateFilter:
			switch {
			case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
				m.filterQuery = m.filterInput.Value()
				m.state = tableBrowserStateRecordList
				m.isFiltering = false
				m.filterInput.Blur()
				m.filteredCount = 0  // Reset filtered count when applying new filter
				return m, m.loadRecords(m.currentTable, m.filterQuery)
			case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
				m.state = tableBrowserStateRecordList
				m.isFiltering = false
				m.filterInput.Blur()
				return m, nil
			}
			m.filterInput, cmd = m.filterInput.Update(msg)
			return m, cmd

		case tableBrowserStateRecordDetail:
			switch {
			case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
				m.state = tableBrowserStateRecordList
				return m, nil
			}
		}

	case tableListLoadedMsg:
		m.availableTables = msg.tables
		items := make([]list.Item, len(msg.tables))
		for i, table := range msg.tables {
			items[i] = table
		}
		m.tableList.SetItems(items)
		m.loading = false
		return m, nil

	case recordsLoadedMsg:
		m.currentRecords = msg.records
		m.updateRecordTable()
		m.loading = false
		return m, nil

	case recordDetailLoadedMsg:
		m.currentRecord = msg.record
		m.loading = false
		return m, nil

	case tableCountLoadedMsg:
		// Update the record count for the specific table
		for i, table := range m.availableTables {
			if table.Name == msg.tableName {
				m.availableTables[i].RecordCount = msg.count
				break
			}
		}
		// Refresh the list items
		items := make([]list.Item, len(m.availableTables))
		for i, table := range m.availableTables {
			items[i] = table
		}
		m.tableList.SetItems(items)
		return m, nil

	case filteredCountLoadedMsg:
		// Update the filtered count if it's for the current table
		if msg.tableName == m.currentTable {
			m.filteredCount = msg.count
		}
		return m, nil

	case errorMsg:
		m.errorMsg = string(msg)
		m.loading = false
		return m, nil
	}

	return m, tea.Batch(cmds...)
}

// Load table list
func (m *TableBrowserModel) loadTableList() tea.Cmd {
	// Common ServiceNow tables with descriptions and estimated record counts as fallback
	tables := []TableInfo{
		{Name: "incident", Label: "Incident", Desc: "Service incidents and disruptions", RecordCount: 1000},
		{Name: "sc_request", Label: "Service Request", Desc: "Service catalog requests", RecordCount: 500},
		{Name: "change_request", Label: "Change Request", Desc: "Change management requests", RecordCount: 200},
		{Name: "problem", Label: "Problem", Desc: "Problem management records", RecordCount: 50},
		{Name: "sys_user", Label: "User", Desc: "System users", RecordCount: 300},
		{Name: "sys_user_group", Label: "Group", Desc: "User groups", RecordCount: 25},
		{Name: "sys_user_role", Label: "Role", Desc: "User roles", RecordCount: 100},
		{Name: "cmdb_ci", Label: "Configuration Item", Desc: "CMDB configuration items", RecordCount: 2000},
		{Name: "cmdb_ci_computer", Label: "Computer", Desc: "Computer configuration items", RecordCount: 800},
		{Name: "cmdb_ci_server", Label: "Server", Desc: "Server configuration items", RecordCount: 150},
		{Name: "cmdb_ci_service", Label: "Service", Desc: "Service configuration items", RecordCount: 75},
		{Name: "sc_cat_item", Label: "Catalog Item", Desc: "Service catalog items", RecordCount: 100},
		{Name: "kb_knowledge", Label: "Knowledge Article", Desc: "Knowledge base articles", RecordCount: 300},
		{Name: "task", Label: "Task", Desc: "General task records", RecordCount: 5000},
		{Name: "sys_audit", Label: "Audit Log", Desc: "System audit trail", RecordCount: 10000},
	}
	
	// Set the tables with fallback counts
	m.availableTables = tables
	items := make([]list.Item, len(tables))
	for i, table := range tables {
		items[i] = table
	}
	m.tableList.SetItems(items)
	
	// Load actual record counts asynchronously using aggregate API
	var cmds []tea.Cmd
	for _, table := range tables {
		cmds = append(cmds, m.loadTableCount(table.Name))
	}
	
	return tea.Batch(cmds...)
}

// Load table record count using aggregate API
func (m *TableBrowserModel) loadTableCount(tableName string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Use aggregate API to get record count
		aggClient := m.client.Aggregate(tableName)
		count, err := aggClient.CountRecordsWithContext(ctx, nil)
		if err != nil {
			// If aggregate API fails, silently keep the fallback count
			// This ensures the UI remains functional even with limited API access
			return nil
		}

		return tableCountLoadedMsg{
			tableName: tableName,
			count:     count,
		}
	}
}

// Load filtered record count using aggregate API
func (m *TableBrowserModel) loadFilteredCount(tableName, filterQuery string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Use aggregate API to get filtered record count
		aggClient := m.client.Aggregate(tableName)
		
		// Create a query with the filter if provided
		aq := aggClient.NewQuery().CountAll("filtered_count")
		
		if filterQuery != "" {
			// Apply the raw filter query to the aggregate query
			// The aggregate API should accept raw queries through its Where clause
			qb := query.New()
			// Add the raw query as a single condition - we'll construct it manually
			// Since we can't use Raw(), we'll create a simple condition manually
			// This is a simplified approach - in production you'd want proper query parsing
			aq.Where("sys_id", query.OpNotEquals, "") // Always true condition to start
			// Apply the filter by manually constructing the query
			if strings.Contains(filterQuery, "=") {
				parts := strings.SplitN(filterQuery, "=", 2)
				if len(parts) == 2 {
					field := strings.TrimSpace(parts[0])
					value := strings.TrimSpace(parts[1])
					aq.Where(field, query.OpEquals, value)
				}
			}
		}
		
		result, err := aq.ExecuteWithContext(ctx)
		if err != nil {
			// If aggregate API fails, silently ignore
			return nil
		}

		count := 0
		if result.Stats != nil {
			if countVal, ok := result.Stats["filtered_count"]; ok {
				count = parseIntFromInterface(countVal)
			}
		}

		return filteredCountLoadedMsg{
			tableName: tableName,
			count:     count,
		}
	}
}

// Helper function to parse interface{} to int (copied from aggregate.go pattern)
func parseIntFromInterface(value interface{}) int {
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case string:
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return 0
}

// Load records from a table
func (m *TableBrowserModel) loadRecords(tableName, filterQuery string) tea.Cmd {
	m.loading = true
	
	// Load both records and filtered count
	loadRecordsCmd := func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Build parameters
		params := map[string]string{
			"sysparm_limit": "100",
		}
		
		if filterQuery != "" {
			params["sysparm_query"] = filterQuery
		}

		records, err := m.client.Table(tableName).ListWithContext(ctx, params)
		if err != nil {
			return errorMsg(fmt.Sprintf("Failed to load records from %s: %v", tableName, err))
		}

		return recordsLoadedMsg{records: records}
	}
	
	// Also load the filtered count if we have a filter
	if filterQuery != "" {
		return tea.Batch(loadRecordsCmd, m.loadFilteredCount(tableName, filterQuery))
	}
	
	return loadRecordsCmd
}

// Load detailed record
func (m *TableBrowserModel) loadRecordDetail(tableName, sysID string) tea.Cmd {
	m.loading = true
	
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		record, err := m.client.Table(tableName).GetWithContext(ctx, sysID)
		if err != nil {
			return errorMsg(fmt.Sprintf("Failed to load record detail: %v", err))
		}

		return recordDetailLoadedMsg{record: record}
	}
}

// Update record table with current data
func (m *TableBrowserModel) updateRecordTable() {
	if len(m.currentRecords) == 0 {
		m.columns = []table.Column{}
		m.rows = []table.Row{}
		m.recordTable.SetColumns(m.columns)
		m.recordTable.SetRows(m.rows)
		return
	}

	// Get column names from first record
	firstRecord := m.currentRecords[0]
	var columnNames []string
	
	// Prioritize important columns
	importantColumns := []string{"sys_id", "number", "state", "priority", "short_description", "sys_created_on"}
	
	// Add important columns first if they exist
	for _, col := range importantColumns {
		if _, exists := firstRecord[col]; exists {
			columnNames = append(columnNames, col)
		}
	}
	
	// Add remaining columns (limit to prevent UI overflow)
	maxColumns := 8
	for col := range firstRecord {
		if len(columnNames) >= maxColumns {
			break
		}
		
		// Skip if already added
		found := false
		for _, existing := range columnNames {
			if existing == col {
				found = true
				break
			}
		}
		if !found {
			columnNames = append(columnNames, col)
		}
	}

	// Create columns
	m.columns = make([]table.Column, len(columnNames))
	for i, name := range columnNames {
		width := 15 // Default width
		if name == "sys_id" {
			width = 32
		} else if name == "short_description" {
			width = 30
		} else if name == "number" {
			width = 20
		}
		
		m.columns[i] = table.Column{
			Title: strings.ToUpper(name),
			Width: width,
		}
	}

	// Create rows
	m.rows = make([]table.Row, len(m.currentRecords))
	for i, record := range m.currentRecords {
		row := make(table.Row, len(columnNames))
		for j, colName := range columnNames {
			if val, exists := record[colName]; exists {
				if val == nil {
					row[j] = ""
				} else {
					// Convert to string and truncate if necessary
					str := fmt.Sprintf("%v", val)
					if len(str) > m.columns[j].Width-2 {
						str = str[:m.columns[j].Width-5] + "..."
					}
					row[j] = str
				}
			} else {
				row[j] = ""
			}
		}
		m.rows[i] = row
	}

	m.recordTable.SetColumns(m.columns)
	m.recordTable.SetRows(m.rows)
}

// View table browser
func (m *TableBrowserModel) View() string {
	if m.loading {
		return "Loading..."
	}

	if m.errorMsg != "" {
		return errorStyle.Render("Error: " + m.errorMsg)
	}

	switch m.state {
	case tableBrowserStateTableList:
		return m.tableList.View()

	case tableBrowserStateRecordList:
		// Find the total record count for the current table
		totalRecords := 0
		for _, table := range m.availableTables {
			if table.Name == m.currentTable {
				totalRecords = table.RecordCount
				break
			}
		}
		
		var content string
		if m.filterQuery != "" {
			// Show filtered results with context
			if m.filteredCount > 0 {
				content = fmt.Sprintf("Table: %s (showing %d of %d filtered, %d total)\n\n", 
					m.currentTable, len(m.currentRecords), m.filteredCount, totalRecords)
			} else {
				content = fmt.Sprintf("Table: %s (showing %d filtered results, %d total)\n\n", 
					m.currentTable, len(m.currentRecords), totalRecords)
			}
			content += fmt.Sprintf("Filter: %s\n\n", m.filterQuery)
		} else {
			content = fmt.Sprintf("Table: %s (%d total records, showing %d)\n\n", 
				m.currentTable, totalRecords, len(m.currentRecords))
		}
		
		if len(m.currentRecords) == 0 {
			content += "No records found."
		} else {
			content += m.recordTable.View()
		}
		
		content += "\n\nPress 'f' to filter, 'r' to refresh, 'enter' to view details, 'esc' to go back"
		return content

	case tableBrowserStateFilter:
		return fmt.Sprintf(
			"Filter Records in %s\n\n%s\n\nPress Enter to apply filter, Esc to cancel",
			m.currentTable,
			m.filterInput.View(),
		)

	case tableBrowserStateRecordDetail:
		return m.renderRecordDetail()
	}

	return "Unknown state"
}

// Render record detail view
func (m *TableBrowserModel) renderRecordDetail() string {
	if m.currentRecord == nil {
		return "No record selected"
	}

	var content strings.Builder
	content.WriteString(fmt.Sprintf("Record Detail - %s\n\n", m.currentTable))

	// Display fields in a nice format
	for key, value := range m.currentRecord {
		content.WriteString(fmt.Sprintf("%-25s: %v\n", key, value))
	}

	content.WriteString("\n\nPress 'esc' to go back")
	return content.String()
}

// Table styles
func tableStyles() table.Styles {
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
	return s
}

