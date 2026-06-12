# Config-Status Projection Adoption Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the crash-prone one-shot REST config load in `atlas-character-factory` and `atlas-world` with the Kafka-backed `configuration/projection` pattern proven in `atlas-login`, so tenants provisioned after pod start are picked up live and missing tenants return an error instead of `log.Fatalf`-crashing the pod.

**Architecture:** Each service copy-ports the **tenant subset** of login's `configuration/projection` package (envelope/caughtup/state/subscriber), rewrites `configuration/registry.go` to an error-returning, readiness-gated read API, and adds a `configuration/bridge.go` ticker that republishes the projection snapshot into the package-level config vars. Readiness is gated at `GET /readyz` via a one-way catch-up gate and declared as a k8s `readinessProbe`. World additionally re-initializes world rates on tenant apply/change.

**Tech Stack:** Go, `libs/atlas-kafka` (consumer manager, `ReadEndOffsets`, `FirstOffset`), `libs/atlas-rest/server` (`MountReadiness`), `libs/atlas-tenant`, `github.com/segmentio/kafka-go`, `github.com/google/uuid`, `github.com/sirupsen/logrus`, `testify`.

> **`<worktree>`** in this plan means the task worktree root:
> `<repo-root>/.worktrees/task-090-config-projection-adoption`. `cd <worktree>`
> before running any command, and verify the branch is
> `task-090-config-projection-adoption` after each commit. All file paths are
> relative to `<worktree>`.

---

## File Structure

**atlas-character-factory** (`services/atlas-character-factory/atlas.com/character-factory/`):
- `configuration/projection/envelope.go` — NEW: `TenantEnvelope`, `DecodeTenantEnvelope`, `IsTombstone`, schema version.
- `configuration/projection/caughtup.go` — NEW: one-way readiness gate (verbatim from login).
- `configuration/projection/state.go` — NEW: in-memory tenant snapshot; `ApplyTenant` sets `Id`.
- `configuration/projection/subscriber.go` — NEW: single tenant-topic consumer.
- `configuration/projection/projection_test.go` — NEW: decode/state/caughtup unit tests.
- `configuration/registry.go` — REWRITE: error-returning, readiness-gated; remove `Init`/`sync.Once`.
- `configuration/registry_test.go` — NEW: block-until-publish + absent-tenant.
- `configuration/bridge.go` — NEW: `RunBridge` republish ticker (`onChange = nil`).
- `configuration/requests.go` — DELETE (only `requestAllTenants`, now unused).
- `main.go` — REWRITE wiring: projection + catch-up gate + `/readyz` + shutdown not-ready.
- `deploy/k8s/base/atlas-character-factory.yaml` — add `readinessProbe`.

**atlas-world** (`services/atlas-world/atlas.com/world/`):
- `configuration/projection/{envelope,caughtup,state,subscriber}.go` + `projection_test.go` — NEW (same shape as factory, module `atlas-world`).
- `configuration/registry.go` — REWRITE; keep `initializeRatesFromConfig`; `GetTenantConfigs()` returns `(map, error)`; remove `Init`/`sync.Once`.
- `configuration/registry_test.go` — NEW.
- `configuration/bridge.go` — NEW: `RunBridge` + `ReinitChangedRates` + `changedTenants`.
- `configuration/bridge_test.go` — NEW: `changedTenants` diff.
- `configuration/requests.go` — DELETE.
- `main.go` — REWRITE wiring; sequence boot sweep after catch-up; handle `GetTenantConfigs` error.
- `world/processor.go` — VERIFY only (already returns provider error).
- `deploy/k8s/base/atlas-world.yaml` — add `readinessProbe`.

**No change:** `atlas-configurations`, `atlas-login`, `atlas-channel`.

---

## Phase F — atlas-character-factory

### Task F1: Port the projection data layer (envelope + caughtup + state)

**Files:**
- Create: `services/atlas-character-factory/atlas.com/character-factory/configuration/projection/envelope.go`
- Create: `services/atlas-character-factory/atlas.com/character-factory/configuration/projection/caughtup.go`
- Create: `services/atlas-character-factory/atlas.com/character-factory/configuration/projection/state.go`
- Test: `services/atlas-character-factory/atlas.com/character-factory/configuration/projection/projection_test.go`

- [ ] **Step 1: Write the failing test**

Create `configuration/projection/projection_test.go`:

```go
package projection_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"atlas-character-factory/configuration/projection"
	"atlas-character-factory/configuration/tenant"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestDecodeTenantEnvelope_ParsesShape(t *testing.T) {
	id := uuid.New()
	bts, err := json.Marshal(map[string]any{
		"schema_version": 1,
		"id":             id.String(),
		"config":         map[string]any{"region": "GMS"},
		"emitted_at":     "2026-06-12T12:00:00Z",
	})
	require.NoError(t, err)
	env, err := projection.DecodeTenantEnvelope(bts)
	require.NoError(t, err)
	require.Equal(t, 1, env.SchemaVersion)
	require.Equal(t, id.String(), env.Id)
	require.NotNil(t, env.Config)
}

func TestDecodeTenantEnvelope_RejectsFutureSchema(t *testing.T) {
	bts, _ := json.Marshal(map[string]any{
		"schema_version": projection.SupportedSchemaVersion + 1,
		"id":             uuid.New().String(),
		"config":         map[string]any{},
	})
	_, err := projection.DecodeTenantEnvelope(bts)
	require.ErrorIs(t, err, projection.ErrUnsupportedSchema)
}

func TestIsTombstone(t *testing.T) {
	require.True(t, projection.IsTombstone(nil))
	require.False(t, projection.IsTombstone([]byte("{}")))
}

func TestState_ApplyAndSnapshot_SetsId(t *testing.T) {
	s := projection.NewState()

	tid := uuid.New()
	trm := tenant.RestModel{Region: "GMS", MajorVersion: 84, MinorVersion: 1}
	trmBts, _ := json.Marshal(trm)
	require.NoError(t, s.ApplyTenant(projection.TenantEnvelope{
		SchemaVersion: 1, Id: tid.String(), Config: trmBts,
	}))

	tenants := s.Snapshot()
	require.Len(t, tenants, 1)
	require.Equal(t, "GMS", tenants[tid].Region)
	// Id is json:"-" in the payload; ApplyTenant must populate it from env.Id.
	require.Equal(t, tid.String(), tenants[tid].Id)

	// Snapshot returns a copy: mutating it does not affect State.
	delete(tenants, tid)
	require.Len(t, s.Snapshot(), 1)

	s.ApplyTenantTombstone(tid)
	require.Empty(t, s.Snapshot())
}

func TestApplyTenant_RejectsBadId(t *testing.T) {
	s := projection.NewState()
	err := s.ApplyTenant(projection.TenantEnvelope{
		SchemaVersion: 1, Id: "not-a-uuid", Config: json.RawMessage(`{"region":"GMS"}`),
	})
	require.Error(t, err)
}

func TestCaughtUp_TransitionsAndUnblocksWaiters(t *testing.T) {
	c := projection.NewCaughtUp()
	require.False(t, c.CaughtUpNow())

	c.SetEndOffsets("T1", map[int]int64{0: 3})
	require.False(t, c.CaughtUpNow())

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	waitDone := make(chan error, 1)
	go func() { waitDone <- c.WaitCaughtUp(ctx) }()

	c.Observe("T1", 0, 1)
	require.False(t, c.CaughtUpNow())
	c.Observe("T1", 0, 2)
	require.True(t, c.CaughtUpNow())

	require.NoError(t, <-waitDone)

	// One-way: a lower observation does not un-flip the gate.
	c.Observe("T1", 0, 0)
	require.True(t, c.CaughtUpNow())
}

func TestCaughtUp_EmptyTopicTriviallyCaughtUp(t *testing.T) {
	c := projection.NewCaughtUp()
	c.SetEndOffsets("T", map[int]int64{})
	require.True(t, c.CaughtUpNow())
}

func TestCaughtUp_EndOffsetOneRequiresObservation(t *testing.T) {
	c := projection.NewCaughtUp()
	c.SetEndOffsets("T", map[int]int64{0: 1})
	require.False(t, c.CaughtUpNow())
	c.Observe("T", 0, 0)
	require.True(t, c.CaughtUpNow())
}

func TestCaughtUp_EmptyPartitionTriviallyCaughtUp(t *testing.T) {
	c := projection.NewCaughtUp()
	c.SetEndOffsets("T", map[int]int64{0: 0})
	require.True(t, c.CaughtUpNow())
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-character-factory/atlas.com/character-factory && go test ./configuration/projection/...`
Expected: FAIL — `package atlas-character-factory/configuration/projection` does not exist / undefined symbols.

- [ ] **Step 3: Create `configuration/projection/envelope.go`**

```go
// Package projection is the consumer-side mirror of atlas-configurations'
// transactional outbox: it consumes the tenant config-status topic,
// maintains an in-memory snapshot of the desired tenant config, and gates
// readiness on a one-shot end-offset catch-up. Unlike atlas-login's
// projection it tracks tenants only — this service runs no per-tenant
// socket listeners, so the service-config half is intentionally absent.
package projection

import (
	"encoding/json"
	"errors"
)

// TenantEnvelope is the wire shape published by atlas-configurations'
// outbox for a tenant config-status event. Kept in sync via the
// schema_version field.
type TenantEnvelope struct {
	SchemaVersion int             `json:"schema_version"`
	Id            string          `json:"id"`
	Config        json.RawMessage `json:"config"`
	EmittedAt     string          `json:"emitted_at"`
}

// ErrUnsupportedSchema is returned when the envelope's schema_version is
// higher than this projection understands. Subscribers log and skip
// rather than crash — a forward-compatible reader is a feature.
var ErrUnsupportedSchema = errors.New("projection: unsupported envelope schema_version")

// SupportedSchemaVersion is the highest envelope schema this projection
// can decode. Held in lockstep with atlas-login/atlas-channel and
// atlas-configurations/outbox.CurrentSchemaVersion.
const SupportedSchemaVersion = 1

// IsTombstone reports whether the kafka message is a log-compaction
// tombstone (nil value). Tombstones drive tenant removal.
func IsTombstone(value []byte) bool { return value == nil }

// DecodeTenantEnvelope decodes the wire bytes. Callers should check
// IsTombstone before calling. Rejects schema_version > SupportedSchemaVersion.
func DecodeTenantEnvelope(value []byte) (TenantEnvelope, error) {
	var env TenantEnvelope
	if err := json.Unmarshal(value, &env); err != nil {
		return TenantEnvelope{}, err
	}
	if env.SchemaVersion > SupportedSchemaVersion {
		return env, ErrUnsupportedSchema
	}
	return env, nil
}
```

- [ ] **Step 4: Create `configuration/projection/caughtup.go`**

This is the login gate, verbatim except the package doc comment is generalized off "atlas-login".

```go
package projection

import (
	"context"
	"sync"
	"sync/atomic"
)

// CaughtUp gates the service's readiness on having consumed past the
// end-offset snapshot taken at boot for each subscribed topic. The flag
// is one-way: once caught up, it never reverts (even if the consumed
// offsets logically lag behind a later end-offset snapshot).
type CaughtUp struct {
	mu         sync.Mutex
	snapshots  map[string]map[int]int64 // topic → partition → boot end offset
	consumed   map[string]map[int]int64 // topic → partition → highest consumed
	caughtUp   atomic.Bool
	readyChans []chan struct{} // one-shot signalers for WaitCaughtUp
}

// NewCaughtUp constructs a gate. SetEndOffsets must be called at least
// once before the gate can transition.
func NewCaughtUp() *CaughtUp {
	return &CaughtUp{
		snapshots: make(map[string]map[int]int64),
		consumed:  make(map[string]map[int]int64),
	}
}

// SetEndOffsets records the topic's boot end-offset snapshot. An empty
// offsets map (topic has no data yet) counts as trivially caught-up for
// that topic.
func (c *CaughtUp) SetEndOffsets(topic string, offsets map[int]int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if offsets == nil {
		offsets = map[int]int64{}
	}
	c.snapshots[topic] = offsets
	if c.consumed[topic] == nil {
		c.consumed[topic] = make(map[int]int64)
	}
	c.evaluateLocked()
}

// Observe records that the subscriber has consumed up to (and including)
// offset on partition p of topic. Idempotent: lower offsets are ignored.
func (c *CaughtUp) Observe(topic string, partition int, offset int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	cur, ok := c.consumed[topic]
	if !ok {
		cur = make(map[int]int64)
		c.consumed[topic] = cur
	}
	if existing, present := cur[partition]; present && existing >= offset {
		return
	}
	cur[partition] = offset
	c.evaluateLocked()
}

// CaughtUpNow is the cheap check the subscriber loop can call between
// every message.
func (c *CaughtUp) CaughtUpNow() bool { return c.caughtUp.Load() }

// WaitCaughtUp blocks until the gate flips or ctx is canceled.
func (c *CaughtUp) WaitCaughtUp(ctx context.Context) error {
	if c.caughtUp.Load() {
		return nil
	}
	c.mu.Lock()
	if c.caughtUp.Load() {
		c.mu.Unlock()
		return nil
	}
	ch := make(chan struct{})
	c.readyChans = append(c.readyChans, ch)
	c.mu.Unlock()
	select {
	case <-ch:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// ReadyChecker returns a func suitable for a /health/ready endpoint.
func (c *CaughtUp) ReadyChecker() func() bool { return c.CaughtUpNow }

func (c *CaughtUp) evaluateLocked() {
	if len(c.snapshots) == 0 {
		// No topics registered yet — not caught up.
		return
	}
	for topic, ends := range c.snapshots {
		got := c.consumed[topic]
		for p, end := range ends {
			// end == 0 means the partition is empty (Kafka end-offset is
			// the next-to-be-written offset); trivially caught up.
			if end == 0 {
				continue
			}
			// "caught up" means we've consumed up to end-1 (offsets are
			// 0-indexed; end is the high-water mark). Distinguish "never
			// observed" from "observed offset 0".
			observed, present := got[p]
			if !present || observed < end-1 {
				return
			}
		}
	}
	if !c.caughtUp.Load() {
		c.caughtUp.Store(true)
		for _, ch := range c.readyChans {
			close(ch)
		}
		c.readyChans = nil
	}
}
```

- [ ] **Step 5: Create `configuration/projection/state.go`**

```go
package projection

import (
	"encoding/json"
	"sync"

	"atlas-character-factory/configuration/tenant"

	"github.com/google/uuid"
)

// State is the in-memory snapshot of tenant config. Concurrent reads are
// RW-locked; writes are serialized by the subscriber's single goroutine.
type State struct {
	mu      sync.RWMutex
	tenants map[uuid.UUID]tenant.RestModel
}

func NewState() *State {
	return &State{tenants: make(map[uuid.UUID]tenant.RestModel)}
}

// ApplyTenant inserts or replaces the tenant config for env.Id. The
// tenant.RestModel.Id field is json:"-" (absent from the envelope config
// payload), so it is populated explicitly from env.Id to keep the
// snapshot model identical to the previously REST-loaded one.
func (s *State) ApplyTenant(env TenantEnvelope) error {
	var cfg tenant.RestModel
	if err := json.Unmarshal(env.Config, &cfg); err != nil {
		return err
	}
	id, err := uuid.Parse(env.Id)
	if err != nil {
		return err
	}
	cfg.Id = env.Id
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tenants[id] = cfg
	return nil
}

// ApplyTenantTombstone removes the tenant config for id.
func (s *State) ApplyTenantTombstone(id uuid.UUID) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.tenants, id)
}

// Snapshot returns a copy of the tenants map so callers iterate decoupled
// from concurrent writes.
func (s *State) Snapshot() map[uuid.UUID]tenant.RestModel {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[uuid.UUID]tenant.RestModel, len(s.tenants))
	for k, v := range s.tenants {
		out[k] = v
	}
	return out
}
```

- [ ] **Step 6: Run tests to verify they pass**

Run: `cd services/atlas-character-factory/atlas.com/character-factory && go test ./configuration/projection/...`
Expected: PASS (all tests). Note: `subscriber.go` is added in F2; the package compiles without it.

- [ ] **Step 7: Commit**

```bash
cd <worktree>
git add services/atlas-character-factory/atlas.com/character-factory/configuration/projection/
git commit -m "feat(character-factory): port projection data layer (envelope/caughtup/state)"
git branch --show-current   # must print task-090-config-projection-adoption
```

---

### Task F2: Port the projection subscriber (tenant consumer)

**Files:**
- Create: `services/atlas-character-factory/atlas.com/character-factory/configuration/projection/subscriber.go`

There is no standalone unit test — the subscriber is Kafka wiring, verified by compilation and the end-to-end repro. It is exercised indirectly by the F1 tests (same package compiles).

- [ ] **Step 1: Create `configuration/projection/subscriber.go`**

```go
package projection

import (
	"context"
	"errors"
	"sync"

	consumer2 "atlas-character-factory/kafka/consumer"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

// Subscriber consumes the tenant config-status topic, snapshots end
// offsets at start (gating CaughtUp), then applies envelopes to State.
type Subscriber struct {
	State    *State
	CaughtUp *CaughtUp

	// TenantTopic is the env-var-resolved topic name for tenant config
	// events (EVENT_TOPIC_CONFIGURATION_TENANT_STATUS).
	TenantTopic string
}

// Start snapshots end offsets for the tenant topic and registers a single
// FirstOffset consumer that decodes envelopes into State. wg is the
// teardown manager's WaitGroup (the atlas-kafka library calls Add
// unconditionally; must not be nil).
//
// When TenantTopic is empty the projection has nothing to consume; it
// registers an empty end-offset snapshot so CaughtUp flips trivially and
// the service runs degraded (FR-5) rather than wedging not-ready.
func (s *Subscriber) Start(ctx context.Context, l logrus.FieldLogger, wg *sync.WaitGroup, groupId string) error {
	if s.TenantTopic == "" {
		s.CaughtUp.SetEndOffsets("", map[int]int64{})
		return nil
	}

	brokers := consumer2.LookupBrokers()

	offsets, err := offsetsOrEmpty(ctx, brokers, s.TenantTopic, l)
	if err != nil {
		return err
	}
	s.CaughtUp.SetEndOffsets(s.TenantTopic, offsets)

	cmf := consumer.GetManager().AddConsumer(l, ctx, wg)
	cmf(consumer.NewConfig(brokers, "configuration_tenant_status", s.TenantTopic, groupId),
		consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser),
		consumer.SetStartOffset(kafka.FirstOffset))
	if _, err := consumer.GetManager().RegisterHandler(s.TenantTopic, s.handleTenant(l)); err != nil {
		return err
	}
	return nil
}

func (s *Subscriber) handleTenant(l logrus.FieldLogger) handler.Handler {
	return func(_ logrus.FieldLogger, _ context.Context, msg kafka.Message) (bool, error) {
		s.CaughtUp.Observe(msg.Topic, msg.Partition, msg.Offset)
		if IsTombstone(msg.Value) {
			// Tenant tombstone: key is "tenant:<uuid>". Strip the prefix.
			k := string(msg.Key)
			const prefix = "tenant:"
			if len(k) <= len(prefix) || k[:len(prefix)] != prefix {
				return true, nil
			}
			id, err := uuid.Parse(k[len(prefix):])
			if err != nil {
				return true, nil
			}
			s.State.ApplyTenantTombstone(id)
			return true, nil
		}
		env, err := DecodeTenantEnvelope(msg.Value)
		if err != nil {
			if !errors.Is(err, ErrUnsupportedSchema) {
				l.WithError(err).Warn("projection.tenant.decode_failed")
			}
			return true, nil
		}
		if err := s.State.ApplyTenant(env); err != nil {
			l.WithError(err).Warn("projection.tenant.apply_failed")
			return true, nil
		}
		return true, nil
	}
}

func offsetsOrEmpty(ctx context.Context, brokers []string, topic string, l logrus.FieldLogger) (map[int]int64, error) {
	off, err := consumer.ReadEndOffsets(ctx, brokers, topic)
	if err != nil {
		// A missing topic shouldn't kill startup; log at warn level so the
		// operator notices, and return an empty snapshot (trivially caught up).
		l.WithError(err).WithField("topic", topic).Warn("projection.read_end_offsets_failed")
		return map[int]int64{}, nil
	}
	return off, nil
}
```

- [ ] **Step 2: Verify the package builds**

Run: `cd services/atlas-character-factory/atlas.com/character-factory && go build ./configuration/projection/... && go test ./configuration/projection/...`
Expected: build clean, tests PASS.

> If `go build` reports `consumer.SpanHeaderParser` / `consumer.TenantHeaderParser` / `consumer.SetHeaderParsers` / `consumer.SetStartOffset` / `consumer.NewConfig` / `consumer.ReadEndOffsets` undefined, confirm the import path is `github.com/Chronicle20/atlas/libs/atlas-kafka/consumer` (the same package login's subscriber uses). Do not invent new helpers.

- [ ] **Step 3: Commit**

```bash
cd <worktree>
git add services/atlas-character-factory/atlas.com/character-factory/configuration/projection/subscriber.go
git commit -m "feat(character-factory): port projection tenant subscriber"
git branch --show-current
```

---

### Task F3: Rewrite `configuration/registry.go` (error-returning, readiness-gated)

**Files:**
- Modify (full rewrite): `services/atlas-character-factory/atlas.com/character-factory/configuration/registry.go`
- Test: `services/atlas-character-factory/atlas.com/character-factory/configuration/registry_test.go`

- [ ] **Step 1: Write the failing test**

Create `configuration/registry_test.go`. Note: this is a **single** test function so it does not race other tests on the package-level `readyCh`/`readyOnce`.

```go
package configuration_test

import (
	"testing"
	"time"

	"atlas-character-factory/configuration"
	"atlas-character-factory/configuration/tenant"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// Verifies the crash-fix: GetTenantConfig blocks until PublishSnapshot
// runs (rather than log.Fatalf-ing the pod), then resolves present and
// absent tenants without crashing.
func TestRegistry_BlocksThenResolvesAndReportsAbsent(t *testing.T) {
	id := uuid.New()
	type result struct {
		cfg tenant.RestModel
		err error
	}
	done := make(chan result, 1)
	go func() {
		c, err := configuration.GetTenantConfig(id)
		done <- result{c, err}
	}()

	// Before any PublishSnapshot, GetTenantConfig must block.
	select {
	case r := <-done:
		t.Fatalf("GetTenantConfig returned before PublishSnapshot (cfg=%v, err=%v)", r.cfg, r.err)
	case <-time.After(100 * time.Millisecond):
	}

	configuration.PublishSnapshot(map[uuid.UUID]tenant.RestModel{
		id: {Id: id.String(), Region: "GMS", MajorVersion: 84, MinorVersion: 1},
	})

	select {
	case r := <-done:
		require.NoError(t, r.err)
		require.Equal(t, "GMS", r.cfg.Region)
	case <-time.After(time.Second):
		t.Fatal("GetTenantConfig did not return after PublishSnapshot")
	}

	// Absent tenant in a ready snapshot → ErrTenantNotConfigured, no crash.
	_, err := configuration.GetTenantConfig(uuid.New())
	require.ErrorIs(t, err, configuration.ErrTenantNotConfigured)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-character-factory/atlas.com/character-factory && go test ./configuration/ -run TestRegistry_BlocksThenResolvesAndReportsAbsent`
Expected: FAIL — current `GetTenantConfig` calls `log.Fatalf`; no `PublishSnapshot`/`ErrTenantNotConfigured`; build fails on undefined symbols.

- [ ] **Step 3: Rewrite `configuration/registry.go`**

Replace the **entire** file contents with:

```go
package configuration

import (
	"atlas-character-factory/configuration/tenant"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
)

var configMu sync.RWMutex
var tenantConfig map[uuid.UUID]tenant.RestModel

// readyCh is closed once PublishSnapshot has populated tenantConfig for
// the first time. Kafka handlers (the seed saga) may fire before the
// projection catches up; GetTenantConfig blocks on readyCh instead of the
// legacy log.Fatalf path, bounded by readyTimeout.
var readyCh = make(chan struct{})
var readyOnce sync.Once

// readyTimeout caps how long GetTenantConfig waits for the projection's
// first PublishSnapshot. Long enough to outlast the catch-up window in a
// fresh PR env, short enough that a wedged projection surfaces as request
// errors rather than goroutine pileup.
const readyTimeout = 60 * time.Second

// ErrNotReady is returned by GetTenantConfig when the projection has not
// yet published a snapshot within readyTimeout. Transient: callers should
// log at DEBUG and skip; /readyz keeps the pod out of service until
// catch-up completes.
var ErrNotReady = errors.New("configuration: projection snapshot not yet published")

// ErrTenantNotConfigured is returned by GetTenantConfig when the requested
// tenant is absent from a ready snapshot. Persistent (vs ErrNotReady) — a
// tenant that was never in the projection won't appear by waiting.
var ErrTenantNotConfigured = errors.New("configuration: tenant not configured")

func waitReady() error {
	select {
	case <-readyCh:
		return nil
	case <-time.After(readyTimeout):
		return ErrNotReady
	}
}

func GetTenantConfig(tenantId uuid.UUID) (tenant.RestModel, error) {
	if err := waitReady(); err != nil {
		return tenant.RestModel{}, err
	}
	configMu.RLock()
	defer configMu.RUnlock()
	val, ok := tenantConfig[tenantId]
	if !ok {
		return tenant.RestModel{}, ErrTenantNotConfigured
	}
	return val, nil
}

// PublishSnapshot replaces the package-level tenant config with the
// snapshot taken from the kafka-backed projection. Called by the bridge
// (configuration.RunBridge) after CaughtUp fires and on each observed
// change so legacy GetTenantConfig callers see the same data. The map is
// copied by value so the caller's projection State can mutate
// independently after the call. The first call closes readyCh,
// unblocking any GetTenantConfig waiters.
func PublishSnapshot(tenants map[uuid.UUID]tenant.RestModel) {
	configMu.Lock()
	next := make(map[uuid.UUID]tenant.RestModel, len(tenants))
	for k, v := range tenants {
		next[k] = v
	}
	tenantConfig = next
	configMu.Unlock()

	readyOnce.Do(func() { close(readyCh) })
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd services/atlas-character-factory/atlas.com/character-factory && go test ./configuration/ -run TestRegistry_BlocksThenResolvesAndReportsAbsent`
Expected: PASS.

> The full `./...` build will not pass yet because `main.go` still calls the now-deleted `configuration.Init`. The `-run` test above scopes to the `configuration` package, which compiles on its own (`requests.go` is still present and self-contained until F4).

- [ ] **Step 5: Commit**

```bash
cd <worktree>
git add services/atlas-character-factory/atlas.com/character-factory/configuration/registry.go services/atlas-character-factory/atlas.com/character-factory/configuration/registry_test.go
git commit -m "feat(character-factory): error-returning readiness-gated tenant registry"
git branch --show-current
```

---

### Task F4: Add `configuration/bridge.go` and delete `configuration/requests.go`

**Files:**
- Create: `services/atlas-character-factory/atlas.com/character-factory/configuration/bridge.go`
- Delete: `services/atlas-character-factory/atlas.com/character-factory/configuration/requests.go`

- [ ] **Step 1: Create `configuration/bridge.go`**

```go
package configuration

import (
	"context"
	"time"

	"atlas-character-factory/configuration/tenant"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// RunBridge republishes the projection snapshot into the package-level
// configuration vars on a ticker, so GetTenantConfig callers see live
// updates. snap returns a fresh copy of the projection State's tenants
// map (pass projection.State.Snapshot). onChange (may be nil) is invoked
// with (prev, next) before each publish so side effects can diff. The
// first publish happens immediately; subsequent publishes fire every
// interval until ctx is canceled.
func RunBridge(
	ctx context.Context,
	l logrus.FieldLogger,
	snap func() map[uuid.UUID]tenant.RestModel,
	interval time.Duration,
	onChange func(prev, next map[uuid.UUID]tenant.RestModel),
) {
	var prev map[uuid.UUID]tenant.RestModel
	publish := func() {
		next := snap()
		if onChange != nil {
			onChange(prev, next)
		}
		PublishSnapshot(next)
		prev = next
	}
	publish()

	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			publish()
		}
	}
}
```

> `l` is intentionally unused in the factory bridge (no per-change side effects); it is kept in the signature for symmetry with world's bridge and future logging. Unused parameters are valid Go.

- [ ] **Step 2: Delete `configuration/requests.go`**

```bash
cd <worktree>
git rm services/atlas-character-factory/atlas.com/character-factory/configuration/requests.go
```

- [ ] **Step 3: Verify the configuration package builds and tests pass**

Run: `cd services/atlas-character-factory/atlas.com/character-factory && go build ./configuration/... && go test ./configuration/...`
Expected: build clean, tests PASS. (`main.go` still references `configuration.Init` and will fail to build at the service level — fixed in F5. Scope the build to `./configuration/...`.)

- [ ] **Step 4: Commit**

```bash
cd <worktree>
git add services/atlas-character-factory/atlas.com/character-factory/configuration/bridge.go
git commit -m "feat(character-factory): projection->config bridge; drop REST tenant load"
git branch --show-current
```

---

### Task F5: Wire the projection into `main.go`

**Files:**
- Modify (full rewrite): `services/atlas-character-factory/atlas.com/character-factory/main.go`

- [ ] **Step 1: Replace the file**

Replace the **entire** file with (keeps the `Server`/`GetServer` helpers unchanged):

```go
package main

import (
	"atlas-character-factory/configuration"
	"atlas-character-factory/configuration/projection"
	"atlas-character-factory/factory"
	"atlas-character-factory/kafka/consumer/saga"
	"atlas-character-factory/logger"
	"context"
	"fmt"
	"os"
	"strconv"
	"sync/atomic"
	"time"

	tracing "github.com/Chronicle20/atlas/libs/atlas-tracing"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	consumergroup "github.com/Chronicle20/atlas/libs/atlas-kafka/consumergroup"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/Chronicle20/atlas/libs/atlas-service"
	"github.com/google/uuid"
)

const serviceName = "atlas-character-factory"

var consumerGroupId = consumergroup.Resolve("Character Factory Service")

type Server struct {
	baseUrl string
	prefix  string
}

func (s Server) GetBaseURL() string {
	return s.baseUrl
}

func (s Server) GetPrefix() string {
	return s.prefix
}

func GetServer() Server {
	return Server{
		baseUrl: "",
		prefix:  "/api/",
	}
}

func main() {
	l := logger.CreateLogger(serviceName)
	l.Infoln("Starting main service.")

	tdm := service.GetTeardownManager()

	tc, err := tracing.InitTracer(serviceName)
	if err != nil {
		l.WithError(err).Fatal("Unable to initialize tracer.")
	}

	// Configuration projection: consume the tenant config-status topic,
	// gate readiness on catch-up, then republish snapshots into the
	// configuration package vars. Replaces the legacy one-shot REST load
	// that crash-looped the pod when a tenant was provisioned after start.
	state := projection.NewState()
	caughtUp := projection.NewCaughtUp()
	tenantTopic := os.Getenv("EVENT_TOPIC_CONFIGURATION_TENANT_STATUS")
	if tenantTopic == "" {
		l.Warn("projection: EVENT_TOPIC_CONFIGURATION_TENANT_STATUS is not set; tenant config updates will not propagate live")
	}
	sub := &projection.Subscriber{State: state, CaughtUp: caughtUp, TenantTopic: tenantTopic}
	// Per-process group id so each container start replays the full
	// compacted log from FirstOffset (a shared group id would resume from
	// the previous run's committed offset and never refill State).
	projectionGroupId := fmt.Sprintf("%s - projection - %s", consumerGroupId, uuid.New().String())
	if err := sub.Start(tdm.Context(), l, tdm.WaitGroup(), projectionGroupId); err != nil {
		l.WithError(err).Fatal("Unable to start configuration projection subscriber.")
	}

	cmf := consumer.GetManager().AddConsumer(l, tdm.Context(), tdm.WaitGroup())
	saga.InitConsumers(l)(cmf)(consumerGroupId)
	if err := saga.InitHandlers(l)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}

	tdm.TeardownFunc(func() { _ = producer.GetManager().Close(l) })

	// Gate startup on catch-up. A startup catch-up timeout fails loudly
	// (no traffic served yet; k8s restarts) — distinct from the
	// request-time crash this task eliminates.
	ctxCaught, cancelCaught := context.WithTimeout(tdm.Context(), parseProjectionCatchupTimeout())
	if err := caughtUp.WaitCaughtUp(ctxCaught); err != nil {
		cancelCaught()
		l.WithError(err).Fatal("Configuration projection failed to catch up.")
	}
	cancelCaught()
	l.Info("Configuration projection caught up.")

	// Process-level shutting-down flag; flipped on SIGTERM teardown so
	// /readyz reports not-ready before the rest of shutdown.
	var shuttingDown atomic.Bool
	ready := func() bool { return caughtUp.CaughtUpNow() && !shuttingDown.Load() }
	tdm.TeardownFunc(func() {
		shuttingDown.Store(true)
		l.Info("Flipped /readyz to not-ready for graceful shutdown.")
	})

	// Republish projection snapshots into the configuration package vars
	// so GetTenantConfig callers (the seed saga, preset client) see live
	// updates. onChange is nil — the factory has no per-change side effects.
	go configuration.RunBridge(tdm.Context(), l, state.Snapshot, time.Second, nil)

	server.New(l).
		WithContext(tdm.Context()).
		WithWaitGroup(tdm.WaitGroup()).
		SetBasePath(GetServer().GetPrefix()).
		SetPort(os.Getenv("REST_PORT")).
		AddRouteInitializer(factory.InitResource(GetServer())).
		AddRouteInitializer(server.MountHandler("/debug/consumers", consumer.GetManager().DebugHandler())).
		AddRouteInitializer(server.MountReadiness("/readyz", ready)).
		Run()

	tdm.TeardownFunc(tracing.Teardown(l)(tc))

	tdm.Wait()
	l.Infoln("Service shutdown.")
}

// parseProjectionCatchupTimeout reads PROJECTION_CATCHUP_TIMEOUT_S from
// env (positive integer seconds) and returns the catch-up window for the
// configuration projection at startup. Default is 5 minutes, which covers
// the fresh-PR-env case where atlas-pr-bootstrap is still writing the
// initial tenant configs when this pod boots.
func parseProjectionCatchupTimeout() time.Duration {
	const def = 5 * time.Minute
	v := os.Getenv("PROJECTION_CATCHUP_TIMEOUT_S")
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return def
	}
	return time.Duration(n) * time.Second
}
```

- [ ] **Step 2: Verify the whole service builds, vets, and tests pass**

Run:
```bash
cd services/atlas-character-factory/atlas.com/character-factory
go build ./... && go vet ./... && go test -race ./...
```
Expected: all clean/PASS.

- [ ] **Step 3: Confirm the crash path is gone**

Run: `cd services/atlas-character-factory/atlas.com/character-factory && grep -rn 'log.Fatalf("tenant not configured")' . ; echo "exit=$?"`
Expected: no matches (`grep` prints nothing, `exit=1`).

- [ ] **Step 4: Commit**

```bash
cd <worktree>
git add services/atlas-character-factory/atlas.com/character-factory/main.go
git commit -m "feat(character-factory): wire config projection + /readyz in main"
git branch --show-current
```

---

### Task F6: Add `readinessProbe` to the factory Deployment

**Files:**
- Modify: `deploy/k8s/base/atlas-character-factory.yaml`

- [ ] **Step 1: Add the probe to the container spec**

In `deploy/k8s/base/atlas-character-factory.yaml`, the `character-factory` container ends with the `env:` list (`LOG_LEVEL`, `SERVICE_ID`, `SERVICE_TYPE`). Add a `readinessProbe` as a sibling of `env:` (same indentation as `ports:`/`envFrom:`/`env:`, i.e. 8 spaces). The resulting container block must read:

```yaml
      containers:
      - name: character-factory
        image: ghcr.io/chronicle20/atlas-character-factory/atlas-character-factory:latest
        ports:
        - containerPort: 8080
        envFrom:
        - configMapRef:
            name: atlas-env
        env:
        - name: LOG_LEVEL
          value: "debug"
        - name: SERVICE_ID
          value: 00000000-0000-0000-0000-000000000000
        - name: SERVICE_TYPE
          value: character-factory
        readinessProbe:
          httpGet:
            path: /readyz
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 10
```

- [ ] **Step 2: Validate the YAML parses (kustomize build)**

Run: `cd <worktree> && kubectl kustomize deploy/k8s/base >/dev/null && echo OK`
Expected: `OK` (no YAML/kustomize errors).

> If `kubectl` is unavailable, fall back to a YAML lint: `python3 -c "import yaml; list(yaml.safe_load_all(open('deploy/k8s/base/atlas-character-factory.yaml'))); print('OK')"`.

- [ ] **Step 3: Commit**

```bash
cd <worktree>
git add deploy/k8s/base/atlas-character-factory.yaml
git commit -m "chore(k8s): readinessProbe on /readyz for atlas-character-factory"
git branch --show-current
```

---

## Phase W — atlas-world

### Task W1: Port the projection data layer (envelope + caughtup + state)

**Files:**
- Create: `services/atlas-world/atlas.com/world/configuration/projection/envelope.go`
- Create: `services/atlas-world/atlas.com/world/configuration/projection/caughtup.go`
- Create: `services/atlas-world/atlas.com/world/configuration/projection/state.go`
- Test: `services/atlas-world/atlas.com/world/configuration/projection/projection_test.go`

- [ ] **Step 1: Write the failing test**

Create `configuration/projection/projection_test.go`:

```go
package projection_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"atlas-world/configuration/projection"
	"atlas-world/configuration/tenant"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestDecodeTenantEnvelope_ParsesShape(t *testing.T) {
	id := uuid.New()
	bts, err := json.Marshal(map[string]any{
		"schema_version": 1,
		"id":             id.String(),
		"config":         map[string]any{"region": "GMS"},
		"emitted_at":     "2026-06-12T12:00:00Z",
	})
	require.NoError(t, err)
	env, err := projection.DecodeTenantEnvelope(bts)
	require.NoError(t, err)
	require.Equal(t, 1, env.SchemaVersion)
	require.Equal(t, id.String(), env.Id)
	require.NotNil(t, env.Config)
}

func TestDecodeTenantEnvelope_RejectsFutureSchema(t *testing.T) {
	bts, _ := json.Marshal(map[string]any{
		"schema_version": projection.SupportedSchemaVersion + 1,
		"id":             uuid.New().String(),
		"config":         map[string]any{},
	})
	_, err := projection.DecodeTenantEnvelope(bts)
	require.ErrorIs(t, err, projection.ErrUnsupportedSchema)
}

func TestIsTombstone(t *testing.T) {
	require.True(t, projection.IsTombstone(nil))
	require.False(t, projection.IsTombstone([]byte("{}")))
}

func TestState_ApplyAndSnapshot_SetsId(t *testing.T) {
	s := projection.NewState()

	tid := uuid.New()
	trm := tenant.RestModel{Region: "GMS", MajorVersion: 84, MinorVersion: 1}
	trmBts, _ := json.Marshal(trm)
	require.NoError(t, s.ApplyTenant(projection.TenantEnvelope{
		SchemaVersion: 1, Id: tid.String(), Config: trmBts,
	}))

	tenants := s.Snapshot()
	require.Len(t, tenants, 1)
	require.Equal(t, "GMS", tenants[tid].Region)
	require.Equal(t, tid.String(), tenants[tid].Id)

	delete(tenants, tid)
	require.Len(t, s.Snapshot(), 1)

	s.ApplyTenantTombstone(tid)
	require.Empty(t, s.Snapshot())
}

func TestApplyTenant_RejectsBadId(t *testing.T) {
	s := projection.NewState()
	err := s.ApplyTenant(projection.TenantEnvelope{
		SchemaVersion: 1, Id: "not-a-uuid", Config: json.RawMessage(`{"region":"GMS"}`),
	})
	require.Error(t, err)
}

func TestCaughtUp_TransitionsAndUnblocksWaiters(t *testing.T) {
	c := projection.NewCaughtUp()
	require.False(t, c.CaughtUpNow())

	c.SetEndOffsets("T1", map[int]int64{0: 3})
	require.False(t, c.CaughtUpNow())

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	waitDone := make(chan error, 1)
	go func() { waitDone <- c.WaitCaughtUp(ctx) }()

	c.Observe("T1", 0, 1)
	require.False(t, c.CaughtUpNow())
	c.Observe("T1", 0, 2)
	require.True(t, c.CaughtUpNow())

	require.NoError(t, <-waitDone)

	c.Observe("T1", 0, 0)
	require.True(t, c.CaughtUpNow())
}

func TestCaughtUp_EmptyTopicTriviallyCaughtUp(t *testing.T) {
	c := projection.NewCaughtUp()
	c.SetEndOffsets("T", map[int]int64{})
	require.True(t, c.CaughtUpNow())
}

func TestCaughtUp_EndOffsetOneRequiresObservation(t *testing.T) {
	c := projection.NewCaughtUp()
	c.SetEndOffsets("T", map[int]int64{0: 1})
	require.False(t, c.CaughtUpNow())
	c.Observe("T", 0, 0)
	require.True(t, c.CaughtUpNow())
}

func TestCaughtUp_EmptyPartitionTriviallyCaughtUp(t *testing.T) {
	c := projection.NewCaughtUp()
	c.SetEndOffsets("T", map[int]int64{0: 0})
	require.True(t, c.CaughtUpNow())
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-world/atlas.com/world && go test ./configuration/projection/...`
Expected: FAIL — package does not exist.

- [ ] **Step 3: Create `configuration/projection/envelope.go`**

```go
// Package projection is the consumer-side mirror of atlas-configurations'
// transactional outbox: it consumes the tenant config-status topic,
// maintains an in-memory snapshot of the desired tenant config, and gates
// readiness on a one-shot end-offset catch-up. Unlike atlas-login's
// projection it tracks tenants only — this service runs no per-tenant
// socket listeners, so the service-config half is intentionally absent.
package projection

import (
	"encoding/json"
	"errors"
)

// TenantEnvelope is the wire shape published by atlas-configurations'
// outbox for a tenant config-status event. Kept in sync via the
// schema_version field.
type TenantEnvelope struct {
	SchemaVersion int             `json:"schema_version"`
	Id            string          `json:"id"`
	Config        json.RawMessage `json:"config"`
	EmittedAt     string          `json:"emitted_at"`
}

// ErrUnsupportedSchema is returned when the envelope's schema_version is
// higher than this projection understands. Subscribers log and skip
// rather than crash — a forward-compatible reader is a feature.
var ErrUnsupportedSchema = errors.New("projection: unsupported envelope schema_version")

// SupportedSchemaVersion is the highest envelope schema this projection
// can decode. Held in lockstep with atlas-login/atlas-channel and
// atlas-configurations/outbox.CurrentSchemaVersion.
const SupportedSchemaVersion = 1

// IsTombstone reports whether the kafka message is a log-compaction
// tombstone (nil value). Tombstones drive tenant removal.
func IsTombstone(value []byte) bool { return value == nil }

// DecodeTenantEnvelope decodes the wire bytes. Callers should check
// IsTombstone before calling. Rejects schema_version > SupportedSchemaVersion.
func DecodeTenantEnvelope(value []byte) (TenantEnvelope, error) {
	var env TenantEnvelope
	if err := json.Unmarshal(value, &env); err != nil {
		return TenantEnvelope{}, err
	}
	if env.SchemaVersion > SupportedSchemaVersion {
		return env, ErrUnsupportedSchema
	}
	return env, nil
}
```

- [ ] **Step 4: Create `configuration/projection/caughtup.go`**

Identical to the factory gate (package-local, no service imports):

```go
package projection

import (
	"context"
	"sync"
	"sync/atomic"
)

// CaughtUp gates the service's readiness on having consumed past the
// end-offset snapshot taken at boot for each subscribed topic. The flag
// is one-way: once caught up, it never reverts (even if the consumed
// offsets logically lag behind a later end-offset snapshot).
type CaughtUp struct {
	mu         sync.Mutex
	snapshots  map[string]map[int]int64 // topic → partition → boot end offset
	consumed   map[string]map[int]int64 // topic → partition → highest consumed
	caughtUp   atomic.Bool
	readyChans []chan struct{} // one-shot signalers for WaitCaughtUp
}

// NewCaughtUp constructs a gate. SetEndOffsets must be called at least
// once before the gate can transition.
func NewCaughtUp() *CaughtUp {
	return &CaughtUp{
		snapshots: make(map[string]map[int]int64),
		consumed:  make(map[string]map[int]int64),
	}
}

// SetEndOffsets records the topic's boot end-offset snapshot. An empty
// offsets map (topic has no data yet) counts as trivially caught-up for
// that topic.
func (c *CaughtUp) SetEndOffsets(topic string, offsets map[int]int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if offsets == nil {
		offsets = map[int]int64{}
	}
	c.snapshots[topic] = offsets
	if c.consumed[topic] == nil {
		c.consumed[topic] = make(map[int]int64)
	}
	c.evaluateLocked()
}

// Observe records that the subscriber has consumed up to (and including)
// offset on partition p of topic. Idempotent: lower offsets are ignored.
func (c *CaughtUp) Observe(topic string, partition int, offset int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	cur, ok := c.consumed[topic]
	if !ok {
		cur = make(map[int]int64)
		c.consumed[topic] = cur
	}
	if existing, present := cur[partition]; present && existing >= offset {
		return
	}
	cur[partition] = offset
	c.evaluateLocked()
}

// CaughtUpNow is the cheap check the subscriber loop can call between
// every message.
func (c *CaughtUp) CaughtUpNow() bool { return c.caughtUp.Load() }

// WaitCaughtUp blocks until the gate flips or ctx is canceled.
func (c *CaughtUp) WaitCaughtUp(ctx context.Context) error {
	if c.caughtUp.Load() {
		return nil
	}
	c.mu.Lock()
	if c.caughtUp.Load() {
		c.mu.Unlock()
		return nil
	}
	ch := make(chan struct{})
	c.readyChans = append(c.readyChans, ch)
	c.mu.Unlock()
	select {
	case <-ch:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// ReadyChecker returns a func suitable for a /health/ready endpoint.
func (c *CaughtUp) ReadyChecker() func() bool { return c.CaughtUpNow }

func (c *CaughtUp) evaluateLocked() {
	if len(c.snapshots) == 0 {
		return
	}
	for topic, ends := range c.snapshots {
		got := c.consumed[topic]
		for p, end := range ends {
			if end == 0 {
				continue
			}
			observed, present := got[p]
			if !present || observed < end-1 {
				return
			}
		}
	}
	if !c.caughtUp.Load() {
		c.caughtUp.Store(true)
		for _, ch := range c.readyChans {
			close(ch)
		}
		c.readyChans = nil
	}
}
```

- [ ] **Step 5: Create `configuration/projection/state.go`**

```go
package projection

import (
	"encoding/json"
	"sync"

	"atlas-world/configuration/tenant"

	"github.com/google/uuid"
)

// State is the in-memory snapshot of tenant config. Concurrent reads are
// RW-locked; writes are serialized by the subscriber's single goroutine.
type State struct {
	mu      sync.RWMutex
	tenants map[uuid.UUID]tenant.RestModel
}

func NewState() *State {
	return &State{tenants: make(map[uuid.UUID]tenant.RestModel)}
}

// ApplyTenant inserts or replaces the tenant config for env.Id. The
// tenant.RestModel.Id field is json:"-" (absent from the envelope config
// payload), so it is populated explicitly from env.Id to keep the
// snapshot model identical to the previously REST-loaded one.
func (s *State) ApplyTenant(env TenantEnvelope) error {
	var cfg tenant.RestModel
	if err := json.Unmarshal(env.Config, &cfg); err != nil {
		return err
	}
	id, err := uuid.Parse(env.Id)
	if err != nil {
		return err
	}
	cfg.Id = env.Id
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tenants[id] = cfg
	return nil
}

// ApplyTenantTombstone removes the tenant config for id.
func (s *State) ApplyTenantTombstone(id uuid.UUID) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.tenants, id)
}

// Snapshot returns a copy of the tenants map so callers iterate decoupled
// from concurrent writes.
func (s *State) Snapshot() map[uuid.UUID]tenant.RestModel {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[uuid.UUID]tenant.RestModel, len(s.tenants))
	for k, v := range s.tenants {
		out[k] = v
	}
	return out
}
```

- [ ] **Step 6: Run tests to verify they pass**

Run: `cd services/atlas-world/atlas.com/world && go test ./configuration/projection/...`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
cd <worktree>
git add services/atlas-world/atlas.com/world/configuration/projection/
git commit -m "feat(world): port projection data layer (envelope/caughtup/state)"
git branch --show-current
```

---

### Task W2: Port the projection subscriber (tenant consumer)

**Files:**
- Create: `services/atlas-world/atlas.com/world/configuration/projection/subscriber.go`

- [ ] **Step 1: Create `configuration/projection/subscriber.go`**

```go
package projection

import (
	"context"
	"errors"
	"sync"

	consumer2 "atlas-world/kafka/consumer"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

// Subscriber consumes the tenant config-status topic, snapshots end
// offsets at start (gating CaughtUp), then applies envelopes to State.
type Subscriber struct {
	State    *State
	CaughtUp *CaughtUp

	// TenantTopic is the env-var-resolved topic name for tenant config
	// events (EVENT_TOPIC_CONFIGURATION_TENANT_STATUS).
	TenantTopic string
}

// Start snapshots end offsets for the tenant topic and registers a single
// FirstOffset consumer that decodes envelopes into State. wg is the
// teardown manager's WaitGroup (must not be nil).
//
// When TenantTopic is empty the projection has nothing to consume; it
// registers an empty end-offset snapshot so CaughtUp flips trivially and
// the service runs degraded (FR-5) rather than wedging not-ready.
func (s *Subscriber) Start(ctx context.Context, l logrus.FieldLogger, wg *sync.WaitGroup, groupId string) error {
	if s.TenantTopic == "" {
		s.CaughtUp.SetEndOffsets("", map[int]int64{})
		return nil
	}

	brokers := consumer2.LookupBrokers()

	offsets, err := offsetsOrEmpty(ctx, brokers, s.TenantTopic, l)
	if err != nil {
		return err
	}
	s.CaughtUp.SetEndOffsets(s.TenantTopic, offsets)

	cmf := consumer.GetManager().AddConsumer(l, ctx, wg)
	cmf(consumer.NewConfig(brokers, "configuration_tenant_status", s.TenantTopic, groupId),
		consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser),
		consumer.SetStartOffset(kafka.FirstOffset))
	if _, err := consumer.GetManager().RegisterHandler(s.TenantTopic, s.handleTenant(l)); err != nil {
		return err
	}
	return nil
}

func (s *Subscriber) handleTenant(l logrus.FieldLogger) handler.Handler {
	return func(_ logrus.FieldLogger, _ context.Context, msg kafka.Message) (bool, error) {
		s.CaughtUp.Observe(msg.Topic, msg.Partition, msg.Offset)
		if IsTombstone(msg.Value) {
			k := string(msg.Key)
			const prefix = "tenant:"
			if len(k) <= len(prefix) || k[:len(prefix)] != prefix {
				return true, nil
			}
			id, err := uuid.Parse(k[len(prefix):])
			if err != nil {
				return true, nil
			}
			s.State.ApplyTenantTombstone(id)
			return true, nil
		}
		env, err := DecodeTenantEnvelope(msg.Value)
		if err != nil {
			if !errors.Is(err, ErrUnsupportedSchema) {
				l.WithError(err).Warn("projection.tenant.decode_failed")
			}
			return true, nil
		}
		if err := s.State.ApplyTenant(env); err != nil {
			l.WithError(err).Warn("projection.tenant.apply_failed")
			return true, nil
		}
		return true, nil
	}
}

func offsetsOrEmpty(ctx context.Context, brokers []string, topic string, l logrus.FieldLogger) (map[int]int64, error) {
	off, err := consumer.ReadEndOffsets(ctx, brokers, topic)
	if err != nil {
		l.WithError(err).WithField("topic", topic).Warn("projection.read_end_offsets_failed")
		return map[int]int64{}, nil
	}
	return off, nil
}
```

- [ ] **Step 2: Verify the package builds and tests pass**

Run: `cd services/atlas-world/atlas.com/world && go build ./configuration/projection/... && go test ./configuration/projection/...`
Expected: build clean, tests PASS.

- [ ] **Step 3: Commit**

```bash
cd <worktree>
git add services/atlas-world/atlas.com/world/configuration/projection/subscriber.go
git commit -m "feat(world): port projection tenant subscriber"
git branch --show-current
```

---

### Task W3: Rewrite `configuration/registry.go` (error-returning, keep rate init)

**Files:**
- Modify (full rewrite): `services/atlas-world/atlas.com/world/configuration/registry.go`
- Test: `services/atlas-world/atlas.com/world/configuration/registry_test.go`

- [ ] **Step 1: Write the failing test**

Create `configuration/registry_test.go` (single test — does not race the package-level `readyCh`):

```go
package configuration_test

import (
	"testing"
	"time"

	"atlas-world/configuration"
	"atlas-world/configuration/tenant"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// Verifies the crash-fix: GetTenantConfig / GetTenantConfigs block until
// PublishSnapshot rather than log.Fatalf-ing the pod, then resolve
// present/absent tenants without crashing.
func TestRegistry_BlocksThenResolvesAndReportsAbsent(t *testing.T) {
	id := uuid.New()
	type result struct {
		cfg tenant.RestModel
		err error
	}
	done := make(chan result, 1)
	go func() {
		c, err := configuration.GetTenantConfig(id)
		done <- result{c, err}
	}()

	select {
	case r := <-done:
		t.Fatalf("GetTenantConfig returned before PublishSnapshot (cfg=%v, err=%v)", r.cfg, r.err)
	case <-time.After(100 * time.Millisecond):
	}

	configuration.PublishSnapshot(map[uuid.UUID]tenant.RestModel{
		id: {Id: id.String(), Region: "GMS", MajorVersion: 84, MinorVersion: 1},
	})

	select {
	case r := <-done:
		require.NoError(t, r.err)
		require.Equal(t, "GMS", r.cfg.Region)
	case <-time.After(time.Second):
		t.Fatal("GetTenantConfig did not return after PublishSnapshot")
	}

	_, err := configuration.GetTenantConfig(uuid.New())
	require.ErrorIs(t, err, configuration.ErrTenantNotConfigured)

	// GetTenantConfigs returns the populated snapshot, no Fatalf on empty.
	all, err := configuration.GetTenantConfigs()
	require.NoError(t, err)
	require.Contains(t, all, id)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-world/atlas.com/world && go test ./configuration/ -run TestRegistry_BlocksThenResolvesAndReportsAbsent`
Expected: FAIL — undefined `PublishSnapshot`/`ErrTenantNotConfigured`; `GetTenantConfigs` returns one value, not two.

- [ ] **Step 3: Rewrite `configuration/registry.go`**

Replace the **entire** file with (note: `initializeRatesFromConfig` is preserved verbatim from the original, now called only by the bridge):

```go
package configuration

import (
	"atlas-world/configuration/tenant"
	"atlas-world/rate"
	"context"
	"errors"
	"sync"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	tenant2 "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

var configMu sync.RWMutex
var tenantConfig map[uuid.UUID]tenant.RestModel

// readyCh is closed once PublishSnapshot has populated tenantConfig for
// the first time. Kafka handlers (channel status) may fire before the
// projection catches up; Get* blocks on readyCh instead of the legacy
// log.Fatalf path, bounded by readyTimeout.
var readyCh = make(chan struct{})
var readyOnce sync.Once

const readyTimeout = 60 * time.Second

// ErrNotReady is returned by Get* when the projection has not yet
// published a snapshot within readyTimeout. Transient.
var ErrNotReady = errors.New("configuration: projection snapshot not yet published")

// ErrTenantNotConfigured is returned by GetTenantConfig when the requested
// tenant is absent from a ready snapshot. Persistent.
var ErrTenantNotConfigured = errors.New("configuration: tenant not configured")

func waitReady() error {
	select {
	case <-readyCh:
		return nil
	case <-time.After(readyTimeout):
		return ErrNotReady
	}
}

func GetTenantConfig(tenantId uuid.UUID) (tenant.RestModel, error) {
	if err := waitReady(); err != nil {
		return tenant.RestModel{}, err
	}
	configMu.RLock()
	defer configMu.RUnlock()
	val, ok := tenantConfig[tenantId]
	if !ok {
		return tenant.RestModel{}, ErrTenantNotConfigured
	}
	return val, nil
}

// GetTenantConfigs returns a copy of the full tenant snapshot. Returns
// ErrNotReady before the first PublishSnapshot; otherwise the (possibly
// empty) map. Never log.Fatalf — callers (the boot channel-status sweep)
// log and skip on error.
func GetTenantConfigs() (map[uuid.UUID]tenant.RestModel, error) {
	if err := waitReady(); err != nil {
		return nil, err
	}
	configMu.RLock()
	defer configMu.RUnlock()
	out := make(map[uuid.UUID]tenant.RestModel, len(tenantConfig))
	for k, v := range tenantConfig {
		out[k] = v
	}
	return out, nil
}

// PublishSnapshot replaces the package-level tenant config with the
// snapshot taken from the kafka-backed projection. The first call closes
// readyCh, unblocking any Get* waiters.
func PublishSnapshot(tenants map[uuid.UUID]tenant.RestModel) {
	configMu.Lock()
	next := make(map[uuid.UUID]tenant.RestModel, len(tenants))
	for k, v := range tenants {
		next[k] = v
	}
	tenantConfig = next
	configMu.Unlock()

	readyOnce.Do(func() { close(readyCh) })
}

// initializeRatesFromConfig initializes the rate registry with rates from
// configuration. Called by the bridge onChange hook (configuration.
// ReinitChangedRates) on initial apply and on each tenant config change.
func initializeRatesFromConfig(l logrus.FieldLogger, tenantId uuid.UUID, tc tenant.RestModel) {
	t, err := tenant2.Create(tenantId, tc.Region, tc.MajorVersion, tc.MinorVersion)
	if err != nil {
		l.WithError(err).Errorf("Unable to create tenant model for rate initialization.")
		return
	}

	ctx := tenant2.WithContext(context.Background(), t)
	for worldId, wc := range tc.Worlds {
		rates := rate.NewModel()
		rates = rates.WithRate(rate.TypeExp, wc.GetExpRate())
		rates = rates.WithRate(rate.TypeMeso, wc.GetMesoRate())
		rates = rates.WithRate(rate.TypeItemDrop, wc.GetItemDropRate())
		rates = rates.WithRate(rate.TypeQuestExp, wc.GetQuestExpRate())

		rate.GetRegistry().InitWorldRates(ctx, world.Id(worldId), rates)
		l.Infof("Initialized world [%d] rates from config: exp=%.2f, meso=%.2f, drop=%.2f, quest=%.2f",
			worldId, wc.GetExpRate(), wc.GetMesoRate(), wc.GetItemDropRate(), wc.GetQuestExpRate())
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd services/atlas-world/atlas.com/world && go test ./configuration/ -run TestRegistry_BlocksThenResolvesAndReportsAbsent`
Expected: PASS.

> Full `./...` build still fails: `main.go:89` calls `GetTenantConfigs()` expecting one return value, and `requests.go`/`main.go` still reference removed code. Fixed in W4/W5.

- [ ] **Step 5: Commit**

```bash
cd <worktree>
git add services/atlas-world/atlas.com/world/configuration/registry.go services/atlas-world/atlas.com/world/configuration/registry_test.go
git commit -m "feat(world): error-returning readiness-gated tenant registry"
git branch --show-current
```

---

### Task W4: Add `configuration/bridge.go` (with rate-reinit diff) and delete `requests.go`

**Files:**
- Create: `services/atlas-world/atlas.com/world/configuration/bridge.go`
- Test: `services/atlas-world/atlas.com/world/configuration/bridge_test.go`
- Delete: `services/atlas-world/atlas.com/world/configuration/requests.go`

- [ ] **Step 1: Write the failing test for the diff helper**

Create `configuration/bridge_test.go`. `changedTenants` is package-private, so the test is in-package (`package configuration`).

```go
package configuration

import (
	"testing"

	"atlas-world/configuration/tenant"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestChangedTenants(t *testing.T) {
	id1 := uuid.New()
	id2 := uuid.New()

	a := tenant.RestModel{Id: id1.String(), Region: "GMS", MajorVersion: 83, MinorVersion: 1}
	aChanged := a
	aChanged.MajorVersion = 84

	prev := map[uuid.UUID]tenant.RestModel{id1: a}
	next := map[uuid.UUID]tenant.RestModel{
		id1: aChanged,                           // changed
		id2: {Id: id2.String(), Region: "GMS"}, // new
	}

	// changed + new are returned.
	require.ElementsMatch(t, []uuid.UUID{id1, id2}, changedTenants(prev, next))

	// Identical maps → nothing changed.
	require.Empty(t, changedTenants(next, next))

	// Removed tenant (in prev, absent from next) is not returned and does
	// not panic — only id1 (which changed back) is reported.
	require.Equal(t, []uuid.UUID{id1}, changedTenants(next, prev))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-world/atlas.com/world && go test ./configuration/ -run TestChangedTenants`
Expected: FAIL — `changedTenants` undefined.

- [ ] **Step 3: Create `configuration/bridge.go`**

```go
package configuration

import (
	"context"
	"reflect"
	"time"

	"atlas-world/configuration/tenant"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// RunBridge republishes the projection snapshot into the package-level
// configuration vars on a ticker, so GetTenantConfig(s) callers see live
// updates. snap returns a fresh copy of the projection State's tenants
// map (pass projection.State.Snapshot). onChange (may be nil) is invoked
// with (prev, next) before each publish so side effects (rate re-init)
// can diff. The first publish happens immediately; subsequent publishes
// fire every interval until ctx is canceled.
func RunBridge(
	ctx context.Context,
	l logrus.FieldLogger,
	snap func() map[uuid.UUID]tenant.RestModel,
	interval time.Duration,
	onChange func(prev, next map[uuid.UUID]tenant.RestModel),
) {
	var prev map[uuid.UUID]tenant.RestModel
	publish := func() {
		next := snap()
		if onChange != nil {
			onChange(prev, next)
		}
		PublishSnapshot(next)
		prev = next
	}
	publish()

	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			publish()
		}
	}
}

// ReinitChangedRates returns an onChange hook that re-initializes world
// rates for tenants whose config newly appeared or changed since the
// previous snapshot. Unchanged tenants are left untouched so live
// SetWorldRate overrides survive between config changes (design Q1). A
// config change clobbers that tenant's live rates from config.
func ReinitChangedRates(l logrus.FieldLogger) func(prev, next map[uuid.UUID]tenant.RestModel) {
	return func(prev, next map[uuid.UUID]tenant.RestModel) {
		for _, id := range changedTenants(prev, next) {
			initializeRatesFromConfig(l, id, next[id])
		}
	}
}

// changedTenants returns the ids in next that are absent from prev or
// whose config differs by value. Removed tenants (in prev, absent from
// next) are not returned and never cause a panic.
func changedTenants(prev, next map[uuid.UUID]tenant.RestModel) []uuid.UUID {
	var out []uuid.UUID
	for id, nc := range next {
		pc, ok := prev[id]
		if !ok || !reflect.DeepEqual(pc, nc) {
			out = append(out, id)
		}
	}
	return out
}
```

- [ ] **Step 4: Delete `configuration/requests.go`**

```bash
cd <worktree>
git rm services/atlas-world/atlas.com/world/configuration/requests.go
```

- [ ] **Step 5: Verify the configuration package builds and tests pass**

Run: `cd services/atlas-world/atlas.com/world && go build ./configuration/... && go test ./configuration/...`
Expected: build clean, tests PASS. (`main.go` still fails to build at service level until W5 — scope to `./configuration/...`.)

- [ ] **Step 6: Commit**

```bash
cd <worktree>
git add services/atlas-world/atlas.com/world/configuration/bridge.go services/atlas-world/atlas.com/world/configuration/bridge_test.go
git commit -m "feat(world): projection->config bridge with rate-reinit diff; drop REST tenant load"
git branch --show-current
```

---

### Task W5: Wire the projection into `main.go` and sequence the boot sweep

**Files:**
- Modify (full rewrite of `main()`): `services/atlas-world/atlas.com/world/main.go`
- Verify only (no edit expected): `services/atlas-world/atlas.com/world/world/processor.go:78`

- [ ] **Step 1: Confirm `world/processor.go` already tolerates the new errors**

Run: `cd services/atlas-world/atlas.com/world && sed -n '76,84p' world/processor.go`
Expected: the call site is `c, err := configuration.GetTenantConfig(p.t.Id())` followed by `if err != nil { return model.ErrorProvider[Model](err) }`. This already surfaces `ErrTenantNotConfigured`/`ErrNotReady` as a provider error — **no change needed**. If it differs, stop and reconcile before continuing.

- [ ] **Step 2: Replace the file**

Replace the **entire** file with (keeps `Server`/`GetServer` unchanged):

```go
package main

import (
	"atlas-world/channel"
	"atlas-world/configuration"
	"atlas-world/configuration/projection"
	channel2 "atlas-world/kafka/consumer/channel"
	"atlas-world/logger"
	"atlas-world/rate"
	"atlas-world/tasks"
	"atlas-world/world"
	"context"
	"fmt"
	"os"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-service"
	tracing "github.com/Chronicle20/atlas/libs/atlas-tracing"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	consumergroup "github.com/Chronicle20/atlas/libs/atlas-kafka/consumergroup"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	atlas "github.com/Chronicle20/atlas/libs/atlas-redis"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
)

const serviceName = "atlas-world"

var consumerGroupId = consumergroup.Resolve("World Orchestrator")

type Server struct {
	baseUrl string
	prefix  string
}

func (s Server) GetBaseURL() string {
	return s.baseUrl
}

func (s Server) GetPrefix() string {
	return s.prefix
}

func GetServer() Server {
	return Server{
		baseUrl: "",
		prefix:  "/api/",
	}
}

func main() {
	l := logger.CreateLogger(serviceName)
	l.Infoln("Starting main service.")

	tdm := service.GetTeardownManager()

	rc := atlas.Connect(l)
	channel.InitRegistry(rc)
	rate.InitRegistry(rc)

	tc, err := tracing.InitTracer(serviceName)
	if err != nil {
		l.WithError(err).Fatal("Unable to initialize tracer.")
	}

	// Configuration projection: consume the tenant config-status topic and
	// gate readiness on catch-up. Created BEFORE the REST server so
	// /readyz can close over caughtUp. Replaces the legacy one-shot REST
	// load that crash-looped the pod when a tenant was provisioned after
	// start.
	state := projection.NewState()
	caughtUp := projection.NewCaughtUp()
	tenantTopic := os.Getenv("EVENT_TOPIC_CONFIGURATION_TENANT_STATUS")
	if tenantTopic == "" {
		l.Warn("projection: EVENT_TOPIC_CONFIGURATION_TENANT_STATUS is not set; tenant config updates will not propagate live")
	}
	sub := &projection.Subscriber{State: state, CaughtUp: caughtUp, TenantTopic: tenantTopic}
	projectionGroupId := fmt.Sprintf("%s - projection - %s", consumerGroupId, uuid.New().String())
	if err := sub.Start(tdm.Context(), l, tdm.WaitGroup(), projectionGroupId); err != nil {
		l.WithError(err).Fatal("Unable to start configuration projection subscriber.")
	}

	cmf := consumer.GetManager().AddConsumer(l, tdm.Context(), tdm.WaitGroup())
	channel2.InitConsumers(l)(cmf)(consumerGroupId)
	if err := channel2.InitHandlers(l)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}

	tdm.TeardownFunc(func() { _ = producer.GetManager().Close(l) })

	// Process-level shutting-down flag; flipped on SIGTERM teardown so
	// /readyz reports not-ready before the rest of shutdown.
	var shuttingDown atomic.Bool
	ready := func() bool { return caughtUp.CaughtUpNow() && !shuttingDown.Load() }
	tdm.TeardownFunc(func() {
		shuttingDown.Store(true)
		l.Info("Flipped /readyz to not-ready for graceful shutdown.")
	})

	server.New(l).
		WithContext(tdm.Context()).
		WithWaitGroup(tdm.WaitGroup()).
		SetBasePath(GetServer().GetPrefix()).
		SetPort(os.Getenv("REST_PORT")).
		AddRouteInitializer(channel.InitResource(GetServer())).
		AddRouteInitializer(world.InitResource(GetServer())).
		AddRouteInitializer(rate.InitResource(GetServer())).
		AddRouteInitializer(server.MountHandler("/debug/consumers", consumer.GetManager().DebugHandler())).
		AddRouteInitializer(server.MountReadiness("/readyz", ready)).
		Run()

	l.Infof("Service started.")

	// Gate on catch-up. A startup catch-up timeout fails loudly (k8s
	// restarts) — distinct from the request-time crash this task removes.
	ctxCaught, cancelCaught := context.WithTimeout(tdm.Context(), parseProjectionCatchupTimeout())
	if err := caughtUp.WaitCaughtUp(ctxCaught); err != nil {
		cancelCaught()
		l.WithError(err).Fatal("Configuration projection failed to catch up.")
	}
	cancelCaught()
	l.Info("Configuration projection caught up.")

	// Republish projection snapshots into the configuration package vars
	// and re-init world rates on tenant apply/change. The first publish
	// runs synchronously inside RunBridge before its ticker, so
	// GetTenantConfigs below (which blocks on readyCh) sees a populated
	// snapshot.
	go configuration.RunBridge(tdm.Context(), l, state.Snapshot, time.Second, configuration.ReinitChangedRates(l))

	// Boot channel-status sweep. GetTenantConfigs blocks until the bridge's
	// first publish closes readyCh; on error (not ready) log and skip
	// rather than Fatal.
	ctx, span := otel.GetTracerProvider().Tracer(serviceName).Start(context.Background(), "startup")
	if tcs, err := configuration.GetTenantConfigs(); err != nil {
		l.WithError(err).Warn("Skipping boot channel-status sweep; tenant configs not ready.")
	} else {
		_ = model.ForEachMap(model.FixedProvider(tcs), channel.RequestStatus(l)(ctx))
	}
	span.End()

	go tasks.Register(l, tdm.Context())(channel.NewExpiration(l, tdm.Context(), time.Second*10))

	tdm.TeardownFunc(tracing.Teardown(l)(tc))

	tdm.Wait()
	l.Infoln("Service shutdown.")
}

// parseProjectionCatchupTimeout reads PROJECTION_CATCHUP_TIMEOUT_S from
// env (positive integer seconds) and returns the catch-up window for the
// configuration projection at startup. Default is 5 minutes, covering the
// fresh-PR-env case where atlas-pr-bootstrap is still writing the initial
// tenant configs when this pod boots.
func parseProjectionCatchupTimeout() time.Duration {
	const def = 5 * time.Minute
	v := os.Getenv("PROJECTION_CATCHUP_TIMEOUT_S")
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return def
	}
	return time.Duration(n) * time.Second
}
```

- [ ] **Step 3: Verify the whole service builds, vets, and tests pass**

Run:
```bash
cd services/atlas-world/atlas.com/world
go build ./... && go vet ./... && go test -race ./...
```
Expected: all clean/PASS.

- [ ] **Step 4: Confirm the crash path is gone**

Run: `cd services/atlas-world/atlas.com/world && grep -rn 'log.Fatalf("tenant not configured")' . ; echo "exit=$?"`
Expected: no matches (`exit=1`).

- [ ] **Step 5: Commit**

```bash
cd <worktree>
git add services/atlas-world/atlas.com/world/main.go
git commit -m "feat(world): wire config projection + /readyz; sequence boot sweep after catch-up"
git branch --show-current
```

---

### Task W6: Add `readinessProbe` to the world Deployment

**Files:**
- Modify: `deploy/k8s/base/atlas-world.yaml`

- [ ] **Step 1: Add the probe to the container spec**

In `deploy/k8s/base/atlas-world.yaml`, add a `readinessProbe` as a sibling of `env:` (8-space indent). The container block must read:

```yaml
      containers:
      - name: world
        image: ghcr.io/chronicle20/atlas-world/atlas-world:latest
        ports:
        - containerPort: 8080
        envFrom:
        - configMapRef:
            name: atlas-env
        env:
        - name: LOG_LEVEL
          value: "debug"
        - name: SERVICE_ID
          value: 00000000-0000-0000-0000-000000000000
        - name: SERVICE_TYPE
          value: world-service
        readinessProbe:
          httpGet:
            path: /readyz
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 10
```

- [ ] **Step 2: Validate the YAML parses (kustomize build)**

Run: `cd <worktree> && kubectl kustomize deploy/k8s/base >/dev/null && echo OK`
Expected: `OK`.

> If `kubectl` is unavailable: `python3 -c "import yaml; list(yaml.safe_load_all(open('deploy/k8s/base/atlas-world.yaml'))); print('OK')"`.

- [ ] **Step 3: Commit**

```bash
cd <worktree>
git add deploy/k8s/base/atlas-world.yaml
git commit -m "chore(k8s): readinessProbe on /readyz for atlas-world"
git branch --show-current
```

---

## Phase V — Full Verification (CLAUDE.md Build & Verification)

### Task V1: Module verification, Docker bake, redis-key-guard

**Files:** none (verification only).

- [ ] **Step 1: Per-module `go test -race`, `go vet`, `go build`**

Run:
```bash
cd <worktree>

for m in services/atlas-character-factory/atlas.com/character-factory services/atlas-world/atlas.com/world; do
  echo "=== $m ===";
  ( cd "$m" && go build ./... && go vet ./... && go test -race ./... ) || { echo "FAILED: $m"; break; }
done
```
Expected: both modules print no errors and tests PASS.

- [ ] **Step 2: redis-key-guard**

Run: `cd <worktree> && GOWORK=off tools/redis-key-guard.sh`
Expected: clean (no banned raw keyed go-redis calls; this task added none).

- [ ] **Step 3: Docker bake both services (mandatory — both go.mods touched)**

Run from the worktree root:
```bash
cd <worktree>
docker buildx bake atlas-character-factory
docker buildx bake atlas-world
```
Expected: both targets build successfully. (No new `libs/` were added, so the shared `Dockerfile`/`go.work` need no edits — these bakes confirm the `COPY` set already covers the changed services.)

- [ ] **Step 4: Final acceptance grep — no Fatalf crash path anywhere in either service**

Run:
```bash
cd <worktree>
grep -rn 'log.Fatalf("tenant not configured")' \
  services/atlas-character-factory services/atlas-world ; echo "exit=$?"
```
Expected: no matches (`exit=1`) — satisfies PRD Acceptance Criterion 3.

- [ ] **Step 5: Confirm branch and clean tree, then summarize**

Run:
```bash
cd <worktree>
git branch --show-current   # task-090-config-projection-adoption
git status --short          # only intended changes; nothing unexpected
git log --oneline -14
```
Expected: branch correct; all task commits present; no stray modifications.

> **Manual repro (post-merge, documented for the operator — not automatable here):** with an `atlas-character-factory` pod already running, provision a new GMS tenant in `atlas-configurations`; within seconds the factory snapshot includes it and a `/api/characters/seed` for that tenant succeeds (or fails gracefully on validation) with `RESTARTS` unchanged. Tombstoning a tenant removes it live; a subsequent request returns `ErrTenantNotConfigured` (no crash). See PRD §10.

---

## Notes for the Executor

- **Do not modify** `atlas-login`, `atlas-channel`, or `atlas-configurations`. They are reference/producer only.
- **`caughtup.go` is identical** in both services and matches login's gate logic; do not "improve" it — the empty-topic and end-offset-1 edge cases are load-bearing (covered by tests).
- **The `Id` assignment in `ApplyTenant`** (`cfg.Id = env.Id`) is the one intentional deviation from login's `state.go`; it exists because factory/world `tenant.RestModel.Id` is `json:"-"`. Do not drop it — `TestState_ApplyAndSnapshot_SetsId` guards it.
- **`PublishSnapshot` takes only `tenants`** (no service config) and closes `readyCh` unconditionally on first call — unlike login, which gates on `svc != nil`. These services have no service config.
- **World ordering matters:** the projection subscriber + `caughtUp` are created *before* `server.New(...).Run()` (so `/readyz` can read `caughtUp`), but the catch-up wait, bridge launch, and boot sweep run *after* `Run()` returns. `GetTenantConfigs()` blocks on `readyCh`, so the boot sweep is race-free.
- If any `docker buildx bake` step fails on a missing `COPY libs/...`, that is a pre-existing Dockerfile gap unrelated to this task (no new libs were added) — stop and report rather than editing the Dockerfile speculatively.
