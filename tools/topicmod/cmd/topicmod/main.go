// Command topicmod applies the topicmod AST codemod to one or more
// directories, retyping Kafka topic-token constants and message-buffer
// keys from string to topic.Token.
//
// Usage:
//
//	topicmod [-check] <dir>...
//	topicmod -fix-tests <module-dir>...
//
// With -check, no files are written; the command reports residue findings
// and exits non-zero if any file would change or any residue was found.
//
// With -fix-tests, each argument is treated as a Go module root: the
// command runs `go vet ./...` in it and applies AST edits for every
// diagnostic it can resolve, repeating until a round makes no further
// edits, then reports whatever diagnostics remain as residue.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/Chronicle20/atlas/tools/topicmod"
)

func main() {
	check := flag.Bool("check", false, "report changes and residue without writing files")
	fixTests := flag.Bool("fix-tests", false, "iteratively resolve `go vet` diagnostics in each argument (a Go module root)")
	flag.Parse()

	dirs := flag.Args()
	if len(dirs) == 0 {
		fmt.Fprintln(os.Stderr, "usage: topicmod [-check] <dir>...\n       topicmod -fix-tests <module-dir>...")
		os.Exit(2)
	}

	if *fixTests {
		var findings []topicmod.Finding
		for _, dir := range dirs {
			f, err := topicmod.FixModule(dir)
			if err != nil {
				fmt.Fprintf(os.Stderr, "topicmod -fix-tests %s: %v\n", dir, err)
				os.Exit(1)
			}
			findings = append(findings, f...)
		}
		for _, f := range findings {
			fmt.Printf("%s: [%s] %s\n", f.Pos, f.Rule, f.Reason)
		}
		if len(findings) > 0 {
			os.Exit(1)
		}
		return
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
