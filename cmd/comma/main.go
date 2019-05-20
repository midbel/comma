package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/midbel/cli"
	"github.com/midbel/comma"
	"github.com/midbel/linewriter"
)

var commands = []*cli.Command{
	{
		Usage: "select [-separator] [-table] [-file] <selection>",
		Short: "",
		Run:   runSelect,
	},
	{
		Usage: "describe <file>",
		Short: "",
		Run:   runDescribe,
	},
	{
		Usage: "filter [-table] [-file] <expression>",
		Short: "",
		Run:   runFilter,
	},
	{
		Usage: "format [-table] [-file] <selection...>",
		Alias: []string{"fmt"},
		Short: "",
		Run:   runFormat,
	},
	{
		Usage: "group [-table] [-file] <selection> [<operation>...]",
		Short: "",
		Run:   runGroup,
	},
	{
		Usage: "transpose [-table] [-file]",
		Short: "",
		Run:   runTranspose,
	},
	{
		Usage: "cat [-table] [-width] [-column] <file...>",
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
		Usage: "show [-file] [-width] [-limit]",
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

	Prefix  string
	Datadir string
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

func runCat(cmd *cli.Command, args []string) error {
	return cmd.Flag.Parse(args)
}

func runSort(cmd *cli.Command, args []string) error {
	return cmd.Flag.Parse(args)
}

func runSplit(cmd *cli.Command, args []string) error {
	o := Options{
		Datadir:   os.TempDir(),
		Separator: Comma(','),
	}
	cmd.Flag.Var(&o.Separator, "separator", "separator")
	cmd.Flag.StringVar(&o.Datadir, "datadir", o.Datadir, "")
	cmd.Flag.StringVar(&o.Prefix, "prefix", o.Prefix, "")

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

	r, err := o.Open("", nil)
	if err != nil {
		return err
	}
	defer r.Close()

	dumps := make(map[string]*Dumper)
	for {
		switch row, err := r.Next(); err {
		case nil:
			ds, id := selectKeys(sel, row)
			if _, ok := dumps[id]; !ok {
				file := strings.Join(ds, "_") + ".csv"
				if o.Prefix != "" {
					file = o.Prefix + "-" + file
				}
				f, err := os.Create(filepath.Join(o.Datadir, strings.ToLower(file)))
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

func runJoin(cmd *cli.Command, args []string) error {
	return cmd.Flag.Parse(args)
}

func runCross(cmd *cli.Command, args []string) error {
	return cmd.Flag.Parse(args)
}

func runEval(cmd *cli.Command, args []string) error {
	return cmd.Flag.Parse(args)
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

	if err := cmd.Flag.Parse(args); err != nil {
		return err
	}
	r, err := o.Open("", nil)
	if err != nil {
		return err
	}
	defer r.Close()

	dump := WriteRecords(os.Stdout, o.Width, true)
	for i := 0; o.Limit <= 0 || i < o.Limit; i++ {
		switch row, err := r.Next(); err {
		case nil:
			dump(row)
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
			dump := WriteRecords(os.Stdout, o.Width, o.Table)
			for _, r := range rows {
				dump(r)
			}
			return nil
		default:
			return err
		}
	}
}

type Aggr struct {
	sel []comma.Selection
	comma.Aggr
}

func (a *Aggr) Update(row []string) error {
	for _, s := range a.sel {
		rs, err := s.Select(row)
		if err != nil {
			return err
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
		case "mean":
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

func runGroup(cmd *cli.Command, args []string) error {
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
	sel, err := comma.ParseSelection(cmd.Flag.Arg(0))
	if err != nil {
		return fmt.Errorf("selection (key): %s", err)
	}

	r, err := o.Open("", nil)
	if err != nil {
		return err
	}
	defer r.Close()

	var rows []Row
	ops := cmd.Flag.Args()
	for {
		switch row, err := r.Next(); err {
		case nil:
			keys, id := selectKeys(sel, row)
			ix := sort.Search(len(rows), func(i int) bool { return rows[i].Hash <= id })
			if ix < len(rows) && rows[ix].Hash == id {
				rows[ix].Count++
			} else {
				r := Row{
					Hash:  id,
					Keys:  keys,
					Count: 1,
				}
				as, err := parseAggr(ops[1:])
				if err != nil {
					return err
				}
				r.Data = append(r.Data, as...)
				if ix >= len(rows) {
					rows = append(rows, r)
				} else {
					rows = append(rows[:ix], append([]Row{r}, rows[ix:]...)...)
				}
			}
			if err := rows[ix].Update(row); err != nil {
				return err
			}
		case io.EOF:
			line := Line(o.Table)
			for i := range rows {
				r := rows[i]
				for _, v := range r.Keys {
					line.AppendString(v, o.Width, linewriter.AlignRight)
				}
				line.AppendUint(r.Count, o.Width, linewriter.AlignRight)
				for _, d := range r.Data {
					for _, r := range d.Result() {
						line.AppendFloat(r, o.Width, 2, linewriter.AlignRight|linewriter.Float)
					}
				}
				io.Copy(os.Stdout, line)
			}
			return nil
		default:
			return err
		}
	}
}

type Row struct {
	Count uint64
	Keys  []string
	Hash  string

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

func selectKeys(sel []comma.Selection, row []string) ([]string, string) {
	var ds []string
	for _, s := range sel {
		vs, err := s.Select(row)
		if err != nil {
			return nil, ""
		}
		ds = append(ds, vs...)
	}
	return ds, strings.Join(ds, "/")
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

	if err := cmd.Flag.Parse(args); err != nil {
		return err
	}
	r, err := o.Open("", cmd.Flag.Args())
	if err != nil {
		return err
	}
	defer r.Close()

	dump := WriteRecords(os.Stdout, o.Width, o.Table)
	for {
		switch row, err := r.Next(); err {
		case nil:
			dump(row)
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

	dump := WriteRecords(os.Stdout, o.Width, o.Table)
	for {
		switch row, err := r.Filter(match); err {
		case nil:
			dump(row)
		case io.EOF:
			return nil
		default:
			return err
		}
	}
}

func runDescribe(cmd *cli.Command, args []string) error {
	return cmd.Flag.Parse(args)
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

	if err := cmd.Flag.Parse(args); err != nil {
		return err
	}
	r, err := o.Open(cmd.Flag.Arg(0), nil)
	if err != nil {
		return err
	}
	defer r.Close()

	dump := WriteRecords(os.Stdout, o.Width, o.Table)
	for {
		switch row, err := r.Next(); err {
		case nil:
			dump(row)
		case io.EOF:
			return nil
		default:
			return err
		}
	}
}

const DefaultWidth = 16

func WriteRecords(w io.Writer, width int, table bool) func([]string) error {
	if width <= 0 {
		width = DefaultWidth
	}
	line := Line(table)
	return func(records []string) error {
		for i := 0; i < len(records); i++ {
			line.AppendString(records[i], width, linewriter.AlignRight)
		}
		_, err := io.Copy(w, line)
		return err
	}
}

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
