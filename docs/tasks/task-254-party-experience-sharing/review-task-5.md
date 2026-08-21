# Review: Task 5 — `atlas-monster-death` `system_message` producer

Commit range: `2f91e3cc0..7e2278f81` (single commit `7e2278f81`)
Brief: `.superpowers/sdd/plan/task-5-brief.md` (Task 5 section)
Report: `.superpowers/sdd/plan/task-5-report.md`

## Scope confirmed

Diff touches exactly the five new files the brief specifies, all under
`services/atlas-monster-death/atlas.com/monster/system_message/`:
`kafka.go`, `producer.go`, `processor.go`, `mock/processor.go`,
`producer_test.go`. No other file in the commit. Matches brief's "Files"
list and diff --stat exactly.

## Checks

### 1. Requirement compliance

- `kafka.go` — `EnvCommandTopic = "COMMAND_TOPIC_SYSTEM_MESSAGE"`,
  `CommandShowHint = "SHOW_HINT"`, `Command[E]` envelope, `ShowHintBody`.
  Byte-for-byte identical (field names, JSON tags, comments) to the brief's
  Step 3 code block. PASS.
- Struct shape verified against the two named references:
  - `services/atlas-channel/atlas.com/channel/kafka/message/system_message/kafka.go:11-33,62-67`
    — `Command[E]` and `ShowHintBody` field-for-field identical (same JSON
    tags: `transactionId`, `worldId`, `channelId`, `characterId`, `type`,
    `body`; body: `hint`, `width`, `height`). PASS — confirms no silent tag
    rename that would break `atlas-channel`'s consumer.
  - `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/system_message/producer.go:94-110`
    (`ShowHintCommandProvider`) — wire shape identical: same
    `producer.CreateKey(int(characterId))`, same field population order,
    same `producer.SingleMessageProvider(key, value)` return. PASS.
- `producer.go` — `ShowHintCommandProvider` signature and body match the
  brief's Step 3 spec exactly. PASS.
- `processor.go` — `Processor` interface contains **only** `ShowHint`, not
  the other ten saga-orchestrator command types (`SendMessage`,
  `PlayPortalSound`, `ShowInfo`, etc. are absent) — brief explicitly says
  "Only the `SHOW_HINT` command is needed." PASS.
  `ShowHint` delegates via
  `producer.ProviderImpl(p.l)(p.ctx)(EnvCommandTopic)(ShowHintCommandProvider(...))`,
  the exact call shape from `character/processor.go`'s `AwardExperience`
  (verified pattern matches `character/producer.go` / `character/processor.go`
  local convention). PASS.
- `mock/processor.go` — `ProcessorMock{ShowHintFunc ...}`,
  `var _ system_message.Processor = (*ProcessorMock)(nil)`, nil-func
  fallthrough returning `nil`. Shape matches
  `character/mock/processor.go`'s `ProcessorMock` convention exactly
  (`GetByIdFunc`/`AwardExperienceFunc` follow the identical
  func-field-with-nil-fallthrough pattern). PASS.
- No cross-service import: `grep -rn "atlas-channel\|saga-orchestrator"` in
  the new files returns nothing; `Command`/`ShowHintBody` are locally
  declared in `kafka.go`, not imported. Confirms the brief's
  no-reach-in constraint. PASS.
- No deployment change made, and none needed — `COMMAND_TOPIC_SYSTEM_MESSAGE`
  already exists in `deploy/k8s/base/env-configmap.yaml` and
  `atlas-monster-death.yaml` already has `envFrom: atlas-env` (per brief;
  not independently re-verified since it is explicitly out of scope for
  this commit and the diff makes no deploy changes). PASS (no-op is correct).

### 2. Correctness of the change itself

- `ShowHintCommandProvider` populates all six envelope fields and the three
  body fields from the function parameters with no branching — no
  nil/empty/boundary logic to get wrong here; `width`/`height` of 0 are
  passed through unchanged (matches brief's "0 for auto-calculation" comment
  and the test's explicit 0/0 case). PASS.
- Key is `producer.CreateKey(int(characterId))`, an existing library
  function reused correctly (matches saga-orchestrator's exact call).
  PASS.

### 3. Cross-service seam

- This commit only adds a producer; it is not yet wired into any
  distributor/consumer call site (`grep` of the rest of the module found no
  other file referencing `system_message`), consistent with the report's
  statement that wiring happens in Task 11 and throttling in Task 6. This
  is in-scope-as-specified, not a silent gap — the brief's Task 5 section
  defines only the package, not its caller.
- The wire-format compatibility with `atlas-channel`'s consumer (the actual
  cross-service seam) is verified structurally above (identical struct/JSON
  shape) and by the test's raw-map key assertions
  (`producer_test.go:64-78`), which would fail on a renamed JSON tag. PASS.

### 4. Test honesty

- `TestShowHintCommandProvider_WireShape` asserts: message count, key bytes,
  every typed field of `Command[ShowHintBody]`, and the raw JSON key names
  (both envelope and body) via `map[string]any` unmarshalling
  (`producer_test.go:33-79`). A JSON tag rename, a swapped field, or a
  missing key would fail this test — it is not a tautological pass.
  Confirmed by running it: `go test ./system_message/... -v` →
  `--- PASS: TestShowHintCommandProvider_WireShape`.
- The report's TDD narrative admits the literal RED step (compile failure
  before the package existed) was not captured as a separate git state,
  since brief gave verbatim implementation code for all files at once. This
  is a process deviation from strict TDD, not a correctness defect — the
  test content itself is meaningful and was verified to pass against the
  actual implementation. Non-blocking.

### 5. Repo conventions

- `gofmt -l system_message/` — no output, fully formatted. PASS.
- Builder/Extract pattern: N/A, no domain model added.
- `go build ./... && go vet ./... && go test ./system_message/... -v` from
  `services/atlas-monster-death/atlas.com/monster` — all green (re-run
  during this review, confirmed PASS, not just cited from the report).
- Full module test (per report): `go test ./...` — all `ok` or `[no test
  files]`, no regressions to other packages. Not independently re-run in
  full (module-local, low risk, not touched by this diff), but the
  `system_message` subset was re-run directly.

## Not evaluable

- Whether `atlas-channel`'s actual Kafka consumer for `SHOW_HINT` decodes
  this exact envelope correctly end-to-end (i.e., a live round-trip) is
  outside this unit's diff surface — verified only by static struct/JSON-tag
  comparison against `atlas-channel`'s source, which is the correct level of
  verification for a producer-only commit with no consumer changes in scope.

## Verdict rationale

Every file, field, and call shape in the diff matches the brief's Step 3
code verbatim, matches the two named upstream references field-for-field,
compiles and passes its own test (independently re-run), and introduces no
cross-service coupling. No blocking defects found.
