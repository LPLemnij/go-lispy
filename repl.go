package main

import (
	"fmt"
	"os"
	"strings"
	"bufio"
	//"encoding/json"
	"github.com/alecthomas/participle"
	"github.com/alecthomas/participle/lexer"
	"github.com/alecthomas/participle/lexer/ebnf"
)

var lispyLexer = lexer.Must(ebnf.New(`
                Operator = "+" | "-" | "*" | "/" .
                Float = ("." | digit) {"." | digit} .
                Whitespace = " " | "\t" | "\n" | "\r" .
		Punct = "!"…"/" | ":"…"@" | "["…` + "\"`\"" + ` | "{"…"~" .
                digit = "0"…"9" .`))

type LISPY struct {
	Expressions []*Expression `@@+`
}

type Expression struct {
	Number        *float64    `      @Float `
	Op            *string     `| "(" @Operator `
	Expressions []*Expression `      @@+  ")"`
}

func add(a float64, b float64) float64 {
        return a + b
}

func subtract(a float64, b float64) float64 {
        return a - b
}

func multiply(a float64, b float64) float64 {
        return a * b
}

func divide(a float64, b float64) float64 {
        return a / b
}

func evalOp(op string, a float64, b float64) float64 {
        switch op {
        case "+":
                return add(a, b)
        case "-":
                return subtract(a, b)
        case "*":
                return multiply(a, b)
        case "/":
                return divide(a, b)
	}
	return 0
}

func eval(testing interface{}) float64 {
	switch testing.(type) {
	case *LISPY:
		testing, _ := testing.(*LISPY)
		return eval(testing.Expressions[0])
	case *Expression:
		testing, _ := testing.(*Expression)
		if testing.Number != nil {
			return *testing.Number
		} else if testing.Op != nil {
			opString := *testing.Op
			accum := eval(testing.Expressions[0])

			for i := 1; i < len(testing.Expressions); i++ {
				accum = evalOp(opString, accum, eval(testing.Expressions[i]))
			}

			return accum
		} else {
			return eval(testing.Expressions[0])
		}
	default:
		return 0

	}
}


func main() {
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
                fmt.Print("Go-Lispy>")
                text, _ := reader.ReadString('\n')
                text = strings.Replace(text, "\n", "", -1)

		//Parse
		err = lispyParser.ParseString(text, lispyRootNode)

		if err != nil {
			fmt.Println(err)
		}

		//j, _ := json.Marshal(lispyRootNode)
		//fmt.Println(string(j))
		//fmt.Println("Evaluating...")
		fmt.Println(eval(lispyRootNode))
        }

}
