package main

import (
	"fmt"
	"reflect"
	"strings"
)

//////////////////////////////////////////////////////////////////////////////////////////////
// This file contains the bread and butter of the actual interpreter. It contains the lval  //
// type along with functions to manipulate, build, delete, and process interpreter nodes of //
// all types. Also, it contains the functionality for translating between participle struct //
// nodes and lval nodes.																	//
//////////////////////////////////////////////////////////////////////////////////////////////

type LValType int

const (
	LVAL_ERR LValType = iota
	LVAL_NUM
	LVAL_SYM
	LVAL_SEXPR
	LVAL_QEXPR
	LVAL_FUN
	LVAL_STR
)

type LVal struct {
	// Type
	Type LValType

	// Basic
	Number float64
	Err    string
	Sym    string
	String string

	// Function
	Builtin LBuiltin
	Env     *LEnv
	Formals *LVal
	Body    *LVal

	// Cells
	Cell []*LVal
}

//Constructors for different kinds of lvals

func lvalString(str string) *LVal {
	val := LVal{Type: LVAL_STR, String: str}
	return &val
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

func lvalLambda(formals *LVal, body *LVal) *LVal {
	v := LVal{Type: LVAL_FUN}

	v.Builtin = nil
	v.Env = &LEnv{Syms: make([]string, 0), Vals: make([]*LVal, 0)}
	v.Formals = formals
	v.Body = body

	return &v
}

// Functions that aid in printing lvals

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
	case LVAL_STR:
		fmt.Print("\"" + l.String + "\"")
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

// Helper functions for processing lvals

// Process a string by removing the quotes around it
func lvalProcessStr(str string) string {
	return strings.Replace(str, "\"", "", -1)
}

//Add an input lval to the first lval's cell
func lvalAdd(v *LVal, x *LVal) *LVal {
	v.Cell = append(v.Cell, x)
	return v
}

// Add all of the cell elements from b onto a. Maintains consistency with the book.
func lvalJoin(a *LVal, b *LVal) *LVal {
	for len(b.Cell) > 0 {
		a = lvalAdd(a, lvalPop(b, 0))
	}

	return a
}

//Pop an element at a certain index from the given lval cell
func lvalPop(v *LVal, i int) *LVal {
	x := v.Cell[i]

	v.Cell = append(v.Cell[:i], v.Cell[i+1:]...)
	return x
}

//Same as pop really, this was just part of the book so I kept it consistent
func lvalTake(v *LVal, i int) *LVal {
	x := lvalPop(v, i)
	return x
}

//Return a deep copy of an lval
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
	case LVAL_STR:
		x.String = v.String
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

// Check if two lvals are equal considering their types
func lvalEq(firstArg *LVal, secondArg *LVal) bool {
	if firstArg.Type != secondArg.Type {
		return false
	}

	switch firstArg.Type {
	case LVAL_NUM:
		return firstArg.Number == secondArg.Number
	case LVAL_ERR:
		return firstArg.Err == secondArg.Err
	case LVAL_SYM:
		return firstArg.Sym == secondArg.Sym
	case LVAL_FUN:
		if firstArg.Builtin != nil || secondArg.Builtin != nil {
			return reflect.ValueOf(firstArg.Builtin) == reflect.ValueOf(secondArg.Builtin)
		}

		return lvalEq(firstArg.Formals, secondArg.Formals) && lvalEq(firstArg.Body, secondArg.Body)
	case LVAL_STR:
		return firstArg.String == secondArg.String
	case LVAL_QEXPR:
		fallthrough
	case LVAL_SEXPR:
		if len(firstArg.Cell) != len(secondArg.Cell) {
			return false
		}

		for i := 0; i < len(firstArg.Cell); i++ {
			if !lvalEq(firstArg.Cell[i], secondArg.Cell[i]) {
				return false
			}
		}

		return true
	}

	return false
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
		} else if node.String != nil {
			x = lvalString(lvalProcessStr(*node.String))
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

// Call a function that is represented by an lval
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

		if sym.Sym == "&" {
			if len(f.Formals.Cell) != 1 {
				return lvalErr("Symbol & not followed by a single symbol.")
			}

			nsym := lvalPop(f.Formals, 0)
			lenvPut(f.Env, nsym, builtinList(e, a))
		}

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
