package combine

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"

	"github.com/user/go3mf/internal/models"
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

	// Read the base model
	baseModel, baseSourceFile, err := c.readModel(inputFiles[0])
	if err != nil {
		return fmt.Errorf("error reading base file: %w", err)
	}

	// Update base object name from filename
	if len(baseModel.Resources.Objects) > 0 {
		name := filepath.Base(inputFiles[0][:len(inputFiles[0])-len(filepath.Ext(inputFiles[0]))])
		baseModel.Resources.Objects[0].Name = name
	}

	// Combine other models
	maxID := c.getMaxObjectID(baseModel)
	for i := 1; i < len(inputFiles); i++ {
		otherModel, _, err := c.readModel(inputFiles[i])
		if err != nil {
			return fmt.Errorf("error reading file %d (%s): %w", i+1, inputFiles[i], err)
		}

		// Get name from filename
		name := filepath.Base(inputFiles[i][:len(inputFiles[i])-len(filepath.Ext(inputFiles[i]))])

		// Add objects from other model
		for _, obj := range otherModel.Resources.Objects {
			maxID++
			obj.ID = strconv.Itoa(maxID)
			obj.Name = name

			baseModel.Resources.Objects = append(baseModel.Resources.Objects, obj)

			// Add item to build
			baseModel.Build.Items = append(baseModel.Build.Items, models.Item{
				ObjectID:  obj.ID,
				Transform: "1 0 0 0 1 0 0 0 1 0 0 0",
			})
		}
	}

	// Write combined model
	return c.writeModel(outputFile, baseModel, baseSourceFile)
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
