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

func printLValExpr(l *LVal, openChar string, closeChar string) {
	fmt.Print(openChar)

	for i := 0; i < len(l.Cell); i++ {
		printLVal(l.Cell[i])
		if i != len(l.Cell) - 1 {
			fmt.Print(" ")
		}
	}

	fmt.Print(closeChar)
}

func printLVal(l *LVal) {
	switch l.Type {
	case LVAL_NUM:
		fmt.Println(l.Number)
	case LVAL_ERR:
		fmt.Println(l.Err)
	case LVAL_SYM:
		fmt.Println(l.Sym)
	case LVAL_SEXPR:
		printLValExpr(l, "(", ")")
	}
}

func lvalPop(v *LVal, i int) *LVal {
	x := v.Cell[i]

	v.Cell = append(v.Cell[:i], v.Cell[i+1:]...)
	return x
}

//Probably don't need this since garbage collection
func lvalTake(v *LVal, i int) *LVal {
	x := lvalPop(v, i)
	return x
}

func builtinOp(a *LVal, op string) *LVal {
	// Make sure all arguments are numbers so we can eval
	for i := 0; i < len(a.Cell); i++ {
		if a.Cell[i].Type != LVAL_NUM {
			return lvalErr("Cannot operate on a non number")
		}
	}

	x := lvalPop(a, 0)
	if op == "-" && len(a.Cell) == 0 {
		x.Number = -x.Number
	}

	for len(a.Cell) > 0 {
		y := lvalPop(a, 0)

		if op == "+" {
			x.Number = add(x.Number, y.Number)
		}

		if op == "-" {
			x.Number = subtract(x.Number, y.Number)
		}

		if op == "*" {
			x.Number = multiply(x.Number, y.Number)
		}

		if op == "/" {
			if y.Number == 0 {
				x = lvalErr("Cannot divide by 0")
				break
			}
			x.Number = divide(x.Number, y.Number)
		}
	}

	return x
}

func lvalEvalSexpr(v *LVal) *LVal {
	// Evaluate Children
	//The recursive case is a bit confusing but you basically just assume you have an lvalEval that works correctly and go through the children and evaulate them
	// The interaction between lvalEvalSexpr and lvalEval is what recursively evaluates the structure and goes deep into the nested sexpressions, evaluating the deepst first
	for i := 0; i < len(v.Cell); i++ {
		v.Cell[i] = lvalEval(v.Cell[i])
	}

	//Check for an error type lval in the evaluation. If found, return that lval 
	//using lvalTake
	for i:= 0; i < len(v.Cell); i++ {
		if v.Cell[i].Type == LVAL_ERR {
			return lvalTake(v, i)
		}
	}

	//If the cell has length 0, this is the empty list
	if len(v.Cell) == 0 {
		return v
	}

	//If the cell has length 1, it looks like (1) and we just want to return the 
	//lval representing the number 1
	if len(v.Cell) == 1 {
		return lvalTake(v, 0)
	}

	//Ensure that the first lval in the s-expression is a symbol
	f := lvalPop(v, 0)
	if f.Type != LVAL_SYM {
		return lvalErr("S-Expression does not start with symbol")
	}

	//Run the operation using the currnet lval and the input symbol
	result := builtinOp(v, f.Sym)
	return result
}

// Evaluating the actual numberical result of the sexpression
func lvalEval(v *LVal) *LVal {
	// If this is a lval representation of an sexpression, we evaluate that
	if v.Type == LVAL_SEXPR {
		return lvalEvalSexpr(v)
	}

	// Otherwise, we just return the lval representation as it is since it is either an lval representing a number or a symbol or error already
	return v
}
//Takes a parser node(Lispy, Expression, or SExpression and turns it into an interpreter lval node)
func lvalRead(node interface{}) *LVal {
	var x *LVal
	switch node.(type) {
	// If it's an expression, then we either return an lval node that represents a number,
	// and lval node that represents a symbol,
	// or an lval node that represents an s expression(which is an lval node whose cell is the lval read of it's expression list)
	// So really we are going all the way to the bottom depth of the sexpression recursively reading it into the lval node.
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
			for i := 0; i < len(node.SExpression.Expressions); i++ {
				x.Cell = append(x.Cell, lvalRead(node.SExpression.Expressions[i]))
			}
		}
	// If it's the root node, we return the lvalRead of each of the expressions, recursively building the lval tree structure depth first
	case *LISPY:
		x = lvalSexpr()
		node, _ := node.(*LISPY)
                for i := 0; i < len(node.Root.Expressions); i++ {
                        x.Cell = append(x.Cell, lvalRead(node.Root.Expressions[i]))
                }
	// Same as the sexpresison above
	case *SExpression:
		x = lvalSexpr()
		node, _ := node.(*SExpression)
                for i := 0; i < len(node.Expressions); i++ {
                        x.Cell = append(x.Cell, lvalRead(node.Expressions[i]))
                }
	}

	return x
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
			return
		}


		//j, _ := json.Marshal(lvalRead(lispyRootNode))
		//fmt.Println(string(j))

		printLVal(lvalEval(lvalRead(lispyRootNode)))
        }


}
