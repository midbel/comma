package comma

import (
	"bufio"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"unicode"
)

var (
	ErrRange  = errors.New("out of range")
	ErrEmpty  = errors.New("empty")
	ErrSyntax = errors.New("invalid syntax")
)

type Option func(*Reader) error

func WithSeparator(c rune) Option {
	return func(r *Reader) error {
		if unicode.IsPunct(c) || c == '|' || c == ' ' || c == '\t' {
			r.inner.Comma = c
		} else {
			return fmt.Errorf("invalid separator %c", c)
		}
		return nil
	}
}

func WithFormatters(specifiers []string) Option {
	split := func(s string) (string, string, string) {
		fields := strings.Split(s, ":")
		for len(fields) < 3 {
			fields = append(fields, "")
		}
		return fields[0], fields[1], fields[2]
	}
	return func(r *Reader) error {
		if len(specifiers) == 0 {
			return nil
		}
		for _, s := range specifiers {
			col, kind, pattern := split(s)
			ix, err := strconv.ParseInt(col, 10, 64)
			if err != nil {
				return err
			}
			ix--
			if ix < 0 {
				return ErrRange
			}
			f := func(v string) (string, error) {
				return v, nil
			}
			switch strings.ToLower(kind) {
			case "date":
				f = formatDate(pattern, []string{"%Y-%m-%d", "%Y/%m/%d", "%Y-%j", "%Y/%j"})
			case "datetime":
				f = formatDate(pattern, []string{"%Y-%m-%d %H:%M:%S"})
			case "timestamp":
				f = formatTimestamp(pattern)
			case "duration":
				f = formatDuration(pattern)
			case "int":
				f = formatInt(pattern)
			case "float", "double", "number":
				f = formatFloat(pattern)
			case "bool", "boolean":
				f = formatBool(pattern)
			case "string":
				f = formatString(pattern)
			case "base64":
				f = formatBase64(pattern)
			case "size":
				f = formatSize(pattern)
			case "enum":
				f = formatEnum(pattern)
			default:
				return fmt.Errorf("unkown column type %s", kind)
			}
			if f == nil {
				return ErrSyntax
			}
			r.formatters = append(r.formatters, formatter{Index: int(ix), Format: f})
		}
		return nil
	}
}

func WithSelection(v string) Option {
	return func(r *Reader) error {
		if v == "" {
			return nil
		}
		cs, err := ParseSelection(v)
		if err == nil {
			r.indices = append(r.indices, cs...)
		}
		return err
	}
}

type Reader struct {
	io.Closer
	inner *csv.Reader

	indices    []Selection
	formatters []formatter

	err error
}

func Open(file string, options ...Option) (*Reader, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	return NewReader(f, options...)
}

func NewReader(r io.Reader, options ...Option) (*Reader, error) {
	var rs Reader

	if x, ok := r.(io.Closer); ok {
		rs.Closer = x
	} else {
		rs.Closer = ioutil.NopCloser(r)
	}

	rb := bufio.NewReader(r)
	rs.inner = csv.NewReader(rb)
	rs.inner.TrimLeadingSpace = true

	for _, opt := range options {
		if err := opt(&rs); err != nil {
			return nil, err
		}
	}
	return &rs, nil
}

func (r *Reader) Err() error {
	return r.err
}

func (r *Reader) Filter(f *Filter) ([]string, error) {
	for {
		row, err := r.Next()
		if err != nil {
			return nil, err
		}
		if f == nil || f.Match(row) {
			return row, nil
		}
	}
}

func (r *Reader) Next() ([]string, error) {
	if r.err != nil {
		return nil, r.err
	}
	row, err := r.inner.Read()
	if err != nil {
		r.err = err
	} else {
		if len(r.formatters) > 0 {
			for _, f := range r.formatters {
				row[f.Index], err = f.Format(row[f.Index])
				if err != nil {
					return nil, err
				}
			}
		}
		if len(r.indices) > 0 {
			ds := make([]string, 0, len(r.indices))
			for _, ix := range r.indices {
				vs, err := ix.Select(row)
				if err != nil {
					r.err = err
					return nil, r.err
				}
				ds = append(ds, vs...)
			}
			row = ds
		}
	}
	return row, r.err
}
