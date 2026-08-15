# Scissors of Karma — IDA derivation record

Captured during the task-223 PRD interview (2026-08-13). Every claim below is a decompile of a named
function in an IDB listed by `idb_list`; addresses are given so the design phase can re-read rather than
re-derive. Nothing here is from general MapleStory knowledge.

IDBs used:

| Session | Binary | Role |
|---|---|---|
| `41f13e0d` | `GMS/v83_Me/MapleStory_dump.exe.i64` | baseline; richer handler naming |
| `79906a1e` | `GMS/v95_0/GMS_v95.0_U_DEVM.exe.i64` | PDB-backed; the only IDB where the karma symbols are named |

## 1. Serverbound wire layout

`CUIKarmaDlg::_SendConsumeCashItemUseRequest`

- gms_v83 `@0x830FB5` — `COutPacket(0x4F)`:
  `Encode2(m_nPOS)` · `Encode4(m_nItemID)` · `Encode4(m_nTargetTI)` · `Encode4(m_nTargetPOS)` ·
  `Encode4(get_update_time())`
- gms_v95 `@0x7D7EF0` — `COutPacket(85 = 0x55)`:
  `Encode4(get_update_time())` · `Encode2(m_nPOS)` · `Encode4(m_nItemID)` · `Encode4(m_nTargetTI)` ·
  `Encode4(m_nTargetPOS)`

Both opcodes match the `USE_CASH_ITEM` row at `docs/packets/audits/STATUS.md:588` (`0x04F` for gms_v83,
`0x055` for gms_v95), which already names this function among that opcode's senders.

The leading `Encode2 + Encode4` pair is the existing `ItemUse` common header
(`libs/atlas-packet/cash/serverbound/item_use.go:37-77`); the `updateTime` position difference is the
already-modeled `UpdateTimeFirst` gate (`item_use.go:21-23`). **The karma sub-body is therefore
`int32 nTargetTI + int32 nTargetPOS`** — byte-identical in shape to `ItemUseSeal`, and *not* the bare-`int16`
`ItemUseTargetSlot` used by Item Tag and the expiration extenders.

### Exclusive-request lock

gms_v83 `@0x830FB5` gates the send on `CWvsContext::CanSendExclRequest(500, 0)` and, after sending, sets the
excl-request flag and stamps `get_update_time()`. The server must unlock on every outcome, refusals included.

### Confirming strings

- gms_v83 chat log on success:
  `SP_4664_YOU_HAVE_USED_THE_SCISSORS_OF_KARMA_SO_1_TIME_OF_TRADING_HAS_BEEN_ENABLED`
- gms_v95 formats the same message with the scissors' display name via
  `CItemInfo::GetKarmaScissorsName(m_nKarmaType)` (`@0x5B4B20`), confirming multiple scissors variants by v95.

## 2. Eligibility gates

`CUIKarmaDlg::PutItem` — gms_v95 `@0x7D7BA0`. Refuses in this order:

1. `pItem->IsProtectedItem()` → notice string `0xD8F`
2. `CItemInfo::GetAppliableKarmaType(nItemID) != this->m_nKarmaType` → notice string `6764`
3. `pItem->IsPossibleTradingItem()` → notice string `4680`

Only after all three does it store `m_nTargetTI` / `m_nTargetPOS`.

Gate 2 is an **equality** test against the scissors' own karma type — not a non-zero test. Two scissors
variants with different karma types gate different target sets.

### The eligibility property

`CItemInfo::GetAppliableKarmaType` — gms_v95 `@0x5C09F0`:

```c
if ( nItemID / 1000000 == 1 )   // equips
  return GetEquipItem(nItemID)->nAppliableKarmaType;
else
  return GetBundleItem(nItemID)->nAppliableKarmaType;
```

So the property lives on both `EQUIPITEM` and `BUNDLEITEM` — it is not equip-only.

The scissors' own karma type is loaded by `CItemInfo::RegisterKarmaScissorsItem` (gms_v95 `@0x5A1120`) into a
`KARMASCISSORSITEM { nItemID, nKarmaType }` record, keyed into `m_mKarmaScissorsItem`. It reads a child node
named by `StringPool` id `0x3D5` (the container, expected to be `info`) and then an integer property named by
`StringPool` id `5595`.

> **Unresolved:** neither StringPool id was resolved to its literal string in this pass. The design phase must
> read the actual WZ property spellings out of a WZ extract or the ingested atlas-data corpus. Do not guess.

## 3. The karma mark bit

Read off `nAttribute` (via `_ZtlSecureFuse<short>` over the secure-tear pair), gms_v95:

| Function | Address | Returns |
|---|---|---|
| `GW_ItemSlotEquip::IsProtectedItem` | `0x4F60B0` | `nAttribute & 0x01` |
| `GW_ItemSlotBundle::IsProtectedItem` | `0x4F6780` | `nAttribute & 0x01` |
| `GW_ItemSlotEquip::IsPossibleTradingItem` | `0x4F6130` | `(nAttribute & 0x10) >> 4` |
| `GW_ItemSlotBundle::IsPossibleTradingItem` | `0x4F67A0` | `(nAttribute & 0x02) >> 1` |

**The karma mark is a different bit depending on slot class: `0x10` on an equip, `0x02` on a bundle.**
"Protected" is `FlagLock` (0x01) in both — a Sealing-Lock'd item cannot be karma'd.

Mapped onto `libs/atlas-constants/asset/flag.go`, the names read backwards from the client's usage:

- `FlagKarmaUse = 0x02` (`flag.go:8`) is the **bundle** karma bit — and shares its value with
  `FlagSpikes = 0x02` (`flag.go:7`), which is an **equip** concept. No true conflict; total context
  dependence.
- `FlagKarmaEquip = 0x10` (`flag.go:11`) is the **equip** karma bit.

This explains the existing round-trip defect: `SetKarmaUsed` writes `0x10` (equip-correct) while
`KarmaUsed()` reads `0x02` (bundle-correct), so a set never reads back. Same inverted pair in
`atlas-inventory`, `atlas-channel`, `atlas-login`, and `atlas-cashshop`. See PRD FR-4.

Note also that `IsPossibleTradingItem() == true` means *the mark is already set*, which is why gate 3 above
refuses — you cannot karma an item that already has a free trade pending.

## 4. Not yet derived

- `GW_ItemSlotPet::IsPossibleTradingItem` (gms_v95 `@0x4F6A70`) — exists, not decompiled. Determines whether
  pets are a valid karma target (PRD OQ-5).
- The v83 bit values. `IsPossibleTradingItem` / `IsProtectedItem` / `GetAppliableKarmaType` are **unnamed** in
  the v83, v84, and v92 IDBs — a `name_regex` sweep for `Karma|IsPossibleTradingItem|IsProtectedItem`
  returned only the v83 sender. Unnamed is not absent: the design phase must locate these functions in the
  v83 IDB, name them, and confirm the `0x10` / `0x02` split holds at the baseline (PRD OQ-2).
- `5520001` (introduced GMS v84) — its karma type and target set are unread (PRD OQ-4).
