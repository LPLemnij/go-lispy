package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/alecthomas/participle"
	"github.com/alecthomas/participle/lexer"
	"github.com/alecthomas/participle/lexer/ebnf"
	"github.com/davecgh/go-spew/spew"
)

var lispyLexer = lexer.Must(ebnf.New(`
                digit = "0"…"9" . 
		Float = ("." | digit) {"." | digit} .
                Symbol = ("a"…"z" | "A"…"Z" | "0"…"9" | "+" | "-" | "*" | "/" | "_" | "=" | "<" | ">" | "!" | "&") { "a"…"z" | "A"…"Z" | "0"…"9" | "+" | "-" | "*" | "/" | "_" | "=" | "<" | ">" | "!" | "&"} .
                Whitespace = " " | "\t" | "\n" | "\r" .
		Punct = "!"…"/" | ":"…"@" | "["…` + "\"`\"" + ` | "{"…"~" .
                `))

type LISPY struct {
	Expressions []*Expression `@@`
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
	SExpression *SExpression `|     @@ `
	QExpression *QExpression `|     @@ `
}

type LBuiltin func(*LEnv, *LVal) *LVal

type LValType int

const (
	LVAL_ERR LValType = iota
	LVAL_NUM
	LVAL_SYM
	LVAL_SEXPR
	LVAL_QEXPR
	LVAL_FUN
)

type LVal struct {
	// Type
	Type LValType

	// Basic
	Number float64
	Err    string
	Sym    string

	// Function
	Builtin LBuiltin
	Env     *LEnv
	Formals *LVal
	Body    *LVal

	// Cells
	Cell []*LVal
}

type LEnv struct {
	Par  *LEnv
	Syms []string
	Vals []*LVal
}

func lvalFun(builtin LBuiltin) *LVal {
	val := LVal{Type: LVAL_FUN, Builtin: builtin}
	return &val
}

func lvalNum(x float64) *LVal {
	val := LVal{Type: LVAL_NUM, Number: x}
	return &val
}

func lvalErr(x string) *LVal {
	val := LVal{Type: LVAL_ERR, Err: x}
	return &val
}

func lvalSym(x string) *LVal {
	val := LVal{Type: LVAL_SYM, Sym: x}
	return &val
}

func lvalSexpr() *LVal {
	val := LVal{Type: LVAL_SEXPR}
	return &val
}

func lvalQexpr() *LVal {
	val := LVal{Type: LVAL_QEXPR}
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
		if i != len(l.Cell)-1 {
			fmt.Print(" ")
		}
	}

	fmt.Print(closeChar)
}

func printLVal(l *LVal) {
	switch l.Type {
	case LVAL_NUM:
		fmt.Print(l.Number)
	case LVAL_ERR:
		fmt.Print(l.Err)
	case LVAL_SYM:
		fmt.Print(l.Sym)
	case LVAL_SEXPR:
		printLValExpr(l, "(", ")")
	case LVAL_QEXPR:
		printLValExpr(l, "{", "}")
	case LVAL_FUN:
		if l.Builtin != nil {
			fmt.Println("builtin")
		} else {
			fmt.Print("(\\ ")
			printLVal(l.Formals)
			fmt.Print(" ")
			printLVal(l.Body)
			fmt.Print(")")
		}
	}
}

func lenvGet(env *LEnv, x *LVal) *LVal {
	fmt.Print("The environment we are getting from: ")
	spew.Dump(env)
	// Check if the requested symbol is in the environment and get it. If not, error
	for i := 0; i < len(env.Syms); i++ {
		if env.Syms[i] == x.Sym {
			return lvalCopy(env.Vals[i])
		}
	}

	// If not in the environment, it may be in the parent environment
	if env.Par != nil {
		return lenvGet(env.Par, x)
	} else {
		return lvalErr("Unbound Symbol")
	}
}

func lenvPut(env *LEnv, key *LVal, val *LVal) {
	fmt.Println("The key to append it to: ")
	spew.Dump(key)
	fmt.Println("The value to add to the environment: ")
	spew.Dump(val)
	//Check if the symbol is already in there. If so, overwrite the value and add the new definition
	for i := 0; i < len(env.Syms); i++ {
		if env.Syms[i] == key.Sym {
			env.Vals[i] = lvalCopy(val)
			return
		}
	}

	//If not, append the symbol to the environment and the value to the values of the environment
	env.Syms = append(env.Syms, key.Sym)
	env.Vals = append(env.Vals, lvalCopy(val))

	fmt.Println("After putting to environment: ")
	spew.Dump(env)
}

func lenvDef(env *LEnv, key *LVal, val *LVal) {
	for env.Par != nil {
		env = env.Par
	}
	lenvPut(env, key, val)
}

func lvalAdd(v *LVal, x *LVal) *LVal {
	v.Cell = append(v.Cell, x)
	return v
}

func lvalPop(v *LVal, i int) *LVal {
	x := v.Cell[i]

	v.Cell = append(v.Cell[:i], v.Cell[i+1:]...)
	return x
}

func lvalCopy(v *LVal) *LVal {
	x := LVal{Cell: make([]*LVal, 0)}
	x.Type = v.Type

	switch v.Type {
	case LVAL_FUN:
		if v.Builtin != nil {
			x.Builtin = v.Builtin
		} else {
			x.Builtin = nil
			x.Env = lenvCopy(v.Env)
			x.Formals = lvalCopy(v.Formals)
			x.Body = lvalCopy(v.Body)
		}
	case LVAL_NUM:
		x.Number = v.Number
	case LVAL_ERR:
		x.Err = v.Err
	case LVAL_SYM:
		x.Sym = v.Sym
	case LVAL_SEXPR:
		for i := 0; i < len(v.Cell); i++ {
			x.Cell = append(x.Cell, lvalCopy(v.Cell[i]))
		}
	case LVAL_QEXPR:
		for i := 0; i < len(v.Cell); i++ {
			x.Cell = append(x.Cell, lvalCopy(v.Cell[i]))
		}
	}

	return &x
}

func lenvCopy(env *LEnv) *LEnv {
	x := LEnv{Syms: make([]string, 0), Vals: make([]*LVal, 0)}
	x.Par = env.Par

	for i := 0; i < len(env.Syms); i++ {
		x.Syms = append(x.Syms, env.Syms[i])
	}
	for i := 0; i < len(env.Vals); i++ {
		x.Vals = append(x.Vals, lvalCopy(env.Vals[i]))
	}

	return &x
}

//Probably don't need this since garbage collection
func lvalTake(v *LVal, i int) *LVal {
	x := lvalPop(v, i)
	return x
}

func builtinHead(e *LEnv, a *LVal) *LVal {
	//There may have been more than one list in the lval or something
	if len(a.Cell) != 1 {
		return lvalErr("Function 'head' was given too many arguments")
	}

	//Not a q expression
	if a.Cell[0].Type != LVAL_QEXPR {
		return lvalErr("Function 'head' must be given a q expression as an argument")
	}

	//Head was passed in an empty list
	if len(a.Cell[0].Cell) == 0 {
		return lvalErr("Function 'head' passed in an empty q expression")
	}

	//Get the actual list out of the cell
	v := lvalTake(a, 0)
	//Remove everything from that list until we have just the lval of that list with one element(which is the head)
	for len(v.Cell) > 1 {
		_ = lvalPop(v, 1)
	}
	return v
}

func builtinLambda(e *LEnv, a *LVal) *LVal {
	if len(a.Cell) != 2 {
		return lvalErr("Lambda was given an improper number of arguments")
	}

	if a.Cell[0].Type != LVAL_QEXPR {
		return lvalErr("First argument was not a q expression")
	}

	if a.Cell[1].Type != LVAL_QEXPR {
		return lvalErr("Second argument was not a q expression")
	}

	for i := 0; i < len(a.Cell[0].Cell); i++ {
		if a.Cell[0].Cell[i].Type != LVAL_SYM {
			return lvalErr("First argument must contain list of symbols")
		}
	}

	formals := lvalPop(a, 0)
	body := lvalPop(a, 0)

	return lvalLambda(formals, body)
}

func builtinTail(e *LEnv, a *LVal) *LVal {
	if len(a.Cell) != 1 {
		return lvalErr("Function 'tail' was given too many arguments")
	}

	if a.Cell[0].Type != LVAL_QEXPR {
		return lvalErr("Function 'tail' must be given a q expression as an argument")
	}

	if len(a.Cell[0].Cell) == 0 {
		return lvalErr("Function 'tail' passed in an empty q expression")
	}

	v := lvalTake(a, 0)
	v.Cell = v.Cell[1:]
	return v
}

func builtinList(e *LEnv, a *LVal) *LVal {
	a.Type = LVAL_QEXPR
	return a
}

func builtinEval(e *LEnv, a *LVal) *LVal {
	if len(a.Cell) != 1 {
		return lvalErr("Function 'eval' given too many arguments")
	}

	if a.Cell[0].Type != LVAL_QEXPR {
		return lvalErr("function 'eval' must be given a q expression as an argument")
	}

	x := lvalTake(a, 0)
	x.Type = LVAL_SEXPR

	return lvalEval(e, x)
}

func builtinJoin(e *LEnv, a *LVal) *LVal {
	for i := 0; i < len(a.Cell); i++ {
		if a.Cell[i].Type != LVAL_QEXPR {
			return lvalErr("One of the arguments to join was not a q expression")
		}
	}

	x := lvalPop(a, 0)
	for len(a.Cell) > 0 {
		x = lvalJoin(x, lvalPop(a, 0))
	}
	return x
}

func lvalJoin(a *LVal, b *LVal) *LVal {
	for len(b.Cell) > 0 {
		a = lvalAdd(a, lvalPop(b, 0))
	}

	return a
}

func lvalCall(e *LEnv, f *LVal, a *LVal) *LVal {
	//If it is a builtin function, return the result of running that function
	if f.Builtin != nil {
		return f.Builtin(e, a)
	}

	//Bind the arguments that were passed into the function
	for len(a.Cell) > 0 {
		if len(f.Formals.Cell) == 0 {
			return lvalErr("Function passed too many arguments")
		}

		sym := lvalPop(f.Formals, 0)
		val := lvalPop(a, 0)
		lenvPut(f.Env, sym, val)
	}

	if len(f.Formals.Cell) == 0 {
		f.Env.Par = e

		return builtinEval(f.Env, lvalAdd(lvalSexpr(), lvalCopy(f.Body)))
	} else {
		return lvalCopy(f)
	}
}

func lvalLambda(formals *LVal, body *LVal) *LVal {
	v := LVal{Type: LVAL_FUN}

	v.Builtin = nil
	v.Env = &LEnv{Syms: make([]string, 0), Vals: make([]*LVal, 0)}
	v.Formals = formals
	v.Body = body

	return &v
}

func lenvAddBuiltin(e *LEnv, name string, f LBuiltin) {
	k := lvalSym(name)
	v := lvalFun(f)
	lenvPut(e, k, v)
}

func lenvAddBuiltins(e *LEnv) {
	lenvAddBuiltin(e, "list", builtinList)
	lenvAddBuiltin(e, "head", builtinHead)
	lenvAddBuiltin(e, "tail", builtinTail)
	lenvAddBuiltin(e, "eval", builtinEval)
	lenvAddBuiltin(e, "join", builtinJoin)
	lenvAddBuiltin(e, "def", builtinDef)
	lenvAddBuiltin(e, "=", builtinPut)
	lenvAddBuiltin(e, "fn", builtinLambda)
	lenvAddBuiltin(e, "+", builtinAdd)
	lenvAddBuiltin(e, "-", builtinSubtract)
	lenvAddBuiltin(e, "*", builtinMultiply)
	lenvAddBuiltin(e, "/", builtinDivide)
}

func builtinVar(e *LEnv, a *LVal, op string) *LVal {
	//When using def, we make sure that the first parameter is a list of symbols
	syms := a.Cell[0]
	for i := 0; i < len(syms.Cell); i++ {
		if syms.Cell[i].Type != LVAL_SYM {
			return lvalErr("Function def cannot define a non symbol")
		}
	}

	//Check if the lists match
	if len(syms.Cell) != len(a.Cell)-1 {
		return lvalErr("The symbol list and the value list are different lengths")
	}

	for i := 0; i < len(syms.Cell); i++ {
		if op == "def" {
			lenvDef(e, syms.Cell[i], a.Cell[i+1])
		}

		if op == "=" {
			lenvPut(e, syms.Cell[i], a.Cell[i+1])
		}
	}

	return lvalSexpr()
}

func builtinDef(e *LEnv, a *LVal) *LVal {
	return builtinVar(e, a, "def")
}

func builtinPut(e *LEnv, a *LVal) *LVal {
	return builtinVar(e, a, "=")
}

func builtinAdd(e *LEnv, a *LVal) *LVal {
	return builtinOp(e, a, "+")
}

func builtinSubtract(e *LEnv, a *LVal) *LVal {
	return builtinOp(e, a, "-")
}

func builtinMultiply(e *LEnv, a *LVal) *LVal {
	return builtinOp(e, a, "*")
}

func builtinDivide(e *LEnv, a *LVal) *LVal {
	return builtinOp(e, a, "/")
}

func builtinOp(e *LEnv, a *LVal, op string) *LVal {
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

func lvalEvalSexpr(e *LEnv, v *LVal) *LVal {
	// Evaluate Children
	//The recursive case is a bit confusing but you basically just assume you have an lvalEval that works correctly and go through the children and evaulate them
	// The interaction between lvalEvalSexpr and lvalEval is what recursively evaluates the structure and goes deep into the nested sexpressions, evaluating the deepst first
	for i := 0; i < len(v.Cell); i++ {
		v.Cell[i] = lvalEval(e, v.Cell[i])
	}

	//Check for an error type lval in the evaluation. If found, return that lval
	//using lvalTake
	for i := 0; i < len(v.Cell); i++ {
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
	if f.Type != LVAL_FUN {
		return lvalErr("First Element is not a function")
	}

	//Run the operation using the currnet lval and the input symbol
	result := lvalCall(e, f, v)
	return result
}

// Evaluating the actual numberical result of the sexpression
func lvalEval(e *LEnv, v *LVal) *LVal {
	if v.Type == LVAL_SYM {
		x := lenvGet(e, v)
		return x
	}

	// If this is a lval representation of an sexpression, we evaluate that
	if v.Type == LVAL_SEXPR {
		return lvalEvalSexpr(e, v)
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
		} else if node.QExpression != nil {
			x = lvalQexpr()
			for i := 0; i < len(node.QExpression.Expressions); i++ {
				x.Cell = append(x.Cell, lvalRead(node.QExpression.Expressions[i]))
			}
		}
	// If it's the root node, we return the lvalRead of each of the expressions, recursively building the lval tree structure depth first
	case *LISPY:
		x = lvalSexpr()
		node, _ := node.(*LISPY)
		for i := 0; i < len(node.Expressions); i++ {
			x.Cell = append(x.Cell, lvalRead(node.Expressions[i]))
		}
	// Same as the sexpresison above
	case *SExpression:
		x = lvalSexpr()
		node, _ := node.(*SExpression)
		for i := 0; i < len(node.Expressions); i++ {
			x.Cell = append(x.Cell, lvalRead(node.Expressions[i]))
		}
	case *QExpression:
		x = lvalQexpr()
		node, _ := node.(*QExpression)
		for i := 0; i < len(node.Expressions); i++ {
			x.Cell = append(x.Cell, lvalRead(node.Expressions[i]))
		}
	}

	return x
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

		//j, _ := json.Marshal(lvalRead(lispyRootNode))
		//fmt.Println(string(j))
		printLVal(lvalEval(e, lvalRead(lispyRootNode)))
	}

}
