# Review: Task 11 — saga-orchestrator dispatch of set/clear back-effect actions

Commit range: `bae72f163..b11dfdf67` (single commit `b11dfdf67`)
Brief: `.superpowers/sdd/plan/task-11-brief.md` (incl. CONTROLLER CORRECTION)
Report: `.superpowers/sdd/plan/task-11-report.md`

## Scope

`git diff --stat bae72f163..b11dfdf67` (re-run):

```
.../saga-orchestrator/kafka/message/map/kafka.go   | 12 ++++
 .../saga-orchestrator/map_command/processor.go     | 10 ++++
 .../saga-orchestrator/map_command/producer.go      | 33 +++++++++++
 .../saga-orchestrator/map_command/producer_test.go | 48 ++++++++++++++++
 .../saga-orchestrator/saga/event_acceptance.go     |  2 ++
 .../atlas.com/saga-orchestrator/saga/handler.go    | 66 ++++++++++++++++++++++
 .../saga-orchestrator/saga/handler_test.go         | 52 ++++++++++++++++
 .../atlas.com/saga-orchestrator/saga/model.go      | 18 ++++++
 8 files changed, 241 insertions(+)
```

Matches exactly the file list in the brief's `### Files` section and the report's "Files changed" list. No files touched outside the brief's declared surface.

## Findings

### 1. Cross-service seam (command constants + body structs)

Re-ran the diff myself against the brief's CONTROLLER CORRECTION target
(`services/atlas-maps/.../kafka/message/map/command.go`, not the sibling
`kafka.go`):

```
services/atlas-maps/atlas.com/maps/kafka/message/map/command.go:15-16,41-48
services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/message/map/kafka.go:20-21,43-51
```

Both sides, byte-identical:

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

Confirmed the *string values* of the type constants match, not just the
struct shapes — `"SET_BACK_EFFECT"` / `"CLEAR_BACK_EFFECT"` on both sides.
Also confirmed atlas-maps' consumer actually branches on these exact
constants: `grep -n "CommandTypeSetBackEffect\|CommandTypeClearBackEffect"
services/atlas-maps/atlas.com/maps/kafka/consumer/map/consumer.go` →
`consumer.go:112` and `:146`, both non-test. **PASS.**

Also confirmed the report correctly distinguished `command.go` (commands, what
this task's producer emits and atlas-maps' consumer decodes) from the sibling
`kafka.go` (status *events* — `BackEffectSet`/`BackEffectClear`,
`EventTopicMapStatusTypeBackEffectSet`/`Clear`), which is a different,
unrelated contract. The report did not conflate the two. **PASS.**

### 2. Self-completing registration

`saga/event_acceptance.go:325-326` (new lines):

```go
sharedsaga.SetBackEffect:              {},
sharedsaga.ClearBackEffect:            {},
```

immediately beside the existing `sharedsaga.PlayJukebox: {}` entry
(`event_acceptance.go:327` after insertion) — same map, same empty-slice
value shape, consistent with `PlayJukebox`'s fire-and-forget /
self-completing registration. **PASS.**

### 3. Handler/dispatch/provider shape fidelity

- `saga/handler.go:190-191` — two new interface methods, matching the
  `handleX(s Saga, st Step[any]) error` signature used throughout.
- `saga/handler.go:1034-1037` — two new `case` arms in `GetHandler`'s
  dispatch switch (`case SetBackEffect: return h.handleSetBackEffect, true`,
  same for Clear). `grep -c` confirms exactly one `case` arm for each in
  `model.go`'s decode switch and one each in `handler.go`'s dispatch switch —
  no duplicates.
- `handleSetBackEffect`/`handleClearBackEffect` bodies
  (`saga/handler.go:3783-3843`) follow `handlePlayJukebox`'s shape exactly:
  type-assert with `errors.New("invalid payload")` on failure, structured
  `logrus.Fields` debug log, `field.NewBuilder(...).SetInstance(...).Build()`,
  a call into `h.mapCommandP`, `h.logActionError` on failure, and
  `_ = NewProcessor(h.l, h.ctx).StepCompleted(s.TransactionId(), true)` on
  success — same discard-with-`_` idiom as `handlePlayJukebox`.
- `map_command/producer.go:49-81` — `SetBackEffectCommandProvider` /
  `ClearBackEffectCommandProvider`, each a direct structural copy of
  `PlayJukeboxCommandProvider` (`producer.go:32-48`): same `producer.CreateKey(int(f.MapId()))`
  key derivation, same `mapKafka.Command[E]` envelope construction, same
  `producer.SingleMessageProvider(key, value)` return.
- `map_command/processor.go:16-17,49-56` — `SetBackEffect`/`ClearBackEffect`
  interface methods and one-line `ProcessorImpl` bodies
  (`producer.ProviderImpl(p.l)(p.ctx)(mapKafka.EnvCommandTopicMap)(...)`),
  identical idiom to `PlayJukebox`'s.
- `saga/model.go:254-257` — action aliases `SetBackEffect = sharedsaga.SetBackEffect`,
  `ClearBackEffect = sharedsaga.ClearBackEffect`, beside the existing
  aliases. `saga/model.go:397-398` — payload type aliases. `saga/model.go:1728-1741`
  — two decode `case` arms in `Step[T].UnmarshalJSON`, same shape as the
  `CreateNote` arm immediately below.

No missing dispatch arm, no unaliased payload, no action constant that
decodes but never dispatches — verified `go build ./...` (exit 0) and
`go vet ./saga/... ./map_command/... ./kafka/...` (clean, no output).
**PASS.**

### 4. Invalid-payload test behaviour

`saga/handler_test.go:1703-1754` — `TestHandleSetBackEffect_InvalidPayload`
and `TestHandleClearBackEffect_InvalidPayload`, each builds a
`NewStep[any]("...", Pending, {Set,Clear}BackEffect, "invalid-payload-type")`
and asserts `err != nil` and `strings.Contains(err.Error(), "invalid payload")`
— matching `TestHandlePlayJukebox_InvalidPayload`'s shape precisely. Re-ran:

```
go test ./saga/... ./map_command/... -run BackEffect -v
--- PASS: TestHandleSetBackEffect_InvalidPayload (0.00s)
--- PASS: TestHandleClearBackEffect_InvalidPayload (0.00s)
--- PASS: TestSetBackEffectCommandProvider (0.00s)
--- PASS: TestClearBackEffectCommandProvider (0.00s)
ok  	atlas-saga-orchestrator/saga	(cached)
ok  	atlas-saga-orchestrator/map_command	(cached)
```

Test honesty: these tests fail to even *compile* without the change (the
brief's Step 2 RED evidence, reproduced in the report, shows
`undefined: SetBackEffect` / `handleSetBackEffect undefined`), so they are
not vacuously-passing coverage. **PASS.**

### 5. Numeric widths across the seam

Read the types directly rather than trusting the report's table:

- `libs/atlas-saga/payloads.go:1322-1332` (`SetBackEffectPayload`):
  `Effect uint8`, `FieldId uint32`, `PageId uint8`, `Duration uint32`
  (plus `WorldId world.Id`, `ChannelId channel.Id`, `MapId _map.Id`,
  `Instance uuid.UUID`).
- `map_command/producer.go:49` — provider parameters:
  `effect uint8, fieldId uint32, pageId uint8, duration uint32` — same widths.
- `kafka/message/map/kafka.go:43-48` — `SetBackEffectCommandBody`: same
  four fields, same widths.
- `services/atlas-maps/.../command.go:41-46` — same widths on decode.

Confirmed the report's table is accurate: no numeric narrowing/widening
anywhere in the payload chain. The one conversion present,
`producer.CreateKey(int(f.MapId()))` (`producer.go:50,66`), is a widening
`uint32 → int` conversion copied verbatim from `PlayJukeboxCommandProvider`
and pre-exists this change's pattern — not a new risk. **PASS.**

### 6. Unchecked error returns

- `saga/handler.go` — both `h.mapCommandP.SetBackEffect(...)` and
  `h.mapCommandP.ClearBackEffect(...)` results are assigned to `err` and
  branched on (`handler.go:3806-3809`, `3834-3837`). The `StepCompleted` call
  uses the pre-existing `_ =` discard idiom, identical to
  `handlePlayJukebox`/`handleFieldEffectWeather` — not a new unchecked-error
  pattern.
- `map_command/producer_test.go` — every provider invocation and
  `json.Unmarshal` call is wrapped in `require.NoError(t, err)`
  (`producer_test.go:44,52,73,81` per the diff hunk). No bare/dropped error.
- `saga/handler_test.go` — `saga, err := NewBuilder()....Build()` followed
  by `assert.NoError(t, err)`; the `handleX` call's return is captured and
  asserted. No dropped error.

No errcheck-class finding in the new code. **PASS.**

## Not evaluable

None. The full review surface (constants, structs, handler dispatch, model
aliases, producer/processor, tests, and the atlas-maps consumer contract they
target) was read and independently re-verified with `git diff`, `grep`,
`go build`, `go vet`, and `go test`.

## Verdict rationale

All six checklist items pass with direct evidence; the implementation is a
faithful, byte-for-byte-verified copy of the `PlayJukebox` pattern with no
scope creep, no missing dispatch arm, no width mismatch, and no unchecked
error. The cross-service seam (command type strings + body structs) is
proven identical against atlas-maps' actual decode target, correcting the
brief's original (wrong) file pointer as the CONTROLLER CORRECTION directed.
