package cmd

import (
	"flag"
	"fmt"
	"io"

	"github.com/Chronicle20/atlas/tools/packet-audit/internal/matrix"
)

// runStatus implements `packet-audit status <version>` (FR-4.4): a read-only
// query over status.json that prints the version's summary, open-gap list, and
// any stale-evidence rows to stdout. Writes nothing.
func runStatus(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("packet-audit status", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var statusPath string
	fs.StringVar(&statusPath, "status", "docs/packets/audits/status.json", "status.json path")
	if err := fs.Parse(args); err != nil {
		return 3
	}
	rest := fs.Args()
	if len(rest) != 1 {
		fmt.Fprintln(stderr, "packet-audit status: usage: status [--status <path>] <version>")
		return 3
	}
	return statusRun(statusPath, rest[0], stdout, stderr)
}

// statusRun is the testable core: load status.json, aggregate the version, and
// print. Returns 3 on load error or unknown version, 0 otherwise.
func statusRun(statusPath, version string, stdout, stderr io.Writer) int {
	m, err := matrix.LoadMatrix(statusPath)
	if err != nil {
		fmt.Fprintf(stderr, "packet-audit status: %v\n", err)
		return 3
	}
	known := false
	for _, vk := range matrix.VersionKeys {
		if vk == version {
			known = true
			break
		}
	}
	if !known {
		fmt.Fprintf(stderr, "packet-audit status: unknown version %q (known: %v)\n", version, matrix.VersionKeys)
		return 3
	}

	s := matrix.Summarize(m, version)
	fmt.Fprintf(stdout, "packet coverage — %s\n", version)
	fmt.Fprintf(stdout, "  verified   %d\n", s.Verified)
	fmt.Fprintf(stdout, "  family     %d\n", s.Family)
	fmt.Fprintf(stdout, "  partial    %d\n", s.Partial)
	fmt.Fprintf(stdout, "  incomplete %d\n", s.Incomplete)
	fmt.Fprintf(stdout, "  n-a        %d\n", s.NACount)
	fmt.Fprintf(stdout, "  conflict   %d\n", s.Conflict)
	fmt.Fprintf(stdout, "  verified%%  %.1f%% (%d of %d applicable)\n\n", s.VerifiedPct, s.Verified, s.Total)

	printGaps(stdout, fmt.Sprintf("conflicts (%d):", len(s.Conflicts)), s.Conflicts)
	printGaps(stdout, fmt.Sprintf("open gaps (%d):", len(s.Unverified)), s.Unverified)
	printGaps(stdout, fmt.Sprintf("stale evidence (%d):", len(s.Stale)), s.Stale)
	return 0
}

func printGaps(w io.Writer, title string, rows []matrix.GapRow) {
	fmt.Fprintln(w, title)
	if len(rows) == 0 {
		fmt.Fprintln(w, "  none")
		fmt.Fprintln(w)
		return
	}
	for _, r := range rows {
		name := r.Op
		if name == "" {
			name = r.Packet
		}
		op := ""
		if r.Opcode >= 0 {
			op = fmt.Sprintf(" %#03x", r.Opcode)
		}
		note := ""
		if r.Note != "" {
			note = " — " + r.Note
		}
		fmt.Fprintf(w, "  [%s]%s %s%s\n", r.State, op, name, note)
	}
	fmt.Fprintln(w)
}
