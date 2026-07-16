package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/Chronicle20/atlas/tools/packet-audit/internal/matrix"
	"gopkg.in/yaml.v3"
)

// dispatcher-lint enforces the mode-prefix dispatcher-family invariants
// documented in docs/packets/DISPATCHER_FAMILY.md (INV-1..INV-5). It is a
// read-only static linter, CI-gated alongside `matrix --check`.
//
// A "dispatcher family" is any base IDA FName that has >=1 `#`-suffixed
// clientbound entry in run.go's candidatesFromFName (e.g.
// CUIMessenger::OnPacket, with arms CUIMessenger::OnPacket#Add, …). Each arm
// maps a per-version mode to one discrete Atlas clientbound struct. The
// invariants guarantee discrete-per-mode structs, config-driven mode
// resolution, and footgun-free body-function APIs.
//
// Exit 0 when clean; non-zero when any violation OUTSIDE the baseline is found.
// Families listed in docs/packets/dispatcher-lint-baseline.yaml are skipped
// (one note line printed each); the baseline only shrinks.

// dispatcherArm is one #-suffixed clientbound candidate entry parsed from
// candidatesFromFName in run.go.
type dispatcherArm struct {
	family string // base FName, e.g. "CUIMessenger::OnPacket"
	mode   string // the part after '#', e.g. "Add"
	name   string // candidate struct name, e.g. "Add"
	pkg    string // candidate sub-domain folder, e.g. "messenger"
}

// dispatcherLintConfig parameterises the linter so tests can point it at a
// temp-dir fixture. The defaults match the repo layout (commands run from the
// repo root via `go run ./tools/packet-audit dispatcher-lint`).
type dispatcherLintConfig struct {
	RunGo        string   // path to run.go (parsed for candidatesFromFName)
	PacketLib    string   // libs/atlas-packet root
	AuditsDir    string   // docs/packets/audits root (for INV-4 report check)
	BaselinePath string   // docs/packets/dispatcher-lint-baseline.yaml
	UsageRoots   []string // roots scanned for constructor/struct-literal usage (INV-5)
	// FR-5.1 family-cap guard inputs.
	DispatchersDir string // docs/packets/dispatchers (family mode-table yamls)
	FamiliesPath   string // docs/packets/evidence/families.yaml (graduated membership)
}

func defaultDispatcherLintConfig() dispatcherLintConfig {
	return dispatcherLintConfig{
		RunGo:        filepath.Join("tools", "packet-audit", "cmd", "run.go"),
		PacketLib:    filepath.Join("libs", "atlas-packet"),
		AuditsDir:    filepath.Join("docs", "packets", "audits"),
		BaselinePath: filepath.Join("docs", "packets", "dispatcher-lint-baseline.yaml"),
		// INV-5 "no body function / feature wraps it" must see callers in the
		// services too — non-operations-backed families (whisper, warp, npc
		// conversation detail) are constructed directly by a channel consumer
		// or handler, not by a libs body function.
		UsageRoots: []string{filepath.Join("libs", "atlas-packet"), "services"},

		DispatchersDir: filepath.Join("docs", "packets", "dispatchers"),
		FamiliesPath:   filepath.Join("docs", "packets", "evidence", "families.yaml"),
	}
}

// violation is a single linter finding. file:line are the source location;
// inv is "INV-1".."INV-5"; msg is the human-readable reason.
type violation struct {
	file string
	line int
	inv  string
	msg  string
}

func (v violation) String() string {
	loc := v.file
	if v.line > 0 {
		loc = fmt.Sprintf("%s:%d", v.file, v.line)
	}
	return fmt.Sprintf("%s\t%s\t%s", loc, v.inv, v.msg)
}

type dispatcherLintBaseline struct {
	ExemptFamilies []string `yaml:"exempt_families"`
}

func runDispatcherLint(args []string, stderr io.Writer) int {
	// No flags today; reject unexpected positional args so a typo isn't a
	// silent no-op.
	for _, a := range args {
		if a == "-h" || a == "--help" || a == "help" {
			fmt.Fprintln(stderr, "usage: packet-audit dispatcher-lint")
			fmt.Fprintln(stderr, "enforces docs/packets/DISPATCHER_FAMILY.md invariants INV-1..INV-5; read-only.")
			return 0
		}
		fmt.Fprintf(stderr, "packet-audit dispatcher-lint: unexpected argument %q\n", a)
		return 3
	}
	return dispatcherLintRun(defaultDispatcherLintConfig(), os.Stdout, stderr)
}

// dispatcherLintRun is the testable core: it returns the process exit code and
// writes the human-readable report to out / errors to stderr.
func dispatcherLintRun(cfg dispatcherLintConfig, out, stderr io.Writer) int {
	exempt, err := loadDispatcherBaseline(cfg.BaselinePath)
	if err != nil {
		fmt.Fprintf(stderr, "packet-audit dispatcher-lint: %v\n", err)
		return 3
	}

	arms, err := parseDispatcherArms(cfg.RunGo)
	if err != nil {
		fmt.Fprintf(stderr, "packet-audit dispatcher-lint: %v\n", err)
		return 3
	}
	arms = modePrefixDispatcherArms(arms)

	violations, err := collectDispatcherViolations(cfg, arms)
	if err != nil {
		fmt.Fprintf(stderr, "packet-audit dispatcher-lint: %v\n", err)
		return 3
	}

	// Partition by baseline membership. A violation is exempt when its family
	// is baselined. Print a single note per exempt family that actually
	// suppressed at least one violation, plus any baselined family with none
	// (so a stale baseline entry is visible).
	var reported []violation
	suppressed := map[string]int{}
	for _, v := range violations {
		fam := familyOfViolation(v)
		if fam != "" && exempt[fam] {
			suppressed[fam]++
			continue
		}
		reported = append(reported, v)
	}

	// Notes for every baselined family (shrink-only contract: surface them).
	famNames := make([]string, 0, len(exempt))
	for f := range exempt {
		famNames = append(famNames, f)
	}
	sort.Strings(famNames)
	for _, f := range famNames {
		fmt.Fprintf(out, "note\t%s\tbaselined — pending migration (%d violation(s) suppressed)\n", f, suppressed[f])
	}

	sort.Slice(reported, func(i, j int) bool {
		if reported[i].file != reported[j].file {
			return reported[i].file < reported[j].file
		}
		if reported[i].line != reported[j].line {
			return reported[i].line < reported[j].line
		}
		return reported[i].inv < reported[j].inv
	})
	for _, v := range reported {
		fmt.Fprintln(out, v.String())
	}

	if len(reported) > 0 {
		fmt.Fprintf(stderr, "packet-audit dispatcher-lint: %d violation(s) (see docs/packets/DISPATCHER_FAMILY.md)\n", len(reported))
		return 1
	}
	fmt.Fprintln(out, "dispatcher-lint: clean")
	return 0
}

// familyOfViolation extracts the dispatcher family encoded in a violation's
// msg via the "[family=…]" tag the collectors append, so baseline filtering
// is exact regardless of file path.
var familyTagRe = regexp.MustCompile(`\[family=([^\]]+)\]`)

func familyOfViolation(v violation) string {
	m := familyTagRe.FindStringSubmatch(v.msg)
	if m == nil {
		return ""
	}
	return m[1]
}

func loadDispatcherBaseline(path string) (map[string]bool, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading baseline %s: %w", path, err)
	}
	var doc dispatcherLintBaseline
	if err := yaml.Unmarshal(b, &doc); err != nil {
		return nil, fmt.Errorf("parsing baseline %s: %w", path, err)
	}
	set := map[string]bool{}
	for _, f := range doc.ExemptFamilies {
		set[strings.TrimSpace(f)] = true
	}
	return set, nil
}

// caseHashRe matches a `case "<Fname>#<Mode>":` line in run.go. The Mode part
// must be non-empty (a real dispatcher arm). Multiple comma-separated case
// labels on one line are not used in run.go's candidatesFromFName, so a single
// label per case is sufficient.
var caseHashRe = regexp.MustCompile(`^\s*case\s+"([^"#]+)#([^"#]+)":\s*$`)

// candidateClientboundRe matches a `{name: "X", pkg: "Y", dir: csvpkg.DirClientbound`
// candidate literal (the dir token anchors it to a clientbound arm; pkg is
// required so login/socket arms without a pkg are excluded).
var candidateClientboundRe = regexp.MustCompile(`\{name:\s*"([^"]+)",\s*pkg:\s*"([^"]+)",\s*dir:\s*csvpkg\.DirClientbound`)

// parseDispatcherArms scans run.go for `case "<Fname>#<Mode>":` lines whose
// following `return []candidate{...}` includes a clientbound candidate with a
// pkg. Each such arm is recorded.
func parseDispatcherArms(runGo string) ([]dispatcherArm, error) {
	b, err := os.ReadFile(runGo)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", runGo, err)
	}
	lines := strings.Split(string(b), "\n")
	var arms []dispatcherArm
	for i := 0; i < len(lines); i++ {
		m := caseHashRe.FindStringSubmatch(lines[i])
		if m == nil {
			continue
		}
		family, mode := m[1], m[2]
		// Scan forward to the first `return []candidate{...}` before the next
		// `case `/`}` at the switch level; capture the first clientbound
		// candidate with a pkg.
		for j := i + 1; j < len(lines) && j < i+40; j++ {
			lj := strings.TrimSpace(lines[j])
			if strings.HasPrefix(lj, "case ") || lj == "}" || lj == "default:" {
				break
			}
			cm := candidateClientboundRe.FindStringSubmatch(lines[j])
			if cm != nil {
				arms = append(arms, dispatcherArm{
					family: family,
					mode:   mode,
					name:   cm[1],
					pkg:    cm[2],
				})
				break
			}
			if strings.HasPrefix(lj, "return ") {
				// a return that is NOT a clientbound-with-pkg candidate
				// (e.g. serverbound) — this arm is not a clientbound
				// dispatcher arm; stop scanning.
				break
			}
		}
	}
	return arms, nil
}

// modePrefixDispatcherArms keeps only arms whose base FName has MORE THAN ONE
// clientbound #-entry. A mode-prefix dispatcher is, by definition, one opcode
// whose leading mode byte routes to N>1 sub-handlers (DISPATCHER_FAMILY.md).
// A base FName with a single clientbound #-entry (e.g. CScriptMan::OnSayImage,
// CStage::OnSetField) is a sub-named non-dispatcher packet, not a mode
// dispatcher, and is excluded so the linter does not false-positive on it.
func modePrefixDispatcherArms(arms []dispatcherArm) []dispatcherArm {
	count := map[string]int{}
	for _, a := range arms {
		count[a.family]++
	}
	var out []dispatcherArm
	for _, a := range arms {
		if count[a.family] > 1 {
			out = append(out, a)
		}
	}
	return out
}

func collectDispatcherViolations(cfg dispatcherLintConfig, arms []dispatcherArm) ([]violation, error) {
	var out []violation

	// INV-1: a struct mapped by >1 dispatcher #-entry.
	inv1, err := checkINV1(arms)
	if err != nil {
		return nil, err
	}
	out = append(out, inv1...)

	// INV-4 (a): every #-entry candidate resolves to an existing
	// `type <name> struct` in libs/atlas-packet/<pkg>/clientbound/.
	inv4a, err := checkINV4Candidates(cfg, arms)
	if err != nil {
		return nil, err
	}
	out = append(out, inv4a...)

	// The struct-level checks (INV-2(a), INV-5) operate on the discrete
	// clientbound structs of each family. Build the family→struct file map
	// from the resolvable arms.
	structs, err := resolveFamilyStructs(cfg, arms)
	if err != nil {
		return nil, err
	}

	inv2a, err := checkINV2ModeLiteral(structs)
	if err != nil {
		return nil, err
	}
	out = append(out, inv2a...)

	// INV-5 enforces "every dispatcher clientbound struct is constructed by a
	// body function" — so it applies only to families that HAVE a body-function
	// layer (the operations-table canonical pattern: cash/npc-shop/storage/MTS/
	// messenger/interaction/field_effect/party/guild). Families that construct
	// their discrete structs directly in services (whisper, guide/OnTutorMsg —
	// no *_body.go) are not body-function families; a discrete struct with no
	// current sender there is a coverage gap, not an AP-6 orphaned-codec
	// regression, so INV-5 does not apply.
	bodyFnFamilies, err := bodyFunctionBackedFamilies(cfg, structs)
	if err != nil {
		return nil, err
	}
	var bodyFnStructs []resolvedStruct
	for _, s := range structs {
		if bodyFnFamilies[s.family] {
			bodyFnStructs = append(bodyFnStructs, s)
		}
	}
	inv5, err := checkINV5Orphans(cfg, bodyFnStructs)
	if err != nil {
		return nil, err
	}
	out = append(out, inv5...)

	// The body-function-file checks (INV-2(b), INV-3) scan the per-family
	// body-function files.
	inv2b3, err := checkBodyFunctionFiles(cfg, arms)
	if err != nil {
		return nil, err
	}
	out = append(out, inv2b3...)

	// INV-4 (b): committed audit reports citing a deleted Atlas file.
	inv4b, err := checkINV4Reports(cfg)
	if err != nil {
		return nil, err
	}
	out = append(out, inv4b...)

	// FR-5.1: every dispatchers/*.yaml family must be discrete-implemented or
	// baseline/families-listed (so a new family can't escape mode-prefix capping).
	famcap, err := checkFamilyCap(cfg)
	if err != nil {
		return nil, err
	}
	out = append(out, famcap...)

	return out, nil
}

// dispatcherYamlFamily is the minimal shape of a docs/packets/dispatchers/*.yaml
// mode-table file — just enough to read which dispatcher family it defines.
type dispatcherYamlFamily struct {
	FName string `yaml:"fname"`
}

// caseLabelFamilies scans run.go for every base FName that has at least one
// `case "<fname>#<mode>":` label in candidatesFromFName — the discrete-per-mode
// wiring signal, independent of the candidate literal's field order (which the
// stricter arm parser is sensitive to). A missing run.go yields an empty set.
func caseLabelFamilies(runGo string) (map[string]bool, error) {
	raw, err := os.ReadFile(runGo)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]bool{}, nil
		}
		return nil, err
	}
	set := map[string]bool{}
	for _, line := range strings.Split(string(raw), "\n") {
		if m := caseHashRe.FindStringSubmatch(line); m != nil {
			set[m[1]] = true
		}
	}
	return set, nil
}

// checkFamilyCap (FR-5.1) enforces that every docs/packets/dispatchers/*.yaml
// family is EITHER discrete-implemented (it has #-suffixed case arms in run.go —
// and is therefore already subject to INV-1..5 and capped at 🧩 until every arm
// is verified) OR explicitly listed in families.yaml / the dispatcher-lint
// baseline. A brand-new dispatcher family file with no discrete implementation
// and no baseline/families entry would otherwise silently escape mode-prefix
// capping — this closes that hole. Violations are tagged [family=<fname>] so a
// baselined family is suppressed by the same partition as the INV checks.
func checkFamilyCap(cfg dispatcherLintConfig) ([]violation, error) {
	entries, err := os.ReadDir(cfg.DispatchersDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	armSet, err := caseLabelFamilies(cfg.RunGo)
	if err != nil {
		return nil, err
	}
	fams, err := matrix.LoadFamilies(cfg.FamiliesPath)
	if err != nil {
		return nil, fmt.Errorf("loading families.yaml %s: %w", cfg.FamiliesPath, err)
	}
	famSet := fams.Set()

	// Sort file names so violation output is deterministic.
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		names = append(names, e.Name())
	}
	sort.Strings(names)

	var out []violation
	for _, name := range names {
		p := filepath.Join(cfg.DispatchersDir, name)
		raw, err := os.ReadFile(p)
		if err != nil {
			return nil, err
		}
		var d dispatcherYamlFamily
		if err := yaml.Unmarshal(raw, &d); err != nil {
			return nil, fmt.Errorf("parsing dispatcher family %s: %w", p, err)
		}
		if d.FName == "" {
			out = append(out, violation{file: p, inv: "FAM-CAP",
				msg: "dispatcher family file declares no `fname:` — cannot confirm it is mode-prefix capped"})
			continue
		}
		if armSet[d.FName] || famSet[d.FName] {
			continue
		}
		out = append(out, violation{file: p, inv: "FAM-CAP",
			msg: fmt.Sprintf("family %s is neither discrete-implemented in run.go (no #-suffixed arms) nor listed in families.yaml/baseline — author it discrete-per-mode or add a baseline/families cap [family=%s]", d.FName, d.FName)})
	}
	return out, nil
}

// resolvedStruct is a dispatcher arm whose struct type was located on disk.
type resolvedStruct struct {
	family string
	pkg    string
	name   string
	file   string // path to the struct's definition file
}

func resolveFamilyStructs(cfg dispatcherLintConfig, arms []dispatcherArm) ([]resolvedStruct, error) {
	seen := map[string]bool{} // dedup by pkg/name (multiple arms can share a struct)
	var out []resolvedStruct
	for _, a := range arms {
		key := a.pkg + "/" + a.name
		if seen[key] {
			continue
		}
		file, ok := findClientboundStructFile(cfg.PacketLib, a.pkg, a.name)
		if !ok {
			continue // INV-4(a) reports the dangling candidate separately
		}
		seen[key] = true
		out = append(out, resolvedStruct{family: a.family, pkg: a.pkg, name: a.name, file: file})
	}
	return out, nil
}

// findClientboundStructFile locates the file defining `type <name> struct` in
// libs/atlas-packet/<pkg>/clientbound/.
func findClientboundStructFile(packetLib, pkg, name string) (string, bool) {
	dir := filepath.Join(packetLib, pkg, "clientbound")
	var hit string
	_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		b, rerr := os.ReadFile(path)
		if rerr != nil {
			return nil
		}
		if structDeclLine(string(b), name) > 0 {
			hit = path
			return filepath.SkipAll
		}
		return nil
	})
	return hit, hit != ""
}

// structDeclLine returns the 1-based line of `type <name> struct` in src, or 0.
func structDeclLine(src, name string) int {
	re := regexp.MustCompile(`(?m)^type\s+` + regexp.QuoteMeta(name) + `\s+struct\b`)
	loc := re.FindStringIndex(src)
	if loc == nil {
		return 0
	}
	return 1 + strings.Count(src[:loc[0]], "\n")
}

// --- INV-1 ---------------------------------------------------------------

func checkINV1(arms []dispatcherArm) ([]violation, error) {
	// pkg/name -> set of "family#mode" labels mapping to it.
	byStruct := map[string][]dispatcherArm{}
	for _, a := range arms {
		key := a.pkg + "/" + a.name
		byStruct[key] = append(byStruct[key], a)
	}
	var out []violation
	keys := make([]string, 0, len(byStruct))
	for k := range byStruct {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		group := byStruct[k]
		if len(group) <= 1 {
			continue
		}
		labels := make([]string, 0, len(group))
		fams := map[string]bool{}
		for _, a := range group {
			labels = append(labels, a.family+"#"+a.mode)
			fams[a.family] = true
		}
		sort.Strings(labels)
		// Emit one violation per involved family so baseline filtering can
		// suppress an exempt family's share independently.
		famList := make([]string, 0, len(fams))
		for f := range fams {
			famList = append(famList, f)
		}
		sort.Strings(famList)
		for _, fam := range famList {
			out = append(out, violation{
				file: "tools/packet-audit/cmd/run.go",
				line: 0,
				inv:  "INV-1",
				msg: fmt.Sprintf("struct %s is mapped by %d dispatcher #-entries (%s) — shared-by-shape (AP-1) [family=%s]",
					k, len(group), strings.Join(labels, ", "), fam),
			})
		}
	}
	return out, nil
}

// --- INV-4(a): dangling candidate ---------------------------------------

func checkINV4Candidates(cfg dispatcherLintConfig, arms []dispatcherArm) ([]violation, error) {
	seen := map[string]bool{}
	var out []violation
	for _, a := range arms {
		key := a.pkg + "/" + a.name
		if seen[key] {
			continue
		}
		seen[key] = true
		if _, ok := findClientboundStructFile(cfg.PacketLib, a.pkg, a.name); !ok {
			out = append(out, violation{
				file: "tools/packet-audit/cmd/run.go",
				line: 0,
				inv:  "INV-4",
				msg: fmt.Sprintf("dispatcher #-entry %s#%s -> {name:%q, pkg:%q} has no `type %s struct` in %s/%s/clientbound/ — dangling candidate (AP-5) [family=%s]",
					a.family, a.mode, a.name, a.pkg, a.name, cfg.PacketLib, a.pkg, a.family),
			})
		}
	}
	return out, nil
}

// --- INV-2(a): hard-coded mode byte in a constructor ----------------------

// modeLiteralRe matches a `mode: 0x..` field initializer (the AP-2 footgun)
// inside a composite literal.
var modeLiteralRe = regexp.MustCompile(`(?m)\bmode:\s*0x[0-9A-Fa-f]+`)

func checkINV2ModeLiteral(structs []resolvedStruct) ([]violation, error) {
	checked := map[string]string{} // file -> family (one violation set per file)
	for _, s := range structs {
		if _, ok := checked[s.file]; !ok {
			checked[s.file] = s.family
		}
	}
	var out []violation
	files := make([]string, 0, len(checked))
	for f := range checked {
		files = append(files, f)
	}
	sort.Strings(files)
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", f, err)
		}
		src := string(b)
		for _, loc := range modeLiteralRe.FindAllStringIndex(src, -1) {
			ln := 1 + strings.Count(src[:loc[0]], "\n")
			out = append(out, violation{
				file: filepath.ToSlash(f),
				line: ln,
				inv:  "INV-2",
				msg:  fmt.Sprintf("hard-coded mode byte literal in a dispatcher struct constructor — bypasses the tenant operations table (AP-2) [family=%s]", checked[f]),
			})
		}
	}
	return out, nil
}

// --- INV-5: orphaned codec (no body function constructs it) ---------------

func checkINV5Orphans(cfg dispatcherLintConfig, structs []resolvedStruct) ([]violation, error) {
	// Build the set of .go files (excluding _test.go) across every usage root,
	// so a constructor call OR struct-literal in a libs body function or a
	// service consumer/handler is detected.
	var goFiles []string
	for _, root := range cfg.UsageRoots {
		files, err := allGoFiles(root)
		if err != nil {
			return nil, err
		}
		goFiles = append(goFiles, files...)
	}
	cache := map[string]string{}
	readFile := func(p string) string {
		if c, ok := cache[p]; ok {
			return c
		}
		b, _ := os.ReadFile(p)
		cache[p] = string(b)
		return string(b)
	}

	var out []violation
	for _, s := range structs {
		// Construction patterns: any `New…(…) <Struct>` constructor defined in
		// the struct's own file, OR a `<Struct>{` composite literal. A struct
		// is wrapped if either appears outside its def file and its _test.go.
		ctors := constructorsForStruct(readFile(s.file), s.name)
		var callPatterns []*regexp.Regexp
		for _, ctor := range ctors {
			callPatterns = append(callPatterns, regexp.MustCompile(`\b`+regexp.QuoteMeta(ctor)+`\s*\(`))
		}
		// Composite-literal usage: <Struct>{ possibly package-qualified
		// (pkg.Struct{). Anchor on a non-identifier char before the name so
		// `FooStruct` doesn't match `MyFooStruct`.
		callPatterns = append(callPatterns, regexp.MustCompile(`(^|[^\w.])`+regexp.QuoteMeta(s.name)+`\{`))
		callPatterns = append(callPatterns, regexp.MustCompile(`\.`+regexp.QuoteMeta(s.name)+`\{`))

		testFile := testFileFor(s.file)
		constructed := false
		for _, gf := range goFiles {
			if gf == s.file || gf == testFile {
				continue
			}
			content := readFile(gf)
			for _, re := range callPatterns {
				if re.MatchString(content) {
					constructed = true
					break
				}
			}
			if constructed {
				break
			}
		}
		if !constructed {
			ctorNote := "no New-style constructor"
			if len(ctors) > 0 {
				ctorNote = "constructor " + strings.Join(ctors, "/")
			}
			out = append(out, violation{
				file: filepath.ToSlash(s.file),
				line: structDeclLine(readFile(s.file), s.name),
				inv:  "INV-5",
				msg:  fmt.Sprintf("dispatcher struct %s (%s) is never constructed outside its own file/_test.go — orphaned codec, no body function or feature wraps it (AP-6) [family=%s]", s.name, ctorNote, s.family),
			})
		}
	}
	return out, nil
}

// constructorsForStruct returns the names of `func New…(…) <Struct>` (or
// `*<Struct>`) functions defined in src.
func constructorsForStruct(src, structName string) []string {
	re := regexp.MustCompile(`(?m)^func\s+(New\w+)\s*\([^)]*\)\s*\*?` + regexp.QuoteMeta(structName) + `\s*\{`)
	var out []string
	for _, m := range re.FindAllStringSubmatch(src, -1) {
		out = append(out, m[1])
	}
	return out
}

func testFileFor(file string) string {
	if strings.HasSuffix(file, ".go") && !strings.HasSuffix(file, "_test.go") {
		return strings.TrimSuffix(file, ".go") + "_test.go"
	}
	return ""
}

func allGoFiles(root string) ([]string, error) {
	var out []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go") {
			out = append(out, path)
		}
		return nil
	})
	return out, err
}

// --- INV-2(b) + INV-3: body-function files --------------------------------

// bodyFuncRe matches an exported `func XxxBody(…)` declaration; the param list
// is captured in group 2. A trailing digit suffix (XxxBody2) is allowed so a
// numbered overload that carries the same footgun is still caught.
var bodyFuncRe = regexp.MustCompile(`(?m)^func\s+([A-Z]\w*Body\d*)\s*\(([^)]*)\)`)

// discardedModeRe matches `func(_ byte) packet.Encoder` (AP-3 — the resolved
// mode is discarded).
var discardedModeRe = regexp.MustCompile(`func\(\s*_\s+byte\s*\)\s*packet\.Encoder`)

// selectorParamRe matches a parameter named op/code/mode/key of type string or
// *Mode in a Body func's parameter list (AP-4). This is the by-name heuristic;
// it is intentionally backed up by the semantic check below (withResolvedKeyRe +
// paramNameSet), because a selector under any other name (errorCode, reason, …)
// reads as the same footgun and must not escape on naming alone.
var selectorParamRe = regexp.MustCompile(`(?:^|,)\s*(op|code|mode|key)\s+(string|\*Mode)\b`)

// withResolvedKeyRe captures the operations KEY argument of a
// `WithResolvedCode("operations", <key>, …)` call when that key is a bare
// identifier. A fixed key is a package const (`MtsOperationFoo`), a `string(Const)`
// cast (the `(` after the cast keyword means no comma follows the bare word, so it
// is not captured), or a string literal (`"RECEIVE"`, starts with a quote) — none
// of which are function parameters. The footgun (AP-4) is when this identifier is
// one of the enclosing Body func's own parameters: the caller, not the packet's
// fixed operation, picks the mode. This catches the selector regardless of its name.
var withResolvedKeyRe = regexp.MustCompile(`WithResolvedCode\(\s*"operations"\s*,\s*([A-Za-z_]\w*)\s*,`)

// paramNameSet parses a Go parameter list (the text between the parens) into the
// set of parameter NAMES. It handles both `name type` and grouped `a, b type`
// forms by treating, in each comma-separated chunk, every leading identifier
// before the type token as a name; a chunk that is a lone identifier (the `a` in
// `a, b type`) is also a name. Good enough for the simple Body signatures here.
func paramNameSet(params string) map[string]bool {
	out := map[string]bool{}
	for _, chunk := range strings.Split(params, ",") {
		fields := strings.Fields(strings.TrimSpace(chunk))
		if len(fields) == 0 {
			continue
		}
		// `name type` → name is fields[0]; lone `name` (grouped) → fields[0].
		name := fields[0]
		if isIdent(name) {
			out[name] = true
		}
	}
	return out
}

func isIdent(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		if r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			continue
		}
		if i > 0 && r >= '0' && r <= '9' {
			continue
		}
		return false
	}
	return true
}

func checkBodyFunctionFiles(cfg dispatcherLintConfig, arms []dispatcherArm) ([]violation, error) {
	// Collect the body-function files for each dispatcher family's pkg:
	//   libs/atlas-packet/<pkg>/operation_body.go
	//   libs/atlas-packet/<pkg>/*_body.go
	//   libs/atlas-packet/<pkg>/clientbound/*_body.go
	// Map each file to its family so violations carry the right family tag.
	fileFamily := map[string]string{}
	for _, a := range arms {
		for _, f := range bodyFileCandidates(cfg.PacketLib, a.pkg) {
			if _, err := os.Stat(f); err == nil {
				if _, ok := fileFamily[f]; !ok {
					fileFamily[f] = a.family
				}
			}
		}
	}

	var out []violation
	files := make([]string, 0, len(fileFamily))
	for f := range fileFamily {
		files = append(files, f)
	}
	sort.Strings(files)
	for _, f := range files {
		fam := fileFamily[f]
		b, err := os.ReadFile(f)
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", f, err)
		}
		src := string(b)

		// INV-2(b): func(_ byte) packet.Encoder anywhere in a body file.
		for _, loc := range discardedModeRe.FindAllStringIndex(src, -1) {
			ln := 1 + strings.Count(src[:loc[0]], "\n")
			out = append(out, violation{
				file: filepath.ToSlash(f),
				line: ln,
				inv:  "INV-2",
				msg:  fmt.Sprintf("`func(_ byte) packet.Encoder` discards the resolved mode — must be `func(mode byte)` passthrough (AP-3) [family=%s]", fam),
			})
		}

		// INV-3: exported *Body func with a caller-specified operation selector.
		// Two complementary signals, deduped to one violation per func:
		//   (a) by-name — a param literally named op/code/mode/key (selectorParamRe);
		//   (b) semantic — a param flows into the `WithResolvedCode("operations", …)`
		//       key, regardless of the param's name (errorCode, reason, …).
		fnMatches := bodyFuncRe.FindAllStringSubmatchIndex(src, -1)
		for i, m := range fnMatches {
			fnName := src[m[2]:m[3]]
			params := src[m[4]:m[5]]
			// Function body text: from this match to the next top-level Body func
			// (or EOF). Enough to see this func's WithResolvedCode call.
			bodyEnd := len(src)
			if i+1 < len(fnMatches) {
				bodyEnd = fnMatches[i+1][0]
			}
			funcText := src[m[0]:bodyEnd]

			var reason string
			if sel := selectorParamRe.FindStringSubmatch(params); sel != nil {
				reason = fmt.Sprintf("a caller-specified mode selector (%s %s)", sel[1], sel[2])
			} else {
				pnames := paramNameSet(params)
				for _, km := range withResolvedKeyRe.FindAllStringSubmatch(funcText, -1) {
					if pnames[km[1]] {
						reason = fmt.Sprintf("its parameter %q as the resolved operations key", km[1])
						break
					}
				}
			}
			if reason != "" {
				ln := 1 + strings.Count(src[:m[0]], "\n")
				out = append(out, violation{
					file: filepath.ToSlash(f),
					line: ln,
					inv:  "INV-3",
					msg:  fmt.Sprintf("body func %s takes %s — the packet maps to ONE operation, so fix the key instead (AP-4) [family=%s]", fnName, reason, fam),
				})
			}
		}
	}
	return out, nil
}

// bodyFunctionBackedFamilies returns the set of families that use the
// operations-table body-function pattern — i.e. at least one of the family's
// discrete structs is constructed inside a `*_body.go` body-function file
// (the canonical wrap). Only those families are subject to INV-5 ("every
// dispatcher clientbound struct is constructed by a body function"); families
// that construct their structs directly in services (no body file references
// them — whisper, guide) are not body-function families.
func bodyFunctionBackedFamilies(cfg dispatcherLintConfig, structs []resolvedStruct) (map[string]bool, error) {
	out := map[string]bool{}
	// Collect every body file referenced by any family's pkg, read once.
	bodySrc := map[string]string{}
	for _, s := range structs {
		for _, f := range bodyFileCandidates(cfg.PacketLib, s.pkg) {
			if _, ok := bodySrc[f]; ok {
				continue
			}
			b, err := os.ReadFile(f)
			if err == nil {
				bodySrc[f] = string(b)
			}
		}
	}
	for _, s := range structs {
		if out[s.family] {
			continue
		}
		// The struct is wrapped if a body file constructs it (New<Ctor>( or a
		// <Struct>{ literal).
		ctors := constructorsForStruct(mustReadFileString(s.file), s.name)
		var pats []*regexp.Regexp
		for _, c := range ctors {
			pats = append(pats, regexp.MustCompile(`\b`+regexp.QuoteMeta(c)+`\s*\(`))
		}
		pats = append(pats, regexp.MustCompile(`(^|[^\w.])`+regexp.QuoteMeta(s.name)+`\{`))
		pats = append(pats, regexp.MustCompile(`\.`+regexp.QuoteMeta(s.name)+`\{`))
		for f, src := range bodySrc {
			// only body files in the struct's own pkg are relevant
			if !strings.Contains(filepath.ToSlash(f), "/"+s.pkg+"/") {
				continue
			}
			for _, re := range pats {
				if re.MatchString(src) {
					out[s.family] = true
					break
				}
			}
			if out[s.family] {
				break
			}
		}
	}
	return out, nil
}

func mustReadFileString(p string) string {
	b, _ := os.ReadFile(p)
	return string(b)
}

func bodyFileCandidates(packetLib, pkg string) []string {
	var out []string
	pkgDir := filepath.Join(packetLib, pkg)
	// <pkg>/operation_body.go and <pkg>/*_body.go
	if matches, err := filepath.Glob(filepath.Join(pkgDir, "*_body.go")); err == nil {
		out = append(out, matches...)
	}
	// <pkg>/clientbound/*_body.go
	if matches, err := filepath.Glob(filepath.Join(pkgDir, "clientbound", "*_body.go")); err == nil {
		out = append(out, matches...)
	}
	return out
}

// --- INV-4(b): phantom audit reports --------------------------------------

// reportAtlasFileJSONRe and reportAtlasFileMDRe extract the recorded Atlas file
// path from a committed audit report (JSON: "AtlasFile": "…"; MD:
// **Atlas file:** `…`).
var reportAtlasFileJSONRe = regexp.MustCompile(`"AtlasFile":\s*"([^"]+)"`)
var reportAtlasFileMDRe = regexp.MustCompile("(?i)\\*\\*Atlas file:\\*\\*\\s*`([^`]+)`")

// reportAtlasFileRepoRel normalizes an AtlasFile path from an audit report to a
// repo-relative path: it strips any leading `../` segments (legacy reports carry
// a `../../` prefix; newer ones are already repo-relative). Matches the matrix
// loader's normalization intent (load.go) so the two agree on file existence.
// (Distinct from run.go's repoRelAtlasFile, which normalizes ABSOLUTE
// --atlas-packet inputs and deliberately leaves relative ../../ paths untouched.)
func reportAtlasFileRepoRel(af string) string {
	af = filepath.ToSlash(af)
	for strings.HasPrefix(af, "../") {
		af = af[len("../"):]
	}
	return af
}

func checkINV4Reports(cfg dispatcherLintConfig) ([]violation, error) {
	var out []violation
	err := filepath.WalkDir(cfg.AuditsDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".json" && ext != ".md" {
			return nil
		}
		b, rerr := os.ReadFile(path)
		if rerr != nil {
			return nil
		}
		src := string(b)
		var re *regexp.Regexp
		if ext == ".json" {
			re = reportAtlasFileJSONRe
		} else {
			re = reportAtlasFileMDRe
		}
		for _, m := range re.FindAllStringSubmatchIndex(src, -1) {
			af := src[m[2]:m[3]]
			if af == "" {
				continue
			}
			// AtlasFile is recorded either repo-relative (libs/atlas-packet/…,
			// newer) or with a legacy ../../ prefix (older reports). Normalize to
			// repo-relative before stat so the check is CWD-independent — statting
			// the raw ../../ form only resolves by accident when CWD happens to be
			// nested exactly that deep (e.g. a 2-level worktree), and fails in CI.
			rel := reportAtlasFileRepoRel(af)
			if _, statErr := os.Stat(rel); os.IsNotExist(statErr) {
				ln := 1 + strings.Count(src[:m[0]], "\n")
				out = append(out, violation{
					file: filepath.ToSlash(path),
					line: ln,
					inv:  "INV-4",
					msg:  fmt.Sprintf("audit report cites Atlas file %q which no longer exists — phantom/deleted-file report (AP-5)", af),
				})
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}
