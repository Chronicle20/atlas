# Kafka Writer Registry Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the per-publish `kafka.Writer` construct/close cycle in `libs/atlas-kafka/producer/` with a singleton, lazy, per-topic Writer registry, then migrate all 47 service `ProviderImpl` wrappers, 4 non-standard direct callsites, and 48 service `main.go` files to use it.

**Architecture:** A new `producer.Manager` type owns a `map[topic]Writer` guarded by `sync.RWMutex` with double-checked locking on first lookup. `GetManager()` is a `sync.Once`-driven singleton. A new `producer.ManagerWriterProvider(l)(token)` returns a `model.Provider[Writer]` thunk that the existing `producer.Produce` consumes unchanged. Each service registers `producer.GetManager().Close(l)` with `service.GetTeardownManager().TeardownFunc(...)` so Writers flush on shutdown. The existing `WriterProvider` helper is deleted; the existing `w.Close()` per-publish call is removed from `Produce`.

**Tech Stack:** Go 1.25, `github.com/segmentio/kafka-go` v0.4.51, `github.com/sirupsen/logrus`, `sync.RWMutex`/`sync.Once`, `model.Provider[T]` thunk pattern, `service.Manager.TeardownFunc`.

---

## File Structure

**Library (`libs/atlas-kafka/producer/`)**

| Path | Action | Responsibility |
|---|---|---|
| `manager.go` | Create | `Manager` type, singleton accessor, configurators, `defaultWriterFactory`, `ManagerWriterProvider`, `ErrManagerClosed`, `ResetInstance` (test-only) |
| `manager_test.go` | Create | Unit tests for lazy creation, concurrent first-touch, idempotent close, error-tolerant close, post-close behavior |
| `producer.go` | Modify | Delete `WriterProvider` (lines ~46–63); remove `w.Close()` block in `Produce` (lines ~90–93); remove now-unused `topic` import |
| `producer_test.go` | Untouched | Existing tests still pass — confirm during Task 8 |

**Per-service (47 services)**

| Path pattern | Action |
|---|---|
| `services/<svc>/atlas.com/<name>/kafka/producer/producer.go` | Replace `WriterProvider(topic.EnvProvider(l)(token))` with `ManagerWriterProvider(l)(token)`; drop `topic` import |
| `services/<svc>/atlas.com/<name>/main.go` | Add `tdm.TeardownFunc(func() { _ = producer.GetManager().Close(l) })` and ensure `producer` is imported |

**Non-standard direct callsites (4 files)**

| Path | Action |
|---|---|
| `services/atlas-quest/atlas.com/quest/kafka/producer/quest/producer.go` | Replace `producer.WriterProvider(topic.EnvProvider(l)(...))` in `emitEvent` with `producer.ManagerWriterProvider(l)(...)` |
| `services/atlas-quest/atlas.com/quest/kafka/producer/saga/producer.go` | Same substitution at the inline call |
| `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/party_quest/processor.go:173` | Same substitution |
| `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/reactor/processor.go:101` | Same substitution |

The full file lists for the 47 wrappers and 47 `main.go` files are in `context.md`.

---

## Task 1: Library — Manager skeleton + first test (lazy create)

**Files:**
- Create: `libs/atlas-kafka/producer/manager.go`
- Create: `libs/atlas-kafka/producer/manager_test.go`

- [ ] **Step 1: Write the failing test for lazy creation**

Add to `libs/atlas-kafka/producer/manager_test.go`:

```go
package producer

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus/hooks/test"
)

// fakeWriter is a Writer implementation used by manager tests. Tracks how many
// times Close was called and supports an injected close error.
type fakeWriter struct {
	topicName string
	closeErr  error
	closes    int32
}

func (f *fakeWriter) Topic() string { return f.topicName }
func (f *fakeWriter) WriteMessages(_ context.Context, _ ...kafka.Message) error {
	return nil
}
func (f *fakeWriter) Close() error {
	atomic.AddInt32(&f.closes, 1)
	return f.closeErr
}

func TestManager_LazyCreate(t *testing.T) {
	ResetInstance()
	var built int32
	factory := func(topicName string) Writer {
		atomic.AddInt32(&built, 1)
		return &fakeWriter{topicName: topicName}
	}
	m := GetManager(ConfigWriterFactory(factory))
	l, _ := test.NewNullLogger()

	w1, err := m.Writer(l, "MY_TOPIC")
	if err != nil {
		t.Fatalf("first Writer call returned error: %v", err)
	}
	w2, err := m.Writer(l, "MY_TOPIC")
	if err != nil {
		t.Fatalf("second Writer call returned error: %v", err)
	}
	if w1 != w2 {
		t.Fatalf("expected same Writer instance on repeat lookup; got distinct pointers")
	}
	if got := atomic.LoadInt32(&built); got != 1 {
		t.Fatalf("factory should be called exactly once; got %d", got)
	}
}

// Suppress unused-import warning until later tests reference these.
var _ = sync.Once{}
var _ = errors.New
```

- [ ] **Step 2: Run the test and confirm it fails**

```bash
cd libs/atlas-kafka && go test ./producer/ -run TestManager_LazyCreate -v
```

Expected: compile failure — `undefined: ResetInstance`, `undefined: GetManager`, `undefined: ConfigWriterFactory`.

- [ ] **Step 3: Create `manager.go` with the minimum surface to compile**

Create `libs/atlas-kafka/producer/manager.go`:

```go
package producer

import (
	"errors"
	"os"
	"sync"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

// WriterFactory builds a Writer for a resolved topic name. Tests inject
// a stub via ConfigWriterFactory; production uses defaultWriterFactory.
type WriterFactory func(topicName string) Writer

type ManagerConfig func(m *Manager)

//goland:noinspection GoUnusedExportedFunction
func ConfigWriterFactory(wf WriterFactory) ManagerConfig {
	return func(m *Manager) { m.wf = wf }
}

type Manager struct {
	mu      sync.RWMutex
	writers map[string]Writer
	wf      WriterFactory
	closed  bool
}

var (
	manager     *Manager
	managerOnce sync.Once
)

// ResetInstance clears the singleton. Test-only.
//
//goland:noinspection GoUnusedExportedFunction
func ResetInstance() {
	manager = nil
	managerOnce = sync.Once{}
}

//goland:noinspection GoUnusedExportedFunction
func GetManager(configurators ...ManagerConfig) *Manager {
	managerOnce.Do(func() {
		manager = &Manager{
			writers: make(map[string]Writer),
			wf:      defaultWriterFactory,
		}
		for _, c := range configurators {
			c(manager)
		}
	})
	return manager
}

var ErrManagerClosed = errors.New("producer manager is closed")

// Writer returns the long-lived Writer for the topic resolved from token,
// constructing it on first request. Concurrent first-touches return the
// same instance.
func (m *Manager) Writer(l logrus.FieldLogger, token string) (Writer, error) {
	t, err := topic.EnvProvider(l)(token)()
	if err != nil {
		return nil, err
	}

	m.mu.RLock()
	if m.closed {
		m.mu.RUnlock()
		return nil, ErrManagerClosed
	}
	if w, ok := m.writers[t]; ok {
		m.mu.RUnlock()
		return w, nil
	}
	m.mu.RUnlock()

	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return nil, ErrManagerClosed
	}
	if w, ok := m.writers[t]; ok { // double-check after acquiring write lock
		return w, nil
	}
	w := m.wf(t)
	m.writers[t] = w
	l.Infof("Created kafka writer for topic [%s].", t)
	return w, nil
}

// Close closes every registered Writer and marks the manager closed.
// Idempotent: subsequent calls are no-ops. Errors from individual
// Writer.Close calls are logged but do not short-circuit the loop.
func (m *Manager) Close(l logrus.FieldLogger) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return nil
	}
	m.closed = true

	var errCount int
	for t, w := range m.writers {
		if err := w.Close(); err != nil {
			errCount++
			l.WithError(err).Warnf("Error closing kafka writer for topic [%s].", t)
		}
	}
	l.Infof("Producer manager shut down %d writers (errors=%d).", len(m.writers), errCount)
	return nil
}

func defaultWriterFactory(topicName string) Writer {
	return WriterImpl{w: &kafka.Writer{
		Addr:                   kafka.TCP(os.Getenv("BOOTSTRAP_SERVERS")),
		Topic:                  topicName,
		Balancer:               &kafka.LeastBytes{},
		BatchTimeout:           50 * time.Millisecond,
		AllowAutoTopicCreation: true,
	}}
}

// ManagerWriterProvider returns a model.Provider[Writer] backed by the
// process-wide manager. Replaces the deleted WriterProvider helper.
//
//goland:noinspection GoUnusedExportedFunction
func ManagerWriterProvider(l logrus.FieldLogger) func(token string) model.Provider[Writer] {
	return func(token string) model.Provider[Writer] {
		return func() (Writer, error) {
			return GetManager().Writer(l, token)
		}
	}
}
```

- [ ] **Step 4: Run the test and confirm it passes**

```bash
cd libs/atlas-kafka && go test ./producer/ -run TestManager_LazyCreate -v
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-kafka/producer/manager.go libs/atlas-kafka/producer/manager_test.go
git commit -m "feat(atlas-kafka): add producer.Manager singleton with lazy Writer creation"
```

---

## Task 2: Library — Concurrent first-touch test

**Files:**
- Modify: `libs/atlas-kafka/producer/manager_test.go`

- [ ] **Step 1: Add the failing test**

Append to `libs/atlas-kafka/producer/manager_test.go`:

```go
func TestManager_ConcurrentFirstTouch(t *testing.T) {
	ResetInstance()
	var built int32
	factory := func(topicName string) Writer {
		atomic.AddInt32(&built, 1)
		return &fakeWriter{topicName: topicName}
	}
	m := GetManager(ConfigWriterFactory(factory))
	l, _ := test.NewNullLogger()

	const goroutines = 64
	results := make([]Writer, goroutines)
	var wg sync.WaitGroup
	wg.Add(goroutines)
	start := make(chan struct{})
	for i := 0; i < goroutines; i++ {
		i := i
		go func() {
			defer wg.Done()
			<-start
			w, err := m.Writer(l, "RACE_TOPIC")
			if err != nil {
				t.Errorf("goroutine %d: %v", i, err)
				return
			}
			results[i] = w
		}()
	}
	close(start)
	wg.Wait()

	if got := atomic.LoadInt32(&built); got != 1 {
		t.Fatalf("factory should be called exactly once across %d racers; got %d", goroutines, got)
	}
	for i := 1; i < goroutines; i++ {
		if results[i] != results[0] {
			t.Fatalf("goroutine %d returned a different Writer than goroutine 0", i)
		}
	}
}
```

- [ ] **Step 2: Run the test (with the race detector) and confirm it passes**

```bash
cd libs/atlas-kafka && go test -race ./producer/ -run TestManager_ConcurrentFirstTouch -v
```

Expected: PASS, no race detector warnings.

- [ ] **Step 3: Commit**

```bash
git add libs/atlas-kafka/producer/manager_test.go
git commit -m "test(atlas-kafka): assert concurrent first-touch single-instance contract"
```

---

## Task 3: Library — Idempotent Close test

**Files:**
- Modify: `libs/atlas-kafka/producer/manager_test.go`

- [ ] **Step 1: Add the failing test**

Append to `libs/atlas-kafka/producer/manager_test.go`:

```go
func TestManager_IdempotentClose(t *testing.T) {
	ResetInstance()
	fw := &fakeWriter{topicName: "T"}
	factory := func(topicName string) Writer { return fw }
	m := GetManager(ConfigWriterFactory(factory))
	l, _ := test.NewNullLogger()

	if _, err := m.Writer(l, "ANY_TOPIC"); err != nil {
		t.Fatalf("Writer: %v", err)
	}
	if err := m.Close(l); err != nil {
		t.Fatalf("first Close: %v", err)
	}
	if err := m.Close(l); err != nil {
		t.Fatalf("second Close: %v", err)
	}
	if got := atomic.LoadInt32(&fw.closes); got != 1 {
		t.Fatalf("underlying Writer.Close should be called exactly once; got %d", got)
	}
}
```

- [ ] **Step 2: Run the test and confirm it passes**

```bash
cd libs/atlas-kafka && go test ./producer/ -run TestManager_IdempotentClose -v
```

Expected: PASS (the existing `Close` impl already guards on `m.closed`).

- [ ] **Step 3: Commit**

```bash
git add libs/atlas-kafka/producer/manager_test.go
git commit -m "test(atlas-kafka): assert manager Close is idempotent"
```

---

## Task 4: Library — Close errors do not short-circuit

**Files:**
- Modify: `libs/atlas-kafka/producer/manager_test.go`

- [ ] **Step 1: Add the failing test**

Append to `libs/atlas-kafka/producer/manager_test.go`:

```go
func TestManager_CloseErrorsDoNotShortCircuit(t *testing.T) {
	ResetInstance()
	writers := map[string]*fakeWriter{
		"A": {topicName: "A"},
		"B": {topicName: "B", closeErr: errors.New("boom")},
		"C": {topicName: "C"},
	}
	factory := func(topicName string) Writer { return writers[topicName] }
	m := GetManager(ConfigWriterFactory(factory))
	l, _ := test.NewNullLogger()

	for _, k := range []string{"A", "B", "C"} {
		if _, err := m.Writer(l, k); err != nil {
			t.Fatalf("Writer(%s): %v", k, err)
		}
	}
	if err := m.Close(l); err != nil {
		t.Fatalf("Close: %v", err)
	}
	for k, w := range writers {
		if got := atomic.LoadInt32(&w.closes); got != 1 {
			t.Fatalf("writer %s closed %d times; want 1", k, got)
		}
	}
}
```

- [ ] **Step 2: Run the test and confirm it passes**

```bash
cd libs/atlas-kafka && go test ./producer/ -run TestManager_CloseErrorsDoNotShortCircuit -v
```

Expected: PASS (the existing `Close` impl already iterates without `return`-on-error).

- [ ] **Step 3: Commit**

```bash
git add libs/atlas-kafka/producer/manager_test.go
git commit -m "test(atlas-kafka): assert manager Close keeps closing on error"
```

---

## Task 5: Library — Writer after Close returns ErrManagerClosed

**Files:**
- Modify: `libs/atlas-kafka/producer/manager_test.go`

- [ ] **Step 1: Add the failing test**

Append to `libs/atlas-kafka/producer/manager_test.go`:

```go
func TestManager_WriterAfterClose(t *testing.T) {
	ResetInstance()
	factory := func(topicName string) Writer { return &fakeWriter{topicName: topicName} }
	m := GetManager(ConfigWriterFactory(factory))
	l, _ := test.NewNullLogger()

	if _, err := m.Writer(l, "PRE"); err != nil {
		t.Fatalf("pre-close Writer: %v", err)
	}
	if err := m.Close(l); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if _, err := m.Writer(l, "POST"); !errors.Is(err, ErrManagerClosed) {
		t.Fatalf("expected ErrManagerClosed; got %v", err)
	}
	// Topics already registered before close also reject lookups now.
	if _, err := m.Writer(l, "PRE"); !errors.Is(err, ErrManagerClosed) {
		t.Fatalf("expected ErrManagerClosed for already-registered topic; got %v", err)
	}
}
```

- [ ] **Step 2: Run the test and confirm it passes**

```bash
cd libs/atlas-kafka && go test ./producer/ -run TestManager_WriterAfterClose -v
```

Expected: PASS (the existing `Writer` impl checks `m.closed` under both R and W locks).

- [ ] **Step 3: Run the entire manager test suite plus race detector**

```bash
cd libs/atlas-kafka && go test -race ./producer/ -run TestManager -v
```

Expected: all `TestManager_*` PASS, no race warnings.

- [ ] **Step 4: Tidy the placeholder `_ = sync.Once{}` / `_ = errors.New` lines**

These were added in Task 1 to keep the imports live before subsequent tests landed. They are no longer needed because Task 2 references `sync.WaitGroup` and Tasks 4–5 reference `errors.New` / `errors.Is`. Remove the two trailing `var _ = ...` lines from the bottom of `manager_test.go`.

- [ ] **Step 5: Re-run the full library suite**

```bash
cd libs/atlas-kafka && go test -race ./... -v
```

Expected: all tests PASS (including the original `TestProducer` / `TestProducer2`).

- [ ] **Step 6: Commit**

```bash
git add libs/atlas-kafka/producer/manager_test.go
git commit -m "test(atlas-kafka): assert manager rejects Writer calls after Close"
```

---

## Task 6: Library — Remove per-publish Writer.Close and delete WriterProvider

**Files:**
- Modify: `libs/atlas-kafka/producer/producer.go`

- [ ] **Step 1: Edit `producer.go` to delete `WriterProvider`**

Open `libs/atlas-kafka/producer/producer.go` and remove the entire `WriterProvider` function (lines ~45–63 — the function with the `//goland:noinspection GoUnusedExportedFunction` directive immediately above it).

After the edit, the imports block must drop `"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"` because no other code in this file references it.

The remaining imports are:

```go
import (
	"context"
	"encoding/binary"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-retry"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)
```

(Note: `os` is also dropped because the only `os.Getenv` call lived inside the deleted `WriterProvider`. After deletion, `producer.go` no longer references `os`.)

- [ ] **Step 2: Edit `producer.go` to remove the per-publish close**

Inside the `Produce` function body, delete the post-loop `w.Close()` block. Replace:

```go
				err = w.Close()
				if err != nil {
					return err
				}

				return nil
```

with:

```go
				return nil
```

- [ ] **Step 3: Build the library and run all tests with race detection**

```bash
cd libs/atlas-kafka && go vet ./... && go test -race ./...
```

Expected: build clean, all tests PASS (including `TestProducer`, `TestProducer2`, and every `TestManager_*`).

- [ ] **Step 4: Verify `producer_test.go` was not modified**

```bash
git diff --stat libs/atlas-kafka/producer/producer_test.go
```

Expected: empty diff (no lines changed).

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-kafka/producer/producer.go
git commit -m "refactor(atlas-kafka): drop per-publish Writer.Close and delete WriterProvider"
```

---

## Task 7: Migrate one canonical service wrapper (atlas-buffs) end-to-end

**Files:**
- Modify: `services/atlas-buffs/atlas.com/buffs/kafka/producer/producer.go`
- Modify: `services/atlas-buffs/atlas.com/buffs/main.go`

This task validates the migration template against a single service before fanning out across the remaining 46.

- [ ] **Step 1: Rewrite the wrapper**

Open `services/atlas-buffs/atlas.com/buffs/kafka/producer/producer.go` and replace its full contents with:

```go
package producer

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/sirupsen/logrus"
)

type Provider func(token string) producer.MessageProducer

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

The only line-level diff vs. the original is:

```diff
-		"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
-		return producer.Produce(l)(producer.WriterProvider(topic.EnvProvider(l)(token)))(sd, td)
+		return producer.Produce(l)(producer.ManagerWriterProvider(l)(token))(sd, td)
```

- [ ] **Step 2: Add the teardown registration to `main.go`**

Open `services/atlas-buffs/atlas.com/buffs/main.go`. Add to the import block, alongside the existing `consumer` import:

```go
"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
```

Insert this line **after** the consumer-handler `InitHandlers` block and **before** the `go tasks.Register(...)` calls:

```go
	tdm.TeardownFunc(func() { _ = producer.GetManager().Close(l) })
```

The relevant region after the edit looks like:

```go
	cmf := consumer.GetManager().AddConsumer(l, tdm.Context(), tdm.WaitGroup())
	character2.InitConsumers(l)(cmf)(consumerGroupId)
	if err := character2.InitHandlers(l)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}

	tdm.TeardownFunc(func() { _ = producer.GetManager().Close(l) })

	go tasks.Register(tasks.NewExpiration(l, 10000))
	go tasks.Register(tasks.NewPoisonTick(l, 1000))
```

- [ ] **Step 3: Build and test atlas-buffs**

```bash
cd services/atlas-buffs/atlas.com/buffs && go build ./... && go test ./...
```

Expected: build clean, tests PASS.

- [ ] **Step 4: Verify the import sweep didn't leave dangling references**

```bash
grep -n "atlas-kafka/topic" services/atlas-buffs/atlas.com/buffs/kafka/producer/producer.go
grep -n "WriterProvider" services/atlas-buffs/
```

Expected: both empty (no matches).

- [ ] **Step 5: Commit**

```bash
git add services/atlas-buffs/atlas.com/buffs/kafka/producer/producer.go services/atlas-buffs/atlas.com/buffs/main.go
git commit -m "refactor(atlas-buffs): use producer.Manager registry for kafka writers"
```

---

## Task 8: Migrate the remaining 46 standard `ProviderImpl` wrappers + main.go

**Files:** the 46 service wrappers and 46 `main.go` files listed below. Every edit is identical in shape to Task 7.

Wrappers (apply Task 7 Step 1 verbatim, adjusting only the package name on line 1 if it differs):

```
services/atlas-account/atlas.com/account/kafka/producer/producer.go
services/atlas-asset-expiration/atlas.com/asset-expiration/kafka/producer/producer.go
services/atlas-ban/atlas.com/ban/kafka/producer/producer.go
services/atlas-buddies/atlas.com/buddies/kafka/producer/producer.go
services/atlas-cashshop/atlas.com/cashshop/kafka/producer/producer.go
services/atlas-chairs/atlas.com/chairs/kafka/producer/producer.go
services/atlas-chalkboards/atlas.com/chalkboards/kafka/producer/producer.go
services/atlas-channel/atlas.com/channel/kafka/producer/producer.go
services/atlas-character/atlas.com/character/kafka/producer/producer.go
services/atlas-character-factory/atlas.com/character-factory/kafka/producer/producer.go
services/atlas-consumables/atlas.com/consumables/kafka/producer/producer.go
services/atlas-data/atlas.com/data/kafka/producer/producer.go
services/atlas-drops/atlas.com/drops/kafka/producer/producer.go
services/atlas-effective-stats/atlas.com/effective-stats/kafka/producer/producer.go
services/atlas-expressions/atlas.com/expressions/kafka/producer/producer.go
services/atlas-fame/atlas.com/fame/kafka/producer/producer.go
services/atlas-families/atlas.com/family/kafka/producer/producer.go
services/atlas-guilds/atlas.com/guilds/kafka/producer/producer.go
services/atlas-inventory/atlas.com/inventory/kafka/producer/producer.go
services/atlas-invites/atlas.com/invites/kafka/producer/producer.go
services/atlas-keys/atlas.com/keys/kafka/producer/producer.go
services/atlas-login/atlas.com/login/kafka/producer/producer.go
services/atlas-map-actions/atlas.com/map-actions/kafka/producer/producer.go
services/atlas-maps/atlas.com/maps/kafka/producer/producer.go
services/atlas-marriages/atlas.com/marriages/kafka/producer/producer.go
services/atlas-merchant/atlas.com/merchant/kafka/producer/producer.go
services/atlas-messages/atlas.com/messages/kafka/producer/producer.go
services/atlas-messengers/atlas.com/messengers/kafka/producer/producer.go
services/atlas-monster-death/atlas.com/monster/kafka/producer/producer.go
services/atlas-monsters/atlas.com/monsters/kafka/producer/producer.go
services/atlas-notes/atlas.com/notes/kafka/producer/producer.go
services/atlas-npc-conversations/atlas.com/npc/kafka/producer/producer.go
services/atlas-npc-shops/atlas.com/npc/kafka/producer/producer.go
services/atlas-parties/atlas.com/parties/kafka/producer/producer.go
services/atlas-party-quests/atlas.com/party-quests/kafka/producer/producer.go
services/atlas-pets/atlas.com/pets/kafka/producer/producer.go
services/atlas-portal-actions/atlas.com/portal/kafka/producer/producer.go
services/atlas-portals/atlas.com/portals/kafka/producer/producer.go
services/atlas-reactor-actions/atlas.com/reactor/kafka/producer/producer.go
services/atlas-reactors/atlas.com/reactors/kafka/producer/producer.go
services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/producer/producer.go
services/atlas-skills/atlas.com/skills/kafka/producer/producer.go
services/atlas-storage/atlas.com/storage/kafka/producer/producer.go
services/atlas-tenants/atlas.com/tenants/kafka/producer/producer.go
services/atlas-transports/atlas.com/transports/kafka/producer/producer.go
services/atlas-world/atlas.com/world/kafka/producer/producer.go
```

Note: every wrapper above declares `package producer` on line 1 — confirmed via spot-check during planning. The body shown in Task 7 Step 1 may be pasted verbatim.

`main.go` files (apply Task 7 Step 2 verbatim — same import addition, same one-line teardown insertion after the consumer-handler block):

```
services/atlas-account/atlas.com/account/main.go
services/atlas-asset-expiration/atlas.com/asset-expiration/main.go
services/atlas-ban/atlas.com/ban/main.go
services/atlas-buddies/atlas.com/buddies/main.go
services/atlas-cashshop/atlas.com/cashshop/main.go
services/atlas-chairs/atlas.com/chairs/main.go
services/atlas-chalkboards/atlas.com/chalkboards/main.go
services/atlas-channel/atlas.com/channel/main.go
services/atlas-character/atlas.com/character/main.go
services/atlas-character-factory/atlas.com/character-factory/main.go
services/atlas-consumables/atlas.com/consumables/main.go
services/atlas-data/atlas.com/data/main.go
services/atlas-drops/atlas.com/drops/main.go
services/atlas-effective-stats/atlas.com/effective-stats/main.go
services/atlas-expressions/atlas.com/expressions/main.go
services/atlas-fame/atlas.com/fame/main.go
services/atlas-families/atlas.com/family/main.go
services/atlas-guilds/atlas.com/guilds/main.go
services/atlas-inventory/atlas.com/inventory/main.go
services/atlas-invites/atlas.com/invites/main.go
services/atlas-keys/atlas.com/keys/main.go
services/atlas-login/atlas.com/login/main.go
services/atlas-map-actions/atlas.com/map-actions/main.go
services/atlas-maps/atlas.com/maps/main.go
services/atlas-marriages/atlas.com/marriages/main.go
services/atlas-merchant/atlas.com/merchant/main.go
services/atlas-messages/atlas.com/messages/main.go
services/atlas-messengers/atlas.com/messengers/main.go
services/atlas-monster-death/atlas.com/monster/main.go
services/atlas-monsters/atlas.com/monsters/main.go
services/atlas-notes/atlas.com/notes/main.go
services/atlas-npc-conversations/atlas.com/npc/main.go
services/atlas-npc-shops/atlas.com/npc/main.go
services/atlas-parties/atlas.com/parties/main.go
services/atlas-party-quests/atlas.com/party-quests/main.go
services/atlas-pets/atlas.com/pets/main.go
services/atlas-portal-actions/atlas.com/portal/main.go
services/atlas-portals/atlas.com/portals/main.go
services/atlas-reactor-actions/atlas.com/reactor/main.go
services/atlas-reactors/atlas.com/reactors/main.go
services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/main.go
services/atlas-skills/atlas.com/skills/main.go
services/atlas-storage/atlas.com/storage/main.go
services/atlas-tenants/atlas.com/tenants/main.go
services/atlas-transports/atlas.com/transports/main.go
services/atlas-world/atlas.com/world/main.go
```

(Note: `services/atlas-quest/atlas.com/quest/main.go` is intentionally absent — atlas-quest's wrapper doesn't follow the standard `ProviderImpl` shape and is handled in Task 9 along with its non-standard direct callsites.)

- [ ] **Step 1: Rewrite each wrapper using the Task 7 Step 1 template**

For each wrapper path listed above, open the file and overwrite its body with the canonical shape (note: line 1 keeps the existing `package producer` declaration). The diff per file is the same two-line change from Task 7.

Tip: a per-file `Edit` is safer than a global `sed` because the original file lengths and decorators differ slightly across services, and a hand-verified rewrite catches any deviations the inventory might have missed.

- [ ] **Step 2: Add the teardown registration to each `main.go`**

For each `main.go` path listed above, apply the Task 7 Step 2 edit:
1. Add `"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"` to the import block if it is not already present.
2. Insert `tdm.TeardownFunc(func() { _ = producer.GetManager().Close(l) })` immediately after the consumer-handler `InitHandlers(...)` block and before `server.New(l)...Run()`.

Notes for variant `main.go` files:
- Some services don't have a `tasks.Register(...)` block — insert directly after `InitHandlers`.
- Some import the producer package as `producer` already; reuse that import.
- A handful of services (e.g. atlas-saga-orchestrator) have multiple `InitHandlers` calls — place the teardown line after the last one, still before `server.New(l)...Run()`.

- [ ] **Step 3: Build every affected service**

Run, from repo root:

```bash
for d in services/atlas-{account,asset-expiration,ban,buddies,cashshop,chairs,chalkboards,channel,character,character-factory,consumables,data,drops,effective-stats,expressions,fame,families,guilds,inventory,invites,keys,login,map-actions,maps,marriages,merchant,messages,messengers,monster-death,monsters,notes,npc-conversations,npc-shops,parties,party-quests,pets,portal-actions,portals,reactor-actions,reactors,saga-orchestrator,skills,storage,tenants,transports,world}/atlas.com/*; do
  echo "=== $d ===";
  (cd "$d" && go build ./... && go test ./...) || { echo "FAIL: $d"; exit 1; };
done
```

Expected: every directory prints `=== ... ===` and exits cleanly. The first failure halts the loop with `FAIL:` plus the path.

- [ ] **Step 4: Verify no wrapper still references the deleted helper**

```bash
grep -rn "producer\.WriterProvider" services/
```

Expected: matches only inside the four files Task 9 will handle (`services/atlas-quest/...` and `services/atlas-saga-orchestrator/...`). The 46 standard wrappers must produce zero hits.

- [ ] **Step 5: Commit**

```bash
git add services/
git commit -m "refactor(services): migrate 46 producer wrappers and main.go to producer.Manager"
```

---

## Task 9: Migrate the 4 non-standard direct producer callsites

**Files:**
- Modify: `services/atlas-quest/atlas.com/quest/kafka/producer/quest/producer.go`
- Modify: `services/atlas-quest/atlas.com/quest/kafka/producer/saga/producer.go`
- Modify: `services/atlas-quest/atlas.com/quest/main.go`
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/party_quest/processor.go`
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/reactor/processor.go`

These callsites bypass the per-service `ProviderImpl` wrapper and call `producer.WriterProvider(topic.EnvProvider(l)(...))` directly. They get the same substitution: `WriterProvider(topic.EnvProvider(l)(X))` → `ManagerWriterProvider(l)(X)`. The `topic` import is dropped only if no other line in the file references it.

- [ ] **Step 1: Edit `services/atlas-quest/atlas.com/quest/kafka/producer/quest/producer.go`**

Inside `emitEvent`, change:

```go
return producer.Produce(l)(producer.WriterProvider(topic.EnvProvider(l)(quest2.EnvStatusEventTopic)))(sd, td)
```

to:

```go
return producer.Produce(l)(producer.ManagerWriterProvider(l)(quest2.EnvStatusEventTopic))(sd, td)
```

Then remove `"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"` from the import block (no other reference in the file).

- [ ] **Step 2: Edit `services/atlas-quest/atlas.com/quest/kafka/producer/saga/producer.go`**

Change:

```go
return producer.Produce(l)(producer.WriterProvider(topic.EnvProvider(l)(topicToken)))(sd, td)(SagaCommandProvider(s))
```

to:

```go
return producer.Produce(l)(producer.ManagerWriterProvider(l)(topicToken))(sd, td)(SagaCommandProvider(s))
```

Then check whether `topic` is still imported elsewhere in the file. If not, remove the import.

- [ ] **Step 3: Edit `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/party_quest/processor.go:173`**

Change:

```go
return producer.Produce(l)(producer.WriterProvider(topic.EnvProvider(l)(EnvCommandTopic)))(sd, td)
```

to:

```go
return producer.Produce(l)(producer.ManagerWriterProvider(l)(EnvCommandTopic))(sd, td)
```

Inspect remaining uses of the `topic` package in the file. Remove the import only if no other reference survives.

- [ ] **Step 4: Edit `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/reactor/processor.go:101`**

Apply the same substitution as Step 3, dropping the `topic` import only if it becomes unused.

- [ ] **Step 5: Add the teardown registration to `services/atlas-quest/atlas.com/quest/main.go`**

Apply the Task 7 Step 2 edit: add `producer` import, insert `tdm.TeardownFunc(func() { _ = producer.GetManager().Close(l) })` after the consumer-handler `InitHandlers` block. atlas-saga-orchestrator was already covered in Task 8.

- [ ] **Step 6: Build atlas-quest and atlas-saga-orchestrator**

```bash
(cd services/atlas-quest/atlas.com/quest && go build ./... && go test ./...)
(cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go build ./... && go test ./...)
```

Expected: both clean, tests PASS.

- [ ] **Step 7: Verify no `producer.WriterProvider` references remain anywhere**

```bash
grep -rn "producer\.WriterProvider" services/ libs/
```

Expected: zero matches.

- [ ] **Step 8: Commit**

```bash
git add services/atlas-quest services/atlas-saga-orchestrator
git commit -m "refactor(quest,saga): migrate non-standard producer callsites to producer.Manager"
```

---

## Task 10: Repository-wide verification sweep

**Files:** none modified — this task only runs verification commands.

- [ ] **Step 1: Confirm the `WriterProvider` deletion took**

```bash
grep -rn "producer\.WriterProvider" services/ libs/
```

Expected: zero matches.

- [ ] **Step 2: Confirm the new helper is wired everywhere it should be**

```bash
grep -rln "producer\.ManagerWriterProvider" services/ | wc -l
```

Expected: `51`. Breakdown:
- 1 wrapper from Task 7 (`atlas-buffs`)
- 46 wrappers from Task 8
- 2 atlas-quest sub-wrappers from Task 9 (`kafka/producer/quest/producer.go`, `kafka/producer/saga/producer.go`)
- 2 atlas-saga-orchestrator processor files from Task 9 (`party_quest/processor.go`, `reactor/processor.go`)

- [ ] **Step 3: Confirm every service main.go registers the teardown**

```bash
grep -rln "producer\.GetManager().Close" services/ | wc -l
```

Expected: `48`. Breakdown:
- 1 from Task 7 (`atlas-buffs`)
- 46 from Task 8
- 1 from Task 9 (`atlas-quest`)

- [ ] **Step 4: Confirm no service constructs `kafka.Writer` directly**

```bash
grep -rn "kafka\.Writer{" services/
```

Expected: zero matches.

- [ ] **Step 5: Confirm no test asserts on per-publish `Close()`**

```bash
grep -rn "Close()" services/*/atlas.com/*/kafka/producer/*_test.go 2>/dev/null
```

Expected: zero matches (none of the per-service producer dirs have test files referencing `Close()`).

- [ ] **Step 6: Run `go vet` and `go test` across the library plus all migrated services**

Library:

```bash
(cd libs/atlas-kafka && go vet ./... && go test -race ./...)
```

Services — run a sweep loop over every service that was edited:

```bash
for d in $(grep -rln "producer\.ManagerWriterProvider" services/ | xargs -I{} dirname {} | xargs -I{} dirname {} | xargs -I{} dirname {} | sort -u); do
  echo "=== $d ===";
  (cd "$d" && go vet ./... && go build ./... && go test ./...) || { echo "FAIL: $d"; exit 1; };
done
```

Expected: every directory exits cleanly.

- [ ] **Step 7: Document smoke-test runbook handoff**

This task does not run the smoke test (it requires a live Kafka broker and a configured `command.data` topic with ≥4 partitions). Confirm the runbook in `docs/tasks/task-026-kafka-writer-registry/design.md` §8 is the canonical reference and link it from the PR description when the change ships:

- §8.1 Pre-flight: create `command.data` with 4 partitions
- §8.4 Drive a publish burst against `POST /api/data/process`
- §8.5 Pass criteria: ≥3 of 4 partitions show non-zero `CURRENT-OFFSET` advancement
- §8.6 Graceful-shutdown verification: log line `Producer manager shut down N writers (errors=0).`
- §8.7 Bonus check: `Created kafka writer for topic [command.data].` appears exactly once

- [ ] **Step 8: Commit (if any tracked file was tweaked during verification)**

If verification surfaced a missed file, fix it and commit with a focused message. If everything was clean, this step is a no-op (no commit needed).

```bash
git status
```

Expected (if clean): `nothing to commit, working tree clean`.

---

## Self-Review Notes (run before handoff)

- **Spec coverage:** every PRD §4 and §10 acceptance criterion maps to a task above —
  - §4.1 Writer Registry → Tasks 1–5
  - §4.2 Producer flow changes → Task 6
  - §4.3 Per-service producer wrappers → Tasks 7–9
  - §4.4 Service `main.go` integration → Tasks 7–9
  - §4.5 Backwards compatibility → Task 10 grep-based verification
  - §8.1–§8.4 NFRs → enforced by manager_test.go suite (Tasks 1–5) and Task 6 producer.go edit
  - §10 acceptance criteria — every box other than the manual smoke test/graceful-shutdown verification is covered by Tasks 1–10; the manual checks live in `design.md` §8 and are referenced in Task 10 Step 7.

- **Placeholder scan:** no `TBD`, `add appropriate error handling`, or `similar to Task N` references survive — every code block is the actual content the executor pastes.

- **Type/name consistency:** `Manager`, `Writer`, `WriterFactory`, `ManagerConfig`, `ConfigWriterFactory`, `GetManager`, `ResetInstance`, `ErrManagerClosed`, `ManagerWriterProvider`, `defaultWriterFactory` — all spelled identically across Tasks 1–10. The fakeWriter type is introduced in Task 1 and re-used by Tasks 2–5 without renames.

- **Known design deviation:** `TestManager_TopicResolutionError` (listed in `design.md` §3.5) is intentionally omitted because the live `topic.EnvProvider` cannot return an error today. The error-propagation branch in `Manager.Writer` remains as defensive code; documented in `context.md` "Deviation from design".
