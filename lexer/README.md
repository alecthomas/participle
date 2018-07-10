# Participle Lexers

## Stateful lexer

Example: string interpolation.

```
hello ${first + "Surname: ${last}"}
```

This can be represented by the following stateful lexer grammar:

```
Enter = "${" Push(Interpolated) .
Text = any .

Interpolated {
  Leave = "}" Pop() .
  Ident = alpha { alpha | number } .
  Number = number { number } .
  Whitespace = "\n" | "\r" | "\t" | " " .
  Operator = "+" | "-" | "*" | "/"
  String = "\"" { "\\" ${" Push(Interpolated) | } "\"" .
}

alpha = "a"…"z" | "A"…"Z" | "_" .
number = "0"…"9" .
any = "\u0000" … "\uffff" .
```

## EBNF

## Regexp

## text/scanner based lexer
