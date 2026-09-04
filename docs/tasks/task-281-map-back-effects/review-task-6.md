# Review: Task 6 — atlas-maps back-effect command consumer / status-event producer

Commit range: `827d1592b..372af1f83` (single commit `372af1f83`,
"feat(atlas-maps): consume back-effect commands and emit status events")

Brief: `.superpowers/sdd/plan/task-6-brief.md`
Report: `.superpowers/sdd/plan/task-6-report.md`
Prior interfaces: `.superpowers/sdd/plan/task-5-report.md`

## Scope

Diff touches exactly the five files named in the brief's "### Files" list:

- `services/atlas-maps/atlas.com/maps/kafka/message/map/command.go`
- `services/atlas-maps/atlas.com/maps/kafka/message/map/kafka.go`
- `services/atlas-maps/atlas.com/maps/map/backeffect/producer.go` (new)
- `services/atlas-maps/atlas.com/maps/kafka/consumer/map/consumer.go`
- `services/atlas-maps/atlas.com/maps/kafka/consumer/map/consumer_test.go`

`git diff --stat` confirms no file outside this set was touched. Scope confirmed, no drift.

## Findings

### 1. Cross-service contract — command/event bodies match the brief verbatim — PASS

`command.go` (`git diff` hunk, `kafka/message/map/command.go:11-12` for the
new consts, `:41-49` for the structs):

```go
CommandTypeSetBackEffect   = "SET_BACK_EFFECT"
CommandTypeClearBackEffect = "CLEAR_BACK_EFFECT"
...
type SetBackEffectCommandBody struct {
	Effect   uint8  `json:"effect"`
	FieldId  uint32 `json:"fieldId"`
	PageId   uint8  `json:"pageId"`
	Duration uint32 `json:"duration"`
}
type ClearBackEffectCommandBody struct{}
```

`kafka.go` (`kafka/message/map/kafka.go:19-20` for the new consts, `:64-72`
for the structs):

```go
EventTopicMapStatusTypeBackEffectSet   = "BACK_EFFECT_SET"
EventTopicMapStatusTypeBackEffectClear = "BACK_EFFECT_CLEAR"
...
type BackEffectSet struct {
	Effect   uint8  `json:"effect"`
	FieldId  uint32 `json:"fieldId"`
	PageId   uint8  `json:"pageId"`
	Duration uint32 `json:"duration"`
}
type BackEffectClear struct{}
```

Field names, order, types, and `json` tag casing (`effect`/`fieldId`/`pageId`/`duration`)
match the brief's "Interfaces produced" block exactly, on both the command
and the event side. Const string values (`"SET_BACK_EFFECT"`,
`"CLEAR_BACK_EFFECT"`, `"BACK_EFFECT_SET"`, `"BACK_EFFECT_CLEAR"`) also match.

`map/backeffect/producer.go:14-45` — `BackEffectSetEventProvider` and
`BackEffectClearEventProvider` signatures match the brief exactly
(`func BackEffectSetEventProvider(transactionId uuid.UUID, f field.Model, e BackEffectEntry) model.Provider[[]kafka.Message]`,
`func BackEffectClearEventProvider(transactionId uuid.UUID, f field.Model) model.Provider[[]kafka.Message]`),
envelope populated with `TransactionId/WorldId/ChannelId/MapId/Instance/Type/Body`
in that order (`producer.go:16-29`, `:35-44`), keyed by
`producer.CreateKey(int(f.MapId()))` — matches `jukebox/producer.go`'s
pattern field-for-field (diffed the two files side by side; only the type
names and field substitutions differ).

### 2. `handleSetBackEffectCommand` rejects invalid `Effect` without mutating state or producing — PASS

`kafka/consumer/map/consumer.go` (new function, ~line 110):

```go
if c.Body.Effect != 0 && c.Body.Effect != 1 {
    l.Warnf("Rejecting set back effect command with invalid effect [%d] for map [%d] instance [%s].", c.Body.Effect, c.MapId, c.Instance)
    return
}
```

This `return` precedes both the `field.NewBuilder(...)` call and the
`backeffect.NewProcessor(l, ctx).Set(...)`/producer call, so a rejected
`Effect` neither mutates the registry nor produces an event. Log line names
the invalid value, map id, and instance, and is at `Warnf` — matches the
brief. Verified by `TestHandleSetBackEffectCommand_RejectsInvalidEffect`
(`consumer_test.go`, `Effect: 2` → `require.Len(t, entries, 0)`).

### 3. `handleClearBackEffectCommand` produces the clear event even when `Clear` returns `false` — PASS

```go
if !backeffect.NewProcessor(l, ctx).Clear(f) {
    l.Debugf("Received clear back effect command for map [%d] instance [%s] with no active entries.", c.MapId, c.Instance)
}

err := producer.ProviderImpl(l)(ctx)(mapKafka.EnvEventTopicMapStatus)(backeffect.BackEffectClearEventProvider(c.TransactionId, f))
```

The produce call sits outside and after the `if !Clear(f)` block, so it
runs unconditionally regardless of whether `Clear` found anything to
remove — matches PRD FR-4 (a desynced client must be resettable). Debug log
(not warn/error) on the empty-field path is proportionate — an empty clear
is not a fault condition.

`TestHandleClearBackEffectCommand_EmptyFieldIsNotAnError` exercises the
`Clear(f) == false` path and asserts the handler doesn't panic and
`GetActive` stays empty; it does not (and per the brief, cannot — no
broker in this test binary) assert the event was actually produced. That
gap is explicitly acknowledged by the brief itself ("neither do these; the
event shape is covered by ... Task 8's channel-side consumer test") and is
not a defect in this task — flagged under "Not evaluable" below for
completeness, not as a finding.

### 4. Type guards and `InitHandlers` registration — PASS

Both handlers guard on `c.Type` first and return early otherwise
(`consumer.go`: `if c.Type != mapKafka.CommandTypeSetBackEffect { return }`
and the `ClearBackEffect` equivalent), matching
`handlePlayJukeboxCommand`'s pattern. `TestHandleSetBackEffectCommand_IgnoresWrongType`
confirms the guard is load-bearing (constructs a `SetBackEffectCommandBody`
with a `CommandTypeWeatherStart` type and asserts no entry is recorded).

`InitHandlers` (`consumer.go:32-47`) registers both new handlers with the
identical `rf(t, message.AdaptHandler(message.PersistentConfig(handleXCommand())))`
shape used for `handleWeatherStartCommand`/`handlePlayJukeboxCommand`, each
wrapped in the same `if _, err := ...; err != nil { return err }` guard.

### 5. Tests use a fresh tenant per case; assertions are genuine — PASS

Every new test case calls `ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)`
followed by `ctx := tenant.WithContext(context.Background(), ten)` before
touching the handler or the processor — same pattern as the pre-existing
jukebox tests, and since the registry keys on `FieldKey{Tenant, Field}`
(`map/backeffect/registry.go:10-13`), a fresh `uuid.New()` tenant per test
prevents cross-test leakage through the package-level singleton registry.

Assertions are genuine, not tautological:
- `RecordsEntry` asserts `require.Len(entries, 1)` plus field-by-field
  equality against the command body — would fail if `Set` were a no-op or
  mis-mapped a field.
- `RejectsInvalidEffect` and `IgnoresWrongType` both assert `Len == 0`
  against a registry that starts empty and would be non-empty if the
  respective guard were missing — genuine negative assertions.
- `RemovesEntries` seeds two entries via the processor directly, then
  asserts `Len == 0` after the handler runs — would fail if `Clear` were a
  no-op.
- `EmptyFieldIsNotAnError` asserts no panic and `Len == 0` on an
  already-empty field — thin, but matches exactly what the brief specifies
  for that case (handler must not error on empty clear).

`go test ./kafka/consumer/map/... ./map/backeffect/...` — all 8 new/existing
cases in `kafka/consumer/map` and 4 in `map/backeffect` pass; `go build ./...`
clean.

### 6. "No clamp on Duration" documented in a comment — PASS

`consumer.go`, inside the `BackEffectEntry{}` literal in
`handleSetBackEffectCommand`:

```go
entry := backeffect.BackEffectEntry{
    Effect:  c.Body.Effect,
    FieldId: c.Body.FieldId,
    PageId:  c.Body.PageId,
    // Duration is not clamped: it is a fade length bounded by the
    // client's own tween, with no denial-of-service shape comparable
    // to pinning a field's BGM (the counterpart to maxJukeboxDuration
    // above).
    Duration: c.Body.Duration,
}
```

This sits directly below `const maxJukeboxDuration = 10 * time.Minute` and
its comment block (`consumer.go`, jukebox section, immediately preceding
`handlePlayJukeboxCommand`), so the "counterpart to the comment above it"
requirement from the brief is satisfied both in wording and in physical
placement within the same file.

## Not evaluable

- Whether the produced Kafka message bytes are actually well-formed on the
  wire (correct `StatusEvent[T]` JSON marshalling, key computation) is not
  exercised by any test in this unit — the brief explicitly defers that to
  Task 8's consumer-side test. Not a defect in Task 6; flagged as
  out-of-surface for this review.
- `backeffect.Processor`'s internal correctness (tenant isolation
  mechanics, `Set`/`Clear`/`GetActive` semantics) is Task 5's surface, not
  re-verified here beyond confirming the Task 6 code calls it as
  documented in `task-5-report.md`.

## Verification run

```
cd services/atlas-maps/atlas.com/maps && go build ./... && go test ./kafka/consumer/map/... ./map/backeffect/...
ok  	atlas-maps/kafka/consumer/map	(cached)
ok  	atlas-maps/map/backeffect	(cached)
```
