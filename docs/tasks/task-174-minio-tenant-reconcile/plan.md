# MinIO Tenant-Prefix Reconciliation + Teardown Hardening — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a daily reconciliation backstop (CronJob orchestrator + operator-gated atlas-data executor endpoint) that reclaims orphaned `tenants/<uuid>/` prefixes from the shared MinIO, plus harden the PreDelete purge hook so leaks are rarer.

**Architecture:** A CronJob in `atlas-main` enumerates the live-tenant UUID union across `atlas-main` + `atlas-pr-*` namespaces (fail-closed) and POSTs it as a keep-list to a new `POST /api/data/minio/reconcile` endpoint. atlas-data — which owns the MinIO client — lists `tenants/<uuid>/` prefixes in the three buckets and deletes those not in the keep-list whose newest object is older than 48 h, refusing an empty keep-list and never touching the canonical sentinel. Deletion is disabled (dry-run) until a ConfigMap flag is flipped.

**Tech Stack:** Go (atlas-data service, `minio-go/v7`), api2go JSON:API, gorilla/mux, POSIX shell + bats (orchestrator + hook), Kustomize/k8s (CronJob + RBAC + ConfigMap).

## Global Constraints

- Operator gate on the executor: `X-Atlas-Operator: 1` header required, else 403 (matches `tenantpurge`/`baseline`).
- Object key scheme is fixed: per-tenant objects live at `tenants/<uuid>/...` in buckets `atlas-wz`, `atlas-assets`, `atlas-renders`. Do not change it.
- Three non-negotiable safety gates in the executor: **(1)** empty keep-list ⇒ refuse (422), **(2)** skip the canonical sentinel `canonical.TenantUUID` (`00000000-0000-0000-0000-000000000000`), **(3)** age guard — only prefixes whose newest object is `>= minAgeHours` old (default 48).
- Reconcile is **MinIO-only** — no DB access (orphan Postgres rows died with their PR namespaces).
- Injected clock: `Reconcile` takes `now time.Time` as a parameter; never call `time.Now()` inside the core so the 48 h boundary is deterministic in tests.
- Fail-closed orchestrator: if any discovered namespace's `/api/tenants` cannot be fetched/parsed, abort without POSTing (a partial keep-list must never delete a live env's data).
- Verification gates (per CLAUDE.md), in the atlas-data module and repo root: `go test -race ./...`, `go vet ./...`, `go build ./...`; `docker buildx bake atlas-data` and `atlas-pr-bootstrap` if their build inputs change; `tools/redis-key-guard.sh`, `tools/goroutine-guard.sh`, `tools/service-registration-guard.sh` clean.
- Shell: portable POSIX/bash reusing `services/atlas-pr-bootstrap/scripts/lib.sh` (`log`, `require_env`, `retry`, `record_error`, `run_phase`, `summarize_phases`).

**Worktree:** all work happens in the task worktree `.worktrees/task-174-minio-tenant-reconcile/` on branch `task-174-minio-tenant-reconcile`. In the shell blocks below, `<worktree-root>` denotes that worktree's absolute path; `cd <worktree-root>` before the repo-relative commands. Verify `git branch --show-current` after each commit.

---

## File Structure

**atlas-data (`services/atlas-data/atlas.com/data/minioreconcile/`):**
- `reconcile.go` — `Store` interface, `Request`/`Report`/`PrefixInfo` types, `Reconcile(ctx,l,store,req,now)` core. No minio-go import.
- `store.go` — `minioStore` adapter wrapping `*minio.Client` (satisfies `Store`); `ListTenantIDs` via delimiter listing.
- `rest.go` — JSON:API `ReconcileInputModel` / `ReconcileOutputModel`.
- `handler.go` — `InitResource(mc)`, `POST /data/minio/reconcile`, gates.
- `reconcile_test.go` — table tests over a fake `Store` + injected clock.
- `handler_test.go` — httptest for 403/422/nil-mc gates.

**atlas-data wiring:**
- `main.go:174` — add `AddRouteInitializer(minioreconcile.InitResource(mc)(GetServer())).`

**Orchestrator + hook (`services/atlas-pr-bootstrap/`):**
- `scripts/reconcile-minio.sh` — CronJob entrypoint (create).
- `scripts/predelete-purge.sh` — add retry/backoff (modify).
- `test/reconcile_minio_test.bats` — orchestrator bats (create).
- `test/predelete_test.bats` — extend for retry behavior (modify).

**K8s (`deploy/k8s/`):**
- `base/atlas-minio-reconcile.yaml` — CronJob + ServiceAccount + ClusterRole + ClusterRoleBinding + ConfigMap (create).
- `base/kustomization.yaml` — register the new file (modify).
- overlays `main`/`pr` — patch schedule/dry-run as needed (modify only if the base default is wrong for an overlay; see Task 5).

---

## Task 1: Reconcile core (Store interface + all safety gates)

Heart of the executor. Pure logic over an injected `Store` and clock — fully unit-tested with a fake. TDD.

**Files:**
- Create: `services/atlas-data/atlas.com/data/minioreconcile/reconcile.go`
- Test: `services/atlas-data/atlas.com/data/minioreconcile/reconcile_test.go`

**Interfaces:**
- Produces:
  - `type Store interface { ListTenantIDs(ctx context.Context, bucket string) ([]string, error); PrefixInfo(ctx context.Context, bucket, prefix string) (PrefixInfo, error); RemovePrefix(ctx context.Context, bucket, prefix string) error; Buckets() []string }`
  - `type PrefixInfo struct { Count int; Bytes int64; Newest time.Time }`
  - `type Request struct { KeepTenantIDs []string; MinAgeHours int; DryRun bool }`
  - `type Report struct { DryRun bool; MinAgeHours int; TotalPrefixes int; TotalBytes int64; Rows []ReportRow }`
  - `type ReportRow struct { Bucket, TenantID, Action string; Count int; Bytes int64; Newest time.Time }` — `Action ∈ {"deleted","would-delete","kept-too-new"}`
  - `var ErrEmptyKeepList = errors.New("keep-list is empty; refusing to reconcile")`
  - `func Reconcile(ctx context.Context, l logrus.FieldLogger, store Store, req Request, now time.Time) (Report, error)`

- [ ] **Step 1: Write the failing tests**

```go
package minioreconcile

import (
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

// fakeStore is an in-memory Store. prefixes[bucket][tenantID] = PrefixInfo.
type fakeStore struct {
	buckets  []string
	prefixes map[string]map[string]PrefixInfo
	removed  []string // "bucket:tenants/<id>/"
}

func (f *fakeStore) Buckets() []string { return f.buckets }
func (f *fakeStore) ListTenantIDs(_ context.Context, bucket string) ([]string, error) {
	ids := make([]string, 0)
	for id := range f.prefixes[bucket] {
		ids = append(ids, id)
	}
	return ids, nil
}
func (f *fakeStore) PrefixInfo(_ context.Context, bucket, prefix string) (PrefixInfo, error) {
	id := prefix[len("tenants/") : len(prefix)-1] // strip "tenants/" and trailing "/"
	return f.prefixes[bucket][id], nil
}
func (f *fakeStore) RemovePrefix(_ context.Context, bucket, prefix string) error {
	f.removed = append(f.removed, bucket+":"+prefix)
	return nil
}

const canonicalUUID = "00000000-0000-0000-0000-000000000000"

func now() time.Time { return time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC) }

// old returns a timestamp h hours before now().
func old(h int) time.Time { return now().Add(-time.Duration(h) * time.Hour) }

func storeWith(rows map[string]PrefixInfo) *fakeStore {
	return &fakeStore{
		buckets:  []string{"atlas-wz"},
		prefixes: map[string]map[string]PrefixInfo{"atlas-wz": rows},
	}
}

func TestReconcile_EmptyKeepListRefused(t *testing.T) {
	s := storeWith(map[string]PrefixInfo{"aaaa": {Count: 1, Bytes: 10, Newest: old(100)}})
	_, err := Reconcile(context.Background(), logrus.New(), s, Request{KeepTenantIDs: nil, MinAgeHours: 48}, now())
	if err != ErrEmptyKeepList {
		t.Fatalf("want ErrEmptyKeepList, got %v", err)
	}
	if len(s.removed) != 0 {
		t.Fatalf("nothing must be removed on refusal, got %v", s.removed)
	}
}

func TestReconcile_KeepListPreserved(t *testing.T) {
	s := storeWith(map[string]PrefixInfo{
		"keep-me": {Count: 1, Bytes: 10, Newest: old(100)},
		"orphan":  {Count: 2, Bytes: 20, Newest: old(100)},
	})
	rep, err := Reconcile(context.Background(), logrus.New(), s, Request{KeepTenantIDs: []string{"keep-me"}, MinAgeHours: 48, DryRun: false}, now())
	if err != nil {
		t.Fatal(err)
	}
	if len(s.removed) != 1 || s.removed[0] != "atlas-wz:tenants/orphan/" {
		t.Fatalf("only orphan should be removed, got %v", s.removed)
	}
	if rep.TotalPrefixes != 1 || rep.TotalBytes != 20 {
		t.Fatalf("report totals wrong: %+v", rep)
	}
}

func TestReconcile_CanonicalExcluded(t *testing.T) {
	s := storeWith(map[string]PrefixInfo{canonicalUUID: {Count: 1, Bytes: 10, Newest: old(100)}})
	_, err := Reconcile(context.Background(), logrus.New(), s, Request{KeepTenantIDs: []string{"someone"}, MinAgeHours: 48}, now())
	if err != nil {
		t.Fatal(err)
	}
	if len(s.removed) != 0 {
		t.Fatalf("canonical sentinel must never be removed, got %v", s.removed)
	}
}

func TestReconcile_AgeGuardBoundary(t *testing.T) {
	s := storeWith(map[string]PrefixInfo{
		"too-new": {Count: 1, Bytes: 10, Newest: old(47)}, // 47h < 48h → kept
		"old":     {Count: 1, Bytes: 10, Newest: old(49)}, // 49h ≥ 48h → eligible
	})
	rep, err := Reconcile(context.Background(), logrus.New(), s, Request{KeepTenantIDs: []string{"keep"}, MinAgeHours: 48, DryRun: false}, now())
	if err != nil {
		t.Fatal(err)
	}
	if len(s.removed) != 1 || s.removed[0] != "atlas-wz:tenants/old/" {
		t.Fatalf("only the >48h prefix should be removed, got %v", s.removed)
	}
	// too-new is reported as kept-too-new, not deleted
	var keptTooNew int
	for _, row := range rep.Rows {
		if row.Action == "kept-too-new" {
			keptTooNew++
		}
	}
	if keptTooNew != 1 {
		t.Fatalf("want 1 kept-too-new row, got %d (%+v)", keptTooNew, rep.Rows)
	}
}

func TestReconcile_DryRunDeletesNothing(t *testing.T) {
	s := storeWith(map[string]PrefixInfo{"orphan": {Count: 1, Bytes: 10, Newest: old(100)}})
	rep, err := Reconcile(context.Background(), logrus.New(), s, Request{KeepTenantIDs: []string{"keep"}, MinAgeHours: 48, DryRun: true}, now())
	if err != nil {
		t.Fatal(err)
	}
	if len(s.removed) != 0 {
		t.Fatalf("dryRun must not remove, got %v", s.removed)
	}
	if rep.TotalPrefixes != 1 || rep.Rows[0].Action != "would-delete" {
		t.Fatalf("dryRun should report would-delete: %+v", rep)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd services/atlas-data/atlas.com/data && go test ./minioreconcile/ -run TestReconcile -v`
Expected: FAIL — `undefined: Reconcile` / `undefined: Store` (package has no non-test source yet).

- [ ] **Step 3: Write `reconcile.go`**

```go
package minioreconcile

import (
	"context"
	"errors"
	"time"

	"atlas-data/canonical"

	"github.com/sirupsen/logrus"
)

// ErrEmptyKeepList is returned when the request carries no tenant ids. An empty
// keep-list must never be interpreted as "delete everything".
var ErrEmptyKeepList = errors.New("keep-list is empty; refusing to reconcile")

// PrefixInfo is the aggregate for one tenants/<uuid>/ prefix.
type PrefixInfo struct {
	Count  int
	Bytes  int64
	Newest time.Time
}

// Store is the narrow MinIO surface Reconcile needs. Implemented by minioStore
// (store.go) in production and by a fake in tests.
type Store interface {
	ListTenantIDs(ctx context.Context, bucket string) ([]string, error)
	PrefixInfo(ctx context.Context, bucket, prefix string) (PrefixInfo, error)
	RemovePrefix(ctx context.Context, bucket, prefix string) error
	Buckets() []string
}

// Request is the reconcile input.
type Request struct {
	KeepTenantIDs []string
	MinAgeHours   int
	DryRun        bool
}

// ReportRow is one prefix's outcome. Action ∈ {"deleted","would-delete","kept-too-new"}.
type ReportRow struct {
	Bucket   string
	TenantID string
	Action   string
	Count    int
	Bytes    int64
	Newest   time.Time
}

// Report aggregates the sweep result. Totals count only eligible prefixes
// (deleted or would-delete), not kept-too-new.
type Report struct {
	DryRun        bool
	MinAgeHours   int
	TotalPrefixes int
	TotalBytes    int64
	Rows          []ReportRow
}

// Reconcile removes tenants/<uuid>/ prefixes not in req.KeepTenantIDs whose
// newest object is at least req.MinAgeHours old, across every bucket the store
// reports. It refuses an empty keep-list and never touches the canonical
// sentinel. `now` is injected for deterministic age tests.
func Reconcile(ctx context.Context, l logrus.FieldLogger, store Store, req Request, now time.Time) (Report, error) {
	if len(req.KeepTenantIDs) == 0 {
		return Report{}, ErrEmptyKeepList
	}
	keep := make(map[string]struct{}, len(req.KeepTenantIDs))
	for _, id := range req.KeepTenantIDs {
		keep[id] = struct{}{}
	}
	minAge := time.Duration(req.MinAgeHours) * time.Hour
	rep := Report{DryRun: req.DryRun, MinAgeHours: req.MinAgeHours}

	for _, bucket := range store.Buckets() {
		ids, err := store.ListTenantIDs(ctx, bucket)
		if err != nil {
			return Report{}, err
		}
		for _, id := range ids {
			if _, ok := keep[id]; ok {
				continue
			}
			if id == canonical.TenantUUID {
				continue // canonical sentinel is never per-tenant data; never delete
			}
			prefix := "tenants/" + id + "/"
			info, err := store.PrefixInfo(ctx, bucket, prefix)
			if err != nil {
				return Report{}, err
			}
			if info.Count == 0 {
				continue
			}
			if now.Sub(info.Newest) < minAge {
				rep.Rows = append(rep.Rows, ReportRow{Bucket: bucket, TenantID: id, Action: "kept-too-new", Count: info.Count, Bytes: info.Bytes, Newest: info.Newest})
				continue
			}
			action := "would-delete"
			if !req.DryRun {
				if err := store.RemovePrefix(ctx, bucket, prefix); err != nil {
					l.WithError(err).Warnf("reconcile: failed to remove %s/%s", bucket, prefix)
					return Report{}, err
				}
				action = "deleted"
			}
			l.Infof("reconcile: %s %s/%s (%d objects, %d bytes, newest %s)", action, bucket, prefix, info.Count, info.Bytes, info.Newest.UTC().Format(time.RFC3339))
			rep.Rows = append(rep.Rows, ReportRow{Bucket: bucket, TenantID: id, Action: action, Count: info.Count, Bytes: info.Bytes, Newest: info.Newest})
			rep.TotalPrefixes++
			rep.TotalBytes += info.Bytes
		}
	}
	return rep, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd services/atlas-data/atlas.com/data && go test ./minioreconcile/ -run TestReconcile -v`
Expected: PASS (all 5 tests).

- [ ] **Step 5: Commit**

```bash
cd <worktree-root>
git add services/atlas-data/atlas.com/data/minioreconcile/reconcile.go services/atlas-data/atlas.com/data/minioreconcile/reconcile_test.go
git commit -m "feat(task-174): reconcile core with keep-list/age-guard/canonical gates"
git branch --show-current   # must print task-174-minio-tenant-reconcile
```

---

## Task 2: minio.Client adapter (`ListTenantIDs`, `PrefixInfo`) + `Store` conformance

Wire the real MinIO client to the `Store` interface. `ListTenantIDs` uses a delimiter listing; `PrefixInfo` reuses the existing recursive `List` to compute count/bytes/newest.

**Files:**
- Modify: `services/atlas-data/atlas.com/data/storage/minio/client.go` (add `ListTenantPrefixes`)
- Create: `services/atlas-data/atlas.com/data/minioreconcile/store.go`
- Test: `services/atlas-data/atlas.com/data/minioreconcile/store_test.go`

**Interfaces:**
- Consumes: `minio.Client.List(ctx,bucket,prefix) ([]minio.ObjectInfo,error)` (exists), `minio.Client.RemovePrefix`, `minio.Client.Cfg() minio.Config` with `BucketWZ/BucketAssets/BucketRenders`.
- Produces: `minio.Client.ListTenantPrefixes(ctx,bucket) ([]string,error)`; `func NewStore(mc *minio.Client) Store`.

- [ ] **Step 1: Write the failing test for `parseTenantID`**

```go
package minioreconcile

import "testing"

func TestParseTenantID(t *testing.T) {
	cases := map[string]string{
		"tenants/1cccd449-6751-4cdd-9b1a-2c33f4b6834d/": "1cccd449-6751-4cdd-9b1a-2c33f4b6834d",
		"tenants/abc/":  "abc",
		"tenants/":      "",   // no id
		"shared/x/":     "",   // wrong prefix
		"tenants/a/b/":  "a",  // only first segment
	}
	for in, want := range cases {
		if got := parseTenantID(in); got != want {
			t.Errorf("parseTenantID(%q)=%q want %q", in, got, want)
		}
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `cd services/atlas-data/atlas.com/data && go test ./minioreconcile/ -run TestParseTenantID -v`
Expected: FAIL — `undefined: parseTenantID`.

- [ ] **Step 3: Add `ListTenantPrefixes` to `client.go` and write `store.go`**

In `services/atlas-data/atlas.com/data/storage/minio/client.go`, add (after `List`):

```go
// ListTenantPrefixes returns the immediate child prefixes under "tenants/"
// (one per tenant uuid) using a delimiter listing, so it does not walk every
// object. Each returned key has the form "tenants/<uuid>/".
func (c *Client) ListTenantPrefixes(ctx context.Context, bucket string) ([]string, error) {
	ch := c.mc.ListObjects(ctx, bucket, miniogo.ListObjectsOptions{Prefix: "tenants/", Recursive: false})
	out := make([]string, 0)
	for obj := range ch {
		if obj.Err != nil {
			return nil, obj.Err
		}
		if obj.Key != "tenants/" { // skip the self entry if returned
			out = append(out, obj.Key)
		}
	}
	return out, nil
}
```

Create `services/atlas-data/atlas.com/data/minioreconcile/store.go`:

```go
package minioreconcile

import (
	"context"
	"strings"
	"time"

	minio "atlas-data/storage/minio"
)

// minioStore adapts *minio.Client to Store.
type minioStore struct{ mc *minio.Client }

// NewStore returns a Store backed by the real MinIO client.
func NewStore(mc *minio.Client) Store { return minioStore{mc: mc} }

func (s minioStore) Buckets() []string {
	c := s.mc.Cfg()
	return []string{c.BucketWZ, c.BucketAssets, c.BucketRenders}
}

func (s minioStore) ListTenantIDs(ctx context.Context, bucket string) ([]string, error) {
	keys, err := s.mc.ListTenantPrefixes(ctx, bucket)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(keys))
	for _, k := range keys {
		if id := parseTenantID(k); id != "" {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

func (s minioStore) PrefixInfo(ctx context.Context, bucket, prefix string) (PrefixInfo, error) {
	objs, err := s.mc.List(ctx, bucket, prefix)
	if err != nil {
		return PrefixInfo{}, err
	}
	var info PrefixInfo
	var newest time.Time
	for _, o := range objs {
		info.Count++
		info.Bytes += o.Size
		if o.LastModified.After(newest) {
			newest = o.LastModified
		}
	}
	info.Newest = newest
	return info, nil
}

func (s minioStore) RemovePrefix(ctx context.Context, bucket, prefix string) error {
	return s.mc.RemovePrefix(ctx, bucket, prefix)
}

// parseTenantID extracts the uuid from a "tenants/<uuid>/" key. Returns "" when
// the key is not a tenant child prefix.
func parseTenantID(key string) string {
	rest := strings.TrimPrefix(key, "tenants/")
	if rest == key { // no "tenants/" prefix
		return ""
	}
	i := strings.IndexByte(rest, '/')
	if i <= 0 {
		return ""
	}
	return rest[:i]
}
```

- [ ] **Step 4: Run the parse test + build**

Run: `cd services/atlas-data/atlas.com/data && go test ./minioreconcile/ -run TestParseTenantID -v && go build ./...`
Expected: PASS; build clean. (Compile confirms `minioStore` satisfies `Store` via `NewStore`'s return type.)

- [ ] **Step 5: Commit**

```bash
cd <worktree-root>
git add services/atlas-data/atlas.com/data/storage/minio/client.go services/atlas-data/atlas.com/data/minioreconcile/store.go services/atlas-data/atlas.com/data/minioreconcile/store_test.go
git commit -m "feat(task-174): minio Store adapter (ListTenantPrefixes, PrefixInfo)"
git branch --show-current
```

---

## Task 3: HTTP endpoint (`POST /api/data/minio/reconcile`) + wiring

Expose `Reconcile` behind the operator gate; wire into `main.go`.

**Files:**
- Create: `services/atlas-data/atlas.com/data/minioreconcile/rest.go`
- Create: `services/atlas-data/atlas.com/data/minioreconcile/handler.go`
- Test: `services/atlas-data/atlas.com/data/minioreconcile/handler_test.go`
- Modify: `services/atlas-data/atlas.com/data/main.go:174`

**Interfaces:**
- Consumes: `rest.RegisterInputHandler[T]`, `rest.HandlerDependency`, `rest.HandlerContext`, `server.RouteInitializer`, `minio.Client`, `Reconcile`, `NewStore`.
- Produces: `func InitResource(mc *minio.Client) func(si jsonapi.ServerInformation) server.RouteInitializer`; unexported `reconcileInner(store Store, clock func() time.Time)`.

- [ ] **Step 1: Read the baseline handler-test wiring**

Read `services/atlas-data/atlas.com/data/baseline/handler_test.go` (if present) or another atlas-data `*_test.go` that exercises a `RegisterInputHandler` route, to copy the exact `jsonapi.ServerInformation` test constructor and router setup. Do not invent a server-information type — reuse the repo's.

- [ ] **Step 2: Write the failing handler tests**

```go
package minioreconcile

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandler_RequiresOperator(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/data/minio/reconcile", strings.NewReader(`{"data":{"type":"minioReconciles","attributes":{"keepTenantIds":["x"]}}}`))
	// no X-Atlas-Operator header
	newTestHandler(t, &fakeStore{}).ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("want 403, got %d", rr.Code)
	}
}

func TestHandler_EmptyKeepListIs422(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/data/minio/reconcile", strings.NewReader(`{"data":{"type":"minioReconciles","attributes":{"keepTenantIds":[]}}}`))
	req.Header.Set("X-Atlas-Operator", "1")
	newTestHandler(t, &fakeStore{buckets: []string{"atlas-wz"}, prefixes: map[string]map[string]PrefixInfo{"atlas-wz": {}}}).ServeHTTP(rr, req)
	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("want 422, got %d", rr.Code)
	}
}
```

> **`newTestHandler`** is a test helper defined inside `handler_test.go` (NOT a `*_testhelpers.go` file — project rule). It builds the same mux route `InitResource` builds, but around `reconcileInner(store, now)` with the injected fake `Store` and the test `now` clock. Mirror the router/server-information setup found in Step 1's reference file.

- [ ] **Step 3: Run to verify it fails**

Run: `cd services/atlas-data/atlas.com/data && go test ./minioreconcile/ -run TestHandler -v`
Expected: FAIL — `undefined: newTestHandler` / `undefined: reconcileInner`.

- [ ] **Step 4: Write `rest.go` and `handler.go`**

`rest.go`:

```go
package minioreconcile

// ReconcileInputModel is the JSON:API input for POST /api/data/minio/reconcile.
type ReconcileInputModel struct {
	Id            string   `json:"-"`
	KeepTenantIDs []string `json:"keepTenantIds"`
	MinAgeHours   int      `json:"minAgeHours"`
	DryRun        bool     `json:"dryRun"`
}

func (ReconcileInputModel) GetName() string                                     { return "minioReconciles" }
func (m ReconcileInputModel) GetID() string                                     { return m.Id }
func (m *ReconcileInputModel) SetID(id string) error                            { m.Id = id; return nil }
func (m *ReconcileInputModel) SetToOneReferenceID(_, _ string) error            { return nil }
func (m *ReconcileInputModel) SetToManyReferenceIDs(_ string, _ []string) error { return nil }

// ReconcileOutputModel is the JSON:API report.
type ReconcileOutputModel struct {
	Id            string      `json:"-"`
	DryRun        bool        `json:"dryRun"`
	MinAgeHours   int         `json:"minAgeHours"`
	TotalPrefixes int         `json:"totalPrefixes"`
	TotalBytes    int64       `json:"totalBytes"`
	Rows          []OutputRow `json:"rows"`
}

type OutputRow struct {
	Bucket   string `json:"bucket"`
	TenantID string `json:"tenantId"`
	Action   string `json:"action"`
	Count    int    `json:"count"`
	Bytes    int64  `json:"bytes"`
	Newest   string `json:"newest"`
}

func (ReconcileOutputModel) GetName() string                                     { return "minioReconciles" }
func (m ReconcileOutputModel) GetID() string                                     { return m.Id }
func (m *ReconcileOutputModel) SetID(id string) error                            { m.Id = id; return nil }
func (m *ReconcileOutputModel) SetToOneReferenceID(_, _ string) error            { return nil }
func (m *ReconcileOutputModel) SetToManyReferenceIDs(_ string, _ []string) error { return nil }
```

`handler.go`:

```go
package minioreconcile

import (
	"errors"
	"net/http"
	"time"

	"atlas-data/rest"
	minio "atlas-data/storage/minio"

	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
)

const defaultMinAgeHours = 48

// InitResource installs POST /data/minio/reconcile.
func InitResource(mc *minio.Client) func(si jsonapi.ServerInformation) server.RouteInitializer {
	return func(si jsonapi.ServerInformation) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			r := router.PathPrefix("/data/minio").Subrouter()
			r.HandleFunc("/reconcile",
				rest.RegisterInputHandler[ReconcileInputModel](l)(si)("minio_reconcile", reconcileInner(mcStoreOrNil(mc), time.Now)),
			).Methods(http.MethodPost)
		}
	}
}

// mcStoreOrNil returns a Store for a non-nil client, else nil (handler 503s).
func mcStoreOrNil(mc *minio.Client) Store {
	if mc == nil {
		return nil
	}
	return NewStore(mc)
}

func reconcileInner(store Store, clock func() time.Time) func(d *rest.HandlerDependency, c *rest.HandlerContext, input ReconcileInputModel) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext, input ReconcileInputModel) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if store == nil {
				http.Error(w, "minio unavailable", http.StatusServiceUnavailable)
				return
			}
			if r.Header.Get("X-Atlas-Operator") != "1" {
				http.Error(w, "operator required", http.StatusForbidden)
				return
			}
			minAge := input.MinAgeHours
			if minAge <= 0 {
				minAge = defaultMinAgeHours
			}
			rep, err := Reconcile(r.Context(), d.Logger(), store, Request{
				KeepTenantIDs: input.KeepTenantIDs,
				MinAgeHours:   minAge,
				DryRun:        input.DryRun,
			}, clock())
			if err != nil {
				if errors.Is(err, ErrEmptyKeepList) {
					http.Error(w, err.Error(), http.StatusUnprocessableEntity)
					return
				}
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}
			out := toOutput(rep)
			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			w.Header().Set("Content-Type", "application/vnd.api+json")
			w.WriteHeader(http.StatusOK)
			server.MarshalResponse[ReconcileOutputModel](d.Logger())(w)(c.ServerInformation())(queryParams)(out)
		}
	}
}

func toOutput(rep Report) ReconcileOutputModel {
	rows := make([]OutputRow, 0, len(rep.Rows))
	for _, r := range rep.Rows {
		rows = append(rows, OutputRow{
			Bucket: r.Bucket, TenantID: r.TenantID, Action: r.Action,
			Count: r.Count, Bytes: r.Bytes, Newest: r.Newest.UTC().Format(time.RFC3339),
		})
	}
	return ReconcileOutputModel{Id: "current", DryRun: rep.DryRun, MinAgeHours: rep.MinAgeHours, TotalPrefixes: rep.TotalPrefixes, TotalBytes: rep.TotalBytes, Rows: rows}
}
```

- [ ] **Step 5: Wire into `main.go`**

In `services/atlas-data/atlas.com/data/main.go`, add the import `"atlas-data/minioreconcile"` and, after line 174 (`tenantpurge.InitResource(...)`), add:

```go
		AddRouteInitializer(minioreconcile.InitResource(mc)(GetServer())).
```

- [ ] **Step 6: Run tests + build + vet**

Run: `cd services/atlas-data/atlas.com/data && go test -race ./minioreconcile/ -v && go vet ./... && go build ./...`
Expected: PASS; vet/build clean.

- [ ] **Step 7: Commit**

```bash
cd <worktree-root>
git add services/atlas-data/atlas.com/data/minioreconcile/rest.go services/atlas-data/atlas.com/data/minioreconcile/handler.go services/atlas-data/atlas.com/data/minioreconcile/handler_test.go services/atlas-data/atlas.com/data/main.go
git commit -m "feat(task-174): POST /api/data/minio/reconcile endpoint + wiring"
git branch --show-current
```

---

## Task 4: Orchestrator script `reconcile-minio.sh` + bats

Enumerate the cross-namespace tenant union (fail-closed) and POST it to the executor.

**Files:**
- Create: `services/atlas-pr-bootstrap/scripts/reconcile-minio.sh`
- Test: `services/atlas-pr-bootstrap/test/reconcile_minio_test.bats`

**Interfaces:**
- Consumes: `lib.sh` (`log`, `record_error`, `run_phase`, `summarize_phases`). Env: `KUBECTL` (default `kubectl`), `CURL` (default `curl`), `ATLAS_DATA_BASE` (default `http://atlas-data.atlas-main.svc.cluster.local:8080`), `RECONCILE_DRY_RUN` (default `true`), `RECONCILE_MIN_AGE_HOURS` (default `48`).
- Produces: exit 0 on success; non-zero on any enumeration failure (fail-closed) or executor non-2xx.

- [ ] **Step 1: Write the failing bats**

```bash
#!/usr/bin/env bats
# test/reconcile_minio_test.bats

setup() {
  SCRIPT_DIR="$(cd "$(dirname "$BATS_TEST_FILENAME")/../scripts" && pwd)"
  export TMP="$BATS_TEST_TMPDIR"
  cat >"$TMP/kubectl" <<'EOF'
#!/usr/bin/env bash
echo "namespace/atlas-main"
echo "namespace/atlas-pr-42"
EOF
  chmod +x "$TMP/kubectl"
  export KUBECTL="$TMP/kubectl"
}

@test "unions tenant ids across namespaces and posts keep-list" {
  cat >"$TMP/curl" <<EOF
#!/usr/bin/env bash
args="\$*"
case "\$args" in
  *atlas-main*tenants*)  echo '{"data":[{"id":"aaaa"}]}'; exit 0 ;;
  *atlas-pr-42*tenants*) echo '{"data":[{"id":"bbbb"}]}'; exit 0 ;;
  *minio/reconcile*)     echo "\$args" >>"$TMP/posted"; echo '{"totalBytes":0}'; exit 0 ;;
esac
exit 0
EOF
  chmod +x "$TMP/curl"; export CURL="$TMP/curl"
  run "$SCRIPT_DIR/reconcile-minio.sh"
  [ "$status" -eq 0 ]
  grep -q '"aaaa"' "$TMP/posted"
  grep -q '"bbbb"' "$TMP/posted"
}

@test "fail-closed: unreachable namespace aborts without POST" {
  cat >"$TMP/curl" <<EOF
#!/usr/bin/env bash
args="\$*"
case "\$args" in
  *atlas-main*tenants*)  echo '{"data":[{"id":"aaaa"}]}'; exit 0 ;;
  *atlas-pr-42*tenants*) exit 7 ;;
  *minio/reconcile*)     echo "posted" >>"$TMP/posted"; exit 0 ;;
esac
exit 0
EOF
  chmod +x "$TMP/curl"; export CURL="$TMP/curl"
  run "$SCRIPT_DIR/reconcile-minio.sh"
  [ "$status" -ne 0 ]
  [ ! -f "$TMP/posted" ]
}

@test "refuses empty union" {
  cat >"$TMP/curl" <<EOF
#!/usr/bin/env bash
args="\$*"
case "\$args" in
  *tenants*)         echo '{"data":[]}'; exit 0 ;;
  *minio/reconcile*) echo "posted" >>"$TMP/posted"; exit 0 ;;
esac
exit 0
EOF
  chmod +x "$TMP/curl"; export CURL="$TMP/curl"
  run "$SCRIPT_DIR/reconcile-minio.sh"
  [ "$status" -ne 0 ]
  [ ! -f "$TMP/posted" ]
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `cd services/atlas-pr-bootstrap && bats test/reconcile_minio_test.bats`
Expected: FAIL — script does not exist.

- [ ] **Step 3: Write `reconcile-minio.sh`**

```bash
#!/usr/bin/env bash
# Daily reconciliation orchestrator. Enumerates the live-tenant UUID union
# across atlas-main + atlas-pr-* namespaces (FAIL-CLOSED: any unreachable env
# aborts the run), then POSTs the keep-list to atlas-data's reconcile endpoint.
#
# Env:
#   KUBECTL                 (default kubectl)
#   CURL                    (default curl)
#   ATLAS_DATA_BASE         (default http://atlas-data.atlas-main.svc.cluster.local:8080)
#   RECONCILE_DRY_RUN       (default true)
#   RECONCILE_MIN_AGE_HOURS (default 48)
set -uo pipefail

. "$(dirname "$0")/lib.sh"

: "${KUBECTL:=kubectl}"
: "${CURL:=curl}"
: "${ATLAS_DATA_BASE:=http://atlas-data.atlas-main.svc.cluster.local:8080}"
: "${RECONCILE_DRY_RUN:=true}"
: "${RECONCILE_MIN_AGE_HOURS:=48}"

do_reconcile() {
  local namespaces ns url ids all=""
  if ! namespaces=$("$KUBECTL" get ns -o name 2>/dev/null \
        | sed 's|^namespace/||' \
        | grep -E '^(atlas-main|atlas-pr-.+)$'); then
    record_error reconcile "could not list namespaces"
    return 1
  fi
  if [ -z "$namespaces" ]; then
    record_error reconcile "no atlas namespaces found"
    return 1
  fi

  while IFS= read -r ns; do
    [ -z "$ns" ] && continue
    url="http://atlas-ingress.${ns}.svc.cluster.local/api/tenants"
    ATLAS_STEP=reconcile log info "enumerating tenants in ${ns}"
    if ! ids=$("$CURL" -fsS -H 'Accept: application/vnd.api+json' "$url" 2>/dev/null \
          | jq -r '.data[].id' 2>/dev/null); then
      # FAIL-CLOSED: a discovered env we cannot read must not be treated as orphaned.
      record_error reconcile "could not enumerate tenants in ${ns}; aborting (fail-closed)"
      return 1
    fi
    all="${all}${ids}"$'\n'
  done <<<"$namespaces"

  local union
  union=$(printf '%s\n' "$all" | sed '/^$/d' | sort -u)
  if [ -z "$union" ]; then
    record_error reconcile "empty tenant union; refusing to reconcile"
    return 1
  fi

  local keep_json body
  keep_json=$(printf '%s\n' "$union" | jq -R . | jq -cs .)
  body=$(jq -cn --argjson keep "$keep_json" \
      --argjson age "$RECONCILE_MIN_AGE_HOURS" \
      --argjson dry "$RECONCILE_DRY_RUN" \
      '{data:{type:"minioReconciles",attributes:{keepTenantIds:$keep,minAgeHours:$age,dryRun:$dry}}}')

  ATLAS_STEP=reconcile log info "posting keep-list ($(printf '%s\n' "$union" | wc -l | tr -d ' ') tenants, dryRun=${RECONCILE_DRY_RUN}, minAgeHours=${RECONCILE_MIN_AGE_HOURS})"
  local status
  status=$("$CURL" -s -o /tmp/reconcile-resp -w '%{http_code}' -X POST \
      -H 'X-Atlas-Operator: 1' \
      -H 'Content-Type: application/vnd.api+json' \
      -d "$body" \
      "${ATLAS_DATA_BASE}/api/data/minio/reconcile" 2>/dev/null || echo 000)
  case "$status" in
    2*) ATLAS_STEP=reconcile log info "reconcile ok (status ${status}): $(cat /tmp/reconcile-resp 2>/dev/null)" ;;
    *)  record_error reconcile "reconcile POST failed (status ${status})"; return 1 ;;
  esac
  return 0
}

ATLAS_PHASE_ERRORS=()
run_phase reconcile do_reconcile
summarize_phases 1
exit $?
```

> Mirror the exact `jq -cn` JSON-building idiom already in `predelete-purge.sh`/`bootstrap.sh`. `minAgeHours`/`dryRun` must be injected as JSON number/bool (`--argjson`), not strings.

- [ ] **Step 4: Run bats to verify pass**

Run: `cd services/atlas-pr-bootstrap && bats test/reconcile_minio_test.bats`
Expected: PASS (3 tests). `jq` must be on PATH (present in the bootstrap image + CI).

- [ ] **Step 5: Commit**

```bash
cd <worktree-root>
chmod +x services/atlas-pr-bootstrap/scripts/reconcile-minio.sh
git add services/atlas-pr-bootstrap/scripts/reconcile-minio.sh services/atlas-pr-bootstrap/test/reconcile_minio_test.bats
git commit -m "feat(task-174): reconcile-minio orchestrator (cross-ns union, fail-closed)"
git branch --show-current
```

---

## Task 5: K8s manifests — CronJob + RBAC + ConfigMap

Schedule the orchestrator daily in `atlas-main`, dry-run by default.

**Files:**
- Create: `deploy/k8s/base/atlas-minio-reconcile.yaml`
- Modify: `deploy/k8s/base/kustomization.yaml` (add the resource)
- Inspect: `deploy/k8s/overlays/main/kustomization.yaml`, `deploy/k8s/overlays/pr/kustomization.yaml`

- [ ] **Step 1: Confirm base image ref + script path + kustomization pattern**

Read `deploy/k8s/base/kustomization.yaml` (how `resources:` are listed), `deploy/k8s/overlays/pr/sync-bootstrap.yaml:96` (exact `atlas-pr-bootstrap` image repo/tag), and `services/atlas-pr-bootstrap/Dockerfile` (where `scripts/` lands in the image — set `command:` accordingly, e.g. `/scripts/reconcile-minio.sh`). Do not guess these — read them.

- [ ] **Step 2: Write `atlas-minio-reconcile.yaml`**

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: atlas-minio-reconcile
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: atlas-minio-reconcile
rules:
  - apiGroups: [""]
    resources: ["namespaces"]
    verbs: ["list"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: atlas-minio-reconcile
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: atlas-minio-reconcile
subjects:
  - kind: ServiceAccount
    name: atlas-minio-reconcile
    namespace: atlas-main   # overlay sets the real namespace
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: atlas-minio-reconcile-config
data:
  RECONCILE_DRY_RUN: "true"
  RECONCILE_MIN_AGE_HOURS: "48"
---
apiVersion: batch/v1
kind: CronJob
metadata:
  name: atlas-minio-reconcile
spec:
  schedule: "0 3 * * *"          # daily 03:00
  concurrencyPolicy: Forbid
  successfulJobsHistoryLimit: 3
  failedJobsHistoryLimit: 3
  jobTemplate:
    spec:
      backoffLimit: 0
      template:
        spec:
          serviceAccountName: atlas-minio-reconcile
          restartPolicy: Never
          containers:
            - name: reconcile
              image: ghcr.io/chronicle20/atlas-pr-bootstrap/atlas-pr-bootstrap:latest  # replace tag with the overlay-pinned one from Step 1
              command: ["/scripts/reconcile-minio.sh"]                                  # replace path per Dockerfile from Step 1
              envFrom:
                - configMapRef:
                    name: atlas-minio-reconcile-config
              env:
                - name: ATLAS_DATA_BASE
                  value: http://atlas-data.atlas-main.svc.cluster.local:8080
```

- [ ] **Step 3: Register in base kustomization**

Add `- atlas-minio-reconcile.yaml` to the `resources:` list in `deploy/k8s/base/kustomization.yaml` (near the other `atlas-*` entries).

- [ ] **Step 4: Decide overlay treatment**

The CronJob is `atlas-main`-only. Read both overlays' `kustomization.yaml`: if the `pr` overlay pulls in base wholesale (which would schedule the CronJob per-PR-env — undesirable), add a `pr`-overlay patch to delete or `suspend: true` the CronJob, keeping it only in `main`. Follow whatever exclusion idiom already exists in the overlays (e.g. `patches:` with `$patch: delete`). If base is not blanket-included by `pr`, no overlay change is needed.

- [ ] **Step 5: Validate kustomize builds**

Run: `kubectl kustomize deploy/k8s/overlays/main >/dev/null && kubectl kustomize deploy/k8s/overlays/pr >/dev/null && echo OK`
Expected: `OK`. If deploy/k8s changed lists the guard checks, run `tools/service-registration-guard.sh` (no new Go service → services.json untouched → expect clean).

- [ ] **Step 6: Commit**

```bash
cd <worktree-root>
git add deploy/k8s/base/atlas-minio-reconcile.yaml deploy/k8s/base/kustomization.yaml deploy/k8s/overlays
git commit -m "feat(task-174): atlas-minio-reconcile CronJob + RBAC + dry-run ConfigMap"
git branch --show-current
```

---

## Task 6: Harden the PreDelete purge hook (B)

Add bounded retry to the per-tenant DELETE, with a visible alert on final failure.

**Files:**
- Modify: `services/atlas-pr-bootstrap/scripts/predelete-purge.sh`
- Modify: `services/atlas-pr-bootstrap/test/predelete_test.bats`

- [ ] **Step 1: Read current hook + its bats**

Read `services/atlas-pr-bootstrap/scripts/predelete-purge.sh` and `test/predelete_test.bats` to match the existing stub/harness conventions before editing.

- [ ] **Step 2: Write a failing test for DELETE retry**

Add to `test/predelete_test.bats` a case where the tenant `DELETE` returns `503` on the first attempt and `202` on the second; assert the hook succeeds (exit 0) and the tenant is reported purged. Model the curl stub on the existing predelete tests (a counter file in `$BATS_TEST_TMPDIR` that flips the status on the 2nd call).

- [ ] **Step 3: Run to verify it fails**

Run: `cd services/atlas-pr-bootstrap && bats test/predelete_test.bats`
Expected: FAIL — current hook treats the first 503 as a permanent failure.

- [ ] **Step 4: Add retry/backoff to `predelete-purge.sh`**

Wrap the per-tenant DELETE in a bounded retry (reuse `retry` from `lib.sh` if its signature fits, else a small inline loop of 3 attempts with a short `sleep`), treating only a final non-2xx as `rc=1`, and emit `log error` (alert-level) on exhaustion. Preserve: the empty-list refusal, the GET enumeration `-f` behavior, and the non-zero exit on any final failure. Keep the change minimal — do not restructure `do_purge_tenants` beyond the retry wrap.

- [ ] **Step 5: Run predelete bats + verify pass**

Run: `cd services/atlas-pr-bootstrap && bats test/predelete_test.bats`
Expected: PASS (existing cases + the new retry case).

- [ ] **Step 6: Commit**

```bash
cd <worktree-root>
git add services/atlas-pr-bootstrap/scripts/predelete-purge.sh services/atlas-pr-bootstrap/test/predelete_test.bats
git commit -m "feat(task-174): harden predelete purge hook with bounded DELETE retry"
git branch --show-current
```

---

## Task 7: Full verification + task docs

Run every gate and record the audit.

**Files:**
- Create/update: `docs/tasks/task-174-minio-tenant-reconcile/audit.md` (via code-review step)

- [ ] **Step 1: Go gates in atlas-data**

Run (from `<worktree-root>`):
```bash
cd services/atlas-data/atlas.com/data
go test -race ./... && go vet ./... && go build ./...
```
Expected: all clean.

- [ ] **Step 2: Repo-root guards**

Run from `<worktree-root>`:
```bash
tools/redis-key-guard.sh && tools/goroutine-guard.sh && tools/service-registration-guard.sh && echo GUARDS-OK
```
Expected: `GUARDS-OK` (reconcile uses no raw redis and spawns no goroutines; service-registration unaffected — no new Go service).

- [ ] **Step 3: Docker bake atlas-data**

Run from `<worktree-root>`:
```bash
docker buildx bake atlas-data
```
Expected: build succeeds. (No new shared lib added → shared Dockerfile needs no COPY change; this confirms the image still builds.)

- [ ] **Step 4: bats suites**

Run:
```bash
cd services/atlas-pr-bootstrap
bats test/reconcile_minio_test.bats test/predelete_test.bats
```
Expected: all pass.

- [ ] **Step 5: Kustomize render**

Run from `<worktree-root>`:
```bash
kubectl kustomize deploy/k8s/overlays/main >/dev/null && kubectl kustomize deploy/k8s/overlays/pr >/dev/null && echo KUSTOMIZE-OK
```
Expected: `KUSTOMIZE-OK`.

- [ ] **Step 6: Code review + commit audit**

Invoke `superpowers:requesting-code-review` (dispatches `plan-adherence-reviewer` + `backend-guidelines-reviewer` since Go changed; no TS). Address findings, then commit the audit doc.

```bash
cd <worktree-root>
git add docs/tasks/task-174-minio-tenant-reconcile/audit.md
git commit -m "docs(task-174): code-review audit"
git branch --show-current
```

---

## Self-Review (author)

**Spec coverage:**
- Component 1 (executor) → Tasks 1–3: empty-refusal (T1), age guard (T1), canonical (T1), keep-list (T1), Store/testability (T1–2), endpoint + operator gate + dryRun default (T3). ✅
- Component 2 (orchestrator) → Task 4 (cross-ns union, fail-closed, empty-refusal) + Task 5 (CronJob/RBAC/ConfigMap dry-run default, daily schedule). ✅
- Component 3 (hook hardening) → Task 6. ✅
- Safety properties → fail-closed (T4), age guard + canonical + empty-refusal (T1), dry-run default (T3 default + T5 ConfigMap), observability (T1 per-row logging + report; T4 logs response). ✅
- Testing matrix (spec §Testing) → T1 unit gates, T3 handler gates, T4 orchestrator bats, T6 hook bats. ✅
- Verification gates → Task 7. ✅

**Placeholder scan:** No TBD/TODO. The "mirror the existing idiom" notes (T3 test-server-information constructor; T4 jq idiom; T5 image tag + script path + overlay exclusion) point at named reference files the implementer must read, not logic left unspecified — required because those helpers/paths are repo-local and must not be invented.

**Type consistency:** `Store`, `PrefixInfo{Count,Bytes,Newest}`, `Request{KeepTenantIDs,MinAgeHours,DryRun}`, `Report{...Rows []ReportRow}`, `Reconcile(ctx,l,store,req,now)`, `reconcileInner(store, clock)`, `NewStore(mc)`, `ListTenantPrefixes`, `parseTenantID` — identical across Tasks 1–3. `RECONCILE_DRY_RUN`/`RECONCILE_MIN_AGE_HOURS`/`ATLAS_DATA_BASE` identical across Tasks 4–5.
