package main

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"

	"github.com/alecthomas/participle"
)

type pathExpr struct {
	Parts []part `@@ { "." @@ }`
}

type part struct {
	Obj string `@Ident`
	Acc []acc  `("[" @@ "]")*`
}

type acc struct {
	Name  *string `@(String|Char|RawString)`
	Index *int    `| @Int`
}

var parser = participle.MustBuild(&pathExpr{})

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: %s <query> <files...>\n", os.Args[0])
		os.Exit(2)
	}

	q := os.Args[1]
	files := os.Args[2:]

	var expr pathExpr
	if err := parser.ParseString(q, &expr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	for _, file := range files {
		f, err := os.Open(file)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		var input map[string]interface{}
		if err := json.NewDecoder(f).Decode(&input); err != nil {
			f.Close()
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		f.Close()

		result, err := match(input, expr)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		switch r := result.(type) {
		case map[string]interface{}:
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			_ = enc.Encode(r)
		default:
			fmt.Printf("%v\n", r)
		}
	}
}

func match(input map[string]interface{}, expr pathExpr) (interface{}, error) {
	v := reflect.ValueOf(input)

	for _, e := range expr.Parts {
		if v.Kind() == reflect.Interface {
			v = v.Elem()
		}
		if v.Kind() != reflect.Map {
			return nil, fmt.Errorf("%q is not a map", e.Obj)
		}
		v = v.MapIndex(reflect.ValueOf(e.Obj))
		if !v.IsValid() {
			return nil, fmt.Errorf("not found: %q", e.Obj)
		}
		if v.Kind() == reflect.Interface {
			v = v.Elem()
		}
		for _, a := range e.Acc {
			if a.Name != nil {
				if v.Kind() != reflect.Map {
					return nil, fmt.Errorf("cannot access named index in %s", v.Kind())
				}
				v = v.MapIndex(reflect.ValueOf(*a.Name))
			}
			if a.Index != nil {
				if v.Kind() != reflect.Slice {
					return nil, fmt.Errorf("cannot access numeric index in %s", v.Kind())
				}
				v = v.Index(*a.Index)
			}
			if !v.IsValid() {
				return nil, fmt.Errorf("not found: %q", e.Obj)
			}
			if v.Kind() == reflect.Interface {
				v = v.Elem()
			}
		}
	}

	return v.Interface(), nil
}
