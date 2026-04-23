# Category 1 Decode Completion — Tasks

Last Updated: 2026-03-11

## Phase 6: CharacterInfo Decode [4/4] COMPLETE

### 6.1 Implement CharacterInfo.Decode
- [x] Implement Decode in `libs/atlas-packet/character/info.go`: characterId, level, jobId, fame, marriageRing(bool), guildName, allianceName, medalInfo, pets (bool-terminated loop), mount, wishList (byte-counted), version-conditional monsterBook/chair, medalId, medalQuests — S

### 6.2 Add accessors
- [x] Add getter methods: CharacterId(), Level(), JobId(), Fame(), GuildName(), Pets(), WishList(), MedalId() — S

### 6.3 Round-trip test
- [x] Added round-trip tests to `libs/atlas-packet/character/info_test.go` (with pets, with wishList, empty pets) — S

### 6.4 Build + test
- [x] Build + test atlas-packet — S

## Phase 7: Pet.Decode + CTS.DecodeForeign [8/8] COMPLETE

### 7.1 Implement Pet.Decode
- [x] Add Decode to `libs/atlas-packet/model/pet.go`: templateId(4) + name(str) + id(8→uint32) + x(2) + y(2) + stance(1) + foothold(2) + nameTag(1) + chatBalloon(1) — S

### 7.2 Pet round-trip test
- [x] Create `libs/atlas-packet/model/pet_test.go` with round-trip test — S

### 7.3 Add ForeignValueReader to CharacterTemporaryStatType
- [x] Add `ForeignValueReader` function type and `foreignValueReader` field to `CharacterTemporaryStatType` — M
- [x] Register corresponding reader for each writer: NoOpReader(0 bytes), ByteReader(1), ShortReader(2), IntReader(4), LevelSourceReader(4), ValueSourceLevelReader(6) — M

### 7.4 Implement CharacterTemporaryStatBase.Decode
- [x] Add Decode to `CharacterTemporaryStatBase`: nOption(4) + rOption(4) + readTime(bool+int32) + conditional usExpireItem(2) — S

### 7.5 Implement SpeedInfusionTemporaryStat.Decode + GuidedBulletTemporaryStat.Decode
- [x] SpeedInfusion: base.Decode + readTime + usExpireItem(2) — S
- [x] GuidedBullet: base.Decode + dwMobId(4) — S

### 7.6 Implement CharacterTemporaryStat.DecodeForeign
- [x] Read 4×uint32 mask → Uint128 via DecodeMask — S
- [x] Iterate registered stat types in shift order, read foreign value for set bits using ForeignValueReader (skip base stats) — M
- [x] Read defenseAtt(1) + defenseState(1) — S
- [x] Read 7 base temporary stats (4×dynamic + 1×non-dynamic + SpeedInfusion + GuidedBullet) — M

### 7.7 CTS round-trip test
- [x] Create `libs/atlas-packet/model/character_temporary_stat_test.go` — round-trip EncodeForeign/DecodeForeign: empty stats, single stat (Byte writer), multiple stats (Byte + Int writers) — M

### 7.8 Build + test
- [x] Build + test atlas-packet — all tests pass — S

## Phase 8: CharacterSpawn Decode [4/4] COMPLETE

### 8.1 Implement CharacterSpawn.Decode
- [x] Full Decode: characterId, level, name, guild, cts.DecodeForeign, jobId, avatar.Decode, version-conditional fields, x/y/stance, pets (bool-terminated loop), mount, rings, version-conditional tail, team — S

### 8.2 Add accessors
- [x] Add getter methods: CharacterId(), Level(), Name(), Guild(), Cts(), JobId(), Avatar(), Pets(), X(), Y(), Stance() — S

### 8.3 Round-trip test
- [x] Added round-trip tests to `libs/atlas-packet/character/spawn_test.go`: enteringField=false (exact round-trip), with pets — M

### 8.4 Build + test
- [x] Build + test atlas-packet + atlas-channel — S

## Phase 8B: Buff Packet Decode (Bonus) [6/6] COMPLETE

### 8B.1 Implement BuffGive.Decode
- [x] Add Decode: CTS.Decode + tDelay(2) + MovementAffecting(byte) — M

### 8B.2 Implement BuffGiveForeign.Decode
- [x] Add Decode: characterId(4) + CTS.DecodeForeign + tDelay(2) + MovementAffecting(byte) — M

### 8B.3 BuffGive round-trip tests
- [x] Create `libs/atlas-packet/character/buff_give_writer_test.go`: BuffGive empty + BuffGiveForeign empty — M

### 8B.4 Implement BuffCancelW.Decode
- [x] Add Decode: CTS DecodeMask (4×uint32) + tSwallowBuffTime(1) — S

### 8B.5 Implement BuffCancelForeign.Decode
- [x] Add Decode: characterId(4) + CTS DecodeMask (4×uint32) + tSwallowBuffTime(1) — S

### 8B.6 BuffCancel round-trip tests
- [x] Create `libs/atlas-packet/character/buff_cancel_writer_test.go`: BuffCancelW + BuffCancelForeign — S

## Phase 8C: StatChanged Decode (Bonus) [3/3] COMPLETE

### 8C.1 Implement StatChanged.Decode
- [x] Add Decode: excl(bool) + mask(uint32) + mask-driven stat values using reverse stat type lookup from options — M

### 8C.2 StatChanged round-trip test
- [x] Create `libs/atlas-packet/stat/changed_test.go`: single stat, multiple stats, empty — M

### 8C.3 Build + test
- [x] Build + test atlas-packet — S

## Phase 9: CharacterData + SetField/CashShopOpen Decode [16/16] COMPLETE

### 9.1-9.9 CharacterData implementation
- [x] Created `libs/atlas-packet/character/data.go` with full CharacterData struct (CharacterStats, InventoryData, SkillEntry, CooldownEntry, QuestProgress, QuestCompleted) — L
- [x] Full Encode mirroring WriteCharacterInfo — L
- [x] Full Decode with inventory section helpers (decodeEquipmentSection, decodeEquipableInventorySection, decodeStackableSection) — L
- [x] Skill decode using `job.IdFromSkillId` + `job.IsFourthJob` from atlas-constants — M

### 9.10 CharacterData round-trip test
- [x] Create `libs/atlas-packet/character/data_test.go`: minimal, with skills, with quests — M

### 9.11 Update SetField
- [x] Replace `characterInfoBytes []byte` with `characterData CharacterData` — M
- [x] Add round-trip test in `libs/atlas-packet/field/set_field_test.go` — M

### 9.12 Update CashShopOpen
- [x] Replace `characterInfoBytes []byte` with `characterData CharacterData` — M
- [x] Add round-trip test in `libs/atlas-packet/cash/shop_open_test.go` — M

### 9.13 Update service adapters
- [x] Create `character_data.go` with BuildCharacterData helper — L
- [x] Update `set_field.go`: use BuildCharacterData, remove WriteCharacterInfo and all helper functions — M
- [x] Update `cash_shop_open.go`: use BuildCharacterData — M

### 9.14 Build + test
- [x] Build + test atlas-packet — all 30 packages pass — S
- [x] Build + test atlas-channel — all packages pass — S

## Phase 10: Interaction Visitor + Room Types [10/10] COMPLETE

### 10.1 Define visitor types
- [x] Create `libs/atlas-packet/interaction/visitor.go` with Visitor struct (VisitorType, slot, avatar, name, record, itemId, merchantName) — M
- [x] Add constructors: NewBaseVisitor, NewGameVisitor, NewMerchantVisitor — S

### 10.2 Implement Visitor.Encode + Decode
- [x] Encode: dispatch on visitor type — M
- [x] Decode: dispatch on visitor type, plus decodeVisitorForRoom for room-context dispatch — M

### 10.3 Define GameRecord
- [x] GameRecord struct inline in Visitor (5×uint32) — S

### 10.4 Visitor round-trip tests
- [x] Round-trip tests for base, game, and merchant visitors — M

### 10.5-10.6 Define room types + Encode/Decode
- [x] Create `libs/atlas-packet/interaction/room.go` with Room struct + RoomShopItem + RoomMessage — L
- [x] Game room, Personal shop, Merchant shop encode/decode with type-dispatch — L

### 10.7 Room round-trip tests
- [x] Round-trip tests for game room, personal shop, merchant shop — M

### 10.8 Update InteractionEnter
- [x] Replace `visitorBytes []byte` with `visitor Visitor` — M

### 10.9 Update InteractionEnterResultSuccess
- [x] Replace `roomBytes []byte` with `room Room` — M

### 10.10 Update service adapter + build
- [x] Add ToPacketVisitor to MiniRoomVisitor interface + all implementations — M
- [x] Add ToPacketRoom to MiniRoom interface + all room implementations — M
- [x] Build + test atlas-packet — all pass — S
- [x] Build + test atlas-channel — all pass — S

## Phase 11: Final Verification [4/4] COMPLETE

- [x] Full Docker build: atlas-channel — S
- [x] Verify no-op Decode count: 3 Category 2 (AttackWriter, EffectSkillUse, EffectSkillUseForeign) + 5 zero-data (CheckWallet, ChalkboardClose, KiteError, Ping, Pong) — S
- [x] Update task doc to mark all phases complete — S
- [x] Update docs/TODO.md with remaining no-op packets, remove stale set_field/cash_shop_open references — S
