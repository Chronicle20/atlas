# Review: Task 1 — Shared compact-config declaration and its two projections

Range reviewed: `b93ab7add..d673eed8f` (single commit `d673eed8f`).
Scope: `services/atlas-kafka-precreate/internal/topics/topics.go`,
`services/atlas-kafka-precreate/internal/topics/topics_test.go`. No other
files touched (`git diff --stat` confirms exactly these two).

## Checklist against the brief / binding constraints

- **Four config constants, verbatim values** — PASS. `topics.go:23,36,47,55`:
  `compactCleanupPolicy = "compact"`, `compactMaxCompactionLagMs = "600000"`,
  `compactSegmentMs = "600000"`, `compactMinCleanableDirtyRatio = "0.01"`.
  All Go `string` constants, none inlined at a call site — every reference in
  `compactTopicConfigs` (`topics.go:74-79`) is by name.

- **Declaration order load-bearing** — PASS. `compactTopicConfigs`
  (`topics.go:74-79`) is `cleanup.policy, max.compaction.lag.ms, segment.ms,
  min.cleanable.dirty.ratio`, matching the brief table and asserted by
  `TestCompactConfigNames` (`topics_test.go` new test) and the extended
  `TestEnsure_SingleCreateRequest` / `TestEnsure_AlterConfigs` element-wise
  comparisons.

- **Single source of truth, two pure projections** — PASS.
  `compactCreateEntries()` (`topics.go:87-93`) and `compactAlterConfigs()`
  (`topics.go:98-108`) both range over `compactTopicConfigs`; no config
  appears in one function's literal but not the other's, because there is no
  separate literal — a config missing from one projection would require
  editing the shared range, which both functions share. Architecture
  requirement met.

- **Fresh slice per call, no shared backing array** — PASS. Both projections
  use `make([]T, len(compactTopicConfigs))` and populate their own slice
  (`topics.go:88`, `99`) — independent allocations, no shared backing array
  across topics or across the two projections.

- **`Ensure` control flow unchanged** — PASS, verified by direct comparison
  against pre-change shape (report + diff): unconditional alter over the
  entire `t.Compact` slice (`topics.go:205-212`, `for i, name := range
  t.Compact`), still exactly one `IncrementalAlterConfigsRequest`
  (`topics.go:214`), still `kafka.ConfigOperationSet` (`topics.go:104`), still
  the `len(t.Compact) == 0` short-circuit (`topics.go:182-184`) before
  building resources, plain-topic loop still emits no `ConfigEntries`
  (`topics.go:145-151`).

- **No swallowed alter errors, no allowlist, no partial-application
  fallback** — PASS. Transport error path (`topics.go:214-217`) returns
  `EnsureResult{}, fmt.Errorf(...)` unconditionally. Per-resource error path
  (`topics.go:219-227`) appends every `res.Error != nil` into `alterFatal`,
  joins with `errors.Join`, and returns fatal if any are present — no
  filtering by config name, no "some topics succeeded, continue anyway"
  branch.

- **No stubs / TODOs / deferred work** — PASS, confirmed by reading the full
  diff; no `// TODO` or placeholder markers introduced.

- **Tests run against `stubClient`, no broker** — PASS. `go test
  ./internal/topics/ -v` run directly in this review (module root
  `services/atlas-kafka-precreate`): full package passes, including
  `TestEnsure_SingleCreateRequest`, `TestEnsure_AlterConfigs`,
  `TestEnsure_AlterConfigs_NoCompactTopics`,
  `TestEnsure_AlterConfigs_ResourceError`, `TestCompactConfigNames`. `go
  build ./...` also clean.

- **Scope: nothing outside `internal/topics/` touched** — PASS. `git diff
  --stat b93ab7add..d673eed8f` shows only the two files under
  `internal/topics/`.

## Non-blocking finding

- `topics.go:130` — `Ensure`'s own doc comment still reads "applies
  cleanup.policy=compact to every topic in t.Compact," unchanged from before
  this commit, and is now inaccurate: the alter now applies four configs, not
  one. The brief's Step 6 named exactly two comment edits (package doc line 3,
  and the `IncrementalAlterConfigs` rationale block at `topics.go:186-204`)
  and did not list this one; the implementer flagged it explicitly in the
  self-review rather than silently leaving it stale. Not blocking — correctly
  scoped to the brief's explicit edit list, but a real drift a future reader
  could be misled by. Worth sweeping in Task 3 (operator docs) or a
  follow-up, as the implementer suggested.

## Verdict

APPROVED_WITH_FINDINGS — the one finding above is documentation drift the
brief itself did not put in scope for this commit, not a functional or
architectural defect.
