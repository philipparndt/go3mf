package renderer

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to render %s: %w", scadFile, err)
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

// CleanupTempFiles removes temporary files
func CleanupTempFiles(files []string) {
	for _, f := range files {
		os.Remove(f)
	}
}
