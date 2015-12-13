package main

import (
	"fmt"
	"runtime/debug"
)

type Node interface {
	Pos() int
	reduce() Node
	String() string

	gen(b, a int)
	genNode() Node

	jumping(t, f int) // ?

	typer() Typer
}

var labelCount int

func newLabel() int {
	labelCount++
	// if labelCount == 10 {
	// 	debug.PrintStack()
	// }
	return labelCount
}

func emitLabel(i int) {
	if option.el {
		debug.PrintStack()
	}
	fmt.Printf("L%d:", i)
}

func emit(str string, args ...interface{}) {
	fmt.Println("\t" + fmt.Sprintf(str, args...))
}

func emitjumps(test string, t, f int) {
	if t != 0 && f != 0 {
		emit("if %s goto L%d", test, t)
		emit("goto L%d", f)
	} else if t != 0 {
		emit("if %s goto L%d", test, t)
	} else if f != 0 {
		// log.Printf("--> %T\n", test)
		// debug.PrintStack()
		emit("iffalse %s goto L%d", test, f)
	}
}

type Line int

func (l Line) Pos() int { return int(l) }
func (l Line) error(msg string) {
	panic(fmt.Errorf("line %d: %s", l, msg))
}

type Stmt struct {
	Line
	after     int
	enclosing *Stmt
	typ       Typer
}

var StmtEnclosing Node

// var StmtNull = &Stmt{}

func (s Stmt) gen(b, a int)     {}
func (s Stmt) String() string   { return "" }
func (s Stmt) reduce() Node     { return s }
func (s Stmt) genNode() Node    { return s }
func (s Stmt) jumping(t, f int) {}
func (s Stmt) typer() Typer     { return s.typ }
func (s Stmt) After() int       { return s.after }

type Seq struct {
	Stmt
	stmt1, stmt2 Node
}

func NewSeq(s1, s2 Node) Seq {
	s := Seq{stmt1: s1, stmt2: s2}
	s.Line = Line(lexerLine)
	return s
}

func (s Seq) gen(b, a int) {
	if s.stmt1 == nil {
		s.stmt2.gen(b, a)
	} else if s.stmt2 == nil {
		s.stmt1.gen(b, a)
	} else {
		label := newLabel()
		s.stmt1.gen(b, label)
		emitLabel(label)
		s.stmt2.gen(label, a)
	}
}

type Expr struct {
	Line
	op  Token
	typ Typer
}

func NewExpr(op Token, typ Typer) Expr {
	e := Expr{op: op, typ: typ, Line: Line(lexerLine)}
	e.Line = Line(lexerLine)
	return e
}

func (e Expr) genNode() Node    { return e }
func (e Expr) gen(int, int)     {}
func (e Expr) reduce() Node     { return e }
func (e Expr) jumping(t, f int) { emitjumps(e.op.String(), t, f) }
func (e Expr) Tag() Tag         { return e.op.Tag() }
func (e Expr) String() string   { return e.op.String() }
func (e Expr) typer() Typer     { return e.typ }

type If struct {
	Stmt
	expr Node
	stmt Node
}

func (i If) gen(b, a int) {
	label := newLabel()
	i.expr.jumping(0, a)
	emitLabel(label)
	if i.stmt != nil {
		i.stmt.gen(label, a)
	}
}

type Else struct {
	Stmt
	expr         Node
	stmt1, stmt2 Node
}

func (e Else) gen(b, a int) {
	label1 := newLabel()
	label2 := newLabel()
	e.expr.jumping(0, label2)
	emitLabel(label1)
	if e.stmt1 != nil {
		e.stmt1.gen(label1, a)
	}
	emit("goto L%d", a)
	emitLabel(label2)
	e.stmt2.gen(label2, a)
}

type While struct {
	Stmt
	expr Node
	stmt Node
}

func (w While) genNode() Node  { return &w }
func (w While) reduce() Node   { return &w }
func (w While) String() string { return "while" }
func (w *While) gen(b, a int) {
	w.after = a
	w.expr.jumping(0, a)
	label := newLabel()
	emitLabel(label)
	w.stmt.gen(label, b)
	emit("goto L%d", b)
}

func (d *While) init(expr Node, stmt Node) {
	d.expr = expr
	d.stmt = stmt
	if expr.typer().Lexeme() != Bool.Lexeme() {
		d.error("boolean required in while")
	}
}

type Do struct {
	Stmt
	expr Node
	stmt Node
}

func (d Do) gen(b, a int) {
	d.after = a
	label := newLabel()
	d.stmt.gen(b, label)
	emitLabel(label)
	d.expr.jumping(b, 0)
}

func (d *Do) init(stmt Node, expr Node) {
	d.stmt = stmt
	d.expr = expr
	if expr.typer().Lexeme() != Bool.Lexeme() {
		d.error("boolean required in do")
	}
}

type Break struct {
	Stmt
	stmt Node
}

func NewBreak() Break {
	var b Break
	if StmtEnclosing == nil {
		b.error("unenclosed break")
	}
	b.Line = Line(lexerLine)
	b.stmt = StmtEnclosing
	return b
}

func (br Break) gen(b, a int) {
	emit("goto L%d", br.stmt.(interface {
		After() int
	}).After())
}

type Id struct {
	Expr
	offset int
}

// func (i Id) Tag() Tag {
// 	return i.typ
// }

type Set struct {
	Stmt
	id   Id
	expr Node
}

func NewSet(id Id, expr Node) Set {
	var s Set
	s.Line = Line(lexerLine)
	s.id = id
	s.expr = expr
	// log.Println(lexerLine)
	// log.Printf("--> %+v\n", id.typer())
	// log.Printf("--> %T\n", expr)
	s.typ = s.check(id.typer(), expr.typer())
	if s.typ == nil {
		s.error("type error")
	}
	return s
}

func (s *Set) check(p1, p2 Typer) Typer {
	if IsNumbericType(p1) && IsNumbericType(p2) {
		return p2
	} else if p1.Lexeme() == Bool.Lexeme() && p2.Lexeme() == Bool.Lexeme() {
		return p2
	}
	return nil
}

func (s Set) gen(b, a int) {
	emit("%s = %s", s.id, s.expr.genNode())
}

type Op struct{ Expr }

func (o Op) genNode() Node { return o }
func (o Op) reduce() Node {
	debug.PrintStack()
	x := o.genNode()
	t := NewTemp(o.typ)
	emit("%s = %s", t, x)
	return t
}

type Temp struct {
	Expr
	number int
}

var tempCount int

func NewTemp(p Typer) Temp {
	var t Temp
	t.Expr = NewExpr(temp, p)
	tempCount++
	t.number = tempCount
	return t
}

func (t Temp) String() string { return fmt.Sprintf("t%d", t.number) }

type Access struct {
	Op
	array Id
	index Node
}

func NewAccess(a Id, i Node, p Typer) Access {
	var as Access
	as.Expr = NewExpr(NewWord("[]", INDEX), p)
	as.array = a
	as.index = i
	return as
}

func (a Access) genNode() Node    { return NewAccess(a.array, a.index.reduce(), a.typ) }
func (a Access) jumping(t, f int) { emitjumps(a.reduce().String(), t, f) }
func (a Access) String() string   { return fmt.Sprintf("%s [ %s ]", a.array, a.index) }
func (a Access) reduce() Node {
	x := a.genNode()
	t := NewTemp(a.typ)
	emit("%s = %s", t, x)
	return t
}

type SetElem struct {
	Stmt
	array Id
	index Node
	expr  Node
}

func NewSetElem(x Access, y Node) SetElem {
	var se SetElem
	se.array = x.array
	se.index = x.index
	se.expr = y
	se.Line = Line(lexerLine)
	if se.check(x.typer(), y.typer()) == nil {
		se.error("type error")
	}
	return se
}

func (s *SetElem) check(p1, p2 Typer) Typer {
	_, ok1 := p1.(Array)
	_, ok2 := p2.(Array)
	if ok1 || ok2 {
		return nil
	} else if p1 == p2 {
		return p2
	} else if IsNumbericType(p1) && IsNumbericType(p2) {
		return p2
	}
	return nil
}

func (s SetElem) gen(b, a int) {
	emit("%s [ %s ] = %s", s.array, s.index.reduce(), s.expr.reduce())
}

type Logical struct {
	Expr
	expr1, expr2 Node
}

func NewLogical(tok Token, x1, x2 Node) Logical {
	var l Logical
	l.Expr = NewExpr(tok, nil)
	l.expr1, l.expr2 = x1, x2
	l.typ = l.check(x1.typer(), x2.typer())
	if l.typ == nil {
		l.error("type error")
	}
	return l
}

func (l *Logical) check(p1, p2 Typer) Typer {
	if p1.(Type) == Bool && p2.(Type) == Bool {
		return Bool
	}
	return nil
}

func (l Logical) genNode() Node {
	f := newLabel()
	a := newLabel()
	temp := NewTemp(l.typ)
	l.jumping(0, f)
	emit("%s = true", temp)
	emit("goto L%d", a)
	emitLabel(f)
	emit("%s = false", temp)
	emitLabel(a)
	return temp
}

func (l Logical) String() string {
	return fmt.Sprintf("%s %s %s", l.expr1, l.op, l.expr2)
}

type Rel struct {
	Logical
}

func (r Rel) genNode() Node {
	f := newLabel()
	a := newLabel()
	temp := NewTemp(r.typ)
	r.jumping(0, f)
	emit("%s = true", temp)
	emit("goto L%d", a)
	emitLabel(f)
	emit("%s = false", temp)
	emitLabel(a)
	return temp
}

func NewRel(tok Token, x1, x2 Node) Rel {
	var r Rel
	r.Expr = NewExpr(tok, nil)
	r.expr1 = x1
	r.expr2 = x2
	r.typ = r.check(x1.typer(), x2.typer())
	if r.typ == nil {
		r.error("type error")
	}
	return r
}

func (l *Rel) check(p1, p2 Typer) Typer {
	_, ok1 := p1.(Array)
	_, ok2 := p2.(Array)
	if ok1 || ok2 {
		return nil
	} else if p1.Lexeme() == p2.Lexeme() {
		return Bool
	}
	return nil
}

func (r Rel) jumping(t, f int) {
	emitjumps(fmt.Sprintf("%s %s %s", r.expr1.reduce(), r.op, r.expr2.reduce()), t, f)
}

type OrNode struct {
	Logical
}

func NewOrNode(tok Token, x1, x2 Node) OrNode {
	return OrNode{Logical: NewLogical(tok, x1, x2)}
}

func (o OrNode) jumping(t, f int) {
	label := t
	if label == 0 {
		label = newLabel()
	}
	o.expr1.jumping(label, 0)
	o.expr2.jumping(t, f)
	if t == 0 {
		emitLabel(label)
	}
}

func (o OrNode) genNode() Node {
	f := newLabel()
	a := newLabel()
	temp := NewTemp(o.typ)
	o.jumping(0, f)
	emit("%s = true", temp)
	emit("goto L%d", a)
	emitLabel(f)
	emit("%s = false", temp)
	emitLabel(a)
	return temp
}

type AndNode struct {
	Logical
}

func NewAndNode(tok Token, x1, x2 Node) AndNode {
	return AndNode{Logical: NewLogical(tok, x1, x2)}
}

func (a AndNode) jumping(t, f int) {
	label := f
	if f == 0 {
		label = newLabel()
	}
	// pretty.Println(a.expr1)
	a.expr1.jumping(0, label)
	a.expr2.jumping(t, f)
	if f == 0 {
		emitLabel(label)
	}
}

func (an AndNode) genNode() Node {
	f := newLabel()
	a := newLabel()
	temp := NewTemp(an.typ)
	an.jumping(0, f)
	emit("%s = true", temp)
	emit("goto L%d", a)
	emitLabel(f)
	emit("%s = false", temp)
	emitLabel(a)
	return temp
}

type Arith struct {
	Op
	expr1, expr2 Node
}

func NewArith(tok Token, x1, x2 Node) Arith {
	var a Arith
	a.Expr = NewExpr(tok, nil)
	a.expr1 = x1
	a.expr2 = x2
	a.typ = MaxType(x1.typer(), x2.typer())
	if a.typ == nil {
		a.error("type error")
	}
	return a
}

func (a Arith) genNode() Node {
	// log.Println(a.op, a.expr1.reduce(), a.expr2.reduce())
	// log.Printf("--> %T\n", a.expr2)
	return NewArith(a.op, a.expr1.reduce(), a.expr2.reduce())
}

func (a Arith) reduce() Node {
	// debug.PrintStack()
	x := a.genNode()
	t := NewTemp(a.typ)
	emit("%s = %s", t, x)
	return t
}

func (a Arith) String() string {
	// log.Println(a.expr1, a.op, a.expr2)
	return fmt.Sprintf("%s %s %s", a.expr1, a.op, a.expr2)
}

type Unary struct {
	Op
	expr Node
}

func NewUnary(tok Token, x Node) Unary {
	var u Unary
	u.Expr = NewExpr(tok, nil)
	u.expr = x
	u.typ = MaxType(Int, x.typer())
	if u.typ == nil {
		u.error("type error")
	}
	return u
}

func (u Unary) genNode() Node {
	return NewUnary(u.Op, u.expr.reduce())
}

func (u Unary) String() string {
	return fmt.Sprintf("%s %s", u.Op, u.expr)
}

type Not struct{ Logical }

func NewNot(tok Token, x2 Node) Not {
	var n Not
	n.Logical = NewLogical(tok, x2, x2)
	return n
}

func (n Not) jumping(t, f int) {
	n.expr2.jumping(f, t)
}

func (n Not) String() string {
	return fmt.Sprintf("%s %s", n.op, n.expr2)
}

func (n Not) genNode() Node {
	f := newLabel()
	a := newLabel()
	temp := NewTemp(n.typ)
	n.jumping(0, f)
	emit("%s = true", temp)
	emit("goto L%d", a)
	emitLabel(f)
	emit("%s = false", temp)
	emitLabel(a)
	return temp
}

type Constant struct{ Expr }

var ConstantTrue = NewConstant(True, Bool)
var ConstantFalse = NewConstant(False, Bool)

func NewConstant(tok Token, p Type) Constant {
	var c Constant
	c.Expr = NewExpr(tok, p)
	return c
}

func NewConstantInt(i int) Constant {
	return NewConstant(Num{value: i}, Int)
}

func (c Constant) jumping(t, f int) {
	if c == ConstantTrue && t != 0 {
		emit("goto L%d", t)
	} else if c == ConstantFalse && f != 0 {
		emit("goto L%d", f)
	}
}

// func (w Seq) reduce() Node      { return w }
// func (w If) reduce() Node       { return w }
// func (w Else) reduce() Node     { return w }
// func (w Do) reduce() Node       { return w }
// func (w Break) reduce() Node    { return w }
// func (w Id) reduce() Node       { return w }
// func (w Set) reduce() Node      { return w }
// func (w Temp) reduce() Node     { return w }
// func (w SetElem) reduce() Node  { return w }
// func (w Logical) reduce() Node  { return w }
// func (w Rel) reduce() Node      { return w }
// func (w OrNode) reduce() Node   { return w }
// func (w Arith) reduce() Node { return w }
// func (w Unary) reduce() Node { return w }
// func (w Not) reduce() Node      { return w }
// func (w Constant) reduce() Node { return w }
