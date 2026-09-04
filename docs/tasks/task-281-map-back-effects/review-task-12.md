# Review: Task 12 — atlas-messages `@backeffect` / `@clearbackeffect` GM commands

Commit range: `b11dfdf67..3e0251dda` (single commit `3e0251dda`,
"feat(atlas-messages): add @backeffect and @clearbackeffect GM commands")

Brief: `.superpowers/sdd/plan/task-12-brief.md`
Report: `.superpowers/sdd/plan/task-12-report.md`

## Scope

Diff touches exactly the four files the brief named:

```
services/atlas-messages/atlas.com/messages/command/map/back_effect.go       | 114 ++
services/atlas-messages/atlas.com/messages/command/map/back_effect_test.go  | 128 ++
services/atlas-messages/atlas.com/messages/kafka/message/map/kafka.go       |  15 +-
services/atlas-messages/atlas.com/messages/main.go                          |   2 +
```

No extraneous files, no edits outside the declared scope. `scope_confirmed`: matches the commit exactly.

## Findings

### 1. Cross-service contract — VERIFIED, no defect

The implementer's claim about the file split is correct. In atlas-maps:

- `services/atlas-maps/atlas.com/maps/kafka/message/map/kafka.go` contains only the **status event**
  types (`EventTopicMapStatusTypeBackEffectSet`/`Clear`, `BackEffectSet`, `BackEffectClear` structs) —
  confirmed via `grep -n -i "command\|CommandType\|BackEffect"` on that file, which returned only the
  event-side symbols.
- `services/atlas-maps/atlas.com/maps/kafka/message/map/command.go` is the actual authority for
  commands:

```go
const (
	EnvCommandTopicMap         = "COMMAND_TOPIC_MAP"
	CommandTypeWeatherStart    = "WEATHER_START"
	CommandTypePlayJukebox     = "PLAY_JUKEBOX"
	CommandTypeSetBackEffect   = "SET_BACK_EFFECT"
	CommandTypeClearBackEffect = "CLEAR_BACK_EFFECT"
)
...
type SetBackEffectCommandBody struct {
	Effect   uint8  `json:"effect"`
	FieldId  uint32 `json:"fieldId"`
	PageId   uint8  `json:"pageId"`
	Duration uint32 `json:"duration"`
}
type ClearBackEffectCommandBody struct{}
```

atlas-messages' new `kafka.go` additions (diff hunk):

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

Constant strings, struct field names, order (cosmetic only — JSON tags govern wire shape, and Go
struct field order never affects `encoding/json`), JSON tags, and integer widths are byte-identical.
`map.Id` is confirmed `uint32` (`go doc .../atlas-constants/map Id` → `type Id uint32`), so
`FieldId: uint32(f.MapId())` in `back_effect.go:72` is a same-width conversion, not a narrowing cast.
`PageId`/`Effect` parsed with `strconv.ParseUint(..., 10, 8)` matching the `uint8` field width;
`Duration` parsed with `strconv.ParseUint(..., 10, 32)` matching the `uint32` field width. No
narrowing anywhere in the chain.

### 2. atlas-maps consumer branches on these constants — VERIFIED

`services/atlas-maps/atlas.com/maps/kafka/consumer/map/consumer.go:110-127` (`handleSetBackEffectCommand`)
gates on `if c.Type != mapKafka.CommandTypeSetBackEffect { return }` and reads `c.Body.Effect`,
`c.Body.FieldId`, `c.Body.PageId`, `c.Body.Duration` — the same field names atlas-messages produces.
`consumer.go:141-155` (`handleClearBackEffectCommand`) gates on `CommandTypeClearBackEffect`. Both
handlers are registered in `InitHandlers` (consumer.go:34-44). Confirmed by direct read of the file,
not inferred.

### 3. `fieldId` source — VERIFIED

`back_effect.go:24` regex `^@backeffect\s+(\d+)\s+([01])(?:\s+(\d+))?$` has exactly three capture
groups: pageId, effect, optional durationMs. No fieldId token. `back_effect.go:72`:
`FieldId: uint32(f.MapId())` — taken from the `field.Model` argument passed into the producer
closure (the invoking character's own field), never from a parsed token. Same pattern as
`weather.go`'s `MapId: f.MapId()` in the outer `Command[E]` wrapper.

### 4. Registration — VERIFIED

`main.go` diff:
```go
command.Registry().Add(_map.BackEffectCommandProducer)
command.Registry().Add(_map.ClearBackEffectCommandProducer)
```
placed immediately after `command.Registry().Add(_map.WeatherCommandProducer)` (confirmed by reading
the surrounding lines). Both producer functions gate on `c.Gm()` inside the closure
(`back_effect.go:30`, `back_effect.go:89`) before any Kafka production — same ordering as
`weather.go:31`.

### 5. Tests — genuine RED, cover the required cases

- Report's RED evidence (`undefined: BackEffectCommandProducer` / `undefined: ClearBackEffectCommandProducer`,
  compile failure) is consistent with a new file being introduced; re-ran locally:
  `go test ./command/map/... -run BackEffect -v` → all 10 subtests PASS (7 for `@backeffect`, 3 for
  `@clearbackeffect`), matching the brief's table exactly (set-with-duration, set-without-duration,
  hide, non-gm-rejected, effect-out-of-range, missing-effect, unrelated-message; gm-clear,
  non-gm-rejected, unrelated-message).
- These are not vacuous existence checks: the non-gm cases assert `found == false` while the
  regex would otherwise match, so removing the `c.Gm()` gate would flip those two subtests to
  failing. The `effect out of range` and `missing effect` cases rely on the `[01]` alternation and
  the two-mandatory-group shape of the regex; verified by hand that `@backeffect 1 2` and
  `@backeffect 1` cannot satisfy the anchored pattern, so a regex regression (e.g. accidentally making
  effect optional, or allowing any digit) would flip these to failing.
- Default-duration path: covered by "set without duration" (`@backeffect 1 0`, expects `found=true`);
  the actual default-to-`0` value is not asserted here because the executor is never invoked (no
  broker in this test binary — same rationale documented for
  `TestWarpCommandProducer_RegexPatterns`). The brief explicitly assigns message-shape verification
  (including the zero-duration default) to Task 11's `map_command/producer_test.go` in
  atlas-saga-orchestrator, which produces the same `Command[SetBackEffectCommandBody]` shape onto the
  same topic and is a different producer of the identical contract. This is a pre-existing design
  decision in the brief, not a gap introduced by this task — not a finding against Task 12.
- Full module test suite: `go test ./...` from `services/atlas-messages/atlas.com/messages` — all
  packages `ok`, no failures, no unexpected skips.

### 6. errcheck cleanliness — VERIFIED by manual inspection

`errcheck` binary is not installed in this environment, so I inspected the diff by hand instead of
relying on module-local `go vet`/`go test` (which do not catch this class, as the brief warns).
`grep -n "_ =\|_, _" back_effect.go back_effect_test.go` → no matches. Every `strconv.ParseUint` call
in `back_effect.go` (lines 34, 39, 46) checks its returned `error` and short-circuits with
`return nil, false`. `producer.ProviderImpl(...)` result is `return`ed directly (lines 54, 95), not
discarded. No new error-returning calls in the test file (only `t.Errorf`, which returns nothing).
`go build ./...` and `go vet ./...` both clean (no output) across the whole module.

## Non-blocking observations

- `ClearBackEffectCommandProducer`'s match-length check (`len(match) != 1`) is correct for a
  zero-capture-group regex (`FindStringSubmatch` returns a 1-element slice — the whole match — when
  there are no groups), consistent with existing conventions elsewhere in the file for the anchored,
  no-argument case.
- Struct field *declaration order* differs cosmetically between atlas-maps (`Effect, FieldId, PageId,
  Duration`) — actually identical order in both files as shown above; no discrepancy to flag.

## Not evaluable

None. All six controller-flagged claims were independently verified against source, not accepted from
the report.

## Verdict rationale

Every asserted claim in the report was independently checked against the actual atlas-maps source
(not the report's paraphrase), the constants/struct fields are byte-identical end-to-end including
integer widths, the consumer wiring exists and branches correctly, `fieldId` is proven to come from
`f.MapId()`, registration and GM-gating are placed correctly, tests are genuine regression guards
(not existence-only), and no unchecked errors exist in the new code. No blocking defects found.
