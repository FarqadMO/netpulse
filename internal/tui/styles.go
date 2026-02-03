package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	Primary   = lipgloss.Color("205")
	Secondary = lipgloss.Color("86")
	Subtle    = lipgloss.Color("241")
	Success   = lipgloss.Color("46")
	Warning   = lipgloss.Color("214")
	Error     = lipgloss.Color("196")
	
	// Header styles
	HeaderStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("15")).
		Background(Primary).
		Padding(0, 2).
		Align(lipgloss.Center)
	
	// Section styles
	SectionStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(Subtle).
		Padding(1, 2).
		MarginBottom(1)
	
	SectionTitleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(Primary).
		MarginBottom(1)
	
	// Label and value styles
	LabelStyle = lipgloss.NewStyle().
		Foreground(Subtle).
		Width(14)
	
	ValueStyle = lipgloss.NewStyle().
		Foreground(Secondary).
		Bold(true)
	
	// Status styles
	SuccessStyle = lipgloss.NewStyle().
		Foreground(Success)
	
	WarningStyle = lipgloss.NewStyle().
		Foreground(Warning)
	
	ErrorStyle = lipgloss.NewStyle().
		Foreground(Error).
		Bold(true)
	
	// Dim style
	DimStyle = lipgloss.NewStyle().
		Foreground(Subtle).
		Italic(true)
	
	// Help style
	HelpStyle = lipgloss.NewStyle().
		Foreground(Subtle).
		MarginTop(1)
	
	// Loading style
	LoadingStyle = lipgloss.NewStyle().
		Foreground(Primary).
		Padding(2, 4)
	
	// Table styles
	TableHeaderStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("15")).
		Background(Subtle).
		Padding(0, 1)
	
	TableRowStyle = lipgloss.NewStyle().
		Padding(0, 1)
	
	TableRowAltStyle = lipgloss.NewStyle().
		Padding(0, 1).
		Background(lipgloss.Color("236"))
)

// RenderStatus returns a styled status indicator.
func RenderStatus(ok bool, okText, failText string) string {
	if ok {
		return SuccessStyle.Render("✓ " + okText)
	}
	return ErrorStyle.Render("✗ " + failText)
}

// RenderBar renders a progress bar.
func RenderBar(value, max int, width int) string {
	if max == 0 {
		max = 1
	}
	
	filled := int(float64(value) / float64(max) * float64(width))
	if filled > width {
		filled = width
	}
	
	bar := ""
	for i := 0; i < width; i++ {
		if i < filled {
			bar += "█"
		} else {
			bar += "░"
		}
	}
	
	return lipgloss.NewStyle().Foreground(Secondary).Render(bar)
}
