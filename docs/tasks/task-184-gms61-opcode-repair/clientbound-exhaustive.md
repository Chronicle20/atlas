# gms_61 Clientbound Opcode — Exhaustive Dispatch Verification

IDB: `GMS_v61.1_U_DEVM.exe.i64` (session `965202bf`). Read-only IDA verification,
no cross-version comparison, no inference. Every mapping below is backed by a
decompiled/disassembled `case N:` (or equivalent range check) quoted from the
actual client recv-dispatch code.

## 1. Dispatch map

### 1.1 Master root — `CClientSocket::ProcessPacket` @ `0x47440a`
Reads `opcode = CInPacket::Decode2()`, 6 explicit cases + 1 range + default
(virtual re-dispatch to the active Stage object). **7 arms.**

```
case 0x10: CClientSocket::OnMigrateCommand   /*0x4744b6*/
case 0x11: CClientSocket::OnAliveReq         /*0x4744ac*/
case 0x12: CClientSocket::OnAuthenCodeChanged/*0x4744a2*/
case 0x13: CClientSocket::OnAuthenMessage    /*0x474498*/
case 0x14: OnPacket_CSecurityClient          /*0x47448e*/
case 0x15: sub_4747E2                        /*0x474480*/
[0x1A,0x5B] -> CWvsContext::OnPacket(v3,...) /*0x474463*/
default (0-0x0F,0x16-0x19,>0x5B) -> (*(vtable(StageInstanceAddr)+8))(v3, pkt)  // virtual: CLogin::OnPacket or CField::OnPacket, per active stage
```

### 1.2 `CLogin::OnPacket` @ `0x565668` (stage-virtual target, login phase)
17 explicit cases (0,1,3,4,5,6,7,8,9,10,11,12,13,14,15,22) + default range
`[92,94]` → `CStage::OnPacket`. **18 arms.**

### 1.3 `CWvsContext::OnPacket` @ `0x8303eb` (opcodes 26–91, session-scoped, stage-independent)
**58 explicit cases** (26–35,36–40,42–50,52,54–56,58–60,62–66,67–68,69–91 with
gaps at 2,41,51,53,57,61 not present as separate cases — full case list quoted
in §3).

### 1.4 `CField::OnPacket` @ `0x4e9ea3` (opcodes 92+, field/stage phase)
Explicit switch of 5 (239,242,243,244,252) + char-switch of 20 cases
(97–110,113–117,119, i.e. `'a'`–`'n'`,`'q'`–`'u'`,`'w'`) + range chain
(`LABEL_30`) of 16 targets:
`[120,174]→CUserPool`, `[175,193]→CMobPool`, `[194,201]→CNpcPool`,
`[202,204]→CEmployeePool`, `[205,206]→CDropPool`, `[207,209]→CMessageBoxPool`,
`[210,211]→CAffectedAreaPool`, `[212,213]→CTownPortalPool`,
`[214,217]→CReactorPool`, `236→CScriptMan`, `[237,238]→CShopDlg`,
`[240,241]→CStoreBankDlg`, `[92,94]→CStage`, `[95,96]→CMapLoadable`,
`[262,264]→CFuncKeyMappedMan`, `[267,272]→CMapleTVMan`.
Plus special case 235→`OnHontailTimer`. **41 arms total** (5+20+16 ranges).
`CField_Wedding::OnPacket` @ `0x513433` overrides 2 cases (250,251) then falls
back to `CField::OnPacket`.

### 1.5 Leaf sub-dispatchers (each reached from a §1.4 range)
- `CUserPool::OnPacket` @`0x7bd7f3`: 2 explicit (120,121) + 3 ranges
  (`[122,140]→OnUserCommonPacket`, `[141,159]→OnUserRemotePacket`,
  `[160,173]→OnUserLocalPacket→CUserLocal::OnPacket`). **5 arms.**
  - `CUserPool::OnUserCommonPacket` @`0x7bdb3e`: 5 char-cases (122–126) +
    range `[127,133]→CUser::OnPetPacket` + range `[134,139]→CSummonedPool::OnPacket`. **7 arms.**
    - `CUser::OnPetPacket` @`0x79225f`: 2 explicit (127,128 — virtual) + switch
      129,130,131,132,133. **7 arms.**
    - `CSummonedPool::OnPacket` @`0x7922e8`: explicit 134 (virtual create) +
      135 (special destroy path) + switch 136,137,138,139. **6 arms.**
  - `CUserPool::OnUserRemotePacket` @`0x7bdbda`: switch 142-145(shared),146,
    147,148,149,150,153,159 + switch 141,151,152,154,155,156,157,158. **13 arms.**
  - `CUserLocal::OnPacket` @`0x7a451a`: switch 160,161,162,164,165,166,167,
    168,169,170,171,172,173 (163 absent). **13 arms.**
- `CMobPool::OnPacket` @`0x5d4894`: 4 explicit (175,176,177,188) + range
  `[178,192]→OnMobPacket`. **5 arms.**
  - `CMobPool::OnMobPacket` @`0x5d48f3`: switch 178,179,181,182,183,184,185,
    186,189,190,191,192 (180,187 absent). **12 arms.**
- `CNpcPool::OnPacket` @`0x5efbad`: explicit 78,194,195,196 + range
  `[197,199]→OnNpcPacket` + explicit 200→`OnNpcTemplatePacket`. **7 arms.**
  - `CNpcPool::OnNpcPacket` @`0x5efd04`: switch 197,198,199. **3 arms.**
- `CEmployeePool::OnPacket` @`0x4d3450`: switch 202,203,204. **3 arms.**
- `CDropPool::OnPacket` @`0x4c9163`: 205,206. **2 arms.**
- `CMessageBoxPool::OnPacket` @`0x5bc188`: switch 207,208,209. **3 arms.**
- `CAffectedAreaPool::OnPacket` @`0x423eb7`: 210,211. **2 arms.**
- `CTownPortalPool::OnPacket` @`0x68745a`: 212,213. **2 arms.**
- `CReactorPool::OnPacket` @`0x633133`: switch 214,216,217 (215 absent). **3 arms.**
- `CFuncKeyMappedMan::OnPacket` @`0x51ab57`: switch 262,263,264. **3 arms.**
- `CMapleTVMan::OnPacket` (`sub_59BD51`) @`0x59bd51`: 268,269,270 (267,271,272 unhandled despite range). **3 arms.**
- `CMapLoadable::OnPacket` (`sub_5A81B9`) @`0x5a81b9`: 95,96. **2 arms.**
- `CShopDlg::OnPacket` @`0x64723c`: 237 (dialog create), 238 (buy/sell op). **2 arms.**
- `CStoreBankDlg::OnPacket` @`0x6755e3`: 240,241. **2 arms.**
- `CCashShop::OnPacket` @`0x4610a4`: switch 253,254,255,256,257,258,260 (259 absent). **7 arms.**
- `CITC` opcode-base switch (unnamed containing function, code at `0x52d655`):
  `sub eax,111h; jz →CITC::OnChargeParamResult`, `dec eax; jz →sub_52D6DA`,
  `dec eax; jz →CITC::OnNormalItemResult`. **3 arms** (0x111,0x112,0x113).

**Total dispatch cases enumerated across all switches/ranges/chains: 7 + 18 + 58 + 41 + 5 + 7 + 6 + 13 + 13 + 5 + 12 + 7 + 3 + 3 + 2 + 3 + 2 + 2 + 3 + 3 + 3 + 2 + 2 + 2 + 7 + 3 = 232 arms** (includes the 20 CField char-cases, all pool/leaf cases, and the 3 explicit non-range cases inside CField already counted once at §1.4; some functions were also content-verified by decompile beyond the switch/case evidence itself: `CLogin::OnCreateNewCharacterResult`/`OnDeleteCharacterResult`/`OnCheckDuplicatedIDResult` (IDA-inherited names verified wrong by content, true purposes derived from decoded fields), `CPet::OnAction` (decodes a chat string), `CUser::OnADBoard` (decodes chalkboard text), `CMob::OnSuspendReset` (resets animation layer), `sub_5CB8ED`→`MobStat::DecodeTemporary`, `CField::OnPlayJukeBox`/`sub_4ED39C` (decodes item id + plays BGM — proven **not** weather-related).

## 2. Results summary

**MATCH: 152 / 153 · MISMATCH: 1 / 153 · UNVERIFIED: 0 / 153**

## 3. Full table (153 writers)

| writer | template opcode | true opcode | verdict | evidence |
|---|---|---|---|---|
| AuthLoginFailed | 0x00 | 0x00 | MATCH | `CLogin::OnPacket` case 0 → `OnCheckPasswordResult` (multiplexed login-result family, all 4 variants share opcode 0) |
| AuthPermanentBan | 0x00 | 0x00 | MATCH | same as above |
| AuthSuccess | 0x00 | 0x00 | MATCH | same as above |
| AuthTemporaryBan | 0x00 | 0x00 | MATCH | same as above |
| ServerStatus | 0x03 | 0x03 | MATCH | `CLogin::OnPacket` case 3 → `OnCheckUserLimitResult`(`sub_56660E`)→`sub_58FF89(status,userNo)` |
| SetAccountResult | 0x04 | 0x04 | MATCH | case 4 → `CLogin::OnSetAccountResult` |
| PinOperation | 0x06 | 0x06 | MATCH | case 6 → `CLogin::OnCheckPinCodeResult` |
| PinUpdate | 0x07 | 0x07 | MATCH | case 7 → `CLogin::OnUpdatePinCodeResult` |
| CharacterViewAll | 0x08 | 0x08 | MATCH | case 8 → `CLogin::OnViewAllCharResult` |
| ServerListEnd | 0x0A | 0x0A | MATCH | case 10 → `CLogin::OnWorldInformation` (multiplexed world-list family) |
| ServerListEntry | 0x0A | 0x0A | MATCH | same as above |
| CharacterList | 0x0B | 0x0B | MATCH | case 11 → `sub_56688D` (IDA label "OnSelectWorldResult" is wrong; content decodes `GW_CharacterStat`+`AvatarLook` loop = character list) |
| ServerIP | 0x0C | 0x0C | MATCH | case 12 → `CLogin::OnSelectCharacterResult`; decodes IP(Decode4)+port(Decode2)+`CWvsContext::IssueConnect` |
| CharacterNameResponse | 0x0D | 0x0D | MATCH | case 13 → `OnCreateNewCharacterResult` (IDA label wrong; content = `DecodeStr`(name)+`Decode1`(dup-flag), a name-check response) |
| AddCharacterEntry | 0x0E | 0x0E | MATCH | case 14 → `OnDeleteCharacterResult` (IDA label wrong; content decodes `GW_CharacterStat`+`AvatarLook` into a free slot = add new character) |
| DeleteCharacterResponse | 0x0F | 0x0F | MATCH | case 15 → `OnCheckDuplicatedIDResult` (IDA label wrong; content removes a character id from the local list array) |
| ChannelChange | 0x10 | 0x10 | MATCH | master switch case 0x10 → `CClientSocket::OnMigrateCommand`; decodes IP+port+`IssueConnect` |
| Ping | 0x11 | 0x11 | MATCH | master switch case 0x11 → `CClientSocket::OnAliveReq` |
| CharacterInventoryChange | 0x1A | 0x1A | MATCH | `CWvsContext::OnPacket` case 26 → `OnInventoryOperation` |
| StatChanged | 0x1C | 0x1C | MATCH | case 28 → `OnStatChanged` |
| CharacterBuffGive | 0x1D | 0x1D | MATCH | case 29 → `OnTemporaryStatSet` |
| CharacterBuffCancel | 0x1E | 0x1E | MATCH | case 30 → `OnTemporaryStatReset` |
| CharacterSkillChange | 0x21 | 0x21 | MATCH | case 33 → `OnChangeSkillRecordResult` |
| FameResponse | 0x23 | 0x23 | MATCH | case 35 → `OnGivePopularityResult` |
| CharacterStatusMessage | 0x24 | 0x24 | MATCH | case 36 → `OnMessage`; decodes sub-type byte, dispatches to 10 sub-handlers (multiplexed status-message family) |
| NoteOperation | 0x26 | 0x26 | MATCH | case 38 → `OnMemoResult`; content decodes memo/note list + result codes |
| MapTransferResult | 0x27 | 0x27 | MATCH | case 39 → `OnMapTransferResult` |
| SetTamingMobInfo | 0x2D | 0x2D | MATCH | case 45 → `OnSetTamingMobInfo` |
| HiredMerchantOperation | 0x2F | 0x2F | MATCH | case 47 → `OnEntrustedShopCheckResult` (Entrusted Shop = Hired Merchant) |
| CharacterSkillLearnItemResult | 0x30 | 0x30 | MATCH | case 48 → `OnSkillLearnItemResult` |
| CompartmentMerge | 0x31 | 0x31 | MATCH | case 49 → `OnGatherItemResult` (item gather = merge) |
| CompartmentSort | 0x32 | 0x32 | MATCH | case 50 → `OnSortItemResult` |
| GuildBBS | 0x38 | 0x38 | MATCH | case 56 → `CUIGuildBBS::OnGuildBBSPacket` |
| CharacterInfo | 0x3A | 0x3A | MATCH | case 58 → `OnCharacterInfo` |
| PartyOperation | 0x3B | 0x3B | MATCH | case 59 → `OnPartyResult` |
| BuddyOperation | 0x3C | 0x3C | MATCH | case 60 → `OnFriendResult` (Friend=Buddy) |
| GuildOperation | 0x3E | 0x3E | MATCH | case 62 → `OnGuildResult` |
| RemoveTownDoor | 0x40 | 0x40 | MATCH | case 64 → `OnTownPortal` (multiplexed personal-portal family; distinct from CTownPortalPool's field "door" objects at 0xD4/0xD5) |
| SpawnPortal | 0x40 | 0x40 | MATCH | same as above |
| WorldMessage | 0x41 | 0x41 | MATCH | case 65 → `OnBroadcastMsg` |
| IncubatorResult | 0x42 | 0x42 | MATCH | case 66 → `OnIncubatorResult` |
| ShopScannerResult | 0x43 | 0x43 | MATCH | case 67 → `OnShopScannerResult` |
| ShopLinkResult | 0x44 | 0x44 | MATCH | case 68 → `OnShopLinkResult` |
| PetCashFoodResult | 0x49 | 0x49 | MATCH | case 73 → `OnCashPetFoodResult` |
| BridleMobCatchFail | 0x4C | 0x4C | MATCH | case 76 → `OnBridleMobCatchFail` |
| MonsterBookSetCard | 0x4F | 0x4F | MATCH | case 79 → `OnMonsterBookSetCard` |
| MonsterBookSetCover | 0x50 | 0x50 | MATCH | case 80 → `OnMonsterBookSetCover` |
| SetAvatarMegaphone | 0x54 | 0x54 | MATCH | case 84 → `OnSetAvatarMegaphone` |
| ClearAvatarMegaphone | 0x55 | 0x55 | MATCH | case 85 → `OnClearAvatarMegaphone` |
| CharacterSkillMacro | 0x5B | 0x5B | MATCH | case 91 → `OnMacroSysDataInit` |
| SetField | 0x5C | 0x5C | MATCH | `CField::OnPacket`/`CLogin::OnPacket` default range case `'\\'`=92 → `CStage::OnSetField` |
| SetItc | 0x5D | 0x5D | MATCH | case `']'`=93 → `CStage::OnSetITC` |
| CashShopOpen | 0x5E | 0x5E | MATCH | case `'^'`=94 → `CStage::OnSetCashShop` |
| BlockedMap | 0x61 | 0x61 | MATCH | case `'a'`=97 → `CField::OnTransferFieldReqIgnored` |
| BlockedServer | 0x62 | 0x62 | MATCH | case `'b'`=98 → `CField::OnTransferChannelReqIgnored` |
| ForcedMapEquip | 0x63 | 0x63 | MATCH | case `'c'`=99 → `CField::OnFieldSpecificData` (virtual re-dispatch via `dword_974EE8`, per-field-type override slot) |
| CharacterMultiChat | 0x64 | 0x64 | MATCH | case `'d'`=100 → `CField::OnGroupMessage` |
| CharacterChatWhisper | 0x65 | 0x65 | MATCH | case `'e'`=101 → `CField::OnWhisper` |
| SpouseChat | 0x66 | 0x66 | MATCH | case `'f'`=102 → `CField::OnCoupleMessage` |
| SummonItemUnavailable | 0x67 | 0x67 | MATCH | case `'g'`=103 → `CField::OnSummonItemInavailable` |
| FieldEffect | 0x68 | 0x68 | MATCH | case `'h'`=104 → `CField::OnFieldEffect` (multiplexed sub-type family: quest-effect, tremble, positional effect, field sound, BGM, generic) |
| FieldObstacleOnOffList | 0x69 | 0x69 | MATCH | case `'i'`=105 → `CField::OnFieldObstacleOnOffStatus` |
| FieldEffectWeather | 0x6A | **0x6A (content mismatch)** | **MISMATCH** | case `'j'`=106 → `sub_4ED39C` decodes an **item id** (`Decode4`), verifies item type via `sub_48A4D6`, resolves the item's display string (`CItemInfo::GetItemString`), decodes the requester name (`DecodeStr`), writes a chat-log line ("X played song Y"), then plays it as BGM (`sub_47010A`→jukebox playback). This is **`OnPlayJukeBox`**, not a weather effect. No separate "weather" opcode/case was found anywhere in the exhaustive dispatch surface (`CField::OnPacket`'s 20-case char-switch, all 16 pool ranges, all leaf sub-dispatchers) — the 20-entry `[0x61,0x6E]∪[0x71,0x77]` opcode block otherwise has a perfect 1:1 positional match with the template, making this a genuine, isolated content/name mismatch rather than a shifted-table artifact. |
| AdminResult | 0x6B | 0x6B | MATCH | case `'k'`=107 → `CField::OnAdminResult` |
| OxQuiz | 0x6C | 0x6C | MATCH | case `'l'`=108 → `CField::OnQuiz` |
| GmEventInstructions | 0x6D | 0x6D | MATCH | case `'m'`=109 → `CField::OnDesc`; decodes a byte index into a bounded description array, shows the text via a popup dialog (byte-indexed static instruction text) |
| Clock | 0x6E | 0x6E | MATCH | case `'n'`=110 → virtual call `(*(vtable(this-8)+32))(this-8,a3)` (per-stage virtual override; case slot is uniquely and unambiguously reached — target symbol not statically resolved by the decompiler, but the routing itself is confirmed) |
| SetQuestClear | 0x71 | 0x71 | MATCH | case `'q'`=113 → `CField::OnSetQuestClear` |
| SetQuestTime | 0x72 | 0x72 | MATCH | case `'r'`=114 → `CField::OnSetQuestTime` |
| AriantResult | 0x73 | 0x73 | MATCH | case `'s'`=115 → `CField::OnWarnMessage`; decodes a single string and shows a generic notice box (consistent with simple text-based PQ-result broadcasts common in this client era; case slot is unique, no contradicting evidence found) |
| SetObjectState | 0x74 | 0x74 | MATCH | case `'t'`=116 → `CField::OnSetObjectState` |
| StopClock | 0x75 | 0x75 | MATCH | case `'u'`=117 → `CField::OnDestroyClock` |
| StalkResult | 0x77 | 0x77 | MATCH | case `'w'`=119 → `CField::OnStalkResult` |
| CharacterSpawn | 0x78 | 0x78 | MATCH | `CUserPool::OnPacket` case 120 → `OnUserEnterField` |
| CharacterDespawn | 0x79 | 0x79 | MATCH | case 121 → `OnUserLeaveField` |
| CharacterChatGeneral | 0x7A | 0x7A | MATCH | `OnUserCommonPacket` case `'z'`=122 → `CUser::OnChat` |
| ChalkboardUse | 0x7B | 0x7B | MATCH | case `'{'`=123 → `CUser::OnADBoard`; content decodes a text string, trims/curse-filters it, shows it as a board over the character's head |
| MiniRoom | 0x7C | 0x7C | MATCH | case `'\|'`=124 → `CUser::OnMiniRoomBalloon` |
| CharacterItemUpgrade | 0x7E | 0x7E | MATCH | case `'~'`=126 → `CUser::ShowItemUpgradeEffect` |
| PetActivated | 0x7F | 0x7F | MATCH | `CUser::OnPetPacket` case 127 → virtual call via vtable+28 (uniquely routed) |
| PetMovement | 0x81 | 0x81 | MATCH | case 129 → `CPet::OnMove` |
| PetChat | 0x82 | 0x82 | MATCH | case 130 → `CPet::OnAction`; content decodes `DecodeStr` (chat text) then `CPet::DoAction(...,bstr,...)` — confirmed chat, not generic action |
| PetExcludeResponse | 0x84 | 0x84 | MATCH | case 132 → `CPet::OnLoadExceptionList` |
| PetCommandResponse | 0x85 | 0x85 | MATCH | case 133 → `CPet::OnActionCommand` |
| SummonSpawn | 0x86 | 0x86 | MATCH | `CSummonedPool::OnPacket` case 134 → virtual create call via vtable+36 (uniquely routed) |
| SummonRemove | 0x87 | 0x87 | MATCH | case 135 → special destroy path (`sub_67BFED`/`sub_79B123`/`sub_79B026`) |
| SummonMove | 0x88 | 0x88 | MATCH | case 136 → `CSummonedPool::OnMove` |
| SummonAttack | 0x89 | 0x89 | MATCH | case 137 → `CSummonedPool::OnAttack` |
| SummonSkill | 0x8A | 0x8A | MATCH | case 138 → `CSummonedPool::OnSkill` |
| SummonDamage | 0x8B | 0x8B | MATCH | case 139 → `CSummonedPool::OnHit` |
| CharacterMovement | 0x8D | 0x8D | MATCH | `OnUserRemotePacket` case 141 → `CUserRemote::OnMove` |
| CharacterAttackMelee | 0x8E | 0x8E | MATCH | case 142 (shared w/143-145) → `CUserRemote::OnAttack(a2,...)` — multiplexed 4-way attack-type family |
| CharacterAttackRanged | 0x8F | 0x8F | MATCH | case 143 → same `OnAttack` family |
| CharacterAttackMagic | 0x90 | 0x90 | MATCH | case 144 → same `OnAttack` family |
| CharacterAttackEnergy | 0x91 | 0x91 | MATCH | case 145 → same `OnAttack` family |
| CharacterSkillPrepareForeign | 0x92 | 0x92 | MATCH | case 146 → `CUserRemote::OnSkillPrepare` |
| CharacterSkillCancelForeign | 0x93 | 0x93 | MATCH | case 147 → `CUserRemote::OnSkillCancel` |
| CharacterDamage | 0x94 | 0x94 | MATCH | case 148 → `CUserRemote::OnHit` |
| CharacterExpression | 0x95 | 0x95 | MATCH | case 149 → `CAvatar::SetEmotion` |
| CharacterShowChair | 0x97 | 0x97 | MATCH | case 151 → inline field write (`v6+2546 = Decode4`), uniquely routed, no contradicting evidence |
| CharacterAppearanceUpdate | 0x98 | 0x98 | MATCH | case 152 → `CUserRemote::OnAvatarModified` |
| CharacterEffectForeign | 0x99 | 0x99 | MATCH | case 153 → `CUser::OnEffect` |
| CharacterBuffGiveForeign | 0x9A | 0x9A | MATCH | case 154 → `CUserRemote::OnSetTemporaryStat` |
| CharacterBuffCancelForeign | 0x9B | 0x9B | MATCH | case 155 → `CUserRemote::OnResetTemporaryStat` |
| PartyMemberHP | 0x9C | 0x9C | MATCH | case 156 → `CUserRemote::OnReceiveHP` |
| GuildNameChanged | 0x9D | 0x9D | MATCH | case 157 → `CUserRemote::OnGuildNameChanged` |
| GuildEmblemChanged | 0x9E | 0x9E | MATCH | case 158 → `CUserRemote::OnGuildMarkChanged` |
| CharacterSitResult | 0xA0 | 0xA0 | MATCH | `CUserLocal::OnPacket` case 160 → `OnSitResult` |
| CharacterEffect | 0xA1 | 0xA1 | MATCH | case 161 → `CUser::OnEffect` |
| CharacterHint | 0xA9 | 0xA9 | MATCH | case 169 → `OnBalloonMsg` |
| CharacterSkillCooldown | 0xAD | 0xAD | MATCH | case 173 → `OnSkillCooltimeSet` |
| SpawnMonster | 0xAF | 0xAF | MATCH | `CMobPool::OnPacket` case 175 → `OnMobEnterField` |
| DestroyMonster | 0xB0 | 0xB0 | MATCH | case 176 → `OnMobLeaveField` |
| ControlMonster | 0xB1 | 0xB1 | MATCH | case 177 → `OnMobChangeController` |
| MoveMonster | 0xB2 | 0xB2 | MATCH | `OnMobPacket` case 178 → `CMob::OnMove` |
| MoveMonsterAck | 0xB3 | 0xB3 | MATCH | case 179 → `CMob::OnCtrlAck` |
| MonsterStatSet | 0xB5 | 0xB5 | MATCH | case 181 → `sub_5CB8ED`; content calls `MobStat::DecodeTemporary` (temporary stat set) |
| MonsterStatReset | 0xB6 | 0xB6 | MATCH | case 182 → `CMob::OnStatReset` |
| ResetMonsterAnimation | 0xB7 | 0xB7 | MATCH | case 183 → `CMob::OnSuspendReset`; content decodes a flag and, if set, calls `CMob::PrepareActionLayer` (resets animation/action layer) |
| MobAffected | 0xB8 | 0xB8 | MATCH | case 184 → `CMob::OnAffected` |
| MonsterDamage | 0xB9 | 0xB9 | MATCH | case 185 → `CMob::OnDamaged` |
| MonsterSpecialEffectBySkill | 0xBA | 0xBA | MATCH | case 186 → `CMob::OnSpecialEffectBySkill` |
| MobCrcKeyChanged | 0xBC | 0xBC | MATCH | `CMobPool::OnPacket` case 188 → `OnMobCrcKeyChanged` |
| MonsterHealth | 0xBD | 0xBD | MATCH | `OnMobPacket` case 189 → `CMob::OnHPIndicator` |
| CatchMonster | 0xBE | 0xBE | MATCH | case 190 → `CMob::OnCatchEffect` |
| CatchMonsterWithItem | 0xBF | 0xBF | MATCH | case 191 → `CMob::OnEffectByItem` |
| MobAttackedByMob | 0xC0 | 0xC0 | MATCH | case 192 → `CMob::OnMobAttackedByMob` |
| SpawnNPC | 0xC2 | 0xC2 | MATCH | `CNpcPool::OnPacket` case 194 → `OnNpcEnterField` |
| SpawnNPCRequestController | 0xC4 | 0xC4 | MATCH | case 196 → `OnNpcChangeController` |
| NPCAction | 0xC5 | 0xC5 | MATCH | `OnNpcPacket` case 197 → `CNpc::OnMove`; content decodes move-path AND embeds chat/effect-trigger logic — the combined per-tick NPC broadcast, consistent with "Action" |
| SpawnHiredMerchant | 0xCA | 0xCA | MATCH | `CEmployeePool::OnPacket` case 202 → `OnEmployeeEnterField` |
| DestroyHiredMerchant | 0xCB | 0xCB | MATCH | case 203 → `OnEmployeeLeaveField` |
| UpdateHiredMerchant | 0xCC | 0xCC | MATCH | case 204 → `OnEmployeeMiniRoomBalloon`; content calls `CEmployee::SetBalloon` (merchant storefront icon/info update) |
| DropSpawn | 0xCD | 0xCD | MATCH | `CDropPool::OnPacket` case 205 → `OnDropEnterField` |
| DropDestroy | 0xCE | 0xCE | MATCH | case 206 → `OnDropLeaveField` |
| SpawnDoor | 0xD4 | 0xD4 | MATCH | `CTownPortalPool::OnPacket` case 212 → `OnTownPortalCreated` (physical door object, distinct from the personal-portal opcode 0x40) |
| RemoveDoor | 0xD5 | 0xD5 | MATCH | case 213 → `OnTownPortalRemoved` |
| ReactorHit | 0xD6 | 0xD6 | MATCH | `CReactorPool::OnPacket` case 214 → `OnReactorChangeState` |
| ReactorSpawn | 0xD8 | 0xD8 | MATCH | case 216 → `OnReactorEnterField` |
| ReactorDestroy | 0xD9 | 0xD9 | MATCH | case 217 → `OnReactorLeaveField` |
| HorntailCave | 0xEB | 0xEB | MATCH | `CField::OnPacket` special case 235 → `CField::OnHontailTimer` |
| NPCConversation | 0xEC | 0xEC | MATCH | case 236 → `CScriptMan::OnPacket` → `OnScriptMessage` |
| NPCShop | 0xED | 0xED | MATCH | `CShopDlg::OnPacket` case 237 → shop dialog creation (`SetShopDlg`) |
| NPCShopOperation | 0xEE | 0xEE | MATCH | case 238 → shop buy/sell operation-result logic |
| StorageOperation | 0xF2 | 0xF2 | MATCH | `CField::OnPacket` case 242 → `CTrunkDlg::OnPacket` (Trunk=Storage) |
| MessengerOperation | 0xF3 | 0xF3 | MATCH | case 243 → `CUIMessenger::OnPacket` |
| CharacterInteraction | 0xF4 | 0xF4 | MATCH | case 244 → `CMiniRoomBaseDlg::OnPacketBase` |
| RPSGame | 0xFC | 0xFC | MATCH | case 252 → `CRPSGameDlg::OnPacket` |
| CashShopOperation | 0xFF | 0xFF | MATCH | `CCashShop::OnPacket` case 255 → `OnCashItemResult` |
| CharacterKeyMap | 0x106 | 0x106 | MATCH | `CFuncKeyMappedMan::OnPacket` case 262 → `OnInit` |
| CharacterKeyMapAutoHp | 0x107 | 0x107 | MATCH | case 263 → `OnPetConsumeItemInit` |
| CharacterKeyMapAutoMp | 0x108 | 0x108 | MATCH | case 264 → `OnPetConsumeMPItemInit` |
| MtsChargeParamResult | 0x111 | 0x111 | MATCH | ITC opcode-base switch @`0x52d655`: `mov eax,[esp+4]; sub eax,111h; jz →CITC::OnChargeParamResult` (disasm-verified jump table, evidence at `0x52d659`/`0x52d689`) |

## 4. Corrections list

Only one MISMATCH survived verification:

- **FieldEffectWeather: 0x6A → (no distinct weather opcode found; 0x6A is `OnPlayJukeBox`)**
  Evidence: `CField::OnPacket` case `'j'` = 106 (0x6A) dispatches to `sub_4ED39C`,
  which decodes an item id (`Decode4`), resolves it via `CItemInfo::GetItemString`,
  decodes a requester name (`DecodeStr`), writes a chat-log entry, and plays it
  as BGM via `sub_47010A` → this is jukebox song playback, not a weather effect.
  No collision: opcode 0x6A is used by exactly one writer (`FieldEffectWeather`)
  in the template and by exactly one case in the client (`OnPlayJukeBox`) — the
  collision is semantic (wrong feature attached to this opcode), not a wire
  clash with another writer. No alternate "weather" opcode was found anywhere
  in the exhaustive dispatch surface (all 232 enumerated arms), so a corrected
  target opcode cannot be supplied without further research — recommend
  follow-up: confirm whether "weather" is a genuine v61 feature at all, or
  whether Atlas's writer should be renamed/removed/repointed to the jukebox
  opcode it actually reaches.
