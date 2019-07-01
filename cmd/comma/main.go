package main

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/midbel/cli"
	"github.com/midbel/comma"
	"github.com/midbel/linewriter"
)

var commands = []*cli.Command{
	{
		Usage: "select [-separator] [-tag] [-table] [-file] <selection>",
		Short: "",
		Run:   runSelect,
	},
	{
		Usage: "describe <file>",
		Short: "",
		Run:   runDescribe,
	},
	{
		Usage: "filter [-table] [-tag] [-file] <expression>",
		Short: "",
		Run:   runFilter,
	},
	{
		Usage: "format [-table] [-tag] [-file] <selection...>",
		Alias: []string{"fmt"},
		Short: "",
		Run:   runFormat,
	},
	{
		Usage: "group [-table] [-tag] [-file] <selection> [<operation>...]",
		Short: "",
		Run:   runGroup,
	},
	{
		Usage: "frequency [-table] [-tag] [-file] <selection>",
		Alias: []string{"freq"},
		Short: "",
		Run:   runFrequency,
	},
	{
		Usage: "transpose [-table] [-file]",
		Short: "",
		Run:   runTranspose,
	},
	{
		Usage: "cat [-append] [-table] [-width] [-separator] <file...>",
		Short: "",
		Run:   runCat,
	},
	{
		Usage: "sort [-table] [-width] [-file] <selection>",
		Short: "",
		Run:   runSort,
	},
	{
		Usage: "split [-datadir] [-prefix] [-file] <selection> <expression>",
		Short: "",
		Run:   runSplit,
	},
	{
		Usage: "eval [-table] [-width] [-file] <expression...>",
		Short: "",
		Run:   runEval,
	},
	{
		Usage: "show [-file] [-tag] [-width] [-limit] [<headers...>]",
		Alias: []string{"table"},
		Short: "",
		Run:   runTable,
	},
}

const helpText = `{{.Name}} helps you to explore your data stored in csv files

Usage:

  {{.Name}} command [options] <arguments>

Available commands:

{{range .Commands}}{{if .Runnable}}{{printf "  %-12s %s" .String .Short}}{{if .Alias}} (alias: {{ join .Alias ", "}}){{end}}{{end}}
{{end}}
Use {{.Name}} [command] -h for more information about its usage.
`

func main() {
	defer func() {
		if err := recover(); err != nil {
			fmt.Fprintf(os.Stderr, "unexpected error: %s\n", err)
			os.Exit(2)
		}
	}()
	err := cli.Run(commands, cli.Usage("comma", helpText, commands), nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}

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

type Options struct {
	File      string
	Separator Comma

	Limit int
	Width int
	Table bool

	Append  bool
	Prefix  string
	Datadir string
	Tag     string
}

func (o Options) Open(cols string, specs []string) (*comma.Reader, error) {
	opts := []comma.Option{
		comma.WithSeparator(o.Separator.Rune()),
		comma.WithSelection(cols),
		comma.WithFormatters(specs),
	}
	var (
		r   *comma.Reader
		err error
	)
	if o.File == "" || o.File == "-" {
		r, err = comma.NewReader(os.Stdin, opts...)
	} else {
		r, err = comma.Open(o.File, opts...)
	}
	return r, err
}

var ErrImplemented = errors.New("not yet implemented")

func runSort(cmd *cli.Command, args []string) error {
	return ErrImplemented
}

func runJoin(cmd *cli.Command, args []string) error {
	return ErrImplemented
}

func runCross(cmd *cli.Command, args []string) error {
	return ErrImplemented
}

func runDescribe(cmd *cli.Command, args []string) error {
	return ErrImplemented
}

func runEval(cmd *cli.Command, args []string) error {
	o := Options{
		Separator: Comma(','),
		Width:     DefaultWidth,
	}
	cmd.Flag.Var(&o.Separator, "separator", "separator")
	cmd.Flag.IntVar(&o.Limit, "limit", 0, "show N first rows")
	cmd.Flag.IntVar(&o.Width, "width", o.Width, "column width")
	cmd.Flag.StringVar(&o.File, "file", "", "input file")
	cmd.Flag.StringVar(&o.Tag, "tag", "", "tag")
	if err := cmd.Flag.Parse(args); err != nil {
		return err
	}
	e, err := comma.Eval(cmd.Flag.Args())
	if err != nil {
		return err
	}

	r, err := o.Open("", nil)
	if err != nil {
		return err
	}
	defer r.Close()

	dump := Dump(os.Stdout, o.Width, o.Table)
	for {
		switch row, err := r.Next(); err {
		case nil:
			row, err := e.Eval(row)
			if err != nil {
				return err
			}
			if o.Tag != "" {
				row = append([]string{o.Tag}, row...)
			}
			dump.Dump(row)
		case io.EOF:
			return nil
		default:
			return err
		}
	}
	return ErrImplemented
}

func runCat(cmd *cli.Command, args []string) error {
	o := Options{
		Separator: Comma(','),
	}
	cmd.Flag.Var(&o.Separator, "separator", "separator")
	cmd.Flag.IntVar(&o.Width, "width", o.Width, "column width")
	cmd.Flag.BoolVar(&o.Append, "append", false, "append")
	cmd.Flag.BoolVar(&o.Table, "table", false, "print data in table format")
	if err := cmd.Flag.Parse(args); err != nil {
		return err
	}
	var cat func([]string, Options) error
	if o.Append {
		cat = appendRows
	} else {
		cat = appendColumns
	}
	return cat(cmd.Flag.Args(), o)
}

func appendRows(files []string, o Options) error {
	var rs []io.Reader
	for _, f := range files {
		r, err := os.Open(f)
		if err != nil {
			return err
		}
		defer r.Close()
		rs = append(rs, r)
	}
	dump := Dump(os.Stdout, o.Width, o.Table)

	rc := csv.NewReader(io.MultiReader(rs...))
	rc.Comma = o.Separator.Rune()
	for {
		switch row, err := rc.Read(); err {
		case nil:
			dump.Dump(row)
		case io.EOF:
			return nil
		default:
			return err
		}
	}
}

func appendColumns(files []string, o Options) error {
	rs := make([]*csv.Reader, len(files))
	for i, f := range files {
		r, err := os.Open(f)
		if err != nil {
			return err
		}
		defer r.Close()

		rs[i] = csv.NewReader(r)
		rs[i].Comma = o.Separator.Rune()
	}

	cols := make([][]string, len(rs))

	var done int
	dump := Dump(os.Stdout, o.Width, o.Table)
	for {
		var row []string
		for i := 0; i < len(rs); i++ {
			if rs == nil {
				row = append(row, cols[i]...)
				continue
			}
			switch vs, err := rs[i].Read(); err {
			case nil:
				row = append(row, vs...)
				if len(cols[i]) == 0 {
					cols[i] = make([]string, len(vs))
				}
			case io.EOF:
				done++
				rs[i] = nil
				if done == len(rs) {
					return nil
				}
			default:
				return err
			}
		}
		dump.Dump(row)
	}
	return nil
}

func runSplit(cmd *cli.Command, args []string) error {
	o := Options{
		Datadir:   os.TempDir(),
		Separator: Comma(','),
	}
	cmd.Flag.Var(&o.Separator, "separator", "separator")
	cmd.Flag.BoolVar(&o.Append, "append", false, "append")
	cmd.Flag.StringVar(&o.Datadir, "datadir", o.Datadir, "")
	cmd.Flag.StringVar(&o.Prefix, "prefix", o.Prefix, "")
	cmd.Flag.StringVar(&o.File, "file", o.File, "")

	if err := cmd.Flag.Parse(args); err != nil {
		return err
	}

	if err := os.MkdirAll(o.Datadir, 0755); err != nil {
		return err
	}

	sel, err := comma.ParseSelection(cmd.Flag.Arg(0))
	if err != nil {
		return fmt.Errorf("selection (key): %s", err)
	}
	var filter *comma.Filter
	if f, err := comma.ParseFilter(cmd.Flag.Arg(1)); err == nil {
		filter = f
	} else {
		return fmt.Errorf("filter: %s", err)
	}

	r, err := o.Open("", nil)
	if err != nil {
		return err
	}
	defer r.Close()

	dumps := make(map[string]*Dumper)
	for {
		switch row, err := r.Filter(filter); err {
		case nil:
			ds, id := selectKeys(sel, row)
			if _, ok := dumps[id]; !ok {
				file := strings.Join(ds, "_") + ".csv"
				if o.Prefix != "" {
					file = o.Prefix + "-" + file
				}
				mode := os.O_CREATE | os.O_WRONLY
				if o.Append {
					mode = mode | os.O_APPEND
				}
				f, err := os.OpenFile(filepath.Join(o.Datadir, strings.ToLower(file)), mode, 0644)
				if err != nil {
					return err
				}
				defer f.Close()

				dumps[id] = Dump(f, o.Width, false)
			}
			if err := dumps[id].Dump(row); err != nil {
				return err
			}
		case io.EOF:
			return nil
		default:
			return err
		}
	}
}

func runTable(cmd *cli.Command, args []string) error {
	o := Options{
		Separator: Comma(','),
		Width:     DefaultWidth,
	}
	cmd.Flag.Var(&o.Separator, "separator", "separator")
	cmd.Flag.IntVar(&o.Limit, "limit", 0, "show N first rows")
	cmd.Flag.IntVar(&o.Width, "width", o.Width, "column width")
	cmd.Flag.StringVar(&o.File, "file", "", "input file")
	cmd.Flag.StringVar(&o.Tag, "tag", "", "tag")

	if err := cmd.Flag.Parse(args); err != nil {
		return err
	}
	r, err := o.Open("", nil)
	if err != nil {
		return err
	}
	defer r.Close()

	dump := Dump(os.Stdout, o.Width, true)
	headers := cmd.Flag.Args()
	if len(headers) > 0 {
		if o.Tag != "" {
			headers = append([]string{"tag"}, headers...)
		}
		dump.Dump(headers)
	}
	for i := 0; o.Limit <= 0 || i < o.Limit; i++ {
		switch row, err := r.Next(); err {
		case nil:
			if o.Tag != "" {
				row = append([]string{o.Tag}, row...)
			}
			if z := len(headers); z > 0 && len(row) > z {
				row = row[:len(headers)]
			}
			dump.Dump(row)
		case io.EOF:
			return nil
		default:
			return err
		}
	}
	return nil
}

func runTranspose(cmd *cli.Command, args []string) error {
	o := Options{
		Separator: Comma(','),
		Width:     DefaultWidth,
	}
	cmd.Flag.Var(&o.Separator, "separator", "separator")
	cmd.Flag.IntVar(&o.Width, "width", o.Width, "column width")
	cmd.Flag.StringVar(&o.File, "file", "", "input file")
	cmd.Flag.BoolVar(&o.Table, "table", false, "print data in table format")

	if err := cmd.Flag.Parse(args); err != nil {
		return err
	}
	r, err := o.Open("", nil)
	if err != nil {
		return err
	}
	defer r.Close()

	var rows [][]string
	for {
		switch row, err := r.Next(); err {
		case nil:
			if n := len(rows); n == 0 {
				rows = make([][]string, len(row))
			}
			for i, r := range row {
				rows[i] = append(rows[i], r)
			}
		case io.EOF:
			dump := Dump(os.Stdout, o.Width, o.Table)
			for _, r := range rows {
				if err := dump.Dump(r); err != nil {
					return err
				}
			}
			return nil
		default:
			return err
		}
	}
}

func runFrequency(cmd *cli.Command, args []string) error {
	o := Options{
		Separator: Comma(','),
		Width:     DefaultWidth,
	}
	cmd.Flag.Var(&o.Separator, "separator", "separator")
	cmd.Flag.IntVar(&o.Width, "width", o.Width, "column width")
	cmd.Flag.StringVar(&o.File, "file", "", "input file")
	cmd.Flag.BoolVar(&o.Append, "count", false, "append count column per group")
	cmd.Flag.BoolVar(&o.Table, "table", false, "print data in table format")
	cmd.Flag.StringVar(&o.Tag, "tag", "", "tag")
	reverse := cmd.Flag.Bool("reverse", false, "reverse")

	if err := cmd.Flag.Parse(args); err != nil {
		return err
	}
	sel, err := comma.ParseSelection(cmd.Flag.Arg(0))
	if err != nil {
		return fmt.Errorf("selection (key): %s", err)
	}

	r, err := o.Open("", nil)
	if err != nil {
		return err
	}
	defer r.Close()

	cumul := Aggr{
		sel:    sel,
		single: true,
		Aggr:   comma.Count(),
	}
	data := Tree{
		Sel:     sel,
		Reverse: *reverse,
	}
	for {
		switch row, err := r.Next(); err {
		case nil:
			if err := data.Upsert(row); err != nil {
				return err
			}
			if err := cumul.Update(row); err != nil {
				return err
			}
		case io.EOF:
			line, results := Line(o.Table), cumul.Result()
			sums := make([]float64, len(results))
			percents := make([]float64, len(results))
			data.Traverse(func(r *Row) {
				if o.Tag != "" {
					line.AppendString(o.Tag, o.Width, linewriter.AlignRight)
				}
				for _, v := range r.Keys {
					line.AppendString(v, o.Width, linewriter.AlignRight)
				}
				for _, d := range r.Data {
					for i, r := range d.Result() {
						sums[i] += r
						percent := r / results[i]
						percents[i] += percent
						line.AppendFloat(r, o.Width, 2, linewriter.AlignRight|linewriter.Float)
						line.AppendFloat(sums[i], o.Width, 2, linewriter.AlignRight|linewriter.Float)
						line.AppendPercent(percent, o.Width, 2, linewriter.AlignRight)
						line.AppendPercent(percents[i], o.Width, 2, linewriter.AlignRight)
					}
				}
				io.Copy(os.Stdout, line)
			})
			return nil
		default:
			return err
		}
	}
}

func runGroup(cmd *cli.Command, args []string) error {
	// defer profile.Start(profile.CPUProfile).Stop()
	o := Options{
		Separator: Comma(','),
		Width:     DefaultWidth,
	}
	cmd.Flag.Var(&o.Separator, "separator", "separator")
	cmd.Flag.IntVar(&o.Width, "width", o.Width, "column width")
	cmd.Flag.StringVar(&o.File, "file", "", "input file")
	cmd.Flag.BoolVar(&o.Table, "table", false, "print data in table format")
	cmd.Flag.StringVar(&o.Tag, "tag", "", "tag")
	reverse := cmd.Flag.Bool("reverse", false, "reverse")

	if err := cmd.Flag.Parse(args); err != nil {
		return err
	}
	sel, err := comma.ParseSelection(cmd.Flag.Arg(0))
	if err != nil {
		return fmt.Errorf("selection (key): %s", err)
	}

	r, err := o.Open("", nil)
	if err != nil {
		return err
	}
	defer r.Close()

	ops := cmd.Flag.Args()
	data := Tree{
		Ops:     ops[1:],
		Sel:     sel,
		Reverse: *reverse,
	}
	for {
		switch row, err := r.Next(); err {
		case nil:
			if err := data.Upsert(row); err != nil {
				return err
			}
		case io.EOF:
			line := Line(o.Table)
			data.Traverse(func(r *Row) {
				if o.Tag != "" {
					line.AppendString(o.Tag, o.Width, linewriter.AlignRight)
				}
				for _, v := range r.Keys {
					line.AppendString(v, o.Width, linewriter.AlignRight)
				}
				for _, d := range r.Data {
					for _, r := range d.Result() {
						line.AppendFloat(r, o.Width, 2, linewriter.AlignRight|linewriter.Float)
					}
				}
				io.Copy(os.Stdout, line)
			})
			return nil
		default:
			return err
		}
	}
}

func runFormat(cmd *cli.Command, args []string) error {
	o := Options{
		Separator: Comma(','),
		Width:     DefaultWidth,
	}
	cmd.Flag.Var(&o.Separator, "separator", "separator")
	cmd.Flag.IntVar(&o.Width, "width", o.Width, "column width")
	cmd.Flag.StringVar(&o.File, "file", "", "input file")
	cmd.Flag.BoolVar(&o.Table, "table", false, "print data in table format")
	cmd.Flag.StringVar(&o.Tag, "tag", "", "tag")

	if err := cmd.Flag.Parse(args); err != nil {
		return err
	}
	r, err := o.Open("", cmd.Flag.Args())
	if err != nil {
		return err
	}
	defer r.Close()

	dump := Dump(os.Stdout, o.Width, o.Table)
	for {
		switch row, err := r.Next(); err {
		case nil:
			if o.Tag != "" {
				row = append([]string{o.Tag}, row...)
			}
			dump.Dump(row)
		case io.EOF:
			return nil
		default:
			return err
		}
	}

	return nil
}

func runFilter(cmd *cli.Command, args []string) error {
	o := Options{
		Separator: Comma(','),
		Width:     DefaultWidth,
	}

	cmd.Flag.Var(&o.Separator, "separator", "separator")
	cmd.Flag.IntVar(&o.Width, "width", o.Width, "column width")
	cmd.Flag.StringVar(&o.File, "file", "", "input file")
	cmd.Flag.BoolVar(&o.Table, "table", false, "print data in table format")
	cmd.Flag.StringVar(&o.Tag, "tag", "", "tag")

	if err := cmd.Flag.Parse(args); err != nil {
		return err
	}
	match, err := comma.ParseFilter(cmd.Flag.Arg(0))
	if err != nil {
		return fmt.Errorf("filter: %s", err)
	}
	r, err := o.Open("", nil)
	if err != nil {
		return err
	}
	defer r.Close()

	dump := Dump(os.Stdout, o.Width, o.Table)
	for {
		switch row, err := r.Filter(match); err {
		case nil:
			if o.Tag != "" {
				row = append([]string{o.Tag}, row...)
			}
			dump.Dump(row)
		case io.EOF:
			return nil
		default:
			return err
		}
	}
}

func runSelect(cmd *cli.Command, args []string) error {
	o := Options{
		Separator: Comma(','),
		Width:     DefaultWidth,
	}
	cmd.Flag.Var(&o.Separator, "separator", "separator")
	cmd.Flag.IntVar(&o.Width, "width", o.Width, "column width")
	cmd.Flag.StringVar(&o.File, "file", "", "input file")
	cmd.Flag.BoolVar(&o.Table, "table", false, "print data in table format")
	cmd.Flag.StringVar(&o.Tag, "tag", "", "tag")

	if err := cmd.Flag.Parse(args); err != nil {
		return err
	}
	r, err := o.Open(cmd.Flag.Arg(0), nil)
	if err != nil {
		return err
	}
	defer r.Close()

	dump := Dump(os.Stdout, o.Width, o.Table)
	for {
		switch row, err := r.Next(); err {
		case nil:
			if o.Tag != "" {
				row = append([]string{o.Tag}, row...)
			}
			dump.Dump(row)
		case io.EOF:
			return nil
		default:
			return err
		}
	}
}

const DefaultWidth = 10

func Line(table bool) *linewriter.Writer {
	var options []linewriter.Option
	if table {
		options = []linewriter.Option{
			linewriter.WithSeparator([]byte("|")),
			linewriter.WithPadding([]byte(" ")),
		}
	} else {
		options = append(options, linewriter.AsCSV(false))
	}
	return linewriter.NewWriter(4096, options...)
}

type Dumper struct {
	Width int

	line  *linewriter.Writer
	inner io.Writer

	io.Closer
}

func Dump(w io.Writer, width int, table bool) *Dumper {
	if width <= 0 {
		width = DefaultWidth
	}
	d := Dumper{
		line:  Line(table),
		inner: w,
		Width: width,
	}
	if c, ok := w.(io.Closer); ok {
		d.Closer = c
	}
	return &d
}

func (d *Dumper) Close() error {
	var err error
	if d.Closer != nil {
		err = d.Closer.Close()
	}
	return err
}

func (d *Dumper) Dump(row []string) error {
	for i := 0; i < len(row); i++ {
		d.line.AppendString(row[i], d.Width, linewriter.AlignRight)
	}
	n, err := io.Copy(d.inner, d.line)
	if err == io.EOF && n > 0 {
		err = nil
	}
	return err
}

type Aggr struct {
	sel    []comma.Selection
	single bool
	comma.Aggr
}

func (a *Aggr) Update(row []string) error {
	for _, s := range a.sel {
		rs, err := s.Select(row)
		if err != nil {
			return err
		}
		if a.single && len(rs) > 0 {
			rs = rs[:1]
		}
		if err := a.Aggr.Aggr(rs); err != nil {
			return err
		}
	}
	return nil
}

func parseAggr(vs []string) ([]Aggr, error) {
	if mod := len(vs) % 2; mod != 0 {
		return nil, fmt.Errorf("no enough argument")
	}
	var as []Aggr
	for i := 0; i < len(vs); i += 2 {
		op, sel := vs[i], vs[i+1]
		s, err := comma.ParseSelection(sel)
		if err != nil {
			return nil, err
		}
		var a comma.Aggr
		switch strings.ToLower(op) {
		case "mean", "avg":
			a = comma.Mean()
		case "sum", "cum":
			a = comma.Sum()
		case "min":
			a = comma.Min()
		case "max":
			a = comma.Max()
		case "count":
			a = comma.Count()
		default:
			return nil, fmt.Errorf("unknown operation %s", op)
		}
		as = append(as, Aggr{sel: s, Aggr: a})
	}
	return as, nil
}

type Row struct {
	Keys []string
	Hash string

	Data []Aggr
}

func (r *Row) Update(row []string) error {
	for _, d := range r.Data {
		if err := d.Update(row); err != nil {
			return err
		}
	}
	return nil
}

type Tree struct {
	root *Node
	Ops  []string
	Sel  []comma.Selection

	Reverse bool
}

func (t *Tree) Find(ks []string) *Row {
	if t.root == nil {
		return nil
	}
	return t.root.Find(ks)
}

func (t *Tree) Upsert(vs []string) error {
	ks := t.selectKeys(vs)
	if len(ks) == 0 {
		return nil
	}
	var r *Row
	if t.root == nil {
		t.root = nodeFromKeys(ks)
		r = t.root.Row
	} else {
		r = t.root.Upsert(ks, t.Reverse)
	}
	if len(r.Data) == 0 {
		var (
			as  []Aggr
			err error
		)
		if len(t.Ops) == 0 {
			a := Aggr{
				sel:    t.Sel,
				single: true,
				Aggr:   comma.Count(),
			}
			as = []Aggr{a}
		} else {
			as, err = parseAggr(t.Ops)
		}
		if err != nil {
			return err
		}
		r.Data = append(r.Data, as...)
	}
	return r.Update(vs)
}

func (t *Tree) selectKeys(row []string) []string {
	ds := make([]string, 0, len(t.Sel)+1)
	for i := 0; i < len(t.Sel); i++ {
		vs, err := t.Sel[i].Select(row)
		if err != nil {
			return nil
		}
		ds = append(ds, vs...)
	}
	return ds
}

func (t *Tree) Traverse(fn func(r *Row)) {
	if t.root == nil {
		return
	} else {
		t.root.Traverse(fn)
	}
}

type Node struct {
	*Row
	Left  *Node
	Right *Node
}

func (n *Node) Find(ks []string) *Row {
	switch cmp := compareKeys(n.Keys, ks); cmp {
	default:
		return n.Row
	case -1:
		if n.IsLeaf() || n.Left == nil {
			return nil
		}
		return n.Left.Find(ks)
	case 1:
		if n.IsLeaf() || n.Right == nil {
			return nil
		}
		return n.Right.Find(ks)
	}
}

func (n *Node) Upsert(ks []string, reverse bool) *Row {
	var r *Row
	cmp := compareKeys(n.Keys, ks)
	if reverse {
		cmp = -cmp
	}
	switch cmp {
	default:
		r = n.Row
	case -1:
		if n.Left == nil {
			n.Left = nodeFromKeys(ks)
			r = n.Left.Row
		} else {
			r = n.Left.Upsert(ks, reverse)
		}
	case 1:
		if n.Right == nil {
			n.Right = nodeFromKeys(ks)
			r = n.Right.Row
		} else {
			r = n.Right.Upsert(ks, reverse)
		}
	}
	return r
}

func (n *Node) Traverse(fn func(*Row)) {
	if !n.IsLeaf() && n.Left != nil {
		n.Left.Traverse(fn)
	}
	fn(n.Row)
	if !n.IsLeaf() && n.Right != nil {
		n.Right.Traverse(fn)
	}
}

func (n *Node) IsLeaf() bool {
	return n.Left == nil && n.Right == nil
}

func compareKeys(k1, k2 []string) int {
	for i := 0; i < len(k1); i++ {
		c := strings.Compare(k1[i], k2[i])
		if c != 0 {
			return c
		}
	}
	return 0
}

func nodeFromKeys(ks []string) *Node {
	r := Row{Keys: ks}
	return &Node{Row: &r}
}

func selectKeys(sel []comma.Selection, row []string) ([]string, string) {
	ds := make([]string, 0, len(sel)+1)
	for _, s := range sel {
		vs, err := s.Select(row)
		if err != nil {
			return nil, ""
		}
		ds = append(ds, vs...)
	}
	return ds, strings.Join(ds, "/")
}
