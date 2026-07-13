package cmd

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Chronicle20/atlas/tools/packet-audit/internal/atlaspacket"
	"github.com/Chronicle20/atlas/tools/packet-audit/internal/marker"
	"github.com/Chronicle20/atlas/tools/packet-audit/internal/matrix"
	"github.com/Chronicle20/atlas/tools/packet-audit/internal/opregistry"
	"github.com/Chronicle20/atlas/tools/packet-audit/internal/template"
)

// exitRuntime is the exit code for runtime errors (missing flags parsed, I/O errors, etc.).
// exitBlocker is the exit code for --check stale/conflict failures.
const (
	exitRuntime = 3
	exitBlocker = 1
)

type matrixOpts struct {
	RegistryDir  string
	AuditsDir    string
	TemplatesDir string
	ExportsDir   string
	EvidenceDir  string // consumed from Phase 2 on; empty = no evidence
	TiersFile    string // consumed from Phase 2 on; defaults to docs/packets/evidence/tiers.yaml
	FamiliesFile string // mode-prefix dispatcher membership; defaults to docs/packets/evidence/families.yaml
	PacketLibDir string // consumed from Phase 3 on (marker scan); empty = no markers
	Versions     []string
	OutDir       string
	Check        bool
}

func runMatrix(args []string, stderr io.Writer) int {
	fs := flag.NewFlagSet("packet-audit matrix", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var o matrixOpts
	var versionsCSV string
	fs.StringVar(&o.RegistryDir, "registry-dir", "docs/packets/registry", "registry YAML dir")
	fs.StringVar(&o.AuditsDir, "audits-dir", "docs/packets/audits", "audit reports parent dir")
	fs.StringVar(&o.TemplatesDir, "templates-dir", "services/atlas-configurations/seed-data/templates", "tenant seed templates dir")
	fs.StringVar(&o.ExportsDir, "exports-dir", "docs/packets/ida-exports", "IDA export JSON dir")
	fs.StringVar(&o.EvidenceDir, "evidence-dir", "docs/packets/evidence", "evidence ledger dir")
	fs.StringVar(&o.TiersFile, "tiers", "docs/packets/evidence/tiers.yaml", "tier-1 membership YAML")
	fs.StringVar(&o.FamiliesFile, "families", "docs/packets/evidence/families.yaml", "mode-prefix dispatcher membership YAML")
	fs.StringVar(&o.PacketLibDir, "packet-lib", "libs/atlas-packet", "atlas-packet root for marker scanning")
	fs.StringVar(&versionsCSV, "versions", strings.Join(matrix.VersionKeys, ","), "comma-separated version keys")
	fs.StringVar(&o.OutDir, "out-dir", "docs/packets/audits", "output dir for STATUS.md/status.json")
	fs.BoolVar(&o.Check, "check", false, "CI mode: verify committed outputs are current; fail on conflicts/drift")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 3
	}
	// TrimSpace each entry and drop empties (handles "v83, v84" or trailing commas).
	raw := strings.Split(versionsCSV, ",")
	for _, v := range raw {
		if s := strings.TrimSpace(v); s != "" {
			o.Versions = append(o.Versions, s)
		}
	}
	return matrixRun(o, os.Stdout, stderr)
}

func matrixRun(o matrixOpts, stdout, stderr io.Writer) int {
	reg, err := opregistry.LoadDir(o.RegistryDir, o.Versions)
	if err != nil {
		fmt.Fprintf(stderr, "packet-audit matrix: %v\n", err)
		return exitRuntime
	}
	in := matrix.Inputs{Registry: reg,
		Reports:     map[string]map[string]matrix.LoadedReport{},
		Routed:      map[string]map[matrix.RouteKey]bool{},
		RoutedNames: map[string]map[matrix.RouteKey]string{},
		Evidence:    map[matrix.EvKey]matrix.EvidenceStatus{},
		Tier1:       map[string]bool{},
		Markers:     map[matrix.EvKey]matrix.MarkerStatus{},
		Families:    map[string]bool{},
	}
	// Mode-prefix dispatcher membership: caps these ops at 🧩 family so a single
	// sub-handler's fixture can't present the whole dispatcher as ✅ verified.
	families, err := matrix.LoadFamilies(o.FamiliesFile)
	if err != nil {
		fmt.Fprintf(stderr, "packet-audit matrix: families: %v\n", err)
		return exitRuntime
	}
	in.Families = families.Set()
	hashes := map[string]string{}
	exportPaths := map[string]string{}
	for _, vk := range o.Versions {
		reps, err := matrix.LoadReports(filepath.Join(o.AuditsDir, vk))
		if err != nil {
			fmt.Fprintf(stderr, "packet-audit matrix: %v\n", err)
			return exitRuntime
		}
		in.Reports[vk] = reps
		in.Routed[vk] = map[matrix.RouteKey]bool{}
		tp := templatePathIn(o.TemplatesDir, vk)
		if t, err := template.Load(tp); os.IsNotExist(err) {
			// Missing template is a warning: grading continues without routing data.
			fmt.Fprintf(stderr, "packet-audit matrix: warning: no template for %s (%v)\n", vk, err)
		} else if err != nil {
			// Other errors (permission denied, corrupt JSON, etc.) are fatal.
			fmt.Fprintf(stderr, "packet-audit matrix: error loading template for %s: %v\n", vk, err)
			return exitRuntime
		} else {
			in.RoutedNames[vk] = map[matrix.RouteKey]string{}
			for op, name := range t.Writers() {
				k := matrix.RouteKey{Opcode: op, Dir: opregistry.DirClientbound}
				in.Routed[vk][k] = true
				in.RoutedNames[vk][k] = name
			}
			for op, name := range t.Handlers() {
				k := matrix.RouteKey{Opcode: op, Dir: opregistry.DirServerbound}
				in.Routed[vk][k] = true
				in.RoutedNames[vk][k] = name
			}
		}
		ep := exportPathIn(o.ExportsDir, vk)
		if raw, err := os.ReadFile(ep); os.IsNotExist(err) {
			// Missing export file: warn but continue (not all versions have exports yet).
			fmt.Fprintf(stderr, "packet-audit matrix: warning: no export file for %s (%s)\n", vk, ep)
		} else if err != nil {
			// Unreadable-but-existing export is a hard failure.
			fmt.Fprintf(stderr, "packet-audit matrix: error reading export for %s: %v\n", vk, err)
			return exitRuntime
		} else {
			hashes[vk] = fmt.Sprintf("%x", sha256.Sum256(raw))
			exportPaths[vk] = ep
		}
	}

	evStatus, evProblems, err := matrix.BuildEvidenceInputs(o.EvidenceDir, exportPaths)
	if err != nil {
		fmt.Fprintf(stderr, "packet-audit matrix: %v\n", err)
		return exitRuntime
	}
	in.Evidence = evStatus
	// Design §13: an evidence record for a (packet, version) with no audit
	// report is dangling — a --check failure. EXCEPTION (commit 6c202cb7): when
	// the version's registry declares the packet via an op's `packet:` field, the
	// evidence record backs the no-report byte-fixture promotion path (grading
	// promotes such a cell on a fresh marker + evidence, no report needed), so it
	// is not dangling. Iterate sorted so stderr output is deterministic.
	evKeys := make([]matrix.EvKey, 0, len(evStatus))
	for k := range evStatus {
		evKeys = append(evKeys, k)
	}
	sort.Slice(evKeys, func(i, j int) bool {
		if evKeys[i].Packet != evKeys[j].Packet {
			return evKeys[i].Packet < evKeys[j].Packet
		}
		return evKeys[i].Version < evKeys[j].Version
	})
	for _, k := range evKeys {
		if _, ok := reportForPacket(in.Reports[k.Version], k.Packet); ok {
			continue
		}
		if registryDeclaresPacket(in.Registry, k.Version, k.Packet) {
			continue
		}
		evProblems = append(evProblems,
			fmt.Sprintf("dangling evidence: %s × %s has no audit report", k.Packet, k.Version))
	}

	tiers, err := matrix.LoadTiers(o.TiersFile)
	if err != nil {
		fmt.Fprintf(stderr, "packet-audit matrix: %v\n", err)
		return exitRuntime
	}

	// Build TypeRegistry once for opaque-type recursion (Task 3.2).
	// A missing or empty PacketLibDir is tolerated — NewTypeRegistry returns an
	// empty registry rather than an error for a non-existent root.
	var typeReg *atlaspacket.TypeRegistry
	if o.PacketLibDir != "" {
		reg, regErr := atlaspacket.NewTypeRegistry(o.PacketLibDir)
		if regErr == nil {
			typeReg = reg
		} else {
			// typeReg stays nil; IsTier1 runs with nil recurseTypes, so
			// opaque-type tier expansion is skipped — warn so a parse failure
			// can't silently narrow tier-1 membership.
			fmt.Fprintf(stderr, "packet-audit matrix: warning: packet-lib analysis unavailable (%v) — opaque-type tier expansion skipped\n", regErr)
		}
	}

	// Populate in.Tier1 for every loaded report's packet id via tiers.IsTier1.
	// Pass the transitive RecurseType set derived from the TypeRegistry so that
	// opaque_types entries in tiers.yaml expand correctly to their consumer packets.
	for _, vk := range o.Versions {
		for _, r := range in.Reports[vk] {
			pkt := matrix.PacketID(r)
			recurseTypes := transitiveRecurseTypes(typeReg, pkt)
			if tiers.IsTier1(pkt, recurseTypes) {
				in.Tier1[pkt] = true
			}
		}
	}

	// Scan marker comments in atlas-packet test files (Task 3.2).
	var checkProblems []string
	checkProblems = append(checkProblems, evProblems...)
	if o.PacketLibDir != "" {
		if _, statErr := os.Stat(o.PacketLibDir); !os.IsNotExist(statErr) {
			// PacketLibDir exists: scan for markers. A non-existent dir means no
			// markers yet (Phase 1/2 runs, CI before atlas-packet has any markers).
			markers, markerErrs, markerErr := marker.Scan(o.PacketLibDir)
			if markerErr != nil {
				fmt.Fprintf(stderr, "packet-audit matrix: marker scan: %v\n", markerErr)
				return exitRuntime
			}
			checkProblems = append(checkProblems, markerErrs...)
			for _, mk := range markers {
				k := matrix.EvKey{Packet: mk.Packet, Version: mk.Version}
				ok := false
				if ev, has := evStatus[k]; has && ev.Address == mk.Address {
					ok = true
				}
				if rep, has := reportForPacket(in.Reports[mk.Version], mk.Packet); has && rep.Address == mk.Address {
					ok = true
				}
				if !ok {
					checkProblems = append(checkProblems,
						fmt.Sprintf("orphan marker %s:%d — %s × %s ida=%s matches no evidence record or audit report",
							mk.File, mk.Line, mk.Packet, mk.Version, mk.Address))
					continue
				}
				in.Markers[k] = matrix.MarkerStatus{Found: true, Address: mk.Address}
			}
		}
	}

	// Resolve per-version sub-struct dispositions from each version's
	// _unimplemented.json (FR-4.1, task-169). A ref names a sub-struct by an
	// explicit `packet` path or a suffix-qualified fname; bare-base-fname
	// dispatcher-arm dispositions are not sub-struct rows and are skipped.
	idaIndex := matrix.BuildIDANameIndex(in.Reports)
	in.Unimplemented = map[string]map[string]bool{}
	for _, vk := range o.Versions {
		refs, uerr := matrix.LoadUnimplemented(filepath.Join(o.AuditsDir, vk, "_unimplemented.json"))
		if uerr != nil {
			fmt.Fprintf(stderr, "packet-audit matrix: error loading _unimplemented.json for %s: %v\n", vk, uerr)
			return exitRuntime
		}
		in.Unimplemented[vk] = matrix.ResolveUnimplemented(refs, idaIndex)
	}

	m := matrix.Build(in, o.Versions)
	m.ExportHashes = hashes
	m.ToolSHA = toolTreeSHA()

	md := matrix.RenderMarkdown(m, o.Versions)
	js, err := matrix.RenderJSON(m)
	if err != nil {
		fmt.Fprintf(stderr, "packet-audit matrix: %v\n", err)
		return exitRuntime
	}
	mdPath := filepath.Join(o.OutDir, "STATUS.md")
	jsPath := filepath.Join(o.OutDir, "status.json")

	if o.Check {
		return matrixCheck(m, md, js, mdPath, jsPath, checkProblems, o.Versions, stderr)
	}
	if err := os.MkdirAll(o.OutDir, 0o755); err != nil {
		fmt.Fprintf(stderr, "packet-audit matrix: %v\n", err)
		return exitRuntime
	}
	if err := os.WriteFile(mdPath, []byte(md), 0o644); err != nil {
		fmt.Fprintf(stderr, "packet-audit matrix: %v\n", err)
		return exitRuntime
	}
	if err := os.WriteFile(jsPath, js, 0o644); err != nil {
		fmt.Fprintf(stderr, "packet-audit matrix: %v\n", err)
		return exitRuntime
	}
	fmt.Fprintf(stdout, "wrote %s and %s\n", mdPath, jsPath)
	return 0
}

// matrixCheck implements the full --check semantics (design §10.1):
// fails on stale committed outputs, problems (drift/dangling/orphan), and
// any conflict cell (conflicts are blockers, never allowlisted).
// versionKeys fixes the iteration order for conflict messages so output is
// deterministic regardless of map iteration order.
func matrixCheck(m matrix.Matrix, md string, js []byte, mdPath, jsPath string, problems []string, versionKeys []string, stderr io.Writer) int {
	fail := false
	for _, p := range problems {
		fmt.Fprintf(stderr, "matrix --check: %s\n", p)
		fail = true
	}
	// Use sorted version keys for deterministic conflict output.
	vks := versionKeys
	if len(vks) == 0 {
		// Fallback: derive from rows (covers callers without explicit keys).
		seen := map[string]bool{}
		for _, r := range m.Rows {
			for vk := range r.Cells {
				seen[vk] = true
			}
		}
		for vk := range seen {
			vks = append(vks, vk)
		}
		sort.Strings(vks)
	}
	for _, r := range m.Rows {
		for _, vk := range vks {
			c, ok := r.Cells[vk]
			if !ok {
				continue
			}
			if c.State == matrix.StateConflict {
				name := r.Op
				if name == "" {
					name = r.Packet
				}
				fmt.Fprintf(stderr, "matrix --check: conflict %s × %s — %s\n", name, vk, c.Note)
				fail = true
			}
		}
	}
	if cur, err := os.ReadFile(mdPath); err != nil || string(cur) != md {
		fmt.Fprintf(stderr, "matrix --check: %s is stale — regenerate and commit\n", mdPath)
		fail = true
	}
	if cur, err := os.ReadFile(jsPath); err != nil || string(cur) != string(js) {
		fmt.Fprintf(stderr, "matrix --check: %s is stale\n", jsPath)
		fail = true
	}
	if fail {
		return exitBlocker
	}
	return 0
}

// reportForPacket finds the LoadedReport (if any) for a given packet id within
// a version's report map. It checks each report using PacketID for normalization.
// Task 3.2 reuses this helper.
func reportForPacket(reps map[string]matrix.LoadedReport, pkt string) (matrix.LoadedReport, bool) {
	for _, r := range reps {
		if matrix.PacketID(r) == pkt {
			return r, true
		}
	}
	return matrix.LoadedReport{}, false
}

// registryDeclaresPacket reports whether the given version's registry has any op
// whose `packet:` field equals pkt. Such an op is the no-report byte-fixture
// promotion path (commit 6c202cb7): its evidence record is intentionally
// report-less and must not be flagged as dangling by --check.
func registryDeclaresPacket(reg opregistry.Registry, version, pkt string) bool {
	vf, ok := reg.Versions[version]
	if !ok || vf == nil {
		return false
	}
	for _, e := range vf.Entries {
		if e.Packet == pkt {
			return true
		}
	}
	return false
}

// transitiveRecurseTypes returns the set of qualified type names reachable via
// KindRecurse calls from the packet's root struct. Used by IsTier1 to expand
// opaque_types tier membership to their consumer packets (Task 3.2).
//
// packetID has the form "pkgdir/WriterName" (e.g. "monster/clientbound/MonsterStatSet").
// The TypeRegistry qualifies structs as "pkgdir.StructName" (e.g.
// "monster/clientbound.StatSet"). We split on the last "/" to derive
// pkgPath + writerName, then look up the qualified struct key via
// TypeForWriter (which resolves the Operation() → const → literal chain).
// If TypeForWriter misses (e.g. the writer name equals the struct name, as in
// most clientbound social packets), we fall back to treating the WriterName as
// the struct name directly.
//
// If typeReg is nil (PacketLibDir empty or unreadable), returns nil — IsTier1
// falls back to prefix/explicit-packet matching only.
func transitiveRecurseTypes(typeReg *atlaspacket.TypeRegistry, packetID string) []string {
	if typeReg == nil || packetID == "" {
		return nil
	}
	i := strings.LastIndex(packetID, "/")
	if i < 0 {
		return nil
	}
	pkgPath := packetID[:i]
	writerName := packetID[i+1:]

	// Primary lookup: TypeForWriter resolves WriterName → qualified struct key
	// via the Pass-1.5 Operation()-const index.
	qualKey, ok := typeReg.TypeForWriter(pkgPath, writerName)
	if !ok {
		// Fallback: treat writerName as the struct name (covers packets where
		// WriterName == struct name, e.g. buddy/clientbound/Invite).
		qualKey = pkgPath + "." + writerName
	}

	// Walk the call tree transitively, collecting all RecurseType names.
	seen := map[string]bool{}
	var walk func(key string)
	walk = func(key string) {
		if seen[key] {
			return
		}
		seen[key] = true
		calls, ok := typeReg.Calls(key)
		if !ok {
			return
		}
		for _, c := range calls {
			if c.Kind == atlaspacket.KindRecurse && c.RecurseType != "" {
				walk(c.RecurseType)
			}
		}
	}
	walk(qualKey)
	// Return all names except the root itself.
	delete(seen, qualKey)
	var out []string
	for k := range seen {
		out = append(out, k)
	}
	// Also return short (unqualified) names so tiers.yaml short names match.
	extra := make([]string, 0, len(out))
	for _, k := range out {
		if j := strings.LastIndex(k, "."); j >= 0 {
			short := k[j+1:]
			// Strip ::EncodeForeign suffix for matching.
			short = strings.TrimSuffix(short, "::EncodeForeign")
			extra = append(extra, short)
		}
	}
	out = append(out, extra...)
	return out
}

func templatePathIn(dir, vk string) string {
	return filepath.Join(dir, filepath.Base(matrix.TemplatePath(vk)))
}

func exportPathIn(dir, vk string) string {
	return filepath.Join(dir, filepath.Base(matrix.ExportPath(vk)))
}

// toolTreeSHA returns a deterministic SHA over the tool's committed Go sources
// only (excluding README/docs/testdata), or "unknown" outside a git checkout.
// Hashing only .go blobs means editing the tool's docs never invalidates the
// matrix (task-169 T2.0 — closes the README churn trap).
func toolTreeSHA() string {
	out, err := exec.Command("git", "ls-tree", "-r", "HEAD", "tools/packet-audit").Output()
	if err != nil {
		return "unknown"
	}
	return hashGoTreeEntries(string(out))
}

// isToolTestdataPath reports whether a repo-relative path lies under a
// `testdata` directory (whose .go fixtures are excluded from the ToolSHA).
func isToolTestdataPath(p string) bool {
	for _, seg := range strings.Split(p, "/") {
		if seg == "testdata" {
			return true
		}
	}
	return false
}

// hashGoTreeEntries parses `git ls-tree -r` output and returns a sha256 over
// only the tool's Go source blobs — non-.go files (README, docs) and testdata
// fixtures are excluded. Entries are sorted by path so the result is
// order-stable and deterministic regardless of git's line order.
func hashGoTreeEntries(lsTree string) string {
	type ent struct{ path, blob string }
	var ents []ent
	for _, line := range strings.Split(lsTree, "\n") {
		line = strings.TrimRight(line, "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}
		// Format: "<mode> <type> <objectsha>\t<path>"
		tab := strings.IndexByte(line, '\t')
		if tab < 0 {
			continue
		}
		meta, path := line[:tab], line[tab+1:]
		fields := strings.Fields(meta)
		if len(fields) != 3 {
			continue
		}
		if !strings.HasSuffix(path, ".go") || isToolTestdataPath(path) {
			continue
		}
		ents = append(ents, ent{path: path, blob: fields[2]})
	}
	sort.Slice(ents, func(i, j int) bool { return ents[i].path < ents[j].path })
	h := sha256.New()
	for _, e := range ents {
		fmt.Fprintf(h, "%s %s\n", e.path, e.blob)
	}
	return hex.EncodeToString(h.Sum(nil))
}
