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
	"substr":   substring,
	"rshift":   rshift,
	"lshift":   lshift,
	"sqrt":     sqrt,
	"abs":      abs,
	"min":      min,
	"max":      max,
	"avg":      average,
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

func substring(vs ...Value) (Value, error) {
	if len(vs) < 2 {
		return nil, ErrArgNum
	}
	t, ok := vs[0].(Text)
	if !ok {
		return nil, ErrArgType
	}
	str := string(t)
	var from, to int

	ix := 1
	if len(vs) == 3 {
		if i, ok := vs[ix].(Literal); !ok {
			return nil, ErrArgType
		} else {
			from = int(i)
			if from < 0 || from >= len(str) {
				return nil, fmt.Errorf("index out of range %d", from)
			}
		}
		ix++
	}
	if i, ok := vs[ix].(Literal); !ok {
		return nil, ErrArgType
	} else {
		to = int(i)
		if to < 0 || to >= len(str) {
			return nil, fmt.Errorf("index out of range %d", to)
		}
	}
	if from >= to {
		return nil, fmt.Errorf("invalid range: %d-%d", from, to)
	}
	return Text(str[from:to]), nil
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

func min(vs ...Value) (Value, error) {
	var m Literal
	for i, v := range vs {
		v, ok := v.(Literal)
		if !ok {
			return m, ErrArgType
		}
		if i == 0 || v < m {
			m = v
		}
	}
	return m, nil
}

func max(vs ...Value) (Value, error) {
	var m Literal
	for i, v := range vs {
		v, ok := v.(Literal)
		if !ok {
			return m, ErrArgType
		}
		if i == 0 || v > m {
			m = v
		}
	}
	return m, nil
}

func average(vs ...Value) (Value, error) {
	if len(vs) == 0 {
		return Literal(0), nil
	}
	var m Literal
	for _, v := range vs {
		v, ok := v.(Literal)
		if !ok {
			return m, ErrArgType
		}
		m += v
	}
	return m / Literal(len(vs)), nil
}
