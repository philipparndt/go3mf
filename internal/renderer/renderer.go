package renderer

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/philipparndt/go3mf/internal/models"
	"github.com/philipparndt/go3mf/internal/ui"
)

// RenderSCAD renders a SCAD file to 3MF format
func RenderSCAD(workDir, scadFile, outputFile string) error {
	// Convert scadFile to absolute path if it's relative
	absScadFile := scadFile
	if !filepath.IsAbs(scadFile) {
		absScadFile = filepath.Join(workDir, scadFile)
	}

	cmd := exec.Command("openscad", "-o", outputFile, absScadFile)
	cmd.Dir = workDir

	// Only show output in verbose mode, otherwise suppress it
	if ui.IsVerbose() {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	} else {
		cmd.Stdout = io.Discard
		cmd.Stderr = io.Discard
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to render %s: %w", scadFile, err)
	}
	return nil
}

// RenderSCADWithConfig renders a SCAD file with optional config content to 3MF format
func RenderSCADWithConfig(workDir, scadFile, outputFile, configContent string) error {
	// Convert scadFile to absolute path if it's relative
	absScadFile := scadFile
	if !filepath.IsAbs(scadFile) {
		absScadFile = filepath.Join(workDir, scadFile)
	}

	// If no config content, use simple render
	if configContent == "" {
		return RenderSCAD(workDir, scadFile, outputFile)
	}

	// Create a temporary config file in /tmp with a unique name
	configFile := filepath.Join("/tmp", filepath.Base(filepath.Dir(absScadFile))+"_config_"+filepath.Base(absScadFile[:len(absScadFile)-len(filepath.Ext(absScadFile))])+".scad")

	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	defer os.Remove(configFile)

	cmd := exec.Command("openscad", "-o", outputFile, "-D", "cfg_file=\""+configFile+"\"", absScadFile)
	cmd.Dir = workDir

	// Only show output in verbose mode, otherwise suppress it
	if ui.IsVerbose() {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	} else {
		cmd.Stdout = io.Discard
		cmd.Stderr = io.Discard
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to render %s with config: %w", scadFile, err)
	}
	return nil
}

// RenderSCADWithConfigFiles renders a SCAD file with multiple config files to 3MF format
func RenderSCADWithConfigFiles(workDir, scadFile, outputFile string, configFiles map[string]string) error {
	// Convert scadFile to absolute path if it's relative
	absScadFile := scadFile
	if !filepath.IsAbs(scadFile) {
		absScadFile = filepath.Join(workDir, scadFile)
	}

	// If no config files, use simple render
	if len(configFiles) == 0 {
		return RenderSCAD(workDir, scadFile, outputFile)
	}

	// Copy the SCAD file to the working directory
	scadFileName := filepath.Base(absScadFile)
	workDirScadFile := filepath.Join(workDir, scadFileName)

	scadContent, err := os.ReadFile(absScadFile)
	if err != nil {
		return fmt.Errorf("failed to read SCAD file %s: %w", absScadFile, err)
	}

	if err := os.WriteFile(workDirScadFile, scadContent, 0644); err != nil {
		return fmt.Errorf("failed to copy SCAD file to working directory: %w", err)
	}

	// Write config files to the working directory
	for filename, content := range configFiles {
		configPath := filepath.Join(workDir, filename)
		if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write config file %s: %w", filename, err)
		}
	}

	// Run OpenSCAD from the working directory with the local SCAD file
	cmd := exec.Command("openscad", "-o", outputFile, scadFileName)
	cmd.Dir = workDir

	// Only show output in verbose mode, otherwise suppress it
	if ui.IsVerbose() {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	} else {
		cmd.Stdout = io.Discard
		cmd.Stderr = io.Discard
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to render %s with config files: %w", scadFile, err)
	}
	return nil
}

// RenderMultipleSCAD renders multiple SCAD files and returns temporary file paths
func RenderMultipleSCAD(baseDir string, scadFiles []string) ([]string, error) {
	var tempFiles []string

	for i, scadFile := range scadFiles {
		tempFile := fmt.Sprintf("/tmp/scad_render_%d.3mf", i)
		tempFiles = append(tempFiles, tempFile)

		if err := RenderSCAD(baseDir, scadFile, tempFile); err != nil {
			return nil, err
		}
	}

	return tempFiles, nil
}

// RenderMultipleSCADWithConfigs renders multiple SCAD files with their config files and returns temporary file paths
func RenderMultipleSCADWithConfigs(baseDir string, scadFiles []models.ScadFile) ([]string, error) {
	var tempFiles []string

	for i, scadFile := range scadFiles {
		tempFile := fmt.Sprintf("/tmp/scad_render_%d.3mf", i)
		tempFiles = append(tempFiles, tempFile)

		// Write config files to the base directory with their original names
		for filename, content := range scadFile.ConfigFiles {
			configPath := filepath.Join(baseDir, filename)

			if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
				return nil, fmt.Errorf("failed to write config file %s: %w", configPath, err)
			}
		}

		// Render this part
		if err := RenderSCAD(baseDir, scadFile.Path, tempFile); err != nil {
			return nil, err
		}
	}

	return tempFiles, nil
}

// CleanupTempFiles removes temporary files
func CleanupTempFiles(files []string) {
	for _, f := range files {
		os.Remove(f)
	}
}
