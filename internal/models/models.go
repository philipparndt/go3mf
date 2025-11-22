package models

import (
	"encoding/xml"
	"strings"
)

// PackingAlgorithm represents the algorithm used for placing objects on the build plate
type PackingAlgorithm string

const (
	// PackingAlgorithmDefault creates a long print surface (shelving algorithm)
	PackingAlgorithmDefault PackingAlgorithm = "default"

	// PackingAlgorithmCompact places objects as compactly as possible in both directions
	PackingAlgorithmCompact PackingAlgorithm = "compact"
)

// NewPackingAlgorithm creates a PackingAlgorithm from a string, defaulting to PackingAlgorithmDefault
func NewPackingAlgorithm(s string) PackingAlgorithm {
	s = strings.ToLower(strings.TrimSpace(s))
	switch s {
	case "compact":
		return PackingAlgorithmCompact
	default:
		return PackingAlgorithmDefault
	}
}

// String returns the string representation of the packing algorithm
func (pa PackingAlgorithm) String() string {
	return string(pa)
}

// Model represents a 3MF model structure
type Model struct {
	XMLName            xml.Name   `xml:"model"`
	Xmlns              string     `xml:"xmlns,attr"`
	XmlnsP             string     `xml:"xmlns:p,attr,omitempty"`
	XmlnsBambuStudio   string     `xml:"xmlns:BambuStudio,attr,omitempty"`
	RequiredExtensions string     `xml:"requiredextensions,attr,omitempty"`
	Unit               string     `xml:"unit,attr"`
	Lang               string     `xml:"xml:lang,attr"`
	Metadata           []Metadata `xml:"metadata"`
	Resources          Resources  `xml:"resources"`
	Build              Build      `xml:"build"`
}

type Metadata struct {
	Name     string `xml:"name,attr"`
	Preserve string `xml:"preserve,attr"`
	Value    string `xml:",chardata"`
}

type Resources struct {
	BaseMaterials *BaseMaterials `xml:"basematerials"`
	Objects       []Object       `xml:"object"`
}

type BaseMaterials struct {
	ID    string `xml:"id,attr"`
	Bases []Base `xml:"base"`
}

type Base struct {
	Name         string `xml:"name,attr"`
	DisplayColor string `xml:"displaycolor,attr"`
}

type Object struct {
	ID         string      `xml:"id,attr"`
	Name       string      `xml:"name,attr"`
	Type       string      `xml:"type,attr"`
	UUID       string      `xml:"p:UUID,attr,omitempty"`
	PID        string      `xml:"pid,attr"`
	PIndex     string      `xml:"pindex,attr"`
	Mesh       *Mesh       `xml:"mesh"`
	Components *Components `xml:"components"`
}

type Components struct {
	Component []Component `xml:"component"`
}

type Component struct {
	ObjectID  string `xml:"objectid,attr"`
	UUID      string `xml:"http://schemas.microsoft.com/3dmanufacturing/production/2015/06 UUID,attr,omitempty"`
	Path      string `xml:"http://schemas.microsoft.com/3dmanufacturing/production/2015/06 path,attr,omitempty"`
	Transform string `xml:"transform,attr,omitempty"`
}

type Mesh struct {
	Vertices   *Vertices  `xml:"vertices"`
	Triangles  *Triangles `xml:"triangles"`
	RawContent string     `xml:"-"`
}

type Vertices struct {
	RawContent string `xml:",innerxml"`
}

type Triangles struct {
	RawContent string `xml:",innerxml"`
}

type Build struct {
	UUID  string `xml:"p:UUID,attr,omitempty"`
	Items []Item `xml:"item"`
}

type Item struct {
	ObjectID  string `xml:"objectid,attr"`
	UUID      string `xml:"p:UUID,attr,omitempty"`
	Transform string `xml:"transform,attr"`
	Printable string `xml:"printable,attr,omitempty"`
}

// ScadFile represents a SCAD file with its target name
type ScadFile struct {
	Path         string
	Name         string
	FilamentSlot int               // 1-4 for AMS slots, 0 for auto-assign
	ConfigFiles  map[string]string // Map of config filename -> content
	RotationX    float64           // Rotation around X axis in degrees
	RotationY    float64           // Rotation around Y axis in degrees
	RotationZ    float64           // Rotation around Z axis in degrees
	PositionX    float64           // Relative position offset in X (mm)
	PositionY    float64           // Relative position offset in Y (mm)
	PositionZ    float64           // Relative position offset in Z (mm)
}

// ObjectGroup represents a group of parts that form a single object
type ObjectGroup struct {
	ID                string     // Object ID in the 3MF model
	Name              string     // Object name
	Parts             []ScadFile // Parts in this object
	NormalizePosition bool       // If true, normalize z-position to ground level
}

// YamlConfig represents the complete YAML configuration file
type YamlConfig struct {
	Output             string           `yaml:"output"`
	PackingDistance    float64          `yaml:"packing_distance,omitempty"`   // Distance between objects in mm (default: 10.0)
	PackingAlgorithm   string           `yaml:"packing_algorithm,omitempty"`  // Packing algorithm: "default" or "compact" (default: "default")
	Objects            []YamlObject     `yaml:"objects"`
}

// YamlObject represents a single object in the model
type YamlObject struct {
	Name              string                   `yaml:"name"`
	Config            []map[string]interface{} `yaml:"config,omitempty"`            // Array of config filename -> content maps (applied to all parts)
	NormalizePosition *bool                    `yaml:"normalize_position,omitempty"` // If true, normalize z-position to ground level (default: true)
	Parts             []YamlPart               `yaml:"parts"`
}

// YamlPart represents a part within an object
type YamlPart struct {
	Name      string                   `yaml:"name"`
	File      string                   `yaml:"file"`
	Config    []map[string]interface{} `yaml:"config,omitempty"`     // Array of config filename -> content maps (part-specific)
	Filament  int                      `yaml:"filament,omitempty"`   // 1-4 for AMS slots, 0 for auto-assign
	RotationX float64                  `yaml:"rotation_x,omitempty"` // Rotation around X axis in degrees
	RotationY float64                  `yaml:"rotation_y,omitempty"` // Rotation around Y axis in degrees
	RotationZ float64                  `yaml:"rotation_z,omitempty"` // Rotation around Z axis in degrees
	PositionX float64                  `yaml:"position_x,omitempty"` // Relative position offset in X (mm)
	PositionY float64                  `yaml:"position_y,omitempty"` // Relative position offset in Y (mm)
	PositionZ float64                  `yaml:"position_z,omitempty"` // Relative position offset in Z (mm)
}

// ModelSettings represents the Bambu Studio model_settings.config structure
type ModelSettings struct {
	XMLName  xml.Name         `xml:"config"`
	Objects  []SettingsObject `xml:"object"`
	Plate    Plate            `xml:"plate"`
	Assemble Assemble         `xml:"assemble"`
}

type SettingsObject struct {
	ID       string             `xml:"id,attr"`
	Metadata []SettingsMetadata `xml:"metadata"`
	Parts    []Part             `xml:"part"`
}

type SettingsMetadata struct {
	Key       string `xml:"key,attr"`
	Value     string `xml:"value,attr,omitempty"`
	FaceCount int    `xml:"face_count,attr,omitempty"`
}

type Part struct {
	ID       string             `xml:"id,attr"`
	Subtype  string             `xml:"subtype,attr"`
	Metadata []SettingsMetadata `xml:"metadata"`
	MeshStat MeshStat           `xml:"mesh_stat"`
}

type MeshStat struct {
	FaceCount        int `xml:"face_count,attr"`
	EdgesFixed       int `xml:"edges_fixed,attr"`
	DegenerateFacets int `xml:"degenerate_facets,attr"`
	FacetsRemoved    int `xml:"facets_removed,attr"`
	FacetsReversed   int `xml:"facets_reversed,attr"`
	BackwardsEdges   int `xml:"backwards_edges,attr"`
}

type Plate struct {
	Metadata       []SettingsMetadata `xml:"metadata"`
	ModelInstances []ModelInstance    `xml:"model_instance"`
}

type ModelInstance struct {
	Metadata []SettingsMetadata `xml:"metadata"`
}

type Assemble struct {
	Items []AssembleItem `xml:"assemble_item"`
}

type AssembleItem struct {
	ObjectID   string `xml:"object_id,attr"`
	InstanceID string `xml:"instance_id,attr"`
	Transform  string `xml:"transform,attr"`
	Offset     string `xml:"offset,attr"`
}
