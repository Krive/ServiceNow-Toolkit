package explorer

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ExportScope defines what data to export
type ExportScope int

const (
	ExportCurrentView ExportScope = iota // Current view with filter and selected columns
	ExportCurrentViewAllFields           // Current view with filter but all fields
	ExportAllRecordsSelectedColumns      // All records (no filter) with selected columns
	ExportAllRecordsAllFields           // All records with all fields
)

// ExportFormat defines the export format
type ExportFormat int

const (
	ExportCSV ExportFormat = iota
	ExportJSON
)

// ReferenceValueMode defines how reference fields should be exported
type ReferenceValueMode int

const (
	ReferenceDisplayValues ReferenceValueMode = iota // Export display values (e.g., "John Doe")
	ReferenceSysIds                                  // Export sys_ids (e.g., "abc123...")
	ReferenceBoth                                    // Export both values (e.g., "John Doe (abc123)")
)

// ExportConfig represents export configuration
type ExportConfig struct {
	Scope          ExportScope
	Format         ExportFormat
	ReferenceMode  ReferenceValueMode
}

// ExportDialog handles the export dialog interface
type ExportDialog struct {
	isActive            bool
	selectedScope       int
	selectedFormat      int
	selectedReferenceMode int
	width               int
	height              int
}

// NewExportDialog creates a new export dialog
func NewExportDialog() *ExportDialog {
	return &ExportDialog{
		isActive:              false,
		selectedScope:         0,
		selectedFormat:        0,
		selectedReferenceMode: 0,
	}
}

// SetActive sets the active state
func (ed *ExportDialog) SetActive(active bool) {
	ed.isActive = active
	if active {
		ed.selectedScope = 0
		ed.selectedFormat = 0
		ed.selectedReferenceMode = 0
	}
}

// IsActive returns whether the dialog is active
func (ed *ExportDialog) IsActive() bool {
	return ed.isActive
}

// SetDimensions sets the width and height
func (ed *ExportDialog) SetDimensions(width, height int) {
	ed.width = width
	ed.height = height
}

// Update handles input events
func (ed *ExportDialog) Update(msg tea.Msg) (*ExportDialog, tea.Cmd) {
	if !ed.isActive {
		return ed, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
			ed.SetActive(false)
			return ed, nil

		case key.Matches(msg, key.NewBinding(key.WithKeys("up", "k"))):
			if ed.selectedScope > 0 {
				ed.selectedScope--
			}
			return ed, nil

		case key.Matches(msg, key.NewBinding(key.WithKeys("down", "j"))):
			if ed.selectedScope < 3 { // 4 scope options (0-3)
				ed.selectedScope++
			}
			return ed, nil

		case key.Matches(msg, key.NewBinding(key.WithKeys("left", "h"))):
			if ed.selectedFormat > 0 {
				ed.selectedFormat--
			}
			return ed, nil

		case key.Matches(msg, key.NewBinding(key.WithKeys("right", "l"))):
			if ed.selectedFormat < 1 { // 2 format options (0-1)
				ed.selectedFormat++
			}
			return ed, nil

		case key.Matches(msg, key.NewBinding(key.WithKeys("shift+left", "H"))):
			if ed.selectedReferenceMode > 0 {
				ed.selectedReferenceMode--
			}
			return ed, nil

		case key.Matches(msg, key.NewBinding(key.WithKeys("shift+right", "L"))):
			if ed.selectedReferenceMode < 2 { // 3 reference mode options (0-2)
				ed.selectedReferenceMode++
			}
			return ed, nil

		case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
			// Return the selected configuration
			ed.SetActive(false)
			return ed, func() tea.Msg {
				return exportRequestMsg{
					Scope:         ExportScope(ed.selectedScope),
					Format:        ExportFormat(ed.selectedFormat),
					ReferenceMode: ReferenceValueMode(ed.selectedReferenceMode),
				}
			}
		}
	}

	return ed, nil
}

// View renders the export dialog
func (ed *ExportDialog) View() string {
	if !ed.isActive {
		return ""
	}

	var content strings.Builder

	// Title
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("86")).
		Bold(true).
		Width(ed.width).
		Align(lipgloss.Center)

	content.WriteString(titleStyle.Render("Export Data"))
	content.WriteString("\n\n")

	// Scope selection
	content.WriteString(ed.renderScopeSelection())
	content.WriteString("\n\n")

	// Format selection
	content.WriteString(ed.renderFormatSelection())
	content.WriteString("\n\n")

	// Reference mode selection
	content.WriteString(ed.renderReferenceModeSelection())
	content.WriteString("\n\n")

	// Instructions
	content.WriteString(ed.renderInstructions())

	return content.String()
}

// renderScopeSelection renders the scope selection options
func (ed *ExportDialog) renderScopeSelection() string {
	scopeStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("86")).
		Width(ed.width).
		Align(lipgloss.Left)

	var content strings.Builder
	content.WriteString(scopeStyle.Render("📊 Export Scope:"))
	content.WriteString("\n")

	scopes := []string{
		"Current view (filtered records, selected columns)",
		"Current view (filtered records, all fields)",
		"All records (no filter, selected columns)",
		"All records (no filter, all fields)",
	}

	for i, scope := range scopes {
		optionStyle := lipgloss.NewStyle().
			Padding(0, 2).
			Width(ed.width - 4)

		if i == ed.selectedScope {
			optionStyle = optionStyle.
				Background(lipgloss.Color("86")).
				Foreground(lipgloss.Color("0")).
				Bold(true)
		}

		content.WriteString(optionStyle.Render(fmt.Sprintf("%d. %s", i+1, scope)))
		content.WriteString("\n")
	}

	return content.String()
}

// renderFormatSelection renders the format selection options
func (ed *ExportDialog) renderFormatSelection() string {
	formatStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("86")).
		Width(ed.width).
		Align(lipgloss.Left)

	var content strings.Builder
	content.WriteString(formatStyle.Render("📁 Export Format:"))
	content.WriteString("\n")

	formats := []string{"CSV", "JSON"}

	// Render formats horizontally
	var formatOptions []string
	for i, format := range formats {
		optionStyle := lipgloss.NewStyle().
			Padding(0, 2).
			Border(lipgloss.RoundedBorder()).
			MarginRight(2)

		if i == ed.selectedFormat {
			optionStyle = optionStyle.
				Background(lipgloss.Color("86")).
				Foreground(lipgloss.Color("0")).
				Bold(true)
		}

		formatOptions = append(formatOptions, optionStyle.Render(format))
	}

	content.WriteString(lipgloss.JoinHorizontal(lipgloss.Left, formatOptions...))

	return content.String()
}

// renderReferenceModeSelection renders the reference mode selection options
func (ed *ExportDialog) renderReferenceModeSelection() string {
	refModeStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("86")).
		Width(ed.width).
		Align(lipgloss.Left)

	var content strings.Builder
	content.WriteString(refModeStyle.Render("🔗 Reference Field Values:"))
	content.WriteString("\n")

	referenceModes := []string{"Display Values", "Sys IDs", "Both"}
	descriptions := []string{
		"Export human-readable names (e.g., \"John Doe\")",
		"Export raw sys_id values (e.g., \"abc123...\")",
		"Export both values (e.g., \"John Doe (abc123)\")",
	}

	// Render reference modes horizontally
	var refModeOptions []string
	for i, mode := range referenceModes {
		optionStyle := lipgloss.NewStyle().
			Padding(1, 2).
			Border(lipgloss.RoundedBorder()).
			MarginRight(2).
			Width(25).
			Height(3)

		if i == ed.selectedReferenceMode {
			optionStyle = optionStyle.
				Background(lipgloss.Color("86")).
				Foreground(lipgloss.Color("0")).
				Bold(true)
		}

		// Create content with title and description
		optionContent := fmt.Sprintf("%s\n%s", mode, descriptions[i])
		refModeOptions = append(refModeOptions, optionStyle.Render(optionContent))
	}

	content.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, refModeOptions...))

	return content.String()
}

// renderInstructions renders help instructions
func (ed *ExportDialog) renderInstructions() string {
	instructionStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("244")).
		Width(ed.width).
		Align(lipgloss.Center)

	instructions := "↑↓/k/j: select scope • ←→/h/l: select format • Shift+←→/H/L: reference mode • Enter: export • Esc: cancel"
	return instructionStyle.Render(instructions)
}

// GetSelectedConfig returns the currently selected export configuration
func (ed *ExportDialog) GetSelectedConfig() ExportConfig {
	return ExportConfig{
		Scope:         ExportScope(ed.selectedScope),
		Format:        ExportFormat(ed.selectedFormat),
		ReferenceMode: ReferenceValueMode(ed.selectedReferenceMode),
	}
}

// Export messages
type exportRequestMsg struct {
	Scope         ExportScope
	Format        ExportFormat
	ReferenceMode ReferenceValueMode
}

type exportCompletedMsg struct {
	FilePath string
	Error    error
}

// performExport performs the actual export operation
func (m *Model) performExport(scope ExportScope, format ExportFormat, referenceMode ReferenceValueMode) tea.Cmd {
	return func() tea.Msg {
		filePath, err := m.executeExport(scope, format, referenceMode)
		return exportCompletedMsg{
			FilePath: filePath,
			Error:    err,
		}
	}
}

// executeExport executes the export operation
func (m *Model) executeExport(scope ExportScope, format ExportFormat, referenceMode ReferenceValueMode) (string, error) {
	// Determine what data to export
	var records []map[string]interface{}
	var fields []string
	var err error

	switch scope {
	case ExportCurrentView:
		// Current view with filter and selected columns
		records = m.records
		fields = m.selectedColumns

	case ExportCurrentViewAllFields:
		// Current view with filter but all fields
		records, err = m.fetchAllFieldsForCurrentRecords(referenceMode)
		if err != nil {
			return "", fmt.Errorf("failed to fetch all fields: %w", err)
		}
		fields = nil // All fields

	case ExportAllRecordsSelectedColumns:
		// All records (no filter) with selected columns
		records, err = m.fetchAllRecordsWithSelectedColumns(referenceMode)
		if err != nil {
			return "", fmt.Errorf("failed to fetch all records: %w", err)
		}
		fields = m.selectedColumns

	case ExportAllRecordsAllFields:
		// All records with all fields
		records, err = m.fetchAllRecordsWithAllFields(referenceMode)
		if err != nil {
			return "", fmt.Errorf("failed to fetch all records: %w", err)
		}
		fields = nil // All fields
	}

	if len(records) == 0 {
		return "", fmt.Errorf("no records to export")
	}

	// Generate filename
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	var filename string
	switch format {
	case ExportCSV:
		filename = fmt.Sprintf("%s_export_%s.csv", m.currentTable, timestamp)
	case ExportJSON:
		filename = fmt.Sprintf("%s_export_%s.json", m.currentTable, timestamp)
	}

	// Get export directory from config
	exportDir := m.getExportDirectory()
	
	// Ensure export directory exists
	if err := os.MkdirAll(exportDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create export directory %s: %w", exportDir, err)
	}

	filePath := filepath.Join(exportDir, filename)

	// Export based on format
	switch format {
	case ExportCSV:
		err = m.exportToCSV(records, fields, filePath, referenceMode)
	case ExportJSON:
		err = m.exportToJSON(records, fields, filePath, referenceMode)
	}

	if err != nil {
		return "", fmt.Errorf("export failed: %w", err)
	}

	return filePath, nil
}

// fetchAllFieldsForCurrentRecords fetches all fields for current filtered records
func (m *Model) fetchAllFieldsForCurrentRecords(referenceMode ReferenceValueMode) ([]map[string]interface{}, error) {
	if m.client == nil {
		// Demo mode - return current records with mock additional fields
		return m.addMockFieldsToRecords(m.records), nil
	}

	// Build sys_id list from current records
	var sysIds []string
	for _, record := range m.records {
		if sysId := getRecordField(record, "sys_id"); sysId != "" {
			sysIds = append(sysIds, sysId)
		}
	}

	if len(sysIds) == 0 {
		return nil, fmt.Errorf("no sys_ids found in current records")
	}

	// Build query to fetch these specific records with all fields
	query := fmt.Sprintf("sys_idIN%s", strings.Join(sysIds, ","))

	params := map[string]string{
		"sysparm_query":         query,
		"sysparm_limit":         fmt.Sprintf("%d", len(sysIds)),
		"sysparm_display_value": getDisplayValueParam(referenceMode),
		// Don't limit fields - get all fields
	}

	return m.client.Table(m.currentTable).List(params)
}

// fetchAllRecordsWithSelectedColumns fetches all records with selected columns
func (m *Model) fetchAllRecordsWithSelectedColumns(referenceMode ReferenceValueMode) ([]map[string]interface{}, error) {
	if m.client == nil {
		// Demo mode - generate more mock records
		return m.generateMockRecordsForExport(m.selectedColumns), nil
	}

	fieldsList := m.buildFieldsList()

	params := map[string]string{
		"sysparm_limit":         "10000", // Large limit for full export
		"sysparm_fields":        fieldsList,
		"sysparm_display_value": getDisplayValueParam(referenceMode),
		"sysparm_query":         "ORDERBYDESCsys_updated_on",
	}

	return m.client.Table(m.currentTable).List(params)
}

// fetchAllRecordsWithAllFields fetches all records with all fields
func (m *Model) fetchAllRecordsWithAllFields(referenceMode ReferenceValueMode) ([]map[string]interface{}, error) {
	if m.client == nil {
		// Demo mode - generate more mock records with all fields
		return m.generateMockRecordsForExport(nil), nil
	}

	params := map[string]string{
		"sysparm_limit":         "10000", // Large limit for full export
		"sysparm_display_value": getDisplayValueParam(referenceMode),
		"sysparm_query":         "ORDERBYDESCsys_updated_on",
		// Don't limit fields - get all fields
	}

	return m.client.Table(m.currentTable).List(params)
}

// exportToCSV exports records to CSV format
func (m *Model) exportToCSV(records []map[string]interface{}, fields []string, filePath string, referenceMode ReferenceValueMode) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Determine fields to include
	var columnNames []string
	if fields != nil && len(fields) > 0 {
		columnNames = fields
	} else {
		// Extract all unique field names from records
		fieldSet := make(map[string]bool)
		for _, record := range records {
			for field := range record {
				fieldSet[field] = true
			}
		}
		for field := range fieldSet {
			columnNames = append(columnNames, field)
		}
		sort.Strings(columnNames)
	}

	// Write header
	if err := writer.Write(columnNames); err != nil {
		return err
	}

	// Write records
	for _, record := range records {
		var row []string
		for _, field := range columnNames {
			value := getExportFieldValue(record, field, referenceMode)
			row = append(row, value)
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}

	return nil
}

// exportToJSON exports records to JSON format
func (m *Model) exportToJSON(records []map[string]interface{}, fields []string, filePath string, referenceMode ReferenceValueMode) error {
	// Filter records if specific fields are requested
	var exportData []map[string]interface{}

	if fields != nil && len(fields) > 0 {
		// Export only selected fields
		for _, record := range records {
			filteredRecord := make(map[string]interface{})
			for _, field := range fields {
				if _, exists := record[field]; exists {
					// Use the same reference mode processing as CSV
					filteredRecord[field] = getExportFieldValue(record, field, referenceMode)
				}
			}
			exportData = append(exportData, filteredRecord)
		}
	} else {
		// Export all fields - need to process each field
		for _, record := range records {
			processedRecord := make(map[string]interface{})
			for field, _ := range record {
				processedRecord[field] = getExportFieldValue(record, field, referenceMode)
			}
			exportData = append(exportData, processedRecord)
		}
	}

	// Create export structure with metadata
	exportStructure := map[string]interface{}{
		"metadata": map[string]interface{}{
			"table":       m.currentTable,
			"exported_at": time.Now().Format(time.RFC3339),
			"record_count": len(exportData),
			"query":       m.currentQuery,
			"fields":      fields,
		},
		"records": exportData,
	}

	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(exportStructure)
}

// Helper functions for demo mode
func (m *Model) addMockFieldsToRecords(records []map[string]interface{}) []map[string]interface{} {
	var enrichedRecords []map[string]interface{}
	for _, record := range records {
		enriched := make(map[string]interface{})
		// Copy existing fields
		for k, v := range record {
			enriched[k] = v
		}
		// Add mock additional fields
		enriched["created_by"] = "admin"
		enriched["updated_by"] = "system"
		enriched["approval_status"] = "approved"
		enriched["work_notes"] = "Mock work notes for export"
		enrichedRecords = append(enrichedRecords, enriched)
	}
	return enrichedRecords
}

func (m *Model) generateMockRecordsForExport(fields []string) []map[string]interface{} {
	var records []map[string]interface{}
	
	// Generate more records for export (simulate full dataset)
	for i := 1; i <= 100; i++ {
		record := m.generateDemoRecord(m.currentTable, i)
		
		// Filter to selected fields if specified
		if fields != nil && len(fields) > 0 {
			filteredRecord := make(map[string]interface{})
			for _, field := range fields {
				if value, exists := record[field]; exists {
					filteredRecord[field] = value
				}
			}
			records = append(records, filteredRecord)
		} else {
			records = append(records, record)
		}
	}
	
	return records
}

// getDisplayValueParam returns the appropriate sysparm_display_value parameter for the reference mode
func getDisplayValueParam(referenceMode ReferenceValueMode) string {
	switch referenceMode {
	case ReferenceDisplayValues:
		return "true"  // Only display values
	case ReferenceSysIds:
		return "false" // Only sys_ids
	case ReferenceBoth:
		return "all"   // Both values
	default:
		return "all"   // Default to both for safety
	}
}

// getExportFieldValue returns the field value formatted according to the reference mode
func getExportFieldValue(record map[string]interface{}, field string, referenceMode ReferenceValueMode) string {
	value, displayValue, isReference := getRecordDisplayValue(record, field)
	
	if !isReference {
		// Not a reference field, return the regular value
		return value
	}
	
	// Handle reference fields based on mode
	switch referenceMode {
	case ReferenceDisplayValues:
		if displayValue != "" {
			return displayValue
		}
		return value // Fallback to sys_id if no display value
	case ReferenceSysIds:
		return value // Return the sys_id
	case ReferenceBoth:
		if displayValue != "" && displayValue != value {
			return fmt.Sprintf("%s (%s)", displayValue, value)
		}
		return value // If display value is same as value or empty, just return value
	default:
		return value
	}
}

// getExportDirectory returns the configured export directory
func (m *Model) getExportDirectory() string {
	if m.configManager != nil {
		settings := m.configManager.GetGlobalSettings()
		if settings.ExportDirectory != "" {
			return settings.ExportDirectory
		}
	}
	
	// Fallback to current working directory if config is not available
	if wd, err := os.Getwd(); err == nil {
		return wd
	}
	
	// Last resort: home directory
	if homeDir, err := os.UserHomeDir(); err == nil {
		return homeDir
	}
	
	// Final fallback: current directory
	return "."
}