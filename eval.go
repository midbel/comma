package comma

import (
	"fmt"
	"strconv"
)

func ParseEval(str string) (Eval, error) {
	i, left, err := scanIndexEval(str)

	var e Eval
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
		return e, fmt.Errorf("unknown operation %c", str[i])
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
		return e, err
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

type Eval struct {
	left  int
	right int
	where int
	apply applyFunc
}

func (e Eval) Eval(row []string) ([]string, error) {
	v, err := checkAndGet(e.left-1, e.right-1, row, e.apply)
	if err == nil {
		if e.where == 0 {

		}
		row = append(row, strconv.FormatFloat(v, 'f', -1, 64))
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
