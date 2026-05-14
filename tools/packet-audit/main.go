package main

import (
	"os"

	"github.com/Chronicle20/atlas/tools/packet-audit/cmd"
)

func main() {
	os.Exit(cmd.Run(os.Args[1:], os.Stderr))
}
