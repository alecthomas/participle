package gen

import (
	"reflect"
)

type CaptureCounter struct {
	result int
}

func NewCaptureCounter() *CaptureCounter {
	return &CaptureCounter{}
}

func (v *CaptureCounter) Visit(a interface {
	Accept(Visitor)
}) (ret int) {
	if a == nil || reflect.ValueOf(a) == reflect.Zero(reflect.TypeOf(a)) {
		return 0
	}
	v.result = 0
	a.Accept(v)
	ret = v.result
	return
}

func (v *CaptureCounter) VisitStruct(s *Struct) {
	if s == nil {
		return
	}
	v.result = v.Visit(s.Fields)
}

func (v *CaptureCounter) VisitStructFields(sf StructFields) {
	var count int
	for _, f := range sf {
		count += v.Visit(f)
	}
	v.result = count
}

func (v *CaptureCounter) VisitStructField(sf *StructField) {
	switch {
	case sf.RawType == "struct{}":
	case sf.SubType != nil:
		v.result = v.Visit(sf.SubType)
	default:
		v.result = 1
	}
}
