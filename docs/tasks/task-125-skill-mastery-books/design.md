# Skill Books and Mastery Books — Design

Version: v2
Status: Draft (for review)
Created: 2026-07-02
Revised: 2026-07-25 (post-`main` merge)
Inputs: `docs/tasks/task-125-skill-mastery-books/prd.md` (approved)

---

## 0. Revision Note (v2 — 2026-07-25)

After merging `main`, the supported client-version set grew and a full IDB-derived
re-verification overturned three assumptions in the v1 design. **The saga /
consumables / compensation architecture (§1–§5.5) is unchanged and still correct;
only the version scope, the clientbound codec shape, the template wiring, and the
fixture campaign changed.**

- **Version scope: 5 → 10.** Wire `gms_12`? **No** (feature genuinely absent).
  All other templates carry the feature: `gms_48, gms_61, gms_72, gms_79, gms_83,
  gms_84, gms_87, gms_92, gms_95, jms_185`. The v1 "gms_92 excluded" decision is
  **reversed** — its opcodes are now IDB-verified.
- **Every version uses the dedicated `CWvsContext::SendSkillLearnItemUseRequest`
  send + `CWvsContext::OnSkillLearnItemResult` receive.** There is **no** generic
  item-use routing to build — legacy clients have the dedicated packet too. (A
  brief mid-design hypothesis that legacy used generic item-use was disproved by
  decompiling the send in every IDB.)
- **The audit CSV / `docs/packets/audits/support/*.md` are UNRELIABLE for this
  feature** — they marked legacy serverbound cells `n-a` and `gms_48` clientbound
  `n-a`/`0x000`, yet the packets demonstrably exist. **All opcodes below are
  derived directly from the IDBs** (dispatch is base-0 in every version:
  `CClientSocket::ProcessPacket` forwards the raw wire opcode unmodified to
  `CWvsContext::OnPacket`, so switch-case == wire opcode).
- **Real clientbound divergence:** the result body gains a leading
  `bOnExclRequest` byte at **v84** (a `v84 ≠ v83` exception — serverbound v84 is
  byte-identical to v83). See §5.1 and the verified matrix there.

## 1. Approach Summary

The flow is: **atlas-channel handler → new `REQUEST_SKILL_BOOK_USE` command → atlas-consumables validates + rolls → two-step saga (destroy book → conditional create/update skill) → consumables gates the result on the saga-status event → new `SKILL_BOOK_RESULT` status event → atlas-channel writer broadcasts `SKILL_LEARN_ITEM_RESULT`.**

Key architectural facts discovered during design (each shaped a decision below):

- **atlas-consumables uses no sagas today.** Its item flows run on the inventory reservation protocol (`RequestReserve` + one-time handler on `EVENT_TOPIC_COMPARTMENT_STATUS`, then `ConsumeItem`) — see `services/atlas-consumables/atlas.com/consumables/consumable/processor.go:515-579` (scroll). This task introduces the service's first saga submission (§4-A weighs this against staying on the reservation idiom).
- **`update_skill` clobbers.** atlas-skills' `Update` applies `SetLevel/SetMasterLevel/SetExpiration` unconditionally via GORM `.Select(columns)` — zero values are written (`services/atlas-skills/atlas.com/skills/skill/processor.go:148-178`, `administrator.go:48-58`). There is no partial update. **The skill step payload must carry the character's current Level and Expiration.**
- **The orchestrator emits saga completion events** consumables can gate on: `EVENT_TOPIC_SAGA_STATUS` with `COMPLETED`/`FAILED`, keyed by `TransactionId` (`services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/message/saga/kafka.go:12-14`); the failed body names the failed step. Five services already consume this topic (channel, npc-conversations, portal-actions, map-actions, character-factory).
- **`destroy_asset_from_slot` has no compensation and no template check.** `DestroyAssetFromSlotPayload` is `{CharacterId, InventoryType, Slot, Quantity, ShowEffect}` (`libs/atlas-saga/payloads.go:101-108`) — no item id — and `CompensateFailedStep` has no case for it (falls to a no-op default) (`services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/handler.go`, `compensator.go:239-293`). The PRD's compensation acceptance criterion requires adding both (§5.4).
- **Cosmic reference read** (`SkillBookHandler.java`, real source, not memory) resolves D-1/D-3 — details in §2.
- **Consumables already has the REST clients it needs except skills**: character (`character/model.go:101,121` — job, HP), full inventory (`inventory/processor.go:28` `GetByCharacterId`, `RootUrl("INVENTORY")`), and atlas-data consumables (`data/consumable/rest.go:56-101` — `success`/`masterLevel`/`reqSkillLevel`/`skills` parsed for both 228 and 229 with no classification gaps). It has **no atlas-skills client** (gap to fill), and the `data/consumable` `Model` stores `masterLevel`/`reqSkillLevel`/`skills` **without accessors** (gap to fill).

## 2. Resolved Open Questions

### D-1 — Skill-book (228) create-vs-update

**Resolved from Cosmic source** (`SkillBookHandler.java:71-87`, `Character.java:1815-1817`): the reference has **no "record must exist" gate**. `changeSkillLevel` inserts a new `SkillEntry` when absent, so a book whose `reqSkillLevel == 0` can teach an unlearned skill at **level 0** with the granted master level. Cosmic does not branch on 228 vs 229 at all — behavior is purely data-driven.

**Atlas decision:** honor both the reference and PRD FR-2.5:
- **228 (skill book):** if no skill record exists and the `reqSkillLevel` gate passes (i.e. `reqSkillLevel == 0`), the saga uses **`create_skill`** with `Level=0, MasterLevel=<book>, Expiration=time.Time{}` (zero time = permanent, matching `services/atlas-messages/atlas.com/messages/command/character/skill/commands.go:70`).
- **229 (mastery book):** PRD FR-2.5 stands — a record with `Level ≥ 1` is required; otherwise reject with `canUse=0`. (In v83 data mastery books carry `reqSkillLevel ≥ 1`, so this matches Cosmic's effective behavior while being explicit.)
- When a record exists (either classification), the saga uses **`update_skill`** with `Level=<current>, MasterLevel=<book>, Expiration=<current>` — current values carried to defeat the clobber (§1).

### D-2 — Rejection routing

**Requester-only.** Validation rejections and saga failures emit the result event with `canUse=0`, and the channel consumer writes it only to the requester's session (`IfPresentByCharacterId`). Success/failure-roll results (`canUse=1`) broadcast to the whole map via `ForSessionsInMap` — the client demuxes locally (glow for everyone, sound/message only for the local user, per the PRD's IDA findings). This matches the scroll consumer's split (`services/atlas-channel/atlas.com/channel/kafka/consumer/consumable/consumer.go:57-101`).

### D-3 — `success` semantics

**Percent out of 100**, confirmed in both places:
- Cosmic: `rollSuccessChance(success)` reduces to pass-probability `success/100` with the default single roll (`ItemInformationProvider.java:625-631`).
- atlas-data: `Success = i.GetIntegerWithDefault("success", 0)` (`services/atlas-data/atlas.com/data/consumable/reader.go:56`) — **defaults to 0, not 100**, when the WZ node omits it.

**Roll:** `rand.Int31n(100) < int32(successRate)`. Note this deliberately uses `<`, not the scroll path's `<=` (`consumable/processor.go:642-645`) — `<=` gives `success=0` a 1% pass rate and shifts every book +1%. `success=0` must never pass; `success=100` always passes.

**Verification item (plan phase):** the repo contains no WZ XML for 0228/0229; enumerate the local v83 WZ to confirm every book carries an explicit `success` node (any book missing it would be unusable at 0% — surface such items rather than silently shipping them).

Cosmic consumes the book **before** the roll and unconditionally once eligible (`SkillBookHandler.java:79-85`) — consume-on-fail confirmed.

### D-4 — Result packet field order (IDB-verified, all 10 versions)

Per-version reads are now pinned (not deferred to fixtures). The clientbound body is:

```
[Decode1 bOnExclRequest]   ← v84+ ONLY (leading byte; server sends 1)
 Decode4 characterId
 Decode1 isMasteryBook
 Decode4 skillId            ← client decodes then discards
 Decode4 masterLevel        ← client decodes then discards
 Decode1 canUse             ← gates the on-avatar glow + clears the excl-request lock
 Decode1 success            ← success vs failure sound/message (local user only)
```

The v1 concern about the "third/fourth ints" is resolved: `skillId` and `masterLevel`
are decoded-then-discarded by the client in every version (v48→jms), but Atlas still
sends the real values per PRD FR-4.1 (do **not** copy Cosmic's hardcoded zeros from
`SkillBookHandler.java:99-100`). The `bOnExclRequest` byte (v84+) tells the requesting
(local) client to clear its exclusive-request lock; observers ignore it, so the server
sends `1` on every result. See §5.1 for the version gate.

### FR-2.4 clarification — target-skill selection

Cosmic selects the **first entry in `skills[]` whose job prefix equals the character's job id**: `curskill / 10000 == playerJob` (`ItemInformationProvider.java:1481-1483`); no match ⇒ unusable. Atlas adopts the same rule: `skillId / 10000 == uint32(characterJobId)`. Check `libs/atlas-constants/skill` for an existing job-prefix helper before writing one (DOM-21).

## 3. Architecture

```
client ──USE_SKILL_BOOK──▶ atlas-channel
                             handler CharacterSkillBookUseHandle (decode, no logic)
                             │  REQUEST_SKILL_BOOK_USE on COMMAND_TOPIC_CONSUMABLE
                             ▼
                           atlas-consumables  RequestSkillBookUse
                             1. validate (character, classification, slot, job match,
                                skill state, reqSkillLevel, master ceiling)
                             ├─ reject ──▶ SKILL_BOOK_RESULT{canUse:0} ─────────────┐
                             2. roll success (rand < success%)                      │
                             3. register one-time handler on EVENT_TOPIC_SAGA_STATUS│
                             4. submit saga (COMMAND_TOPIC_SAGA, txId)              │
                             ▼                                                      │
                           atlas-saga-orchestrator  type skill_book_use             │
                             step 1: destroy_asset_from_slot (book, always)        │
                             step 2: update_skill | create_skill (success roll only)│
                             compensation: step 2 failed ⇒ re-award book            │
                             │  COMPLETED / FAILED (txId)                           │
                             ▼                                                      │
                           atlas-consumables one-time handler                       │
                             │  SKILL_BOOK_RESULT on EVENT_TOPIC_CONSUMABLE_STATUS ◀┘
                             ▼
                           atlas-channel consumer
                             canUse=1 → ForSessionsInMap (map broadcast)
                             canUse=0 → IfPresentByCharacterId (requester only)
                             writer CharacterSkillLearnItemResult → SKILL_LEARN_ITEM_RESULT
```

The client's skill-window refresh needs no new work: the saga's `create_skill`/`update_skill` makes atlas-skills emit `CREATED`/`UPDATED` on `EVENT_TOPIC_SKILL_STATUS`, which atlas-channel's existing skill consumer already announces as `CharacterSkillChange` to the requester (`services/atlas-channel/atlas.com/channel/kafka/consumer/skill/consumer.go:76,94`). Do not duplicate that packet in the new consumer.

## 4. Alternatives Considered

### A — Saga shape (the core decision)

**Chosen: full two-step saga with new compensation support (PRD-literal).**
Steps `destroy_asset_from_slot` → conditional `create_skill`/`update_skill`; a saga-type-specific reverse walk re-awards the book when the skill step fails. Rationale: the saga durably owns the invariant (orchestrator state persists across a consumables crash mid-flight), the PRD's atomicity goal and acceptance criterion ("saga compensation verified") are met literally, and destroy-first makes duplicate requests safe (second saga's destroy fails on an empty slot ⇒ saga FAILED ⇒ `canUse=0`; no double master-level grant).

**Rejected: reservation lock + skill-only saga.** Reserve the book (existing consumables idiom), submit a skill-update saga, then `ConsumeItem` on COMPLETED / `CancelItemReservation` on FAILED. Attractive because rollback is a reservation-cancel (book never leaves the slot, can't land in a different slot) and the orchestrator needs zero changes. Rejected because: a single-step saga adds no atomicity over a direct skill command (making the saga decorative); the consume/cancel decision lives in consumables' memory, so a crash between saga completion and `ConsumeItem` leaves a raised master level with the book still reserved; and the PRD explicitly frames destruction as a saga step.

**Rejected: optimistic result emission** (emit `SKILL_BOOK_RESULT` right after submitting the saga). Simplest, but shows success while the saga can still fail — the client would render a masterLevel raise that never lands. Violates the PRD's honesty about atomicity.

### B — Entry point

**Chosen: dedicated `REQUEST_SKILL_BOOK_USE` command type** on the existing `COMMAND_TOPIC_CONSUMABLE`, mirroring how `REQUEST_SCROLL` coexists with `REQUEST_ITEM_CONSUME` (`services/atlas-consumables/atlas.com/consumables/kafka/message/consumable/kafka.go:16-19`). The client uses a distinct opcode, and the downstream flow shares nothing with `ConsumeStandard`.

**Rejected: reuse `REQUEST_ITEM_CONSUME` + classification branch.** Books currently fall through `RequestItemConsume` to the `ConsumeBare` fallback (`consumable/processor.go:196-215`); branching there would entangle the reservation idiom with the saga idiom in one method and hide a distinct client verb inside a generic one.

### C — Result gating

**Chosen: one-time handler on `EVENT_TOPIC_SAGA_STATUS`** (registered before saga submission, matching the service's established `message.OneTimeConfig` reservation pattern, `consumable/processor.go:569-572`; new `kafka/once/saga` validator matching the transaction id). The closure captures the pending context (characterId, isMasteryBook, skillId, masterLevel, roll outcome), so no persistence is needed.

**Rejected: channel consumes `EVENT_TOPIC_SAGA_STATUS` directly.** The completed body (`StatusEventCompletedBody{SagaType, Results}`) doesn't carry the writer's inputs; consumables owns the domain result.

**Accepted limitation:** if the consumables pod dies while a saga is in flight, the saga still completes correctly (items/skills consistent) but no result packet is emitted, and the client's exclusive-request lock stays set until relog. The existing reservation flows share this failure mode; fixing it (persistent pending-request store) is out of scope.

## 5. Component Design

### 5.1 libs/atlas-packet — two new codecs

Both fnames are `CWvsContext::` ⇒ family `character` (owning-class table, `docs/packets/IMPLEMENTING_A_PACKET.md:97-103`). No existing codec overlaps (grep confirmed; genuine new packets, not renames).

- **`character/serverbound/use_skill_book.go`** — op `USE_SKILL_BOOK`, fname `CWvsContext::SendSkillLearnItemUseRequest`. Handle const `CharacterSkillBookUseHandle`. Body (IDB-verified **identical in all 10 versions**): `updateTime uint32`, `slot int16`, `itemId uint32`. **No version gate.** Immutable model + `Decode` mirroring `Encode` per the recipe.
- **`character/clientbound/skill_learn_item_result.go`** — op `SKILL_LEARN_ITEM_RESULT`, fname `CWvsContext::OnSkillLearnItemResult`. Writer const `CharacterSkillLearnItemResultWriter = "CharacterSkillLearnItemResult"`. Body (IDB-verified, §D-4): `[bOnExclRequest byte — v84+ only]`, `characterId uint32`, `isMasteryBook byte`, `skillId uint32`, `masterLevel uint32`, `canUse byte`, `success byte`. **Version gate: prepend the `bOnExclRequest` byte (value `1`) when `MajorVersion() >= 84`.** This is the real boundary (v83 = 15-byte body, v84+ = 16-byte body) — NOT `>= 87`, and NOT `> 83` treating v84 as v83; v84's clientbound genuinely diverges from v83 even though its serverbound is identical.

**IDB-verified opcode matrix** (raw wire opcodes; dispatch base-0 in every version):

| version | serverbound `USE_SKILL_BOOK` | clientbound `SKILL_LEARN_ITEM_RESULT` | clientbound body |
|---|---|---|---|
| gms_48  | 0x40 | 0x2B | 15-byte |
| gms_61  | 0x4B | 0x30 | 15-byte |
| gms_72  | 0x51 | 0x30 | 15-byte |
| gms_79  | 0x50 | 0x30 | 15-byte |
| gms_83  | 0x52 | 0x33 | 15-byte |
| gms_84  | 0x52 | 0x33 | **16-byte** |
| gms_87  | 0x55 | 0x33 | **16-byte** |
| gms_92  | 0x59 | 0x34 | **16-byte** |
| gms_95  | 0x58 | 0x32 | **16-byte** |
| jms_185 | 0x4A | 0x30 | **16-byte** |

Every cell was decompiled per version (send: `COutPacket(op)` ctor arg + `Encode4/Encode2/Encode4` body, gated `itemId/10000 ∈ {228,229}`; receive: `CWvsContext::OnPacket` switch case + `Decode` order). The audit CSV/support-docs are NOT the source — several cells were wrongly marked `n-a`/`0x000` there.

Tests: round-trip across `pt.Variants`; explicit golden-byte assertions for BOTH body shapes — a v83 15-byte golden and a v84 (or v95) 16-byte golden proving the `bOnExclRequest` gate; `// packet-audit:verify` markers per version (§8).

### 5.2 atlas-channel

- **Handler** `socket/handler/character_skill_book_use.go`: `CharacterSkillBookUseHandleFunc` — decode, debug-log, forward via a new `consumable.NewProcessor(l, ctx).RequestSkillBookUse(s.Field(), s.CharacterId(), slot, itemId)` (drop `updateTime` after logging; no existing command forwards it). Register in `produceHandlers()` (`main.go:878` area).
- **Producer**: new provider + `RequestSkillBookUseBody{Slot int16, ItemId uint32}` in the channel-side `kafka/message/consumable` mirror, keyed by characterId, same envelope as the existing commands (`services/atlas-channel/atlas.com/channel/kafka/message/consumable/kafka.go:20-28`).
- **Writer** `socket/writer/character_skill_learn_item_result.go`: thin `packet.Encode` body delegating to the new codec. Append `CharacterSkillLearnItemResultWriter` to `produceWriters()` (`main.go:660-699`). Opcode comes from tenant `WriterConfig`; the body has no mode byte, so no `WithResolvedCode` table is needed.
- **Consumer**: extend the existing consumable status consumer (`kafka/consumer/consumable/consumer.go`) with `handleSkillBookResultEvent`: tenant-filter, `IfPresentByCharacterId`; if `CanUse` → `_map.NewProcessor(l, ctx).ForSessionsInMap(s.Field(), announce)` (scroll pattern, `consumer.go:94-96`), else announce to the requester session only. Add the `SKILL_BOOK_RESULT` type + body to the channel-side message mirror.

### 5.3 atlas-consumables

- **Command**: `CommandRequestSkillBookUse = "REQUEST_SKILL_BOOK_USE"` + `RequestSkillBookUseBody{Slot int16, ItemId uint32}` in `kafka/message/consumable/kafka.go`; handler in `kafka/consumer/consumable/consumer.go` (type-guard demux, same as `handleRequestScroll` at `consumer.go:58-66`).
- **Data accessors**: add `MasterLevel()`, `ReqSkillLevel()`, `Skills()` getters to `data/consumable/model.go` (fields exist at `model.go:60-64,105`; only `SuccessRate()` is exposed today).
- **New skills client** package `skill/`: `requests.go` + `rest.go` + `model.go` copied from the established shape (`services/atlas-messages/atlas.com/messages/skill/requests.go:10-15`, resource `characters/%d/skills` under `RootUrl("SKILLS")`); model exposes `Id()`, `Level()`, `MasterLevel()`, `Expiration()`. Requires the `SKILLS` base-URL env var in the consumables deployment (verify `deploy/` overlays; add if absent).
- **Saga client**: `saga/producer.go` + minimal message mirror emitting to `COMMAND_TOPIC_SAGA`, per the `services/atlas-map-actions/atlas.com/map-actions/saga/producer.go:10-13` pattern; env var added to the deployment.
- **Processor** `RequestSkillBookUse(field, characterId, slot, itemId)` — validation order (every rejection emits `SKILL_BOOK_RESULT{canUse:0}` and warn-logs characterId/itemId/reason):
  1. classification ∈ {228, 229} via `item.GetClassification` (`libs/atlas-constants/item/constants.go:41-42`);
  2. character fetch — reject if `Hp() == 0`;
  3. consumable data fetch — reject if `Skills()` empty;
  4. slot integrity via `inventory.GetByCharacterId` — USE compartment asset at `slot` exists, `TemplateId == itemId`, `Quantity ≥ 1`;
  5. target skill = first of `Skills()` with `skillId/10000 == uint32(jobId)` — reject if none;
  6. skills fetch — for 229 require record with `Level ≥ 1`; for 228 absent record is allowed only when `ReqSkillLevel() == 0`;
  7. `Level ≥ ReqSkillLevel()` (absent record counts as level 0);
  8. `MasterLevel < book.MasterLevel()`.
  Then: roll (`rand.Int31n(100) < int32(SuccessRate())`, info-log outcome); build the saga (txId = `uuid.New()`, type `skill_book_use`): step 1 `destroy_asset_from_slot{CharacterId, InventoryType: USE, Slot, Quantity: 1, TemplateId: itemId}` (TemplateId is the §5.4 payload extension); step 2 only on a passing roll — `update_skill{Level: current, MasterLevel: book, Expiration: current}` when a record exists, else `create_skill{Level: 0, MasterLevel: book, Expiration: time.Time{}}` (conditional-step precedent: `services/atlas-messages/.../skill/commands.go:56-92`; multi-step builder precedent: `services/atlas-npc-conversations/.../conversation/operation_executor.go:1516-1524,1612-1620`). Register the one-time saga-status handler (new `kafka/once/saga/once.go` validator on txId), then submit.
  On `COMPLETED`: emit `SKILL_BOOK_RESULT{IsMasteryBook, SkillId, MasterLevel: book, CanUse: true, Success: <roll>}`. On `FAILED`: emit `{CanUse: false}` (book either untouched — destroy failed — or restored by compensation) and warn-log the failed step/reason from `StatusEventFailedBody`.
- **Event**: `EventTypeSkillBookResult = "SKILL_BOOK_RESULT"` + body `{IsMasteryBook bool, SkillId uint32, MasterLevel uint32, CanUse bool, Success bool}` on the existing `EVENT_TOPIC_CONSUMABLE_STATUS` (`kafka/message/consumable/kafka.go:59`), keyed by characterId; producer alongside `ScrollEventProvider` (`producer.go:28`).

### 5.4 libs/atlas-saga + atlas-saga-orchestrator

- **Payload extension**: add `TemplateId uint32 \`json:"templateId"\`` to `DestroyAssetFromSlotPayload` in `libs/atlas-saga/payloads.go:101-108` and the orchestrator's local mirror. Additive and backward-compatible (existing producers omit it → 0). It exists so the compensator can reconstruct the destroyed book; the destroy handler itself passes through unchanged (`handler.go:1036-1050`).
- **New saga type** `skill_book_use` in `libs/atlas-saga/model.go` type constants and the orchestrator mirror (`saga/model.go:86-87` area).
- **Compensation**: in `CompensateFailedStep` (`compensator.go:239-259`), route saga type `skill_book_use` to a new reverse-walk dispatcher (precedent: `compensateSelectGachaponReward` at `compensator.go:848-880`, `DispatchPetEvolutionRollbacks` at `:1112-1126`): when the failed step is `update_skill`/`create_skill` and a completed `destroy_asset_from_slot` step precedes it, re-award via the existing create-asset path using the payload's `TemplateId` + `Quantity` (slot position is not preserved — acceptable; the freed slot guarantees space), then mark the saga failed. A failed step 1 (destroy) needs no compensation — nothing happened.
- **Step completion** needs no new consumers: `DestroyAssetFromSlot → {AssetDeleted, AssetQuantityChanged}` and `UpdateSkill/CreateSkill → {SkillUpdated/SkillCreated}` acceptances already exist (`event_acceptance.go:100,126-127`).
- **Acceptance-correlation caveat** (recorded, no action): step completion matches TransactionId + event kind only, not skillId — fine here since each saga has at most one skill step.

### 5.5 atlas-skills

**No code change.** `RequestUpdateBody` already carries `SkillId/Level/MasterLevel/Expiration` (`kafka/message/skill/kafka.go:33-38`) and emits `UPDATED` for saga completion (`skill/processor.go:174`). The clobber semantics are handled caller-side (§2 D-1). `Create` fails on an existing record (`processor.go:108-134`) and `Update` fails on a missing one (`processor.go:156`) — the conditional step selection in §5.3 must therefore be correct, and a lost race (record created between read and saga execution) fails the step and compensates, which is safe.

### 5.6 atlas-configurations — seed templates + live patch

Add to **all ten** templates in `services/atlas-configurations/seed-data/templates/` (entry shapes per `template_gms_83_1.json:8-12,1156-1159`). Opcodes are IDB-verified (§5.1); collision-checked FREE in every template:

| Template | handler `CharacterSkillBookUseHandle` (validator `LoggedInValidator`) | writer `CharacterSkillLearnItemResult` |
|---|---|---|
| gms_48  | 0x40 | 0x2B |
| gms_61  | 0x4B | 0x30 |
| gms_72  | 0x51 | 0x30 |
| gms_79  | 0x50 | 0x30 |
| gms_83  | 0x52 | 0x33 |
| gms_84  | 0x52 | 0x33 |
| gms_87  | 0x55 | 0x33 |
| gms_92  | 0x59 | 0x34 |
| gms_95  | 0x58 | 0x32 |
| jms_185 | 0x4A | 0x30 |

**`gms_12` is excluded** — the feature is genuinely absent in that client (no `CWvsContext::SendSkillLearnItemUseRequest`, no result opcode). This is the only exclusion, and it is evidence-backed (not a deferral). Re-check collisions within every touched template (summons-task precedent, commit `a2207e7c7`).

Note: `gms_48/61/72/79` templates each already carry `CharacterItemUseHandle` and, mostly, `CharacterItemUseScrollHandle` — but the skill-book handler is a **dedicated** opcode independent of that family (no generic-item-use routing). `gms_92`'s template is minimal (only `CharacterCashItemUseHandle`); the dedicated skill-book handler still works standalone.

Every handler entry carries the validator — a validator-less entry is silently dropped (`libs/atlas-opcodes/producer.go:47-51`).

**Live rollout:** seed templates apply only at tenant creation; PATCH the live tenant configurations for existing tenants and restart atlas-channel (handlers/writers don't hot-reload) — the new-opcodes gotcha.

## 6. Error Handling Matrix

Every inbound request produces exactly one result packet (the client's exclusive-request lock clears only on receipt):

| Path | Book | Skill | Result event | Packet routing |
|---|---|---|---|---|
| Any FR-2 validation rejection | untouched | untouched | `canUse=0` immediately | requester only |
| Roll fails | consumed (1-step saga) | untouched | `canUse=1, success=0` on COMPLETED | map broadcast |
| Roll passes, saga completes | consumed | master level = book's | `canUse=1, success=1` on COMPLETED | map broadcast |
| Destroy step fails (dupe/race/moved item) | untouched | untouched | `canUse=0` on FAILED | requester only |
| Skill step fails after destroy | **restored by compensation** | untouched | `canUse=0` on FAILED | requester only |
| Consumables crash mid-saga | consistent (saga owns both steps) | consistent | none — client wedged until relog (accepted, §4-C) | — |

Duplicate packets (excl-lock bypassed): both validate, both submit sagas; the second saga's destroy fails on the emptied slot → `canUse=0`. No double-consume, no double-grant. (A 2-stack of the same book would legitimately allow two consumes; master level is idempotent.)

## 7. Testing

- **Codec tests** (libs/atlas-packet): round-trip + v83 golden bytes for both packets; `packet-audit:verify` markers per §8.
- **Consumables unit tests** (table-driven, Builder pattern, no test-helper files): every FR-2 rejection path; target-skill selection (job match, first-match, no-match); 228-create vs 229-learned gating; roll boundaries (`success=0` never, `success=100` always); saga construction shape for all three outcomes (fail-roll 1-step, update 2-step with current Level/Expiration carried, create 2-step with zero expiration); result-event emission on COMPLETED/FAILED.
- **Orchestrator tests**: `skill_book_use` compensation — forced skill-step failure re-awards `TemplateId`×1 and fails the saga; destroy-step failure compensates nothing.
- **Channel tests**: consumer routing (canUse split), writer body wiring.
- **Mocks**: update `data/consumable` and any touched interface mocks in the same commit.
- **Verification gates** (CLAUDE.md): `go test -race`, `go vet`, `go build` per changed module; `docker buildx bake` for atlas-channel, atlas-consumables, atlas-saga-orchestrator, atlas-configurations (and any module whose `go.mod` changed via libs edits — libs/atlas-saga and libs/atlas-packet touch many); `tools/redis-key-guard.sh`; live acceptance per PRD §10 on a v83 tenant.

## 8. Fixture Campaign (FR-6)

Per `docs/packets/audits/VERIFYING_A_PACKET.md`, one packet × version cell at a time (packet-verifier flow): decompile the client read/build order, byte-fixture with marker, pin evidence, regenerate the matrix — promoting STATUS.md rows 72 and 570 across all applicable version columns.

**All IDBs are loaded** (v48, v61, v72, v79, v83, v84, v87, v92, v95, jms — verified 2026-07-25), and the send/receive addresses for every cell are already derived (§5.1 matrix) and named in each IDB (the derivation pass renamed the real `Send…`/`On…Result` functions and neutralized the v72/v79 mis-ports). So the v1 "IDB-blocked" and "no audit-report address" obstacles are **resolved** — the fixture campaign can promote every cell:

- **20 cells** = 2 packets × 10 versions (only `gms_12` has no column for this feature).
- The audit CSV/support-docs disagree with the IDB reality for several cells (legacy serverbound `n-a`; `gms_48` clientbound `n-a`/delta hint `47/48` vs real `0x2B`). **Trust the decompiled dispatch chain; correct the audit artifacts as part of the campaign** and note the corrections.
- Two body shapes: assert the 15-byte fixture for v48/61/72/79/83 and the 16-byte (leading `bOnExclRequest`) fixture for v84/87/92/95/jms.
- If any fname genuinely fails to resolve in an IDB, stop and escalate — never substitute or fake (not expected; all are resolved).

## 9. Execution-Phase Dependencies & Risks

1. **WZ `success` sweep** (§2 D-3): enumerate local v83 WZ 0228/0229 for missing `success` nodes before go-live.
2. **`SKILLS` / `COMMAND_TOPIC_SAGA` env wiring** for atlas-consumables deployment overlays (base + envs); missing base-URL env vars fail at request time, not boot.
3. **libs/atlas-saga payload change** ripples: rebuild/bake every service importing it (additive field, no behavioral change expected; verify with the workspace-wide build).
4. **Clientbound version gate at v84** (§5.1/§D-4): the `bOnExclRequest` byte boundary is `MajorVersion() >= 84`, a documented `v84 ≠ v83` exception — golden tests must cover both body shapes so a regression to a `>= 87` (or `> 83`) gate is caught.
5. **Live PATCH now spans 10 lineages** (gms_48/61/72/79/83/84/87/92/95/jms): every existing tenant on those versions needs the handler+writer PATCH + channel restart. Full sweep, not a spot-check (new-opcodes gotcha).
6. **Audit-artifact corrections**: the fixture campaign must fix the wrong `n-a`/`0x000` cells in `docs/packets/audits/support/*.md` and the Ops CSVs; leaving them stale re-poisons future version passes.
7. **Result-packet loss on consumables crash** (accepted limitation, §4-C) — matches existing scroll-flow behavior.
