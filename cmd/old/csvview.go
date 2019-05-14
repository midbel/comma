package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/midbel/linewriter"
)

func main() {
	width := flag.Int("w", 16, "column widht")
	flag.Parse()

	var r io.Reader
	if flag.NArg() == 0 || flag.Arg(0) == "-" {
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
	rs := csv.NewReader(r)
	line := Line()
	for {
		records, err := rs.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		for _, i := range records {
			line.AppendString(i, *width, linewriter.AlignRight)
		}
		io.Copy(os.Stdout, line)
	}
}

func Line() *linewriter.Writer {
	options := []linewriter.Option{
		linewriter.WithPadding([]byte(" ")),
		linewriter.WithSeparator([]byte("|")),
	}
	return linewriter.NewWriter(4096, options...)
}
