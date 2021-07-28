package gen

import (
	"reflect"
)

// CaptureCounter determines how many capturing fields
// are present in a tree of Participle proto-structs.
type CaptureCounter struct {
	result int
}

// NewCaptureCounter returns a CaptureCounter.
func NewCaptureCounter() *CaptureCounter {
	return &CaptureCounter{}
}

// Visit counts the captures in a tree of Participle proto-structs.
func (v *CaptureCounter) Visit(n Node) (ret int) {
	if n == nil || reflect.ValueOf(n) == reflect.Zero(reflect.TypeOf(n)) {
		return 0
	}
	v.result = 0
	n.Accept(v)
	ret = v.result
	return
}

// VisitStruct implements the Visitor interface.
func (v *CaptureCounter) VisitStruct(s *Struct) {
	if s == nil {
		return
	}
	v.result = v.Visit(s.Fields)
}

// VisitStructFields implements the Visitor interface.
func (v *CaptureCounter) VisitStructFields(sf StructFields) {
	var count int
	for _, f := range sf {
		count += v.Visit(f)
	}
	v.result = count
}

// VisitStructField implements the Visitor interface.
func (v *CaptureCounter) VisitStructField(sf *StructField) {
	switch {
	case sf.RawType == "struct{}":
	case sf.SubType != nil:
		v.result = v.Visit(sf.SubType)
	default:
		v.result = 1
	}
}
