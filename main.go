package main

import (
	"log"

	"flag"
	"fmt"
	"os"
)

var option struct {
	lr     bool
	pmatch bool
	file   string
	el     bool
	ps     bool
}

func init() {
	log.SetFlags(log.Lshortfile)
	flag.BoolVar(&option.lr, "lr", false, "log lexer read")
	flag.BoolVar(&option.pmatch, "pm", false, "log parser match")
	flag.BoolVar(&option.el, "el", false, "log emitLabel")
	flag.BoolVar(&option.ps, "ps", false, "print program block")
	flag.StringVar(&option.file, "file", "", "test file")
}

func main() {
	flag.Parse()
	var lex *Lexer
	if option.file == "" {
		lex = NewLexer(os.Stdin)
	} else {
		f, err := os.Open(option.file)
		if err != nil {
			panic(err)
		}
		lex = NewLexer(f)
	}
	parser := NewParser(lex)
	parser.program()
	fmt.Println()
}
