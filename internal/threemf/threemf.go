package threemf

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"math"
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

// WriteBambuWithPlates writes a model to a 3MF file with Bambu Studio multi-plate support
func (w *Writer) WriteBambuWithPlates(outputFile string, model *models.Model, sourceFile string, objectGroups []models.ObjectGroup, buildItems []models.Item, plateGroups []models.PlateGroup, plateObjectIDs map[int][]string) error {
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

	// Write Bambu model settings with multi-plate support
	if err := WriteModelSettingsWithPlates(outZip, objectGroups, buildItems, plateGroups, plateObjectIDs); err != nil {
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
	Debug  bool // Enable debug output
}

// NewCombiner creates a new Combiner
func NewCombiner() *Combiner {
	return &Combiner{
		reader: &Reader{},
		writer: &Writer{},
	}
}

// SetDebug enables or disables debug output
func (c *Combiner) SetDebug(debug bool) {
	c.Debug = debug
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
		// Position objects along the X axis with spacing (rotation already baked into mesh)
		transform := geometry.BuildTranslationTransform(currentXOffset, 0, 0)

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
	meshMinZ := make(map[int]float64) // mesh index -> minZ after rotation
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

			// Apply rotation only (no Z normalization yet - will be done at group level)
			scadFile := scadFiles[i]
			minZ, err := geometry.RotateMeshVertices(&obj, scadFile.RotationX, scadFile.RotationY, scadFile.RotationZ)
			if err != nil {
				return fmt.Errorf("error rotating mesh vertices for %s: %w", scadFile.Name, err)
			}
			meshMinZ[i] = minZ

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
		bboxOffsetX  float64 // Offset needed to align rotated bbox corner to origin
		bboxOffsetY  float64
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
		// Note: Rotation is already baked into mesh vertices, so we use standard bounding box
		var width, height float64
		var bboxOffsetX, bboxOffsetY float64 // Offset to align bbox corner to origin
		if len(meshIDs) == 1 {
			// Use standard bounding box (rotation already baked into mesh)
			bbox, err := geometry.CalculateBoundingBox(&groupObjects[0])
			if err == nil {
				width = bbox.Width()
				height = bbox.Height()
				// Store offset needed to bring the bbox corner to the expected position
				bboxOffsetX = -bbox.MinX
				bboxOffsetY = -bbox.MinY
				if c.Debug {
					fmt.Printf("DEBUG: %s - bbox(%.1f,%.1f)-(%.1f,%.1f) size(%.1f,%.1f) offset(%.1f,%.1f)\n",
						objectName, bbox.MinX, bbox.MinY, bbox.MaxX, bbox.MaxY, width, height, bboxOffsetX, bboxOffsetY)
				}
			} else {
				width, height = 50.0, 50.0 // fallback
				if c.Debug {
					fmt.Printf("DEBUG: %s - fallback size(50,50) - error: %v\n", objectName, err)
				}
			}
		} else {
			// For multi-part objects, calculate combined bounding box
			var combinedBBox *geometry.BoundingBox
			for i, obj := range groupObjects {
				scadFile := groupScadFiles[i]
				bbox, err := geometry.CalculateBoundingBox(&obj)
				if err != nil {
					continue
				}
				// Apply position offsets
				bbox.MinX += scadFile.PositionX
				bbox.MaxX += scadFile.PositionX
				bbox.MinY += scadFile.PositionY
				bbox.MaxY += scadFile.PositionY

				if combinedBBox == nil {
					combinedBBox = bbox
				} else {
					combinedBBox.MinX = math.Min(combinedBBox.MinX, bbox.MinX)
					combinedBBox.MinY = math.Min(combinedBBox.MinY, bbox.MinY)
					combinedBBox.MaxX = math.Max(combinedBBox.MaxX, bbox.MaxX)
					combinedBBox.MaxY = math.Max(combinedBBox.MaxY, bbox.MaxY)
				}
			}
			if combinedBBox != nil {
				width = combinedBBox.Width()
				height = combinedBBox.Height()
				// Store offset needed to bring the combined bbox corner to the expected position
				bboxOffsetX = -combinedBBox.MinX
				bboxOffsetY = -combinedBBox.MinY
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
			bboxOffsetX  float64
			bboxOffsetY  float64
		}{meshIDs, objectName, groupObjects, groupScadFiles, bboxOffsetX, bboxOffsetY}

		packingID++
	}

	// Apply group-level Z normalization
	// For each object group, calculate the minimum Z (considering PositionZ offsets) and normalize
	for _, info := range objectInfoMap {
		// Determine if we should normalize position for this group
		normalizePosition := true // default to true
		if objectGroups != nil {
			for _, og := range objectGroups {
				if og.Name == info.objectName {
					normalizePosition = og.NormalizePosition
					break
				}
			}
		}

		if !normalizePosition {
			continue
		}

		// Calculate the minimum Z across all parts in this group (considering PositionZ)
		groupMinZ := math.MaxFloat64
		for i, meshID := range info.meshIDs {
			partMinZ := meshMinZ[meshID-1] // meshMinZ is 0-indexed, meshID is 1-indexed
			partPositionZ := info.scadFiles[i].PositionZ
			effectiveMinZ := partMinZ + partPositionZ
			if effectiveMinZ < groupMinZ {
				groupMinZ = effectiveMinZ
			}
		}

		// Apply the group-level Z offset to normalize to ground level
		if groupMinZ != math.MaxFloat64 && groupMinZ != 0 {
			zOffset := -groupMinZ
			for _, meshID := range info.meshIDs {
				if err := geometry.ApplyZOffset(&allMeshObjects[meshID-1], zOffset); err != nil {
					return fmt.Errorf("error applying Z offset to mesh: %w", err)
				}
			}
		}
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
		bboxOffsetX := info.bboxOffsetX
		bboxOffsetY := info.bboxOffsetY

		if c.Debug {
			fmt.Printf("DEBUG PACK: %s - packer pos(%.1f,%.1f) size(%.1f,%.1f) offset(%.1f,%.1f) -> final pos(%.1f,%.1f) occupies(%.1f,%.1f)-(%.1f,%.1f)\n",
				objectName, result.X, result.Y, result.Width, result.Height, bboxOffsetX, bboxOffsetY,
				result.X+bboxOffsetX, result.Y+bboxOffsetY,
				result.X, result.Y, result.X+result.Width, result.Y+result.Height)
		}

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

		// Z offset is 0 since rotation and Z normalization are already baked into mesh vertices
		var zOffset float64 = 0
		if normalizePosition && len(meshIDs) > 1 {
			// For multi-part objects, we still need to calculate Z offset for position offsets
			for i := range info.groupObjects {
				scadFile := groupScadFiles[i]
				if scadFile.PositionZ != 0 {
					// Note: position offsets are still applied via transforms
					break
				}
			}
		}

		// Build transform for positioning on the build plate (translation only, rotation is baked)
		var buildTransform string

		// If only one part in this object, add it directly to build
		if len(meshIDs) == 1 {
			objectID := strconv.Itoa(meshIDs[0])

			// Use translation-only transform since rotation is baked into mesh
			scadFile := groupScadFiles[0]
			buildTransform = geometry.BuildTranslationTransform(
				result.X+scadFile.PositionX+bboxOffsetX, result.Y+scadFile.PositionY+bboxOffsetY, zOffset+scadFile.PositionZ)

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
				// Apply only position offsets from ScadFile to each component (rotation is baked)
				scadFile := groupScadFiles[i]
				transform := geometry.BuildTranslationTransform(
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

			// Apply bboxOffset to position the object correctly
			buildTransform = geometry.BuildTranslationTransform(result.X+bboxOffsetX, result.Y+bboxOffsetY, zOffset)

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

// CombineWithPlateGroups combines multiple 3MF files with multi-plate support
func (c *Combiner) CombineWithPlateGroups(tempFiles []string, plateGroups []models.PlateGroup, outputFile string, packingDistance float64, algorithm models.PackingAlgorithm, plateWidth float64) error {
	var allMeshObjects []models.Object
	var allScadFiles []models.ScadFile
	var allObjectGroups []models.ObjectGroup
	nextID := 1

	// Build a map from scadFile.Name to temp file index
	fileIndex := 0
	scadFileToTempIndex := make(map[string]int)

	// Collect all parts from all plates for file indexing
	for _, plate := range plateGroups {
		for _, obj := range plate.Objects {
			for _, part := range obj.Parts {
				scadFileToTempIndex[part.Name] = fileIndex
				fileIndex++
			}
		}
	}

	// Read all models and collect their mesh objects
	for i, tempFile := range tempFiles {
		model, err := c.reader.Read(tempFile)
		if err != nil {
			return fmt.Errorf("error reading 3MF file %d: %w", i, err)
		}

		// Collect mesh objects
		for _, obj := range model.Resources.Objects {
			obj.ID = strconv.Itoa(nextID)
			obj.UUID = ""
			allMeshObjects = append(allMeshObjects, obj)
			nextID++
		}
	}

	// Flatten all plates into scadFiles and objectGroups with plate info
	fileIdx := 0
	for _, plate := range plateGroups {
		for _, obj := range plate.Objects {
			allObjectGroups = append(allObjectGroups, obj)
			for _, part := range obj.Parts {
				part.Name = obj.Name
				if len(obj.Parts) > 1 {
					// Only use composite name for multi-part objects
					// The part.Name has already been set correctly in ConvertToPlateGroups
				}
				allScadFiles = append(allScadFiles, part)
				fileIdx++
			}
		}
	}

	// Group mesh objects by their base object name
	objectGroupsMap := make(map[string][]int)
	objectOrder := []string{}

	for i, scadFile := range allScadFiles {
		objectName := scadFile.Name
		for j := 0; j < len(objectName); j++ {
			if objectName[j] == '/' {
				objectName = objectName[:j]
				break
			}
		}

		if _, exists := objectGroupsMap[objectName]; !exists {
			objectOrder = append(objectOrder, objectName)
		}
		objectGroupsMap[objectName] = append(objectGroupsMap[objectName], i+1)
	}

	// Determine which plate each object belongs to
	objectToPlate := make(map[string]int)
	for plateIdx, plate := range plateGroups {
		for _, obj := range plate.Objects {
			objectToPlate[obj.Name] = plateIdx
		}
	}

	// Create parent objects and build items
	var parentObjects []models.Object
	var buildItems []models.Item
	var settingsGroups []models.ObjectGroup

	// Prepare objects for bin packing per plate
	type platePackingInfo struct {
		packingObjects []geometry.Rectangle
		objectInfoMap  map[int]struct {
			meshIDs      []int
			objectName   string
			groupObjects []models.Object
			scadFiles    []models.ScadFile
			bboxOffsetX  float64
			bboxOffsetY  float64
		}
	}

	platePacking := make(map[int]*platePackingInfo)
	for i := range plateGroups {
		platePacking[i] = &platePackingInfo{
			objectInfoMap: make(map[int]struct {
				meshIDs      []int
				objectName   string
				groupObjects []models.Object
				scadFiles    []models.ScadFile
				bboxOffsetX  float64
				bboxOffsetY  float64
			}),
		}
	}

	packingIDCounter := 0
	for _, objectName := range objectOrder {
		meshIDs := objectGroupsMap[objectName]
		plateIdx := objectToPlate[objectName]

		var groupObjects []models.Object
		var groupScadFiles []models.ScadFile
		for _, meshID := range meshIDs {
			groupObjects = append(groupObjects, allMeshObjects[meshID-1])
			groupScadFiles = append(groupScadFiles, allScadFiles[meshID-1])
		}

		// Calculate dimensions for packing (rotation already baked into mesh)
		var width, height float64
		var bboxOffsetX, bboxOffsetY float64
		if len(meshIDs) == 1 {
			bbox, err := geometry.CalculateBoundingBox(&groupObjects[0])
			if err == nil {
				width = bbox.Width()
				height = bbox.Height()
				bboxOffsetX = -bbox.MinX
				bboxOffsetY = -bbox.MinY
			} else {
				width, height = 50.0, 50.0
			}
		} else {
			var combinedBBox *geometry.BoundingBox
			for i, obj := range groupObjects {
				scadFile := groupScadFiles[i]
				bbox, err := geometry.CalculateBoundingBox(&obj)
				if err != nil {
					continue
				}
				bbox.MinX += scadFile.PositionX
				bbox.MaxX += scadFile.PositionX
				bbox.MinY += scadFile.PositionY
				bbox.MaxY += scadFile.PositionY

				if combinedBBox == nil {
					combinedBBox = bbox
				} else {
					combinedBBox.MinX = math.Min(combinedBBox.MinX, bbox.MinX)
					combinedBBox.MinY = math.Min(combinedBBox.MinY, bbox.MinY)
					combinedBBox.MaxX = math.Max(combinedBBox.MaxX, bbox.MaxX)
					combinedBBox.MaxY = math.Max(combinedBBox.MaxY, bbox.MaxY)
				}
			}
			if combinedBBox != nil {
				width = combinedBBox.Width()
				height = combinedBBox.Height()
				bboxOffsetX = -combinedBBox.MinX
				bboxOffsetY = -combinedBBox.MinY
			} else {
				width, height = 100.0, 100.0
			}
		}

		packingID := packingIDCounter
		packingIDCounter++

		platePacking[plateIdx].packingObjects = append(platePacking[plateIdx].packingObjects, geometry.Rectangle{
			Width:  width,
			Height: height,
			ID:     packingID,
		})

		platePacking[plateIdx].objectInfoMap[packingID] = struct {
			meshIDs      []int
			objectName   string
			groupObjects []models.Object
			scadFiles    []models.ScadFile
			bboxOffsetX  float64
			bboxOffsetY  float64
		}{meshIDs, objectName, groupObjects, groupScadFiles, bboxOffsetX, bboxOffsetY}
	}

	// Track which build items belong to which plate
	plateObjectIDs := make(map[int][]string) // plateIdx -> list of object IDs

	// Pack and position objects per plate
	margin := packingDistance
	for plateIdx := range plateGroups {
		info := platePacking[plateIdx]
		if len(info.packingObjects) == 0 {
			continue
		}

		packer := geometry.NewPacker(margin)
		var packingResults []geometry.PackingResult

		switch algorithm {
		case models.PackingAlgorithmCompact:
			packingResults = packer.PackCompact(info.packingObjects)
		default:
			packingResults = packer.PackOptimal(info.packingObjects, plateWidth)
		}

		// Apply plate X offset
		plateXOffset := float64(plateIdx) * plateWidth

		for _, result := range packingResults {
			objInfo := info.objectInfoMap[result.ID]
			meshIDs := objInfo.meshIDs
			objectName := objInfo.objectName
			groupScadFiles := objInfo.scadFiles
			bboxOffsetX := objInfo.bboxOffsetX
			bboxOffsetY := objInfo.bboxOffsetY

			// Find normalization setting
			normalizePosition := true
			for _, og := range allObjectGroups {
				if og.Name == objectName {
					normalizePosition = og.NormalizePosition
					break
				}
			}

			// Z offset is 0 since rotation and Z normalization are already baked into mesh vertices
			var zOffset float64 = 0

			// Build position with plate offset
			posX := result.X + plateXOffset
			posY := result.Y

			var objectID string

			if len(meshIDs) == 1 {
				objectID = strconv.Itoa(meshIDs[0])
				scadFile := groupScadFiles[0]
				// Use translation-only transform since rotation is baked into mesh
				buildTransform := geometry.BuildTranslationTransform(
					posX+scadFile.PositionX+bboxOffsetX, posY+scadFile.PositionY+bboxOffsetY, zOffset+scadFile.PositionZ)

				buildItems = append(buildItems, models.Item{
					ObjectID:  objectID,
					Transform: buildTransform,
					Printable: "1",
				})

				settingsGroups = append(settingsGroups, models.ObjectGroup{
					ID:                objectID,
					Name:              objectName,
					Parts:             groupScadFiles,
					NormalizePosition: normalizePosition,
				})
			} else {
				var components []models.Component
				for i, meshID := range meshIDs {
					scadFile := groupScadFiles[i]
					// Use translation-only transform since rotation is baked into mesh
					transform := geometry.BuildTranslationTransform(
						scadFile.PositionX, scadFile.PositionY, scadFile.PositionZ)
					components = append(components, models.Component{
						ObjectID:  strconv.Itoa(meshID),
						Transform: transform,
					})
				}

				parentID := strconv.Itoa(nextID)
				nextID++
				objectID = parentID

				parentObjects = append(parentObjects, models.Object{
					ID:   parentID,
					Name: objectName,
					Type: "model",
					Components: &models.Components{
						Component: components,
					},
				})

				// Apply bboxOffset to position the object correctly
				buildTransform := geometry.BuildTranslationTransform(posX+bboxOffsetX, posY+bboxOffsetY, zOffset)
				buildItems = append(buildItems, models.Item{
					ObjectID:  parentID,
					Transform: buildTransform,
					Printable: "1",
				})

				settingsGroups = append(settingsGroups, models.ObjectGroup{
					ID:                parentID,
					Name:              objectName,
					Parts:             groupScadFiles,
					NormalizePosition: normalizePosition,
				})
			}

			plateObjectIDs[plateIdx] = append(plateObjectIDs[plateIdx], objectID)
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

	// Write combined model with multi-plate support
	return c.writer.WriteBambuWithPlates(outputFile, combinedModel, tempFiles[0], settingsGroups, buildItems, plateGroups, plateObjectIDs)
}
