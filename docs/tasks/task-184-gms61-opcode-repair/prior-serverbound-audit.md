# gms_61 serverbound opcode audit — full 113-row sweep

## Session

- IDB session id: `965202bf`
- Binary: `E:\Programs\Nexon\IDBs_v9\GMS\v61\GMS_v61.1_U_DEVM.exe.i64` (`GMS_v61.1_U_DEVM.exe.i64`)
- Confirmed live via `idb_list` immediately before starting this pass (matched by binary name, not port — ports/sessions rotate per project convention).
- Method: bulk `func_query` name-regex sweeps (`Send.*`, `.*Attack.*`, `.*Move.*`, `.*Chat.*`, `.*Party.*`, `.*Guild.*`, login/char-select keywords, etc.) to build a name→address map of client SEND functions, then targeted `decompile` (and, where the pseudocode was too large/truncated, `disasm`) of each candidate to read the literal `COutPacket::COutPacket(this, N)` constructor argument — the true wire opcode. No opcode enum exists in this IDB (`type_query kind:enum` returned none besides COM/CRT enums), so every value below is a directly observed integer literal, not inferred from a symbol.

No template file was edited. This is a read-only investigation.

## Result table — all 113 template rows

Legend: **MATCH** = template opcode equals observed client wire opcode. **MISMATCH** = template opcode is wrong; corrected value given. **UNRESOLVED** = could not pin the client send path in this pass (see notes for what was tried). A `†` marks a *structural* mismatch: the template's numeric opcode is a real value the client does send, but for a **different** feature than the handler's name implies (see §3).

| template_opcode | handler_name | IDB_opcode (truth) | verdict | evidence |
|---|---|---|---|---|
| 0x01 | LoginHandle | ? | UNRESOLVED | `SendLoginPacket@CLogin`(0x564dc9) and `SendCheckPasswordPacket@CLogin`(0x564418) both exist as candidates; neither's ctor arg was extracted this pass |
| 0x04 | ServerListRequestHandle | ? | UNRESOLVED | no dedicated send function located |
| 0x05 | CharacterListWorldHandle | ? | UNRESOLVED | no dedicated send function located |
| 0x06 | ServerStatusHandle | ? | UNRESOLVED | no dedicated send function located |
| 0x07 | AcceptTosHandle | ? | UNRESOLVED | `OnAcceptLicense@CLogin`(0x56842a) is the likely UI trigger; ctor arg not extracted |
| 0x08 | SetGenderHandle | **0x08** | MATCH | `SendSetGenderPacket@CLogin`@0x5686a4: `COutPacket::COutPacket(v3, 8)`; confirmed again via `SendCancelGenderPacket@CLogin`@0x5686ff: `COutPacket::COutPacket(v2, 8)` (submode byte 0 vs a2) |
| 0x09 | AfterLoginHandle | ? | UNRESOLVED | no dedicated send function located |
| 0x0A | RegisterPinHandle | ? | UNRESOLVED | `EnterPinCode@CPinCodeDlg`(0x585dc5) is the likely UI trigger; not decompiled |
| 0x0B | ServerListRequestHandle | ? | UNRESOLVED | duplicate handler name of 0x04 row; same gap |
| 0x0D | CharacterViewAllHandle | **0x0D** | MATCH | `SendViewAllCharPacket@CLogin`@0x567117: unconditional second send `COutPacket::COutPacket(v3, 13)` (a conditional first send uses opcode 12/0x0C, which has no template row — informational only, not a mismatch since nothing claims 0x0C) |
| 0x0E | CharacterViewAllSelectedHandle | ? | UNRESOLVED | candidate `SendSelectCharPacketByVAC@CLogin`(0x5650b6); ctor arg not extracted |
| 0x0F | CharacterViewAllPongHandle | ? | UNRESOLVED | no dedicated send function located |
| 0x13 | CharacterSelectedHandle | **0x13** | MATCH | `SendSelectCharPacket@CLogin`@0x564f79: `COutPacket::COutPacket(v4, 19)` |
| 0x14 | CharacterLoggedInHandle | ? | UNRESOLVED | no dedicated send function located |
| 0x15 | CharacterCheckNameHandle | **0x15** | MATCH | `sub_565537` (`CLogin` check-duplicate-name path)@0x565537: `COutPacket::COutPacket(v6, 21)` |
| 0x16 | CreateCharacterHandle | **0x16** | MATCH | `sub_5653E9` (`SendNewCharPacket@CLogin`)@0x5653e9: `COutPacket::COutPacket(v9, 22)` |
| 0x17 | DeleteCharacterHandle | ? | UNRESOLVED | only the receive side (`OnDeleteCharacterResult@CLogin`) was found by name; no `Send*Delete*` function exists — likely an inline literal in char-select UI code, not located |
| 0x18 | PongHandle | ? | UNRESOLVED | no dedicated send function located (distinct from the alive/keepalive path on `CClientSocket`, which was not chased down) |
| 0x19 | StartErrorHandle | ? | UNRESOLVED | no dedicated send function located |
| 0x23 | MapChangeHandle | **0x23** | MATCH | `CField::SendTransferFieldRequest`@0x4e8f58: `COutPacket::COutPacket(v16, 35)` |
| 0x24 | ChannelChangeHandle | **0x24** | MATCH | `CField::SendTransferChannelRequest`@0x4e90ab: `COutPacket::COutPacket(v16, 36)` |
| 0x25 | CashShopEntryHandle | **0x25** | MATCH | `CWvsContext::SendMigrateToShopRequest`@0x82e811: `COutPacket::COutPacket(v17, v15)` where `v15=37` on the taken path |
| 0x26 | CharacterMoveHandle | ? | UNRESOLVED | player movement appears to flow through `CMovePath::Encode`/`Flush` (0x5e298d/0x5e2ca3), called from inside `CUserLocal::Update` (huge, 0x162a bytes); the actual COutPacket ctor call site for the player's own move-send was not isolated this pass |
| 0x27 | CharacterChairInteractionHandle | **0x27** | MATCH | `CWvsContext::SendGetUpFromChairRequest`@0x8374fe: `COutPacket::COutPacket(v6, 39)` |
| 0x28 | CharacterChairPortableHandle | **0x28** | MATCH | `sub_8372C1` (sit-on-portable-chair path)@0x8372c1: `COutPacket::COutPacket(v15, 40)` |
| 0x29 | CharacterMeleeAttackHandle | ? | UNRESOLVED | `CUserLocal::TryDoingMeleeAttack`@0x7a45f1 (0x2017 bytes) is the right function by name/behavior, but its ctor call site was not located within budget (very large function, multiple COutPacket-adjacent call sites) |
| 0x2A | CharacterRangedAttackHandle | ? | UNRESOLVED | `TryDoingShootAttack@CUserLocal`@0x7a67e9 identified by name only |
| 0x2B | CharacterMagicAttackHandle | ? | UNRESOLVED | `CUserLocal::TryDoingMagicAttack`@0x7a8572 identified by name only |
| 0x2C | CharacterTouchAttackHandle | ? | UNRESOLVED | no distinct send function separated from the melee/body-attack family located |
| 0x2D | CharacterDamageHandle | ? | UNRESOLVED | no dedicated send function located (distinct from `CMob::OnDamaged`, which is receive-side) |
| 0x2E | CharacterChatGeneralHandle | ? | UNRESOLVED | `SendChatMsg@CField`@0x4e73af is the clear name-match candidate; ctor arg not extracted this pass |
| 0x2F | ChalkboardCloseHandle | ? | UNRESOLVED | no dedicated send function located |
| 0x30 | CharacterExpressionHandle | **0x30** | MATCH | `CWvsContext::SendEmotionChange`@0x845e8f: `COutPacket::COutPacket(v11, 48)` |
| 0x34 | MobBanishPlayer | **0x34** | MATCH | `CUserLocal::SendBanMapByMobRequest`@0x7aed83: `COutPacket::COutPacket(v3, 52)` |
| 0x35 | MonsterBookCover | ? | UNRESOLVED | no dedicated send function located |
| 0x36 | NPCStartConversationHandle | ? | UNRESOLVED | no dedicated send function located (NPC talk-start likely triggered from click handlers not swept this pass) |
| 0x38 | NPCContinueConversationHandle | ? | UNRESOLVED | `CScriptMan::OnAsk*` family found are all receive-side (server→client script prompts); the client's "continue" reply send was not isolated |
| 0x39 | NPCShopHandle | **0x39** | MATCH | `CShopDlg` dispatcher confirmed 3 ways: buy `sub_646C41`@0x646c41 `COutPacket(v28,57)` submode 0; sell `sub_646EAE`@0x646eae `COutPacket(v19,57)` submode 1; both = opcode 57 = 0x39 |
| 0x3A | StorageOperationHandle | ? | UNRESOLVED | `CTrunkDlg` was checked and **ruled out** — its Get/PutItem sends both resolve to opcode 0x6F (see CharacterInteractionHandle row), i.e. `CTrunkDlg` is a trade-room item-shuttle dialog, not NPC storage. `CStoreBankDlg` (`SendCalculateFeeRequest`@0x67547c, `SendGetAllRequest`@0x6754e1) both send opcode 60 (0x3C), a value with **no row at all** in the 113-handler list — a different, unregistered feature, not evidence for 0x3A. No `CStorage`/`Warehouse`-named class exists in this IDB. True NPC-storage send path not found |
| 0x3E | HiredMerchantOperationHandle | 0x3B? (low confidence) | UNRESOLVED (flagged) | `SendEntrustedShopCheckRequest@CWvsContext`@0x848a01: `COutPacket::COutPacket(v25, 59)` = 0x3B. "Entrusted Shop" is the formal KMS term for Hired Merchant, so this is a plausible lead — but it may be a distinct "pre-check" opcode separate from the main operation dispatcher, so **not asserted as a firm mismatch** |
| 0x3F | OwlWarpHandle | 0x4C? (low confidence) | UNRESOLVED (flagged) | `SendShopScannerItemUseRequest@CWvsContext`@0x832680: `COutPacket::COutPacket(v3, 76)` = 0x4C. Project memory pairs "owl/shop-scanner" as one legacy feature, but this could equally be an unrelated item-use opcode distinct from an NPC-driven owl-warp opcode — **not asserted as a firm mismatch** (see also 0x4C collision note below) |
| 0x40 | CompartmentMergeHandle | **0x40** | MATCH | `CWvsContext::SendGatherItemRequest`@0x8314d0: `COutPacket::COutPacket(v8, 64)` |
| 0x41 | CompartmentSortHandle | **0x41** | MATCH | `sub_831564` (sort-item path)@0x831564: `COutPacket::COutPacket(v8, 65)` |
| 0x44 | CharacterItemCancelHandle | **0x44** | MATCH | `sub_831BB7` (`SendStatChangeItemCancelRequest`)@0x831bb7: `COutPacket::COutPacket(v2, 68)` |
| 0x46 | CharacterInventoryMoveHandle | ? | UNRESOLVED — **flagged at risk** | true "move item between inventory slots" send function not located (searched `.*Inventory.*`, `.*ItemMove.*`, `.*MoveItem.*` — only receive-side `OnInventoryOperation`/`OnInventoryGrow` and an unrelated `CPersonalShopDlg::MoveItemToInventory` turned up). **However**, opcode 0x46 is independently confirmed as the real wire value for `SendMobSummonItemUseRequest` (Summon Bag item use, see 0x4A row below) — if the server dispatches 0x46 to inventory-move logic, a Summon Bag item-use packet would be misdecoded. This collision is real; the *correct* inventory-move opcode remains unknown |
| 0x47 | CharacterItemUseHandle | **0x43** | MISMATCH | `CWvsContext::SendStatChangeItemUseRequest`@0x831880: `COutPacket::COutPacket(v13, 67)` = 0x43 (re-confirms prior finding) |
| 0x49 | CharacterCashItemUseHandle | **0x49** | MATCH | `CWvsContext::SendConsumeCashItemUseRequest`@0x832a5d — pseudocode too large to render fully (15676-byte function); `disasm` at the ctor call site (function+0x66) shows `push 49h` immediately before `call ??0COutPacket@@QAE@J@Z` = 0x49 exactly |
| 0x4A | CharacterItemUseSummonBagHandle | 0x46 (of `SendMobSummonItemUseRequest`) † | MISMATCH (structural) | Two distinct client functions land near this slot: `SendMobSummonItemUseRequest@CWvsContext`@0x831c83 (`COutPacket::COutPacket(v16, 70)` = 0x46, item-prefix check `a3/1000==2109 \|\| a3==2100067` — genuinely "summon bag" by item id) vs. `SendBridleItemUseRequest@CWvsContext`@0x832005 (`COutPacket::COutPacket(v55, 74)` = **0x4A exactly**, item-prefix check `a3/10000==227` — the Bridle/pet-leash item, an unrelated feature). Template's 0x4A slot is real and IS sent by the client — just for **Bridle**, not Summon Bag. The true Summon Bag opcode (0x46) collides with `CharacterInventoryMoveHandle`'s row (see above) |
| 0x4B | PetFoodHandle | **0x47** | MISMATCH | `CWvsContext::SendPetFoodItemUseRequest`@0x831de9: `COutPacket::COutPacket(v20, 71)` = 0x47 (re-confirms prior finding) |
| 0x4C | MountFoodHandle | 0x48 (of `SendTamingMobFoodItemUseRequest`) † | MISMATCH (structural) | `SendTamingMobFoodItemUseRequest@CWvsContext`@0x831f44 (mount/taming-mob food, item-prefix `a3/10000==226`): `COutPacket::COutPacket(v10, 72)` = 0x48 — **no template row claims 0x48 at all**. Meanwhile `SendShopScannerItemUseRequest`@0x832680 sends exactly **0x4C** (item-prefix `a2/10000==231`, an unrelated shop-scanner/owl feature). Template's 0x4C slot is real and IS sent by the client — just for Shop Scanner, not Mount Food |
| 0x4D | TeleportRockUseHandle | **0x4D** | MATCH | `sub_8327DB` (`CWvsContext::SendMapTransferItemUseRequest`, item-prefix `a3/10000==232`)@0x8327db: `COutPacket::COutPacket(v10, 77)` = 0x4D |
| 0x4E | CharacterItemUseTownScrollHandle | **0x4E** | MATCH | `sub_841AA5` (`SendPortalScrollUseRequest`)@0x841aa5: `COutPacket::COutPacket(v31, 78)` = 0x4E |
| 0x4F | CharacterItemUseScrollHandle | ? | UNRESOLVED | not distinguished from the town-scroll/teleport-rock cluster above; no separate send function identified |
| 0x50 | CharacterDistributeApHandle | ? | UNRESOLVED | no dedicated send function located |
| 0x51 | CharacterHealOverTimeHandle | **0x51** | MATCH | `CWvsContext::SendStatChangeRequest`@0x8421f0: `COutPacket::COutPacket(v5, 81)` = 0x51 |
| 0x52 | CharacterDistributeSpHandle | ? | UNRESOLVED | no dedicated send function located |
| 0x54 | CharacterBuffCancel | **0x54** | MATCH | `CUserLocal::SendSkillCancelRequest`@0x7ba445: `COutPacket::COutPacket(v8, 84)` = 0x54 |
| 0x55 | CharacterSkillPrepareHandle | **0x55** | MATCH | `CUserLocal::DoActiveSkill_Prepare`@0x7b8001: `COutPacket::COutPacket(v42, 85)` = 0x55 |
| 0x56 | CharacterDropMesoHandle | **0x56** | MATCH | `sub_8459DD` (`SendDropMoneyRequest`)@0x8459dd: `COutPacket::COutPacket(v8, 86)` = 0x56 |
| 0x57 | FameChangeHandle | **0x57** | MATCH | `sub_845A65` (`SendGivePopularityRequest`)@0x845a65: `COutPacket::COutPacket(v10, 87)` = 0x57 |
| 0x59 | CharacterInfoRequestHandle | **0x59** | MATCH | `sub_845B68` (`SendCharacterInfoRequest`)@0x845b68: `COutPacket::COutPacket(v11, 89)` = 0x59 |
| 0x5A | CharacterUseSkillHandle | 0x53 (of `SendSkillUseRequest`) † | MISMATCH (structural) | `sub_7BA213` (`SendSkillUseRequest`, the real "cast a skill" send)@0x7ba213: `COutPacket::COutPacket(v27, 83)` = **0x53 — unclaimed by any template row**. Meanwhile `sub_845C36` (`SendSkillResetItemUseRequest`, an item that resets skill points — an unrelated feature)@0x845c36: `COutPacket::COutPacket(v21, 90)` = **0x5A exactly**. Template's 0x5A slot is real and IS sent by the client — just for the skill-reset item, not for casting a skill |
| 0x5C | PortalScriptHandle | ? | UNRESOLVED | no dedicated send function located |
| 0x5E | TeleportRockAddMapHandle | **0x5E** | MATCH | `sub_8478EA` (`CWvsContext::SendMapTransferRequest`)@0x8478ea: `COutPacket::COutPacket(v4, 94)` = 0x5E |
| 0x62 | QuestActionHandle | ? | UNRESOLVED | only receive-side (`CQuestMan`/`CQuest`) and `CUserLocal::OnQuestResult` (receive) were found; the client's quest-action send was not isolated |
| 0x68 | SueCharacter | ? | UNRESOLVED | only the receive side `OnSueCharacterResult@CWvsContext`@0x84a04e was found by name; no `Send*Sue*` function exists |
| 0x6B | CharacterMultiChatHandle | ? | UNRESOLVED | `SendGroupMessage@CUIStatusBar`@0x74467d is a plausible name-match candidate; ctor arg not extracted |
| 0x6C | CharacterChatWhisperHandle | **0x6C** | MATCH | `sub_4E8635` (`SendChatMsgWhisper@CField`)@0x4e8635: `COutPacket::COutPacket(v12, 108)` = 0x6C |
| 0x6D | CharacterSpouseChatHandle | ? | UNRESOLVED | no dedicated send function located |
| 0x6E | MessengerOperationHandle | ? | UNRESOLVED | `CUIMessenger` class located (`OnCreate`/`OnInvite`/`ProcessChat`/`OnPacket`), but none of its found members are clearly a serverbound send with an extracted ctor value |
| 0x6F | CharacterInteractionHandle | **0x6F** | MATCH | Confirmed three ways, all opcode 111 (0x6F): `sub_4E87E1` (`SendInviteTradingRoomMsg@CField`) submodes 0/3/0 then 2; `sub_68C7E6` (`SendPutItemRequest@CTrunkDlg`) submode 0xE; `sub_68CA10` (`SendGetItemRequest@CTrunkDlg`) submode 0xF |
| 0x70 | PartyOperationHandle | **0x70** | MATCH | Confirmed three ways, all opcode 112 (0x70): `sub_4E898B` (`SendCreateNewPartyMsg@CField`) submode 1; `sub_4E8A90` (`SendWithdrawPartyMsg@CField`) submode 2; `CField::SendJoinPartyMsg`@0x4e8b29 submode 4 |
| 0x71 | PartyInviteRejectHandle | ? | UNRESOLVED | no distinct "reject invite" opcode found separate from the 0x70 dispatcher above; may in fact be a submode of 0x70 rather than its own opcode — not confirmed either way |
| 0x72 | GuildOperationHandle | **0x72** | MATCH | `CField::SendCreateGuildAgreeMsg`@0x4e9260: `COutPacket::COutPacket(v4, 114)` = 0x72, submode 0x1E |
| 0x76 | BuddyOperationHandle | **0x76** | MATCH | `sub_4E9C03` (`SendSetFriendMsg@CField`)@0x4e9c03: `COutPacket::COutPacket(v17, 118)` = 0x76, submode 1 |
| 0x77 | NoteOperationHandle | ? | UNRESOLVED | no dedicated send function located |
| 0x79 | UseDoor | ? | UNRESOLVED | no dedicated send function located |
| 0x7B | CharacterKeyMapChangeHandle | **0x7B** | MATCH | `CFuncKeyMappedMan::SaveFuncKeyMap`@0x51ac0d: `COutPacket::COutPacket(&v7, 123)` = 0x7B |
| 0x7C | RPSActionHandle | **0x7C** | MATCH | `sub_63C4B4` (`SendSelection@CRPSGameDlg`)@0x63c4b4: `COutPacket::COutPacket(v6, 124)` = 0x7C |
| 0x7E | AdminCommand | ? | UNRESOLVED | no dedicated send function located (GM/admin commands likely piggyback on the chat-slash path, `SendChatMsgSlash@CField`@0x4e7469, not decompiled — it is a 0x10a4-byte function) |
| 0x7F | WeddingAction | ? | UNRESOLVED | only receive-side `OnWeddingProgress@CField_Wedding` / `OnWeddingGiftResult@CWvsContext` found; no `Send*Wedding*` function exists |
| 0x80 | WeddingTalk | ? | UNRESOLVED | same as above |
| 0x86 | GuildBBSHandle | **0x86** | MATCH | `sub_6BB596` (`SendLoadListRequest@CUIGuildBBS`)@0x6bb596: `COutPacket::COutPacket(v3, 134)` = 0x86, submode 2 |
| 0x87 | EnterMtsHandle | **0x87** | MATCH | `CWvsContext::SendMigrateToITCRequest`@0x839b94: `COutPacket::COutPacket(v13, 135)` = 0x87 (ITC = the pre-rename name for the MTS/trading-center feature in this client build; opcode matches exactly) |
| 0x88 | MobCrcKeyChangedReply | ? | UNRESOLVED | only receive-side `OnMobCrcKeyChanged@CMobPool`@0x5d4d23 found; the client's reply-send was not isolated |
| 0x8A | PetMovementHandle | ? | UNRESOLVED | `OnMove@CPet`@0x613522 is receive-side; no distinct pet-move send function located (may share the generic `CMovePath` mechanism like player movement) |
| 0x8B | PetChatHandle | ? | UNRESOLVED | `ParseCommand@CPet`/`DoAction@CPet` handle local pet commands; the network send for pet "chat" bubble was not isolated |
| 0x8C | PetCommandHandle | ? | UNRESOLVED | same family as above; not isolated |
| 0x8D | PetDropPickUpHandle | **0x8D** | MATCH | `sub_6149DB` (`SendDropPickUpRequest@CPet`)@0x6149db: `COutPacket::COutPacket(v10, 141)` = 0x8D |
| 0x8E | PetItemUseHandle | ? | UNRESOLVED | `SendStatChangeItemUseRequestByPetQ@CWvsContext`@0x831ab9 sends opcode 142 (0x8E) — but this is the "use consumable item via pet auto-queue," a plausible but not certain match for "PetItemUseHandle" (could instead be a distinct feature); noted as a lead, not confirmed |
| 0x8F | PetItemExcludeHandle | **0x8F** | MATCH | `sub_61504D` (`SendUpdateExceptionListRequest@CPet`)@0x61504d: `COutPacket::COutPacket(v7, 143)` = 0x8F |
| 0x92 | SummonMoveHandle | ? | UNRESOLVED | `OnMove@CSummonedPool` is receive-side; summon movement likely rides the same generic `CMovePath` mechanism, send site not isolated |
| 0x93 | SummonAttackHandle | ? | UNRESOLVED | `CSummoned::TryDoingAttackManual`/`AttackToTargetMob` found by name only; not decompiled |
| 0x94 | SummonDamageHandle | ? | UNRESOLVED | no dedicated send function located |
| 0x9B | MonsterMovementHandle | ? | UNRESOLVED | this is properly a clientbound feature in most versions (server tells client how mobs move) — no serverbound send counterpart found, consistent with there being none |
| 0x9D | MobDropPickupRequest | ? | UNRESOLVED | no dedicated send function located, distinct from the player/pet drop-pickup handlers already confirmed |
| 0x9E | FieldDamageMob | ? | UNRESOLVED | no dedicated send function located |
| 0x9F | MonsterDamageFriendlyHandle | ? | UNRESOLVED | no dedicated send function located |
| 0xA0 | MonsterBomb | ? | UNRESOLVED | no dedicated send function located |
| 0xA1 | MobDamageMob | ? | UNRESOLVED | no dedicated send function located |
| 0xA4 | NPCActionHandle | ? | UNRESOLVED | no dedicated send function located |
| 0xA9 | DropPickUpHandle | **0xA9** | MATCH | `sub_8316B8` (`SendDropPickUpRequest@CWvsContext`)@0x8316b8: `COutPacket::COutPacket(v9, 169)` = 0xA9 |
| 0xAC | ReactorHitHandle | ? | UNRESOLVED | `CReactorPool` located, but all found members (`OnReactorChangeState`/`OnReactorEnterField`/`OnReactorLeaveField`/`FindHitReactor`) are receive-side or local-only; the hit-send was not isolated |
| 0xB0 | Snowball | ? | UNRESOLVED | `CField_SnowBall::BasicActionAttack`@0x50c0dd found by name only; not decompiled |
| 0xB1 | LeftKnockback | ? | UNRESOLVED | no dedicated send function located |
| 0xB2 | Coconut | ? | UNRESOLVED | `CField_Coconut` class exists but only its constructor was enumerated; no send located |
| 0xB4 | GuildBoss | ? | UNRESOLVED | `CField_GuildBoss::BasicActionAttack`@0x4ff6d2 found by name only; not decompiled |
| 0xB7 | MonsterCarnival | ? | UNRESOLVED | `CField_MonsterCarnival*` constructors found only; no send located |
| 0xC3 | CashShopCheckWalletHandle | ? | UNRESOLVED | `TrySendQueryCashRequest@CCashShop`@0x45c33e is a plausible candidate; not decompiled |
| 0xC4 | CashShopOperationHandle | ? | UNRESOLVED | many `CCashShop::On*` (Buy/Gift/etc.) submode handlers found by name, all clearly a shared-opcode dispatcher family, but the ctor value was not extracted from any of them this pass |
| 0xD5 | ItcStatusChargeHandle | ? | UNRESOLVED | `CITC::OnStatusCharge`@0x528ed7 found by name only (may be receive-side, "On" prefix) |
| 0xD6 | ItcQueryCashRequestHandle | ? | UNRESOLVED | `CITC::TrySendQueryCashRequest`@0x5291cc is a plausible candidate; not decompiled |
| 0xD7 | ItcOperationHandle | ? | UNRESOLVED | many `CITC::On*` submode handlers found by name (mirrors the 0xC4 CashShop family); ctor value not extracted |

## Summary

- **MATCH: 38** — `0x08, 0x0D, 0x13, 0x15, 0x16, 0x23, 0x24, 0x25, 0x27, 0x28, 0x30, 0x34, 0x39, 0x40, 0x41, 0x44, 0x49, 0x4D, 0x4E, 0x51, 0x54, 0x55, 0x56, 0x57, 0x59, 0x5E, 0x6C, 0x6F, 0x70, 0x72, 0x76, 0x7B, 0x7C, 0x86, 0x87, 0x8D, 0x8F, 0xA9`
- **MISMATCH (confirmed, corrected value known): 5**
  - `CharacterItemUseHandle`: `0x47 -> 0x43`
  - `PetFoodHandle`: `0x4B -> 0x47`
  - `CharacterItemUseSummonBagHandle`: `0x4A -> 0x46` (the real Summon Bag opcode; note 0x4A itself is genuinely sent by the client, but for **Bridle** item use, not Summon Bag)
  - `MountFoodHandle`: `0x4C -> 0x48` (the real Mount/Taming-food opcode; note 0x4C itself is genuinely sent by the client, but for **Shop Scanner** item use, not Mount Food)
  - `CharacterUseSkillHandle`: `0x5A -> 0x53` (the real skill-cast opcode; note 0x5A itself is genuinely sent by the client, but for the **skill-reset item**, not for casting a skill)
- **UNRESOLVED: 70** (includes 2 low-confidence leads noted inline for `HiredMerchantOperationHandle`→0x3B? and `OwlWarpHandle`→0x4C?, and one at-risk flag for `CharacterInventoryMoveHandle` colliding with the true Summon Bag opcode 0x46)

## Observed shift pattern

There is **no uniform shift** across the table. The corruption is confined to a single item-use-adjacent cluster in the `0x44`–`0x5A` range, and even within that cluster it is **not a simple arithmetic shift** — it's a mix of two distinct defect types:

1. **Simple wrong-value (classic shift, -4):** `CharacterItemUseHandle` (0x47→0x43) and `PetFoodHandle` (0x4B→0x47) are each off by exactly -4, with no other handler occupying their true slot.
2. **Cross-wired slots (structural, not a fixed shift):** `CharacterItemUseSummonBagHandle` (0x4A), `MountFoodHandle` (0x4C), and `CharacterUseSkillHandle` (0x5A) each have a template opcode that the client genuinely sends — just for an unrelated feature (Bridle, Shop-Scanner-item-use, and Skill-Reset-item-use, respectively). Each of these three real intended features (Summon Bag, Mount Food, Skill Use) instead sends an opcode that is either already claimed by a different handler (0x46, claimed by `CharacterInventoryMoveHandle`) or claimed by nothing in the template at all (0x48, 0x53).

Everywhere else the table was sampled — the login/char-select boundary (0x08–0x16), the movement/map/social boundary (0x23–0x30), the back half of the item-use cluster and beyond (0x51, 0x54–0x59, 0x5E), and the entire social/guild/party/messenger/key-map/minigame tail (0x6C–0x8F, 0xA9) — every opcode independently checked came back an **exact match**, including three-way cross-checks within shared-opcode submode dispatchers (NPCShop@0x39, CharacterInteraction@0x6F, PartyOperation@0x70). This strongly suggests the corruption is localized to the `0x44`–`0x5A` item-use/skill-use neighborhood and does not extend to the rest of the table — but roughly 62% of the 113 rows (70/113) remain UNRESOLVED because no send function could be pinned down in this pass (see per-row notes for what was searched), so this conclusion should be treated as a strong hypothesis for the checked ~46%, not a proven fact for the whole file.

## UNRESOLVED handlers (full list, 70)

0x01, 0x04, 0x05, 0x06, 0x07, 0x09, 0x0A, 0x0B, 0x0E, 0x0F, 0x14, 0x17, 0x18, 0x19, 0x26, 0x29, 0x2A, 0x2B, 0x2C, 0x2D, 0x2E, 0x2F, 0x35, 0x36, 0x38, 0x3A, 0x3E, 0x3F, 0x46, 0x4F, 0x50, 0x52, 0x5C, 0x62, 0x68, 0x6B, 0x6D, 0x6E, 0x71, 0x77, 0x79, 0x7E, 0x7F, 0x80, 0x88, 0x8A, 0x8B, 0x8C, 0x8E, 0x92, 0x93, 0x94, 0x9B, 0x9D, 0x9E, 0x9F, 0xA0, 0xA1, 0xA4, 0xAC, 0xB0, 0xB1, 0xB2, 0xB4, 0xB7, 0xC3, 0xC4, 0xD5, 0xD6, 0xD7

(0x0C is not in this list — it is not a template row at all, just an incidental opcode observed inside the `CharacterViewAllHandle` send function.)
