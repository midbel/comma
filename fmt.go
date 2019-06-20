package comma

import (
	"encoding/base64"
	"fmt"
	"math/rand"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/midbel/sizefmt"
	"github.com/midbel/timefmt"
)

type formatter struct {
	Index  int
	Format func(string) (string, error)
}

func formatString(method string) func(string) (string, error) {
	return func(v string) (string, error) {
		switch method {
		case "title":
			v = strings.Title(v)
		case "upper":
			v = strings.ToUpper(v)
		case "lower":
			v = strings.ToLower(v)
		case "base":
			v = filepath.Base(v)
		case "dir":
			v = filepath.Dir(v)
		case "ext":
			v = filepath.Ext(v)
		case "file":
			v = strings.TrimSuffix(v, filepath.Ext(v))
		case "random":
			vs := []byte(v)
			rand.Shuffle(len(vs), func(i, j int) {
				vs[i], vs[j] = vs[j], vs[i]
			})
			v = string(vs)
		default:
		}
		return v, nil
	}
}

func formatDuration(resolution string) func(string) (string, error) {
	return func(v string) (string, error) {
		d, err := time.ParseDuration(v)
		if err == nil {
			switch resolution {
			case "", "seconds":
				if d < time.Second {
					d = time.Second
				}
				v = fmt.Sprintf("%.0f", d.Seconds())
			case "minutes":
				v = fmt.Sprintf("%.0f", d.Minutes())
			}
		}
		return v, err
	}
}

func formatBase64(method string) func(string) (string, error) {
	e := base64.StdEncoding
	if method == "url" {
		e = base64.URLEncoding
	}
	return func(v string) (string, error) {
		return e.EncodeToString([]byte(v)), nil
	}
}

func formatEnum(str string) func(string) (string, error) {
	values := strings.FieldsFunc(str, func(r rune) bool { return r == '=' || r == ',' })
	set := make(map[string]string)
	for i := 0; i < len(values); i += 2 {
		if i+1 >= len(values) {
			return nil
		}
		k, v := strings.TrimSpace(values[i]), strings.TrimSpace(values[i+1])
		set[k] = v
	}
	return func(v string) (string, error) {
		s, ok := set[v]
		if !ok {
			s = v
		}
		return s, nil
	}
}

func formatInt(pattern string) func(string) (string, error) {
	if pattern == "" {
		pattern = "%d"
	}
	return func(v string) (string, error) {
		i, err := strconv.ParseInt(v, 0, 64)
		if err == nil {
			switch pattern {
			case "seconds":
				d := time.Duration(i) * time.Second
				v = d.String()
			default:
				v = fmt.Sprintf(pattern, i)
			}
		}
		return v, err
	}
}

func formatFloat(pattern string) func(string) (string, error) {
	if pattern == "" {
		pattern = "%f"
	}
	return func(v string) (string, error) {
		f, err := strconv.ParseFloat(v, 64)
		if err == nil {
			if pattern == "percent" {
				f *= 100.0
				pattern = "%.2f%%"
			}
			v = fmt.Sprintf(pattern, f)
		}
		return v, err
	}
}

func formatBool(method string) func(string) (string, error) {
	var t, f string
	switch method {
	case "onoff":
		t, f = "on", "off"
	case "yesno":
		t, f = "yes", "no"
	case "status":
		t, f = "enabled", "disabled"
	case "vx":
		t, f = "v", "x"
	default:
		t, f = "true", "false"
	}
	return func(v string) (string, error) {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return "", err
		}
		str := f
		if b {
			str = t
		}
		return str, nil
	}
}

func formatSize(method string) func(string) (string, error) {
	if method == "" {
		method = sizefmt.SI
	}
	return func(v string) (string, error) {
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return "", err
		}
		return sizefmt.Format(f, method), nil
	}
}

func formatDate(pattern string, fs []string) func(string) (string, error) {
	return func(v string) (string, error) {
		if pattern == "" {
			return v, nil
		}
		for _, f := range fs {
			w, err := timefmt.Parse(v, f)
			if err == nil {
				return timefmt.Format(w, pattern), nil
			}
		}
		return "", fmt.Errorf("invalid date/datetime")
	}
}
