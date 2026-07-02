# Teleport Rocks (Regular + VIP) and the Saved-Map List — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-07-02
---

## 1. Overview

MapleStory's teleport rocks let a player warp instantly to a map they have saved (or to another player), instead of walking or taking transport. The regular Teleport Rock is a consumable with a 5-slot saved-map list; the VIP Teleport Rock is a duration-based cash item with its own 10-slot saved-map list that the player manages in-game (add/remove maps via the item UI).

Atlas currently has no teleport rock support at all. The only traces are the classification constant (`ClassificationTeleportRock = 504` in `libs/atlas-constants/item/constants.go:73`), the atlas-data classification string (`services/atlas-data/atlas.com/data/item/classify.go:181`), and the cash-slot type mapping in `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:137` — which then falls through to the warn-and-drop path. The character-data codec hardcodes all 5 regular + 10 VIP trock slots to `EmptyMapId` (`libs/atlas-packet/character/data.go:700-709`), so the client always shows an empty list. All three packet operations are unimplemented across every version in the coverage matrix (`docs/packets/audits/STATUS.md`): `USE_TELEPORT_ROCK` ❌, `TROCK_ADD_MAP` ❌, `MAP_TRANSFER_RESULT` ❌.

This task implements the feature end-to-end for all supported tenant versions: persistence of both saved-map lists in atlas-character, the two serverbound operations (use rock, add/remove map), the clientbound result writer, threading the real lists into the character-data codec, and the warp/consume saga — with full byte-fixture packet verification per the packet-audit discipline.

## 2. Goals

Primary goals:
- A player can use a regular Teleport Rock (2320000) or cash rock (5040000 "The Teleport Rock", 5040001 "Teleport Coke", 5041000 "VIP Teleport Rock") to warp to a saved map or to a target player, with faithful (client/IDA-verified) validation rules.
- A player can add and remove maps from the regular (5-slot) and VIP (10-slot) saved-map lists, and the lists persist across sessions and channel changes.
- The character-data packet carries the real saved-map lists so the client UI shows them.
- Failed warps return the correct `MAP_TRANSFER_RESULT` error to the client and consume nothing.
- All three packet operations are byte-fixture verified (packet-verifier discipline) on every IDB-backed version.

Non-goals:
- Married-partner map transfer (`NOTIFY_MARRIED_PARTNER_MAP_TRANSFER`).
- Dojo warp, inner-portal teleport, or any other `CUserLocal::OnTeleport`-family operation.
- Post-Big-Bang Hyper Rock.
- Cash Shop purchase flow for the rocks (commodity data already exists; buying is the cash shop feature's concern).
- atlas-ui changes.

## 3. User Stories

- As a player, I want to save my current map to my teleport rock list so that I can return to it later.
- As a player, I want to remove a map from my teleport rock list so that I can free a slot.
- As a player, I want to use a teleport rock to warp to a saved map so that I skip travel time.
- As a player, I want to use a teleport rock to warp to another player by name so that I can join them quickly.
- As a player, I want a clear in-game error (not a silent no-op) when a warp is not allowed so that I understand why and my item is not wasted.
- As a player, I want my saved lists to survive relogs, channel changes, and world transfers within the tenant so that the lists are durable.

## 4. Functional Requirements

### 4.1 Item semantics (verified against local GMS 83.1 WZ)

| Item | Name | Source | Kind | List used | Consumption |
|---|---|---|---|---|---|
| 2320000 | Teleport Rock | `Item.wz/Consume/0232.img.xml` | USE item, `slotMax=1`, `timeLimited=1`, `only=1`, `tradeBlock=1`, `notSale=1` | regular (5 slots) | consume 1 on **successful** warp |
| 5040000 | The Teleport Rock | `Item.wz/Cash/0504.img.xml` | cash item | regular (5 slots) | not consumed per use; item expiry governs |
| 5040001 | Teleport Coke | `Item.wz/Cash/0504.img.xml` | cash item | regular (5 slots) | not consumed per use; item expiry governs |
| 5041000 | VIP Teleport Rock | `Item.wz/Cash/0504.img.xml` | cash item | VIP (10 slots) | not consumed per use; item expiry governs |

- FR-1: A failed warp (any validation error) MUST NOT consume the regular rock and MUST emit the appropriate `MAP_TRANSFER_RESULT` error.
- FR-2: Cash rocks (504x) MUST NOT be destroyed on use; the regular rock (2320000) MUST be decremented by exactly 1 only after the warp step succeeds (saga ordering: validate → warp → consume).

### 4.2 Saved-map lists

- FR-3: Each character has two independent ordered lists: regular (max 5 entries) and VIP (max 10 entries). Empty slots are represented as `_map.EmptyMapId` (999999999) on the wire.
- FR-4: `TROCK_ADD_MAP` requests carry an add/remove flag and a list type (regular vs VIP); exact byte layout per version MUST be IDA-derived during design (see §9 Q1).
- FR-5: Adding a map: the map being added is the character's **current** map (client sends the map id; server MUST validate it equals the character's current map id). Reject when: list is full, map already present in that list, or the current map is barred from being saved (field-limit rule, IDA/WZ-verified — see §9 Q2).
- FR-6: Removing a map: reject when the map id is not present in the list.
- FR-7: Every successful add/remove MUST persist and immediately emit the `MAP_TRANSFER_RESULT` mode that refreshes the client's list (the client updates its UI from this packet, not optimistically).
- FR-8: Lists are created empty for new characters and deleted when the character is deleted.

### 4.3 Using a rock (warp)

- FR-9: `USE_TELEPORT_ROCK` (`CWvsContext::SendMapTransferItemUseRequest`) carries a target discriminator (warp-to-saved-map vs warp-to-player-by-name), the item source slot/id, and the target payload. Exact per-version layout MUST be IDA-derived during design (see §9 Q1).
- FR-10: Regular rock target validation (faithful): target map must be in the character's regular list (or the target player located), target must be reachable per regular-rock rules (same-region restriction as enforced by the v83-era client/server contract — exact rule IDA-verified, see §9 Q3), and neither the source map nor the target map may forbid teleport-rock use via field limits.
- FR-11: VIP rock relaxes the region restriction (may warp across continents) but still honors field-limit bans on both ends.
- FR-12: Warp-to-player: resolve the target character by name within the same tenant + world; reject if offline, not found, in a barred map, or (regular rock) outside the allowed region. On success the warp target is the target player's current map + a valid portal.
- FR-13: All rejection cases map to the correct `MAP_TRANSFER_RESULT` error mode (e.g. "unable to find the character" / "cannot go to that place" — exact mode set and codes per version from the client switch, IDA-verified).
- FR-14: Successful warp uses the existing character warp path (same mechanism the mystic door / portal flows use) so map membership, spawn packets, and transition invariants are preserved.

### 4.4 Character-data codec

- FR-15: `libs/atlas-packet/character/data.go` `CharacterData` gains regular (5) and VIP (10) teleport-map fields; `encodeTeleports`/`decodeTeleports` MUST use them, preserving the existing version gate (VIP block only when `(GMS && MajorVersion() > 28) || JMS`) and defaulting absent entries to `EmptyMapId`.
- FR-16: atlas-channel's character-data writer (`services/atlas-channel/atlas.com/channel/socket/writer/character_data.go`) MUST populate those fields from atlas-character at field-enter/character-load time.

### 4.5 Version and configuration coverage

- FR-17: Handler entries for both serverbound ops (each with `LoggedInValidator` — a validator-less entry is silently dropped) and the writer entry for `MAP_TRANSFER_RESULT` MUST be added to the seed templates for **all** supported tenant versions: gms_83, gms_84, gms_87, gms_92, gms_95, jms.
- FR-18: `MAP_TRANSFER_RESULT` is a mode-byte result packet: its mode values MUST be resolved from the per-version `operations` table (`WithResolvedCode`), never hard-coded, and the table keys MUST be populated for every version's template (modes are version-dependent; the missing-operations-table failure mode is a known crash class).
- FR-19: Existing live tenants do not pick up seed-template changes; the rollout MUST include patching live tenant configurations and restarting atlas-channel (documented in the plan as an explicit deploy step).

### 4.6 Packet verification (full-fixture)

- FR-20: All three operations get packet-audit treatment per `docs/packets/audits/VERIFYING_A_PACKET.md`: IDA-derived read/write order, byte-fixture tests with `packet-audit:verify` markers, pinned evidence records, and matrix regeneration — for every IDB-backed version (v83, v84, v87, v95, jms). gms_92 has no IDB; its opcodes/modes are seeded from the template lineage and its cells remain at the matrix's unverified designation (this is the one version-scoped exception, consistent with prior tasks).
- FR-21: STATUS.md rows `USE_TELEPORT_ROCK` (v83 0x054), `TROCK_ADD_MAP` (v83 0x066), and `MAP_TRANSFER_RESULT` (v83 0x02A) promote from ❌ on the verified versions.

## 5. API Surface

New JSON:API resource in atlas-character (exact naming finalized in design; shape below is the contract intent):

- `GET /characters/{characterId}/teleport-rock-maps` — both lists for the character.
  - Response: resource with attributes `{ regular: [mapId…≤5], vip: [mapId…≤10] }` (empty slots omitted; wire-level padding to `EmptyMapId` is the codec's job, not the API's).
- Mutations ride Kafka commands (game flow), not REST: `COMMAND_TELEPORT_ROCK` topic (or equivalent) with `ADD_MAP` / `REMOVE_MAP` bodies emitted by atlas-channel; atlas-character validates, persists, and emits a status event that atlas-channel projects into `MAP_TRANSFER_RESULT`.
- Character-data enrichment: the existing character REST model (or the dedicated GET above) is extended so atlas-channel can populate the codec fields in one fetch at load time — design decides between embedding vs separate call, following how `saved_location` style data reaches the channel today.
- Error cases surface as status events with failure reasons (list full, duplicate map, map not present, map barred), which atlas-channel maps to `MAP_TRANSFER_RESULT` error modes.

Warp execution reuses the existing saga action for character warp plus `DestroyAsset` (quantity 1) for the regular rock — same pattern as `character_cash_item_use.go`'s field-effect saga.

## 6. Data Model

New dedicated GORM domain in atlas-character (pattern: `saved_location`, but list-shaped):

```
teleport_rock_maps
  id            uuid PK
  tenant_id     uuid  not null  (uniqueIndex part 1)
  character_id  uint32 not null (uniqueIndex part 2)
  list_type     string not null (uniqueIndex part 3)  -- "regular" | "vip"
  slot          int    not null (uniqueIndex part 4)  -- 0-based position
  map_id        _map.Id not null
```

- Unique on `(tenant_id, character_id, list_type, slot)`; capacity (5/10) enforced in the processor, not the schema.
- Immutable model + Builder, Processor Interface/Impl (`NewProcessor(l, ctx)`), pure vs `AndEmit` variants, per project patterns.
- AutoMigrate migration; deletion hook when a character is deleted (same lifecycle as `saved_location` cleanup).
- No backfill needed — feature is net-new; existing characters start with empty lists.

## 7. Service Impact

- **atlas-character** — new `teleport rock maps` domain (entity, model, builder, processor, provider, REST resource, administrator), Kafka consumer for add/remove commands, status event producer, character-deletion cleanup, mock processor update.
- **atlas-channel** — serverbound handlers for `USE_TELEPORT_ROCK` and `TROCK_ADD_MAP`; route the `CashSlotItemType(12)` branch of `character_cash_item_use.go` into the same use-flow; `MAP_TRANSFER_RESULT` writer (operations-table-resolved modes); character-data writer populates the new codec fields; consumer projecting teleport-rock status events to the client.
- **libs/atlas-packet** — serverbound decoders for the two request ops (per-version layouts), clientbound `MapTransferResult` model + codec, `CharacterData` teleport fields + codec change, byte-fixture tests for all of the above.
- **atlas-saga / orchestrator** — reuse existing warp + `DestroyAsset` actions; only if design finds no suitable warp action does a new step type get added.
- **Seed templates + live tenant config** — handler/writer/operations entries for gms_83/84/87/92/95 + jms; live-tenant patch + channel restart at rollout.
- **atlas-data** — no change expected (classification already present); design confirms whether any spec field of 2320000 needs reading.
- **docs/packets** — audit artifacts, evidence records, STATUS.md regeneration.

## 8. Non-Functional Requirements

- **Multi-tenancy**: all persistence tenant-scoped (`tenant_id` in every row/index); tenant from `tenant.MustFromContext(ctx)`; version gates via `tenant.Model` region/major-version, following the `>28` gate already in the codec.
- **Atomicity**: add/remove is a single-row-set mutation inside one transaction; warp+consume ordering is saga-enforced so a failed warp never consumes.
- **Performance**: list reads happen at character load and per rock use — single indexed lookup; no caching layer needed.
- **Observability**: handlers log op + decoded request at debug (existing pattern); rejections log at warn with reason; no new metrics required.
- **Safety**: no hard-coded mode bytes (config-resolved only); every handler entry has a validator; new Redis usage (none expected) would have to route through `libs/atlas-redis`.

## 9. Open Questions

Resolved-by-design (not blocking, but MUST be IDA/WZ-verified before implementation — no values may be invented):

1. **Exact request layouts** of `CWvsContext::SendMapTransferItemUseRequest` and `CWvsContext::SendMapTransferRequest` per version (field order, name-string encoding, updateTime placement, and how the regular-vs-VIP discriminator is encoded).
2. **Field-limit semantics**: which `fieldLimit` bit(s) bar saving/warping, and whether save-map and warp-target use the same bit — verify against WZ `Map.img` info + client checks.
3. **Regular-rock region restriction**: the precise "nearby/same-continent" rule the v83-era client+server enforce (and what the client itself pre-filters), so server validation is faithful rather than Cosmic-copied.
4. **`MAP_TRANSFER_RESULT` mode set** per version (list-refresh modes vs error codes) from each client's `OnMapTransferResult` switch, feeding the per-version operations tables.
5. **Teleport Coke (5040001)**: assumed to behave exactly as 5040000 (regular-list cash rock). Verify in IDA that the client treats all 5040xxx as regular and only 5041xxx as VIP.
6. **Warp-to-player cross-channel behavior**: whether a regular/VIP rock can target a player on another channel (and if so, whether the result is a channel-change flow or a rejection). Verify in client/IDA before designing the lookup.

## 10. Acceptance Criteria

- [ ] Regular Teleport Rock (2320000) warps to a saved map and to a player by name on a v83 tenant; the item is consumed exactly 1 on success and not at all on failure.
- [ ] The Teleport Rock (5040000), Teleport Coke (5040001), and VIP Teleport Rock (5041000) warp without being destroyed; VIP uses the 10-slot list.
- [ ] Add-map and remove-map persist across relog and channel change; capacity (5/10), duplicate, and barred-map rejections behave faithfully and surface the correct client message.
- [ ] Character-data packet shows the real saved lists (no longer hardcoded `EmptyMapId`) on all supported versions; empty slots still encode `EmptyMapId`.
- [ ] All rejection paths emit `MAP_TRANSFER_RESULT` with IDA-verified mode/code values resolved from per-version operations tables (no literals).
- [ ] Byte-fixture tests with `packet-audit:verify` markers exist for `USE_TELEPORT_ROCK`, `TROCK_ADD_MAP`, and `MAP_TRANSFER_RESULT` on v83, v84, v87, v95, jms; evidence pinned; STATUS.md regenerated with those cells promoted; `packet-audit` matrix/operations `--check` exit 0.
- [ ] Seed templates updated for gms_83/84/87/92/95 + jms (handlers with validators, writer, operations keys); live-tenant patch procedure documented in the plan.
- [ ] `go test -race ./...`, `go vet ./...`, `go build ./...` clean in every changed module; `docker buildx bake` clean for atlas-character and atlas-channel; `tools/redis-key-guard.sh` clean.
- [ ] Code review (plan-adherence + backend-guidelines) run before PR.
