package tui

import (
	"fmt"
	"strings"
	"time"
	
	"github.com/charmbracelet/lipgloss"
)

// Common styles used across TUI components
var (
	titleStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("86")).
		Bold(true).
		Padding(0, 1)

	headerStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("86")).
		Bold(true).
		Border(lipgloss.DoubleBorder(), false, false, true, false).
		Padding(1, 2)

	contentStyle = lipgloss.NewStyle().
		Padding(1, 2)

	errorStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("196")).
		Bold(true).
		Padding(1, 2)

	breadcrumbStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("244")).
		Padding(0, 1)

	footerStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("244")).
		Border(lipgloss.NormalBorder(), true, false, false, false).
		Padding(1, 2)

	// Exported style variables for use in other components
	TitleStyle      = titleStyle
	HeaderStyle     = headerStyle
	ContentStyle    = contentStyle
	ErrorStyle      = errorStyle
	BreadcrumbStyle = breadcrumbStyle
	FooterStyle     = footerStyle

	selectedItemStyle = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(lipgloss.Color("86")).
		Foreground(lipgloss.Color("86")).
		Bold(true)

	selectedItemDescStyle = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(lipgloss.Color("86")).
		Foreground(lipgloss.Color("244"))

	// Loading styles
	loadingStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("86")).
		Bold(true)

	// Info box styles
	infoBoxStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("86")).
		Padding(1, 2).
		Margin(1, 0)

	// Exported info style
	InfoStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("39")).
		Italic(true)

	// Warning styles
	warningStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("214")).
		Bold(true)

	// Success styles
	successStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("46")).
		Bold(true)

	// Highlight styles
	highlightStyle = lipgloss.NewStyle().
		Background(lipgloss.Color("57")).
		Foreground(lipgloss.Color("229")).
		Bold(true).
		Padding(0, 1)

	// Code/monospace styles
	codeStyle = lipgloss.NewStyle().
		Background(lipgloss.Color("235")).
		Foreground(lipgloss.Color("252")).
		Padding(0, 1)

	// Help text styles
	helpStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("244")).
		Italic(true)
)

// Color constants for consistency
const (
	ColorPrimary   = lipgloss.Color("86")  // Green
	ColorSecondary = lipgloss.Color("244") // Gray
	ColorError     = lipgloss.Color("196") // Red
	ColorWarning   = lipgloss.Color("214") // Orange
	ColorSuccess   = lipgloss.Color("46")  // Bright green
	ColorInfo      = lipgloss.Color("39")  // Blue
	ColorHighlight = lipgloss.Color("57")  // Dark blue
)

// Common layout constants
const (
	DefaultPadding = 2
	DefaultMargin  = 1
	DefaultWidth   = 80
	DefaultHeight  = 24
)

// Status indicators
func StatusActive() string {
	return successStyle.Render("●")
}

func StatusInactive() string {
	return errorStyle.Render("●")
}

func StatusUnknown() string {
	return warningStyle.Render("●")
}

// Common UI elements
func RenderBox(title, content string, width int) string {
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorPrimary).
		Width(width).
		Padding(1, 2)

	if title != "" {
		titleBox := lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true).
			Render(title)
		
		return style.Render(titleBox + "\n\n" + content)
	}
	
	return style.Render(content)
}

func RenderKeyHelp(keys []string) string {
	var helpText []string
	for _, key := range keys {
		helpText = append(helpText, helpStyle.Render(key))
	}
	return strings.Join(helpText, " • ")
}

// Progress indicators
func RenderProgress(current, total int) string {
	if total == 0 {
		return "0/0"
	}
	
	percentage := float64(current) / float64(total) * 100
	return fmt.Sprintf("%d/%d (%.1f%%)", current, total, percentage)
}

// Truncate text with ellipsis
func TruncateText(text string, maxLength int) string {
	if len(text) <= maxLength {
		return text
	}
	if maxLength <= 3 {
		return "..."
	}
	return text[:maxLength-3] + "..."
}

// Format file size
func FormatFileSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// Format duration
func FormatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	} else if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	} else if d < time.Hour {
		return fmt.Sprintf("%.1fm", d.Minutes())
	}
	return fmt.Sprintf("%.1fh", d.Hours())
}