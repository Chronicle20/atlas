# Shared Script Operation Implementations — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Extract the sixteen script operations implemented in more than one of
`atlas-map-actions` / `atlas-reactor-actions` / `atlas-portal-actions` /
`atlas-npc-conversations` into one pure step-builder package
`libs/atlas-script-core/ops`, converge their drifted parameter contracts, and rewire
all four executors to delegate.

**Architecture:** Each shared operation becomes a pure function
`func Op(p map[string]string, r Resolver, t Target, characterId uint32) (Step, error)` —
params in, `(Step, error)` out, no I/O. Every service keeps its own dispatch table,
saga assembly, transaction-id minting, step-id format, `initiatedBy` string, logging,
and its service-local operations. The seam that moves is exactly
`params map[string]string → (saga.Action, payload)`.

**Tech Stack:** Go 1.27.0, `go.work` multi-module workspace, `libs/atlas-saga` payload
structs and `saga.Action` constants, `libs/atlas-constants/field.Model`,
`libs/atlas-script-core/context.EvaluateValueAsInt`, `github.com/google/uuid`,
table-driven `testing` tests with no mocks.

**Spec:** `docs/tasks/task-300-shared-script-operations/design.md`
(PRD: `docs/tasks/task-300-shared-script-operations/prd.md`)

## Global Constraints

- Worktree: `.worktrees/task-300-shared-script-operations`, branch
  `task-300-shared-script-operations`. Never edit the main repo.
- **No service `go.mod` changes are needed.** All four services already `require` both
  `libs/atlas-saga` and `libs/atlas-script-core` at
  `v0.0.0-00010101000000-000000000000` with `replace` lines in place
  (map `go.mod:11-12,94,96`; reactor `go.mod:11-12,94,96`; portal `go.mod:12-13,101,105`;
  npc `go.mod:13-14,100,102`). Only `libs/atlas-script-core/go.mod` gains `atlas-saga`.
- No import cycle: `libs/atlas-saga/go.mod` requires only `atlas-constants` and
  `google/uuid` — it does not require `atlas-script-core`. Verified.
- Shared operations perform **no** network, Redis, Kafka or REST I/O, never call a saga
  processor, and never log. They return errors.
- Shared operations never call `strconv` on a raw param value. Every raw value goes
  through the injected `Resolver`, then through the package's range helpers.
- Every parse failure is a hard `*ParamError` naming the operation, the parameter and
  the offending value (FR-19, NFR-Observability).
- `saga.Status` for every step built here is `saga.Pending`.
- Use the project's Builder pattern for construction; no `*_testhelpers.go`.
- Preserve existing line endings. Repo-relative paths only in committed files.
- Flagless `tools/verify.sh` must exit 0 before the branch is done.
- `services/atlas-portal-actions/atlas.com/portal/script/optable_test.go` must pass
  **unchanged** — it is the FR-9 regression detector.

---

## Reference: the converged contracts

Every task below builds against this table. It is derived from the four current
implementations, cited by `path:line`.

| Op | Action const | Required | Optional (default) |
|---|---|---|---|
| `SendMessage` | `saga.SendMessage` | `message` | `messageType` **or** `type` (`"PINK_TEXT"`) |
| `SpawnMonster` | `saga.SpawnMonster` | `monsterId` | `mapId` (target map), `x`/`y` (target position, else 0), `count` (1), `team` (0) |
| `MoveEnvironment` | `saga.MoveEnvironment` | `name`, `value` | `kind` (`""` → `ObjectKindEnvironment`) |
| `ResetEnvironment` | `saga.ResetEnvironment` | — | — |
| `ShowIntro` | `saga.ShowIntro` | `path` | — |
| `ShowHint` | `saga.ShowHint` | `hint` | `width` (0), `height` (0) |
| `PlayPortalSound` | `saga.PlayPortalSound` | — | — |
| `ApplyConsumableEffect` | `saga.ApplyConsumableEffect` | `itemId` | — |
| `CreateSkill` | `saga.CreateSkill` | `skillId` | `level` (1), `masterLevel` (1), `expiration` (now+365d) |
| `UpdateSkill` | `saga.UpdateSkill` | `skillId` | `level` (1), `masterLevel` (1), `expiration` (now+365d) |
| `WarpToPortal` | `saga.WarpToPortal` | `mapId` | `portalId` (0), `portalName` (`""`) |
| `WarpToSavedLocation` | `saga.WarpToSavedLocation` | `locationType` | — |
| `SaveLocation` | `saga.SaveLocation` | `locationType` | `mapId` (target map), `portalId` (target portal) |
| `StartInstanceTransport` | `saga.StartInstanceTransport` | `routeName` | — |
| `StartQuest` | `saga.StartQuest` | `questId` (or `QuestDefaults.QuestId`) | `npcId` (`QuestDefaults.NpcId`) |
| `StageClearAttemptPq` | `saga.StageClearAttemptPq` | — (caller supplies instanceId **or** characterId) | — |

Behaviour changes vs. today (design §5), each asserted by a test:

- **FR-13** `SendMessage`: `messageType` wins over `type`; `"5"`→`PINK_TEXT`,
  `"6"`→`BLUE_TEXT` applies to **both** keys; npc's previously-required `messageType`
  (`operation_executor.go:2132`) becomes optional.
- **FR-15** `SpawnMonster`: reactor's swallowed parse failures
  (`executor.go:206,212,217`) become hard errors; npc's required `x`/`y`
  (`operation_executor.go:1842,1852`) become optional.
- **FR-16 / OQ-3** `SpawnMonster`: `Instance` is `Target.Field().Instance()` **only when
  the effective map id equals `Target.Field().MapId()`**, else `uuid.Nil`. map-actions
  previously hard-coded `uuid.Nil` (`executor.go:188`); npc never set it.
  `Team` is now populated from the param on the map path too.
- **§5.3** `WarpToPortal`: `mapId` becomes **required** (npc defaulted it to 0 at
  `operation_executor.go:1417-1424`). `Instance` stays `uuid.Nil` — neither caller sets
  it today and this task does not change that.
- **§5.4** `CreateSkill`/`UpdateSkill`: npc now honours `expiration` with portal's
  semantics (npc hard-coded now+365d at `operation_executor.go:1561,1605`).
  `level`/`masterLevel` widen from portal's `ParseInt(...,8)` to the full `byte` range.
- **FR-17** `StageClearAttemptPq`: one builder, with an explicit "exactly one of
  instanceId/characterId" assertion neither caller makes today.

---

## Task 1: `ops` package foundation

Creates the package: `Step`, `Target`, `Resolver`, `DirectResolver`, `ParamError`,
the param/range helpers, and the overridable clock. No operations yet.

### Files

- `libs/atlas-script-core/go.mod` — add the `atlas-saga` require + replace
- `libs/atlas-script-core/ops/ops.go` — **new file**
- `libs/atlas-script-core/ops/ops_test.go` — **new file**

Module root for `go build`/`go test`: `libs/atlas-script-core`.

Patterns to copy: `libs/atlas-script-core/operation/builder.go:8-47` (Builder pattern
with unexported fields + `Build()`); `libs/atlas-saga/go.mod:1-12` (the require/replace
shape to mirror).

**Interfaces produced** (every later task consumes these):

```go
package ops

type Step struct { /* unexported: status, action, payload */ }
func (s Step) Status() saga.Status
func (s Step) Action() saga.Action
func (s Step) Payload() any
func (s Step) AppendTo(b *saga.Builder, id string) *saga.Builder
func PayloadOf[T any](s Step) (T, error)

type Target struct { /* unexported: field, x, y, hasPosition, portalId */ }
func NewTargetBuilder(f field.Model) *TargetBuilder
func (b *TargetBuilder) SetPosition(x, y int16) *TargetBuilder
func (b *TargetBuilder) SetPortalId(id uint32) *TargetBuilder
func (b *TargetBuilder) Build() Target
func (t Target) Field() field.Model
func (t Target) Position() (int16, int16, bool)
func (t Target) PortalId() uint32

type Resolver interface {
    String(characterId uint32, param string, raw string) (string, error)
    Int(characterId uint32, param string, raw string) (int, error)
}
type DirectResolver struct{}

type ParamError struct { Op, Param, Value string; Err error }
func (e *ParamError) Error() string
func (e *ParamError) Unwrap() error
```

### Steps

- [ ] **Step 1: Add `atlas-saga` to `libs/atlas-script-core/go.mod`**

Edit `libs/atlas-script-core/go.mod` so the require block and replace block read:

```
require (
	github.com/Chronicle20/atlas/libs/atlas-constants v0.0.0
	github.com/Chronicle20/atlas/libs/atlas-saga v0.0.0
	github.com/google/uuid v1.6.0
)

replace github.com/Chronicle20/atlas/libs/atlas-constants => ../atlas-constants

replace github.com/Chronicle20/atlas/libs/atlas-routine => ../atlas-routine

replace github.com/Chronicle20/atlas/libs/atlas-saga => ../atlas-saga
```

(`google/uuid` moves out of the `// indirect` require — `ops` uses it directly.)
Do **not** add a `go.work` entry: `./libs/atlas-script-core` is already at `go.work:20`.

- [ ] **Step 2: Write the failing test**

Create `libs/atlas-script-core/ops/ops_test.go`, `package ops`. Table-driven; no mocks.

`TestDirectResolver` — subtests, calling `DirectResolver{}.String(1, "p", raw)` and
`DirectResolver{}.Int(1, "p", raw)`:

| subtest | raw | `String` result | `Int` result |
|---|---|---|---|
| identity | `"PINK_TEXT"` | `"PINK_TEXT"`, nil err | — |
| plain int | `"42"` | `"42"` | `42`, nil err |
| negative | `"-1"` | `"-1"` | `-1`, nil err |
| arithmetic | `"10 * 5"` | `"10 * 5"` | `50`, nil err |
| not a number | `"abc"` | `"abc"` | err non-nil |

(`Int` delegates to `context.EvaluateValueAsInt`, `arithmetic.go:91`; the `"-1"` case is
load-bearing — `EvaluateArithmeticExpression` skips the subtraction branch on a leading
`-` and falls through to `strconv.Atoi`, `arithmetic.go:69`.)

`TestParamErrorMessage` — subtests asserting `err.Error()` **exactly**:

| subtest | constructor | expected message |
|---|---|---|
| missing | `missingParam("spawn_monster", "monsterId")` | `spawn_monster: parameter "monsterId" is required` |
| invalid | `invalidParam("spawn_monster", "x", "abc", errors.New("value [abc] is not a valid integer"))` | `spawn_monster: parameter "x" value "abc": value [abc] is not a valid integer` |

Also assert `errors.As(missingParam("op","p"), &target)` succeeds with
`target *ParamError` and `target.Param == "p"`.

`TestTargetBuilder` — subtests:

| subtest | build | assertions |
|---|---|---|
| field only | `NewTargetBuilder(field.NewBuilder(0, 1, 910010000).Build()).Build()` | `Field().MapId() == 910010000`; `Position()` returns `(0, 0, false)`; `PortalId() == 0` |
| with position | `.SetPosition(-120, 33)` | `Position()` returns `(-120, 33, true)` |
| with portal | `.SetPortalId(7)` | `PortalId() == 7` |
| with instance | `field.NewBuilder(0,1,910010000).SetInstance(uuid.MustParse("11111111-1111-1111-1111-111111111111")).Build()` | `Field().Instance()` equals that UUID |

`TestStepAppendTo` — build a `Step` via the internal `newStep(saga.SendMessage,
saga.SendMessagePayload{CharacterId: 5})`, then:

```
b := saga.NewBuilder().SetSagaType(saga.InventoryTransaction).SetInitiatedBy("test")
s := st.AppendTo(b, "message-5").Build()
```

Assert `len(s.Steps) == 1`, `s.Steps[0].StepId == "message-5"`,
`s.Steps[0].Status == saga.Pending`, `s.Steps[0].Action == saga.SendMessage`.
(Check the exported field names on `saga.Saga`/`saga.Step` at
`libs/atlas-saga/model.go:340-346,411-418` and use whatever they actually are.)

`TestPayloadOf` — `PayloadOf[saga.SendMessagePayload](st)` returns the payload with
`CharacterId == 5` and a nil error; `PayloadOf[saga.SpawnMonsterPayload](st)` returns a
non-nil error and the zero value.

`TestRangeHelpers` — table over `rangedInt16`, `rangedInt8`, `rangedUint16`,
`rangedUint32`, `rangedByte`:

| helper | in-range input | expected | out-of-range input | expected error substring |
|---|---|---|---|---|
| `rangedInt16` | `-120` | `int16(-120)` | `40000` | `out of range for int16` |
| `rangedInt8` | `-1` | `int8(-1)` | `200` | `out of range for int8` |
| `rangedUint16` | `65535` | `uint16(65535)` | `65536` | `out of range for uint16` |
| `rangedUint32` | `4294967295` | `uint32(4294967295)` | `-1` | `out of range for uint32` |
| `rangedByte` | `255` | `byte(255)` | `256` | `out of range for byte` |

Each out-of-range case must return a `*ParamError` whose `Op` and `Param` are the ones
passed in.

`TestOptionalIntUsesResolver` — a `recordingResolver` defined **in the test file** that
records every `(param, raw)` pair it is asked for and delegates to `DirectResolver{}`.
Assert `optionalInt(map[string]string{"count": "3"}, rec, 1, "spawn_monster", "count", 1)`
returns `3` and that `rec` observed exactly `{"count", "3"}` — i.e. the helper never
parses the raw string itself.

- [ ] **Step 3: Run the test to verify it fails**

Run: `cd libs/atlas-script-core && go test ./ops/...`
Expected: FAIL — `no Go files in .../ops` / undefined symbols.

- [ ] **Step 4: Write `libs/atlas-script-core/ops/ops.go`**

`package ops`. Contents:

```go
// now is the package clock. Tests override it to make expiration-bearing
// payloads deterministic.
var now = time.Now

type Step struct {
	status  saga.Status
	action  saga.Action
	payload any
}

func newStep(action saga.Action, payload any) Step {
	return Step{status: saga.Pending, action: action, payload: payload}
}

func (s Step) Status() saga.Status { return s.status }
func (s Step) Action() saga.Action { return s.action }
func (s Step) Payload() any        { return s.payload }

// AppendTo adds the step to a saga builder under the caller's step id.
// Step-id composition stays with the caller (FR-8).
func (s Step) AppendTo(b *saga.Builder, id string) *saga.Builder {
	return b.AddStep(id, s.status, s.action, s.payload)
}

// PayloadOf type-asserts a step's payload. Callers whose step-id format embeds
// a parsed field (map-actions' "spawn-%d-%d" uses the monster id) use this
// rather than re-parsing the param.
func PayloadOf[T any](s Step) (T, error) {
	p, ok := s.payload.(T)
	if !ok {
		var zero T
		return zero, fmt.Errorf("step payload is %T, not %T", s.payload, zero)
	}
	return p, nil
}
```

`Target` + `TargetBuilder` per the Interfaces block above: `NewTargetBuilder(f
field.Model) *TargetBuilder` seeds `field`; `SetPosition` sets `x`, `y` and
`hasPosition = true`; `SetPortalId` sets `portalId`; `Build()` returns the value type.

`Resolver` and `DirectResolver`:

```go
// DirectResolver resolves without conversation state: String is the identity,
// Int is context.EvaluateValueAsInt (which supports arithmetic expressions).
// Used by map-actions, reactor-actions and portal-actions.
type DirectResolver struct{}

func (DirectResolver) String(_ uint32, _ string, raw string) (string, error) { return raw, nil }
func (DirectResolver) Int(_ uint32, _ string, raw string) (int, error) {
	return scriptcontext.EvaluateValueAsInt(raw)
}
```

Import `scriptcontext "github.com/Chronicle20/atlas/libs/atlas-script-core/context"`
(aliased because `ops` also needs no stdlib `context`, but the alias keeps the intent
obvious).

`ParamError` + constructors:

```go
type ParamError struct {
	Op    string
	Param string
	Value string
	Err   error
}

func (e *ParamError) Error() string {
	if e.Err == nil {
		return fmt.Sprintf("%s: parameter %q is required", e.Op, e.Param)
	}
	return fmt.Sprintf("%s: parameter %q value %q: %v", e.Op, e.Param, e.Value, e.Err)
}

func (e *ParamError) Unwrap() error { return e.Err }

func missingParam(op, name string) error { return &ParamError{Op: op, Param: name} }

func invalidParam(op, name, value string, err error) error {
	return &ParamError{Op: op, Param: name, Value: value, Err: err}
}
```

Param helpers — all unexported, all routed through the `Resolver`:

```go
func requiredString(p map[string]string, r Resolver, cid uint32, op, name string) (string, error)
func optionalString(p map[string]string, r Resolver, cid uint32, op, name, def string) (string, error)
func requiredInt(p map[string]string, r Resolver, cid uint32, op, name string) (int, error)
func optionalInt(p map[string]string, r Resolver, cid uint32, op, name string, def int) (int, error)
```

`requiredString` returns `missingParam` when the key is absent; otherwise it returns
`r.String(cid, name, raw)`, wrapping any resolver error in `invalidParam`.
`optionalString` returns `def` when absent. `requiredInt`/`optionalInt` are the same
shape over `r.Int`.

Range helpers, each returning `invalidParam(op, name, strconv.Itoa(v), fmt.Errorf("out
of range for <type>"))` when the value does not fit:

```go
func rangedInt8(op, name string, v int) (int8, error)
func rangedInt16(op, name string, v int) (int16, error)
func rangedByte(op, name string, v int) (byte, error)
func rangedUint16(op, name string, v int) (uint16, error)
func rangedUint32(op, name string, v int) (uint32, error)
```

- [ ] **Step 5: Run the test to verify it passes**

Run: `cd libs/atlas-script-core && go test ./ops/... && go vet ./ops/...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add libs/atlas-script-core/go.mod libs/atlas-script-core/go.sum libs/atlas-script-core/ops/ops.go libs/atlas-script-core/ops/ops_test.go
git commit -m "feat(script-core): add ops package foundation (Step, Target, Resolver, ParamError)"
```

---

## Task 2: `SendMessage` (FR-13, FR-14)

### Files

- `libs/atlas-script-core/ops/message.go` — **new file**
- `libs/atlas-script-core/ops/message_test.go` — **new file**

Module root: `libs/atlas-script-core`.

Patterns to copy: the four current bodies this converges —
`services/atlas-map-actions/atlas.com/map-actions/script/executor.go:220-252`,
`services/atlas-reactor-actions/atlas.com/reactor/script/executor.go:365-405`
(the `"5"`/`"6"` mapping),
`services/atlas-portal-actions/atlas.com/portal/script/executor.go:183-215`,
`services/atlas-npc-conversations/atlas.com/npc/conversation/operation_executor.go:2127-2160`
(all read-only references).

**Interfaces produced:**

```go
func SendMessage(p map[string]string, r Resolver, t Target, characterId uint32) (Step, error)
```

### Steps

- [ ] **Step 1: Write the failing test**

Create `libs/atlas-script-core/ops/message_test.go`. `TestSendMessage` — table-driven,
one `t.Run` per case, using `DirectResolver{}` and
`NewTargetBuilder(field.NewBuilder(0, 1, 910010000).Build()).Build()`. Assert the
returned step's `Action() == saga.SendMessage`, `Status() == saga.Pending`, and its
`saga.SendMessagePayload` field-by-field.

| case | params | expect |
|---|---|---|
| missing message | `{}` | err `send_message: parameter "message" is required` |
| default type | `{"message":"hi"}` | `MessageType: "PINK_TEXT"`, `Message: "hi"`, `CharacterId: 7`, `WorldId: 0`, `ChannelId: 1` |
| messageType key | `{"message":"hi","messageType":"NOTICE"}` | `MessageType: "NOTICE"` |
| type key | `{"message":"hi","type":"NOTICE"}` | `MessageType: "NOTICE"` |
| messageType wins | `{"message":"hi","messageType":"NOTICE","type":"POP_UP"}` | `MessageType: "NOTICE"` |
| numeric 5 via type | `{"message":"hi","type":"5"}` | `MessageType: "PINK_TEXT"` |
| numeric 6 via type | `{"message":"hi","type":"6"}` | `MessageType: "BLUE_TEXT"` |
| numeric 5 via messageType | `{"message":"hi","messageType":"5"}` | `MessageType: "PINK_TEXT"` |
| numeric 6 via messageType | `{"message":"hi","messageType":"6"}` | `MessageType: "BLUE_TEXT"` |
| unknown numeric passes through | `{"message":"hi","type":"9"}` | `MessageType: "9"` |

Add `TestSendMessageResolvesThroughResolver`: with the `recordingResolver` from
`ops_test.go`, call with `{"message":"hi","messageType":"NOTICE"}` and assert the
resolver observed `{"message","hi"}` and `{"messageType","NOTICE"}` — nothing else.

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd libs/atlas-script-core && go test ./ops/ -run TestSendMessage -v`
Expected: FAIL — `undefined: SendMessage`.

- [ ] **Step 3: Write `libs/atlas-script-core/ops/message.go`**

```go
package ops

const opSendMessage = "send_message"

// SendMessage builds a SendMessage step. It backs both the `send_message`
// (npc-conversations) and `drop_message` (map/reactor/portal actions) script
// operations — FR-14 keeps both dispatch names valid.
//
// Parameters:
//   - message      (required) the message text.
//   - messageType  (optional) one of "NOTICE", "POP_UP", "PINK_TEXT",
//     "BLUE_TEXT". `type` is accepted as an alias; `messageType` wins when both
//     are present. Numeric "5" maps to "PINK_TEXT" and "6" to "BLUE_TEXT" for
//     either key (carried forward from reactor-actions). Defaults to
//     "PINK_TEXT" when absent.
func SendMessage(p map[string]string, r Resolver, t Target, characterId uint32) (Step, error) {
	message, err := requiredString(p, r, characterId, opSendMessage, "message")
	if err != nil {
		return Step{}, err
	}

	messageType := "PINK_TEXT"
	key := ""
	if _, ok := p["messageType"]; ok {
		key = "messageType"
	} else if _, ok := p["type"]; ok {
		key = "type"
	}
	if key != "" {
		raw, err := requiredString(p, r, characterId, opSendMessage, key)
		if err != nil {
			return Step{}, err
		}
		switch raw {
		case "5":
			messageType = "PINK_TEXT"
		case "6":
			messageType = "BLUE_TEXT"
		default:
			messageType = raw
		}
	}

	return newStep(saga.SendMessage, saga.SendMessagePayload{
		CharacterId: characterId,
		WorldId:     t.Field().WorldId(),
		ChannelId:   t.Field().ChannelId(),
		MessageType: messageType,
		Message:     message,
	}), nil
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `cd libs/atlas-script-core && go test ./ops/... && go vet ./ops/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-script-core/ops/message.go libs/atlas-script-core/ops/message_test.go
git commit -m "feat(script-core): add shared SendMessage step builder (FR-13, FR-14)"
```

---

## Task 3: `SpawnMonster` (FR-15, FR-16, OQ-3)

### Files

- `libs/atlas-script-core/ops/monster.go` — **new file**
- `libs/atlas-script-core/ops/monster_test.go` — **new file**

Module root: `libs/atlas-script-core`.

Patterns to copy (read-only): `services/atlas-map-actions/atlas.com/map-actions/script/executor.go:125-197`,
`services/atlas-reactor-actions/atlas.com/reactor/script/executor.go:188-246`,
`services/atlas-npc-conversations/atlas.com/npc/conversation/operation_executor.go:1825-1898`.

**Interfaces produced:**

```go
func SpawnMonster(p map[string]string, r Resolver, t Target, characterId uint32) (Step, error)
```

### Steps

- [ ] **Step 1: Write the failing test**

Create `libs/atlas-script-core/ops/monster_test.go`. `TestSpawnMonster` — table-driven,
`DirectResolver{}`, `characterId = 7`. Two targets used across cases:

- `plain` = `NewTargetBuilder(field.NewBuilder(0, 1, 910010000).Build()).Build()`
- `instanced` = `NewTargetBuilder(field.NewBuilder(0, 1, 910010000).SetInstance(instID).Build()).SetPosition(-120, 33).Build()`
  where `instID = uuid.MustParse("22222222-2222-2222-2222-222222222222")`

Assert `Action() == saga.SpawnMonster` and the `saga.SpawnMonsterPayload` field-by-field.

| case | target | params | expect |
|---|---|---|---|
| missing monsterId | plain | `{}` | err `spawn_monster: parameter "monsterId" is required` |
| bad monsterId | plain | `{"monsterId":"abc"}` | err, `errors.As` → `*ParamError{Op:"spawn_monster",Param:"monsterId",Value:"abc"}` |
| defaults, no position | plain | `{"monsterId":"100100"}` | `MonsterId:100100, MapId:910010000, X:0, Y:0, Count:1, Team:0, Instance:uuid.Nil, CharacterId:7, WorldId:0, ChannelId:1` |
| position default from target | instanced | `{"monsterId":"100100"}` | `X:-120, Y:33, Instance:instID` |
| explicit x/y override target | instanced | `{"monsterId":"100100","x":"5","y":"6"}` | `X:5, Y:6` |
| FR-15 bad x hard-errors | instanced | `{"monsterId":"100100","x":"abc"}` | err `*ParamError{Param:"x",Value:"abc"}` — **reactor previously kept the default** (`executor.go:206-211`) |
| FR-15 bad y hard-errors | instanced | `{"monsterId":"100100","y":"abc"}` | err `*ParamError{Param:"y"}` |
| FR-15 bad count hard-errors | plain | `{"monsterId":"100100","count":"abc"}` | err `*ParamError{Param:"count"}` — **reactor previously kept `1`** (`executor.go:200-204`) |
| count | plain | `{"monsterId":"100100","count":"3"}` | `Count:3` |
| team | plain | `{"monsterId":"100100","team":"1"}` | `Team:1` |
| team out of range | plain | `{"monsterId":"100100","team":"200"}` | err containing `out of range for int8` |
| x out of range | plain | `{"monsterId":"100100","x":"40000"}` | err containing `out of range for int16` |
| OQ-3 same map keeps instance | instanced | `{"monsterId":"100100","mapId":"910010000"}` | `MapId:910010000, Instance:instID` |
| OQ-3 cross map drops instance | instanced | `{"monsterId":"100100","mapId":"910510202"}` | `MapId:910510202, Instance:uuid.Nil` |
| mapId default keeps instance | instanced | `{"monsterId":"100100"}` | `MapId:910010000, Instance:instID` |
| arithmetic count | plain | `{"monsterId":"100100","count":"2 * 3"}` | `Count:6` |

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd libs/atlas-script-core && go test ./ops/ -run TestSpawnMonster -v`
Expected: FAIL — `undefined: SpawnMonster`.

- [ ] **Step 3: Write `libs/atlas-script-core/ops/monster.go`**

```go
package ops

const opSpawnMonster = "spawn_monster"

// SpawnMonster builds a SpawnMonster step.
//
// Parameters:
//   - monsterId (required) the monster template id.
//   - mapId     (optional) defaults to the target's map.
//   - x, y      (optional) default to the target's position when it carries one
//     (reactor-actions passes the reactor's coordinates), otherwise 0.
//   - count     (optional) defaults to 1.
//   - team      (optional) defaults to 0.
//
// Every parse failure is a hard error (FR-15). Instance is taken from the
// target only when the effective map id equals the target's map id; a spawn
// aimed at a different map carries uuid.Nil, because the target's instance
// belongs to the current map's field and would address a field that does not
// exist (OQ-3).
func SpawnMonster(p map[string]string, r Resolver, t Target, characterId uint32) (Step, error) {
	monsterIdInt, err := requiredInt(p, r, characterId, opSpawnMonster, "monsterId")
	if err != nil {
		return Step{}, err
	}
	monsterId, err := rangedUint32(opSpawnMonster, "monsterId", monsterIdInt)
	if err != nil {
		return Step{}, err
	}

	mapIdInt, err := optionalInt(p, r, characterId, opSpawnMonster, "mapId", int(t.Field().MapId()))
	if err != nil {
		return Step{}, err
	}
	mapIdU, err := rangedUint32(opSpawnMonster, "mapId", mapIdInt)
	if err != nil {
		return Step{}, err
	}
	mapId := _map.Id(mapIdU)

	defX, defY, _ := t.Position()
	xInt, err := optionalInt(p, r, characterId, opSpawnMonster, "x", int(defX))
	if err != nil {
		return Step{}, err
	}
	x, err := rangedInt16(opSpawnMonster, "x", xInt)
	if err != nil {
		return Step{}, err
	}
	yInt, err := optionalInt(p, r, characterId, opSpawnMonster, "y", int(defY))
	if err != nil {
		return Step{}, err
	}
	y, err := rangedInt16(opSpawnMonster, "y", yInt)
	if err != nil {
		return Step{}, err
	}

	count, err := optionalInt(p, r, characterId, opSpawnMonster, "count", 1)
	if err != nil {
		return Step{}, err
	}

	teamInt, err := optionalInt(p, r, characterId, opSpawnMonster, "team", 0)
	if err != nil {
		return Step{}, err
	}
	team, err := rangedInt8(opSpawnMonster, "team", teamInt)
	if err != nil {
		return Step{}, err
	}

	instance := uuid.Nil
	if mapId == t.Field().MapId() {
		instance = t.Field().Instance()
	}

	return newStep(saga.SpawnMonster, saga.SpawnMonsterPayload{
		CharacterId: characterId,
		WorldId:     t.Field().WorldId(),
		ChannelId:   t.Field().ChannelId(),
		MapId:       mapId,
		Instance:    instance,
		MonsterId:   monsterId,
		X:           x,
		Y:           y,
		Team:        team,
		Count:       count,
	}), nil
}
```

Import `_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"` — confirm the
actual import path/alias used at
`services/atlas-map-actions/atlas.com/map-actions/script/executor.go` and mirror it.

- [ ] **Step 4: Run the test to verify it passes**

Run: `cd libs/atlas-script-core && go test ./ops/... && go vet ./ops/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-script-core/ops/monster.go libs/atlas-script-core/ops/monster_test.go
git commit -m "feat(script-core): add shared SpawnMonster step builder (FR-15, FR-16, OQ-3)"
```

---

## Task 4: environment and effect operations

Six operations with no convergence conflicts (design §5.7): `MoveEnvironment`,
`ResetEnvironment`, `ShowIntro`, `ShowHint`, `PlayPortalSound`, `ApplyConsumableEffect`.

### Files

- `libs/atlas-script-core/ops/environment.go` — **new file**
- `libs/atlas-script-core/ops/environment_test.go` — **new file**
- `libs/atlas-script-core/ops/effect.go` — **new file**
- `libs/atlas-script-core/ops/effect_test.go` — **new file**

Module root: `libs/atlas-script-core`.

Patterns to copy (read-only): `services/atlas-map-actions/atlas.com/map-actions/script/executor.go:257-323`
(move/reset environment) and `:97-123` (show_intro);
`services/atlas-portal-actions/atlas.com/portal/script/executor.go:101-119`
(play_portal_sound), `:218-260` (show_hint), `:494-527` (apply_consumable_effect);
`services/atlas-npc-conversations/atlas.com/npc/conversation/operation_executor.go:2336-2378`
(show_hint), `:2414-2436` (show_intro), `:2103-2125` (apply_consumable_effect).

**Interfaces produced:**

```go
func MoveEnvironment(p map[string]string, r Resolver, t Target, characterId uint32) (Step, error)
func ResetEnvironment(p map[string]string, r Resolver, t Target, characterId uint32) (Step, error)
func ShowIntro(p map[string]string, r Resolver, t Target, characterId uint32) (Step, error)
func ShowHint(p map[string]string, r Resolver, t Target, characterId uint32) (Step, error)
func PlayPortalSound(p map[string]string, r Resolver, t Target, characterId uint32) (Step, error)
func ApplyConsumableEffect(p map[string]string, r Resolver, t Target, characterId uint32) (Step, error)
```

All six keep the uniform signature even where a parameter is unused, so a dispatch
table can hold them interchangeably.

### Steps

- [ ] **Step 1: Write the failing tests**

`libs/atlas-script-core/ops/environment_test.go`, target
`instanced = NewTargetBuilder(field.NewBuilder(0, 1, 910010000).SetInstance(instID).Build()).Build()`,
`characterId = 7`:

`TestMoveEnvironment` — assert `Action() == saga.MoveEnvironment` and the
`saga.MoveEnvironmentPayload`:

| case | params | expect |
|---|---|---|
| missing name | `{"value":"3"}` | err `move_environment: parameter "name" is required` |
| blank name | `{"name":"  ","value":"3"}` | err `move_environment: parameter "name" is required` |
| missing value | `{"name":"gate01"}` | err `move_environment: parameter "value" is required` |
| bad value | `{"name":"gate01","value":"abc"}` | `*ParamError{Param:"value",Value:"abc"}` |
| defaults kind | `{"name":"gate01","value":"3"}` | `WorldId:0, ChannelId:1, MapId:910010000, Instance:instID, Kind:field.ObjectKindEnvironment, Name:"gate01", State:3` |
| explicit ENVIRONMENT | `{"name":"g","value":"1","kind":"ENVIRONMENT"}` | `Kind: field.ObjectKindEnvironment` |
| explicit OBSTACLE | `{"name":"g","value":"1","kind":"OBSTACLE"}` | `Kind: field.ObjectKindObstacle` |
| bad kind | `{"name":"g","value":"1","kind":"BOGUS"}` | `*ParamError{Param:"kind",Value:"BOGUS"}` |
| negative value out of range | `{"name":"g","value":"-1"}` | err containing `out of range for uint32` |

`TestResetEnvironment` — `{}` params: `Action() == saga.ResetEnvironment`,
payload `saga.ResetEnvironmentPayload{WorldId:0, ChannelId:1, MapId:910010000, Instance:instID}`, nil error.

`libs/atlas-script-core/ops/effect_test.go`, target
`plain = NewTargetBuilder(field.NewBuilder(0, 1, 910010000).Build()).Build()`:

`TestShowIntro`:

| case | params | expect |
|---|---|---|
| missing path | `{}` | err `show_intro: parameter "path" is required` |
| ok | `{"path":"Effect/Direction1.img/aranTutorial/ClickPoleArm"}` | `saga.ShowIntroPayload{CharacterId:7, WorldId:0, ChannelId:1, Path:"Effect/Direction1.img/aranTutorial/ClickPoleArm"}` |

`TestShowHint`:

| case | params | expect |
|---|---|---|
| missing hint | `{}` | err `show_hint: parameter "hint" is required` |
| defaults | `{"hint":"go left"}` | `Hint:"go left", Width:0, Height:0` |
| width/height | `{"hint":"go left","width":"200","height":"50"}` | `Width:200, Height:50` |
| bad width | `{"hint":"h","width":"abc"}` | `*ParamError{Param:"width",Value:"abc"}` |
| width out of range | `{"hint":"h","width":"65536"}` | err containing `out of range for uint16` |

`TestPlayPortalSound` — `{}`: `Action() == saga.PlayPortalSound`, payload
`saga.PlayPortalSoundPayload{CharacterId:7, WorldId:0, ChannelId:1}`.

`TestApplyConsumableEffect`:

| case | params | expect |
|---|---|---|
| missing itemId | `{}` | err `apply_consumable_effect: parameter "itemId" is required` |
| ok | `{"itemId":"2000000"}` | `saga.ApplyConsumableEffectPayload{CharacterId:7, WorldId:0, ChannelId:1, ItemId:2000000}` |
| bad itemId | `{"itemId":"abc"}` | `*ParamError{Param:"itemId",Value:"abc"}` |

- [ ] **Step 2: Run the tests to verify they fail**

Run: `cd libs/atlas-script-core && go test ./ops/ -run 'TestMoveEnvironment|TestResetEnvironment|TestShowIntro|TestShowHint|TestPlayPortalSound|TestApplyConsumableEffect' -v`
Expected: FAIL — undefined symbols.

- [ ] **Step 3: Write `environment.go` and `effect.go`**

`environment.go`, `package ops`:

```go
const (
	opMoveEnvironment  = "move_environment"
	opResetEnvironment = "reset_environment"
)

// MoveEnvironment builds a MoveEnvironment step, setting the state of one named
// field object.
//
// Parameters:
//   - name  (required) opaque object name; not validated against WZ data.
//     Whitespace-only is treated as absent.
//   - value (required) the new object state, uint32.
//   - kind  (optional) "ENVIRONMENT" or "OBSTACLE"; blank defaults to
//     ENVIRONMENT (see field.ParseObjectKind).
func MoveEnvironment(p map[string]string, r Resolver, t Target, characterId uint32) (Step, error) {
	name, err := requiredString(p, r, characterId, opMoveEnvironment, "name")
	if err != nil {
		return Step{}, err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return Step{}, missingParam(opMoveEnvironment, "name")
	}

	if strings.TrimSpace(p["value"]) == "" {
		return Step{}, missingParam(opMoveEnvironment, "value")
	}
	valueInt, err := requiredInt(p, r, characterId, opMoveEnvironment, "value")
	if err != nil {
		return Step{}, err
	}
	state, err := rangedUint32(opMoveEnvironment, "value", valueInt)
	if err != nil {
		return Step{}, err
	}

	kind, err := field.ParseObjectKind(p["kind"])
	if err != nil {
		return Step{}, invalidParam(opMoveEnvironment, "kind", p["kind"], err)
	}

	return newStep(saga.MoveEnvironment, saga.MoveEnvironmentPayload{
		WorldId:   t.Field().WorldId(),
		ChannelId: t.Field().ChannelId(),
		MapId:     t.Field().MapId(),
		Instance:  t.Field().Instance(),
		Kind:      kind,
		Name:      name,
		State:     state,
	}), nil
}

// ResetEnvironment builds a ResetEnvironment step, clearing every tracked field
// object and restoring the field's objects to their default state. Takes no
// parameters.
func ResetEnvironment(p map[string]string, r Resolver, t Target, characterId uint32) (Step, error) {
	return newStep(saga.ResetEnvironment, saga.ResetEnvironmentPayload{
		WorldId:   t.Field().WorldId(),
		ChannelId: t.Field().ChannelId(),
		MapId:     t.Field().MapId(),
		Instance:  t.Field().Instance(),
	}), nil
}
```

`kind` is read raw rather than through the resolver, matching both current call sites
(`map .../executor.go:271`, `reactor .../executor.go:302`). Note that in the doc comment.

`effect.go` implements the four remaining builders in the same shape:
`ShowIntro` (`requiredString "path"` → `saga.ShowIntroPayload`),
`ShowHint` (`requiredString "hint"`, `optionalInt "width"/"height"` default 0 →
`rangedUint16` → `saga.ShowHintPayload`),
`PlayPortalSound` (no params → `saga.PlayPortalSoundPayload`),
`ApplyConsumableEffect` (`requiredInt "itemId"` → `rangedUint32` →
`saga.ApplyConsumableEffectPayload`). Each carries its parameter contract as a doc
comment, moved from the inline comments named above rather than duplicated.

- [ ] **Step 4: Run the tests to verify they pass**

Run: `cd libs/atlas-script-core && go test ./ops/... && go vet ./ops/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-script-core/ops/environment.go libs/atlas-script-core/ops/environment_test.go libs/atlas-script-core/ops/effect.go libs/atlas-script-core/ops/effect_test.go
git commit -m "feat(script-core): add shared environment and effect step builders"
```

---

## Task 5: skill and quest operations

`CreateSkill`, `UpdateSkill` (§5.4 — npc gains `expiration`), `StartQuest`,
`StageClearAttemptPq` (FR-17).

### Files

- `libs/atlas-script-core/ops/skill.go` — **new file**
- `libs/atlas-script-core/ops/skill_test.go` — **new file**
- `libs/atlas-script-core/ops/quest.go` — **new file**
- `libs/atlas-script-core/ops/quest_test.go` — **new file**

Module root: `libs/atlas-script-core`.

Patterns to copy (read-only): `services/atlas-portal-actions/atlas.com/portal/script/executor.go:311-379`
(create_skill, incl. the `expiration: "-1"` sentinel), `:380-448` (update_skill),
`:653-691` (start_quest);
`services/atlas-npc-conversations/atlas.com/npc/conversation/operation_executor.go:1524-1566`
(create_skill), `:1568-1610` (update_skill), `:1964-2012` (start_quest context defaults),
`:2641-2648` (stage_clear_attempt_pq);
`services/atlas-reactor-actions/atlas.com/reactor/script/executor.go:535-556`
(stage_clear_attempt).

**Interfaces produced:**

```go
func CreateSkill(p map[string]string, r Resolver, t Target, characterId uint32) (Step, error)
func UpdateSkill(p map[string]string, r Resolver, t Target, characterId uint32) (Step, error)

type QuestDefaults struct { QuestId uint32; NpcId uint32 }
func StartQuest(p map[string]string, r Resolver, t Target, characterId uint32, d QuestDefaults) (Step, error)
func StageClearAttemptPq(t Target, characterId uint32, instanceId uuid.UUID) (Step, error)
```

### Steps

- [ ] **Step 1: Write the failing tests**

`libs/atlas-script-core/ops/skill_test.go`. Both skill tests override the package clock
so expirations are exact:

```go
func withFixedNow(t *testing.T, at time.Time) {
	t.Helper()
	prev := now
	now = func() time.Time { return at }
	t.Cleanup(func() { now = prev })
}
```

Use `base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)`.

`TestCreateSkill` (and an identical `TestUpdateSkill` asserting
`saga.UpdateSkillPayload` / `saga.UpdateSkill` — repeat the table, do not cross-refer):

| case | params | expect |
|---|---|---|
| missing skillId | `{}` | err `create_skill: parameter "skillId" is required` |
| bad skillId | `{"skillId":"abc"}` | `*ParamError{Param:"skillId",Value:"abc"}` |
| defaults | `{"skillId":"1001003"}` | `CharacterId:7, SkillId:1001003, Level:1, MasterLevel:1, Expiration: base.Add(365*24*time.Hour)` |
| level/masterLevel | `{"skillId":"1001003","level":"5","masterLevel":"20"}` | `Level:5, MasterLevel:20` |
| level widened past 127 | `{"skillId":"1001003","level":"200"}` | `Level:200` (portal previously rejected — `ParseInt(...,10,8)` at `executor.go:328`) |
| level out of byte range | `{"skillId":"1001003","level":"256"}` | err containing `out of range for byte` |
| expiration -1 sentinel | `{"skillId":"1001003","expiration":"-1"}` | `Expiration: base.Add(100*365*24*time.Hour)` |
| expiration epoch ms | `{"skillId":"1001003","expiration":"1767225600000"}` | `Expiration: time.UnixMilli(1767225600000)` |
| expiration zero falls back | `{"skillId":"1001003","expiration":"0"}` | `Expiration: base.Add(365*24*time.Hour)` |
| bad expiration | `{"skillId":"1001003","expiration":"abc"}` | `*ParamError{Param:"expiration",Value:"abc"}` |

Compare times with `.Equal(...)`, not `==`.

`libs/atlas-script-core/ops/quest_test.go`, `plain` target as in Task 4:

`TestStartQuest`:

| case | params | defaults | expect |
|---|---|---|---|
| param questId | `{"questId":"2000"}` | `QuestDefaults{}` | `saga.StartQuestPayload{CharacterId:7, WorldId:0, QuestId:2000, NpcId:0}` |
| param npcId | `{"questId":"2000","npcId":"1063017"}` | `QuestDefaults{}` | `NpcId:1063017` |
| default questId | `{}` | `QuestDefaults{QuestId:2000, NpcId:9010000}` | `QuestId:2000, NpcId:9010000` |
| param wins over default | `{"questId":"3000"}` | `QuestDefaults{QuestId:2000, NpcId:9010000}` | `QuestId:3000, NpcId:9010000` |
| no questId anywhere | `{}` | `QuestDefaults{NpcId:9010000}` | err `start_quest: parameter "questId" is required` |
| bad questId | `{"questId":"abc"}` | `QuestDefaults{}` | `*ParamError{Param:"questId",Value:"abc"}` |

`TestStageClearAttemptPq`:

| case | characterId | instanceId | expect |
|---|---|---|---|
| reactor path | 7 | `uuid.MustParse("33333333-3333-3333-3333-333333333333")` | `saga.StageClearAttemptPqPayload{InstanceId: that uuid, CharacterId: 0}` |
| npc path | 7 | `uuid.Nil` | `saga.StageClearAttemptPqPayload{InstanceId: uuid.Nil, CharacterId: 7}` |
| neither set | 0 | `uuid.Nil` | err `stage_clear_attempt_pq: exactly one of instanceId or characterId must be set` |

- [ ] **Step 2: Run the tests to verify they fail**

Run: `cd libs/atlas-script-core && go test ./ops/ -run 'TestCreateSkill|TestUpdateSkill|TestStartQuest|TestStageClearAttemptPq' -v`
Expected: FAIL — undefined symbols.

- [ ] **Step 3: Write `skill.go` and `quest.go`**

`skill.go`, `package ops`. `CreateSkill` and `UpdateSkill` share one unexported helper
so the two contracts cannot drift again:

```go
const (
	opCreateSkill = "create_skill"
	opUpdateSkill = "update_skill"
)

type skillParams struct {
	skillId     uint32
	level       byte
	masterLevel byte
	expiration  time.Time
}

// decodeSkillParams reads the parameter contract shared by create_skill and
// update_skill.
//
// Parameters:
//   - skillId     (required)
//   - level       (optional) defaults to 1.
//   - masterLevel (optional) defaults to 1.
//   - expiration  (optional) epoch milliseconds. The sentinel "-1" means "no
//     expiration" and resolves to 100 years out. A non-positive value other
//     than the sentinel falls back to the 1-year default. Absent defaults to
//     1 year out. (npc-conversations previously ignored this parameter and
//     always used the 1-year default — design §5.4.)
func decodeSkillParams(p map[string]string, r Resolver, cid uint32, op string) (skillParams, error)
```

`decodeSkillParams` uses `requiredInt`+`rangedUint32` for `skillId`,
`optionalInt`+`rangedByte` (default 1) for `level`/`masterLevel`, and for `expiration`
uses `optionalString(..., "expiration", "")` — **not** `Int`, so the `"-1"` sentinel is
never routed through arithmetic evaluation — then:

```go
expiration := now().Add(365 * 24 * time.Hour)
if raw != "" {
	if raw == "-1" {
		expiration = now().Add(100 * 365 * 24 * time.Hour)
	} else {
		expMs, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return skillParams{}, invalidParam(op, "expiration", raw, err)
		}
		if expMs > 0 {
			expiration = time.UnixMilli(expMs)
		}
	}
}
```

This is the one place the package calls `strconv` on a value, and it is deliberate:
`expiration` is a 64-bit epoch that must not go through the int-width resolver, and the
sentinel is compared before any parse. Say so in a comment.

`CreateSkill` returns `newStep(saga.CreateSkill, saga.CreateSkillPayload{...})`;
`UpdateSkill` returns `newStep(saga.UpdateSkill, saga.UpdateSkillPayload{...})`. Neither
payload carries world/channel — see `libs/atlas-saga/payloads.go:289-306`.

`quest.go`, `package ops`:

```go
const (
	opStartQuest          = "start_quest"
	opStageClearAttemptPq = "stage_clear_attempt_pq"
)

// QuestDefaults carries the values a caller resolves from its own state when
// the script omits them. npc-conversations reads them from the Redis
// conversation context (questId and the conversation NPC); portal-actions
// passes the zero value.
type QuestDefaults struct {
	QuestId uint32
	NpcId   uint32
}

// StartQuest builds a StartQuest step.
//
// Parameters:
//   - questId (optional in params, but required overall) falls back to
//     d.QuestId; a zero on both sides is an error.
//   - npcId   (optional) falls back to d.NpcId, else 0.
func StartQuest(p map[string]string, r Resolver, t Target, characterId uint32, d QuestDefaults) (Step, error)

// StageClearAttemptPq builds a StageClearAttemptPq step. It backs both the
// `stage_clear_attempt` (reactor-actions) and `stage_clear_attempt_pq`
// (npc-conversations) script operations — FR-17 keeps both dispatch names
// valid.
//
// The orchestrator branches on which field is set
// (saga-orchestrator/saga/handler.go:3717-3734), so exactly one of them must
// be: reactor-actions resolves the PQ instance over REST and passes
// instanceId; npc-conversations passes uuid.Nil and lets the orchestrator look
// the instance up from characterId.
func StageClearAttemptPq(t Target, characterId uint32, instanceId uuid.UUID) (Step, error)
```

`StartQuest` body: if `p["questId"]` is present use `requiredInt`+`rangedUint32`; else
use `d.QuestId`; if the result is 0 return `missingParam(opStartQuest, "questId")`.
`npcId` likewise, defaulting to `d.NpcId` with no error when both are absent. Payload is
`saga.StartQuestPayload{CharacterId, WorldId: t.Field().WorldId(), QuestId, NpcId}` —
`Rewards` is left nil (neither caller populates it).

`StageClearAttemptPq` body: when `instanceId != uuid.Nil` emit
`saga.StageClearAttemptPqPayload{InstanceId: instanceId}`; else when `characterId != 0`
emit `saga.StageClearAttemptPqPayload{CharacterId: characterId}`; else return
`fmt.Errorf("%s: exactly one of instanceId or characterId must be set", opStageClearAttemptPq)`.

- [ ] **Step 4: Run the tests to verify they pass**

Run: `cd libs/atlas-script-core && go test ./ops/... && go vet ./ops/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-script-core/ops/skill.go libs/atlas-script-core/ops/skill_test.go libs/atlas-script-core/ops/quest.go libs/atlas-script-core/ops/quest_test.go
git commit -m "feat(script-core): add shared skill and quest step builders (FR-17, design 5.4)"
```

---

## Task 6: movement operations

`WarpToPortal` (§5.3 — `mapId` becomes required), `WarpToSavedLocation`, `SaveLocation`,
`StartInstanceTransport`.

### Files

- `libs/atlas-script-core/ops/movement.go` — **new file**
- `libs/atlas-script-core/ops/movement_test.go` — **new file**

Module root: `libs/atlas-script-core`.

Patterns to copy (read-only): `services/atlas-portal-actions/atlas.com/portal/script/executor.go:122-180`
(warp), `:449-491` (start_instance_transport), `:528-576` (save_location, with the
portal-id default), `:579-617` (warp_to_saved_location);
`services/atlas-npc-conversations/atlas.com/npc/conversation/operation_executor.go:1414-1455`
(warp_to_map), `:2507-2529` (start_instance_transport), `:2531-2572` (save_location),
`:2574-2595` (warp_to_saved_location).

**Interfaces produced:**

```go
func WarpToPortal(p map[string]string, r Resolver, t Target, characterId uint32) (Step, error)
func WarpToSavedLocation(p map[string]string, r Resolver, t Target, characterId uint32) (Step, error)
func SaveLocation(p map[string]string, r Resolver, t Target, characterId uint32) (Step, error)
func StartInstanceTransport(p map[string]string, r Resolver, t Target, characterId uint32) (Step, error)
```

**Note for the implementer:** `libs/atlas-saga` also defines
`WarpToRandomPortalPayload` / `saga.WarpToRandomPortal`
(`payloads.go:36-40`, `model.go:117`), used by npc's `warp_to_random_portal` case at
`operation_executor.go:1458+`. That op is **out of scope** — do not touch it and do not
confuse the two Warp payloads.

### Steps

- [ ] **Step 1: Write the failing test**

`libs/atlas-script-core/ops/movement_test.go`, `characterId = 7`, targets:

- `plain` = `NewTargetBuilder(field.NewBuilder(0, 1, 910010000).Build()).Build()`
- `withPortal` = `NewTargetBuilder(field.NewBuilder(0, 1, 910010000).SetInstance(instID).Build()).SetPortalId(7).Build()`

`TestWarpToPortal`:

| case | target | params | expect |
|---|---|---|---|
| missing mapId | plain | `{}` | err `warp_to_portal: parameter "mapId" is required` — **tightening; npc defaulted to 0** (`operation_executor.go:1417-1424`) |
| bad mapId | plain | `{"mapId":"abc"}` | `*ParamError{Param:"mapId",Value:"abc"}` |
| defaults | plain | `{"mapId":"104000000"}` | `saga.WarpToPortalPayload{CharacterId:7, WorldId:0, ChannelId:1, MapId:104000000, Instance:uuid.Nil, PortalId:0, PortalName:""}` |
| portalId | plain | `{"mapId":"104000000","portalId":"3"}` | `PortalId:3` |
| portalName | plain | `{"mapId":"104000000","portalName":"west00"}` | `PortalName:"west00"` |
| instance never carried | withPortal | `{"mapId":"910010000"}` | `Instance: uuid.Nil` — neither caller sets it today; preserved deliberately |
| bad portalId | plain | `{"mapId":"1","portalId":"abc"}` | `*ParamError{Param:"portalId"}` |

`TestWarpToSavedLocation`:

| case | params | expect |
|---|---|---|
| missing locationType | `{}` | err `warp_to_saved_location: parameter "locationType" is required` |
| ok | `{"locationType":"FREE_MARKET"}` | `saga.WarpToSavedLocationPayload{CharacterId:7, WorldId:0, ChannelId:1, LocationType:"FREE_MARKET"}` |

`TestSaveLocation`:

| case | target | params | expect |
|---|---|---|---|
| missing locationType | plain | `{}` | err `save_location: parameter "locationType" is required` |
| defaults, no portal on target | plain | `{"locationType":"FREE_MARKET"}` | `MapId:910010000, PortalId:0` |
| default portal from target | withPortal | `{"locationType":"FREE_MARKET"}` | `MapId:910010000, PortalId:7` (portal-actions' "current portal" default, `executor.go:548`) |
| explicit override | withPortal | `{"locationType":"EVENT","mapId":"104000000","portalId":"2"}` | `LocationType:"EVENT", MapId:104000000, PortalId:2` |
| bad mapId | plain | `{"locationType":"E","mapId":"abc"}` | `*ParamError{Param:"mapId"}` |

`TestStartInstanceTransport`:

| case | params | expect |
|---|---|---|
| missing routeName | `{}` | err `start_instance_transport: parameter "routeName" is required` |
| ok | `{"routeName":"kerning-square-subway-in"}` | `saga.StartInstanceTransportPayload{CharacterId:7, WorldId:0, ChannelId:1, RouteName:"kerning-square-subway-in"}` |

`failureMessage` is deliberately **not** read here — it feeds portal's pending-action
registry (`executor.go:459-466`), not the payload. Assert that a
`{"routeName":"r","failureMessage":"nope"}` call produces a payload identical to the
plain case.

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd libs/atlas-script-core && go test ./ops/ -run 'TestWarpToPortal|TestWarpToSavedLocation|TestSaveLocation|TestStartInstanceTransport' -v`
Expected: FAIL — undefined symbols.

- [ ] **Step 3: Write `libs/atlas-script-core/ops/movement.go`**

`package ops`, constants `opWarpToPortal = "warp_to_portal"`,
`opWarpToSavedLocation = "warp_to_saved_location"`, `opSaveLocation = "save_location"`,
`opStartInstanceTransport = "start_instance_transport"`. Each builder follows the shape
established in Tasks 2–5 and carries its parameter contract as a doc comment.

`WarpToPortal` doc comment must state:

```
// WarpToPortal builds a WarpToPortal step. It backs both the `warp`
// (portal-actions) and `warp_to_map` (npc-conversations) script operations —
// FR-18 keeps both dispatch names valid, and `warp` stays opClassMoving in
// portal's opTable.
//
// Parameters:
//   - mapId      (required) the destination map. npc-conversations previously
//     defaulted this to 0; it is now required (design §5.3). The FR-20 sweep
//     confirmed all 3,377 seeded warp_to_map and 594 warp operations carry it.
//   - portalId   (optional) defaults to 0.
//   - portalName (optional) defaults to "".
//
// Instance is deliberately left uuid.Nil: neither caller populates it today
// and this task does not change destination-field addressing.
```

`SaveLocation` defaults `mapId` to `t.Field().MapId()` and `portalId` to
`t.PortalId()`, which is how portal's "current portal" default and npc's "0" default
become the same code path with no per-service branch.

- [ ] **Step 4: Run the test to verify it passes**

Run: `cd libs/atlas-script-core && go test ./ops/... && go vet ./ops/...`
Expected: PASS — all sixteen builders now exist and are green with no service touched.

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-script-core/ops/movement.go libs/atlas-script-core/ops/movement_test.go
git commit -m "feat(script-core): add shared movement step builders (FR-18, design 5.3)"
```

---

## Task 7: `atlas-map-actions` delegation

### Files

- `services/atlas-map-actions/atlas.com/map-actions/script/executor.go` — delegate the five shared handlers
- `services/atlas-map-actions/atlas.com/map-actions/script/executor_test.go` — update the converged assertions

Module root: `services/atlas-map-actions/atlas.com/map-actions`.

Patterns to copy: the existing test scaffolding at
`services/atlas-map-actions/atlas.com/map-actions/script/executor_test.go:17-50`
(`captureSagaProcessor`, `newTestExecutor`, `newOperation`, `testField`) — reuse it, do
not rewrite it.

**Interfaces consumed:** `ops.NewTargetBuilder`, `ops.DirectResolver`,
`ops.SendMessage`, `ops.SpawnMonster`, `ops.ShowIntro`, `ops.MoveEnvironment`,
`ops.ResetEnvironment`, `ops.PayloadOf`, `Step.AppendTo` (Tasks 1–6).

### Steps

- [ ] **Step 1: Update the failing tests**

In `executor_test.go`, add/extend cases that assert the new behaviour. Each new or
changed assertion gets a comment naming the FR.

Add `TestExecuteSpawnMonsterCarriesInstance` (FR-16 / OQ-3):

| case | field | params | expect on `saga.SpawnMonsterPayload` |
|---|---|---|---|
| same map carries instance | `field.NewBuilder(0,1,910010000).SetInstance(instID).Build()` | `{"monsterId":"100100"}` | `Instance: instID`, `MapId: 910010000` |
| cross map drops instance | same | `{"monsterId":"100100","mapId":"910510202"}` | `Instance: uuid.Nil`, `MapId: 910510202` |
| team now populated | same | `{"monsterId":"100100","team":"1"}` | `Team: 1` (was always 0 — `executor.go:188` hard-coded `uuid.Nil` and omitted `Team`) |

Add `TestExecuteDropMessageAcceptsTypeAlias` (FR-13): params
`{"message":"hi","type":"6"}` produce `MessageType: "BLUE_TEXT"`; params
`{"message":"hi","messageType":"NOTICE","type":"POP_UP"}` produce
`MessageType: "NOTICE"`.

Assert the **step ids are unchanged** in every case: `spawn-%d-%d`
(characterId, monsterId), `message-%d`, `intro-%d`, `move-environment-%s`,
`reset-environment-%d`, and the `initiatedBy` strings `map-action-spawn`,
`map-action-message`, `map-action-intro`, `map-action-move-environment`,
`map-action-reset-environment` (FR-8).

- [ ] **Step 2: Run the tests to verify they fail**

Run: `cd services/atlas-map-actions/atlas.com/map-actions && go test ./script/...`
Expected: FAIL on the instance/team/alias assertions.

- [ ] **Step 3: Delegate the five handlers**

Leave `ExecuteOperation`'s `switch` at `executor.go:37-58` **exactly as it is**. Rewrite
each of the five handler bodies to the shape:

```go
func (e *OperationExecutor) executeSpawnMonster(f field.Model, characterId uint32, op operation.Model) error {
	t := ops.NewTargetBuilder(f).Build()
	st, err := ops.SpawnMonster(op.Params(), ops.DirectResolver{}, t, characterId)
	if err != nil {
		return err
	}
	p, err := ops.PayloadOf[saga.SpawnMonsterPayload](st)
	if err != nil {
		return err
	}

	e.l.Debugf("Spawning monster [%d] at (%d,%d) count [%d] for character [%d].", p.MonsterId, p.X, p.Y, p.Count, characterId)

	s := st.AppendTo(
		saga.NewBuilder().
			SetSagaType(saga.InventoryTransaction).
			SetInitiatedBy("map-action-spawn"),
		fmt.Sprintf("spawn-%d-%d", characterId, p.MonsterId),
	).Build()

	return e.sagaP.Create(s)
}
```

`PayloadOf` is what lets the step id keep embedding the monster id without re-parsing
the param. The other four need no `PayloadOf` — their step ids use only `characterId`,
`f.MapId()`, or the raw `name` param.

Drop every now-unused import (`strconv`, `strings`, `_map`, `uuid`, `field` for
`ParseObjectKind`) that the delegation orphans; keep `field` if the handler signatures
still take `field.Model` (they do).

- [ ] **Step 4: Run the tests to verify they pass**

Run: `cd services/atlas-map-actions/atlas.com/map-actions && go build ./... && go test ./script/... && go vet ./script/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-map-actions/atlas.com/map-actions/script/executor.go services/atlas-map-actions/atlas.com/map-actions/script/executor_test.go
git commit -m "refactor(map-actions): delegate shared script operations to atlas-script-core/ops"
```

---

## Task 8: `atlas-reactor-actions` delegation

### Files

- `services/atlas-reactor-actions/atlas.com/reactor/script/executor.go` — delegate five handlers
- `services/atlas-reactor-actions/atlas.com/reactor/script/executor_test.go` — update the FR-15 assertions

Module root: `services/atlas-reactor-actions/atlas.com/reactor`.

Patterns to copy: the test scaffolding at
`services/atlas-reactor-actions/atlas.com/reactor/script/executor_test.go:17-55`
(`captureSagaProcessor`, `newTestExecutor`, `newOperation`, `testReactorContext`) —
reuse it. The delegation shape is Task 7 Step 3.

**Interfaces consumed:** `ops.NewTargetBuilder(...).SetPosition(rc.X, rc.Y).Build()`,
`ops.DirectResolver`, `ops.SendMessage`, `ops.SpawnMonster`, `ops.MoveEnvironment`,
`ops.ResetEnvironment`, `ops.StageClearAttemptPq`, `Step.AppendTo`.

### Steps

- [ ] **Step 1: Update the failing tests**

Add `TestExecuteSpawnMonsterRejectsBadNumerics` (FR-15). Each of these currently
**silently keeps the default** (`executor.go:200-204,206-211,212-217`) and must now
return an error whose text names the operation, the parameter and the value:

| case | params | expect |
|---|---|---|
| bad x | `{"monsterId":"100100","x":"abc"}` | error, 0 sagas created, message contains `spawn_monster`, `"x"` and `"abc"` |
| bad y | `{"monsterId":"100100","y":"abc"}` | error, 0 sagas created |
| bad count | `{"monsterId":"100100","count":"abc"}` | error, 0 sagas created |
| good defaults still use reactor position | `{"monsterId":"100100"}` | `X` = `testReactorContext().X`, `Y` = `testReactorContext().Y` |

Add `TestExecuteDropMessageAcceptsMessageTypeAlias` (FR-13):
`{"message":"hi","messageType":"NOTICE"}` → `MessageType:"NOTICE"` (reactor previously
read only `type`, `executor.go:374`); `{"message":"hi","type":"5"}` still →
`"PINK_TEXT"`.

Any pre-existing case that asserted the swallow-and-default behaviour is updated in
place with a `// FR-15:` comment explaining the change.

Assert step ids and `initiatedBy` are unchanged: `spawn-%s-%d`
(`rc.Classification`, monsterId), `message-%d`, `move-environment-%s-%s`,
`reset-environment-%s`, `stage-clear-%s-%d`; `reactor-action-spawn`,
`reactor-action-message`, `reactor-action-move-environment`,
`reactor-action-reset-environment`, `reactor-action-stage-clear`.

- [ ] **Step 2: Run the tests to verify they fail**

Run: `cd services/atlas-reactor-actions/atlas.com/reactor && go test ./script/...`
Expected: FAIL — the bad-numeric cases still succeed.

- [ ] **Step 3: Delegate the five handlers**

`switch` at `executor.go:49-90` unchanged. Each handler builds its target as:

```go
t := ops.NewTargetBuilder(rc.Field).SetPosition(rc.X, rc.Y).Build()
```

`SetPosition` is what makes "default to the reactor's coordinates" a property of the
caller rather than a branch inside the shared op.

`executeStageClearAttempt` keeps its REST lookup verbatim and passes the result in:

```go
func (e *OperationExecutor) executeStageClearAttempt(rc ReactorContext, characterId uint32, op operation.Model) error {
	pqInstance, err := e.getPqInstanceByCharacter(characterId)
	if err != nil {
		return fmt.Errorf("failed to get PQ instance for character %d: %w", characterId, err)
	}

	t := ops.NewTargetBuilder(rc.Field).SetPosition(rc.X, rc.Y).Build()
	st, err := ops.StageClearAttemptPq(t, characterId, pqInstance.Id)
	if err != nil {
		return err
	}

	e.l.Debugf("Attempting stage clear: instance=%s, reactor=%s, character=%d", pqInstance.Id, rc.Classification, characterId)

	s := st.AppendTo(
		saga.NewBuilder().
			SetSagaType(saga.InventoryTransaction).
			SetInitiatedBy("reactor-action-stage-clear"),
		fmt.Sprintf("stage-clear-%s-%d", rc.Classification, characterId),
	).Build()

	return e.sagaP.Create(s)
}
```

`getPqInstanceByCharacter` (`executor.go:559-566`) and the seven service-local handlers
(`drop_items`, `spray_items`, `weaken_area_boss`, `kill_all_monsters`,
`update_pq_state`, `hit_reactor`, `broadcast_pq_message`) are untouched.

- [ ] **Step 4: Run the tests to verify they pass**

Run: `cd services/atlas-reactor-actions/atlas.com/reactor && go build ./... && go test ./script/... && go vet ./script/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-reactor-actions/atlas.com/reactor/script/executor.go services/atlas-reactor-actions/atlas.com/reactor/script/executor_test.go
git commit -m "refactor(reactor-actions): delegate shared script operations to atlas-script-core/ops (FR-15)"
```

---

## Task 9: `atlas-portal-actions` delegation

### Files

- `services/atlas-portal-actions/atlas.com/portal/script/executor.go` — delegate ten handlers
- `services/atlas-portal-actions/atlas.com/portal/script/executor_test.go` — extend
- `services/atlas-portal-actions/atlas.com/portal/script/optable.go` — read-only; **must not change**
- `services/atlas-portal-actions/atlas.com/portal/script/optable_test.go` — read-only; **must pass unchanged** (FR-9 regression detector)

Module root: `services/atlas-portal-actions/atlas.com/portal`.

Patterns to copy: `services/atlas-portal-actions/atlas.com/portal/script/executor_test.go:26-45`
(the `TestMain` that stands up `miniredis` + `action.InitRegistry`, required because
`executeWarp`/`executeWarpToSavedLocation` touch the package-global pending-action
registry) and the `fakeSagaProcessor` beside it.

**Interfaces consumed:** `ops.NewTargetBuilder(f).SetPortalId(portalId).Build()`,
`ops.DirectResolver`, `ops.SendMessage`, `ops.ShowHint`, `ops.PlayPortalSound`,
`ops.CreateSkill`, `ops.UpdateSkill`, `ops.StartInstanceTransport`,
`ops.ApplyConsumableEffect`, `ops.SaveLocation`, `ops.WarpToSavedLocation`,
`ops.WarpToPortal`, `ops.StartQuest` + `ops.QuestDefaults{}`, `Step.AppendTo`.

### Steps

- [ ] **Step 1: Update the failing tests**

Add `TestExecuteDropMessageAcceptsTypeAlias` (FR-13): `{"message":"hi","type":"6"}` →
`MessageType:"BLUE_TEXT"`.

Add `TestExecuteCreateSkillWidensLevel` (§5.4): `{"skillId":"1001003","level":"200"}`
now yields `Level: 200` where `strconv.ParseInt(levelStr, 10, 8)` at `executor.go:328`
previously errored.

Add `TestExecuteWarpKeepsTransactionWiring` — assert that after a successful
`{"mapId":"104000000","portalId":"3"}` warp: exactly one saga was created; its
transaction id is non-nil; its timeout equals `warpSagaTimeout`; the step id is
`warp-%d` and `initiatedBy` is `portal-action-warp`; and
`action.GetRegistry()` holds a `PendingAction` under the saga's transaction id with
`Kind: action.KindWarp`. (Read the registry accessor's real name from
`services/atlas-portal-actions/atlas.com/portal/...` and use it as written.)

- [ ] **Step 2: Run the tests to verify they fail**

Run: `cd services/atlas-portal-actions/atlas.com/portal && go test ./script/...`
Expected: FAIL on the alias and level-widening cases.

- [ ] **Step 3: Delegate the ten handlers, below the table**

`optable.go` is **not edited** — the delegation happens inside each `executeX`, so every
entry and every `opClass` stays verbatim (FR-9).

Handlers that just build a step (`play_portal_sound`, `drop_message`, `show_hint`,
`create_skill`, `update_skill`, `apply_consumable_effect`, `save_location`,
`start_quest`) take the Task 7 Step 3 shape, with
`t := ops.NewTargetBuilder(f).SetPortalId(portalId).Build()` where the handler receives
`portalId` (only `executeSaveLocation` does today, `optable.go:68-70`) and
`ops.NewTargetBuilder(f).Build()` otherwise.

`executeStartQuest` passes `ops.QuestDefaults{}` — portal has no conversation state.

The three that mint a transaction id keep that wrapper exactly and delegate only the
payload construction. `executeWarp` becomes:

```go
func (e *OperationExecutor) executeWarp(f field.Model, characterId uint32, op operation.Model) error {
	t := ops.NewTargetBuilder(f).Build()
	st, err := ops.WarpToPortal(op.Params(), ops.DirectResolver{}, t, characterId)
	if err != nil {
		return err
	}
	p, err := ops.PayloadOf[saga.WarpToPortalPayload](st)
	if err != nil {
		return err
	}

	e.l.Debugf("Warping character [%d] to map [%d] portal [%d/%s]", characterId, p.MapId, p.PortalId, p.PortalName)

	// The transaction id is minted here so the pending action can be registered
	// under it. If this warp is suppressed from unlocking the client
	// (consumer.go, task-184 FR-2.3), this registration is what lets
	// handleStatusEventFailed release the player when the warp does not land.
	sagaId := uuid.New()
	action.GetRegistry().AddWithTTL(e.l, e.ctx, sagaId, action.PendingAction{
		CharacterId: characterId,
		WorldId:     f.WorldId(),
		ChannelId:   f.ChannelId(),
		Kind:        action.KindWarp,
	}, pendingActionTTL)

	s := st.AppendTo(
		saga.NewBuilder().
			SetTransactionId(sagaId).
			SetSagaType(saga.InventoryTransaction).
			SetInitiatedBy("portal-action-warp").
			SetTimeout(warpSagaTimeout),
		fmt.Sprintf("warp-%d", characterId),
	).Build()

	return e.sagaP.Create(s)
}
```

`executeWarpToSavedLocation` (`executor.go:579-617`) and
`executeStartInstanceTransport` (`executor.go:449-491`) follow the same pattern, keeping
their own registry calls — note `executeStartInstanceTransport` reads
`params["failureMessage"]` for the registry only, which the shared op does not consume.

`executeBlockPortal` and `executeCancelConsumableEffect` are untouched.

- [ ] **Step 4: Run the tests to verify they pass**

Run: `cd services/atlas-portal-actions/atlas.com/portal && go build ./... && go test ./script/... && go vet ./script/...`
Expected: PASS, including `optable_test.go` with no edits to it.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-portal-actions/atlas.com/portal/script/executor.go services/atlas-portal-actions/atlas.com/portal/script/executor_test.go
git commit -m "refactor(portal-actions): delegate shared script operations to atlas-script-core/ops (FR-9)"
```

---

## Task 10: `atlas-npc-conversations` — resolver adapter and the first seven cases

The npc file is 2,738 lines and the shared cases are interleaved with local ones, so the
fourteen cases split across two tasks. This task lands the adapter and the seven cases
that need no conversation-state default.

### Files

- `services/atlas-npc-conversations/atlas.com/npc/conversation/operation_executor.go` — add the adapter; delegate seven cases
- `services/atlas-npc-conversations/atlas.com/npc/conversation/operation_executor_test.go` — update the converged assertions

Module root: `services/atlas-npc-conversations/atlas.com/npc`.

Patterns to copy: `services/atlas-npc-conversations/atlas.com/npc/conversation/operation_executor_test.go:25-51`
(the `miniredis.RunT` + `InitRegistry` + `tenant.WithContext` +
`NewConversationContextBuilder` + `OperationExecutorImpl{...}` setup) — reuse it.

**Interfaces produced (consumed by Task 11):**

```go
// contextResolver adapts the conversation-context evaluators to ops.Resolver.
type contextResolver struct{ e *OperationExecutorImpl }
func (c contextResolver) String(characterId uint32, param string, raw string) (string, error)
func (c contextResolver) Int(characterId uint32, param string, raw string) (int, error)
func (e *OperationExecutorImpl) resolver() ops.Resolver
func (e *OperationExecutorImpl) target(f field.Model) ops.Target
```

Cases delegated here (line numbers are the current `case` statements):
`send_message` :2127, `spawn_monster` :1825, `show_intro` :2414, `show_hint` :2336,
`play_portal_sound` :2240, `apply_consumable_effect` :2103, `create_skill` :1524.

### Steps

- [ ] **Step 1: Update the failing tests**

Add `TestCreateStepSendMessageDefaultsMessageType` (FR-13): params
`{"message":"hi"}` — previously `errors.New("missing messageType parameter for
send_message operation")` at `operation_executor.go:2133` — now yields
`saga.SendMessagePayload{MessageType: "PINK_TEXT", Message: "hi"}`. Also assert the
`type` alias and the `"6"`→`BLUE_TEXT` mapping.

Add `TestCreateStepSpawnMonsterOptionalPosition` (FR-15/FR-16): params
`{"monsterId":"100100"}` — previously `errors.New("missing x parameter ...")` at
`operation_executor.go:1843` — now yields `X: 0, Y: 0`. With the executor's field built
as `field.NewBuilder(0,1,910010000).SetInstance(instID).Build()`, assert
`Instance: instID` (npc never set it before) and, with
`{"monsterId":"100100","mapId":"910510202"}`, `Instance: uuid.Nil` (OQ-3).

Add `TestCreateStepCreateSkillHonoursExpiration` (§5.4): params
`{"skillId":"1001003","expiration":"-1"}` now produce an `Expiration` more than 50 years
out, where `operation_executor.go:1561` previously hard-coded ~1 year.

Every changed assertion carries a comment naming its FR.

- [ ] **Step 2: Run the tests to verify they fail**

Run: `cd services/atlas-npc-conversations/atlas.com/npc && go test ./conversation/...`
Expected: FAIL on the three new tests.

- [ ] **Step 3: Add the adapter and delegate the seven cases**

Add near `evaluateContextValueAsInt` (`operation_executor.go:179`):

```go
// contextResolver adapts the conversation-context evaluators to ops.Resolver,
// so the shared step builders resolve "{context.xxx}" references and
// arithmetic expressions through exactly the Redis reads this file performs
// today, at the same call sites.
type contextResolver struct{ e *OperationExecutorImpl }

func (c contextResolver) String(characterId uint32, param string, raw string) (string, error) {
	return c.e.evaluateContextValue(characterId, param, raw)
}

func (c contextResolver) Int(characterId uint32, param string, raw string) (int, error) {
	return c.e.evaluateContextValueAsInt(characterId, param, raw)
}

func (e *OperationExecutorImpl) resolver() ops.Resolver { return contextResolver{e: e} }

func (e *OperationExecutorImpl) target(f field.Model) ops.Target {
	return ops.NewTargetBuilder(f).Build()
}
```

npc leaves both `SetPosition` and `SetPortalId` unset, which is what preserves its
"x/y default to 0" and "portalId defaults to 0" behaviour with no per-service branch.

Each of the seven cases collapses to:

```go
case "send_message":
	st, err := ops.SendMessage(operation.Params(), e.resolver(), e.target(f), characterId)
	if err != nil {
		return "", "", "", nil, err
	}
	return stepId, st.Status(), st.Action(), st.Payload(), nil
```

`createSagaForOperation`, `createSagaForOperations`, the `builtStep` post-processing,
`appendCancelAllBuffsStep`, `suppressAwardAssetByCompleteQuest`, and every local
dialogue / cosmetic / quest-progress / pet / storage case are untouched. The
`(stepId, status, action, payload, error)` return shape is preserved exactly (FR-10).

- [ ] **Step 4: Run the tests to verify they pass**

Run: `cd services/atlas-npc-conversations/atlas.com/npc && go build ./... && go test ./conversation/... && go vet ./conversation/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-npc-conversations/atlas.com/npc/conversation/operation_executor.go services/atlas-npc-conversations/atlas.com/npc/conversation/operation_executor_test.go
git commit -m "refactor(npc-conversations): add ops resolver adapter and delegate seven shared cases"
```

---

## Task 11: `atlas-npc-conversations` — the remaining seven cases

### Files

- `services/atlas-npc-conversations/atlas.com/npc/conversation/operation_executor.go` — delegate seven more cases
- `services/atlas-npc-conversations/atlas.com/npc/conversation/operation_executor_test.go` — assert the `warp_to_map` tightening and the quest defaults

Module root: `services/atlas-npc-conversations/atlas.com/npc`.

**Interfaces consumed:** the adapter from Task 10 (`e.resolver()`, `e.target(f)`), plus
`ops.UpdateSkill`, `ops.WarpToPortal`, `ops.WarpToSavedLocation`, `ops.SaveLocation`,
`ops.StartInstanceTransport`, `ops.StartQuest` + `ops.QuestDefaults`,
`ops.StageClearAttemptPq`.

Cases delegated here: `update_skill` :1568, `warp_to_map` :1414,
`warp_to_saved_location` :2574, `save_location` :2531, `start_instance_transport` :2507,
`start_quest` :1964, `stage_clear_attempt_pq` :2641.

Explicitly **not** touched: `warp_to_random_portal` :1458 — a different action
(`saga.WarpToRandomPortal`) and out of scope.

### Steps

- [ ] **Step 1: Update the failing tests**

Add `TestCreateStepWarpToMapRequiresMapId` (§5.3): params `{}` and
`{"portalId":"3"}` now return an error whose text names `warp_to_portal` and `mapId`,
where `operation_executor.go:1417-1424` previously produced
`saga.WarpToPortalPayload{MapId: 0}`. Params `{"mapId":"104000000","portalName":"west00"}`
still yield `MapId: 104000000, PortalName: "west00"`.

Add `TestCreateStepStartQuestUsesContextDefaults`: seed the conversation context with
`questId = "2000"` and NPC id `9010000` via the builder used at
`operation_executor_test.go:40-45`, then call with params `{}` and assert
`saga.StartQuestPayload{QuestId: 2000, NpcId: 9010000}`. With params
`{"questId":"3000"}` assert `QuestId: 3000, NpcId: 9010000`. With an empty context and
params `{}`, assert an error naming `start_quest` and `questId`.

Add `TestCreateStepStageClearAttemptPq`: params `{}` yield
`saga.StageClearAttemptPqPayload{CharacterId: <the test character id>, InstanceId: uuid.Nil}`.

- [ ] **Step 2: Run the tests to verify they fail**

Run: `cd services/atlas-npc-conversations/atlas.com/npc && go test ./conversation/... -run 'TestCreateStepWarpToMap|TestCreateStepStartQuest|TestCreateStepStageClearAttemptPq' -v`
Expected: FAIL.

- [ ] **Step 3: Delegate the seven cases**

Six of them take the Task 10 Step 3 shape. `stage_clear_attempt_pq` passes `uuid.Nil`
for the instance, which is what routes the orchestrator to the character-lookup branch:

```go
case "stage_clear_attempt_pq":
	st, err := ops.StageClearAttemptPq(e.target(f), characterId, uuid.Nil)
	if err != nil {
		return "", "", "", nil, err
	}
	return stepId, st.Status(), st.Action(), st.Payload(), nil
```

`start_quest` keeps its conversation-context read local — the Redis lookup does not move
into the library (design OQ-4) — and hands the result in as defaults:

```go
case "start_quest":
	var d ops.QuestDefaults
	_, hasQuestId := operation.Params()["questId"]
	_, hasNpcId := operation.Params()["npcId"]
	if !hasQuestId || !hasNpcId {
		ctx, err := GetRegistry().GetPreviousContext(e.ctx, characterId)
		if err != nil {
			return "", "", "", nil, fmt.Errorf("failed to get conversation context for start_quest: %w", err)
		}
		if v, exists := ctx.Context()["questId"]; exists {
			parsed, err := strconv.Atoi(v)
			if err != nil {
				return "", "", "", nil, fmt.Errorf("invalid questId in context: %w", err)
			}
			d.QuestId = uint32(parsed)
		}
		d.NpcId = ctx.NpcId()
	}

	st, err := ops.StartQuest(operation.Params(), e.resolver(), e.target(f), characterId, d)
	if err != nil {
		return "", "", "", nil, err
	}
	return stepId, st.Status(), st.Action(), st.Payload(), nil
```

This reads the context at most once per operation, where the current code
(`operation_executor.go:1978,2000`) reads it up to twice. Confirm `ctx.NpcId()`'s return
type at the call site and convert only if it is not already `uint32`.

- [ ] **Step 4: Run the tests to verify they pass**

Run: `cd services/atlas-npc-conversations/atlas.com/npc && go build ./... && go test ./conversation/... && go vet ./conversation/...`
Expected: PASS — including `operation_executor_petevolution_test.go` and
`processor_rps_test.go`.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-npc-conversations/atlas.com/npc/conversation/operation_executor.go services/atlas-npc-conversations/atlas.com/npc/conversation/operation_executor_test.go
git commit -m "refactor(npc-conversations): delegate remaining shared cases to atlas-script-core/ops (FR-18)"
```

---

## Task 12: remove the npc `saga` re-export shim (FR-11)

Deliberately its own commit: mechanical, compiler-checked, and easy to review or revert
alone. This task touches seven files — see `context.md` for why it is not split.

### Files

- `services/atlas-npc-conversations/atlas.com/npc/saga/model.go` — delete the alias blocks; keep `ValidateCharacterStatePayload`
- `services/atlas-npc-conversations/atlas.com/npc/saga/builder.go` — **delete the file**
- `services/atlas-npc-conversations/atlas.com/npc/saga/processor.go` — switch `Create(s Saga)` to the shared type
- `services/atlas-npc-conversations/atlas.com/npc/conversation/operation_executor.go` — import `libs/atlas-saga` directly
- `services/atlas-npc-conversations/atlas.com/npc/conversation/processor.go` — same
- `services/atlas-npc-conversations/atlas.com/npc/conversation/operation_executor_test.go` — same
- `services/atlas-npc-conversations/atlas.com/npc/conversation/operation_executor_petevolution_test.go` — same
- `services/atlas-npc-conversations/atlas.com/npc/conversation/processor_rps_test.go` — same (already imports it as `sharedsaga`)

Module root: `services/atlas-npc-conversations/atlas.com/npc`.

Also read-only: `services/atlas-npc-conversations/atlas.com/npc/saga/producer.go` (stays
as is).

The exhaustive importer list is the five files above, confirmed by
`grep -rln 'atlas-npc-conversations/.*/npc/saga' services/atlas-npc-conversations`:
`conversation/operation_executor.go:9`, `conversation/operation_executor_petevolution_test.go:6`,
`conversation/operation_executor_test.go:5`, `conversation/processor.go:6`,
`conversation/processor_rps_test.go:4`.

### Steps

- [ ] **Step 1: Confirm the shim's exact extent before deleting**

Run: `sed -n '1,15p;84,90p;168,199p' services/atlas-npc-conversations/atlas.com/npc/saga/model.go`
Expected: the type-alias block spans `:10-84`, the const-alias block `:87-171`, and the
genuine local `ValidateCharacterStatePayload` + `ToSharedPayload()` spans `:174-199`.
If the boundaries differ, use what the file actually shows.

- [ ] **Step 2: Delete the aliases**

Rewrite `saga/model.go` to keep only the package clause, the imports it still needs
(`sharedsaga "github.com/Chronicle20/atlas/libs/atlas-saga"` and the local `validation`
package), and `ValidateCharacterStatePayload` + `ToSharedPayload()` unchanged.

Delete `saga/builder.go` entirely — it is two alias lines
(`type Builder = sharedsaga.Builder`, `var NewBuilder = sharedsaga.NewBuilder`).

In `saga/processor.go`, change the `Create` signature from the local `Saga` alias to
`sharedsaga.Saga` and add the import. The types are identical, so no conversion is
needed anywhere.

- [ ] **Step 3: Repoint the five importers**

In each of the five files, replace the local `saga` import with
`saga "github.com/Chronicle20/atlas/libs/atlas-saga"` — keeping the local identifier
`saga` means no call site in the body changes. Where a file already imports
`libs/atlas-saga` under another name (`processor_rps_test.go:4` uses `sharedsaga`),
collapse to one import and adjust the references in that file only.

The local `saga` package is still imported where `ValidateCharacterStatePayload` or the
processor/producer are used — keep that import under a distinct alias (e.g.
`npcsaga`) in those files.

- [ ] **Step 4: Verify the whole module compiles and tests pass**

Run: `cd services/atlas-npc-conversations/atlas.com/npc && go build ./... && go test ./... && go vet ./...`
Expected: PASS. Because the aliases were type identities, the compiler catches every
missed call site; there should be no behaviour change to assert.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-npc-conversations/atlas.com/npc/saga services/atlas-npc-conversations/atlas.com/npc/conversation
git commit -m "refactor(npc-conversations): drop the atlas-saga re-export shim (FR-11)"
```

---

## Task 13: `script-ops-guard.sh` and verify.sh wiring (FR-12)

Without this, FR-12's acceptance criterion is a one-time manual grep that rots on the
next feature.

### Files

- `tools/script-ops-guard.sh` — **new file**
- `tools/script-ops-guard_test.sh` — **new file**
- `tools/verify.sh` — wire the guard in
- `docs/TODO.md` — re-point lines 292-294

Patterns to copy: `tools/npc-conversation-contract-mirror-guard.sh:1-39` (the whole
script shape: `#!/usr/bin/env bash`, `set -euo pipefail`, `ROOT="$(cd "$(dirname
"$0")/.." && pwd)"`, an explanatory header comment, a non-zero exit with a `FAIL —`
message and an `OK —` line on success). Wiring shape:
`tools/verify.sh:722-727` (the `if [ "$ALL" -eq 1 ] || touched '<regex>'; then step
"<name>" ./tools/<x>-guard.sh; else skip "<name> (<reason>)"; fi` idiom).

**Note:** no guard in `tools/` has a `_test.sh` sibling today — this is the first.
`tools/verify.sh:848-866` already discovers and runs `tools/*_test.sh` for changed
`tools/` scripts via `changed_tool_suites`, so no extra wiring is needed for the test
file. Copy the harness style from `tools/lib/analyzer-guard_test.sh` (80 lines: a
`tmpdir` fixture, a pass case, a fail case, a summary line, non-zero exit on any
failure).

### Steps

- [ ] **Step 1: Write the failing guard test**

Create `tools/script-ops-guard_test.sh`, executable, `#!/usr/bin/env bash`,
`set -uo pipefail`. It builds a throwaway tree under `mktemp -d` mirroring
`services/atlas-x/` and runs the guard against it via an env override
(`SCRIPT_OPS_GUARD_ROOT`), asserting:

| case | fixture | expect |
|---|---|---|
| clean tree | a service file with no banned literal | exit 0, stdout contains `OK` |
| banned literal | `services/atlas-map-actions/x.go` containing `saga.SpawnMonsterPayload{` | exit 1, stderr/stdout names the file and `SpawnMonsterPayload` |
| orchestrator exempt | `services/atlas-saga-orchestrator/y.go` containing `saga.SpawnMonsterPayload{` | exit 0 |
| test file not exempt | `services/atlas-map-actions/x_test.go` with the literal | exit 1 — a test that constructs the payload directly is the same duplication |
| comment not counted | a line `// saga.SpawnMonsterPayload{` | exit 0 |

- [ ] **Step 2: Run the test to verify it fails**

Run: `./tools/script-ops-guard_test.sh`
Expected: FAIL — `tools/script-ops-guard.sh: No such file or directory`.

- [ ] **Step 3: Write `tools/script-ops-guard.sh`**

Header comment explains: after task-300, the sixteen shared script operations have
exactly one implementation, in `libs/atlas-script-core/ops`. A service that constructs
one of their payloads directly is a second implementation — the drift this task removed.
`atlas-saga-orchestrator` is exempt because it legitimately *consumes* the payloads.

Body: `ROOT="${SCRIPT_OPS_GUARD_ROOT:-$(cd "$(dirname "$0")/.." && pwd)"}"`, then a
`grep -rn` over `$ROOT/services` for the sixteen literals, excluding
`services/atlas-saga-orchestrator/`, with `//`-comment lines stripped before matching.
The banned list, exactly:

```
saga.SendMessagePayload{
saga.SpawnMonsterPayload{
saga.MoveEnvironmentPayload{
saga.ResetEnvironmentPayload{
saga.ShowIntroPayload{
saga.ShowHintPayload{
saga.PlayPortalSoundPayload{
saga.ApplyConsumableEffectPayload{
saga.CreateSkillPayload{
saga.UpdateSkillPayload{
saga.WarpToPortalPayload{
saga.WarpToSavedLocationPayload{
saga.SaveLocationPayload{
saga.StartInstanceTransportPayload{
saga.StartQuestPayload{
saga.StageClearAttemptPqPayload{
```

Each hit prints `script-ops-guard: FAIL — <repo-relative path>:<line> constructs
<Payload> directly; build it via libs/atlas-script-core/ops`. Exit 1 if any hit,
otherwise print `script-ops-guard: OK — no shared script-operation payload constructed
under services/.` and exit 0.

Note in the header that the local `saga` alias may differ per file, so the guard matches
on the payload type name preceded by any package qualifier — write the pattern as
`[A-Za-z_][A-Za-z0-9_]*\.SendMessagePayload{` etc.

- [ ] **Step 4: Run the guard test and the guard itself**

Run: `./tools/script-ops-guard_test.sh && ./tools/script-ops-guard.sh && shellcheck tools/script-ops-guard.sh tools/script-ops-guard_test.sh`
Expected: the test suite passes; the guard exits 0 against the real tree (Tasks 7–11
removed every direct construction); shellcheck is clean.

If the guard fires against the real tree, that is a genuine miss from Tasks 7–11 — fix
the service, not the guard.

- [ ] **Step 5: Wire it into `tools/verify.sh`**

Insert next to the other guard steps (after the `npc-conversation contract mirror guard`
block at `tools/verify.sh:842-846`):

```sh
if [ "$ALL" -eq 1 ] || touched '^services/atlas-(map|reactor|portal)-actions/|^services/atlas-npc-conversations/|^libs/atlas-script-core/|^tools/script-ops-guard\.sh$'; then
    step "shared script ops guard" ./tools/script-ops-guard.sh
else
    skip "shared script ops guard (no script-operation source changed)"
fi
```

- [ ] **Step 6: Re-point `docs/TODO.md:292-294`**

The three reactor entries currently read:

```
- [ ] Create saga action for boss weakening (`script/executor.go:229,243`)
- [ ] Create saga action for environment object manipulation (`script/executor.go:250,260`)
- [ ] Create saga action for mass monster killing (`script/executor.go:267,272`)
```

All three line references are stale. Replace with the current locations, and **retire**
the environment entry — `move_environment`/`reset_environment` are implemented today and
now live in the library:

```
- [ ] Create saga action for boss weakening (`services/atlas-reactor-actions/atlas.com/reactor/script/executor.go:262`)
- [ ] Create saga action for mass monster killing (`services/atlas-reactor-actions/atlas.com/reactor/script/executor.go:354`)
```

Confirm both line numbers with
`grep -n 'func (e \*OperationExecutor) executeWeakenAreaBoss\|func (e \*OperationExecutor) executeKillAllMonsters' services/atlas-reactor-actions/atlas.com/reactor/script/executor.go`
after Task 8's edits, and use whatever it prints.

- [ ] **Step 7: Commit**

```bash
git add tools/script-ops-guard.sh tools/script-ops-guard_test.sh tools/verify.sh docs/TODO.md
git commit -m "chore(tools): add shared script-ops guard and re-point stale reactor TODOs (FR-12)"
```

---

## Task 14: orchestrator seam test and the FR-20 sweep re-run

### Files

- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/handler_test.go` — add the instance-carrying seam test
- `docs/tasks/task-300-shared-script-operations/sweep-result.md` — **new file**, records the re-run

Module root: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator`.

Read-only: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/handler.go:2087-2124`
(`handleSpawnMonster`); `docs/tasks/task-300-shared-script-operations/sweep-seed-scripts.py`.

No orchestrator production code changes — this is the cross-service seam CLAUDE.md
requires be traced by hand, plus a test that pins the NEW contract.

### Steps

- [ ] **Step 1: Read the seam and confirm the assertion target**

Run: `sed -n '2087,2124p' services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/handler.go`
Expected: `handleSpawnMonster` builds
`field.NewBuilder(payload.WorldId, payload.ChannelId, payload.MapId).SetInstance(payload.Instance).Build()`
and passes it to `h.monsterP.SpawnMonster(f, payload.MonsterId, payload.X, payload.Y, int16(fh), payload.Team)`.
`Instance` therefore selects the field the monster spawns into — which is exactly what
FR-16 changes for the map-actions and npc paths.

- [ ] **Step 2: Write the failing test**

In `handler_test.go`, add `TestHandleSpawnMonsterCarriesInstanceToField`. Copy the
existing handler-test setup shape from the nearest table-driven test in that file (find
it with `grep -n '^func Test' services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/handler_test.go`
and read the closest one that stubs `monsterP`).

| case | `SpawnMonsterPayload` | expect on the `field.Model` reaching `monsterP.SpawnMonster` |
|---|---|---|
| instanced | `{WorldId:0, ChannelId:1, MapId:910010000, Instance: uuid.MustParse("44444444-4444-4444-4444-444444444444"), MonsterId:100100, X:5, Y:6, Team:1, Count:1}` | `f.Instance()` equals that UUID; `f.MapId() == 910010000`; the `team` argument is `1` |
| base field | same with `Instance: uuid.Nil` | `f.Instance() == uuid.Nil` |

Assert `Team` reaches the processor too — map-actions previously omitted it and sent the
zero value, so populating it is additive and worth pinning.

- [ ] **Step 3: Run the test to verify it fails**

Run: `cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go test ./saga/ -run TestHandleSpawnMonsterCarriesInstanceToField -v`
Expected: FAIL — undefined test / the stub does not record the field yet.

- [ ] **Step 4: Make it pass**

Only the test's stub wiring should need writing; `handleSpawnMonster` already forwards
`Instance`. If the test fails for any reason other than missing stub plumbing, stop and
report it — that would mean the seam does not behave as design §6 states.

Run: `cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go test ./saga/... && go vet ./saga/...`
Expected: PASS.

- [ ] **Step 5: Re-run the FR-20 sweep and record the result**

Run from the worktree root: `python3 docs/tasks/task-300-shared-script-operations/sweep-seed-scripts.py`

Write `docs/tasks/task-300-shared-script-operations/sweep-result.md` containing the
command, the date it was run, and the verbatim output. Then compare against design §7's
recorded findings:

- every `spawn_monster` `monsterId`/`x`/`y`/`count`/`mapId` value is a plain integer
- no seeded `spawn_monster` carries `team`
- no `warp`/`warp_to_map` operation is missing `mapId`
- no `drop_message` uses the `type` key or a numeric `5`/`6`
- no `send_message` is missing `messageType`
- `deploy/seed/*/npc-conversations/npc/npc-1063017.json` is the one cross-map spawn

If the re-run disagrees with design §7 on any row, **stop and report it** — a seeded
script now depends on a contract this task tightened, and it must be fixed in this
change per FR-20. Do not paper over a mismatch.

- [ ] **Step 6: Run the full gate**

Run: `./tools/verify.sh`
Expected: exit 0. This is the flagless run CLAUDE.md requires before the branch is done;
`--quick`/`--no-docker` do not count.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/handler_test.go docs/tasks/task-300-shared-script-operations/sweep-result.md
git commit -m "test(saga-orchestrator): pin the SpawnMonster instance seam; record FR-20 sweep re-run"
```

---

## Acceptance checklist (PRD §10)

- [ ] `libs/atlas-script-core/ops` exists with one exported step-builder per shared
      operation, each carrying its parameter contract as a doc comment — Tasks 1–6.
- [ ] No shared operation performs network/Redis/Kafka/REST I/O or calls a saga
      processor — enforced by the package having no such imports; check with
      `go list -deps ./ops` from `libs/atlas-script-core`.
- [ ] All four executors delegate; `./tools/script-ops-guard.sh` exits 0 — Tasks 7–11, 13.
- [ ] `optable_test.go` passes with no edits to it — Task 9.
- [ ] npc still batches many operations into one saga and keeps its
      `(stepId, status, action, payload, error)` handler shape — Tasks 10–11.
- [ ] `saga/model.go` and `saga/builder.go` no longer re-export `libs/atlas-saga` — Task 12.
- [ ] `drop_message`/`send_message` converge on one implementation with both keys, the
      `"5"`/`"6"` mapping, and the `PINK_TEXT` default — Task 2.
- [ ] `spawn_monster` hard-errors on every unparseable numeric with the op name and the
      offending value — Task 3.
- [ ] Every `SpawnMonsterPayload` built by the library carries `Instance` and `Team`,
      including on the map-actions path — Tasks 3, 7, 14.
- [ ] The convergence table exists in `design.md` §5.
- [ ] The FR-20 sweep is re-run and its result recorded — Task 14.
- [ ] Table-driven ops tests cover missing-required, each optional default, each parse
      failure and each alias — Tasks 1–6.
- [ ] The four services' executor tests pass, with each updated assertion carrying an
      FR comment — Tasks 7–11.
- [ ] The `SpawnMonster` seam is traced by hand and pinned by a test — Task 14.
- [ ] Flagless `tools/verify.sh` exits 0 — Task 14 Step 6.
- [ ] `docs/TODO.md:292-294` re-pointed — Task 13 Step 6.
