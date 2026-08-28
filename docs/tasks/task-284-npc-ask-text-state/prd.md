# NPC Free-Text Input State (`askText`) — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-08-28
---

## 1. Overview

The Atlas NPC conversation engine models a conversation as a JSON state machine. Its state types
(`services/atlas-npc-conversations/atlas.com/npc/conversation/model.go:36-47`) are `dialogue`,
`genericAction`, `craftAction`, `transportAction`, `gachaponAction`, `rpsAction`, `partyQuestAction`,
`partyQuestBonusAction`, `listSelection`, `askNumber`, `askStyle`, and `askSlideMenu`. There is no state
that prompts the player for **free text**. The MapleStory client supports this natively — the `ASK_TEXT`
NPC talk type renders an input box — and the reference server (Cosmic) uses `cm.sendGetText()` /
`cm.getText()` in 14 NPC scripts. Without an `askText` state, none of those NPCs can be converted, which
in turn blocks the Magatia laboratory door chain (quest 3360 "Verifying the Password"), the Sealed Shrine
entrance, the Puppeteer path, and the Nautilus seagull quiz.

The wire layer is already largely in place. `libs/atlas-packet/npc/clientbound/conversation.go:170` defines
`AskTextConversationDetail{Message, Def, Min, Max}` with a working `Encode`, and
`libs/atlas-packet/npc/serverbound/continue_conversation_text.go:34` decodes the player's reply string.
`ASK_TEXT` is a named message type (`conversation.go:22`), and the channel's continue-conversation handler
already classifies `ASK_TEXT`/`ASK_BOX_TEXT` as carrying a trailing text body
(`services/atlas-channel/atlas.com/channel/socket/handler/npc_continue_conversation.go:39`). The gap is
everything between those two ends: the handler decodes the reply and then **throws it away**
(`// TODO set return text`, same file), the Kafka command body has no text field
(`services/atlas-channel/atlas.com/channel/kafka/message/npc/kafka.go:33-37`), and although
`CommandTypeText = "TEXT"` is declared
(`services/atlas-channel/atlas.com/channel/kafka/message/npc/conversation/kafka.go:11`) there is no
`CommandTextBody` and no consumer arm for it, so the engine has no way to *ask* for text either.

This task closes that loop end-to-end: an `askText` state in the engine, an outbound `TEXT` conversation
command, the inbound text carried on the `ContinueConversation` command, branching on the entered value,
a supporting `get_quest_progress` operation, editor support in `atlas-ui`, the version-config gaps that
prevent `ASK_TEXT` from working on several client versions, and conversion of the Cosmic scripts this
unblocks.

## 2. Goals

Primary goals:

- Add an `askText` conversation state that renders the client's text-input dialog and captures the reply.
- Carry the player's typed string from the socket handler through Kafka into the conversation context,
  where it is addressable as `{context.<contextKey>}` like every other captured value.
- Let a conversation branch on the entered text without a round trip to the validation service, via an
  ordered `matches` list on the state itself.
- Support comparing the entered text against a character's quest progress string (the Magatia lab door),
  by adding a context-loading `get_quest_progress` operation.
- Make `ASK_TEXT` actually work on every client version whose template registers the
  continue-conversation handler.
- Convert every Cosmic `sendGetText` script that this engine work unblocks into seed JSON, and state
  plainly which ones remain blocked and on what.

Non-goals:

- `ASK_BOX_TEXT` (the multi-line box variant). The handler already classifies it as a text body; no state
  type is added for it here.
- `ASK_QUIZ` / `ASK_SPEED_QUIZ` (the timed quiz talk types).
- Character rename (`changeName.js`) — requires a rename capability in `atlas-character` that does not exist.
- Guild creation by name entry (`2010009.js`) — belongs to guild-system work.
- Expedition/Ariant-PQ support (`2101014.js`) and the NLC vending machine (`1052014.js`) — each depends on
  a subsystem Atlas does not implement.
- Regex or wildcard matching on the entered text. Matching is exact-string only (with an explicit
  case-sensitivity rule, §4.4).

## 3. User Stories

- As a player standing at the Magatia lab door, I want to type the password I recorded in quest 3360 so
  that the door opens and warps me into the lab.
- As a player at the Sealed Shrine / Thief hideout / Puppeteer path, I want to type the passphrase so that
  I am warped in when it is correct and told "Wrong!" when it is not.
- As a player taking the Nautilus seagull quiz, I want to type my answer so that a correct answer advances
  my quest progress and an incorrect one lets me retry.
- As a content author, I want an `askText` state in the conversation JSON so that I can build any
  free-text NPC without engine changes.
- As a content author using the Atlas UI conversation editor, I want to create, inspect, and rewire an
  `askText` state the same way I do an `askNumber` state.
- As an operator, I want a mistyped or over-long reply to be rejected server-side with a logged reason,
  not silently accepted.

## 4. Functional Requirements

### 4.1 The `askText` state

A new `StateType` constant `AskTextType = "askText"` is added alongside the existing types, with a
matching `AskTextModel`, a `StateBuilder.SetAskText`, REST transform/extract pair, validator rules, and
JSON (de)serialization — mirroring the `askNumber` plumbing at `model.go:603-640`, `builder.go:185+`,
`rest.go`, `model_json.go`, and `validator.go`.

`AskTextModel` fields:

| Field | Type | Required | Meaning |
|---|---|---|---|
| `text` | `string` | yes | The prompt. Supports `{context.x}` placeholders via `ReplaceContextPlaceholders`. |
| `defaultText` | `string` | no (default `""`) | Pre-filled value in the client's input box (`Def` on the wire). |
| `minLength` | `uint16` | no (default `0`) | Client-enforced minimum length (`Min` on the wire). |
| `maxLength` | `uint16` | yes | Client-enforced maximum length (`Max` on the wire). |
| `contextKey` | `string` | yes | Conversation-context key the reply is stored under. |
| `matches` | `[]AskTextMatchModel` | no | Ordered branch table, evaluated first-match-wins. |
| `nextState` | `string` | yes | Fallback next state when no match fires (or when `matches` is empty). |

`AskTextMatchModel` fields:

| Field | Type | Required | Meaning |
|---|---|---|---|
| `value` | `string` | one of `value` / `valueFromContext` | Literal comparison target. |
| `valueFromContext` | `string` | one of `value` / `valueFromContext` | A `{context.x}` reference resolved at evaluation time. |
| `nextState` | `string` | yes | State to transition to when this match fires. |

Validation rules (enforced in `validator.go`, surfaced as REST 4xx):

- `text` non-empty; `contextKey` non-empty; `nextState` non-empty and referencing an existing state id.
- `maxLength` > 0 and `minLength` <= `maxLength`.
- Each `matches` entry sets exactly one of `value` / `valueFromContext`, and its `nextState` references an
  existing state id.
- `valueFromContext` must be syntactically a context reference accepted by `ExtractContextValue`.

### 4.2 Rendering the prompt (outbound)

- `processAskTextState` mirrors `processAskNumberState` (`processor.go:1215-1237`): resolve context
  placeholders in `text`, call a new `npcSender.SendText(channel, characterId, npcId, message,
  defaultText, minLength, maxLength)`, and return the current state id to park the conversation awaiting
  input.
- `SendText` emits a `CommandEvent` on `COMMAND_TOPIC_NPC_CONVERSATION` with `Type = CommandTypeText`
  (constant already exists) and a new `CommandTextBody{DefaultValue string, MinLength uint16, MaxLength uint16}`.
- `atlas-channel`'s conversation consumer gains a `TEXT` arm and an `announceTextConversation` that builds
  `npcpkt.AskTextConversationDetail{Message, Def, Min, Max}` and announces it, matching
  `announceNumberConversation` (`consumer.go:155-168`).
- `getNPCTalkType` (`consumer.go:216-241`) gains a `"TEXT"` case returning
  `NpcConversationMessageTypeAskText`. Note that this function currently **panics** on an unknown talk
  type; the new case must be added, not relied on by default.

### 4.3 Capturing the reply (inbound)

- `NPCContinueConversationHandleFunc` replaces the `// TODO set return text` with the decoded
  `sp.Text()` and passes it through: `ContinueConversation(characterId, action, lastMessageType,
  selection, text)`.
- `ContinueConversationCommandBody` gains `Text string \`json:"text"\`` in
  `services/atlas-channel/atlas.com/channel/kafka/message/npc/kafka.go` and in the mirrored definition in
  `services/atlas-npc-conversations/atlas.com/npc/kafka/message/npc/kafka.go`. The field is additive and
  optional, so existing producers/consumers remain compatible.
- The engine's continue-conversation consumer threads `Text` into the state-advance path alongside
  `selection`.
- When `action == 0` on an `ASK_TEXT` state (the player cancelled), the existing dispose path is kept.

### 4.4 Advancing from an `askText` state

In the engine's advance switch (the `AskNumberType` arm at `processor.go:372-405` is the model):

1. Reject the input if `len(text)` is outside `[minLength, maxLength]`; log at error level with the state
   id and character id and return an error, exactly as the `askNumber` range check does. (Both bounds are
   client-enforced too; the server re-checks because the client is not trusted.)
2. Store the raw text into `choiceContext[contextKey]`.
3. Evaluate `matches` in declaration order. For a `valueFromContext` entry, resolve via
   `ExtractContextValue` against the current context first. Comparison is **exact and case-sensitive**,
   with leading/trailing whitespace trimmed from the player's input before comparing. The first entry
   whose target equals the input wins and supplies the next state.
4. If no entry matches (or `matches` is absent), advance to `nextState`.

The stored text remains available to downstream states as `{context.<contextKey>}` for use in dialogue
text and operation parameters.

### 4.5 `get_quest_progress` operation

The Magatia door compares the typed text to `cm.getQuestProgress(3360)` — the quest's progress *string*,
which is a read, not a numeric validation. The existing `questProgress` condition in
`atlas-query-aggregator` (`validation/rest.go:238`) only supports a numeric comparison against a named
step, and the whole condition path coerces values to `int` (`conversation/evaluator.go:66`), so it cannot
serve here.

A new context-loading operation `get_quest_progress` is added to `operation_executor.go`, following the
established pattern of read operations that write into conversation context (`get_saved_location`,
`fetch_map_player_counts`, `enumerate_evolvable_pets`).

- Params: `questId` (required), `step`/`infoNumber` (optional — omitted means the quest's default progress
  value), `contextKey` (required, where to store the result).
- The value is stored as a string, unmodified, so an `askText` `matches` entry can reference it via
  `valueFromContext`.
- If the quest is not started or has no progress recorded, the operation stores the empty string and does
  not fail the conversation.

### 4.6 Version configuration

`bodyKindFor` (`npc_continue_conversation.go`) resolves the wire `lastMessageType` byte through the
tenant's `messageType` table; a byte that is absent from that table falls through to `bodyNone`, which
means the trailing text is never read. Current state of the templates in
`services/atlas-configurations/seed-data/templates/`:

| Template | `NPCContinueConversationHandle` registered | `messageType` options | `ASK_TEXT` present |
|---|---|---|---|
| gms_12_1 | no | — | n/a (no NPC conversation wiring at all) |
| gms_48_1 … gms_84_1, gms_95_1 | yes | yes | yes |
| gms_87_1 | yes (`0x3F`) | **no `options` block at all** | no |
| gms_92_1 | **not registered** | — | no |
| jms_185_1 | yes (`0x34`) | **no `options` block at all** | no |

Requirements:

- gms_87_1 and jms_185_1: add a complete `options.messageType` table to their
  `NPCContinueConversationHandle` entry, with each byte derived from that version's client — never copied
  from another version and never guessed. This also repairs `ASK_NUMBER`/`ASK_MENU` selection parsing,
  which is silently broken on both today.
- gms_92_1: register `NPCContinueConversationHandle` with its opcode and `messageType` table, derived from
  the v92 client. NPC conversation continuation is entirely unwired on v92 today.
- gms_12_1: out of scope — the template has no NPC conversation wiring to extend.

### 4.7 Packet coverage

`docs/packets/audits/STATUS.md:1022` shows `npc/clientbound/NpcAskTextConversationDetail` verified on v48,
v61, v72, v79, v83, v87, v95, and JMS185, and **❌ on v84 and v92**. Both cells are promoted to verified in
this task by following `docs/packets/audits/VERIFYING_A_PACKET.md` — client read order derived from the
IDB, byte-fixture test with a `packet-audit:verify` marker, evidence record pinned, matrix regenerated.

`STATUS.md:571` also shows `npc/serverbound/NpcContinueConversation` ❌ on **v92**, consistent with §4.6's
finding that v92 never registers the handler. Verifying that cell is in scope only insofar as v92's
handler registration requires deriving the opcode; promoting the serverbound cell is a stretch goal, not
an acceptance criterion.

### 4.8 Editor support (atlas-ui)

`askText` is added to the conversation editor with parity to `askNumber`:

- `src/types/models/conversation.ts` — the `AskText` state shape and its union member.
- `src/components/features/npc/conversation/stateMeta.ts` — label, icon, and creation defaults.
- `.../transitions.ts` — outgoing edges: one per `matches` entry plus the fallback `nextState`.
- `.../editorOps.ts` — rewire on state rename/delete (`editorOps.ts:55-57`, `:292`) must cover both
  `nextState` and every `matches[].nextState`.
- `.../ConversationInspector.tsx` — an editor panel for prompt, default, min/max length, context key, and
  the ordered matches list (add/remove/reorder).

### 4.9 Content conversion

The 14 Cosmic scripts that call `sendGetText`, and their disposition:

| Script | NPC / purpose | Disposition |
|---|---|---|
| `MagatiaPassword.js` | Magatia lab door, password vs quest 3360 progress | **Convert** (needs `get_quest_progress`) |
| `2111017.js`, `2111018.js`, `2111019.js` | Magatia lab pipes, "my love Phyllia" | **Convert** |
| `ThiefPassword.js` | "Open Sesame", gated on quest 3925 completed | **Convert** |
| `2091009.js` | Sealed Shrine, "Actions speak louder than words" | **Convert** |
| `PupeteerPassword.js` | "Francis is a genius Puppeteer!", quest-gated warps | **Convert** |
| `1092019.js` | Nautilus seagull quiz, answer vs question index | **Convert** |
| `2090004.js` | Craft quantity prompt | **Already seeded** as `askNumber` (`npc-2090004.json`, all versions) — no change |
| `2101014.js` | Ariant PQ participant limit | **Blocked** — expedition system not implemented |
| `1052014.js` | NLC vending machine ticket quantity | **Blocked** — vending-machine subsystem not implemented |
| `2010009.js` | Guild Union name entry | **Blocked** — guild creation flow, out of scope (§2) |
| `changeName.js` | Character rename | **Blocked** — no rename capability, out of scope (§2) |
| `2030013_old.js` | Zakum instance password | **Skip** — dead script (`_old`) |

Converted NPCs are written to `deploy/seed/gms/{48,61,72,79,83,84,87,92,95}_1/npc-conversations/npc/` and
`deploy/seed/jms/185_1/...`, matching the existing convention that the npc-conversation seed set is
identical across versions. `gms/12_1` is excluded (no NPC conversation wiring). Each conversion follows the
`/convert-npc` skill.

`PupeteerPassword.js` and `1092019.js` reference quests (20730/21731/21728, 6400) whose conversations may
not be seeded; where a referenced quest conversation is missing, the NPC is still converted and the
dependency is recorded in §9, not silently dropped.

## 5. API Surface

No new HTTP endpoints. The existing NPC conversation REST resource
(`services/atlas-npc-conversations/atlas.com/npc/conversation/rest.go`) gains an optional `askText`
attribute on the state object, following JSON:API conventions like its `askNumber` sibling:

```jsonc
{
  "id": "door-password",
  "type": "askText",
  "askText": {
    "text": "The door reacts to the entry pass inserted. #bPassword#k!",
    "defaultText": "",
    "minLength": 1,
    "maxLength": 32,
    "contextKey": "answer",
    "matches": [
      { "valueFromContext": "{context.magatiaPassword}", "nextState": "open-door" }
    ],
    "nextState": "wrong-password"
  }
}
```

Error cases (HTTP 400, JSON:API error object) on create/update: missing `text` / `contextKey` /
`nextState` / `maxLength`; `minLength > maxLength`; a `matches` entry with neither or both of
`value`/`valueFromContext`; any `nextState` referencing an unknown state id.

Kafka contract changes (both additive):

- `COMMAND_TOPIC_NPC_CONVERSATION` — new `type: "TEXT"` with
  `body: { defaultValue: string, minLength: number, maxLength: number }`.
- `COMMAND_TOPIC_NPC` `CONTINUE_CONVERSATION` — new optional `body.text: string`.

## 6. Data Model

No relational schema change. NPC conversations are stored as JSONB
(`services/atlas-npc-conversations/docs/storage.md`, `conversations` table) and are tenant-scoped by the
existing `tenant_id` column, so the new state type is absorbed by the existing document column and
requires no migration. Conversations carrying an `askText` state simply fail to load on an older binary
that does not know the type — deployment order is engine first, seed second.

Documentation artifacts to update:

- `services/atlas-npc-conversations/docs/npc_conversation_schema.json` — add `askText` to the state-type
  enum (`:44-47`) and a full `askText` property block mirroring `askNumber` (`:387+`).
- `services/atlas-npc-conversations/docs/quest_conversation_schema.json` — same, if quest state machines
  share the state-type enum.
- `services/atlas-npc-conversations/docs/domain.md` — describe the new state.
- `docs/npc_conversation_conversion_spec.md` — document the `sendGetText` → `askText` mapping.
- `docs/research/missing-features/npc-content.md` §5 — update once landed.

## 7. Service Impact

| Service / library | Change |
|---|---|
| `libs/atlas-packet` | No wire change. Byte-fixture verification for `NpcAskTextConversationDetail` on v84 and v92. |
| `services/atlas-channel` | Handler passes decoded text through; `ContinueConversationCommandBody.Text`; producer/processor signature; `CommandTextBody` + `TEXT` consumer arm + `announceTextConversation`; `getNPCTalkType` `"TEXT"` case. |
| `services/atlas-npc-conversations` | `AskTextType` + `AskTextModel` + `AskTextMatchModel`; builder, REST, JSON, validator; `processAskTextState` and the advance arm; `SendText` on the npc sender processor; `get_quest_progress` operation; schema/docs. |
| `services/atlas-saga-orchestrator` | Mirrors the npc command constants (`kafka/message/npc/kafka.go:30`); keep the `ContinueConversationCommandBody` definition in sync if it carries one. |
| `services/atlas-query-aggregator` | Only if `get_quest_progress` reads through it rather than the quest service directly — a design-phase decision. |
| `services/atlas-configurations` | `messageType` tables for gms_87_1, jms_185_1; handler registration + table for gms_92_1. |
| `services/atlas-ui` | `askText` in types, stateMeta, transitions, editorOps, ConversationInspector. |
| `deploy/seed` | 7 new NPC conversation JSON files × 10 seed versions. |

## 8. Non-Functional Requirements

- **Multi-tenancy:** every new path resolves tenant from context; the `messageType` byte is read from
  tenant configuration and never hard-coded (the existing `bodyKindFor` contract).
- **Security / input handling:** the typed string is player-controlled input. It is length-checked
  server-side before use, stored as an opaque context value, and never interpolated into a query, a
  filesystem path, or a shell invocation. Text used in a subsequent dialogue is emitted through the same
  path as any other context value.
- **Observability:** a rejected reply (length out of bounds) logs at error level with character id and
  state id. A `lastMessageType` byte missing from the tenant `messageType` table already logs a warning;
  that warning must not fire for `ASK_TEXT` on any configured version once §4.6 lands.
- **Backwards compatibility:** both Kafka additions are optional fields on existing message shapes; a
  conversation with no `askText` state behaves identically before and after.
- **Performance:** no additional service round trip on the hot path. `matches` are evaluated in-process
  against the conversation context; `get_quest_progress` is an explicit, author-controlled operation.
- **Immutability / style:** models are immutable with builder construction and no test-only constructors,
  per `backend-dev-guidelines`.

## 9. Open Questions

1. Should whitespace trimming of the player's reply (§4.4) apply to `matches` comparison only, or also to
   the value stored in `contextKey`? Proposed: trim for comparison, store trimmed.
2. Does `get_quest_progress` read the quest service directly or go through `atlas-query-aggregator`?
   Design-phase decision; the aggregator's existing `questProgress` support is numeric-only.
3. Cosmic's `getQuestProgress(3360)` with no info-number returns the quest's default progress string. The
   exact Atlas equivalent for a quest with no recorded progress needs confirming against the quest service
   before the Magatia conversion can be asserted correct.
4. Are the quests referenced by the converted scripts (3360, 3925, 20730, 21728, 21731, 6400) seeded as
   quest conversations? `docs/research/missing-features/npc-content.md` lists quest 3360 among the missing
   quest conversations. If a dependency is missing, does this task seed it or record it?
5. Promoting `npc/serverbound/NpcContinueConversation` on v92 (§4.7) — in or out?
6. gms_12_1 has no NPC conversation wiring at all. Confirm this is intentional (a deliberately minimal
   early-version template) rather than a gap this task should fill.

## 10. Acceptance Criteria

- [ ] `AskTextType`/`AskTextModel`/`AskTextMatchModel` exist with builder, REST transform/extract, JSON
      round-trip, and validator coverage; unit tests cover each validation rule in §4.1.
- [ ] `processAskTextState` emits a `TEXT` conversation command with resolved context placeholders in the
      prompt; `atlas-channel` consumes it and announces `AskTextConversationDetail` with the correct
      `Message`/`Def`/`Min`/`Max`.
- [ ] The socket handler no longer discards the decoded reply; `ContinueConversationCommandBody.Text`
      carries it to the engine. A test asserts the text survives the full handler → command → advance path.
- [ ] A reply shorter than `minLength` or longer than `maxLength` is rejected server-side and logged.
- [ ] `matches` evaluate first-match-wins, support both `value` and `valueFromContext`, and fall back to
      `nextState`; covered by unit tests including a `valueFromContext` case.
- [ ] The reply is readable from a later state as `{context.<contextKey>}`.
- [ ] `get_quest_progress` stores a quest's progress string into conversation context and returns the
      empty string (not an error) for an unstarted quest.
- [ ] `template_gms_87_1.json` and `template_jms_185_1.json` carry a complete `messageType` options table
      including `ASK_TEXT`, with every byte derived from the corresponding client.
- [ ] `template_gms_92_1.json` registers `NPCContinueConversationHandle` with its derived opcode and
      `messageType` table.
- [ ] `docs/packets/audits/STATUS.md` shows `npc/clientbound/NpcAskTextConversationDetail` ✅ on v84 and
      v92, with pinned evidence records and `packet-audit:verify`-marked fixtures.
- [ ] The atlas-ui conversation editor can create, inspect, rewire, and delete an `askText` state,
      including its `matches` edges; renaming a state rewires every `matches[].nextState`.
- [ ] Seed JSON exists in all 10 seed version directories for: `MagatiaPassword` (2111011-range door),
      `2111017`, `2111018`, `2111019`, `ThiefPassword`, `2091009`, `PupeteerPassword`, and `1092019`.
- [ ] `npc_conversation_schema.json`, `domain.md`, and the conversion spec document `askText`; the
      research doc §5 is updated.
- [ ] Flagless `tools/verify.sh` exits 0.
- [ ] Code review completed before the PR is opened.
