package threemf

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/philipparndt/go3mf/internal/geometry"
	"github.com/philipparndt/go3mf/internal/models"
)

// Reader reads 3MF files
type Reader struct{}

// Read reads and parses a 3MF file
func (r *Reader) Read(filename string) (*models.Model, error) {
	zr, err := zip.OpenReader(filename)
	if err != nil {
		return nil, fmt.Errorf("error opening ZIP: %w", err)
	}
	defer zr.Close()

	var modelFile *zip.File
	for _, f := range zr.File {
		if f.Name == "3D/3dmodel.model" {
			modelFile = f
			break
		}
	}

	if modelFile == nil {
		return nil, fmt.Errorf("3D/3dmodel.model not found in archive")
	}

	rc, err := modelFile.Open()
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
		return nil, fmt.Errorf("error parsing XML: %w", err)
	}

	return &model, nil
}

// Writer writes 3MF files
type Writer struct{}

// WriteBambu writes a model to a 3MF file with Bambu Studio support
func (w *Writer) WriteBambu(outputFile string, model *models.Model, sourceFile string, objectGroups []models.ObjectGroup, buildItems []models.Item) error {
	// Add Bambu metadata
	AddBambuMetadata(model)

	// Read source ZIP to get metadata files
	sourceZip, err := zip.OpenReader(sourceFile)
	if err != nil {
		return fmt.Errorf("error opening source ZIP: %w", err)
	}
	defer sourceZip.Close()

	// Create output ZIP
	outFile, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("error creating output file: %w", err)
	}
	defer outFile.Close()

	outZip := zip.NewWriter(outFile)
	defer outZip.Close()

	// Write model XML
	modelXML, err := xml.MarshalIndent(model, "", "\t")
	if err != nil {
		return fmt.Errorf("error marshaling XML: %w", err)
	}

	w_, err := outZip.Create("3D/3dmodel.model")
	if err != nil {
		return fmt.Errorf("error creating model entry: %w", err)
	}

	// Write XML declaration
	if _, err := w_.Write([]byte(xml.Header)); err != nil {
		return fmt.Errorf("error writing XML header: %w", err)
	}

	if _, err := w_.Write(modelXML); err != nil {
		return fmt.Errorf("error writing model XML: %w", err)
	}

	// Write Bambu model settings
	if err := WriteModelSettings(outZip, objectGroups, buildItems); err != nil {
		return fmt.Errorf("error writing model settings: %w", err)
	}

	// Copy other files from source
	for _, file := range sourceZip.File {
		if file.Name == "3D/3dmodel.model" || file.Name == "Metadata/model_settings.config" {
			continue
		}

		srcFile, err := file.Open()
		if err != nil {
			return fmt.Errorf("error opening source file: %w", err)
		}

		dst, err := outZip.Create(file.Name)
		if err != nil {
			srcFile.Close()
			return fmt.Errorf("error creating ZIP entry: %w", err)
		}

		if _, err := io.Copy(dst, srcFile); err != nil {
			srcFile.Close()
			return fmt.Errorf("error copying file: %w", err)
		}

		srcFile.Close()
	}

	return nil
}

// Write writes a model to a 3MF file, copying metadata from sourceFile
func (w *Writer) Write(outputFile string, model *models.Model, sourceFile string) error {
	// Read source ZIP to get metadata files
	sourceZip, err := zip.OpenReader(sourceFile)
	if err != nil {
		return fmt.Errorf("error opening source ZIP: %w", err)
	}
	defer sourceZip.Close()

	// Create output ZIP
	outFile, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("error creating output file: %w", err)
	}
	defer outFile.Close()

	outZip := zip.NewWriter(outFile)
	defer outZip.Close()

	// Write model XML
	modelXML, err := xml.MarshalIndent(model, "", "\t")
	if err != nil {
		return fmt.Errorf("error marshaling XML: %w", err)
	}

	w_, err := outZip.Create("3D/3dmodel.model")
	if err != nil {
		return fmt.Errorf("error creating model entry: %w", err)
	}

	// Write XML declaration
	if _, err := w_.Write([]byte(xml.Header)); err != nil {
		return fmt.Errorf("error writing XML header: %w", err)
	}

	if _, err := w_.Write(modelXML); err != nil {
		return fmt.Errorf("error writing model XML: %w", err)
	}

	// Copy other files from source
	for _, file := range sourceZip.File {
		if file.Name == "3D/3dmodel.model" {
			continue
		}

		srcFile, err := file.Open()
		if err != nil {
			return fmt.Errorf("error opening source file: %w", err)
		}

		dst, err := outZip.Create(file.Name)
		if err != nil {
			srcFile.Close()
			return fmt.Errorf("error creating ZIP entry: %w", err)
		}

		if _, err := io.Copy(dst, srcFile); err != nil {
			srcFile.Close()
			return fmt.Errorf("error copying file: %w", err)
		}

		srcFile.Close()
	}

	return nil
}

// Combiner combines multiple 3MF models
type Combiner struct {
	reader *Reader
	writer *Writer
}

// NewCombiner creates a new Combiner
func NewCombiner() *Combiner {
	return &Combiner{
		reader: &Reader{},
		writer: &Writer{},
	}
}

// Combine combines multiple 3MF files into one
func (c *Combiner) Combine(tempFiles []string, scadFiles []models.ScadFile, outputFile string) error {
	c.CombineWithDistance(tempFiles, scadFiles, outputFile, 10.0)
	return nil
}

// CombineWithDistance combines multiple 3MF files with a configurable packing distance
func (c *Combiner) CombineWithDistance(tempFiles []string, scadFiles []models.ScadFile, outputFile string, packingDistance float64) error {
	var allObjects []models.Object

	// Read all models and collect their objects
	for i, tempFile := range tempFiles {
		model, err := c.reader.Read(tempFile)
		if err != nil {
			return fmt.Errorf("error reading 3MF file %d: %w", i, err)
		}

		// Collect mesh objects
		for _, obj := range model.Resources.Objects {
			obj.ID = strconv.Itoa(i + 1)
			obj.Name = scadFiles[i].Name
			obj.UUID = "" // Will be set in components

			// Set PID (Production ID) based on filament slot
			filamentSlot := scadFiles[i].FilamentSlot
			if filamentSlot == 0 {
				// Auto-assign filament slot if not specified
				filamentSlot = ((i % 4) + 1)
			}
			obj.PID = strconv.Itoa(filamentSlot)
			obj.PIndex = "0"

			allObjects = append(allObjects, obj)
		}
	}

	// Create a parent object with components
	// Arrange objects side by side with spacing to avoid overlap
	margin := packingDistance // mm margin between objects
	var components []models.Component
	currentXOffset := 0.0

	for i := range allObjects {
		// Position objects along the X axis with spacing and apply rotation
		scadFile := scadFiles[i]
		transform := geometry.BuildRotationTransform(
			scadFile.RotationX, scadFile.RotationY, scadFile.RotationZ,
			currentXOffset, 0, 0)

		components = append(components, models.Component{
			ObjectID:  strconv.Itoa(i + 1),
			Transform: transform,
		})

		// Calculate width of this object for next position
		bbox, err := geometry.CalculateBoundingBox(&allObjects[i])
		if err == nil {
			currentXOffset += bbox.Width() + margin
		} else {
			currentXOffset += 50.0 // fallback spacing
		}
	}

	parentID := strconv.Itoa(len(allObjects) + 1)
	parentObject := models.Object{
		ID:   parentID,
		Type: "model",
		Components: &models.Components{
			Component: components,
		},
	}

	buildItems := []models.Item{
		{
			ObjectID:  parentID,
			Transform: "1 0 0 0 1 0 0 0 1 0 0 0",
			Printable: "1",
		},
	}

	// Create the combined model
	combinedModel := &models.Model{
		Xmlns: "http://schemas.microsoft.com/3dmanufacturing/core/2015/02",
		Unit:  "millimeter",
		Lang:  "en-US",
		Resources: models.Resources{
			Objects: append(allObjects, parentObject),
		},
		Build: models.Build{
			Items: buildItems,
		},
	}

	// Create single object group for settings
	objectGroups := []models.ObjectGroup{
		{
			ID:    parentID,
			Name:  "combined",
			Parts: scadFiles,
		},
	}

	// Write combined model to output file with Bambu support
	return c.writer.WriteBambu(outputFile, combinedModel, tempFiles[0], objectGroups, buildItems)
}

// CombineWithGroups combines multiple 3MF files into one, grouping parts by object name
func (c *Combiner) CombineWithGroups(tempFiles []string, scadFiles []models.ScadFile, outputFile string) error {
	c.CombineWithGroupsAndDistance(tempFiles, scadFiles, outputFile, 10.0, models.PackingAlgorithmDefault)
	return nil
}

// CombineWithObjectGroups combines multiple 3MF files with ObjectGroup metadata including normalization settings
func (c *Combiner) CombineWithObjectGroups(tempFiles []string, objectGroups []models.ObjectGroup, outputFile string, packingDistance float64, algorithm models.PackingAlgorithm) error {
	// Flatten object groups into scadFiles for compatibility
	var scadFiles []models.ScadFile
	objectGroupMap := make(map[string]bool) // map object name -> normalize_position

	for _, group := range objectGroups {
		for _, part := range group.Parts {
			scadFiles = append(scadFiles, part)
			// Store the normalize_position setting for this object
			objectGroupMap[group.Name] = group.NormalizePosition
		}
	}

	return c.combineWithGroupsAndDistanceInternal(tempFiles, scadFiles, objectGroups, outputFile, packingDistance, algorithm)
}

// CombineWithGroupsAndDistance combines multiple 3MF files with grouping and configurable packing distance
func (c *Combiner) CombineWithGroupsAndDistance(tempFiles []string, scadFiles []models.ScadFile, outputFile string, packingDistance float64, algorithm models.PackingAlgorithm) error {
	// When ObjectGroups are not provided, create default ones with normalize_position=true
	return c.combineWithGroupsAndDistanceInternal(tempFiles, scadFiles, nil, outputFile, packingDistance, algorithm)
}

func (c *Combiner) combineWithGroupsAndDistanceInternal(tempFiles []string, scadFiles []models.ScadFile, objectGroups []models.ObjectGroup, outputFile string, packingDistance float64, algorithm models.PackingAlgorithm) error {
	var allMeshObjects []models.Object
	nextID := 1

	// Read all models and collect their mesh objects
	for i, tempFile := range tempFiles {
		model, err := c.reader.Read(tempFile)
		if err != nil {
			return fmt.Errorf("error reading 3MF file %d: %w", i, err)
		}

		// Collect mesh objects
		for _, obj := range model.Resources.Objects {
			obj.ID = strconv.Itoa(nextID)
			obj.Name = scadFiles[i].Name
			obj.UUID = "" // Will be set in components

			// Set PID (Production ID) based on filament slot
			filamentSlot := scadFiles[i].FilamentSlot
			if filamentSlot == 0 {
				// Auto-assign filament slot if not specified
				filamentSlot = ((i % 4) + 1)
			}
			obj.PID = strconv.Itoa(filamentSlot)
			obj.PIndex = "0"

			allMeshObjects = append(allMeshObjects, obj)
			nextID++
		}
	}

	// Group mesh objects by their base object name (before the '/')
	objectGroupsMap := make(map[string][]int) // object name -> list of mesh object IDs
	objectOrder := []string{}                 // preserve order of objects

	for i, scadFile := range scadFiles {
		// Extract object name (part before '/')
		objectName := scadFile.Name
		for j := 0; j < len(objectName); j++ {
			if objectName[j] == '/' {
				objectName = objectName[:j]
				break
			}
		}

		// Track first occurrence for ordering
		if _, exists := objectGroupsMap[objectName]; !exists {
			objectOrder = append(objectOrder, objectName)
		}

		// Map object name to mesh object ID (1-based)
		objectGroupsMap[objectName] = append(objectGroupsMap[objectName], i+1)
	}

	// Create parent objects for each group
	var parentObjects []models.Object
	var buildItems []models.Item
	var settingsGroups []models.ObjectGroup

	// Prepare objects for bin packing
	margin := packingDistance // mm margin between objects
	var packingObjects []geometry.Rectangle
	objectInfoMap := make(map[int]struct {
		meshIDs      []int
		objectName   string
		groupObjects []models.Object
		scadFiles    []models.ScadFile
	})

	packingID := 0
	for _, objectName := range objectOrder {
		meshIDs := objectGroupsMap[objectName]

		// Calculate bounding box for this group of objects
		var groupObjects []models.Object
		var groupScadFiles []models.ScadFile
		for _, meshID := range meshIDs {
			groupObjects = append(groupObjects, allMeshObjects[meshID-1])
			groupScadFiles = append(groupScadFiles, scadFiles[meshID-1])
		}

		// Calculate dimensions for packing
		var width, height float64
		if len(meshIDs) == 1 {
			bbox, err := geometry.CalculateBoundingBox(&groupObjects[0])
			if err == nil {
				width = bbox.Width()
				height = bbox.Height()
			} else {
				width, height = 50.0, 50.0 // fallback
			}
		} else {
			// For multi-part objects, calculate combined bounding box
			transforms := make([]string, len(groupObjects))
			for i := range transforms {
				transforms[i] = "1 0 0 0 1 0 0 0 1 0 0 0" // All at origin
			}
			groupBBox, err := geometry.CalculateCombinedBoundingBox(groupObjects, transforms)
			if err == nil {
				width = groupBBox.Width()
				height = groupBBox.Height()
			} else {
				width, height = 100.0, 100.0 // fallback
			}
		}

		packingObjects = append(packingObjects, geometry.Rectangle{
			Width:  width,
			Height: height,
			ID:     packingID,
		})

		objectInfoMap[packingID] = struct {
			meshIDs      []int
			objectName   string
			groupObjects []models.Object
			scadFiles    []models.ScadFile
		}{meshIDs, objectName, groupObjects, groupScadFiles}

		packingID++
	}

	// Use bin packing algorithm to arrange objects based on selected algorithm
	packer := geometry.NewPacker(margin)
	var packingResults []geometry.PackingResult
	
	switch algorithm {
	case models.PackingAlgorithmCompact:
		packingResults = packer.PackCompact(packingObjects)
	default:
		packingResults = packer.PackOptimal(packingObjects, 256.0) // 256mm typical build plate width
	}

	// Create objects and build items based on packing results
	for _, result := range packingResults {
		info := objectInfoMap[result.ID]
		meshIDs := info.meshIDs
		objectName := info.objectName
		groupScadFiles := info.scadFiles

		// Determine if we should normalize position
		normalizePosition := true // default to true
		if objectGroups != nil {
			// Look up the normalize_position setting from objectGroups
			for _, og := range objectGroups {
				if og.Name == objectName {
					normalizePosition = og.NormalizePosition
					break
				}
			}
		}

		// Calculate z-offset for normalization if needed
		var zOffset float64 = 0
		if normalizePosition {
			// For single-part objects
			if len(meshIDs) == 1 {
				zOffset = geometry.CalculateGroupZOffset([]models.Object{info.groupObjects[0]})
			} else {
				// For multi-part objects, calculate transforms first then get z-offset
				var transforms []string
				for i := range info.groupObjects {
					scadFile := groupScadFiles[i]
					transform := geometry.BuildRotationTransform(
						scadFile.RotationX, scadFile.RotationY, scadFile.RotationZ,
						scadFile.PositionX, scadFile.PositionY, scadFile.PositionZ)
					transforms = append(transforms, transform)
				}
				zOffset = geometry.CalculateZOffsetWithTransforms(info.groupObjects, transforms)
			}
		}

		// Build transform for positioning on the build plate
		buildTransform := geometry.BuildTranslationTransform(result.X, result.Y, zOffset)

		// If only one part in this object, add it directly to build
		if len(meshIDs) == 1 {
			objectID := strconv.Itoa(meshIDs[0])

			// Apply rotation and position offsets from ScadFile to the build transform
			scadFile := groupScadFiles[0]
			buildTransform = geometry.BuildRotationTransform(
				scadFile.RotationX, scadFile.RotationY, scadFile.RotationZ,
				result.X+scadFile.PositionX, result.Y+scadFile.PositionY, zOffset+scadFile.PositionZ)

			buildItems = append(buildItems, models.Item{
				ObjectID:  objectID,
				Transform: buildTransform,
				Printable: "1",
			})

			// Add to settings groups
			settingsGroups = append(settingsGroups, models.ObjectGroup{
				ID:                objectID,
				Name:              objectName,
				Parts:             groupScadFiles,
				NormalizePosition: normalizePosition,
			})
		} else {
			// Create a parent object with multiple components
			// Parts within an object maintain their relative positions
			var components []models.Component

			for i, meshID := range meshIDs {
				// Apply rotation and position offsets from ScadFile to each component
				scadFile := groupScadFiles[i]
				transform := geometry.BuildRotationTransform(
					scadFile.RotationX, scadFile.RotationY, scadFile.RotationZ,
					scadFile.PositionX, scadFile.PositionY, scadFile.PositionZ)

				components = append(components, models.Component{
					ObjectID:  strconv.Itoa(meshID),
					Transform: transform,
				})
			}

			parentID := strconv.Itoa(nextID)
			nextID++

			parentObject := models.Object{
				ID:   parentID,
				Name: objectName,
				Type: "model",
				Components: &models.Components{
					Component: components,
				},
			}

			parentObjects = append(parentObjects, parentObject)

			buildItems = append(buildItems, models.Item{
				ObjectID:  parentID,
				Transform: buildTransform,
				Printable: "1",
			})

			// Add to settings groups
			settingsGroups = append(settingsGroups, models.ObjectGroup{
				ID:                parentID,
				Name:              objectName,
				Parts:             groupScadFiles,
				NormalizePosition: normalizePosition,
			})
		}
	}

	// Combine all objects
	allObjects := append(allMeshObjects, parentObjects...)

	// Create the combined model
	combinedModel := &models.Model{
		Xmlns: "http://schemas.microsoft.com/3dmanufacturing/core/2015/02",
		Unit:  "millimeter",
		Lang:  "en-US",
		Resources: models.Resources{
			Objects: allObjects,
		},
		Build: models.Build{
			Items: buildItems,
		},
	}

	// Write combined model to output file with Bambu support
	return c.writer.WriteBambu(outputFile, combinedModel, tempFiles[0], settingsGroups, buildItems)
}

func getMaxObjectID(model *models.Model) int {
	maxID := 0
	for _, obj := range model.Resources.Objects {
		if id, err := strconv.Atoi(obj.ID); err == nil && id > maxID {
			maxID = id
		}
	}
	return maxID
}
