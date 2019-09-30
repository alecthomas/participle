package main

import (
	"encoding/json"
	"fmt"
	"os"

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
	var v interface{} = input
	for _, e := range expr.Parts {
		switch m := v.(type) {
		case map[string]interface{}:
			val, ok := m[e.Obj]
			if !ok {
				return nil, fmt.Errorf("not found: %q", e.Obj)
			}
			v = val
			for _, a := range e.Acc {
				if a.Name != nil {
					switch m := v.(type) {
					case map[string]interface{}:
						val, ok = m[*a.Name].(map[string]interface{})
						if !ok {
							return nil, fmt.Errorf("not found: %q does not contain %q", e.Obj, *a.Name)
						}
						v = val
					default:
						return nil, fmt.Errorf("cannot access named index in %T", v)
					}
				}
				if a.Index != nil {
					switch s := v.(type) {
					case []interface{}:
						if len(s) <= *a.Index {
							return nil, fmt.Errorf("not found: %q does contains %d items", e.Obj, len(s))
						}
						v = s[*a.Index]
					default:
						return nil, fmt.Errorf("cannot access numeric index in %T", v)
					}
				}
			}
		default:
			return nil, fmt.Errorf("cannot read %q, parent is not a map", e.Obj)
		}
	}
	return v, nil
}
