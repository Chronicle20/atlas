# Review — Task 8: `atlas-channel` writers and status-event broadcast

Commit range: `a84d158..057060bee` (single commit `057060bee`).
Brief: `.superpowers/sdd/plan/task-8-brief.md`.
Report: `.superpowers/sdd/plan/task-8-report.md`.

## Scope

`git diff --stat a84d158..057060bee`:

```
services/atlas-channel/.../kafka/consumer/map/consumer.go       | 50 ++
services/atlas-channel/.../kafka/consumer/map/consumer_test.go  | 174 ++
services/atlas-channel/.../kafka/message/map/kafka.go           | 11 ++
services/atlas-channel/.../main.go                               | 2 ++
services/atlas-channel/.../socket/writer/clear_back_effect.go   | 10 ++ (new)
services/atlas-channel/.../socket/writer/set_back_effect.go     | 10 ++ (new)
```

`git diff a84d158..057060bee -- services/atlas-maps` is empty — this commit touches only
`atlas-channel`, confirming it does not overlap with the concurrent lint fix in
`services/atlas-maps/.../map/backeffect/resource_test.go`. Scope matches the brief.

## Findings

### 1. Seam diff re-run (byte-identical claim)

```
$ diff services/atlas-maps/atlas.com/maps/kafka/message/map/kafka.go services/atlas-channel/atlas.com/channel/kafka/message/map/kafka.go
[ok] Files are identical
$ echo $?
0
```

Verified independently: the two `kafka/message/map/kafka.go` files are byte-identical,
whole-file, exit 0. Confirms the report's claim (task-8-report.md, "Cross-service seam
verification" section).

**PASS.**

### 2. `decodeSetBackEffect` — real codec, not struct-equality

`services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer_test.go:824-835`:

```go
func decodeSetBackEffect(t *testing.T, body []byte) fieldcb.SetBackEffect {
	t.Helper()
	req := request.Request(body)
	reader := request.NewRequestReader(&req, 0)
	var m fieldcb.SetBackEffect
	m.Decode(logrus.New(), context.Background())(&reader, nil)
	return m
}
```

This constructs a `request.Reader` over the captured wire bytes and calls the real
`fieldcb.SetBackEffect.Decode` (`libs/atlas-packet/field/clientbound/set_back_effect.go:60-67`).
`TestHandleStatusEventBackEffectSet_BroadcastsToField`
(`consumer_test.go:837-883`) asserts `m.Effect()`, `m.FieldId()`, `m.PageId()`,
`m.Duration()` against the recorded body — these getters read the struct fields the
decoder just populated, not the pre-encode input. This is a genuine round-trip
assertion through the wire, not a struct-equality shortcut.

**PASS.**

### 3. Narrowing conversions in the handler

`consumer.go:1175`:

```go
fieldcb.NewSetBackEffect(byte(e.Body.Effect), e.Body.FieldId, byte(e.Body.PageId), e.Body.Duration).Encode
```

Field types, both sides:
- `_map3.BackEffectSet` (`kafka/message/map/kafka.go:65-70`): `Effect uint8`, `FieldId uint32`, `PageId uint8`, `Duration uint32`.
- `fieldcb.NewSetBackEffect` (`libs/atlas-packet/field/clientbound/set_back_effect.go:39`): `(effect byte, fieldId uint32, pageId byte, duration uint32)`.

`byte` is a Go built-in alias for `uint8`, so `byte(e.Body.Effect)` and
`byte(e.Body.PageId)` are identity conversions (`uint8` -> `uint8`), not narrowing.
`FieldId`/`Duration` are already `uint32` on both sides — no conversion, no truncation
possible. `TestHandleStatusEventBackEffectSet_BroadcastsToField` uses `PageId: 1`,
`FieldId: 100000000`, `Duration: 1000` — none of these values are near a `uint8`
boundary, so the test would not have caught a real narrowing bug had one existed, but
the type analysis independently rules one out.

**PASS.**

### 4. Type guard + `sc.Is` guard, both present and tested

Both handlers carry both guards:

`consumer.go:1164-1170` (`handleStatusEventBackEffectSet`):
```go
if e.Type != _map3.EventTopicMapStatusTypeBackEffectSet {
	return
}
if !sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.ChannelId) {
	return
}
```
Same pair at `consumer.go:1184-1190` for `handleStatusEventBackEffectClear`.

`TestHandleStatusEventBackEffectSet_IgnoresOtherChannel` (`consumer_test.go:885-912`)
sets `ChannelId: channel.Id(1)` while `newTestServerModel` registers the server at
channel 0 — this exercises the `sc.Is` guard specifically (the `Type` field matches, so
only the channel guard can suppress the broadcast). With the guard removed, the handler
would call `doorAnnounce` unconditionally, and the test's `len(calls) != 0` assertion
would fail — confirmed by inspection (the guard is the only conditional standing between
event receipt and the `ForSessionsInMap` broadcast).

Ran the suite to confirm: `go test ./kafka/consumer/map/... -run BackEffect -v` — all
three back-effect tests PASS (see below).

**PASS**, with one non-blocking gap: unlike the jukebox handlers
(`TestHandleStatusEventJukeboxStart_IgnoresOtherEventTypes`, `consumer_test.go:578-605`),
there is no back-effect test that asserts the *type* guard alone (wrong `Type`, correct
channel) — only the channel guard is directly tested. The type guard is exercised
implicitly by every passing test using the correct type, but its removal would not be
caught by any test in this file. Not blocking: the type guard is a one-line
`==`/`!=` copied verbatim from the already-tested jukebox pattern, and the brief's test
table (task-8-brief.md:58-62) did not call for it either.

### 5. Writer body functions and `main.go` registration

Both writer files match the existing convention exactly:

- `socket/writer/set_back_effect.go` (new) — modeled on `play_jukebox.go`, both are
  thin `fieldcb.New...().Encode` wrappers with no other call site in the module.
- `socket/writer/clear_back_effect.go` (new) — modeled on `field_obstacle_all_reset.go`,
  same pattern (`fieldcb.FieldObstacleAllResetBody` also has no production caller —
  confirmed by inspection, it exists purely to satisfy the file-per-writer convention).

`main.go:872-878`:
```go
fieldcb.FieldObstacleAllResetWriter,
fieldcb.SetBackEffectWriter,
fieldcb.ClearBackEffectWriter,
...
fieldcb.PlayJukeboxWriter,
```
Both new writer names are present in the writer-name slice, placed immediately after
`FieldObstacleAllResetWriter` as the brief specified.

This is not dead code introduced by this task — it matches a pre-existing convention
(`FieldObstacleAllResetBody` already had no call site before this commit), so the
implementer's framing in the report is accurate.

**PASS.**

### 6. Clear-event broadcast is unconditional (PRD FR-4)

`consumer.go:1183-1198` (`handleStatusEventBackEffectClear`) has no conditional on
"does this field have recorded back-effect entries" — after the two guards, it always
calls:
```go
err := _map.NewProcessor(l, ctx).ForSessionsInMap(f, func(s session.Model) error {
	return doorAnnounce(l, ctx, wp, fieldcb.ClearBackEffectWriter, fieldcb.NewClearBackEffect().Encode, s)
})
```
`TestHandleStatusEventBackEffectClear_BroadcastsToField` confirms one announce is
recorded with a zero-length body for a single session, with no setup implying any
prior "set" state. The producer side (Task 6, out of scope here) is responsible for
deciding when to emit `BACK_EFFECT_CLEAR`; once the channel receives it, this handler
broadcasts unconditionally to every session in the field, satisfying FR-4's "always
resettable" requirement at this layer.

**PASS.**

## Build / test verification

```
$ cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./kafka/consumer/map/... -run BackEffect -v
=== RUN   TestHandleStatusEventBackEffectSet_BroadcastsToField
--- PASS: TestHandleStatusEventBackEffectSet_BroadcastsToField (0.00s)
=== RUN   TestHandleStatusEventBackEffectSet_IgnoresOtherChannel
--- PASS: TestHandleStatusEventBackEffectSet_IgnoresOtherChannel (0.00s)
=== RUN   TestHandleStatusEventBackEffectClear_BroadcastsToField
--- PASS: TestHandleStatusEventBackEffectClear_BroadcastsToField (0.00s)
PASS
ok  	atlas-channel/kafka/consumer/map	(cached)
```

`go build ./...` produced no errors.

## Not evaluable

- Producer-side (`atlas-maps`) emission logic and its own tests (Task 6) were treated as
  a fixed contract per the task boundary; not re-reviewed here beyond the byte-identical
  message-type diff.
- `libs/atlas-packet/field/clientbound/set_back_effect.go` /
  `clear_back_effect.go` (Task 3's encode/decode implementation) were read only to
  confirm the signature and field types the handler calls into; their own correctness
  (wire layout vs. the client decompile) is Task 3's review surface, not this one's.

## Summary

All six review questions check out against `file:line` evidence. The seam diff is
genuinely byte-identical (verified independently, not just trusted from the report).
`decodeSetBackEffect` exercises the real `Decode` path. No narrowing conversion exists
(`byte`/`uint8` are the same type). Both guards are present in both handlers, and the
channel guard is asserted by a test that would fail without it. Both writers are
registered in `main.go`, matching the pre-existing convention for orphan writer-body
files. The clear broadcast is unconditional. Build and the new tests pass.

One non-blocking gap: no test isolates the `Type` guard alone for the back-effect
handlers (only the channel guard is directly tested); the jukebox handlers this task
copied from have that additional case.
