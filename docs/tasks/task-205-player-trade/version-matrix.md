# task-205 — per-version trade derivation table

Every value below was read from the version's own IDB this session (ten live
ida-pro sessions, resolved by binary name). Nothing is inferred from the v83
column. Where a value could not be established it says UNRESOLVED — it is never
guessed.

Design §10.1 procedure, one row per version:

1. locate the trade dialog's mode dispatcher (the virtual `CMiniRoomBaseDlg::OnPacketBase`
   forwards every mode it does not handle itself into the concrete sub-dialog),
2. enumerate the FULL switch — arm absence is asserted only from a decompiled
   dispatcher that lacks the case,
3. read the enter-result tail virtual and `OnLeave` off the class vtable,
4. look for a second trading-room class with its own dispatcher.

Task 23 (templates) consumes §1, §3 and §4 verbatim.

---

## 1. Clientbound mode bytes (`CTradingRoomDlg::OnPacket` switch)

| version | TRADE_PUT_ITEM | TRADE_ADD_MESO | TRADE_CONFIRM | TRADE_MESO_LIMIT | dispatcher |
|---|---|---|---|---|---|
| gms_v48  | 13 `@0x5e6ace` | 14 `@0x5e6ba5` | 15 `@0x5e6bd3` | 18 `@0x5e6be7` | `@0x5e6a84` |
| gms_v61  | 14 `@0x68b37f` | 15 `@0x68b456` | 16 `@0x68b484` | 19 `@0x68b498` | `@0x68b335` |
| gms_v72  | 14 `@0x6fdce7` | 15 `@0x6fddbe` | 16 `@0x6fddec` | 19 `@0x6fde00` | `@0x6fdc9d` |
| gms_v79  | 15 `@0x7357bf` | 16 `@0x735896` | 17 `@0x7358c4` | 20 `@0x7358d8` | `@0x735775` |
| gms_v83  | 15 `@0x7c1fb7` | 16 `@0x7c208e` | 17 `@0x7c20bc` | 21 `@0x7c21bd` | `@0x7c1f6d` |
| gms_v84  | 15 `@0x7e80fd` | 16 `@0x7e81d4` | 17 `@0x7e8202` | 21 `@0x7e8303` | `@0x7e80b3` |
| gms_v87  | 15 `@0x81566e` | 16 `@0x815745` | 17 `@0x815773` | 21 `@0x815877` | `@0x815624` |
| gms_v92  | 15 `@0x743f80` | 16 `@0x743e70` | 17 `@0x744440` | 21 `@0x744680` | `@0x744ec0` |
| gms_v95  | 15 `@0x763a60` | 16 `@0x763950` | 17 `@0x763f20` | 21 `@0x764160` | `@0x7649a0` |
| jms_v185 | 13 `@0x845dd0` | 14 `@0x845ea7` | 15 `@0x845ed5` | **n-a** — dispatcher `@0x845d95` has exactly three cases (13/14/15) | `@0x845d95` |

The mode-21 arm carries a different IDB symbol per build — `OnMesoLimitRefused`
(v48/v61/v72/v83/v84/v92 after this session's renames) vs `OnExceedLimit` (the
v79/v87/v95 PDB spelling) — same arm, same shape. Its StringPool notice id is
per-version and NOT stable: v48 3626 · v61 3901 · v72 3953 · v79 3956 ·
v83 SP_3977 · v84 3980 · v87 3985 · v92 4051 · v95 4018.

**Body layouts are byte-identical on every version that has the arm:**

- `TRADE_PUT_ITEM` — `Decode1` side, `Decode1` tradeSlot, `GW_ItemSlotBase::Decode`.
- `TRADE_ADD_MESO` — `Decode1` side, `Decode4` amount. The amount is **ASSIGNED**
  (`this[side + N] = Decode4()`) on all ten versions, never accumulated, so a
  re-echo of the last valid amount is an authoritative correction (design §4.2).
- `TRADE_CONFIRM` — bodyless on all ten (no `Decode*` at all).
- `TRADE_MESO_LIMIT` — bodyless everywhere it exists.

No version diverges in field order or width, so
`libs/atlas-packet/interaction/clientbound/version_gate.go` is **not created**
(design §10.1 step 2 / plan Step 4: an always-false gate would be dead code).
The only per-version variable is the mode byte, which is config-resolved through
`atlas_packet.WithResolvedCode("operations", KEY, …)` (DOM-25) and belongs in the
tenant templates, not in Go.

## 2. `TRADE_CONFIRM` auto-reply — resolves PRD open question 3 (design §10.2)

Design §10.2 predicted `TRANSACTION` is absent below v83 because there is no CRC
to attest, with jms_v185 named as the exception to check directly. **Both halves
of the prediction are confirmed.**

| version | does the confirm arm send a serverbound reply? | evidence |
|---|---|---|
| gms_v48  | **no** | `@0x5e6bd3` body is `this[108]=1; InvalidateRect(this,0);` — no `COutPacket` |
| gms_v61  | **no** | `@0x68b484` body is `*((_DWORD*)this+108)=1; InvalidateRect(this,0);` |
| gms_v72  | **no** | `@0x6fddec` is four instructions: `push 0; mov [ecx+1B8h],1; call InvalidateRect; retn 4` |
| gms_v79  | **no** | `@0x7358c4` body is `*((_DWORD*)this+110)=1; InvalidateRect(this,0);` (20 bytes) |
| gms_v83  | **yes** | `@0x7c20bc` → `COutPacket(123)`, `Encode1(0x14)`, `Encode1(count)`, `{Encode4 itemId, Encode4 crc}`… |
| gms_v84  | **yes** | `@0x7e8202` → `COutPacket(125)`, `Encode1(0x14)`, same list |
| gms_v87  | **yes** | `@0x815773` → `COutPacket(0x81)`, `Encode1(0x14)`, same list |
| gms_v92  | **yes** | `@0x744440` → `COutPacket(0x8D)`, mode byte `20` (0x14), same list |
| gms_v95  | **yes** | `@0x763f20` → `COutPacket(144)`, mode byte `20` (0x14), same list |
| jms_v185 | **yes** | `@0x845ed5` → `COutPacket(0x7C)`, `Encode1(0x12)`, same list |

So the boundary sits exactly where `tradeCrcPresent`
(`libs/atlas-packet/interaction/serverbound/version_gate.go`) already puts it,
and **jms_v185 DOES have `TRANSACTION`** — at mode **0x12 (18)**, not 0x14, the
same −2 shift the jms trade block carries elsewhere.

> **Open item for Task 23.** `docs/packets/dispatchers/character_interaction_handle.yaml`
> currently records `TRANSACTION` with `modes: { gms_v83: 20, gms_v84: 20, gms_v87: 20, gms_v95: 20 }`
> and its header comment claims TRANSACTION is "server-driven, no client send" and
> therefore n-a on jms. Two corrections are due there: **gms_v92: 20** (the arm
> exists, `@0x744440`) and **jms_v185: 18** (`@0x845ed5`). Both are outside this
> task's committed diff because that yaml is the templates' source of truth and
> Task 23 owns it.

### fname correction (design §11.1) — landed here

`CCashTradingRoomDlg::Trade` is NOT the `TRANSACTION` sender. It is the cash
room's Trade BUTTON handler and encodes `Encode1(0x11)` — `TRADE_CONFIRM` —
exactly like `CTradingRoomDlg::Trade`:

| version | `CCashTradingRoomDlg::Trade` | mode encoded |
|---|---|---|
| gms_v83  | `@0x485dcd` | `Encode1(0x11)` + count + `{itemId, crc}` list |
| gms_v95  | `@0x49e180` | `Encode1(0x11)` (`*(v31 + v66++) = 17;` @0x49e63f) |
| gms_v79  | `@0x47e5f5` | `Encode1(0x11)`, no entry list |
| jms_v185 | `@0x499b67` | `Encode1(0x0F)` — the jms cash-room confirm |

`interaction/serverbound/InteractionOperationTransaction` is therefore re-keyed
to `CTradingRoomDlg::OnTrade` in `tools/packet-audit/cmd/run.go`, each version's
export gained a surgical `CTradingRoomDlg::OnTrade` entry, and the six versions
that have the sender were re-reported (all now grade `Verdict 0`, up from
`Verdict 3 / FlatInvalid`) and re-pinned. The four legacy versions are recorded
`n-a` in their `_unimplemented.json` with the decompile evidence above.

## 3. Enter-result tail virtual and `OnLeave`

The enter-result tail virtual is EMPTY on all ten versions, so design §1.3 holds
everywhere and `NewTradeRoom` needs no version gate: the trade room's
ENTER_RESULT body is exactly the `CMiniRoomBaseDlg::OnEnterResultBase` frame.

| version | class vtable | tail virtual | `OnLeave` | leave status bytes |
|---|---|---|---|---|
| gms_v48  | `off_7A1270` | `+64` → `@0x5e6ac4` (InvalidateRect only) | `+72` → `@0x5e6c45` | 2 · 6 · 7 · 8 · 9 |
| gms_v61  | `off_8EBA30` | `+72` → `@0x5bea98` `nullsub_95` | `+76` → `@0x68b4f6` | 2 · 6 · 7 · 8 · 9 |
| gms_v72  | `off_9D60B8` | `+68` → `@0x6fdcdd` (InvalidateRect only) | `+72` → `@0x6fde60` | 2 · 6 · 7 · 8 · 9 |
| gms_v79  | `off_A32FB8` | `+72` → `@0x47baac` `nullsub_58` | `+76` → `@0x735938` | 2 · 7 · 8 · 9 · 12 |
| gms_v83  | `off_B37448` | `+72` → `@0x48314d` `nullsub_94` | `+76` → `@0x7c221d` | 2 · 7 · 8 · 9 · 12 · 13 |
| gms_v84  | `0xb8a940`   | `+72` → `@0x4864fa` `nullsub_68` | `+76` → `@0x7e8363` | 2 · 7 · 8 · 9 · 12 · 13 |
| gms_v87  | `off_BDE810` | `+72` → `@0x491faa` `nullsub_104` | `+76` → `@0x8158d7` | 2 · 7 · 8 · 9 · 12 · 13 |
| gms_v92  | `0xb75e88` (OnPacket slot) | `@0x49bf00` `CMiniRoomBaseDlg::OnEnterResult` (empty) | `@0x744f30` | 2 · 7 · 8 · 9 · 12 · 13 |
| gms_v95  | `0xb98df8` | `+72` → `@0x49fa60` `CMiniRoomBaseDlg::OnEnterResult` (empty) | `+76` → `@0x764a10` | 2 · 7 · 8 · 9 · 12 · 13 |
| jms_v185 | `off_BECB00` | `+72` → `@0x496d41` `nullsub_91` | `+76` → `@0x845fd6` | 2 · 7 · 8 · 9 · 12 · 13 |

Absolute vtable slot offsets shift across builds (v48 puts the dispatcher at +60/+64
and pushes `OnLeave` to +72; v72 puts the empty tail at +68). The *relative*
shape — dispatcher, enter-tail, `OnLeave` in consecutive slots — is stable, and
the tail is empty in every one.

### `leaveReason` per version — the table Task 23 writes into the templates

The v83 six-status set is NOT universal. **gms_v48/v61/v72 shift success to 6 and
have no different-map or CRC status at all**; gms_v79 has no CRC status.

| key (`interaction_body.go`) | v48 | v61 | v72 | v79 | v83 | v84 | v87 | v92 | v95 | jms |
|---|---|---|---|---|---|---|---|---|---|---|
| `TRADE_CANCELLED`     | 2 | 2 | 2 | 2 | 2 | 2 | 2 | 2 | 2 | 2 |
| `TRADE_SUCCESS`       | 6 | 6 | 6 | 7 | 7 | 7 | 7 | 7 | 7 | 7 |
| `TRADE_FAILED`        | 7 | 7 | 7 | 8 | 8 | 8 | 8 | 8 | 8 | 8 |
| `TRADE_CANNOT_CARRY`  | 8 | 8 | 8 | 9 | 9 | 9 | 9 | 9 | 9 | 9 |
| `TRADE_DIFFERENT_MAP` | 9 | 9 | 9 | 12 | 12 | 12 | 12 | 12 | 12 | 12 |
| `TRADE_CRC_FAILED`    | **n-a** | **n-a** | **n-a** | **n-a** | 13 | 13 | 13 | 13 | 13 | 13 |

Read the v48/v61/v72 column carefully: those clients have a FIVE-status set
{2,6,7,8,9}. The fifth value (9) is the "some items you cannot carry" /
"different map" tail of the chain, so `TRADE_CANNOT_CARRY` and
`TRADE_DIFFERENT_MAP` collapse onto 8/9 there rather than 9/12. Per-version
StringPool ids for the six/five arms:

- v48 390 · 391/392 · 393 · 394 · 395
- v61 420 · 421/422 · 423 · 424 · 425
- v72 408 · 409/410 · 411 · 412 · 413
- v79 408 · 409/410 · 411 · 412 · 413
- v83 SP_406 · SP_407/SP_408 · SP_409 · SP_410 · SP_411 · SP_5566
- v84 409 · 410/411 · 412 · 413 · 414 · 5729
- v87 416 · 417/418 · 419 · 420 · 421 · 5881
- v92 421 · 422/423 · 424 · 425 · 426 · 6317
- v95 421 · 422/423 · 424 · 425 · 426 · 6759
- jms 441 · 442/443 · 444 · 445 · 448 · 5903

The paired ids on the success row are the "no fee" / "received %d mesos after
fees" variants; the client picks between them from its OWN `CharacterData` meso
delta, not from the packet — which is why the server must apply the meso award
BEFORE sending `LEAVE success` (design §6.4).

## 4. Cash trading room — resolves PRD open question 4 (design §10.3)

| version | second trading-room class? | dispatcher | cases | evidence when absent |
|---|---|---|---|---|
| gms_v48  | **NO**  | — | — | mini-room factory `sub_5458CC @0x5458cc` enumerates exactly 5 room types (Omok/MemoryGame/TradingRoom/PersonalShop/EntrustedShop), nil default |
| gms_v61  | **NO**  | — | — | room-type factory `sub_5BEB71 @0x5beb71`, 5 cases, no 6th |
| gms_v72  | **NO**  | — | — | `CMiniRoomBaseDlg::MiniRoomFactory sub_60E13D @0x60e13d`, 5 cases, `default -> return 0` |
| gms_v79  | **yes** | `@0x47bd13` | 15 / 16 / 17 | — |
| gms_v83  | **yes** | `@0x4833b4` | 15 / 16 / 17 | — |
| gms_v84  | **yes** | `@0x486761` | 15 / 16 / 17 | — |
| gms_v87  | **yes** | `@0x492214` | 15 / 16 / 17 | — |
| gms_v92  | **yes** | `@0x499610` | 15 / 16 / 17 | — |
| gms_v95  | **yes** | `@0x49d6b0` | 15 / 16 / 17 | — |
| jms_v185 | **yes** | `@0x496fa8` | 13 / 14 / 15 | — |

Two consequences:

- **No cash room has a meso-limit arm** on ANY version — every cash dispatcher
  above has exactly three cases. Design §4.2's "where the arm is absent the
  rejection degrades to the authoritative `TRADE_ADD_MESO` re-echo" is the
  cash-room path on all ten versions, and the whole-version path on jms_v185.
- **`CCashTradingRoomDlg` DOES exist on jms_v185.** This refutes design §10.3's
  reading of the seed templates (`CASH_TRADE_OPEN` absent from the jms template)
  as evidence of version-absence: the class, its dispatcher `@0x496fa8`, its
  three arms and its own `Trade` sender `@0x499b67` (`Encode1(0x0F)`) are all
  present. The template gap is a WIRING gap, not a version gap — Task 23 should
  add `CASH_TRADE_OPEN` to `template_jms_185_1.json`, and must NOT add it to
  gms_v48/v61/v72 where the class genuinely does not exist.

The v79/v92 cash rooms also carry two `OnLeave` statuses the plain room lacks —
v79 `10 → SP 5239`, `11 → SP 5238`; v92 `10 → 5359`, `11 → 5358` — cash-specific
failure reasons. They are out of this task's `leaveReason` key set.

## 5. What this session renamed in the IDBs

Naming symbols while reversing, per CLAUDE.md. All renames are IDB-local.

| version | renamed |
|---|---|
| gms_v48  | `0x5e6a84` OnPacket · `0x5e6ace` OnPutItem · `0x5e6ba5` OnPutMoney · `0x5e6bd3` OnTrade · `0x5e6be7` OnMesoLimitRefused · `0x5e6c45` OnLeave |
| gms_v61  | `0x68b055` ctor · `0x68b335` OnPacket · `0x68b37f` OnPutItem · `0x68b456` OnPutMoney · `0x68b484` OnTrade · `0x68b498` OnMesoLimitRefused · `0x68b4f6` OnLeave |
| gms_v72  | `0x6fddec` OnTrade (rest already symbol-named) |
| gms_v79  | `0x735938` `CTradingRoomDlg::OnLeave` · `0x47be67` `CCashTradingRoomDlg::OnLeave` |
| gms_v84  | `0x7e80b3` OnPacket · `0x7e80fd` OnPutItem · `0x7e81d4` OnPutMoney · `0x7e8202` OnTrade · `0x7e8303` OnMesoLimitRefused · `0x7e8363` OnLeave · `0x486761` `CCashTradingRoomDlg::OnPacket` |
| gms_v87  | `0x492214` `CCashTradingRoomDlg::OnPacket` · `0x8158d7` `CTradingRoomDlg::OnLeave` |
| gms_v92  | `0x7446f0` PutItem · `0x744970` PutMoney · `0x744bd0` Trade · `0x499170` cash PutItem · `0x4993e0` cash PutMoney |
| jms_v185 | `0x845d95` OnPacket · `0x845dd0` OnPutItem · `0x845ea7` OnPutMoney · `0x845fd6` OnLeave |

gms_v83 and gms_v95 were already fully symbol-named; nothing was renamed there.

### One UNRESOLVED

`CCashTradingRoomDlg::Trade` on **gms_v92** could not be located. Every unnamed
function in the class's contiguous code block (`0x498520`–`0x4998c1`) and the
block after `OnLeave` was decompiled; none builds a `COutPacket(0x8D)` with
`Encode1(0x11)`. Unlike the plain room — where `PutItem`/`PutMoney`/`Trade` sit
as three consecutive functions before `OnPacket` — the cash room has only two
sends adjacent to its dispatcher. This is recorded as unresolved, **not** as
absence: no whole-`.text` scan for the byte signature was run. It does not block
anything in this task (the v92 cash room's three receive arms are all located)
and it is not a `TRANSACTION` sender either way, since v92's `TRANSACTION` is
`CTradingRoomDlg::OnTrade @0x744440`.
