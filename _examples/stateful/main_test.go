package main

import (
	"log"
	"testing"

	"github.com/alecthomas/repr"
)

func TestExe(t *testing.T) {
	actual := &String{}
	err := parser.ParseString("", `"hello $(world) ${first + "${last}"}"`, actual)
	if err != nil {
		log.Fatal(err)
	}
	repr.Println(actual)
}
