package parser

import (
	"fmt"
	"reflect"
	"text/scanner"
)

type builder func(s scanner.Scanner, target reflect.Value)

type Parser struct {
	builder builder
}

func New(grammar interface{}) (*Parser, error) {
	t := reflect.TypeOf(grammar)
	if t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("grammar must be a struct")
	}
	p := &Parser{}
	return p, nil
}
