package cmd

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/Chronicle20/atlas/tools/packet-audit/internal/matrix"
)

// runSupportSummary implements `packet-audit support-summary` (FR-4.3): read the
// committed status.json and write one deterministic per-version support document
// under docs/packets/audits/support/<version>.md (totals, verified%, and a gap
// table split into n-a / unverified / conflict).
func runSupportSummary(args []string, stderr io.Writer) int {
	fs := flag.NewFlagSet("packet-audit support-summary", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var statusPath, outDir string
	fs.StringVar(&statusPath, "status", "docs/packets/audits/status.json", "status.json path")
	fs.StringVar(&outDir, "out", "docs/packets/audits/support", "output directory for per-version summaries")
	if err := fs.Parse(args); err != nil {
		return 3
	}

	m, err := matrix.LoadMatrix(statusPath)
	if err != nil {
		fmt.Fprintf(stderr, "packet-audit support-summary: %v\n", err)
		return 3
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		fmt.Fprintf(stderr, "packet-audit support-summary: %v\n", err)
		return 3
	}
	for _, vk := range matrix.VersionKeys {
		s := matrix.Summarize(m, vk)
		md := matrix.RenderSupportMarkdown(s)
		p := filepath.Join(outDir, vk+".md")
		if err := os.WriteFile(p, []byte(md), 0o644); err != nil {
			fmt.Fprintf(stderr, "packet-audit support-summary: %v\n", err)
			return 3
		}
		fmt.Fprintf(os.Stdout, "wrote %s\n", p)
	}
	return 0
}
