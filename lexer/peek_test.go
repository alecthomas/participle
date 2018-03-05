package lexer

import (
  "testing"

  "github.com/gotestyourself/gotestyourself/assert"
)

type staticLexer struct {
  tokens []Token
}

func (s *staticLexer) Next() Token {
  if len(s.tokens) == 0 {
    return EOFToken
  }
  t := s.tokens[0]
  s.tokens = s.tokens[1:]
  return t
}

func TestUpgrade(t *testing.T) {
  t0 := Token{Type: 1, Value: "moo"}
  t1 := Token{Type: 2, Value: "blah"}
  l := Upgrade(&staticLexer{tokens: []Token{t0, t1}})
  assert.Equal(t, t0, l.Peek(0))
  assert.Equal(t, t0, l.Peek(0))
  assert.Equal(t, t1, l.Peek(1))
  assert.Equal(t, t1, l.Peek(1))
  assert.Equal(t, EOFToken, l.Peek(2))
  assert.Equal(t, EOFToken, l.Peek(3))
}
