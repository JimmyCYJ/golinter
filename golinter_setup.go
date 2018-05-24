package main

import (
	"os"
	"strings"
)

// TestType is type ID of tests
type TestType int

// All types of tests to parse.
const (
	UnitTest  TestType = iota // UnitTest == 0
	IntegTest TestType = iota // INTEGTEST == 1
	E2eTest   TestType = iota // E2ETEST == 2
)

// TestTypeString contains test types in string. The order should be in line with enum above.
var TestTypeString = []string{"unit test", "integration test", "e2e test"}

// TestFileFilters contains filters for parser. The order should be in line with enum above.
var TestFileFilters = []TestFileFilter{unitTestFileFilter, integTestFileFilter, e2eTestFileFilter}

// TestFileFilter defines filter function signature for parser.
type TestFileFilter func(os.FileInfo) bool

// unitTestFileFilter filters unit test files.
func unitTestFileFilter(info os.FileInfo) bool {
	if strings.HasSuffix(info.Name(), "_test.go") &&
			!strings.HasSuffix(info.Name(), "_integ_test.go") {
		return true
	}
	return false
}

// integTestFileFilter filters integration test files.
func integTestFileFilter(info os.FileInfo) bool {
	if strings.HasSuffix(info.Name(), "_integ_test.go") {
		return true
	}
	return false
}

// e2eTestFileFilter filters unit test files.
func e2eTestFileFilter(info os.FileInfo) bool {
	if strings.HasSuffix(info.Name(), "_test.go") {
		return true
	}
	return false
}
