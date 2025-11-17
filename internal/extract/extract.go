package extract

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/philipparndt/go3mf/internal/models"
	"github.com/philipparndt/go3mf/internal/stl"
	"github.com/philipparndt/go3mf/internal/ui"
)

// Extractor extracts 3D models from 3MF files
type Extractor struct {
	stlWriter *stl.Writer
}

// NewExtractor creates a new Extractor
func NewExtractor() *Extractor {
	return &Extractor{
		stlWriter: stl.NewWriter(),
	}
}

// Vertex represents a 3D vertex
type Vertex struct {
	X, Y, Z float32
}

// Triangle represents a triangle by vertex indices
type Triangle struct {
	V1, V2, V3 int
}

// ParsedMesh represents a parsed mesh with vertices and triangles
type ParsedMesh struct {
	Vertices  []Vertex
	Triangles []Triangle
}

// Extract extracts all 3D models from a 3MF file to STL files
func (e *Extractor) Extract(filename string, outputDir string, binary bool) error {
	// Create output directory if it doesn't exist
	if err := ensureDir(outputDir); err != nil {
		return fmt.Errorf("error creating output directory: %w", err)
	}

	// Open the 3MF file
	zr, err := zip.OpenReader(filename)
	if err != nil {
		return fmt.Errorf("error opening 3MF file: %w", err)
	}
	defer zr.Close()

	// Find and read the model file
	var modelFile *zip.File
	for _, f := range zr.File {
		if f.Name == "3D/3dmodel.model" {
			modelFile = f
			break
		}
	}

	if modelFile == nil {
		return fmt.Errorf("3D/3dmodel.model not found in archive")
	}

	rc, err := modelFile.Open()
	if err != nil {
		return fmt.Errorf("error opening model file: %w", err)
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		return fmt.Errorf("error reading model file: %w", err)
	}

	var model models.Model
	if err := xml.Unmarshal(data, &model); err != nil {
		return fmt.Errorf("error parsing XML: %w", err)
	}

	// Read object names from model_settings.config if available
	objectNames := e.readObjectNames(&zr.Reader)

	// Extract each mesh object
	extractedCount := 0
	for _, obj := range model.Resources.Objects {
		// Get the object name from settings if available
		objectName := obj.Name
		if settingsName, ok := objectNames[obj.ID]; ok && settingsName != "" {
			objectName = settingsName
		}

		// Check if object has a direct mesh
		if obj.Mesh != nil && obj.Mesh.Vertices != nil && obj.Mesh.Triangles != nil {
			if err := e.extractMesh(objectName, obj.ID, obj.Mesh, outputDir, binary, extractedCount); err != nil {
				ui.PrintError(fmt.Sprintf("Error extracting mesh for object %s (ID: %s): %v", objectName, obj.ID, err))
				continue
			}
			extractedCount++
		} else if obj.Components != nil && len(obj.Components.Component) > 0 {
			// Object has components - need to look up referenced models
			for compIdx, comp := range obj.Components.Component {
				// Check if component references an external model file
				if comp.Path != "" {
					// Read the external model file
					externalMesh, externalName, err := e.readExternalModel(&zr.Reader, comp.Path)
					if err != nil {
						ui.PrintError(fmt.Sprintf("Error reading external model %s: %v", comp.Path, err))
						continue
					}

					// Generate a name for this component
					name := objectName
					if name == "" {
						name = fmt.Sprintf("object_%s_component_%d", obj.ID, compIdx)
					} else if len(obj.Components.Component) > 1 {
						// Use part name from external model if available
						if externalName != "" {
							name = externalName
						} else {
							name = fmt.Sprintf("%s_part_%d", name, compIdx+1)
						}
					}

					if err := e.extractMesh(name, obj.ID, externalMesh, outputDir, binary, extractedCount); err != nil {
						ui.PrintError(fmt.Sprintf("Error extracting component mesh: %v", err))
						continue
					}
					extractedCount++
				}
			}
		}
	}

	if extractedCount == 0 {
		return fmt.Errorf("no mesh objects found in 3MF file")
	}

	ui.PrintSuccess(fmt.Sprintf("Successfully extracted %d model(s) to %s", extractedCount, outputDir))
	return nil
}

// extractMesh extracts a single mesh and writes it to an STL file
func (e *Extractor) extractMesh(name, id string, mesh *models.Mesh, outputDir string, binary bool, index int) error {
	// Parse the mesh
	parsedMesh, err := e.parseMesh(mesh)
	if err != nil {
		return fmt.Errorf("error parsing mesh: %w", err)
	}

	// Convert to STL mesh
	stlMesh := e.convertToSTLMesh(parsedMesh, name)

	// Generate output filename
	outputFilename := e.generateFilename(name, id, outputDir, index)

	// Write STL file
	if binary {
		err = e.stlWriter.WriteBinary(stlMesh, outputFilename)
	} else {
		err = e.stlWriter.WriteASCII(stlMesh, outputFilename)
	}

	if err != nil {
		return fmt.Errorf("error writing STL file: %w", err)
	}

	ui.PrintInfo(fmt.Sprintf("Extracted: %s", outputFilename))
	return nil
}

// readExternalModel reads an external model file from the ZIP archive
func (e *Extractor) readExternalModel(zr *zip.Reader, path string) (*models.Mesh, string, error) {
	// Remove leading slash if present (component paths often have it, but ZIP entries don't)
	cleanPath := strings.TrimPrefix(path, "/")

	// Find the external model file in the archive
	var externalFile *zip.File
	for _, f := range zr.File {
		if f.Name == cleanPath || f.Name == path {
			externalFile = f
			break
		}
	}

	if externalFile == nil {
		return nil, "", fmt.Errorf("external model file %s not found", path)
	}

	// Open and read the file
	rc, err := externalFile.Open()
	if err != nil {
		return nil, "", fmt.Errorf("error opening external model file: %w", err)
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		return nil, "", fmt.Errorf("error reading external model file: %w", err)
	}

	// Parse the external model
	var externalModel models.Model
	if err := xml.Unmarshal(data, &externalModel); err != nil {
		return nil, "", fmt.Errorf("error parsing external model XML: %w", err)
	}

	// Find the first object with a mesh in the external model
	for _, obj := range externalModel.Resources.Objects {
		if obj.Mesh != nil && obj.Mesh.Vertices != nil && obj.Mesh.Triangles != nil {
			return obj.Mesh, obj.Name, nil
		}
	}

	return nil, "", fmt.Errorf("no mesh found in external model")
}

// parseMesh parses the XML mesh data into a structured mesh
func (e *Extractor) parseMesh(mesh *models.Mesh) (*ParsedMesh, error) {
	parsed := &ParsedMesh{}

	// Parse vertices
	verticesXML := mesh.Vertices.RawContent
	vertexLines := strings.Split(verticesXML, "<vertex")

	for _, line := range vertexLines {
		if !strings.Contains(line, "x=") {
			continue
		}

		var x, y, z float32
		// Extract x, y, z attributes
		if _, err := fmt.Sscanf(line, ` x="%f" y="%f" z="%f"`, &x, &y, &z); err != nil {
			// Try without quotes
			if _, err := fmt.Sscanf(line, ` x=%f y=%f z=%f`, &x, &y, &z); err != nil {
				continue
			}
		}

		parsed.Vertices = append(parsed.Vertices, Vertex{X: x, Y: y, Z: z})
	}

	// Parse triangles
	trianglesXML := mesh.Triangles.RawContent
	triangleLines := strings.Split(trianglesXML, "<triangle")

	for _, line := range triangleLines {
		if !strings.Contains(line, "v1=") {
			continue
		}

		var v1, v2, v3 int
		// Extract v1, v2, v3 attributes
		if _, err := fmt.Sscanf(line, ` v1="%d" v2="%d" v3="%d"`, &v1, &v2, &v3); err != nil {
			// Try without quotes
			if _, err := fmt.Sscanf(line, ` v1=%d v2=%d v3=%d`, &v1, &v2, &v3); err != nil {
				continue
			}
		}

		parsed.Triangles = append(parsed.Triangles, Triangle{V1: v1, V2: v2, V3: v3})
	}

	if len(parsed.Vertices) == 0 || len(parsed.Triangles) == 0 {
		return nil, fmt.Errorf("no vertices or triangles found")
	}

	return parsed, nil
}

// convertToSTLMesh converts a parsed mesh to an STL mesh
func (e *Extractor) convertToSTLMesh(mesh *ParsedMesh, name string) *stl.Mesh {
	stlMesh := &stl.Mesh{
		Name:      name,
		Triangles: []stl.Triangle{},
	}

	for _, tri := range mesh.Triangles {
		if tri.V1 >= len(mesh.Vertices) || tri.V2 >= len(mesh.Vertices) || tri.V3 >= len(mesh.Vertices) {
			continue
		}

		v1 := mesh.Vertices[tri.V1]
		v2 := mesh.Vertices[tri.V2]
		v3 := mesh.Vertices[tri.V3]

		// Calculate normal (cross product of two edges)
		// Edge1 = v2 - v1
		// Edge2 = v3 - v1
		// Normal = Edge1 x Edge2
		edge1 := stl.Vector3{X: v2.X - v1.X, Y: v2.Y - v1.Y, Z: v2.Z - v1.Z}
		edge2 := stl.Vector3{X: v3.X - v1.X, Y: v3.Y - v1.Y, Z: v3.Z - v1.Z}

		normal := stl.Vector3{
			X: edge1.Y*edge2.Z - edge1.Z*edge2.Y,
			Y: edge1.Z*edge2.X - edge1.X*edge2.Z,
			Z: edge1.X*edge2.Y - edge1.Y*edge2.X,
		}

		// Normalize the normal vector
		lengthSquared := float64(normal.X)*float64(normal.X) +
			float64(normal.Y)*float64(normal.Y) +
			float64(normal.Z)*float64(normal.Z)
		if lengthSquared > 0 {
			length := float32(math.Sqrt(lengthSquared))
			normal.X /= length
			normal.Y /= length
			normal.Z /= length
		}

		stlTriangle := stl.Triangle{
			Normal: normal,
			V1:     stl.Vector3{X: v1.X, Y: v1.Y, Z: v1.Z},
			V2:     stl.Vector3{X: v2.X, Y: v2.Y, Z: v2.Z},
			V3:     stl.Vector3{X: v3.X, Y: v3.Y, Z: v3.Z},
		}

		stlMesh.Triangles = append(stlMesh.Triangles, stlTriangle)
	}

	return stlMesh
}

// generateFilename generates an output filename for an extracted model
func (e *Extractor) generateFilename(name string, id string, outputDir string, index int) string {
	// Clean the name for use as a filename
	cleanName := name
	if cleanName == "" {
		cleanName = fmt.Sprintf("object_%s", id)
	}

	// Remove invalid filename characters
	cleanName = strings.ReplaceAll(cleanName, "/", "_")
	cleanName = strings.ReplaceAll(cleanName, "\\", "_")
	cleanName = strings.ReplaceAll(cleanName, ":", "_")
	cleanName = strings.ReplaceAll(cleanName, "*", "_")
	cleanName = strings.ReplaceAll(cleanName, "?", "_")
	cleanName = strings.ReplaceAll(cleanName, "\"", "_")
	cleanName = strings.ReplaceAll(cleanName, "<", "_")
	cleanName = strings.ReplaceAll(cleanName, ">", "_")
	cleanName = strings.ReplaceAll(cleanName, "|", "_")

	// Ensure unique filenames by adding index if needed
	baseFilename := fmt.Sprintf("%s_%s.stl", cleanName, id)
	if index > 0 {
		baseFilename = fmt.Sprintf("%s_%s_%d.stl", cleanName, id, index)
	}

	return filepath.Join(outputDir, baseFilename)
}

// sanitizeID converts object ID to a valid filename part
func sanitizeID(id string) string {
	// Convert ID to integer and back to ensure it's clean
	if idNum, err := strconv.Atoi(id); err == nil {
		return strconv.Itoa(idNum)
	}
	return id
}

// ensureDir creates a directory if it doesn't exist
func ensureDir(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return os.MkdirAll(dir, 0755)
	}
	return nil
}

// readObjectNames reads object names from model_settings.config
func (e *Extractor) readObjectNames(zr *zip.Reader) map[string]string {
	objectNames := make(map[string]string)

	// Find the model_settings.config file
	var settingsFile *zip.File
	for _, f := range zr.File {
		if f.Name == "Metadata/model_settings.config" {
			settingsFile = f
			break
		}
	}

	if settingsFile == nil {
		// File doesn't exist, return empty map
		return objectNames
	}

	// Open and read the file
	rc, err := settingsFile.Open()
	if err != nil {
		return objectNames
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		return objectNames
	}

	// Parse the XML
	var settings models.ModelSettings
	if err := xml.Unmarshal(data, &settings); err != nil {
		return objectNames
	}

	// Extract object names
	for _, obj := range settings.Objects {
		for _, metadata := range obj.Metadata {
			if metadata.Key == "name" && metadata.Value != "" {
				// Remove .stl extension if present
				name := metadata.Value
				if strings.HasSuffix(name, ".stl") {
					name = name[:len(name)-4]
				}
				objectNames[obj.ID] = name
				break
			}
		}
	}

	return objectNames
}
