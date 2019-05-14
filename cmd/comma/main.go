package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"

	"github.com/midbel/cli"
	"github.com/midbel/linewriter"
)

var commands = []*cli.Command{
	{
		Usage: "slice [-separator] [-table] [-file] <columns>",
		Alias: []string{"split", "select"},
		Short: "",
		Run:   runSlice,
	},
	{
		Usage: "describe <file>",
		Short: "",
		Run:   runDescribe,
	},
}

const helpText = "comma helps you to explore your data stored in csv files"

func main() {
	err := cli.Run(commands, cli.Usage("tmcat", helpText, commands), nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}

func runDescribe(cmd *cli.Command, args []string) error {
	return cmd.Flag.Parse(args)
}

func runSlice(cmd *cli.Command, args []string) error {
	c := struct {
		Predicate string
		File      string
		Separator Comma
		Table     bool
		Width     int
	}{
		Separator: Comma(','),
		Width:     DefaultWidth,
	}

	cmd.Flag.Var(&c.Separator, "separator", "separator")
	cmd.Flag.IntVar(&c.Width, "width", c.Width, "column width")
	cmd.Flag.StringVar(&c.File, "file", "", "input file")
	cmd.Flag.BoolVar(&c.Table, "table", false, "print data in table format")
	if err := cmd.Flag.Parse(args); err != nil {
		return err
	}
	cols, err := ParseSelection(cmd.Flag.Arg(0))
	if err != nil {
		return fmt.Errorf("selection: %s", err)
	}
	var r io.Reader
	if c.File == "" || c.File == "-" {
		r = os.Stdin
	} else {
		f, err := os.Open(c.File)
		if err != nil {
			return err
		}
		defer f.Close()
		r = f
	}

	rs := csv.NewReader(r)
	rs.Comma = c.Separator.Rune()
	rs.TrimLeadingSpace = true

	dump := WriteRecords(os.Stdout, c.Width, c.Table)
	for {
		switch row, err := rs.Read(); err {
		case nil:
			var vs []string
			if n := len(cols); n == 0 {
				vs = row
			} else {
				vs = make([]string, n)
				for i := 0; i < n; i++ {
					if ix := cols[i]; ix >= len(row) {
						return fmt.Errorf("index out of range")
					} else {
						vs[i] = row[ix]
					}
				}
			}
			dump(vs)
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
