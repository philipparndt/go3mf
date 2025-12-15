package combine

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/philipparndt/go3mf/internal/models"
)

// Combiner combines multiple 3MF files without rendering
type Combiner struct{}

// NewCombiner creates a new 3MF combiner
func NewCombiner() *Combiner {
	return &Combiner{}
}

// Combine combines multiple 3MF files into one
func (c *Combiner) Combine(inputFiles []string, outputFile string) error {
	if len(inputFiles) < 2 {
		return fmt.Errorf("at least 2 files required for combining")
	}

	var allObjects []models.Object
	var scadFiles []models.ScadFile

	// Read all models and collect their objects
	for i, inputFile := range inputFiles {
		model, _, err := c.readModel(inputFile)
		if err != nil {
			return fmt.Errorf("error reading file %d (%s): %w", i+1, inputFile, err)
		}

		// Get name from filename
		name := filepath.Base(inputFile[:len(inputFile)-len(filepath.Ext(inputFile))])

		// Collect mesh objects
		for _, obj := range model.Resources.Objects {
			obj.ID = strconv.Itoa(i + 1)
			obj.Name = name
			obj.UUID = "" // Will be set in components
			allObjects = append(allObjects, obj)
		}

		// Create ScadFile entry for settings (auto-assign filament)
		scadFiles = append(scadFiles, models.ScadFile{
			Path:         inputFile,
			Name:         name,
			FilamentSlot: 0, // Auto-assign
		})
	}

	// Create a parent object with components
	// Arrange objects side by side with spacing to avoid overlap
	spacing := 50.0 // mm spacing between objects
	var components []models.Component
	for i := range allObjects {
		// Position objects along the X axis with spacing
		xOffset := float64(i) * spacing
		transform := fmt.Sprintf("1 0 0 0 1 0 0 0 1 %.2f 0 0", xOffset)

		components = append(components, models.Component{
			ObjectID:  strconv.Itoa(i + 1),
			Transform: transform,
		})
	}

	parentID := strconv.Itoa(len(allObjects) + 1)
	parentObject := models.Object{
		ID:   parentID,
		Type: "model",
		Components: &models.Components{
			Component: components,
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
			Items: []models.Item{
				{
					ObjectID:  parentID,
					Transform: "1 0 0 0 1 0 0 0 1 0 0 0",
					Printable: "1",
				},
			},
		},
	}

	// Write combined model
	return c.writeModelBambu(outputFile, combinedModel, inputFiles[0], scadFiles)
}

// readModel reads and parses a 3MF file
func (c *Combiner) readModel(filename string) (*models.Model, string, error) {
	zr, err := zip.OpenReader(filename)
	if err != nil {
		return nil, "", fmt.Errorf("error opening file: %w", err)
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
		return nil, "", fmt.Errorf("3D/3dmodel.model not found")
	}

	rc, err := modelFile.Open()
	if err != nil {
		return nil, "", fmt.Errorf("error opening model file: %w", err)
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		return nil, "", fmt.Errorf("error reading model file: %w", err)
	}

	var model models.Model
	if err := xml.Unmarshal(data, &model); err != nil {
		return nil, "", fmt.Errorf("error parsing XML: %w", err)
	}

	return &model, filename, nil
}

// writeModelBambu writes a model to a 3MF file with Bambu Studio support
func (c *Combiner) writeModelBambu(outputFile string, model *models.Model, sourceFile string, scadFiles []models.ScadFile) error {
	// Add Bambu metadata
	addBambuMetadata(model)

	// Read source ZIP to get metadata
	sourceZip, err := zip.OpenReader(sourceFile)
	if err != nil {
		return fmt.Errorf("error opening source file: %w", err)
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

	w, err := outZip.Create("3D/3dmodel.model")
	if err != nil {
		return fmt.Errorf("error creating model entry: %w", err)
	}

	// Write XML declaration
	if _, err := w.Write([]byte(xml.Header)); err != nil {
		return fmt.Errorf("error writing XML header: %w", err)
	}

	if _, err := w.Write(modelXML); err != nil {
		return fmt.Errorf("error writing model XML: %w", err)
	}

	// Write Bambu model settings
	if err := writeModelSettings(outZip, scadFiles); err != nil {
		return fmt.Errorf("error writing model settings: %w", err)
	}

	// Copy other files from source
	for _, file := range sourceZip.File {
		if file.Name == "3D/3dmodel.model" || file.Name == "Metadata/model_settings.config" {
			continue
		}

		srcFile, err := file.Open()
		if err != nil {
			continue
		}

		dst, err := outZip.Create(file.Name)
		if err != nil {
			srcFile.Close()
			continue
		}

		io.Copy(dst, srcFile)
		srcFile.Close()
	}

	return nil
}

// writeModel writes a model to a 3MF file
func (c *Combiner) writeModel(outputFile string, model *models.Model, sourceFile string) error {
	// Read source ZIP to get metadata
	sourceZip, err := zip.OpenReader(sourceFile)
	if err != nil {
		return fmt.Errorf("error opening source file: %w", err)
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

	w, err := outZip.Create("3D/3dmodel.model")
	if err != nil {
		return fmt.Errorf("error creating model entry: %w", err)
	}

	// Write XML declaration
	if _, err := w.Write([]byte(xml.Header)); err != nil {
		return fmt.Errorf("error writing XML header: %w", err)
	}

	if _, err := w.Write(modelXML); err != nil {
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

// getMaxObjectID finds the highest object ID in a model
func (c *Combiner) getMaxObjectID(model *models.Model) int {
	maxID := 0
	for _, obj := range model.Resources.Objects {
		if id, err := strconv.Atoi(obj.ID); err == nil && id > maxID {
			maxID = id
		}
	}
	return maxID
}

// addBambuMetadata adds Bambu Studio specific metadata to a model
func addBambuMetadata(model *models.Model) {
	model.XmlnsBambuStudio = "http://schemas.bambulab.com/package/2021"
	model.XmlnsP = "http://schemas.microsoft.com/3dmanufacturing/production/2015/06"
	model.RequiredExtensions = "p"

	// Add Bambu-specific metadata
	model.Metadata = append([]models.Metadata{
		{Name: "Application", Value: "go3mf"},
		{Name: "BambuStudio:3mfVersion", Value: "1"},
		{Name: "CreationDate", Value: time.Now().Format("2006-01-02")},
		{Name: "ModificationDate", Value: time.Now().Format("2006-01-02")},
	}, model.Metadata...)
}

// writeModelSettings writes the Bambu Studio model_settings.config file
func writeModelSettings(outZip *zip.Writer, scadFiles []models.ScadFile) error {
	// Create parts with filament assignments
	var parts []models.Part
	totalFaces := 0

	for i, scadFile := range scadFiles {
		filamentSlot := scadFile.FilamentSlot
		if filamentSlot == 0 {
			filamentSlot = ((i) % 4) + 1
		}

		faceCount := 12 // Placeholder - would need actual mesh analysis
		totalFaces += faceCount

		parts = append(parts, models.Part{
			ID:      strconv.Itoa(i + 1),
			Subtype: "normal_part",
			Metadata: []models.SettingsMetadata{
				{Key: "name", Value: scadFile.Name},
				{Key: "matrix", Value: "1 0 0 0 0 1 0 0 0 0 1 0 0 0 0 1"},
				{Key: "source_file", Value: "combined.3mf"},
				{Key: "source_object_id", Value: strconv.Itoa(i)},
				{Key: "source_volume_id", Value: "0"},
				{Key: "extruder", Value: strconv.Itoa(filamentSlot)},
			},
			MeshStat: models.MeshStat{
				FaceCount: faceCount,
			},
		})
	}

	parentID := strconv.Itoa(len(scadFiles) + 1)

	settings := models.ModelSettings{
		Objects: []models.SettingsObject{
			{
				ID: parentID,
				Metadata: []models.SettingsMetadata{
					{Key: "name", Value: "combined"},
					{Key: "extruder", Value: "1"},
					{FaceCount: totalFaces},
				},
				Parts: parts,
			},
		},
		Plates: []models.Plate{
			{
				Metadata: []models.SettingsMetadata{
					{Key: "plater_id", Value: "1"},
					{Key: "plater_name", Value: ""},
					{Key: "locked", Value: "false"},
					{Key: "filament_map_mode", Value: "Auto For Flush"},
				},
				ModelInstances: []models.ModelInstance{
					{
						Metadata: []models.SettingsMetadata{
							{Key: "object_id", Value: parentID},
							{Key: "instance_id", Value: "0"},
							{Key: "identify_id", Value: "1"},
						},
					},
				},
			},
		},
		Assemble: models.Assemble{
			Items: []models.AssembleItem{
				{
					ObjectID:   parentID,
					InstanceID: "0",
					Transform:  "1 0 0 0 1 0 0 0 1 0 0 0",
					Offset:     "0 0 0",
				},
			},
		},
	}

	settingsXML, err := xml.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling settings XML: %w", err)
	}

	writer, err := outZip.Create("Metadata/model_settings.config")
	if err != nil {
		return fmt.Errorf("error creating settings entry: %w", err)
	}

	if _, err := writer.Write([]byte(xml.Header)); err != nil {
		return fmt.Errorf("error writing XML header: %w", err)
	}

	if _, err := writer.Write(settingsXML); err != nil {
		return fmt.Errorf("error writing settings XML: %w", err)
	}

	return nil
}
