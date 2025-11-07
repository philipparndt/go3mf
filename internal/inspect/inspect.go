package inspect

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/user/go3mf/internal/models"
	"github.com/user/go3mf/internal/ui"
)

// Inspector provides functionality to inspect 3MF files
type Inspector struct{}

// NewInspector creates a new Inspector
func NewInspector() *Inspector {
	return &Inspector{}
}

// Inspect reads and displays the contents of a 3MF file
func (i *Inspector) Inspect(filename string) error {
	// Check if file exists
	if _, err := os.Stat(filename); err != nil {
		return fmt.Errorf("file not found: %s", filename)
	}

	ui.PrintTitle("3MF File Inspector")
	ui.PrintKeyValue("File", filename)

	// Read the 3MF file
	model, settings, err := i.read3MFFile(filename)
	if err != nil {
		return fmt.Errorf("error reading 3MF file: %w", err)
	}

	// Print basic information
	ui.PrintHeader("File Information")
	ui.PrintKeyValue("Unit", model.Unit)
	ui.PrintKeyValue("Language", model.Lang)

	// Print metadata if available
	if len(model.Metadata) > 0 {
		ui.PrintHeader("Metadata")
		for _, meta := range model.Metadata {
			ui.PrintKeyValue(meta.Name, meta.Value)
		}
	}

	// Print build items (what's on the build plate)
	ui.PrintHeader("Build Plate Items")
	if len(model.Build.Items) == 0 {
		ui.PrintInfo("No items on build plate")
	} else {
		for idx, item := range model.Build.Items {
			objectName := i.getObjectName(model, item.ObjectID)
			printable := "✓ yes"
			if item.Printable == "0" {
				printable = "✗ no"
			}

			// Get offset information from transform
			offsetInfo := ""
			if item.Transform != "" {
				if x, y, z, ok := ParseTransformOffset(item.Transform); ok {
					if x != 0 || y != 0 || z != 0 {
						offsetInfo = fmt.Sprintf(" 📍 [%.2f, %.2f, %.2f]", x, y, z)
					}
				}
			}

			ui.PrintItem(fmt.Sprintf("#%d Object %s: %s (printable: %s)%s", idx+1, item.ObjectID, objectName, printable, offsetInfo))
		}
	}

	// Print object hierarchy
	ui.PrintHeader("Model Objects")
	printer := NewModelPrinter()
	printer.PrintObjectHierarchy(model, settings)

	ui.PrintSeparator()
	ui.PrintSuccess("Inspection complete!")
	// Convert to relative path if possible
	relPath, err := filepath.Rel(".", filename)
	if err != nil {
		relPath = filename
	}
	ui.PrintKeyValue("File", relPath)

	return nil
}

// Read3MFFile reads a 3MF file and returns the model and settings (exported for use by other packages)
func (i *Inspector) Read3MFFile(filename string) (*models.Model, *models.ModelSettings, error) {
	return i.read3MFFile(filename)
}

// read3MFFile reads a 3MF file and returns the model and settings
func (i *Inspector) read3MFFile(filename string) (*models.Model, *models.ModelSettings, error) {
	zr, err := zip.OpenReader(filename)
	if err != nil {
		return nil, nil, fmt.Errorf("error opening file: %w", err)
	}
	defer zr.Close()

	// Read the main model file
	var modelFile *zip.File
	var settingsFile *zip.File
	for _, f := range zr.File {
		if f.Name == "3D/3dmodel.model" {
			modelFile = f
		}
		if f.Name == "Metadata/model_settings.config" {
			settingsFile = f
		}
	}

	if modelFile == nil {
		return nil, nil, fmt.Errorf("3D/3dmodel.model not found in archive")
	}

	// Parse the model
	model, err := i.parseModel(modelFile)
	if err != nil {
		return nil, nil, err
	}

	// Parse settings if available (Bambu Studio specific)
	var settings *models.ModelSettings
	if settingsFile != nil {
		settings, _ = i.parseSettings(settingsFile)
	}

	return model, settings, nil
}

// parseModel parses the 3D model XML
func (i *Inspector) parseModel(file *zip.File) (*models.Model, error) {
	rc, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("error opening model file: %w", err)
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		return nil, fmt.Errorf("error reading model file: %w", err)
	}

	var model models.Model
	if err := xml.Unmarshal(data, &model); err != nil {
		return nil, fmt.Errorf("error parsing model XML: %w", err)
	}

	return &model, nil
}

// parseSettings parses the Bambu Studio settings file
func (i *Inspector) parseSettings(file *zip.File) (*models.ModelSettings, error) {
	rc, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		return nil, err
	}

	var settings models.ModelSettings
	if err := xml.Unmarshal(data, &settings); err != nil {
		return nil, err
	}

	return &settings, nil
}

// getObjectName returns the name of an object by ID
func (i *Inspector) getObjectName(model *models.Model, objectID string) string {
	for _, obj := range model.Resources.Objects {
		if obj.ID == objectID {
			if obj.Name != "" {
				return obj.Name
			}
			return "(unnamed)"
		}
	}
	return "(not found)"
}
