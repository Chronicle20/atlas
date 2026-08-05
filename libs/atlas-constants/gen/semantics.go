// semantics.go implements the atlas-constants per-version semantics
// generator (task-187 Task 4): the ONLY version-sensitive layer in the
// whole library. Every other property of an Identity (predicates, display
// name, advancement tier, ...) is version-blind; only "which wire id means
// this Identity in version V" varies, and that binding lives here.
//
// Pipeline:
//
//	divergences.csv (Task 1 audit ledger, all versions)
//	  --[-author-semantics, one-off]--> gen/semantics/<r>_<maj>_<min>.yaml (checked in, per version)
//	  --[BuildSemantics, every generator run]--> map[wireId]identityName (both domains)
//	  --[EmitSemantics]--> skill/version_<r>_<maj>_<min>_gen.go, job/version_<r>_<maj>_<min>_gen.go
//
// The per-version YAML is authored once from divergences.csv (Step 1) and
// then checked in as its own source of record -- BuildSemantics reads it
// directly (plus the pinned wzsnapshot), it does not re-parse the master
// CSV on every generator run. -author-semantics regenerates it
// deterministically so it is never hand-typed.
package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/Chronicle20/atlas/libs/atlas-constants/gen/wzsnapshot"
)

// versionKey identifies one provisioned (region, major, minor) column.
type versionKey struct {
	Region string
	Major  uint16
	Minor  uint16
}

// key returns the canonical "<region>_<major>_<minor>" identifier used in
// generated file names, package-level Go identifiers, and semantics.yaml
// file names.
func (v versionKey) key() string {
	return fmt.Sprintf("%s_%d_%d", v.Region, v.Major, v.Minor)
}

// provisionedVersions is the exact set from deploy/k8s/base/versions.json
// (context.md §4). Kept as a literal (mirrors the equivalent list in
// audit_validate_test.go) so this generator has no dependency beyond the
// standard library + the checked-in inputs.
var provisionedVersions = []versionKey{
	{"gms", 12, 1},
	{"gms", 48, 1},
	{"gms", 61, 1},
	{"gms", 72, 1},
	{"gms", 79, 1},
	{"gms", 83, 1},
	{"gms", 84, 1},
	{"gms", 87, 1},
	{"gms", 92, 1},
	{"gms", 95, 1},
	{"jms", 185, 1},
}

const (
	semanticsDir       = "semantics"
	divergencesCSVPath = "../../../docs/tasks/task-187-version-aware-id-semantics/audit/divergences.csv"
	skillVersionGenDir = "../skill"
	jobVersionGenDir   = "../job"
)

// divergenceRow is one row of divergences.csv
// (region,major,minor,domain,wireId,identityName,evidence).
type divergenceRow struct {
	Region       string
	Major        uint16
	Minor        uint16
	Domain       string
	WireId       uint64
	IdentityName string
	Evidence     string
}

// loadDivergences parses the Task 1 audit ledger.
func loadDivergences(path string) ([]divergenceRow, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()

	r := csv.NewReader(f)
	r.FieldsPerRecord = 7
	all, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	if len(all) == 0 {
		return nil, fmt.Errorf("%s: empty file", path)
	}
	header := all[0]
	col := func(name string) int {
		for i, h := range header {
			if h == name {
				return i
			}
		}
		return -1
	}
	regionI, majorI, minorI, domainI := col("region"), col("major"), col("minor"), col("domain")
	wireIDI, identityI, evidenceI := col("wireId"), col("identityName"), col("evidence")
	if regionI < 0 || majorI < 0 || minorI < 0 || domainI < 0 || wireIDI < 0 || identityI < 0 || evidenceI < 0 {
		return nil, fmt.Errorf("%s: header missing required column(s): %v", path, header)
	}

	var out []divergenceRow
	for i, rec := range all[1:] {
		major, err := strconv.ParseUint(rec[majorI], 10, 16)
		if err != nil {
			return nil, fmt.Errorf("%s row %d: parsing major %q: %w", path, i+2, rec[majorI], err)
		}
		minor, err := strconv.ParseUint(rec[minorI], 10, 16)
		if err != nil {
			return nil, fmt.Errorf("%s row %d: parsing minor %q: %w", path, i+2, rec[minorI], err)
		}
		wireID, err := strconv.ParseUint(rec[wireIDI], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("%s row %d: parsing wireId %q: %w", path, i+2, rec[wireIDI], err)
		}
		out = append(out, divergenceRow{
			Region:       rec[regionI],
			Major:        uint16(major),
			Minor:        uint16(minor),
			Domain:       rec[domainI],
			WireId:       wireID,
			IdentityName: rec[identityI],
			Evidence:     rec[evidenceI],
		})
	}
	return out, nil
}

// bareIdentifierRe matches a well-formed Go identifier of the shape the
// generator emits identity names as (PascalCase, letters+digits, starting
// with a letter). A divergences.csv row whose identityName does NOT match
// this shape (contains brackets, parens, spaces, slashes, ...) is a
// documentation-only row by construction -- see the "resolve-or-exclude
// contract" in semantics.go's package doc and task-4-brief.md.
var bareIdentifierRe = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9]*$`)

// resolves reports whether row's identityName is a semantic override for
// domain -- a bare Go identifier that names a known identity. names is the
// set of valid identity names for row.Domain (from identities.yaml).
func (row divergenceRow) resolves(names map[string]bool) bool {
	return bareIdentifierRe.MatchString(row.IdentityName) && names[row.IdentityName]
}

// semanticsEntry is one wireId->identityName binding (divergent override or
// excluded documentation row) as persisted in a per-version semantics.yaml.
type semanticsEntry struct {
	Domain       string `yaml:"domain"`
	WireId       uint64 `yaml:"wireId"`
	IdentityName string `yaml:"identityName"`
	Evidence     string `yaml:"evidence"`
}

// semanticsFile is the checked-in per-version source of record
// (gen/semantics/<region>_<major>_<minor>.yaml).
type semanticsFile struct {
	Provenance string           `yaml:"provenance"`
	Region     string           `yaml:"region"`
	Major      uint16           `yaml:"major"`
	Minor      uint16           `yaml:"minor"`
	Stable     string           `yaml:"stable"`
	Divergent  []semanticsEntry `yaml:"divergent"`
	Excluded   []semanticsEntry `yaml:"excluded,omitempty"`
}

// buildSemanticsFiles classifies every divergences.csv row for every
// provisioned version into the version's `divergent` (resolving semantic
// override) or `excluded` (documentation-only, logged not dropped) list --
// the resolve-or-exclude contract. names maps domain -> set of valid
// identity names (from identities.yaml), used as the resolution test.
func buildSemanticsFiles(rows []divergenceRow, names map[string]map[string]bool) map[versionKey]semanticsFile {
	out := make(map[versionKey]semanticsFile, len(provisionedVersions))
	for _, v := range provisionedVersions {
		out[v] = semanticsFile{
			Provenance: fmt.Sprintf(
				"Generated by `go run . -author-semantics` from docs/tasks/task-187-version-aware-id-semantics/audit/divergences.csv (task-187 Task 4). "+
					"DO NOT hand-edit the divergent/excluded lists -- edit divergences.csv and re-run -author-semantics. "+
					"stable:auto means: every snapshot wire id whose value equals a known canonical identity token in gen/identities.yaml auto-binds to that identity; "+
					"this file's divergent list then overlays (overrides) those auto-binds for %s.",
				v.key()),
			Region: v.Region,
			Major:  v.Major,
			Minor:  v.Minor,
			Stable: "auto",
		}
	}

	for _, row := range rows {
		v := versionKey{Region: row.Region, Major: row.Major, Minor: row.Minor}
		sf, ok := out[v]
		if !ok {
			// Not a provisioned version; buildSemanticsFiles only emits
			// files for the provisioned set, mirroring
			// audit_validate_test.go's provisioned-set gate on the CSV.
			continue
		}
		entry := semanticsEntry{Domain: row.Domain, WireId: row.WireId, IdentityName: row.IdentityName, Evidence: row.Evidence}
		if row.resolves(names[row.Domain]) {
			sf.Divergent = append(sf.Divergent, entry)
		} else {
			sf.Excluded = append(sf.Excluded, entry)
		}
		out[v] = sf
	}

	for v, sf := range out {
		sortEntries(sf.Divergent)
		sortEntries(sf.Excluded)
		out[v] = sf
	}
	return out
}

func sortEntries(entries []semanticsEntry) {
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].Domain != entries[j].Domain {
			return entries[i].Domain < entries[j].Domain
		}
		return entries[i].WireId < entries[j].WireId
	})
}

// identityNameSets loads identities.yaml and returns, per domain, the set
// of valid identity names -- the resolution test buildSemanticsFiles uses
// to distinguish a semantic override from a documentation row.
func identityNameSets(identitiesYAMLPath string) (map[string]map[string]bool, error) {
	ids, err := LoadIdentities(identitiesYAMLPath)
	if err != nil {
		return nil, err
	}
	out := make(map[string]map[string]bool)
	for _, id := range ids {
		if out[id.Domain] == nil {
			out[id.Domain] = make(map[string]bool)
		}
		out[id.Domain][id.Name] = true
	}
	return out, nil
}

// canonicalTokenNames loads identities.yaml and returns, per domain, the
// canonical-token -> identity-name table used for auto-bind.
func canonicalTokenNames(identitiesYAMLPath string) (map[string]map[uint64]string, error) {
	ids, err := LoadIdentities(identitiesYAMLPath)
	if err != nil {
		return nil, err
	}
	out := make(map[string]map[uint64]string)
	for _, id := range ids {
		if out[id.Domain] == nil {
			out[id.Domain] = make(map[uint64]string)
		}
		out[id.Domain][id.CanonicalToken] = id.Name
	}
	return out, nil
}

// runAuthorSemantics is the -author-semantics one-off (main.go Step 1):
// classify every divergences.csv row per provisioned version and write the
// 11 gen/semantics/<r>_<maj>_<min>.yaml files deterministically.
func runAuthorSemantics() error {
	rows, err := loadDivergences(divergencesCSVPath)
	if err != nil {
		return err
	}
	names, err := identityNameSets(identitiesYAMLPath)
	if err != nil {
		return err
	}
	files := buildSemanticsFiles(rows, names)

	if err := os.MkdirAll(semanticsDir, 0o755); err != nil {
		return fmt.Errorf("creating %s: %w", semanticsDir, err)
	}

	var nDiv, nExcl int
	for _, v := range provisionedVersions {
		sf := files[v]
		nDiv += len(sf.Divergent)
		nExcl += len(sf.Excluded)
		if err := writeSemanticsFile(v, sf); err != nil {
			return err
		}
	}
	fmt.Printf("wrote %d semantics YAMLs: %d divergent overrides, %d excluded documentation rows\n", len(provisionedVersions), nDiv, nExcl)
	return nil
}

func writeSemanticsFile(v versionKey, sf semanticsFile) error {
	path := fmt.Sprintf("%s/%s.yaml", semanticsDir, v.key())
	var b strings.Builder
	enc := yaml.NewEncoder(&b)
	enc.SetIndent(2)
	if err := enc.Encode(sf); err != nil {
		return fmt.Errorf("marshalling %s: %w", path, err)
	}
	if err := enc.Close(); err != nil {
		return fmt.Errorf("closing yaml encoder for %s: %w", path, err)
	}
	header := fmt.Sprintf(
		"# Code generated by `go run . -author-semantics`; DO NOT EDIT directly -- edit\n"+
			"# docs/tasks/task-187-version-aware-id-semantics/audit/divergences.csv and re-run.\n"+
			"# excluded (documentation): %d row(s) -- Big Bang v0.92<->v0.95 reorg rows, UNVERIFIED\n"+
			"# rows, and the DualBlade (job token 430) gap; these are NOT applied as wire<->identity\n"+
			"# overrides, only kept here as an audit trail. See excluded: below and\n"+
			"# docs/tasks/task-187-version-aware-id-semantics/audit/README.md.\n",
		len(sf.Excluded))
	if err := os.WriteFile(path, []byte(header+b.String()), 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return nil
}

// loadSemanticsFile reads the checked-in per-version semantics.yaml.
func loadSemanticsFile(region string, major, minor uint16) (semanticsFile, error) {
	v := versionKey{Region: region, Major: major, Minor: minor}
	path := fmt.Sprintf("%s/%s.yaml", semanticsDir, v.key())
	b, err := os.ReadFile(path)
	if err != nil {
		return semanticsFile{}, fmt.Errorf("semantics: no pinned semantics.yaml for %s %d.%d (%s): %w -- run `go run . -author-semantics`", region, major, minor, path, err)
	}
	var sf semanticsFile
	if err := yaml.Unmarshal(b, &sf); err != nil {
		return semanticsFile{}, fmt.Errorf("semantics: parsing %s: %w", path, err)
	}
	if sf.Region != region || sf.Major != major || sf.Minor != minor {
		return semanticsFile{}, fmt.Errorf("semantics: %s: embedded identity (%s %d.%d) does not match requested (%s %d.%d)",
			path, sf.Region, sf.Major, sf.Minor, region, major, minor)
	}
	return sf, nil
}

// SemanticsMap is the join result of one version's wireId<->identityName
// bindings, both domains. Keys are the domain's native wire-id width
// (skill.Id is uint32, job.Id is uint16); values are identity names (Go
// identifiers valid in skill/job's identities_gen.go const block).
//
// JobByName/SkillByName are the inverse (name->wireId) view -- a thin index
// over the same data, added for task-187 Task 5 so callers (the
// availability generator, its tests) can ask "is identity X present in this
// version's semantics" by name without re-inverting Job/Skill themselves.
type SemanticsMap struct {
	Skill map[uint32]string
	Job   map[uint16]string

	SkillByName map[string]uint32
	JobByName   map[string]uint16
}

// BuildSemantics joins one version's pinned wzsnapshot with its checked-in
// semantics.yaml: auto-bind every snapshot wire id whose value equals a
// canonical identity token, then overlay the divergent list (overriding
// auto-binds). This is the ONLY place per-version wire<->identity
// divergence is resolved -- see the package doc.
func BuildSemantics(region string, major, minor uint16) (SemanticsMap, error) {
	sf, err := loadSemanticsFile(region, major, minor)
	if err != nil {
		return SemanticsMap{}, err
	}

	skillIds, jobIds, _, err := wzsnapshot.LoadSnapshot(region, major, minor)
	if err != nil {
		return SemanticsMap{}, err
	}
	skillSet := make(map[uint64]bool, len(skillIds))
	for _, id := range skillIds {
		skillSet[uint64(id)] = true
	}
	jobSet := make(map[uint64]bool, len(jobIds))
	for _, id := range jobIds {
		jobSet[uint64(id)] = true
	}

	tokenNames, err := canonicalTokenNames(identitiesYAMLPath)
	if err != nil {
		return SemanticsMap{}, err
	}
	validNames, err := identityNameSets(identitiesYAMLPath)
	if err != nil {
		return SemanticsMap{}, err
	}

	skillMap := make(map[uint32]string, len(skillIds))
	for _, id := range skillIds {
		if name, ok := tokenNames["skill"][uint64(id)]; ok {
			skillMap[id] = name
		}
	}
	jobMap := make(map[uint16]string, len(jobIds))
	for _, id := range jobIds {
		if name, ok := tokenNames["job"][uint64(id)]; ok {
			jobMap[id] = name
		}
	}

	if err := applyDivergent(region, major, minor, sf.Divergent, skillSet, jobSet, validNames, skillMap, jobMap); err != nil {
		return SemanticsMap{}, err
	}

	skillByName := make(map[string]uint32, len(skillMap))
	for w, n := range skillMap {
		skillByName[n] = w
	}
	jobByName := make(map[string]uint16, len(jobMap))
	for w, n := range jobMap {
		jobByName[n] = w
	}

	return SemanticsMap{Skill: skillMap, Job: jobMap, SkillByName: skillByName, JobByName: jobByName}, nil
}

// applyDivergent overlays a version's divergent list onto skillMap/jobMap.
// These entries are already filtered to resolving semantic overrides by
// -author-semantics, but this re-validates every one (never trust a
// checked-in file blindly): an override must carry evidence, must name a
// known identity, must target a wire id actually present in this version's
// snapshot, and must not repeat a wire id already bound earlier in the SAME
// domain's divergent list for this version (the brief's Step 4 "no
// duplicate wireId" rule).
//
// Validation accumulates into local overlay maps first and is only merged
// into skillMap/jobMap after every entry in divergent has passed -- so a
// failing entry (including the duplicate-wireId case) can never leave a
// partial write behind, not even the earlier, individually-valid entries
// from the same failing batch. No silent overwrite, no silent partial
// apply. Split out from BuildSemantics so the duplicate-wireId path is
// directly unit-testable without needing a real
// wzsnapshot/identities.yaml/CSV fixture.
func applyDivergent(region string, major, minor uint16, divergent []semanticsEntry, skillSet, jobSet map[uint64]bool, validNames map[string]map[string]bool, skillMap map[uint32]string, jobMap map[uint16]string) error {
	seenSkill := make(map[uint64]string, len(divergent))
	seenJob := make(map[uint64]string, len(divergent))
	skillOverlay := make(map[uint32]string, len(divergent))
	jobOverlay := make(map[uint16]string, len(divergent))

	for _, e := range divergent {
		if e.Evidence == "" {
			return fmt.Errorf("semantics %s %d.%d: divergent %s wireId %d (%s): missing evidence citation", region, major, minor, e.Domain, e.WireId, e.IdentityName)
		}
		if !bareIdentifierRe.MatchString(e.IdentityName) || !validNames[e.Domain][e.IdentityName] {
			return fmt.Errorf("semantics %s %d.%d: divergent %s wireId %d binds to unknown identity %q", region, major, minor, e.Domain, e.WireId, e.IdentityName)
		}
		switch e.Domain {
		case "skill":
			if !skillSet[e.WireId] {
				return fmt.Errorf("semantics %s %d.%d: divergent skill wireId %d (%s) absent from wzsnapshot", region, major, minor, e.WireId, e.IdentityName)
			}
			if prev, dup := seenSkill[e.WireId]; dup {
				return fmt.Errorf("semantics %s %d.%d: duplicate divergent skill wireId %d: bound to both %q and %q", region, major, minor, e.WireId, prev, e.IdentityName)
			}
			seenSkill[e.WireId] = e.IdentityName
			skillOverlay[uint32(e.WireId)] = e.IdentityName
		case "job":
			if !jobSet[e.WireId] {
				return fmt.Errorf("semantics %s %d.%d: divergent job wireId %d (%s) absent from wzsnapshot", region, major, minor, e.WireId, e.IdentityName)
			}
			if prev, dup := seenJob[e.WireId]; dup {
				return fmt.Errorf("semantics %s %d.%d: duplicate divergent job wireId %d: bound to both %q and %q", region, major, minor, e.WireId, prev, e.IdentityName)
			}
			seenJob[e.WireId] = e.IdentityName
			jobOverlay[uint16(e.WireId)] = e.IdentityName
		default:
			return fmt.Errorf("semantics %s %d.%d: divergent entry has unknown domain %q", region, major, minor, e.Domain)
		}
	}

	for w, n := range skillOverlay {
		skillMap[w] = n
	}
	for w, n := range jobOverlay {
		jobMap[w] = n
	}
	return nil
}

// pascalVersionName returns the exported-constructor-friendly PascalCase
// name for v (task-187 Task 6), e.g. {"gms",48,1} -> "GMS481",
// {"jms",185,1} -> "JMS1851". Deterministic and collision-free across
// provisionedVersions -- region+major already disambiguates every entry.
func pascalVersionName(v versionKey) string {
	return strings.ToUpper(v.Region) + strconv.Itoa(int(v.Major)) + strconv.Itoa(int(v.Minor))
}

// versionGenPackage renders one domain's generated Go source for a version:
// the wireId->Identity and Identity->wireId maps, the task-187 Task 5
// available/names maps, the newSet_<key> constructor, and (task-187 Task 6)
// the exported NewSet<PascalVersion> wrapper constants.For's generated
// registry calls. m is the wireId->identityName join for that domain only
// (widened to uint64 so one function serves both skill.Id (uint32) and
// job.Id (uint16) -- the emitted Go source narrows back via the domain's
// real Id type). v is the version key (e.g. {"gms",48,1}).
// availableNames is the set of identity names (within m) that are
// release-available at this version (task-187 Task 5, computeAvailable);
// displayNames is the domain's name->displayName index from
// identities.yaml, used to populate the names_<key> map for every identity
// present at this version (available or not -- Name() must work regardless
// of Available()).
func versionGenPackage(domain string, v versionKey, m map[uint64]string, availableNames map[string]bool, displayNames map[string]string) (string, error) {
	key := v.key()
	type kv struct {
		wireId uint64
		name   string
	}
	entries := make([]kv, 0, len(m))
	revCheck := make(map[string]uint64, len(m))
	for w, n := range m {
		entries = append(entries, kv{wireId: w, name: n})
		if prev, ok := revCheck[n]; ok && prev != w {
			return "", fmt.Errorf("version %s domain %s: identity %s bound to two different wire ids (%d and %d)", key, domain, n, prev, w)
		}
		revCheck[n] = w
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].wireId < entries[j].wireId })

	var b strings.Builder
	b.WriteString("// Code generated by gen; DO NOT EDIT.\n\n")
	fmt.Fprintf(&b, "package %s\n\n", domain)

	fmt.Fprintf(&b, "var wireToIdentity_%s = map[Id]Identity{\n", key)
	for _, e := range entries {
		fmt.Fprintf(&b, "\t%d: %s,\n", e.wireId, e.name)
	}
	b.WriteString("}\n\n")

	fmt.Fprintf(&b, "var identityToWire_%s = map[Identity]Id{\n", key)
	for _, e := range entries {
		fmt.Fprintf(&b, "\t%s: %d,\n", e.name, e.wireId)
	}
	b.WriteString("}\n\n")

	fmt.Fprintf(&b, "var available_%s = map[Identity]struct{}{\n", key)
	for _, e := range entries {
		if availableNames[e.name] {
			fmt.Fprintf(&b, "\t%s: {},\n", e.name)
		}
	}
	b.WriteString("}\n\n")

	fmt.Fprintf(&b, "var names_%s = map[Identity]string{\n", key)
	for _, e := range entries {
		fmt.Fprintf(&b, "\t%s: %q,\n", e.name, displayNames[e.name])
	}
	b.WriteString("}\n\n")

	fmt.Fprintf(&b, "func newSet_%s() Set {\n", key)
	fmt.Fprintf(&b, "\treturn Set{byWire: wireToIdentity_%s, byIdentity: identityToWire_%s, available: available_%s, names: names_%s}\n", key, key, key, key)
	b.WriteString("}\n\n")

	// task-187 Task 6: exported wrapper so constants/registry_gen.go (a
	// different package) can reference this version's Set without the
	// unexported newSet_<key> constructor leaking package-private access.
	exported := pascalVersionName(v)
	fmt.Fprintf(&b, "// NewSet%s returns this version's identity Set (generated; task-187 Task 6).\n", exported)
	fmt.Fprintf(&b, "func NewSet%s() Set {\n", exported)
	fmt.Fprintf(&b, "\treturn newSet_%s()\n", key)
	b.WriteString("}\n")

	return b.String(), nil
}

// EmitSemantics builds one version's semantics map and renders the
// generated Go source for both domains: skill/version_<key>_gen.go and
// job/version_<key>_gen.go.
func EmitSemantics(region string, major, minor uint16) (skillGo, jobGo string, err error) {
	m, err := BuildSemantics(region, major, minor)
	if err != nil {
		return "", "", err
	}
	v := versionKey{Region: region, Major: major, Minor: minor}

	ids, err := LoadIdentities(identitiesYAMLPath)
	if err != nil {
		return "", "", err
	}
	matrix, err := loadReleaseMatrix(region, major, minor)
	if err != nil {
		return "", "", err
	}
	availJob, availSkill := computeAvailable(m, ids, matrix)
	skillDisplay := make(map[string]string)
	jobDisplay := make(map[string]string)
	for _, id := range ids {
		switch id.Domain {
		case "skill":
			skillDisplay[id.Name] = id.DisplayName
		case "job":
			jobDisplay[id.Name] = id.DisplayName
		}
	}

	skillWide := make(map[uint64]string, len(m.Skill))
	for w, n := range m.Skill {
		skillWide[uint64(w)] = n
	}
	jobWide := make(map[uint64]string, len(m.Job))
	for w, n := range m.Job {
		jobWide[uint64(w)] = n
	}

	skillGo, err = versionGenPackage("skill", v, skillWide, availSkill, skillDisplay)
	if err != nil {
		return "", "", err
	}
	jobGo, err = versionGenPackage("job", v, jobWide, availJob, jobDisplay)
	if err != nil {
		return "", "", err
	}
	return skillGo, jobGo, nil
}

// emitAllVersionFiles runs EmitSemantics for every provisioned version and
// gofmt-formats + writes (or, in check mode, diffs) the resulting 22 files.
// Used by main.go's default `go run .` and `-check` paths.
func emitAllVersionFiles(check bool, gofmtOrRaw func(string) ([]byte, error), checkDrift func(string, []byte) error, writeFile func(string, []byte) error) (int, error) {
	n := 0
	for _, v := range provisionedVersions {
		skillGo, jobGo, err := EmitSemantics(v.Region, v.Major, v.Minor)
		if err != nil {
			return n, fmt.Errorf("EmitSemantics(%s,%d,%d): %w", v.Region, v.Major, v.Minor, err)
		}
		skillFmt, err := gofmtOrRaw(skillGo)
		if err != nil {
			return n, fmt.Errorf("formatting skill/version_%s_gen.go: %w", v.key(), err)
		}
		jobFmt, err := gofmtOrRaw(jobGo)
		if err != nil {
			return n, fmt.Errorf("formatting job/version_%s_gen.go: %w", v.key(), err)
		}

		skillPath := fmt.Sprintf("%s/version_%s_gen.go", skillVersionGenDir, v.key())
		jobPath := fmt.Sprintf("%s/version_%s_gen.go", jobVersionGenDir, v.key())

		if check {
			if err := checkDrift(skillPath, skillFmt); err != nil {
				return n, err
			}
			if err := checkDrift(jobPath, jobFmt); err != nil {
				return n, err
			}
		} else {
			if err := writeFile(skillPath, skillFmt); err != nil {
				return n, fmt.Errorf("writing %s: %w", skillPath, err)
			}
			if err := writeFile(jobPath, jobFmt); err != nil {
				return n, fmt.Errorf("writing %s: %w", jobPath, err)
			}
		}
		n += 2
	}
	return n, nil
}
