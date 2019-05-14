package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

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

func main() {
	var cols Columns
	flag.Var(&cols, "c", "columns")
	flag.Parse()

	var w io.Writer
	if flag.NArg() >= 2 {
		f, err := os.Create(flag.Arg(1))
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		defer f.Close()
		w = f
	} else {
		w = os.Stdout
	}

	r, err := os.Open(flag.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer r.Close()

	if err := takeColumns(r, w, cols.Ints()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(3)
	}
}

func takeColumns(r io.Reader, w io.Writer, cols []int) error {
	rs := csv.NewReader(r)
	ws := csv.NewWriter(w)

	data := make([]string, len(cols))
	for {
		records, err := rs.Read()
		switch err {
		case nil:
		case io.EOF:
			return nil
		default:
			return err
		}
		if len(records) == 0 {
			break
		}
		for i, ix := range cols {
			data[i] = records[ix]
		}
		if err := ws.Write(data); err != nil {
			return err
		}
	}
	ws.Flush()
	return ws.Error()
}
