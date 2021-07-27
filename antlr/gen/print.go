package gen

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
)

var dquoEscaper = regexp.MustCompile(`\\([\s\S])|(")`)

// Printer builds a string representation of a tree of Participle proto-structs.
type Printer struct {
	altTagFormat bool
	depth        int
	result       string
}

// NewPrinter returns a ready Printer.
// If altTagFormat is true, struct tags will use the parser:"xxx" format.
func NewPrinter(altTagFormat bool) *Printer {
	return &Printer{altTagFormat: altTagFormat}
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
	tag := sf.Tag
	if v.altTagFormat {
		tag = `parser:"` + dquoEscaper.ReplaceAllString(tag, "\\$1$2") + `"`
	}
	v.result = fmt.Sprintf("%s%s %s%s `%s`", strings.Repeat("\t", v.depth), sf.Name, sf.RawType, sub, tag)
}
