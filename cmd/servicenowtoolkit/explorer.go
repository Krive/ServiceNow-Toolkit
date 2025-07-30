package main

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/Krive/ServiceNow-Toolkit/pkg/servicenow"
)

var (
	demoMode bool
	resolvedInstanceURL string // Store the resolved instance URL for header display
)

// Helper function to get credential from flag or environment variable
func getCredentialLocal(flagValue, envVar string) string {
	if flagValue != "" {
		return flagValue
	}
	return os.Getenv(envVar)
}

func dedent(s string) string {
	lines := strings.Split(s, "\n")
	minIndent := -1
	for _, line := range lines {
		trimmed := strings.TrimLeft(line, " \t")
		if trimmed == "" {
			continue // Skip empty lines for indent calculation
		}
		indent := len(line) - len(trimmed)
		if minIndent == -1 || indent < minIndent {
			minIndent = indent
		}
	}
	if minIndent == -1 {
		return s // No non-empty lines
	}
	var result []string
	for _, line := range lines {
		if len(line) > minIndent {
			result = append(result, line[minIndent:])
		} else {
			result = append(result, line)
		}
	}
	return strings.Join(result, "\n")
}

var explorerCmd = &cobra.Command{
	Use:   "explorer",
	Short: "Launch interactive ServiceNow explorer",
	RunE: func(cmd *cobra.Command, args []string) error {
		var client *servicenow.Client
		var err error

		if demoMode {
			client = nil // Demo mode
			resolvedInstanceURL = "" // Clear for demo mode
		} else {
			// Capture the resolved instance URL before creating client
			resolvedInstanceURL = getCredentialLocal(instanceURL, "SERVICENOW_INSTANCE_URL")
			
			client, err = createClient()
			if err != nil {
				return fmt.Errorf("failed to create ServiceNow client: %w", err)
			}
		}

		model := newSimpleExplorer(client)
		program := tea.NewProgram(model, tea.WithAltScreen())

		_, err = program.Run()
		return err
	},
}

func init() {
	rootCmd.AddCommand(explorerCmd)
	explorerCmd.Flags().BoolVar(&demoMode, "demo", false, "Run in demo mode")
}

// Simple states
type simpleState int

const (
	simpleStateMain simpleState = iota
	simpleStateTableList
	simpleStateTableRecords
	simpleStateRecordDetail
	simpleStateCustomTable
	simpleStateQueryFilter
	simpleStateXMLSearch
)

// Messages for async operations
type recordsLoadedMsg struct {
	records []map[string]interface{}
	total   int
}

type recordsErrorMsg struct {
	err error
}

type recordXMLLoadedMsg struct {
	xml string
}

// Simple model
type simpleModel struct {
	state  simpleState
	client *servicenow.Client
	list   list.Model
	width  int
	height int

	// Navigation
	currentTable string
	records      []map[string]interface{}

	// Pagination
	currentPage  int
	totalPages   int
	pageSize     int
	totalRecords int
	loading      bool

	// Record detail
	selectedRecord  map[string]interface{}
	recordXML       string
	xmlScrollOffset int

	// XML search
	xmlSearchQuery    string
	xmlSearchResults  []int // line numbers containing matches
	xmlSearchIndex    int   // current match index

	// Custom table input
	customTableInput string

	// Query/filter
	currentQuery string

	// Keys
	keys simpleKeyMap
}

type simpleKeyMap struct {
	Up          key.Binding
	Down        key.Binding
	Enter       key.Binding
	Back        key.Binding
	Quit        key.Binding
	NextPage    key.Binding
	PrevPage    key.Binding
	Refresh     key.Binding
	CustomTable key.Binding
	Filter      key.Binding
	ViewXML     key.Binding
	Search      key.Binding
}

func (k simpleKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Enter, k.NextPage, k.PrevPage, k.Back, k.Quit}
}

func (k simpleKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Enter, k.Back},
		{k.NextPage, k.PrevPage, k.Refresh, k.Quit},
	}
}

// Simple menu item
type simpleItem struct {
	title string
	desc  string
	id    string
}

func (i simpleItem) Title() string       { return i.title }
func (i simpleItem) Description() string { return i.desc }
func (i simpleItem) FilterValue() string { return i.title }

// Calculate actual header height based on state
func (m *simpleModel) getHeaderHeight() int {
	if m.state == simpleStateMain {
		return 0 // No header for main menu
	}
	// Header with border and padding typically takes 3-4 lines
	return 4
}

// Calculate actual footer height based on loading state
func (m *simpleModel) getFooterHeight() int {
	baseFooterHeight := 2 // Help text with padding
	if m.loading {
		return baseFooterHeight + 2 // Add loading indicator
	}
	return baseFooterHeight
}

// Calculate available content height
func (m *simpleModel) getAvailableContentHeight() int {
	headerHeight := m.getHeaderHeight()
	footerHeight := m.getFooterHeight()
	availableHeight := m.height - headerHeight - footerHeight
	
	// Ensure minimum usable height
	minHeight := 3
	if availableHeight < minHeight {
		availableHeight = minHeight
	}
	
	return availableHeight
}

// Enforce absolute terminal width boundary - truncates any string to fit
func (m *simpleModel) enforceWidth(text string) string {
	if len(text) <= m.width {
		return text
	}
	
	lines := strings.Split(text, "\n")
	var constrainedLines []string
	
	for _, line := range lines {
		// Count visible characters (ignore ANSI escape sequences)
		visibleLen := m.countVisibleChars(line)
		if visibleLen > m.width {
			// Truncate preserving ANSI sequences when possible
			constrainedLines = append(constrainedLines, m.truncatePreserveAnsi(line, m.width))
		} else {
			constrainedLines = append(constrainedLines, line)
		}
	}
	
	return strings.Join(constrainedLines, "\n")
}

// Count visible characters excluding ANSI escape sequences
func (m *simpleModel) countVisibleChars(s string) int {
	// Simple approach: remove common ANSI sequences
	ansiRegex := `\x1b\[[0-9;]*[a-zA-Z]`
	re := regexp.MustCompile(ansiRegex)
	cleaned := re.ReplaceAllString(s, "")
	return len(cleaned)
}

// Truncate string while trying to preserve ANSI sequences
func (m *simpleModel) truncatePreserveAnsi(s string, maxWidth int) string {
	if len(s) <= maxWidth {
		return s
	}
	
	// Simple truncation for now - more sophisticated ANSI handling could be added
	return s[:maxWidth]
}

// Enforce absolute terminal height boundary - truncates content to fit
func (m *simpleModel) enforceHeight(text string) string {
	lines := strings.Split(text, "\n")
	if len(lines) <= m.height {
		return text
	}
	
	// Hard truncate at terminal height
	return strings.Join(lines[:m.height], "\n")
}

// Update list size based on current state and terminal size
func (m *simpleModel) updateListSize() {
	if m.width > 0 && m.height > 0 {
		// Calculate precise layout dimensions
		headerHeight := 3 // All states now have headers
		footerHeight := 2
		
		contentHeight := m.height - headerHeight - footerHeight
		if contentHeight < 3 {
			contentHeight = 3
		}
		
		// For main menu, account for extra welcome text (5 lines: welcome + status + instruction + 2 spacing)
		listHeight := contentHeight - 2 // Account for padding
		if m.state == simpleStateMain {
			listHeight = contentHeight - 5 // Account for welcome section: 3 text lines + 2 spacing
		}
		if listHeight < 3 {
			listHeight = 3
		}
		
		m.list.SetSize(m.width-2, listHeight)
	}
}

// Create new simple explorer
func newSimpleExplorer(client *servicenow.Client) *simpleModel {
	// Create main menu items
	items := []list.Item{
		simpleItem{title: "📋 Table Browser", desc: "Browse and explore ServiceNow tables with filters and search", id: "tables"},
		simpleItem{title: "👥 Identity Management", desc: "Manage users, roles, and groups (Coming Soon)", id: "identity"},
		simpleItem{title: "🏗️ CMDB Explorer", desc: "Explore configuration items and relationships (Coming Soon)", id: "cmdb"},
		simpleItem{title: "🔍 Global Search", desc: "Search across multiple tables and records (Coming Soon)", id: "search"},
		simpleItem{title: "📊 Analytics", desc: "View reports and data analysis (Coming Soon)", id: "analytics"},
		simpleItem{title: "🛒 Service Catalog", desc: "Browse and request services (Coming Soon)", id: "catalog"},
	}

	if client == nil {
		items = append(items, simpleItem{title: "🎭 Demo Mode Active", desc: "Currently running without ServiceNow connection", id: "demo"})
	}

	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Main Menu"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)

	keys := simpleKeyMap{
		Up:          key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
		Down:        key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
		Enter:       key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select")),
		Back:        key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
		Quit:        key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
		NextPage:    key.NewBinding(key.WithKeys("right", "l", "n"), key.WithHelp("→/l/n", "next page")),
		PrevPage:    key.NewBinding(key.WithKeys("left", "h", "p"), key.WithHelp("←/h/p", "prev page")),
		Refresh:     key.NewBinding(key.WithKeys("r", "f5"), key.WithHelp("r", "refresh")),
		CustomTable: key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "custom table")),
		Filter:      key.NewBinding(key.WithKeys("f"), key.WithHelp("f", "filter")),
		ViewXML:     key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "view XML")),
		Search:      key.NewBinding(key.WithKeys("s", "/"), key.WithHelp("s", "search")),
	}

	return &simpleModel{
		state:    simpleStateMain,
		client:   client,
		list:     l,
		keys:     keys,
		pageSize: 100, // Default page size
	}
}

// Get compact ServiceNow Toolkit logo for headers
func getCompactLogo() string {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("39")).
		Bold(true).
		Render("🚀 ServiceNow Toolkit")
}

// Get instance name from resolved instance URL
func (m *simpleModel) getInstanceName() string {
	if resolvedInstanceURL == "" {
		return ""
	}
	
	// Extract instance name from URL
	// URLs are typically like: https://dev12345.service-now.com or https://companyname.service-now.com
	url := resolvedInstanceURL
	
	// Remove protocol
	if strings.HasPrefix(url, "https://") {
		url = strings.TrimPrefix(url, "https://")
	} else if strings.HasPrefix(url, "http://") {
		url = strings.TrimPrefix(url, "http://")
	}
	
	// Remove path and query parameters
	if idx := strings.Index(url, "/"); idx != -1 {
		url = url[:idx]
	}
	if idx := strings.Index(url, "?"); idx != -1 {
		url = url[:idx]
	}
	
	// Extract the instance name (everything before first dot)
	if idx := strings.Index(url, "."); idx != -1 {
		return url[:idx]
	}
	
	return url
}

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

// Load main menu
func (m *simpleModel) loadMainMenu() {
	items := []list.Item{
		simpleItem{title: "📋 Table Browser", desc: "Browse and explore ServiceNow tables with filters and search", id: "tables"},
		simpleItem{title: "👥 Identity Management", desc: "Manage users, roles, and groups (Coming Soon)", id: "identity"},
		simpleItem{title: "🏗️ CMDB Explorer", desc: "Explore configuration items and relationships (Coming Soon)", id: "cmdb"},
		simpleItem{title: "🔍 Global Search", desc: "Search across multiple tables and records (Coming Soon)", id: "search"},
		simpleItem{title: "📊 Analytics", desc: "View reports and data analysis (Coming Soon)", id: "analytics"},
		simpleItem{title: "🛒 Service Catalog", desc: "Browse and request services (Coming Soon)", id: "catalog"},
	}

	if m.client == nil {
		items = append(items, simpleItem{title: "🎭 Demo Mode Active", desc: "Currently running without ServiceNow connection", id: "demo"})
	}

	m.list.SetItems(items)
	m.list.Title = "Main Menu"
}

// Load table list
func (m *simpleModel) loadTableList() {
	items := []list.Item{
		simpleItem{title: "🔧 Custom Table", desc: "Enter custom table name", id: "custom_table"},
		simpleItem{title: "incident", desc: "Service incidents", id: "incident"},
		simpleItem{title: "problem", desc: "Problem records", id: "problem"},
		simpleItem{title: "change_request", desc: "Change requests", id: "change_request"},
		simpleItem{title: "sys_user", desc: "System users", id: "sys_user"},
		simpleItem{title: "sys_user_group", desc: "User groups", id: "sys_user_group"},
		simpleItem{title: "cmdb_ci_server", desc: "Server CIs", id: "cmdb_ci_server"},
		simpleItem{title: "cmdb_ci_computer", desc: "Computer CIs", id: "cmdb_ci_computer"},
		simpleItem{title: "sc_request", desc: "Service requests", id: "sc_request"},
		simpleItem{title: "kb_knowledge", desc: "Knowledge articles", id: "kb_knowledge"},
		simpleItem{title: "← Back to Main Menu", desc: "Return to main menu", id: "back"},
	}

	m.list.SetItems(items)
	m.list.Title = "ServiceNow Tables"
}

// Load table records command - returns a command instead of directly loading
func (m *simpleModel) loadTableRecordsCmd(tableName string) tea.Cmd {
	return func() tea.Msg {
		if m.client == nil {
			// Demo mode - generate paginated mock data
			return m.loadDemoRecordsSync(tableName)
		} else {
			// Real mode - load from ServiceNow with pagination
			return m.loadRealRecordsSync(tableName)
		}
	}
}

// Load table records with query/filter command
func (m *simpleModel) loadTableRecordsWithQueryCmd(tableName, query string) tea.Cmd {
	return func() tea.Msg {
		if m.client == nil {
			// Demo mode - just return filtered demo data
			return m.loadDemoRecordsSync(tableName)
		} else {
			// Real mode - load from ServiceNow with query filter
			return m.loadRealRecordsWithQuerySync(tableName, query)
		}
	}
}

// Load real records from ServiceNow with query filter
func (m *simpleModel) loadRealRecordsWithQuerySync(tableName, query string) tea.Msg {
	offset := m.currentPage * m.pageSize

	// Build query parameters with user-provided filter
	params := map[string]string{
		"sysparm_limit":         fmt.Sprintf("%d", m.pageSize),
		"sysparm_offset":        fmt.Sprintf("%d", offset),
		"sysparm_fields":        "sys_id,number,name,title,display_name,short_description,description,state,priority,category,assigned_to,assignment_group,caller_id,user_name,email,active,department,sys_updated_on,sys_created_on",
		"sysparm_query":         query, // Use the user's query directly
		"sysparm_display_value": "all", // Get display values for reference fields
	}

	// Real ServiceNow API call with query
	records, err := m.client.Table(tableName).List(params)
	if err != nil {
		return recordsErrorMsg{err: err}
	}

	// Get actual total records using aggregate API with query
	// Proper fallback estimate based on loaded records
	total := offset + len(records)
	if len(records) == m.pageSize {
		// Got a full page, assume there might be more
		total = offset + len(records) + 1 // At least one more record exists
	}
	
	// Try to get exact count using aggregate API with the filter (only on first page)
	if m.currentPage == 0 {
		if aggClient := m.client.Aggregate(tableName); aggClient != nil {
		// Create a basic query builder with the raw query
		// This is a simplified approach - in production you'd want proper parsing
		aq := aggClient.NewQuery().CountAll("filtered_count")
		
		// Apply the query filter if it's a simple condition
		if strings.Contains(query, "=") && !strings.Contains(query, "ORDERBY") {
			// Extract the query part before any ORDERBY clause
			queryPart := query
			if idx := strings.Index(query, "ORDERBY"); idx > 0 {
				queryPart = strings.TrimSpace(query[:idx])
				// Remove trailing ^ if present
				queryPart = strings.TrimSuffix(queryPart, "^")
			}
			
			// For simple field=value queries, apply them
			if queryPart != "" && queryPart != "ORDERBYDESCsys_updated_on" {
				parts := strings.SplitN(queryPart, "=", 2)
				if len(parts) == 2 {
					field := strings.TrimSpace(parts[0])
					value := strings.TrimSpace(parts[1])
					aq.Equals(field, value)
				}
			}
		}
		
		if result, err := aq.Execute(); err == nil && result.Stats != nil {
			if countVal, ok := result.Stats["filtered_count"]; ok {
				if actualTotal := parseIntFromInterface(countVal); actualTotal > 0 {
					total = actualTotal
				}
			}
		}
		// If aggregate fails, keep the estimated total as fallback
		}
	} else {
		// On subsequent pages, use previously calculated total if it seems reasonable
		if m.totalRecords > offset+len(records) {
			total = m.totalRecords
		}
	}

	return recordsLoadedMsg{
		records: records,
		total:   total,
	}
}

// Load demo records synchronously and return message
func (m *simpleModel) loadDemoRecordsSync(tableName string) tea.Msg {
	// Simulate total records (varies by table)
	totalRecords := map[string]int{
		"incident":         347,
		"problem":          89,
		"change_request":   156,
		"sys_user":         892,
		"sys_user_group":   45,
		"cmdb_ci_server":   234,
		"cmdb_ci_computer": 567,
		"sc_request":       123,
		"kb_knowledge":     78,
	}

	total := totalRecords[tableName]
	if total == 0 {
		total = 50 // Default for unknown tables
	}

	// Generate current page records
	var records []map[string]interface{}
	startRecord := m.currentPage*m.pageSize + 1
	endRecord := startRecord + m.pageSize - 1
	if endRecord > total {
		endRecord = total
	}

	for i := startRecord; i <= endRecord; i++ {
		record := m.generateDemoRecord(tableName, i)
		records = append(records, record)
	}

	return recordsLoadedMsg{
		records: records,
		total:   total,
	}
}


// Load real records from ServiceNow synchronously
func (m *simpleModel) loadRealRecordsSync(tableName string) tea.Msg {
	// Calculate offset for current page
	offset := m.currentPage * m.pageSize

	// Build query parameters for ServiceNow API with sorting
	params := map[string]string{
		"sysparm_limit":         fmt.Sprintf("%d", m.pageSize),
		"sysparm_offset":        fmt.Sprintf("%d", offset),
		"sysparm_fields":        "sys_id,number,name,title,display_name,short_description,description,state,priority,category,assigned_to,assignment_group,caller_id,user_name,email,active,department,sys_updated_on,sys_created_on",
		"sysparm_query":         "ORDERBYDESCsys_updated_on", // Sort by sys_updated_on descending
		"sysparm_display_value": "all",                       // Get display values for reference fields
	}

	// Real ServiceNow API call
	records, err := m.client.Table(tableName).List(params)
	if err != nil {
		return recordsErrorMsg{err: err}
	}

	// Get actual total records using aggregate API
	// Proper fallback estimate based on loaded records
	total := offset + len(records)
	if len(records) == m.pageSize {
		// Got a full page, assume there might be more
		total = offset + len(records) + 1 // At least one more record exists
	}
	
	// Try to get exact count using aggregate API (only on first page to avoid repeated calls)
	if m.currentPage == 0 {
		if aggClient := m.client.Aggregate(tableName); aggClient != nil {
			if actualTotal, err := aggClient.CountRecords(nil); err == nil && actualTotal > 0 {
				total = actualTotal
			}
			// If aggregate fails, keep the estimated total as fallback
		}
	} else {
		// On subsequent pages, use previously calculated total if it seems reasonable
		if m.totalRecords > offset+len(records) {
			total = m.totalRecords
		}
	}

	return recordsLoadedMsg{
		records: records,
		total:   total,
	}
}

// Load record XML command
func (m *simpleModel) loadRecordXMLCmd(recordID string) tea.Cmd {
	return func() tea.Msg {
		if m.client == nil {
			// Demo mode - simple XML generation
			xml := generateXMLFromRecord(m.selectedRecord, m.currentTable, recordID)
			return recordXMLLoadedMsg{xml: xml}
		}

		// Real mode - fetch complete record for XML representation
		return m.loadCompleteRecordXML(recordID)
	}
}

// Load complete record from ServiceNow for XML display
func (m *simpleModel) loadCompleteRecordXML(recordID string) tea.Msg {
	// Fetch the complete record with all fields
	params := map[string]string{
		"sysparm_query":         fmt.Sprintf("sys_id=%s", recordID),
		"sysparm_limit":         "1",
		"sysparm_display_value": "all", // Get display values for reference fields
		// Don't limit fields - get all fields for complete XML
	}

	records, err := m.client.Table(m.currentTable).List(params)
	if err != nil {
		return recordsErrorMsg{err: fmt.Errorf("failed to load complete record: %w", err)}
	}

	if len(records) == 0 {
		return recordsErrorMsg{err: fmt.Errorf("record not found: %s", recordID)}
	}

	// Generate XML from the complete record
	xml := generateXMLFromRecord(records[0], m.currentTable, recordID)
	return recordXMLLoadedMsg{xml: xml}
}

// Process loaded records and update UI
func (m *simpleModel) processLoadedRecords(records []map[string]interface{}, total int) {
	m.totalRecords = total
	m.totalPages = (total + m.pageSize - 1) / m.pageSize

	// Store records for later access (for XML view)
	m.records = records

	// Convert records to list items
	var items []list.Item
	for i, record := range records {
		title, desc := m.formatRecordDisplay(record, i)

		items = append(items, simpleItem{
			title: title,
			desc:  desc,
			id:    getRecordField(record, "sys_id"),
		})
	}

	// Add back navigation item
	items = append(items, simpleItem{
		title: "← Back to Table List",
		desc:  "Return to table selection",
		id:    "back",
	})

	m.list.SetItems(items)
	m.updateRecordTitle(m.currentTable)
}

// Format record display based on table type and available fields
func (m *simpleModel) formatRecordDisplay(record map[string]interface{}, index int) (string, string) {
	// Try common display fields in order of preference
	displayFields := []string{
		"number", "name", "title", "display_name", "sys_name",
		"short_description", "description", "user_name", "email",
	}

	var title string
	for _, field := range displayFields {
		if val := getRecordField(record, field); val != "" {
			title = val
			break
		}
	}

	// Fallback to sys_id if no display field found
	if title == "" {
		if sysId := getRecordField(record, "sys_id"); sysId != "" {
			title = fmt.Sprintf("Record %s", sysId[:8]) // Show first 8 chars of sys_id
		} else {
			title = fmt.Sprintf("Record #%d", m.currentPage*m.pageSize+index+1)
		}
	}

	// Build description from multiple fields
	var descParts []string

	// Add short description if not already used as title
	if shortDesc := getRecordField(record, "short_description"); shortDesc != "" && shortDesc != title {
		descParts = append(descParts, shortDesc)
	}

	// Add state information
	if state := getRecordField(record, "state"); state != "" {
		descParts = append(descParts, fmt.Sprintf("State: %s", state))
	}

	// Add priority if available
	if priority := getRecordField(record, "priority"); priority != "" {
		descParts = append(descParts, fmt.Sprintf("Priority: %s", priority))
	}

	// Add category if available
	if category := getRecordField(record, "category"); category != "" {
		descParts = append(descParts, fmt.Sprintf("Category: %s", category))
	}

	// Add assigned to if available
	if assignedTo := getRecordField(record, "assigned_to"); assignedTo != "" {
		descParts = append(descParts, fmt.Sprintf("Assigned: %s", assignedTo))
	}

	// Add updated timestamp
	if updatedOn := getRecordField(record, "sys_updated_on"); updatedOn != "" {
		descParts = append(descParts, fmt.Sprintf("Updated: %s", updatedOn))
	}

	desc := strings.Join(descParts, " | ")
	if desc == "" {
		desc = fmt.Sprintf("Table: %s", m.currentTable)
	}

	return title, desc
}

// Generate demo record based on table type
func (m *simpleModel) generateDemoRecord(tableName string, recordNum int) map[string]interface{} {
	baseRecord := map[string]interface{}{
		"sys_id":         fmt.Sprintf("%s-sys-id-%05d", tableName, recordNum),
		"sys_updated_on": "2024-01-15 12:00:00",
		"sys_created_on": "2024-01-01 10:00:00",
		"sys_created_by": "admin",
		"sys_updated_by": "system",
	}

	switch tableName {
	case "incident":
		baseRecord["number"] = fmt.Sprintf("INC%07d", recordNum)
		baseRecord["short_description"] = fmt.Sprintf("Server connectivity issue #%d", recordNum)
		baseRecord["state"] = fmt.Sprintf("%d", (recordNum%6)+1) // 1-6 states
		baseRecord["priority"] = fmt.Sprintf("%d", (recordNum%4)+1)
		baseRecord["category"] = "Network"
		baseRecord["assignment_group"] = "IT Support Team"
		baseRecord["assignment_group_sys_id"] = fmt.Sprintf("grp-sys-id-%05d", recordNum%10)
		baseRecord["caller_id"] = fmt.Sprintf("Demo User %d", recordNum%100)
		baseRecord["caller_id_sys_id"] = fmt.Sprintf("usr-sys-id-%05d", recordNum%100)
		baseRecord["assigned_to"] = fmt.Sprintf("John Smith %d", recordNum%50)
		baseRecord["assigned_to_sys_id"] = fmt.Sprintf("usr-sys-id-%05d", recordNum%50)

	case "problem":
		baseRecord["number"] = fmt.Sprintf("PRB%07d", recordNum)
		baseRecord["short_description"] = fmt.Sprintf("Root cause analysis for recurring issue #%d", recordNum)
		baseRecord["state"] = fmt.Sprintf("%d", (recordNum%4)+1)
		baseRecord["priority"] = fmt.Sprintf("%d", (recordNum%3)+2)
		baseRecord["category"] = "Infrastructure"

	case "sys_user":
		baseRecord["user_name"] = fmt.Sprintf("user.demo.%d", recordNum)
		baseRecord["name"] = fmt.Sprintf("Demo User %d", recordNum)
		baseRecord["email"] = fmt.Sprintf("user%d@demo.com", recordNum)
		baseRecord["active"] = "true"
		baseRecord["department"] = []string{"IT", "Finance", "HR", "Operations"}[recordNum%4]

	case "change_request":
		baseRecord["number"] = fmt.Sprintf("CHG%07d", recordNum)
		baseRecord["short_description"] = fmt.Sprintf("System maintenance change #%d", recordNum)
		baseRecord["state"] = fmt.Sprintf("%d", (recordNum%7)+1)
		baseRecord["priority"] = fmt.Sprintf("%d", (recordNum%4)+1)
		baseRecord["category"] = "Normal"
		baseRecord["type"] = "Standard"

	default:
		// Generic record
		prefix := "REC"
		if len(tableName) >= 3 {
			prefix = strings.ToUpper(tableName[:3])
		}
		baseRecord["number"] = fmt.Sprintf("%s%05d", prefix, recordNum)
		baseRecord["name"] = fmt.Sprintf("Demo %s record %d", tableName, recordNum)
		baseRecord["short_description"] = fmt.Sprintf("Demo %s record #%d", tableName, recordNum)
		baseRecord["state"] = fmt.Sprintf("%d", (recordNum%4)+1)
	}

	return baseRecord
}

// Helper function to safely extract field from ServiceNow record
func getRecordField(record map[string]interface{}, field string) string {
	if val, ok := record[field]; ok && val != nil {
		// Handle ServiceNow reference field objects
		if refMap, isMap := val.(map[string]interface{}); isMap {
			// This is a reference field with link and value
			// For table display, prefer display_value over value (sys_id)
			if dispVal, hasDisplay := refMap["display_value"]; hasDisplay && dispVal != nil {
				return fmt.Sprintf("%v", dispVal)
			}
			if value, hasValue := refMap["value"]; hasValue && value != nil {
				return fmt.Sprintf("%v", value)
			}
		}
		return fmt.Sprintf("%v", val)
	}
	return ""
}

// Helper function to extract display value from ServiceNow reference field
func getRecordDisplayValue(record map[string]interface{}, field string) (value string, displayValue string, isReference bool) {
	if val, ok := record[field]; ok && val != nil {
		// Handle ServiceNow reference field objects
		if refMap, isMap := val.(map[string]interface{}); isMap {
			// This is a reference field with link and value
			if refValue, hasValue := refMap["value"]; hasValue && refValue != nil {
				valueStr := fmt.Sprintf("%v", refValue)

				// Check if there's a display_value field
				if dispVal, hasDisplay := refMap["display_value"]; hasDisplay && dispVal != nil {
					return valueStr, fmt.Sprintf("%v", dispVal), true
				}

				// For now, return the sys_id as both value and display
				return valueStr, valueStr, true
			}
		}
		// Regular field
		valueStr := fmt.Sprintf("%v", val)
		return valueStr, valueStr, false
	}
	return "", "", false
}

// Generate XML representation from ServiceNow record
func generateXMLFromRecord(record map[string]interface{}, tableName, recordID string) string {
	var xml strings.Builder
	xml.WriteString(fmt.Sprintf("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n"))
	xml.WriteString(fmt.Sprintf("<record table=\"%s\" sys_id=\"%s\">\n", tableName, recordID))

	// Iterate through all fields in the record
	for key, value := range record {
		if value != nil {
			sysIdValue, displayValue, isReference := getRecordDisplayValue(record, key)

			if isReference && displayValue != sysIdValue {
				// Reference field with display value
				xml.WriteString(fmt.Sprintf("    <%s display_value=\"%s\">%s</%s>\n",
					key, escapeXML(displayValue), escapeXML(sysIdValue), key))
			} else {
				// Regular field or reference without display value
				xml.WriteString(fmt.Sprintf("    <%s>%s</%s>\n", key, escapeXML(sysIdValue), key))
			}
		} else {
			xml.WriteString(fmt.Sprintf("    <%s/>\n", key))
		}
	}

	xml.WriteString("</record>")
	return xml.String()
}

// Helper function to escape XML special characters
func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}

// Update record title with pagination info
func (m *simpleModel) updateRecordTitle(tableName string) {
	pageInfo := ""
	if m.totalPages > 1 {
		pageInfo = fmt.Sprintf(" (Page %d/%d)", m.currentPage+1, m.totalPages)
	}
	
	// Add total records count if available
	totalInfo := ""
	if m.totalRecords > 0 {
		totalInfo = fmt.Sprintf(" - %d total", m.totalRecords)
	}
	
	m.list.Title = fmt.Sprintf("Records: %s%s%s", tableName, totalInfo, pageInfo)
}

// View method
func (m simpleModel) View() string {
	// Define minimum terminal requirements
	const minWidth = 80
	const minHeight = 20
	
	// Check if terminal is too small
	if m.height < minHeight || m.width < minWidth {
		message := fmt.Sprintf("Terminal too small!\n\nCurrent: %dx%d\nRequired: %dx%d\n\nPlease resize your terminal window.", 
			m.width, m.height, minWidth, minHeight)
		
		// Center the message in available space
		style := lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true).
			Align(lipgloss.Center).
			Width(m.width).
			Height(m.height)
		
		return style.Render(message)
	}

	// Calculate layout areas with safe dimensions
	headerHeight := 0
	footerHeight := 2
	
	var headerContent string
	
	// Build header if needed
	instanceName := m.getInstanceName()
	instanceSuffix := ""
	if instanceName != "" {
		instanceSuffix = fmt.Sprintf(" [%s]", instanceName)
	}
	
	switch m.state {
	case simpleStateMain:
		headerContent = getCompactLogo() + " - ServiceNow TUI Explorer" + instanceSuffix
		headerHeight = 3
	case simpleStateTableList:
		headerContent = getCompactLogo() + " - 📋 Table Browser" + instanceSuffix
		headerHeight = 3
	case simpleStateTableRecords:
		headerContent = fmt.Sprintf("%s - 📋 Table: %s%s", getCompactLogo(), m.currentTable, instanceSuffix)
		headerHeight = 3
	case simpleStateRecordDetail:
		headerContent = fmt.Sprintf("%s - 📄 Record XML: %s%s", getCompactLogo(), m.currentTable, instanceSuffix)
		headerHeight = 3
	case simpleStateCustomTable:
		headerContent = getCompactLogo() + " - 🔧 Custom Table Input" + instanceSuffix
		headerHeight = 3
	case simpleStateQueryFilter:
		headerContent = fmt.Sprintf("%s - 🔍 Filter: %s%s", getCompactLogo(), m.currentTable, instanceSuffix)
		headerHeight = 3
	case simpleStateXMLSearch:
		headerContent = fmt.Sprintf("%s - 🔍 Search XML: %s%s", getCompactLogo(), m.currentTable, instanceSuffix)
		headerHeight = 3
	}

	// Calculate content dimensions with absolute terminal constraints
	contentHeight := m.height - headerHeight - footerHeight
	if contentHeight < 3 {
		contentHeight = 3
	}

	// Build header section with absolute width constraint
	var header string
	if headerHeight > 0 {
		// Truncate header content to fit terminal width (account for border chars)
		maxHeaderWidth := m.width - 4
		if maxHeaderWidth < 1 {
			maxHeaderWidth = 1
		}
		if len(headerContent) > maxHeaderWidth {
			headerContent = headerContent[:maxHeaderWidth-3] + "..."
		}
		
		// Use width that accounts for border characters (borders add ~2 chars)
		headerWidth := m.width - 2
		if headerWidth < 10 {
			headerWidth = 10
		}
		
		headerStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("86")).
			Bold(true).
			Width(headerWidth).
			Align(lipgloss.Center).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("86")).
			Height(1)
		
		header = headerStyle.Render(headerContent)
	}

	// Build content section with safe constraints
	var content string
	switch m.state {
	case simpleStateRecordDetail:
		if m.recordXML != "" {
			content = m.renderScrollableXML()
		} else if m.loading {
			content = "Loading record XML..."
		} else {
			content = "No XML data available. Press Esc to go back."
		}
	case simpleStateCustomTable:
		content = m.renderCustomTableInput()
	case simpleStateQueryFilter:
		content = m.renderQueryFilter()
	case simpleStateXMLSearch:
		content = m.renderXMLSearch()
	case simpleStateMain:
		// Main menu with better spacing and organization
		var connectionStatus string
		if m.client == nil {
			connectionStatus = "🎭 Demo Mode - No ServiceNow connection"
		} else {
			connectionStatus = "🔗 Connected to ServiceNow instance"
		}

		// Create a welcome section with better styling
		welcomeStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("86")).
			Bold(true).
			Width(m.width).
			Align(lipgloss.Center)

		statusStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")).
			Italic(true).
			Width(m.width).
			Align(lipgloss.Center)

		instructionStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")).
			Italic(true).
			Width(m.width).
			Align(lipgloss.Center)

		welcomeMsg := welcomeStyle.Render("Welcome to ServiceNow Toolkit!")
		statusMsg := statusStyle.Render(connectionStatus)
		instructionMsg := instructionStyle.Render("Select an option below to get started:")

		// Combine with clean spacing (no extra margins)
		topSection := welcomeMsg + "\n" + statusMsg + "\n" + instructionMsg + "\n\n"
		mainContent := topSection + m.list.View()
		content = mainContent
	default:
		content = m.list.View()
	}

	// Enforce height constraint on content
	contentLines := strings.Split(content, "\n")
	if len(contentLines) > contentHeight {
		content = strings.Join(contentLines[:contentHeight], "\n")
	}

	// Build footer section with safe width
	var helpText string
	switch m.state {
	case simpleStateTableRecords:
		if m.totalPages > 1 {
			helpText = "↑/↓: navigate • enter: XML • ←/→: prev/next page • f: filter • t: custom table • x: view XML • r: refresh • esc: back • q: quit"
		} else {
			helpText = "↑/↓: navigate • enter: XML • f: filter • t: custom table • x: view XML • r: refresh • esc: back • q: quit"
		}
	case simpleStateTableList:
		helpText = "↑/↓: navigate • enter: select • t: custom table • esc: back • q: quit"
	case simpleStateRecordDetail:
		if len(m.xmlSearchResults) > 0 {
			helpText = fmt.Sprintf("↑/↓: scroll • s: search • n/N: next/prev match (%d/%d) • esc: back • q: quit", m.xmlSearchIndex+1, len(m.xmlSearchResults))
		} else {
			helpText = "↑/↓: scroll • s: search • esc: back • q: quit"
		}
	case simpleStateCustomTable:
		helpText = "Type table name • enter: load table • esc: back • q: quit"
	case simpleStateQueryFilter:
		helpText = "Type ServiceNow query • enter: apply filter • esc: back • q: quit"
	case simpleStateXMLSearch:
		helpText = "Type search term • enter: search • esc: cancel • q: quit"
	default:
		helpText = "↑/↓: navigate • enter: select • esc: back • q: quit"
	}

	// Truncate help text to fit terminal width
	maxFooterWidth := m.width - 2
	if maxFooterWidth < 1 {
		maxFooterWidth = 1
	}
	if len(helpText) > maxFooterWidth {
		helpText = helpText[:maxFooterWidth-3] + "..."
	}

	footerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("244")).
		Width(m.width).
		Height(footerHeight).
		Padding(0, 1)

	footer := footerStyle.Render(helpText)

	// Add loading indicator if loading (with width constraint)
	if m.loading {
		loadingText := "🔄 Loading..."
		if len(loadingText) > m.width {
			loadingText = "Loading..."
		}
		loadingStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("86")).
			Bold(true).
			Width(m.width)
		footer = footer + "\n" + loadingStyle.Render(loadingText)
	}

	// Combine sections with absolute terminal constraints
	var sections []string
	if headerHeight > 0 {
		sections = append(sections, header)
	}
	sections = append(sections, content, footer)

	finalView := lipgloss.JoinVertical(lipgloss.Left, sections...)
	
	// Final height enforcement only (width should be properly calculated now)
	finalView = m.enforceHeight(finalView)
	
	return finalView
}

// Render custom table input
func (m simpleModel) renderCustomTableInput() string {
	var content strings.Builder
	content.WriteString("Enter a ServiceNow table name to browse:\n\n")

	// Calculate input box width accounting for borders
	inputWidth := m.width - 8 // Conservative border + padding accounting
	if inputWidth < 20 {
		inputWidth = 20
	}
	if inputWidth > 50 {
		inputWidth = 50
	}

	inputBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(1, 2).
		Width(inputWidth).
		Render(m.customTableInput + "_")

	content.WriteString(inputBox)
	content.WriteString("\n\nExamples: incident, problem, change_request, sys_user, cmdb_ci_server")
	content.WriteString("\n\nPress Enter to load the table or Esc to go back.")

	return content.String()
}

// Render query filter
func (m simpleModel) renderQueryFilter() string {
	var content strings.Builder
	content.WriteString(fmt.Sprintf("Filter records in table: %s\n\n", m.currentTable))

	// Calculate input box width accounting for borders
	inputWidth := m.width - 8 // Conservative border + padding accounting
	if inputWidth < 30 {
		inputWidth = 30
	}
	if inputWidth > 60 {
		inputWidth = 60
	}

	inputBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(1, 2).
		Width(inputWidth).
		Render(m.currentQuery + "_")

	content.WriteString(inputBox)
	content.WriteString("\n\nServiceNow Query Examples:")
	content.WriteString("\n• state=1^priority=1")
	content.WriteString("\n• short_descriptionCONTAINSserver")
	content.WriteString("\n• sys_created_on>2024-01-01")
	content.WriteString("\n• active=true^ORDERBYDESCsys_updated_on")
	content.WriteString("\n\nPress Enter to apply filter or Esc to go back.")

	return content.String()
}

// Render XML search
func (m simpleModel) renderXMLSearch() string {
	var content strings.Builder
	content.WriteString(fmt.Sprintf("Search in XML content of record from: %s\n\n", m.currentTable))

	// Calculate input box width accounting for borders
	inputWidth := m.width - 8 // Conservative border + padding accounting
	if inputWidth < 30 {
		inputWidth = 30
	}
	if inputWidth > 60 {
		inputWidth = 60
	}

	inputBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(1, 2).
		Width(inputWidth).
		Render(m.xmlSearchQuery + "_")

	content.WriteString(inputBox)
	
	if len(m.xmlSearchResults) > 0 {
		content.WriteString(fmt.Sprintf("\n\n🔍 Found %d matches", len(m.xmlSearchResults)))
		content.WriteString("\nPress Enter to finish searching and navigate with n/N")
	} else if m.xmlSearchQuery != "" {
		content.WriteString("\n\n❌ No matches found")
	}
	
	content.WriteString("\n\nSearch Examples:")
	content.WriteString("\n• sys_id")
	content.WriteString("\n• state")
	content.WriteString("\n• 2024-01-01")
	content.WriteString("\n• priority")
	content.WriteString("\n\nPress Enter to search or Esc to cancel.")

	return content.String()
}

// Render scrollable XML with navigation
func (m simpleModel) renderScrollableXML() string {
	if m.recordXML == "" {
		return "No XML data"
	}

	lines := strings.Split(m.recordXML, "\n")
	
	// Calculate safe content area for XML
	headerHeight := 3 // XML view always has header
	footerHeight := 2
	contentHeight := m.height - headerHeight - footerHeight
	if contentHeight < 3 {
		contentHeight = 3
	}
	
	// Calculate width accounting for borders (borders need ~4 chars total)
	xmlWidth := m.width - 6 // Conservative border accounting
	if xmlWidth < 20 {
		xmlWidth = 20
	}
	
	xmlHeight := contentHeight - 4 // Leave room for borders and scroll info
	if xmlHeight < 1 {
		xmlHeight = 1
	}

	// Calculate visible lines
	startLine := m.xmlScrollOffset
	endLine := startLine + xmlHeight
	if endLine > len(lines) {
		endLine = len(lines)
	}

	// Get visible portion and truncate long lines
	var visibleLines []string
	for i := startLine; i < endLine && i < len(lines); i++ {
		line := lines[i]
		
		// Highlight search matches if we have search results
		if m.xmlSearchQuery != "" && len(m.xmlSearchResults) > 0 {
			// Check if this line contains a match
			isMatch := false
			for _, matchLine := range m.xmlSearchResults {
				if matchLine == i {
					isMatch = true
					break
				}
			}
			
			if isMatch {
				// Highlight the search term in this line
				queryLower := strings.ToLower(m.xmlSearchQuery)
				lineLower := strings.ToLower(line)
				
				if strings.Contains(lineLower, queryLower) {
					// Simple highlighting - this could be improved
					highlightStyle := lipgloss.NewStyle().Background(lipgloss.Color("11")).Foreground(lipgloss.Color("0"))
					line = strings.ReplaceAll(line, m.xmlSearchQuery, highlightStyle.Render(m.xmlSearchQuery))
					// Also handle case variations
					if m.xmlSearchQuery != strings.ToLower(m.xmlSearchQuery) {
						line = strings.ReplaceAll(line, strings.ToLower(m.xmlSearchQuery), highlightStyle.Render(strings.ToLower(m.xmlSearchQuery)))
					}
					if m.xmlSearchQuery != strings.ToUpper(m.xmlSearchQuery) {
						line = strings.ReplaceAll(line, strings.ToUpper(m.xmlSearchQuery), highlightStyle.Render(strings.ToUpper(m.xmlSearchQuery)))
					}
				}
				
				// Mark current search result with an indicator
				if len(m.xmlSearchResults) > 0 && m.xmlSearchIndex < len(m.xmlSearchResults) && m.xmlSearchResults[m.xmlSearchIndex] == i {
					line = "► " + line
				}
			}
		}
		
		// Truncate lines that are too long for the XML width
		if len(line) > xmlWidth-2 { // Account for padding
			line = line[:xmlWidth-5] + "..."
		}
		visibleLines = append(visibleLines, line)
	}
	visibleXML := strings.Join(visibleLines, "\n")

	// Add scroll indicators
	var scrollInfo string
	if len(lines) > xmlHeight {
		scrollInfo = fmt.Sprintf("Lines %d-%d of %d", startLine+1, endLine, len(lines))
		// Truncate scroll info if too long
		if len(scrollInfo) > xmlWidth {
			scrollInfo = fmt.Sprintf("%d-%d/%d", startLine+1, endLine, len(lines))
		}
	}

	// Style the XML content with absolute terminal constraints
	xmlStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(1).
		Width(xmlWidth).
		Height(xmlHeight)

	content := xmlStyle.Render(visibleXML)

	if scrollInfo != "" {
		infoStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")).
			Italic(true).
			Width(xmlWidth).
			Align(lipgloss.Center)
		scrollInfoRendered := infoStyle.Render(scrollInfo)
		content += "\n" + scrollInfoRendered
	}

	return content
}

// Helper function to parse interface{} to int (same as in table_browser.go)
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

// Init method
func (m simpleModel) Init() tea.Cmd {
	return nil
}
