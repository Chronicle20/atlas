# Atlas-Packet Improvements

Last Updated: 2026-03-12

## Executive Summary

Four targeted improvements to the atlas-packet library and its service adapters. These follow on from the completed writer-packet-extraction project and address remaining design issues: split encoding ownership, untestable foreign effects, duplicated code resolution, and unnecessary no-op Decode methods.

**Phases:**
1. Extract shared code resolution utility (low risk, eliminates ~400 lines of boilerplate)
2. Add concrete foreign effect structs (medium effort, enables round-trip testing for all effects)
3. Move inventory change entry encoding into atlas-packet (medium effort, eliminates split encoding)
4. Implement Decode for solvable no-op packets (low effort per packet, improves test coverage)

## Current State Analysis

### Code Resolution Duplication
- `getCode` is copy-pasted in both `atlas-login/socket/writer/writer.go` and `atlas-channel/socket/writer/writer.go` with identical logic
- `getCharacterEffect` and 20 other domain-specific getters in atlas-channel hardcode `options["operations"]` and duplicate the same type-assertion + fallback logic
- All fall back to byte `99` on misconfiguration ("likely cause a client crash")
- Total: 22 separate resolver functions across 2 services doing the same thing

### Foreign Effect Pattern
- `EffectSimpleForeign` is the only concrete foreign struct (characterId + mode)
- All other foreign effects use the generic `EffectForeign` wrapper with pre-encoded `[]byte` inner bytes
- 13 effect types go through `EffectForeign`, making them non-round-trippable
- `character_effect.go` in atlas-channel is 512 lines of near-identical adapter boilerplate: ~20 effect types x 2 variants (local + foreign)

### Inventory Change Split Encoding
- Service layer (`character_inventory_change.go`) uses `response.Writer` directly to encode individual change entries (Add/Move/Remove/QuantityUpdate)
- `ChangeBatch` in atlas-packet receives `[][]byte` pre-encoded entries and wraps them
- atlas-packet already has structured `Add`, `QuantityUpdate`, `ChangeMove`, `Remove` types in `inventory/change.go` with proper Encode/Decode
- The service bypasses these and writes directly, creating a parallel encoding path

### No-Op Decode Methods
- 17 packets across 10 files have no-op Decode
- 7 are trivially solvable (all data on wire)
- 5 are inherently undecodable (pre-encoded composite service data)
- 3 have opaque entry arrays (decodable structure, opaque entries)
- 2 have conditional logic barriers (flags not on wire)

## Proposed Future State

### After Phase 1 (Code Resolution)
- Single `ResolveCode(options, property, key)` utility in atlas-packet (or a shared package)
- Service adapters call the shared utility instead of maintaining local resolvers
- Effects adapter functions use the shared resolver, eliminating per-domain getter boilerplate
- atlas-login's `getCode` and atlas-channel's 21 getters all replaced

### After Phase 2 (Foreign Effects)
- Every effect type has a dedicated `XxxForeign` struct with characterId + typed fields
- All foreign effects are fully round-trippable via Encode/Decode
- `EffectForeign` (generic wrapper) deleted
- `character_effect.go` adapter in atlas-channel reduced from 512 to ~200 lines by using the shared code resolver and simpler delegation

### After Phase 3 (Inventory Entries)
- `ChangeBatch` accepts structured entry types instead of `[][]byte`
- Service layer builds `[]inventory.ChangeEntry` (an interface or sum type) and passes to atlas-packet
- All inventory change encoding lives in atlas-packet, service layer is a pure thin adapter
- Round-trip tests possible for complete inventory change packets

### After Phase 4 (No-Op Decodes) — DONE
- 5 fixed-format cash shop packets gained working Decode + round-trip tests
- 7 inherently undecodable packets remain as documented no-ops
- 6 packets with pre-encoded `[]byte` fields identified as needing structural refactor (Phase 5)

### After Phase 5 (Structural Refactor for Pre-Encoded Bytes)
- 6 remaining pre-encoded byte packets converted to hold typed models
- Messenger AddW/UpdateW hold `model.Avatar` instead of `avatarBytes []byte`
- Storage Show/UpdateAssets hold `[]model.Asset` instead of `assetEntryBytes [][]byte`
- Note Display holds `[]NoteEntry` instead of `noteEntryBytes [][]byte`
- CashItemMovedToInventory holds `model.Asset` instead of `assetBytes []byte`
- All 6 gain working Decode + round-trip tests
- Total no-ops reduced from 17 to 7 (inherently undecodable only)

## Implementation Phases

### Phase 1: Extract Shared Code Resolution Utility

**Goal:** Eliminate duplicated code resolution logic across both services.

**Approach:** Create a `ResolveCode` function in atlas-packet that encapsulates the options-map lookup + type assertion + error logging pattern. Both services' writer packages call this instead of maintaining local copies.

**Key decisions:**
- Location: `libs/atlas-packet/resolve.go` — atlas-packet already has the right dependency position
- Signature: `func ResolveCode(l logrus.FieldLogger, options map[string]interface{}, property string, key string) byte`
- Keeps the same runtime behavior (log error + return 99 on missing key)
- Domain-specific mode types (`CharacterEffectMode`, `FieldEffectMode`, etc.) stay in their service adapter files as type aliases for string — they just call the shared resolver

**Files changed:**
- New: `libs/atlas-packet/resolve.go`
- Modified: `services/atlas-login/atlas.com/login/socket/writer/writer.go` (remove getCode, use shared)
- Modified: `services/atlas-channel/atlas.com/channel/socket/writer/writer.go` (remove getCode, use shared)
- Modified: 17 atlas-channel writer files that define domain-specific getters (replace with shared calls)

### Phase 2: Add Concrete Foreign Effect Structs

**Goal:** Replace the generic `EffectForeign` wrapper with dedicated foreign structs for each effect type, enabling round-trip testing.

**Approach:** For each of the 13 effect types currently using `EffectForeign`, create an `XxxForeign` struct (like the existing `EffectSimpleForeign`) with `characterId` + the same typed fields as the local version. Implement Encode (write characterId + fields) and Decode (read characterId + fields).

**Effect types needing foreign structs (13):**
1. `EffectSkillUseForeign` — characterId + mode + skillId + characterLevel + skillLevel + conditional bools
2. `EffectSkillAffectedForeign` — characterId + mode + skillId + skillLevel
3. `EffectQuestForeign` — characterId + mode + rewards/message + nEffect
4. `EffectPetForeign` — characterId + mode + effectType + petIndex
5. `EffectWithIdForeign` — characterId + mode + id (used for SkillSpecial, BuffItem, ConsumeEffect)
6. `EffectProtectOnDieForeign` — characterId + mode + safetyCharm + usesRemaining + days + conditional itemId
7. `EffectIncDecHPForeign` — characterId + mode + delta
8. `EffectWithMessageForeign` — characterId + mode + message (ShowIntro, Reserved, Battlefield, PlaySound)
9. `EffectShowInfoForeign` — characterId + mode + path
10. `EffectLotteryUseForeign` — characterId + mode + itemId + success + conditional message
11. `EffectItemMakerForeign` — characterId + mode + state
12. `EffectUpgradeTombForeign` — characterId + mode + usesRemaining
13. `EffectIncubatorUseForeign` — characterId + mode + itemId + message

**After creating foreign structs**, update `character_effect.go` in atlas-channel to use them directly instead of the encode-then-wrap pattern. Combined with Phase 1's shared resolver, this should cut the file roughly in half.

**Files changed:**
- Modified: `libs/atlas-packet/character/effect.go` (add foreign structs for types already there)
- New or modified: `libs/atlas-packet/character/effect_skill_use.go` (add EffectSkillUseForeign)
- New or modified: `libs/atlas-packet/character/effect_quest.go` (add EffectQuestForeign)
- Deleted: `libs/atlas-packet/character/effect_foreign.go` (generic wrapper removed)
- Modified: `services/atlas-channel/atlas.com/channel/socket/writer/character_effect.go` (use new foreign structs)

### Phase 3: Move Inventory Change Entry Encoding into atlas-packet

**Goal:** Eliminate split encoding by having atlas-packet own all inventory change encoding.

**Approach:** The structured types already exist in `libs/atlas-packet/inventory/change.go` (`Add`, `QuantityUpdate`, `ChangeMove`, `Remove`). Modify `ChangeBatch` to accept a slice of these instead of `[][]byte`. The service layer builds the structured entries and passes them down.

**Key challenge:** `InventoryAddBodyWriter` currently accepts an `itemWriter model.Operator[*response.Writer]` for asset encoding. The asset writing needs to be callable from atlas-packet. Since `model/asset.go` already has `Asset.Encode()` in atlas-packet, the service can pass the packet-layer `Asset` model rather than a writer function.

**Steps:**
1. Define `ChangeEntry` interface or use existing concrete types in `inventory/change.go`
2. Modify `ChangeBatch` to accept `[]ChangeEntry` and encode each entry internally
3. Update service-layer `CharacterInventoryChangeBody` to build structured entries
4. Add round-trip tests for `ChangeBatch` with structured entries
5. Remove `InventoryChangeWriter` type and the four `Inventory*BodyWriter` functions from the service

**Files changed:**
- Modified: `libs/atlas-packet/inventory/change_batch.go` (structured entries instead of `[][]byte`)
- Modified: `libs/atlas-packet/inventory/change.go` (ensure types are composable)
- Modified: `services/atlas-channel/atlas.com/channel/socket/writer/character_inventory_change.go` (thin adapter)
- Modified: `services/atlas-channel/atlas.com/channel/socket/model/asset.go` (may simplify)
- New: `libs/atlas-packet/inventory/change_batch_test.go` (round-trip tests)

### Phase 4: Implement Decode for Solvable No-Op Packets (PARTIALLY COMPLETE)

**Goal:** Reduce no-op Decode count and add round-trip tests.

**Completed — fixed-format packets (5):**
1. ~~`CashItemMovedToCashInventory`~~ — DONE: mode + CashInventoryItem
2. ~~`CashShopInventory`~~ — DONE: mode + count + items + slots + characterSlots
3. ~~`CashShopPurchaseSuccess`~~ — DONE: mode + CashInventoryItem
4. ~~`CashShopGifts`~~ — DONE: mode + count(0)
5. ~~`EffectQuest`~~ — DONE in Phase 2: mode + conditional rewards/message

**Leave as no-op with documentation (7 — inherently undecodable):**
- `SetField`, `CashShopOpen`, `CharacterInfo`, `CharacterSpawn` — inherently undecodable (pre-encoded service composites)
- `AttackWriter` — flags not on wire
- `EffectSkillUse` — conditional bools not self-describing (foreign variant has no-op Decode by design)
- `InteractionEnter`, `InteractionEnterResultSuccess` — no length prefix on wire

**Files changed:** `libs/atlas-packet/cash/shop_inventory.go`, `libs/atlas-packet/cash/shop_item_moved.go`, new test files

### Phase 5: Structural Refactor for Pre-Encoded Byte Packets (NEW — NOT STARTED)

**Goal:** Apply the same pattern used in Phase 3 (ChangeBatch `[][]byte` → `[]ChangeEntry`) to the remaining 6 packets that hold pre-encoded `[]byte` fields from the service layer, enabling full round-trip Decode and testing.

**Pattern:** Each packet currently accepts opaque bytes because the service layer pre-encodes models before passing them down. The fix is to have the packet struct hold the typed model directly and call its Encode internally — the service layer passes structured data instead of bytes.

**5.1 Messenger AddW / UpdateW (2 packets)**
- Current: `avatarBytes []byte` — pre-encoded `model.Avatar` from service
- Target: Replace `avatarBytes []byte` with `avatar model.Avatar`
- Encode calls `avatar.Encode()` internally instead of writing raw bytes
- Decode calls `avatar.Decode()` to reconstruct
- Prerequisite: `model.Avatar` must have working Encode/Decode (verify)
- Service callers in `atlas-channel/socket/writer/messenger_operation.go` updated to pass `model.Avatar` struct

**5.2 Storage Show / UpdateAssets (2 packets)**
- Current: `assetEntryBytes [][]byte` — pre-encoded `model.Asset` entries from service
- Target: Replace `assetEntryBytes [][]byte` with `assets []model.Asset`
- Encode iterates assets and calls each `asset.Encode()` internally
- Decode reads count, then decodes each `model.Asset` from wire
- Prerequisite: `model.Asset` already has Encode/Decode
- Service callers in `atlas-channel/socket/writer/storage_operation.go` updated to pass `[]model.Asset`

**5.3 Note Display (1 packet)**
- Current: `noteEntryBytes [][]byte` — pre-encoded note entries from service
- Target: Define a `NoteEntry` struct with typed fields, replace `[][]byte` with `[]NoteEntry`
- Need to identify the wire format of note entries from the service-layer encoder
- Service callers in `atlas-channel/socket/writer/note_operation.go` updated

**5.4 CashItemMovedToInventory (1 packet)**
- Current: `assetBytes []byte` — pre-encoded `model.Asset` from service
- Target: Replace `assetBytes []byte` with `asset model.Asset`
- Service callers in `atlas-channel/socket/writer/cash_shop_operation.go` updated

**Key differences from Phase 3:**
- Phase 3 defined a new `ChangeEntry` interface because there were 4 polymorphic entry types. Here, most packets just need a single typed model substitution.
- Avatar and Asset models already exist in atlas-packet with Encode/Decode. NoteEntry needs a new struct.
- Service adapter changes are straightforward: pass the model instead of calling `model.Encode()` first.

**Files changed per sub-phase:**
- 5.1: `libs/atlas-packet/messenger/add_writer.go`, `update_writer.go` + service `messenger_operation.go`
- 5.2: `libs/atlas-packet/storage/show.go`, `update_assets.go` + service `storage_operation.go`
- 5.3: `libs/atlas-packet/note/display.go` + new `note/entry.go` + service `note_operation.go`
- 5.4: `libs/atlas-packet/cash/shop_item_moved.go` + service `cash_shop_operation.go`
- New test files for each sub-phase

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Phase 3 breaks inventory change wire format | Medium | High | Round-trip test existing encoding before refactoring; byte-compare old vs new output |
| Phase 1 resolver changes miss an edge case in options map structure | Low | Medium | Options map structure is consistent across all usages; test with real config |
| Phase 2 foreign effect wire format doesn't match encode-then-wrap output | Low | High | Write tests that compare new foreign struct Encode output vs old EffectForeign output byte-for-byte |
| Breaking atlas-login or atlas-channel Docker builds | Medium | Medium | Build + test both services after each phase |

## Success Metrics

**Phases 1-4 (COMPLETE):**
- [x] All 17 atlas-channel writer files with local getters reduced to calling shared resolver
- [x] `getCode` removed from both service `writer.go` files
- [x] `EffectForeign` generic wrapper deleted, all 13 foreign effects round-trip tested
- [x] `character_effect.go` reduced from 512 to ~200 lines
- [x] `ChangeBatch` accepts structured `ChangeEntry` types, round-trip tested
- [x] 5 cash shop / effect packets gained working Decode + round-trip tests
- [x] All builds pass: `atlas-login`, `atlas-channel`, `atlas-packet` tests
- [x] Docker builds verified for both services

**Phase 5 (TODO):**
- [ ] 6 pre-encoded byte packets converted to typed models
- [ ] All 6 gain working Decode + round-trip tests
- [ ] No-op Decode count reduced from 17 to 7 (inherently undecodable only)
- [ ] Service adapter callers updated to pass typed models
- [ ] Docker builds verified

## Dependencies

- `libs/atlas-socket` — Encoder/Decoder interfaces (no changes needed)
- `libs/atlas-packet` — primary target
- `services/atlas-login` — Phase 1 only
- `services/atlas-channel` — Phases 1-3
- `atlas-constants` — not needed (skill identification stays in service layer)
