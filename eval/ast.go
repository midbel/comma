package eval

import (
	"fmt"
	"math"
	"strings"
)

type Infix struct {
	operator rune
	left     Expression
	right    Expression
}

func (x Infix) String() string {
	var b strings.Builder

	b.WriteRune(lparen)
	b.WriteString(x.left.String())
	b.WriteRune(space)
	switch x.operator {
	default:
		b.WriteRune(x.operator)
	case or:
		b.WriteString("||")
	case and:
		b.WriteString("&&")
	case lesser:
		b.WriteString("<")
	case lesseq:
		b.WriteString("<=")
	case greater:
		b.WriteString(">")
	case greateq:
		b.WriteString(">=")
	case equal:
		b.WriteString("==")
	case notequal:
		b.WriteString("!=")
	}
	b.WriteRune(space)
	b.WriteString(x.right.String())
	b.WriteRune(rparen)

	return b.String()
}

func (x Infix) Value(row []string) (Value, error) {
	left, err := x.left.Value(row)
	if err != nil {
		return nil, err
	}
	right, err := x.right.Value(row)
	if err != nil {
		return nil, err
	}
	var v Value
	switch x.operator {
	default:
		return nil, fmt.Errorf("unsupported operator: %s %c %s", left.Type(), x.operator, right.Type())
	case plus:
		v, err = evalAdd(left, right)
	case minus:
		v, err = evalSubtract(left, right)
	case multiply:
		v, err = evalMultiply(left, right)
	case divide:
		v, err = evalDivide(left, right)
	case modulo:
		v, err = evalModulo(left, right)
	case or:
		v, err = evalOr(left, right)
	case and:
		v, err = evalAnd(left, right)
	case equal, notequal:
		v, err = evalEqual(left, right, x.operator == notequal)
	case lesser, lesseq:
		v, err = evalLesser(left, right, x.operator == lesseq)
	case greater, greateq:
		v, err = evalGreater(left, right, x.operator == greateq)
	}
	return v, err
}

type Prefix struct {
	operator rune
	right    Expression
}

func (x Prefix) String() string {
	var b strings.Builder

	b.WriteRune(lparen)
	b.WriteRune(x.operator)
	if x.right != nil {
		b.WriteString(x.right.String())
	} else {
		b.WriteString("<nil>")
	}
	b.WriteRune(rparen)

	return b.String()
}

func (x Prefix) Value(row []string) (Value, error) {
	v, err := x.right.Value(row)
	if err != nil {
		return nil, err
	}
	switch {
	default:
		return nil, fmt.Errorf("unsupported operator: %c%s", x.operator, v.Type())
	case x.operator == bang && v.Type() == Boolean:
		tmp := v.(Bool)
		v = Bool(!tmp)
	case x.operator == minus && v.Type() == Number:
		tmp := v.(Literal)
		v = Literal(-tmp)
	}
	return v, nil
}

type Ternary struct {
	cond  Expression
	left  Expression // consequence
	right Expression // alternative
}

func (t Ternary) String() string {
	var b strings.Builder

	b.WriteRune(lparen)
	b.WriteString(t.cond.String())
	b.WriteRune(space)
	b.WriteRune(question)
	b.WriteRune(space)
	if t.left == nil {
		b.WriteString("<nil>")
	} else {
		b.WriteString(t.left.String())
	}
	b.WriteRune(space)
	b.WriteRune(colon)
	b.WriteRune(space)
	if t.right == nil {
		b.WriteString("<nil>")
	} else {
		b.WriteString(t.right.String())
	}
	b.WriteRune(rparen)

	return b.String()
}

func (t Ternary) Value(row []string) (Value, error) {
	v, err := t.cond.Value(row)
	if err != nil {
		return nil, err
	}

	if isTrue(v) {
		return t.left.Value(row)
	} else {
		return t.right.Value(row)
	}
}

type Assign struct {
	left  Expression
	right Expression
}

func (a Assign) String() string {
	var b strings.Builder
	if a.left == nil {
		b.WriteString("<nil>")
	} else {
		b.WriteString(a.left.String())
	}
	b.WriteRune(space)
	b.WriteRune(assign)
	b.WriteRune(space)
	if a.right == nil {
		b.WriteString("<nil>")
	} else {
		b.WriteString(a.right.String())
	}
	return b.String()
}

func (a Assign) Eval(row []string) ([]string, error) {
	right, err := a.Value(row)
	if err != nil {
		return nil, err
	}
	if a.left == nil {
		return append(row, right.String()), nil
	}
	var (
		ix  int
		lit bool
	)
	switch i := a.left.(type) {
	default:
		return nil, fmt.Errorf("oups")
	case Literal:
		ix, lit = int(i), true
	case Identifier:
		ix = int(i.Index)
	}
	ix--
	if ix < 0 || ix >= len(row) {
		return nil, ErrIndex
	}

	if str := right.String(); lit {
		row = append(row[:ix], append([]string{str}, row[ix:]...)...)
	} else {
		row[ix] = str
	}
	return row, nil
}

func (a Assign) Value(row []string) (Value, error) {
	return a.right.Value(row)
}

func isTrue(v Value) bool {
	if v.Type() == Boolean {
		tmp := v.(Bool)
		return bool(tmp)
	}
	if v.Type() == Number {
		tmp := v.(Literal)
		return tmp != 0
	}
	if v.Type() == String {
		tmp := v.(Text)
		return len(tmp) > 0
	}
	return false
}

func isEqual(left, right Value, not bool) (bool, error) {
	var b bool
	if left.Type() == Number && right.Type() == Number {
		b = left.(Literal) == right.(Literal)
	} else if left.Type() == String && right.Type() == String {
		b = left.(Text) == right.(Text)
	} else if left.Type() == Boolean && right.Type() == Boolean {
		b = left.(Bool) == right.(Bool)
	} else {
		var op rune = equal
		if not {
			op = notequal
		}
		return false, mismatch(op, left.Type(), right.Type())
	}
	if not {
		b = !b
	}
	return b, nil
}

func evalEqual(left, right Value, not bool) (Value, error) {
	b, err := isEqual(left, right, not)
	return Bool(b), err
}

func evalGreater(left, right Value, equal bool) (Value, error) {
	b, err := isEqual(left, right, false)
	if err != nil {
		return nil, err
	}
	if equal && b {
		return Bool(b), nil
	}
	if left.Type() == Number && right.Type() == Number {
		b = left.(Literal) > right.(Literal)
	} else if left.Type() == String && right.Type() == String {
		b = left.(Text) > right.(Text)
	} else {
		var op rune = greater
		if equal {
			op = greateq
		}
		return nil, mismatch(op, left.Type(), right.Type())
	}
	return Bool(b), nil
}

func evalLesser(left, right Value, equal bool) (Value, error) {
	b, err := isEqual(left, right, false)
	if err != nil {
		return nil, err
	}
	if equal && b {
		return Bool(b), nil
	}
	if left.Type() == Number && right.Type() == Number {
		b = left.(Literal) < right.(Literal)
	} else if left.Type() == String && right.Type() == String {
		b = left.(Text) < right.(Text)
	} else {
		var op rune = lesser
		if equal {
			op = lesseq
		}
		return nil, mismatch(op, left.Type(), right.Type())
	}
	return Bool(b), nil
}

func evalAnd(left, right Value) (Value, error) {
	v := isTrue(left) && isTrue(right)
	return Bool(v), nil
}

func evalOr(left, right Value) (Value, error) {
	if left.Type() == Number && right.Type() == Number {
		if isTrue(left) {
			return left, nil
		} else {
			return right, nil
		}
	}
	v := isTrue(left) || isTrue(right)
	return Bool(v), nil
}

func evalAdd(left, right Value) (Value, error) {
	if left.Type() == Number && right.Type() == Number {
		x, y := left.(Literal), right.(Literal)
		return Literal(x + y), nil
	} else if left.Type() == String && right.Type() == String {
		x, y := left.(Text), right.(Text)
		return Text(x + y), nil
	} else {
		return nil, mismatch(plus, left.Type(), right.Type())
	}
}

func evalSubtract(left, right Value) (Value, error) {
	if left.Type() == Number && right.Type() == Number {
		x, y := left.(Literal), right.(Literal)
		return Literal(x - y), nil
	} else {
		return nil, mismatch(minus, left.Type(), right.Type())
	}
}

func evalMultiply(left, right Value) (Value, error) {
	if left.Type() == Number && right.Type() == Number {
		x, y := left.(Literal), right.(Literal)
		return Literal(x * y), nil
	} else if left.Type() == Number && right.Type() == String {
		x := left.(Literal)
		y := right.(Text)
		return Text(strings.Repeat(string(y), int(x))), nil
	} else if left.Type() == String && right.Type() == Number {
		x := left.(Text)
		y := right.(Literal)
		return Text(strings.Repeat(string(x), int(y))), nil
	} else {
		return nil, mismatch(multiply, left.Type(), right.Type())
	}
}

func evalDivide(left, right Value) (Value, error) {
	if left.Type() == Number && right.Type() == Number {
		x, y := left.(Literal), right.(Literal)
		if y == 0 {
			return nil, ErrZero
		}
		return Literal(x / y), nil
	} else {
		return nil, mismatch(divide, left.Type(), right.Type())
	}
}

func evalModulo(left, right Value) (Value, error) {
	if left.Type() == Number && right.Type() == Number {
		x, y := left.(Literal), right.(Literal)
		if y == 0 {
			return nil, ErrZero
		}
		v := math.Mod(float64(x), float64(y))
		return Literal(v), nil
	} else {
		return nil, mismatch(modulo, left.Type(), right.Type())
	}
}
