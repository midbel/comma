package comma

import (
	"strconv"
	"time"
	// "github.com/midbel/timefmt"
)

type ParseFunc func(string) (Cell, error)

func Sniff(v string) (Cell, error) {
	fs := []ParseFunc{
		ParseInt,
		ParseFloat,
		ParseDuration,
		// ParseDate,
		// ParseDatetime,
	}
	for _, parse := range fs {
		c, err := parse(v)
		if err == nil {
			return c, nil
		}
	}
	return ParseString(v)
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
