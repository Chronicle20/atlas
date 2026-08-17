package main

import (
	"golang.org/x/tools/go/analysis/singlechecker"

	"github.com/Chronicle20/atlas/tools/producerseamguard"
)

func main() {
	singlechecker.Main(producerseamguard.Analyzer)
}
