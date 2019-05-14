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

type settings struct {
	Predicate string
	File      string
	Width     int
  Table     bool
  Separator Comma
}

func runFilter(cmd *cli.Command, args []string) error {
	return cmd.Flag.Parse(args)
}

func runDescribe(cmd *cli.Command, args []string) error {
	return cmd.Flag.Parse(args)
}

func runSelect(cmd *cli.Command, args []string) error {
	s := settings{
		Separator: Comma(','),
		Width:     DefaultWidth,
	}

	cmd.Flag.Var(&s.Separator, "separator", "separator")
	cmd.Flag.IntVar(&s.Width, "width", s.Width, "column width")
	cmd.Flag.StringVar(&s.File, "file", "", "input file")
	cmd.Flag.BoolVar(&s.Table, "table", false, "print data in table format")

	if err := cmd.Flag.Parse(args); err != nil {
		return err
	}
	var (
		r   *comma.Reader
		err error
	)
	sep := s.Separator.Rune()
	opt := comma.WithSelection(cmd.Flag.Arg(0))
	if s.File == "" || s.File == "-" {
		r, err = comma.NewReader(os.Stdin, sep, opt)
	} else {
		r, err = comma.Open(s.File, sep, opt)
	}
	if err != nil {
		return err
	}
	defer r.Close()

	dump := WriteRecords(os.Stdout, s.Width, s.Table)
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
