package inspect

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/user/go3mf/internal/models"
	"github.com/user/go3mf/internal/ui"
)

// ModelPrinter handles printing model hierarchy and details
type ModelPrinter struct{}

// NewModelPrinter creates a new ModelPrinter
func NewModelPrinter() *ModelPrinter {
	return &ModelPrinter{}
}

// ParseTransformOffset extracts X, Y, Z offset from a transform matrix string
// Transform format: "m11 m12 m13 m21 m22 m23 m31 m32 m33 x y z"
func ParseTransformOffset(transform string) (x, y, z float64, ok bool) {
	parts := strings.Fields(transform)
	if len(parts) != 12 {
		return 0, 0, 0, false
	}

	x, errX := strconv.ParseFloat(parts[9], 64)
	y, errY := strconv.ParseFloat(parts[10], 64)
	z, errZ := strconv.ParseFloat(parts[11], 64)

	if errX != nil || errY != nil || errZ != nil {
		return 0, 0, 0, false
	}

	return x, y, z, true
}

// parseTransformOffset is a wrapper for backward compatibility
func parseTransformOffset(transform string) (x, y, z float64, ok bool) {
	return ParseTransformOffset(transform)
}

// PrintObjectHierarchy prints the object hierarchy with components and filaments
func (p *ModelPrinter) PrintObjectHierarchy(model *models.Model, settings *models.ModelSettings) {
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
		p.printObject(model, &obj, settingsMap, partsMap, 0)
	}

	if objectCount == 0 {
		ui.PrintInfo("No objects found")
	}
}

// printObject recursively prints an object and its components
func (p *ModelPrinter) printObject(model *models.Model, obj *models.Object, settingsMap map[string]*models.SettingsObject, partsMap map[string]*models.Part, depth int) {
	name := obj.Name
	if name == "" {
		name = "(unnamed)"
	}

	// Get filament information
	filament := ""
	if settings, ok := settingsMap[obj.ID]; ok {
		for _, meta := range settings.Metadata {
			if meta.Key == "extruder" && meta.Value != "" {
				filament = fmt.Sprintf("filament:%-2s", meta.Value)
				break
			}
		}
	}

	// Build details
	details := []string{}
	if obj.Mesh != nil {
		details = append(details, "mesh")
	}
	if obj.Components != nil && len(obj.Components.Component) > 0 {
		details = append(details, fmt.Sprintf("%d parts", len(obj.Components.Component)))
	}

	// Format the line with proper spacing
	detailStr := ""
	if len(details) > 0 {
		detailStr = fmt.Sprintf("[%s]", strings.Join(details, ", "))
	}

	line := fmt.Sprintf("%-30s  id:%-6s  %-14s  %s", name, obj.ID, filament, detailStr)
	ui.PrintItem(strings.TrimRight(line, " "))

	// Print each component
	if obj.Components != nil {
		for _, comp := range obj.Components.Component {
			// Find the component object
			for _, compObj := range model.Resources.Objects {
				if compObj.ID == comp.ObjectID {
					p.printComponent(&compObj, comp, partsMap, depth+1)
					break
				}
			}
		}
	}
}

// printComponent prints a component with its filament information
func (p *ModelPrinter) printComponent(obj *models.Object, comp models.Component, partsMap map[string]*models.Part, depth int) {
	name := obj.Name
	if name == "" {
		name = "(unnamed)"
	}

	// Add indentation prefix
	name = "  └─ " + name

	// Get filament information from part settings
	filament := ""
	if part, ok := partsMap[obj.ID]; ok {
		for _, meta := range part.Metadata {
			if meta.Key == "extruder" && meta.Value != "" {
				filament = fmt.Sprintf("filament:%-2s", meta.Value)
				break
			}
			if meta.Key == "name" && meta.Value != "" {
				name = "  └─ " + meta.Value
			}
		}
	}

	// Get offset information from transform
	offset := ""
	if comp.Transform != "" {
		if x, y, z, ok := parseTransformOffset(comp.Transform); ok {
			// Only show offset if it's not zero
			if x != 0 || y != 0 || z != 0 {
				offset = fmt.Sprintf("[offset: %.1f, %.1f, %.1f]", x, y, z)
			}
		}
	}

	// Format the line with proper spacing
	line := fmt.Sprintf("%-30s  id:%-6s  %-14s  %s", name, obj.ID, filament, offset)
	ui.PrintItem(strings.TrimRight(line, " "))
}
