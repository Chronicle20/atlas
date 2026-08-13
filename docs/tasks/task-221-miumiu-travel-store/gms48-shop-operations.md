# gms_48 `NPCShopOperation` derivation (OQ-4)

## Scope

Design §1.2 resolved OQ-4: v48's `?OnPacket@CStoreBankDlg@@SAXJAAVCInPacket@@@Z`
(decompiler-rendered type name; the function is `CShopDlg::OnPacket` — the
enclosing type name is a stale-local-type naming artifact, see design.md §1.2)
at `0x5b7a38` branches on `nType == 229` (allocate `CShopDlg` +
`SetShopDlg`) and `nType == 230` (transaction-result switch), giving
`OPEN_NPC_SHOP = 0xE5`, `CONFIRM_SHOP_TRANSACTION = 0xE6`.

This document derives the `NPCShopOperation` operations table for the
`nType == 230` arm (opcode `0xE6`) from the v48 IDB, case by case, rather than
copying `template_gms_83_1.json`'s table
(`[[bug_gms_61_72_79_interaction_operations_wrong]]`).

IDB used: `GMS_v48_1_DEVM.exe.i64`, session `93cc947e` (resolved from
`idb_list` by binary name, not by port — `[[reference_packet_audit_select_instance_dead]]`).

## Method

v48's `StringPool_GetString_decode_v48` (called via wrapper `sub_5D75AF` @
`0x5d75af`) resolves a numeric StringPool id to notice text at runtime by
decrypting an in-memory table (`sub_5D7774` @ `0x5d7774` — XOR/checksum loop
against `String.wz`-sourced data); the literal notice text is not embedded in
the executable and no `SP_*`/StringPool enum type exists in this IDB
(`type_query kind=enum filter=*String*|*SP_*|*Pool*` → 0 hits each), so the
notice text itself cannot be read directly from the binary. The mapping below
is therefore established **structurally**: by decompiling v83's already-known
`CShopDlg::OnPacket` @ `0x756da7` (whose `switch` arms carry the human-readable
`SP_*` enum constant names, e.g. `SP_852_YOU_DONT_HAVE_ENOUGH_IN_STOCK`) side
by side with v48's, and matching each v48 case to a v83 case on **both**
(a) case number and (b) decode mechanism (fixed-string `sub_5D75AF`/
`StringPool::GetString` call vs. the `Decode1()`-flag-gated conditional
`DecodeStr` used only by the "with reason" arm) — the same identification
method task 14 used to split v87/92/95's `GENERIC_ERROR`/
`GENERIC_ERROR_WITH_REASON`. Structural coincidence of case number alone is
not treated as sufficient; each row below states the mechanism match.

## v83 reference (known-good, `template_gms_83_1.json` — unchanged by this task)

Decompiled `CShopDlg::OnPacket` @ `0x756da7` (v83, session `41f13e0d`):

```
case 0:                 tab/scroll UI update, no notice           -> OK
case 1: case 5: case 9:  SP_852_YOU_DONT_HAVE_ENOUGH_IN_STOCK      -> OUT_OF_STOCK / _2 / _3
case 2: case 10:         SP_5599_YOU_DO_NOT_HAVE_ENOUGH_MESOS      -> NOT_ENOUGH_MONEY / _2
case 3:                  SP_853_PLEASE_CHECK_IF_YOUR_INVENTORY...  -> INVENTORY_FULL
case 4: case 8:          return; (no notice)                       -> (unmapped)
case 13:                 SP_5434_YOU_NEED_MORE_ITEMS                -> NEED_MORE_ITEMS
case 14:                 Decode4() + Format(SP_5450_...UNDER_LVD)  -> UNDER_LEVEL_REQUIREMENT
case 15:                 Decode4() + Format(SP_5449_...OVER_LVD)   -> OVER_LEVEL_REQUIREMENT
case 16:                 SP_3977_...TRADE_1_MILLION_MESOS_PER_DAY  -> TRADE_LIMIT
case 17:                 if (!Decode1()) goto default (SP_855...)  -> GENERIC_ERROR
                          else DecodeStr() -> dynamic Notice        -> GENERIC_ERROR_WITH_REASON
default:                 SP_855_DUE_TO_AN_ERROR_THE_TRADE_DID_NOT_HAPPEN -> GENERIC_ERROR
```

This confirms `template_gms_83_1.json`'s existing table (`GENERIC_ERROR` and
`GENERIC_ERROR_WITH_REASON` both bound to `17` — the same case number branches
on a secondary `Decode1()` flag, not on the switch value, so both names share
one numeric value) and is not itself modified by this task.

## v48 derivation

Decompiled `CShopDlg::OnPacket` @ `0x5b7a38` (v48, session `93cc947e`). The
`Decode1()` result (`v4`) is tested via a chain of subtractions
(`v14=v4-8`, `v15=v14-1`, `v16=v15-1`, `v17=v16-3`) rather than a literal
`switch` in the decompiler's rendering; each branch is restated below as the
`v4` value it corresponds to.

| `v4` (case) | Address | Mechanism | Notice string id | v83 case at same id/mechanism | Resulting name |
|---|---|---|---|---|---|
| 0 | `0x5b7b09`-`0x5b7b6a` | tab/scroll UI update, no notice call | — | case 0 (identical: tab/scroll UI update, no notice) | `OK` |
| 1 | `0x5b7c42` (via `LABEL_39`) | fixed `sub_5D75AF` notice | 774 | case 1/5/9 (fixed notice, shared id across 3 cases) | `OUT_OF_STOCK` |
| 2 | `0x5b7c24` (via `LABEL_37`) | fixed `sub_5D75AF` notice | 263 | case 2/10 (fixed notice, shared id across 2 cases) | `NOT_ENOUGH_MONEY` |
| 3 | `0x5b7b04` | fixed `sub_5D75AF` notice | 775 | case 3 (fixed notice, unique id) | `INVENTORY_FULL` |
| 4 | `0x5b7aea` | `return;` — no notice call at all | — | case 4/8 (identical: silent return) | *(no case in v83's table either — omitted)* |
| 5 | `0x5b7ad9`→`LABEL_39` | fixed `sub_5D75AF` notice, same target as `v4==1` | 774 | case 1/5/9 | `OUT_OF_STOCK_2` |
| 6 | `0x5b7ba5` (via `LABEL_32`) | fixed `sub_5D75AF` notice, catch-all target | 777 | falls to v83's `default:` (no distinct v83 case at 6) | *(unmapped default; not a named case in v83 either)* |
| 7 | `0x5b7ba5` (via `LABEL_32`) | same catch-all as `v4==6` | 777 | same as above | *(unmapped default)* |
| 8 | `0x5b7ae7`→`0x5b7aea` | `return;` — no notice call | — | case 4/8 | *(no case — omitted, same as `v4==4`)* |
| 9 | `0x5b7ad9`→`LABEL_39` | fixed `sub_5D75AF` notice, same target as `v4==1`/`5` | 774 | case 1/5/9 | `OUT_OF_STOCK_3` |
| 10 | `0x5b7ae0`→`LABEL_37` | fixed `sub_5D75AF` notice, same target as `v4==2` | 263 | case 2/10 | `NOT_ENOUGH_MONEY_2` |
| 11 | `0x5b7bb4` (via `LABEL_32`) | catch-all, same target as `v4==6/7` | 777 | falls to `default:` | *(unmapped default)* |
| 12 | `0x5b7bb4` (via `LABEL_32`) | catch-all | 777 | falls to `default:` | *(unmapped default)* |
| 13 | `0x5b7c0d` | fixed `sub_5D75AF` notice, unique id | 3626 | case 13 (fixed notice, unique id) — same case number | `NEED_MORE_ITEMS` |
| 14, `Decode1()==0` | `0x5b7bb4`(`goto LABEL_32`) | falls to catch-all (no numeric-level `Decode4()` read, unlike v83 case 14/15) | 777 | mechanism matches v83 case **17**'s `!Decode1()` branch (fixed catch-all notice), **not** v83 case 14 (which reads a numeric level via `Decode4()`+`Format`) | `GENERIC_ERROR` |
| 14, `Decode1()!=0` | `0x5b7bbd`-`0x5b7bd9` | `Decode1()` flag gate, then `CInPacket::DecodeStr` → dynamic `ZXString` assigned into the Notice call | dynamic (no fixed id) | mechanism matches v83 case **17**'s `Decode1()!=0` branch (`DecodeStr` → dynamic notice) exactly | `GENERIC_ERROR_WITH_REASON` |
| 15 | `0x5b7ba5` (via `LABEL_32`) | catch-all, same target as 6/7/11/12 | 777 | falls to `default:` | *(unmapped default; see round-2 note below — the "does the client render distinct text" test is not the right bar here, but 15 genuinely has no wire-value precedent in the family, unlike 16)* |
| 16 | `0x5b7ba5` (via `LABEL_32`) | catch-all, same target as 6/7/11/12/15/17+ — **traced individually, not bucketed** | 777 | mechanism matches, but see round-2 correction below: bound to `TRADE_LIMIT` by direct precedent, not omitted | `TRADE_LIMIT` |
| 17 and above | `0x5b7ba5` (via `LABEL_32`) | catch-all, same target as 6/7/11/12/15/16 | 777 | falls to `default:` | *(unmapped default)* |

Every one of 6, 7, 11, 12, 15, 16, 17, 18, … was traced **individually** through
the raw x86 disassembly of `0x5b7a38` (227 instructions, `disasm`, not just the
decompiler's rendering) to confirm none of them is tested by a distinct
`cmp`/`jz` anywhere in the function — the full `dec eax; jz` chain explicitly
tests only `{0,1,2,3,4,5,8,9,10,13,14}`, and everything else (with no
exception, including 16) falls through to the single `loc_5B7B96` → `loc_5B7BA0`
→ `push 309h` (777) catch-all at `0x5b7b96`-`0x5b7ba5`. This was re-verified at
the instruction level after round 1 undercounted the bucket (see "Round 2" below).

`GENERIC_ERROR` / `GENERIC_ERROR_WITH_REASON` bind to the same numeric value
(`14`) for the same reason v83 binds both to `17`: the switch dispatches once
on the primary `Decode1()` result to reach this arm, and a **second**
`Decode1()` call inside the arm — not a second switch case — decides between
the fixed catch-all message and the dynamic reason string. v48 compresses this
into `v4==14`; v83 places it at `v4==17`. The case *number* differs between
versions; the *mechanism* (single arm, secondary flag byte, conditional
`DecodeStr`) is what identifies both v48's `v4==14` and v83's `v4==17` as the
same pair of named operations.

## Round 2 correction: `TRADE_LIMIT` is bound at `16`, not omitted

**Round 1 of this document omitted `TRADE_LIMIT` on the reasoning "v48's
`v4==16` has no distinct client arm, therefore no name."** That test was wrong,
and code review caught it by pointing at in-repo, IDA-verified, TIER1-FIXTURE
evidence for the immediately-adjacent versions:

- `docs/packets/dispatchers/npc_shop_operation.yaml` lists
  `TRADE_LIMIT: { gms_v72: 16, gms_v79: 16, gms_v83: 16, gms_v84: 16, gms_v87: 16, gms_v95: 16, jms_v185: 16 }`.
- `docs/packets/evidence/gms_v61/npc.clientbound.NpcShopOperationTradeLimit.yaml`
  (`CShopDlg::OnPacket#TradeLimit` @ `0x64723c`) and the corresponding
  `libs/atlas-packet/npc/clientbound/shop_operation_v61_test.go` — comment:
  *"16 TRADE_LIMIT (default notice)"* — mode 16 is **not** a distinct arm in
  v61 either; it explicitly goes through the same default-notice bucket as
  `GENERIC_ERROR`, and is still registered as `TRADE_LIMIT: 16`.
- `docs/packets/evidence/gms_v72/...` and `shop_operation_v72_test.go` —
  comment: *"16 TRADE_LIMIT (856 default)"* — same pattern, `856` is the same
  string id `GENERIC_ERROR` uses in v72.

To settle whether v48 genuinely matches this pattern (rather than my having
mis-traced the bucket), I re-decompiled `CShopDlg::OnPacket` for v61
(`0x64723c`, session `415bf585`) and v79 (`0x6d6eb9`, session `1438cecd`)
directly. Both are **structurally identical** to v48's function — the exact
same `v4-8`/`v4-9`/`v4-10`/`v4-13` subtraction chain, testing only
`{8,9,10,13,14}` explicitly and falling everything else (6,7,11,12,**16**,
17,…) to one shared default-notice call. v79's test comment ("notice modes
…16… byte-identical to v83") refers to the *wire byte value* 16 matching
v83's numbering, not to v79 having a distinct client arm for it — confirmed by
decompiling v79 directly and finding the identical bucketed structure, not a
separate `case 16`.

So the correct test is not "does this version's client render unique text for
this byte" (v61/v72/v79/v48 all say no) but "is byte 16 the protocol's stable,
established wire value for the trade-limit condition across this version
family" — and it is, verified independently at v61/v72/v79, all three sharing
v48's exact bucketed-default mechanism. `template_gms_48_1.json` therefore
binds `TRADE_LIMIT: 16`, at the same address (`0x5b7ba5`, the shared default
call) as `GENERIC_ERROR`, matching v61/v72's precedent exactly (both bind
`TRADE_LIMIT` to their version's default-bucket address too).

This does **not** retroactively validate `OVER_LEVEL_REQUIREMENT` /
`UNDER_LEVEL_REQUIREMENT` by the same logic: the dispatcher yaml has no
`gms_v61`/`gms_v72`/`gms_v79` column for either name (both start at `gms_v83`),
and the v61/v72/v79 test-file comments explicitly say "no `Decode4()` anywhere
in the function, so OVER/UNDER_LEVEL_REQUIREMENT are version-absent" — the
same absence I found in v48. Unlike `TRADE_LIMIT`, there is no verified
precedent in any earlier version for these two names sharing the default
bucket; the *evidence itself* draws the line between "shares the default
bucket, still a named op" (`TRADE_LIMIT`) and "has no representation at all
pre-v83, name is absent" (`OVER_LEVEL_REQUIREMENT`/`UNDER_LEVEL_REQUIREMENT`).

## Names omitted from the v48 table

Two of the thirteen `NPCShopOperation` names present in `template_gms_83_1.json`
have **no v48 case and no earlier-version precedent**, so they are omitted
rather than guessed:

- `OVER_LEVEL_REQUIREMENT` (v83 case 15, `Decode4()`-driven dynamic level
  message) — v48 has no arm that calls a `Decode4()`-equivalent numeric read
  anywhere in this function (confirmed from the full 227-instruction
  disassembly: the only decode calls are `CInPacket::Decode1`, twice); the
  only dynamic-content arm in v48 (`v4==14`, `Decode1()!=0`) reads a *string*
  (`DecodeStr`), not a numeric level. v61/v72/v79 independently confirm this
  same absence for their versions.
- `UNDER_LEVEL_REQUIREMENT` (v83 case 14, same `Decode4()` mechanism) — same
  reasoning; v48's `v4==14` is not this arm (see mechanism match above).

## Result

```json
"operations": {
  "OK": 0,
  "OUT_OF_STOCK": 1,
  "NOT_ENOUGH_MONEY": 2,
  "INVENTORY_FULL": 3,
  "OUT_OF_STOCK_2": 5,
  "OUT_OF_STOCK_3": 9,
  "NOT_ENOUGH_MONEY_2": 10,
  "NEED_MORE_ITEMS": 13,
  "GENERIC_ERROR": 14,
  "GENERIC_ERROR_WITH_REASON": 14,
  "TRADE_LIMIT": 16
}
```

Applied to `services/atlas-configurations/seed-data/templates/template_gms_48_1.json`'s
`writers` array at `0xE6` (`NPCShopOperation`), alongside the `0xE5`
(`NPCShop`) writer entry, both inserted at their sorted `opCode` position
(between the existing `0xD5` and `0xED` entries). Key order in the JSON
follows ascending numeric value (`GENERIC_ERROR`/`GENERIC_ERROR_WITH_REASON`
at 14 before `TRADE_LIMIT` at 16), matching `template_gms_83_1.json`'s
convention.

## Verification status

Per design.md §7.1, these two cells (`0xE5`/`0xE6`) remain **⬜ unknown**, not
promoted to ✅, until a `/verify-packet` byte-fixture pass. This task only
registers the writers with an IDB-derived (not copied) operations table; it
does not move the coverage-matrix cells.
