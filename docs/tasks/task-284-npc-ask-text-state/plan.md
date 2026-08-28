# Implementation Plan — NPC Free-Text Input State (`askText`)

Task: `task-284-npc-ask-text-state`
Phase: 3 (Plan)
Inputs: [`prd.md`](prd.md), [`design.md`](design.md)
Companion: [`context.md`](context.md) — grounding corrections, verified facts, open dependencies

---

## Reading this plan

Tasks 1–11 are the critical path and are strictly ordered: each depends on the
one before it compiling. Tasks 12–21 depend on 1–11 but are independent of each
other and may run in any order.

`context.md` §1 records four places where `design.md` describes something the
repository does not actually hold. **The plan follows the repository.** Two of
those corrections change work materially and are restated at their task:

- The new operation is **`local:get_quest_progress`** (Task 11), because
  `operation_executor.go:317-320` routes any un-prefixed operation to
  atlas-saga-orchestrator, which cannot write conversation context.
- atlas-npc-conversations **already has a quest client** at
  `conversation/quest/status/` on `RootUrlFor(ctx, "QUEST")`, so Task 10 adds a
  `progress` sibling rather than the new top-level `quest/` package design §8
  describes. No deployment env change is needed.

A third correction matters to Task 6: `AskTextConversationDetail.Min`/`Max` are
`uint16` written with `WriteShort`, and `Def` is a `string`, not a number
(`libs/atlas-packet/npc/clientbound/conversation.go:171-187`). `askNumber`'s
`uint32` types must **not** be copied.

---

## Task 1: Add `Text` to the continue-conversation command contract

Additive `Text string \`json:"text"\`` on the continue-conversation body in all
three copies. Nothing reads it yet; this task only moves the contract so the
later tasks have somewhere to put the value.

`tools/npc-conversation-contract-mirror-guard.sh` diffs the
atlas-npc-conversations and atlas-saga-orchestrator files from their `package`
clause onward and fails on **any** difference, so those two must be edited
identically — same field, same tag, same position, same surrounding whitespace.
The atlas-channel copy is a different shape and is not covered by the guard.

### Files

- `services/atlas-npc-conversations/atlas.com/npc/kafka/message/npc/kafka.go` — add `Text` to `CommandConversationContinueBody` (lines 64-70)
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/message/npc/kafka.go` — add `Text` to `CommandConversationContinueBody` (lines 69-73); must end byte-identical to the file above from `package` onward
- `services/atlas-channel/atlas.com/channel/kafka/message/npc/kafka.go` — add `Text` to `ContinueConversationCommandBody` (lines 33-37)

Module roots: `services/atlas-npc-conversations/atlas.com/npc`,
`services/atlas-saga-orchestrator/atlas.com/saga-orchestrator`,
`services/atlas-channel/atlas.com/channel`.

- [ ] **Step 1: Add the field to all three bodies**

Each struct currently reads:

```go
type CommandConversationContinueBody struct {
	Action          byte  `json:"action"`
	LastMessageType byte  `json:"lastMessageType"`
	Selection       int32 `json:"selection"`
}
```

Append one field, in this position, with this tag:

```go
	Text            string `json:"text"`
```

The atlas-channel struct is named `ContinueConversationCommandBody` but has the
same three fields; append the same line there.

- [ ] **Step 2: Verify the mirror guard**

```sh
./tools/npc-conversation-contract-mirror-guard.sh
```

Must print `OK — both copies identical.` If it prints a diff, the two files
disagree somewhere other than the leading doc comment — align them exactly
rather than adjusting the guard.

- [ ] **Step 3: Build all three modules**

```sh
cd services/atlas-npc-conversations/atlas.com/npc && go build ./...
cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go build ./...
cd services/atlas-channel/atlas.com/channel && go build ./...
```

---

## Task 2: `AskTextType`, `AskTextModel`, `AskTextMatchModel`, and their JSONB codec

The immutable models and their storage serialization. No builder yet — Task 3
owns that — so this task's models are constructed only inside `model_json.go`'s
unmarshal path.

### Files

- `services/atlas-npc-conversations/atlas.com/npc/conversation/model.go` — `AskTextType` constant, `StateModel.askText` field + accessor, `AskTextModel`, `AskTextMatchModel`
- `services/atlas-npc-conversations/atlas.com/npc/conversation/model_json.go` — marshal/unmarshal for both new models, and the `StateModel` envelope
- `services/atlas-npc-conversations/atlas.com/npc/conversation/model_json_test.go` — new file

Patterns to copy: `conversation/model.go:602-640` (`AskNumberModel` — struct
with unexported fields at 602-610, accessors at 612-640);
`conversation/model_json.go:350-382` (`AskNumberModel` marshal at 352-361,
unmarshal at 363-382).

Module root: `services/atlas-npc-conversations/atlas.com/npc`.

- [ ] **Step 1: Write the failing test**

`TestAskTextJSONRoundTrip` in `model_json_test.go` — table-driven. There is no
existing test file in this package for `model_json.go`; the closest builder
setup to copy is `conversation/rest_transform_test.go:373-395` (the `AskNumber`
round-trip), but this test constructs `AskTextModel` through the unmarshal path
only, since Task 3 introduces the builder.

Marshal each fixture, unmarshal the result, and assert
`reflect.DeepEqual` against the original.

| case | `matches` content | asserts |
|---|---|---|
| `no matches` | `nil` | `matches` is absent from the marshalled JSON (`omitempty`), and unmarshals back to a nil/empty slice |
| `empty matches` | `[]` | round-trips without becoming `nil`-vs-`[]` unequal — pick one representation and assert it in both directions |
| `literal match` | one entry, `value: "Open Sesame"`, `nextState: "open"` | field-by-field equality |
| `context match` | one entry, `valueFromContext: "{context.magatiaPassword}"`, `nextState: "open"` | field-by-field equality |
| `order preserved` | three entries with `value` `"a"`, `"b"`, `"c"` and `nextState` `"sa"`, `"sb"`, `"sc"` | after round-trip the slice is exactly `a,b,c` in that order — **not** a set comparison |

Fixture values for the state itself, used by every case:
`text: "The door reacts to the entry pass inserted. #bPassword#k!"`,
`defaultText: ""`, `minLength: 1`, `maxLength: 32`, `contextKey: "answer"`,
`nextState: "wrong-password"`.

Also add `TestStateModelJSONRoundTripAskText`: a `StateModel` whose
`stateType` is `AskTextType` and whose `askText` is the `literal match` fixture,
asserting the envelope carries `"askText"` and that every *other* state model
pointer is nil after unmarshal.

- [ ] **Step 2: Add the type constant and the `StateModel` plumbing**

In `model.go`:

- Add `AskTextType StateType = "askText"` to the const block at lines 36-49,
  positioned after `AskNumberType` (line 46) to match the existing grouping.
- Add `askText *AskTextModel` to the `StateModel` struct (lines 52-67), after
  the `askNumber` field at line 64.
- Add the accessor, mirroring `AskNumber()` at lines 124-127:

```go
func (s StateModel) AskText() *AskTextModel {
	return s.askText
}
```

- [ ] **Step 3: Add the two models**

Immutable, unexported fields, accessors only — no setters, no exported struct
literals, no test-only constructors. Place them next to `AskNumberModel`
(after line 640).

```go
type AskTextMatchModel struct {
	value            string
	valueFromContext string
	nextState        string
}

func (m AskTextMatchModel) Value() string            { return m.value }
func (m AskTextMatchModel) ValueFromContext() string { return m.valueFromContext }
func (m AskTextMatchModel) NextState() string        { return m.nextState }

type AskTextModel struct {
	text        string
	defaultText string
	minLength   uint16
	maxLength   uint16
	contextKey  string
	matches     []AskTextMatchModel
	nextState   string
}

func (a AskTextModel) Text() string               { return a.text }
func (a AskTextModel) DefaultText() string        { return a.defaultText }
func (a AskTextModel) MinLength() uint16          { return a.minLength }
func (a AskTextModel) MaxLength() uint16          { return a.maxLength }
func (a AskTextModel) ContextKey() string         { return a.contextKey }
func (a AskTextModel) Matches() []AskTextMatchModel { return a.matches }
func (a AskTextModel) NextState() string          { return a.nextState }
```

`minLength`/`maxLength` are `uint16` because the wire fields they feed —
`AskTextConversationDetail.Min`/`Max` — are `uint16`
(`libs/atlas-packet/npc/clientbound/conversation.go:174-175`, written with
`WriteShort`). Do not copy `askNumber`'s `uint32`.

- [ ] **Step 4: Add the JSONB codec**

In `model_json.go`, mirroring the `AskNumberModel` pair at lines 350-382. JSON
tags for the internal storage format (which is distinct from the REST format —
see Task 4): `text`, `defaultText`, `minLength`, `maxLength`, `contextKey`,
`matches` (with `omitempty`), `nextState`; and on the match model `value`,
`valueFromContext`, `nextState` (the first two with `omitempty`).

Then add `AskText *AskTextModel \`json:"askText,omitempty"\`` to the
`StateModel` marshal and unmarshal anonymous structs (lines 445-493 — the
`AskNumber` field appears at 456 and 473, the assembly at 459, the tagged read
at 489). Add the `askText` counterpart at each of those four sites.

- [ ] **Step 5: Run the tests**

```sh
cd services/atlas-npc-conversations/atlas.com/npc && go test ./conversation/...
```

---

## Task 3: `StateBuilder.SetAskText`, the `Build` arm, and the two model builders

### Files

- `services/atlas-npc-conversations/atlas.com/npc/conversation/builder.go` — `askText` field, `SetAskText`, the sibling-clear line in **all 12** existing setters, the `Build()` arm and struct literal, `AskTextBuilder`, `AskTextMatchBuilder`
- `services/atlas-npc-conversations/atlas.com/npc/conversation/builder_test.go` — new file

Patterns to copy: `conversation/builder.go:201-217` (`SetAskNumber`),
`:298-301` (the `Build()` `AskNumberType` arm), `:1106-1185`
(`AskNumberBuilder` — struct 1107-1114, `NewAskNumberBuilder` 1117-1121, setters
1124-1157, `Build()` 1160-1185).

Module root: `services/atlas-npc-conversations/atlas.com/npc`.

This task edits 3 files but touches 13 sites in `builder.go`. Twelve of them are
the same one-line mechanical change repeated, which is why it is not split (see
`context.md` §4).

- [ ] **Step 1: Write the failing test**

`TestStateBuilderSetAskTextClearsSiblings` — for each of the 11 other
`Set*` methods, build a `StateBuilder`, call that setter, then call
`SetAskText`, and assert the other setter's model pointer is nil on the built
`StateModel` while `AskText()` is non-nil.

The 11 siblings to iterate (from `builder.go:41-253`): `SetDialogue`,
`SetGenericAction`, `SetCraftAction`, `SetTransportAction`,
`SetGachaponAction`, `SetRPSAction`, `SetPartyQuestAction`,
`SetPartyQuestBonusAction`, `SetListSelection`, `SetAskNumber`, `SetAskStyle`,
`SetAskSlideMenu`.

`TestStateBuilderSetAskTextIsClearedBySiblings` — the reverse direction: call
`SetAskText`, then each sibling setter, and assert `AskText()` is nil. This is
the assertion that catches a missed clear site, which is the actual failure
mode here.

`TestAskTextBuilderBuild` — table-driven:

| case | inputs | expect |
|---|---|---|
| `minimal valid` | `text: "Password!"`, `maxLength: 32`, `contextKey: "answer"`, `nextState: "wrong"` | builds; `MinLength() == 0`; `DefaultText() == ""`; `Matches()` empty |
| `full` | above plus `defaultText: "hint"`, `minLength: 1`, two matches | builds; `Matches()` length 2 in declaration order |
| `missing text` | `maxLength: 32`, `contextKey: "a"`, `nextState: "n"` | error, message contains `text` |
| `zero maxLength` | `text: "t"`, `maxLength: 0`, `contextKey: "a"`, `nextState: "n"` | error, message contains `maxLength` |
| `minLength > maxLength` | `text: "t"`, `minLength: 10`, `maxLength: 5`, `contextKey: "a"`, `nextState: "n"` | error |
| `missing contextKey` | `text: "t"`, `maxLength: 32`, `nextState: "n"` | error, message contains `contextKey` |
| `missing nextState` | `text: "t"`, `maxLength: 32`, `contextKey: "a"` | error, message contains `nextState` |

`TestAskTextMatchBuilderBuild`:

| case | inputs | expect |
|---|---|---|
| `literal` | `value: "Open Sesame"`, `nextState: "open"` | builds |
| `from context` | `valueFromContext: "{context.pw}"`, `nextState: "open"` | builds |
| `neither` | `nextState: "open"` | error naming both fields |
| `both` | `value: "x"`, `valueFromContext: "{context.pw}"`, `nextState: "open"` | error naming both fields |
| `missing nextState` | `value: "x"` | error, message contains `nextState` |

`TestStateBuilderBuildRejectsAskTextWithNilModel` — set `stateType` to
`AskTextType` without calling `SetAskText` and assert `Build()` returns an
error.

- [ ] **Step 2: Add the builder field and setter**

Add `askText *AskTextModel` to the `StateBuilder` struct (lines 12-27), after
the `askNumber` field at line 24.

Add `SetAskText`, copying the exact shape of `SetAskNumber` at lines 201-217 —
set `b.stateType = AskTextType`, nil **all 12** other sibling fields (the 11
listed in Step 1 plus `askNumber`), then assign `b.askText`.

- [ ] **Step 3: Add `b.askText = nil` to every other setter**

Twelve edits, one line each, in `SetDialogue` (41-56), `SetGenericAction`,
`SetCraftAction`, `SetTransportAction`, `SetGachaponAction`, `SetRPSAction`,
`SetPartyQuestAction`, `SetPartyQuestBonusAction`, `SetListSelection`
(185-199), `SetAskNumber` (201-217), `SetAskStyle` (220-235), and
`SetAskSlideMenu` (238-253). Place the new line alongside the existing
`b.askNumber = nil` in each so the block stays in a consistent order.

`TestStateBuilderSetAskTextIsClearedBySiblings` fails on any one of these being
missed.

- [ ] **Step 4: Add the `Build()` arm and struct field**

In the `switch b.stateType` at lines 256-330, add after the `AskNumberType`
arm (298-301):

```go
	case AskTextType:
		if b.askText == nil {
			return StateModel{}, errors.New("askText is required for askText state")
		}
```

Add `askText: b.askText,` to the final struct literal at lines 314-329,
alongside `askNumber:` at line 326.

- [ ] **Step 5: Add `AskTextBuilder` and `AskTextMatchBuilder`**

Place after `AskNumberBuilder` (line 1185). Follow its shape: a struct of the
same fields as the model, a `New…Builder()` constructor, one `Set…` per field
returning the builder, and a `Build() (*AskTextModel, error)` performing the
validation in the Step 1 table.

`NewAskTextBuilder()` defaults `contextKey` to `"answer"` — the analogue of
`NewAskNumberBuilder`'s `"quantity"` default at `builder.go:1117-1121`. The
`missing contextKey` case above therefore only fails when the caller
explicitly sets it to `""`; assert that, not an unset-field error.

`AskTextBuilder` needs an `AddMatch(AskTextMatchModel) *AskTextBuilder` that
appends, so match order is the caller's declaration order.

- [ ] **Step 6: Run the tests**

```sh
cd services/atlas-npc-conversations/atlas.com/npc && go test ./conversation/...
```

---

## Task 4: REST transform and extract, in both REST layers

`conversation/quest/rest.go` is a full parallel copy of the NPC REST layer for
quest-authored state machines. Both must gain the same six sites or an
`askText` state authored on a quest conversation silently loses its payload.

### Files

- `services/atlas-npc-conversations/atlas.com/npc/conversation/rest.go` — `RestAskTextModel`, `RestAskTextMatchModel`, `RestStateModel.AskText`, transform arm, `TransformAskText`, extract arm, `ExtractAskText`
- `services/atlas-npc-conversations/atlas.com/npc/conversation/quest/rest.go` — the same six sites
- `services/atlas-npc-conversations/atlas.com/npc/conversation/rest_transform_test.go` — add cases

Patterns to copy: `conversation/rest.go:176-184` (`RestAskNumberModel`), `:24`
(the `RestStateModel` field), `:338-343` (transform arm), `:495-505`
(`TransformAskNumber`), `:651-660` (extract arm), `:897-911`
(`ExtractAskNumber`). The quest mirror's corresponding sites:
`conversation/quest/rest.go:90` (field), `:159` (`RestAskNumberModel`),
`:264-269` (transform arm), `:377` (`TransformAskNumber`), `:489-497` (extract
arm), `:651` (`ExtractAskNumber`).

Module root: `services/atlas-npc-conversations/atlas.com/npc`.

- [ ] **Step 1: Write the failing test**

Add to `rest_transform_test.go`, copying the setup shape of the existing
`t.Run("AskNumber", …)` at lines 373-395 (`NewAskNumberBuilder()…Build()` →
`TransformAskNumber` → `ExtractAskNumber` → `reflect.DeepEqual`).

`t.Run("AskText", …)` — build via `NewAskTextBuilder()` with
`text: "The door reacts to the entry pass inserted. #bPassword#k!"`,
`defaultText: ""`, `minLength: 1`, `maxLength: 32`, `contextKey: "answer"`,
`nextState: "wrong-password"`, and two matches:
`{value: "Open Sesame", nextState: "open"}` then
`{valueFromContext: "{context.magatiaPassword}", nextState: "open-magatia"}`.
Transform, extract, and assert `reflect.DeepEqual` against the original —
including that the two matches survive in that order.

`t.Run("AskTextNoMatches", …)` — the same fixture with no matches, asserting
the round-trip does not turn a nil slice into a non-nil empty one or vice
versa.

`t.Run("AskTextQuestLayer", …)` — the identical assertions against the
`conversation/quest` package's `TransformAskText` / `ExtractAskText`. If that
package has its own `_test.go`, put this case there instead; either way it must
exist, because the two layers are separate code with no compiler link between
them.

- [ ] **Step 2: Add the REST models**

In `rest.go`, after `RestAskNumberModel` (line 184):

```go
type RestAskTextMatchModel struct {
	Value            string `json:"value,omitempty"`
	ValueFromContext string `json:"valueFromContext,omitempty"`
	NextState        string `json:"nextState"`
}

type RestAskTextModel struct {
	Text        string                  `json:"text"`
	DefaultText string                  `json:"defaultText"`
	MinLength   uint16                  `json:"minLength"`
	MaxLength   uint16                  `json:"maxLength"`
	ContextKey  string                  `json:"contextKey,omitempty"`
	Matches     []RestAskTextMatchModel `json:"matches,omitempty"`
	NextState   string                  `json:"nextState,omitempty"`
}
```

These tags match the PRD §5 example payload. They deliberately differ from
`RestAskNumberModel`'s `default`/`min`/`max` (`rest.go:176-184`) because these
bound a *length*, not a *value* — see `context.md` §2.

Add `AskText *RestAskTextModel \`json:"askText,omitempty"\`` to
`RestStateModel` (line 24, beside `AskNumber`).

- [ ] **Step 3: Add transform and extract**

`TransformAskText(m AskTextModel) RestAskTextModel` and
`ExtractAskText(r RestAskTextModel) (*AskTextModel, error)`, mirroring lines
495-505 and 897-911. `ExtractAskText` builds through `NewAskTextBuilder` and
`NewAskTextMatchBuilder` so the Task 3 validation runs on the REST path too.

Add the switch arms: transform at the site matching lines 338-343, extract at
the site matching lines 651-660 (which returns an error when
`r.AskText == nil` for an `askText`-typed state, then calls
`stateBuilder.SetAskText(askText)`).

- [ ] **Step 4: Mirror all six sites into `conversation/quest/rest.go`**

At the line references given above. This file is a copy, not an import — the
same declarations are re-declared in the `quest` package.

- [ ] **Step 5: Run the tests**

```sh
cd services/atlas-npc-conversations/atlas.com/npc && go test ./conversation/...
```

---

## Task 5: Validator rules for `askText`

### Files

- `services/atlas-npc-conversations/atlas.com/npc/conversation/validator.go` — dispatch arm, `validateAskText`, circular-reference visitor arm, next-state collector arm
- `services/atlas-npc-conversations/atlas.com/npc/conversation/validator_asktext_test.go` — new file

Patterns to copy: `conversation/validator.go:109-110` (dispatch arm),
`:396-421` (`validateAskNumber`), `:547-549` (circular-reference visitor),
`:699-701` (next-state collector). For the test file's setup shape,
`services/atlas-npc-conversations/atlas.com/npc/conversation/validator_rps_test.go`
is the existing per-state-type validator test in this package.

The function this task adds:

```go
func (v *ValidatorImpl) validateAskText(stateId string, askText *AskTextModel, stateIds map[string]bool, result *ValidationResult)
```

Match the receiver type, the `stateIds` container type, and the result type to
whatever `validateAskNumber` (`:396-421`) actually uses — read the signature
there rather than trusting this sketch.

Module root: `services/atlas-npc-conversations/atlas.com/npc`.

- [ ] **Step 1: Write the failing test**

`TestValidateAskText` — table-driven, one case per PRD §4.1 rule. Every case
validates a two-state conversation whose other state has id `"wrong-password"`,
so `nextState: "wrong-password"` resolves and any other target does not.

| case | `askText` under test | expect |
|---|---|---|
| `valid` | `text: "Password!"`, `minLength: 1`, `maxLength: 32`, `contextKey: "answer"`, `nextState: "wrong-password"`, no matches | no error |
| `nil model` | state typed `askText` with a nil model | error |
| `empty text` | `text: ""`, rest valid | error, field `text`, code `required` |
| `empty contextKey` | `contextKey: ""`, rest valid | error, field `contextKey`, code `required` |
| `empty nextState` | `nextState: ""`, rest valid | error, field `nextState`, code `required` |
| `unknown nextState` | `nextState: "nope"`, rest valid | error, code `invalid_reference` |
| `zero maxLength` | `maxLength: 0`, rest valid | error, field `maxLength` |
| `min exceeds max` | `minLength: 33`, `maxLength: 32` | error, field `minLength` |
| `min equals max` | `minLength: 32`, `maxLength: 32` | no error — the bound is inclusive |
| `match with neither` | one match, `value: ""`, `valueFromContext: ""`, `nextState: "wrong-password"` | error naming both fields |
| `match with both` | one match, `value: "x"`, `valueFromContext: "{context.pw}"` | error naming both fields |
| `match unknown nextState` | one match, `value: "x"`, `nextState: "nope"` | error, code `invalid_reference` — this is the rule that turns a typo into a 400 instead of a runtime dead end |
| `match valueFromContext malformed` | one match, `valueFromContext: "magatiaPassword"` (no `{context.…}` wrapper) | error — must be syntactically a context reference |
| `match valueFromContext valid` | one match, `valueFromContext: "{context.magatiaPassword}"`, `nextState: "wrong-password"` | no error |
| `defaultText shorter than minLength` | `defaultText: ""`, `minLength: 5`, `maxLength: 32` | **no error** — the client pre-fills the default and the player may clear it; constraining it would block the common `defaultText: ""` case |

`TestValidateAskTextCircularReference` — three states where the `askText`'s
**match** `nextState` closes a cycle back to itself. This asserts the visitor
arm walks `matches[].nextState` and not only the fallback `nextState`; a
visitor that walks only the fallback reports no cycle and the test fails.

- [ ] **Step 2: Add the dispatch arm and `validateAskText`**

In the `validateState` switch (lines 90-115), after the `AskNumberType` arm at
109-110:

```go
	case AskTextType:
		v.validateAskText(state.Id(), state.AskText(), stateIds, result)
```

`validateAskText` mirrors `validateAskNumber` (396-421) — same result-appending
helpers, same error codes (`required`, `invalid`, `invalid_reference`) — and
adds a loop over `Matches()` that, per entry, checks the exactly-one-of rule,
the `{context.…}` syntax of `valueFromContext`, and that `NextState()` is in
`stateIds`.

Reuse whatever the package already uses to recognise a context reference —
`ExtractContextValue` and `ReplaceContextPlaceholders` live in
`conversation/context_replacer.go`. Do not introduce a second regexp; grep for
the existing one and call it.

- [ ] **Step 3: Add the visitor and collector arms**

Circular-reference visitor, at the site matching lines 547-549 — visit the
fallback **and** every match target:

```go
	case AskTextType:
		if askText := state.AskText(); askText != nil {
			for _, m := range askText.Matches() {
				visit(m.NextState())
			}
			visit(askText.NextState())
		}
```

Next-state collector, at the site matching lines 699-701 — append the same set.

- [ ] **Step 4: Run the tests**

```sh
cd services/atlas-npc-conversations/atlas.com/npc && go test ./conversation/...
```

---

## Task 6: Outbound — `CommandTextBody`, `SendText`, and `processAskTextState`

The engine → channel leg. `CommandTypeText = "TEXT"` already exists
(`kafka/message/npc/kafka.go:25`) and is unused; this task gives it a body, a
provider, and a caller.

### Files

- `services/atlas-npc-conversations/atlas.com/npc/kafka/message/npc/kafka.go` — add `CommandTextBody`
- `services/atlas-npc-conversations/atlas.com/npc/npc/producer.go` — add `textConversationProvider`
- `services/atlas-npc-conversations/atlas.com/npc/npc/processor.go` — add `SendText` to the processor and its interface
- `services/atlas-npc-conversations/atlas.com/npc/conversation/processor.go` — add `processAskTextState` and the `ProcessState` dispatch arm
- `services/atlas-npc-conversations/atlas.com/npc/conversation/processor_asktext_test.go` — new file

Patterns to copy: `npc/producer.go:44-62` (`numberConversationProvider`),
`npc/processor.go:141-142` (`SendNumber`),
`conversation/processor.go:1215-1238` (`processAskNumberState`), `:607-609`
(the `ProcessState` `AskNumberType` dispatch arm),
`kafka/message/npc/kafka.go:91-95` (`CommandNumberBody`).

Module root: `services/atlas-npc-conversations/atlas.com/npc`.

Do **not** edit `kafka/message/npc/kafka.go`'s `CommandConversationContinueBody`
in this task — Task 1 owns it, and the mirror guard fires on any change to this
file. Adding `CommandTextBody` does change the file, so re-run the guard after
this task and mirror the addition into atlas-saga-orchestrator if the guard
demands it. (It will: the guard compares the whole file from `package` onward.)

- [ ] **Step 1: Write the failing test**

`TestProcessAskTextState` in `conversation/processor_asktext_test.go` — asserts
the command the state emits. Build the conversation through the Task 3
builders; capture the emitted command by whatever seam the package's existing
processor tests use for `processAskNumberState`. If none exists, capture at the
`npcSender` interface by substituting a recording implementation — the
processor already resolves it through `npcSender.NewProcessor(p.l, p.ctx)`, so
this task may need to thread that dependency; prefer the smallest change that
makes the emission observable and do not add a test-only constructor.

| case | state | conversation context | expect on the emitted command |
|---|---|---|---|
| `plain prompt` | `text: "Password!"`, `defaultText: ""`, `minLength: 1`, `maxLength: 32` | empty | `Type == "TEXT"`; `Message == "Password!"`; `DefaultValue == ""`; `MinLength == 1`; `MaxLength == 32` |
| `context placeholder resolved` | `text: "Enter #b{context.hint}#k!"` | `{"hint": "the password"}` | `Message == "Enter #bthe password#k!"` |
| `default value carried` | `defaultText: "prefill"` | empty | `DefaultValue == "prefill"` |
| `zero minLength` | `minLength: 0`, `maxLength: 8` | empty | `MinLength == 0`; `MaxLength == 8` |

Plus: `processAskTextState` returns `state.Id()` — the conversation parks on
this state awaiting input, exactly as `processAskNumberState` does at
`processor.go:1237`.

- [ ] **Step 2: Add `CommandTextBody`**

Beside `CommandNumberBody` (lines 91-95):

```go
type CommandTextBody struct {
	DefaultValue string `json:"defaultValue"`
	MinLength    uint16 `json:"minLength"`
	MaxLength    uint16 `json:"maxLength"`
}
```

The prompt itself rides on `ConversationCommand.Message`, as it does for
`NUMBER` — the body carries only what is specific to the text dialog.

- [ ] **Step 3: Add the provider and `SendText`**

`textConversationProvider` in `npc/producer.go`, copying
`numberConversationProvider` (44-62) and emitting
`npc2.ConversationCommand[npc2.CommandTextBody]` with
`Type: npc2.CommandTypeText`.

`SendText` in `npc/processor.go`, beside `SendNumber` (141-142):

```go
func (p *ProcessorImpl) SendText(ch channel.Model, characterId uint32, npcId uint32, message string, def string, min uint16, max uint16) error
```

Add it to the `Processor` interface in the same file, and to any mock of that
interface (grep for other implementors before building).

- [ ] **Step 4: Add `processAskTextState` and the dispatch arm**

The function this task adds:

```go
func (p *ProcessorImpl) processAskTextState(ctx ConversationContext, state StateModel) (string, error)
```

Match the receiver, parameter, and return types to `processAskNumberState`
(`processor.go:1215-1238`) — read that signature rather than trusting this
sketch.

Copy `processAskNumberState` (1215-1238) verbatim in shape: replace context
placeholders in `askText.Text()`, call `SendText` with
`askText.DefaultText()`, `askText.MinLength()`, `askText.MaxLength()`, return
`state.Id(), nil`.

Add to the `ProcessState` switch (which begins at line 531), beside the
`AskNumberType` arm at 607-609:

```go
	case AskTextType:
		return p.processAskTextState(ctx, state)
```

- [ ] **Step 5: Run the tests and the mirror guard**

```sh
cd services/atlas-npc-conversations/atlas.com/npc && go test ./...
./tools/npc-conversation-contract-mirror-guard.sh
```

---

## Task 7: Inbound — thread `text` through `Continue` and add the `AskTextType` arm

The channel → engine leg. This is the branch-evaluation core.

### Files

- `services/atlas-npc-conversations/atlas.com/npc/conversation/processor.go` — `Continue` signature (line 320) and the new `AskTextType` arm
- `services/atlas-npc-conversations/atlas.com/npc/kafka/consumer/npc/consumer.go` — pass `c.Body.Text` at line 102
- `services/atlas-npc-conversations/atlas.com/npc/conversation/mock/processor.go` — `ContinueFunc` field (line 23) and `Continue` method (57-63)
- `services/atlas-npc-conversations/atlas.com/npc/conversation/processor_asktext_test.go` — new file in Task 6; add cases here

Patterns to copy: `conversation/processor.go:372-408` (the `AskNumberType` arm
inside `Continue`, including the range check at 386-393 and the context store
at 397).

Module root: `services/atlas-npc-conversations/atlas.com/npc`.

- [ ] **Step 1: Write the failing test**

`TestContinueAskText` — table-driven. Each case drives `Continue` on a
conversation parked on an `askText` state and asserts the resulting next state
and context.

Fixture state, shared by every case unless the row overrides it:
`minLength: 3`, `maxLength: 20`, `contextKey: "answer"`,
`nextState: "wrong-password"`, matches in this order:

1. `{value: "Open Sesame", nextState: "s-literal"}`
2. `{valueFromContext: "{context.pw}", nextState: "s-context"}`
3. `{value: "Open Sesame", nextState: "s-duplicate"}`

| case | `text` argument | pre-existing context | expect next state | expect `context["answer"]` |
|---|---|---|---|---|
| `first match wins` | `"Open Sesame"` | `{}` | `s-literal` | `"Open Sesame"` |
| `duplicate never reached` | `"Open Sesame"` | `{}` | `s-literal` — never `s-duplicate` | `"Open Sesame"` |
| `context match` | `"hunter2xy"` | `{"pw": "hunter2xy"}` | `s-context` | `"hunter2xy"` |
| `context match unresolved` | `"hunter2xy"` | `{}` (no `pw` key) | `wrong-password` — an unresolvable `valueFromContext` does not match, it does not error | `"hunter2xy"` |
| `no match falls back` | `"wrong answer"` | `{}` | `wrong-password` | `"wrong answer"` |
| `empty matches` | `"anything ok"` | `{}`, state overridden to no matches | `wrong-password` | `"anything ok"` |
| `trimmed before match` | `"  Open Sesame  "` | `{}` | `s-literal` | `"Open Sesame"` — **trimmed**, so `{context.answer}` agrees with the branch taken |
| `trimmed before length check` | `"  ab  "` (7 bytes raw, 2 trimmed) | `{}` | error; conversation stays parked | unchanged |
| `case sensitive` | `"open sesame"` | `{}` | `wrong-password` | `"open sesame"` |
| `below minLength` | `"ab"` | `{}` | error; conversation stays parked | unchanged |
| `at minLength` | `"abc"` | `{}` | `wrong-password` | `"abc"` |
| `at maxLength` | 20-byte string `"abcdefghijklmnopqrst"` | `{}` | `wrong-password` | that string |
| `above maxLength` | 21-byte string `"abcdefghijklmnopqrstu"` | `{}` | error; conversation stays parked | unchanged |

The two length-rejection cases must also assert an error-level log carrying the
character id and the state id (PRD §8 observability).

`TestContinueAskTextDownstreamContextRead` — a two-state conversation: an
`askText` with `contextKey: "answer"` and `nextState: "echo"`, where `echo` is
a `dialogue` state with `text: "You said {context.answer}."`. Drive `Continue`
with `"Open Sesame"` and assert the rendered dialogue text is
`"You said Open Sesame."` This is the PRD acceptance criterion "the reply is
readable from a later state".

- [ ] **Step 2: Change the `Continue` signature and its three touch points**

`conversation/processor.go:320` becomes:

```go
func (p *ProcessorImpl) Continue(npcId uint32, characterId uint32, action byte, lastMessageType byte, selection int32, text string) error
```

Update the `Processor` interface declaration in the same file, the mock
(`conversation/mock/processor.go:23` field type and `:57-63` method), and the
single production caller at
`kafka/consumer/npc/consumer.go:102`, which becomes:

```go
_ = conversation.NewProcessor(l, ctx, db).Continue(c.NpcId, c.CharacterId, c.Body.Action, c.Body.LastMessageType, c.Body.Selection, c.Body.Text)
```

`c.Body.Text` exists because of Task 1. A positional parameter is deliberate —
see `design.md` §5.

- [ ] **Step 3: Add the `AskTextType` arm**

In `Continue`'s switch (which begins around line 346), beside the
`AskNumberType` arm at 372-408. Order of operations, exactly:

1. `trimmed := strings.TrimSpace(text)` — **once**, before anything else.
2. Length check on `len(trimmed)` (bytes, as decoded by `ReadAsciiString`):
   below `MinLength()` or above `MaxLength()` → log at error with character id
   and state id, return an error, leave `nextStateId` unset so the conversation
   stays parked. Mirror the shape of the `askNumber` bounds check at 386-393.
3. Store `trimmed` into `choiceContext[askText.ContextKey()]`, as line 397 does
   for `askNumber`.
4. Walk `askText.Matches()` in order. For a `value` entry compare
   `trimmed == m.Value()`. For a `valueFromContext` entry, resolve against the
   context first with the package's existing extractor and compare the
   resolved value; an unresolvable reference is a non-match, not an error.
   First hit sets `nextStateId = m.NextState()` and stops the walk.
5. If nothing matched, `nextStateId = askText.NextState()`.

Comparison is exact and case-sensitive. No regex, no wildcards, no
normalisation beyond the single trim.

- [ ] **Step 4: Run the tests**

```sh
cd services/atlas-npc-conversations/atlas.com/npc && go test ./...
```

---

## Task 8: atlas-channel — stop discarding the decoded reply

`npc_continue_conversation.go:76` reads the player's text into `sp` and then
throws it away; `selection` stays `-1` and the engine receives nothing.

### Files

- `services/atlas-channel/atlas.com/channel/socket/handler/npc_continue_conversation.go` — the `bodyText` arm (lines 70-79)
- `services/atlas-channel/atlas.com/channel/npc/processor.go` — `ContinueConversation` signature (locate by grep; it is the function the handler calls at line 77)
- `services/atlas-channel/atlas.com/channel/npc/producer.go` — the provider building `ContinueConversationCommandBody` (same grep)
- `services/atlas-channel/atlas.com/channel/socket/handler/npc_continue_conversation_test.go` — add a case

Patterns to copy: the existing test's setup block at
`npc_continue_conversation_test.go:13-28` (`gms83MessageType`, a hand-built
tenant messageType table) and its table-driven body at `:38-62`.

Module root: `services/atlas-channel/atlas.com/channel`.

- [ ] **Step 1: Write the failing test**

`TestContinueConversationCarriesText` — drive
`NPCContinueConversationHandleFunc` with a reader holding an `ASK_TEXT`
continuation and assert the produced `ContinueConversationCommandBody`.

Reuse `gms83MessageType` (`:13-28`) for the tenant options, under which
`ASK_TEXT` is byte `2`.

| case | `action` | `lastMessageType` | trailing body | expect `Text` | expect `Selection` |
|---|---|---|---|---|---|
| `text reply carried` | `1` | `2` (`ASK_TEXT`) | ascii string `"Open Sesame"` | `"Open Sesame"` | `-1` |
| `empty reply carried` | `1` | `2` | ascii string `""` | `""` | `-1` |
| `box text reply carried` | `1` | `13` (`ASK_BOX_TEXT`) | ascii string `"multi line"` | `"multi line"` | `-1` |
| `cancel disposes` | `0` | `2` | none | — | the existing dispose path runs unchanged; no continue command is produced |
| `selection path unaffected` | `1` | `3` (`ASK_NUMBER`) | int32 `7` | `""` | `7` |

Capturing the produced command needs a seam at the producer. Use whatever the
package already provides for asserting emitted Kafka commands; if there is
none, assert at the `ContinueConversation` call instead by substituting a
recording npc processor. `TestContinueConversationBodyKind` (`:38-62`) already
covers the `bodyKind` classification including `{"ASK_TEXT", 2, bodyText}` at
line 50 — do not duplicate it.

- [ ] **Step 2: Thread the text through**

In `npc_continue_conversation.go`:

- Delete the commented-out `returnText := ""` declaration at line 66 rather than
  reviving it — the value is now read where it is used.
- Delete the placeholder comment at line 76 (the one reading "set return text")
  and replace the `bodyText` arm's call with one passing `sp.Text()`. No comment
  replaces it; the code now does the thing the comment described.
- Leave the unrelated quest-continuation marker comment at line 74 alone; it
  belongs to a different concern and this task does not touch it.
- The `bodySelection` (83-97) and `bodyNone` (89-97) arms pass `""`.
- The `action == 0` cancel path is untouched and still disposes.

Add the `text string` parameter to `ContinueConversation` on the channel's npc
processor and carry it into `ContinueConversationCommandBody.Text` in the
producer.

- [ ] **Step 3: Run the tests and the cancel-path guard**

```sh
cd services/atlas-channel/atlas.com/channel && go test ./...
./tools/operator-cancel-path-guard.sh
```

The guard is gated on this exact directory (`tools/verify.sh:457-462`).

---

## Task 9: atlas-channel — the `TEXT` consumer arm and `announceTextConversation`

### Files

- `services/atlas-channel/atlas.com/channel/kafka/message/npc/conversation/kafka.go` — add `CommandTextBody`
- `services/atlas-channel/atlas.com/channel/kafka/consumer/npc/conversation/consumer.go` — register a fifth handler, add `handleTextConversationCommand` and `announceTextConversation`, add the `"TEXT"` case to `getNPCTalkType`
- `services/atlas-channel/atlas.com/channel/kafka/consumer/npc/conversation/consumer_test.go` — **new file**

Patterns to copy: `consumer.go:42-61` (`InitHandlers`, which registers the four
existing handlers), `:85-100` (`handleNumberConversationCommand`), `:156-167`
(`announceNumberConversation`), `:182-193` (`announceSlideMenuConversation` —
the precedent that passes the message type directly instead of routing through
`getNPCTalkType`), `:213-235` (`getNPCTalkType`).

Module root: `services/atlas-channel/atlas.com/channel`.

There is no existing test for this consumer — the only `_test.go` in the
consumer tree is `kafka/consumer/npc/shop/consumer_test.go`, an unrelated
domain. The new file is genuinely new.

- [ ] **Step 1: Write the failing test**

`TestAnnounceTextConversation` in the new `consumer_test.go` — assert the
announced packet model, not the bytes (the byte encoding is Task 13's job).

| case | command body | `Message` | expect on `AskTextConversationDetail` | expect on `NpcConversation` |
|---|---|---|---|---|
| `full` | `{DefaultValue: "prefill", MinLength: 1, MaxLength: 32}` | `"Password!"` | `Message == "Password!"`, `Def == "prefill"`, `Min == uint16(1)`, `Max == uint16(32)` | msg type is `NpcConversationMessageTypeAskText` |
| `empty default` | `{DefaultValue: "", MinLength: 0, MaxLength: 8}` | `"Enter:"` | `Def == ""`, `Min == 0`, `Max == 8` | as above |
| `secondary npc` | as `full`, with a non-zero `SecondaryNpcId` | as `full` | the secondary id is carried through | as above |

`TestHandleTextConversationCommandIgnoresOtherTypes` — a command whose `Type`
is `"NUMBER"` must be ignored by `handleTextConversationCommand`, matching the
guard at `consumer.go:86`.

`TestGetNPCTalkTypeText` — `getNPCTalkType("TEXT")` returns
`npcpkt.NpcConversationMessageTypeAskText` and does not panic. `getNPCTalkType`
panics on anything unmapped (`consumer.go:234`), so this test is the one that
proves the new case landed.

- [ ] **Step 2: Add `CommandTextBody`**

In `kafka/message/npc/conversation/kafka.go`, beside `CommandNumberBody`
(lines 34-38). Same three fields and JSON tags as the engine-side body in
Task 6 — `defaultValue`, `minLength`, `maxLength` — because they are two ends
of one wire contract.

- [ ] **Step 3: Add the handler and the announce function**

`handleTextConversationCommand`, copying `handleNumberConversationCommand`
(85-100) and guarding on `c.Type != conversation2.CommandTypeText` — the
constant already exists at `kafka/message/npc/conversation/kafka.go:11`.

`announceTextConversation`, copying `announceNumberConversation` (156-167) but
following `announceSlideMenuConversation`'s precedent (182-193): pass
`npcpkt.NpcConversationMessageTypeAskText` **directly** rather than calling
`getNPCTalkType(talkType)`. `getNPCTalkType` panics on an unrecognised string
(line 234), and nothing on this path should be able to panic on a config or
string mismatch. Build
`&npcpkt.AskTextConversationDetail{Message, Def, Min, Max}` — note `Def` is a
`string` and `Min`/`Max` are `uint16`
(`libs/atlas-packet/npc/clientbound/conversation.go:171-176`).

Register the handler in `InitHandlers` (42-61) as the fifth entry.

- [ ] **Step 4: Add the `"TEXT"` case to `getNPCTalkType`**

One line in the switch at 213-235, returning
`npcpkt.NpcConversationMessageTypeAskText`. PRD §4.2 requires it and it shrinks
the panic surface for any future caller, even though the announce path in
Step 3 does not depend on it.

- [ ] **Step 5: Run the tests**

```sh
cd services/atlas-channel/atlas.com/channel && go test ./...
```

---

## Task 10: A quest-progress client for atlas-npc-conversations

`design.md` §8 says the service "has no quest client". It does: `conversation/quest/status/`
already calls `RootUrlFor(ctx, "QUEST")` and is consumed at
`conversation/quest/processor.go:147`. This task adds a `progress` sibling in
the same convention, so no new env key and no deployment change is needed.

### Files

- `services/atlas-npc-conversations/atlas.com/npc/conversation/quest/progress/requests.go` — **new file**
- `services/atlas-npc-conversations/atlas.com/npc/conversation/quest/progress/rest.go` — **new file**
- `services/atlas-npc-conversations/atlas.com/npc/conversation/quest/progress/rest_test.go` — **new file**

Patterns to copy: `services/atlas-npc-conversations/atlas.com/npc/conversation/quest/status/requests.go`
(the whole file — same package layout, same `RootUrlFor(ctx, "QUEST")` base,
same `characters/%d/quests/%d` path prefix) and
`.../quest/status/rest.go` (the `RestModel` + `GetName`/`GetID`/`SetID` shape).

For the paginated read: `services/atlas-npc-shops/atlas.com/npc/data/consumable/processor.go:44`
— `requests.DrainProvider[RestModel, Model](p.l, p.ctx)(url, 250, Extract, model.Filters[Model]())()`
is the concrete client-side pattern for draining a paginated JSON:API
collection. `DrainProvider` is declared at `libs/atlas-rest/requests/paged.go:116`.
Note it takes a **bare URL string**, not a `requests.Request` — see the comment
at `services/atlas-query-aggregator/atlas.com/query-aggregator/quest/requests.go:31-34`
explaining why paginated endpoints expose a URL builder instead.

Module root: `services/atlas-npc-conversations/atlas.com/npc`.

Server side, verified: `services/atlas-quest/atlas.com/quest/quest/resource.go:51`
registers `GET /characters/{characterId}/quests/{questId}/progress`, handled at
`:258-306`, which returns **404** when the character has no record for that
quest (`:271-274`) and otherwise a paginated collection. The REST model is
`services/atlas-quest/atlas.com/quest/quest/progress/rest.go:8-12`, verbatim:

```go
type RestModel struct {
	Id         uint32 `json:"-"`
	InfoNumber uint32 `json:"infoNumber"`
	Progress   string `json:"progress"`
}
```

- [ ] **Step 1: Write the failing test**

`TestProgressRestRoundTrip` in `rest_test.go` — mirrors
`.../quest/status/`'s own round-trip test shape.

| case | RestModel | asserts |
|---|---|---|
| `numeric progress` | `{Id: 1, InfoNumber: 0, Progress: "72"}` | `Progress` stays the **string** `"72"`; it is never parsed to an int |
| `text progress` | `{Id: 2, InfoNumber: 0, Progress: "Open Sesame"}` | round-trips unchanged |
| `named info number` | `{Id: 3, InfoNumber: 9300285, Progress: "0"}` | `InfoNumber` round-trips |
| `empty progress` | `{Id: 4, InfoNumber: 0, Progress: ""}` | round-trips as empty, not absent |

Assert `GetName()` returns the JSON:API type name the server emits — read it
from `services/atlas-quest/atlas.com/quest/quest/progress/rest.go`'s own
`GetName()` and use that exact string; a mismatch makes every decode return an
empty collection with no error.

- [ ] **Step 2: Add `rest.go`**

`RestModel` with the three fields above and identical JSON tags, plus
`GetName`/`GetID`/`SetID`, a `Model` with unexported fields and
`InfoNumber()`/`Progress()` accessors, and `Extract(RestModel) (Model, error)`.

Progress is stored and returned as a `string`, unparsed — parsing it when it
happens to look numeric would make `valueFromContext` comparison depend on the
content of the data, which is exactly the coercion problem that ruled out the
query-aggregator path (`design.md` §8).

- [ ] **Step 3: Add `requests.go`**

```go
func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "QUEST")
}
```

and a `ByCharacterAndQuestUrl(ctx, characterId, questId uint32) (string, error)`
returning `root + "characters/%d/quests/%d/progress"` — a bare URL, for
`DrainProvider`.

- [ ] **Step 4: Add the processor**

A `Processor` interface with one method returning every progress entry for a
character's quest, backed by `requests.DrainProvider`. It must distinguish a
**404** (no quest record) from a transport error, because Task 11 treats the
two differently. Follow whatever error the `status` sibling surfaces for 404
and reuse it rather than minting a second sentinel; if `status` has none, add
an `ErrNotFound` in this package following
`services/atlas-npc-conversations/atlas.com/npc/saved_location`'s
`savedlocation.ErrNotFound` (referenced at
`conversation/operation_executor.go:759`).

- [ ] **Step 5: Build and test**

```sh
cd services/atlas-npc-conversations/atlas.com/npc && go build ./... && go test ./conversation/...
```

---

## Task 11: The `local:get_quest_progress` operation

**The operation type is `local:get_quest_progress`, not `get_quest_progress`.**
`operation_executor.go:317-320` routes by prefix:

```go
func isLocalOperationType(operationType string) bool {
	return strings.HasPrefix(operationType, "local:")
}
```

An un-prefixed operation is packaged into a saga and dispatched to
atlas-saga-orchestrator (`:296-315`), which has no way to write conversation
context. Every sibling context-loading read is namespaced the same way:
`local:get_saved_location` (`:727`), `local:fetch_map_player_counts` (`:634`),
`local:enumerate_evolvable_pets` (`:803`). This corrects `prd.md` §4.5 and
`design.md` §8; the behaviour is unchanged.

### Files

- `services/atlas-npc-conversations/atlas.com/npc/conversation/operation_executor.go` — the `questProgressP` field and wiring, and the new `case "get_quest_progress":` arm
- `services/atlas-npc-conversations/atlas.com/npc/conversation/operation_executor_test.go` — add cases
- `docs/npc_conversation_conversion_spec.md` — read-only here; Task 21 documents the operation

Patterns to copy: `conversation/operation_executor.go:727-796` — the
`get_saved_location` arm, which is the exact shape this needs: required-param
extraction with `errors.New` (733-736), `evaluateContextValue` for placeholder
substitution (738-741), optional params with defaults (743-753), the processor
call (756), an `errors.Is(err, savedlocation.ErrNotFound)` branch with a
fallback that returns `nil` (757-784) versus a hard failure (785), and
`e.setContextValue` on success (789-794). Also `:39-50` (the impl struct's
processor fields, `savedLocationP` at 49) and `:53-68` (`NewOperationExecutor`,
wiring `savedLocationP` at 66).

Module root: `services/atlas-npc-conversations/atlas.com/npc`.

- [ ] **Step 1: Write the failing test**

`TestGetQuestProgressOperation` — table-driven, following the setup shape
already in `operation_executor_test.go`.

Params contract:

| Param | Required | Meaning |
|---|---|---|
| `questId` | yes | Quest template id |
| `infoNumber` | no, default `"0"` | Which progress entry. `step` is accepted as an alias for author familiarity with the `questProgress` condition; `infoNumber` is canonical and wins if both are present. |
| `contextKey` | yes | Context key to store the result under |

| case | params | quest service returns | expect `context[contextKey]` | expect error |
|---|---|---|---|---|
| `default info number` | `questId: "3360"`, `contextKey: "magatiaPassword"` | `[{infoNumber: 0, progress: "Open Sesame"}]` | `"Open Sesame"` | nil |
| `named info number` | `questId: "20730"`, `infoNumber: "9300285"`, `contextKey: "gate"` | `[{infoNumber: 0, progress: "x"}, {infoNumber: 9300285, progress: "0"}]` | `"0"` | nil |
| `step alias` | `questId: "20730"`, `step: "9300285"`, `contextKey: "gate"` | as above | `"0"` | nil |
| `infoNumber wins over step` | `questId: "20730"`, `infoNumber: "9300285"`, `step: "1"`, `contextKey: "gate"` | as above | `"0"` | nil |
| `numeric-looking progress stays a string` | `questId: "6400"`, `contextKey: "seagull"` | `[{infoNumber: 0, progress: "72"}]` | `"72"` — a string, never an int | nil |
| `unstarted quest` | `questId: "3360"`, `contextKey: "pw"` | **404** | `""` | **nil** — an unstarted quest is a content condition, not a fault |
| `info number absent from collection` | `questId: "3360"`, `infoNumber: "5"`, `contextKey: "pw"` | `[{infoNumber: 0, progress: "x"}]` | `""` | nil |
| `empty collection` | `questId: "3360"`, `contextKey: "pw"` | `[]` | `""` | nil |
| `transport error` | `questId: "3360"`, `contextKey: "pw"` | connection failure | unchanged | **non-nil** — a dead quest service is a fault |
| `missing questId` | `contextKey: "pw"` | not called | unchanged | non-nil, message contains `questId` |
| `missing contextKey` | `questId: "3360"` | not called | unchanged | non-nil, message contains `contextKey` |
| `questId from context` | `questId: "{context.qid}"`, `contextKey: "pw"`, context `{"qid": "3360"}` | `[{infoNumber: 0, progress: "z"}]` | `"z"` | nil — proves `evaluateContextValue` is applied |

The 404-returns-nil rule is load-bearing: without it the Magatia door (Task 16)
breaks for every player who has not started quest 3360, which
`secretDoor.js` already guards against upstream.

- [ ] **Step 2: Wire the processor**

Add a progress-processor field to `OperationExecutorImpl` (lines 39-50,
alongside `savedLocationP` at 49) and construct it in `NewOperationExecutor`
(53-68, alongside line 66), using the package from Task 10.

- [ ] **Step 3: Add the operation arm**

`case "get_quest_progress":` inside `executeLocalOperation`'s switch — the same
switch that holds `get_saved_location` at line 727. Structure it exactly like
that arm: the documenting comment block, required-param extraction,
`evaluateContextValue` on `questId` (and on `infoNumber`/`step` so a context
reference works there too), the processor call, the not-found branch that
stores `""` and returns `nil`, the hard-failure branch that returns the error,
and `e.setContextValue` on success.

Scan the drained collection for the entry whose `InfoNumber()` equals the
requested one. A quest's progress set is single-digit in size, so a linear scan
is correct and no streaming concern applies.

- [ ] **Step 4: Run the operation-executor tests**

```sh
cd services/atlas-npc-conversations/atlas.com/npc && go test ./conversation/...
```

---

## Task 12: Version configuration — `messageType` tables for gms_87_1, gms_92_1, jms_185_1

`bodyKindFor` (`npc_continue_conversation.go:50-57`) resolves the wire
`lastMessageType` byte through the tenant's `messageType` table and falls
through to `bodyNone` when the byte is absent. On gms_87_1 and jms_185_1 the
handler is registered with **no `options` block at all**, so every byte falls
through: `ASK_NUMBER`, `ASK_MENU`, `ASK_AVATAR`, and `ASK_SLIDE_MENU` replies
are parsed as bodiless on both versions today. On gms_92_1 the handler is not
registered at all. Fixing this is a prerequisite for `askText`, not a side
effect of it.

### Files

- `services/atlas-configurations/seed-data/templates/template_gms_87_1.json` — add `options.messageType` to the handler at JSON path `/socket/handlers/44` (opCode `0x3F`)
- `services/atlas-configurations/seed-data/templates/template_gms_92_1.json` — add a new `NPCContinueConversationHandle` entry with opCode and `options.messageType`
- `services/atlas-configurations/seed-data/templates/template_jms_185_1.json` — add `options.messageType` to the handler at JSON path `/socket/handlers/32` (opCode `0x34`)
- `services/atlas-channel/atlas.com/channel/socket/handler/npc_continue_conversation.go` — read-only; the consumer of these tables
- `docs/tasks/task-284-npc-ask-text-state/messagetype-derivation.md` — new file

**Do not copy a table from another version.** There are four distinct
numberings already in the tree, verified by extracting every template:

| Template | handler path | opCode | `ASK_TEXT` | shape |
|---|---|---|---|---|
| gms_48_1 | `/socket/handlers/35` | `0x2F` | `3` | 9 entries; no `ASK_QUIZ`, `ASK_BOX_TEXT`, `ASK_SLIDE_MENU` |
| gms_61_1 | `/socket/handlers/39` | `0x38` | `13` | `ASK_BOX_TEXT: 3` |
| gms_72_1 | `/socket/handlers/40` | `0x3B` | `13` | as v61 |
| gms_79_1 | `/socket/handlers/40` | `0x3A` | `13` | as v61 |
| gms_83_1 | `/socket/handlers/46` | `0x3C` | `2` | 14 entries; no `SAY_IMAGE` |
| gms_84_1 | `/socket/handlers/46` | `0x3C` | `2` | identical to v83 |
| gms_95_1 | `/socket/handlers/48` | `0x41` | `3` | 15 entries; `SAY_IMAGE: 1` |

v87 sits between v84 and v95, and v92 between v87 and v95 — neither table can be
interpolated. Each is derived from its own client.

Use the `NpcConversationMessageType` names already defined at
`libs/atlas-packet/npc/clientbound/conversation.go:18-34` as the keys.

- [ ] **Step 1: Derive the byte→name table per version**

For each of gms_87_1, gms_92_1, jms_185_1, decompile
`CScriptMan::OnScriptMessage` — the switch on the message-type byte — and read
which case dispatches to which `CScriptMan::OnAsk*` handler. Addresses, from
`docs/packets/ida-exports/`:

| Version | binary | md5 | `OnScriptMessage` |
|---|---|---|---|
| gms_v87 | `GMSv87_4GB.exe` | `2e692f3ab5078e04138d264f8ea1e668` | `0x791666` |
| gms_v92 | `GMS_v92_1_DEVM.exe` | `bdef16653b92eefca2361fd5668cc509` | `0x6d1650` |
| jms_185 | `MapleStory_dump_SCY.exe (JMS v185.1)` | `af6652ff9b7c549341f35e3569d7564a` | `0x7b7160` |

The v87 and jms185 exports additionally carry every `CScriptMan::OnAsk*`
address, so the switch targets can be matched by address rather than by symbol
name — for v87: `OnAskText 0x791cd0`, `OnAskNumber 0x792020`,
`OnAskMenu 0x7921a8`, `OnAskBoxText 0x791e79`, `OnAskSlideMenu 0x792bb4`,
`OnSay 0x791828`, `OnSayImage 0x7919a9`, `OnAskYesNo 0x791b70`,
`OnAskAvatar 0x792330`, `OnAskMembershopAvatar 0x7924cc`, `OnAskPet 0x792663`,
`OnAskPetAll 0x7928f1`, `OnAskQuiz 0x792b90`, `OnAskSpeedQuiz 0x792ba2`. For
jms185: `OnAskText 0x7b77bd`, `OnAskNumber 0x7b7b0d`, `OnAskMenu 0x7b7c95`,
`OnAskBoxText 0x7b7966`, `OnAskSlideMenu 0x7b8513`, `OnSay 0x7b7315`,
`OnSayImage 0x7b7496`, `OnAskYesNo 0x7b765d`, `OnAskAvatar 0x7b7e1d`,
`OnAskPet 0x7b7fc2`, `OnAskPetAll 0x7b8250`, `OnAskQuiz 0x7b84ef`,
`OnAskSpeedQuiz 0x7b8501`.

The v92 export carries only the switch address, so its arm addresses come from
the decompilation itself.

Follow `docs/reverse-engineering.md` for reading the binary. **A byte
transcribed wrong does not crash** — it silently degrades that message type to
`bodyNone` and the reply is dropped. Transcribe from the decompilation, never
from memory of another version.

- [ ] **Step 2: Record the derivation**

Write `messagetype-derivation.md` in the task folder: per version, the binary,
its md5, the function and address decompiled, and the full byte→name table with
the address each arm dispatched to. This is what makes the tables auditable
later; the templates themselves carry no provenance.

- [ ] **Step 3: Add the tables to gms_87_1 and jms_185_1**

Add an `options` object to the existing `NPCContinueConversationHandle` entry at
`/socket/handlers/44` (gms_87_1) and `/socket/handlers/32` (jms_185_1), matching
the shape the other templates already use:

```json
"options": { "messageType": { "SAY": 0 } }
```

Leave `opCode` (`0x3F`, `0x34`) and `validator` (`LoggedInValidator`)
unchanged.

- [ ] **Step 4: Register the gms_92_1 handler**

`template_gms_92_1.json` has no `NPCContinueConversationHandle` entry. Add one
with `validator: "LoggedInValidator"`, the derived `options.messageType`, and
opCode `0x042`.

`0x042` comes from `docs/packets/audits/STATUS.md:571` — the `NPC_TALK_MORE`
row, whose per-version opcodes match every registered template exactly
(v48 `0x02F`, v61 `0x038`, v72 `0x03B`, v79 `0x03A`, v83 `0x03C`, v84 `0x03C`,
v87 `0x03F`, v92 `0x042`, v95 `0x041`, jms185 `0x034`). That is an opcode-table
fact, not a verified codec — the v92 cell in that row is ❌ — so **confirm it
against `docs/packets/registry/gms_v92.yaml` before landing.**

`tools/template-opcode-order-guard.sh` enforces a sort invariant across the
handler list, so the new entry goes at its ordered position, not appended.

- [ ] **Step 5: Run the template guards**

```sh
./tools/template-opcode-order-guard.sh
./tools/template-duplicate-binding-guard.sh
./tools/template-movement-types-guard.sh
./tools/operator-cancel-path-guard.sh
```

All four are gated on this directory (`tools/verify.sh:449-462`). Then confirm
the warn log at `npc_continue_conversation.go:52` no longer fires for `ASK_TEXT`
on any configured version — that warning is the observable symptom this task
removes (PRD §8).

---

## Task 13: Promote `NpcAskTextConversationDetail` on v84 and v92

`docs/packets/audits/STATUS.md:1022` shows the row, in column order
(v48, v61, v72, v79, v83, v84, v87, v92, v95, jms185):

```
✅ ✅ ✅ ✅ ✅ ❌ ✅ ❌ ✅ ✅
```

Exactly two cells to promote: **v84 and v92**.

### Files

- `libs/atlas-packet/npc/clientbound/conversation.go` — read-only; `AskTextConversationDetail` at lines 171-187 already encodes correctly
- `libs/atlas-packet/npc/clientbound/conversation_test.go` — add the two byte-fixture tests
- `docs/packets/audits/STATUS.md` — regenerated, not hand-edited
- `docs/packets/audits/VERIFYING_A_PACKET.md` — read-only; the playbook to follow

Patterns to copy: `conversation_test.go:201-239`
(`TestAskSlideMenuConversationDetailEncode`) is the closest structural
precedent — there is no `TestAskNumberConversationDetailEncode` or
`TestAskTextConversationDetailEncode` in the file today, only
`packet-audit:verify` markers at lines 35, 49, 62, 74.

Module root: `libs/atlas-packet`.

- [ ] **Step 1: Follow `docs/packets/audits/VERIFYING_A_PACKET.md` for each cell**

Per cell: derive the client read order from the IDB, write the byte-fixture test
carrying a `packet-audit:verify` marker, pin the evidence record, regenerate the
matrix, and commit the three artifacts together. Do not hand-edit `STATUS.md`.

v84's `CScriptMan::OnAskText` is at `0x768b6b` and its `OnScriptMessage` at
`0x76850a`; v92's `OnScriptMessage` is at `0x6d1650` and its `OnAskText` address
comes from that decompilation — Task 12 derives it too, so reuse it rather than
deriving twice.

- [ ] **Step 2: Write the fixtures**

The encoder under test, verbatim from `conversation.go:178-187`:

```go
w.WriteAsciiString(a.Message)
w.WriteAsciiString(a.Def)
w.WriteShort(a.Min)
w.WriteShort(a.Max)
```

`TestAskTextConversationDetailEncodeV84` and `…V92` — one fixture each. Use the
same input in both, so a divergence between versions shows up as a byte
difference and not as two unrelated tests:

| field | value |
|---|---|
| `Message` | `"Password!"` |
| `Def` | `""` |
| `Min` | `uint16(1)` |
| `Max` | `uint16(32)` |

Expected bytes: `WriteAsciiString` emits a length-prefixed string — read the
prefix width from `libs/atlas-packet`'s writer before writing the fixture down;
do not assume it. `Min` and `Max` are two little-endian `uint16`s, not `uint32`s.

**If the v84 or v92 read order turns out to differ from the current encoder,
stop and report it** rather than changing the encoder — it is verified on eight
other versions, and a wire change there is a different task.

- [ ] **Step 3: Confirm both cells promoted**

Regenerate the matrix and check `STATUS.md:1022` shows ✅ in the v84 and v92
columns. A cell that does not promote is a failure to report, never a prose
claim that it worked.

`npc/serverbound/NpcContinueConversation` on v92 (`STATUS.md:571`) is **out of
scope as a criterion** (PRD §4.7). If Task 12's v92 decompilation yields its
read order for free, promoting it is a bonus — never a gate.

---

## Task 14: atlas-ui — types, state metadata, transitions, and editor ops

### Files

- `services/atlas-ui/src/types/models/conversation.ts` — `AskTextState`, `AskTextMatch`, union member, optional field
- `services/atlas-ui/src/components/features/npc/conversation/stateMeta.ts` — label, icon, creation defaults, summary line
- `services/atlas-ui/src/components/features/npc/conversation/transitions.ts` — one edge per match plus the fallback edge
- `services/atlas-ui/src/components/features/npc/conversation/editorOps.ts` — creation defaults and rewire, at all three sites

askNumber reference lines, per file:

- `conversation.ts` — `AskNumberState` interface at line 67; `| "askNumber"` union member at line 121; `askNumber?: AskNumberState` on `StateModel` at line 136.
- `stateMeta.ts` — metadata entry at line 31; summary arm at lines 108-109.
- `transitions.ts` — `case "askNumber":` at line 102; the edge at lines 103-105.
- `editorOps.ts` — rewire at lines 55-57; creation defaults at lines 292-293; string-target clone at line 574.

Module root: `services/atlas-ui`.

- [ ] **Step 1: Write the failing test**

Follow whatever test runner and file layout the existing conversation-editor
tests use; if `transitions.ts` and `editorOps.ts` have no tests today, add
`transitions.test.ts` and `editorOps.test.ts` beside them.

`transitions` — given an `askText` state with `nextState: "fallback"` and
matches `[{value: "a", nextState: "sa"}, {valueFromContext: "{context.pw}",
nextState: "sb"}, {value: "c", nextState: "sa"}]`:

| assertion | expected |
|---|---|
| edge count | 4 — one per match plus the fallback |
| edge targets, in order | `sa`, `sb`, `sa`, `fallback` |
| edge labels | `"a"`, `"{context.pw}"`, `"c"`, and the fallback edge's existing default label |
| duplicate targets | both edges to `sa` are present; they are not deduplicated |
| no matches | an `askText` with `matches: []` produces exactly 1 edge, to `fallback` |

`editorOps` rename rewire — state `"sa"` renamed to `"sz"`, where an `askText`
references `sa` from `matches[0].nextState` and `matches[2].nextState` but
**not** from `nextState`:

| assertion | expected |
|---|---|
| `matches[0].nextState` | `"sz"` |
| `matches[2].nextState` | `"sz"` |
| `matches[1].nextState` | unchanged |
| `nextState` | unchanged |

`editorOps` delete rewire — deleting `"sa"` clears or re-points every
`matches[].nextState` that referenced it, matching the existing `askNumber`
delete behaviour at lines 55-57.

`editorOps` `setTransitionTarget` — the case with no precedent. Every existing
state type has at most one outgoing edge, so `setTransitionTarget` (line 574)
can assume a single target. An `askText` has a variable edge count, so the call
must address a specific match **by index**:

| case | call | expected |
|---|---|---|
| retarget match 1 | set the edge at match index 1 to `"sx"` | `matches[1].nextState === "sx"`; matches 0 and 2 and `nextState` unchanged |
| retarget fallback | set the fallback edge to `"sx"` | `nextState === "sx"`; every match unchanged |

- [ ] **Step 2: Add the types**

```ts
export interface AskTextMatch {
  value?: string;
  valueFromContext?: string;
  nextState: string;
}

export interface AskTextState {
  text: string;
  defaultText: string;
  minLength: number;
  maxLength: number;
  contextKey?: string;
  matches?: AskTextMatch[];
  nextState?: string;
}
```

Field names and optionality match the REST payload from Task 4. Add
`| "askText"` to the union at line 121 and `askText?: AskTextState` to
`StateModel` at line 136.

- [ ] **Step 3: `stateMeta.ts`**

A metadata entry beside `askNumber`'s (line 31) — label `"Ask Text"`, an icon
consistent with the sibling choices, and creation defaults
`{ text: "", defaultText: "", minLength: 0, maxLength: 32, contextKey: "answer", matches: [], nextState: "" }`.
Add a summary arm beside lines 108-109 rendering the prompt and the match count.

- [ ] **Step 4: `transitions.ts`**

A `case "askText":` beside line 102 emitting one edge per `matches` entry —
labelled with `value` when set, otherwise the `valueFromContext` reference —
plus the fallback `nextState` edge. Do not deduplicate: two matches pointing at
the same state are two edges.

- [ ] **Step 5: `editorOps.ts` — all three sites**

Lines 55-57 (rename/delete rewire), 292-293 (creation defaults), 574
(`setTransitionTarget`). The third is the one that needs real thought: it must
take the match index rather than assume a single outgoing edge.

- [ ] **Step 6: Lint, typecheck, test**

Read `services/atlas-ui/package.json` for the actual script names rather than
assuming; then run the lint, build, and test scripts it defines.

---

## Task 15: atlas-ui — the `askText` inspector panel

### Files

- `services/atlas-ui/src/components/features/npc/conversation/ConversationInspector.tsx` — import, render arm, KV rows, form component
- `services/atlas-ui/src/types/models/conversation.ts` — read-only; the types from Task 14

Patterns to copy, all in `ConversationInspector.tsx`: the `AskNumberState` import
at line 38; `case "askNumber":` render arm at line 481 with its KV rows at
484-493 (`text`, `default`, `min`, `max`, `contextKey`); the `<AskNumberForm …/>`
invocation at line 503; and the `AskNumberForm` component at lines 1259-1276+,
including the `a: AskNumberState = state.askNumber ?? {…}` props default at 1268,
the `update` helper at 1275-1276, and the `onUpdateState` call at 1322.

Module root: `services/atlas-ui`.

- [ ] **Step 1: Add the read-only view**

A `case "askText":` arm beside line 481, with KV rows for `text`, `defaultText`,
`minLength`, `maxLength`, `contextKey`, and a rendered list of matches showing
each entry's comparison target and its `nextState`.

- [ ] **Step 2: Add `AskTextForm`**

Modelled on `AskNumberForm` (1259-1276+): controlled inputs for prompt, default
text, min length, max length, and context key, each calling the `update` helper
so edits flow through `onUpdateState` (1322).

Then the part `askNumber` has no analogue for — an ordered matches editor:

- **Add** appends an entry with `{ value: "", nextState: "" }`.
- **Remove** deletes by index.
- **Reorder** moves an entry up or down by one. Order is semantic
  (first-match-wins at runtime), so this control is not cosmetic.
- Each row toggles between a literal `value` and a `valueFromContext` reference,
  clearing the other field on switch — the validator (Task 5) rejects an entry
  that sets both.

- [ ] **Step 3: Lint, typecheck, test**

Same scripts as Task 14 Step 6.

---

## Task 16: Content — `npc-2111024.json`, the Magatia lab door

Source: `<cosmic>/scripts/npc/MagatiaPassword.js` (the Cosmic checkout, not this
repository). Follow the `/convert-npc` skill.

The player's typed password is compared against the **quest progress string**
they recorded earlier — the only conversion that needs `local:get_quest_progress`.

Cosmic's `cm.getQuestProgress(3360)` is the one-argument overload, which
delegates to `getQuestProgress(id, 0)` — **info number 0**
(`<cosmic>/src/main/java/scripting/AbstractPlayerInteraction.java:414-415`).
`cm.setQuestProgress(3360, 1)` is likewise the two-argument overload,
`setQuestProgress(id, 0, "1")` — **info number 0, progress `"1"`** (`:402-403`).

### Files

- `deploy/seed/gms/48_1/npc-conversations/npc/npc-2111024.json` — new file
- `deploy/seed/gms/61_1/npc-conversations/npc/npc-2111024.json` — new file
- `deploy/seed/gms/72_1/npc-conversations/npc/npc-2111024.json` — new file
- `deploy/seed/gms/79_1/npc-conversations/npc/npc-2111024.json` — new file
- `deploy/seed/gms/83_1/npc-conversations/npc/npc-2111024.json` — new file
- `deploy/seed/gms/84_1/npc-conversations/npc/npc-2111024.json` — new file
- `deploy/seed/gms/87_1/npc-conversations/npc/npc-2111024.json` — new file
- `deploy/seed/gms/92_1/npc-conversations/npc/npc-2111024.json` — new file
- `deploy/seed/gms/95_1/npc-conversations/npc/npc-2111024.json` — new file
- `deploy/seed/jms/185_1/npc-conversations/npc/npc-2111024.json` — new file

`deploy/seed/gms/12_1/` is excluded — that template registers no NPC
conversation handlers at all (design §7). The ten files are byte-identical;
author one and copy.

Patterns to copy: `deploy/seed/gms/83_1/npc-conversations/npc/npc-2090004.json`
(the envelope and the existing `askNumber` state) and
`deploy/seed/gms/83_1/npc-conversations/npc/npc-2111003.json` (a compact
`warp_to_map` with `portalName`, three states).

- [ ] **Step 1: Author the conversation**

Envelope, verbatim from the existing seeds:

```json
{ "data": { "attributes": { "npcId": 2111024, "startState": "loadPassword", "states": [] },
            "id": "2111024", "type": "npc-conversation" } }
```

States, in order:

| id | type | content |
|---|---|---|
| `loadPassword` | `genericAction` | one operation `{"type": "local:get_quest_progress", "params": {"questId": "3360", "infoNumber": "0", "contextKey": "magatiaPassword"}}`; one outcome, no conditions, `nextState: "askPassword"` |
| `askPassword` | `askText` | `text: "The door reacts to the entry pass inserted. #bPassword#k!"`, `defaultText: ""`, `minLength: 1`, `maxLength: 32`, `contextKey: "answer"`, matches `[{"valueFromContext": "{context.magatiaPassword}", "nextState": "openDoor"}]`, `nextState: "wrongPassword"` |
| `openDoor` | `genericAction` | operations, **in this order**: `{"type": "set_quest_progress", "params": {"questId": "3360", "infoNumber": "0", "progress": "1"}}` then `{"type": "play_portal_sound", "params": {}}`; outcomes: first `[{"type": "mapId", "operator": "=", "value": "261010000"}]` → `warpJenu`, then `[]` → `warpAlca` |
| `warpJenu` | `genericAction` | `{"type": "warp_to_map", "params": {"mapId": "261030000", "portalName": "sp_jenu"}}`; outcome `[]` → `null` |
| `warpAlca` | `genericAction` | `{"type": "warp_to_map", "params": {"mapId": "261030000", "portalName": "sp_alca"}}`; outcome `[]` → `null` |
| `wrongPassword` | `dialogue` | `dialogueType: "sendOk"`, `text: "#rWrong!"`, choices `[{"text": "Ok", "nextState": null}, {"text": "Exit", "nextState": null}]` |

The `sp_jenu` / `sp_alca` split reproduces
`"sp_" + ((cm.getMapId() == 261010000) ? "jenu" : "alca")` exactly. The `mapId`
condition and the `portalName` parameter are both already supported
(`operation_executor.go:1352-1368`; `npc-2111003.json` uses `portalName`).

`secretDoor.js`'s quest-complete and quest-unstarted branches are **not** part of
this file — they belong to the portal, which owns them upstream, and a portal
conversion is a separate workstream (`context.md` D5). Do not add them here.

- [ ] **Step 2: Fan out to the other nine directories**

Copy the authored file into each of the ten paths above. Do **not** touch any
`CATALOG_REVISION` file — it holds a commit sha stamped by the image-overlay CI
(`tools/catalog-lint/main.go:44`).

- [ ] **Step 3: Validate**

Confirm each file parses and that every `nextState` names a state in the same
file or is `null`. Run the repo's seed validation — find it with
`ls tools/ | grep -i catalog` rather than assuming a script name.

---

## Task 17: Content — `npc-2111017.json`, `npc-2111018.json`, `npc-2111019.json`, the Magatia lab pipes

Sources: `<cosmic>/scripts/npc/2111017.js`, `2111018.js`, `2111019.js`.

Three NPCs sharing one structure and one password. Quest 3339 gates entry;
progress is tracked on quest **23339, info number 1**, and each pipe advances it
by one from a specific value. Verified against the three scripts:

| NPC | advances | at progress 3 | other progress `< 3` |
|---|---|---|---|
| 2111017 | `0` → `1` | prompts for the password | resets to `0` |
| 2111019 | `1` → `2` | prompts for the password | resets to `0` |
| 2111018 | `2` → `3` **and prompts in the same turn** | prompts for the password | resets to `0` |

All three, when quest 3339 is **not** started: if 3339 is completed, warp to
`261000001` portal `1`; otherwise do nothing. When progress is `> 3`: warp to
`261000001` portal `1`.

### Files

Three files × ten directories. For each of `2111017`, `2111018`, `2111019`:

- `deploy/seed/gms/48_1/npc-conversations/npc/npc-2111017.json` — new file (and `-2111018`, `-2111019`)
- `deploy/seed/gms/61_1/npc-conversations/npc/npc-2111017.json` — new file (and `-2111018`, `-2111019`)
- `deploy/seed/gms/72_1/npc-conversations/npc/npc-2111017.json` — new file (and `-2111018`, `-2111019`)
- `deploy/seed/gms/79_1/npc-conversations/npc/npc-2111017.json` — new file (and `-2111018`, `-2111019`)
- `deploy/seed/gms/83_1/npc-conversations/npc/npc-2111017.json` — new file (and `-2111018`, `-2111019`)
- `deploy/seed/gms/84_1/npc-conversations/npc/npc-2111017.json` — new file (and `-2111018`, `-2111019`)
- `deploy/seed/gms/87_1/npc-conversations/npc/npc-2111017.json` — new file (and `-2111018`, `-2111019`)
- `deploy/seed/gms/92_1/npc-conversations/npc/npc-2111017.json` — new file (and `-2111018`, `-2111019`)
- `deploy/seed/gms/95_1/npc-conversations/npc/npc-2111017.json` — new file (and `-2111018`, `-2111019`)
- `deploy/seed/jms/185_1/npc-conversations/npc/npc-2111017.json` — new file (and `-2111018`, `-2111019`)

Patterns to copy: `deploy/seed/gms/83_1/npc-conversations/npc/npc-2111003.json`
(the `questStatus` gate plus `warp_to_map` shape).

- [ ] **Step 1: Author `npc-2111017.json`**

`startState: "gate"`.

| id | type | content |
|---|---|---|
| `gate` | `genericAction` | no operations; outcomes in order: `[{"type": "questStatus", "referenceId": "3339", "operator": "=", "value": "2"}]` → `checkProgress`; `[{"type": "questStatus", "referenceId": "3339", "operator": "=", "value": "3"}]` → `warpIn`; `[]` → `null` |
| `checkProgress` | `genericAction` | no operations; outcomes in order: progress `= 3` → `askPassword`; progress `> 3` → `warpIn`; progress `= 0` → `advanceTo1`; `[]` → `resetProgress` |
| `advanceTo1` | `genericAction` | `{"type": "set_quest_progress", "params": {"questId": "23339", "infoNumber": "1", "progress": "1"}}`; outcome `[]` → `null` |
| `resetProgress` | `genericAction` | `{"type": "set_quest_progress", "params": {"questId": "23339", "infoNumber": "1", "progress": "0"}}`; outcome `[]` → `null` |
| `askPassword` | `askText` | `text: "The pipe reacts as the water starts flowing. A secret compartment with a keypad shows up. #bPassword#k!"`, `defaultText: ""`, `minLength: 1`, `maxLength: 32`, `contextKey: "answer"`, matches `[{"value": "my love Phyllia", "nextState": "unlock"}]`, `nextState: "wrongPassword"` |
| `unlock` | `genericAction` | `{"type": "set_quest_progress", "params": {"questId": "23339", "infoNumber": "1", "progress": "4"}}` then `{"type": "warp_to_map", "params": {"mapId": "261000001", "portalId": "1"}}`; outcome `[]` → `null` |
| `warpIn` | `genericAction` | `{"type": "warp_to_map", "params": {"mapId": "261000001", "portalId": "1"}}`; outcome `[]` → `null` |
| `wrongPassword` | `dialogue` | `sendOk`, `"#rWrong!"`, choices `[{"text": "Ok", "nextState": null}, {"text": "Exit", "nextState": null}]` |

The quest-progress conditions carry the info number in `step`:
`{"type": "questProgress", "referenceId": "23339", "step": "1", "operator": "=", "value": "3"}`
— `step` is a string field on the condition model (`conversation/rest.go:110`)
and `referenceId` carries the quest id.

`questStatus` values are `1` not-started, `2` started, `3` completed
(`docs/npc_conversation_conversion_spec.md:426-437`).

- [ ] **Step 2: Author `npc-2111019.json`**

Identical to `2111017` except `checkProgress`'s third outcome is progress `= 1`
→ `advanceTo2`, and `advanceTo2` sets progress `"2"`.

- [ ] **Step 3: Author `npc-2111018.json`**

Identical to `2111017` except `checkProgress`'s third outcome is progress `= 2`
→ `advanceTo3`, and `advanceTo3` sets progress `"3"` **and then routes to
`askPassword`** rather than to `null`. This is the one asymmetry in the trio:
`2111018.js` prompts in the same turn it advances, while `2111017` and
`2111019` dispose.

- [ ] **Step 4: Fan out all three files to the other nine directories**

Thirty files total; twenty-seven are copies. Do not touch `CATALOG_REVISION`.

- [ ] **Step 5: Validate**

Every `nextState` resolves within its own file or is `null`.

---

## Task 18: Content — `npc-1063011.json`, merging the Thief and Puppeteer passwords

Sources: `<cosmic>/scripts/npc/ThiefPassword.js` and
`<cosmic>/scripts/npc/PupeteerPassword.js`.

Cosmic binds **two** scripts to NPC **1063011** because the *portal* names the
script: `<cosmic>/scripts/portal/thief_in1.js` calls
`pi.openNpc(1063011, "ThiefPassword")` and
`<cosmic>/scripts/portal/enterDollcave.js` calls
`pi.openNpc(1063011, "PupeteerPassword")`. Atlas keys conversations by NPC id
(one `npc-<id>.json` per NPC), and `atlas-portal-actions` has no operation that
opens an NPC conversation, so the two must merge into one file.

Both scripts prompt with the **same** text and differ only in the accepted
password and the gate behind it, so the merge is natural: one `askText` with two
`matches`.

### Files

- `deploy/seed/gms/48_1/npc-conversations/npc/npc-1063011.json` — new file
- `deploy/seed/gms/61_1/npc-conversations/npc/npc-1063011.json` — new file
- `deploy/seed/gms/72_1/npc-conversations/npc/npc-1063011.json` — new file
- `deploy/seed/gms/79_1/npc-conversations/npc/npc-1063011.json` — new file
- `deploy/seed/gms/83_1/npc-conversations/npc/npc-1063011.json` — new file
- `deploy/seed/gms/84_1/npc-conversations/npc/npc-1063011.json` — new file
- `deploy/seed/gms/87_1/npc-conversations/npc/npc-1063011.json` — new file
- `deploy/seed/gms/92_1/npc-conversations/npc/npc-1063011.json` — new file
- `deploy/seed/gms/95_1/npc-conversations/npc/npc-1063011.json` — new file
- `deploy/seed/jms/185_1/npc-conversations/npc/npc-1063011.json` — new file

- [ ] **Step 1: Derive the doll-cave map id — or stop and ask**

`PupeteerPassword.js` has a pre-check that runs **before** the prompt: if quest
21728 is started, it shows a hint, sets progress on 21728 info 21761 to `0`, and
disposes without prompting. `ThiefPassword.js` has no such pre-check.
Reproducing that faithfully requires knowing which map the `enterDollcave`
portal sits on, so the pre-check can be gated on `mapId`.

That map id is **not derivable from this repository**: WZ map data is not checked
in (`atlas-data` reads it from a mounted volume at runtime), and neither portal
is seeded under `deploy/seed/*/portal-actions/portals/`.

Derive it from the WZ `Map.wz` portal bindings or a running `atlas-data` map
endpoint. **If neither is reachable, stop and ask.** A degraded alternative
exists — moving the pre-check to fire only after the Puppeteer password is
entered correctly, which needs no map id — but it changes behaviour for a
21728-started player who does not know the password, and choosing it is the
user's call, not the implementer's.

- [ ] **Step 2: Author the conversation**

`npcId: 1063011`, `startState: "puppeteerPreCheck"`.

| id | type | content |
|---|---|---|
| `puppeteerPreCheck` | `genericAction` | no operations; outcomes in order: `[{"type": "mapId", "operator": "=", "value": "<DOLLCAVE_MAP_ID>"}, {"type": "questStatus", "referenceId": "21728", "operator": "=", "value": "2"}]` → `puppeteerBlocked`; `[]` → `askPassword` |
| `puppeteerBlocked` | `dialogue` | `sendOk`, `"You search for any hints of the Puppeteer, but it seems a powerful force blocks the path... Better return to #b#p1061019##k."`, choices `[{"text": "Ok", "nextState": "puppeteerBlockedSetProgress"}, {"text": "Exit", "nextState": "puppeteerBlockedSetProgress"}]` |
| `puppeteerBlockedSetProgress` | `genericAction` | `{"type": "set_quest_progress", "params": {"questId": "21728", "infoNumber": "21761", "progress": "0"}}`; outcome `[]` → `null` |
| `askPassword` | `askText` | `text: "A suspicious voice pierces through the silence. #bPassword#k!"`, `defaultText: ""`, `minLength: 1`, `maxLength: 40`, `contextKey: "answer"`, matches `[{"value": "Open Sesame", "nextState": "thiefGate"}, {"value": "Francis is a genius Puppeteer!", "nextState": "puppeteerGate1"}]`, `nextState: "wrongPassword"` |
| `thiefGate` | `genericAction` | no operations; outcomes: `[{"type": "questStatus", "referenceId": "3925", "operator": "=", "value": "3"}]` → `thiefWarp`; `[]` → `thiefBlockedMessage` |
| `thiefWarp` | `genericAction` | `{"type": "warp_to_map", "params": {"mapId": "260010402", "portalId": "1"}}`; outcome `[]` → `null` |
| `thiefBlockedMessage` | `genericAction` | `{"type": "send_message", "params": {"messageType": "PINK_TEXT", "message": "Although you said the right answer, the door will not budge."}}`; outcome `[]` → `null` |
| `puppeteerGate1` | `genericAction` | no operations; outcomes: `[{"type": "questStatus", "referenceId": "20730", "operator": "=", "value": "2"}, {"type": "questProgress", "referenceId": "20730", "step": "9300285", "operator": "=", "value": "0"}]` → `puppeteerWarp`; `[]` → `puppeteerGate2` |
| `puppeteerGate2` | `genericAction` | no operations; outcomes: `[{"type": "questStatus", "referenceId": "21731", "operator": "=", "value": "2"}, {"type": "questProgress", "referenceId": "21731", "step": "9300346", "operator": "=", "value": "0"}]` → `puppeteerWarp`; `[]` → `puppeteerBlockedMessage` |
| `puppeteerWarp` | `genericAction` | `{"type": "warp_to_map", "params": {"mapId": "910510001", "portalId": "1"}}`; outcome `[]` → `null` |
| `puppeteerBlockedMessage` | `genericAction` | `{"type": "send_message", "params": {"messageType": "PINK_TEXT", "message": "Although you said the right answer, some mysterious forces are blocking the way in."}}`; outcome `[]` → `null` |
| `wrongPassword` | `dialogue` | `sendOk`, `"#rWrong!"`, choices `[{"text": "Ok", "nextState": null}, {"text": "Exit", "nextState": null}]` |

`puppeteerGate1` and `puppeteerGate2` are two sequential states because Cosmic's
gate is an `OR` of two quest conditions and a `genericAction`'s condition list is
an `AND`. Chained on failure, the two states are correct under either semantics,
so no engine change is needed either way (`design.md` §9).

`send_message` with `messageType: "PINK_TEXT"` is Cosmic's `playerMessage(5, …)`
(`operation_executor.go:2042`).

`maxLength: 40` accommodates `"Francis is a genius Puppeteer!"` (30 bytes) with
headroom; `"Open Sesame"` is 11.

`enterDollcave.js`'s quest-completed pre-branch (20730 or 21734 completed → warp
`105040201` portal `2`) belongs to the **portal**, not the NPC. Out of scope
here, exactly as `secretDoor.js`'s branches are for Task 16.

- [ ] **Step 3: Fan out and validate**

Ten files. Do not touch `CATALOG_REVISION`.

---

## Task 19: Content — `npc-2091009.json`, the Sealed Shrine entrance

Source: `<cosmic>/scripts/npc/2091009.js`.

### Files

- `deploy/seed/gms/48_1/npc-conversations/npc/npc-2091009.json` — new file
- `deploy/seed/gms/61_1/npc-conversations/npc/npc-2091009.json` — new file
- `deploy/seed/gms/72_1/npc-conversations/npc/npc-2091009.json` — new file
- `deploy/seed/gms/79_1/npc-conversations/npc/npc-2091009.json` — new file
- `deploy/seed/gms/83_1/npc-conversations/npc/npc-2091009.json` — new file
- `deploy/seed/gms/84_1/npc-conversations/npc/npc-2091009.json` — new file
- `deploy/seed/gms/87_1/npc-conversations/npc/npc-2091009.json` — new file
- `deploy/seed/gms/92_1/npc-conversations/npc/npc-2091009.json` — new file
- `deploy/seed/gms/95_1/npc-conversations/npc/npc-2091009.json` — new file
- `deploy/seed/jms/185_1/npc-conversations/npc/npc-2091009.json` — new file

- [ ] **Step 1: Author the conversation**

`npcId: 2091009`, `startState: "askPassword"`.

| id | type | content |
|---|---|---|
| `askPassword` | `askText` | `text: "The entrance of the Sealed Shrine... #bPassword#k!"`, `defaultText: ""`, `minLength: 1`, `maxLength: 40`, `contextKey: "answer"`, matches `[{"value": "Actions speak louder than words", "nextState": "checkOccupancy"}]`, `nextState: "wrongPassword"` |
| `checkOccupancy` | `genericAction` | no operations; outcomes: `[{"type": "mapCapacity", "referenceId": "925040100", "operator": ">", "value": "0"}]` → `occupied`; `[]` → `checkQuest` |
| `occupied` | `dialogue` | `sendOk`, `"Someone is already attending the Sealed Shrine."`, choices `[{"text": "Ok", "nextState": null}, {"text": "Exit", "nextState": null}]` |
| `checkQuest` | `genericAction` | no operations; outcomes: `[{"type": "questStatus", "referenceId": "21747", "operator": "=", "value": "2"}, {"type": "questProgress", "referenceId": "21747", "step": "9300351", "operator": "=", "value": "0"}]` → `warpIn`; `[]` → `blockedMessage` |
| `warpIn` | `genericAction` | `{"type": "warp_to_map", "params": {"mapId": "925040100", "portalId": "0"}}`; outcome `[]` → `null` |
| `blockedMessage` | `genericAction` | `{"type": "send_message", "params": {"messageType": "PINK_TEXT", "message": "Although you said the right answer, some mysterious forces are blocking the way in."}}`; outcome `[]` → `null` |
| `wrongPassword` | `dialogue` | `sendOk`, `"#rWrong!"`, choices `[{"text": "Ok", "nextState": null}, {"text": "Exit", "nextState": null}]` |

`mapCapacity` takes the map id in `referenceId`; the shape
`{"operator": ">=", "referenceId": "910220004", "type": "mapCapacity", "value": "5"}`
already appears in the seed set.

**Deliberate ordering deviation — record it in the conversion notes.**
`2091009.js` runs the occupancy check *before* comparing the password. Atlas
evaluates `matches` on the `askText` state itself, so the password comparison
necessarily comes first. The only behavioural difference is that a player who
types the wrong password into an occupied shrine now sees `"#rWrong!"` instead of
`"Someone is already attending the Sealed Shrine."` — the warp is gated
identically under both orderings.

- [ ] **Step 2: Fan out and validate**

Ten files. Do not touch `CATALOG_REVISION`.

---

## Task 20: Content — `npc-1092019.json`, the Nautilus seagull quiz

Source: `<cosmic>/scripts/npc/1092019.js`.

Progress lives on quest **6400, info number 1**. The script has three arms; two
convert and one is blocked.

**The `seagullProgress == 1` arm does not convert.** It routes into
`cm.getEventManager("4jaerial").startInstance(…)`, the nine-Barts instance, and
Atlas has no instance or event-manager capability reachable from a conversation.
That is an external blocker, not a prerequisite this task can produce. The arm is
simply **absent** — no placeholder state, no stub dialogue, no "coming soon"
text. Its absence is documented in the conversion notes and in Task 21's
research-doc update.

The quiz has one question and one answer, from the single-element `seagullQuestion`
and `seagullAnswer` arrays, so `seagullIdx` is always `0` and no randomisation is
needed.

### Files

- `deploy/seed/gms/48_1/npc-conversations/npc/npc-1092019.json` — new file
- `deploy/seed/gms/61_1/npc-conversations/npc/npc-1092019.json` — new file
- `deploy/seed/gms/72_1/npc-conversations/npc/npc-1092019.json` — new file
- `deploy/seed/gms/79_1/npc-conversations/npc/npc-1092019.json` — new file
- `deploy/seed/gms/83_1/npc-conversations/npc/npc-1092019.json` — new file
- `deploy/seed/gms/84_1/npc-conversations/npc/npc-1092019.json` — new file
- `deploy/seed/gms/87_1/npc-conversations/npc/npc-1092019.json` — new file
- `deploy/seed/gms/92_1/npc-conversations/npc/npc-1092019.json` — new file
- `deploy/seed/gms/95_1/npc-conversations/npc/npc-1092019.json` — new file
- `deploy/seed/jms/185_1/npc-conversations/npc/npc-1092019.json` — new file
- `docs/tasks/task-284-npc-ask-text-state/conversion-notes.md` — new file; records every deviation and dependency across Tasks 16–20

- [ ] **Step 1: Author the conversation**

`npcId: 1092019`, `startState: "gate"`.

| id | type | content |
|---|---|---|
| `gate` | `genericAction` | no operations; outcomes: `[{"type": "questStatus", "referenceId": "6400", "operator": "=", "value": "2"}]` → `branchProgress`; `[]` → `notStarted` |
| `notStarted` | `dialogue` | `sendOk`, `"Who are you talking to me? If you're just bored, go bother somebody else."`, choices `[{"text": "Ok", "nextState": null}, {"text": "Exit", "nextState": null}]` |
| `branchProgress` | `genericAction` | no operations; outcomes in order: progress `= 0` → `questionIntro`; progress `= 2` → `finalPraise`; `[]` → `null` (the omitted `progress == 1` arm falls through and disposes) |
| `questionIntro` | `dialogue` | `sendNext`, `"Ok then! I'll give you the first question now! You better be ready because this one's a hard one. Even the seagulls here think this one's pretty tough. It's a pretty difficult problem."`, choices `[{"text": "Next", "nextState": "askAnswer"}, {"text": "Exit", "nextState": null}]` |
| `askAnswer` | `askText` | `text: "One day, I went to the ocean and caught 62 Octopi for dinner. But then some kid came by and gave me 10 Octopi as a gift! How many Octopi do I have then, in total?"`, `defaultText: ""`, `minLength: 1`, `maxLength: 16`, `contextKey: "answer"`, matches `[{"value": "72", "nextState": "correct"}]`, `nextState: "incorrect"` |
| `correct` | `dialogue` | `sendNext`, `"What! I can't believe how incredibly smart you are! Incredible! In the seagull world, that kind of intellingence would give you a Ph.D. and then some. You're really amazing... I can't believe it... I simply can't believe it!"`, choices `[{"text": "Next", "nextState": "advanceProgress"}, {"text": "Exit", "nextState": "advanceProgress"}]` |
| `advanceProgress` | `genericAction` | `{"type": "set_quest_progress", "params": {"questId": "6400", "infoNumber": "1", "progress": "1"}}`; outcome `[]` → `null` |
| `incorrect` | `dialogue` | `sendOk`, `"Hmm, that's not quite how I recall it. Try again!"`, choices `[{"text": "Ok", "nextState": null}, {"text": "Exit", "nextState": null}]` |
| `finalPraise` | `dialogue` | `sendNext`, `"Ohhhh! Now that was impressive! I considered my test quite difficult, and for you to pass that... you are indeed an integral member of the Pirate family, and a friend of seagulls. We are now bonded by the mutual friendship that will last a lifetime! And, most of all, friends are there to help you out when you are in dire straits. If you are in a state of emergency, call us seagulls."`, choices `[{"text": "Next", "nextState": "finalHint"}, {"text": "Exit", "nextState": null}]` |
| `finalHint` | `dialogue` | `sendNextPrev`, `"Notify us using the skill Air Strike, and we will be there to help you out, because that's what friends are for.\r\n\r\n  #s5221003#    #b#q5221003##k"`, choices `[{"text": "Next", "nextState": "finalClose"}, {"text": "Prev", "nextState": "finalPraise"}]` |
| `finalClose` | `dialogue` | `sendNextPrev`, `"You have met all my challenges, and passed! Good job!"`, choices `[{"text": "Next", "nextState": null}, {"text": "Prev", "nextState": "finalHint"}]` |

The transcribed strings must match `1092019.js` byte for byte, including the
misspelling `"intellingence"` in `correct` and the `\r\n\r\n` plus leading
double-space in `finalHint`. **Copy them from the script; do not retype them.**

The three commented-out lines in the script's final branch (`cm.gainExp`,
`cm.teachSkill`, `cm.forceCompleteQuest`) are commented out in Cosmic too — they
are not converted.

- [ ] **Step 2: Write `conversion-notes.md`**

One file covering Tasks 16–20. Per conversion: the source script, the NPC id,
every deviation from the source and why, and every dependency left open. At
minimum:

- The omitted `seagullProgress == 1` arm and its blocker (this task).
- The occupancy/password ordering swap (Task 19).
- The NPC 1063011 merge and the map id it depends on (Task 18).
- The portal-owned branches excluded from Tasks 16 and 18.
- The quests referenced but not seeded as quest conversations: 3360, 3339,
  23339, 3925, 20730, 21728, 21731, 21747, 6400. None of the conversions
  requires one to exist — quests are read through `questStatus` /
  `questProgress` conditions and written through `set_quest_progress`.

- [ ] **Step 3: Fan out and validate**

Ten files. Do not touch `CATALOG_REVISION`.

---

## Task 21: Documentation

### Files

- `services/atlas-npc-conversations/docs/npc_conversation_schema.json` — add `askText` to the state-type enum (lines 44-47) and a full `askText` property block mirroring `askNumber` (from line 387)
- `services/atlas-npc-conversations/docs/quest_conversation_schema.json` — the same, if that schema shares the state-type enum; check before editing
- `services/atlas-npc-conversations/docs/domain.md` — describe the new state
- `docs/npc_conversation_conversion_spec.md` — the `sendGetText` → `askText` mapping, and `local:get_quest_progress` in the operations reference
- `docs/research/missing-features/npc-content.md` — new file; absent from this worktree, and its §5 records which sendGetText scripts are now converted and which remain blocked

- [ ] **Step 1: The two JSON schemas**

Add `"askText"` to the state-type enum and a property block with `text`,
`defaultText`, `minLength`, `maxLength`, `contextKey`, `matches` (an array of
objects with `value`, `valueFromContext`, `nextState`), and `nextState`.
Required: `text`, `maxLength`, `contextKey`, `nextState`. The match object
requires `nextState` and exactly one of `value` / `valueFromContext` — express
that with `oneOf` if the schema draft in use supports it, and otherwise as prose
in the description.

Field names and types must match Task 4's `RestAskTextModel` exactly, since the
schema documents the REST payload.

- [ ] **Step 2: `domain.md`**

A section for `askText` beside the existing state types, covering the prompt, the
captured value's availability as `{context.<contextKey>}`, the first-match-wins
`matches` semantics, and the trim-once/case-sensitive comparison rule.

- [ ] **Step 3: The conversion spec**

Two additions to `docs/npc_conversation_conversion_spec.md`:

- A `sendGetText` → `askText` mapping, in the style of the existing state
  sections, with a worked example. State plainly that matching is exact,
  case-sensitive, and whitespace-trimmed, and that regex and wildcards are not
  supported.
- `local:get_quest_progress` in the "Local Operations" list (which currently ends
  at `local:debug`), plus a "Detailed" subsection modelled on
  `local:get_saved_location` (lines 730-820): parameters `questId` (required),
  `infoNumber` (optional, default `0`, alias `step`), `contextKey` (required);
  behaviour on a missing quest or missing entry (stores `""`, does not fail); and
  the JavaScript mapping from `cm.getQuestProgress(id)` and
  `cm.getQuestProgressInt(id, info)`.

The `local:` prefix is not optional — an un-prefixed operation is dispatched to
the saga orchestrator and cannot write context.

- [ ] **Step 4: The missing-features research doc**

`docs/research/missing-features/npc-content.md`
does not exist in this worktree. Create it, or if the branch has since
picked it up from main, update its §5 in place. It records which `sendGetText`
scripts are now converted (`MagatiaPassword`, `2111017`, `2111018`, `2111019`,
`ThiefPassword`, `PupeteerPassword`, `2091009`, `1092019`) and which remain
blocked and on what (`2101014` expedition, `1052014` vending machine, `2010009`
guild creation, `changeName` rename, `2030013_old` dead), plus the `1092019`
nine-Barts arm.

Use repo-relative paths or placeholders throughout — never a literal home or
absolute path (enforced under `docs/`).

- [ ] **Step 5: Verify no path leaked**

```sh
grep -rn '/home/' docs/ --include='*.md' | grep -v 'docs/tasks/task-284-npc-ask-text-state/plan.md'
```

Must return nothing. The exclusion is for this plan's own self-check line, which
necessarily contains the pattern it searches for.

---

## Final gate

After every task:

```sh
tools/verify.sh
```

Flagless, exit 0. `--quick` and `--no-docker` skip the bake and `-race` and do
not count.

Then code review — a green `verify.sh` cannot see a cross-service seam defect,
and this branch has three of them: the continue-conversation contract across
atlas-channel / atlas-npc-conversations / atlas-saga-orchestrator (Task 1), the
`TEXT` conversation command between the engine producer and the channel consumer
(Tasks 6 and 9), and the quest-progress read between atlas-npc-conversations and
atlas-quest (Task 10). Trace each into its consumer by hand and confirm a test
asserts the new contract. Only then open the PR.
