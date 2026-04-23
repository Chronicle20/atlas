# Category 1 Decode Completion — Full Plan

Last Updated: 2026-03-11

## Executive Summary

Implement working Decode for all 6 Category 1 "pre-encoded composite byte" packets that currently have no-op Decode methods. These packets are not inherently undecodable — they hold pre-encoded bytes only because the service layer encodes before passing to atlas-packet. The fix for each is to either (a) implement Decode directly where fields are already typed, or (b) replace `[]byte` with structured types and implement Decode on those.

**NO DEFERRALS** — every packet in this plan will gain a working Decode with round-trip tests.

**6 Core Packets:**
1. CharacterInfo — implement Decode directly (all fields already typed)
2. CharacterSpawn — implement DecodeForeign + Pet.Decode, then CharacterSpawn.Decode
3. SetField — extract CharacterData struct, replace `characterInfoBytes []byte`
4. CashShopOpen — reuse CharacterData from SetField, replace `characterInfoBytes []byte`
5. InteractionEnter — define visitor structs in atlas-packet, replace `visitorBytes []byte`
6. InteractionEnterResultSuccess — define room structs in atlas-packet, replace `roomBytes []byte`

**5 Bonus Packets (unlocked by CTS work in Phase 7):**
7. BuffGive — uses CTS.Encode, unlocked by CTS.Decode
8. BuffGiveForeign — uses CTS.EncodeForeign, unlocked by CTS.DecodeForeign
9. BuffCancelW — uses CTS.EncodeMask, unlocked by CTS mask decode
10. BuffCancelForeign — uses CTS.EncodeMask, unlocked by CTS mask decode
11. StatChanged — mask-driven stat encoding, decodable with options context

**No-op trajectory:** 14 total no-ops → 3 remaining (Category 2: AttackWriter ×2, EffectSkillUse)

**Effort estimate:** XL total (8 phases, ~65 tasks)

## Current State Analysis

### CharacterInfo (`libs/atlas-packet/character/info.go`)
- All fields are already typed: characterId, level, jobId, fame, guildName, pets[]InfoPet, wishList[]uint32, medalId
- Encode uses deterministic patterns: 3-slot pet iteration with bool presence markers, byte-counted wishList, version-conditional monsterBook/chair
- Decode is no-op but STRAIGHTFORWARD — read in same order as Encode
- **Zero blockers.** Simplest of all 6.

### CharacterSpawn (`libs/atlas-packet/character/spawn.go`)
- Already holds typed models: `cts *model.CharacterTemporaryStat`, `avatar model.Avatar`, `pets []SpawnPet`
- **Blocker 1:** `model.CharacterTemporaryStat` has `EncodeForeign()` but NO `DecodeForeign()`
- **Blocker 2:** `model.Pet` has `Encode()` but NO `Decode()`
- **Category 2 issue:** `enteringField` bool controls whether `y-42` and `stance=6` are written vs raw position — this flag is NOT on the wire. However, Decode can read x/y/stance literally without reconstructing the flag.
- Version-conditional: driver/passenger (GMS>87/JMS), completedSetItemId (GMS>83), newYearCard (GMS<95), nPhase (GMS>87)

### SetField (`libs/atlas-packet/field/set_field.go`)
- Holds `characterInfoBytes []byte` — pre-encoded by `WriteCharacterInfo()` in the service layer
- `WriteCharacterInfo()` (~300 lines in `set_field.go:42-93`) writes: dbcharFlag, character stats (padded name, gender, skin, face, hair, pet IDs, level, job, str/dex/int/luk, hp/mp/maxhp/maxmp, ap, sp, exp, fame, map, spawnPoint...), buddy capacity, meso, 5 inventory compartments (equipment + items per compartment), skills (with cooldowns), quests (started + completed), miniGame, rings, teleports, monsterBook, newYear, area
- **This is the LARGEST task.** Requires creating a `CharacterData` struct in atlas-packet with sub-structs for each section.
- model.Asset already has Encode/Decode in atlas-packet (used for inventory items)

### CashShopOpen (`libs/atlas-packet/cash/shop_open.go`)
- Same `characterInfoBytes []byte` from `WriteCharacterInfo()`
- Once CharacterData is built for SetField, CashShopOpen reuses it
- Minimal incremental effort on top of SetField

### InteractionEnter (`libs/atlas-packet/interaction/interaction_writer.go:86-111`)
- Holds `visitorBytes []byte` — polymorphic visitor encoding from service layer
- 3 concrete visitor types:
  - **MiniRoomVisitorBase:** slot(1) + avatar(variable) + name(string)
  - **MiniGameRoomVisitor:** base bytes + MiniGameRecord (5 × uint32 = 20 bytes)
  - **MerchantOwnerVisitor:** slot=0(1) + itemId(4) + merchantName(string)
- The visitor list terminates with `0xFF` sentinel in room encoding, but InteractionEnter writes a single visitor — no sentinel needed
- **Decode challenge:** Discriminating visitor type from wire. MerchantOwnerVisitor always writes slot=0 then a uint32 itemId, while the other two write slot then an Avatar. Avatar starts with gender byte, so the first byte after slot gives the type: if it's decodable as Avatar (has gender=0/1), it's a base visitor; otherwise it's merchant. However, the simpler approach: the room context (set at construction time) determines the visitor type, and we can tag the visitor with a type byte.

### InteractionEnterResultSuccess (`libs/atlas-packet/interaction/interaction_writer.go:113-139`)
- Holds `roomBytes []byte` — polymorphic room encoding from service layer
- 3 concrete room types that implement Enter():
  - **GameMiniRoom:** type(1) + capacity(1) + visitors... + 0xFF + title(str) + gameKind(1) + tournament(bool) + optional round(1)
  - **PersonalShopMiniRoom:** type(1) + capacity(1) + visitors... + 0xFF + title(str) + maxItemCount(1) + count(1) + items...
  - **MerchantShopMiniRoom:** type(1) + capacity(1) + visitors... + 0xFF + messages(count+entries) + ownerName(str) + maxItemCount(1) + meso(4) + count(1) + items...
- **Decode dispatch:** First byte is room type (1=omok, 2=matchcard, 4=personalShop, 5=merchantShop). Type determines wire format.
- **MerchantShopMiniRoom** has an `isOwner` condition (`characterId == m.ownerId`) that controls whether messages are written. For Decode purposes, the count=0 case vs count>0 is distinguishable on wire, so this is decodable.
- Visitor decoding within rooms uses same visitor types from InteractionEnter

## Proposed Future State

After completion:
- **14 → 3 no-op Decode packets** — only Category 2 (AttackWriter ×2, EffectSkillUse) remain as documented no-ops
- All 11 solvable packets gain round-trip tests
- CharacterData is a reusable struct for any packet needing full character encoding
- Pet.Decode and CharacterTemporaryStat.DecodeForeign are available as model primitives
- CTS.Decode unlocks BuffGive, BuffCancel, and their Foreign variants
- StatChanged gains Decode using existing options-based stat index resolution
- Interaction visitor/room types live in atlas-packet, enabling independent testing

## Implementation Phases

### Phase 6: CharacterInfo Decode (Simplest — No Dependencies)

**Goal:** Implement Decode for CharacterInfo, add round-trip test.

**Approach:** Read fields in same order as Encode. For pets, iterate 3 slots reading bool presence markers. For wishList, read byte count then that many uint32s. Version-conditional sections (monsterBook, chair) follow the same tenant checks.

**Wire format:**
```
characterId(4) + level(1) + jobId(2) + fame(2) + marriageRing(bool/1) + guildName(str) + allianceName(str) + medalInfo(1)
+ 3× [present(bool) + if true: templateId(4) + name(str) + level(1) + closeness(2) + fullness(1) + skill(2) + itemId(4)]
+ morePets(bool/1) + mount(1)
+ wishListCount(1) + wishListCount × itemSN(4)
+ if GMS<87 or JMS: monsterBookLevel(4) + normalCard(4) + specialCard(4) + totalCards(4) + cover(4)
+ medalId(4) + medalQuests(2)
+ if GMS>83 or JMS: chair(4)
```

**Files:**
- Modified: `libs/atlas-packet/character/info.go` — implement Decode
- New: `libs/atlas-packet/character/info_test.go` — round-trip test
- Accessors needed: CharacterId(), Level(), JobId(), Fame(), GuildName(), Pets(), WishList(), MedalId()

### Phase 7: Pet.Decode + CharacterTemporaryStat.DecodeForeign (Prerequisites for CharacterSpawn)

**Goal:** Add missing Decode methods on model primitives.

#### 7A: Pet.Decode

**Wire format (from Pet.Encode):**
```
templateId(4) + name(str) + id(8) + x(2) + y(2) + stance(1) + foothold(2) + nameTag(1) + chatBalloon(1)
```
Fixed-format, straightforward.

**Files:**
- Modified: `libs/atlas-packet/model/pet.go` — add Decode method
- New: `libs/atlas-packet/model/pet_test.go` — round-trip test

#### 7B: CharacterTemporaryStat.DecodeForeign

**Wire format (from EncodeForeign):**
```
mask(4×uint32 = 16 bytes)
+ for each set bit in mask (sorted by shift): foreignValueWriter output (varies: 0/1/2/4/6 bytes per stat)
+ defenseAtt(1) + defenseState(1)
+ 7 base temporary stats:
  - 4× CharacterTemporaryStatBase(dynamic=true): nOption(4) + rOption(4) + time(bool+int32=5) + usExpireItem(2) = 15 bytes
  - 1× CharacterTemporaryStatBase(dynamic=false): nOption(4) + rOption(4) + time(5) = 13 bytes
  - 1× SpeedInfusion: base(13) + time(5) + usExpireItem(2) = 20 bytes
  - 1× GuidedBullet: base(13) + dwMobId(4) = 17 bytes
```

**Decoding challenge:** Need to know which foreign value size each stat type has. This is already encoded in the `ForeignValueWriter` field of `CharacterTemporaryStatType`, but the writer functions are write-only.

**Approach:** Add a parallel `ForeignValueReader` function type and a `foreignValueSize` byte to `CharacterTemporaryStatType`. For DecodeForeign:
1. Read 4×uint32 mask → reconstruct Uint128
2. Build ordered list of stat types whose bit is set (iterate all registered types, check mask)
3. For each set stat type, read the foreign value (size determined by the stat type's foreignValueReader)
4. Read defenseAtt(1), defenseState(1)
5. Read 7 base temporary stats with known format

This is the most complex Decode due to the mask-driven variable-length encoding, but it's fully deterministic given the tenant context.

**Investigation confirmed:** All 7 base stats (EnergyCharge, DashSpeed, DashJump, MonsterRiding, SpeedInfusion, HomingBeacon, Undead) use NoOpForeignValueWriter (0 bytes). The mask always includes these 7 bits, but they contribute 0 bytes to the foreign value section — no ambiguity during decode.

**Files:**
- Modified: `libs/atlas-packet/model/character_temporary_stat.go` — add DecodeForeign, ForeignValueReader, Decode on base stats
- New: `libs/atlas-packet/model/character_temporary_stat_test.go` — round-trip test for EncodeForeign/DecodeForeign

### Phase 8: CharacterSpawn Decode

**Goal:** Implement full Decode for CharacterSpawn.

**Approach:** Read fields in Encode order. For cts, call DecodeForeign (from Phase 7B). For avatar, call existing Avatar.Decode. For pets, iterate 3 slots reading bool markers + Pet.Decode (from Phase 7A). The `enteringField` flag is not reconstructed — Decode reads x/y/stance literally (the "as-written" values).

**Wire format:**
```
characterId(4) + level(1) + name(str)
+ guild: name(str) + logoBg(2) + logoBgColor(1) + logo(2) + logoColor(1)
+ cts.EncodeForeign bytes (variable)
+ jobId(2) + avatar bytes (variable)
+ if GMS>87 or JMS: driverId(4) + passengerId(4)
+ chocoCount(4) + itemEffect(4)
+ if GMS>83: completedSetItemId(4)
+ chair(4)
+ x(2) + y(2) + stance(1)
+ fh(2) + adminEffect(1)
+ 3× [present(bool) + if true: pet bytes (variable)]
+ endPets(1)
+ mountLevel(4) + mountExp(4) + mountTiredness(4)
+ miniRoom(1) + adBoard(1) + coupleRing(1) + friendshipRing(1) + marriageRing(1)
+ if GMS<95: newYearCard(1)
+ berserk(1)
+ version-conditional tail bytes
+ team(1)
```

**Design decision for enteringField:** The Decode reads x, y, stance as they appear on the wire. The round-trip test must account for the transformation: if `enteringField=true`, Encode writes `y-42` and `stance=6`, so Decode will read those transformed values. For testing, we test with `enteringField=false` to get exact round-trip, and add a separate test verifying the transformation.

**Files:**
- Modified: `libs/atlas-packet/character/spawn.go` — implement Decode
- New: `libs/atlas-packet/character/spawn_test.go` — round-trip test (enteringField=false)

### Phase 9: CharacterData Struct + SetField/CashShopOpen Decode (Largest Phase)

**Goal:** Extract the service-layer `WriteCharacterInfo` encoding into a structured `CharacterData` type in atlas-packet with full Encode+Decode. Replace `characterInfoBytes []byte` in both SetField and CashShopOpen.

#### 9A: Define CharacterData and sub-structs

The CharacterData struct mirrors `WriteCharacterInfo` section by section:

```go
// libs/atlas-packet/character/data.go
type CharacterData struct {
    Stats        CharacterStats
    BuddyCapacity byte
    Meso         uint32
    Inventory    InventoryData
    Skills       []SkillEntry
    Cooldowns    []CooldownEntry
    StartedQuests  []QuestProgress
    CompletedQuests []QuestCompleted
    // MiniGame, Rings, Teleport, MonsterBook, NewYear, Area — all zero-valued currently
}
```

**Sub-structs:**

```go
type CharacterStats struct {
    Id             uint32
    Name           string    // 13-byte padded
    Gender         byte
    SkinColor      byte
    Face           uint32
    Hair           uint32
    PetIds         []uint64  // up to 3 pet cash IDs
    Level          byte
    JobId          uint16
    Str, Dex, Int, Luk uint16
    Hp, MaxHp, Mp, MaxMp uint16
    Ap             uint16
    Sp             uint16    // or SP table (not currently implemented)
    Exp            uint32
    Fame           int16
    GachaponExp    uint32
    MapId          uint32
    SpawnPoint     byte
    // version-conditional tail fields
}

type InventoryData struct {
    EquipCapacity, UseCapacity, SetupCapacity, EtcCapacity, CashCapacity byte
    RegularEquipment []model.Asset  // equipped items (terminated)
    CashEquipment    []model.Asset  // cash-equipped items (terminated)
    EquipInventory   []model.Asset  // equipable inventory (terminated)
    UseInventory     []model.Asset  // consumable inventory (terminated)
    SetupInventory   []model.Asset  // setup inventory (terminated)
    EtcInventory     []model.Asset  // etc inventory (terminated)
    CashInventory    []model.Asset  // cash inventory (terminated)
}

type SkillEntry struct {
    Id          uint32
    Level       uint32
    Expiration  int64
    MasterLevel uint32  // only for 4th job skills
    FourthJob   bool
}

type CooldownEntry struct {
    SkillId  uint32
    Remaining uint16  // seconds
}

type QuestProgress struct {
    QuestId  uint16
    Progress string
}

type QuestCompleted struct {
    QuestId     uint16
    CompletedAt int64  // msTime
}
```

**Critical findings from investigation:**

1. **Asset.Decode does NOT read the slot position.** Asset.Encode writes the slot as the first field (via `encodeSlot`), but Asset.Decode starts AFTER the slot. The inventory-level code must read the slot byte/short first, then call Asset.Decode. Current callers (Storage Show/UpdateAssets) use `zeroPosition=true` which skips slot writing entirely.

2. **Inventory terminator asymmetry.** Equipment sections (regularEquip, cashEquip) terminate with WriteShort(0) for GMS>28 or WriteByte(0) for older. But the equipable inventory section uses WriteInt(0) as terminator despite WriteShort for slot prefix — this is an unexplained mismatch that needs a byte-comparison test before implementing Decode.

3. **Time-dependent values.** Fields like `getTime(-2)`, `msTime()`, cooldown durations are computed from `time.Now()`. CharacterData stores these as raw `int64` wire values, NOT `time.Time`. This ensures exact round-trip.

4. **IsFourthJob already exists** in `libs/atlas-constants/job/model.go` via `job.FromSkillId(skillId)` + `job.IsFourthJob()`. No duplication needed — import from atlas-constants. 23 fourth job IDs are enumerated.

5. **SkillEntry.FourthJob detection during Decode:** Use `job.IdFromSkillId()` to get the job ID, then `job.IsFourthJob()` to determine if masterLevel should be read. This exactly mirrors the service-layer `s.IsFourthJob()` check.

#### 9B: Implement CharacterData.Encode

Mirror `WriteCharacterInfo()` exactly, reading from the struct fields instead of `character.Model`:
- dbcharFlag (version-conditional int64 or int16)
- CharacterStats encoding (version-conditional pet IDs, gachapon exp, sub-job, etc.)
- Buddy capacity
- Linked name (always 0 / not linked)
- Meso
- JMS: characterId + dama
- InventoryData encoding (version-conditional capacity positions, equipped items with terminator, 5 inventory compartments with terminators)
- Skills + cooldowns
- Quests (started + completed)
- Zero-valued sections: miniGame, rings, teleports, monsterBook, newYear, area

**Equipment terminator pattern:** The service writes assets via `model.ForEachSlice` then writes a terminator. Terminator sizes vary by section:
- Regular equipment: WriteShort(0) for GMS>28, WriteByte(0) for older
- Cash equipment: same as regular
- Equipable inventory: WriteInt(0) — NOTE: this is 4 bytes despite WriteShort slot prefix (2 bytes). Requires byte-comparison test to verify.
- Stackable compartments (use/setup/etc/cash): WriteByte(0)
For Decode, read slot at inventory level (Asset.Decode does NOT read it), then dispatch to Asset.Decode. Read until terminator (0 byte/short/int depending on section).

#### 9C: Implement CharacterData.Decode

Read in same order as Encode. Key patterns:
- **Padded name:** Read 13 bytes, trim trailing zeros
- **Pet IDs:** Version-conditional: GMS>28 reads 3×uint64, GMS≤28 reads 1×uint64
- **Equipment lists:** Read assets until terminator (0 byte for position/slot marker signals end)
- **Skills:** Read count(uint16), then per skill: id(4)+level(4)+expiration(8)+optional masterLevel(4). During Decode, use `job.IdFromSkillId(skillId)` + `job.IsFourthJob()` from `libs/atlas-constants/job/model.go` to determine if masterLevel should be read. Set `FourthJob=true` on the decoded SkillEntry when detected. This exactly mirrors the Encode path.
- **Cooldowns:** Version-conditional, read count(uint16), then per cooldown: id(4)+seconds(2)
- **Quests:** Started: count(uint16), per quest: id(2)+progress(str). Completed: version-conditional, count(uint16), per quest: id(2)+msTime(8)
- **Zero sections:** Read fixed-count zeros for miniGame, rings, teleports, monsterBook, newYear, area

#### 9D: Update SetField

Replace `characterInfoBytes []byte` with `characterData CharacterData` in the struct. Encode calls `characterData.Encode()` internally. Decode reads outer frame then calls `characterData.Decode()`.

#### 9E: Update CashShopOpen

Replace `characterInfoBytes []byte` with `characterData CharacterData`. Same pattern as SetField.

#### 9F: Update service adapters

- `set_field.go`: `SetFieldBody()` builds a `CharacterData` from `character.Model` + `buddylist.Model`, passes to `fieldpkt.NewSetField(channelId, charData)`
- `cash_shop_open.go`: `CashShopOpenBody()` builds `CharacterData` same way, passes to `cashpkt.NewCashShopOpen(charData, accountName)`
- `WriteCharacterInfo()` helper and all its sub-functions (`WriteCharacterStatistics`, `WriteInventoryInfo`, `WriteSkillInfo`, `WriteQuestInfo`, etc.) are replaced by CharacterData construction + delegation to atlas-packet

**Files:**
- New: `libs/atlas-packet/character/data.go` — CharacterData, CharacterStats, InventoryData, SkillEntry, etc.
- New: `libs/atlas-packet/character/data_test.go` — round-trip test
- Modified: `libs/atlas-packet/field/set_field.go` — replace []byte with CharacterData
- Modified: `libs/atlas-packet/cash/shop_open.go` — replace []byte with CharacterData
- Modified: `services/atlas-channel/atlas.com/channel/socket/writer/set_field.go` — build CharacterData
- Modified: `services/atlas-channel/atlas.com/channel/socket/writer/cash_shop_open.go` — build CharacterData
- New: `libs/atlas-packet/field/set_field_decode_test.go` — SetField round-trip test

### Phase 10: Interaction Visitor + Room Structs

**Goal:** Define concrete visitor and room types in atlas-packet, replace `visitorBytes` and `roomBytes` with typed structs.

#### 10A: Visitor types in atlas-packet

```go
// libs/atlas-packet/interaction/visitor.go
type VisitorType byte
const (
    VisitorTypeBase     VisitorType = 0
    VisitorTypeGame     VisitorType = 1
    VisitorTypeMerchant VisitorType = 2
)

type Visitor struct {
    visitorType VisitorType
    slot        byte
    avatar      model.Avatar
    name        string
    // Game visitor additions
    record      GameRecord
    // Merchant owner additions
    itemId      uint32
    merchantName string
}

type GameRecord struct {
    Unknown, Wins, Ties, Losses, Points uint32
}
```

For Encode: dispatch on VisitorType:
- Base: slot + avatar.Encode + name
- Game: base + gameRecord (5×uint32)
- Merchant: slot(always 0) + itemId + merchantName

For Decode: This is used within the context of a room, where the room type determines the visitor type:
- Game rooms (type 1,2) → Game visitors
- Shop rooms (type 4,5) → Base visitors
- Merchant owner → Merchant visitor (but only the first visitor in a merchant room is the merchant owner; subsequent are base visitors)

**Wire discrimination approach:** The InteractionEnter packet sends a SINGLE visitor. The room type context is not in this packet. Solution: Include a `visitorType` field in InteractionEnter and let the service adapter set it based on the room context it knows. For round-trip testing, the visitor type is preserved.

#### 10B: Room types in atlas-packet

```go
// libs/atlas-packet/interaction/room.go
type RoomType byte
const (
    RoomTypeOmok         RoomType = 1
    RoomTypeMatchCard    RoomType = 2
    RoomTypePersonalShop RoomType = 4
    RoomTypeMerchantShop RoomType = 5
)

type Room struct {
    roomType     RoomType
    capacity     byte
    visitors     []Visitor
    title        string
    // Game room fields
    gameKind     byte
    tournament   bool
    round        byte
    // Shop fields
    maxItemCount byte
    items        []ShopItem
    // Merchant fields
    messages     []RoomMessage
    ownerName    string
    meso         uint32
}

type ShopItem struct {
    PerBundle uint16
    Quantity  uint16
    Price     uint32
    Asset     model.Asset
}

type RoomMessage struct {
    Message string
    Slot    byte
}
```

For Decode: First byte is RoomType. Dispatch:
- Type 1,2 (game): read capacity, visitors (each is game type) until 0xFF, title, gameKind, tournament bool, optional round
- Type 4 (personal shop): read capacity, visitors (base type) until 0xFF, title, maxItemCount, count, items
- Type 5 (merchant shop): read capacity, visitors (first=merchant, rest=base) until 0xFF, messageCount, messages, ownerName, maxItemCount, meso, count, items

#### 10C: Update InteractionEnter

Replace `visitorBytes []byte` with `visitor Visitor`. Encode writes mode + visitor.Encode(). Decode reads mode + visitor.Decode().

#### 10D: Update InteractionEnterResultSuccess

Replace `roomBytes []byte` with `room Room`. Encode writes mode + room.Encode(). Decode reads mode + type byte dispatch + room fields.

#### 10E: Update service adapter

`character_interaction.go`:
- `CharacterInteractionEnterBody()` builds an `interactionpkt.Visitor` from the service-layer `model.MiniRoomVisitor`
- `CharacterInteractionEnterResultSuccessBody()` builds an `interactionpkt.Room` from `model.MiniRoom`

**Files:**
- New: `libs/atlas-packet/interaction/visitor.go` — Visitor + GameRecord
- New: `libs/atlas-packet/interaction/room.go` — Room + ShopItem + RoomMessage
- Modified: `libs/atlas-packet/interaction/interaction_writer.go` — replace []byte in InteractionEnter + InteractionEnterResultSuccess
- New: `libs/atlas-packet/interaction/interaction_visitor_test.go` — round-trip tests
- New: `libs/atlas-packet/interaction/interaction_room_test.go` — round-trip tests
- Modified: `services/atlas-channel/atlas.com/channel/socket/writer/character_interaction.go` — build typed structs

### Phase 8B: Buff Packet Decode (Bonus — Unlocked by CTS Work)

**Goal:** Implement Decode for 4 buff packets that become solvable once CTS Decode/DecodeForeign exist.

**Packets:**
1. **BuffGive** (`libs/atlas-packet/character/buff_give_writer.go`): Uses CTS.Encode + tDelay(2) + MovementAffecting(bool). Decode calls CTS.Decode + reads tDelay + MovementAffecting.
2. **BuffGiveForeign** (same file): Uses characterId(4) + CTS.EncodeForeign + tDelay(2) + MovementAffecting(bool). Decode reads characterId, calls CTS.DecodeForeign, reads rest.
3. **BuffCancelW** (`libs/atlas-packet/character/buff_cancel_writer.go`): Uses CTS.EncodeMask + tSwallowBuffTime(4). Decode reads mask (4×uint32) + tSwallowBuffTime.
4. **BuffCancelForeign** (same file): Uses characterId(4) + CTS.EncodeMask. Decode reads characterId + mask.

**Files:**
- Modified: `libs/atlas-packet/character/buff_give_writer.go` — implement Decode for BuffGive + BuffGiveForeign
- Modified: `libs/atlas-packet/character/buff_cancel_writer.go` — implement Decode for BuffCancelW + BuffCancelForeign
- New: `libs/atlas-packet/character/buff_give_writer_test.go` — round-trip tests
- New: `libs/atlas-packet/character/buff_cancel_writer_test.go` — round-trip tests

### Phase 8C: StatChanged Decode (Bonus)

**Goal:** Implement Decode for StatChanged packet.

**Approach:** StatChanged uses a mask-driven encoding similar to CTS but for character stats. The stat index is resolved via `getStatIndex()` which reads from `options["statistics"]`. Each stat type has a fixed value size (1, 2, 4, or 8 bytes). Decode reads the mask, iterates stat types in index order, reads sized values for set bits. The `options` map is available during Decode via the context.

**Files:**
- Modified: `libs/atlas-packet/stat/changed.go` — implement Decode
- New: `libs/atlas-packet/stat/changed_test.go` — round-trip test

### Phase 11: Verification

**Goal:** Full builds, tests, and Docker verification.

- Build + test atlas-packet
- Build + test atlas-channel
- Docker build atlas-channel

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| CharacterData encoding doesn't match WriteCharacterInfo byte-for-byte | Medium | High | Write byte-comparison test: encode via old path vs new path, assert identical bytes |
| CTS DecodeForeign mask parsing misidentifies active stats | Low | High | **CONFIRMED SAFE:** All 7 base stats use NoOp (0 bytes). Still test: empty mask, single stat, multi-stat |
| Inventory terminator detection fails on edge cases | Medium | High | **ASYMMETRY FOUND:** Equipable inventory uses WriteInt(0) terminator but WriteShort slot prefix — write byte-comparison test FIRST (Phase 9.0) before implementing Decode |
| Skill 4th-job detection during Decode doesn't match Encode | Low | Medium | Use `job.IdFromSkillId()` + `job.IsFourthJob()` from atlas-constants — same logic as service layer |
| Interaction room Decode fails on MerchantShop owner/visitor messages | Low | Medium | Test both isOwner=true (messages present) and isOwner=false (count=0) cases |
| Pet.Decode Id field: Encode writes WriteLong(uint64(b.Id)) but Pet.Id is uint32 | Low | High | Pet Id is stored as uint32 but wire format is uint64 — Decode reads uint64, truncates to uint32 (same as Encode expands) |
| Asset.Decode doesn't read slot position | Low | High | **CONFIRMED:** Asset.Decode starts after slot. Inventory-level code reads slot byte/short first, then calls Asset.Decode. Existing callers use zeroPosition=true (no slot bytes) |
| Time-dependent fields prevent exact round-trip | Low | Medium | Store as raw int64 wire values, not time.Time. Round-trip test uses fixed int64 values |

## Success Metrics

- [ ] CharacterInfo has working Decode + round-trip test
- [ ] Pet has working Decode + round-trip test
- [ ] CharacterTemporaryStat has working DecodeForeign + round-trip test
- [ ] CharacterSpawn has working Decode + round-trip test
- [ ] BuffGive + BuffGiveForeign have working Decode + round-trip tests
- [ ] BuffCancelW + BuffCancelForeign have working Decode + round-trip tests
- [ ] StatChanged has working Decode + round-trip test
- [ ] CharacterData struct exists with Encode + Decode
- [ ] SetField has working Decode + round-trip test
- [ ] CashShopOpen has working Decode + round-trip test
- [ ] Interaction visitor types exist in atlas-packet with Encode + Decode
- [ ] InteractionEnter has working Decode + round-trip test
- [ ] Interaction room types exist in atlas-packet with Encode + Decode
- [ ] InteractionEnterResultSuccess has working Decode + round-trip test
- [ ] WriteCharacterInfo + helper functions removed from service layer
- [ ] All builds pass: atlas-packet tests, atlas-channel build + test
- [ ] Docker build verified: atlas-channel
- [ ] No-op Decode count reduced from 14 to 3 (Category 2 only)

## Dependencies

- `libs/atlas-packet/model/avatar.go` — Avatar.Decode already exists (verified)
- `libs/atlas-packet/model/asset.go` — Asset.Decode already exists (verified)
- `libs/atlas-packet/tool/uint128.go` — Uint128 with And/Or/ShiftLeft already exists
- `libs/atlas-constants/character` — TemporaryStatType constants (already imported)
- `libs/atlas-constants/inventory/slot` — slot.Slots for equipment iteration
- No external dependency changes needed
