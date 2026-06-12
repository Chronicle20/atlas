# Evidence-Graded Packet Coverage Matrix Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the operation registry, evidence ledger, byte-test linkage scanner, and `matrix` subcommand in `tools/packet-audit` so that every (packet × direction × version) cell renders as a machine-graded `verified / partial / incomplete / n-a / conflict` state in a generated `docs/packets/audits/STATUS.md`, plus the playbook/skill/agent that promote cells and the `discover-ops` capability that grows the registry from the IDBs.

**Architecture:** Four new internal packages (`opregistry`, `seedcsv`, `evidence`, `marker`) plus a `matrix` package that joins registry applicability, latest audit-report verdicts, evidence records, tier membership, and test linkage into STATUS.md/status.json with a `--check` CI mode. Four new subcommands (`registry`, `matrix`, `evidence`, `discover-ops`) follow the existing flag-dispatch pattern in `cmd/root.go`. The grading engine is written **once, complete** in Phase 1 with empty evidence/marker/tier inputs; Phases 2–3 only add loaders and wire the inputs in.

**Tech Stack:** Go 1.24 (`tools/packet-audit` module, gaining its first external dep `gopkg.in/yaml.v3`), stdlib `encoding/csv`/`encoding/json`/`crypto/sha256`, existing `internal/csv`, `internal/report`, `internal/template`, `internal/idasrc` packages, table-driven tests + golden files in `testdata/` (existing convention).

**Read first:** `context.md` in this folder — it locks the design-ambiguity resolutions (hash basis, no-date stamp, conflict-rule refinement, tier expansion semantics, sub-struct section join) and the verified file:line references this plan depends on.

---

## Out of scope (explicit)

- **Design Phase 6 (tier-1 fixture campaign)** — per design §12 it is split into family-sized follow-up tasks with matrix-cell scope. This plan only makes those campaigns *possible* (markers, evidence, playbook, agent).
- Live-wire capture/replay; semantic/branch-aware diff engine (design §3).
- Auditing versions beyond the five baselines.

## Execution constraints

- **Worktree:** all work happens in `.worktrees/task-085-packet-audit-coverage-matrix/` on branch `task-085-packet-audit-coverage-matrix`. Every subagent prompt must `cd` there first and verify `git branch --show-current` after each commit.
- **Live IDA:** Tasks 5.6 and 5.7 (and only those) need live ida-pro-mcp instances. They are operator-gated checkpoints — pause and ask the user to bring instances up. Everything else runs from checked-in exports and fixtures.
- **Module verification** after each phase: from `tools/packet-audit/`: `go test -race ./... && go vet ./...`. No service `go.mod` is touched, so no `docker buildx bake` is required; run `tools/redis-key-guard.sh` once before PR (no Redis code here, it must stay clean).

---

# Phase 1 — Operation registry + matrix generator (read-only)

## Task 1.1: Add the yaml.v3 dependency

**Files:**
- Modify: `tools/packet-audit/go.mod`
- Modify: `go.work.sum` (repo root, regenerated)

The tool is currently zero-dep. The design mandates YAML for registry/evidence/tiers files (§5.1, §6.1, §8); `gopkg.in/yaml.v3` is the standard choice.

- [ ] **Step 1: Add the dependency**

```bash
cd tools/packet-audit
go get gopkg.in/yaml.v3@v3.0.1
cd ../..
go work sync
```

- [ ] **Step 2: Verify the workspace still builds**

```bash
cd tools/packet-audit && go build ./... && go test ./... && cd ../..
```
Expected: PASS (no code uses yaml yet; `go.mod` now lists `gopkg.in/yaml.v3 v3.0.1`).

- [ ] **Step 3: Commit**

```bash
git add tools/packet-audit/go.mod tools/packet-audit/go.sum go.work.sum
git commit -m "task-085: add yaml.v3 dependency to packet-audit"
```

## Task 1.2: `internal/opregistry` — registry types, loader, applicability

**Files:**
- Create: `tools/packet-audit/internal/opregistry/opregistry.go`
- Create: `tools/packet-audit/internal/opregistry/opregistry_test.go`
- Create: `tools/packet-audit/internal/opregistry/testdata/good_version.yaml`
- Create: `tools/packet-audit/internal/opregistry/testdata/dup_version.yaml`

The registry is one YAML file per version under `docs/packets/registry/`, schema per design §5.1.

- [ ] **Step 1: Write the failing tests**

`opregistry_test.go`:

```go
package opregistry

import (
	"path/filepath"
	"testing"
)

func TestLoadVersion(t *testing.T) {
	v, err := LoadVersion(filepath.Join("testdata", "good_version.yaml"))
	if err != nil {
		t.Fatalf("LoadVersion: %v", err)
	}
	e, ok := v.Lookup("LOGIN_STATUS", DirClientbound)
	if !ok {
		t.Fatalf("LOGIN_STATUS clientbound not found")
	}
	if e.Opcode != 0x000 || e.FName != "CLogin::OnCheckPasswordResult" {
		t.Errorf("entry = %+v", e)
	}
	if e.Provenance != "csv-import" {
		t.Errorf("provenance = %q", e.Provenance)
	}
	// fname_alts round-trip (multiline CSV cells)
	alt, _ := v.Lookup("SERVERLIST_REREQUEST", DirServerbound)
	if len(alt.FNameAlts) != 1 || alt.FNameAlts[0] != "CLogin::ChangeStepImmediate" {
		t.Errorf("fname_alts = %v", alt.FNameAlts)
	}
}

func TestLoadVersionDuplicate(t *testing.T) {
	_, err := LoadVersion(filepath.Join("testdata", "dup_version.yaml"))
	if err == nil {
		t.Fatal("expected duplicate (op,direction) error")
	}
}

func TestRegistryApplicability(t *testing.T) {
	r := Registry{Versions: map[string]*VersionFile{
		"gms_v83": mustLoad(t, "good_version.yaml"),
	}}
	if got := r.Applicability("LOGIN_STATUS", DirClientbound, "gms_v83"); got != Present {
		t.Errorf("present op = %v", got)
	}
	if got := r.Applicability("NOPE", DirClientbound, "gms_v83"); got != Absent {
		t.Errorf("missing op in existing file = %v", got)
	}
	if got := r.Applicability("LOGIN_STATUS", DirClientbound, "gms_v99"); got != Unknown {
		t.Errorf("missing version file = %v", got)
	}
}

func mustLoad(t *testing.T, name string) *VersionFile {
	t.Helper()
	v, err := LoadVersion(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return v
}
```

`testdata/good_version.yaml`:

```yaml
- op: LOGIN_STATUS
  direction: clientbound
  opcode: 0x000
  fname: "CLogin::OnCheckPasswordResult"
  provenance: csv-import
- op: SERVERLIST_REREQUEST
  direction: serverbound
  opcode: 0x004
  fname: "CLogin::Init"
  fname_alts:
    - "CLogin::ChangeStepImmediate"
  provenance: csv-import
- op: SPAWN_MONSTER
  direction: clientbound
  opcode: 0x0EC
  fname: "CMobPool::OnMobEnterField"
  provenance: ida-discovered
  ida:
    address: 0x5e1230
```

`testdata/dup_version.yaml`:

```yaml
- op: LOGIN_STATUS
  direction: clientbound
  opcode: 0x000
  fname: "CLogin::OnCheckPasswordResult"
  provenance: csv-import
- op: LOGIN_STATUS
  direction: clientbound
  opcode: 0x001
  fname: "CLogin::OnCheckPasswordResult"
  provenance: manual
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd tools/packet-audit && go test ./internal/opregistry/
```
Expected: FAIL (package does not exist).

- [ ] **Step 3: Implement `opregistry.go`**

```go
// Package opregistry owns the per-version operation universe: which packet
// operations exist in which client version, with provenance (design §5.1).
package opregistry

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"
)

type Direction string

const (
	DirClientbound Direction = "clientbound"
	DirServerbound Direction = "serverbound"
)

// Applicability of an (op, direction) in a version.
type Applicability int

const (
	Unknown Applicability = iota // no registry file for the version
	Absent                       // file exists, op not listed
	Present                      // op listed
)

type IDARef struct {
	Address uint64 `yaml:"address"`
}

type Entry struct {
	Op         string    `yaml:"op"`
	Direction  Direction `yaml:"direction"`
	Opcode     int       `yaml:"opcode"`
	FName      string    `yaml:"fname"`
	FNameAlts  []string  `yaml:"fname_alts,omitempty"`
	Provenance string    `yaml:"provenance"` // csv-import | ida-discovered | manual
	IDA        *IDARef   `yaml:"ida,omitempty"`
	Note       string    `yaml:"note,omitempty"`
}

type VersionFile struct {
	Entries []Entry
	byKey   map[string]Entry // "op|direction"
}

func key(op string, dir Direction) string { return op + "|" + string(dir) }

func validProvenance(p string) bool {
	return p == "csv-import" || p == "ida-discovered" || p == "manual"
}

// LoadVersion reads one registry YAML and validates schema + uniqueness.
func LoadVersion(path string) (*VersionFile, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var entries []Entry
	if err := yaml.Unmarshal(raw, &entries); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	vf := &VersionFile{Entries: entries, byKey: make(map[string]Entry, len(entries))}
	for i, e := range entries {
		if e.Op == "" || (e.Direction != DirClientbound && e.Direction != DirServerbound) {
			return nil, fmt.Errorf("%s entry %d: missing op or invalid direction %q", path, i, e.Direction)
		}
		if !validProvenance(e.Provenance) {
			return nil, fmt.Errorf("%s entry %d (%s): invalid provenance %q", path, i, e.Op, e.Provenance)
		}
		if e.Provenance == "ida-discovered" && e.IDA == nil {
			return nil, fmt.Errorf("%s entry %d (%s): ida-discovered without ida.address", path, i, e.Op)
		}
		k := key(e.Op, e.Direction)
		if _, dup := vf.byKey[k]; dup {
			return nil, fmt.Errorf("%s: duplicate (op,direction) %s/%s", path, e.Op, e.Direction)
		}
		vf.byKey[k] = e
	}
	return vf, nil
}

func (v *VersionFile) Lookup(op string, dir Direction) (Entry, bool) {
	e, ok := v.byKey[key(op, dir)]
	return e, ok
}

// ByFName returns the entry whose fname (or any fname_alt) matches.
func (v *VersionFile) ByFName(fname string, dir Direction) (Entry, bool) {
	for _, e := range v.Entries {
		if e.Direction != dir {
			continue
		}
		if e.FName == fname {
			return e, true
		}
		for _, a := range e.FNameAlts {
			if a == fname {
				return e, true
			}
		}
	}
	return Entry{}, false
}

// Registry is the loaded set of version files.
type Registry struct {
	Versions map[string]*VersionFile // version key -> file (nil entry means file missing)
}

// LoadDir loads every <version>.yaml present in dir for the given version keys.
// Missing files are recorded as absent from the map (Applicability => Unknown).
func LoadDir(dir string, versionKeys []string) (Registry, error) {
	r := Registry{Versions: make(map[string]*VersionFile)}
	for _, vk := range versionKeys {
		p := filepath.Join(dir, vk+".yaml")
		if _, err := os.Stat(p); os.IsNotExist(err) {
			continue
		}
		vf, err := LoadVersion(p)
		if err != nil {
			return Registry{}, err
		}
		r.Versions[vk] = vf
	}
	return r, nil
}

func (r Registry) Applicability(op string, dir Direction, versionKey string) Applicability {
	vf, ok := r.Versions[versionKey]
	if !ok {
		return Unknown
	}
	if _, present := vf.Lookup(op, dir); present {
		return Present
	}
	return Absent
}

// AllOps returns the union of (op, direction) across all loaded versions,
// sorted for deterministic iteration: clientbound first, then by op name.
func (r Registry) AllOps() []struct {
	Op  string
	Dir Direction
} {
	seen := map[string]struct {
		Op  string
		Dir Direction
	}{}
	for _, vf := range r.Versions {
		for _, e := range vf.Entries {
			seen[key(e.Op, e.Direction)] = struct {
				Op  string
				Dir Direction
			}{e.Op, e.Direction}
		}
	}
	out := make([]struct {
		Op  string
		Dir Direction
	}, 0, len(seen))
	for _, v := range seen {
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Dir != out[j].Dir {
			return out[i].Dir == DirClientbound
		}
		return out[i].Op < out[j].Op
	})
	return out
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/opregistry/ -v
```
Expected: PASS (3 tests).

- [ ] **Step 5: Commit**

```bash
cd ../..
git add tools/packet-audit/internal/opregistry/
git commit -m "task-085: opregistry package — registry schema, loader, applicability"
```

## Task 1.3: `internal/seedcsv` — CSV seed parser (op + per-version index/opcode pairs)

**Files:**
- Create: `tools/packet-audit/internal/seedcsv/seedcsv.go`
- Create: `tools/packet-audit/internal/seedcsv/seedcsv_test.go`
- Create: `tools/packet-audit/internal/seedcsv/testdata/clientbound_excerpt.csv`
- Create: `tools/packet-audit/internal/seedcsv/testdata/serverbound_excerpt.csv`

The existing `internal/csv` parser (`csv.go:61-123`) drops the `Op` name column and the index cells — it only keeps opcodes keyed by FName. Seeding needs **presence** semantics, which live in the *(index, opcode)* pair. Verified CSV layout (from `docs/packets/MapleStory Ops - ClientBound.csv` line 1):

```
Op,FName,Index,GMS v12,,GMS v48,,GMS v61,...,GMS v95,,GMS v111,,JMS v185
```

Each version's pair is **(the column immediately BEFORE the labeled column, the labeled column)** = (index, opcode). For the first version the index column is the one literally named `Index`. Presence rule (verified against rows `LOGIN_STATUS` — present at v83 with index `0` opcode `0x000` — and `ACCOUNT_INFO` — absent at JMS with empty index + `0x000`):

> present ⇔ index cell non-empty OR opcode != 0

Multiline FName cells (RFC 4180 quoted, e.g. ServerBound row 5 `SERVERLIST_REREQUEST` = `"CLogin::Init\nCLogin::ChangeStepImmediate"`) split into fname (first line) + fname_alts (rest). Rows with an Op name but empty FName (e.g. `GUEST_LOGIN`) are kept with `fname: ""`.

- [ ] **Step 1: Create testdata excerpts (verbatim from the real CSVs)**

`testdata/clientbound_excerpt.csv`:

```csv
Op,FName,Index,GMS v12,,GMS v48,,GMS v61,,GMS v72,,GMS v79,,GMS v83,,GMS v87,,GMS v92,,GMS v95,,GMS v111,,JMS v185
LOGIN_STATUS,CLogin::OnCheckPasswordResult,1,0x001,,0x000,,0x000,,0x000,,0x000,0,0x000,0,0x000,0,0x000,0,0x000,0,0x000,0,0x000
GUEST_ID_LOGIN,CLogin::OnGuestIDLoginResult,,0x000,,0x000,,0x000,,0x000,,0x000,1,0x001,1,0x001,1,0x001,1,0x001,1,0x001,1,0x001
ACCOUNT_INFO,CLogin::OnAccountInfoResult,,0x000,,0x000,,0x000,,0x000,,0x000,2,0x002,2,0x002,2,0x002,2,0x002,2,0x002,,0x000
```

`testdata/serverbound_excerpt.csv`:

```csv
Op,FName,Index,GMS v12,,GMS v83,,GMS v87,,GMS v92,,GMS v95,,GMS v111,,JMS v185
LOGIN_PASSWORD,CLogin::SendCheckPasswordPacket,1,0x001,1,0x001,1,0x001,1,0x001,1,0x001,21,0x015,1,0x001
GUEST_LOGIN,,,0x000,2,0x002,2,0x002,2,0x002,2,0x002,22,0x016,2,0x002
SERVERLIST_REREQUEST,"CLogin::Init
CLogin::ChangeStepImmediate",,0x000,4,0x004,4,0x004,4,0x004,4,0x004,24,0x018,3,0x003
```

- [ ] **Step 2: Write the failing tests**

`seedcsv_test.go`:

```go
package seedcsv

import (
	"path/filepath"
	"testing"
)

func TestLoadClientbound(t *testing.T) {
	rows, err := Load(filepath.Join("testdata", "clientbound_excerpt.csv"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("rows = %d, want 3", len(rows))
	}

	ls := rows[0]
	if ls.Op != "LOGIN_STATUS" || ls.FName != "CLogin::OnCheckPasswordResult" {
		t.Errorf("row0 = %+v", ls)
	}
	// Index-column quirk: first version pair uses the named Index column.
	v12 := ls.Versions["GMS:12"]
	if !v12.Present || v12.Opcode != 0x001 {
		t.Errorf("v12 = %+v", v12)
	}
	// Present with opcode 0x000 but non-empty index.
	v83 := ls.Versions["GMS:83"]
	if !v83.Present || v83.Opcode != 0x000 {
		t.Errorf("v83 = %+v (presence must come from index cell)", v83)
	}
	// Absent: empty index + 0x000.
	v48 := ls.Versions["GMS:48"]
	if v48.Present {
		t.Errorf("v48 should be absent: %+v", v48)
	}

	// ACCOUNT_INFO absent in JMS185.
	ai := rows[2]
	if ai.Versions["JMS:185"].Present {
		t.Errorf("ACCOUNT_INFO JMS:185 should be absent")
	}
}

func TestLoadServerboundQuirks(t *testing.T) {
	rows, err := Load(filepath.Join("testdata", "serverbound_excerpt.csv"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	// Empty FName kept.
	gl := rows[1]
	if gl.Op != "GUEST_LOGIN" || gl.FName != "" {
		t.Errorf("row1 = %+v", gl)
	}
	if !gl.Versions["GMS:83"].Present || gl.Versions["GMS:83"].Opcode != 0x002 {
		t.Errorf("GUEST_LOGIN v83 = %+v", gl.Versions["GMS:83"])
	}
	// Multiline FName splits into fname + alts.
	sr := rows[2]
	if sr.FName != "CLogin::Init" || len(sr.FNameAlts) != 1 || sr.FNameAlts[0] != "CLogin::ChangeStepImmediate" {
		t.Errorf("multiline fname = %q alts=%v", sr.FName, sr.FNameAlts)
	}
}

func TestLoadBadOpcodeFailsLoudly(t *testing.T) {
	_, err := LoadFromString("Op,FName,Index,GMS v83\nX,CFoo::Bar,1,zzz\n")
	if err == nil {
		t.Fatal("expected loud failure with row number")
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

```bash
go test ./internal/seedcsv/
```
Expected: FAIL (package does not exist).

- [ ] **Step 4: Implement `seedcsv.go`**

```go
// Package seedcsv parses the hand-maintained MapleStory Ops CSVs with full
// presence semantics (per-version index+opcode pairs) for the one-time
// registry seeding (design §5.1 "CSV seeding").
//
// CSV shape: Op,FName,Index,<REGION vN>,,<REGION vN>,, ...
// A version's pair is (column before the labeled column, labeled column) =
// (index, opcode). present ⇔ index non-empty OR opcode != 0.
package seedcsv

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

type Cell struct {
	Present bool
	Opcode  int
}

type Row struct {
	Op        string
	FName     string
	FNameAlts []string
	Versions  map[string]Cell // "REGION:major" -> cell
}

func Load(path string) ([]Row, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return load(f, path)
}

func LoadFromString(s string) ([]Row, error) {
	return load(strings.NewReader(s), "<string>")
}

func load(src io.Reader, name string) ([]Row, error) {
	r := csv.NewReader(src)
	r.FieldsPerRecord = -1
	records, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	if len(records) < 2 {
		return nil, fmt.Errorf("%s: missing header or data rows", name)
	}
	header := records[0]

	// versionCols[i] = "REGION:major" when header[i] is the OPCODE column of
	// that version; the index column is i-1.
	versionCols := make([]string, len(header))
	for i, col := range header {
		if region, major, ok := splitRegionMajor(strings.TrimSpace(col)); ok {
			versionCols[i] = fmt.Sprintf("%s:%d", region, major)
		}
	}

	var out []Row
	for rowNum, rec := range records[1:] {
		if len(rec) < 2 {
			continue
		}
		op := strings.TrimSpace(rec[0])
		if op == "" || strings.HasPrefix(op, "#") {
			continue
		}
		fnameLines := strings.Split(strings.TrimSpace(rec[1]), "\n")
		row := Row{Op: op, FName: strings.TrimSpace(fnameLines[0]), Versions: map[string]Cell{}}
		for _, alt := range fnameLines[1:] {
			if a := strings.TrimSpace(alt); a != "" {
				row.FNameAlts = append(row.FNameAlts, a)
			}
		}
		for i, vk := range versionCols {
			if vk == "" || i >= len(rec) {
				continue
			}
			opcodeCell := strings.TrimSpace(rec[i])
			indexCell := ""
			if i-1 >= 0 && i-1 < len(rec) {
				indexCell = strings.TrimSpace(rec[i-1])
			}
			if opcodeCell == "" && indexCell == "" {
				continue // ragged short row tail
			}
			opcode := 0
			if opcodeCell != "" {
				opcode, err = parseOpcode(opcodeCell)
				if err != nil {
					return nil, fmt.Errorf("%s row %d (%s) col %d: unparseable opcode %q",
						name, rowNum+2, op, i+1, opcodeCell)
				}
			}
			row.Versions[vk] = Cell{Present: indexCell != "" || opcode != 0, Opcode: opcode}
		}
		out = append(out, row)
	}
	return out, nil
}

// splitRegionMajor mirrors internal/csv: "GMS v95"/"JMS v185" and bare "v95"
// (assumed GMS) are version labels; everything else (Op, FName, Index, "") is not.
func splitRegionMajor(col string) (string, uint16, bool) {
	parts := strings.Fields(col)
	var region, vpart string
	switch len(parts) {
	case 1:
		region, vpart = "GMS", parts[0]
	case 2:
		region, vpart = parts[0], parts[1]
	default:
		return "", 0, false
	}
	if !strings.HasPrefix(vpart, "v") && !strings.HasPrefix(vpart, "V") {
		return "", 0, false
	}
	n, err := strconv.ParseUint(vpart[1:], 10, 16)
	if err != nil {
		return "", 0, false
	}
	return region, uint16(n), true
}

func parseOpcode(s string) (int, error) {
	if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
		n, err := strconv.ParseInt(s[2:], 16, 32)
		return int(n), err
	}
	n, err := strconv.ParseInt(s, 10, 32)
	return int(n), err
}
```

Note the deliberate difference from `internal/csv`: bad opcode cells are a **loud error with row number** (design §13 bullet 1), not a silent skip.

- [ ] **Step 5: Run tests to verify they pass**

```bash
go test ./internal/seedcsv/ -v
```
Expected: PASS (3 tests). Watch `TestLoadClientbound/v83`: if it fails with `Present=false`, the index column offset is wrong — the index is at `i-1`, the labeled column `i` is the opcode.

- [ ] **Step 6: Commit**

```bash
cd ../..
git add tools/packet-audit/internal/seedcsv/
git commit -m "task-085: seedcsv parser — presence-aware CSV reader for registry seeding"
```

## Task 1.4: `registry seed` subcommand + generate the real registry

**Files:**
- Create: `tools/packet-audit/cmd/registry.go`
- Create: `tools/packet-audit/cmd/registry_test.go`
- Modify: `tools/packet-audit/cmd/root.go` (dispatch, after the existing `triage` block)
- Create (generated): `docs/packets/registry/gms_v83.yaml`, `gms_v84.yaml`, `gms_v87.yaml`, `gms_v95.yaml`, `jms_v185.yaml`
- Create: `docs/packets/registry/README.md`

Seeding maps version key → CSV column: `gms_v83` → `GMS:83`, `jms_v185` → `JMS:185`. **`gms_v84` is seeded from the `GMS:83` column** (the CSVs have no v84 column) with a note, per design §5.1.

- [ ] **Step 1: Write the failing test**

`cmd/registry_test.go`:

```go
package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/Chronicle20/atlas/tools/packet-audit/internal/opregistry"
)

func TestRegistrySeed(t *testing.T) {
	out := t.TempDir()
	code := runRegistry([]string{
		"seed",
		"--clientbound", filepath.Join("..", "internal", "seedcsv", "testdata", "clientbound_excerpt.csv"),
		"--serverbound", filepath.Join("..", "internal", "seedcsv", "testdata", "serverbound_excerpt.csv"),
		"--out", out,
	}, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("seed exit = %d", code)
	}
	for _, vk := range []string{"gms_v83", "gms_v84", "gms_v87", "gms_v95", "jms_v185"} {
		vf, err := opregistry.LoadVersion(filepath.Join(out, vk+".yaml"))
		if err != nil {
			t.Fatalf("%s: %v", vk, err)
		}
		if _, ok := vf.Lookup("LOGIN_STATUS", opregistry.DirClientbound); !ok {
			t.Errorf("%s: LOGIN_STATUS missing", vk)
		}
	}
	// v84 mirrors v83 (no CSV column).
	v84, _ := opregistry.LoadVersion(filepath.Join(out, "gms_v84.yaml"))
	e, ok := v84.Lookup("GUEST_LOGIN", opregistry.DirServerbound)
	if !ok || e.Opcode != 0x002 {
		t.Errorf("v84 GUEST_LOGIN = %+v ok=%v (want copy of v83)", e, ok)
	}
	// ACCOUNT_INFO absent in jms_v185 → no entry.
	jms, _ := opregistry.LoadVersion(filepath.Join(out, "jms_v185.yaml"))
	if _, ok := jms.Lookup("ACCOUNT_INFO", opregistry.DirClientbound); ok {
		t.Errorf("jms_v185 must not contain ACCOUNT_INFO")
	}
	// Determinism: seeding twice produces identical bytes.
	b1, _ := os.ReadFile(filepath.Join(out, "gms_v83.yaml"))
	out2 := t.TempDir()
	runRegistry([]string{"seed",
		"--clientbound", filepath.Join("..", "internal", "seedcsv", "testdata", "clientbound_excerpt.csv"),
		"--serverbound", filepath.Join("..", "internal", "seedcsv", "testdata", "serverbound_excerpt.csv"),
		"--out", out2}, &bytes.Buffer{})
	b2, _ := os.ReadFile(filepath.Join(out2, "gms_v83.yaml"))
	if !bytes.Equal(b1, b2) {
		t.Error("seed output not deterministic")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./cmd/ -run TestRegistrySeed
```
Expected: FAIL (`runRegistry` undefined).

- [ ] **Step 3: Implement `cmd/registry.go`**

```go
package cmd

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Chronicle20/atlas/tools/packet-audit/internal/opregistry"
	"github.com/Chronicle20/atlas/tools/packet-audit/internal/seedcsv"
	"gopkg.in/yaml.v3"
)

// seedVersions maps registry version keys to CSV column keys. gms_v84 has no
// CSV column (task-083: v84 ≡ v83 byte-identical); it copies GMS:83 with a note.
var seedVersions = []struct {
	Key    string
	CSVKey string
	Note   string
}{
	{"gms_v83", "GMS:83", ""},
	{"gms_v84", "GMS:83", "seeded from the v83 CSV column — the CSVs have no v84 column; task-083 found v84 byte-identical to v83. Corrected by discover-ops against the v84 IDB."},
	{"gms_v87", "GMS:87", ""},
	{"gms_v95", "GMS:95", ""},
	{"jms_v185", "JMS:185", ""},
}

func runRegistry(args []string, stderr io.Writer) int {
	if len(args) == 0 || args[0] != "seed" {
		fmt.Fprintln(stderr, "packet-audit registry: unknown subcommand (expected: seed)")
		return 3
	}
	fs := flag.NewFlagSet("packet-audit registry seed", flag.ContinueOnError)
	fs.SetOutput(stderr)
	cb := fs.String("clientbound", "docs/packets/MapleStory Ops - ClientBound.csv", "clientbound ops CSV")
	sb := fs.String("serverbound", "docs/packets/MapleStory Ops - ServerBound.csv", "serverbound ops CSV")
	out := fs.String("out", "docs/packets/registry", "output directory for registry YAMLs")
	if err := fs.Parse(args[1:]); err != nil {
		return 3
	}

	cbRows, err := seedcsv.Load(*cb)
	if err != nil {
		fmt.Fprintf(stderr, "packet-audit registry seed: %v\n", err)
		return 1
	}
	sbRows, err := seedcsv.Load(*sb)
	if err != nil {
		fmt.Fprintf(stderr, "packet-audit registry seed: %v\n", err)
		return 1
	}
	if err := os.MkdirAll(*out, 0o755); err != nil {
		fmt.Fprintf(stderr, "packet-audit registry seed: %v\n", err)
		return 1
	}
	for _, sv := range seedVersions {
		entries := seedEntries(cbRows, opregistry.DirClientbound, sv.CSVKey, sv.Note)
		entries = append(entries, seedEntries(sbRows, opregistry.DirServerbound, sv.CSVKey, sv.Note)...)
		sort.Slice(entries, func(i, j int) bool {
			if entries[i].Direction != entries[j].Direction {
				return entries[i].Direction == opregistry.DirClientbound
			}
			if entries[i].Opcode != entries[j].Opcode {
				return entries[i].Opcode < entries[j].Opcode
			}
			return entries[i].Op < entries[j].Op
		})
		raw, err := yaml.Marshal(entries)
		if err != nil {
			fmt.Fprintf(stderr, "packet-audit registry seed: %v\n", err)
			return 1
		}
		header := "# Generated by `packet-audit registry seed` — operation universe for " + sv.Key + ".\n" +
			"# Corrections/additions are hand-edited here (provenance: manual|ida-discovered);\n" +
			"# the source CSVs are frozen historical reference (design task-085 §5.1).\n"
		p := filepath.Join(*out, sv.Key+".yaml")
		if err := os.WriteFile(p, []byte(header+string(raw)), 0o644); err != nil {
			fmt.Fprintf(stderr, "packet-audit registry seed: %v\n", err)
			return 1
		}
	}
	return 0
}

func seedEntries(rows []seedcsv.Row, dir opregistry.Direction, csvKey, note string) []opregistry.Entry {
	var out []opregistry.Entry
	for _, r := range rows {
		cell, ok := r.Versions[csvKey]
		if !ok || !cell.Present {
			continue
		}
		e := opregistry.Entry{
			Op:         r.Op,
			Direction:  dir,
			Opcode:     cell.Opcode,
			FName:      r.FName,
			FNameAlts:  append([]string(nil), r.FNameAlts...),
			Provenance: "csv-import",
			Note:       note,
		}
		out = append(out, e)
	}
	return out
}

var _ = strings.TrimSpace // keep imports tidy if unused after edits
```

(Drop the trailing `var _` line if `strings` ends up unused — it is only there to remind you to clean imports.)

Wire into `cmd/root.go` next to the other dispatch blocks (after the `triage` block, around line 46):

```go
	if len(args) > 0 && args[0] == "registry" {
		return runRegistry(args[1:], stderr)
	}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./cmd/ -run TestRegistrySeed -v && go test ./... && go vet ./...
```
Expected: PASS, whole module green.

- [ ] **Step 5: Generate the real registry**

From the **repo root** (worktree root):

```bash
go run ./tools/packet-audit registry seed
ls docs/packets/registry/        # 5 yaml files
grep -c '^- op:' docs/packets/registry/gms_v83.yaml   # expect ≈ several hundred (CB+SB present rows)
diff <(sed 's/gms_v84/gms_v83/' docs/packets/registry/gms_v84.yaml) docs/packets/registry/gms_v83.yaml | head
```
The diff should show only the header/note lines differing (v84 is a v83 copy + note). If `registry seed` fails with a row error, that row's opcode cell is genuinely malformed — fix the CSV cell in the same commit and note it (design §13: correct, don't skip).

- [ ] **Step 6: Write `docs/packets/registry/README.md`**

```markdown
# Packet operation registry

One YAML per client version: the authoritative per-version operation universe
(rows + applicability) for the coverage matrix (`packet-audit matrix`).

- Seeded once from `docs/packets/MapleStory Ops - {ClientBound,ServerBound}.csv`
  via `packet-audit registry seed` (provenance: `csv-import`). **The CSVs are
  frozen as historical reference** — corrections and additions land here, not
  there.
- Grown/corrected by `packet-audit discover-ops` against the version's IDA
  database (provenance: `ida-discovered`, with the handler/site address).
- Human adjudications (CSV transcription error vs discovery blind spot) are
  recorded as `provenance: manual` with an IDA citation in `note`.
- `gms_v84.yaml` was seeded as a copy of the v83 column (no v84 CSV column;
  task-083 found v84 byte-identical to v83) and is corrected by discovery.

Schema per entry: `op`, `direction`, `opcode`, `fname`, optional `fname_alts`,
`provenance`, optional `ida.address`, optional `note`. Uniqueness:
(op, direction) per file. See task-085 design §5.1–5.2.
```

- [ ] **Step 7: Commit**

```bash
git add tools/packet-audit/cmd/registry.go tools/packet-audit/cmd/registry_test.go tools/packet-audit/cmd/root.go docs/packets/registry/
git commit -m "task-085: registry seed subcommand + seeded five-version registry"
git branch --show-current   # must print task-085-packet-audit-coverage-matrix
```

## Task 1.5: `internal/matrix` — cell model + input loading (reports, templates)

**Files:**
- Create: `tools/packet-audit/internal/matrix/model.go`
- Create: `tools/packet-audit/internal/matrix/load.go`
- Create: `tools/packet-audit/internal/matrix/load_test.go`
- Create: `tools/packet-audit/internal/matrix/testdata/audits/gms_v83/Invite.json`
- Create: `tools/packet-audit/internal/matrix/testdata/templates/template_gms_83_1.json`

- [ ] **Step 1: Write `model.go` (no test yet — pure declarations)**

```go
// Package matrix joins registry applicability, audit verdicts, evidence,
// tier membership and byte-test linkage into the coverage matrix
// (task-085 design §4, §5, §9).
package matrix

import (
	"github.com/Chronicle20/atlas/tools/packet-audit/internal/opregistry"
)

// VersionKeys is the canonical baseline column order (design §3).
var VersionKeys = []string{"gms_v83", "gms_v84", "gms_v87", "gms_v95", "jms_v185"}

// ExportPath maps a version key to its IDA export JSON. jms_v185's export
// kept its historical gms_jms_185 name (see memory: jms audit-dir mismatch).
func ExportPath(versionKey string) string {
	if versionKey == "jms_v185" {
		return "docs/packets/ida-exports/gms_jms_185.json"
	}
	return "docs/packets/ida-exports/" + versionKey + ".json"
}

// TemplatePath maps a version key to the tenant seed template.
// gms_v83 -> services/atlas-configurations/seed-data/templates/template_gms_83_1.json
func TemplatePath(versionKey string) string {
	name := map[string]string{
		"gms_v83":  "template_gms_83_1.json",
		"gms_v84":  "template_gms_84_1.json",
		"gms_v87":  "template_gms_87_1.json",
		"gms_v95":  "template_gms_95_1.json",
		"jms_v185": "template_jms_185_1.json",
	}[versionKey]
	return "services/atlas-configurations/seed-data/templates/" + name
}

// State is the graded cell state, in design §5 precedence order.
type State int

const (
	StateNA State = iota
	StateConflict
	StateVerified
	StatePartial
	StateIncomplete
)

func (s State) Symbol() string {
	switch s {
	case StateNA:
		return "⬜"
	case StateConflict:
		return "🟥"
	case StateVerified:
		return "✅"
	case StatePartial:
		return "🟡"
	default:
		return "❌"
	}
}

func (s State) Name() string {
	switch s {
	case StateNA:
		return "n-a"
	case StateConflict:
		return "conflict"
	case StateVerified:
		return "verified"
	case StatePartial:
		return "partial"
	default:
		return "incomplete"
	}
}

// Cell is one graded (op|packet, direction, version) cell.
type Cell struct {
	State State  `json:"state"`
	Note  string `json:"note,omitempty"` // conflict detail / degradation reason
}

// RowKind separates op rows (registry-joined) from sub-struct rows
// (audited shared structures with no opcode — design §10 rule 4).
type RowKind int

const (
	RowOp RowKind = iota
	RowSubStruct
)

type MatrixRow struct {
	Kind      RowKind                    `json:"kind"`
	Op        string                     `json:"op,omitempty"`     // RowOp only
	Packet    string                     `json:"packet,omitempty"` // "buddy/clientbound/Invite" when an Atlas struct exists
	Direction opregistry.Direction       `json:"direction"`
	Tier1     bool                       `json:"tier1"`
	Cells     map[string]Cell            `json:"cells"` // version key -> cell
}

// Matrix is the full joined result.
type Matrix struct {
	ToolSHA      string            `json:"toolSha"`      // git SHA of the tools/packet-audit tree
	ExportHashes map[string]string `json:"exportHashes"` // version key -> sha256 of export file
	Rows         []MatrixRow       `json:"rows"`
}
```

- [ ] **Step 2: Write the failing loader tests**

`load_test.go`:

```go
package matrix

import (
	"path/filepath"
	"testing"
)

func TestLoadReports(t *testing.T) {
	reps, err := LoadReports(filepath.Join("testdata", "audits", "gms_v83"))
	if err != nil {
		t.Fatalf("LoadReports: %v", err)
	}
	r, ok := reps["Invite"]
	if !ok {
		t.Fatalf("Invite report missing; got %v", keysOf(reps))
	}
	if r.IDAName != "CWvsContext::OnFriendResult#Invite" {
		t.Errorf("IDAName = %q", r.IDAName)
	}
	// Packet id derived from AtlasFile + WriterName, with the legacy ../../
	// prefix normalized away.
	if got := PacketID(r); got != "buddy/clientbound/Invite" {
		t.Errorf("PacketID = %q", got)
	}
}

func keysOf(m map[string]LoadedReport) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	return out
}
```

`testdata/audits/gms_v83/Invite.json` (shape copied from real reports, e.g. `docs/packets/audits/gms_v83/Action.json`):

```json
{
  "WriterName": "Invite",
  "IDAName": "CWvsContext::OnFriendResult#Invite",
  "Address": "0xa3f2e8",
  "Variant": "GMS/v83",
  "BranchDepth": 0,
  "AtlasFile": "../../libs/atlas-packet/buddy/clientbound/invite.go",
  "Rows": [],
  "Verdict": 0,
  "FlatInvalid": false
}
```

`testdata/templates/template_gms_83_1.json` — copy the real file's envelope with two handler + two writer entries; take the exact JSON structure from `services/atlas-configurations/seed-data/templates/template_gms_83_1.json` (envelope keys verified by `internal/template/template.go:19-45`: it reads `socket.handlers[].opCode` + `.handler` and `socket.writers[].opCode` + `.writer`). Trim to a minimal valid instance.

- [ ] **Step 3: Run tests to verify they fail**

```bash
go test ./internal/matrix/
```
Expected: FAIL (`LoadReports` undefined).

- [ ] **Step 4: Implement `load.go`**

```go
package matrix

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Chronicle20/atlas/tools/packet-audit/internal/diff"
)

// LoadedReport is the subset of report.Packet the matrix consumes, read back
// from the committed per-packet JSON files.
type LoadedReport struct {
	WriterName  string
	IDAName     string
	Address     string
	AtlasFile   string
	Verdict     diff.Verdict
	FlatInvalid bool
}

// LoadReports reads every per-packet JSON in an audit dir, skipping the
// non-report artifacts (SUMMARY/_pending/_unimplemented and any _-prefixed file).
// A missing dir is not an error: gms_v84 has no audit reports yet (design §3);
// its cells grade incomplete from absence.
func LoadReports(auditDir string) (map[string]LoadedReport, error) {
	out := map[string]LoadedReport{}
	entries, err := os.ReadDir(auditDir)
	if os.IsNotExist(err) {
		return out, nil
	}
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".json") || strings.HasPrefix(name, "_") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(auditDir, name))
		if err != nil {
			return nil, err
		}
		var r LoadedReport
		if err := json.Unmarshal(raw, &r); err != nil {
			return nil, fmt.Errorf("%s/%s: %w", auditDir, name, err)
		}
		if r.WriterName == "" {
			continue // not a report-shaped JSON
		}
		out[r.WriterName] = r
	}
	return out, nil
}

// PacketID derives the canonical packet identity "pkgdir/Struct" from a
// report: AtlasFile (normalized to libs/atlas-packet-relative — older
// committed reports carry a ../../ prefix, newer ones are repo-relative
// per PR #729) plus WriterName.
func PacketID(r LoadedReport) string {
	f := r.AtlasFile
	if i := strings.Index(f, "libs/atlas-packet/"); i >= 0 {
		f = f[i+len("libs/atlas-packet/"):]
	}
	return filepath.ToSlash(filepath.Dir(f)) + "/" + r.WriterName
}
```

- [ ] **Step 5: Run tests to verify they pass**

```bash
go test ./internal/matrix/ -v
```
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
cd ../..
git add tools/packet-audit/internal/matrix/
git commit -m "task-085: matrix model + report/template loading"
```

## Task 1.6: `internal/matrix` — the grading engine (complete, all rules)

**Files:**
- Create: `tools/packet-audit/internal/matrix/grade.go`
- Create: `tools/packet-audit/internal/matrix/grade_test.go`

The engine implements design §5 **in full** now. Evidence, tiers, and markers
arrive as *inputs* (`Inputs` struct fields); Phases 2–3 wire real loaders in.
Until then callers pass empty maps and the rules involving them are inert —
but they are tested now with hand-built inputs, so wiring is risk-free later.

- [ ] **Step 1: Write the failing grading tests (one per §5 rule + precedence)**

`grade_test.go`:

```go
package matrix

import (
	"testing"

	"github.com/Chronicle20/atlas/tools/packet-audit/internal/diff"
	"github.com/Chronicle20/atlas/tools/packet-audit/internal/opregistry"
)

// helper: a one-version Inputs scaffold the cases mutate.
func baseInputs() Inputs {
	return Inputs{
		Registry: opregistry.Registry{Versions: map[string]*opregistry.VersionFile{}},
		Reports:  map[string]map[string]LoadedReport{},      // version -> writer -> report
		Routed:   map[string]map[routeKey]bool{},            // version -> (opcode,dir) routed
		RoutedAnywhere: map[routeKey]bool{},                 // (opcode,dir) routed in ANY version
		Evidence: map[evKey]EvidenceStatus{},                // (packet,version) -> status
		Tier1:    map[string]bool{},                         // packet id -> tier1
		Markers:  map[evKey]MarkerStatus{},                  // (packet,version) -> marker
	}
}

func TestGradeNA(t *testing.T) {
	in := baseInputs()
	in.Registry.Versions["gms_v83"] = vfWith(t /* no entries */)
	c := gradeOpCell(in, opEntryRef{Op: "ACCOUNT_INFO", Dir: opregistry.DirClientbound, Opcode: 0x002}, "gms_v83")
	if c.State != StateNA {
		t.Errorf("state = %v (%s)", c.State.Name(), c.Note)
	}
}

func TestGradeConflictTemplateRoutesAbsentOp(t *testing.T) {
	in := baseInputs()
	in.Registry.Versions["gms_v83"] = vfWith(t) // op absent
	in.Routed["gms_v83"] = map[routeKey]bool{{0x002, opregistry.DirClientbound}: true}
	c := gradeOpCell(in, opEntryRef{Op: "ACCOUNT_INFO", Dir: opregistry.DirClientbound, Opcode: 0x002}, "gms_v83")
	if c.State != StateConflict {
		t.Errorf("state = %v", c.State.Name())
	}
}

func TestGradeConflictAtlasClaimsAbsentOp(t *testing.T) {
	in := baseInputs()
	in.Registry.Versions["gms_v83"] = vfWith(t) // absent
	in.Reports["gms_v83"] = map[string]LoadedReport{"AccountInfo": {
		WriterName: "AccountInfo", IDAName: "CLogin::OnAccountInfoResult",
		AtlasFile: "libs/atlas-packet/login/clientbound/account_info.go", Verdict: diff.VerdictMatch,
	}}
	in.FNameToWriter = map[string]map[string]string{"gms_v83": {"CLogin::OnAccountInfoResult": "AccountInfo"}}
	c := gradeOpCell(in, opEntryRef{Op: "ACCOUNT_INFO", Dir: opregistry.DirClientbound, Opcode: 0x002,
		FName: "CLogin::OnAccountInfoResult"}, "gms_v83")
	if c.State != StateConflict {
		t.Errorf("state = %v (%s)", c.State.Name(), c.Note)
	}
}

func TestGradeConflictCrossVersionTemplateGap(t *testing.T) {
	// Registry: present. This version's template does NOT route it, but some
	// other version's template does -> the task-067/068 gap class (context.md
	// decision D3 refines design §5).
	in := baseInputs()
	in.Registry.Versions["gms_v83"] = vfWith(t, opregistry.Entry{
		Op: "ACCOUNT_INFO", Direction: opregistry.DirClientbound, Opcode: 0x002,
		FName: "CLogin::OnAccountInfoResult", Provenance: "csv-import"})
	in.Routed["gms_v83"] = map[routeKey]bool{} // not routed here
	in.RoutedAnywhere = map[routeKey]bool{{0x002, opregistry.DirClientbound}: true}
	c := gradeOpCell(in, opEntryRef{Op: "ACCOUNT_INFO", Dir: opregistry.DirClientbound, Opcode: 0x002,
		FName: "CLogin::OnAccountInfoResult"}, "gms_v83")
	if c.State != StateConflict {
		t.Errorf("state = %v (%s)", c.State.Name(), c.Note)
	}
}

func TestGradeUnroutedEverywhereIsIncompleteNotConflict(t *testing.T) {
	in := baseInputs()
	in.Registry.Versions["gms_v83"] = vfWith(t, opregistry.Entry{
		Op: "ACCOUNT_INFO", Direction: opregistry.DirClientbound, Opcode: 0x002,
		FName: "CLogin::OnAccountInfoResult", Provenance: "csv-import"})
	c := gradeOpCell(in, opEntryRef{Op: "ACCOUNT_INFO", Dir: opregistry.DirClientbound, Opcode: 0x002,
		FName: "CLogin::OnAccountInfoResult"}, "gms_v83")
	if c.State != StateIncomplete {
		t.Errorf("state = %v (%s)", c.State.Name(), c.Note)
	}
}

func TestGradePartialToolPassNoTest(t *testing.T) {
	in := presentWithReport(t, diff.VerdictMatch, false)
	c := gradeOpCell(in, refACCOUNT(), "gms_v83")
	if c.State != StatePartial {
		t.Errorf("state = %v (%s)", c.State.Name(), c.Note)
	}
}

func TestGradeVerifiedTier0(t *testing.T) {
	in := presentWithReport(t, diff.VerdictMatch, false)
	in.Markers[evKey{"login/clientbound/AccountInfo", "gms_v83"}] = MarkerStatus{Found: true, Address: "0xa3f2e8"}
	c := gradeOpCell(in, refACCOUNT(), "gms_v83")
	if c.State != StateVerified {
		t.Errorf("state = %v (%s)", c.State.Name(), c.Note)
	}
}

func TestGradeTier1ToolPassCapsAtPartial(t *testing.T) {
	in := presentWithReport(t, diff.VerdictMatch, false)
	in.Tier1["login/clientbound/AccountInfo"] = true
	c := gradeOpCell(in, refACCOUNT(), "gms_v83")
	if c.State != StatePartial {
		t.Errorf("tier1 tool-pass must cap at partial; state = %v", c.State.Name())
	}
}

func TestGradeTier1FixturePromotes(t *testing.T) {
	in := presentWithReport(t, diff.VerdictDeferred, true) // diff verdict advisory on tier1
	in.Tier1["login/clientbound/AccountInfo"] = true
	in.Markers[evKey{"login/clientbound/AccountInfo", "gms_v83"}] = MarkerStatus{Found: true, Address: "0xa3f2e8"}
	in.Evidence[evKey{"login/clientbound/AccountInfo", "gms_v83"}] = EvidenceStatus{Exists: true, Fresh: true, Address: "0xa3f2e8"}
	c := gradeOpCell(in, refACCOUNT(), "gms_v83")
	if c.State != StateVerified {
		t.Errorf("state = %v (%s)", c.State.Name(), c.Note)
	}
}

func TestGradeEvidencePinnedDeferralIsPartial(t *testing.T) {
	in := presentWithReport(t, diff.VerdictDeferred, false)
	in.Evidence[evKey{"login/clientbound/AccountInfo", "gms_v83"}] = EvidenceStatus{Exists: true, Fresh: true}
	c := gradeOpCell(in, refACCOUNT(), "gms_v83")
	if c.State != StatePartial {
		t.Errorf("state = %v (%s)", c.State.Name(), c.Note)
	}
}

func TestGradeStaleEvidenceDegrades(t *testing.T) {
	in := presentWithReport(t, diff.VerdictDeferred, false)
	in.Evidence[evKey{"login/clientbound/AccountInfo", "gms_v83"}] = EvidenceStatus{Exists: true, Fresh: false}
	c := gradeOpCell(in, refACCOUNT(), "gms_v83")
	if c.State != StateIncomplete {
		t.Errorf("stale evidence must degrade; state = %v (%s)", c.State.Name(), c.Note)
	}
}

func TestGradeBlockerVerdictIncomplete(t *testing.T) {
	in := presentWithReport(t, diff.VerdictBlocker, false)
	c := gradeOpCell(in, refACCOUNT(), "gms_v83")
	if c.State != StateIncomplete {
		t.Errorf("state = %v", c.State.Name())
	}
}

func TestGradeUnknownVersionFile(t *testing.T) {
	in := baseInputs() // no registry file at all for gms_v84
	c := gradeOpCell(in, refACCOUNT(), "gms_v84")
	if c.State != StateIncomplete || c.Note == "" {
		t.Errorf("unknown applicability must be incomplete+note; got %v %q", c.State.Name(), c.Note)
	}
}

// --- helpers ---

func refACCOUNT() opEntryRef {
	return opEntryRef{Op: "ACCOUNT_INFO", Dir: opregistry.DirClientbound, Opcode: 0x002,
		FName: "CLogin::OnAccountInfoResult"}
}

func presentWithReport(t *testing.T, v diff.Verdict, flatInvalid bool) Inputs {
	t.Helper()
	in := baseInputs()
	in.Registry.Versions["gms_v83"] = vfWith(t, opregistry.Entry{
		Op: "ACCOUNT_INFO", Direction: opregistry.DirClientbound, Opcode: 0x002,
		FName: "CLogin::OnAccountInfoResult", Provenance: "csv-import"})
	in.Routed["gms_v83"] = map[routeKey]bool{{0x002, opregistry.DirClientbound}: true}
	in.RoutedAnywhere = map[routeKey]bool{{0x002, opregistry.DirClientbound}: true}
	in.Reports["gms_v83"] = map[string]LoadedReport{"AccountInfo": {
		WriterName: "AccountInfo", IDAName: "CLogin::OnAccountInfoResult", Address: "0xa3f2e8",
		AtlasFile: "libs/atlas-packet/login/clientbound/account_info.go",
		Verdict: v, FlatInvalid: flatInvalid,
	}}
	in.FNameToWriter = map[string]map[string]string{"gms_v83": {"CLogin::OnAccountInfoResult": "AccountInfo"}}
	return in
}

// vfWith builds a VersionFile from entries via LoadVersion round-trip semantics.
func vfWith(t *testing.T, entries ...opregistry.Entry) *opregistry.VersionFile {
	t.Helper()
	return opregistry.NewVersionFile(entries) // small exported ctor; add to opregistry
}
```

Note this test requires one small addition to `opregistry`: an exported
constructor used only for assembling in-memory registries (tests + discover-ops
reconciliation later):

```go
// NewVersionFile builds an in-memory VersionFile (no schema validation —
// callers that need validation go through LoadVersion).
func NewVersionFile(entries []Entry) *VersionFile {
	vf := &VersionFile{Entries: entries, byKey: make(map[string]Entry, len(entries))}
	for _, e := range entries {
		vf.byKey[key(e.Op, e.Direction)] = e
	}
	return vf
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/matrix/ -run TestGrade
```
Expected: FAIL (`Inputs`, `gradeOpCell` undefined).

- [ ] **Step 3: Implement `grade.go`**

```go
package matrix

import (
	"fmt"

	"github.com/Chronicle20/atlas/tools/packet-audit/internal/diff"
	"github.com/Chronicle20/atlas/tools/packet-audit/internal/opregistry"
)

type routeKey struct {
	Opcode int
	Dir    opregistry.Direction
}

type evKey struct {
	Packet  string // "buddy/clientbound/Invite"
	Version string // "gms_v83"
}

// EvidenceStatus is the matrix-facing summary of one evidence record
// (loader lands in Phase 2; until then the map is empty).
type EvidenceStatus struct {
	Exists  bool
	Fresh   bool   // decompile_sha256 matches current export
	Address string // pinned ida address ("0x...")
	Note    string // degradation reason when !Fresh ("hash drift", "citation unresolvable")
}

// MarkerStatus is the matrix-facing summary of byte-test linkage
// (scanner lands in Phase 3; until then the map is empty).
type MarkerStatus struct {
	Found   bool
	Address string
}

// Inputs is everything grading consumes. All maps may be empty; rules that
// depend on them simply never fire.
type Inputs struct {
	Registry       opregistry.Registry
	Reports        map[string]map[string]LoadedReport // version -> WriterName -> report
	FNameToWriter  map[string]map[string]string       // version -> FName -> WriterName (built from Reports)
	Routed         map[string]map[routeKey]bool       // version -> routed (opcode, dir)
	RoutedAnywhere map[routeKey]bool                  // routed in any version's template
	Evidence       map[evKey]EvidenceStatus
	Tier1          map[string]bool // packet id -> tier-1
	Markers        map[evKey]MarkerStatus
}

// opEntryRef carries the union-row identity being graded for one version.
type opEntryRef struct {
	Op     string
	Dir    opregistry.Direction
	Opcode int
	FName  string
}

// gradeOpCell evaluates design §5 in precedence order for one op×version.
func gradeOpCell(in Inputs, ref opEntryRef, version string) Cell {
	app := in.Registry.Applicability(ref.Op, ref.Dir, version)
	routed := in.Routed[version][routeKey{ref.Opcode, ref.Dir}]
	rep, hasReport := findReport(in, ref, version)

	switch app {
	case opregistry.Unknown:
		return Cell{State: StateIncomplete, Note: "applicability unknown — no registry file for " + version}
	case opregistry.Absent:
		if routed {
			return Cell{State: StateConflict, Note: fmt.Sprintf("registry says absent but template routes opcode 0x%03X", ref.Opcode)}
		}
		if hasReport {
			return Cell{State: StateConflict, Note: "registry says absent but an Atlas audit report exists (" + rep.WriterName + ")"}
		}
		return Cell{State: StateNA}
	}

	// Present from here on.
	if !routed && in.RoutedAnywhere[routeKey{ref.Opcode, ref.Dir}] {
		return Cell{State: StateConflict, Note: "op present in client and routed in another version's template, but unrouted here (template coverage gap)"}
	}
	if !hasReport {
		return Cell{State: StateIncomplete, Note: "no audit report"}
	}

	pkt := PacketID(rep)
	ev, hasEv := in.Evidence[evKey{pkt, version}]
	mk := in.Markers[evKey{pkt, version}]
	tier1 := in.Tier1[pkt] || rep.FlatInvalid

	if hasEv && !ev.Fresh {
		note := ev.Note
		if note == "" {
			note = "evidence stale (decompile hash drift)"
		}
		return Cell{State: StateIncomplete, Note: note}
	}

	toolPass := rep.Verdict == diff.VerdictMatch && !rep.FlatInvalid

	if tier1 {
		// Diff verdict is advisory; only a linked byte-fixture promotes.
		if mk.Found && hasEv && ev.Fresh {
			return Cell{State: StateVerified}
		}
		if mk.Found {
			return Cell{State: StateIncomplete, Note: "byte-test marker present but no fresh evidence record"}
		}
		if toolPass || (hasEv && ev.Fresh) {
			return Cell{State: StatePartial, Note: "tier-1: needs byte-fixture test to verify"}
		}
		return Cell{State: StateIncomplete, Note: "tier-1 without fixture; verdict " + rep.Verdict.Symbol()}
	}

	// Tier 0.
	if toolPass && mk.Found {
		return Cell{State: StateVerified}
	}
	if toolPass {
		return Cell{State: StatePartial, Note: "tool ✅ without byte-test"}
	}
	if hasEv && ev.Fresh {
		return Cell{State: StatePartial, Note: "evidence-pinned deferral"}
	}
	return Cell{State: StateIncomplete, Note: "verdict " + rep.Verdict.Symbol()}
}

// findReport joins a registry op to its audit report via FName -> WriterName.
func findReport(in Inputs, ref opEntryRef, version string) (LoadedReport, bool) {
	wn, ok := in.FNameToWriter[version][ref.FName]
	if !ok {
		return LoadedReport{}, false
	}
	r, ok := in.Reports[version][wn]
	return r, ok
}
```

Note on the FName→Writer join: report `IDAName` values carry case suffixes
(`CWvsContext::OnFriendResult#Invite`). When building `FNameToWriter` (Task
1.7), strip everything from `#` before indexing, so registry FNames
(`CWvsContext::OnFriendResult`) join every per-case report of that dispatcher;
when an FName maps to multiple writers, the join picks the **worst** cell
(max State value after grading each candidate) so a dispatcher family is only
as green as its weakest audited case. Implement that in the Task 1.7 builder,
not here.

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/matrix/ -run TestGrade -v
```
Expected: PASS (12 grading tests).

- [ ] **Step 5: Commit**

```bash
cd ../..
git add tools/packet-audit/internal/matrix/ tools/packet-audit/internal/opregistry/
git commit -m "task-085: matrix grading engine — full §5 rules with precedence tests"
```

## Task 1.7: `internal/matrix` — assembly + STATUS.md / status.json rendering

**Files:**
- Create: `tools/packet-audit/internal/matrix/build.go`
- Create: `tools/packet-audit/internal/matrix/render.go`
- Create: `tools/packet-audit/internal/matrix/render_test.go`
- Create: `tools/packet-audit/internal/matrix/testdata/golden_STATUS.md`

- [ ] **Step 1: Write the failing end-to-end render test**

`render_test.go`:

```go
package matrix

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Chronicle20/atlas/tools/packet-audit/internal/diff"
	"github.com/Chronicle20/atlas/tools/packet-audit/internal/opregistry"
)

// e2eInputs: 2 versions, 3 ops (one verified-less partial, one n-a in v87,
// one conflict), plus one sub-struct report that joins no registry op.
func e2eInputs(t *testing.T) Inputs {
	t.Helper()
	in := Inputs{
		Registry: opregistry.Registry{Versions: map[string]*opregistry.VersionFile{
			"gms_v83": opregistry.NewVersionFile([]opregistry.Entry{
				{Op: "LOGIN_STATUS", Direction: opregistry.DirClientbound, Opcode: 0x000, FName: "CLogin::OnCheckPasswordResult", Provenance: "csv-import"},
				{Op: "ACCOUNT_INFO", Direction: opregistry.DirClientbound, Opcode: 0x002, FName: "CLogin::OnAccountInfoResult", Provenance: "csv-import"},
			}),
			"gms_v87": opregistry.NewVersionFile([]opregistry.Entry{
				{Op: "LOGIN_STATUS", Direction: opregistry.DirClientbound, Opcode: 0x000, FName: "CLogin::OnCheckPasswordResult", Provenance: "csv-import"},
			}),
		}},
		Reports: map[string]map[string]LoadedReport{
			"gms_v83": {"AuthResult": {WriterName: "AuthResult", IDAName: "CLogin::OnCheckPasswordResult", Address: "0x5e1230",
				AtlasFile: "libs/atlas-packet/login/clientbound/auth_result.go", Verdict: diff.VerdictMatch},
				"StatRegistry": {WriterName: "StatRegistry", IDAName: "GW_CharacterStat::Decode", Address: "0x123456",
					AtlasFile: "libs/atlas-packet/model/stat_registry.go", Verdict: diff.VerdictMatch}},
			"gms_v87": {"AuthResult": {WriterName: "AuthResult", IDAName: "CLogin::OnCheckPasswordResult", Address: "0x6f1230",
				AtlasFile: "libs/atlas-packet/login/clientbound/auth_result.go", Verdict: diff.VerdictDeferred}},
		},
		Routed: map[string]map[routeKey]bool{
			"gms_v83": {{0x000, opregistry.DirClientbound}: true},
			"gms_v87": {{0x000, opregistry.DirClientbound}: true, {0x002, opregistry.DirClientbound}: true},
		},
		RoutedAnywhere: map[routeKey]bool{
			{0x000, opregistry.DirClientbound}: true,
			{0x002, opregistry.DirClientbound}: true,
		},
		Evidence: map[evKey]EvidenceStatus{},
		Tier1:    map[string]bool{},
		Markers:  map[evKey]MarkerStatus{},
	}
	return in
}

func TestBuildAndRenderGolden(t *testing.T) {
	m := Build(e2eInputs(t), []string{"gms_v83", "gms_v87"})
	m.ToolSHA = "testsha"
	m.ExportHashes = map[string]string{"gms_v83": "aaa", "gms_v87": "bbb"}

	got := RenderMarkdown(m, []string{"gms_v83", "gms_v87"})
	golden := filepath.Join("testdata", "golden_STATUS.md")
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		os.WriteFile(golden, []byte(got), 0o644)
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("read golden: %v (run once with UPDATE_GOLDEN=1)", err)
	}
	if got != string(want) {
		t.Errorf("STATUS.md drifted from golden:\n%s", got)
	}

	// Spot-check semantics independent of the golden bytes:
	if !strings.Contains(got, "## Conflicts") {
		t.Error("missing conflicts section")
	}
	// ACCOUNT_INFO is absent in v87 registry but routed by v87 template -> conflict.
	if !strings.Contains(got, "ACCOUNT_INFO") {
		t.Error("conflict row missing")
	}
	// Sub-struct section exists with StatRegistry.
	if !strings.Contains(got, "## Sub-structs") || !strings.Contains(got, "StatRegistry") {
		t.Error("sub-struct section missing")
	}
	// No wall-clock date anywhere (determinism; context.md D2).
	if strings.Contains(got, "20") && strings.Contains(got, "T") && strings.Contains(got, "Z") {
		// crude guard: full ISO timestamps must not appear
		for _, line := range strings.Split(got, "\n") {
			if strings.Contains(line, "Z") && strings.Contains(line, ":") && strings.Contains(line, "T") {
				t.Errorf("timestamp-looking line breaks determinism: %q", line)
			}
		}
	}
}

func TestRenderDeterminism(t *testing.T) {
	m := Build(e2eInputs(t), []string{"gms_v83", "gms_v87"})
	a := RenderMarkdown(m, []string{"gms_v83", "gms_v87"})
	b := RenderMarkdown(Build(e2eInputs(t), []string{"gms_v83", "gms_v87"}), []string{"gms_v83", "gms_v87"})
	if a != b {
		t.Fatal("two consecutive builds differ")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/matrix/ -run 'TestBuild|TestRenderDeterminism'
```
Expected: FAIL (`Build`, `RenderMarkdown` undefined).

- [ ] **Step 3: Implement `build.go`**

```go
package matrix

import (
	"sort"
	"strings"

	"github.com/Chronicle20/atlas/tools/packet-audit/internal/opregistry"
)

// baseFName strips the per-case suffix: "CWvsContext::OnFriendResult#Invite"
// -> "CWvsContext::OnFriendResult".
func baseFName(idaName string) string {
	if i := strings.Index(idaName, "#"); i >= 0 {
		return idaName[:i]
	}
	return idaName
}

// Build joins all inputs into the Matrix. versionKeys fixes column order.
func Build(in Inputs, versionKeys []string) Matrix {
	// Index FName -> writers per version (a dispatcher FName may map to many
	// per-case writers; the op row takes the WORST graded cell of them).
	fnameWriters := map[string]map[string][]string{}
	for vk, reps := range in.Reports {
		fnameWriters[vk] = map[string][]string{}
		for wn, r := range reps {
			f := baseFName(r.IDAName)
			fnameWriters[vk][f] = append(fnameWriters[vk][f], wn)
		}
		for f := range fnameWriters[vk] {
			sort.Strings(fnameWriters[vk][f])
		}
	}
	// FNameToWriter for gradeOpCell's single-report path: filled per candidate
	// below, so leave the simple map for the worst-of loop.

	usedWriters := map[string]map[string]bool{} // version -> writer consumed by an op row
	for _, vk := range versionKeys {
		usedWriters[vk] = map[string]bool{}
	}

	var rows []MatrixRow
	for _, od := range in.Registry.AllOps() {
		row := MatrixRow{Kind: RowOp, Op: od.Op, Direction: od.Dir, Cells: map[string]Cell{}}
		for _, vk := range versionKeys {
			ref := opEntryRef{Op: od.Op, Dir: od.Dir}
			if vf, ok := in.Registry.Versions[vk]; ok {
				if e, ok := vf.Lookup(od.Op, od.Dir); ok {
					ref.Opcode, ref.FName = e.Opcode, e.FName
				} else if e, ok := lookupAnyVersion(in.Registry, od.Op, od.Dir); ok {
					ref.Opcode, ref.FName = e.Opcode, e.FName // opcode for routing checks of absent ops
				}
			} else if e, ok := lookupAnyVersion(in.Registry, od.Op, od.Dir); ok {
				ref.Opcode, ref.FName = e.Opcode, e.FName
			}
			cell := worstCandidateCell(in, fnameWriters, ref, vk, usedWriters)
			row.Cells[vk] = cell
		}
		// Tier + packet annotation from any version's report.
		row.Packet, row.Tier1 = rowPacketAndTier(in, fnameWriters, row, versionKeys)
		rows = append(rows, row)
	}

	// Sub-struct rows: reports never consumed by an op row.
	sub := map[string]MatrixRow{}
	for _, vk := range versionKeys {
		for wn, r := range in.Reports[vk] {
			if usedWriters[vk][wn] {
				continue
			}
			pkt := PacketID(r)
			mr, ok := sub[pkt]
			if !ok {
				mr = MatrixRow{Kind: RowSubStruct, Packet: pkt, Cells: map[string]Cell{}}
			}
			mr.Tier1 = mr.Tier1 || in.Tier1[pkt] || r.FlatInvalid
			mr.Cells[vk] = gradeSubStructCell(in, r, pkt, vk)
			sub[pkt] = mr
		}
	}
	var subKeys []string
	for k := range sub {
		subKeys = append(subKeys, k)
	}
	sort.Strings(subKeys)
	for _, k := range subKeys {
		mr := sub[k]
		for _, vk := range versionKeys { // fill gaps so columns align
			if _, ok := mr.Cells[vk]; !ok {
				mr.Cells[vk] = Cell{State: StateIncomplete, Note: "no audit report"}
			}
		}
		rows = append(rows, mr)
	}
	return Matrix{Rows: rows}
}

func lookupAnyVersion(r opregistry.Registry, op string, dir opregistry.Direction) (opregistry.Entry, bool) {
	var vks []string
	for vk := range r.Versions {
		vks = append(vks, vk)
	}
	sort.Strings(vks)
	for _, vk := range vks {
		if e, ok := r.Versions[vk].Lookup(op, dir); ok {
			return e, true
		}
	}
	return opregistry.Entry{}, false
}

// worstCandidateCell grades each writer candidate for the op's FName and keeps
// the worst (max State); marks candidates as consumed by op rows.
func worstCandidateCell(in Inputs, fw map[string]map[string][]string, ref opEntryRef, vk string, used map[string]map[string]bool) Cell {
	writers := fw[vk][ref.FName]
	if len(writers) == 0 {
		in.FNameToWriter = map[string]map[string]string{vk: {}}
		return gradeOpCell(in, ref, vk)
	}
	worst := Cell{State: StateNA, Note: ""}
	first := true
	for _, wn := range writers {
		used[vk][wn] = true
		in.FNameToWriter = map[string]map[string]string{vk: {ref.FName: wn}}
		c := gradeOpCell(in, ref, vk)
		if first || c.State > worst.State {
			worst, first = c, false
		}
	}
	return worst
}

func gradeSubStructCell(in Inputs, r LoadedReport, pkt, vk string) Cell {
	// Same evidence/tier/marker rules, minus applicability (no registry op).
	in.FNameToWriter = map[string]map[string]string{vk: {baseFName(r.IDAName): r.WriterName}}
	in.Routed = map[string]map[routeKey]bool{vk: {{0, ""}: false}}
	// Reuse the present-branch by grading against a synthetic present entry:
	saved := in.Registry
	in.Registry = opregistry.Registry{Versions: map[string]*opregistry.VersionFile{
		vk: opregistry.NewVersionFile([]opregistry.Entry{{
			Op: "__SUB__" + pkt, Direction: opregistry.Direction(dirOf(pkt)), Opcode: -1,
			FName: baseFName(r.IDAName), Provenance: "manual"}}),
	}}
	in.RoutedAnywhere = map[routeKey]bool{} // sub-structs are never "routed"
	c := gradeOpCell(in, opEntryRef{Op: "__SUB__" + pkt, Dir: opregistry.Direction(dirOf(pkt)), Opcode: -1,
		FName: baseFName(r.IDAName)}, vk)
	in.Registry = saved
	return c
}

func dirOf(pkt string) string {
	if strings.Contains(pkt, "/serverbound/") {
		return "serverbound"
	}
	return "clientbound"
}

func rowPacketAndTier(in Inputs, fw map[string]map[string][]string, row MatrixRow, versionKeys []string) (string, bool) {
	for _, vk := range versionKeys {
		if vf, ok := in.Registry.Versions[vk]; ok {
			if e, ok := vf.Lookup(row.Op, row.Direction); ok {
				for _, wn := range fw[vk][e.FName] {
					r := in.Reports[vk][wn]
					pkt := PacketID(r)
					return pkt, in.Tier1[pkt] || r.FlatInvalid
				}
			}
		}
	}
	return "", false
}
```

> Implementation note for the executor: `gradeSubStructCell` above shows the
> intended *semantics* (present + unrouted-anywhere ⇒ never conflict, evidence/
> tier/marker rules apply). If mutating `Inputs` copies feels fragile in
> practice, refactor `gradeOpCell` to take a small `gradeArgs` struct
> (applicability, routed, routedAnywhere, report, evidence, marker, tier1) and
> have both `worstCandidateCell` and `gradeSubStructCell` call that core — the
> §5 rule tests from Task 1.6 must keep passing unchanged.
>
> Design §13 last bullet (two Atlas structs claiming the same op): per-case
> writers of one dispatcher legitimately share a base FName and are handled by
> worst-of. But if two DIFFERENT writers carry the **identical full IDAName**
> (no `#case` suffix distinguishing them), that is a genuine duplicate claim —
> `worstCandidateCell` must return `Cell{State: StateConflict, Note: "two
> Atlas structs claim <FName>: <w1>, <w2>"}` instead of grading. Add a unit
> test for this in `grade_test.go` (two reports, same suffix-free IDAName).

- [ ] **Step 4: Implement `render.go`**

```go
package matrix

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/Chronicle20/atlas/tools/packet-audit/internal/opregistry"
)

func colLabel(vk string) string {
	return map[string]string{
		"gms_v83": "v83", "gms_v84": "v84", "gms_v87": "v87",
		"gms_v95": "v95", "jms_v185": "JMS185",
	}[vk]
}

// RenderMarkdown produces STATUS.md (design §9): per direction one table,
// totals per version, conflicts section, generation stamp (tool SHA + export
// hashes; deliberately NO date — context.md D2, determinism).
func RenderMarkdown(m Matrix, versionKeys []string) string {
	var b strings.Builder
	b.WriteString("# Packet coverage matrix\n\n")
	b.WriteString("> Generated by `packet-audit matrix` — never hand-edit. Cell states (design task-085 §5):\n")
	b.WriteString("> ✅ verified · 🟡 partial · ❌ incomplete · ⬜ n-a · 🟥 conflict\n\n")
	fmt.Fprintf(&b, "Tool: `%s`\n\n", m.ToolSHA)
	var hk []string
	for k := range m.ExportHashes {
		hk = append(hk, k)
	}
	sort.Strings(hk)
	for _, k := range hk {
		fmt.Fprintf(&b, "- export %s: `%s`\n", k, m.ExportHashes[k])
	}
	b.WriteString("\n")

	renderDirection(&b, m, versionKeys, opregistry.DirClientbound, "## Clientbound")
	renderDirection(&b, m, versionKeys, opregistry.DirServerbound, "## Serverbound")
	renderSubStructs(&b, m, versionKeys)
	renderTotals(&b, m, versionKeys)
	renderConflicts(&b, m, versionKeys)
	return b.String()
}

func renderDirection(b *strings.Builder, m Matrix, versionKeys []string, dir opregistry.Direction, title string) {
	fmt.Fprintf(b, "%s\n\n| Op | Packet |", title)
	for _, vk := range versionKeys {
		fmt.Fprintf(b, " %s |", colLabel(vk))
	}
	b.WriteString("\n|----|--------|")
	for range versionKeys {
		b.WriteString("-----|")
	}
	b.WriteString("\n")
	for _, r := range m.Rows {
		if r.Kind != RowOp || r.Direction != dir {
			continue
		}
		pkt := r.Packet
		if r.Tier1 && pkt != "" {
			pkt += " (T1)"
		}
		fmt.Fprintf(b, "| %s | %s |", r.Op, pkt)
		for _, vk := range versionKeys {
			fmt.Fprintf(b, " %s |", r.Cells[vk].State.Symbol())
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")
}

func renderSubStructs(b *strings.Builder, m Matrix, versionKeys []string) {
	b.WriteString("## Sub-structs & shared types\n\n| Packet |")
	for _, vk := range versionKeys {
		fmt.Fprintf(b, " %s |", colLabel(vk))
	}
	b.WriteString("\n|--------|")
	for range versionKeys {
		b.WriteString("-----|")
	}
	b.WriteString("\n")
	for _, r := range m.Rows {
		if r.Kind != RowSubStruct {
			continue
		}
		pkt := r.Packet
		if r.Tier1 {
			pkt += " (T1)"
		}
		fmt.Fprintf(b, "| %s |", pkt)
		for _, vk := range versionKeys {
			fmt.Fprintf(b, " %s |", r.Cells[vk].State.Symbol())
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")
}

func renderTotals(b *strings.Builder, m Matrix, versionKeys []string) {
	b.WriteString("## Totals\n\n| Version | ✅ | 🟡 | ❌ | ⬜ | 🟥 | verified% |\n|---------|----|----|----|----|----|-----------|\n")
	for _, vk := range versionKeys {
		var counts [5]int
		total := 0
		for _, r := range m.Rows {
			c := r.Cells[vk]
			counts[c.State]++
			if c.State != StateNA {
				total++
			}
		}
		pct := 0.0
		if total > 0 {
			pct = 100 * float64(counts[StateVerified]) / float64(total)
		}
		fmt.Fprintf(b, "| %s | %d | %d | %d | %d | %d | %.1f%% |\n",
			colLabel(vk), counts[StateVerified], counts[StatePartial],
			counts[StateIncomplete], counts[StateNA], counts[StateConflict], pct)
	}
	b.WriteString("\n")
}

func renderConflicts(b *strings.Builder, m Matrix, versionKeys []string) {
	b.WriteString("## Conflicts\n\n")
	any := false
	for _, r := range m.Rows {
		for _, vk := range versionKeys {
			if c := r.Cells[vk]; c.State == StateConflict {
				any = true
				name := r.Op
				if name == "" {
					name = r.Packet
				}
				fmt.Fprintf(b, "- 🟥 **%s** × %s — %s\n", name, colLabel(vk), c.Note)
			}
		}
	}
	if !any {
		b.WriteString("None.\n")
	}
	b.WriteString("\n")
}

// RenderJSON produces status.json with the identical data (design §9).
func RenderJSON(m Matrix) ([]byte, error) {
	return json.MarshalIndent(m, "", "  ")
}
```

- [ ] **Step 5: Generate the golden, then run all matrix tests**

```bash
UPDATE_GOLDEN=1 go test ./internal/matrix/ -run TestBuildAndRenderGolden
go test ./internal/matrix/ -v
```
Expected: PASS. Inspect `testdata/golden_STATUS.md` by hand once — the v83
column must show AuthResult 🟡 (tool ✅, no byte-test), ACCOUNT_INFO ❌ in v83
(present, routed nowhere relevant... verify against the fixture's `Routed`
maps) and 🟥 in v87, StatRegistry under Sub-structs.

- [ ] **Step 6: Commit**

```bash
cd ../..
git add tools/packet-audit/internal/matrix/
git commit -m "task-085: matrix assembly + STATUS.md/status.json rendering"
```

## Task 1.8: `matrix` subcommand + first real STATUS.md

**Files:**
- Create: `tools/packet-audit/cmd/matrix.go`
- Create: `tools/packet-audit/cmd/matrix_test.go`
- Modify: `tools/packet-audit/cmd/root.go`
- Create (generated): `docs/packets/audits/STATUS.md`, `docs/packets/audits/status.json`

- [ ] **Step 1: Write the failing subcommand test**

`cmd/matrix_test.go` — drive `runMatrix` against a temp tree assembled from the
`internal/matrix/testdata` fixtures (registry yaml from Task 1.2 testdata, one
audit dir, one template), assert exit 0 and that both output files are written
and byte-identical across two runs:

```go
package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestMatrixSubcommandWritesOutputs(t *testing.T) {
	root := t.TempDir()
	// Layout: registry/, audits/gms_v83/, templates dir, exports dir.
	mustCopy(t, filepath.Join("..", "internal", "opregistry", "testdata", "good_version.yaml"),
		filepath.Join(root, "registry", "gms_v83.yaml"))
	mustCopy(t, filepath.Join("..", "internal", "matrix", "testdata", "audits", "gms_v83", "Invite.json"),
		filepath.Join(root, "audits", "gms_v83", "Invite.json"))
	mustCopy(t, filepath.Join("..", "internal", "matrix", "testdata", "templates", "template_gms_83_1.json"),
		filepath.Join(root, "templates", "template_gms_83_1.json"))
	mustCopy(t, filepath.Join("testdata", "gms_v95_mini.json"),
		filepath.Join(root, "exports", "gms_v83.json"))

	args := []string{
		"--registry-dir", filepath.Join(root, "registry"),
		"--audits-dir", filepath.Join(root, "audits"),
		"--templates-dir", filepath.Join(root, "templates"),
		"--exports-dir", filepath.Join(root, "exports"),
		"--versions", "gms_v83",
		"--out-dir", filepath.Join(root, "audits"),
	}
	if code := runMatrix(args, os.Stderr); code != 0 {
		t.Fatalf("matrix exit = %d", code)
	}
	md1, err := os.ReadFile(filepath.Join(root, "audits", "STATUS.md"))
	if err != nil {
		t.Fatalf("STATUS.md not written: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "audits", "status.json")); err != nil {
		t.Fatalf("status.json not written: %v", err)
	}
	if code := runMatrix(args, os.Stderr); code != 0 {
		t.Fatalf("second run exit = %d", code)
	}
	md2, _ := os.ReadFile(filepath.Join(root, "audits", "STATUS.md"))
	if !bytes.Equal(md1, md2) {
		t.Error("matrix output not deterministic")
	}
}

func mustCopy(t *testing.T, src, dst string) {
	t.Helper()
	b, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, b, 0o644); err != nil {
		t.Fatal(err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./cmd/ -run TestMatrixSubcommand
```
Expected: FAIL (`runMatrix` undefined).

- [ ] **Step 3: Implement `cmd/matrix.go`**

```go
package cmd

import (
	"crypto/sha256"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Chronicle20/atlas/tools/packet-audit/internal/matrix"
	"github.com/Chronicle20/atlas/tools/packet-audit/internal/opregistry"
	"github.com/Chronicle20/atlas/tools/packet-audit/internal/template"
)

type matrixOpts struct {
	RegistryDir  string
	AuditsDir    string
	TemplatesDir string
	ExportsDir   string
	EvidenceDir  string // consumed from Phase 2 on; empty = no evidence
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
	fs.StringVar(&o.PacketLibDir, "packet-lib", "libs/atlas-packet", "atlas-packet root for marker scanning")
	fs.StringVar(&versionsCSV, "versions", strings.Join(matrix.VersionKeys, ","), "comma-separated version keys")
	fs.StringVar(&o.OutDir, "out-dir", "docs/packets/audits", "output dir for STATUS.md/status.json")
	fs.BoolVar(&o.Check, "check", false, "CI mode: verify committed outputs are current; fail on conflicts/drift")
	if err := fs.Parse(args); err != nil {
		return 3
	}
	o.Versions = strings.Split(versionsCSV, ",")
	return matrixRun(o, os.Stdout, stderr)
}

func matrixRun(o matrixOpts, stdout, stderr io.Writer) int {
	reg, err := opregistry.LoadDir(o.RegistryDir, o.Versions)
	if err != nil {
		fmt.Fprintf(stderr, "packet-audit matrix: %v\n", err)
		return 1
	}
	in := matrix.Inputs{Registry: reg,
		Reports:        map[string]map[string]matrix.LoadedReport{},
		Routed:         map[string]map[matrix.RouteKey]bool{},
		RoutedAnywhere: map[matrix.RouteKey]bool{},
		Evidence:       map[matrix.EvKey]matrix.EvidenceStatus{},
		Tier1:          map[string]bool{},
		Markers:        map[matrix.EvKey]matrix.MarkerStatus{},
	}
	hashes := map[string]string{}
	for _, vk := range o.Versions {
		reps, err := matrix.LoadReports(filepath.Join(o.AuditsDir, vk))
		if err != nil {
			fmt.Fprintf(stderr, "packet-audit matrix: %v\n", err)
			return 1
		}
		in.Reports[vk] = reps
		in.Routed[vk] = map[matrix.RouteKey]bool{}
		tp := templatePathIn(o.TemplatesDir, vk)
		if t, err := template.Load(tp); err == nil {
			for op := range t.Writers() {
				k := matrix.RouteKey{Opcode: op, Dir: opregistry.DirClientbound}
				in.Routed[vk][k] = true
				in.RoutedAnywhere[k] = true
			}
			for op := range t.Handlers() {
				k := matrix.RouteKey{Opcode: op, Dir: opregistry.DirServerbound}
				in.Routed[vk][k] = true
				in.RoutedAnywhere[k] = true
			}
		} else {
			fmt.Fprintf(stderr, "packet-audit matrix: warning: no template for %s (%v)\n", vk, err)
		}
		ep := exportPathIn(o.ExportsDir, vk)
		if raw, err := os.ReadFile(ep); err == nil {
			hashes[vk] = fmt.Sprintf("%x", sha256.Sum256(raw))
		}
	}
	m := matrix.Build(in, o.Versions)
	m.ExportHashes = hashes
	m.ToolSHA = toolTreeSHA()

	md := matrix.RenderMarkdown(m, o.Versions)
	js, err := matrix.RenderJSON(m)
	if err != nil {
		fmt.Fprintf(stderr, "packet-audit matrix: %v\n", err)
		return 1
	}
	mdPath := filepath.Join(o.OutDir, "STATUS.md")
	jsPath := filepath.Join(o.OutDir, "status.json")

	if o.Check {
		return matrixCheck(m, md, js, mdPath, jsPath, stderr)
	}
	if err := os.WriteFile(mdPath, []byte(md), 0o644); err != nil {
		fmt.Fprintf(stderr, "packet-audit matrix: %v\n", err)
		return 1
	}
	if err := os.WriteFile(jsPath, js, 0o644); err != nil {
		fmt.Fprintf(stderr, "packet-audit matrix: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "wrote %s and %s\n", mdPath, jsPath)
	return 0
}

// matrixCheck implements --check (full semantics finish in Phase 2 Task 2.4;
// at this stage: committed-output freshness only).
func matrixCheck(m matrix.Matrix, md string, js []byte, mdPath, jsPath string, stderr io.Writer) int {
	fail := false
	if cur, err := os.ReadFile(mdPath); err != nil || string(cur) != md {
		fmt.Fprintf(stderr, "matrix --check: %s is stale — regenerate with `packet-audit matrix` and commit\n", mdPath)
		fail = true
	}
	if cur, err := os.ReadFile(jsPath); err != nil || string(cur) != string(js) {
		fmt.Fprintf(stderr, "matrix --check: %s is stale\n", jsPath)
		fail = true
	}
	if fail {
		return 1
	}
	return 0
}

func templatePathIn(dir, vk string) string {
	return filepath.Join(dir, filepath.Base(matrix.TemplatePath(vk)))
}

func exportPathIn(dir, vk string) string {
	return filepath.Join(dir, filepath.Base(matrix.ExportPath(vk)))
}

// toolTreeSHA returns `git rev-parse HEAD:tools/packet-audit` (the tree SHA of
// the tool itself), or "unknown" outside a git checkout.
func toolTreeSHA() string {
	out, err := exec.Command("git", "rev-parse", "HEAD:tools/packet-audit").Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(out))
}
```

Two mechanical adjustments this implies (do them in this step):
1. Export `RouteKey` and `EvKey` from `internal/matrix` (rename `routeKey` →
   `RouteKey`, `evKey` → `EvKey`, fields exported) so `cmd` can build `Inputs`.
   Update the Task 1.5/1.6/1.7 files accordingly — tests keep their semantics.
2. Wire dispatch in `cmd/root.go`:

```go
	if len(args) > 0 && args[0] == "matrix" {
		return runMatrix(args[1:], stderr)
	}
```

- [ ] **Step 4: Run the tests**

```bash
go test ./cmd/ -run TestMatrixSubcommand -v && go test ./... && go vet ./...
```
Expected: PASS, module green.

- [ ] **Step 5: Generate the first real STATUS.md**

From the worktree root:

```bash
go run ./tools/packet-audit matrix
head -60 docs/packets/audits/STATUS.md
grep -c '🟥' docs/packets/audits/STATUS.md || true
```
Expect: a large table; mostly 🟡/❌ (no markers/evidence yet — design §12
phase 1 "graded honestly"); v84 column ❌/⬜; a non-empty Conflicts section
(seed inaccuracy noise is expected at this stage — design §15 risk 3; Phase 5
discovery repairs it). **Do not “fix” conflicts by editing the registry now**
unless a spot-check against the CSV shows a transcription bug in the seeder
itself.

Sanity-check one known case from the retrospective: `grep 'LOGIN_STATUS' docs/packets/audits/STATUS.md`
should show v83 non-⬜.

- [ ] **Step 6: Commit (include a one-paragraph honesty note in the message)**

```bash
git add tools/packet-audit/cmd/ docs/packets/audits/STATUS.md docs/packets/audits/status.json
git commit -m "task-085: matrix subcommand + first honest STATUS.md baseline

First generated coverage matrix. Mostly partial/incomplete by design:
verified requires byte-test linkage (phase 3); evidence ledger lands in
phase 2; conflict noise from CSV seed gaps is expected until discover-ops
(phase 5) reconciles the registry against the IDBs."
git branch --show-current
```

---

# Phase 2 — Evidence ledger + drift check

## Task 2.1: `internal/evidence` — schema, loader, canonical function hash

**Files:**
- Create: `tools/packet-audit/internal/evidence/evidence.go`
- Create: `tools/packet-audit/internal/evidence/hash.go`
- Create: `tools/packet-audit/internal/evidence/evidence_test.go`
- Create: `tools/packet-audit/internal/evidence/testdata/buddy.clientbound.Invite.yaml`
- Create: `tools/packet-audit/internal/evidence/testdata/export_mini.json`

**Hash basis (context.md D1):** the IDA exports do not store decompile text —
they store the parsed function record (`address`, `direction`, `calls`, ...).
`decompile_sha256` is therefore sha256 over the **canonical re-marshaled JSON
of that function's entry** in the version's export file (unmarshal to
`map[string]any`, re-marshal with `encoding/json` — Go sorts map keys, giving
a stable byte form). The same helper serves `evidence pin` and `matrix --check`,
so they cannot disagree (design §6.4).

- [ ] **Step 1: Write the failing tests**

`evidence_test.go`:

```go
package evidence

import (
	"path/filepath"
	"testing"
)

func TestLoadRecord(t *testing.T) {
	r, err := LoadRecord(filepath.Join("testdata", "buddy.clientbound.Invite.yaml"))
	if err != nil {
		t.Fatalf("LoadRecord: %v", err)
	}
	if r.Packet != "buddy/clientbound/Invite" || r.Version != "gms_v83" {
		t.Errorf("record = %+v", r)
	}
	if r.Category != "OPAQUE" || r.IDA.Function != "CWvsContext::OnFriendResult" {
		t.Errorf("record = %+v", r)
	}
	if len(r.Verifies) != 1 {
		t.Errorf("verifies = %v", r.Verifies)
	}
}

func TestLoadRecordRejectsBadCategory(t *testing.T) {
	_, err := loadRecordBytes([]byte(
		"packet: a/b/C\ndirection: clientbound\nversion: gms_v83\ncategory: BOGUS\nida:\n  function: F\n  address: 0x1\n  decompile_sha256: aa\n"), "x.yaml")
	if err == nil {
		t.Fatal("expected category validation error")
	}
}

func TestFunctionHashStableAndDriftSensitive(t *testing.T) {
	h1, err := FunctionHash(filepath.Join("testdata", "export_mini.json"), "CLogin::OnFoo")
	if err != nil {
		t.Fatalf("FunctionHash: %v", err)
	}
	h2, _ := FunctionHash(filepath.Join("testdata", "export_mini.json"), "CLogin::OnFoo")
	if h1 != h2 {
		t.Error("hash not stable")
	}
	hBar, _ := FunctionHash(filepath.Join("testdata", "export_mini.json"), "CLogin::OnBar")
	if h1 == hBar {
		t.Error("different functions must hash differently")
	}
	if _, err := FunctionHash(filepath.Join("testdata", "export_mini.json"), "CLogin::Missing"); err == nil {
		t.Error("missing function must error (citation unresolvable, design §13)")
	}
}
```

`testdata/buddy.clientbound.Invite.yaml` (schema per design §6.1):

```yaml
packet: buddy/clientbound/Invite
direction: clientbound
version: gms_v83
category: OPAQUE
ida:
  function: "CWvsContext::OnFriendResult"
  address: 0xa3f2e8
  decompile_sha256: "ab12cd34"
verifies:
  - libs/atlas-packet/buddy/clientbound/invite_test.go#TestInviteByteOutput
notes: >
  Optional human context. Never consulted by grading.
```

`testdata/export_mini.json` — copy `cmd/testdata/gms_v95_mini.json` (it already
has `CLogin::OnFoo` / `CLogin::OnBar`).

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/evidence/
```
Expected: FAIL (package does not exist).

- [ ] **Step 3: Implement**

`evidence.go`:

```go
// Package evidence is the structured per-(packet,version) evidence ledger
// replacing prose acceptance (task-085 design §6).
package evidence

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

var validCategories = map[string]bool{
	"OPAQUE": true, "TRUNCATION": true, "REPRESENTATION": true,
	"OP-MODE-PREFIX": true, "LOOP-EXCLUSIVE-BRANCH": true,
	"VERSION-ABSENT": true, "TIER1-FIXTURE": true,
}

type IDACitation struct {
	Function       string `yaml:"function"`
	Address        string `yaml:"address"`
	DecompileSHA256 string `yaml:"decompile_sha256"`
}

type Record struct {
	Packet    string      `yaml:"packet"`
	Direction string      `yaml:"direction"`
	Version   string      `yaml:"version"`
	Category  string      `yaml:"category"`
	IDA       IDACitation `yaml:"ida"`
	Verifies  []string    `yaml:"verifies,omitempty"`
	Notes     string      `yaml:"notes,omitempty"`
}

func LoadRecord(path string) (Record, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Record{}, err
	}
	return loadRecordBytes(raw, path)
}

func loadRecordBytes(raw []byte, path string) (Record, error) {
	var r Record
	if err := yaml.Unmarshal(raw, &r); err != nil {
		return Record{}, fmt.Errorf("%s: %w", path, err)
	}
	if r.Packet == "" || r.Version == "" || r.IDA.Function == "" || r.IDA.Address == "" {
		return Record{}, fmt.Errorf("%s: packet/version/ida.function/ida.address are required", path)
	}
	if !validCategories[r.Category] {
		return Record{}, fmt.Errorf("%s: invalid category %q", path, r.Category)
	}
	if r.Direction != "clientbound" && r.Direction != "serverbound" {
		return Record{}, fmt.Errorf("%s: invalid direction %q", path, r.Direction)
	}
	return r, nil
}

// RecordPath is the canonical file location for a record:
// <dir>/<version>/<packet with / -> .>.yaml
func RecordPath(dir, version, packet string) string {
	return filepath.Join(dir, version, strings.ReplaceAll(packet, "/", ".")+".yaml")
}

// LoadDir loads every record under dir (one subdir per version).
func LoadDir(dir string) ([]Record, error) {
	var out []Record
	versions, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	for _, v := range versions {
		if !v.IsDir() {
			continue
		}
		files, err := os.ReadDir(filepath.Join(dir, v.Name()))
		if err != nil {
			return nil, err
		}
		for _, f := range files {
			if !strings.HasSuffix(f.Name(), ".yaml") {
				continue
			}
			r, err := LoadRecord(filepath.Join(dir, v.Name(), f.Name()))
			if err != nil {
				return nil, err
			}
			if r.Version != v.Name() {
				return nil, fmt.Errorf("%s: version %q does not match directory %q",
					f.Name(), r.Version, v.Name())
			}
			out = append(out, r)
		}
	}
	return out, nil
}
```

`hash.go`:

```go
package evidence

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
)

// FunctionHash computes the canonical sha256 of one function's record in an
// IDA export JSON. Canonical form: unmarshal the function entry to
// map[string]any and re-marshal with encoding/json (sorted keys). Errors when
// the function is absent — the caller renders "citation unresolvable".
func FunctionHash(exportPath, fname string) (string, error) {
	raw, err := os.ReadFile(exportPath)
	if err != nil {
		return "", err
	}
	var file struct {
		Functions map[string]json.RawMessage `json:"functions"`
	}
	if err := json.Unmarshal(raw, &file); err != nil {
		return "", fmt.Errorf("%s: %w", exportPath, err)
	}
	entry, ok := file.Functions[fname]
	if !ok {
		return "", fmt.Errorf("%s: function %q not in export (citation unresolvable)", exportPath, fname)
	}
	var v any
	if err := json.Unmarshal(entry, &v); err != nil {
		return "", err
	}
	canon, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", sha256.Sum256(canon)), nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/evidence/ -v
```
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
cd ../..
git add tools/packet-audit/internal/evidence/
git commit -m "task-085: evidence ledger — schema, loader, canonical function hash"
```

## Task 2.2: `evidence pin` subcommand

**Files:**
- Create: `tools/packet-audit/cmd/evidence.go`
- Create: `tools/packet-audit/cmd/evidence_test.go`
- Modify: `tools/packet-audit/cmd/root.go`

- [ ] **Step 1: Write the failing test**

```go
package cmd

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/Chronicle20/atlas/tools/packet-audit/internal/evidence"
)

func TestEvidencePinScaffoldsRecord(t *testing.T) {
	out := t.TempDir()
	code := runEvidence([]string{
		"pin",
		"--packet", "login/clientbound/Foo",
		"--version", "gms_v83",
		"--ida", "CLogin::OnFoo",
		"--category", "TIER1-FIXTURE",
		"--export", filepath.Join("testdata", "gms_v95_mini.json"),
		"--evidence-dir", out,
	}, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("pin exit = %d", code)
	}
	r, err := evidence.LoadRecord(evidence.RecordPath(out, "gms_v83", "login/clientbound/Foo"))
	if err != nil {
		t.Fatalf("record not written/loadable: %v", err)
	}
	if r.IDA.Function != "CLogin::OnFoo" || r.IDA.DecompileSHA256 == "" || r.IDA.Address == "" {
		t.Errorf("record = %+v", r)
	}
	// Pin must use the same hash code path as --check.
	want, _ := evidence.FunctionHash(filepath.Join("testdata", "gms_v95_mini.json"), "CLogin::OnFoo")
	if r.IDA.DecompileSHA256 != want {
		t.Errorf("hash mismatch pin=%s want=%s", r.IDA.DecompileSHA256, want)
	}
}

func TestEvidencePinUnresolvableCitationFails(t *testing.T) {
	code := runEvidence([]string{
		"pin", "--packet", "x/clientbound/Y", "--version", "gms_v83",
		"--ida", "CLogin::Missing", "--category", "OPAQUE",
		"--export", filepath.Join("testdata", "gms_v95_mini.json"),
		"--evidence-dir", t.TempDir(),
	}, &bytes.Buffer{})
	if code == 0 {
		t.Fatal("pin must fail when the export lacks the cited function")
	}
}
```

- [ ] **Step 2: Run to verify failure, then implement `cmd/evidence.go`**

```go
package cmd

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/Chronicle20/atlas/tools/packet-audit/internal/evidence"
	"github.com/Chronicle20/atlas/tools/packet-audit/internal/matrix"
	"gopkg.in/yaml.v3"
)

func runEvidence(args []string, stderr io.Writer) int {
	if len(args) == 0 || args[0] != "pin" {
		fmt.Fprintln(stderr, "packet-audit evidence: unknown subcommand (expected: pin)")
		return 3
	}
	fs := flag.NewFlagSet("packet-audit evidence pin", flag.ContinueOnError)
	fs.SetOutput(stderr)
	packet := fs.String("packet", "", "packet id, e.g. buddy/clientbound/Invite (required)")
	version := fs.String("version", "", "version key, e.g. gms_v83 (required)")
	ida := fs.String("ida", "", "IDA function name as it appears in the export (required)")
	category := fs.String("category", "", "evidence category (required)")
	export := fs.String("export", "", "export JSON path (default: derived from version)")
	dir := fs.String("evidence-dir", "docs/packets/evidence", "evidence ledger dir")
	if err := fs.Parse(args[1:]); err != nil {
		return 3
	}
	if *packet == "" || *version == "" || *ida == "" || *category == "" {
		fmt.Fprintln(stderr, "packet-audit evidence pin: --packet, --version, --ida, --category are required")
		return 3
	}
	if *export == "" {
		*export = matrix.ExportPath(*version)
	}
	hash, err := evidence.FunctionHash(*export, *ida)
	if err != nil {
		fmt.Fprintf(stderr, "packet-audit evidence pin: %v\n", err)
		return 1
	}
	addr, err := functionAddress(*export, *ida)
	if err != nil {
		fmt.Fprintf(stderr, "packet-audit evidence pin: %v\n", err)
		return 1
	}
	dirOf := "clientbound"
	if strings.Contains(*packet, "/serverbound/") {
		dirOf = "serverbound"
	}
	rec := evidence.Record{
		Packet: *packet, Direction: dirOf, Version: *version, Category: *category,
		IDA: evidence.IDACitation{Function: *ida, Address: addr, DecompileSHA256: hash},
	}
	raw, err := yaml.Marshal(rec)
	if err != nil {
		fmt.Fprintf(stderr, "packet-audit evidence pin: %v\n", err)
		return 1
	}
	p := evidence.RecordPath(*dir, *version, *packet)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		fmt.Fprintf(stderr, "packet-audit evidence pin: %v\n", err)
		return 1
	}
	if err := os.WriteFile(p, raw, 0o644); err != nil {
		fmt.Fprintf(stderr, "packet-audit evidence pin: %v\n", err)
		return 1
	}
	fmt.Fprintf(os.Stdout, "pinned %s\n", p)
	return 0
}

func functionAddress(exportPath, fname string) (string, error) {
	raw, err := os.ReadFile(exportPath)
	if err != nil {
		return "", err
	}
	var file struct {
		Functions map[string]struct {
			Address string `json:"address"`
		} `json:"functions"`
	}
	if err := json.Unmarshal(raw, &file); err != nil {
		return "", err
	}
	fn, ok := file.Functions[fname]
	if !ok {
		return "", fmt.Errorf("%s: function %q not in export", exportPath, fname)
	}
	return fn.Address, nil
}
```

Wire dispatch in root.go:

```go
	if len(args) > 0 && args[0] == "evidence" {
		return runEvidence(args[1:], stderr)
	}
```

- [ ] **Step 3: Run tests, then commit**

```bash
go test ./cmd/ -run TestEvidencePin -v && go test ./... && go vet ./...
cd ../..
git add tools/packet-audit/cmd/
git commit -m "task-085: evidence pin subcommand"
```

## Task 2.3: Wire evidence + drift into `matrix` (and finish `--check`)

**Files:**
- Modify: `tools/packet-audit/cmd/matrix.go`
- Create: `tools/packet-audit/internal/matrix/evidence_input.go`
- Create: `tools/packet-audit/internal/matrix/evidence_input_test.go`

- [ ] **Step 1: Write the failing test (drift degrades a cell, dangling evidence fails check)**

`evidence_input_test.go`:

```go
package matrix

import (
	"os"
	"path/filepath"
	"testing"
)

// writeMini writes an export with one function entry whose body we can mutate
// to simulate drift.
func writeMini(t *testing.T, dir, body string) string {
	t.Helper()
	p := filepath.Join(dir, "gms_v83.json")
	content := `{"binary":"x","md5":"0","generated_at":"0","functions":{"CLogin::OnFoo":` + body + `}}`
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestEvidenceInputFreshAndDrifted(t *testing.T) {
	dir := t.TempDir()
	exp := writeMini(t, dir, `{"address":"0x1","direction":"clientbound","calls":[{"op":"Decode4","comment":"a"}]}`)

	evDir := filepath.Join(dir, "evidence")
	os.MkdirAll(filepath.Join(evDir, "gms_v83"), 0o755)
	// Pin against the current export via the shared hash helper.
	rec := "packet: login/clientbound/Foo\ndirection: clientbound\nversion: gms_v83\ncategory: OPAQUE\nida:\n  function: \"CLogin::OnFoo\"\n  address: \"0x1\"\n  decompile_sha256: \"HASH\"\n"
	h := mustHash(t, exp, "CLogin::OnFoo")
	os.WriteFile(filepath.Join(evDir, "gms_v83", "login.clientbound.Foo.yaml"),
		[]byte(replaceHash(rec, h)), 0o644)

	st, problems, err := BuildEvidenceInputs(evDir, map[string]string{"gms_v83": exp})
	if err != nil {
		t.Fatal(err)
	}
	if len(problems) != 0 {
		t.Fatalf("unexpected problems: %v", problems)
	}
	if !st[EvKey{"login/clientbound/Foo", "gms_v83"}].Fresh {
		t.Error("expected fresh evidence")
	}

	// Mutate the export -> drift.
	writeMini(t, dir, `{"address":"0x1","direction":"clientbound","calls":[{"op":"Decode1","comment":"CHANGED"}]}`)
	st2, problems2, _ := BuildEvidenceInputs(evDir, map[string]string{"gms_v83": exp})
	if st2[EvKey{"login/clientbound/Foo", "gms_v83"}].Fresh {
		t.Error("expected stale evidence after export change")
	}
	if len(problems2) == 0 {
		t.Error("drift must be reported as a check problem")
	}
}

func mustHash(t *testing.T, exp, fn string) string { /* call evidence.FunctionHash */ 
	t.Helper()
	h, err := evidenceFunctionHash(exp, fn)
	if err != nil {
		t.Fatal(err)
	}
	return h
}

func replaceHash(rec, h string) string {
	return stringsReplaceAll(rec, "HASH", h)
}
```

(Use real `strings.ReplaceAll` and a thin `evidenceFunctionHash` alias to
`evidence.FunctionHash` — spelled out here only to keep the import story
obvious.)

- [ ] **Step 2: Implement `evidence_input.go`**

```go
package matrix

import (
	"fmt"

	"github.com/Chronicle20/atlas/tools/packet-audit/internal/evidence"
)

// BuildEvidenceInputs loads the ledger and grades freshness per record.
// problems collects --check failures: hash drift, unresolvable citations,
// dangling versions (no export on disk).
func BuildEvidenceInputs(evidenceDir string, exportPaths map[string]string) (map[EvKey]EvidenceStatus, []string, error) {
	recs, err := evidence.LoadDir(evidenceDir)
	if err != nil {
		return nil, nil, err
	}
	out := map[EvKey]EvidenceStatus{}
	var problems []string
	for _, r := range recs {
		k := EvKey{Packet: r.Packet, Version: r.Version}
		exp, ok := exportPaths[r.Version]
		if !ok {
			out[k] = EvidenceStatus{Exists: true, Fresh: false, Address: r.IDA.Address,
				Note: "no IDA export for " + r.Version}
			problems = append(problems, fmt.Sprintf("evidence %s×%s: no export for version", r.Packet, r.Version))
			continue
		}
		h, err := evidence.FunctionHash(exp, r.IDA.Function)
		if err != nil {
			out[k] = EvidenceStatus{Exists: true, Fresh: false, Address: r.IDA.Address,
				Note: "citation unresolvable: " + r.IDA.Function}
			problems = append(problems, fmt.Sprintf("evidence %s×%s: %v", r.Packet, r.Version, err))
			continue
		}
		fresh := h == r.IDA.DecompileSHA256
		if !fresh {
			problems = append(problems, fmt.Sprintf("evidence %s×%s: decompile hash drift (re-pin after review)", r.Packet, r.Version))
		}
		out[k] = EvidenceStatus{Exists: true, Fresh: fresh, Address: r.IDA.Address}
	}
	return out, problems, nil
}
```

- [ ] **Step 3: Wire into `cmd/matrix.go`**

In `matrixRun`, after export hashes are gathered:

```go
	exportPaths := map[string]string{}
	for _, vk := range o.Versions {
		p := exportPathIn(o.ExportsDir, vk)
		if _, err := os.Stat(p); err == nil {
			exportPaths[vk] = p
		}
	}
	evStatus, evProblems, err := matrix.BuildEvidenceInputs(o.EvidenceDir, exportPaths)
	if err != nil {
		fmt.Fprintf(stderr, "packet-audit matrix: %v\n", err)
		return 1
	}
	in.Evidence = evStatus
	// Design §13: an evidence record for a (packet, version) with no audit
	// report is dangling — a --check failure.
	for k := range evStatus {
		if _, ok := reportForPacket(in.Reports[k.Version], k.Packet); !ok {
			evProblems = append(evProblems,
				fmt.Sprintf("dangling evidence: %s × %s has no audit report", k.Packet, k.Version))
		}
	}
```

(`reportForPacket` is shown in Task 3.2; since Phase 2 lands first, define it
here — Task 3.2 then reuses it.)

And extend `matrixCheck` to take `evProblems []string` plus the built matrix's
conflicts (design §10.1 — conflicts are blockers, never allowlisted):

```go
func matrixCheck(m matrix.Matrix, md string, js []byte, mdPath, jsPath string, evProblems []string, stderr io.Writer) int {
	fail := false
	for _, p := range evProblems {
		fmt.Fprintf(stderr, "matrix --check: %s\n", p)
		fail = true
	}
	for _, r := range m.Rows {
		for vk, c := range r.Cells {
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
		return 1
	}
	return 0
}
```

- [ ] **Step 4: Tests + regenerate + commit**

```bash
go test ./... && go vet ./...
cd ../..
go run ./tools/packet-audit matrix     # picks up (still empty) evidence dir
git add tools/packet-audit/ docs/packets/audits/STATUS.md docs/packets/audits/status.json
git commit -m "task-085: evidence drift detection wired into matrix + full --check"
```

## Task 2.4: `tiers.yaml` + tier loader

**Files:**
- Create: `docs/packets/evidence/tiers.yaml`
- Create: `tools/packet-audit/internal/matrix/tiers.go`
- Create: `tools/packet-audit/internal/matrix/tiers_test.go`
- Modify: `tools/packet-audit/cmd/matrix.go`

**Tier semantics (context.md D4):** `tiers.yaml` is the explicit, reviewed
membership artifact (design §8). It has three stanzas; the *expansion* to a
packet set is deterministic tool code:
- `packets:` — explicit packet ids.
- `packet_prefixes:` — dispatcher families by package dir (e.g.
  `party/clientbound/` covers every mode of the party dispatcher).
- `opaque_types:` — Go type names; any packet whose pre-analyzed Call tree
  (`atlaspacket.TypeRegistry`, `registry.go:88+`) recurses into one is tier 1.
  **Phase-2 shortcut:** wiring TypeRegistry recursion in is allowed to slip to
  the same commit as Task 3.2 if it fights you here; until then `opaque_types`
  entries MUST be duplicated as `packet_prefixes`/`packets` so membership is
  not silently narrower than the design's 8 families.
- Additionally `FlatInvalid` reports are tier 1 unconditionally (already
  implemented in `grade.go`).

- [ ] **Step 1: Author `docs/packets/evidence/tiers.yaml`**

Derive the families from `docs/packets/audits/OPAQUE_LEDGER.md` (8 type
families, read its Ledger table) and design §8's dispatcher list:

```yaml
# Tier-1 membership (task-085 design §8): these packets can only reach
# `verified` through a linked byte-fixture test; the flat-diff verdict is
# advisory. Moving a packet between tiers is a reviewed edit to this file.

opaque_types:
  # the 8 opaque families from OPAQUE_LEDGER.md — copy the exact Go type
  # names from the ledger's rows when authoring this file for real
  - MobTemporaryStat
  - MovePath
  - CPetBody          # adjust to the ledger's exact type name
  - AvatarLook
  - Asset             # ItemSlotBase family
  - GuildMember
  - Visitor           # interaction Visitor/Room
  - GWCharacterStat

packet_prefixes:
  - party/
  - guild/
  - buddy/
  - messenger/
  - note/
  - npc/conversation/   # adjust to actual package dirs in libs/atlas-packet
  - interaction/
  - storage/
  - cash/
  - memo/

packets: []
```

**The type names and package dirs above are placeholders-by-construction**:
the authoring step is *reading the ledger and `ls libs/atlas-packet`* and
writing the real names. Verify each prefix exists:
`for p in party guild buddy messenger note interaction storage cash memo; do ls libs/atlas-packet/$p >/dev/null || echo "MISSING $p"; done`

- [ ] **Step 2: Failing test for the loader/expander**

```go
package matrix

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTierMembership(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "tiers.yaml"), []byte(
		"opaque_types:\n  - MovePath\npacket_prefixes:\n  - party/\npackets:\n  - login/clientbound/Special\n"), 0o644)
	tiers, err := LoadTiers(filepath.Join(dir, "tiers.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	cases := map[string]bool{
		"party/clientbound/Invite":      true,  // prefix
		"login/clientbound/Special":     true,  // explicit
		"login/clientbound/AuthResult":  false, // tier 0
	}
	for pkt, want := range cases {
		if got := tiers.IsTier1(pkt, nil); got != want {
			t.Errorf("IsTier1(%s) = %v, want %v", pkt, got, want)
		}
	}
	// opaque recursion: recurseTypes lists the packet's transitive RecurseType set
	if !tiers.IsTier1("monster/clientbound/Move", []string{"MovePath"}) {
		t.Error("opaque-type recursion must be tier 1")
	}
}
```

- [ ] **Step 3: Implement `tiers.go`**

```go
package matrix

import (
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type Tiers struct {
	OpaqueTypes    []string `yaml:"opaque_types"`
	PacketPrefixes []string `yaml:"packet_prefixes"`
	Packets        []string `yaml:"packets"`
}

func LoadTiers(path string) (Tiers, error) {
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return Tiers{}, nil // no tiers file: everything tier 0
	}
	if err != nil {
		return Tiers{}, err
	}
	var t Tiers
	if err := yaml.Unmarshal(raw, &t); err != nil {
		return Tiers{}, err
	}
	return t, nil
}

// IsTier1 reports tier-1 membership for a packet id. recurseTypes is the
// packet's transitive sub-struct type set (from atlaspacket.TypeRegistry);
// pass nil when unavailable.
func (t Tiers) IsTier1(packet string, recurseTypes []string) bool {
	for _, p := range t.Packets {
		if p == packet {
			return true
		}
	}
	for _, pre := range t.PacketPrefixes {
		if strings.HasPrefix(packet, pre) {
			return true
		}
	}
	for _, rt := range recurseTypes {
		for _, ot := range t.OpaqueTypes {
			if rt == ot {
				return true
			}
		}
	}
	return false
}
```

Wire into `cmd/matrix.go` (`--tiers` flag defaulting to
`docs/packets/evidence/tiers.yaml`; populate `in.Tier1[pkt]` for every loaded
report's packet id via `tiers.IsTier1(pkt, nil)` — TypeRegistry recursion
joins in Task 3.2).

- [ ] **Step 4: Tests + regenerate + commit**

```bash
go test ./... && go vet ./... && cd ../..
go run ./tools/packet-audit matrix
git add tools/packet-audit/ docs/packets/evidence/tiers.yaml docs/packets/audits/STATUS.md docs/packets/audits/status.json
git commit -m "task-085: tier-1 membership file + loader; tier-1 caps at partial"
```

## Task 2.5: Migrate recoverable prose acceptances into the ledger; freeze the prose files

**Files:**
- Create: `docs/packets/evidence/<version>/*.yaml` (one per recoverable acceptance)
- Modify: `docs/packets/ida-exports/_pending.md` (banner prepended)
- Modify: `docs/packets/audits/gms_v95/_pending.md` (banner prepended)
- Modify: `docs/packets/audits/OPAQUE_LEDGER.md` (banner prepended)

This is bounded content work driven by an explicit inventory. Migration rule
(design §6.3): migrate **only** entries whose IDA citation (function and/or
address) is recoverable from the prose AND whose function still resolves in
the current export — everything else stays unmigrated and its cell stays ❌
(the honest state; expect the "honesty shock", design §15 risk 1).

- [ ] **Step 1: Build the inventory**

```bash
grep -n '0x[0-9a-fA-F]\{4,\}' docs/packets/ida-exports/_pending.md | head -50
grep -n '0x[0-9a-fA-F]\{4,\}' docs/packets/audits/gms_v95/_pending.md | head -50
grep -n '|' docs/packets/audits/OPAQUE_LEDGER.md | head -40
```
Write the inventory as a scratch list (packet id, version, IDA function,
address, category, verifying test if named). The OPAQUE_LEDGER rows name their
verifying byte-tests — those become `verifies:` entries and `category: OPAQUE`.
`_pending` exclusion categories map onto the schema enum: analyzer-boundary →
`REPRESENTATION` or `LOOP-EXCLUSIVE-BRANCH` per entry wording, version-absent
modes → `VERSION-ABSENT`, truncation rows → `TRUNCATION`, op/mode prefix rows →
`OP-MODE-PREFIX`.

- [ ] **Step 2: Pin each recoverable entry**

For each inventory line (repeat per applicable version):

```bash
go run ./tools/packet-audit evidence pin \
  --packet <pkg/dir/Struct> --version <gms_v83|gms_v87|gms_v95|jms_v185> \
  --ida "<FName exactly as in the export>" --category <CATEGORY>
```
Then hand-edit the produced YAML to add `verifies:` (when the ledger names a
test) and a one-line `notes:` citing the source (`_pending.md §N` /
`OPAQUE_LEDGER row`). A pin that fails with "not in export" is **not
recoverable** — skip it, leave the cell ❌, list it in the commit message.

- [ ] **Step 3: Freeze the prose files**

Prepend to each of the three files:

```markdown
> **FROZEN (task-085).** This file is historical reference only. Machine-graded
> acceptance now lives in `docs/packets/evidence/` (one YAML per packet ×
> version, hash-pinned to the IDA exports) and renders in
> `docs/packets/audits/STATUS.md`. Entries here were migrated only where their
> IDA citation was recoverable; everything else grades ❌ in the matrix until
> verified through `docs/packets/audits/VERIFYING_A_PACKET.md`.
```

The `_unimplemented.json` allowlists are NOT frozen — `validate` still consumes
them (design §6.3); they just no longer influence grading.

- [ ] **Step 4: Regenerate, sanity-check the delta, commit**

```bash
go run ./tools/packet-audit matrix
git diff --stat docs/packets/audits/STATUS.md   # expect ❌→🟡 flips equal to migrated count
git add docs/packets/evidence/ docs/packets/ida-exports/_pending.md docs/packets/audits/gms_v95/_pending.md docs/packets/audits/OPAQUE_LEDGER.md docs/packets/audits/STATUS.md docs/packets/audits/status.json
git commit -m "task-085: migrate recoverable acceptances to evidence ledger; freeze prose registries

Migrated: <N> records (list). Not recoverable (stay ❌): <M> entries (list)."
git branch --show-current
```

---

# Phase 3 — Byte-test linkage

## Task 3.1: `internal/marker` — marker comment scanner

**Files:**
- Create: `tools/packet-audit/internal/marker/marker.go`
- Create: `tools/packet-audit/internal/marker/marker_test.go`
- Create: `tools/packet-audit/internal/marker/testdata/invite_test.go.txt`

Marker grammar (design §7), one line, scanned with a plain line scan (no AST):

```
// packet-audit:verify packet=buddy/clientbound/Invite version=gms_v83 ida=0xa3f2e8
```

- [ ] **Step 1: Create the testdata excerpt**

`testdata/invite_test.go.txt` (a `.txt` so `go build` ignores it — same
convention as `internal/atlaspacket/testdata/*.go.txt`):

```
package clientbound

import "testing"

// packet-audit:verify packet=buddy/clientbound/Invite version=gms_v83 ida=0xa3e31c
// packet-audit:verify packet=buddy/clientbound/Invite version=gms_v87 ida=0xad697a
func TestInviteByteOutput(t *testing.T) {}

// packet-audit:verify packet=buddy/clientbound/Invite
func TestBrokenMarker(t *testing.T) {}
```

- [ ] **Step 2: Write the failing tests**

```go
package marker

import (
	"path/filepath"
	"testing"
)

func TestScanFile(t *testing.T) {
	ms, errs := scanReader(mustOpen(t, filepath.Join("testdata", "invite_test.go.txt")), "invite_test.go")
	if len(ms) != 2 {
		t.Fatalf("markers = %d, want 2 (%v)", len(ms), ms)
	}
	if ms[0].Packet != "buddy/clientbound/Invite" || ms[0].Version != "gms_v83" || ms[0].Address != "0xa3e31c" {
		t.Errorf("marker0 = %+v", ms[0])
	}
	if ms[0].File != "invite_test.go" || ms[0].Line != 5 {
		t.Errorf("marker0 location = %s:%d", ms[0].File, ms[0].Line)
	}
	// Malformed marker (missing version/ida) is an error, not a silent skip.
	if len(errs) != 1 {
		t.Errorf("errs = %v, want 1 malformed-marker error", errs)
	}
}

func TestScanDuplicateMarkerIsError(t *testing.T) {
	ms, errs := scanString("// packet-audit:verify packet=a/b/C version=gms_v83 ida=0x1\n// packet-audit:verify packet=a/b/C version=gms_v83 ida=0x2\n", "x_test.go")
	_ = ms
	if len(errs) == 0 {
		t.Error("duplicate (packet,version) across markers must error (design §7: one marker per cell)")
	}
}
```

- [ ] **Step 3: Implement `marker.go`**

```go
// Package marker scans libs/atlas-packet test files for
// `packet-audit:verify` linkage comments (task-085 design §7).
package marker

import (
	"bufio"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type Marker struct {
	Packet  string
	Version string
	Address string
	File    string // path relative to scan root
	Line    int
}

const prefix = "// packet-audit:verify "

// Scan walks root for *_test.go files and collects all markers.
// Returned errs cover malformed markers and duplicate (packet,version) —
// both fail matrix --check.
func Scan(root string) ([]Marker, []string, error) {
	var all []Marker
	var errs []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, "_test.go") {
			return nil
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		rel, _ := filepath.Rel(root, path)
		ms, es := scanReader(f, filepath.ToSlash(rel))
		all = append(all, ms...)
		errs = append(errs, es...)
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	seen := map[string]Marker{}
	for _, m := range all {
		k := m.Packet + "|" + m.Version
		if prev, dup := seen[k]; dup {
			errs = append(errs, fmt.Sprintf("duplicate marker for %s × %s (%s:%d and %s:%d)",
				m.Packet, m.Version, prev.File, prev.Line, m.File, m.Line))
			continue
		}
		seen[k] = m
	}
	return all, errs, nil
}

func scanReader(r io.Reader, file string) ([]Marker, []string) {
	var ms []Marker
	var errs []string
	sc := bufio.NewScanner(r)
	line := 0
	seen := map[string]int{}
	for sc.Scan() {
		line++
		txt := strings.TrimSpace(sc.Text())
		if !strings.HasPrefix(txt, strings.TrimSpace(prefix)) {
			continue
		}
		m := Marker{File: file, Line: line}
		for _, kv := range strings.Fields(strings.TrimPrefix(txt, strings.TrimSpace(prefix))) {
			parts := strings.SplitN(kv, "=", 2)
			if len(parts) != 2 {
				continue
			}
			switch parts[0] {
			case "packet":
				m.Packet = parts[1]
			case "version":
				m.Version = parts[1]
			case "ida":
				m.Address = parts[1]
			}
		}
		if m.Packet == "" || m.Version == "" || m.Address == "" {
			errs = append(errs, fmt.Sprintf("%s:%d: malformed packet-audit:verify marker (need packet=, version=, ida=)", file, line))
			continue
		}
		k := m.Packet + "|" + m.Version
		if prev, dup := seen[k]; dup {
			errs = append(errs, fmt.Sprintf("%s: duplicate marker for %s × %s (lines %d and %d)", file, m.Packet, m.Version, prev, line))
			continue
		}
		seen[k] = line
		ms = append(ms, m)
	}
	return ms, errs
}

func scanString(s, file string) ([]Marker, []string) {
	return scanReader(strings.NewReader(s), file)
}
```

(Add the trivial `mustOpen` test helper.)

- [ ] **Step 4: Run tests, commit**

```bash
go test ./internal/marker/ -v && go test ./... && go vet ./...
cd ../..
git add tools/packet-audit/internal/marker/
git commit -m "task-085: byte-test marker scanner"
```

## Task 3.2: Wire markers into `matrix` — `verified` promotion + orphan checks

**Files:**
- Modify: `tools/packet-audit/cmd/matrix.go`
- Create: `tools/packet-audit/cmd/matrix_markers_test.go`

- [ ] **Step 1: Write the failing test**

End-to-end through `runMatrix` on a temp tree (reuse the Task 1.8 fixture
helper): add a fake `libs/atlas-packet` dir containing one `invite_test.go`
with a valid marker matching the Invite report's address, plus a fresh
evidence record for a tier-1 case. Assert:
- the tier-0 packet with marker renders ✅ in status.json,
- an orphan marker (marker whose `ida` address matches neither the evidence
  record nor the audit report address for that packet/version) makes
  `--check` exit non-zero with a message containing "orphan marker".

```go
func TestMatrixMarkerPromotionAndOrphanCheck(t *testing.T) {
	root := t.TempDir()
	// Same tree as TestMatrixSubcommandWritesOutputs...
	mustCopy(t, filepath.Join("..", "internal", "opregistry", "testdata", "good_version.yaml"),
		filepath.Join(root, "registry", "gms_v83.yaml"))
	mustCopy(t, filepath.Join("..", "internal", "matrix", "testdata", "audits", "gms_v83", "Invite.json"),
		filepath.Join(root, "audits", "gms_v83", "Invite.json"))
	mustCopy(t, filepath.Join("..", "internal", "matrix", "testdata", "templates", "template_gms_83_1.json"),
		filepath.Join(root, "templates", "template_gms_83_1.json"))
	mustCopy(t, filepath.Join("testdata", "gms_v95_mini.json"),
		filepath.Join(root, "exports", "gms_v83.json"))
	// ...plus a packet lib with a marker matching Invite.json's Address.
	lib := filepath.Join(root, "packetlib", "buddy", "clientbound")
	os.MkdirAll(lib, 0o755)
	marker := "package clientbound\n\n// packet-audit:verify packet=buddy/clientbound/Invite version=gms_v83 ida=0xa3f2e8\nfunc TestX(t *T) {}\n"
	os.WriteFile(filepath.Join(lib, "invite_test.go"), []byte(marker), 0o644)

	// NOTE: for the registry join to produce the Invite cell, gms_v83.yaml
	// must contain an entry whose fname matches Invite.json's IDAName base
	// ("CWvsContext::OnFriendResult"); extend the copied YAML in-place here:
	appendLine(t, filepath.Join(root, "registry", "gms_v83.yaml"),
		"- op: BUDDY_RESULT\n  direction: clientbound\n  opcode: 0x03F\n  fname: \"CWvsContext::OnFriendResult\"\n  provenance: csv-import\n")

	args := []string{
		"--registry-dir", filepath.Join(root, "registry"),
		"--audits-dir", filepath.Join(root, "audits"),
		"--templates-dir", filepath.Join(root, "templates"),
		"--exports-dir", filepath.Join(root, "exports"),
		"--evidence-dir", filepath.Join(root, "evidence"), // empty
		"--packet-lib", filepath.Join(root, "packetlib"),
		"--versions", "gms_v83",
		"--out-dir", filepath.Join(root, "audits"),
	}
	if code := runMatrix(args, os.Stderr); code != 0 {
		t.Fatalf("matrix exit = %d", code)
	}
	var m matrix.Matrix
	raw, _ := os.ReadFile(filepath.Join(root, "audits", "status.json"))
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	cell := findOpCell(t, m, "BUDDY_RESULT", "gms_v83")
	if cell.State.Name() != "verified" {
		t.Errorf("Invite cell = %s (%s), want verified", cell.State.Name(), cell.Note)
	}

	// Orphan: address matches nothing -> --check exits 1.
	orphan := "package clientbound\n\n// packet-audit:verify packet=buddy/clientbound/Invite version=gms_v83 ida=0xdeadbeef\nfunc TestX(t *T) {}\n"
	os.WriteFile(filepath.Join(lib, "invite_test.go"), []byte(orphan), 0o644)
	var errBuf bytes.Buffer
	if code := runMatrix(append(args, "--check"), &errBuf); code == 0 {
		t.Fatal("orphan marker must fail --check")
	}
	if !strings.Contains(errBuf.String(), "orphan marker") {
		t.Errorf("stderr = %q", errBuf.String())
	}
}
```

Add the two tiny helpers (`appendLine` opens O_APPEND and writes;
`findOpCell` scans `m.Rows` for Kind==RowOp && Op match and returns
`r.Cells[version]`, failing the test when absent). Note the BUDDY_RESULT
opcode must also be routed by the fixture template's writers list — add a
writer entry with opCode 0x03F to the trimmed template JSON, otherwise the
cross-version gap rule (D3) never fires and the cell grades through normally.

- [ ] **Step 2: Implement the wiring in `matrixRun`**

```go
	markers, markerErrs, err := marker.Scan(o.PacketLibDir)
	if err != nil {
		fmt.Fprintf(stderr, "packet-audit matrix: marker scan: %v\n", err)
		return 1
	}
	checkProblems := append([]string{}, evProblems...)
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
```

with the small helper:

```go
func reportForPacket(reps map[string]matrix.LoadedReport, pkt string) (matrix.LoadedReport, bool) {
	for _, r := range reps {
		if matrix.PacketID(r) == pkt {
			return r, true
		}
	}
	return matrix.LoadedReport{}, false
}
```

Orphan markers are `--check` failures (design §7) but do NOT fail plain
`matrix` generation — the cell simply doesn't promote. Also complete the
deferred `opaque_types` recursion from Task 2.4 here if it was deferred:
build the TypeRegistry once (`atlaspacket.NewTypeRegistry(o.PacketLibDir)`),
compute each report packet's transitive `RecurseType` set from
`TypeRegistry.Calls`, and pass it to `tiers.IsTier1`.

- [ ] **Step 3: Tests + commit**

```bash
go test ./... && go vet ./...
cd ../..
git add tools/packet-audit/
git commit -m "task-085: marker linkage — verified promotion + orphan --check"
```

## Task 3.3: Retrofit markers onto the existing byte-fixture tests

**Files:**
- Modify: ~34 test functions across `libs/atlas-packet/**/ *_test.go` (inventory below)
- Create: matching evidence records under `docs/packets/evidence/`
- Modify (regenerated): `docs/packets/audits/STATUS.md`, `status.json`

- [ ] **Step 1: Build the inventory**

```bash
grep -rln 'IDA' libs/atlas-packet --include='*_test.go' | sort
grep -rn '0x[0-9a-f]\{6,\}' libs/atlas-packet --include='*_test.go' | grep -i 'ida\|@0x' | head -60
```
Known representatives (from exploration): `party/clientbound/invite_test.go`,
`model/movement_test.go`, `cash/serverbound/shop_operation_gift_test.go`.
Expect ≈34 byte-fixture test functions citing IDA addresses in comments.

- [ ] **Step 2: For each test, add markers + evidence**

Per test function, for each version variant it asserts (the IDA comments name
function + address per version — e.g. invite_test.go cites
`v83 OnPartyResult@0xa3e31c` and `v87 ...@0xad697a`):

1. Add one marker line per (packet, version) directly above the function:
```go
// packet-audit:verify packet=party/clientbound/Invite version=gms_v83 ida=0xa3e31c
// packet-audit:verify packet=party/clientbound/Invite version=gms_v87 ida=0xad697a
// packet-audit:verify packet=party/clientbound/Invite version=gms_v95 ida=<v95 addr from comment>
// packet-audit:verify packet=party/clientbound/Invite version=jms_v185 ida=<jms addr if cited>
```
Only add a version's marker when the test actually asserts that version's
bytes AND the comment cites that version's address — do not infer addresses.

2. Pin the matching evidence record when none exists:
```bash
go run ./tools/packet-audit evidence pin --packet party/clientbound/Invite \
  --version gms_v83 --ida "CParty::OnPartyResult" --category TIER1-FIXTURE
```
then fill `verifies:` with `libs/atlas-packet/party/clientbound/invite_test.go#TestInviteByteOutput`.

**Address mismatch rule:** if the pinned export address differs from the test
comment's address, the export wins (it is current); update the marker to the
export's address and note the stale comment in the commit message. If the
function is missing from the export entirely, skip the marker for that version
(cell stays 🟡/❌) and list it.

3. After every ~10 tests: `go test ./libs/atlas-packet/... ` (markers are
comments — compilation cannot break, but keep the loop tight anyway).

- [ ] **Step 3: Regenerate + verify promotions**

```bash
go run ./tools/packet-audit matrix
go run ./tools/packet-audit matrix --check   # must exit 0 — no orphans
grep -c '✅' docs/packets/audits/STATUS.md   # must be > 0 now
```
Diff STATUS.md: every flip must be 🟡→✅ or ❌→🟡; any cell that DEGRADES
means a marker/evidence mistake — stop and fix before committing.

- [ ] **Step 4: Commit**

```bash
cd libs/atlas-packet && go test ./... && cd ../..
git add libs/atlas-packet docs/packets/evidence/ docs/packets/audits/STATUS.md docs/packets/audits/status.json
git commit -m "task-085: retrofit packet-audit:verify markers onto existing byte-fixture tests"
git branch --show-current
```

---

# Phase 4 — Playbook, skill, agent

## Task 4.1: `docs/packets/audits/VERIFYING_A_PACKET.md`

**Files:**
- Create: `docs/packets/audits/VERIFYING_A_PACKET.md`

- [ ] **Step 1: Write the playbook** — full content (the 8 steps of design
§11.1, written for an executor with zero context). The document must contain,
at minimum, these sections with this exact operational content:

```markdown
# Verifying a packet (single packet × single version)

The canonical procedure for promoting a coverage-matrix cell
(`docs/packets/audits/STATUS.md`) to ✅. Written for a human or an agent.
Hard rule (CLAUDE.md "Verification Over Memory"): every byte in a fixture must
trace to a decompile line — never to MapleStory knowledge from memory.

## 0. Prerequisites
- The five registry files in `docs/packets/registry/`.
- The version's IDA export in `docs/packets/ida-exports/` (jms_v185 uses
  `gms_jms_185.json`).
- For fresh decompiles: a live ida-pro-mcp instance with the version's IDB.

## 1. Resolve scope
Look the op up in `docs/packets/registry/<version>.yaml`. If absent there:
your job is confirming `n-a` (verify the template doesn't route the opcode in
`services/atlas-configurations/seed-data/templates/` and no Atlas struct
claims it) or filing a 🟥 conflict — then stop.

## 2. Check current state
- The cell in STATUS.md and `status.json`.
- Any evidence record: `docs/packets/evidence/<version>/<packet dots>.yaml`.
- The latest audit report: `docs/packets/audits/<version>/<Writer>.{json,md}`.

## 3. Decompile the client side
- Enumerate live instances (`mcp__ida-pro__list_instances`) and
  `select_instance` the one whose loaded IDB matches the target version —
  ports vary by IDA launch order, NEVER hardcode them.
- Decompile the registry entry's `fname` (batch `decompile`); descend into
  helper reads (address-based descent, same rule as the exporter).
- Write down the full ordered read/write list including guards and loop bounds.

## 4. Compare against Atlas
The encoder/decoder in `libs/atlas-packet/<pkg>/`, including version gates
(`MajorVersion()` comparisons — beware the v84 off-by-one class: `>83` must be
`>=87` when v84 matches v83). Divergence ⇒ wire fix FIRST (own commit, own
review), then continue.

## 5. Derive expected bytes
Concrete model fixture; hand-compute the byte sequence from the client read
order. One fixture per mode for mode-driven packets. Cite the decompile line
for every field in a comment.

## 6. Write the byte-test
With the marker above the function:
    // packet-audit:verify packet=<pkg/dir/Struct> version=<key> ida=<0xaddr>
Use the existing `pt.Variants` table pattern
(`libs/atlas-packet/party/clientbound/invite_test.go` is the reference).

## 7. Pin evidence
    go run ./tools/packet-audit evidence pin --packet <id> --version <key> \
      --ida "<FName>" --category TIER1-FIXTURE
Fill `verifies:` with `<test file>#<TestName>`.

## 8. Regenerate and verify promotion
    go run ./tools/packet-audit matrix
    go run ./tools/packet-audit matrix --check
The cell must now be ✅. Commit test + evidence + STATUS.md/status.json
together.

## Failure modes (design §13)
- `evidence pin` fails "not in export" → the citation is unresolvable; the
  export needs re-harvest (task-081 playbook) before this cell can verify.
- `matrix --check` reports an orphan marker → the marker's ida= address
  matches neither the evidence record nor the audit report; fix the address,
  never delete the check.
- Hash drift on an existing record → see STATUS degradation paths
  (design §10.2): cosmetic decompile churn → re-pin; material change →
  re-verify from step 3.
```

- [ ] **Step 2: Commit**

```bash
git add docs/packets/audits/VERIFYING_A_PACKET.md
git commit -m "task-085: VERIFYING_A_PACKET playbook"
```

## Task 4.2: `/verify-packet` command

**Files:**
- Create: `.claude/commands/verify-packet.md`

Format reference: `.claude/commands/convert-npc.md` (frontmatter:
`description` + `argument-hint`).

- [ ] **Step 1: Write the command file**

```markdown
---
description: Walk the VERIFYING_A_PACKET playbook for one packet × version — promote a coverage-matrix cell to verified
argument-hint: <packet id, e.g. buddy/clientbound/Invite> <version key, e.g. gms_v83>
---

You are verifying one packet × version cell of the packet coverage matrix.
Follow `docs/packets/audits/VERIFYING_A_PACKET.md` step by step. Read it FIRST.

Arguments: $ARGUMENTS (packet id + version key).

Non-negotiable rules:
1. **Verification Over Memory** — every byte in the fixture must trace to a
   decompile line you obtained in this session (or a checked-in export entry).
   If you cannot resolve a read order, STOP and report the cell as blocked;
   never fabricate from MapleStory knowledge.
2. Resolve the IDA instance by its loaded IDB, never by hardcoded port
   (list_instances → select_instance).
3. A wire divergence found in step 4 is its own commit (fix + test) BEFORE the
   verification commit.
4. The three artifacts land together in one commit: byte-test (with
   `packet-audit:verify` marker), evidence YAML, regenerated
   STATUS.md/status.json. `packet-audit matrix --check` must exit 0.
5. If the packet is tier-1 mode-driven, one fixture per mode (the registry
   entry and audit report's dispatch selectors enumerate the modes).

Output: the promoted cell (op × version, old state → new state) plus the
commit SHA, or a precise blocker (which playbook step, what failed).
```

- [ ] **Step 2: Commit**

```bash
git add .claude/commands/verify-packet.md
git commit -m "task-085: /verify-packet command"
```

## Task 4.3: `packet-verifier` agent

**Files:**
- Create: `.claude/agents/packet-verifier.md`

Format reference: `.claude/agents/backend-guidelines-reviewer.md` (YAML
frontmatter `name` + multiline `description` with `<example>` blocks).

- [ ] **Step 1: Write the agent definition**

```markdown
---
name: packet-verifier
description: |
  Use this agent to verify one packet × version cell of the packet coverage
  matrix (docs/packets/audits/STATUS.md): it follows
  docs/packets/audits/VERIFYING_A_PACKET.md, decompiles the client read order
  via ida-pro-mcp (or the checked-in export), writes the byte-fixture test
  with a packet-audit:verify marker, pins the evidence record, regenerates the
  matrix, and commits the three artifacts together. Dispatched in fan-out
  during tier-1 fixture campaigns — one agent per packet × version, batched
  per IDB. Output is machine-checked: a cell that does not promote is a
  failure report, never a prose claim.

  <example>
  Context: The party dispatcher family campaign is running.
  user: "Verify party/clientbound/UpdateParty for gms_v83."
  assistant: "Dispatching packet-verifier for party/clientbound/UpdateParty × gms_v83."
  </example>

  <example>
  Context: A matrix cell degraded after a re-export (hash drift).
  user: "Re-verify buddy/clientbound/Invite on v87 — the evidence went stale."
  assistant: "Dispatching packet-verifier to re-derive the read order and re-pin."
  </example>
---

You verify exactly one (packet, version) cell. You are working in the task
worktree given in your prompt — `cd` there first and verify the branch.

Procedure: follow `docs/packets/audits/VERIFYING_A_PACKET.md` literally,
steps 1–8. Constraints, in priority order:

1. NEVER fabricate bytes, opcodes, or read orders from MapleStory knowledge.
   Every fixture byte traces to a decompile line or export entry you cite
   (function + address) in the test comment.
2. Resolve IDA instances by loaded IDB via list_instances/select_instance.
   If no instance has the right IDB and the export lacks the function, STOP
   and report blocked.
3. Wire divergences (step 4) are a separate commit before the verification
   commit, with a byte-test proving the fix.
4. Final commit contains: the test (+marker), the evidence YAML, regenerated
   STATUS.md + status.json, and `packet-audit matrix --check` exits 0.
5. Report format: `<packet> × <version>: <old state> → <new state>, commit
   <sha>` or `BLOCKED at step <n>: <reason>`.
---
```

- [ ] **Step 2: Commit**

```bash
git add .claude/agents/packet-verifier.md
git commit -m "task-085: packet-verifier agent definition"
```

## Task 4.4: End-to-end validation of the promotion loop (3 packets)

**Files:**
- Create: up to 3 new/extended test files in `libs/atlas-packet/`
- Create: matching evidence records
- Modify (regenerated): STATUS.md, status.json

Validate the playbook before any campaign (design §12 phase 4): one tier-0,
one tier-1 mode-driven, one opaque-family member. **No live IDA needed** if
candidates are chosen whose FNames resolve in the checked-in exports — the
read order comes from the export's `calls` array (which IS the IDA-derived
read order, with comments); a live decompile is only required when the export
entry is missing or `unresolved`.

- [ ] **Step 1: Pick candidates from the matrix**

```bash
grep '🟡' docs/packets/audits/STATUS.md | head -30
```
Pick: (a) a tier-0 cell that is 🟡 "tool ✅ without byte-test" with a simple
shape (few calls in the export entry); (b) a tier-1 dispatcher cell (e.g. a
messenger or note mode) that already has a fresh evidence record; (c) an
opaque-family member listed in OPAQUE_LEDGER with an existing byte-test that
Task 3.3 didn't cover.

- [ ] **Step 2: Run the playbook for each, literally**

Follow `VERIFYING_A_PACKET.md` steps 1–8 per candidate. For step 3 use the
export entry (`docs/packets/ida-exports/<version>.json`, the function's
`calls` array) as the read-order source; record in the test comment that the
source was the export + its address. Write the byte-test with marker
(reference pattern: `party/clientbound/invite_test.go`), pin evidence, fill
`verifies:`, regenerate.

- [ ] **Step 3: Confirm promotions + checks**

```bash
go run ./tools/packet-audit matrix --check  # exit 0
git diff docs/packets/audits/STATUS.md      # exactly 3 cells flip to ✅
cd libs/atlas-packet && go test -race ./... && cd ../..
```
If a candidate does NOT promote, that is the validation working — debug the
loop (marker address? evidence freshness? tier rule?) before declaring this
task done. The loop being validated is the deliverable, not the 3 cells.

- [ ] **Step 4: Commit**

```bash
git add libs/atlas-packet docs/packets/evidence docs/packets/audits/STATUS.md docs/packets/audits/status.json
git commit -m "task-085: e2e-validate the promotion loop on three cells (tier-0, tier-1, opaque)"
git branch --show-current
```

---

# Phase 5 — Operation discovery from the IDBs

## Task 5.1: Clientbound dispatch-walk parser

**Files:**
- Create: `tools/packet-audit/internal/discover/discover.go`
- Create: `tools/packet-audit/internal/discover/discover_test.go`
- Create: `tools/packet-audit/internal/discover/testdata/process_packet_v83.c.txt`

Discovery reuses the existing MCP plumbing: `idasrc.MCPClient`
(`internal/idasrc/mcp.go:32-37` — `GetFunctionByName`, `DecompileFunction`,
`GetCallees`, `StructInfo`) and the Hex-Rays text conventions already handled
by `internal/idasrc/parse.go` (note: case labels can carry `u` suffixes, e.g.
`case 200u:`).

Clientbound algorithm: decompile the client dispatcher
(`CClientSocket::ProcessPacket` — flag-overridable name), parse its
`switch (op)` / cascading `if (op == N)` structure, and for each case record
(opcode, called handler address, demangled handler name) using `GetCallees`
to resolve call targets per case body.

- [ ] **Step 1: Capture a real fixture**

Copy ~80 representative lines of a real `ProcessPacket` decompile into
`testdata/process_packet_v83.c.txt`. Source: the decompile text can be
reconstructed from an existing IDA session export or written synthetically in
the exact Hex-Rays shape (`case 0x11u:` labels, `CLogin::OnFoo(...)` direct
calls, a `sub_5E1230(...)` unnamed call, a fallthrough pair `case 0x12u: case
0x13u:`). The fixture MUST include: hex and decimal labels, a `u` suffix, a
fallthrough pair, one unnamed `sub_` call, and a default arm.

- [ ] **Step 2: Write the failing parser test**

```go
package discover

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseDispatchSwitch(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "process_packet_v83.c.txt"))
	if err != nil {
		t.Fatal(err)
	}
	cases, err := ParseDispatch(string(raw))
	if err != nil {
		t.Fatal(err)
	}
	byOp := map[int]DispatchCase{}
	for _, c := range cases {
		byOp[c.Opcode] = c
	}
	if c := byOp[0x11]; c.Handler != "CLogin::OnFoo" {
		t.Errorf("0x11 -> %+v", c)
	}
	// fallthrough pair: both opcodes map to the same handler
	if byOp[0x12].Handler == "" || byOp[0x12].Handler != byOp[0x13].Handler {
		t.Errorf("fallthrough not shared: %+v / %+v", byOp[0x12], byOp[0x13])
	}
	// unnamed callee preserved as sub_ address-name, not dropped
	found := false
	for _, c := range cases {
		if c.Handler == "sub_5E1230" {
			found = true
		}
	}
	if !found {
		t.Error("sub_ handler dropped — discovery must keep unnamed handlers")
	}
}
```

- [ ] **Step 3: Implement `ParseDispatch`**

```go
// Package discover enumerates a version's operation universe from its IDA
// database (task-085 design §5.2): clientbound via the client packet
// dispatcher's switch, serverbound via send-op constant sites.
package discover

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type DispatchCase struct {
	Opcode  int
	Handler string // demangled name or sub_XXXX
}

var (
	caseRe = regexp.MustCompile(`^\s*case\s+(0x[0-9a-fA-F]+|\d+)u?\s*:`)
	callRe = regexp.MustCompile(`([A-Za-z_][A-Za-z0-9_:]*(?:::[~A-Za-z_][A-Za-z0-9_]*)?|sub_[0-9A-Fa-f]+)\s*\(`)
)

// ParseDispatch walks Hex-Rays text of the packet dispatcher and yields one
// DispatchCase per case label, binding pending (fallthrough) labels to the
// first call in the case body.
func ParseDispatch(text string) ([]DispatchCase, error) {
	var out []DispatchCase
	var pending []int
	for _, line := range strings.Split(text, "\n") {
		if m := caseRe.FindStringSubmatch(line); m != nil {
			op, err := parseIntLabel(m[1])
			if err != nil {
				return nil, fmt.Errorf("bad case label %q: %w", m[1], err)
			}
			pending = append(pending, op)
			// a call on the same line as the label binds immediately
		}
		if len(pending) == 0 {
			continue
		}
		if m := callRe.FindStringSubmatch(line); m != nil && !isNoise(m[1]) {
			for _, op := range pending {
				out = append(out, DispatchCase{Opcode: op, Handler: m[1]})
			}
			pending = nil
		}
		if strings.Contains(line, "break;") {
			pending = nil // empty case body: no handler discovered
		}
	}
	return out, nil
}

func parseIntLabel(s string) (int, error) {
	if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
		n, err := strconv.ParseInt(s[2:], 16, 32)
		return int(n), err
	}
	n, err := strconv.ParseInt(s, 10, 32)
	return int(n), err
}

// isNoise filters non-handler calls that appear inside dispatch arms.
func isNoise(name string) bool {
	switch name {
	case "memset", "memcpy", "operator", "if", "while", "switch":
		return true
	}
	return strings.HasPrefix(name, "CInPacket::")
}
```

- [ ] **Step 4: Run tests, commit**

```bash
go test ./internal/discover/ -v
cd ../..
git add tools/packet-audit/internal/discover/
git commit -m "task-085: discover-ops clientbound dispatch parser"
```

## Task 5.2: Serverbound send-site enumeration + reconciliation

**Files:**
- Modify: `tools/packet-audit/internal/discover/discover.go` (add `Reconcile`)
- Create: `tools/packet-audit/internal/discover/reconcile_test.go`

Serverbound discovery note: enumerating `COutPacket::COutPacket(this, op)`
construction sites needs cross-reference data the current `MCPClient`
interface does not expose. **Do not extend the interface speculatively.** For
this task, serverbound discovery = decompiling the FNames the registry already
lists for the version (the send functions) and confirming the `COutPacket`
constructor's op constant matches the registry opcode — a *verification* pass.
Full xref-based enumeration is deferred to the live-run task (5.4), where the
operator decides whether `xrefs_to`-based extension is worth adding (the
ida-pro-mcp server exposes `find_xref_signatures` / `xrefs_to`; extending
`MCPClient` + `MCPHTTPClient` then follows the exact pattern of
`GetCallees` at `mcphttp.go:405`).

Reconciliation (pure, design §5.2 + §13): discovery output vs a seeded
registry version file →
- discovered op not in registry → append `provenance: ida-discovered` + address,
- registry entry not found by discovery → flag for review (NEVER auto-delete),
- discovered opcode colliding with a different existing entry → review worklist.

- [ ] **Step 1: Write the failing reconciliation test (synthetic, per design §14)**

```go
package discover

import (
	"testing"

	"github.com/Chronicle20/atlas/tools/packet-audit/internal/opregistry"
)

func TestReconcile(t *testing.T) {
	seeded := opregistry.NewVersionFile([]opregistry.Entry{
		{Op: "LOGIN_STATUS", Direction: opregistry.DirClientbound, Opcode: 0x000, FName: "CLogin::OnCheckPasswordResult", Provenance: "csv-import"},
		{Op: "GHOST_OP", Direction: opregistry.DirClientbound, Opcode: 0x0FF, FName: "CFoo::OnGhost", Provenance: "csv-import"},
	})
	discovered := []Discovered{
		{Opcode: 0x000, Handler: "CLogin::OnCheckPasswordResult", Address: "0x5e1230"}, // match
		{Opcode: 0x002, Handler: "CLogin::OnAccountInfoResult", Address: "0x5e2000"},  // new
		{Opcode: 0x0FF, Handler: "CBar::OnSomethingElse", Address: "0x5e3000"},        // collision
	}
	res := Reconcile(seeded, discovered, opregistry.DirClientbound)

	if len(res.Append) != 1 || res.Append[0].FName != "CLogin::OnAccountInfoResult" ||
		res.Append[0].Provenance != "ida-discovered" || res.Append[0].IDA == nil {
		t.Errorf("append = %+v", res.Append)
	}
	if len(res.MissingAtDiscovery) != 0 {
		// GHOST_OP's opcode WAS discovered (collision), so it's a collision, not missing
		t.Errorf("missing = %+v", res.MissingAtDiscovery)
	}
	if len(res.Collisions) != 1 || res.Collisions[0].Entry.Op != "GHOST_OP" {
		t.Errorf("collisions = %+v", res.Collisions)
	}
}

func TestReconcileMissing(t *testing.T) {
	seeded := opregistry.NewVersionFile([]opregistry.Entry{
		{Op: "NEVER_FOUND", Direction: opregistry.DirClientbound, Opcode: 0x0AA, FName: "CFoo::OnNever", Provenance: "csv-import"},
	})
	res := Reconcile(seeded, nil, opregistry.DirClientbound)
	if len(res.MissingAtDiscovery) != 1 {
		t.Errorf("missing = %+v (registry entries discovery can't find are flagged, never deleted)", res.MissingAtDiscovery)
	}
}
```

- [ ] **Step 2: Implement `Reconcile`**

```go
type Discovered struct {
	Opcode  int
	Handler string
	Address string
}

type Collision struct {
	Entry      opregistry.Entry
	Discovered Discovered
}

type ReconcileResult struct {
	Append             []opregistry.Entry // new ops, provenance ida-discovered
	MissingAtDiscovery []opregistry.Entry // in registry, not found in IDB — review worklist
	Collisions         []Collision        // same opcode, different handler — review worklist
}

func Reconcile(vf *opregistry.VersionFile, discovered []Discovered, dir opregistry.Direction) ReconcileResult {
	var res ReconcileResult
	byOpcode := map[int]opregistry.Entry{}
	for _, e := range vf.Entries {
		if e.Direction == dir {
			byOpcode[e.Opcode] = e
		}
	}
	seenOpcode := map[int]bool{}
	for _, d := range discovered {
		seenOpcode[d.Opcode] = true
		if e, ok := byOpcode[d.Opcode]; ok {
			if e.FName != d.Handler && !hasAlt(e, d.Handler) && d.Handler != "" {
				res.Collisions = append(res.Collisions, Collision{Entry: e, Discovered: d})
			}
			continue
		}
		addr := parseAddr(d.Address)
		res.Append = append(res.Append, opregistry.Entry{
			Op:         opNameFor(d), // OP_<HANDLER_BASE> placeholder name; human renames via manual edit
			Direction:  dir,
			Opcode:     d.Opcode,
			FName:      d.Handler,
			Provenance: "ida-discovered",
			IDA:        &opregistry.IDARef{Address: addr},
		})
	}
	for _, e := range vf.Entries {
		if e.Direction == dir && !seenOpcode[e.Opcode] {
			res.MissingAtDiscovery = append(res.MissingAtDiscovery, e)
		}
	}
	return res
}
```

with the small helpers (`hasAlt` over `FNameAlts`; `parseAddr` hex-string →
uint64; `opNameFor` derives `IDA_<OPCODE-HEX>` e.g. `IDA_0X002` as the
placeholder op name — discovered ops get real canonical names only by human
edit, recorded with `provenance: manual`).

- [ ] **Step 3: Run tests, commit**

```bash
go test ./internal/discover/ -v && go test ./... && go vet ./...
cd ../..
git add tools/packet-audit/internal/discover/
git commit -m "task-085: discover-ops reconciliation (append/missing/collision)"
```

## Task 5.3: `discover-ops` subcommand

**Files:**
- Create: `tools/packet-audit/cmd/discover_ops.go`
- Create: `tools/packet-audit/cmd/discover_ops_test.go`
- Modify: `tools/packet-audit/cmd/root.go`

- [ ] **Step 1: Failing test with a fake MCP client**

Follow the `validateFakeMCP` pattern in `cmd/validate_test.go`: a fake
`idasrc.MCPClient` whose `GetFunctionByName("CClientSocket::ProcessPacket")`
returns a fixed address and whose `DecompileFunction` returns the Task 5.1
fixture text. Drive `runDiscoverOps` with `--version gms_v83 --registry-dir
<temp seeded dir> --apply=false` and assert the emitted worklist markdown
(written to `--out <tmp>/discover_gms_v83.md`) contains an `## Append`
section with the new op and a `## Review` section with collisions/missing.
With `--apply=true` assert the registry YAML gained the appended entries and
`opregistry.LoadVersion` still validates it.

- [ ] **Step 2: Implement `runDiscoverOps`**

Flag set: `--version` (required), `--registry-dir` (default
`docs/packets/registry`), `--dispatcher` (default
`CClientSocket::ProcessPacket`), `--ida-url` (default
`http://127.0.0.1:13337/mcp`), `--ida-port` (instance select, optional),
`--out` (worklist path, default `docs/packets/registry/discover_<version>.md`),
`--apply` (bool, default false — append-only registry mutation).

Core flow: resolve dispatcher FName → decompile → `discover.ParseDispatch` →
resolve each handler's demangled name (reuse `idasrc` demangle helpers where
the callee is `sub_`-named, via `GetCallees` on the dispatcher) →
`discover.Reconcile` → write worklist markdown; on `--apply`, rewrite
`<registry-dir>/<version>.yaml` with appended entries (stable sort as in
`registry seed`) and refuse to apply when `Collisions` or schema errors exist
(design §13: never auto-resolved).

Wire dispatch in root.go:

```go
	if len(args) > 0 && args[0] == "discover-ops" {
		return runDiscoverOps(args[1:], stderr)
	}
```

- [ ] **Step 3: Tests + commit**

```bash
go test ./cmd/ -run TestDiscoverOps -v && go test ./... && go vet ./...
cd ../..
git add tools/packet-audit/cmd/
git commit -m "task-085: discover-ops subcommand (worklist + apply modes)"
```

## Task 5.4: OPERATOR-GATED — run discovery against the five IDBs

**Files:**
- Modify: `docs/packets/registry/*.yaml` (reconciled)
- Create: `docs/packets/registry/discover_<version>.md` ×5 (review worklists)

> **CHECKPOINT: requires live ida-pro-mcp.** Pause and ask the user to start
> the IDA instances (multi-instance is supported; one IDB per instance — v83,
> v84, v87, v95, jms185). Resolve each instance by its loaded IDB
> (`list_instances`), never by hardcoded port.

- [ ] **Step 1: Per version (5×), run discovery in worklist mode**

```bash
go run ./tools/packet-audit discover-ops --version gms_v83 --ida-port <port-of-v83-instance>
```
Review `docs/packets/registry/discover_gms_v83.md`. Adjudicate each Review
item against the IDB (CSV transcription error vs discovery blind spot);
record adjudications as `provenance: manual` edits with an IDA citation in
`note` (design §5.2).

- [ ] **Step 2: Apply appends**

```bash
go run ./tools/packet-audit discover-ops --version gms_v83 --ida-port <port> --apply
```
Repeat per version. v84 is the important pass: it converts the v83-copied
seed into IDA-evidenced truth (expect near-zero delta per task-083's
v84≡v83 finding — a large delta is itself a finding to investigate).

- [ ] **Step 3: Regenerate matrix; expect conflict noise to DROP**

```bash
go run ./tools/packet-audit matrix
git diff --stat docs/packets/audits/STATUS.md
grep -c '🟥' docs/packets/audits/STATUS.md
```
Compare against the pre-discovery conflict count (Task 1.8 step 5). Remaining
conflicts are genuine findings — list them in the commit message; do NOT
resolve them in this task (each follows design §10.1's remediation path in
follow-up work, EXCEPT pure registry-leg fixes which are in-scope manual edits
here).

- [ ] **Step 4: Commit (one commit per version, then the matrix regen)**

```bash
git add docs/packets/registry/ docs/packets/audits/STATUS.md docs/packets/audits/status.json
git commit -m "task-085: registry reconciled against the five IDBs via discover-ops"
git branch --show-current
```

## Task 5.5: OPERATOR-GATED — v84 export harvest + first audit pass

**Files:**
- Create: `docs/packets/ida-exports/gms_v84.json`
- Create: `docs/packets/audits/gms_v84/` (reports)
- Modify (regenerated): STATUS.md, status.json

> **CHECKPOINT: requires the v84 IDB live.** This brings the v84 column to
> parity (design §12 phase 5). The exact `export`/`validate` invocations are
> documented in `docs/packets/audits/STARTING_A_NEW_VERSION_PASS.md`; use them
> with `--version gms_v84`. Watch the known degraded-input failure modes from
> task-081 (memory: alias sets, unnamed sub_XXXX descent, switch-expr) — if
> the exporter chokes on the v84 Hex-Rays output, record the failure and stop;
> exporter hardening is NOT in this task's scope.

- [ ] **Step 1: Harvest**

```bash
go run ./tools/packet-audit export --version gms_v84 --output docs/packets/ida-exports/gms_v84.json [--ida-port <port>]
```

- [ ] **Step 2: First audit pass**

```bash
go run ./tools/packet-audit validate --version gms_v84 --output docs/packets/audits
```
(Exact flags per STARTING_A_NEW_VERSION_PASS.md — read it first; the audit-dir
default quirks are documented there and in the tool's `--help`.)

- [ ] **Step 3: Regenerate + verify the v84 column populated**

```bash
go run ./tools/packet-audit matrix
grep -A8 '## Totals' docs/packets/audits/STATUS.md
```
Expect v84 totals to move from all-❌ toward the v83 distribution (task-083
predicts near-mirror). Material v83/v84 divergence = finding, list it.

- [ ] **Step 4: Commit**

```bash
git add docs/packets/ida-exports/gms_v84.json docs/packets/audits/gms_v84/ docs/packets/audits/STATUS.md docs/packets/audits/status.json
git commit -m "task-085: gms_v84 export harvest + first audit pass — v84 column at parity"
```

---

# Phase 6 — Process wiring (design phase 7)

## Task 6.1: CI job — `matrix --check`

**Files:**
- Create: `.github/workflows/packet-matrix.yml`

Modeled on `.github/workflows/catalog-lint.yml` (path-filtered, single job).
Trigger paths per design §10 rule 2.

- [ ] **Step 1: Write the workflow**

```yaml
name: packet-matrix

on:
  pull_request:
    branches: [main]
    paths:
      - 'tools/packet-audit/**'
      - 'libs/atlas-packet/**'
      - 'docs/packets/**'
      - 'services/atlas-configurations/seed-data/templates/**'

env:
  GO_VERSION: '1.25.5'

jobs:
  check:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: ${{ env.GO_VERSION }}

      - name: Packet-audit tests
        run: cd tools/packet-audit && go test ./...

      - name: Coverage matrix check
        run: |
          set -e
          go run ./tools/packet-audit matrix --check
```

- [ ] **Step 2: Verify locally that the gate passes on this branch**

```bash
go run ./tools/packet-audit matrix --check; echo "exit=$?"
```
**Must exit 0 before this lands.** If conflicts survived Task 5.4, the gate
would be born red — in that case keep the workflow but add an explicit
`continue-on-error: true` on the matrix step **with a tracking note in the
workflow file** naming the follow-up task that burns down the conflicts, and
say so in the PR description. (Design §10.1 wants conflicts blocking; a gate
that is red on day one just gets ignored — make the blocking flip a one-line
follow-up once conflicts are zero.)

- [ ] **Step 3: Commit**

```bash
git add .github/workflows/packet-matrix.yml
git commit -m "task-085: CI packet-matrix --check gate"
```

## Task 6.2: Rewrite `STARTING_A_NEW_VERSION_PASS.md` + task-close gate

**Files:**
- Modify: `docs/packets/audits/STARTING_A_NEW_VERSION_PASS.md` (full rewrite)

- [ ] **Step 1: Rewrite as a thin orchestration doc** (design §11.1 end):
structure it as:

1. **Set up the column**: registry file via `discover-ops` (exact invocation
   incl. `--ida-port` + instance-by-IDB rule), tenant template, IDA export via
   `export` (exact invocation incl. the task-081 subcommand flags the old doc
   omitted — `export`, `validate`, `decompose`, `triage`, `resolve-dispatch`
   with their real flag sets, copy from `--help` output), audit pass via
   `validate`.
2. **Regenerate the matrix**: `matrix` + `--check`; the new column appears
   pre-filled from applicability (⬜/❌).
3. **Promote cells**: apply `docs/packets/audits/VERIFYING_A_PACKET.md` per
   cell, hottest tier first; fan out with the `packet-verifier` agent for
   campaigns.
4. **Task-close gate** (design §10 rule 1, verbatim policy): an audit/version
   task is done when every cell in its declared scope is ✅, 🟡-with-evidence,
   or ⬜ — the scope declaration is a list of matrix cells in the task PRD.
   No prose acceptance. Cell regressions in a PR fail `matrix --check` unless
   the regenerated STATUS.md is committed and the PR description owns them.
5. **Degradation remediation paths**: copy design §10.1 (conflict legs:
   registry / template+live-config+restart / Atlas code) and §10.2 (hash
   drift, broken linkage, verdict flip) so the doc is self-contained. Include
   the live-tenant warning: template fixes do nothing for existing tenants
   without a config patch + channel restart.

- [ ] **Step 2: Commit**

```bash
git add docs/packets/audits/STARTING_A_NEW_VERSION_PASS.md
git commit -m "task-085: STARTING_A_NEW_VERSION_PASS rewritten around the matrix workflow"
```

---

# Final verification + handoff

## Task 7.1: Full verification sweep

- [ ] **Step 1: Module checks (the only changed Go module is the tool)**

```bash
cd tools/packet-audit && go test -race ./... && go vet ./... && go build ./... && cd ../..
cd libs/atlas-packet && go test -race ./... && go vet ./... && cd ../..
tools/redis-key-guard.sh
```
Expected: all clean. (`libs/atlas-packet` changed only by comment markers +
new tests in 3.3/4.4 — `go.mod` untouched, so no `docker buildx bake`
requirement triggers; run `docker buildx bake atlas-login` once anyway as a
canary if any non-test file in libs/atlas-packet was touched by a wire fix.)

- [ ] **Step 2: The four generated-artifact invariants**

```bash
go run ./tools/packet-audit matrix --check && echo CHECK-OK
go run ./tools/packet-audit registry seed --out /tmp/reseed && diff -q /tmp/reseed/gms_v83.yaml <(git show HEAD:docs/packets/registry/gms_v83.yaml) || echo "expected: differs only by post-seed discovery/manual entries — eyeball the diff"
```

- [ ] **Step 3: Update the task retrospective pointer**

Append one line to `docs/tasks/task-085-packet-audit-coverage-matrix/retrospective.md`:
"Implementation landed via plan.md in this folder; STATUS.md is the artifact of record."

- [ ] **Step 4: Code review + PR**

Invoke `superpowers:requesting-code-review` (plan-adherence-reviewer +
backend-guidelines-reviewer — Go changed). Address findings. Then
`superpowers:finishing-a-development-branch`. PR body must include: the
honesty-shock expectation (first STATUS.md is mostly 🟡/❌ — design §15
risk 1), the conflict count trajectory (post-seed vs post-discovery), and
which cells the e2e validation promoted.

---

## Task-ordering / dependency notes for the dispatcher

- Strict order within Phase 1 (1.1 → 1.8); Phase 2 needs 1.8; Phase 3 needs
  2.3 + 2.4; Task 3.3 needs 3.2; Phase 4 needs 3.3 (retrofitted examples are
  the playbook's reference pattern); Tasks 5.1–5.3 are independent of Phases
  2–4 (only need 1.4) and may interleave; 5.4–5.5 need 5.3 + a human; Phase 6
  needs everything except 5.5 (the CI gate must not require v84 reports —
  `LoadReports` tolerates the missing dir, and after 5.5 it simply has data).
- If live IDA never becomes available during execution: land 5.4/5.5 as a
  follow-up task, note it in the PR, and in Task 6.1 expect a non-zero
  conflict count → use the `continue-on-error` escape hatch with its tracking
  note. Everything else in this plan is still landable and useful (design §12:
  each phase lands independently).
