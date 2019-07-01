package comma

import (
	"strings"

	"github.com/midbel/comma/eval"
)

type evaluator struct {
	es []eval.Evaluator
}

func Eval(sources []string) (eval.Evaluator, error) {
	es := make([]eval.Evaluator, 0, len(sources))
	for _, str := range sources {
		p, err := eval.Parse(str)
		if err != nil {
			return nil, err
		}
		e, err := p.ParseEvaluator()
		if err != nil {
			return nil, err
		}
		es = append(es, e)
	}
	return evaluator{es}, nil
}

func (e evaluator) Eval(row []string) ([]string, error) {
	var err error
	for _, e := range e.es {
		row, err = e.Eval(row)
		if err != nil {
			break
		}
	}
	return row, err
}

func (e evaluator) String() string {
	var b strings.Builder
	for _, e := range e.es {
		b.WriteRune('[')
		b.WriteString(e.String())
		b.WriteRune(']')
		b.WriteRune(';')
	}
	return b.String()
}
