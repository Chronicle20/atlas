# Atlas-Packet Improvements — Context

Last Updated: 2026-03-11

## Key Files

### atlas-packet Library
- `libs/atlas-packet/packet.go` — Packet interface (Operation + Stringer + Encoder + Decoder)
- `libs/atlas-packet/character/effect.go` — 13 effect types (EffectSimple, EffectSimpleForeign, EffectSkillAffected, EffectPet, EffectWithId, EffectWithMessage, EffectProtectOnDie, EffectIncDecHP, EffectShowInfo, EffectLotteryUse, EffectItemMaker, EffectUpgradeTomb, EffectIncubatorUse)
- `libs/atlas-packet/character/effect_skill_use.go` — EffectSkillUse (no-op Decode, conditional bools)
- `libs/atlas-packet/character/effect_quest.go` — EffectQuest (no-op Decode, conditional rewards)
- `libs/atlas-packet/character/effect_foreign.go` — Generic EffectForeign wrapper (characterId + []byte)
- `libs/atlas-packet/inventory/change_batch.go` — ChangeBatch (pre-encoded [][]byte entries)
- `libs/atlas-packet/inventory/change.go` — Structured types: Add, QuantityUpdate, ChangeMove, Remove (already have Encode+Decode)
- `libs/atlas-packet/model/asset.go` — Asset model with full Encode/Decode
- `libs/atlas-packet/test/roundtrip.go` — RoundTrip test helper (validates unconsumed bytes)

### Service Adapters
- `services/atlas-channel/atlas.com/channel/socket/writer/writer.go` — getCode (duplicated)
- `services/atlas-login/atlas.com/login/socket/writer/writer.go` — getCode (duplicated)
- `services/atlas-channel/atlas.com/channel/socket/writer/character_effect.go` — 512 lines, 40+ functions, getCharacterEffect resolver
- `services/atlas-channel/atlas.com/channel/socket/writer/character_inventory_change.go` — InventoryChangeWriter type, 4 entry builders, uses response.Writer directly
- `services/atlas-channel/atlas.com/channel/socket/model/asset.go` — NewAssetWriter, converts service asset to packet Asset

### No-Op Decode Files (17 packets, 10 files)
- `libs/atlas-packet/character/attack_writer.go` — AttackWriter (flags not on wire)
- `libs/atlas-packet/character/effect_skill_use.go` — EffectSkillUse (conditional bools)
- `libs/atlas-packet/character/effect_quest.go` — EffectQuest (solvable)
- `libs/atlas-packet/character/effect_foreign.go` — EffectForeign (generic wrapper)
- `libs/atlas-packet/character/info.go` — CharacterInfo (pre-encoded composite)
- `libs/atlas-packet/character/spawn.go` — CharacterSpawn (pre-encoded composite)
- `libs/atlas-packet/field/set_field.go` — SetField (pre-encoded composite)
- `libs/atlas-packet/cash/shop_open.go` — CashShopOpen (pre-encoded composite)
- `libs/atlas-packet/cash/shop_inventory.go` — 3 structs (solvable)
- `libs/atlas-packet/cash/shop_item_moved.go` — 2 structs (solvable)
- `libs/atlas-packet/storage/update_assets.go` — UpdateAssets (opaque entries)
- `libs/atlas-packet/storage/show.go` — Show (opaque entries)
- `libs/atlas-packet/note/display.go` — Display (opaque entries)
- `libs/atlas-packet/messenger/add_writer.go` — AddW (opaque avatar bytes)
- `libs/atlas-packet/messenger/update_writer.go` — UpdateW (opaque avatar bytes)
- `libs/atlas-packet/interaction/interaction_writer.go` — 2 structs (no length prefix)

## Code Resolution Patterns

### atlas-login: Parametrized getCode
```go
getCode(l)(requester, code, codeProperty, options)
```
10 distinct (requester, codeProperty) pairs across 9 writer files.
Properties used: `"codes"`, `"failedReasonCodes"`, `"modes"`

### atlas-channel: Hardcoded Domain Getters
21 dedicated functions, all with hardcoded options keys:
- 17 use `"operations"` key
- 2 use `"errors"` key
- 1 uses `"enterError"` key
- 1 uses `"modes"` key

Heaviest: `getCharacterEffect` with 33 usages in character_effect.go alone.

## Design Decisions

### D1: ResolveCode Location
Put in `libs/atlas-packet/resolve.go`. Atlas-packet is already imported by both services, avoids new dependency.

### D2: Foreign Effect Strategy
Create dedicated `XxxForeign` structs rather than generic wrapping. Pattern matches the existing `EffectSimpleForeign` which already works well. Each struct is small (characterId + 2-5 fields).

### D3: Inventory Change Entry Type
Use the existing structured types from `inventory/change.go` (Add, QuantityUpdate, ChangeMove, Remove). They already implement Encode and Decode. `ChangeBatch` wraps them and calls their Encode internally rather than receiving pre-encoded bytes.

### D4: Asset Encoding for Inventory Add
Service layer converts `asset.Model` to `packetmodel.Asset` (already happens in `socket/model/asset.go`). The packet `Asset` gets passed to `inventory.Add` struct. Atlas-packet's `Add.Encode()` calls `Asset.Encode()` internally.

### D5: EffectSkillUse Decode
Leave as no-op. The conditional bools (`isBerserk`, `isDragonFury`, `isMonsterMagnet`) determine whether extra bytes are written, but aren't self-describing on the wire. The Decode can't know which optionals are present without skill-ID domain knowledge that belongs in atlas-constants, not atlas-packet.

### D6: AttackWriter Decode
Leave as no-op. `isMesoExplosion`, `isStrafe`, `hasKeydown` flags affect wire structure but aren't encoded directly. Damage counts are derivable from header bits but the full reconstruction is complex and this packet is server-send-only.

## Previous Work
- Writer-packet-extraction (COMPLETE): 143/143 tasks, all 130+ packets extracted to atlas-packet
- Dev docs: `dev/active/writer-packet-extraction/`
- Branch: `merchant-service`
