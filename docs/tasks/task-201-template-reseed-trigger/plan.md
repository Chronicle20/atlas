# Template Re-seed Trigger Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make template drift from the seed file baked into the running image visible as a read-only attribute, and give operators an explicit one-template "reset to shipped defaults" button.

**Architecture:** One canonical `Revision(RestModel) (string, error)` SHA-256 function serves both the shipped side and the stored side, so they cannot drift apart. A `Catalog` loaded once from `$SEED_DATA_PATH/templates` — living in package `templates`, not `seeder`, to avoid an import cycle — becomes the single parse path for shipped files, and the boot seeder reads from it. The three computed attributes ride on a `ViewRestModel` that embeds `RestModel`, so no write path can ever persist them. Re-seed rewrites `Entity.Data` on the existing row using `Create`'s exact validation-and-marshal semantics (never `UpdateById`, whose preset validator would perturb the bytes).

**Tech Stack:** Go 1.x (`crypto/sha256`, `encoding/hex`, `encoding/json` — all stdlib, no new module dependency), GORM, gorilla/mux, api2go JSON:API, logrus. Frontend: React 19 + TypeScript, TanStack React Query, shadcn/ui (`Badge`, `AlertDialog`, `Tooltip`), vitest + @testing-library/react.

## Global Constraints

- **Worktree.** All work happens in `.worktrees/task-201-template-reseed-trigger` on branch `task-201-template-reseed-trigger`. Never edit the main repo.
- **No schema change, no migration.** The `templates` table is untouched.
- **No new Go module dependency.** `go.mod` must not change, so `docker buildx bake atlas-configurations` is not required.
- **Boot behaviour is frozen.** `importTemplate` stays create-if-absent; no startup code path may modify an existing template row (FR-4.1, FR-4.2).
- **No Kafka / outbox participation.** Re-seed emits nothing (FR-3.8).
- **Badge wording is fixed at "Differs from image"** — used verbatim in the badge label, the badge tooltip ("Differs from the configuration shipped in this image"), and the confirm dialog. Badge variant is `secondary`, never `destructive` (NFR-4: the flag is advisory, not an error).
- **Endpoint is not tenant-scoped.** Templates are global; no tenant headers (NFR-5).
- **`exactOptionalPropertyTypes` is on** in atlas-ui. The three new attributes are optional (`?:`); read sites must handle `undefined`, not assume `""` / `false`.
- **Verification before PR:** `go test -race ./...`, `go vet ./...`, `go build ./...` clean in `services/atlas-configurations`; `tools/lint.sh --check` clean from repo root; `npm run build` and vitest clean in `services/atlas-ui`.
- Paths below are relative to the worktree root unless stated. The Go service tree root is `services/atlas-configurations/atlas.com/configurations/`; run `go test` from there.

---

## File Structure

**Go — `services/atlas-configurations/atlas.com/configurations/`**

| File | Responsibility |
|---|---|
| `templates/revision.go` *(new)* | The one canonical revision function. Nothing else. |
| `templates/revision_test.go` *(new)* | Id-invariance and determinism of `Revision`. |
| `templates/shipped.go` *(new)* | `CatalogEntry`, `Catalog`, `LoadCatalog` (pure), `InitShippedCatalog` / `ShippedCatalog` (singleton). |
| `templates/shipped_test.go` *(new)* | Parse-failure tolerance, duplicate-key resolution, real-corpus coverage. |
| `templates/testdata/templates/*` *(new)* | Fixtures for the catalog's failure paths. |
| `templates/rest.go` | Gains `ViewRestModel` only. `RestModel` untouched. |
| `templates/processor.go` | `catalog` field + `WithCatalog`; `canonicalBytes` extracted from `Create`; `makeView` + three view providers; `ReseedById`; sentinel errors. |
| `templates/processor_test.go` | Drift, canonical-bytes and re-seed tests appended to the existing sqlite harness. |
| `templates/resource.go` | Read handlers marshal `ViewRestModel`; new `POST /{templateId}/reseed`; `writeJSONAPIError`. |
| `templates/resource_reseed_test.go` *(new)* | 204 / 404 / 409 and the JSON:API error-document shape. |
| `templates/mock/processor.go` | The five new interface members. |
| `seeder/seeder.go` | `ConfigMetadata` / `extractMetadata` / `discoverFiles` deleted; `Seeder` holds a `templates.Catalog`; `importTemplate` takes a `templates.CatalogEntry`. |
| `seeder/seeder_test.go` | Discovery/metadata cases migrate to `templates`; the skip-existing regression guard is added. |
| `main.go` | `templates.InitShippedCatalog` before `s.Run()`, unconditional on `SEED_ENABLED`. |

**TypeScript — `services/atlas-ui/src/`**

| File | Responsibility |
|---|---|
| `types/models/template.ts` | Three optional attributes on `TemplateAttributes`. |
| `services/api/templates.service.ts` | `reseed(id, options?)`. |
| `lib/hooks/api/useTemplates.ts` | `useReseedTemplate()` with invalidation. |
| `components/features/templates/TemplateReseedButton.tsx` *(new)* | Button + confirm dialog. |
| `components/features/templates/TemplateDetailLayout.tsx` | Mounts the button. |
| `pages/templates-columns.tsx` | The drift badge column. |
| `lib/utils/config-export.ts` | Strips the three computed keys from the export payload. |

**Docs**

| File | Responsibility |
|---|---|
| `docs/packets/TEMPLATE_CONVENTIONS.md` | One line recording the `presets-carry-ids` condition (design §7). |

---

## Task 1: The canonical revision function

**Files:**
- Create: `services/atlas-configurations/atlas.com/configurations/templates/revision.go`
- Test: `services/atlas-configurations/atlas.com/configurations/templates/revision_test.go`

**Interfaces:**
- Consumes: `templates.RestModel` (`templates/rest.go:11`), `socket.Normalize` (`templates/socket/rest.go:30`).
- Produces: `func Revision(rm RestModel) (string, error)` — lowercase hex SHA-256 of `json.Marshal` applied to `rm` after clearing `Id` and applying `socket.Normalize`. Every later task uses exactly this name and signature.

- [x] **Step 1: Write the failing test**

Create `templates/revision_test.go`:

```go
package templates

import (
	"testing"

	"atlas-configurations/templates/socket"
	"atlas-configurations/templates/socket/handler"
)

// FR-2.2 / design §5: RestModel.Id carries json:"-", but Revision clears it
// explicitly rather than relying on the tag. This test is what keeps that
// stronger guarantee meaningful instead of tautological.
func TestRevisionIgnoresId(t *testing.T) {
	a := createTestRestModel("GMS", 83, 1)
	b := createTestRestModel("GMS", 83, 1)
	b.Id = "11111111-1111-1111-1111-111111111111"

	ra, err := Revision(a)
	if err != nil {
		t.Fatalf("Revision(a): %v", err)
	}
	rb, err := Revision(b)
	if err != nil {
		t.Fatalf("Revision(b): %v", err)
	}
	if ra != rb {
		t.Errorf("revision changed with Id set: %q != %q", ra, rb)
	}
}

// The revision must be blind to the nil-vs-empty distinction Normalize erases,
// because Make normalizes on read and Create normalizes on write. A revision
// that saw the difference would report drift on every template whose stored
// document omits a socket collection.
func TestRevisionNormalizesSocket(t *testing.T) {
	withNil := createTestRestModel("GMS", 83, 1)
	withNil.Socket = socket.RestModel{}

	withEmpty := createTestRestModel("GMS", 83, 1)
	withEmpty.Socket = socket.RestModel{
		Handlers: []handler.RestModel{},
	}

	rn, err := Revision(withNil)
	if err != nil {
		t.Fatalf("Revision(withNil): %v", err)
	}
	re, err := Revision(withEmpty)
	if err != nil {
		t.Fatalf("Revision(withEmpty): %v", err)
	}
	if rn != re {
		t.Errorf("nil and empty socket collections produced different revisions: %q != %q", rn, re)
	}
}

// A revision is a lowercase hex SHA-256: 64 characters, stable across calls.
func TestRevisionIsStableLowercaseHex(t *testing.T) {
	m := createTestRestModel("GMS", 83, 1)
	first, err := Revision(m)
	if err != nil {
		t.Fatalf("Revision: %v", err)
	}
	second, err := Revision(m)
	if err != nil {
		t.Fatalf("Revision (second call): %v", err)
	}
	if first != second {
		t.Errorf("Revision is not deterministic: %q != %q", first, second)
	}
	if len(first) != 64 {
		t.Errorf("revision length = %d, want 64: %q", len(first), first)
	}
	for _, c := range first {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			t.Fatalf("revision is not lowercase hex: %q", first)
		}
	}
}

// Two templates that differ in content must not collide.
func TestRevisionDiffersOnContentChange(t *testing.T) {
	a := createTestRestModel("GMS", 83, 1)
	b := createTestRestModel("GMS", 83, 1)
	b.UsesPin = !a.UsesPin

	ra, err := Revision(a)
	if err != nil {
		t.Fatalf("Revision(a): %v", err)
	}
	rb, err := Revision(b)
	if err != nil {
		t.Fatalf("Revision(b): %v", err)
	}
	if ra == rb {
		t.Errorf("differing models produced the same revision: %q", ra)
	}
}
```

`createTestRestModel` already exists at `templates/processor_test.go:52` — do not redefine it.

- [x] **Step 2: Run test to verify it fails**

```bash
cd services/atlas-configurations/atlas.com/configurations
go test ./templates/ -run 'TestRevision' -v
```

Expected: FAIL — `undefined: Revision`.

- [x] **Step 3: Write the implementation**

Create `templates/revision.go`:

```go
package templates

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"atlas-configurations/templates/socket"
)

// Revision is the ONE definition of a template's content hash. Both sides of
// the drift comparison call it: LoadCatalog hashes the parsed seed file
// (FR-1.4), and makeView hashes the RestModel that Make produced from the
// stored row (FR-2.1). Wording them as two definitions that happen to agree
// is how they eventually stop agreeing, so there is only one.
//
// Id is cleared rather than trusted to its json:"-" tag - strictly stronger
// than FR-2.2, and it costs one assignment. Socket is normalized because both
// Make (read) and Create (write) normalize, so a revision that saw the
// nil-vs-empty distinction would report drift on every template whose stored
// document omits a socket collection.
func Revision(rm RestModel) (string, error) {
	rm.Id = ""
	rm.Socket = socket.Normalize(rm.Socket)

	b, err := json.Marshal(rm)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}
```

- [x] **Step 4: Run tests to verify they pass**

```bash
cd services/atlas-configurations/atlas.com/configurations
go test ./templates/ -run 'TestRevision' -v
```

Expected: PASS — all four tests.

- [x] **Step 5: Commit**

```bash
git add services/atlas-configurations/atlas.com/configurations/templates/revision.go \
        services/atlas-configurations/atlas.com/configurations/templates/revision_test.go
git commit -m "feat(atlas-configurations): add canonical template Revision function"
```

---

## Task 2: The shipped-template catalog

**Files:**
- Create: `services/atlas-configurations/atlas.com/configurations/templates/shipped.go`
- Create: `services/atlas-configurations/atlas.com/configurations/templates/shipped_test.go`
- Create: `services/atlas-configurations/atlas.com/configurations/templates/testdata/templates/valid_template.json`
- Create: `services/atlas-configurations/atlas.com/configurations/templates/testdata/templates/invalid_json.json`
- Create: `services/atlas-configurations/atlas.com/configurations/templates/testdata/templates/missing_region.json`
- Create: `services/atlas-configurations/atlas.com/configurations/templates/testdata/templates/not_json.txt`
- Create: `services/atlas-configurations/atlas.com/configurations/templates/testdata/duplicates/a_first.json`
- Create: `services/atlas-configurations/atlas.com/configurations/templates/testdata/duplicates/b_second.json`

**Interfaces:**
- Consumes: `Revision` from Task 1; `RestModel`.
- Produces:
  - `type CatalogEntry struct { FileName string; Model RestModel; Revision string }`
  - `type Catalog struct { … }` with methods `Lookup(region string, majorVersion uint16, minorVersion uint16) (CatalogEntry, bool)`, `Entries() []CatalogEntry` (file-name sort order), `Len() int`. The zero `Catalog` is usable and empty.
  - `func LoadCatalog(l logrus.FieldLogger, dir string) Catalog` — pure, no globals.
  - `func InitShippedCatalog(l logrus.FieldLogger, dir string) Catalog` and `func ShippedCatalog() Catalog` — the `sync.Once` + `sync.RWMutex` singleton (FR-1.2).

- [x] **Step 1: Create the test fixtures**

`templates/testdata/templates/valid_template.json`:

```json
{
  "region": "TEST",
  "majorVersion": 1,
  "minorVersion": 0,
  "usesPin": false,
  "socket": {
    "handlers": [],
    "writers": []
  },
  "characters": {
    "templates": []
  },
  "npcs": [],
  "worlds": []
}
```

`templates/testdata/templates/invalid_json.json`:

```
{ this is not valid json
```

`templates/testdata/templates/missing_region.json`:

```json
{
  "majorVersion": 1,
  "minorVersion": 0
}
```

`templates/testdata/templates/not_json.txt`:

```
this file is not a template and must be ignored by extension
```

`templates/testdata/duplicates/a_first.json`:

```json
{
  "region": "DUP",
  "majorVersion": 7,
  "minorVersion": 3,
  "usesPin": false,
  "socket": { "handlers": [], "writers": [] },
  "characters": { "templates": [] },
  "npcs": [],
  "worlds": []
}
```

`templates/testdata/duplicates/b_second.json`:

```json
{
  "region": "DUP",
  "majorVersion": 7,
  "minorVersion": 3,
  "usesPin": true,
  "socket": { "handlers": [], "writers": [] },
  "characters": { "templates": [] },
  "npcs": [],
  "worlds": []
}
```

`usesPin` differs so the duplicate test can prove *which* file won, not merely that one did.

- [x] **Step 2: Write the failing test**

Create `templates/shipped_test.go`:

```go
package templates

import (
	"path/filepath"
	"testing"
)

// FR-1.5 / FR-1.6: a directory containing an unparseable file, a file missing
// the required region, and a non-.json file must still yield the one good
// entry. A bad file is logged and omitted; it never fails the load.
func TestLoadCatalogTolerantOfBadFiles(t *testing.T) {
	c := LoadCatalog(testLogger(), filepath.Join("testdata", "templates"))

	if c.Len() != 1 {
		t.Fatalf("Len() = %d, want 1 (entries: %v)", c.Len(), c.Entries())
	}
	e, ok := c.Lookup("TEST", 1, 0)
	if !ok {
		t.Fatalf("Lookup(TEST,1,0) = miss, want hit")
	}
	if e.FileName != "valid_template.json" {
		t.Errorf("FileName = %q, want valid_template.json", e.FileName)
	}
	if e.Model.Region != "TEST" {
		t.Errorf("Model.Region = %q, want TEST", e.Model.Region)
	}
	want, err := Revision(e.Model)
	if err != nil {
		t.Fatalf("Revision: %v", err)
	}
	if e.Revision != want {
		t.Errorf("Revision = %q, want %q", e.Revision, want)
	}
}

// FR-1.6: two files resolving to the same key - the first in file-name sort
// order wins. a_first.json sorts before b_second.json and carries
// usesPin:false, so the winner is identifiable.
func TestLoadCatalogDuplicateKeyFirstWins(t *testing.T) {
	c := LoadCatalog(testLogger(), filepath.Join("testdata", "duplicates"))

	if c.Len() != 1 {
		t.Fatalf("Len() = %d, want 1", c.Len())
	}
	e, ok := c.Lookup("DUP", 7, 3)
	if !ok {
		t.Fatalf("Lookup(DUP,7,3) = miss, want hit")
	}
	if e.FileName != "a_first.json" {
		t.Errorf("FileName = %q, want a_first.json (sort-order first)", e.FileName)
	}
	if e.Model.UsesPin {
		t.Errorf("UsesPin = true, want false - b_second.json overwrote a_first.json")
	}
}

// A missing directory is not an error: an empty catalog reports "no shipped
// file" for every template (FR-2.4), which is the safe degradation.
func TestLoadCatalogMissingDirectoryIsEmpty(t *testing.T) {
	c := LoadCatalog(testLogger(), filepath.Join("testdata", "does-not-exist"))
	if c.Len() != 0 {
		t.Errorf("Len() = %d, want 0", c.Len())
	}
	if _, ok := c.Lookup("TEST", 1, 0); ok {
		t.Errorf("Lookup on empty catalog = hit, want miss")
	}
}

// The zero Catalog must be usable - ProcessorImpl without WithCatalog holds
// one, and it must degrade to "no shipped file" rather than panic.
func TestZeroCatalogIsUsable(t *testing.T) {
	var c Catalog
	if c.Len() != 0 {
		t.Errorf("Len() = %d, want 0", c.Len())
	}
	if _, ok := c.Lookup("GMS", 83, 1); ok {
		t.Errorf("Lookup on zero catalog = hit, want miss")
	}
	if got := len(c.Entries()); got != 0 {
		t.Errorf("len(Entries()) = %d, want 0", got)
	}
}

// The real corpus: all eleven shipped seed files load, and the two versions
// that have historically been confused (GMS 83.1 and GMS 84.1) resolve to
// distinct keys. Migrated from seeder_test.go's TestExtractMetadataGmsV84 /
// TestGmsV84DistinctFromV83 / TestSeedDataDiscoversBothV83AndV84, which tested
// the now-deleted extractMetadata/discoverFiles path.
func TestLoadCatalogRealSeedCorpus(t *testing.T) {
	c := LoadCatalog(testLogger(), seedTemplatesDir())

	if c.Len() != 11 {
		t.Fatalf("Len() = %d, want 11 - a seed file was added or removed without updating this test", c.Len())
	}

	e83, ok := c.Lookup("GMS", 83, 1)
	if !ok {
		t.Fatalf("Lookup(GMS,83,1) = miss")
	}
	if e83.FileName != "template_gms_83_1.json" {
		t.Errorf("GMS 83.1 FileName = %q, want template_gms_83_1.json", e83.FileName)
	}

	e84, ok := c.Lookup("GMS", 84, 1)
	if !ok {
		t.Fatalf("Lookup(GMS,84,1) = miss")
	}
	if e84.FileName != "template_gms_84_1.json" {
		t.Errorf("GMS 84.1 FileName = %q, want template_gms_84_1.json", e84.FileName)
	}

	if e83.Revision == e84.Revision {
		t.Errorf("GMS 83.1 and GMS 84.1 share a revision - they are not distinct documents")
	}

	// Entries() must be deterministic and complete.
	entries := c.Entries()
	if len(entries) != c.Len() {
		t.Fatalf("len(Entries()) = %d, Len() = %d", len(entries), c.Len())
	}
	for i := 1; i < len(entries); i++ {
		if entries[i-1].FileName > entries[i].FileName {
			t.Errorf("Entries() not sorted by file name: %q before %q", entries[i-1].FileName, entries[i].FileName)
		}
	}
}

// The singleton wrapper initializes once and thereafter serves the same
// catalog (FR-1.2). It is exercised here rather than mocked because main.go
// is its only other caller.
func TestInitShippedCatalogIsIdempotent(t *testing.T) {
	first := InitShippedCatalog(testLogger(), seedTemplatesDir())
	if first.Len() != 11 {
		t.Fatalf("first Len() = %d, want 11", first.Len())
	}
	// A second call with a bogus directory must NOT replace the loaded catalog.
	second := InitShippedCatalog(testLogger(), filepath.Join("testdata", "does-not-exist"))
	if second.Len() != first.Len() {
		t.Errorf("second InitShippedCatalog changed the catalog: Len() = %d, want %d", second.Len(), first.Len())
	}
	if ShippedCatalog().Len() != first.Len() {
		t.Errorf("ShippedCatalog().Len() = %d, want %d", ShippedCatalog().Len(), first.Len())
	}
}
```

Add the shared path helper to `templates/shipped_test.go` as well:

```go
// seedTemplatesDir is the checked-in seed corpus, relative to this package
// directory (where `go test` runs).
func seedTemplatesDir() string {
	return filepath.Join("..", "..", "..", "seed-data", "templates")
}
```

`testLogger()` already exists at `templates/processor_test.go:46` — do not redefine it.

- [x] **Step 3: Run test to verify it fails**

```bash
cd services/atlas-configurations/atlas.com/configurations
go test ./templates/ -run 'TestLoadCatalog|TestZeroCatalog|TestInitShippedCatalog' -v
```

Expected: FAIL — `undefined: LoadCatalog`, `undefined: Catalog`.

- [x] **Step 4: Write the implementation**

Create `templates/shipped.go`:

```go
package templates

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"github.com/sirupsen/logrus"
)

// CatalogEntry is one seed file as it ships in the running image.
type CatalogEntry struct {
	// FileName is the base name, e.g. "template_gms_83_1.json". Carried so
	// the NFR-3 re-seed log names the source.
	FileName string
	// Model is the parsed document, NOT normalized - Revision and
	// canonicalBytes both normalize, so storing a normalized copy would put a
	// second, silently-diverging normalization point in the tree.
	Model RestModel
	// Revision is Revision(Model), precomputed once at load.
	Revision string
}

type catalogKey struct {
	region string
	major  uint16
	minor  uint16
}

// Catalog is the set of templates baked into this image, keyed by
// (region, majorVersion, minorVersion). The zero value is a usable empty
// catalog: every Lookup misses, which is the FR-2.4 "no shipped file"
// behaviour. An un-wired processor therefore degrades safely.
type Catalog struct {
	byKey   map[catalogKey]CatalogEntry
	ordered []CatalogEntry
}

// Lookup returns the shipped entry for a region/version, if one ships.
func (c Catalog) Lookup(region string, majorVersion uint16, minorVersion uint16) (CatalogEntry, bool) {
	e, ok := c.byKey[catalogKey{region: region, major: majorVersion, minor: minorVersion}]
	return e, ok
}

// Entries returns every entry in file-name sort order. The boot seeder
// iterates this, so the order is the seeder's import order and must stay
// deterministic.
func (c Catalog) Entries() []CatalogEntry {
	out := make([]CatalogEntry, len(c.ordered))
	copy(out, c.ordered)
	return out
}

// Len is the number of entries.
func (c Catalog) Len() int {
	return len(c.ordered)
}

// LoadCatalog reads every *.json file under dir into a catalog. It is pure -
// no globals, no environment - so tests use it directly against a fixture
// directory. It never returns an error: a missing directory yields an empty
// catalog, and a file that fails to parse is logged at ERROR and omitted
// (FR-1.5) rather than failing the load or blocking startup.
func LoadCatalog(l logrus.FieldLogger, dir string) Catalog {
	c := Catalog{byKey: make(map[catalogKey]CatalogEntry)}

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			l.WithField("directory", dir).Debug("Shipped template directory does not exist")
			return c
		}
		l.WithError(err).WithField("directory", dir).Error("Unable to read shipped template directory")
		return c
	}

	var names []string
	for _, de := range entries {
		if de.IsDir() {
			continue
		}
		if filepath.Ext(de.Name()) != ".json" {
			continue
		}
		names = append(names, de.Name())
	}
	// Deterministic ordering: this is what makes FR-1.6's "first wins" and the
	// seeder's import order reproducible.
	sort.Strings(names)

	for _, name := range names {
		ll := l.WithField("file", name)

		b, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			ll.WithError(err).Error("Unable to read shipped template file")
			continue
		}

		var rm RestModel
		if err := json.Unmarshal(b, &rm); err != nil {
			ll.WithError(err).Error("Unable to parse shipped template file")
			continue
		}
		if rm.Region == "" {
			ll.Error("Shipped template file is missing required field: region")
			continue
		}

		rev, err := Revision(rm)
		if err != nil {
			ll.WithError(err).Error("Unable to compute revision for shipped template file")
			continue
		}

		k := catalogKey{region: rm.Region, major: rm.MajorVersion, minor: rm.MinorVersion}
		if existing, dup := c.byKey[k]; dup {
			ll.WithFields(logrus.Fields{
				"region":       rm.Region,
				"majorVersion": rm.MajorVersion,
				"minorVersion": rm.MinorVersion,
				"keeping":      existing.FileName,
			}).Error("Duplicate shipped template key; keeping the first file in sort order")
			continue
		}

		e := CatalogEntry{FileName: name, Model: rm, Revision: rev}
		c.byKey[k] = e
		c.ordered = append(c.ordered, e)
	}

	return c
}

var (
	shippedOnce sync.Once
	shippedMu   sync.RWMutex
	shipped     Catalog
)

// InitShippedCatalog loads the singleton catalog from dir exactly once and
// returns it (FR-1.2). Called from main.go before the seeder runs and before
// routes are registered.
//
// It is deliberately NOT gated on SEED_ENABLED: that flag governs whether
// templates are IMPORTED, not whether the service knows what ships. An
// operator who has disabled seeding still needs the drift badge and the reset
// button.
func InitShippedCatalog(l logrus.FieldLogger, dir string) Catalog {
	shippedOnce.Do(func() {
		c := LoadCatalog(l, dir)

		shippedMu.Lock()
		shipped = c
		shippedMu.Unlock()

		ll := l.WithFields(logrus.Fields{"directory": dir, "count": c.Len()})
		if c.Len() == 0 {
			// Loud, because the symptom is silent: every template reports "no
			// shipped file", the badge never lights up, and re-seed 409s.
			ll.Warn("Shipped template catalog is empty; drift detection and re-seed are inert")
		} else {
			ll.Info("Shipped template catalog loaded")
		}
	})
	return ShippedCatalog()
}

// ShippedCatalog returns the singleton catalog. Before InitShippedCatalog runs
// it is the zero Catalog, which reports "no shipped file" for everything.
func ShippedCatalog() Catalog {
	shippedMu.RLock()
	defer shippedMu.RUnlock()
	return shipped
}
```

- [x] **Step 5: Run tests to verify they pass**

```bash
cd services/atlas-configurations/atlas.com/configurations
go test ./templates/ -run 'TestLoadCatalog|TestZeroCatalog|TestInitShippedCatalog' -v
```

Expected: PASS — all six tests.

- [x] **Step 6: Commit**

```bash
git add services/atlas-configurations/atlas.com/configurations/templates/shipped.go \
        services/atlas-configurations/atlas.com/configurations/templates/shipped_test.go \
        services/atlas-configurations/atlas.com/configurations/templates/testdata
git commit -m "feat(atlas-configurations): add shipped-template catalog"
```

---

## Task 3: Extract `canonicalBytes` from `Create`

**Files:**
- Modify: `services/atlas-configurations/atlas.com/configurations/templates/processor.go:86-122`
- Test: `services/atlas-configurations/atlas.com/configurations/templates/processor_test.go` (append)

**Interfaces:**
- Consumes: `socket.Normalize`, `socketValidate` (`templates/processor.go:167`), `validationFailureError` (`templates/validation_error.go:14`).
- Produces: `func canonicalBytes(input RestModel) (json.RawMessage, error)` — unexported; returns the exact bytes `Create` persists, or a `*validationFailureError` when the socket config fails validation. `Create` and (Task 5) `ReseedById` both call it; `UpdateById` deliberately does not.

- [x] **Step 1: Write the failing test**

Append to `templates/processor_test.go`:

```go
// Design §5 step 2: Entity.Data must equal canonicalBytes(input) exactly.
// This is the pin that keeps re-seed producing a row byte-identical to what a
// fresh boot seed of the same file would produce. If Create ever grows a
// transformation canonicalBytes lacks, re-seeded rows report drift the instant
// they are reset, and this test is what catches it.
func TestCanonicalBytesMatchesCreate(t *testing.T) {
	db := setupTestDB(t)
	l := testLogger()
	ctx := context.Background()
	p := NewProcessor(l, ctx, db)

	in := createTestRestModel("GMS", 83, 1)

	id, err := p.Create(in)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	want, err := canonicalBytes(in)
	if err != nil {
		t.Fatalf("canonicalBytes: %v", err)
	}

	var e Entity
	if err := db.Where("id = ?", id).First(&e).Error; err != nil {
		t.Fatalf("read back: %v", err)
	}
	if string(e.Data) != string(want) {
		t.Errorf("Entity.Data != canonicalBytes\n got: %s\nwant: %s", e.Data, want)
	}
}

// canonicalBytes must reject an invalid socket configuration with the same
// error type Create surfaces, so the re-seed handler's 400 branch can reuse
// the existing validationFailureError rendering verbatim.
func TestCanonicalBytesSurfacesValidationFailure(t *testing.T) {
	in := createTestRestModel("GMS", 83, 1)
	// A handler with no validator is silently dropped by the channel
	// dispatcher, so socket.Validate rejects it.
	in.Socket = socket.RestModel{
		Handlers: []handler.RestModel{{OpCode: "0x01", Handler: "SomeHandle"}},
	}

	_, err := canonicalBytes(in)
	if err == nil {
		t.Fatalf("canonicalBytes accepted an invalid socket configuration")
	}
	var ve *validationFailureError
	if !errors.As(err, &ve) {
		t.Fatalf("error type = %T, want *validationFailureError", err)
	}
}
```

Add these imports to `templates/processor_test.go` if not already present: `errors`, `atlas-configurations/templates/socket`, `atlas-configurations/templates/socket/handler`.

> If `socket.Validate` does not in fact reject a validator-less handler, replace the fixture in `TestCanonicalBytesSurfacesValidationFailure` with whichever minimal socket document `atlas-configurations/socket.Validate` does reject — read `socket/validate.go` and pick the first blocking rule. The assertion (a `*validationFailureError` surfaces) is the point; the specific invalid document is not.

- [x] **Step 2: Run test to verify it fails**

```bash
cd services/atlas-configurations/atlas.com/configurations
go test ./templates/ -run 'TestCanonicalBytes' -v
```

Expected: FAIL — `undefined: canonicalBytes`.

- [x] **Step 3: Write the implementation**

In `templates/processor.go`, add `canonicalBytes` and rewrite `Create` to call it. Replace the body of `Create` (currently lines 86-122) with:

```go
// canonicalBytes applies the write-path normalization and validation and
// returns the EXACT bytes Create persists. Both Create and ReseedById call it,
// which is what makes a re-seeded row byte-identical to a freshly seeded one.
//
// UpdateById deliberately does NOT call it: UpdateById additionally runs the
// preset validator, which reassigns input.Characters.Presets before
// marshalling. Re-seeding through that path would persist bytes differing from
// the shipped file, and the row would report drift again the instant it was
// reset (FR-3.4).
func canonicalBytes(input RestModel) (json.RawMessage, error) {
	input.Socket = socket.Normalize(input.Socket)
	if issues := socketValidate(input.Socket); len(issues) > 0 {
		return nil, &validationFailureError{socketIssues: issues}
	}

	res, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(res), nil
}

func (p *ProcessorImpl) Create(input RestModel) (uuid.UUID, error) {
	data, err := canonicalBytes(input)
	if err != nil {
		return uuid.Nil, err
	}

	// Generate UUID in Go for database portability
	templateId := uuid.New()
	err = database.ExecuteTransaction(p.db, func(db *gorm.DB) error {
		e := &Entity{
			Id:           templateId,
			Region:       input.Region,
			MajorVersion: input.MajorVersion,
			MinorVersion: input.MinorVersion,
			Data:         data,
		}
		return db.Create(e).Error
	})
	if err != nil {
		return uuid.Nil, err
	}
	return templateId, nil
}
```

The removed `rm := &json.RawMessage{}; rm.UnmarshalJSON(res)` round-trip was a no-op: `json.RawMessage.UnmarshalJSON` copies its argument without validating it, so `json.RawMessage(res)` is byte-equivalent.

- [x] **Step 4: Run the full templates package**

```bash
cd services/atlas-configurations/atlas.com/configurations
go test ./templates/... -v
```

Expected: PASS — the two new tests plus every pre-existing test in the package (`Create`'s observable behaviour is unchanged).

- [x] **Step 5: Commit**

```bash
git add services/atlas-configurations/atlas.com/configurations/templates/processor.go \
        services/atlas-configurations/atlas.com/configurations/templates/processor_test.go
git commit -m "refactor(atlas-configurations): extract canonicalBytes from template Create"
```

---

## Task 4: `ViewRestModel`, catalog injection, and drift detection

**Files:**
- Modify: `services/atlas-configurations/atlas.com/configurations/templates/rest.go`
- Modify: `services/atlas-configurations/atlas.com/configurations/templates/processor.go`
- Modify: `services/atlas-configurations/atlas.com/configurations/templates/mock/processor.go`
- Test: `services/atlas-configurations/atlas.com/configurations/templates/processor_test.go` (append)

**Interfaces:**
- Consumes: `Revision` (Task 1), `Catalog` / `LoadCatalog` (Task 2), `Make` (`templates/processor.go:67`).
- Produces:
  - `type ViewRestModel struct { RestModel; ShippedRevision string \`json:"shippedRevision"\`; StoredRevision string \`json:"storedRevision"\`; SeedDrift bool \`json:"seedDrift"\` }`
  - `Processor` gains: `WithCatalog(c Catalog) Processor`, `ViewByIdProvider(templateId uuid.UUID) model.Provider[ViewRestModel]`, `ViewByRegionAndVersionProvider(region string, majorVersion uint16, minorVersion uint16) model.Provider[ViewRestModel]`, `AllViewProvider(page model.Page) model.Provider[model.Paged[ViewRestModel]]`.
  - `ProcessorImpl` gains an unexported `makeView(rm RestModel) (ViewRestModel, error)`.

- [x] **Step 1: Write the failing test**

Append to `templates/processor_test.go`:

```go
// THE load-bearing test (PRD acceptance criteria, design §5). Every shipped
// seed file, created into a fresh database and read back through the view
// provider, must report NO drift. If Create or Normalize perturbs anything the
// revision sees, every template reports permanent phantom drift and the badge
// becomes noise. Table-driven over the directory so a new version bring-up is
// covered without editing this test.
func TestShippedSeedsReportNoDrift(t *testing.T) {
	l := testLogger()
	catalog := LoadCatalog(l, seedTemplatesDir())
	if catalog.Len() == 0 {
		t.Fatalf("no seed templates found at %s - a wrong path would make this test vacuously pass", seedTemplatesDir())
	}

	for _, entry := range catalog.Entries() {
		t.Run(entry.FileName, func(t *testing.T) {
			db := setupTestDB(t)
			ctx := context.Background()
			p := NewProcessor(l, ctx, db).WithCatalog(catalog)

			id, err := p.Create(entry.Model)
			if err != nil {
				t.Fatalf("Create(%s): %v", entry.FileName, err)
			}

			v, err := p.ViewByIdProvider(id)()
			if err != nil {
				t.Fatalf("ViewByIdProvider: %v", err)
			}

			if v.ShippedRevision != entry.Revision {
				t.Errorf("ShippedRevision = %q, want %q", v.ShippedRevision, entry.Revision)
			}
			if v.StoredRevision != entry.Revision {
				t.Errorf("StoredRevision = %q, want %q (Marshal∘Unmarshal is not the identity on this document - see design §5)", v.StoredRevision, entry.Revision)
			}
			if v.SeedDrift {
				t.Errorf("SeedDrift = true for a freshly seeded template")
			}
		})
	}
}

// FR-2.3: a stored row whose content no longer matches the shipped file drifts.
func TestDriftDetectedAfterMutation(t *testing.T) {
	db := setupTestDB(t)
	l := testLogger()
	ctx := context.Background()
	catalog := LoadCatalog(l, seedTemplatesDir())

	entry, ok := catalog.Lookup("GMS", 83, 1)
	if !ok {
		t.Fatalf("GMS 83.1 missing from the seed corpus")
	}

	p := NewProcessor(l, ctx, db).WithCatalog(catalog)
	id, err := p.Create(entry.Model)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Mutate the stored document out from under the row, the way a UI PATCH
	// or an out-of-band edit would.
	mutated := entry.Model
	mutated.UsesPin = !mutated.UsesPin
	data, err := canonicalBytes(mutated)
	if err != nil {
		t.Fatalf("canonicalBytes: %v", err)
	}
	if err := db.Model(&Entity{}).Where("id = ?", id).Update("data", []byte(data)).Error; err != nil {
		t.Fatalf("mutate: %v", err)
	}

	v, err := p.ViewByIdProvider(id)()
	if err != nil {
		t.Fatalf("ViewByIdProvider: %v", err)
	}
	if !v.SeedDrift {
		t.Errorf("SeedDrift = false after mutating the stored document")
	}
	if v.StoredRevision == v.ShippedRevision {
		t.Errorf("revisions are equal after mutation: %q", v.StoredRevision)
	}
	if v.ShippedRevision != entry.Revision {
		t.Errorf("ShippedRevision = %q, want %q - the shipped side must not move", v.ShippedRevision, entry.Revision)
	}
}

// FR-2.4: absence of a shipped file is NOT drift. The template reports an
// empty shippedRevision and seedDrift false.
func TestNoCatalogEntryIsNotDrift(t *testing.T) {
	db := setupTestDB(t)
	l := testLogger()
	ctx := context.Background()

	// A catalog that knows only about the TEST fixture, so GMS 83.1 misses.
	catalog := LoadCatalog(l, filepath.Join("testdata", "templates"))
	p := NewProcessor(l, ctx, db).WithCatalog(catalog)

	id, err := p.Create(createTestRestModel("GMS", 83, 1))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	v, err := p.ViewByIdProvider(id)()
	if err != nil {
		t.Fatalf("ViewByIdProvider: %v", err)
	}
	if v.ShippedRevision != "" {
		t.Errorf("ShippedRevision = %q, want empty", v.ShippedRevision)
	}
	if v.SeedDrift {
		t.Errorf("SeedDrift = true with no shipped file")
	}
	if v.StoredRevision == "" {
		t.Errorf("StoredRevision is empty - it is always computed")
	}
}

// A processor with no catalog wired must degrade to FR-2.4 behaviour rather
// than panicking on a nil map.
func TestUnwiredProcessorReportsNoShippedFile(t *testing.T) {
	db := setupTestDB(t)
	l := testLogger()
	ctx := context.Background()
	p := NewProcessor(l, ctx, db)

	id, err := p.Create(createTestRestModel("GMS", 83, 1))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	v, err := p.ViewByIdProvider(id)()
	if err != nil {
		t.Fatalf("ViewByIdProvider: %v", err)
	}
	if v.ShippedRevision != "" || v.SeedDrift {
		t.Errorf("unwired processor reported shipped=%q drift=%v, want \"\"/false", v.ShippedRevision, v.SeedDrift)
	}
}

// The list and by-region-and-version read paths must carry the same computed
// attributes as the by-id path - the list page renders per-row badges from them.
func TestViewProvidersAgreeAcrossReadPaths(t *testing.T) {
	db := setupTestDB(t)
	l := testLogger()
	ctx := context.Background()
	catalog := LoadCatalog(l, seedTemplatesDir())

	entry, ok := catalog.Lookup("GMS", 83, 1)
	if !ok {
		t.Fatalf("GMS 83.1 missing from the seed corpus")
	}
	p := NewProcessor(l, ctx, db).WithCatalog(catalog)
	id, err := p.Create(entry.Model)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	byId, err := p.ViewByIdProvider(id)()
	if err != nil {
		t.Fatalf("ViewByIdProvider: %v", err)
	}
	byVersion, err := p.ViewByRegionAndVersionProvider("GMS", 83, 1)()
	if err != nil {
		t.Fatalf("ViewByRegionAndVersionProvider: %v", err)
	}
	paged, err := p.AllViewProvider(model.Page{Number: 1, Size: 50})()
	if err != nil {
		t.Fatalf("AllViewProvider: %v", err)
	}
	if len(paged.Items) != 1 {
		t.Fatalf("AllViewProvider returned %d items, want 1", len(paged.Items))
	}

	for name, got := range map[string]ViewRestModel{"byVersion": byVersion, "all": paged.Items[0]} {
		if got.ShippedRevision != byId.ShippedRevision || got.StoredRevision != byId.StoredRevision || got.SeedDrift != byId.SeedDrift {
			t.Errorf("%s view disagrees with by-id view: %+v vs %+v", name, got, byId)
		}
	}
}

// D3: the three computed attributes must NOT be persisted. If they ever land
// on RestModel, Create writes them into Entity.Data, Make reads them back, and
// the revision is computed over bytes containing a previous revision.
func TestComputedAttributesAreNotPersisted(t *testing.T) {
	db := setupTestDB(t)
	l := testLogger()
	ctx := context.Background()
	catalog := LoadCatalog(l, seedTemplatesDir())

	entry, ok := catalog.Lookup("GMS", 83, 1)
	if !ok {
		t.Fatalf("GMS 83.1 missing from the seed corpus")
	}
	p := NewProcessor(l, ctx, db).WithCatalog(catalog)
	id, err := p.Create(entry.Model)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	var e Entity
	if err := db.Where("id = ?", id).First(&e).Error; err != nil {
		t.Fatalf("read back: %v", err)
	}
	var stored map[string]any
	if err := json.Unmarshal(e.Data, &stored); err != nil {
		t.Fatalf("unmarshal stored document: %v", err)
	}
	for _, k := range []string{"shippedRevision", "storedRevision", "seedDrift"} {
		if _, present := stored[k]; present {
			t.Errorf("computed attribute %q leaked into the stored document", k)
		}
	}
}

// The view model's JSON:API identity must promote from the embedded RestModel,
// or api2go emits the wrong resource type and no id.
func TestViewRestModelIdentityPromotes(t *testing.T) {
	v := ViewRestModel{RestModel: RestModel{Id: "abc"}}
	if v.GetName() != "templates" {
		t.Errorf("GetName() = %q, want templates", v.GetName())
	}
	if v.GetID() != "abc" {
		t.Errorf("GetID() = %q, want abc", v.GetID())
	}
	if err := v.SetID("def"); err != nil {
		t.Fatalf("SetID: %v", err)
	}
	if v.GetID() != "def" {
		t.Errorf("after SetID, GetID() = %q, want def", v.GetID())
	}
}

// encoding/json flattens anonymous embedded structs, so the wire shape is
// RestModel's attributes plus exactly three keys - no per-field restating.
func TestViewRestModelFlattensEmbeddedFields(t *testing.T) {
	v := ViewRestModel{
		RestModel:       createTestRestModel("GMS", 83, 1),
		ShippedRevision: "aa",
		StoredRevision:  "bb",
		SeedDrift:       true,
	}
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	for _, k := range []string{"region", "majorVersion", "minorVersion", "usesPin", "socket", "shippedRevision", "storedRevision", "seedDrift"} {
		if _, present := got[k]; !present {
			t.Errorf("key %q missing from the marshalled view model", k)
		}
	}
	if _, present := got["RestModel"]; present {
		t.Errorf("embedded struct was nested under \"RestModel\" instead of flattened")
	}
}
```

Add `path/filepath` to `templates/processor_test.go`'s imports if not already present.

- [x] **Step 2: Run test to verify it fails**

```bash
cd services/atlas-configurations/atlas.com/configurations
go test ./templates/ -run 'TestShippedSeeds|TestDrift|TestNoCatalogEntry|TestUnwired|TestViewProviders|TestComputedAttributes|TestViewRestModel' -v
```

Expected: FAIL — `undefined: ViewRestModel`, `p.WithCatalog undefined`, `p.ViewByIdProvider undefined`.

- [x] **Step 3: Add `ViewRestModel`**

Append to `templates/rest.go`:

```go
// ViewRestModel is the READ-ONLY projection of a template: RestModel plus the
// three computed drift attributes. It is a separate type on purpose (design
// D3): Create persists json.Marshal(input) verbatim, so any field with a JSON
// tag on RestModel would be written INTO the stored document, read back by
// Make, and folded into the next revision - self-reference and permanent
// phantom drift. Keeping the write model untouched means that failure class
// does not exist rather than being defended against.
//
// encoding/json flattens anonymous embedded structs, and api2go builds the
// attributes object with a plain json.Marshal, so the wire shape is exactly
// RestModel's attributes plus three keys. GetName / GetID / SetID promote from
// the embedded RestModel.
//
// The PATCH path still binds RestModel, so the three attributes are ignored on
// write by omission rather than by code (PRD §5.1).
type ViewRestModel struct {
	RestModel
	ShippedRevision string `json:"shippedRevision"`
	StoredRevision  string `json:"storedRevision"`
	SeedDrift       bool   `json:"seedDrift"`
}
```

- [x] **Step 4: Add catalog injection and the view providers**

In `templates/processor.go`:

Extend the `Processor` interface (after `WithValidator`):

```go
type Processor interface {
	WithValidator(v *preset.Validator) Processor
	WithCatalog(c Catalog) Processor
	ByRegionAndVersionProvider(region string, majorVersion uint16, minorVersion uint16) model.Provider[RestModel]
	ByIdProvider(templateId uuid.UUID) model.Provider[RestModel]
	AllProvider(page model.Page) model.Provider[model.Paged[RestModel]]
	ViewByRegionAndVersionProvider(region string, majorVersion uint16, minorVersion uint16) model.Provider[ViewRestModel]
	ViewByIdProvider(templateId uuid.UUID) model.Provider[ViewRestModel]
	AllViewProvider(page model.Page) model.Provider[model.Paged[ViewRestModel]]
	GetByRegionAndVersion(region string, majorVersion uint16, minorVersion uint16) (RestModel, error)
	GetById(templateId uuid.UUID) (RestModel, error)
	Create(input RestModel) (uuid.UUID, error)
	UpdateById(templateId uuid.UUID, input RestModel) error
	DeleteById(templateId uuid.UUID) error
}
```

Add the `catalog` field to `ProcessorImpl`:

```go
type ProcessorImpl struct {
	l         logrus.FieldLogger
	ctx       context.Context
	db        *gorm.DB
	validator *preset.Validator
	catalog   Catalog
}
```

Add, next to `WithValidator`:

```go
// WithCatalog injects the shipped-template catalog used to compute drift.
// Unset means the zero Catalog, which reports "no shipped file" for every
// template - the FR-2.4 behaviour - so an un-wired processor degrades safely
// rather than panicking.
func (p *ProcessorImpl) WithCatalog(c Catalog) Processor {
	p.catalog = c
	return p
}
```

Add the view providers next to the existing ones:

```go
// makeView decorates a RestModel with its revisions and drift flag (FR-2).
// Drift is computed on read; nothing is persisted, so there is no cache to
// invalidate and no state that can itself go stale.
func (p *ProcessorImpl) makeView(rm RestModel) (ViewRestModel, error) {
	stored, err := Revision(rm)
	if err != nil {
		return ViewRestModel{}, err
	}

	v := ViewRestModel{RestModel: rm, StoredRevision: stored}
	if entry, ok := p.catalog.Lookup(rm.Region, rm.MajorVersion, rm.MinorVersion); ok {
		v.ShippedRevision = entry.Revision
		v.SeedDrift = entry.Revision != stored
	}
	// No catalog entry: shippedRevision stays empty and seedDrift stays false.
	// Absence of a shipped file is not drift (FR-2.4).
	return v, nil
}

func (p *ProcessorImpl) ViewByRegionAndVersionProvider(region string, majorVersion uint16, minorVersion uint16) model.Provider[ViewRestModel] {
	return model.Map(p.makeView)(p.ByRegionAndVersionProvider(region, majorVersion, minorVersion))
}

func (p *ProcessorImpl) ViewByIdProvider(templateId uuid.UUID) model.Provider[ViewRestModel] {
	return model.Map(p.makeView)(p.ByIdProvider(templateId))
}

// AllViewProvider maps over AllProvider, which already runs ParallelMap, so
// the per-row SHA-256 is parallel across the page (NFR-2) without a cache.
func (p *ProcessorImpl) AllViewProvider(page model.Page) model.Provider[model.Paged[ViewRestModel]] {
	return model.MapPaged(p.makeView)(p.AllProvider(page))(model.ParallelMap())
}
```

> `model.MapPaged` is used by `AllProvider` at `templates/processor.go:64` over an entity provider. If its signature does not accept a `model.Provider[model.Paged[RestModel]]` directly, map the paged provider inline instead — read `libs/atlas-model/model` for the exact combinator and use whichever one composes `Provider[Paged[A]]` + `func(A) (B, error)` into `Provider[Paged[B]]`. Keep `model.ParallelMap()` in the call so NFR-2 still holds.

- [x] **Step 5: Update the mock**

In `templates/mock/processor.go`, add the four fields and four methods:

```go
	WithCatalogFunc                    func(c templates.Catalog) templates.Processor
	ViewByRegionAndVersionProviderFunc func(region string, majorVersion uint16, minorVersion uint16) model.Provider[templates.ViewRestModel]
	ViewByIdProviderFunc               func(templateId uuid.UUID) model.Provider[templates.ViewRestModel]
	AllViewProviderFunc                func(page model.Page) model.Provider[model.Paged[templates.ViewRestModel]]
```

```go
func (m *ProcessorMock) WithCatalog(c templates.Catalog) templates.Processor {
	if m.WithCatalogFunc != nil {
		return m.WithCatalogFunc(c)
	}
	return m
}

func (m *ProcessorMock) ViewByRegionAndVersionProvider(region string, majorVersion uint16, minorVersion uint16) model.Provider[templates.ViewRestModel] {
	if m.ViewByRegionAndVersionProviderFunc != nil {
		return m.ViewByRegionAndVersionProviderFunc(region, majorVersion, minorVersion)
	}
	return model.FixedProvider(templates.ViewRestModel{})
}

func (m *ProcessorMock) ViewByIdProvider(templateId uuid.UUID) model.Provider[templates.ViewRestModel] {
	if m.ViewByIdProviderFunc != nil {
		return m.ViewByIdProviderFunc(templateId)
	}
	return model.FixedProvider(templates.ViewRestModel{})
}

func (m *ProcessorMock) AllViewProvider(page model.Page) model.Provider[model.Paged[templates.ViewRestModel]] {
	if m.AllViewProviderFunc != nil {
		return m.AllViewProviderFunc(page)
	}
	return model.FixedProvider(model.Paged[templates.ViewRestModel]{})
}
```

- [x] **Step 6: Run tests to verify they pass**

```bash
cd services/atlas-configurations/atlas.com/configurations
go build ./... && go test ./templates/... -v
```

Expected: PASS — every new test, including all eleven subtests of `TestShippedSeedsReportNoDrift`, plus every pre-existing test.

If `TestShippedSeedsReportNoDrift` fails on `StoredRevision != ShippedRevision`, do **not** weaken the test: `Marshal ∘ Unmarshal` is no longer the identity on `RestModel`, which is exactly the failure design §5 exists to catch. Find the offending field (add a `t.Logf` dumping both marshalled documents and diff them) and fix the round-trip.

- [x] **Step 7: Commit**

```bash
git add services/atlas-configurations/atlas.com/configurations/templates/rest.go \
        services/atlas-configurations/atlas.com/configurations/templates/processor.go \
        services/atlas-configurations/atlas.com/configurations/templates/processor_test.go \
        services/atlas-configurations/atlas.com/configurations/templates/mock/processor.go
git commit -m "feat(atlas-configurations): compute template seed drift on read"
```

---

## Task 5: `ReseedById` and the sentinel errors

**Files:**
- Modify: `services/atlas-configurations/atlas.com/configurations/templates/processor.go`
- Modify: `services/atlas-configurations/atlas.com/configurations/templates/mock/processor.go`
- Test: `services/atlas-configurations/atlas.com/configurations/templates/processor_test.go` (append)

**Interfaces:**
- Consumes: `canonicalBytes` (Task 3), `Catalog.Lookup` (Task 2), `Revision` (Task 1), `update` (`templates/administrator.go:12`), `byIdEntityProvider` (`templates/provider.go:20`), `database.ExecuteTransaction`.
- Produces:
  - `var ErrTemplateNotFound = errors.New("template not found")`
  - `var ErrNoShippedTemplate = errors.New("no shipped template")`
  - `Processor` gains `ReseedById(templateId uuid.UUID) error`.

- [x] **Step 1: Write the failing test**

Append to `templates/processor_test.go`:

```go
// FR-3.1 / FR-3.2 / FR-3.3: re-seed restores the shipped content, preserves
// the row's UUID, and leaves the region/version key untouched.
func TestReseedRestoresShippedContent(t *testing.T) {
	db := setupTestDB(t)
	l := testLogger()
	ctx := context.Background()
	catalog := LoadCatalog(l, seedTemplatesDir())

	entry, ok := catalog.Lookup("GMS", 83, 1)
	if !ok {
		t.Fatalf("GMS 83.1 missing from the seed corpus")
	}
	p := NewProcessor(l, ctx, db).WithCatalog(catalog)
	id, err := p.Create(entry.Model)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Drift the row.
	mutated := entry.Model
	mutated.UsesPin = !mutated.UsesPin
	data, err := canonicalBytes(mutated)
	if err != nil {
		t.Fatalf("canonicalBytes: %v", err)
	}
	if err := db.Model(&Entity{}).Where("id = ?", id).Update("data", []byte(data)).Error; err != nil {
		t.Fatalf("mutate: %v", err)
	}
	before, err := p.ViewByIdProvider(id)()
	if err != nil {
		t.Fatalf("ViewByIdProvider (before): %v", err)
	}
	if !before.SeedDrift {
		t.Fatalf("precondition failed: template is not drifted")
	}

	if err := p.ReseedById(id); err != nil {
		t.Fatalf("ReseedById: %v", err)
	}

	after, err := p.ViewByIdProvider(id)()
	if err != nil {
		t.Fatalf("ViewByIdProvider (after): %v", err)
	}
	if after.SeedDrift {
		t.Errorf("SeedDrift = true after re-seed")
	}
	if after.StoredRevision != entry.Revision {
		t.Errorf("StoredRevision = %q, want %q", after.StoredRevision, entry.Revision)
	}
	if after.UsesPin != entry.Model.UsesPin {
		t.Errorf("UsesPin = %v, want %v - the mutation survived the re-seed", after.UsesPin, entry.Model.UsesPin)
	}

	var e Entity
	if err := db.Where("id = ?", id).First(&e).Error; err != nil {
		t.Fatalf("read back: %v", err)
	}
	if e.Id != id {
		t.Errorf("Id = %s, want %s - re-seed must never delete-and-recreate", e.Id, id)
	}
	if e.Region != "GMS" || e.MajorVersion != 83 || e.MinorVersion != 1 {
		t.Errorf("key changed: (%s,%d,%d), want (GMS,83,1)", e.Region, e.MajorVersion, e.MinorVersion)
	}
}

// FR-3.7: re-seeding an undrifted template succeeds and leaves the row
// byte-identical.
func TestReseedIsIdempotent(t *testing.T) {
	db := setupTestDB(t)
	l := testLogger()
	ctx := context.Background()
	catalog := LoadCatalog(l, seedTemplatesDir())

	entry, ok := catalog.Lookup("GMS", 83, 1)
	if !ok {
		t.Fatalf("GMS 83.1 missing from the seed corpus")
	}
	p := NewProcessor(l, ctx, db).WithCatalog(catalog)
	id, err := p.Create(entry.Model)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	var first Entity
	if err := db.Where("id = ?", id).First(&first).Error; err != nil {
		t.Fatalf("read back: %v", err)
	}

	for i := 0; i < 2; i++ {
		if err := p.ReseedById(id); err != nil {
			t.Fatalf("ReseedById (call %d): %v", i+1, err)
		}
	}

	var last Entity
	if err := db.Where("id = ?", id).First(&last).Error; err != nil {
		t.Fatalf("read back: %v", err)
	}
	if string(last.Data) != string(first.Data) {
		t.Errorf("re-seed is not idempotent - Data changed")
	}
}

// The 404 branch: an id with no row.
func TestReseedUnknownIdReturnsNotFound(t *testing.T) {
	db := setupTestDB(t)
	l := testLogger()
	ctx := context.Background()
	p := NewProcessor(l, ctx, db).WithCatalog(LoadCatalog(l, seedTemplatesDir()))

	err := p.ReseedById(uuid.New())
	if !errors.Is(err, ErrTemplateNotFound) {
		t.Errorf("error = %v, want ErrTemplateNotFound", err)
	}
}

// The 409 branch: the row exists but no seed file ships for its key.
func TestReseedNoShippedFileReturnsConflict(t *testing.T) {
	db := setupTestDB(t)
	l := testLogger()
	ctx := context.Background()

	// A catalog that knows only the TEST fixture, so GMS 83.1 misses.
	p := NewProcessor(l, ctx, db).WithCatalog(LoadCatalog(l, filepath.Join("testdata", "templates")))
	id, err := p.Create(createTestRestModel("GMS", 83, 1))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	err = p.ReseedById(id)
	if !errors.Is(err, ErrNoShippedTemplate) {
		t.Errorf("error = %v, want ErrNoShippedTemplate", err)
	}
}

// FR-3.4: re-seed must produce a row byte-identical to a fresh Create of the
// same file - i.e. it must NOT route through UpdateById's preset validator.
func TestReseedProducesSameBytesAsFreshCreate(t *testing.T) {
	l := testLogger()
	ctx := context.Background()
	catalog := LoadCatalog(l, seedTemplatesDir())

	entry, ok := catalog.Lookup("GMS", 83, 1)
	if !ok {
		t.Fatalf("GMS 83.1 missing from the seed corpus")
	}

	freshDB := setupTestDB(t)
	freshP := NewProcessor(l, ctx, freshDB).WithCatalog(catalog)
	freshId, err := freshP.Create(entry.Model)
	if err != nil {
		t.Fatalf("Create (fresh): %v", err)
	}
	var fresh Entity
	if err := freshDB.Where("id = ?", freshId).First(&fresh).Error; err != nil {
		t.Fatalf("read back (fresh): %v", err)
	}

	reseedDB := setupTestDB(t)
	reseedP := NewProcessor(l, ctx, reseedDB).WithCatalog(catalog)
	reseedId, err := reseedP.Create(createTestRestModel("GMS", 83, 1))
	if err != nil {
		t.Fatalf("Create (stub): %v", err)
	}
	if err := reseedP.ReseedById(reseedId); err != nil {
		t.Fatalf("ReseedById: %v", err)
	}
	var reseeded Entity
	if err := reseedDB.Where("id = ?", reseedId).First(&reseeded).Error; err != nil {
		t.Fatalf("read back (reseeded): %v", err)
	}

	if string(reseeded.Data) != string(fresh.Data) {
		t.Errorf("re-seeded bytes differ from a fresh boot seed of the same file")
	}
}
```

Add `errors` and `github.com/google/uuid` to `templates/processor_test.go`'s imports if not already present.

- [x] **Step 2: Run test to verify it fails**

```bash
cd services/atlas-configurations/atlas.com/configurations
go test ./templates/ -run 'TestReseed' -v
```

Expected: FAIL — `undefined: ErrTemplateNotFound`, `p.ReseedById undefined`.

- [x] **Step 3: Write the implementation**

In `templates/processor.go`, add the sentinels near the top of the file (after the imports):

```go
// Sentinel errors the re-seed handler maps to HTTP statuses. server.
// WriteErrorResponse maps everything to 500, so the handler switches on these
// and writes the JSON:API error document itself (design D6).
var (
	// ErrTemplateNotFound wraps gorm.ErrRecordNotFound for a template id -> 404.
	ErrTemplateNotFound = errors.New("template not found")
	// ErrNoShippedTemplate means the row exists but this image ships no seed
	// file for its region/version -> 409. There is nothing to reset to.
	ErrNoShippedTemplate = errors.New("no shipped template")
)
```

Add `errors` and `fmt` to the file's imports.

Add `ReseedById(templateId uuid.UUID) error` to the `Processor` interface, and implement it:

```go
// ReseedById replaces a template's stored content with the file this image
// ships for its region/version (FR-3.1).
//
// It writes through canonicalBytes - Create's exact validation and marshalling
// - not UpdateById, whose preset validator would reassign
// input.Characters.Presets and persist bytes differing from the shipped file
// (FR-3.4). It reuses the existing `update` transaction function with the
// ENTITY's region/version columns rather than the file's, so a hypothetical
// key mismatch cannot rewrite the lookup key (FR-3.3).
func (p *ProcessorImpl) ReseedById(templateId uuid.UUID) error {
	e, err := byIdEntityProvider(p.ctx)(templateId)(p.db)()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("%w: %s", ErrTemplateNotFound, templateId)
		}
		return err
	}

	entry, ok := p.catalog.Lookup(e.Region, e.MajorVersion, e.MinorVersion)
	if !ok {
		return fmt.Errorf("%w for %s %d.%d", ErrNoShippedTemplate, e.Region, e.MajorVersion, e.MinorVersion)
	}

	data, err := canonicalBytes(entry.Model)
	if err != nil {
		return err
	}

	// Best-effort: the point of re-seed is to repair a row, so a row whose
	// stored document is too broken to hash must still be repairable. Log and
	// carry on with an empty before-revision rather than failing.
	beforeRevision := ""
	if rm, mErr := Make(e); mErr == nil {
		if rev, rErr := Revision(rm); rErr == nil {
			beforeRevision = rev
		} else {
			p.l.WithError(rErr).WithField("templateId", templateId.String()).Warn("Unable to compute pre-reseed revision")
		}
	} else {
		p.l.WithError(mErr).WithField("templateId", templateId.String()).Warn("Unable to read pre-reseed template document")
	}

	if err := database.ExecuteTransaction(p.db, update(p.ctx, templateId, e.Region, e.MajorVersion, e.MinorVersion, data)); err != nil {
		return err
	}

	// NFR-3: the change must be reconstructable from logs alone.
	p.l.WithFields(logrus.Fields{
		"templateId":     templateId.String(),
		"region":         e.Region,
		"majorVersion":   e.MajorVersion,
		"minorVersion":   e.MinorVersion,
		"file":           entry.FileName,
		"beforeRevision": beforeRevision,
		"afterRevision":  entry.Revision,
	}).Info("Template re-seeded from shipped defaults")

	return nil
}
```

- [x] **Step 4: Update the mock**

In `templates/mock/processor.go`, add:

```go
	ReseedByIdFunc func(templateId uuid.UUID) error
```

```go
func (m *ProcessorMock) ReseedById(templateId uuid.UUID) error {
	if m.ReseedByIdFunc != nil {
		return m.ReseedByIdFunc(templateId)
	}
	return nil
}
```

- [x] **Step 5: Run tests to verify they pass**

```bash
cd services/atlas-configurations/atlas.com/configurations
go build ./... && go test ./templates/... -v
```

Expected: PASS — all five `TestReseed*` tests plus everything before them.

- [x] **Step 6: Commit**

```bash
git add services/atlas-configurations/atlas.com/configurations/templates/processor.go \
        services/atlas-configurations/atlas.com/configurations/templates/processor_test.go \
        services/atlas-configurations/atlas.com/configurations/templates/mock/processor.go
git commit -m "feat(atlas-configurations): add template ReseedById"
```

---

## Task 6: The REST surface — view attributes and `POST /{templateId}/reseed`

**Files:**
- Modify: `services/atlas-configurations/atlas.com/configurations/templates/resource.go`
- Create: `services/atlas-configurations/atlas.com/configurations/templates/resource_reseed_test.go`

**Interfaces:**
- Consumes: `ViewRestModel`, `ShippedCatalog()`, `ReseedById`, `ErrTemplateNotFound`, `ErrNoShippedTemplate`, `validationFailureError.AsJSONAPIErrors` (`templates/validation_error.go:29`), `rest.ParseTemplateId` (`rest/handler.go:55`).
- Produces: route `POST /configurations/templates/{templateId}/reseed`; unexported `func writeJSONAPIError(w http.ResponseWriter, status int, title string, detail string)`. The three GET handlers and the POST-create handler now marshal `ViewRestModel`.

- [x] **Step 1: Write the failing test**

Create `templates/resource_reseed_test.go`:

```go
package templates

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

func TestReseedReturnsNoContent(t *testing.T) {
	db := setupTestDB(t)
	l := testLogger()
	InitShippedCatalog(l, seedTemplatesDir())

	catalog := ShippedCatalog()
	entry, ok := catalog.Lookup("GMS", 83, 1)
	if !ok {
		t.Fatalf("GMS 83.1 missing from the seed corpus")
	}

	p := NewProcessor(l, context.Background(), db).WithCatalog(catalog)
	id, err := p.Create(entry.Model)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Drift the row so the re-seed has something to undo.
	mutated := entry.Model
	mutated.UsesPin = !mutated.UsesPin
	data, err := canonicalBytes(mutated)
	if err != nil {
		t.Fatalf("canonicalBytes: %v", err)
	}
	if err := db.Model(&Entity{}).Where("id = ?", id).Update("data", []byte(data)).Error; err != nil {
		t.Fatalf("mutate: %v", err)
	}

	router := mux.NewRouter()
	InitResource(testServerInformation{})(db)(router, l)

	req := httptest.NewRequest(http.MethodPost, "/configurations/templates/"+id.String()+"/reseed", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204, body=%s", rr.Code, rr.Body.String())
	}
	if rr.Body.Len() != 0 {
		t.Fatalf("body = %q, want empty", rr.Body.String())
	}

	v, err := p.ViewByIdProvider(id)()
	if err != nil {
		t.Fatalf("ViewByIdProvider: %v", err)
	}
	if v.SeedDrift {
		t.Errorf("SeedDrift = true after a 204 re-seed")
	}
}

func TestReseedUnknownIdReturns404(t *testing.T) {
	db := setupTestDB(t)
	l := testLogger()
	InitShippedCatalog(l, seedTemplatesDir())

	router := mux.NewRouter()
	InitResource(testServerInformation{})(db)(router, l)

	req := httptest.NewRequest(http.MethodPost, "/configurations/templates/"+uuid.New().String()+"/reseed", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404, body=%s", rr.Code, rr.Body.String())
	}
	assertJSONAPIErrorDocument(t, rr, "404")
}

func TestReseedNoShippedFileReturns409(t *testing.T) {
	db := setupTestDB(t)
	l := testLogger()
	InitShippedCatalog(l, seedTemplatesDir())

	// A region/version the real corpus does not ship.
	p := NewProcessor(l, context.Background(), db).WithCatalog(ShippedCatalog())
	id, err := p.Create(createTestRestModel("NOSUCH", 999, 999))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	router := mux.NewRouter()
	InitResource(testServerInformation{})(db)(router, l)

	req := httptest.NewRequest(http.MethodPost, "/configurations/templates/"+id.String()+"/reseed", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409, body=%s", rr.Code, rr.Body.String())
	}
	assertJSONAPIErrorDocument(t, rr, "409")
}

// The GET-by-id read path must carry the three computed attributes, or the UI
// has nothing to render a badge from.
func TestGetTemplateByIdCarriesDriftAttributes(t *testing.T) {
	db := setupTestDB(t)
	l := testLogger()
	InitShippedCatalog(l, seedTemplatesDir())

	entry, ok := ShippedCatalog().Lookup("GMS", 83, 1)
	if !ok {
		t.Fatalf("GMS 83.1 missing from the seed corpus")
	}
	p := NewProcessor(l, context.Background(), db).WithCatalog(ShippedCatalog())
	id, err := p.Create(entry.Model)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	router := mux.NewRouter()
	InitResource(testServerInformation{})(db)(router, l)

	req := httptest.NewRequest(http.MethodGet, "/configurations/templates/"+id.String(), nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rr.Code, rr.Body.String())
	}

	var doc struct {
		Data struct {
			Type       string         `json:"type"`
			Id         string         `json:"id"`
			Attributes map[string]any `json:"attributes"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &doc); err != nil {
		t.Fatalf("unmarshal response: %v (body=%s)", err, rr.Body.String())
	}
	if doc.Data.Type != "templates" {
		t.Errorf("type = %q, want templates", doc.Data.Type)
	}
	if doc.Data.Id != id.String() {
		t.Errorf("id = %q, want %q", doc.Data.Id, id.String())
	}
	if got := doc.Data.Attributes["shippedRevision"]; got != entry.Revision {
		t.Errorf("shippedRevision = %v, want %q", got, entry.Revision)
	}
	if got := doc.Data.Attributes["storedRevision"]; got != entry.Revision {
		t.Errorf("storedRevision = %v, want %q", got, entry.Revision)
	}
	if got := doc.Data.Attributes["seedDrift"]; got != false {
		t.Errorf("seedDrift = %v, want false", got)
	}
	// The read shape must still carry the ordinary template attributes.
	if _, present := doc.Data.Attributes["socket"]; !present {
		t.Errorf("socket attribute missing - the embedded RestModel did not flatten")
	}
}

func assertJSONAPIErrorDocument(t *testing.T, rr *httptest.ResponseRecorder, wantStatus string) {
	t.Helper()
	if ct := rr.Header().Get("Content-Type"); ct != "application/vnd.api+json" {
		t.Errorf("Content-Type = %q, want application/vnd.api+json", ct)
	}
	var doc struct {
		Errors []struct {
			Status string `json:"status"`
			Title  string `json:"title"`
			Detail string `json:"detail"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &doc); err != nil {
		t.Fatalf("unmarshal error document: %v (body=%s)", err, rr.Body.String())
	}
	if len(doc.Errors) != 1 {
		t.Fatalf("len(errors) = %d, want 1 (body=%s)", len(doc.Errors), rr.Body.String())
	}
	if doc.Errors[0].Status != wantStatus {
		t.Errorf("errors[0].status = %q, want %q", doc.Errors[0].Status, wantStatus)
	}
	if doc.Errors[0].Title == "" || doc.Errors[0].Detail == "" {
		t.Errorf("errors[0] has an empty title or detail: %+v", doc.Errors[0])
	}
}
```

- [x] **Step 2: Run test to verify it fails**

```bash
cd services/atlas-configurations/atlas.com/configurations
go test ./templates/ -run 'TestReseedReturns|TestReseedUnknownIdReturns404|TestReseedNoShippedFileReturns409|TestGetTemplateByIdCarries' -v
```

Expected: FAIL — 404/405 on the reseed route, and `shippedRevision` absent from the GET response.

- [x] **Step 3: Register the route and switch the read handlers**

In `templates/resource.go`:

Add the route inside `InitResource`, after the DELETE line:

```go
				r.HandleFunc("/{templateId}/reseed", rest.RegisterHandler(l)(si)("reseed_configuration_template", handleReseedConfigurationTemplate(db))).Methods(http.MethodPost)
```

Add a small constructor so every handler wires the catalog identically:

```go
// viewProcessor is the read/re-seed processor: the ordinary processor with the
// shipped-template catalog attached, so drift is computable. The write paths
// (create/update) deliberately do NOT need it.
func viewProcessor(d *rest.HandlerDependency, db *gorm.DB) Processor {
	return NewProcessor(d.Logger(), d.Context(), db).WithCatalog(ShippedCatalog())
}
```

Change the three GET handlers and the create handler to produce `ViewRestModel`:

- `handleGetConfigurationTemplate`: replace the `GetByRegionAndVersion` call with `viewProcessor(d, db).ViewByRegionAndVersionProvider(region, majorVersion, minorVersion)()` and marshal with `server.MarshalResponse[ViewRestModel]`.
- `handleGetConfigurationTemplates`: replace `AllProvider(page)()` with `viewProcessor(d, db).AllViewProvider(page)()` and marshal with `server.MarshalPaginatedResponse[[]ViewRestModel]`.
- `handleGetConfigurationTemplateById`: replace `GetById(templateId)` with `viewProcessor(d, db).ViewByIdProvider(templateId)()` and marshal with `server.MarshalResponse[ViewRestModel]`.
- `handleCreateConfigurationTemplate`: after a successful `Create`, read the persisted row back so the create response has the identical shape as every read:

```go
				// Read back through the view provider so POST returns exactly
				// what a subsequent GET returns - same attributes, same
				// computed revisions (design D3).
				view, err := viewProcessor(d, db).ViewByIdProvider(templateId)()
				if err != nil {
					d.Logger().WithError(err).Errorf("Unable to read back created configuration template.")
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				// Set the Location header to the URL of the newly created resource
				w.Header().Set("Location", "/configurations/templates/"+templateId.String())

				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				w.WriteHeader(http.StatusCreated)
				server.MarshalResponse[ViewRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(view)
```

(The old `input.Id = templateId.String()` echo goes away — the response is now a real read.)

Add the error writer and the re-seed handler:

```go
// writeJSONAPIError emits the same document shape validationFailureError
// renders, for the statuses server.WriteErrorResponse cannot express (it maps
// everything to 500/503). Keeps the re-seed endpoint's 404 and 409 consistent
// with the existing 400s.
func writeJSONAPIError(w http.ResponseWriter, status int, title string, detail string) {
	w.Header().Set("Content-Type", "application/vnd.api+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"errors": []map[string]any{{
			"status": strconv.Itoa(status),
			"title":  title,
			"detail": detail,
		}},
	})
}

func handleReseedConfigurationTemplate(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseTemplateId(d.Logger(), func(templateId uuid.UUID) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				err := viewProcessor(d, db).ReseedById(templateId)
				if err == nil {
					w.WriteHeader(http.StatusNoContent)
					return
				}

				switch {
				case errors.Is(err, ErrTemplateNotFound):
					writeJSONAPIError(w, http.StatusNotFound, "template not found", "No configuration template exists with id "+templateId.String()+".")
				case errors.Is(err, ErrNoShippedTemplate):
					writeJSONAPIError(w, http.StatusConflict, "no shipped template", "This image ships no seed file for the template's region and version, so there is nothing to reset to.")
				default:
					var ve *validationFailureError
					if errors.As(err, &ve) {
						// A broken seed file: CI-guarded, so this should not
						// occur. Rendered identically to create/update
						// validation failures.
						w.Header().Set("Content-Type", "application/vnd.api+json")
						w.WriteHeader(http.StatusBadRequest)
						_ = json.NewEncoder(w).Encode(map[string]any{"errors": ve.AsJSONAPIErrors()})
						return
					}
					d.Logger().WithError(err).Errorf("Unable to re-seed configuration template.")
					server.WriteErrorResponse(d.Logger())(w)(err)
				}
			}
		})
	}
}
```

Add `strconv` to the file's imports.

- [x] **Step 4: Run tests to verify they pass**

```bash
cd services/atlas-configurations/atlas.com/configurations
go build ./... && go test ./templates/... -v
```

Expected: PASS — the four new resource tests plus every pre-existing one, including `resource_paginate_test.go` and `resource_no_content_test.go`. If a pre-existing paginate assertion reads the attribute set, update its expectation to include the three new keys — do not remove the keys.

- [x] **Step 5: Commit**

```bash
git add services/atlas-configurations/atlas.com/configurations/templates/resource.go \
        services/atlas-configurations/atlas.com/configurations/templates/resource_reseed_test.go
git commit -m "feat(atlas-configurations): expose drift attributes and POST /templates/{id}/reseed"
```

---

## Task 7: Single parse path — seeder reads the catalog

**Files:**
- Modify: `services/atlas-configurations/atlas.com/configurations/seeder/seeder.go`
- Modify: `services/atlas-configurations/atlas.com/configurations/seeder/seeder_test.go`
- Modify: `services/atlas-configurations/atlas.com/configurations/main.go`

**Interfaces:**
- Consumes: `templates.Catalog`, `templates.CatalogEntry`, `templates.InitShippedCatalog` (Task 2).
- Produces: `func NewSeeder(l logrus.FieldLogger, ctx context.Context, db *gorm.DB, config Config, catalog templates.Catalog) *Seeder`. `ConfigMetadata`, `Seeder.extractMetadata` and `Seeder.discoverFiles` are removed. `Seeder.importTemplate(entry templates.CatalogEntry) string` still returns `"imported"` / `"skipped"` / `"failed"`, and `SeedResult` is unchanged.

- [x] **Step 1: Write the failing test**

In `seeder/seeder_test.go`:

**Delete** these tests — they exercise `discoverFiles` / `extractMetadata`, which no longer exist. Their coverage moved to `templates/shipped_test.go` in Task 2 (`TestLoadCatalogTolerantOfBadFiles`, `TestLoadCatalogRealSeedCorpus`):
`TestDiscoverFiles`, `TestDiscoverFilesSorting`, `TestExtractMetadata`, `TestDiscoverFilesOnlyJson`, `TestExtractMetadataGmsV84`, `TestGmsV84DistinctFromV83`, `TestSeedDataDiscoversBothV83AndV84`.

**Keep** `TestDefaultConfig`, `TestDefaultConfigWithEnvVars`, `TestRunWithSeedingDisabled` (update the `Seeder` literal in the last one only if it fails to compile — it sets fields directly and needs no catalog).

**Add** the regression guard the PRD calls for:

```go
// THE regression guard (PRD acceptance criteria, FR-4.1/FR-4.2): boot-time
// seeding is create-if-absent. Given an existing row and a seed file whose
// content differs, the seeder must report "skipped" and leave the row
// byte-identical. This is what protects "UI edits survive a redeploy" from
// being quietly broken by a later change.
func TestSeederSkipsExistingWithDifferentContent(t *testing.T) {
	db := setupSeederTestDB(t)
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	ctx := context.Background()

	catalog := templates.LoadCatalog(l, filepath.Join("testdata", "templates"))
	entry, ok := catalog.Lookup("TEST", 1, 0)
	if !ok {
		t.Fatalf("TEST 1.0 fixture missing from testdata/templates")
	}

	// Pre-create a row on the same key with DIFFERENT content.
	p := templates.NewProcessor(l, ctx, db)
	existing := entry.Model
	existing.UsesPin = !existing.UsesPin
	id, err := p.Create(existing)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	before, err := readTemplateData(db, id)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}

	s := NewSeeder(l, ctx, db, Config{SeedPath: "testdata", Enabled: true}, catalog)
	result := s.seedTemplates()

	if result.Skipped != 1 || result.Imported != 0 || result.Failed != 0 {
		t.Fatalf("SeedResult = %+v, want {Imported:0 Skipped:1 Failed:0}", result)
	}

	after, err := readTemplateData(db, id)
	if err != nil {
		t.Fatalf("read back (after): %v", err)
	}
	if after != before {
		t.Errorf("the seeder rewrote an existing row:\nbefore: %s\nafter:  %s", before, after)
	}
}

// A key with no existing row is imported.
func TestSeederImportsMissingTemplate(t *testing.T) {
	db := setupSeederTestDB(t)
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	ctx := context.Background()

	catalog := templates.LoadCatalog(l, filepath.Join("testdata", "templates"))
	s := NewSeeder(l, ctx, db, Config{SeedPath: "testdata", Enabled: true}, catalog)

	result := s.seedTemplates()
	if result.Imported != 1 || result.Skipped != 0 || result.Failed != 0 {
		t.Fatalf("SeedResult = %+v, want {Imported:1 Skipped:0 Failed:0}", result)
	}

	got, err := templates.NewProcessor(l, ctx, db).GetByRegionAndVersion("TEST", 1, 0)
	if err != nil {
		t.Fatalf("GetByRegionAndVersion: %v", err)
	}
	if got.Region != "TEST" {
		t.Errorf("Region = %q, want TEST", got.Region)
	}
}
```

And the sqlite harness helpers, at the top of `seeder/seeder_test.go`:

```go
// seederTestEntity mirrors templates.Entity with SQLite-compatible column
// types, the same way templates/processor_test.go's testEntity does. The
// templates package's own harness is unexported, so it is restated here.
type seederTestEntity struct {
	Id           uuid.UUID       `gorm:"type:text;primaryKey"`
	Region       string          `gorm:"not null"`
	MajorVersion uint16          `gorm:"not null"`
	MinorVersion uint16          `gorm:"not null"`
	Data         json.RawMessage `gorm:"type:text;not null"`
}

func (seederTestEntity) TableName() string {
	return "templates"
}

func setupSeederTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}
	if err := db.AutoMigrate(&seederTestEntity{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	return db
}

func readTemplateData(db *gorm.DB, id uuid.UUID) (string, error) {
	var e seederTestEntity
	if err := db.Where("id = ?", id).First(&e).Error; err != nil {
		return "", err
	}
	return string(e.Data), nil
}
```

Imports needed in `seeder/seeder_test.go`: `encoding/json`, `path/filepath`, `github.com/google/uuid`, `gorm.io/driver/sqlite`, `gorm.io/gorm`, `gorm.io/gorm/logger`, `atlas-configurations/templates`.

- [x] **Step 2: Run test to verify it fails**

```bash
cd services/atlas-configurations/atlas.com/configurations
go test ./seeder/ -v
```

Expected: FAIL to compile — `NewSeeder` takes four arguments, not five.

- [x] **Step 3: Rewrite the seeder**

In `seeder/seeder.go`:

- Delete `ConfigMetadata` (lines 39-44), `discoverFiles` (138-163) and `extractMetadata` (165-182).
- Drop the now-unused imports (`encoding/json`, `path/filepath`, `sort`; keep `os` for `DefaultConfig`, keep `errors` for `templateExists`).
- Add the `catalog` field and extend `NewSeeder`:

```go
// Seeder handles importing seed data into the database
type Seeder struct {
	l   logrus.FieldLogger
	ctx context.Context
	db  *gorm.DB
	// catalog is the shipped-template catalog, loaded once in main.go. The
	// seeder no longer reads or parses files itself: templates.LoadCatalog is
	// the single parse path (FR-1.7), so the drift comparison and the boot
	// import can never disagree about what a file contains.
	catalog templates.Catalog
	config  Config
}

// NewSeeder creates a new Seeder instance
func NewSeeder(l logrus.FieldLogger, ctx context.Context, db *gorm.DB, config Config, catalog templates.Catalog) *Seeder {
	return &Seeder{
		l:       l,
		ctx:     ctx,
		db:      db,
		config:  config,
		catalog: catalog,
	}
}
```

- Rewrite `seedTemplates` and `importTemplate`:

```go
// seedTemplates imports every shipped template that does not already exist.
// The outcome strings and SeedResult counters are unchanged.
func (s *Seeder) seedTemplates() SeedResult {
	result := SeedResult{}

	entries := s.catalog.Entries()
	if len(entries) == 0 {
		s.l.WithField("path", filepath.Join(s.config.SeedPath, "templates")).Debug("No template seed files found")
		return result
	}

	s.l.WithField("count", len(entries)).Info("Discovered template seed files")

	for _, entry := range entries {
		switch s.importTemplate(entry) {
		case "imported":
			result.Imported++
		case "skipped":
			result.Skipped++
		case "failed":
			result.Failed++
		}
	}

	return result
}

// importTemplate creates a template if no row exists on its key.
//
// CREATE-IF-ABSENT, DELIBERATELY (FR-4.1). It returns "skipped" for an
// existing key regardless of whether the file's content differs from the
// stored row. Reconciling on boot would silently discard operator edits on
// every redeploy; correcting drift is an explicit operator action
// (POST /configurations/templates/{id}/reseed), never a startup side effect.
func (s *Seeder) importTemplate(entry templates.CatalogEntry) string {
	l := s.l.WithFields(logrus.Fields{
		"file":         entry.FileName,
		"region":       entry.Model.Region,
		"majorVersion": entry.Model.MajorVersion,
		"minorVersion": entry.Model.MinorVersion,
	})

	exists, err := s.templateExists(entry.Model.Region, entry.Model.MajorVersion, entry.Model.MinorVersion)
	if err != nil {
		l.WithError(err).Error("Failed to check template existence")
		return "failed"
	}
	if exists {
		l.Debug("Template already exists, skipping")
		return "skipped"
	}

	id, err := templates.NewProcessor(s.l, s.ctx, s.db).Create(entry.Model)
	if err != nil {
		l.WithError(err).Error("Failed to create template")
		return "failed"
	}

	l.WithField("id", id.String()).Info("Template imported successfully")
	return "imported"
}
```

`filepath` is still needed for the Debug log above — keep that import.

- In `main.go`, load the catalog before the seeder runs and pass it in:

```go
	// Run seed import
	seedConfig := seeder.DefaultConfig()
	l.WithFields(map[string]interface{}{
		"seedPath":    seedConfig.SeedPath,
		"seedEnabled": seedConfig.Enabled,
	}).Info("Seed configuration loaded")

	// The shipped-template catalog is loaded UNCONDITIONALLY, before the
	// seeder and before route registration. SEED_ENABLED gates whether
	// templates are imported, not whether the service knows what ships - an
	// operator who has disabled seeding still needs drift detection and the
	// reset button.
	catalog := templates.InitShippedCatalog(l, filepath.Join(seedConfig.SeedPath, "templates"))

	s := seeder.NewSeeder(l, rcontext.Background(), db, seedConfig, catalog)
	if err := s.Run(); err != nil {
		l.WithError(err).Error("Seed import failed")
	}
```

Add `path/filepath` to `main.go`'s imports.

- [x] **Step 4: Run tests to verify they pass**

```bash
cd services/atlas-configurations/atlas.com/configurations
go build ./... && go test ./... -v
```

Expected: PASS across every package.

- [x] **Step 5: Commit**

```bash
git add services/atlas-configurations/atlas.com/configurations/seeder/seeder.go \
        services/atlas-configurations/atlas.com/configurations/seeder/seeder_test.go \
        services/atlas-configurations/atlas.com/configurations/main.go
git commit -m "refactor(atlas-configurations): seed templates from the shipped catalog"
```

---

## Task 8: Go verification gate

**Files:** none modified — this task only runs the gates.

- [x] **Step 1: Full race-enabled test run**

```bash
cd services/atlas-configurations/atlas.com/configurations
go test -race ./...
```

Expected: `ok` for every package. `InitShippedCatalog`'s `sync.Once` + `sync.RWMutex` are what `-race` is checking here.

- [x] **Step 2: Vet and build**

```bash
cd services/atlas-configurations/atlas.com/configurations
go vet ./... && go build ./...
```

Expected: no output.

- [x] **Step 3: Confirm `go.mod` is untouched**

```bash
git diff --stat -- services/atlas-configurations/atlas.com/configurations/go.mod services/atlas-configurations/atlas.com/configurations/go.sum
```

Expected: empty. If either changed, a non-stdlib dependency crept in — remove it, or `docker buildx bake atlas-configurations` becomes mandatory per CLAUDE.md item 4.

- [x] **Step 4: Repo-root guards**

From the worktree root:

```bash
tools/redis-key-guard.sh
tools/goroutine-guard.sh
tools/lint.sh --check
```

Expected: exit 0 each. `tools/lint.sh --check` needs nvm on PATH — if it false-fails on the atlas-ui leg, run `tools/lint.sh` (fix mode) first and re-check.

- [x] **Step 5: Commit any formatting the linter rewrote**

```bash
git add -A services/atlas-configurations
git diff --cached --quiet || git commit -m "style(atlas-configurations): apply lint formatting"
```

---

## Task 9: UI types and the service call

**Files:**
- Modify: `services/atlas-ui/src/types/models/template.ts:73-98`
- Modify: `services/atlas-ui/src/services/api/templates.service.ts`
- Test: `services/atlas-ui/src/services/api/__tests__/templates.service.test.ts` (create if absent, otherwise append)

**Interfaces:**
- Consumes: `api.post` (`src/lib/api/client.ts:238`), `ServiceOptions` (`src/lib/api/query-params.ts`).
- Produces:
  - `TemplateAttributes` gains `shippedRevision?: string; storedRevision?: string; seedDrift?: boolean`.
  - `templatesService.reseed(id: string, options?: ServiceOptions): Promise<void>`.

- [x] **Step 1: Write the failing test**

Create (or append to) `src/services/api/__tests__/templates.service.test.ts`:

```ts
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { templatesService } from "@/services/api/templates.service";

describe("templatesService.reseed", () => {
  let fetchMock: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    fetchMock = vi.fn();
    vi.stubGlobal("fetch", fetchMock);
  });
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("POSTs to the reseed sub-resource with no body", async () => {
    fetchMock.mockResolvedValue({
      ok: true,
      status: 204,
      json: async () => ({}),
    });

    await templatesService.reseed("abc-123");

    expect(fetchMock).toHaveBeenCalledTimes(1);
    const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(url).toBe("/api/configurations/templates/abc-123/reseed");
    expect(init.method).toBe("POST");
    expect(init.body).toBeUndefined();
  });

  it("rejects when the server returns an error status", async () => {
    fetchMock.mockResolvedValue({
      ok: false,
      status: 409,
      json: async () => ({
        errors: [
          {
            status: "409",
            title: "no shipped template",
            detail: "nothing to reset to",
          },
        ],
      }),
    });

    await expect(templatesService.reseed("abc-123")).rejects.toBeDefined();
  });
});
```

- [x] **Step 2: Run test to verify it fails**

```bash
cd services/atlas-ui
npx vitest run src/services/api/__tests__/templates.service.test.ts
```

Expected: FAIL — `templatesService.reseed is not a function`.

- [x] **Step 3: Add the type fields**

In `src/types/models/template.ts`, inside `TemplateAttributes` (after `minorVersion`):

```ts
  /**
   * SHA-256 of the seed file baked into the RUNNING image for this
   * region/version. Empty string when no such file ships. Computed
   * server-side; ignored on write.
   */
  shippedRevision?: string;
  /** SHA-256 of the persisted template content. Computed server-side. */
  storedRevision?: string;
  /**
   * True when shippedRevision is non-empty and differs from storedRevision.
   * Advisory and image-relative (NFR-4) - during a rolling update two replicas
   * may briefly disagree - so this is never an error state.
   */
  seedDrift?: boolean;
```

All three are optional because fixtures and any older API predate them, and `exactOptionalPropertyTypes` is on: read sites must handle `undefined`.

- [x] **Step 4: Add the service call**

In `src/services/api/templates.service.ts`, next to `delete` (around line 354):

```ts
  /**
   * Resets one template to the configuration shipped in the currently deployed
   * image. Destructive: any edit made through the UI is overwritten.
   *
   * The endpoint returns 204 with no body, so there is nothing to sort or
   * validate on the way back out - the caller invalidates and refetches.
   */
  async reseed(id: string, options?: ServiceOptions): Promise<void> {
    await api.post<void>(`${BASE_PATH}/${id}/reseed`, undefined, options);
  },
```

`api.post` omits the body entirely when `data` is `undefined` (`src/lib/api/client.ts:242`), which is what the test asserts.

- [x] **Step 5: Run tests to verify they pass**

```bash
cd services/atlas-ui
npx vitest run src/services/api/__tests__/templates.service.test.ts
```

Expected: PASS — both tests.

- [x] **Step 6: Commit**

```bash
git add services/atlas-ui/src/types/models/template.ts \
        services/atlas-ui/src/services/api/templates.service.ts \
        services/atlas-ui/src/services/api/__tests__/templates.service.test.ts
git commit -m "feat(atlas-ui): add template reseed service call and drift attributes"
```

---

## Task 10: The `useReseedTemplate` mutation

**Files:**
- Modify: `services/atlas-ui/src/lib/hooks/api/useTemplates.ts`
- Create: `services/atlas-ui/src/lib/hooks/api/__tests__/useReseedTemplate.test.tsx`

**Interfaces:**
- Consumes: `templatesService.reseed` (Task 9), `templateKeys` (`src/lib/hooks/api/useTemplates.ts:31`).
- Produces: `export function useReseedTemplate(): UseMutationResult<void, Error, { id: string }>`. Callers invoke `mutateAsync({ id })`.

- [x] **Step 1: Write the failing test**

Create `src/lib/hooks/api/__tests__/useReseedTemplate.test.tsx`:

```tsx
import { describe, it, expect, vi, beforeEach } from "vitest";
import type { ReactNode } from "react";
import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useReseedTemplate, templateKeys } from "@/lib/hooks/api/useTemplates";
import { templatesService } from "@/services/api/templates.service";

vi.mock("@/services/api/templates.service", () => ({
  templatesService: {
    reseed: vi.fn(),
  },
}));

function makeWrapper(qc: QueryClient) {
  return function Wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={qc}>{children}</QueryClientProvider>;
  };
}

describe("useReseedTemplate", () => {
  beforeEach(() => vi.clearAllMocks());

  it("calls the service and invalidates the detail and list queries", async () => {
    vi.mocked(templatesService.reseed).mockResolvedValue(undefined);
    const qc = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    const invalidate = vi.spyOn(qc, "invalidateQueries");

    const { result } = renderHook(() => useReseedTemplate(), {
      wrapper: makeWrapper(qc),
    });

    await result.current.mutateAsync({ id: "abc-123" });

    expect(templatesService.reseed).toHaveBeenCalledWith("abc-123");
    await waitFor(() => {
      expect(invalidate).toHaveBeenCalledWith({
        queryKey: templateKeys.detail("abc-123"),
      });
      expect(invalidate).toHaveBeenCalledWith({
        queryKey: templateKeys.lists(),
      });
    });
  });

  it("surfaces a rejection and invalidates nothing", async () => {
    vi.mocked(templatesService.reseed).mockRejectedValue(
      new Error("no shipped template"),
    );
    const qc = new QueryClient({
      defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
    });
    const invalidate = vi.spyOn(qc, "invalidateQueries");

    const { result } = renderHook(() => useReseedTemplate(), {
      wrapper: makeWrapper(qc),
    });

    await expect(result.current.mutateAsync({ id: "abc-123" })).rejects.toThrow(
      "no shipped template",
    );
    expect(invalidate).not.toHaveBeenCalled();
  });
});
```

- [x] **Step 2: Run test to verify it fails**

```bash
cd services/atlas-ui
npx vitest run src/lib/hooks/api/__tests__/useReseedTemplate.test.tsx
```

Expected: FAIL — `useReseedTemplate` is not exported.

- [x] **Step 3: Write the hook**

In `src/lib/hooks/api/useTemplates.ts`, after `useDeleteTemplate`:

```ts
/**
 * Resets one template to the configuration shipped in the deployed image.
 *
 * Invalidates on SUCCESS only (FR-5.6): a failed re-seed changed nothing
 * server-side, so refetching would only churn. The detail query is invalidated
 * so the open page re-reads, and the lists query so the drift badge clears
 * without a manual reload.
 */
export function useReseedTemplate(): UseMutationResult<
  void,
  Error,
  { id: string }
> {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id }) => templatesService.reseed(id),
    onSuccess: (_data, { id }) => {
      queryClient.invalidateQueries({ queryKey: templateKeys.detail(id) });
      queryClient.invalidateQueries({ queryKey: templateKeys.lists() });
    },
  });
}
```

- [x] **Step 4: Run tests to verify they pass**

```bash
cd services/atlas-ui
npx vitest run src/lib/hooks/api/__tests__/useReseedTemplate.test.tsx
```

Expected: PASS — both tests.

- [x] **Step 5: Commit**

```bash
git add services/atlas-ui/src/lib/hooks/api/useTemplates.ts \
        services/atlas-ui/src/lib/hooks/api/__tests__/useReseedTemplate.test.tsx
git commit -m "feat(atlas-ui): add useReseedTemplate mutation"
```

---

## Task 11: The "Reset to shipped defaults" button

**Files:**
- Create: `services/atlas-ui/src/components/features/templates/TemplateReseedButton.tsx`
- Modify: `services/atlas-ui/src/components/features/templates/TemplateDetailLayout.tsx:37`
- Create: `services/atlas-ui/src/components/features/templates/__tests__/TemplateReseedButton.test.tsx`

**Interfaces:**
- Consumes: `useTemplate` (`src/lib/hooks/api/useTemplates.ts:113`), `useReseedTemplate` (Task 10), `Button`, `AlertDialog*` (`src/components/ui/alert-dialog.tsx`), `Tooltip*` (`src/components/ui/tooltip.tsx`), `toast` from `sonner`, `createErrorFromUnknown` (`src/types/api/errors.ts`), `buttonVariants` + `cn`.
- Produces: `export function TemplateReseedButton({ id }: { id: string | undefined })`.

- [x] **Step 1: Write the failing test**

Create `src/components/features/templates/__tests__/TemplateReseedButton.test.tsx`:

```tsx
import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { TemplateReseedButton } from "@/components/features/templates/TemplateReseedButton";
import { useTemplate, useReseedTemplate } from "@/lib/hooks/api/useTemplates";

vi.mock("@/lib/hooks/api/useTemplates", () => ({
  useTemplate: vi.fn(),
  useReseedTemplate: vi.fn(),
}));

const toastError = vi.fn();
const toastSuccess = vi.fn();
vi.mock("sonner", () => ({
  toast: {
    error: (...args: unknown[]) => toastError(...args),
    success: (...args: unknown[]) => toastSuccess(...args),
  },
}));

function template(attrs: Record<string, unknown>) {
  return {
    id: "abc-123",
    attributes: {
      region: "GMS",
      majorVersion: 83,
      minorVersion: 1,
      ...attrs,
    },
  };
}

function mockHooks({
  data,
  mutateAsync,
}: {
  data: unknown;
  mutateAsync?: ReturnType<typeof vi.fn>;
}) {
  vi.mocked(useTemplate).mockReturnValue({ data } as never);
  vi.mocked(useReseedTemplate).mockReturnValue({
    mutateAsync: mutateAsync ?? vi.fn().mockResolvedValue(undefined),
  } as never);
}

describe("TemplateReseedButton", () => {
  beforeEach(() => vi.clearAllMocks());

  it("is disabled when the template ships no seed file", () => {
    mockHooks({ data: template({ shippedRevision: "" }) });
    render(<TemplateReseedButton id="abc-123" />);
    expect(
      screen.getByRole("button", { name: /reset to shipped defaults/i }),
    ).toBeDisabled();
  });

  it("is disabled when shippedRevision is absent", () => {
    mockHooks({ data: template({}) });
    render(<TemplateReseedButton id="abc-123" />);
    expect(
      screen.getByRole("button", { name: /reset to shipped defaults/i }),
    ).toBeDisabled();
  });

  it("is enabled when a seed file ships", () => {
    mockHooks({ data: template({ shippedRevision: "aa" }) });
    render(<TemplateReseedButton id="abc-123" />);
    expect(
      screen.getByRole("button", { name: /reset to shipped defaults/i }),
    ).toBeEnabled();
  });

  it("issues no request when the dialog is dismissed", async () => {
    const mutateAsync = vi.fn().mockResolvedValue(undefined);
    mockHooks({ data: template({ shippedRevision: "aa" }), mutateAsync });
    const user = userEvent.setup();

    render(<TemplateReseedButton id="abc-123" />);
    await user.click(
      screen.getByRole("button", { name: /reset to shipped defaults/i }),
    );
    await screen.findByRole("alertdialog");
    await user.click(screen.getByRole("button", { name: /^cancel$/i }));

    expect(mutateAsync).not.toHaveBeenCalled();
  });

  it("posts the re-seed on confirmation", async () => {
    const mutateAsync = vi.fn().mockResolvedValue(undefined);
    mockHooks({ data: template({ shippedRevision: "aa" }), mutateAsync });
    const user = userEvent.setup();

    render(<TemplateReseedButton id="abc-123" />);
    await user.click(
      screen.getByRole("button", { name: /reset to shipped defaults/i }),
    );
    await screen.findByRole("alertdialog");
    await user.click(screen.getByRole("button", { name: /^reset template$/i }));

    await waitFor(() =>
      expect(mutateAsync).toHaveBeenCalledWith({ id: "abc-123" }),
    );
    await waitFor(() => expect(toastSuccess).toHaveBeenCalled());
  });

  it("surfaces an error toast when the re-seed fails", async () => {
    const mutateAsync = vi.fn().mockRejectedValue(new Error("409 conflict"));
    mockHooks({ data: template({ shippedRevision: "aa" }), mutateAsync });
    const user = userEvent.setup();

    render(<TemplateReseedButton id="abc-123" />);
    await user.click(
      screen.getByRole("button", { name: /reset to shipped defaults/i }),
    );
    await screen.findByRole("alertdialog");
    await user.click(screen.getByRole("button", { name: /^reset template$/i }));

    await waitFor(() => expect(toastError).toHaveBeenCalled());
    expect(toastSuccess).not.toHaveBeenCalled();
  });

  it("names the image comparison in the confirm dialog", async () => {
    mockHooks({ data: template({ shippedRevision: "aa" }) });
    const user = userEvent.setup();

    render(<TemplateReseedButton id="abc-123" />);
    await user.click(
      screen.getByRole("button", { name: /reset to shipped defaults/i }),
    );
    const dialog = await screen.findByRole("alertdialog");
    expect(dialog).toHaveTextContent(/shipped in this image/i);
    expect(dialog).toHaveTextContent(/edits made through the UI will be lost/i);
  });
});
```

- [x] **Step 2: Run test to verify it fails**

```bash
cd services/atlas-ui
npx vitest run src/components/features/templates/__tests__/TemplateReseedButton.test.tsx
```

Expected: FAIL — cannot resolve `TemplateReseedButton`.

- [x] **Step 3: Write the component**

Create `src/components/features/templates/TemplateReseedButton.tsx`:

```tsx
import { useState } from "react";
import { Loader2, RotateCcw } from "lucide-react";
import { toast } from "sonner";

import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Button, buttonVariants } from "@/components/ui/button";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";
import { useReseedTemplate, useTemplate } from "@/lib/hooks/api/useTemplates";
import { createErrorFromUnknown } from "@/types/api/errors";

export interface TemplateReseedButtonProps {
  id: string | undefined;
}

/**
 * Resets the viewed template to the configuration shipped in the currently
 * deployed image.
 *
 * Lives in the detail LAYOUT header beside ConfigExportButton, for the same
 * reason that component does: it acts on the whole template document, not on
 * the sub-tab being viewed, so putting it in the layout gives it every sub-tab
 * with no per-page wiring.
 *
 * Disabled when the template's shippedRevision is empty or absent - this image
 * ships no file for its region/version, so there is nothing to reset to and the
 * server would 409 (FR-5.4).
 */
export function TemplateReseedButton({ id }: TemplateReseedButtonProps) {
  const [open, setOpen] = useState(false);
  const [isReseeding, setIsReseeding] = useState(false);
  const query = useTemplate(id ?? "");
  const reseed = useReseedTemplate();

  const shippedRevision = query.data?.attributes.shippedRevision;
  const shipsSeedFile = Boolean(shippedRevision);
  const disabled = !id || !query.data || !shipsSeedFile;

  const onConfirm = async () => {
    if (!id) return;
    setIsReseeding(true);
    try {
      await reseed.mutateAsync({ id });
      toast.success("Template reset to shipped defaults");
      setOpen(false);
    } catch (e) {
      // Route through createErrorFromUnknown so the server's JSON:API error
      // detail survives into the toast rather than a generic string. Same
      // convention as ConfigExportButton and every other async-catch site
      // under components/features/. The dialog stays open and the displayed
      // template is untouched (FR-5.7).
      toast.error(createErrorFromUnknown(e, "Reset failed").message);
    } finally {
      setIsReseeding(false);
    }
  };

  const trigger = (
    <Button
      type="button"
      variant="outline"
      size="sm"
      disabled={disabled}
      onClick={() => setOpen(true)}
    >
      <RotateCcw className="h-4 w-4" aria-hidden="true" />
      Reset to shipped defaults
    </Button>
  );

  return (
    <>
      {/*
        The Tooltip root is ALWAYS mounted and only its content is gated -
        React reconciles by element type at a position, so swapping between a
        bare Button and a Tooltip-wrapped one would remount the button's DOM
        node and silently drop focus. Same reasoning as ConfigExportButton.
      */}
      <Tooltip>
        <TooltipTrigger asChild>{trigger}</TooltipTrigger>
        {disabled ? (
          <TooltipContent>
            This image ships no seed file for this template&apos;s region and
            version, so there is nothing to reset to.
          </TooltipContent>
        ) : null}
      </Tooltip>

      <AlertDialog open={open} onOpenChange={setOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Reset to shipped defaults?</AlertDialogTitle>
            <AlertDialogDescription>
              This template will be overwritten with the version shipped in this
              image. Edits made through the UI will be lost. The template&apos;s
              id, region and version are unchanged.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            {/*
              Cancel renders FIRST so it holds the dialog's default focus - the
              destructive action must never be what Enter triggers (FR-5.5).
            */}
            <AlertDialogCancel disabled={isReseeding}>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={(e) => {
                e.preventDefault();
                void onConfirm();
              }}
              disabled={isReseeding}
              className={cn(buttonVariants({ variant: "destructive" }))}
            >
              {isReseeding ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  Resetting...
                </>
              ) : (
                "Reset Template"
              )}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  );
}
```

- [x] **Step 4: Mount it in the layout**

In `src/components/features/templates/TemplateDetailLayout.tsx`, add the import:

```tsx
import { TemplateReseedButton } from "@/components/features/templates/TemplateReseedButton";
```

and replace the single-child header action with a flex row:

```tsx
          <div className="flex items-center gap-2">
            <TemplateReseedButton id={id} />
            <ConfigExportButton kind="template" id={id} />
          </div>
```

- [x] **Step 5: Run tests to verify they pass**

```bash
cd services/atlas-ui
npx vitest run src/components/features/templates/__tests__/
```

Expected: PASS — all seven new tests plus the existing `TemplateDetailLayout.test.tsx`. If the layout test asserts on the header's child structure, update its expectation to include the new button rather than removing the button.

- [x] **Step 6: Commit**

```bash
git add services/atlas-ui/src/components/features/templates/
git commit -m "feat(atlas-ui): add reset-to-shipped-defaults action to template detail"
```

---

## Task 12: The drift badge on the templates list

**Files:**
- Modify: `services/atlas-ui/src/pages/templates-columns.tsx`
- Create: `services/atlas-ui/src/pages/__tests__/templates-columns.test.tsx`

**Interfaces:**
- Consumes: `Badge` (`src/components/ui/badge.tsx`), `Tooltip*`, `DataTableColumnDef`, `Template`.
- Produces: a new column with `id: "seedDrift"` in the array `getColumns` returns, placed after the `attributes.minorVersion` column and before `actions`.

- [x] **Step 1: Write the failing test**

Create `src/pages/__tests__/templates-columns.test.tsx`:

```tsx
import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { getColumns } from "@/pages/templates-columns";
import type { Template } from "@/types/models/template";

function driftCell(attributes: Partial<Template["attributes"]>) {
  const columns = getColumns({});
  const column = columns.find((c) => c.id === "seedDrift");
  if (!column?.cell || typeof column.cell !== "function") {
    throw new Error("seedDrift column is missing or has no cell renderer");
  }
  const row = {
    original: { id: "abc-123", attributes } as Template,
  };
  return column.cell({ row } as never);
}

describe("templates-columns seedDrift", () => {
  it("renders the badge when the template has drifted", () => {
    render(<MemoryRouter>{driftCell({ seedDrift: true })}</MemoryRouter>);
    expect(screen.getByText("Differs from image")).toBeInTheDocument();
  });

  it("renders nothing when the template has not drifted", () => {
    const { container } = render(
      <MemoryRouter>{driftCell({ seedDrift: false })}</MemoryRouter>,
    );
    expect(container).not.toHaveTextContent("Differs from image");
  });

  it("renders nothing when no seed file ships", () => {
    const { container } = render(
      <MemoryRouter>
        {driftCell({ shippedRevision: "", seedDrift: false })}
      </MemoryRouter>,
    );
    expect(container).not.toHaveTextContent("Differs from image");
  });

  it("renders nothing when the attribute is absent", () => {
    const { container } = render(<MemoryRouter>{driftCell({})}</MemoryRouter>);
    expect(container).not.toHaveTextContent("Differs from image");
  });
});
```

- [x] **Step 2: Run test to verify it fails**

```bash
cd services/atlas-ui
npx vitest run src/pages/__tests__/templates-columns.test.tsx
```

Expected: FAIL — "seedDrift column is missing".

- [x] **Step 3: Add the column**

In `src/pages/templates-columns.tsx`, add the imports:

```tsx
import { Badge } from "@/components/ui/badge";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
```

and insert this column object between the `attributes.minorVersion` column and the `actions` column:

```tsx
  {
    id: "seedDrift",
    header: "Seed",
    cell: ({ row }) => {
      // Strictly `=== true`: the attribute is optional (an older API or a
      // fixture may omit it) and `undefined` must read as "no badge", never as
      // truthy-ish. FR-5.2 falls out of the server contract - seedDrift is
      // false whenever shippedRevision is empty.
      if (row.original.attributes.seedDrift !== true) return null;
      return (
        <Tooltip>
          <TooltipTrigger asChild>
            {/*
              `secondary`, not `destructive`: the flag is advisory and
              image-relative (NFR-4). A template an operator edited on purpose
              is not in an error state.
            */}
            <Badge variant="secondary">Differs from image</Badge>
          </TooltipTrigger>
          <TooltipContent>
            Differs from the configuration shipped in this image
          </TooltipContent>
        </Tooltip>
      );
    },
  },
```

- [x] **Step 4: Run tests to verify they pass**

```bash
cd services/atlas-ui
npx vitest run src/pages/__tests__/templates-columns.test.tsx src/pages/__tests__/TemplatesPage.test.tsx
```

Expected: PASS — the four new tests plus the existing page test. If `TemplatesPage.test.tsx` asserts a column count, update the expected number.

- [x] **Step 5: Commit**

```bash
git add services/atlas-ui/src/pages/templates-columns.tsx \
        services/atlas-ui/src/pages/__tests__/templates-columns.test.tsx
git commit -m "feat(atlas-ui): show a seed-drift badge on the templates list"
```

---

## Task 13: Keep the computed keys out of the config export

**Files:**
- Modify: `services/atlas-ui/src/lib/utils/config-export.ts`
- Modify: `services/atlas-ui/src/lib/utils/__tests__/config-export.test.ts`

**Interfaces:**
- Consumes: nothing new.
- Produces: `toConfigExportPayload` now deletes `shippedRevision`, `storedRevision` and `seedDrift` from its output. The signature is unchanged.

- [x] **Step 1: Write the failing test**

Append to `src/lib/utils/__tests__/config-export.test.ts`:

```ts
describe("toConfigExportPayload computed attributes", () => {
  it("strips the server-computed drift keys", () => {
    const out = toConfigExportPayload(
      fixture({
        shippedRevision: "aa".repeat(32),
        storedRevision: "bb".repeat(32),
        seedDrift: true,
      }) as never,
    ) as Record<string, unknown>;

    expect(out).not.toHaveProperty("shippedRevision");
    expect(out).not.toHaveProperty("storedRevision");
    expect(out).not.toHaveProperty("seedDrift");
  });

  it("leaves the configured document intact", () => {
    const out = toConfigExportPayload(
      fixture({ seedDrift: true }) as never,
    ) as Record<string, unknown>;

    expect(out.region).toBe("GMS");
    expect(out.majorVersion).toBe(83);
    expect(out.minorVersion).toBe(1);
    expect(out).toHaveProperty("socket");
    expect(out).toHaveProperty("characters");
  });
});
```

- [x] **Step 2: Run test to verify it fails**

```bash
cd services/atlas-ui
npx vitest run src/lib/utils/__tests__/config-export.test.ts
```

Expected: FAIL — `shippedRevision` is present in the output.

- [x] **Step 3: Strip the keys**

In `src/lib/utils/config-export.ts`, inside `toConfigExportPayload`, immediately after the `out` object is built and before the `npcs` / `worlds` normalization:

```ts
  // The ONLY place this module's "everything else is passed through untouched,
  // so we never track a key list" principle is knowingly broken. These three
  // are COMPUTED by atlas-configurations (task-201), not configured: they are
  // not part of the document's shape at all, and the exported file exists to
  // be promoted into seed-data/templates/. A committed seed file carrying a
  // stale hash of itself is exactly the noise task-201 exists to remove. The
  // server drops them on parse either way, so this is hygiene, not a fix.
  delete out.shippedRevision;
  delete out.storedRevision;
  delete out.seedDrift;
```

- [x] **Step 4: Run tests to verify they pass**

```bash
cd services/atlas-ui
npx vitest run src/lib/utils/__tests__/config-export.test.ts
```

Expected: PASS — the two new tests plus every existing one.

- [x] **Step 5: Commit**

```bash
git add services/atlas-ui/src/lib/utils/config-export.ts \
        services/atlas-ui/src/lib/utils/__tests__/config-export.test.ts
git commit -m "fix(atlas-ui): keep computed drift keys out of the config export"
```

---

## Task 14: Documentation and the full verification sweep

**Files:**
- Modify: `docs/packets/TEMPLATE_CONVENTIONS.md`

- [x] **Step 1: Record the presets-carry-ids condition**

Design §7 established empirically that no shipped template can be perturbed by the preset validator: six seed files carry presets and every preset already has an id (`templates/characters/preset/validator.go:37-41` assigns one only when `Id == ""`), and the other five carry none. That is a property of the current corpus, not an invariant.

Append to `docs/packets/TEMPLATE_CONVENTIONS.md` (locate the file first with `Glob`; if the heading structure differs, place this under whichever section covers `characters.presets`):

```markdown
## Character presets must carry ids

Every entry in a seed template's `characters.presets` array must have a
non-empty `id`. The preset validator assigns a UUID only to presets that lack
one (`templates/characters/preset/validator.go`), and it runs on the PATCH
path - so an id-less preset in a seed file means the stored row diverges from
the file the moment the template is edited through the UI, and the "Differs
from image" badge (task-201) lights up for a reason unrelated to what the
operator changed.

Not machine-checked: the consequence is a spuriously-lit badge on one
template, not a gameplay failure. As of task-201, all eleven shipped templates
satisfy it - six carry presets, all with ids; five carry none.
```

- [x] **Step 2: Full Go verification**

From the worktree root:

```bash
cd services/atlas-configurations/atlas.com/configurations && \
  go test -race ./... && go vet ./... && go build ./...
```

Expected: `ok` per package, no vet output, no build output.

- [x] **Step 3: Full UI verification**

```bash
cd services/atlas-ui
npx vitest run
npm run build
```

Expected: all vitest suites pass; the build succeeds (it type-checks tests, so a `exactOptionalPropertyTypes` violation in the new optional attributes surfaces here and nowhere else).

- [x] **Step 4: Repo-root guards**

From the worktree root:

```bash
tools/redis-key-guard.sh
tools/goroutine-guard.sh
tools/lint.sh --check
git diff --stat -- services/atlas-configurations/atlas.com/configurations/go.mod
```

Expected: exit 0 for the three guards; empty `go.mod` diff (so `docker buildx bake atlas-configurations` is not required — if the diff is non-empty, run the bake and make it pass before proceeding).

No template seed file under `services/atlas-configurations/seed-data/templates/` is modified by this task, so the four template guards (opcode-order, duplicate-binding, movement-types) do not apply. Confirm with:

```bash
git diff --name-only main... -- services/atlas-configurations/seed-data/
```

Expected: empty.

- [x] **Step 5: Commit the docs**

```bash
git add docs/packets/TEMPLATE_CONVENTIONS.md
git commit -m "docs: record the presets-carry-ids condition for seed templates"
```

- [x] **Step 6: Code review before PR**

Per CLAUDE.md ("Code Review Before PR"), invoke `superpowers:requesting-code-review`. It dispatches `plan-adherence-reviewer`, `backend-guidelines-reviewer` (Go changed) and `frontend-guidelines-reviewer` (TS changed); findings land in `docs/tasks/task-201-template-reseed-trigger/audit.md`. Pin the reviewer subagents to a cheaper model per the project's model preference. Do not open the PR until the audit is addressed.

- [ ] **Step 7: PR description must carry the remediation note**

The PR body must state, verbatim in substance:

> Shipping this does **not** repair the GMS 87.1, GMS 92.1, GMS 95.1 and JMS 185.1 template rows in `atlas-main` — all four are missing `QuestActionHandle` and will remain so after this merges. It makes them **visible** (they will report `seedDrift: true` with a "Differs from image" badge) and makes the repair a button press ("Reset to shipped defaults" on each template's detail page). Remediation is a manual post-deploy step.

---

## Self-Review

**Spec coverage.** Every FR and NFR maps to a task:

| Requirement | Task |
|---|---|
| FR-1.1 catalog loads all `*.json` | 2 |
| FR-1.2 singleton, `sync.Once` + `sync.RWMutex` | 2 (`InitShippedCatalog` / `ShippedCatalog`, `TestInitShippedCatalogIsIdempotent`) |
| FR-1.3 entry holds file name, model, revision | 2 (`CatalogEntry`) |
| FR-1.4 shipped revision definition | 1 + 2 (`Revision` called by `LoadCatalog`) |
| FR-1.5 parse failure logged, omitted, non-fatal | 2 (`TestLoadCatalogTolerantOfBadFiles`) |
| FR-1.6 duplicate key, first-in-sort-order wins | 2 (`TestLoadCatalogDuplicateKeyFirstWins`) |
| FR-1.7 seeder reads the catalog, one parse path | 7 |
| FR-2.1 stored revision via `Make`, not raw bytes | 4 (`makeView` maps over `ByIdProvider`, which maps `Make`) |
| FR-2.2 `Id` does not participate | 1 (`TestRevisionIgnoresId`) |
| FR-2.3 drift = entry exists && revisions differ | 4 (`TestDriftDetectedAfterMutation`) |
| FR-2.4 no entry ⇒ not drifted, empty shipped revision | 4 (`TestNoCatalogEntryIsNotDrift`, `TestUnwiredProcessorReportsNoShippedFile`) |
| FR-2.5 computed on read, never persisted | 4 (`TestComputedAttributesAreNotPersisted`) |
| FR-3.1 POST reseed replaces content | 5 + 6 |
| FR-3.2 UUID preserved, update not delete-recreate | 5 (`TestReseedRestoresShippedContent`) |
| FR-3.3 region/version unchanged | 5 (same test; `update` is passed the entity's columns) |
| FR-3.4 `Create` semantics, not `UpdateById` | 3 + 5 (`TestReseedProducesSameBytesAsFreshCreate`) |
| FR-3.5 single transaction | 5 (`database.ExecuteTransaction`) |
| FR-3.6 204 No Content | 6 (`TestReseedReturnsNoContent`) |
| FR-3.7 idempotent | 5 (`TestReseedIsIdempotent`) |
| FR-3.8 no Kafka/outbox | 5 (nothing enqueued; noted in the task) |
| FR-4.1 / FR-4.2 boot stays create-if-absent | 7 (`TestSeederSkipsExistingWithDifferentContent`) |
| FR-5.1 drift badge on the list | 12 |
| FR-5.2 no badge when no seed file ships | 12 |
| FR-5.3 detail-page reset action | 11 |
| FR-5.4 disabled + tooltip when nothing ships | 11 |
| FR-5.5 confirm dialog, destructive not default-focused | 11 |
| FR-5.6 invalidate on success | 10 |
| FR-5.7 failure surfaces an error, template unchanged | 11 |
| §5.1 three read-only attributes, ignored on write | 4 + 6 |
| §5.2 404 / 409 / 400 / 500 status table | 6 |
| NFR-1 startup cost, not rebuilt per request | 2 (singleton) + 7 (one load in `main.go`) |
| NFR-2 read cost, no cache | 4 (`AllViewProvider` keeps `ParallelMap`) |
| NFR-3 INFO log with before/after revisions | 5 |
| NFR-4 image-relative, not an error state | badge variant `secondary`, wording — Global Constraints + 12 |
| NFR-5 not tenant-scoped | 6 (no tenant header parsing on the route) |
| NFR-6 same exposure as PATCH/DELETE | 6 (registered on the same subrouter) |
| Design §6 export leak | 13 |
| Design §7 presets-carry-ids note | 14 |

**Known deviations from the PRD, both from the design and both deliberate:**

1. **PRD §7 puts the catalog under `seeder/`.** It lives in `templates/` instead (design D1) — `seeder` already imports `templates`, and the catalog's payload is `templates.RestModel`, so the PRD's placement closes an import cycle.
2. **FR-2.2 says the revision relies on `Id`'s `json:"-"` tag.** `Revision` clears `Id` explicitly (design §5) — strictly stronger, and it keeps `TestRevisionIgnoresId` meaningful rather than tautological.

A third item the design flagged and this plan acts on: the design's §8 claim that "existing `seeder_test.go` cases keep passing" is wrong for seven of them — they call `discoverFiles` / `extractMetadata`, which D5 deletes. Task 7 deletes those tests and Task 2 carries their coverage forward against `LoadCatalog`, including the GMS 83.1-vs-84.1 distinctness assertion.

**Placeholder scan.** No "TBD", no "add error handling", no "similar to Task N". Two steps carry an explicit fallback instruction rather than a placeholder: Task 3 Step 1 (if `socket.Validate` does not reject a validator-less handler, pick another invalid document — the assertion is what matters) and Task 4 Step 4 (if `model.MapPaged`'s signature does not compose as written, use whichever combinator does). Both name the exact file to read and the invariant to preserve.

**Type consistency.** `Revision(RestModel) (string, error)`, `Catalog.Lookup(string, uint16, uint16) (CatalogEntry, bool)`, `Catalog.Entries() []CatalogEntry`, `Catalog.Len() int`, `LoadCatalog(logrus.FieldLogger, string) Catalog`, `canonicalBytes(RestModel) (json.RawMessage, error)`, `ViewRestModel`, `ReseedById(uuid.UUID) error`, `writeJSONAPIError(http.ResponseWriter, int, string, string)`, `templatesService.reseed(string, ServiceOptions?) Promise<void>`, `useReseedTemplate()` taking `{ id }` — each name and signature is used identically everywhere it appears.
