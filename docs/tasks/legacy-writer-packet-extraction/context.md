# Writer Packet Extraction - Context

Last Updated: 2026-03-10

## Key Files

### Core Interface
- `libs/atlas-packet/packet.go` — Packet interface (Operation, String, Encode, Decode)

### Atlas-Packet Library Structure
- `libs/atlas-packet/` — 29 sub-packages, 195+ Go files
- `libs/atlas-packet/model/` — 20 shared model files
- `libs/atlas-packet/test/` — Test utilities (RoundTrip, CreateContext, Variants)
- `libs/atlas-packet/go.mod` — Module: `github.com/Chronicle20/atlas-packet`

### Login Writers (18 files)
- `services/atlas-login/atlas.com/login/socket/writer/*.go`
- `services/atlas-login/atlas.com/login/session/writer.go` (WriteHello)
- `services/atlas-login/atlas.com/login/configuration/tenant/socket/writer/rest.go`
- `services/atlas-login/atlas.com/login/main.go` (writer registration at lines 137-162)

### Channel Writers (91 files)
- `services/atlas-channel/atlas.com/channel/socket/writer/*.go`
- `services/atlas-channel/atlas.com/channel/session/writer.go` (WriteHello)
- `services/atlas-channel/atlas.com/channel/configuration/tenant/socket/writer/rest.go`

### Service-Local Models Referenced by Writers
- `services/atlas-login/atlas.com/login/socket/model/` — avatar.go (maps to atlas-packet/model/Avatar), channel_load.go (simple DTO), world_recommendation.go (simple DTO)
- `services/atlas-channel/atlas.com/channel/socket/model/` — avatar.go, character.go (type aliases to atlas-packet types)
- `services/atlas-channel/atlas.com/channel/character/model.go` — full character aggregate
- `services/atlas-channel/atlas.com/channel/equipment/model.go` — equipment slots
- `services/atlas-channel/atlas.com/channel/inventory/model.go` — 5 compartments
- `services/atlas-channel/atlas.com/channel/character/skill/model.go` — skill with cooldowns
- `services/atlas-channel/atlas.com/channel/data/skill/` — external skill data processor (queries data service)

## Resolved Decisions

### 1. Attack Writers Extract with Pre-Computed Primitives (FINAL)
`character_attack_common.go` currently mixes serialization with game logic. The circular dependency comes from:
- `computeMasteryForWeapon()` → `getMasteryFromSkill()` → `atlas-channel/data/skill.NewProcessor()` (HTTP to data service)
- Bullet resolution → iterates `character.Inventory().Cash().Assets()` and `.Consumable().Assets()`
- Skill level lookup → iterates `character.Skills()`

But the packet only **writes** 3 derived primitives: `mastery byte`, `bulletItemId uint32`, `skillLevel byte`. Everything else comes from `model.AttackInfo` (already in atlas-packet) and simple character fields (id, level).

**Resolution**: Separate computation from serialization:
- **atlas-packet** gets a `CommonAttack` struct taking pre-computed primitives
- **atlas-channel** keeps `computeMasteryForWeapon()`, `getMasteryFromSkill()`, and bullet resolution as service-layer helpers called by the handler before constructing the packet

```go
// atlas-packet/character/attack/common.go — packet struct (primitives only)
type CommonAttack struct {
    characterId  uint32
    level        byte
    skillId      uint32
    skillLevel   byte           // pre-computed by service
    attackInfo   model.AttackInfo
    mastery      byte           // pre-computed by service
    bulletItemId uint32         // pre-computed by service
}
```

```go
// atlas-channel handler call site
mastery := computeMasteryForWeapon(l)(ctx)(weaponId, c.JobId(), skillId, c.Skills())
bulletId := resolveBulletItemId(c, ai)
skillLvl := lookupSkillLevel(c.Skills(), ai.SkillId())
p := attack.NewCommonAttack(c.Id(), c.Level(), ai.SkillId(), skillLvl, ai, mastery, bulletId)
session.Announce(l)(ctx)(wp)(writer.CharacterAttackMelee)(p.Encode)(s)
```

The 4 attack types (melee, ranged, magic, energy) share the `CommonAttack` struct but return different `Operation()` strings. They can be separate type aliases or thin wrapper structs.

**All 5 files extract to atlas-packet. Only computation helpers stay in atlas-channel.**

### 2. Multi-Operation Writer Patterns (FINAL)
Existing atlas-packet handler pattern: parent struct with mode byte → separate sub-operation structs in own files.

Applied to writers:

**login_status.go → 4 separate structs**
- AuthSuccess, AuthTemporaryBan, AuthPermanentBan, AuthLoginFailed
- Reason: Completely different payloads (AuthSuccess has ~20 fields, others have 1-3), registered as 4 separate writer constants. No shared structure.

**character_effect.go → 26 effect structs + EffectForeign wrapper**
- Each effect (LevelUp, SkillUse, SkillAffected, etc.) has different parameters
- All share a common mode byte written via options lookup
- Foreign variants prepend characterId then delegate to base effect encoder
- Pattern: `EffectForeign{characterId, effect EffectEncoder}` wrapper

**cash_shop_operation.go → separate structs per operation**
- Matches handler-side `cash/shop_operation*.go` pattern
- Each operation: `CashShopLoadInventory`, `CashShopPurchaseSuccess`, etc.

**character_inventory_change.go → single struct with composer pattern**
- Unique among writers: allows batching multiple changes in one packet
- Pattern: `InventoryChange{changes []InventoryChangeEntry}` where each entry has mode (add/quantity/move/remove) + mode-specific fields
- Preserves functional composer: `InventoryAddWriter()`, `InventoryQuantityUpdateWriter()`, etc.

**guild_operation.go → separate structs per operation (27+)**
**buddy_operation.go → separate structs per operation**
**party_operation.go → separate structs + shared PartyInfo model**
**hired_merchant_operation.go → separate structs per mode (9)**

### 3. Service Integration (FINAL)
The `session.Announce` call chain:
```go
session.Announce(l)(ctx)(wp)(writerName)(encoder)(session)
```
Where `writerName` is a string key and `encoder` is `packet.Encode`.

After extraction:
- `writerName` constant stays in service (it's the map lookup key for opcode resolution)
- The `packet.Encode` is provided by the atlas-packet struct's `.Encode` method
- Services construct the packet struct, pass `.Encode` to Announce
- Registration (`produceWriters()`, `getWriterProducer()`) unchanged

Example before:
```go
session.Announce(l)(ctx)(wp)(writer.CharacterList)(writer.CharacterListBody(chars, worldId, ...))(s)
```
Example after:
```go
p := character.NewList(chars, worldId, ...)
session.Announce(l)(ctx)(wp)(writer.CharacterList)(p.Encode)(s)
```

The writer constant can reference the packet's Operation(): `const CharacterList = character.List{}.Operation()`

### 4. writer.go and Configuration Stay in Services (FINAL)
- `writer.go` — `BodyFunc`/`Producer` type aliases + `getCode()` helper = socket infrastructure
- `configuration/tenant/socket/writer/rest.go` — `RestModel{OpCode, Writer, Options}` = deployment config
- Neither contains packet structure. Both stay.

### 5. Shared CharacterStatistics Model (FINAL)
Both `character_list.go` (login:101-173) and `set_field.go` (channel:383-456) write identical stat blocks.

Extract to `libs/atlas-packet/model/character_statistics.go`:
```go
type CharacterStatistics struct {
    Id              uint32
    Name            string    // 13-byte padded
    Gender          byte
    SkinColor       byte
    Face            uint32
    Hair            uint32
    PetIds          [3]uint64 // caller maps from service pet model
    Level           byte
    JobId           uint16
    Strength        uint16
    Dexterity       uint16
    Intelligence    uint16
    Luck            uint16
    Hp              uint16
    MaxHp           uint16
    Mp              uint16
    MaxMp           uint16
    Ap              uint16
    Sp              uint16    // or custom remaining SP handler
    Experience      uint32
    Fame            int16
    GachaponExp     uint32
    MapId           uint32
    SpawnPoint      byte
}
```

Encode/Decode methods handle region/version branching (GMS vs JMS, major version checks for GachaponExp, pet slots, padding).

### 6. Pet ID Divergence (FLAGGED)
- Login `character_list.go:124`: `w.WriteLong(character.Pets()[0].CashId())`
- Channel `set_field.go:406`: `w.WriteLong(uint64(character.Pets()[0].Id()))`

The CharacterStatistics model takes `PetIds [3]uint64` — caller provides the correct value.

**Action during implementation**: Verify whether channel's `pet.Id()` equals login's `pet.CashId()`. If they differ, channel likely has a bug and should use `CashId()` to match login. The protocol field represents the cash item serial number.

### 7. set_field.go Decomposition (FINAL)
461 lines, 17 helper functions. Decomposed into 12 atlas-packet components:

| Component | File | Lines in original | Complexity |
|-----------|------|-------------------|------------|
| WarpToMap | `field/warp_to_map.go` | 37-65 | Low — 4 fields, version branch |
| SetField orchestrator | `field/set_field.go` | 67-100 | Medium — calls sub-encoders |
| CharacterStatistics | `model/character_statistics.go` | 383-456 | Medium — shared, 20+ fields |
| InventoryInfo | `model/inventory_info.go` | 162-251 | High — 5 compartments × N assets |
| SkillInfo | `model/skill_info.go` | 253-301 | Medium — skills + cooldowns |
| QuestInfo | `model/quest_info.go` | 303-340 | Medium — started + completed |
| RingInfo | `model/ring_info.go` | 342-358 | Low — stub counts |
| TeleportInfo | `model/teleport_info.go` | 360-381 | Low — 5-10 ints |
| MonsterBookInfo | `model/monster_book_info.go` | 102-112 | Low — cover + cards |
| MiniGameInfo | `model/mini_game_info.go` | 114-118 | Trivial — 1 short |
| NewYearInfo | `model/new_year_info.go` | 120-124 | Trivial — 1 short |
| AreaInfo | `model/area_info.go` | 126-130 | Trivial — 1 short |

WriteCharacterInfo (lines 100-160) becomes the SetField struct's Encode method, composing all sub-models.

WriteCashEquipableIfPresent / WriteEquipableIfPresent (lines 252-280) fold into InventoryInfo.

getTime / timeNow helpers map to existing `model.MsTime()` in atlas-packet.

### 8. Models Already in atlas-packet (No New Work)
| Model | File | Used By Writers |
|-------|------|----------------|
| Avatar | `model/avatar.go` | character_list, character_view_all, character_spawn |
| Asset | `model/asset.go` | set_field (inventory), character_inventory_change |
| Pet | `model/pet.go` | character_spawn, pet writers |
| Buddy | `model/buddy.go` | buddy_operation |
| AttackInfo | `model/attack_info.go` | attack writers (CommonAttack struct) |
| DamageInfo | `model/damage_info.go` | character_damage, monster_damage |
| CharacterTemporaryStat | `model/character_temporary_stat.go` | character_buff_give, monster_stat |
| Movement | `model/movement.go` | character_movement, monster_movement, pet_movement |
| PaddedString | `model/padded_string.go` | party_operation (member names) |
| MsTime | `model/ms_time.go` | set_field (time conversion) |
| GuildMember | `model/guild_member.go` | guild_operation |
| Monster | `model/monster.go` | monster_spawn |
| Macros | `model/macros.go` | skill_macro |

### 9. New Models Needed
| Model | File | Fields | Phase |
|-------|------|--------|-------|
| CharacterStatistics | `model/character_statistics.go` | 20+ stat fields (see Decision 5) | 1 |
| ChannelLoad | `model/channel_load.go` | channelId byte, capacity uint16 | 1 |
| WorldRecommendation | `model/world_recommendation.go` | worldId uint32, reason string | 1 |
| InventoryInfo | `model/inventory_info.go` | compartment map: type → {capacity, assets} | 3 |
| SkillInfo | `model/skill_info.go` | skills: [{id, level, expiration, masterLevel}], cooldowns | 3 |
| QuestInfo | `model/quest_info.go` | started: [{id, progress}], completed: [{id, timestamp}] | 3 |
| TeleportInfo | `model/teleport_info.go` | locations [5]uint32 or [10]uint32 | 3 |
| RingInfo | `model/ring_info.go` | crushCount, friendCount, partnerCount uint16 | 3 |
| PartyInfo | `model/party_info.go` | members with padded names, jobs, levels | 6 |
| InventoryChange | `model/inventory_change.go` | mode + per-mode fields (add/qty/move/remove) | 7 |

## Phase Dependencies

```
Phase 1 (Foundation + Shared Models)
  ├── Phase 2 (Login Writers) → depends on CharacterStatistics, ChannelLoad, WorldRecommendation
  └── Phase 3 (Channel Core) → depends on CharacterStatistics + new sub-models (InventoryInfo, SkillInfo, QuestInfo, etc.)
        ├── Phase 4 (Effects & Monsters) — independent
        ├── Phase 5 (NPC, Drop, Pet) — independent
        ├── Phase 6 (Social & Commerce) → depends on PartyInfo model
        └── Phase 7 (Remaining) → depends on InventoryChange model
Phase 8 (Cleanup) — after all phases complete
```

Phases 4, 5 are fully independent of each other.
Phase 6 needs PartyInfo model (can be created inline).
Phase 7 needs InventoryChange model (can be created inline).
