package config

import (
	"strings"
	"testing"

	"github.com/philipparndt/go3mf/internal/models"
)

// TestConvertMapToScadFunctions tests the conversion of key-value maps to SCAD function definitions
func TestConvertMapToScadFunctions(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		expected map[string]bool // Map of expected strings (using map for unordered check)
	}{
		{
			name: "integers",
			input: map[string]interface{}{
				"h":      6,
				"width":  38,
				"length": 90,
			},
			expected: map[string]bool{
				"function get_h() = 6;":      true,
				"function get_width() = 38;": true,
				"function get_length() = 90;": true,
			},
		},
		{
			name: "floats",
			input: map[string]interface{}{
				"distance":    48.5,
				"offset_l":    -0.75,
				"offset_r":    0.75,
			},
			expected: map[string]bool{
				"function get_distance() = 48.5;": true,
				"function get_offset_l() = -0.75;": true,
				"function get_offset_r() = 0.75;": true,
			},
		},
		{
			name: "strings",
			input: map[string]interface{}{
				"name":  "test",
				"color": "red",
			},
			expected: map[string]bool{
				"function get_name() = \"test\";": true,
				"function get_color() = \"red\";": true,
			},
		},
		{
			name: "booleans",
			input: map[string]interface{}{
				"enabled":  true,
				"disabled": false,
			},
			expected: map[string]bool{
				"function get_enabled() = true;":  true,
				"function get_disabled() = false;": true,
			},
		},
		{
			name: "mixed types",
			input: map[string]interface{}{
				"h":        6,
				"diameter": 16.5,
				"name":     "holder",
				"active":   true,
			},
			expected: map[string]bool{
				"function get_h() = 6;":                true,
				"function get_diameter() = 16.5;":      true,
				"function get_name() = \"holder\";":    true,
				"function get_active() = true;":        true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertMapToScadFunctions(tt.input)

			// Split result into lines and check each one
			lines := strings.Split(strings.TrimSpace(result), "\n")

			if len(lines) != len(tt.expected) {
				t.Errorf("Expected %d functions, got %d", len(tt.expected), len(lines))
				t.Logf("Result: %s", result)
				return
			}

			// Check that all expected lines are present
			for _, line := range lines {
				if !tt.expected[line] {
					t.Errorf("Unexpected line: %s", line)
				}
			}
		})
	}
}

// TestConvertToScadFiles_OldFormat tests backward compatibility with the old string format
func TestConvertToScadFiles_OldFormat(t *testing.T) {
	loader := NewLoader()

	config := &models.YamlConfig{
		Output: "test.3mf",
		Objects: []models.YamlObject{
			{
				Name: "TestObject",
				Parts: []models.YamlPart{
					{
						Name: "part1",
						File: "test.scad",
						Config: []map[string]interface{}{
							{
								"cfg.scad": "function get_h() = 6;\nfunction get_width() = 38;\n",
							},
						},
					},
				},
			},
		},
	}

	scadFiles := loader.ConvertToScadFiles(config)

	if len(scadFiles) != 1 {
		t.Fatalf("Expected 1 scad file, got %d", len(scadFiles))
	}

	if scadFiles[0].ConfigFiles["cfg.scad"] != "function get_h() = 6;\nfunction get_width() = 38;\n" {
		t.Errorf("Config content mismatch: %s", scadFiles[0].ConfigFiles["cfg.scad"])
	}
}

// TestConvertToScadFiles_NewFormat tests the new map-based format
func TestConvertToScadFiles_NewFormat(t *testing.T) {
	loader := NewLoader()

	config := &models.YamlConfig{
		Output: "test.3mf",
		Objects: []models.YamlObject{
			{
				Name: "TestObject",
				Parts: []models.YamlPart{
					{
						Name: "part1",
						File: "test.scad",
						Config: []map[string]interface{}{
							{
								"cfg.scad": map[string]interface{}{
									"h":      6,
									"width":  38,
									"length": 90,
								},
							},
						},
					},
				},
			},
		},
	}

	scadFiles := loader.ConvertToScadFiles(config)

	if len(scadFiles) != 1 {
		t.Fatalf("Expected 1 scad file, got %d", len(scadFiles))
	}

	content := scadFiles[0].ConfigFiles["cfg.scad"]

	// Check that the content contains the expected function definitions
	expectedFunctions := []string{
		"function get_h() = 6;",
		"function get_width() = 38;",
		"function get_length() = 90;",
	}

	for _, expected := range expectedFunctions {
		if !strings.Contains(content, expected) {
			t.Errorf("Expected content to contain '%s', got: %s", expected, content)
		}
	}
}

// TestConvertToScadFiles_MixedFormats tests using both formats in the same config
func TestConvertToScadFiles_MixedFormats(t *testing.T) {
	loader := NewLoader()

	config := &models.YamlConfig{
		Output: "test.3mf",
		Objects: []models.YamlObject{
			{
				Name: "TestObject",
				Config: []map[string]interface{}{
					{
						// Old format at object level
						"global.scad": "function global_value() = 100;\n",
					},
				},
				Parts: []models.YamlPart{
					{
						Name: "part1",
						File: "test.scad",
						Config: []map[string]interface{}{
							{
								// New format at part level
								"cfg.scad": map[string]interface{}{
									"h":      6,
									"width":  38,
								},
							},
						},
					},
				},
			},
		},
	}

	scadFiles := loader.ConvertToScadFiles(config)

	if len(scadFiles) != 1 {
		t.Fatalf("Expected 1 scad file, got %d", len(scadFiles))
	}

	// Check global config (old format)
	globalContent := scadFiles[0].ConfigFiles["global.scad"]
	if globalContent != "function global_value() = 100;\n" {
		t.Errorf("Global config mismatch: %s", globalContent)
	}

	// Check part config (new format)
	partContent := scadFiles[0].ConfigFiles["cfg.scad"]
	if !strings.Contains(partContent, "function get_h() = 6;") {
		t.Errorf("Part config should contain 'function get_h() = 6;', got: %s", partContent)
	}
	if !strings.Contains(partContent, "function get_width() = 38;") {
		t.Errorf("Part config should contain 'function get_width() = 38;', got: %s", partContent)
	}
}

// TestConvertToScadFiles_PartOverridesObject tests that part-level config overrides object-level
func TestConvertToScadFiles_PartOverridesObject(t *testing.T) {
	loader := NewLoader()

	config := &models.YamlConfig{
		Output: "test.3mf",
		Objects: []models.YamlObject{
			{
				Name: "TestObject",
				Config: []map[string]interface{}{
					{
						"cfg.scad": map[string]interface{}{
							"h": 10,
						},
					},
				},
				Parts: []models.YamlPart{
					{
						Name: "part1",
						File: "test.scad",
						Config: []map[string]interface{}{
							{
								"cfg.scad": map[string]interface{}{
									"h": 6, // Should override the object-level value
								},
							},
						},
					},
				},
			},
		},
	}

	scadFiles := loader.ConvertToScadFiles(config)

	if len(scadFiles) != 1 {
		t.Fatalf("Expected 1 scad file, got %d", len(scadFiles))
	}

	content := scadFiles[0].ConfigFiles["cfg.scad"]

	// Should contain the part-level value (6), not the object-level value (10)
	if !strings.Contains(content, "function get_h() = 6;") {
		t.Errorf("Expected part-level config to override object-level, got: %s", content)
	}
	if strings.Contains(content, "function get_h() = 10;") {
		t.Errorf("Object-level config should be overridden, got: %s", content)
	}
}
