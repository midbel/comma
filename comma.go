package comma

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"io"
	"io/ioutil"
	"os"
)

type Reader struct {
	closer io.Closer
	inner  *csv.Reader

	Err error
}

func Open(file string) (*Reader, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	return NewReader(f), nil
}

func NewReader(r io.Reader) *Reader {
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
		closer: c,
		inner:  rs,
	}
}

func (r *Reader) ReadAll() ([][]string, error) {
	if r.Err != nil {
		return nil, r.Err
	}
	rs, err := r.inner.ReadAll()
	if err != nil {
		r.Err = err
	} else {
		r.closer.Close()
		err = io.EOF
	}
	return rs, r.Err
}

func (r *Reader) Read() ([]string, error) {
	if r.Err != nil {
		return nil, r.Err
	}
	rs, err := r.inner.Read()
	if err != nil {
		r.Err = err
		if r.Err == io.EOF {
			r.closer.Close()
		}
	}
	return rs, r.Err
}
