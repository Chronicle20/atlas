package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// gate-lint (task-169 FR-3.1a / T4.1b) flags the genuinely OFF-BY-ONE-PRONE
// client-version boundary comparisons on wire-encode paths in libs/atlas-packet
// — the exact shape of the documented `>83` footgun class (memory:
// bug_majorversion_gt83): a strict `MajorVersion() > N` (or its left-operand
// twin `N < MajorVersion()`) and an inclusive `MajorVersion() <= N` (twin
// `N >= MajorVersion()`). Those split the version axis at N/N+1, so the version
// ADJACENT to the boundary is silently re-bucketed the day a new column lands
// between the two straddling versions (exactly how `> 83` mis-grouped v84 with
// v87 once v84 was added). The recommended form is `MajorAtLeast(N)` / a named
// boundary helper.
//
// NARROWED in task-169 T4.1b: the CORRECT idioms `>= N` and `< N` (right form)
// and their twins `<= N` / `> N` (left form) are NOT flagged — they split at
// N-1/N and do not re-bucket version N. Phase 4a flagged all four operators and
// hit ~220 legitimate `>= N` sites (pure noise); the narrowed form drops those.
//
// Only the boundary constants {61,72,79,83,84,87,95} are considered, so
// base-version gates (`> 12`, `> 28`), Region() checks, and non-boundary
// constants are ignored. An inline `//gate-lint:allow <reason>` comment on the
// line suppresses a hit.
//
// REPORT-ONLY on the current tree: the ~35 narrowed hits are all task-113
// code-gate-audit VERIFIED-CORRECT gates whose boundary happens to sit between
// two adjacent version columns (e.g. `>87` == `>=95` today because no v88..v94
// exists). Making the check blocking would require an allow-annotation on each
// of those wire-source files — out of scope for a pure-tooling phase — so it
// stays report-only. Default mode prints the inventory and exits 0; `--check`
// exits non-zero when any hit is found, for targeted manual use and the
// fires-on-violation test.

// gateLintBoundaries are the client-version boundaries where MajorAtLeast(N) is
// the idiomatic gate. NOT a base-version gate (12) or the ancient 28 boundary.
var gateLintBoundaries = map[int]bool{61: true, 72: true, 79: true, 83: true, 84: true, 87: true, 95: true}

// gateLintConfig parameterises the scan so tests can point it at a fixture dir.
type gateLintConfig struct {
	Root  string // scanned root (default libs/atlas-packet)
	Check bool   // exit non-zero when any hit is found
}

func defaultGateLintConfig() gateLintConfig {
	return gateLintConfig{Root: filepath.Join("libs", "atlas-packet")}
}

// gateLintHit is one raw boundary comparison.
type gateLintHit struct {
	file     string
	line     int
	boundary int
	op       string
	text     string
}

// majorRightRe matches `MajorVersion() OP N` (helper call on the left).
var majorRightRe = regexp.MustCompile(`MajorVersion\(\)\s*(>=|<=|>|<)\s*(\d+)`)

// majorLeftRe matches `N OP ...MajorVersion()` (constant on the left).
var majorLeftRe = regexp.MustCompile(`(\d+)\s*(>=|<=|>|<)\s*[A-Za-z_.]*MajorVersion\(\)`)

func runGateLint(args []string, stderr io.Writer) int {
	cfg := defaultGateLintConfig()
	for _, a := range args {
		switch a {
		case "--check":
			cfg.Check = true
		case "-h", "--help", "help":
			fmt.Fprintln(stderr, "usage: packet-audit gate-lint [--check]")
			fmt.Fprintln(stderr, "flags raw MajorVersion() boundary comparisons that should use MajorAtLeast(N); report-only unless --check.")
			return 0
		default:
			if strings.HasPrefix(a, "--root=") {
				cfg.Root = strings.TrimPrefix(a, "--root=")
				continue
			}
			fmt.Fprintf(stderr, "packet-audit gate-lint: unexpected argument %q\n", a)
			return 3
		}
	}
	return gateLintRun(cfg, os.Stdout, stderr)
}

// gateLintRun is the testable core. Default (report) mode prints the inventory
// to out and exits 0; --check prints to stderr and exits 1 when any hit exists.
func gateLintRun(cfg gateLintConfig, out, stderr io.Writer) int {
	hits, err := collectGateLintHits(cfg)
	if err != nil {
		fmt.Fprintf(stderr, "packet-audit gate-lint: %v\n", err)
		return 3
	}
	w := out
	if cfg.Check {
		w = stderr
	}
	for _, h := range hits {
		fmt.Fprintf(w, "%s:%d\tMajorVersion() %s %d — prefer MajorAtLeast(%d): %s\n",
			h.file, h.line, h.op, h.boundary, atLeastArg(h), strings.TrimSpace(h.text))
	}
	if cfg.Check {
		if len(hits) > 0 {
			fmt.Fprintf(stderr, "packet-audit gate-lint: %d raw boundary comparison(s) found (add //gate-lint:allow <reason> to suppress a verified site).\n", len(hits))
			return 1
		}
		fmt.Fprintln(out, "gate-lint: no raw client-version boundary comparisons found.")
		return 0
	}
	fmt.Fprintf(out, "gate-lint: %d raw client-version boundary comparison(s) under %s (report-only).\n", len(hits), cfg.Root)
	return 0
}

// atLeastArg maps the operator+boundary to the equivalent MajorAtLeast argument
// where it is unambiguous (>= N → N; > N → N+1). For < / <= it just echoes the
// boundary (the caller negates), so the suggestion stays advisory.
func atLeastArg(h gateLintHit) int {
	if h.op == ">" {
		return h.boundary + 1
	}
	return h.boundary
}

// collectGateLintHits walks cfg.Root for non-test .go files and returns every
// raw boundary comparison, sorted by file then line.
func collectGateLintHits(cfg gateLintConfig) ([]gateLintHit, error) {
	var hits []gateLintHit
	err := filepath.WalkDir(cfg.Root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		b, rerr := os.ReadFile(path)
		if rerr != nil {
			return rerr
		}
		for i, ln := range strings.Split(string(b), "\n") {
			if strings.Contains(ln, "gate-lint:allow") {
				continue
			}
			for _, m := range majorRightRe.FindAllStringSubmatch(ln, -1) {
				if n, ok := boundaryFromMatch(m[2]); ok && rightFormFootgun(m[1]) {
					hits = append(hits, gateLintHit{file: filepath.ToSlash(path), line: i + 1, boundary: n, op: m[1], text: ln})
				}
			}
			for _, m := range majorLeftRe.FindAllStringSubmatch(ln, -1) {
				if n, ok := boundaryFromMatch(m[1]); ok && leftFormFootgun(m[2]) {
					hits = append(hits, gateLintHit{file: filepath.ToSlash(path), line: i + 1, boundary: n, op: m[2], text: ln})
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.SliceStable(hits, func(a, b int) bool {
		if hits[a].file != hits[b].file {
			return hits[a].file < hits[b].file
		}
		return hits[a].line < hits[b].line
	})
	return hits, nil
}

// rightFormFootgun reports whether `MajorVersion() OP N` is off-by-one-prone.
// `> N` and `<= N` split at N/N+1 (re-bucket the version adjacent to the
// boundary); `>= N` and `< N` split at N-1/N and are the correct idioms.
func rightFormFootgun(op string) bool {
	return op == ">" || op == "<="
}

// leftFormFootgun reports whether `N OP MajorVersion()` is off-by-one-prone.
// `N < Major` ≡ `Major > N` and `N >= Major` ≡ `Major <= N` are the footguns;
// `N <= Major` ≡ `Major >= N` and `N > Major` ≡ `Major < N` are correct.
func leftFormFootgun(op string) bool {
	return op == "<" || op == ">="
}

func boundaryFromMatch(s string) (int, bool) {
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, false
	}
	return n, gateLintBoundaries[n]
}
