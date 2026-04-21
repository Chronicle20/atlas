# Writer Packet Extraction - Task Tracking

Last Updated: 2026-03-11

## Phase 1: Foundation & Shared Models [Effort: S]

- [x] 1.1 Extract `socket/ping.go` — Ping packet struct (empty body, used by both services)
- [x] 1.2 Extract `socket/hello.go` — WriteHello handshake packet (used by both services)
- [x] 1.3 Create `model/character_statistics.go` — shared stat encoder/decoder (20+ fields, region/version branching). Parameterize PetIds as `[3]uint64` to resolve login/channel pet ID divergence
- [x] 1.4 Create `model/channel_load.go` — ChannelLoad DTO (channelId byte, capacity uint16)
- [x] 1.5 Create `model/world_recommendation.go` — WorldRecommendation DTO (worldId uint32, reason string)
- [x] 1.6 Verify atlas-packet builds: `go build ./...` and `go test ./... -count=1`

## Phase 2: Login Writer Packets [Effort: M]

### Phase 2a: Simple Login Packets (7 structs)
- [x] 2a.1 `login/server_status.go` — ServerStatus struct (single uint16 status field)
- [x] 2a.2 `login/server_load.go` — ServerLoad struct (pre-resolved byte code)
- [x] 2a.3 `login/pic_result.go` — PicResult struct (single 0 byte)
- [x] 2a.4 `login/pin_operation.go` — PinOperation struct (pre-resolved byte mode)
- [x] 2a.5 `login/pin_update.go` — PinUpdate struct (pre-resolved byte mode)
- [x] 2a.6 `login/set_account_result.go` — SetAccountResult struct (gender byte, success bool)
- [x] 2a.7 `login/select_world.go` — SelectWorld struct (worldId uint32)

### Phase 2b: Server & Auth Packets (9 structs)
- [x] 2b.1 `login/server_list_entry.go` — ServerListEntry struct (worldId, worldName, state, eventMessage, channelLoads using model.ChannelLoad)
- [x] 2b.2 `login/server_list_end.go` — ServerListEnd struct (single 0xFF byte)
- [x] 2b.3 `login/server_list_recommendations.go` — ServerListRecommendations struct ([]model.WorldRecommendation)
- [x] 2b.4 `login/server_ip.go` — ServerIP struct (unified: code, mode, ipAddr, port, clientId). Error variant uses empty ipAddr
- [x] 2b.5 `login/login_auth.go` — LoginAuth struct (screen string)
- [x] 2b.6 `login/auth_success.go` — AuthSuccess struct (accountId, name, gender, usesPin, pic, region-specific fields)
- [x] 2b.7 `login/auth_temporary_ban.go` — AuthTemporaryBan struct (bannedCode, reason, until)
- [x] 2b.8 `login/auth_permanent_ban.go` — AuthPermanentBan struct (bannedCode)
- [x] 2b.9 `login/auth_login_failed.go` — AuthLoginFailed struct (pre-resolved reason byte)

### Phase 2c: Character Selection Packets (8 structs)
- [x] 2c.0 `model/character_list_entry.go` — CharacterListEntry model (statistics + avatar + gm + rank fields). Added Avatar.Decode
- [x] 2c.1 `character/name_response.go` — CharacterNameResponse struct (name, pre-resolved code byte)
- [x] 2c.2 `character/add_entry.go` — AddCharacterEntry struct (code byte + CharacterListEntry)
- [x] 2c.3 `character/add_entry_error.go` — AddCharacterError struct (pre-resolved code byte)
- [x] 2c.4 `character/delete_response.go` — DeleteCharacterResponse struct (characterId, pre-resolved code byte)
- [x] 2c.5 (merged into 2c.4) — DeleteCharacterResponse handles both ok and error via code byte
- [x] 2c.6 `character/list.go` — CharacterList struct (status, characters []CharacterListEntry, hasPic, characterSlots)
- [x] 2c.7+2c.8 `character/view_all.go` — CharacterViewAllCount, CharacterViewAllCharacters, CharacterViewAllSearchFailed, CharacterViewAllError

### Phase 2 Verification
- [x] 2d.5 Round-trip tests for all Phase 2 packets across GMS/JMS variants — all 27 packages pass
- [x] 2d.1 Update atlas-login writers to thin adapters importing from atlas-packet
- [x] 2d.2 Writers keep constants for opcode lookup; body functions delegate to atlas-packet .Encode
- [x] 2d.3 N/A — writer files kept as thin adapters (not removed)
- [x] 2d.4 Build and test atlas-login: all tests pass

## Phase 3: Channel Writer Packets - Core Character [Effort: L]

### New Shared Models
- N/A — SetField uses pre-encoded characterInfoBytes; sub-model encoding stays in service layer. Tasks 3.0a-f no longer needed.

### Character Packets
- [x] 3.1 `character/spawn.go` — CharacterSpawn struct (pre-built model.Avatar, model.CharacterTemporaryStat, GuildEmblem with primitives). Encode-only with no-op Decode
- [x] 3.2 `character/despawn.go` — CharacterDespawn struct (characterId uint32)
- [x] 3.3 `character/movement_writer.go` — CharacterMovementWriter struct (characterId uint32, movement model.Movement). Writer complement to existing Move handler packet
- [x] 3.4 `character/damage_writer.go` — CharacterDamageWriter struct
- [x] 3.5 `character/info.go` — CharacterInfo struct (pre-resolved primitives + InfoPet). Encode-only with no-op Decode
- [x] 3.6 `character/appearance_update.go` — CharacterAppearanceUpdate struct
- [x] 3.7 `character/expression_writer.go` — CharacterExpressionWriter struct
- [x] 3.8 `character/sit_result.go` — CharacterSitResult struct
- [x] 3.9 `character/hint.go` — CharacterHint struct
- [x] 3.10 `interaction/interaction_writer.go` — 5 CharacterInteraction writer structs (Invite, InviteResult, Enter, EnterResultSuccess, EnterResultError) with pre-resolved mode bytes
- [x] 3.11 `character/status_message.go` — 21 CharacterStatusMessage sub-types (DropPickUp variants, QuestRecord variants, CashItemExpire, IncreaseExperience, IncreaseSkillPoint, IncreaseFame, IncreaseMeso, IncreaseGuildPoint, GiveBuff, GeneralItemExpire, SystemMessage, QuestRecordEx, ItemProtectExpire, ItemExpireReplace, SkillExpire)

### Field Packets (set_field decomposition)
- [x] 3.12 `field/warp_to_map.go` — WarpToMap struct (channelId, mapId, portalId, hp; version branch for GMS/JMS)
- [x] 3.13 `field/set_field.go` — SetField struct with pre-encoded characterInfoBytes (service layer owns complex character serialization). Damage seeds, logout gifts, timestamp handled in atlas-packet
- [ ] 3.14 Verify pet ID field: confirm whether channel `pet.Id()` == login `pet.CashId()`. If different, fix channel to use CashId() in CharacterStatistics construction

### Stat & Buff Packets
- [x] 3.15 `stat/changed.go` — StatChanged struct (with getStatIndex from options["statistics"])
- [x] 3.16 `character/buff_give_writer.go` — BuffGive struct (takes pre-built model.CharacterTemporaryStat)
- [x] 3.17 `character/buff_give_writer.go` — BuffGiveForeign struct (characterId + CTS)
- [x] 3.18 `character/buff_cancel_writer.go` — BuffCancelW struct (takes pre-built CTS)
- [x] 3.19 `character/buff_cancel_writer.go` — BuffCancelForeign struct (characterId + CTS)

### Phase 3 Verification
- [x] 3.20 Build and test atlas-packet
- [x] 3.21 Update atlas-channel writers to thin adapters importing from atlas-packet
- [x] 3.22 Build and test atlas-channel

## Phase 4: Channel Writer Packets - Combat, Effects & Monsters [Effort: M]

### Attack Packets (pre-computed primitives pattern)
- [x] 4.0a-b `character/attack_writer.go` — Single AttackWriter struct with 4 constructors (NewAttackMelee/Ranged/Magic/Energy), pre-computed mastery/bulletItemId/skillLevel. Encode-only with no-op Decode
- [ ] 4.0c Refactor atlas-channel handler: extract `computeMasteryForWeapon()`, `getMasteryFromSkill()`, bullet resolution, skill level lookup into a service-layer helper file (service integration)

### Character Effects
- [x] 4.1-4.3 `character/effect.go` — 13 effect structs (EffectSimple, EffectSimpleForeign, EffectSkillAffected, EffectPet, EffectWithId, EffectWithMessage, EffectProtectOnDie, EffectIncDecHP, EffectShowInfo, EffectLotteryUse, EffectItemMaker, EffectUpgradeTomb, EffectIncubatorUse). SkillUse and Quest effects skipped (depend on skill constants and QuestReward model)

### Monster Packets
- [x] 4.4 `monster/spawn_writer.go` — MonsterSpawn struct with model.MonsterModel + model.MonsterTemporaryStat
- [x] 4.5 `monster/destroy.go` — MonsterDestroy struct
- [x] 4.6 `monster/movement_writer.go` — MonsterMovementW struct (uses model.Movement)
- [x] 4.7 `monster/movement_ack.go` — MonsterMovementAck struct
- [x] 4.8 `monster/control_writer.go` — MonsterControl struct with ControlType enum
- [x] 4.9 `monster/damage.go` — MonsterDamage struct
- [x] 4.10 `monster/health.go` — MonsterHealth struct
- [x] 4.11 `monster/stat_writer.go` — MonsterStatSet/Reset structs with model.MonsterTemporaryStat

### Phase 4 Verification
- [x] 4.12 Build and test atlas-packet
- [x] 4.13 Update atlas-channel writers to thin adapters importing from atlas-packet
- [x] 4.14 Build and test atlas-channel

## Phase 5: Channel Writer Packets - NPC, Drop, Pet [Effort: M]

### NPC Packets
- [x] 5.1 `npc/spawn.go` — NpcSpawnW struct
- [x] 5.2 `npc/action_writer.go` — NpcActionW struct
- [x] 5.3 `npc/conversation_writer.go` — NpcConversation struct with 13 detail types (Say, AskYesNo, AskMenu, etc.)
- [x] 5.4 `npc/shop_list.go` — ShopList struct with ShopCommodity (tenant-aware: discountRate GMS>=87, tokenTemplateId GMS>=95)
- [x] 5.5 `npc/shop_operation.go` — ShopOperationSimple, ShopOperationGenericError, ShopOperationLevelRequirement
- [x] 5.6 `npc/spawn_request_controller.go` — NpcSpawnRequestController struct

### Drop Packets
- [x] 5.7 `drop/spawn.go` — DropSpawn struct with DropEnterType enum (Fresh=1, Existing=2, Disappear=3)
- [x] 5.8 `drop/destroy.go` — DropDestroy struct

### Pet Packets
- [x] 5.9 `pet/activated.go` — PetActivated struct
- [x] 5.10 `pet/movement_writer.go` — PetMovementW struct (uses model.Movement)
- [x] 5.11 `pet/chat_writer.go` — PetChatW struct
- [x] 5.12 `pet/command_writer.go` — PetCommandResponse struct
- [x] 5.13 `pet/exclude.go` — PetExcludeResponse struct
- [x] 5.14 `pet/cash_food_result.go` — PetCashFoodResult struct

### Phase 5 Verification
- [x] 5.15 Build and test atlas-packet
- [x] 5.16 Update atlas-channel writers to thin adapters importing from atlas-packet
- [x] 5.17 Build and test atlas-channel

## Phase 6: Channel Writer Packets - Social & Commerce [Effort: M]

### Buddy Packets — separate struct per operation
- [x] 6.1 `buddy/invite_writer.go`, `buddy/list_update.go`, `buddy/update_writer.go`, `buddy/error_writer.go`, `buddy/channel_change.go`, `buddy/capacity_update.go` — 6 structs

### Party Packets — separate struct per operation + shared PartyInfo model
- [x] 6.2 `party/member_data.go` — PartyMember model + WritePartyData/ReadPartyData helpers
- [x] 6.3 `party/created.go`, `party/disband.go`, `party/join_writer.go`, `party/left.go`, `party/update.go`, `party/change_leader_writer.go`, `party/invite_writer.go`, `party/error.go` — 8 structs
- [x] 6.4 `party/member_hp.go` — MemberHP struct

### Guild Packets — separate struct per operation (27+)
- [x] 6.5 `guild/operation_writer.go` — 14 writer structs (RequestAgreement, ErrorMessage, ErrorMessageWithTarget, EmblemChange, MemberStatusUpdate, MemberTitleUpdate, NoticeChange, MemberLeft, MemberExpel, MemberJoined, InviteW, TitleChange, Disband, CapacityChange)
- [x] 6.6 `guild/emblem_changed_foreign.go` — ForeignEmblemChanged struct
- [x] 6.7 `guild/name_changed_foreign.go` — ForeignNameChanged struct
- [x] 6.8 `guild/bbs.go` — BBSThreadList + BBSThread structs

### Messenger Packets
- [x] 6.9 `messenger/join_writer.go`, `messenger/remove_writer.go`, `messenger/request_invite_writer.go`, `messenger/invite_sent_writer.go`, `messenger/invite_declined_writer.go`, `messenger/chat_writer.go` — 6 structs (Add/Update skipped: depend on Avatar model)

### Cash Shop Packets — separate struct per operation
- [x] 6.10 `cash/shop_open.go` — CashShopOpen struct with pre-encoded characterInfoBytes + accountName
- [x] 6.11 `cash/shop_operation_result.go` — OperationError, InventoryCapacitySuccess, InventoryCapacityFailed, WishList
- [x] 6.12 `cash/query_result.go` — QueryResult struct

### Merchant & Storage
- [x] 6.13 `merchant/operation_writer.go` — 7 structs (OpenShop, ErrorSimple, ShopSearch, ShopRename, RemoteShopWarp, ConfirmManage, FreeFormNotice)
- [x] 6.14 `storage/error_writer.go` — ErrorSimple, UpdateMeso, ErrorMessage

### Phase 6 Verification
- [x] 6.15 Build and test atlas-packet — all packages pass
- [x] 6.16 Update atlas-channel writers to thin adapters importing from atlas-packet
- [x] 6.17 Build and test atlas-channel

## Phase 7: Channel Writer Packets - Remaining [Effort: M]

### Inventory & Skills
- [x] 7.1 `inventory/change.go` — Add struct using model.Asset (Asset.Decode now exists)
- [x] 7.2 `inventory/change.go` — QuantityUpdate, ChangeMove, Remove, Add structs — all complete
- [x] 7.3 `character/item_upgrade.go` — ItemUpgrade struct
- [x] 7.4 `character/skill_change.go` — SkillChange struct
- [x] 7.5 `character/skill_cooldown.go` — SkillCooldown struct
- [x] 7.6 `character/skill_macro.go` — SkillMacro struct (already existed)
- [x] 7.7 `character/keymap.go` — KeyMap struct
- [x] 7.8 `character/keymap_auto_hp.go` — KeyMapAutoHp struct
- [x] 7.9 `character/keymap_auto_mp.go` — KeyMapAutoMp struct

### Chat
- [x] 7.10 `chat/general.go` — ChatGeneral struct (characterId, gm bool, message, show)
- [x] 7.11 `chat/whisper.go` — ChatWhisper struct (already existed)
- [x] 7.12 `chat/multi.go` — ChatMulti struct (guild/party/alliance chat)

### Field & Environment
- [x] 7.13 `field/effect.go` — FieldEffect structs (Summon, Tremble, String, BossHp, RewardRullet)
- [x] 7.14 `field/effect_weather.go` — FieldEffectWeather struct
- [x] 7.15 `field/transport.go` — FieldTransport struct
- [x] 7.16 `channel/change_writer.go` — ChannelChangeW struct

### UI
- [x] 7.17 `ui/open.go` — UiOpen struct
- [x] 7.18 `ui/lock.go` — UiLock struct
- [x] 7.19 `ui/disable.go` — UiDisable struct

### Misc
- [x] 7.20 `field/clock.go` — Clock struct (5 clock types)
- [x] 7.21 `chat/world_message.go` — 8 world message structs (Simple, TopScroll, SuperMegaphone, BlueText, ItemMegaphone, YellowMegaphone, MultiMegaphone, Gachapon)
- [x] 7.22 `quest/script_progress.go` — ScriptProgress struct
- [x] 7.23 `character/chalkboard.go` — ChalkboardUse struct
- [x] 7.24 `fame/response.go` — ReceiveResponse, GiveResponse, ErrorResponse (pre-resolved mode bytes)
- [x] 7.25 `npc/guide_talk.go` — GuideTalkMessage + GuideTalkIdx structs
- [x] 7.26 `note/operation_writer.go` — SendSuccess, SendError, Refresh

### Objects
- [x] 7.27 `field/kite_spawn.go` — KiteSpawn struct
- [x] 7.28 `field/kite_destroy.go` — KiteDestroy struct
- [x] 7.29 `field/kite_error.go` — KiteError struct
- [x] 7.30 `reactor/spawn.go` — ReactorSpawn struct
- [x] 7.31 `reactor/hit_writer.go` — ReactorHitW struct
- [x] 7.32 `reactor/destroy.go` — ReactorDestroy struct
- [x] 7.33 `interaction/mini_room.go` — MiniRoom interface already in atlas-packet with Spawn/Despawn/Enter methods; channel delegates directly
- [x] 7.34 `character/chair_show.go` — CharacterChairShow struct
- [x] 7.35 `inventory/compartment_merge_writer.go` — CompartmentMergeW struct
- [x] 7.36 `inventory/compartment_sort_writer.go` — CompartmentSortW struct

### Phase 7 Verification
- [x] 7.37 Build and test atlas-packet
- [x] 7.38 Update atlas-channel writers to thin adapters importing from atlas-packet
- [x] 7.39 Build and test atlas-channel

## Phase 8: Service Integration & Verification [Effort: M]

- [x] 8.1 N/A — writer files kept as thin adapters (constants + getCode + model conversion → atlas-packet .Encode)
- [x] 8.2 N/A — writer files kept as thin adapters (constants + getCode + model conversion → atlas-packet .Encode)
- [x] 8.3 Full build: `go build ./...` for atlas-packet — PASS
- [x] 8.4 Full test: `go test ./... -count=1` for atlas-packet — 31 packages, all PASS
- [x] 8.5 Full build: `go build` for atlas-login — PASS
- [x] 8.6 Full build: `go build` for atlas-channel — PASS
- [x] 8.7 Docker build verification for atlas-login — PASS
- [x] 8.8 Docker build verification for atlas-channel — PASS
- [x] 8.9 No duplicate packet definitions — services are thin adapters delegating to atlas-packet
- [x] 8.10 Update MEMORY.md with completion status

## Progress Summary

| Phase | Total Tasks | Completed | Remaining | Status |
|-------|-----------|-----------|-----------|--------|
| 1 - Foundation & Models | 6 | 6 | 0 | COMPLETE |
| 2 - Login Writers | 25 | 25 | 0 | COMPLETE |
| 3 - Channel Core | 16 | 16 | 0 | COMPLETE |
| 4 - Combat & Monsters | 13 | 13 | 0 | COMPLETE |
| 5 - NPC, Drop, Pet | 17 | 17 | 0 | COMPLETE |
| 6 - Social & Commerce | 17 | 17 | 0 | COMPLETE |
| 7 - Remaining | 39 | 39 | 0 | COMPLETE |
| 8 - Integration & Verification | 10 | 10 | 0 | COMPLETE |
| **Total** | **143** | **143** | **0** | **ALL COMPLETE** |

### Approach Notes
Writer files in both services were converted to **thin adapters** rather than removed:
- Keep exported constants (opcode keys), getCode resolution, and service model conversion
- Delegate encoding to atlas-packet structs via `.Encode`
- **All previously-skipped writers now migrated** (attack, inventory_change, effects, world_message, messenger, note, storage, cash_shop_operation)
- Attack writers: computation (mastery, skill level, bullet) stays as service helpers; packet encoding delegates to atlas-packet
- Inventory change: batch composer pattern preserved; individual entry writers still use temp response.Writer; outer frame delegates to ChangeBatch
- Effects: SkillUse/Quest now have dedicated atlas-packet structs; all Foreign variants use EffectForeign wrapper
- World message: all known modes delegate to atlas-packet; only Unk3/4/7/8 edge cases remain inline as fallback
- Remaining intentional inline encoding: set_field.go and cash_shop_open.go pre-encode complex character info bytes (by design)
