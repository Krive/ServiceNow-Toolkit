package explorer

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/Krive/ServiceNow-Toolkit/internal/tui"
)

// ViewConfiguration represents a saved view configuration
type ViewConfiguration struct {
	Name         string   `json:"name"`
	TableName    string   `json:"table_name"`
	Columns      []string `json:"columns"`
	Query        string   `json:"query"`
	Description  string   `json:"description"`
	CreatedAt    string   `json:"created_at"`
}

// ColumnCustomizer handles the slushbucket interface for column selection
type ColumnCustomizer struct {
	// Available and selected fields
	availableFields []tui.FieldMetadata
	selectedFields  []tui.FieldMetadata
	
	// Search functionality
	searchQuery    string
	filteredFields []tui.FieldMetadata
	
	// UI state
	isActive       bool
	currentPane    int // 0 = search, 1 = available, 2 = selected
	selectedIndex  int // Index in current pane
	
	// Dimensions
	width  int
	height int
}

// NewColumnCustomizer creates a new column customizer
func NewColumnCustomizer() *ColumnCustomizer {
	return &ColumnCustomizer{
		isActive:    false,
		currentPane: 1, // Start with available fields
	}
}

// SetActive sets the active state
func (cc *ColumnCustomizer) SetActive(active bool) {
	cc.isActive = active
	if active {
		cc.currentPane = 1 // Reset to available fields
		cc.selectedIndex = 0
		cc.searchQuery = ""
		
		// If we don't have any fields initialized, create mock fields as fallback
		if len(cc.availableFields) == 0 && len(cc.selectedFields) == 0 {
			cc.createMockFields([]string{"sys_id"}) // Default fallback
		}
	}
}

// IsActive returns whether the customizer is active
func (cc *ColumnCustomizer) IsActive() bool {
	return cc.isActive
}

// SetDimensions sets the width and height
func (cc *ColumnCustomizer) SetDimensions(width, height int) {
	cc.width = width
	cc.height = height
}

// InitializeFields sets up the available and selected fields
func (cc *ColumnCustomizer) InitializeFields(tableMetadata *tui.TableFieldMetadata, selectedColumns []string) {
	if tableMetadata == nil {
		// In demo mode or when metadata is not available, create mock fields
		cc.createMockFields(selectedColumns)
		return
	}
	
	// Ensure we have fields from metadata
	if len(tableMetadata.Fields) == 0 {
		// If metadata exists but has no fields, fall back to mock fields
		cc.createMockFields(selectedColumns)
		return
	}
	
	// Initialize available fields
	cc.availableFields = make([]tui.FieldMetadata, len(tableMetadata.Fields))
	copy(cc.availableFields, tableMetadata.Fields)
	
	// Sort available fields alphabetically
	sort.Slice(cc.availableFields, func(i, j int) bool {
		return cc.availableFields[i].Name < cc.availableFields[j].Name
	})
	
	// Initialize selected fields based on current selection
	cc.selectedFields = []tui.FieldMetadata{}
	selectedMap := make(map[string]bool)
	for _, col := range selectedColumns {
		selectedMap[col] = true
	}
	
	// Move selected fields to selected list and remove from available
	var remainingAvailable []tui.FieldMetadata
	for _, field := range cc.availableFields {
		if selectedMap[field.Name] {
			cc.selectedFields = append(cc.selectedFields, field)
		} else {
			remainingAvailable = append(remainingAvailable, field)
		}
	}
	cc.availableFields = remainingAvailable
	
	// Update filtered fields
	cc.updateFilteredFields()
}

// createMockFields creates mock field metadata for demo mode
func (cc *ColumnCustomizer) createMockFields(selectedColumns []string) {
	// Common ServiceNow fields with proper metadata
	mockFields := []tui.FieldMetadata{
		{Name: "sys_id", Label: "Sys ID", Type: tui.FieldTypeString},
		{Name: "number", Label: "Number", Type: tui.FieldTypeString},
		{Name: "name", Label: "Name", Type: tui.FieldTypeString},
		{Name: "title", Label: "Title", Type: tui.FieldTypeString},
		{Name: "short_description", Label: "Short Description", Type: tui.FieldTypeString},
		{Name: "description", Label: "Description", Type: tui.FieldTypeString},
		{Name: "state", Label: "State", Type: tui.FieldTypeChoice},
		{Name: "priority", Label: "Priority", Type: tui.FieldTypeChoice},
		{Name: "category", Label: "Category", Type: tui.FieldTypeString},
		{Name: "assigned_to", Label: "Assigned To", Type: tui.FieldTypeReference},
		{Name: "assignment_group", Label: "Assignment Group", Type: tui.FieldTypeReference},
		{Name: "caller_id", Label: "Caller", Type: tui.FieldTypeReference},
		{Name: "user_name", Label: "User Name", Type: tui.FieldTypeString},
		{Name: "email", Label: "Email", Type: tui.FieldTypeEmail},
		{Name: "active", Label: "Active", Type: tui.FieldTypeBoolean},
		{Name: "department", Label: "Department", Type: tui.FieldTypeString},
		{Name: "sys_created_on", Label: "Created", Type: tui.FieldTypeDateTime},
		{Name: "sys_updated_on", Label: "Updated", Type: tui.FieldTypeDateTime},
		{Name: "sys_created_by", Label: "Created By", Type: tui.FieldTypeReference},
		{Name: "sys_updated_by", Label: "Updated By", Type: tui.FieldTypeReference},
	}
	
	// Initialize available fields
	cc.availableFields = mockFields
	
	// Initialize selected fields based on current selection
	cc.selectedFields = []tui.FieldMetadata{}
	selectedMap := make(map[string]bool)
	for _, col := range selectedColumns {
		selectedMap[col] = true
	}
	
	// Move selected fields to selected list and remove from available
	var remainingAvailable []tui.FieldMetadata
	for _, field := range cc.availableFields {
		if selectedMap[field.Name] {
			cc.selectedFields = append(cc.selectedFields, field)
		} else {
			remainingAvailable = append(remainingAvailable, field)
		}
	}
	cc.availableFields = remainingAvailable
	
	// Update filtered fields
	cc.updateFilteredFields()
}

// updateFilteredFields updates the filtered fields based on search query
func (cc *ColumnCustomizer) updateFilteredFields() {
	if cc.searchQuery == "" {
		cc.filteredFields = cc.availableFields
		return
	}
	
	query := strings.ToLower(cc.searchQuery)
	cc.filteredFields = []tui.FieldMetadata{}
	
	for _, field := range cc.availableFields {
		if strings.Contains(strings.ToLower(field.Name), query) ||
		   strings.Contains(strings.ToLower(field.Label), query) {
			cc.filteredFields = append(cc.filteredFields, field)
		}
	}
}

// Update handles input events
func (cc *ColumnCustomizer) Update(msg tea.Msg) (*ColumnCustomizer, tea.Cmd) {
	if !cc.isActive {
		return cc, nil
	}
	
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
			cc.SetActive(false)
			return cc, nil
			
		case key.Matches(msg, key.NewBinding(key.WithKeys("tab"))):
			// Cycle through panes: search -> available -> selected -> search
			cc.currentPane = (cc.currentPane + 1) % 3
			cc.selectedIndex = 0
			return cc, nil
			
		case key.Matches(msg, key.NewBinding(key.WithKeys("shift+tab"))):
			// Cycle backwards through panes
			cc.currentPane = (cc.currentPane + 2) % 3 // +2 % 3 = -1 % 3
			cc.selectedIndex = 0
			return cc, nil
			
		case key.Matches(msg, key.NewBinding(key.WithKeys("/"))) :
			// Focus search pane
			cc.currentPane = 0
			return cc, nil
		}
		
		// Handle pane-specific input
		switch cc.currentPane {
		case 0: // Search pane
			return cc.handleSearchInput(msg)
		case 1: // Available fields pane
			return cc.handleAvailableFieldsInput(msg)
		case 2: // Selected fields pane
			return cc.handleSelectedFieldsInput(msg)
		}
	}
	
	return cc, nil
}

// handleSearchInput handles input in the search pane
func (cc *ColumnCustomizer) handleSearchInput(msg tea.KeyMsg) (*ColumnCustomizer, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		// Move to available fields after search
		cc.currentPane = 1
		cc.selectedIndex = 0
		return cc, nil
	case tea.KeyBackspace:
		if len(cc.searchQuery) > 0 {
			cc.searchQuery = cc.searchQuery[:len(cc.searchQuery)-1]
			cc.updateFilteredFields()
		}
	case tea.KeyRunes:
		cc.searchQuery += string(msg.Runes)
		cc.updateFilteredFields()
	}
	return cc, nil
}

// handleAvailableFieldsInput handles input in the available fields pane
func (cc *ColumnCustomizer) handleAvailableFieldsInput(msg tea.KeyMsg) (*ColumnCustomizer, tea.Cmd) {
	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("up", "k"))):
		if len(cc.filteredFields) > 0 {
			cc.selectedIndex = (cc.selectedIndex - 1 + len(cc.filteredFields)) % len(cc.filteredFields)
		}
	case key.Matches(msg, key.NewBinding(key.WithKeys("down", "j"))):
		if len(cc.filteredFields) > 0 {
			cc.selectedIndex = (cc.selectedIndex + 1) % len(cc.filteredFields)
		}
	case key.Matches(msg, key.NewBinding(key.WithKeys("enter", "right", "l"))):
		// Move field from available to selected
		if len(cc.filteredFields) > 0 && cc.selectedIndex < len(cc.filteredFields) {
			cc.moveFieldToSelected(cc.selectedIndex)
		}
	}
	return cc, nil
}

// handleSelectedFieldsInput handles input in the selected fields pane
func (cc *ColumnCustomizer) handleSelectedFieldsInput(msg tea.KeyMsg) (*ColumnCustomizer, tea.Cmd) {
	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("up", "k"))):
		if len(cc.selectedFields) > 0 {
			cc.selectedIndex = (cc.selectedIndex - 1 + len(cc.selectedFields)) % len(cc.selectedFields)
		}
	case key.Matches(msg, key.NewBinding(key.WithKeys("down", "j"))):
		if len(cc.selectedFields) > 0 {
			cc.selectedIndex = (cc.selectedIndex + 1) % len(cc.selectedFields)
		}
	case key.Matches(msg, key.NewBinding(key.WithKeys("enter", "left", "h"))):
		// Move field from selected to available
		if len(cc.selectedFields) > 0 && cc.selectedIndex < len(cc.selectedFields) {
			cc.moveFieldToAvailable(cc.selectedIndex)
		}
	case key.Matches(msg, key.NewBinding(key.WithKeys("shift+up", "K"))):
		// Move field up in selected list
		if len(cc.selectedFields) > 1 && cc.selectedIndex > 0 {
			cc.moveSelectedFieldUp()
		}
	case key.Matches(msg, key.NewBinding(key.WithKeys("shift+down", "J"))):
		// Move field down in selected list
		if len(cc.selectedFields) > 1 && cc.selectedIndex < len(cc.selectedFields)-1 {
			cc.moveSelectedFieldDown()
		}
	}
	return cc, nil
}

// moveFieldToSelected moves a field from available to selected
func (cc *ColumnCustomizer) moveFieldToSelected(index int) {
	if index < 0 || index >= len(cc.filteredFields) {
		return
	}
	
	field := cc.filteredFields[index]
	
	// Add to selected
	cc.selectedFields = append(cc.selectedFields, field)
	
	// Remove from available
	for i, availField := range cc.availableFields {
		if availField.Name == field.Name {
			cc.availableFields = append(cc.availableFields[:i], cc.availableFields[i+1:]...)
			break
		}
	}
	
	// Update filtered fields
	cc.updateFilteredFields()
	
	// Adjust selected index if needed
	if cc.selectedIndex >= len(cc.filteredFields) && len(cc.filteredFields) > 0 {
		cc.selectedIndex = len(cc.filteredFields) - 1
	}
}

// moveFieldToAvailable moves a field from selected to available
func (cc *ColumnCustomizer) moveFieldToAvailable(index int) {
	if index < 0 || index >= len(cc.selectedFields) {
		return
	}
	
	field := cc.selectedFields[index]
	
	// Add to available
	cc.availableFields = append(cc.availableFields, field)
	
	// Sort available fields alphabetically
	sort.Slice(cc.availableFields, func(i, j int) bool {
		return cc.availableFields[i].Name < cc.availableFields[j].Name
	})
	
	// Remove from selected
	cc.selectedFields = append(cc.selectedFields[:index], cc.selectedFields[index+1:]...)
	
	// Update filtered fields
	cc.updateFilteredFields()
	
	// Adjust selected index if needed
	if cc.selectedIndex >= len(cc.selectedFields) && len(cc.selectedFields) > 0 {
		cc.selectedIndex = len(cc.selectedFields) - 1
	}
}

// moveSelectedFieldUp moves a field up in the selected list
func (cc *ColumnCustomizer) moveSelectedFieldUp() {
	if cc.selectedIndex <= 0 || cc.selectedIndex >= len(cc.selectedFields) {
		return
	}
	
	// Swap with previous field
	cc.selectedFields[cc.selectedIndex], cc.selectedFields[cc.selectedIndex-1] = 
		cc.selectedFields[cc.selectedIndex-1], cc.selectedFields[cc.selectedIndex]
	
	cc.selectedIndex--
}

// moveSelectedFieldDown moves a field down in the selected list
func (cc *ColumnCustomizer) moveSelectedFieldDown() {
	if cc.selectedIndex < 0 || cc.selectedIndex >= len(cc.selectedFields)-1 {
		return
	}
	
	// Swap with next field
	cc.selectedFields[cc.selectedIndex], cc.selectedFields[cc.selectedIndex+1] = 
		cc.selectedFields[cc.selectedIndex+1], cc.selectedFields[cc.selectedIndex]
	
	cc.selectedIndex++
}

// GetSelectedColumns returns the names of selected columns in order
func (cc *ColumnCustomizer) GetSelectedColumns() []string {
	columns := make([]string, len(cc.selectedFields))
	for i, field := range cc.selectedFields {
		columns[i] = field.Name
	}
	return columns
}

// View renders the column customizer interface
func (cc *ColumnCustomizer) View() string {
	if !cc.isActive {
		return ""
	}
	
	// Calculate safe content area using same approach as XML view
	headerHeight := 3 // Column customizer always has header (from main view)
	footerHeight := cc.calculateHelpFooterHeight() // Use consistent footer calculation
	loadingHeight := 1 // Always reserve space for loading indicator
	contentHeight := cc.height - headerHeight - footerHeight - loadingHeight
	if contentHeight < 10 { // Minimum content height for column customizer
		contentHeight = 10
	}
	
	var content strings.Builder
	
	// Title (1 line + spacing = 3 lines total)
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("86")).
		Bold(true).
		Width(cc.width).
		Align(lipgloss.Center)
	
	content.WriteString(titleStyle.Render("Column Customizer"))
	content.WriteString("\n\n")
	
	// Search box (3 lines with border and spacing)
	content.WriteString(cc.renderSearchBox())
	content.WriteString("\n\n")
	
	// Slushbucket (remaining space minus instructions)
	instructionsHeight := 2 // Instructions take about 2 lines
	slushbucketHeight := contentHeight - 3 - 3 - instructionsHeight - 2 // Title + search + instructions + spacing
	if slushbucketHeight < 5 {
		slushbucketHeight = 5 // Minimum slushbucket height
	}
	
	content.WriteString(cc.renderSlushbucket(slushbucketHeight))
	content.WriteString("\n\n")
	
	// Instructions
	content.WriteString(cc.renderInstructions())
	
	return content.String()
}

// renderSearchBox renders the search input box
func (cc *ColumnCustomizer) renderSearchBox() string {
	// Calculate search box width with proper terminal constraints
	searchWidth := cc.width - 4 // Account for margins
	if searchWidth < 30 {
		searchWidth = 30 // Minimum search box width
	}
	if searchWidth > cc.width - 2 {
		searchWidth = cc.width - 2 // Don't exceed terminal width
	}
	
	searchStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(0, 1).
		Width(searchWidth)
	
	if cc.currentPane == 0 {
		searchStyle = searchStyle.
			BorderForeground(lipgloss.Color("86")).
			Bold(true)
	}
	
	searchText := "🔍 Search fields: " + cc.searchQuery
	if cc.currentPane == 0 {
		searchText += "_"
	}
	
	return searchStyle.Render(searchText)
}

// renderSlushbucket renders the main slushbucket interface
func (cc *ColumnCustomizer) renderSlushbucket(availableHeight int) string {
	// Calculate dimensions for each column with proper terminal constraints
	columnWidth := (cc.width - 10) / 2 // Leave space for borders and arrows
	listHeight := availableHeight
	
	// Apply minimum constraints
	if columnWidth < 20 {
		columnWidth = 20
	}
	if listHeight < 5 {
		listHeight = 5
	}
	
	// Ensure we don't exceed terminal width
	maxColumnWidth := (cc.width - 10) / 2
	if columnWidth > maxColumnWidth {
		columnWidth = maxColumnWidth
	}
	
	// Render available fields column
	availableHeader := "Available Fields"
	if cc.searchQuery != "" {
		availableHeader = fmt.Sprintf("Available Fields (filtered: %d/%d)", 
			len(cc.filteredFields), len(cc.availableFields))
	}
	
	availableColumn := cc.renderFieldList(
		availableHeader,
		cc.filteredFields,
		cc.currentPane == 1,
		columnWidth,
		listHeight,
	)
	
	// Render selected fields column
	selectedColumn := cc.renderFieldList(
		fmt.Sprintf("Selected Fields (%d)", len(cc.selectedFields)),
		cc.selectedFields,
		cc.currentPane == 2,
		columnWidth,
		listHeight,
	)
	
	// Render arrows between columns
	arrowStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("244")).
		Width(4).
		Align(lipgloss.Center)
	
	arrows := arrowStyle.Render("→\n←")
	
	// Combine columns horizontally
	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		availableColumn,
		arrows,
		selectedColumn,
	)
}

// renderFieldList renders a list of fields
func (cc *ColumnCustomizer) renderFieldList(title string, fields []tui.FieldMetadata, isActive bool, width, height int) string {
	// Header style
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Width(width).
		Align(lipgloss.Center).
		Padding(0, 1)
	
	if isActive {
		headerStyle = headerStyle.Foreground(lipgloss.Color("86"))
	}
	
	header := headerStyle.Render(title)
	
	// Container style
	containerStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Width(width).
		Height(height)
	
	if isActive {
		containerStyle = containerStyle.BorderForeground(lipgloss.Color("86"))
	}
	
	// Field list content
	var listContent strings.Builder
	
	if len(fields) == 0 {
		emptyStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")).
			Italic(true).
			Width(width - 4).
			Align(lipgloss.Center)
		listContent.WriteString(emptyStyle.Render("No fields"))
	} else {
		// Calculate visible range
		maxVisible := height - 4 // Account for borders and padding
		if maxVisible < 1 {
			maxVisible = 1
		}
		
		startIndex := 0
		if isActive && cc.selectedIndex >= maxVisible {
			startIndex = cc.selectedIndex - maxVisible + 1
		}
		
		endIndex := startIndex + maxVisible
		if endIndex > len(fields) {
			endIndex = len(fields)
		}
		
		for i := startIndex; i < endIndex; i++ {
			field := fields[i]
			fieldText := field.Name
			if field.Label != "" && field.Label != field.Name {
				fieldText += fmt.Sprintf(" (%s)", field.Label)
			}
			
			// Truncate if too long
			maxFieldWidth := width - 6
			if len(fieldText) > maxFieldWidth {
				fieldText = fieldText[:maxFieldWidth-3] + "..."
			}
			
			fieldStyle := lipgloss.NewStyle().
				Width(width - 4).
				Padding(0, 1)
			
			// Highlight selected item
			if isActive && i == cc.selectedIndex {
				fieldStyle = fieldStyle.
					Background(lipgloss.Color("86")).
					Foreground(lipgloss.Color("0")).
					Bold(true)
			}
			
			listContent.WriteString(fieldStyle.Render(fieldText))
			if i < endIndex-1 {
				listContent.WriteString("\n")
			}
		}
		
		// Add scroll indicator if needed
		if len(fields) > maxVisible {
			scrollInfo := fmt.Sprintf("\n[%d-%d/%d]", startIndex+1, endIndex, len(fields))
			scrollStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("244")).
				Width(width - 4).
				Align(lipgloss.Center)
			listContent.WriteString(scrollStyle.Render(scrollInfo))
		}
	}
	
	container := containerStyle.Render(listContent.String())
	
	return header + "\n" + container
}

// renderInstructions renders help instructions
func (cc *ColumnCustomizer) renderInstructions() string {
	// Calculate instruction width with proper terminal constraints
	instructionWidth := cc.width
	if instructionWidth < 40 {
		instructionWidth = 40 // Minimum instruction width
	}
	
	instructionStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("244")).
		Width(instructionWidth).
		Align(lipgloss.Center)
	
	var instructions string
	switch cc.currentPane {
	case 0: // Search
		instructions = "Type to search • Enter: go to available fields • Tab: next pane"
	case 1: // Available
		instructions = "↑↓/k/j: navigate • →/l/Enter: add field • Tab: next pane • /: search"
	case 2: // Selected
		instructions = "↑↓/k/j: navigate • ←/h/Enter: remove field • Shift+↑↓/K/J: reorder • Tab: next pane"
	}
	instructions += " • Esc: finish"
	
	return instructionStyle.Render(instructions)
}

// calculateHelpFooterHeight calculates the height needed for help footer
func (cc *ColumnCustomizer) calculateHelpFooterHeight() int {
	var instructions string
	switch cc.currentPane {
	case 0: // Search
		instructions = "Type to search • Enter: go to available fields • Tab: next pane"
	case 1: // Available
		instructions = "↑↓/k/j: navigate • →/l/Enter: add field • Tab: next pane • /: search"
	case 2: // Selected
		instructions = "↑↓/k/j: navigate • ←/h/Enter: remove field • Shift+↑↓/K/J: reorder • Tab: next pane"
	}
	instructions += " • Esc: finish"
	
	// Calculate how many lines the instructions will take when wrapped
	helpLines := 1
	if len(instructions) > 0 {
		// Account for padding - effective width is reduced by padding
		effectiveWidth := cc.width - 2 // Account for padding(0, 1) = 2 chars total
		if effectiveWidth > 0 {
			helpLines = (len(instructions) + effectiveWidth - 1) / effectiveWidth // Ceiling division
			if helpLines < 1 {
				helpLines = 1
			}
		}
	}
	
	// Footer height is just help text lines
	footerHeight := helpLines
	
	// Minimum footer height of 1, maximum of 3 to prevent excessive footer space
	if footerHeight < 1 {
		footerHeight = 1
	}
	if footerHeight > 3 {
		footerHeight = 3
	}
	
	return footerHeight
}