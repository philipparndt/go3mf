package geometry

import (
	"fmt"
	"math"
)

// BuildRotationTransform creates a 3MF transformation matrix string with rotation and translation.
// The transformation matrix format is: m11 m12 m13 m21 m22 m23 m31 m32 m33 tx ty tz
// Rotations are applied in the order: Z, Y, X (intrinsic rotations)
// This matches the typical 3D transformation pipeline.
func BuildRotationTransform(rotX, rotY, rotZ, tx, ty, tz float64) string {
	// Convert degrees to radians
	rx := rotX * math.Pi / 180.0
	ry := rotY * math.Pi / 180.0
	rz := rotZ * math.Pi / 180.0

	// Calculate sin and cos values
	cosX, sinX := math.Cos(rx), math.Sin(rx)
	cosY, sinY := math.Cos(ry), math.Sin(ry)
	cosZ, sinZ := math.Cos(rz), math.Sin(rz)

	// Build combined rotation matrix (Z * Y * X)
	// This is the standard rotation order used in 3D graphics
	m11 := cosY * cosZ
	m12 := cosY * sinZ
	m13 := -sinY

	m21 := sinX*sinY*cosZ - cosX*sinZ
	m22 := sinX*sinY*sinZ + cosX*cosZ
	m23 := sinX * cosY

	m31 := cosX*sinY*cosZ + sinX*sinZ
	m32 := cosX*sinY*sinZ - sinX*cosZ
	m33 := cosX * cosY

	// Format as 3MF transformation matrix string
	// Use %.8f for precision to avoid rounding errors
	return fmt.Sprintf("%.8f %.8f %.8f %.8f %.8f %.8f %.8f %.8f %.8f %.2f %.2f %.2f",
		m11, m12, m13,
		m21, m22, m23,
		m31, m32, m33,
		tx, ty, tz)
}

// BuildTranslationTransform creates a simple translation transformation matrix (no rotation)
func BuildTranslationTransform(tx, ty, tz float64) string {
	return fmt.Sprintf("1 0 0 0 1 0 0 0 1 %.2f %.2f %.2f", tx, ty, tz)
}
