package gen

import (
	"fmt"
	"reflect"
	"strings"
)

// Printer builds a string representation of a tree of Participle proto-structs.
type Printer struct {
	depth  int
	result string
}

// Visit builds a string representation of a tree of proto-structs.
func (v *Printer) Visit(n Node) string {
	if n == nil || reflect.ValueOf(n) == reflect.Zero(reflect.TypeOf(n)) {
		return ""
	}
	v.result = ""
	n.Accept(v)
	return v.result
}

// VisitStruct implements the Visitor interface.
func (v *Printer) VisitStruct(s *Struct) {
	if s == nil {
		return
	}
	if v.depth == 0 {
		v.result = fmt.Sprintf("type %s struct {\n%s\n}", s.Name, v.Visit(s.Fields))
	} else {
		v.result = fmt.Sprintf("[]*struct{\n%s\n}", v.Visit(s.Fields))
	}
}

// VisitStructFields implements the Visitor interface.
func (v *Printer) VisitStructFields(sf StructFields) {
	ret := make([]string, len(sf))
	for i, f := range sf {
		ret[i] = v.Visit(f)
	}
	v.result = strings.Join(ret, "\n")
}

// VisitStructField implements the Visitor interface.
func (v *Printer) VisitStructField(sf *StructField) {
	v.depth++
	sub := v.Visit(sf.SubType)
	v.depth--
	v.result = fmt.Sprintf("%s%s %s%s `%s`", strings.Repeat("\t", v.depth), sf.Name, sf.RawType, sub, sf.Tag)
}
