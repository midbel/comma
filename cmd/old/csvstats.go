package main

import (
	"bytes"
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/midbel/linewriter"
)

type Field interface {
	fmt.Stringer
}

type Columns []int

func (c *Columns) Set(v string) error {
	for _, v := range strings.Split(v, ",") {
		i, err := strconv.Atoi(v)
		if err != nil {
			return err
		}
		i--
		if i < 0 {
			return fmt.Errorf("invalid index")
		}
		*c = append(*c, i)
	}
	return nil
}

func (c *Columns) String() string {
	return fmt.Sprint(*c)
}

func (c *Columns) Ints() []int {
	return []int(*c)
}

type Date struct {
	Value time.Time
}

func ParseDate(v string) (Field, error) {
	w, err := time.Parse("2006-01-02", v)
	if err != nil {
		return nil, err
	}
	return Date{w.Truncate(time.Hour * 24)}, nil
}

func ParseDatetime(v string) (Field, error) {
	w, err := time.Parse("2006-01-02 15:04:05.000", v)
	if err != nil {
		return nil, err
	}
	return Date{w.Truncate(time.Hour * 24)}, nil
}

func (d Date) String() string {
	return d.Value.Format("2006-01-02")
}

type Int struct {
	Value int64
	count int
	Min   int64
	Max   int64
}

func ParseInt(v string) (Field, error) {
	var i Int
	if v != "" {
		n, err := strconv.ParseInt(v, 0, 64)
		if err != nil {
			return nil, err
		}
		i.Value = n
	}
	i.count++
	return i, nil
}

func (i Int) Mean() float64 {
	if i.count == 0 {
		return 0
	}
	return float64(i.Value) / float64(i.count)
}

func (i Int) Add(other Field) Field {
	if x, ok := other.(Int); ok {
		i.Value += x.Value

		if i.count == 1 || x.Value <= i.Min {
			i.Min = x.Value
		}
		if i.count == 1 || x.Value >= i.Max {
			i.Max = x.Value
		}
		i.count++
	}
	return i
}

func (i Int) String() string {
	return strconv.FormatInt(i.Value, 10)
}

type Float struct {
	Value float64
	Min   float64
	Max   float64
	count int
}

func ParseFloat(v string) (Field, error) {
	var f Float
	if v != "" {
		n, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return nil, err
		}
		f.Value = n
	}
	f.count++
	return f, nil
}

func (f Float) Mean() float64 {
	if f.count == 0 {
		return 0
	}
	return f.Value / float64(f.count)
}

func (f Float) String() string {
	return strconv.FormatFloat(f.Value, 'f', 2, 64)
}

func (f Float) Add(other Field) Field {
	if x, ok := other.(Float); ok {
		f.Value += x.Value

		if f.count == 1 || x.Value <= f.Min {
			f.Min = x.Value
		}
		if f.count == 1 || x.Value >= f.Max {
			f.Max = x.Value
		}
		f.count++
	}
	return f
}

type String struct {
	Value string
}

func ParseString(v string) (Field, error) {
	return String{v}, nil
}

func (s String) String() string {
	return s.Value
}

type Duration struct {
	Value time.Duration
	Min   time.Duration
	Max   time.Duration

	count int
}

func ParseDuration(v string) (Field, error) {
	var d Duration
	if v != "" {
		n, err := time.ParseDuration(v)
		if err != nil {
			return nil, err
		}
		if n < 0 {
			n = 0
		}
		d.Value = n
	}
	d.count++
	return d, nil
}

func (d Duration) Add(other Field) Field {
	if x, ok := other.(Duration); ok {
		d.Value += x.Value
		if d.count == 1 || x.Value <= d.Min {
			d.Min = x.Value
		}
		if d.count == 1 || x.Value >= d.Max {
			d.Max = x.Value
		}
		d.count++
	}
	return d
}

func (d Duration) Mean() time.Duration {
	if d.count == 0 {
		return 0
	}
	return d.Value / time.Duration(d.count)
}

func (d Duration) String() string {
	return d.Value.String()
}

type Multifield struct {
	Fields []Field
}

func (mf Multifield) String() string {
	var buf bytes.Buffer
	buf.WriteRune('[')
	for i := range mf.Fields {
		buf.WriteString(mf.Fields[i].String())
		if i < len(mf.Fields)-1 {
			buf.WriteRune('@')
		}
	}
	buf.WriteRune(']')
	return buf.String()
}

func main() {
	var cols, vals Columns
	comma := flag.Bool("s", false, "csv")
	flag.Var(&cols, "c", "columns")
	flag.Var(&vals, "x", "columns")
	flag.Parse()

	r, err := os.Open(flag.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(3)
	}
	defer r.Close()

	ps, err := sniffTypes(r)
	if err != nil {
		fmt.Fprintln(os.Stderr, "sniffing types:", err)
		os.Exit(12)
	}

	data, err := groupBy(r, ps, cols.Ints(), vals.Ints())
	if err != nil {
		fmt.Fprintln(os.Stderr, "aggregating data:", err)
		os.Exit(7)
	}
	var options []linewriter.Option
	if *comma {
		options = append(options, linewriter.AsCSV(false))
	} else {
		options = []linewriter.Option{
			linewriter.WithPadding([]byte(" ")),
			linewriter.WithSeparator([]byte("|")),
		}
	}
	line := linewriter.NewWriter(1024, options...)
	for _, g := range data {
		for _, v := range g.Key.Fields {
			writeLine(line, v, false)
		}
		if !*comma {
			line.AppendSeparator(2)
		}
		for i, v := range g.Values {
			writeLine(line, v, true)
			if !*comma && i < len(g.Values)-1 {
				line.AppendSeparator(2)
			}
		}
		io.Copy(os.Stdout, line)
	}
}

func writeLine(line *linewriter.Writer, f Field, all bool) {
	switch f := f.(type) {
	case Int:
		line.AppendInt(f.Value, 8, linewriter.AlignRight)
		if all {
			line.AppendInt(f.Min, 8, linewriter.AlignRight)
			line.AppendInt(f.Max, 8, linewriter.AlignRight)
			line.AppendInt(int64(f.count), 8, linewriter.AlignRight)
			line.AppendFloat(f.Mean(), 12, 2, linewriter.AlignRight|linewriter.Float)
		}
	case Float:
		line.AppendFloat(f.Value, 8, 2, linewriter.AlignRight)
	case Date:
		line.AppendTime(f.Value, "2006-01-02", linewriter.AlignRight)
	case Duration:
		line.AppendDuration(f.Value, 8, linewriter.AlignRight)
		if all {
			line.AppendDuration(f.Min, 8, linewriter.AlignRight)
			line.AppendDuration(f.Max, 8, linewriter.AlignRight)
			line.AppendInt(int64(f.count), 8, linewriter.AlignRight)
			line.AppendDuration(f.Mean(), 12, linewriter.AlignRight|linewriter.Millisecond)
		}
	default:
	}
}

type Group struct {
	Key    Multifield
	Values []Field
}

type adder interface {
	Add(Field) Field
}

func (g *Group) Update(vs []Field) {
	if len(g.Values) == 0 {
		g.Values = append(g.Values, vs...)
		return
	}
	for i, f := range vs {
		v := g.Values[i]
		if a, ok := v.(adder); ok {
			g.Values[i] = a.Add(f)
		}
	}
}

func groupBy(r io.Reader, ps []ParseFunc, cols, vals []int) (map[string]*Group, error) {
	rs := csv.NewReader(r)
	data := make(map[string]*Group)

	for {
		records, err := rs.Read()
		if records == nil {
			break
		}
		if err != nil && err != io.EOF {
			return nil, err
		}
		var mf Multifield
		for _, ix := range cols {
			parse := ps[ix]
			f, err := parse(records[ix])
			if err != nil {
				return nil, err
			}
			mf.Fields = append(mf.Fields, f)
		}

		g, ok := data[mf.String()]
		if !ok {
			g = &Group{Key: mf}
			data[mf.String()] = g
		}
		fs := make([]Field, len(vals))
		for j, ix := range vals {
			parse := ps[ix]
			f, err := parse(records[ix])
			if err != nil {
				continue
			}
			fs[j] = f
		}
		g.Update(fs)
	}
	return data, nil
}

type ParseFunc func(string) (Field, error)

func sniffTypes(r io.Reader) ([]ParseFunc, error) {
	rs, err := csv.NewReader(r).Read()
	if err != nil && err != io.EOF {
		return nil, err
	}
	if len(rs) == 0 {
		return nil, io.EOF
	}
	ps := make([]ParseFunc, len(rs))
	fs := []func(string) string{tryNumber, tryDate, tryDatetime, tryDuration}
	for i := range rs {
		var (
			str   string
			parse ParseFunc
		)
		for _, f := range fs {
			str = f(rs[i])
			if str != "" {
				break
			}
		}
		if str == "" {
			str = "string"
		}
		switch str {
		default:
			parse = ParseString
		case "int":
			parse = ParseInt
		case "date":
			parse = ParseDate
		case "datetime":
			parse = ParseDatetime
		case "float":
			parse = ParseFloat
		case "duration":
			parse = ParseDuration
		}
		ps[i] = func(v string) (Field, error) {
			v = strings.TrimSpace(v)
			return parse(strings.TrimFunc(v, func(r rune) bool { return r == '"' }))
		}
	}
	if s, ok := r.(io.Seeker); ok {
		_, err = s.Seek(0, io.SeekStart)
	}
	return ps, err
}

func tryNumber(v string) string {
	if _, err := strconv.ParseInt(v, 0, 64); err == nil {
		return "int"
	}
	if _, err := strconv.ParseFloat(v, 64); err != nil {
		return ""
	}
	return "float"
}

func tryDate(v string) string {
	if _, err := time.Parse("2006-01-02", v); err != nil {
		return ""
	}
	return "date"
}

func tryDatetime(v string) string {
	fs := []string{
		time.RFC3339,
		"2006-01-02 15:04:05.000",
	}
	for _, f := range fs {
		if _, err := time.Parse(f, v); err == nil {
			return "datetime"
		}
	}
	return ""
}

func tryDuration(v string) string {
	if _, err := time.ParseDuration(v); err != nil {
		return ""
	}
	return "duration"
}
