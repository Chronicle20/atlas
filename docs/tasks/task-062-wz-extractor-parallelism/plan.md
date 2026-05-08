# atlas-wz-extractor Parallelism Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace `atlas-wz-extractor`'s single-goroutine extraction loop with a Kafka-fanout job model — one `START_EXTRACTION_UNIT` message per WZ file, Redis-backed per-tenant lock and job state, and a `GET /jobs/{jobId}` status endpoint that works from any pod.

**Architecture:** A REST `POST /api/wz/extractions` *dispatcher* acquires a Redis NX tenant lock, wipes character cache, creates a Redis job record + N pending unit records, and emits N Kafka messages on `COMMAND_TOPIC_WZ_EXTRACTION` (consumer group `wz-extractor-extraction`). Within-pod parallelism comes from Kafka partition count (default 16). Each consumer runs the existing per-WZ-file logic via a new `ExtractUnit` method, transitions the unit's Redis state with a WATCH/MULTI/EXEC guard against redelivery, and the "last one home" consumer declares the terminal job status and releases the lock. A 60-minute lock with auto-refresh every 20 minutes survives long Map.wz units. Map.wz remains one unit (its internal `RenderMaps` pool keeps within-pod parallelism for that one file).

**Tech Stack:** Go 1.25, `libs/atlas-kafka` (managers + curried InitConsumers), `libs/atlas-redis` (`Connect`, `Lock`), `libs/atlas-tenant`, `libs/atlas-rest/server`, `redis/go-redis/v9` (for WATCH/Lua), `alicebob/miniredis/v2` (tests), `segmentio/kafka-go`.

---

## File map

**Created:**
- `services/atlas-wz-extractor/atlas.com/wz-extractor/extraction/job/keys.go` — Redis key helpers (private to package).
- `services/atlas-wz-extractor/atlas.com/wz-extractor/extraction/job/model.go` — `Job`, `Unit` immutable models + Builders + status enums.
- `services/atlas-wz-extractor/atlas.com/wz-extractor/extraction/job/store.go` — `Store` interface + Redis impl (`storeImpl`).
- `services/atlas-wz-extractor/atlas.com/wz-extractor/extraction/job/store_test.go` — `miniredis`-backed unit tests.
- `services/atlas-wz-extractor/atlas.com/wz-extractor/extraction/lock/tenant_lock.go` — Redis lock wrapper with auto-refresh + CAS release.
- `services/atlas-wz-extractor/atlas.com/wz-extractor/extraction/lock/tenant_lock_test.go` — `miniredis`-backed tests.
- `services/atlas-wz-extractor/atlas.com/wz-extractor/extraction/pool.go` — bounded worker pool over `ExtractUnit`.
- `services/atlas-wz-extractor/atlas.com/wz-extractor/extraction/pool_test.go` — pool tests.
- `services/atlas-wz-extractor/atlas.com/wz-extractor/extraction/dispatcher.go` — `handleExtract` orchestration.
- `services/atlas-wz-extractor/atlas.com/wz-extractor/extraction/dispatcher_test.go` — dispatcher tests.
- `services/atlas-wz-extractor/atlas.com/wz-extractor/extraction/job_handler.go` — `GET /api/wz/extractions/jobs/{jobId}`.
- `services/atlas-wz-extractor/atlas.com/wz-extractor/extraction/job_handler_test.go` — handler tests.
- `services/atlas-wz-extractor/atlas.com/wz-extractor/kafka/consumer/consumer.go` — `NewConfig` curried builder, `LookupBrokers`.
- `services/atlas-wz-extractor/atlas.com/wz-extractor/kafka/consumer/extraction/kafka.go` — env names + command + body types.
- `services/atlas-wz-extractor/atlas.com/wz-extractor/kafka/consumer/extraction/consumer.go` — `InitConsumers`, `InitHandlers`, `handleStartExtractionUnit`.
- `services/atlas-wz-extractor/atlas.com/wz-extractor/kafka/consumer/extraction/consumer_test.go` — handler tests with miniredis + fake processor.
- `services/atlas-wz-extractor/atlas.com/wz-extractor/kafka/producer/producer.go` — copy of atlas-data's `ProviderImpl`.
- `services/atlas-wz-extractor/atlas.com/wz-extractor/kafka/message/extraction/kafka.go` — producer-side env name + provider.
- `services/atlas-wz-extractor/atlas.com/wz-extractor/extraction/dispatcher_emit.go` — `startExtractionUnitCommandProvider` building `[]kafka.Message`.
- `services/atlas-wz-extractor/atlas.com/wz-extractor/docs/kafka.md` — topic name, partition count, header parsers.
- `services/atlas-wz-extractor/atlas.com/wz-extractor/docs/storage.md` — Redis schema for jobs/units/lock.

**Modified:**
- `services/atlas-wz-extractor/atlas.com/wz-extractor/extraction/processor.go` — split `Extract` into `ExtractUnit` + `Extract` (using pool).
- `services/atlas-wz-extractor/atlas.com/wz-extractor/extraction/processor_test.go` — adjust to new method shape, add `ExtractUnit` tests.
- `services/atlas-wz-extractor/atlas.com/wz-extractor/extraction/resource.go` — accept new deps (`job.Store`, `lock.TenantLock`, producer provider, dirs); register POST + GET-by-id; remove the inline goroutine handler (moved to `dispatcher.go`).
- `services/atlas-wz-extractor/atlas.com/wz-extractor/extraction/resource_test.go` — update to new `InitResource` signature.
- `services/atlas-wz-extractor/atlas.com/wz-extractor/main.go` — wire Redis client, Kafka producer manager teardown, consumer registration.
- `services/atlas-wz-extractor/atlas.com/wz-extractor/go.mod` — add new deps.
- `services/atlas-wz-extractor/atlas.com/wz-extractor/go.sum` — go mod tidy.
- `go.work.sum` — go mod tidy at workspace level.
- `deploy/k8s/atlas-wz-extractor.yaml` — env vars + resources + topic provisioning hint.

**Deleted:**
- `services/atlas-wz-extractor/atlas.com/wz-extractor/extraction/mutex.go`
- `services/atlas-wz-extractor/atlas.com/wz-extractor/extraction/mutex_test.go`

---

## Pre-flight checks

Always work in the task worktree.

- [ ] **Step 0.1: Verify cwd and branch**

```bash
cd /home/tumidanski/source/atlas-ms/atlas/.worktrees/task-062-wz-extractor-parallelism
pwd                       # must end with /.worktrees/task-062-wz-extractor-parallelism
git branch --show-current # must print task-062-wz-extractor-parallelism
```

- [ ] **Step 0.2: Verify clean working tree**

```bash
git status --short
```

Expected: empty output (or only the in-progress plan/context files we just committed).

---

## Task 1: Add module dependencies

**Files:**
- Modify: `services/atlas-wz-extractor/atlas.com/wz-extractor/go.mod`

- [ ] **Step 1.1: Add Redis, Kafka, miniredis, kafka-go imports placeholder**

We can't `go get` for local-replace libs. Add the libs/atlas-* imports by importing them in code in subsequent tasks; `go mod tidy` will resolve. For the third-party deps, run from the service module root:

```bash
cd services/atlas-wz-extractor/atlas.com/wz-extractor
go get github.com/redis/go-redis/v9
go get github.com/alicebob/miniredis/v2
go get github.com/segmentio/kafka-go
```

- [ ] **Step 1.2: Verify go.mod entries appear**

```bash
grep -E "redis/go-redis|miniredis|segmentio/kafka-go" services/atlas-wz-extractor/atlas.com/wz-extractor/go.mod
```

Expected: three matching lines.

- [ ] **Step 1.3: Commit**

```bash
git add services/atlas-wz-extractor/atlas.com/wz-extractor/go.mod services/atlas-wz-extractor/atlas.com/wz-extractor/go.sum go.work.sum
git commit -m "chore(atlas-wz-extractor): add redis/kafka/miniredis deps"
```

---

## Task 2: `extraction/job/keys.go` — Redis key helpers

**Files:**
- Create: `services/atlas-wz-extractor/atlas.com/wz-extractor/extraction/job/keys.go`

- [ ] **Step 2.1: Write the file**

```go
package job

import "fmt"

// Key namespace prefix. Kept private to the package so the layout cannot drift.
const namespace = "wz-extractor"

func jobKey(jobId string) string {
	return fmt.Sprintf("%s:job:%s", namespace, jobId)
}

func unitsKey(jobId string) string {
	return fmt.Sprintf("%s:job:%s:units", namespace, jobId)
}

// LockKey composes the tenant-lock key. Exported so the lock package and the
// dispatcher can both reference it without re-deriving the format.
func LockKey(tenantId, region string, major, minor uint16) string {
	return fmt.Sprintf("%s:tenant-lock:%s:%s:%d.%d", namespace, tenantId, region, major, minor)
}
```

- [ ] **Step 2.2: Compile**

```bash
cd services/atlas-wz-extractor/atlas.com/wz-extractor && go build ./extraction/job/...
```

Expected: no output (success).

- [ ] **Step 2.3: Commit**

```bash
git add services/atlas-wz-extractor/atlas.com/wz-extractor/extraction/job/keys.go
git commit -m "feat(atlas-wz-extractor): add job package key helpers"
```

---

## Task 3: `extraction/job/model.go` — domain types

**Files:**
- Create: `services/atlas-wz-extractor/atlas.com/wz-extractor/extraction/job/model.go`

- [ ] **Step 3.1: Write the failing test (model behavior)**

Create `services/atlas-wz-extractor/atlas.com/wz-extractor/extraction/job/model_test.go`:

```go
package job

import (
	"testing"
	"time"
)

func TestJobBuilderAndGetters(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	j := NewJobBuilder().
		SetId("job-1").
		SetTenantId("tenant-1").
		SetRegion("GMS").
		SetMajorVersion(83).
		SetMinorVersion(1).
		SetStatus(JobPending).
		SetUnitsTotal(11).
		SetXmlOnly(true).
		SetImagesOnly(false).
		SetCreatedAt(now).
		SetUpdatedAt(now).
		Build()

	if j.Id() != "job-1" {
		t.Fatalf("Id: got %s", j.Id())
	}
	if j.UnitsTotal() != 11 {
		t.Fatalf("UnitsTotal: got %d", j.UnitsTotal())
	}
	if j.XmlOnly() != true || j.ImagesOnly() != false {
		t.Fatalf("flags: %v %v", j.XmlOnly(), j.ImagesOnly())
	}
	if j.Status() != JobPending {
		t.Fatalf("Status: got %s", j.Status())
	}
}

func TestUnitBuilderAndGetters(t *testing.T) {
	u := NewUnitBuilder().
		SetWzFile("Map.wz").
		SetStatus(UnitPending).
		Build()
	if u.WzFile() != "Map.wz" || u.Status() != UnitPending {
		t.Fatalf("Unit fields: %v", u)
	}
}
```

- [ ] **Step 3.2: Run the failing test**

```bash
cd services/atlas-wz-extractor/atlas.com/wz-extractor && go test ./extraction/job/ -run "TestJobBuilderAndGetters|TestUnitBuilderAndGetters" -v
```

Expected: build error (types don't exist).

- [ ] **Step 3.3: Implement `model.go`**

Create `services/atlas-wz-extractor/atlas.com/wz-extractor/extraction/job/model.go`:

```go
package job

import "time"

// JobStatus is the terminal/intermediate state of an extraction job.
type JobStatus string

const (
	JobPending             JobStatus = "pending"
	JobRunning             JobStatus = "running"
	JobCompleted           JobStatus = "completed"
	JobCompletedWithErrors JobStatus = "completed_with_errors"
	JobFailed              JobStatus = "failed"
)

// UnitStatus is per-WZ-file state.
type UnitStatus string

const (
	UnitPending   UnitStatus = "pending"
	UnitRunning   UnitStatus = "running"
	UnitSucceeded UnitStatus = "succeeded"
	UnitFailed    UnitStatus = "failed"
	UnitSkipped   UnitStatus = "skipped"
)

// Job is an immutable snapshot of one extraction job.
type Job struct {
	id             string
	tenantId       string
	region         string
	majorVersion   uint16
	minorVersion   uint16
	status         JobStatus
	unitsTotal     int
	unitsCompleted int
	unitsFailed    int
	xmlOnly        bool
	imagesOnly     bool
	createdAt      time.Time
	updatedAt      time.Time
	completedAt    time.Time
}

func (j Job) Id() string             { return j.id }
func (j Job) TenantId() string       { return j.tenantId }
func (j Job) Region() string         { return j.region }
func (j Job) MajorVersion() uint16   { return j.majorVersion }
func (j Job) MinorVersion() uint16   { return j.minorVersion }
func (j Job) Status() JobStatus      { return j.status }
func (j Job) UnitsTotal() int        { return j.unitsTotal }
func (j Job) UnitsCompleted() int    { return j.unitsCompleted }
func (j Job) UnitsFailed() int       { return j.unitsFailed }
func (j Job) XmlOnly() bool          { return j.xmlOnly }
func (j Job) ImagesOnly() bool       { return j.imagesOnly }
func (j Job) CreatedAt() time.Time   { return j.createdAt }
func (j Job) UpdatedAt() time.Time   { return j.updatedAt }
func (j Job) CompletedAt() time.Time { return j.completedAt }

// JobBuilder constructs an immutable Job.
type JobBuilder struct{ j Job }

func NewJobBuilder() *JobBuilder { return &JobBuilder{} }

func (b *JobBuilder) SetId(v string) *JobBuilder             { b.j.id = v; return b }
func (b *JobBuilder) SetTenantId(v string) *JobBuilder       { b.j.tenantId = v; return b }
func (b *JobBuilder) SetRegion(v string) *JobBuilder         { b.j.region = v; return b }
func (b *JobBuilder) SetMajorVersion(v uint16) *JobBuilder   { b.j.majorVersion = v; return b }
func (b *JobBuilder) SetMinorVersion(v uint16) *JobBuilder   { b.j.minorVersion = v; return b }
func (b *JobBuilder) SetStatus(v JobStatus) *JobBuilder      { b.j.status = v; return b }
func (b *JobBuilder) SetUnitsTotal(v int) *JobBuilder        { b.j.unitsTotal = v; return b }
func (b *JobBuilder) SetUnitsCompleted(v int) *JobBuilder    { b.j.unitsCompleted = v; return b }
func (b *JobBuilder) SetUnitsFailed(v int) *JobBuilder       { b.j.unitsFailed = v; return b }
func (b *JobBuilder) SetXmlOnly(v bool) *JobBuilder          { b.j.xmlOnly = v; return b }
func (b *JobBuilder) SetImagesOnly(v bool) *JobBuilder       { b.j.imagesOnly = v; return b }
func (b *JobBuilder) SetCreatedAt(v time.Time) *JobBuilder   { b.j.createdAt = v; return b }
func (b *JobBuilder) SetUpdatedAt(v time.Time) *JobBuilder   { b.j.updatedAt = v; return b }
func (b *JobBuilder) SetCompletedAt(v time.Time) *JobBuilder { b.j.completedAt = v; return b }
func (b *JobBuilder) Build() Job                             { return b.j }

// Unit is an immutable per-WZ-file record.
type Unit struct {
	wzFile      string
	status      UnitStatus
	startedAt   time.Time
	completedAt time.Time
	errMsg      string
}

func (u Unit) WzFile() string         { return u.wzFile }
func (u Unit) Status() UnitStatus     { return u.status }
func (u Unit) StartedAt() time.Time   { return u.startedAt }
func (u Unit) CompletedAt() time.Time { return u.completedAt }
func (u Unit) ErrorMessage() string   { return u.errMsg }

type UnitBuilder struct{ u Unit }

func NewUnitBuilder() *UnitBuilder                            { return &UnitBuilder{} }
func (b *UnitBuilder) SetWzFile(v string) *UnitBuilder        { b.u.wzFile = v; return b }
func (b *UnitBuilder) SetStatus(v UnitStatus) *UnitBuilder    { b.u.status = v; return b }
func (b *UnitBuilder) SetStartedAt(v time.Time) *UnitBuilder  { b.u.startedAt = v; return b }
func (b *UnitBuilder) SetCompletedAt(v time.Time) *UnitBuilder {
	b.u.completedAt = v
	return b
}
func (b *UnitBuilder) SetErrorMessage(v string) *UnitBuilder { b.u.errMsg = v; return b }
func (b *UnitBuilder) Build() Unit                           { return b.u }

// Counters returned by FinalizeUnit; what the consumer needs to decide whether
// it's the "last one home" without a second Redis read.
type Counters struct {
	UnitsTotal     int
	UnitsCompleted int
	UnitsFailed    int
	AllDone        bool
	LockKey        string
}
```

- [ ] **Step 3.4: Run tests, expect PASS**

```bash
cd services/atlas-wz-extractor/atlas.com/wz-extractor && go test ./extraction/job/ -run "TestJobBuilderAndGetters|TestUnitBuilderAndGetters" -v
```

Expected: PASS.

- [ ] **Step 3.5: Commit**

```bash
git add services/atlas-wz-extractor/atlas.com/wz-extractor/extraction/job/model.go services/atlas-wz-extractor/atlas.com/wz-extractor/extraction/job/model_test.go
git commit -m "feat(atlas-wz-extractor): add job/unit immutable models with builders"
```

---

## Task 4: `extraction/job/store.go` — Store interface skeleton

**Files:**
- Create: `services/atlas-wz-extractor/atlas.com/wz-extractor/extraction/job/store.go`

- [ ] **Step 4.1: Write the interface and constructor stub**

```go
package job

import (
	"context"
	"errors"

	goredis "github.com/redis/go-redis/v9"
)

// Store persists Jobs and Units in Redis. All methods are safe for concurrent
// use; correctness across pods is enforced by Redis primitives (HINCRBY,
// WATCH/MULTI/EXEC) inside the implementation.
type Store interface {
	Create(ctx context.Context, j Job, units []Unit, ttlSeconds int) error
	Get(ctx context.Context, jobId string) (Job, []Unit, error)

	MarkJobRunning(ctx context.Context, jobId string) error
	MarkUnitRunning(ctx context.Context, jobId, wzFile string) (claimed bool, err error)
	FinalizeUnit(ctx context.Context, jobId, wzFile string, terminal UnitStatus, runErr error) (Counters, error)
	MarkJobTerminal(ctx context.Context, jobId string, terminal JobStatus) (claimed bool, err error)
	MarkUnitsSkippedByStatus(ctx context.Context, jobId string, fromStatuses []UnitStatus) error

	Delete(ctx context.Context, jobId string) error
}

// ErrNotFound is returned by Get when the jobId does not exist.
var ErrNotFound = errors.New("job not found")

type storeImpl struct {
	client *goredis.Client
}

func NewStore(client *goredis.Client) Store {
	return &storeImpl{client: client}
}
```

- [ ] **Step 4.2: Compile (interface only, methods unimplemented)**

```bash
cd services/atlas-wz-extractor/atlas.com/wz-extractor && go build ./extraction/job/...
```

Expected: build error: `*storeImpl does not implement Store`. That's intentional — next task implements methods one by one.

- [ ] **Step 4.3: Commit (with build still broken — next tasks fix it)**

Don't commit yet; we'll commit along with the first method implementation in Task 5 to keep the tree green at every commit.

---

## Task 5: `Store.Create` + `Store.Get` + `Store.Delete`

**Files:**
- Modify: `services/atlas-wz-extractor/atlas.com/wz-extractor/extraction/job/store.go`
- Create: `services/atlas-wz-extractor/atlas.com/wz-extractor/extraction/job/store_test.go`

- [ ] **Step 5.1: Write the failing test**

```go
package job

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
)

func newTestClient(t *testing.T) *goredis.Client {
	t.Helper()
	mr := miniredis.RunT(t)
	return goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
}

func TestStore_CreateGetDelete(t *testing.T) {
	ctx := context.Background()
	c := newTestClient(t)
	s := NewStore(c)

	now := time.Now().UTC().Truncate(time.Second)
	j := NewJobBuilder().
		SetId("job-1").
		SetTenantId("tenant-1").
		SetRegion("GMS").
		SetMajorVersion(83).SetMinorVersion(1).
		SetStatus(JobPending).
		SetUnitsTotal(2).
		SetXmlOnly(false).SetImagesOnly(false).
		SetCreatedAt(now).SetUpdatedAt(now).
		Build()
	units := []Unit{
		NewUnitBuilder().SetWzFile("Map.wz").SetStatus(UnitPending).Build(),
		NewUnitBuilder().SetWzFile("Mob.wz").SetStatus(UnitPending).Build(),
	}

	if err := s.Create(ctx, j, units, 3600); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, gotUnits, err := s.Get(ctx, "job-1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Id() != "job-1" || got.UnitsTotal() != 2 || got.Status() != JobPending {
		t.Fatalf("Get returned: %+v", got)
	}
	if len(gotUnits) != 2 {
		t.Fatalf("expected 2 units, got %d", len(gotUnits))
	}

	if err := s.Delete(ctx, "job-1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, _, err := s.Get(ctx, "job-1"); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound after Delete, got %v", err)
	}
}
```

- [ ] **Step 5.2: Run test, expect FAIL (Store methods unimplemented)**

```bash
cd services/atlas-wz-extractor/atlas.com/wz-extractor && go test ./extraction/job/ -run "TestStore_CreateGetDelete" -v
```

Expected: FAIL (compile error or method missing).

- [ ] **Step 5.3: Implement `Create`, `Get`, `Delete`**

Append to `services/atlas-wz-extractor/atlas.com/wz-extractor/extraction/job/store.go`:

```go
import (
	// existing imports kept
	"encoding/json"
	"strconv"
	"time"
)

type unitJSON struct {
	Status      string `json:"status"`
	StartedAt   string `json:"startedAt,omitempty"`
	CompletedAt string `json:"completedAt,omitempty"`
	Error       string `json:"error,omitempty"`
}

func unitToJSON(u Unit) (string, error) {
	uj := unitJSON{Status: string(u.Status())}
	if !u.StartedAt().IsZero() {
		uj.StartedAt = u.StartedAt().UTC().Format(time.RFC3339)
	}
	if !u.CompletedAt().IsZero() {
		uj.CompletedAt = u.CompletedAt().UTC().Format(time.RFC3339)
	}
	uj.Error = u.ErrorMessage()
	b, err := json.Marshal(uj)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func unitFromJSON(wzFile, raw string) (Unit, error) {
	var uj unitJSON
	if err := json.Unmarshal([]byte(raw), &uj); err != nil {
		return Unit{}, err
	}
	b := NewUnitBuilder().SetWzFile(wzFile).SetStatus(UnitStatus(uj.Status))
	if uj.StartedAt != "" {
		if t, err := time.Parse(time.RFC3339, uj.StartedAt); err == nil {
			b = b.SetStartedAt(t)
		}
	}
	if uj.CompletedAt != "" {
		if t, err := time.Parse(time.RFC3339, uj.CompletedAt); err == nil {
			b = b.SetCompletedAt(t)
		}
	}
	if uj.Error != "" {
		b = b.SetErrorMessage(uj.Error)
	}
	return b.Build(), nil
}

func (s *storeImpl) Create(ctx context.Context, j Job, units []Unit, ttlSeconds int) error {
	jKey := jobKey(j.Id())
	uKey := unitsKey(j.Id())

	jobFields := map[string]interface{}{
		"tenantId":       j.TenantId(),
		"region":         j.Region(),
		"majorVersion":   strconv.Itoa(int(j.MajorVersion())),
		"minorVersion":   strconv.Itoa(int(j.MinorVersion())),
		"status":         string(j.Status()),
		"unitsTotal":     strconv.Itoa(j.UnitsTotal()),
		"unitsCompleted": "0",
		"unitsFailed":    "0",
		"xmlOnly":        strconv.FormatBool(j.XmlOnly()),
		"imagesOnly":     strconv.FormatBool(j.ImagesOnly()),
		"createdAt":      j.CreatedAt().UTC().Format(time.RFC3339),
		"updatedAt":      j.UpdatedAt().UTC().Format(time.RFC3339),
	}

	pipe := s.client.TxPipeline()
	pipe.HSet(ctx, jKey, jobFields)
	if ttlSeconds > 0 {
		pipe.Expire(ctx, jKey, time.Duration(ttlSeconds)*time.Second)
	}

	uMap := map[string]interface{}{}
	for _, u := range units {
		raw, err := unitToJSON(u)
		if err != nil {
			return err
		}
		uMap[u.WzFile()] = raw
	}
	if len(uMap) > 0 {
		pipe.HSet(ctx, uKey, uMap)
		if ttlSeconds > 0 {
			pipe.Expire(ctx, uKey, time.Duration(ttlSeconds)*time.Second)
		}
	}
	_, err := pipe.Exec(ctx)
	return err
}

func (s *storeImpl) Get(ctx context.Context, jobId string) (Job, []Unit, error) {
	fields, err := s.client.HGetAll(ctx, jobKey(jobId)).Result()
	if err != nil {
		return Job{}, nil, err
	}
	if len(fields) == 0 {
		return Job{}, nil, ErrNotFound
	}

	parseInt := func(v string) int {
		n, _ := strconv.Atoi(v)
		return n
	}
	parseTime := func(v string) time.Time {
		if v == "" {
			return time.Time{}
		}
		t, _ := time.Parse(time.RFC3339, v)
		return t
	}
	parseBool := func(v string) bool {
		b, _ := strconv.ParseBool(v)
		return b
	}

	jb := NewJobBuilder().
		SetId(jobId).
		SetTenantId(fields["tenantId"]).
		SetRegion(fields["region"]).
		SetMajorVersion(uint16(parseInt(fields["majorVersion"]))).
		SetMinorVersion(uint16(parseInt(fields["minorVersion"]))).
		SetStatus(JobStatus(fields["status"])).
		SetUnitsTotal(parseInt(fields["unitsTotal"])).
		SetUnitsCompleted(parseInt(fields["unitsCompleted"])).
		SetUnitsFailed(parseInt(fields["unitsFailed"])).
		SetXmlOnly(parseBool(fields["xmlOnly"])).
		SetImagesOnly(parseBool(fields["imagesOnly"])).
		SetCreatedAt(parseTime(fields["createdAt"])).
		SetUpdatedAt(parseTime(fields["updatedAt"])).
		SetCompletedAt(parseTime(fields["completedAt"]))
	j := jb.Build()

	uMap, err := s.client.HGetAll(ctx, unitsKey(jobId)).Result()
	if err != nil {
		return Job{}, nil, err
	}
	units := make([]Unit, 0, len(uMap))
	for wzFile, raw := range uMap {
		u, err := unitFromJSON(wzFile, raw)
		if err != nil {
			return Job{}, nil, err
		}
		units = append(units, u)
	}
	return j, units, nil
}

func (s *storeImpl) Delete(ctx context.Context, jobId string) error {
	_, err := s.client.Del(ctx, jobKey(jobId), unitsKey(jobId)).Result()
	return err
}
```

Stub the remaining methods so the package compiles:

```go
func (s *storeImpl) MarkJobRunning(ctx context.Context, jobId string) error {
	return errors.New("not implemented")
}
func (s *storeImpl) MarkUnitRunning(ctx context.Context, jobId, wzFile string) (bool, error) {
	return false, errors.New("not implemented")
}
func (s *storeImpl) FinalizeUnit(ctx context.Context, jobId, wzFile string, terminal UnitStatus, runErr error) (Counters, error) {
	return Counters{}, errors.New("not implemented")
}
func (s *storeImpl) MarkJobTerminal(ctx context.Context, jobId string, terminal JobStatus) (bool, error) {
	return false, errors.New("not implemented")
}
func (s *storeImpl) MarkUnitsSkippedByStatus(ctx context.Context, jobId string, fromStatuses []UnitStatus) error {
	return errors.New("not implemented")
}
```

- [ ] **Step 5.4: Run tests, expect PASS**

```bash
cd services/atlas-wz-extractor/atlas.com/wz-extractor && go test ./extraction/job/ -run "TestStore_CreateGetDelete" -v
```

Expected: PASS.

- [ ] **Step 5.5: Commit**

```bash
git add services/atlas-wz-extractor/atlas.com/wz-extractor/extraction/job/store.go services/atlas-wz-extractor/atlas.com/wz-extractor/extraction/job/store_test.go
git commit -m "feat(atlas-wz-extractor): job.Store Create/Get/Delete with miniredis tests"
```

---

## Task 6: `Store.MarkJobRunning`

**Files:**
- Modify: `services/atlas-wz-extractor/atlas.com/wz-extractor/extraction/job/store.go`
- Modify: `services/atlas-wz-extractor/atlas.com/wz-extractor/extraction/job/store_test.go`

- [ ] **Step 6.1: Write the failing test**

Append to `store_test.go`:

```go
func TestStore_MarkJobRunning(t *testing.T) {
	ctx := context.Background()
	c := newTestClient(t)
	s := NewStore(c)

	now := time.Now().UTC().Truncate(time.Second)
	j := NewJobBuilder().SetId("j2").SetStatus(JobPending).
		SetUnitsTotal(1).SetCreatedAt(now).SetUpdatedAt(now).Build()
	if err := s.Create(ctx, j, []Unit{NewUnitBuilder().SetWzFile("Map.wz").SetStatus(UnitPending).Build()}, 3600); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := s.MarkJobRunning(ctx, "j2"); err != nil {
		t.Fatalf("MarkJobRunning: %v", err)
	}

	got, _, err := s.Get(ctx, "j2")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Status() != JobRunning {
		t.Fatalf("status: got %s", got.Status())
	}
}
```

- [ ] **Step 6.2: Run test, expect FAIL (returns "not implemented")**

```bash
cd services/atlas-wz-extractor/atlas.com/wz-extractor && go test ./extraction/job/ -run "TestStore_MarkJobRunning" -v
```

Expected: FAIL.

- [ ] **Step 6.3: Replace stub with implementation**

```go
func (s *storeImpl) MarkJobRunning(ctx context.Context, jobId string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	pipe := s.client.TxPipeline()
	pipe.HSet(ctx, jobKey(jobId), "status", string(JobRunning), "updatedAt", now)
	_, err := pipe.Exec(ctx)
	return err
}
```

- [ ] **Step 6.4: Run test, expect PASS**

```bash
cd services/atlas-wz-extractor/atlas.com/wz-extractor && go test ./extraction/job/ -run "TestStore_MarkJobRunning" -v
```

Expected: PASS.

- [ ] **Step 6.5: Commit**

```bash
git add services/atlas-wz-extractor/atlas.com/wz-extractor/extraction/job/store.go services/atlas-wz-extractor/atlas.com/wz-extractor/extraction/job/store_test.go
git commit -m "feat(atlas-wz-extractor): job.Store.MarkJobRunning"
```

---

## Task 7: `Store.MarkUnitRunning` (idempotent transition)

**Files:**
- Modify: `services/atlas-wz-extractor/atlas.com/wz-extractor/extraction/job/store.go`
- Modify: `services/atlas-wz-extractor/atlas.com/wz-extractor/extraction/job/store_test.go`

- [ ] **Step 7.1: Write the failing tests**

Append to `store_test.go`:

```go
func TestStore_MarkUnitRunning_FirstTime(t *testing.T) {
	ctx := context.Background()
	c := newTestClient(t)
	s := NewStore(c)
	seedJob(t, ctx, s, "j3", []string{"Map.wz"})

	claimed, err := s.MarkUnitRunning(ctx, "j3", "Map.wz")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !claimed {
		t.Fatalf("expected claimed=true on first transition")
	}
}

func TestStore_MarkUnitRunning_AlreadyTerminal(t *testing.T) {
	ctx := context.Background()
	c := newTestClient(t)
	s := NewStore(c)
	seedJob(t, ctx, s, "j4", []string{"Map.wz"})

	// Manually set the unit to terminal.
	raw, _ := unitToJSON(NewUnitBuilder().SetWzFile("Map.wz").SetStatus(UnitSucceeded).Build())
	if err := c.HSet(ctx, unitsKey("j4"), "Map.wz", raw).Err(); err != nil {
		t.Fatalf("seed terminal: %v", err)
	}

	claimed, err := s.MarkUnitRunning(ctx, "j4", "Map.wz")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if claimed {
		t.Fatalf("expected claimed=false on already-terminal unit (redelivery)")
	}
}

// helper used by store tests
func seedJob(t *testing.T, ctx context.Context, s Store, id string, files []string) {
	t.Helper()
	now := time.Now().UTC().Truncate(time.Second)
	j := NewJobBuilder().SetId(id).SetTenantId("t").SetRegion("GMS").
		SetMajorVersion(83).SetMinorVersion(1).
		SetStatus(JobRunning).
		SetUnitsTotal(len(files)).
		SetCreatedAt(now).SetUpdatedAt(now).Build()
	units := make([]Unit, 0, len(files))
	for _, f := range files {
		units = append(units, NewUnitBuilder().SetWzFile(f).SetStatus(UnitPending).Build())
	}
	if err := s.Create(ctx, j, units, 3600); err != nil {
		t.Fatalf("seed Create: %v", err)
	}
}
```

- [ ] **Step 7.2: Run tests, expect FAIL**

```bash
cd services/atlas-wz-extractor/atlas.com/wz-extractor && go test ./extraction/job/ -run "TestStore_MarkUnitRunning" -v
```

Expected: FAIL ("not implemented").

- [ ] **Step 7.3: Replace stub**

```go
func (s *storeImpl) MarkUnitRunning(ctx context.Context, jobId, wzFile string) (bool, error) {
	uKey := unitsKey(jobId)

	var claimed bool
	txn := func(tx *goredis.Tx) error {
		raw, err := tx.HGet(ctx, uKey, wzFile).Result()
		if err != nil && err != goredis.Nil {
			return err
		}
		if err == goredis.Nil {
			return ErrNotFound
		}
		u, err := unitFromJSON(wzFile, raw)
		if err != nil {
			return err
		}
		if u.Status() == UnitSucceeded || u.Status() == UnitFailed || u.Status() == UnitSkipped {
			claimed = false
			return nil
		}
		nu := NewUnitBuilder().SetWzFile(wzFile).SetStatus(UnitRunning).
			SetStartedAt(time.Now().UTC()).Build()
		nraw, err := unitToJSON(nu)
		if err != nil {
			return err
		}
		_, err = tx.TxPipelined(ctx, func(p goredis.Pipeliner) error {
			p.HSet(ctx, uKey, wzFile, nraw)
			p.HSet(ctx, jobKey(jobId), "updatedAt", time.Now().UTC().Format(time.RFC3339))
			return nil
		})
		if err == nil {
			claimed = true
		}
		return err
	}

	for attempt := 0; attempt < 5; attempt++ {
		err := s.client.Watch(ctx, txn, uKey)
		if err == nil {
			return claimed, nil
		}
		if err == goredis.TxFailedErr {
			continue
		}
		return false, err
	}
	return false, errors.New("MarkUnitRunning: too many WATCH retries")
}
```

- [ ] **Step 7.4: Run tests, expect PASS**

```bash
cd services/atlas-wz-extractor/atlas.com/wz-extractor && go test ./extraction/job/ -run "TestStore_MarkUnitRunning" -v
```

Expected: PASS.

- [ ] **Step 7.5: Commit**

```bash
git add services/atlas-wz-extractor/atlas.com/wz-extractor/extraction/job/store.go services/atlas-wz-extractor/atlas.com/wz-extractor/extraction/job/store_test.go
git commit -m "feat(atlas-wz-extractor): job.Store.MarkUnitRunning with WATCH guard"
```

---

## Task 8: `Store.FinalizeUnit` (atomic transition + counter increment)

**Files:**
- Modify: `services/atlas-wz-extractor/atlas.com/wz-extractor/extraction/job/store.go`
- Modify: `services/atlas-wz-extractor/atlas.com/wz-extractor/extraction/job/store_test.go`

- [ ] **Step 8.1: Write the failing tests**

Append to `store_test.go`:

```go
func TestStore_FinalizeUnit_Succeeded(t *testing.T) {
	ctx := context.Background()
	c := newTestClient(t)
	s := NewStore(c)
	seedJob(t, ctx, s, "j5", []string{"Map.wz", "Mob.wz"})
	if _, err := s.MarkUnitRunning(ctx, "j5", "Map.wz"); err != nil {
		t.Fatalf("MarkUnitRunning: %v", err)
	}

	cnt, err := s.FinalizeUnit(ctx, "j5", "Map.wz", UnitSucceeded, nil)
	if err != nil {
		t.Fatalf("FinalizeUnit: %v", err)
	}
	if cnt.UnitsCompleted != 1 || cnt.UnitsFailed != 0 || cnt.UnitsTotal != 2 || cnt.AllDone {
		t.Fatalf("counters: %+v", cnt)
	}

	got, units, _ := s.Get(ctx, "j5")
	if got.UnitsCompleted() != 1 {
		t.Fatalf("Get unitsCompleted: %d", got.UnitsCompleted())
	}
	for _, u := range units {
		if u.WzFile() == "Map.wz" && u.Status() != UnitSucceeded {
			t.Fatalf("unit not succeeded: %v", u.Status())
		}
	}
}

func TestStore_FinalizeUnit_Failed(t *testing.T) {
	ctx := context.Background()
	c := newTestClient(t)
	s := NewStore(c)
	seedJob(t, ctx, s, "j6", []string{"Map.wz"})
	if _, err := s.MarkUnitRunning(ctx, "j6", "Map.wz"); err != nil {
		t.Fatalf("MarkUnitRunning: %v", err)
	}

	cnt, err := s.FinalizeUnit(ctx, "j6", "Map.wz", UnitFailed, errors.New("open failed"))
	if err != nil {
		t.Fatalf("FinalizeUnit: %v", err)
	}
	if cnt.UnitsFailed != 1 || cnt.UnitsCompleted != 0 || !cnt.AllDone {
		t.Fatalf("counters: %+v", cnt)
	}
}

func TestStore_FinalizeUnit_RedeliveryNoOp(t *testing.T) {
	ctx := context.Background()
	c := newTestClient(t)
	s := NewStore(c)
	seedJob(t, ctx, s, "j7", []string{"Map.wz"})
	if _, err := s.MarkUnitRunning(ctx, "j7", "Map.wz"); err != nil {
		t.Fatalf("MarkUnitRunning: %v", err)
	}
	if _, err := s.FinalizeUnit(ctx, "j7", "Map.wz", UnitSucceeded, nil); err != nil {
		t.Fatalf("first finalize: %v", err)
	}

	// Redelivery: a second FinalizeUnit with the unit already terminal must
	// not double-increment counters.
	cnt, err := s.FinalizeUnit(ctx, "j7", "Map.wz", UnitSucceeded, nil)
	if err != nil {
		t.Fatalf("redelivery finalize: %v", err)
	}
	if cnt.UnitsCompleted != 1 {
		t.Fatalf("redelivery double-counted: %+v", cnt)
	}
}
```

- [ ] **Step 8.2: Run tests, expect FAIL**

```bash
cd services/atlas-wz-extractor/atlas.com/wz-extractor && go test ./extraction/job/ -run "TestStore_FinalizeUnit" -v
```

Expected: FAIL.

- [ ] **Step 8.3: Replace stub**

```go
func (s *storeImpl) FinalizeUnit(ctx context.Context, jobId, wzFile string, terminal UnitStatus, runErr error) (Counters, error) {
	if terminal != UnitSucceeded && terminal != UnitFailed {
		return Counters{}, errors.New("FinalizeUnit: terminal must be Succeeded or Failed")
	}
	jKey := jobKey(jobId)
	uKey := unitsKey(jobId)

	var out Counters
	txn := func(tx *goredis.Tx) error {
		raw, err := tx.HGet(ctx, uKey, wzFile).Result()
		if err == goredis.Nil {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		u, err := unitFromJSON(wzFile, raw)
		if err != nil {
			return err
		}

		// Already terminal (redelivery): read counters, return them, no-op.
		if u.Status() == UnitSucceeded || u.Status() == UnitFailed || u.Status() == UnitSkipped {
			j, _, gerr := s.Get(ctx, jobId)
			if gerr != nil {
				return gerr
			}
			out = Counters{
				UnitsTotal:     j.UnitsTotal(),
				UnitsCompleted: j.UnitsCompleted(),
				UnitsFailed:    j.UnitsFailed(),
				AllDone:        (j.UnitsCompleted() + j.UnitsFailed()) == j.UnitsTotal(),
				LockKey:        LockKey(j.TenantId(), j.Region(), j.MajorVersion(), j.MinorVersion()),
			}
			return nil
		}

		nb := NewUnitBuilder().SetWzFile(wzFile).SetStatus(terminal).
			SetStartedAt(u.StartedAt()).
			SetCompletedAt(time.Now().UTC())
		if runErr != nil {
			nb = nb.SetErrorMessage(runErr.Error())
		}
		nraw, err := unitToJSON(nb.Build())
		if err != nil {
			return err
		}

		field := "unitsCompleted"
		if terminal == UnitFailed {
			field = "unitsFailed"
		}

		var totalCmd, completedCmd, failedCmd, tenantCmd, regionCmd, majCmd, minCmd *goredis.StringCmd
		var newCounter *goredis.IntCmd
		_, err = tx.TxPipelined(ctx, func(p goredis.Pipeliner) error {
			p.HSet(ctx, uKey, wzFile, nraw)
			newCounter = p.HIncrBy(ctx, jKey, field, 1)
			p.HSet(ctx, jKey, "updatedAt", time.Now().UTC().Format(time.RFC3339))
			totalCmd = p.HGet(ctx, jKey, "unitsTotal")
			completedCmd = p.HGet(ctx, jKey, "unitsCompleted")
			failedCmd = p.HGet(ctx, jKey, "unitsFailed")
			tenantCmd = p.HGet(ctx, jKey, "tenantId")
			regionCmd = p.HGet(ctx, jKey, "region")
			majCmd = p.HGet(ctx, jKey, "majorVersion")
			minCmd = p.HGet(ctx, jKey, "minorVersion")
			return nil
		})
		if err != nil {
			return err
		}
		_ = newCounter // increment already applied above; counters re-read post-EXEC

		total, _ := strconv.Atoi(totalCmd.Val())
		completed, _ := strconv.Atoi(completedCmd.Val())
		failed, _ := strconv.Atoi(failedCmd.Val())
		maj, _ := strconv.Atoi(majCmd.Val())
		min, _ := strconv.Atoi(minCmd.Val())
		out = Counters{
			UnitsTotal:     total,
			UnitsCompleted: completed,
			UnitsFailed:    failed,
			AllDone:        (completed + failed) == total,
			LockKey:        LockKey(tenantCmd.Val(), regionCmd.Val(), uint16(maj), uint16(min)),
		}
		return nil
	}

	for attempt := 0; attempt < 5; attempt++ {
		err := s.client.Watch(ctx, txn, uKey)
		if err == nil {
			return out, nil
		}
		if err == goredis.TxFailedErr {
			continue
		}
		return Counters{}, err
	}
	return Counters{}, errors.New("FinalizeUnit: too many WATCH retries")
}
```

- [ ] **Step 8.4: Run tests, expect PASS**

```bash
cd services/atlas-wz-extractor/atlas.com/wz-extractor && go test ./extraction/job/ -run "TestStore_FinalizeUnit" -v
```

Expected: PASS.

- [ ] **Step 8.5: Commit**

```bash
git add services/atlas-wz-extractor/atlas.com/wz-extractor/extraction/job/store.go services/atlas-wz-extractor/atlas.com/wz-extractor/extraction/job/store_test.go
git commit -m "feat(atlas-wz-extractor): job.Store.FinalizeUnit with redelivery guard"
```

---

## Task 9: `Store.MarkJobTerminal` (CAS for last-one-home)

**Files:**
- Modify: `services/atlas-wz-extractor/atlas.com/wz-extractor/extraction/job/store.go`
- Modify: `services/atlas-wz-extractor/atlas.com/wz-extractor/extraction/job/store_test.go`

- [ ] **Step 9.1: Write the failing tests**

Append to `store_test.go`:

```go
func TestStore_MarkJobTerminal_Once(t *testing.T) {
	ctx := context.Background()
	c := newTestClient(t)
	s := NewStore(c)
	seedJob(t, ctx, s, "j8", []string{"Map.wz"})
	if err := s.MarkJobRunning(ctx, "j8"); err != nil {
		t.Fatalf("MarkJobRunning: %v", err)
	}

	claimed1, err := s.MarkJobTerminal(ctx, "j8", JobCompleted)
	if err != nil {
		t.Fatalf("first MarkJobTerminal: %v", err)
	}
	if !claimed1 {
		t.Fatalf("expected first call to claim")
	}

	claimed2, err := s.MarkJobTerminal(ctx, "j8", JobCompleted)
	if err != nil {
		t.Fatalf("second MarkJobTerminal: %v", err)
	}
	if claimed2 {
		t.Fatalf("expected second call to NOT claim")
	}

	got, _, _ := s.Get(ctx, "j8")
	if got.Status() != JobCompleted {
		t.Fatalf("status: %s", got.Status())
	}
	if got.CompletedAt().IsZero() {
		t.Fatalf("completedAt not set")
	}
}
```

- [ ] **Step 9.2: Run tests, expect FAIL**

```bash
cd services/atlas-wz-extractor/atlas.com/wz-extractor && go test ./extraction/job/ -run "TestStore_MarkJobTerminal" -v
```

Expected: FAIL.

- [ ] **Step 9.3: Replace stub**

```go
func (s *storeImpl) MarkJobTerminal(ctx context.Context, jobId string, terminal JobStatus) (bool, error) {
	jKey := jobKey(jobId)

	var claimed bool
	txn := func(tx *goredis.Tx) error {
		cur, err := tx.HGet(ctx, jKey, "status").Result()
		if err == goredis.Nil {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if JobStatus(cur) != JobRunning {
			claimed = false
			return nil
		}
		now := time.Now().UTC().Format(time.RFC3339)
		_, err = tx.TxPipelined(ctx, func(p goredis.Pipeliner) error {
			p.HSet(ctx, jKey, "status", string(terminal), "updatedAt", now, "completedAt", now)
			return nil
		})
		if err == nil {
			claimed = true
		}
		return err
	}
	for attempt := 0; attempt < 5; attempt++ {
		err := s.client.Watch(ctx, txn, jKey)
		if err == nil {
			return claimed, nil
		}
		if err == goredis.TxFailedErr {
			continue
		}
		return false, err
	}
	return false, errors.New("MarkJobTerminal: too many WATCH retries")
}
```

- [ ] **Step 9.4: Run tests, expect PASS**

```bash
cd services/atlas-wz-extractor/atlas.com/wz-extractor && go test ./extraction/job/ -run "TestStore_MarkJobTerminal" -v
```

Expected: PASS.

- [ ] **Step 9.5: Commit**

```bash
git add services/atlas-wz-extractor/atlas.com/wz-extractor/extraction/job/store.go services/atlas-wz-extractor/atlas.com/wz-extractor/extraction/job/store_test.go
git commit -m "feat(atlas-wz-extractor): job.Store.MarkJobTerminal CAS"
```

---

## Task 10: `Store.MarkUnitsSkippedByStatus`

Used by the dispatcher when it errors after partial publish: any unit still `pending` (no message published yet) is moved to `skipped` so the in-flight units can finalize correctly.

**Files:**
- Modify: `services/atlas-wz-extractor/atlas.com/wz-extractor/extraction/job/store.go`
- Modify: `services/atlas-wz-extractor/atlas.com/wz-extractor/extraction/job/store_test.go`

- [ ] **Step 10.1: Write the failing test**

```go
func TestStore_MarkUnitsSkippedByStatus(t *testing.T) {
	ctx := context.Background()
	c := newTestClient(t)
	s := NewStore(c)
	seedJob(t, ctx, s, "j9", []string{"Map.wz", "Mob.wz", "Item.wz"})
	if _, err := s.MarkUnitRunning(ctx, "j9", "Mob.wz"); err != nil {
		t.Fatalf("MarkUnitRunning: %v", err)
	}

	if err := s.MarkUnitsSkippedByStatus(ctx, "j9", []UnitStatus{UnitPending}); err != nil {
		t.Fatalf("MarkUnitsSkippedByStatus: %v", err)
	}

	_, units, err := s.Get(ctx, "j9")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	statuses := map[string]UnitStatus{}
	for _, u := range units {
		statuses[u.WzFile()] = u.Status()
	}
	if statuses["Map.wz"] != UnitSkipped {
		t.Fatalf("Map.wz: %s", statuses["Map.wz"])
	}
	if statuses["Item.wz"] != UnitSkipped {
		t.Fatalf("Item.wz: %s", statuses["Item.wz"])
	}
	if statuses["Mob.wz"] != UnitRunning {
		t.Fatalf("Mob.wz must NOT have been skipped: %s", statuses["Mob.wz"])
	}
}
```

- [ ] **Step 10.2: Run test, expect FAIL**

```bash
cd services/atlas-wz-extractor/atlas.com/wz-extractor && go test ./extraction/job/ -run "TestStore_MarkUnitsSkippedByStatus" -v
```

Expected: FAIL.

- [ ] **Step 10.3: Replace stub**

```go
func (s *storeImpl) MarkUnitsSkippedByStatus(ctx context.Context, jobId string, fromStatuses []UnitStatus) error {
	allow := map[UnitStatus]bool{}
	for _, st := range fromStatuses {
		allow[st] = true
	}
	uKey := unitsKey(jobId)
	rawAll, err := s.client.HGetAll(ctx, uKey).Result()
	if err != nil {
		return err
	}
	pipe := s.client.TxPipeline()
	now := time.Now().UTC()
	changed := 0
	for wzFile, raw := range rawAll {
		u, err := unitFromJSON(wzFile, raw)
		if err != nil {
			continue
		}
		if !allow[u.Status()] {
			continue
		}
		nu := NewUnitBuilder().SetWzFile(wzFile).SetStatus(UnitSkipped).
			SetStartedAt(u.StartedAt()).SetCompletedAt(now).Build()
		nraw, err := unitToJSON(nu)
		if err != nil {
			continue
		}
		pipe.HSet(ctx, uKey, wzFile, nraw)
		changed++
	}
	if changed == 0 {
		return nil
	}
	pipe.HSet(ctx, jobKey(jobId), "updatedAt", now.Format(time.RFC3339))
	_, err = pipe.Exec(ctx)
	return err
}
```

- [ ] **Step 10.4: Run test, expect PASS**

```bash
cd services/atlas-wz-extractor/atlas.com/wz-extractor && go test ./extraction/job/ -run "TestStore_MarkUnitsSkippedByStatus" -v
```

Expected: PASS.

- [ ] **Step 10.5: Run full job package test suite**

```bash
cd services/atlas-wz-extractor/atlas.com/wz-extractor && go test ./extraction/job/ -v
```

Expected: all PASS.

- [ ] **Step 10.6: Commit**

```bash
git add services/atlas-wz-extractor/atlas.com/wz-extractor/extraction/job/store.go services/atlas-wz-extractor/atlas.com/wz-extractor/extraction/job/store_test.go
git commit -m "feat(atlas-wz-extractor): job.Store.MarkUnitsSkippedByStatus"
```

---

## Task 11: `extraction/lock` package — TenantLock

**Files:**
- Create: `services/atlas-wz-extractor/atlas.com/wz-extractor/extraction/lock/tenant_lock.go`
- Create: `services/atlas-wz-extractor/atlas.com/wz-extractor/extraction/lock/tenant_lock_test.go`

- [ ] **Step 11.1: Write the failing tests**

```go
package lock

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
)

func newTestClient(t *testing.T) (*goredis.Client, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	c := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	return c, mr
}

func TestTenantLock_AcquireRelease(t *testing.T) {
	ctx := context.Background()
	c, _ := newTestClient(t)
	tl := NewTenantLock(c, time.Minute)

	ok1, err := tl.Acquire(ctx, "key1", "owner-A")
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	if !ok1 {
		t.Fatalf("expected first Acquire to succeed")
	}

	ok2, err := tl.Acquire(ctx, "key1", "owner-B")
	if err != nil {
		t.Fatalf("Acquire 2: %v", err)
	}
	if ok2 {
		t.Fatalf("expected second Acquire to fail (held)")
	}

	if err := tl.Release(ctx, "key1", "owner-A"); err != nil {
		t.Fatalf("Release: %v", err)
	}

	ok3, err := tl.Acquire(ctx, "key1", "owner-B")
	if err != nil {
		t.Fatalf("Acquire 3: %v", err)
	}
	if !ok3 {
		t.Fatalf("expected re-Acquire after Release to succeed")
	}
}

func TestTenantLock_ReleaseOnlyOwner(t *testing.T) {
	ctx := context.Background()
	c, _ := newTestClient(t)
	tl := NewTenantLock(c, time.Minute)

	if _, err := tl.Acquire(ctx, "key2", "owner-A"); err != nil {
		t.Fatalf("Acquire: %v", err)
	}

	// Different owner attempts release; lock must remain held.
	if err := tl.Release(ctx, "key2", "owner-B"); err != nil {
		t.Fatalf("Release wrong owner: %v", err)
	}

	ok, err := tl.Acquire(ctx, "key2", "owner-C")
	if err != nil {
		t.Fatalf("Acquire after wrong-owner release: %v", err)
	}
	if ok {
		t.Fatalf("lock should still be held after wrong-owner Release")
	}
}

func TestTenantLock_RefreshExtendsTTL(t *testing.T) {
	ctx := context.Background()
	c, mr := newTestClient(t)
	tl := NewTenantLock(c, 10*time.Second)

	if _, err := tl.Acquire(ctx, "key3", "owner-A"); err != nil {
		t.Fatalf("Acquire: %v", err)
	}

	// Advance miniredis time so the original TTL would have nearly expired.
	mr.FastForward(8 * time.Second)
	if err := tl.Refresh(ctx, "key3", "owner-A"); err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	mr.FastForward(8 * time.Second)
	// TTL was reset; lock should still be held.
	ok, err := tl.Acquire(ctx, "key3", "owner-B")
	if err != nil {
		t.Fatalf("Acquire after refresh: %v", err)
	}
	if ok {
		t.Fatalf("expected lock still held after Refresh")
	}
}
```

- [ ] **Step 11.2: Run tests, expect FAIL (package missing)**

```bash
cd services/atlas-wz-extractor/atlas.com/wz-extractor && go test ./extraction/lock/ -v
```

Expected: build error.

- [ ] **Step 11.3: Implement `tenant_lock.go`**

```go
package lock

import (
	"context"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// TenantLock wraps Redis SETNX-with-owner semantics for the per-tenant
// extraction lock. The Lua compare-and-delete on Release prevents a stale
// holder (whose TTL just expired) from accidentally releasing a lock that has
// since been re-acquired by a different job.
type TenantLock struct {
	client *goredis.Client
	ttl    time.Duration
}

func NewTenantLock(client *goredis.Client, ttl time.Duration) *TenantLock {
	return &TenantLock{client: client, ttl: ttl}
}

func (t *TenantLock) TTL() time.Duration { return t.ttl }

// Acquire attempts SET NX EX. Returns (true, nil) when the lock is now held
// with `owner` as the value, (false, nil) when held by someone else.
func (t *TenantLock) Acquire(ctx context.Context, key, owner string) (bool, error) {
	ok, err := t.client.SetNX(ctx, key, owner, t.ttl).Result()
	if err != nil {
		return false, err
	}
	return ok, nil
}

// Refresh extends the TTL only if the caller still owns the lock.
const refreshLua = `
if redis.call("GET", KEYS[1]) == ARGV[1] then
  return redis.call("EXPIRE", KEYS[1], ARGV[2])
else
  return 0
end
`

func (t *TenantLock) Refresh(ctx context.Context, key, owner string) error {
	secs := int64(t.ttl / time.Second)
	if secs <= 0 {
		secs = 1
	}
	return t.client.Eval(ctx, refreshLua, []string{key}, owner, secs).Err()
}

// Release deletes the lock only if the value matches `owner`.
const releaseLua = `
if redis.call("GET", KEYS[1]) == ARGV[1] then
  return redis.call("DEL", KEYS[1])
else
  return 0
end
`

func (t *TenantLock) Release(ctx context.Context, key, owner string) error {
	return t.client.Eval(ctx, releaseLua, []string{key}, owner).Err()
}
```

- [ ] **Step 11.4: Run tests, expect PASS**

```bash
cd services/atlas-wz-extractor/atlas.com/wz-extractor && go test ./extraction/lock/ -v
```

Expected: PASS.

- [ ] **Step 11.5: Commit**

```bash
git add services/atlas-wz-extractor/atlas.com/wz-extractor/extraction/lock/
git commit -m "feat(atlas-wz-extractor): TenantLock with owner-match Refresh and Release"
```

---

## Task 12: `extraction/pool.go` — bounded worker pool

**Files:**
- Create: `services/atlas-wz-extractor/atlas.com/wz-extractor/extraction/pool.go`
- Create: `services/atlas-wz-extractor/atlas.com/wz-extractor/extraction/pool_test.go`

- [ ] **Step 12.1: Write the failing test**

```go
package extraction

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sirupsen/logrus/hooks/test"
)

func TestRunPool_BoundsConcurrency(t *testing.T) {
	l, _ := test.NewNullLogger()
	const N = 32
	const workers = 4

	var inflight int32
	var maxInflight int32

	jobs := make([]string, N)
	for i := range jobs {
		jobs[i] = "wz"
	}

	worker := func(ctx context.Context, _ string) error {
		cur := atomic.AddInt32(&inflight, 1)
		for {
			prev := atomic.LoadInt32(&maxInflight)
			if cur <= prev || atomic.CompareAndSwapInt32(&maxInflight, prev, cur) {
				break
			}
		}
		time.Sleep(2 * time.Millisecond)
		atomic.AddInt32(&inflight, -1)
		return nil
	}

	runPool(context.Background(), l, jobs, workers, worker)

	if got := atomic.LoadInt32(&maxInflight); got > int32(workers) {
		t.Fatalf("maxInflight=%d exceeded workers=%d", got, workers)
	}
}

func TestRunPool_ContinuesOnError(t *testing.T) {
	l, _ := test.NewNullLogger()
	jobs := []string{"a", "b", "c"}
	var ran int32
	var mu sync.Mutex
	worker := func(ctx context.Context, j string) error {
		mu.Lock()
		atomic.AddInt32(&ran, 1)
		mu.Unlock()
		if j == "b" {
			return context.Canceled // stand-in for a per-unit error
		}
		return nil
	}
	runPool(context.Background(), l, jobs, 2, worker)
	if atomic.LoadInt32(&ran) != 3 {
		t.Fatalf("expected all 3 to run, ran=%d", ran)
	}
}
```

- [ ] **Step 12.2: Run test, expect FAIL (function missing)**

```bash
cd services/atlas-wz-extractor/atlas.com/wz-extractor && go test ./extraction/ -run "TestRunPool" -v
```

Expected: FAIL.

- [ ] **Step 12.3: Implement `pool.go`**

```go
package extraction

import (
	"context"
	"sync"

	"github.com/sirupsen/logrus"
)

// runPool runs `worker` over each job in `jobs` with at most `workers` in
// flight. A per-job error does not abort sibling work — the worker logs the
// error and the pool continues. The function returns when all jobs complete or
// ctx is cancelled.
//
// This is the in-process fan-out path used by Extract (whole-list). The
// cross-pod path uses Kafka partition assignment instead.
func runPool[T any](ctx context.Context, l logrus.FieldLogger, jobs []T, workers int, worker func(context.Context, T) error) {
	if workers < 1 {
		workers = 1
	}
	ch := make(chan T)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(workerId int) {
			defer wg.Done()
			wl := l.WithField("worker", workerId)
			for j := range ch {
				if ctx.Err() != nil {
					return
				}
				if err := worker(ctx, j); err != nil {
					wl.WithError(err).Warn("pool worker returned error; continuing")
				}
			}
		}(i)
	}
	for _, j := range jobs {
		if ctx.Err() != nil {
			break
		}
		ch <- j
	}
	close(ch)
	wg.Wait()
}
```

- [ ] **Step 12.4: Run tests, expect PASS**

```bash
cd services/atlas-wz-extractor/atlas.com/wz-extractor && go test ./extraction/ -run "TestRunPool" -v
```

Expected: PASS.

- [ ] **Step 12.5: Commit**

```bash
git add services/atlas-wz-extractor/atlas.com/wz-extractor/extraction/pool.go services/atlas-wz-extractor/atlas.com/wz-extractor/extraction/pool_test.go
git commit -m "feat(atlas-wz-extractor): bounded worker pool for in-process fan-out"
```

---

## Task 13: Refactor `processor.go` — split `Extract` into `ExtractUnit` + pool-backed `Extract`

**Files:**
- Modify: `services/atlas-wz-extractor/atlas.com/wz-extractor/extraction/processor.go`
- Modify: `services/atlas-wz-extractor/atlas.com/wz-extractor/extraction/processor_test.go`

- [ ] **Step 13.1: Write the failing test for `ExtractUnit`**

Append to `extraction/processor_test.go`:

```go
func TestExtractUnit_FailsWhenWzCannotBeOpened(t *testing.T) {
	tenantId := uuid.New()
	tt, _ := tenant.Create(tenantId, "GMS", 83, 1)
	ctx := tenant.WithContext(context.Background(), tt)

	inputDir := t.TempDir()
	xmlDir := t.TempDir()
	imgDir := t.TempDir()
	tenantInput := filepath.Join(inputDir, tenantId.String(), "GMS", "83.1")
	if err := os.MkdirAll(tenantInput, 0o755); err != nil {
		t.Fatal(err)
	}
	bad := filepath.Join(tenantInput, "Bad.wz")
	if err := os.WriteFile(bad, []byte("not a real wz"), 0o644); err != nil {
		t.Fatal(err)
	}

	p := NewProcessor(inputDir, xmlDir, imgDir)
	l, _ := test.NewNullLogger()
	if err := p.ExtractUnit(l, ctx, "Bad.wz", false, false); err == nil {
		t.Fatalf("expected non-nil error when wz.Open fails")
	}
}

func TestExtractUnit_RejectsMissingWzFile(t *testing.T) {
	tenantId := uuid.New()
	tt, _ := tenant.Create(tenantId, "GMS", 83, 1)
	ctx := tenant.WithContext(context.Background(), tt)
	p := NewProcessor(t.TempDir(), t.TempDir(), t.TempDir())
	l, _ := test.NewNullLogger()
	if err := p.ExtractUnit(l, ctx, "Nope.wz", false, false); err == nil {
		t.Fatalf("expected error for missing wz file")
	}
}
```

- [ ] **Step 13.2: Run test, expect FAIL (method missing)**

```bash
cd services/atlas-wz-extractor/atlas.com/wz-extractor && go test ./extraction/ -run "TestExtractUnit" -v
```

Expected: FAIL.

- [ ] **Step 13.3: Replace `processor.go`**

```go
package extraction

import (
	wzimage "atlas-wz-extractor/image"
	"atlas-wz-extractor/wz"
	wzxml "atlas-wz-extractor/xml"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"

	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const envParallelism = "WZ_EXTRACT_PARALLELISM"

type Processor interface {
	// Extract preserves today's entry-point: list every WZ file under the
	// tenant's input dir, wipe the character cache (unless xmlOnly), and
	// process them in parallel via a bounded worker pool. Used by the
	// in-process tests; the cross-pod path uses ExtractUnit through Kafka.
	Extract(l logrus.FieldLogger, ctx context.Context, xmlOnly, imagesOnly bool) error

	// ExtractUnit processes one WZ file. Returns non-nil error only when
	// wz.Open fails ("couldn't even open the file"); per-stage errors are
	// logged but do not flip the unit to failed (continue-on-error semantics
	// from the original whole-list loop).
	ExtractUnit(l logrus.FieldLogger, ctx context.Context, wzFile string, xmlOnly, imagesOnly bool) error
}

type processorImpl struct {
	inputDir     string
	outputXmlDir string
	outputImgDir string
}

func NewProcessor(inputDir, outputXmlDir, outputImgDir string) Processor {
	return &processorImpl{
		inputDir:     inputDir,
		outputXmlDir: outputXmlDir,
		outputImgDir: outputImgDir,
	}
}

func (p *processorImpl) Extract(l logrus.FieldLogger, ctx context.Context, xmlOnly, imagesOnly bool) error {
	t := tenant.MustFromContext(ctx)
	tenantPath := TenantPath(t)
	inputPath := filepath.Join(p.inputDir, tenantPath)

	wzFiles, err := filepath.Glob(filepath.Join(inputPath, "*.wz"))
	if err != nil {
		return fmt.Errorf("unable to list WZ files: %w", err)
	}
	if len(wzFiles) == 0 {
		return fmt.Errorf("no WZ files found in [%s]", inputPath)
	}
	l.Infof("Found [%d] WZ files in [%s].", len(wzFiles), inputPath)

	if !xmlOnly {
		imgOutPath := filepath.Join(p.outputImgDir, tenantPath)
		if err := wipeCharacterCache(imgOutPath); err != nil {
			l.WithError(err).Warnf("Unable to wipe character cache.")
		}
	}

	workers := parallelismFromEnv(l)
	files := make([]string, 0, len(wzFiles))
	for _, full := range wzFiles {
		files = append(files, filepath.Base(full))
	}

	runPool(ctx, l, files, workers, func(c context.Context, wz string) error {
		return p.ExtractUnit(l, c, wz, xmlOnly, imagesOnly)
	})
	return nil
}

func (p *processorImpl) ExtractUnit(l logrus.FieldLogger, ctx context.Context, wzFile string, xmlOnly, imagesOnly bool) error {
	t := tenant.MustFromContext(ctx)
	tenantPath := TenantPath(t)
	inputPath := filepath.Join(p.inputDir, tenantPath)
	xmlOutPath := filepath.Join(p.outputXmlDir, tenantPath)
	imgOutPath := filepath.Join(p.outputImgDir, tenantPath)
	wzPath := filepath.Join(inputPath, wzFile)

	l = l.WithField("wzFile", wzFile)
	l.Info("processing wz unit")

	f, err := wz.Open(l, wzPath)
	if err != nil {
		return fmt.Errorf("unable to open WZ file [%s]: %w", wzFile, err)
	}
	defer f.Close()

	if !imagesOnly {
		if err := wzxml.SerializeToDirectory(l, f, xmlOutPath); err != nil {
			l.WithError(err).Errorf("Unable to serialize [%s] to XML.", wzFile)
		}
	}

	if !xmlOnly {
		if err := wzimage.ExtractIcons(l, f, imgOutPath); err != nil {
			l.WithError(err).Errorf("Unable to extract icons from [%s].", wzFile)
		}
		if err := wzimage.ExtractMinimaps(l, f, imgOutPath); err != nil {
			l.WithError(err).Errorf("Unable to extract minimaps from [%s].", wzFile)
		}
		if err := RenderMaps(ctx, l, f, imgOutPath); err != nil {
			l.WithError(err).Errorf("Unable to render maps from [%s].", wzFile)
		}
	}

	return nil
}

// parallelismFromEnv reads WZ_EXTRACT_PARALLELISM with a runtime.NumCPU()
// fallback. Invalid/zero values fall back to default and log a warning.
func parallelismFromEnv(l logrus.FieldLogger) int {
	v := os.Getenv(envParallelism)
	if v == "" {
		return runtime.NumCPU()
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		l.WithField("value", v).Warnf("invalid %s; using runtime.NumCPU()", envParallelism)
		return runtime.NumCPU()
	}
	return n
}

// wipeCharacterCache removes the {imgOut}/character directory so a fresh
// extraction does not serve stale renders against newly extracted assets.
func wipeCharacterCache(imgOut string) error {
	target := filepath.Join(imgOut, "character")
	if err := os.RemoveAll(target); err != nil {
		return fmt.Errorf("remove %s: %w", target, err)
	}
	return nil
}
```

- [ ] **Step 13.4: Update existing tests for the new shape**

`processor_test.go`'s `TestRunExtraction_NoWzFiles` and friends call the now-removed `runExtraction` — replace those with calls to `Extract` and assertions about the wrapping error shape.

In `processor_test.go`, replace:

```go
func TestRunExtraction_NoWzFiles(t *testing.T) {
	dir := t.TempDir()
	p := &processorImpl{inputDir: dir, outputXmlDir: t.TempDir(), outputImgDir: t.TempDir()}
	l, _ := test.NewNullLogger()
	err := p.runExtraction(context.Background(), l, dir, t.TempDir(), t.TempDir(), false, false)
	if err == nil {
		t.Fatal("expected error for empty input directory")
	}
	if got := err.Error(); got != "no WZ files found in ["+dir+"]" {
		t.Errorf("unexpected error: %s", got)
	}
}
```

with:

```go
func TestExtract_NoWzFiles(t *testing.T) {
	tenantId := uuid.New()
	tt, _ := tenant.Create(tenantId, "GMS", 83, 1)
	ctx := tenant.WithContext(context.Background(), tt)

	inputDir := t.TempDir()
	tenantInput := filepath.Join(inputDir, tenantId.String(), "GMS", "83.1")
	if err := os.MkdirAll(tenantInput, 0o755); err != nil {
		t.Fatal(err)
	}

	p := NewProcessor(inputDir, t.TempDir(), t.TempDir())
	l, _ := test.NewNullLogger()
	err := p.Extract(l, ctx, false, false)
	if err == nil || !strings.Contains(err.Error(), "no WZ files found") {
		t.Fatalf("expected no-WZ-files error, got %v", err)
	}
}
```

Add `"strings"` to the test imports.

Remove `TestRunExtraction_InvalidInputDir` and `TestRunExtraction_NoFallbackToFlatInput` (their assertions exercised the removed `runExtraction`; their behaviour is now covered by `TestExtract_NoWzFiles` plus the new `ExtractUnit` tests).

`TestRunExtractionWipesCharacterCache` is unchanged — `wipeCharacterCache` is still in this file.

`TestExtract_OutputPathConstruction` and `TestExtract_TenantPathFormat` continue to call `Extract` so they still work — leave them as-is.

- [ ] **Step 13.5: Run full extraction package tests, expect PASS**

```bash
cd services/atlas-wz-extractor/atlas.com/wz-extractor && go test ./extraction/ -v
```

Expected: PASS (still includes the not-yet-deleted `mutex_test.go` — those tests still pass while the file exists).

- [ ] **Step 13.6: Commit**

```bash
git add services/atlas-wz-extractor/atlas.com/wz-extractor/extraction/processor.go services/atlas-wz-extractor/atlas.com/wz-extractor/extraction/processor_test.go
git commit -m "refactor(atlas-wz-extractor): split Extract into ExtractUnit + pool-backed Extract"
```

---

## Task 14: Delete in-process tenant mutex

**Files:**
- Delete: `services/atlas-wz-extractor/atlas.com/wz-extractor/extraction/mutex.go`
- Delete: `services/atlas-wz-extractor/atlas.com/wz-extractor/extraction/mutex_test.go`

The `Acquire`/`TryAcquire`/`Release` symbols are referenced by `resource.go`'s current `handleExtract`. We replace that handler in Task 19; in this task we delete the helper *after* updating `resource.go` to no longer reference it.

- [ ] **Step 14.1: Verify nothing else imports the symbols outside `resource.go`**

```bash
cd services/atlas-wz-extractor/atlas.com/wz-extractor && grep -rn "extraction\.Acquire\|extraction\.TryAcquire\|extraction\.Release\|^\s*Acquire(\|TryAcquire(\|Release(" --include="*.go"
```

Expected: only matches inside `extraction/mutex.go`, `extraction/mutex_test.go`, and `extraction/resource.go`.

- [ ] **Step 14.2: Inline-update `resource.go` to drop the mutex calls in the existing handler (keep handler working temporarily)**

Replace the call sites inside `extraction/resource.go`'s `handleExtract`:

```go
				wg.Add(1)
				go func() {
					defer wg.Done()
					key := TenantKey(t)
					m := Acquire(key)
					defer Release(m)
					if err := p.Extract(d.Logger(), asyncCtx, xmlOnly, imagesOnly); err != nil {
```

with the no-mutex form (we'll rewrite this whole handler in Task 19; this is a transitional change so the file compiles after deleting `mutex.go`):

```go
				wg.Add(1)
				go func() {
					defer wg.Done()
					if err := p.Extract(d.Logger(), asyncCtx, xmlOnly, imagesOnly); err != nil {
```

Also remove the now-unused `TenantKey(t)` line and the `_ = key` if needed. Build to confirm.

- [ ] **Step 14.3: Delete the files**

```bash
git rm services/atlas-wz-extractor/atlas.com/wz-extractor/extraction/mutex.go services/atlas-wz-extractor/atlas.com/wz-extractor/extraction/mutex_test.go
```

- [ ] **Step 14.4: Build the package**

```bash
cd services/atlas-wz-extractor/atlas.com/wz-extractor && go build ./extraction/...
```

Expected: success.

- [ ] **Step 14.5: Commit**

```bash
git add services/atlas-wz-extractor/atlas.com/wz-extractor/extraction/resource.go
git commit -m "refactor(atlas-wz-extractor): remove in-process tenant mutex (replaced by Redis lock)"
```

---

## Task 15: Producer wiring — `kafka/producer/producer.go` (verbatim copy)

**Files:**
- Create: `services/atlas-wz-extractor/atlas.com/wz-extractor/kafka/producer/producer.go`

- [ ] **Step 15.1: Write the file**

```go
package producer

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/sirupsen/logrus"
)

// Provider returns a MessageProducer for a given topic env-var token.
type Provider func(token string) producer.MessageProducer

// ProviderImpl mirrors atlas-data: each call yields a producer with the
// span+tenant header decorators attached.
func ProviderImpl(l logrus.FieldLogger) func(ctx context.Context) func(token string) producer.MessageProducer {
	return func(ctx context.Context) func(token string) producer.MessageProducer {
		sd := producer.SpanHeaderDecorator(ctx)
		td := producer.TenantHeaderDecorator(ctx)
		return func(token string) producer.MessageProducer {
			return producer.Produce(l)(producer.ManagerWriterProvider(l)(token))(sd, td)
		}
	}
}
```

- [ ] **Step 15.2: Build**

```bash
cd services/atlas-wz-extractor/atlas.com/wz-extractor && go build ./kafka/producer/...
```

Expected: success.

- [ ] **Step 15.3: Commit**

```bash
git add services/atlas-wz-extractor/atlas.com/wz-extractor/kafka/producer/producer.go
git commit -m "feat(atlas-wz-extractor): kafka producer provider"
```

---

## Task 16: Consumer config helper — `kafka/consumer/consumer.go`

**Files:**
- Create: `services/atlas-wz-extractor/atlas.com/wz-extractor/kafka/consumer/consumer.go`

- [ ] **Step 16.1: Write the file**

```go
package consumer

import (
	"os"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/sirupsen/logrus"
)

// NewConfig is a curried builder mirroring atlas-data's. It looks up the topic
// name from the env var `token` and produces a consumer.Config.
func NewConfig(l logrus.FieldLogger) func(name string) func(token string) func(groupId string) consumer.Config {
	return func(name string) func(token string) func(groupId string) consumer.Config {
		return func(token string) func(groupId string) consumer.Config {
			t, _ := topic.EnvProvider(l)(token)()
			return func(groupId string) consumer.Config {
				return consumer.NewConfig(LookupBrokers(), name, t, groupId)
			}
		}
	}
}

// LookupBrokers reads the cluster bootstrap servers from BOOTSTRAP_SERVERS.
func LookupBrokers() []string {
	return []string{os.Getenv("BOOTSTRAP_SERVERS")}
}
```

- [ ] **Step 16.2: Build**

```bash
cd services/atlas-wz-extractor/atlas.com/wz-extractor && go build ./kafka/consumer/
```

Expected: success.

- [ ] **Step 16.3: Commit**

```bash
git add services/atlas-wz-extractor/atlas.com/wz-extractor/kafka/consumer/consumer.go
git commit -m "feat(atlas-wz-extractor): kafka consumer config helper"
```

---

## Task 17: Extraction command + provider + consumer

**Files:**
- Create: `services/atlas-wz-extractor/atlas.com/wz-extractor/kafka/consumer/extraction/kafka.go`
- Create: `services/atlas-wz-extractor/atlas.com/wz-extractor/kafka/consumer/extraction/consumer.go`
- Create: `services/atlas-wz-extractor/atlas.com/wz-extractor/kafka/consumer/extraction/consumer_test.go`
- Create: `services/atlas-wz-extractor/atlas.com/wz-extractor/kafka/message/extraction/kafka.go`

- [ ] **Step 17.1: Write `kafka/consumer/extraction/kafka.go`**

```go
package extraction

const (
	EnvCommandTopic            = "COMMAND_TOPIC_WZ_EXTRACTION"
	CommandStartExtractionUnit = "START_EXTRACTION_UNIT"
)

type command[E any] struct {
	Type string `json:"type"`
	Body E      `json:"body"`
}

type startExtractionUnitBody struct {
	JobId      string `json:"jobId"`
	WzFile     string `json:"wzFile"`
	XmlOnly    bool   `json:"xmlOnly"`
	ImagesOnly bool   `json:"imagesOnly"`
}
```

- [ ] **Step 17.2: Write `kafka/message/extraction/kafka.go`** (producer-side env name + value provider helper)

```go
package extraction

import (
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

const (
	EnvCommandTopic            = "COMMAND_TOPIC_WZ_EXTRACTION"
	CommandStartExtractionUnit = "START_EXTRACTION_UNIT"
)

type Command[E any] struct {
	Type string `json:"type"`
	Body E      `json:"body"`
}

type StartExtractionUnitBody struct {
	JobId      string `json:"jobId"`
	WzFile     string `json:"wzFile"`
	XmlOnly    bool   `json:"xmlOnly"`
	ImagesOnly bool   `json:"imagesOnly"`
}

// StartExtractionUnitProvider builds one kafka.Message keyed by jobId so all
// of one job's units land in the same partition (when partition count permits)
// — but partition count >= 16 means cross-job parallelism still works.
func StartExtractionUnitProvider(jobId, wzFile string, xmlOnly, imagesOnly bool) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(djb2(jobId)))
	value := &Command[StartExtractionUnitBody]{
		Type: CommandStartExtractionUnit,
		Body: StartExtractionUnitBody{
			JobId:      jobId,
			WzFile:     wzFile,
			XmlOnly:    xmlOnly,
			ImagesOnly: imagesOnly,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

// djb2 hashes a string into an int suitable for producer.CreateKey, which
// expects an int. The exact hash is unimportant — we just want consistent
// keying so per-job units are not all glued to the same partition.
func djb2(s string) uint32 {
	var h uint32 = 5381
	for i := 0; i < len(s); i++ {
		h = h*33 + uint32(s[i])
	}
	return h
}
```

- [ ] **Step 17.3: Write the failing consumer-handler test**

`services/atlas-wz-extractor/atlas.com/wz-extractor/kafka/consumer/extraction/consumer_test.go`:

```go
package extraction

import (
	"atlas-wz-extractor/extraction/job"
	"atlas-wz-extractor/extraction/lock"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
)

type fakeProcessor struct {
	calls   int
	failOn  string
	failErr error
}

func (f *fakeProcessor) ExtractUnit(l logrus.FieldLogger, ctx context.Context, wzFile string, xmlOnly, imagesOnly bool) error {
	f.calls++
	if wzFile == f.failOn {
		return f.failErr
	}
	return nil
}

func newRedis(t *testing.T) *goredis.Client {
	t.Helper()
	mr := miniredis.RunT(t)
	return goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
}

func TestHandler_HappyPath_FinalizesJob(t *testing.T) {
	ctx := context.Background()
	c := newRedis(t)
	store := job.NewStore(c)
	tl := lock.NewTenantLock(c, time.Minute)

	// seed: one-unit job, lock held by that job
	now := time.Now().UTC().Truncate(time.Second)
	j := job.NewJobBuilder().SetId("J").SetTenantId("T").SetRegion("GMS").
		SetMajorVersion(83).SetMinorVersion(1).
		SetStatus(job.JobRunning).SetUnitsTotal(1).SetCreatedAt(now).SetUpdatedAt(now).Build()
	if err := store.Create(ctx, j, []job.Unit{job.NewUnitBuilder().SetWzFile("Map.wz").SetStatus(job.UnitPending).Build()}, 3600); err != nil {
		t.Fatal(err)
	}
	lockKey := job.LockKey("T", "GMS", 83, 1)
	if _, err := tl.Acquire(ctx, lockKey, "J"); err != nil {
		t.Fatal(err)
	}

	fp := &fakeProcessor{}
	l, _ := test.NewNullLogger()
	h := handleStartExtractionUnit(fp, store, tl)

	h(l, ctx, command[startExtractionUnitBody]{
		Type: CommandStartExtractionUnit,
		Body: startExtractionUnitBody{JobId: "J", WzFile: "Map.wz"},
	})

	gotJob, units, err := store.Get(ctx, "J")
	if err != nil {
		t.Fatal(err)
	}
	if gotJob.Status() != job.JobCompleted {
		t.Fatalf("status: %s", gotJob.Status())
	}
	if len(units) != 1 || units[0].Status() != job.UnitSucceeded {
		t.Fatalf("unit status: %+v", units)
	}
	// lock was released
	heldBy := c.Get(ctx, lockKey).Val()
	if heldBy != "" {
		t.Fatalf("lock should be released; still held by %q", heldBy)
	}
}

func TestHandler_RedeliverySkipsWork(t *testing.T) {
	ctx := context.Background()
	c := newRedis(t)
	store := job.NewStore(c)
	tl := lock.NewTenantLock(c, time.Minute)

	now := time.Now().UTC().Truncate(time.Second)
	j := job.NewJobBuilder().SetId("J2").SetTenantId("T").SetRegion("GMS").
		SetMajorVersion(83).SetMinorVersion(1).
		SetStatus(job.JobRunning).SetUnitsTotal(1).SetCreatedAt(now).SetUpdatedAt(now).Build()
	_ = store.Create(ctx, j, []job.Unit{job.NewUnitBuilder().SetWzFile("Map.wz").SetStatus(job.UnitSucceeded).SetCompletedAt(now).Build()}, 3600)
	_ = store.MarkJobTerminal(ctx, "J2", job.JobCompleted)

	fp := &fakeProcessor{}
	l, _ := test.NewNullLogger()
	h := handleStartExtractionUnit(fp, store, tl)

	h(l, ctx, command[startExtractionUnitBody]{
		Type: CommandStartExtractionUnit,
		Body: startExtractionUnitBody{JobId: "J2", WzFile: "Map.wz"},
	})

	if fp.calls != 0 {
		t.Fatalf("expected ExtractUnit to be skipped on redelivery, got %d calls", fp.calls)
	}
}

func TestHandler_FailedUnit_MarksJobFailed(t *testing.T) {
	ctx := context.Background()
	c := newRedis(t)
	store := job.NewStore(c)
	tl := lock.NewTenantLock(c, time.Minute)

	now := time.Now().UTC().Truncate(time.Second)
	j := job.NewJobBuilder().SetId("J3").SetTenantId("T").SetRegion("GMS").
		SetMajorVersion(83).SetMinorVersion(1).
		SetStatus(job.JobRunning).SetUnitsTotal(1).SetCreatedAt(now).SetUpdatedAt(now).Build()
	_ = store.Create(ctx, j, []job.Unit{job.NewUnitBuilder().SetWzFile("Bad.wz").SetStatus(job.UnitPending).Build()}, 3600)
	lockKey := job.LockKey("T", "GMS", 83, 1)
	_, _ = tl.Acquire(ctx, lockKey, "J3")

	fp := &fakeProcessor{failOn: "Bad.wz", failErr: errors.New("boom")}
	l, _ := test.NewNullLogger()
	h := handleStartExtractionUnit(fp, store, tl)

	h(l, ctx, command[startExtractionUnitBody]{
		Type: CommandStartExtractionUnit,
		Body: startExtractionUnitBody{JobId: "J3", WzFile: "Bad.wz"},
	})

	gotJob, _, _ := store.Get(ctx, "J3")
	if gotJob.Status() != job.JobFailed {
		t.Fatalf("expected JobFailed, got %s", gotJob.Status())
	}
}
```

- [ ] **Step 17.4: Write `kafka/consumer/extraction/consumer.go`**

```go
package extraction

import (
	"atlas-wz-extractor/extraction/job"
	"atlas-wz-extractor/extraction/lock"
	consumer2 "atlas-wz-extractor/kafka/consumer"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

// processor is the slim subset of extraction.Processor that this consumer
// needs. Defined here so the test can inject a fake without pulling
// extraction.Processor.
type processor interface {
	ExtractUnit(l logrus.FieldLogger, ctx context.Context, wzFile string, xmlOnly, imagesOnly bool) error
}

func InitConsumers(l logrus.FieldLogger) func(rf func(consumer.Config, ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(consumer.Config, ...model.Decorator[consumer.Config])) func(string) {
		return func(consumerGroupId string) {
			rf(
				consumer2.NewConfig(l)("wz_extraction_command")(EnvCommandTopic)(consumerGroupId),
				consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser),
				consumer.SetStartOffset(kafka.LastOffset),
			)
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(p processor, store job.Store, tl *lock.TenantLock) func(rf func(string, handler.Handler) (string, error)) error {
	return func(p processor, store job.Store, tl *lock.TenantLock) func(rf func(string, handler.Handler) (string, error)) error {
		return func(rf func(string, handler.Handler) (string, error)) error {
			t, _ := topic.EnvProvider(l)(EnvCommandTopic)()
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStartExtractionUnit(p, store, tl)))); err != nil {
				return err
			}
			return nil
		}
	}
}

func handleStartExtractionUnit(p processor, store job.Store, tl *lock.TenantLock) message.Handler[command[startExtractionUnitBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c command[startExtractionUnitBody]) {
		if c.Type != CommandStartExtractionUnit {
			return
		}
		ll := l.WithFields(logrus.Fields{"jobId": c.Body.JobId, "wzFile": c.Body.WzFile})

		claimed, err := store.MarkUnitRunning(ctx, c.Body.JobId, c.Body.WzFile)
		if err != nil {
			ll.WithError(err).Error("MarkUnitRunning failed; will retry via Kafka redelivery")
			return
		}
		if !claimed {
			ll.Info("unit already terminal; skipping (redelivery)")
			return
		}

		runErr := p.ExtractUnit(ll, ctx, c.Body.WzFile, c.Body.XmlOnly, c.Body.ImagesOnly)
		terminal := job.UnitSucceeded
		if runErr != nil {
			terminal = job.UnitFailed
		}

		cnt, err := store.FinalizeUnit(ctx, c.Body.JobId, c.Body.WzFile, terminal, runErr)
		if err != nil {
			ll.WithError(err).Error("FinalizeUnit failed; will retry via Kafka redelivery")
			return
		}

		if !cnt.AllDone {
			return
		}

		jobTerminal := job.JobCompleted
		switch {
		case cnt.UnitsFailed == cnt.UnitsTotal:
			jobTerminal = job.JobFailed
		case cnt.UnitsFailed > 0:
			jobTerminal = job.JobCompletedWithErrors
		}
		claimedTerminal, err := store.MarkJobTerminal(ctx, c.Body.JobId, jobTerminal)
		if err != nil {
			ll.WithError(err).Error("MarkJobTerminal failed")
			return
		}
		if claimedTerminal {
			if err := tl.Release(ctx, cnt.LockKey, c.Body.JobId); err != nil {
				ll.WithError(err).Warn("Release tenant lock failed")
			}
			ll.WithField("status", jobTerminal).Info("job finalized")
		}
	}
}
```

- [ ] **Step 17.5: Run consumer tests, expect PASS**

```bash
cd services/atlas-wz-extractor/atlas.com/wz-extractor && go test ./kafka/consumer/extraction/ -v
```

Expected: PASS.

- [ ] **Step 17.6: Commit**

```bash
git add services/atlas-wz-extractor/atlas.com/wz-extractor/kafka/consumer/extraction/ services/atlas-wz-extractor/atlas.com/wz-extractor/kafka/message/extraction/
git commit -m "feat(atlas-wz-extractor): kafka consumer for START_EXTRACTION_UNIT"
```

---

## Task 18: Dispatcher

**Files:**
- Create: `services/atlas-wz-extractor/atlas.com/wz-extractor/extraction/dispatcher.go`
- Create: `services/atlas-wz-extractor/atlas.com/wz-extractor/extraction/dispatcher_test.go`

- [ ] **Step 18.1: Write the failing tests**

```go
package extraction

import (
	"atlas-wz-extractor/extraction/job"
	"atlas-wz-extractor/extraction/lock"
	mext "atlas-wz-extractor/kafka/message/extraction"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	goredis "github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus/hooks/test"
)

func newRedisD(t *testing.T) *goredis.Client {
	t.Helper()
	mr := miniredis.RunT(t)
	return goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
}

type fakeEmitter struct {
	mu       sync.Mutex
	messages []mext.Command[mext.StartExtractionUnitBody]
	failOn   int
}

// Provide adapts to producer.MessageProducer signature: func(token string) func(model.Provider[[]kafka.Message]) error
func (f *fakeEmitter) Emit(token string) func(model.Provider[[]kafka.Message]) error {
	return func(mp model.Provider[[]kafka.Message]) error {
		ms, err := mp()
		if err != nil {
			return err
		}
		f.mu.Lock()
		defer f.mu.Unlock()
		for _, m := range ms {
			var c mext.Command[mext.StartExtractionUnitBody]
			_ = json.Unmarshal(m.Value, &c)
			f.messages = append(f.messages, c)
			if f.failOn > 0 && len(f.messages) >= f.failOn {
				return os.ErrClosed // arbitrary non-nil
			}
		}
		return nil
	}
}

func (f *fakeEmitter) provider() func(string) producer.MessageProducer {
	return func(token string) producer.MessageProducer {
		return f.Emit(token)
	}
}

func setupDispatcher(t *testing.T, fe *fakeEmitter, c *goredis.Client) (*mux.Router, *uuid.UUID, string) {
	t.Helper()
	tenantId := uuid.New()
	inputDir := t.TempDir()
	tenantInput := filepath.Join(inputDir, tenantId.String(), "GMS", "83.1")
	if err := os.MkdirAll(tenantInput, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Map.wz", "Mob.wz"} {
		if err := os.WriteFile(filepath.Join(tenantInput, name), []byte("dummy"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	store := job.NewStore(c)
	tl := lock.NewTenantLock(c, time.Minute)
	dirs := Dirs{InputDir: inputDir, OutputXmlDir: t.TempDir(), OutputImgDir: t.TempDir()}

	router := mux.NewRouter()
	l, _ := test.NewNullLogger()
	initFn := InitResource(NewProcessor(inputDir, dirs.OutputXmlDir, dirs.OutputImgDir), store, tl, fe.provider(), &sync.WaitGroup{}, dirs)
	routeInit := initFn(serverInfo{})
	routeInit(router, l)

	return router, &tenantId, inputDir
}

func TestDispatcher_HappyPath202(t *testing.T) {
	c := newRedisD(t)
	fe := &fakeEmitter{}
	router, tid, _ := setupDispatcher(t, fe, c)

	req := httptest.NewRequest(http.MethodPost, "/wz/extractions", nil)
	req.Header.Set("TENANT_ID", tid.String())
	req.Header.Set("REGION", "GMS")
	req.Header.Set("MAJOR_VERSION", "83")
	req.Header.Set("MINOR_VERSION", "1")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("status: %d body=%s", w.Code, w.Body.String())
	}
	var body struct {
		JobId      string `json:"jobId"`
		UnitsTotal int    `json:"unitsTotal"`
		Status     string `json:"status"`
	}
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.JobId == "" || body.UnitsTotal != 2 || body.Status != "running" {
		t.Fatalf("body: %+v", body)
	}

	fe.mu.Lock()
	defer fe.mu.Unlock()
	if len(fe.messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(fe.messages))
	}
}

func TestDispatcher_EmptyInput400(t *testing.T) {
	c := newRedisD(t)
	fe := &fakeEmitter{}
	tenantId := uuid.New()
	inputDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(inputDir, tenantId.String(), "GMS", "83.1"), 0o755); err != nil {
		t.Fatal(err)
	}
	store := job.NewStore(c)
	tl := lock.NewTenantLock(c, time.Minute)

	router := mux.NewRouter()
	l, _ := test.NewNullLogger()
	initFn := InitResource(NewProcessor(inputDir, t.TempDir(), t.TempDir()), store, tl, fe.provider(), &sync.WaitGroup{}, Dirs{InputDir: inputDir, OutputXmlDir: t.TempDir(), OutputImgDir: t.TempDir()})
	initFn(serverInfo{})(router, l)

	req := httptest.NewRequest(http.MethodPost, "/wz/extractions", nil)
	req.Header.Set("TENANT_ID", tenantId.String())
	req.Header.Set("REGION", "GMS")
	req.Header.Set("MAJOR_VERSION", "83")
	req.Header.Set("MINOR_VERSION", "1")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status: %d body=%s", w.Code, w.Body.String())
	}
}

func TestDispatcher_LockConflict409(t *testing.T) {
	c := newRedisD(t)
	fe := &fakeEmitter{}
	router, tid, _ := setupDispatcher(t, fe, c)

	req := func() *http.Request {
		r := httptest.NewRequest(http.MethodPost, "/wz/extractions", nil)
		r.Header.Set("TENANT_ID", tid.String())
		r.Header.Set("REGION", "GMS")
		r.Header.Set("MAJOR_VERSION", "83")
		r.Header.Set("MINOR_VERSION", "1")
		return r
	}

	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req())
	if w1.Code != http.StatusAccepted {
		t.Fatalf("first call: %d body=%s", w1.Code, w1.Body.String())
	}

	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req())
	if w2.Code != http.StatusConflict {
		t.Fatalf("second call: %d body=%s", w2.Code, w2.Body.String())
	}
}
```

- [ ] **Step 18.2: Run tests, expect FAIL (`InitResource` signature changed but isn't yet)**

```bash
cd services/atlas-wz-extractor/atlas.com/wz-extractor && go test ./extraction/ -run "TestDispatcher" -v
```

Expected: build error or test fail.

- [ ] **Step 18.3: Write `dispatcher.go`**

```go
package extraction

import (
	"atlas-wz-extractor/extraction/job"
	"atlas-wz-extractor/extraction/lock"
	mext "atlas-wz-extractor/kafka/message/extraction"
	"atlas-wz-extractor/rest"
	"context"
	"encoding/json"
	"net/http"
	"path/filepath"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

type producerProvider func(token string) producer.MessageProducer

const (
	jobTTLSeconds  = 24 * 60 * 60
	lockRefreshDiv = 3 // refresh every TTL/3
)

func handleExtract(p Processor, store job.Store, tl *lock.TenantLock, prod producerProvider, dirs Dirs) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			t := tenant.MustFromContext(d.Context())
			xmlOnly := r.URL.Query().Get("xmlOnly") == "true"
			imagesOnly := r.URL.Query().Get("imagesOnly") == "true"
			ll := d.Logger().WithFields(logrus.Fields{
				"tenantId": t.Id().String(),
				"region":   t.Region(),
				"version":  TenantPath(t),
			})

			tenantInput := filepath.Join(dirs.InputDir, TenantPath(t))
			wzFiles, err := filepath.Glob(filepath.Join(tenantInput, "*.wz"))
			if err != nil {
				ll.WithError(err).Error("glob failed")
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}
			if len(wzFiles) == 0 {
				http.Error(w, "no .wz files staged for tenant; upload via PATCH /api/wz/input first", http.StatusBadRequest)
				return
			}

			jobId := uuid.NewString()
			lockKey := job.LockKey(t.Id().String(), t.Region(), t.MajorVersion(), t.MinorVersion())

			acquired, err := tl.Acquire(d.Context(), lockKey, jobId)
			if err != nil {
				ll.WithError(err).Error("redis lock Acquire failed")
				http.Error(w, "redis unavailable", http.StatusServiceUnavailable)
				return
			}
			if !acquired {
				http.Error(w, "another extraction is already in flight for this tenant", http.StatusConflict)
				return
			}

			// wipeCharacterCache must run once before any unit message is published.
			if !xmlOnly {
				imgOutPath := filepath.Join(dirs.OutputImgDir, TenantPath(t))
				if err := wipeCharacterCache(imgOutPath); err != nil {
					ll.WithError(err).Warn("Unable to wipe character cache.")
				}
			}

			now := time.Now().UTC()
			wzNames := make([]string, 0, len(wzFiles))
			units := make([]job.Unit, 0, len(wzFiles))
			for _, full := range wzFiles {
				name := filepath.Base(full)
				wzNames = append(wzNames, name)
				units = append(units, job.NewUnitBuilder().SetWzFile(name).SetStatus(job.UnitPending).Build())
			}

			j := job.NewJobBuilder().
				SetId(jobId).
				SetTenantId(t.Id().String()).
				SetRegion(t.Region()).
				SetMajorVersion(t.MajorVersion()).SetMinorVersion(t.MinorVersion()).
				SetStatus(job.JobPending).
				SetUnitsTotal(len(wzNames)).
				SetXmlOnly(xmlOnly).SetImagesOnly(imagesOnly).
				SetCreatedAt(now).SetUpdatedAt(now).Build()

			if err := store.Create(d.Context(), j, units, jobTTLSeconds); err != nil {
				ll.WithError(err).Error("Create job failed")
				_ = tl.Release(d.Context(), lockKey, jobId)
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}
			if err := store.MarkJobRunning(d.Context(), jobId); err != nil {
				ll.WithError(err).Error("MarkJobRunning failed")
				_ = tl.Release(d.Context(), lockKey, jobId)
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}

			// Publish one START_EXTRACTION_UNIT per WZ file.
			emit := prod(mext.EnvCommandTopic)
			publishErr := error(nil)
			published := 0
			for _, name := range wzNames {
				prov := mext.StartExtractionUnitProvider(jobId, name, xmlOnly, imagesOnly)
				if err := emit(prov); err != nil {
					publishErr = err
					break
				}
				published++
			}
			if publishErr != nil {
				ll.WithError(publishErr).WithField("publishedSoFar", published).Error("producer error after partial publish")
				_ = store.MarkUnitsSkippedByStatus(d.Context(), jobId, []job.UnitStatus{job.UnitPending})
				_, _ = store.MarkJobTerminal(d.Context(), jobId, job.JobFailed)
				_ = tl.Release(d.Context(), lockKey, jobId)
				http.Error(w, "kafka publish failed", http.StatusInternalServerError)
				return
			}

			// Spawn refresh goroutine; lifetime bounded by lock TTL or job-poll loop.
			startLockRefresh(d.Logger(), store, tl, lockKey, jobId, tl.TTL()/lockRefreshDiv)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusAccepted)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"jobId":      jobId,
				"unitsTotal": len(wzNames),
				"status":     "running",
			})
		}
	}
}

// startLockRefresh runs a goroutine that periodically refreshes the tenant
// lock until the job reaches a terminal status (or the lock disappears).
func startLockRefresh(l logrus.FieldLogger, store job.Store, tl *lock.TenantLock, lockKey, jobId string, period time.Duration) {
	if period <= 0 {
		return
	}
	go func() {
		ctx := context.Background()
		ticker := time.NewTicker(period)
		defer ticker.Stop()
		for {
			<-ticker.C
			j, _, err := store.Get(ctx, jobId)
			if err != nil {
				return
			}
			if j.Status() != job.JobRunning && j.Status() != job.JobPending {
				return
			}
			if err := tl.Refresh(ctx, lockKey, jobId); err != nil {
				l.WithError(err).Warn("tenant-lock refresh failed; will exit refresh loop")
				return
			}
		}
	}()
}

// silence unused-import warnings if the producer package's types are only
// referenced indirectly through the type alias.
var _ producer.MessageProducer
var _ kafka.Message
var _ = model.FixedProvider[[]kafka.Message]
```

(The `var _ = ...` lines exist to keep `goimports` from removing the imports if a maintainer runs it before the producer code is wired through; the dispatcher actually uses `prod(token)(model.FixedProvider(...))` indirectly through `emit(prov)`. Leave the type aliases.)

- [ ] **Step 18.4: Update `Dirs` to include `OutputImgDir`**

In `extraction/resource.go` (top of file), change:

```go
type Dirs struct {
	InputDir     string
	OutputXmlDir string
}
```

to:

```go
type Dirs struct {
	InputDir     string
	OutputXmlDir string
	OutputImgDir string
}
```

(Resource refactor in Task 19 will pass it through; the dispatcher consumes it.)

- [ ] **Step 18.5: Run dispatcher tests, expect PASS — first need Task 19 to update InitResource**

We can't run dispatcher tests yet because they call the new `InitResource` signature. Move on; tests run together with Task 19.

- [ ] **Step 18.6: Build the package (compile only)**

```bash
cd services/atlas-wz-extractor/atlas.com/wz-extractor && go build ./extraction/...
```

Expect: build error in `resource.go` (it still calls the old `handleExtract` shape). That's fixed in Task 19.

- [ ] **Step 18.7: Stage but do not commit yet — Task 19 ships together**

---

## Task 19: Refactor `resource.go` + JSON:API `wzExtractionJob` handler

**Files:**
- Modify: `services/atlas-wz-extractor/atlas.com/wz-extractor/extraction/resource.go`
- Create: `services/atlas-wz-extractor/atlas.com/wz-extractor/extraction/job_handler.go`
- Create: `services/atlas-wz-extractor/atlas.com/wz-extractor/extraction/job_handler_test.go`
- Modify: `services/atlas-wz-extractor/atlas.com/wz-extractor/extraction/resource_test.go`

- [ ] **Step 19.1: Replace `resource.go`**

```go
package extraction

import (
	"atlas-wz-extractor/extraction/job"
	"atlas-wz-extractor/extraction/lock"
	"atlas-wz-extractor/rest"
	"net/http"
	"sync"

	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
)

type Dirs struct {
	InputDir     string
	OutputXmlDir string
	OutputImgDir string
}

// InitResource wires the WZ extraction REST surface. The wg parameter is kept
// for API compatibility with the previous synchronous-goroutine handler;
// under the Kafka-backed model it is effectively a no-op (the unit work runs
// in consumer pods, not on a goroutine here). Removal is planned as a
// follow-up — see design.md §11.
func InitResource(p Processor, store job.Store, tl *lock.TenantLock, prod producerProvider, wg *sync.WaitGroup, dirs Dirs) func(si jsonapi.ServerInformation) server.RouteInitializer {
	u := &uploadDeps{inputDir: dirs.InputDir}
	s := &statusDeps{inputDir: dirs.InputDir, outputXmlDir: dirs.OutputXmlDir}
	_ = wg
	return func(si jsonapi.ServerInformation) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			register := rest.RegisterHandler(l)(si)

			ext := router.PathPrefix("/wz/extractions").Subrouter()
			ext.HandleFunc("", register("create_extraction", handleExtract(p, store, tl, prod, dirs))).Methods(http.MethodPost)
			ext.HandleFunc("", register("get_extraction_status", s.handleExtractionStatus())).Methods(http.MethodGet)
			ext.HandleFunc("/jobs/{jobId}", register("get_extraction_job", handleJobStatus(store))).Methods(http.MethodGet)

			in := router.PathPrefix("/wz/input").Subrouter()
			in.HandleFunc("", register("upload_wz", u.handleUploadBridge())).Methods(http.MethodPatch)
			in.HandleFunc("", register("get_input_status", s.handleInputStatus())).Methods(http.MethodGet)
		}
	}
}

func (u *uploadDeps) handleUploadBridge() rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return u.handleUpload(d.Logger(), d.Context())
	}
}

func (s *statusDeps) handleInputStatus() rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return s.renderInputStatus(d.Logger(), d.Context())
	}
}

func (s *statusDeps) handleExtractionStatus() rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return s.renderExtractionStatus(d.Logger(), d.Context())
	}
}
```

- [ ] **Step 19.2: Write `job_handler.go`**

```go
package extraction

import (
	"atlas-wz-extractor/extraction/job"
	"atlas-wz-extractor/rest"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

type jobUnitJSON struct {
	WzFile      string  `json:"wzFile"`
	Status      string  `json:"status"`
	StartedAt   *string `json:"startedAt"`
	CompletedAt *string `json:"completedAt"`
	Error       *string `json:"error"`
}

type jobAttributesJSON struct {
	TenantId       string        `json:"tenantId"`
	Region         string        `json:"region"`
	MajorVersion   uint16        `json:"majorVersion"`
	MinorVersion   uint16        `json:"minorVersion"`
	Status         string        `json:"status"`
	XmlOnly        bool          `json:"xmlOnly"`
	ImagesOnly     bool          `json:"imagesOnly"`
	UnitsTotal     int           `json:"unitsTotal"`
	UnitsCompleted int           `json:"unitsCompleted"`
	UnitsFailed    int           `json:"unitsFailed"`
	CreatedAt      string        `json:"createdAt"`
	UpdatedAt      string        `json:"updatedAt"`
	CompletedAt    *string       `json:"completedAt"`
	Units          []jobUnitJSON `json:"units"`
}

type jobResource struct {
	Type       string            `json:"type"`
	Id         string            `json:"id"`
	Attributes jobAttributesJSON `json:"attributes"`
}

type jobEnvelope struct {
	Data jobResource `json:"data"`
}

func handleJobStatus(store job.Store) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			vars := mux.Vars(r)
			jobId := vars["jobId"]
			j, units, err := store.Get(d.Context(), jobId)
			if errors.Is(err, job.ErrNotFound) {
				http.Error(w, "job not found", http.StatusNotFound)
				return
			}
			if err != nil {
				d.Logger().WithError(err).Error("Get job failed")
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}

			fmtTime := func(t time.Time) string {
				if t.IsZero() {
					return ""
				}
				return t.UTC().Format(time.RFC3339)
			}
			optTime := func(t time.Time) *string {
				if t.IsZero() {
					return nil
				}
				s := t.UTC().Format(time.RFC3339)
				return &s
			}

			ujs := make([]jobUnitJSON, 0, len(units))
			for _, u := range units {
				var errPtr *string
				if u.ErrorMessage() != "" {
					e := u.ErrorMessage()
					errPtr = &e
				}
				ujs = append(ujs, jobUnitJSON{
					WzFile:      u.WzFile(),
					Status:      string(u.Status()),
					StartedAt:   optTime(u.StartedAt()),
					CompletedAt: optTime(u.CompletedAt()),
					Error:       errPtr,
				})
			}

			env := jobEnvelope{
				Data: jobResource{
					Type: "wzExtractionJob",
					Id:   j.Id(),
					Attributes: jobAttributesJSON{
						TenantId:       j.TenantId(),
						Region:         j.Region(),
						MajorVersion:   j.MajorVersion(),
						MinorVersion:   j.MinorVersion(),
						Status:         string(j.Status()),
						XmlOnly:        j.XmlOnly(),
						ImagesOnly:     j.ImagesOnly(),
						UnitsTotal:     j.UnitsTotal(),
						UnitsCompleted: j.UnitsCompleted(),
						UnitsFailed:    j.UnitsFailed(),
						CreatedAt:      fmtTime(j.CreatedAt()),
						UpdatedAt:      fmtTime(j.UpdatedAt()),
						CompletedAt:    optTime(j.CompletedAt()),
						Units:          ujs,
					},
				},
			}
			w.Header().Set("Content-Type", "application/vnd.api+json")
			_ = json.NewEncoder(w).Encode(env)
		}
	}
}
```

- [ ] **Step 19.3: Write `job_handler_test.go`**

```go
package extraction

import (
	"atlas-wz-extractor/extraction/job"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gorilla/mux"
	goredis "github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus/hooks/test"
)

func newRedisJ(t *testing.T) *goredis.Client {
	t.Helper()
	mr := miniredis.RunT(t)
	return goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
}

func TestJobHandler_404Unknown(t *testing.T) {
	c := newRedisJ(t)
	store := job.NewStore(c)
	router := mux.NewRouter()
	l, _ := test.NewNullLogger()
	dirs := Dirs{InputDir: t.TempDir(), OutputXmlDir: t.TempDir(), OutputImgDir: t.TempDir()}
	initFn := InitResource(NewProcessor(dirs.InputDir, dirs.OutputXmlDir, dirs.OutputImgDir), store, nil, nil, &sync.WaitGroup{}, dirs)
	initFn(serverInfo{})(router, l)

	req := httptest.NewRequest(http.MethodGet, "/wz/extractions/jobs/does-not-exist", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status: %d", w.Code)
	}
	_ = miniredis.RunT // ensure miniredis import used in race-mode
}

func TestJobHandler_200Returns_wzExtractionJob(t *testing.T) {
	c := newRedisJ(t)
	store := job.NewStore(c)
	now := time.Now().UTC().Truncate(time.Second)
	j := job.NewJobBuilder().SetId("J").SetTenantId("T").SetRegion("GMS").
		SetMajorVersion(83).SetMinorVersion(1).
		SetStatus(job.JobRunning).SetUnitsTotal(2).SetUnitsCompleted(1).
		SetCreatedAt(now).SetUpdatedAt(now).Build()
	if err := store.Create(context.Background(), j, []job.Unit{
		job.NewUnitBuilder().SetWzFile("Map.wz").SetStatus(job.UnitSucceeded).Build(),
		job.NewUnitBuilder().SetWzFile("Mob.wz").SetStatus(job.UnitRunning).Build(),
	}, 3600); err != nil {
		t.Fatal(err)
	}

	router := mux.NewRouter()
	l, _ := test.NewNullLogger()
	dirs := Dirs{InputDir: t.TempDir(), OutputXmlDir: t.TempDir(), OutputImgDir: t.TempDir()}
	initFn := InitResource(NewProcessor(dirs.InputDir, dirs.OutputXmlDir, dirs.OutputImgDir), store, nil, nil, &sync.WaitGroup{}, dirs)
	initFn(serverInfo{})(router, l)

	req := httptest.NewRequest(http.MethodGet, "/wz/extractions/jobs/J", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: %d body=%s", w.Code, w.Body.String())
	}
	var env jobEnvelope
	if err := json.NewDecoder(w.Body).Decode(&env); err != nil {
		t.Fatal(err)
	}
	if env.Data.Type != "wzExtractionJob" || env.Data.Id != "J" {
		t.Fatalf("envelope: %+v", env)
	}
	if env.Data.Attributes.UnitsTotal != 2 || len(env.Data.Attributes.Units) != 2 {
		t.Fatalf("attrs: %+v", env.Data.Attributes)
	}
}
```

- [ ] **Step 19.4: Update `resource_test.go`** to compile against the new `InitResource` signature

The existing tests rely on `InitResource(p, wg, dirs)`. The new signature is `InitResource(p, store, tl, prod, wg, dirs)`. Replace `setupRouter` and `setupRouterWithDirs`:

```go
func setupRouter(p Processor, wg *sync.WaitGroup) *mux.Router {
	return setupRouterWithDirs(p, wg, Dirs{})
}

func setupRouterWithDirs(p Processor, wg *sync.WaitGroup, dirs Dirs) *mux.Router {
	router := mux.NewRouter()
	l, _ := test.NewNullLogger()

	// Resource tests today only exercise the existing endpoints; new
	// endpoints (POST → dispatcher, GET /jobs) are tested in dispatcher_test.go
	// and job_handler_test.go. Pass nils for redis-backed deps to keep these
	// tests focused on routing/upload/status.
	initFn := InitResource(p, nil, nil, nil, wg, dirs)
	routeInit := initFn(serverInfo{})
	routeInit(router, l)
	return router
}
```

The pre-existing `TestHandleExtract_*` tests assert today's `{"status": "started"}` 202. Since the contract changed to `{"jobId":"...","unitsTotal":N,"status":"running"}` AND the dispatcher now writes Redis state, those tests are superseded by `dispatcher_test.go`. **Delete** the following tests from `resource_test.go` because they're obsolete:

- `TestHandleExtract_Returns202`
- `TestHandleExtract_CallsProcessor`
- `TestHandleExtract_XmlOnly`
- `TestHandleExtract_ImagesOnly`
- `TestHandleExtract_DefaultBothModes`
- `TestHandleExtract_TracksGoroutineInWaitGroup`
- `TestHandleExtract_PropagatesTenantToProcessor`
- The `mockProcessor`, `newMockProcessor`, `mockProcessor.waitForExtract`, and `contextCapturingProcessor` helpers — they're no longer used.

Keep `TestHandleExtract_MissingTenantHeader_Returns400` only if the tenant middleware still rejects (look at `setupRouter` — the tenant header parsing happens in middleware before our handler runs, so it still applies). Update the assertion logic if needed; if the mock processor is referenced, replace with a real `NewProcessor(...)` returning Processor; this test won't reach the processor because the tenant header is missing.

Replace the tenant-missing test with:

```go
func TestHandleExtract_MissingTenantHeader_Returns400(t *testing.T) {
	wg := &sync.WaitGroup{}
	router := setupRouter(NewProcessor(t.TempDir(), t.TempDir(), t.TempDir()), wg)

	req := httptest.NewRequest(http.MethodPost, "/wz/extractions", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}
```

Drop unused imports (`context`, `time`, etc.) once the deletions are made; `goimports` will handle this.

- [ ] **Step 19.5: Run all extraction tests, expect PASS**

```bash
cd services/atlas-wz-extractor/atlas.com/wz-extractor && go test ./extraction/... -v
```

Expected: PASS (dispatcher tests, job-handler tests, processor tests, resource tests, pool tests, lock tests, store tests).

- [ ] **Step 19.6: Run service-wide tests**

```bash
cd services/atlas-wz-extractor/atlas.com/wz-extractor && go test ./...
```

Expected: PASS.

- [ ] **Step 19.7: Commit (Tasks 18 + 19 together — they ship as one compilable unit)**

```bash
git add services/atlas-wz-extractor/atlas.com/wz-extractor/extraction/dispatcher.go \
        services/atlas-wz-extractor/atlas.com/wz-extractor/extraction/dispatcher_test.go \
        services/atlas-wz-extractor/atlas.com/wz-extractor/extraction/resource.go \
        services/atlas-wz-extractor/atlas.com/wz-extractor/extraction/resource_test.go \
        services/atlas-wz-extractor/atlas.com/wz-extractor/extraction/job_handler.go \
        services/atlas-wz-extractor/atlas.com/wz-extractor/extraction/job_handler_test.go
git commit -m "feat(atlas-wz-extractor): kafka-backed dispatcher + GET /jobs/{jobId}"
```

---

## Task 20: Wire `main.go`

**Files:**
- Modify: `services/atlas-wz-extractor/atlas.com/wz-extractor/main.go`

- [ ] **Step 20.1: Replace `main.go`**

```go
package main

import (
	"atlas-wz-extractor/characterimage"
	"atlas-wz-extractor/characterrender"
	"atlas-wz-extractor/extraction"
	"atlas-wz-extractor/extraction/job"
	"atlas-wz-extractor/extraction/lock"
	extconsumer "atlas-wz-extractor/kafka/consumer/extraction"
	wzproducer "atlas-wz-extractor/kafka/producer"
	"atlas-wz-extractor/logger"
	"context"
	"os"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	atlasredis "github.com/Chronicle20/atlas/libs/atlas-redis"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/Chronicle20/atlas/libs/atlas-service"
	tracing "github.com/Chronicle20/atlas/libs/atlas-tracing"
)

const serviceName = "atlas-wz-extractor"
const consumerGroupId = "wz-extractor-extraction"

type Server struct {
	baseUrl string
	prefix  string
}

func (s Server) GetBaseURL() string { return s.baseUrl }
func (s Server) GetPrefix() string  { return s.prefix }

func GetServer() Server { return Server{baseUrl: "", prefix: "/api/"} }

const lockTTL = 60 * time.Minute

func main() {
	l := logger.CreateLogger(serviceName)
	l.Infoln("Starting main service.")

	inputDir := os.Getenv("INPUT_WZ_DIR")
	outputXmlDir := os.Getenv("OUTPUT_XML_DIR")
	outputImgDir := os.Getenv("OUTPUT_IMG_DIR")
	if inputDir == "" || outputXmlDir == "" || outputImgDir == "" {
		l.Fatal("Required environment variables not set: INPUT_WZ_DIR, OUTPUT_XML_DIR, OUTPUT_IMG_DIR")
	}

	tdm := service.GetTeardownManager()

	tc, err := tracing.InitTracer(serviceName)
	if err != nil {
		l.WithError(err).Fatal("Unable to initialize tracer.")
	}

	rc := atlasredis.Connect(l)
	defer rc.Close()

	store := job.NewStore(rc)
	tl := lock.NewTenantLock(rc, lockTTL)

	p := extraction.NewProcessor(inputDir, outputXmlDir, outputImgDir)
	cren := characterimage.NewCompositor()

	cmf := consumer.GetManager().AddConsumer(l, tdm.Context(), tdm.WaitGroup())
	extconsumer.InitConsumers(l)(cmf)(consumerGroupId)
	if err := extconsumer.InitHandlers(l)(p, store, tl)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}

	tdm.TeardownFunc(func() { _ = producer.GetManager().Close(l) })

	prodProvider := wzproducer.ProviderImpl(l)(context.Background())

	server.New(l).
		WithContext(tdm.Context()).
		WithWaitGroup(tdm.WaitGroup()).
		SetBasePath(GetServer().GetPrefix()).
		SetPort(os.Getenv("REST_PORT")).
		SetReadTimeout(60 * time.Minute).
		SetWriteTimeout(60 * time.Minute).
		AddRouteInitializer(extraction.InitResource(p, store, tl, prodProvider, tdm.WaitGroup(), extraction.Dirs{InputDir: inputDir, OutputXmlDir: outputXmlDir, OutputImgDir: outputImgDir})(GetServer())).
		AddRouteInitializer(characterrender.InitResource(outputImgDir, cren)(GetServer())).
		AddRouteInitializer(server.MountHandler("/debug/consumers", consumer.GetManager().DebugHandler())).
		Run()

	tdm.TeardownFunc(tracing.Teardown(l)(tc))
	tdm.Wait()
	l.Infoln("Service shutdown.")
}
```

- [ ] **Step 20.2: Build the service**

```bash
cd services/atlas-wz-extractor/atlas.com/wz-extractor && go build ./...
```

Expected: success. Resolve any missed imports.

- [ ] **Step 20.3: Run service-wide tests**

```bash
cd services/atlas-wz-extractor/atlas.com/wz-extractor && go test ./...
```

Expected: PASS.

- [ ] **Step 20.4: Commit**

```bash
git add services/atlas-wz-extractor/atlas.com/wz-extractor/main.go
git commit -m "feat(atlas-wz-extractor): wire redis client + kafka consumer/producer in main"
```

---

## Task 21: Topic provisioning doc — `docs/kafka.md`

**Files:**
- Create: `services/atlas-wz-extractor/atlas.com/wz-extractor/docs/kafka.md`

- [ ] **Step 21.1: Write the doc**

```markdown
# atlas-wz-extractor Kafka topology

## Topics

| Direction | Env var | Purpose | Recommended config |
|---|---|---|---|
| Consume | `COMMAND_TOPIC_WZ_EXTRACTION` | One `START_EXTRACTION_UNIT` message per WZ file in a job. | partitions ≥ 16 (must be ≥ `WZ_EXTRACT_PARALLELISM`); replication 3; cleanup `delete`; retention 24h. |

The dispatcher (REST `POST /api/wz/extractions`) is also a producer on this
topic; it does not run any unit synchronously even with `replicas=1`.

## Consumer group

- Group ID: `wz-extractor-extraction`
- Header parsers: `consumer.SpanHeaderParser`, `consumer.TenantHeaderParser`
- Start offset: `kafka.LastOffset`
- Persistent handler config (matches atlas-data)

## Within-pod parallelism

A single pod's parallelism is bounded by the Kafka partitions assigned to it.
With partition count of 16 and `replicas=3`, each pod is assigned ~5
partitions, which means up to 5 units run in parallel per pod. With
`replicas=1`, all 16 partitions are assigned to one pod.

## Message envelope

```
{
  "type": "START_EXTRACTION_UNIT",
  "body": {
    "jobId": "uuid-v4",
    "wzFile": "Map.wz",
    "xmlOnly": false,
    "imagesOnly": false
  }
}
```

Headers: standard tenant + span headers (`SpanHeaderDecorator`,
`TenantHeaderDecorator`). Key: hash of jobId so a job's units have a stable
key, but partition count > 1 ensures cross-job parallelism.

## Idempotency

Units are at-least-once. The consumer guards against redelivery via Redis
WATCH/MULTI/EXEC on the unit's status field — see `docs/storage.md`.
```

- [ ] **Step 21.2: Commit**

```bash
git add services/atlas-wz-extractor/atlas.com/wz-extractor/docs/kafka.md
git commit -m "docs(atlas-wz-extractor): kafka topology"
```

---

## Task 22: Storage doc — `docs/storage.md`

**Files:**
- Create: `services/atlas-wz-extractor/atlas.com/wz-extractor/docs/storage.md`

- [ ] **Step 22.1: Write the doc**

```markdown
# atlas-wz-extractor storage

## Redis schema

All keys live under the namespace `wz-extractor:`.

### `wz-extractor:job:{jobId}` (HASH)

| Field | Type | Notes |
|---|---|---|
| `tenantId` | string (UUIDv4) | informational; jobId itself is the access key |
| `region` | string | e.g. "GMS" |
| `majorVersion` | int (decimal string) | |
| `minorVersion` | int (decimal string) | |
| `status` | enum (`pending`, `running`, `completed`, `completed_with_errors`, `failed`) | |
| `unitsTotal` | int (decimal string) | |
| `unitsCompleted` | int (decimal string) | `HINCRBY` target |
| `unitsFailed` | int (decimal string) | `HINCRBY` target |
| `xmlOnly` | bool (`true` / `false`) | |
| `imagesOnly` | bool | |
| `createdAt` | RFC3339 | |
| `updatedAt` | RFC3339 | |
| `completedAt` | RFC3339 \| "" | empty until terminal |

TTL: 24h, set on Create.

### `wz-extractor:job:{jobId}:units` (HASH)

Field name = WZ file base name (e.g. `Map.wz`). Value = JSON:

```json
{
  "status": "pending|running|succeeded|failed|skipped",
  "startedAt": "RFC3339",
  "completedAt": "RFC3339",
  "error": "optional error string"
}
```

TTL: shares the parent's 24h.

### `wz-extractor:tenant-lock:{tenantId}:{region}:{maj}.{min}` (STRING)

Value: `jobId` of the holder (so debugging tools can identify ownership).
TTL: 60 minutes; auto-refreshed every 20 minutes by the dispatcher's
goroutine. Released by the "last one home" consumer with a Lua
compare-and-delete (only if the value still matches the holder's jobId).

## Idempotency invariants

1. `MarkUnitRunning` is gated by WATCH on the unit's hash field. If the unit
   is already in a terminal state (succeeded / failed / skipped), the call
   returns `claimed=false` and the consumer skips the work. This is the
   redelivery guard.
2. `FinalizeUnit` is gated the same way. A redelivered finalize over an
   already-terminal unit does not increment counters.
3. `MarkJobTerminal` uses WATCH on the job's `status` field. Only a transition
   from `running` to a terminal status succeeds. The "last one home" race is
   resolved by exactly one CAS winner.

## Multi-tenancy

The job hash is keyed by jobId (UUIDv4) and stores `tenantId` as a field. The
tenant lock key is tenant-scoped. The status endpoint receives only the
jobId; UUIDv4 unguessability is the access control, consistent with the rest
of the service today.

## Connecting

`atlas-wz-extractor` uses the shared `libs/atlas-redis` connection. Required
env: `REDIS_URL`. Optional: `REDIS_PASSWORD`. Default `REDIS_URL` is
`localhost:6379` per the lib.
```

- [ ] **Step 22.2: Commit**

```bash
git add services/atlas-wz-extractor/atlas.com/wz-extractor/docs/storage.md
git commit -m "docs(atlas-wz-extractor): redis schema for jobs/units/lock"
```

---

## Task 23: Update deployment manifest

**Files:**
- Modify: `deploy/k8s/atlas-wz-extractor.yaml`

- [ ] **Step 23.1: Verify the manifest path**

```bash
ls deploy/k8s/atlas-wz-extractor.yaml 2>/dev/null || find deploy -name "*wz-extractor*"
```

If the file does not exist, locate the equivalent (it may live under `deploy/`, `helm/`, or `k8s/`). Note the actual path before editing.

- [ ] **Step 23.2: Add new env vars + `resources` + topic provisioning hint**

The exact YAML diff depends on the current file. The required changes:

1. Add to the container `env:` list:
   - `REDIS_URL` (from existing project ConfigMap or Secret).
   - Optional: `REDIS_PASSWORD` (from Secret).
   - `BOOTSTRAP_SERVERS` (already present in other services — copy that pattern).
   - `COMMAND_TOPIC_WZ_EXTRACTION`: literal string, e.g. `"command.wz.extraction"`.
   - `WZ_EXTRACT_PARALLELISM`: literal `"16"`.

2. Add to the container spec:

```yaml
resources:
  requests:
    cpu: "1"
    memory: "2Gi"
  limits:
    cpu: "4"
    memory: "8Gi"
```

3. Keep `replicas: 1` for the initial rollout.

4. If the project provisions Kafka topics via a Topic CR (look for `KafkaTopic` resources alongside other services' manifests), add a CR for `COMMAND_TOPIC_WZ_EXTRACTION` with `partitions: 16` and `replicas: 3`. If topics are infra-provisioned out-of-band, add a comment in the manifest pointing to where to add the topic.

- [ ] **Step 23.3: Commit**

```bash
git add deploy/k8s/atlas-wz-extractor.yaml
git commit -m "chore(deploy): atlas-wz-extractor env vars + resources + topic"
```

---

## Task 24: Repo grep for callers of `POST /api/wz/extractions`

**Files:** None pre-determined. We modify any caller that parses the old `{"status":"started"}` response shape.

- [ ] **Step 24.1: Grep**

```bash
grep -rn "wz/extractions" --include="*.go" --include="*.ts" --include="*.tsx" --include="*.sh" --include="*.py" --include="*.yaml" --include="*.yml"
```

- [ ] **Step 24.2: For each match, decide**

- If it only sends the POST (does not parse the body) → no change.
- If it parses `{"status":"started"}` → update to consume `{"jobId":"...","unitsTotal":N,"status":"running"}` and (if useful) record the jobId for downstream polling.
- If it parses 202 status code only → no change.

Document each match in the commit message. If there are zero matches outside the service itself, the commit message says so.

- [ ] **Step 24.3: Build any modified caller services + run their tests**

For each modified service:

```bash
cd services/<service>/atlas.com/<svc> && go build ./... && go test ./...
```

- [ ] **Step 24.4: Commit (only if changes were made)**

```bash
git add <files>
git commit -m "refactor: update wz-extractions POST callers for new response shape"
```

If no changes are required, skip the commit and note it in the PR description.

---

## Task 25: Final verification

- [ ] **Step 25.1: Build the whole service**

```bash
cd services/atlas-wz-extractor/atlas.com/wz-extractor && go build ./...
```

Expected: success.

- [ ] **Step 25.2: Run all tests**

```bash
cd services/atlas-wz-extractor/atlas.com/wz-extractor && go test ./...
```

Expected: PASS.

- [ ] **Step 25.3: `go vet`**

```bash
cd services/atlas-wz-extractor/atlas.com/wz-extractor && go vet ./...
```

Expected: no output.

- [ ] **Step 25.4: Verify no stale references**

```bash
cd services/atlas-wz-extractor/atlas.com/wz-extractor && grep -rn "extraction\.Acquire\|extraction\.TryAcquire\|extraction\.Release\|tenantMutexRegistry" --include="*.go"
```

Expected: no matches.

- [ ] **Step 25.5: Verify `runExtraction` is gone (it was the old serial loop method)**

```bash
cd services/atlas-wz-extractor/atlas.com/wz-extractor && grep -rn "runExtraction" --include="*.go"
```

Expected: no matches.

- [ ] **Step 25.6: Confirm git state**

```bash
git rev-parse --show-toplevel  # must end with /.worktrees/task-062-wz-extractor-parallelism
git branch --show-current      # must be task-062-wz-extractor-parallelism
git status --short             # empty
git log --oneline -20
```

- [ ] **Step 25.7: Code review**

Run `superpowers:requesting-code-review`. The dispatcher (Go-only changes) means `plan-adherence-reviewer` + `backend-guidelines-reviewer` are invoked. No frontend changes, so `frontend-guidelines-reviewer` is skipped. Review output lands in `docs/tasks/task-062-wz-extractor-parallelism/audit.md`.

Address review feedback before opening the PR.

---

## Acceptance criteria mapping (PRD §10)

| PRD requirement | Plan task |
|---|---|
| `extraction/processor.go` no longer iterates `wzFiles` serially; `ExtractUnit` exists | Task 13 |
| `WZ_EXTRACT_PARALLELISM` env var is honored with NumCPU fallback | Task 13 (`parallelismFromEnv`) |
| `POST /api/wz/extractions` publishes one `START_EXTRACTION_UNIT` per WZ file under a fresh jobId, returns 202 `{jobId, unitsTotal, status}`, runs no unit synchronously | Task 18 |
| Kafka consumer in group `wz-extractor-extraction` handles `START_EXTRACTION_UNIT` via `ExtractUnit` and updates Redis | Task 17 |
| Two concurrent POSTs same tenant → one 202 + one 409 | Task 11 (lock) + Task 18 (dispatcher) — covered by `TestDispatcher_LockConflict409` |
| All units finalize → status ∈ {completed, completed_with_errors, failed} per §4.6 + lock release | Task 17 (`handleStartExtractionUnit`) — covered by `TestHandler_HappyPath_FinalizesJob` |
| `GET /api/wz/extractions/jobs/{jobId}` JSON:API resource correct from any pod | Task 19 (handler reads only Redis) |
| `GET .../jobs/{unknown}` → 404 | Task 19 — covered by `TestJobHandler_404Unknown` |
| Unit-level failure does not abort siblings; `unitsFailed > 0` → `completed_with_errors` | Task 17 (`handleStartExtractionUnit`) — covered by `TestHandler_FailedUnit_MarksJobFailed` |
| Pod-crash mid-unit → redelivery → no double-count, no stuck `running` | Tasks 7-10 (WATCH guards + `MarkJobTerminal` CAS) — covered by `TestStore_FinalizeUnit_RedeliveryNoOp` and `TestStore_MarkJobTerminal_Once` |
| Deployment manifest declares `resources` + supports `replicas > 1` (kept at 1) | Task 23 |
| Topic creation / partition count documented | Task 21 |
| Redis schema documented | Task 22 |
| Per-unit logs include `jobId` and `wzFile` structured fields | Task 17 (`l.WithFields` at handler entry) |
| All affected Go packages build and tests pass | Task 25 |
| Manual smoke (replicas=1 / replicas=3 wall-clock) | Out of plan scope — done in rollout (design §10) |

---

## Execution Handoff

Plan complete and saved to `docs/tasks/task-062-wz-extractor-parallelism/plan.md`. The user's `/execute-task` command will reuse this plan via `superpowers:subagent-driven-development`. No execution kicked off here.
