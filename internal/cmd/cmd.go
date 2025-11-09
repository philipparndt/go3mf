package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/user/go3mf/internal/buildplan"
	"github.com/user/go3mf/internal/inspect"
	"github.com/user/go3mf/internal/ui"
	"github.com/user/go3mf/version"
)

type CLI struct {
	Combine *CombineCmd `cmd:"" help:"Combine files into single 3MF (supports YAML, SCAD, 3MF, STL)"`
	Inspect *InspectCmd `cmd:"" help:"Inspect a 3MF file and show its contents"`
	Version *VersionCmd `cmd:"" help:"Show version information"`
}

// AfterApply adds examples to the help output
func (cli *CLI) AfterApply() error {
	return nil
}

type CombineCmd struct {
	Output string   `help:"Output file path (default: combined.3mf)" short:"o"`
	Object bool     `help:"Start a new object group. Follow with: -n NAME [-c FILAMENT] file1 file2... Repeat --object for multiple groups." name:"object"`
	Open   bool     `help:"Open the result file in the default application after combining"`
	Files  []string `arg:"" optional:"" help:"Files to combine. Simple mode: file.scad or file.scad:name:filament. Object mode: use --object flag (see below)."`

	Objects []buildplan.ObjectGroup `kong:"-"` // Parsed object groups
}

// Help adds additional help text with examples
func (c *CombineCmd) Help() string {
	return renderCombineHelp()
}

// openFile opens a file in the default application for the current platform
func openFile(filepath string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", filepath)
	case "linux":
		cmd = exec.Command("xdg-open", filepath)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", "", filepath)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	return cmd.Start()
}

func (c *CombineCmd) Run() error {
	// If --object flag is used anywhere, parse from raw args for better UX
	if c.Object || containsObjectFlag(os.Args) {
		var err error
		c.Objects, err = parseObjectGroupsFromRawArgs(os.Args)
		if err != nil {
			ui.PrintError("Failed to parse object groups: " + err.Error())
			os.Exit(1)
		}
		if len(c.Objects) > 0 {
			c.Files = nil
		}
	}

	// Validate that we have either Files or Objects, but require at least one
	if len(c.Files) == 0 && len(c.Objects) == 0 {
		ui.PrintError("No files or objects specified")
		os.Exit(1)
	}

	// Determine output file if not specified
	outputFile := c.Output
	if outputFile == "" {
		outputFile = "combined.3mf"
	}

	// Create build plan
	planner := buildplan.NewPlanner()
	plan, err := planner.CreatePlan(c.Files, c.Objects, outputFile)
	if err != nil {
		ui.PrintError("Failed to create build plan: " + err.Error())
		os.Exit(1)
	}

	// Execute the plan
	if err := plan.Execute(); err != nil {
		ui.PrintError(err.Error())
		os.Exit(1)
	}

	// Open the file in default application if requested
	if c.Open {
		if err := openFile(outputFile); err != nil {
			ui.PrintError("Failed to open file: " + err.Error())
		}
	}

	return nil
}

// containsObjectFlag checks if --object is present in args
func containsObjectFlag(args []string) bool {
	for _, arg := range args {
		if arg == "--object" {
			return true
		}
	}
	return false
}

// parseObjectGroupsFromRawArgs parses the new flag-based format:
// --object -n "Name" -c 1 file1.scad -c 2 file2.scad --object -n "Next" file3.scad
func parseObjectGroupsFromRawArgs(args []string) ([]buildplan.ObjectGroup, error) {
	var groups []buildplan.ObjectGroup
	var currentGroup *buildplan.ObjectGroup
	var currentFilament int
	inCombineCmd := false
	i := 0

	for i < len(args) {
		arg := args[i]

		// Wait until we're in the combine command
		if arg == "combine" {
			inCombineCmd = true
			i++
			continue
		}

		if !inCombineCmd {
			i++
			continue
		}

		// Skip output flag and its value
		if arg == "-o" || arg == "--output" {
			i += 2
			continue
		}

		// Skip open flag
		if arg == "--open" {
			i++
			continue
		}

		// Start new object group
		if arg == "--object" {
			// Save previous group if exists
			if currentGroup != nil && len(currentGroup.Files) > 0 {
				groups = append(groups, *currentGroup)
			}
			currentGroup = &buildplan.ObjectGroup{
				Name:  "",
				Files: []string{},
			}
			currentFilament = 0
			i++
			continue
		}

		// Parse -n flag (object name)
		if (arg == "-n" || arg == "--name") && currentGroup != nil {
			if i+1 < len(args) {
				currentGroup.Name = args[i+1]
				i += 2
				continue
			}
		}

		// Parse -c flag (filament/color)
		if (arg == "-c" || arg == "--color" || arg == "--filament") && currentGroup != nil {
			if i+1 < len(args) {
				fmt.Sscanf(args[i+1], "%d", &currentFilament)
				i += 2
				continue
			}
		}

		// If we have a current group and this looks like a file, add it
		if currentGroup != nil && isFile(arg) {
			// Format: path or path:name or path:name:filament
			// If filament was set via -c, append it
			fileSpec := arg
			if currentFilament > 0 {
				// Check how many colons are in the arg
				colonCount := strings.Count(arg, ":")
				if colonCount == 0 {
					// Simple path, use filename as name and add filament
					// Will be: path::filament (name will be derived from filename)
					fileSpec = fmt.Sprintf("%s::%d", arg, currentFilament)
				} else if colonCount == 1 {
					// path:name format, add filament
					fileSpec = fmt.Sprintf("%s:%d", arg, currentFilament)
				}
				// If already has 2+ colons, keep as-is (fully specified)
			}
			currentGroup.Files = append(currentGroup.Files, fileSpec)
			currentFilament = 0 // Reset after using
			i++
			continue
		}

		i++
	}

	// Don't forget the last group
	if currentGroup != nil && len(currentGroup.Files) > 0 {
		if currentGroup.Name == "" {
			return nil, fmt.Errorf("object group must have a name (use -n flag)")
		}
		groups = append(groups, *currentGroup)
	}

	return groups, nil
}

// isFile checks if a string looks like a file path
func isFile(s string) bool {
	// Check for file extensions or path indicators
	return strings.HasSuffix(s, ".scad") ||
		strings.HasSuffix(s, ".3mf") ||
		strings.HasSuffix(s, ".stl") ||
		strings.Contains(s, "/") ||
		strings.Contains(s, "\\") ||
		(strings.Contains(s, ".") && !strings.HasPrefix(s, "-"))
}

type InspectCmd struct {
	File string `arg:"" help:"3MF file to inspect"`
}

func (c *InspectCmd) Run() error {
	inspector := inspect.NewInspector()
	return inspector.Inspect(c.File)
}

type VersionCmd struct{}

func (c *VersionCmd) Run() error {
	info := version.Get()
	fmt.Println(info.String())
	return nil
}

// Parse parses command line arguments and executes the appropriate command
func Parse() {
	// Check if we're using the new --object syntax before Kong parses
	if containsObjectFlag(os.Args) {
		// Handle this specially
		if err := parseAndRunWithObjects(); err != nil {
			ui.PrintError(err.Error())
			os.Exit(1)
		}
		return
	}

	// Normal Kong parsing for other cases
	cli := &CLI{}
	ctx := kong.Parse(cli,
		kong.Name("go3mf"),
		kong.Description("3D model file combiner and SCAD renderer"),
		kong.UsageOnError(),
	)
	err := ctx.Run()
	if err != nil {
		ui.PrintError(err.Error())
		os.Exit(1)
	}
}

// parseAndRunWithObjects handles the special --object syntax separately from Kong
func parseAndRunWithObjects() error {
	// Extract output file and open flag
	outputFile := "combined.3mf"
	shouldOpen := false
	for i, arg := range os.Args {
		if (arg == "-o" || arg == "--output") && i+1 < len(os.Args) {
			outputFile = os.Args[i+1]
		}
		if arg == "--open" {
			shouldOpen = true
		}
	}

	// Parse object groups
	groups, err := parseObjectGroupsFromRawArgs(os.Args)
	if err != nil {
		return fmt.Errorf("failed to parse object groups: %w", err)
	}

	if len(groups) == 0 {
		return fmt.Errorf("no objects defined")
	}

	// Create and execute build plan
	planner := buildplan.NewPlanner()
	plan, err := planner.CreatePlan(nil, groups, outputFile)
	if err != nil {
		return fmt.Errorf("failed to create build plan: %w", err)
	}

	if err := plan.Execute(); err != nil {
		return err
	}

	// Open the file in default application if requested
	if shouldOpen {
		if err := openFile(outputFile); err != nil {
			ui.PrintError("Failed to open file: " + err.Error())
		}
	}

	return nil
}
