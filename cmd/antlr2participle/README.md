# A translator from Antlr to Participle

This command converts an Antlr `*.g4` grammar file into a Participle parser.

## Installation

```
go install github.com/alecthomas/participle/v2/cmd/antlr2participle
```

## Usage

```
antlr2participle grammar.g4
```

## Notes

- Lexer modes are not yet implemented.
- Recursive lexing is not yet implemented.
- The `skip` lexer command is supported.  The `channel` lexer command acts like `skip`.  No other lexer commands are supported yet.
- Actions and predicates are not supported.
- Rule element labels are partially supported.
- Alternative labels are parsed but not supported in the generator.
- Rule arguments are not supported.
