# Extra-Expression (Emote) Cash Items — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Validate and ownership-gate the emote request path, and carry `duration` / `byItemOption` end-to-end from the serverbound `ExpressionRequest` to the clientbound `CharacterExpression` broadcast.

**Architecture:** Three independent slices sharing one handler file. Slice A adds an emote→cash-item mapping to `libs/atlas-constants/item` and two guards (range, then ownership via a package-var test seam) to `CharacterExpressionHandleFunc`. Slice B widens four Kafka structs, two processors, two producers and one packet constructor so the two fields survive the round trip; the single `int32`→`uint32` narrowing lives at the writer boundary. Slice C corrects one row in the missing-features backlog.

**Tech Stack:** Go 1.x multi-module monorepo (`libs/atlas-constants`, `libs/atlas-packet`, `services/atlas-channel/atlas.com/channel`, `services/atlas-expressions/atlas.com/expressions`), Kafka via `atlas-kafka`, table-driven `testing` tests plus `testify/assert` in `atlas-expressions`.

**Spec:** `docs/tasks/task-247-extra-expression-items/design.md` (PRD: `docs/tasks/task-247-extra-expression-items/prd.md`)

## Global Constraints

- No wire-format change. `ExpressionRequest.Encode/Decode` and `CharacterExpression.Encode/Decode` version gates are untouched. Every expected-byte string in `libs/atlas-packet/character/clientbound/v61_test.go`, `v72_test.go`, `v79_test.go` MUST remain character-for-character identical; only the constructor call gains an argument.
- `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go` MUST NOT be modified. No `CashSlotItemType(6)` arm.
- `services/atlas-expressions/atlas.com/expressions/expression/model.go` and `registry.go` MUST NOT gain `duration` / `byItemOption` fields (FR-3.8).
- No clamp on `duration`. `-1` must reach the wire as `0xFFFFFFFF` (FR-3.5, design §1.3).
- Ownership lookup fails closed: an error is a drop, never a pass (FR-2.5).
- No `*_testhelpers.go` files. Reuse the existing helpers in the `handler` package (`newCashItemUseTestSession`, `newCashItemUseTestSessionForVersion`, `installCapturingProducer`).
- Emote range constant: `MaxEmoteId = 23`, `MaxBaseEmoteId = 7`. Mapping: `itemId = uint32(ClassificationExpression)*10000 + emote - MaxBaseEmoteId - 1`.
- Every `go build ./... && go test ./...` in a task is run from the module root named in that task's Files block. Do not run `tools/verify.sh` from an implementer task.

---

### Task 1: Emote → cash-item mapping constants

**Module root:** `libs/atlas-constants`

### Files

- `libs/atlas-constants/item/expression.go` — **new file**; the constants and two helpers
- `libs/atlas-constants/item/expression_test.go` — **new file**; boundary tests
- `libs/atlas-constants/item/constants.go` — read-only; `type Id uint32` (line 5), `type Classification uint32` (line 7), `ClassificationExpression = Classification(516)` (line 96)

Patterns to copy: `libs/atlas-constants/item/vegas_spell.go` (same file shape: doc comment citing the wz/IDA source, named ids, one predicate) and `libs/atlas-constants/item/vegas_spell_test.go` (package `item_test`, table-driven, `t.Run` per case).

**Interfaces:**
- Consumes: nothing.
- Produces: `item.MaxEmoteId` (`uint32` = 23), `item.MaxBaseEmoteId` (`uint32` = 7), `item.IsExtraExpressionEmote(emote uint32) bool`, `item.ExtraExpressionItemId(emote uint32) (item.Id, bool)`.

- [ ] **Step 1: Write the failing test**

New file `libs/atlas-constants/item/expression_test.go`, package `item_test`, imports `testing` and `github.com/Chronicle20/atlas/libs/atlas-constants/item`.

`TestIsExtraExpressionEmote` — table-driven, one `t.Run` per case, subtest name is the `name` column:

| name | emote | want |
|---|---|---|
| `zero` | 0 | false |
| `base emote upper bound` | 7 | false |
| `first extra` | 8 | true |
| `last extra in v83 data` | 22 | true |
| `gated upper bound` | 23 | true |
| `above client cap` | 24 | false |

`TestExtraExpressionItemId` — table-driven, one `t.Run` per case:

| name | emote | wantId | wantOk |
|---|---|---|---|
| `emote 8 maps to Queasy` | 8 | `item.Id(5160000)` | true |
| `emote 22 maps to the last v83 item` | 22 | `item.Id(5160014)` | true |
| `emote 23 maps to an id no character can own` | 23 | `item.Id(5160015)` | true |
| `base emote is not gated` | 7 | `item.Id(0)` | false |
| `above client cap` | 24 | `item.Id(0)` | false |

`TestExtraExpressionItemIdClassification` — asserts `item.GetClassification(id) == item.ClassificationExpression` for the ids returned at emote 8 and emote 22. Failure message shape: `t.Errorf("ExtraExpressionItemId(%d) = %d, want %d", ...)`.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd libs/atlas-constants && go test ./item/ -run 'TestIsExtraExpressionEmote|TestExtraExpressionItemId' -v`
Expected: FAIL — `undefined: item.IsExtraExpressionEmote`, `undefined: item.ExtraExpressionItemId`.

- [ ] **Step 3: Write minimal implementation**

New file `libs/atlas-constants/item/expression.go`:

```go
package item

// Extra-expression (emote) cash items — Item.wz/Cash/0516.img. These are
// permanent unlocks with no `spec` node: owning one extends the character's
// emote palette beyond the seven free base emotes. They are never consumed.

// MaxEmoteId is the highest emote id a stock client can put on the wire in
// either direction. CWvsContext::SendEmotionChange@0x9f9386 (GMS v95) refuses
// to send above 0x17, and CAvatar::SetEmotion@0x466b00 re-checks the same
// bound before applying a received emotion.
const MaxEmoteId = uint32(23)

// MaxBaseEmoteId is the highest emote every character has without owning a
// cash item. Emotes above it are ClassificationExpression (516) unlocks.
const MaxBaseEmoteId = uint32(7)

// IsExtraExpressionEmote reports whether an emote requires an owned
// ClassificationExpression cash item.
func IsExtraExpressionEmote(emote uint32) bool {
	return emote > MaxBaseEmoteId && emote <= MaxEmoteId
}

// ExtraExpressionItemId maps an extra-expression emote to the cash item that
// unlocks it, reporting false for emotes outside the gated range.
//
// CWvsContext::SendEtcCashItemUseRequest@0xa02c86 and
// CUserLocal::UseFuncKeyMapped case 3u@0x933874 (GMS v95) both compute the
// emote as nItemID % 100 + 8, so the item's index within classification 516
// is emote - 8: emote 8 -> 5160000, emote 22 -> 5160014. Emote 23 yields
// 5160015, which has no entry in v83.1 data — no special case is needed
// because the caller's ownership check fails on it naturally.
func ExtraExpressionItemId(emote uint32) (Id, bool) {
	if !IsExtraExpressionEmote(emote) {
		return Id(0), false
	}
	return Id(uint32(ClassificationExpression)*10000 + emote - MaxBaseEmoteId - 1), true
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd libs/atlas-constants && go build ./... && go test ./...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-constants/item/expression.go libs/atlas-constants/item/expression_test.go
git commit -m "feat(atlas-constants): map extra-expression emotes to their 516xxxx cash items"
```

---

### Task 2: Expose `byItemOption` on `NewCharacterExpression`

**Module roots:** `libs/atlas-packet`, then `services/atlas-channel/atlas.com/channel` (the call-site fix — `atlas-packet` is a `replace` dependency, so widening the constructor breaks atlas-channel's build until the call site is updated).

### Files

- `libs/atlas-packet/character/clientbound/expression.go` — widen `NewCharacterExpression` (line 41)
- `libs/atlas-packet/character/clientbound/expression_test.go` — add the `-1` byte-level test
- `libs/atlas-packet/character/clientbound/v61_test.go:724` — add `, false` to the constructor call; expected bytes unchanged
- `libs/atlas-packet/character/clientbound/v72_test.go:506` — same
- `libs/atlas-packet/character/clientbound/v79_test.go:520` — same
- `services/atlas-channel/atlas.com/channel/kafka/consumer/expression/consumer.go:62` — add `, false` to the constructor call only. **Leave the TODO block at lines 57–61 in place; Task 4 deletes it.**

Patterns to copy: `libs/atlas-packet/character/clientbound/v79_test.go:518-528` (byte-fixture shape: `pt.CreateContext`, `bytes.Equal`, annotated `want` slice).

**Interfaces:**
- Consumes: nothing.
- Produces: `clientbound.NewCharacterExpression(characterId uint32, expression uint32, duration uint32, byItemOption bool) CharacterExpression`.

- [ ] **Step 1: Write the failing test**

Append to `libs/atlas-packet/character/clientbound/expression_test.go` (the file already imports `pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"`; add `"bytes"`):

`TestCharacterExpressionByteOutputV95NegativeDuration` — single case, no table. Builds `ctx := pt.CreateContext("GMS", 95, 1)`, calls `NewCharacterExpression(12345, 8, uint32(int32(-1)), false).Encode(nil, ctx)(nil)` and asserts exactly:

```
0x39, 0x30, 0x00, 0x00, // characterId 12345 (dispatcher Decode4)
0x08, 0x00, 0x00, 0x00, // expression 8 (Decode4 @CUser::OnEmotion 0x8e0150)
0xff, 0xff, 0xff, 0xff, // duration -1 reinterpreted as uint32 (Decode4)
0x00,                   // byItemOption false (Decode1)
```

Failure message shape: `t.Errorf("v95 CharacterExpression wire: got %x want %x", got, want)`.

Add a second case in the same function (or a sibling `TestCharacterExpressionByteOutputV95ByItemOption`) asserting `NewCharacterExpression(12345, 8, uint32(int32(-1)), true)` ends in `0xff, 0xff, 0xff, 0xff, 0x01` — this is the assertion that proves `byItemOption` is reachable through the constructor at all.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd libs/atlas-packet && go test ./character/clientbound/ -run TestCharacterExpressionByteOutputV95 -v`
Expected: FAIL to compile — `too many arguments in call to NewCharacterExpression`.

- [ ] **Step 3: Widen the constructor and fix every call site**

`libs/atlas-packet/character/clientbound/expression.go` — replace the constructor:

```go
func NewCharacterExpression(characterId uint32, expression uint32, duration uint32, byItemOption bool) CharacterExpression {
	return CharacterExpression{characterId: characterId, expression: expression, duration: duration, byItemOption: byItemOption}
}
```

Then in each of `v61_test.go:724`, `v72_test.go:506`, `v79_test.go:520`, change

```go
got := NewCharacterExpression(12345, 5, 3000).Encode(nil, ctx)(nil)
```

to

```go
got := NewCharacterExpression(12345, 5, 3000, false).Encode(nil, ctx)(nil)
```

and **change nothing else in those three files** — the `want` slices stay byte-for-byte identical, which is the demonstration that this is a Go-surface change only.

Then in `services/atlas-channel/atlas.com/channel/kafka/consumer/expression/consumer.go:62`, change `charpkt.NewCharacterExpression(e.CharacterId, e.Expression, 0)` to `charpkt.NewCharacterExpression(e.CharacterId, e.Expression, 0, false)`.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd libs/atlas-packet && go build ./... && go test ./...`
Expected: PASS, including the three unmodified version fixtures.

Run: `cd services/atlas-channel/atlas.com/channel && go build ./...`
Expected: no output.

- [ ] **Step 5: Verify no expected-byte string moved**

Run: `git diff -- libs/atlas-packet/character/clientbound/v61_test.go libs/atlas-packet/character/clientbound/v72_test.go libs/atlas-packet/character/clientbound/v79_test.go`
Expected: exactly three changed lines, each adding `, false` inside a `NewCharacterExpression(...)` call. Any diff touching a `0x` literal is a defect — revert and redo.

- [ ] **Step 6: Commit**

```bash
git add libs/atlas-packet/character/clientbound/expression.go libs/atlas-packet/character/clientbound/expression_test.go libs/atlas-packet/character/clientbound/v61_test.go libs/atlas-packet/character/clientbound/v72_test.go libs/atlas-packet/character/clientbound/v79_test.go services/atlas-channel/atlas.com/channel/kafka/consumer/expression/consumer.go
git commit -m "feat(atlas-packet): expose byItemOption on NewCharacterExpression"
```

---

### Task 3: Thread `duration` / `byItemOption` through atlas-expressions

**Module root:** `services/atlas-expressions/atlas.com/expressions`

### Files

- `services/atlas-expressions/atlas.com/expressions/kafka/message/expression/kafka.go` — add two fields to `Command` and `StatusEvent`
- `services/atlas-expressions/atlas.com/expressions/kafka/consumer/expression/consumer.go` — `handleChangeCommand` passes them to `ChangeAndEmit`
- `services/atlas-expressions/atlas.com/expressions/expression/processor.go` — widen `Change` / `ChangeAndEmit` / `changeInput`
- `services/atlas-expressions/atlas.com/expressions/expression/producer.go` — `expressionEventProvider` gains two parameters
- `services/atlas-expressions/atlas.com/expressions/expression/task.go:50` — `revertExpression` passes `0, false`
- `services/atlas-expressions/atlas.com/expressions/expression/mock/processor.go` — widen `ChangeFunc` / `ChangeAndEmitFunc` and their methods
- `services/atlas-expressions/atlas.com/expressions/expression/processor_test.go` — update the five `p.Change(...)` calls; add the field-propagation test
- `services/atlas-expressions/atlas.com/expressions/expression/model.go` — read-only; **must not change** (FR-3.8)

Patterns to copy: `services/atlas-expressions/atlas.com/expressions/expression/processor_test.go:119-132` (`message.NewBuffer()`, `mb.GetAll()`, `testify/assert`).

**Interfaces:**
- Consumes: nothing from earlier tasks.
- Produces (for Task 4's JSON contract only — no Go coupling):
  `expression.Command` and `expression.StatusEvent` both gain
  `Duration int32 \`json:"duration"\`` and `ByItemOption bool \`json:"byItemOption"\``.

- [ ] **Step 1: Write the failing test**

Append to `services/atlas-expressions/atlas.com/expressions/expression/processor_test.go`:

`TestProcessor_Change_PropagatesDurationAndByItemOption` — single case, setup copied from `TestProcessor_Change_AddsMessageToBuffer` (`setupProcessorTest`, `setupTestTenant`, `setupTestContext`, `setupTestLogger`, `message.NewBuffer()`, `field.NewBuilder(0, 1, 100000000).Build()`).

Calls `p.Change(mb, uuid.New(), 1000, f, 8, int32(-1), true)`, then reads `mb.GetAll()[expression2.EnvExpressionEvent]`, asserts exactly one message, `json.Unmarshal`s its `.Value` into `expression2.StatusEvent`, and asserts:

| field | expected |
|---|---|
| `Expression` | `uint32(8)` |
| `Duration` | `int32(-1)` |
| `ByItemOption` | `true` |

(the file must import `encoding/json` and `expression2 "atlas-expressions/kafka/message/expression"`).

`TestRevertExpressionEmitsZeroDurationAndFalseByItemOption` — asserts the revert provider's payload, built directly rather than through the Kafka producer: call `expressionEventProvider(uuid.New(), 1000, f, 0, 0, false)()`, assert no error and exactly one message, unmarshal into `expression2.StatusEvent`, and assert `Expression == 0`, `Duration == 0`, `ByItemOption == false` (FR-3.7).

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-expressions/atlas.com/expressions && go test ./expression/ -run 'PropagatesDurationAndByItemOption|RevertExpressionEmits' -v`
Expected: FAIL to compile — `too many arguments in call to p.Change`.

- [ ] **Step 3: Write the implementation**

`kafka/message/expression/kafka.go` — add to **both** `StatusEvent` and `Command`, after `Expression`:

```go
	Duration      int32      `json:"duration"`
	ByItemOption  bool       `json:"byItemOption"`
```

`expression/producer.go`:

```go
func expressionEventProvider(transactionId uuid.UUID, characterId uint32, field field.Model, expressionId uint32, duration int32, byItemOption bool) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &expression.StatusEvent{
		TransactionId: transactionId,
		CharacterId:   characterId,
		WorldId:       field.WorldId(),
		ChannelId:     field.ChannelId(),
		MapId:         field.MapId(),
		Instance:      field.Instance(),
		Expression:    expressionId,
		Duration:      duration,
		ByItemOption:  byItemOption,
	}
	return producer.SingleMessageProvider(key, value)
}
```

`expression/processor.go` — widen the interface methods, the impl, and `changeInput`:

```go
	// Change changes the expression for a character
	Change(mb *message.Buffer, transactionId uuid.UUID, characterId uint32, field field.Model, expression uint32, duration int32, byItemOption bool) (Model, error)
	// ChangeAndEmit changes the expression for a character and emits an event
	ChangeAndEmit(transactionId uuid.UUID, characterId uint32, field field.Model, expression uint32, duration int32, byItemOption bool) (Model, error)
```

```go
type changeInput struct {
	transactionId uuid.UUID
	characterId   uint32
	field         field.Model
	expression    uint32
	duration      int32
	byItemOption  bool
}
```

`ProcessorImpl.Change` still calls `GetRegistry().add(p.ctx, characterId, field, expression)` unchanged — the registry `Model` does not learn about the new fields (FR-3.8) — and passes the two new values through to `expressionEventProvider`. `ChangeAndEmit` fills the two new `changeInput` fields and its inner closure forwards `input.duration, input.byItemOption`.

`expression/task.go:50` — `revertExpression` becomes:

```go
	// Revert always restores the neutral face: expression 0 with no duration
	// and no item option (FR-3.7). The registry Model deliberately does not
	// persist the original duration/byItemOption, so there is nothing to
	// replay here.
	return producer.ProviderImpl(l)(ctx)(expression.EnvExpressionEvent)(expressionEventProvider(transactionId, exp.CharacterId(), exp.Field(), 0, 0, false))
```

`kafka/consumer/expression/consumer.go` — `handleChangeCommand`'s last line:

```go
	_, _ = processor.ChangeAndEmit(c.TransactionId, c.CharacterId, f, c.Expression, c.Duration, c.ByItemOption)
```

`expression/mock/processor.go` — widen `ChangeFunc`, `ChangeAndEmitFunc` and the two methods identically (add `duration int32, byItemOption bool` after `expr uint32`, and forward them).

Update the five existing `p.Change(mb, ...)` calls in `processor_test.go` (lines 90, 112, 129, 188, 210) by appending `, 0, false`.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd services/atlas-expressions/atlas.com/expressions && go build ./... && go test ./...`
Expected: PASS.

- [ ] **Step 5: Verify the registry model did not change**

Run: `git diff --stat -- services/atlas-expressions/atlas.com/expressions/expression/model.go services/atlas-expressions/atlas.com/expressions/expression/registry.go`
Expected: empty output.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-expressions/atlas.com/expressions
git commit -m "feat(atlas-expressions): carry duration and byItemOption through the expression command and event"
```

---

### Task 4: Thread `duration` / `byItemOption` through atlas-channel

**Module root:** `services/atlas-channel/atlas.com/channel`

### Files

- `services/atlas-channel/atlas.com/channel/kafka/message/expression/kafka.go` — add two fields to `Command` and `Event`
- `services/atlas-channel/atlas.com/channel/character/expression/producer.go` — `SetCommandProvider` gains two parameters
- `services/atlas-channel/atlas.com/channel/character/expression/processor.go` — widen `Processor.Change`
- `services/atlas-channel/atlas.com/channel/character/expression/producer_test.go` — **new file**
- `services/atlas-channel/atlas.com/channel/kafka/consumer/expression/consumer.go` — pass the fields to the writer; **delete** the TODO block at lines 57–61
- `services/atlas-channel/atlas.com/channel/socket/handler/character_expression.go` — forward `p.Duration()` / `p.ByItemOption()` into `Change`. **Only the `Change` call changes in this task; Task 5 adds the guards.**

Patterns to copy: `services/atlas-expressions/atlas.com/expressions/expression/producer.go` (the provider shape being mirrored).

**Interfaces:**
- Consumes: `clientbound.NewCharacterExpression(characterId, expression, duration uint32, byItemOption bool)` from Task 2. The JSON keys `duration` / `byItemOption` must match Task 3's tags exactly.
- Produces: `expression.Processor.Change(characterId uint32, f field.Model, expression uint32, duration int32, byItemOption bool) error` — Task 5 calls this.

- [ ] **Step 1: Write the failing test**

New file `services/atlas-channel/atlas.com/channel/character/expression/producer_test.go`, package `expression`.

`TestSetCommandProviderCarriesDurationAndByItemOption` — table-driven, one `t.Run` per case. Each case builds `f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(100000000)).Build()`, calls `SetCommandProvider(1000, f, tc.expression, tc.duration, tc.byItemOption)()`, asserts no error and exactly one message, `json.Unmarshal`s `msgs[0].Value` into `expression2.Command`, and checks the four fields.

| case | expression | duration | byItemOption | expect Duration | expect ByItemOption |
|---|---|---|---|---|---|
| `v95 extra expression` | 8 | -1 | false | -1 | false |
| `item option set` | 12 | 3000 | true | 3000 | true |
| `pre-v95 zero values` | 5 | 0 | false | 0 | false |

Also assert `cmd.CharacterId == 1000`, `cmd.Expression == tc.expression`, `cmd.MapId == _map.Id(100000000)` in every case.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./character/expression/ -v`
Expected: FAIL to compile — `too many arguments in call to SetCommandProvider`.

- [ ] **Step 3: Write the implementation**

`kafka/message/expression/kafka.go` — add to **both** `Command` and `Event`, after `Expression`:

```go
	Duration     int32      `json:"duration"`
	ByItemOption bool       `json:"byItemOption"`
```

`character/expression/producer.go`:

```go
func SetCommandProvider(characterId uint32, f field.Model, expression uint32, duration int32, byItemOption bool) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &expression2.Command{
		CharacterId:  characterId,
		WorldId:      f.WorldId(),
		ChannelId:    f.ChannelId(),
		MapId:        f.MapId(),
		Instance:     f.Instance(),
		Expression:   expression,
		Duration:     duration,
		ByItemOption: byItemOption,
	}
	return producer.SingleMessageProvider(key, value)
}
```

`character/expression/processor.go` — widen both the interface method and the impl:

```go
type Processor interface {
	Change(characterId uint32, f field.Model, expression uint32, duration int32, byItemOption bool) error
}
```

```go
func (p *ProcessorImpl) Change(characterId uint32, f field.Model, expression uint32, duration int32, byItemOption bool) error {
	p.l.Debugf("Changing character [%d] expression to [%d].", characterId, expression)
	return producer.ProviderImpl(p.l)(p.ctx)(expression2.EnvExpressionCommand)(SetCommandProvider(characterId, f, expression, duration, byItemOption))
}
```

(the existing Debugf logs `f.MapId()` where it says "expression" — correct it to `expression` as shown while the line is being edited.)

`kafka/consumer/expression/consumer.go` — delete the whole five-line `task-028 follow-up` comment block (lines 57–61; it opens with the deferred-work marker `grep -n "task-028 follow-up"` finds) and replace the announce line with:

```go
		// e.Duration is int32 and is -1 for every emote a GMS v95 client
		// originates (CWvsContext::SendEmotionChange@0xa02c86 and
		// CUserLocal::UseFuncKeyMapped case 3u@0x933874 both pass nDuration
		// = -1). Go's int32 -> uint32 conversion preserves the bit pattern, so
		// -1 reaches the wire as FF FF FF FF — exactly what the sending client
		// encoded. This is deliberate, not a lost sign.
		err := _map.NewProcessor(l, ctx).ForOtherSessionsInMap(sc.Field(e.MapId, e.Instance), e.CharacterId, session.Announce(l)(ctx)(wp)(charpkt.CharacterExpressionWriter)(charpkt.NewCharacterExpression(e.CharacterId, e.Expression, uint32(e.Duration), e.ByItemOption).Encode))
```

`socket/handler/character_expression.go` — the last line becomes:

```go
		_ = expression.NewProcessor(l, ctx).Change(s.CharacterId(), s.Field(), p.Emote(), p.Duration(), p.ByItemOption())
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./...`
Expected: PASS.

- [ ] **Step 5: Verify the TODO is gone**

Run: `grep -rn "task-028 follow-up" services/atlas-channel/atlas.com/channel/kafka/consumer/expression/`
Expected: no output (exit 1).

- [ ] **Step 6: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/kafka/message/expression/kafka.go services/atlas-channel/atlas.com/channel/character/expression services/atlas-channel/atlas.com/channel/kafka/consumer/expression/consumer.go services/atlas-channel/atlas.com/channel/socket/handler/character_expression.go
git commit -m "feat(atlas-channel): carry duration and byItemOption end-to-end for expressions"
```

---

### Task 5: Range check and extra-expression ownership gate

**Module root:** `services/atlas-channel/atlas.com/channel`

### Files

- `services/atlas-channel/atlas.com/channel/socket/handler/character_expression.go` — the two guards and the ownership seam
- `services/atlas-channel/atlas.com/channel/socket/handler/character_expression_test.go` — **new file**
- `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:1009-1036` — read-only; the package-var seam precedent
- `services/atlas-channel/atlas.com/channel/socket/handler/character_interaction.go:122-131` — read-only; the cash-ownership idiom being copied
- `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use_test.go:23-84` — read-only; source of `mustTenant`, `newCashItemUseTestSession`, `newCashItemUseTestSessionForVersion`
- `services/atlas-channel/atlas.com/channel/socket/handler/cash_item_gachapon_test.go:50-60` — read-only; source of `installCapturingProducer`
- `services/atlas-channel/atlas.com/channel/compartment/model.go:59` — read-only; `FindFirstByItemId`

**Interfaces:**
- Consumes: `item.MaxEmoteId`, `item.ExtraExpressionItemId` (Task 1); `expression.Processor.Change(characterId, f, expression, duration, byItemOption)` (Task 4).
- Produces: package-var `expressionItemOwnedFunc func(l logrus.FieldLogger, ctx context.Context, characterId uint32, itemId item.Id) (bool, error)`.

- [ ] **Step 1: Write the failing test**

New file `services/atlas-channel/atlas.com/channel/socket/handler/character_expression_test.go`, package `handler`.

Two local helpers in this file:

- `installExpressionItemOwnedSeam(t *testing.T, owns bool, err error) (*[]item.Id, func())` — swaps `expressionItemOwnedFunc` for one that appends the requested `itemId` to a slice and returns `(owns, err)`, returning the slice pointer (the call log) and a restore func. Shape copied from `installCashItemInSlotSeam` (`character_cash_item_use_test.go:34`).
- `expressionRequestBytes(l logrus.FieldLogger, emote uint32) *request.Reader` — writes `WriteInt(emote)` with a `response.NewWriter(l)` and wraps it as `request.Request` / `request.NewRequestReader(&req, 0)`, exactly as `kiteUseRequest` does (`character_cash_item_use_kite_test.go:58-64`). GMS v83 sessions decode the emote only, so four bytes is the whole body.

`TestCharacterExpressionHandleFunc_Gate` — table-driven, one `t.Run` per case. Each case: `installCapturingProducer()`, `installExpressionItemOwnedSeam(t, tc.owns, tc.seamErr)`, `newCashItemUseTestSession(t, 555)` (GMS v83), then invoke `CharacterExpressionHandleFunc(logrus.New(), ctx, nil)(s, expressionRequestBytes(l, tc.emote), map[string]interface{}{})` and assert against `(*captured)[expression2.EnvExpressionCommand]` (import `expression2 "atlas-channel/kafka/message/expression"`).

| case | emote | seam returns | expect seam calls | expect commands emitted |
|---|---|---|---|---|
| `base emote skips the lookup` | 5 | n/a | 0 | 1 |
| `base emote upper bound skips the lookup` | 7 | n/a | 0 | 1 |
| `owned extra expression is forwarded` | 8 | `(true, nil)` | 1, with `item.Id(5160000)` | 1 |
| `unowned extra expression is dropped` | 8 | `(false, nil)` | 1, with `item.Id(5160000)` | 0 |
| `lookup error fails closed` | 8 | `(false, errors.New("boom"))` | 1, with `item.Id(5160000)` | 0 |
| `gated upper bound is dropped when unowned` | 23 | `(false, nil)` | 1, with `item.Id(5160015)` | 0 |
| `out of range never reaches the lookup` | 24 | n/a | 0 | 0 |

For the forwarded cases also unmarshal the single captured message into `expression2.Command` and assert `Expression == tc.emote` and `CharacterId == 555`.

`TestCharacterExpressionHandleFunc_ForwardsDurationAndByItemOption` — a single GMS v95 case proving the Task 4 wiring survives the gate. Session from `newCashItemUseTestSessionForVersion(t, 555, "GMS", 95)`; the request body is `WriteInt(8)`, `WriteInt32(-1)`, `WriteBool(false)`. Seam returns `(true, nil)`. Asserts the captured `expression2.Command` has `Duration == int32(-1)` and `ByItemOption == false`.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./socket/handler/ -run TestCharacterExpressionHandleFunc -v`
Expected: FAIL to compile — `undefined: expressionItemOwnedFunc`.

- [ ] **Step 3: Write the implementation**

Replace `services/atlas-channel/atlas.com/channel/socket/handler/character_expression.go` in full:

```go
package handler

import (
	"atlas-channel/character"
	"atlas-channel/character/expression"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	character2 "github.com/Chronicle20/atlas/libs/atlas-packet/character/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

// expressionItemOwnedFunc is a test seam for the extra-expression ownership
// check (package-var injection precedent: cashItemInSlotFunc in
// character_cash_item_use.go). Handler tests must not require a live character
// service to assert which branch a request reached. It returns only a bool
// because nothing downstream needs the asset itself.
var expressionItemOwnedFunc = func(l logrus.FieldLogger, ctx context.Context, characterId uint32, itemId item.Id) (bool, error) {
	cp := character.NewProcessor(l, ctx)
	c, err := cp.GetById(cp.InventoryDecorator)(characterId)
	if err != nil {
		return false, err
	}
	_, ok := c.Inventory().Cash().FindFirstByItemId(uint32(itemId))
	return ok, nil
}

func CharacterExpressionHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := character2.ExpressionRequest{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())

		emote := p.Emote()

		// CWvsContext::SendEmotionChange@0x9f9386 (GMS v95) refuses to send an
		// emotion above 0x17, and the receiving CAvatar::SetEmotion@0x466b00
		// drops one anyway, so a larger value cannot come from a stock client.
		if emote > item.MaxEmoteId {
			l.Warnf("Character [%d] requested out-of-range expression [%d]. Dropping.", s.CharacterId(), emote)
			return
		}

		// Extra expressions require the matching 516xxxx cash item. The stock
		// client applies the same rule itself: CUserLocal::UseFuncKeyMapped
		// case 3u@0x933884 gates on CWvsContext::IsExist(nItemID) before
		// sending. Fail closed on a lookup error — a broken character service
		// must not read as ownership.
		if itemId, ok := item.ExtraExpressionItemId(emote); ok {
			owns, err := expressionItemOwnedFunc(l, ctx, s.CharacterId(), itemId)
			if err != nil {
				l.WithError(err).Warnf("Unable to verify character [%d] owns item [%d] for expression [%d]. Dropping.", s.CharacterId(), itemId, emote)
				return
			}
			if !owns {
				l.Warnf("Character [%d] requested expression [%d] without owning item [%d]. Dropping.", s.CharacterId(), emote, itemId)
				return
			}
		}

		_ = expression.NewProcessor(l, ctx).Change(s.CharacterId(), s.Field(), emote, p.Duration(), p.ByItemOption())
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./...`
Expected: PASS.

- [ ] **Step 5: Verify the cash-item-use handler was not touched**

Run: `git diff --stat main -- services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go`
Expected: empty output.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/socket/handler/character_expression.go services/atlas-channel/atlas.com/channel/socket/handler/character_expression_test.go
git commit -m "feat(atlas-channel): range-check and ownership-gate extra expression emotes"
```

---

### Task 6: Populate the expression command's `TransactionId`

**Module root:** `services/atlas-channel/atlas.com/channel`

Kept as its own commit so review can drop it without unpicking Tasks 4 or 5 (design §4.2).

### Files

- `services/atlas-channel/atlas.com/channel/kafka/message/expression/kafka.go` — add `TransactionId` to `Command`
- `services/atlas-channel/atlas.com/channel/character/expression/producer.go` — set it in `SetCommandProvider`
- `services/atlas-channel/atlas.com/channel/character/expression/producer_test.go` — add the assertion
- `services/atlas-expressions/atlas.com/expressions/kafka/message/expression/kafka.go` — read-only; already declares `TransactionId uuid.UUID \`json:"transactionId"\``, which is the field being populated for the first time

**Interfaces:**
- Consumes: `SetCommandProvider(characterId, f, expression, duration, byItemOption)` from Task 4.
- Produces: nothing further.

- [ ] **Step 1: Write the failing test**

Add to `services/atlas-channel/atlas.com/channel/character/expression/producer_test.go`:

`TestSetCommandProviderSetsTransactionId` — calls `SetCommandProvider(1000, f, 8, -1, false)()` twice, unmarshals both messages into `expression2.Command`, and asserts:

- `cmd.TransactionId != uuid.Nil` for each
- the two transaction ids differ (proving a fresh `uuid.New()` per command, not a package-level constant)

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./character/expression/ -run TestSetCommandProviderSetsTransactionId -v`
Expected: FAIL to compile — `cmd.TransactionId undefined`.

- [ ] **Step 3: Write the implementation**

`kafka/message/expression/kafka.go` — add as the first field of `Command` (mirroring `atlas-expressions`' field order):

```go
	TransactionId uuid.UUID  `json:"transactionId"`
```

`character/expression/producer.go` — set it, with the reason recorded:

```go
	// atlas-expressions' Command has always declared transactionId but this
	// producer never set it, so every command arrived with the zero UUID and
	// carried it onto every StatusEvent. One id per command emitted.
	value := &expression2.Command{
		TransactionId: uuid.New(),
		...
	}
```

(add `"github.com/google/uuid"` to the imports).

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/kafka/message/expression/kafka.go services/atlas-channel/atlas.com/channel/character/expression/producer.go services/atlas-channel/atlas.com/channel/character/expression/producer_test.go
git commit -m "fix(atlas-channel): populate transactionId on the expression command"
```

---

### Task 7: Correct the missing-features backlog entry

### Files

- `docs/research/missing-features/items-and-consumables.md:49` — the row asserting a missing type-6 dispatch arm

- [ ] **Step 1: Locate the claim**

Run: `grep -n "Extra-expression" docs/research/missing-features/items-and-consumables.md`
Expected: one hit. The PRD cites lines 31 and 80 from an older revision — those line numbers are stale; the claim now appears once. If the grep returns a different line number than 49, use what it returns.

- [ ] **Step 2: Rewrite the entry**

Delete the row

```
| Extra-expression (emote) items | `ClassificationExpression` → type 6 | — | S |
```

from the "Remaining one-off cash types" table (whose preamble — *"All mapped in `GetCashSlotItemType` but unimplemented (fall to warn)"* — is the false premise), and add a short subsection immediately after that table:

```markdown
#### Extra-expression (emote) items — closed by task-247, and not a cash-item-use gap

Extra-expression items (`ClassificationExpression` = 516, `Item.wz/Cash/0516.img`,
`05160000`–`05160014`) are **not** routed through the cash-item-use handler, so a
`CashSlotItemType(6)` dispatch arm would be dead code.
`CDraggableItem::OnDoubleClicked` @`0x50814b` (GMS v95) checks
`get_etc_cash_item_type` first and calls
`CWvsContext::SendEtcCashItemUseRequest` @`0x508165`, whose `case 6:` @`0xa02c86`
issues `CWvsContext::SendEmotionChange(nItemID % 100 + 8, 0, -1)`. The keyboard
path is the same: `CUserLocal::UseFuncKeyMapped` `case 3u` @`0x933874`. GMS v87
matches (`SendEtcCashItemUseRequest` @`0xab4f91`).

The real gap was on the emote path — `CharacterExpressionHandleFunc` accepted any
emote id with no range check and no ownership check — and task-247 closes it.
```

- [ ] **Step 3: Verify the false premise is gone**

Run: `grep -n "ClassificationExpression. → type 6" docs/research/missing-features/items-and-consumables.md`
Expected: no output (exit 1).

- [ ] **Step 4: Commit**

```bash
git add docs/research/missing-features/items-and-consumables.md
git commit -m "docs(missing-features): correct the extra-expression cash item premise"
```

---

## Post-plan verification (controller, not an implementer task)

- Flagless `tools/verify.sh` exits 0.
- `git diff main --stat` shows `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go` absent.
- `git diff main -- libs/atlas-packet/character/clientbound/v61_test.go libs/atlas-packet/character/clientbound/v72_test.go libs/atlas-packet/character/clientbound/v79_test.go` shows only the three `, false` additions.
- `backend-guidelines-reviewer` over the changed Go packages in both services; `plan-adherence-reviewer` over this plan.
