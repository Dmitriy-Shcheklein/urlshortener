// Package analyzer содержит статический анализатор, запрещающий вызовы panic()
// в любом пакете и ограничивающий log.Fatal и os.Exit функцией main пакета main.
package analyzer

import (
	"go/ast"
	"strings"

	"golang.org/x/tools/go/analysis"
)

var Analyzer = &analysis.Analyzer{
	Name: "customlinter",
	Doc:  "проверяет, что не используется panic, что log.Fatal и os.Exit вызываются только в main",
	Run:  run,
}

//nolint:gocognit
func run(pass *analysis.Pass) (any, error) {
	for _, file := range pass.Files {
		if strings.HasSuffix(pass.Fset.Position(file.Pos()).Filename, "_test.go") {
			continue
		}
		insideMain := false
		ast.Inspect(
			file, func(node ast.Node) bool {
				if fn, ok := node.(*ast.FuncDecl); ok {
					insideMain = pass.Pkg.Name() == "main" && fn.Name.Name == "main"
					return true
				}
				callExp, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				pName, fName := getFunctionName(pass, callExp)
				if fName == "panic" {
					pass.Reportf(callExp.Pos(), "найден вызов panic()")
				}
				if !insideMain {
					if pName == "log" && fName == "Fatal" {
						pass.Reportf(callExp.Pos(), "найден вызов log.Fatal")
					}
					if pName == "os" && fName == "Exit" {
						pass.Reportf(callExp.Pos(), "найден вызов os.Exit")
					}
				}
				return true
			},
		)
	}
	return nil, nil
}

func getFunctionName(pass *analysis.Pass, call *ast.CallExpr) (string, string) {

	switch f := call.Fun.(type) {
	case *ast.Ident:
		return "", f.Name
	case *ast.SelectorExpr:
		if _, ok := f.X.(*ast.Ident); ok {
			return pass.TypesInfo.Uses[f.Sel].Pkg().Name(), f.Sel.Name
		}
		return "", f.Sel.Name
	default:
		return "", ""
	}
}
