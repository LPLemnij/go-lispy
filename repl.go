package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/alecthomas/participle"
	"github.com/alecthomas/participle/lexer"
	"github.com/alecthomas/participle/lexer/ebnf"
)

//////////////////////////////////////////////////////////////////////////////////////////////
// This file contains the parsing construct in participle and also the repl logic/entry 	//
// point of the program. It takes text you write in the command line and turns it into 		//
// participle nodes which then get translated into interpreter constructs using functions 	//
// inside of lval.go																		//
//////////////////////////////////////////////////////////////////////////////////////////////

//This grammar can be cleaned up. I should probably do that at some point if I'm ever going to add this as a resume project or something.
// As a note, I can make lowercase be its own range, upper case be its own range, and the symbols have their own range and then construct
// Symbol and String out of those constructs.
var lispyLexer = lexer.Must(ebnf.New(`
		digit = "0"…"9" . 
		String = "\"" ("a"…"z" | "A"…"Z" | "0"…"9" | "\"" | "[" | "]" | "{" | "}" | "(" | ")" | "<" | ">" | "'"  | "=" | "|" | "." | "," | ";" | "\\") { "a"…"z" | "A"…"Z" | "0"…"9" "[" | "]" | "{" | "}" | "(" | ")" | "<" | ">" | "'" | "=" | "|" | "." | "," | ";" | "\\" 		} "\"" .
		Float = ("." | digit) {"." | digit} .
        Symbol = ("a"…"z" | "A"…"Z" | "0"…"9" | "+" | "-" | "*" | "/" | "_" | "=" | "<" | ">" | "!" | "&") { "a"…"z" | "A"…"Z" | "0"…"9" | "+" | "-" | "*" | "/" | "_" | "=" | "<" | ">" | "!" | "&"} .
        Whitespace = " " | "\t" | "\n" | "\r" .
                `))

type LISPY struct {
	Expressions []*Expression `@@*`
}

type QExpression struct {
	Expressions []*Expression ` "{" @@* "}"`
}

type SExpression struct {
	Expressions []*Expression ` "(" @@* ")"`
}

type Expression struct {
	Number      *float64     `      @Float `
	Sym         *string      `|     @Symbol `
	String      *string      `|     @String `
	SExpression *SExpression `|     @@ `
	QExpression *QExpression `|     @@ `
}

func main() {
	e := &LEnv{Syms: make([]string, 0), Vals: make([]*LVal, 0)}
	lenvAddBuiltins(e)
	reader := bufio.NewReader(os.Stdin)
	lispyRootNode := &LISPY{}
	lispyParser, err := participle.Build(&LISPY{},
		participle.Lexer(lispyLexer),
		participle.Elide("Whitespace"),
	)

	if err != nil {
		fmt.Println(err)
	}

	fmt.Println("My Go Lisp v1")

	for {
		//Read
		fmt.Print("\nGo-Lispy>")
		text, _ := reader.ReadString('\n')
		text = strings.Replace(text, "\n", "", -1)
		//Parse
		err = lispyParser.ParseString(text, lispyRootNode)

		if err != nil {
			fmt.Println(err)
			return
		}

		printLVal(lvalEval(e, lvalRead(lispyRootNode)))
	}

}
