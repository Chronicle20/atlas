# Item Tag, Sealing Locks, and Incubator (Cash 506 Family) — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-07-02
---

## 1. Overview

The cash item family `Cash/0506.img` ("item imprints") contains seven items across three behaviors: **Item Tag** (5060000) stamps the using character's name onto an equipped item, **Sealing Lock** (5060001 permanent; 5061000–5061003 timed) sets the trade-block lock flag on an equip with an optional expiration, and **Incubator** (5060002) sacrifices a target item to hatch a random reward. All seven items exist in v83-era WZ data (verified against `Item.wz/Cash/0506.img.xml`: the timed locks carry `protectTime` values of 7/30/90/365 days).

Atlas already routes these items: `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go` maps `ClassificationItemImprints` to cash slot item types 25 (tag), 26 (seal), 27 (incubator), and 64/65/74 for the 5061xxx/5062xxx sub-ranges — but every arm falls through to the "unhandled" warning. The equip packet codec hardcodes an empty owner string (`libs/atlas-packet/model/asset.go:209,261,287`) and the inventory asset domain has no owner field at all. The lock flag infrastructure, by contrast, already exists (`libs/atlas-constants/asset/flag.go` `FlagLock`; `asset.Model.Locked()`); Sealing Lock is a mutation-and-expiry problem, not a new-field problem. The `INCUBATOR_RESULT` clientbound packet (`CWvsContext::OnIncubatorResult`) is unimplemented for all versions (STATUS.md row 89, ❌ across gms_v83/v84/v87/v95/jms).

This task implements all three behaviors end-to-end for **all supported tenant versions** (GMS v83/v84/v87/v92/v95, JMS v185), following the same carve-an-arm-out pattern as the sibling tasks (megaphones task-123, teleport rocks task-124, mastery books task-125, AP/SP resets task-126).

Reference behavior comes from Cosmic `UseCashItemHandler` itemType 506 (lines 232–289), with one deliberate deviation: Cosmic's incubator is half-implemented (rewards indexed by item id instead of a random roll, and `INCUBATOR_RESULT` never sent). Atlas implements the correct behavior: a weighted random roll from a tenant-configured reward pool, with the result packet.

## 2. Goals

Primary goals:
- Item Tag persists an owner-name snapshot on the target equip and renders it in the equip tooltip via the packet owner string.
- Sealing Lock sets `FlagLock` on the target equip; timed variants apply a WZ-derived expiration after which the lock clears without destroying the item.
- Incubator consumes the used incubator and the sacrificed target item, grants a weighted-random reward from a tenant configuration resource, and responds with `INCUBATOR_RESULT`.
- Owner field exists end-to-end: DB column → asset domain model → REST/Kafka → atlas-channel projection → packet codec (replacing the hardcoded `WriteAsciiString("")`).
- All flows work on every supported tenant version, with byte-fixtured packet verification.

Non-goals:
- Cosmetic coupons (515x family) — separate backlog item.
- Pet imprints (546x) / character imprints (540x) — already routed elsewhere; unrelated internals.
- Changes to trade/MTS enforcement beyond honoring the existing `FlagUntradeable`/`FlagLock` semantics.
- UI (atlas-ui) management pages for the incubator reward configuration — the resource is seeded via tenant templates and manageable via the atlas-tenants REST API.

## 3. User Stories

- As a player, I want to use an Item Tag on an equipped item so that my character's name is permanently stamped on it and visible in its tooltip.
- As a player, I want to use a Sealing Lock on an equip so that it cannot be dropped, traded, or sold accidentally.
- As a player, I want a timed Sealing Lock (7/30/90/365 days) to expire on schedule so that the item becomes tradeable again without being destroyed.
- As a player, I want to use an Incubator on a sacrificial item so that I receive a random reward, with the hatch result shown by the client.
- As a server operator, I want the incubator reward pool defined per tenant in configuration so that I can tune reward tables without a code change.

## 4. Functional Requirements

### 4.1 Item Tag (5060000, cash slot type 25)

1. Decode the type-25 serverbound sub-body of the cash `ItemUse` packet per version. The v83 reference read order (Cosmic) is: `short` target equipped slot, trailing `int` updateTime. Each version's actual read order MUST be IDA-verified before implementation (do not assume Cosmic's order for v87+/JMS).
2. The target MUST be an item in the **equipped** compartment (negative slot semantics). A zero/invalid slot, an empty slot, or a non-equip target is a validated no-op (log at warn, consume nothing).
3. On success: set the asset's owner to the using character's **current name as a snapshot string** (later renames do not update the tag), consume 1 Item Tag from the cash compartment, and push an inventory update to the client so the tooltip refreshes without relog.
4. The mutation and the consume MUST be atomic via the saga orchestrator (compensation restores state on partial failure).
5. Re-tagging an already-owned equip overwrites the owner (matches reference behavior — `setOwner` is unconditional).

### 4.2 Sealing Lock (5060001, 5061000–5061003; cash slot types 26 and 64/65)

1. Decode the type-26/64/65 sub-body per version. The v83 reference read order is: `int` inventory type, `int` slot, trailing `int` updateTime. IDA-verify per version.
2. The target MUST be an equip (equip compartment or equipped slot, per decoded inventory type). Non-equip targets are a validated no-op. Empty slot is a validated no-op.
3. On success: set `FlagLock` (`libs/atlas-constants/asset/flag.go` 0x01) on the target asset and consume the lock item, atomically via saga.
4. Timed variants apply an expiration of `now + protectTime days`, with `protectTime` read from WZ item data via atlas-data (verified values: 5061000=7, 5061001=30, 5061002=90, 5061003=365). 5060001 sets the flag with no expiration (permanent).
5. Expiration semantics: the lock rides the asset's expiration field (this is what the equip packet encodes and what the client renders as a lock timer). When a **locked** asset's expiration passes, the expire flow MUST clear `FlagLock` and reset the expiration instead of destroying the asset. Assets expiring without `FlagLock` keep today's destroy behavior.
6. Guard: using a sealing lock on an asset that already has a non-lock expiration (a genuinely time-limited item) is rejected (reference behavior — prevents laundering an expiring item into a permanent one). Stacking a timed lock onto an already-locked item extends from the current expiration.

### 4.3 Incubator (5060002, cash slot type 27)

1. Decode the type-27 sub-body per version. The v83 reference read order is: `int` inventory type, `int` slot (the sacrificial target item), trailing `int` updateTime. IDA-verify per version.
2. On success, atomically via saga: destroy 1 of the target item, destroy 1 incubator, grant the rolled reward, and emit `INCUBATOR_RESULT` to the using client with the rewarded item id/quantity.
3. The reward is a **weighted random roll** over the tenant's incubator reward configuration resource (§5/§6). An empty or missing pool for the tenant is a validated no-op (log at warn, consume nothing).
4. If the reward cannot be granted (target inventory full), the use is rejected before anything is consumed.
5. `INCUBATOR_RESULT` (`CWvsContext::OnIncubatorResult`) writer implemented for all supported versions, registered in seed templates AND patched into live tenant configs (new opcodes do not reach existing tenants otherwise), with byte fixtures per version promoted through the packet coverage matrix (`docs/packets/audits/VERIFYING_A_PACKET.md` flow).

### 4.4 Owner field end-to-end

1. New `owner` string column on the inventory asset storage (migration; default empty string; no backfill needed).
2. Asset domain model/builder expose `Owner()`/`SetOwner` following the immutable-model + builder pattern; REST JSON:API and Kafka asset representations carry the attribute.
3. atlas-channel's asset projection populates owner, and `libs/atlas-packet/model/asset.go` encodes it in the equip codecs (`encodeEquipableInfo`, `encodeCashEquipableInfo`) replacing the hardcoded `WriteAsciiString("")`. The stackable codec (line 287) continues writing the asset's owner value (empty for stackables — tags target equips only) rather than a hardcoded literal.
4. Existing byte fixtures that assert an empty owner string keep passing for assets with no owner; new fixtures cover a non-empty owner.

### 4.5 Handler routing

1. The three arms hook into `CharacterCashItemUseHandleFunc` at the existing `GetCashSlotItemType` mapping (types 25/26/27, and 64/65 for 5061xxx per version branch). No changes to the classification mapping are expected.
2. Type 74 (5062xxx on GMS ≥95): investigate during design whether v95-era WZ actually ships 5062xxx items; if the tenant data has none, explicitly document it as dead routing and leave the arm unimplemented (validated no-op), not silently ignored.

## 5. API Surface

### atlas-tenants — new configuration resource `incubator-rewards`

Follows the established generic-JSONB configuration pattern (`configurations` table; REST model + Transform/Extract + providers + processor methods + handlers + routes + Kafka events + mock update).

- `GET /tenants/{tenantId}/configurations/incubator-rewards` — JSON:API list of reward entries.
- Entry attributes: `itemId` (uint32), `quantity` (uint32, ≥1), `weight` (uint32, ≥1). Roll = weighted choice over entries.
- Standard configuration create/update semantics as for `routes`/`vessels`/`instance-routes`.
- Seed templates for all supported tenant versions include a starter pool (design phase picks contents; WZ-plausible v83-era item ids only, verified against local WZ data).

### atlas-inventory — asset mutations

- New compartment/asset Kafka command(s) alongside the existing set (`kafka/message/compartment/kafka.go`): set-owner and apply-lock (flag + optional expiration delta). Exact command shape (new commands vs. extending `MODIFY_EQUIPMENT`) is a design decision.
- Corresponding status events so atlas-channel can emit the client inventory-update packet.
- REST asset representation gains the `owner` attribute.

### atlas-saga-orchestrator

- New saga step action(s) for set-owner and apply-lock; incubator flow composed from existing `DestroyAsset` + award/create-asset steps plus the result emission. Error cases compensate (nothing consumed on failed grant).

## 6. Data Model

- `asset` table (atlas-inventory): add `owner` column, `varchar`, not null, default `''`. GORM AutoMigrate-compatible additive change; note the baseline publish/restore column-order caveat — publish/restore use explicit name-keyed column lists, so verify the baseline tooling covers the new column (re-publish canonical baselines if the asset table participates).
- Lock expiration reuses the existing asset `expiration` field (no new column). The expire flow branches on `FlagLock` (§4.2.5).
- Incubator reward pool lives in atlas-tenants generic JSONB `configurations` (`resource_name = "incubator-rewards"`), tenant-scoped by design; no new table.

## 7. Service Impact

- **atlas-channel** — three handler arms + per-version sub-body decoders in the cash `ItemUse` flow; `INCUBATOR_RESULT` writer registration; asset projection carries owner.
- **atlas-inventory** — owner column + domain field; set-owner and apply-lock command handling; lock-aware expire flow; status events.
- **atlas-tenants** — `incubator-rewards` configuration resource (full pattern incl. mock).
- **atlas-saga-orchestrator** — new step actions; incubator saga composition.
- **libs/atlas-packet** — owner in the three asset codecs; `INCUBATOR_RESULT` clientbound writer + serverbound sub-body structs; byte fixtures.
- **libs/atlas-constants** — item id constants for the seven 506 items if not already present.
- **Tenant seed templates + live configs** — `INCUBATOR_RESULT` opcode per version; live-config patch + channel restart for existing tenants.
- **atlas-data** — read path for `protectTime` from cash item WZ data if not already exposed (verify; add spec field if missing).

## 8. Non-Functional Requirements

- **Multi-tenancy**: all behavior tenant-scoped via `tenant.MustFromContext(ctx)`; reward pools per tenant; per-version packet structure resolved from tenant, never hardcoded bytes (config-derived codes where a mode/operations table applies).
- **Atomicity**: every consume+mutate pair goes through the saga orchestrator; no fire-and-forget multi-step mutations. Saga timeouts must account for step count (flat-timeout bug class).
- **Verification**: `go test -race`, `go vet`, `go build` per changed module; `docker buildx bake` for every service whose `go.mod` changed; `tools/redis-key-guard.sh`; packet byte fixtures with `packet-audit` markers + matrix regeneration for `INCUBATOR_RESULT` cells and new serverbound sub-bodies.
- **Observability**: validated no-ops log at warn with character/item/slot context, consistent with the existing handler's style.
- **Safety**: all target-slot inputs are client-controlled — validate compartment type, slot occupancy, and item classification server-side before mutating (anti-hacking parity with reference checks).

## 9. Open Questions

1. Exact serverbound read order for the three sub-bodies on v87/v92/v95/JMS — requires IDA verification during design/implementation (v83/v84 expected to match the Cosmic reference; do not assume for later versions).
2. Whether GMS ≥95 tenants ship 5062xxx items (slot type 74) — resolve from v95 WZ data during design; if absent, document dead routing (§4.5.2).
3. Whether the apply-lock mutation extends `MODIFY_EQUIPMENT` or adds a dedicated command — design decision.
4. Starter contents of the seeded incubator reward pools — design phase, WZ-verified item ids.

## 10. Acceptance Criteria

- [ ] Using Item Tag on an equipped item shows the character's name in the item tooltip immediately and survives relog; the tag is consumed; renaming the character does not change existing tags.
- [ ] Item Tag on an empty/invalid/non-equip slot consumes nothing and logs a warning.
- [ ] Sealing Lock sets the lock flag (visible client-side); timed variants show the correct expiration (7/30/90/365 days per WZ `protectTime`); the lock item is consumed.
- [ ] A locked asset whose lock expiration passes loses the flag and keeps existing; unlocked expiring assets still destroy as today.
- [ ] Sealing a genuinely expiring (non-locked, expiration-bearing) item is rejected.
- [ ] Incubator destroys the target item and the incubator, grants a weighted-random reward from the tenant's `incubator-rewards` configuration, and the client displays the hatch result via `INCUBATOR_RESULT`.
- [ ] Incubator with full target inventory or an empty tenant pool consumes nothing.
- [ ] All flows verified on every supported tenant version (GMS v83/v84/v87/v92/v95, JMS v185).
- [ ] `INCUBATOR_RESULT` STATUS.md cells promoted with byte fixtures + pinned evidence for all versions; new serverbound sub-bodies byte-fixtured.
- [ ] Asset owner field present in DB, domain, REST, Kafka, and packet encode; existing empty-owner fixtures unaffected.
- [ ] Live tenant configs patched with the new opcode(s) and channel restarted (or documented as the deploy step).
- [ ] Full verification suite clean: `go test -race ./...`, `go vet ./...`, `go build ./...`, `docker buildx bake` for touched services, `tools/redis-key-guard.sh`.
