// Package main generates internal files (e.g., JSON) for test cases
package main

import (
	"bytes"
	"encoding/json"
	"log"
	"os"

	"github.com/alecthomas/participle/v2/lexer"
	"github.com/alecthomas/participle/v2/lexer/internal"
)

func prettifyJSON(src []byte) ([]byte, error) {
	var prettyBytes bytes.Buffer
	err := json.Indent(&prettyBytes, src, "", "  ")
	if err != nil {
		return nil, err
	}
	return prettyBytes.Bytes(), nil
}

func generateLexerJSON(rules lexer.Rules, outputFilename string) error {
	lexerDef := lexer.MustStateful(rules)
	bytes, err := lexerDef.MarshalJSON()
	if err != nil {
		return err
	}
	prettyBytes, err := prettifyJSON(bytes)
	if err != nil {
		return err
	}
	return os.WriteFile(outputFilename, prettyBytes, 0644)
}

func main() {
	log.Println("Generating internal files...")
	if err := generateLexerJSON(internal.ARules, "lexer/internal/alexer.json"); err != nil {
		log.Fatal(err)
	}
	if err := generateLexerJSON(internal.BasicRules, "lexer/internal/basiclexer.json"); err != nil {
		log.Fatal(err)
	}
	if err := generateLexerJSON(internal.HeredocRules, "lexer/internal/heredoclexer.json"); err != nil {
		log.Fatal(err)
	}
	if err := generateLexerJSON(internal.HeredocWithWhitespaceRules, "lexer/internal/heredocwithwhitespacelexer.json"); err != nil {
		log.Fatal(err)
	}
	if err := generateLexerJSON(internal.InterpolatedRules, "lexer/internal/interpolatedlexer.json"); err != nil {
		log.Fatal(err)
	}
	if err := generateLexerJSON(internal.InterpolatedWithWhitespaceRules, "lexer/internal/interpolatedwithwhitespacelexer.json"); err != nil {
		log.Fatal(err)
	}
	if err := generateLexerJSON(internal.ReferenceRules, "lexer/internal/referencelexer.json"); err != nil {
		log.Fatal(err)
	}
	log.Println("Generated lexer json files...")
}
