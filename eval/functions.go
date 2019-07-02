package eval

import (
	"errors"
	"fmt"
	"math"
	"strings"
)

var (
	ErrArgNum  = errors.New("wrong number of arguments")
	ErrArgType = errors.New("wrong type of arguments")
)

var funcs = map[string]func(...Value) (Value, error){
	"len":      size,
	"contains": contains,
	"tolower":  toLower,
	"toupper":  toUpper,
	"title":    title,
	"rshift":   rshift,
	"lshift":   lshift,
	"sqrt":     sqrt,
	"abs":      abs,
}

func size(vs ...Value) (Value, error) {
	if len(vs) != 1 {
		return nil, ErrArgNum
	}
	t, ok := vs[0].(Text)
	if !ok {
		return nil, ErrArgType
	}
	return Literal(len(t)), nil
}

func contains(vs ...Value) (Value, error) {
	if len(vs) < 2 {
		return nil, ErrArgNum
	}
	t, ok := vs[0].(Text)
	if !ok {
		return nil, ErrArgType
	}
	str := string(t)
	for _, v := range vs[1:] {
		s, ok := v.(Text)
		if !ok {
			return nil, ErrArgType
		}
		if strings.Contains(str, string(s)) {
			return Bool(true), nil
		}
	}
	return Bool(false), nil
}

func title(vs ...Value) (Value, error) {
	if len(vs) != 1 {
		return nil, ErrArgNum
	}
	t, ok := vs[0].(Text)
	if !ok {
		return nil, ErrArgType
	}
	s := strings.Title(string(t))
	return Text(s), nil
}

func toLower(vs ...Value) (Value, error) {
	if len(vs) != 1 {
		return nil, ErrArgNum
	}
	t, ok := vs[0].(Text)
	if !ok {
		return nil, ErrArgType
	}
	s := strings.ToLower(string(t))
	return Text(s), nil
}

func toUpper(vs ...Value) (Value, error) {
	if len(vs) != 1 {
		return nil, ErrArgNum
	}
	t, ok := vs[0].(Text)
	if !ok {
		return nil, ErrArgType
	}
	s := strings.ToUpper(string(t))
	return Text(s), nil
}

func rshift(vs ...Value) (Value, error) {
	if len(vs) != 2 {
		return nil, ErrArgNum
	}
	i, ok := vs[0].(Literal)
	if !ok {
		return nil, ErrArgType
	}
	j, ok := vs[1].(Literal)
	if !ok {
		return nil, ErrArgType
	}
	if i < 0 || j < 0 {
		return nil, fmt.Errorf("rshift only on uint")
	}
	x := uint(i) << uint(j)
	return Literal(x), nil
}

func lshift(vs ...Value) (Value, error) {
	if len(vs) != 2 {
		return nil, ErrArgNum
	}
	i, ok := vs[0].(Literal)
	if !ok {
		return nil, ErrArgType
	}
	j, ok := vs[1].(Literal)
	if !ok {
		return nil, ErrArgType
	}
	if i < 0 || j < 0 {
		return nil, fmt.Errorf("lshift only on uint")
	}
	x := uint(i) >> uint(j)
	return Literal(x), nil
}

func abs(vs ...Value) (Value, error) {
	if len(vs) != 1 {
		return nil, ErrArgNum
	}
	i, ok := vs[0].(Literal)
	if !ok {
		return nil, ErrArgType
	}
	q := math.Abs(float64(i))
	return Literal(q), nil
}

func sqrt(vs ...Value) (Value, error) {
	if len(vs) != 1 {
		return nil, ErrArgNum
	}
	i, ok := vs[0].(Literal)
	if !ok {
		return nil, ErrArgType
	}
	q := math.Sqrt(float64(i))
	return Literal(q), nil
}
