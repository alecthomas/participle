package gen

import (
	"fmt"
	"reflect"
	"strings"
)

type Printer struct {
	depth  int
	result string
}

func NewPrinter() *Printer {
	return &Printer{}
}

func (v *Printer) Visit(a interface {
	Accept(Visitor)
}) string {
	if a == nil || reflect.ValueOf(a) == reflect.Zero(reflect.TypeOf(a)) {
		return ""
	}
	v.result = ""
	a.Accept(v)
	return v.result
}

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

func (v *Printer) VisitStructFields(sf StructFields) {
	ret := make([]string, len(sf))
	for i, f := range sf {
		ret[i] = v.Visit(f)
	}
	v.result = strings.Join(ret, "\n")
}

func (v *Printer) VisitStructField(sf *StructField) {
	v.depth++
	sub := v.Visit(sf.SubType)
	v.depth--
	v.result = fmt.Sprintf("%s%s %s%s `%s`", strings.Repeat("\t", v.depth), sf.Name, sf.RawType, sub, sf.Tag)
}
