package cmd

import (
	_ "embed"
	"strings"

	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/charmbracelet/lipgloss"
)

//go:embed example.yaml
var exampleYAML string

// renderCombineHelp renders the help text for the combine command with lipgloss styling
func renderCombineHelp() string {
	// Define styles
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("12")).
		MarginTop(1)

	sectionStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("10"))

	commandStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("14"))

	commentStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")).
		Italic(true)

	flagStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("11"))

	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(titleStyle.Render("Examples"))
	b.WriteString("\n\n")

	// Simple mode
	b.WriteString(sectionStyle.Render("Simple mode - combine files as individual parts"))
	b.WriteString("\n")
	b.WriteString("  " + commandStyle.Render("go3mf combine file1.scad file2.scad -o output.3mf"))
	b.WriteString("\n\n")

	// Simple mode with names
	b.WriteString(sectionStyle.Render("Simple mode - specify names and filament slots"))
	b.WriteString("\n")
	b.WriteString("  " + commandStyle.Render("go3mf combine file1.scad:part1:1 file2.scad:part2:2 -o output.3mf"))
	b.WriteString("\n\n")

	// Object grouping mode
	b.WriteString(sectionStyle.Render("Object grouping mode - organize parts into objects"))
	b.WriteString("\n")
	b.WriteString("  " + commandStyle.Render("go3mf combine -o output.3mf \\"))
	b.WriteString("\n")
	b.WriteString("    " + commandStyle.Render("--object -n \"Case\" -c 1 bottom.scad top.scad \\"))
	b.WriteString("\n")
	b.WriteString("    " + commandStyle.Render("--object -n \"Inserts\" -c 2 insert.scad"))
	b.WriteString("\n\n")

	// Object mode details
	b.WriteString(sectionStyle.Render("Object mode flags:"))
	b.WriteString("\n")

	// Define flag descriptions
	flags := []struct {
		flag string
		desc string
	}{
		{"--object", "Start new object group"},
		{"-n \"Name\"", "Set object name (required)"},
		{"-c N", "Set filament slot 1-4 for next file (optional)"},
		{"Files", "List of files to include in this object (.stl, .3mf, .scad)"},
	}

	// Calculate max flag width for alignment
	maxWidth := 0
	for _, f := range flags {
		if len(f.flag) > maxWidth {
			maxWidth = len(f.flag)
		}
	}

	// Render flags with proper alignment
	for _, f := range flags {
		padding := strings.Repeat(" ", maxWidth-len(f.flag)+2)
		b.WriteString("  " + flagStyle.Render(f.flag) + padding + commentStyle.Render(f.desc))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	// YAML config mode
	b.WriteString(sectionStyle.Render("YAML config mode"))
	b.WriteString("\n")
	b.WriteString("  " + commandStyle.Render("go3mf combine config.yaml"))
	b.WriteString("\n\n")

	// Add YAML example with syntax highlighting
	b.WriteString(renderYAMLExample(commentStyle, flagStyle, commandStyle))

	return b.String()
}

// renderYAMLExample renders a YAML configuration example with syntax highlighting using Chroma
func renderYAMLExample(commentStyle, keyStyle, valueStyle lipgloss.Style) string {
	lexer := lexers.Get("yaml")
	if lexer == nil {
		lexer = lexers.Fallback
	}

	style := styles.Get("monokai")
	if style == nil {
		style = styles.Fallback
	}

	formatter := formatters.Get("terminal16m")
	if formatter == nil {
		formatter = formatters.Fallback
	}

	var b strings.Builder
	iterator, err := lexer.Tokenise(nil, exampleYAML)
	if err != nil {
		// Fallback to plain text if highlighting fails
		return "  " + strings.ReplaceAll(exampleYAML, "\n", "\n  ")
	}

	err = formatter.Format(&b, style, iterator)
	if err != nil {
		// Fallback to plain text if formatting fails
		return "  " + strings.ReplaceAll(exampleYAML, "\n", "\n  ")
	}

	// Create a bordered style for the YAML example
	borderStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("8")).
		Padding(0, 1).
		MarginLeft(2)

	// Render with border
	return borderStyle.Render(strings.TrimRight(b.String(), "\n"))
}
