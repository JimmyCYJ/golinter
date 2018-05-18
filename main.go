// To run this linter
// Option 1:
// go build main.go
// ./main ../counter/
// Option 2:
// go install .
// gometalinter --config=gometalinter.json ../counter/

package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"sort"
	"strings"
)

var exitCode int

func main() {
	flag.Parse()
	for _, r := range getReport(flag.Args()) {
		reportErr(r)
	}
	os.Exit(exitCode)
}

func getReport(args []string) []string {
	var reports []string
	if len(args) == 0 {
		reports = doAllDirs([]string{"."})
	} else {
		reports = doAllDirs(args)
	}
	return reports
}

func doAllDirs(args []string) []string {
	reports := make([]string, 0)
	for _, name := range args {
		// Is it a directory?
		if fi, err := os.Stat(name); err == nil && fi.IsDir() {
			for _, r := range doDir(name) {
				reports = append(reports, r.msg)
			}
		} else {
			reportErr(fmt.Sprintf("not a directory: %s", name))
		}
	}
	return reports
}

func doDir(name string) reports {
	testfiles := func(info os.FileInfo) bool {
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".go") &&
			strings.HasSuffix(info.Name(), "_test.go") {
			return true
		}
		return false
	}
	fs := token.NewFileSet()
	pkgs, err := parser.ParseDir(fs, name, testfiles, parser.Mode(0))
	if err != nil {
		reportErr(fmt.Sprintf("%v", err))
		return nil
	}
	rpts := make(reports, 0)
	for _, pkg := range pkgs {
		rpts = append(rpts, doPackage(fs, pkg)...)
	}
	sort.Sort(rpts)
	return rpts
}

func doPackage(fs *token.FileSet, pkg *ast.Package) reports {
	v := newVisitor(fs)
	for _, file := range pkg.Files {
		ast.Walk(&v, file)
	}
	return v.reports
}

func newVisitor(fs *token.FileSet) visitor {
	return visitor{
		fs: fs,
	}
}

type visitor struct {
	reports reports
	fs      *token.FileSet
}

/*
Validates the following for _test.go files
1. Disallow use of time.Sleep() in _test.go
2. Disallow use of testing.Short() in _test.go
*/
func (v *visitor) Visit(node ast.Node) ast.Visitor {
	if node == nil {
		return nil
	}

	ce, ok := node.(*ast.CallExpr)
	if ok {
		if isInvalidCall(ce.Fun, "time", "Sleep") {
			v.reports = append(v.reports, v.invalidCallReport(ce.Pos(), "time.Sleep()"))
		} else if isInvalidCall(ce.Fun, "testing", "Short") {
			v.reports = append(v.reports, v.invalidCallReport(ce.Pos(), "testing.Short()"))
		}
	}

	return v
}

func isInvalidCall(expr ast.Expr, pkgName, methodName string) bool {
	sel, ok := expr.(*ast.SelectorExpr)
	return ok && isIdent(sel.X, pkgName) && isIdent(sel.Sel, methodName)
}

func isIdent(expr ast.Expr, ident string) bool {
	id, ok := expr.(*ast.Ident)
	return ok && id.Name == ident
}

func (v *visitor) invalidCallReport(pos token.Pos, pkgMethodName string) report {
	return report{
		pos,
		fmt.Sprintf("%v:%v:%v:%s",
			v.fs.Position(pos).Filename,
			v.fs.Position(pos).Line,
			v.fs.Position(pos).Column,
			"invalid " + pkgMethodName + " call in unit tests."),
	}
}

type report struct {
	pos token.Pos
	msg string
}

type reports []report

func (l reports) Len() int           { return len(l) }
func (l reports) Less(i, j int) bool { return l[i].pos < l[j].pos }
func (l reports) Swap(i, j int)      { l[i], l[j] = l[j], l[i] }

func reportErr(msg string) {
	fmt.Fprintln(os.Stderr, msg)
	exitCode = 2
}
