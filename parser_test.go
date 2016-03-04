package parser

import (
	"reflect"
	"testing"

	"github.com/alecthomas/assert"
)

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

type testScalar struct {
	A string `@"one"`
}

func TestParseScalar(t *testing.T) {
	g := testScalar{}
	e := parseType(reflect.TypeOf(g))
	actual := e.Parse(StringScanner("one"))
	assert.NotNil(t, actual)
	assert.Equal(t, 1, len(actual))
	assert.Equal(t, testScalar{"one"}, actual[0].Interface())
}

type testGroup struct {
	A string `@("one" | "two")`
}

func TestParseGroup(t *testing.T) {
	g := testGroup{}
	e := parseType(reflect.TypeOf(g))

	actual := e.Parse(StringScanner("one"))
	assert.NotNil(t, actual)
	assert.Equal(t, 1, len(actual))
	assert.Equal(t, testGroup{"one"}, actual[0].Interface())

	actual = e.Parse(StringScanner("two"))
	assert.NotNil(t, actual)
	assert.Equal(t, 1, len(actual))
	assert.Equal(t, testGroup{"two"}, actual[0].Interface())
}

type testAlternative struct {
	A string `@"one" |`
	B string `@"two"`
}

func TestAlternative(t *testing.T) {
	g := testAlternative{}
	e := parseType(reflect.TypeOf(g))

	actual := e.Parse(StringScanner("one"))
	assert.NotNil(t, actual)
	assert.Equal(t, 1, len(actual))
	assert.Equal(t, testAlternative{A: "one"}, actual[0].Interface())

	actual = e.Parse(StringScanner("two"))
	assert.NotNil(t, actual)
	assert.Equal(t, 1, len(actual))
	assert.Equal(t, testAlternative{B: "two"}, actual[0].Interface())
}

type testNested struct {
	A struct {
		B string `@"one"`
		C string `@"two"`
	} `@@`
}

func TestNested(t *testing.T) {
	a := testNested{}
	e := parseType(reflect.TypeOf(a))

	actual := e.Parse(StringScanner("onetwo"))
	assert.NotNil(t, actual)
	assert.Equal(t, 1, len(actual))
	assert.Equal(t, testNested{}, actual[0].Interface())
}
