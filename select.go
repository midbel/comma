package comma

import (
	"bytes"
	"strconv"
	"unicode"
	"unicode/utf8"
)

const (
	colon   = ':'
	virgule = ','
)

type Selection struct {
	start    int
	end      int
	interval bool
	open     bool
}

func ParseSelection(v string) ([]Selection, error) {
	return parseSelection(v)
}

func (s Selection) IsOpen() bool {
	return s.interval && (s.start == 0 || s.end == 0)
}

func (s Selection) String() string {
	tmp := make([]byte, 0, 64)
	if s.interval {
		if s.start > 0 {
			tmp = strconv.AppendInt(tmp, int64(s.start), 10)
		}
		tmp = append(tmp, ':')
		if s.end > 0 {
			tmp = strconv.AppendInt(tmp, int64(s.end), 10)
		}
	} else {
		tmp = strconv.AppendInt(tmp, int64(s.start), 10)
	}
	return string(tmp)
}

func (s Selection) Select(values []string) ([]string, error) {
	if s.interval {
		return s.selectOpen(values)
	} else {
		return s.selectSingle(values)
	}
}

func (s Selection) selectOpen(values []string) ([]string, error) {
	var start, end int
	switch {
	case s.start == 0 && s.end > 0:
		end = s.end - 1
	case s.end == 0 && s.start > 0:
		start, end = s.start-1, len(values)-1
	case s.start == 0 && s.end == 0:
		start, end = 0, len(values)-1
	default:
		start, end = s.start-1, s.end-1
	}

	if start < 0 {
		return nil, ErrRange
	}
	if end >= len(values) {
		return nil, ErrRange
	}
	var reverse bool
	if start > end {
		reverse = !reverse
		end, start = start, end
	}
	vs := make([]string, (end+1)-start)
	if n := copy(vs, values[start:end+1]); reverse {
		for i, j := 0, n-1; i < n/2; i, j = i+1, j-1 {
			vs[i], vs[j] = vs[j], vs[i]
		}
	}
	return vs, nil
}

func (s Selection) selectSingle(values []string) ([]string, error) {
	if i := s.start - 1; i < 0 || i >= len(values) {
		return nil, ErrRange
	} else {
		return values[i : i+1], nil
	}
}

func parseSelection(v string) ([]Selection, error) {
	if len(v) == 0 {
		return nil, nil
	}
	var (
		n        int
		cs       []Selection
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
			var i int
			if str.Len() > 0 {
				j, err := strconv.ParseInt(str.String(), 10, 64)
				if err != nil {
					return nil, err
				}
				i = int(j)
				str.Reset()
			}
			if n := len(cs); n > 0 && cs[n-1].interval && interval {
				cs[n-1].end = i
			} else {
				cs = append(cs, Selection{start: i})
			}
			interval = false

			if k == utf8.RuneError {
				return cs, nil
			}
		case k == colon:
			var s Selection
			if str.Len() > 0 {
				i, err := strconv.ParseInt(str.String(), 10, 64)
				if err != nil {
					return nil, err
				}
				str.Reset()
				s.start = int(i)
			}
			s.open, s.interval = true, true
			cs = append(cs, s)
			interval = true
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
