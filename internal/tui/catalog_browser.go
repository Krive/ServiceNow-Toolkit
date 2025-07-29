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

// Catalog Browser states
type catalogBrowserState int

const (
	catalogStateMenu catalogBrowserState = iota
	catalogStateCatalogList
	catalogStateItemList
	catalogStateItemDetail
	catalogStateCart
	catalogStateSearch
)

// Catalog Browser Model
type CatalogBrowserModel struct {
	state           catalogBrowserState
	client          *servicenow.Client
	
	// UI Components
	list            list.Model
	table           table.Model
	textInput       textinput.Model
	
	// Data
	catalogs        []CatalogInfo
	items           []CatalogItem
	selectedCatalog string
	selectedItem    *CatalogItem
	cart           []CartItem
	searchResults   []CatalogItem
	
	// Navigation
	breadcrumb      []string
	
	// Dimensions
	width, height   int
	
	// Loading
	loading         bool
	errorMsg        string
}

// Catalog Info represents a service catalog
type CatalogInfo struct {
	SysID       string
	Title       string
	Description string
	Active      bool
	ItemCount   int
}

// Catalog Item represents a catalog item
type CatalogItem struct {
	SysID         string
	Name          string
	ShortDesc     string
	Description   string
	Price         string
	Category      string
	Available     bool
	Icon          string
	Variables     []CatalogVariable
}

// Catalog Variable represents an item variable
type CatalogVariable struct {
	Name         string
	Label        string
	Type         string
	Mandatory    bool
	DefaultValue string
	Options      []string
}

// Cart Item represents an item in the shopping cart
type CartItem struct {
	Item      CatalogItem
	Quantity  int
	Variables map[string]string
}

// Catalog Menu Items
type catalogMenuItem struct {
	title       string
	description string
	action      string
}

func (m catalogMenuItem) Title() string       { return m.title }
func (m catalogMenuItem) Description() string { return m.description }
func (m catalogMenuItem) FilterValue() string { return m.title }

// NewCatalogBrowser creates a new catalog browser
func NewCatalogBrowser(client *servicenow.Client) *CatalogBrowserModel {
	// Initialize menu items
	items := []list.Item{
		catalogMenuItem{
			title:       "🛒 Browse Catalogs",
			description: "View available service catalogs",
			action:      "catalogs",
		},
		catalogMenuItem{
			title:       "💻 IT Services",
			description: "Request IT equipment and services",
			action:      "it_services",
		},
		catalogMenuItem{
			title:       "🏢 Facilities",
			description: "Office and facility requests",
			action:      "facilities",
		},
		catalogMenuItem{
			title:       "📱 Software",
			description: "Software licenses and applications",
			action:      "software",
		},
		catalogMenuItem{
			title:       "🔍 Search Items",
			description: "Search all catalog items",
			action:      "search",
		},
		catalogMenuItem{
			title:       "🛍️  Shopping Cart",
			description: "View your current cart",
			action:      "cart",
		},
		catalogMenuItem{
			title:       "📋 My Requests",
			description: "View your submitted requests",
			action:      "requests",
		},
	}

	// Create list component
	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Service Catalog"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)

	// Initialize text input
	ti := textinput.New()
	ti.Placeholder = "Search catalog items..."
	ti.Focus()

	// Initialize table
	columns := []table.Column{
		{Title: "Name", Width: 30},
		{Title: "Description", Width: 40},
		{Title: "Category", Width: 20},
		{Title: "Price", Width: 10},
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

	return &CatalogBrowserModel{
		state:     catalogStateMenu,
		client:    client,
		list:      l,
		table:     t,
		textInput: ti,
		breadcrumb: []string{"Service Catalog"},
		cart:      []CartItem{},
	}
}

// Init initializes the catalog browser
func (m *CatalogBrowserModel) Init() tea.Cmd {
	// Initialize with static menu items - no loading needed
	return nil
}

// Update handles messages
func (m *CatalogBrowserModel) Update(msg tea.Msg) (*CatalogBrowserModel, tea.Cmd) {
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
		case catalogStateMenu:
			return m.updateMenu(msg)
		case catalogStateCatalogList:
			return m.updateCatalogList(msg)
		case catalogStateItemList:
			return m.updateItemList(msg)
		case catalogStateItemDetail:
			return m.updateItemDetail(msg)
		case catalogStateCart:
			return m.updateCart(msg)
		case catalogStateSearch:
			return m.updateSearch(msg)
		}

	case catalogLoadCompleteMsg:
		m.loading = false
		return m.handleLoadComplete(msg)

	case catalogErrorMsg:
		m.loading = false
		m.errorMsg = string(msg)
		return m, nil
	}

	return m, cmd
}

// Handle menu updates
func (m *CatalogBrowserModel) updateMenu(msg tea.KeyMsg) (*CatalogBrowserModel, tea.Cmd) {
	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
		if item, ok := m.list.SelectedItem().(catalogMenuItem); ok {
			return m.handleMenuSelection(item)
		}
	}
	
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// Handle catalog list updates
func (m *CatalogBrowserModel) updateCatalogList(msg tea.KeyMsg) (*CatalogBrowserModel, tea.Cmd) {
	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
		return m.selectCatalog()
	}
	
	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

// Handle item list updates
func (m *CatalogBrowserModel) updateItemList(msg tea.KeyMsg) (*CatalogBrowserModel, tea.Cmd) {
	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
		return m.viewItemDetail()
	case key.Matches(msg, key.NewBinding(key.WithKeys("c"))):
		return m.addToCart()
	}
	
	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

// Handle item detail updates
func (m *CatalogBrowserModel) updateItemDetail(msg tea.KeyMsg) (*CatalogBrowserModel, tea.Cmd) {
	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("c"))):
		return m.addToCart()
	case key.Matches(msg, key.NewBinding(key.WithKeys("o"))):
		return m.orderNow()
	}
	return m, nil
}

// Handle cart updates
func (m *CatalogBrowserModel) updateCart(msg tea.KeyMsg) (*CatalogBrowserModel, tea.Cmd) {
	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
		return m.submitCart()
	case key.Matches(msg, key.NewBinding(key.WithKeys("d"))):
		return m.removeFromCart()
	}
	
	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

// Handle search updates
func (m *CatalogBrowserModel) updateSearch(msg tea.KeyMsg) (*CatalogBrowserModel, tea.Cmd) {
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
func (m *CatalogBrowserModel) handleMenuSelection(item catalogMenuItem) (*CatalogBrowserModel, tea.Cmd) {
	switch item.action {
	case "catalogs":
		return m.loadCatalogs()
	case "search":
		m.state = catalogStateSearch
		m.breadcrumb = append(m.breadcrumb, "Search")
		return m, textinput.Blink
	case "cart":
		return m.viewCart()
	default:
		// Load specific category items
		return m.loadCategoryItems(item.action)
	}
}

// Handle back navigation
func (m *CatalogBrowserModel) handleBack() (*CatalogBrowserModel, tea.Cmd) {
	if len(m.breadcrumb) > 1 {
		m.breadcrumb = m.breadcrumb[:len(m.breadcrumb)-1]
		
		switch m.state {
		case catalogStateCatalogList, catalogStateSearch, catalogStateCart:
			m.state = catalogStateMenu
		case catalogStateItemList:
			if m.selectedCatalog != "" {
				m.state = catalogStateCatalogList
			} else {
				m.state = catalogStateMenu
			}
		case catalogStateItemDetail:
			m.state = catalogStateItemList
			return m.updateItemListView()
		}
	}
	return m, nil
}

// Load catalogs
func (m *CatalogBrowserModel) loadCatalogs() (*CatalogBrowserModel, tea.Cmd) {
	m.loading = true
	m.state = catalogStateCatalogList
	m.breadcrumb = append(m.breadcrumb, "Catalogs")
	
	return m, func() tea.Msg {
		if m.client == nil {
			// Demo mode
			return catalogLoadCompleteMsg{
				dataType: "catalogs",
				catalogs: []CatalogInfo{
					{SysID: "1", Title: "IT Services", Description: "Technology services and equipment", Active: true, ItemCount: 45},
					{SysID: "2", Title: "Facilities", Description: "Office and building services", Active: true, ItemCount: 23},
					{SysID: "3", Title: "HR Services", Description: "Human resources and employee services", Active: true, ItemCount: 18},
				},
			}
		}
		
		// Real implementation would query sc_catalog table
		records, err := m.client.Table("sc_catalog").
			Where("active", query.OpEquals, "true").
			Execute()
		if err != nil {
			return catalogErrorMsg(fmt.Sprintf("Failed to load catalogs: %v", err))
		}
		
		var catalogs []CatalogInfo
		for _, record := range records {
			catalog := CatalogInfo{
				SysID:       fmt.Sprintf("%v", record["sys_id"]),
				Title:       fmt.Sprintf("%v", record["title"]),
				Description: fmt.Sprintf("%v", record["description"]),
				Active:      fmt.Sprintf("%v", record["active"]) == "true",
			}
			catalogs = append(catalogs, catalog)
		}
		
		return catalogLoadCompleteMsg{
			dataType: "catalogs",
			catalogs: catalogs,
		}
	}
}

// Load category items
func (m *CatalogBrowserModel) loadCategoryItems(category string) (*CatalogBrowserModel, tea.Cmd) {
	m.loading = true
	m.state = catalogStateItemList
	m.breadcrumb = append(m.breadcrumb, strings.Title(category))
	
	return m, func() tea.Msg {
		return m.getMockCatalogItems(category)
	}
}

// Get mock catalog items for demo
func (m *CatalogBrowserModel) getMockCatalogItems(category string) catalogLoadCompleteMsg {
	switch category {
	case "it_services":
		return catalogLoadCompleteMsg{
			dataType: "items",
			items: []CatalogItem{
				{SysID: "1", Name: "Laptop Request", ShortDesc: "Request a new laptop", Category: "Hardware", Price: "$1,200", Available: true, Icon: "💻"},
				{SysID: "2", Name: "Software License", ShortDesc: "Request software license", Category: "Software", Price: "$299", Available: true, Icon: "📀"},
				{SysID: "3", Name: "VPN Access", ShortDesc: "Request VPN access", Category: "Network", Price: "Free", Available: true, Icon: "🔒"},
			},
		}
	case "facilities":
		return catalogLoadCompleteMsg{
			dataType: "items",
			items: []CatalogItem{
				{SysID: "4", Name: "Office Supplies", ShortDesc: "Request office supplies", Category: "Supplies", Price: "$50", Available: true, Icon: "📝"},
				{SysID: "5", Name: "Meeting Room", ShortDesc: "Book a meeting room", Category: "Space", Price: "Free", Available: true, Icon: "🏢"},
				{SysID: "6", Name: "Parking Pass", ShortDesc: "Request parking access", Category: "Access", Price: "$25/month", Available: true, Icon: "🚗"},
			},
		}
	default:
		return catalogLoadCompleteMsg{
			dataType: "items",
			items: []CatalogItem{
				{SysID: "7", Name: "General Request", ShortDesc: "General service request", Category: "General", Price: "Varies", Available: true, Icon: "📋"},
			},
		}
	}
}

// Handle load complete
func (m *CatalogBrowserModel) handleLoadComplete(msg catalogLoadCompleteMsg) (*CatalogBrowserModel, tea.Cmd) {
	switch msg.dataType {
	case "catalogs":
		m.catalogs = msg.catalogs
		return m.updateCatalogListView()
	case "items":
		m.items = msg.items
		return m.updateItemListView()
	}
	return m, nil
}

// Update catalog list view
func (m *CatalogBrowserModel) updateCatalogListView() (*CatalogBrowserModel, tea.Cmd) {
	columns := []table.Column{
		{Title: "Title", Width: 25},
		{Title: "Description", Width: 40},
		{Title: "Items", Width: 10},
		{Title: "Status", Width: 10},
	}
	m.table.SetColumns(columns)
	
	rows := make([]table.Row, len(m.catalogs))
	for i, catalog := range m.catalogs {
		status := "Inactive"
		if catalog.Active {
			status = "Active"
		}
		
		rows[i] = table.Row{
			catalog.Title,
			catalog.Description,
			fmt.Sprintf("%d", catalog.ItemCount),
			status,
		}
	}
	
	m.table.SetRows(rows)
	return m, nil
}

// Update item list view
func (m *CatalogBrowserModel) updateItemListView() (*CatalogBrowserModel, tea.Cmd) {
	columns := []table.Column{
		{Title: "Name", Width: 30},
		{Title: "Description", Width: 40},
		{Title: "Category", Width: 15},
		{Title: "Price", Width: 10},
	}
	m.table.SetColumns(columns)
	
	rows := make([]table.Row, len(m.items))
	for i, item := range m.items {
		rows[i] = table.Row{
			item.Icon + " " + item.Name,
			item.ShortDesc,
			item.Category,
			item.Price,
		}
	}
	
	m.table.SetRows(rows)
	return m, nil
}

// Select catalog
func (m *CatalogBrowserModel) selectCatalog() (*CatalogBrowserModel, tea.Cmd) {
	if len(m.catalogs) == 0 {
		return m, nil
	}
	
	selectedIdx := m.table.Cursor()
	if selectedIdx >= len(m.catalogs) {
		return m, nil
	}
	
	catalog := m.catalogs[selectedIdx]
	m.selectedCatalog = catalog.SysID
	m.breadcrumb = append(m.breadcrumb, catalog.Title)
	
	return m.loadCatalogItems(catalog.SysID)
}

// Load catalog items
func (m *CatalogBrowserModel) loadCatalogItems(catalogID string) (*CatalogBrowserModel, tea.Cmd) {
	m.loading = true
	m.state = catalogStateItemList
	
	return m, func() tea.Msg {
		if m.client == nil {
			return m.getMockCatalogItems("general")
		}
		
		// Real implementation would query sc_cat_item table
		records, err := m.client.Table("sc_cat_item").
			Where("sc_catalog", query.OpEquals, catalogID).
			Where("active", query.OpEquals, "true").
			Execute()
		if err != nil {
			return catalogErrorMsg(fmt.Sprintf("Failed to load catalog items: %v", err))
		}
		
		var items []CatalogItem
		for _, record := range records {
			item := CatalogItem{
				SysID:     fmt.Sprintf("%v", record["sys_id"]),
				Name:      fmt.Sprintf("%v", record["name"]),
				ShortDesc: fmt.Sprintf("%v", record["short_description"]),
				Category:  fmt.Sprintf("%v", record["category"]),
				Available: fmt.Sprintf("%v", record["active"]) == "true",
			}
			items = append(items, item)
		}
		
		return catalogLoadCompleteMsg{
			dataType: "items",
			items:    items,
		}
	}
}

// View item detail
func (m *CatalogBrowserModel) viewItemDetail() (*CatalogBrowserModel, tea.Cmd) {
	if len(m.items) == 0 {
		return m, nil
	}
	
	selectedIdx := m.table.Cursor()
	if selectedIdx >= len(m.items) {
		return m, nil
	}
	
	item := m.items[selectedIdx]
	m.selectedItem = &item
	m.state = catalogStateItemDetail
	m.breadcrumb = append(m.breadcrumb, item.Name)
	
	return m, nil
}

// Add to cart
func (m *CatalogBrowserModel) addToCart() (*CatalogBrowserModel, tea.Cmd) {
	if m.selectedItem == nil && len(m.items) > 0 {
		selectedIdx := m.table.Cursor()
		if selectedIdx < len(m.items) {
			m.selectedItem = &m.items[selectedIdx]
		}
	}
	
	if m.selectedItem != nil {
		// Check if item already in cart
		for i, cartItem := range m.cart {
			if cartItem.Item.SysID == m.selectedItem.SysID {
				m.cart[i].Quantity++
				return m, nil
			}
		}
		
		// Add new item to cart
		m.cart = append(m.cart, CartItem{
			Item:      *m.selectedItem,
			Quantity:  1,
			Variables: map[string]string{},
		})
	}
	
	return m, nil
}

// Order now
func (m *CatalogBrowserModel) orderNow() (*CatalogBrowserModel, tea.Cmd) {
	if m.selectedItem != nil {
		// Add to cart and submit immediately
		m.cart = []CartItem{{
			Item:      *m.selectedItem,
			Quantity:  1,
			Variables: map[string]string{},
		}}
		return m.submitCart()
	}
	return m, nil
}

// View cart
func (m *CatalogBrowserModel) viewCart() (*CatalogBrowserModel, tea.Cmd) {
	m.state = catalogStateCart
	m.breadcrumb = append(m.breadcrumb, "Cart")
	return m.updateCartView()
}

// Update cart view
func (m *CatalogBrowserModel) updateCartView() (*CatalogBrowserModel, tea.Cmd) {
	columns := []table.Column{
		{Title: "Item", Width: 30},
		{Title: "Price", Width: 15},
		{Title: "Quantity", Width: 10},
		{Title: "Total", Width: 15},
	}
	m.table.SetColumns(columns)
	
	rows := make([]table.Row, len(m.cart))
	for i, cartItem := range m.cart {
		rows[i] = table.Row{
			cartItem.Item.Icon + " " + cartItem.Item.Name,
			cartItem.Item.Price,
			fmt.Sprintf("%d", cartItem.Quantity),
			cartItem.Item.Price, // Simplified - would calculate actual total
		}
	}
	
	m.table.SetRows(rows)
	return m, nil
}

// Submit cart
func (m *CatalogBrowserModel) submitCart() (*CatalogBrowserModel, tea.Cmd) {
	if len(m.cart) == 0 {
		return m, nil
	}
	
	// In real implementation, would submit to ServiceNow
	m.cart = []CartItem{} // Clear cart
	
	return m, nil
}

// Remove from cart
func (m *CatalogBrowserModel) removeFromCart() (*CatalogBrowserModel, tea.Cmd) {
	if len(m.cart) == 0 {
		return m, nil
	}
	
	selectedIdx := m.table.Cursor()
	if selectedIdx < len(m.cart) {
		m.cart = append(m.cart[:selectedIdx], m.cart[selectedIdx+1:]...)
		return m.updateCartView()
	}
	
	return m, nil
}

// Perform search
func (m *CatalogBrowserModel) performSearch() (*CatalogBrowserModel, tea.Cmd) {
	searchQuery := m.textInput.Value()
	m.loading = true
	
	return m, func() tea.Msg {
		// Demo search results
		results := []CatalogItem{
			{SysID: "search1", Name: fmt.Sprintf("Search Result for '%s'", searchQuery), ShortDesc: "Demo search result", Category: "Search", Price: "Free", Available: true, Icon: "🔍"},
		}
		
		return catalogLoadCompleteMsg{
			dataType: "items",
			items:    results,
		}
	}
}

// View renders the catalog browser
func (m *CatalogBrowserModel) View() string {
	if m.loading {
		return "Loading catalog data..."
	}
	
	if m.errorMsg != "" {
		return ErrorStyle.Render("Error: " + m.errorMsg)
	}
	
	// Header with breadcrumb
	header := HeaderStyle.Render(
		TitleStyle.Render("Service Catalog") + " " +
		BreadcrumbStyle.Render("/ "+strings.Join(m.breadcrumb, " / ")),
	)
	
	var content string
	
	switch m.state {
	case catalogStateMenu:
		content = m.list.View()
		
	case catalogStateCatalogList, catalogStateItemList, catalogStateCart:
		content = m.table.View()
		if m.state == catalogStateItemList {
			content += "\n\n" + InfoStyle.Render("Press 'c' to add to cart, 'enter' for details")
		} else if m.state == catalogStateCart {
			content += "\n\n" + InfoStyle.Render("Press 'enter' to submit cart, 'd' to remove item")
			if len(m.cart) > 0 {
				content += fmt.Sprintf("\n\nCart Total: %d items", len(m.cart))
			}
		}
		
	case catalogStateItemDetail:
		content = m.renderItemDetail()
		
	case catalogStateSearch:
		searchBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Padding(1, 2).
			Render(m.textInput.Value())
		
		content = fmt.Sprintf(
			"Search Catalog Items\n\n%s\n\nEnter search query and press Enter",
			searchBox,
		)
		
		if len(m.searchResults) > 0 {
			content += "\n\n" + m.table.View()
		}
	}
	
	// Footer with help
	var footerText string
	switch m.state {
	case catalogStateItemList:
		footerText = "↑/↓: navigate • enter: details • c: add to cart • esc: back • q: quit"
	case catalogStateCart:
		footerText = "↑/↓: navigate • enter: submit • d: remove • esc: back • q: quit"
	default:
		footerText = "↑/↓: navigate • enter: select • esc: back • q: quit"
	}
	footer := FooterStyle.Render(footerText)
	
	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		ContentStyle.Render(content),
		footer,
	)
}

// Render item detail
func (m *CatalogBrowserModel) renderItemDetail() string {
	if m.selectedItem == nil {
		return "No item selected"
	}
	
	item := m.selectedItem
	
	var details strings.Builder
	details.WriteString(fmt.Sprintf("%s %s\n\n", item.Icon, item.Name))
	
	details.WriteString(lipgloss.NewStyle().Bold(true).Render("Description:"))
	details.WriteString("\n")
	details.WriteString(item.ShortDesc)
	details.WriteString("\n\n")
	
	details.WriteString(lipgloss.NewStyle().Bold(true).Render("Details:"))
	details.WriteString("\n")
	details.WriteString(fmt.Sprintf("Category: %s\n", item.Category))
	details.WriteString(fmt.Sprintf("Price: %s\n", item.Price))
	details.WriteString(fmt.Sprintf("Available: %t\n", item.Available))
	
	if len(item.Variables) > 0 {
		details.WriteString("\n")
		details.WriteString(lipgloss.NewStyle().Bold(true).Render("Variables:"))
		details.WriteString("\n")
		for _, variable := range item.Variables {
			details.WriteString(fmt.Sprintf("• %s (%s)\n", variable.Label, variable.Type))
		}
	}
	
	details.WriteString("\n")
	details.WriteString(InfoStyle.Render("Press 'c' to add to cart, 'o' to order now, 'esc' to go back"))
	
	return details.String()
}

// Message types
type catalogLoadCompleteMsg struct {
	dataType string
	catalogs []CatalogInfo
	items    []CatalogItem
}

type catalogErrorMsg string