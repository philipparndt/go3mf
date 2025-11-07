package buildplan

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/user/go3mf/internal/config"
	"github.com/user/go3mf/internal/inspect"
	"github.com/user/go3mf/internal/models"
	"github.com/user/go3mf/internal/preconditions"
	"github.com/user/go3mf/internal/renderer"
	"github.com/user/go3mf/internal/stl"
	"github.com/user/go3mf/internal/threemf"
	"github.com/user/go3mf/internal/threemf/combine"
	"github.com/user/go3mf/internal/ui"
)

// FileType represents the type of input file
type FileType int

const (
	FileTypeUnknown FileType = iota
	FileTypeYAML
	FileTypeSCAD
	FileType3MF
	FileTypeSTL
)

// ObjectGroup represents a group of files belonging to the same object
type ObjectGroup struct {
	Name  string
	Files []string
}

// BuildStep represents a single step in the build plan
type BuildStep interface {
	Name() string
	Execute() error
}

// BuildPlan contains all steps needed to process and combine files
type BuildPlan struct {
	Steps      []BuildStep
	OutputFile string
}

// Planner creates build plans based on input files
type Planner struct{}

// NewPlanner creates a new build planner
func NewPlanner() *Planner {
	return &Planner{}
}

// CreatePlan analyzes input files and creates an execution plan
func (p *Planner) CreatePlan(inputs []string, objects []ObjectGroup, outputFile string) (*BuildPlan, error) {
	// If objects are specified via --object flags, create YAML-style plan
	if len(objects) > 0 {
		return p.createObjectGroupPlan(objects, outputFile)
	}

	// If single input is a YAML file, use YAML-based plan
	if len(inputs) == 1 && detectFileType(inputs[0]) == FileTypeYAML {
		return p.createYAMLPlan(inputs[0])
	}

	// Otherwise, detect file types and create appropriate plan
	fileTypes := make(map[FileType][]string)
	for _, input := range inputs {
		ft := detectFileType(input)
		if ft == FileTypeUnknown {
			return nil, fmt.Errorf("unknown file type: %s", input)
		}
		fileTypes[ft] = append(fileTypes[ft], input)
	}

	// Check if all files are the same type
	if len(fileTypes) > 1 {
		return nil, fmt.Errorf("cannot mix different file types (found: %v)", getFileTypeNames(fileTypes))
	}

	// Get the single file type
	var fileType FileType
	var files []string
	for ft, fs := range fileTypes {
		fileType = ft
		files = fs
	}

	switch fileType {
	case FileTypeSCAD:
		return p.createSCADPlan(files, outputFile)
	case FileType3MF:
		return p.create3MFPlan(files, outputFile)
	case FileTypeSTL:
		return p.createSTLPlan(files, outputFile)
	default:
		return nil, fmt.Errorf("unsupported file type")
	}
}

// createYAMLPlan creates a plan for YAML configuration file
func (p *Planner) createYAMLPlan(yamlFile string) (*BuildPlan, error) {
	plan := &BuildPlan{}

	// Step 1: Load YAML configuration
	plan.Steps = append(plan.Steps, &LoadYAMLStep{
		ConfigPath: yamlFile,
	})

	// Step 2: Check preconditions (OpenSCAD)
	plan.Steps = append(plan.Steps, &CheckPreconditionsStep{})

	// Step 3: Validate files
	plan.Steps = append(plan.Steps, &ValidateFilesStep{})

	// Step 4: Render SCAD files
	plan.Steps = append(plan.Steps, &RenderSCADFilesStep{})

	// Step 5: Combine with groups
	plan.Steps = append(plan.Steps, &CombineWithGroupsStep{})

	return plan, nil
}

// createObjectGroupPlan creates a plan for command-line object groups
func (p *Planner) createObjectGroupPlan(objectGroups []ObjectGroup, outputFile string) (*BuildPlan, error) {
	plan := &BuildPlan{
		OutputFile: outputFile,
	}

	// Step 1: Parse object groups into YAML config structure
	plan.Steps = append(plan.Steps, &ParseObjectGroupsStep{
		ObjectGroups: objectGroups,
		OutputFile:   outputFile,
	})

	// Step 2: Check preconditions (OpenSCAD)
	plan.Steps = append(plan.Steps, &CheckPreconditionsStep{})

	// Step 3: Validate files
	plan.Steps = append(plan.Steps, &ValidateFilesStep{})

	// Step 4: Render SCAD files
	plan.Steps = append(plan.Steps, &RenderSCADFilesStep{})

	// Step 5: Combine with groups
	plan.Steps = append(plan.Steps, &CombineWithGroupsStep{})

	return plan, nil
}

// createSCADPlan creates a plan for direct SCAD files
func (p *Planner) createSCADPlan(scadFiles []string, outputFile string) (*BuildPlan, error) {
	plan := &BuildPlan{
		OutputFile: outputFile,
	}

	// Step 1: Parse SCAD file arguments and convert to single object group
	plan.Steps = append(plan.Steps, &ParseSCADArgsAsSingleObjectStep{
		Args:       scadFiles,
		OutputFile: outputFile,
	})

	// Step 2: Check preconditions (OpenSCAD)
	plan.Steps = append(plan.Steps, &CheckPreconditionsStep{})

	// Step 3: Validate files
	plan.Steps = append(plan.Steps, &ValidateFilesStep{})

	// Step 4: Render SCAD files
	plan.Steps = append(plan.Steps, &RenderSCADFilesStep{})

	// Step 5: Combine with groups (using single object with multiple parts)
	plan.Steps = append(plan.Steps, &CombineWithGroupsStep{})

	return plan, nil
}

// create3MFPlan creates a plan for 3MF files
func (p *Planner) create3MFPlan(files []string, outputFile string) (*BuildPlan, error) {
	plan := &BuildPlan{
		OutputFile: outputFile,
	}

	// Step 1: Validate 3MF files
	plan.Steps = append(plan.Steps, &Validate3MFFilesStep{
		Files: files,
	})

	// Step 2: Combine 3MF files
	plan.Steps = append(plan.Steps, &Combine3MFFilesStep{
		Files:      files,
		OutputFile: outputFile,
	})

	return plan, nil
}

// createSTLPlan creates a plan for STL files
func (p *Planner) createSTLPlan(files []string, outputFile string) (*BuildPlan, error) {
	plan := &BuildPlan{
		OutputFile: outputFile,
	}

	// Step 1: Validate STL files
	plan.Steps = append(plan.Steps, &ValidateSTLFilesStep{
		Files: files,
	})

	// Step 2: Convert STL files to 3MF
	plan.Steps = append(plan.Steps, &ConvertSTLTo3MFStep{
		Files: files,
	})

	// Step 3: Combine converted 3MF files
	plan.Steps = append(plan.Steps, &CombineConverted3MFFilesStep{
		OutputFile: outputFile,
	})

	return plan, nil
}

// Execute runs all steps in the plan
func (p *BuildPlan) Execute() error {
	if ui.IsVerbose() {
		ui.PrintTitle("Build Plan Execution")
		ui.PrintInfo(fmt.Sprintf("Total steps: %d", len(p.Steps)))
		ui.PrintSeparator()
	}
	
	for i, step := range p.Steps {
		if ui.IsVerbose() {
			ui.PrintHeader(fmt.Sprintf("Step %d/%d: %s", i+1, len(p.Steps), step.Name()))
		}
		if err := step.Execute(); err != nil {
			return err
		}
	}
	
	// Update OutputFile from buildContext if not already set
	if p.OutputFile == "" && buildContext.OutputFile != "" {
		p.OutputFile = buildContext.OutputFile
	}
	
	ui.PrintSeparator()
	ui.PrintSuccess("Build completed successfully!")
	if p.OutputFile != "" {
		// Convert to relative path if possible
		relPath, err := filepath.Rel(".", p.OutputFile)
		if err != nil {
			relPath = p.OutputFile
		}
		ui.PrintKeyValue("Output file", relPath)
	}
	return nil
}

// DetectFileType determines the file type based on extension
func (p *Planner) DetectFileType(path string) FileType {
	return detectFileType(path)
}

// detectFileType determines the file type based on extension (internal helper)
func detectFileType(path string) FileType {
	// Handle colon-separated format (e.g., "file.scad:name:slot")
	// Extract just the file path part
	if colonIdx := strings.Index(path, ":"); colonIdx != -1 {
		path = path[:colonIdx]
	}

	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".yaml", ".yml":
		return FileTypeYAML
	case ".scad":
		return FileTypeSCAD
	case ".3mf":
		return FileType3MF
	case ".stl":
		return FileTypeSTL
	default:
		return FileTypeUnknown
	}
}

// getFileTypeNames returns a list of file type names for error messages
func getFileTypeNames(fileTypes map[FileType][]string) []string {
	var names []string
	for ft := range fileTypes {
		switch ft {
		case FileTypeYAML:
			names = append(names, "YAML")
		case FileTypeSCAD:
			names = append(names, "SCAD")
		case FileType3MF:
			names = append(names, "3MF")
		case FileTypeSTL:
			names = append(names, "STL")
		}
	}
	return names
}

// Context holds shared data between build steps
type Context struct {
	YAMLConfig    *models.YamlConfig
	SCADFiles     []models.ScadFile
	RenderedFiles []string
	OutputFile    string
	OriginalSTLs  []string // Store original STL filenames for proper naming
}

var buildContext = &Context{}

// ParseObjectGroupsStep parses command-line object groups into YAML config
type ParseObjectGroupsStep struct {
	ObjectGroups []ObjectGroup
	OutputFile   string
}

func (s *ParseObjectGroupsStep) Name() string {
	return "Parse object groups"
}

func (s *ParseObjectGroupsStep) Execute() error {
	// Create YamlConfig from object groups
	yamlConfig := &models.YamlConfig{
		Output:  s.OutputFile,
		Objects: make([]models.YamlObject, 0, len(s.ObjectGroups)),
	}

	for _, objGroup := range s.ObjectGroups {
		yamlObj := models.YamlObject{
			Name:  objGroup.Name,
			Parts: make([]models.YamlPart, 0, len(objGroup.Files)),
		}

		for _, fileArg := range objGroup.Files {
			// Parse file argument: path or path:name:slot
			parts := strings.Split(fileArg, ":")
			path := parts[0]

			// Convert to absolute path
			absPath, err := filepath.Abs(path)
			if err != nil {
				return fmt.Errorf("invalid file path %s: %w", path, err)
			}

			// Use custom name if provided, otherwise derive from filename
			name := ""
			if len(parts) > 1 && parts[1] != "" {
				name = parts[1]
			} else {
				// Use filename without extension
				name = filepath.Base(absPath[:len(absPath)-len(filepath.Ext(absPath))])
			}

			// Parse optional filament slot (format: path:name:slot or path::slot)
			filamentSlot := 0 // 0 means auto-assign
			if len(parts) > 2 {
				slot := 0
				_, err := fmt.Sscanf(parts[2], "%d", &slot)
				if err == nil && slot >= 1 && slot <= 4 {
					filamentSlot = slot
				} else if parts[2] != "" {
					return fmt.Errorf("invalid filament slot '%s' for %s. Must be 1-4", parts[2], path)
				}
			}

			yamlObj.Parts = append(yamlObj.Parts, models.YamlPart{
				Name:     name,
				File:     absPath,
				Filament: filamentSlot,
			})
		}

		yamlConfig.Objects = append(yamlConfig.Objects, yamlObj)
	}

	buildContext.YAMLConfig = yamlConfig
	buildContext.OutputFile = s.OutputFile

	// Display configuration summary
	ui.PrintSuccess(fmt.Sprintf("Parsed %d object(s)", len(yamlConfig.Objects)))
	if ui.IsVerbose() {
		for _, obj := range yamlConfig.Objects {
			ui.PrintItem(fmt.Sprintf("Object: %s (%d part%s)", obj.Name, len(obj.Parts), pluralize(len(obj.Parts))))
			for _, part := range obj.Parts {
				filamentInfo := ""
				if part.Filament > 0 {
					filamentInfo = fmt.Sprintf(" [filament %d]", part.Filament)
				}
				ui.PrintItem(fmt.Sprintf("  └─ %s: %s%s", part.Name, filepath.Base(part.File), filamentInfo))
			}
		}
	}

	return nil
}

// pluralize returns "s" if count != 1, empty string otherwise
func pluralize(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}

// LoadYAMLStep loads and validates YAML configuration
type LoadYAMLStep struct {
	ConfigPath string
	Plan       *BuildPlan
}

func (s *LoadYAMLStep) Name() string {
	return "Load YAML configuration"
}

func (s *LoadYAMLStep) Execute() error {
	loader := config.NewLoader()
	cfg, err := loader.Load(s.ConfigPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	buildContext.YAMLConfig = cfg
	buildContext.OutputFile = cfg.Output
	ui.PrintSuccess(fmt.Sprintf("Loaded configuration with %d object(s)", len(cfg.Objects)))

	// Display configuration summary only in verbose mode
	if ui.IsVerbose() {
		for _, obj := range cfg.Objects {
			ui.PrintItem(fmt.Sprintf("Object: %s (%d part%s)", obj.Name, len(obj.Parts), pluralize(len(obj.Parts))))
			for _, part := range obj.Parts {
				filamentInfo := ""
				if part.Filament > 0 {
					filamentInfo = fmt.Sprintf(" [filament %d]", part.Filament)
				}
				ui.PrintItem(fmt.Sprintf("  └─ %s: %s%s", part.Name, filepath.Base(part.File), filamentInfo))
			}
		}
	}
	return nil
}

// CheckPreconditionsStep checks if OpenSCAD is installed
type CheckPreconditionsStep struct{}

func (s *CheckPreconditionsStep) Name() string {
	return "Check preconditions"
}

func (s *CheckPreconditionsStep) Execute() error {
	if err := preconditions.Check(); err != nil {
		return fmt.Errorf("OpenSCAD not found: %w", err)
	}
	if ui.IsVerbose() {
		ui.PrintSuccess("✓ OpenSCAD is available")
	}
	return nil
}

// ValidateFilesStep validates that all files exist
type ValidateFilesStep struct{}

func (s *ValidateFilesStep) Name() string {
	return "Validate files"
}

func (s *ValidateFilesStep) Execute() error {
	var allPaths []string
	if buildContext.YAMLConfig != nil {
		loader := config.NewLoader()
		scadFiles := loader.ConvertToScadFiles(buildContext.YAMLConfig)
		buildContext.SCADFiles = scadFiles
		for _, scad := range scadFiles {
			allPaths = append(allPaths, scad.Path)
		}
	} else if len(buildContext.SCADFiles) > 0 {
		for _, scad := range buildContext.SCADFiles {
			allPaths = append(allPaths, scad.Path)
		}
	}

	if len(allPaths) > 0 {
		if err := preconditions.ValidateFiles(allPaths); err != nil {
			return err
		}
		if ui.IsVerbose() {
			ui.PrintSuccess(fmt.Sprintf("✓ Validated %d file(s)", len(allPaths)))
		}
	}
	return nil
}

// RenderSCADFilesStep renders all SCAD files to 3MF
type RenderSCADFilesStep struct{}

func (s *RenderSCADFilesStep) Name() string {
	return "Render SCAD files"
}

func (s *RenderSCADFilesStep) Execute() error {
	if len(buildContext.SCADFiles) == 0 {
		return fmt.Errorf("no SCAD files to render")
	}

	baseDir := filepath.Dir(buildContext.SCADFiles[0].Path)
	var scadPaths []string
	for _, scad := range buildContext.SCADFiles {
		scadPaths = append(scadPaths, scad.Path)
	}

	if !ui.IsVerbose() {
		ui.PrintInfo(fmt.Sprintf("Rendering %d SCAD file(s)...", len(scadPaths)))
	}
	
	tempFiles, err := renderer.RenderMultipleSCAD(baseDir, scadPaths)
	if err != nil {
		return err
	}
	buildContext.RenderedFiles = tempFiles

	if ui.IsVerbose() {
		for _, scad := range buildContext.SCADFiles {
			ui.PrintItem(fmt.Sprintf("✓ %s → %s", filepath.Base(scad.Path), scad.Name))
		}
	}
	ui.PrintSuccess(fmt.Sprintf("Rendered %d file(s)", len(tempFiles)))
	return nil
}

// CombineWithGroupsStep combines rendered files using YAML grouping
type CombineWithGroupsStep struct{}

func (s *CombineWithGroupsStep) Name() string {
	return "Combine with groups"
}

func (s *CombineWithGroupsStep) Execute() error {
	defer renderer.CleanupTempFiles(buildContext.RenderedFiles)

	ui.PrintInfo("Merging objects and materials...")
	
	combiner := threemf.NewCombiner()
	if err := combiner.CombineWithGroups(buildContext.RenderedFiles, buildContext.SCADFiles, buildContext.OutputFile); err != nil {
		return err
	}

	// Print success
	ui.PrintSuccess("Combined 3MF file created!")

	// Show objects using the same printer as inspect
	inspector := inspect.NewInspector()
	model, settings, err := inspector.Read3MFFile(buildContext.OutputFile)
	if err == nil {
		ui.PrintHeader("Model Contents")
		printer := inspect.NewModelPrinter()
		printer.PrintObjectHierarchy(model, settings)
	}

	return nil
}

// ParseSCADArgsStep parses SCAD file arguments
type ParseSCADArgsStep struct {
	Args []string
}

func (s *ParseSCADArgsStep) Name() string {
	return "Parse SCAD arguments"
}

func (s *ParseSCADArgsStep) Execute() error {
	var scadFiles []models.ScadFile
	for _, arg := range s.Args {
		parts := strings.Split(arg, ":")
		path := parts[0]

		// Convert to absolute path
		absPath, err := filepath.Abs(path)
		if err != nil {
			return fmt.Errorf("invalid file path %s: %w", path, err)
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
				return fmt.Errorf("invalid filament slot '%s' for %s. Must be 1-4", parts[2], path)
			}
		}

		scadFiles = append(scadFiles, models.ScadFile{
			Path:         absPath,
			Name:         name,
			FilamentSlot: filamentSlot,
		})
	}
	buildContext.SCADFiles = scadFiles
	return nil
}

// ParseSCADArgsAsSingleObjectStep parses SCAD file arguments and creates a single object with multiple parts
type ParseSCADArgsAsSingleObjectStep struct {
	Args       []string
	OutputFile string
}

func (s *ParseSCADArgsAsSingleObjectStep) Name() string {
	return "Parse SCAD arguments as single object"
}

func (s *ParseSCADArgsAsSingleObjectStep) Execute() error {
	var parts []models.YamlPart
	for _, arg := range s.Args {
		argParts := strings.Split(arg, ":")
		path := argParts[0]

		// Convert to absolute path
		absPath, err := filepath.Abs(path)
		if err != nil {
			return fmt.Errorf("invalid file path %s: %w", path, err)
		}

		// Use custom name if provided
		name := ""
		filamentSlot := 0 // 0 means auto-assign

		if len(argParts) > 1 && argParts[1] != "" {
			name = argParts[1]
		} else {
			// Use filename without extension
			name = filepath.Base(absPath[:len(absPath)-len(filepath.Ext(absPath))])
		}

		// Parse optional filament slot (format: path:name:slot)
		if len(argParts) > 2 {
			slot := 0
			_, err := fmt.Sscanf(argParts[2], "%d", &slot)
			if err == nil && slot >= 1 && slot <= 4 {
				filamentSlot = slot
			} else {
				return fmt.Errorf("invalid filament slot '%s' for %s. Must be 1-4", argParts[2], path)
			}
		}

		parts = append(parts, models.YamlPart{
			Name:     name,
			File:     absPath,
			Filament: filamentSlot,
		})
	}

	// Create a YAML config with a single object containing all parts
	buildContext.YAMLConfig = &models.YamlConfig{
		Output: s.OutputFile,
		Objects: []models.YamlObject{
			{
				Name:  "Combined",
				Parts: parts,
			},
		},
	}
	buildContext.OutputFile = s.OutputFile

	return nil
}

// CombineRenderedStep combines rendered 3MF files
type CombineRenderedStep struct {
	OutputFile string
}

func (s *CombineRenderedStep) Name() string {
	return "Combine rendered files"
}

func (s *CombineRenderedStep) Execute() error {
	ui.PrintHeader("Combining 3MF files...")

	defer renderer.CleanupTempFiles(buildContext.RenderedFiles)

	combiner := threemf.NewCombiner()
	if err := combiner.Combine(buildContext.RenderedFiles, buildContext.SCADFiles, s.OutputFile); err != nil {
		return err
	}

	// Print success
	ui.PrintSuccess("Combined 3MF file created: " + s.OutputFile)
	var names []string
	for _, scad := range buildContext.SCADFiles {
		names = append(names, scad.Name)
	}
	ui.PrintObjectList(names)
	return nil
}

// Validate3MFFilesStep validates 3MF files exist
type Validate3MFFilesStep struct {
	Files []string
}

func (s *Validate3MFFilesStep) Name() string {
	return "Validate 3MF files"
}

func (s *Validate3MFFilesStep) Execute() error {
	for _, file := range s.Files {
		if _, err := os.Stat(file); err != nil {
			return fmt.Errorf("file not found: %s", file)
		}
		ui.PrintItem(fmt.Sprintf("✓ %s", filepath.Base(file)))
	}
	ui.PrintSuccess(fmt.Sprintf("Validated %d 3MF file(s)", len(s.Files)))
	return nil
}

// Combine3MFFilesStep combines 3MF files
type Combine3MFFilesStep struct {
	Files      []string
	OutputFile string
}

func (s *Combine3MFFilesStep) Name() string {
	return "Combine 3MF files"
}

func (s *Combine3MFFilesStep) Execute() error {
	ui.PrintInfo("Merging 3MF files...")
	combiner := combine.NewCombiner()
	if err := combiner.Combine(s.Files, s.OutputFile); err != nil {
		return err
	}
	ui.PrintSuccess("Combined 3MF created successfully!")
	return nil
}

// ValidateSTLFilesStep validates STL files exist
type ValidateSTLFilesStep struct {
	Files []string
}

func (s *ValidateSTLFilesStep) Name() string {
	return "Validate STL files"
}

func (s *ValidateSTLFilesStep) Execute() error {
	for _, file := range s.Files {
		if _, err := os.Stat(file); err != nil {
			return fmt.Errorf("file not found: %s", file)
		}
		ui.PrintItem(fmt.Sprintf("✓ %s", filepath.Base(file)))
	}
	ui.PrintSuccess(fmt.Sprintf("Validated %d STL file(s)", len(s.Files)))
	return nil
}

// ConvertSTLTo3MFStep converts STL files to 3MF format
type ConvertSTLTo3MFStep struct {
	Files []string
}

func (s *ConvertSTLTo3MFStep) Name() string {
	return "Convert STL files to 3MF"
}

func (s *ConvertSTLTo3MFStep) Execute() error {
	converter := stl.NewConverter()
	buildContext.RenderedFiles = []string{}
	buildContext.OriginalSTLs = s.Files

	ui.PrintInfo(fmt.Sprintf("Converting %d STL file(s) to 3MF...", len(s.Files)))
	
	for i, stlFile := range s.Files {
		// Create temp 3MF file
		tempFile := filepath.Join(os.TempDir(), fmt.Sprintf("stl_converted_%d.3mf", i))

		if err := converter.ConvertTo3MF(stlFile, tempFile); err != nil {
			return fmt.Errorf("error converting %s: %w", stlFile, err)
		}

		buildContext.RenderedFiles = append(buildContext.RenderedFiles, tempFile)
		ui.PrintItem(fmt.Sprintf("✓ %s → %s", filepath.Base(stlFile), filepath.Base(tempFile)))
	}
	
	ui.PrintSuccess(fmt.Sprintf("Converted %d file(s)", len(s.Files)))
	return nil
}

// CombineConverted3MFFilesStep combines converted 3MF files
type CombineConverted3MFFilesStep struct {
	OutputFile string
}

func (s *CombineConverted3MFFilesStep) Name() string {
	return "Combine converted 3MF files"
}

func (s *CombineConverted3MFFilesStep) Execute() error {
	ui.PrintHeader("Combining converted 3MF files...")

	if len(buildContext.RenderedFiles) == 0 {
		return fmt.Errorf("no converted files to combine")
	}

	combiner := threemf.NewCombiner()

	// Create ScadFile entries using original STL filenames for proper naming
	scadFiles := make([]models.ScadFile, len(buildContext.RenderedFiles))
	for i, file := range buildContext.RenderedFiles {
		// Use original STL filename without extension
		originalName := filepath.Base(buildContext.OriginalSTLs[i])
		name := strings.TrimSuffix(originalName, filepath.Ext(originalName))

		scadFiles[i] = models.ScadFile{
			Path: file,
			Name: name,
		}
	}

	if err := combiner.Combine(buildContext.RenderedFiles, scadFiles, s.OutputFile); err != nil {
		return err
	}

	ui.PrintSuccess("Combined 3MF file created: " + s.OutputFile)

	// Clean up temp files
	for _, file := range buildContext.RenderedFiles {
		os.Remove(file)
	}

	return nil
}
