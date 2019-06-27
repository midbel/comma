package eval

import (
	"bytes"
	"fmt"
	"strconv"
	"unicode/utf8"
)

const (
	bindLowest = -(iota + 1)
	bindAssign
	bindSum
	bindProd
	bindPrefix
	bindGroup
)

var bindings = map[rune]int{
	plus:   bindSum,
	minus:  bindSum,
	star:   bindProd,
	slash:  bindProd,
	lparen: bindGroup,
	rparen: bindGroup,
}

type index int

func (i index) Value(row []string) (float64, error) {
	x := int(i)
	if x < 0 || x >= len(row) {
		return 0, fmt.Errorf("index out of range")
	}
	return strconv.ParseFloat(row[x], 64)
}

type literal float64

func (i literal) Value(_ []string) (float64, error) {
	return float64(i), nil
}

type prefix struct {
	right   interface{}
	operand rune
}

type infix struct {
	left    interface{}
	right   interface{}
	operand rune
}

type assign struct {
	left  interface{}
	right interface{}
}

func (a assign) Eval(row []string) ([]string, error) {
	return nil, nil
}

type parser struct {
	lex *lexer

	curr token
	peek token

	infix  map[rune]func(interface{}) error
	prefix map[rune]func() error
}

func Parse(str string) *parser {
	var p parser

	p.lex = lex(str)
	p.infix = map[rune]func(interface{}) error{
		plus:  p.parseInfix,
		minus: p.parseInfix,
		slash: p.parseInfix,
		star:  p.parseInfix,
	}
	p.prefix = map[rune]func() error{
		minus:   p.parsePrefix,
		percent: p.parsePrefix,
		number:  p.parsePrefix,
	}

	p.nextToken()
	p.nextToken()

	return &p
}

func (p *parser) Parse() error {
	return nil
}

func (p *parser) parseExpression() error {
	return nil
}

func (p *parser) parsePrefix() error {
	return nil
}

func (p *parser) parseInfix(_ interface{}) error {
	return nil
}

func (p *parser) nextToken() {
	p.curr = p.peek
	p.peek = p.lex.Next()
}

const (
	dollar  = '$'
	percent = '%'
	plus    = '+'
	minus   = '-'
	star    = '*'
	slash   = '/'
	equal   = '='
	rparen  = ')'
	lparen  = '('
	space   = ' '
	bang    = '!'
	dot     = '.'
)

const (
	eof rune = -(iota + 1)
	number
	invalid
)

type token struct {
	Char    rune
	Literal string
}

type lexer struct {
	input  []byte
	offset int
}

func lex(str string) *lexer {
	x := lexer{input: []byte(str)}
	return &x
}

func (x *lexer) Next() token {
	var k token
	switch k.Char = x.nextRune(); {
	case isDigit(k.Char):
		var buf bytes.Buffer

		buf.WriteRune(k.Char)
		for {
			if r := x.peekRune(); !isDigit(r) {
				break
			} else {
				buf.WriteRune(x.nextRune())
			}
		}
		k.Literal, k.Char = buf.String(), number
	default:
	}
	return k
}

func (x *lexer) peekRune() rune {
	k, _ := utf8.DecodeRune(x.input[x.offset:])
	return k
}

func (x *lexer) nextRune() rune {
	if x.offset >= len(x.input) {
		return eof
	}
	k, nn := utf8.DecodeRune(x.input[x.offset:])
	if k == utf8.RuneError {
		if nn == 0 {
			return eof
		} else {
			return invalid
		}
	}
	x.offset += nn
	if k == space {
		return x.nextRune()
	}
	return k
}

func isDigit(k rune) bool {
	return (k >= '0' && k <= '9') || k == dot
}
