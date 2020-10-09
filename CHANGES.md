<!-- TOC depthFrom:2 insertAnchor:true updateOnSave:true -->

- [v1](#v1)

<!-- /TOC -->

<a id="markdown-v1" name="v1"></a>
## v1

v1 was released in October 2020. It contains the following changes, some of
which are backwards-incompatible:

- Added optional `LexString()` and `LexBytes()` methods that lexer
  definitions can implement to fast-path lexing of bytes and strings.
- A `filename` must now be passed to all `Parse*()` and `Lex*()` methods.
- The `text/scanner` lexer no longer automatically unquotes strings or
  supports arbitary length single quoted strings. The tokens it produces are
  identical to that of the `text/scanner` package. Use `Unquote()` to remove
  quotes.
- `Tok` and `EndTok` are no longer supported.
- If a field named `Token []lexer.Token` exists it will be populated with the
  raw tokens that the node parsed from the lexer.
- Support capturing directly into lexer.Token fields. eg.

      type ast struct {
          Head lexer.Token   `@Ident`
          Tail []lexer.Token `@(Ident*)`
      }
- Improved performance of the stateul lexer by using a rune lookup table (thanks to @ceymard!).
- Add an `experimental/codegen` for stateful lexers. This provides ~10x
  performance improvement with zero garbage when lexing strings.

