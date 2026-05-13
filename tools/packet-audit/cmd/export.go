package cmd

import (
	"fmt"
	"io"
)

func runExport(args []string, stderr io.Writer) int {
	fmt.Fprintln(stderr, "packet-audit export: requires --ida-source mcp with a configured MCP client")
	fmt.Fprintln(stderr, "(maintainer-only path; see README)")
	return 3
}
