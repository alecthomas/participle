package parser

import (
	"reflect"
	"testing"

	"github.com/alecthomas/assert"
)

func TestScanner(t *testing.T) {
	type testScanner struct {
		A string `12`
		B string `34`
	}

	scan := newStructScanner(reflect.TypeOf(testScanner{}))
	assert.Equal(t, '1', scan.Peek())
	assert.Equal(t, 0, scan.field)
	assert.Equal(t, '1', scan.Next())
	assert.Equal(t, 0, scan.field)
	assert.Equal(t, '2', scan.Peek())
	assert.Equal(t, 0, scan.field)

	assert.Equal(t, '2', scan.Next())
	assert.Equal(t, 0, scan.field)

	assert.Equal(t, '3', scan.Peek())
	assert.Equal(t, 0, scan.field)
	assert.Equal(t, '3', scan.Next())
	assert.Equal(t, 1, scan.field)

	assert.Equal(t, '4', scan.Peek())
	assert.Equal(t, 1, scan.field)
	assert.Equal(t, '4', scan.Next())
	assert.Equal(t, 1, scan.field)

	assert.Equal(t, EOF, scan.Next())
}

func TestStructScanner(t *testing.T) {
	g := struct {
		A string `"a"|`
		B string `"b"`
	}{}

	gt := reflect.TypeOf(g)
	r := newStructScanner(gt)
	f := []reflect.StructField{}
	s := ""
	for {
		r.Peek()
		rn := r.Next()
		if rn == EOF {
			break
		}
		f = append(f, r.Field())
		s += string(rn)
	}
	assert.Equal(t, `"a"|"b"`, s)
	f0 := gt.Field(0)
	f1 := gt.Field(1)
	assert.Equal(t, []reflect.StructField{f0, f0, f0, f0, f1, f1, f1}, f)
}

func TestReaderScanner(t *testing.T) {
	scan := StringScanner("hi")
	assert.Equal(t, 'h', scan.Peek())
	assert.Equal(t, 'h', scan.Next())
	assert.Equal(t, 'i', scan.Peek())
	assert.Equal(t, 'i', scan.Next())
	assert.Equal(t, EOF, scan.Peek())
	assert.Equal(t, EOF, scan.Next())
}
