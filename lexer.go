package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
)

type Tag int

const (
	AND   Tag = 256
	BASIC Tag = 257
	BREAK Tag = 258
	DO    Tag = 259
	ELSE  Tag = 260

	EQ    Tag = 261
	FALSE Tag = 262
	GE    Tag = 263
	ID    Tag = 264
	IF    Tag = 265

	INDEX Tag = 266
	LE    Tag = 267
	MINUS Tag = 268
	NE    Tag = 269
	NUM   Tag = 270

	OR    Tag = 271
	REAL  Tag = 272
	TEMP  Tag = 273
	TRUE  Tag = 274
	WHILE Tag = 275
)

func (t Tag) Tag() Tag {
	return t
}

func (t Tag) String() string {
	switch t {
	case AND:
		return "and"
	case BASIC:
		return "basic"
	case BREAK:
		return "break"
	case DO:
		return "do"
	case ELSE:
		return "else"
	case EQ:
		return "eq"
	case FALSE:
		return "false"
	case GE:
		return "ge"
	case ID:
		return "id"
	case IF:
		return "if"
	case INDEX:
		return "index"
	case LE:
		return "le"
	case MINUS:
		return "minus"
	case NE:
		return "ne"
	case NUM:
		return "num"
	case OR:
		return "or"
	case REAL:
		return "real"
	case TEMP:
		return "temp"
	case TRUE:
		return "true"
	case WHILE:
		return "while"
		// case INT:
		// 	return "int"
		// case FLOAT:
		// 	return "float"
		// case CHAR:
		// 	return "char"
		// case BOOL:
		// 	return "bool"
	}

	return string(t)
}

type Token interface {
	Tag() Tag
	String() string
}

type Lexer struct {
	_peek byte
	// line   int
	tokens map[string]Token
	// *scanner.Scanner
	r   io.Reader
	err error
}

var lexerLine = 1

func NewLexer(r io.Reader) *Lexer {
	l := Lexer{tokens: map[string]Token{}, r: r}
	l.reserve(&Word{lexeme: "if", tag: IF})
	l.reserve(&Word{lexeme: "else", tag: ELSE})
	l.reserve(&Word{lexeme: "while", tag: WHILE})
	l.reserve(&Word{lexeme: "do", tag: DO})
	l.reserve(&Word{lexeme: "break", tag: BREAK})

	l.reserve(True)
	l.reserve(False)

	l.reserve(Int)
	l.reserve(Char)
	l.reserve(Bool)
	l.reserve(Float)
	return &l
}

func (l *Lexer) reserve(n Token) {
	l.tokens[n.String()] = n
}

func (l *Lexer) read() byte {
	if l.err != nil {
		return 0
	}
	if l._peek != 0 {
		b := l._peek
		l._peek = 0
		return b
	}
	buf := make([]byte, 1)
	_, l.err = l.r.Read(buf)
	return buf[0]
}

func (l *Lexer) readch(c byte) bool {
	l.read()
	if l.peek() != c {
		return false
	}
	l._peek = ' '
	return true
}

func (l *Lexer) peek() byte {
	if l._peek != 0 {
		return l._peek
	} else if l.err != nil {
		return 0
	}
	l._peek = l.read()
	return l._peek
}

func (l *Lexer) Scan() (Token, error) {
read:
	for ; ; l.read() {
		if option.lr {
			log.Printf("%q\n", string(l.peek()))
		}
		switch l.peek() {
		case ' ', '\t':
		case '\n':
			lexerLine++
		case 0:
			if l.err != nil {
				return nil, l.getErr()
			}
		default:
			break read
		}
	}

	// log.Println(string(l.peek()))

	switch l.peek() {
	case '&':
		if l.readch('&') {
			return And, l.getErr()
		} else {
			return Tag('&'), l.getErr()
			// return NewWord("&", '&'), l.getErr()
		}
	case '|':
		if l.readch('|') {
			return Or, l.getErr()
		} else {
			return Tag('|'), l.getErr()
			// return NewWord("|", '|'), l.getErr()
		}
	case '=':
		if l.readch('=') {
			return Eq, l.getErr()
		} else {
			return Tag('='), l.getErr()
			// return NewWord("=", '='), l.getErr()
		}
	case '!':
		if l.readch('=') {
			return Ne, l.getErr()
		} else {
			return Tag('!'), l.getErr()
			// return NewWord("!", '!'), l.getErr()
		}
	case '<':
		if l.readch('=') {
			return Le, l.getErr()
		} else {
			return Tag('<'), l.getErr()
			// return NewWord("<", '<'), l.getErr()
		}
	case '>':
		if l.readch('=') {
			return Ge, l.getErr()
		} else {
			return Tag('>'), l.getErr()
			// return NewWord(">", '>'), l.getErr()
		}
	}
	if isDigit(l.peek()) {
		v := 0
		for {
			v = 10*v + int(l.peek()-'0')
			l.read()
			if !isDigit(l.peek()) {
				break
			}
		}
		if l.peek() != '.' {
			return NewNum(v), nil
		}
		x := float64(v)
		d := float64(10)
		for {
			l.read()
			if !isDigit(l.peek()) {
				break
			}
			x = x + float64(l.peek()-'0')/d
			d *= 10
		}
		return NewReal(x), nil
	}
	if isLetter(l.peek()) {
		var b bytes.Buffer
		for {
			b.WriteByte(l.peek())
			l.read()
			if !isLetter(l.peek()) && !isDigit(l.peek()) {
				break
			}
		}
		s := b.String()
		w, ok := l.tokens[s]
		if !ok {
			w = NewWord(s, ID)
			l.tokens[s] = w
		}
		return w, nil
	}
	tok := Tag(l.peek())
	// log.Printf("--> %+v\n", string(l.peek()))
	l._peek = ' '
	return tok, nil
}

func isDigit(c byte) bool {
	return '0' <= c && c <= '9'
}

func isLetter(c byte) bool {
	return c == '_' || ('A' <= c && c <= 'Z') || ('a' <= c && c <= 'z')
}

func (l *Lexer) getErr() error {
	if l.err == nil {
		return nil
	} else if l.err == io.EOF {
		return l.err
	}
	// return fmt.Errorf("line %d: %s", l.line, l.err)
	return fmt.Errorf("line %d: %s", lexerLine, l.err)
}

type Word struct {
	lexeme string
	tag    Tag
}

func NewWord(lexeme string, tag Tag) Word {
	return Word{lexeme: lexeme, tag: tag}
}
func (w Word) Tag() Tag { return w.tag }
func (w Word) String() string {
	// return fmt.Sprintf("%s:%s", w.tag, w.lexeme)
	return w.lexeme
}

// type Token struct {
// 	tag Tag
// }

// func (t Token) String() string {
// 	return t.tag.String()
// }

var (
	And = Word{lexeme: "&&", tag: AND}
	Or  = Word{lexeme: "||", tag: OR}

	Eq = Word{lexeme: "==", tag: EQ}
	Ne = Word{lexeme: "!=", tag: NE}

	Le = Word{lexeme: "<=", tag: LE}
	Ge = Word{lexeme: ">=", tag: GE}

	Minus = Word{lexeme: "minus", tag: MINUS}

	True  = Word{lexeme: "true", tag: TRUE}
	False = Word{lexeme: "false", tag: FALSE}

	temp = Word{lexeme: "t", tag: TEMP}
)

type Typer interface {
	Lexeme() string
	Width() int
}

type Type struct {
	lexeme string
	tag    Tag
	width  int
}

func (t Type) Lexeme() string { return t.lexeme }
func (t Type) String() string { return t.lexeme }
func (t Type) Tag() Tag       { return t.tag }
func (t Type) Width() int     { return t.width }

var (
	Int   = Type{lexeme: "int", tag: BASIC, width: 4}
	Float = Type{lexeme: "float", tag: BASIC, width: 8}
	Char  = Type{lexeme: "char", tag: BASIC, width: 1}
	Bool  = Type{lexeme: "bool", tag: BASIC, width: 1}
)

func IsNumbericType(t Typer) bool {
	if t == nil {
		return false
	}
	return t.Lexeme() == "int" || t.Lexeme() == "float" || t.Lexeme() == "char"
}

func MaxType(t1, t2 Typer) Typer {
	if !IsNumbericType(t1) || !IsNumbericType(t2) {
		return nil
	}
	if t1.Lexeme() == "float" || t2.Lexeme() == "float" {
		return Float
	}
	if t1.Lexeme() == "int" || t2.Lexeme() == "int" {
		return Int
	}

	return Char
}

type Array struct {
	size   int
	elem   Typer
	tag    Tag
	lexeme string
	width  int
}

func NewArray(sz int, p Typer) Array {
	return Array{tag: INDEX, lexeme: "[]", size: sz, elem: p, width: sz * p.Width()}
}

func (a Array) Tag() Tag       { return a.tag }
func (a Array) Lexeme() string { return a.lexeme }
func (a Array) Width() int     { return a.width }
func (a Array) String() string {
	return fmt.Sprintf("[%d]%s", a.size, a.elem)
}

type Env struct {
	prev  *Env
	table map[Token]Id
}

func NewEnv(prev *Env) *Env { return &Env{table: map[Token]Id{}, prev: prev} }

func (e *Env) put(k Token, v Id) { e.table[k] = v }
func (e *Env) get(k Token) (Id, bool) {
	for env := e; env != nil; env = env.prev {
		if id, ok := env.table[k]; ok {
			return id, true
		}
	}
	return Id{}, false
}

type Num struct {
	value int
	// tag Tag
}

func NewNum(v int) Num {
	return Num{value: v}
}
func (n Num) Tag() Tag       { return NUM }
func (n Num) String() string { return fmt.Sprint(n.value) }

type Real struct {
	value float64
}

func NewReal(v float64) Real {
	return Real{value: v}
}
func (r Real) Tag() Tag       { return REAL }
func (r Real) String() string { return fmt.Sprint(r.value) }
