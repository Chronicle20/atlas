# Atlas-Packet Improvements — Tasks

Last Updated: 2026-03-12

## Phase 1: Extract Shared Code Resolution Utility [8/8] COMPLETE

### 1.1 Create ResolveCode utility
- [x] Create `libs/atlas-packet/resolve.go` with `ResolveCode(l, options, property, key) byte` — S
- [x] Add test for ResolveCode: valid key, missing property, missing key, wrong type — S

### 1.2 Migrate atlas-login
- [x] Replace `getCode` in `atlas-login/socket/writer/writer.go` with import of shared `ResolveCode` — S
- [x] Update all 9 login writer files that call `getCode` to use new signature — S
- [x] Build + test atlas-login — S

### 1.3 Migrate atlas-channel
- [x] Replace `getCode` in `atlas-channel/socket/writer/writer.go` with import of shared `ResolveCode` — S
- [x] Replace all 21 domain-specific getter functions across 17 writer files with calls to `ResolveCode` — M
- [x] Build + test atlas-channel — S

## Phase 2: Add Concrete Foreign Effect Structs [18/18] COMPLETE

### 2.1 Create foreign structs in atlas-packet
- [x] `EffectSkillUseForeign` — characterId + mode + skillId + characterLevel + skillLevel + conditional bools + Encode + Decode (no-op) — M
- [x] `EffectSkillAffectedForeign` — characterId + mode + skillId + skillLevel + Encode + Decode — S
- [x] `EffectQuestForeign` — characterId + mode + rewards/message + nEffect + Encode + Decode — S
- [x] `EffectPetForeign` — characterId + mode + effectType + petIndex + Encode + Decode — S
- [x] `EffectWithIdForeign` — characterId + mode + id + Encode + Decode — S
- [x] `EffectProtectOnDieForeign` — characterId + mode + safetyCharm + usesRemaining + days + itemId + Encode + Decode — S
- [x] `EffectIncDecHPForeign` — characterId + mode + delta + Encode + Decode — S
- [x] `EffectWithMessageForeign` — characterId + mode + message + Encode + Decode — S
- [x] `EffectShowInfoForeign` — characterId + mode + path + Encode + Decode — S
- [x] `EffectLotteryUseForeign` — characterId + mode + itemId + success + message + Encode + Decode — S
- [x] `EffectItemMakerForeign` — characterId + mode + state + Encode + Decode — S
- [x] `EffectUpgradeTombForeign` — characterId + mode + usesRemaining + Encode + Decode — S
- [x] `EffectIncubatorUseForeign` — characterId + mode + itemId + message + Encode + Decode — S

### 2.2 Add round-trip tests for all foreign effects
- [x] Round-trip tests for all 13 new foreign structs (verify Encode→Decode identity + no unconsumed bytes) — M

### 2.3 Byte-compare verification
- [x] Write test that verifies new foreign struct Encode produces identical bytes to old EffectForeign(characterId, innerBytes) pattern — M

### 2.4 Update service adapter + cleanup
- [x] Update `character_effect.go` in atlas-channel to use new foreign structs directly (eliminate encode-then-wrap) — M
- [x] Delete `libs/atlas-packet/character/effect_foreign.go` (generic EffectForeign wrapper) — S
- [x] Build + test atlas-channel + atlas-packet — S

## Phase 3: Move Inventory Change Entry Encoding into atlas-packet [7/7] COMPLETE

### 3.1 Design structured batch entry
- [x] Define `ChangeEntry` interface in `inventory/change_entry.go` that Add/QuantityUpdate/Move/Remove implement — S
- [x] Ensure each ChangeEntry type exposes `EntryAddMov() int8` for equipment tracking — S

### 3.2 Refactor ChangeBatch
- [x] Modify `ChangeBatch` to accept `[]ChangeEntry` instead of `[][]byte` — M
- [x] ChangeBatch.Encode iterates entries, calls each entry's Encode, writes bytes — S
- [x] Implement `ChangeBatch.Decode` — reads entry count, decodes each by mode byte dispatch — M

### 3.3 Update service adapter + callers
- [x] Update `character_inventory_change.go` in atlas-channel to build structured entries and pass to ChangeBatch — M
- [x] Update `kafka/consumer/asset/consumer.go` — 7 call sites migrated from old BodyWriter functions — M

### 3.4 Verification
- [x] Round-trip tests for ChangeBatch with each entry type (Add, QuantityUpdate, Move, Remove, multi-entry) — M
- [x] Build + test atlas-channel + atlas-packet — S

## Phase 4: Implement Decode for Solvable No-Op Packets [5/5] COMPLETE

### 4.1 Cash shop packets
- [x] `CashShopGifts` — Decode: mode + count(0) — S
- [x] `CashItemMovedToCashInventory` — Decode: mode + CashInventoryItem — S
- [x] `CashShopInventory` — Decode: mode + count + items + slots + characterSlots — S
- [x] `CashShopPurchaseSuccess` — Decode: mode + CashInventoryItem — S

### 4.1b Already done in Phase 2
- [x] `EffectQuest` — Decode: mode + conditional rewards/message — S

### 4.3 Round-trip tests
- [x] Round-trip tests for CashShopInventory, CashShopPurchaseSuccess, CashShopGifts, CashItemMovedToCashInventory — M

### 4.5 Verification
- [x] Build + test atlas-packet — S

## Phase 5: Structural Refactor for Pre-Encoded Byte Packets [14/14] COMPLETE

### 5.1 Messenger AddW / UpdateW
- [x] Verify `model.Avatar` has working Encode/Decode — S
- [x] Replace `avatarBytes []byte` with `avatar model.Avatar` in `AddW` and `UpdateW` — M
- [x] Implement Decode for `AddW` and `UpdateW` — S
- [x] Update `messenger_operation.go` in atlas-channel to pass `model.Avatar` struct — M
- [x] Round-trip tests for `AddW` and `UpdateW` — S

### 5.2 Storage Show / UpdateAssets
- [x] Replace `assetEntryBytes [][]byte` with `assets []model.Asset` in `Show` and `UpdateAssets` — M
- [x] Implement Decode for `Show` and `UpdateAssets` — S
- [x] Update `storage_operation.go` in atlas-channel to pass `[]model.Asset` — M
- [x] Round-trip tests for `Show` and `UpdateAssets` — S

### 5.3 Note Display
- [x] Define `NoteEntry` struct in `libs/atlas-packet/note/entry.go` — S
- [x] Replace `noteEntryBytes [][]byte` with `[]NoteEntry` in `Display`, implement Decode — M
- [x] Update `note_operation.go` in atlas-channel to pass `[]NoteEntry` — M
- [x] Round-trip test for `Display` — S

### 5.4 CashItemMovedToInventory
- [x] Replace `assetBytes []byte` with `asset model.Asset` in `CashItemMovedToInventory`, implement Decode — S
- [x] Update `cash_shop_operation.go` in atlas-channel to pass `model.Asset` — S
- [x] Round-trip test for `CashItemMovedToInventory` — S

### 5.5 Verification
- [x] Build + test atlas-packet — S
- [x] Build + test atlas-channel — S
- [x] Full Docker build: atlas-channel — S

## Cross-Phase Verification (Phases 1-4) [2/2] COMPLETE

- [x] Full Docker build: atlas-login — S
- [x] Full Docker build: atlas-channel — S
