package comma

import (
	"strconv"
)

type Aggr interface {
	Aggr([]string) error
	Result() []float64
}

type min struct {
	values []float64
}

func Min() Aggr {
	var m min
	return &m
}

func (m *min) Aggr(row []string) error {
	if len(row) == 0 {
		return nil
	}
	n := len(m.values)
	if n == 0 {
		m.values = make([]float64, len(row))
	} else {
		if len(row) < n {
			return ErrRange
		}
	}
	for i, r := range row {
		f, err := strconv.ParseFloat(r, 64)
		if err != nil {
			return err
		}
		if n == 0 || f < m.values[i] {
			m.values[i] = f
		}
	}
	return nil
}

func (m *min) Result() []float64 {
	return m.values
}

type max struct {
	values []float64
}

func Max() Aggr {
	var m max
	return &m
}

func (m *max) Aggr(row []string) error {
	if len(row) == 0 {
		return nil
	}
	n := len(m.values)
	if n == 0 {
		m.values = make([]float64, len(row))
	} else {
		if len(row) < n {
			return ErrRange
		}
	}
	for i, r := range row {
		f, err := strconv.ParseFloat(r, 64)
		if err != nil {
			return err
		}
		if n == 0 || f > m.values[i] {
			m.values[i] = f
		}
	}
	return nil
}

func (m *max) Result() []float64 {
	return m.values
}

type sum struct {
	values []float64
}

func Sum() Aggr {
	var s sum
	return &s
}

func (s *sum) Aggr(vs []string) error {
	if len(vs) == 0 {
		return nil
	}
	if len(s.values) == 0 {
		s.values = make([]float64, len(vs))
	} else {
		if len(s.values) != len(vs) {
			return ErrRange
		}
	}
	for i, v := range vs {
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return err
		}
		s.values[i] += f
	}
	return nil
}

func (s *sum) Result() []float64 {
	return s.values
}

type count struct {
	values []int64
}

func Count() Aggr {
	var c count
	return &c
}

func (c *count) Aggr(vs []string) error {
	if len(vs) == 0 {
		return nil
	}
	if len(c.values) == 0 {
		c.values = make([]int64, len(vs))
	} else {
		if len(c.values) != len(vs) {
			return ErrRange
		}
	}
	for i := range c.values {
		c.values[i]++
	}
	return nil
}

func (c *count) Result() []float64 {
	vs := make([]float64, len(c.values))
	for i := range c.values {
		vs[i] = float64(c.values[i])
	}
	return vs
}

type mean struct {
	aggr  *sum
	count int
}

func Mean() Aggr {
	return &mean{aggr: new(sum)}
}

func (m *mean) Aggr(vs []string) error {
	err := m.aggr.Aggr(vs)
	if err == nil {
		m.count++
	}
	return err
}

func (m *mean) Result() []float64 {
	vs := m.aggr.Result()
	cs := make([]float64, len(vs))
	if m.count == 0 {
		return cs
	}
	c := float64(m.count)
	for i := range vs {
		cs[i] = vs[i] / c
	}
	return cs
}
