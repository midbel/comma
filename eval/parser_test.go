package eval

import (
	"strings"
	"testing"
)

func TestParseValue(t *testing.T) {
	data := []struct {
		Input string
		Type  Type
		Want  interface{}
	}{
		{Input: "\"helloworld\"", Want: "helloworld", Type: String},
		{Input: "\"helloworld\"::text", Want: "helloworld", Type: String},
		{Input: "2", Want: 2., Type: Number},
		{Input: "\"2\"::number", Want: 2., Type: Number},
		{Input: "2::text", Want: "2", Type: String},
		{Input: "substr(\"helloworld\", 5)::text", Want: "hello", Type: String},
		{Input: "substr(\"1000\", 2)::text", Want: "10", Type: String},
		{Input: "substr(\"1000\", 2)::number", Want: 10., Type: Number},
	}
	for i, d := range data {
		e, err := parseExpression(d.Input)
		if err != nil {
			t.Errorf("%d) fail to parse %s: %s", i+1, d.Input, err)
			continue
		}
		v, err := e.Value(nil)
		if err != nil {
			t.Errorf("%d) fail to get value: %s", i+1, err)
			continue
		}
		if typ := v.Type(); typ != d.Type {
			t.Errorf("%d) wrong type: want %s, got %s", i+1, d.Type, typ)
			continue
		}
		switch v := v.(type) {
		case Text:
			want, ok := d.Want.(string)
			if !ok {
				t.Errorf("%d) type mismatch: want 'string', got %T", i+1, d.Want)
				continue
			}
			got := string(v)
			if got != want {
				t.Errorf("%d) wrong value: want %s, got %s", i+1, want, got)
			}
		case Literal:
			want, ok := d.Want.(float64)
			if !ok {
				t.Errorf("%d) type mismatch: want 'float64', got %T", i+1, d.Want)
				continue
			}
			got := float64(want)
			if got != want {
				t.Errorf("%d) wrong value: want %f, got %f", i+1, want, got)
			}
		}
	}
}

func TestParseInfix(t *testing.T) {
	data := []struct {
		Input  string
		Want   string
		Values []string
		Result Value
	}{
		{
			Input:  "1+2",
			Want:   "(1 + 2)",
			Result: Literal(3),
		},
		{
			Input:  "1+2/100",
			Want:   "(1 + (2 / 100))",
			Result: Literal(1.02),
		},
		{
			Input:  "(1-2)/100",
			Want:   "((1 - 2) / 100)",
			Result: Literal(-0.01),
		},
		{
			Input:  "7 % 5",
			Want:   "(7 % 5)",
			Result: Literal(2),
		},
		{
			Input:  "\"hello\" + \" \" + \"world\"",
			Want:   "((hello +  ) + world)",
			Result: Text("hello world"),
		},
		{
			Input:  "\"hello\" + \"world\"",
			Want:   "(hello + world)",
			Result: Text("helloworld"),
		},
	}
	for i, d := range data {
		e, err := parseExpression(d.Input)
		if err != nil {
			t.Errorf("%d) fail to parse %s: %s", i+1, d.Input, err)
			continue
		}
		x, ok := e.(Infix)
		if !ok {
			t.Errorf("%d) expected <infix> expression, got %T", i+1, x)
			continue
		}
		if got := e.String(); got != d.Want {
			t.Errorf("%d) parsing error: want %s, got %s", i+1, d.Want, got)
			continue
		}
		r, err := x.Value(d.Values)
		if err != nil {
			t.Errorf("%d) fail to evaluate expression (%s): %s", i+1, d.Input, err)
			continue
		}
		switch d.Result.(type) {
		default:
		case Literal:
			v, ok := r.(Literal)
			if !ok {
				t.Errorf("%d) expected <literal>, got %T", i+1, r)
				continue
			}
			if v != d.Result {
				t.Errorf("%d) expression badly evaluate: want %s, got %s", i+1, d.Result, v)
			}
		case Text:
			v, ok := r.(Text)
			if !ok {
				t.Errorf("%d) expected <text>, got %T", i+1, r)
				continue
			}
			if v != d.Result {
				t.Errorf("%d) expression badly evaluate: want %s, got %s", i+1, d.Result, v)
			}
		}
	}
}

func TestParseTernary(t *testing.T) {
	data := []struct {
		Input  string
		Want   string
		Values []string
		Result Value
	}{
		{
			Input:  "$1 ? $1 : $2",
			Want:   "($1 ? $1 : $2)",
			Values: []string{"10", "20"},
			Result: Literal(10),
		},
		{
			Input:  "0 ? ($1+2) : $2+3",
			Want:   "(0 ? ($1 + 2) : ($2 + 3))",
			Values: []string{"5", "8"},
			Result: Literal(11),
		},
	}
	for i, d := range data {
		e, err := parseExpression(d.Input)
		if err != nil {
			t.Errorf("%d) fail to parse %s: %s", i+1, d.Input, err)
			continue
		}
		x, ok := e.(Ternary)
		if !ok {
			t.Errorf("%d) expected <ternary> expression, got %T", i+1, x)
			continue
		}
		if got := e.String(); got != d.Want {
			t.Errorf("%d) parsing error: want %s, got %s", i+1, d.Want, got)
			continue
		}
		v, err := x.Value(d.Values)
		if err != nil {
			t.Errorf("%d) fail to evaluate expression (%s): %s", i+1, d.Input, err)
			continue
		}
		if v, ok := v.(Literal); !ok {
			t.Errorf("%d) expected <literal>, got %T", i+1, v)
			continue
		} else {
			if v != d.Result {
				t.Errorf("%d) expression badly evaluate: want %s, got %s", i+1, d.Result, v)
			}
		}
	}
}

func TestParseAssign(t *testing.T) {
	data := []struct {
		Input  string
		Want   string
		Values []string
	}{
		{Input: "=$1+$2", Want: "<nil> = ($1 + $2)", Values: []string{"10", "90", "100"}},
		{Input: "$1=$1+$2", Want: "$1 = ($1 + $2)", Values: []string{"100", "90"}},
		{Input: "1=$1+$2", Want: "1 = ($1 + $2)", Values: []string{"100", "10", "90"}},
		{Input: "1=$1+1", Want: "1 = ($1 + 1)", Values: []string{"11", "10", "90"}},
		{Input: "1=$2+1", Want: "1 = ($2 + 1)", Values: []string{"91", "10", "90"}},
		{Input: "1=$-1+1", Want: "1 = ($-1 + 1)", Values: []string{"91", "10", "90"}},
		{Input: "1=$-2+1", Want: "1 = ($-2 + 1)", Values: []string{"11", "10", "90"}},
	}
	for i, d := range data {
		e, err := parseExpression(d.Input)
		if err != nil {
			t.Errorf("%d) fail to parse %s: %s", i+1, d.Input, err)
			continue
		}
		a, ok := e.(Assign)
		if !ok {
			t.Errorf("%d) expected <assign> expression, got %T", i+1, e)
			continue
		} else {
			if a.right == nil {
				t.Errorf("%d) parsing error: right field can not be nil", i+1)
				continue
			}
		}
		if got := e.String(); got != d.Want {
			t.Errorf("%d) parsing error: want %s, got %s", i+1, d.Want, got)
		}
		values := []string{"10", "90"}
		vs, err := a.Eval(values)
		if err != nil {
			t.Errorf("%d) evaluation error (%s): %s", i+1, d.Input, err)
			continue
		}
		got, want := strings.Join(vs, "+"), strings.Join(d.Values, "+")
		if got != want {
			t.Errorf("%d) evaluation error (%s): want %s, got %s", i+1, d.Input, want, got)
		}
	}
}

func parseExpression(str string) (Expression, error) {
	p, err := Parse(str)
	if err != nil {
		return nil, err
	}
	return p.ParseExpression()
}
