# Review: Task 10 — Carry `TransactionId` on the seed status envelope

Commit range: `ad1557fb9..b9581c6` (single commit `b9581c675`,
`feat(atlas-character-factory): carry transactionId on seed status events`).
Brief: `.superpowers/sdd/plan/task-10-brief.md` (Controller addendum).
Report: `.superpowers/sdd/plan/task-10-report.md`.

## Scope confirmed

`git diff --stat` matches the brief's declared file list exactly:

```
.../kafka/consumer/saga/consumer.go                |   4 +-
.../kafka/consumer/saga/consumer_test.go           | 159 +++++++++++++++++++++
.../character-factory/kafka/message/seed/kafka.go  |   8 +-
.../kafka/producer/seed/producer.go                |  14 +-
4 files changed, 175 insertions(+), 10 deletions(-)
```

No changes outside `services/atlas-character-factory`. Single commit, matches
the stated range. No scope mismatch.

## Findings

### 1. Requirement compliance — PASS

- `StatusEvent[E]` gains `TransactionId string` between `AccountId` and
  `Type`, exactly as specified
  (`services/atlas-character-factory/.../kafka/message/seed/kafka.go:9-16`).
- Tag is spelled correctly: `` `json:"transactionId,omitempty"` `` — verified
  byte-for-byte with `cat -A`, no stray space, no `omitEmpty` typo
  (kafka.go:14).
- Both providers gained the `transactionId string` parameter and set it on
  the struct literal (`producer.go:12-20,28-36`).
- Both `consumer.go` call sites pass `e.TransactionId.String()`
  (`consumer.go:67,120`).

### 2. Cross-service seam — atlas-login untouched — PASS

- `git status --porcelain services/atlas-login/` and
  `git diff --stat ad1557fb9..b9581c6 -- services/atlas-login/` both return
  empty. Confirmed directly, not taken from the report.
- `atlas-login`'s own `seed.StatusEvent` copy
  (`services/atlas-login/atlas.com/login/kafka/message/seed/kafka.go:9-13`)
  still has only `AccountId`, `Type`, `Body` — field names and JSON tags
  identical to the pre-change factory struct, so Go's default
  ignore-unknown-fields decoding drops the new `transactionId` key silently.
  Nothing is mis-bound (no field named similarly that could collide).
- `cd services/atlas-login/atlas.com/login && go build ./... && go test
  ./kafka/consumer/seed/... -v` — build succeeds, package reports
  `[no test files]` (there was no coverage there before this change either,
  so this is a build-clean confirmation, not new evidence of behavior).
- Swept for other seed-status consumers: `grep -rl "EVENT_TOPIC_SEED_STATUS"
  services/` returns only `atlas-character-factory` (producer + this commit's
  consumer/test) and `atlas-login` (consumer + docs). No third consumer
  exists today, so the "trace into every consumer" requirement is satisfied
  by inspecting exactly these two.

### 3. Test honesty — PASS

- `consumer_test.go` is genuinely new (the bridge had zero test coverage
  before). Confirmed the pre-commit `kafka.go` has no `TransactionId` field
  (`git show ad1557fb9:.../kafka.go | grep -c TransactionId` → `0`), so the
  test file's struct literals (`seedMessage.StatusEvent[...]{TransactionId:
  ...}` at consumer_test.go:54,84) and the 3-arg provider calls the bridge
  makes would not compile against the pre-edit code — the report's claim of
  a compile-time RED state is accurate.
- `TestStatusEventMarshalsTransactionId` (consumer_test.go:142-159) proves
  BOTH halves: `"transactionId":"tx-1"` present when set, and
  `require.NotContains(t, string(raw), "transactionId")` when empty — non-
  vacuous, asserts the omitempty contract directly rather than merely
  compiling.
- `TestCompletedBridgeCarriesTransactionId` and
  `TestFailedBridgeCarriesTransactionId` drive the real handler functions
  (`handleSagaCompletedEvent`, `handleSagaFailedEvent`) with a captured
  producer (`producertest.InstallCapturing()`, a pre-existing shared library
  at `libs/atlas-kafka/producer/producertest/producertest.go`, untouched by
  this commit — confirmed via `git diff --stat` on that path returning
  nothing), decode the emitted Kafka message, and assert `ev.TransactionId ==
  transactionId.String()`. This is not a fixture that never fires: `require.Len(t,
  msgs, 1)` before decoding proves the producer path executed.
- `TestNonCharacterCreationSagaStillDropped` drives both handlers with
  `SagaType = "inventory_transaction"` and asserts
  `require.Empty(t, emitted.Messages(...))` for both — pins the filter
  correctly.
- Ran `go test ./kafka/... -v` fresh: all four tests pass. Ran
  `go build ./... && go test ./...` for the whole module: all packages
  build/pass (or report `[no test files]` where expected).

### 4. Saga-type filter unchanged — PASS

Diffed `consumer.go` line-by-line: the only changes are the two provider
call-site argument additions. `handleSagaCompletedEvent`'s
`e.Body.SagaType != string(sharedsaga.CharacterCreation)` early-return and
`handleSagaFailedEvent`'s equivalent check are byte-identical to before this
commit; no restructuring of the surrounding dispatch.

### 5. `e.TransactionId.String()` is the right, consistent source — PASS

- `saga.StatusEvent[E].TransactionId` is typed `uuid.UUID` (confirmed via
  `grep -n TransactionId services/atlas-character-factory/.../kafka/message/saga/*.go`
  → `TransactionId uuid.UUID`), so `.String()` is the correct conversion to
  the seed envelope's plain-string field.
- Both the completed path (consumer.go:67) and the failed path
  (consumer.go:120) read from the same `e.TransactionId` on the same
  incoming saga event — no divergent source between the two paths.

### 6. Type agreement with Task 9's channel client — PASS

`services/atlas-channel/atlas.com/channel/character/factory/rest.go:55` —
`TransactionId string \`json:"transactionId"\`` — both sides are plain
`string` on the wire (the channel-side field is exported directly from JSON,
the factory-side field is a `uuid.UUID.String()` written into a plain
`string` field). No type disagreement.

### 7. Repo conventions — PASS

- Builder-pattern / shared fixtures: uses the repo's existing
  `producertest.InstallCapturing()` capture harness rather than inventing a
  local mock producer.
- No `*_testhelpers.go` file was added; `consumer_test.go` is a standard
  `_test.go` file.
- No stub, placeholder, or TODO in the diff.

## Not evaluable

None. The full review surface (four changed files, the one upstream consumer,
the one downstream client reference) was traced and verified directly against
repo state and command output, within the scope of this unit.

## Verdict rationale

Every controller-addendum concern — the backward-compatibility proof, the
non-vacuous marshal test, the unchanged filter, the consistent transaction-id
source, the type agreement with Task 9 — is independently confirmed against
actual file contents and command output, not merely asserted by the report.
No defects found.
