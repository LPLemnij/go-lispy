package main

//////////////////////////////////////////////////////////////////////////////////////////////
// This file contains all of the functionality for the environment handling of different    //
// functions and also handles the initial adding of builtin functions to the global env.    //
//////////////////////////////////////////////////////////////////////////////////////////////

type LEnv struct {
	Par  *LEnv
	Syms []string
	Vals []*LVal
}

// Get a value out of an environment or its parent chain
func lenvGet(env *LEnv, x *LVal) *LVal {
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

// Add a value into the environment depending on whether it exists or not
func lenvPut(env *LEnv, key *LVal, val *LVal) {
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
}

// Define a new variable or function
func lenvDef(env *LEnv, key *LVal, val *LVal) {
	for env.Par != nil {
		env = env.Par
	}
	lenvPut(env, key, val)
}

// Copy all symbols and values from one environment to another and set its parent accordingly
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

// Add a builtin function to the environment passed in
func lenvAddBuiltin(e *LEnv, name string, f LBuiltin) {
	k := lvalSym(name)
	v := lvalFun(f)
	lenvPut(e, k, v)
}

// Initialize the global environment with the suite of builtin functions
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
	lenvAddBuiltin(e, "<", builtinLessThan)
	lenvAddBuiltin(e, ">", builtinGreaterThan)
	lenvAddBuiltin(e, "<=", builtinLessThanOrEqualTo)
	lenvAddBuiltin(e, ">=", builtinGreaterThanOrEqualTo)
	lenvAddBuiltin(e, "==", builtinEq)
	lenvAddBuiltin(e, "if", builtinIf)
	lenvAddBuiltin(e, "load", builtinLoad)
	lenvAddBuiltin(e, "print", builtinPrint)
}
