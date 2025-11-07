package stl

import (
	"archive/zip"
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Vector3 represents a 3D vector
type Vector3 struct {
	X, Y, Z float32
}

// Triangle represents a triangle in 3D space
type Triangle struct {
	Normal     Vector3
	V1, V2, V3 Vector3
}

// Mesh represents an STL mesh
type Mesh struct {
	Name      string
	Triangles []Triangle
}

// Parser parses STL files
type Parser struct{}

// NewParser creates a new STL parser
func NewParser() *Parser {
	return &Parser{}
}

// Parse reads an STL file and returns the mesh data
func (p *Parser) Parse(filename string) (*Mesh, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("cannot open file: %w", err)
	}
	defer file.Close()

	// Read first few bytes to detect format
	header := make([]byte, 80)
	if _, err := io.ReadFull(file, header); err != nil {
		return nil, fmt.Errorf("error reading header: %w", err)
	}

	// Reset file position
	if _, err := file.Seek(0, 0); err != nil {
		return nil, fmt.Errorf("error seeking: %w", err)
	}

	// Check if it's ASCII (starts with "solid")
	if strings.HasPrefix(string(header), "solid") {
		return p.parseASCII(file, filename)
	}
	return p.parseBinary(file, filename)
}

// parseASCII parses an ASCII STL file
func (p *Parser) parseASCII(reader io.Reader, filename string) (*Mesh, error) {
	scanner := bufio.NewScanner(reader)
	mesh := &Mesh{
		Name:      filepath.Base(filename),
		Triangles: []Triangle{},
	}

	var currentTriangle Triangle
	var vertexCount int

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		fields := strings.Fields(line)

		if len(fields) == 0 {
			continue
		}

		switch fields[0] {
		case "solid":
			if len(fields) > 1 {
				mesh.Name = strings.Join(fields[1:], " ")
			}
		case "facet":
			if len(fields) >= 5 && fields[1] == "normal" {
				fmt.Sscanf(strings.Join(fields[2:], " "), "%f %f %f",
					&currentTriangle.Normal.X, &currentTriangle.Normal.Y, &currentTriangle.Normal.Z)
			}
			vertexCount = 0
		case "vertex":
			if len(fields) >= 4 {
				var v Vector3
				fmt.Sscanf(strings.Join(fields[1:], " "), "%f %f %f", &v.X, &v.Y, &v.Z)
				switch vertexCount {
				case 0:
					currentTriangle.V1 = v
				case 1:
					currentTriangle.V2 = v
				case 2:
					currentTriangle.V3 = v
				}
				vertexCount++
			}
		case "endfacet":
			mesh.Triangles = append(mesh.Triangles, currentTriangle)
			currentTriangle = Triangle{}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	return mesh, nil
}

// parseBinary parses a binary STL file
func (p *Parser) parseBinary(reader io.Reader, filename string) (*Mesh, error) {
	mesh := &Mesh{
		Name: filepath.Base(filename),
	}

	// Read 80-byte header
	header := make([]byte, 80)
	if _, err := io.ReadFull(reader, header); err != nil {
		return nil, fmt.Errorf("error reading header: %w", err)
	}

	// Read triangle count
	var triangleCount uint32
	if err := binary.Read(reader, binary.LittleEndian, &triangleCount); err != nil {
		return nil, fmt.Errorf("error reading triangle count: %w", err)
	}

	// Read triangles
	mesh.Triangles = make([]Triangle, triangleCount)
	for i := uint32(0); i < triangleCount; i++ {
		var triangle Triangle

		// Read normal
		if err := binary.Read(reader, binary.LittleEndian, &triangle.Normal); err != nil {
			return nil, fmt.Errorf("error reading normal: %w", err)
		}

		// Read vertices
		if err := binary.Read(reader, binary.LittleEndian, &triangle.V1); err != nil {
			return nil, fmt.Errorf("error reading vertex 1: %w", err)
		}
		if err := binary.Read(reader, binary.LittleEndian, &triangle.V2); err != nil {
			return nil, fmt.Errorf("error reading vertex 2: %w", err)
		}
		if err := binary.Read(reader, binary.LittleEndian, &triangle.V3); err != nil {
			return nil, fmt.Errorf("error reading vertex 3: %w", err)
		}

		// Skip attribute byte count
		var attributeCount uint16
		if err := binary.Read(reader, binary.LittleEndian, &attributeCount); err != nil {
			return nil, fmt.Errorf("error reading attribute count: %w", err)
		}

		mesh.Triangles[i] = triangle
	}

	return mesh, nil
}

// Converter converts STL meshes to 3MF format
type Converter struct {
	parser *Parser
}

// NewConverter creates a new STL to 3MF converter
func NewConverter() *Converter {
	return &Converter{
		parser: NewParser(),
	}
}

// ConvertTo3MF converts an STL file to 3MF format
func (c *Converter) ConvertTo3MF(stlFile, outputFile string) error {
	mesh, err := c.parser.Parse(stlFile)
	if err != nil {
		return fmt.Errorf("error parsing STL: %w", err)
	}

	return c.write3MF(mesh, outputFile)
}

// write3MF writes a mesh to a 3MF file
func (c *Converter) write3MF(mesh *Mesh, outputFile string) error {
	// Create output file
	outFile, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("error creating output file: %w", err)
	}
	defer outFile.Close()

	zipWriter := zip.NewWriter(outFile)
	defer zipWriter.Close()

	// Build vertices and triangles XML
	verticesXML, trianglesXML := c.buildMeshXML(mesh)

	// Create 3MF model XML
	modelXML := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<model unit="millimeter" xml:lang="en-US" xmlns="http://schemas.microsoft.com/3dmanufacturing/core/2015/02">
	<resources>
		<object id="1" type="model">
			<mesh>
				<vertices>
%s				</vertices>
				<triangles>
%s				</triangles>
			</mesh>
		</object>
	</resources>
	<build>
		<item objectid="1" transform="1 0 0 0 1 0 0 0 1 0 0 0"/>
	</build>
</model>`, verticesXML, trianglesXML)

	// Write model file
	modelWriter, err := zipWriter.Create("3D/3dmodel.model")
	if err != nil {
		return fmt.Errorf("error creating model entry: %w", err)
	}
	if _, err := modelWriter.Write([]byte(modelXML)); err != nil {
		return fmt.Errorf("error writing model XML: %w", err)
	}

	// Write [Content_Types].xml
	contentTypesXML := `<?xml version="1.0" encoding="UTF-8"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
	<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
	<Default Extension="model" ContentType="application/vnd.ms-package.3dmanufacturing-3dmodel+xml"/>
</Types>`

	contentWriter, err := zipWriter.Create("[Content_Types].xml")
	if err != nil {
		return fmt.Errorf("error creating content types: %w", err)
	}
	if _, err := contentWriter.Write([]byte(contentTypesXML)); err != nil {
		return fmt.Errorf("error writing content types: %w", err)
	}

	// Write _rels/.rels
	relsXML := `<?xml version="1.0" encoding="UTF-8"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
	<Relationship Id="rel0" Target="/3D/3dmodel.model" Type="http://schemas.microsoft.com/3dmanufacturing/2013/01/3dmodel"/>
</Relationships>`

	relsWriter, err := zipWriter.Create("_rels/.rels")
	if err != nil {
		return fmt.Errorf("error creating rels: %w", err)
	}
	if _, err := relsWriter.Write([]byte(relsXML)); err != nil {
		return fmt.Errorf("error writing rels: %w", err)
	}

	return nil
}

// buildMeshXML builds the vertices and triangles XML from a mesh
func (c *Converter) buildMeshXML(mesh *Mesh) (string, string) {
	// Build a map of unique vertices
	vertexMap := make(map[Vector3]int)
	var vertices []Vector3
	vertexIndex := 0

	getVertexIndex := func(v Vector3) int {
		if idx, exists := vertexMap[v]; exists {
			return idx
		}
		vertexMap[v] = vertexIndex
		vertices = append(vertices, v)
		vertexIndex++
		return vertexIndex - 1
	}

	// Build vertices and triangle indices
	var verticesBuf bytes.Buffer
	var trianglesBuf bytes.Buffer

	// Process all triangles to build vertex list
	type TriangleIndices struct {
		V1, V2, V3 int
	}
	var triangleIndices []TriangleIndices

	for _, tri := range mesh.Triangles {
		idx1 := getVertexIndex(tri.V1)
		idx2 := getVertexIndex(tri.V2)
		idx3 := getVertexIndex(tri.V3)
		triangleIndices = append(triangleIndices, TriangleIndices{idx1, idx2, idx3})
	}

	// Write vertices XML
	for _, v := range vertices {
		verticesBuf.WriteString(fmt.Sprintf("\t\t\t\t\t<vertex x=\"%.6f\" y=\"%.6f\" z=\"%.6f\"/>\n", v.X, v.Y, v.Z))
	}

	// Write triangles XML
	for _, tri := range triangleIndices {
		trianglesBuf.WriteString(fmt.Sprintf("\t\t\t\t\t<triangle v1=\"%d\" v2=\"%d\" v3=\"%d\"/>\n", tri.V1, tri.V2, tri.V3))
	}

	return verticesBuf.String(), trianglesBuf.String()
}
