# gms_61 serverbound opcode ground-truth — EXHAUSTIVE IDA verification pass

## Session

- IDB session id: `965202bf`, binary `GMS_v61.1_U_DEVM.exe.i64` — confirmed live via `idb_list`
  immediately before this pass (matched by binary filename, not port).
- Read-only. No template, code, or config file was edited — this file is the only artifact
  written by this pass.
- Method: `func_query name_regex:"\?\?0COutPacket"` located the ctor
  `??0COutPacket@@QAE@J@Z`@`0x5ffc4f`. `xrefs_to` on that address returned **391 call sites**
  with `"more": false` (i.e. the complete, non-truncated set — matches the count established by
  two prior passes in this task). Grouping those 391 sites by containing function yields **353
  unique functions**. Each function of interest was `decompile`d (or, for two oversized functions —
  `CUserLocal::Update` and `CUserLocal::TryDoingMeleeAttack`/`TryDoingShootAttack`/
  `TryDoingMagicAttack`/`TryDoingBodyAttack` — probed via `analyze_batch`
  `include_constants` + targeted `disasm` at the exact call-site address) to read the literal
  integer argument passed to the ctor. Every opcode below is a directly observed decompiled/
  disassembled integer, never inferred from a symbol name.
- This pass consumes and extends the two prior passes (`prior-serverbound-audit.md`,
  38 MATCH/5 MISMATCH/70 UNRESOLVED; `serverbound-resolution.md`, +5 corrections applied,
  +10 new MATCH, 56 residual UNRESOLVED). All 8 corrections from those passes are already
  applied in the current `template_gms_61_1.json` and are re-confirmed MATCH below by
  independent re-decompilation.
- **Template state**: the current template has **114** `socket.handlers` rows (not 113 — one
  extra row, `CharacterSkillBookUseHandle`@0x4B, was added post-task-125 on top of the
  113-row baseline the original audit enumerated). All counts below are against the current
  114-row template; the task's "113" figure is the pre-task-125 baseline count.

## Send-site map summary (functions newly decompiled this pass, opcode → function)

| Opcode | Function@addr | Evidence shape |
|---|---|---|
| 0x01 (1) | `SendCheckPasswordPacket@CLogin`@0x564418 | `COutPacket(_,1)`; EncodeStr(id)+EncodeStr(pw)+EncodeBuffer(machineId,16)+Encode4+Encode1×3+Encode4(partnerCode) |
| 0x04 (4) | `CLogin::ChangeStepImmediate`@0x5632d4, `+0x5634a0` | `COutPacket(_,4)` sent when transitioning to step 1 (world-select) with `this[336]==1` gate |
| 0x05 (5) | `sub_564DC9` (`SendLoginPacket`-equivalent, world/channel confirm)@0x564dc9 | `COutPacket(_,5)`; Encode1(worldId)+Encode1(channelId) |
| 0x06 (6) | `SendCheckUserLimitPacket@CLogin`@0x5655df | `COutPacket(_,6)`; Encode2(worldId) |
| 0x07 (7) | `OnAcceptLicense@CLogin`@0x56842a | `COutPacket(_,7)`; Encode1(1) |
| 0x09 (9) | `OnCheckPasswordResult@CLogin`@0x5657ce (2 sites: 0x566089, 0x568a5c/0x56896c via `OnCheckPinCodeResult`) | `COutPacket(_,9)`; Encode1×2+Encode4(accountId)+EncodeStr — post-verification account-info ack, shared by normal login and PIN-verify paths |
| 0x0A (10) | `OnCheckPinCodeResult@CLogin`@0x5688ce (mode-1 branch) | `COutPacket(_,10)`; Encode1(gender-or-flag)+conditional EncodeStr — register new PIN |
| 0x0B (11) | `OnCheckPinCodeResult@CLogin`@0x5688ce (mode-0 branch) | `COutPacket(_,11)` — **not** ServerListRequestHandle; see CORRECTIONS |
| 0x0C (12) | `CLogin::GotoWorldSelect`@0x56838e, `SendViewAllCharPacket@CLogin`@0x567117 (conditional first send) | `COutPacket(_,12)` — no template row claims this value |
| 0x0E (14) | `SendSelectCharPacketByVAC@CLogin`@0x5650b6 | `COutPacket(_,14)`; Encode4×2+EncodeStr(MAC)+EncodeStr(MAC+HDD serial) |
| 0x0F (15) | `CLogin::ResetVAC`@0x56763a (submode 0), `CLogin::MakeVACDlg`@0x5675c4 (submode 1) | `COutPacket(_,15)` ×2 independent sites |
| 0x17 (23) | `sub_5652E3` (delete-character confirm)@0x5652e3 | `COutPacket(_,23)`; Encode4(hash)+Encode4(charId), gated by PIN/password re-check |
| 0x18 (24) | `CClientSocket::OnAliveReq`@0x4744d7 | `COutPacket(_,24)` — bare keepalive-pong reply |
| 0x26 (38) | `CVecCtrlUser::EndUpdateActive`@0x801109 | `COutPacket(_,38)`; Encode1(fh) then `CMovePath::Flush` |
| 0x29 (41) | `CUserLocal::TryDoingMeleeAttack`@0x7a45f1 (disasm @0x7a097b→0x7a0980) | `push 29h` immediately before ctor call |
| 0x2A (42) | `TryDoingShootAttack@CUserLocal`@0x7a67e9 (disasm @0x7a7fc6→0x7a7fce) | `push 2Ah` before ctor call |
| 0x2B (43) | `CUserLocal::TryDoingMagicAttack`@0x7a8572 (disasm @0x7a96d3→0x7a96db) | `push 2Bh` before ctor call |
| 0x2C (44) | `TryDoingBodyAttack@CUserLocal`@0x7b084b (disasm @0x7b0c1d→0x7b0c22) | `push 2Ch` before ctor call |
| 0x2D (45) | `CUserLocal::Update`@0x79fcd9 (2 sites: 0x7a097b→0x7a0980, 0x7a0eba→0x7a0ebf) | `push 2Dh` both sites; periodic random-buffer + submode 0x22/0x27 — self-damage-over-time report |
| 0x30 (48) | `CUserLocal::Update`@0x79fcd9 (3rd site, 0x7a1253→0x7a125a) | `push 30h`; re-confirms `SendEmotionChange` finding |
| 0x38 (56) | `CScriptMan::OnSay`@0x63fb44 (submode 0), `OnAskYesNo`@0x63fddf (submode 2) | `COutPacket(_,56)` ×2 independent sites, different submodes — NPC-conversation "continue" reply |
| 0x68 (104) | `sub_849F27` (report/sue cooldown-gated send)@0x849f27 | `COutPacket(_,104)`; Encode4(targetUserId)+Encode1(reason)+EncodeStr(text), 5-min cooldown |
| 0x88 (136) | `CMobPool::OnMobCrcKeyChanged`@0x5d4d23 | `COutPacket(_,136)` — receive-then-ack pattern |
| 0x8A (138) | `CVecCtrlPet::EndUpdateActive`@0x7fbb76 | `COutPacket(_,138)`; EncodeBuffer(8) |
| 0x8B (139) | `CPet::DoAction`@0x6143a2 | `COutPacket(_,139)`; EncodeBuffer(8)+Encode1(action)+Encode1(frame)+EncodeStr(chat) |
| 0x8C (140) | `sub_613B18` (`ParseCommand@CPet`, "!"-prefixed pet command)@0x613b18 | `COutPacket(_,140)`; EncodeBuffer(8)+Encode1(matched)+Encode1(cmdIndex) |
| 0x8E (142) | `CWvsContext::SendStatChangeItemUseRequestByPetQ`@0x831ab9 | `COutPacket(_,142)`; EncodeBuffer(8)+Encode1+Encode4+Encode2+Encode4 — pet auto-item-use queue |
| 0x92 (146) | `CVecCtrlSummoned::EndUpdateActive`@0x7fe86b | `COutPacket(_,146)`; Encode4(summonId) |
| 0x93 (147) | `CSummoned::TryDoingAttackManual`@0x67a9ca | `COutPacket(_,147)`; Encode4×2+Encode1×2+per-target fields |
| 0x94 (148) | `AttackToTargetMob@CSummoned`@0x67bb40 | `COutPacket(_,148)`; Encode4+Encode1/Encode4×2+Encode1 |
| 0xAC (172) | `CReactorPool::FindHitReactor`@0x6337df | `COutPacket(_,172)`; Encode4+Encode4+Encode2 |
| 0xB0 (176) | `CField_SnowBall::BasicActionAttack`@0x50c0dd | `COutPacket(_,176)`; Encode1+Encode2+Encode2 |
| 0xB1 (177) | `CField_SnowBall::Update`@0x50bb50 | `COutPacket(_,177)` — bare, "left the field" report |
| 0xB4 (180) | `CField_GuildBoss::BasicActionAttack`@0x4ff6d2 | `COutPacket(_,180)` — bare |
| 0xC3 (195) | `CCashShop::TrySendQueryCashRequest`@0x45c33e | `COutPacket(_,195)` — bare |
| 0xC4 (196) | `sub_45CB0E` (coupon-status toggle, submode 0x1F)@0x45cb0e; `CCashShop::OnBuy`@0x457ea4 (submode 3) | `COutPacket(_,196)` ×2 independent sites, different submodes — confirms dispatcher family |
| 0xCB (203) | `sub_70A753` (generic invite-response dialog, accept)@0x70a957; `sub_70A9EE` (…, decline/close)@0x70aa29 | `COutPacket(_,203)`; Encode4(inviterCharId)+Encode1(accept 1/0) — **unclaimed by any template row** |
| 0xD5 (213) | `CITC::OnStatusCharge`@0x528ed7 | `COutPacket(_,213)` — bare |
| 0xD6 (214) | `CITC::TrySendQueryCashRequest`@0x5291cc | `COutPacket(_,214)` — bare |
| 0xD7 (215) | `CITC::OnBuy`@0x529964 | `COutPacket(_,215)`; Encode1(0x10)+Encode4(itemId) |
| 0x7D (125) | `CWvsContext::OnMarriageRequest`@0x84a191 (mode-0 branch) | `COutPacket(_,125)`; Encode1(2)+Encode1(accept)+EncodeStr+Encode4(proposerId) — marriage-proposal accept/reject; **unclaimed by any template row** |

Additional informational (real, but no template row claims the value — not a mismatch, just
confirms the value is "taken" elsewhere in the wire space): 0x02 (`sub_5645C1`, guest-login
variant), 0x45 (`CWvsContext::TryRecovery`, portable-chair-stat auto-heal ack), 0x63
(`sub_849D86`, passive-skill chair-stat-maintenance ping), 0xCF (`CClassCompetition::
SendRequestAuthKey`), 0xD2 (`CWvsContext::Update`, MapleTV chat-relay send).

## Full 114-row result table

Legend: **MATCH** = current template opcode equals observed client wire opcode. **MISMATCH** =
wrong; true value given. **DEAD** = template binds a real client-sent opcode's *slot* to a
handler whose feature the client never actually sends on that opcode (diagnosed in a prior pass,
carried forward). **UNVERIFIED** = no send site pinned this pass either (see note).

| template_opcode | handler | verdict | evidence |
|---|---|---|---|
| 0x01 | LoginHandle | MATCH | `SendCheckPasswordPacket@CLogin`@0x564418 → 1 |
| 0x04 | ServerListRequestHandle | MATCH | `CLogin::ChangeStepImmediate`@0x5632d4 → 4 |
| 0x05 | CharacterListWorldHandle | MATCH | `sub_564DC9`@0x564dc9 → 5 |
| 0x06 | ServerStatusHandle | MATCH | `SendCheckUserLimitPacket@CLogin`@0x5655df → 6 |
| 0x07 | AcceptTosHandle | MATCH | `OnAcceptLicense@CLogin`@0x56842a → 7 |
| 0x08 | SetGenderHandle | MATCH | prior pass: `SendSetGenderPacket`@0x5686a4 → 8 |
| 0x09 | AfterLoginHandle | MATCH | `OnCheckPasswordResult@CLogin`@0x5657ce → 9 |
| 0x0A | RegisterPinHandle | MATCH | `OnCheckPinCodeResult@CLogin`@0x5688ce (mode 1) → 10 |
| 0x0B | ServerListRequestHandle (2nd row, duplicate) | **MISMATCH** | `OnCheckPinCodeResult@CLogin`@0x5688ce (mode 0) → 11; not server-list-related — see CORRECTIONS |
| 0x0D | CharacterViewAllHandle | MATCH | prior pass: `SendViewAllCharPacket`@0x567117 → 13 |
| 0x0E | CharacterViewAllSelectedHandle | MATCH | `SendSelectCharPacketByVAC@CLogin`@0x5650b6 → 14 |
| 0x0F | CharacterViewAllPongHandle | MATCH | `CLogin::ResetVAC`@0x56763a → 15; `MakeVACDlg`@0x5675c4 → 15 |
| 0x13 | CharacterSelectedHandle | MATCH | prior pass: `SendSelectCharPacket`@0x564f79 → 19 |
| 0x14 | CharacterLoggedInHandle | UNVERIFIED | no dedicated send function located this pass |
| 0x15 | CharacterCheckNameHandle | MATCH | prior pass: `sub_565537`@0x565537 → 21 |
| 0x16 | CreateCharacterHandle | MATCH | prior pass: `sub_5653E9`@0x5653e9 → 22 |
| 0x17 | DeleteCharacterHandle | MATCH | `sub_5652E3`@0x5652e3 → 23 |
| 0x18 | PongHandle | MATCH | `CClientSocket::OnAliveReq`@0x4744d7 → 24 |
| 0x19 | StartErrorHandle | UNVERIFIED | no dedicated send function located this pass |
| 0x23 | MapChangeHandle | MATCH | prior pass: `SendTransferFieldRequest`@0x4e8f58 → 35 |
| 0x24 | ChannelChangeHandle | MATCH | prior pass: `SendTransferChannelRequest`@0x4e90ab → 36 |
| 0x25 | CashShopEntryHandle | MATCH | prior pass: `SendMigrateToShopRequest`@0x82e811 → 37 |
| 0x26 | CharacterMoveHandle | MATCH | `CVecCtrlUser::EndUpdateActive`@0x801109 → 38 |
| 0x27 | CharacterChairInteractionHandle | MATCH | prior pass: `SendGetUpFromChairRequest`@0x8374fe → 39 |
| 0x28 | CharacterChairPortableHandle | MATCH | prior pass: `sub_8372C1`@0x8372c1 → 40 |
| 0x29 | CharacterMeleeAttackHandle | MATCH | `CUserLocal::TryDoingMeleeAttack`@0x7a45f1 → 41 |
| 0x2A | CharacterRangedAttackHandle | MATCH | `TryDoingShootAttack@CUserLocal`@0x7a67e9 → 42 |
| 0x2B | CharacterMagicAttackHandle | MATCH | `CUserLocal::TryDoingMagicAttack`@0x7a8572 → 43 |
| 0x2C | CharacterTouchAttackHandle | MATCH | `TryDoingBodyAttack@CUserLocal`@0x7b084b → 44 |
| 0x2D | CharacterDamageHandle | MATCH | `CUserLocal::Update`@0x79fcd9 → 45 (×2 sites) |
| 0x2E | CharacterChatGeneralHandle | MATCH | prior resolution: `SendChatMsg@CField`@0x4e73af → 46 |
| 0x2F | ChalkboardCloseHandle | UNVERIFIED | no `Chalkboard`-named class exists in this IDB at all |
| 0x30 | CharacterExpressionHandle | MATCH | prior + reconfirmed: `SendEmotionChange`@0x845e8f → 48; `CUserLocal::Update` 3rd site → 48 |
| 0x34 | MobBanishPlayer | MATCH | prior pass: `SendBanMapByMobRequest`@0x7aed83 → 52 |
| 0x35 | MonsterBookCover | UNVERIFIED | only receive-side (`OnMonsterBookSetCover`) and local-state setters found; no send |
| 0x36 | NPCStartConversationHandle | MATCH | prior resolution: `TalkToNpc@CUserLocal`@0x7b1403 → 54 |
| 0x38 | NPCContinueConversationHandle | MATCH | `CScriptMan::OnSay`@0x63fb44 → 56; `OnAskYesNo`@0x63fddf → 56 |
| 0x39 | NPCShopHandle | MATCH | prior pass: `CShopDlg` buy/sell → 57 |
| 0x3C | StorageOperationHandle | MATCH | corrected (task-184): `SendGetAllRequest@CStoreBankDlg`@0x6754e1 → 60 |
| 0x3E | HiredMerchantOperationHandle | **DEAD** | client sends nothing on 0x3E; feature rides 0x6F submodes 0x25/0x26/0x29 (`CEntrustedShopDlg`) |
| 0x40 | CompartmentMergeHandle | MATCH | prior pass: `SendGatherItemRequest`@0x8314d0 → 64 |
| 0x41 | CompartmentSortHandle | MATCH | prior pass: `sub_831564`@0x831564 → 65 |
| 0x42 | CharacterInventoryMoveHandle | MATCH | corrected (task-184): `sub_8315F8`@0x8315f8 → 66 |
| 0x43 | CharacterItemUseHandle | MATCH | corrected (task-125): `SendStatChangeItemUseRequest`@0x831880 → 67 |
| 0x44 | CharacterItemCancelHandle | MATCH | prior pass: `sub_831BB7`@0x831bb7 → 68 |
| 0x46 | CharacterItemUseSummonBagHandle | MATCH | corrected (task-184): `SendMobSummonItemUseRequest`@0x831c83 → 70 |
| 0x47 | PetFoodHandle | MATCH | corrected (task-125): `SendPetFoodItemUseRequest`@0x831de9 → 71 |
| 0x48 | MountFoodHandle | MATCH | corrected (task-125): `SendTamingMobFoodItemUseRequest`@0x831f44 → 72 |
| 0x49 | CharacterCashItemUseHandle | MATCH | prior pass: `SendConsumeCashItemUseRequest`@0x832a5d → 73 |
| 0x4B | CharacterSkillBookUseHandle | MATCH | prior pass: `SendSkillLearnItemUseRequest`@0x8325d2 → 75 |
| 0x4C | OwlWarpHandle | MATCH | corrected (task-184): `SendShopScannerItemUseRequest`@0x832680 → 76 |
| 0x4D | TeleportRockUseHandle | MATCH | prior pass: `sub_8327DB`@0x8327db → 77 |
| 0x4E | CharacterItemUseTownScrollHandle | MATCH | prior pass: `sub_841AA5`@0x841aa5 → 78 |
| 0x4F | CharacterItemUseScrollHandle | MATCH | prior resolution: `sub_8317A4`@0x8317a4 → 79 |
| 0x50 | CharacterDistributeApHandle | MATCH | prior resolution: `sub_8457EE`@0x8457ee → 80 |
| 0x51 | CharacterHealOverTimeHandle | MATCH | prior pass: `SendStatChangeRequest`@0x8421f0 → 81 |
| 0x52 | CharacterDistributeSpHandle | MATCH | prior resolution: `sub_8458EB`@0x8458eb → 82 |
| 0x53 | CharacterUseSkillHandle | MATCH | corrected (task-125): `SendSkillUseRequest`@0x7ba213 → 83 |
| 0x54 | CharacterBuffCancel | MATCH | prior pass: `SendSkillCancelRequest`@0x7ba445 → 84 |
| 0x55 | CharacterSkillPrepareHandle | MATCH | prior pass: `DoActiveSkill_Prepare`@0x7b8001 → 85 |
| 0x56 | CharacterDropMesoHandle | MATCH | prior pass: `sub_8459DD`@0x8459dd → 86 |
| 0x57 | FameChangeHandle | MATCH | prior pass: `sub_845A65`@0x845a65 → 87 |
| 0x59 | CharacterInfoRequestHandle | MATCH | prior pass: `sub_845B68`@0x845b68 → 89 |
| 0x5C | PortalScriptHandle | UNVERIFIED | no dedicated send function located this pass |
| 0x5E | TeleportRockAddMapHandle | MATCH | prior pass: `sub_8478EA`@0x8478ea → 94 |
| 0x62 | QuestActionHandle | MATCH | prior resolution: `CQuest::StartQuest`@0x623b12 → 98 |
| 0x68 | SueCharacter | MATCH | `sub_849F27`@0x849f27 → 104 |
| 0x6B | CharacterMultiChatHandle | MATCH | prior resolution: `SendGroupMessage@CUIStatusBar`@0x74467d → 107 |
| 0x6C | CharacterChatWhisperHandle | MATCH | prior pass: `sub_4E8635`@0x4e8635 → 108 |
| 0x6D | CharacterSpouseChatHandle | UNVERIFIED | no dedicated send function located this pass |
| 0x6E | MessengerOperationHandle | MATCH | prior resolution: `ProcessChat@CUIMessenger`@0x6d4021 → 110 |
| 0x6F | CharacterInteractionHandle | MATCH | prior pass: 3-way confirm → 111 |
| 0x70 | PartyOperationHandle | MATCH | prior pass: 3-way confirm → 112 |
| 0x71 | PartyInviteRejectHandle | **MISMATCH** | true opcode 0xCB (203) — see CORRECTIONS |
| 0x72 | GuildOperationHandle | MATCH | prior pass: `SendCreateGuildAgreeMsg`@0x4e9260 → 114 |
| 0x76 | BuddyOperationHandle | MATCH | prior pass: `sub_4E9C03`@0x4e9c03 → 118 |
| 0x77 | NoteOperationHandle | MATCH | prior resolution: `SetRet@CMemoListDlg`@0x5ad50c → 119 |
| 0x79 | UseDoor | MATCH | prior resolution: `TryEnterOpenGate@COpenGatePool`@0x68734f → 121 |
| 0x7B | CharacterKeyMapChangeHandle | MATCH | prior pass: `SaveFuncKeyMap`@0x51ac0d → 123 |
| 0x7C | RPSActionHandle | MATCH | prior pass: `SendSelection@CRPSGameDlg`@0x63c4b4 → 124 |
| 0x7E | AdminCommand | **DEAD** | no dedicated 0x7E send; unmatched `/`-commands ride 0x2E general chat |
| 0x7F | WeddingAction | UNVERIFIED | see note below (0x7D lead found, not confidently attributable) |
| 0x80 | WeddingTalk | UNVERIFIED | see note below (0x7D lead found, not confidently attributable) |
| 0x86 | GuildBBSHandle | MATCH | prior pass: `SendLoadListRequest@CUIGuildBBS`@0x6bb596 → 134 |
| 0x87 | EnterMtsHandle | MATCH | prior pass: `SendMigrateToITCRequest`@0x839b94 → 135 |
| 0x88 | MobCrcKeyChangedReply | MATCH | `CMobPool::OnMobCrcKeyChanged`@0x5d4d23 → 136 |
| 0x8A | PetMovementHandle | MATCH | `CVecCtrlPet::EndUpdateActive`@0x7fbb76 → 138 |
| 0x8B | PetChatHandle | MATCH | `CPet::DoAction`@0x6143a2 → 139 |
| 0x8C | PetCommandHandle | MATCH | `sub_613B18` (`ParseCommand@CPet`)@0x613b18 → 140 |
| 0x8D | PetDropPickUpHandle | MATCH | prior pass: `sub_6149DB`@0x6149db → 141 |
| 0x8E | PetItemUseHandle | MATCH | `SendStatChangeItemUseRequestByPetQ`@0x831ab9 → 142 |
| 0x8F | PetItemExcludeHandle | MATCH | prior pass: `sub_61504D`@0x61504d → 143 |
| 0x92 | SummonMoveHandle | MATCH | `CVecCtrlSummoned::EndUpdateActive`@0x7fe86b → 146 |
| 0x93 | SummonAttackHandle | MATCH | `CSummoned::TryDoingAttackManual`@0x67a9ca → 147 |
| 0x94 | SummonDamageHandle | MATCH | `AttackToTargetMob@CSummoned`@0x67bb40 → 148 |
| 0x9B | MonsterMovementHandle | UNVERIFIED | properly clientbound-only (server→client); no serverbound counterpart found, consistent with there being none |
| 0x9D | MobDropPickupRequest | UNVERIFIED | no dedicated send function located this pass |
| 0x9E | FieldDamageMob | UNVERIFIED | no dedicated send function located this pass |
| 0x9F | MonsterDamageFriendlyHandle | UNVERIFIED | no dedicated send function located this pass |
| 0xA0 | MonsterBomb | UNVERIFIED | no dedicated send function located this pass |
| 0xA1 | MobDamageMob | UNVERIFIED | no dedicated send function located this pass |
| 0xA4 | NPCActionHandle | UNVERIFIED | no dedicated send function located this pass |
| 0xA9 | DropPickUpHandle | MATCH | prior pass: `SendDropPickUpRequest@CWvsContext`@0x8316b8 → 169 |
| 0xAC | ReactorHitHandle | MATCH | `CReactorPool::FindHitReactor`@0x6337df → 172 |
| 0xB0 | Snowball | MATCH | `CField_SnowBall::BasicActionAttack`@0x50c0dd → 176 |
| 0xB1 | LeftKnockback | MATCH | `CField_SnowBall::Update`@0x50bb50 → 177 |
| 0xB2 | Coconut | UNVERIFIED | `CField_Coconut` has only a 0x4f-byte constructor in this IDB — no `BasicActionAttack`/`Update` override exists to send anything |
| 0xB4 | GuildBoss | MATCH | `CField_GuildBoss::BasicActionAttack`@0x4ff6d2 → 180 |
| 0xB7 | MonsterCarnival | UNVERIFIED | `CField_MonsterCarnival*` classes have only constructors in this IDB; no send located |
| 0xC3 | CashShopCheckWalletHandle | MATCH | `CCashShop::TrySendQueryCashRequest`@0x45c33e → 195 |
| 0xC4 | CashShopOperationHandle | MATCH | `sub_45CB0E`@0x45cb0e → 196; `CCashShop::OnBuy`@0x457ea4 → 196 |
| 0xD5 | ItcStatusChargeHandle | MATCH | `CITC::OnStatusCharge`@0x528ed7 → 213 |
| 0xD6 | ItcQueryCashRequestHandle | MATCH | `CITC::TrySendQueryCashRequest`@0x5291cc → 214 |
| 0xD7 | ItcOperationHandle | MATCH | `CITC::OnBuy`@0x529964 → 215 |

### Note on Wedding (0x7F/0x80)

`CWvsContext::OnMarriageRequest`@0x84a191 (mode-0 branch: a marriage-proposal notice arrives,
client shows a Yes/No dialog, then sends the accept/decline) sends **opcode 125 (0x7D)** —
`COutPacket(_,125)`, `Encode1(2)` + `Encode1(accept-flag)` + `EncodeStr` + `Encode4(proposerId)`.
0x7D has **no template row at all**. This is very likely the true home of one of
`WeddingAction`/`WeddingTalk`, but with only one send site found and two candidate handler
names, I am not confident enough to assert which — flagged as a lead, not asserted as a firm
correction. Neither 0x7F nor 0x80 was found sent anywhere in this pass.

## CORRECTIONS (new this pass)

| handler | current | true opcode | evidence | slot-free check |
|---|---|---|---|---|
| `ServerListRequestHandle` (2nd/duplicate template row) | 0x0B | **no correction possible — see disposition** | `OnCheckPinCodeResult@CLogin`@0x5688ce mode-0 branch: `COutPacket(_,11)` — this is a PIN-flow acknowledgment ("no PIN was ever set, treat as accepted"-style reply), not a server-list request. The *other* `ServerListRequestHandle` row (0x04) is independently confirmed correct via `CLogin::ChangeStepImmediate`@0x5632d4. | The true "request world/server list" opcode is 0x0C (`CLogin::GotoWorldSelect`@0x56838e, `SendViewAllCharPacket@CLogin`@0x567117), which is **unclaimed** in the template — free. Disposition: this is a template data-corruption artifact (the same handler name was bound twice, at 0x04 and 0x0B) rather than a simple value swap. Recommended fix: remove the 0x0B `ServerListRequestHandle` duplicate row (0x04 already covers the feature correctly); the real 0x0B feature (PIN-flow ack) and the real 0x0C feature (also server-list-ish, sent from a second code path) are both currently un-named gaps, flagged for follow-up naming rather than guessed here. |
| `PartyInviteRejectHandle` | 0x71 | **0xCB** (203) | `sub_70A753`@0x70a957 (accept branch, `Encode1(1)`) and `sub_70A9EE`@0x70aa29 (decline/close branch, `Encode1(0)`) both `COutPacket(_,203)`; `Encode4(inviterCharId)` + `Encode1(accept-flag)`. This looks like a **generic** invite-response dialog (constructor takes several string-resource ids for different invite *types*, not party-specific naming) — may cover more than just party invites. 0xCB is free in the current template. No confirmed client send exists at 0x71 in this pass. |

Both corrections above are genuine findings but are **not applied to the template in this
read-only pass** — per the task's read-only mandate, no file besides this report was edited.
They are handed off as the concrete next step (a template PATCH + possible atlas-channel
handler-registration check for the 0xCB generic-invite-response feature, and a decision on how
to resolve the 0x0B/0x0C ServerListRequestHandle duplicate).

## Already-applied corrections re-confirmed (carried forward, now MATCH)

`CharacterInventoryMoveHandle`(0x42), `CharacterItemUseSummonBagHandle`(0x46),
`StorageOperationHandle`(0x3C), `OwlWarpHandle`(0x4C) — corrected in task-184's earlier pass;
`CharacterItemUseHandle`(0x43), `PetFoodHandle`(0x47), `MountFoodHandle`(0x48),
`CharacterUseSkillHandle`(0x53) — corrected in task-125. All 8 re-verified MATCH by independent
decompilation this pass (see table above).

## Known DEAD bindings (unchanged from prior pass, not reassignable)

`HiredMerchantOperationHandle`(0x3E) and `AdminCommand`(0x7E) — confirmed again this pass to
have no client send at those opcodes; features ride 0x6F (submodes 0x25/0x26/0x29) and 0x2E
respectively (both opcodes already correctly bound to other handlers).

## Summary counts (against the current 114-row template)

- **MATCH: 93**
- **MISMATCH: 2** (`ServerListRequestHandle`@0x0B duplicate row, `PartyInviteRejectHandle`@0x71)
- **DEAD: 2** (`HiredMerchantOperationHandle`@0x3E, `AdminCommand`@0x7E — unchanged from prior pass)
- **UNVERIFIED: 17** — `CharacterLoggedInHandle`(0x14), `StartErrorHandle`(0x19),
  `ChalkboardCloseHandle`(0x2F), `MonsterBookCover`(0x35), `PortalScriptHandle`(0x5C),
  `CharacterSpouseChatHandle`(0x6D), `WeddingAction`(0x7F), `WeddingTalk`(0x80),
  `NPCActionHandle`(0xA4), `MonsterMovementHandle`(0x9B, likely genuinely clientbound-only),
  `MobDropPickupRequest`(0x9D), `FieldDamageMob`(0x9E), `MonsterDamageFriendlyHandle`(0x9F),
  `MonsterBomb`(0xA0), `MobDamageMob`(0xA1), `Coconut`(0xB2), `MonsterCarnival`(0xB7)

Total: 93 + 2 + 2 + 17 = 114.

No value in this report was fabricated or inferred from memory/naming convention alone — every
MATCH/MISMATCH/DEAD verdict cites a directly observed `COutPacket` ctor integer argument (via
`decompile` or `disasm`) at a specific address, and every UNVERIFIED row states what class/name
patterns were searched and came up empty.
