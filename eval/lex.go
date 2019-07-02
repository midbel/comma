package eval

import (
	"fmt"
)

const (
	eof = -(iota + 1)
	variable
	env
	number
	text
	cast
	and
	or
	equal
	notequal
	lesser
	lesseq
	greater
	greateq
	invalid
)

const (
	null      = 0
	space     = ' '
	tab       = '\t'
	plus      = '+'
	minus     = '-'
	divide    = '/'
	multiply  = '*'
	index     = '$'
	modulo    = '%'
	lparen    = '('
	rparen    = ')'
	dot       = '.'
	assign    = '='
	quote     = '"'
	colon     = ':'
	bang      = '!'
	question  = '?'
	ampersand = '&'
	pipe      = '|'
	langle    = '<'
	rangle    = '>'
	lcurly    = '{'
	rcurly    = '}'
	caret     = '^'
	comma     = ','
)

type Token struct {
	Type    rune
	Literal string
}

func (t Token) String() string {
	switch t.Type {
	default:
		return fmt.Sprintf("<invalid(%c)>", t.Type)
	case eof:
		return "<eof>"
	case text:
		return fmt.Sprintf("<text(%s)>", t.Literal)
	case variable:
		return fmt.Sprintf("<variable(%s)>", t.Literal)
	case number:
		return fmt.Sprintf("<literal(%s)>", t.Literal)
	case cast:
		return fmt.Sprintf("<cast(%s)>", t.Literal)
	case env:
		return fmt.Sprintf("<env(%s)>", t.Literal)
	case space, tab:
		return "<space>"
	case plus:
		return "<plus>"
	case minus:
		return "<minus>"
	case divide:
		return "<divide>"
	case multiply:
		return "<multiply>"
	case index:
		return fmt.Sprintf("<index(%s)>", t.Literal)
	case modulo:
		return "<modulo>"
	case lparen, rparen:
		return fmt.Sprintf("<paren [%c]>", t.Type)
	case dot:
		return "<dot>"
	case assign:
		return "<assign>"
	case and:
		return "<and>"
	case or:
		return "<or>"
	case colon:
		return "<colon>"
	case question:
		return "<question>"
	case bang:
		return "<not>"
	case equal:
		return "<equal>"
	case notequal:
		return "<notequal>"
	case lesser:
		return "<lesser>"
	case greater:
		return "<greater>"
	case lesseq:
		return "<lesseq>"
	case greateq:
		return "<greateq>"
	}
}

type lexer struct {
	input []byte

	pos  int
	next int
	char byte
}

func lex(str string) *lexer {
	x := lexer{input: []byte(str)}
	x.readByte()

	return &x
}

func (x *lexer) Next() Token {
	x.skipWhitespace()

	var t Token
	switch {
	case isText(x.char):
		x.readText(&t)
	case isVariable(x.char, false):
		x.readVariable(&t)
	case isDigit(x.char, true):
		x.readNumber(&t)
	case isIndex(x.char):
		x.readIndex(&t)
	case x.char == null:
		t.Type = eof
	case isEnv(x.char):
		x.readEnv(&t)
	case isMath(x.char) || isPunct(x.char):
		t.Type = rune(x.char)
	case x.char == langle:
		t.Type = lesser
		if c := x.peekByte(); c == assign {
			t.Type = lesseq
			x.readByte()
		}
	case x.char == rangle:
		t.Type = greater
		if c := x.peekByte(); c == assign {
			t.Type = greateq
			x.readByte()
		}
	case x.char == assign:
		if c := x.peekByte(); c == assign {
			t.Type = equal
			x.readByte()
		} else {
			t.Type = rune(x.char)
		}
	case x.char == bang:
		if c := x.peekByte(); c == assign {
			t.Type = notequal
			x.readByte()
		} else {
			t.Type = rune(x.char)
		}
	case x.char == ampersand:
		if c := x.peekByte(); c == x.char {
			t.Type = and
			x.readByte()
		} else {
			t.Type = invalid
		}
	case x.char == pipe:
		if c := x.peekByte(); c == x.char {
			t.Type = or
			x.readByte()
		} else {
			t.Type = invalid
		}
	case x.char == colon:
		t.Type = colon
		if c := x.peekByte(); c == colon {
			x.readCast(&t)
		}
	default:
		t.Type = invalid
	}
	x.readByte()
	return t
}

func (x *lexer) readEnv(t *Token) {
	x.readByte()
	pos := x.pos
	for x.char >= 'A' && x.char <= 'Z' {
		x.readByte()
	}
	if x.char != rcurly {
		t.Type = invalid
		return
	}
	t.Literal, t.Type = string(x.input[pos:x.pos]), env
}

func (x *lexer) readText(t *Token) {
	x.readByte()
	pos := x.pos
	for !isText(x.char) {
		x.readByte()
	}
	t.Literal, t.Type = string(x.input[pos:x.pos]), text
}

func (x *lexer) readCast(t *Token) {
	x.readByte()
	x.readByte()
	pos := x.pos
	for x.char >= 'a' && x.char <= 'z' {
		x.readByte()
	}
	t.Literal, t.Type = string(x.input[pos:x.pos]), cast
	x.unreadByte()
}

func (x *lexer) readVariable(t *Token) {
	pos := x.pos
	for isVariable(x.char, true) {
		x.readByte()
	}
	t.Literal, t.Type = string(x.input[pos:x.pos]), variable
	x.unreadByte()
}

func (x *lexer) readIndex(t *Token) {
	x.readByte()
	pos := x.pos
	if x.char == minus {
		x.readByte()
	}
	for isDigit(x.char, false) {
		x.readByte()
	}
	t.Literal, t.Type = string(x.input[pos:x.pos]), index
	x.unreadByte()
}

func (x *lexer) readNumber(t *Token) {
	pos := x.pos
	for isDigit(x.char, false) {
		x.readByte()
	}
	if x.char == dot {
		x.readByte()
		for isDigit(x.char, false) {
			x.readByte()
		}
		if x.char == dot {
			t.Type = invalid
			return
		}
	}
	t.Literal, t.Type = string(x.input[pos:x.pos]), number
	x.unreadByte()
}

func (x *lexer) readByte() {
	if x.next >= len(x.input) {
		x.char = null
	} else {
		x.char = x.input[x.next]
	}
	x.pos = x.next
	x.next++
}

func (x *lexer) unreadByte() {
	x.next = x.pos
	x.pos--
}

func (x *lexer) peekByte() byte {
	if x.next >= len(x.input) {
		return 0
	}
	next := x.next
	for next < len(x.input) && isWhitespace(x.input[next]) {
		next++
	}
	return x.input[next]
}

func (x *lexer) skipWhitespace() {
	for isWhitespace(x.char) {
		x.readByte()
	}
}

func isText(x byte) bool {
	return x == quote
}

func isVariable(x byte, all bool) bool {
	ok := (x >= 'a' && x <= 'z') || (x >= 'A' && x <= 'Z')
	if all {
		ok = ok || x == '_' || isDigit(x, false)
	}
	return ok
}

func isMath(x byte) bool {
	return x == plus || x == minus || x == multiply || x == divide || x == modulo || x == caret || x == comma
}

func isIndex(x byte) bool {
	return x == index
}

func isEnv(x byte) bool {
	return x == lcurly
}

func isPunct(x byte) bool {
	return x == lparen || x == rparen || x == question
}

func isDigit(x byte, all bool) bool {
	ok := x >= '0' && x <= '9'
	if all {
		ok = ok || x == dot
	}
	return ok
}

func isWhitespace(x byte) bool {
	return x == space || x == tab
}
