package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// DateRangePickerMode represents the picker mode
type DateRangePickerMode int

const (
	DateRangePickerModeStartDate DateRangePickerMode = iota
	DateRangePickerModeEndDate
	DateRangePickerModePresets
)

// DateRangePreset represents a predefined date range
type DateRangePreset struct {
	Name      string
	StartDate time.Time
	EndDate   time.Time
}

// DateRangePickerModel handles date range selection with calendar widgets
type DateRangePickerModel struct {
	// Display state
	width, height int
	mode          DateRangePickerMode
	isActive      bool
	
	// Calendar widgets
	startCalendar *CalendarModel
	endCalendar   *CalendarModel
	
	// Date range
	startDate *time.Time
	endDate   *time.Time
	
	// Presets
	presets         []DateRangePreset
	selectedPreset  int
	showPresets     bool
	
	// Settings
	showTime        bool
	enforceOrder    bool // Ensure end date is after start date
	allowSameDay    bool
	
	// Styles
	styles DateRangePickerStyles
}

// DateRangePickerStyles holds styling for the date range picker
type DateRangePickerStyles struct {
	Header          lipgloss.Style
	SubHeader       lipgloss.Style
	ActivePanel     lipgloss.Style
	InactivePanel   lipgloss.Style
	PresetItem      lipgloss.Style
	SelectedPreset  lipgloss.Style
	DateRange       lipgloss.Style
	Help            lipgloss.Style
	Error           lipgloss.Style
}

// NewDateRangePickerModel creates a new date range picker
func NewDateRangePickerModel() *DateRangePickerModel {
	startCal := NewCalendarModel()
	endCal := NewCalendarModel()
	
	// Set default time to start/end of day
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endOfDay := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, now.Location())
	
	startCal.SetSelectedDate(startOfDay)
	endCal.SetSelectedDate(endOfDay)
	
	return &DateRangePickerModel{
		startCalendar:  startCal,
		endCalendar:    endCal,
		mode:          DateRangePickerModeStartDate,
		enforceOrder:  true,
		allowSameDay:  true,
		showPresets:   true,
		presets:       getDefaultDatePresets(),
		styles:        defaultDateRangePickerStyles(),
	}
}

// SetTimeEnabled enables or disables time selection
func (m *DateRangePickerModel) SetTimeEnabled(enabled bool) {
	m.showTime = enabled
	m.startCalendar.SetTimeEnabled(enabled)
	m.endCalendar.SetTimeEnabled(enabled)
}

// SetDateRange sets the current date range
func (m *DateRangePickerModel) SetDateRange(start, end *time.Time) {
	m.startDate = start
	m.endDate = end
	
	if start != nil {
		m.startCalendar.SetSelectedDate(*start)
	}
	if end != nil {
		m.endCalendar.SetSelectedDate(*end)
	}
}

// GetDateRange returns the current date range
func (m *DateRangePickerModel) GetDateRange() (start, end *time.Time) {
	startDate := m.startCalendar.GetSelectedDate()
	endDate := m.endCalendar.GetSelectedDate()
	return &startDate, &endDate
}

// Activate activates the date range picker
func (m *DateRangePickerModel) Activate() {
	m.isActive = true
	if m.mode == DateRangePickerModeStartDate {
		m.startCalendar.Activate()
	}
}

// Deactivate deactivates the date range picker
func (m *DateRangePickerModel) Deactivate() {
	m.isActive = false
	m.startCalendar.Deactivate()
	m.endCalendar.Deactivate()
}

// IsActive returns whether the picker is active
func (m *DateRangePickerModel) IsActive() bool {
	return m.isActive
}

// Init initializes the date range picker
func (m *DateRangePickerModel) Init() tea.Cmd {
	return tea.Batch(m.startCalendar.Init(), m.endCalendar.Init())
}

// Update handles date range picker updates
func (m *DateRangePickerModel) Update(msg tea.Msg) (*DateRangePickerModel, tea.Cmd) {
	if !m.isActive {
		return m, nil
	}

	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		startCal, cmd1 := m.startCalendar.Update(msg)
		m.startCalendar = startCal
		cmds = append(cmds, cmd1)
		
		endCal, cmd2 := m.endCalendar.Update(msg)
		m.endCalendar = endCal
		cmds = append(cmds, cmd2)
		
		return m, tea.Batch(cmds...)

	case tea.KeyMsg:
		// Global keys
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("tab"))):
			return m.switchMode(), nil
		case key.Matches(msg, key.NewBinding(key.WithKeys("p"))):
			if m.showPresets {
				m.mode = DateRangePickerModePresets
				m.startCalendar.Deactivate()
				m.endCalendar.Deactivate()
			}
			return m, nil
		case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
			if m.mode == DateRangePickerModePresets {
				m.switchToDateMode()
				return m, nil
			}
			m.isActive = false
			return m, nil
		}

		// Mode-specific handling
		switch m.mode {
		case DateRangePickerModeStartDate:
			startCal, cmd := m.startCalendar.Update(msg)
			m.startCalendar = startCal
			cmds = append(cmds, cmd)
			
			// Auto-switch to end date when start date is selected and calendar deactivates
			if !m.startCalendar.IsActive() && m.startCalendar.IsActive() != startCal.IsActive() {
				m.mode = DateRangePickerModeEndDate
				m.endCalendar.Activate()
			}

		case DateRangePickerModeEndDate:
			endCal, cmd := m.endCalendar.Update(msg)
			m.endCalendar = endCal
			cmds = append(cmds, cmd)
			
			// Auto-complete when end date is selected
			if !m.endCalendar.IsActive() && m.endCalendar.IsActive() != endCal.IsActive() {
				if m.validateDateRange() {
					m.isActive = false
				} else {
					// Invalid range, stay active
					m.endCalendar.Activate()
				}
			}

		case DateRangePickerModePresets:
			return m.handlePresetMode(msg), nil
		}
	}

	return m, tea.Batch(cmds...)
}

// switchMode switches between start and end date modes
func (m *DateRangePickerModel) switchMode() *DateRangePickerModel {
	switch m.mode {
	case DateRangePickerModeStartDate:
		m.mode = DateRangePickerModeEndDate
		m.startCalendar.Deactivate()
		m.endCalendar.Activate()
	case DateRangePickerModeEndDate:
		m.mode = DateRangePickerModeStartDate
		m.endCalendar.Deactivate()
		m.startCalendar.Activate()
	case DateRangePickerModePresets:
		m.switchToDateMode()
	}
	return m
}

// switchToDateMode switches to date selection mode
func (m *DateRangePickerModel) switchToDateMode() {
	if m.mode == DateRangePickerModeStartDate || m.startDate == nil {
		m.mode = DateRangePickerModeStartDate
		m.startCalendar.Activate()
		m.endCalendar.Deactivate()
	} else {
		m.mode = DateRangePickerModeEndDate
		m.endCalendar.Activate()
		m.startCalendar.Deactivate()
	}
}

// handlePresetMode handles preset selection
func (m *DateRangePickerModel) handlePresetMode(msg tea.KeyMsg) *DateRangePickerModel {
	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("up", "k"))):
		if m.selectedPreset > 0 {
			m.selectedPreset--
		}
	case key.Matches(msg, key.NewBinding(key.WithKeys("down", "j"))):
		if m.selectedPreset < len(m.presets)-1 {
			m.selectedPreset++
		}
	case key.Matches(msg, key.NewBinding(key.WithKeys("enter", " "))):
		preset := m.presets[m.selectedPreset]
		m.startCalendar.SetSelectedDate(preset.StartDate)
		m.endCalendar.SetSelectedDate(preset.EndDate)
		m.startDate = &preset.StartDate
		m.endDate = &preset.EndDate
		m.isActive = false
	}
	return m
}

// validateDateRange validates the current date range
func (m *DateRangePickerModel) validateDateRange() bool {
	startDate := m.startCalendar.GetSelectedDate()
	endDate := m.endCalendar.GetSelectedDate()
	
	if m.enforceOrder {
		if endDate.Before(startDate) {
			return false
		}
		if !m.allowSameDay && startDate.Year() == endDate.Year() && 
		   startDate.Month() == endDate.Month() && startDate.Day() == endDate.Day() {
			return false
		}
	}
	
	return true
}

// View renders the date range picker
func (m *DateRangePickerModel) View() string {
	if !m.isActive {
		return ""
	}

	var content strings.Builder
	
	// Header
	content.WriteString(m.styles.Header.Render("Date Range Selection"))
	content.WriteString("\n\n")
	
	// Current range display
	startStr := "Not selected"
	endStr := "Not selected"
	
	if m.startDate != nil || m.startCalendar.IsActive() {
		startStr = m.startCalendar.FormatSelectedDate()
	}
	if m.endDate != nil || m.endCalendar.IsActive() {
		endStr = m.endCalendar.FormatSelectedDate()
	}
	
	rangeStr := fmt.Sprintf("From: %s  To: %s", startStr, endStr)
	content.WriteString(m.styles.DateRange.Render(rangeStr))
	content.WriteString("\n\n")
	
	// Mode-specific content
	switch m.mode {
	case DateRangePickerModeStartDate:
		content.WriteString(m.styles.SubHeader.Render("Select Start Date"))
		content.WriteString("\n")
		
		panelStyle := m.styles.ActivePanel
		content.WriteString(panelStyle.Render(m.startCalendar.View()))
		
	case DateRangePickerModeEndDate:
		content.WriteString(m.styles.SubHeader.Render("Select End Date"))
		content.WriteString("\n")
		
		panelStyle := m.styles.ActivePanel
		content.WriteString(panelStyle.Render(m.endCalendar.View()))
		
	case DateRangePickerModePresets:
		content.WriteString(m.styles.SubHeader.Render("Quick Presets"))
		content.WriteString("\n\n")
		
		for i, preset := range m.presets {
			style := m.styles.PresetItem
			if i == m.selectedPreset {
				style = m.styles.SelectedPreset
			}
			
			presetText := fmt.Sprintf("%s (%s - %s)", 
				preset.Name,
				preset.StartDate.Format("Jan 2"),
				preset.EndDate.Format("Jan 2"))
			content.WriteString(style.Render(presetText))
			content.WriteString("\n")
		}
	}
	
	// Validation error
	if !m.validateDateRange() && (m.startDate != nil && m.endDate != nil) {
		content.WriteString("\n")
		errorMsg := "Invalid date range: End date must be after start date"
		if !m.allowSameDay {
			errorMsg = "Invalid date range: End date must be after start date (same day not allowed)"
		}
		content.WriteString(m.styles.Error.Render(errorMsg))
	}
	
	// Help text
	content.WriteString("\n\n")
	var helpText string
	switch m.mode {
	case DateRangePickerModePresets:
		helpText = "↑↓/kj: Navigate | Enter: Select | Esc: Back to calendar"
	default:
		helpText = "Tab: Switch start/end | p: Presets | Esc: Close"
	}
	content.WriteString(m.styles.Help.Render(helpText))
	
	return content.String()
}

// FormatForServiceNow formats the date range for ServiceNow queries
func (m *DateRangePickerModel) FormatForServiceNow() (startQuery, endQuery string) {
	startDate := m.startCalendar.GetSelectedDate()
	endDate := m.endCalendar.GetSelectedDate()
	
	startQuery = startDate.Format("2006-01-02 15:04:05")
	endQuery = endDate.Format("2006-01-02 15:04:05")
	
	return startQuery, endQuery
}

// GetFormattedRange returns a human-readable date range string
func (m *DateRangePickerModel) GetFormattedRange() string {
	start, end := m.GetDateRange()
	if start == nil || end == nil {
		return "No date range selected"
	}
	
	if m.showTime {
		return fmt.Sprintf("%s - %s", 
			start.Format("2006-01-02 15:04:05"),
			end.Format("2006-01-02 15:04:05"))
	}
	return fmt.Sprintf("%s - %s", 
		start.Format("2006-01-02"),
		end.Format("2006-01-02"))
}

// getDefaultDatePresets returns common date range presets
func getDefaultDatePresets() []DateRangePreset {
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endOfDay := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, now.Location())
	
	return []DateRangePreset{
		{
			Name:      "Today",
			StartDate: startOfDay,
			EndDate:   endOfDay,
		},
		{
			Name:      "Yesterday",
			StartDate: startOfDay.AddDate(0, 0, -1),
			EndDate:   startOfDay.AddDate(0, 0, -1).Add(time.Hour*23 + time.Minute*59 + time.Second*59),
		},
		{
			Name:      "Last 7 days",
			StartDate: startOfDay.AddDate(0, 0, -7),
			EndDate:   endOfDay,
		},
		{
			Name:      "Last 30 days",
			StartDate: startOfDay.AddDate(0, 0, -30),
			EndDate:   endOfDay,
		},
		{
			Name:      "This week",
			StartDate: startOfWeek(now),
			EndDate:   endOfDay,
		},
		{
			Name:      "This month",
			StartDate: time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()),
			EndDate:   endOfDay,
		},
		{
			Name:      "Last month",
			StartDate: time.Date(now.Year(), now.Month()-1, 1, 0, 0, 0, 0, now.Location()),
			EndDate:   time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).AddDate(0, 0, -1).Add(time.Hour*23 + time.Minute*59 + time.Second*59),
		},
		{
			Name:      "This year",
			StartDate: time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location()),
			EndDate:   endOfDay,
		},
	}
}

// startOfWeek returns the start of the week (Sunday) for the given date
func startOfWeek(date time.Time) time.Time {
	weekday := int(date.Weekday())
	return time.Date(date.Year(), date.Month(), date.Day()-weekday, 0, 0, 0, 0, date.Location())
}

// defaultDateRangePickerStyles returns default styling
func defaultDateRangePickerStyles() DateRangePickerStyles {
	return DateRangePickerStyles{
		Header: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205")).
			Align(lipgloss.Center).
			Width(50),
		SubHeader: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("99")),
		ActivePanel: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("205")).
			Padding(0, 1),
		InactivePanel: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1),
		PresetItem: lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")).
			Padding(0, 2),
		SelectedPreset: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("15")).
			Background(lipgloss.Color("205")).
			Padding(0, 2),
		DateRange: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("39")).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("39")).
			Padding(0, 1).
			Align(lipgloss.Center),
		Help: lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")).
			Italic(true),
		Error: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("196")),
	}
}