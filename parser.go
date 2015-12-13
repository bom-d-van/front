package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"runtime/debug"
)

type Parser struct {
	lexer *Lexer
	look  Token
	top   *Env
	used  int
	err   error
}

func NewParser(l *Lexer) Parser {
	var p Parser
	p.lexer = l
	p.move()
	return p
}

func (p *Parser) move() {
	p.look, p.err = p.lexer.Scan()
	// log.Printf("%#v %s", p.look, string(p.look.Tag()))
	if p.err != nil && p.err != io.EOF {
		p.error(p.err)
	}
}

func (p *Parser) error(err error) {
	// panic(err)
	log.Println(err)
	debug.PrintStack()
	os.Exit(1)
}

func (p *Parser) match(tag Tag) bool {
	// if option.pmatch {
	// log.Printf("--> %#v %s\n", p.look, tag)
	// }
	if p.look.Tag() == tag {
		p.move()
		return true
	}
	p.error(fmt.Errorf("line %d: syntax error: want %s but got %s", lexerLine, tag, p.look))
	return false
}

func (p *Parser) program() {
	s := p.block()
	// if option.ps {
	// }
	// pretty.Println(s)
	begin, after := newLabel(), newLabel()
	emitLabel(begin)
	s.gen(begin, after)
	emitLabel(after)
}

func (p *Parser) block() Node {
	p.match('{')
	savedEnv := p.top
	p.top = NewEnv(p.top)
	p.decls()
	s := p.stmts()
	p.match('}')
	p.top = savedEnv
	return s
}

func (p *Parser) decls() {
	for p.look.Tag() == BASIC {
		typ := p.typ()
		tok := p.look
		p.match(ID)
		p.match(';')
		id := Id{Expr: NewExpr(tok, typ), offset: p.used}
		p.top.put(tok, id)
		p.used += typ.Width()
	}
}

func (p *Parser) typ() Typer {
	typ := p.look.(Type)
	p.match(BASIC)
	if p.look.Tag() != '[' {
		return typ
	}
	return p.dims(typ)
}

func (p *Parser) dims(typ Typer) Typer {
	p.match('[')
	tok := p.look
	p.match(NUM)
	p.match(']')
	if p.look.Tag() == '[' {
		typ = p.dims(typ)
	}
	// return Array{size: tok.(Num).value, elem: typ}
	return NewArray(tok.(Num).value, typ)
}

func (p *Parser) stmts() Node {
	if p.look.Tag() == '}' {
		return nil
	}
	return NewSeq(p.stmt(), p.stmts())
}

func (p *Parser) stmt() Node {
	var x Node
	var s1, s2 Node
	var savedStmt Node

	switch p.look.Tag() {
	case ';':
		p.move()
		return nil
	case IF:
		p.match(IF)
		p.match('(')
		x = p.bool()
		p.match(')')
		s1 = p.stmt()
		// log.Printf("--> %+v\n", p.look.Tag().Tag())
		// log.Printf("--> %+v\n", p.look.Tag().Tag() == ELSE)
		if p.look.Tag().Tag() != ELSE {
			return If{expr: x, stmt: s1, Stmt: Stmt{Line: p.NewLine()}}
		}
		p.match(ELSE)
		s2 = p.stmt()
		return Else{expr: x, stmt1: s1, stmt2: s2, Stmt: Stmt{Line: p.NewLine()}}
	case WHILE:
		var while While
		savedStmt = StmtEnclosing
		StmtEnclosing = &while
		p.match(WHILE)
		p.match('(')
		x = p.bool()
		p.match(')')
		// p.match(';')
		s1 = p.stmt()
		while.init(x, s1)
		StmtEnclosing = savedStmt
		return &while
	case DO:
		var do Do
		savedStmt = StmtEnclosing
		StmtEnclosing = do
		p.match(DO)
		s1 = p.stmt()
		p.match(WHILE)
		p.match('(')
		x = p.bool()
		p.match(')')
		p.match(';')
		do.init(s1, x)
		StmtEnclosing = savedStmt
		return do
	case BREAK:
		p.match(BREAK)
		p.match(';')
		b := NewBreak()
		// if StmtEnclosing == nil {
		// 	b.error("unenclosed break")
		// }
		// b.stmt = StmtEnclosing
		return b
	case '{':
		return p.block()
	default:
		return p.assign()
	}
}

func (p *Parser) NewLine() Line { return Line(lexerLine) }

func (p *Parser) assign() Node {
	var stmt Node
	t := p.look
	// log.Printf("--> %T %[1]#v\n", p.look)
	p.match(ID)
	id, ok := p.top.get(t)
	if !ok {
		p.error(fmt.Errorf("line %d: %s undeclared", lexerLine, t.(Word).lexeme))
	}
	if p.look.Tag() == '=' { // S -> id = E ;
		p.move()
		stmt = NewSet(id, p.bool())
	} else { // S -> L = E ;
		x := p.offset(id)
		p.match('=')
		stmt = NewSetElem(x, p.bool())
	}
	p.match(';')
	return stmt
}

func (p *Parser) bool() Node {
	x := p.join()
	for p.look.Tag() == OR {
		tok := p.look
		p.move()
		// x = Or{Expr: Expr{op: tok, typ: typ, Line: Line(p.lexer.line)}}
		x = NewOrNode(tok, x, p.join())
	}
	return x
}

func (p *Parser) join() Node {
	x := p.equality()
	for p.look.Tag() == AND {
		tok := p.look
		p.move()
		x = NewAndNode(tok, x, p.equality())
	}
	return x
}

func (p *Parser) equality() Node {
	x := p.rel()
	for p.look.Tag() == EQ || p.look.Tag() == NE {
		tok := p.look
		p.move()
		x = NewRel(tok, x, p.rel())
	}
	return x
}

func (p *Parser) rel() Node {
	x := p.expr()
	switch p.look.Tag() {
	case '<', LE, GE, '>':
		tok := p.look
		p.move()
		r := NewRel(tok, x, p.expr())
		return r
	}
	return x
}

func (p *Parser) expr() Node {
	x := p.term()
	for p.look.Tag() == '+' || p.look.Tag() == '-' {
		tok := p.look
		p.move()
		x = NewArith(tok, x, p.term())
	}
	return x
}

func (p *Parser) term() Node {
	x := p.unary()
	for p.look.Tag() == '*' || p.look.Tag() == '/' {
		tok := p.look
		p.move()
		x = NewArith(tok, x, p.unary())
	}
	return x
}

func (p *Parser) unary() Node {
	if p.look.Tag() == '-' {
		p.move()
		return NewUnary(Minus, p.unary())
	} else if p.look.Tag() == '!' {
		tok := p.look
		p.move()
		x := p.unary()
		return NewNot(tok, x)
	}
	return p.factor()
}

func (p *Parser) factor() Node {
	var x Node
	switch p.look.Tag() {
	case '(':
		p.move()
		x = p.bool()
		p.match(')')
	case NUM:
		x = NewConstant(p.look, Int)
		p.move()
	case REAL:
		x = NewConstant(p.look, Float)
		p.move()
	case TRUE:
		x = ConstantTrue
		p.move()
	case FALSE:
		x = ConstantFalse
		p.move()
	case ID:
		s := p.look.String()
		id, ok := p.top.get(p.look)
		if !ok {
			p.error(fmt.Errorf("%s undeclared", s))
		}
		p.move()
		if p.look.Tag() != '[' {
			return id
		}
		return p.offset(id)
	default:
		p.error(fmt.Errorf("syntax error"))
	}

	return x
}

// I -> [E] | [E] I
func (p *Parser) offset(a Id) Access {
	var i, w, t1, t2, loc Node
	typ := a.typ
	p.match('[')
	i = p.bool()
	p.match(']')
	typ = typ.(Array).elem
	w = NewConstantInt(typ.Width())
	t1 = NewArith(Tag('*'), i, w)
	loc = t1
	for p.look == Tag('[') { // multi-dimensional I -> [ E ] I
		p.match('[')
		i = p.bool()
		p.match(']')
		typ = typ.(Array).elem
		w = NewConstantInt(typ.Width())
		t1 = NewArith(Tag('*'), i, w)
		t2 = NewArith(Tag('+'), loc, t1)
		loc = t2
	}
	return NewAccess(a, loc, typ)
}
