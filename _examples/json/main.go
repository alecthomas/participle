package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"reflect"

	"github.com/alecthomas/kong"
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

var (
	parser = participle.MustBuild(&pathExpr{})
	cli    struct {
		File string `arg:"" type:"existingfile" help:"File to parse."`
	}
)

func main() {
	ctx := kong.Parse(&cli)
	r, err := os.Open(cli.File)
	ctx.FatalIfErrorf(err)
	defer r.Close()

	var input map[string]interface{}
	err = json.NewDecoder(r).Decode(&input)
	ctx.FatalIfErrorf(err)

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")

	for {
		// Example: check_run.check_suite.pull_requests[0].head.sha
		fmt.Fprint(os.Stderr, "Enter a json path: ")
		q, err := bufio.NewReader(os.Stdin).ReadString('\n')
		ctx.FatalIfErrorf(err)

		var expr pathExpr
		if err := parser.ParseString(q, &expr); err != nil {
			fmt.Fprintln(os.Stderr, err)
			continue
		}

		result, err := match(input, expr)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			continue
		}
		enc.Encode(result)
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
