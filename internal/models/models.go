package models

import "encoding/xml"

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
	UUID      string `xml:"p:UUID,attr,omitempty"`
	Path      string `xml:"p:path,attr,omitempty"`
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
	FilamentSlot int // 1-4 for AMS slots, 0 for auto-assign
}

// YamlConfig represents the complete YAML configuration file
type YamlConfig struct {
	Output  string       `yaml:"output"`
	Objects []YamlObject `yaml:"objects"`
}

// YamlObject represents a single object in the model
type YamlObject struct {
	Name  string     `yaml:"name"`
	Parts []YamlPart `yaml:"parts"`
}

// YamlPart represents a part within an object
type YamlPart struct {
	Name     string `yaml:"name"`
	File     string `yaml:"file"`
	Filament int    `yaml:"filament,omitempty"` // 1-4 for AMS slots, 0 for auto-assign
}

// ModelSettings represents the Bambu Studio model_settings.config structure
type ModelSettings struct {
	XMLName  xml.Name       `xml:"config"`
	Object   SettingsObject `xml:"object"`
	Plate    Plate          `xml:"plate"`
	Assemble Assemble       `xml:"assemble"`
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
	Metadata      []SettingsMetadata `xml:"metadata"`
	ModelInstance ModelInstance      `xml:"model_instance"`
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
