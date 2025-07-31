package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

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