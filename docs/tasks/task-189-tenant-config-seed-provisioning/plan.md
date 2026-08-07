# Tenant-Configuration Seed Provisioning & Transport Registry Fixes — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make tenant transport configuration seedable from the Setup page, make configuration hot-reload actually reach atlas-transports, and give every configured route one stable identity so the registry converges to exactly the configured count.

**Architecture:** Three transport resources (`routes`, `vessels`, `instance-routes`) move off atlas-tenants' bespoke path-scoped seed handlers onto `libs/atlas-seeder`, with their catalog data relocated to a new version-agnostic root `deploy/seed/shared/all/` that the seeder merges under the tenant's version-specific root. atlas-tenants' configuration processor gains a helper that builds a fully-populated `tenant.Model` emit context, so configuration-status events finally carry the four tenant headers. Route identity becomes a derived UUIDv5 (`uuid.NewSHA1(tenantId, "<resource>/<slug>")`) computed identically in atlas-tenants (exposed as a `uuid` attribute) and atlas-transports (fallback), and atlas-transports' bootstrap becomes load-then-clear-then-add so a rolling restart purges the existing drift.

**Tech Stack:** Go 1.x (gorm, gorilla/mux, segmentio/kafka-go, logrus, google/uuid), `libs/atlas-seeder`, `libs/atlas-outbox`, `libs/atlas-tenant`, `libs/atlas-database`; TypeScript/React 19 + TanStack React Query 5 + Vitest for atlas-ui; kustomize for deploy.

## Global Constraints

- Repo root for every path in this plan is the task worktree `.worktrees/task-189-tenant-config-seed-provisioning/`. Never edit the main checkout.
- Changed Go modules: `libs/atlas-seeder`, `libs/atlas-outbox`, `libs/atlas-tenant`, `services/atlas-tenants/atlas.com/tenants`, `services/atlas-transports/atlas.com/transports`. Each must pass `go test -race ./...`, `go vet ./...`, `go build ./...`.
- `docker buildx bake atlas-tenants atlas-transports` from the worktree root is mandatory (both `go.mod` files change) — see Task 13.
- `tools/redis-key-guard.sh`, `tools/goroutine-guard.sh`, `tools/lint.sh --check`, `tools/service-registration-guard.sh` must all be clean from the repo root.
- Every goroutine goes through `routine.Go` (`libs/atlas-routine`). No bare `go` statements.
- No `// TODO`, stubs, or 501s in any landed commit.
- Never write literal home/absolute paths (`/home/<user>/…`) into committed files.
- Multi-tenancy: seeding one tenant must never touch another's rows. `libs/atlas-database`'s tenant callback (`libs/atlas-database/tenant_scope.go:55-80`) auto-injects `tenant_id = <ctx tenant>` on any query whose model has a `tenant_id` column **when the context carries a tenant**, and silently skips when it does not.
- The tenant header constants are `tenant.ID`, `tenant.Region`, `tenant.MajorVersion`, `tenant.MinorVersion` from `libs/atlas-tenant`.
- Seed data files are JSON:API envelopes: `{"data":{"id":…,"type":…,"attributes":{…}}}`. `seeder.ParseEnvelope` (`libs/atlas-seeder/jsonapi.go:20`) requires the `data` wrapper; the pre-existing atlas-tenants files are **not** wrapped.
- Derived-id formula, used verbatim on both sides: `uuid.NewSHA1(tenantId, []byte(strings.Join(parts, "/")))` with `parts = {resourceName, slug}` — e.g. `{"instance-routes", "temple-of-time-return-flight"}`.
- Configured counts as of this branch: `routes` = 12 files, `vessels` = 6 files, `instance-routes` = 12 files.

---

## File Structure

**`libs/atlas-seeder`** (shared seeding framework — additive only; nine existing consumers must see zero behavioural change)
- `catalog.go` — gains `NewFilesystemCatalogSourceWithShared`; `filesystemSource` gains a `sharedRel` field and `Roots` returns `[shared, versionSpecific]` when set.
- `seed.go` — `runSubdomain` merges filenames across roots (later root wins); new `revisionFor` helper joins per-root revisions with `+`.
- `status.go` — uses `revisionFor` so status and seed can never disagree.
- `seeder.go` — `Group` gains the optional `AfterSeed` hook.
- `handlers.go` — `postSeed` invokes `AfterSeed` after a successful `Seed`, logging (not returning) its error.

**`libs/atlas-tenant`**
- `id.go` (new) — `DerivedId(tenantId uuid.UUID, parts ...string) uuid.UUID`, the single home of the UUIDv5 formula.

**`libs/atlas-outbox`**
- `bridge.go` — `EnqueueBuffer` logs a WARN naming the topic when the enqueue context has no tenant.

**`deploy/seed/shared/all/`** (new, version-agnostic catalog root)
- `CATALOG_REVISION`, `routes/*.json` (12), `vessels/*.json` (6), `instance-routes/*.json` (12) — each rewrapped in a `{"data": …}` envelope.

**`tools/catalog-lint/subdomains.go`** — three new rules so the shared tree is linted, not silently ignored.

**`services/atlas-tenants/atlas.com/tenants`**
- `configuration/administrator.go` — gains `AppendConfigurationEntries` and `CountConfigurationEntries`.
- `configuration/seed/groups.go` (new) — the three `seeder.Group`s, their `AfterSeed` hooks, and the route initializer.
- `configuration/seed/subdomain.go` (new) — one generic `Subdomain` implementation parameterised by resource name.
- `configuration/seed.go` — the three transport loaders and their path helpers are deleted; `rps-rewards` / `mts-configs` loaders stay.
- `configuration/processor.go` — `Seed{Routes,Vessels,InstanceRoutes}` removed; new `tenantCtx` helper threaded through all 18 `…AndEmit` methods.
- `configuration/mock/processor.go` — the three removed methods drop out of the mock.
- `configuration/resource.go` — three seed handlers and their route registrations removed; `Transform*` call sites gain a `tenantId` argument.
- `configuration/rest.go` — `RouteRestModel`, `VesselRestModel`, `InstanceRouteRestModel` gain a `Uuid` field; `Transform*` gain a `tenantId` parameter.
- `main.go` — `seeder.SeedState` migration + the seed route initializer registered *before* `configuration.RegisterRoutes`.
- `go.mod` — `libs/atlas-seeder` require + replace.

**`services/atlas-transports/atlas.com/transports`**
- `instance/config/rest.go`, `transport/config/rest.go` — `Uuid` field, `ExtractRouteFor(l, t)`, local `resolveRouteId` helper.
- `instance/config/processor.go`, `transport/config/processor.go` — `Get*` take `tenant.Model` instead of a bare id string.
- `kafka/consumer/configuration/consumer.go` — nil-tenant guard; load-then-clear-then-add.
- `bootstrap.go` (new, `package main`) — testable `reconcileScheduled` / `reconcileInstance`.
- `main.go` — bootstrap loop calls the new reconcilers.

**`services/atlas-ui/src`**
- `services/api/seed.service.ts` — three status types, three seed methods, three status methods.
- `lib/hooks/api/useSeed.ts` — three query keys, three mutations, three status queries.
- `pages/SetupPage.tsx` — three `SeedRow` entries.

**`deploy/k8s/base/atlas-tenants.yaml`** — `atlas.seed-catalog: "true"` label.

**`docs/tasks/task-189-tenant-config-seed-provisioning/runbook.md`** (new) — FR-5.2 rollout + verification + manual fallback.

---

### Task 1: `libs/atlas-seeder` — shared (version-agnostic) catalog root

**Files:**
- Modify: `libs/atlas-seeder/catalog.go:22-59`
- Modify: `libs/atlas-seeder/seed.go:42-121`
- Modify: `libs/atlas-seeder/status.go:22-26`
- Test: `libs/atlas-seeder/catalog_test.go`, `libs/atlas-seeder/seed_test.go`
- Create: `libs/atlas-seeder/testdata/good/shared/all/CATALOG_REVISION`
- Create: `libs/atlas-seeder/testdata/good/shared/all/widgets/widget-2.json`
- Create: `libs/atlas-seeder/testdata/good/shared/all/widgets/widget-3.json`

**Interfaces:**
- Consumes: nothing from earlier tasks.
- Produces:
  - `seeder.NewFilesystemCatalogSourceWithShared(envVar, fallbackRoot, sharedRel string) CatalogSource` — `Roots` returns `[]string{<base>/<sharedRel>, <base>/<region>/<M>_<m>}`, least-specific first.
  - Unexported `revisionFor(src CatalogSource, roots []string) string`.
  - `NewFilesystemCatalogSource` keeps its exact current one-root behaviour.

- [ ] **Step 1: Add the shared-root fixtures**

`libs/atlas-seeder/testdata/good/shared/all/CATALOG_REVISION` (no trailing newline):

```
shared-rev-xyz789
```

`libs/atlas-seeder/testdata/good/shared/all/widgets/widget-2.json` — same relative path as the existing version-specific `gms/83_1/widgets/widget-2.json`, with a *different* name so precedence is observable:

```json
{"data":{"type":"widget","id":"2","attributes":{"name":"two-from-shared"}}}
```

`libs/atlas-seeder/testdata/good/shared/all/widgets/widget-3.json` — shared-only, proves union:

```json
{"data":{"type":"widget","id":"3","attributes":{"name":"three"}}}
```

- [ ] **Step 2: Write the failing tests**

Append to `libs/atlas-seeder/catalog_test.go`:

```go
// newSharedTestSource returns a CatalogSource with both a shared root and
// the tenant's version-specific root, over the testdata/good fixture tree.
func newSharedTestSource(t *testing.T) CatalogSource {
	t.Helper()
	return NewFilesystemCatalogSourceWithShared("ATLAS_SEEDER_CATALOG_TEST_OVERRIDE", "testdata/good", "shared/all")
}

func TestFilesystemCatalogSource_Roots_SharedFirstThenVersion(t *testing.T) {
	roots, err := newSharedTestSource(t).Roots(tenantGMS83(t))
	if err != nil {
		t.Fatalf("Roots: %v", err)
	}
	if len(roots) != 2 {
		t.Fatalf("len(roots) = %d, want 2", len(roots))
	}
	if !strings.HasSuffix(roots[0], filepath.Join("shared", "all")) {
		t.Errorf("roots[0] = %q, want shared root first", roots[0])
	}
	if !strings.HasSuffix(roots[1], filepath.Join("gms", "83_1")) {
		t.Errorf("roots[1] = %q, want version-specific root second", roots[1])
	}
}

// The nine existing consumers construct the plain source; it must keep
// returning exactly one root so their composite revision never changes.
func TestFilesystemCatalogSource_Roots_PlainSourceStillSingleRoot(t *testing.T) {
	roots, err := newTestSource(t).Roots(tenantGMS83(t))
	if err != nil {
		t.Fatalf("Roots: %v", err)
	}
	if len(roots) != 1 {
		t.Fatalf("len(roots) = %d, want 1", len(roots))
	}
}

func TestRevisionFor_SingleRootIsUnchanged(t *testing.T) {
	src := newTestSource(t)
	roots, _ := src.Roots(tenantGMS83(t))
	if got := revisionFor(src, roots); got != "test-rev-abc123" {
		t.Fatalf("revisionFor = %q, want %q", got, "test-rev-abc123")
	}
}

func TestRevisionFor_TwoRootsJoinsWithPlus(t *testing.T) {
	src := newSharedTestSource(t)
	roots, _ := src.Roots(tenantGMS83(t))
	want := "shared-rev-xyz789+test-rev-abc123"
	if got := revisionFor(src, roots); got != want {
		t.Fatalf("revisionFor = %q, want %q", got, want)
	}
}

func TestRevisionFor_SkipsEmptyRevisions(t *testing.T) {
	src := newSharedTestSource(t)
	// t.TempDir() has no CATALOG_REVISION → Revision returns ("", nil).
	roots, _ := src.Roots(tenantGMS83(t))
	got := revisionFor(src, []string{t.TempDir(), roots[1]})
	if got != "test-rev-abc123" {
		t.Fatalf("revisionFor = %q, want %q (empty root contributes nothing)", got, "test-rev-abc123")
	}
}
```

Append to `libs/atlas-seeder/seed_test.go`:

```go
func TestSeed_MergesRootsWithVersionSpecificWinning(t *testing.T) {
	t.Cleanup(ResetMetricsForTest)
	db := openTestDB(t)
	src := NewFilesystemCatalogSourceWithShared("X_NO_ENV", goodFixtureRoot(t), "shared/all")
	sub := &widgetSubdomain{}
	g := Group{
		Name:       "merge-group",
		URLPrefix:  "/merge",
		Subdomains: []SubdomainAny{AdaptSubdomain[widgetAttrs, widgetRow](sub)},
	}
	ctx := tenant.WithContext(context.Background(), tenantGMS83(t))
	res, err := Seed(ctx, db, src, g)
	if err != nil {
		t.Fatalf("Seed: %v", err)
	}
	// gms/83_1 has widget-1 + widget-2; shared/all has widget-2 + widget-3.
	// Union = 3 files, and widget-2 resolves to the VERSION-SPECIFIC copy.
	if res.Subdomains["widgets"].Created != 3 {
		t.Fatalf("created = %d, want 3 (union of both roots)", res.Subdomains["widgets"].Created)
	}
	rows := sub.Rows()
	sort.Slice(rows, func(i, j int) bool { return rows[i].ID < rows[j].ID })
	want := []widgetRow{{ID: 1, Name: "one"}, {ID: 2, Name: "two"}, {ID: 3, Name: "three"}}
	if len(rows) != len(want) {
		t.Fatalf("rows = %+v, want %+v", rows, want)
	}
	for i := range rows {
		if rows[i] != want[i] {
			t.Fatalf("rows[%d] = %+v, want %+v (version-specific must win)", i, rows[i], want[i])
		}
	}
	if res.CatalogRevision != "shared-rev-xyz789+test-rev-abc123" {
		t.Fatalf("revision = %q, want composite", res.CatalogRevision)
	}
}

func TestSeed_MissingSharedRootIsANoOp(t *testing.T) {
	t.Cleanup(ResetMetricsForTest)
	db := openTestDB(t)
	src := NewFilesystemCatalogSourceWithShared("X_NO_ENV", goodFixtureRoot(t), "shared/does-not-exist")
	sub := &widgetSubdomain{}
	g := Group{
		Name:       "missing-shared",
		URLPrefix:  "/missing",
		Subdomains: []SubdomainAny{AdaptSubdomain[widgetAttrs, widgetRow](sub)},
	}
	ctx := tenant.WithContext(context.Background(), tenantGMS83(t))
	res, err := Seed(ctx, db, src, g)
	if err != nil {
		t.Fatalf("Seed: %v", err)
	}
	if res.Subdomains["widgets"].Created != 2 {
		t.Fatalf("created = %d, want 2 (version root only)", res.Subdomains["widgets"].Created)
	}
	if len(res.Subdomains["widgets"].Errors) != 0 {
		t.Fatalf("errors = %v, want none (absent root is not an error)", res.Subdomains["widgets"].Errors)
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `cd libs/atlas-seeder && go test ./... -run 'Shared|RevisionFor|MergesRoots|MissingSharedRoot|PlainSourceStillSingleRoot' -v`
Expected: FAIL — `undefined: NewFilesystemCatalogSourceWithShared`, `undefined: revisionFor`.

- [ ] **Step 4: Implement the shared root in `catalog.go`**

Replace `libs/atlas-seeder/catalog.go:22-31` with:

```go
type filesystemSource struct {
	envVar       string
	fallbackRoot string
	// sharedRel, when non-empty, is a base-relative root that is
	// neither region- nor version-qualified. It is returned FIRST from
	// Roots so version-specific entries override it (FR-1.4).
	sharedRel string
}

// NewFilesystemCatalogSource returns a CatalogSource that resolves its base
// directory from envVar when set, otherwise from fallbackRoot.
func NewFilesystemCatalogSource(envVar, fallbackRoot string) CatalogSource {
	return &filesystemSource{envVar: envVar, fallbackRoot: fallbackRoot}
}

// NewFilesystemCatalogSourceWithShared behaves like
// NewFilesystemCatalogSource but additionally resolves a shared,
// version-agnostic root at <base>/<sharedRel>. Seed and ReadStatus merge
// entries across every returned root, with later (more specific) roots
// winning on a relative-path collision.
//
// Only construct this when the service genuinely has version-agnostic
// catalog data. Folding a shared root into every consumer would change
// their composite CATALOG_REVISION and trigger a one-time spurious
// "seed catalog drift detected" warning fleet-wide.
func NewFilesystemCatalogSourceWithShared(envVar, fallbackRoot, sharedRel string) CatalogSource {
	return &filesystemSource{envVar: envVar, fallbackRoot: fallbackRoot, sharedRel: sharedRel}
}
```

Replace the `return []string{root}, nil` at `catalog.go:58` with:

```go
	if s.sharedRel == "" {
		return []string{root}, nil
	}
	return []string{filepath.Join(s.base(), s.sharedRel), root}, nil
```

- [ ] **Step 5: Implement merge + composite revision in `seed.go`**

In `libs/atlas-seeder/seed.go`, add `"sort"` and `"strings"` to the import block.

Replace `rev, _ := src.Revision(roots[0])` (line 56) with:

```go
	rev := revisionFor(src, roots)
```

Replace `counts := runSubdomain(gctx, db, src, roots[0], t, sd)` (line 64) with:

```go
			counts := runSubdomain(gctx, db, src, roots, t, sd)
```

Replace the signature and file-walk block of `runSubdomain` (lines 86-121) with:

```go
func runSubdomain(ctx context.Context, db *gorm.DB, src CatalogSource, roots []string, t tenant.Model, sd SubdomainAny) SubdomainCounts {
	var counts SubdomainCounts
	deleted, err := sd.DeleteAllForTenant(db.WithContext(ctx))
	if err != nil {
		counts.Errors = appendError(counts.Errors, fmt.Sprintf("delete: %v", err))
		return counts
	}
	counts.Deleted = deleted

	// Merge the subdomain's files across every root. Roots arrive
	// least-specific first, so a later root's copy of the same filename
	// overwrites the earlier one (FR-1.4). Walk returns (nil, nil) for a
	// missing directory, so a root that contributes nothing costs one
	// ReadDir ENOENT.
	owner := make(map[string]string)
	for _, root := range roots {
		files, err := src.Walk(root, sd.Path())
		if err != nil {
			counts.Errors = appendError(counts.Errors, fmt.Sprintf("walk %s: %v", sd.Path(), err))
			return counts
		}
		for _, name := range files {
			owner[name] = root
		}
	}
	names := make([]string, 0, len(owner))
	for name := range owner {
		names = append(names, name)
	}
	sort.Strings(names)

	pattern := sd.EntityIDPattern()
	for _, name := range names {
		if err := ctx.Err(); err != nil {
			counts.Errors = appendError(counts.Errors, fmt.Sprintf("%s: %v", name, err))
			return counts
		}
		rows, err := loadOne(ctx, src, owner[name], t, sd, pattern, name)
		if err != nil {
			counts.Failed++
			counts.Errors = appendError(counts.Errors, fmt.Sprintf("%s: %v", name, err))
			continue
		}
		if err := sd.BulkCreate(db.WithContext(ctx), rows); err != nil {
			counts.Failed++
			counts.Errors = appendError(counts.Errors, fmt.Sprintf("%s: bulkcreate: %v", name, err))
			continue
		}
		counts.Created += rowCount(rows)
	}
	return counts
}

// revisionFor joins each root's non-empty CATALOG_REVISION with "+" in
// root order. A single-root source yields exactly the string it yielded
// before shared roots existed, so the nine existing consumers see no
// change. A plain join (not a hash) keeps the value readable in logs.
func revisionFor(src CatalogSource, roots []string) string {
	parts := make([]string, 0, len(roots))
	for _, root := range roots {
		rev, err := src.Revision(root)
		if err != nil || rev == "" {
			continue
		}
		parts = append(parts, rev)
	}
	return strings.Join(parts, "+")
}
```

- [ ] **Step 6: Route `status.go` through the same helper**

Replace `libs/atlas-seeder/status.go:22-26` with:

```go
	roots, err := src.Roots(t)
	if err == nil && len(roots) > 0 {
		out.CatalogRevision = revisionFor(src, roots)
	}
```

- [ ] **Step 7: Run the tests**

Run: `cd libs/atlas-seeder && go test -race ./...`
Expected: PASS, including the pre-existing single-root tests.

- [ ] **Step 8: Commit**

```bash
git add libs/atlas-seeder/catalog.go libs/atlas-seeder/seed.go libs/atlas-seeder/status.go \
        libs/atlas-seeder/catalog_test.go libs/atlas-seeder/seed_test.go \
        libs/atlas-seeder/testdata/good/shared
git commit -m "feat(atlas-seeder): support a shared version-agnostic catalog root"
```

---

### Task 2: `libs/atlas-seeder` — `Group.AfterSeed` hook

**Files:**
- Modify: `libs/atlas-seeder/seeder.go`
- Modify: `libs/atlas-seeder/handlers.go:47-71`
- Test: `libs/atlas-seeder/handlers_test.go`

**Interfaces:**
- Consumes: `Group` and `Result` from Task 1's module (unchanged shapes).
- Produces: `Group.AfterSeed func(ctx context.Context, db *gorm.DB, res Result) error` — nil for the nine existing groups; invoked exactly once by `postSeed` after `Seed` returns a nil error, with the tenant-bearing background context. Its error is logged, never returned to the HTTP caller (the seed already committed).

- [ ] **Step 1: Write the failing test**

Append to `libs/atlas-seeder/handlers_test.go`:

```go
func TestRegisterRoutes_AfterSeedRunsOnceWithTenantContext(t *testing.T) {
	t.Cleanup(ResetMetricsForTest)
	t.Cleanup(backgroundSeeds.Wait)

	db := openTestDB(t)
	src := NewFilesystemCatalogSource("X_NO_ENV", goodFixtureRoot(t))

	var mu sync.Mutex
	calls := 0
	var sawTenant uuid.UUID
	var sawGroup string

	g := Group{
		Name:       "widgets-group",
		URLPrefix:  "/widgets",
		Subdomains: []SubdomainAny{AdaptSubdomain[widgetAttrs, widgetRow](&widgetSubdomain{})},
		AfterSeed: func(ctx context.Context, _ *gorm.DB, res Result) error {
			mu.Lock()
			defer mu.Unlock()
			calls++
			sawTenant = tenant.MustFromContext(ctx).Id()
			sawGroup = res.GroupName
			return nil
		},
	}
	r := mux.NewRouter()
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	RegisterRoutes(r, db, logger, src, g)
	srv := httptest.NewServer(r)
	defer srv.Close()

	req := requestWithTenant(t, "POST", srv.URL+"/widgets/seed", nil)
	wantTenant := req.Header.Get(tenant.ID)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer resp.Body.Close()
	backgroundSeeds.Wait()

	mu.Lock()
	defer mu.Unlock()
	if calls != 1 {
		t.Fatalf("AfterSeed called %d times, want exactly 1", calls)
	}
	if sawTenant.String() != wantTenant {
		t.Fatalf("AfterSeed tenant = %s, want %s", sawTenant, wantTenant)
	}
	if sawGroup != "widgets-group" {
		t.Fatalf("AfterSeed result.GroupName = %q, want %q", sawGroup, "widgets-group")
	}
}

func TestRegisterRoutes_NilAfterSeedIsANoOp(t *testing.T) {
	t.Cleanup(ResetMetricsForTest)
	t.Cleanup(backgroundSeeds.Wait)

	db := openTestDB(t)
	src := NewFilesystemCatalogSource("X_NO_ENV", goodFixtureRoot(t))
	g := Group{
		Name:       "widgets-group",
		URLPrefix:  "/widgets",
		Subdomains: []SubdomainAny{AdaptSubdomain[widgetAttrs, widgetRow](&widgetSubdomain{})},
	}
	r := mux.NewRouter()
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	RegisterRoutes(r, db, logger, src, g)
	srv := httptest.NewServer(r)
	defer srv.Close()

	resp, err := http.DefaultClient.Do(requestWithTenant(t, "POST", srv.URL+"/widgets/seed", nil))
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer resp.Body.Close()
	backgroundSeeds.Wait()

	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("status = %d, want 202", resp.StatusCode)
	}
}
```

Add `"context"`, `"sync"`, `"github.com/google/uuid"`, and `"gorm.io/gorm"` to `handlers_test.go`'s import block.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd libs/atlas-seeder && go test ./... -run AfterSeed -v`
Expected: FAIL — `unknown field AfterSeed in struct literal of type Group`.

- [ ] **Step 3: Add the field**

Replace `libs/atlas-seeder/seeder.go` in full:

```go
package seeder

import (
	"context"

	"gorm.io/gorm"
)

// Group declares one (POST /<prefix>/seed, GET /<prefix>/seed/status) pair.
type Group struct {
	Name       string // stored as seed_state.group_name; e.g. "drops"
	URLPrefix  string // e.g. "/drops" → routes POST /drops/seed
	Subdomains []SubdomainAny
	// AfterSeed, when non-nil, runs exactly once after a successful
	// Seed with the tenant-bearing seed context. Use it to emit a
	// domain event announcing that the group's data changed. Errors
	// are logged, not returned to the HTTP caller — the seed has
	// already committed by the time this runs.
	AfterSeed func(ctx context.Context, db *gorm.DB, res Result) error
}
```

- [ ] **Step 4: Invoke it from `postSeed`**

In `libs/atlas-seeder/handlers.go`, replace the body of the `routine.Go` closure (lines 52-68) with:

```go
			defer backgroundSeeds.Done()
			bgCtx := tenant.WithContext(context.Background(), t)
			res, err := Seed(bgCtx, db, src, g)
			if err != nil {
				l.WithError(err).WithFields(logrus.Fields{
					"tenant_id":  t.Id(),
					"group_name": g.Name,
				}).Error("Seed failed")
				return
			}
			l.WithFields(logrus.Fields{
				"tenant_id":        t.Id(),
				"group_name":       g.Name,
				"catalog_revision": res.CatalogRevision,
				"subdomains":       summarize(res.Subdomains),
			}).Info("Seed complete")
			if g.AfterSeed == nil {
				return
			}
			if err := g.AfterSeed(bgCtx, db, res); err != nil {
				l.WithError(err).WithFields(logrus.Fields{
					"tenant_id":  t.Id(),
					"group_name": g.Name,
				}).Error("AfterSeed hook failed; seeded data is committed but downstream consumers were not notified")
			}
```

- [ ] **Step 5: Run the tests**

Run: `cd libs/atlas-seeder && go test -race ./...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add libs/atlas-seeder/seeder.go libs/atlas-seeder/handlers.go libs/atlas-seeder/handlers_test.go
git commit -m "feat(atlas-seeder): add optional Group.AfterSeed hook"
```

---

### Task 3: `libs/atlas-tenant` — the derived-id formula

**Files:**
- Create: `libs/atlas-tenant/id.go`
- Create: `libs/atlas-tenant/id_test.go`

**Interfaces:**
- Consumes: nothing.
- Produces: `tenant.DerivedId(tenantId uuid.UUID, parts ...string) uuid.UUID` — UUIDv5 with the tenant id as namespace and `strings.Join(parts, "/")` as the name. Used by atlas-tenants (Task 8) and both atlas-transports config packages (Task 10). The pinned vectors in `id_test.go` are what stop a future edit from silently re-keying the Redis registry.

- [ ] **Step 1: Write the failing test**

`libs/atlas-tenant/id_test.go`:

```go
package tenant

import (
	"testing"

	"github.com/google/uuid"
)

// tenantA / tenantB are fixed so the expected UUIDs below are stable
// vectors, not values recomputed from the implementation under test.
var (
	tenantA = uuid.MustParse("ec876921-c363-4cc6-9c51-5bb8d57f9553")
	tenantB = uuid.MustParse("11111111-2222-3333-4444-555555555555")
)

func TestDerivedId_IsStableAcrossCalls(t *testing.T) {
	first := DerivedId(tenantA, "instance-routes", "temple-of-time-return-flight")
	second := DerivedId(tenantA, "instance-routes", "temple-of-time-return-flight")
	if first != second {
		t.Fatalf("DerivedId not stable: %s != %s", first, second)
	}
}

func TestDerivedId_DiffersAcrossTenants(t *testing.T) {
	a := DerivedId(tenantA, "routes", "boat-ellinia-orbis")
	b := DerivedId(tenantB, "routes", "boat-ellinia-orbis")
	if a == b {
		t.Fatalf("DerivedId collided across tenants: %s", a)
	}
}

func TestDerivedId_DiffersAcrossResourcesForSameSlug(t *testing.T) {
	scheduled := DerivedId(tenantA, "routes", "shared-slug")
	instance := DerivedId(tenantA, "instance-routes", "shared-slug")
	if scheduled == instance {
		t.Fatalf("DerivedId collided across resources: %s", scheduled)
	}
}

// Pinned vectors. If these change, every Redis route-registry key
// changes with them and the deployed registry silently duplicates.
// Do not "fix" this test by recomputing the expectations.
func TestDerivedId_PinnedVectors(t *testing.T) {
	cases := []struct {
		tenant uuid.UUID
		parts  []string
		want   string
	}{
		{tenantA, []string{"instance-routes", "temple-of-time-return-flight"}, uuid.NewSHA1(tenantA, []byte("instance-routes/temple-of-time-return-flight")).String()},
		{tenantA, []string{"routes", "boat-ellinia-orbis"}, uuid.NewSHA1(tenantA, []byte("routes/boat-ellinia-orbis")).String()},
		{tenantB, []string{"vessels", "subway-kc-nlc"}, uuid.NewSHA1(tenantB, []byte("vessels/subway-kc-nlc")).String()},
	}
	for _, c := range cases {
		got := DerivedId(c.tenant, c.parts...).String()
		if got != c.want {
			t.Errorf("DerivedId(%s, %v) = %s, want %s", c.tenant, c.parts, got, c.want)
		}
	}
}

func TestDerivedId_IsVersion5(t *testing.T) {
	id := DerivedId(tenantA, "routes", "x")
	if id.Version() != 5 {
		t.Fatalf("version = %d, want 5", id.Version())
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd libs/atlas-tenant && go test ./... -run DerivedId -v`
Expected: FAIL with `undefined: DerivedId`.

- [ ] **Step 3: Implement**

`libs/atlas-tenant/id.go`:

```go
package tenant

import (
	"strings"

	"github.com/google/uuid"
)

// DerivedId returns a stable UUIDv5 for a tenant-scoped entity, using the
// tenant id as the namespace and the "/"-joined parts as the name.
//
// It exists so a configuration entry identified by a human slug can have
// one deterministic UUID that every service computes identically: stable
// across repeated loads, replicas, restarts and re-seeds; different per
// tenant for the same slug; and different across resource types that
// happen to share a slug.
//
// This formula is load-bearing. It keys the atlas-transports Redis route
// registries, so changing it re-keys every entry and silently duplicates
// the deployed registry. libs/atlas-tenant/id_test.go pins known vectors
// precisely to make such a change fail loudly.
func DerivedId(tenantId uuid.UUID, parts ...string) uuid.UUID {
	return uuid.NewSHA1(tenantId, []byte(strings.Join(parts, "/")))
}
```

- [ ] **Step 4: Run the tests**

Run: `cd libs/atlas-tenant && go test -race ./...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-tenant/id.go libs/atlas-tenant/id_test.go
git commit -m "feat(atlas-tenant): add DerivedId, the stable per-tenant UUIDv5 formula"
```

---

### Task 4: Move transport seed data into the shared catalog

**Files:**
- Create: `deploy/seed/shared/all/CATALOG_REVISION`
- Create: `deploy/seed/shared/all/routes/*.json` (12), `deploy/seed/shared/all/vessels/*.json` (6), `deploy/seed/shared/all/instance-routes/*.json` (12)
- Delete: `services/atlas-tenants/configurations/routes/`, `.../vessels/`, `.../instance-routes/`
- Modify: `tools/catalog-lint/subdomains.go:12-28`

**Interfaces:**
- Consumes: nothing at compile time.
- Produces: on-disk catalog paths `shared/all/routes`, `shared/all/vessels`, `shared/all/instance-routes` with JSON:API-enveloped files whose `data.type` is `routes` / `vessels` / `instance-routes` and whose `data.id` is the slug. Task 6's subdomains read exactly these.

**Why `shared/all` and not `_shared`:** `tools/catalog-lint/main.go:33,55` skips any directory whose name starts with `_`, and all three CI workflows stamp `CATALOG_REVISION` with the glob `deploy/seed/*/*/` (`.github/workflows/main-publish.yml:287`, `pr-validation.yml:619`, `reconcile-bump-prs.yml:198`). An underscore-prefixed root would get no revision file, no envelope validation, and no error anywhere — the exact silent-failure class this task exists to remove. `shared/all` is region-shaped/version-shaped, so the existing glob and linter pick it up with no workflow change, while `filesystemSource.Roots` can never mistake it for a tenant root (it *formats* `<region>/<major>_<minor>` and `all` is not `%d_%d`).

- [ ] **Step 1: Create the shared tree and rewrap every file**

```bash
set -e
mkdir -p deploy/seed/shared/all/routes deploy/seed/shared/all/vessels deploy/seed/shared/all/instance-routes
cp deploy/seed/gms/83_1/CATALOG_REVISION deploy/seed/shared/all/CATALOG_REVISION

python3 - <<'PY'
import json, pathlib

src_base = pathlib.Path("services/atlas-tenants/configurations")
dst_base = pathlib.Path("deploy/seed/shared/all")
for res in ("routes", "vessels", "instance-routes"):
    for src in sorted((src_base / res).glob("*.json")):
        entry = json.loads(src.read_text())
        assert "data" not in entry, f"{src} is already enveloped"
        assert entry["type"] == res, f"{src} type {entry['type']} != {res}"
        dst = dst_base / res / src.name
        dst.write_text(json.dumps({"data": entry}, indent=2) + "\n")
        print(f"{src} -> {dst}")
PY
```

- [ ] **Step 2: Verify the rewrap kept every file and every field**

```bash
for res in routes vessels instance-routes; do
  a=$(ls services/atlas-tenants/configurations/$res/*.json | wc -l)
  b=$(ls deploy/seed/shared/all/$res/*.json | wc -l)
  echo "$res: src=$a dst=$b"
  [ "$a" = "$b" ] || { echo "COUNT MISMATCH in $res"; exit 1; }
done
python3 - <<'PY'
import json, pathlib, sys
ok = True
for res in ("routes", "vessels", "instance-routes"):
    for src in sorted(pathlib.Path(f"services/atlas-tenants/configurations/{res}").glob("*.json")):
        dst = pathlib.Path(f"deploy/seed/shared/all/{res}/{src.name}")
        if json.loads(src.read_text()) != json.loads(dst.read_text())["data"]:
            print("MISMATCH", src); ok = False
print("all entries byte-equivalent" if ok else "MISMATCHES FOUND")
sys.exit(0 if ok else 1)
PY
```

Expected: `routes: src=12 dst=12`, `vessels: src=6 dst=6`, `instance-routes: src=12 dst=12`, then `all entries byte-equivalent`.

- [ ] **Step 3: Delete the three migrated source directories**

`rps-rewards` and `mts-configs` stay in the image — they are explicitly out of scope and still read from `/configurations/<res>` via `configuration/seed.go`. That is why `Dockerfile:138` and `Dockerfile:163` (`COPY --from=build-env /app/configurations /configurations`) must **not** be touched.

```bash
git rm -r services/atlas-tenants/configurations/routes \
           services/atlas-tenants/configurations/vessels \
           services/atlas-tenants/configurations/instance-routes
ls services/atlas-tenants/configurations/
```

Expected: `mts-configs` and `rps-rewards` remain (`mts-configs` may not exist as a directory on this branch — if `ls` shows only `rps-rewards`, that is fine and expected; `configuration/seed.go` tolerates a missing directory by returning a load error rather than crashing).

- [ ] **Step 4: Teach `catalog-lint` about the three new subdomains**

In `tools/catalog-lint/subdomains.go`, insert into the `rules` slice, immediately before the `// widgets fixture used in tests` comment:

```go
	// Version-agnostic transport configuration, seeded from
	// deploy/seed/shared/all by atlas-tenants (task-189). pattern is nil
	// because the filename does not encode the entity id — the id lives
	// in data.id (e.g. flight-temple-of-time-leafre.json holds
	// "temple-of-time-return-flight").
	{path: "routes", typ: "routes", pattern: nil},
	{path: "vessels", typ: "vessels", pattern: nil},
	{path: "instance-routes", typ: "instance-routes", pattern: nil},
```

- [ ] **Step 5: Run the linter over the whole catalog**

Run: `go run ./tools/catalog-lint deploy/seed`
Expected: exit 0, no output.

- [ ] **Step 6: Commit**

```bash
git add deploy/seed/shared tools/catalog-lint/subdomains.go
git add -A services/atlas-tenants/configurations
git commit -m "refactor(seed): move transport configuration to deploy/seed/shared/all"
```

---

### Task 5: atlas-tenants — append/count administrators for the configurations row

**Files:**
- Modify: `services/atlas-tenants/atlas.com/tenants/configuration/administrator.go`
- Test: `services/atlas-tenants/atlas.com/tenants/configuration/administrator_test.go` (create)

**Interfaces:**
- Consumes: `configuration.Entity` (`entity.go:11-17`) — one row per `(tenant_id, resource_name)` whose `resource_data` is a JSON:API document with an **array** `data`.
- Produces:
  - `configuration.AppendConfigurationEntries(db *gorm.DB, tenantID uuid.UUID, resourceName string, entries []map[string]interface{}) error` — create-or-append, the same read-modify-write `CreateRoute` (`processor.go:181-273`) already performs, extracted once for all three subdomains.
  - `configuration.CountConfigurationEntries(db *gorm.DB, tenantID uuid.UUID, resourceName string) (int64, *time.Time, error)` — array length (or 1 for the legacy single-object shape, 0 when no row exists) plus the row's `updated_at`.

- [ ] **Step 1: Write the failing test**

`services/atlas-tenants/atlas.com/tenants/configuration/administrator_test.go`:

```go
package configuration_test

import (
	"testing"

	"atlas-tenants/configuration"
	"atlas-tenants/test"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// configTestDB reuses the service's shared in-memory sqlite helper
// (services/atlas-tenants/atlas.com/tenants/test/database.go), which
// already migrates tenant.Entity + configuration.Entity. The append and
// count administrators scope by an explicit tenant_id predicate, so no
// tenant GORM callback registration is needed here.
func configTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := test.SetupTestDB(t)
	t.Cleanup(func() { test.CleanupTestDB(db) })
	return db
}

func entry(id string) map[string]interface{} {
	return map[string]interface{}{
		"id":         id,
		"type":       "routes",
		"attributes": map[string]interface{}{"name": id},
	}
}

func TestAppendConfigurationEntries_CreatesThenAppends(t *testing.T) {
	db := configTestDB(t)
	tid := uuid.New()

	if err := configuration.AppendConfigurationEntries(db, tid, "routes", []map[string]interface{}{entry("a")}); err != nil {
		t.Fatalf("first append: %v", err)
	}
	if err := configuration.AppendConfigurationEntries(db, tid, "routes", []map[string]interface{}{entry("b")}); err != nil {
		t.Fatalf("second append: %v", err)
	}

	count, updatedAt, err := configuration.CountConfigurationEntries(db, tid, "routes")
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 2 {
		t.Fatalf("count = %d, want 2", count)
	}
	if updatedAt == nil {
		t.Fatal("updatedAt = nil, want the row's updated_at")
	}

	all, err := configuration.GetAllRoutesProvider(tid)(db)()
	if err != nil {
		t.Fatalf("GetAllRoutesProvider: %v", err)
	}
	if len(all) != 2 || all[0]["id"] != "a" || all[1]["id"] != "b" {
		t.Fatalf("entries = %+v, want [a b] in insertion order", all)
	}
}

func TestCountConfigurationEntries_NoRowIsZero(t *testing.T) {
	db := configTestDB(t)
	count, updatedAt, err := configuration.CountConfigurationEntries(db, uuid.New(), "routes")
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 0 {
		t.Fatalf("count = %d, want 0", count)
	}
	if updatedAt != nil {
		t.Fatalf("updatedAt = %v, want nil", updatedAt)
	}
}

// The most severe possible regression for this task is a cross-tenant
// leak, so it gets a dedicated test at the administrator layer.
func TestAppendConfigurationEntries_IsTenantScoped(t *testing.T) {
	db := configTestDB(t)
	a, b := uuid.New(), uuid.New()

	if err := configuration.AppendConfigurationEntries(db, a, "routes", []map[string]interface{}{entry("only-a")}); err != nil {
		t.Fatalf("append for A: %v", err)
	}

	countB, _, err := configuration.CountConfigurationEntries(db, b, "routes")
	if err != nil {
		t.Fatalf("count for B: %v", err)
	}
	if countB != 0 {
		t.Fatalf("tenant B count = %d, want 0 — tenant A's seed leaked", countB)
	}

	if _, err := configuration.DeleteConfigurationByResourceName(db, b, "routes"); err != nil {
		t.Fatalf("delete for B: %v", err)
	}
	countA, _, err := configuration.CountConfigurationEntries(db, a, "routes")
	if err != nil {
		t.Fatalf("count for A: %v", err)
	}
	if countA != 1 {
		t.Fatalf("tenant A count = %d, want 1 — tenant B's delete wiped A", countA)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-tenants/atlas.com/tenants && go test ./configuration/... -run 'AppendConfigurationEntries|CountConfigurationEntries' -v`
Expected: FAIL — `undefined: configuration.AppendConfigurationEntries`.

- [ ] **Step 3: Implement**

Append to `services/atlas-tenants/atlas.com/tenants/configuration/administrator.go` (add `"time"` and `"gorm.io/gorm/clause"` to the imports only if the compiler asks — the code below needs `"time"` and already-imported `encoding/json`, `errors`, `uuid`, `gorm`, `database`):

```go
// AppendConfigurationEntries appends entries to the single
// (tenant_id, resource_name) configuration row, creating the row when it
// does not exist yet. The stored shape is a JSON:API document whose
// "data" is an array of {id, type, attributes} objects — the same shape
// CreateRoute/CreateVessel/CreateInstanceRoute produce, so a seeded row
// is indistinguishable from a hand-created one.
func AppendConfigurationEntries(db *gorm.DB, tenantID uuid.UUID, resourceName string, entries []map[string]interface{}) error {
	if len(entries) == 0 {
		return nil
	}
	existing, err := GetByTenantIdAndResourceNameProvider(tenantID, resourceName)(db)()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		payload, mErr := json.Marshal(map[string]interface{}{"data": entries})
		if mErr != nil {
			return mErr
		}
		return CreateConfiguration(db, Entity{
			ID:           uuid.New(),
			TenantId:     tenantID,
			ResourceName: resourceName,
			ResourceData: payload,
		})
	}
	if err != nil {
		return err
	}

	var document map[string]interface{}
	if err := json.Unmarshal(existing.ResourceData, &document); err != nil {
		return err
	}
	var merged []interface{}
	switch data := document["data"].(type) {
	case []interface{}:
		merged = data
	case map[string]interface{}:
		// Legacy single-object shape: promote it to an array so the
		// append below is uniform.
		merged = []interface{}{data}
	}
	for _, e := range entries {
		merged = append(merged, e)
	}
	document["data"] = merged

	payload, err := json.Marshal(document)
	if err != nil {
		return err
	}
	existing.ResourceData = payload
	return UpdateConfiguration(db, existing)
}

// CountConfigurationEntries returns the number of entries stored in the
// tenant's configuration row for resourceName, along with the row's
// updated_at. A missing row is (0, nil, nil) — not an error — because an
// unseeded tenant is a normal state the status endpoint must report.
func CountConfigurationEntries(db *gorm.DB, tenantID uuid.UUID, resourceName string) (int64, *time.Time, error) {
	existing, err := GetByTenantIdAndResourceNameProvider(tenantID, resourceName)(db)()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, nil, nil
	}
	if err != nil {
		return 0, nil, err
	}
	var document map[string]interface{}
	if err := json.Unmarshal(existing.ResourceData, &document); err != nil {
		return 0, nil, err
	}
	updatedAt := existing.UpdatedAt
	switch data := document["data"].(type) {
	case []interface{}:
		return int64(len(data)), &updatedAt, nil
	case map[string]interface{}:
		return 1, &updatedAt, nil
	default:
		return 0, &updatedAt, nil
	}
}
```

- [ ] **Step 4: Run the tests**

Run: `cd services/atlas-tenants/atlas.com/tenants && go test -race ./configuration/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-tenants/atlas.com/tenants/configuration/administrator.go \
        services/atlas-tenants/atlas.com/tenants/configuration/administrator_test.go
git commit -m "feat(atlas-tenants): add append/count administrators for configuration rows"
```

---

### Task 6: atlas-tenants — adopt `libs/atlas-seeder` for the three transport resources

**Files:**
- Create: `services/atlas-tenants/atlas.com/tenants/configuration/seed/subdomain.go`
- Create: `services/atlas-tenants/atlas.com/tenants/configuration/seed/groups.go`
- Create: `services/atlas-tenants/atlas.com/tenants/configuration/seed/groups_test.go`
- Modify: `services/atlas-tenants/atlas.com/tenants/main.go:44-85`
- Modify: `services/atlas-tenants/atlas.com/tenants/go.mod`
- Modify: `services/atlas-tenants/atlas.com/tenants/configuration/seed.go` (delete the three transport loaders + path helpers)
- Modify: `services/atlas-tenants/atlas.com/tenants/configuration/processor.go:149-160,1151-1262` (delete the three `Seed*` methods + their interface entries)
- Modify: `services/atlas-tenants/atlas.com/tenants/configuration/mock/processor.go:63-67,487-510` (delete the three mock funcs)
- Modify: `services/atlas-tenants/atlas.com/tenants/configuration/resource.go:817-878,1259,1267,1275` (delete the three handlers + registrations)

**Interfaces:**
- Consumes: `seeder.NewFilesystemCatalogSourceWithShared` and merge-across-roots (Task 1); `seeder.Group.AfterSeed` (Task 2); `configuration.AppendConfigurationEntries` / `CountConfigurationEntries` (Task 5); the shared catalog tree (Task 4).
- Produces:
  - `seed.InitResource(db *gorm.DB) server.RouteInitializer` — registers all six endpoints.
  - Endpoints (under the `/api/` base path set at `main.go:78`): `POST|GET /api/tenants/configurations/{routes,vessels,instance-routes}/seed[/status]`.
  - `seeder.Group.Name` values `routes` / `vessels` / `instance-routes`; subdomain `Name()` values identical — these are the keys atlas-ui reads in Task 12.
- Removed: `Processor.SeedRoutes`, `Processor.SeedVessels`, `Processor.SeedInstanceRoutes`, `configuration.LoadRouteFiles`, `LoadVesselFiles`, `LoadInstanceRouteFiles`, `getRoutesPath`, `getVesselsPath`, `getInstanceRoutesPath`, and the three `POST /tenants/{tenantId}/configurations/<res>/seed` routes. `SeedRpsRewards` / `SeedMtsConfigs` and their routes are untouched.

**No mux collision:** the new literal path `/tenants/configurations/routes/seed` has `configurations` as its **second** segment, while every surviving CRUD pattern is `/tenants/{tenantId}/configurations/<res>[/{id}]` with `configurations` as its **third**. No pattern can match. The seed initializer is nonetheless registered before `configuration.RegisterRoutes` as belt-and-braces, and Step 7's route test asserts both directions.

- [ ] **Step 1: Add the module dependency**

```bash
cd services/atlas-tenants/atlas.com/tenants
go mod edit -require=github.com/Chronicle20/atlas/libs/atlas-seeder@v0.0.0
go mod edit -replace=github.com/Chronicle20/atlas/libs/atlas-seeder=../../../../libs/atlas-seeder
```

Do **not** run `go work sync`. Run `go mod tidy` only after Step 4 compiles.

- [ ] **Step 2: Write the failing test**

`services/atlas-tenants/atlas.com/tenants/configuration/seed/groups_test.go`:

```go
package seed_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"atlas-tenants/configuration"
	"atlas-tenants/configuration/seed"
	"atlas-tenants/test"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"io"

	"github.com/sirupsen/logrus"

	outbox "github.com/Chronicle20/atlas/libs/atlas-outbox"
	seeder "github.com/Chronicle20/atlas/libs/atlas-seeder"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func newRouter(t *testing.T) *mux.Router {
	t.Helper()
	// test.SetupTestDB migrates tenant.Entity + configuration.Entity;
	// seed_state and the outbox are this task's additions.
	db := test.SetupTestDB(t)
	t.Cleanup(func() { test.CleanupTestDB(db) })
	if err := db.AutoMigrate(&seeder.SeedState{}); err != nil {
		t.Fatalf("migrate seed_state: %v", err)
	}
	if err := outbox.Migration(db); err != nil {
		t.Fatalf("migrate outbox: %v", err)
	}
	l := logrus.New()
	l.SetOutput(io.Discard)
	r := mux.NewRouter()
	seed.InitResource(db)(r, l)
	configuration.RegisterRoutes(db)(stubServerInformation{})(r, l)
	return r
}

type stubServerInformation struct{}

func (stubServerInformation) GetBaseURL() string { return "" }
func (stubServerInformation) GetPrefix() string  { return "/api/" }

func withTenantHeaders(req *http.Request) *http.Request {
	req.Header.Set(tenant.ID, uuid.New().String())
	req.Header.Set(tenant.Region, "GMS")
	req.Header.Set(tenant.MajorVersion, "83")
	req.Header.Set(tenant.MinorVersion, "1")
	return req
}

func TestSeedRoutesDispatch(t *testing.T) {
	r := newRouter(t)
	for _, res := range []string{"routes", "vessels", "instance-routes"} {
		for _, c := range []struct {
			method string
			path   string
			want   int
		}{
			{http.MethodPost, "/tenants/configurations/" + res + "/seed", http.StatusAccepted},
			{http.MethodGet, "/tenants/configurations/" + res + "/seed/status", http.StatusOK},
		} {
			rr := httptest.NewRecorder()
			r.ServeHTTP(rr, withTenantHeaders(httptest.NewRequest(c.method, c.path, nil)))
			if rr.Code != c.want {
				t.Errorf("%s %s = %d, want %d", c.method, c.path, rr.Code, c.want)
			}
		}
	}
}

func TestSeedEndpointsRequireTenantHeaders(t *testing.T) {
	r := newRouter(t)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/tenants/configurations/routes/seed", nil))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 without tenant headers", rr.Code)
	}
}

// The literal seed paths must not shadow the surviving path-scoped CRUD
// routes, and vice versa.
func TestCrudRoutesStillDispatch(t *testing.T) {
	r := newRouter(t)
	path := "/tenants/" + uuid.New().String() + "/configurations/routes"
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, path, nil))
	if rr.Code == http.StatusNotFound {
		t.Fatalf("GET %s returned 404 — the seed routes shadowed the CRUD handler", path)
	}
}

// The three removed path-scoped seed endpoints must be gone; the two
// out-of-scope ones must remain.
func TestRemovedAndRetainedSeedEndpoints(t *testing.T) {
	r := newRouter(t)
	tid := uuid.New().String()
	for _, res := range []string{"routes", "vessels", "instance-routes"} {
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/tenants/"+tid+"/configurations/"+res+"/seed", nil))
		if rr.Code != http.StatusNotFound {
			t.Errorf("POST path-scoped %s/seed = %d, want 404 (endpoint must be removed)", res, rr.Code)
		}
	}
	for _, res := range []string{"rps-rewards", "mts-configs"} {
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/tenants/"+tid+"/configurations/"+res+"/seed", nil))
		if rr.Code == http.StatusNotFound {
			t.Errorf("POST path-scoped %s/seed = 404, want it retained (out of scope)", res)
		}
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `cd services/atlas-tenants/atlas.com/tenants && go test ./configuration/seed/... -v`
Expected: FAIL — package `atlas-tenants/configuration/seed` does not exist.

- [ ] **Step 4: Implement the subdomain**

`services/atlas-tenants/atlas.com/tenants/configuration/seed/subdomain.go`:

```go
package seed

import (
	"encoding/json"
	"fmt"
	"regexp"
	"time"

	"gorm.io/gorm"

	seeder "github.com/Chronicle20/atlas/libs/atlas-seeder"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"

	"atlas-tenants/configuration"
)

// Entry is the decoded JSON:API data object from one catalog file:
// {id, type, attributes}. It is stored verbatim as one element of the
// configuration row's resource_data.data array, so a seeded entry is
// byte-identical to one created through the CRUD API.
type Entry = map[string]interface{}

// Subdomain adapts one tenant-configuration resource onto
// libs/atlas-seeder. All three transport resources share one storage
// shape — a single (tenant, resource_name) row whose resource_data.data
// is an array — so they share one implementation parameterised by name.
type Subdomain struct {
	// resource is simultaneously the seeder subdomain name, the catalog
	// subdirectory, the JSON:API data.type, and the configurations
	// row's resource_name. Keeping them identical is what lets the UI
	// read the status map by resource name.
	resource string
}

var _ seeder.Subdomain[Entry, Entry] = Subdomain{}

func NewSubdomain(resource string) Subdomain { return Subdomain{resource: resource} }

func (s Subdomain) Name() string { return s.resource }
func (s Subdomain) Path() string { return s.resource }
func (s Subdomain) Type() string { return s.resource }

// EntityIDPattern is nil: the catalog filename does not encode the
// entity id. flight-temple-of-time-leafre.json holds
// "temple-of-time-return-flight", so the id comes from data.id
// (libs/atlas-seeder/seed.go handles the nil-pattern case).
func (s Subdomain) EntityIDPattern() *regexp.Regexp { return nil }

func (s Subdomain) DeleteAllForTenant(db *gorm.DB) (int64, error) {
	t, err := tenantFrom(db)
	if err != nil {
		return 0, err
	}
	return configuration.DeleteConfigurationByResourceName(db, t.Id(), s.resource)
}

func (s Subdomain) Decode(payload []byte) (Entry, error) {
	env, err := seeder.ParseEnvelope(payload)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", s.resource, err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(payload, &raw); err != nil {
		return nil, fmt.Errorf("%s: parse payload: %w", s.resource, err)
	}
	var entry Entry
	if err := json.Unmarshal(raw["data"], &entry); err != nil {
		return nil, fmt.Errorf("%s: parse data object: %w", s.resource, err)
	}
	if env.Data.Type != s.resource {
		return nil, fmt.Errorf("%s: data.type = %q", s.resource, env.Data.Type)
	}
	return entry, nil
}

func (s Subdomain) Build(_ tenant.Model, entityID string, entry Entry) ([]Entry, error) {
	if entry == nil {
		return nil, fmt.Errorf("%s: nil entry for %q", s.resource, entityID)
	}
	entry["id"] = entityID
	return []Entry{entry}, nil
}

// BulkCreate appends this file's single entry to the tenant's
// configuration row. seeder.Seed holds a per-(tenant, group) mutex for
// the whole run and the three resources live in different groups, so
// these read-modify-writes never contend.
func (s Subdomain) BulkCreate(db *gorm.DB, entries []Entry) error {
	t, err := tenantFrom(db)
	if err != nil {
		return err
	}
	return configuration.AppendConfigurationEntries(db, t.Id(), s.resource, entries)
}

func (s Subdomain) Count(db *gorm.DB) (int64, *time.Time, error) {
	t, err := tenantFrom(db)
	if err != nil {
		return 0, nil, err
	}
	return configuration.CountConfigurationEntries(db, t.Id(), s.resource)
}

// tenantFrom reads the tenant off the *gorm.DB's context. libs/atlas-seeder
// hands every Subdomain method a db already carrying the seed context
// (seed.go's db.WithContext(ctx)), and Seed itself guarantees that context
// has a tenant. Failing loudly here beats silently seeding the zero tenant.
func tenantFrom(db *gorm.DB) (tenant.Model, error) {
	t, err := tenant.FromContext(db.Statement.Context)()
	if err != nil {
		return tenant.Model{}, fmt.Errorf("seed: no tenant in context: %w", err)
	}
	return t, nil
}
```

- [ ] **Step 5: Implement the groups and route initializer**

`services/atlas-tenants/atlas.com/tenants/configuration/seed/groups.go`:

```go
package seed

import (
	"context"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	outbox "github.com/Chronicle20/atlas/libs/atlas-outbox"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	seeder "github.com/Chronicle20/atlas/libs/atlas-seeder"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"

	"atlas-tenants/configuration"
	"atlas-tenants/kafka/message"
)

// seededResourceId is the ResourceId carried by the single
// configuration-status event a seed run emits. atlas-transports switches
// on ResourceType and uses ResourceId only for logging, so a synthetic
// value is correct and reads clearly in a log line.
const seededResourceId = "*"

// InitResource registers POST /<prefix>/seed and GET /<prefix>/seed/status
// for each transport configuration resource. Register this BEFORE
// configuration.RegisterRoutes so the literal seed paths are matched ahead
// of the /tenants/{tenantId}/... patterns.
func InitResource(db *gorm.DB) server.RouteInitializer {
	return func(router *mux.Router, l logrus.FieldLogger) {
		// The three transport resources are version-agnostic: one shared
		// set of files applies to every tenant regardless of region or
		// client version, so the source resolves deploy/seed/shared/all
		// in addition to the tenant's version root.
		src := seeder.NewFilesystemCatalogSourceWithShared("SEED_CATALOG_ROOT", "./deploy/seed", "shared/all")
		for _, g := range Groups(l) {
			seeder.RegisterRoutes(router, db, l, src, g)
		}
	}
}

// Groups returns one seeder.Group per transport configuration resource.
// Separate groups (rather than one group with three subdomains) keep each
// resource's seed_state revision independent and let the Setup page seed
// them individually, as FR-2.1 requires.
func Groups(l logrus.FieldLogger) []seeder.Group {
	return []seeder.Group{
		newGroup(l, "routes", configuration.EventTypeRouteUpdated, configuration.CreateRouteStatusEventProvider),
		newGroup(l, "vessels", configuration.EventTypeVesselUpdated, configuration.CreateVesselStatusEventProvider),
		newGroup(l, "instance-routes", configuration.EventTypeInstanceRouteUpdated, configuration.CreateInstanceRouteStatusEventProvider),
	}
}

func newGroup(
	l logrus.FieldLogger,
	resource string,
	eventType string,
	provider func(tenantId uuid.UUID, eventType string, resourceId string) model.Provider[[]kafka.Message],
) seeder.Group {
	return seeder.Group{
		Name:      resource,
		URLPrefix: "/tenants/configurations/" + resource,
		Subdomains: []seeder.SubdomainAny{
			seeder.AdaptSubdomain[Entry, Entry](NewSubdomain(resource)),
		},
		AfterSeed: afterSeed(l, eventType, provider),
	}
}

// afterSeed enqueues exactly ONE configuration-status event per seed run.
// The atlas-transports handler for these events is ClearTenant + full
// reload, so one event per file would trigger N reloads and — worse — a
// reload could land mid-seed and load a partial set.
//
// The context here is libs/atlas-seeder's tenant-bearing background
// context, so outbox.EnqueueBuffer snapshots the four tenant headers by
// construction. That is the whole point of routing the emit through
// AfterSeed rather than through BulkCreate.
func afterSeed(
	l logrus.FieldLogger,
	eventType string,
	provider func(tenantId uuid.UUID, eventType string, resourceId string) model.Provider[[]kafka.Message],
) func(context.Context, *gorm.DB, seeder.Result) error {
	return func(ctx context.Context, db *gorm.DB, _ seeder.Result) error {
		t, err := tenant.FromContext(ctx)()
		if err != nil {
			return err
		}
		return database.ExecuteTransaction(db.WithContext(ctx), func(tx *gorm.DB) error {
			return message.Emit(outbox.EmitProvider(l, ctx, tx))(func(mb *message.Buffer) error {
				return mb.Put(configuration.EventTopicConfigurationStatus, provider(t.Id(), eventType, seededResourceId))
			})
		})
	}
}
```

- [ ] **Step 6: Wire it into `main.go`**

In `services/atlas-tenants/atlas.com/tenants/main.go`, add `"atlas-tenants/configuration/seed"` and `seeder "github.com/Chronicle20/atlas/libs/atlas-seeder"` to the imports, then:

Replace line 48 with:

```go
	db := database.Connect(l, database.SetMigrations(
		tenant.MigrateEntities,
		configuration.MigrateEntities,
		outboxlib.Migration,
		func(db *gorm.DB) error { return db.AutoMigrate(&seeder.SeedState{}) },
	))
```

(add `"gorm.io/gorm"` to the imports for the closure's parameter type)

Replace line 80 with:

```go
		AddRouteInitializer(seed.InitResource(db)).
		AddRouteInitializer(configuration.RegisterRoutes(db)(GetServer())).
```

- [ ] **Step 7: Delete the bespoke seed path**

1. `configuration/seed.go` — delete `defaultRoutesPath`, `defaultInstanceRoutesPath`, `defaultVesselsPath`, `getRoutesPath`, `getInstanceRoutesPath`, `getVesselsPath`, `LoadRouteFiles`, `LoadInstanceRouteFiles`, `LoadVesselFiles`. Keep `SeedResult`, `defaultRpsRewardsPath`, `defaultMtsConfigsPath`, `getRpsRewardsPath`, `getMtsConfigsPath`, `LoadRpsRewardFiles`, `LoadMtsConfigFiles`, `loadSeedFiles`.
2. `configuration/processor.go` — delete the `SeedRoutes` / `SeedInstanceRoutes` / `SeedVessels` entries from the `Processor` interface (lines 150-155) and their implementations (lines 1151-1262). Keep `SeedRpsRewards` and `SeedMtsConfigs`.
3. `configuration/mock/processor.go` — delete `SeedRoutesFunc`, `SeedInstanceRoutesFunc`, `SeedVesselsFunc` and their three methods.
4. `configuration/resource.go` — delete `SeedRoutesHandler`, `SeedInstanceRoutesHandler`, `SeedVesselsHandler` (lines 817-878) and the three `r.HandleFunc(...)` registrations at lines 1259, 1267, 1275.

- [ ] **Step 8: Tidy and build**

```bash
cd services/atlas-tenants/atlas.com/tenants
go mod tidy
go build ./...
```

Expected: clean. If `go build` reports an unused import in `seed.go` or `resource.go`, remove it.

- [ ] **Step 9: Run the tests**

Run: `cd services/atlas-tenants/atlas.com/tenants && go test -race ./... && go vet ./...`
Expected: PASS, including the new `configuration/seed` package tests.

- [ ] **Step 10: Commit**

```bash
git add services/atlas-tenants/atlas.com/tenants
git commit -m "feat(atlas-tenants): seed transport configuration via libs/atlas-seeder"
```

---

### Task 7: atlas-tenants — a real tenant in every configuration emit context

**Files:**
- Modify: `services/atlas-tenants/atlas.com/tenants/configuration/processor.go` (all 18 `…AndEmit` methods)
- Test: `services/atlas-tenants/atlas.com/tenants/configuration/emit_tenant_test.go` (create)

**Interfaces:**
- Consumes: `tenant.Processor.GetById` (`services/atlas-tenants/atlas.com/tenants/tenant/processor.go:251`) and `tenant.Create` from `libs/atlas-tenant` (`processor.go:31`). The service-local package is imported as `tenants "atlas-tenants/tenant"`; the library as `atlastenant "github.com/Chronicle20/atlas/libs/atlas-tenant"`.
- Produces: unexported `func (p *ProcessorImpl) tenantCtx(tenantId uuid.UUID) (context.Context, error)`.

**Root cause being fixed:** `ProcessorImpl` is constructed as `NewProcessor(l, ctx, db)` with the tenant-free server context, and every `…AndEmit` enqueues through `outbox.EmitProvider(p.l, p.ctx, tx)`. `outbox.EnqueueBuffer` snapshots headers from that context, and `producer.TenantHeaderDecorator` (`libs/atlas-kafka/producer/header.go:34-37`) returns **empty headers with a nil error** when the context has no tenant. The consumer's `TenantHeaderParser` then installs the **zero** tenant rather than none, which is why the reload ran against `00000000-0000-0000-0000-000000000000` instead of panicking.

**Why not the idiomatic fix:** dropping `tenantId uuid.UUID` from the `Processor` interface and reading the tenant from context is the right end-state, but it rewrites a 1731-line processor, a 1305-line resource file, and their tests — and the REST layer is path-scoped (`/tenants/{tenantId}/…`), so the handlers would have to build the context anyway. Recorded as future cleanup; not done here.

**Watch for:** putting a tenant into the context also activates `libs/atlas-database`'s tenant callback for any query on `configurations` (it has a `tenant_id` column). The added `tenant_id = <ctx tenant>` predicate is identical to the explicit one the existing code already passes, because the context tenant is built from the same path-scoped `tenantId`. Step 4 asserts this.

- [ ] **Step 1: Write the failing test**

`services/atlas-tenants/atlas.com/tenants/configuration/emit_tenant_test.go`:

```go
package configuration_test

import (
	"context"
	"encoding/json"
	"testing"

	"atlas-tenants/configuration"
	tenants "atlas-tenants/tenant"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	outbox "github.com/Chronicle20/atlas/libs/atlas-outbox"
	atlastenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// newEmitTestDB adds the outbox table to the shared test database
// (test.SetupTestDB already migrates tenant.Entity + configuration.Entity)
// so an …AndEmit call can be observed at the outbox row it writes. This is
// the same pattern rankings_handler_test.go already uses.
func newEmitTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := configTestDB(t) // from administrator_test.go
	if err := outbox.Migration(db); err != nil {
		t.Fatalf("migrate outbox: %v", err)
	}
	return db
}

func seedTenantRow(t *testing.T, db *gorm.DB) uuid.UUID {
	t.Helper()
	l := logrus.New()
	l.SetLevel(logrus.PanicLevel)
	m, err := tenants.NewProcessor(l, context.Background(), db).Create("emit-test", "GMS", 83, 1)
	if err != nil {
		t.Fatalf("create tenant: %v", err)
	}
	return m.Id()
}

// The regression this task exists for: a configuration-status event must
// carry all four tenant headers. Asserting at the outbox row covers the
// enqueue-time header snapshot that actually failed in production.
func TestCreateRouteAndEmit_OutboxRowCarriesTenantHeaders(t *testing.T) {
	db := newEmitTestDB(t)
	tid := seedTenantRow(t, db)

	l := logrus.New()
	l.SetLevel(logrus.PanicLevel)
	p := configuration.NewProcessor(l, context.Background(), db)

	if _, err := p.CreateRouteAndEmit(tid, map[string]interface{}{
		"id":         "boat-ellinia-orbis",
		"type":       "routes",
		"attributes": map[string]interface{}{"name": "boat-ellinia-orbis"},
	}); err != nil {
		t.Fatalf("CreateRouteAndEmit: %v", err)
	}

	var rows []outbox.Entity
	if err := db.Find(&rows).Error; err != nil {
		t.Fatalf("read outbox: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("outbox rows = %d, want 1", len(rows))
	}

	var headers map[string]string
	if err := json.Unmarshal(rows[0].Headers, &headers); err != nil {
		t.Fatalf("decode headers: %v", err)
	}
	if headers[atlastenant.ID] != tid.String() {
		t.Errorf("%s = %q, want %q", atlastenant.ID, headers[atlastenant.ID], tid.String())
	}
	if headers[atlastenant.Region] != "GMS" {
		t.Errorf("%s = %q, want %q", atlastenant.Region, headers[atlastenant.Region], "GMS")
	}
	if _, ok := headers[atlastenant.MajorVersion]; !ok {
		t.Errorf("%s header missing", atlastenant.MajorVersion)
	}
	if _, ok := headers[atlastenant.MinorVersion]; !ok {
		t.Errorf("%s header missing", atlastenant.MinorVersion)
	}
}

// An unknown tenant must abort the write rather than emit tenant-free:
// the operation is meaningless for a tenant that does not exist, and a
// tenant-free emit is exactly the defect being closed.
func TestCreateRouteAndEmit_UnknownTenantFails(t *testing.T) {
	db := newEmitTestDB(t)
	l := logrus.New()
	l.SetLevel(logrus.PanicLevel)
	p := configuration.NewProcessor(l, context.Background(), db)

	if _, err := p.CreateRouteAndEmit(uuid.New(), map[string]interface{}{
		"id":         "ghost",
		"type":       "routes",
		"attributes": map[string]interface{}{"name": "ghost"},
	}); err == nil {
		t.Fatal("CreateRouteAndEmit(unknown tenant) returned nil error, want failure")
	}

	var count int64
	if err := db.Model(&outbox.Entity{}).Count(&count).Error; err != nil {
		t.Fatalf("count outbox: %v", err)
	}
	if count != 0 {
		t.Fatalf("outbox rows = %d, want 0 (nothing may be emitted for an unknown tenant)", count)
	}
}

// Rankings emits through the same tenant-free context and was verified
// during design to share the defect (processor.go CreateRankingsAndEmit).
func TestCreateRankingsAndEmit_OutboxRowCarriesTenantHeaders(t *testing.T) {
	db := newEmitTestDB(t)
	tid := seedTenantRow(t, db)

	l := logrus.New()
	l.SetLevel(logrus.PanicLevel)
	p := configuration.NewProcessor(l, context.Background(), db)

	if _, err := p.CreateRankingsAndEmit(tid, map[string]interface{}{
		"id":         "rankings",
		"type":       "rankings",
		"attributes": map[string]interface{}{},
	}); err != nil {
		t.Fatalf("CreateRankingsAndEmit: %v", err)
	}

	var rows []outbox.Entity
	if err := db.Find(&rows).Error; err != nil {
		t.Fatalf("read outbox: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("outbox rows = %d, want 1", len(rows))
	}
	var headers map[string]string
	if err := json.Unmarshal(rows[0].Headers, &headers); err != nil {
		t.Fatalf("decode headers: %v", err)
	}
	if headers[atlastenant.ID] != tid.String() {
		t.Errorf("%s = %q, want %q", atlastenant.ID, headers[atlastenant.ID], tid.String())
	}
}
```

If `tenants.NewProcessor(...).Create` has a different signature on this branch, read `services/atlas-tenants/atlas.com/tenants/tenant/processor.go` and adapt the call — the assertion is what matters, not the constructor.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-tenants/atlas.com/tenants && go test ./configuration/... -run 'CarriesTenantHeaders|UnknownTenantFails' -v`
Expected: FAIL — headers map is empty (`TENANT_ID` missing), and `UnknownTenantFails` gets a nil error.

- [ ] **Step 3: Add the helper**

In `services/atlas-tenants/atlas.com/tenants/configuration/processor.go`, add these imports:

```go
	tenants "atlas-tenants/tenant"

	atlastenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
```

and add, immediately after `var _ Processor = (*ProcessorImpl)(nil)` (line 178):

```go
// tenantCtx returns p.ctx enriched with a fully-populated tenant.Model
// for tenantId.
//
// The configuration processor threads the tenant as a bare uuid.UUID
// (the REST layer is path-scoped), so the server context it was built
// with has no tenant at all. Without this, producer.TenantHeaderDecorator
// silently drops all four tenant headers and the downstream reload runs
// against the zero tenant. atlas-tenants owns the tenants table, so it
// has the region and versions needed to build a real model.
//
// A resolution failure aborts the caller rather than emitting
// tenant-free: the operation is meaningless for an unknown tenant.
func (p *ProcessorImpl) tenantCtx(tenantId uuid.UUID) (context.Context, error) {
	m, err := tenants.NewProcessor(p.l, p.ctx, p.db).GetById(tenantId)
	if err != nil {
		return nil, fmt.Errorf("resolve tenant %s for configuration emit: %w", tenantId, err)
	}
	t, err := atlastenant.Create(m.Id(), m.Region(), m.MajorVersion(), m.MinorVersion())
	if err != nil {
		return nil, fmt.Errorf("build tenant model %s: %w", tenantId, err)
	}
	return atlastenant.WithContext(p.ctx, t), nil
}
```

- [ ] **Step 4: Thread it through all 18 `…AndEmit` methods**

Every `…AndEmit` currently has this shape:

```go
func (p *ProcessorImpl) CreateRouteAndEmit(tenantId uuid.UUID, route map[string]interface{}) (Model, error) {
	var result Model
	txErr := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		var err error
		result, err = message.EmitWithResult[Model, uuid.UUID](outbox.EmitProvider(p.l, p.ctx, tx))(func(mb *message.Buffer) func(uuid.UUID) (Model, error) {
			return func(tenantId uuid.UUID) (Model, error) {
				return NewProcessor(p.l, p.ctx, tx).CreateRoute(mb)(tenantId)(route)
			}
		})(tenantId)
		return err
	})
	return result, txErr
}
```

Rewrite each to resolve the context first and use it in all three places (`p.db.WithContext`, `outbox.EmitProvider`, and the inner `NewProcessor`):

```go
func (p *ProcessorImpl) CreateRouteAndEmit(tenantId uuid.UUID, route map[string]interface{}) (Model, error) {
	ctx, err := p.tenantCtx(tenantId)
	if err != nil {
		return Model{}, err
	}
	var result Model
	txErr := database.ExecuteTransaction(p.db.WithContext(ctx), func(tx *gorm.DB) error {
		var err error
		result, err = message.EmitWithResult[Model, uuid.UUID](outbox.EmitProvider(p.l, ctx, tx))(func(mb *message.Buffer) func(uuid.UUID) (Model, error) {
			return func(tenantId uuid.UUID) (Model, error) {
				return NewProcessor(p.l, ctx, tx).CreateRoute(mb)(tenantId)(route)
			}
		})(tenantId)
		return err
	})
	return result, txErr
}
```

Delete-shaped methods return only `error`, so their early return is `return err`:

```go
func (p *ProcessorImpl) DeleteRouteAndEmit(tenantId uuid.UUID, routeID string) error {
	ctx, err := p.tenantCtx(tenantId)
	if err != nil {
		return err
	}
	return database.ExecuteTransaction(p.db.WithContext(ctx), func(tx *gorm.DB) error {
		return message.Emit(outbox.EmitProvider(p.l, ctx, tx))(func(mb *message.Buffer) error {
			return NewProcessor(p.l, ctx, tx).DeleteRoute(mb)(tenantId)(routeID)
		})
	})
}
```

Apply the same transformation to every one of the 18: `Create/Update/Delete` × `Route`, `Vessel`, `InstanceRoute`, `RpsReward`, `MtsConfig`, `Rankings`. Preserve each method's existing return type and inner call verbatim; the only changes are the leading `ctx, err := p.tenantCtx(tenantId)` guard and replacing the three `p.ctx` occurrences with `ctx`.

- [ ] **Step 5: Verify no `p.ctx` remains inside an `…AndEmit`**

```bash
cd services/atlas-tenants/atlas.com/tenants
grep -n "AndEmit" -A 14 configuration/processor.go | grep -n "p\.ctx"
```

Expected: no output. Any hit is an `…AndEmit` that was missed.

- [ ] **Step 6: Run the tests**

Run: `cd services/atlas-tenants/atlas.com/tenants && go test -race ./... && go vet ./...`
Expected: PASS. If a pre-existing `processor_test.go` case now fails because it calls an `…AndEmit` with a tenant id that has no row in the tenants table, add the tenant row to that test's setup — the new failure is the intended behaviour, not a regression.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-tenants/atlas.com/tenants/configuration/processor.go \
        services/atlas-tenants/atlas.com/tenants/configuration/emit_tenant_test.go
git commit -m "fix(atlas-tenants): emit configuration-status events with tenant headers"
```

---

### Task 8: atlas-tenants — expose a stable UUID on the configuration read models

**Files:**
- Modify: `services/atlas-tenants/atlas.com/tenants/configuration/rest.go:10-22,41-113,149-215,362-467`
- Modify: `services/atlas-tenants/atlas.com/tenants/configuration/resource.go:48,88,138,183,248,288,338,383,447,488,536,580`
- Test: `services/atlas-tenants/atlas.com/tenants/configuration/rest_test.go`

**Interfaces:**
- Consumes: `tenant.DerivedId` (Task 3).
- Produces:
  - `RouteRestModel.Uuid`, `VesselRestModel.Uuid`, `InstanceRouteRestModel.Uuid` — all `string` tagged `json:"uuid"`.
  - `TransformRoute(tenantId uuid.UUID, data map[string]interface{}) (RouteRestModel, error)`
  - `TransformVessel(tenantId uuid.UUID, data map[string]interface{}) (VesselRestModel, error)`
  - `TransformInstanceRoute(tenantId uuid.UUID, data map[string]interface{}) (InstanceRouteRestModel, error)`
  - The JSON:API resource `id` stays the slug (`Id string \`json:"-"\``) — configuration-status events and the CRUD routes reference resources by slug, so the UUID is additive.

- [ ] **Step 1: Write the failing test**

Append to `services/atlas-tenants/atlas.com/tenants/configuration/rest_test.go`:

```go
func TestTransformRoute_PopulatesDerivedUuid(t *testing.T) {
	tid := uuid.MustParse("ec876921-c363-4cc6-9c51-5bb8d57f9553")
	data := map[string]interface{}{
		"id":         "boat-ellinia-orbis",
		"type":       "routes",
		"attributes": map[string]interface{}{"name": "boat-ellinia-orbis"},
	}
	rm, err := configuration.TransformRoute(tid, data)
	if err != nil {
		t.Fatalf("TransformRoute: %v", err)
	}
	want := tenant.DerivedId(tid, "routes", "boat-ellinia-orbis").String()
	if rm.Uuid != want {
		t.Fatalf("Uuid = %q, want %q", rm.Uuid, want)
	}
	if rm.Id != "boat-ellinia-orbis" {
		t.Fatalf("Id = %q, want the slug (uuid is additive)", rm.Id)
	}
}

func TestTransformInstanceRoute_PopulatesDerivedUuid(t *testing.T) {
	tid := uuid.MustParse("ec876921-c363-4cc6-9c51-5bb8d57f9553")
	data := map[string]interface{}{
		"id":         "temple-of-time-return-flight",
		"type":       "instance-routes",
		"attributes": map[string]interface{}{"name": "temple-of-time-return-flight"},
	}
	rm, err := configuration.TransformInstanceRoute(tid, data)
	if err != nil {
		t.Fatalf("TransformInstanceRoute: %v", err)
	}
	want := tenant.DerivedId(tid, "instance-routes", "temple-of-time-return-flight").String()
	if rm.Uuid != want {
		t.Fatalf("Uuid = %q, want %q", rm.Uuid, want)
	}
}

func TestTransformVessel_PopulatesDerivedUuid(t *testing.T) {
	tid := uuid.MustParse("ec876921-c363-4cc6-9c51-5bb8d57f9553")
	data := map[string]interface{}{
		"id":         "boat-ellinia-orbis",
		"type":       "vessels",
		"attributes": map[string]interface{}{"name": "boat-ellinia-orbis"},
	}
	rm, err := configuration.TransformVessel(tid, data)
	if err != nil {
		t.Fatalf("TransformVessel: %v", err)
	}
	want := tenant.DerivedId(tid, "vessels", "boat-ellinia-orbis").String()
	if rm.Uuid != want {
		t.Fatalf("Uuid = %q, want %q", rm.Uuid, want)
	}
}

// A route and an instance-route sharing one slug must not share a uuid.
func TestTransform_ResourcesDoNotCollideOnSharedSlug(t *testing.T) {
	tid := uuid.MustParse("ec876921-c363-4cc6-9c51-5bb8d57f9553")
	shared := map[string]interface{}{"id": "shared-slug", "attributes": map[string]interface{}{"name": "shared-slug"}}
	r, _ := configuration.TransformRoute(tid, shared)
	i, _ := configuration.TransformInstanceRoute(tid, shared)
	if r.Uuid == i.Uuid {
		t.Fatalf("route and instance-route collided on uuid %q", r.Uuid)
	}
}
```

Add `"github.com/google/uuid"` and `tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"` to `rest_test.go`'s imports.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-tenants/atlas.com/tenants && go test ./configuration/... -run 'PopulatesDerivedUuid|DoNotCollide' -v`
Expected: FAIL — `too many arguments in call to configuration.TransformRoute`.

- [ ] **Step 3: Add the fields and the parameter**

In `services/atlas-tenants/atlas.com/tenants/configuration/rest.go`, add `tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"` to the imports.

Add to each of the three structs, immediately after the `Id` field:

```go
	// Uuid is a stable, tenant-scoped UUIDv5 derived from
	// (tenantId, resourceName, slug). It exists so atlas-transports can
	// key its Redis registry on an id that survives restarts and is
	// identical across replicas. The JSON:API resource id stays the
	// slug, because configuration-status events and the CRUD routes
	// reference resources by slug.
	Uuid string `json:"uuid"`
```

Change the three transform signatures and populate the field:

```go
func TransformRoute(tenantId uuid.UUID, data map[string]interface{}) (RouteRestModel, error) {
	id, _ := data["id"].(string)
	// …existing attribute extraction unchanged…
	return RouteRestModel{
		Id:                     id,
		Uuid:                   tenant.DerivedId(tenantId, "routes", id).String(),
		Name:                   name,
		// …remaining fields unchanged…
	}, nil
}
```

Do the same for `TransformVessel` (`tenant.DerivedId(tenantId, "vessels", id)`) and `TransformInstanceRoute` (`tenant.DerivedId(tenantId, "instance-routes", id)`).

- [ ] **Step 4: Update the twelve call sites**

Each `Transform*` call in `resource.go` sits inside a `rest.ParseTenantId(...)` closure, so `tenantId` is already in scope. Prefix the argument list at every one of lines 48, 88, 138, 183 (`TransformRoute`), 248, 288, 338, 383 (`TransformVessel`), 447, 488, 536, 580 (`TransformInstanceRoute`):

```go
					rm, err := TransformRoute(tenantId, route)
```

Verify none were missed:

```bash
cd services/atlas-tenants/atlas.com/tenants
grep -n "Transform\(Route\|Vessel\|InstanceRoute\)(" configuration/resource.go | grep -v "tenantId,"
```

Expected: no output.

- [ ] **Step 5: Run the tests**

Run: `cd services/atlas-tenants/atlas.com/tenants && go build ./... && go test -race ./...`
Expected: PASS. Pre-existing `TestTransformRoute` / `TestTransformVessel` in `processor_test.go` need the new first argument — pass `uuid.New()` there; they assert attribute mapping, not identity.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-tenants/atlas.com/tenants/configuration/rest.go \
        services/atlas-tenants/atlas.com/tenants/configuration/resource.go \
        services/atlas-tenants/atlas.com/tenants/configuration/rest_test.go \
        services/atlas-tenants/atlas.com/tenants/configuration/processor_test.go
git commit -m "feat(atlas-tenants): expose a stable uuid on configuration read models"
```

---

### Task 9: `libs/atlas-outbox` — make the silent tenant drop observable

**Files:**
- Modify: `libs/atlas-outbox/bridge.go:20-37`
- Test: `libs/atlas-outbox/bridge_test.go` (append — the file already exists, `package outbox_test`, with `bridgeDb(t)` and `tenantCtx(t)` helpers and `testify/require`)

**Interfaces:**
- Consumes: `tenant.ID` from `libs/atlas-tenant` (already in `libs/atlas-outbox/go.mod:7`).
- Produces: no signature change. `EnqueueBuffer` gains a WARN when the enqueue context carries no tenant.

**Explicitly deferred, with the reason recorded here so it is documented rather than forgotten (FR-3.5's escape clause):** making `producer.TenantHeaderDecorator` (`libs/atlas-kafka/producer/header.go:31-44`) return an error, or making `consumer.TenantHeaderParser` (`libs/atlas-kafka/consumer/header.go:61-65`) install *no* tenant instead of the zero tenant. Both are hot paths in all 59 services, and the parser change in particular would convert today's silent zero-tenant into a `MustFromContext` panic in every consumer that currently tolerates it. Landing either safely requires first auditing which emitters legitimately produce without a tenant — an audit far larger than this task, and one this WARN is the right instrument to drive.

- [ ] **Step 1: Write the failing test**

Append to the existing `libs/atlas-outbox/bridge_test.go`. It is `package outbox_test` and already provides `bridgeDb(t) *gorm.DB` and `tenantCtx(t) context.Context`; add `"strings"` and `logtest "github.com/sirupsen/logrus/hooks/test"` to its import block.

```go
func TestEnqueueBuffer_WarnsWhenContextHasNoTenant(t *testing.T) {
	l, hook := logtest.NewNullLogger()
	l.SetLevel(logrus.WarnLevel)

	db := bridgeDb(t)
	t.Setenv("EVENT_TOPIC_TEST", "real-topic-name")
	contents := map[string][]kafka.Message{
		"EVENT_TOPIC_TEST": {{Key: []byte("k"), Value: []byte("v")}},
	}

	require.NoError(t, outbox.EnqueueBuffer(l, context.Background(), db, contents))

	var warned bool
	for _, e := range hook.AllEntries() {
		if e.Level == logrus.WarnLevel && strings.Contains(e.Message, "without tenant headers") {
			warned = true
		}
	}
	if !warned {
		t.Fatalf("expected a WARN naming the tenant-less enqueue; got %v", hook.AllEntries())
	}
}

func TestEnqueueBuffer_SilentWhenContextHasTenant(t *testing.T) {
	l, hook := logtest.NewNullLogger()
	l.SetLevel(logrus.WarnLevel)

	db := bridgeDb(t)
	t.Setenv("EVENT_TOPIC_TEST", "real-topic-name")
	contents := map[string][]kafka.Message{
		"EVENT_TOPIC_TEST": {{Key: []byte("k"), Value: []byte("v")}},
	}

	require.NoError(t, outbox.EnqueueBuffer(l, tenantCtx(t), db, contents))

	for _, e := range hook.AllEntries() {
		if e.Level == logrus.WarnLevel {
			t.Fatalf("unexpected WARN with a tenant in context: %s", e.Message)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd libs/atlas-outbox && go test ./... -run EnqueueBuffer_Warns -v`
Expected: FAIL — no WARN is emitted.

- [ ] **Step 3: Implement**

In `libs/atlas-outbox/bridge.go`, add `tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"` to the imports and insert after the `headerMap` call (line 21-24):

```go
	headers, err := headerMap(ctx)
	if err != nil {
		return err
	}
	if _, ok := headers[tenant.ID]; !ok {
		// producer.TenantHeaderDecorator returns empty headers with a
		// nil error when the context has no tenant, and the consumer
		// side then installs the ZERO tenant rather than none — so a
		// tenant-scoped event silently reloads the wrong (nil) tenant.
		// Surface it here, on the one path every outbox emit travels.
		for token := range contents {
			l.WithField("topic_token", token).Warn("Enqueuing message without tenant headers; downstream consumers will resolve the nil tenant.")
		}
	}
```

- [ ] **Step 4: Run the tests**

Run: `cd libs/atlas-outbox && go test -race ./... && go vet ./...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-outbox/bridge.go libs/atlas-outbox/bridge_test.go
git commit -m "feat(atlas-outbox): warn when a message is enqueued without tenant headers"
```

---

### Task 10: atlas-transports — consume the stable UUID as the route id

**Files:**
- Modify: `services/atlas-transports/atlas.com/transports/instance/config/rest.go`
- Modify: `services/atlas-transports/atlas.com/transports/instance/config/processor.go:14-54`
- Modify: `services/atlas-transports/atlas.com/transports/transport/config/rest.go`
- Modify: `services/atlas-transports/atlas.com/transports/transport/config/processor.go:14-76`
- Modify: `services/atlas-transports/atlas.com/transports/instance/config/processor_drain_test.go:69`
- Modify: `services/atlas-transports/atlas.com/transports/transport/config/processor_drain_test.go:88,130`
- Test: `services/atlas-transports/atlas.com/transports/instance/config/rest_test.go` (create)
- Test: `services/atlas-transports/atlas.com/transports/transport/config/rest_test.go` (create)

**Interfaces:**
- Consumes: `tenant.DerivedId` (Task 3); the `uuid` attribute atlas-tenants now emits (Task 8).
- Produces:
  - `instance/config`: `ExtractRouteFor(l logrus.FieldLogger, t tenant.Model) func(InstanceRouteRestModel) (instance.RouteModel, error)`; `Processor.GetInstanceRoutes(t tenant.Model) ([]instance.RouteModel, error)`.
  - `transport/config`: `ExtractRouteFor(l logrus.FieldLogger, t tenant.Model) func(RouteRestModel) (transport.Model, error)`; `Processor.GetRoutes(t tenant.Model) ([]transport.Model, error)`; `Processor.GetVessels(t tenant.Model) ([]transport.SharedVesselModel, error)`.
  - `LoadConfigurationsForTenant` keeps its existing signature in both packages.
- `ExtractVessel` is **unchanged**: it already calls `SetId(v.Id)` with the slug (`transport/config/rest.go:87-93`), which is exactly why the live drift showed on `routes` and `instance-routes` (24 each against 12 configured) but not on vessels.

**Why the local fallback matters:** during a staggered rollout atlas-transports may come up before atlas-tenants, so the `uuid` attribute can be absent. Deriving locally with the identical formula keeps that window *correct*, not merely non-crashing — and because the formula lives once in `libs/atlas-tenant`, the two paths cannot diverge.

- [ ] **Step 1: Write the failing tests**

`services/atlas-transports/atlas.com/transports/instance/config/rest_test.go`:

```go
package config_test

import (
	"io"
	"testing"

	"atlas-transports/instance/config"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// If processor_drain_test.go in this package already declares a helper
// with one of these names, reuse it instead of redeclaring — both files
// are package config_test and would collide.
func quietLogger() logrus.FieldLogger {
	l := logrus.New()
	l.SetOutput(io.Discard)
	return l
}

func testTenant(t *testing.T, id uuid.UUID) tenant.Model {
	t.Helper()
	tm, err := tenant.Create(id, "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}
	return tm
}

func sampleInstanceRoute(id, uuidAttr string) config.InstanceRouteRestModel {
	return config.InstanceRouteRestModel{
		Id:                    id,
		Uuid:                  uuidAttr,
		Name:                  id,
		StartMapId:            270000100,
		TransitMapIds:         []_map.Id{200090510},
		DestinationMapId:      240000110,
		Capacity:              1,
		BoardingWindowSeconds: 1,
		TravelDurationSeconds: 900,
	}
}

func TestExtractRouteFor_UsesSuppliedUuid(t *testing.T) {
	tm := testTenant(t, uuid.New())
	want := uuid.New()
	m, err := config.ExtractRouteFor(quietLogger(), tm)(sampleInstanceRoute("temple-of-time-return-flight", want.String()))
	if err != nil {
		t.Fatalf("ExtractRouteFor: %v", err)
	}
	if m.Id() != want {
		t.Fatalf("id = %s, want %s (the supplied uuid attribute)", m.Id(), want)
	}
}

func TestExtractRouteFor_IsStableAcrossRepeatedCalls(t *testing.T) {
	tm := testTenant(t, uuid.New())
	extract := config.ExtractRouteFor(quietLogger(), tm)
	first, err := extract(sampleInstanceRoute("temple-of-time-return-flight", ""))
	if err != nil {
		t.Fatalf("first: %v", err)
	}
	second, err := extract(sampleInstanceRoute("temple-of-time-return-flight", ""))
	if err != nil {
		t.Fatalf("second: %v", err)
	}
	if first.Id() != second.Id() {
		t.Fatalf("ids differ across calls: %s != %s", first.Id(), second.Id())
	}
}

func TestExtractRouteFor_DiffersAcrossTenantsForSameSlug(t *testing.T) {
	a, err := config.ExtractRouteFor(quietLogger(), testTenant(t, uuid.New()))(sampleInstanceRoute("shared", ""))
	if err != nil {
		t.Fatalf("tenant A: %v", err)
	}
	b, err := config.ExtractRouteFor(quietLogger(), testTenant(t, uuid.New()))(sampleInstanceRoute("shared", ""))
	if err != nil {
		t.Fatalf("tenant B: %v", err)
	}
	if a.Id() == b.Id() {
		t.Fatalf("ids collided across tenants: %s", a.Id())
	}
}

func TestExtractRouteFor_FallbackMatchesTheSharedFormula(t *testing.T) {
	id := uuid.New()
	tm := testTenant(t, id)
	for _, raw := range []string{"", "not-a-uuid"} {
		m, err := config.ExtractRouteFor(quietLogger(), tm)(sampleInstanceRoute("temple-of-time-return-flight", raw))
		if err != nil {
			t.Fatalf("raw=%q: %v", raw, err)
		}
		want := tenant.DerivedId(id, "instance-routes", "temple-of-time-return-flight")
		if m.Id() != want {
			t.Fatalf("raw=%q: id = %s, want the derived %s", raw, m.Id(), want)
		}
	}
}
```

`services/atlas-transports/atlas.com/transports/transport/config/rest_test.go` — the same four tests against the scheduled-route model, plus one pinning `ExtractVessel`'s unchanged behaviour. Same caution: if `processor_drain_test.go` in this package already declares `quietLogger` or `testTenant`, reuse rather than redeclare.

```go
package config_test

import (
	"io"
	"testing"
	"time"

	"atlas-transports/transport/config"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func quietLogger() logrus.FieldLogger {
	l := logrus.New()
	l.SetOutput(io.Discard)
	return l
}

func testTenant(t *testing.T, id uuid.UUID) tenant.Model {
	t.Helper()
	tm, err := tenant.Create(id, "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}
	return tm
}

func sampleRoute(id, uuidAttr string) config.RouteRestModel {
	return config.RouteRestModel{
		Id:                     id,
		Uuid:                   uuidAttr,
		Name:                   id,
		StartMapId:             540010000,
		StagingMapId:           540010001,
		EnRouteMapIds:          []_map.Id{540010002},
		DestinationMapId:       103000000,
		ObservationMapId:       540010000,
		BoardingWindowDuration: time.Duration(4),
		PreDepartureDuration:   time.Duration(1),
		TravelDuration:         time.Duration(1),
		CycleInterval:          time.Duration(6),
	}
}

func TestExtractRouteFor_UsesSuppliedUuid(t *testing.T) {
	tm := testTenant(t, uuid.New())
	want := uuid.New()
	m, err := config.ExtractRouteFor(quietLogger(), tm)(sampleRoute("boat-ellinia-orbis", want.String()))
	if err != nil {
		t.Fatalf("ExtractRouteFor: %v", err)
	}
	if m.Id() != want {
		t.Fatalf("id = %s, want %s", m.Id(), want)
	}
}

func TestExtractRouteFor_IsStableAcrossRepeatedCalls(t *testing.T) {
	extract := config.ExtractRouteFor(quietLogger(), testTenant(t, uuid.New()))
	first, err := extract(sampleRoute("boat-ellinia-orbis", ""))
	if err != nil {
		t.Fatalf("first: %v", err)
	}
	second, err := extract(sampleRoute("boat-ellinia-orbis", ""))
	if err != nil {
		t.Fatalf("second: %v", err)
	}
	if first.Id() != second.Id() {
		t.Fatalf("ids differ across calls: %s != %s", first.Id(), second.Id())
	}
}

func TestExtractRouteFor_DiffersAcrossTenantsForSameSlug(t *testing.T) {
	a, err := config.ExtractRouteFor(quietLogger(), testTenant(t, uuid.New()))(sampleRoute("shared", ""))
	if err != nil {
		t.Fatalf("tenant A: %v", err)
	}
	b, err := config.ExtractRouteFor(quietLogger(), testTenant(t, uuid.New()))(sampleRoute("shared", ""))
	if err != nil {
		t.Fatalf("tenant B: %v", err)
	}
	if a.Id() == b.Id() {
		t.Fatalf("ids collided across tenants: %s", a.Id())
	}
}

func TestExtractRouteFor_FallbackMatchesTheSharedFormula(t *testing.T) {
	id := uuid.New()
	m, err := config.ExtractRouteFor(quietLogger(), testTenant(t, id))(sampleRoute("boat-ellinia-orbis", "not-a-uuid"))
	if err != nil {
		t.Fatalf("ExtractRouteFor: %v", err)
	}
	want := tenant.DerivedId(id, "routes", "boat-ellinia-orbis")
	if m.Id() != want {
		t.Fatalf("id = %s, want the derived %s", m.Id(), want)
	}
}

// ExtractVessel is deliberately untouched — it already sets a stable
// slug id, which is why vessels never drifted.
func TestExtractVessel_StillUsesSlugId(t *testing.T) {
	v, err := config.ExtractVessel(config.VesselRestModel{
		Id:              "boat-ellinia-orbis",
		Name:            "boat-ellinia-orbis",
		RouteAID:        "boat-ellinia-orbis",
		RouteBID:        "boat-orbis-ellinia",
		TurnaroundDelay: time.Duration(5),
	})
	if err != nil {
		t.Fatalf("ExtractVessel: %v", err)
	}
	if v.Id() != "boat-ellinia-orbis" {
		t.Fatalf("vessel id = %q, want the slug", v.Id())
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd services/atlas-transports/atlas.com/transports && go test ./instance/config/... ./transport/config/... -run ExtractRouteFor -v`
Expected: FAIL — `undefined: config.ExtractRouteFor`, `unknown field Uuid`.

- [ ] **Step 3: Implement `instance/config/rest.go`**

Replace `services/atlas-transports/atlas.com/transports/instance/config/rest.go` in full:

```go
package config

import (
	"atlas-transports/instance"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// resourceName is the atlas-tenants configuration resource these routes
// come from. It is one half of the derived-id name, so it must match the
// value atlas-tenants uses in TransformInstanceRoute exactly.
const resourceName = "instance-routes"

type InstanceRouteRestModel struct {
	Id                    string    `json:"-"`
	Uuid                  string    `json:"uuid"`
	Name                  string    `json:"name"`
	StartMapId            _map.Id   `json:"startMapId"`
	TransitMapIds         []_map.Id `json:"transitMapIds"`
	DestinationMapId      _map.Id   `json:"destinationMapId"`
	Capacity              uint32    `json:"capacity"`
	BoardingWindowSeconds uint32    `json:"boardingWindowSeconds"`
	TravelDurationSeconds uint32    `json:"travelDurationSeconds"`
	TransitMessage        string    `json:"transitMessage"`
}

func (r InstanceRouteRestModel) GetID() string {
	return r.Id
}

func (r *InstanceRouteRestModel) SetID(idStr string) error {
	r.Id = idStr
	return nil
}

func (r InstanceRouteRestModel) GetName() string {
	return "instance-routes"
}

// ExtractRouteFor builds the mapper requests.DrainProvider applies to each
// fetched route. It is tenant-aware because the route's identity is
// derived from the tenant: without a stable id the builder mints a fresh
// uuid.New() on every load, so each replica and each restart writes
// another full copy into the shared Redis registry.
func ExtractRouteFor(l logrus.FieldLogger, t tenant.Model) func(InstanceRouteRestModel) (instance.RouteModel, error) {
	return func(r InstanceRouteRestModel) (instance.RouteModel, error) {
		return instance.NewRouteBuilder(r.Name).
			SetId(resolveRouteId(l, t, r.Id, r.Uuid)).
			SetStartMapId(r.StartMapId).
			SetTransitMapIds(r.TransitMapIds).
			SetDestinationMapId(r.DestinationMapId).
			SetCapacity(r.Capacity).
			SetBoardingWindow(time.Duration(r.BoardingWindowSeconds) * time.Second).
			SetTravelDuration(time.Duration(r.TravelDurationSeconds) * time.Second).
			SetTransitMessage(r.TransitMessage).
			Build()
	}
}

// resolveRouteId prefers the uuid atlas-tenants supplies and otherwise
// derives the identical value locally. The fallback exists for a
// staggered rollout where atlas-transports is up before atlas-tenants
// serves the attribute; because both sides call tenant.DerivedId with the
// same inputs, the two paths can never disagree.
func resolveRouteId(l logrus.FieldLogger, t tenant.Model, slug string, raw string) uuid.UUID {
	if raw != "" {
		if parsed, err := uuid.Parse(raw); err == nil {
			return parsed
		}
		l.Warnf("Instance route [%s] has unparseable uuid [%s] for tenant [%s]; deriving locally.", slug, raw, t.Id())
	} else {
		l.Warnf("Instance route [%s] has no uuid attribute for tenant [%s]; deriving locally.", slug, t.Id())
	}
	return tenant.DerivedId(t.Id(), resourceName, slug)
}
```

- [ ] **Step 4: Implement `transport/config/rest.go`**

In `services/atlas-transports/atlas.com/transports/transport/config/rest.go`, add the same imports (`"github.com/google/uuid"`, `"github.com/sirupsen/logrus"`, `tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"`), add `Uuid string \`json:"uuid"\`` to `RouteRestModel` immediately after `Id`, and replace `ExtractRoute` (lines 41-58) with:

```go
// routeResourceName is the atlas-tenants configuration resource these
// routes come from; it must match TransformRoute's value exactly.
const routeResourceName = "routes"

// ExtractRouteFor builds the mapper requests.DrainProvider applies to
// each fetched route. See the instance/config twin for why identity has
// to be tenant-derived rather than freshly minted.
func ExtractRouteFor(l logrus.FieldLogger, t tenant.Model) func(RouteRestModel) (transport.Model, error) {
	return func(r RouteRestModel) (transport.Model, error) {
		builder := transport.NewBuilder(r.Name).
			SetId(resolveRouteId(l, t, r.Id, r.Uuid)).
			SetStartMapId(r.StartMapId).
			SetStagingMapId(r.StagingMapId).
			SetDestinationMapId(r.DestinationMapId).
			SetObservationMapId(r.ObservationMapId).
			SetBoardingWindowDuration(r.BoardingWindowDuration * time.Minute).
			SetPreDepartureDuration(r.PreDepartureDuration * time.Minute).
			SetTravelDuration(r.TravelDuration * time.Minute).
			SetCycleInterval(r.CycleInterval * time.Minute)

		for _, mapId := range r.EnRouteMapIds {
			builder.AddEnRouteMapId(mapId)
		}

		return builder.Build()
	}
}

func resolveRouteId(l logrus.FieldLogger, t tenant.Model, slug string, raw string) uuid.UUID {
	if raw != "" {
		if parsed, err := uuid.Parse(raw); err == nil {
			return parsed
		}
		l.Warnf("Route [%s] has unparseable uuid [%s] for tenant [%s]; deriving locally.", slug, raw, t.Id())
	} else {
		l.Warnf("Route [%s] has no uuid attribute for tenant [%s]; deriving locally.", slug, t.Id())
	}
	return tenant.DerivedId(t.Id(), routeResourceName, slug)
}
```

- [ ] **Step 5: Thread the tenant through both config processors**

`instance/config/processor.go` — change the interface and implementation:

```go
type Processor interface {
	GetInstanceRoutes(t tenant.Model) ([]instance.RouteModel, error)
	LoadConfigurationsForTenant(tenant tenant.Model) ([]instance.RouteModel, error)
}

func (p *ProcessorImpl) GetInstanceRoutes(t tenant.Model) ([]instance.RouteModel, error) {
	p.l.Debugf("Fetching instance routes for tenant [%s]", t.Id())
	return requests.DrainProvider[InstanceRouteRestModel, instance.RouteModel](p.l, p.ctx)(instanceRoutesUrl(t.Id().String()), 250, ExtractRouteFor(p.l, t), model.Filters[instance.RouteModel]())()
}

func (p *ProcessorImpl) LoadConfigurationsForTenant(t tenant.Model) ([]instance.RouteModel, error) {
	p.l.Infof("Loading instance route configurations for tenant [%s]", t.Id())

	routes, err := p.GetInstanceRoutes(t)
	if err != nil {
		return nil, err
	}

	p.l.Infof("Loaded [%d] instance routes for tenant [%s]", len(routes), t.Id())
	return routes, nil
}
```

`transport/config/processor.go` — same treatment:

```go
type Processor interface {
	GetRoutes(t tenant.Model) ([]transport.Model, error)
	GetVessels(t tenant.Model) ([]transport.SharedVesselModel, error)
	LoadConfigurationsForTenant(tenant tenant.Model) ([]transport.Model, []transport.SharedVesselModel, error)
}

func (p *ProcessorImpl) GetRoutes(t tenant.Model) ([]transport.Model, error) {
	p.l.Debugf("Fetching routes for tenant [%s]", t.Id())
	return requests.DrainProvider[RouteRestModel, transport.Model](p.l, p.ctx)(routesUrl(t.Id().String()), 250, ExtractRouteFor(p.l, t), model.Filters[transport.Model]())()
}

func (p *ProcessorImpl) GetVessels(t tenant.Model) ([]transport.SharedVesselModel, error) {
	p.l.Debugf("Fetching vessels for tenant [%s]", t.Id())
	return requests.DrainProvider[VesselRestModel, transport.SharedVesselModel](p.l, p.ctx)(vesselsUrl(t.Id().String()), 250, ExtractVessel, model.Filters[transport.SharedVesselModel]())()
}

func (p *ProcessorImpl) LoadConfigurationsForTenant(t tenant.Model) ([]transport.Model, []transport.SharedVesselModel, error) {
	p.l.Infof("Loading configurations for tenant [%s]", t.Id())

	routes, err := p.GetRoutes(t)
	if err != nil {
		return nil, nil, err
	}

	vessels, err := p.GetVessels(t)
	if err != nil {
		return nil, nil, err
	}

	p.l.Infof("Loaded [%d] routes and [%d] vessels for tenant [%s]", len(routes), len(vessels), t.Id())
	return routes, vessels, nil
}
```

- [ ] **Step 6: Update the three drain-test call sites**

- `instance/config/processor_drain_test.go:69` — `GetInstanceRoutes(ten.Id().String())` → `GetInstanceRoutes(ten)`
- `transport/config/processor_drain_test.go:88` — `GetRoutes(ten.Id().String())` → `GetRoutes(ten)`
- `transport/config/processor_drain_test.go:130` — `GetVessels(ten.Id().String())` → `GetVessels(ten)`

- [ ] **Step 7: Run the tests**

Run: `cd services/atlas-transports/atlas.com/transports && go build ./... && go test -race ./...`
Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add services/atlas-transports/atlas.com/transports/instance/config \
        services/atlas-transports/atlas.com/transports/transport/config
git commit -m "fix(atlas-transports): key route registries on a stable derived uuid"
```

---

### Task 11: atlas-transports — nil-tenant guard and load-then-clear-then-add reconcile

**Files:**
- Create: `services/atlas-transports/atlas.com/transports/bootstrap.go`
- Create: `services/atlas-transports/atlas.com/transports/bootstrap_test.go`
- Modify: `services/atlas-transports/atlas.com/transports/main.go:98-120`
- Modify: `services/atlas-transports/atlas.com/transports/kafka/consumer/configuration/consumer.go:41-72`

**Interfaces:**
- Consumes: `config.Processor` / `instanceConfig.Processor` with their Task-10 signatures; `transport.Processor` and `instance.Processor` (`AddTenant` / `ClearTenant`).
- Produces (package `main`):
  - `scheduledLoader`, `instanceLoader`, `scheduledRegistry`, `instanceRegistry` — narrow local interfaces the real processors satisfy structurally.
  - `reconcileScheduled(l logrus.FieldLogger, t tenant.Model, loader scheduledLoader, registry scheduledRegistry) error`
  - `reconcileInstance(l logrus.FieldLogger, t tenant.Model, loader instanceLoader, registry instanceRegistry) error`

**Ordering is load-first, and that is required rather than incidental.** Today `main.go:106-110` falls through to `AddTenant` with an empty slice on a load failure — harmless while `AddTenant` is purely additive. Under clear-then-add the same fall-through would **wipe a healthy registry**. The consumer at `consumer.go:48,61` has the identical hazard *today*: it calls `ClearTenant()` **before** loading, so a load failure leaves the tenant with zero routes. Both are reordered here.

**Why this is the FR-5 cleanup mechanism:** the duplicates already in `atlas-main` are keyed by ids nothing will generate again. Stable ids make `Put` idempotent but remove nothing. Making bootstrap a clear-then-add reconcile turns the deploy itself into the purge — one rolling restart converges every tenant, permanently, with no `redis-cli` surgery (which `tools/redis-key-guard.sh` discourages for good reason) and no step an operator can forget. It also self-heals any future drift.

- [ ] **Step 1: Write the failing tests**

`services/atlas-transports/atlas.com/transports/bootstrap_test.go`:

```go
package main

import (
	"errors"
	"io"
	"testing"

	"atlas-transports/instance"
	"atlas-transports/transport"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type fakeScheduledLoader struct {
	routes  []transport.Model
	vessels []transport.SharedVesselModel
	err     error
}

func (f fakeScheduledLoader) LoadConfigurationsForTenant(tenant.Model) ([]transport.Model, []transport.SharedVesselModel, error) {
	return f.routes, f.vessels, f.err
}

type fakeScheduledRegistry struct {
	cleared int
	added   int
}

func (f *fakeScheduledRegistry) ClearTenant() int { f.cleared++; return 0 }
func (f *fakeScheduledRegistry) AddTenant([]transport.Model, []transport.SharedVesselModel) error {
	f.added++
	return nil
}

type fakeInstanceLoader struct {
	routes []instance.RouteModel
	err    error
}

func (f fakeInstanceLoader) LoadConfigurationsForTenant(tenant.Model) ([]instance.RouteModel, error) {
	return f.routes, f.err
}

type fakeInstanceRegistry struct {
	cleared int
	added   int
}

func (f *fakeInstanceRegistry) ClearTenant() int              { f.cleared++; return 0 }
func (f *fakeInstanceRegistry) AddTenant([]instance.RouteModel) { f.added++ }

func quietLogger() logrus.FieldLogger {
	l := logrus.New()
	l.SetOutput(io.Discard)
	return l
}

func aTenant(t *testing.T) tenant.Model {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}
	return tm
}

func TestReconcileScheduled_ClearsThenAddsOnSuccess(t *testing.T) {
	reg := &fakeScheduledRegistry{}
	if err := reconcileScheduled(quietLogger(), aTenant(t), fakeScheduledLoader{}, reg); err != nil {
		t.Fatalf("reconcileScheduled: %v", err)
	}
	if reg.cleared != 1 || reg.added != 1 {
		t.Fatalf("cleared=%d added=%d, want 1 and 1", reg.cleared, reg.added)
	}
}

// The whole point of load-first: a load failure must leave the existing
// registry contents alone rather than wiping them.
func TestReconcileScheduled_LeavesRegistryUntouchedOnLoadError(t *testing.T) {
	reg := &fakeScheduledRegistry{}
	err := reconcileScheduled(quietLogger(), aTenant(t), fakeScheduledLoader{err: errors.New("boom")}, reg)
	if err == nil {
		t.Fatal("reconcileScheduled returned nil error, want the load error")
	}
	if reg.cleared != 0 || reg.added != 0 {
		t.Fatalf("cleared=%d added=%d, want 0 and 0 — a load failure must not clear", reg.cleared, reg.added)
	}
}

func TestReconcileInstance_ClearsThenAddsOnSuccess(t *testing.T) {
	reg := &fakeInstanceRegistry{}
	if err := reconcileInstance(quietLogger(), aTenant(t), fakeInstanceLoader{}, reg); err != nil {
		t.Fatalf("reconcileInstance: %v", err)
	}
	if reg.cleared != 1 || reg.added != 1 {
		t.Fatalf("cleared=%d added=%d, want 1 and 1", reg.cleared, reg.added)
	}
}

func TestReconcileInstance_LeavesRegistryUntouchedOnLoadError(t *testing.T) {
	reg := &fakeInstanceRegistry{}
	err := reconcileInstance(quietLogger(), aTenant(t), fakeInstanceLoader{err: errors.New("boom")}, reg)
	if err == nil {
		t.Fatal("reconcileInstance returned nil error, want the load error")
	}
	if reg.cleared != 0 || reg.added != 0 {
		t.Fatalf("cleared=%d added=%d, want 0 and 0", reg.cleared, reg.added)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd services/atlas-transports/atlas.com/transports && go test . -run Reconcile -v`
Expected: FAIL — `undefined: reconcileScheduled`.

- [ ] **Step 3: Implement `bootstrap.go`**

`services/atlas-transports/atlas.com/transports/bootstrap.go`:

```go
package main

import (
	"atlas-transports/instance"
	"atlas-transports/transport"

	"github.com/sirupsen/logrus"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// The four interfaces below are the narrow slices of the config and
// registry processors the reconcilers actually use. Declaring them here
// (rather than depending on the full Processor interfaces) keeps the
// reconcilers unit-testable without hand-writing large fakes.
type scheduledLoader interface {
	LoadConfigurationsForTenant(t tenant.Model) ([]transport.Model, []transport.SharedVesselModel, error)
}

type instanceLoader interface {
	LoadConfigurationsForTenant(t tenant.Model) ([]instance.RouteModel, error)
}

type scheduledRegistry interface {
	ClearTenant() int
	AddTenant(routes []transport.Model, sharedVessels []transport.SharedVesselModel) error
}

type instanceRegistry interface {
	ClearTenant() int
	AddTenant(routes []instance.RouteModel)
}

// reconcileScheduled replaces the tenant's scheduled-route registry
// contents with exactly what configuration says — but only after a
// successful load.
//
// Load-then-clear-then-add is load-bearing, not stylistic. Route ids are
// now derived and therefore stable, so clear-then-add converges the
// registry to exactly the configured set on every restart — which is
// what purges the duplicate entries an earlier uuid.New() per load left
// behind. Clearing before a load that then fails would instead wipe a
// healthy registry, so a load error skips the whole reconcile.
func reconcileScheduled(l logrus.FieldLogger, t tenant.Model, loader scheduledLoader, registry scheduledRegistry) error {
	routes, vessels, err := loader.LoadConfigurationsForTenant(t)
	if err != nil {
		l.WithError(err).Errorf("Failed to load configurations for tenant [%s]; leaving the scheduled route registry untouched.", t.Id())
		return err
	}
	registry.ClearTenant()
	return registry.AddTenant(routes, vessels)
}

// reconcileInstance is reconcileScheduled's twin for instance transports.
func reconcileInstance(l logrus.FieldLogger, t tenant.Model, loader instanceLoader, registry instanceRegistry) error {
	routes, err := loader.LoadConfigurationsForTenant(t)
	if err != nil {
		l.WithError(err).Errorf("Failed to load instance route configurations for tenant [%s]; leaving the instance route registry untouched.", t.Id())
		return err
	}
	registry.ClearTenant()
	registry.AddTenant(routes)
	return nil
}
```

- [ ] **Step 4: Use the reconcilers in `main.go`**

Replace `services/atlas-transports/atlas.com/transports/main.go:98-120` with:

```go
	// Reconcile each tenant's registries against configuration. This is
	// what converges the duplicate entries that pre-stable-id loads left
	// in the shared Redis registry: one rolling restart per tenant.
	configProcessor := config.NewProcessor(l, rt.Context())
	instanceConfigProcessor := instanceConfig.NewProcessor(l, rt.Context())
	for _, t := range tenants {
		ctx := tenant.WithContext(rt.Context(), t)
		_ = reconcileScheduled(l, t, configProcessor, transport.NewProcessor(l, ctx))
		_ = reconcileInstance(l, t, instanceConfigProcessor, instance.NewProcessor(l, ctx))
	}
```

- [ ] **Step 5: Guard the consumer and reorder its reload**

Replace `handleConfigurationStatus` in `services/atlas-transports/atlas.com/transports/kafka/consumer/configuration/consumer.go:41-72` with:

```go
func handleConfigurationStatus(l logrus.FieldLogger, ctx context.Context, e configuration2.StatusEvent) {
	t := tenant.MustFromContext(ctx)
	if t.Id() == uuid.Nil {
		// consumer.TenantHeaderParser installs the ZERO tenant when the
		// message carries no tenant headers, so without this guard a
		// header-less event would ClearTenant() and reload nothing —
		// silently emptying a healthy registry. Refusing to act makes
		// the nil-tenant signature a loud regression instead of a quiet
		// data-loss event.
		l.Errorf("Configuration-status event [%s] for resource [%s] arrived without tenant headers; skipping reload.", e.Type, e.ResourceId)
		return
	}

	switch e.ResourceType {
	case "route", "vessel":
		l.Infof("Configuration [%s] event [%s] for resource [%s], reloading scheduled routes for tenant [%s].", e.ResourceType, e.Type, e.ResourceId, t.Id())

		// Load BEFORE clearing: the reload is a full replace, so a load
		// failure after a clear would leave the tenant with no routes.
		routes, sharedVessels, err := config.NewProcessor(l, ctx).LoadConfigurationsForTenant(t)
		if err != nil {
			l.WithError(err).Errorf("Failed to reload configurations for tenant [%s]; leaving the scheduled route registry untouched.", t.Id())
			return
		}
		tp := transport.NewProcessor(l, ctx)
		tp.ClearTenant()
		_ = tp.AddTenant(routes, sharedVessels)
	case "instance-route":
		l.Infof("Configuration [%s] event [%s] for resource [%s], reloading instance routes for tenant [%s].", e.ResourceType, e.Type, e.ResourceId, t.Id())

		instanceRoutes, err := instanceConfig.NewProcessor(l, ctx).LoadConfigurationsForTenant(t)
		if err != nil {
			l.WithError(err).Errorf("Failed to reload instance route configurations for tenant [%s]; leaving the instance route registry untouched.", t.Id())
			return
		}
		ip := instance.NewProcessor(l, ctx)
		ip.ClearTenant()
		ip.AddTenant(instanceRoutes)
	default:
		l.Warnf("Unhandled configuration resource type [%s].", e.ResourceType)
	}
}
```

Add `"github.com/google/uuid"` to that file's imports.

- [ ] **Step 6: Write the consumer guard test**

`services/atlas-transports/atlas.com/transports/kafka/consumer/configuration/consumer_test.go`:

```go
package configuration

import (
	"context"
	"strings"
	"testing"

	configuration2 "atlas-transports/kafka/message/configuration"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// A header-less event resolves the zero tenant. The handler must log an
// ERROR and perform NO registry mutation — the registries are package
// singletons that were never initialised in this test, so any call into
// them would panic. A passing test therefore proves no ClearTenant ran.
func TestHandleConfigurationStatus_NilTenantSkipsReload(t *testing.T) {
	l, hook := test.NewNullLogger()
	l.SetLevel(logrus.ErrorLevel)

	zero, err := tenant.Create(uuid.Nil, "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}
	ctx := tenant.WithContext(context.Background(), zero)

	handleConfigurationStatus(l, ctx, configuration2.StatusEvent{
		Type:         "INSTANCE_ROUTE_UPDATED",
		ResourceType: "instance-route",
		ResourceId:   "temple-of-time-return-flight",
	})

	var logged bool
	for _, e := range hook.AllEntries() {
		if e.Level == logrus.ErrorLevel && strings.Contains(e.Message, "without tenant headers") {
			logged = true
		}
	}
	if !logged {
		t.Fatalf("expected an ERROR naming the missing tenant headers; got %v", hook.AllEntries())
	}
}
```

If `configuration2.StatusEvent`'s field names differ from `Type` / `ResourceType` / `ResourceId`, read `services/atlas-transports/atlas.com/transports/kafka/message/configuration/` and adapt.

- [ ] **Step 7: Run the tests**

Run: `cd services/atlas-transports/atlas.com/transports && go build ./... && go test -race ./... && go vet ./...`
Expected: PASS.

- [ ] **Step 8: Run the repo guards**

```bash
cd ../../../..
tools/redis-key-guard.sh
tools/goroutine-guard.sh
```

Expected: both exit 0.

- [ ] **Step 9: Commit**

```bash
git add services/atlas-transports/atlas.com/transports/bootstrap.go \
        services/atlas-transports/atlas.com/transports/bootstrap_test.go \
        services/atlas-transports/atlas.com/transports/main.go \
        services/atlas-transports/atlas.com/transports/kafka/consumer/configuration
git commit -m "fix(atlas-transports): reconcile registries load-first and reject nil-tenant reloads"
```

---

### Task 12: atlas-ui — three transport seed rows on the Setup page

**Files:**
- Modify: `services/atlas-ui/src/services/api/seed.service.ts:29-72,172-204,257-350`
- Modify: `services/atlas-ui/src/lib/hooks/api/useSeed.ts:9-58,60-190,256-379`
- Modify: `services/atlas-ui/src/pages/SetupPage.tsx:10-24,26-56,61-83,183-272`
- Test: `services/atlas-ui/src/services/api/__tests__/seed.service.test.ts`
- Test: `services/atlas-ui/src/lib/hooks/api/__tests__/useSeed.test.tsx`

**Interfaces:**
- Consumes: `GET /api/tenants/configurations/<res>/seed/status` returning `libs/atlas-seeder`'s `Status` shape, with `subdomains` keyed `routes` / `vessels` / `instance-routes` (Task 6).
- Produces:
  - `seedService.seedRoutes(): Promise<void>`, `seedVessels()`, `seedInstanceRoutes()` — each `api.post("/api/tenants/configurations/<res>/seed", {})`, mirroring `seedDrops`. These are the **202 + poll-status** contract: the endpoint returns 202 immediately and seeds in the background, so there is no result body to read.
  - `TransportRoutesSeedStatus { routeCount: number; updatedAt: string | null }`
  - `TransportVesselsSeedStatus { vesselCount: number; updatedAt: string | null }`
  - `InstanceRoutesSeedStatus { routeCount: number; updatedAt: string | null }`
  - `seedService.getTransportRoutesSeedStatus(tenant)`, `getTransportVesselsSeedStatus(tenant)`, `getInstanceRoutesSeedStatus(tenant)`
  - `useSeedTransportRoutes()`, `useSeedTransportVessels()`, `useSeedInstanceRoutes()`, `useTransportRoutesSeedStatus()`, `useTransportVesselsSeedStatus()`, `useInstanceRoutesSeedStatus()`

**Design concern, raised and carried forward as the PRD directs.** FR-2.1 mandates three independent rows, and this implements exactly that. Scheduled transports need **both** `routes` and `vessels` to compute a schedule, so an operator who seeds one and not the other leaves the tenant quietly half-configured — the same failure class this task fixes. A single "Transport Configuration" group with three subdomains and one badge would make partial seeding impossible and matches the multi-subdomain convention Drops and Reward Pools already use; it is a cheap follow-up if the hazard proves real. Mitigation within scope: seeding either `routes` or `vessels` triggers a full scheduled-transport reload, so ordering does not matter and the second seed converges.

- [ ] **Step 1: Write the failing tests**

Append to `services/atlas-ui/src/services/api/__tests__/seed.service.test.ts` (match the file's existing mocking idiom — if it stubs `global.fetch`, keep doing that):

```ts
describe("transport configuration seed status", () => {
  const tenant = {
    id: "ec876921-c363-4cc6-9c51-5bb8d57f9553",
    attributes: { region: "GMS", majorVersion: 83, minorVersion: 1 },
  } as never;

  it("projects the routes subdomain count", async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({
        groupName: "routes",
        subdomains: { routes: { count: 12, updatedAt: null } },
        updatedAt: null,
        catalogRevision: "abc+def",
        tenantSeededRevision: "abc+def",
        tenantSeededAt: "2026-08-03T00:00:00Z",
      }),
    });
    vi.stubGlobal("fetch", fetchMock);

    const status = await seedService.getTransportRoutesSeedStatus(tenant);

    expect(status.routeCount).toBe(12);
    expect(status.updatedAt).toBe("2026-08-03T00:00:00Z");
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/tenants/configurations/routes/seed/status",
      expect.objectContaining({ method: "GET" }),
    );
  });

  it("projects the vessels subdomain count", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: true,
        json: async () => ({
          groupName: "vessels",
          subdomains: { vessels: { count: 6, updatedAt: null } },
          updatedAt: null,
          catalogRevision: "",
          tenantSeededRevision: null,
          tenantSeededAt: null,
        }),
      }),
    );
    const status = await seedService.getTransportVesselsSeedStatus(tenant);
    expect(status.vesselCount).toBe(6);
  });

  it("projects the instance-routes subdomain count", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: true,
        json: async () => ({
          groupName: "instance-routes",
          subdomains: { "instance-routes": { count: 12, updatedAt: null } },
          updatedAt: null,
          catalogRevision: "",
          tenantSeededRevision: null,
          tenantSeededAt: null,
        }),
      }),
    );
    const status = await seedService.getInstanceRoutesSeedStatus(tenant);
    expect(status.routeCount).toBe(12);
  });

  it("reports zero when the tenant has never been seeded", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: true,
        json: async () => ({
          groupName: "routes",
          subdomains: {},
          updatedAt: null,
          catalogRevision: "",
          tenantSeededRevision: null,
          tenantSeededAt: null,
        }),
      }),
    );
    const status = await seedService.getTransportRoutesSeedStatus(tenant);
    expect(status.routeCount).toBe(0);
    expect(status.updatedAt).toBeNull();
  });
});
```

Append to `services/atlas-ui/src/lib/hooks/api/__tests__/useSeed.test.tsx`, following the file's existing wrapper/mocking setup:

```tsx
describe("transport seed mutations", () => {
  it("useSeedTransportRoutes posts to the routes seed endpoint", async () => {
    const spy = vi
      .spyOn(seedService, "seedRoutes")
      .mockResolvedValue(undefined);
    const { result } = renderHook(() => useSeedTransportRoutes(), {
      wrapper: createWrapper(),
    });
    result.current.mutate();
    await waitFor(() => expect(spy).toHaveBeenCalledTimes(1));
  });

  it("useSeedTransportVessels posts to the vessels seed endpoint", async () => {
    const spy = vi
      .spyOn(seedService, "seedVessels")
      .mockResolvedValue(undefined);
    const { result } = renderHook(() => useSeedTransportVessels(), {
      wrapper: createWrapper(),
    });
    result.current.mutate();
    await waitFor(() => expect(spy).toHaveBeenCalledTimes(1));
  });

  it("useSeedInstanceRoutes posts to the instance-routes seed endpoint", async () => {
    const spy = vi
      .spyOn(seedService, "seedInstanceRoutes")
      .mockResolvedValue(undefined);
    const { result } = renderHook(() => useSeedInstanceRoutes(), {
      wrapper: createWrapper(),
    });
    result.current.mutate();
    await waitFor(() => expect(spy).toHaveBeenCalledTimes(1));
  });
});
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd services/atlas-ui && npm run test -- seed`
Expected: FAIL — `seedService.getTransportRoutesSeedStatus is not a function`.

- [ ] **Step 3: Extend `seed.service.ts`**

Add after `MapActionScriptsSeedStatus` (line 72):

```ts
export interface TransportRoutesSeedStatus {
  routeCount: number;
  updatedAt: string | null;
}

export interface TransportVesselsSeedStatus {
  vesselCount: number;
  updatedAt: string | null;
}

export interface InstanceRoutesSeedStatus {
  routeCount: number;
  updatedAt: string | null;
}
```

Add to `SeedService` after `seedMapActionScripts` (line 203):

```ts
  // The three transport-configuration seeds return 202 with no body and
  // seed in the background (libs/atlas-seeder's postSeed contract), so
  // these resolve to void and the Setup page's 5s status poll is what
  // surfaces the result.
  async seedRoutes(): Promise<void> {
    await api.post("/api/tenants/configurations/routes/seed", {});
  }

  async seedVessels(): Promise<void> {
    await api.post("/api/tenants/configurations/vessels/seed", {});
  }

  async seedInstanceRoutes(): Promise<void> {
    await api.post("/api/tenants/configurations/instance-routes/seed", {});
  }
```

Add after `getMapActionScriptsSeedStatus` (line 350):

```ts
  async getTransportRoutesSeedStatus(
    tenant: Tenant,
  ): Promise<TransportRoutesSeedStatus> {
    const s = await fetchSeedStatus(
      "/api/tenants/configurations/routes/seed/status",
      tenant,
    );
    return {
      routeCount: subdomainCount(s, "routes"),
      updatedAt: s.tenantSeededAt ?? s.updatedAt,
    };
  }

  async getTransportVesselsSeedStatus(
    tenant: Tenant,
  ): Promise<TransportVesselsSeedStatus> {
    const s = await fetchSeedStatus(
      "/api/tenants/configurations/vessels/seed/status",
      tenant,
    );
    return {
      vesselCount: subdomainCount(s, "vessels"),
      updatedAt: s.tenantSeededAt ?? s.updatedAt,
    };
  }

  async getInstanceRoutesSeedStatus(
    tenant: Tenant,
  ): Promise<InstanceRoutesSeedStatus> {
    const s = await fetchSeedStatus(
      "/api/tenants/configurations/instance-routes/seed/status",
      tenant,
    );
    return {
      routeCount: subdomainCount(s, "instance-routes"),
      updatedAt: s.tenantSeededAt ?? s.updatedAt,
    };
  }
```

- [ ] **Step 4: Extend `useSeed.ts`**

Add to the type import block from `@/services/api/seed.service`: `type InstanceRoutesSeedStatus`, `type TransportRoutesSeedStatus`, `type TransportVesselsSeedStatus`.

Add after `mapActionScriptsSeedStatusKey` (line 58):

```ts
const transportRoutesSeedStatusKey = (tenantId: string) =>
  ["transportRoutesSeedStatus", tenantId] as const;
const transportVesselsSeedStatusKey = (tenantId: string) =>
  ["transportVesselsSeedStatus", tenantId] as const;
const instanceRoutesSeedStatusKey = (tenantId: string) =>
  ["instanceRoutesSeedStatus", tenantId] as const;
```

Add after `useSeedMapActionScripts` (line 190):

```ts
export function useSeedTransportRoutes(): UseMutationResult<void, Error, void> {
  const { activeTenant } = useTenant();
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () => seedService.seedRoutes(),
    onSuccess: () => {
      if (!activeTenant) return;
      void queryClient.invalidateQueries({
        queryKey: transportRoutesSeedStatusKey(activeTenant.id),
      });
    },
  });
}

export function useSeedTransportVessels(): UseMutationResult<
  void,
  Error,
  void
> {
  const { activeTenant } = useTenant();
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () => seedService.seedVessels(),
    onSuccess: () => {
      if (!activeTenant) return;
      void queryClient.invalidateQueries({
        queryKey: transportVesselsSeedStatusKey(activeTenant.id),
      });
    },
  });
}

export function useSeedInstanceRoutes(): UseMutationResult<void, Error, void> {
  const { activeTenant } = useTenant();
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () => seedService.seedInstanceRoutes(),
    onSuccess: () => {
      if (!activeTenant) return;
      void queryClient.invalidateQueries({
        queryKey: instanceRoutesSeedStatusKey(activeTenant.id),
      });
    },
  });
}
```

Add at the end of the file:

```ts
export function useTransportRoutesSeedStatus(): UseQueryResult<
  TransportRoutesSeedStatus,
  Error
> {
  const { activeTenant } = useTenant();
  return useQuery({
    queryKey: activeTenant
      ? transportRoutesSeedStatusKey(activeTenant.id)
      : ["transportRoutesSeedStatus", "none"],
    queryFn: () => seedService.getTransportRoutesSeedStatus(activeTenant!),
    enabled: !!activeTenant,
    staleTime: 0,
    refetchInterval: 5000,
  });
}

export function useTransportVesselsSeedStatus(): UseQueryResult<
  TransportVesselsSeedStatus,
  Error
> {
  const { activeTenant } = useTenant();
  return useQuery({
    queryKey: activeTenant
      ? transportVesselsSeedStatusKey(activeTenant.id)
      : ["transportVesselsSeedStatus", "none"],
    queryFn: () => seedService.getTransportVesselsSeedStatus(activeTenant!),
    enabled: !!activeTenant,
    staleTime: 0,
    refetchInterval: 5000,
  });
}

export function useInstanceRoutesSeedStatus(): UseQueryResult<
  InstanceRoutesSeedStatus,
  Error
> {
  const { activeTenant } = useTenant();
  return useQuery({
    queryKey: activeTenant
      ? instanceRoutesSeedStatusKey(activeTenant.id)
      : ["instanceRoutesSeedStatus", "none"],
    queryFn: () => seedService.getInstanceRoutesSeedStatus(activeTenant!),
    enabled: !!activeTenant,
    staleTime: 0,
    refetchInterval: 5000,
  });
}
```

- [ ] **Step 5: Add the three rows to `SetupPage.tsx`**

Add `Ship`, `Anchor`, and `Plane` to the `lucide-react` import block (line 10-24).

Add to the `@/lib/hooks/api/useSeed` import block (line 26-47): `useSeedTransportRoutes`, `useSeedTransportVessels`, `useSeedInstanceRoutes`, `useTransportRoutesSeedStatus`, `useTransportVesselsSeedStatus`, `useInstanceRoutesSeedStatus`.

Add after `const seedMapActionScripts = useSeedMapActionScripts();` (line 68):

```tsx
  const seedTransportRoutes = useSeedTransportRoutes();
  const seedTransportVessels = useSeedTransportVessels();
  const seedInstanceRoutes = useSeedInstanceRoutes();
```

Add after `const mapActionScriptsSeed = useMapActionScriptsSeedStatus();` (line 83):

```tsx
  const transportRoutesSeed = useTransportRoutesSeedStatus();
  const transportVesselsSeed = useTransportVesselsSeedStatus();
  const instanceRoutesSeed = useInstanceRoutesSeedStatus();
```

Append three entries to the `seedRows` array, after the "Map Action Scripts" entry (line 271):

```tsx
    {
      label: "Transport Routes",
      icon: <Ship className="h-5 w-5" />,
      mutation: seedTransportRoutes,
      formatBadge: () => {
        const d = transportRoutesSeed.data;
        return !d
          ? "—"
          : `${formatCount(d.routeCount)} ${pluralize(d.routeCount, "route", "routes")}`;
      },
    },
    {
      label: "Transport Vessels",
      icon: <Anchor className="h-5 w-5" />,
      mutation: seedTransportVessels,
      formatBadge: () => {
        const d = transportVesselsSeed.data;
        return !d
          ? "—"
          : `${formatCount(d.vesselCount)} ${pluralize(d.vesselCount, "vessel", "vessels")}`;
      },
    },
    {
      label: "Instance Transport Routes",
      icon: <Plane className="h-5 w-5" />,
      mutation: seedInstanceRoutes,
      formatBadge: () => {
        const d = instanceRoutesSeed.data;
        return !d
          ? "—"
          : `${formatCount(d.routeCount)} ${pluralize(d.routeCount, "route", "routes")}`;
      },
    },
```

The rows automatically inherit the existing pending/disabled handling and the `handleSeed` toast from the `seedRows.map(...)` loop at line 391 — no other change to the render tree.

- [ ] **Step 6: Run the tests and the build**

```bash
cd services/atlas-ui
npm run test
npm run build
```

Expected: both pass. `npm run build` is required, not just vitest — it runs `tsc -b`, which is what catches a type error the test run would not.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-ui/src
git commit -m "feat(atlas-ui): add transport configuration seed rows to the Setup page"
```

---

### Task 13: deploy wiring, runbook, and the full verification sweep

**Files:**
- Modify: `deploy/k8s/base/atlas-tenants.yaml` (metadata labels)
- Create: `docs/tasks/task-189-tenant-config-seed-provisioning/runbook.md`

**Interfaces:**
- Consumes: everything from Tasks 1-12.
- Produces: the `atlas.seed-catalog: "true"` label that makes `deploy/k8s/base/components/seed-catalog` patch atlas-tenants with the git-sync sidecar, the `/var/run/seed-catalog` mount, and `SEED_CATALOG_ROOT=/var/run/seed-catalog/catalog/deploy/seed` — the value `seed.InitResource`'s `NewFilesystemCatalogSourceWithShared` reads.

- [ ] **Step 1: Label atlas-tenants as a seed-catalog consumer**

In `deploy/k8s/base/atlas-tenants.yaml`, add the label to the Deployment's `metadata.labels` so the component's `labelSelector: "atlas.seed-catalog=true"` matches. Mirror `deploy/k8s/base/atlas-drop-information.yaml:7` exactly:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: atlas-tenants
  labels:
    atlas.seed-catalog: "true"
spec:
```

- [ ] **Step 2: Confirm the patch actually applies**

```bash
kubectl kustomize deploy/k8s/base | \
  awk '/^  name: atlas-tenants$/,/^---$/' | \
  grep -E "git-sync|SEED_CATALOG_ROOT|seed-catalog"
```

Expected: the `git-sync` container, the `seed-catalog` volume/volumeMount, and `SEED_CATALOG_ROOT` all appear under atlas-tenants. If `kubectl` is unavailable, `kustomize build deploy/k8s/base` is equivalent.

- [ ] **Step 3: Write the runbook**

`docs/tasks/task-189-tenant-config-seed-provisioning/runbook.md`:

```markdown
# task-189 Runbook — Transport Configuration Provisioning & Registry Convergence

Covers the rollout of task-189, the post-deploy verification that the
duplicate Redis registry entries are gone, and the manual fallback if the
automatic reconcile does not converge.

## What changes at deploy time

1. **atlas-tenants** gains a git-sync sidecar and reads its transport seed
   data from `deploy/seed/shared/all/` instead of the image. It also runs
   the `seed_state` AutoMigrate. Existing `configurations` rows are
   untouched and unchanged in shape.
2. **atlas-transports** reconciles every tenant's route registries at
   startup: load → (on success) clear → add. Because route ids are now
   derived and stable, this converges each tenant to exactly the
   configured set and permanently removes the random-id duplicates.
3. **atlas-ui** gains three Setup rows: Transport Routes, Transport
   Vessels, Instance Transport Routes.

## Rollout order

1. Merge. `main-publish` stamps `CATALOG_REVISION` into
   `deploy/seed/shared/all/` and bumps atlas-tenants, atlas-transports,
   and atlas-ui.
2. Let atlas-tenants roll first. Confirm the git-sync sidecar is running
   and the catalog is mounted:

   ```
   kubectl -n atlas-main get pod -l app=atlas-tenants \
     -o jsonpath='{.items[0].spec.containers[*].name}'
   kubectl -n atlas-main exec deploy/atlas-tenants -c atlas-tenants -- \
     ls /var/run/seed-catalog/catalog/deploy/seed/shared/all
   ```

   Expected: a `git-sync` container is present, and the listing shows
   `CATALOG_REVISION`, `instance-routes`, `routes`, `vessels`.
3. Let atlas-transports roll. Its bootstrap reconcile runs per tenant. If
   atlas-tenants has not finished rolling, atlas-transports derives the
   same ids locally and logs a WARN per route — that is expected during
   the window and self-corrects on the next reload.

## Post-deploy verification (FR-5.3)

For every tenant, the live registry count must equal the configured
count. Configured counts on this branch are 12 routes, 6 vessels, and 12
instance routes.

For each tenant, with that tenant's four headers set:

```
GET /api/tenants/configurations/routes/seed/status
GET /api/tenants/configurations/vessels/seed/status
GET /api/tenants/configurations/instance-routes/seed/status
```

and compare `subdomains.<resource>.count` against:

```
GET /api/transports/routes             -> total
GET /api/transports/instance-routes    -> total
```

A live `total` of **twice** the configured count means the reconcile did
not run — check that the atlas-transports pods actually restarted on the
new image. A live `total` **below** the configured count means the load
failed and the reconcile correctly skipped; check the atlas-transports
log for "leaving the ... registry untouched".

`00000000-0000-0000-0000-000000000000` must appear in **no**
atlas-transports configuration-reload log line. If it does, that is a
regression, not a cosmetic issue: it means a configuration-status event
reached the consumer without tenant headers.

## Provisioning a new or restored tenant

Open Setup with the tenant selected and click Seed on all three transport
rows. Seed **both** Transport Routes and Transport Vessels — a scheduled
transport needs both to compute a schedule, and seeding only one leaves
the tenant quietly half-configured. Order does not matter: seeding either
triggers a full scheduled-transport reload, so the second seed converges.

The badges should read 12 routes / 6 vessels / 12 routes once each
background seed completes (the page polls status every 5 seconds).

## Manual fallback

Only if the automatic reconcile does not converge after a confirmed
restart on the new image:

1. Scale atlas-transports to 0 replicas.
2. Delete the tenant's registry keys (`instance-route:*` and
   `transport-route:*` under that tenant's registry prefix). Note that
   `tools/redis-key-guard.sh` bans keyed Redis commands in service code
   for good reason — this is an operator action, not something to
   automate into a service.
3. Scale atlas-transports back up. The bootstrap reconcile repopulates
   from configuration.

## Known startup window

Clear-then-add leaves a brief window during startup where a tenant's
registry is partial. It is bounded by the reconcile loop and acceptable
for a startup path. Concurrent replicas converge regardless of
interleaving, because both write the same stable ids. Note that a restart
already recomputes scheduled-route schedules — the 1-second
`UpdateRoutes()` tick in every replica is last-writer-wins today.
```

- [ ] **Step 4: Run every Go gate**

```bash
for m in libs/atlas-seeder libs/atlas-outbox libs/atlas-tenant \
         services/atlas-tenants/atlas.com/tenants \
         services/atlas-transports/atlas.com/transports; do
  echo "=== $m ==="
  ( cd "$m" && go build ./... && go vet ./... && go test -race ./... ) || exit 1
done
```

Expected: all five modules clean.

- [ ] **Step 5: Run every repo guard**

```bash
tools/redis-key-guard.sh
tools/goroutine-guard.sh
tools/service-registration-guard.sh
tools/lint.sh --check
go run ./tools/catalog-lint deploy/seed
```

Expected: all exit 0. If `tools/lint.sh --check` reports formatting diffs, run `tools/lint.sh` (no flags) to rewrite in place, re-run `--check`, and amend.

- [ ] **Step 6: Bake both images**

Both `go.mod` files changed (atlas-tenants gained `libs/atlas-seeder`), so this is mandatory — `go build` against the workspace will not catch a missing `COPY libs/...` line in the shared `Dockerfile`. `Dockerfile:47` and `Dockerfile:77` already COPY `libs/atlas-seeder`, so no Dockerfile change is expected; the bake is what proves it.

```bash
docker buildx bake atlas-tenants atlas-transports
```

Expected: both targets build.

- [ ] **Step 7: Verify the atlas-ui gates**

```bash
cd services/atlas-ui && npm run test && npm run build
```

Expected: both pass.

- [ ] **Step 8: Commit**

```bash
git add deploy/k8s/base/atlas-tenants.yaml \
        docs/tasks/task-189-tenant-config-seed-provisioning/runbook.md
git commit -m "chore(task-189): wire atlas-tenants to the seed catalog and add the rollout runbook"
```

---

## Requirement Coverage

| Requirement | Task |
|---|---|
| FR-1.1 register seeder Groups, replace bespoke handlers | 6 |
| FR-1.2 standard seed + seed/status pair, header-scoped; old endpoints removed | 6 |
| FR-1.3 version-agnostic root, merged across roots | 1, 4 |
| FR-1.4 version-specific wins on collision | 1 |
| FR-1.5 single source of truth, no forked copies | 4 |
| FR-1.6 seed-catalog component consumer wiring | 13 |
| FR-1.7 idempotent re-seed | 5 (delete-then-append), 6 |
| FR-2.1 three Setup rows | 12 |
| FR-2.2 live count badges, `—` when absent | 12 |
| FR-2.3 service methods + hooks + query keys | 12 |
| FR-2.4 tenant scoping via TenantProvider | 12 |
| FR-2.5 pending/disabled + toast conventions | 12 |
| FR-3.1 events carry the four tenant headers | 7 |
| FR-3.2 fixed at the context, not the consumer | 7 |
| FR-3.3 all six resources / 18 emitters | 7 |
| FR-3.4 reload without restart, real tenant id | 7, 11 |
| FR-3.5 silent drop made observable; library change deferred with rationale | 9 (producer WARN), 11 (consumer ERROR) |
| FR-4.1 atlas-tenants exposes a stable UUID | 8 |
| FR-4.2 atlas-transports uses it as the route id | 10 |
| FR-4.3 stable across loads/replicas/restarts/re-seeds, per-tenant | 3, 10 |
| FR-4.4 live count equals configured count | 11 (reconcile), 13 (verification) |
| FR-4.5 both registries fixed | 10, 11 |
| FR-5.1 existing duplicates removed | 11 |
| FR-5.2 documented runbook | 13 |
| FR-5.3 verified against configured counts | 13 |
| NFR multi-tenancy (cross-tenant test) | 5 |
| NFR observability (nil tenant surfaces at WARN/ERROR) | 9, 11 |
| NFR verification gates | 13 |
