package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/philipparndt/go3mf/internal/models"
	"gopkg.in/yaml.v3"
)

// Loader handles loading and validating YAML configuration files
type Loader struct{}

// NewLoader creates a new config loader
func NewLoader() *Loader {
	return &Loader{}
}

// Load reads and parses a YAML configuration file
func (l *Loader) Load(configPath string) (*models.YamlConfig, error) {
	// Read the config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse YAML
	var config models.YamlConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Validate the configuration
	if err := l.Validate(&config, configPath); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Convert relative paths to absolute paths (relative to config file)
	configDir := filepath.Dir(configPath)
	absConfigDir, err := filepath.Abs(configDir)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path of config directory: %w", err)
	}

	for i := range config.Objects {
		for j := range config.Objects[i].Parts {
			part := &config.Objects[i].Parts[j]
			if !filepath.IsAbs(part.File) {
				part.File = filepath.Join(absConfigDir, part.File)
			}
		}
	}

	return &config, nil
}

// Validate checks if the configuration is valid
func (l *Loader) Validate(config *models.YamlConfig, configPath string) error {
	if config.Output == "" {
		return fmt.Errorf("output file must be specified")
	}

	if len(config.Objects) == 0 {
		return fmt.Errorf("at least one object must be defined")
	}

	configDir := filepath.Dir(configPath)

	for i, obj := range config.Objects {
		if obj.Name == "" {
			return fmt.Errorf("object %d: name is required", i)
		}

		if len(obj.Parts) == 0 {
			return fmt.Errorf("object %s: at least one part must be defined", obj.Name)
		}

		for j, part := range obj.Parts {
			if part.Name == "" {
				return fmt.Errorf("object %s, part %d: name is required", obj.Name, j)
			}

			if part.File == "" {
				return fmt.Errorf("object %s, part %s: file is required", obj.Name, part.Name)
			}

			// Check if file exists (handle relative paths)
			filePath := part.File
			if !filepath.IsAbs(filePath) {
				filePath = filepath.Join(configDir, filePath)
			}

			if _, err := os.Stat(filePath); err != nil {
				return fmt.Errorf("object %s, part %s: file not found: %s", obj.Name, part.Name, part.File)
			}

			// Validate filament slot
			if part.Filament < 0 || part.Filament > 4 {
				return fmt.Errorf("object %s, part %s: filament must be 0-4 (0=auto, 1-4=AMS slots)", obj.Name, part.Name)
			}
		}
	}

	return nil
}

// convertMapToScadFunctions converts a map of key-value pairs to SCAD function definitions
// Example: {"h": 6, "width": 38} -> "function get_h() = 6;\nfunction get_width() = 38;\n"
func convertMapToScadFunctions(configMap map[string]interface{}) string {
	var builder strings.Builder

	for key, value := range configMap {
		builder.WriteString(fmt.Sprintf("function get_%s() = ", key))

		// Format the value based on its type
		switch v := value.(type) {
		case string:
			// String values need to be quoted
			builder.WriteString(fmt.Sprintf("\"%s\"", v))
		case int, int8, int16, int32, int64:
			builder.WriteString(fmt.Sprintf("%d", v))
		case float32, float64:
			builder.WriteString(fmt.Sprintf("%g", v))
		case bool:
			if v {
				builder.WriteString("true")
			} else {
				builder.WriteString("false")
			}
		default:
			// For any other type, use fmt.Sprintf which should handle most cases
			builder.WriteString(fmt.Sprintf("%v", v))
		}

		builder.WriteString(";\n")
	}

	return builder.String()
}

// convertConfigContent converts a config value to a SCAD string
// Handles both old format (string) and new format (map)
func convertConfigContent(content interface{}) string {
	switch v := content.(type) {
	case string:
		// Direct string content (old format)
		return v
	case map[string]interface{}:
		// Map format (new format) - convert to SCAD functions
		return convertMapToScadFunctions(v)
	case map[interface{}]interface{}:
		// YAML might parse it as map[interface{}]interface{}, convert it
		converted := make(map[string]interface{})
		for k, val := range v {
			if strKey, ok := k.(string); ok {
				converted[strKey] = val
			}
		}
		return convertMapToScadFunctions(converted)
	default:
		// Fallback: treat as string
		return fmt.Sprintf("%v", v)
	}
}

// ConvertToScadFiles converts YAML config to ScadFile list for backward compatibility
func (l *Loader) ConvertToScadFiles(config *models.YamlConfig) []models.ScadFile {
	var scadFiles []models.ScadFile

	for _, obj := range config.Objects {
		// Determine how many copies of this object to create
		count := obj.Count
		if count < 1 {
			count = 1
		}

		for copyIdx := 0; copyIdx < count; copyIdx++ {
			for _, part := range obj.Parts {
				// Create a composite name: object_name/part_name
				// Add copy number suffix if count > 1
				objName := obj.Name
				if count > 1 {
					objName = fmt.Sprintf("%s_%d", obj.Name, copyIdx+1)
				}

				compositeName := objName
				if len(obj.Parts) > 1 {
					compositeName = objName + "/" + part.Name
				}

				// Combine object-level and part-level config files
				// Part-level config takes precedence (overrides object-level for same filename)
				configFiles := make(map[string]string)

				// Start with object-level configs
				for _, configMap := range obj.Config {
					for filename, content := range configMap {
						configFiles[filename] = convertConfigContent(content)
					}
				}

				// Override with part-level configs
				for _, configMap := range part.Config {
					for filename, content := range configMap {
						configFiles[filename] = convertConfigContent(content)
					}
				}

				scadFiles = append(scadFiles, models.ScadFile{
					Path:         part.File,
					Name:         compositeName,
					FilamentSlot: part.Filament,
					ConfigFiles:  configFiles,
					RotationX:    part.RotationX,
					RotationY:    part.RotationY,
					RotationZ:    part.RotationZ,
					PositionX:    part.PositionX,
					PositionY:    part.PositionY,
					PositionZ:    part.PositionZ,
				})
			}
		}
	}

	return scadFiles
}

// ConvertToObjectGroups converts YAML config to ObjectGroup list with normalization settings
func (l *Loader) ConvertToObjectGroups(config *models.YamlConfig) []models.ObjectGroup {
	var objectGroups []models.ObjectGroup

	for _, obj := range config.Objects {
		// Default normalize_position to true if not specified
		normalizePosition := true
		if obj.NormalizePosition != nil {
			normalizePosition = *obj.NormalizePosition
		}

		// Determine how many copies of this object to create
		count := obj.Count
		if count < 1 {
			count = 1
		}

		for copyIdx := 0; copyIdx < count; copyIdx++ {
			// Generate object name with copy number suffix if count > 1
			objName := obj.Name
			if count > 1 {
				objName = fmt.Sprintf("%s_%d", obj.Name, copyIdx+1)
			}

			var parts []models.ScadFile
			for _, part := range obj.Parts {
				// Create a composite name: object_name/part_name
				compositeName := objName
				if len(obj.Parts) > 1 {
					compositeName = objName + "/" + part.Name
				}

				// Combine object-level and part-level config files
				configFiles := make(map[string]string)

				// Start with object-level configs
				for _, configMap := range obj.Config {
					for filename, content := range configMap {
						configFiles[filename] = convertConfigContent(content)
					}
				}

				// Override with part-level configs
				for _, configMap := range part.Config {
					for filename, content := range configMap {
						configFiles[filename] = convertConfigContent(content)
					}
				}

				parts = append(parts, models.ScadFile{
					Path:         part.File,
					Name:         compositeName,
					FilamentSlot: part.Filament,
					ConfigFiles:  configFiles,
					RotationX:    part.RotationX,
					RotationY:    part.RotationY,
					RotationZ:    part.RotationZ,
					PositionX:    part.PositionX,
					PositionY:    part.PositionY,
					PositionZ:    part.PositionZ,
				})
			}

			objectGroups = append(objectGroups, models.ObjectGroup{
				Name:              objName,
				Parts:             parts,
				NormalizePosition: normalizePosition,
			})
		}
	}

	return objectGroups
}

