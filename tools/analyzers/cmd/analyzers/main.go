package main

import (
	"spicedb/tools/analyzers/closeafterusagecheck"
	"spicedb/tools/analyzers/exprstatementcheck"
	"spicedb/tools/analyzers/nilvaluecheck"
	"spicedb/tools/analyzers/paniccheck"
	"golang.org/x/tools/go/analysis/multichecker"
)

func main() {
	multichecker.Main(
		nilvaluecheck.Analyzer(),
		exprstatementcheck.Analyzer(),
		closeafterusagecheck.Analyzer(),
		paniccheck.Analyzer(),
	)
}
