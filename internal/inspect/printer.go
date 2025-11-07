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
		ui.PrintStep("No objects found")
	}
}

// printObject recursively prints an object and its components
func (p *ModelPrinter) printObject(model *models.Model, obj *models.Object, settingsMap map[string]*models.SettingsObject, partsMap map[string]*models.Part, depth int) {
	indent := strings.Repeat("  ", depth)

	name := obj.Name
	if name == "" {
		name = "(unnamed)"
	}

	// Get filament information
	filamentInfo := ""
	if settings, ok := settingsMap[obj.ID]; ok {
		for _, meta := range settings.Metadata {
			if meta.Key == "extruder" && meta.Value != "" {
				filamentInfo = fmt.Sprintf(" (filament: %s)", meta.Value)
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
		ui.PrintStep(fmt.Sprintf("%s• %s (ID: %s) - %d part(s)%s%s", indent, name, obj.ID, len(obj.Components.Component), filamentInfo, meshInfo))

		// Print each component
		for _, comp := range obj.Components.Component {
			// Find the component object
			for _, compObj := range model.Resources.Objects {
				if compObj.ID == comp.ObjectID {
					p.printComponent(&compObj, comp, partsMap, depth+1)
					break
				}
			}
		}
	} else {
		// Leaf object (just a mesh)
		ui.PrintStep(fmt.Sprintf("%s• %s (ID: %s)%s%s", indent, name, obj.ID, filamentInfo, meshInfo))
	}
}

// printComponent prints a component with its filament information
func (p *ModelPrinter) printComponent(obj *models.Object, comp models.Component, partsMap map[string]*models.Part, depth int) {
	indent := strings.Repeat("  ", depth)

	name := obj.Name
	if name == "" {
		name = "(unnamed)"
	}

	// Get filament information from part settings
	filamentInfo := ""
	if part, ok := partsMap[obj.ID]; ok {
		for _, meta := range part.Metadata {
			if meta.Key == "extruder" && meta.Value != "" {
				filamentInfo = fmt.Sprintf(" (filament: %s)", meta.Value)
				break
			}
			if meta.Key == "name" && meta.Value != "" {
				name = meta.Value
			}
		}
	}

	// Get offset information from transform
	offsetInfo := ""
	if comp.Transform != "" {
		if x, y, z, ok := parseTransformOffset(comp.Transform); ok {
			// Only show offset if it's not zero
			if x != 0 || y != 0 || z != 0 {
				offsetInfo = fmt.Sprintf(" [offset: %.2f, %.2f, %.2f]", x, y, z)
			}
		}
	}

	ui.PrintStep(fmt.Sprintf("%s- %s (ID: %s)%s%s", indent, name, obj.ID, filamentInfo, offsetInfo))
}
