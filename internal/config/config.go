package config

import (
	"fmt"
	"os"
	"path/filepath"

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

// ConvertToScadFiles converts YAML config to ScadFile list for backward compatibility
func (l *Loader) ConvertToScadFiles(config *models.YamlConfig) []models.ScadFile {
	var scadFiles []models.ScadFile

	for _, obj := range config.Objects {
		for _, part := range obj.Parts {
			// Create a composite name: object_name/part_name
			compositeName := obj.Name
			if len(obj.Parts) > 1 {
				compositeName = obj.Name + "/" + part.Name
			}

			// Combine object-level and part-level config files
			// Part-level config takes precedence (overrides object-level for same filename)
			configFiles := make(map[string]string)

			// Start with object-level configs
			for _, configMap := range obj.Config {
				for filename, content := range configMap {
					configFiles[filename] = content
				}
			}

			// Override with part-level configs
			for _, configMap := range part.Config {
				for filename, content := range configMap {
					configFiles[filename] = content
				}
			}

			scadFiles = append(scadFiles, models.ScadFile{
				Path:         part.File,
				Name:         compositeName,
				FilamentSlot: part.Filament,
				ConfigFiles:  configFiles,
			})
		}
	}

	return scadFiles
}
