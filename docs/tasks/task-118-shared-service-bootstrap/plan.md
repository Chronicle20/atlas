# Shared Service Bootstrap Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** One canonical Kafka producer wrapper (`libs/atlas-kafka/producer.Provider/ProviderImpl`), one canonical logger with snake_case field normalization (`libs/atlas-service.CreateLogger`), and one `service.Bootstrap(serviceName, opts...) *Runtime` owning logger/tracer/teardown/readiness/projection wiring — with all 58 Go service `main.go`s migrated onto it and every local copy deleted.

**Architecture:** Functional options + `Runtime` handle (design D4). Readiness stays mounted at effective `/api/readyz` via one explicit `MountReadiness` line per `main.go` (D5). The projection option wires around each of the four projection services' service-local `projection` packages through the two-method `Projection` interface, adapted via the lib's `ProjectionFuncs` struct (D6). atlas-renders adopts Bootstrap without a tracer, keeps its root mux and `/healthz`, and gains a root `/readyz` (D7).

**Tech Stack:** Go 1.25.x, logrus v1.9.4, ecslogrus v1.0.0, google/uuid v1.6.0, gorilla/mux, existing `libs/atlas-tracing`, `libs/atlas-kafka`, `libs/atlas-rest`.

## Global Constraints

- **Sequencing gate (FR-6.1):** No fleet edits until task-114 (outbox adoption) and task-116 (processor gen3 unification) are merged to main AND this branch is rebased onto that state. Task 0 is a blocking precondition for every other task.
- **Re-measure rule (FR-6.2):** The measured shapes in this plan are a 2026-07-02 snapshot. Task 0 re-runs the measurement commands; if the canonical `producer.go`/`main.go`/logger shapes changed, the verbatim bodies in Tasks 1–3 and the Recipes are re-derived from the post-rebase canon before any sweep. The design decisions (D1–D7) are shape-independent and stand.
- **No re-export shims:** no type alias, no wrapper package, no `producer.go` that forwards to the lib (owner decision; CLAUDE.md "straightforward moves over re-exports").
- **Behavior preservation:** the ONLY intentional observable changes fleet-wide are (a) snake_case log keys, (b) new `/readyz` routes, (c) the three design-accepted micro-changes: projection subscriber starts slightly earlier in login/channel; the projection warn condition unifies to "tenant topic unset"; atlas-renders' log format becomes ecslogrus. Anything else different at runtime is a bug.
- **Exact log messages preserved** (grep-verifiable): `"Starting main service."`, `"Service shutdown."`, `"Unable to initialize tracer."`, `"Unable to start configuration projection subscriber."`, `"Configuration projection failed to catch up."`, `"Flipped /readyz to not-ready for graceful shutdown."`, the projection warn line (world/character-factory spelling: `"projection: EVENT_TOPIC_CONFIGURATION_TENANT_STATUS is not set; tenant config updates will not propagate live"`).
- **Service `go.mod`/`go.sum` files are NOT touched** in the sweep (no `go mod tidy` per service). Unused requires (e.g. `atlas-tracing`, `ecslogrus` after main.go stops importing them directly) are harmless and keep the module graph + replace directives valid — `replace` directives are only honored from the main module, so every service MUST keep its `atlas-tracing` require+replace because `libs/atlas-service` now depends on it. Exception: atlas-renders (Task 12) genuinely gains new deps and runs `GOWORK=off go mod tidy` after adding the required `replace` lines.
- **Commit discipline:** lib commits first (Tasks 1–5), then exactly one commit per service (Tasks 6–12), then docs (Task 13). All on branch `task-118-shared-service-bootstrap`; verify `git branch --show-current` after each commit.
- **Verification (CLAUDE.md, all mandatory before "done"):** `go test -race ./...`, `go vet ./...`, `go build ./...` clean in every changed module; `docker buildx bake all-go-services` from the worktree root; `tools/redis-key-guard.sh` clean.
- No `// TODO`, stubs, or 501s in any commit. No absolute home paths written into files.

## Plan-time corrections to design §2 (measured 2026-07-02)

1. **57 local logger packages, not 56.** atlas-monster-book's copy is `logger/logger.go` (not `init.go`) — byte-identical to the canon (`md5 473b31e275b2900d442a9915fb6a095a`, same as atlas-fame's `init.go`). The acceptance grep must cover both names.
2. **One non-main importer of a local logger package:** `services/atlas-cashshop/atlas.com/cashshop/cashshop/inventory/rest_test.go` imports `"atlas-cashshop/logger"`. It must be rewritten to the lib `service.CreateLogger` in atlas-cashshop's commit.
3. **atlas-storage has a domain package named `projection`** (`atlas-storage/projection`, storage-document projections). It is NOT the configuration projection; do not touch it. atlas-storage is a plain Cohort A service.
4. **232 files** import a service-local `kafka/producer` package (matches design).

## Migration Recipes (referenced by Tasks 6–12)

These are plan-level shared material; every cohort task applies them verbatim. `<svc>` is the service directory name without the `atlas-` prefix (e.g. `services/atlas-fame/atlas.com/fame`).

### Recipe R1 — producer import sweep (services with a local `kafka/producer/producer.go`)

1. Delete the wrapper: `git rm services/atlas-<svc>/atlas.com/<svc>/kafka/producer/producer.go`. Keep every sibling under `kafka/producer/<domain>/` and any `producer_test.go` (atlas-reactors, atlas-marriages).
2. Rewrite all import sites (the module path inside quotes is the service's short module name, e.g. `atlas-fame`):
   ```bash
   grep -rl '"atlas-<svc>/kafka/producer"' services/atlas-<svc> --include='*.go' \
     | xargs -r sed -i 's|"atlas-<svc>/kafka/producer"|"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"|g'
   ```
   Both packages are named `producer`, so unaliased call sites (`producer.Provider`, `producer.ProviderImpl`) compile unchanged. Aliased local imports (e.g. `producer2 "atlas-x/kafka/producer"`) survive as aliases of the lib — also fine.
3. `cd services/atlas-<svc>/atlas.com/<svc> && go build ./...`. Fix the two known error classes:
   - **Duplicate import** (file imported the local wrapper AND the lib, e.g. `kproducer "github.com/Chronicle20/atlas/libs/atlas-kafka/producer"` alongside the now-rewritten unaliased import): delete one import line, keep a single import named `producer`, and rename the dropped alias's references (`kproducer.X` → `producer.X`).
   - **Unused import** where a file only used the local wrapper for its `Provider` type and the sed made it self-referential: remove per compiler guidance.

### Recipe R2 — `main.go` Bootstrap rewrite

Apply these exact edits to `services/atlas-<svc>/atlas.com/<svc>/main.go`:

1. Replace this canonical opening block:
   ```go
   l := logger.CreateLogger(serviceName)
   l.Infoln("Starting main service.")

   tdm := service.GetTeardownManager()

   tc, err := tracing.InitTracer(serviceName)
   if err != nil {
   	l.WithError(err).Fatal("Unable to initialize tracer.")
   }
   ```
   with:
   ```go
   rt := service.Bootstrap(serviceName)
   l := rt.Logger()
   ```
2. Replace every `tdm.Context()` → `rt.Context()`, `tdm.WaitGroup()` → `rt.WaitGroup()`, `tdm.TeardownFunc(` → `rt.TeardownFunc(`.
3. Delete the line `tdm.TeardownFunc(tracing.Teardown(l)(tc))` (Bootstrap registers it).
4. If a callee is typed on `*service.Manager` (e.g. login's `buildListener`), pass `rt.TeardownManager()`.
5. In the REST builder chain, add directly after the `MountHandler("/debug/consumers", ...)` initializer (or as the last `AddRouteInitializer` if there is none):
   ```go
   AddRouteInitializer(server.MountReadiness("/readyz", rt.Ready)).
   ```
   (use the file's actual alias — `restserver.MountReadiness` in login/channel).
6. Replace the trailing pair:
   ```go
   tdm.Wait()
   l.Infoln("Service shutdown.")
   ```
   with:
   ```go
   rt.Wait()
   ```
7. Remove now-unused imports: `"atlas-<svc>/logger"` and `tracing "github.com/Chronicle20/atlas/libs/atlas-tracing"`. Keep `"github.com/Chronicle20/atlas/libs/atlas-service"` (now used for `service.Bootstrap`).
8. Delete the local logger package: `git rm -r services/atlas-<svc>/atlas.com/<svc>/logger`.

### Recipe R3 — per-service verify + commit

```bash
cd services/atlas-<svc>/atlas.com/<svc>
go build ./... && go vet ./... && go test -race ./...
cd -  # back to worktree root
git add -A services/atlas-<svc>
git commit -m "refactor(<svc>): migrate to service.Bootstrap"
git branch --show-current   # must print task-118-shared-service-bootstrap
```
Expected: build/vet/test all clean; any failure is fixed inside the same service before committing.

---

### Task 0: Rebase gate + re-measure (BLOCKING)

**Files:**
- No code files. Produces a short measurement note appended to `docs/tasks/task-118-shared-service-bootstrap/context.md` ("Post-rebase measurements" section).

**Interfaces:**
- Produces: a green light (or STOP) for Tasks 1–14, plus refreshed counts/shapes.

- [ ] **Step 1: Verify task-114 and task-116 are merged to main**

Run:
```bash
git fetch origin main
git log origin/main --oneline | grep -iE "task-114|outbox" | head -5
git log origin/main --oneline | grep -iE "task-116|gen3" | head -5
```
Expected: both greps show merge/squash commits. **If either is missing, STOP — report BLOCKED on the sequencing gate. Do not proceed to any other task.**

- [ ] **Step 2: Rebase this branch onto main**

Run:
```bash
git rebase origin/main
```
Expected: clean rebase (this branch only carries `docs/tasks/` commits at this point). Resolve doc-only conflicts if any.

- [ ] **Step 3: Re-run the design §2 measurement commands**

Run from the worktree root:
```bash
find services -name main.go -path '*/atlas.com/*' | wc -l
find services -path '*/atlas.com/*/logger/init.go' -o -path '*/atlas.com/*/logger/logger.go' | wc -l
find services -path '*/atlas.com/*/kafka/producer/producer.go' | wc -l
find services -path '*/atlas.com/*/kafka/producer/producer.go' -exec md5sum {} + | awk '{print $1}' | sort | uniq -c
grep -rl 'parseProjectionCatchupTimeout' services | wc -l
md5sum services/atlas-fame/atlas.com/fame/logger/init.go services/atlas-fame/atlas.com/fame/kafka/producer/producer.go
diff <(cat services/atlas-fame/atlas.com/fame/main.go) <(git show HEAD~0:services/atlas-fame/atlas.com/fame/main.go) >/dev/null && echo unchanged
```
Expected (pre-rebase snapshot): 58 mains, 57 logger files, 52 wrappers (51× one hash + atlas-quest), 4 projection helpers.

- [ ] **Step 4: Compare against the snapshot and re-derive if drifted**

If the wrapper hash census, the atlas-fame `main.go` shape, or the projection blocks differ from the bodies quoted in this plan (task-114/116 rewrote them): re-read the new canonical files and update the verbatim code in Tasks 1, 6, 10, 11 and Recipes R1/R2 to the post-rebase canon. Record what changed in `context.md` under "Post-rebase measurements". The lib API surface (Task 4/5 signatures) does not change.

- [ ] **Step 5: Commit the measurement note**

```bash
git add docs/tasks/task-118-shared-service-bootstrap/context.md
git commit -m "chore(task-118): post-rebase measurements"
```

---

### Task 1: `libs/atlas-kafka/producer` — `Provider` + `ProviderImpl`

**Files:**
- Create: `libs/atlas-kafka/producer/provider.go`
- Test: `libs/atlas-kafka/producer/provider_test.go`

**Interfaces:**
- Consumes: existing `Produce`, `ManagerWriterProvider`, `SpanHeaderDecorator`, `TenantHeaderDecorator`, `MessageProducer` (all already in this package).
- Produces: `type Provider func(token string) MessageProducer`; `func ProviderImpl(l logrus.FieldLogger) func(ctx context.Context) Provider`. Every service import site in Tasks 6–12 relies on exactly these two names.

- [ ] **Step 1: Write the failing test**

`libs/atlas-kafka/producer/provider_test.go` (same package `producer`; reuses the `MockWriter` already defined in `producer_test.go`):

```go
package producer

import (
	"context"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus/hooks/test"
)

func TestProviderImplComposesSpanAndTenantHeaders(t *testing.T) {
	ResetInstance()
	t.Cleanup(ResetInstance)

	mw := &MockWriter{topic: "provider-test-topic"}
	GetManager(ConfigWriterFactory(func(topicName string) Writer { return mw }))
	t.Setenv("EVENT_TOPIC_PROVIDER_TEST", "provider-test-topic")

	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx := tenant.WithContext(context.Background(), ten)

	l, _ := test.NewNullLogger()

	var p Provider = ProviderImpl(l)(ctx) // compile-time: returns the named Provider type
	if err := p("EVENT_TOPIC_PROVIDER_TEST")(model.FixedProvider([]kafka.Message{{Value: []byte("v")}})); err != nil {
		t.Fatalf("produce: %v", err)
	}

	if len(mw.writtenMessages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(mw.writtenMessages))
	}
	headers := map[string]string{}
	for _, h := range mw.writtenMessages[0].Headers {
		headers[h.Key] = string(h.Value)
	}
	if headers[tenant.ID] != ten.Id().String() {
		t.Errorf("missing/wrong tenant id header: %q", headers[tenant.ID])
	}
	if headers[tenant.Region] != "GMS" {
		t.Errorf("missing/wrong region header: %q", headers[tenant.Region])
	}
}
```

Note: if `MockWriter`'s field names in `producer_test.go` differ post-rebase, adapt the test to the actual fake — do not add a new fake.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd libs/atlas-kafka && go test -race ./producer/ -run TestProviderImplComposesSpanAndTenantHeaders -v`
Expected: FAIL — `undefined: Provider` / `undefined: ProviderImpl`.

- [ ] **Step 3: Write the implementation**

`libs/atlas-kafka/producer/provider.go`:

```go
package producer

import (
	"context"

	"github.com/sirupsen/logrus"
)

// Provider resolves a topic token to a ready-to-use MessageProducer.
type Provider func(token string) MessageProducer

// ProviderImpl is the canonical provider: span + tenant header decorators
// over the manager-owned writer for the token's topic.
func ProviderImpl(l logrus.FieldLogger) func(ctx context.Context) Provider {
	return func(ctx context.Context) Provider {
		sd := SpanHeaderDecorator(ctx)
		td := TenantHeaderDecorator(ctx)
		return func(token string) MessageProducer {
			return Produce(l)(ManagerWriterProvider(l)(token))(sd, td)
		}
	}
}
```

This is the 51-way-identical service wrapper verbatim with intra-package references. If Task 0 found a post-114 canon with a different body, use THAT body instead (same file/test structure).

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd libs/atlas-kafka && go test -race ./... && go vet ./...`
Expected: PASS, vet clean.

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-kafka/producer/provider.go libs/atlas-kafka/producer/provider_test.go
git commit -m "feat(atlas-kafka): add canonical Provider/ProviderImpl to producer lib"
```

---

### Task 2: `libs/atlas-service` — snake_case field-key normalizer hook (CP-9)

**Files:**
- Create: `libs/atlas-service/fieldnorm.go`
- Test: `libs/atlas-service/fieldnorm_test.go`
- Modify: `libs/atlas-service/go.mod` (add logrus)

**Interfaces:**
- Produces: unexported `fieldKeyNormalizerHook{}` (logrus.Hook) and `normalizeFieldKey(k string) (string, bool)`. Task 3's `CreateLogger` registers the hook last.

- [ ] **Step 1: Write the failing tests**

`libs/atlas-service/fieldnorm_test.go`:

```go
package service

import (
	"testing"

	"github.com/sirupsen/logrus"
)

func TestNormalizeFieldKey(t *testing.T) {
	tests := []struct {
		in      string
		want    string
		changed bool
	}{
		{"characterId", "character_id", true},
		{"characterID", "character_id", true},
		{"transactionId", "transaction_id", true},
		{"worldId2", "world_id2", true},
		{"HTTPServer", "http_server", true},
		{"Name", "name", true},
		{"character_id", "character_id", false}, // already snake
		{"tenant", "tenant", false},             // plain lowercase
		{"service.name", "service.name", false}, // dotted ECS key passes through
		{"ms.Version", "ms.Version", false},     // any dotted key passes through, even with uppercase
	}
	for _, tc := range tests {
		got, changed := normalizeFieldKey(tc.in)
		if got != tc.want || changed != tc.changed {
			t.Errorf("normalizeFieldKey(%q) = (%q, %v), want (%q, %v)", tc.in, got, changed, tc.want, tc.changed)
		}
	}
}

func fireNormalizer(t *testing.T, data logrus.Fields) logrus.Fields {
	t.Helper()
	entry := &logrus.Entry{Data: data}
	if err := (fieldKeyNormalizerHook{}).Fire(entry); err != nil {
		t.Fatal(err)
	}
	return entry.Data
}

func TestNormalizerHookRewritesKeys(t *testing.T) {
	got := fireNormalizer(t, logrus.Fields{"characterId": 42, "world_id": 1, "service.name": "x"})
	if got["character_id"] != 42 {
		t.Errorf("character_id = %v, want 42", got["character_id"])
	}
	if _, ok := got["characterId"]; ok {
		t.Error("camelCase key survived")
	}
	if got["world_id"] != 1 || got["service.name"] != "x" {
		t.Errorf("passthrough keys damaged: %v", got)
	}
}

func TestNormalizerHookCollisionSnakeCaseWins(t *testing.T) {
	got := fireNormalizer(t, logrus.Fields{"characterId": 1, "character_id": 2})
	if got["character_id"] != 2 {
		t.Errorf("collision: character_id = %v, want the explicit snake_case value 2", got["character_id"])
	}
	if len(got) != 1 {
		t.Errorf("expected exactly 1 key after collision, got %v", got)
	}
}

func TestNormalizerHookIdempotent(t *testing.T) {
	data := logrus.Fields{"characterId": 42}
	first := fireNormalizer(t, data)
	second := fireNormalizer(t, first)
	if second["character_id"] != 42 || len(second) != 1 {
		t.Errorf("second pass changed data: %v", second)
	}
}
```

- [ ] **Step 2: Add logrus to the lib's go.mod and verify the tests fail**

Run:
```bash
cd libs/atlas-service
go get github.com/sirupsen/logrus@v1.9.4
go test -race ./... 2>&1 | head -20
```
Expected: FAIL — `undefined: normalizeFieldKey` / `undefined: fieldKeyNormalizerHook`.

- [ ] **Step 3: Write the implementation**

`libs/atlas-service/fieldnorm.go`:

```go
package service

import (
	"sort"
	"strings"

	"github.com/sirupsen/logrus"
)

// fieldKeyNormalizerHook rewrites camelCase log field keys to snake_case at
// emit time so the fleet's ~1,500 legacy WithField call sites converge on
// one spelling without a rename sweep (CP-9).
//
// Safe to mutate entry.Data in place: logrus v1.9.4 duplicates the entry
// (including the Data map) before firing hooks (entry.Dup(), entry.go:227),
// so callers retaining a derived *Entry never observe the rewrite.
//
// Ordering caveat: keys added by hooks registered AFTER this one escape
// normalization. CreateLogger registers it last.
type fieldKeyNormalizerHook struct{}

func (fieldKeyNormalizerHook) Levels() []logrus.Level { return logrus.AllLevels }

func (fieldKeyNormalizerHook) Fire(entry *logrus.Entry) error {
	var renames [][2]string // nil for the common fully-normalized entry: zero allocation
	for k := range entry.Data {
		if nk, changed := normalizeFieldKey(k); changed {
			renames = append(renames, [2]string{k, nk})
		}
	}
	if renames == nil {
		return nil
	}
	// Sort so collision resolution is deterministic regardless of map order.
	sort.Slice(renames, func(i, j int) bool { return renames[i][0] < renames[j][0] })
	for _, r := range renames {
		v := entry.Data[r[0]]
		delete(entry.Data, r[0])
		// Collision rule: an explicitly snake_case key wins; the camelCase
		// duplicate is dropped (documented in docs/observability.md).
		if _, exists := entry.Data[r[1]]; !exists {
			entry.Data[r[1]] = v
		}
	}
	return nil
}

// normalizeFieldKey converts a camelCase ASCII key to snake_case. Keys
// containing a dot (ECS/namespaced, e.g. service.name) and keys with no
// uppercase letters pass through unchanged (changed=false, no allocation).
func normalizeFieldKey(k string) (string, bool) {
	if strings.ContainsRune(k, '.') {
		return k, false
	}
	hasUpper := false
	for i := 0; i < len(k); i++ {
		if k[i] >= 'A' && k[i] <= 'Z' {
			hasUpper = true
			break
		}
	}
	if !hasUpper {
		return k, false
	}
	isLowerOrDigit := func(c byte) bool { return (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') }
	isUpper := func(c byte) bool { return c >= 'A' && c <= 'Z' }
	var b strings.Builder
	b.Grow(len(k) + 4)
	for i := 0; i < len(k); i++ {
		c := k[i]
		if isUpper(c) {
			if i > 0 && isLowerOrDigit(k[i-1]) {
				// lower/digit → upper boundary: characterId → character_id
				b.WriteByte('_')
			} else if i > 0 && isUpper(k[i-1]) && i+1 < len(k) && k[i+1] >= 'a' && k[i+1] <= 'z' {
				// last upper of an upper-run followed by lower: HTTPServer → http_server
				b.WriteByte('_')
			}
			b.WriteByte(c + ('a' - 'A'))
		} else {
			b.WriteByte(c)
		}
	}
	nk := b.String()
	return nk, nk != k
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd libs/atlas-service && go test -race ./... && go vet ./...`
Expected: PASS (including the existing teardown tests), vet clean.

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-service/fieldnorm.go libs/atlas-service/fieldnorm_test.go libs/atlas-service/go.mod libs/atlas-service/go.sum
git commit -m "feat(atlas-service): snake_case log field-key normalizer hook (CP-9)"
```

---

### Task 3: `libs/atlas-service` — `CreateLogger` (DUP-2)

**Files:**
- Create: `libs/atlas-service/logger.go`
- Test: `libs/atlas-service/logger_test.go`
- Modify: `libs/atlas-service/go.mod` (+ ecslogrus)

**Interfaces:**
- Consumes: `fieldKeyNormalizerHook` from Task 2.
- Produces: `func CreateLogger(serviceName string) *logrus.Logger`. Task 4's `Bootstrap` calls it; atlas-cashshop's `rest_test.go` calls it directly.

- [ ] **Step 1: Write the failing tests**

`libs/atlas-service/logger_test.go`:

```go
package service

import (
	"bytes"
	"strings"
	"sync"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestCreateLoggerEmitsServiceNameAndNormalizedKeys(t *testing.T) {
	l := CreateLogger("atlas-test")
	var buf bytes.Buffer
	l.SetOutput(&buf)

	l.WithField("characterId", 42).Info("hello")

	out := buf.String()
	if !strings.Contains(out, "character_id") {
		t.Errorf("emitted record missing normalized key: %s", out)
	}
	if strings.Contains(out, "characterId") {
		t.Errorf("emitted record still contains camelCase key: %s", out)
	}
	if !strings.Contains(out, "atlas-test") {
		t.Errorf("emitted record missing service name: %s", out)
	}
}

func TestCreateLoggerLogLevelEnv(t *testing.T) {
	t.Setenv("LOG_LEVEL", "debug")
	if l := CreateLogger("atlas-test"); l.GetLevel() != logrus.DebugLevel {
		t.Errorf("LOG_LEVEL=debug not honored, got %v", l.GetLevel())
	}
	t.Setenv("LOG_LEVEL", "not-a-level")
	if l := CreateLogger("atlas-test"); l.GetLevel() != logrus.InfoLevel {
		t.Errorf("invalid LOG_LEVEL must silently keep the default, got %v", l.GetLevel())
	}
}

// Pin the logrus v1.9.4 safety property the normalizer relies on: hooks fire
// on a per-emission copy (entry.Dup()), so a shared derived entry logged
// from parallel goroutines does not race the in-place key rewrite.
func TestCreateLoggerSharedEntryParallelEmitNoRace(t *testing.T) {
	l := CreateLogger("atlas-test")
	l.SetOutput(&safeBuffer{})
	e := l.WithField("characterId", 42)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				e.Info("parallel")
			}
		}()
	}
	wg.Wait()
}

type safeBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *safeBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd libs/atlas-service && go test -race ./... 2>&1 | head -10`
Expected: FAIL — `undefined: CreateLogger`.

- [ ] **Step 3: Write the implementation**

`libs/atlas-service/logger.go` — the canonical 53-way-identical service body, with the normalizer registered LAST:

```go
package service

import (
	"os"

	"github.com/sirupsen/logrus"
	"go.elastic.co/ecslogrus"
)

// CreateLogger is the fleet-canonical logger: stdout, ECS JSON formatting,
// a service.name field on every record, LOG_LEVEL env parsing (invalid
// values silently keep the default), and emit-time snake_case field-key
// normalization (see fieldnorm.go). The normalizer must stay the LAST
// registered hook so it sees keys added by earlier hooks.
func CreateLogger(serviceName string) *logrus.Logger {
	l := logrus.New()
	l.SetOutput(os.Stdout)
	l.AddHook(newServiceNameHook(serviceName))
	l.SetFormatter(&ecslogrus.Formatter{})
	if val, ok := os.LookupEnv("LOG_LEVEL"); ok {
		if level, err := logrus.ParseLevel(val); err == nil {
			l.SetLevel(level)
		}
	}
	l.AddHook(fieldKeyNormalizerHook{})
	return l
}

type serviceNameHook struct {
	service string
}

func newServiceNameHook(name string) *serviceNameHook {
	return &serviceNameHook{service: name}
}

func (h *serviceNameHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (h *serviceNameHook) Fire(entry *logrus.Entry) error {
	entry.Data["service.name"] = h.service
	return nil
}
```

Then: `go get go.elastic.co/ecslogrus@v1.0.0`.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd libs/atlas-service && go test -race ./... && go vet ./...`
Expected: PASS (the race test exercises the Dup() safety under `-race`), vet clean.

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-service/logger.go libs/atlas-service/logger_test.go libs/atlas-service/go.mod libs/atlas-service/go.sum
git commit -m "feat(atlas-service): canonical CreateLogger with ECS formatting (DUP-2)"
```

---

### Task 4: `libs/atlas-service` — `Bootstrap`, `Runtime`, readiness controller

**Files:**
- Create: `libs/atlas-service/bootstrap.go`
- Test: `libs/atlas-service/bootstrap_test.go`, `libs/atlas-service/zz_lifecycle_test.go`
- Modify: `libs/atlas-service/go.mod` (+ atlas-tracing require & replace)

**Interfaces:**
- Consumes: `CreateLogger` (Task 3), existing `GetTeardownManager`/`Manager`, `tracing.InitTracer(serviceName) (*sdktrace.TracerProvider, error)` and `tracing.Teardown(l)(tp) func()` from `libs/atlas-tracing`.
- Produces (used by every migrated `main.go`):
  - `func Bootstrap(serviceName string, opts ...Option) *Runtime`
  - `type Option func(*bootstrapConfig)`; `func WithoutTracer() Option`; `func WithReadinessGate(fn func() bool) Option`
  - `Runtime` methods: `Logger() *logrus.Logger`, `Context() context.Context`, `WaitGroup() *sync.WaitGroup`, `TeardownFunc(f func())`, `TeardownManager() *Manager`, `Ready() bool`, `Wait()`
  - (Task 5 adds `WithConfigProjection` + `AwaitProjectionCatchUp` and the `rt.projection` field it arms.)

- [ ] **Step 1: Add the atlas-tracing dependency**

Append to `libs/atlas-service/go.mod`:
```
replace github.com/Chronicle20/atlas/libs/atlas-tracing => ../atlas-tracing
```
Then run: `cd libs/atlas-service && go get github.com/Chronicle20/atlas/libs/atlas-tracing@v0.0.0 || true` — if `go get` rejects the pseudo-version, add `require github.com/Chronicle20/atlas/libs/atlas-tracing v0.0.0` manually (matching `libs/atlas-kafka/go.mod`'s pattern for sibling libs) and run `GOWORK=off go mod tidy`.
Expected: `go build ./...` clean. No import cycle: atlas-tracing imports only otel + logrus.

- [ ] **Step 2: Write the failing tests**

`libs/atlas-service/bootstrap_test.go`:

```go
package service

import (
	"sync/atomic"
	"testing"
)

// All Bootstrap tests use WithoutTracer so unit tests don't install global
// otel state. GetTeardownManager is a process-wide singleton; that is fine —
// each Bootstrap call just registers more teardown funcs on it.

func TestBootstrapRuntimeAccessors(t *testing.T) {
	rt := Bootstrap("atlas-test", WithoutTracer())
	if rt.Logger() == nil {
		t.Fatal("Logger() is nil")
	}
	if rt.Context() == nil {
		t.Fatal("Context() is nil")
	}
	if rt.WaitGroup() == nil {
		t.Fatal("WaitGroup() is nil")
	}
	if rt.TeardownManager() != GetTeardownManager() {
		t.Fatal("TeardownManager() must return the process singleton")
	}
	if !rt.Ready() {
		t.Fatal("fresh Runtime must be Ready")
	}
}

func TestBootstrapReadinessGatesAnd(t *testing.T) {
	var gateA, gateB atomic.Bool
	gateA.Store(true)
	gateB.Store(true)
	rt := Bootstrap("atlas-test", WithoutTracer(),
		WithReadinessGate(gateA.Load),
		WithReadinessGate(gateB.Load),
	)
	if !rt.Ready() {
		t.Fatal("all gates true → Ready")
	}
	gateB.Store(false)
	if rt.Ready() {
		t.Fatal("any gate false → not Ready")
	}
}
```

`libs/atlas-service/zz_lifecycle_test.go` — file name starts with `zz_` so it compiles/runs LAST in the package: it closes the singleton teardown manager, after which no other test may use teardown-context APIs.

```go
package service

import (
	"syscall"
	"testing"
	"time"
)

// TestBootstrapLifecycleSIGTERMFlipsReadiness sends a real SIGTERM to the
// test process and drives Manager.Wait() end-to-end: teardown funcs fire,
// the readiness controller flips, Wait returns. MUST run last in the
// package (the zz_ filename enforces source ordering) because the teardown
// manager singleton cannot be re-armed.
func TestBootstrapLifecycleSIGTERMFlipsReadiness(t *testing.T) {
	rt := Bootstrap("atlas-test", WithoutTracer())
	rt.Logger().SetOutput(testWriter{t})

	if !rt.Ready() {
		t.Fatal("must be Ready before SIGTERM")
	}

	done := make(chan struct{})
	go func() {
		rt.Wait()
		close(done)
	}()
	if err := syscall.Kill(syscall.Getpid(), syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("Wait() did not return after SIGTERM")
	}
	if rt.Ready() {
		t.Fatal("Ready() must be false after teardown")
	}
}

type testWriter struct{ t *testing.T }

func (w testWriter) Write(p []byte) (int, error) {
	w.t.Log(string(p))
	return len(p), nil
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `cd libs/atlas-service && go test -race ./... 2>&1 | head -10`
Expected: FAIL — `undefined: Bootstrap`.

- [ ] **Step 4: Write the implementation**

`libs/atlas-service/bootstrap.go`:

```go
package service

import (
	"context"
	"sync"
	"sync/atomic"

	tracing "github.com/Chronicle20/atlas/libs/atlas-tracing"
	"github.com/sirupsen/logrus"
)

type bootstrapConfig struct {
	tracer     bool
	gates      []func() bool
	projection *projectionConfig
}

// Option configures Bootstrap.
type Option func(*bootstrapConfig)

// WithoutTracer skips otel tracer initialization (atlas-renders, tests).
func WithoutTracer() Option {
	return func(c *bootstrapConfig) { c.tracer = false }
}

// WithReadinessGate ANDs fn into Runtime.Ready(). Services with richer
// readiness (e.g. projection catch-up state) pass their gate here.
func WithReadinessGate(fn func() bool) Option {
	return func(c *bootstrapConfig) { c.gates = append(c.gates, fn) }
}

// Runtime is the handle Bootstrap returns; main.go composes the rest of
// startup (DB/Redis, consumers, REST server, tasks) around it.
type Runtime struct {
	logger       *logrus.Logger
	tdm          *Manager
	shuttingDown atomic.Bool
	gates        []func() bool
	projection   Projection
}

// Bootstrap owns the fleet-canonical startup sequence: logger, teardown
// manager, tracer (with teardown registered), the readiness controller,
// and — when the option is present — configuration-projection wiring.
// Fatal semantics match the per-service code it replaces (FR-4.5).
func Bootstrap(serviceName string, opts ...Option) *Runtime {
	cfg := &bootstrapConfig{tracer: true}
	for _, o := range opts {
		o(cfg)
	}

	l := CreateLogger(serviceName)
	l.Infoln("Starting main service.")

	tdm := GetTeardownManager()
	rt := &Runtime{logger: l, tdm: tdm, gates: cfg.gates}

	if cfg.tracer {
		tc, err := tracing.InitTracer(serviceName)
		if err != nil {
			l.WithError(err).Fatal("Unable to initialize tracer.")
		}
		tdm.TeardownFunc(tracing.Teardown(l)(tc))
	}

	// Readiness controller: SIGTERM teardown flips /readyz to 503 before
	// downstream teardowns destroy state in-flight handlers might touch.
	// Teardown funcs fire concurrently on doneChan close, so registration
	// order here is not semantically meaningful.
	tdm.TeardownFunc(func() {
		rt.shuttingDown.Store(true)
		l.Info("Flipped /readyz to not-ready for graceful shutdown.")
	})

	if cfg.projection != nil {
		rt.startProjection(cfg.projection)
	}

	return rt
}

func (r *Runtime) Logger() *logrus.Logger     { return r.logger }
func (r *Runtime) Context() context.Context   { return r.tdm.Context() }
func (r *Runtime) WaitGroup() *sync.WaitGroup { return r.tdm.WaitGroup() }
func (r *Runtime) TeardownFunc(f func())      { r.tdm.TeardownFunc(f) }

// TeardownManager exposes the underlying *Manager for callees typed on it
// (e.g. atlas-login's buildListener).
func (r *Runtime) TeardownManager() *Manager { return r.tdm }

// Ready reports readiness for /readyz: not shutting down AND every
// WithReadinessGate fn true.
func (r *Runtime) Ready() bool {
	if r.shuttingDown.Load() {
		return false
	}
	for _, g := range r.gates {
		if !g() {
			return false
		}
	}
	return true
}

// Wait blocks until teardown completes, then logs the canonical
// shutdown line.
func (r *Runtime) Wait() {
	r.tdm.Wait()
	r.logger.Infoln("Service shutdown.")
}
```

To keep this task compiling before Task 5 exists, also create the projection file stub CONTENTS AS PART OF TASK 5 — for THIS task, temporarily declare in `bootstrap.go`:

```go
// projectionConfig and Projection are defined in projection.go (Task 5).
```

…and if executing Task 4 standalone, add a minimal `libs/atlas-service/projection.go` containing only the type declarations Task 5 will flesh out:

```go
package service

import (
	"context"
	"sync"

	"github.com/sirupsen/logrus"
)

// Projection is the two-method surface Bootstrap drives for opt-in
// configuration-projection wiring (design D6). Full wiring in Task 5.
type Projection interface {
	Start(ctx context.Context, l logrus.FieldLogger, wg *sync.WaitGroup, groupId string) error
	WaitCaughtUp(ctx context.Context) error
}

type projectionConfig struct {
	baseGroupId string
	build       ProjectionBuilder
}

// ProjectionBuilder builds the service's Projection from the resolved topics.
type ProjectionBuilder func(t ProjectionTopics) Projection

// ProjectionTopics carries the env-resolved config-status topic names.
type ProjectionTopics struct {
	ServiceStatus string
	TenantStatus  string
}

func (r *Runtime) startProjection(pc *projectionConfig) {
	panic("projection wiring lands in Task 5; no caller passes WithConfigProjection yet")
}
```

(Task 5 replaces the panic with the real wiring in the same file — Tasks 4 and 5 land as consecutive commits, and no service uses the projection option until Task 10, so the panic is unreachable in any landed state. If Tasks 4 and 5 are executed by the same engineer back-to-back, skip the panic body and write Task 5's real body directly.)

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd libs/atlas-service && go test -race ./... && go vet ./...`
Expected: PASS — accessors, gate ANDing, and the zz_ lifecycle test (SIGTERM → Wait returns → Ready false).

- [ ] **Step 6: Commit**

```bash
git add libs/atlas-service/
git commit -m "feat(atlas-service): Bootstrap/Runtime with readiness controller (DUP-3, OPS-2)"
```

---

### Task 5: `libs/atlas-service` — configuration-projection option

**Files:**
- Modify: `libs/atlas-service/projection.go` (replace Task 4's stub body)
- Test: `libs/atlas-service/projection_test.go`
- Modify: `libs/atlas-service/go.mod` (+ google/uuid)

**Interfaces:**
- Consumes: `Runtime`, `bootstrapConfig`, `Option` from Task 4.
- Produces (used by Tasks 10–11):
  - `func WithConfigProjection(baseGroupId string, build ProjectionBuilder) Option`
  - `type ProjectionFuncs struct { StartFunc func(ctx context.Context, l logrus.FieldLogger, wg *sync.WaitGroup, groupId string) error; WaitCaughtUpFunc func(ctx context.Context) error }` implementing `Projection`
  - `func (r *Runtime) AwaitProjectionCatchUp()` — Fatal on timeout; panics if no projection option
  - unexported `parseProjectionCatchupTimeout() time.Duration` (env `PROJECTION_CATCHUP_TIMEOUT_S`, 5-minute default)

- [ ] **Step 1: Write the failing tests**

`libs/atlas-service/projection_test.go`:

```go
package service

import (
	"context"
	"errors"
	"io"
	"regexp"
	"sync"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

type fakeProjection struct {
	mu           sync.Mutex
	startGroupId string
	startCtx     context.Context
	startErr     error
	waitFn       func(ctx context.Context) error
}

func (f *fakeProjection) Start(ctx context.Context, _ logrus.FieldLogger, _ *sync.WaitGroup, groupId string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.startCtx = ctx
	f.startGroupId = groupId
	return f.startErr
}

func (f *fakeProjection) WaitCaughtUp(ctx context.Context) error {
	if f.waitFn != nil {
		return f.waitFn(ctx)
	}
	return nil
}

func TestWithConfigProjectionStartsSubscriberWithGeneratedGroupId(t *testing.T) {
	t.Setenv("EVENT_TOPIC_CONFIGURATION_SERVICE_STATUS", "svc-status")
	t.Setenv("EVENT_TOPIC_CONFIGURATION_TENANT_STATUS", "tenant-status")
	fake := &fakeProjection{}
	var gotTopics ProjectionTopics
	rt := Bootstrap("atlas-test", WithoutTracer(),
		WithConfigProjection("Test Service - abc", func(topics ProjectionTopics) Projection {
			gotTopics = topics
			return fake
		}),
	)
	if gotTopics.ServiceStatus != "svc-status" || gotTopics.TenantStatus != "tenant-status" {
		t.Fatalf("topics not resolved from env: %+v", gotTopics)
	}
	want := regexp.MustCompile(`^Test Service - abc - projection - [0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	if !want.MatchString(fake.startGroupId) {
		t.Fatalf("groupId %q does not match per-process pattern", fake.startGroupId)
	}
	if fake.startCtx == nil {
		t.Fatal("Start not bound to teardown context")
	}
	rt.AwaitProjectionCatchUp() // fake catches up immediately; must return
}

func TestAwaitProjectionCatchUpTimeoutFatal(t *testing.T) {
	t.Setenv("EVENT_TOPIC_CONFIGURATION_TENANT_STATUS", "tenant-status")
	t.Setenv("PROJECTION_CATCHUP_TIMEOUT_S", "1")
	fake := &fakeProjection{waitFn: func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	}}
	rt := Bootstrap("atlas-test", WithoutTracer(),
		WithConfigProjection("Test Service", func(ProjectionTopics) Projection { return fake }),
	)
	rt.Logger().SetOutput(io.Discard)
	exited := false
	rt.Logger().ExitFunc = func(int) { exited = true; panic("exit") }
	func() {
		defer func() { _ = recover() }()
		rt.AwaitProjectionCatchUp()
	}()
	if !exited {
		t.Fatal("catch-up timeout must Fatal (process exit)")
	}
}

func TestAwaitProjectionCatchUpWithoutOptionPanics(t *testing.T) {
	rt := Bootstrap("atlas-test", WithoutTracer())
	defer func() {
		if recover() == nil {
			t.Fatal("AwaitProjectionCatchUp without WithConfigProjection must panic")
		}
	}()
	rt.AwaitProjectionCatchUp()
}

func TestProjectionFuncsAdapts(t *testing.T) {
	var startedGroup string
	waitErr := errors.New("nope")
	p := ProjectionFuncs{
		StartFunc: func(_ context.Context, _ logrus.FieldLogger, _ *sync.WaitGroup, g string) error {
			startedGroup = g
			return nil
		},
		WaitCaughtUpFunc: func(context.Context) error { return waitErr },
	}
	if err := p.Start(context.Background(), nil, nil, "g1"); err != nil || startedGroup != "g1" {
		t.Fatalf("Start delegation broken: %v %q", err, startedGroup)
	}
	if !errors.Is(p.WaitCaughtUp(context.Background()), waitErr) {
		t.Fatal("WaitCaughtUp delegation broken")
	}
}

func TestParseProjectionCatchupTimeout(t *testing.T) {
	tests := []struct {
		val  string
		want time.Duration
	}{
		{"", 5 * time.Minute},
		{"30", 30 * time.Second},
		{"0", 5 * time.Minute},
		{"-4", 5 * time.Minute},
		{"garbage", 5 * time.Minute},
	}
	for _, tc := range tests {
		t.Setenv("PROJECTION_CATCHUP_TIMEOUT_S", tc.val)
		if got := parseProjectionCatchupTimeout(); got != tc.want {
			t.Errorf("PROJECTION_CATCHUP_TIMEOUT_S=%q → %v, want %v", tc.val, got, tc.want)
		}
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd libs/atlas-service && go test -race ./... 2>&1 | head -10`
Expected: FAIL — `undefined: WithConfigProjection` / `undefined: ProjectionFuncs` (or the Task 4 stub panic).

- [ ] **Step 3: Write the implementation**

Replace `libs/atlas-service/projection.go` in full:

```go
package service

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// Projection is the two-method surface Bootstrap drives for opt-in
// configuration-projection wiring (design D6). Each projection service's
// existing Subscriber/CaughtUp pair satisfies it via ProjectionFuncs.
type Projection interface {
	Start(ctx context.Context, l logrus.FieldLogger, wg *sync.WaitGroup, groupId string) error
	WaitCaughtUp(ctx context.Context) error
}

// ProjectionTopics carries the env-resolved config-status topic names.
type ProjectionTopics struct {
	ServiceStatus string
	TenantStatus  string
}

// ProjectionBuilder builds the service's Projection from the resolved topics.
type ProjectionBuilder func(t ProjectionTopics) Projection

// ProjectionFuncs adapts a service's Subscriber.Start / CaughtUp.WaitCaughtUp
// pair to the Projection interface without a per-service adapter type.
type ProjectionFuncs struct {
	StartFunc        func(ctx context.Context, l logrus.FieldLogger, wg *sync.WaitGroup, groupId string) error
	WaitCaughtUpFunc func(ctx context.Context) error
}

func (p ProjectionFuncs) Start(ctx context.Context, l logrus.FieldLogger, wg *sync.WaitGroup, groupId string) error {
	return p.StartFunc(ctx, l, wg, groupId)
}

func (p ProjectionFuncs) WaitCaughtUp(ctx context.Context) error {
	return p.WaitCaughtUpFunc(ctx)
}

type projectionConfig struct {
	baseGroupId string
	build       ProjectionBuilder
}

// WithConfigProjection makes Bootstrap read the config-status topic env
// vars, build the service's Projection, and start it bound to the teardown
// context/waitgroup under a per-process consumer group id (replaying the
// compacted log from FirstOffset on every container start). The catch-up
// gate stays an explicit rt.AwaitProjectionCatchUp() call so each service
// keeps its own gate position (world/character-factory gate after the REST
// server starts; login/channel gate before building listeners).
func WithConfigProjection(baseGroupId string, build ProjectionBuilder) Option {
	return func(c *bootstrapConfig) {
		c.projection = &projectionConfig{baseGroupId: baseGroupId, build: build}
	}
}

func (r *Runtime) startProjection(pc *projectionConfig) {
	topics := ProjectionTopics{
		ServiceStatus: os.Getenv("EVENT_TOPIC_CONFIGURATION_SERVICE_STATUS"),
		TenantStatus:  os.Getenv("EVENT_TOPIC_CONFIGURATION_TENANT_STATUS"),
	}
	if topics.TenantStatus == "" {
		r.logger.Warn("projection: EVENT_TOPIC_CONFIGURATION_TENANT_STATUS is not set; tenant config updates will not propagate live")
	}
	p := pc.build(topics)
	// Per-process group id so each container start replays the full
	// compacted log from FirstOffset; a shared group would resume from the
	// previous run's committed offset and leave the in-memory State empty.
	groupId := fmt.Sprintf("%s - projection - %s", pc.baseGroupId, uuid.New().String())
	if err := p.Start(r.tdm.Context(), r.logger, r.tdm.WaitGroup(), groupId); err != nil {
		r.logger.WithError(err).Fatal("Unable to start configuration projection subscriber.")
	}
	r.projection = p
}

// AwaitProjectionCatchUp blocks until the projection reports caught-up or
// the PROJECTION_CATCHUP_TIMEOUT_S window (default 5 minutes — covers
// fresh PR envs where atlas-pr-bootstrap is still writing initial configs)
// elapses, in which case it Fatals. Panics if Bootstrap was not given
// WithConfigProjection (programmer error, not a silent no-op).
func (r *Runtime) AwaitProjectionCatchUp() {
	if r.projection == nil {
		panic("service.Runtime.AwaitProjectionCatchUp called without WithConfigProjection")
	}
	ctx, cancel := context.WithTimeout(r.tdm.Context(), parseProjectionCatchupTimeout())
	defer cancel()
	if err := r.projection.WaitCaughtUp(ctx); err != nil {
		r.logger.WithError(err).Fatal("Configuration projection failed to catch up.")
	}
}

// parseProjectionCatchupTimeout reads PROJECTION_CATCHUP_TIMEOUT_S from env
// (positive integer seconds); default 5 minutes. Invalid values silently
// keep the default, matching the four service copies this replaces.
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

Then: `cd libs/atlas-service && go get github.com/google/uuid@v1.6.0`.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd libs/atlas-service && go test -race ./... && go vet ./...`
Expected: PASS. (The zz_ lifecycle test still runs last and still passes.)

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-service/
git commit -m "feat(atlas-service): opt-in configuration-projection wiring (FR-4.2)"
```

---

### Task 6: Pilot migration — atlas-fame

**Files:**
- Modify: `services/atlas-fame/atlas.com/fame/main.go`
- Delete: `services/atlas-fame/atlas.com/fame/logger/init.go` (whole `logger/` dir)
- Delete: `services/atlas-fame/atlas.com/fame/kafka/producer/producer.go`
- Modify: every file matched by `grep -rl '"atlas-fame/kafka/producer"' services/atlas-fame`

**Interfaces:**
- Consumes: `service.Bootstrap`, `Runtime` (Task 4), lib `producer.Provider`/`ProviderImpl` (Task 1), `server.MountReadiness` (existing).
- Produces: the validated worked example every cohort commit follows.

- [ ] **Step 1: Apply Recipe R1 (producer sweep) to atlas-fame**

Follow Recipe R1 with `<svc>` = `fame`. Expected: `kafka/producer/producer.go` deleted; domain producer files' imports rewritten; `go build ./...` clean.

- [ ] **Step 2: Apply Recipe R2 (main.go rewrite)**

The full target `services/atlas-fame/atlas.com/fame/main.go` (assuming the pre-rebase shape; re-derive retained lines from the post-rebase file per Task 0):

```go
package main

import (
	"atlas-fame/fame"
	"atlas-fame/kafka/consumer/character"
	fame2 "atlas-fame/kafka/consumer/fame"
	"os"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	consumergroup "github.com/Chronicle20/atlas/libs/atlas-kafka/consumergroup"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/Chronicle20/atlas/libs/atlas-service"
)

const serviceName = "atlas-fame"

var consumerGroupId = consumergroup.Resolve("Fame Service")

func main() {
	rt := service.Bootstrap(serviceName)
	l := rt.Logger()

	db := database.Connect(l, database.SetMigrations(fame.Migration))

	cmf := consumer.GetManager().AddConsumer(l, rt.Context(), rt.WaitGroup())
	fame2.InitConsumers(l)(cmf)(consumerGroupId)
	if err := fame2.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}
	character.InitConsumers(l)(cmf)(consumerGroupId)
	if err := character.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}

	rt.TeardownFunc(func() { _ = producer.GetManager().Close(l) })

	server.New(l).
		WithContext(rt.Context()).
		WithWaitGroup(rt.WaitGroup()).
		SetBasePath("/api/").
		SetPort(os.Getenv("REST_PORT")).
		AddRouteInitializer(server.MountHandler("/debug/consumers", consumer.GetManager().DebugHandler())).
		AddRouteInitializer(server.MountReadiness("/readyz", rt.Ready)).
		Run()

	rt.Wait()
}
```

Then `git rm -r services/atlas-fame/atlas.com/fame/logger`.

- [ ] **Step 3: Verify the migrated main.go against the checklist**

Run from the worktree root:
```bash
grep -c "tracing.InitTracer\|logger.CreateLogger\|GetTeardownManager\|Infoln(\"Service shutdown" services/atlas-fame/atlas.com/fame/main.go || true
grep -c "MountReadiness" services/atlas-fame/atlas.com/fame/main.go
```
Expected: first grep 0 matches (exit 1), second grep prints 1.

- [ ] **Step 4: Apply Recipe R3 (verify + commit)**

Commit message: `refactor(fame): migrate to service.Bootstrap`.

- [ ] **Step 5: Bake the pilot image**

Run: `docker buildx bake atlas-fame`
Expected: clean build. This validates the Dockerfile COPY set against the new lib deps ONCE before the sweep; the full-fleet bake runs in Task 14.

---

### Task 7: Cohort A sweep — 44 standard services

**Files:** per service `atlas-<svc>`: `main.go`, `logger/` (delete), `kafka/producer/producer.go` (delete), import-site rewrites.

**Interfaces:**
- Consumes: Recipes R1–R3 and the Task 6 worked example. Every service in this cohort has the atlas-fame shape plus zero or more of: `database.Connect`, `atlas.Connect` (Redis), `tasks.Register` goroutines, extra teardowns — all retained verbatim, only the R2 substitutions apply.

Apply R1 → R2 → R3 to each service, one commit per service, in this order:

- [ ] atlas-account
- [ ] atlas-asset-expiration
- [ ] atlas-ban
- [ ] atlas-buddies
- [ ] atlas-buffs
- [ ] atlas-cashshop — ALSO rewrite `atlas.com/cashshop/cashshop/inventory/rest_test.go`: replace the `"atlas-cashshop/logger"` import with `"github.com/Chronicle20/atlas/libs/atlas-service"` and the `logger.CreateLogger(...)` call with `service.CreateLogger(...)` (same signature).
- [ ] atlas-chairs
- [ ] atlas-chalkboards
- [ ] atlas-character
- [ ] atlas-consumables
- [ ] atlas-data
- [ ] atlas-doors
- [ ] atlas-drops
- [ ] atlas-effective-stats
- [ ] atlas-expressions
- [ ] atlas-families
- [ ] atlas-guilds
- [ ] atlas-inventory
- [ ] atlas-invites
- [ ] atlas-keys
- [ ] atlas-map-actions — its `logger/init.go` is one of the 3 drifted-but-semantically-identical variants; deletion is identical.
- [ ] atlas-maps
- [ ] atlas-marriages — has `kafka/producer/producer_test.go`; keep it, rewrite its import per R1.
- [ ] atlas-messages
- [ ] atlas-messengers
- [ ] atlas-monster-book — local logger file is `logger/logger.go` (NOT `init.go`); delete the `logger/` dir all the same.
- [ ] atlas-monster-death
- [ ] atlas-monsters
- [ ] atlas-mounts
- [ ] atlas-notes
- [ ] atlas-npc-conversations
- [ ] atlas-npc-shops
- [ ] atlas-parties
- [ ] atlas-party-quests
- [ ] atlas-pets
- [ ] atlas-portal-actions — drifted logger variant; same deletion.
- [ ] atlas-reactor-actions — drifted logger variant; same deletion.
- [ ] atlas-reactors — has `kafka/producer/producer_test.go`; keep it, rewrite its import per R1.
- [ ] atlas-portals
- [ ] atlas-skills
- [ ] atlas-storage — its `atlas-storage/projection` package is a storage-domain projection, NOT config projection; leave untouched. Standard R1/R2 apply.
- [ ] atlas-summons
- [ ] atlas-tenants
- [ ] atlas-transports

Per-service steps (every checkbox above):

- [ ] **Step 1:** Recipe R1 (producer sweep).
- [ ] **Step 2:** Recipe R2 (main.go rewrite + logger deletion). Retain all service-specific lines (Redis `atlas.Connect`, `database.Connect`, `tasks.Register`, extra `TeardownFunc`s, route initializers) verbatim with only the `tdm.` → `rt.` substitutions.
- [ ] **Step 3:** Recipe R3 (build/vet/test + one commit `refactor(<svc>): migrate to service.Bootstrap`).

---

### Task 8: Cohort B — 5 services without a producer wrapper

**Files:** per service: `main.go`, `logger/` (delete). No `kafka/producer/producer.go` exists; skip Recipe R1 entirely.

**Interfaces:** Consumes Recipes R2–R3 only.

- [ ] atlas-configurations
- [ ] atlas-drop-information
- [ ] atlas-gachapons
- [ ] atlas-query-aggregator
- [ ] atlas-rates

Per-service steps:

- [ ] **Step 1:** Recipe R2 (main.go rewrite + logger deletion). These services may lack the `producer.GetManager().Close` teardown line — do NOT add one; only transform what exists.
- [ ] **Step 2:** Recipe R3 (verify + commit).

---

### Task 9: Cohort C — special-shape services

**Interfaces:** Consumes Recipes R1–R3 plus the per-service notes below.

- [ ] **atlas-quest**

`kafka/producer/producer.go` defines ONLY `type Provider func(token string) producer.MessageProducer` (no `ProviderImpl`). Recipe R1's sed makes every reference resolve to the lib's identical `Provider`; delete the wrapper. Recipe R2/R3 as normal — quest's `main.go` never wired `ProviderImpl` and must not gain it (FR-1.4).

- [ ] **atlas-merchant**

Extra step before R2: delete the private teardown-manager copy.
```bash
ls services/atlas-merchant/atlas.com/merchant/service/
grep -rn '"atlas-merchant/service"' services/atlas-merchant --include='*.go'
```
If `service/` contains only `teardown.go` (byte-equivalent to `libs/atlas-service/teardown.go`): `git rm -r services/atlas-merchant/atlas.com/merchant/service` and rewrite every `"atlas-merchant/service"` import to `"github.com/Chronicle20/atlas/libs/atlas-service"` (package name is `service` in both — call sites compile unchanged). If other files exist in the dir, delete only `teardown.go` and rewrite only the teardown-manager references. Then R1/R2/R3 as normal.

- [ ] **atlas-saga-orchestrator**

The dual-import-heavy service: files like `atlas.com/saga-orchestrator/saga/producer.go` import BOTH the local wrapper (`"atlas-saga-orchestrator/kafka/producer"`) and the lib (`kproducer "github.com/Chronicle20/atlas/libs/atlas-kafka/producer"`). After R1's sed these become duplicate imports of the lib — for each affected file, keep the single unaliased `producer` import and rename `kproducer.X` → `producer.X`. Then R2/R3 as normal.

---

### Task 10: Cohort D1 — atlas-world + atlas-character-factory (tenant-topic projections)

**Files:**
- Modify: `services/atlas-world/atlas.com/world/main.go`, `services/atlas-character-factory/atlas.com/character-factory/main.go`
- Delete: both services' `logger/` dirs and `kafka/producer/producer.go` wrappers (R1/R2 mechanics).

**Interfaces:**
- Consumes: `service.WithConfigProjection`, `service.WithReadinessGate`, `service.ProjectionFuncs`, `rt.AwaitProjectionCatchUp()` (Task 5); each service's existing `configuration/projection` package (`NewState`, `NewCaughtUp`, `Subscriber{State, CaughtUp, TenantTopic}`, `CaughtUp.CaughtUpNow`, `CaughtUp.WaitCaughtUp`) — unchanged.
- Behavior contract: identical runtime semantics to the hand-rolled blocks; catch-up gate stays AFTER the REST server starts (readyz serves 503 during catch-up); readiness gate = `configuration.SnapshotReady` (world) / `caughtUp.CaughtUpNow` (character-factory).

- [ ] **Step 1: atlas-world — Recipe R1, then rewrite main.go**

The projection-specific region of the new `services/atlas-world/atlas.com/world/main.go` (everything else follows R2; retained lines — Redis, registries, consumers, bridge, sweep, tasks — stay verbatim):

```go
func main() {
	state := projection.NewState()
	caughtUp := projection.NewCaughtUp()

	rt := service.Bootstrap(serviceName,
		service.WithConfigProjection(consumerGroupId, func(t service.ProjectionTopics) service.Projection {
			sub := &projection.Subscriber{State: state, CaughtUp: caughtUp, TenantTopic: t.TenantStatus}
			return service.ProjectionFuncs{StartFunc: sub.Start, WaitCaughtUpFunc: caughtUp.WaitCaughtUp}
		}),
		service.WithReadinessGate(configuration.SnapshotReady),
	)
	l := rt.Logger()

	rc := atlas.Connect(l)
	channel.InitRegistry(rc)
	rate.InitRegistry(rc)

	// ... consumers, producer-close teardown, REST server — R2 substitutions,
	// with the readiness line becoming:
	//   AddRouteInitializer(server.MountReadiness("/readyz", rt.Ready)).

	l.Infof("Service started.")

	rt.AwaitProjectionCatchUp()
	l.Info("Configuration projection caught up.")

	// ... bridge goroutine, boot sweep, tasks — unchanged, with rt.Context().

	rt.Wait()
}
```

Deletions specific to this service: the whole hand-rolled projection block (env reads, warn, `Subscriber` literal, `projectionGroupId`, `sub.Start` + Fatal), the `var shuttingDown atomic.Bool` + `ready := ...` + flip-teardown block, the `ctxCaught` catch-up block, the `parseProjectionCatchupTimeout` function, and the now-unused imports (`strconv`, `sync/atomic`, `fmt`, `uuid` — verify each with the compiler; `context` stays if the boot sweep still uses it).

Accepted micro-changes (design D6, restate in the commit message): subscriber starts inside `Bootstrap()` (slightly earlier than today), and the success log line stays in main.go.

- [ ] **Step 2: atlas-world — line-by-line behavior diff**

Run: `git diff HEAD -- services/atlas-world/atlas.com/world/main.go`
Check against the old file: same topics read, same warn message, same group-id pattern, same Fatal messages, gate still `SnapshotReady && !shuttingDown` (now via `WithReadinessGate` + controller), catch-up still gated after `Run()`, bridge/sweep untouched.

- [ ] **Step 3: atlas-world — Recipe R3**

Commit message: `refactor(world): migrate to service.Bootstrap with projection option`.

- [ ] **Step 4: atlas-character-factory — same transformation**

Identical pattern to Step 1 with two differences, matching its current `main.go`: the readiness gate is `service.WithReadinessGate(caughtUp.CaughtUpNow)`, and its retained body (no Redis, its own consumers/resources) stays verbatim. Its `Subscriber` literal is also `{State: state, CaughtUp: caughtUp, TenantTopic: t.TenantStatus}`. Catch-up gate position: keep exactly where its current `ctxCaught` block sits relative to the REST server (per the current file: after `Run()`, like world).

- [ ] **Step 5: atlas-character-factory — behavior diff + Recipe R3**

Commit message: `refactor(character-factory): migrate to service.Bootstrap with projection option`.

- [ ] **Step 6: Probe-path invariant check (bug_readiness_probe_path_under_api_basepath)**

Run:
```bash
grep -n "readyz" deploy/k8s/base/atlas-world.yaml deploy/k8s/base/atlas-character-factory.yaml
grep -n "SetBasePath" services/atlas-world/atlas.com/world/main.go services/atlas-character-factory/atlas.com/character-factory/main.go
```
Expected: manifests probe `/api/readyz`; both mains still produce that effective path (basePath `/api/` + `MountReadiness("/readyz", ...)`). No manifest edits in this task.

---

### Task 11: Cohort D2 — atlas-login + atlas-channel (dual-topic socket projections)

**Files:**
- Modify: `services/atlas-login/atlas.com/login/main.go`, `services/atlas-channel/atlas.com/channel/main.go`
- Delete: both services' `logger/` dirs and producer wrappers (R1/R2 mechanics).

**Interfaces:**
- Consumes: same lib surface as Task 10, plus `rt.TeardownManager()` for `buildListener` (typed on `*service.Manager`).
- Behavior contract: catch-up gate BEFORE listener/registry construction (unchanged position); readiness gate = `caughtUp.CaughtUpNow`; `Subscriber` keeps `ServiceTopic`, `TenantTopic`, AND `ServiceId`; drain teardown order preserved.

- [ ] **Step 1: atlas-login — Recipe R1, then rewrite the opening of main.go**

Replace the current opening (logger → tdm → tracer → serviceId → maps → cmf → producer teardown → consumer inits → hand-rolled projection block → catch-up block) with:

```go
func main() {
	state := projection.NewState()
	caughtUp := projection.NewCaughtUp()
	serviceId := uuid.MustParse(os.Getenv("SERVICE_ID"))
	var consumerGroupId = consumergroup.Resolve(consumerGroupIdTemplate, serviceId.String())

	rt := service.Bootstrap(serviceName,
		service.WithConfigProjection(consumerGroupId, func(t service.ProjectionTopics) service.Projection {
			sub := &projection.Subscriber{
				State:        state,
				CaughtUp:     caughtUp,
				ServiceTopic: t.ServiceStatus,
				TenantTopic:  t.TenantStatus,
				ServiceId:    serviceId,
			}
			return service.ProjectionFuncs{StartFunc: sub.Start, WaitCaughtUpFunc: caughtUp.WaitCaughtUp}
		}),
		service.WithReadinessGate(caughtUp.CaughtUpNow),
	)
	l := rt.Logger()

	validatorMap := produceValidators()
	handlerMap := produceHandlers()
	writerList := produceWriters()

	cmf := consumer.GetManager().AddConsumer(l, rt.Context(), rt.WaitGroup())

	rt.TeardownFunc(func() { _ = producer.GetManager().Close(l) })

	account2.InitConsumers(l)(cmf)(consumerGroupId)
	session2.InitConsumers(l)(cmf)(consumerGroupId)
	seed.InitConsumers(l)(cmf)(consumerGroupId)

	rt.AwaitProjectionCatchUp()
	l.Info("Configuration projection caught up; starting listener apply loop.")

	// ... publishSnapshot, listenerRegistry, drain teardown, apply loop,
	// ticker, session timeout task, session teardown — retained verbatim
	// with rt.Context()/rt.WaitGroup()/rt.TeardownFunc substitutions.
```

Service-specific deletions beyond R2: the hand-rolled projection block (both env reads + login's both-topics-unset warn — the lib's tenant-topic warn replaces it, a design-accepted micro-change), the `projectionGroupId`/`sub.Start` block, the `ctxCaught` catch-up block, the `var shuttingDown atomic.Bool` + `ready := ...` + flip teardown (the drain-listeners teardown STAYS), `parseProjectionCatchupTimeout()`, and unused imports (`strconv` stays — `parseDrainDeadline` uses it; verify each with the compiler).
Service-specific substitutions: `buildListener(l, tdm, ...)` → `buildListener(l, rt.TeardownManager(), ...)`; the REST readiness line becomes `AddRouteInitializer(restserver.MountReadiness("/readyz", rt.Ready)).`; `socket.CreateSocketService(fl, tctx, tdm.WaitGroup())` inside `buildListener` keeps its `tdm` parameter (that function receives the `*service.Manager` and is unchanged).
`parseDrainDeadline()` and everything below `buildListener` are untouched.

- [ ] **Step 2: atlas-login — behavior diff + Recipe R3**

Run `git diff HEAD -- services/atlas-login/atlas.com/login/main.go` and verify: gate position unchanged (before registry/listeners), `ServiceId` still fed to the Subscriber, drain teardown intact, session teardown intact, Fatal messages identical. Commit: `refactor(login): migrate to service.Bootstrap with projection option`.

- [ ] **Step 3: atlas-channel — same transformation**

atlas-channel's current main.go has the same block anatomy (its `Subscriber` literal spans main.go:234-240; map it field-for-field exactly as login: `State`, `CaughtUp`, `ServiceTopic: t.ServiceStatus`, `TenantTopic: t.TenantStatus`, and its `ServiceId`-equivalent field if present — read the current literal and preserve every field, sourcing topics from `ProjectionTopics`). Gate = `caughtUp.CaughtUpNow`; catch-up call at the current gate position (before its listener registry); channel-specific retained code (field registries, socket writers, tasks) untouched. Commit: `refactor(channel): migrate to service.Bootstrap with projection option`.

- [ ] **Step 4: Probe-path invariant check for login/channel**

```bash
grep -rn "readyz" deploy/k8s/base/atlas-login.yaml deploy/k8s/base/atlas-channel.yaml || echo "no probes configured"
```
Expected: whatever probes exist keep pointing at a path both services still serve. No manifest edits.

---

### Task 12: Cohort E — atlas-renders

**Files:**
- Modify: `services/atlas-renders/atlas.com/renders/main.go`
- Delete: `services/atlas-renders/atlas.com/renders/logger.go`
- Modify: `services/atlas-renders/atlas.com/renders/go.mod` + `go.sum` (the ONE service allowed a go.mod change)

**Interfaces:**
- Consumes: `service.Bootstrap`, `service.WithoutTracer()`, `rt.Ready`, `rt.TeardownFunc`, `rt.Wait`.
- Accepted observable change (design D7, flag in commit + PR): log format moves from plain `logrus.JSONFormatter` to fleet-standard ecslogrus + `service.name` + snake_case.

- [ ] **Step 1: Update go.mod**

Append to `services/atlas-renders/atlas.com/renders/go.mod` (replace paths relative to the module dir, matching other services' pattern):
```
replace github.com/Chronicle20/atlas/libs/atlas-service => ../../../../libs/atlas-service

replace github.com/Chronicle20/atlas/libs/atlas-tracing => ../../../../libs/atlas-tracing
```
Then: `cd services/atlas-renders/atlas.com/renders && go get github.com/Chronicle20/atlas/libs/atlas-service@v0.0.0 || true` and `GOWORK=off go mod tidy`.
Expected: go.mod gains `atlas-service` (direct) and `atlas-tracing` (indirect) requires; go.sum gains ecslogrus/uuid/otel entries.

- [ ] **Step 2: Rewrite main.go**

New `services/atlas-renders/atlas.com/renders/main.go` `main()` and middleware bypass (the rest of the file — `contextFromHeaders`, handlers — unchanged):

```go
const serviceName = "atlas-renders"

func main() {
	rt := service.Bootstrap(serviceName, service.WithoutTracer())
	l := rt.Logger()

	s, err := storage.New(l, storage.ConfigFromEnv())
	if err != nil {
		l.WithError(err).Warn("storage init failed; render handlers will 503")
		s = nil
	}
	r := mux.NewRouter()
	r.Use(tenantMiddleware(l))
	r.HandleFunc("/api/wz/character/render/{tenant}/{region}/{version}/{hash}.png", character.Handler(l, s)).Methods(http.MethodGet)
	r.HandleFunc("/api/wz/map/render/{tenant}/{region}/{version}/{mapId}/{kind}.png", mapr.Handler(l, s)).Methods(http.MethodGet)
	r.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintln(w, "ok")
	})
	r.HandleFunc("/readyz", func(w http.ResponseWriter, _ *http.Request) {
		if rt.Ready() {
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprintln(w, "ready")
			return
		}
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = fmt.Fprintln(w, "not ready")
	})
	port := os.Getenv("REST_PORT")
	if port == "" {
		port = "8080"
	}
	srv := &http.Server{Addr: ":" + port, Handler: r}
	rt.TeardownFunc(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	})
	go func() {
		l.Infof("atlas-renders listening on :%s", port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			l.WithError(err).Fatal("server exited")
		}
	}()
	rt.Wait()
}
```

AND in `tenantMiddleware`, extend the probe bypass:

```go
if r.URL.Path == "/healthz" || r.URL.Path == "/readyz" {
	next.ServeHTTP(w, r)
	return
}
```

Add imports `context`, `errors`, `time`, `"github.com/Chronicle20/atlas/libs/atlas-service"`; delete `logger.go` (`git rm`).
`/healthz` is untouched — the live manifest (`deploy/k8s/base/atlas-renders.yaml`) probes it (FR-5.3). `/readyz` is root-mounted (this service has no `/api` base router) and SIGTERM-aware via the shared controller; graceful shutdown replaces the bare `ListenAndServe` exit.

- [ ] **Step 3: Verify + commit**

```bash
cd services/atlas-renders/atlas.com/renders && go build ./... && go vet ./... && go test -race ./...
cd - && git add -A services/atlas-renders && git commit -m "refactor(renders): migrate to service.Bootstrap (no tracer); add root /readyz + graceful shutdown"
```
Expected: clean. Note the log-format change in the commit body.

---

### Task 13: Documentation

**Files:**
- Modify: `docs/observability.md`
- Modify: `docs/architectural-improvements.md`

- [ ] **Step 1: observability.md — snake_case convention section**

Add a section (after the existing logging/dashboard material):

```markdown
## Log field naming

Structured-log field keys are **snake_case** (`character_id`, `world_id`,
`transaction_id`). This is enforced at emit time: `CreateLogger` in
`libs/atlas-service` registers a normalization hook that rewrites legacy
camelCase keys (`characterId` → `character_id`) on every record, so Loki
queries need only the snake_case spelling. Dotted ECS keys (`service.name`)
pass through unchanged. If a record carries both spellings, the explicit
snake_case key wins and the camelCase duplicate is dropped.

New code should write snake_case keys directly; the hook exists so the
~1,500 legacy call sites don't need a rename sweep and drift cannot
reappear.
```

- [ ] **Step 2: architectural-improvements.md — mark findings resolved**

Locate the DUP-1, DUP-2, DUP-3, and CP-9 findings and mark each resolved (✓, "resolved by task-118") following the file's existing resolved-entry style. Annotate OPS-2: endpoint half done by task-118 (every service serves `/readyz`; effective `/api/readyz` under the standard base path, root `/readyz` on atlas-renders); the probe/manifest half remains with OPS-1. Note under OPS-3 that task-118 deliberately preserved `/api/readyz` semantics (design D5).

- [ ] **Step 3: Commit**

```bash
git add docs/observability.md docs/architectural-improvements.md
git commit -m "docs(task-118): snake_case log convention; mark DUP-1/2/3 + CP-9 resolved"
```

---

### Task 14: Fleet verification + acceptance

**Files:** none created (fixes only if verification fails).

- [ ] **Step 1: Acceptance greps (PRD §10)**

Run from the worktree root — every command's expected output is exactly as annotated:

```bash
find services -path '*/kafka/producer/producer.go' | wc -l                      # 0
find services -path '*/atlas.com/*/logger/init.go' -o -path '*/atlas.com/*/logger/logger.go' | wc -l  # 0
grep -rl parseProjectionCatchupTimeout services | wc -l                          # 0
grep -l "tracing.InitTracer" services/*/atlas.com/*/main.go | wc -l              # 0
grep -rn "logger.CreateLogger" services --include='*.go' | wc -l                 # 0
grep -L "MountReadiness" services/*/atlas.com/*/main.go                          # exactly one line: atlas-renders (root-mounts /readyz by hand)
grep -l "service.Bootstrap" services/*/atlas.com/*/main.go | wc -l               # 58
grep -rn "shuttingDown" services/*/atlas.com/*/main.go | wc -l                   # 0
```

- [ ] **Step 2: Per-module build/vet/test sweep**

```bash
for mod in libs/atlas-kafka libs/atlas-service $(ls -d services/*/atlas.com/*/ | sed 's|/$||'); do
  echo "== $mod"
  (cd "$mod" && go build ./... && go vet ./... && go test -race ./...) || { echo "FAILED: $mod"; break; }
done
```
Expected: every module clean. Fix failures in the owning service's follow-up commit (`fix(<svc>): ...`) before proceeding.

- [ ] **Step 3: Redis key guard + full bake**

```bash
tools/redis-key-guard.sh
docker buildx bake all-go-services
```
Expected: both clean. The bake is mandatory (every service's source is touched); expect it to take a while. A missing-COPY failure is impossible here (no new lib module was created — `libs/atlas-service` and `libs/atlas-kafka` are already in both Dockerfile COPY blocks), but the bake also validates each service's `go.sum` against the libs' new requires; if a service fails hash verification, run `GOWORK=off go mod tidy` in THAT module only and amend its commit.

- [ ] **Step 4: Runtime verification — readiness + snake_case (acceptance criteria)**

Local smoke (candidate: atlas-query-aggregator — logger+REST only):
```bash
cd services/atlas-query-aggregator/atlas.com/query-aggregator
REST_PORT=18080 go run . &   # consumers without brokers retry in background; REST still serves
sleep 3
curl -s -o /dev/null -w "%{http_code}\n" localhost:18080/api/readyz   # expect 200
kill -TERM %1                                                          # SIGTERM
# capture stdout: expect the "Flipped /readyz to not-ready" line, then "Service shutdown."
```
If the service cannot boot far enough locally (hard dependency Fatals first), fall back to the PR-env flow: open the PR, add the `deploy-env` label (per project convention this triggers the ephemeral build/deploy), then against the PR env verify (a) `curl .../api/readyz` → 200 on a previously-readiness-less service, (b) `kubectl delete pod` on it and observe 503/flip log during termination, (c) one live log line from any migrated service shows `character_id`-style keys, and (d) atlas-world/atlas-character-factory pods still pass their existing `/api/readyz` probes (rollout completes).

- [ ] **Step 5: Final acceptance checklist against the PRD**

Walk PRD §10 line by line and check every box is evidenced (greps above, lib tests, runtime checks, docs commits, rebase-gate commit from Task 0). Record the evidence (command + output) in `docs/tasks/task-118-shared-service-bootstrap/context.md` under "Acceptance evidence".

- [ ] **Step 6: Commit any verification artifacts + request code review**

```bash
git add docs/tasks/task-118-shared-service-bootstrap/context.md
git commit -m "chore(task-118): acceptance evidence"
```
Then run `superpowers:requesting-code-review` (mandatory before PR per CLAUDE.md). Do NOT open the PR from this plan — that is the finishing-a-development-branch flow, run after review findings are addressed.
