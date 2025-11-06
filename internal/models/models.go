package models

import "encoding/xml"

// Model represents a 3MF model structure
type Model struct {
	XMLName   xml.Name   `xml:"model"`
	Xmlns     string     `xml:"xmlns,attr"`
	Unit      string     `xml:"unit,attr"`
	Lang      string     `xml:"xml:lang,attr"`
	Metadata  []Metadata `xml:"metadata"`
	Resources Resources  `xml:"resources"`
	Build     Build      `xml:"build"`
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
	ID     string `xml:"id,attr"`
	Name   string `xml:"name,attr"`
	Type   string `xml:"type,attr"`
	UUID   string `xml:"p:UUID,attr"`
	PID    string `xml:"pid,attr"`
	PIndex string `xml:"pindex,attr"`
	Mesh   *Mesh  `xml:"mesh"`
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
	Items []Item `xml:"item"`
}

type Item struct {
	ObjectID  string `xml:"objectid,attr"`
	Transform string `xml:"transform,attr"`
}

// ScadFile represents a SCAD file with its target name
type ScadFile struct {
	Path string
	Name string
}
