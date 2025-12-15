package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/charmbracelet/huh"
	"github.com/philipparndt/go3mf/internal/buildplan"
	"github.com/philipparndt/go3mf/internal/extract"
	"github.com/philipparndt/go3mf/internal/inspect"
	"github.com/philipparndt/go3mf/internal/ui"
	"github.com/philipparndt/go3mf/version"
)

type CLI struct {
	Combine    *CombineCmd    `cmd:"" help:"Combine files into single 3MF (supports YAML, SCAD, 3MF, STL)"`
	Build      *CombineCmd    `cmd:"" help:"Alias for 'combine' - build files into single 3MF (supports YAML, SCAD, 3MF, STL)" aliases:"build"`
	Init       *InitCmd       `cmd:"" help:"Generate a default YAML configuration file from input files"`
	Inspect    *InspectCmd    `cmd:"" help:"Inspect a 3MF file and show its contents"`
	Extract    *ExtractCmd    `cmd:"" help:"Extract 3D models from a 3MF file as STL files"`
	Version    *VersionCmd    `cmd:"" help:"Show version information"`
	Completion *CompletionCmd `cmd:"" help:"Generate shell completion script"`
}

// AfterApply adds examples to the help output
func (cli *CLI) AfterApply() error {
	return nil
}

type CombineCmd struct {
	Output string   `help:"Output file path (default: combined.3mf)" short:"o"`
	Object bool     `help:"Start a new object group. Follow with: -n NAME [-c FILAMENT] file1 file2... Repeat --object for multiple groups." name:"object"`
	Open   bool     `help:"Open the result file in the default application after combining"`
	Debug  bool     `help:"Enable debug output (verbose mode)"`
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

	// Set debug mode if requested
	buildplan.SetDebug(c.Debug)

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
		if err := openFile(plan.OutputFile); err != nil {
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

		// Wait until we're in the combine or build command
		if arg == "combine" || arg == "build" {
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

		// Skip debug flag
		if arg == "--debug" {
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

type ExtractCmd struct {
	File      string `arg:"" help:"3MF file to extract models from"`
	OutputDir string `help:"Output directory for STL files (default: current directory)" short:"o" default:"."`
	ASCII     bool   `help:"Output ASCII STL files instead of binary" short:"a"`
}

func (c *ExtractCmd) Run() error {
	extractor := extract.NewExtractor()
	return extractor.Extract(c.File, c.OutputDir, !c.ASCII)
}

type InitCmd struct {
	Output string   `help:"Output YAML file path (default: config.yaml)" short:"o" default:"config.yaml"`
	Files  []string `arg:"" help:"Files or glob patterns to include (e.g., *.stl, models/*.scad)"`
}

func (c *InitCmd) Run() error {
	if len(c.Files) == 0 {
		return fmt.Errorf("at least one file or pattern must be specified")
	}

	// Expand glob patterns
	expandedFiles, err := expandGlobPatterns(c.Files)
	if err != nil {
		return fmt.Errorf("error expanding patterns: %w", err)
	}

	if len(expandedFiles) == 0 {
		return fmt.Errorf("no files matched the specified pattern(s)")
	}

	// Check if output file already exists
	if _, err := os.Stat(c.Output); err == nil {
		ui.PrintError(fmt.Sprintf("File %s already exists. Please remove it or specify a different output file with -o", c.Output))
		os.Exit(1)
	}

	ui.PrintTitle("go3mf Init")
	ui.PrintHeader("Configuration Setup")

	ui.PrintInfo(fmt.Sprintf("Creating configuration from %d file(s)", len(expandedFiles)))
	fmt.Println()

	// Ask the user if files should be separate parts or separate objects
	var organizationType string
	err = huh.NewSelect[string]().
		Title("How should the files be organized?").
		Options(
			huh.NewOption("Separate parts (all files in one object)", "parts"),
			huh.NewOption("Separate objects (each file is a separate object)", "objects"),
		).
		Value(&organizationType).
		Run()

	if err != nil {
		return fmt.Errorf("selection cancelled: %w", err)
	}

	var yamlContent string
	if organizationType == "parts" {
		yamlContent = generateSeparatePartsYAML(expandedFiles, c.Output)
	} else {
		yamlContent = generateSeparateObjectsYAML(expandedFiles, c.Output)
	}

	// Write the YAML file
	if err := os.WriteFile(c.Output, []byte(yamlContent), 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	fmt.Println()
	ui.PrintSuccess(fmt.Sprintf("Configuration file created: %s", c.Output))
	fmt.Println()

	ui.PrintHeader("Next Steps")
	ui.PrintStep("Customize your configuration:")
	ui.PrintItem("Part/object names")
	ui.PrintItem("Filament assignments (AMS slots)")
	ui.PrintItem("Packing distance")
	ui.PrintItem("Config files for OpenSCAD variables")
	fmt.Println()

	ui.PrintBox(fmt.Sprintf("go3mf build %s --open", c.Output))

	return nil
}

// generateSeparatePartsYAML generates a YAML config with all files as parts in one object
func generateSeparatePartsYAML(files []string, outputPath string) string {
	var builder strings.Builder

	// Determine output 3MF filename from config filename
	baseOutput := strings.TrimSuffix(outputPath, filepath.Ext(outputPath))
	threemfOutput := baseOutput + ".3mf"

	builder.WriteString("# Generated configuration file\n")
	builder.WriteString("# All files are organized as separate parts within a single object\n")
	builder.WriteString("# Documentation: https://github.com/philipparndt/go3mf\n\n")

	builder.WriteString(fmt.Sprintf("output: %s\n\n", threemfOutput))

	builder.WriteString("# Packing distance between objects in mm (default: 10.0)\n")
	builder.WriteString("# packing_distance: 10.0\n\n")

	builder.WriteString("# Packing algorithm: \"default\" or \"compact\" (default: \"default\")\n")
	builder.WriteString("# packing_algorithm: default\n\n")

	builder.WriteString("objects:\n")
	builder.WriteString("  - name: Combined\n")
	builder.WriteString("    # count: 1  # Number of copies of this object (default: 1)\n")
	builder.WriteString("    # normalize_position: true  # Place object at ground level (default: true)\n")
	builder.WriteString("    # config:  # OpenSCAD config applied to all parts\n")
	builder.WriteString("    #   - config.scad:\n")
	builder.WriteString("    #       variable_name: value\n")
	builder.WriteString("    parts:\n")

	for i, file := range files {
		partName := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
		builder.WriteString(fmt.Sprintf("      - name: %s\n", partName))
		builder.WriteString(fmt.Sprintf("        file: %s\n", file))

		// Add all optional fields as comments
		builder.WriteString("        # filament: 1  # AMS slot (1-4), 0 or omit for auto\n")
		builder.WriteString("        # rotation_x: 0  # Rotation around X axis in degrees\n")
		builder.WriteString("        # rotation_y: 0  # Rotation around Y axis in degrees\n")
		builder.WriteString("        # rotation_z: 0  # Rotation around Z axis in degrees\n")
		builder.WriteString("        # position_x: 0  # Relative X position offset in mm\n")
		builder.WriteString("        # position_y: 0  # Relative Y position offset in mm\n")
		builder.WriteString("        # position_z: 0  # Relative Z position offset in mm\n")
		builder.WriteString("        # config:  # Part-specific OpenSCAD config (overrides object config)\n")
		builder.WriteString("        #   - config.scad:\n")
		builder.WriteString("        #       variable_name: value\n")

		if i < len(files)-1 {
			builder.WriteString("\n")
		}
	}

	return builder.String()
}

// generateSeparateObjectsYAML generates a YAML config with each file as a separate object
func generateSeparateObjectsYAML(files []string, outputPath string) string {
	var builder strings.Builder

	// Determine output 3MF filename from config filename
	baseOutput := strings.TrimSuffix(outputPath, filepath.Ext(outputPath))
	threemfOutput := baseOutput + ".3mf"

	builder.WriteString("# Generated configuration file\n")
	builder.WriteString("# Each file is organized as a separate object\n")
	builder.WriteString("# Documentation: https://github.com/philipparndt/go3mf\n\n")

	builder.WriteString(fmt.Sprintf("output: %s\n\n", threemfOutput))

	builder.WriteString("# Packing distance between objects in mm (default: 10.0)\n")
	builder.WriteString("# packing_distance: 10.0\n\n")

	builder.WriteString("# Packing algorithm: \"default\" or \"compact\" (default: \"default\")\n")
	builder.WriteString("# packing_algorithm: default\n\n")

	builder.WriteString("objects:\n")

	for i, file := range files {
		objectName := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
		builder.WriteString(fmt.Sprintf("  - name: %s\n", objectName))
		builder.WriteString("    # count: 1  # Number of copies of this object (default: 1)\n")
		builder.WriteString("    # normalize_position: true  # Place object at ground level (default: true)\n")
		builder.WriteString("    # config:  # OpenSCAD config applied to all parts\n")
		builder.WriteString("    #   - config.scad:\n")
		builder.WriteString("    #       variable_name: value\n")
		builder.WriteString("    parts:\n")
		builder.WriteString("      - name: main\n")
		builder.WriteString(fmt.Sprintf("        file: %s\n", file))
		builder.WriteString("        # filament: 1  # AMS slot (1-4), 0 or omit for auto\n")
		builder.WriteString("        # rotation_x: 0  # Rotation around X axis in degrees\n")
		builder.WriteString("        # rotation_y: 0  # Rotation around Y axis in degrees\n")
		builder.WriteString("        # rotation_z: 0  # Rotation around Z axis in degrees\n")
		builder.WriteString("        # position_x: 0  # Relative X position offset in mm\n")
		builder.WriteString("        # position_y: 0  # Relative Y position offset in mm\n")
		builder.WriteString("        # position_z: 0  # Relative Z position offset in mm\n")
		builder.WriteString("        # config:  # Part-specific OpenSCAD config (overrides object config)\n")
		builder.WriteString("        #   - config.scad:\n")
		builder.WriteString("        #       variable_name: value\n")

		if i < len(files)-1 {
			builder.WriteString("\n")
		}
	}

	return builder.String()
}

// expandGlobPatterns expands glob patterns in the file list
func expandGlobPatterns(patterns []string) ([]string, error) {
	var result []string
	seen := make(map[string]bool)

	for _, pattern := range patterns {
		// Try to expand as glob pattern
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid pattern %q: %w", pattern, err)
		}

		if len(matches) == 0 {
			// No matches - treat as literal filename (will fail later if doesn't exist)
			if !seen[pattern] {
				result = append(result, pattern)
				seen[pattern] = true
			}
		} else {
			// Add all matches, avoiding duplicates
			for _, match := range matches {
				if !seen[match] {
					result = append(result, match)
					seen[match] = true
				}
			}
		}
	}

	return result, nil
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
		// Debug flag is handled globally by IsVerbose(), no need to parse here
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
		if err := openFile(plan.OutputFile); err != nil {
			ui.PrintError("Failed to open file: " + err.Error())
		}
	}

	return nil
}
