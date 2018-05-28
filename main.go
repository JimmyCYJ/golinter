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
	"os"
	"path/filepath"
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
	rpts := make(LintReports, 0)
	pFilter := newPathFilter()
	ffl := newForbiddenFunctionList()
	for _, path := range args {
		if !filepath.IsAbs(path) {
			path, _ = filepath.Abs(path)
		}
		err := filepath.Walk(path, func(fpath string, info os.FileInfo, err error) error {
			if err != nil {
				reportErr(fmt.Sprintf("pervent panic by handling failure accessing a path %q: %v", fpath, err))
				return err
			}
			if ok, testType := pFilter.IsTestFile(fpath, info); ok {
				lt := newLinter(fpath, testType, &ffl)
				lt.Run()
				rpts = append(rpts, lt.LReport()...)
			}
			return nil
		})
		if err != nil {
			reportErr(fmt.Sprintf("error visiting the path %q: %v", path, err))
		}
	}
	reports := make([]string, 0)
	for _, r := range rpts {
		reports = append(reports, r.msg)
	}
	return reports
}

func reportErr(msg string) {
	fmt.Fprintln(os.Stderr, msg)
	exitCode = 2
}
