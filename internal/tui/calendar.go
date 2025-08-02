package tui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// CalendarMode represents the calendar display mode
type CalendarMode int

const (
	CalendarModeMonth CalendarMode = iota
	CalendarModeYear
	CalendarModeTime
)

// CalendarModel represents a calendar widget for date/time selection
type CalendarModel struct {
	// Display state
	width, height   int
	currentDate     time.Time  // The date being viewed (for navigation)
	selectedDate    time.Time  // The date that's selected
	mode            CalendarMode
	isActive        bool
	
	// Time components for time mode
	hour    int
	minute  int
	second  int
	focusedTimeComponent int // 0=hour, 1=minute, 2=second
	
	// Navigation
	minDate, maxDate *time.Time
	
	// Settings
	showTime        bool
	use24Hour       bool
	showSeconds     bool
	allowTimeEdit   bool
	
	// Style configuration
	styles CalendarStyles
}

// CalendarStyles holds styling configuration for the calendar
type CalendarStyles struct {
	Header           lipgloss.Style
	DayHeader        lipgloss.Style
	Day              lipgloss.Style
	Today            lipgloss.Style
	Selected         lipgloss.Style
	OutOfMonth       lipgloss.Style
	Weekend          lipgloss.Style
	TimeComponent    lipgloss.Style
	FocusedTime      lipgloss.Style
	Border           lipgloss.Style
	Navigation       lipgloss.Style
}

// NewCalendarModel creates a new calendar model
func NewCalendarModel() *CalendarModel {
	now := time.Now()
	return &CalendarModel{
		currentDate:     now,
		selectedDate:    now,
		mode:           CalendarModeMonth,
		hour:           now.Hour(),
		minute:         now.Minute(),
		second:         now.Second(),
		use24Hour:      true,
		showSeconds:    true,
		allowTimeEdit:  true,
		styles:         defaultCalendarStyles(),
	}
}

// SetSelectedDate sets the selected date
func (m *CalendarModel) SetSelectedDate(date time.Time) {
	m.selectedDate = date
	m.currentDate = date
	m.hour = date.Hour()
	m.minute = date.Minute()
	m.second = date.Second()
}

// GetSelectedDate returns the current selected date with time components
func (m *CalendarModel) GetSelectedDate() time.Time {
	return time.Date(
		m.selectedDate.Year(),
		m.selectedDate.Month(),
		m.selectedDate.Day(),
		m.hour,
		m.minute,
		m.second,
		0,
		m.selectedDate.Location(),
	)
}

// SetDateRange sets the allowed date range
func (m *CalendarModel) SetDateRange(min, max *time.Time) {
	m.minDate = min
	m.maxDate = max
}

// SetTimeEnabled enables or disables time selection
func (m *CalendarModel) SetTimeEnabled(enabled bool) {
	m.showTime = enabled
	if enabled {
		m.allowTimeEdit = true
	}
}

// Activate activates the calendar for input
func (m *CalendarModel) Activate() {
	m.isActive = true
}

// Deactivate deactivates the calendar
func (m *CalendarModel) Deactivate() {
	m.isActive = false
}

// IsActive returns whether the calendar is active
func (m *CalendarModel) IsActive() bool {
	return m.isActive
}

// Init initializes the calendar
func (m *CalendarModel) Init() tea.Cmd {
	return nil
}

// Update handles calendar updates
func (m *CalendarModel) Update(msg tea.Msg) (*CalendarModel, tea.Cmd) {
	if !m.isActive {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil

	case tea.KeyMsg:
		switch m.mode {
		case CalendarModeMonth:
			return m.updateMonthMode(msg)
		case CalendarModeYear:
			return m.updateYearMode(msg)
		case CalendarModeTime:
			return m.updateTimeMode(msg)
		}
	}

	return m, nil
}

// updateMonthMode handles month view navigation
func (m *CalendarModel) updateMonthMode(msg tea.KeyMsg) (*CalendarModel, tea.Cmd) {
	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("left", "h"))):
		m.currentDate = m.currentDate.AddDate(0, 0, -1)
	case key.Matches(msg, key.NewBinding(key.WithKeys("right", "l"))):
		m.currentDate = m.currentDate.AddDate(0, 0, 1)
	case key.Matches(msg, key.NewBinding(key.WithKeys("up", "k"))):
		m.currentDate = m.currentDate.AddDate(0, 0, -7)
	case key.Matches(msg, key.NewBinding(key.WithKeys("down", "j"))):
		m.currentDate = m.currentDate.AddDate(0, 0, 7)
	case key.Matches(msg, key.NewBinding(key.WithKeys("enter", " "))):
		if m.isDateInRange(m.currentDate) {
			m.selectedDate = m.currentDate
			if m.showTime && m.allowTimeEdit {
				m.mode = CalendarModeTime
			}
		}
	case key.Matches(msg, key.NewBinding(key.WithKeys("n"))):
		// Go to today
		now := time.Now()
		m.currentDate = now
		m.selectedDate = now
		m.hour = now.Hour()
		m.minute = now.Minute()
		m.second = now.Second()
	case key.Matches(msg, key.NewBinding(key.WithKeys("m"))):
		// Previous month
		m.currentDate = m.currentDate.AddDate(0, -1, 0)
	case key.Matches(msg, key.NewBinding(key.WithKeys("M"))):
		// Next month
		m.currentDate = m.currentDate.AddDate(0, 1, 0)
	case key.Matches(msg, key.NewBinding(key.WithKeys("y"))):
		// Switch to year mode
		m.mode = CalendarModeYear
	case key.Matches(msg, key.NewBinding(key.WithKeys("t"))):
		// Switch to time mode if time is enabled
		if m.showTime {
			m.mode = CalendarModeTime
		}
	case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
		m.isActive = false
	}
	return m, nil
}

// updateYearMode handles year view navigation
func (m *CalendarModel) updateYearMode(msg tea.KeyMsg) (*CalendarModel, tea.Cmd) {
	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("left", "h"))):
		m.currentDate = m.currentDate.AddDate(-1, 0, 0)
	case key.Matches(msg, key.NewBinding(key.WithKeys("right", "l"))):
		m.currentDate = m.currentDate.AddDate(1, 0, 0)
	case key.Matches(msg, key.NewBinding(key.WithKeys("up", "k"))):
		m.currentDate = m.currentDate.AddDate(-5, 0, 0)
	case key.Matches(msg, key.NewBinding(key.WithKeys("down", "j"))):
		m.currentDate = m.currentDate.AddDate(5, 0, 0)
	case key.Matches(msg, key.NewBinding(key.WithKeys("enter", " "))):
		m.mode = CalendarModeMonth
	case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
		m.mode = CalendarModeMonth
	}
	return m, nil
}

// updateTimeMode handles time selection
func (m *CalendarModel) updateTimeMode(msg tea.KeyMsg) (*CalendarModel, tea.Cmd) {
	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("left", "h"))):
		if m.focusedTimeComponent > 0 {
			m.focusedTimeComponent--
		}
	case key.Matches(msg, key.NewBinding(key.WithKeys("right", "l"))):
		maxComponent := 1 // hour, minute
		if m.showSeconds {
			maxComponent = 2 // hour, minute, second
		}
		if m.focusedTimeComponent < maxComponent {
			m.focusedTimeComponent++
		}
	case key.Matches(msg, key.NewBinding(key.WithKeys("up", "k"))):
		m.incrementTimeComponent(1)
	case key.Matches(msg, key.NewBinding(key.WithKeys("down", "j"))):
		m.incrementTimeComponent(-1)
	case key.Matches(msg, key.NewBinding(key.WithKeys("tab"))):
		maxComponent := 1
		if m.showSeconds {
			maxComponent = 2
		}
		m.focusedTimeComponent = (m.focusedTimeComponent + 1) % (maxComponent + 1)
	case key.Matches(msg, key.NewBinding(key.WithKeys("enter", " "))):
		m.mode = CalendarModeMonth
	case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
		m.mode = CalendarModeMonth
	}
	
	// Handle number key input for direct time entry
	if len(msg.String()) == 1 && msg.String() >= "0" && msg.String() <= "9" {
		if num, err := strconv.Atoi(msg.String()); err == nil {
			m.handleTimeDigitInput(num)
		}
	}
	
	return m, nil
}

// incrementTimeComponent increments/decrements the focused time component
func (m *CalendarModel) incrementTimeComponent(delta int) {
	switch m.focusedTimeComponent {
	case 0: // Hour
		m.hour = (m.hour + delta + 24) % 24
	case 1: // Minute
		m.minute = (m.minute + delta + 60) % 60
	case 2: // Second
		m.second = (m.second + delta + 60) % 60
	}
}

// handleTimeDigitInput handles direct digit input for time components
func (m *CalendarModel) handleTimeDigitInput(digit int) {
	switch m.focusedTimeComponent {
	case 0: // Hour
		newHour := (m.hour%10)*10 + digit
		if newHour < 24 {
			m.hour = newHour
		}
	case 1: // Minute
		newMinute := (m.minute%10)*10 + digit
		if newMinute < 60 {
			m.minute = newMinute
		}
	case 2: // Second
		newSecond := (m.second%10)*10 + digit
		if newSecond < 60 {
			m.second = newSecond
		}
	}
}

// isDateInRange checks if a date is within the allowed range
func (m *CalendarModel) isDateInRange(date time.Time) bool {
	if m.minDate != nil && date.Before(*m.minDate) {
		return false
	}
	if m.maxDate != nil && date.After(*m.maxDate) {
		return false
	}
	return true
}

// View renders the calendar
func (m *CalendarModel) View() string {
	if !m.isActive {
		return ""
	}

	switch m.mode {
	case CalendarModeMonth:
		return m.renderMonthView()
	case CalendarModeYear:
		return m.renderYearView()
	case CalendarModeTime:
		return m.renderTimeView()
	}
	return ""
}

// renderMonthView renders the month calendar view
func (m *CalendarModel) renderMonthView() string {
	var content strings.Builder
	
	// Header with month/year and navigation
	header := fmt.Sprintf("◄ %s %d ►", m.currentDate.Format("January"), m.currentDate.Year())
	content.WriteString(m.styles.Header.Render(header))
	content.WriteString("\n\n")
	
	// Day headers
	dayHeaders := []string{"Su", "Mo", "Tu", "We", "Th", "Fr", "Sa"}
	var dayHeaderRow strings.Builder
	for _, day := range dayHeaders {
		dayHeaderRow.WriteString(m.styles.DayHeader.Render(fmt.Sprintf("%3s", day)))
	}
	content.WriteString(dayHeaderRow.String())
	content.WriteString("\n")
	
	// Calendar grid
	firstOfMonth := time.Date(m.currentDate.Year(), m.currentDate.Month(), 1, 0, 0, 0, 0, m.currentDate.Location())
	lastOfMonth := firstOfMonth.AddDate(0, 1, -1)
	startOfWeek := firstOfMonth.AddDate(0, 0, -int(firstOfMonth.Weekday()))
	
	currentDay := startOfWeek
	today := time.Now()
	
	for week := 0; week < 6; week++ {
		var weekRow strings.Builder
		
		for day := 0; day < 7; day++ {
			dayStyle := m.styles.Day
			dayStr := fmt.Sprintf("%2d", currentDay.Day())
			
			// Apply styling based on date properties
			if currentDay.Month() != m.currentDate.Month() {
				dayStyle = m.styles.OutOfMonth
			} else if currentDay.Weekday() == time.Saturday || currentDay.Weekday() == time.Sunday {
				dayStyle = m.styles.Weekend
			}
			
			// Highlight today
			if currentDay.Year() == today.Year() && currentDay.Month() == today.Month() && currentDay.Day() == today.Day() {
				dayStyle = m.styles.Today
			}
			
			// Highlight selected date
			if currentDay.Year() == m.selectedDate.Year() && currentDay.Month() == m.selectedDate.Month() && currentDay.Day() == m.selectedDate.Day() {
				dayStyle = m.styles.Selected
			}
			
			// Highlight current navigation date
			if currentDay.Year() == m.currentDate.Year() && currentDay.Month() == m.currentDate.Month() && currentDay.Day() == m.currentDate.Day() {
				dayStyle = dayStyle.Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("205"))
			}
			
			// Dim out-of-range dates
			if !m.isDateInRange(currentDay) {
				dayStyle = dayStyle.Foreground(lipgloss.Color("240"))
			}
			
			weekRow.WriteString(dayStyle.Render(fmt.Sprintf("%3s", dayStr)))
			currentDay = currentDay.AddDate(0, 0, 1)
		}
		
		content.WriteString(weekRow.String())
		content.WriteString("\n")
		
		// Stop if we've shown the entire month
		if currentDay.After(lastOfMonth) && week >= 4 {
			break
		}
	}
	
	// Time display if enabled
	if m.showTime {
		content.WriteString("\n")
		timeStr := m.formatTime()
		content.WriteString(m.styles.TimeComponent.Render("Time: " + timeStr))
		content.WriteString("\n")
	}
	
	// Help text
	content.WriteString("\n")
	helpText := "↑↓←→/hjkl: Navigate | Enter/Space: Select | n: Today | m/M: Month | y: Year"
	if m.showTime {
		helpText += " | t: Time"
	}
	helpText += " | Esc: Close"
	content.WriteString(m.styles.Navigation.Render(helpText))
	
	return m.styles.Border.Render(content.String())
}

// renderYearView renders the year selection view
func (m *CalendarModel) renderYearView() string {
	var content strings.Builder
	
	currentYear := m.currentDate.Year()
	startYear := (currentYear/10)*10 - 5
	
	content.WriteString(m.styles.Header.Render(fmt.Sprintf("Select Year (%d-%d)", startYear, startYear+19)))
	content.WriteString("\n\n")
	
	for row := 0; row < 4; row++ {
		var yearRow strings.Builder
		for col := 0; col < 5; col++ {
			year := startYear + row*5 + col
			yearStyle := m.styles.Day
			
			if year == currentYear {
				yearStyle = m.styles.Selected
			}
			
			yearRow.WriteString(yearStyle.Render(fmt.Sprintf("%6d", year)))
		}
		content.WriteString(yearRow.String())
		content.WriteString("\n")
	}
	
	content.WriteString("\n")
	content.WriteString(m.styles.Navigation.Render("↑↓←→/hjkl: Navigate | Enter: Select | Esc: Back"))
	
	return m.styles.Border.Render(content.String())
}

// renderTimeView renders the time selection view
func (m *CalendarModel) renderTimeView() string {
	var content strings.Builder
	
	content.WriteString(m.styles.Header.Render("Set Time"))
	content.WriteString("\n\n")
	
	// Time components
	var timeRow strings.Builder
	
	// Hour
	hourStyle := m.styles.TimeComponent
	if m.focusedTimeComponent == 0 {
		hourStyle = m.styles.FocusedTime
	}
	timeRow.WriteString(hourStyle.Render(fmt.Sprintf("%02d", m.hour)))
	timeRow.WriteString(":")
	
	// Minute
	minuteStyle := m.styles.TimeComponent
	if m.focusedTimeComponent == 1 {
		minuteStyle = m.styles.FocusedTime
	}
	timeRow.WriteString(minuteStyle.Render(fmt.Sprintf("%02d", m.minute)))
	
	// Second (if enabled)
	if m.showSeconds {
		timeRow.WriteString(":")
		secondStyle := m.styles.TimeComponent
		if m.focusedTimeComponent == 2 {
			secondStyle = m.styles.FocusedTime
		}
		timeRow.WriteString(secondStyle.Render(fmt.Sprintf("%02d", m.second)))
	}
	
	content.WriteString(timeRow.String())
	content.WriteString("\n\n")
	
	// Help text
	helpText := "←→/hl: Change component | ↑↓/kj: Adjust value | Tab: Next | 0-9: Direct input | Enter: Done | Esc: Back"
	content.WriteString(m.styles.Navigation.Render(helpText))
	
	return m.styles.Border.Render(content.String())
}

// formatTime formats the current time according to settings
func (m *CalendarModel) formatTime() string {
	if m.use24Hour {
		if m.showSeconds {
			return fmt.Sprintf("%02d:%02d:%02d", m.hour, m.minute, m.second)
		}
		return fmt.Sprintf("%02d:%02d", m.hour, m.minute)
	} else {
		// 12-hour format
		hour12 := m.hour
		ampm := "AM"
		if hour12 >= 12 {
			ampm = "PM"
			if hour12 > 12 {
				hour12 -= 12
			}
		}
		if hour12 == 0 {
			hour12 = 12
		}
		
		if m.showSeconds {
			return fmt.Sprintf("%02d:%02d:%02d %s", hour12, m.minute, m.second, ampm)
		}
		return fmt.Sprintf("%02d:%02d %s", hour12, m.minute, ampm)
	}
}

// FormatSelectedDate formats the selected date for display
func (m *CalendarModel) FormatSelectedDate() string {
	date := m.GetSelectedDate()
	if m.showTime {
		return date.Format("2006-01-02 15:04:05")
	}
	return date.Format("2006-01-02")
}

// FormatForServiceNow formats the selected date for ServiceNow queries
func (m *CalendarModel) FormatForServiceNow() string {
	date := m.GetSelectedDate()
	return date.Format("2006-01-02 15:04:05")
}

// defaultCalendarStyles returns the default calendar styling
func defaultCalendarStyles() CalendarStyles {
	return CalendarStyles{
		Header: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205")).
			Align(lipgloss.Center).
			Width(21),
		DayHeader: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("99")),
		Day: lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")).
			Align(lipgloss.Center),
		Today: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("15")).
			Background(lipgloss.Color("99")).
			Align(lipgloss.Center),
		Selected: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("15")).
			Background(lipgloss.Color("205")).
			Align(lipgloss.Center),
		OutOfMonth: lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Align(lipgloss.Center),
		Weekend: lipgloss.NewStyle().
			Foreground(lipgloss.Color("214")).
			Align(lipgloss.Center),
		TimeComponent: lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")).
			Padding(0, 1).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")),
		FocusedTime: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("15")).
			Background(lipgloss.Color("205")).
			Padding(0, 1).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("205")),
		Border: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(1),
		Navigation: lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")).
			Italic(true),
	}
}