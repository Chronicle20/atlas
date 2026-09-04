# Review: Task 10 — `libs/atlas-saga` set/clear back-effect actions and payloads

Commit range: `be28fe948..bae72f163` (single commit `bae72f163`)
Brief: `.superpowers/sdd/plan/task-10-brief.md`
Report: `.superpowers/sdd/plan/task-10-report.md`

## Scope

`git diff --stat be28fe948..bae72f163`:

```
libs/atlas-saga/model.go          |  6 ++++++
libs/atlas-saga/payloads.go       | 20 ++++++++++++++++++++
libs/atlas-saga/unmarshal.go      | 12 ++++++++++++
libs/atlas-saga/unmarshal_test.go | 30 ++++++++++++++++++++++++++++++
```

Matches the brief's file list exactly (`model.go`, `payloads.go`, `unmarshal.go`,
`unmarshal_test.go`). No files touched outside `libs/atlas-saga`.

## Findings

### 1. `Action` constant naming/value/comment convention — PASS

`libs/atlas-saga/model.go:290-295`:

```go
PlayJukebox Action = "play_jukebox"
// SetBackEffect starts a back effect (background animation) in one field.
// Duration is a fade length in milliseconds, not a lifetime -- atlas-maps
// owns how long the effect itself persists.
SetBackEffect Action = "set_back_effect"
// ClearBackEffect stops the active back effect in one field.
ClearBackEffect Action = "clear_back_effect"
```

Values are lower_snake_case matching `PlayJukebox`'s `"play_jukebox"` style
exactly. The `SetBackEffect` comment draws the same "duration ≠ lifetime,
atlas-maps owns X" distinction that `PlayJukebox`'s comment draws about
`DurationMs` (`model.go:287-289`). `ClearBackEffect`'s comment is terser,
appropriately (no duration field to caveat). PASS.

### 2. Payload struct field names/JSON tags vs. Task 6 Kafka command bodies — PASS

Compared against the actual committed producer-side *command* body (not the
event body — see note below), `services/atlas-maps/atlas.com/maps/kafka/message/map/command.go:41-48`:

```go
type SetBackEffectCommandBody struct {
	Effect   uint8  `json:"effect"`
	FieldId  uint32 `json:"fieldId"`
	PageId   uint8  `json:"pageId"`
	Duration uint32 `json:"duration"`
}

type ClearBackEffectCommandBody struct{}
```

vs. `libs/atlas-saga/payloads.go:1321-1336`:

```go
type SetBackEffectPayload struct {
	WorldId   world.Id   `json:"worldId"`
	ChannelId channel.Id `json:"channelId"`
	MapId     _map.Id    `json:"mapId"`
	Instance  uuid.UUID  `json:"instance"`
	Effect    uint8      `json:"effect"`
	FieldId   uint32     `json:"fieldId"`
	PageId    uint8      `json:"pageId"`
	Duration  uint32     `json:"duration"`
}

type ClearBackEffectPayload struct {
	WorldId   world.Id   `json:"worldId"`
	ChannelId channel.Id `json:"channelId"`
	MapId     _map.Id    `json:"mapId"`
	Instance  uuid.UUID  `json:"instance"`
}
```

`Effect`/`FieldId`/`PageId`/`Duration` are name-for-name and type-for-type
identical (`uint8`/`uint32`/`uint8`/`uint32`), tags identical
(`effect`/`fieldId`/`pageId`/`duration`). This mirrors the existing
`PlayJukeboxPayload` (`payloads.go:1311-1319`) vs. `PlayJukeboxCommandBody`
(`command.go:36-39`) pattern of flattening the command's `Body` fields
together with the `Command[E]` envelope's routing fields
(`WorldId`/`ChannelId`/`MapId`/`Instance`) into one saga payload struct. The
routing fields on `SetBackEffectPayload`/`ClearBackEffectPayload` also match
`Command[E]`'s field names/types (`command.go:19-27`).

Note on which Task-6 file is the correct comparison target: the report cites
`services/atlas-maps/.../kafka/message/map/kafka.go`, which defines
`BackEffectSet`/`BackEffectClear` — those are the **event** (`StatusEvent[E]`)
bodies atlas-maps emits outward, not the **command** bodies the saga sends
in. The saga payload's job is to produce a command, so the correct
comparison target is `command.go`'s `SetBackEffectCommandBody`/
`ClearBackEffectCommandBody`. Both files happen to define field-identical
structs (`Effect`/`FieldId`/`PageId`/`Duration`, all matching types), so the
report's conclusion ("field-for-field identical") is not wrong, but it
verified against the wrong struct. No functional divergence found either
way — flagged as a non-blocking accuracy note on the report, not a code
defect.

### 3. Unmarshal case arms wired and exercised via real `json.Unmarshal` — PASS

`libs/atlas-saga/unmarshal.go:648-659` adds `case SetBackEffect:` and
`case ClearBackEffect:` between the existing `case PlayJukebox:` and
`case SetAssetOwner:` arms, byte-for-byte the same shape (unmarshal into a
locally-typed `payload` var, wrap errors with
`fmt.Errorf("failed to unmarshal payload for action %s: %w", ...)`, assign
via `s.Payload = any(payload).(T)`).

`TestUnmarshalSetBackEffectStep` and `TestUnmarshalClearBackEffectStep`
(`libs/atlas-saga/unmarshal_test.go:1666-1695`) both build a raw JSON
literal string, call `json.Unmarshal(data, &s)` on a `Step[any]`, and
type-assert `s.Payload` to the concrete payload type, exactly matching the
`TestUnmarshalPlayJukeboxStep` pattern (`unmarshal_test.go:1649-1663`) — not
constructing the payload directly. Confirmed by running:

```
$ go test ./... -run BackEffect -v
=== RUN   TestUnmarshalSetBackEffectStep
--- PASS: TestUnmarshalSetBackEffectStep (0.00s)
=== RUN   TestUnmarshalClearBackEffectStep
--- PASS: TestUnmarshalClearBackEffectStep (0.00s)
PASS
```

Both tests assert concrete field values (`MapId`, `Effect`, `PageId`,
`Duration`), so they would fail to compile without the payload types and
would fail the type assertion without the case arms — genuine, not vacuous
coverage.

### 4. Missing-case / default-arm hazard for other `Action` switches — NOT A DEFECT (deferred by design)

Grepped for `switch.*\.Action\b` and `switch.*Action()` across the repo.
Found several real switch sites over saga actions outside `libs/atlas-saga`,
most notably `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/processor.go:1210`
and multiple sites in `saga/compensator.go`. None of these were updated by
this commit, and none needed to be — checked
`.superpowers/sdd/plan/task-11-brief.md` and `task-12-brief.md`:

- Task 11 wires `SetBackEffect`/`ClearBackEffect` into
  `atlas-saga-orchestrator`'s `saga/model.go` re-exports, the executor
  switch, and `map_command` producers/handlers (`task-11-brief.md:52,62-70,84`).
- Task 12 wires the `atlas-messages` command producers
  (`BackEffectCommandProducer`/`ClearBackEffectCommandProducer`) and
  registers them in `main.go` (`task-12-brief.md:41-42,64-65,117`).

So the consumer-side seam is explicitly the next two tasks' job, not this
one's. Task 10's own scope (`libs/atlas-saga` only) is self-contained: the
new `Action` constants are inert until Task 11 references them, and nothing
in this commit's own package silently swallows the new cases (the
`switch s.Action` in `unmarshal.go` has no `default:` fallthrough that would
mask a missing arm — every action requires an explicit case, and one was
added for both). PASS for in-scope hazard; the cross-service wiring is
correctly deferred, not dropped.

### 5. Unchecked errors (errcheck class) — PASS

Both new `case` arms in `unmarshal.go` check the `json.Unmarshal` error and
wrap it (`unmarshal.go:650-652`, `654-656`). Both new tests check the
`json.Unmarshal` error via `t.Fatal(err)` (`unmarshal_test.go:1669-1671`,
`1684-1686`). No other fallible calls were introduced. `go vet ./...` clean.

## Verification run (module-local only, per task scope)

```
$ cd libs/atlas-saga && go build ./... && go vet ./... && go test ./... -run BackEffect -v
ok
$ go test ./...
ok  	github.com/Chronicle20/atlas/libs/atlas-saga	0.010s
```

## Not evaluable

- Whether Task 11/12's consumption of these constants correctly wires the
  executor switch, compensator, and command producers is out of this unit's
  scope and will be reviewed with those commits.

## Verdict

APPROVED_WITH_FINDINGS — one non-blocking accuracy note on the report (§2:
report verified field parity against the event-body struct
`BackEffectSet`/`BackEffectClear` in `kafka.go` rather than the
command-body struct `SetBackEffectCommandBody`/`ClearBackEffectCommandBody`
in `command.go`; the conclusion is correct but the citation is not the
struct that actually matters for a saga *command* payload). No blocking
defects found in the code itself.
