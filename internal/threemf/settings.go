package threemf

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"strconv"
	"time"

	"github.com/philipparndt/go3mf/internal/models"
)

// WriteModelSettings writes the Bambu Studio model_settings.config file
func WriteModelSettings(outZip *zip.Writer, objectGroups []models.ObjectGroup, buildItems []models.Item) error {
	var settingsObjects []models.SettingsObject
	var modelInstances []models.ModelInstance
	var assembleItems []models.AssembleItem
	partID := 1
	sourceObjectID := 0

	// Create settings object for each group
	for _, group := range objectGroups {
		var parts []models.Part
		totalFaces := 0

		for volumeIndex, scadFile := range group.Parts {
			filamentSlot := scadFile.FilamentSlot
			if filamentSlot == 0 {
				filamentSlot = ((partID - 1) % 4) + 1
			}

			faceCount := 12 // Placeholder - would need actual mesh analysis
			totalFaces += faceCount

			// Build metadata list
			metadata := []models.SettingsMetadata{
				{Key: "name", Value: scadFile.Name},
				{Key: "matrix", Value: "1 0 0 0 0 1 0 0 0 0 1 0 0 0 0 1"},
				{Key: "source_file", Value: "combined.3mf"},
				{Key: "source_object_id", Value: strconv.Itoa(sourceObjectID)},
				{Key: "source_volume_id", Value: strconv.Itoa(volumeIndex)},
			}

			// Only add extruder metadata if not using default filament (1)
			if filamentSlot != 1 {
				metadata = append(metadata, models.SettingsMetadata{
					Key:   "extruder",
					Value: strconv.Itoa(filamentSlot),
				})
			}

			parts = append(parts, models.Part{
				ID:       strconv.Itoa(partID),
				Subtype:  "normal_part",
				Metadata: metadata,
				MeshStat: models.MeshStat{
					FaceCount: faceCount,
				},
			})
			partID++
		}
		sourceObjectID++

		settingsObjects = append(settingsObjects, models.SettingsObject{
			ID: group.ID,
			Metadata: []models.SettingsMetadata{
				{Key: "name", Value: group.Name},
				{Key: "extruder", Value: "1"},
				{FaceCount: totalFaces},
			},
			Parts: parts,
		})

		modelInstances = append(modelInstances, models.ModelInstance{
			Metadata: []models.SettingsMetadata{
				{Key: "object_id", Value: group.ID},
				{Key: "instance_id", Value: "0"},
				{Key: "identify_id", Value: group.ID},
			},
		})
	}

	// Create assemble items from build items
	for _, item := range buildItems {
		assembleItems = append(assembleItems, models.AssembleItem{
			ObjectID:   item.ObjectID,
			InstanceID: "0",
			Transform:  item.Transform,
			Offset:     "0 0 0",
		})
	}

	settings := models.ModelSettings{
		Objects: settingsObjects,
		Plate: models.Plate{
			Metadata: []models.SettingsMetadata{
				{Key: "plater_id", Value: "1"},
				{Key: "plater_name", Value: ""},
				{Key: "locked", Value: "false"},
				{Key: "filament_map_mode", Value: "Auto For Flush"},
			},
			ModelInstances: modelInstances,
		},
		Assemble: models.Assemble{
			Items: assembleItems,
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
