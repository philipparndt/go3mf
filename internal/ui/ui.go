package ui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)
	infoStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	checkmark    = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).SetString("✓")
	dot          = lipgloss.NewStyle().Foreground(lipgloss.Color("8")).SetString("•")
)

// PrintHeader prints a section header
func PrintHeader(title string) {
	fmt.Println(titleStyle.Render(title))
}

// PrintStep prints a step with indentation
func PrintStep(step string) {
	fmt.Printf("  %s %s\n", dot.String(), step)
}

// PrintSuccess prints a success message
func PrintSuccess(message string) {
	fmt.Printf("%s %s\n", checkmark.String(), successStyle.Render(message))
}

// PrintInfo prints an info message
func PrintInfo(message string) {
	fmt.Printf("  %s\n", infoStyle.Render(message))
}

// PrintObjectList prints a list of objects
func PrintObjectList(objects []string) {
	fmt.Println("  Objects:")
	for _, obj := range objects {
		fmt.Printf("    • %s\n", obj)
	}
}

// PrintError prints an error message
func PrintError(message string) {
	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true)
	fmt.Printf("✗ %s\n", errorStyle.Render(message))
}
