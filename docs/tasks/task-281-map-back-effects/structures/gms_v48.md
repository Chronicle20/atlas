# gms_v48 — SET_BACK_EFFECT / CLEAR_BACK_EFFECT: ABSENT

IDB session: `12a398ce` (`GMS_v48_1_DEVM.exe.i64`)
Export: none used — live IDB.

## Router arms present

There is **no `CMapLoadable::OnPacket` on this version.** The IDB symbolizes 33
distinct `*::OnPacket` methods (`?OnPacket@CAffectedAreaPool@@`,
`?OnPacket@CCashShop@@`, ..., `?OnPacket@CWvsContext@@`) and
`?OnPacket@CMapLoadable@@` is not among them. `CMapLoadable` itself exists but
only three of its methods are symbolized: `TransientLayer_NewYear` @ `0x477419`,
`LoadMap` @ `0x5291bb`, `PrepareNextBGM` @ `0x53034e` — none takes a `CInPacket&`.

The field-loadable router is therefore `CField::OnPacket` @ `0x4c66f2`, and its
arms are, in full:

Direct switch (opcode <= 232):

| opcode | handler |
|---|---|
| 0x4D (77) | `CField::OnTransferFieldReqIgnored` @ `0x4c6b01` |
| 0x4E (78) | `CField::OnTransferChannelReqIgnored` @ `0x4c6be7` |
| 0x4F (79) | `CField::OnFieldSpecificData` @ `0x4c6ca1` |
| 0x50 (80) | `CField::OnGroupMessage` @ `0x4c6dd6` |
| 0x51 (81) | `CField::OnWhisper` @ `0x4c71d5` |
| 0x52 (82) | `CField::OnCoupleMessage` @ `0x4c6faf` |
| 0x53 (83) | `CField::OnSummonItemInavailable` @ `0x4c7b1f` |
| 0x54 (84) | `CField::OnFieldEffect` @ `0x4c7b59` |
| 0x55 (85) | `CField::OnBlowWeather` @ `0x4c930a` |
| 0x56 (86) | `CField::OnPlayJukeBox` @ `0x4c95f2` |
| 0x57 (87) | `CField::OnAdminResult` @ `0x4c96c4` |
| 0x58 (88) | `CField::OnQuiz` @ `0x4c9e20` |
| 0x59 (89) | `CField::OnDesc` @ `0x4ca491` |
| 0x5A (90) | virtual call `(*(*(this-2) + 28))(this-2, iPacket)` — CField vtable slot 7 |
| 0x5D (93) | `sub_4CBB78` @ `0x4cbb78` |
| 0x5E (94) | `sub_4CBC9A` @ `0x4cbc9a` |
| 0x5F (95) | `CField::OnSetQuestTime` @ `0x4cbcad` |
| 0x60 (96) | `CField::OnWarnMessage` @ `0x4ca7d4` |
| 0x61 (97) | `CField::OnSetObjectState` @ `0x4cbe02` |
| 0x62 (98) | `CField::OnDestroyClock` @ `0x4c6aef` |
| 232 | `sub_5EC4F3` @ `0x5ec4f3` |
| 237 | `CRPSGameDlg::OnPacket` |
| 238 | `CUIMessenger::OnPacket` |
| 239 | `CMiniRoomBaseDlg::OnPacketBase` |
| 247 | `CParcelDlg::OnPacket` |

Range fall-through arms (the `LABEL_29` chain): 72–75 `CStage::OnPacket`;
99–156 `CUserPool::OnPacket`; 157–175 `CMobPool::OnPacket`; 176–186
`CNpcPool::OnPacket`; 187–191 `CEmployeePool::OnPacket`; 192–195
`CDropPool::OnPacket`; 196–200 `CMessageBoxPool::OnPacket`; 201–204
`CAffectedAreaPool::OnPacket`; 205–208 `CTownPortalPool::OnPacket`; 209–214
`CReactorPool::OnPacket`; 225–227 `CScriptMan::OnPacket`; 228–231
`CShopDlg::OnPacket`; 233–236 `z_MISLABELED_notRPS_channelFindDlg`; 263–266
`CFuncKeyMappedMan::OnPacket`; 267–272 `sub_527238`.

**No arm and no range in `CField::OnPacket` routes to any `CMapLoadable` router
or to any back-effect handler.** The only opcodes in the field block that reach
nothing at all are `0x5B` (91) and `0x5C` (92): they miss every `case` and every
range test and fall off the end of the function.

## Positive absence evidence

- **The opcode slots a back-effect pair would occupy are owned by other named
  ops.** On the next version up (gms_v61, session `921fdbb5`) the ops are
  routed by `CField::OnPacket` @ `0x4e9ea3` handing opcodes **95..96** to
  `CMapLoadable::OnPacket` @ `0x5a81b9`. On gms_v48 opcode 95 is
  `CField::OnSetQuestTime` (`0x4cbcad`) and opcode 96 is `CField::OnWarnMessage`
  (`0x4ca7d4`) — both in the direct switch, neither a back-effect handler. There
  is no CMapLoadable opcode window on this version at all.
- **No reachable back-layer alpha tween.** The gms_v61 handler shape is
  `sub_5A8316` @ `0x5a8316` — a `ZMap<long, ZRef<ZList<IWzGr2DLayer>>>::GetAt` on
  the field's back-layer member (`sub_5A8E05`) followed by a vtbl `+144` alpha
  `RelMove` — fed by the decode `sub_5163AE` @ `0x5163ae`. On gms_v48 the
  CMapLoadable neighbourhood (`0x5291bb`–`0x531607`) contains no analogue: the two
  functions occupying the positions their v61 counterparts hold relative to the
  reload function are `sub_530CB9` @ `0x530cb9` (the config reload, below) and
  `sub_531094` @ `0x531094` (a WZ-property/canvas layout helper — it takes no
  `CInPacket&` and performs no packet read; single xref from `sub_4D00BC` @
  `0x4d00bc`). Neither decodes anything.

  Scope note on method: this is a neighbourhood sweep plus the exhaustive router
  enumeration above, **not** an image-wide instruction-pattern scan for the
  `Decode1/Decode4/Decode1/Decode4` sequence — that scan was not run. It does not
  weaken the VERSION-ABSENT conclusion: `CField::OnPacket` @ `0x4c66f2` is the
  complete clientbound field router on this version, and no opcode in it reaches a
  back-effect handler. A tween function that no opcode can reach is not a packet.
- **Binary-wide search for `ReloadBack` / the whole-field back reload: 1
  candidate, and it is not packet-reachable.** The search was anchored on
  `CUser::GetVecCtrl` @ `0x472a6c`, the tail call of the reference `ReloadBack`
  shape; it has **82 xrefs in the whole image** (complete list, not truncated).
  The single candidate inside the `CMapLoadable` address neighbourhood is
  `sub_530CB9` @ `0x530cb9` (call site `0x530e9c`), and it carries the reference
  clear shape (`IWzGr2D::Getcenter` at vtbl `+100` with
  `stru_80F0C0`, then `sub_531607(0, 0)` at vtbl `+64`, then `sub_52B315(this)` =
  `RestoreBack`, then `CUser::GetVecCtrl` @ `0x472a6c`, then
  `CAnimationDisplayer::Effect_Quest` @ `0x42263c` — the adjacent-symbol drift for
  `SetCenterOrigin`). It has exactly **one xref**: `CConfig::ApplySysOpt` @
  `0x46b06d` (call site `0x46b098`). It is a video-option reload, entered only
  when `g_CConfig_pInstance` changes, and **no zero-read thunk and no packet
  handler calls it.**

  The contrast with gms_v61 is the discriminator: there, `ReloadBack`
  (`sub_5A81E2` @ `0x5a81e2`) also has exactly one xref — and that xref *is* the
  packet thunk `sub_5A871B` @ `0x5a871b`, the `case 96` arm of
  `CMapLoadable::OnPacket`. On gms_v48 no such thunk exists.
- `docs/packets/registry/gms_v48.yaml` carries no entry for either op, and the
  checked-in export records `CMapLoadable::OnSetBackEffect` as `unresolved: true`.
  Recorded for completeness only — neither was treated as evidence; the router
  enumeration and the three searches above are the proof.

Conclusion: `SET_BACK_EFFECT` and `CLEAR_BACK_EFFECT` do not exist on gms_v48.
Both cells are **VERSION-ABSENT**.
