package inspect

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strings"

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

	ui.PrintHeader(fmt.Sprintf("Inspecting: %s", filename))

	// Read the 3MF file
	model, settings, err := i.read3MFFile(filename)
	if err != nil {
		return fmt.Errorf("error reading 3MF file: %w", err)
	}

	// Print basic information
	ui.PrintStep(fmt.Sprintf("Unit: %s", model.Unit))
	ui.PrintStep(fmt.Sprintf("Language: %s", model.Lang))

	// Print metadata if available
	if len(model.Metadata) > 0 {
		ui.PrintStep("Metadata:")
		for _, meta := range model.Metadata {
			ui.PrintStep(fmt.Sprintf("  - %s: %s", meta.Name, meta.Value))
		}
	}

	// Print build items (what's on the build plate)
	ui.PrintHeader("Build Plate Items:")
	if len(model.Build.Items) == 0 {
		ui.PrintStep("No items on build plate")
	} else {
		for idx, item := range model.Build.Items {
			objectName := i.getObjectName(model, item.ObjectID)
			printable := "yes"
			if item.Printable == "0" {
				printable = "no"
			}
			ui.PrintStep(fmt.Sprintf("%d. Object ID %s: %s (printable: %s)", idx+1, item.ObjectID, objectName, printable))
		}
	}

	// Print object hierarchy
	ui.PrintHeader("Objects in Model:")
	i.printObjectHierarchy(model, settings)

	return nil
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

// printObjectHierarchy prints the object hierarchy with components and colors
func (i *Inspector) printObjectHierarchy(model *models.Model, settings *models.ModelSettings) {
	// Create a map of object IDs to settings info
	settingsMap := make(map[string]*models.SettingsObject)
	partsMap := make(map[string]*models.Part)
	
	if settings != nil {
		for idx := range settings.Objects {
			obj := &settings.Objects[idx]
			settingsMap[obj.ID] = obj
			for pidx := range obj.Parts {
				part := &obj.Parts[pidx]
				partsMap[part.ID] = part
			}
		}
	}

	// Track which objects are components (not top-level)
	componentIDs := make(map[string]bool)
	for _, obj := range model.Resources.Objects {
		if obj.Components != nil {
			for _, comp := range obj.Components.Component {
				componentIDs[comp.ObjectID] = true
			}
		}
	}

	// Print top-level objects (those that have components or are in build items)
	objectCount := 0
	for _, obj := range model.Resources.Objects {
		// Skip objects that are only used as components
		if obj.Components == nil && componentIDs[obj.ID] {
			continue
		}

		objectCount++
		i.printObject(model, &obj, settingsMap, partsMap, 0)
	}

	if objectCount == 0 {
		ui.PrintStep("No objects found")
	}
}

// printObject recursively prints an object and its components
func (i *Inspector) printObject(model *models.Model, obj *models.Object, settingsMap map[string]*models.SettingsObject, partsMap map[string]*models.Part, depth int) {
	indent := strings.Repeat("  ", depth)
	
	name := obj.Name
	if name == "" {
		name = "(unnamed)"
	}

	// Get color/filament information
	colorInfo := ""
	if settings, ok := settingsMap[obj.ID]; ok {
		for _, meta := range settings.Metadata {
			if meta.Key == "extruder" && meta.Value != "" {
				colorInfo = fmt.Sprintf(" (color: %s)", meta.Value)
				break
			}
		}
	}

	// Check if this object has a mesh (actual geometry)
	hasMesh := obj.Mesh != nil
	meshInfo := ""
	if hasMesh {
		meshInfo = " [has mesh]"
	}

	// Print the object
	if obj.Components != nil && len(obj.Components.Component) > 0 {
		// Parent object with components
		ui.PrintStep(fmt.Sprintf("%s• %s (ID: %s) - %d part(s)%s%s", indent, name, obj.ID, len(obj.Components.Component), colorInfo, meshInfo))
		
		// Print each component
		for _, comp := range obj.Components.Component {
			// Find the component object
			for _, compObj := range model.Resources.Objects {
				if compObj.ID == comp.ObjectID {
					i.printComponent(&compObj, comp, partsMap, depth+1)
					break
				}
			}
		}
	} else {
		// Leaf object (just a mesh)
		ui.PrintStep(fmt.Sprintf("%s• %s (ID: %s)%s%s", indent, name, obj.ID, colorInfo, meshInfo))
	}
}

// printComponent prints a component with its color information
func (i *Inspector) printComponent(obj *models.Object, comp models.Component, partsMap map[string]*models.Part, depth int) {
	indent := strings.Repeat("  ", depth)
	
	name := obj.Name
	if name == "" {
		name = "(unnamed)"
	}

	// Get color/filament information from part settings
	colorInfo := ""
	if part, ok := partsMap[obj.ID]; ok {
		for _, meta := range part.Metadata {
			if meta.Key == "extruder" && meta.Value != "" {
				colorInfo = fmt.Sprintf(" (color: %s)", meta.Value)
				break
			}
			if meta.Key == "name" && meta.Value != "" {
				name = meta.Value
			}
		}
	}

	ui.PrintStep(fmt.Sprintf("%s- %s (ID: %s)%s", indent, name, obj.ID, colorInfo))
}
