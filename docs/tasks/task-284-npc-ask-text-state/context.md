# Context — NPC Free-Text Input State (`askText`)

Task: `task-284-npc-ask-text-state`
Phase: 3 (Plan)
Inputs: [`prd.md`](prd.md), [`design.md`](design.md)

This document records what was verified against the repository and the client
exports while writing [`plan.md`](plan.md), the decisions the plan encodes, and
the two places where the plan deliberately departs from — or sharpens — the
design.

---

## 1. Grounding corrections to the design

Five things in `design.md` are not quite what the repository actually holds.
The plan follows the repository.

### C1 — the operation is `local:get_quest_progress`, not `get_quest_progress`

`design.md` §8 and `prd.md` §4.5 both name the new operation
`get_quest_progress`. In `operation_executor.go` the local/remote split is by
name prefix:

```go
// operation_executor.go:317-320
func isLocalOperationType(operationType string) bool {
	return strings.HasPrefix(operationType, "local:")
}
```

An operation without the `local:` prefix is packaged into a saga and dispatched
to atlas-saga-orchestrator (`operation_executor.go:296-315`), which cannot write
into conversation context. Every sibling context-loading read is namespaced:
`local:get_saved_location` (`:727`), `local:fetch_map_player_counts` (`:634`),
`local:enumerate_evolvable_pets` (`:803`).

The plan therefore specifies **`local:get_quest_progress`**, implemented as
`case "get_quest_progress":` inside `executeLocalOperation`. This is a naming
correction, not a scope change — the behaviour is exactly what the design
specifies.

### C2 — the `messageType` tables vary far more than the design records

`design.md` §7 characterises the split as "v83/v84 vs v95, differing by
`SAY_IMAGE` at 1". The real tables, extracted from
`services/atlas-configurations/seed-data/templates/`, are:

| Template | handler JSON path | opCode | `ASK_TEXT` | notes |
|---|---|---|---|---|
| gms_12_1 | — | — | — | handler not registered; out of scope (design §7) |
| gms_48_1 | `/socket/handlers/35` | `0x2F` | `3` | 9 entries, no `ASK_QUIZ`/`ASK_BOX_TEXT`/`ASK_SLIDE_MENU` |
| gms_61_1 | `/socket/handlers/39` | `0x38` | `13` | `ASK_BOX_TEXT: 3` |
| gms_72_1 | `/socket/handlers/40` | `0x3B` | `13` | same shape as v61 |
| gms_79_1 | `/socket/handlers/40` | `0x3A` | `13` | same shape as v61 |
| gms_83_1 | `/socket/handlers/46` | `0x3C` | `2` | 14 entries, no `SAY_IMAGE` |
| gms_84_1 | `/socket/handlers/46` | `0x3C` | `2` | identical to v83 |
| gms_87_1 | `/socket/handlers/44` | `0x3F` | — | **`options` absent** |
| gms_92_1 | — | — | — | **handler not registered** |
| gms_95_1 | `/socket/handlers/48` | `0x41` | `3` | 15 entries, `SAY_IMAGE: 1` |
| jms_185_1 | `/socket/handlers/32` | `0x34` | — | **`options` absent** |

There are **four** distinct numberings in play (v48; v61–v79; v83–v84; v95), not
two. The design's conclusion holds and is strengthened: no table may be copied
into v87, v92, or JMS185. Each is derived from its own client.

The JSON paths above are recorded so the implementer edits the right handler
entry without re-searching a 190–250 KB template.

### C3 — the Cosmic scripts are outside the repository

`design.md` cites `scripts/portal/thief_in1.js` and friends as if repo-relative.
They are not in this repository; they live in the Cosmic checkout, referred to
throughout the plan as `<cosmic>/scripts/...`. Their content has been read and
the exact literals, quest ids, map ids, and progress transitions are transcribed
into the plan's task bodies, so the implementer does not need the Cosmic
checkout to do the conversions.

### C4 — `docs/research/missing-features/npc-content.md` does not exist here

The design's §13 step 10 and the PRD's acceptance list both name it. It is
untracked in the main repo and absent from this worktree, so the plan marks it
`new file`.

### C5 — atlas-npc-conversations already has a quest client

`design.md` §8 states the service "today has no quest client; it reaches quest
state only through `validation` (the query-aggregator)" and concludes that a new
top-level `quest/` package is needed. That is not the case:

```go
// services/atlas-npc-conversations/atlas.com/npc/conversation/quest/status/requests.go
func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "QUEST")
}

// RequestByCharacterAndQuest … GET /characters/{characterId}/quests/{questId}
```

and it is already consumed at
`services/atlas-npc-conversations/atlas.com/npc/conversation/quest/processor.go:147`.

Two consequences:

- The plan adds a **`progress` sibling** at
  `conversation/quest/progress/`, matching the `status/` convention, rather than
  a new top-level package. Note that `conversation/quest/` itself is the
  quest-conversation *domain* package; `status/` and the new `progress/` are the
  quest-service HTTP clients living beneath it.
- The env root url key is `"QUEST"` (singular) and is **already wired**, so no
  deployment or configuration change is needed. The design's implicit assumption
  that a new service dependency had to be plumbed does not apply.

The endpoint, verified server-side at
`services/atlas-quest/atlas.com/quest/quest/resource.go:51` and handled at
`:258-306`: `GET /characters/{characterId}/quests/{questId}/progress`, returning
**404** when the character has no record for that quest and otherwise a
**paginated** JSON:API collection of
`{"infoNumber": uint32, "progress": string}`
(`services/atlas-quest/atlas.com/quest/quest/progress/rest.go:8-12`).

There is no paginated-GET client precedent inside atlas-npc-conversations —
`saved_location/`, `map/`, and `pet/` are all single-resource reads. The pattern
comes from `libs/atlas-rest/requests/paged.go:116` (`DrainProvider`), used
concretely at
`services/atlas-npc-shops/atlas.com/npc/data/consumable/processor.go:44`.
`DrainProvider` takes a bare URL string rather than a `requests.Request`, which
is why paginated endpoints expose a URL builder — see the comment at
`services/atlas-query-aggregator/atlas.com/query-aggregator/quest/requests.go:31-34`.

---

## 2. Facts established that the plan depends on

### The saga mirror is machine-enforced, byte-for-byte

`tools/npc-conversation-contract-mirror-guard.sh` diffs
`services/atlas-npc-conversations/atlas.com/npc/kafka/message/npc/kafka.go`
against
`services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/message/npc/kafka.go`
from the `package` clause onward and fails on any difference. It is gated in
`tools/verify.sh:493` on `kafka/message/npc/kafka.go`.

This removes the design §12 risk "three mirrors can drift" for **two** of the
three: adding `Text` to one and not the other is a hard gate failure, not a
silent runtime bug. The channel's copy
(`services/atlas-channel/atlas.com/channel/kafka/message/npc/kafka.go`) is a
different shape and is **not** covered by the guard — that seam still needs the
hand-trace CLAUDE.md requires.

### The v92 opcode is a recorded fact, not a guess

`docs/packets/audits/STATUS.md:571` (`NPC_TALK_MORE` row) carries per-version
opcodes that match every registered template exactly:

```
v48 0x02F · v61 0x038 · v72 0x03B · v79 0x03A · v83 0x03C
v84 0x03C · v87 0x03F · v92 0x042 · v95 0x041 · jms185 0x034
```

`0x042` is the v92 value the plan's config task starts from, cross-checked
against `docs/packets/registry/gms_v92.yaml` before landing.

### The packet cells to promote are exactly two

`STATUS.md:1022`, `npc/clientbound/NpcAskTextConversationDetail`, column order
(v48, v61, v72, v79, v83, v84, v87, v92, v95, jms185):

```
✅ ✅ ✅ ✅ ✅ ❌ ✅ ❌ ✅ ✅
```

→ **v84 and v92**. Confirms PRD §4.7.

### `CScriptMan::OnScriptMessage` addresses, per version

From `docs/packets/ida-exports/`, so the config task's IDA work starts at a
named address rather than a search:

| Version | binary | md5 | `OnScriptMessage` |
|---|---|---|---|
| gms_v83 | `MapleStory_dump.exe (v83 Me)` | `80ff438ced539b831f0d2ed95099275d` | `0x74660a` |
| gms_v84 | (unnamed in export) | — | `0x76850a` |
| gms_v87 | `GMSv87_4GB.exe` | `2e692f3ab5078e04138d264f8ea1e668` | `0x791666` |
| gms_v92 | `GMS_v92_1_DEVM.exe` | `bdef16653b92eefca2361fd5668cc509` | `0x6d1650` |
| gms_v95 | `GMS_v95.0_U_DEVM.exe` | `3c71fd8872d5efbe16183ae8c51f887d` | `0x6de0f0` |
| jms_185 | `MapleStory_dump_SCY.exe (JMS v185.1)` | `af6652ff9b7c549341f35e3569d7564a` | `0x7b7160` |

v87 and jms185 exports additionally carry every `CScriptMan::OnAsk*` address
(the per-arm targets the switch dispatches to); v92's export has the switch
address only, so its arm addresses come from decompiling `0x6d1650`. v84's
`OnAskText` is at `0x768b6b`.

The exports record packet **read order**, not switch constants — the byte→name
table itself must come from decompiling the switch via ida-pro-mcp or the live
IDB.

### Seed JSON shape (ground truth, not inferred)

`deploy/seed/gms/83_1/npc-conversations/npc/npc-2090004.json` — the existing
`askNumber` state:

```json
{ "askNumber": { "contextKey": "quantity", "default": 1, "max": 100, "min": 1,
                 "nextState": "medicine0_confirm", "text": "How many …?" },
  "id": "medicine0_quantity", "type": "askNumber" }
```

`npc-2111003.json` — the `warp_to_map` + `portalName` shape:

```json
{ "genericAction": { "operations": [ { "params": { "mapId": "926120300",
    "portalName": "out00" }, "type": "warp_to_map" } ],
    "outcomes": [ { "conditions": [], "nextState": null } ] },
  "id": "warpToSnowRose", "type": "genericAction" }
```

Note the sibling uses `default` / `min` / `max`. The plan keeps the PRD's
`defaultText` / `minLength` / `maxLength` because the semantics differ — these
bound a *length*, not a *value* — and reusing `min`/`max` next to a numeric
`min`/`max` on a sibling state type is the kind of near-miss that produces a
wrong-but-plausible conversion.

`CATALOG_REVISION` is stamped by CI (`tools/catalog-lint/main.go:44`, the
`main-publish.yml` / `pr-validation.yml` "Stamp CATALOG_REVISION" step) and is
**never hand-edited**.

### Guards this branch will trip

From `tools/verify.sh`:

| Guard | Gate | Triggered by |
|---|---|---|
| `npc-conversation-contract-mirror-guard.sh` | `kafka/message/npc/kafka.go` | Task 1 |
| `template-opcode-order-guard.sh`, `template-duplicate-binding-guard.sh`, `template-movement-types-guard.sh` | `seed-data/templates/` | Task 12 |
| `operator-cancel-path-guard.sh` | socket handler **or** templates | Tasks 8, 12 |
| `go-analyzer-guards.sh`, `scope-guard.sh`, `producer-seam-guard.sh`, `env-domain-guard.sh` | any `services/**/*.go` | Tasks 1–11 |

Task 12 is the one most likely to fail a guard on first run, because it adds a
handler registration to `template_gms_92_1.json` and the opcode-order guard
enforces a sort invariant across the whole handler list.

---

## 3. Design decisions carried into the plan unchanged

- **`matches` is a state-local, in-process branch table** — ordered,
  first-match-wins, exact and case-sensitive, no regex. The two rejected
  alternatives (a `contextEquals` condition in atlas-query-aggregator; reusing
  `listSelection`'s shape) stay rejected for the reasons in `design.md` §3.
- **Trim once, store trimmed.** One canonical value under `contextKey`, so
  `{context.<key>}` always agrees with the branch that was taken.
- **`len(text)` is bytes.** The known JMS multi-byte limitation is accepted and
  documented rather than papered over with rune counting that would disagree
  with the wire on GMS.
- **`Text` is an additive field on the existing `CONTINUE_CONVERSATION`
  command**, not a second command type — a single command type on a
  character-keyed partition keeps continuations of one conversation ordered.
- **`announceTextConversation` passes
  `NpcConversationMessageTypeAskText` directly** rather than routing through
  `getNPCTalkType`, which panics on an unknown talk type. The `"TEXT"` case is
  still added to `getNPCTalkType` (PRD §4.2) but nothing on the new path depends
  on it.
- **`local:get_quest_progress` reads the quest service directly**, not through
  atlas-query-aggregator: conditions there are pass/fail and the evaluator
  coerces to `int` (`conversation/evaluator.go:66`), and a password has no
  integer form.
- **A missing quest or missing `infoNumber` stores `""` and returns `nil`.** An
  unstarted quest is a content condition; a dead quest service is a fault.
- **Engine first, seed second.** Task ordering reflects this: the state type,
  its JSON deserialization, and its validator all land (Tasks 2–7) before any
  seed file naming `askText` exists (Tasks 16–20).

---

## 4. Task sizing notes

Per the Step 5a rule, tasks over ~6 files or crossing a service boundary are
split. Three are deliberately left large:

- **Task 3 (builder + JSON + `Build` arm)** touches 3 files but edits 12
  sibling-clear sites in `builder.go`. It is one mechanical change repeated, so
  it batches fine and splitting it would spread a single invariant across two
  contexts.
- **Task 12 (version configuration)** edits three templates in one service. It
  is one file per version and the three derivations share a single IDA session;
  splitting it triples the IDB setup cost. Its risk is the opcode-order guard,
  not size.
- **Tasks 16, 18, 19, and 20 (content)** each convert one NPC and then fan the
  result out to 10 seed directories — 10 files, but 9 of them are byte-identical
  copies. The fan-out is a `cp` loop, not authoring. `plan-lint` flags all four
  as F4-oversized; this is the reason, and it is deliberate. Task 17 converts
  three NPCs and so writes 30 files by the same mechanism.

Two tasks were split that a naive reading would have merged:

- The contract change (Task 1) is separated from every consumer of it, because
  it crosses three services and is guard-enforced; a failure there should not be
  entangled with engine logic.
- `processAskTextState` (outbound, Task 6) and the `Continue` `AskTextType` arm
  (inbound, Task 7) are separate tasks. They fail independently, they have
  disjoint tests, and the design treats them as two legs for exactly that
  reason.

---

## 5. Open dependencies

These are recorded, not resolved. Each is either a genuine external blocker or a
derivation that must happen against a source this planning session does not
have.

| # | Item | Why it is not resolved here | Where it surfaces |
|---|---|---|---|
| D1 | The two map ids behind the NPC 1063011 merge (design F1) — which map carries the `thief_in1` portal and which carries `enterDollcave`. | WZ map data is not in this repository; `atlas-data` reads it from a mounted volume at runtime (`services/atlas-data/atlas.com/data/map/reader_test.go` shows the XML shape, not the data). Neither portal is seeded under `deploy/seed/*/portal-actions/portals/`. | Task 18. The implementer derives both from the WZ `Map.wz` portal bindings or the running `atlas-data` map endpoint. **If neither source is reachable, stop and ask — do not guess a map id.** |
| D2 | The `messageType` byte tables for gms_87_1, gms_92_1, jms_185_1. | Requires decompiling the client switch; the checked-in IDA exports carry read order, not switch constants. | Task 12, from the addresses in §2. |
| D3 | The v92 `NPC_TALK_MORE` opcode `0x042` is a matrix-recorded opcode fact, not a verified codec. | `STATUS.md:571` shows ❌ for the v92 cell. | Task 12 confirms it against `docs/packets/registry/gms_v92.yaml` before landing. |
| D4 | `1092019.js`'s `seagullProgress == 1` arm routes into `cm.getEventManager("4jaerial").startInstance(...)`. | Atlas has no instance/event-manager capability reachable from a conversation. A genuine external blocker. | Task 20 converts the `0` and `2` arms faithfully and omits the `1` arm — **no placeholder state, no stub dialogue**. The omission is documented in the conversion notes. |
| D5 | `secretDoor.js`'s quest-complete and quest-unstarted branches belong to the *portal*, not NPC 2111024. | A portal conversion is a distinct workstream. | Task 16 implements the password exchange only; the portal rule is recorded as a dependency. |
| D6 | The quest conversations for 3360, 3339/23339, 3925, 20730, 21728, 21731, 21747, 6400. | None of the conversions requires one to exist — quests are read via `questStatus`/`questProgress` conditions and written via `set_quest_progress`. | Recorded in the conversion notes and in `docs/research/missing-features/npc-content.md` (Task 21). |

---

## 6. Out of scope, restated so the plan does not widen

- `ASK_BOX_TEXT`, `ASK_QUIZ`, `ASK_SPEED_QUIZ` — no new state types.
- Character rename, guild creation, expedition, the NLC vending machine —
  subsystems Atlas does not implement (PRD §2).
- `2030013_old.js` — dead script.
- `2090004.js` — already seeded as `askNumber`; unchanged.
- `gms_12_1` — registers no NPC conversation handlers at all; excluded from both
  the config work and the seed fan-out.
- Promoting `npc/serverbound/NpcContinueConversation` on v92 — a bonus if the
  v92 opcode derivation yields the read order for free, never a gate.
