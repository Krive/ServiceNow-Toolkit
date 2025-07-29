package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Krive/ServiceNow-Toolkit/pkg/servicenow"
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
	filterQuery   string
	isFiltering   bool
	
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
	if t.RecordCount > 0 {
		return fmt.Sprintf("%s - %d records", t.Desc, t.RecordCount)
	}
	return t.Desc
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

	case errorMsg:
		m.errorMsg = string(msg)
		m.loading = false
		return m, nil
	}

	return m, tea.Batch(cmds...)
}

// Load table list
func (m *TableBrowserModel) loadTableList() tea.Cmd {
	// Don't set loading to true for static table list
	
	// Common ServiceNow tables with descriptions
	tables := []TableInfo{
		{Name: "incident", Label: "Incident", Desc: "Service incidents and disruptions", RecordCount: 0},
		{Name: "sc_request", Label: "Service Request", Desc: "Service catalog requests", RecordCount: 0},
		{Name: "change_request", Label: "Change Request", Desc: "Change management requests", RecordCount: 0},
		{Name: "problem", Label: "Problem", Desc: "Problem management records", RecordCount: 0},
		{Name: "sys_user", Label: "User", Desc: "System users", RecordCount: 0},
		{Name: "sys_user_group", Label: "Group", Desc: "User groups", RecordCount: 0},
		{Name: "sys_user_role", Label: "Role", Desc: "User roles", RecordCount: 0},
		{Name: "cmdb_ci", Label: "Configuration Item", Desc: "CMDB configuration items", RecordCount: 0},
		{Name: "cmdb_ci_computer", Label: "Computer", Desc: "Computer configuration items", RecordCount: 0},
		{Name: "cmdb_ci_server", Label: "Server", Desc: "Server configuration items", RecordCount: 0},
		{Name: "cmdb_ci_service", Label: "Service", Desc: "Service configuration items", RecordCount: 0},
		{Name: "sc_cat_item", Label: "Catalog Item", Desc: "Service catalog items", RecordCount: 0},
		{Name: "kb_knowledge", Label: "Knowledge Article", Desc: "Knowledge base articles", RecordCount: 0},
		{Name: "task", Label: "Task", Desc: "General task records", RecordCount: 0},
		{Name: "sys_audit", Label: "Audit Log", Desc: "System audit trail", RecordCount: 0},
	}
	
	// Directly set the tables instead of using async message
	m.availableTables = tables
	items := make([]list.Item, len(tables))
	for i, table := range tables {
		items[i] = table
	}
	m.tableList.SetItems(items)
	
	return nil
}

// Load records from a table
func (m *TableBrowserModel) loadRecords(tableName, filterQuery string) tea.Cmd {
	m.loading = true
	
	return func() tea.Msg {
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
		content := fmt.Sprintf("Table: %s (%d records)\n\n", m.currentTable, len(m.currentRecords))
		
		if m.filterQuery != "" {
			content += fmt.Sprintf("Filter: %s\n\n", m.filterQuery)
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

