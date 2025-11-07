package ui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	// Color palette
	primaryColor   = lipgloss.Color("#7D56F4") // Purple
	secondaryColor = lipgloss.Color("#00D9FF") // Cyan
	successColor   = lipgloss.Color("#04B575") // Green
	errorColor     = lipgloss.Color("#FF5F87") // Pink/Red
	warningColor   = lipgloss.Color("#FFAF00") // Orange
	mutedColor     = lipgloss.Color("#626262") // Gray
	accentColor    = lipgloss.Color("#FFD700") // Gold

	// Title styles
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor).
			MarginTop(1).
			MarginBottom(1).
			PaddingLeft(1)

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(secondaryColor).
			MarginTop(1).
			PaddingLeft(1)

	// Status styles
	successStyle = lipgloss.NewStyle().
			Foreground(successColor).
			Bold(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(errorColor).
			Bold(true)

	warningStyle = lipgloss.NewStyle().
			Foreground(warningColor)

	infoStyle = lipgloss.NewStyle().
			Foreground(mutedColor)

	// Icon styles
	checkmark = lipgloss.NewStyle().
			Foreground(successColor).
			Bold(true).
			SetString("✓")

	cross = lipgloss.NewStyle().
		Foreground(errorColor).
		Bold(true).
		SetString("✗")

	arrow = lipgloss.NewStyle().
		Foreground(secondaryColor).
		SetString("→")

	dot = lipgloss.NewStyle().
		Foreground(mutedColor).
		SetString("•")

	star = lipgloss.NewStyle().
		Foreground(accentColor).
		SetString("★")

	// Item styles
	stepStyle = lipgloss.NewStyle().
			PaddingLeft(2)

	itemStyle = lipgloss.NewStyle().
			PaddingLeft(4).
			Foreground(lipgloss.Color("#FAFAFA"))

	highlightStyle = lipgloss.NewStyle().
			Foreground(accentColor).
			Bold(true)

	// Box style for important info
	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(primaryColor).
			Padding(0, 1).
			MarginTop(1).
			MarginBottom(1)
)

// PrintTitle prints a major title (for app name or major sections)
func PrintTitle(title string) {
	fmt.Println(titleStyle.Render("╭─ " + title + " ─╮"))
}

// PrintHeader prints a section header
func PrintHeader(title string) {
	fmt.Println(headerStyle.Render("\n▸ " + title))
}

// PrintStep prints a step with indentation
func PrintStep(step string) {
	fmt.Println(stepStyle.Render(arrow.String() + " " + step))
}

// PrintItem prints an item in a list
func PrintItem(item string) {
	fmt.Println(itemStyle.Render(dot.String() + " " + item))
}

// PrintSuccess prints a success message
func PrintSuccess(message string) {
	fmt.Println(stepStyle.Render(checkmark.String() + " " + successStyle.Render(message)))
}

// PrintError prints an error message
func PrintError(message string) {
	fmt.Println(stepStyle.Render(cross.String() + " " + errorStyle.Render(message)))
}

// PrintWarning prints a warning message
func PrintWarning(message string) {
	fmt.Println(stepStyle.Render("⚠ " + warningStyle.Render(message)))
}

// PrintInfo prints an info message
func PrintInfo(message string) {
	fmt.Println(stepStyle.Render(infoStyle.Render(message)))
}

// PrintHighlight prints highlighted text
func PrintHighlight(message string) {
	fmt.Println(stepStyle.Render(star.String() + " " + highlightStyle.Render(message)))
}

// PrintBox prints text in a rounded box
func PrintBox(content string) {
	fmt.Println(boxStyle.Render(content))
}

// PrintObjectList prints a list of objects
func PrintObjectList(objects []string) {
	fmt.Println(stepStyle.Render("Objects:"))
	for _, obj := range objects {
		PrintItem(obj)
	}
}

// PrintSeparator prints a visual separator
func PrintSeparator() {
	separator := lipgloss.NewStyle().
		Foreground(mutedColor).
		Render("─────────────────────────────────────────────")
	fmt.Println(separator)
}

// PrintKeyValue prints a key-value pair with nice formatting
func PrintKeyValue(key, value string) {
	keyStyle := lipgloss.NewStyle().
		Foreground(secondaryColor).
		Bold(true)
	fmt.Println(stepStyle.Render(keyStyle.Render(key+":") + " " + value))
}

// PrintTableRow prints a formatted table row with columns
func PrintTableRow(columns ...string) {
	if len(columns) == 0 {
		return
	}
	
	// Define column widths
	widths := []int{30, 15, 20, 30} // Name, ID, Filament, Info
	
	row := ""
	for i, col := range columns {
		if i >= len(widths) {
			break
		}
		
		// Truncate or pad the column
		if len(col) > widths[i] {
			col = col[:widths[i]-3] + "..."
		} else {
			col = col + strings.Repeat(" ", widths[i]-len(col))
		}
		
		row += col
		if i < len(columns)-1 {
			row += " │ "
		}
	}
	
	fmt.Println(stepStyle.Render(row))
}

// PrintTableHeader prints a table header
func PrintTableHeader(headers ...string) {
	headerStyle := lipgloss.NewStyle().
		Foreground(secondaryColor).
		Bold(true)
	
	widths := []int{30, 15, 20, 30}
	row := ""
	
	for i, header := range headers {
		if i >= len(widths) {
			break
		}
		
		if len(header) > widths[i] {
			header = header[:widths[i]]
		} else {
			header = header + strings.Repeat(" ", widths[i]-len(header))
		}
		
		row += header
		if i < len(headers)-1 {
			row += " │ "
		}
	}
	
	fmt.Println(stepStyle.Render(headerStyle.Render(row)))
	
	// Print separator line
	separator := ""
	for i := range headers {
		if i >= len(widths) {
			break
		}
		separator += strings.Repeat("─", widths[i])
		if i < len(headers)-1 {
			separator += "─┼─"
		}
	}
	fmt.Println(stepStyle.Render(infoStyle.Render(separator)))
}

// IsVerbose checks if verbose output is enabled
func IsVerbose() bool {
	// Check for CI environment variable or --progress=plain flag
	if os.Getenv("CI") != "" {
		return true
	}
	// Check for --progress=plain in command line args
	for _, arg := range os.Args {
		if arg == "--progress=plain" {
			return true
		}
	}
	return false
}

// PrintProgress prints a progress indicator
func PrintProgress(current, total int, message string) {
	if IsVerbose() {
		return // Don't print progress in verbose mode
	}
	
	barWidth := 30
	filled := (current * barWidth) / total
	if filled > barWidth {
		filled = barWidth
	}
	
	bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)
	pct := (current * 100) / total
	
	// Use carriage return to overwrite the line
	fmt.Printf("\r  [%s] %d%% %s", bar, pct, message)
	
	// Print newline on completion
	if current >= total {
		fmt.Println()
	}
}
