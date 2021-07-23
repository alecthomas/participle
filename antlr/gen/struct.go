package gen

import (
	"strings"
)

type Visitor interface {
	VisitStruct(s *Struct)
	VisitStructFields(sf StructFields)
	VisitStructField(sf *StructField)
}

type Struct struct {
	Name   string
	Fields StructFields
}

func (s *Struct) Accept(v Visitor) {
	v.VisitStruct(s)
}

func (s *Struct) AddFields(sf ...*StructField) {
	s.Fields = append(s.Fields, sf...)
}

type StructFields []*StructField

func (sf StructFields) Accept(v Visitor) {
	v.VisitStructFields(sf)
}

func (sf StructFields) Tags() (ret []string) {
	ret = make([]string, len(sf))
	for i, v := range sf {
		ret[i] = v.Tag
	}
	return
}

func (sf StructFields) AreCapturing() bool {
	for _, f := range sf {
		if f.IsCapturing() {
			return true
		}
	}
	return false
}

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

type StructField struct {
	Name    string
	RawType string
	SubType *Struct
	Tag     string
}

func (sf *StructField) Accept(v Visitor) {
	v.VisitStructField(sf)
}

func (sf *StructField) CanMerge(sf2 *StructField) bool {
	return strings.TrimPrefix(sf.RawType, "[]") == strings.TrimPrefix(sf2.RawType, "[]") &&
		(strings.HasPrefix(sf.RawType, "[]") || strings.HasPrefix(sf2.RawType, "[]")) &&
		sf.SubType == sf2.SubType &&
		sf.Name == sf2.Name
}

func (sf *StructField) IsCapturing() bool {
	return sf.RawType != "struct{}"
}
