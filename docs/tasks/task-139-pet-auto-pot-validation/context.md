# task-139 — Context

Companion to `plan.md`. Key files, decisions, dependencies, and rollout notes for the pet auto-pot validation + pet skill pouch implementation.

## Key files by service

### libs/atlas-constants
- `item/constants.go` — cash classification block (517 → **519 new** → 520). `GetClassification` derives from item id, no other wiring needed.
- `pet/skill/constants.go` (new) — nine semantic keys (0519 WZ spelling), canonical `Flag` bits `1<<0..1<<8`, `All/BitFor/Has/Apply`.

### libs/atlas-packet
- `resolve.go` — `ResolveCode` (byte, loud 99 default) + **new `ResolveCode16`** (uint16, soft miss). usPetSkill bits reach 0x100, which a byte cannot hold.
- `model/asset.go:296-318` — `encodePetCashItemInfo`; the `w.WriteShort(0) // skill` at ~line 313 becomes the config-resolved wire mask. New additive `SetPetFlag`/`PetFlag` (SetPetInfo signature unchanged — avoids cross-module breakage).
- `cash/serverbound/item_use.go` — common cash-item-use prefix (`updateTimeFirst = GMS && >=95`); `item_use_pet_consumable.go` is the shape template for the new `item_use_pet_skill.go` (type-28, jms-only sender).
- `cash/serverbound/item_use_pet_skill.go` — **Task 8 IDA result deviated from the plan's assumed template.** The plan (Task 8 Step 1) assumed the same `petId u64` + trailing `updateTime`, `updateTimeFirst`-gated shape as `ItemUsePetConsumable`. The verified jms wire layout is a **bare `petId uint64` with no `updateTime` field at all**: `NewItemUsePetSkill()` takes zero arguments (not `updateTimeFirst bool`), and there is no `UpdateTime()` accessor — only `PetId()`. Consequently Task 9's `character_cash_item_use.go` type-28 arm has **no `updateTimeFirst` branch**; the `updateTime` already decoded from the common `ItemUse` header is passed through to `RequestItemConsumeWithPet` unconditionally, unlike the sibling pet-consumable arm.
- `pet/serverbound/item_use.go` — PET_AUTO_POT packet, already byte-fixtured for all five versions. `buffSkill` is misnamed: it is damage-mitigation context (0/1=PowerGuard/2=MesoGuard/4/8), log-only.

### atlas-channel
- `socket/handler/pet_item_use.go` — the handler being validated. Currently forwards blindly.
- `socket/handler/character_skill_use.go:131-137` — `enableActions` (moves to shared `enable_actions.go`); `:51-53` the alive-check precedent.
- `socket/handler/character_cash_item_use.go` — slot↔item validation already done at `:37-41`; `category == 519 → 28` literal at `:297-299`; new type-28 arm decodes petId.
- `pet/processor.go` — `GetById(uint32)`; `pet/model.go` — `OwnerId()/Slot()/Flag()` all present; `Flag` already hydrated from REST (`pet/rest.go:28`).
- `character/processor.go` — `GetById(decorators...)`; `PetAssetEnrichmentDecorator` (gains `SetPetFlag`); `GetItemInSlot` shows the compartment fetch pattern (`p.cp.GetByType`).
- `character/model.go:284` `SetInventory` — **worn cash equips are stored at `position − 100`** in the raw equip compartment; pet equips are cash items, so the gate normalizes (`-124 → -24`).
- `kafka/consumer/pet/consumer.go:228-255` — `announcePetStatUpdate` (requires `PetAssetEnrichmentDecorator`; re-encodes the pet cash asset via `InventoryChangeWriter`). `handleClosenessChanged` at `:257` is the model for `handleFlagChanged`.
- `kafka/consumer/consumable/consumer.go:57-81` — downstream ERROR-event unstick; fires only for forwarded requests, so handler-side rejection cannot double-unstick.
- `data/` — had only cash/map/npc/portal/quest/skill clients; **`data/consumable` and `data/equipment` are new** (Task 10). JSON:API types: `consumables`, `statistics`; URLs `data/consumables/%d`, `data/equipment/%d`.

### atlas-pets
- `pet/entity.go:30` — `Flag uint16` column **already exists and persists** (`not null;default:0`); the PRD's "no column" claim was wrong. Gaps were: no updater, no command/event, no channel consumer, not encoded to clients.
- `pet/administrator.go` — update-fn pattern (`updateSlot` at `:38`); `pet/producer.go:113-126` — `fullnessChangedEventProvider` event pattern.
- `pet/processor.go:1025-1031` — **`SetSkillAndEmit` deviates from the plan's snippet.** The plan (Task 6) wrote `SetSkillAndEmit` as a direct `message.Emit(p.kp)(...)` call with a `database.ExecuteTransaction` only inside the per-skill loop. The landed code wraps the whole operation in `database.ExecuteTransaction(p.db.WithContext(p.ctx), ...)` and emits via `message.Emit(outbox.EmitProvider(p.l, p.ctx, tx))(...)` — the codebase's actual outbox-pattern idiom (transactional outbox row written in the same DB transaction as the flag update), not the bare `p.kp` producer the plan's pseudocode showed. `message.Emit(p.kp)` as written in plan.md does not exist in this codebase.
- `pet/processor.go:209-211` — `Spawned(m) = m.Slot() >= 0` (canonical spawned semantics).
- `pet/processor_test.go` — sqlite in-memory harness (`testDatabase/testContext/mustBuild`).

### atlas-consumables
- `consumable/processor.go` — `RequestItemConsume` branch dispatch at `:184-226`; `ConsumeCashPetFood` at `:438-460` is the closest sibling (note: its Use/Cash inventory-type mix-up is pre-existing; do **not** copy it — use `TypeValueCash` consistently).
- `kafka/message/pet/kafka.go` — Command mirror had `PetId uint64` (wrong-typed vs atlas-pets `uint32`); aligned in Task 7.
- `pet/processor.go` — REST client with `Spawned()` filter and `GetById(uint64)`; `pet/rest.go` — REST `Id` is uint64-typed but values are pet ids (uint32 range).
- `cash/rest.go`/`cash/model.go` — client mirror of atlas-data cash items; gains `petSkills`/`petSkillAdd`.

### atlas-data
- `cash/reader.go:99-124` — spec parsing; the 0519 skill keys + `add` live under **`info`** (verified in WZ), parsed with `GetBool` (string-fallback helpers exist, `xml/model.go`).
- `equipment/reader.go` — info block struct literal (~`:88-118`); pet-equip abilities are **string-typed** `"0"/"1"` in `Character.wz/PetEquip`.
- `data/workers/character.go:70` — equipment registration walks all Character.wz subdirs, PetEquip included; no ingestion wiring needed, but **existing tenants need a re-ingest** for the new fields.

### atlas-configurations
- `seed-data/templates/template_*.json` — `PetItemUseHandle` entries on **eight** templates: gms_61 `0x8E`, gms_72 `0xA5`, gms_79 `0xA7`, gms_83 `0xAB`, gms_84 `0xB0`, gms_87 `0xB7`, gms_95 `0xCB`, jms_185 `0xAE`. All eight already carry `LoggedInValidator` (the gms_95 gap was fixed on main). No entry in gms_48 (four other pet handlers only) or in the partial gms_12 / gms_92 templates (no pet handlers at all).
- Writer entries needing the `petSkill` options table: `"CharacterInventoryChange"`, `"SetField"` (both encode pet cash assets). **Known gap, pre-existing on main, out of this task's scope to fix:** `template_gms_95_1.json` has **no `CharacterInventoryChange` writer entry at all** — opCode `0x1C` is `PicResult` and `0x1E` is `StatChanged` (the opCode gms_92 uses for `CharacterInventoryChange`); `CharacterInventoryChange` was dropped on gms_95 rather than moved to a new opCode. Consequence: on gms_95 the `FLAG_CHANGED` re-announce (`kafka/consumer/pet/consumer.go`'s `announcePetStatUpdate`) has no route to re-encode the pet cash asset via that writer, though `SetField`'s `petSkill` table (present on gms_95, confirmed at `template_gms_95_1.json` line ~1432) still carries the flag correctly on a full field load/relog. This task's own `petSkill` table addition to gms_95's `SetField` entry is unaffected and correct; the missing `CharacterInventoryChange` writer is a separate, pre-existing defect this task does not introduce and does not fix.
- Handler `options` flow: `HandlerConfig.Options` (`libs/atlas-opcodes/config.go:8`) → `BuildHandlerMap` → adapter → the handler's `readerOptions` param. So `readerOptions["skillGate"]` works with zero new plumbing.

## Load-bearing decisions

1. **FR-3 gate is version-family-specific, config-selected** (`skillGate: equipAbility` GMS / `petSkillFlag` JMS), because IDA showed GMS auto-pot gates on worn 1812xxx pet-ability equips (`CPet::UpdatePetAbility` ORs equip `dwPetAbilityFlag`; pouch skills never consulted), while only jms gates on the pouch flag (`GetUpgradePetSkill & 0x20`). Gating GMS on the pouch flag would falsely reject every legitimate player (FR-13 violation). Unknown/missing gate value fails closed (`skill_gate_unconfigured`).
2. **Server enforces exactly the client family's own gate** — never a stricter foreign one. On GMS a server rejection therefore implies a forged packet or state drift.
3. **Atlas-canonical flag bits ≠ wire bits** (DOM-25). Storage uses `pet/skill` bits `1<<0..1<<8` in 0519-key order; the wire `usPetSkill` short is translated per tenant through the sparse `petSkill` writer-options table. Unverified bits are omitted and encode as absent — never guessed.
4. **HP-vs-MP intent is not on the wire** (TryConsumePetHP/MP encode identical packets); the server classifies by the item's spec (`hp/hpR/mp/mpR`), and dual-recovery items pass on either matching source.
5. **`buffSkill` is log-only** (damage-mitigation context byte, IDA-verified per design §1.2). It appears in warn/debug logs, never in control flow.
6. **Validation lives in the socket handler**, not consumables: consumables has no session (cannot unstick) and the auto-pot command doesn't need the pet downstream. The 0519 pouch path is the opposite: petId rides the command because consumables performs the pet-side work.
7. **Parallel fetch, no new projection** — `model.NewGroup` fan-out (pet, character, item spec) keeps the hot path at ~1 RTT; a Kafka-fed cache was rejected as scope explosion (client-side cooldowns bound the rate).
8. **`SetPetFlag` is additive** (SetPetInfo signature untouched) so libs and services compile independently between tasks.
9. **No-op SET_SKILL writes ack but skip the event** (bit already in desired state) — prevents announce spam.
10. **`equip_data_missing` is a distinct reject reason**: worn candidates exist but carry no ability data → the atlas-data re-ingest hasn't happened. Makes the deploy-ordering failure diagnosable in logs.

## Planning-phase corrections to the design

- 0519 skill keys + `add` are under the item's **`info`** node, not `spec` (WZ-verified).
- Pet-equip ability attributes are **string-typed** in WZ; use the xml `GetBool` string fallback.
- atlas-channel had no `data/consumable`/`data/equipment` clients — they are created, not extended.
- `ResolveCode` returns a byte; autoSpeaking's verified wire bit is 0x100 → new `ResolveCode16`.
- The worn-equip position question (design §9 item 3) resolved during planning: raw equip compartment stores worn cash equips at `position − 100`; the gate normalizes rather than assuming.
- ~~gms_95's validator gap covers **eight** pet handlers~~ — fixed on main before the rebase; the repair left this task's scope.

## Rebase revision (branch updated onto main)

- **gms_48 has pet auto-pot and is in scope** (opcode **0x75**, encoder `0x70dc8d`, senders `0x6a840c`/`0x6a8596`, keymap ids at `CFuncKeyMappedMan+896/+900` via `0x4e5eb7`, gate `CPet+264/+272`). Its wire layout has **no petId** — single-pet client — so the codec gates the leading u64 on `(GMS && MajorAtLeast(61)) || JMS` and the handler resolves the character's spawned pet there (FR-1a). The matrix ⬜ came from `gms_v48/PetItemUse.json` recording `"function not found in IDB"` against a then-unnamed encoder; all the symbols are named now, so re-harvest. The `?TryConsumePetMP@CUserLocal@@` label at `0x7204db` was a mis-port — job 132 + `DarkKnightBerserkId` (1320006) + bodyless opcode 86 — renamed accordingly.
- **Nine versions total, not five.** gms_61/72/79 route `PetItemUseHandle` too. All three use the *same* worn-equip ability gate as v83 — verified this pass in `TryConsumePetHP` (v61 `0x7ad8e4`, v72 `0x86805e`, v79 `0x8b3a06`) and in v61's previously-unnamed `CPet::UpdatePetAbility` (`0x614b60`, named `CPet__UpdatePetAbility`), whose slot list and `0x01..0x80` ability-bit map are identical to v83's. So `skillGate: equipAbility` extends unchanged; no new gate mode.
- **Legacy serverbound layout already matches the codec** — encoders v61 `0x831ab9` (op 0x8E), v72 `0x903f8b` (0xA5), v79 `0x9552d0` (0xA7) all emit `petSN(8) · byte · updateTime(4) · slot(2) · itemId(4)`. The 🟡ᶠ cells need fixtures, not codec work (new Task 15).
- **`GW_ItemSlotPet::RawDecode` is not uniform**: v48 (`0x49c77e`) and v61 (`0x4b52f2`) read no `remainLife`/trailing `attribute`, v72 (`0x4d06dd`) reads `remainLife` only, v79 (`0x4d84c4`) and v83 (`0x4e4219`) read both. `encodePetCashItemInfo` has no version gates on main and over-encodes 6 bytes on v61 / 2 on v72 — corrected in Task 3 Step 3b because this task edits that exact function. **Note:** the plan's original Task 3 Step 3b text (written pre-rebase) only scoped this gate to v61/v72; the rebase-discovered gms_48 finding above extends the *same* shortened-trailer branch (`MajorAtLeast(72)` false) to v48 as well, so the landed `encodePetCashItemInfo` gates v48 alongside v61 (both omit `remainLife` and the trailing `attribute`; v72 omits only the trailing `attribute`) — confirmed against `libs/atlas-packet/model/asset.go`'s `MajorAtLeast(72)`/`MajorAtLeast(79)` conditionals and the per-version byte-length fixture `TestAssetPetCashItemTrailerVersionGate` in `libs/atlas-packet/model/asset_test.go`.
- **Open, cannot be closed from this workspace:** whether legacy tenant WZ carries the `consumeHP`/`consumeMP` pet-equip attributes at all (only v83 dumps exist locally). Task 15 Step 3 checks it against a live legacy tenant; if absent, the gate choice for that version is a user decision, not a silent default.
- **Task 15 Step 3 result: inconclusive, not "absent."** Queried live `atlas-data.atlas-main` (`GET /api/data/equipment/1812002` with `TENANT_ID`/`REGION`/`MAJOR_VERSION`/`MINOR_VERSION` headers) against the live GMS v61 tenant (`0d250dc9-64c4-45ae-8bc2-fc0a9cdb5578`): no `petAbilities` attribute in the response. But the SAME query against the live GMS v83 tenant (`ec876921-c363-4cc6-9c51-5bb8d57f9553`) also returned no `petAbilities`, even though the local v83 WZ dump (`tmp/ec876921-c363-4cc6-9c51-5bb8d57f9553/GMS/83.1/Character.wz/PetEquip/01812002.img.xml`) plainly has `consumeHP=1`. Root cause: `24cb23cc4` (task 5, "expose pet-ability equip attributes") is this branch's commit and is NOT an ancestor of `main` (`git merge-base --is-ancestor` confirms), so the live `atlas-data` pod is running pre-task-5 code and cannot surface `petAbilities` on ANY version yet — a deploy-lag gap, not a legacy-WZ-content gap. No local WZ dump exists for v48/v61/v72/v79 to check directly (only the v83 tenant and two unrelated others are extracted under `tmp/`). **Net: whether legacy WZ itself carries `consumeHP`/`consumeMP` remains genuinely unverified** — closing it needs either a legacy WZ extract or an environment running this branch's atlas-data past task 5. Not something producible from this workspace without one of those two inputs.
- `character_cash_item_use.go` grew on main — the `category == 519` site is now `:824`; locate by symbol, not by the line numbers recorded above.

## Resolved IDA findings (were listed as blocking during planning; now landed)

- **Task 8 Step 1** (was: jms `SendConsumeCashItemUseRequest` case-28 exact byte order): resolved as a bare `petId uint64`, no `updateTime` — see the `libs/atlas-packet` entry above.
- **Task 13 Step 3** (was: per-version usPetSkill bits): resolved and wired into every seed template's `petSkill` writer-options table (`CharacterInventoryChange`/`SetField`, verified by direct grep of `services/atlas-configurations/seed-data/templates/template_*.json`):
  - `autoSpeaking = 0x100` on **every** GMS version (gms_48 excepted — see below) and on jms.
  - jms additionally carries `consumeHP = 0x20` and `consumeMP = 0x40` (GMS templates carry only `autoSpeaking`, consistent with decision 1: GMS gates on worn equips, not the pouch flag, so the client never needs `consumeHP`/`consumeMP` on the wire).
  - **gms_48 correctly has no `petSkill` table at all** — v48 has no such wire mechanism (matches the FR-1a single-pet, no-multi-skill-wire finding); its template only gains the `PetItemUseHandle` entry at `0x75`.

## Dependencies between tasks

```
T1 constants ─┬─ T3 packet encode ── T11 channel flag sync ─┐
              ├─ T4 data cash ────── T7 consumables ────────┤
              ├─ T5 data equipment ─ T10 channel data ──────┼─ T12 handler ── T13 templates ── T14 verify
              ├─ T6 pets SET_SKILL ─ T7 consumables         │
T2 resolve16 ─┘   T8 jms packet ──── T9 channel cash arm ───┘
```
(T4/T5/T6/T8 are independent of each other; T7 needs T4+T6; T9 needs T8; T12 needs T10+T11.)

## Rollout notes (for the eventual deploy — record in the PR)

1. **Seed templates apply only to newly-created tenants.** Existing envs need a live config PATCH: `skillGate` handler options **×9 versions** (plus, on v48 tenants, the whole new `PetItemUseHandle` 0x75 entry — without it v48 auto-pot stays unhandled) (legacy included — an unpatched version rejects all auto-pot, because the gate fails closed) and `petSkill` writer tables on `CharacterInventoryChange`/`SetField` — then an atlas-channel restart (projection does not hot-reload handlers/writers). The gms_95 validator PATCH is no longer needed. **gms_95 exception:** its live tenant config has no `CharacterInventoryChange` writer to PATCH a `petSkill` table onto (see the known-gap note above) — only the `SetField` PATCH applies there; do not treat a missing `CharacterInventoryChange` PATCH target on gms_95 as an error.
2. **atlas-data re-ingest before the channel gate goes live on GMS tenants**, or every GMS auto-pot rejects with `equip_data_missing` (fail-closed by design, diagnosable in logs). Deploy order: atlas-data + re-ingest → atlas-pets/consumables → atlas-channel → config PATCH.
3. **No player-visible change for legitimate clients** (server now enforces the same gate the client already enforces). The new observable is warn logs on forged traffic.
4. Two acceptance items are runtime-only (pouch use end-to-end on a jms tenant; auto-pot pass-after-pouch): verify in a deployed env, not claimable from unit tests.
5. v83 WZ has **no consumeMP remover** (5191002 removes longRange etc.; no `add=0` consumeMP item exists) — expected, matches retail data.
