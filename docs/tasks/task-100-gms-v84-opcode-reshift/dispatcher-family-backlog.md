# Dispatcher-family body-verification backlog

## Why this exists

The packet coverage matrix grades at **(op, version)** granularity. A handful of
ops are **mode-prefix dispatchers**: one opcode carries N logically distinct
sub-packets, selected by a leading mode/discriminator byte, each arm reading a
different body. The audit export for such a function is truncated to the leading
`Decode1(mode)`, so the flat diff compared 1-byte-vs-1-byte and "matched" — and a
single sub-handler's byte-fixture promoted the **whole op** to ✅. That overstated
coverage: the remaining mode arms were neither implemented nor verified.

Fixed by the 🧩 **family** cell state (see `docs/packets/evidence/families.yaml`
and `tools/packet-audit/internal/matrix/grade.go`): a dispatcher op is now capped
at 🧩 and can never reach ✅ on one sub-handler. This document tracks the real
remaining work — verifying (and, where Atlas should support them, implementing)
the per-mode arms — so it is **visible** rather than hidden behind a green cell.

## The 7 dispatcher ops (all clientbound) — capped at 🧩

Each maps to one mode-byte dispatcher (decompile-confirmed, v95 PDB). "Arms" =
number of switch cases in the v95 dispatcher. "Currently fixtured" = the single
Atlas writer/arm the matrix joins today.

| Op | Dispatcher fname | v95 addr | Arms | Currently fixtured (one arm) |
|----|------------------|----------|------|------------------------------|
| CASHSHOP_OPERATION | `CCashShop::OnCashItemResult` | 0x499370 | 57 | `cash/clientbound/CashCashShopInventory` (load-locker) |
| MTS_OPERATION | `CITC::OnNormalItemResult` | 0x5771d0 | 35 | `field/clientbound/FieldMtsOperation` |
| OPEN_NPC_SHOP | `CShopDlg::OnPacket` | 0x6eb7d0 | mode | `npc/clientbound/NpcShopOperationGenericError` |
| CONFIRM_SHOP_TRANSACTION | `CShopDlg::OnPacket` | 0x6eb7d0 | mode | `npc/clientbound/NpcShopOperationGenericError` |
| MESSENGER (clientbound) | `CUIMessenger::OnPacket` | 0x7f5e40 | 9 | `messenger/clientbound/MessengerAdd` |
| PLAYER_INTERACTION | `CMiniRoomBaseDlg::OnPacketBase` | 0x639e10 | multi | `interaction/clientbound/InteractionInteractionChat` |
| STORAGE | `CTrunkDlg::OnPacket` | 0x76a990 | mode | `storage/clientbound/StorageShow` |

### Per-family work to close the 🧩

For each family, the backlog is: decompose the dispatcher into its mode arms,
decide which arms Atlas must support, then per supported arm — add the codec,
write a byte-fixture (`// packet-audit:verify` marker), pin evidence. A family
should only graduate off 🧩 once **every supported arm** is covered (the grader
keeps it capped until then; full graduation to ✅ requires the per-arm coverage
model, a follow-up — see "Grader follow-up" below).

- **CASHSHOP_OPERATION** (`OnCashItemResult`, 57 arms): buy/gift/coupon/locker/
  wish/package/gachapon/transfer-world/maple-point arms (cases 0x54–0xBC). Only
  load-locker is fixtured.
- **MTS_OPERATION** (`OnNormalItemResult`, 35 arms): ITC list/search/register/
  sale/wish/buy/zzim arms (cases 0x15–0x3E).
- **OPEN_NPC_SHOP / CONFIRM_SHOP_TRANSACTION** (`CShopDlg::OnPacket`): most mode
  arms are a StringPool notice (no further fields); arms 0xE/0xF append `Decode4`,
  arm 0x11 reads `Decode1+DecodeStr`. Only the generic-error arm is modeled.
- **MESSENGER** (`CUIMessenger::OnPacket`, 9 arms): Enter/SelfEnterResult/Leave/
  Invite/InviteResult/Blocked/Chat/Avatar/Migrated (cases 0–8). Only add/enter
  is fixtured.
- **PLAYER_INTERACTION** (`CMiniRoomBaseDlg::OnPacketBase`): invite/enter/avatar/
  leave/check-ssn arms + the trade/minigame sub-dialog dispatch. Only chat.
- **STORAGE** (`CTrunkDlg::OnPacket`): get-items arms (0xD/0xF/0x13), notice arms,
  set-trunk (0x16), error-with-string (0x18). Only show is modeled.

## Confirmed NOT family (do not re-flag)

Decompiled and verified to be a different shape — these grade normally and their
✅ are legitimate:

- **Opcode (nType) demuxers** — switch on the *opcode*, each leaf is its own
  registry op with its own opcode + body: `CField::OnPacket`,
  `CField_Tournament::OnPacket` (TOURNAMENT*), `CField_Witchtower::OnPacket`
  (ARIANT_SCORE / WITCH_TOWER_SCORE_UPDATE), `CSummonedPool::OnPacket`
  (SPAWN/REMOVE_SPECIAL_MAPOBJECT, MOVE/SUMMON_ATTACK/...).
- **Flat packets** (no switch): `CCashShop::OnQueryCashResult` (3 ints),
  `CITC::OnQueryCashResult` (2 ints) → QUERY_CASH_RESULT.

## Candidates still to verify (not yet classified)

Do NOT add to `families.yaml` without decompiling first (no guessing):

- **MTS_OPERATION2** — fnames `CField::OnCharacterSale` (forwards on nType to
  `m_pCharacterSaleDlg->OnPacket`) + `CITC::OnQueryCashResult` (flat). The
  CharacterSale dialog's `OnPacket` was not decompiled here; classify it before
  deciding whether MTS_OPERATION2 is a family.
- **Serverbound shop/cash/messenger sends** (NPC_SHOP, FREDRICK_ACTION,
  DUEY_ACTION, MESSENGER serverbound, ENTER_MTS, …) — currently ❌, so not a
  false-pass today, but several are themselves send-side mode dispatchers; if
  they are ever fixtured on one arm, they need the same family cap.

## Grader follow-up (out of scope here)

The family state *prevents the false pass* but does not yet *track per-arm
coverage*. A full model would decompose each dispatcher into per-mode sub-rows so
each arm is independently gradeable and a family graduates to ✅ only when all
supported arms are verified. That is a larger packet-audit change; this doc is the
input to it.
