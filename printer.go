package participle

import (
	"fmt"
	"reflect"
	"strings"
)

func dumpNode(v node) string {
	seen := map[reflect.Value]bool{}
	return nodePrinter(seen, v)
}

func nodePrinter(seen map[reflect.Value]bool, v node) string {
	if seen[reflect.ValueOf(v)] {
		return "<>"
	}
	seen[reflect.ValueOf(v)] = true
	switch n := v.(type) {
	case disjunction:
		out := []string{}
		for _, n := range n {
			out = append(out, nodePrinter(seen, n))
		}
		return strings.Join(out, "|")

	case *strct:
		return fmt.Sprintf("strct(type=%s, expr=%s)", n.typ, nodePrinter(seen, n.expr))

	case sequence:
		out := []string{}
		for _, n := range n {
			out = append(out, nodePrinter(seen, n))
		}
		return fmt.Sprintf("(%s)", strings.Join(out, " "))

	case *reference:
		return fmt.Sprintf("@(field=%s, node=%s)", n.field.Name, nodePrinter(seen, n.node))

	case *tokenReference:
		return fmt.Sprintf("token(%q)", n.identifier)

	case *optional:
		return fmt.Sprintf("[%s]", nodePrinter(seen, n.node))

	case *repetition:
		return fmt.Sprintf("{ %s }", nodePrinter(seen, n.node))

	case *literal:
		return n.String()

	}
	return "?"
}

type definitionsList struct {
	definitionsMap map[string]node
	definitionsList []string
}

func (d *definitionsList) addDefinition(definitionName string, n node) {
	if _, ok := d.definitionsMap[definitionName]; ok {
		return
	}

	d.definitionsMap[definitionName] = n
	d.definitionsList = append(d.definitionsList, definitionName)
}

func (d *definitionsList) getKeys() []string {
	return d.definitionsList
}

func (d *definitionsList) getValue(key string) node {
	return d.definitionsMap[key]
}

func dumpEbnfNode(v node) string {
	seen := map[reflect.Value]bool{}
	definitions := &definitionsList{map[string]node{},[]string{}}
	result := ""

	findDefinitions(seen, definitions, v)
	for _, definition := range definitions.getKeys() {
		node := definitions.getValue(definition)
		parsedDef := parseDefinition(node)
		result += fmt.Sprintf("%s := %s . \n", definition, parsedDef)
	}

	return result
}

func findDefinitions(seen map[reflect.Value]bool, definitions *definitionsList, v node) {
	if seen[reflect.ValueOf(v)] {
		return
	}
	seen[reflect.ValueOf(v)] = true
	switch n := v.(type) {
	case disjunction:
		for _, n := range n {
			findDefinitions(seen, definitions, n)
		}
		return

	case *strct:
		findDefinitions(seen, definitions, n.expr)
		return

	case sequence:
		for _, n := range n {
			findDefinitions(seen, definitions, n)
		}
		return

	case *reference:
		definitions.addDefinition(n.field.Name, n.node)
		findDefinitions(seen, definitions, n.node)
		return

	case *tokenReference:
		return

	case *optional:
		findDefinitions(seen, definitions, n.node)
		return

	case *repetition:
		findDefinitions(seen, definitions, n.node)
		return

	case *literal:
		return

	}
}

func parseDefinition(v node) string {
	switch n := v.(type) {
	case disjunction:
		out := []string{}
		for _, n := range n {
			out = append(out, n.Definition())
		}
		return strings.Join(out, "|")

	case *strct:
		return parseDefinition(n.expr)

	case sequence:
		out := []string{}
		for _, n := range n {
			out = append(out, n.Definition())
		}
		return strings.Join(out, " ")

	case *reference:
		return n.field.Name

	case *tokenReference:
		return fmt.Sprintf("token(%q)", n.Definition())

	case *optional:
		return fmt.Sprintf("[%s]", parseDefinition(n.node))

	case *repetition:
		return fmt.Sprintf("{ %s }", parseDefinition(n.node))

	case *literal:
		return n.Definition()

	}
	return "?"
}