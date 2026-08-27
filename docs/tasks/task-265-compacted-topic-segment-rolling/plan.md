# Compacted Config-Status Topics Never Actually Compact — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the three config-status compacted topics actually compact by shipping `max.compaction.lag.ms`, `segment.ms`, and `min.cleanable.dirty.ratio` alongside `cleanup.policy=compact` on both the `CreateTopics` and `IncrementalAlterConfigs` paths of `atlas-kafka-precreate`.

**Architecture:** One package-level declaration (`compactTopicConfigs`) in `internal/topics/topics.go` is the single source of truth; two pure projections (`compactCreateEntries`, `compactAlterConfigs`) build the two request bodies, so the defect class — a config present in one request builder and absent from the other — becomes unrepresentable. `Ensure`'s control flow is unchanged: the alter still runs unconditionally over the whole compacted set, which is what heals the already-grown `-main` topics.

**Tech Stack:** Go 1.27, `github.com/segmentio/kafka-go` (`CreateTopicsRequest`, `IncrementalAlterConfigsRequest`), `logrus`, stdlib `testing` with the package-local `stubClient`.

**Spec:** `docs/tasks/task-265-compacted-topic-segment-rolling/design.md` (PRD at `prd.md`)

## Global Constraints

- **Module root for every `go build` / `go test` in this plan:** `services/atlas-kafka-precreate` (module `atlas.com/kafka-precreate`). All Go paths below are relative to that directory unless the path starts with `services/`.
- **Nothing outside `services/atlas-kafka-precreate/` is touched.** No deploy/overlay change, no `libs/` change, no new `discover` category.
- **Config values, verbatim (PRD FR-1.1):** `cleanup.policy` = `"compact"`, `max.compaction.lag.ms` = `"600000"`, `segment.ms` = `"600000"`, `min.cleanable.dirty.ratio` = `"0.01"`. All four are Go `string` constants; none is an inline literal at a call site.
- **Declaration order is load-bearing** (a test asserts it): `cleanup.policy`, `max.compaction.lag.ms`, `segment.ms`, `min.cleanable.dirty.ratio`.
- **No swallowed alter errors.** There is no allowlist of tolerable broker rejections and no partial-application fallback: transport error → fatal; any per-resource error → fatal, joined, naming the topic (design §3.3).
- **No stubs, no `// TODO`, no deferred work.** Every task lands complete.
- Tests must not require a broker; they run against the existing `stubClient` in `internal/topics/topics_test.go:24-80`.

---

### Task 1: Shared compact-config declaration and its two projections

Introduces the single declaration and both projections in `internal/topics/topics.go`, rewires `Ensure`'s two call sites onto them, and updates the two alter error wraps. This is the whole behaviour change; Tasks 2–3 add the anti-regression test surface and the operator-facing documentation.

### Files

- `services/atlas-kafka-precreate/internal/topics/topics.go` — replace the lone `compactCleanupPolicy` const with the four-constant block, add `compactTopicConfig`, `compactTopicConfigs`, `compactCreateEntries()`, `compactAlterConfigs()`, `CompactConfigNames()`; rewire the `t.Compact` create loop (currently `topics.go:54-63`) and the alter resource loop (`topics.go:100-113`); reword the two error wraps at `topics.go:116` and `topics.go:122`
- `services/atlas-kafka-precreate/internal/topics/topics_test.go` — extend `TestEnsure_AlterConfigs` (`:282-330`) and `TestEnsure_SingleCreateRequest`'s compacted-entry assertion (`:165-171`) to the new four-config expectation; add `TestCompactConfigNames`

Module root: `services/atlas-kafka-precreate`.

Patterns to copy: the existing const + `Ensure` shape already in `services/atlas-kafka-precreate/internal/topics/topics.go:23`; test setup shape from `services/atlas-kafka-precreate/internal/topics/topics_test.go:282-306` (stub construction, `Ensure` call, `sort.Slice` over `req.Resources`).

**Interfaces:**
- Consumes: `kafkaops.AdminClient` (`internal/kafkaops/ops.go`), `discover.Topics{Plain, Compact []string}` — both unchanged.
- Produces:
  - `func CompactConfigNames() []string` — exported; returns the four config names in declaration order. Task 3 calls it from `main.go`.
  - unexported: `type compactTopicConfig struct{ name, value string }`, `var compactTopicConfigs []compactTopicConfig`, `func compactCreateEntries() []kafka.ConfigEntry`, `func compactAlterConfigs() []kafka.IncrementalAlterConfigsRequestConfig`. Task 2's tests reference `compactTopicConfigs` and `CompactConfigNames` only.

- [ ] **Step 1: Write the failing tests**

Three edits in `internal/topics/topics_test.go`. Setup is unchanged in each — reuse the existing stub construction verbatim.

**1a.** In `TestEnsure_SingleCreateRequest`, replace the compacted-topic `want` block (currently the one-entry `[]kafka.ConfigEntry{{ConfigName: "cleanup.policy", ConfigValue: "compact"}}` and its `cfg.ConfigEntries[0] != want[0]` comparison) with a four-entry expectation compared element-wise in order. Expected value, literals not constants:

| i | `ConfigName` | `ConfigValue` |
|---|---|---|
| 0 | `cleanup.policy` | `compact` |
| 1 | `max.compaction.lag.ms` | `600000` |
| 2 | `segment.ms` | `600000` |
| 3 | `min.cleanable.dirty.ratio` | `0.01` |

The `else if len(cfg.ConfigEntries) != 0` plain-topic branch stays exactly as it is.

**1b.** In `TestEnsure_AlterConfigs`, replace `if len(res.Configs) != 1` / the single-`want` comparison with the four-config expectation, same order, each carrying `ConfigOperation: kafka.ConfigOperationSet`:

```go
wantConfigs := []kafka.IncrementalAlterConfigsRequestConfig{
	{Name: "cleanup.policy", Value: "compact", ConfigOperation: kafka.ConfigOperationSet},
	{Name: "max.compaction.lag.ms", Value: "600000", ConfigOperation: kafka.ConfigOperationSet},
	{Name: "segment.ms", Value: "600000", ConfigOperation: kafka.ConfigOperationSet},
	{Name: "min.cleanable.dirty.ratio", Value: "0.01", ConfigOperation: kafka.ConfigOperationSet},
}
if len(res.Configs) != len(wantConfigs) {
	t.Fatalf("resource %q: expected %d configs, got %d", res.ResourceName, len(wantConfigs), len(res.Configs))
}
for j, want := range wantConfigs {
	if res.Configs[j] != want {
		t.Errorf("resource %q config %d: expected %+v, got %+v", res.ResourceName, j, want, res.Configs[j])
	}
}
```

Everything else in that test is unchanged: still `len(stub.alterCalls) != 1`, still `len(req.Resources) != 2`, still the `sort.Slice` by `ResourceName`, still `ResourceType == kafka.ResourceTypeTopic`.

**1c.** Add `TestCompactConfigNames` — not table-driven, a single ordered comparison:

```go
func TestCompactConfigNames(t *testing.T) {
	want := []string{"cleanup.policy", "max.compaction.lag.ms", "segment.ms", "min.cleanable.dirty.ratio"}
	got := CompactConfigNames()
	if len(got) != len(want) {
		t.Fatalf("expected %d names, got %d: %v", len(want), len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("name %d: expected %q, got %q", i, want[i], got[i])
		}
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run from `services/atlas-kafka-precreate`:

```bash
go test ./internal/topics/ -run 'TestEnsure_SingleCreateRequest|TestEnsure_AlterConfigs$|TestCompactConfigNames' -v
```

Expected: compile failure — `undefined: CompactConfigNames`. (The two extended tests cannot report a value failure until the package compiles; that is the expected shape of this red step.)

- [ ] **Step 3: Write the declaration and projections**

In `internal/topics/topics.go`, replace `const compactCleanupPolicy = "compact"` (line 23) with the following block, placed in the same spot, above `EnsureResult`. The comments are part of the deliverable — FR-4.2 requires a future reader be able to tell which knob is load-bearing.

```go
const (
	compactCleanupPolicy = "compact"

	// compactMaxCompactionLagMs bounds how long a record may sit
	// uncompacted. This is the knob that makes cleanup.policy=compact mean
	// anything: the cleaner never touches a partition's ACTIVE segment, so
	// a topic whose segment never rolls is never cleaned no matter what its
	// cleanup policy says. For a compacted topic the broker's effective
	// roll deadline is min(segment.ms, max.compaction.lag.ms), so setting
	// this to 10 minutes makes the segment roll ten minutes after its first
	// record — and the roll is what hands the cleaner something to work on.
	// Verified against apache/kafka:4.1.1 (design §2.2): a topic carrying
	// only this config rolled and compacted 200 records over 3 keys down to
	// 3, with no segment.ms set at all.
	compactMaxCompactionLagMs = "600000"

	// compactSegmentMs is the same 10-minute deadline expressed on the
	// policy-independent knob. It is deliberately equal to
	// compactMaxCompactionLagMs and is therefore redundant while the topic
	// is compacted (the broker takes the min of the two). It is set anyway
	// so the roll bound survives someone changing cleanup.policy, and so
	// the roll cadence is legible from a --describe without knowing the
	// min() rule. It is NOT the knob doing the work — deleting
	// max.compaction.lag.ms and keeping this one would not be equivalent
	// for a compacted topic.
	compactSegmentMs = "600000"

	// compactMinCleanableDirtyRatio lets the cleaner select a segment whose
	// dirty fraction is small, rather than the 0.5 default. This one was
	// not isolated by the live experiments (both ran at a dirty ratio near
	// 1.0) and the forced-cleaning path max.compaction.lag.ms drives
	// bypasses the ratio check anyway; it is cheap steady-state insurance,
	// inert-but-harmless rather than load-bearing.
	compactMinCleanableDirtyRatio = "0.01"
)

// compactTopicConfig is one (name, value) pair in the configuration every
// compacted topic must carry. It exists so the CreateTopics and
// IncrementalAlterConfigs request bodies are two projections of one
// declaration: a config applied at creation but not at alter (or the
// reverse) is precisely the defect this set was added to fix.
//
// It is a neutral local pair type rather than []kafka.ConfigEntry because
// the alter direction needs a third field (ConfigOperation) that
// kafka.ConfigEntry has no counterpart for — one of the two directions is a
// projection either way, so the canonical declaration keeps kafka types out
// of it and reads as policy.
type compactTopicConfig struct {
	name  string
	value string
}

var compactTopicConfigs = []compactTopicConfig{
	{name: "cleanup.policy", value: compactCleanupPolicy},
	{name: "max.compaction.lag.ms", value: compactMaxCompactionLagMs},
	{name: "segment.ms", value: compactSegmentMs},
	{name: "min.cleanable.dirty.ratio", value: compactMinCleanableDirtyRatio},
}

// compactCreateEntries projects the declaration onto a CreateTopics
// per-topic config body. It builds a fresh slice per call:
// kafka.TopicConfig.ConfigEntries is a per-topic field kafka-go reads but
// does not document as immutable, and sharing one backing array across N
// resources would be a silent aliasing hazard for no measurable gain at
// these sizes.
func compactCreateEntries() []kafka.ConfigEntry {
	entries := make([]kafka.ConfigEntry, len(compactTopicConfigs))
	for i, cfg := range compactTopicConfigs {
		entries[i] = kafka.ConfigEntry{ConfigName: cfg.name, ConfigValue: cfg.value}
	}
	return entries
}

// compactAlterConfigs projects the same declaration onto an
// IncrementalAlterConfigs per-resource config body, set-only. Fresh slice
// per call, for the reason on compactCreateEntries.
func compactAlterConfigs() []kafka.IncrementalAlterConfigsRequestConfig {
	configs := make([]kafka.IncrementalAlterConfigsRequestConfig, len(compactTopicConfigs))
	for i, cfg := range compactTopicConfigs {
		configs[i] = kafka.IncrementalAlterConfigsRequestConfig{
			Name:            cfg.name,
			Value:           cfg.value,
			ConfigOperation: kafka.ConfigOperationSet,
		}
	}
	return configs
}

// CompactConfigNames returns the names of the configs applied to every
// compacted topic, in declaration order. It exists so the alter-phase log
// line cannot drift from what was actually sent. Names only: the values are
// four short numbers recoverable from a --describe.
func CompactConfigNames() []string {
	names := make([]string, len(compactTopicConfigs))
	for i, cfg := range compactTopicConfigs {
		names[i] = cfg.name
	}
	return names
}
```

- [ ] **Step 4: Rewire the two call sites in `Ensure`**

Replace the inline `ConfigEntries` literal in the `t.Compact` create loop:

```go
	for _, name := range t.Compact {
		cfgs = append(cfgs, kafka.TopicConfig{
			Topic:             name,
			NumPartitions:     1,
			ReplicationFactor: 1,
			ConfigEntries:     compactCreateEntries(),
		})
	}
```

and the inline `Configs` literal in the alter resource loop:

```go
	resources := make([]kafka.IncrementalAlterConfigsRequestResource, len(t.Compact))
	for i, name := range t.Compact {
		resources[i] = kafka.IncrementalAlterConfigsRequestResource{
			ResourceType: kafka.ResourceTypeTopic,
			ResourceName: name,
			Configs:      compactAlterConfigs(),
		}
	}
```

Do **not** change anything else in `Ensure`: the plain-topic loop still emits no `ConfigEntries`; the `len(t.Compact) == 0` short circuit stays; the alter still runs over all of `t.Compact` rather than the subset `CreateTopics` reported as newly created — that unconditional sweep is the entire self-healing mechanism for the already-grown `-main` topics.

- [ ] **Step 5: Reword the two alter error wraps**

`"setting cleanup.policy=compact"` now names only one of four configs, so both wraps become:

```go
	alterResp, err := c.IncrementalAlterConfigs(ctx, &kafka.IncrementalAlterConfigsRequest{Addr: addr, Resources: resources})
	if err != nil {
		return EnsureResult{}, fmt.Errorf("applying compacted topic config: %w", err)
	}

	var alterFatal []error
	for _, res := range alterResp.Resources {
		if res.Error != nil {
			alterFatal = append(alterFatal, fmt.Errorf("applying compacted topic config on topic %q: %w", res.ResourceName, res.Error))
		}
	}
```

Behaviour is unchanged — both paths still return fatal, still joined, still naming the topic. Deliberately absent: any allowlist of configs whose rejection is tolerated, and any partial-application retry. A silently-uncompacted compacted topic is the bug being fixed and must not be reachable through a swallowed error.

- [ ] **Step 6: Extend the `IncrementalAlterConfigs` rationale comment**

The existing comment block above `resources := make(...)` explains incremental-vs-legacy and the unconditional sweep. Keep every sentence and append the new reason the sweep matters:

```go
	// The set is now four configs, not one. cleanup.policy=compact on its
	// own is inert: the cleaner never touches a partition's active
	// segment, so a topic whose segment never rolls is never cleaned. The
	// unconditional sweep is therefore also the repair path for topics
	// created by an earlier version of this tool — applying the new roll
	// bound makes their oversized active segment roll on the next append,
	// and the cleaner collapses it on its next pass (design §2.3).
```

Also update the package doc comment's "applies the compacted cleanup policy" phrase (line 3 of `topics.go`) to "applies the compacted topic configuration".

- [ ] **Step 7: Run the tests to verify they pass**

```bash
go build ./... && go test ./internal/topics/ -v
```

Expected: PASS for every test in the package, including `TestEnsure_SingleCreateRequest`, `TestEnsure_AlterConfigs`, `TestEnsure_AlterConfigs_NoCompactTopics`, `TestEnsure_AlterConfigs_ResourceError`, and `TestCompactConfigNames`.

Note: `TestEnsure_AlterConfigs_ResourceError` asserts only that the error contains `"c2"`, so the reworded wrap does not break it. If it fails, the wrap lost the topic name — fix the wrap, not the test.

- [ ] **Step 8: Commit**

```bash
git add services/atlas-kafka-precreate/internal/topics/topics.go services/atlas-kafka-precreate/internal/topics/topics_test.go
git commit -m "fix(kafka-precreate): apply segment-roll configs to compacted topics"
```

---

### Task 2: Anti-regression tests for the two-request agreement and the plain-topic negative

Two new tests that guard the defect *class* rather than this instance of it: that the create and alter bodies are the same set of pairs, and that plain topics are untouched by both. The current suite checks the plain-topic-not-in-alter half inside a loop over compacted resources only, where it can never fire.

### Files

- `services/atlas-kafka-precreate/internal/topics/topics_test.go` — add `TestEnsure_CompactConfigsMatchAcrossRequests` and `TestEnsure_PlainTopicsCarryNoConfig`; delete the dead `if res.ResourceName == "p1"` assertion inside `TestEnsure_AlterConfigs`'s compacted-resource loop (`:325-327`), which the new standalone test replaces
- `services/atlas-kafka-precreate/internal/topics/topics.go` — read-only; the tests assert against `compactTopicConfigs` only through the two projections

Module root: `services/atlas-kafka-precreate`.

Patterns to copy: `services/atlas-kafka-precreate/internal/topics/topics_test.go:282-306` (stub + `Ensure` + single-alter-call assertions) — both new tests use the same setup shape with a different topic mix.

**Interfaces:**
- Consumes: from Task 1 — `compactCreateEntries()`, `compactAlterConfigs()` (indirectly, via the request bodies `Ensure` produces).
- Produces: nothing consumed by later tasks.

- [ ] **Step 1: Write the failing tests**

**1a. `TestEnsure_CompactConfigsMatchAcrossRequests`** — not table-driven; one run with one plain and two compacted topics, asserted as a set comparison so it does not encode slice order. Setup copied from `topics_test.go:282-290`, with `createFn` returning `Errors: map[string]error{"p1": nil, "c1": nil, "c2": nil}` and `topicsIn := discover.Topics{Plain: []string{"p1"}, Compact: []string{"c1", "c2"}}`.

Assertions, in order:

1. Build `wantPairs := map[string]string{"cleanup.policy": "compact", "max.compaction.lag.ms": "600000", "segment.ms": "600000", "min.cleanable.dirty.ratio": "0.01"}` — literal values, not the constants. A test that reads the constant cannot catch someone changing the constant.
2. For each `cfg` in `stub.createCalls[0].Topics` whose `cfg.Topic` is `"c1"` or `"c2"`: collapse `cfg.ConfigEntries` into a `map[string]string` (fail on a duplicate `ConfigName`) and require it to equal `wantPairs`.
3. For each `res` in `stub.alterCalls[0].Resources`: collapse `res.Configs` into a `map[string]string` keyed by `Name` (fail on duplicate) and require it to equal `wantPairs`; additionally require every entry's `ConfigOperation == kafka.ConfigOperationSet`.
4. Fail if either loop matched zero topics/resources — a vacuous pass is the failure mode this test exists to prevent.

Use `reflect.DeepEqual` for the two map comparisons (add `"reflect"` to the test file's imports).

**1b. `TestEnsure_PlainTopicsCarryNoConfig`** — same setup, `topicsIn := discover.Topics{Plain: []string{"p1", "p2"}, Compact: []string{"c1"}}`, `createFn` returning `Errors: map[string]error{"p1": nil, "p2": nil, "c1": nil}`.

| assertion | expected |
|---|---|
| `stub.createCalls[0].Topics` entry with `Topic == "p1"` | `len(ConfigEntries) == 0` |
| `stub.createCalls[0].Topics` entry with `Topic == "p2"` | `len(ConfigEntries) == 0` |
| `stub.alterCalls[0].Resources` | exactly 1 resource, `ResourceName == "c1"` |
| any resource with `ResourceName` in `{"p1","p2"}` | none — `t.Errorf` if found |

Write the plain-name check as a standalone scan over `req.Resources` against a `map[string]struct{}{"p1": {}, "p2": {}}`, not nested inside a loop over compacted names.

**1c.** Delete these three lines from `TestEnsure_AlterConfigs`'s loop — the loop only ever iterates compacted resources, so the condition is unreachable and 1b now covers it properly:

```go
		if res.ResourceName == "p1" {
			t.Errorf("expected p1 to not appear in alter resources")
		}
```

- [ ] **Step 2: Run the tests to verify they pass**

```bash
go test ./internal/topics/ -run 'TestEnsure_CompactConfigsMatchAcrossRequests|TestEnsure_PlainTopicsCarryNoConfig' -v
```

Expected: PASS. These are anti-regression tests over Task 1's already-correct implementation, so green on first run is the correct outcome — there is no red step to stage.

- [ ] **Step 3: Prove the agreement test can fail**

The set-comparison test is worthless if it passes vacuously. Temporarily delete the `{name: "segment.ms", ...}` line from `compactTopicConfigs` in `topics.go` and re-run:

```bash
go test ./internal/topics/ -run TestEnsure_CompactConfigsMatchAcrossRequests -v
```

Expected: FAIL, reporting the create-side map missing `segment.ms`. Restore the line and re-run to confirm PASS before continuing. Do not commit the deletion.

- [ ] **Step 4: Run the full package suite**

```bash
go build ./... && go test ./... 
```

Expected: PASS across `internal/topics`, `internal/discover`, `internal/groups`, `internal/kafkaops`.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-kafka-precreate/internal/topics/topics_test.go
git commit -m "test(kafka-precreate): assert create and alter compact configs agree"
```

---

### Task 3: Alter-phase log field and README documentation

The operator-facing half: the alter log line names the configs it applied, and the README stops claiming compacted topics carry only `cleanup.policy`.

### Files

- `services/atlas-kafka-precreate/main.go` — the alter log block at `main.go:73-79`; add the `configs` field and change the message text
- `services/atlas-kafka-precreate/README.md` — the `COMMAND_TOPIC_*` / `EVENT_TOPIC_*` row (`README.md:21`), the Create/Alter phase bullet (`README.md:29-32`), and a new "Compacted topics" subsection
- `services/atlas-kafka-precreate/internal/topics/topics.go` — read-only; `CompactConfigNames()` from Task 1 is what `main.go` calls

Module root: `services/atlas-kafka-precreate`.

**Interfaces:**
- Consumes: `topics.CompactConfigNames() []string` from Task 1.
- Produces: nothing consumed by later tasks.

- [ ] **Step 1: Update the alter log line**

In `main.go`, replace the block at lines 73-79 with:

```go
	if len(t.Compact) > 0 {
		logrus.WithFields(logrus.Fields{
			"phase":   "alter",
			"topics":  len(t.Compact),
			"policy":  "compact",
			"configs": topics.CompactConfigNames(),
		}).Info("compacted topic configuration applied")
	}
```

The `phase`, `topics`, and `policy` fields are unchanged — nothing that parses this line loses a field. `topics` is already imported by `main.go` (it calls `topics.Ensure`); do not add an import.

- [ ] **Step 2: Verify it compiles and the suite still passes**

```bash
go build ./... && go test ./...
```

Expected: PASS.

- [ ] **Step 3: Update the README env-var row**

Replace the `Meaning` cell of the `COMMAND_TOPIC_*`, `EVENT_TOPIC_*` row at `README.md:21` so it names the full configuration instead of only `cleanup.policy`:

> Every variable with one of these prefixes and a non-empty value names a topic to create. Three specific `EVENT_TOPIC_*` variables (the configuration-projection topics) are created and converged with the compacted topic configuration — `cleanup.policy=compact`, `max.compaction.lag.ms=600000`, `segment.ms=600000`, `min.cleanable.dirty.ratio=0.01` (see [Compacted topics](#compacted-topics)); every other topic is created plain.

- [ ] **Step 4: Update the Create/Alter phase bullet**

Replace phase bullet 2 (`README.md:29-32`) with:

> 2. **Create/Alter** — create the full topic union in one `CreateTopics`
>    request, then apply the compacted topic configuration to the compacted
>    topics in one `IncrementalAlterConfigs` request. This phase always runs;
>    topics are created whether or not there is a group to seed. The alter
>    runs over the whole compacted set on every run, not just the topics this
>    run created, so a topic created by an earlier version of the tool
>    converges to the current configuration. `cleanup.policy=compact` on its
>    own is inert — the log cleaner never touches a partition's active
>    segment, so a topic whose segment never rolls is never cleaned;
>    `max.compaction.lag.ms` is what lowers the segment-roll deadline and
>    therefore what makes the policy mean anything.

- [ ] **Step 5: Add the "Compacted topics" subsection**

Insert a `## Compacted topics` section after the Phases list. Content, verbatim:

> ## Compacted topics
>
> The three configuration-projection topics are compacted: each carries the
> latest record per key, which is exactly the projection replay model. They
> are created with, and converged to, this configuration:
>
> | Config | Value | Purpose |
> | --- | --- | --- |
> | `cleanup.policy` | `compact` | Retain the latest record per key instead of deleting by age. |
> | `max.compaction.lag.ms` | `600000` (10 min) | Upper bound on how long a record may remain uncompacted. For a compacted topic the broker's effective segment-roll deadline is `min(segment.ms, max.compaction.lag.ms)`, so this lowers the roll deadline to 10 minutes — and the roll is what hands the cleaner a non-active segment to work on. |
> | `segment.ms` | `600000` (10 min) | The same deadline on the policy-independent knob. |
> | `min.cleanable.dirty.ratio` | `0.01` | Lets the cleaner select a segment whose dirty fraction is below the 0.5 default. |
>
> Which knob is doing the work, stated plainly so nobody deletes the wrong
> one: **`max.compaction.lag.ms` is load-bearing.** Verified against
> `apache/kafka:4.1.1`, a topic carrying `cleanup.policy=compact` and
> `max.compaction.lag.ms` alone — no `segment.ms` — rolled its segment on the
> next append past the lag and was compacted from 200 records over 3 keys
> down to 3. `segment.ms` at the same value is exactly redundant given the
> `min()` rule; it is set so the roll bound survives someone changing
> `cleanup.policy`, and so the cadence is readable from a `--describe`
> without knowing that rule. `min.cleanable.dirty.ratio` was not isolated by
> that experiment and the forced-cleaning path bypasses the ratio check
> anyway — it is cheap steady-state insurance, not a load-bearing knob.
>
> The roll is triggered **on append**, not by a timer: a quiescent compacted
> topic does not roll and does not churn segments. Index and timeindex files
> are preallocated for the active segment only and trimmed to their real size
> on roll, so the steady-state footprint is one active segment's preallocated
> pair plus a compacted tail of kilobytes.
>
> ### Verifying compaction on a live broker
>
> Log-start-offset is **not** the signal — compaction rewrites a segment in
> place, keeping surviving records at their original offsets, so a fully
> compacted partition can still report a log start offset of `0`. The correct
> signals, in decreasing sharpness:
>
> 1. `grep <topic> /var/lib/kafka/data/kafka/cleaner-offset-checkpoint` returns
>    a line — the direct statement that the cleaner has processed the partition.
> 2. The partition directory shows more than one `.log`, with the base-0 `.log`
>    far smaller than the uncompacted total.
> 3. A `--from-beginning` consume returns on the order of the key count rather
>    than the record count.

- [ ] **Step 6: Verify the README anchor resolves**

Read back `services/atlas-kafka-precreate/README.md` and confirm two things the
edits above must both have produced: the `## Compacted topics` heading exists
(Step 5), and the env-var row's `[Compacted topics](#compacted-topics)` link
(Step 3) points at it. A link with no heading is a broken anchor; a heading with
no link is a section nothing reaches.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-kafka-precreate/main.go services/atlas-kafka-precreate/README.md
git commit -m "docs(kafka-precreate): document the compacted topic configuration"
```

---

### Task 4: Full verification gate

### Files

- `tools/verify.sh` — read-only; the repo-wide gate
- `services/atlas-kafka-precreate/` — read-only unless the gate reports a failure to fix

- [ ] **Step 1: Run the flagless gate**

From the worktree root:

```bash
tools/verify.sh
```

Expected: exit 0. `--quick` / `--no-docker` do **not** satisfy this step — they skip the bake and `-race`.

- [ ] **Step 2: Fix any failure and re-run**

If the gate fails, fix the cause in `services/atlas-kafka-precreate/` and re-run the flagless gate until it exits 0. Do not report the branch as done from a flagged or partial run.

- [ ] **Step 3: Confirm the acceptance criteria**

Check each against the code as committed, not from memory:

- [ ] `topics.go` declares `max.compaction.lag.ms`, `segment.ms`, and `min.cleanable.dirty.ratio` as named constants with values `600000`, `600000`, `0.01`.
- [ ] Both request builders derive from `compactTopicConfigs`; `TestEnsure_CompactConfigsMatchAcrossRequests` asserts the two bodies agree.
- [ ] The alter path still uses `ConfigOperationSet`, still runs unconditionally over the whole compacted set, still issues exactly one request.
- [ ] Plain topics carry no `ConfigEntries` and appear in no alter resource, asserted standalone by `TestEnsure_PlainTopicsCarryNoConfig`.
- [ ] A per-resource alter error remains fatal and names the topic.
- [ ] README and in-code comments state the full configuration and why `cleanup.policy` alone is inert.

**Post-deploy verification is an operator step, not part of this branch.** After the `atlas-main` wave-0 Job re-runs: `kafka-configs.sh --entity-type topics --entity-name EVENT_TOPIC_CONFIGURATION_ENVIRONMENT_STATUS-main --describe` shows all four configs, `grep EVENT_TOPIC_CONFIGURATION_ENVIRONMENT_STATUS-main /var/lib/kafka/data/kafka/cleaner-offset-checkpoint` returns a line (it returns nothing today), and the partition directory's base-0 `.log` is far below 9.8 MB.
