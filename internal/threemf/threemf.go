package threemf

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/user/go3mf/internal/models"
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
func (w *Writer) WriteBambu(outputFile string, model *models.Model, sourceFile string, scadFiles []models.ScadFile) error {
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
	if err := WriteModelSettings(outZip, scadFiles); err != nil {
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
			allObjects = append(allObjects, obj)
		}
	}

	// Create a parent object with components
	var components []models.Component
	for i := range allObjects {
		components = append(components, models.Component{
			ObjectID:  strconv.Itoa(i + 1),
			Transform: "1 0 0 0 1 0 0 0 1 0 0 0",
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

	// Write combined model to output file with Bambu support
	return c.writer.WriteBambu(outputFile, combinedModel, tempFiles[0], scadFiles)
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
