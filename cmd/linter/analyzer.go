package main

import (
	"go/ast"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
	"strings"
)

const doc = `analyzer checks for forbidden function calls

This analyzer reports:
1. Usage of panic() function
2. Calls to log.Fatal() or os.Exit() outside main function of main package`

var Analyzer = &analysis.Analyzer{
	Name:     "paniclogexit",
	Doc:      doc,
	Run:      run,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
}

func run(pass *analysis.Pass) (interface{}, error) {
	inspector := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{
		(*ast.CallExpr)(nil),
	}

	inspector.Preorder(nodeFilter, func(node ast.Node) {
		callExpr := node.(*ast.CallExpr)

		// Проверка panic()
		if ident, ok := callExpr.Fun.(*ast.Ident); ok && ident.Name == "panic" {
			pass.Reportf(callExpr.Pos(), "panic() should not be used in production code called from panic")
			return
		}

		// Проверка log.Fatal() и os.Exit()
		if selExpr, ok := callExpr.Fun.(*ast.SelectorExpr); ok {
			if xIdent, ok := selExpr.X.(*ast.Ident); ok {
				pkgName := xIdent.Name
				funcName := selExpr.Sel.Name

				// Проверка на log.Fatal или log.Fatal*
				if pkgName == "log" && strings.HasPrefix(funcName, "Fatal") {
					if !isInMainFunction(pass, node) {
						pass.Reportf(
							callExpr.Pos(),
							"log.%s() should only be called from main function in main package",
							funcName,
						)
					}
					return
				}

				// Проверка на os.Exit()
				if pkgName == "os" && funcName == "Exit" {
					if !isInMainFunction(pass, node) {
						pass.Reportf(callExpr.Pos(), "os.Exit() should only be called from main function in main package called from os.Exit")
					}
					return
				}
			}
		}
	})

	return nil, nil
}

func isInMainFunction(pass *analysis.Pass, node ast.Node) bool {
	if pass.Pkg.Name() != "main" {
		return false
	}

	for _, file := range pass.Files {
		for _, decl := range file.Decls {
			if fn, ok := decl.(*ast.FuncDecl); ok && fn.Name.Name == "main" && fn.Body != nil {
				if node.Pos() >= fn.Body.Lbrace && node.Pos() <= fn.Body.Rbrace {
					return true
				}
			}
		}
	}
	return false
}
