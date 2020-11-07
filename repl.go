package main

import (
	"fmt"
	"os"
	"strings"
	"bufio"
	"encoding/json"
	"github.com/alecthomas/participle"
	"github.com/alecthomas/participle/lexer"
	"github.com/alecthomas/participle/lexer/ebnf"
)

var lispyLexer = lexer.Must(ebnf.New(`
                Symbol = "max" | "+" | "-" | "*" | "/" .
                Float = ("." | digit) {"." | digit} .
                Whitespace = " " | "\t" | "\n" | "\r" .
		Punct = "!"…"/" | ":"…"@" | "["…` + "\"`\"" + ` | "{"…"~" .
                digit = "0"…"9" .`))
type LISPY struct {
	Root *SExpression  `@@`
}

type SExpression struct {
	Expressions []*Expression ` "(" @@+ ")"`
}

type Expression struct {
	Number        *float64     `      @Float `
	Sym           *string      `|     @Symbol `
	SExpression   *SExpression `|     @@ `
}

type LValType int
const (
	LVAL_ERR LValType = iota
	LVAL_NUM
	LVAL_SYM
	LVAL_SEXPR
)

type LVal struct {
	Number float64
	Type   LValType
	Err    string
	Sym    string
	Cell   []*LVal
}

func lvalNum(x float64) *LVal {
	val := LVal{ Type: LVAL_NUM, Number: x }
	return &val
}

func lvalErr(x string) *LVal {
	val := LVal{ Type: LVAL_ERR, Err: x }
	return &val
}

func lvalSym(x string) *LVal {
	val := LVal{ Type: LVAL_SYM, Sym: x}
	return &val
}

func lvalSexpr() *LVal {
	val := LVal{ Type: LVAL_SEXPR }
	return &val
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

func max(a float64, b float64) float64 {
	if a <= b {
		return b
	}
	return a
}

func min(a float64, b float64) float64 {
	if a <= b {
		return a
	}
	return b
}

func printLVal(l LVal) {
	if l.Err != "" {
		fmt.Println(l.Err)
	}

	fmt.Println(l.Number)
}

/* func evalOp(op string, a LVal, b LVal) LVal {
	if a.Err != "" {
		return a
	} else if b.Err != "" {
		return b
	}
	var val LVal
        switch op {
        case "+":
		val.Number = new(float64)
		*val.Number = add(*a.Number, *b.Number)
        case "-":
		val.Number = new(float64)
		*val.Number = subtract(*a.Number, *b.Number)
        case "*":
		val.Number = new(float64)
		*val.Number = multiply(*a.Number, *b.Number)
        case "/":
                if *b.Number == 0 {
			*val.Err = "Cannot divide by zero"
		} else {
			val.Number = new(float64)
			*val.Number = divide(*a.Number, *b.Number)
		}
	case "max":
		*val.Number = max(*a.Number, *b.Number)
	default:
		*val.Err = "Bad Operator"
	}
	return val
}*/

func lvalRead(node interface{}) *LVal {
	var x *LVal
	switch node.(type) {
	case *Expression:
		node, _ := node.(*Expression)
		if node.Number != nil {
			x = lvalNum(*node.Number)
			return x
		} else if node.Sym != nil {
			x = lvalSym(*node.Sym)
			return x
		} else if node.SExpression != nil {
			x = lvalSexpr()
		}
	case *LISPY:
		x = lvalSexpr()
		node, _ := node.(*LISPY)
                for i := 0; i < len(node.Root.Expressions); i++ {
                        x.Cell = append(x.Cell, lvalRead(node.Root.Expressions[i]))
                }
	case *SExpression:
		x = lvalSexpr()
		node, _ := node.(*SExpression)
                for i := 0; i < len(node.Expressions); i++ {
                        x.Cell = append(x.Cell, lvalRead(node.Expressions[i]))
                }
	}

	return x
}

/*func eval(testing interface{}) LVal {
	switch testing.(type) {
	case *LISPY:
		testing, _ := testing.(*LISPY)
		return eval(testing.Expressions[0])
	case *Expression:
		testing, _ := testing.(*Expression)
		if testing.Op != nil {
			opString := *testing.Op
			accum := eval(testing.Expressions[0])
			for i := 1; i < len(testing.Expressions); i++ {
				accum = evalOp(opString, accum, eval(testing.Expressions[i]))
			}
			return accum
		} else if testing.Value.Err != nil {
			return *testing.Value
		} else if testing.Value.Number != nil {
			return *testing.Value
		}else {
			return eval(testing.Expressions[0])
		}
	}
	var x LVal;
	*(x.Err) = LERR_BAD_OP
	return x
} */

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
			return
		}

		j, _ := json.Marshal(lvalRead(lispyRootNode))
		fmt.Println(string(j))
		fmt.Println("Evaluating...")

		//printLVal(eval(lispyRootNode))
        }


}
