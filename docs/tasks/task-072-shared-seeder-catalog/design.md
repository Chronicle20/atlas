# Shared Seeder Library and GitOps Catalog — Design Document

Version: v1
Status: Draft
Created: 2026-05-20
Companion PRD: `prd.md`

---

## 1. Purpose and Scope

This document records the architectural decisions for extracting the duplicated seed-orchestration pattern from eight Atlas services into `libs/atlas-seeder`, moving bundled catalog data to a top-level `deploy/seed/` GitOps tree, and adding the per-tenant `seed_state` revision metadata that enables future drift reconciliation.

The PRD (`prd.md`) is approved. This design covers **how** the work is structured, not **what** ships. Where the PRD already specifies a concrete interface (e.g., the `Subdomain[J, M]` generic shape, the `CATALOG_REVISION` placement, the `seed_state` schema), this document treats that as a fixed input and focuses on the architectural questions the PRD deliberately left open: library internals, error semantics, the migration sequencing, testing strategy, and the k8s/compose plumbing.

Two non-goals are reaffirmed from the PRD: no reconciler Job lands in this task (only the metadata that enables it), and `atlas-configurations` / `atlas-tenants` are not migrated (their seed shapes differ enough that bolting them on would distort the library).

---

## 2. Architecture Overview

The design has four layers that compose top-down:

```
┌─────────────────────────────────────────────────────────────────┐
│ Service main.go                                                 │
│  - constructs Subdomain[J, M] instances                         │
│  - assembles Group(s)                                           │
│  - calls seeder.RegisterRoutes(...)                             │
│  - AutoMigrate(&seeder.SeedState{})                             │
└──────────────────────────┬──────────────────────────────────────┘
                           │
┌──────────────────────────▼──────────────────────────────────────┐
│ libs/atlas-seeder (new module)                                  │
│  - RegisterRoutes: HTTP wiring (POST /seed, GET /seed/status)   │
│  - Seed:           per-group orchestrator (errgroup fan-out)    │
│  - Status:         parallel Count + seed_state lookup           │
│  - SubdomainAny:   type-erased wrapper over Subdomain[J, M]     │
│  - SeedState:      GORM entity for revision metadata            │
│  - CatalogSource:  abstraction over file lookup                 │
└──────────────────────────┬──────────────────────────────────────┘
                           │
┌──────────────────────────▼──────────────────────────────────────┐
│ CatalogSource (FilesystemCatalogSource v1)                      │
│  - tenant → resolved root: $SEED_CATALOG_ROOT/<region>/<v>/     │
│  - Walk / Open / Revision per root                              │
└──────────────────────────┬──────────────────────────────────────┘
                           │
┌──────────────────────────▼──────────────────────────────────────┐
│ Mount substrate                                                  │
│  - k8s:     git-sync sidecar → emptyDir → /var/run/seed-catalog │
│  - compose: ./deploy/seed bind-mounted → /var/run/seed-catalog  │
│  - dev:     repo-relative deploy/seed                           │
└─────────────────────────────────────────────────────────────────┘
```

The library is the only piece that needs unit tests against fixture catalogs; the mount substrate is plain Kubernetes/Compose YAML and is validated by `kubectl --dry-run` and `docker compose config`.

---

## 3. The Library: `libs/atlas-seeder`

### 3.1 Module shape

Standalone Go module at `libs/atlas-seeder/`:

```
libs/atlas-seeder/
├── go.mod                            # module Chronicle20/atlas/libs/atlas-seeder
├── go.sum
├── README.md
├── seeder.go                         # public API: Group, RegisterRoutes
├── subdomain.go                      # Subdomain[J, M], SubdomainAny, adapter
├── catalog.go                        # CatalogSource, FilesystemCatalogSource
├── state.go                          # SeedState entity + repo helpers
├── seed.go                           # Seed(...) orchestrator
├── status.go                         # Status(...) reader
├── result.go                         # Result, Counts, Status DTOs
├── handlers.go                       # POST /seed, GET /seed/status http.HandlerFuncs
├── metrics.go                        # Prometheus counter/histogram registration
├── jsonapi.go                        # envelope parse + id-vs-filename check
├── testdata/                         # JSON:API fixture catalogs
│   ├── good/...
│   └── bad/...
└── *_test.go
```

### 3.2 Public API (exact signatures)

```go
package seeder

// Subdomain declares one tenant-scoped catalog dataset.
//
//	J = the JSON:API attributes shape parsed from a catalog file
//	M = the GORM model shape persisted to the database
type Subdomain[J any, M any] interface {
    Name() string                                              // unique within Group
    Path() string                                              // subdir under catalog root (e.g. "drops/monsters")
    EntityIDPattern() *regexp.Regexp                           // capture group 1 = entity id
    Type() string                                              // expected JSON:API data.type
    DeleteAllForTenant(db *gorm.DB) (int64, error)
    Decode(payload []byte) (J, error)                          // decode attributes object
    Build(t tenant.Model, entityId string, j J) ([]M, error)   // 1 file may fan out to N rows
    BulkCreate(db *gorm.DB, models []M) error
    Count(db *gorm.DB) (count int64, mostRecentUpdate *time.Time, err error)
}

// SubdomainAny is the type-erased form Group holds. Services don't construct
// these directly; they wrap a typed Subdomain via AdaptSubdomain.
type SubdomainAny interface { /* internal */ }

func AdaptSubdomain[J any, M any](s Subdomain[J, M]) SubdomainAny

// Group declares one (POST /<prefix>/seed, GET /<prefix>/seed/status) pair.
type Group struct {
    Name       string         // stored as seed_state.group_name; e.g. "drops", "npc-conversations:npc"
    URLPrefix  string         // e.g. "/drops" → routes POST /drops/seed
    Subdomains []SubdomainAny
}

// CatalogSource abstracts where catalog files live. The v1 impl is
// FilesystemCatalogSource, rooted at SEED_CATALOG_ROOT and resolving
// per-request via tenant.Region/Major/Minor.
type CatalogSource interface {
    Roots(t tenant.Model) ([]string, error)
    Revision(root string) (string, error)
    Open(root, relPath string) (io.ReadCloser, error)
    Walk(root, relPath string) ([]string, error) // relative file paths under relPath
}

func NewFilesystemCatalogSource(rootEnv string, fallbackRoot string) CatalogSource

// RegisterRoutes wires POST <prefix>/seed and GET <prefix>/seed/status.
// One call per Group.
func RegisterRoutes(
    router *mux.Router,
    db *gorm.DB,
    si jsonapi.ServerInformation,
    src CatalogSource,
    g Group,
)

// Seed orchestrates per-subdomain delete-then-bulk-insert in parallel.
// Persists one row to seed_state on completion (even on partial failure).
func Seed(ctx context.Context, db *gorm.DB, src CatalogSource, g Group) (Result, error)

// Status returns per-subdomain counts + catalog revisions for the tenant.
func Status(ctx context.Context, db *gorm.DB, src CatalogSource, g Group) (Status, error)

// Result is the seeding outcome serialized to seed_state.result_summary.
type Result struct {
    GroupName        string                       `json:"groupName"`
    CatalogRevision  string                       `json:"catalogRevision"`
    Subdomains       map[string]SubdomainCounts   `json:"subdomains"` // key = Subdomain.Name()
    StartedAt        time.Time                    `json:"startedAt"`
    CompletedAt      time.Time                    `json:"completedAt"`
}

type SubdomainCounts struct {
    Deleted int64    `json:"deleted"`
    Created int64    `json:"created"`
    Failed  int64    `json:"failed"`
    Errors  []string `json:"errors,omitempty"` // capped at 100 entries
}

type Status struct {
    GroupName            string                       `json:"groupName"`
    Subdomains           map[string]SubdomainStatus   `json:"subdomains"`
    UpdatedAt            *time.Time                   `json:"updatedAt"`            // max across subdomains
    CatalogRevision      string                       `json:"catalogRevision"`      // pod-mounted (always present)
    TenantSeededRevision *string                      `json:"tenantSeededRevision"` // null if never seeded
    TenantSeededAt       *time.Time                   `json:"tenantSeededAt"`       // null if never seeded
}

type SubdomainStatus struct {
    Count     int64      `json:"count"`
    UpdatedAt *time.Time `json:"updatedAt"`
}
```

### 3.3 Type erasure rationale

A `Group` holds heterogeneously-typed subdomains (`drops` has `MonsterDrop`, `ContinentDrop`, `ReactorDrop` with different J/M types). Go generics can't yet express "slice of subdomains with different type parameters." The chosen pattern is the standard Go workaround:

- `Subdomain[J, M]` is the type-safe public interface services implement.
- `AdaptSubdomain[J, M](s)` returns a `SubdomainAny` (private interface or concrete struct) that captures the type parameters in closures over `s`.
- The library's orchestrator works in terms of `SubdomainAny` and never sees J/M directly.

This keeps the typed API ergonomic for services while letting the library treat a group uniformly. The adapter is a ~30-line file.

### 3.4 `Seed` orchestration

```
Seed(ctx, db, src, g):
  1. t := tenant.MustFromContext(ctx)
  2. roots, _ := src.Roots(t)
  3. rev, _ := src.Revision(roots[0])     // first root = primary revision
  4. result := Result{GroupName, CatalogRevision: rev, Subdomains: {}, StartedAt: now}
  5. errgroup.WithContext(ctx):
       per subdomain sd in g.Subdomains:
         go func() {
           deleted := sd.DeleteAllForTenant(db.WithContext(gctx))
           files   := walkAllRoots(src, roots, sd.Path())   // overlay-aware
           models  := []
           for f in files:
             payload := openAndExtractAttributes(src, f)
             j := sd.Decode(payload)
             rows := sd.Build(t, entityIdFromFilename, j)
             models = append(models, rows...)
           created := sd.BulkCreate(db, models)
           result.Subdomains[sd.Name()] = SubdomainCounts{...}
         }
  6. result.CompletedAt = now
  7. writeSeedState(db, t.Id(), g.Name, rev, result)  // even on partial failure
  8. metrics: counter atlas_seeder_runs_total{service, group, outcome}++
  9. metrics: histogram atlas_seeder_duration_seconds{service, group}.Observe(elapsed)
  10. return result, nil
```

Key invariants:

- **Subdomains run in parallel via `errgroup`.** Each subdomain's delete + walk + bulk-insert is a single goroutine. The number of subdomains per group is small (≤3 today), so this is fan-out, not flooding.
- **Per-subdomain failures do not abort the group.** Each subdomain's errors land in `SubdomainCounts.Errors` (capped at 100); the group continues. Only an `errgroup`-level error (e.g., context cancellation, database connection death) aborts. This matches existing service behavior — a malformed monster drop file does not prevent reactor drops from seeding.
- **`seed_state` is written on completion regardless of partial failure.** The `result_summary` JSON captures whatever happened. The PRD's "successful seed" wording is loosened here: we record the last attempt and its outcome; a future reconciler decides whether to retry.
- **`outcome` label on the counter** is one of `success` (all subdomains zero failures), `partial` (≥1 failure but ≥1 success), `failure` (all subdomains failed). This gives operators a Prometheus-side signal for catalog regressions.

### 3.5 `Status` orchestration

```
Status(ctx, db, src, g):
  1. t := tenant.MustFromContext(ctx)
  2. roots, _ := src.Roots(t)
  3. catalogRev, _ := src.Revision(roots[0])
  4. seedRow, _ := readSeedState(db, t.Id(), g.Name)   // may be missing
  5. errgroup: per subdomain → Count(db) in parallel
  6. assemble Status struct; max UpdatedAt across subdomains
  7. if seedRow == nil: TenantSeededRevision = nil, TenantSeededAt = nil
     else:              both = values from row
  8. if catalogRev != *TenantSeededRevision: log.Warn (drift signal)
  9. return Status, nil
```

`GET /seed/status` is purely synchronous and budgeted at ~50ms p95; the per-subdomain `Count` calls dominate (each is one indexed `SELECT count(*), max(updated_at)`), and `seed_state` is one PK lookup.

### 3.6 `CatalogSource` and tenant resolution

`FilesystemCatalogSource` resolves a tenant's catalog root via:

```
root := $SEED_CATALOG_ROOT / tenant.Region() / fmt.Sprintf("%d_%d", tenant.MajorVersion(), tenant.MinorVersion())
```

The `Roots(t)` method returns `[]string{root}` in v1. The "ordered list" is dead code on day one but is the seam for the future `_base/<region>/` overlay model. The loader walks each root and merges by filename; later roots win, and `<filename>.tombstone` suppresses earlier roots. The loader honors the order so introducing overlays is purely a config change.

`SEED_CATALOG_ROOT` defaults to `/var/run/seed-catalog` (containers). On dev (no env var set), the fallback root is the constructor's second argument; each service passes its own service-tree fallback (typically `./deploy/seed`, repo-relative). PRD Open Question 4 is resolved as: **default to `./deploy/seed` in dev** so `go run` from a service dir picks up the in-tree catalog with no env wiring.

`Revision(root)` reads `<root>/CATALOG_REVISION` and trims whitespace. A missing file returns the empty string; the lib logs a `WARN` but does not abort. The PRD's "lib does not 4xx for missing catalog" rule is preserved: the first `POST /seed` after the mount disappears produces a logged failure and a `seed_state` row with all-zero counts, but still returns 202.

`Walk(root, relPath)` skips entries whose names start with `_` or `.` and skips subdirectories whose names start with `_`. The `_global/` convention falls out of this naturally — services that need non-entity files load them via a dedicated `Subdomain` whose `EntityIDPattern` is `nil` (signaling "this subdomain does not iterate entity files; it loads exactly one named file"). Gachapons is the only service that needs this on day one.

### 3.7 JSON:API parsing

The library parses every file as:

```go
type envelope struct {
    Data struct {
        Type          string          `json:"type"`
        ID            string          `json:"id"`
        Attributes    json.RawMessage `json:"attributes"`
        Relationships json.RawMessage `json:"relationships,omitempty"`
    } `json:"data"`
}
```

Validation per file:

1. JSON parses.
2. `data.type == sd.Type()`.
3. `data.id` matches the entity id extracted from the filename via `sd.EntityIDPattern()` (when the pattern is non-nil).
4. `sd.Decode(data.attributes)` returns successfully.

Any failure appends to `SubdomainCounts.Errors` with the filename prefix and increments `Failed`.

The same `envelope` parse code is reused by the CI catalog linter (see §6) — exporting a `ParseEnvelope([]byte) (Envelope, error)` helper from `libs/atlas-seeder` avoids re-implementing the parse in the linter.

### 3.8 `SeedState` entity

```go
package seeder

type SeedState struct {
    TenantID         uuid.UUID       `gorm:"type:uuid;primaryKey"`
    GroupName        string          `gorm:"type:text;primaryKey"`
    CatalogRevision  string          `gorm:"type:text;not null"`
    SeededAt         time.Time       `gorm:"type:timestamptz;not null"`
    ResultSummary    datatypes.JSON  `gorm:"type:jsonb;not null"`
}

func (SeedState) TableName() string { return "seed_state" }
```

`datatypes.JSON` is from `gorm.io/datatypes` (already a transitive dep in services using GORM). Each service calls `db.AutoMigrate(&seeder.SeedState{})` once at bootstrap. No service-side migration files; GORM's idempotent migration is the only schema source. (If we later need explicit migrations, the entity is the input to a generator — this stays open.)

### 3.9 Metrics

Two Prometheus metrics registered via `promauto` lazily on first use:

- `atlas_seeder_runs_total{service, group, outcome}` — counter.
- `atlas_seeder_duration_seconds{service, group}` — histogram with default buckets.

`service` is read from `os.Getenv("ATLAS_SERVICE_NAME")` with a fallback of the binary name; each service already sets this env in its k8s manifest and compose entry.

---

## 4. Per-Service Migration Pattern

Each of the eight services adopts the same shape:

```
services/atlas-<svc>/atlas.com/<svc>/
├── main.go                              # adds AutoMigrate(&seeder.SeedState{}), calls seed.Init
└── seed/
    └── groups.go                        # ~80 lines: defines Group(s) + their Subdomains
```

The per-service `seed/groups.go` follows a template:

```go
package seed

func Init(db *gorm.DB, si jsonapi.ServerInformation) func(*mux.Router) {
    return func(r *mux.Router) {
        src := seeder.NewFilesystemCatalogSource("SEED_CATALOG_ROOT", "./deploy/seed")

        seeder.RegisterRoutes(r, db, si, src, seeder.Group{
            Name:      "drops",
            URLPrefix: "/drops",
            Subdomains: []seeder.SubdomainAny{
                seeder.AdaptSubdomain(monsterDropSubdomain{}),
                seeder.AdaptSubdomain(continentDropSubdomain{}),
                seeder.AdaptSubdomain(reactorDropSubdomain{}),
            },
        })
    }
}
```

Each `Subdomain` implementation lives next to its domain code (e.g., `monster/drop/subdomain.go`) and is ~40 lines: it reuses the existing `Builder`, `BulkCreate*`, `Count`, and `DeleteAll*` functions. Only the `Decode` and `Build` methods are genuinely new — and `Decode` is usually a one-line `json.Unmarshal` into the existing `JSONModel` shape.

**Endpoint URLs are preserved verbatim.** The PRD's Open Questions 1 and 2 (exact paths for the script services and npc-conversations) resolve at migration time by *reading the current `router.HandleFunc(...)` lines before deletion* and copying them into the new `Group.URLPrefix`. The plan task records the actual paths.

### 4.1 Migration order

Services migrate in this sequence; each is one PR-equivalent commit on the task branch:

1. **`libs/atlas-seeder`** ships first with full unit-test coverage. No service depends on it yet.
2. **`atlas-gachapons`** migrates next. It has the smallest catalog (~150 KB across two files) and the most distinctive subdomain shape (the `_global/` non-entity file), so it stresses the library's edge cases first.
3. **`atlas-drop-information`** migrates next. The largest catalog (3.1 MB) and three subdomains; if the splitter and the per-subdomain orchestrator survive this, the rest are mechanical.
4. **The five remaining services** migrate one at a time in any order. Each is the same recipe: write subdomains, register groups, delete old code, delete bundled data, wire k8s/compose, update Dockerfile.

This ordering means the lib's first real consumer surfaces design bugs before five other services have copy-pasted around them. Tasks are sequenced this way in `plan.md`.

### 4.2 What gets deleted from each service

For each service, the deletion checklist:

- The service's `seed/` package (or the seed code inside non-seed-named packages, e.g., `conversation/npc/seed.go`).
- The bundled catalog data directory under the service tree (`data/`, `drops/`, `scripts/`, `conversations/`, `party-quests/`).
- The `COPY` lines in the service's `Dockerfile` that pulled the catalog into the image.
- Any service-local env-var defaults for catalog paths (e.g., `GACHAPONS_DATA_PATH`).

What survives:

- The domain types (`JSONModel`, `Model`, the `Builder`, `BulkCreate*`, `Count`, `DeleteAll*`). These are exactly what the new `Subdomain` impl calls. No type churn.

### 4.3 Multi-group services

`atlas-npc-conversations` and `atlas-npc-shops` register two `Group`s each. They share one `seed_state` table; rows are discriminated by `group_name`. The two groups walk distinct subtrees (`npc-conversations/npc/` vs `npc-conversations/quests/`).

### 4.4 Dockerfile lib-list update

Per CLAUDE.md, adding `libs/atlas-seeder` requires updating the Dockerfile in all four lib-list locations (`go.mod` stage `COPY`s, the synthesized `go.work use(...)` block, source `COPY`s, and the `go mod edit -replace=...` flags). Each of the eight services' migration commits performs this update. The plan task gives each service migration its own `docker build` verification step to catch drift before CI.

---

## 5. Catalog Tree and Splitters

### 5.1 Directory layout (concrete)

```
deploy/seed/
├── _schema/                                          # JSON Schemas used by linter
│   ├── envelope.schema.json
│   ├── monster-drop.schema.json
│   ├── gachapon.schema.json
│   └── ...
├── gms/
│   ├── 12_1/   CATALOG_REVISION  drops/  gachapons/  npc-conversations/  map-actions/  portal-actions/  reactor-actions/  npc-shops/  party-quests/
│   ├── 83_1/   (same)
│   ├── 87_1/   (same; bootstrapped from 83_1)
│   ├── 92_1/   (same; bootstrapped from 83_1)
│   └── 95_1/   (same; bootstrapped from 83_1)
└── jms/
    └── 185_1/  (same; bootstrapped from 83_1)
```

Bootstrap copies are byte-identical to `gms/83_1/` except for `CATALOG_REVISION`, which reads `bootstrapped-from-gms-83_1-@<sha>` so operators can tell at a glance that nothing has diverged yet. Real per-version edits land lazily.

### 5.2 Splitter shape

Each splitter is a Go program under `tools/seed-splitters/<name>/main.go` with a uniform CLI:

```
split-monster-drops --input services/atlas-drop-information/drops/monsters/monster_drops.json \
                    --output deploy/seed/gms/83_1/drops/monsters
```

Determinism rules — every splitter:

- Reads input, sorts records by primary id ascending.
- Writes one file per id with `data.id` as the string form of the numeric id.
- Uses `json.MarshalIndent` with two-space indent and a trailing newline.
- Never writes a timestamp, hostname, or random value into output.
- Reruns produce byte-identical output (verified by a `go test` in `tools/seed-splitters/<name>/` that runs splitter twice on fixture input and `diff`s).

The four splitters:

- `split-monster-drops` — fans monster_drops.json (one row per (monster,item)) into one file per monster with an `attributes.drops` array.
- `split-continent-drops` — same pattern for continent drops.
- `split-gachapons` — merges `gachapons.json` + `gachapon_items.json` into one file per gachapon with inline items; emits `_global/items.json` for the global pool.
- `wrap-jsonapi` — generic. Takes `--input-dir`, `--output-dir`, `--type`, `--id-field`. Reads every `*.json` file in input-dir, treats each as `attributes`, wraps in JSON:API envelope, writes to output-dir.

Splitters are **not** run by CI. They are one-shot bootstrappers checked in for reproducibility. The plan task includes an "execute splitters once and commit the output" step; after that, the catalog files are the source of truth and the splitter inputs (the old bundled files) are deleted.

### 5.3 CI catalog linter

`tools/catalog-lint/main.go` is invoked as `go run ./tools/catalog-lint <root>`. It:

1. Walks `<root>/**/*.json`, skipping `_*` files and directories starting with `_` or `.`.
2. Identifies the owning subdomain from the parent directory path against a hard-coded mapping (`drops/monsters` → monster-drop, etc.). The mapping lives in `tools/catalog-lint/subdomains.go`.
3. Parses each file via `seeder.ParseEnvelope`.
4. Checks `data.type` matches the expected type.
5. Extracts the entity id from filename and checks `data.id == filename id`.
6. Validates kebab-case filenames ending in `.json`.
7. Walks every `<region>/<major>_<minor>/` directory and requires `CATALOG_REVISION` to exist and be non-empty.
8. Exits 0 if all checks pass, 1 if any fail, with a per-file error report on stderr.

The GitHub Actions workflow `.github/workflows/catalog-lint.yml` runs the linter on PRs that touch `deploy/seed/**` or `tools/catalog-lint/**`. PRD Open Question 6 is resolved: **strict on PRs, advisory on `main`** — the `main` branch run logs failures but does not block deploys. This prevents a catalog mistake on main from blocking unrelated PRs while still surfacing the failure.

The linter's fixture catalogs (good and bad) live under `tools/catalog-lint/testdata/`. A `lint_test.go` runs the linter against both and asserts exit codes.

---

## 6. k8s and Compose Plumbing

### 6.1 k8s Kustomize component

New component at `deploy/k8s/base/components/seed-catalog/`:

```
deploy/k8s/base/components/seed-catalog/
├── kustomization.yaml                    # kind: Component
├── volume.yaml                           # patch: adds emptyDir volume "seed-catalog"
├── sidecar.yaml                          # patch: adds git-sync container
└── service-mount.yaml                    # patch: adds volumeMount + env SEED_CATALOG_ROOT
```

Each of the eight service manifests references the component:

```yaml
# deploy/k8s/base/atlas-drop-information/kustomization.yaml
components:
  - ../components/seed-catalog
```

The component is parameterized via a single ConfigMap (`seed-catalog-config`) that an overlay can patch:

```yaml
apiVersion: v1
kind: ConfigMap
metadata: { name: seed-catalog-config }
data:
  GITSYNC_REPO: "https://github.com/Chronicle20/atlas"
  GITSYNC_REF:  "main"        # patched to PR SHA in PR overlays
  GITSYNC_ROOT: "/git"
  GITSYNC_DIR:  "deploy/seed"
  GITSYNC_PERIOD: "60s"
```

`git-sync` runs as a sidecar (not initContainer) with `requests` of 50m/64Mi and `limits` of 200m/256Mi. The image is pinned to `registry.k8s.io/git-sync/git-sync:v4.4.0` (current stable).

PRD Open Question 5 resolves via the ConfigMap-and-patch pattern: ArgoCD's PR overlay patches `GITSYNC_REF` to the PR SHA via a Kustomize patch. No ApplicationSet parameter is needed.

### 6.2 docker-compose anchor

`deploy/compose/docker-compose.yml` adds:

```yaml
x-seed-catalog: &seed-catalog
  volumes:
    - ../seed:/var/run/seed-catalog:ro
  environment:
    SEED_CATALOG_ROOT: /var/run/seed-catalog
```

Each of the eight service blocks references it:

```yaml
services:
  atlas-drop-information:
    <<: *seed-catalog
    image: ...
    # other config
```

YAML merge keys (`<<`) merge `environment` and `volumes` into the service block. Existing per-service `environment` and `volumes` entries are preserved.

### 6.3 Dev (no container) fallback

When running a service via `go run` against the source tree, no env is set; `FilesystemCatalogSource` falls back to its constructor's `fallbackRoot` argument. Services pass `./deploy/seed` (relative to the worktree root). This works because `go run` from `services/atlas-drop-information/atlas.com/dis/` resolves `./deploy/seed` against the binary's cwd — which the dev script (`build.sh` / direct `go run`) sets to the service's working directory. **The service's bootstrap normalizes the fallback to an absolute path** via `filepath.Abs` so the dev case doesn't break when the binary is invoked from elsewhere.

---

## 7. Testing Strategy

### 7.1 Library unit tests

`libs/atlas-seeder/` ships with these test files:

- `seeder_test.go` — integration-style: in-memory SQLite, fixture catalog, two fake subdomains, calls `Seed` and `Status`, asserts row counts, `seed_state` contents, and `Result` shape.
- `subdomain_test.go` — verifies the type-erased adapter preserves J/M semantics.
- `catalog_test.go` — `FilesystemCatalogSource`: tenant → root resolution, revision read, `_global/` skip, tombstone suppression, missing `CATALOG_REVISION` handling.
- `jsonapi_test.go` — envelope parsing: valid, missing data, type mismatch, id mismatch, malformed JSON.
- `state_test.go` — `SeedState` upsert semantics, partial-failure persistence, never-seeded read.
- `errgroup_test.go` — concurrency: one subdomain fails, others continue; `errgroup` cancellation propagates only on context death.
- `metrics_test.go` — counter and histogram are registered exactly once and observable via Prometheus testutil.

Fixture catalogs live at `libs/atlas-seeder/testdata/`:

```
testdata/
├── good/gms/83_1/
│   ├── CATALOG_REVISION
│   ├── widgets/widget-1.json
│   ├── widgets/widget-2.json
│   ├── gizmos/_global/pool.json
│   └── gizmos/gizmo-100.json
├── bad/
│   ├── filename-id-mismatch/...
│   ├── type-mismatch/...
│   └── missing-revision/...
```

Tests use `sqlite::memory:` via `gorm.io/driver/sqlite` so each test has a fresh DB.

### 7.2 Per-service tests

Each service keeps its existing `seed_test.go` and `status_test.go` (the table-level tests). The migration replaces their innards to call into the library, but the test names and assertions stay (so reviewers can see behavior preserved).

### 7.3 Splitter and linter tests

Each splitter has a `main_test.go` that:

1. Runs the splitter on a tiny fixture input.
2. Runs it a second time.
3. Asserts byte-identical output between runs and against a checked-in expected output directory.

The linter has the symmetric test: a `good/` fixture passes; each `bad/<scenario>/` fixture fails with a specific exit code and stderr line.

### 7.4 Compose smoke test (manual gate)

The PRD's E2E criterion (`POST /drops/seed` in compose returns 202, status reports counts and `catalogRevision`) is a manual gate executed during the plan task's verification step. It's not in CI because the compose stack is heavyweight; the plan task records the exact `curl` commands and expected responses.

---

## 8. Tradeoffs and Alternatives Considered

### 8.1 Library granularity: one Group per service vs. one library per service

**Chosen: one library, services compose Groups.** This is what the PRD prescribes.

Alternative considered: code-generate a per-service library that hard-codes its Groups. Rejected because the duplication-elimination goal would just shift from runtime to build-time, and the generator would itself be code to maintain.

### 8.2 JSON:API vs. plain JSON for catalog files

**Chosen: JSON:API for everything.** One parser, one linter, one fixture shape.

Alternative considered: keep current per-service file shapes (some JSON:API, some plain arrays, some maps). Rejected because the linter would need N parsers, and the library would need a per-subdomain decoder pipeline. The cost of envelope-wrapping every file is one one-shot splitter run.

### 8.3 Type erasure vs. interface-only `Subdomain`

**Chosen: `Subdomain[J, M]` generic + type-erased adapter.** Services get type-safe `Decode`/`Build` signatures; the library treats subdomains uniformly.

Alternative considered: non-generic `Subdomain` interface with `Decode([]byte) (any, error)` and `Build(t, id, any) ([]any, error)`. Rejected because services would write `any.(*JSONModel)` casts everywhere — every existing code path is statically typed today.

### 8.4 Persisting `seed_state` on partial failure

**Chosen: persist on every `Seed` completion, even partial failures.** `result_summary` records what happened; counts make it queryable.

Alternative considered: persist only on full success. Rejected because the row's primary role is reconciler input — a reconciler needs to see "last attempted revision" even if it failed, to decide whether to retry vs. log-and-skip.

### 8.5 `git-sync` sidecar vs. initContainer

**Chosen: sidecar (continuously polls).** Catalog updates flow without pod restart; aligns with PRD §4.6.

Alternative considered: initContainer (one-shot at pod start). Rejected because a catalog edit would require a pod restart to land, which negates much of the "ship at git push cadence" goal.

### 8.6 `seed_state` table per service vs. centralized

**Chosen: per-service `seed_state` table.** Each service's DB is independent; no cross-service joins.

Alternative considered: centralized `seed_state` in `atlas-tenants` or a new metadata service. Rejected because (a) it introduces a cross-service runtime dependency for `POST /seed`, (b) reconciler logic would need to fan out, (c) it forces tenants to grant another service write access to its seed log. Per-service tables keep ownership clear.

### 8.7 Linter strict on `main` vs. strict on PRs

**Chosen: strict on PRs, advisory on `main`.** A catalog mistake on main shouldn't block unrelated PRs.

Alternative considered: strict everywhere. Rejected for the rollback risk: if a malformed file lands on main (despite PR enforcement), every subsequent PR until the fix would fail CI.

### 8.8 Overlay model (`_base/` overlays) on day one

**Deferred.** The PRD calls out that `CatalogSource.Roots()` returns an ordered list but v1 only ever returns one element. The loader walks the list anyway. This is "build the seam, leave it dormant."

The cost is ~40 lines of unused code paths in the loader. The benefit is that the day we want to dedupe `gms/87_1` from `gms/83_1`, it's a config change rather than a refactor. The unit tests exercise the multi-root path with fixtures so it doesn't bitrot.

### 8.9 Per-entity bucketing (e.g., `monsters/1/100100.json`)

**Deferred.** Files start flat. Splitter outputs ~10,000 monster files into one directory; this is fine for filesystems (ext4 directory limits are in the millions) and for `find`/`ls`. If the directory becomes operationally painful, bucketing by leading id digit is a Kustomize-level decision (split the directory; update `EntityIDPattern` to capture across one extra path segment).

---

## 9. Risks and Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| `git-sync` sidecar fails on a PR overlay; service pod starts but catalog is empty. | Medium | High (seeds insert zero rows). | Library logs `WARN` when `CATALOG_REVISION` is missing or empty; `seed_state.result_summary` records all-zero counts; ops can `kubectl logs -c git-sync` to diagnose. |
| Service's `Dockerfile` lib-list is updated in 3-of-4 locations; image builds locally but fails CI. | High (this has happened repeatedly per CLAUDE.md). | Medium (CI catches it). | Each per-service migration commit runs `docker build -f services/<svc>/Dockerfile .` from the worktree root as a verification gate. The plan task makes this an explicit checkbox per service. |
| Splitter is non-deterministic; reruns produce different output, polluting diffs. | Low | Medium (review pain). | Every splitter has a "run twice, assert byte-identical" test. The plan task includes "rerun splitter and verify zero diff" as a check before committing the catalog. |
| JSON:API envelope wrapping breaks an existing consumer that reads catalog files outside the seed flow. | Low | High (silent breakage). | grep across the repo for direct catalog reads before deletion; the PRD's "8 services + catalog tree" scoping verified there are no other consumers, but the plan task re-runs the grep on the actual catalog paths. |
| `seed_state` table conflicts with an existing service-local table of the same name. | Very low | Low (compile error). | grep for `seed_state` across the eight services before the migration commit. No collisions found in current code. |
| `tenant.MajorVersion/MinorVersion` not populated for a tenant in a fresh DB. | Medium during dev | Medium (catalog root resolves to `<region>/0_0/`). | `FilesystemCatalogSource.Roots()` returns an error when either is zero; lib treats this as "no catalog available" and seeds zero rows with a clear log message. |
| The atlas-ui's existing seed buttons issue requests that no longer match (e.g., URL drift during migration). | Low | High (operator-facing breakage). | URL preservation is a hard rule — the plan task's per-service step explicitly diffs `router.HandleFunc` calls before vs. after the migration. Any URL change blocks the commit. |

---

## 10. Open Items Carried Into `plan.md`

The PRD's Open Questions are resolved as follows; the plan task carries each as a concrete sub-step:

| PRD OQ | Resolution in this design | Plan-task action |
|---|---|---|
| 1. Today's script-service URLs | Preserve verbatim; copy from existing `router.HandleFunc`. | Per-service migration sub-step reads + records current URL. |
| 2. npc-conversations URLs | Same — preserve verbatim. | Same. |
| 3. Portal id encoding in filenames | Decide at splitter-implementation time; recommend URL-safe kebab-case via lowercased portal name with non-`[a-z0-9-]` chars replaced by `-`. | Splitter spec encodes the exact rule. |
| 4. `SEED_CATALOG_ROOT` dev default | Resolved: lib's `NewFilesystemCatalogSource` takes a per-service fallback path; services pass `./deploy/seed`. | Each service's `seed/groups.go` template includes the fallback. |
| 5. git-sync ref strategy | Resolved: Kustomize patch on `seed-catalog-config` ConfigMap per overlay. | Overlay patch lands as part of the deploy task. |
| 6. Linter strict on main vs. PRs | Resolved: strict on PRs, advisory on main. | CI workflow encodes this. |

---

## 11. Deliverable Summary

The implementation plan (next phase) decomposes into these task groups:

1. **`libs/atlas-seeder`** — create module, implement types and orchestrators, ship unit tests.
2. **Splitters** — implement four splitters, run them, commit output.
3. **Catalog tree** — assemble `deploy/seed/<region>/<version>/` directories, write `CATALOG_REVISION` files, bootstrap non-v83 versions.
4. **Per-service migration (×8)** — for each of gachapons, drop-information, map-actions, reactor-actions, portal-actions, npc-conversations, npc-shops, party-quests: replace seed code, delete bundled data, update Dockerfile, add AutoMigrate. Sequenced gachapons → drop-information → rest.
5. **Infra** — k8s Kustomize component, compose anchor, per-service manifest references.
6. **CI catalog linter** — implement linter, ship GitHub Actions workflow.
7. **End-to-end verification** — compose smoke tests, k8s dry-run, audit per CLAUDE.md.

Acceptance criteria are inherited verbatim from PRD §10. The plan-task elaborates each into ordered, individually-verifiable sub-tasks.
