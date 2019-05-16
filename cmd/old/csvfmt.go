package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/midbel/linewriter"
	"github.com/midbel/sizefmt"
	"github.com/midbel/timefmt"
)

type Column struct {
	Index int
	Type  string
	In    string
	Out   string
}

func (c Column) Parse(vs []string) error {
	v := strings.TrimSpace(vs[c.Index])
	switch c.Type {
	case "date", "datetime":
		w := timefmt.Parse(v, c.In)
		vs[c.Index] = timefmt.Format(w, c.Out)
	case "size":
		n, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return err
		}
		vs[c.Index] = sizefmt.Format(n, sizefmt.IEC)
	default:
	}
	return nil
}

func main() {
	comma := flag.Bool("c", false, "csv format")
	sep := flag.String("separator", "", "csv separator")
	flag.Parse()
	var r io.Reader
	if file := flag.Arg(0); flag.NArg() == 0 || file == "-" {
		r = os.Stdin
	} else {
		f, err := os.Open(flag.Arg(0))
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		defer f.Close()

		r = f
	}

	cs := make([]Column, flag.NArg()-1)
	for i := 1; i < flag.NArg(); i++ {
		c, err := parseFormat(flag.Arg(i))
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		cs[i-1] = *c
	}

	line := Line(*comma)
	rs := csv.NewReader(r)
	if *sep != "" {
		rs.Comma = []rune(*sep)[0] //','
	}
	for {
		records, err := rs.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(3)
		}

		for i := 0; i < len(cs); i++ {
			if err := cs[i].Parse(records); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(4)
			}
		}
		for i := 0; i < len(records); i++ {
			line.AppendString(records[i], 12, linewriter.AlignRight)
		}
		io.Copy(os.Stdout, line)
	}
}

func Line(comma bool) *linewriter.Writer {
	var options []linewriter.Option
	if comma {
		options = append(options, linewriter.AsCSV(false))
	} else {
		options = []linewriter.Option{
			linewriter.WithPadding([]byte(" ")),
			linewriter.WithSeparator([]byte("|")),
		}
	}
	return linewriter.NewWriter(4096, options...)
}

func parseFormat(v string) (*Column, error) {
	var (
		c   Column
		err error
	)

	vs := strings.Split(v, ":")
	c.Index, err = strconv.Atoi(vs[0])
	if err != nil {
		return nil, err
	}
	c.Index--
	if c.Index < 0 {
		return nil, fmt.Errorf("invalid index: %s", v)
	}
	if len(vs) > 1 {
		c.Type = vs[1]
		switch c.Type {
		case "date", "datetime":
			c.In = vs[2]
			c.Out = vs[3]
		case "size":
		default:
		}
	}

	return &c, nil
}
