package comma

import (
	"fmt"
	"strconv"
	"time"
)

type Kind int

const (
	KindNull Kind = iota
	KindInt
	KindFloat
	KindDuration
	KindDate
	KindString
)

func (k Kind) String() string {
	switch k {
	default:
		return "unknown"
	case KindNull:
		return "null"
	case KindInt:
		return "integer"
	case KindFloat:
		return "float"
	case KindDuration:
		return "duration"
	case KindDate:
		return "date"
	case KindString:
		return "string"
	}
}

type Cell interface {
	// Reset() Cell
	// Update(Cell) Cell
	Kind() Kind
	fmt.Stringer
}

type String struct {
	Value string
}

func ParseString(v string) (Cell, error) {
	return String{v}, nil
}

func (s String) Kind() Kind { return KindString }

func (s String) String() string {
	return s.Value
}

type Float struct {
	Value float64
	Min   float64
	Max   float64
	Count int
}

func ParseFloat(v string) (Cell, error) {
	var f Float
	if v != "" {
		n, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return nil, err
		}
		f.Value, f.Min, f.Max = n, n, n
	}
	f.Count++
	return f, nil
}

func (f Float) Kind() Kind { return KindFloat }

func (f Float) Mean() float64 {
	if f.Count == 0 {
		return 0
	}
	return f.Value / float64(f.Count)
}

func (f Float) Update(other Cell) Cell {
	if x, ok := other.(Float); ok {
		f.Value += x.Value

		if f.Count == 1 || x.Value <= f.Min {
			f.Min = x.Value
		}
		if f.Count == 1 || x.Value >= f.Max {
			f.Max = x.Value
		}
		f.Count++
	}
	return f
}

func (f Float) String() string {
	return strconv.FormatFloat(f.Value, 'f', 2, 64)
}

type Int struct {
	Value int64
	Min   int64
	Max   int64
	Count int
}

func ParseInt(v string) (Cell, error) {
	var i Int
	if v != "" {
		n, err := strconv.ParseInt(v, 0, 64)
		if err != nil {
			return nil, err
		}
		i.Value, i.Min, i.Max = n, n, n
	}
	i.Count++
	return i, nil
}

func (i Int) Kind() Kind { return KindInt }

func (i Int) Mean() float64 {
	if i.Count == 0 {
		return 0
	}
	return float64(i.Value) / float64(i.Count)
}

func (i Int) Update(other Cell) Cell {
	if x, ok := other.(Int); ok {
		i.Value += x.Value

		if i.Count == 1 || x.Value <= i.Min {
			i.Min = x.Value
		}
		if i.Count == 1 || x.Value >= i.Max {
			i.Max = x.Value
		}
		i.Count++
	}
	return i
}

func (i Int) String() string {
	return strconv.FormatInt(i.Value, 10)
}

type Duration struct {
	Value time.Duration
	Min   time.Duration
	Max   time.Duration
	Count int
}

func ParseDuration(v string) (Cell, error) {
	var d Duration
	if v != "" {
		n, err := time.ParseDuration(v)
		if err != nil {
			return nil, err
		}
		if n < 0 {
			n = 0
		}
		d.Value, d.Min, d.Max = n, n, n
	}
	d.Count++
	return d, nil
}

func (d Duration) Kind() Kind { return KindDuration }

func (d Duration) Mean() time.Duration {
	if d.Count == 0 {
		return 0
	}
	return d.Value / time.Duration(d.Count)
}

func (d Duration) Update(other Cell) Cell {
	if x, ok := other.(Duration); ok {
		d.Value += x.Value
		if d.Count == 1 || x.Value <= d.Min {
			d.Min = x.Value
		}
		if d.Count == 1 || x.Value >= d.Max {
			d.Max = x.Value
		}
		d.Count++
	}
	return d
}

func (d Duration) String() string {
	return d.Value.String()
}
