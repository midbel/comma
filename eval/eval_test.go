package eval

import (
	"testing"
)

func TestLex(t *testing.T) {
	data := []struct {
		Input string
		Want  []rune
	}{
		{Input: "1+2", Want: []rune{number, plus, number}},
		{Input: "%3=%1+2", Want: []rune{percent, number, equal, percent, number, plus, number}},
		{Input: "%3=%1+2.1", Want: []rune{percent, number, equal, percent, number, plus, number}},
		{Input: "%3 = (%1 / %4) * 100", Want: []rune{percent, number, equal, lparen, percent, number, slash, percent, number, rparen, star, number}},
	}
	for i, d := range data {
		x := lex(d.Input)
		for j := 0; ; j++ {
			k := x.Next()
			if k.Char == eof {
				if j == 0 {
					t.Errorf("%d) no tokens scanned! want %d tokens", i+1, len(d.Want))
				}
				break
			}
			if j >= len(d.Want) {
				t.Errorf("%d) got tokens than expected (%s: got %d tokens, want %d tokens)", i+1, d.Input, j+1, len(d.Want))
				break
			}
			if d.Want[j] != k.Char {
				t.Errorf("%d) unexpected token! got: %02x, want: %02x (at %d)", i+1, k.Char, d.Want[j], x.offset)
				break
			}
		}
	}
}
