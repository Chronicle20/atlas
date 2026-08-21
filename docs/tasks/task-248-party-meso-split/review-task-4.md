# Review: Task 4 — atlas-character credits each MESO_AWARDED recipient

Range reviewed: `2ef8dd600..23d12c455` (single commit `23d12c455`), module
`services/atlas-character/atlas.com/character`.

Inputs: `.superpowers/sdd/plan/task-4-brief.md`, `.superpowers/sdd/plan/task-4-report.md`,
`.superpowers/sdd/plan/review-2ef8dd600..23d12c455.diff`.

## Scope confirmed

The diff touches exactly the five files the brief names:
`kafka/message/drop/kafka.go`, `kafka/consumer/drop/consumer.go`,
`kafka/consumer/drop/consumer_test.go` (new), `character/processor.go`,
`character/meso_award_test.go` (new). Single commit, message matches the
brief's Step 8 text. `git diff --stat` matches the review package's stat
block exactly. No `go.mod`/`go.sum` changes. Scope matches the brief.

## Requirement-by-requirement

### 1. `AwardPickedUpMeso` signature and interface replacement

`character/processor.go:130` (interface) and the implementation (post-line
928) both read:

```go
AwardPickedUpMeso(transactionId uuid.UUID, f field.Model, characterId uint32, dropId uint32, meso uint32, picker bool) error
```

matching the brief exactly. PASS.

### 2. `AttemptMesoPickUp` fully removed

```
$ grep -rn AttemptMesoPickUp services/atlas-character/atlas.com/character
character/processor.go:936:// Deliberate asymmetry, and the reason this replaced AttemptMesoPickUp: ...
```

Only remaining occurrence is the explanatory comment; no interface entry, no
implementation, no caller. `go build ./...` (module root) succeeds. PASS.

### 3. Credit runs in its own transaction, emits `MESO_CHANGED` + `STAT_CHANGED` via outbox

`AwardPickedUpMeso` wraps the credit in `database.ExecuteTransaction(...)`
and, inside the same transaction, calls
`message.Emit(outbox.EmitProvider(...))` to `buf.Put` a `MESO_CHANGED` event
then a `STAT_CHANGED` event on `character2.EnvEventTopicCharacterStatus`.
`mesoChangedStatusEventProvider(transactionId, characterId, c.WorldId(), int32(meso), dropId, actorTypeDrop, true)`
— confirmed against its declaration at `character/producer.go:151`
(`func mesoChangedStatusEventProvider(transactionId uuid.UUID, characterId uint32, worldId world.Id, amount int32, actorId uint32, actorType string, showEffect bool)`)
— passes `dropId` as `actorId` and `true` as `showEffect`, matching the
test's expectation `ActorId: uint32(4242)`, `ShowEffect: true`. PASS,
verified by call-site/signature cross-reference, not by test claim alone.

### 4. Only the `Picker: true` award completes the pickup

```go
if picker {
    if err := drop.NewProcessor(p.l, p.ctx).RequestPickUp(f, dropId, characterId); err != nil { ... }
}
```

Unconditional on `picker`'s value alone, not on the credit outcome. PASS.

### 5. `Amount == 0` skips the transaction entirely, but a picker's award still completes

```go
var txErr error
if meso > 0 {
    txErr = database.ExecuteTransaction(...)
    ...
}
if picker { ... }
return txErr
```

When `meso == 0`, the `if meso > 0` block never runs — no `GetById`, no
`dynamicUpdate`, no outbox emit — and `picker` is evaluated independently
below. `TestAwardPickedUpMeso_ZeroAmountRunsNoTransactionButCompletesThePickUp`
asserts balance unchanged, outbox row count unchanged, and exactly one
pick-up command. PASS.

### 6. Overflow above `math.MaxInt32` rejected with `ErrMesoOverflow` before the `uint32` balance guard

```go
if meso > math.MaxInt32 {
    ...
    return ErrMesoOverflow
}
if meso > (math.MaxUint32 - c.Meso()) {
    ...
    return ErrMesoOverflow
}
```

The `int32` guard precedes the `uint32` balance guard exactly as required.
`TestAwardPickedUpMeso_AmountAboveInt32IsRejected` (amount `2147483648`,
starting balance 0) and
`TestAwardPickedUpMeso_OverflowSkipsTheCreditButStillCompletesThePickUp`
(amount `math.MaxInt32`, balance already at `3147483647` after a prior
`RequestChangeMeso`) both assert `ErrMesoOverflow`, unchanged balance,
unchanged outbox row count, and exactly one pick-up command (picker still
completes despite the credit error — the old-bug regression check). PASS.

### 7. Old bug (early return on `txErr != nil` before `RequestPickUp`) not reproduced

Prior code:
```go
if txErr != nil {
    return txErr
}
return drop.NewProcessor(p.l, p.ctx).RequestPickUp(field, dropId, characterId)
```
New code separates the two: `txErr` is captured, logged, but the `if picker`
block always runs afterward regardless of `txErr`, and only the final
`return txErr` carries the credit outcome back to the caller (for logging by
the consumer, per the brief). This is exactly the fix. PASS, and covered by
the two overflow tests above (picker still completes despite a returned
error).

### 8. `RESERVED` unchanged; meso no longer credited on it

`ReservedStatusEventBody` in `kafka/message/drop/kafka.go` is byte-identical
to its pre-diff shape (diff shows no changes to that struct). The consumer's
`handleDropReservation` function — the only place that read
`ReservedStatusEventBody.Meso` and invoked `AttemptMesoPickUp` — is deleted
outright; `InitHandlers` now registers `handleMesoAwarded` on the same topic
in its place. `grep -rn "ReservedStatusEventBody\|StatusEventTypeReserved"`
across the module now returns only the type declaration, the const
declaration, and the new regression test's literal use of the type-name
string for the type guard — no code path reads `.Meso` off a `RESERVED`
event anymore. PASS.

### 9. Cross-service seam — `MESO_AWARDED` envelope and body fields

Compared field-for-field against the producer
(`services/atlas-drops/atlas.com/drops/kafka/message/drop/kafka.go`):

Envelope (`StatusEvent[E]`), both sides:
```
TransactionId uuid.UUID  `json:"transactionId"`
WorldId       world.Id   `json:"worldId"`
ChannelId     channel.Id `json:"channelId"`
MapId         _map.Id    `json:"mapId"`
Instance      uuid.UUID  `json:"instance"`
DropId        uint32     `json:"dropId"`
Type          string     `json:"type"`
Body          E          `json:"body"`
```
Identical field names, types, and JSON tags, in the same order, on both
sides — including the newly added `TransactionId`.

Body (`MesoAwardedStatusEventBody` / `StatusEventMesoAwardedBody`), both
sides:
```
CharacterId uint32 `json:"characterId"`
Amount      uint32 `json:"amount"`
Picker      bool   `json:"picker"`
```
Identical. `StatusEventTypeMesoAwarded = "MESO_AWARDED"` matches on both
sides. No import of `atlas-drops` packages from `atlas-character` — the
mirror is a hand-copied local type, per the module-boundary constraint.
PASS — verified by direct file comparison, not by claim.

The producer side (`drop/producer.go`, `drop/processor.go`, `drop/split.go`
in atlas-drops) already emits `MESO_AWARDED` with `Picker` set — that logic
is Task 3's scope and out of this review's surface; confirmed only that the
symbols the mirror depends on exist there.

### 10. `AwardPickedUpMeso` interface entry is the only consumer of the removed function; no dangling callers

`grep -rn AttemptMesoPickUp` (above) confirms this; `go build ./...` and
`go vet ./...` both succeed clean at the module root.

### 11. Repo conventions

- `actorTypeDrop = "DROP"` is a local const next to the function, matching
  the plan's resolved-ambiguity note (`libs/atlas-constants/` has no
  actor-type home; `"SYSTEM"`/`"CHARACTER"`/`"ITEM"` remain bare literals
  elsewhere). Accepted per the task's own "known and accepted" note.
- Test file uses Builder construction (`field.NewBuilder(...).SetInstance(...).Build()`),
  and the pre-existing `outboxTestDb`/`createTestCharacter`/`testTenant`/`testLogger`
  helpers rather than a new `*_testhelpers.go` file.
- Behavioral-derivation comments cite repo source/brief reasoning
  (`processor.go`'s "Deliberate asymmetry" comment references the actual old
  code path), no Cosmic citations found.
- No `// TODO`, no stub bodies, no unimplemented status responses.

### 12. Test honesty

Each of the six `meso_award_test.go` tests and the one
`consumer_test.go` test asserts a concrete, code-path-specific outcome
(balance value, outbox row delta, specific event ordering, specific error
identity, pick-up command presence/absence) rather than a tautology. The
`TestHandleMesoAwarded_IgnoresNonMesoAwardedEvents` test's use of a
`nil *gorm.DB` is a real trap for the type guard — if the guard were removed
or miswired, the test would panic (not merely fail an assertion), which is a
stronger signal than an ordinary assertion failure. Implementer's report
honestly discloses that the RED step was not captured as an isolated
failing-test transcript (steps 3–5 landed in the same edit pass as the
tests); this is a process-honesty gap in the TDD *evidence*, not a defect in
the shipped code — the six tests do exercise real, distinguishable branches
(confirmed above by reading the branches directly), so I do not treat this
as a blocking finding, only a note.

## Verification commands run in this review

```
cd services/atlas-character/atlas.com/character && go build ./...   # clean
cd services/atlas-character/atlas.com/character && go vet ./...     # clean
```
Per instructions, the module's own test suite (already run and reported
green by the implementer) was not re-run.

## Not evaluable

- Whether the atlas-drops producer (Task 3) actually sets `Picker: true` for
  exactly one recipient at the split-computation level is out of this
  task's surface (no file under `atlas-drops` is touched by this diff); it
  was reviewed, or should be reviewed, under Task 3's own review artifact.
- `tools/verify.sh` (flagless, full repo) was not run in this review; that
  is the controller's gate per the brief's own "Verification" section, not
  a substitute for this diff-scoped review.

## Findings

None blocking. One non-blocking note:

- Non-blocking: the implementer's report documents that the RED (failing
  test) step for TDD was not captured as an isolated transcript — the
  interface/implementation replacement and the new tests were authored in
  the same edit pass, so "AwardPickedUpMeso undefined" was never actually
  observed as a live test failure, only inferred from reading the pre-edit
  source. This does not affect the shipped code's correctness (verified
  independently above) but is a process gap worth naming for the task's
  overall TDD-discipline quality bar.

## Verdicts

- Spec compliance: every requirement in the brief's behavioral contract,
  the cross-service seam, and the global constraints is met, verified
  against source rather than accepted on claim. No blocking findings.
- Task quality: implementation and tests are correct, well-documented
  (comments explain the deliberate asymmetry and cite the actual prior
  bug), and conventions are followed. The one ding is procedural (TDD
  RED-step evidence), not a code defect.
