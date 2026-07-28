# v48 Stage E — BATCH 4: cash (CCashShop / CashShop) family

Anchor v61 fast-path. IDB port 13337 (GMS_v48_1_DEVM.exe). Branch
`task-113-gms-legacy-versions`.

## Result summary

- **8 cells promoted** (v48 verified 40 → 48): 7 serverbound CASHSHOP_OPERATION
  sub-struct arms + 1 clientbound CASHSHOP_CASH_ITEM_RESULT op-cell (9-arm
  family).
- **9 serverbound arms n-a** (genuinely absent; dispositioned via
  `_unimplemented.json` + stripped unresolved export stubs). They render ❌ in
  the matrix because the tool has **no sub-struct n-a state** (zero sub-struct
  `n-a` cells exist repo-wide) — identical to how v61/v72/v79 leave these same
  legacy-absent arms ❌.
- **0 blocked.**
- matrix `--check` exit 0; problem-grep 0; v48 conflicts 0; all existing
  versions hold (v61 208 / v72 216 / v79 228 / v83 367 / v84 345 / v87 379 /
  v95 399 / jms 362). Non-cash matrix drift: 0.

## v48 CCashShop::OnCashItemResult (clientbound) mode table

Dispatcher @0x4537a8: `Decode1(mode)` then a ~40-case switch spanning modes
0x29–0x6B (mostly unnamed `sub_XXXX`; Atlas models 9 arms). **In v48 the result
dispatcher's OUTER opcode is 256 (CASHSHOP_CASH_ITEM_RESULT)** — not the
CASHSHOP_OPERATION slot used from v61 (255) onward. This is the Δ≈-36 cash
clientbound opcode block shift Stage A flagged. The per-mode dispatch bytes are
resolved from the tenant operations table, not the codec; the codec BODY gates
only on `MajorVersion()>12` and a lone `>=95` field (shop_inventory.go:133), so
v48 (>12, <95) bodies are byte-identical to the IDA-verified v83 encode — same
cross-version-equality discipline as the accepted v61/v72/v79 fixtures.

## v48 CASHSHOP_OPERATION (serverbound) mode table, re-derived from the switch

All arms send `COutPacket(160) + Encode1(mode) + body`. Re-derived per send-site
(NOT copied from v61 — distrust IDB names, body-verified from each COutPacket):

| Atlas arm | v48 fname @addr (send) | mode | body (mode stripped) | vs v61 |
|---|---|---|---|---|
| Buy | OnBuy @0x44b0cf (0x44b38a) | 2 | isPoints(1)+serial(4) | v61 has currency(4) — **v48 drops it** |
| BuyCouple | OnBuyCouple @0x44b4c1 (0x44b79b) | 0x1A | isPoints(1)+serial(4) | v61 legacy has currency — **v48 drops it** |
| BuyPackage | OnBuyPackage @0x44b837 (0x44b9e1) | 0x1C | serial(4) | = v61 legacy |
| Gift | OnGift @0x44ba5d (0x44bd4e) | 3 | birthday(4)+serial(4)+name+message | = GMS<87 gift path |
| SetWishlist | OnSetWish @0x44ce9b (0x44cf78) | 4 | 10×serial(4) | = v61 |
| BuyNormal | OnBuyNormal @0x44cbb2 (0x44cdaf) | 0x1F | spw(4)+serial(4)+name+message | **v48 mislabel** — gift-shaped, not v83+ serial-only |
| BuyFriendship | OnBuyFriendship @0x44c879 (0x44cadb) | (serial/1000==9110)+5 | pointType(1)+flag(1)+serial(4) | **v48** friendship-ring/equip-slot buy; flag byte where v61 has currency int |

## Codec gates added (legacy range only; v61+ unchanged)

- `buyOmitsCurrency(t) = GMS && MajorVersion < 61` — Buy/BuyCouple/BuyFriendship/
  BuyNormal drop the currency int below v61.
- BuyFriendship: new `flag byte` field, v48 legacy path pointType+flag+serial.
- BuyNormal: new spw/name/message fields, v48 legacy path (gift-shaped).
- Round-trip tests (buy/buy_couple/buy_friendship) guarded for the <61 path.

## Arms n-a'd (genuinely v48-absent; decompile evidence in _unimplemented.json)

Unresolved export stubs stripped (8) + IncCharacterSlot (already absent from
export). `func_query` over GMS_v48_1_DEVM.exe returns no send function for any:

- BuyNameChange (no name-change op), BuyWorldTransfer (world transfer is
  `sub_44FB95` → **COutPacket(20)**, a separate opcode, NOT a mode-160 arm),
  EnableEquipSlot (folded into OnBuyFriendship ring buy), IncreaseCharacterSlot,
  IncreaseInventory (OnBuySlotInc), IncreaseStorage (OnIncTrunkCount),
  MoveFromCashInventory (OnMoveCashItemLtoS), MoveToCashInventory
  (OnMoveCashItemStoL), RebateLockerItem.

## CASHSHOP_OPERATION serverbound op-cell — cannot promote (not a regression)

Grades worst-of-siblings; the 9 absent arms render ❌, so the op stays
incomplete. **v61 (the anchor) has this same op-cell incomplete for the same
reason** — v72/v79 promote it only because those versions implement more arms.
This is a tool limitation (no sub-struct n-a), not a v48-specific gap.

## Verification

- `go test -race ./libs/atlas-packet/cash/...` green (both packages).
- `go vet ./libs/atlas-packet/cash/...` clean.
- matrix `--check` exit 0; problem-grep 0; conflicts 0; no version dropped; 0
  non-cash matrix drift (git-verified `git show --stat` touches only cash +
  matrix files).
- Branch `task-113-gms-legacy-versions` after both commits.

## Commits

1. `de138722b8` — serverbound operation family (7 verified + 9 n-a).
2. `f5c08550a0` — clientbound OnCashItemResult result family (op verified).
