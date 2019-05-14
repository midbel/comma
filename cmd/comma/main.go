package main

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"os"

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
}

const helpText = "comma helps you to explore your data stored in csv files"

func main() {
	err := cli.Run(commands, cli.Usage("tmcat", helpText, commands), nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}

type settings struct {
	Predicate string
	File      string
	Separator Comma
	Table     bool
	Width     int

	Fields int
}

func (s *settings) Open(r io.Reader) *csv.Reader {
	rb := bufio.NewReader(r)
	if bs, err := rb.Peek(4096); err == nil {
		rs := csv.NewReader(bytes.NewReader(bs))
		if _, err := rs.Read(); err == nil {
			s.Fields = rs.FieldsPerRecord
		}
	}
	rs := csv.NewReader(rb)
	rs.Comma = s.Separator.Rune()
	rs.TrimLeadingSpace = true
	if s.Fields > 0 {
		rs.FieldsPerRecord = s.Fields
	}

	return rs
}

func runDescribe(cmd *cli.Command, args []string) error {
	s := settings{
		Separator: Comma(symbol),
	}
	cmd.Flag.Var(&s.Separator, "separator", "separator")
	cmd.Flag.StringVar(&s.File, "file", "", "input file")
	if err := cmd.Flag.Parse(args); err != nil {
		return err
	}

	var r io.Reader
	if s.File == "" || s.File == "-" {
		r = os.Stdin
	} else {
		f, err := os.Open(s.File)
		if err != nil {
			return err
		}
		defer f.Close()
		r = f
	}
	rs := s.Open(r)

	row, err := rs.Read()
	if err != nil {
		return err
	}
	line := Line(true)
	for i := range row {
		c, err := comma.Sniff(row[i])
		if err != nil {
			return err
		}
		line.AppendInt(int64(i+1), 4, linewriter.AlignLeft)
		line.AppendString(c.Kind().String(), 12, linewriter.AlignLeft)
		line.AppendString(c.String(), 12, linewriter.AlignLeft)

		io.Copy(os.Stdout, line)
	}
	return nil
}

func runSelect(cmd *cli.Command, args []string) error {
	s := settings{
		Separator: Comma(symbol),
		Width:     DefaultWidth,
	}

	cmd.Flag.Var(&s.Separator, "separator", "separator")
	cmd.Flag.IntVar(&s.Width, "width", s.Width, "column width")
	cmd.Flag.StringVar(&s.File, "file", "", "input file")
	cmd.Flag.BoolVar(&s.Table, "table", false, "print data in table format")

	if err := cmd.Flag.Parse(args); err != nil {
		return err
	}
	var r io.Reader
	if s.File == "" || s.File == "-" {
		r = os.Stdin
	} else {
		f, err := os.Open(s.File)
		if err != nil {
			return err
		}
		defer f.Close()
		r = f
	}
	rs := s.Open(r)

	cols, err := ParseSelection(cmd.Flag.Arg(0), s.Fields)
	if err != nil {
		return fmt.Errorf("selection: %s", err)
	}

	dump := WriteRecords(os.Stdout, s.Width, s.Table)
	for {
		switch row, err := rs.Read(); err {
		case nil:
			vs := make([]string, len(cols))
			for i := 0; i < len(cols); i++ {
				if ix := cols[i]; ix < 0 || ix >= len(row) {
					return fmt.Errorf("index out of range")
				} else {
					vs[i] = row[ix]
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
