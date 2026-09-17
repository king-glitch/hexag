package parser

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

// groupPrefix maps a handler type name (e.g. "IdentityHandler") to the URL
// group prefix each of its Register* methods was mounted under (e.g.
// "Register" -> "/identity"), read from the composition-root file
// (handler.go) where `xHandler.Register(api.Group("/x"))` is wired.
type registerPrefixes map[string]map[string]string // handlerType -> registerMethod -> prefix

// parseComposition finds every `<param>.<RegisterMethod>(<expr>.Group("/x", ...))`
// call in the given file, resolving <param> to its declared parameter type
// within the enclosing function so the result is keyed by Go handler type,
// not by the local variable name used in that one file.
func parseComposition(file string) (registerPrefixes, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, file, nil, 0)
	if err != nil {
		return nil, fmt.Errorf("parse composition file %s: %w", file, err)
	}

	result := registerPrefixes{}

	for _, decl := range node.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}

		paramTypes := map[string]string{}
		if fn.Type.Params != nil {
			for _, p := range fn.Type.Params.List {
				typeName := identOrSelectorName(p.Type)
				if typeName == "" {
					continue
				}
				for _, n := range p.Names {
					paramTypes[n.Name] = typeName
				}
			}
		}

		for _, stmt := range fn.Body.List {
			collectRegisterCalls(stmt, paramTypes, result)
		}
	}

	return result, nil
}

// collectRegisterCalls walks a statement (and any nested blocks/closures,
// since the composition root wraps its wiring in a func literal passed to
// hexhttpx.New) looking for `<handlerVar>.Register*(<groupExpr>)` calls.
func collectRegisterCalls(stmt ast.Stmt, paramTypes map[string]string, out registerPrefixes) {
	ast.Inspect(stmt, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || !strings.HasPrefix(sel.Sel.Name, "Register") {
			return true
		}
		recv, ok := sel.X.(*ast.Ident)
		if !ok {
			return true
		}
		handlerType, ok := paramTypes[recv.Name]
		if !ok || len(call.Args) == 0 {
			return true
		}

		prefix := groupPrefixFromArg(call.Args[0])
		if out[handlerType] == nil {
			out[handlerType] = map[string]string{}
		}
		out[handlerType][sel.Sel.Name] = prefix
		return true
	})
}

// groupPrefixFromArg reads the string literal passed to `x.Group("/prefix", ...)`.
// Returns "" if the argument isn't a recognizable Group(...) call.
func groupPrefixFromArg(arg ast.Expr) string {
	call, ok := arg.(*ast.CallExpr)
	if !ok {
		return ""
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Group" || len(call.Args) == 0 {
		return ""
	}
	lit, ok := call.Args[0].(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return ""
	}
	return strings.Trim(lit.Value, "\"`")
}

func identOrSelectorName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		return t.Sel.Name
	case *ast.StarExpr:
		return identOrSelectorName(t.X)
	default:
		return ""
	}
}
