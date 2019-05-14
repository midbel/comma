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

func WithSelection(v string) Option {
	return func(r *Reader) error {
		cs, err := ParseSelection(v, r.fields)
		if err == nil {
			r.indices = append(r.indices, cs...)
		}
		return err
	}
}

type Reader struct {
	io.Closer
	inner *csv.Reader

	fields  int
	indices []int
	// types []string

	Err error
}

func Open(file string, sep rune, options ...Option) (*Reader, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	return NewReader(f, sep, options...)
}

func NewReader(r io.Reader, sep rune, options ...Option) (*Reader, error) {
	var rs Reader

	if x, ok := r.(io.Closer); ok {
		rs.Closer = x
	} else {
		rs.Closer = ioutil.NopCloser(r)
	}

	rb := bufio.NewReader(r)
	if bs, err := rb.Peek(4096); err == nil {
		rc := csv.NewReader(bytes.NewReader(bs))
		rc.Comma = sep
		rc.TrimLeadingSpace = true

		if _, err := rc.Read(); err == nil {
			rs.fields = rc.FieldsPerRecord
		}
	}
	for _, opt := range options {
		if err := opt(&rs); err != nil {
			return nil, err
		}
	}

	rs.inner = csv.NewReader(rb)
	rs.inner.Comma = sep
	rs.inner.TrimLeadingSpace = true
	if rs.fields > 0 {
		rs.inner.FieldsPerRecord = rs.fields
	}
	return &rs, nil
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
			ds := make([]string, len(r.indices))
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
		case unicode.IsSpace(k):
			if last, _ := utf8.DecodeLastRuneInString(v[:n-nn]); last != virgule {
				return nil, ErrSyntax
			}
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
