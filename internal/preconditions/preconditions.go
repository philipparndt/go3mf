package preconditions

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Check verifies all preconditions are met
func Check() error {
	checks := []struct {
		name string
		fn   func() error
	}{
		{"OpenSCAD", checkOpenSCAD},
	}

	for _, check := range checks {
		if err := check.fn(); err != nil {
			return fmt.Errorf("%s: %w", check.name, err)
		}
	}

	return nil
}

func checkOpenSCAD() error {
	_, err := exec.LookPath("openscad")
	if err != nil {
		return fmt.Errorf("not found in PATH. Please install OpenSCAD from https://openscad.org/")
	}
	return nil
}

// ValidateFiles checks if files exist and are readable
// Supports SCAD, STL, and 3MF files
func ValidateFiles(paths []string) error {
	for _, path := range paths {
		// Parse path:name format if provided
		parts := strings.Split(path, ":")
		filePath := parts[0]

		info, err := os.Stat(filePath)
		if err != nil {
			return fmt.Errorf("cannot access file %s: %w", filePath, err)
		}

		if info.IsDir() {
			return fmt.Errorf("%s is a directory, not a file", filePath)
		}

		if !isSupportedFile(filePath) {
			return fmt.Errorf("%s is not a supported file type (must be .scad, .stl, or .3mf)", filePath)
		}

		file, err := os.Open(filePath)
		if err != nil {
			return fmt.Errorf("cannot read file %s: %w", filePath, err)
		}
		file.Close()
	}

	return nil
}

func isSupportedFile(path string) bool {
	lowerPath := strings.ToLower(path)
	return strings.HasSuffix(lowerPath, ".scad") ||
		strings.HasSuffix(lowerPath, ".stl") ||
		strings.HasSuffix(lowerPath, ".3mf")
}

// IsScadFile checks if a file has a .scad extension
func IsScadFile(path string) bool {
	return strings.HasSuffix(strings.ToLower(path), ".scad")
}

// IsSTLFile checks if a file has a .stl extension
func IsSTLFile(path string) bool {
	return strings.HasSuffix(strings.ToLower(path), ".stl")
}

// Is3MFFile checks if a file has a .3mf extension
func Is3MFFile(path string) bool {
	return strings.HasSuffix(strings.ToLower(path), ".3mf")
}

// ValidateOutputPath checks if the output path is writable
func ValidateOutputPath(path string) error {
	// Check if parent directory exists and is writable
	dir := path
	if dir == "" {
		dir = "."
	}

	// Get the directory of the output path
	for dir != "" && dir != "." && dir != "/" {
		info, err := os.Stat(dir)
		if err == nil {
			if info.IsDir() && (info.Mode()&0200) != 0 {
				return nil
			}
		}
		parent := dir[:len(dir)-1]
		if idx := len(parent) - 1; idx >= 0 && parent[idx] == '/' {
			dir = parent
		} else {
			break
		}
	}

	// If parent doesn't exist or isn't writable, check current directory
	dir = "."
	if info, err := os.Stat(dir); err != nil || !info.IsDir() || (info.Mode()&0200) == 0 {
		return fmt.Errorf("output directory is not writable")
	}

	return nil
}
