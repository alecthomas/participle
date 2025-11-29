// Package main generates Railroad Diagrams from Participle grammar EBNF.
package main

import (
	"embed"
	"flag"
	"fmt"
	"os"

	"github.com/alecthomas/repr"

	"github.com/alecthomas/participle/v2/ebnf"
)

const (
	mergeRefThreshold  = -1
	mergeSizeThreshold = 0
)

type production struct {
	*ebnf.Production
	refs int
	size int
}

// Embed the railroad-diagrams css and js files for later output.
// From here: https://github.com/tabatkins/railroad-diagrams
//
//go:embed assets/*
var assets embed.FS

func generate(productions map[string]*production, n ebnf.Node) (s string) {
	switch n := n.(type) {
	case *ebnf.EBNF:
		s += `<!DOCTYPE html>
<style>
body {
	background-color: hsl(30,20%, 95%);
}
h1 {
	font-family: sans-serif;
	font-size: 1em;
}
</style>
<!-- From https://github.com/tabatkins/railroad-diagrams -->
<link rel='stylesheet' href='railroad-diagrams.css'>
<script src='railroad-diagrams.js'></script>
<body>
`
		for _, p := range n.Productions {
			s += generate(productions, p) + "\n"
		}
		s += "</body>\n"

	case *ebnf.Production:
		if productions[n.Production].refs <= mergeRefThreshold {
			break
		}
		s += `<h1 id="` + n.Production + `">` + n.Production + "</h1>\n"
		s += "<script>\n"
		s += "Diagram("
		s += generate(productions, n.Expression)
		s += ").addTo();\n"
		s += "</script>\n"

	case *ebnf.Expression:
		s += "Choice(0, "
		for i, a := range n.Alternatives {
			if i > 0 {
				s += ", "
			}
			s += generate(productions, a)
		}
		s += ")"

	case *ebnf.SubExpression:
		s += generate(productions, n.Expr)
		if n.Lookahead != ebnf.LookaheadAssertionNone {
			s = fmt.Sprintf(`Group(%s, "?%c")`, s, n.Lookahead)
		}

	case *ebnf.Sequence:
		s += "Sequence("
		for i, t := range n.Terms {
			if i > 0 {
				s += ", "
			}
			s += generate(productions, t)
		}
		s += ")"

	case *ebnf.Term:
		closeParen := false
		switch n.Repetition {
		case "*":
			s += "ZeroOrMore("
			closeParen = true
		case "+":
			s += "OneOrMore("
			closeParen = true
		case "?":
			s += "Optional("
			closeParen = true
		}
		switch {
		case n.Name != "":
			p := productions[n.Name]
			if p.refs > mergeRefThreshold {
				s += fmt.Sprintf("NonTerminal(%q, {href:\"#%s\"})", n.Name, n.Name)
			} else {
				s += generate(productions, p.Expression)
			}

		case n.Group != nil:
			s += generate(productions, n.Group)

		case n.Literal != "":
			s += fmt.Sprintf("Terminal(%s)", n.Literal)

		case n.Token != "":
			s += fmt.Sprintf("NonTerminal(%q)", n.Token)

		default:
			panic(repr.String(n))

		}
		if closeParen {
			s += ")"
		}
		if n.Negation {
			s = fmt.Sprintf(`Group(%s, "~")`, s)
		}

	default:
		panic(repr.String(n))
	}
	return
}

func countProductions(productions map[string]*production, n ebnf.Node) (size int) {
	switch n := n.(type) {
	case *ebnf.EBNF:
		for _, p := range n.Productions {
			productions[p.Production] = &production{Production: p}
		}
		for _, p := range n.Productions {
			countProductions(productions, p)
		}
		for _, p := range n.Productions {
			if productions[p.Production].size <= mergeSizeThreshold {
				productions[p.Production].refs = mergeRefThreshold
			}
		}
	case *ebnf.Production:
		productions[n.Production].size = countProductions(productions, n.Expression)
	case *ebnf.Expression:
		for _, a := range n.Alternatives {
			size += countProductions(productions, a)
		}
	case *ebnf.SubExpression:
		size += countProductions(productions, n.Expr)
	case *ebnf.Sequence:
		for _, t := range n.Terms {
			size += countProductions(productions, t)
		}
	case *ebnf.Term:
		if n.Name != "" {
			productions[n.Name].refs++
			size++
		} else if n.Group != nil {
			size += countProductions(productions, n.Group)
		} else {
			size++
		}
	default:
		panic(repr.String(n))
	}
	return
}

func main() {
	fmt.Fprintln(os.Stderr, "Generates railroad diagrams from a Participle EBNF grammar on stdin.")
	fmt.Fprintln(os.Stderr, "  (EBNF is available from .String() on your parser)")
	fmt.Fprintln(os.Stderr, "  (Use control-D to end input)")

	help := flag.Bool("h", false, "output help and quit")
	writeAssets := flag.Bool("w", false, "write css and js files")
	outputFile := flag.String("o", "", "file to write html to")

	flag.Parse()
	if *help {

		flag.PrintDefaults()
		os.Exit(0)
	}

	ast, err := ebnf.Parse(os.Stdin)
	if err != nil {
		panic(err)
	}

	productions := map[string]*production{}
	countProductions(productions, ast)
	str := generate(productions, ast)

	if *outputFile != "" {
		err := os.WriteFile(*outputFile, []byte(str), 0644) // nolint
		if err != nil {
			panic(err)
		}

		if *writeAssets {
			err := writeAssetFiles()
			if err != nil {
				panic(err)
			}
		} else {
			fmt.Fprintln(os.Stderr, ">>> Copy railroad-diagrams.{css,js} from https://github.com/tabatkins/railroad-diagrams")
		}

		fmt.Fprintf(os.Stderr, ">>> File written: %s\n", *outputFile)
	} else {
		fmt.Println(str)
		fmt.Fprintln(os.Stderr, ">>> Copy railroad-diagrams.{css,js} from https://github.com/tabatkins/railroad-diagrams")
	}
}

func writeAssetFiles() (err error) {
	files, err := assets.ReadDir("assets")
	if err != nil {
		return
	}

	for _, f := range files {
		fileName := f.Name()
		data, err := assets.ReadFile(fmt.Sprintf("assets/%s", fileName))
		if err != nil {
			return err
		}
		err = os.WriteFile(fileName, data, 0644) // nolint
		if err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, ">>> File written: %s\n", fileName)
	}

	return
}
