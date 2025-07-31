package main

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/Krive/ServiceNow-Toolkit/pkg/servicenow"
)

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
	xmlSearchQuery   string
	xmlSearchResults []int // line numbers containing matches
	xmlSearchIndex   int   // current match index

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
		pageSize: 20, // Default page size
	}
}

// Init method
func (m simpleModel) Init() tea.Cmd {
	return nil
}