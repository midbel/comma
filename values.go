package comma

import (
	"time"
)

type Value interface {
	Eq(Value) bool
	Lt(Value) bool
	Gt(Value) bool
	Le(Value) bool
	Ge(Value) bool
}

type real struct {
	value float64
}

func (r real) Eq(other Value) bool {
	x, ok := other.(real)
	if !ok {
		return false
	}
	return x.value == r.value
}

func (r real) Lt(other Value) bool {
	x, ok := other.(real)
	if !ok {
		return false
	}
	return r.value < x.value
}

func (r real) Le(other Value) bool {
	return r.Eq(other) || r.Lt(other)
}

func (r real) Gt(other Value) bool {
	return !r.Le(other)
}

func (r real) Ge(other Value) bool {
	return r.Eq(other) || r.Gt(other)
}

type datetime struct {
	value time.Time
}

func (d datetime) Eq(other Value) bool {
	x, ok := other.(datetime)
	if !ok {
		return false
	}
	return d.value.Equal(x.value)
}

func (d datetime) Lt(other Value) bool {
	x, ok := other.(datetime)
	if !ok {
		return false
	}
	return d.value.Before(x.value)
}

func (d datetime) Le(other Value) bool {
	return d.Eq(other) || d.Lt(other)
}

func (d datetime) Gt(other Value) bool {
	return !d.Le(other)
}

func (d datetime) Ge(other Value) bool {
	return d.Eq(other) || d.Gt(other)
}
