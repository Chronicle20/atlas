# Kafka Precreate Go Tool — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the sync-wave-0 `atlas-kafka-precreate` bash/JVM-CLI Job with a small Go binary that does the entire topic pass in one `CreateTopics` request and seeds override consumer-group offsets through typed `segmentio/kafka-go` admin calls.

**Architecture:** A new flat Go module at `services/atlas-kafka-precreate/` (module `atlas.com/kafka-precreate`, no `atlas.com/<name>` nesting, no `libs/atlas-*` dependency). `main.go` runs five phases in order under one 240s context: discover (pure env scrape) → create topics (one request) → metadata settle → seed group offsets → verify. All Kafka access goes through a seven-method `kafkaops.AdminClient` interface that `*kafka.Client` satisfies as-is, so every phase is unit-testable with a hand-written stub and no broker.

**Tech Stack:** Go 1.25.5, `github.com/segmentio/kafka-go v0.4.51`, `github.com/sirupsen/logrus v1.10.0`. Nothing else.

**Spec:** [design.md](design.md) (PRD: [prd.md](prd.md), spike evidence: [spike-findings.md](spike-findings.md))

## Global Constraints

- Module path is `atlas.com/kafka-precreate`; module root is `services/atlas-kafka-precreate/`. **All `go build` / `go test` commands in this plan run from that directory.**
- `go 1.25.5` in `go.mod` (matches every other module in the repo; `go.work` is at `go 1.26.0` and that is fine).
- Exactly two direct dependencies: `github.com/segmentio/kafka-go v0.4.51`, `github.com/sirupsen/logrus v1.10.0`. Do not add `libs/atlas-*`, do not add a test framework — table-driven stdlib `testing` only.
- **No `*_testhelpers.go` files.** Stubs live in `_test.go` files, package-local.
- The kafka-go error constant for code 25 is **`kafka.UnknownMemberId`** (lowercase `d`), not `UnknownMemberID`. The design doc writes it the other way; the library spelling wins. Verified at `github.com/segmentio/kafka-go@v0.4.51/error.go:40`.
- Discriminate Kafka errors with `errors.Is` only. **No string matching on any Kafka error or response anywhere in this tool** (PRD acceptance criterion 3).
- Every fatal path must name the offending topic or group and wrap the underlying typed error (`%w`).
- `kafka.Client` has no per-request timeout field; the effective per-request deadline is `Client.Timeout` (halved internally, `client.go:125-139`) or the remaining context deadline, whichever is smaller. `main` sets `Timeout: 60 * time.Second`.
- Preserve existing line endings on every file you modify under `deploy/` and `.github/`.
- Never commit to `main`; the branch is `task-260-kafka-precreate-go-tool`.

---

### Task 1: Module scaffold and the `discover` package

The pure half of the tool: environment scrape, compaction classification, group-list parsing, and the seedable-state allowlist. Zero I/O, zero Kafka types, so it is fully table-testable.

### Files

- `services/atlas-kafka-precreate/go.mod` — **new file**; module `atlas.com/kafka-precreate`, `go 1.25.5`
- `services/atlas-kafka-precreate/go.sum` — **new file**; produced by `go mod tidy`
- `services/atlas-kafka-precreate/internal/discover/discover.go` — **new file**; everything below
- `services/atlas-kafka-precreate/internal/discover/discover_test.go` — **new file**; the tables below
- `go.work` — modify: add `./services/atlas-kafka-precreate` to the `use (...)` block, in sorted position (between `./services/atlas-inventory/atlas.com/inventory` and `./services/atlas-keys/atlas.com/keys`)
- `deploy/k8s/base/kafka-precreate.sh` — **read-only reference**; the behaviour being ported (`precreate_topics` lines 39-115, `state_is_seedable` lines 168-173, `seed_override_offsets` lines 245-253 for the newline split)

Module root for `go build` / `go test`: `services/atlas-kafka-precreate`.

**Interfaces**

- Consumes: nothing.
- Produces:
  ```go
  package discover

  type Topics struct {
      Plain   []string // sorted, de-duplicated
      Compact []string // sorted, de-duplicated
  }

  func (t Topics) Union() []string                  // sorted, Plain ∪ Compact
  func FromEnviron(environ []string) Topics         // environ entries are "KEY=VALUE"
  func Groups(raw string) []string
  func StateIsSeedable(state string) bool
  ```

- [ ] **Step 1: Create the module**

```bash
mkdir -p services/atlas-kafka-precreate/internal/discover
cd services/atlas-kafka-precreate
cat > go.mod <<'EOF'
module atlas.com/kafka-precreate

go 1.25.5

require (
	github.com/segmentio/kafka-go v0.4.51
	github.com/sirupsen/logrus v1.10.0
)
EOF
```

Do not hand-write `go.sum`; `go mod tidy` in Step 6 produces it. Add `./services/atlas-kafka-precreate` to `go.work`'s `use (...)` block in the same edit.

- [ ] **Step 2: Write the failing tests**

`services/atlas-kafka-precreate/internal/discover/discover_test.go`. Three table-driven test functions, plain `testing`, no external assert library. Compare slices with `reflect.DeepEqual` and report with `%q`.

`TestFromEnviron` — each case supplies `environ []string` and expects an exact `Topics`:

| case | environ | expect Plain | expect Compact |
|---|---|---|---|
| `prefix selection` | `COMMAND_TOPIC_CREATE_CHARACTER=cmd-char`, `EVENT_TOPIC_CHARACTER_STATUS=evt-char`, `PATH=/usr/bin`, `BOOTSTRAP_SERVERS=kafka:9092`, `KAFKA_CONSUMER_GROUP=g` | `["cmd-char","evt-char"]` | `[]` |
| `empty value skipped` (FR-1.2) | `COMMAND_TOPIC_A=`, `EVENT_TOPIC_B=evt-b` | `["evt-b"]` | `[]` |
| `duplicates collapsed` (FR-1.3) | `COMMAND_TOPIC_A=shared`, `COMMAND_TOPIC_B=shared`, `EVENT_TOPIC_C=shared` | `["shared"]` | `[]` |
| `config-status vars are compacted` (FR-1.4) | `EVENT_TOPIC_CONFIGURATION_TENANT_STATUS=cfg-tenant`, `EVENT_TOPIC_CONFIGURATION_SERVICE_STATUS=cfg-service`, `EVENT_TOPIC_CONFIGURATION_ENVIRONMENT_STATUS=cfg-env`, `EVENT_TOPIC_OTHER=evt-other` | `["evt-other"]` | `["cfg-env","cfg-service","cfg-tenant"]` |
| `compaction wins on collision` (FR-1.5) | `EVENT_TOPIC_CONFIGURATION_TENANT_STATUS=both`, `COMMAND_TOPIC_X=both` | `[]` | `["both"]` |
| `underscore vs hyphen ordering is byte order` | `COMMAND_TOPIC_A=topic_b`, `COMMAND_TOPIC_B=topic-a`, `COMMAND_TOPIC_C=topicZ` | `["topic-a","topicZ","topic_b"]` | `[]` |
| `no matching vars` | `PATH=/usr/bin` | `[]` | `[]` |
| `value containing =` | `COMMAND_TOPIC_A=a=b` | `["a=b"]` | `[]` |

The ordering case is the point of the port: byte order puts `-` (0x2D) before `Z` (0x5A) before `_` (0x5F). `sort.Strings` gives exactly that, with no locale to pin.

Assert empty results as length-0, not nil-vs-empty: write the expectation as `[]string{}` and compare with a helper that treats nil and empty as equal, or have `FromEnviron` always return non-nil slices. Pick the second — return `[]string{}` from `sortedKeys` when the set is empty — and then `reflect.DeepEqual` works directly.

`TestGroups` — each case supplies `raw string`, expects `[]string`:

| case | raw | expect |
|---|---|---|
| `unset` (NG6, FR-7.2) | `""` | `[]string{}` |
| `whitespace only` | `"\n  \n"` | `[]string{"  "}` — a line of spaces is a group name; only *empty* lines are dropped |
| `single group` | `"Account Service [pr-123]"` | `["Account Service [pr-123]"]` |
| `multi-line` | `"World Service [pr-123]\nChannel Service - 3f8c [pr-123]"` | `["World Service [pr-123]","Channel Service - 3f8c [pr-123]"]` |
| `blank lines dropped` | `"a\n\n\nb\n"` | `["a","b"]` |
| `CRLF trimmed` | `"a\r\nb\r\n"` | `["a","b"]` |
| `spaces and brackets round-trip` (FR-7.3) | `"Channel Service - 7c2f8b1e-0d4a-4a1b-9f3e-2c1d5e6f7a8b [pr-1450]"` | one element, byte-for-byte identical to the input |
| `leading dash` | `"-weird group"` | `["-weird group"]` |
| `order preserved` | `"z\na"` | `["z","a"]` — group order is input order, NOT sorted |

Interior whitespace is never trimmed. Only a trailing `\r` is stripped (the value arrives from a YAML block scalar). `"  "` (two spaces) survives as a group name — that case exists to prove the parser does not `strings.TrimSpace` each line.

`TestStateIsSeedable` — each case a state string and a bool:

| state | expect |
|---|---|
| `""` | true |
| `"Empty"` | true |
| `"Dead"` | true |
| `"Stable"` | false |
| `"PreparingRebalance"` | false |
| `"CompletingRebalance"` | false |
| `"AssigningPartitions"` | false |
| `"SomeFutureKafkaState"` | false |
| `"empty"` | false — the allowlist is case-sensitive, matching the shell `case` statement |

- [ ] **Step 3: Run the tests and confirm they fail**

Run from `services/atlas-kafka-precreate`:

```bash
go test ./internal/discover/ -v
```

Expected: compile failure — `undefined: discover.FromEnviron` (and the other three symbols).

- [ ] **Step 4: Implement `discover.go`**

```go
// Package discover holds the pure half of the precreate tool: the
// environment scrape that decides which topics exist, and the two
// classification rules the seeding pass keys off. Nothing here performs
// I/O or touches a Kafka type, which is what lets the whole of it be
// table-tested without a broker (design §5.2).
package discover

import (
	"sort"
	"strings"
)

const (
	commandPrefix = "COMMAND_TOPIC_"
	eventPrefix   = "EVENT_TOPIC_"
)

// compactVars names the three config-projection variables whose topics must
// carry cleanup.policy=compact. Their consumers replay from first-offset at
// every boot to rebuild tenant/service config state and the outbox never
// re-emits a (topic, key) it already delivered, so under the default DELETE
// cleanup retention empties the topic ~7 days after the last config change
// and every later projection boot has nothing to replay. Events are keyed,
// so compaction retains the latest snapshot per key forever.
var compactVars = map[string]struct{}{
	"EVENT_TOPIC_CONFIGURATION_TENANT_STATUS":      {},
	"EVENT_TOPIC_CONFIGURATION_SERVICE_STATUS":     {},
	"EVENT_TOPIC_CONFIGURATION_ENVIRONMENT_STATUS": {},
}
```

`FromEnviron(environ []string) Topics`:
- split each entry on the **first** `=` only (`strings.Cut`); an entry with no `=` is ignored.
- keep only names with one of the two prefixes; skip empty values (FR-1.2).
- accumulate into `plain` and `compact` `map[string]struct{}` (FR-1.3).
- delete every `compact` key from `plain` (FR-1.5) — this is the whole of what the shell needed `sort -u | comm -23` and an `LC_ALL=C` pin for.
- return both as sorted `[]string`, never nil.

`Union()` merges both sets and returns a sorted, de-duplicated `[]string`, never nil.

`Groups(raw string) []string`: `strings.Split(raw, "\n")`, `strings.TrimSuffix(line, "\r")`, drop `""`, preserve order, return non-nil.

`StateIsSeedable(state string) bool`: `switch state { case "Empty", "Dead", "": return true }; return false`. Doc-comment it as a deliberate allowlist: a state a future Kafka version introduces falls into "active" and is skipped, and skipping never mutates a committed offset (FR-3.4, NFR-6).

- [ ] **Step 5: Run the tests and confirm they pass**

```bash
go test ./internal/discover/ -v
```

Expected: PASS, all three test functions.

- [ ] **Step 6: Tidy and build**

```bash
go mod tidy
go build ./...
go vet ./...
```

`go mod tidy` writes `go.sum`. `go build ./...` must succeed with only the `discover` package present.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-kafka-precreate go.work
git commit -m "feat(kafka-precreate): module scaffold and pure discover package"
```

---

### Task 2: The `kafkaops` seam and the coordinator retry

The narrow Kafka interface every other package is written against, plus the bounded retry that Q4 of the spike proved is a production requirement, not a test artifact.

### Files

- `services/atlas-kafka-precreate/internal/kafkaops/ops.go` — **new file**
- `services/atlas-kafka-precreate/internal/kafkaops/ops_test.go` — **new file**
- `services/atlas-kafka-precreate/go.mod` — read-only; `kafka-go` is already required

Module root for `go build` / `go test`: `services/atlas-kafka-precreate`.

**Interfaces**

- Consumes: nothing from Task 1.
- Produces:
  ```go
  package kafkaops

  type AdminClient interface {
      CreateTopics(context.Context, *kafka.CreateTopicsRequest) (*kafka.CreateTopicsResponse, error)
      IncrementalAlterConfigs(context.Context, *kafka.IncrementalAlterConfigsRequest) (*kafka.IncrementalAlterConfigsResponse, error)
      Metadata(context.Context, *kafka.MetadataRequest) (*kafka.MetadataResponse, error)
      ListOffsets(context.Context, *kafka.ListOffsetsRequest) (*kafka.ListOffsetsResponse, error)
      DescribeGroups(context.Context, *kafka.DescribeGroupsRequest) (*kafka.DescribeGroupsResponse, error)
      OffsetCommit(context.Context, *kafka.OffsetCommitRequest) (*kafka.OffsetCommitResponse, error)
      OffsetFetch(context.Context, *kafka.OffsetFetchRequest) (*kafka.OffsetFetchResponse, error)
  }

  type RetryConfig struct {
      Base   time.Duration       // first backoff; default 250ms
      Max    time.Duration       // per-attempt cap; default 2s
      Budget time.Duration       // total wall-clock budget; default 60s
      Sleep  func(time.Duration) // nil ⇒ time.Sleep
      Now    func() time.Time    // nil ⇒ time.Now
  }

  func DefaultRetryConfig() RetryConfig
  func WithCoordinatorRetry(ctx context.Context, cfg RetryConfig, fn func() error) error
  ```

  `*kafka.Client` satisfies `AdminClient` with no adapter. Add a compile-time
  assertion in `ops.go`: `var _ AdminClient = (*kafka.Client)(nil)`.

- [ ] **Step 1: Write the failing test**

`services/atlas-kafka-precreate/internal/kafkaops/ops_test.go`.

The clock seam: a `fakeClock` struct holding `now time.Time`; its `Sleep(d)` advances `now` by `d` and appends `d` to a `slept []time.Duration`; its `Now()` returns `now`. Build the config as `RetryConfig{Base: 250 * time.Millisecond, Max: 2 * time.Second, Budget: 60 * time.Second, Sleep: c.Sleep, Now: c.Now}`. No test sleeps for real.

`TestWithCoordinatorRetry` — table-driven, each case a sequence of errors `fn` returns on successive calls:

| case | fn returns, in order | expect err | expect calls | expect backoffs |
|---|---|---|---|---|
| `succeeds first try` | `[nil]` | nil | 1 | `[]` |
| `retries NotCoordinatorForGroup` | `[kafka.NotCoordinatorForGroup, nil]` | nil | 2 | `[250ms]` |
| `retries GroupCoordinatorNotAvailable` | `[kafka.GroupCoordinatorNotAvailable, nil]` | nil | 2 | `[250ms]` |
| `backoff doubles and caps` | `[NotCoordinatorForGroup ×6, nil]` | nil | 7 | `[250ms, 500ms, 1s, 2s, 2s, 2s]` |
| `does not retry an unrelated error` | `[kafka.UnknownMemberId]` | `errors.Is(err, kafka.UnknownMemberId)` is true | 1 | `[]` |
| `does not retry a transport error` | `[errors.New("dial tcp: connection refused")]` | err is that exact error | 1 | `[]` |
| `gives up at the budget` | always `kafka.NotCoordinatorForGroup` | non-nil, and `errors.Is(err, kafka.NotCoordinatorForGroup)` is true | ≥2 | last backoff must not push elapsed past 60s |

For the budget case assert two things: the call eventually returns (the fake clock guarantees termination), and `c.now.Sub(start) <= 60*time.Second`. Also assert the returned error still unwraps to the coordinator error so the operator sees *why* it gave up (NFR-8).

Add `TestWithCoordinatorRetry_ContextCancelled`: a context already cancelled with `context.WithCancel` + immediate `cancel()`, `fn` always returning `kafka.NotCoordinatorForGroup`. Expect `errors.Is(err, context.Canceled)` and `fn` called at most once.

`TestKafkaClientSatisfiesAdminClient` is unnecessary — the `var _ AdminClient = (*kafka.Client)(nil)` assertion in `ops.go` is the compile-time check.

- [ ] **Step 2: Run the test and confirm it fails**

```bash
go test ./internal/kafkaops/ -v
```

Expected: compile failure — `undefined: kafkaops.WithCoordinatorRetry`.

- [ ] **Step 3: Implement `ops.go`**

`isCoordinatorError(err error) bool` returns `errors.Is(err, kafka.NotCoordinatorForGroup) || errors.Is(err, kafka.GroupCoordinatorNotAvailable)`.

`WithCoordinatorRetry` loop:
1. resolve `Sleep`/`Now` defaults to `time.Sleep`/`time.Now`; resolve zero `Base`/`Max`/`Budget` to the `DefaultRetryConfig` values.
2. record `start := now()`; `backoff := cfg.Base`.
3. each iteration: check `ctx.Err()` first and return it if non-nil; call `fn()`; return nil on success; return err immediately unless `isCoordinatorError(err)`.
4. if `now().Sub(start)+backoff > cfg.Budget`, return the error wrapped with the exhausted budget (`fmt.Errorf("coordinator retry budget of %s exhausted: %w", cfg.Budget, err)`).
5. `sleep(backoff)`; `backoff = min(backoff*2, cfg.Max)`.

Doc-comment the scope: this wraps **only** the three group-coordinator calls (`DescribeGroups`, `OffsetCommit`, `OffsetFetch`). `CreateTopics`, `Metadata`, and `IncrementalAlterConfigs` route to the controller and cannot produce these two codes; retrying anything else would turn a diagnosable failure into a timeout (design §4 OQ-3).

- [ ] **Step 4: Run the test and confirm it passes**

```bash
go test ./internal/kafkaops/ -v
go build ./... && go vet ./...
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-kafka-precreate/internal/kafkaops
git commit -m "feat(kafka-precreate): AdminClient seam and bounded coordinator retry"
```

---

### Task 3: The `topics` package — create, settle, end offsets

Everything that talks to the cluster controller: one `CreateTopics` for the whole union, one `IncrementalAlterConfigs` for the compacted set, the metadata settle loop that makes offset routing safe, and the single `ListOffsets` that both groups phases consume.

### Files

- `services/atlas-kafka-precreate/internal/topics/topics.go` — **new file**
- `services/atlas-kafka-precreate/internal/topics/topics_test.go` — **new file**
- `services/atlas-kafka-precreate/internal/discover/discover.go` — read-only; supplies `discover.Topics`
- `services/atlas-kafka-precreate/internal/kafkaops/ops.go` — read-only; supplies `AdminClient`

Module root for `go build` / `go test`: `services/atlas-kafka-precreate`.

**Interfaces**

- Consumes: `discover.Topics` and `discover.Topics.Union()` (Task 1); `kafkaops.AdminClient` (Task 2).
- Produces:
  ```go
  package topics

  type EnsureResult struct {
      Created  int
      Existing int
  }

  type SettleConfig struct {
      Poll     time.Duration       // default 250ms
      Ceiling  time.Duration       // default 30s
      Sleep    func(time.Duration) // nil ⇒ time.Sleep
      Now      func() time.Time    // nil ⇒ time.Now
  }

  func Ensure(ctx context.Context, c kafkaops.AdminClient, addr net.Addr, t discover.Topics) (EnsureResult, error)
  func Settle(ctx context.Context, c kafkaops.AdminClient, addr net.Addr, names []string, cfg SettleConfig) (map[string][]int, error)
  func EndOffsets(ctx context.Context, c kafkaops.AdminClient, addr net.Addr, partitions map[string][]int) (map[string]map[int]int64, error)
  ```

  `Settle` returns topic → sorted partition IDs. `EndOffsets` returns topic → partition → end-of-log offset.

- [ ] **Step 1: Write the failing tests**

`services/atlas-kafka-precreate/internal/topics/topics_test.go`.

The stub, declared once at the top of the file (package-local, not a `_testhelpers.go`):

```go
type stubClient struct {
	createCalls []*kafka.CreateTopicsRequest
	createFn    func(*kafka.CreateTopicsRequest) (*kafka.CreateTopicsResponse, error)

	alterCalls []*kafka.IncrementalAlterConfigsRequest
	alterFn    func(*kafka.IncrementalAlterConfigsRequest) (*kafka.IncrementalAlterConfigsResponse, error)

	metadataCalls []*kafka.MetadataRequest
	metadataFn    func(*kafka.MetadataRequest) (*kafka.MetadataResponse, error)

	listCalls []*kafka.ListOffsetsRequest
	listFn    func(*kafka.ListOffsetsRequest) (*kafka.ListOffsetsResponse, error)
}
```

Each of the seven `AdminClient` methods is implemented on `*stubClient`; the four above record the request and delegate to their `*Fn` field (returning a zero-value response when the field is nil); the three group methods (`DescribeGroups`, `OffsetCommit`, `OffsetFetch`) `panic("unexpected call")` — `topics` must never issue them, and a panic proves it.

`TestEnsure_SingleCreateRequest` (acceptance criterion 2): build `discover.Topics{Plain: 170 generated names "plain-000".."plain-169", Compact: ["cfg-env","cfg-service","cfg-tenant"]}`, stub returns `Errors` with `nil` for all 173. Assert:
- `len(stub.createCalls) == 1`
- that one request carries exactly 173 `TopicConfig` entries
- every entry has `NumPartitions == 1` and `ReplicationFactor == 1`
- the three compact entries carry exactly `[]kafka.ConfigEntry{{ConfigName: "cleanup.policy", ConfigValue: "compact"}}`
- every plain entry has `len(ConfigEntries) == 0`
- result is `EnsureResult{Created: 173, Existing: 0}`

`TestEnsure_CreateErrors` — table-driven over the stub's `Errors` map, with `Topics{Plain: ["a","b"], Compact: ["c"]}`:

| case | Errors map | expect |
|---|---|---|
| `all created` | `{"a": nil, "b": nil, "c": nil}` | no error; `EnsureResult{Created: 3, Existing: 0}` |
| `all already exist` (FR-7.4) | all three ⇒ `kafka.TopicAlreadyExists` | **no error**; `EnsureResult{Created: 0, Existing: 3}` |
| `mixed` | `{"a": nil, "b": kafka.TopicAlreadyExists, "c": nil}` | no error; `EnsureResult{Created: 2, Existing: 1}` |
| `one fatal` | `{"a": nil, "b": kafka.InvalidTopic, "c": nil}` | error; message contains `"b"`; `errors.Is(err, kafka.InvalidTopic)` is true |
| `two fatal, both named` | `{"a": kafka.InvalidTopic, "b": nil, "c": kafka.InvalidPartitionNumber}` | error naming **both** `"a"` and `"c"` — collect-then-fail, not fail-fast |
| `transport error` | `createFn` returns `(nil, errors.New("dial tcp: connection refused"))` | that error, wrapped; no alter request issued |

`kafka.InvalidTopic` (17) and `kafka.InvalidPartitionNumber` (37) are real `kafka.Error` constants — verified at `kafka-go@v0.4.51/error.go:32` and `:52`.

For the two-fatal case, assert with `errors.Join` semantics or a single formatted message; whichever the implementation picks, the test asserts both topic names appear in `err.Error()`.

`TestEnsure_AlterConfigs`: `Topics{Plain: ["p1"], Compact: ["c1","c2"]}`, all creates `nil`. Assert:
- `len(stub.alterCalls) == 1`
- that request's `Resources` has exactly 2 entries, one per compacted topic, sorted by `ResourceName` (`"c1"`, `"c2"`)
- each has `ResourceType == kafka.ResourceTypeTopic`
- each has exactly one `Config`: `{Name: "cleanup.policy", Value: "compact", ConfigOperation: kafka.ConfigOperationSet}`
- `"p1"` appears in **no** alter resource

`TestEnsure_AlterConfigs_NoCompactTopics`: `Topics{Plain: ["p1"], Compact: []}`. Assert `len(stub.alterCalls) == 0` — no empty request is issued.

`TestEnsure_AlterConfigs_ResourceError`: alter response returns `Resources: [{ResourceName: "c1", Error: nil}, {ResourceName: "c2", Error: kafka.PolicyViolation}]`. Expect an error naming `"c2"`. Use whatever non-nil `kafka.Error` constant compiles.

`TestSettle` — fake clock as in Task 2 (`Sleep` advances `Now`), `SettleConfig{Poll: 250 * time.Millisecond, Ceiling: 30 * time.Second, Sleep: c.Sleep, Now: c.Now}`:

| case | metadata responses, in order | expect |
|---|---|---|
| `present on first poll` | one response with topics `a` (partitions 0) and `b` (partitions 0,1) | `map[string][]int{"a": {0}, "b": {0,1}}`; exactly 1 metadata call; no sleep |
| `appears on third poll` | resp1: only `a`; resp2: only `a`; resp3: `a` and `b` | success; 3 metadata calls; backoffs `[250ms, 250ms]` |
| `topic present but zero partitions` | resp1: `a` with `Partitions: []`; resp2: `a` with partitions 0 | success on the second poll |
| `topic carries a metadata error` | `a` with `Error: kafka.UnknownTopicOrPartition`, then `a` clean | success on the second poll — a transient topic-level metadata error is a not-yet-present signal, not a fatal |
| `never appears` | every response returns only `a`, never `b` | error naming `"b"` (and not `"a"`); elapsed ≤ 30s |
| `transport error` | `metadataFn` returns `(nil, errors.New("boom"))` every time | error; retried until the ceiling, then returns naming the transport failure |
| `empty names` | — | `map[string][]int{}`, **zero** metadata calls |

Partition IDs in the returned map must be sorted ascending.

`TestEndOffsets`:

| case | partitions in | stub ListOffsets response | expect |
|---|---|---|---|
| `two topics` | `{"a": {0}, "b": {0,1}}` | `Topics: {"a": [{Partition: 0, LastOffset: 5}], "b": [{Partition: 0, LastOffset: 0}, {Partition: 1, LastOffset: 42}]}` | `{"a": {0: 5}, "b": {0: 0, 1: 42}}`; exactly 1 ListOffsets call; the request's `Topics` map carries `kafka.LastOffsetOf(p)` for every pair (assert `Timestamp == kafka.LastOffset`) |
| `partition error` | `{"a": {0}}` | `Topics: {"a": [{Partition: 0, Error: kafka.LeaderNotAvailable}]}` | error naming topic `a` and partition `0` |
| `missing partition in response` | `{"a": {0,1}}` | response only carries partition 0 | error naming topic `a` and partition `1` |
| `transport error` | `{"a": {0}}` | `(nil, errors.New("boom"))` | that error, wrapped |
| `empty input` | `{}` | — | empty map, **zero** ListOffsets calls |

- [ ] **Step 2: Run the tests and confirm they fail**

```bash
go test ./internal/topics/ -v
```

Expected: compile failure — `undefined: topics.Ensure` etc.

- [ ] **Step 3: Implement `topics.go`**

`Ensure`:
- build one `[]kafka.TopicConfig` from `t.Plain` then `t.Compact`, `NumPartitions: 1`, `ReplicationFactor: 1`; compact entries carry `ConfigEntries: []kafka.ConfigEntry{{ConfigName: "cleanup.policy", ConfigValue: "compact"}}`.
- one `c.CreateTopics(ctx, &kafka.CreateTopicsRequest{Addr: addr, Topics: cfgs})`. **No list-then-diff** (FR-2.5) — comment that this is deliberate: per-topic `TopicAlreadyExists` makes the create idempotent on its own, and the diff is exactly where the shell version's locale-collation bug lived.
- walk `resp.Errors`: `nil` ⇒ `Created++`; `errors.Is(err, kafka.TopicAlreadyExists)` ⇒ `Existing++`; anything else ⇒ append to a `[]error` and keep going. Non-empty at the end ⇒ return the joined error.
- then, if `len(t.Compact) > 0`, one `IncrementalAlterConfigs` over all compacted topics with `ConfigOperationSet`. Comment why `IncrementalAlterConfigs` and not `AlterConfigs`: the legacy API is full-replace for topic configs and would reset every other topic-level override to broker default (design §4 OQ-1). Unconditional over the compacted set, newly created and pre-existing alike — a topic created before this policy existed is exactly the case FR-2.6 is for, and setting a config that is already set is a no-op.
- per-resource `Error != nil` ⇒ error naming `ResourceName`.

`Settle`:
- return `map[string][]int{}` immediately for an empty `names`.
- loop: `c.Metadata(ctx, &kafka.MetadataRequest{Addr: addr, Topics: names})`; on success collect every returned topic that has `Error == nil` and `len(Partitions) > 0` into the result; if every name is present, return.
- otherwise, if `now().Sub(start)+poll > ceiling`, return an error naming the missing topics (or the last transport error).
- `sleep(poll)`; repeat.
- Doc-comment the reason this phase exists at all: `connPool.roundTrip` answers `Metadata` from the transport's **cached** cluster state (`transport.go:352-362`), refreshed on `Transport.MetadataTTL`. `ListOffsets` is split per topic-partition and routed to each leader looked up in that same cache, so a topic missing from it resolves to `Broker{ID: -1}`. Without this loop a first-sync run could create 170 topics and then fail to route offsets for them.

`EndOffsets`:
- return `map[string]map[int]int64{}` for empty input, no call.
- build `map[string][]kafka.OffsetRequest` with `kafka.LastOffsetOf(p)` for every pair; one `ListOffsets`.
- for each requested `(topic, partition)`, find it in the response; `Error != nil` ⇒ error naming topic and partition; absent ⇒ error naming topic and partition; else record `LastOffset`.

- [ ] **Step 4: Run the tests and confirm they pass**

```bash
go test ./internal/topics/ -v
go build ./... && go vet ./...
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-kafka-precreate/internal/topics
git commit -m "feat(kafka-precreate): topic create, alter, metadata settle and end offsets"
```

---

### Task 4: The `groups` package — seed and verify

The group-coordinator half: probe state, seed end-of-log offsets as a non-member, triage the probe/commit race, and run the asymmetric verification gate.

### Files

- `services/atlas-kafka-precreate/internal/groups/groups.go` — **new file**
- `services/atlas-kafka-precreate/internal/groups/groups_test.go` — **new file**
- `services/atlas-kafka-precreate/internal/discover/discover.go` — read-only; supplies `StateIsSeedable`
- `services/atlas-kafka-precreate/internal/kafkaops/ops.go` — read-only; supplies `AdminClient` and `RetryConfig`
- `deploy/k8s/base/kafka-precreate.sh` — **read-only reference**; `seed_override_offsets` (lines 245-300) and `verify_group_offsets` (lines 322-405) are the behaviour being ported

Module root for `go build` / `go test`: `services/atlas-kafka-precreate`.

**Interfaces**

- Consumes: `discover.StateIsSeedable` (Task 1); `kafkaops.AdminClient`, `kafkaops.RetryConfig`, `kafkaops.WithCoordinatorRetry` (Task 2); the `map[string][]int` from `topics.Settle` and the `map[string]map[int]int64` from `topics.EndOffsets` (Task 3).
- Produces:
  ```go
  package groups

  type SeedResult struct {
      Seeded  []string        // groups seeded this run, input order
      Skipped []string        // groups skipped (active state, or the commit race)
      States  map[string]string // group → observed state, for logging
  }

  func (r SeedResult) WasSkipped(group string) bool

  func Seed(ctx context.Context, c kafkaops.AdminClient, addr net.Addr, groupIDs []string,
      partitions map[string][]int, offsets map[string]map[int]int64,
      retry kafkaops.RetryConfig) (SeedResult, error)

  type VerifyReport struct {
      Group   string
      Total   int      // union (topic, partition) pairs checked
      Missing []string // topic names with at least one uncommitted partition, sorted
  }

  func Verify(ctx context.Context, c kafkaops.AdminClient, addr net.Addr, groupIDs []string,
      partitions map[string][]int, seeded SeedResult,
      retry kafkaops.RetryConfig) ([]VerifyReport, error)
  ```

  `Verify` returns one report per **skipped** group (the WARN material); it returns an error for the first seeded group with a missing offset. Reports for groups with zero missing entries are still returned, so `main` can log the "committed offsets present on all N topics" line.

- [ ] **Step 1: Write the failing tests**

`services/atlas-kafka-precreate/internal/groups/groups_test.go`.

The stub, package-local at the top of the file:

```go
type stubClient struct {
	describeCalls []*kafka.DescribeGroupsRequest
	describeFn    func(*kafka.DescribeGroupsRequest) (*kafka.DescribeGroupsResponse, error)

	commitCalls []*kafka.OffsetCommitRequest
	commitFn    func(*kafka.OffsetCommitRequest) (*kafka.OffsetCommitResponse, error)

	fetchCalls []*kafka.OffsetFetchRequest
	fetchFn    func(*kafka.OffsetFetchRequest) (*kafka.OffsetFetchResponse, error)
}
```

The other four `AdminClient` methods `panic("unexpected call")` — `groups` must never issue `CreateTopics`, `IncrementalAlterConfigs`, `Metadata`, or `ListOffsets`.

All tests pass `kafkaops.RetryConfig{Base: time.Millisecond, Max: time.Millisecond, Budget: 10 * time.Millisecond, Sleep: func(time.Duration) {}, Now: time.Now}` so no test waits.

Shared fixtures: `partitions = map[string][]int{"a": {0}, "b": {0}}`, `offsets = map[string]map[int]int64{"a": {0: 5}, "b": {0: 0}}`, and the realistic group name `const chanGroup = "Channel Service - 7c2f8b1e-0d4a-4a1b-9f3e-2c1d5e6f7a8b [pr-1450]"`.

`TestSeed` — table-driven:

| case | groups | describe response | commit response | expect |
|---|---|---|---|---|
| `empty group list` (NG6, FR-3.1) | `[]` | — | — | `SeedResult{}` with both slices empty; **zero** describe calls; zero commit calls |
| `seedable Empty group` | `[chanGroup]` | `Groups: [{GroupID: chanGroup, GroupState: "Empty"}]` | all partitions `Error: nil` | `Seeded == [chanGroup]`; 1 commit call |
| `commit request shape` (FR-3.5, FR-3.7) | `[chanGroup]` | state `"Empty"` | success | the commit request has `GroupID == chanGroup`, `GenerationID == -1`, `MemberID == ""`, and `Topics` carrying `{"a": [{Partition: 0, Offset: 5}], "b": [{Partition: 0, Offset: 0}]}` |
| `Dead group is seedable` | `[chanGroup]` | state `"Dead"` | success | seeded |
| `absent group is seedable` (FR-3.4) | `[chanGroup]` | `Groups: []` | success | seeded; the observed state recorded as `""` |
| `per-group describe error is seedable` | `[chanGroup]` | `Groups: [{GroupID: chanGroup, Error: kafka.GroupIdNotFound}]` | success | seeded; state `""` |
| `describe transport error is seedable` | `[chanGroup]` | `(nil, errors.New("boom"))` | success | seeded; state `""`; the probe can never itself fail the Job |
| `Stable group is skipped` | `[chanGroup]` | state `"Stable"` | — | `Skipped == [chanGroup]`; **zero** commit calls; `States[chanGroup] == "Stable"` |
| `future state is skipped` | `[chanGroup]` | state `"AssigningPartitions"` | — | skipped; zero commit calls |
| `commit race is a non-fatal skip` (FR-3.6) | `[chanGroup]` | state `"Empty"` | `Topics: {"a": [{Partition: 0, Error: kafka.UnknownMemberId}]}` | no error; `Skipped == [chanGroup]`; `Seeded` empty |
| `other commit error is fatal` | `[chanGroup]` | state `"Empty"` | `Topics: {"a": [{Partition: 0, Error: kafka.OffsetMetadataTooLarge}]}` | error naming the group, topic `a`, partition `0` |
| `commit transport error is fatal` | `[chanGroup]` | state `"Empty"` | `(nil, errors.New("boom"))` | error naming the group |
| `mixed groups` | `["World Service [pr-1450]", chanGroup]` | first `"Stable"`, second `"Empty"` | success | `Skipped == ["World Service [pr-1450]"]`, `Seeded == [chanGroup]`; exactly 1 commit call, and its `GroupID` is `chanGroup` |

`kafka.GroupIdNotFound` (69) and `kafka.OffsetMetadataTooLarge` (12) are real `kafka.Error` constants — verified at `kafka-go@v0.4.51/error.go:84` and `:27`. So are `kafka.NotLeaderForPartition` (`:21`) and `kafka.GroupAuthorizationFailed` (`:45`), used in `TestVerify` below.

Add `TestSeed_DescribeIsRetried`: `describeFn` returns `kafka.NotCoordinatorForGroup` twice, then a `"Empty"` state. Expect success and 3 describe calls — proving `WithCoordinatorRetry` wraps the probe.

`TestVerify`:

| case | groups | seeded/skipped | fetch response | expect |
|---|---|---|---|---|
| `empty group list` (FR-4.5) | `[]` | — | — | nil error, no reports, **zero** fetch calls |
| `seeded group fully committed` | `[chanGroup]` | seeded | `{"a": [{Partition: 0, CommittedOffset: 5}], "b": [{Partition: 0, CommittedOffset: 0}]}` | no error; one report with `Missing` empty |
| `seeded group missing an offset` (FR-4.2) | `[chanGroup]` | seeded | `{"a": [{Partition: 0, CommittedOffset: -1}], "b": [{Partition: 0, CommittedOffset: 0}]}` | **error** naming the group and topic `a` |
| `seeded group missing from response` | `[chanGroup]` | seeded | `{"a": [{Partition: 0, CommittedOffset: 5}]}` — `b` absent | error naming the group and topic `b` |
| `skipped group missing offsets warns` (FR-4.3) | `[chanGroup]` | skipped | both partitions `CommittedOffset: -1` | **no error**; one report with `Total == 2`, `Missing == ["a","b"]` |
| `skipped group fully committed` | `[chanGroup]` | skipped | both committed | no error; report with `Missing` empty |
| `partition-level fetch error on a seeded group` | `[chanGroup]` | seeded | `{"a": [{Partition: 0, Error: kafka.NotLeaderForPartition}]}` | error naming the group, topic, and partition |
| `top-level fetch error` | `[chanGroup]` | seeded | `Error: kafka.GroupAuthorizationFailed` | error naming the group — fatal for a seeded **and** a skipped group; an RPC-level failure is not the same as a missing offset |

`CommittedOffset < 0` is the missing sentinel — it is the typed replacement for the shell's `awk`-extracted `"-"`.

`TestVerify_MissingListIsSorted`: a skipped group with 3 missing topics supplied to `partitions` in the order `z`, `a`, `m`; expect `Missing == ["a","m","z"]`. The 10-name bound is `main`'s formatting concern (Task 5), not `Verify`'s — `VerifyReport.Missing` carries the full sorted list.

- [ ] **Step 2: Run the tests and confirm they fail**

```bash
go test ./internal/groups/ -v
```

Expected: compile failure — `undefined: groups.Seed`.

- [ ] **Step 3: Implement `groups.go`**

`Seed`:
- return a zero `SeedResult` immediately for an empty `groupIDs`, before any Kafka call. Doc-comment it as NG6: that is `main`, whose groups carry real committed offsets that must never be reset, and this being the first statement is what makes "`main` is provably never touched" a property of the code rather than of the caller.
- per group: `state, _ := probe(...)` — `WithCoordinatorRetry` around `DescribeGroups{GroupIDs: []string{g}}`; a transport error, a per-group `Error`, or an absent group all collapse to `""`.
- `discover.StateIsSeedable(state)` false ⇒ record in `Skipped`/`States`, continue, **no commit issued**.
- true ⇒ build `map[string][]kafka.OffsetCommit` from `partitions` + `offsets` and issue one `OffsetCommit` with `GenerationID: -1`, `MemberID: ""`, wrapped in `WithCoordinatorRetry`.
- walk `resp.Topics[t][p].Error`: `errors.Is(err, kafka.UnknownMemberId)` ⇒ non-fatal, record in `Skipped`, stop processing that group. Any other non-nil ⇒ fatal, naming group, topic, partition, and error. Doc-comment the nuance from spike Q3: code 25 is not literally "group is active", but because the tool always sends `MemberID: ""` it unambiguously means active — and the intent is only explicit because of the `DescribeGroups` pre-check, which is why the pre-check is kept rather than replaced by error handling alone.

`Verify`:
- return `nil, nil` immediately for an empty `groupIDs` (FR-4.5), symmetric with `Seed`.
- per group: one `OffsetFetch{GroupID: g, Topics: partitions}` wrapped in `WithCoordinatorRetry`.
- `resp.Error != nil` ⇒ fatal naming the group, regardless of seeded/skipped.
- per partition: `Error != nil` ⇒ fatal naming group/topic/partition. `CommittedOffset < 0`, or the pair absent from the response, ⇒ missing.
- a requested `(topic, partition)` missing ⇒ if the group is in `seeded.Skipped`, add the topic name to the report; otherwise return an error naming group and topic (FR-4.2).
- doc-comment the asymmetry: a group skipped because it was already active has live consumers joined to it, which is the very end state this gate exists to establish. Re-proving it against the full union would fail the Job the first time a topic is added to a live environment, so for a skipped group the gate degrades to a report and the Job stays green (an unseeded topic falls back to the consumer's own `auto.offset.reset`).

- [ ] **Step 4: Run the tests and confirm they pass**

```bash
go test ./internal/groups/ -v
go build ./... && go vet ./...
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-kafka-precreate/internal/groups
git commit -m "feat(kafka-precreate): group offset seeding and verification gate"
```

---

### Task 5: `main.go`, the Dockerfile, and the README

Wiring only: read the environment, build the client, run the five phases in order, format the logs, map failures to exit codes. Plus the scratch image that ships it.

### Files

- `services/atlas-kafka-precreate/main.go` — **new file**
- `services/atlas-kafka-precreate/Dockerfile` — **new file**
- `services/atlas-kafka-precreate/README.md` — **new file**
- `services/atlas-kafka-precreate/internal/{discover,topics,groups,kafkaops}` — read-only; all four packages are complete

Module root for `go build` / `go test`: `services/atlas-kafka-precreate`.

Patterns to copy: `services/atlas-pr-bootstrap/Dockerfile` (a flat support image with its own Dockerfile and a service-directory build context — the precedent this follows; ignore its Alpine/`rpk` content, only the shape applies).

**Interfaces**

- Consumes: every exported symbol from Tasks 1-4.
- Produces: the binary `/atlas-kafka-precreate`.

- [ ] **Step 1: Write `main.go`**

No test file for `main` — it is wiring over four fully-tested packages, and a `TestMain`-style harness would need the process environment, which is exactly what `discover.FromEnviron(os.Environ())` exists to keep out of the packages that matter.

Structure:

```go
func main() {
	if err := run(); err != nil {
		logrus.WithError(err).Error("kafka precreate failed")
		os.Exit(1)
	}
}

func run() error { … }
```

`run()`, in order:

1. `logrus.SetFormatter(&logrus.JSONFormatter{})` — matches the Atlas Go services' output so Job logs land in the same pipeline (NFR-7).
2. `bootstrap := os.Getenv("BOOTSTRAP_SERVERS")`; empty ⇒ return an error saying `BOOTSTRAP_SERVERS not set in atlas-env` (FR-1.6). This is checked **before** any Kafka construction.
3. `ctx, cancel := context.WithTimeout(context.Background(), 240*time.Second)`; `defer cancel()`. Comment: comfortably inside the Job's `activeDeadlineSeconds: 300`, so the tool's own deadline fires first and produces a named error rather than an opaque pod kill (NFR-4).
4. Build the client:
   ```go
   addr := kafka.TCP(strings.Split(bootstrap, ",")...)
   client := &kafka.Client{
       Addr:      addr,
       Timeout:   60 * time.Second,
       Transport: &kafka.Transport{MetadataTTL: 1 * time.Second},
   }
   ```

   `addr` is bound separately because every `topics.*` and `groups.*` entry point takes it explicitly (each request struct carries its own `Addr`, and the request's takes precedence over the client's).
   Comment `MetadataTTL`: the default is 6s and the transport serves `Metadata` from that cache, so a shorter TTL is what lets `topics.Settle` observe freshly created topics quickly (design §3 Phase C).
5. Phase A: `t := discover.FromEnviron(os.Environ())`; `groupIDs := discover.Groups(os.Getenv("KAFKA_CONSUMER_GROUP"))`; `union := t.Union()`.
6. Phase B: `topics.Ensure(...)`; log `phase=create plain=<len> compact=<len> created=<n> existing=<n>` and, when `len(t.Compact) > 0`, `phase=alter topics=<n> policy=compact`.
7. If `len(groupIDs) == 0`: log `phase=seed skipped=true reason="KAFKA_CONSUMER_GROUP unset — main, NG6"` and **return nil**. Phases C, D and E are all skipped; nothing downstream needs partition data, which keeps the `main` environment's run to exactly two RPCs.
8. Phase C: `parts, err := topics.Settle(ctx, client, addr, union, topics.SettleConfig{})` (zero config ⇒ 250ms poll, 30s ceiling, real clock).
9. `offsets, err := topics.EndOffsets(ctx, client, addr, parts)`.
10. Phase D: `res, err := groups.Seed(ctx, client, addr, groupIDs, parts, offsets, kafkaops.DefaultRetryConfig())`. Log one line per group — `phase=seed group=<name> outcome=seeded partitions=<n>` or `phase=seed group=<name> outcome=skipped state=<state>` — then the summary `phase=seed seeded=<n> skipped=<n>`. When `len(res.Seeded) == 0 && len(res.Skipped) > 0`, also log the shell's re-sync no-op line: `all N override consumer groups were already active — nothing seeded this run (re-sync no-op)`.
11. Phase E: `reports, err := groups.Verify(ctx, client, addr, groupIDs, parts, res, kafkaops.DefaultRetryConfig())`. For each report with `len(Missing) == 0`, log `phase=verify group=<name> outcome=ok topics=<total>`. Otherwise `logrus.Warn` with `group`, `missing=<len(Missing)>`, `of=<Total>`, `topics="<first up to 10, comma-joined>"` and, when more than 10 are missing, `more=<len(Missing)-10>` (FR-4.4). Finish with `phase=verify ok`.

The 10-name bound lives here and nowhere else. Write it as a small unexported helper so the truncation is one expression, not an inline loop.

Every group name goes into a logrus **field**, never into the message string — a name containing spaces and brackets must not need quoting rules to stay readable.

- [ ] **Step 2: Build and vet**

```bash
go build ./...
go vet ./...
go test ./...
```

Expected: builds clean; all package tests from Tasks 1-4 still pass.

- [ ] **Step 3: Write the Dockerfile**

`services/atlas-kafka-precreate/Dockerfile`. Two stages, build context is the service directory (which is possible only because the module depends on no `libs/atlas-*`):

```dockerfile
# syntax=docker/dockerfile:1.26
# Sync-wave-0 Kafka topic pre-creation tool. Replaces the apache/kafka:3.7.2
# image and its JVM CLI (deploy/k8s/base/kafka-precreate.sh, deleted in
# task-260): the entire topic pass is one CreateTopics request over one
# connection instead of ~170 JVM cold starts.
#
# The runtime stage is scratch on purpose (NFR-9): no shell, no Kafka CLI,
# no JRE. The only thing copied in beside the binary is the CA bundle.
ARG GO_VERSION=1.26.0
ARG ALPINE_VERSION=3.23

FROM golang:${GO_VERSION}-alpine${ALPINE_VERSION} AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /atlas-kafka-precreate .

FROM scratch
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build /atlas-kafka-precreate /atlas-kafka-precreate
USER 65532:65532
ENTRYPOINT ["/atlas-kafka-precreate"]
```

`COPY . .` picks up the `internal/` tree; there is no `.dockerignore` in the service directory and none is needed (the only other files are `Dockerfile` and `README.md`).

- [ ] **Step 4: Build the image**

```bash
docker build -t atlas-kafka-precreate:local services/atlas-kafka-precreate
```

Expected: a successful build. **This is the only place the image is ever built by the gate path** — `tools/verify.sh`'s bake step selects only `type == "go-service"` entries (`verify.sh:293-302`), so it never builds this target. Do not skip this step.

- [ ] **Step 5: Write the README**

`services/atlas-kafka-precreate/README.md`, short. Cover: what the tool is and where it runs (sync-wave 0 Job, `deploy/k8s/base/atlas-kafka-precreate.yaml`); the three input environment variables and what each does; the five phases; the exit-code contract; that it replaced `deploy/k8s/base/kafka-precreate.sh`; and that `tools/verify.sh` does not build its image, so `docker build` is a manual pre-PR step. Repo-relative paths only, no absolute paths.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-kafka-precreate/main.go services/atlas-kafka-precreate/Dockerfile services/atlas-kafka-precreate/README.md
git commit -m "feat(kafka-precreate): main wiring, scratch image and README"
```

---

### Task 6: Cutover — build registration, Job manifest, and deleting the shell

Mechanical, but every edit here is load-bearing for CI. Read the two hard constraints in Step 3 before touching the Job manifest.

### Files

- `.github/config/services.json` — modify: add the `atlas-kafka-precreate` entry
- `docker-bake.hcl` — modify: add the target and extend the `all-services` group
- `deploy/k8s/base/atlas-kafka-precreate.yaml` — modify: image, drop `command`/volume/volumeMount, add `activeDeadlineSeconds`, repair the header comment
- `deploy/k8s/base/kustomization.yaml` — modify: delete the `atlas-kafka-precreate-script` `configMapGenerator` entry (lines 87-92) including its comment
- `deploy/k8s/base/kafka-precreate.sh` — **delete**
- `deploy/k8s/base/atlas-kafka-precreate_test.sh` — **delete**

Overlay wiring is Task 7, not this task.

- [ ] **Step 1: Register the service**

`.github/config/services.json` — insert in the same alphabetical position the file already uses, mirroring the `atlas-pr-bootstrap` entry's shape exactly:

```json
{
  "name": "atlas-kafka-precreate",
  "type": "support-image",
  "path": "services/atlas-kafka-precreate",
  "docker_image": "ghcr.io/chronicle20/atlas-kafka-precreate/atlas-kafka-precreate",
  "docker_context": "services/atlas-kafka-precreate"
},
```

`type` is `support-image`, **not** `go-service`. `tools/service-registration-guard.sh:119` only requires `services/{name}/atlas.com/*/main.go` for `go-service` entries, and `verify.sh`'s bake step only builds `go-service` entries; a wrong `type` here would demand the nested layout and fail the guard.

- [ ] **Step 2: Add the bake target**

`docker-bake.hcl` — after the `atlas-pr-bootstrap` target:

```hcl
target "atlas-kafka-precreate" {
  # Flat Go module with no libs/atlas-* dependency, so the context is the
  # service directory rather than the repo root (same as atlas-pr-bootstrap).
  context    = "services/atlas-kafka-precreate"
  dockerfile = "Dockerfile"
  tags       = ["atlas-kafka-precreate:${ATLAS_IMAGE_TAG}"]
}
```

and extend the group:

```hcl
group "all-services" {
  targets = concat(go_services, ["atlas-ui", "atlas-pr-bootstrap", "atlas-kafka-precreate"])
}
```

**Guard interaction — get the form exactly right.** `tools/service-registration-guard.sh:254-256` scans the comment-stripped bake file with `^\s*"(atlas-[a-z0-9-]+)",?\s*$` and fails on any match that is not a `go-service`. That regex matches a line consisting *solely* of a quoted name — i.e. a `go_services = [...]` entry. `target "…" {` has a trailing brace and the single-line `concat(…)` has extra content, so neither matches. **Do not** reformat the `concat` call onto multiple lines: putting `"atlas-kafka-precreate",` alone on a line would fail the guard.

- [ ] **Step 3: Rewrite the Job manifest**

`deploy/k8s/base/atlas-kafka-precreate.yaml`. The container block becomes:

```yaml
spec:
  backoffLimit: 3
  ttlSecondsAfterFinished: 600
  activeDeadlineSeconds: 300
  template:
    spec:
      restartPolicy: OnFailure
      containers:
        - name: kafka-precreate
          image: ghcr.io/chronicle20/atlas-kafka-precreate/atlas-kafka-precreate:latest
          envFrom:
            - configMapRef:
                name: atlas-env
```

Remove `command`, `volumeMounts`, and the whole `volumes:` block. Keep the sync-wave annotation, the `Force=true,Replace=true` sync-option, `backoffLimit`, `ttlSecondsAfterFinished`, `restartPolicy`, and — critically — the container **name** `kafka-precreate`.

**Two hard constraints, both from `.github/workflows/pr-validation.yml:1091-1095`:**

1. The container must stay at **index 0** — the JSON-6902 patch targets `/spec/template/spec/containers/0/env`.
2. The container must keep **`envFrom` only, with no `env:` key**. That patch uses `op: add` on the whole `env` path, which *creates* the key. If this manifest ever grows an `env:` key, the patch silently **replaces** it and `KAFKA_CONSUMER_GROUP` becomes the container's only environment variable. Add a comment saying so, immediately above `envFrom`, so the next editor sees it.

The image tag: check what the other base manifests use for their `image:` line (the overlays' `images:` blocks retag by name) and match that convention exactly rather than inventing one — `grep -n 'image:' deploy/k8s/base/atlas-account.yaml` shows the house style. If base manifests use a bare `:latest`, use `:latest`.

The header comment block is the only surviving prose explanation of *why* wave 0 exists, why `Force=true,Replace=true` is there, and why an active group is skipped rather than reset. **Preserve it and update it in place** — do not delete it along with the script it references. Specifically:
- the "Mechanism:" paragraph must stop describing `kafka-topics.sh --create --if-not-exists once per topic` and describe one `CreateTopics` request instead;
- "Idempotent: … (`--if-not-exists`)" becomes per-topic `TopicAlreadyExists` tolerance;
- the "see kafka-precreate.sh for the mechanism" pointer becomes `services/atlas-kafka-precreate/`;
- the final sentence about the script being mounted from the `atlas-kafka-precreate-script` ConfigMap so it is "independently sourceable and testable" is now false — replace it with a pointer to the Go unit tests.

Everything the comment says about NG6, the first-sync/re-sync distinction, and why skipping an active group is the safety property stays true and stays put.

- [ ] **Step 4: Remove the configMap generator**

`deploy/k8s/base/kustomization.yaml` — delete lines 87-92, i.e. both the three-line comment beginning `# The atlas-kafka-precreate Job's script body, extracted (task-232 Task 45)` and the `- name: atlas-kafka-precreate-script` entry with its `files:` list. Leave the `atlas-ingress-routes` generator and the `resources:` entry at line 33 untouched.

- [ ] **Step 5: Delete the shell script and its test**

```bash
git rm deploy/k8s/base/kafka-precreate.sh deploy/k8s/base/atlas-kafka-precreate_test.sh
```

- [ ] **Step 6: Commit**

```bash
git add .github/config/services.json docker-bake.hcl deploy/k8s/base
git commit -m "feat(kafka-precreate): cut the wave-0 Job over to the Go image"
```

The tree still contains dangling references to the deleted script from the `pr-sparse` overlay; Task 7 clears them. Do not run the guards yet.

---

### Task 7: Overlay wiring and dangling references

The three overlays need an `images:` entry for the new image, and one comment still points at the script Task 6 deleted. Small, but the guards do not pass until it is done.

### Files

- `deploy/k8s/overlays/main/kustomization.yaml` — modify: `images:` entry (insert after line 281)
- `deploy/k8s/overlays/pr/kustomization.yaml` — modify: `images:` entry (insert after line 435)
- `deploy/k8s/overlays/pr-sparse/kustomization.yaml` — modify: `images:` entry (insert after line 570) **and** the comment at line 274
- `deploy/k8s/overlays/pr-cleanup/kustomization.yaml` — **read-only**; deliberately gets no entry, see Step 2

`deploy/k8s/overlays/pr/kustomization.yaml` carries **no** reference to the deleted script — verified by grep. Do not go looking for one.

- [ ] **Step 1: Repair the pr-sparse comment**

`deploy/k8s/overlays/pr-sparse/kustomization.yaml:272-274` currently reads:

```
  # with a JSON-6902 patch adding a newline-delimited KAFKA_CONSUMER_GROUP
  # to the wave-0 atlas-kafka-precreate Job, so seed_override_offsets /
  # verify_group_offsets stop taking their unset-guard early return
  # (kafka-precreate.sh:176, task-243 design §1.1 — the mechanism exists and
```

Rewrite the two middle lines so they name the Go entry points instead of the deleted shell functions and point at `services/atlas-kafka-precreate/internal/groups/` rather than `kafka-precreate.sh:176`. Keep the surrounding lines — the anchor convention, and the reason the patch targets the Job rather than `atlas-env` — byte-identical.

- [ ] **Step 2: Add the image to the three overlays' `images:` blocks**

The design doc does not mention this and it is not optional: `atlas-pr-bootstrap`, the other `support-image`, carries an entry in all three overlays, and without one the new image is never retagged and every environment pulls a stale `:latest`.

Insert, in alphabetical position between the `atlas-invites` and `atlas-keys` entries:

- `deploy/k8s/overlays/main/kustomization.yaml` — after line 281 (`atlas-invites` + its `newTag`). Copy the `newTag:` value from the `atlas-pr-bootstrap` entry at line 328 (`main-2080dc7`), **not** from the neighboring services — main's overlay carries per-image tags and CI's reconcile job bumps them independently.
- `deploy/k8s/overlays/pr/kustomization.yaml` — after line 435; `newTag: latest`.
- `deploy/k8s/overlays/pr-sparse/kustomization.yaml` — after line 570; `newTag: latest`.

Each entry is two lines:

```yaml
  - name: ghcr.io/chronicle20/atlas-kafka-precreate/atlas-kafka-precreate
    newTag: latest
```

`deploy/k8s/overlays/pr-cleanup/kustomization.yaml` gets **no** entry — its `images:` block lists only `atlas-pr-bootstrap` because that is the only image its teardown Job runs. The precreate Job is not part of cleanup.

- [ ] **Step 3: Verify nothing still points at the deleted files**

```bash
grep -rn 'kafka-precreate\.sh\|atlas-kafka-precreate-script\|atlas-kafka-precreate_test\|seed_override_offsets\|verify_group_offsets\|state_is_seedable\|seed_group' --exclude-dir=.git --exclude-dir=docs .
```

Expected: **no output**. Any hit outside `docs/` is a dangling reference to fix now. (`docs/` is excluded because the task's own PRD and design quote the old names deliberately.)

- [ ] **Step 4: Run the guards**

```bash
./tools/service-registration-guard.sh
./tools/overlay-env-guard.sh
./tools/pr-sparse-mirror-guard.sh
```

Expected: exit 0 from all three. If the first reports `docker-bake.hcl lists "atlas-kafka-precreate" which is not a go-service`, the bake entry was written on its own line inside a list — go back to Step 2 and use the single-line `concat` form.

- [ ] **Step 5: Commit**

```bash
git add deploy/k8s/overlays
git commit -m "feat(kafka-precreate): wire the new image into the three overlays"
```

---

### Task 8: Full gate

- [ ] **Step 1: Run the flagless gate**

```bash
./tools/verify.sh
```

This branch changes `go.work`, so `fanout_paths` (`verify.sh:186-189`) matches and the run fans out to **every** module in the workspace. That is expected and unavoidable for a new module. While iterating on an earlier failure, `--base <last-gated-commit>` narrows the change set; the final run that gates the PR must be flagless.

Expected: exit 0. Anything else is a real failure — read `docs/verification.md` for the specific guard rather than working around it.

- [ ] **Step 2: Confirm the image still builds**

```bash
docker buildx bake atlas-kafka-precreate
```

`verify.sh` does not build this target (its bake step filters to `type == "go-service"`), so this is a separate, mandatory check.

- [ ] **Step 3: Manual acceptance against a real broker**

Not automatable in this repo — there is no Kafka test harness and adding one is out of scope. Run the binary against a throwaway single-node KRaft broker (`apache/kafka:3.7.2`) with a handful of `COMMAND_TOPIC_*` / `EVENT_TOPIC_*` variables and a `KAFKA_CONSUMER_GROUP` containing a name with spaces and brackets. Confirm: exit 0, topics created, `cleanup.policy=compact` on the three config-status topics, offsets committed, and a second run (steady state) completing in well under 5 seconds (NFR-1). Record the observed wall-clock in the PR description.

- [ ] **Step 4: Commit any fixes and stop**

Do **not** open a PR from this task. Code review runs first (CLAUDE.md: "Never open a PR without code review").
