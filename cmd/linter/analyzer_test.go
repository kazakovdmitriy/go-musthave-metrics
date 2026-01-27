package main

import (
	"golang.org/x/tools/go/analysis/analysistest"
	"testing"
)

func TestAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, Analyzer, "panic", "main_in_main", "main_outside_main", "nonmain_in_main", "os_exit", "log_fatalf")
}
