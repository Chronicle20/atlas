# Design — NPC Free-Text Input State (`askText`)

Task: `task-284-npc-ask-text-state`
Phase: 2 (Design)
Input: [`prd.md`](prd.md)
Status: Draft

---

## 1. Scope of this document

The PRD fixes *what* to build. This document fixes *how*, resolves every open
question in PRD §9, and records the alternatives that were considered and
rejected. It also records four findings from grounding the PRD against real
sources (repo code, the config templates, `STATUS.md`, and the Cosmic scripts)
that change the shape of the work:

- **F1** — `ThiefPassword.js` and `PupeteerPassword.js` are both bound to
  **NPC 1063011** (`scripts/portal/thief_in1.js`, `scripts/portal/enterDollcave.js`).
  Atlas seeds NPC conversations one file per NPC id, so they must merge into a
  single conversation that branches on the character's current map.
- **F2** — `MagatiaPassword.js` is bound to **NPC 2111024**
  (`scripts/portal/secretDoor.js`), and the portal itself — not the NPC — owns
  the "quest not started / already complete" branches.
- **F3** — `STATUS.md:571` already records the v92 `NPC_TALK_MORE` opcode as
  `0x042`. The v92 handler registration has a derived opcode to start from; only
  the `messageType` table is genuinely missing.
- **F4** — the v83/v84 and v95 `messageType` tables **disagree**: v95 inserts
  `SAY_IMAGE` at 1 and shifts everything after it by one. v87, v92, and JMS185
  therefore cannot be copied from either; each must be derived from its own
  client.

Everything else in the PRD was verified accurate against the source.

---

## 2. Architecture overview

The change is a single request/response loop that already has both endpoints
built and nothing in the middle. Outbound (engine → client) and inbound
(client → engine) are designed separately because they fail independently.

```
                    atlas-npc-conversations                atlas-channel                 client
                    ────────────────────────                ─────────────                 ──────
 outbound   processAskTextState
              └─ npcSender.SendText ──► COMMAND_TOPIC_NPC_CONVERSATION
                                          type: TEXT              │
                                          body: {defaultValue,    ▼
                                                 minLength,   handleTextConversationCommand
                                                 maxLength}    └─ announceTextConversation
                                                                    └─ AskTextConversationDetail ──► input box
                                                                       {Message, Def, Min, Max}

 inbound                                                       NPCContinueConversationHandleFunc  ◄── reply
                                                                 └─ bodyKindFor → bodyText
                                                                 └─ ContinueConversationText.Text()
              Continue(..., text) ◄── COMMAND_TOPIC_NPC ◄────────── ContinueConversation(..., text)
                └─ AskTextType arm                type: CONTINUE_CONVERSATION
                     ├─ length check              body: {action, lastMessageType, selection, text}
                     ├─ choiceContext[contextKey] = text
                     └─ matches → nextState
```

Two supporting workstreams hang off this spine and are otherwise independent:

- **Configuration** — the three templates whose `messageType` table is missing
  (§7). Without this the inbound leg silently degrades to `bodyNone` and the
  reply is never read, no matter how correct the Go code is.
- **Content** — `local:get_quest_progress` plus the eight NPC conversions (§8, §9).

---

## 3. Decision: where branching on the entered text lives

### Chosen — an ordered `matches` list on the `askText` state

`AskTextMatchModel{value | valueFromContext, nextState}`, evaluated
first-match-wins inside `Continue`'s `AskTextType` arm, against the context that
already exists on `ConversationContext` at that point
(`processor.go:496-505` shows `ctx.Context()` is in hand there).

### Rejected — a new `contextEquals` condition in atlas-query-aggregator

A `genericAction` downstream of the `askText` state, with a condition comparing
`{context.answer}` to a literal. Rejected on three independent grounds:

1. **It cannot represent the comparison.** The conversation evaluator coerces
   every condition value to `int` (`conversation/evaluator.go:66`), and the
   existing `questProgress` condition (`validation/rest.go:238`) is a numeric
   step comparison. "Open Sesame" has no integer form.
2. **It costs a network round trip per branch** on a path the PRD's NFR section
   requires to stay in-process.
3. **It cannot compare two dynamic values.** The Magatia door compares the
   player's reply to a *quest progress string*, i.e. context against context.
   Conditions compare a character attribute against a literal.

### Rejected — reuse `listSelection`'s `ChoiceFromSelection` shape

`listSelection` keys on an integer index and carries a per-choice `context` map.
Text matching keys on a string and needs no per-branch context (the reply is
already stored under `contextKey`). Forcing the shapes together would add an
unused `context` field to every match and an unused `value` to every choice.

### Consequence

`matches` is a state-local, in-process branch table. It is deliberately **not**
a general expression language: exact string equality only, no regex, no
wildcards (PRD §2 non-goals). A conversation needing richer logic stores the
reply and branches with a `genericAction` on some *other* attribute, as
`ThiefPassword` does for its quest gate (§9).

---

## 4. Decision: comparison and validation semantics

Resolving PRD §9 Q1.

| Aspect | Decision | Rationale |
|---|---|---|
| Whitespace | `strings.TrimSpace` the reply **once**, before both the length check and the match; store the **trimmed** value under `contextKey`. | One canonical value. Storing raw and comparing trimmed would make `{context.answer}` disagree with the branch that was taken — a debugging trap. Cosmic compares raw, but a trailing space typed into a password box is a client artifact, not player intent. |
| Case | Exact, **case-sensitive**. | Matches Cosmic's `==`. "Open Sesame" and "my love Phyllia" are capitalised deliberately. A case-insensitive default cannot be tightened later without breaking authored content. |
| Length bounds | Re-checked server-side against `[minLength, maxLength]` after trimming; out of range → log at error with character id and state id, return an error, leave the conversation parked. | The client enforces these too, so a violation means a crafted packet. Mirrors the `askNumber` range check (`processor.go:379-392`). |
| Length unit | `len(text)` — **bytes**, as decoded by `ReadAsciiString`. | This is exactly the count the client's `Min`/`Max` constrain for a single-byte codepage. Known limitation: for a multi-byte JMS reply, byte length exceeds character length, so the server bound is stricter than the client's. Recorded in §12 rather than papered over with rune counting that would then disagree with the wire for GMS. |
| Types | `minLength`/`maxLength` are `uint16`. | `AskTextConversationDetail.Min`/`Max` are `uint16` (`libs/atlas-packet/npc/clientbound/conversation.go:170-176`). `askNumber` uses `uint32` because *its* wire fields are `uint32`; copying that here would need a lossy narrowing at the producer. |

### Empty `matches`

Absent or empty `matches` is legal and means "capture the text, always go to
`nextState`" — the Nautilus-style "store the answer, decide later" pattern.
`nextState` is required in all cases so a state can never dead-end on a
non-matching reply.

---

## 5. Decision: how the reply reaches the engine

Resolving the transport shape.

### Chosen — an additive `Text` field on the existing `CONTINUE_CONVERSATION` command

```go
type ContinueConversationCommandBody struct {   // atlas-channel
    Action          byte   `json:"action"`
    LastMessageType byte   `json:"lastMessageType"`
    Selection       int32  `json:"selection"`
    Text            string `json:"text"`
}
```

with the identical field added to the three mirrors that must not drift:

| File | Type |
|---|---|
| `services/atlas-channel/.../kafka/message/npc/kafka.go` | `ContinueConversationCommandBody` |
| `services/atlas-npc-conversations/.../kafka/message/npc/kafka.go` | `CommandConversationContinueBody` |
| `services/atlas-saga-orchestrator/.../kafka/message/npc/kafka.go:69` | `CommandConversationContinueBody` |

### Rejected — a separate `CONTINUE_CONVERSATION_TEXT` command type

It would add a second consumer arm on `COMMAND_TOPIC_NPC` for a message that is
otherwise byte-identical, and — more importantly — the two command types would
be independently ordered relative to each other only by luck. A single command
type on a character-keyed partition keeps every continuation of one
conversation strictly ordered. The empty-string cost for non-text states is
negligible.

### `Continue` signature

`Continue(npcId, characterId uint32, action, lastMessageType byte, selection int32, text string) error`
— a positional parameter, matching the existing style. There is exactly one
caller (`kafka/consumer/npc/consumer.go:102`), so the ripple is contained; the
`conversation/mock` Processor mock is regenerated to match.

Rejected: an input struct. It would be the only struct-shaped call in this
processor and buys nothing at six parameters.

### Handler change

`npc_continue_conversation.go` replaces `// TODO set return text` with
`sp.Text()`. The `action == 0` cancel path is untouched and still disposes. The
commented-out `// returnText := ""` declaration at the top of the function is
removed rather than revived, since the value is now read where it is used.

---

## 6. Decision: rendering the prompt

`processAskTextState` mirrors `processAskNumberState`
(`processor.go:1215-1237`) exactly: resolve `{context.x}` in `text`, call
`SendText`, return `state.Id()` to park.

`npcSender.Processor` gains:

```go
SendText(ch channel.Model, characterId, npcId uint32, message string,
         def string, min uint16, max uint16) error
```

backed by a `textConversationProvider` alongside `numberConversationProvider`
(`npc/producer.go:43`), emitting `CommandTypeText` with

```go
type CommandTextBody struct {
    DefaultValue string `json:"defaultValue"`
    MinLength    uint16 `json:"minLength"`
    MaxLength    uint16 `json:"maxLength"`
}
```

added to both the engine's and the channel's `conversation` message packages.

### Channel-side talk-type resolution — deliberate deviation from PRD §4.2

`getNPCTalkType` **panics** on an unrecognised talk type
(`consumer.go:241`). `announceSlideMenuConversation` already sidesteps it by
passing `npcpkt.NpcConversationMessageTypeAskSlideMenu` directly
(`consumer.go:186`). `announceTextConversation` follows that precedent and
passes `NpcConversationMessageTypeAskText` directly, so the new path cannot
panic on a config or string mismatch.

The `"TEXT"` case is **still** added to `getNPCTalkType` as PRD §4.2 requires —
it costs one line and shrinks the panic surface for any future caller — but the
announce path does not depend on it. This is a strengthening of the PRD, not a
substitution for it.

---

## 7. Decision: version configuration

Resolving PRD §4.6 and §9 Q5/Q6.

### The failure being fixed

`bodyKindFor` resolves the wire byte through the tenant `messageType` table and
falls through to `bodyNone` when the byte is absent
(`npc_continue_conversation.go:47-55`). On gms_87_1 and jms_185_1 the handler is
registered with **no `options` block at all**, so *every* byte falls through.
This does not just break `ASK_TEXT` — it means `ASK_NUMBER`, `ASK_MENU`,
`ASK_AVATAR`, and `ASK_SLIDE_MENU` replies are parsed as bodiless on both
versions today. Fixing it is a prerequisite, not a side effect.

Verified current state (`services/atlas-configurations/seed-data/templates/`):

| Template | opCode | `options.messageType` |
|---|---|---|
| gms_48_1 … gms_84_1 | `0x3C` (v83, v84) | present, `SAY:0 … ASK_SLIDE_MENU:14`, **no `SAY_IMAGE`** |
| gms_87_1 | `0x3F` | **absent** |
| gms_92_1 | — | **handler not registered** |
| gms_95_1 | `0x41` | present, `SAY:0, SAY_IMAGE:1, … ASK_SLIDE_MENU:15` |
| jms_185_1 | `0x34` | **absent** |

The v83/v84 and v95 numberings differ by the insertion of `SAY_IMAGE` at 1.
**Do not copy either table into v87, v92, or JMS185.** Each is derived from its
own client.

### Derivation procedure (per version)

For each of `gms_v87`, `gms_v92`, `gms_jms_185`:

1. Open the version's IDA export (`docs/packets/ida-exports/<key>.json`; JMS185
   uses `gms_jms_185.json`) or the live IDB.
2. Decompile the `CScriptMan` script-message dispatch — the switch that
   `CScriptMan::OnScriptMessage` performs on the message-type byte, cross-checked
   against the per-type `CScriptMan::OnAsk*` handlers named in the `NPC_TALK_MORE`
   fname list (`STATUS.md:571`).
3. Transcribe the byte→name mapping into the template's
   `NPCContinueConversationHandle.options.messageType`, using the
   `NpcConversationMessageType` names already defined in
   `libs/atlas-packet/npc/clientbound/conversation.go:18-34`.
4. Record the derivation (function, address, version) in the task folder so the
   table is auditable.

For gms_92_1 the handler entry itself is created. Its opCode starts from
`STATUS.md:571`, which records `0x042` for v92 in the `NPC_TALK_MORE` row (F3),
and is confirmed against `docs/packets/registry/gms_v92.yaml` before landing —
the matrix records an opcode-table fact, not a verified codec.

**gms_12_1 (PRD §9 Q6): out of scope, confirmed intentional.** The template is
25 KB against 128–252 KB for every other version and registers no NPC
conversation handlers whatsoever. It is a deliberately minimal early-version
template, not a gap this task fills. It is likewise excluded from the seed fan-out.

### Packet coverage (PRD §4.7, §9 Q5)

- `npc/clientbound/NpcAskTextConversationDetail` is `❌` on **v84** and **v92**
  (`STATUS.md:1022`) and both are promoted, per `VERIFYING_A_PACKET.md`:
  client read order from the IDB, byte-fixture test carrying a
  `packet-audit:verify` marker, evidence record pinned, matrix regenerated,
  three artifacts committed together.
- `npc/serverbound/NpcContinueConversation` is `❌` on v92 (`STATUS.md:1028`).
  **Decision: out of scope as an acceptance criterion**, matching PRD §4.7. The
  v92 opcode derivation needed for the handler registration happens regardless;
  if that work also yields the read order for free, promoting the cell is a
  bonus, not a gate. This is a scoping call the PRD already made, restated so
  the plan does not silently widen.

---

## 8. Decision: `local:get_quest_progress`

Resolving PRD §9 Q2 and Q3.

### Q2 — data source: a direct quest client, not atlas-query-aggregator

`atlas-npc-conversations` today has no quest client; it reaches quest state only
through `validation` (the query-aggregator). That path cannot serve this
operation:

- Conditions are **pass/fail**, not value-returning. `local:get_quest_progress` must
  yield a string into context.
- The evaluator coerces to `int` (`conversation/evaluator.go:66`).

**Chosen:** a new `quest/` package in `services/atlas-npc-conversations/atlas.com/npc/`,
following the shape already used by `saved_location/`, `map/`, and `pet/` — the
established pattern for a read that feeds a context-loading local operation
(`local:get_saved_location`, `local:fetch_map_player_counts`, `local:enumerate_evolvable_pets`).
The `local:` prefix is required — an un-prefixed operation is dispatched to the saga
orchestrator and cannot write context.

Endpoint (verified in `services/atlas-quest/atlas.com/quest/quest/resource.go:51`):

```
GET /characters/{characterId}/quests/{questId}/progress
```

returning a **paginated** JSON:API collection of
`{"infoNumber": uint32, "progress": string}`
(`quest/quest/progress/rest.go`), `404` when the character has no record for
that quest.

### Q3 — semantics for a missing quest or missing progress

Cosmic's `cm.getQuestProgress(id)` reads the progress entry at **infoNumber 0**;
`cm.getQuestProgressInt(id, info)` reads a named one. Mapping:

| Param | Required | Meaning |
|---|---|---|
| `questId` | yes | Quest template id. |
| `infoNumber` | no, default `0` | Which progress entry. (`step` accepted as an alias for author familiarity with the `questProgress` condition; `infoNumber` is canonical.) |
| `contextKey` | yes | Context key to store the result under. |

Behaviour:

- Match found → store `progress` verbatim as a **string** (never parsed), so an
  `askText` `matches` entry can reference it via `valueFromContext`.
- `404`, or a `200` whose collection has no entry with the requested
  `infoNumber` → store `""` and **return `nil`**. An unstarted quest is a
  content condition, not a fault; failing the conversation here would break the
  Magatia door for any player who has not started 3360, which the portal already
  guards against upstream.
- Transport/decode error → return the error. A dead quest service is a fault.

Pagination is handled by requesting the collection with the standard paginated
fetch and scanning for the `infoNumber`; a quest's progress set is small
(single-digit entries), so no streaming concern.

Rejected: storing the progress value as an int when it parses. It would make
`valueFromContext` comparison depend on the *content* of the data, which is
exactly the coercion bug that disqualified the query-aggregator path.

---

## 9. Decision: content conversion

Resolving PRD §4.9 and §9 Q4, grounded in the actual Cosmic sources.

### F1 — the NPC 1063011 collision

`scripts/portal/thief_in1.js` calls `pi.openNpc(1063011, "ThiefPassword")` and
`scripts/portal/enterDollcave.js` calls `pi.openNpc(1063011, "PupeteerPassword")`.
Cosmic can bind two scripts to one NPC id because the portal names the script;
Atlas keys conversations by NPC id
(`deploy/seed/<v>/npc-conversations/npc/npc-<id>.json`), and
`atlas-portal-actions` has **no** operation that opens an NPC conversation
(verified: its operation set is `warp`, `warp_to_saved_location`,
`play_portal_sound`, `save_location`, `block_portal`, `start_quest`,
`show_hint`, `start_instance_transport`, `create_skill`, `update_skill`,
`cancel_consumable_effect`, `drop_message` — none conversational).

**Decision: merge into one `npc-1063011.json`** whose start state is a
`genericAction` branching on the existing `mapId` condition (the character's
current map — `docs/npc_conversation_conversion_spec.md:394`), routing to the
Thief-hideout arm or the doll-cave arm. The two portals sit on different maps,
so the branch is total.

The two map ids are **derived from the portal bindings in the WZ data during
implementation** and are not guessed here.

Rejected: minting a synthetic second NPC id. It would not match any client
packet, since the client sends the real template id.

### F2 — MagatiaPassword scope

`scripts/portal/secretDoor.js` opens NPC **2111024** only when quest 3360 is
started *and* its progress string is longer than one character; the
quest-complete and quest-unstarted branches are handled by the portal itself.
`npc-2111024.json` therefore implements only the password exchange, exactly as
`MagatiaPassword.js` does. The portal-side rule is a separate portal conversion
and is **recorded as a dependency**, not silently absorbed.

### Per-script conversion plan

Every operation and condition named below was verified present in
`operation_executor.go` / the conversion spec; nothing here assumes a capability
that does not exist.

| # | Seed file | Source | Shape | Needs |
|---|---|---|---|---|
| 1 | `npc-2111024.json` | `MagatiaPassword.js` | `local:get_quest_progress(3360, info 0) → ctx.magatiaPassword`; `askText` with one `valueFromContext` match; on match `set_quest_progress(3360, 1)` + `play_portal_sound` + `warp_to_map(261030000, portalName sp_jenu\|sp_alca)` branched on `mapId == 261010000`; else "#rWrong!" | `local:get_quest_progress` **(new)**, `mapId` condition, `warp_to_map` `portalName` |
| 2–4 | `npc-2111017.json`, `npc-2111018.json`, `npc-2111019.json` | `2111017/18/19.js` | Entry `genericAction` on `questStatus(3339)` + `questProgress(23339, step 1)`; per-NPC progress advance (`17`: 0→1, `19`: 1→2, `18`: 2→3); at progress 3, `askText` matching literal `"my love Phyllia"` → `set_quest_progress(23339, 1, 4)` + `warp_to_map(261000001, portal 1)` | existing only |
| 5 | `npc-1063011.json` (Thief arm) | `ThiefPassword.js` | `askText` matching `"Open Sesame"` → `genericAction` on `questStatus(3925) = completed` → `warp_to_map(260010402, portal 1)`, else `send_message` PINK_TEXT | existing only |
| 6 | `npc-1063011.json` (Puppeteer arm) | `PupeteerPassword.js` | Pre-check `questStatus(21728) = started` → `sendOk` + `set_quest_progress(21728, 21761, 0)`; else `askText` matching `"Francis is a genius Puppeteer!"` → **two sequential** `genericAction` gates (20730/9300285, then 21731/9300346), each warping to `910510001` portal 1, falling through to `send_message` PINK_TEXT | existing only |
| 7 | `npc-2091009.json` | `2091009.js` | `askText` matching `"Actions speak louder than words"` → `mapCapacity(925040100)` occupancy check → `questStatus(21747) = started` + `questProgress(21747, step 9300351) = 0` → `warp_to_map(925040100, portal 0)`; else `send_message` PINK_TEXT | existing only |
| 8 | `npc-1092019.json` | `1092019.js` | Gate on `questStatus(6400) = started`; branch on `questProgress(6400, step 1)`; the `0` arm renders the question dialogue then `askText` matching `"72"` → `set_quest_progress(6400, 1, 1)`; the `2` arm renders the closing dialogue | existing only — **partial**, see below |

`send_message` maps Cosmic `playerMessage(5, …)` to `messageType: "PINK_TEXT"`
(`operation_executor.go:2042-2066`).

**Note on `genericAction` condition sets:** the Puppeteer arm's two quest gates
are an `OR` in Cosmic. If the engine's condition list is `AND`-only (confirm
during planning), it is expressed as two sequential `genericAction` states, each
with its own condition and its own warp, chained on failure — which is what the
table specifies. If `OR` turns out to be supported, one state suffices; either
way the behaviour is identical and no engine change is needed.

### Genuinely blocked, and why

- **`1092019.js`, `seagullProgress == 1` arm** — routes into
  `cm.getEventManager("4jaerial").startInstance(...)`, the nine-Barts instance.
  Atlas has no equivalent instance/event-manager capability reachable from a
  conversation. This is an external blocker, not a missing prerequisite this
  task could produce. The `0` and `2` arms convert faithfully; the `1` arm is
  recorded as a dependency in the task's conversion notes. **No placeholder
  state, no stub dialogue** — the arm is simply absent, and its absence is
  documented.
- `2101014.js` (expedition), `1052014.js` (vending machine), `2010009.js`
  (guild creation), `changeName.js` (rename) — subsystems Atlas does not
  implement; PRD §2 non-goals. Unchanged.
- `2030013_old.js` — dead (`_old`). Skipped.
- `2090004.js` — already seeded as `askNumber` (`npc-2090004.json`). Unchanged.

### PRD §9 Q4 — missing quest conversations

**Decision: record, do not seed.** The referenced quests (3360, 3339/23339,
3925, 20730, 21728, 21731, 21747, 6400) are read via `questStatus` /
`questProgress` conditions and written via `set_quest_progress`; none of that
requires a *quest conversation* to be seeded. Seeding quest conversations is a
distinct content workstream with its own scope. Any missing one is listed in the
task's conversion notes and in `docs/research/missing-features/npc-content.md`.

### Fan-out

Converted files land in
`deploy/seed/gms/{48,61,72,79,83,84,87,92,95}_1/npc-conversations/npc/` and
`deploy/seed/jms/185_1/npc-conversations/npc/` — 10 directories, matching the
existing convention that the npc-conversation seed set is identical across
versions (verified: `deploy/seed/gms/83_1/npc-conversations/npc/` holds 464
files). `gms/12_1` is excluded per §7. Seven new files × 10 directories = 70
files. Each conversion follows the `/convert-npc` skill.

---

## 10. Decision: model, REST, and editor surface

### Engine model

`AskTextType StateType = "askText"` joins the constant block
(`conversation/model.go:36-47`); `StateModel` gains an `askText *AskTextModel`
field and accessor; `StateBuilder` gains `SetAskText` and clears `askText` in
every other `SetX` (the existing pattern — 9 clear sites per setter,
`builder.go:52-249`); `Build()` gains an `AskTextType` arm.

`AskTextModel` and `AskTextMatchModel` are immutable with unexported fields and
`AskTextBuilder` / `AskTextMatchBuilder` constructors, mirroring
`AskNumberBuilder` (`builder.go:1106-1180`). No test-only constructors.

### The seven files that must move together

Verified by sweeping every non-test `askNumber` touchpoint:

| File | Change |
|---|---|
| `conversation/model.go` | type constant, struct field, accessor, model + match model |
| `conversation/builder.go` | `SetAskText`, clear sites, `Build` arm, two model builders |
| `conversation/model_json.go` | JSONB (de)serialization for storage |
| `conversation/rest.go` | NPC-conversation REST transform/extract |
| `conversation/quest/rest.go` | **quest** state-machine REST transform/extract — a parallel, duplicated REST layer (`quest/rest.go:264, 489`) that is easy to miss |
| `conversation/validator.go` | the §4.1 rules |
| `conversation/processor.go` | `processAskTextState` + the `Continue` `AskTextType` arm |

Plus `conversation/mock` for the changed `Continue` signature.

### Validation rules

As PRD §4.1, with two additions the PRD implies but does not state:

- `matches` entries are validated against the state-id set in the same pass that
  validates `nextState`, so a typo in a match target is a 400 and not a runtime
  dead-end.
- `defaultText`'s length is **not** constrained by `minLength`/`maxLength`. The
  client pre-fills it and the player may clear it; rejecting a short default
  would block the common `defaultText: ""` case.

### atlas-ui

Parity with `askNumber`, at the five sites verified to reference it:

| File | Change |
|---|---|
| `src/types/models/conversation.ts:67,121,136` | `AskTextState`, `AskTextMatch`, union member, optional field |
| `.../conversation/stateMeta.ts:31,108` | label, icon, creation defaults, summary line |
| `.../conversation/transitions.ts:102` | one edge per `matches` entry (labelled with `value` or the context ref) plus the fallback `nextState` edge |
| `.../conversation/editorOps.ts:55,292,574` | creation defaults, **and rewire over `matches[].nextState` at all three sites** — the rename/delete rewire (`:55`) and the `setTransitionTarget` path (`:574`) both currently touch only `askNumber.nextState` |
| `.../conversation/ConversationInspector.tsx` | prompt, default, min/max length, context key, ordered matches list with add / remove / reorder |

`editorOps.ts:574` is called out because a `matches` array makes the edge set
per-state variable for the first time; `setTransitionTarget` must address a
specific match by index, not assume a single outgoing edge.

---

## 11. Testing strategy

Unit tests, per `superpowers:test-driven-development`, using the project Builder
pattern for setup.

| Area | Test |
|---|---|
| Model/builder | `SetAskText` clears sibling models; `Build` rejects `AskTextType` with nil `askText` |
| JSON | round-trip of an `askText` state with and without `matches`, preserving match order |
| REST | transform/extract round-trip on **both** `conversation/rest.go` and `conversation/quest/rest.go` |
| Validator | one test per §4.1 rule, including `minLength > maxLength`, a match with neither `value` nor `valueFromContext`, a match with both, and an unknown `nextState` |
| `processAskTextState` | emits `CommandTypeText` with `{context.x}` resolved in the prompt and the right `Def`/`Min`/`Max` |
| `Continue` `AskTextType` arm | below-min and above-max rejected and logged; text stored trimmed under `contextKey`; first-match-wins across three ordered entries; `valueFromContext` resolved against existing context; fallback to `nextState` on no match and on empty `matches` |
| Downstream read | a dialogue state after an `askText` renders `{context.<contextKey>}` |
| `local:get_quest_progress` | value stored as a string; unstarted quest (404) stores `""` and returns nil; unknown `infoNumber` stores `""`; transport error propagates |
| Channel handler | the decoded reply survives handler → `ContinueConversationCommandBody.Text` (PRD acceptance criterion) |
| Channel consumer | `TEXT` command announces `AskTextConversationDetail` with the correct field values |
| Packet | byte fixtures for `NpcAskTextConversationDetail` on v84 and v92 with `packet-audit:verify` markers |
| UI | transitions emit one edge per match plus fallback; rename rewires every `matches[].nextState` |

---

## 12. Risks and limitations

| Risk | Mitigation |
|---|---|
| A `messageType` byte transcribed wrong silently degrades to `bodyNone` — no crash, no error, the reply is just dropped. | Derive from the IDB per §7, record the derivation, and cover the fix with the existing warn log (`npc_continue_conversation.go:52`), which must stop firing for `ASK_TEXT` on every configured version. |
| Deployment order: a conversation carrying an `askText` state fails to load on a binary that does not know the type. | Engine first, seed second. Stated in PRD §6; carried into the plan's ordering. |
| Byte-vs-character length on JMS multi-byte replies makes the server bound stricter than the client's (§4). | Accepted and documented. The converted content is all ASCII; revisit if JMS-specific text content is authored. |
| The NPC 1063011 merge (F1) depends on two map ids not yet derived. | Derived from WZ portal bindings during implementation; the branch is a plain `mapId` condition either way, so the design does not change if the ids surprise us. |
| `genericAction` conditions may be AND-only, affecting the Puppeteer OR gate. | Handled by construction: the design specifies two sequential states, which is correct under either semantics (§9). |
| Three separate `ContinueConversationCommandBody` mirrors can drift. | All three named explicitly in §5; a cross-service seam, so trace the field into its consumer by hand at review time per `CLAUDE.md`. |

---

## 13. Suggested implementation order

Each group is independently verifiable; groups 1–3 are the critical path.

1. **Contract** — the three `Text` field mirrors, `CommandTextBody` in both
   conversation message packages, `Continue` signature + mock.
2. **Engine outbound** — `AskTextType`/models/builder/JSON/REST (both REST
   layers)/validator, `SendText` + provider, `processAskTextState`.
3. **Engine inbound** — the `Continue` `AskTextType` arm (length check, trim,
   store, `matches`).
4. **Channel** — handler passes `sp.Text()`; `TEXT` consumer arm +
   `announceTextConversation`; `getNPCTalkType` `"TEXT"` case.
5. **`local:get_quest_progress`** — `quest/` client package + the operation arm.
6. **Configuration** — derive and land the three `messageType` tables; register
   the gms_92_1 handler.
7. **Packet coverage** — promote `NpcAskTextConversationDetail` on v84 and v92.
8. **atlas-ui** — types, stateMeta, transitions, editorOps, inspector.
9. **Content** — the seven conversions × 10 seed directories.
10. **Docs** — `npc_conversation_schema.json`, `quest_conversation_schema.json`,
    `domain.md`, `docs/npc_conversation_conversion_spec.md` (the `sendGetText` →
    `askText` mapping and the `local:get_quest_progress` operation entry),
    `docs/research/missing-features/npc-content.md` §5.

Gate: flagless `tools/verify.sh` exits 0, then code review, then PR.

---

## 14. Resolved open questions (PRD §9)

| # | Question | Resolution |
|---|---|---|
| 1 | Trim for comparison only, or also for storage? | Trim once; store trimmed. §4. |
| 2 | Quest service directly, or via atlas-query-aggregator? | Directly — a new `quest/` client package. The aggregator coerces to int and returns pass/fail. §8. |
| 3 | Atlas equivalent of `getQuestProgress(id)` with no info number? | `GET /characters/{id}/quests/{qid}/progress`, entry with `infoNumber == 0`. Missing quest or missing entry → `""`, no error. §8. |
| 4 | Seed the referenced quest conversations, or record? | Record. None of the conversions requires a quest conversation to exist. §9. |
| 5 | Promote `NpcContinueConversation` on v92? | Out of scope as a criterion; the opcode derivation happens anyway for the handler registration. §7. |
| 6 | Is gms_12_1's lack of NPC wiring intentional? | Yes — a deliberately minimal early-version template. Excluded from both config and seed work. §7. |
