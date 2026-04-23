# Writer Packet Extraction to atlas-packet

Last Updated: 2026-03-10

## Executive Summary

Migrate all 109 server-send ("writer") packets from atlas-login (18 files) and atlas-channel (91 files) into the shared `libs/atlas-packet` library. Each writer function becomes a struct implementing the `Packet` interface. All files are extractable — attack writers accept pre-computed primitives (mastery byte, resolved bullet ID) so the computation stays in atlas-channel while the packet struct moves to atlas-packet.

## Current State

### Writer Location
- **atlas-login**: `services/atlas-login/atlas.com/login/socket/writer/` (18 writer files + writer.go)
- **atlas-channel**: `services/atlas-channel/atlas.com/channel/socket/writer/` (91 writer files + writer.go)
- **Session writers**: Both services have `session/writer.go` with `WriteHello()` handshake

### Current Writer Pattern
```go
const CharacterList = "CharacterList"

func CharacterListBody(characters []character.Model, ...) packet.Encode {
    return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
        w := response.NewWriter(l)
        return func(options map[string]interface{}) []byte {
            // serialize fields
            return w.Bytes()
        }
    }
}
```

### Overlap Between Services
- **ping.go**: Identical empty body in both services
- **session/writer.go**: Identical `WriteHello()` handshake in both services
- **WriteCharacterStatistics**: Shared helper between `character_list.go` (login) and `set_field.go` (channel) — identical field order/structure with one known divergence (pet ID field, see Decision 8)

## Resolved Design Decisions

### Decision 1: Attack Writers Extract with Pre-Computed Primitives
**character_attack_common.go** currently mixes packet serialization with game logic:
- `computeMasteryForWeapon()` — depends on `atlas-channel/data/skill` (HTTP to data service), `atlas-constants/job`, `atlas-constants/item`
- `getMasteryFromSkill()` — depends on `atlas-channel/character/skill` model
- Bullet resolution — iterates `character.Inventory().Cash().Assets()` and `.Consumable().Assets()`
- Skill level lookup — iterates `character.Skills()` array

Moving these computations to atlas-packet would create circular dependencies. However, the packet itself only **writes** simple primitives:
- `mastery` → 1 byte (line 79)
- `bulletItemId` → 1 uint32 (line 96)
- `skillLevel` → 1 byte (line 37)

**Resolution**: The atlas-packet struct accepts pre-computed values. Computation stays in atlas-channel as a service-layer helper:

```go
// libs/atlas-packet/character/attack/common.go
type CommonAttack struct {
    characterId  uint32
    level        byte
    skillId      uint32
    skillLevel   byte       // pre-computed: looked up from character.Skills()
    attackInfo   model.AttackInfo
    mastery      byte       // pre-computed: computeMasteryForWeapon() result
    bulletItemId uint32     // pre-computed: resolved from inventory
}
```

```go
// services/atlas-channel/socket/writer/character_attack_helper.go (stays in service)
// computeMasteryForWeapon() and getMasteryFromSkill() remain here
// resolveBulletItemId() remains here
// Called by handler before constructing the packet struct
```

The 4 attack type files (melee, ranged, magic, energy) are currently single-line delegates to `WriteCommonAttackBody` — they become trivial constructors for the same `CommonAttack` struct with different `Operation()` strings.

**All 5 attack files extract to atlas-packet. Only computation helpers stay in atlas-channel.**

### Decision 2: Multi-Operation Writer Decomposition
Analysis of existing handler-side patterns shows atlas-packet uses **separate structs per sub-operation** (see `interaction/operation*.go`, `guild/operation*.go`, `cash/shop_operation*.go`). Writers follow the same approach:

| Writer File | Current Pattern | Atlas-Packet Pattern |
|-------------|----------------|---------------------|
| `login_status.go` (4 ops: AuthSuccess, AuthTemporaryBan, AuthPermanentBan, AuthLoginFailed) | 4 separate functions with completely different payloads, registered as 4 separate writer constants | **4 separate structs** — payloads share no fields; already registered separately |
| `character_effect.go` (26 modes) | 26 body functions sharing a mode byte prefix, registered as 2 constants (CharacterEffect, CharacterEffectForeign) | **26 separate structs** + a parent `Effect` struct with mode field — matches handler pattern; Foreign variants are separate structs that embed the base + add characterId |
| `cash_shop_operation.go` (15+ ops) | Separate functions sharing mode byte lookup, registered as 1 constant | **Separate structs per operation** — matches handler-side `cash/shop_operation*.go` pattern |
| `character_inventory_change.go` (4 modes) | Functional composer pattern (multiple changes per packet) | **Single struct with mode + slice of InventoryChangeWriter** — composer pattern preserved since multiple changes can be batched in one packet |
| `guild_operation.go` (27+ ops) | Separate functions with mode lookup | **Separate structs per operation** — matches handler-side `guild/operation*.go` |
| `buddy_operation.go` | Separate functions with mode lookup | **Separate structs per operation** — matches handler-side `buddy/` |
| `party_operation.go` | Separate functions with helper WriteParty() | **Separate structs per operation** + shared WriteParty model in `atlas-packet/model/` |

### Decision 3: Service Integration Pattern
Writer constant strings are **runtime lookup keys** in the writer producer map. The `session.Announce` chain is:
```
session.Announce(l)(ctx)(wp)(writerName)(encoder)(session)
  → wp(writerName) → BodyFunc (map lookup by string)
  → BodyFunc(l, ctx)(encoder) → []byte
```

After extraction:
- **Writer constant** (`const CharacterList = "CharacterList"`) stays in service OR is re-exported from the packet's `Operation()` return
- **BodyFunc** (`CharacterListBody(...)`) moves to atlas-packet as `packet.Encode` via the struct's `.Encode` method
- **Registration** in `main.go` / `produceWriters()` unchanged
- **Announce call site** changes from `writer.CharacterListBody(...)` to constructing a packet and passing `.Encode`

### Decision 4: writer.go Utility Stays in Services
Both services define in `writer.go`:
- `BodyFunc` type alias → `sw.BodyFunc` (from atlas-socket)
- `Producer` type alias → `sw.Producer`
- `getCode()` helper → reads operation codes from `options["operations"]` map

These are service infrastructure for the socket layer. They stay.

### Decision 5: Configuration and RestModel Stay in Services
`configuration/tenant/socket/writer/rest.go` defines `RestModel{OpCode, Writer, Options}` for loading tenant-specific opcode mappings from config. This is deployment/configuration concern, not packet structure. Stays.

### Decision 6: Shared CharacterStatistics Model
Both `character_list.go` (login) and `set_field.go` (channel) contain `WriteCharacterStatistics()` with **identical field order**:
- Id (int32), Name (13-byte padded), Gender, SkinColor, Face, Hair
- Pets (3 slots or 1 long, region-dependent)
- Level, JobId, Str/Dex/Int/Luk, HP/MaxHP, MP/MaxMP, AP, SP
- Experience, Fame, GachaponExperience (v28+), MapId, SpawnPoint
- Regional padding

**Extract to `atlas-packet/model/character_statistics.go`** as a shared encode/decode model. Both services construct it from their respective `character.Model` types.

### Decision 7: set_field.go Decomposition (461 lines)
This is the largest single writer. Decompose into:

| Component | Atlas-Packet Location | Description |
|-----------|----------------------|-------------|
| WarpToMap | `field/warp_to_map.go` | Simple: channelId, mapId, portalId, hp |
| SetField (top-level) | `field/set_field.go` | Orchestrator: calls sub-encoders |
| CharacterStatistics | `model/character_statistics.go` | Shared with character_list (Decision 6) |
| InventoryInfo | `model/inventory_info.go` | 5 compartments (Equip/Use/Setup/ETC/Cash) with capacity + items |
| SkillInfo | `model/skill_info.go` | Skills array with cooldowns, master levels |
| QuestInfo | `model/quest_info.go` | Started + completed quests |
| RingInfo | `model/ring_info.go` | Crush/friendship/partner rings |
| TeleportInfo | `model/teleport_info.go` | 5-10 teleport rock locations |
| MonsterBookInfo | `model/monster_book_info.go` | Cover + cards |
| MiniGameInfo | `model/mini_game_info.go` | Stub (1 short) |
| NewYearInfo | `model/new_year_info.go` | Stub (1 short) |
| AreaInfo | `model/area_info.go` | Stub (1 short) |

Each sub-model has its own Encode/Decode for round-trip testing. The SetField struct composes them.

Service-local types (`character.Model`, `equipment.Model`, `inventory.Model`, `skill.Model`, `quest.Model`) are mapped to these packet models at the call site in atlas-channel. The packet models contain only serializable primitive fields.

### Decision 8: Pet ID Divergence
`character_list.go` (login) writes `character.Pets()[0].CashId()` while `set_field.go` (channel) writes `character.Pets()[0].Id()`. Both are writing the same protocol field (pet unique ID in character stats).

**Resolution**: The shared CharacterStatistics model takes a `petId uint64` parameter. Each service maps from its own pet model:
- Login: passes `pet.CashId()` (this is the correct value — the client expects the cash item ID here)
- Channel: passes `pet.Id()` — **investigate if this is a bug**. Flag for verification during implementation. If channel's pet.Id() returns the same value as login's pet.CashId(), no issue. If different, channel needs to pass CashId() too.

### Decision 9: New Shared Models Needed
| Model | Package | Fields | Used By |
|-------|---------|--------|---------|
| CharacterStatistics | `model/` | id, name, gender, skin, face, hair, petIds, level, job, stats, hp/mp, ap/sp, exp, fame, mapId, spawnPoint | character_list, set_field, character_spawn, character_view_all |
| ChannelLoad | `model/` | channelId byte, capacity uint16 | server_list |
| WorldRecommendation | `model/` | worldId uint32, reason string | server_list_recommendations |
| InventoryInfo | `model/` | compartments map with capacity + asset arrays | set_field |
| SkillInfo | `model/` | skills array (id, level, expiration, masterLevel) + cooldowns | set_field |
| QuestInfo | `model/` | started (id, progress) + completed (id, completedAt) | set_field |
| PartyInfo | `model/` | member array with padded name, job, level, channel, map | party_operation |
| InventoryChange | `model/` | mode + per-mode fields (add/quantity/move/remove) | character_inventory_change |

Models already in atlas-packet that are reused: Avatar, Asset, Pet, Buddy, AttackInfo, DamageInfo, CharacterTemporaryStat, Movement, PaddedString, MsTime.

### Decision 10: character_effect.go Foreign Variants
`CharacterEffect` writes effects on self (no characterId prefix). `CharacterEffectForeign` writes the same effect prefixed with the source characterId (shown to other players).

**Pattern**: Each of the 26 effects gets a struct. Foreign variants are handled by a wrapper struct:
```go
type EffectForeign struct {
    characterId uint32
    effect      EffectEncoder  // interface with Encode method
}
```
This avoids duplicating 26 structs. The `EffectEncoder` interface is satisfied by all 26 effect structs.

## Implementation Phases

### Phase 1: Foundation & Shared Models (S)
- Extract `socket/ping.go` — shared Ping packet (empty body)
- Extract `socket/hello.go` — WriteHello handshake
- Extract `model/character_statistics.go` — shared character stat encoder/decoder
- Extract `model/channel_load.go` — ChannelLoad DTO
- Extract `model/world_recommendation.go` — WorldRecommendation DTO
- Verify atlas-packet builds

### Phase 2: Login Writer Packets (M)
Login has 18 files. All are extractable (no circular dependency issues).

**2a — Simple packets (7)**: server_status, server_load, pic_result, pin_operation (6 modes → 6 body functions, 1 struct with mode field), pin_update, set_account_result, select_world

**2b — Server/Auth packets (5)**: server_list_entry + server_list_end (2 structs), server_list_recommendations, server_ip (3 variants: Ok, SimpleError, Error → 3 structs), login_auth

**2c — Login status (4 separate structs)**: auth_success, auth_temporary_ban, auth_permanent_ban, auth_login_failed — completely different payloads, already registered as 4 separate writer constants

**2d — Character selection (5)**: character_name_response, add_character_entry (Ok + Error → 2 structs), delete_character_response (Ok + Error → 2 structs), character_list (uses shared CharacterStatistics + Avatar), character_view_all (4 sub-operations: Count, SearchFailed, Error, Characters)

**Verification**: Update atlas-login imports, remove old files, build + test

### Phase 3: Channel Writer Packets - Core Character (L)
Most complex phase due to set_field.go decomposition.

- character_spawn (uses Avatar, CharacterTemporaryStat, guild info, pets)
- character_despawn (simple: characterId)
- character_movement writer (characterId + Movement model — already in atlas-packet)
- character_damage, character_info, character_appearance_update
- character_expression, character_sit_result, character_hint
- character_interaction, character_status_message
- stat_changed
- **set_field decomposition** (Decision 7): WarpToMap + SetField orchestrator + sub-models
- character_buff_give + Foreign, character_buff_cancel writer + Foreign

### Phase 4: Channel Writer Packets - Combat, Effects & Monsters (M)
- **attack packets**: CommonAttack struct with pre-computed mastery/bullet/skillLevel (Decision 1). 4 attack types (melee, ranged, magic, energy) share the struct with different Operation() values. `computeMasteryForWeapon()`, `getMasteryFromSkill()`, and bullet resolution stay in atlas-channel as service helpers
- **character_effect**: 26 effect structs + EffectForeign wrapper (Decision 10)
- monster_spawn (+ WithEffect variant), monster_destroy
- monster_movement writer, monster_movement_ack
- monster_control, monster_damage, monster_health, monster_stat

### Phase 5: Channel Writer Packets - NPC, Drop, Pet (M)
All straightforward extractions — no circular dependencies.

- npc_spawn, npc_action writer, npc_conversation, npc_shop, npc_shop_operation, npc_spawn_request_controller
- drop_spawn (with DropEnterType enum), drop_destroy
- pet_activated, pet_movement writer, pet_chat, pet_command, pet_exclude, pet_cash_food_result

### Phase 6: Channel Writer Packets - Social & Commerce (M)
Multi-operation writers decomposed per Decision 2.

- **buddy_operation**: separate structs per operation (Update, Invite, ChannelChange, CapacityChange, errors)
- **party_operation**: separate structs + shared PartyInfo model in atlas-packet/model
- party_member_hp
- **guild_operation**: separate structs per operation (27+ ops, matching handler-side pattern)
- guild_emblem_changed, guild_name_changed, guild_bbs
- messenger_operation
- **cash_shop_open**, **cash_shop_operation** (separate structs per operation), cash_query_result
- **hired_merchant_operation**: separate structs per mode (9 modes)
- storage_operation

### Phase 7: Channel Writer Packets - Remaining (M)
- **inventory**: character_inventory_change (composer pattern with InventoryChange model), item_upgrade
- **skills**: skill_change, skill_cooldown, skill_macro
- **keymap**: key_map, auto_hp, auto_mp
- **chat**: general, whisper, multi
- **field**: field_effect, field_effect_weather, field_transport, channel_change writer
- **misc**: clock, world_message, script_progress, chalkboard, fame_response, guide_talk
- **UI**: ui_open, ui_lock, ui_disable
- **objects**: note_operation, kite_spawn/destroy/error, reactor_spawn/hit/destroy writers
- **other**: mini_room, chair_show, compartment_merge, compartment_sort

### Phase 8: Service Cleanup & Verification (M)
- Remove all extracted writer files from both services
- Keep in atlas-channel: writer.go utility + attack computation helpers (computeMasteryForWeapon, getMasteryFromSkill, bullet resolution)
- Keep in atlas-login: writer.go utility
- Full build + test for atlas-packet, atlas-login, atlas-channel
- Docker build verification
- Round-trip test coverage for all extracted packets

## Risk Assessment

| Risk | Impact | Likelihood | Mitigation | Status |
|------|--------|-----------|------------|--------|
| Writer-only packets need Decode() | Low | Certain | Implement reverse of Encode; enables round-trip testing | RESOLVED — standard approach |
| set_field.go complexity (461 lines, 17 helpers) | High | Certain | Decompose into 12 sub-models (Decision 7) | RESOLVED — decomposition plan defined |
| Attack writers have game logic dependencies | Medium | Certain | Pre-compute mastery/bullet/skillLevel in service; packet takes primitives (Decision 1) | RESOLVED — all extract |
| Multi-operation writer patterns unclear | Medium | Certain | Separate structs per operation (Decision 2) | RESOLVED — per-type analysis done |
| Pet ID divergence between login/channel | Low | Possible bug | Parameterize in CharacterStatistics (Decision 8) | RESOLVED — flagged for verification |
| Service model mapping overhead | Low | Certain | Shared models are primitives-only; mapping at call site | RESOLVED — model list defined (Decision 9) |
| Writer constant registration unchanged | Low | Certain | Constants stay in services; only BodyFunc moves (Decision 3) | RESOLVED |
| Large file count (104 extractions) | Medium | Medium | Phase the work; merge each phase before starting next | Accepted |

## Success Metrics

1. All 109 writer files extracted to atlas-packet structs implementing full Packet interface
2. Both services build and pass tests after each phase
3. Round-trip encode/decode tests pass across GMS/JMS variants
4. Shared CharacterStatistics model used by both login and channel
5. No duplicate packet definitions between services
6. Attack mastery/bullet computation helpers remain in atlas-channel as service-layer code (not packet code)

## Dependencies

- `libs/atlas-packet` handler packets (COMPLETE)
- `github.com/Chronicle20/atlas-socket` v1.2.7
- `github.com/Chronicle20/atlas-tenant` v1.0.7
- Shared models in `atlas-packet/model/` (Avatar, Movement, etc.) — already exist
- New shared models defined in Decision 9 — created in Phase 1
