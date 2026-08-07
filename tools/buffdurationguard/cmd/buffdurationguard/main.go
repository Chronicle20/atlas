package main

import (
	"golang.org/x/tools/go/analysis/singlechecker"

	"github.com/Chronicle20/atlas/tools/buffdurationguard"
)

func main() {
	singlechecker.Main(buffdurationguard.Analyzer)
}
