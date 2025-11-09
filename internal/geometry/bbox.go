package geometry

import (
	"encoding/xml"
	"fmt"
	"math"
	"strconv"

	"github.com/philipparndt/go3mf/internal/models"
)

// BoundingBox represents a 3D bounding box
type BoundingBox struct {
	MinX, MinY, MinZ float64
	MaxX, MaxY, MaxZ float64
}

// Width returns the width (X dimension) of the bounding box
func (b *BoundingBox) Width() float64 {
	return b.MaxX - b.MinX
}

// Height returns the height (Y dimension) of the bounding box
func (b *BoundingBox) Height() float64 {
	return b.MaxY - b.MinY
}

// Depth returns the depth (Z dimension) of the bounding box
func (b *BoundingBox) Depth() float64 {
	return b.MaxZ - b.MinZ
}

// Vertex represents a 3D vertex for parsing
type Vertex struct {
	X string `xml:"x,attr"`
	Y string `xml:"y,attr"`
	Z string `xml:"z,attr"`
}

// Vertices represents a collection of vertices
type Vertices struct {
	Vertex []Vertex `xml:"vertex"`
}

// Mesh represents a mesh for parsing
type Mesh struct {
	Vertices Vertices `xml:"vertices"`
}

// CalculateBoundingBox calculates the bounding box of a mesh object
func CalculateBoundingBox(obj *models.Object) (*BoundingBox, error) {
	if obj.Mesh == nil {
		return nil, fmt.Errorf("object has no mesh")
	}

	if obj.Mesh.Vertices == nil {
		return nil, fmt.Errorf("mesh has no vertices")
	}

	// Parse the raw vertices content
	var vertices Vertices
	verticesXML := fmt.Sprintf("<vertices>%s</vertices>", obj.Mesh.Vertices.RawContent)
	if err := xml.Unmarshal([]byte(verticesXML), &vertices); err != nil {
		return nil, fmt.Errorf("failed to parse mesh vertices: %w", err)
	}

	if len(vertices.Vertex) == 0 {
		return nil, fmt.Errorf("mesh has no vertices")
	}

	// Initialize with first vertex
	firstVertex := vertices.Vertex[0]
	x0, err := strconv.ParseFloat(firstVertex.X, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid vertex X coordinate: %w", err)
	}
	y0, err := strconv.ParseFloat(firstVertex.Y, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid vertex Y coordinate: %w", err)
	}
	z0, err := strconv.ParseFloat(firstVertex.Z, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid vertex Z coordinate: %w", err)
	}

	bbox := &BoundingBox{
		MinX: x0,
		MinY: y0,
		MinZ: z0,
		MaxX: x0,
		MaxY: y0,
		MaxZ: z0,
	}

	// Iterate through all vertices to find min/max
	for _, vertex := range vertices.Vertex {
		x, err := strconv.ParseFloat(vertex.X, 64)
		if err != nil {
			continue
		}
		y, err := strconv.ParseFloat(vertex.Y, 64)
		if err != nil {
			continue
		}
		z, err := strconv.ParseFloat(vertex.Z, 64)
		if err != nil {
			continue
		}

		bbox.MinX = math.Min(bbox.MinX, x)
		bbox.MinY = math.Min(bbox.MinY, y)
		bbox.MinZ = math.Min(bbox.MinZ, z)
		bbox.MaxX = math.Max(bbox.MaxX, x)
		bbox.MaxY = math.Max(bbox.MaxY, y)
		bbox.MaxZ = math.Max(bbox.MaxZ, z)
	}

	return bbox, nil
}

// CalculateCombinedBoundingBox calculates the bounding box for multiple objects
// taking into account their transforms
func CalculateCombinedBoundingBox(objects []models.Object, transforms []string) (*BoundingBox, error) {
	if len(objects) == 0 {
		return nil, fmt.Errorf("no objects provided")
	}

	if len(transforms) != len(objects) {
		return nil, fmt.Errorf("number of transforms must match number of objects")
	}

	var combinedBBox *BoundingBox

	for i, obj := range objects {
		bbox, err := CalculateBoundingBox(&obj)
		if err != nil {
			continue // Skip objects without valid meshes
		}

		// Apply transform to bounding box
		transform := transforms[i]
		dx, dy, dz := parseTransform(transform)

		transformedBBox := &BoundingBox{
			MinX: bbox.MinX + dx,
			MinY: bbox.MinY + dy,
			MinZ: bbox.MinZ + dz,
			MaxX: bbox.MaxX + dx,
			MaxY: bbox.MaxY + dy,
			MaxZ: bbox.MaxZ + dz,
		}

		if combinedBBox == nil {
			combinedBBox = transformedBBox
		} else {
			combinedBBox.MinX = math.Min(combinedBBox.MinX, transformedBBox.MinX)
			combinedBBox.MinY = math.Min(combinedBBox.MinY, transformedBBox.MinY)
			combinedBBox.MinZ = math.Min(combinedBBox.MinZ, transformedBBox.MinZ)
			combinedBBox.MaxX = math.Max(combinedBBox.MaxX, transformedBBox.MaxX)
			combinedBBox.MaxY = math.Max(combinedBBox.MaxY, transformedBBox.MaxY)
			combinedBBox.MaxZ = math.Max(combinedBBox.MaxZ, transformedBBox.MaxZ)
		}
	}

	if combinedBBox == nil {
		return nil, fmt.Errorf("no valid objects found")
	}

	return combinedBBox, nil
}

// parseTransform extracts the translation (dx, dy, dz) from a transform matrix
// Transform format: "m11 m12 m13 m21 m22 m23 m31 m32 m33 dx dy dz"
func parseTransform(transform string) (dx, dy, dz float64) {
	var parts [12]float64
	_, err := fmt.Sscanf(transform, "%f %f %f %f %f %f %f %f %f %f %f %f",
		&parts[0], &parts[1], &parts[2],
		&parts[3], &parts[4], &parts[5],
		&parts[6], &parts[7], &parts[8],
		&parts[9], &parts[10], &parts[11])

	if err != nil {
		return 0, 0, 0
	}

	return parts[9], parts[10], parts[11]
}
