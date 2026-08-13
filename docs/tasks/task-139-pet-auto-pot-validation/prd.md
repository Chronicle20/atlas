# Pet Auto-Pot Validation & Pet Skill Pouches — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-07-09
---

## 1. Overview

The `PET_AUTO_POT` serverbound packet (`CWvsContext::SendStatChangeItemUseRequestByPetQ`, STATUS.md row 691) is routed on **eight** of the nine matrix versions — gms_61 (0x8E), gms_72 (0xA5), gms_79 (0xA7), gms_83 (0xAB), gms_84 (0xB0), gms_87 (0xB7), gms_95 (0xCB), jms_185 (0xAE); the last five are ✅ and the three legacy columns are 🟡ᶠ (fixture-pending, layout confirmed identical — design §1.2). **gms_48 is the ninth**: the matrix shows ⬜ n-a, but that is an audit artifact — the client sends this packet at opcode 0x75 with a petId-less layout, and the harvest simply could not find the then-unnamed encoder (design §1.7). It is handled by `PetItemUseHandle` in `services/atlas-channel/atlas.com/channel/socket/handler/pet_item_use.go`. Today the handler decodes the packet and forwards blindly to the generic `consumable.RequestItemConsume` flow. It ignores the packet's `petId` and `buffSkill` fields entirely and performs no pet-side validation: any client can trigger a pet auto-pot consume with no spawned pet, with a pet it does not own, while dead, or without the pet skill that legitimately enables auto-pot. The potion still applies — this is an abuse/validation gap, not a functional break.

Closing the gap requires more than a spawned-pet check. Legitimate auto-pot in GMS is gated by pet skill pouch items — WZ-verified in v83 (`Item.wz/Cash/0519.img.xml`): `5190001` grants `consumeHP` (Auto HP), `5190006` grants `consumeMP` (Auto MP), `5191001` removes `consumeHP` (`add=0`; no `consumeMP` remover exists in v83 data). Atlas has a `flag uint16` field on the pet model (atlas-pets and the atlas-channel projection, exposed in the pets REST model) intended to hold these skill bits, but **nothing ever sets it, and it is not persisted** (no column in `pet/entity.go`). This task therefore includes a minimal pet-skill pouch implementation — applying/removing skill flags via 0519 item use, persisting them — as the prerequisite for enforcing the pouch check.

Reference-server context (verified against Cosmic source): Cosmic's `PetAutoPotHandler`/`PetAutopotProcessor` does *not* check pet-spawned or pouch possession; it checks character-alive and slot/item match only. This task makes Atlas stricter than Cosmic and closer to intended retail behavior. Cosmic's multi-pot drain (`USE_COMPULSORY_AUTOPOT`) and server-side autopot retrigger are Cosmic-custom features and are out of scope.

## 2. Goals

Primary goals:
- Reject pet auto-pot requests that are not backed by a specifically identified, owned, currently-spawned pet.
- Reject pet auto-pot requests from dead characters.
- Reject pet auto-pot requests when the identified pet lacks the matching pet skill (`consumeHP` for HP recovery, `consumeMP` for MP recovery).
- Implement pet skill pouch item application/removal (item category 0519) so the flag gate is satisfiable by legitimate play: flag bits set/cleared on the pet, persisted, and projected to atlas-channel.
- On rejection: send the client-unstick response (enable actions / stat-changed) and log at warn level with enough context to identify abuse (character, pet, item, reason) — a rejected request is erroneous client behavior, not a normal path.

Non-goals:
- Implementing behaviors for the other 0519 pet skills (`pickupItem`, `longRange`, `dropSweep`, `pickupAll`, `ignorePickup`, `recall`, `autoSpeaking`). Flag storage is generic; only the auto-pot bits gain enforcement behavior.
- Cosmic's multi-pot drain-to-ratio consumption and server-side autopot retrigger on HP depletion.
- Keymap work — the auto-HP/MP keymap slots (91/92) are already encoded via `CharacterKeyMapAutoHp`/`CharacterKeyMapAutoMp` writers.
- Any change to the downstream reservation flow in atlas-consumables for the consume itself (slot/item/quantity mismatch is already rejected by `RequestReserve`).

## 3. User Stories

- As a server operator, I want pet auto-pot requests validated against pet ownership, spawn state, and pet skills so that packet-editing clients cannot consume potions through a pathway they have not legitimately unlocked.
- As a server operator, I want rejected auto-pot requests logged at warn level so that I can identify players sending forged packets.
- As a player, I want to use an Auto HP Potion Pouch (5190001) / Auto MP Potion Pouch (5190006) on my pet so that my pet automatically feeds me potions at my configured HP/MP thresholds.
- As a player, I want a skill remover (e.g. 5191001) to clear the corresponding skill from my pet.
- As a player with a legitimate pet and pouch, I want auto-pot to keep working exactly as before — validation must not add false rejections.

## 4. Functional Requirements

### A. Auto-pot request validation (atlas-channel)

FR-1 **Specific pet validation.** `PetItemUseHandle` MUST resolve the packet's `petId` (currently ignored) and reject the request unless: the pet exists, `OwnerId()` equals the requesting character id, and the pet is currently spawned (`Slot() >= 0`; spawned-slot semantics per `atlas-pets pet/processor.go:210`). "Character has some spawned pet" is NOT sufficient.

FR-1a **Single-pet versions.** On gms_48 the packet carries no `petId` at all (design §1.7 — the client holds one pet, not an array), so there is nothing to identify. There the handler MUST resolve the character's spawned pet and apply the identical ownership/spawn/skill checks; no spawned pet still rejects. This is not a relaxation of FR-1 — with at most one pet the two rules coincide. The petId-bearing versions MUST NOT fall back to this lookup.

FR-2 **Alive check.** The request MUST be rejected when the requesting character's HP is 0.

FR-3 **Pet skill gate.** The identified pet MUST have the matching skill flag: `consumeHP` when the consumed item recovers HP, `consumeMP` when it recovers MP. For items that recover both, possession of either flag is sufficient (design may tighten this if IDA shows the client gates differently — see Open Questions).

FR-4 **Rejection behavior.** On any FR-1/2/3 failure the handler MUST NOT forward to `RequestItemConsume`. It MUST send the client-unstick response (enable-actions / stat-changed empty update, matching what the client expects to release the pending item-use state) and MUST log at warn level including character id, pet id, item id, slot, and the rejection reason.

FR-5 **Pass-through.** On successful validation, behavior is unchanged: forward to `consumable.RequestItemConsume` exactly as today.

FR-6 **`buffSkill` field.** The design phase MUST investigate (IDA, per-version) what the client encodes in the `buffSkill` byte of `SendStatChangeItemUseRequestByPetQ` and specify handler behavior accordingly. Until specified, the value is logged; it must not be silently load-bearing.

### B. Pet skill pouch system (prerequisite)

FR-7 **Item classification.** Add classification `519` to `libs/atlas-constants/item` (pet skill items; sits between existing `ClassificationPetImprints = 517` and `ClassificationPetConsumable = 524`).

FR-8 **Pouch application.** Using a 0519 item MUST apply (`add=1` in item data) or remove (`add=0`) the corresponding pet skill bit on the target pet, and consume the item via the existing reservation flow. The skill-to-item mapping is data-driven from item WZ attributes (`pickupItem`, `consumeHP`, `longRange`, `dropSweep`, `pickupAll`, `ignorePickup`, `consumeMP`, `recall`, `autoSpeaking`) — not hardcoded per item id. atlas-data MUST expose these attributes if it does not already.

FR-9 **Flag persistence.** The pet `flag` MUST be persisted in atlas-pets (new column on the pet entity; currently the model/REST field exists but no entity column does). Existing pets default to 0.

FR-10 **Flag projection.** Flag changes MUST be observable by atlas-channel through the existing pet event/REST patterns so FR-3 evaluates current state (the channel pet projection already carries `Flag()`).

FR-11 **Target pet.** Applying a pouch MUST target a specific pet. The design phase determines how the client identifies the target pet in the 0519 item-use packet (single-pet default vs. multi-pet selection) and mirrors it.

FR-12 **Client synchronization.** The design phase MUST determine (IDA) how the v83+ client learns a pet's skill state (pet spawn/stat packet attribute vs. client-inferred). If a clientbound wire value carries the flag, its bit values are client-interpreted and MUST be config-resolved per tenant version (DOM-25) — no hardcoded bit constants in domain services.

### B2. Version coverage (added after the rebase onto main)

FR-14 **Every version whose client speaks this packet is configured.** Nine:
the five original + v61/v72/v79 + **gms_48, which additionally needs its
`PetItemUseHandle` entry created** (opcode 0x75 — the client has the feature, the
template never routed it). Because the gate fails closed when unconfigured, a
version left out of the seed templates is a regression (all auto-pot rejected),
not a no-op. Only the partial gms_12 / gms_92 templates are excluded, on evidence
(design §3.6).

FR-15 **Legacy fixtures and codec gate.** The v61/v72/v79 `PET_AUTO_POT` matrix
cells MUST be promoted off 🟡ᶠ with byte fixtures pinned to the verified
encoders, and the gms_48 cell MUST be re-harvested off ⬜ now that its encoder is
named. The codec MUST version-gate the leading `petId u64` — present for
`(GMS && MajorAtLeast(61)) || JMS`, absent on v48 — with a fixture per version,
so the version set this task claims to support is the version set the matrix
shows.

FR-16 **Pet-item trailer gates.** The pet cash-asset encode this task modifies
MUST be version-gated to the layout each client actually reads (design §1.6:
v61 omits `remainLife` + trailing `attribute`; v72 omits the trailing
`attribute`). This corrects a pre-existing over-encode on main that the task's
own edit would otherwise silently inherit.

### C. Non-regression

FR-13 A character with a spawned, owned pet holding the matching pouch flag experiences identical auto-pot behavior to today (no new latency-visible failure modes, no double enable-actions when the downstream reservation also fails).

## 5. API Surface

- **No new REST endpoints expected.** The pets REST model already exposes `flag` (`atlas-pets pet/rest.go`); it starts reflecting real data.
- **Kafka:** a new command (atlas-consumables → atlas-pets) to apply/remove a pet skill flag as part of the 0519 consume branch, plus the corresponding pet status event on flag change, following the existing consumable→pets command/event patterns (as with `ConsumeCashPetFood`). Exact topic/type names are design-phase decisions following existing conventions.
- **Error cases:** rejected auto-pot requests produce no new API surface — socket-level unstick response + warn log only.

## 6. Data Model

- `atlas-pets` pet entity: add `flag` (uint16, NOT NULL, default 0) column. AutoMigrate-added column; no backfill needed (0 = no skills, correct for all existing pets).
- No other schema changes. Item skill attributes come from WZ data via atlas-data, not the database.

## 7. Service Impact

- **atlas-channel** — `PetItemUseHandle` gains the validation chain (pet lookup by id, ownership/spawn/flag checks, character-alive check, unstick response, warn logging). Uses the existing channel `pet` and `character` processors.
- **atlas-pets** — flag persistence (entity column), apply/remove-skill processor + Kafka command consumer + status event emission.
- **atlas-consumables** — new `RequestItemConsume` branch for classification 519 emitting the pet-skill command (analogous to the existing `ClassificationPetConsumable` branch).
- **atlas-data** — expose 0519 item skill attributes if not already readable.
- **libs/atlas-constants** — new item classification constant (519).
- **libs/atlas-packet** — only if FR-12's investigation shows a clientbound packet must carry pet skill state; otherwise untouched.

## 8. Non-Functional Requirements

- **Multi-tenancy:** all lookups and events tenant-scoped via context, per existing patterns; validation semantics identical across supported versions unless IDA shows version differences.
- **Logging:** rejections at warn (erroneous client behavior), successful validation silent beyond existing debug lines. Logs must carry character id, pet id, item id, slot, reason.
- **Performance:** validation adds pet + character lookups on a hot-ish path (auto-pot can fire on every damage tick under threshold). Design should reuse the channel's existing projections/caches where available rather than adding per-request REST round-trips if a cheaper source exists.
- **Wire-value discipline (DOM-25):** any client-interpreted flag bits or packet bytes introduced by FR-12 resolve from tenant configuration, never hardcoded constants.
- **Security posture:** validation failures are treated as forged-packet signals; no client-visible error detail beyond the unstick response.

## 9. Open Questions

1. **`buffSkill` semantics** — what does the client put in this byte, per version? (IDA, design phase; FR-6.)
2. **Client knowledge of pet skills** — does the client require a wire update (pet spawn/stat attribute) to enable auto-pot sends, or does it track pouch use locally? Determines FR-12 scope. (IDA.)
3. **Flag bit values** — the client's pet-attribute bit assignments for `consumeHP`/`consumeMP` (and the other seven skills), per version. (IDA; feeds FR-12/DOM-25 decision.)
4. **Pouch target-pet encoding** — how the 0519 item-use packet designates the pet when multiple pets are spawned (multi-pet exists in v83 via Follow the Lead). (FR-11.)
5. **Dual HP+MP potions** — is "either flag" (FR-3) the client's actual gate, or does the trigger (auto-HP vs auto-MP keyslot) determine the required skill? The packet does not carry the trigger explicitly — unless investigation shows `buffSkill` or another field encodes it.
6. **Does the v83 client itself gate the auto-pot send on pouch possession?** If yes, legitimate clients never hit FR-3 rejections; if no, FR-3 changes behavior for vanilla clients and the rollout note must say so.
7. **No `consumeMP` remover in v83 WZ** — confirm this matches retail (likely yes; it simply means the MP skill can't be removed in v83).

## 10. Acceptance Criteria

- [ ] Auto-pot request with no spawned pet, a non-owned petId, or an unspawned petId is rejected: no consume occurs, unstick response sent, warn log emitted with character/pet/item/reason.
- [ ] Auto-pot request from a dead character (HP 0) is rejected the same way.
- [ ] Auto-pot request for an HP potion from a pet without `consumeHP` (and analogously MP/`consumeMP`) is rejected the same way.
- [ ] Using item 5190001 on a spawned pet sets its `consumeHP` flag, consumes the item, persists across pet despawn/respawn and service restart, and is visible in the pets REST model.
- [ ] Using item 5191001 clears `consumeHP`; using 5190006 sets `consumeMP`.
- [ ] After applying the matching pouch, the same auto-pot request that was rejected now passes and the potion applies exactly as before this task.
- [ ] `buffSkill` handling is specified and implemented per the design-phase IDA finding (or explicitly documented as ignored with the IDA evidence cited).
- [ ] Client-interpreted wire values introduced by this task (if any) are config-resolved per tenant version (DOM-25).
- [ ] Auto-pot behaves identically on all nine versions: a legitimate v48/v61/v72/v79 player with the matching worn pet-ability equip is not rejected (FR-14), and no version's `PetItemUseHandle` entry is left without `skillGate`.
- [ ] gms_48 routes `PetItemUseHandle` at 0x75 and its auto-pot validates via the spawned-pet lookup (FR-1a), with no `petId` read from the wire.
- [ ] v48/v61/v72/v79 `PET_AUTO_POT` cells are ✅ in the regenerated matrix (FR-15); no cell is left ⬜ on the strength of a "function not found in IDB" note.
- [ ] The pet cash-asset encode emits exactly the trailer each client reads (FR-16), proven by per-version byte fixtures.
- [ ] `go test -race ./...`, `go vet ./...`, `go build ./...` clean in every changed module; `docker buildx bake` clean for every touched service; `tools/redis-key-guard.sh`, `tools/goroutine-guard.sh`, `tools/lint.sh --check` and (templates changed) `tools/template-opcode-order-guard.sh` clean.
