package main

import (
	"fmt"
	"io"
	"os"
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
		Usage: "group [-table] [-file]",
		Short: "",
		Run:   nil,
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

func runGroup(cmd *cli.Command, args []string) error {
	return cmd.Flag.Parse(args)
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
