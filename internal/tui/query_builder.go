package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	
	"github.com/Krive/ServiceNow-Toolkit/pkg/servicenow"
	"github.com/Krive/ServiceNow-Toolkit/pkg/servicenow/query"
)

// QueryBuilderState represents the current state of the query builder
type QueryBuilderState int

const (
	QueryBuilderStateFieldSelection QueryBuilderState = iota
	QueryBuilderStateOperatorSelection
	QueryBuilderStateValueInput
	QueryBuilderStateChoiceSelection
	QueryBuilderStateReferenceSearch
	QueryBuilderStateDateRange
	QueryBuilderStateCalendar
	QueryBuilderStateDateRangePicker
	QueryBuilderStateLogicalOperator
	QueryBuilderStatePreview
)

// LogicalOperator represents AND/OR logic between conditions
type LogicalOperator string

const (
	LogicalAnd LogicalOperator = "AND"
	LogicalOr  LogicalOperator = "OR"
)

// QueryCondition represents a single query condition
type QueryCondition struct {
	Field    FieldMetadata    `json:"field"`
	Operator query.Operator   `json:"operator"`
	Value    string          `json:"value"`
	StartDate *time.Time     `json:"start_date,omitempty"`
	EndDate   *time.Time     `json:"end_date,omitempty"`
	LogicalOp LogicalOperator `json:"logical_op,omitempty"` // AND/OR for connecting to next condition
}

// OperatorInfo contains display information for operators
type OperatorInfo struct {
	Operator    query.Operator
	Label       string
	Description string
	RequiresValue bool
	DateOnly     bool
}

// QueryBuilderModel handles the interactive query building interface
type QueryBuilderModel struct {
	state        QueryBuilderState
	width, height int
	
	// ServiceNow client for reference lookups
	client *servicenow.Client
	
	// Field metadata
	tableMetadata *TableFieldMetadata
	
	// UI Components
	fieldList     list.Model
	operatorList  list.Model
	logicalOpList list.Model
	choiceList    list.Model
	referenceList list.Model
	valueInput    textinput.Model
	referenceSearchInput textinput.Model
	dateStartInput textinput.Model
	dateEndInput   textinput.Model
	
	// Enhanced date/time components
	calendar          *CalendarModel
	dateRangePicker   *DateRangePickerModel
	useAdvancedDate   bool
	
	// Current condition being built
	currentCondition QueryCondition
	conditions       []QueryCondition
	
	// Navigation
	selectedIndex    int
	focusedInput     int // 0 = start date, 1 = end date
	
	// State
	isActive         bool
	showPreview      bool
	previewQuery     string
	errorMsg         string
	
	// Validation
	validator        *QueryValidator
	validationResult *ValidationResult
	showValidation   bool
	
	// Reference lookup data
	referenceRecords []map[string]interface{}
	loadingReferences bool
}

// Available operators grouped by field type
var operatorsByFieldType = map[FieldType][]OperatorInfo{
	FieldTypeString: {
		{query.OpEquals, "Equals", "Exact match", true, false},
		{query.OpNotEquals, "Not Equals", "Does not match", true, false},
		{query.OpContains, "Contains", "Contains text", true, false},
		{query.OpDoesNotContain, "Does Not Contain", "Does not contain text", true, false},
		{query.OpStartsWith, "Starts With", "Begins with text", true, false},
		{query.OpEndsWith, "Ends With", "Ends with text", true, false},
		{query.OpIsEmpty, "Is Empty", "Field is empty", false, false},
		{query.OpIsNotEmpty, "Is Not Empty", "Field is not empty", false, false},
		{query.OpLike, "Like", "Pattern match (use % wildcards)", true, false},
	},
	FieldTypeInteger: {
		{query.OpEquals, "Equals", "Equal to number", true, false},
		{query.OpNotEquals, "Not Equals", "Not equal to number", true, false},
		{query.OpGreaterThan, "Greater Than", "Greater than number", true, false},
		{query.OpGreaterThanOrEqual, "Greater or Equal", "Greater than or equal to number", true, false},
		{query.OpLessThan, "Less Than", "Less than number", true, false},
		{query.OpLessThanOrEqual, "Less or Equal", "Less than or equal to number", true, false},
		{query.OpBetween, "Between", "Between two numbers", true, false},
		{query.OpIsEmpty, "Is Empty", "Field is empty", false, false},
		{query.OpIsNotEmpty, "Is Not Empty", "Field is not empty", false, false},
	},
	FieldTypeDate: {
		{query.OpEquals, "On Date", "On specific date", true, false},
		{query.OpNotEquals, "Not On Date", "Not on specific date", true, false},
		{query.OpGreaterThan, "After", "After date", true, false},
		{query.OpLessThan, "Before", "Before date", true, false},
		{query.OpBetween, "Date Range", "Between two dates", true, true},
		{query.OpToday, "Today", "Today", false, true},
		{query.OpYesterday, "Yesterday", "Yesterday", false, true},
		{query.OpThisWeek, "This Week", "This week", false, true},
		{query.OpLastWeek, "Last Week", "Last week", false, true},
		{query.OpThisMonth, "This Month", "This month", false, true},
		{query.OpLastMonth, "Last Month", "Last month", false, true},
		{query.OpThisYear, "This Year", "This year", false, true},
		{query.OpLastYear, "Last Year", "Last year", false, true},
		{query.OpIsEmpty, "Is Empty", "Field is empty", false, false},
		{query.OpIsNotEmpty, "Is Not Empty", "Field is not empty", false, false},
	},
	FieldTypeDateTime: {
		{query.OpEquals, "On Date", "On specific date/time", true, false},
		{query.OpNotEquals, "Not On Date", "Not on specific date/time", true, false},
		{query.OpGreaterThan, "After", "After date/time", true, false},
		{query.OpLessThan, "Before", "Before date/time", true, false},
		{query.OpBetween, "Date Range", "Between two dates", true, true},
		{query.OpToday, "Today", "Today", false, true},
		{query.OpYesterday, "Yesterday", "Yesterday", false, true},
		{query.OpThisWeek, "This Week", "This week", false, true},
		{query.OpLastWeek, "Last Week", "Last week", false, true},
		{query.OpThisMonth, "This Month", "This month", false, true},
		{query.OpLastMonth, "Last Month", "Last month", false, true},
		{query.OpThisYear, "This Year", "This year", false, true},
		{query.OpLastYear, "Last Year", "Last year", false, true},
		{query.OpIsEmpty, "Is Empty", "Field is empty", false, false},
		{query.OpIsNotEmpty, "Is Not Empty", "Field is not empty", false, false},
	},
	FieldTypeBoolean: {
		{query.OpEquals, "Is True", "Value is true", false, false},
		{query.OpNotEquals, "Is False", "Value is false", false, false},
	},
	FieldTypeChoice: {
		{query.OpEquals, "Equals", "Equals choice value", true, false},
		{query.OpNotEquals, "Not Equals", "Not equals choice value", true, false},
		{query.OpIn, "In List", "In list of values", true, false},
		{query.OpNotIn, "Not In List", "Not in list of values", true, false},
		{query.OpIsEmpty, "Is Empty", "Field is empty", false, false},
		{query.OpIsNotEmpty, "Is Not Empty", "Field is not empty", false, false},
	},
	FieldTypeReference: {
		{query.OpEquals, "Equals", "References specific record", true, false},
		{query.OpNotEquals, "Not Equals", "Does not reference record", true, false},
		{query.OpIsEmpty, "Is Empty", "Field is empty", false, false},
		{query.OpIsNotEmpty, "Is Not Empty", "Field is not empty", false, false},
	},
}

// Create items for field list
type fieldListItem struct {
	field FieldMetadata
}

func (i fieldListItem) Title() string { 
	return fmt.Sprintf("%s (%s)", i.field.Label, i.field.Name) 
}

func (i fieldListItem) Description() string { 
	typeStr := string(i.field.Type)
	if i.field.Mandatory {
		typeStr += " - Required"
	}
	if i.field.ReadOnly {
		typeStr += " - Read Only"
	}
	return typeStr
}

func (i fieldListItem) FilterValue() string { 
	return i.field.Label + " " + i.field.Name 
}

// Create items for operator list
type operatorListItem struct {
	operator OperatorInfo
}

func (i operatorListItem) Title() string { 
	return i.operator.Label 
}

func (i operatorListItem) Description() string { 
	return i.operator.Description 
}

func (i operatorListItem) FilterValue() string { 
	return i.operator.Label + " " + i.operator.Description 
}

// Create items for logical operator list
type logicalOpListItem struct {
	op    LogicalOperator
	label string
	desc  string
}

func (i logicalOpListItem) Title() string { 
	return i.label 
}

func (i logicalOpListItem) Description() string { 
	return i.desc 
}

func (i logicalOpListItem) FilterValue() string { 
	return i.label + " " + i.desc 
}

// Create items for choice list
type choiceListItem struct {
	choice FieldChoice
}

func (i choiceListItem) Title() string { 
	return fmt.Sprintf("%s (%s)", i.choice.Label, i.choice.Value)
}

func (i choiceListItem) Description() string { 
	return fmt.Sprintf("Value: %s", i.choice.Value)
}

func (i choiceListItem) FilterValue() string { 
	return i.choice.Label + " " + i.choice.Value 
}

// sortFieldsByPriority sorts fields with commonly used ones first
func sortFieldsByPriority(fields []FieldMetadata) []FieldMetadata {
	// Define priority fields in order of importance
	priorityFields := []string{
		"state", "priority", "assigned_to", "assignment_group", "caller_id",
		"number", "short_description", "description", "category", "subcategory",
		"active", "name", "title", "subject", "urgency", "impact",
		"due_date", "work_start", "work_end", "opened_at", "closed_at",
		"sys_created_on", "sys_updated_on", "sys_created_by", "sys_updated_by",
		"comments", "work_notes", "close_notes", "approval",
	}
	
	// Create priority map for fast lookup
	priorityMap := make(map[string]int)
	for i, field := range priorityFields {
		priorityMap[field] = i
	}
	
	// Sort fields
	sortedFields := make([]FieldMetadata, len(fields))
	copy(sortedFields, fields)
	
	// Custom sort function
	for i := 0; i < len(sortedFields)-1; i++ {
		for j := 0; j < len(sortedFields)-i-1; j++ {
			field1 := sortedFields[j]
			field2 := sortedFields[j+1]
			
			priority1, exists1 := priorityMap[field1.Name]
			priority2, exists2 := priorityMap[field2.Name]
			
			shouldSwap := false
			
			if exists1 && exists2 {
				// Both have priority - sort by priority
				shouldSwap = priority1 > priority2
			} else if exists1 && !exists2 {
				// field1 has priority, field2 doesn't - field1 comes first
				shouldSwap = false
			} else if !exists1 && exists2 {
				// field2 has priority, field1 doesn't - field2 comes first
				shouldSwap = true
			} else {
				// Neither has priority - sort alphabetically by label
				label1 := field1.Label
				if label1 == "" {
					label1 = field1.Name
				}
				label2 := field2.Label
				if label2 == "" {
					label2 = field2.Name
				}
				shouldSwap = label1 > label2
			}
			
			if shouldSwap {
				sortedFields[j], sortedFields[j+1] = sortedFields[j+1], sortedFields[j]
			}
		}
	}
	
	return sortedFields
}

// Create items for reference list
type referenceListItem struct {
	sysId       string
	displayName string
	tableName   string
}

func (i referenceListItem) Title() string { 
	return i.displayName
}

func (i referenceListItem) Description() string { 
	return fmt.Sprintf("sys_id: %s", i.sysId)
}

func (i referenceListItem) FilterValue() string { 
	return i.displayName + " " + i.sysId 
}

// NewQueryBuilderModel creates a new query builder
func NewQueryBuilderModel(client *servicenow.Client, tableMetadata *TableFieldMetadata) *QueryBuilderModel {
	// Initialize field list
	fieldItems := make([]list.Item, 0)
	if tableMetadata != nil {
		// Sort fields by priority - commonly used fields first
		sortedFields := sortFieldsByPriority(tableMetadata.Fields)
		for _, field := range sortedFields {
			fieldItems = append(fieldItems, fieldListItem{field: field})
		}
	}
	
	// Create styled delegate for field list
	fieldDelegate := list.NewDefaultDelegate()
	fieldDelegate.Styles.SelectedTitle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("170")).
		Background(lipgloss.Color("57")).
		Bold(true)
	fieldDelegate.Styles.SelectedDesc = lipgloss.NewStyle().
		Foreground(lipgloss.Color("244")).
		Background(lipgloss.Color("57"))
	
	fieldList := list.New(fieldItems, fieldDelegate, 60, 15) // Start with reasonable size
	fieldList.Title = "Select Field"
	fieldList.SetShowStatusBar(false) // We'll handle this in our custom view
	fieldList.SetFilteringEnabled(true)
	fieldList.SetShowHelp(false) // We'll show custom help
	fieldList.Styles.Title = lipgloss.NewStyle().
		Foreground(lipgloss.Color("205")).
		Bold(true)
	fieldList.Styles.FilterPrompt = lipgloss.NewStyle().
		Foreground(lipgloss.Color("244"))
	fieldList.Styles.FilterCursor = lipgloss.NewStyle().
		Foreground(lipgloss.Color("170"))
	
	// Initialize operator list
	operatorDelegate := list.NewDefaultDelegate()
	operatorDelegate.Styles.SelectedTitle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("170")).
		Background(lipgloss.Color("57")).
		Bold(true)
	operatorDelegate.Styles.SelectedDesc = lipgloss.NewStyle().
		Foreground(lipgloss.Color("244")).
		Background(lipgloss.Color("57"))
	
	operatorList := list.New([]list.Item{}, operatorDelegate, 60, 15) // Start with reasonable size
	operatorList.Title = "Select Operator"
	operatorList.SetShowStatusBar(false)
	operatorList.SetFilteringEnabled(false)
	operatorList.Styles.Title = lipgloss.NewStyle().
		Foreground(lipgloss.Color("205")).
		Bold(true)
	
	// Initialize logical operator list
	logicalOpDelegate := list.NewDefaultDelegate()
	logicalOpDelegate.Styles.SelectedTitle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("170")).
		Background(lipgloss.Color("57")).
		Bold(true)
	logicalOpDelegate.Styles.SelectedDesc = lipgloss.NewStyle().
		Foreground(lipgloss.Color("244")).
		Background(lipgloss.Color("57"))
	
	logicalOpItems := []list.Item{
		logicalOpListItem{op: LogicalAnd, label: "AND", desc: "Both conditions must be true"},
		logicalOpListItem{op: LogicalOr, label: "OR", desc: "Either condition can be true"},
	}
	
	logicalOpList := list.New(logicalOpItems, logicalOpDelegate, 60, 8)
	logicalOpList.Title = "How to combine with next condition?"
	logicalOpList.SetShowStatusBar(false)
	logicalOpList.SetFilteringEnabled(false)
	logicalOpList.Styles.Title = lipgloss.NewStyle().
		Foreground(lipgloss.Color("205")).
		Bold(true)
	
	// Initialize choice list
	choiceDelegate := list.NewDefaultDelegate()
	choiceDelegate.Styles.SelectedTitle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("170")).
		Background(lipgloss.Color("57")).
		Bold(true)
	choiceDelegate.Styles.SelectedDesc = lipgloss.NewStyle().
		Foreground(lipgloss.Color("244")).
		Background(lipgloss.Color("57"))
	
	choiceList := list.New([]list.Item{}, choiceDelegate, 60, 15)
	choiceList.Title = "Select Choice Value"
	choiceList.SetShowStatusBar(true)
	choiceList.SetFilteringEnabled(true)
	choiceList.Styles.Title = lipgloss.NewStyle().
		Foreground(lipgloss.Color("205")).
		Bold(true)
	choiceList.Styles.FilterPrompt = lipgloss.NewStyle().
		Foreground(lipgloss.Color("244"))
	choiceList.Styles.FilterCursor = lipgloss.NewStyle().
		Foreground(lipgloss.Color("170"))
	
	// Initialize reference list
	referenceDelegate := list.NewDefaultDelegate()
	referenceDelegate.Styles.SelectedTitle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("170")).
		Background(lipgloss.Color("57")).
		Bold(true)
	referenceDelegate.Styles.SelectedDesc = lipgloss.NewStyle().
		Foreground(lipgloss.Color("244")).
		Background(lipgloss.Color("57"))
	
	referenceList := list.New([]list.Item{}, referenceDelegate, 60, 15)
	referenceList.Title = "Select Reference Record"
	referenceList.SetShowStatusBar(true)
	referenceList.SetFilteringEnabled(true)
	referenceList.Styles.Title = lipgloss.NewStyle().
		Foreground(lipgloss.Color("205")).
		Bold(true)
	referenceList.Styles.FilterPrompt = lipgloss.NewStyle().
		Foreground(lipgloss.Color("244"))
	referenceList.Styles.FilterCursor = lipgloss.NewStyle().
		Foreground(lipgloss.Color("170"))
	
	// Initialize inputs
	valueInput := textinput.New()
	valueInput.Placeholder = "Enter value..."
	valueInput.Width = 40
	
	referenceSearchInput := textinput.New()
	referenceSearchInput.Placeholder = "Type to search..."
	referenceSearchInput.Width = 40
	
	dateStartInput := textinput.New()
	dateStartInput.Placeholder = "YYYY-MM-DD HH:MM:SS"
	dateStartInput.Width = 20
	
	dateEndInput := textinput.New()
	dateEndInput.Placeholder = "YYYY-MM-DD HH:MM:SS"
	dateEndInput.Width = 20
	
	// Initialize calendar components
	calendar := NewCalendarModel()
	dateRangePicker := NewDateRangePickerModel()
	
	return &QueryBuilderModel{
		state:                QueryBuilderStateFieldSelection,
		client:               client,
		tableMetadata:        tableMetadata,
		fieldList:            fieldList,
		operatorList:         operatorList,
		logicalOpList:        logicalOpList,
		choiceList:           choiceList,
		referenceList:        referenceList,
		valueInput:           valueInput,
		referenceSearchInput: referenceSearchInput,
		dateStartInput:       dateStartInput,
		dateEndInput:         dateEndInput,
		calendar:             calendar,
		dateRangePicker:      dateRangePicker,
		useAdvancedDate:      true, // Enable advanced date picker by default
		conditions:           make([]QueryCondition, 0),
		referenceRecords:     make([]map[string]interface{}, 0),
		validator:            NewQueryValidator(tableMetadata),
	}
}

// Init initializes the query builder
func (m *QueryBuilderModel) Init() tea.Cmd {
	return nil
}

// Update handles query builder updates
func (m *QueryBuilderModel) Update(msg tea.Msg) (*QueryBuilderModel, tea.Cmd) {
	var cmd tea.Cmd
	
	if !m.isActive {
		return m, nil
	}
	
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		// Give lists proper size
		listWidth := msg.Width - 4
		listHeight := msg.Height - 8 // Leave room for header and footer
		if listWidth < 40 {
			listWidth = 40
		}
		if listHeight < 10 {
			listHeight = 10
		}
		m.fieldList.SetSize(listWidth, listHeight)
		m.operatorList.SetSize(listWidth, listHeight)
		m.logicalOpList.SetSize(listWidth, min(8, listHeight))
		m.choiceList.SetSize(listWidth, listHeight)
		m.referenceList.SetSize(listWidth, listHeight)
		return m, nil
		
	case tea.KeyMsg:
		// Global key handling for all states
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+v"))):
			// Toggle validation display
			m.showValidation = !m.showValidation
			return m, nil
		case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+r"))):
			// Re-validate current query
			if len(m.conditions) > 0 {
				m.validationResult = &ValidationResult{}
				*m.validationResult = m.validator.ValidateQuery(m.conditions)
				m.showValidation = true
			}
			return m, nil
		case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+c"))):
			// Toggle calendar mode
			m.useAdvancedDate = !m.useAdvancedDate
			return m, nil
		}
		
		switch m.state {
		case QueryBuilderStateFieldSelection:
			switch {
			case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
				if item, ok := m.fieldList.SelectedItem().(fieldListItem); ok {
					m.currentCondition.Field = item.field
					m.setupOperatorList()
					m.state = QueryBuilderStateOperatorSelection
				}
			case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
				m.isActive = false
				return m, nil
			}
			m.fieldList, cmd = m.fieldList.Update(msg)
			return m, cmd
			
		case QueryBuilderStateOperatorSelection:
			switch {
			case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
				if item, ok := m.operatorList.SelectedItem().(operatorListItem); ok {
					m.currentCondition.Operator = item.operator.Operator
					
					if !item.operator.RequiresValue {
						// No value needed, decide if we need logical operator
						m.decideNextStep()
					} else if item.operator.DateOnly {
						// Date range input needed - use advanced picker or simple input
						if m.useAdvancedDate {
							m.state = QueryBuilderStateDateRangePicker
							m.dateRangePicker.SetTimeEnabled(m.currentCondition.Field.Type == FieldTypeDateTime)
							m.dateRangePicker.Activate()
						} else {
							m.state = QueryBuilderStateDateRange
							m.dateStartInput.Focus()
							return m, textinput.Blink
						}
					} else if m.currentCondition.Field.Type == FieldTypeChoice && len(m.currentCondition.Field.Choices) > 0 {
						// Choice field with available choices - show choice list
						m.setupChoiceList()
						m.state = QueryBuilderStateChoiceSelection
					} else if m.currentCondition.Field.Type == FieldTypeReference && m.currentCondition.Field.Reference != "" {
						// Reference field - show reference lookup
						m.state = QueryBuilderStateReferenceSearch
						m.referenceSearchInput.Focus()
						return m, textinput.Blink
					} else if (m.currentCondition.Field.Type == FieldTypeDate || m.currentCondition.Field.Type == FieldTypeDateTime) && m.useAdvancedDate {
						// Date/DateTime field - use calendar picker
						m.state = QueryBuilderStateCalendar
						m.calendar.SetTimeEnabled(m.currentCondition.Field.Type == FieldTypeDateTime)
						m.calendar.Activate()
					} else {
						// Regular value input needed
						m.state = QueryBuilderStateValueInput
						m.setupValueInput()
						m.valueInput.Focus()
						return m, textinput.Blink
					}
				}
			case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
				m.state = QueryBuilderStateFieldSelection
			}
			m.operatorList, cmd = m.operatorList.Update(msg)
			return m, cmd
			
		case QueryBuilderStateValueInput:
			switch {
			case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
				m.currentCondition.Value = m.valueInput.Value()
				m.decideNextStep()
			case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
				m.state = QueryBuilderStateOperatorSelection
				m.valueInput.Blur()
			}
			m.valueInput, cmd = m.valueInput.Update(msg)
			return m, cmd
			
		case QueryBuilderStateChoiceSelection:
			switch {
			case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
				if item, ok := m.choiceList.SelectedItem().(choiceListItem); ok {
					m.currentCondition.Value = item.choice.Value
					m.decideNextStep()
				}
			case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
				m.state = QueryBuilderStateOperatorSelection
			}
			m.choiceList, cmd = m.choiceList.Update(msg)
			return m, cmd
			
		case QueryBuilderStateReferenceSearch:
			switch {
			case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
				if len(m.referenceList.Items()) > 0 {
					if item, ok := m.referenceList.SelectedItem().(referenceListItem); ok {
						m.currentCondition.Value = item.sysId
						m.decideNextStep()
					}
				} else {
					// Use the typed value directly if no search results
					m.currentCondition.Value = m.referenceSearchInput.Value()
					m.decideNextStep()
				}
			case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
				m.state = QueryBuilderStateOperatorSelection
				m.referenceSearchInput.Blur()
			case key.Matches(msg, key.NewBinding(key.WithKeys("tab"))):
				// Switch between search input and results list
				if m.referenceSearchInput.Focused() {
					m.referenceSearchInput.Blur()
					// Focus on list if it has items
					if len(m.referenceList.Items()) > 0 {
						// List will handle its own focus state
					}
				} else {
					m.referenceSearchInput.Focus()
					return m, textinput.Blink
				}
			}
			
			// Handle search input changes
			oldValue := m.referenceSearchInput.Value()
			m.referenceSearchInput, cmd = m.referenceSearchInput.Update(msg)
			newValue := m.referenceSearchInput.Value()
			
			// Perform search if value changed and we have a client
			if oldValue != newValue && len(newValue) >= 2 && m.client != nil {
				return m, m.searchReferenceRecords(newValue)
			}
			
			// Also update the reference list
			if !m.referenceSearchInput.Focused() {
				m.referenceList, cmd = m.referenceList.Update(msg)
			}
			
			return m, cmd
			
		case QueryBuilderStateLogicalOperator:
			switch {
			case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
				if item, ok := m.logicalOpList.SelectedItem().(logicalOpListItem); ok {
					m.currentCondition.LogicalOp = item.op
					m.addCurrentCondition()
					m.resetForNewCondition()
				}
			case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
				// Skip logical operator, just add condition with default AND
				m.currentCondition.LogicalOp = LogicalAnd
				m.addCurrentCondition()
				m.resetForNewCondition()
			}
			m.logicalOpList, cmd = m.logicalOpList.Update(msg)
			return m, cmd
			
		case QueryBuilderStateDateRange:
			switch {
			case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
				if m.focusedInput == 0 {
					// Move to end date
					m.focusedInput = 1
					m.dateStartInput.Blur()
					m.dateEndInput.Focus()
					return m, textinput.Blink
				} else {
					// Both dates entered, process
					if err := m.processDates(); err != nil {
						m.errorMsg = err.Error()
					} else {
						m.decideNextStep()
					}
				}
			case key.Matches(msg, key.NewBinding(key.WithKeys("tab"))):
				if m.focusedInput == 0 {
					m.focusedInput = 1
					m.dateStartInput.Blur()
					m.dateEndInput.Focus()
					return m, textinput.Blink
				} else {
					m.focusedInput = 0
					m.dateEndInput.Blur()
					m.dateStartInput.Focus()
					return m, textinput.Blink
				}
			case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
				m.state = QueryBuilderStateOperatorSelection
				m.dateStartInput.Blur()
				m.dateEndInput.Blur()
				m.focusedInput = 0
			}
			
			if m.focusedInput == 0 {
				m.dateStartInput, cmd = m.dateStartInput.Update(msg)
			} else {
				m.dateEndInput, cmd = m.dateEndInput.Update(msg)
			}
			return m, cmd
			
		case QueryBuilderStateCalendar:
			// Handle escape key to go back to operator selection
			if key.Matches(msg, key.NewBinding(key.WithKeys("esc"))) {
				m.calendar.Deactivate()
				m.state = QueryBuilderStateOperatorSelection
				return m, nil
			}
			
			calendar, cmd := m.calendar.Update(msg)
			m.calendar = calendar
			
			// Check if calendar was deactivated (date selected)
			if !m.calendar.IsActive() {
				selectedDate := m.calendar.GetSelectedDate()
				m.currentCondition.Value = m.calendar.FormatForServiceNow()
				m.currentCondition.StartDate = &selectedDate
				m.decideNextStep()
			}
			return m, cmd
			
		case QueryBuilderStateDateRangePicker:
			// Handle escape key to go back to operator selection
			if key.Matches(msg, key.NewBinding(key.WithKeys("esc"))) {
				m.dateRangePicker.Deactivate()
				m.state = QueryBuilderStateOperatorSelection
				return m, nil
			}
			
			dateRangePicker, cmd := m.dateRangePicker.Update(msg)
			m.dateRangePicker = dateRangePicker
			
			// Check if date range picker was deactivated (range selected)
			if !m.dateRangePicker.IsActive() {
				start, end := m.dateRangePicker.GetDateRange()
				if start != nil && end != nil {
					m.currentCondition.StartDate = start
					m.currentCondition.EndDate = end
					
					// Format for ServiceNow BETWEEN query
					startQuery, endQuery := m.dateRangePicker.FormatForServiceNow()
					m.currentCondition.Value = fmt.Sprintf("javascript:gs.dateGenerate('%s','%s')", startQuery, endQuery)
					
					m.decideNextStep()
				}
			}
			return m, cmd
		}
		
		// Global keys
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+p"))):
			m.showPreview = !m.showPreview
			if m.showPreview {
				m.previewQuery = m.BuildQuery()
			}
		case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+r"))):
			m.resetAll()
		case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+d"))):
			if len(m.conditions) > 0 {
				m.conditions = m.conditions[:len(m.conditions)-1]
			}
		}
		
	case referenceSearchMsg:
		if msg.err != nil {
			m.errorMsg = fmt.Sprintf("Reference search failed: %v", msg.err)
			m.loadingReferences = false
		} else {
			m.processReferenceSearchResults(msg.results)
		}
		return m, nil
	}
	
	return m, nil
}

// setupOperatorList populates the operator list based on the selected field type
func (m *QueryBuilderModel) setupOperatorList() {
	operators, exists := operatorsByFieldType[m.currentCondition.Field.Type]
	if !exists {
		// Default to string operators
		operators = operatorsByFieldType[FieldTypeString]
	}
	
	items := make([]list.Item, len(operators))
	for i, op := range operators {
		items[i] = operatorListItem{operator: op}
	}
	
	m.operatorList.SetItems(items)
}

// setupChoiceList populates the choice list for choice fields
func (m *QueryBuilderModel) setupChoiceList() {
	field := m.currentCondition.Field
	
	items := make([]list.Item, len(field.Choices))
	for i, choice := range field.Choices {
		items[i] = choiceListItem{choice: choice}
	}
	
	m.choiceList.SetItems(items)
}

// setupValueInput configures the value input based on field type
func (m *QueryBuilderModel) setupValueInput() {
	field := m.currentCondition.Field
	
	switch field.Type {
	case FieldTypeChoice:
		if len(field.Choices) > 0 {
			var choiceLabels []string
			for _, choice := range field.Choices {
				choiceLabels = append(choiceLabels, fmt.Sprintf("%s (%s)", choice.Label, choice.Value))
			}
			m.valueInput.Placeholder = fmt.Sprintf("Available: %s", strings.Join(choiceLabels[:min(3, len(choiceLabels))], ", "))
		}
	case FieldTypeBoolean:
		m.valueInput.Placeholder = "true or false"
	case FieldTypeInteger:
		m.valueInput.Placeholder = "Enter number..."
	case FieldTypeReference:
		m.valueInput.Placeholder = fmt.Sprintf("Enter %s sys_id or display value", field.Reference)
	default:
		m.valueInput.Placeholder = "Enter value..."
	}
	
	m.valueInput.SetValue("")
}

// processDates processes the date range inputs
func (m *QueryBuilderModel) processDates() error {
	startStr := strings.TrimSpace(m.dateStartInput.Value())
	endStr := strings.TrimSpace(m.dateEndInput.Value())
	
	if startStr == "" || endStr == "" {
		return fmt.Errorf("both start and end dates are required")
	}
	
	// Try multiple date formats
	formats := []string{
		"2006-01-02 15:04:05",
		"2006-01-02",
		"01/02/2006",
		"01/02/2006 15:04:05",
	}
	
	var startDate, endDate time.Time
	var err error
	
	for _, format := range formats {
		startDate, err = time.Parse(format, startStr)
		if err == nil {
			break
		}
	}
	if err != nil {
		return fmt.Errorf("invalid start date format: %s", startStr)
	}
	
	for _, format := range formats {
		endDate, err = time.Parse(format, endStr)
		if err == nil {
			break
		}
	}
	if err != nil {
		return fmt.Errorf("invalid end date format: %s", endStr)
	}
	
	if endDate.Before(startDate) {
		return fmt.Errorf("end date must be after start date")
	}
	
	m.currentCondition.StartDate = &startDate
	m.currentCondition.EndDate = &endDate
	
	// Format for ServiceNow
	m.currentCondition.Value = fmt.Sprintf("javascript:gs.dateGenerate('%s','%s')", 
		startDate.Format("2006-01-02 15:04:05"), 
		endDate.Format("2006-01-02 15:04:05"))
	
	return nil
}

// decideNextStep determines whether to ask for logical operator or complete the condition
func (m *QueryBuilderModel) decideNextStep() {
	// If this is the first condition, just add it without logical operator
	if len(m.conditions) == 0 {
		m.addCurrentCondition()
		m.resetForNewCondition()
	} else {
		// Ask for logical operator to combine with previous conditions
		m.state = QueryBuilderStateLogicalOperator
	}
}

// addCurrentCondition adds the current condition to the list
func (m *QueryBuilderModel) addCurrentCondition() {
	// Validate the condition before adding
	if validationErrors := m.validator.ValidateCondition(m.currentCondition); len(validationErrors) > 0 {
		// Check for errors (not just warnings)
		hasErrors := false
		for _, err := range validationErrors {
			if err.Severity == SeverityError {
				hasErrors = true
				m.errorMsg = err.Message
				if err.Suggestion != "" {
					m.errorMsg += " (" + err.Suggestion + ")"
				}
				break
			}
		}
		
		// If there are validation errors, don't add the condition
		if hasErrors {
			return
		}
		
		// If only warnings, show them but still add the condition
		if len(validationErrors) > 0 {
			m.errorMsg = "Warning: " + validationErrors[0].Message
		}
	}
	
	m.conditions = append(m.conditions, m.currentCondition)
	
	// Update validation result for the entire query
	m.validationResult = &ValidationResult{}
	*m.validationResult = m.validator.ValidateQuery(m.conditions)
}

// resetForNewCondition resets the builder for a new condition
func (m *QueryBuilderModel) resetForNewCondition() {
	m.currentCondition = QueryCondition{}
	m.state = QueryBuilderStateFieldSelection
	m.valueInput.SetValue("")
	m.valueInput.Blur()
	m.dateStartInput.SetValue("")
	m.dateStartInput.Blur()
	m.dateEndInput.SetValue("")
	m.dateEndInput.Blur()
	m.focusedInput = 0
}

// resetAll resets the entire query builder
func (m *QueryBuilderModel) resetAll() {
	m.conditions = make([]QueryCondition, 0)
	m.resetForNewCondition()
	m.showPreview = false
	m.errorMsg = ""
}

// BuildQuery builds the final ServiceNow query string
func (m *QueryBuilderModel) BuildQuery() string {
	if len(m.conditions) == 0 {
		return ""
	}
	
	qb := query.New()
	
	for i, condition := range m.conditions {
		if i > 0 {
			// Use the logical operator from the previous condition
			if m.conditions[i-1].LogicalOp == LogicalOr {
				qb.Or()
			} else {
				qb.And() // Default to AND
			}
		}
		
		switch condition.Operator {
		case query.OpIsEmpty, query.OpIsNotEmpty:
			qb.Where(condition.Field.Name, condition.Operator, "")
		case query.OpToday, query.OpYesterday, query.OpThisWeek, query.OpLastWeek,
			 query.OpThisMonth, query.OpLastMonth, query.OpThisYear, query.OpLastYear:
			qb.Where(condition.Field.Name, condition.Operator, "")
		default:
			qb.Where(condition.Field.Name, condition.Operator, condition.Value)
		}
	}
	
	return qb.BuildQuery()
}

// SetActive sets the active state of the query builder
func (m *QueryBuilderModel) SetActive(active bool) {
	m.isActive = active
}

// IsActive returns whether the query builder is active
func (m *QueryBuilderModel) IsActive() bool {
	return m.isActive
}

// GetConditions returns the current conditions
func (m *QueryBuilderModel) GetConditions() []QueryCondition {
	return m.conditions
}

// SetConditions sets the conditions (useful for loading saved queries)
func (m *QueryBuilderModel) SetConditions(conditions []QueryCondition) {
	m.conditions = conditions
}

// View renders the query builder
func (m *QueryBuilderModel) View() string {
	if !m.isActive {
		return ""
	}
	
	var content strings.Builder
	
	// Header
	content.WriteString(queryBuilderHeaderStyle.Render("Query Builder"))
	content.WriteString("\n\n")
	
	// Error message
	if m.errorMsg != "" {
		content.WriteString(errorStyle.Render("Error: " + m.errorMsg))
		content.WriteString("\n\n")
	}
	
	// Current conditions
	if len(m.conditions) > 0 {
		content.WriteString(queryBuilderSubtitleStyle.Render("Current Conditions:"))
		content.WriteString("\n")
		for i, condition := range m.conditions {
			conditionStr := fmt.Sprintf("%d. %s %s %s", 
				i+1, 
				condition.Field.Label, 
				condition.Operator, 
				condition.Value)
			if condition.LogicalOp != "" && i < len(m.conditions)-1 {
				conditionStr += fmt.Sprintf(" %s", condition.LogicalOp)
			}
			content.WriteString("  " + conditionStr)
			content.WriteString("\n")
		}
		content.WriteString("\n")
	}
	
	// Current state view
	switch m.state {
	case QueryBuilderStateFieldSelection:
		content.WriteString(queryBuilderSubtitleStyle.Render("Select a Field"))
		content.WriteString("\n\n")
		
		// Show field count and search hint
		fieldCount := len(m.fieldList.Items())
		if fieldCount > 20 {
			content.WriteString(queryBuilderHelpStyle.Render(fmt.Sprintf("%d fields available - use filtering to search", fieldCount)))
		} else {
			content.WriteString(queryBuilderHelpStyle.Render(fmt.Sprintf("%d fields available", fieldCount)))
		}
		content.WriteString("\n\n")
		
		// Custom field list view with better formatting
		listView := m.renderFieldListView()
		content.WriteString(listView)
		
		content.WriteString("\n\n")
		content.WriteString(queryBuilderHelpStyle.Render("Type to filter • Enter to select • Esc to cancel"))
		
	case QueryBuilderStateOperatorSelection:
		content.WriteString(fmt.Sprintf("Field: %s\n\n", m.currentCondition.Field.Label))
		content.WriteString(m.operatorList.View())
		content.WriteString("\n\nPress Enter to select operator, Esc to go back")
		
	case QueryBuilderStateValueInput:
		content.WriteString(fmt.Sprintf("Field: %s\n", m.currentCondition.Field.Label))
		content.WriteString(fmt.Sprintf("Operator: %s\n\n", m.currentCondition.Operator))
		content.WriteString("Value: " + m.valueInput.View())
		content.WriteString("\n\nPress Enter to add condition, Esc to go back")
		
	case QueryBuilderStateChoiceSelection:
		content.WriteString(fmt.Sprintf("Field: %s\n", m.currentCondition.Field.Label))
		content.WriteString(fmt.Sprintf("Operator: %s\n\n", m.currentCondition.Operator))
		content.WriteString(m.choiceList.View())
		content.WriteString("\n\nPress Enter to select choice, Esc to go back")
		
	case QueryBuilderStateReferenceSearch:
		content.WriteString(fmt.Sprintf("Field: %s (references %s)\n", m.currentCondition.Field.Label, m.currentCondition.Field.Reference))
		content.WriteString(fmt.Sprintf("Operator: %s\n\n", m.currentCondition.Operator))
		
		// Search input
		inputStyle := queryBuilderInputStyle
		if m.referenceSearchInput.Focused() {
			inputStyle = queryBuilderFocusedInputStyle
		}
		content.WriteString("Search: " + inputStyle.Render(m.referenceSearchInput.View()))
		content.WriteString("\n\n")
		
		// Loading indicator or results
		if m.loadingReferences {
			content.WriteString("🔍 Searching...")
		} else if len(m.referenceList.Items()) > 0 {
			content.WriteString("Results:\n")
			content.WriteString(m.referenceList.View())
		} else if m.referenceSearchInput.Value() != "" {
			content.WriteString("No results found. Press Enter to use typed value.")
		} else {
			content.WriteString("Type to search for records...")
		}
		
		content.WriteString("\n\nPress Tab to switch between search and results, Enter to select, Esc to go back")
		
	case QueryBuilderStateDateRange:
		content.WriteString(fmt.Sprintf("Field: %s\n", m.currentCondition.Field.Label))
		content.WriteString(fmt.Sprintf("Operator: %s\n\n", m.currentCondition.Operator))
		
		startStyle := queryBuilderInputStyle
		endStyle := queryBuilderInputStyle
		if m.focusedInput == 0 {
			startStyle = queryBuilderFocusedInputStyle
		} else {
			endStyle = queryBuilderFocusedInputStyle
		}
		
		content.WriteString("Start Date: " + startStyle.Render(m.dateStartInput.View()))
		content.WriteString("\n")
		content.WriteString("End Date:   " + endStyle.Render(m.dateEndInput.View()))
		content.WriteString("\n\nPress Tab to switch fields, Enter to confirm, Esc to go back")
		
	case QueryBuilderStateCalendar:
		content.WriteString(fmt.Sprintf("Field: %s\n", m.currentCondition.Field.Label))
		content.WriteString(fmt.Sprintf("Operator: %s\n\n", m.currentCondition.Operator))
		content.WriteString("Select Date")
		if m.currentCondition.Field.Type == FieldTypeDateTime {
			content.WriteString(" and Time")
		}
		content.WriteString(":\n\n")
		content.WriteString(m.calendar.View())
		
	case QueryBuilderStateDateRangePicker:
		content.WriteString(fmt.Sprintf("Field: %s\n", m.currentCondition.Field.Label))
		content.WriteString(fmt.Sprintf("Operator: %s\n\n", m.currentCondition.Operator))
		content.WriteString("Select Date Range")
		if m.currentCondition.Field.Type == FieldTypeDateTime {
			content.WriteString(" with Time")
		}
		content.WriteString(":\n\n")
		content.WriteString(m.dateRangePicker.View())
		
	case QueryBuilderStateLogicalOperator:
		content.WriteString(fmt.Sprintf("Field: %s\n", m.currentCondition.Field.Label))
		content.WriteString(fmt.Sprintf("Operator: %s\n", m.currentCondition.Operator))
		content.WriteString(fmt.Sprintf("Value: %s\n\n", m.currentCondition.Value))
		content.WriteString(m.logicalOpList.View())
		content.WriteString("\n\nPress Enter to select, Esc to use AND")
	}
	
	// Preview
	if m.showPreview && len(m.conditions) > 0 {
		content.WriteString("\n\n")
		content.WriteString(queryBuilderSubtitleStyle.Render("Query Preview:"))
		content.WriteString("\n")
		content.WriteString(codeStyle.Render(m.previewQuery))
	}
	
	// Validation info
	if m.showValidation && m.validationResult != nil {
		content.WriteString("\n\n")
		content.WriteString(queryBuilderSubtitleStyle.Render("Validation Results:"))
		content.WriteString("\n")
		
		if m.validationResult.IsValid {
			content.WriteString(validationSuccessStyle.Render("✓ Query is valid"))
		} else {
			content.WriteString(validationErrorStyle.Render("✗ Query has validation errors"))
		}
		
		// Show validation errors/warnings
		if len(m.validationResult.Errors) > 0 {
			content.WriteString("\n")
			for i, err := range m.validationResult.Errors {
				if i >= 3 { // Limit to first 3 errors to save space
					remaining := len(m.validationResult.Errors) - 3
					content.WriteString(fmt.Sprintf("... and %d more issues\n", remaining))
					break
				}
				
				var prefix string
				var style lipgloss.Style
				switch err.Severity {
				case SeverityError:
					prefix = "❌"
					style = validationErrorStyle
				case SeverityWarning:
					prefix = "⚠️ "
					style = validationWarningStyle
				case SeverityInfo:
					prefix = "ℹ️ "
					style = validationInfoStyle
				}
				
				errorMsg := fmt.Sprintf("%s %s", prefix, err.Message)
				if err.Suggestion != "" {
					errorMsg += fmt.Sprintf(" (%s)", err.Suggestion)
				}
				content.WriteString(style.Render(errorMsg))
				content.WriteString("\n")
			}
		}
	}
	
	// Help
	content.WriteString("\n\n")
	dateMode := "Simple"
	if m.useAdvancedDate {
		dateMode = "Calendar"
	}
	helpText := fmt.Sprintf("Ctrl+P: Preview | Ctrl+V: Validation | Ctrl+C: Date Mode (%s) | Ctrl+R: Reset | Ctrl+D: Delete", dateMode)
	content.WriteString(helpStyle.Render(helpText))
	
	return content.String()
}

// renderFieldListView creates a custom, well-formatted field selection view
func (m *QueryBuilderModel) renderFieldListView() string {
	if len(m.fieldList.Items()) == 0 {
		return queryBuilderHelpStyle.Render("No fields available")
	}
	
	var content strings.Builder
	
	// Show filter input if list is filtering
	if m.fieldList.FilterState() == list.Filtering {
		filterPrompt := "🔍 Filter: " + m.fieldList.FilterValue()
		content.WriteString(queryBuilderFocusedInputStyle.Render(filterPrompt))
		content.WriteString("\n\n")
	} else if m.fieldList.FilterState() == list.FilterApplied {
		filterInfo := fmt.Sprintf("🔍 Filtered by: %s", m.fieldList.FilterValue())
		content.WriteString(queryBuilderInputStyle.Render(filterInfo))
		content.WriteString("\n\n")
	}
	
	// Get visible items
	visibleItems := m.fieldList.VisibleItems()
	if len(visibleItems) == 0 {
		content.WriteString(queryBuilderHelpStyle.Render("No fields match your filter"))
		return content.String()
	}
	
	// Calculate display bounds
	selectedIndex := m.fieldList.Index()
	maxDisplayItems := 12 // Show max 12 items at once
	startIndex := 0
	
	if len(visibleItems) > maxDisplayItems {
		// Calculate scroll position to keep selected item visible
		if selectedIndex >= maxDisplayItems/2 {
			startIndex = selectedIndex - maxDisplayItems/2
			if startIndex+maxDisplayItems > len(visibleItems) {
				startIndex = len(visibleItems) - maxDisplayItems
			}
		}
	}
	
	endIndex := startIndex + maxDisplayItems
	if endIndex > len(visibleItems) {
		endIndex = len(visibleItems)
	}
	
	// Show pagination info if needed
	if len(visibleItems) > maxDisplayItems {
		paginationInfo := fmt.Sprintf("Showing %d-%d of %d fields", startIndex+1, endIndex, len(visibleItems))
		content.WriteString(queryBuilderHelpStyle.Render(paginationInfo))
		content.WriteString("\n\n")
	}
	
	// Render field items
	for i := startIndex; i < endIndex; i++ {
		item := visibleItems[i]
		fieldItem, ok := item.(fieldListItem)
		if !ok {
			continue
		}
		
		field := fieldItem.field
		isSelected := i == selectedIndex
		
		// Format field entry
		var fieldLine strings.Builder
		
		// Add selection indicator
		if isSelected {
			fieldLine.WriteString("▶ ")
		} else {
			fieldLine.WriteString("  ")
		}
		
		// Field name and label
		fieldName := field.Label
		if fieldName == "" {
			fieldName = field.Name
		}
		
		// Add type indicator with emoji
		typeEmoji := getFieldTypeEmoji(field.Type)
		fieldLine.WriteString(fmt.Sprintf("%s %s", typeEmoji, fieldName))
		
		// Add field name in parentheses if different from label
		if field.Label != "" && field.Label != field.Name {
			fieldLine.WriteString(fmt.Sprintf(" (%s)", field.Name))
		}
		
		// Add field attributes
		var attributes []string
		if field.Mandatory {
			attributes = append(attributes, "Required")
		}
		if field.ReadOnly {
			attributes = append(attributes, "Read-only")
		}
		if field.Type == FieldTypeReference && field.Reference != "" {
			attributes = append(attributes, fmt.Sprintf("→ %s", field.Reference))
		}
		if field.Type == FieldTypeChoice && len(field.Choices) > 0 {
			attributes = append(attributes, fmt.Sprintf("%d choices", len(field.Choices)))
		}
		
		if len(attributes) > 0 {
			fieldLine.WriteString(" • " + strings.Join(attributes, " • "))
		}
		
		// Apply styling
		line := fieldLine.String()
		if isSelected {
			line = selectedFieldStyle.Render(line)
		} else {
			line = fieldStyle.Render(line)
		}
		
		content.WriteString(line)
		content.WriteString("\n")
	}
	
	// Show navigation hints if list is long
	if len(visibleItems) > maxDisplayItems {
		content.WriteString("\n")
		content.WriteString(queryBuilderHelpStyle.Render("↑/↓ Navigate • Page Up/Down for faster scrolling"))
	}
	
	return content.String()
}

// getFieldTypeEmoji returns an emoji representing the field type
func getFieldTypeEmoji(fieldType FieldType) string {
	switch fieldType {
	case FieldTypeString, FieldTypeTranslatedText:
		return "📝"
	case FieldTypeInteger, FieldTypeDecimal:
		return "🔢"
	case FieldTypeBoolean:
		return "☑️"
	case FieldTypeDateTime, FieldTypeDate:
		return "📅"
	case FieldTypeReference:
		return "🔗"
	case FieldTypeChoice:
		return "📋"
	case FieldTypeJournal:
		return "📄"
	case FieldTypeHTML:
		return "🌐"
	case FieldTypeURL:
		return "🌍"
	case FieldTypeEmail:
		return "📧"
	case FieldTypePassword:
		return "🔒"
	default:
		return "📋"
	}
}

// Helper function
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Message types for reference search
type referenceSearchMsg struct {
	results []map[string]interface{}
	err     error
}

type referenceSearchStartMsg struct{}

// searchReferenceRecords searches for reference records matching the query
func (m *QueryBuilderModel) searchReferenceRecords(searchTerm string) tea.Cmd {
	if m.client == nil || m.currentCondition.Field.Reference == "" {
		return nil
	}
	
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		
		// Start loading
		m.loadingReferences = true
		
		// Build search query - look for records where common display fields contain the search term
		refTable := m.currentCondition.Field.Reference
		displayFields := []string{"name", "title", "display_name", "short_description", "number", "user_name", "email"}
		
		// Create OR conditions for different display fields
		queryParts := make([]string, 0)
		for _, field := range displayFields {
			queryParts = append(queryParts, fmt.Sprintf("%sCONTAINS%s", field, searchTerm))
		}
		searchQuery := strings.Join(queryParts, "^OR")
		
		params := map[string]string{
			"sysparm_query":         searchQuery,
			"sysparm_limit":         "20", // Limit results for performance
			"sysparm_display_value": "all",
			"sysparm_fields":        "sys_id," + strings.Join(displayFields, ","),
		}
		
		records, err := m.client.Table(refTable).ListWithContext(ctx, params)
		if err != nil {
			return referenceSearchMsg{results: nil, err: err}
		}
		
		return referenceSearchMsg{results: records, err: nil}
	}
}

// processReferenceSearchResults processes the search results and updates the reference list
func (m *QueryBuilderModel) processReferenceSearchResults(results []map[string]interface{}) {
	m.loadingReferences = false
	
	items := make([]list.Item, 0, len(results))
	for _, record := range results {
		sysId := getRecordFieldFromMap(record, "sys_id")
		
		// Try to get the best display name
		displayName := ""
		displayFields := []string{"name", "title", "display_name", "short_description", "number", "user_name", "email"}
		
		for _, field := range displayFields {
			if val := getRecordFieldFromMap(record, field); val != "" {
				displayName = val
				break
			}
		}
		
		if displayName == "" {
			displayName = sysId // Fallback to sys_id
		}
		
		items = append(items, referenceListItem{
			sysId:       sysId,
			displayName: displayName,
			tableName:   m.currentCondition.Field.Reference,
		})
	}
	
	m.referenceList.SetItems(items)
}

// Helper function to extract field value from record map
func getRecordFieldFromMap(record map[string]interface{}, field string) string {
	if val, ok := record[field]; ok && val != nil {
		// Handle ServiceNow reference field objects
		if refMap, isMap := val.(map[string]interface{}); isMap {
			// For reference fields, prefer display_value over value
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

// Local styles for query builder
var (
	queryBuilderHeaderStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205")).
		MarginBottom(1)
		
	queryBuilderSubtitleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("99"))
		
	queryBuilderInputStyle = lipgloss.NewStyle().
		Padding(0, 1).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240"))
		
	queryBuilderFocusedInputStyle = lipgloss.NewStyle().
		Padding(0, 1).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("205"))
		
	// Field selection styles
	fieldStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("252")).
		Padding(0, 1)
		
	selectedFieldStyle = lipgloss.NewStyle().
		Background(lipgloss.Color("57")).
		Foreground(lipgloss.Color("230")).
		Bold(true).
		Padding(0, 1)
		
	queryBuilderHelpStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("244")).
		Italic(true)
		
	// Validation styles
	validationSuccessStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("46")). // Green
		Bold(true)
		
	validationErrorStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("196")). // Red
		Bold(true)
		
	validationWarningStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("214")). // Orange
		Bold(true)
		
	validationInfoStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("39")). // Blue
		Bold(true)
)