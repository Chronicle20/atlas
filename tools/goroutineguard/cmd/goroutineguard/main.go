package main

import (
	"github.com/Chronicle20/atlas/tools/goroutineguard"
	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
	singlechecker.Main(goroutineguard.Analyzer)
}
