// Command topicmod applies the topicmod AST codemod to one or more
// directories, retyping Kafka topic-token constants and message-buffer
// keys from string to topic.Token.
//
// Usage:
//
//	topicmod [-check] <dir>...
//
// With -check, no files are written; the command reports residue findings
// and exits non-zero if any file would change or any residue was found.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/Chronicle20/atlas/tools/topicmod"
)

func main() {
	check := flag.Bool("check", false, "report changes and residue without writing files")
	flag.Parse()

	dirs := flag.Args()
	if len(dirs) == 0 {
		fmt.Fprintln(os.Stderr, "usage: topicmod [-check] <dir>...")
		os.Exit(2)
	}

	findings, err := topicmod.Run(dirs, *check)
	if err != nil {
		fmt.Fprintf(os.Stderr, "topicmod: %v\n", err)
		os.Exit(1)
	}

	for _, f := range findings {
		fmt.Printf("%s: [%s] %s\n", f.Pos, f.Rule, f.Reason)
	}

	if len(findings) > 0 {
		os.Exit(1)
	}
}
