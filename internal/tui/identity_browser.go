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

	"github.com/Krive/ServiceNow-Toolkit/pkg/servicenow"
	"github.com/Krive/ServiceNow-Toolkit/pkg/servicenow/identity"
)

// Identity browser states
type identityBrowserState int

const (
	identityBrowserStateMenu identityBrowserState = iota
	identityBrowserStateUserList
	identityBrowserStateUserDetail
	identityBrowserStateRoleList
	identityBrowserStateRoleDetail
	identityBrowserStateGroupList
	identityBrowserStateGroupDetail
	identityBrowserStateSearch
)

// Identity browser model
type IdentityBrowserModel struct {
	client       *servicenow.Client
	identityClient *identity.IdentityClient
	state        identityBrowserState
	width, height int

	// UI Components
	menuList     list.Model
	userTable    table.Model
	roleTable    table.Model
	groupTable   table.Model
	searchInput  textinput.Model

	// Data
	currentUser    *identity.User
	currentRole    *identity.Role
	currentGroup   *identity.Group
	users          []*identity.User
	roles          []*identity.Role
	groups         []*identity.Group

	// Navigation
	selectedIndex int
	
	// Search state
	searchQuery   string
	isSearching   bool
	
	// Loading
	loading       bool
	errorMsg      string
}

// Identity menu items
type identityMenuItem struct {
	title       string
	description string
	action      string
}

func (i identityMenuItem) Title() string       { return i.title }
func (i identityMenuItem) Description() string { return i.description }
func (i identityMenuItem) FilterValue() string { return i.title }

// Messages for identity browser
type usersLoadedMsg struct {
	users []*identity.User
}

type rolesLoadedMsg struct {
	roles []*identity.Role
}

type groupsLoadedMsg struct {
	groups []*identity.Group
}

type userDetailLoadedMsg struct {
	user *identity.User
}

type roleDetailLoadedMsg struct {
	role *identity.Role
}

type groupDetailLoadedMsg struct {
	group *identity.Group
}

// Create new identity browser
func NewIdentityBrowser(client *servicenow.Client) *IdentityBrowserModel {
	identityClient := identity.NewIdentityClient(client.Core())

	// Initialize menu
	items := []list.Item{
		identityMenuItem{
			title:       "👥 Users",
			description: "Browse and manage system users",
			action:      "users",
		},
		identityMenuItem{
			title:       "🛡️  Roles",
			description: "Explore user roles and permissions",
			action:      "roles",
		},
		identityMenuItem{
			title:       "👨‍👩‍👧‍👦 Groups",
			description: "View user groups and memberships",
			action:      "groups",
		},
		identityMenuItem{
			title:       "🔍 Search Users",
			description: "Search for specific users",
			action:      "search",
		},
		identityMenuItem{
			title:       "📊 Access Analysis",
			description: "Analyze user access and permissions",
			action:      "analysis",
		},
	}

	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = selectedItemStyle
	delegate.Styles.SelectedDesc = selectedItemDescStyle
	
	menuList := list.New(items, delegate, 0, 0)
	menuList.Title = "Identity Management"
	menuList.SetShowStatusBar(false)
	menuList.SetFilteringEnabled(false)
	menuList.Styles.Title = titleStyle

	// Initialize tables
	userTable := table.New(
		table.WithColumns([]table.Column{
			{Title: "USERNAME", Width: 20},
			{Title: "NAME", Width: 25},
			{Title: "EMAIL", Width: 30},
			{Title: "DEPARTMENT", Width: 20},
			{Title: "ACTIVE", Width: 8},
		}),
		table.WithRows([]table.Row{}),
		table.WithFocused(true),
		table.WithHeight(15),
	)
	userTable.SetStyles(tableStyles())

	roleTable := table.New(
		table.WithColumns([]table.Column{
			{Title: "NAME", Width: 30},
			{Title: "DESCRIPTION", Width: 40},
			{Title: "ACTIVE", Width: 8},
			{Title: "ELEVATED", Width: 10},
		}),
		table.WithRows([]table.Row{}),
		table.WithFocused(true),
		table.WithHeight(15),
	)
	roleTable.SetStyles(tableStyles())

	groupTable := table.New(
		table.WithColumns([]table.Column{
			{Title: "NAME", Width: 30},
			{Title: "DESCRIPTION", Width: 40},
			{Title: "TYPE", Width: 15},
			{Title: "ACTIVE", Width: 8},
		}),
		table.WithRows([]table.Row{}),
		table.WithFocused(true),
		table.WithHeight(15),
	)
	groupTable.SetStyles(tableStyles())

	// Initialize search input
	searchInput := textinput.New()
	searchInput.Placeholder = "Enter username or email to search..."
	searchInput.Width = 50

	return &IdentityBrowserModel{
		client:         client,
		identityClient: identityClient,
		state:          identityBrowserStateMenu,
		menuList:       menuList,
		userTable:      userTable,
		roleTable:      roleTable,
		groupTable:     groupTable,
		searchInput:    searchInput,
	}
}

// Initialize identity browser
func (m *IdentityBrowserModel) Init() tea.Cmd {
	return nil
}

// Update identity browser
func (m *IdentityBrowserModel) Update(msg tea.Msg) (*IdentityBrowserModel, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.menuList.SetSize(msg.Width-4, msg.Height-10)
		m.userTable.SetWidth(msg.Width - 4)
		m.userTable.SetHeight(msg.Height - 15)
		m.roleTable.SetWidth(msg.Width - 4)
		m.roleTable.SetHeight(msg.Height - 15)
		m.groupTable.SetWidth(msg.Width - 4)
		m.groupTable.SetHeight(msg.Height - 15)
		return m, nil

	case tea.KeyMsg:
		switch m.state {
		case identityBrowserStateMenu:
			switch {
			case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
				if item, ok := m.menuList.SelectedItem().(identityMenuItem); ok {
					return m.handleMenuSelection(item)
				}
			}
			m.menuList, cmd = m.menuList.Update(msg)
			return m, cmd

		case identityBrowserStateUserList:
			switch {
			case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
				if len(m.users) > 0 && m.userTable.Cursor() < len(m.users) {
					selectedUser := m.users[m.userTable.Cursor()]
					m.state = identityBrowserStateUserDetail
					return m, m.loadUserDetail(selectedUser.SysID)
				}
			case key.Matches(msg, key.NewBinding(key.WithKeys("r"))):
				return m, m.loadUsers()
			case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
				m.state = identityBrowserStateMenu
				return m, nil
			}
			m.userTable, cmd = m.userTable.Update(msg)
			return m, cmd

		case identityBrowserStateRoleList:
			switch {
			case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
				if len(m.roles) > 0 && m.roleTable.Cursor() < len(m.roles) {
					selectedRole := m.roles[m.roleTable.Cursor()]
					m.state = identityBrowserStateRoleDetail
					return m, m.loadRoleDetail(selectedRole.SysID)
				}
			case key.Matches(msg, key.NewBinding(key.WithKeys("r"))):
				return m, m.loadRoles()
			case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
				m.state = identityBrowserStateMenu
				return m, nil
			}
			m.roleTable, cmd = m.roleTable.Update(msg)
			return m, cmd

		case identityBrowserStateGroupList:
			switch {
			case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
				if len(m.groups) > 0 && m.groupTable.Cursor() < len(m.groups) {
					selectedGroup := m.groups[m.groupTable.Cursor()]
					m.state = identityBrowserStateGroupDetail
					return m, m.loadGroupDetail(selectedGroup.SysID)
				}
			case key.Matches(msg, key.NewBinding(key.WithKeys("r"))):
				return m, m.loadGroups()
			case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
				m.state = identityBrowserStateMenu
				return m, nil
			}
			m.groupTable, cmd = m.groupTable.Update(msg)
			return m, cmd

		case identityBrowserStateSearch:
			switch {
			case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
				if m.searchInput.Value() != "" {
					return m, m.searchUsers(m.searchInput.Value())
				}
			case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
				m.state = identityBrowserStateMenu
				m.isSearching = false
				m.searchInput.Blur()
				return m, nil
			}
			m.searchInput, cmd = m.searchInput.Update(msg)
			return m, cmd

		case identityBrowserStateUserDetail, identityBrowserStateRoleDetail, identityBrowserStateGroupDetail:
			switch {
			case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
				// Go back to previous list state
				switch m.state {
				case identityBrowserStateUserDetail:
					m.state = identityBrowserStateUserList
				case identityBrowserStateRoleDetail:
					m.state = identityBrowserStateRoleList
				case identityBrowserStateGroupDetail:
					m.state = identityBrowserStateGroupList
				}
				return m, nil
			}
		}

	case usersLoadedMsg:
		m.users = msg.users
		m.updateUserTable()
		m.loading = false
		return m, nil

	case rolesLoadedMsg:
		m.roles = msg.roles
		m.updateRoleTable()
		m.loading = false
		return m, nil

	case groupsLoadedMsg:
		m.groups = msg.groups
		m.updateGroupTable()
		m.loading = false
		return m, nil

	case userDetailLoadedMsg:
		m.currentUser = msg.user
		m.loading = false
		return m, nil

	case roleDetailLoadedMsg:
		m.currentRole = msg.role
		m.loading = false
		return m, nil

	case groupDetailLoadedMsg:
		m.currentGroup = msg.group
		m.loading = false
		return m, nil

	case errorMsg:
		m.errorMsg = string(msg)
		m.loading = false
		return m, nil
	}

	return m, tea.Batch(cmds...)
}

// Handle menu selection
func (m *IdentityBrowserModel) handleMenuSelection(item identityMenuItem) (*IdentityBrowserModel, tea.Cmd) {
	switch item.action {
	case "users":
		m.state = identityBrowserStateUserList
		return m, m.loadUsers()
	case "roles":
		m.state = identityBrowserStateRoleList
		return m, m.loadRoles()
	case "groups":
		m.state = identityBrowserStateGroupList
		return m, m.loadGroups()
	case "search":
		m.state = identityBrowserStateSearch
		m.isSearching = true
		m.searchInput.Focus()
		return m, textinput.Blink
	default:
		m.errorMsg = fmt.Sprintf("Feature '%s' not yet implemented", item.action)
		return m, nil
	}
}

// Load users
func (m *IdentityBrowserModel) loadUsers() tea.Cmd {
	m.loading = true
	
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		filter := &identity.UserFilter{
			Limit: 100,
		}

		users, err := m.identityClient.ListUsersWithContext(ctx, filter)
		if err != nil {
			return errorMsg(fmt.Sprintf("Failed to load users: %v", err))
		}

		return usersLoadedMsg{users: users}
	}
}

// Load roles
func (m *IdentityBrowserModel) loadRoles() tea.Cmd {
	m.loading = true
	
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		filter := &identity.RoleFilter{
			Limit: 100,
		}

		roleClient := m.identityClient.NewRoleClient()
		roles, err := roleClient.ListRolesWithContext(ctx, filter)
		if err != nil {
			return errorMsg(fmt.Sprintf("Failed to load roles: %v", err))
		}

		return rolesLoadedMsg{roles: roles}
	}
}

// Load groups
func (m *IdentityBrowserModel) loadGroups() tea.Cmd {
	m.loading = true
	
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		filter := &identity.GroupFilter{
			Limit: 100,
		}

		groupClient := m.identityClient.NewGroupClient()
		groups, err := groupClient.ListGroupsWithContext(ctx, filter)
		if err != nil {
			return errorMsg(fmt.Sprintf("Failed to load groups: %v", err))
		}

		return groupsLoadedMsg{groups: groups}
	}
}

// Load user detail
func (m *IdentityBrowserModel) loadUserDetail(sysID string) tea.Cmd {
	m.loading = true
	
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		user, err := m.identityClient.GetUserWithContext(ctx, sysID)
		if err != nil {
			return errorMsg(fmt.Sprintf("Failed to load user detail: %v", err))
		}

		return userDetailLoadedMsg{user: user}
	}
}

// Load role detail
func (m *IdentityBrowserModel) loadRoleDetail(sysID string) tea.Cmd {
	m.loading = true
	
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		roleClient := m.identityClient.NewRoleClient()
		role, err := roleClient.GetRoleWithContext(ctx, sysID)
		if err != nil {
			return errorMsg(fmt.Sprintf("Failed to load role detail: %v", err))
		}

		return roleDetailLoadedMsg{role: role}
	}
}

// Load group detail
func (m *IdentityBrowserModel) loadGroupDetail(sysID string) tea.Cmd {
	m.loading = true
	
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		groupClient := m.identityClient.NewGroupClient()
		group, err := groupClient.GetGroupWithContext(ctx, sysID)
		if err != nil {
			return errorMsg(fmt.Sprintf("Failed to load group detail: %v", err))
		}

		return groupDetailLoadedMsg{group: group}
	}
}

// Search users
func (m *IdentityBrowserModel) searchUsers(query string) tea.Cmd {
	m.loading = true
	
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		filter := &identity.UserFilter{
			Email: query,
			Limit: 50,
		}

		users, err := m.identityClient.ListUsersWithContext(ctx, filter)
		if err != nil {
			return errorMsg(fmt.Sprintf("Failed to search users: %v", err))
		}

		return usersLoadedMsg{users: users}
	}
}

// Update user table
func (m *IdentityBrowserModel) updateUserTable() {
	rows := make([]table.Row, len(m.users))
	for i, user := range m.users {
		active := "No"
		if user.Active {
			active = "Yes"
		}
		
		rows[i] = table.Row{
			user.UserName,
			fmt.Sprintf("%s %s", user.FirstName, user.LastName),
			user.Email,
			user.Department,
			active,
		}
	}
	m.userTable.SetRows(rows)
}

// Update role table
func (m *IdentityBrowserModel) updateRoleTable() {
	rows := make([]table.Row, len(m.roles))
	for i, role := range m.roles {
		active := "No"
		if role.Active {
			active = "Yes"
		}
		
		elevated := "No"
		if role.ElevatedPrivilege {
			elevated = "Yes"
		}
		
		rows[i] = table.Row{
			role.Name,
			role.Description,
			active,
			elevated,
		}
	}
	m.roleTable.SetRows(rows)
}

// Update group table
func (m *IdentityBrowserModel) updateGroupTable() {
	rows := make([]table.Row, len(m.groups))
	for i, group := range m.groups {
		active := "No"
		if group.Active {
			active = "Yes"
		}
		
		rows[i] = table.Row{
			group.Name,
			group.Description,
			group.Type,
			active,
		}
	}
	m.groupTable.SetRows(rows)
}

// View identity browser
func (m *IdentityBrowserModel) View() string {
	if m.loading {
		return "Loading..."
	}

	if m.errorMsg != "" {
		return errorStyle.Render("Error: " + m.errorMsg)
	}

	switch m.state {
	case identityBrowserStateMenu:
		return m.menuList.View()

	case identityBrowserStateUserList:
		content := fmt.Sprintf("Users (%d found)\n\n", len(m.users))
		if len(m.users) == 0 {
			content += "No users found."
		} else {
			content += m.userTable.View()
		}
		content += "\n\nPress 'r' to refresh, 'enter' to view details, 'esc' to go back"
		return content

	case identityBrowserStateRoleList:
		content := fmt.Sprintf("Roles (%d found)\n\n", len(m.roles))
		if len(m.roles) == 0 {
			content += "No roles found."
		} else {
			content += m.roleTable.View()
		}
		content += "\n\nPress 'r' to refresh, 'enter' to view details, 'esc' to go back"
		return content

	case identityBrowserStateGroupList:
		content := fmt.Sprintf("Groups (%d found)\n\n", len(m.groups))
		if len(m.groups) == 0 {
			content += "No groups found."
		} else {
			content += m.groupTable.View()
		}
		content += "\n\nPress 'r' to refresh, 'enter' to view details, 'esc' to go back"
		return content

	case identityBrowserStateSearch:
		return fmt.Sprintf(
			"Search Users\n\n%s\n\nPress Enter to search, Esc to cancel",
			m.searchInput.View(),
		)

	case identityBrowserStateUserDetail:
		return m.renderUserDetail()

	case identityBrowserStateRoleDetail:
		return m.renderRoleDetail()

	case identityBrowserStateGroupDetail:
		return m.renderGroupDetail()
	}

	return "Unknown state"
}

// Render user detail
func (m *IdentityBrowserModel) renderUserDetail() string {
	if m.currentUser == nil {
		return "No user selected"
	}

	var content strings.Builder
	content.WriteString(fmt.Sprintf("User Detail - %s\n\n", m.currentUser.UserName))

	content.WriteString(fmt.Sprintf("%-20s: %s\n", "Full Name", fmt.Sprintf("%s %s", m.currentUser.FirstName, m.currentUser.LastName)))
	content.WriteString(fmt.Sprintf("%-20s: %s\n", "Email", m.currentUser.Email))
	content.WriteString(fmt.Sprintf("%-20s: %s\n", "Department", m.currentUser.Department))
	content.WriteString(fmt.Sprintf("%-20s: %s\n", "Title", m.currentUser.Title))
	content.WriteString(fmt.Sprintf("%-20s: %s\n", "Company", m.currentUser.Company))
	content.WriteString(fmt.Sprintf("%-20s: %s\n", "Location", m.currentUser.Location))
	content.WriteString(fmt.Sprintf("%-20s: %s\n", "Manager", m.currentUser.Manager))
	content.WriteString(fmt.Sprintf("%-20s: %v\n", "Active", m.currentUser.Active))
	content.WriteString(fmt.Sprintf("%-20s: %v\n", "VIP", m.currentUser.VIP))
	content.WriteString(fmt.Sprintf("%-20s: %s\n", "Source", m.currentUser.Source))

	if !m.currentUser.LastLoginTime.IsZero() {
		content.WriteString(fmt.Sprintf("%-20s: %s\n", "Last Login", m.currentUser.LastLoginTime.Format(time.RFC3339)))
	}

	content.WriteString(fmt.Sprintf("%-20s: %s\n", "Created By", m.currentUser.CreatedBy))
	content.WriteString(fmt.Sprintf("%-20s: %s\n", "Created On", m.currentUser.CreatedOn.Format(time.RFC3339)))

	content.WriteString("\n\nPress 'esc' to go back")
	return content.String()
}

// Render role detail
func (m *IdentityBrowserModel) renderRoleDetail() string {
	if m.currentRole == nil {
		return "No role selected"
	}

	var content strings.Builder
	content.WriteString(fmt.Sprintf("Role Detail - %s\n\n", m.currentRole.Name))

	content.WriteString(fmt.Sprintf("%-20s: %s\n", "Description", m.currentRole.Description))
	content.WriteString(fmt.Sprintf("%-20s: %v\n", "Active", m.currentRole.Active))
	content.WriteString(fmt.Sprintf("%-20s: %v\n", "Assignable", m.currentRole.Assignable))
	content.WriteString(fmt.Sprintf("%-20s: %v\n", "Can Delegate", m.currentRole.CanDelegate))
	content.WriteString(fmt.Sprintf("%-20s: %v\n", "Elevated Privilege", m.currentRole.ElevatedPrivilege))
	content.WriteString(fmt.Sprintf("%-20s: %v\n", "Grants Admin", m.currentRole.GrantsAdmin))
	content.WriteString(fmt.Sprintf("%-20s: %s\n", "Application", m.currentRole.ApplicationScope))

	content.WriteString("\n\nPress 'esc' to go back")
	return content.String()
}

// Render group detail
func (m *IdentityBrowserModel) renderGroupDetail() string {
	if m.currentGroup == nil {
		return "No group selected"
	}

	var content strings.Builder
	content.WriteString(fmt.Sprintf("Group Detail - %s\n\n", m.currentGroup.Name))

	content.WriteString(fmt.Sprintf("%-20s: %s\n", "Description", m.currentGroup.Description))
	content.WriteString(fmt.Sprintf("%-20s: %s\n", "Type", m.currentGroup.Type))
	content.WriteString(fmt.Sprintf("%-20s: %v\n", "Active", m.currentGroup.Active))
	content.WriteString(fmt.Sprintf("%-20s: %s\n", "Email", m.currentGroup.Email))
	content.WriteString(fmt.Sprintf("%-20s: %s\n", "Manager", m.currentGroup.Manager))
	content.WriteString(fmt.Sprintf("%-20s: %s\n", "Parent", m.currentGroup.Parent))
	content.WriteString(fmt.Sprintf("%-20s: %s\n", "Cost Center", m.currentGroup.CostCenter))

	content.WriteString("\n\nPress 'esc' to go back")
	return content.String()
}