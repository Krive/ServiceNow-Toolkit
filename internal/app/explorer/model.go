package explorer

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/Krive/ServiceNow-Toolkit/pkg/servicenow"
	"github.com/Krive/ServiceNow-Toolkit/internal/tui"
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
	simpleStateAdvancedFilter
	simpleStateFilterBrowser
	simpleStateQuitConfirm
	simpleStateColumnCustomizer
	simpleStateViewManager
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

// Model represents the main explorer TUI model
type Model struct {
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
	filterInput  string // Temporary input for filter editing

	// Quit confirmation
	quitConfirmSelection int        // 0 = No, 1 = Yes
	previousState        simpleState // State before quit confirmation

	// Column customization
	selectedColumns       []string // Field names of selected columns
	columnCustomizer      *ColumnCustomizer
	viewConfigurations    map[string]*ViewConfiguration // Saved view configurations (in-memory)
	viewManagerSelection  int                             // Selected index in view manager
	configManager         *ConfigManager                  // Persistent configuration manager

	// Advanced filtering components (lazy-loaded)
	fieldMetadataService *tui.FieldMetadataService
	queryBuilder         *tui.QueryBuilderModel
	filterBrowser        *tui.FilterBrowserModel
	filterManager        *tui.FilterManager
	tableMetadata        *tui.TableFieldMetadata

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
	Filter            key.Binding
	ViewXML           key.Binding
	Search            key.Binding
	ColumnCustomizer  key.Binding
	SaveView          key.Binding
	ViewManager       key.Binding
	ResetColumns      key.Binding
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

// New creates a new explorer model
func New(client *servicenow.Client) *Model {
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
	l.SetShowHelp(false) // We use custom footers instead

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
		Filter:           key.NewBinding(key.WithKeys("f"), key.WithHelp("f", "filter")),
		ViewXML:          key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "view XML")),
		Search:           key.NewBinding(key.WithKeys("s", "/"), key.WithHelp("s", "search")),
		ColumnCustomizer: key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "columns")),
		SaveView:         key.NewBinding(key.WithKeys("ctrl+s"), key.WithHelp("ctrl+s", "save view")),
		ViewManager:      key.NewBinding(key.WithKeys("v"), key.WithHelp("v", "views")),
		ResetColumns:     key.NewBinding(key.WithKeys("ctrl+r"), key.WithHelp("ctrl+r", "reset columns")),
	}

	// Initialize configuration manager and load saved settings
	configManager := NewConfigManager()
	configManager.LoadConfig() // Ignore errors for now - will use defaults
	
	return &Model{
		state:              simpleStateMain,
		client:             client,
		list:               l,
		keys:               keys,
		pageSize:           20, // Default page size
		selectedColumns:    []string{"sys_id"}, // Default to showing just sys_id
		viewConfigurations: configManager.GetViewConfigurations(),
		configManager:      configManager,
	}
}

// Init method
func (m Model) Init() tea.Cmd {
	return nil
}