// Checks unit test files and look for forbidden function call in each test function.
// And checks if integration test files and e2e test files have mandatory function call
// at the beginning of each test function.

package main

import (
	"go/ast"
	"go/token"
)

// FuncCheckItem defines the information of a function that needs to check.
type UnitTestCheckItem struct {
	pkgName string // package name of the function call.
	mName   string // method name of the function call.
}

// UnitTestCheckList lists all functions calls should not exist in unit test files.
var UnitTestCheckList = []UnitTestCheckItem{
	UnitTestCheckItem{
		"time",
		"Sleep",
	},
	UnitTestCheckItem{
		"testing",
		"Short",
	},
}

// scanForbiddenFunctionCallInTest scans tests and checks forbidden function call.
func scanForbiddenFunctionCallInTest(v *reportCollector, pkg *ast.Package) {
	for _, file := range pkg.Files {
		ast.Walk(v, file)
	}
}

func isForbiddenCall(expr ast.Expr, utci UnitTestCheckItem) bool {
	sel, ok := expr.(*ast.SelectorExpr)
	return ok && isIdent(sel.X, utci.pkgName) && isIdent(sel.Sel, utci.mName)
}

func isIdent(expr ast.Expr, ident string) bool {
	id, ok := expr.(*ast.Ident)
	return ok && id.Name == ident
}

// scanMandatoryFunctionCallInTest scans tests and checks if mandatory function call is placed at
// the beginning of each test.
func scanMandatoryFunctionCallInTest(v *reportCollector, pkg *ast.Package) {
	for _, file := range pkg.Files {
		testFuncs := []*ast.FuncDecl{}
		for _, d := range file.Decls {
			if fn, isFn := d.(*ast.FuncDecl); isFn {
				testFuncs = append(testFuncs, fn)
			}
		}
		// Checks each test function named TestXxx.
		for _, function := range testFuncs {
			// log.Printf("-- function %s", function.Name.String())
			if !hasMandatoryCall(function.Body.List) {
				v.reports = append(v.reports, v.MissingMandatoryCallReport(file.Pos(), function))
			}
		}
	}
}

// hasMandatoryCall examines the mandatory function call in a function of the form TestXxx.
// Currently we check the following calls.
// case 1:
// func Testxxx(t *testing.T) {
// 	if testing.Short() {
//		t.Skip("xxx")
//	}
//	...
// }
// case 2:
// func Testxxx(t *testing.T) {
// 	if !testing.Short() {
// 	...
// 	}
// }
func hasMandatoryCall(stmts []ast.Stmt) bool {
	if len(stmts) == 0 {
		return false
	} else if len(stmts) == 1 {
		if ifStmt, ok := stmts[0].(*ast.IfStmt); ok {
			if uExpr, ok := ifStmt.Cond.(*ast.UnaryExpr); ok {
				if call, ok := uExpr.X.(*ast.CallExpr); ok && uExpr.Op == token.NOT {
					if fun, ok := call.Fun.(*ast.SelectorExpr); ok {
						if astid, ok := fun.X.(*ast.Ident); ok {
							return astid.String() == "testing" && fun.Sel.String() == "Short"
						}
					}
				}
			}
		}
	} else {
		hasShortAtTop := false
		hasSkipAtTop := false
		if ifStmt, ok := stmts[0].(*ast.IfStmt); ok {
			if call, ok := ifStmt.Cond.(*ast.CallExpr); ok {
				if fun, ok := call.Fun.(*ast.SelectorExpr); ok {
					if astid, ok := fun.X.(*ast.Ident); ok {
						if astid.String() == "testing" && fun.Sel.String() == "Short" {
							hasShortAtTop = true
						}
					}
				}
				if len(ifStmt.Body.List) > 0 {
					if exprStmt, ok := ifStmt.Body.List[0].(*ast.ExprStmt); ok {
						if call, ok := exprStmt.X.(*ast.CallExpr); ok {
							if fun, ok := call.Fun.(*ast.SelectorExpr); ok {
								if astid, ok := fun.X.(*ast.Ident); ok {
									if astid.String() == "t" && fun.Sel.String() == "Skip" {
										hasSkipAtTop = true
									}
								}
							}
						}
					}
				}
			}
		}
		return hasShortAtTop && hasSkipAtTop
	}
	return false
}
