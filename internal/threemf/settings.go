package threemf

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"strconv"
	"time"

	"github.com/user/go3mf/internal/models"
)

// WriteModelSettings writes the Bambu Studio model_settings.config file
func WriteModelSettings(outZip *zip.Writer, scadFiles []models.ScadFile) error {
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
		Object: models.SettingsObject{
			ID: parentID,
			Metadata: []models.SettingsMetadata{
				{Key: "name", Value: "combined"},
				{Key: "extruder", Value: "1"},
				{FaceCount: totalFaces},
			},
			Parts: parts,
		},
		Plate: models.Plate{
			Metadata: []models.SettingsMetadata{
				{Key: "plater_id", Value: "1"},
				{Key: "plater_name", Value: ""},
				{Key: "locked", Value: "false"},
				{Key: "filament_map_mode", Value: "Auto For Flush"},
			},
			ModelInstance: models.ModelInstance{
				Metadata: []models.SettingsMetadata{
					{Key: "object_id", Value: parentID},
					{Key: "instance_id", Value: "0"},
					{Key: "identify_id", Value: "1"},
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

// AddBambuMetadata adds Bambu Studio specific metadata to a model
func AddBambuMetadata(model *models.Model) {
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
