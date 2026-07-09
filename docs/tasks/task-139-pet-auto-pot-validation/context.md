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
- `seed-data/templates/template_*.json` — `PetItemUseHandle` entries: gms_83 `0xAB`, gms_84 `0xB0`, gms_87 `0xB7`, gms_95 `0xCB` (**no validator — dead**, along with the seven other pet handlers 0x52/0x6E/0xC7-0xCC), jms_185 `0xAE`.
- Writer entries needing the `petSkill` options table: `"CharacterInventoryChange"`, `"SetField"` (both encode pet cash assets).
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
- gms_95's validator gap covers **eight** pet handlers, not five.

## Remaining IDA-blocking items (in-plan, not deferred)

- **Task 8 Step 1**: jms `SendConsumeCashItemUseRequest` case-28 exact byte order (updateTime placement around the petId u64). Escalate if the arm can't be located — never substitute a layout.
- **Task 13 Step 3**: per-version usPetSkill bits (jms consumeMP expected 0x40; autoSpeaking per version). Omit any bit that fails verification.

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

1. **Seed templates apply only to newly-created tenants.** Existing envs need a live config PATCH: `skillGate` handler options ×5 versions, gms_95 pet-block validators, `petSkill` writer tables on `CharacterInventoryChange`/`SetField` — then an atlas-channel restart (projection does not hot-reload handlers/writers).
2. **atlas-data re-ingest before the channel gate goes live on GMS tenants**, or every GMS auto-pot rejects with `equip_data_missing` (fail-closed by design, diagnosable in logs). Deploy order: atlas-data + re-ingest → atlas-pets/consumables → atlas-channel → config PATCH.
3. **No player-visible change for legitimate clients** (server now enforces the same gate the client already enforces). The new observable is warn logs on forged traffic.
4. Two acceptance items are runtime-only (pouch use end-to-end on a jms tenant; auto-pot pass-after-pouch): verify in a deployed env, not claimable from unit tests.
5. v83 WZ has **no consumeMP remover** (5191002 removes longRange etc.; no `add=0` consumeMP item exists) — expected, matches retail data.
