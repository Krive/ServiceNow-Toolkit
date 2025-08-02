package explorer

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// View method
func (m Model) View() string {
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
	// Fixed footer height (only help text, no loading)
	footerHeight := m.calculateHelpFooterHeight()
	// Always reserve space for loading indicator to maintain consistent layout
	loadingHeight := 1

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
	case simpleStateAdvancedFilter:
		headerContent = fmt.Sprintf("%s - 🔧 Query Builder: %s%s", getCompactLogo(), m.currentTable, instanceSuffix)
		headerHeight = 3
	case simpleStateFilterBrowser:
		headerContent = fmt.Sprintf("%s - 📚 Filter Browser: %s%s", getCompactLogo(), m.currentTable, instanceSuffix)
		headerHeight = 3
	case simpleStateQuitConfirm:
		headerContent = getCompactLogo() + " - 🚪 Quit Confirmation" + instanceSuffix
		headerHeight = 3
	case simpleStateColumnCustomizer:
		headerContent = fmt.Sprintf("%s - 🎛️ Column Customizer: %s%s", getCompactLogo(), m.currentTable, instanceSuffix)
		headerHeight = 3
	case simpleStateViewManager:
		headerContent = fmt.Sprintf("%s - 📋 View Manager: %s%s", getCompactLogo(), m.currentTable, instanceSuffix)
		headerHeight = 3
	}

	// Calculate content dimensions with absolute terminal constraints
	contentHeight := m.height - headerHeight - footerHeight - loadingHeight
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
	case simpleStateAdvancedFilter:
		if m.queryBuilder != nil {
			content = m.queryBuilder.View()
		} else {
			content = "Query builder not available"
		}
	case simpleStateFilterBrowser:
		if m.filterBrowser != nil {
			content = m.filterBrowser.View()
		} else {
			content = "Filter browser not available"
		}
	case simpleStateQuitConfirm:
		content = m.renderQuitConfirmation()
	case simpleStateColumnCustomizer:
		if m.columnCustomizer != nil {
			content = m.columnCustomizer.View()
		} else {
			content = "Column customizer not available"
		}
	case simpleStateViewManager:
		content = m.renderViewManager()
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

	// Create footer with only help text (consistent height)
	footer := m.buildHelpFooter()
	
	// Create loading section separately if needed
	var loadingSection string
	if m.loading {
		loadingStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("86")). // Bright color for loading
			Width(m.width).
			Padding(0, 1).
			Bold(true)
		
		loadingSection = loadingStyle.Render("🔄 Loading...")
	}

	// Combine sections with absolute terminal constraints
	var sections []string
	if headerHeight > 0 {
		sections = append(sections, header)
	}
	sections = append(sections, content)
	// Always include footer for consistent layout
	sections = append(sections, footer)
	// Always add loading section (empty when not loading) for consistent layout
	// This ensures the number of sections never changes
	if loadingSection == "" {
		// Create empty loading section to maintain consistent layout
		loadingSection = strings.Repeat(" ", m.width)
	}
	sections = append(sections, loadingSection)

	finalView := lipgloss.JoinVertical(lipgloss.Left, sections...)

	// Final height enforcement only (width should be properly calculated now)
	finalView = m.enforceHeight(finalView)

	return finalView
}

// Render custom table input
func (m Model) renderCustomTableInput() string {
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
func (m Model) renderQueryFilter() string {
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
		Render(m.filterInput + "_")

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
func (m Model) renderXMLSearch() string {
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
func (m Model) renderScrollableXML() string {
	if m.recordXML == "" {
		return "No XML data"
	}

	lines := strings.Split(m.recordXML, "\n")

	// Calculate safe content area for XML
	headerHeight := 3 // XML view always has header
	footerHeight := m.calculateHelpFooterHeight() // Use help footer height
	loadingHeight := 1 // Always reserve space for loading indicator
	contentHeight := m.height - headerHeight - footerHeight - loadingHeight
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

// Get help text for current state - centralized function
func (m Model) getHelpText() string {
	switch m.state {
	case simpleStateTableRecords:
		if m.totalPages > 1 {
			return "↑↓/k/j: navigate • enter: view XML • ←→/h/l: pages • f: filter • c: columns • v: views • ctrl+s: save • ctrl+r: reset • r: refresh • esc: back • q: quit"
		} else {
			return "↑↓/k/j: navigate • enter: view XML • f: filter • c: columns • v: views • ctrl+s: save • ctrl+r: reset • r: refresh • esc: back • q: quit"
		}
	case simpleStateTableList:
		return "↑↓/k/j: navigate • enter: select table • t: custom table • esc: back • q: quit"
	case simpleStateRecordDetail:
		if len(m.xmlSearchResults) > 0 {
			return fmt.Sprintf("↑↓: scroll • s: search • n/N: next/prev match (%d/%d) • esc: back • q: quit", m.xmlSearchIndex+1, len(m.xmlSearchResults))
		} else {
			return "↑↓: scroll • s: search XML • esc: back • q: quit"
		}
	case simpleStateCustomTable:
		return "Type table name • enter: load table • esc: back • q: quit"
	case simpleStateQueryFilter:
		return "Type ServiceNow query • enter: apply filter • esc: back • q: quit"
	case simpleStateXMLSearch:
		return "Type search term • enter: search • esc: cancel • q: quit"
	case simpleStateAdvancedFilter:
		return "Query Builder active - follow on-screen instructions • esc: back • q: quit"
	case simpleStateFilterBrowser:
		return "Filter Browser active - follow on-screen instructions • esc: back • q: quit"
	case simpleStateQuitConfirm:
		return "←→/h/l: select button • y: yes • n: no • enter: confirm • esc: cancel"
	case simpleStateColumnCustomizer:
		return "Column Customizer active - follow on-screen instructions • esc: finish"
	case simpleStateViewManager:
		return "↑↓/k/j: navigate • enter: apply view • d: delete • esc: back • q: quit"
	default:
		return "↑↓/k/j: navigate • enter: select • esc: back • q: quit"
	}
}

// Calculate help footer height (excluding loading indicator)
func (m Model) calculateHelpFooterHeight() int {
	helpText := m.getHelpText()
	
	// Calculate how many lines the help text will take when wrapped
	helpLines := 1
	if len(helpText) > 0 {
		// Account for padding - effective width is reduced by padding
		effectiveWidth := m.width - 2 // Account for padding(0, 1) = 2 chars total
		if effectiveWidth > 0 {
			helpLines = (len(helpText) + effectiveWidth - 1) / effectiveWidth // Ceiling division
			if helpLines < 1 {
				helpLines = 1
			}
		}
	}
	
	// Footer height is just help text lines (no loading)
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

// Build help footer content (excluding loading indicator)
func (m Model) buildHelpFooter() string {
	helpText := m.getHelpText()
	
	// Always add help text if available
	if helpText != "" {
		// Style help text as muted with proper wrapping
		helpStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")). // Muted color for help
			Width(m.width).
			Padding(0, 1)
		
		return helpStyle.Render(helpText)
	}
	
	return ""
}

// Render quit confirmation dialog
func (m Model) renderQuitConfirmation() string {
	var content strings.Builder
	
	// Main message
	content.WriteString("Are you sure you want to quit?\n\n")
	
	// Button styling
	noButtonStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(0, 2).
		MarginRight(2)
		
	yesButtonStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(0, 2)
	
	// Apply selection highlighting
	if m.quitConfirmSelection == 0 {
		// "No" is selected
		noButtonStyle = noButtonStyle.
			Background(lipgloss.Color("86")).
			Foreground(lipgloss.Color("0")).
			Bold(true)
	} else {
		// "Yes" is selected
		yesButtonStyle = yesButtonStyle.
			Background(lipgloss.Color("196")).
			Foreground(lipgloss.Color("15")).
			Bold(true)
	}
	
	// Render buttons
	noButton := noButtonStyle.Render("No")
	yesButton := yesButtonStyle.Render("Yes")
	
	// Combine buttons horizontally
	buttons := lipgloss.JoinHorizontal(lipgloss.Left, noButton, yesButton)
	
	// Center the buttons
	buttonContainer := lipgloss.NewStyle().
		Width(m.width).
		Align(lipgloss.Center).
		Render(buttons)
	
	content.WriteString(buttonContainer)
	
	return content.String()
}

// Render view manager interface
func (m Model) renderViewManager() string {
	var content strings.Builder
	
	// Title
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("86")).
		Bold(true).
		Width(m.width).
		Align(lipgloss.Center)
	
	content.WriteString(titleStyle.Render("Saved View Configurations"))
	content.WriteString("\n\n")
	
	// Filter configurations by current table
	compatibleConfigs := make(map[string]*ViewConfiguration)
	for name, config := range m.viewConfigurations {
		if config.TableName == m.currentTable {
			compatibleConfigs[name] = config
		}
	}
	
	if len(compatibleConfigs) == 0 {
		emptyStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")).
			Italic(true).
			Width(m.width).
			Align(lipgloss.Center)
		
		content.WriteString(emptyStyle.Render(fmt.Sprintf("No saved views found for table: %s", m.currentTable)))
		content.WriteString("\n\n")
		content.WriteString(emptyStyle.Render("Use Ctrl+S in table view to save current configuration"))
	} else {
		// Get sorted config names for consistent ordering (only compatible ones)
		configNames := make([]string, 0, len(compatibleConfigs))
		for name := range compatibleConfigs {
			configNames = append(configNames, name)
		}
		
		// Sort alphabetically
		sort.Strings(configNames)
		
		// List saved views with selection highlighting
		for i, name := range configNames {
			config := compatibleConfigs[name]
			
			viewStyle := lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				Padding(1, 2).
				MarginBottom(1).
				Width(m.width - 4)
			
			// Highlight selected item
			if i == m.viewManagerSelection {
				viewStyle = viewStyle.
					BorderForeground(lipgloss.Color("86")).
					Background(lipgloss.Color("235"))
			}
			
			// Since we're only showing compatible configs, no need for compatibility indicator
			compatible := " ✓"
			
			viewInfo := fmt.Sprintf("📋 %s%s\n", name, compatible)
			viewInfo += fmt.Sprintf("Columns: %d selected", len(config.Columns))
			if len(config.Columns) > 0 {
				viewInfo += fmt.Sprintf(" (%s", config.Columns[0])
				if len(config.Columns) > 1 {
					viewInfo += fmt.Sprintf(", %s", config.Columns[1])
				}
				if len(config.Columns) > 2 {
					viewInfo += fmt.Sprintf(", +%d more", len(config.Columns)-2)
				}
				viewInfo += ")"
			}
			viewInfo += "\n"
			
			if config.Query != "" {
				queryDisplay := config.Query
				if len(queryDisplay) > 50 {
					queryDisplay = queryDisplay[:47] + "..."
				}
				viewInfo += fmt.Sprintf("Filter: %s\n", queryDisplay)
			}
			
			if config.Description != "" {
				viewInfo += fmt.Sprintf("Description: %s", config.Description)
			}
			
			content.WriteString(viewStyle.Render(viewInfo))
			content.WriteString("\n")
		}
		
		// Instructions
		instructionStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")).
			Width(m.width).
			Align(lipgloss.Center).
			Italic(true)
		
		instructions := "Use ↑↓ to navigate • Enter: apply view • d: delete view"
		content.WriteString(instructionStyle.Render(instructions))
	}
	
	return content.String()
}