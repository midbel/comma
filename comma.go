package comma

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"unicode"
)

var (
	ErrRange  = errors.New("out of range")
	ErrEmpty  = errors.New("empty")
	ErrSyntax = errors.New("invalid syntax")
)

type Option func(*Reader) error

func WithSeparator(c rune) Option {
	return func(r *Reader) error {
		if unicode.IsPunct(c) || c == '|' || c == ' ' || c == '\t' {
			r.inner.Comma = c
		} else {
			return fmt.Errorf("invalid separator %c", c)
		}
		return nil
	}
}

func WithFormatters(specifiers []string) Option {
	split := func(s string) (string, string, string) {
		fields := strings.Split(s, ":")
		for len(fields) < 3 {
			fields = append(fields, "")
		}
		return fields[0], fields[1], fields[2]
	}
	return func(r *Reader) error {
		if len(specifiers) == 0 {
			return nil
		}
		for _, s := range specifiers {
			col, kind, pattern := split(s)
			ix, err := strconv.ParseInt(col, 10, 64)
			if err != nil {
				return err
			}
			ix--
			if ix < 0 {
				return ErrRange
			}
			f := func(v string) (string, error) {
				return v, nil
			}
			switch strings.ToLower(kind) {
			case "date":
				f = formatDate(pattern, []string{"%Y-%m-%d", "%Y/%m/%d"})
			case "datetime":
				f = formatDate(pattern, []string{"%Y-%m-%d %H:%M:%S"})
			case "duration":
				f = formatDuration(pattern)
			case "int":
				f = formatInt(pattern)
			case "float", "double", "number":
				f = formatFloat(pattern)
			case "bool", "boolean":
				f = formatBool(pattern)
			case "string":
				f = formatString(pattern)
			case "base64":
				f = formatBase64(pattern)
			case "size":
				f = formatSize(pattern)
			case "enum":
				f = formatEnum(pattern)
			default:
				return fmt.Errorf("unkown column type %s", kind)
			}
			if f == nil {
				return ErrSyntax
			}
			r.formatters = append(r.formatters, formatter{Index: int(ix), Format: f})
		}
		return nil
	}
}

func WithSelection(v string) Option {
	return func(r *Reader) error {
		if v == "" {
			return nil
		}
		cs, err := ParseSelection(v)
		if err == nil {
			r.indices = append(r.indices, cs...)
		}
		return err
	}
}

type Reader struct {
	io.Closer
	inner *csv.Reader

	indices    []Selection
	formatters []formatter

	err error
}

func Open(file string, options ...Option) (*Reader, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	return NewReader(f, options...)
}

func NewReader(r io.Reader, options ...Option) (*Reader, error) {
	var rs Reader

	if x, ok := r.(io.Closer); ok {
		rs.Closer = x
	} else {
		rs.Closer = ioutil.NopCloser(r)
	}

	rb := bufio.NewReader(r)
	rs.inner = csv.NewReader(rb)
	rs.inner.TrimLeadingSpace = true

	for _, opt := range options {
		if err := opt(&rs); err != nil {
			return nil, err
		}
	}
	return &rs, nil
}

func (r *Reader) Err() error {
	return r.err
}

func (r *Reader) Filter(m Matcher) ([]string, error) {
	for {
		row, err := r.Next()
		if err != nil {
			return nil, err
		}
		if m == nil || m.Match(row) {
			return row, nil
		}
	}
}

func (r *Reader) Next() ([]string, error) {
	if r.err != nil {
		return nil, r.err
	}
	row, err := r.inner.Read()
	if err != nil {
		r.err = err
	} else {
		if len(r.formatters) > 0 {
			for _, f := range r.formatters {
				row[f.Index], err = f.Format(row[f.Index])
				if err != nil {
					return nil, err
				}
			}
		}
		if len(r.indices) > 0 {
			ds := make([]string, 0, len(r.indices))
			for _, ix := range r.indices {
				vs, err := ix.Select(row)
				if err != nil {
					r.err = err
					return nil, r.err
				}
				ds = append(ds, vs...)
			}
			row = ds
		}
	}
	return row, r.err
}

type Matcher interface {
	Match(row []string) bool
}

type Expr interface {
	Matcher
	Set(int, string)
}

func ParseFilter(v string) (Matcher, error) {
	if len(v) == 0 {
		return always{}, nil
	}
	return parseFilter(strings.NewReader(v))
}

func parseFilter(r io.RuneScanner) (Matcher, error) {
	expr, err := parseExpression(r)
	if err != nil {
		return nil, err
	}
	var match Matcher = expr
	k, _, err := r.ReadRune()
	switch k {
	case 0:
		return match, nil
	case '&', '|':
		err := peekNext(k, r)
		if err != nil {
			return nil, err
		}
		a := and{op: k, left: expr}
		if a.right, err = parseFilter(r); err != nil {
			return nil, err
		}
		match = a
	default:
		return nil, ErrSyntax
	}
	return match, nil
}

func parseExpression(rs io.RuneScanner) (Expr, error) {
	defer skipSpaces(rs)

	ix, err := parseIndex(rs)
	if err != nil {
		return nil, err
	}

	m, err := parseOperator(rs)
	if err != nil {
		return nil, err
	}

	value, err := parseValue(rs)
	if err != nil {
		return nil, err
	}

	m.Set(ix, value)
	return m, nil
}

func skipSpaces(r io.RuneScanner) error {
	for {
		k, _, err := r.ReadRune()
		if err != nil {
			return err
		}
		if !unicode.IsSpace(k) {
			break
		}
	}
	return r.UnreadRune()
}

func parseValue(r io.RuneScanner) (string, error) {
	if err := skipSpaces(r); err != nil {
		return "", err
	}
	var str bytes.Buffer
	for {
		k, _, err := r.ReadRune()
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", err
		}
		if unicode.IsLetter(k) || unicode.IsDigit(k) {
			str.WriteRune(k)
		} else {
			r.UnreadRune()
			break
		}
	}
	return str.String(), nil
}

func parseIndex(r io.RuneScanner) (int, error) {
	if err := skipSpaces(r); err != nil {
		return 0, err
	}
	var str bytes.Buffer
	for {
		k, _, err := r.ReadRune()
		if err != nil {
			return 0, err
		}
		if !unicode.IsDigit(k) {
			r.UnreadRune()
			break
		}
		str.WriteRune(k)
	}
	i, err := strconv.ParseInt(str.String(), 10, 64)
	if err == nil {
		i--
	}
	return int(i), err
}

func peekNext(want rune, r io.RuneScanner) error {
	got, _, err := r.ReadRune()
	if err != nil {
		return err
	}
	if got != want {
		err = ErrSyntax
	}
	return err
}

func parseOperator(r io.RuneScanner) (Expr, error) {
	if err := skipSpaces(r); err != nil {
		return nil, err
	}
	k, _, err := r.ReadRune()
	if err != nil {
		return nil, err
	}

	var e Expr
	switch k {
	case '=', '!':
		if err = peekNext('=', r); err == nil {
			e = new(equal) // TODO: really ugly - rewrite!!!
			if k == '!' {
				e = &not{expr: e}
			}
		}
	case '~':
		e = new(almost)
	default:
		err = ErrSyntax
	}
	return e, err
}

type and struct {
	op    rune
	left  Matcher
	right Matcher
}

func (a and) Match(row []string) bool {
	if a.op == '|' {
		return a.matchOR(row)
	} else {
		return a.matchAND(row)
	}
}

func (a and) matchOR(row []string) bool {
	if ok := a.left.Match(row); ok {
		return ok
	}
	return a.right.Match(row)
}

func (a and) matchAND(row []string) bool {
	if ok := a.left.Match(row); !ok {
		return ok
	}
	return a.right.Match(row)
}

type almost struct {
	Index int
	Value string

	hint string
}

func (a *almost) Set(ix int, value string) {
	a.Index = ix
	a.Value = value
}

func (a *almost) Match(row []string) bool {
	if a.Index < 0 || a.Index >= len(row) {
		return false
	}
	return strings.Contains(row[a.Index], a.Value)
}

type equal struct {
	Index int
	Value string

	hint string
}

func (e *equal) Set(ix int, value string) {
	e.Index = ix
	e.Value = value
}

func (e *equal) Match(row []string) bool {
	if e.Value == "" {
		return true
	}
	if e.Index < 0 {
		return e.matchAny(row)
	}
	return e.matchStrict(row)
}

func (e equal) matchStrict(row []string) bool {
	return row[e.Index] == e.Value
}

func (e equal) matchAny(row []string) bool {
	for _, r := range row {
		if r == e.Value {
			return true
		}
	}
	return false
}

type always struct{}

func (_ always) Match(row []string) bool {
	return true
}

type not struct {
	expr Expr
}

func (n *not) Set(ix int, value string) {
	if n.expr != nil {
		n.expr.Set(ix, value)
	}
}

func (n *not) Match(row []string) bool {
	if n.expr == nil {
		return true
	}
	return !n.expr.Match(row)
}
