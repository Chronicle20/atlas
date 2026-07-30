# GMS v48 clientbound opcode map (IDA-derived)

IDB: `GMS_v48_1_DEVM.exe.i64` (ida-pro-mcp session `0bb5f11a`). Read-only derivation.
All opcodes are the raw `CInPacket::Decode2` value, which is what a tenant socket
template's `opCode` field carries (established in task-139: v48 `CPet::SendDropPickUpRequest`
emits 116 = template `0x74`).

## Top-level routing — `CClientSocket::ProcessPacket` @ `0x464fb6`

| Opcode | Target |
|---|---|
| `0x10` | `CClientSocket::OnMigrateCommand` |
| `0x11` | `CClientSocket::OnAliveReq` |
| `0x12` | `CClientSocket::OnAuthenCodeChanged` |
| `0x13` | `CClientSocket::OnAuthenMessage` |
| `0x14` | `CSecurityClient::OnPacket` @ `0x733b6b` |
| `0x15` | `sub_465378` |
| `0x18`–`0x47` | `CWvsContext::OnPacket` @ `0x70d215` |
| everything else | stage instance virtual `OnPacket` (`CField::OnPacket` @ `0x4c66f2`) |

Login-phase packets go through the stage instance while `CLogin` is the active stage
(`CLogin::OnPacket` @ `0x5007c4`).

## `CWvsContext::OnPacket` @ `0x70d215` (opcodes 25–70 / `0x19`–`0x46`)

| Dec | Hex | Client function |
|---|---|---|
| 25 | `0x19` | `OnInventoryOperation` |
| 26 | `0x1A` | `OnInventoryGrow` |
| 27 | `0x1B` | `OnStatChanged` |
| 28 | `0x1C` | `OnTemporaryStatSet` |
| 29 | `0x1D` | `OnTemporaryStatReset` |
| 30 | `0x1E` | `OnChangeSkillRecordResult` |
| 31 | `0x1F` | `OnSkillUseResult` |
| 32 | `0x20` | `OnGivePopularityResult` |
| 33 | `0x21` | `OnMessage` |
| 34 | `0x22` | `OnMemoResult` |
| 35 | `0x23` | `OnMapTransferResult` |
| 36 | `0x24` | `OnAntiMacroResult` |
| 37 | `0x25` | `OnClaimResult` |
| 38 | `0x26` | `OnSetClaimSvrAvailableTime` |
| 39 | `0x27` | `OnClaimSvrStatusChanged` |
| 40 | `0x28` | `OnSetTamingMobInfo` |
| 41 | `0x29` | `OnQuestClear` |
| 42 | `0x2A` | `OnEntrustedShopCheckResult` |
| 43 | `0x2B` | `OnSkillLearnItemResult` |
| 44 | `0x2C` | `OnSueCharacterResult` |
| 46 | `0x2E` | `OnTradeMoneyLimit` |
| 47 | `0x2F` | `OnSetGender` |
| 49 | `0x31` | `OnCharacterInfo` |
| 50 | `0x32` | `OnPartyResult` |
| 51 | `0x33` | `OnAllianceResult` |
| 53 | `0x35` | `OnGuildResult` |
| 54 | `0x36` | `OnTownPortal` |
| 55 | `0x37` | `OnBroadcastMsg` |
| 56 | `0x38` | `OnIncubatorResult` |
| 57 | `0x39` | `OnShopScannerResult` |
| 58 | `0x3A` | `sub_72025D` |
| 59 | `0x3B` | `OnMarriageRequest` |
| 60 | `0x3C` | `OnMarriageResult` |
| 61 | `0x3D` | `OnWeddingGiftResult` |
| 62 | `0x3E` | `sub_713202` |
| 63 | `0x3F` | `OnCashPetFoodResult` |
| 64 | `0x40` | `OnMapleTVUseRes` |
| 65 | `0x41` | `OnAvatarMegaphoneRes` |
| 66 | `0x42` | `OnSetAvatarMegaphone` |
| 67 | `0x43` | `OnClearAvatarMegaphone` |
| 68 | `0x44` | `sub_721481` |
| 69 | `0x45` | `OnDestroyShopResult` |
| 70 | `0x46` | `sub_7215EA` |

Gaps (no case): 45, 48, 52 — genuinely unhandled by this build.

## `CField::OnPacket` @ `0x4c66f2`

Direct cases:

| Dec | Hex | Client function |
|---|---|---|
| 77 | `0x4D` | `CField::OnTransferFieldReqIgnored` |
| 78 | `0x4E` | `CField::OnTransferChannelReqIgnored` |
| 79 | `0x4F` | `CField::OnFieldSpecificData` |
| 80 | `0x50` | `CField::OnGroupMessage` |
| 81 | `0x51` | `CField::OnWhisper` |
| 82 | `0x52` | `CField::OnCoupleMessage` |
| 83 | `0x53` | `CField::OnSummonItemInavailable` |
| 84 | `0x54` | `CField::OnFieldEffect` |
| 85 | `0x55` | `CField::OnBlowWeather` |
| 86 | `0x56` | `CField::OnPlayJukeBox` |
| 87 | `0x57` | `CField::OnAdminResult` |
| 88 | `0x58` | `CField::OnQuiz` |
| 89 | `0x59` | `CField::OnDesc` |
| 90 | `0x5A` | virtual call (vtable slot 7) |
| 93 | `0x5D` | `sub_4CBB78` |
| 94 | `0x5E` | `sub_4CBC9A` |
| 95 | `0x5F` | `CField::OnSetQuestTime` |
| 96 | `0x60` | `CField::OnWarnMessage` |
| 97 | `0x61` | `CField::OnSetObjectState` |
| 98 | `0x62` | `CField::OnDestroyClock` |
| 232 | `0xE8` | `sub_5EC4F3` |
| 237 | `0xED` | `CRPSGameDlg::OnPacket` @ `0x5adb94` |
| 238 | `0xEE` | `CUIMessenger::OnPacket` @ `0x61d8b8` |
| 239 | `0xEF` | `CMiniRoomBaseDlg::OnPacketBase` @ `0x5459c4` |
| 247 | `0xF7` | `CTrunkDlg::OnPacket` @ `0x58332c` |

Range delegations:

| Range (dec) | Range (hex) | Sub-dispatcher |
|---|---|---|
| 72–75 | `0x48`–`0x4B` | `CStage::OnPacket` @ `0x5c45ed` |
| 99–156 | `0x63`–`0x9C` | `CUserPool::OnPacket` @ `0x6b2710` |
| 157–175 | `0x9D`–`0xAF` | `CMobPool::OnPacket` @ `0x559340` |
| 176–186 | `0xB0`–`0xBA` | `CNpcPool::OnPacket` @ `0x56d413` |
| 187–191 | `0xBB`–`0xBF` | `CEmployeePool::OnPacket` @ `0x4b3264` |
| 192–195 | `0xC0`–`0xC3` | `CDropPool::OnPacket` @ `0x4aaccf` |
| 196–200 | `0xC4`–`0xC8` | `CMessageBoxPool::OnPacket` @ `0x54329b` |
| 201–204 | `0xC9`–`0xCC` | `CAffectedAreaPool::OnPacket` @ `0x42182f` |
| 205–208 | `0xCD`–`0xD0` | `CTownPortalPool::OnPacket` @ `0x5e318d` |
| 209–214 | `0xD1`–`0xD6` | `CReactorPool::OnPacket` @ `0x5a5390` |
| 225–227 | `0xE1`–`0xE3` | `CScriptMan::OnPacket` @ `0x5b0ace` |
| 228–231 | `0xE4`–`0xE7` | `CStoreBankDlg::OnPacket` @ `0x5b7a38` |
| 233–236 | `0xE9`–`0xEC` | `z_MISLABELED_notRPS_channelFindDlg` @ `0x5d5544` |
| 263–266 | `0x107`–`0x10A` | `CFuncKeyMappedMan::OnPacket` @ `0x4e5f06` |
| 267–272 | `0x10B`–`0x110` | `sub_527238` |

## `CStage::OnPacket` @ `0x5c45ed`

| Dec | Hex | Client function |
|---|---|---|
| 73 | `0x49` | **`CStage::OnSetField`** @ `0x5c4616` |
| 74 | `0x4A` | `CStage::OnSetITC` @ `0x5c4d9c` |

**`SET_FIELD` exists in v48 at `0x49`.** `docs/packets/audits/status.json` currently records
`SET_FIELD × gms_v48` as `n-a` with `opcode: -1`, which is false — the same class of error
already recorded for `PET_AUTO_POT × gms_v48`. The v48 column carries 628 `n-a` cells, 194 of
which are `verified` on gms_v83; that column cannot be used as an applicability oracle.

## Pool sub-dispatchers

`CUserPool::OnPacket` @ `0x6b2710`: 100 `OnUserEnterField`, 101 `OnUserLeaveField`,
102–124 → `OnUserCommonPacket` @ `0x6b2a51`, 125–143 → `OnUserRemotePacket` @ `0x6b2ae4`,
144–155 → `OnUserLocalPacket` @ `0x6b2c6d` → `CUserLocal::OnPacket` @ `0x6a048d`.

`OnUserCommonPacket` (102–124): 103 `CUser::OnChat`, 104 `CUser::OnADBoard`,
105 `CUser::OnMiniRoomBalloon`, 106 `CUser::SetConsumeItemEffect`,
107 `CUser::ShowItemUpgradeEffect`, 108–115 → `CUser::OnPetPacket` @ `0x69221b`,
116–123 → `CSummonedPool::OnPacket` @ `0x692281`.

`OnUserRemotePacket` (125–143): 126 `OnMove`, 127–129 `OnAttack`, 130 `OnSkillPrepare`,
131 `OnMovingShootAttackPrepare`, 132 `OnHit`, 133 `CAvatar::SetEmotion`,
134 `CUser::SetActiveEffectItem`, 135 inline store, 136 `OnAvatarModified`,
137 `CUser::OnEffect`, 138 `OnSetTemporaryStat`, 139 `OnResetTemporaryStat`,
140 `OnReceiveHP`, 141 `OnGuildNameChanged`, 142 `OnGuildMarkChanged`.

`CUserLocal::OnPacket` (144–155): 145 `sub_6A69B1`, 146 `CUser::OnEffect`, 147 `sub_6A6941`,
149 `sub_6A6C47`, 150 `sub_6A6CD1`, 151 `sub_6A6CFB`, 152 `sub_6A83ED`, 153 `nullsub_12`,
154 `sub_6A87A0`.

`CSummonedPool::OnPacket` (116–123): 117 virtual, 118 special-cased, 119 `OnMove`,
120 `OnAttack`, 121 `sub_5DB064`, 122 `OnSkill`.

`CMobPool::OnPacket` @ `0x559340`: 158 `OnMobEnterField`, 159 `OnMobLeaveField`,
160 `OnMobChangeController`, 161–174 → `OnMobPacket` @ `0x559390`
(162 `OnMove`, 163 `OnCtrlAck`, 165 `OnStatSet`, 166 `OnStatReset`, 167 `OnSuspendReset`,
168 `OnAffected`, 169 `OnDamaged`, 170 `OnSpecialEffectBySkill`, 172 `sub_5511F4`,
173 `sub_551481`).

`CNpcPool::OnPacket` @ `0x56d413`: 177 `OnNpcEnterField`, 178 `OnNpcLeaveField`,
179 `OnNpcChangeController`, 180–182 → `OnNpcPacket` @ `0x56d48a`, 183–185 → `sub_56D510`.

`CDropPool::OnPacket` @ `0x4aaccf`: 193 `OnDropEnterField`, 194 `OnDropLeaveField`.

---

## Root cause (established with the repo's own tooling, not by hand)

`RE_AUDITING_A_COLUMN.md` documents the maintenance path for this exact
situation. Running it:

```
packet-audit validate -version gms_v48 \
  -ida-url http://192.168.20.3:8745/mcp -ida-database <session> \
  -report docs/tasks/task-188-v48-template-completeness/validate-gms_v48.md
```

→ `verified 287 / divergent 58 / missing-mode 0 / extra-mode 3 / unverifiable 701`

The dominant `unverifiable` reason is
`base resolve failed: ... Failed to parse address (missing 0x prefix):` — the
committed baseline export carries an **empty `address`** for those entries.
`CStage::OnSetField` is one of them.

Address coverage in `docs/packets/ida-exports/`:

| Version | Entries with NO address |
|---|---|
| `gms_v48` | **667 / 1046 (63%)** — 441 distinct base functions |
| `gms_v61` | **384 / 1003 (38%)** |
| `gms_v83` | 0 / 718 |
| `gms_v95` | 0 / 756 |

So v48 and v61 are half-harvested columns, while v83/v95 are complete. An entry
with no address can never be verified, which is how the v48 matrix column
accumulated 628 `n-a` cells — absence of a resolved address was recorded as
absence of the feature. The template is thin as a downstream consequence.

The hand-derived dispatch map above remains valid as an independent cross-check
of whatever the re-harvest produces (`CStage::OnSetField` @ `0x5c4616`, opcode
`0x49`), but the primary fix is to re-resolve the export's addresses rather than
to hand-populate the template.

**Note:** the documented toolchain could not target the current IDA-MCP server
until this task added `-ida-database` to the maintenance subcommands
(commit `d7f233f5e`); only `export` had it.
