package eval

import (
	"errors"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"
)

func init() {
	rand.Seed(time.Now().Unix())
}

var (
	ErrZero  = errors.New("division by zero")
	ErrIndex = errors.New("index out of range")
)

type CastError struct {
	cast    string
	literal string
}

func failtocast(cast, literal string) error {
	return CastError{
		cast:    cast,
		literal: literal,
	}
}

func (e CastError) Error() string {
	return fmt.Sprintf("can not cast %s to %s", e.literal, e.cast)
}

type TypeError struct {
	left  Type
	right Type
	op    rune
}

func mismatch(op rune, left, right Type) error {
	return TypeError{
		left:  left,
		right: right,
		op:    op,
	}
}

func (e TypeError) Error() string {
	switch e.op {
	case and:
		return fmt.Sprintf("type mismatch %s && %s", e.left, e.right)
	case or:
		return fmt.Sprintf("type mismatch %s || %s", e.left, e.right)
	case equal:
		return fmt.Sprintf("type mismatch %s == %s", e.left, e.right)
	case notequal:
		return fmt.Sprintf("type mismatch %s != %s", e.left, e.right)
	case lesser:
		return fmt.Sprintf("type mismatch %s < %s", e.left, e.right)
	case lesseq:
		return fmt.Sprintf("type mismatch %s <= %s", e.left, e.right)
	case greater:
		return fmt.Sprintf("type mismatch %s > %s", e.left, e.right)
	case greateq:
		return fmt.Sprintf("type mismatch %s >= %s", e.left, e.right)
	default:
		return fmt.Sprintf("type mismatch %s %c %s", e.left, e.op, e.right)
	}
}

type Type int

const (
	unknown Type = iota
	Number
	String
	Boolean
)

func (t Type) String() string {
	switch t {
	case Number:
		return "number"
	case String:
		return "string"
	case Boolean:
		return "boolean"
	default:
		return "unknown"
	}
}

type Value interface {
	Type() Type
	fmt.Stringer
}

type Cast struct {
	Cast  string
	Inner Expression
}

func castTo(e Expression, typ string) Expression {
	return Cast{Cast: typ, Inner: e}
}

func (c Cast) Type() Type {
	switch c.Cast {
	case "number":
		return Number
	case "text":
		return String
	case "boolean":
		return Boolean
	default:
		return unknown
	}
}

func (c Cast) Value(row []string) (Value, error) {
	v, err := c.Inner.Value(row)
	if err == nil {
		switch c.Type() {
		default:
			return nil, failtocast(c.Cast, c.String())
		case Number:
			switch x := v.(type) {
			case Bool:
				if x {
					v = Literal(1)
				} else {
					v = Literal(0)
				}
			case Literal:
				v = x
			case Text:
				if x, err := strconv.ParseFloat(string(x), 64); err != nil {
					return nil, err
				} else {
					v = Literal(x)
				}
			}
		case Boolean:
			switch x := v.(type) {
			case Bool:
				v = v
			case Text:
				v = Bool(len(x) == 0)
			case Literal:
				v = Bool(x == 0)
			}
		case String:
			v = Text(v.String())
		}
	}
	return v, err
}

func (c Cast) String() string {
	var b strings.Builder
	b.WriteString(c.Inner.String())
	b.WriteRune(colon)
	b.WriteRune(colon)
	b.WriteString(c.Cast)

	return b.String()
}

type Literal float64

func (i Literal) Type() Type                      { return Number }
func (i Literal) String() string                  { return strconv.FormatFloat(float64(i), 'f', -1, 64) }
func (i Literal) Value(_ []string) (Value, error) { return i, nil }

type Text string

func (t Text) Type() Type                      { return String }
func (t Text) String() string                  { return string(t) }
func (t Text) Value(_ []string) (Value, error) { return t, nil }

type Bool bool

func (b Bool) Type() Type                      { return Boolean }
func (b Bool) String() string                  { return strconv.FormatBool(bool(b)) }
func (b Bool) Value(_ []string) (Value, error) { return b, nil }

type Internal string

func (i Internal) Type() Type {
	switch str := string(i); str {
	case "NOW", "RAND":
		return Number
	default:
		return String
	}
}

func (i Internal) String() string {
	str := string(i)
	switch str {
	default:
		if s, ok := os.LookupEnv(str); ok {
			str = s
		} else {
			str = "unset"
		}
		str = fmt.Sprintf("<env(%s)>", str)
	case "RAND":
		str = fmt.Sprintf("<RAND(%d)>", rand.Int())
	case "NOW":
		n := time.Now()
		str = fmt.Sprintf("<NOW(%s)>", n.Format(time.RFC3339))
	case "HOST":
		host, err := os.Hostname()
		if err != nil {
			host = "localhost"
		}
		str = fmt.Sprintf("<HOST(%s)>", host)
	}
	return str
}

func (i Internal) Value(_ []string) (Value, error) {
	return i, nil
}

type Identifier struct {
	Index int
	Cast  string
}

func (i Identifier) String() string {
	var b strings.Builder
	b.WriteString("$")
	b.WriteString(strconv.FormatInt(int64(i.Index), 10))
	if i.Cast != "" {
		b.WriteRune(colon)
		b.WriteRune(colon)
		b.WriteString(i.Cast)
	}
	return b.String()
}

func (i Identifier) Value(row []string) (Value, error) {
	x := i.Index
	if x < 0 {
		x = len(row) + x
	} else {
		x--
	}
	if x < 0 || x >= len(row) {
		return nil, ErrIndex
	}
	switch i.Cast {
	default:
		return nil, failtocast(i.Cast, row[x])
	case "", "float", "int":
		f, err := strconv.ParseFloat(row[x], 64)
		if err != nil {
			return nil, failtocast(i.Cast, row[x])
		}
		return Literal(f), nil
	case "bool":
		b, err := strconv.ParseBool(row[x])
		if err != nil {
			return nil, failtocast(i.Cast, row[x])
		}
		return Bool(b), nil
	case "text":
		return Text(row[x]), nil
	}
}
