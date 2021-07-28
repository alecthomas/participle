package gen

import (
	"strings"
)

// Node is conformed to by all AST nodes.
type Node interface {
	Accept(Visitor)
}

// Visitor allows for walking a tree of Participle proto-structs.
type Visitor interface {
	VisitStruct(s *Struct)
	VisitStructFields(sf StructFields)
	VisitStructField(sf *StructField)
}

// Struct is the result of translating an Antlr rule to a Participle node,
// and will become an actual struct in the generated code.
type Struct struct {
	Name   string
	Fields StructFields
}

// Accept is used for the Visitor interface.
func (s *Struct) Accept(v Visitor) {
	v.VisitStruct(s)
}

// AddFields is a utility method to append to the end of the struct's fields.
func (s *Struct) AddFields(sf ...*StructField) {
	s.Fields = append(s.Fields, sf...)
}

// StructFields is a convenience type.
type StructFields []*StructField

// Accept is used for the Visitor interface.
func (sf StructFields) Accept(v Visitor) {
	v.VisitStructFields(sf)
}

// Tags returns the struct tag data for every field.
func (sf StructFields) Tags() (ret []string) {
	ret = make([]string, len(sf))
	for i, v := range sf {
		ret[i] = v.Tag
	}
	return
}

// AreCapturing checks if any field in the set is a capturing field.
func (sf StructFields) AreCapturing() bool {
	for _, f := range sf {
		if f.IsCapturing() {
			return true
		}
	}
	return false
}

// SquashToIndex combines a slice of StructFields into a single instance
// by concatenating their struct tag data, and retaining name and type information
// from entry idx in the slice.  If idx is -1, synthesize a struct{} field with
// the name Meta.
func (sf StructFields) SquashToIndex(idx int) *StructField {
	if idx == -1 {
		return &StructField{
			Name:    "Meta",
			RawType: "struct{}",
			Tag:     strings.Join(sf.Tags(), " "),
		}
	}
	return &StructField{
		Name:    sf[idx].Name,
		RawType: sf[idx].RawType,
		SubType: sf[idx].SubType,
		Tag:     strings.Join(sf.Tags(), " "),
	}
}

// StructField is the information for one field of a Participle proto-struct.
type StructField struct {
	Name    string
	RawType string
	SubType *Struct
	Tag     string
}

// Accept is used for the Visitor interface.
func (sf *StructField) Accept(v Visitor) {
	v.VisitStructField(sf)
}

// CanMerge defines if two adjacent capturing fields can combine.
// They must have matching types, names, and one of them must already be plural (a slice type).
func (sf *StructField) CanMerge(sf2 *StructField) bool {
	return strings.TrimPrefix(sf.RawType, "[]") == strings.TrimPrefix(sf2.RawType, "[]") &&
		(strings.HasPrefix(sf.RawType, "[]") || strings.HasPrefix(sf2.RawType, "[]")) &&
		sf.SubType == sf2.SubType &&
		sf.Name == sf2.Name
}

// IsCapturing returns if the field is capturng information from the parse.
// Currently every field captures except fields of type struct{}
func (sf *StructField) IsCapturing() bool {
	return sf.RawType != "struct{}"
}
