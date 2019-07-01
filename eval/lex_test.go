package eval

import (
	"testing"
)

func TestLexer(t *testing.T) {
	data := []struct {
		Input string
		Want  []Token
	}{
		{
			Input: "=1+2",
			Want: []Token{
				{Type: assign},
				{Type: number, Literal: "1"},
				{Type: plus},
				{Type: number, Literal: "2"},
				{Type: eof},
			},
		},
		{
			Input: "=$1/($2+100)",
			Want: []Token{
				{Type: assign},
				{Type: index, Literal: "1"},
				{Type: divide},
				{Type: lparen},
				{Type: index, Literal: "2"},
				{Type: plus},
				{Type: number, Literal: "100"},
				{Type: rparen},
				{Type: eof},
			},
		},
		{
			Input: "$1::text || ($2 + $5)",
			Want: []Token{
				{Type: index, Literal: "1"},
				{Type: cast, Literal: "text"},
				{Type: or},
				{Type: lparen},
				{Type: index, Literal: "2"},
				{Type: plus},
				{Type: index, Literal: "5"},
				{Type: rparen},
				{Type: eof},
			},
		},
		{
			Input: "\"OPS\"=={INSTANCE}",
			Want: []Token{
				{Type: text, Literal: "OPS"},
				{Type: equal},
				{Type: env, Literal: "INSTANCE"},
				{Type: eof},
			},
		},
		{
			Input: "!($3 == {RAND})",
			Want: []Token{
				{Type: bang},
				{Type: lparen},
				{Type: index, Literal: "3"},
				{Type: equal},
				{Type: env, Literal: "RAND"},
				{Type: rparen},
				{Type: eof},
			},
		},
		{
			Input: "= -$3 + $2",
			Want: []Token{
				{Type: assign},
				{Type: minus},
				{Type: index, Literal: "3"},
				{Type: plus},
				{Type: index, Literal: "2"},
				{Type: eof},
			},
		},
		{
			Input: "= $-3 + $2",
			Want: []Token{
				{Type: assign},
				{Type: index, Literal: "-3"},
				{Type: plus},
				{Type: index, Literal: "2"},
				{Type: eof},
			},
		},
		{
			Input: "= -$-3 + $2",
			Want: []Token{
				{Type: assign},
				{Type: minus},
				{Type: index, Literal: "-3"},
				{Type: plus},
				{Type: index, Literal: "2"},
				{Type: eof},
			},
		},
	}
	for i, d := range data {
		x := lex(d.Input)
		for j := 0; ; j++ {
			if j >= len(d.Want) {
				t.Errorf("%d) too many tokens generated (%d >= %d)", i+1, j, len(d.Want))
				break
			}
			k := x.Next()
			if ok := cmpToken(k, d.Want[j]); !ok {
				t.Errorf("%d) invalid token! got %s, want %s", i+1, k, d.Want[j])
				break
			}
			if k.Type == eof {
				break
			}
		}
	}
}

func cmpToken(got, want Token) bool {
	if got.Type == want.Type {
		return got.Literal == want.Literal
	}
	return false
}
