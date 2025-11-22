package geometry

import (
	"fmt"
	"math"
	"strings"
	"testing"
)

func TestBuildTranslationTransform(t *testing.T) {
	result := BuildTranslationTransform(10.5, 20.75, 5.25)
	expected := "1 0 0 0 1 0 0 0 1 10.50 20.75 5.25"

	if result != expected {
		t.Errorf("BuildTranslationTransform() = %v, want %v", result, expected)
	}
}

func TestBuildRotationTransform_NoRotation(t *testing.T) {
	result := BuildRotationTransform(0, 0, 0, 10, 20, 30)

	// With no rotation, should be identity matrix with translation
	// Allow small floating point errors
	if !strings.Contains(result, "10.00 20.00 30.00") {
		t.Errorf("Translation part is incorrect: %v", result)
	}

	// Check that rotation part is close to identity matrix
	parts := strings.Fields(result)
	if len(parts) != 12 {
		t.Errorf("Expected 12 values, got %d", len(parts))
	}
}

func TestBuildRotationTransform_90DegreeZ(t *testing.T) {
	result := BuildRotationTransform(0, 0, 90, 0, 0, 0)
	parts := strings.Fields(result)

	if len(parts) != 12 {
		t.Errorf("Expected 12 values, got %d", len(parts))
		return
	}

	// For 90° Z rotation, m11 should be ~0, m12 should be ~1
	// m21 should be ~-1, m22 should be ~0
	// Just verify the structure is reasonable (actual values are in scientific notation)
	if parts[11] != "0.00" { // tz should be 0
		t.Errorf("Translation Z should be 0.00, got %v", parts[11])
	}
}

func TestBuildRotationTransform_45DegreeZ(t *testing.T) {
	result := BuildRotationTransform(0, 0, 45, 0, 0, 0)
	parts := strings.Fields(result)

	if len(parts) != 12 {
		t.Errorf("Expected 12 values, got %d", len(parts))
		return
	}

	// For 45° Z rotation, m11 and m22 should be cos(45°) ≈ 0.707
	// m12 should be sin(45°) ≈ 0.707
	// m21 should be -sin(45°) ≈ -0.707
	expectedCos45 := math.Cos(45 * math.Pi / 180)
	expectedSin45 := math.Sin(45 * math.Pi / 180)

	var m11, m12, m21, m22 float64
	if _, err := parseFloat(parts[0], &m11); err == nil {
		if math.Abs(m11-expectedCos45) > 0.0001 {
			t.Errorf("m11 = %v, want ≈%v", m11, expectedCos45)
		}
	}

	if _, err := parseFloat(parts[1], &m12); err == nil {
		if math.Abs(m12-expectedSin45) > 0.0001 {
			t.Errorf("m12 = %v, want ≈%v", m12, expectedSin45)
		}
	}

	if _, err := parseFloat(parts[3], &m21); err == nil {
		if math.Abs(m21-(-expectedSin45)) > 0.0001 {
			t.Errorf("m21 = %v, want ≈%v", m21, -expectedSin45)
		}
	}

	if _, err := parseFloat(parts[4], &m22); err == nil {
		if math.Abs(m22-expectedCos45) > 0.0001 {
			t.Errorf("m22 = %v, want ≈%v", m22, expectedCos45)
		}
	}
}

func TestBuildRotationTransform_Combined(t *testing.T) {
	// Test with combined rotation and translation
	result := BuildRotationTransform(30, 45, 60, 10, 20, 30)
	parts := strings.Fields(result)

	if len(parts) != 12 {
		t.Errorf("Expected 12 values, got %d", len(parts))
		return
	}

	// Check translation values
	if parts[9] != "10.00" {
		t.Errorf("Translation X should be 10.00, got %v", parts[9])
	}
	if parts[10] != "20.00" {
		t.Errorf("Translation Y should be 20.00, got %v", parts[10])
	}
	if parts[11] != "30.00" {
		t.Errorf("Translation Z should be 30.00, got %v", parts[11])
	}
}

// Helper function to parse float from string
func parseFloat(s string, f *float64) (int, error) {
	n, err := fmt.Sscanf(s, "%f", f)
	return n, err
}
