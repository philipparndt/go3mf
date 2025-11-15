package config

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestAllExamplesLoadSuccessfully tests that all example YAML files can be loaded and validated
func TestAllExamplesLoadSuccessfully(t *testing.T) {
	examples := []struct {
		name string
		file string
	}{
		{"simple config", "../../example/simple-config.yaml"},
		{"advanced config", "../../example/advanced-config.yaml"},
		{"packing config", "../../example/packing-config.yaml"},
		{"plate config", "../../example/plate-config.yaml"},
		{"simple text demo", "../../example/simple-text-demo.yaml"},
		{"config formats demo", "../../example/config-formats-demo.yaml"},
	}

	loader := NewLoader()

	for _, tt := range examples {
		t.Run(tt.name, func(t *testing.T) {
			// Get absolute path for the example file
			absPath, err := filepath.Abs(tt.file)
			if err != nil {
				t.Fatalf("Failed to get absolute path: %v", err)
			}

			config, err := loader.Load(absPath)
			if err != nil {
				t.Fatalf("Failed to load %s: %v", tt.name, err)
			}

			if config == nil {
				t.Fatalf("Config is nil for %s", tt.name)
			}

			if config.Output == "" {
				t.Errorf("Output file not specified in %s", tt.name)
			}

			if len(config.Objects) == 0 {
				t.Errorf("No objects defined in %s", tt.name)
			}
		})
	}
}

// TestConfigFormatsDemoExample tests the config-formats-demo.yaml example specifically
func TestConfigFormatsDemoExample(t *testing.T) {
	loader := NewLoader()
	absPath, _ := filepath.Abs("../../example/config-formats-demo.yaml")

	config, err := loader.Load(absPath)
	if err != nil {
		t.Fatalf("Failed to load config-formats-demo.yaml: %v", err)
	}

	// Should have 4 objects
	if len(config.Objects) != 4 {
		t.Fatalf("Expected 4 objects, got %d", len(config.Objects))
	}

	// Convert to SCAD files
	scadFiles := loader.ConvertToScadFiles(config)

	// Test 1: NewFormatExample - map-based format
	t.Run("NewFormatExample", func(t *testing.T) {
		// Find the NewFormatExample scad file
		var newFormatFile string
		found := false
		for _, sf := range scadFiles {
			if sf.Name == "NewFormatExample" {
				newFormatFile = sf.ConfigFiles["cfg.scad"]
				found = true
				break
			}
		}

		if !found {
			t.Fatal("NewFormatExample not found in scad files")
		}

		// Verify all expected function definitions are present
		expectedFunctions := []string{
			"function get_h() = 6;",
			"function get_width() = 38;",
			"function get_length() = 90;",
			"function get_diameter() = 16;",
			"function get_distance() = 48.5;",
			"function get_nutDistance() = 6;",
			"function get_separation() = 10;",
			"function get_offset_l() = -0.75;",
			"function get_offset_r() = 0.75;",
		}

		for _, expected := range expectedFunctions {
			if !strings.Contains(newFormatFile, expected) {
				t.Errorf("Expected to find '%s' in config, got:\n%s", expected, newFormatFile)
			}
		}
	})

	// Test 2: OldFormatExample - string-based format
	t.Run("OldFormatExample", func(t *testing.T) {
		var oldFormatFile string
		found := false
		for _, sf := range scadFiles {
			if sf.Name == "OldFormatExample" {
				oldFormatFile = sf.ConfigFiles["cfg.scad"]
				found = true
				break
			}
		}

		if !found {
			t.Fatal("OldFormatExample not found in scad files")
		}

		// Verify the old format content is preserved exactly
		if !strings.Contains(oldFormatFile, "function get_h() = 6;") {
			t.Errorf("Old format not preserved correctly")
		}
	})

	// Test 3: MixedFormatExample - both formats in same object
	t.Run("MixedFormatExample", func(t *testing.T) {
		var globalScad, cfgScad string
		found := false

		for _, sf := range scadFiles {
			if sf.Name == "MixedFormatExample" {
				globalScad = sf.ConfigFiles["global.scad"]
				cfgScad = sf.ConfigFiles["cfg.scad"]
				found = true
				break
			}
		}

		if !found {
			t.Fatal("MixedFormatExample not found in scad files")
		}

		// Verify global.scad has old format
		if !strings.Contains(globalScad, "function global_scale() = 1.0;") {
			t.Errorf("global.scad should contain old format content")
		}

		// Verify cfg.scad has new format conversions
		if !strings.Contains(cfgScad, "function get_h() = 8;") {
			t.Errorf("cfg.scad should contain new format converted content")
		}
		if !strings.Contains(cfgScad, "function get_width() = 50;") {
			t.Errorf("cfg.scad should contain new format converted content")
		}
	})

	// Test 4: DataTypesExample - different data types
	t.Run("DataTypesExample", func(t *testing.T) {
		var dataTypesFile string
		found := false
		for _, sf := range scadFiles {
			if sf.Name == "DataTypesExample" {
				dataTypesFile = sf.ConfigFiles["cfg.scad"]
				found = true
				break
			}
		}

		if !found {
			t.Fatal("DataTypesExample not found in scad files")
		}

		// Test different data types
		tests := []struct {
			desc     string
			expected string
		}{
			{"integer", "function get_count() = 5;"},
			{"float", "function get_height() = 12.5;"},
			{"negative float", "function get_offset_x() = -2.5;"},
			{"string", "function get_label() = \"Demo Part\";"},
			{"boolean true", "function get_rounded() = true;"},
			{"boolean false", "function get_hollow() = false;"},
		}

		for _, tt := range tests {
			if !strings.Contains(dataTypesFile, tt.expected) {
				t.Errorf("%s: expected to find '%s' in config", tt.desc, tt.expected)
			}
		}
	})
}

// TestPlateConfigExample tests the updated plate-config.yaml with new format
func TestPlateConfigExample(t *testing.T) {
	loader := NewLoader()
	absPath, _ := filepath.Abs("../../example/plate-config.yaml")

	config, err := loader.Load(absPath)
	if err != nil {
		t.Fatalf("Failed to load plate-config.yaml: %v", err)
	}

	scadFiles := loader.ConvertToScadFiles(config)

	// Should have 2 parts (base_plate and text_overlay)
	if len(scadFiles) != 2 {
		t.Fatalf("Expected 2 scad files, got %d", len(scadFiles))
	}

	// Test base_plate - should inherit object-level config
	t.Run("base_plate config", func(t *testing.T) {
		var basePlate string
		found := false
		for _, sf := range scadFiles {
			if strings.Contains(sf.Name, "base_plate") {
				basePlate = sf.ConfigFiles["cfg.scad"]
				found = true
				break
			}
		}

		if !found {
			t.Fatal("base_plate not found")
		}

		// Should have object-level config converted from map format
		expectedFunctions := []string{
			"function get_plate_width() = 120;",
			"function get_plate_height() = 80;",
			"function get_plate_thickness() = 4;",
			"function get_text_content() = \"Go3MF\";",
		}

		for _, expected := range expectedFunctions {
			if !strings.Contains(basePlate, expected) {
				t.Errorf("Expected to find '%s' in base_plate config", expected)
			}
		}
	})

	// Test text_overlay - part config should override object config
	t.Run("text_overlay config override", func(t *testing.T) {
		var textOverlay string
		found := false
		for _, sf := range scadFiles {
			if strings.Contains(sf.Name, "text_overlay") {
				textOverlay = sf.ConfigFiles["cfg.scad"]
				found = true
				break
			}
		}

		if !found {
			t.Fatal("text_overlay not found")
		}

		// Should have overridden values from part-level config
		if !strings.Contains(textOverlay, "function get_text_content() = \"DEMO\";") {
			t.Errorf("Part-level config should override object-level for text_content")
		}
		if !strings.Contains(textOverlay, "function get_text_size() = 20;") {
			t.Errorf("Part-level config should set text_size")
		}

		// Should NOT have the object-level value for overridden keys
		if strings.Contains(textOverlay, "function get_text_content() = \"Go3MF\";") {
			t.Errorf("Object-level text_content should be overridden")
		}
	})
}

// TestNewFormatVsOldFormatEquivalence verifies that both formats produce equivalent results
func TestNewFormatVsOldFormatEquivalence(t *testing.T) {
	loader := NewLoader()
	absPath, _ := filepath.Abs("../../example/config-formats-demo.yaml")

	config, err := loader.Load(absPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	scadFiles := loader.ConvertToScadFiles(config)

	// Find NewFormatExample and OldFormatExample
	var newFormat, oldFormat string
	foundNew, foundOld := false, false
	for _, sf := range scadFiles {
		if sf.Name == "NewFormatExample" {
			newFormat = sf.ConfigFiles["cfg.scad"]
			foundNew = true
		}
		if sf.Name == "OldFormatExample" {
			oldFormat = sf.ConfigFiles["cfg.scad"]
			foundOld = true
		}
	}

	if !foundNew || !foundOld {
		t.Fatal("Could not find both format examples")
	}

	// Both should contain the same function definitions (order might differ)
	expectedFunctions := []string{
		"function get_h() = 6;",
		"function get_width() = 38;",
		"function get_length() = 90;",
		"function get_diameter() = 16;",
		"function get_distance() = 48.5;",
		"function get_nutDistance() = 6;",
		"function get_separation() = 10;",
		"function get_offset_l() = -0.75;",
		"function get_offset_r() = 0.75;",
	}

	for _, fn := range expectedFunctions {
		if !strings.Contains(newFormat, fn) {
			t.Errorf("New format missing: %s", fn)
		}
		if !strings.Contains(oldFormat, fn) {
			t.Errorf("Old format missing: %s", fn)
		}
	}
}

// TestExampleFilaments verifies filament assignments in examples
func TestExampleFilaments(t *testing.T) {
	loader := NewLoader()
	absPath, _ := filepath.Abs("../../example/config-formats-demo.yaml")

	config, err := loader.Load(absPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	scadFiles := loader.ConvertToScadFiles(config)

	// Verify filament assignments
	expectedFilaments := map[string]int{
		"NewFormatExample":  1,
		"OldFormatExample":  2,
		"MixedFormatExample": 3,
		"DataTypesExample":  4,
	}

	for name, expectedFilament := range expectedFilaments {
		found := false
		for _, sf := range scadFiles {
			if sf.Name == name {
				found = true
				if sf.FilamentSlot != expectedFilament {
					t.Errorf("%s: expected filament %d, got %d", name, expectedFilament, sf.FilamentSlot)
				}
				break
			}
		}
		if !found {
			t.Errorf("Example %s not found in scad files", name)
		}
	}
}
