// To run this linter
// Option 1:
// go build main.go control.go
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
	"log"
	"os"
	"path/filepath"
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
			err := filepath.Walk(name, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					reportErr(fmt.Sprintf("prevent panic by handling failure accessing a path %q: %v\n", path, err))
					return err
				}
				if info.IsDir() {
					if isE2eTestPath(path) {
						for _, r := range doE2eDir(path) {
							reports = append(reports, r.msg)
						}
					} else {
						for _, r := range doDir(path) {
							reports = append(reports, r.msg)
						}
					}
				}
				return nil
			})
			if err != nil {
				reportErr(fmt.Sprintf("error walking the path %q: %v", name, err))
			}
		} else {
			reportErr(fmt.Sprintf("not a directory: %s", name))
		}
	}
	return reports
}

// isE2eTestPath returns true if path is a sub directory in e2e test folders. e.g. e2e/foo/bar/...
func isE2eTestPath(path string) bool {
	subdirs := strings.Split(path, "/")
	for _, subdir := range subdirs {
		if subdir == "e2e" {
			return true
		}
	}
	return false
}

func doDir(path string) reports {
	rpts := testReportsByType(path, UnitTest)
	rpts = append(rpts, testReportsByType(path, IntegTest)...)
	return rpts
}

func doE2eDir(path string) reports {
	rpts := testReportsByType(path, E2eTest)
	return rpts
}

func testReportsByType(path string, testTypeID TestType) reports {
	fs := token.NewFileSet()
	pkgs, err := parser.ParseDir(fs, path, TestFileFilters[testTypeID], parser.Mode(0))
	if err != nil {
		reportErr(fmt.Sprintf("%v", err))
		return nil
	}
	rpts := make(reports, 0)
	for _, pkg := range pkgs {
		rpts = append(rpts, doPackage(fs, pkg, testTypeID)...)
	}
	sort.Sort(rpts)
	return rpts
}

func doPackage(fs *token.FileSet, pkg *ast.Package, testTypeID TestType) reports {
	v := newReportCollector(fs, testTypeID)
	switch testTypeID {
	case UnitTest:
		scanForbiddenFunctionCallInTest(&v, pkg)
	case IntegTest:
		scanMandatoryFunctionCallInTest(&v, pkg)
	case E2eTest:
		scanMandatoryFunctionCallInTest(&v, pkg)
	default:
		log.Printf("Test type is invalid %d", testTypeID)
	}
	return v.reports
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

func newReportCollector(fs *token.FileSet, testTypeID TestType) reportCollector {
	return reportCollector{
		fs:     fs,
		typeID: testTypeID,
	}
}

type reportCollector struct {
	reports reports
	fs      *token.FileSet
	typeID  TestType
}

/*
Validates the following for _test.go files
1. Disallow use of time.Sleep() in _test.go
2. Disallow use of testing.Short() in _test.go
*/
func (v *reportCollector) Visit(node ast.Node) ast.Visitor {
	if node == nil {
		return nil
	}

	ce, ok := node.(*ast.CallExpr)
	if ok {
		for _, utci := range UnitTestCheckList {
			if isForbiddenCall(ce.Fun, utci) {
				v.reports = append(v.reports, v.forbiddenCallReport(ce.Pos(), utci))
			}
		}
	}

	return v
}

func (v *reportCollector) forbiddenCallReport(pos token.Pos, utci UnitTestCheckItem) report {
	return report{
		pos,
		fmt.Sprintf("%v:%v:%v:%s",
			v.fs.Position(pos).Filename,
			v.fs.Position(pos).Line,
			v.fs.Position(pos).Column,
			"invalid "+utci.pkgName+"."+utci.mName+"() call in "+TestTypeString[v.typeID]),
	}
}

func (v *reportCollector) MissingMandatoryCallReport(pos token.Pos, testFunc *ast.FuncDecl) report {
	return report{
		pos,
		fmt.Sprintf("%v:%v:%v:%s %s %s",
			v.fs.Position(pos).Filename,
			v.fs.Position(testFunc.Pos()).Line,
			v.fs.Position(testFunc.Pos()).Column,
			"Missing testing.Short() call and t.Skip() call at the beginning of",
			TestTypeString[v.typeID], testFunc.Name.String()),
	}
}
