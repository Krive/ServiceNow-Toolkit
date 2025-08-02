package explorer

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	demoMode            bool
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

// Calculate actual header height based on state
func (m *Model) getHeaderHeight() int {
	if m.state == simpleStateMain {
		return 0 // No header for main menu
	}
	// Header with border and padding typically takes 3-4 lines
	return 4
}

// Calculate actual footer height based on loading state
func (m *Model) getFooterHeight() int {
	baseFooterHeight := 2 // Help text with padding
	if m.loading {
		return baseFooterHeight + 2 // Add loading indicator
	}
	return baseFooterHeight
}

// Calculate available content height
func (m *Model) getAvailableContentHeight() int {
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
func (m *Model) enforceWidth(text string) string {
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
func (m *Model) countVisibleChars(s string) int {
	// Simple approach: remove common ANSI sequences
	ansiRegex := `\x1b\[[0-9;]*[a-zA-Z]`
	re := regexp.MustCompile(ansiRegex)
	cleaned := re.ReplaceAllString(s, "")
	return len(cleaned)
}

// Truncate string while trying to preserve ANSI sequences
func (m *Model) truncatePreserveAnsi(s string, maxWidth int) string {
	if len(s) <= maxWidth {
		return s
	}

	// Simple truncation for now - more sophisticated ANSI handling could be added
	return s[:maxWidth]
}

// Enforce absolute terminal height boundary - truncates content to fit
func (m *Model) enforceHeight(text string) string {
	lines := strings.Split(text, "\n")
	
	// Only enforce if we significantly exceed terminal height
	// This prevents minor overflow issues while still preventing major layout breaks
	maxAllowedHeight := m.height + 1 // Allow 1 line overflow tolerance
	if len(lines) <= maxAllowedHeight {
		return text
	}

	// Truncate at terminal height, preserving layout structure
	return strings.Join(lines[:m.height], "\n")
}

// Update list size based on current state and terminal size
func (m *Model) updateListSize() {
	if m.width > 0 && m.height > 0 {
		// Calculate precise layout dimensions
		headerHeight := 3 // All states now have headers
		footerHeight := m.calculateHelpFooterHeight() // Use help footer height (no loading)
		// Always reserve space for loading indicator to maintain consistent layout
		loadingHeight := 1

		contentHeight := m.height - headerHeight - footerHeight - loadingHeight
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

// Get compact ServiceNow Toolkit logo for headers
func getCompactLogo() string {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("39")).
		Bold(true).
		Render("🚀 ServiceNow Toolkit")
}

// Get instance name from resolved instance URL
func (m *Model) getInstanceName() string {
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

// Format record display based on selected columns
func (m *Model) formatRecordDisplay(record map[string]interface{}, index int) (string, string) {
	if len(m.selectedColumns) == 0 {
		// Fallback to original behavior if no columns selected
		return m.formatRecordDisplayFallback(record, index)
	}

	// Use selected columns to build display
	var columnParts []string
	var title string
	
	for i, column := range m.selectedColumns {
		if val := getRecordField(record, column); val != "" {
			// Truncate long values
			displayValue := val
			if len(displayValue) > 50 {
				displayValue = displayValue[:47] + "..."
			}
			
			// First column becomes the title
			if i == 0 {
				title = displayValue
			} else {
				// Add column label for clarity
				columnLabel := column
				if m.tableMetadata != nil {
					if field := m.tableMetadata.FindField(column); field != nil && field.Label != "" {
						columnLabel = field.Label
					}
				}
				columnParts = append(columnParts, fmt.Sprintf("%s: %s", columnLabel, displayValue))
			}
		} else if i == 0 {
			// Ensure we always have a title
			title = fmt.Sprintf("%s: (empty)", column)
		}
	}

	// If no title was set (all values empty), fallback
	if title == "" {
		if sysId := getRecordField(record, "sys_id"); sysId != "" {
			title = fmt.Sprintf("Record %s", sysId[:8])
		} else {
			title = fmt.Sprintf("Record #%d", m.currentPage*m.pageSize+index+1)
		}
	}

	desc := strings.Join(columnParts, " | ")
	if desc == "" {
		desc = fmt.Sprintf("Table: %s | Columns: %d", m.currentTable, len(m.selectedColumns))
	}

	return title, desc
}

// Fallback to original display logic when no columns are selected
func (m *Model) formatRecordDisplayFallback(record map[string]interface{}, index int) (string, string) {
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
func (m *Model) generateDemoRecord(tableName string, recordNum int) map[string]interface{} {
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

// Build fields list for API calls based on selected columns
func (m *Model) buildFieldsList() string {
	// Always include sys_id for record identification
	fieldsSet := map[string]bool{
		"sys_id": true,
	}
	
	// Add selected columns
	for _, column := range m.selectedColumns {
		fieldsSet[column] = true
	}
	
	// If no columns selected, use intelligent defaults
	if len(m.selectedColumns) == 0 {
		defaultFields := []string{
			"number", "name", "title", "display_name", "sys_name",
			"short_description", "description", "state", "priority", 
			"category", "assigned_to", "assignment_group", "caller_id",
			"user_name", "email", "active", "department", 
			"sys_updated_on", "sys_created_on",
		}
		for _, field := range defaultFields {
			fieldsSet[field] = true
		}
	}
	
	// Convert to slice and join
	var fields []string
	for field := range fieldsSet {
		fields = append(fields, field)
	}
	
	return strings.Join(fields, ",")
}

// Setup intelligent default columns for a table
func (m *Model) setupDefaultColumnsForTable(tableName string) {
	// Define table-specific intelligent defaults
	tableDefaults := map[string][]string{
		"incident": {"number", "short_description", "state", "priority", "assigned_to", "sys_updated_on"},
		"problem": {"number", "short_description", "state", "priority", "sys_updated_on"},
		"change_request": {"number", "short_description", "state", "priority", "type", "sys_updated_on"},
		"sys_user": {"user_name", "name", "email", "department", "active"},
		"sys_user_group": {"name", "description", "active", "sys_updated_on"},
		"cmdb_ci_server": {"name", "operational_status", "environment", "location", "sys_updated_on"},
		"cmdb_ci_computer": {"name", "operational_status", "model_id", "location", "sys_updated_on"},
		"sc_request": {"number", "short_description", "state", "priority", "requested_for", "sys_updated_on"},
		"kb_knowledge": {"number", "title", "knowledge_base", "workflow_state", "sys_updated_on"},
	}
	
	// Use table-specific defaults if available, otherwise use generic defaults
	if defaults, exists := tableDefaults[tableName]; exists {
		m.selectedColumns = defaults
	} else {
		// Generic defaults for unknown tables
		m.selectedColumns = []string{"sys_id", "name", "sys_updated_on"}
	}
}

// Get configurations compatible with current table
func (m *Model) getCompatibleConfigurations() map[string]*ViewConfiguration {
	compatible := make(map[string]*ViewConfiguration)
	for name, config := range m.viewConfigurations {
		if config.TableName == m.currentTable {
			compatible[name] = config
		}
	}
	return compatible
}

// Update record title with pagination info and current query
func (m *Model) updateRecordTitle(tableName string) {
	pageInfo := ""
	if m.totalPages > 1 {
		pageInfo = fmt.Sprintf(" (Page %d/%d)", m.currentPage+1, m.totalPages)
	}

	// Add total records count if available
	totalInfo := ""
	if m.totalRecords > 0 {
		totalInfo = fmt.Sprintf(" - %d total", m.totalRecords)
	}

	// Add current query info if one is applied
	queryInfo := ""
	if m.currentQuery != "" && m.currentQuery != "ORDERBYDESCsys_updated_on" {
		// Show a shortened version of the query
		shortQuery := m.currentQuery
		if len(shortQuery) > 50 {
			shortQuery = shortQuery[:47] + "..."
		}
		queryInfo = fmt.Sprintf(" | Filter: %s", shortQuery)
	}

	m.list.Title = fmt.Sprintf("Records: %s%s%s%s", tableName, totalInfo, pageInfo, queryInfo)
}