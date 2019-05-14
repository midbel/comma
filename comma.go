package comma

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"strconv"
	"unicode"
	"unicode/utf8"
)

var (
	ErrRange  = errors.New("out of range")
	ErrEmpty  = errors.New("empty")
	ErrSyntax = errors.New("invalid syntax")
)

type Option func(*Reader) error

// func WithSeparator(r rune) Option {
// 	return func(r *Reader) error {
// 		return nil
// 	}
// }
//
// func WithTypes() Option {
//   return func(r *Reader) error {
//     return nil
//   }
// }

func WithSelection(v string) Option {
	return func(r *Reader) error {
		cs, err := ParseSelection(v, r.inner.FieldsPerRecord)
		if err == nil {
			r.indices = append(r.indices, cs...)
		}
		return err
	}
}

type Reader struct {
	io.Closer
	inner *csv.Reader

	indices []int

	Err error
}

func Open(file string, options ...Option) (*Reader, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	return NewReader(f, options...), nil
}

func NewReader(r io.Reader, options ...Option) *Reader {
	var cols int

	rb := bufio.NewReader(r)
	if bs, err := rb.Peek(4096); err == nil {
		rs := csv.NewReader(bytes.NewReader(bs))
		if _, err := rs.Read(); err == nil {
			cols = rs.FieldsPerRecord
		}
	}
	rs := csv.NewReader(rb)
	if cols > 0 {
		rs.FieldsPerRecord = cols
	}
	rs.TrimLeadingSpace = true

	var c io.Closer
	if x, ok := r.(io.Closer); ok {
		c = x
	} else {
		c = ioutil.NopCloser(r)
	}
	return &Reader{
		Closer: c,
		inner:  rs,
	}
}

func (r *Reader) Next() ([]string, error) {
	if r.Err != nil {
		return nil, r.Err
	}
	if r.Err != nil {
		return nil, r.Err
	}
	row, err := r.inner.Read()
	if err != nil {
		r.Err = err
	} else {
		if len(r.indices) > 0 {
			ds := make([]string, r.inner.FieldsPerRecord)
			for i, ix := range r.indices {
				ds[i] = row[ix]
			}
			row = ds
		}
	}
	return row, r.Err
}

const (
	colon   = ':'
	virgule = ','
)

func ParseSelection(v string, fields int) ([]int, error) {
	if len(v) == 0 {
		vs := make([]int, fields)
		for i := 0; i < fields; i++ {
			vs[i] = i
		}
		return vs, nil //ErrEmpty
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
		case k == virgule || k == utf8.RuneError:
			i, err := parseIndex(str.String(), fields)
			if !interval {
				if err != nil {
					return nil, err
				}
			} else {
				if last, _ := utf8.DecodeLastRuneInString(v[:n-nn]); last == colon {
					i = fields - 1
				}
			}
			if interval {
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
				i, err := parseIndex(str.String(), fields)
				if err != nil {
					return nil, err
				}
				j = i
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

func parseIndex(str string, lim int) (int, error) {
	i, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return 0, err
	}
	if i < 1 || i > int64(lim) {
		return 0, ErrRange
	}
	i--
	return int(i), nil
}
