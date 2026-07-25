# Admin Navigation by Blast Radius + Canonical Baselines Manager — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Regroup the atlas-ui sidebar by blast radius (tenant vs deployment), make the tenant switcher inert on deployment-wide routes, split Bootstrap into a tenant-only Setup page and a new deployment-wide Baselines page with a full canonical upload→process→publish workflow driven by synthetic (nil-tenant) headers, and add a `GET /data/baselines` endpoint to atlas-data that lists published canonical baselines from MinIO.

**Architecture:** atlas-data gains one read-only handler in the existing `baseline` package, backed by a new `minio.Client.List` method and a `Lister` behind a package-local `objectStore` interface (map-backed fake in tests). atlas-ui gains a single `isDeploymentRoute` predicate consumed by both the tenant switcher and a shell-mounted scope banner; the canonical service layer is a set of separate functions taking a `CanonicalSelection` (region/major/minor) instead of a `Tenant`, so the tenant path structurally cannot issue shared-scope requests.

**Tech Stack:** Go 1.x (gorilla/mux, api2go JSON:API, minio-go v7, logrus, testify-free table tests), React 19 + TypeScript (Vite, react-router-dom, TanStack React Query, shadcn/ui, sonner, vitest + @testing-library/react).

## Global Constraints

- Worktree root: `.worktrees/task-134-admin-nav-baselines`, branch `task-134-admin-nav-baselines`. Every implementer MUST `cd` into the worktree first and verify `git branch --show-current` = `task-134-admin-nav-baselines` after each commit.
- Commit message format: `<type>(task-134): <message>` (e.g. `feat(task-134): ...`).
- Nil tenant UUID (exact): `00000000-0000-0000-0000-000000000000`.
- Operator header (exact): `X-Atlas-Operator: 1`.
- Deployment route prefixes (exact set): `/templates`, `/tenants`, `/services`, `/baselines`.
- Scope banner copy (exact): "Changes on this page affect all tenants."
- Deployment group caption (exact): "Applies to all tenants".
- Inert switcher copy (exact): "Deployment-wide" / secondary line "tenant selection inactive".
- Re-publish confirmation copy (exact): "This will replace the shared canonical baseline for {region} v{major}.{minor}."
- Baselines JSON:API resource type (exact): `baselines`; id format `<region>/<major>.<minor>` via the existing `PublishOutputId`.
- MinIO keys (existing, do not change): dump `baseline/regions/<region>/versions/<major>.<minor>/documents.dump`, sidecar `...documents.dump.sha256` (`DumpKey`/`ShaKey` in `baseline/dump.go`).
- The tar header's internal `publishedAt` is epoch-zero by design and MUST NOT be read; `publishedAt` comes from the MinIO object's LastModified.
- No `// TODO`, stubs, or 501s in landed commits.
- Final gates (Task 17): atlas-data `go test -race ./...`, `go vet ./...`, `go build ./...`, `docker buildx bake atlas-data` from the worktree root, `tools/redis-key-guard.sh` from the worktree root; atlas-ui `npm run build` + `npm run test`.
- All `go` commands for atlas-data run from `services/atlas-data/atlas.com/data/`. All `npm` commands run from `services/atlas-ui/`.

---

### Task 1: `minio.Client.List` — per-object prefix listing

**Files:**
- Modify: `services/atlas-data/atlas.com/data/storage/minio/client.go`

**Interfaces:**
- Consumes: existing `miniogo.Client.ListObjects` (already used by `PrefixStats`).
- Produces: `type ObjectInfo struct { Key string; Size int64; LastModified time.Time }` and `func (c *Client) List(ctx context.Context, bucket, prefix string) ([]ObjectInfo, error)` — Task 3's `objectStore` interface and Task 4's handler depend on these exact names.

This is a thin SDK wrapper with no unit-testable seam (it needs a live MinIO); its behavior is exercised in Task 3 through the `objectStore` fake, and compilation is the gate here. Do not add a fake-server test.

- [ ] **Step 1: Add `ObjectInfo` and `List` to the minio client**

Append to `services/atlas-data/atlas.com/data/storage/minio/client.go` (after `PrefixStats`):

```go
// ObjectInfo is the per-object subset of MinIO metadata returned by List.
type ObjectInfo struct {
	Key          string
	Size         int64
	LastModified time.Time
}

// List returns every object under bucket/prefix (recursive), one entry per
// object. Mirrors PrefixStats but preserves per-object keys for callers that
// need to enumerate rather than aggregate.
func (c *Client) List(ctx context.Context, bucket, prefix string) ([]ObjectInfo, error) {
	ch := c.mc.ListObjects(ctx, bucket, miniogo.ListObjectsOptions{Prefix: prefix, Recursive: true})
	out := make([]ObjectInfo, 0)
	for obj := range ch {
		if obj.Err != nil {
			return nil, obj.Err
		}
		out = append(out, ObjectInfo{Key: obj.Key, Size: obj.Size, LastModified: obj.LastModified})
	}
	return out, nil
}
```

No new imports are needed (`context`, `time`, and `miniogo` are already imported).

- [ ] **Step 2: Verify it compiles and existing tests still pass**

Run from `services/atlas-data/atlas.com/data/`:

```bash
go build ./... && go test -race ./storage/...
```

Expected: both succeed.

- [ ] **Step 3: Commit**

```bash
git add services/atlas-data/atlas.com/data/storage/minio/client.go
git commit -m "feat(task-134): add per-object List to minio storage client"
```

---

### Task 2: `parseDumpKey` — canonical dump-key parsing

**Files:**
- Create: `services/atlas-data/atlas.com/data/baseline/list.go`
- Create: `services/atlas-data/atlas.com/data/baseline/list_test.go`

**Interfaces:**
- Consumes: `DumpKey(region string, major, minor int) string` from `baseline/dump.go` (round-trip tests).
- Produces: `func parseDumpKey(key string) (region string, major, minor int, ok bool)` — Task 3's `Lister.List` calls it.

- [ ] **Step 1: Write the failing tests**

Create `services/atlas-data/atlas.com/data/baseline/list_test.go`:

```go
package baseline

import (
	"testing"
)

func TestParseDumpKeyRoundTrip(t *testing.T) {
	cases := []struct {
		region string
		major  int
		minor  int
	}{
		{"GMS", 83, 1},
		{"GMS", 84, 1},
		{"JMS", 185, 1},
	}
	for _, c := range cases {
		key := DumpKey(c.region, c.major, c.minor)
		region, major, minor, ok := parseDumpKey(key)
		if !ok {
			t.Fatalf("parseDumpKey(%q) not ok", key)
		}
		if region != c.region || major != c.major || minor != c.minor {
			t.Fatalf("parseDumpKey(%q) = %s/%d.%d", key, region, major, minor)
		}
	}
}

func TestParseDumpKeyRejectsMalformed(t *testing.T) {
	bad := []string{
		"",
		"baseline/regions/GMS/versions/83.1/other.file",
		"baseline/regions/GMS/versions/83.1",
		"baseline/regions/GMS/versions/831/documents.dump",
		"baseline/regions/GMS/versions/x.y/documents.dump",
		"baseline/regions/GMS/versions/83./documents.dump",
		"baseline/regions/GMS/versions/.1/documents.dump",
		"baseline/regions/GMS/versions/-1.2/documents.dump",
		"baseline/regions/GMS/versions/83.-2/documents.dump",
		"baseline/regions//versions/83.1/documents.dump",
		"shared/regions/GMS/versions/83.1/documents.dump",
		"baseline/other/GMS/versions/83.1/documents.dump",
		"baseline/regions/GMS/versions/83.1/extra/documents.dump",
	}
	for _, key := range bad {
		if _, _, _, ok := parseDumpKey(key); ok {
			t.Fatalf("parseDumpKey(%q) unexpectedly ok", key)
		}
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run from `services/atlas-data/atlas.com/data/`:

```bash
go test ./baseline/ -run TestParseDumpKey -v
```

Expected: compile FAILURE — `undefined: parseDumpKey`.

- [ ] **Step 3: Implement `parseDumpKey`**

Create `services/atlas-data/atlas.com/data/baseline/list.go`:

```go
package baseline

import (
	"strconv"
	"strings"
)

// parseDumpKey extracts (region, major, minor) from a canonical dump key of
// the exact shape DumpKey produces:
// baseline/regions/<region>/versions/<major>.<minor>/documents.dump.
// Keys that do not parse are the caller's cue to skip-and-warn, never fail.
func parseDumpKey(key string) (string, int, int, bool) {
	parts := strings.Split(key, "/")
	if len(parts) != 6 || parts[0] != "baseline" || parts[1] != "regions" ||
		parts[3] != "versions" || parts[5] != "documents.dump" {
		return "", 0, 0, false
	}
	region := parts[2]
	if region == "" {
		return "", 0, 0, false
	}
	ver := parts[4]
	dot := strings.LastIndex(ver, ".")
	if dot <= 0 || dot == len(ver)-1 {
		return "", 0, 0, false
	}
	major, err := strconv.Atoi(ver[:dot])
	if err != nil || major < 0 {
		return "", 0, 0, false
	}
	minor, err := strconv.Atoi(ver[dot+1:])
	if err != nil || minor < 0 {
		return "", 0, 0, false
	}
	return region, major, minor, true
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./baseline/ -run TestParseDumpKey -v
```

Expected: PASS (both tests).

- [ ] **Step 5: Commit**

```bash
git add services/atlas-data/atlas.com/data/baseline/list.go services/atlas-data/atlas.com/data/baseline/list_test.go
git commit -m "feat(task-134): parse canonical baseline dump keys"
```

---

### Task 3: `Lister` + `ListItemModel` — derive the baselines collection

**Files:**
- Modify: `services/atlas-data/atlas.com/data/baseline/list.go`
- Modify: `services/atlas-data/atlas.com/data/baseline/list_test.go`
- Modify: `services/atlas-data/atlas.com/data/baseline/rest.go`

**Interfaces:**
- Consumes: `minio.ObjectInfo` and the `List`/`Get` method shapes from Task 1; `parseDumpKey` from Task 2; existing `ShaKey`, `PublishOutputId`.
- Produces:
  - `type ListItemModel struct { Id string; Region string; MajorVersion int; MinorVersion int; Sha256 string; PublishedAt string; SizeBytes int64 }` with `GetName() string` returning `"baselines"`, `GetID()`, `SetID(string) error` (JSON:API surface; in `rest.go`).
  - `type objectStore interface { List(ctx, bucket, prefix string) ([]minio.ObjectInfo, error); Get(ctx, bucket, key string) (io.ReadCloser, error) }` (unexported, in `list.go`).
  - `type Lister struct { MC objectStore; Bucket string; L logrus.FieldLogger }` with `func (li Lister) List(ctx context.Context) ([]ListItemModel, error)` — Task 4's handler constructs `Lister{MC: mc, Bucket: mc.Cfg().BucketCanonical, L: d.Logger()}`.

- [ ] **Step 1: Add `ListItemModel` to `rest.go`**

Append to `services/atlas-data/atlas.com/data/baseline/rest.go`:

```go
// ListItemModel is one published baseline in the GET /data/baselines
// JSON:API collection. PublishedAt is RFC3339 (the MinIO object's
// LastModified — the tar header's internal publishedAt is epoch-zero by
// design and is never read). Sha256 is "" when the sidecar is missing or
// unreadable so a partially-published baseline stays visible.
type ListItemModel struct {
	Id           string `json:"-"`
	Region       string `json:"region"`
	MajorVersion int    `json:"majorVersion"`
	MinorVersion int    `json:"minorVersion"`
	Sha256       string `json:"sha256"`
	PublishedAt  string `json:"publishedAt"`
	SizeBytes    int64  `json:"sizeBytes"`
}

func (ListItemModel) GetName() string          { return "baselines" }
func (m ListItemModel) GetID() string          { return m.Id }
func (m *ListItemModel) SetID(id string) error { m.Id = id; return nil }
```

- [ ] **Step 2: Write the failing `Lister` tests**

Append to `services/atlas-data/atlas.com/data/baseline/list_test.go` (add the new imports to the existing import block):

```go
import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	minio "atlas-data/storage/minio"

	"github.com/sirupsen/logrus"
)

// fakeStore is the map-backed objectStore used to drive Lister without a
// live MinIO.
type fakeStore struct {
	objs    []minio.ObjectInfo
	blobs   map[string][]byte
	listErr error
}

func (f *fakeStore) List(_ context.Context, _ string, prefix string) ([]minio.ObjectInfo, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	out := make([]minio.ObjectInfo, 0)
	for _, o := range f.objs {
		if strings.HasPrefix(o.Key, prefix) {
			out = append(out, o)
		}
	}
	return out, nil
}

func (f *fakeStore) Get(_ context.Context, _ string, key string) (io.ReadCloser, error) {
	b, ok := f.blobs[key]
	if !ok {
		return nil, errors.New("NoSuchKey")
	}
	return io.NopCloser(bytes.NewReader(b)), nil
}

func quietLogger() logrus.FieldLogger {
	l := logrus.New()
	l.SetOutput(io.Discard)
	return l
}

func validSha() string { return strings.Repeat("ab", 32) } // 64 hex chars

func TestListerEmptyBucketReturnsEmptyCollection(t *testing.T) {
	li := Lister{MC: &fakeStore{}, Bucket: "canonical", L: quietLogger()}
	items, err := li.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if items == nil {
		t.Fatalf("List returned nil; want empty non-nil slice (marshals to \"data\": [])")
	}
	if len(items) != 0 {
		t.Fatalf("len = %d, want 0", len(items))
	}
}

func TestListerParsesSortsAndFillsAttributes(t *testing.T) {
	t1 := time.Date(2026, 7, 4, 12, 34, 56, 0, time.UTC)
	t2 := time.Date(2026, 7, 3, 1, 2, 3, 0, time.UTC)
	fs := &fakeStore{
		objs: []minio.ObjectInfo{
			// Deliberately out of order: JMS first, GMS 84 before GMS 83.
			{Key: DumpKey("JMS", 185, 1), Size: 300, LastModified: t2},
			{Key: DumpKey("GMS", 84, 1), Size: 200, LastModified: t2},
			{Key: DumpKey("GMS", 83, 1), Size: 100, LastModified: t1},
		},
		blobs: map[string][]byte{
			ShaKey("GMS", 83, 1): []byte(validSha()),
			ShaKey("GMS", 84, 1): []byte(validSha()),
			// JMS sidecar intentionally absent.
		},
	}
	li := Lister{MC: fs, Bucket: "canonical", L: quietLogger()}
	items, err := li.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("len = %d, want 3", len(items))
	}
	// Sorted by (region, major, minor) ascending.
	if items[0].Id != "GMS/83.1" || items[1].Id != "GMS/84.1" || items[2].Id != "JMS/185.1" {
		t.Fatalf("order = %s, %s, %s", items[0].Id, items[1].Id, items[2].Id)
	}
	first := items[0]
	if first.Region != "GMS" || first.MajorVersion != 83 || first.MinorVersion != 1 {
		t.Fatalf("first identity = %s/%d.%d", first.Region, first.MajorVersion, first.MinorVersion)
	}
	if first.Sha256 != validSha() {
		t.Fatalf("first sha = %q", first.Sha256)
	}
	if first.PublishedAt != "2026-07-04T12:34:56Z" {
		t.Fatalf("first publishedAt = %q", first.PublishedAt)
	}
	if first.SizeBytes != 100 {
		t.Fatalf("first size = %d", first.SizeBytes)
	}
	// Missing sidecar -> listed with empty sha, not dropped.
	if items[2].Sha256 != "" {
		t.Fatalf("JMS sha = %q, want empty", items[2].Sha256)
	}
}

func TestListerSkipsNonDumpAndUnparseableKeys(t *testing.T) {
	t1 := time.Date(2026, 7, 4, 0, 0, 0, 0, time.UTC)
	fs := &fakeStore{
		objs: []minio.ObjectInfo{
			{Key: DumpKey("GMS", 83, 1), Size: 100, LastModified: t1},
			{Key: ShaKey("GMS", 83, 1), Size: 64, LastModified: t1},                                    // sidecar itself: not a dump
			{Key: "baseline/regions/GMS/notes.txt", Size: 5, LastModified: t1},                          // junk under prefix
			{Key: "baseline/regions/BAD/versions/x.y/documents.dump", Size: 5, LastModified: t1},        // unparseable version
		},
		blobs: map[string][]byte{ShaKey("GMS", 83, 1): []byte(validSha())},
	}
	li := Lister{MC: fs, Bucket: "canonical", L: quietLogger()}
	items, err := li.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(items) != 1 || items[0].Id != "GMS/83.1" {
		t.Fatalf("items = %+v, want single GMS/83.1", items)
	}
}

func TestListerMalformedSidecarYieldsEmptySha(t *testing.T) {
	t1 := time.Date(2026, 7, 4, 0, 0, 0, 0, time.UTC)
	for name, blob := range map[string][]byte{
		"short":  []byte("abc123"),
		"nonhex": []byte(strings.Repeat("zz", 32)),
	} {
		fs := &fakeStore{
			objs:  []minio.ObjectInfo{{Key: DumpKey("GMS", 83, 1), Size: 100, LastModified: t1}},
			blobs: map[string][]byte{ShaKey("GMS", 83, 1): blob},
		}
		li := Lister{MC: fs, Bucket: "canonical", L: quietLogger()}
		items, err := li.List(context.Background())
		if err != nil {
			t.Fatalf("%s: List: %v", name, err)
		}
		if len(items) != 1 || items[0].Sha256 != "" {
			t.Fatalf("%s: items = %+v, want single entry with empty sha", name, items)
		}
	}
}

func TestListerSurfacesListError(t *testing.T) {
	li := Lister{MC: &fakeStore{listErr: errors.New("boom")}, Bucket: "canonical", L: quietLogger()}
	if _, err := li.List(context.Background()); err == nil {
		t.Fatalf("expected error")
	}
}
```

Note: `list_test.go` already has a `testing` import from Task 2 — merge the import blocks rather than duplicating them.

- [ ] **Step 3: Run tests to verify they fail**

```bash
go test ./baseline/ -run TestLister -v
```

Expected: compile FAILURE — `undefined: Lister`.

- [ ] **Step 4: Implement `objectStore`, `Lister`, `readSha`**

Extend `services/atlas-data/atlas.com/data/baseline/list.go` to:

```go
package baseline

import (
	"context"
	"encoding/hex"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"

	minio "atlas-data/storage/minio"

	"github.com/sirupsen/logrus"
)

// baselinePrefix is the canonical-bucket prefix every published baseline
// lives under (see DumpKey).
const baselinePrefix = "baseline/regions/"

// objectStore is the narrow slice of *minio.Client the Lister consumes.
// Tests inject a map-backed fake; *minio.Client satisfies it.
type objectStore interface {
	List(ctx context.Context, bucket, prefix string) ([]minio.ObjectInfo, error)
	Get(ctx context.Context, bucket, key string) (io.ReadCloser, error)
}

// Lister derives the published-baseline collection from canonical-bucket
// objects. Same construction shape as Publisher/Restorer.
type Lister struct {
	MC     objectStore
	Bucket string
	L      logrus.FieldLogger
}

// List enumerates baseline/regions/, keeps keys that parse as dump objects,
// reads each sha sidecar (degrading to "" on any failure), and returns the
// collection sorted by (region, major, minor) ascending so the response is
// deterministic. One bad key never fails the listing.
func (li Lister) List(ctx context.Context) ([]ListItemModel, error) {
	objs, err := li.MC.List(ctx, li.Bucket, baselinePrefix)
	if err != nil {
		return nil, err
	}
	items := make([]ListItemModel, 0)
	for _, o := range objs {
		if !strings.HasSuffix(o.Key, "/documents.dump") {
			continue
		}
		region, major, minor, ok := parseDumpKey(o.Key)
		if !ok {
			li.L.Warnf("baseline list: skipping unparseable key %s", o.Key)
			continue
		}
		items = append(items, ListItemModel{
			Id:           PublishOutputId(region, major, minor),
			Region:       region,
			MajorVersion: major,
			MinorVersion: minor,
			Sha256:       li.readSha(ctx, region, major, minor),
			PublishedAt:  o.LastModified.UTC().Format(time.RFC3339),
			SizeBytes:    o.Size,
		})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Region != items[j].Region {
			return items[i].Region < items[j].Region
		}
		if items[i].MajorVersion != items[j].MajorVersion {
			return items[i].MajorVersion < items[j].MajorVersion
		}
		return items[i].MinorVersion < items[j].MinorVersion
	})
	return items, nil
}

// readSha reads the .sha256 sidecar and expects exactly 64 hex characters.
// Any failure (missing object, read error, malformed content) logs a WARN
// and returns "" so a partially-published baseline is visible rather than
// hidden.
func (li Lister) readSha(ctx context.Context, region string, major, minor int) string {
	rc, err := li.MC.Get(ctx, li.Bucket, ShaKey(region, major, minor))
	if err != nil {
		li.L.Warnf("baseline list: sha sidecar unavailable for %s/%d.%d: %v", region, major, minor, err)
		return ""
	}
	defer rc.Close()
	b, err := io.ReadAll(rc)
	if err != nil {
		li.L.Warnf("baseline list: sha sidecar read failed for %s/%d.%d: %v", region, major, minor, err)
		return ""
	}
	sum := strings.TrimSpace(string(b))
	if raw, decErr := hex.DecodeString(sum); decErr != nil || len(raw) != 32 {
		li.L.Warnf("baseline list: sha sidecar malformed for %s/%d.%d", region, major, minor)
		return ""
	}
	return sum
}

// parseDumpKey — as implemented in Task 2 (unchanged).
```

(Keep the Task 2 `parseDumpKey` function at the bottom of the file; only the imports and the new declarations above it change.)

- [ ] **Step 5: Run the package tests**

```bash
go test -race ./baseline/ -v
```

Expected: PASS — all `TestParseDumpKey*`, all `TestLister*`, and every pre-existing test in the package.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-data/atlas.com/data/baseline/
git commit -m "feat(task-134): baseline Lister derives published-baseline collection from MinIO"
```

---

### Task 4: `GET /data/baselines` handler + route registration

**Files:**
- Modify: `services/atlas-data/atlas.com/data/baseline/handler.go`
- Modify: `services/atlas-data/atlas.com/data/baseline/handler_test.go`

**Interfaces:**
- Consumes: `Lister` from Task 3; `mc.Cfg().BucketCanonical` (existing `minio.Config` field); `rest.RegisterHandler` (the no-input-body variant, exactly as `wzinput/resource.go` uses); `server.MarshalResponse[[]ListItemModel]` (the slice marshaller, as `commodity/resource.go:43` uses).
- Produces: route `GET /data/baselines` (registered inside the existing `baseline.InitResource` — `main.go` line 164 already wires it; no main.go change) and `func listInner(mc *minio.Client) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc`.

- [ ] **Step 1: Write the failing gate tests**

Append to `services/atlas-data/atlas.com/data/baseline/handler_test.go`:

```go
func TestListNilMcReturns503(t *testing.T) {
	d, c := newDeps()
	h := listInner(nil)(&d, &c)
	req := httptest.NewRequest(http.MethodGet, "/api/data/baselines", nil)
	rr := httptest.NewRecorder()
	h(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rr.Code)
	}
}

// TestListRefusesNonOperator mirrors TestPublishRefusesNonOperator: the
// sentinel non-nil client bypasses the 503 gate; without X-Atlas-Operator: 1
// the handler must 403 BEFORE dereferencing the client (listInner only
// touches mc.Cfg() after the operator gate).
func TestListRefusesNonOperator(t *testing.T) {
	d, c := newDeps()
	mc := nonNilSentinelClient()
	h := listInner(mc)(&d, &c)
	req := httptest.NewRequest(http.MethodGet, "/api/data/baselines", nil)
	rr := httptest.NewRecorder()
	h(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rr.Code)
	}
}

func TestListItemModelJsonApiIdentity(t *testing.T) {
	var m ListItemModel
	if m.GetName() != "baselines" {
		t.Fatalf("GetName = %s", m.GetName())
	}
	if err := m.SetID("GMS/83.1"); err != nil {
		t.Fatalf("SetID: %v", err)
	}
	if m.GetID() != "GMS/83.1" {
		t.Fatalf("GetID = %s", m.GetID())
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./baseline/ -run 'TestList' -v
```

Expected: compile FAILURE — `undefined: listInner` (the Lister tests from Task 3 also match `TestList` and will run once it compiles; that's fine).

- [ ] **Step 3: Implement `listInner` and register the route**

In `services/atlas-data/atlas.com/data/baseline/handler.go`, update `InitResource` and add `listInner`:

```go
// InitResource installs POST /data/baseline/publish, POST /data/baseline/restore,
// and GET /data/baselines.
func InitResource(db *gorm.DB, mc *minio.Client) func(si jsonapi.ServerInformation) server.RouteInitializer {
	return func(si jsonapi.ServerInformation) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			r := router.PathPrefix("/data/baseline").Subrouter()
			r.HandleFunc("/publish", rest.RegisterInputHandler[PublishInputModel](l)(si)("baseline_publish", publishInner(db, mc, l))).Methods(http.MethodPost)
			r.HandleFunc("/restore", rest.RegisterInputHandler[RestoreInputModel](l)(si)("baseline_restore", restoreInner(db, mc, l))).Methods(http.MethodPost)
			// Plural collection route deliberately outside the /data/baseline
			// subrouter: GET /data/baselines lists published canonical baselines.
			router.HandleFunc("/data/baselines", rest.RegisterHandler(l)(si)("baselines_list", listInner(mc))).Methods(http.MethodGet)
		}
	}
}

// listInner serves GET /data/baselines. Gate order matches publishInner:
// nil-mc 503 first, then the operator 403, then the listing. The ParseTenant
// middleware runs on the route (all RegisterHandler routes get it) but the
// handler never reads the tenant — the nil-UUID synthetic tenant the UI sends
// is accepted and ignored.
func listInner(mc *minio.Client) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if mc == nil {
				http.Error(w, "minio unavailable", http.StatusServiceUnavailable)
				return
			}
			if r.Header.Get("X-Atlas-Operator") != "1" {
				http.Error(w, "operator required", http.StatusForbidden)
				return
			}
			items, err := (Lister{MC: mc, Bucket: mc.Cfg().BucketCanonical, L: d.Logger()}).List(r.Context())
			if err != nil {
				d.Logger().WithError(err).Errorf("baseline list failed")
				http.Error(w, fmt.Sprintf("list failed: %s", err.Error()), http.StatusInternalServerError)
				return
			}
			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			w.Header().Set("Content-Type", "application/vnd.api+json")
			server.MarshalResponse[[]ListItemModel](d.Logger())(w)(c.ServerInformation())(queryParams)(items)
		}
	}
}
```

No import changes are needed in `handler.go` (`fmt`, `net/http`, `rest`, `minio`, `server`, `mux`, `jsonapi`, `logrus`, `gorm` are all already imported).

- [ ] **Step 4: Run the full package tests**

```bash
go test -race ./baseline/ -v
```

Expected: PASS — new gate tests plus everything from Tasks 2–3 and all pre-existing tests.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-data/atlas.com/data/baseline/handler.go services/atlas-data/atlas.com/data/baseline/handler_test.go
git commit -m "feat(task-134): GET /data/baselines lists published canonical baselines"
```

---

### Task 5: atlas-data verification gates

**Files:** none created; verification only.

- [ ] **Step 1: Full Go gates**

Run from `services/atlas-data/atlas.com/data/`:

```bash
go test -race ./... && go vet ./... && go build ./...
```

Expected: all clean.

- [ ] **Step 2: Docker bake**

Run from the worktree root (`.worktrees/task-134-admin-nav-baselines`):

```bash
docker buildx bake atlas-data
```

Expected: image builds successfully. This is mandatory (CLAUDE.md) — `go build` cannot catch shared-Dockerfile gaps.

- [ ] **Step 3: Redis key guard**

Run from the worktree root:

```bash
tools/redis-key-guard.sh
```

Expected: clean (no raw keyed go-redis calls were added; this confirms it).

- [ ] **Step 4: Fix-and-rebuild if anything failed, then commit any fixes**

If all three steps were already clean, there is nothing to commit; move on.

---

### Task 6: `isDeploymentRoute` predicate

**Files:**
- Create: `services/atlas-ui/src/lib/deployment-routes.ts`
- Create: `services/atlas-ui/src/lib/__tests__/deployment-routes.test.ts`

**Interfaces:**
- Produces: `export const DEPLOYMENT_ROUTE_PREFIXES: readonly string[]` and `export function isDeploymentRoute(pathname: string): boolean` — consumed by the tenant switcher (Task 13), the scope banner (Task 14), and the sidebar sync test (Task 12).

- [ ] **Step 1: Write the failing test**

Create `services/atlas-ui/src/lib/__tests__/deployment-routes.test.ts`:

```ts
import { describe, it, expect } from 'vitest';
import { isDeploymentRoute } from '@/lib/deployment-routes';

describe('isDeploymentRoute', () => {
  it.each([
    '/templates',
    '/templates/abc123/writers',
    '/tenants',
    '/tenants/9f8e/properties',
    '/tenants/9f8e/character/presets',
    '/services',
    '/services/atlas-data',
    '/baselines',
  ])('returns true for deployment route %s', (path) => {
    expect(isDeploymentRoute(path)).toBe(true);
  });

  it.each([
    '/',
    '/setup',
    '/accounts',
    '/characters/42',
    '/servicesx',       // prefix guard: no false positive on sibling names
    '/templatesfoo',
    '/baselines-old',
  ])('returns false for non-deployment route %s', (path) => {
    expect(isDeploymentRoute(path)).toBe(false);
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run from `services/atlas-ui/`:

```bash
npm run test -- src/lib/__tests__/deployment-routes.test.ts
```

Expected: FAIL — cannot resolve `@/lib/deployment-routes`.

- [ ] **Step 3: Implement the predicate**

Create `services/atlas-ui/src/lib/deployment-routes.ts`:

```ts
/**
 * The single definition of "Deployment route" — pages whose changes affect
 * every tenant. The tenant switcher's inert state and the deployment scope
 * banner both consume this predicate; they can never disagree.
 */
export const DEPLOYMENT_ROUTE_PREFIXES = [
  '/templates',
  '/tenants',
  '/services',
  '/baselines',
] as const;

export function isDeploymentRoute(pathname: string): boolean {
  return DEPLOYMENT_ROUTE_PREFIXES.some(
    (prefix) => pathname === prefix || pathname.startsWith(prefix + '/'),
  );
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
npm run test -- src/lib/__tests__/deployment-routes.test.ts
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-ui/src/lib/deployment-routes.ts services/atlas-ui/src/lib/__tests__/deployment-routes.test.ts
git commit -m "feat(task-134): single isDeploymentRoute predicate"
```

---

### Task 7: `canonicalHeaders` helper + shared `formatBytes`

**Files:**
- Modify: `services/atlas-ui/src/lib/headers.tsx`
- Create: `services/atlas-ui/src/lib/format.ts`
- Create: `services/atlas-ui/src/lib/__tests__/headers.test.ts`
- Create: `services/atlas-ui/src/lib/__tests__/format.test.ts`
- Modify: `services/atlas-ui/src/pages/SetupPage.tsx` (import swap only)

**Interfaces:**
- Produces:
  - `export const CANONICAL_TENANT_ID = "00000000-0000-0000-0000-000000000000"`.
  - `export interface CanonicalSelection { region: string; majorVersion: number; minorVersion: number }`.
  - `export function canonicalHeaders(sel: CanonicalSelection): Headers` — every canonical service call (Tasks 8–9) builds headers through this and only this.
  - `export function formatBytes(bytes: number): string` in `@/lib/format` — used by SetupPage (now) and BaselinesPage (Task 16).

- [ ] **Step 1: Write the failing tests**

Create `services/atlas-ui/src/lib/__tests__/headers.test.ts`:

```ts
import { describe, it, expect } from 'vitest';
import { canonicalHeaders, CANONICAL_TENANT_ID } from '@/lib/headers';

describe('canonicalHeaders', () => {
  it('produces nil-UUID tenant, selection-derived version headers, and the operator header', () => {
    const headers = canonicalHeaders({ region: 'GMS', majorVersion: 83, minorVersion: 1 });
    expect(CANONICAL_TENANT_ID).toBe('00000000-0000-0000-0000-000000000000');
    expect(headers.get('TENANT_ID')).toBe(CANONICAL_TENANT_ID);
    expect(headers.get('REGION')).toBe('GMS');
    expect(headers.get('MAJOR_VERSION')).toBe('83');
    expect(headers.get('MINOR_VERSION')).toBe('1');
    expect(headers.get('X-Atlas-Operator')).toBe('1');
  });
});
```

Create `services/atlas-ui/src/lib/__tests__/format.test.ts`:

```ts
import { describe, it, expect } from 'vitest';
import { formatBytes } from '@/lib/format';

describe('formatBytes', () => {
  it('formats zero', () => {
    expect(formatBytes(0)).toBe('0 B');
  });
  it('formats bytes without decimals', () => {
    expect(formatBytes(512)).toBe('512 B');
  });
  it('formats small unit values with one decimal', () => {
    expect(formatBytes(1536)).toBe('1.5 KB');
  });
  it('formats values >= 10 without decimals', () => {
    expect(formatBytes(10 * 1024 * 1024)).toBe('10 MB');
  });
});
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
npm run test -- src/lib/__tests__/headers.test.ts src/lib/__tests__/format.test.ts
```

Expected: FAIL — `canonicalHeaders` not exported / `@/lib/format` unresolved.

- [ ] **Step 3: Implement**

Append to `services/atlas-ui/src/lib/headers.tsx`:

```tsx
/**
 * The synthetic tenant id used for canonical (deployment-wide) requests.
 * atlas-data's shared scope never reads the tenant id — ResolveScope gates
 * only on X-Atlas-Operator and the shared prefix is keyed by region/version —
 * but the shared REST middleware requires syntactically valid tenant headers,
 * and uuid.Parse accepts the nil UUID.
 */
export const CANONICAL_TENANT_ID = "00000000-0000-0000-0000-000000000000";

export interface CanonicalSelection {
    region: string;
    majorVersion: number;
    minorVersion: number;
}

/**
 * Headers for canonical-scope requests. Baking X-Atlas-Operator in here means
 * a canonical request cannot be assembled without the operator header — one
 * construction path, no drift.
 */
export function canonicalHeaders(sel: CanonicalSelection): Headers {
    const headers = new Headers();
    headers.set("TENANT_ID", CANONICAL_TENANT_ID);
    headers.set("REGION", sel.region);
    headers.set("MAJOR_VERSION", String(sel.majorVersion));
    headers.set("MINOR_VERSION", String(sel.minorVersion));
    headers.set("X-Atlas-Operator", "1");
    return headers;
}
```

Create `services/atlas-ui/src/lib/format.ts` (the function moves verbatim from `SetupPage.tsx`):

```ts
export function formatBytes(bytes: number): string {
  if (!bytes) return "0 B";
  const units = ["B", "KB", "MB", "GB", "TB"];
  let value = bytes;
  let unit = 0;
  while (value >= 1024 && unit < units.length - 1) {
    value /= 1024;
    unit++;
  }
  const formatted = new Intl.NumberFormat(undefined, {
    maximumFractionDigits: value >= 10 || unit === 0 ? 0 : 1,
  }).format(value);
  return `${formatted} ${units[unit]}`;
}
```

In `services/atlas-ui/src/pages/SetupPage.tsx`: delete the local `formatBytes` function (lines 49–62) and add to the imports:

```tsx
import { formatBytes } from "@/lib/format";
```

- [ ] **Step 4: Run tests + typecheck-level verification**

```bash
npm run test -- src/lib/__tests__/headers.test.ts src/lib/__tests__/format.test.ts && npx tsc -b --noEmit
```

Expected: PASS, no type errors.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-ui/src/lib/headers.tsx services/atlas-ui/src/lib/format.ts services/atlas-ui/src/lib/__tests__/headers.test.ts services/atlas-ui/src/lib/__tests__/format.test.ts services/atlas-ui/src/pages/SetupPage.tsx
git commit -m "feat(task-134): canonicalHeaders helper and shared formatBytes"
```

---

### Task 8: canonical service functions + `listBaselines`

**Files:**
- Modify: `services/atlas-ui/src/services/api/seed.service.ts`
- Modify: `services/atlas-ui/src/services/api/baseline.service.ts`
- Modify: `services/atlas-ui/src/services/api/__tests__/seed.service.test.ts`
- Modify: `services/atlas-ui/src/services/api/__tests__/baseline.service.test.ts`

**Interfaces:**
- Consumes: `canonicalHeaders`, `CanonicalSelection` from Task 7.
- Produces (all consumed by Task 11's hooks):
  - `seedService.uploadCanonicalWzFiles(sel: CanonicalSelection, file: File): Promise<void>`
  - `seedService.runCanonicalDataProcessing(sel: CanonicalSelection): Promise<void>`
  - `seedService.getCanonicalWzInputStatus(sel: CanonicalSelection): Promise<WzInputStatus>`
  - `seedService.getCanonicalDataStatus(sel: CanonicalSelection): Promise<DataStatus>`
  - `export interface Baseline { region: string; majorVersion: number; minorVersion: number; sha256: string; publishedAt: string; sizeBytes: number }`
  - `baselineService.listBaselines(): Promise<Baseline[]>`
- Tenant-facing signatures are **unchanged in this task** (they change in Task 10); the internals are refactored into shared private helpers so tenant and canonical paths cannot diverge.

- [ ] **Step 1: Write the failing tests**

Append to `services/atlas-ui/src/services/api/__tests__/seed.service.test.ts` (inside the existing `describe('seedService', ...)` if present, otherwise as a new top-level describe; reuse the file's existing fetch-stubbing pattern — `vi.stubGlobal('fetch', fetchMock)` in `beforeEach`, `vi.unstubAllGlobals()` in `afterEach`):

```ts
describe('canonical (shared-scope) functions', () => {
  const sel = { region: 'GMS', majorVersion: 83, minorVersion: 1 };
  const NIL_UUID = '00000000-0000-0000-0000-000000000000';

  it('uploadCanonicalWzFiles PATCHes scope=shared with synthetic canonical headers', async () => {
    fetchMock.mockResolvedValue({ ok: true, status: 202 });
    const file = new File(['zipbytes'], 'Data.zip', { type: 'application/zip' });
    await seedService.uploadCanonicalWzFiles(sel, file);
    expect(fetchMock).toHaveBeenCalledWith(
      '/api/data/wz?scope=shared',
      expect.objectContaining({ method: 'PATCH' }),
    );
    const headers = (fetchMock.mock.calls[0]![1] as RequestInit).headers as Headers;
    expect(headers.get('TENANT_ID')).toBe(NIL_UUID);
    expect(headers.get('REGION')).toBe('GMS');
    expect(headers.get('MAJOR_VERSION')).toBe('83');
    expect(headers.get('MINOR_VERSION')).toBe('1');
    expect(headers.get('X-Atlas-Operator')).toBe('1');
  });

  it('uploadCanonicalWzFiles surfaces status on the thrown error', async () => {
    fetchMock.mockResolvedValue({
      ok: false,
      status: 409,
      statusText: 'Conflict',
      json: async () => ({ error: 'busy' }),
    });
    const file = new File(['zipbytes'], 'Data.zip', { type: 'application/zip' });
    await expect(seedService.uploadCanonicalWzFiles(sel, file)).rejects.toMatchObject({
      message: 'busy',
      status: 409,
    });
  });

  it('runCanonicalDataProcessing POSTs scope=shared with canonical headers', async () => {
    fetchMock.mockResolvedValue({ ok: true, status: 202 });
    await seedService.runCanonicalDataProcessing(sel);
    expect(fetchMock).toHaveBeenCalledWith(
      '/api/data/process?scope=shared',
      expect.objectContaining({ method: 'POST' }),
    );
    const headers = (fetchMock.mock.calls[0]![1] as RequestInit).headers as Headers;
    expect(headers.get('TENANT_ID')).toBe(NIL_UUID);
    expect(headers.get('X-Atlas-Operator')).toBe('1');
  });

  it('getCanonicalWzInputStatus GETs scope=shared and unwraps attributes', async () => {
    fetchMock.mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({
        data: { type: 'wzInputStatus', id: 'current', attributes: { fileCount: 3, totalBytes: 999, updatedAt: null } },
      }),
    });
    const status = await seedService.getCanonicalWzInputStatus(sel);
    expect(fetchMock).toHaveBeenCalledWith('/api/data/wz?scope=shared', expect.objectContaining({ method: 'GET' }));
    expect(status.fileCount).toBe(3);
    const headers = (fetchMock.mock.calls[0]![1] as RequestInit).headers as Headers;
    expect(headers.get('TENANT_ID')).toBe(NIL_UUID);
  });

  it('getCanonicalDataStatus GETs scope=shared and unwraps attributes', async () => {
    fetchMock.mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({
        data: {
          type: 'dataStatus',
          id: 'current',
          attributes: { documentCount: 42, updatedAt: null, baselineRestoredAt: null, baselineSha256: null },
        },
      }),
    });
    const status = await seedService.getCanonicalDataStatus(sel);
    expect(fetchMock).toHaveBeenCalledWith('/api/data/status?scope=shared', expect.objectContaining({ method: 'GET' }));
    expect(status.documentCount).toBe(42);
  });
});
```

(If the existing test file scopes `fetchMock` inside a `describe`, hoist a new `fetchMock` for this block using the same `beforeEach`/`afterEach` pattern the file already uses.)

Append to `services/atlas-ui/src/services/api/__tests__/baseline.service.test.ts`:

```ts
describe('listBaselines', () => {
  it('GETs /api/data/baselines with canonical dummy headers and decodes the collection', async () => {
    fetchMock.mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({
        data: [
          {
            type: 'baselines',
            id: 'GMS/83.1',
            attributes: {
              region: 'GMS',
              majorVersion: 83,
              minorVersion: 1,
              sha256: 'a'.repeat(64),
              publishedAt: '2026-07-04T12:34:56Z',
              sizeBytes: 123456789,
            },
          },
        ],
      }),
    });
    const baselines = await baselineService.listBaselines();
    expect(fetchMock).toHaveBeenCalledWith('/api/data/baselines', expect.objectContaining({ method: 'GET' }));
    const headers = (fetchMock.mock.calls[0]![1] as RequestInit).headers as Headers;
    expect(headers.get('TENANT_ID')).toBe('00000000-0000-0000-0000-000000000000');
    expect(headers.get('X-Atlas-Operator')).toBe('1');
    expect(baselines).toEqual([
      {
        region: 'GMS',
        majorVersion: 83,
        minorVersion: 1,
        sha256: 'a'.repeat(64),
        publishedAt: '2026-07-04T12:34:56Z',
        sizeBytes: 123456789,
      },
    ]);
  });

  it('throws with the decoded server message on failure', async () => {
    fetchMock.mockResolvedValue({
      ok: false,
      status: 503,
      json: async () => ({ error: 'minio unavailable' }),
    });
    await expect(baselineService.listBaselines()).rejects.toThrow('minio unavailable');
  });
});
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
npm run test -- src/services/api/__tests__/seed.service.test.ts src/services/api/__tests__/baseline.service.test.ts
```

Expected: FAIL — `uploadCanonicalWzFiles` / `listBaselines` are not functions.

- [ ] **Step 3: Implement in `seed.service.ts`**

In `services/atlas-ui/src/services/api/seed.service.ts`:

1. Extend the imports:

```ts
import { tenantHeaders, canonicalHeaders, type CanonicalSelection } from '@/lib/headers';
```

2. Change `fetchJsonApi` to take pre-built headers (behavior-preserving refactor):

```ts
async function fetchJsonApi<A>(url: string, headers: Headers): Promise<A> {
  headers.set('Accept', 'application/vnd.api+json');
  const response = await fetch(url, { method: 'GET', headers });
  if (!response.ok) {
    throw new Error(`GET ${url} failed: ${response.status} ${response.statusText}`);
  }
  const body = (await response.json()) as JsonApiEnvelope<A>;
  return body.data.attributes;
}
```

3. Extract the WZ-upload and process bodies into private helpers (above the class):

```ts
async function patchWzZip(url: string, headers: Headers, file: File): Promise<void> {
  const formData = new FormData();
  formData.append('zip_file', file);

  const response = await fetch(url, { method: 'PATCH', headers, body: formData });

  if (!response.ok) {
    let message = `Upload failed: ${response.status} ${response.statusText}`;
    try {
      const body = (await response.json()) as { error?: string };
      if (body.error) {
        message = body.error;
      }
    } catch {
      // non-JSON error body; fall back to status text
    }
    const err = new Error(message) as Error & { status?: number };
    err.status = response.status;
    throw err;
  }
}

async function postProcess(url: string, headers: Headers): Promise<void> {
  const response = await fetch(url, { method: 'POST', headers });
  if (!response.ok) {
    throw new Error(`Data processing failed: ${response.status} ${response.statusText}`);
  }
}
```

4. Rewrite the four scope-touching methods to delegate, keeping tenant signatures intact for now (Task 10 removes the `scope` parameters):

```ts
  async uploadWzFiles(tenant: Tenant, file: File, scope: 'tenant' | 'shared' = 'tenant'): Promise<void> {
    const headers = tenantHeaders(tenant);
    if (scope === 'shared') {
      headers.set('X-Atlas-Operator', '1');
    }
    return patchWzZip(`/api/data/wz?scope=${scope}`, headers, file);
  }

  async runDataProcessing(tenant: Tenant, scope: 'tenant' | 'shared' = 'tenant'): Promise<void> {
    const headers = tenantHeaders(tenant);
    if (scope === 'shared') {
      headers.set('X-Atlas-Operator', '1');
    }
    return postProcess(`/api/data/process?scope=${scope}`, headers);
  }

  async getWzInputStatus(tenant: Tenant, scope: 'tenant' | 'shared' = 'tenant'): Promise<WzInputStatus> {
    const headers = tenantHeaders(tenant);
    if (scope === 'shared') {
      headers.set('X-Atlas-Operator', '1');
    }
    return fetchJsonApi<WzInputStatus>(`/api/data/wz?scope=${scope}`, headers);
  }

  async getDataStatus(tenant: Tenant, scope: 'tenant' | 'shared' = 'tenant'): Promise<DataStatus> {
    const headers = tenantHeaders(tenant);
    if (scope === 'shared') {
      headers.set('X-Atlas-Operator', '1');
    }
    return fetchJsonApi<DataStatus>(`/api/data/status?scope=${scope}`, headers);
  }

  // Canonical (deployment-wide) variants: no Tenant anywhere — headers are
  // synthesized from the explicit region/version selection. This is what lets
  // an operator publish canonical data for a version with no live tenant.
  async uploadCanonicalWzFiles(sel: CanonicalSelection, file: File): Promise<void> {
    return patchWzZip('/api/data/wz?scope=shared', canonicalHeaders(sel), file);
  }

  async runCanonicalDataProcessing(sel: CanonicalSelection): Promise<void> {
    return postProcess('/api/data/process?scope=shared', canonicalHeaders(sel));
  }

  async getCanonicalWzInputStatus(sel: CanonicalSelection): Promise<WzInputStatus> {
    return fetchJsonApi<WzInputStatus>('/api/data/wz?scope=shared', canonicalHeaders(sel));
  }

  async getCanonicalDataStatus(sel: CanonicalSelection): Promise<DataStatus> {
    return fetchJsonApi<DataStatus>('/api/data/status?scope=shared', canonicalHeaders(sel));
  }
```

(Also update `fetchSeedStatus`'s callers — it is untouched; only `fetchJsonApi` changed signature, and its two callers are rewritten above.)

- [ ] **Step 4: Implement in `baseline.service.ts`**

In `services/atlas-ui/src/services/api/baseline.service.ts`, extend the imports and add the type + method (leave `restore` and `publish` untouched in this task):

```ts
import { tenantHeaders, canonicalHeaders, type CanonicalSelection } from '@/lib/headers';
```

```ts
export interface Baseline {
  region: string;
  majorVersion: number;
  minorVersion: number;
  sha256: string;
  publishedAt: string; // RFC3339
  sizeBytes: number;
}

interface JsonApiCollection<A> {
  data: Array<{ type: string; id: string; attributes: A }>;
}

// GET /data/baselines needs tenant headers only to clear the shared REST
// middleware; the server ignores their values. A fixed dummy selection keeps
// the call signature tenant-free.
const LIST_HEADER_SELECTION: CanonicalSelection = { region: 'NONE', majorVersion: 0, minorVersion: 0 };
```

Inside `BaselineService`:

```ts
  async listBaselines(): Promise<Baseline[]> {
    const headers = canonicalHeaders(LIST_HEADER_SELECTION);
    headers.set('Accept', 'application/vnd.api+json');
    const r = await fetch('/api/data/baselines', { method: 'GET', headers });
    if (!r.ok) {
      const message = await decodeErrorMessage(r, `baselines list failed: ${r.status}`);
      throw new Error(message);
    }
    const body = (await r.json()) as JsonApiCollection<Baseline>;
    return body.data.map((d) => ({ ...d.attributes }));
  }
```

- [ ] **Step 5: Run the service tests**

```bash
npm run test -- src/services/api/__tests__/seed.service.test.ts src/services/api/__tests__/baseline.service.test.ts && npx tsc -b --noEmit
```

Expected: PASS (new tests plus all pre-existing service tests — the tenant-path refactor must not change any existing assertion).

- [ ] **Step 6: Commit**

```bash
git add services/atlas-ui/src/services/api/seed.service.ts services/atlas-ui/src/services/api/baseline.service.ts services/atlas-ui/src/services/api/__tests__/
git commit -m "feat(task-134): canonical service functions and listBaselines"
```

---

### Task 9: migrate `publish` off the Tenant

**Files:**
- Modify: `services/atlas-ui/src/services/api/baseline.service.ts`
- Modify: `services/atlas-ui/src/services/api/__tests__/baseline.service.test.ts`
- Modify: `services/atlas-ui/src/lib/hooks/api/useBaseline.ts`
- Modify: `services/atlas-ui/src/pages/SetupPage.tsx`

**Interfaces:**
- Produces: `baselineService.publish(sel: CanonicalSelection): Promise<void>` — Task 11's `usePublishCanonicalBaseline` consumes it. `usePublishBaseline` is **deleted**; `useRestoreBaseline` is unchanged.
- The Setup page loses its publish row (PRD FR-4.3) in the same commit so every intermediate state compiles.

- [ ] **Step 1: Update the publish tests to the new signature**

In `services/atlas-ui/src/services/api/__tests__/baseline.service.test.ts`, replace the existing `publish` describe-block assertions: `publish` is now called as `baselineService.publish({ region: 'GMS', majorVersion: 83, minorVersion: 1 })` and the header assertions become:

```ts
      expect(headers.get('TENANT_ID')).toBe('00000000-0000-0000-0000-000000000000');
      expect(headers.get('REGION')).toBe('GMS');
      expect(headers.get('MAJOR_VERSION')).toBe('83');
      expect(headers.get('MINOR_VERSION')).toBe('1');
      expect(headers.get('X-Atlas-Operator')).toBe('1');
      expect(headers.get('Content-Type')).toBe('application/json');
```

The body assertion (JSON:API envelope with type `baselinePublishes` and attributes `{ region, majorVersion, minorVersion }`) stays as-is. Keep any publish error-path test, updating only the call signature.

- [ ] **Step 2: Run tests to verify they fail**

```bash
npm run test -- src/services/api/__tests__/baseline.service.test.ts
```

Expected: FAIL — publish still takes `(tenant, region, major, minor)`.

- [ ] **Step 3: Change `publish` and delete `usePublishBaseline`**

In `services/atlas-ui/src/services/api/baseline.service.ts` replace the `publish` method:

```ts
  // publish was always a shared-scope operation; the former Tenant argument
  // only fed headers. It now takes the explicit canonical selection.
  async publish(sel: CanonicalSelection): Promise<void> {
    const headers = canonicalHeaders(sel);
    headers.set('Content-Type', 'application/json');
    const r = await fetch('/api/data/baseline/publish', {
      method: 'POST',
      headers,
      body: JSON.stringify({
        data: {
          type: 'baselinePublishes',
          attributes: {
            region: sel.region,
            majorVersion: sel.majorVersion,
            minorVersion: sel.minorVersion,
          },
        },
      }),
    });
    if (!r.ok) {
      const message = await decodeErrorMessage(r, `publish failed: ${r.status}`);
      throw new Error(message);
    }
  }
```

In `services/atlas-ui/src/lib/hooks/api/useBaseline.ts`: delete the entire `usePublishBaseline` export (keep `useRestoreBaseline` byte-for-byte).

In `services/atlas-ui/src/pages/SetupPage.tsx` remove the publish surface:
- Import line: `import { useRestoreBaseline, usePublishBaseline } from "@/lib/hooks/api/useBaseline";` → `import { useRestoreBaseline } from "@/lib/hooks/api/useBaseline";`
- Remove `Send` from the lucide-react import list.
- Delete `const publishMutation = usePublishBaseline(activeTenant);`
- Delete the whole `handlePublishBaseline` function.
- Delete the `showPublishRow` const and the entire `{showPublishRow && ( <SetupRow ... label="Publish Canonical Baseline" ... /> )}` JSX block.

- [ ] **Step 4: Run tests + typecheck**

```bash
npm run test -- src/services/api/__tests__/baseline.service.test.ts && npx tsc -b --noEmit
```

Expected: PASS, no type errors (nothing else referenced `usePublishBaseline`).

- [ ] **Step 5: Commit**

```bash
git add services/atlas-ui/src/services/api/baseline.service.ts services/atlas-ui/src/services/api/__tests__/baseline.service.test.ts services/atlas-ui/src/lib/hooks/api/useBaseline.ts services/atlas-ui/src/pages/SetupPage.tsx
git commit -m "feat(task-134): publish takes CanonicalSelection; Setup page loses publish row"
```

---

### Task 10: de-scope the tenant path (Setup page, seed service, seed hooks)

**Files:**
- Modify: `services/atlas-ui/src/services/api/seed.service.ts`
- Modify: `services/atlas-ui/src/lib/hooks/api/useSeed.ts`
- Modify: `services/atlas-ui/src/pages/SetupPage.tsx`
- Delete: `services/atlas-ui/src/components/features/setup/ScopeToggle.tsx`
- Delete: `services/atlas-ui/src/components/features/setup/__tests__/ScopeToggle.test.tsx`
- Modify: `services/atlas-ui/src/services/api/__tests__/seed.service.test.ts`
- Create: `services/atlas-ui/src/pages/__tests__/SetupPage.test.tsx`

**Interfaces:**
- Produces (consumed by SetupPage and Task 11):
  - `seedService.uploadWzFiles(tenant: Tenant, file: File): Promise<void>` — always `scope=tenant`, never sends `X-Atlas-Operator`.
  - `seedService.runDataProcessing(tenant: Tenant): Promise<void>`
  - `seedService.getWzInputStatus(tenant: Tenant): Promise<WzInputStatus>`
  - `seedService.getDataStatus(tenant: Tenant): Promise<DataStatus>`
  - `useSeed.ts`: `useWzInputStatus(): UseQueryResult<WzInputStatus, Error>` and `useDataStatus(): UseQueryResult<DataStatus, Error>` (no parameters); `useUploadWzFiles(): UseMutationResult<void, Error, { file: File }>`; `useRunDataProcessing(): UseMutationResult<void, Error, void>`; and `export function showWzUploadErrorToast(error: Error): void` (shared with Task 11's canonical upload hook).
- After this task the tenant path is **incapable** of issuing `scope=shared` requests (PRD §8 capability removal).

- [ ] **Step 1: Update seed.service tests for the de-scoped tenant path**

In `services/atlas-ui/src/services/api/__tests__/seed.service.test.ts`:
- Delete every test that passes `'shared'` to `uploadWzFiles` / `runDataProcessing` / `getWzInputStatus` / `getDataStatus` (the canonical variants from Task 8 cover shared scope now).
- Update remaining tenant-path tests to the two-argument/one-argument signatures and add these assertions to one tenant-path test per method group:

```ts
    // Tenant path is structurally incapable of shared scope.
    expect(fetchMock.mock.calls[0]![0]).toContain('scope=tenant');
    const headers = (fetchMock.mock.calls[0]![1] as RequestInit).headers as Headers;
    expect(headers.get('X-Atlas-Operator')).toBeNull();
```

- [ ] **Step 2: Write the failing SetupPage test**

Create `services/atlas-ui/src/pages/__tests__/SetupPage.test.tsx`:

```tsx
import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import { SetupPage } from '@/pages/SetupPage';

const mockTenant = {
  id: '11111111-1111-1111-1111-111111111111',
  attributes: { name: 'Test Tenant', region: 'GMS', majorVersion: 83, minorVersion: 1 },
};

const idleMutation = { mutate: vi.fn(), isPending: false };
const emptyStatus = { data: undefined };

// Mutable per-test data-status so individual tests can flip documentCount.
let dataStatusData: { documentCount: number; updatedAt: string | null; baselineRestoredAt: string | null; baselineSha256: string | null } = {
  documentCount: 0,
  updatedAt: null,
  baselineRestoredAt: null,
  baselineSha256: null,
};

vi.mock('@/context/tenant-context', () => ({
  useTenant: () => ({ activeTenant: mockTenant }),
}));

vi.mock('@/lib/hooks/api/useBaseline', () => ({
  useRestoreBaseline: () => idleMutation,
}));

vi.mock('@/lib/hooks/api/useSeed', () => ({
  useSeedDrops: () => idleMutation,
  useSeedGachapons: () => idleMutation,
  useSeedNpcConversations: () => idleMutation,
  useSeedQuestConversations: () => idleMutation,
  useSeedNpcShops: () => idleMutation,
  useSeedPortalScripts: () => idleMutation,
  useSeedReactorScripts: () => idleMutation,
  useSeedMapActionScripts: () => idleMutation,
  useUploadWzFiles: () => idleMutation,
  useRunDataProcessing: () => idleMutation,
  useWzInputStatus: () => ({ data: { fileCount: 2, totalBytes: 1024, updatedAt: null } }),
  useDataStatus: () => ({ data: dataStatusData }),
  useDropsSeedStatus: () => emptyStatus,
  useGachaponsSeedStatus: () => emptyStatus,
  useNpcConversationsSeedStatus: () => emptyStatus,
  useQuestConversationsSeedStatus: () => emptyStatus,
  useNpcShopsSeedStatus: () => emptyStatus,
  usePortalScriptsSeedStatus: () => emptyStatus,
  useReactorScriptsSeedStatus: () => emptyStatus,
  useMapActionScriptsSeedStatus: () => emptyStatus,
  showWzUploadErrorToast: vi.fn(),
}));

describe('SetupPage (tenant-only)', () => {
  it('is titled Setup and has no scope toggle and no publish row', () => {
    render(<SetupPage />);
    expect(screen.getByRole('heading', { name: 'Setup' })).toBeInTheDocument();
    expect(screen.queryByTestId('scope-toggle')).not.toBeInTheDocument();
    expect(screen.queryByText(/Publish Canonical Baseline/i)).not.toBeInTheDocument();
  });

  it('shows the restore row when the tenant document count is 0', () => {
    dataStatusData = { documentCount: 0, updatedAt: null, baselineRestoredAt: null, baselineSha256: null };
    render(<SetupPage />);
    expect(screen.getByText(/Restore Canonical Baseline/i)).toBeInTheDocument();
  });

  it('hides the restore row when documents exist', () => {
    dataStatusData = { documentCount: 5, updatedAt: null, baselineRestoredAt: null, baselineSha256: null };
    render(<SetupPage />);
    expect(screen.queryByText(/Restore Canonical Baseline/i)).not.toBeInTheDocument();
  });

  it('renders all eight seed rows', () => {
    render(<SetupPage />);
    for (const label of [
      'Monster & Reactor Drops',
      'Gachapons',
      'NPC Conversations',
      'Quest Conversations',
      'NPC Shops',
      'Portal Scripts',
      'Reactor Scripts',
      'Map Action Scripts',
    ]) {
      expect(screen.getByText(label)).toBeInTheDocument();
    }
  });
});
```

- [ ] **Step 3: Run tests to verify they fail**

```bash
npm run test -- src/pages/__tests__/SetupPage.test.tsx src/services/api/__tests__/seed.service.test.ts
```

Expected: FAIL — heading is still "Bootstrap", scope toggle still renders, seed.service still requires/propagates `scope`, `showWzUploadErrorToast` doesn't exist.

- [ ] **Step 4: Implement the de-scope**

`services/atlas-ui/src/services/api/seed.service.ts` — the four tenant methods become:

```ts
  async uploadWzFiles(tenant: Tenant, file: File): Promise<void> {
    return patchWzZip('/api/data/wz?scope=tenant', tenantHeaders(tenant), file);
  }

  async runDataProcessing(tenant: Tenant): Promise<void> {
    return postProcess('/api/data/process?scope=tenant', tenantHeaders(tenant));
  }

  async getWzInputStatus(tenant: Tenant): Promise<WzInputStatus> {
    return fetchJsonApi<WzInputStatus>('/api/data/wz?scope=tenant', tenantHeaders(tenant));
  }

  async getDataStatus(tenant: Tenant): Promise<DataStatus> {
    return fetchJsonApi<DataStatus>('/api/data/status?scope=tenant', tenantHeaders(tenant));
  }
```

`services/atlas-ui/src/lib/hooks/api/useSeed.ts`:
- Remove `import type { Scope } from '@/components/features/setup/ScopeToggle';`
- Add `import { toast } from 'sonner';`
- Add the shared error helper (exported — Task 11's canonical upload hook reuses it):

```ts
// Shared WZ-upload error toast: one copy of the 409/400 wording for the
// tenant (Setup) and canonical (Baselines) upload paths.
export function showWzUploadErrorToast(error: Error): void {
  const err = error as Error & { status?: number };
  if (err.status === 409) {
    toast.error('Another upload or processing job is in progress for this scope. Try again in a moment.');
  } else if (err.status === 400) {
    toast.error(`Upload rejected: ${err.message}`);
  } else {
    toast.error(`Upload failed: ${err.message}`);
  }
}
```

- `UploadWzFilesInput` becomes `{ file: File }`; `useUploadWzFiles` gains hook-level `onError`:

```ts
export interface UploadWzFilesInput {
  file: File;
}

export function useUploadWzFiles(): UseMutationResult<void, Error, UploadWzFilesInput> {
  const { activeTenant } = useTenant();
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ file }: UploadWzFilesInput) => seedService.uploadWzFiles(activeTenant!, file),
    onSuccess: () => {
      if (!activeTenant) return;
      void queryClient.invalidateQueries({ queryKey: wzInputStatusKey(activeTenant.id) });
    },
    onError: showWzUploadErrorToast,
  });
}
```

- `useRunDataProcessing` drops its variable:

```ts
export function useRunDataProcessing(): UseMutationResult<void, Error, void> {
  const { activeTenant } = useTenant();
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () => seedService.runDataProcessing(activeTenant!),
    onSuccess: () => {
      if (!activeTenant) return;
      void queryClient.invalidateQueries({ queryKey: dataStatusKey(activeTenant.id) });
    },
  });
}
```

- `useWzInputStatus` / `useDataStatus` lose the parameter and the third key segment (delete the scope-key comment above them):

```ts
export function useWzInputStatus(): UseQueryResult<WzInputStatus, Error> {
  const { activeTenant } = useTenant();
  return useQuery({
    queryKey: activeTenant ? wzInputStatusKey(activeTenant.id) : ['wzInputStatus', 'none'],
    queryFn: () => seedService.getWzInputStatus(activeTenant!),
    enabled: !!activeTenant,
    staleTime: 0,
    refetchInterval: 5000,
  });
}

export function useDataStatus(): UseQueryResult<DataStatus, Error> {
  const { activeTenant } = useTenant();
  return useQuery({
    queryKey: activeTenant ? dataStatusKey(activeTenant.id) : ['dataStatus', 'none'],
    queryFn: () => seedService.getDataStatus(activeTenant!),
    enabled: !!activeTenant,
    staleTime: 0,
    refetchInterval: 5000,
  });
}
```

`services/atlas-ui/src/pages/SetupPage.tsx`:
- Remove `import { ScopeToggle, type Scope } from "@/components/features/setup/ScopeToggle";` and the `useState` scope state (`const [scope, setScope] = useState<Scope>('tenant');`); drop `useState` from the react import if now unused (keep `useRef`).
- `useWzInputStatus(scope)` → `useWzInputStatus()`; `useDataStatus(scope)` → `useDataStatus()`.
- In `handleFileUpload`: `uploadWz.mutate({ file, scope }, {...})` becomes:

```tsx
    uploadWz.mutate({ file }, {
      onSuccess: () => {
        toast.success(`WZ files uploaded (${formatBytes(size)})`);
      },
    });
```

(the `onError` branch is deleted — the hook now owns it).
- In `handleRunProcessing`: `runProcessing.mutate(scope, {...})` → `runProcessing.mutate(undefined, {...})` with the same success/error toasts.
- Delete the `tenantRegion`/`tenantVersion` consts (only ScopeToggle used them).
- Delete the `<div className="mb-4"><ScopeToggle .../></div>` block.
- Retitle (FR-4.4):

```tsx
        <h2 className="text-2xl font-bold tracking-tight">Setup</h2>
        <p className="text-muted-foreground">Prepare the selected tenant&apos;s game data and seeded services.</p>
```

- Game Data card description drops the scope sentence:

```tsx
          <CardDescription>
            Upload a WZ zip and process it into atlas-data for the selected tenant.
          </CardDescription>
```

Delete the two ScopeToggle files:

```bash
git rm services/atlas-ui/src/components/features/setup/ScopeToggle.tsx services/atlas-ui/src/components/features/setup/__tests__/ScopeToggle.test.tsx
```

- [ ] **Step 5: Run tests + typecheck**

```bash
npm run test -- src/pages/__tests__/SetupPage.test.tsx src/services/api/__tests__/seed.service.test.ts && npx tsc -b --noEmit
```

Expected: PASS, no type errors, no dangling ScopeToggle references (`grep -rn "ScopeToggle" src/` returns nothing).

- [ ] **Step 6: Commit**

```bash
git add -A services/atlas-ui/src
git commit -m "feat(task-134): Setup page is tenant-only; scope toggle removed"
```

---

### Task 11: canonical React Query hooks (`useCanonicalData.ts`)

**Files:**
- Create: `services/atlas-ui/src/lib/hooks/api/useCanonicalData.ts`

**Interfaces:**
- Consumes: Task 8/9 service functions; `showWzUploadErrorToast` from Task 10; `CanonicalSelection` from Task 7.
- Produces (all consumed by Tasks 15–16):
  - `useCanonicalWzInputStatus(sel: CanonicalSelection | null): UseQueryResult<WzInputStatus, Error>`
  - `useCanonicalDataStatus(sel: CanonicalSelection | null): UseQueryResult<DataStatus, Error>`
  - `useUploadCanonicalWz(sel: CanonicalSelection | null): UseMutationResult<void, Error, File>`
  - `useRunCanonicalProcessing(sel: CanonicalSelection | null): UseMutationResult<void, Error, void>`
  - `useBaselines(): UseQueryResult<Baseline[], Error>`
  - `usePublishCanonicalBaseline(sel: CanonicalSelection | null): UseMutationResult<void, Error, void>`
  - `export const baselinesKey = ['baselines'] as const`

These hooks are exercised through the BaselinesPage component tests (Task 16); no separate hook-harness test file.

- [ ] **Step 1: Implement the hook module**

Create `services/atlas-ui/src/lib/hooks/api/useCanonicalData.ts`:

```ts
import {
  useMutation,
  useQuery,
  useQueryClient,
  type UseMutationResult,
  type UseQueryResult,
} from '@tanstack/react-query';
import { seedService, type DataStatus, type WzInputStatus } from '@/services/api/seed.service';
import { baselineService, type Baseline } from '@/services/api/baseline.service';
import type { CanonicalSelection } from '@/lib/headers';
import { showWzUploadErrorToast } from '@/lib/hooks/api/useSeed';

const canonicalWzInputKey = (sel: CanonicalSelection) =>
  ['canonical', 'wzInput', sel.region, sel.majorVersion, sel.minorVersion] as const;
const canonicalDataStatusKey = (sel: CanonicalSelection) =>
  ['canonical', 'dataStatus', sel.region, sel.majorVersion, sel.minorVersion] as const;
export const baselinesKey = ['baselines'] as const;

export function useCanonicalWzInputStatus(sel: CanonicalSelection | null): UseQueryResult<WzInputStatus, Error> {
  return useQuery({
    queryKey: sel ? canonicalWzInputKey(sel) : ['canonical', 'wzInput', 'none'],
    queryFn: () => seedService.getCanonicalWzInputStatus(sel!),
    enabled: !!sel,
    staleTime: 0,
    refetchInterval: 5000,
  });
}

export function useCanonicalDataStatus(sel: CanonicalSelection | null): UseQueryResult<DataStatus, Error> {
  return useQuery({
    queryKey: sel ? canonicalDataStatusKey(sel) : ['canonical', 'dataStatus', 'none'],
    queryFn: () => seedService.getCanonicalDataStatus(sel!),
    enabled: !!sel,
    staleTime: 0,
    refetchInterval: 5000,
  });
}

export function useUploadCanonicalWz(sel: CanonicalSelection | null): UseMutationResult<void, Error, File> {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (file: File) => {
      if (!sel) {
        throw new Error('useUploadCanonicalWz: no region/version selected');
      }
      return seedService.uploadCanonicalWzFiles(sel, file);
    },
    onSuccess: () => {
      if (!sel) return;
      void queryClient.invalidateQueries({ queryKey: canonicalWzInputKey(sel) });
    },
    onError: showWzUploadErrorToast,
  });
}

export function useRunCanonicalProcessing(sel: CanonicalSelection | null): UseMutationResult<void, Error, void> {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () => {
      if (!sel) {
        throw new Error('useRunCanonicalProcessing: no region/version selected');
      }
      return seedService.runCanonicalDataProcessing(sel);
    },
    onSuccess: () => {
      if (!sel) return;
      void queryClient.invalidateQueries({ queryKey: canonicalDataStatusKey(sel) });
    },
  });
}

export function useBaselines(): UseQueryResult<Baseline[], Error> {
  return useQuery({
    queryKey: baselinesKey,
    queryFn: () => baselineService.listBaselines(),
  });
}

export function usePublishCanonicalBaseline(sel: CanonicalSelection | null): UseMutationResult<void, Error, void> {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () => {
      if (!sel) {
        throw new Error('usePublishCanonicalBaseline: no region/version selected');
      }
      return baselineService.publish(sel);
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: baselinesKey });
      if (!sel) return;
      void queryClient.invalidateQueries({ queryKey: canonicalDataStatusKey(sel) });
    },
  });
}
```

- [ ] **Step 2: Typecheck**

```bash
npx tsc -b --noEmit
```

Expected: clean.

- [ ] **Step 3: Commit**

```bash
git add services/atlas-ui/src/lib/hooks/api/useCanonicalData.ts
git commit -m "feat(task-134): canonical data React Query hooks"
```

---

### Task 12: sidebar regroup + Deployment treatment + breadcrumb rename

**Files:**
- Modify: `services/atlas-ui/src/components/app-sidebar.tsx`
- Modify: `services/atlas-ui/src/lib/breadcrumbs/routes.ts`
- Create: `services/atlas-ui/src/components/__tests__/app-sidebar.test.tsx`

**Interfaces:**
- Consumes: `isDeploymentRoute` (sync test only).
- Produces: `export const sidebarItems` (exported so the sync test can assert nav/predicate agreement). Groups top-to-bottom: Operations, Security, Setup, Deployment; Deployment children in order Templates, Tenants, Services, Baselines; Deployment has `separated: true` and `caption: "Applies to all tenants"`.

- [ ] **Step 1: Write the failing test**

Create `services/atlas-ui/src/components/__tests__/app-sidebar.test.tsx`:

```tsx
import { describe, it, expect, vi, beforeAll } from 'vitest';
import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { AppSidebar, sidebarItems } from '@/components/app-sidebar';
import { SidebarProvider } from '@/components/ui/sidebar';
import { isDeploymentRoute } from '@/lib/deployment-routes';

vi.mock('@/components/app-tenant-switcher', () => ({
  TenantSwitcher: () => <div data-testid="tenant-switcher-stub" />,
}));

beforeAll(() => {
  // SidebarProvider's mobile detection needs matchMedia, absent in jsdom.
  Object.defineProperty(window, 'matchMedia', {
    writable: true,
    value: vi.fn().mockImplementation((query: string) => ({
      matches: false,
      media: query,
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(),
    })),
  });
});

function renderSidebar(initialPath = '/') {
  return render(
    <MemoryRouter initialEntries={[initialPath]}>
      <SidebarProvider>
        <AppSidebar />
      </SidebarProvider>
    </MemoryRouter>,
  );
}

describe('AppSidebar', () => {
  it('declares groups in blast-radius order with the Deployment children ordered', () => {
    expect(sidebarItems.map((g) => g.title)).toEqual(['Operations', 'Security', 'Setup', 'Deployment']);
    const deployment = sidebarItems[3]!;
    expect(deployment.children.map((c) => c.title)).toEqual(['Templates', 'Tenants', 'Services', 'Baselines']);
    expect(deployment.separated).toBe(true);
    expect(deployment.caption).toBe('Applies to all tenants');
    const setup = sidebarItems[2]!;
    expect(setup.children).toEqual([{ title: 'Setup', url: '/setup' }]);
  });

  it('keeps the sidebar and the route predicate in sync', () => {
    const deployment = sidebarItems.find((g) => g.title === 'Deployment')!;
    for (const child of deployment.children) {
      expect(isDeploymentRoute(child.url), `${child.url} must be a Deployment route`).toBe(true);
    }
    for (const group of sidebarItems.filter((g) => g.title !== 'Deployment')) {
      for (const child of group.children) {
        expect(isDeploymentRoute(child.url), `${child.url} must NOT be a Deployment route`).toBe(false);
      }
    }
  });

  it('renders the Deployment caption', () => {
    renderSidebar();
    expect(screen.getByText('Applies to all tenants')).toBeInTheDocument();
  });

  it('renders the Baselines link', () => {
    renderSidebar('/baselines');
    expect(screen.getByRole('link', { name: 'Baselines' })).toHaveAttribute('href', '/baselines');
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

```bash
npm run test -- src/components/__tests__/app-sidebar.test.tsx
```

Expected: FAIL — `sidebarItems` not exported, groups are Operations/Security/Administration.

- [ ] **Step 3: Regroup the sidebar**

Rewrite the data + render portions of `services/atlas-ui/src/components/app-sidebar.tsx`:

```tsx
import {
    Sidebar,
    SidebarContent,
    SidebarFooter,
    SidebarGroup,
    SidebarGroupContent,
    SidebarHeader,
    SidebarMenu,
    SidebarMenuButton,
    SidebarMenuItem,
    SidebarMenuSub,
    SidebarMenuSubButton,
    SidebarMenuSubItem,
    SidebarSeparator,
} from "@/components/ui/sidebar"
import {Cog, MonitorCog, Shield, Wrench, type LucideIcon} from "lucide-react";
import {Fragment} from "react";
import {Collapsible, CollapsibleContent, CollapsibleTrigger} from "@/components/ui/collapsible";
import { Link } from "react-router-dom";
import { useLocation } from "react-router-dom";
import {TenantSwitcher} from "@/components/app-tenant-switcher";
const logoImage = "/logo.png";

interface SidebarChildItem {
    title: string;
    url: string;
}

export interface SidebarGroupItem {
    title: string;
    url: string;
    icon: LucideIcon;
    /** Render a separator above this group (Deployment only). */
    separated?: boolean;
    /** Muted caption under the group label (Deployment only). */
    caption?: string;
    children: SidebarChildItem[];
}

// Menu items, grouped by blast radius: everything outside Deployment follows
// the tenant switcher; nothing inside it does. Exported so the sync test can
// assert Deployment children agree with isDeploymentRoute.
export const sidebarItems: SidebarGroupItem[] = [
    {
        title: "Operations",
        url: "#",
        icon: Cog,
        children: [
            { title: "Accounts", url: "/accounts" },
            { title: "Characters", url: "/characters" },
            { title: "Guilds", url: "/guilds" },
            { title: "NPCs", url: "/npcs" },
            { title: "Quests", url: "/quests" },
            { title: "Monsters", url: "/monsters" },
            { title: "Items", url: "/items" },
            { title: "Jobs", url: "/jobs" },
            { title: "Merchants", url: "/merchants" },
            { title: "Maps", url: "/maps" },
            { title: "Reactors", url: "/reactors" },
            { title: "Gachapons", url: "/gachapons" },
        ],
    },
    {
        title: "Security",
        url: "#",
        icon: Shield,
        children: [
            { title: "Bans", url: "/bans" },
            { title: "Login History", url: "/login-history" },
        ],
    },
    {
        title: "Setup",
        url: "#",
        icon: Wrench,
        children: [
            { title: "Setup", url: "/setup" },
        ],
    },
    {
        title: "Deployment",
        url: "#",
        icon: MonitorCog,
        separated: true,
        caption: "Applies to all tenants",
        children: [
            { title: "Templates", url: "/templates" },
            { title: "Tenants", url: "/tenants" },
            { title: "Services", url: "/services" },
            { title: "Baselines", url: "/baselines" },
        ],
    },
]
```

And the render loop (only the mapped block changes — header/footer stay as-is):

```tsx
                            {sidebarItems.map((item) => {
                                const isGroupActive = item.children.some((child) =>
                                    pathname === child.url || pathname.startsWith(child.url + "/")
                                )
                                return (
                                <Fragment key={item.title}>
                                {item.separated && <SidebarSeparator />}
                                <Collapsible defaultOpen={isGroupActive}>
                                <SidebarMenuItem className="group/collapsible">
                                    <CollapsibleTrigger asChild>
                                    <SidebarMenuButton className={item.caption ? "h-auto" : undefined}>
                                        <item.icon />
                                        <div className="grid flex-1 text-left leading-tight">
                                            <span>{item.title}</span>
                                            {item.caption && (
                                                <span className="text-xs text-muted-foreground">{item.caption}</span>
                                            )}
                                        </div>
                                    </SidebarMenuButton>
                                    </CollapsibleTrigger>
                                    <CollapsibleContent>
                                    <SidebarMenuSub>
                                        {item.children.map((child) => {
                                            const isActive = pathname === child.url || pathname.startsWith(child.url + "/")
                                            return (
                                            <SidebarMenuSubItem key={child.title}>
                                                <SidebarMenuSubButton asChild isActive={isActive}>
                                                    <Link to={child.url}>
                                                        <span>{child.title}</span>
                                                    </Link>
                                                </SidebarMenuSubButton>
                                            </SidebarMenuSubItem>
                                            )
                                        })}
                                    </SidebarMenuSub>
                                    </CollapsibleContent>
                                </SidebarMenuItem>
                                </Collapsible>
                                </Fragment>
                                )
                            })}
```

In `services/atlas-ui/src/lib/breadcrumbs/routes.ts`, update the Setup entry label (route path unchanged, FR-1.5):

```ts
  // Setup routes
  {
    pattern: '/setup',
    label: 'Setup',
    parent: '/',
  },
```

- [ ] **Step 4: Run test to verify it passes**

```bash
npm run test -- src/components/__tests__/app-sidebar.test.tsx && npx tsc -b --noEmit
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-ui/src/components/app-sidebar.tsx services/atlas-ui/src/components/__tests__/app-sidebar.test.tsx services/atlas-ui/src/lib/breadcrumbs/routes.ts
git commit -m "feat(task-134): sidebar regrouped by blast radius with Deployment treatment"
```

---

### Task 13: scope-aware tenant switcher

**Files:**
- Modify: `services/atlas-ui/src/components/app-tenant-switcher.tsx`
- Create: `services/atlas-ui/src/components/__tests__/app-tenant-switcher.test.tsx`

**Interfaces:**
- Consumes: `isDeploymentRoute` from Task 6; `useLocation` from react-router-dom.
- Produces: no API change — on Deployment routes the component renders an inert non-picker block; elsewhere the existing dropdown byte-for-byte. `TenantContext` is never touched on route change (FR-2.3 satisfied structurally).

- [ ] **Step 1: Write the failing test**

Create `services/atlas-ui/src/components/__tests__/app-tenant-switcher.test.tsx`:

```tsx
import { describe, it, expect, vi, beforeAll } from 'vitest';
import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { TenantSwitcher } from '@/components/app-tenant-switcher';
import { SidebarProvider } from '@/components/ui/sidebar';

const mockTenant = {
  id: '11111111-1111-1111-1111-111111111111',
  attributes: { name: 'Test Tenant', region: 'GMS', majorVersion: 83, minorVersion: 1 },
};

const setActiveTenant = vi.fn();

vi.mock('@/context/tenant-context', () => ({
  useTenant: () => ({
    tenants: [mockTenant],
    activeTenant: mockTenant,
    setActiveTenant,
    refreshTenants: vi.fn(),
  }),
}));

vi.mock('@/components/features/tenants/CreateTenantDialog', () => ({
  CreateTenantDialog: () => null,
}));

beforeAll(() => {
  Object.defineProperty(window, 'matchMedia', {
    writable: true,
    value: vi.fn().mockImplementation((query: string) => ({
      matches: false,
      media: query,
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(),
    })),
  });
});

function renderAt(path: string) {
  return render(
    <MemoryRouter initialEntries={[path]}>
      <SidebarProvider>
        <TenantSwitcher />
      </SidebarProvider>
    </MemoryRouter>,
  );
}

describe('TenantSwitcher', () => {
  it.each(['/templates', '/tenants/9f8e/writers', '/services', '/baselines'])(
    'renders the inert Deployment-wide state on %s',
    (path) => {
      renderAt(path);
      expect(screen.getByText('Deployment-wide')).toBeInTheDocument();
      expect(screen.getByText('tenant selection inactive')).toBeInTheDocument();
      // No dropdown affordance: the picker trigger must not exist.
      expect(screen.queryByRole('button')).not.toBeInTheDocument();
      expect(screen.queryByText('Test Tenant')).not.toBeInTheDocument();
    },
  );

  it.each(['/', '/accounts', '/setup', '/characters/42'])(
    'renders the interactive picker on %s',
    (path) => {
      renderAt(path);
      expect(screen.getByText('Test Tenant')).toBeInTheDocument();
      expect(screen.queryByText('Deployment-wide')).not.toBeInTheDocument();
    },
  );

  it('never writes tenant state from the inert branch', () => {
    renderAt('/templates');
    expect(setActiveTenant).not.toHaveBeenCalled();
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

```bash
npm run test -- src/components/__tests__/app-tenant-switcher.test.tsx
```

Expected: FAIL — "Deployment-wide" never renders.

- [ ] **Step 3: Add the inert branch**

In `services/atlas-ui/src/components/app-tenant-switcher.tsx`, add imports:

```tsx
import {useLocation} from "react-router-dom";
import {isDeploymentRoute} from "@/lib/deployment-routes";
```

At the top of the component body (after the existing hooks — hooks must run unconditionally before the early return):

```tsx
export function TenantSwitcher() {
    const {isMobile} = useSidebar()
    const {tenants, activeTenant, setActiveTenant, refreshTenants} = useTenant()
    const [createDialogOpen, setCreateDialogOpen] = React.useState(false)
    const {pathname} = useLocation()

    // On Deployment routes the switcher is inert: purely presentational, no
    // dropdown, no writes. TenantContext state and localStorage are never
    // touched, so the prior selection survives the round-trip (FR-2.3).
    if (isDeploymentRoute(pathname)) {
        return (
            <SidebarMenu>
                <SidebarMenuItem>
                    <SidebarMenuButton
                        size="lg"
                        asChild
                        aria-disabled="true"
                        className="pointer-events-none opacity-70"
                    >
                        <div data-testid="tenant-switcher-inert">
                            <div className="grid flex-1 text-left text-sm leading-tight">
                                <span className="truncate font-semibold">Deployment-wide</span>
                                <span className="truncate text-xs text-muted-foreground">tenant selection inactive</span>
                            </div>
                        </div>
                    </SidebarMenuButton>
                </SidebarMenuItem>
            </SidebarMenu>
        )
    }

    return (
        // ... existing dropdown JSX, unchanged ...
```

- [ ] **Step 4: Run test to verify it passes**

```bash
npm run test -- src/components/__tests__/app-tenant-switcher.test.tsx && npx tsc -b --noEmit
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-ui/src/components/app-tenant-switcher.tsx services/atlas-ui/src/components/__tests__/app-tenant-switcher.test.tsx
git commit -m "feat(task-134): tenant switcher renders inert Deployment-wide state on deployment routes"
```

---

### Task 14: deployment scope banner in the shell

**Files:**
- Create: `services/atlas-ui/src/components/common/deployment-scope-banner.tsx`
- Modify: `services/atlas-ui/src/components/features/navigation/app-shell.tsx`
- Create: `services/atlas-ui/src/components/common/__tests__/deployment-scope-banner.test.tsx`

**Interfaces:**
- Consumes: `isDeploymentRoute` from Task 6; shadcn `Alert`/`AlertDescription`.
- Produces: `export function DeploymentScopeBanner()` — self-conditions on the route (returns `null` off Deployment routes) so `AppShell` mounts it unconditionally in exactly one place; every current and future Deployment subpage inherits it.

- [ ] **Step 1: Write the failing test**

Create `services/atlas-ui/src/components/common/__tests__/deployment-scope-banner.test.tsx`:

```tsx
import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { DeploymentScopeBanner } from '@/components/common/deployment-scope-banner';

function renderAt(path: string) {
  return render(
    <MemoryRouter initialEntries={[path]}>
      <DeploymentScopeBanner />
    </MemoryRouter>,
  );
}

describe('DeploymentScopeBanner', () => {
  it.each(['/templates', '/templates/abc/writers', '/tenants/9f8e/character/presets', '/services', '/baselines'])(
    'shows the banner on deployment route %s (including subpages)',
    (path) => {
      renderAt(path);
      expect(screen.getByText('Changes on this page affect all tenants.')).toBeInTheDocument();
    },
  );

  it.each(['/', '/setup', '/accounts', '/characters/42'])('renders nothing on %s', (path) => {
    renderAt(path);
    expect(screen.queryByText('Changes on this page affect all tenants.')).not.toBeInTheDocument();
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

```bash
npm run test -- src/components/common/__tests__/deployment-scope-banner.test.tsx
```

Expected: FAIL — module unresolved.

- [ ] **Step 3: Implement the banner and mount it in the shell**

Create `services/atlas-ui/src/components/common/deployment-scope-banner.tsx`:

```tsx
import { Globe } from "lucide-react";
import { useLocation } from "react-router-dom";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { isDeploymentRoute } from "@/lib/deployment-routes";

/**
 * Slim, non-dismissible callout shown on every Deployment page (and all
 * their subpages). Mounted once in AppShell; self-conditions on the same
 * route predicate the tenant switcher uses, so the two scope signals can
 * never disagree.
 */
export function DeploymentScopeBanner() {
  const { pathname } = useLocation();
  if (!isDeploymentRoute(pathname)) return null;
  return (
    <Alert className="mx-2 w-auto border-amber-500/50 bg-amber-500/10 py-2 text-amber-900 dark:text-amber-200 [&>svg]:text-amber-600">
      <Globe className="h-4 w-4" />
      <AlertDescription>Changes on this page affect all tenants.</AlertDescription>
    </Alert>
  );
}
```

In `services/atlas-ui/src/components/features/navigation/app-shell.tsx`, add the import and mount the banner between the header and the content:

```tsx
import { DeploymentScopeBanner } from "@/components/common/deployment-scope-banner";
```

```tsx
        </header>
        <DeploymentScopeBanner />
        <div className="flex flex-1 flex-col overflow-hidden gap-4 p-2 pt-0">
```

- [ ] **Step 4: Run test to verify it passes**

```bash
npm run test -- src/components/common/__tests__/deployment-scope-banner.test.tsx && npx tsc -b --noEmit
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-ui/src/components/common/deployment-scope-banner.tsx services/atlas-ui/src/components/common/__tests__/deployment-scope-banner.test.tsx services/atlas-ui/src/components/features/navigation/app-shell.tsx
git commit -m "feat(task-134): deployment scope banner mounted once in the app shell"
```

---

### Task 15: `BaselineTargetPicker`

**Files:**
- Create: `services/atlas-ui/src/components/features/baselines/BaselineTargetPicker.tsx`
- Create: `services/atlas-ui/src/components/features/baselines/__tests__/BaselineTargetPicker.test.tsx`

**Interfaces:**
- Consumes: `useTemplates()` (`Template[]`, attributes `region`/`majorVersion`/`minorVersion`), `useTenants()` (`TenantBasic[]`, same attribute names), `CanonicalSelection`.
- Produces (Task 16 consumes all three):
  - `export function BaselineTargetPicker({ value, onChange }: { value: CanonicalSelection | null; onChange: (sel: CanonicalSelection | null) => void })`
  - `export function dedupeSelections(templates: Array<{ attributes: { region: string; majorVersion: number; minorVersion: number } }>, tenants: Array<{ attributes: { region: string; majorVersion: number; minorVersion: number } }>): CanonicalSelection[]` (pure; exported for tests)
  - `export function parseCustomSelection(region: string, major: string, minor: string): CanonicalSelection | null` (pure; exported for tests)
  - `export function selectionKey(sel: CanonicalSelection): string` returning `"<region>/<major>.<minor>"`

Radix `Select` interaction is jsdom-hostile (pointer-capture APIs), so behavior tests target the exported pure helpers plus a smoke render; the full flow is covered by the BaselinesPage tests (Task 16), which stub this picker.

- [ ] **Step 1: Write the failing tests**

Create `services/atlas-ui/src/components/features/baselines/__tests__/BaselineTargetPicker.test.tsx`:

```tsx
import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import {
  BaselineTargetPicker,
  dedupeSelections,
  parseCustomSelection,
  selectionKey,
} from '@/components/features/baselines/BaselineTargetPicker';

vi.mock('@/lib/hooks/api/useTemplates', () => ({
  useTemplates: () => ({
    data: [
      { id: 't1', attributes: { region: 'GMS', majorVersion: 83, minorVersion: 1 } },
      { id: 't2', attributes: { region: 'JMS', majorVersion: 185, minorVersion: 1 } },
    ],
  }),
}));

vi.mock('@/lib/hooks/api/useTenants', () => ({
  useTenants: () => ({
    data: [
      // Duplicate of the GMS template combo — must dedupe.
      { id: 'x1', attributes: { name: 'gms', region: 'GMS', majorVersion: 83, minorVersion: 1 } },
      { id: 'x2', attributes: { name: 'v87', region: 'GMS', majorVersion: 87, minorVersion: 1 } },
    ],
  }),
}));

describe('selectionKey', () => {
  it('formats region/major.minor', () => {
    expect(selectionKey({ region: 'GMS', majorVersion: 83, minorVersion: 1 })).toBe('GMS/83.1');
  });
});

describe('dedupeSelections', () => {
  it('unions templates and tenants, dedupes, and sorts', () => {
    const out = dedupeSelections(
      [
        { attributes: { region: 'JMS', majorVersion: 185, minorVersion: 1 } },
        { attributes: { region: 'GMS', majorVersion: 83, minorVersion: 1 } },
      ],
      [
        { attributes: { region: 'GMS', majorVersion: 83, minorVersion: 1 } },
        { attributes: { region: 'GMS', majorVersion: 87, minorVersion: 1 } },
      ],
    );
    expect(out.map(selectionKey)).toEqual(['GMS/83.1', 'GMS/87.1', 'JMS/185.1']);
  });

  it('works with zero tenants (templates only)', () => {
    const out = dedupeSelections(
      [{ attributes: { region: 'GMS', majorVersion: 83, minorVersion: 1 } }],
      [],
    );
    expect(out.map(selectionKey)).toEqual(['GMS/83.1']);
  });
});

describe('parseCustomSelection', () => {
  it('accepts a valid custom entry', () => {
    expect(parseCustomSelection('GMS', '92', '1')).toEqual({
      region: 'GMS',
      majorVersion: 92,
      minorVersion: 1,
    });
  });
  it.each([
    ['', '92', '1'],
    ['  ', '92', '1'],
    ['GMS', '', '1'],
    ['GMS', '-1', '1'],
    ['GMS', '9.5', '1'],
    ['GMS', 'abc', '1'],
    ['GMS', '92', '-2'],
    ['GMS', '92', ''],
  ])('rejects region=%j major=%j minor=%j', (region, major, minor) => {
    expect(parseCustomSelection(region, major, minor)).toBeNull();
  });
});

describe('BaselineTargetPicker render', () => {
  it('renders the trigger with a placeholder when nothing is selected', () => {
    render(<BaselineTargetPicker value={null} onChange={() => {}} />);
    expect(screen.getByText(/select region and version/i)).toBeInTheDocument();
  });
});
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
npm run test -- src/components/features/baselines/__tests__/BaselineTargetPicker.test.tsx
```

Expected: FAIL — module unresolved.

- [ ] **Step 3: Implement the picker**

Create `services/atlas-ui/src/components/features/baselines/BaselineTargetPicker.tsx`:

```tsx
import { useMemo, useState } from 'react';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { useTemplates } from '@/lib/hooks/api/useTemplates';
import { useTenants } from '@/lib/hooks/api/useTenants';
import type { CanonicalSelection } from '@/lib/headers';

const CUSTOM = '__custom__';

export function selectionKey(sel: CanonicalSelection): string {
  return `${sel.region}/${sel.majorVersion}.${sel.minorVersion}`;
}

interface HasRegionVersion {
  attributes: { region: string; majorVersion: number; minorVersion: number };
}

/**
 * Deduplicated union of (region, major, minor) combos from templates and
 * tenants, sorted by (region, major, minor). Provenance is irrelevant —
 * these are just seeds for the picker.
 */
export function dedupeSelections(
  templates: HasRegionVersion[],
  tenants: HasRegionVersion[],
): CanonicalSelection[] {
  const map = new Map<string, CanonicalSelection>();
  for (const item of [...templates, ...tenants]) {
    const sel: CanonicalSelection = {
      region: item.attributes.region,
      majorVersion: item.attributes.majorVersion,
      minorVersion: item.attributes.minorVersion,
    };
    map.set(selectionKey(sel), sel);
  }
  return [...map.values()].sort((a, b) => {
    if (a.region !== b.region) return a.region.localeCompare(b.region);
    if (a.majorVersion !== b.majorVersion) return a.majorVersion - b.majorVersion;
    return a.minorVersion - b.minorVersion;
  });
}

/**
 * Validates a custom entry: non-empty region, non-negative integer versions.
 * Returns null while invalid so workflow rows stay disabled.
 */
export function parseCustomSelection(
  region: string,
  major: string,
  minor: string,
): CanonicalSelection | null {
  const trimmed = region.trim();
  if (!trimmed) return null;
  if (!/^\d+$/.test(major) || !/^\d+$/.test(minor)) return null;
  return { region: trimmed, majorVersion: Number(major), minorVersion: Number(minor) };
}

interface BaselineTargetPickerProps {
  value: CanonicalSelection | null;
  onChange: (sel: CanonicalSelection | null) => void;
}

export function BaselineTargetPicker({ value, onChange }: BaselineTargetPickerProps) {
  const { data: templates } = useTemplates();
  const { data: tenants } = useTenants();
  const [selectedKey, setSelectedKey] = useState<string>('');
  const [customRegion, setCustomRegion] = useState('');
  const [customMajor, setCustomMajor] = useState('');
  const [customMinor, setCustomMinor] = useState('');

  const options = useMemo(
    () => dedupeSelections(templates ?? [], tenants ?? []),
    [templates, tenants],
  );

  const isCustom = selectedKey === CUSTOM;
  const customInvalid =
    isCustom &&
    (customRegion !== '' || customMajor !== '' || customMinor !== '') &&
    parseCustomSelection(customRegion, customMajor, customMinor) === null;

  const handleSelect = (key: string) => {
    setSelectedKey(key);
    if (key === CUSTOM) {
      onChange(parseCustomSelection(customRegion, customMajor, customMinor));
      return;
    }
    onChange(options.find((o) => selectionKey(o) === key) ?? null);
  };

  const handleCustomChange = (region: string, major: string, minor: string) => {
    setCustomRegion(region);
    setCustomMajor(major);
    setCustomMinor(minor);
    onChange(parseCustomSelection(region, major, minor));
  };

  return (
    <div className="flex flex-col gap-3" data-testid="baseline-target-picker">
      <Select value={selectedKey} onValueChange={handleSelect}>
        <SelectTrigger className="w-64" aria-label="Region and version">
          <SelectValue placeholder="Select region and version…" />
        </SelectTrigger>
        <SelectContent>
          {options.map((o) => (
            <SelectItem key={selectionKey(o)} value={selectionKey(o)}>
              {o.region} {o.majorVersion}.{o.minorVersion}
            </SelectItem>
          ))}
          <SelectItem value={CUSTOM}>Custom…</SelectItem>
        </SelectContent>
      </Select>
      {isCustom && (
        <div className="flex items-end gap-2">
          <div className="flex flex-col gap-1">
            <Label htmlFor="custom-region">Region</Label>
            <Input
              id="custom-region"
              className="w-28"
              value={customRegion}
              onChange={(e) => handleCustomChange(e.target.value, customMajor, customMinor)}
            />
          </div>
          <div className="flex flex-col gap-1">
            <Label htmlFor="custom-major">Major</Label>
            <Input
              id="custom-major"
              className="w-20"
              inputMode="numeric"
              value={customMajor}
              onChange={(e) => handleCustomChange(customRegion, e.target.value, customMinor)}
            />
          </div>
          <div className="flex flex-col gap-1">
            <Label htmlFor="custom-minor">Minor</Label>
            <Input
              id="custom-minor"
              className="w-20"
              inputMode="numeric"
              value={customMinor}
              onChange={(e) => handleCustomChange(customRegion, customMajor, e.target.value)}
            />
          </div>
        </div>
      )}
      {customInvalid && (
        <p className="text-sm text-destructive">
          Region must be non-empty; major and minor must be non-negative integers.
        </p>
      )}
      {value && (
        <p className="text-sm text-muted-foreground">
          Selected: {value.region} v{value.majorVersion}.{value.minorVersion}
        </p>
      )}
    </div>
  );
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
npm run test -- src/components/features/baselines/__tests__/BaselineTargetPicker.test.tsx && npx tsc -b --noEmit
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-ui/src/components/features/baselines/
git commit -m "feat(task-134): baseline target picker with template/tenant seeds and custom entry"
```

---

### Task 16: BaselinesPage + route + breadcrumb

**Files:**
- Create: `services/atlas-ui/src/pages/BaselinesPage.tsx`
- Modify: `services/atlas-ui/src/App.tsx`
- Modify: `services/atlas-ui/src/lib/breadcrumbs/routes.ts`
- Create: `services/atlas-ui/src/pages/__tests__/BaselinesPage.test.tsx`

**Interfaces:**
- Consumes: Task 11 hooks, Task 15 picker (stubbed in tests), `SetupRow`/`formatCount`/`pluralize` from `@/components/features/setup/SetupRow`, `formatBytes` from `@/lib/format`, `Baseline` type, shadcn `AlertDialog`/`Table`/`Card`/`Button`, sonner toasts.
- Produces: route `/baselines` → `BaselinesPage`; breadcrumb entry `{ pattern: '/baselines', label: 'Baselines', parent: '/' }` and `BASELINES: '/baselines'` in `ROUTE_PATTERNS`.

- [ ] **Step 1: Write the failing tests**

Create `services/atlas-ui/src/pages/__tests__/BaselinesPage.test.tsx`:

```tsx
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { BaselinesPage } from '@/pages/BaselinesPage';
import type { CanonicalSelection } from '@/lib/headers';
import type { Baseline } from '@/services/api/baseline.service';

// The picker has its own tests; stub it with a button that selects GMS 83.1.
vi.mock('@/components/features/baselines/BaselineTargetPicker', () => ({
  BaselineTargetPicker: ({ onChange }: { onChange: (sel: CanonicalSelection | null) => void }) => (
    <button onClick={() => onChange({ region: 'GMS', majorVersion: 83, minorVersion: 1 })}>
      pick-gms-83
    </button>
  ),
}));

// Mutable fixture state, reset per test.
let baselines: Baseline[] = [];
let wzStatus: { fileCount: number; totalBytes: number; updatedAt: string | null } | undefined;
let dataStatus:
  | { documentCount: number; updatedAt: string | null; baselineRestoredAt: string | null; baselineSha256: string | null }
  | undefined;
const uploadMutate = vi.fn();
const processMutate = vi.fn();
const publishMutate = vi.fn();

vi.mock('@/lib/hooks/api/useCanonicalData', () => ({
  useBaselines: () => ({ data: baselines, isLoading: false, isError: false, error: null }),
  useCanonicalWzInputStatus: () => ({ data: wzStatus }),
  useCanonicalDataStatus: () => ({ data: dataStatus }),
  useUploadCanonicalWz: () => ({ mutate: uploadMutate, isPending: false }),
  useRunCanonicalProcessing: () => ({ mutate: processMutate, isPending: false }),
  usePublishCanonicalBaseline: () => ({ mutate: publishMutate, isPending: false }),
}));

beforeEach(() => {
  baselines = [];
  wzStatus = undefined;
  dataStatus = undefined;
  uploadMutate.mockClear();
  processMutate.mockClear();
  publishMutate.mockClear();
});

const sampleBaseline: Baseline = {
  region: 'GMS',
  majorVersion: 83,
  minorVersion: 1,
  sha256: 'a'.repeat(64),
  publishedAt: '2026-07-04T12:34:56Z',
  sizeBytes: 123456789,
};

describe('BaselinesPage', () => {
  it('renders the empty state when no baselines are published', () => {
    render(<BaselinesPage />);
    expect(screen.getByText(/no canonical baselines published yet/i)).toBeInTheDocument();
  });

  it('renders baseline rows with truncated sha and an em dash for a blank sha', () => {
    baselines = [sampleBaseline, { ...sampleBaseline, region: 'JMS', majorVersion: 185, sha256: '', sizeBytes: 1024 }];
    render(<BaselinesPage />);
    expect(screen.getByText('GMS')).toBeInTheDocument();
    expect(screen.getByText('83.1')).toBeInTheDocument();
    expect(screen.getByText(`${'a'.repeat(12)}…`)).toBeInTheDocument();
    // The workflow badges also render em dashes while nothing is selected,
    // so assert at-least-one rather than exactly-one.
    expect(screen.getAllByText('—').length).toBeGreaterThan(0);
    // 123456789 bytes -> value >= 10 in MB -> zero decimals.
    expect(screen.getByText('118 MB')).toBeInTheDocument();
    expect(screen.getByText('1 KB')).toBeInTheDocument();
  });

  it('disables all workflow rows until a selection exists', () => {
    render(<BaselinesPage />);
    expect(screen.getByRole('button', { name: /upload/i })).toBeDisabled();
    expect(screen.getByRole('button', { name: /process data/i })).toBeDisabled();
    expect(screen.getByRole('button', { name: /publish baseline/i })).toBeDisabled();
  });

  it('enables upload after selection; process stays disabled with 0 wz files', () => {
    wzStatus = { fileCount: 0, totalBytes: 0, updatedAt: null };
    dataStatus = { documentCount: 0, updatedAt: null, baselineRestoredAt: null, baselineSha256: null };
    render(<BaselinesPage />);
    fireEvent.click(screen.getByText('pick-gms-83'));
    expect(screen.getByRole('button', { name: /upload/i })).toBeEnabled();
    expect(screen.getByRole('button', { name: /process data/i })).toBeDisabled();
    expect(screen.getByRole('button', { name: /publish baseline/i })).toBeDisabled();
  });

  it('enables process with wz files and publish with documents', () => {
    wzStatus = { fileCount: 10, totalBytes: 2048, updatedAt: null };
    dataStatus = { documentCount: 42, updatedAt: null, baselineRestoredAt: null, baselineSha256: null };
    render(<BaselinesPage />);
    fireEvent.click(screen.getByText('pick-gms-83'));
    expect(screen.getByRole('button', { name: /process data/i })).toBeEnabled();
    expect(screen.getByRole('button', { name: /publish baseline/i })).toBeEnabled();
  });

  it('publishes immediately when the selection has no existing baseline', () => {
    wzStatus = { fileCount: 10, totalBytes: 2048, updatedAt: null };
    dataStatus = { documentCount: 42, updatedAt: null, baselineRestoredAt: null, baselineSha256: null };
    render(<BaselinesPage />);
    fireEvent.click(screen.getByText('pick-gms-83'));
    fireEvent.click(screen.getByRole('button', { name: /publish baseline/i }));
    expect(publishMutate).toHaveBeenCalledTimes(1);
    expect(screen.queryByText(/replace the shared canonical baseline/i)).not.toBeInTheDocument();
  });

  it('requires confirmation when re-publishing over an existing baseline', () => {
    baselines = [sampleBaseline];
    wzStatus = { fileCount: 10, totalBytes: 2048, updatedAt: null };
    dataStatus = { documentCount: 42, updatedAt: null, baselineRestoredAt: null, baselineSha256: null };
    render(<BaselinesPage />);
    fireEvent.click(screen.getByText('pick-gms-83'));
    fireEvent.click(screen.getByRole('button', { name: /publish baseline/i }));
    expect(publishMutate).not.toHaveBeenCalled();
    expect(screen.getByText(/this will replace the shared canonical baseline for GMS v83\.1/i)).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: /replace baseline/i }));
    expect(publishMutate).toHaveBeenCalledTimes(1);
  });
});
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
npm run test -- src/pages/__tests__/BaselinesPage.test.tsx
```

Expected: FAIL — `@/pages/BaselinesPage` unresolved.

- [ ] **Step 3: Implement the page**

Create `services/atlas-ui/src/pages/BaselinesPage.tsx`:

```tsx
import { useRef, useState } from 'react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog';
import { Copy, FileArchive, FileText, Loader2, Send, Upload } from 'lucide-react';
import { Toaster, toast } from 'sonner';
import { SetupRow, formatCount, pluralize } from '@/components/features/setup/SetupRow';
import { BaselineTargetPicker } from '@/components/features/baselines/BaselineTargetPicker';
import {
  useBaselines,
  useCanonicalDataStatus,
  useCanonicalWzInputStatus,
  usePublishCanonicalBaseline,
  useRunCanonicalProcessing,
  useUploadCanonicalWz,
} from '@/lib/hooks/api/useCanonicalData';
import { formatBytes } from '@/lib/format';
import type { CanonicalSelection } from '@/lib/headers';

export function BaselinesPage() {
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [sel, setSel] = useState<CanonicalSelection | null>(null);
  const [confirmOpen, setConfirmOpen] = useState(false);

  const baselinesQuery = useBaselines();
  const wzInput = useCanonicalWzInputStatus(sel);
  const dataStatus = useCanonicalDataStatus(sel);
  const uploadWz = useUploadCanonicalWz(sel);
  const runProcessing = useRunCanonicalProcessing(sel);
  const publish = usePublishCanonicalBaseline(sel);

  const baselines = baselinesQuery.data ?? [];
  const wzData = wzInput.data;
  const docData = dataStatus.data;

  const existingBaseline = sel
    ? baselines.find(
        (b) =>
          b.region === sel.region &&
          b.majorVersion === sel.majorVersion &&
          b.minorVersion === sel.minorVersion,
      )
    : undefined;

  const handleFileUpload = (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (!file) return;
    if (!file.name.toLowerCase().endsWith('.zip')) {
      toast.error('Please select a .zip file');
      return;
    }
    const size = file.size;
    uploadWz.mutate(file, {
      onSuccess: () => {
        toast.success(`WZ files uploaded (${formatBytes(size)})`);
      },
    });
    if (fileInputRef.current) {
      fileInputRef.current.value = '';
    }
  };

  const handleRunProcessing = () => {
    runProcessing.mutate(undefined, {
      onSuccess: () => {
        toast.success('Data processing started');
      },
      onError: (error) => {
        toast.error(`Data processing failed: ${error.message}`);
      },
    });
  };

  const doPublish = () => {
    publish.mutate(undefined, {
      onSuccess: () => {
        toast.success('Canonical baseline published');
      },
      onError: (error) => {
        toast.error(`Baseline publish failed: ${error.message}`);
      },
    });
  };

  const handlePublish = () => {
    if (existingBaseline) {
      setConfirmOpen(true);
      return;
    }
    doPublish();
  };

  const handleCopySha = (sha: string) => {
    void navigator.clipboard.writeText(sha).then(() => toast.success('SHA-256 copied'));
  };

  const wzBadge = !sel
    ? '—'
    : !wzData
      ? '—'
      : wzData.fileCount === 0
        ? '0 .wz files'
        : `${formatCount(wzData.fileCount)} ${pluralize(wzData.fileCount, '.wz file', '.wz files')}, ${formatBytes(wzData.totalBytes)}`;

  const docBadge = !sel
    ? '—'
    : !docData
      ? '—'
      : `${formatCount(docData.documentCount)} ${pluralize(docData.documentCount, 'document loaded', 'documents loaded')}`;

  const processDisabled = !sel || !wzData || wzData.fileCount === 0 || uploadWz.isPending || runProcessing.isPending;
  const publishDisabled = !sel || !docData || docData.documentCount === 0 || publish.isPending;

  return (
    <div className="flex flex-col space-y-6 p-10 pb-16 overflow-y-auto">
      <div className="items-center justify-between space-y-2">
        <h2 className="text-2xl font-bold tracking-tight">Baselines</h2>
        <p className="text-muted-foreground">
          Manage canonical game-data baselines shared by all tenants of a region and version.
        </p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="text-lg">Published Baselines</CardTitle>
          <CardDescription>
            Canonical baselines new tenants restore their game data from.
          </CardDescription>
        </CardHeader>
        <CardContent>
          {baselinesQuery.isError ? (
            <p className="text-sm text-destructive">
              Failed to load baselines: {baselinesQuery.error?.message}
            </p>
          ) : baselines.length === 0 ? (
            <p className="text-sm text-muted-foreground">
              No canonical baselines published yet. A baseline is a published snapshot of processed
              canonical game data for one region/version; publish one from the workflow below and
              new tenants of that version will restore from it.
            </p>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Region</TableHead>
                  <TableHead>Version</TableHead>
                  <TableHead>SHA-256</TableHead>
                  <TableHead>Published</TableHead>
                  <TableHead>Size</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {baselines.map((b) => (
                  <TableRow key={`${b.region}/${b.majorVersion}.${b.minorVersion}`}>
                    <TableCell>{b.region}</TableCell>
                    <TableCell>{`${b.majorVersion}.${b.minorVersion}`}</TableCell>
                    <TableCell>
                      {b.sha256 ? (
                        <span className="inline-flex items-center gap-1 font-mono text-xs">
                          {`${b.sha256.slice(0, 12)}…`}
                          <Button
                            variant="ghost"
                            size="icon"
                            className="h-6 w-6"
                            aria-label={`Copy SHA-256 for ${b.region} ${b.majorVersion}.${b.minorVersion}`}
                            onClick={() => handleCopySha(b.sha256)}
                          >
                            <Copy className="h-3 w-3" />
                          </Button>
                        </span>
                      ) : (
                        '—'
                      )}
                    </TableCell>
                    <TableCell>{new Date(b.publishedAt).toLocaleString()}</TableCell>
                    <TableCell>{formatBytes(b.sizeBytes)}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-lg">Canonical Workflow</CardTitle>
          <CardDescription>
            Pick a region and version, then upload a WZ zip, process it, and publish the baseline.
            No tenant is involved — this works before any tenant of the version exists.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <input
            ref={fileInputRef}
            type="file"
            accept=".zip"
            className="hidden"
            onChange={handleFileUpload}
            aria-label="Upload WZ zip archive"
          />

          <div className="mb-4">
            <BaselineTargetPicker value={sel} onChange={setSel} />
          </div>

          <SetupRow
            icon={<FileArchive className="h-5 w-5" />}
            label="Upload WZ"
            badge={wzBadge}
            action={
              <Button
                size="sm"
                onClick={() => fileInputRef.current?.click()}
                disabled={!sel || uploadWz.isPending}
              >
                {uploadWz.isPending ? (
                  <>
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    Uploading…
                  </>
                ) : (
                  <>
                    <Upload className="mr-2 h-4 w-4" />
                    Upload
                  </>
                )}
              </Button>
            }
          />

          <SetupRow
            icon={<FileText className="h-5 w-5" />}
            label="Process Data"
            badge={docBadge}
            action={
              <Button
                size="sm"
                variant="outline"
                onClick={handleRunProcessing}
                disabled={processDisabled}
                title={sel && wzData && wzData.fileCount === 0 ? 'Upload WZ files first' : undefined}
              >
                {runProcessing.isPending ? (
                  <>
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    Processing…
                  </>
                ) : (
                  'Process Data'
                )}
              </Button>
            }
          />

          <SetupRow
            icon={<Send className="h-5 w-5" />}
            label="Publish Baseline"
            badge={
              existingBaseline?.sha256
                ? `current sha256:${existingBaseline.sha256.slice(0, 12)}…`
                : existingBaseline
                  ? 'published (sha unavailable)'
                  : 'not yet published'
            }
            action={
              <Button size="sm" variant="outline" onClick={handlePublish} disabled={publishDisabled}>
                {publish.isPending ? (
                  <>
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    Publishing…
                  </>
                ) : (
                  'Publish Baseline'
                )}
              </Button>
            }
          />
        </CardContent>
      </Card>

      <AlertDialog open={confirmOpen} onOpenChange={setConfirmOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Replace existing baseline?</AlertDialogTitle>
            <AlertDialogDescription>
              This will replace the shared canonical baseline for {sel?.region} v
              {sel?.majorVersion}.{sel?.minorVersion}.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
              onClick={doPublish}
            >
              Replace Baseline
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <Toaster richColors />
    </div>
  );
}
```

In `services/atlas-ui/src/App.tsx` add the lazy import (alphabetically with the others, after `BanDetailPage`):

```tsx
const BaselinesPage = lazy(() => import("@/pages/BaselinesPage").then(m => ({ default: m.BaselinesPage })));
```

and the route (after the `/bans/:banId` route):

```tsx
                    <Route path="/baselines" element={<BaselinesPage />} />
```

In `services/atlas-ui/src/lib/breadcrumbs/routes.ts` add to `ROUTE_CONFIGS` (next to the other main entity list routes):

```ts
  {
    pattern: '/baselines',
    label: 'Baselines',
    parent: '/',
  },
```

and to `ROUTE_PATTERNS`:

```ts
  BASELINES: '/baselines',
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
npm run test -- src/pages/__tests__/BaselinesPage.test.tsx && npx tsc -b --noEmit
```

Expected: PASS. (If the `118 MB` size assertion fails due to `Intl` locale rounding, quote the actual rendered value from the failure output and update the expected string — the fixture is 123456789 bytes.)

- [ ] **Step 5: Commit**

```bash
git add services/atlas-ui/src/pages/BaselinesPage.tsx services/atlas-ui/src/pages/__tests__/BaselinesPage.test.tsx services/atlas-ui/src/App.tsx services/atlas-ui/src/lib/breadcrumbs/routes.ts
git commit -m "feat(task-134): Baselines page with canonical upload-process-publish workflow"
```

---

### Task 17: full verification sweep

**Files:** none created; verification only. Fix anything that fails, then re-run until clean.

- [ ] **Step 1: atlas-ui full test suite + production build**

Run from `services/atlas-ui/`:

```bash
npm run test && npm run build
```

Expected: every test file passes (including all pre-existing ones); build completes with no type errors.

- [ ] **Step 2: atlas-data full gates (re-run — the UI tasks must not have touched Go, this confirms it)**

Run from `services/atlas-data/atlas.com/data/`:

```bash
go test -race ./... && go vet ./... && go build ./...
```

Expected: clean.

- [ ] **Step 3: bake + guard from the worktree root**

```bash
docker buildx bake atlas-data && tools/redis-key-guard.sh
```

Expected: clean.

- [ ] **Step 4: Residue scan**

```bash
grep -rn "ScopeToggle" services/atlas-ui/src/ ; grep -rn "usePublishBaseline" services/atlas-ui/src/ ; grep -rn "TODO" services/atlas-ui/src/pages/BaselinesPage.tsx services/atlas-ui/src/lib/deployment-routes.ts services/atlas-data/atlas.com/data/baseline/list.go
```

Expected: no matches from any of the three greps.

- [ ] **Step 5: Commit any fixes and verify branch state**

```bash
git status --short
git rev-parse --show-toplevel   # must end with .worktrees/task-134-admin-nav-baselines
git branch --show-current       # must be task-134-admin-nav-baselines
```

Expected: clean tree (or commit stragglers with `fix(task-134): ...`), correct worktree and branch.

After this task: run the code-review step (`superpowers:requesting-code-review`) before opening a PR — mandatory per CLAUDE.md.
