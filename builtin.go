package main

import (
	"io/ioutil"

	"github.com/alecthomas/participle"
)

//////////////////////////////////////////////////////////////////////////////////////////////
// This file contains the suite of builtin functions that my go version of lisp comes with. //
// Any builtin functions can be added by following the LBuiltin construct and adding them   //
// to the global environment in lenv.go														//
//////////////////////////////////////////////////////////////////////////////////////////////

type LBuiltin func(*LEnv, *LVal) *LVal

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

func builtinPrint(e *LEnv, a *LVal) *LVal {
	printLVal(a)

	return lvalSexpr()
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

func builtinLessThan(e *LEnv, a *LVal) *LVal {
	return builtinCond(e, a, "<")
}

func builtinGreaterThan(e *LEnv, a *LVal) *LVal {
	return builtinCond(e, a, ">")
}

func builtinGreaterThanOrEqualTo(e *LEnv, a *LVal) *LVal {
	return builtinCond(e, a, ">=")
}

func builtinLessThanOrEqualTo(e *LEnv, a *LVal) *LVal {
	return builtinCond(e, a, "<=")
}

func builtinEq(e *LEnv, a *LVal) *LVal {
	return builtinCond(e, a, "==")
}

func builtinIf(e *LEnv, a *LVal) *LVal {
	if len(a.Cell) != 3 {
		return lvalErr("The if condition must be passed in three arguments.")
	}

	if a.Cell[0].Type != LVAL_NUM {
		return lvalErr("The if condition's first arugment must be of type num")
	}
	if a.Cell[1].Type != LVAL_QEXPR {
		return lvalErr("The if condition's second argument must be of type qexpr")
	}

	if a.Cell[2].Type != LVAL_QEXPR {
		return lvalErr("The if condition's third argument must be of type qexpr.")
	}

	var x *LVal
	a.Cell[1].Type = LVAL_SEXPR
	a.Cell[2].Type = LVAL_SEXPR

	if a.Cell[0].Number != 1 {
		x = lvalEval(e, lvalPop(a, 2))
	} else {
		x = lvalEval(e, lvalPop(a, 1))
	}

	return x
}

func builtinCond(e *LEnv, a *LVal, cond string) *LVal {
	x := LVal{Type: LVAL_NUM, Number: 0}

	firstArg := lvalPop(a, 0)
	secondArg := lvalPop(a, 0)

	if cond == "<" {
		if firstArg.Number < secondArg.Number {
			x.Number = 1
		}
	}

	if cond == "<=" {
		if firstArg.Number <= secondArg.Number {
			x.Number = 1
		}
	}

	if cond == ">" {
		if firstArg.Number > secondArg.Number {
			x.Number = 1
		}
	}

	if cond == ">=" {
		if firstArg.Number >= secondArg.Number {
			x.Number = 1
		}
	}

	// Equality has to be a different thing since we don't only mess with numbers
	if cond == "==" {
		eq := lvalEq(firstArg, secondArg)
		if eq == true {
			x.Number = 1
		}
	}

	return &x
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

func builtinLoad(e *LEnv, a *LVal) *LVal {
	fileRootNode := &LISPY{}
	lispyParser, err := participle.Build(&LISPY{}, participle.Lexer(lispyLexer), participle.Elide("Whitespace"))

	if err != nil {
		return lvalErr(string(err.Error()))
	}

	if a.Cell[0].String != "" {
		fileBytes, err := ioutil.ReadFile(a.Cell[0].String)

		if err != nil {
			return lvalErr(string(err.Error()))
		}

		fileText := string(fileBytes)

		err = lispyParser.ParseString(fileText, fileRootNode)

		rootLVal := lvalRead(fileRootNode)

		for len(rootLVal.Cell) > 0 {
			x := lvalEval(e, lvalPop(rootLVal, 0))

			if x.Type == LVAL_ERR {
				printLVal(x)
			}
		}
	} else {
		printLVal(lvalErr("Cannot parse the given file due to syntax errors."))
	}

	return lvalSexpr()
}
