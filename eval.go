package comma

import (
	"fmt"
	"strconv"
)

type Eval interface {
	Eval([]string) ([]string, error)
}

func Evaluate(vs []string) (Eval, error) {
	es := make([]Eval, 0, len(vs))
	for _, s := range vs {
		e, err := ParseEval(s)
		if err != nil {
			return nil, err
		}
		es = append(es, e)
	}
	return multieval{es}, nil
}

func ParseEval(str string) (Eval, error) {
	var (
		j int
		e eval
	)
	for j < len(str) && str[j] != '=' {
		j++
	}
	if j == len(str) {
		return nil, ErrSyntax
	}

	if j > 0 {
		e.replace = str[0] == '%'

		let := str[:j]
		if e.replace {
			let = let[1:]
		}
		if c, err := strconv.ParseInt(let, 10, 64); err != nil {
			return nil, err
		} else {
			e.where = int(c - 1)
		}
	}
	j++

	i, left, err := scanIndexEval(str[j:])
	if err != nil {
		return nil, err
	}
	i += j

	switch str[i] {
	case '+':
		e.apply = add
	case '-':
		e.apply = subtract
	case '/':
		e.apply = divide
	case '*':
		e.apply = multiply
	default:
		return nil, fmt.Errorf("unknown operation %c", str[i])
	}
	i++
	for i < len(str[i:]) {
		if c := str[i]; c == ' ' || c == '\t' {
			i++
		} else {
			break
		}
	}
	_, right, err := scanIndexEval(str[i:])
	if err != nil {
		return nil, err
	}
	e.left, e.right = left, right
	return e, nil
}

func scanIndexEval(str string) (int, int, error) {
	if str[0] != '%' {
		return -1, 0, fmt.Errorf("expected %% but got %c", str[0])
	}
	i := 1
	for i < len(str) {
		if c := str[i]; c >= '0' && c <= '9' {
			i++
		} else {
			break
		}
	}
	c, err := strconv.ParseInt(str[1:i], 10, 64)
	if err != nil {
		return -1, 0, err
	}
	for i < len(str) {
		if c := str[i]; c == ' ' || c == '\t' {
			i++
		} else {
			break
		}
	}
	return i, int(c), nil
}

type multieval struct {
	evals []Eval
}

func (e multieval) Eval(row []string) ([]string, error) {
	var err error
	for _, x := range e.evals {
		row, err = x.Eval(row)
		if err != nil {
			break
		}
	}
	return row, err
}

type eval struct {
	left  int
	right int

	replace bool
	where   int

	apply applyFunc
}

func (e eval) Eval(row []string) ([]string, error) {
	v, err := checkAndGet(e.left-1, e.right-1, row, e.apply)
	if err == nil {
		str := strconv.FormatFloat(v, 'f', -1, 64)
		if e.where == 0 {
			row = append(row, str)
		} else {
			if e.where < 0 || e.where >= len(row) {
				return nil, ErrRange
			}
			if e.replace {
				row[e.where] = str
			} else {
				row = append(row[:e.where], append([]string{str}, row[e.where:]...)...)
			}
		}
	}
	return row, err
}

type applyFunc func(float64, float64) (float64, error)

func checkAndGet(left, right int, row []string, fn applyFunc) (float64, error) {
	if left < 0 || right < 0 || left >= len(row) || right >= len(row) {
		return 0, ErrRange
	}
	rs := make([]float64, 2)
	for j, i := range []int{left, right} {
		v, err := parseFloat(row[i])
		if err != nil {
			return 0, err
		}
		rs[j] = v
	}
	return fn(rs[0], rs[1])
}

func add(x, y float64) (float64, error) {
	return x + y, nil
}

func subtract(x, y float64) (float64, error) {
	return x - y, nil
}

func divide(x, y float64) (float64, error) {
	if y == 0 {
		return 0, fmt.Errorf("division by zero")
	}
	return x / y, nil
}

func multiply(x, y float64) (float64, error) {
	return x * y, nil
}
