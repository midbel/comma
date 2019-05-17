package main

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/midbel/cli"
	"github.com/midbel/comma"
	"github.com/midbel/linewriter"
)

var commands = []*cli.Command{
	{
		Usage: "select [-separator] [-table] [-file] <selections>",
		Short: "",
		Run:   runSelect,
	},
	{
		Usage: "describe <file>",
		Short: "",
		Run:   runDescribe,
	},
	{
		Usage: "filter [-table] [-file] <criteria>",
		Short: "",
		Run:   runFilter,
	},
	{
		Usage: "format [-table] [-file] <selections>",
		Alias: []string{"fmt"},
		Short: "",
		Run:   runFormat,
	},
	{
		Usage: "group [-table] [-file] <key> [<operation>...]",
		Short: "",
		Run:   runGroup,
	},
	{
		Usage: "transpose [-table] [-file]",
		Short: "",
		Run:   runTranspose,
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
	Predicate string
	File      string
	Width     int
	Table     bool
	Separator Comma
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
		return fmt.Errorf("selection: %s", err)
	}

	r, err := o.Open("", nil)
	if err != nil {
		return err
	}
	defer r.Close()

	var rows []Row
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
				if ix >= len(rows) {
					rows = append(rows, r)
				} else {
					rows = append(rows[:ix], append([]Row{r}, rows[ix:]...)...)
				}
			}
		case io.EOF:
			line := Line(o.Table)
			for i := range rows {
				r := rows[i]
				for _, v := range r.Keys {
					line.AppendString(v, o.Width, linewriter.AlignRight)
				}
				line.AppendUint(r.Count, o.Width, linewriter.AlignRight)
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
