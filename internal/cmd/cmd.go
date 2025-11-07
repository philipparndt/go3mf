package cmd

import (
	"fmt"
	"os"

	"path/filepath"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/user/go3mf/internal/config"
	"github.com/user/go3mf/internal/models"
	"github.com/user/go3mf/internal/preconditions"
	"github.com/user/go3mf/internal/renderer"
	"github.com/user/go3mf/internal/stl"
	"github.com/user/go3mf/internal/threemf"
	"github.com/user/go3mf/internal/threemf/combine"
	"github.com/user/go3mf/internal/ui"
	"github.com/user/go3mf/version"
)

type CLI struct {
	CombineScad *CombineScadCmd `cmd:"" help:"Render SCAD files and combine into single 3MF"`
	CombineYaml *CombineYamlCmd `cmd:"" help:"Combine files based on YAML configuration"`
	Combine3MF  *Combine3MFCmd  `cmd:"" help:"Combine multiple 3MF files into single model"`
	CombineSTL  *CombineSTLCmd  `cmd:"" help:"Combine multiple STL files into single model"`
	Version     *VersionCmd     `cmd:"" help:"Show version information"`
}

type CombineScadCmd struct {
	Output string   `help:"Output 3MF file path" default:"combined.3mf" short:"o"`
	Files  []string `arg:"" help:"SCAD files to combine. Format: path or path:name" required:""`
}

func (c *CombineScadCmd) Run() error {
	// Check preconditions
	ui.PrintHeader("Checking preconditions...")
	if err := preconditions.Check(); err != nil {
		ui.PrintError("OpenSCAD not found: " + err.Error())
		os.Exit(1)
	}
	ui.PrintSuccess("OpenSCAD is installed")

	// Validate files
	ui.PrintStep("Validating SCAD files...")
	if err := preconditions.ValidateFiles(c.Files); err != nil {
		ui.PrintError(err.Error())
		os.Exit(1)
	}

	// Parse input files
	var scadFiles []models.ScadFile
	for _, arg := range c.Files {
		parts := strings.Split(arg, ":")
		path := parts[0]

		// Convert to absolute path
		absPath, err := filepath.Abs(path)
		if err != nil {
			ui.PrintError(fmt.Sprintf("Invalid file path %s: %v", path, err))
			os.Exit(1)
		}

		// Use custom name if provided
		name := ""
		filamentSlot := 0 // 0 means auto-assign

		if len(parts) > 1 {
			name = parts[1]
		} else {
			// Use filename without extension
			name = filepath.Base(absPath[:len(absPath)-len(filepath.Ext(absPath))])
		}

		// Parse optional filament slot (format: path:name:slot)
		if len(parts) > 2 {
			slot := 0
			_, err := fmt.Sscanf(parts[2], "%d", &slot)
			if err == nil && slot >= 1 && slot <= 4 {
				filamentSlot = slot
			} else {
				ui.PrintError(fmt.Sprintf("Invalid filament slot '%s' for %s. Must be 1-4.", parts[2], path))
				os.Exit(1)
			}
		}

		scadFiles = append(scadFiles, models.ScadFile{
			Path:         absPath,
			Name:         name,
			FilamentSlot: filamentSlot,
		})
	}

	// Render SCAD files
	ui.PrintHeader("Rendering SCAD files...")
	baseDir := filepath.Dir(scadFiles[0].Path)
	var scadPaths []string
	for _, scad := range scadFiles {
		scadPaths = append(scadPaths, scad.Path)
	}

	tempFiles, err := renderer.RenderMultipleSCAD(baseDir, scadPaths)
	if err != nil {
		ui.PrintError(err.Error())
		os.Exit(1)
	}
	defer renderer.CleanupTempFiles(tempFiles)

	for _, scad := range scadFiles {
		ui.PrintStep("Rendered " + scad.Path + " → " + scad.Name)
	}

	// Combine 3MF files
	ui.PrintHeader("Combining 3MF files...")
	combiner := threemf.NewCombiner()
	if err := combiner.Combine(tempFiles, scadFiles, c.Output); err != nil {
		ui.PrintError(err.Error())
		os.Exit(1)
	}

	// Print success
	ui.PrintSuccess("Combined 3MF file created: " + c.Output)
	var names []string
	for _, scad := range scadFiles {
		names = append(names, scad.Name)
	}
	ui.PrintObjectList(names)
	return nil
}

type CombineYamlCmd struct {
	Config string `arg:"" help:"YAML configuration file path" required:""`
}

func (c *CombineYamlCmd) Run() error {
	// Load and validate YAML configuration
	ui.PrintHeader("Loading configuration...")
	loader := config.NewLoader()
	cfg, err := loader.Load(c.Config)
	if err != nil {
		ui.PrintError("Failed to load config: " + err.Error())
		os.Exit(1)
	}
	ui.PrintSuccess(fmt.Sprintf("Loaded configuration with %d object(s)", len(cfg.Objects)))

	// Display configuration summary
	for _, obj := range cfg.Objects {
		ui.PrintStep(fmt.Sprintf("Object '%s' with %d part(s)", obj.Name, len(obj.Parts)))
		for _, part := range obj.Parts {
			filamentInfo := ""
			if part.Filament > 0 {
				filamentInfo = fmt.Sprintf(" [filament %d]", part.Filament)
			}
			ui.PrintStep(fmt.Sprintf("  - %s: %s%s", part.Name, filepath.Base(part.File), filamentInfo))
		}
	}

	// Check preconditions
	ui.PrintHeader("Checking preconditions...")
	if err := preconditions.Check(); err != nil {
		ui.PrintError("OpenSCAD not found: " + err.Error())
		os.Exit(1)
	}
	ui.PrintSuccess("OpenSCAD is installed")

	// Convert YAML config to ScadFile list for rendering
	scadFiles := loader.ConvertToScadFiles(cfg)

	// Validate all files exist
	ui.PrintStep("Validating files...")
	var allPaths []string
	for _, scad := range scadFiles {
		allPaths = append(allPaths, scad.Path)
	}
	if err := preconditions.ValidateFiles(allPaths); err != nil {
		ui.PrintError(err.Error())
		os.Exit(1)
	}

	// Render SCAD files
	ui.PrintHeader("Rendering SCAD files...")
	baseDir := filepath.Dir(scadFiles[0].Path)
	var scadPaths []string
	for _, scad := range scadFiles {
		scadPaths = append(scadPaths, scad.Path)
	}

	tempFiles, err := renderer.RenderMultipleSCAD(baseDir, scadPaths)
	if err != nil {
		ui.PrintError(err.Error())
		os.Exit(1)
	}
	defer renderer.CleanupTempFiles(tempFiles)

	for _, scad := range scadFiles {
		ui.PrintStep("Rendered " + filepath.Base(scad.Path) + " → " + scad.Name)
	}

	// Combine 3MF files
	ui.PrintHeader("Combining 3MF files...")
	combiner := threemf.NewCombiner()
	if err := combiner.CombineWithGroups(tempFiles, scadFiles, cfg.Output); err != nil {
		ui.PrintError(err.Error())
		os.Exit(1)
	}

	// Print success
	ui.PrintSuccess("Combined 3MF file created: " + cfg.Output)

	// Show objects grouped structure
	ui.PrintStep("Objects in model:")
	for _, obj := range cfg.Objects {
		if len(obj.Parts) == 1 {
			ui.PrintStep(fmt.Sprintf("  • %s (1 part)", obj.Name))
		} else {
			ui.PrintStep(fmt.Sprintf("  • %s (%d parts)", obj.Name, len(obj.Parts)))
			for _, part := range obj.Parts {
				ui.PrintStep(fmt.Sprintf("    - %s", part.Name))
			}
		}
	}
	return nil
}

type Combine3MFCmd struct {
	Output string   `help:"Output 3MF file path" default:"combined.3mf" short:"o"`
	Files  []string `arg:"" help:"3MF files to combine" required:""`
}

func (c *Combine3MFCmd) Run() error {
	// Validate files exist
	ui.PrintHeader("Validating 3MF files...")
	for _, file := range c.Files {
		if _, err := os.Stat(file); err != nil {
			ui.PrintError("File not found: " + file)
			os.Exit(1)
		}
		ui.PrintStep("Found " + file)
	}

	// Combine 3MF files
	ui.PrintHeader("Combining 3MF files...")
	combiner := combine.NewCombiner()
	if err := combiner.Combine(c.Files, c.Output); err != nil {
		ui.PrintError(err.Error())
		os.Exit(1)
	}

	ui.PrintSuccess("Combined 3MF file created: " + c.Output)
	return nil
}

type CombineSTLCmd struct {
	Output string   `help:"Output STL file path" default:"combined.stl" short:"o"`
	Files  []string `arg:"" help:"STL files to combine" required:""`
}

func (c *CombineSTLCmd) Run() error {
	// Validate files exist
	ui.PrintHeader("Validating STL files...")
	for _, file := range c.Files {
		if _, err := os.Stat(file); err != nil {
			ui.PrintError("File not found: " + file)
			os.Exit(1)
		}
		ui.PrintStep("Found " + file)
	}

	// Combine STL files
	ui.PrintHeader("Combining STL files...")
	combiner := stl.NewCombiner()
	if err := combiner.Combine(c.Files, c.Output); err != nil {
		ui.PrintError(err.Error())
		os.Exit(1)
	}

	ui.PrintSuccess("Combined STL file created: " + c.Output)
	return nil
}

type VersionCmd struct{}

func (c *VersionCmd) Run() error {
	info := version.Get()
	fmt.Println(info.String())
	return nil
}

// Parse parses command line arguments and executes the appropriate command
func Parse() {
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
