# Review: bug-shadow-buffs-dead-code (commit 037d1a7fa)

## Scope

Commit `037d1a7fa` only:

- `services/atlas-channel/atlas.com/channel/character/snapshot/processor.go` (+22/-2)
- `services/atlas-channel/atlas.com/channel/character/snapshot/shadow_test.go` (+62)
- `services/atlas-channel/atlas.com/channel/kafka/consumer/buff/consumer.go` (+28)
- `services/atlas-channel/atlas.com/channel/kafka/consumer/buff/consumer_test.go` (+79)

Read-only reference pulled in for correctness because the brief says
correctness depends on it: `services/atlas-buffs/atlas.com/buffs/character/processor.go`
(UpdateStatValue, `~L229-256`) and its producer
(`services/atlas-buffs/atlas.com/buffs/character/producer.go`,
`statUpdatedStatusEventProvider`), plus
`services/atlas-channel/atlas.com/channel/character/snapshot/registry.go`
(`View`, `ComposedIfValid`, `InvalidateBuffs`, `filterActive`/`Expired()`
semantics) and `shadow.go` (`compareProjection`'s buff branch).

Scope matches the brief exactly — both parts of the two-part fix are present,
plus tests for each.

## Part 1 — threading `servedBuffs` into `maybeShadow`

`processor.go:61-70` (fast path, `ComposedIfValid` hit) and `:159-172` (slow
path, `fullHit`) both now read `r.View(...)`/`v` and pass
`filterActive(buffs)` when `BuffsValid`, else `nil`. `nil` correctly remains
the "skip the component" signal per `shadow.go:130` and
`compareProjection`'s `snapBuffs != nil` gate (`shadow.go:139`) — matches
controller ruling R9 as described in the brief.

`filterActive` (`processor.go:206-213`) filters on `b.Expired()`, the same
predicate `GetBuffs()` uses at `processor.go:184`/`195`. So the shadow
sample's "served buffs" definition matches what a same-moment `GetBuffs()`
call would actually hand a caller — the REST-side comparison expectation
described in scrutiny point 2 holds.

**TOCTOU window (fast path only).** `processor.go:57` calls
`r.ComposedIfValid(...)`, which under its own lock requires
`e.buffsValid` to build/return the composed model but does not return the
buffs themselves (buffs are not part of `character.Model`'s composition —
`composeLocked`, `registry.go:166-174`, never touches buffs). The fix then
takes a *second*, independent lock via `r.View(...)` (`processor.go:67`) to
read the buffs actually served to the shadow sample. Between these two lock
acquisitions a concurrent event (e.g. the new `STAT_UPDATED` handler calling
`InvalidateBuffs`, or `RemoveBuff`/`UpsertBuff`) can mutate the buffs set.
Consequences, evaluated against the brief's specific worry (false-positive
divergence / shadow noise):

- If the buffs component is invalidated in the gap, `bv.BuffsValid` is
  false and `servedBuffs` becomes `nil` — this only *suppresses* a sample,
  it cannot manufacture a divergence.
- If a buff is added/removed in the gap without invalidation (there is no
  such event today — `UpsertBuff`/`RemoveBuff` both keep `buffsValid` true
  but change the *set*), `servedBuffs` reflects buffs "as of `View()`" that
  may differ from what `ComposedIfValid` observed as valid a few
  instructions earlier. Because `shadowCompare` re-fetches REST
  asynchronously afterward anyway (there is no atomic "this is what I
  served" snapshot even for core/inv/skills — `shadow.go:79-98` fetches
  fresh REST well after the sample was taken), this narrow gap is
  consistent with — not worse than — the staleness the shadow verifier
  already tolerates by design (position tolerance banding exists for
  exactly this class of drift). It is a real, non-atomic read, but it does
  not introduce a new class of false-positive that the existing REST-fetch
  lag doesn't already permit, and the two lock acquisitions are adjacent
  with no I/O between them, so the window is a handful of CPU
  instructions, not blocking work. Non-blocking finding, noted for the
  record since the brief flagged it for scrutiny.

## Part 2 — `STAT_UPDATED` consumer arm

`consumer.go:685-697` (`handleSnapshotBuffStatUpdated`) follows the exact
shape of the sibling `handleSnapshotBuffApplied`/`handleSnapshotBuffExpired`
handlers immediately above it (self-filter on `e.Type`, `sc.IsWorld` tenant
gate, then a single registry call) and is wired into `InitHandlers` at
`consumer.go:100-104` with the same `rf(...)` / `handles = append(...)`
pattern as every other handler in that function.

**Cross-service contract, verified byte-for-byte:**

- Status string: `atlas-buffs/character/producer.go:117` emits
  `character2.EventStatusTypeStatUpdated` = `"STAT_UPDATED"`
  (`atlas-buffs/kafka/message/character/kafka.go:133`); atlas-channel's
  `buff2.EventStatusTypeStatUpdated` (`kafka/message/buff/kafka.go:97`) is
  the same literal `"STAT_UPDATED"`.
- Payload shape: `atlas-buffs`'s `StatUpdatedStatusEventBody` producer struct
  (`producer.go:118-125`: `SourceId, Level, Duration, Changes, CreatedAt,
  ExpiresAt`) matches atlas-channel's `buff2.StatUpdatedStatusEventBody`
  field-for-field (`kafka.go:132-139`), including the deliberate absence of
  `NoExpiry` on both sides (already-existing type, not touched by this
  commit — see `handleStatusEventStatUpdated`'s comment at
  `consumer.go:238-242`, which independently documents the same omission).
- Tenant derivation: `tenant.MustFromContext(ctx)` + `sc.IsWorld(t,
  e.WorldId)` — identical to `handleSnapshotBuffApplied`/`Expired` and to
  the pre-existing `handleStatusEventStatUpdated`, all fed by the same
  consumer-group header parsers (`InitConsumers`,
  `consumer.SetHeaderParsers(..., consumer.TenantHeaderParser, ...)`).

The new handler correctly invalidates rather than applies in place
(`snapshot.GetRegistry().InvalidateBuffs(t, e.CharacterId)`,
`registry.go:636-641`: bumps `buffsGen`, sets `buffsValid = false`), per the
brief's explicit decision.

## Test honesty (both parts)

Verified directly by reverting each production file to its pre-fix content
in the worktree and re-running the new tests:

- `TestShadow_BuffsDivergenceFires` against pre-fix `processor.go`:
  `shadow_test.go:148: shadow divergence must record the buffs component
  once: before=0 after=0` — FAILS pre-fix, confirming it pins the fix (the
  divergence metric never fires without threaded `servedBuffs`).
- `TestHandleSnapshotBuffStatUpdated` against pre-fix `consumer.go`:
  `kafka/consumer/buff/consumer_test.go:190:5: undefined:
  handleSnapshotBuffStatUpdated` — build failure pre-fix (the handler
  doesn't exist), confirming the test is new coverage, not a pass-either-way
  assertion.

Worktree was restored to HEAD after this check (`git status --porcelain`
clean).

**Registered-handler path vs. registry method (scrutiny point 3):** the
consumer test invokes `handleSnapshotBuffStatUpdated(sc, nil)(logrus.New(),
ctx, su)` directly rather than routing a raw Kafka message through
`InitHandlers`'s `message.AdaptHandler(message.PersistentConfig(...))`
wrapper. This is the same pattern every other handler test in this file uses
(`TestHandleSnapshotBuff` for Applied/Expired, `TestHandleSnapshotBuff...`
elsewhere) — it exercises the handler function's own logic (type filter,
tenant/world gate, registry call) but not the `AdaptHandler`/decode layer
itself. That decode layer is untouched, generic, shared infrastructure
(`message.AdaptHandler`/`message.PersistentConfig`), already exercised by
every other handler registered in this same `InitHandlers` for years, and is
not part of this commit's diff — so this is a pre-existing test-depth
convention in the file, not a new gap introduced by this commit. Not
blocking; noted because the brief asked for scrutiny.

## Build / test verification

```
cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./...
```
All packages pass (module-local; matches the brief's stated verification
scope — "Module-local `go build ./... && go test ./...` only").

## Findings

### Blocking

None.

### Non-blocking

1. `processor.go:57-70` (fast path) — the buffs read for the shadow sample
   is a second, independently-locked `r.View()` call after
   `r.ComposedIfValid()`, not part of the same critical section. This is a
   genuine (if narrow, non-I/O-bound) TOCTOU window; it can only suppress a
   sample (via `BuffsValid` flipping false) or reflect a buff set that
   changed a few instructions after the composed-model check, never
   fabricate a divergence out of nothing. Consistent with, not worse than,
   the pre-existing staleness the async REST-refetch in `shadowCompare`
   already tolerates. No action requested; recorded for the design record
   since design.md §8's "same-moment" framing is now only approximately
   true for the buffs component on the fast path.

## Not evaluable

None — both call sites, the consumer arm, the cross-service contract, and
both new tests were traced end to end within the commit's diff and its
directly-referenced dependencies (atlas-buffs producer/processor, registry,
shadow.go).
