package gen

import (
	"sort"
)

type structAndIndex struct {
	s       *Struct
	indexes []int
}

// TypeExtractor finds nested type definitions in a slice of Participle
// proto-structs and pulls them out into the top layer.
type TypeExtractor struct {
	structs map[string]structAndIndex
	pos     intStack
}

// NewTypeExtractor returns a TypeExtractor.
func NewTypeExtractor(ss []*Struct) *TypeExtractor {
	ret := &TypeExtractor{
		structs: make(map[string]structAndIndex, len(ss)),
	}
	for i, s := range ss {
		ret.structs[s.Name] = structAndIndex{s, []int{i}}
	}
	return ret
}

// Extract visits a tree of Participle proto-structs, extracts the sub-types
// to the top level, and sorts the results appropriately.
func (v *TypeExtractor) Extract() []*Struct {
	loopCopy := v.structs
	v.structs = make(map[string]structAndIndex, len(v.structs))
	for _, s := range loopCopy {
		v.pos.push(s.indexes[0])
		s.s.Accept(v)
		v.pos.pop()
	}
	toSort := make([]structAndIndex, 0, len(v.structs))
	for _, s := range v.structs {
		toSort = append(toSort, s)
	}
	sort.SliceStable(toSort, func(i, j int) bool {
		l1, l2 := len(toSort[i].indexes), len(toSort[j].indexes)
		for depth := 0; depth < l1 && depth < l2; depth++ {
			idx1, idx2 := toSort[i].indexes[depth], toSort[j].indexes[depth]
			if idx1 != idx2 {
				return idx1 < idx2
			}
		}
		return l1 < l2
	})
	ret := make([]*Struct, len(toSort))
	for i, v := range toSort {
		ret[i] = v.s
	}
	return ret
}

// VisitStruct implements the Visitor interface.
func (v *TypeExtractor) VisitStruct(s *Struct) {
	s.Fields.Accept(v)
	sai := structAndIndex{s: s, indexes: append([]int{}, v.pos.stack...)}
	v.structs[s.Name] = sai
}

// VisitStructFields implements the Visitor interface.
func (v *TypeExtractor) VisitStructFields(sf StructFields) {
	for i, f := range sf {
		v.pos.push(i)
		f.Accept(v)
		v.pos.pop()
	}
}

// VisitStructField implements the Visitor interface.
func (v *TypeExtractor) VisitStructField(sf *StructField) {
	if sf.SubType == nil {
		return
	}
	if _, ok := v.structs[sf.SubType.Name]; !ok {
		sf.SubType.Accept(v)
	}
	sf.RawType = "[]*" + sf.SubType.Name
	sf.SubType = nil
}

type intStack struct {
	stack []int
}

func (is *intStack) push(i int) {
	is.stack = append(is.stack, i)
}

func (is *intStack) pop() int {
	b := is.peek()
	is.stack = is.stack[:len(is.stack)-1]
	return b
}

func (is *intStack) peek() int {
	return is.stack[len(is.stack)-1]
}

func (is *intStack) safePeek() int {
	if len(is.stack) == 0 {
		return 0
	}
	return is.peek()
}

func (is *intStack) depth() int {
	return len(is.stack)
}

func (is *intStack) add(i int) {
	is.push(is.pop() + i)
}

func (is *intStack) set(i int) {
	is.pop()
	is.push(i)
}
