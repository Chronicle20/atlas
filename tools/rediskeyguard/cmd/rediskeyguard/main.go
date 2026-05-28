package main

import (
	"github.com/Chronicle20/atlas/tools/rediskeyguard"
	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
	singlechecker.Main(rediskeyguard.Analyzer)
}
