package main

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"unicode"
	"unicode/utf8"
)

var (
	ErrEmpty  = errors.New("empty")
	ErrSyntax = errors.New("invalid syntax")
)

type Comma rune

func (c *Comma) Set(v string) error {
	k, _ := utf8.DecodeRuneInString(v)
	if k != utf8.RuneError {
		*c = Comma(k)
	} else {
		return fmt.Errorf("invalid separator provided %s", v)
	}
	return nil
}

func (c *Comma) Rune() rune {
	return rune(*c)
}

func (c *Comma) String() string {
	return fmt.Sprintf("%c", *c)
}

const (
	colon = ':'
	comma = ','
)

func ParseSelection(v string) ([]int, error) {
	if len(v) == 0 {
		return nil, ErrEmpty
	}
	var (
		n        int
		cs       []int
		str      bytes.Buffer
		interval bool
	)
	for {
		k, nn := utf8.DecodeRuneInString(v[n:])
		if k == utf8.RuneError && nn > 0 {
			return nil, ErrSyntax
		}
		n += nn
		switch {
		case k == comma || k == utf8.RuneError:
			i, err := strconv.ParseInt(str.String(), 10, 64)
			if err != nil {
				return nil, err
			}
			i--
			if i := int(i); interval {
				var j int
				if n := len(cs); n > 0 {
					j = cs[n-1]
				}
				if j < i {
					for k := j + 1; k < i; k++ {
						cs = append(cs, k)
					}
				} else {
					for k := j - 1; k > i; k-- {
						cs = append(cs, k)
					}
				}
			}
			str.Reset()
			cs, interval = append(cs, int(i)), false

			if k == utf8.RuneError {
				return cs, nil
			}
		case k == colon:
			var j int
			if str.Len() > 0 {
				i, err := strconv.ParseInt(str.String(), 10, 64)
				if err != nil {
					return nil, err
				}
				j = int(i) - 1
			}
			str.Reset()
			cs, interval = append(cs, j), true
		case unicode.IsDigit(k):
			str.WriteRune(k)
			for {
				k, nn = utf8.DecodeRuneInString(v[n:])
				if !unicode.IsDigit(k) {
					break
				}
				n += nn
				str.WriteRune(k)
			}
		default:
			return nil, ErrSyntax
		}
	}
}
