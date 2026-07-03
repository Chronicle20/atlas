# Task 131 — Random Reward Items: Design

Version: v1
Status: Approved for planning
Created: 2026-07-02
PRD: `docs/tasks/task-131-random-reward-items/prd.md`

---

## 1. Summary

Implement the end-to-end "use a reward box" flow: a new serverbound codec + atlas-channel handler for `LOTTERY_ITEM_USE_REQUEST`, a new `ConsumeReward` flow in atlas-consumables (validate → reserve → roll → grant → commit/cancel), per-entry `Effect`/`worldMsg`/`period` parsing in atlas-data, and clientbound presentation via the **already-implemented but unwired** `EffectLotteryUse` packets plus the existing status-message and world-message writers.

All six PRD open questions (§9) were resolved during design with primary evidence (IDA on v83/v95, local WZ sweep, code recon). They are documented in §2 — the rest of the design builds on them.

## 2. Resolved open questions (design-phase verification results)

### 2.1 Serverbound body — IDA-verified: `slot int16, itemId int32`, NO updateTime

Decompiled `CWvsContext::SendLotteryItemUseRequest` on both live IDBs:

- **v83** (`MapleStory_dump.exe`, fn `0xa1249f`): `COutPacket(0x70)` → `Encode2(slot)` → `Encode4(itemId)`. Guarded by `CanSendExclRequest(this, 200, 0)`; sets `m_bExclRequestSent = 1` after send.
- **v95** (`GMS_v95.0_U_DEVM.exe`, fn `0x9d6c50`): `COutPacket(124 = 0x7C)` → `Encode2(nPos)` → `Encode4(nItemID)`. Same excl-request guard/set.

Unlike sibling item-use packets (`InventoryItemUse` reads `updateTime, slot, itemId`), **there is no leading updateTime** — Cosmic's read order (`short slot, int itemId`) is confirmed correct.

Client-side routing confirmed (v95 `CDraggableItem::OnDoubleClicked`, `0x506e10`): for Consume-tab items the lottery check runs **first** — `CItemInfo::GetLotteryItem(itemId) != null → SendLotteryItemUseRequest(slot, itemId)` — before `is_state_change_item` and every other consume dispatch. The client populates its lottery registry from every Item.wz Consume/Install/Etc item bearing a `reward` node (`CItemInfo::IterateLotteryItem`/`RegisterLotteryItem`). So the server only receives this opcode for reward-node items; anything else on this opcode is a hacked client.

Per-version opcodes (registry `docs/packets/MapleStory Ops - ServerBound.csv:256`, columns v12/v83/v87/v92/v95/v111/jms; STATUS.md:606 agrees):

| version | opcode | evidence |
|---|---|---|
| v83 | 0x070 | IDA-verified this task (fn 0xa1249f) |
| v84 | 0x070 | registry lineage (STATUS.md; task-100 reshifted table). No v84 IDB — no-IDB convention |
| v87 | 0x073 | registry + CSV (no v87 instance loaded; CSV row and STATUS.md agree) |
| v92 | 0x07B | CSV (v92 has no IDB — template-lineage convention, flag in evidence) |
| v95 | 0x07C | IDA-verified this task (fn 0x9d6c50) |
| jms | 0x06B | registry — **out of scope** (§2.6) |

### 2.2 `Effect` delivery — server-sent `CUser::OnEffect` LOTTERY_USE arm; codecs already exist

The client **never reads** the reward entry's `sEffect`/`sWorldMsg`/`nPeriod` at runtime. Proof (v95): `CItemInfo::LOTTERY_ENTITY` = `{nItemID, nProb, nQuantity, sEffect, sWorldMsg, nPeriod, sDateExpire}`; the only reader of `CItemInfo.m_mLotteryItem` is `GetLotteryItem` (sole xref to the map's `ZMap::GetAt` instantiation `0x500780`), and its only two callers (`OnDoubleClicked`, `UseFuncKeyMapped`) just null-check the pointer to route the use request. So effect playback is entirely server-driven.

The mechanism is the `LOTTERY_USE` arm of `CUser::OnEffect`, IDA-verified on both IDBs with identical semantics:

- **v95 case 16** (`0x8f9a70`): `Decode4 → itemId`, `play_item_sound(itemId, SE_ITEM_USE)`; `Decode1 → success`, if 0 **return**; `DecodeStr → path`, passed to `CAnimationDisplayer::Effect_General(path, …)` at the user's position.
- **v83 case 14** (`0x9377d9`): byte-identical structure (`play_item_sound(id, 41)` → success gate → `DecodeStr` → `Effect_General`).

So the packet's string field is the **effect WZ path** (the reward entry's `Effect`, e.g. `Effect/BasicEff/Event1/Good`), not a chat message, and the int is the **box item id** (drives the use sound).

Everything needed already exists in the codebase, unwired:
- Codecs `EffectLotteryUse` / `EffectLotteryUseForeign` (`libs/atlas-packet/character/clientbound/effect.go:617,662`) with round-trip tests — encode `mode, itemId, success, [message-if-success]`, matching the verified read order.
- Body factories `CharacterLotteryUseEffectBody` / `...ForeignBody` (`libs/atlas-packet/character/effect_body.go:233-242`), mode resolved via `WithResolvedCode("operations", "LOTTERY_USE")`.
- `LOTTERY_USE` operations-table entries present in **all** seed templates (v83/v84/v87: 14, v95: 16, jms: 15 — task-112 backfill), and the v83 (14) / v95 (16) values match the IDA switch labels verified above.

Design decision: on a successful roll where the entry has a non-empty `Effect`, send `EffectLotteryUse(boxItemId, success=true, path=entry.Effect)` to the user and `EffectLotteryUseForeign(...)` to other sessions in the map (pattern: `socket/handler/effects.go` `AnnounceForeignSkillUse`). When `Effect` is empty (16 of the 56 v83 boxes have none), send nothing — the client renders nothing useful from an empty path, and `success=false` would merely play a sound and return.

### 2.3 `period` unit — MINUTES (evidence-based; Cosmic's math is the bug its author suspected)

- The client never reads `nPeriod` (§2.2), so the unit is a server-side convention; the reference server is the only interpreter.
- Cosmic computes `period * 60 * 60 * 10` ms (= 36 s per unit) with the comment `// TODO is this a bug, meant to be 60 * 60 * 1000?` (`ItemRewardHandler.java:66-67`) — not usable as authority.
- Decisive cross-reference in the same WZ generation: **quest reward items carry the same per-grant `period` key**, Cosmic interprets it as minutes (`ItemAction.java:159`: `MINUTES.toMillis(period)`), and the data only makes sense that way: quest trial weapons `period=10080` = exactly 7 days.
- Observed reward values under minutes: `7200` = exactly 5 days (event belt 1132010, all 23 golden-pig boxes), Cash-box reward `21600` = exactly 15 days. Under Cosmic-as-written they'd be 3/9 days (non-round), under hours 300/900 days (absurd).

**Decision: `expiration = now + period minutes` when `period > 0`; `period <= 0` (default −1) = no expiration.** The grant command's `Expiration time.Time` field (`CreateAssetCommandBody`, atlas-inventory `kafka.go:100-108`) carries the absolute timestamp; the existing `atlas-asset-expiration` service handles the rest. No new expiration machinery.

### 2.4 Prob-sum sweep — pure weights confirmed; clean weighted pick is safe

Swept all of `Item.wz/Consume` in the local v83 WZ dump (`~/source/Cosmic/wz`): exactly **56 boxes** with `reward` nodes (matching the PRD count). Prob sums range 20 → 19,864 with no common denominator (20, 60, 100×18, 454, 554, 609, 789×2, 1000×2, ~9.9k–19.9k×33). There is no "sum < fixed denominator = authored chance of nothing" table — the data is weights, not percentages. Supporting signal: the authored data encodes *failure as a booby-prize entry* (Effect strings `Event1Failure`, `FindPrize/Failure` on entries that still grant a junk item), i.e. the data itself expects every use to yield exactly one reward. The owner-decided single weighted pick changes no authored economics.

Field inventory from the sweep (drives §6): per-entry keys are `item`, `count`, `prob`, plus optional `Effect` (capital E — 1,440 entries), `worldMsg` (230 entries, all on the 23 golden-pig boxes, all using `/name` and `/item` tokens), `period` (23 entries, all `7200`, all on equip 1132010).

### 2.5 Grant path — direct compartment `CREATE_ASSET` + reservation compensation (no saga)

See §5 for the decision and alternatives.

### 2.6 jms — out of scope (default confirmed)

No jms IDB is loaded (current instance set: v48/v61/v72/v79/v83/v95 — verified via `list_instances` this session), so the "cheap parity check" gate in the PRD fails. The registry opcode (`0x06B`) and the jms template's `LOTTERY_USE: 15` operations entry suggest the same body applies, but that is unverified. jms is excluded from v1; no handler entry is added to `template_jms_185_1.json`.

## 3. Architecture overview

```
client double-clicks reward box
  └─ CWvsContext::SendLotteryItemUseRequest [slot int16, itemId int32]   (client locks: excl request)
       └─ atlas-channel: LotteryItemUseHandle (new)
            └─ Kafka: COMMAND_TOPIC_CONSUMABLE / REQUEST_ITEM_REWARD (new type)
                 └─ atlas-consumables: RequestItemReward
                      1. fetch consumable data; validate reward table (non-empty, Σprob > 0)
                      2. reserve box slot (existing compartment reservation machinery)
                      3. [once-handler on RESERVED] ConsumeReward:
                         a. weighted roll (crypto/rand) over reward entries
                         b. emit CREATE_ASSET for rolled item (+expiration if period>0)
                         c. [once-handler on compartment status]
                            CREATED          → commit: ConsumeItem(box) + presentation events
                            CREATION_FAILED  → cancel reservation + INVENTORY_FULL error event
                      any validation/infra error → ConsumeError (cancel reservation + ERROR event)
                 presentation / feedback (atlas-channel consumers):
                   success: EVENT reward effect  → EffectLotteryUse (self) + Foreign (map)
                            EVENT reward won     → WorldMessageBlueText fan-out (if worldMsg)
                            inventory ops        → existing inventory event flow (unsticks client)
                   inventory full: ERROR/INVENTORY_FULL → StatusMessageDropPickUpInventoryFull
                                                          + StatChanged(enableActions)  (box kept)
                   other errors:   ERROR (existing)     → StatChanged(enableActions)
```

The client is excl-locked from the moment it sends the request (§2.1), so **every** terminal path must emit something that unsticks it: success does so via the inventory-operation events from consume+grant (exactly as `ConsumeStandard` does today), all failure paths via the existing ERROR arm's `StatChanged([], enableActions=true)` (`atlas-channel kafka/consumer/consumable/consumer.go:77`).

## 4. Alternatives considered

### 4.1 Grant path (PRD open question 5)

**A. Direct `CREATE_ASSET` from atlas-consumables + reservation compensation — CHOSEN.**
atlas-consumables already owns the reserve/commit/cancel lifecycle (`ConsumeScroll`, `consumable/processor.go:606-738`) and the one-time-Kafka-handler correlation machinery (`once.ReservationValidator`). Adding a `RequestCreateItem` producer to its `compartment` processor (mirroring saga-orchestrator's `compartment/processor.go:56-66`) plus a second once-validator for `CREATED`/`CREATION_FAILED` (both carry `transactionId`) keeps the whole flow in one service, one transaction id, no new inter-service choreography. Grant-before-commit ordering gives the PRD's "inventory-full detected before the box is consumed" for free — and without a TOCTOU window, because the fit check *is* the grant attempt.

**B. Saga (gachapon-style `AwardAsset` steps in atlas-saga-orchestrator) — rejected.**
Precedent exists (`gachapon/` handlers, dynamic step injection, re-award compensation), but: atlas-consumables initiates zero sagas today (new dependency direction); the roll stays in consumables either way (owner decision), so the saga would only wrap grant+consume; saga timeout handling has a history of foot-guns for data-driven flows (task-086); and compensation semantics (re-award on failure) are not better than reservation-cancel for this shape. More moving parts, no atomicity gain over A.

**C. Pre-check capacity via REST, then grant — rejected.**
There is no "will it fit" endpoint (recon: atlas-inventory only reports fullness by failing `CREATE_ASSET` with `CREATE_ASSET_INVENTORY_FULL`); a fetched-capacity check in consumables would race with concurrent inventory mutations. A's attempt-then-react is both simpler and race-free.

**Atomicity achieved (PRD 4.7):** identical class to the scroll flow. The reservation pins the box between reserve and commit; grant failure cancels the reservation (box untouched); the box is consumed only after `CREATED` is observed. The residual window (grant CREATED, then commit command lost to infra failure) is the same exposure `ConsumeScroll` accepts today (`ChangeStat` then `ConsumeItem`, errors logged) — documented, not new.

### 4.2 worldMsg transport

**Chosen: new broadcast event on `EVENT_TOPIC_CONSUMABLE_STATUS`** (`REWARD_WON`, body: characterId, boxItemId, rolled itemId, pre-substituted message), consumed by atlas-channel with `session.AllInChannelProvider` fan-out — the exact pattern of the gachapon win broadcast (`consumer/gachapon/consumer.go:51-88`), which is how "world-wide" is achieved: every channel pod consumes and announces to its own sessions.
- *Rejected:* per-character `COMMAND_TOPIC_SYSTEM_MESSAGE`/`SEND_MESSAGE` — targeted at one characterId (`handleSendMessage` → `IfPresentByCharacterId`), unsuitable for broadcast without enumerating characters.
- *Rejected:* reusing `EVENT_TOPIC_GACHAPON_REWARD_WON` — wrong domain; its body/rendering (gachapon megaphone with channel + name) doesn't match the reference presentation.

Rendering: Cosmic uses server-notice type 6 (light blue). Atlas equivalent: `WorldMessageBlueTextBody` (`socket/writer/world_message.go:95`, mode resolved from the per-version operations table; codec `WorldMessageBlueText` mode 6 writes `mode, message, itemId`). The rolled itemId rides in the packet's int field.

Substitution happens **in atlas-consumables** (single place, once per win, avoiding Cosmic's `replaceAll` no-op bug): `/name` → character name (consumables' existing character processor), `/item` → `GET {DATA}/data/item-strings/{itemId}` `.name` (endpoint exists: `atlas-data item/string_resource.go:29`). Lookup failure → warn-log and skip the announcement (presentation-only; never blocks the grant).

### 4.3 Inventory-full feedback

New error type `INVENTORY_FULL` on the **existing** consumable ERROR event. The channel's `handleErrorConsumableEvent` gains an arm: announce `StatusMessageDropPickUpInventoryFull` (`libs/atlas-packet/character/clientbound/status_message.go:54-74` — mode + int8(−1), the Atlas form of Cosmic's `getShowInventoryStatus` 0xFF) **and** the existing `StatChanged([], enableActions=true)` (Cosmic sends the analogous pair: `getInventoryFull()` + `getShowInventoryStatus()`). Reuses the verified status-message family; no new writer.

## 5. Component design

### 5.1 `libs/atlas-packet` — serverbound codec

New `inventory/serverbound/lottery_item_use.go`: struct `LotteryItemUse{source int16, itemId uint32}`, doc marker `// packet-audit:fname CWvsContext::SendLotteryItemUseRequest`, registration name `CharacterItemUseLotteryHandle` — shaped exactly like `item_use.go` minus the updateTime field. Decode order: `ReadInt16` (slot), `ReadUint32` (itemId). No version branching — the body is invariant across v83–v95 (§2.1).

Fixture test `lottery_item_use_test.go` per `item_use_test.go` precedent (`pt.Variants` round-trip) with `packet-audit:verify` markers:
- `version=gms_v83 ida=0xa1249f`, `version=gms_v95 ida=0x9d6c50` (IDA-verified this session);
- v84/v87/v92 markers/evidence per the no-IDB / registry-lineage convention used by sibling tasks (packet-verifier playbook, `docs/packets/audits/VERIFYING_A_PACKET.md`), promoting the STATUS.md row cells accordingly.

### 5.2 atlas-channel — handler + command emit

- `socket/handler/character_item_use.go`: add `CharacterItemUseLotteryHandleFunc` — decode, then `consumable.NewProcessor(l, ctx).RequestItemReward(s.Field(), character.Id(s.CharacterId()), item.Id(p.ItemId()), slot.Position(p.Source()))`.
- `consumable/processor.go` + `producer.go`: `RequestItemReward` emitting new command type on `COMMAND_TOPIC_CONSUMABLE`.
- `kafka/message/consumable/kafka.go`: `CommandRequestItemReward = "REQUEST_ITEM_REWARD"`, body `{Source slot.Position, ItemId item.Id}` (no updateTime — none exists on the wire).
- `main.go`: `handlerMap[invsb.CharacterItemUseLotteryHandle] = handler.CharacterItemUseLotteryHandleFunc`.

### 5.3 Kafka contracts (mirrored channel ↔ consumables)

| message | topic | direction | body |
|---|---|---|---|
| `REQUEST_ITEM_REWARD` (new command type) | `COMMAND_TOPIC_CONSUMABLE` | channel → consumables | source slot, itemId |
| `ERROR` w/ new `ErrorTypeInventoryFull = "INVENTORY_FULL"` | `EVENT_TOPIC_CONSUMABLE_STATUS` | consumables → channel | existing `ErrorBody` |
| `REWARD_EFFECT` (new event type) | `EVENT_TOPIC_CONSUMABLE_STATUS` | consumables → channel | characterId, boxItemId, effect path |
| `REWARD_WON` (new event type) | `EVENT_TOPIC_CONSUMABLE_STATUS` | consumables → channel | characterId, boxItemId, itemId, message (pre-substituted) |

`REWARD_EFFECT` is separate from `REWARD_WON` because most rolls have an Effect and no worldMsg (1,440 vs 230 entries); the two presentations are independent.

### 5.4 atlas-consumables — `RequestItemReward` + `ConsumeReward`

New entry point `RequestItemReward(f field.Model, characterId uint32, itemId item.Id, source slot.Position)` (sibling of `RequestScroll`, `processor.go:515`):
1. Fetch consumable data (`p.cdp.GetById`). Validate: rewards non-empty **and** `Σprob > 0` — else warn-log and emit ERROR (client unstick) *without* reserving (nothing to cancel yet; a reward request for a non-reward item only occurs from a tampered client).
2. Reserve the box: `cpp.RequestReserve(transactionId, characterId, TypeValueUse, [{Slot: source, ItemId: itemId, Quantity: 1}])`, one-time handler `once.ReservationValidator(transactionId, itemId)` → `ConsumeReward`.

`ConsumeReward(transactionId, characterId, slot, itemId, rewards) ItemConsumer`:
1. **Roll** — `rollReward(rewards []RewardModel) (RewardModel, error)`: pure function; `crypto/rand` `rand.Int(rand.Reader, big.NewInt(total))`, cumulative-threshold walk (algorithm mirror of `atlas-gachapons reward/processor.go:121 selectTier`). Skips zero-prob entries naturally; error on `total == 0` (defense in depth — validated earlier).
2. **Grant** — new `compartment.RequestCreateItem(transactionId, characterId, templateId, quantity, expiration)` producer (mirror of saga-orchestrator's; `CREATE_ASSET` command). `quantity = max(count, 1)`; `expiration = now + period minutes` if `period > 0`, else zero time. Register one-time handler on `EVENT_TOPIC_COMPARTMENT_STATUS` correlated by transactionId for `CREATED` / `CREATION_FAILED`.
3. **On `CREATED`** — commit: `cpp.ConsumeItem(characterId, TypeValueUse, transactionId, slot)`; then emit `REWARD_EFFECT` (if entry.Effect ≠ "") and `REWARD_WON` (if entry.worldMsg ≠ "", after substitution §4.2). Debug-log the roll: characterId, boxId, rolled itemId, count, prob, total.
4. **On `CREATION_FAILED`** — `cpp.CancelItemReservation(...)`; error code `CREATE_ASSET_INVENTORY_FULL` → emit ERROR/`INVENTORY_FULL`; other codes → ERROR (generic). Box preserved in both cases.
5. Any pre-grant error → existing `ConsumeError` (cancel + ERROR).

Model extension (`data/consumable/model.go` + `rest.go`): `RewardModel` gains `effect string`, `worldMsg string`, `period int32` with getters (`ItemId()`, `Count()`, `Prob()` getters must be added too — the current struct has none); `RewardRestModel` gains the three JSON fields; `ExtractReward` maps them. Immutable-model + getter pattern per DOM guidelines; check `libs/atlas-constants` before introducing any new type (expiration stays `time.Time`, no new constants expected).

### 5.5 atlas-channel — new consumer arms

In `kafka/consumer/consumable/consumer.go`:
- `handleErrorConsumableEvent`: add `ErrorTypeInventoryFull` arm → `IfPresentByCharacterId` announce `CharacterStatusMessageWriter`(`StatusMessageDropPickUpInventoryFull`) **then** `StatChangedWriter`(`NewStatChanged([], true)`).
- `handleRewardEffectConsumableEvent` (new, gated on `REWARD_EFFECT`): self → `CharacterEffectWriter` + `CharacterLotteryUseEffectBody(boxItemId, true, path)`; observers → `_map.ForSessionsInMap` minus self with `CharacterEffectForeignWriter` + `CharacterLotteryUseEffectForeignBody(characterId, boxItemId, true, path)` (pattern: `effects.go:19,31`).
- `handleRewardWonConsumableEvent` (new, gated on `REWARD_WON`): fan-out via `session.AllInChannelProvider(worldId, channelId)` announcing `WorldMessageWriter` + `WorldMessageBlueTextBody(message, itemId)` (gachapon-consumer pattern).

No new writers, no new operations-table entries — `LOTTERY_USE` and the world-message/status-message modes already exist in every supported template (§2.2; task-112).

### 5.6 atlas-data — reward-node field parsing

`consumable/reader.go:164-172`: parse per-entry `Effect` (string, **capital E** — verified casing in WZ), `worldMsg` (string), `period` (int, default −1) off each reward child node; switch the positional `RewardRestModel{...}` literal to keyed. `rest.go:125`: add `Effect string \"effect\"`, `WorldMsg string \"worldMsg\"`, `Period int32 \"period\"` JSON fields. These are per-entry fields, distinct from the item-level `effect`/`worldMsg` at `reader.go:80-82`; keep the item-level parse untouched.

### 5.7 Data rollout (existing tenants)

atlas-data consumables are stored JSON documents; existing rows lack the new fields until re-processed. Absent fields degrade gracefully (`""`/`-1` → no effect, no announce, no expiration — never a crash). Rollout, documented in the task and performed if a canonical baseline is in play:
1. Re-ingest the canonical tenant (`POST /api/data/process`).
2. `POST /api/data/baseline/publish` (new canonical snapshot to MinIO).
3. Per live tenant: `POST /api/data/baseline/restore` (or per-tenant `/api/data/process`). Tenants relying on the canonical-fallback read path (storage.go:44-60) pick the fields up from step 2 alone.

### 5.8 Config rollout

Seed templates (`services/atlas-configurations/seed-data/templates/`): add to `socket.handlers` in `template_gms_83_1.json`, `_84_`, `_87_`, `_92_`, `_95_` (NOT jms, §2.6):

```json
{ "opCode": "<per-version, §2.1>", "validator": "LoggedInValidator", "handler": "CharacterItemUseLotteryHandle" }
```

`validator` is **mandatory** — verified this session that `BuildHandlerMap` (`libs/atlas-opcodes/producer.go:47-50`) silently `continue`s when the validator key doesn't resolve, and only `"NoOpValidator"`/`"LoggedInValidator"` are registered (channel `main.go:904-909`); an omitted field means a dead handler.

Live tenants: seed templates apply only at creation — document the config PATCH adding the same handler entry per tenant + atlas-channel restart (projection does not hot-reload handlers), per the established convention (project memory: new-opcodes-not-in-live-tenant-config).

## 6. Error handling matrix

| failure | when | action | client outcome |
|---|---|---|---|
| item has no/empty reward table, or Σprob = 0 | pre-reserve | warn log + ERROR event | unstuck (StatChanged), box untouched |
| character doesn't own box at slot | reservation | inventory rejects reserve; no RESERVED event → once-handler never fires; ERROR path on reserve failure event per existing machinery | unstuck, box untouched |
| roll infra error (crypto/rand) | post-reserve | `ConsumeError` → cancel + ERROR | unstuck, box untouched |
| inventory full (`CREATE_ASSET_INVENTORY_FULL`) | grant | cancel reservation + ERROR/`INVENTORY_FULL` | "inventory full" status message + unstuck, **box kept** |
| other `CREATION_FAILED` codes | grant | cancel reservation + ERROR | unstuck, box untouched |
| consume-commit emit fails after CREATED | commit | error log (same exposure as `ConsumeScroll`; reservation still pins the box) | reward granted; box release requires ops intervention — accepted, pre-existing class |
| worldMsg/effect lookup or emit fails | presentation | warn log, skip | grant + consume unaffected |

## 7. Non-functional requirements

- **Multi-tenancy:** tenant headers on all messages (existing `Command`/`Event` envelopes); consumable data reads tenant-scoped; zero hardcoded version branches — opcodes and effect modes come from tenant config tables exclusively.
- **Randomness:** `crypto/rand` only (§5.4.1); no `math/rand` anywhere in the arm.
- **Observability:** debug-log per roll (characterId, boxId, rolled itemId, prob/total); warn-log every validation rejection and presentation-path failure. No new metrics.
- **Fail-safety:** every terminal path emits an unsticking event (§3); no silently swallowed request.

## 8. Testing strategy

- **Unit (consumables):** `rollReward` — distribution over a weighted table (statistical bounds on large N), single entry, zero-prob entries excluded, total=0 error; `period → expiration` conversion (7200 → +5d, −1/0 → zero time); worldMsg substitution (`/name`, `/item`, both, multiples — regression against Cosmic's no-op bug); count=0 → quantity 1. Builder-based setup per existing `processor_test.go` patterns; no `*_testhelpers.go`.
- **Unit (atlas-data):** reader parses `Effect`/`worldMsg`/`period` per entry (capitalized-`Effect` casing) and defaults (missing → `""`/`""`/−1); rest round-trip.
- **Codec fixtures:** `lottery_item_use_test.go` round-trip across `pt.Variants` + verify markers (§5.1); evidence records + `packet-audit` matrix regeneration promoting the STATUS.md row (v92/v84/v87 cells per no-IDB convention).
- **Consumer-arm tests:** channel consumable-consumer arms for `INVENTORY_FULL`, `REWARD_EFFECT`, `REWARD_WON` following existing consumer test patterns where present.
- **Verification gates:** `go test -race ./...`, `go vet ./...`, `go build ./...` in every changed module; `docker buildx bake` for atlas-channel, atlas-consumables, atlas-data, atlas-configurations (if templates ship in its image) + any lib-touched service set; `tools/redis-key-guard.sh`.

## 9. Scope

**In:** v83, v84, v87, v92, v95 GMS tenants; the 56 v83 Consume reward boxes (and any reward-node Consume item in other versions' data — the flow is data-driven).
**Out:** jms (§2.6); gachapon machines; monster-catch / Solomon items; scripted items; Cash-tab reward boxes (e.g. Cash/0553 — routed by the client through `SendCashSlotItemUseRequest`/cash flows, not this opcode; noted for a future task); atlas-ui.

## 10. Affected services (delta from PRD §7)

Unchanged from PRD except: **atlas-saga-orchestrator is NOT touched** (§4.1 chose the direct path), and **atlas-configurations** is added (seed-template handler entries). atlas-inventory: no code change (existing `CREATE_ASSET` semantics suffice, including `CREATE_ASSET_INVENTORY_FULL`).

## 11. Design-phase evidence log

| claim | source |
|---|---|
| v83 body/opcode | IDA v83 dump port 13342, `SendLotteryItemUseRequest` @ 0xa1249f |
| v95 body/opcode | IDA v95 port 13341, @ 0x9d6c50 |
| client routes reward items to this opcode, lottery check first | v95 `CDraggableItem::OnDoubleClicked` @ 0x506e10 |
| client never reads sEffect/sWorldMsg/nPeriod | v95 `LOTTERY_ENTITY` type + sole-reader call graph (`GetLotteryItem` ← 2 use-trigger callers; `ZMap::GetAt` instantiation 0x500780 single xref) |
| OnEffect LOTTERY_USE semantics (itemId sound, success gate, path → Effect_General) | v95 OnEffect @ 0x8f9a70 case 16; v83 OnEffect @ 0x9377d9 case 14 |
| per-version opcodes v87/v92 | `docs/packets/MapleStory Ops - ServerBound.csv:256` + STATUS.md:606 |
| 56 boxes, prob sums, field inventory | WZ sweep script over `~/source/Cosmic/wz/Item.wz/Consume` (scratchpad `reward_sweep.py`/`reward_detail.py`) |
| period = minutes | Quest.wz Act.img `period=10080` (=7d) + Cosmic `ItemAction.java:159` `MINUTES.toMillis`; reward values 7200/21600 = exact 5/15 days |
| LOTTERY_USE operations entries all templates | grep seed templates (v83/84/87:14, v95:16, jms:15) — v83/v95 values match IDA switch labels |
| validator-less handler entries are dropped | `libs/atlas-opcodes/producer.go:44-51` + channel `main.go:904-909` |
| unstick path | channel `kafka/consumer/consumable/consumer.go:77` (`StatChanged([], true)`); excl-lock set confirmed in both IDBs |
