# Task-071 Followups Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Close the 20 followup items inventoried in task-071's `followups.md` — production hot-path bug fixes (F1, F2, F3, F5), deploy determinism (F7), routes-config dedupe (F8), wz/lib hygiene (F4, F6, F12–F16), coverage gaps (F17, F18), deferred carve-outs (F19, F20), and three operational one-shots (F9, F10, F11) — in two ordered waves.

**Architecture:** Twenty small, independently-revertable commits in one PR, ordered as Wave 1 (production hot path) → Wave 2 (debt) → operational one-shots → conditional. No new services, no schema migrations, one new lib dependency (`libs/atlas-wz` → `libs/atlas-constants`). The two unknown-root-cause items (F1 publish 500, F20 Henesys portal duplication) are diagnose-then-fix: a repro log is committed before the fix.

**Tech Stack:** Go 1.25, GORM, pgx v5, minio-go, kustomize, nginx, bash. Tests use the project Builder pattern (no `*_testhelpers.go` files — see CLAUDE.md "Test Helper Pattern").

**Required cwd:** `.worktrees/task-076-task071-followups`. The worktree is on branch `task-076-task071-followups`. Every command below assumes that cwd unless noted otherwise.

**Commit shape:** Each task commits with `fix(<service>): F<id> <short summary>` (verb fits — `feat(deploy):` for F7/F19, `docs(ops):` for F9/F10, `chore(...)` for orphan deletions). Subagent runs MUST `cd` into the worktree first and verify the branch is `task-076-task071-followups` after every commit (project Memory rule).

**Per-Go-module verification gate (run after each task that touches Go):**

```bash
cd .worktrees/task-076-task071-followups
( cd <changed-module> && go test -race ./... && go vet ./... && go build ./... )
```

Module roots: `services/atlas-data/atlas.com/data`, `services/atlas-renders/atlas.com/renders`, `libs/atlas-wz`, etc. — check the directory containing the touched `go.mod`. If `go.mod` itself was modified (only F16), additionally bake every importer:

```bash
docker buildx bake atlas-data atlas-renders atlas-character-factory
```

---

## Wave 1 — Production Hot Path

### Task 1: F1 — Diagnose and fix `POST /data/baseline/publish` 500

**Files:**
- Modify: `services/atlas-data/atlas.com/data/baseline/publish.go:31-75`
- Modify: `services/atlas-data/atlas.com/data/baseline/handler.go:39-43`
- Create: `services/atlas-data/atlas.com/data/baseline/publish_test.go`
- Create: `docs/tasks/task-076-task071-followups/repro/f1-publish-500.log`

- [ ] **Step 1: Capture a pre-fix repro log against a local atlas-data instance OR atlas-main.**

Pick whichever is fastest. If a fresh PR env is available, that's authoritative. Otherwise stand up `atlas-data` against compose's postgres+minio and exercise the same curl recipe (the goal is logging coverage, not the production cause specifically).

Run (substitute pod or local URL):

```bash
DATA=$(kubectl -n atlas-main get pod -l app=atlas-data -o jsonpath='{.items[0].metadata.name}')
kubectl -n atlas-main exec $DATA -- wget -qS -O- \
  --post-data='{"data":{"type":"baselinePublishes","attributes":{"region":"GMS","majorVersion":83,"minorVersion":1}}}' \
  --header='Content-Type: application/vnd.api+json' \
  --header='TENANT_ID: ec876921-c363-4cc6-9c51-5bb8d57f9553' \
  --header='REGION: GMS' --header='MAJOR_VERSION: 83' --header='MINOR_VERSION: 1' \
  --header='X-Atlas-Operator: 1' \
  'http://atlas-data:8080/api/data/baseline/publish' 2>&1 | tee docs/tasks/task-076-task071-followups/repro/f1-publish-500.log
kubectl -n atlas-main logs $DATA --since=2m | tee -a docs/tasks/task-076-task071-followups/repro/f1-publish-500.log
```

Expected: HTTP 500 with empty body and at most a `Handling [POST]` log line.

If the recipe is non-reproducible at this point (e.g., the bug self-resolved on a redeploy), document that in the same log and continue — the fix below is still warranted because the current code path swallows errors.

- [ ] **Step 2: Write the failing test.**

Create `services/atlas-data/atlas.com/data/baseline/publish_test.go`:

```go
package baseline

import (
	"context"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
)

// TestPublishErrorIsContextualized asserts that when the dump-table step
// fails, the error surfaced by Publisher.Publish includes the step name so
// operators can locate the failure in logs. Pre-fix Publisher returned the
// raw error (which can be empty for tar/io.Pipe failure modes), producing
// the empty-500 observed on atlas-main 2026-05-22.
func TestPublishErrorIsContextualized(t *testing.T) {
	// nil DB triggers an early failure in dumpTable.
	p := Publisher{DB: nil, MC: nil, L: logrus.New()}
	_, err := p.Publish(context.Background(), "GMS", 83, 1)
	if err == nil {
		t.Fatal("expected error from Publish with nil deps")
	}
	if !strings.Contains(err.Error(), "publish:") {
		t.Fatalf("error %q lacks `publish:` step prefix", err.Error())
	}
}
```

- [ ] **Step 3: Run test to verify it fails.**

```bash
( cd services/atlas-data/atlas.com/data && go test ./baseline -run TestPublishErrorIsContextualized -v )
```

Expected: FAIL (current code returns unwrapped error or panics on nil MC).

- [ ] **Step 4: Refactor `publish.go` to buffer the tar to a temp file and log/wrap each step.**

Replace the existing `Publish` method body in `services/atlas-data/atlas.com/data/baseline/publish.go` with the following implementation. The pattern mirrors `restore.go`'s temp-file staging and produces a typed wrap at every step.

```go
// Publish builds a tar of header.json + one COPY-binary entry per table to a
// temp file, computes the sha256, uploads the tar plus a sha sidecar to the
// canonical bucket, and returns the hex-encoded sha256.
//
// The earlier implementation streamed via io.Pipe directly into MC.Put; when
// any step in the writer goroutine returned an error, MC.Put could finish
// with the half-written body and the handler would surface an empty error.
// Buffering to a temp file makes the steps observable and lets MC.Put run
// with a known Content-Length.
func (p Publisher) Publish(ctx context.Context, region string, major, minor int) (string, error) {
	p.L.Infof("publish: start region=%s ver=%d.%d", region, major, minor)

	tmp, err := os.CreateTemp("", "baseline-publish-*.tar")
	if err != nil {
		return "", fmt.Errorf("publish: create-tempfile: %w", err)
	}
	defer os.Remove(tmp.Name())
	defer tmp.Close()

	h := sha256.New()
	tw := tar.NewWriter(io.MultiWriter(tmp, h))

	hdr := Header{
		SchemaVersion: SchemaVersion,
		Region:        region,
		MajorVersion:  major,
		MinorVersion:  minor,
		Tables:        DumpTables,
		PublishedAt:   time.Unix(0, 0).UTC(),
	}
	hdrBytes, err := MarshalHeader(hdr)
	if err != nil {
		return "", fmt.Errorf("publish: marshal-header: %w", err)
	}
	if err := writeTarEntry(tw, "header.json", hdrBytes); err != nil {
		return "", fmt.Errorf("publish: write-header: %w", err)
	}
	for _, table := range DumpTables {
		p.L.Debugf("publish: dump-table %s", table)
		if err := dumpTable(ctx, p.DB, table, tw); err != nil {
			return "", fmt.Errorf("publish: dump-table %s: %w", table, err)
		}
	}
	if err := tw.Close(); err != nil {
		return "", fmt.Errorf("publish: close-tar: %w", err)
	}

	size, err := tmp.Seek(0, io.SeekEnd)
	if err != nil {
		return "", fmt.Errorf("publish: seek-end: %w", err)
	}
	if _, err := tmp.Seek(0, io.SeekStart); err != nil {
		return "", fmt.Errorf("publish: seek-start: %w", err)
	}

	p.L.Infof("publish: upload tar size=%d", size)
	if err := p.MC.Put(ctx, p.MC.Cfg().BucketCanonical, DumpKey(region, major, minor), tmp, size, "application/x-tar"); err != nil {
		return "", fmt.Errorf("publish: put-tar: %w", err)
	}

	sum := hex.EncodeToString(h.Sum(nil))
	if err := p.MC.Put(ctx, p.MC.Cfg().BucketCanonical, ShaKey(region, major, minor), strReader(sum), int64(len(sum)), "text/plain"); err != nil {
		return "", fmt.Errorf("publish: put-sha: %w", err)
	}
	p.L.Infof("publish: ok sha=%s", sum)
	return sum, nil
}
```

Required new imports in `publish.go`: add `"os"` to the existing import block.

- [ ] **Step 5: Surface the error in the handler.**

In `services/atlas-data/atlas.com/data/baseline/handler.go:39-43`, replace:

```go
sum, err := (Publisher{DB: db, MC: mc, L: d.Logger()}).Publish(r.Context(), input.Region, input.MajorVersion, input.MinorVersion)
if err != nil {
    http.Error(w, err.Error(), http.StatusInternalServerError)
    return
}
```

with:

```go
sum, err := (Publisher{DB: db, MC: mc, L: d.Logger()}).Publish(r.Context(), input.Region, input.MajorVersion, input.MinorVersion)
if err != nil {
    d.Logger().WithError(err).Errorf("baseline publish failed")
    http.Error(w, fmt.Sprintf("publish failed: %s", err.Error()), http.StatusInternalServerError)
    return
}
```

Add `"fmt"` to `handler.go`'s import block if not already present.

- [ ] **Step 6: Run the test to verify it passes.**

```bash
( cd services/atlas-data/atlas.com/data && go test ./baseline -run TestPublishErrorIsContextualized -v )
```

Expected: PASS.

- [ ] **Step 7: Run the full baseline test suite + race + vet + build.**

```bash
( cd services/atlas-data/atlas.com/data && go test -race ./baseline && go vet ./baseline && go build ./... )
```

Expected: all pass.

- [ ] **Step 8: Re-run the curl recipe against the fixed binary (or local atlas-data).**

If you ran step 1 against atlas-main, re-deploy the changed binary first (`docker buildx bake atlas-data` from worktree root then push via the standard image-bump flow). For a local repro, restart atlas-data. Capture the post-fix log alongside the pre-fix log:

```bash
# append to docs/tasks/task-076-task071-followups/repro/f1-publish-500.log
```

Expected: HTTP 202 with a JSON:API envelope `{"data":{"type":"baselinePublishes","id":"GMS/83.1","attributes":{"sha256":"..."}}}` OR (if the upstream issue was MinIO-side) a 500 whose body now starts with `publish failed: publish: <step>:`.

- [ ] **Step 9: Commit.**

```bash
git add services/atlas-data/atlas.com/data/baseline/publish.go \
        services/atlas-data/atlas.com/data/baseline/handler.go \
        services/atlas-data/atlas.com/data/baseline/publish_test.go \
        docs/tasks/task-076-task071-followups/repro/f1-publish-500.log
git commit -m "fix(atlas-data): F1 buffer publish tar to tempfile and surface step errors"
```

---

### Task 2: F7 — Pin atlas-renders in main overlay

**Files:**
- Modify: `deploy/k8s/overlays/main/kustomization.yaml:179-291` (images list)

- [ ] **Step 1: Pick the SHA.**

The current atlas-data pin is `main-68f755b` (line 207). Use the same SHA for atlas-renders so renovate's next bump sweeps it along with the others. Confirm the SHA is a real built image:

```bash
gh api -H "Accept: application/vnd.github+json" \
  /users/chronicle20/packages/container/atlas-renders%2Fatlas-renders/versions \
  --jq '.[].metadata.container.tags[]' | grep -E '^main-' | head -3
```

If `main-68f755b` is not in the list, pick the most recent `main-<sha>` tag instead.

- [ ] **Step 2: Insert the pin alphabetically between atlas-reactors (line 276–277) and atlas-saga-orchestrator (line 278–279).**

In `deploy/k8s/overlays/main/kustomization.yaml`, between the existing entries:

```yaml
  - name: ghcr.io/chronicle20/atlas-reactors/atlas-reactors
    newTag: main-72332d8
  - name: ghcr.io/chronicle20/atlas-saga-orchestrator/atlas-saga-orchestrator
    newTag: main-72332d8
```

Insert:

```yaml
  - name: ghcr.io/chronicle20/atlas-renders/atlas-renders
    newTag: main-68f755b
```

So the resulting block reads:

```yaml
  - name: ghcr.io/chronicle20/atlas-reactors/atlas-reactors
    newTag: main-72332d8
  - name: ghcr.io/chronicle20/atlas-renders/atlas-renders
    newTag: main-68f755b
  - name: ghcr.io/chronicle20/atlas-saga-orchestrator/atlas-saga-orchestrator
    newTag: main-72332d8
```

- [ ] **Step 3: Validate kustomize build still resolves cleanly.**

```bash
kustomize build deploy/k8s/overlays/main > /tmp/main-built.yaml && grep -A1 'atlas-renders/atlas-renders' /tmp/main-built.yaml | head -6
```

Expected: the rendered Deployment for atlas-renders shows the pinned image with the chosen SHA.

- [ ] **Step 4: Commit.**

```bash
git add deploy/k8s/overlays/main/kustomization.yaml
git commit -m "feat(deploy): F7 pin atlas-renders in main overlay images list"
```

---

### Task 3: F3 — Don't cache "shared" verdicts in scope/smap resolvers

**Files:**
- Modify: `services/atlas-renders/atlas.com/renders/storage/scope.go:18-34`
- Modify: `services/atlas-renders/atlas.com/renders/storage/smap.go:61-76`
- Create: `services/atlas-renders/atlas.com/renders/storage/scope_test.go`

- [ ] **Step 1: Write the failing test for scope.go.**

Create `services/atlas-renders/atlas.com/renders/storage/scope_test.go`:

```go
package storage

import (
	"context"
	"testing"
)

// fakeHasAny lets a test alternate between negative and positive HEAD probe
// results without touching MinIO. The boolean slice is consumed in order.
type fakeHasAny struct {
	calls   int
	results []bool
}

func (f *fakeHasAny) HasAny(_ context.Context, _, _ string) (bool, error) {
	if f.calls >= len(f.results) {
		return false, nil
	}
	out := f.results[f.calls]
	f.calls++
	return out, nil
}

// TestResolveScopeDoesNotCacheNegative asserts that a probe returning
// "shared" is NOT cached: a subsequent probe sees the next HasAny result
// rather than the pinned "shared".
//
// Pre-fix behavior pinned "shared" for the pod lifetime; ingest publishing
// tenant data later did not propagate, requiring a pod restart on PR-544.
func TestResolveScopeDoesNotCacheNegative(t *testing.T) {
	// Construct an isolated Storage whose probe is replaceable. The
	// production Storage holds *minio.Client; the test exercises the
	// caching gate via a helper that wraps the same logic. If Storage
	// is refactored later to take an interface for the probe, switch to
	// injecting fakeHasAny directly.
	t.Skip("placeholder — implemented in step 3 after extracting the gate")
}
```

(The skipped test acts as a stub; the real test goes in step 3 once we know whether to inject a probe interface or to exercise the gate at a lower level.)

- [ ] **Step 2: Inspect Storage to decide injection shape.**

```bash
sed -n '1,60p' services/atlas-renders/atlas.com/renders/storage/storage.go
grep -n "HasAny\|s.MC" services/atlas-renders/atlas.com/renders/storage/*.go | head -30
```

If `s.MC` is a concrete `*minio.Client`, the simplest test path is to extract the cache-gate decision into a small private helper and unit-test that. The gate is one line: "only Add to cache if has==true."

- [ ] **Step 3: Implement the fix in `scope.go`.**

Replace lines 28–33 in `services/atlas-renders/atlas.com/renders/storage/scope.go`:

```go
	scope := "shared"
	if has {
		scope = "tenants/" + tenantID
	}
	s.Caches.Scope.Add(cacheKey, scope)
	return scope, nil
```

with:

```go
	if has {
		scope := "tenants/" + tenantID
		s.Caches.Scope.Add(cacheKey, scope)
		return scope, nil
	}
	// Do not cache the "shared" verdict. A negative probe becomes wrong
	// the moment ingest publishes tenant data, and the cost of re-probing
	// is one MinIO HEAD-list per non-cached path (acceptable; the steady
	// state for most maps is shared). Asymmetric with positive caching:
	// once a tenant scope is observed it cannot become "shared" without
	// an explicit teardown flush.
	return "shared", nil
```

- [ ] **Step 4: Implement the same fix in `smap.go`.**

In `services/atlas-renders/atlas.com/renders/storage/smap.go:67-75`, replace:

```go
	if has, err := s.MC.Stat(ctx, s.Cfg.BucketAssets, tenantKey); err == nil && has {
		s.Caches.Scope.Add(cacheKey, "tenants/"+tenantID)
		return "tenants/" + tenantID, nil
	}
	// Fall back to shared scope. We don't HEAD-probe shared first because
	// almost every deployment relies on the shared sidecar — a miss here is
	// recoverable (atlas-renders disables occlusion and logs a warning).
	s.Caches.Scope.Add(cacheKey, "shared")
	return "shared", nil
```

with:

```go
	if has, err := s.MC.Stat(ctx, s.Cfg.BucketAssets, tenantKey); err == nil && has {
		s.Caches.Scope.Add(cacheKey, "tenants/"+tenantID)
		return "tenants/" + tenantID, nil
	}
	// Fall back to shared scope. We don't cache this verdict — see F3 in
	// docs/tasks/task-076-task071-followups: a negative probe becomes
	// wrong the moment ingest publishes the smap sidecar for this tenant.
	return "shared", nil
```

- [ ] **Step 5: Replace the placeholder test with a real one.**

Rewrite `services/atlas-renders/atlas.com/renders/storage/scope_test.go` to exercise the negative-skip behavior end-to-end. Pick the approach that matches the actual Storage shape (verified in step 2):

If `s.MC.HasAny` can be replaced with an interface, define an inline interface in the test file and inject a fake. If not, write a behavioural test that constructs a Storage with an in-memory MinIO double (compose's `minio:RELEASE.2024-...` is already a dependency). The test must:

1. Probe (tenant=t1, region=GMS, version=83.1, subPath=map/100000000) — assert result is "shared".
2. Add an object under the tenant prefix (the test's MinIO double or a stub HasAny that returns true the second call).
3. Probe again — assert result is "tenants/t1" without any cache invalidation call.

```go
func TestResolveScopeReprobesOnNegative(t *testing.T) {
	// Pseudocode — replace MC with the storage_test.go helper that
	// constructs a real or doubled Storage. See storage_test.go for the
	// existing pattern.
	st := newTestStorage(t)
	probeFn := func() (string, error) {
		return st.ResolveScope(context.Background(), "t1", "GMS", "83.1", "map/100000000")
	}

	scope, err := probeFn()
	if err != nil { t.Fatal(err) }
	if scope != "shared" {
		t.Fatalf("first probe = %q, want shared", scope)
	}

	st.putObject(t, "tenants/t1/regions/GMS/versions/83.1/map/100000000/layout.json", []byte(`{}`))

	scope, err = probeFn()
	if err != nil { t.Fatal(err) }
	if scope != "tenants/t1" {
		t.Fatalf("second probe after publish = %q, want tenants/t1", scope)
	}
}
```

If `newTestStorage` / `putObject` helpers don't exist, fall back to a minimal interface-injection unit test using a `probeFn` closure passed into a refactored package-private helper:

```go
// in scope.go:
func resolveCacheGate(cache *lru.Cache, key, tenantID string, has bool) (string, bool /*shouldCache*/) {
    if has {
        return "tenants/" + tenantID, true
    }
    return "shared", false
}
```

And test `resolveCacheGate` directly. This keeps the change reviewable without inventing an interface where none existed.

- [ ] **Step 6: Verify tests pass + race + vet + build.**

```bash
( cd services/atlas-renders/atlas.com/renders && go test -race ./storage && go vet ./storage && go build ./... )
```

Expected: all pass.

- [ ] **Step 7: Commit.**

```bash
git add services/atlas-renders/atlas.com/renders/storage/scope.go \
        services/atlas-renders/atlas.com/renders/storage/smap.go \
        services/atlas-renders/atlas.com/renders/storage/scope_test.go
git commit -m "fix(atlas-renders): F3 stop caching negative scope verdicts"
```

---

### Task 4: F2 — Commodity worker per-row commits (or chunked)

**Files:**
- Modify: `services/atlas-data/atlas.com/data/commodity/processor.go:36-46`
- Create: `services/atlas-data/atlas.com/data/commodity/processor_test.go`

- [ ] **Step 1: Inspect the storage shape.**

```bash
sed -n '1,80p' services/atlas-data/atlas.com/data/document/storage.go
```

Confirm whether `Storage.Add(ctx)(m)()` opens its own transaction or relies on the caller's. If it opens its own, dropping the outer `ExecuteTransaction` already chunks per-row.

- [ ] **Step 2: Write the failing test (chunking guarantee).**

Create `services/atlas-data/atlas.com/data/commodity/processor_test.go`:

```go
package commodity

import (
	"context"
	"errors"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/sirupsen/logrus"
)

// TestRegisterCommitsPerItem asserts that a Register call which fails on the
// Nth item still commits items 1..N-1. Pre-fix, the entire import was
// wrapped in one ExecuteTransaction; a failure or conn drop rolled back
// every row.
func TestRegisterCommitsPerItem(t *testing.T) {
	// Build a stub Storage whose Add fails on the second item.
	calls := 0
	s := &stubStorage{addFn: func(_ context.Context, _ RestModel) error {
		calls++
		if calls == 2 {
			return errors.New("simulated conn drop")
		}
		return nil
	}}
	items := []RestModel{{Id: "1"}, {Id: "2"}, {Id: "3"}}
	provider := func() ([]RestModel, error) { return items, nil }

	err := registerWithStorage(s, logrus.New(), context.Background(), provider)
	if err == nil {
		t.Fatal("expected error from failing second item")
	}
	if s.committed != 1 {
		t.Fatalf("committed = %d, want 1 (only first item before failure)", s.committed)
	}
}

type stubStorage struct {
	addFn     func(context.Context, RestModel) error
	committed int
}

func (s *stubStorage) Add(ctx context.Context, m RestModel) error {
	if err := s.addFn(ctx, m); err != nil {
		return err
	}
	s.committed++
	return nil
}

// registerWithStorage is the chunked Register shape; the test compiles only
// if the production code exposes it (see step 3).
func registerWithStorageBound(s *stubStorage) func(logrus.FieldLogger, context.Context, model.Provider[[]RestModel]) error {
	return func(l logrus.FieldLogger, ctx context.Context, p model.Provider[[]RestModel]) error {
		_ = l
		ms, err := p()
		if err != nil {
			return err
		}
		for _, m := range ms {
			if err := s.Add(ctx, m); err != nil {
				return err
			}
		}
		return nil
	}
}

// registerWithStorage adapter used by the test above. Wires to the package-
// private helper introduced in step 3.
var registerWithStorage = func(s *stubStorage, l logrus.FieldLogger, ctx context.Context, p func() ([]RestModel, error)) error {
	return registerWithStorageBound(s)(l, ctx, p)
}
```

- [ ] **Step 3: Run test to verify it fails (compile or behavior).**

```bash
( cd services/atlas-data/atlas.com/data && go test ./commodity -run TestRegisterCommitsPerItem -v )
```

If the test compiles, it should still PASS — the existing `Register` already loops per item. The failing piece is the **transaction wrapping** in `RegisterCommodity`. So we need a test that pinpoints transaction boundaries, not the loop. Replace the body above with the following sharper test that pulls in `sqlmock` if available, or asserts that `RegisterCommodity` no longer wraps in `database.ExecuteTransaction`:

```go
// TestRegisterCommodityDoesNotWrapInOuterTx asserts the production
// RegisterCommodity uses per-row commits rather than one outer
// ExecuteTransaction across the whole Etc.wz import.
//
// The test is a structural check: we grep the function body via go/ast to
// confirm `database.ExecuteTransaction` is not present.
func TestRegisterCommodityDoesNotWrapInOuterTx(t *testing.T) {
	src, err := os.ReadFile("processor.go")
	if err != nil { t.Fatal(err) }
	if strings.Contains(string(src), "database.ExecuteTransaction") {
		t.Fatal("processor.go still wraps Register in a single ExecuteTransaction; chunk per-item commits required (see F2)")
	}
}
```

Add imports `"os"` and `"strings"` to the test.

Run:

```bash
( cd services/atlas-data/atlas.com/data && go test ./commodity -run TestRegisterCommodityDoesNotWrapInOuterTx -v )
```

Expected: FAIL (`database.ExecuteTransaction` is currently present).

- [ ] **Step 4: Implement the fix.**

Rewrite `services/atlas-data/atlas.com/data/commodity/processor.go`:

```go
package commodity

import (
	"atlas-data/document"
	"atlas-data/xml"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func NewStorage(l logrus.FieldLogger, db *gorm.DB) *document.Storage[string, RestModel] {
	return document.NewStorage(l, db, GetModelRegistry(), "COMMODITY")
}

// Register adds each item via the storage's per-call commit. No outer
// transaction wraps the loop — a connection drop or single-row failure now
// preserves successfully-committed rows so a retry can converge.
//
// See task-076 F2: the prior outer ExecuteTransaction wrapped the entire
// Etc.wz import in one long-lived transaction and was fatal to any conn
// blip across a multi-thousand-row register.
func Register(s *document.Storage[string, RestModel]) func(ctx context.Context) func(r model.Provider[[]RestModel]) error {
	return func(ctx context.Context) func(r model.Provider[[]RestModel]) error {
		return func(r model.Provider[[]RestModel]) error {
			ms, err := r()
			if err != nil {
				return err
			}
			for _, m := range ms {
				if _, err := s.Add(ctx)(m)(); err != nil {
					return err
				}
			}
			return nil
		}
	}
}

func RegisterCommodity(db *gorm.DB) func(l logrus.FieldLogger) func(ctx context.Context) func(path string) error {
	return func(l logrus.FieldLogger) func(ctx context.Context) func(path string) error {
		return func(ctx context.Context) func(path string) error {
			return func(path string) error {
				return Register(NewStorage(l, db))(ctx)(Read(l)(xml.FromPathProvider(path)))
			}
		}
	}
}
```

Note: drop the `database "github.com/Chronicle20/atlas/libs/atlas-database"` import along with the wrap.

- [ ] **Step 5: Run test to verify it passes.**

```bash
( cd services/atlas-data/atlas.com/data && go test ./commodity -run TestRegisterCommodityDoesNotWrapInOuterTx -v )
```

Expected: PASS.

- [ ] **Step 6: Run module-wide verification.**

```bash
( cd services/atlas-data/atlas.com/data && go test -race ./commodity && go vet ./commodity && go build ./... )
```

Expected: all pass.

- [ ] **Step 7: Commit.**

```bash
git add services/atlas-data/atlas.com/data/commodity/processor.go \
        services/atlas-data/atlas.com/data/commodity/processor_test.go
git commit -m "fix(atlas-data): F2 drop outer transaction in commodity register"
```

---

### Task 5: F5 — Restore atomicity via two-phase finalization

**Files:**
- Modify: `services/atlas-data/atlas.com/data/baseline/restore.go:44-128`
- Create: `services/atlas-data/atlas.com/data/baseline/restore_failure_test.go`

- [ ] **Step 1: Write the failing test.**

Create `services/atlas-data/atlas.com/data/baseline/restore_failure_test.go`:

```go
package baseline

import (
	"context"
	"os"
	"strings"
	"testing"
)

// TestRestoreDoesNotMarkOnPartialFailure asserts a mid-restore failure
// leaves NO tenant_baselines row for the target. Pre-fix, per-table
// transactions committed individually and the final UPSERT into
// tenant_baselines could run even if downstream tables had already been
// partially DELETE+COPY'd, yielding half-restored data the marker called
// "ready".
//
// Implementation: structural assertion that Restore() defers the marker
// UPSERT until after the loop completes successfully. We grep restore.go
// for the UPSERT being inside a deferred cleanup-on-error path or after
// every table loop iteration succeeds.
func TestRestoreDeferredMarkerStructure(t *testing.T) {
	src, err := os.ReadFile("restore.go")
	if err != nil {
		t.Fatal(err)
	}
	body := string(src)
	if !strings.Contains(body, "cleanupAfterFailure") {
		t.Fatal("restore.go missing cleanupAfterFailure helper required by F5")
	}
	// The UPSERT must be the last statement in the success path.
	idxMarker := strings.Index(body, "INSERT INTO tenant_baselines")
	idxLoopEnd := strings.LastIndex(body, "restoreOneTable(")
	if idxMarker < idxLoopEnd {
		t.Fatal("tenant_baselines UPSERT appears before the restoreOneTable loop; F5 requires it to run only after all tables succeed")
	}
}
```

- [ ] **Step 2: Run test to verify it fails.**

```bash
( cd services/atlas-data/atlas.com/data && go test ./baseline -run TestRestoreDeferredMarkerStructure -v )
```

Expected: FAIL (`cleanupAfterFailure` not defined).

- [ ] **Step 3: Implement the fix.**

Rewrite the body of `Restore` (starting at the comment `// 3) Mutations only after both gates pass.` around line 94) in `services/atlas-data/atlas.com/data/baseline/restore.go`:

```go
	// 3) Mutations only after both gates pass. Two-phase finalization: per-
	//    table transactions are unchanged; the tenant_baselines UPSERT is
	//    deferred until every table loop iteration AND every ANALYZE
	//    succeeds. A mid-restore failure triggers cleanupAfterFailure to
	//    DELETE every DumpTables row for `target` so subsequent reads see
	//    "never restored" rather than half-restored.
	if err := runRestoreTables(ctx, r.DB, tr, target); err != nil {
		r.L.WithError(err).Warnf("restore: table loop failed for target=%s region=%s ver=%d.%d; cleaning partial state", target, region, major, minor)
		cleanupAfterFailure(ctx, r.L, r.DB, target)
		return err
	}

	// ANALYZE all tables — failure here also triggers cleanup, since
	// readers consume tables that were re-populated above.
	for _, t := range DumpTables {
		if err := r.DB.WithContext(ctx).Exec("ANALYZE " + t).Error; err != nil {
			r.L.WithError(err).Warnf("restore: ANALYZE %s failed; cleaning partial state", t)
			cleanupAfterFailure(ctx, r.L, r.DB, target)
			return err
		}
	}

	// UPSERT tenant_baselines — the only finalization step. After this
	// returns nil the restore is observable to downstream readers.
	if err := r.DB.WithContext(ctx).Exec(`
        INSERT INTO tenant_baselines (tenant_id, region, major_version, minor_version, baseline_sha256, restored_at)
        VALUES (?, ?, ?, ?, ?, now())
        ON CONFLICT (tenant_id) DO UPDATE SET region=EXCLUDED.region, major_version=EXCLUDED.major_version,
            minor_version=EXCLUDED.minor_version, baseline_sha256=EXCLUDED.baseline_sha256, restored_at=now()
    `, target.String(), region, major, minor, expectedSum).Error; err != nil {
		return err
	}
	r.L.Infof("restore: finalized target=%s region=%s ver=%d.%d sha=%s", target, region, major, minor, expectedSum)
	return nil
}

// runRestoreTables consumes the tar reader and dispatches each table entry
// to restoreOneTable. Pulled out of Restore so the marker UPSERT can be
// deferred until every entry succeeds.
func runRestoreTables(ctx context.Context, db *gorm.DB, tr *tar.Reader, target uuid.UUID) error {
	for {
		e, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		table := strings.TrimSuffix(e.Name, ".binary")
		if !contains(DumpTables, table) {
			return fmt.Errorf("unexpected table %s", table)
		}
		if err := restoreOneTable(ctx, db, table, tr, target); err != nil {
			return fmt.Errorf("restore table %s: %w", table, err)
		}
	}
}

// cleanupAfterFailure DELETEs every DumpTables row for target in its own
// transaction so a subsequent restore is not blocked by stale rows. Best-
// effort: if the cleanup itself errors (e.g., DB unreachable mid-cleanup),
// the warning is logged and the original restore error is still returned
// to the caller. The next successful restore's per-table DELETE will sweep
// residue.
func cleanupAfterFailure(ctx context.Context, l logrus.FieldLogger, db *gorm.DB, target uuid.UUID) {
	for _, t := range DumpTables {
		if err := db.WithContext(ctx).Exec("DELETE FROM "+t+" WHERE tenant_id = ?", target.String()).Error; err != nil {
			l.WithError(err).Warnf("restore: cleanup DELETE FROM %s failed (best-effort)", t)
		}
	}
}
```

Now delete the original loop and ANALYZE/UPSERT block that lived at lines 94–127 of the pre-fix file (replaced by the above).

- [ ] **Step 4: Run test to verify it passes.**

```bash
( cd services/atlas-data/atlas.com/data && go test ./baseline -run TestRestoreDeferredMarkerStructure -v )
```

Expected: PASS.

- [ ] **Step 5: Verify module-wide.**

```bash
( cd services/atlas-data/atlas.com/data && go test -race ./baseline && go vet ./baseline && go build ./... )
```

Expected: all pass.

- [ ] **Step 6: Commit.**

```bash
git add services/atlas-data/atlas.com/data/baseline/restore.go \
        services/atlas-data/atlas.com/data/baseline/restore_failure_test.go
git commit -m "fix(atlas-data): F5 two-phase finalize baseline restore"
```

---

## Wave 2 — Debt

### Task 6: F8 — Dedupe routes-config via kustomize configMapGenerator

**Files:**
- Create: `tools/gen-routes.sh`
- Modify: `deploy/k8s/base/atlas-ingress.yaml` (remove inline `routes.conf.template`, keep only `nginx.conf` ConfigMap; switch routes mount to the generated CM)
- Modify: `deploy/k8s/base/kustomization.yaml` (add `configMapGenerator`)
- Create: `deploy/k8s/base/routes.conf.template.generated` (output of `gen-routes.sh`, committed for reproducibility)

- [ ] **Step 1: Create the generator script.**

Write `tools/gen-routes.sh`:

```bash
#!/usr/bin/env bash
# Generates deploy/k8s/base/routes.conf.template.generated from
# deploy/shared/routes.conf by rewriting bare service hostnames to FQDNs
# templated on ${POD_NAMESPACE}. See task-076 F8.
set -euo pipefail

REPO_ROOT="$(git rev-parse --show-toplevel)"
SRC="$REPO_ROOT/deploy/shared/routes.conf"
OUT="$REPO_ROOT/deploy/k8s/base/routes.conf.template.generated"

if [[ ! -f "$SRC" ]]; then
  echo "missing $SRC" >&2
  exit 1
fi

# Rewrite `set $u "atlas-XXX:8080"` and `proxy_pass http://atlas-XXX:8080...`
# to use `.${POD_NAMESPACE}.svc.cluster.local`. The `minio:9000` upstream is
# already namespace-qualified in the shared file (per the F8/F18 cross-ns
# guard), so we leave that untouched.
sed -E \
  -e 's|set \$u "(atlas-[a-z-]+):8080"|set $u "\1.${POD_NAMESPACE}.svc.cluster.local:8080"|g' \
  -e 's|proxy_pass http://(atlas-[a-z-]+):8080|proxy_pass http://\1.${POD_NAMESPACE}.svc.cluster.local:8080|g' \
  -e 's|set \$u "atlas-ui:80"|set $u "atlas-ui.${POD_NAMESPACE}.svc.cluster.local:80"|g' \
  "$SRC" > "$OUT"

echo "wrote $OUT ($(wc -l <"$OUT") lines)"
```

Make it executable: `chmod +x tools/gen-routes.sh`.

- [ ] **Step 2: Run it and inspect.**

```bash
bash tools/gen-routes.sh
diff <(grep -E '^\s*location' deploy/shared/routes.conf | sort) \
     <(grep -E '^\s*location' deploy/k8s/base/routes.conf.template.generated | sort)
```

Expected: empty diff (location lines match exactly).

- [ ] **Step 3: Update the kustomize base to consume the generated file.**

In `deploy/k8s/base/atlas-ingress.yaml`, delete the entire `routes.conf.template: |` block (lines ~46–497) from the inline `atlas-ingress-configmap` ConfigMap. The ConfigMap should now contain ONLY the `nginx.conf:` key.

The Deployment's `volumeMounts` already mount `routes.conf.template` from `atlas-ingress-configmap` (line 555 — `mountPath: /etc/nginx/templates/routes.conf.template`, `subPath: routes.conf.template`). After this change, that `subPath` will be missing from `atlas-ingress-configmap`, so we'll redirect it.

In `deploy/k8s/base/kustomization.yaml`, add a `configMapGenerator` entry. First inspect what's there:

```bash
cat deploy/k8s/base/kustomization.yaml
```

Then append (or extend the existing `configMapGenerator:` list):

```yaml
configMapGenerator:
  - name: atlas-ingress-routes
    files:
      - routes.conf.template=routes.conf.template.generated
```

Update the Deployment volume mount in `atlas-ingress.yaml` to mount from the new ConfigMap. Replace lines 549–555:

```yaml
        volumeMounts:
        - name: nginx-conf-volume
          mountPath: /etc/nginx/nginx.conf
          subPath: nginx.conf
        - name: nginx-conf-volume
          mountPath: /etc/nginx/templates/routes.conf.template
          subPath: routes.conf.template
```

with:

```yaml
        volumeMounts:
        - name: nginx-conf-volume
          mountPath: /etc/nginx/nginx.conf
          subPath: nginx.conf
        - name: nginx-routes-volume
          mountPath: /etc/nginx/templates/routes.conf.template
          subPath: routes.conf.template
```

And replace the volumes block (lines 556–559):

```yaml
      volumes:
      - name: nginx-conf-volume
        configMap:
          name: atlas-ingress-configmap
```

with:

```yaml
      volumes:
      - name: nginx-conf-volume
        configMap:
          name: atlas-ingress-configmap
      - name: nginx-routes-volume
        configMap:
          name: atlas-ingress-routes
```

- [ ] **Step 4: Validate kustomize build.**

```bash
kustomize build deploy/k8s/base > /tmp/base-built.yaml
grep -A2 'atlas-ingress-routes' /tmp/base-built.yaml | head -10
```

Expected: the generated ConfigMap appears with a hash suffix (kustomize default behavior) and contains the route block.

Kustomize may append a name suffix (e.g., `atlas-ingress-routes-abc123`). The Deployment's volume reference must match — kustomize's `configMapGenerator` automatically rewrites references to the generated name. Confirm:

```bash
grep -B1 -A3 'configMap:' /tmp/base-built.yaml | grep -A1 atlas-ingress-routes
```

- [ ] **Step 5: Run all deploy-side tests.**

```bash
bash deploy/shared/test/routes_nginxt.sh
kustomize build deploy/k8s/overlays/main > /dev/null
```

Expected: nginx-t passes (already does); kustomize build succeeds; the generated ConfigMap is present in the main overlay output.

- [ ] **Step 6: Commit.**

```bash
git add tools/gen-routes.sh \
        deploy/k8s/base/atlas-ingress.yaml \
        deploy/k8s/base/kustomization.yaml \
        deploy/k8s/base/routes.conf.template.generated
git commit -m "fix(deploy): F8 dedupe routes-config via kustomize configMapGenerator"
```

---

### Task 7: F18 — Routes drift validation in CI

**Files:**
- Modify: `deploy/shared/test/routes_nginxt.sh`

- [ ] **Step 1: Add a drift check that fails the script when the generated file is stale.**

Edit `deploy/shared/test/routes_nginxt.sh` to add a final drift check after the existing python tenant-header check (after the `PY` heredoc ends, around line 102):

```bash

# F18: confirm the generated k8s routes file is in sync with the canonical
# shared source. If shared/routes.conf changes, the committer MUST also run
# tools/gen-routes.sh and commit the resulting routes.conf.template.generated.
GEN="$REPO_ROOT/deploy/k8s/base/routes.conf.template.generated"
if [[ -f "$GEN" ]]; then
  EXPECTED=$(mktemp)
  trap 'rm -f "$EXPECTED" "$TMPDIR/expected"' EXIT
  # Re-run the generator into a scratch file (don't overwrite the committed
  # artifact). Compare with the committed copy.
  bash "$REPO_ROOT/tools/gen-routes.sh" >/dev/null
  # gen-routes.sh writes the committed file in place; capture a fresh
  # version into a temp and restore the committed copy from git afterwards.
  cp "$GEN" "$EXPECTED"
  git -C "$REPO_ROOT" diff --quiet -- deploy/k8s/base/routes.conf.template.generated || {
    echo "error: deploy/shared/routes.conf changed but routes.conf.template.generated is stale." >&2
    echo "       run tools/gen-routes.sh and commit the result." >&2
    git -C "$REPO_ROOT" --no-pager diff -- deploy/k8s/base/routes.conf.template.generated | head -40 >&2
    exit 1
  }
  echo "routes drift check (shared vs k8s-generated): OK"
else
  echo "warn: $GEN does not exist; skipping F18 drift check (was Task 6 / F8 applied?)" >&2
fi
```

- [ ] **Step 2: Run the script and confirm it passes when files are in sync.**

```bash
bash deploy/shared/test/routes_nginxt.sh
```

Expected: all three checks pass (`nginx -t`, MinIO cross-ns, atlas-renders headers, drift).

- [ ] **Step 3: Simulate divergence and confirm the script fails.**

```bash
echo "" >> deploy/shared/routes.conf
bash deploy/shared/test/routes_nginxt.sh && echo "BUG: drift check did not catch divergence" || echo "drift check correctly failed"
git checkout -- deploy/shared/routes.conf  # restore
```

Expected: the second `bash` invocation exits non-zero with the "stale" message.

- [ ] **Step 4: Commit.**

```bash
git add deploy/shared/test/routes_nginxt.sh
git commit -m "fix(deploy): F18 validate generated k8s routes file in CI"
```

---

### Task 8: F6 — `Properties()` returns `([]Property, error)` (atomic monorepo update)

**Files:**
- Modify: `libs/atlas-wz/wz/image.go:43-73`
- Modify all 18 call sites listed in `context.md` under F6.

- [ ] **Step 1: Change the signature in `image.go`.**

In `libs/atlas-wz/wz/image.go`, replace the existing `Properties()` body (lines 43–73) with:

```go
// Properties returns the parsed properties of this image plus any error
// observed during lazy parsing. Parses on first access; subsequent calls
// return the cached result.
//
// Returning the error surface forces every caller to make an explicit
// decision about parse failures instead of silently consuming an empty
// property slice. See task-076 F6: a previous version logged the error
// and dropped it, which made downstream zero-row imports indistinguishable
// from "parsed and genuinely empty."
//
// Goroutine safety: parse() Seek+Reads the shared *os.File; the file-wide
// parseMu in *File serialises every Seek-based parse. The double-check
// inside the critical section makes the lazy initialisation idempotent.
//
// In-memory images created via NewParsedImage have parsed=true and
// wzFile=nil, so they short-circuit without lock acquisition and always
// return a nil error.
func (i *Image) Properties() ([]property.Property, error) {
	if i.parsed {
		return i.properties, i.parseErr
	}
	if i.wzFile == nil {
		i.parsed = true
		return i.properties, nil
	}
	unlock := i.wzFile.LockParse()
	defer unlock()
	if i.parsed {
		return i.properties, i.parseErr
	}
	if err := i.parse(); err != nil {
		i.wzFile.l.WithError(err).Warnf("Unable to parse image [%s].", i.name)
		i.parsed = true
		i.parseErr = err
		return i.properties, err
	}
	i.parsed = true
	return i.properties, nil
}
```

Add `parseErr error` to the `Image` struct (line 10 area). After modification it reads:

```go
type Image struct {
	name       string
	wzFile     *File
	dataOffset int64
	dataSize   int32
	properties []property.Property
	parsed     bool
	parseErr   error
}
```

- [ ] **Step 2: Run `go build ./...` from `libs/atlas-wz` and capture the compile errors.**

```bash
( cd libs/atlas-wz && go build ./... 2>&1 | tee /tmp/f6-build-errs.txt | head -100 )
```

Expected: a list of call sites that need updates.

- [ ] **Step 3: Update each caller in `libs/atlas-wz`.**

Apply the edits below verbatim. Where the existing call ignores the result entirely, propagate the error; where the prior code already had a "best-effort" semantic, add an explicit `// best-effort:` comment.

`libs/atlas-wz/charparts/smap.go:40` — change:

```go
return smapFromProps(smapImg.Properties()), nil
```

to:

```go
props, err := smapImg.Properties()
if err != nil {
	return nil, fmt.Errorf("smap properties: %w", err)
}
return smapFromProps(props), nil
```

Add `"fmt"` to imports if not present.

`libs/atlas-wz/charparts/extract.go:240` — change:

```go
props := img.Properties()
```

to:

```go
props, err := img.Properties()
if err != nil {
	return PartSet{}, false
}
```

(adjust return type to match the function's existing `(PartSet, bool)` signature — verify with `sed -n '230,250p' libs/atlas-wz/charparts/extract.go` before editing).

`libs/atlas-wz/mapimage/minimap.go:18` — change:

```go
cp := findMinimapCanvas(img.Properties())
```

to:

```go
props, err := img.Properties()
if err != nil {
	return nil, fmt.Errorf("minimap properties: %w", err)
}
cp := findMinimapCanvas(props)
```

Apply the same shape to `minimap.go:52`.

`libs/atlas-wz/mapimage/layers.go:49` (`ExtractLayout`) — change:

```go
root := img.Properties()
```

to:

```go
root, err := img.Properties()
if err != nil {
	return maplayout.Layout{}, fmt.Errorf("layers properties: %w", err)
}
```

Same shape at `layers.go:135` (`ExtractLayers`) but with the three-value return:

```go
root, err := img.Properties()
if err != nil {
	return nil, maplayout.Layout{}, fmt.Errorf("layers properties: %w", err)
}
```

`libs/atlas-wz/mapimage/decoder.go:80,116,137` — three sites. Each currently does `findSub(img.Properties(), ...)`. Refactor to:

```go
props, err := img.Properties()
if err != nil {
	return nil, fmt.Errorf("decoder properties: %w", err)
}
backSub := findSub(props, "back")
```

(matching the existing return shape; check each function's signature individually.)

`libs/atlas-wz/icons/extract.go:51,118,166`:

```go
props, err := img.Properties()
if err != nil {
	// best-effort: extract loop skips this image rather than aborting
	// the whole icon sweep — matches pre-F6 behavior where parse
	// failures were warned-and-dropped.
	l.WithError(err).Warnf("icons: skip image [%s]", img.Name())
	continue
}
```

(verify each call site is inside a loop that supports `continue` — three of them are.)

`libs/atlas-wz/icons/extract.go:194` — `linked.Properties()` inside the linked-image sweep:

```go
linkedProps, err := linked.Properties()
if err != nil {
	// best-effort: a linked image failing to parse falls back to the
	// outer image's properties (donor parity).
	continue
}
```

`libs/atlas-wz/wz/parse_race_test.go:81` — change:

```go
if got := img.Properties(); got != nil {
```

to:

```go
got, _ := img.Properties()
if got != nil {
```

The discard here is acceptable: the test specifically exercises the fast-path that returns the (nil) cached properties without parsing; there is no error to surface.

- [ ] **Step 4: Update callers in `services/atlas-data`.**

`services/atlas-data/atlas.com/data/data/workers/ui.go:44` — change:

```go
loginProps = img.Properties()
```

to:

```go
loginProps, err = img.Properties()
if err != nil {
	return nil, fmt.Errorf("ui worker: parse Login.img: %w", err)
}
```

(verify the enclosing function signature; if `err` already shadowed in scope, use `pErr`.)

`services/atlas-data/atlas.com/data/data/workers/item.go:77` — change:

```go
for _, prop := range img.Properties() {
```

to:

```go
props, err := img.Properties()
if err != nil {
	return fmt.Errorf("item worker: parse %s: %w", img.Name(), err)
}
for _, prop := range props {
```

`services/atlas-data/atlas.com/data/data/workers/skill.go:64` — change:

```go
skillDir := findSub(img.Properties(), "skill")
```

to:

```go
props, err := img.Properties()
if err != nil {
	return fmt.Errorf("skill worker: parse %s: %w", img.Name(), err)
}
skillDir := findSub(props, "skill")
```

`services/atlas-data/atlas.com/data/data/wztoxml/adapter.go:76` — change:

```go
root.Children = propertiesToElements(img.Properties())
```

to:

```go
props, err := img.Properties()
if err != nil {
	return nil, fmt.Errorf("wztoxml adapter: %s: %w", img.Name(), err)
}
root.Children = propertiesToElements(props)
```

- [ ] **Step 5: Verify libs/atlas-wz builds and tests pass.**

```bash
( cd libs/atlas-wz && go build ./... && go test -race ./... && go vet ./... )
```

Expected: all pass.

- [ ] **Step 6: Verify atlas-data builds and tests pass.**

```bash
( cd services/atlas-data/atlas.com/data && go build ./... && go test -race ./... && go vet ./... )
```

Expected: all pass.

- [ ] **Step 7: Verify atlas-renders and atlas-character-factory build (they import libs/atlas-wz transitively).**

```bash
( cd services/atlas-renders/atlas.com/renders && go build ./... )
( cd services/atlas-character-factory/atlas.com/character-factory && go build ./... )
```

Expected: clean builds. If either fails because of an additional `Properties()` call site not listed in step 3/4, apply the same propagation pattern and re-run.

- [ ] **Step 8: Docker bake (build impact check).**

`libs/atlas-wz/go.mod` is NOT modified here — only the public symbol changed — so per CLAUDE.md a bake isn't strictly mandatory. But because the symbol is consumed by services, run a smoke bake:

```bash
docker buildx bake atlas-data atlas-renders atlas-character-factory
```

Expected: all three targets build successfully.

- [ ] **Step 9: Commit.**

```bash
git add libs/atlas-wz/wz/image.go libs/atlas-wz/charparts/ libs/atlas-wz/mapimage/ libs/atlas-wz/icons/ libs/atlas-wz/wz/parse_race_test.go \
        services/atlas-data/atlas.com/data/data/
git commit -m "fix(atlas-wz): F6 surface parse errors from Image.Properties()"
```

---

### Task 9: F4 — Annotate wz seek-path concurrency invariants

**Files:**
- Modify: `services/atlas-data/atlas.com/data/data/workers/runtime.go:108-134`
- Modify: `libs/atlas-wz/wz/file.go:222-293` (`tryParseWithVersion`)
- Modify: `libs/atlas-wz/wz/image.go:107` (`parsePropertyList` doc)

- [ ] **Step 1: Add annotation to `fetchArchive`.**

Above the `func fetchArchive(...)` line in `services/atlas-data/atlas.com/data/data/workers/runtime.go:115` (or just after the existing comment block at 108–114), insert:

```go
// Concurrency: fetchArchive opens a fresh wz.File on a freshly-downloaded
// local path. The returned *wz.File is published to a single worker
// goroutine that holds it until the caller-provided cleanup runs. Other
// workers fetching the same archive name receive their own *wz.File on
// their own local copy (archiveSerialization memoizes the SERIALIZED form
// but not the *wz.File itself), so the Seek+Read sequence inside
// wz.Open's parseHeader/detectVersion/parseRoot runs single-threaded
// per File and needs no parseMu coverage. See task-076 F4 / CONCURRENCY-03.
```

- [ ] **Step 2: Add annotation to `tryParseWithVersion`.**

In `libs/atlas-wz/wz/file.go` above the `func (wz *File) tryParseWithVersion(...)` line (~222), insert:

```go
// Concurrency: tryParseWithVersion runs only during Open() (called from
// detectVersion) before the *File is published to any consumer goroutine.
// The Seek+Read against wz.reader is therefore single-threaded by
// construction and needs no parseMu coverage. Image.parse() acquires
// parseMu for all post-Open seek-based parsing; this is the canonical
// invariant — adding new public seek paths requires either parseMu
// coverage or an analogous single-threaded guarantee.
```

- [ ] **Step 3: Add invariant comment to `parsePropertyList`.**

In `libs/atlas-wz/wz/image.go:107` above `func (wz *File) parsePropertyList(...)`, replace the existing one-line doc with:

```go
// parsePropertyList reads a list of key-value property entries.
// imageOffset is the base offset for resolving offset-referenced strings
// within the image.
//
// Invariant: caller holds wz.parseMu. Entered via Image.parse() which
// acquires the lock unconditionally. Future contributors must not call
// this from outside that path without first acquiring the lock — the
// underlying wz.reader is shared across all Image instances backed by
// the same *File and is not safe to Seek concurrently.
```

- [ ] **Step 4: Run race tests to confirm no regressions.**

```bash
( cd libs/atlas-wz && go test -race ./... )
( cd services/atlas-data/atlas.com/data && go test -race ./data/workers/... )
```

Expected: all pass.

- [ ] **Step 5: Commit.**

```bash
git add services/atlas-data/atlas.com/data/data/workers/runtime.go \
        libs/atlas-wz/wz/file.go libs/atlas-wz/wz/image.go
git commit -m "fix(atlas-wz): F4 annotate wz seek-path concurrency invariants"
```

---

### Task 10: F15 — Extract shared helper for `ExtractLayout` / `ExtractLayers`

**Files:**
- Modify: `libs/atlas-wz/mapimage/layers.go`

- [ ] **Step 1: Inspect existing tests for `mapimage` so the refactor's no-output-drift is observable.**

```bash
ls libs/atlas-wz/mapimage/*_test.go
go test -count=1 -v ./libs/atlas-wz/mapimage/... 2>&1 | tail -30
```

If tests exist and pass currently, they form the regression net.

- [ ] **Step 2: Introduce a private `extractLayoutCommon` helper.**

In `libs/atlas-wz/mapimage/layers.go`, after the `LayerOutput` struct definition (~line 22), add the helper type and function:

```go
// layerSubInfo collects the per-layer parsing output shared between
// ExtractLayout and ExtractLayers so neither has to re-walk the property
// tree.
type layerSubInfo struct {
	ID         int
	Name       string
	Props      []property.Property
	Objs       []objEntry
	Tiles      []tileEntry
}

// extractLayoutCommon resolves bounds, walks foothold/portal/NPC/zmap
// subtrees, and produces the per-layer info slice for layers that have
// at least one tile or obj. Shared body for ExtractLayout (metadata-only
// emit) and ExtractLayers (composites pixels per layer).
func extractLayoutCommon(img *wz.Image) (maplayout.Layout, []layerSubInfo, error) {
	if img == nil {
		return maplayout.Layout{}, nil, fmt.Errorf("layout: nil image")
	}
	root, err := img.Properties()
	if err != nil {
		return maplayout.Layout{}, nil, fmt.Errorf("layout properties: %w", err)
	}
	info := childrenOf(root, "info")

	bounds, err := resolveBounds(info, root)
	if err != nil {
		return maplayout.Layout{}, nil, fmt.Errorf("resolve bounds: %w", err)
	}
	if bounds.W <= 0 || bounds.H <= 0 {
		return maplayout.Layout{}, nil, fmt.Errorf("invalid bounds %dx%d", bounds.W, bounds.H)
	}

	layout := maplayout.Layout{
		Version:   maplayout.SchemaVersion,
		MapID:     parseMapID(img.Name()),
		Bounds:    maplayout.Bounds{Left: bounds.X, Top: bounds.Y, Right: bounds.X + bounds.W, Bottom: bounds.Y + bounds.H},
		Footholds: extractFootholds(root),
		Portals:   extractPortals(root),
		NPCs:      extractNPCs(root),
	}

	subs := make([]layerSubInfo, 0, maxLayers)
	for layer := 0; layer < maxLayers; layer++ {
		layerSub := findSub(root, strconv.Itoa(layer))
		if layerSub == nil {
			continue
		}
		layerProps := layerSub.Children()
		objs := loadObjEntries(layerProps)
		layerInfo := childrenOf(layerProps, "info")
		tS := stringVal(layerInfo, "tS", "")
		tiles := loadTileEntries(layerProps, tS)
		if len(objs) == 0 && len(tiles) == 0 {
			continue
		}
		subs = append(subs, layerSubInfo{
			ID:    layer,
			Name:  fmt.Sprintf("layer-%d", layer),
			Props: layerProps,
			Objs:  objs,
			Tiles: tiles,
		})
	}
	return layout, subs, nil
}
```

Note: the helper does NOT include `ZMap` because `ExtractLayout` adds it but `ExtractLayers` does not. Callers add it back to the layout after the call.

- [ ] **Step 3: Rewrite `ExtractLayout` to use the helper.**

Replace `ExtractLayout` (`layers.go:45-94`) with:

```go
func ExtractLayout(img *wz.Image) (maplayout.Layout, error) {
	layout, subs, err := extractLayoutCommon(img)
	if err != nil {
		return maplayout.Layout{}, err
	}
	layout.ZMap = lookupZMap(img)
	layerMetas := make([]maplayout.Layer, 0, len(subs))
	for _, s := range subs {
		layerMetas = append(layerMetas, maplayout.Layer{
			ID:     s.ID,
			Name:   s.Name,
			Z:      s.ID,
			Source: s.Name,
		})
	}
	layout.Layers = layerMetas
	return layout, nil
}
```

- [ ] **Step 4: Rewrite `ExtractLayers` to use the helper.**

Replace `ExtractLayers` (`layers.go:131-197`) with:

```go
func ExtractLayers(idx *Index, img *wz.Image) ([]LayerOutput, maplayout.Layout, error) {
	layout, subs, err := extractLayoutCommon(img)
	if err != nil {
		return nil, maplayout.Layout{}, err
	}

	if idx == nil && img.File() != nil {
		idx = NewIndex(img.File())
	}
	// Re-derive bounds from layout for compositing.
	world := WorldBounds{
		X: layout.Bounds.Left,
		Y: layout.Bounds.Top,
		W: layout.Bounds.Right - layout.Bounds.Left,
		H: layout.Bounds.Bottom - layout.Bounds.Top,
	}

	outputs := make([]LayerOutput, 0, len(subs))
	layerMetas := make([]maplayout.Layer, 0, len(subs))
	for _, s := range subs {
		layerImg, err := compositeLayer(idx, world, s.Props, s.Objs, s.Tiles)
		if err != nil {
			return nil, maplayout.Layout{}, fmt.Errorf("layer %d: %w", s.ID, err)
		}
		outputs = append(outputs, LayerOutput{
			ID:    s.ID,
			Z:     s.ID,
			Image: layerImg,
			Name:  s.Name,
		})
		layerMetas = append(layerMetas, maplayout.Layer{
			ID:     s.ID,
			Name:   s.Name,
			Z:      s.ID,
			Source: s.Name,
		})
	}
	layout.Layers = layerMetas
	return outputs, layout, nil
}
```

Note the `WorldBounds` struct field names (`X/Y/W/H`) — verify against `libs/atlas-wz/mapimage/decoder.go` or wherever it's defined. Adjust if the field names differ.

- [ ] **Step 5: Run mapimage tests for byte-identical output.**

```bash
( cd libs/atlas-wz && go test -count=1 ./mapimage/... )
```

Expected: all existing tests pass (byte-identical layer/layout output).

- [ ] **Step 6: Verify race + vet + build.**

```bash
( cd libs/atlas-wz && go test -race ./... && go vet ./... && go build ./... )
```

Expected: all pass.

- [ ] **Step 7: Commit.**

```bash
git add libs/atlas-wz/mapimage/layers.go
git commit -m "refactor(atlas-wz): F15 extract layout-common helper for ExtractLayout/ExtractLayers"
```

---

### Task 11: F16 — `accessoryPartClassFor` delegates to `libs/atlas-constants`

**Files:**
- Modify: `libs/atlas-wz/go.mod`
- Modify: `libs/atlas-wz/go.sum` (auto)
- Modify: `libs/atlas-wz/charparts/extract.go:97-108`

- [ ] **Step 1: Add the dep to `libs/atlas-wz/go.mod`.**

```bash
( cd libs/atlas-wz && go get github.com/Chronicle20/atlas/libs/atlas-constants )
```

Or manually append to the `require` block in `libs/atlas-wz/go.mod`:

```go
require (
	github.com/Chronicle20/atlas/libs/atlas-constants v0.0.0-00010101000000-000000000000
	github.com/sirupsen/logrus v1.9.4
)
```

Since the workspace `go.work` already lists `./libs/atlas-constants`, the version is resolved via replace — no module fetch required.

- [ ] **Step 2: Replace the body of `accessoryPartClassFor`.**

In `libs/atlas-wz/charparts/extract.go`, replace lines 93–108:

```go
// accessoryPartClassFor classifies an Accessory subdirectory .img by its id
// range. v83 Character.wz/Accessory stores eye/face/earring accessories under
// the single dir; atlas-renders splits them via classifications 101xxxx
// (FaceAccessory), 102xxxx (EyeAccessory), 103xxxx (Earrings).
func accessoryPartClassFor(id uint32) (string, bool) {
	c := id / 10000
	switch c {
	case 101:
		return "FaceAccessory", true
	case 102:
		return "EyeAccessory", true
	case 103:
		return "Earrings", true
	}
	return "", false
}
```

with:

```go
// accessoryPartClassFor classifies an Accessory subdirectory .img by its id
// range. v83 Character.wz/Accessory stores eye/face/earring accessories under
// the single dir; atlas-renders splits them via the constants exported by
// libs/atlas-constants/item (DOM-21 single-source). See task-076 F16.
func accessoryPartClassFor(id uint32) (string, bool) {
	switch item.Classification(id / 10000) {
	case item.ClassificationFaceAccessory:
		return "FaceAccessory", true
	case item.ClassificationEyeAccessory:
		return "EyeAccessory", true
	case item.ClassificationEarring:
		return "Earrings", true
	}
	return "", false
}
```

Add the import to `libs/atlas-wz/charparts/extract.go`:

```go
import (
	// ... existing imports ...
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
)
```

- [ ] **Step 3: Run the existing test (extract_test.go:91-94).**

```bash
( cd libs/atlas-wz && go test -count=1 ./charparts -run TestAccessoryPartClassFor -v )
```

Expected: PASS (the test cases cover ids 1010000, 1020000, 1030000, etc. — all still map identically).

If the test isn't named exactly that, run the whole test file and verify the existing accessory-classification cases pass:

```bash
( cd libs/atlas-wz && go test -count=1 -v ./charparts )
```

- [ ] **Step 4: Verify libs/atlas-wz race + vet + build.**

```bash
( cd libs/atlas-wz && go test -race ./... && go vet ./... && go build ./... )
```

- [ ] **Step 5: Verify importer services build.**

```bash
( cd services/atlas-data/atlas.com/data && go build ./... )
( cd services/atlas-renders/atlas.com/renders && go build ./... )
( cd services/atlas-character-factory/atlas.com/character-factory && go build ./... )
```

- [ ] **Step 6: Docker bake (CLAUDE.md hard requirement — `go.mod` was touched).**

```bash
docker buildx bake atlas-data atlas-renders atlas-character-factory
```

Expected: all three targets build successfully. If a target fails because the shared Dockerfile is missing a `COPY libs/atlas-constants` line, fix it (CLAUDE.md: append one mod-only `COPY` and one source `COPY` in the root Dockerfile). Per CLAUDE.md, atlas-constants is already present in the root Dockerfile (verify with `grep atlas-constants Dockerfile`).

- [ ] **Step 7: Commit.**

```bash
git add libs/atlas-wz/go.mod libs/atlas-wz/go.sum libs/atlas-wz/charparts/extract.go
git commit -m "fix(atlas-wz): F16 delegate accessory classification to libs/atlas-constants"
```

---

### Task 12: F17 — Concurrent `Properties()` regression test

**Files:**
- Modify: `libs/atlas-wz/wz/parse_race_test.go`
- Create (conditional): `libs/atlas-wz/wz/testdata/concurrent.wz`

- [ ] **Step 1: Determine if a writer exists to generate a small fixture.**

```bash
grep -rn "func.*Write\|NewWriter" libs/atlas-wz/wz/ | head -10
```

If no writer is exposed, the fallback path is to write a `TestMain` that uses an existing committed fixture (look for any `.wz` file already in `libs/atlas-wz/wz/testdata/`) or skip-with-message when none exist:

```bash
ls libs/atlas-wz/wz/testdata/ 2>/dev/null || echo "no testdata yet"
find libs/atlas-wz -name "*.wz" 2>/dev/null
```

If neither a writer nor a usable existing fixture exists, the test must be authored against `NewParsedImage` (which doesn't hit the disk-Seek path the lock guards). That weakens the negative-control validation — flag it in the commit message.

- [ ] **Step 2: Write the concurrent test.**

Append to `libs/atlas-wz/wz/parse_race_test.go`:

```go
// TestPropertiesConcurrentParse exercises the parseMu invariant under load:
// 16 goroutines call Properties() against different *Image children of the
// same *wz.File. With parseMu in place this is race-free; without it the
// shared seek cursor corrupts cross-image reads and `go test -race` flags
// the goroutines.
//
// Fixture: requires a real WZ archive with ≥4 Image children at the root.
// If libs/atlas-wz/wz/testdata/concurrent.wz is absent the test
// t.Skip()s — the negative-control validation (run with parseMu removed
// to confirm the test would catch the regression) is documented in
// task-076 F17 and is operator-side, not CI-gated.
func TestPropertiesConcurrentParse(t *testing.T) {
	path := "testdata/concurrent.wz"
	if _, err := os.Stat(path); err != nil {
		t.Skipf("fixture missing (%s); see task-076 F17", path)
	}
	f, err := Open(logrus.New(), path)
	if err != nil {
		t.Fatalf("open fixture: %v", err)
	}
	defer f.Close()

	root := f.Root()
	if root == nil {
		t.Fatal("nil root")
	}
	imgs := root.Images()
	if len(imgs) < 4 {
		t.Fatalf("fixture has %d images, want >=4", len(imgs))
	}

	const goroutines = 16
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for g := 0; g < goroutines; g++ {
		go func(i int) {
			defer wg.Done()
			img := imgs[i%len(imgs)]
			if _, err := img.Properties(); err != nil {
				t.Errorf("Properties() error: %v", err)
			}
		}(g)
	}
	wg.Wait()
}
```

Add the imports `"os"` and `"github.com/sirupsen/logrus"` to the test file.

- [ ] **Step 3: Run under `-race`.**

```bash
( cd libs/atlas-wz && go test -race -count=1 ./wz -run TestPropertiesConcurrentParse -v )
```

Expected: PASS (if fixture present) or SKIP (if not). Either is acceptable — the fixture-presence path is the deliverable; the skip allows CI to remain green until the fixture is committed.

If the fixture can be committed (step 1 found one), the test executes for real. If not, document this in the commit message: the test is in place, awaiting the fixture as a follow-up.

- [ ] **Step 4: Negative-control validation (operator-side, off-CI).**

Manually verify the test would catch a parseMu regression. Temporarily comment out the lock acquisition in `Image.Properties()`:

```bash
# In libs/atlas-wz/wz/image.go, comment out:
#   unlock := i.wzFile.LockParse()
#   defer unlock()
```

Then re-run:

```bash
( cd libs/atlas-wz && go test -race -count=1 ./wz -run TestPropertiesConcurrentParse -v )
```

Expected: FAIL with race detector output. Revert the comment-out before committing.

- [ ] **Step 5: Commit.**

```bash
git add libs/atlas-wz/wz/parse_race_test.go
# Conditionally:
# git add libs/atlas-wz/wz/testdata/concurrent.wz
git commit -m "test(atlas-wz): F17 add concurrent Properties() regression test"
```

Mention in the commit body whether the fixture was committed or the test is currently a skip.

---

### Task 13: F14 — `wzinput/status.go` uses `server.MarshalResponse[Status]`

**Files:**
- Modify: `services/atlas-data/atlas.com/data/wzinput/status.go`

- [ ] **Step 1: Make `Status` satisfy the JSON:API model interface (if not already).**

Check if `Status` already implements `GetName()` and `GetID()`:

```bash
grep -n "GetName\|GetID" services/atlas-data/atlas.com/data/wzinput/status.go
```

If not, add:

```go
// GetName returns the JSON:API resource type. Matches the pre-F14 wire
// shape "wzInputStatus".
func (s Status) GetName() string { return "wzInputStatus" }

// GetID returns the JSON:API resource id. The status endpoint has a
// single per-scope singleton resource, "current".
func (s Status) GetID() string { return "current" }
```

If `MarshalResponse[T]` requires `SetID(string)` too, add:

```go
func (s *Status) SetID(string) error { return nil }
```

- [ ] **Step 2: Replace the manual envelope with `MarshalResponse`.**

In `services/atlas-data/atlas.com/data/wzinput/status.go`, replace lines 40–48 (`w.Header().Set` through the `json.NewEncoder` Encode block) with:

```go
				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				w.Header().Set("Content-Type", "application/vnd.api+json")
				server.MarshalResponse[Status](d.Logger())(w)(c.ServerInformation())(queryParams)(
					Status{FileCount: s.Count, TotalBytes: s.Size, UpdatedAt: s.UpdatedAt},
				)
```

Add imports:

```go
import (
	// ... existing ...
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/jtumidanski/api2go/jsonapi"
)
```

Drop the now-unused `"encoding/json"` import.

- [ ] **Step 3: Verify byte-for-byte wire shape unchanged.**

Manually inspect the rendered response shape:

```bash
( cd services/atlas-data/atlas.com/data && go test ./wzinput/... -v 2>&1 | head -30 )
```

If a `status_test.go` exists, it will catch any drift. If not, the audit-listed wire shape (data.type=wzInputStatus, data.id=current, attributes={fileCount,totalBytes,updatedAt}) is the contract — confirm `MarshalResponse[Status]` emits that shape.

- [ ] **Step 4: Module-wide verification.**

```bash
( cd services/atlas-data/atlas.com/data && go test -race ./wzinput && go vet ./wzinput && go build ./... )
```

- [ ] **Step 5: Commit.**

```bash
git add services/atlas-data/atlas.com/data/wzinput/status.go
git commit -m "fix(atlas-data): F14 use server.MarshalResponse in wz status handler"
```

---

### Task 14: F13 — Delete dead `processData` orphan

**Files:**
- Modify: `services/atlas-data/atlas.com/data/data/resource.go:22-59`

- [ ] **Step 1: Confirm `processData` is genuinely unreferenced.**

```bash
grep -rn "processData\b" services/atlas-data/ --include="*.go"
```

Expected: only the definition in `data/resource.go:31` and its accompanying comment.

- [ ] **Step 2: Delete the function and tighten the comment.**

In `services/atlas-data/atlas.com/data/data/resource.go`, delete lines 31–59 (the entire `func processData(...)` block).

In the same file, replace the comment block at lines 22–25:

```go
// POST /data/process is registered by runtime/rest.InitResource — that
// handler creates a k8s ingest Job. The legacy in-process processData
// (now-orphaned in this file) walked a PVC mount that no longer exists
// in the MinIO-backed model and would shadow the new handler.
```

with:

```go
// POST /data/process is registered by runtime/rest.InitResource — that
// handler creates a k8s ingest Job. The legacy in-process processData
// was removed in task-076 F13.
```

Drop the now-unused imports (`database`, `_map`, `document` may still be needed by other functions — only remove what `goimports`/`go build` flags).

- [ ] **Step 3: Verify build.**

```bash
( cd services/atlas-data/atlas.com/data && go build ./... && go vet ./data/... )
```

Expected: clean build. Any unused-import errors point at imports that became orphaned along with the function — remove them.

- [ ] **Step 4: Commit.**

```bash
git add services/atlas-data/atlas.com/data/data/resource.go
git commit -m "chore(atlas-data): F13 remove dead processData orphan"
```

---

### Task 15: F12 — Comment `wzinput` PATCH multipart bypass

**Files:**
- Modify: `services/atlas-data/atlas.com/data/wzinput/resource.go:15-24`

- [ ] **Step 1: Add an inline comment above the PATCH route registration.**

In `services/atlas-data/atlas.com/data/wzinput/resource.go`, above line 20 (`r.HandleFunc("/wz", rest.RegisterHandler(l)(si)("wz_upload", uploadHandler(mc)))...`), insert:

```go
// PATCH /data/wz streams the WZ multipart body directly to MinIO via
// uploadHandler. It deliberately uses rest.RegisterHandler (not
// RegisterInputHandler[T]) because the request body is binary multipart
// content, not a JSON:API envelope — there is no input model to decode.
// RegisterInputHandler[T] would consume the body as JSON and fail; the
// byte-stream path is the only correct shape for very large WZ uploads.
```

- [ ] **Step 2: Verify build.**

```bash
( cd services/atlas-data/atlas.com/data && go build ./wzinput/... && go vet ./wzinput/... )
```

- [ ] **Step 3: Commit.**

```bash
git add services/atlas-data/atlas.com/data/wzinput/resource.go
git commit -m "docs(atlas-data): F12 document wz PATCH multipart bypass"
```

---

## Operational One-Shots

### Task 16: F9 — Recreate-strategy cutover runbook

**Files:**
- Create: `docs/deploy/runbooks/recreate-strategy-cutover.md`

- [ ] **Step 1: Create the runbook.**

Write `docs/deploy/runbooks/recreate-strategy-cutover.md`:

```markdown
# Runbook: `RollingUpdate` → `Recreate` Deployment strategy cutover

## When to use

A Deployment that was previously deployed with `strategy.type=RollingUpdate`
and `strategy.rollingUpdate.{maxSurge,maxUnavailable}` needs to switch to
`strategy.type=Recreate`. Server-Side Apply (SSA) leaves the old
`rollingUpdate` keys attached to the resource as orphan-managed fields
even after the manifest drops them, which prevents the strategy change
from taking effect.

Confirmed bite on 2026-05-22 during atlas-data migration.

## Symptoms

- `kubectl describe deploy/atlas-<svc> | grep -A3 Strategy` shows
  `RollingUpdate` even after `kustomize build … | kubectl apply -f-`.
- Pods continue to roll one-at-a-time instead of being torn down before
  the next set comes up.

## Workaround

Strip the orphan fields with a JSON patch BEFORE re-applying:

\`\`\`bash
kubectl -n atlas-main patch deploy/atlas-data --type=json -p='[
  {"op":"remove","path":"/spec/strategy/rollingUpdate"}
]'
\`\`\`

Then re-apply the manifest:

\`\`\`bash
kustomize build deploy/k8s/overlays/main | kubectl apply -f -
\`\`\`

Verify:

\`\`\`bash
kubectl -n atlas-main describe deploy/atlas-data | grep -A3 Strategy
\`\`\`

Expected: `Strategy: Recreate`.

## Optional: prevent recurrence on first deploy elsewhere

For a future Deployment slated for `Recreate` from the start, add a
kustomize patch that explicitly nulls `/spec/strategy/rollingUpdate`:

\`\`\`yaml
patches:
  - target:
      kind: Deployment
      name: atlas-<svc>
    patch: |-
      - op: remove
        path: /spec/strategy/rollingUpdate
\`\`\`

Remove the patch after the first apply succeeds — keeping it around
forever costs nothing but adds noise.

## Notes

- atlas-main was unblocked on 2026-05-22 with the `kubectl patch` recipe.
- This issue only resurfaces on similar strategy migrations; routine
  Deployments are unaffected.
- Origin: SSA orphan-field semantics, not a kustomize bug.
```

- [ ] **Step 2: Commit.**

```bash
mkdir -p docs/deploy/runbooks
git add docs/deploy/runbooks/recreate-strategy-cutover.md
git commit -m "docs(ops): F9 add Recreate-strategy cutover runbook"
```

---

### Task 17: F10 — Stale `layer-*.png` cleanup runbook + execution

**Files:**
- Create: `docs/deploy/runbooks/clean-stale-layer-pngs.md`

- [ ] **Step 1: Write the runbook.**

Create `docs/deploy/runbooks/clean-stale-layer-pngs.md`:

```markdown
# Runbook: Remove stale `layer-*.png` files from MinIO

## When to use

Task-071 moved per-map layer composition to render-time (atlas-renders)
and stopped emitting `layer-N.png` files from atlas-data during ingest.
Pre-refactor uploads in any (tenant, region, version) tuple are now dead
weight in MinIO. Each cleanup is one-shot per env.

## What to clean

Under `atlas-assets`, every prefix matching:

\`\`\`
tenants/<tenantId>/regions/<region>/versions/<x.y>/map/<mapId>/layers/
\`\`\`

(Note: the shared scope prefix `shared/regions/.../layers/` is also fair
game if your env populated it from a pre-refactor ingest.)

## Procedure (per env)

\`\`\`bash
# Enumerate the (tenant, region, version) tuples currently restored.
mc alias set adm http://minio.minio.svc.cluster.local:9000 <accessKey> <secretKey>
mc find adm/atlas-assets --regex 'layers/$' --type d | head -20

# Dry-run.
mc find adm/atlas-assets --regex 'layers/' --type f | head -20

# Execute.
mc find adm/atlas-assets --regex 'layers/' --type f -exec 'mc rm {}'
\`\`\`

For atlas-main run the execute step inside a one-shot Job (image
`minio/mc:latest`) using the existing MinIO credentials secret.

## Verification

\`\`\`bash
mc find adm/atlas-assets --regex 'layers/' --type f | wc -l   # expect 0
\`\`\`

## Frequency

Run once per env after task-076 lands. New `layers/` prefixes should not
reappear — atlas-renders composites in-memory now.
```

- [ ] **Step 2: Execute against atlas-main (operator-side).**

This step is operator-driven, not automated by the plan. Coordinate with whoever owns atlas-main credentials, run the cleanup, and append the result to the runbook under a "Last executed" section if desired.

- [ ] **Step 3: Commit the runbook.**

```bash
git add docs/deploy/runbooks/clean-stale-layer-pngs.md
git commit -m "docs(ops): F10 add stale layer-png cleanup runbook"
```

---

### Task 18: F11 — Triage 359 "no-bounds" maps

**Files:**
- Create: `tools/triage-no-bounds.sh`
- Create: `docs/tasks/task-076-task071-followups/no-bounds-triage.json`

- [ ] **Step 1: Inspect what the Map worker logs on `extractLayoutErrs`.**

```bash
grep -rn "extractLayoutErrs\|no-bounds\|no_bounds" services/atlas-data/ | head -20
```

Find where the 359 count comes from. The likely shape: a per-ingest summary log includes `extractLayoutErrs=N` and the failing map IDs are logged per-map. Capture a recent ingest log (Loki or `kubectl logs`):

```bash
kubectl -n atlas-main logs -l app=atlas-data --tail=2000 | grep -E 'extractLayout|no-bounds' | tee /tmp/no-bounds.log
```

Extract the map IDs:

```bash
grep -oE 'map[ =][0-9]+' /tmp/no-bounds.log | grep -oE '[0-9]+' | sort -u > /tmp/no-bounds-ids.txt
wc -l /tmp/no-bounds-ids.txt   # expect ~359
```

- [ ] **Step 2: Write a triage script that cross-references against portal `tm` targets.**

Create `tools/triage-no-bounds.sh`:

```bash
#!/usr/bin/env bash
# Cross-references the no-bounds map ID list against the portal "tm"
# (target-map) field across all parsed maps. A no-bounds map ID that
# appears as ANY portal's tm is potentially user-visible.
#
# Inputs:
#   - /tmp/no-bounds-ids.txt — newline-separated map IDs from ingest logs.
#   - atlas-data running in a reachable env with the documents table
#     populated (POST-ingest).
#
# Output:
#   - docs/tasks/task-076-task071-followups/no-bounds-triage.json
set -euo pipefail

REPO_ROOT="$(git rev-parse --show-toplevel)"
IDS_FILE="${IDS_FILE:-/tmp/no-bounds-ids.txt}"
OUT="$REPO_ROOT/docs/tasks/task-076-task071-followups/no-bounds-triage.json"

if [[ ! -f "$IDS_FILE" ]]; then
  echo "missing $IDS_FILE — capture the no-bounds IDs from a recent ingest first" >&2
  exit 1
fi

# Dump every map's portal targets via the atlas-data API. Adjust the URL
# to a reachable env; default to localhost.
DATA_URL="${DATA_URL:-http://localhost:8080}"
TENANT_ID="${TENANT_ID:-ec876921-c363-4cc6-9c51-5bb8d57f9553}"
REGION="${REGION:-GMS}"
VER="${VER:-83.1}"

# Pull every map document, extract portal.tm fields. Schema:
# documents.content -> JSONB with .Layout.Portals[].Target.
curl -fsS \
  -H "TENANT_ID: $TENANT_ID" -H "REGION: $REGION" -H "MAJOR_VERSION: 83" -H "MINOR_VERSION: 1" \
  "$DATA_URL/api/data/maps" \
| jq -r '.data[].attributes.layout.portals[]?.target' \
| sort -un > /tmp/portal-targets.txt

# Reachable subset: no-bounds IDs that appear as some other map's portal target.
comm -12 \
  <(sort -u "$IDS_FILE") \
  <(sort -u /tmp/portal-targets.txt) > /tmp/reachable-ids.txt

# Unreachable subset: no-bounds IDs with no portal pointing to them.
comm -23 \
  <(sort -u "$IDS_FILE") \
  <(sort -u /tmp/portal-targets.txt) > /tmp/unreachable-ids.txt

jq -n \
  --argjson reachable "$(jq -R 'tonumber? // empty' /tmp/reachable-ids.txt | jq -s .)" \
  --argjson unreachable "$(jq -R 'tonumber? // empty' /tmp/unreachable-ids.txt | jq -s .)" \
  --arg generated "$(date -u +%FT%TZ)" \
  '{
    generatedAt: $generated,
    reachable: $reachable,
    unreachable: $unreachable,
    counts: { reachable: ($reachable|length), unreachable: ($unreachable|length), total: (($reachable|length) + ($unreachable|length)) }
  }' > "$OUT"

echo "wrote $OUT (reachable=$(wc -l < /tmp/reachable-ids.txt), unreachable=$(wc -l < /tmp/unreachable-ids.txt))"
```

Make executable: `chmod +x tools/triage-no-bounds.sh`.

- [ ] **Step 2b: Run it against a reachable env.**

```bash
bash tools/triage-no-bounds.sh
cat docs/tasks/task-076-task071-followups/no-bounds-triage.json | jq .counts
```

Expected: a JSON file with counts. If the env isn't reachable (or atlas-data doesn't expose `/api/data/maps`), capture the data via direct postgres query against the documents table instead and re-run.

- [ ] **Step 3: File a follow-up task per cluster of reachable map IDs (operator-side, off-CI).**

For any non-empty `reachable` array, open `docs/tasks/task-NNN-no-bounds-<cluster>/prd.md` skeletons (use `/spec-task` or the manual flow). This step doesn't block task-076's PR.

- [ ] **Step 4: Commit the triage artifacts.**

```bash
git add tools/triage-no-bounds.sh docs/tasks/task-076-task071-followups/no-bounds-triage.json
git commit -m "fix(atlas-data): F11 triage 359 no-bounds maps"
```

---

## Conditional

### Task 19: F19 — Add `atlas-renders` to `docker-compose.core.yml`

**Files:**
- Modify: `deploy/compose/docker-compose.core.yml`

- [ ] **Step 1: Locate the alphabetical insertion point.**

```bash
grep -n "^  atlas-" deploy/compose/docker-compose.core.yml | head -30
```

`atlas-renders` slots between `atlas-reactors` and `atlas-saga-orchestrator` (alphabetical). Find the exact line numbers.

- [ ] **Step 2: Inspect the k8s manifest for atlas-renders to mirror env/ports.**

```bash
sed -n '1,80p' deploy/k8s/base/atlas-renders.yaml
```

Note env vars (LOG_LEVEL, REST_PORT, MINIO_*, …) and any volume mounts.

- [ ] **Step 3: Insert the service block.**

After the `atlas-reactors:` block in `deploy/compose/docker-compose.core.yml`, before `atlas-saga-orchestrator:`, insert (adjust env to match the k8s manifest):

```yaml
  atlas-renders:
    <<: *atlas-defaults
    container_name: atlas-renders
    build:
      context: ../..
      dockerfile: Dockerfile
      args:
        SERVICE: atlas-renders
    image: atlas-renders:${ATLAS_IMAGE_TAG:-local}
    environment:
      LOG_LEVEL: debug
      REST_PORT: 8080
      MINIO_ENDPOINT: minio:9000
      MINIO_ACCESS_KEY: minioadmin
      MINIO_SECRET_KEY: minioadmin
      MINIO_USE_SSL: "false"
      BUCKET_ASSETS: atlas-assets
    volumes:
      - ../../tmp/wz-scratch:/scratch/wz
    # Internal-only — no `ports:` mapping. PRD §8 Security.
```

- [ ] **Step 4: Verify locally.**

```bash
( cd deploy/compose && docker compose -f docker-compose.yml -f docker-compose.core.yml build atlas-renders )
( cd deploy/compose && docker compose -f docker-compose.yml -f docker-compose.core.yml up -d atlas-renders )
docker logs atlas-renders --tail 30
( cd deploy/compose && docker compose -f docker-compose.yml -f docker-compose.core.yml down )
```

Expected: atlas-renders starts cleanly and logs `REST server listening on :8080` (or whatever the canonical startup log is).

- [ ] **Step 5: Commit.**

```bash
git add deploy/compose/docker-compose.core.yml
git commit -m "feat(deploy): F19 add atlas-renders to docker-compose.core.yml"
```

---

### Task 20: F20 — Diagnose-then-fix Henesys portal duplication

**Files:**
- Modify (conditional): `libs/atlas-wz/mapimage/layers.go:294-317` (`extractPortals`) OR `services/atlas-portals/...`
- Create: `libs/atlas-wz/mapimage/layers_portal_test.go` (regression)
- Create: `docs/tasks/task-076-task071-followups/repro/f20-henesys-portals.md` (diagnosis log)

- [ ] **Step 1: Probe the read-side first per OQ-6.**

```bash
# atlas-portals returns the portal list for a map.
curl -fsS \
  -H 'TENANT_ID: ec876921-c363-4cc6-9c51-5bb8d57f9553' -H 'REGION: GMS' \
  -H 'MAJOR_VERSION: 83' -H 'MINOR_VERSION: 1' \
  http://localhost:80/api/portals?map=100000000 | jq . | tee /tmp/henesys-portals-read.json
```

If atlas-portals is not exposed on the test env, swap to the atlas-data documents endpoint:

```bash
curl -fsS \
  -H 'TENANT_ID: ec876921-c363-4cc6-9c51-5bb8d57f9553' -H 'REGION: GMS' \
  -H 'MAJOR_VERSION: 83' -H 'MINOR_VERSION: 1' \
  http://localhost:8080/api/data/maps/100000000 | jq '.data.attributes.layout.portals' | tee /tmp/henesys-portals-data.json
```

Document the observed duplicate entries in `docs/tasks/task-076-task071-followups/repro/f20-henesys-portals.md`. Include sample JSON snippets.

- [ ] **Step 2: Pinpoint the root cause.**

If the data-side JSON already contains duplicates, the bug is in `extractPortals` (`libs/atlas-wz/mapimage/layers.go:294-317`) which iterates portal subtrees with no dedup. Verify by inspecting the parsed WZ tree for Henesys 100000000 — confirm WZ data itself has duplicate or shadow entries.

If the data-side JSON is clean but atlas-portals' output is not, the bug is in atlas-portals (read-side amplification). Walk back to that service.

Record the conclusion in the diagnosis log.

- [ ] **Step 3 (if fix is bounded — extraction-side):** Dedup in `extractPortals`.

Modify `libs/atlas-wz/mapimage/layers.go:295-317`:

```go
// extractPortals walks portal/<i>/ entries into a flat list. Dedup by
// (name, target, x, y) — v83 WZ data for some maps (Henesys 100000000)
// includes shadow entries (pn="", pn="0") that collide with player-
// visible portals on the same coordinate. Without dedup these surface as
// duplicate entries in atlas-portals' response. See task-076 F20.
func extractPortals(root []property.Property) []maplayout.Portal {
	portal := findSub(root, "portal")
	if portal == nil {
		return nil
	}
	seen := make(map[string]struct{})
	var out []maplayout.Portal
	for _, p := range portal.Children() {
		sub, ok := p.(*property.SubProperty)
		if !ok {
			continue
		}
		ch := sub.Children()
		target := uint32(intVal(ch, "tm", 0))
		entry := maplayout.Portal{
			Name:   stringVal(ch, "pn", ""),
			Type:   intVal(ch, "pt", 0),
			Target: target,
			X:      intVal(ch, "x", 0),
			Y:      intVal(ch, "y", 0),
		}
		key := fmt.Sprintf("%s|%d|%d|%d", entry.Name, entry.Target, entry.X, entry.Y)
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, entry)
	}
	return out
}
```

- [ ] **Step 4: Add the regression test.**

Create `libs/atlas-wz/mapimage/layers_portal_test.go`:

```go
package mapimage

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-wz/wz/property"
)

// TestExtractPortalsDeduplicates pins the F20 fix: two portal entries
// with the same (name, target, x, y) collapse to one. Mirror the WZ
// shape observed for Henesys 100000000 in the f20 repro log.
func TestExtractPortalsDeduplicates(t *testing.T) {
	mkPortal := func(name string, tm int32) property.Property {
		return property.NewSub("0", []property.Property{
			property.NewString("pn", name),
			property.NewInt("tm", tm),
			property.NewInt("pt", 2),
			property.NewInt("x", 100),
			property.NewInt("y", 200),
		})
	}
	portalSub := property.NewSub("portal", []property.Property{
		mkPortal("sp", 999999999),
		mkPortal("sp", 999999999), // duplicate
		mkPortal("east00", 100000001),
	})
	root := []property.Property{portalSub}

	out := extractPortals(root)
	if len(out) != 2 {
		t.Fatalf("got %d portals, want 2 (dedup), entries: %+v", len(out), out)
	}
}
```

(Verify the `property.NewSub/NewString/NewInt` constructor names against `libs/atlas-wz/wz/property/`; adjust if they differ.)

- [ ] **Step 5: Run the test and module verification.**

```bash
( cd libs/atlas-wz && go test -race ./mapimage -run TestExtractPortalsDeduplicates -v )
( cd libs/atlas-wz && go test -race ./... && go vet ./... && go build ./... )
```

Expected: PASS.

- [ ] **Step 6 (structural escape hatch):** If diagnosis exposes a structural bug.

If step 2 concluded the fix requires a schema change (e.g., portal model needs a `kind` field to disambiguate visible vs. internal portals), STOP. Commit only the diagnosis log + a placeholder test (skipped with a TODO referencing the new follow-up task ID). Open `task-NNN-portal-dedupe-schema` for the structural fix.

- [ ] **Step 7: Commit.**

If bounded fix:

```bash
git add libs/atlas-wz/mapimage/layers.go \
        libs/atlas-wz/mapimage/layers_portal_test.go \
        docs/tasks/task-076-task071-followups/repro/f20-henesys-portals.md
git commit -m "fix(atlas-wz): F20 dedup portals in extractPortals"
```

If diagnosis-only:

```bash
git add docs/tasks/task-076-task071-followups/repro/f20-henesys-portals.md \
        libs/atlas-wz/mapimage/layers_portal_test.go
git commit -m "fix(atlas-wz): F20 diagnose Henesys portal duplication (fix deferred)"
```

---

## Final Verification

### Task 21: Branch-level acceptance

- [ ] **Step 1: Run all module gates from worktree root.**

For each module touched (atlas-data, atlas-renders, atlas-character-factory, atlas-wz, atlas-constants):

```bash
for mod in services/atlas-data/atlas.com/data services/atlas-renders/atlas.com/renders services/atlas-character-factory/atlas.com/character-factory libs/atlas-wz libs/atlas-constants; do
  echo "=== $mod ==="
  ( cd "$mod" && go test -race ./... && go vet ./... && go build ./... ) || { echo "FAILED in $mod" >&2; exit 1; }
done
```

Expected: all green.

- [ ] **Step 2: Docker bake every service whose `go.mod` was touched.**

`libs/atlas-wz/go.mod` was modified in Task 11 (F16). Bake every importer:

```bash
docker buildx bake atlas-data atlas-renders atlas-character-factory
```

Expected: all three targets build successfully. Per CLAUDE.md this is mandatory — do not skip.

- [ ] **Step 3: Run the routes test.**

```bash
bash deploy/shared/test/routes_nginxt.sh
```

Expected: all four checks pass (nginx -t, MinIO cross-ns, atlas-renders headers, F18 drift).

- [ ] **Step 4: Kustomize sanity checks.**

```bash
kustomize build deploy/k8s/base > /dev/null
kustomize build deploy/k8s/overlays/main > /dev/null
```

Expected: clean output.

- [ ] **Step 5: Confirm git status is clean and branch is correct.**

```bash
git status
git rev-parse --show-toplevel
git branch --show-current
```

Expected:
- `git status` → working tree clean.
- worktree path ends with `/.worktrees/task-076-task071-followups`.
- branch is `task-076-task071-followups`.

- [ ] **Step 6: Code review BEFORE opening the PR (CLAUDE.md hard rule).**

```bash
# Invoke superpowers:requesting-code-review (parallel: plan-adherence,
# backend-guidelines, optionally frontend-guidelines if any TS changed —
# this task touches none, so backend-only).
```

The reviewer dispatches:
- `plan-adherence-reviewer` (verifies each Task above mapped to actual commits).
- `backend-guidelines-reviewer` (DOM-* checklist — F16 specifically must show DOM-21 closure).

Each agent writes findings to `docs/tasks/task-076-task071-followups/audit.md`. Address red flags before opening the PR.

- [ ] **Step 7: PR description.**

Open the PR via `gh pr create` with a body that lists every followup (F1–F20) with a checked/unchecked box and a one-line outcome. The body MUST link to `../../task-071-gamedata-minio-consolidation/docs/tasks/task-071-gamedata-minio-consolidation/followups.md`.

```bash
gh pr create --title "task-076: close task-071 followups (F1-F20)" --body "$(cat <<'EOF'
## Summary

Closes task-071's followups inventory across two waves.

### Wave 1 (production hot path)
- [x] F1 — publish 500: buffer tar to tempfile, surface step errors
- [x] F7 — pin atlas-renders in main overlay
- [x] F3 — stop caching negative scope verdicts
- [x] F2 — Commodity worker drops outer transaction
- [x] F5 — two-phase finalize baseline restore

### Wave 2 (debt)
- [x] F8 — routes-config dedupe via kustomize configMapGenerator
- [x] F18 — routes drift validation in CI
- [x] F6 — Properties() returns ([]Property, error)
- [x] F4 — wz seek-path concurrency annotations
- [x] F15 — extract layout-common helper
- [x] F16 — accessory classification → libs/atlas-constants
- [x] F17 — concurrent Properties() regression test
- [x] F14 — wzinput status.go MarshalResponse
- [x] F13 — delete dead processData orphan
- [x] F12 — document wz PATCH multipart bypass

### Operational one-shots
- [x] F9 — Recreate cutover runbook
- [x] F10 — stale layer-png cleanup runbook
- [x] F11 — 359 no-bounds maps triage

### Conditional
- [ ] F19 — atlas-renders in docker-compose.core.yml (status filled per execution)
- [ ] F20 — Henesys portal duplication (status filled per execution)

Source: ../../task-071-gamedata-minio-consolidation/docs/tasks/task-071-gamedata-minio-consolidation/followups.md

## Test plan
- [x] go test -race ./... clean per changed module
- [x] go vet ./... clean
- [x] go build ./... clean
- [x] docker buildx bake atlas-data atlas-renders atlas-character-factory (F16 go.mod touch)
- [x] deploy/shared/test/routes_nginxt.sh passes (F18)
- [x] kustomize build deploy/k8s/overlays/main resolves

🤖 Generated with [Claude Code](https://claude.com/claude-code)
EOF
)"
```

(Update the conditional checkboxes for F19/F20 to reflect whether they landed or were spun out.)

---

## Notes on Wave/Task Ordering

- **Wave 1 (Tasks 1–5)** MUST land before Wave 2 in review priority. Each can land in any internal order — but F7 (Task 2) is the simplest and should land first as a smoke-check that the worktree is healthy.
- **F6 (Task 8) is the highest-merge-conflict risk in Wave 2** because it touches 18 call sites across the monorepo. Land it as early in Wave 2 as possible and rebase quickly.
- **F8 / F18 (Tasks 6, 7) form a pair**: F18 depends on F8's generator existing. Land them adjacent.
- **F4 / F15 / F16 / F17 (Tasks 9–12)** are independent and can be reviewed in any internal order.
- **F12 / F13 / F14 (Tasks 13–15)** are pure hygiene; they can land any time but conventionally last in Wave 2.
- **Operational one-shots (Tasks 16–18)** are documentation + tooling; F10's actual `mc rm` execution is operator-side and not gated by the PR.
- **F19 / F20 (Tasks 19–20)** are conditional. If F20's diagnosis exposes a structural bug, only the diagnosis lands here per PRD §2 non-goal and per design.md §7 R4.

## What's NOT in this plan

Per PRD §2 non-goals and design.md §8:
- No new Kafka topics, REST endpoints, or shared libs.
- No retrofit of task-071 docs.
- No expansion of the followups inventory.
- F11 user-visible-map fixes are spun out as separate tasks if discovered.
- F20 structural fix (if diagnosis warrants) is spun out as a separate task.
