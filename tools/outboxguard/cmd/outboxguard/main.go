package main

import (
	"github.com/Chronicle20/atlas/tools/outboxguard"
	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
	singlechecker.Main(outboxguard.Analyzer)
}
