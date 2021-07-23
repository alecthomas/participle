package gen

import "strconv"

type FieldRenamer struct {
	nameCounter []map[string]int
}

func NewFieldRenamer() *FieldRenamer {
	return &FieldRenamer{}
}

func (v *FieldRenamer) VisitStruct(s *Struct) {
	if s == nil {
		return
	}
	v.nameCounter = append(v.nameCounter, map[string]int{})
	s.Fields.Accept(v)
	v.nameCounter = v.nameCounter[:len(v.nameCounter)-1]
}

func (v *FieldRenamer) VisitStructFields(sf StructFields) {
	for _, f := range sf {
		f.Accept(v)
	}
}

func (v *FieldRenamer) VisitStructField(sf *StructField) {
	nc := v.nameCounter[len(v.nameCounter)-1]
	nc[sf.Name]++
	if c := nc[sf.Name]; c > 1 {
		sf.Name += strconv.Itoa(c)
	}
	sf.SubType.Accept(v)
}
