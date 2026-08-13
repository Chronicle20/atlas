# v48 writers resolved from the IDB (clientbound)

v83 template writers absent from the v48 template, resolved to a concrete v48 opcode
by matching the fully-qualified client function through the v48 dispatch tree
(see `v48-clientbound-map.md`). `[options]` marks entries whose v83 counterpart carries
an `options` mode table — those tables are version-specific and are NOT yet derived.

| v48 opCode | Writer | Client function | options needed |
|---|---|---|---|
| `0x2A` | `HiredMerchantOperation` | `CWvsContext::OnEntrustedShopCheckResult` | yes |
| `0x38` | `IncubatorResult` | `CWvsContext::OnIncubatorResult` | — |
| `0x39` | `ShopScannerResult` | `CWvsContext::OnShopScannerResult` | yes |
| `0x3F` | `PetCashFoodResult` | `CWvsContext::OnCashPetFoodResult` | — |
| `0x41` | `AvatarMegaphoneResult` | `CWvsContext::OnAvatarMegaphoneRes` | yes |
| `0x49` | `SetField` | `CStage::OnSetField` | — |
| `0x4A` | `SetItc` | `CStage::OnSetITC` | — |
| `0x56` | `PlayJukebox` | `CField::OnPlayJukeBox` | — |
| `0x67` | `CharacterChatGeneral` | `CUser::OnChat` | — |
| `0x68` | `ChalkboardUse` | `CUser::OnADBoard` | — |
| `0x69` | `MiniRoom` | `CUser::OnMiniRoomBalloon` | yes |
| `0x6B` | `CharacterItemUpgrade` | `CUser::ShowItemUpgradeEffect` | — |
| `0x77` | `SummonMove` | `CSummonedPool::OnMove` | — |
| `0x78` | `SummonAttack` | `CSummonedPool::OnAttack` | — |
| `0x7A` | `SummonDamage` | `CSummonedPool::OnSkill` | — |
| `0x7E` | `CharacterMovement` | `CUserRemote::OnMove` | yes |
| `0x7F` | `CharacterAttackMelee` | `CUserRemote::OnAttack` | — |
| `0x7F` | `CharacterAttackEnergy` | `CUserRemote::OnAttack` | — |
| `0x80` | `CharacterAttackRanged` | `CUserRemote::OnAttack case 128 -> sub_6BD4F8` | — |
| `0x81` | `CharacterAttackMagic` | `CUserRemote::OnAttack case 129 -> sub_6BE1D0` | — |
| `0x82` | `CharacterSkillPrepareForeign` | `CUserRemote::OnSkillPrepare` | — |
| `0x84` | `CharacterDamage` | `CUserRemote::OnHit` | — |
| `0x85` | `CharacterExpression` | `CAvatar::SetEmotion` | — |
| `0x88` | `CharacterAppearanceUpdate` | `CUserRemote::OnAvatarModified` | — |
| `0x89` | `CharacterEffectForeign` | `CUser::OnEffect (via CUserPool::OnUserRemotePacket case 137)` | yes |
| `0x8A` | `CharacterBuffGiveForeign` | `CUserRemote::OnSetTemporaryStat` | — |
| `0x8B` | `CharacterBuffCancelForeign` | `CUserRemote::OnResetTemporaryStat` | — |
| `0x8C` | `PartyMemberHP` | `CUserRemote::OnReceiveHP` | — |
| `0x8D` | `GuildNameChanged` | `CUserRemote::OnGuildNameChanged` | — |
| `0x8E` | `GuildEmblemChanged` | `CUserRemote::OnGuildMarkChanged` | — |
| `0x92` | `CharacterEffect` | `CUser::OnEffect (via CUserLocal::OnPacket case 146)` | yes |
| `0x9E` | `SpawnMonster` | `CMobPool::OnMobEnterField` | — |
| `0x9F` | `DestroyMonster` | `CMobPool::OnMobLeaveField` | — |
| `0xA0` | `ControlMonster` | `CMobPool::OnMobChangeController` | — |
| `0xA2` | `MoveMonster` | `CMob::OnMove` | yes |
| `0xA3` | `MoveMonsterAck` | `CMob::OnCtrlAck` | — |
| `0xA5` | `MonsterStatSet` | `CMob::OnStatSet` | — |
| `0xA6` | `MonsterStatReset` | `CMob::OnStatReset` | — |
| `0xA7` | `ResetMonsterAnimation` | `CMob::OnSuspendReset` | — |
| `0xA8` | `MobAffected` | `CMob::OnAffected` | — |
| `0xA9` | `MonsterDamage` | `CMob::OnDamaged` | — |
| `0xAA` | `MonsterSpecialEffectBySkill` | `CMob::OnSpecialEffectBySkill` | — |
| `0xB1` | `SpawnNPC` | `CNpcPool::OnNpcEnterField` | — |
| `0xB3` | `SpawnNPCRequestController` | `CNpcPool::OnNpcChangeController` | yes |
| `0xC1` | `DropSpawn` | `CDropPool::OnDropEnterField` | — |
| `0xC2` | `DropDestroy` | `CDropPool::OnDropLeaveField` | — |

**Total resolved: 46** of 147 writers the v48 template lacks.

## Ambiguities resolved by dispatch site (not by name)

`CUser::OnEffect` is reached from two different dispatchers, so the function name alone
cannot separate the foreign and local variants:

- opcode **137 / `0x89`** — `CUserPool::OnUserRemotePacket` @ `0x6b2ae4` case 137, i.e. a
  *remote* user's effect → `CharacterEffectForeign`.
- opcode **146 / `0x92`** — `CUserLocal::OnPacket` @ `0x6a048d` case 146, i.e. the *local*
  user's effect → `CharacterEffect`.

This matches v83's relative ordering (`CharacterEffectForeign` `0xC6` < `CharacterEffect` `0xCE`).

`CUserRemote::OnAttack` @ `0x6bca49` serves three opcodes with three distinct bodies:

| v48 opcode | Tail call | Distinguishing read | Writer |
|---|---|---|---|
| 127 / `0x7F` | `sub_6BD074` | — | `CharacterAttackMelee` |
| 128 / `0x80` | `sub_6BD4F8` | passes `v36` (`Decode4` before the damage loop) | `CharacterAttackRanged` |
| 129 / `0x81` | `sub_6BE1D0` | passes `v40`, read only for skills `2121001`/`2221001`/`2321001` | `CharacterAttackMagic` |

## Negative finding — `CharacterAttackEnergy` is ABSENT in v48

v83 has four attack writers (`0xBA` melee, `0xBB` ranged, `0xBC` magic, `0xBD` energy).
v48's `OnAttack` switch has only cases 127/128/129 — there is no fourth arm, and 130 is
already `CUserRemote::OnSkillPrepare`. Energy Charge is a Pirate skill and the Pirate job
postdates this build. **Do not add `CharacterAttackEnergy` to the v48 template.**

## Remaining work

100 of the 147 missing writers are not yet resolved. They fall into two groups that must be
separated by decompiling the remaining leaf dispatchers — an unresolved name is NOT evidence
of absence:

- **Leaf dispatchers not yet read:** `CUser::OnPetPacket` (108–115), `CSummonedPool` detail,
  `CUserLocal` (144–155, several `sub_*`), `CNpcPool::OnNpcPacket` (180–182), `sub_56D510`
  (183–185), `CMobPool::OnMobPacket` remainder, `CEmployeePool` (187–191),
  `CMessageBoxPool` (196–200), `CAffectedAreaPool` (201–204), `CTownPortalPool` (205–208),
  `CReactorPool` (209–214), `CScriptMan` (225–227), `CStoreBankDlg` (228–231),
  `z_MISLABELED_notRPS_channelFindDlg` (233–236), `CRPSGameDlg` (237), `CUIMessenger` (238),
  `CMiniRoomBaseDlg` (239), `CTrunkDlg` (247), `CFuncKeyMappedMan` (263–266),
  `sub_527238` (267–272), and `CLogin::OnPacket` @ `0x5007c4` for the login-phase writers.
- **Genuinely post-v48 features** (expected to stay absent): PIC/second-password, Nexon
  passport, monster book, guild BBS, owl/entrusted-shop scanning, Ariant/pyramid events.

`options` mode tables are derived for none of the resolved entries yet. Those values are
client-specific per version and must each come from the corresponding v48 mode switch —
v83's numbers must never be copied across (task-139 established this for `usPetSkill`).

---

## Mode tables — what verified and what did not (task-188)

`ShopScannerResult` `0x39` and `SpawnNPCRequestController` `0xB3` are wired with tables
derived from the v48 binary:

- `CNpcPool::OnNpcChangeController` @ `0x56d617` reads `Decode1` then `Decode4(npcId)`
  and branches `SetLocalNpc` / `SetRemoteNpc` on that flag → `{GRANT: 1, REVOKE: 0}`,
  matching v61/v72/v79/v83.
- `CWvsContext::OnShopScannerResult` @ `0x71ff8e` switches on `Decode1() - 6`: arm 0
  (mode 6) is the commodity/search-result branch, arm 1 (mode 7) is the id-list branch
  → `{RESULT: 6, HOT_LIST: 7}`, identical to v61/v72/v79/v83.

Three remain unwired, deliberately:

**`HiredMerchantOperation` `0x2A`** — `CWvsContext::OnEntrustedShopCheckResult` @ `0x71f72a`
switches on `Decode1() - 6` with arms at modes 6, 7, 8, 9, 10, 12, 13, 14, 15. v83 uses
7, 8, 9, 10, 11, 13, 14, 15, 16, 17, 18. The sets are NOT a uniform shift: v48 mode 6 is
`SendOpenShopRequest` (v83's `OPEN_SHOP` is 7), but v48 mode 7 builds a channel-name
message, which resembles v83's `REMOTE_SHOP_WARP` (16) rather than its `ERROR_UNKNOWN` (8).
Mapping names onto these arms needs the per-arm string resources resolved; a positional
shift would be a guess of exactly the kind that produced the `BLOW_WEATHER` error this
task corrected.

**`StorageOperation` `0xF7`** — `CTrunkDlg::OnPacket` @ `0x58332c` not yet decompiled.
Note v61/v72/v79/v83 all carry an identical table, so it is plausibly stable, but v48 has
repeatedly proved to be the outlier version and has not been checked.

**`AvatarMegaphoneResult` `0x41`** — `CWvsContext::OnAvatarMegaphoneRes` @ `0x7211cd`
switches on `Decode1() - 48`, special-casing exactly two codes: 48 (notice string 3621)
and 49 (notice string 3445); anything else carries a `DecodeStr` message. v83 names its
two `{WAITING_LINE: 83, LEVEL_GATE: 84}`. The v48 CODES are certain (48/49); which name
belongs to which is not, the string-pool ids are unresolved, and no legacy template
carries this writer to corroborate against. Left out rather than assigned positionally.

## Tooling note

`mcp__ida-pro__decompile` printed `CWvsContext::OnIncubatorResult` as the header for the
function at `0x71f72a`. That is a stale display name — `lookup_funcs` on the same address
returns `?OnEntrustedShopCheckResult@CWvsContext@@` (size `0x2df`), and `OnIncubatorResult`
is a distinct function at `0x71fa31` (size `0x2d5`). Trust `lookup_funcs`/`func_query` over
the decompiler's printed header; taking the header at face value here would have reverted
the correct registry fix in 5270b70fe.

---

## Final classification of the writers still absent from the v48 template

The v48 template ended this task at **113 writers** (from 62). The rest are not
oversights — they were checked against the binary and fall into three groups.

### 1. Verified ABSENT from the v48 client

Batched `func_query` over the v48 IDB returned NO match for any of these methods.
The query technique is sound: run in the same batches, `OnMove@CPet`,
`OnNameChanged@CPet`, `OnSkill@CSummonedPool`, `OnAttack@CSummonedPool`,
`OnEffect@CUser` and `OnPetPacket@CUser` all resolved, so a zero result is a real
absence, not a lookup failure.

`OnActionCommand` · `OnBalloonMsg` · `OnCatchEffect` · `OnEffectByItem` ·
`OnHPIndicator` · `OnIncMobChargeCount` · `OnMobAttackedByMob` · `OnMobSpeaking` ·
`OnMobCrcKeyChanged` · `OnLoadExceptionList` · `OnSitResult` · `OnSkillCooltimeSet` ·
`OnTutorMsg` · `OnOpenUI` · `OnSetStandAloneMode` · `OnSetDirectionMode` ·
`OnSetActivePortableChair` · `OnSkillCancel` · `OnCreated` · `OnRemoved` ·
`OnBridleMobCatchFail` · `OnCharacterSale` · `OnChargeParamResult` ·
`OnCheckSPWResult` · `OnContiMove` · `OnContiState` · `OnFieldObstacle*` ·
`OnFriendResult` · `OnGatherItemResult` · `OnHontailTimer` · `OnItemUpgrade` ·
`OnLatestConnectedWorld` · `OnMacroSysDataInit` · `OnMonsterBookSetCard/Cover` ·
`OnNormalItemResult` · `OnQueryCashResult` · `OnRecommendWorldMessage` ·
`OnScriptProgressMessage` · `OnShopLinkResult` · `OnSortItemResult` ·
`OnStalkResult` · `OnZakumTimer` · `OnChaosZakumTimer` · `OnSetMessage` ·
`OnClearMessage` · `OnSendMessageResult` · `OnVegaResult` · `OnGuildBBSPacket`

Whole event-field classes are absent too (`CField_MonsterCarnival`,
`CField_Coconut`, `CField_AriantArena`, `CField_Battlefield`, `CField_Tournament`,
`CField_Witchtower`, `CField_Massacre`, `CField_ContiMove`), which accounts for the
Monster Carnival, Coconut, Ariant, Sheep Ranch, Tournament, Witch Tower and Pyramid
writers in one stroke. So: PIC, monster book, guild BBS, buddy list, MTS,
compartment merge/sort, and the event minigames are all post-v48 and correctly
have no template entry.

`CharacterAttackEnergy` is absent for the same reason — `CUserRemote::OnAttack`
@ `0x6bca49` has only cases 127/128/129 and 130 is already `OnSkillPrepare`.

### 2. Opcode known, mode table NOT derivable by analogy — deliberately unwired

| Writer | v48 opcode | v48 dispatcher | why it is not wired |
|---|---|---|---|
| `HiredMerchantOperation` | `0x2A` | `OnEntrustedShopCheckResult` @ `0x71f72a` | arms at modes 6,7,8,9,10,12,13,14,15 vs v83's 7,8,9,10,11,13,14,15,16,17,18 — not a uniform shift (v48 mode 6 is `SendOpenShopRequest`; v48 mode 7 builds a channel-name message resembling v83's `REMOTE_SHOP_WARP` 16) |
| `StorageOperation` | `0xF7` | `CTrunkDlg::OnPacket` @ `0x58332c` | arms at modes 8,18,23,24,25,26,27 vs v83's 9..23 — again a genuinely different set |
| `AvatarMegaphoneResult` | `0x41` | `OnAvatarMegaphoneRes` @ `0x7211cd` | switches on `Decode1()-48`, special-casing exactly codes 48 and 49; v83 names its two `WAITING_LINE`/`LEVEL_GATE` at 83/84. Codes certain, name assignment is not, and no legacy template carries this writer |
| `CharacterEffect` / `CharacterEffectForeign` | `0x92` / `0x89` | `CUser::OnEffect` via local/remote dispatch | opcodes verified (see the ambiguity section above), but both carry an `operations` table that has not been derived for v48 |

Each of these has a correct, known opcode. They are omitted because an
options-less dispatcher-family writer resolves its mode to the loud `99` default
at runtime, which is worse than being absent — and because assigning names by
positional analogy is precisely what produced the `BLOW_WEATHER` defect this task
corrected.

### 3. Not yet walked

`CStoreBankDlg` (228-231), `CCashShop` (254-261), `z_MISLABELED_notRPS_channelFindDlg`
(233-236) and the `CUserLocal` `sub_6A*` leaves (145-154). No writer in group 1 depends
on these; they would only add entries beyond the verified-absent set.

---

## Serverbound handler harvest (task-188)

Serverbound has no dispatch switch to walk — each handler is a separate client-side
send-site — so the method is: take the handler's v83 sender FName, look it up in the v48
IDB, and read the `COutPacket(N)` constructor argument. `COutPacket::COutPacket` is
`??0COutPacket@@QAE@J@Z` @ `0x57b77e`; the opcode is its second argument.

The committed registry's serverbound set was already exhausted against this template —
80 of its 81 entries sit on opcodes the template already used — so these are new.

| v48 opCode | Handler | Sender | evidence |
|---|---|---|---|
| `0x32` | `HiredMerchantOperationHandle` | `CWvsContext::SendEntrustedShopCheckRequest` @ `0x71f56a` | `COutPacket(50)`, then `Encode1(0)` + `EncodeBuffer(8)` |
| `0x3B` | `CharacterItemUseSummonBagHandle` | `CWvsContext::SendMobSummonItemUseRequest` @ `0x70ddaa` | `COutPacket(59)`, `Encode4`/`Encode2(slot)`/`Encode4(itemId)` |
| `0x3C` | `PetFoodHandle` | `CWvsContext::SendPetFoodItemUseRequest` @ `0x70df2f` | `COutPacket(60)`, `Encode4`/`Encode2(slot)`/`Encode4(itemId)` |
| `0x42` | `CharacterItemUseScrollHandle` | `CWvsContext::SendUpgradeItemUseRequest` @ `0x70da60` | `COutPacket(66)`, `Encode4`/`Encode2`×3/`Encode1` |
| `0x4D` | `PetSpawnHandle` | `CWvsContext::SendActivatePetRequest` @ `0x71d118` | `COutPacket(77)`, `Encode4`/`Encode2(slot)` |

`PetFoodHandle` landing at `0x3C`, directly before the existing `MountFoodHandle`
`0x3D`, matches the pet/mount adjacency v61 shows (`0x47`/`0x48`) — an independent
consistency check on the harvest.

### Not added

`CharacterItemUseTownScrollHandle` — `CWvsContext::SendPortalScrollUseRequest` @
`0x719dd9` emits `COutPacket(65)` = `0x41`, but `0x41` is already
`CharacterItemUseHandle` in this template. v48 therefore routes the town-scroll through
the generic item-use opcode rather than a dedicated one; binding a second handler to
`0x41` would double-map it at listener-build time. Left alone deliberately.

### Remaining

The other ~46 handlers' senders were not found by name in the v48 IDB in the batches run
so far (`DoActiveSkill_*`, `TryDoing*`, the MTS/auction `On*` family, the cash-shop
`On*` family). Several are plainly post-v48 features; the rest need either the unnamed
`sub_*` senders identified or a full xref sweep of `COutPacket::COutPacket` @ `0x57b77e`,
which is the exhaustive form of this harvest and has not been run.

## Exhaustive serverbound sweep — `COutPacket` xrefs

`xrefs_to ??0COutPacket@@QAE@J@Z` @ `0x57b77e` returns **317 call sites, untruncated**
(`more: false`). That is the complete set of packet-construction sites in the v48
client, so any serverbound opcode the client can emit is reachable from it.

Note `packet-audit discover-ops` does NOT cover this: its worklist is explicitly
"clientbound only — serverbound FName verification is deferred". It walks dispatcher
switches, not send-sites. (It also needed `-ida-database` added to reach the session
server; done in this task alongside the other subcommands.)

Handlers added from the sweep:

| v48 opCode | Handler | Sender | evidence |
|---|---|---|---|
| `0x46` | `CharacterUseSkillHandle` | `CUserLocal::SendSkillUseRequest` @ `0x6afa91` | `COutPacket(70)`, `Encode4`/`Encode4(skillId)`/`Encode1` + skill-specific tails |
| `0x87` | `MonsterBomb` | `CMob::TryFirstSelfDestruction` @ `0x551f38` | `COutPacket(135)`, `Encode4(mobId)` |

### Negative results — handlers that are NOT separate opcodes in v48

These matter as much as the additions: each looked like a gap and is not one.

- **`EnterDoorHandle`** — `CField::TryEnterTownPortal` @ `0x4cb90a` emits
  `COutPacket(103)` = `0x67`, which this template already binds to `UseDoor`. Same
  packet, different Atlas name; adding it would double-bind `0x67`.
- **`CharacterItemUseTownScrollHandle`** — `CWvsContext::SendPortalScrollUseRequest`
  @ `0x719dd9` emits `COutPacket(65)` = `0x41`, already `CharacterItemUseHandle`. v48
  routes the town scroll through the generic item-use opcode.
- **`CharacterAutoDistributeApHandle`** — `CWvsContext::SendAbilityUpRequest`
  @ `0x71cd00` emits `COutPacket(67)` = `0x43`, already `CharacterDistributeApHandle`.
  v48 has no separate auto-distribute packet.
- **`CashShopCheckWalletHandle`** — `CCashShop::TrySendQueryCashRequest` @ `0x44f756`
  emits `COutPacket(160)` = `0xA0` with `Encode1(0x1D)`. `0xA0` is
  `CashShopOperationHandle` and `0x1D` = 29 is its `APPLY_WISHLIST` mode — a mode of an
  existing packet, not a packet of its own.

### Still unresolved

Handlers whose only sender is a multi-opcode function cannot be resolved by this
join — `CField::SendChatMsgSlash` @ `0x4c3e96` alone is `0x102a` bytes and backs
`AdminChat`, `AdminCommand`, `AdminLog`, `MatchTable` and `SueCharacter`, each with its
own `COutPacket` inside the one function. Splitting those needs per-arm reads of that
function rather than a sender-level lookup. The `DoActiveSkill_*` family all funnel
into `SendSkillUseRequest` (`0x46`), so they are covered by that entry rather than
being separate opcodes.
