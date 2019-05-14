package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"
)

func main() {
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

	if err := sniffTypes(r); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}

func sniffTypes(r io.Reader) error {
	rs, err := csv.NewReader(r).Read()
	if err != nil && err != io.EOF {
		return err
	}
	if len(rs) == 0 {
		return nil
	}
	fs := []func(string) string{tryNumber, tryDate, tryDatetime, tryDuration}
	for i := range rs {
		var str string
		for _, f := range fs {
			str = f(rs[i])
			if str != "" {
				break
			}
		}
		if str == "" {
			str = "string"
		}
		fmt.Printf("%d: %s (%s)\n", i+1, str, rs[i])
	}
	return nil
}

func tryNumber(v string) string {
	if _, err := strconv.ParseInt(v, 0, 64); err == nil {
		return "int"
	}
	if _, err := strconv.ParseFloat(v, 64); err != nil {
		return ""
	}
	return "float"
}

func tryDate(v string) string {
	if _, err := time.Parse("2006-01-02", v); err != nil {
		return ""
	}
	return "date"
}

func tryDatetime(v string) string {
	fs := []string{
		time.RFC3339,
		"2006-01-02 15:04:05.000",
	}
	for _, f := range fs {
		if _, err := time.Parse(f, v); err == nil {
			return "datetime"
		}
	}
	return ""
}

func tryDuration(v string) string {
	if _, err := time.ParseDuration(v); err != nil {
		return ""
	}
	return "duration"
}
