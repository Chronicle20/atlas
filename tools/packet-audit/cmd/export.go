package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"time"

	csvpkg "github.com/Chronicle20/atlas/tools/packet-audit/internal/csv"
	"github.com/Chronicle20/atlas/tools/packet-audit/internal/idasrc"
)

// exportOpts is the injectable configuration for the export driver. Every value
// affecting the output bytes is fixed up front (notably GeneratedAt — there is
// NO time.Now in the core), so re-running with identical opts+client produces
// byte-identical output.
type exportOpts struct {
	Version      string
	Output       string
	PriorExport  string // path to docs/packets/ida-exports/<version>.json
	Pending      string // path to _pending.md (OPTIONAL — skip source (c) if empty or file missing)
	IDAURL       string
	IDATimeout   time.Duration
	DescentDepth int
	GeneratedAt  string // FIXED provenance timestamp (NO time.Now in the core)
}

// fnameToken matches an FName-looking Class::Method token (used to scrape
// candidate roster entries out of _pending.md prose).
var fnameToken = regexp.MustCompile(`[A-Z][A-Za-z0-9]+::[A-Za-z0-9_]+`)

// exportRun is the injectable core: it builds the roster, harvests via the given
// client, backfills direction, marshals deterministically, writes Output, and
// prints the unresolved summary to stderr. Returns a process exit code.
func exportRun(opts exportOpts, client idasrc.MCPClient, stdout, stderr io.Writer) int {
	roster := buildRoster(opts)
	if len(roster) == 0 {
		fmt.Fprintln(stderr, "export: empty roster — nothing to harvest (check --version / prior export)")
		return 3
	}

	// Direction source (shared by the harvest DirOf closure and the post-harvest
	// backfill): prior export's direction wins; else first candidatesFromFName
	// candidate's direction; else default clientbound. directionFor returns the
	// string form ("" when neither source knows the name); dirOf adapts it to the
	// idasrc.Direction the parser needs (default clientbound).
	prior := priorDirections(opts.PriorExport)
	directionFor := func(name string) string {
		if d, ok := prior[name]; ok && d != "" {
			return d
		}
		if cands := candidatesFromFName(name); len(cands) > 0 {
			return directionString(cands[0].dir)
		}
		return ""
	}
	dirOf := func(name string) idasrc.Direction {
		if directionFor(name) == "serverbound" {
			return idasrc.DirServerbound
		}
		return idasrc.DirClientbound
	}

	ef, err := idasrc.Harvest(context.Background(), client, roster, idasrc.HarvestOpts{
		DescentDepth: opts.DescentDepth,
		GeneratedAt:  opts.GeneratedAt,
		DirOf:        dirOf,
	})
	if err != nil {
		fmt.Fprintln(stderr, "export: harvest:", err)
		return 3
	}

	// Direction backfill: prior export's direction wins; else first
	// candidatesFromFName candidate's direction; else leave empty. Reuses the
	// same directionFor source the harvest DirOf closure used.
	for name := range ef.Functions {
		fn := ef.Functions[name]
		if fn.Direction != "" {
			continue
		}
		if d := directionFor(name); d != "" {
			fn.Direction = d
			ef.Functions[name] = fn
		}
	}

	// Deterministic marshal: json.MarshalIndent sorts map[string] keys, and the
	// harvested Calls preserve parser source-order. Single trailing newline.
	rosterSet := map[string]bool{}
	for _, r := range roster {
		rosterSet[r] = true
	}
	b, err := json.MarshalIndent(ef, "", "  ")
	if err != nil {
		fmt.Fprintln(stderr, "export: marshal:", err)
		return 3
	}
	b = append(b, '\n')

	if dir := filepath.Dir(opts.Output); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			fmt.Fprintln(stderr, "export: mkdir:", err)
			return 3
		}
	}
	if err := os.WriteFile(opts.Output, b, 0o644); err != nil {
		fmt.Fprintln(stderr, "export: write:", err)
		return 3
	}

	// Unresolved summary to stderr. resolved = a roster function with calls and
	// not Unresolved; unresolved = fn.Unresolved OR any call Op=="Unresolved";
	// descended-helper = a harvested function NOT in the roster (discovered via
	// Delegate descent). Done inline because ef's type (idasrc.exportFile) is
	// unexported and cannot be named in a helper's signature.
	var resolved, descended, unresolved int
	var unresolvedNames []string
	for name := range ef.Functions {
		fn := ef.Functions[name]
		bad := fn.Unresolved
		for _, c := range fn.Calls {
			if c.Op == "Unresolved" {
				bad = true
				break
			}
		}
		switch {
		case bad:
			unresolved++
			unresolvedNames = append(unresolvedNames, name)
		case !rosterSet[name]:
			descended++
		default:
			resolved++
		}
	}
	sort.Strings(unresolvedNames)
	fmt.Fprintf(stderr, "export: %d resolved, %d descended-helper, %d unresolved\n",
		resolved, descended, unresolved)
	for _, n := range unresolvedNames {
		fmt.Fprintln(stderr, "  unresolved:", n)
	}
	return 0
}

// buildRoster computes the sorted, de-duplicated union of:
//
//	(a) the prior export's FName keys (PRIMARY source).
//	(c) FName-looking tokens scraped from _pending.md, if present.
//
// Source (b) — candidatesFromFName — is NOT separately enumerable here: it is a
// switch, not a list. In practice it is subsumed by (a), since the audit only
// ever calls candidatesFromFName on export FNames, so every (b) input already
// arrives via the prior export's keys.
func buildRoster(opts exportOpts) []string {
	set := map[string]bool{}
	for _, fn := range idaExportFunctions(opts.PriorExport) {
		set[fn] = true
	}
	for _, fn := range pendingFNames(opts.Pending) {
		set[fn] = true
	}
	out := make([]string, 0, len(set))
	for fn := range set {
		out = append(out, fn)
	}
	sort.Strings(out)
	return out
}

// pendingFNames scrapes FName-looking tokens out of _pending.md. Returns nil if
// the path is empty or the file is missing (source (c) is optional). Obvious
// non-target decode/encode helpers are filtered out.
func pendingFNames(path string) []string {
	if path == "" {
		return nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	skip := map[string]bool{
		"CInPacket::DecodeN":  true,
		"COutPacket::EncodeN": true,
	}
	var out []string
	for _, tok := range fnameToken.FindAllString(string(b), -1) {
		if skip[tok] {
			continue
		}
		out = append(out, tok)
	}
	return out
}

// priorDirections reads the prior export and returns FName→direction. Missing
// or unparseable file yields an empty map (backfill simply finds nothing).
func priorDirections(path string) map[string]string {
	out := map[string]string{}
	if path == "" {
		return out
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return out
	}
	var doc struct {
		Functions map[string]struct {
			Direction string `json:"direction"`
		} `json:"functions"`
	}
	if err := json.Unmarshal(b, &doc); err != nil {
		return out
	}
	for name, fn := range doc.Functions {
		out[name] = fn.Direction
	}
	return out
}

func directionString(d csvpkg.Direction) string {
	if d == csvpkg.DirServerbound {
		return "serverbound"
	}
	return "clientbound"
}
