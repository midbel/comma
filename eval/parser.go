package eval

import (
	"fmt"
	"strconv"
)

type Evaluator interface {
	fmt.Stringer
	Eval([]string) ([]string, error)
}

type Expression interface {
	fmt.Stringer
	Value([]string) (Value, error)
}

const (
	bindLowest    = iota
	bindAssign    // =
	bindCondition // ?:
	bindLogical   // &&, ||
	bindRelation  // ==, !=, <, >, <=, >=
	bindSum       // +, -
	bindProduct   // *, /
	bindPower     // ^
	bindPrefix    // !, -
	bindGroup     // ()
	bindCall      // ()
)

var bindings = map[rune]int{
	assign:   bindAssign,
	question: bindCondition,
	colon:    bindCondition,
	plus:     bindSum,
	minus:    bindSum,
	multiply: bindProduct,
	divide:   bindProduct,
	modulo:   bindProduct,
	caret:    bindPower,
	// lparen:   bindGroup,
	lparen:   bindCall,
	or:       bindLogical,
	and:      bindLogical,
	equal:    bindRelation,
	notequal: bindRelation,
	lesser:   bindRelation,
	greater:  bindRelation,
	lesseq:   bindRelation,
	greateq:  bindRelation,
}

type Parser struct {
	lex *lexer

	curr Token
	peek Token
	err  error

	infix  map[rune]func(Expression) (Expression, error)
	prefix map[rune]func() (Expression, error)
}

func Parse(str string) (*Parser, error) {
	var p Parser

	p.lex = lex(str)
	p.infix = map[rune]func(Expression) (Expression, error){
		plus:     p.parseInfix,
		minus:    p.parseInfix,
		divide:   p.parseInfix,
		multiply: p.parseInfix,
		modulo:   p.parseInfix,
		or:       p.parseInfix,
		and:      p.parseInfix,
		equal:    p.parseInfix,
		notequal: p.parseInfix,
		lesser:   p.parseInfix,
		lesseq:   p.parseInfix,
		greater:  p.parseInfix,
		greateq:  p.parseInfix,
		caret:    p.parseInfix,
		assign:   p.parseAssignInfix,
		lparen:   p.parseCall,
		question: p.parseCondition,
	}
	p.prefix = map[rune]func() (Expression, error){
		minus:    p.parsePrefix,
		bang:     p.parsePrefix,
		index:    p.parseIndex,
		number:   p.parseValue,
		text:     p.parseValue,
		env:      p.parseValue,
		variable: p.parseValue,
		lparen:   p.parseGroup,
		assign:   p.parseAssignPrefix,
	}

	p.nextToken()
	p.nextToken()

	return &p, p.err
}

func (p *Parser) ParseExpression() (Expression, error) {
	return p.parseExpression(bindLowest)
}

func (p *Parser) ParseEvaluator() (Evaluator, error) {
	if !(p.curr.Type == assign || p.peek.Type == assign) {
		return nil, fmt.Errorf("%s: invalid syntax", p.lex.input)
	}
	e, err := p.parseExpression(bindLowest)
	if err != nil {
		return nil, err
	}
	if x, ok := e.(Evaluator); !ok {
		return nil, fmt.Errorf("%s: invalid syntax", p.lex.input)
	} else {
		return x, nil
	}
}

func (p *Parser) parseExpression(bp int) (Expression, error) {
	// fmt.Println("-> parseExpression:", p.curr.String())
	if p.err != nil {
		return nil, p.err
	}
	prefix, ok := p.prefix[p.curr.Type]
	if !ok {
		return nil, fmt.Errorf("no prefix function registered for %s", p.curr)
	}
	left, err := prefix()
	if err != nil {
		return nil, err
	}
	for p.peek.Type != eof && bp < p.peekPower() {
		infix, ok := p.infix[p.peek.Type]
		if !ok {
			return nil, fmt.Errorf("no infix function registered for '%c'", p.peek.Type)
		}

		p.nextToken()
		left, err = infix(left)
		if err != nil {
			return nil, err
		}
	}
	return left, nil
}

func (p *Parser) parseCall(left Expression) (Expression, error) {
	if p.peek.Type == rparen {
		return left, nil
	}
	p.nextToken()

	fn, ok := left.(Function)
	if !ok {
		return nil, fmt.Errorf("parser error: expected <function>, got %T", left)
	}
	e, err := p.parseExpression(bindLowest)
	if err != nil {
		return nil, err
	}
	fn.params = append(fn.params, e)
	for p.peek.Type == comma {
		p.nextToken()
		p.nextToken()
		e, err = p.parseExpression(bindLowest)
		if err != nil {
			return nil, err
		}
		fn.params = append(fn.params, e)
	}
	if p.peek.Type != rparen {
		return nil, fmt.Errorf("parser error: expected ), got %s", p.peek)
	} else {
		p.nextToken()
	}
	return fn, nil
}

func (p *Parser) parseCondition(left Expression) (Expression, error) {
	// fmt.Println("-> parseCondition:", p.curr.String())
	p.nextToken()
	cdt := Ternary{cond: left}

	left, err := p.parseExpression(bindCondition)
	if err != nil {
		return nil, err
	}
	cdt.left = left

	if p.peek.Type != colon {
		return nil, fmt.Errorf("parser error: expected :, got %s", p.peek)
	} else {
		p.nextToken()
		p.nextToken()
	}

	right, err := p.parseExpression(bindLowest)
	if err != nil {
		return nil, err
	}
	cdt.right = right

	return cdt, nil
}

func (p *Parser) parsePrefix() (Expression, error) {
	// fmt.Println("-> parsePrefix:", p.curr.String())
	var (
		exp Expression
		err error
	)
	switch op := p.curr.Type; op {
	default:
		err = fmt.Errorf("parser error: can not parse %s", p.curr)
	case minus, bang:
		p.nextToken()
		if x, e := p.parseExpression(bindPrefix); e != nil {
			err = e
		} else {
			exp = Prefix{operator: op, right: x}
		}
	}
	return exp, err
}

func (p *Parser) parseValue() (Expression, error) {
	// fmt.Println("-> parseValue:", p.curr.String())
	var (
		exp Expression
		err error
	)
	switch p.curr.Type {
	default:
		err = fmt.Errorf("parser error: can not parse %s", p.curr)
	case variable:
		if lit := p.curr.Literal; lit == "true" || lit == "false" {
			if b, e := strconv.ParseBool(p.curr.Literal); e != nil {
				err = e
			} else {
				exp = Bool(b)
			}
		} else {
			exp = Function{name: lit}
		}
	case text:
		exp = Text(p.curr.Literal)
	case number:
		if f, e := strconv.ParseFloat(p.curr.Literal, 64); e != nil {
			err = e
		} else {
			exp = Literal(f)
		}
	case env:
		exp = Internal(p.curr.Literal)
	}
	if p.peek.Type == cast {
		p.nextToken()
		switch exp.(type) {
		case Bool, Text, Literal:
			exp = castTo(exp, p.curr.Literal)
		default:
			return nil, fmt.Errorf("parser error: %T can not be casted!", exp)
		}
	}
	return exp, err
}

func (p *Parser) parseIndex() (Expression, error) {
	// fmt.Println("-> parseIndex:", p.curr.String())
	i, err := strconv.ParseInt(p.curr.Literal, 10, 64)
	if err != nil {
		return nil, err
	}
	exp := Identifier{Index: int(i)}
	if p.peek.Type == cast {
		p.nextToken()
		exp.Cast = p.curr.Literal
	}
	return exp, nil
}

func (p *Parser) parseGroup() (Expression, error) {
	// fmt.Println("-> parseGroup:", p.curr.String())
	p.nextToken()
	exp, err := p.parseExpression(bindLowest)
	if err != nil {
		return nil, err
	}
	if p.peek.Type != rparen {
		return nil, fmt.Errorf("parser error: expected ), got %s", p.peek)
	} else {
		p.nextToken()
	}
	return exp, nil
}

func (p *Parser) parseAssignPrefix() (Expression, error) {
	// fmt.Println("-> parseAssignPrefix", p.curr.String())
	return p.parseAssignInfix(nil)
}

func (p *Parser) parseAssignInfix(left Expression) (Expression, error) {
	var bp int
	if left == nil {
		bp = bindAssign
	} else {
		// fmt.Println("-> parseAssignInfix:", p.curr.String())
		switch left.(type) {
		case Prefix:
		case Literal:
		case Identifier:
		default:
			return nil, fmt.Errorf("parser error: only index or literal can be used in assignment, got %T", left)
		}
		bp = p.currPower()
	}
	p.nextToken()
	exp := Assign{left: left}

	right, err := p.parseExpression(bp)
	if err == nil {
		exp.right = right
	}
	return exp, err
}

func (p *Parser) parseInfix(left Expression) (Expression, error) {
	// fmt.Println("-> parseInfix:", p.curr.String())
	exp := Infix{
		left:     left,
		operator: p.curr.Type,
	}
	bp := p.currPower()
	p.nextToken()

	right, err := p.parseExpression(bp)
	if err == nil {
		exp.right = right
	}
	return exp, err
}

func (p *Parser) currPower() int {
	return bindingPower(p.curr)
}

func (p *Parser) peekPower() int {
	return bindingPower(p.peek)
}

func bindingPower(t Token) int {
	b, ok := bindings[t.Type]
	if !ok {
		b = bindLowest
	}
	return b
}

func (p *Parser) nextToken() {
	p.curr = p.peek
	p.peek = p.lex.Next()

	if p.curr.Type == invalid && p.err == nil {
		p.err = fmt.Errorf("invalid token found")
	}
}
