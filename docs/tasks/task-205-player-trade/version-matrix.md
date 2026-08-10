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

> **Closed by Task 23.** Both corrections are applied; see the amendments section
> at the end of this file for what Task 23 changed and why one recommendation in
> §4 was overturned on fresh evidence.
>
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

---

## 6. Task 23 amendments (templates pass)

Task 23 re-derived every byte it wrote rather than copying this file, and three
things changed. All addresses below were decompiled in the version's own IDB
during the Task 23 pass.

### 6.1 gms_v48's SERVERBOUND trade block was wrong by +1 — corrected

`character_interaction_handle.yaml` and `template_gms_48_1.json` carried
`TRADE_PUT_ITEM=14 / TRADE_ADD_MESO=15 / TRADE_CONFIRM=16` for gms_v48. Those
values were never IDA-derived — task-127 §1 lists the v48 seven-mode table as
pre-existing template content. The v48 send sites say 13/14/15:

| sender | evidence |
|---|---|
| `CTradingRoomDlg::PutItem` @0x5E7F74 | `COutPacket(93)` · `Encode1(0xD)` · slot · 2×`Encode2` · count |
| `CTradingRoomDlg::PutMoney` @0x5E819A | `COutPacket(93)` · `Encode1(0xE)` · `Encode4(amount)`; 1,000,000 gate + SP 3626 |
| `CTradingRoomDlg::Trade` @0x5E836C | `COutPacket(93)` · `Encode1(0xF)`; SP 397 YesNo |
| `CTradingRoomDlg::SetRet` @0x5E69EA | `COutPacket(93)` · `Encode1(0xA)` — EXIT=10 |

which agrees with v48's own receive switch @0x5e6a84 (cases 13/14/15/18, §1). A
client sending mode 13 into a server routing 14 = a silently mis-dispatched
put-item on every v48 trade. Fixed in both files.

Also observed and **not** changed (non-trade, out of this task's scope):
`EXIT=10` exists on gms_v48 (`@0x5E69EA`) but is absent from the v48 handler
table and from the yaml's EXIT row.

### 6.2 `CASH_TRADE_OPEN` is n-a on jms_v185 — §4's recommendation overturned

§4 concluded the jms template's missing `CASH_TRADE_OPEN` was a wiring gap
because `CCashTradingRoomDlg` exists on jms. That inference does not hold: the
`CASH_TRADE_OPEN` handler key is the **mode-14 SSN2 request**
(`interaction/serverbound.OperationCashTradeOpen`), and jms has no such request.

- jms `CMiniRoomBaseDlg::OnPacketBase` @0x6da198 dispatches 2/3/5 (no room) and
  3/4/6/9/10/default (in room). There is no mode-14 arm; gms_v83..v95 and gms_v92
  send mode 14 into `OnCheckSSN2Static`.
- The jms IDB has no `OnCheckSSN2Static` function at all.
- jms opens the cash trading room through `CField::SendInviteTradingRoomMsg`
  @0x56c859: `COutPacket(0x7C)` `Encode1(0)` **CREATE**, `Encode1(6)` roomType 6,
  `Encode4(0)`, `Encode4(targetId)`, then `Encode1(2)` **INVITE**. That is exactly
  the fallback path gms_v92's `OnCheckSSN2Static` @0x62b990 takes for `nProc==0`.

So the class exists, its serverbound OPEN request does not. `CASH_TRADE_OPEN`
stays omitted for jms_v185, and the yaml's original "folds into CREATE/VISIT"
note is correct. jms `SendInviteResult` @0x6da8e6 and `SendCashInviteResult`
@0x6da99e are byte-identical (decline 3 / accept 4) — the cash room reuses the
plain room's invite modes.

### 6.3 gms_v92's serverbound trade block — fully derived

| key | mode | evidence (`COutPacket(0x8D)` throughout) |
|---|---|---|
| CREATE | 0 | `CField::SendInviteTradingRoomMsg` @0x528eb0 `Encode1(0)`+roomType 3 |
| INVITE | 2 | same function, second packet `Encode1(2)`+`Encode4(targetId)` |
| INVITE_DECLINE | 3 | `CMiniRoomBaseDlg::SendInviteResult` @0x62b740, `nErrCode != 0` arm |
| VISIT | 4 | same function, `nErrCode == 0` arm |
| CASH_TRADE_OPEN | 14 | `CMiniRoomBaseDlg::OnCheckSSN2Static` @0x62b990 `Encode1(14)` |
| TRADE_PUT_ITEM | 15 | `CTradingRoomDlg::PutItem` @0x7446f0 `Encode1(15)` |
| TRADE_ADD_MESO | 16 | `CTradingRoomDlg::PutMoney` @0x744970 `Encode1(16)`; 1,000,000 gate + SP 4051 |
| TRADE_CONFIRM | 17 | `CTradingRoomDlg::Trade` @0x744bd0, inline `= 17` + {itemId, crc} list |
| TRANSACTION | 20 | `CTradingRoomDlg::OnTrade` @0x744440, inline `= 20` + {itemId, crc} list |

These are routed in `template_gms_92_1.json` but are **not** added to the two
dispatcher yamls: `packet-audit operations --check` treats a version column as
all-or-nothing (one gms_v92 mode turns every other v92 template key into an
`EXTRA`), and the rest of the v92 tables are the hand-set, non-IDA-verified set
from task-133. Both yaml headers record this explicitly.

### 6.4 `inviteResult` — derived on all ten, uniform

`CMiniRoomBaseDlg::OnInviteResultStatic` decodes one byte and branches
1/2/3/4 on every version; arm 1 reads **no** name, arms 2–4 `DecodeStr` one.
`CANNOT_FIND_CHARACTER = 1` and `BUSY = 2` everywhere:

| version | function | arm-1 SP | arm-2 SP |
|---|---|---|---|
| gms_v48  | @0x54607d (via `OnPacketBase` @0x5459c4) | 355 | 387 |
| gms_v61  | @0x5bf352 (via `OnPacketBase` @0x5bec69 case 3) | 382 | 417 |
| gms_v72  | @0x60e952 | 368 | 405 |
| gms_v79  | @0x62d5dd | 368 | 405 |
| gms_v83  | @0x65e848 | SP_366 | SP_403 |
| gms_v84  | @0x6746b1 (via `OnPacketBase` @0x673db5) | 369 | 406 |
| gms_v87  | @0x698b61 | 376 | 413 |
| gms_v92  | @0x62c020 | 379 | 418 |
| gms_v95  | @0x637d70 | 0x17B (379) | 0x1A2 (418) |
| jms_v185 | @0x6daa56 | 0x194 (404) | 0x1B5 (437) |

Naming note: in the gms_v72 and gms_v79 IDBs the symbols `OnInviteResultStatic`
and `OnLeaveBase` sit on functions whose bodies are the opposite shape (the
address above is the invite-result body in both). The names were left alone —
only one half of each pair was verified this pass.

### 6.5 jms `enterError`

The jms template had no `enterError` table at all. Its enter-error switch is
`CMiniRoomBaseDlg::OnEnterResultStatic` @0x6da234, and it carries the same
present/absent code set as gms (1–7, 9–22, 24, plus a jms-only 25) with the same
tell: **case 7 and case 20 resolve the SAME StringPool id (0x1B2)**, exactly the
gms `TRADE_NOT_ALLOWED` / `TRADE_NOT_ALLOWED_2` pairing. Only those two keys were
written; the remaining arms are positionally suggestive but their strings were not
read, so they are left underived rather than mapped by position.

### 6.6 `INVITE_DECLINE` exists on gms_v48/v61/v72 — added (Task 23 fix round)

`character_interaction_handle.yaml` listed `INVITE_DECLINE` from gms_v79 up, and
the three legacy templates omitted it. That was wrong: the sender exists on all
three, and it is the same function whose accept arm supplies the already-routed
`VISIT = 4`.

| version | `CMiniRoomBaseDlg::SendInviteResult` | decline arm | accept arm | handler opCode |
|---|---|---|---|---|
| gms_v48 | `@0x545FCD` | `COutPacket(93)` · `Encode1(3u)` · `Encode4(dwSN)` · `Encode1(nErrCode)` | `Encode1(4u)` | 0x5D = 93 ✓ |
| gms_v61 | `@0x5BF2A2` | `COutPacket(111)` · `Encode1(3u)` · `Encode4(dwSN)` · `Encode1(nErrCode)` | `Encode1(4u)` | 0x6F = 111 ✓ |
| gms_v72 | `@0x60E8A4` | `COutPacket(121)` · `Encode1(3u)` · `Encode4(dwSN)` · `Encode1(nErrCode)` | `Encode1(4u)` | 0x79 = 121 ✓ |

Each `COutPacket` opcode equals that template's own `CharacterInteractionHandle`
`opCode`, so these are unambiguously the same route. Without the key,
`isCharacterInteraction`
(`services/atlas-channel/atlas.com/channel/socket/handler/character_interaction.go`)
logs "Code [INVITE_DECLINE] not configured for use" and drops the packet — the
inviter is never notified and the trade invite dangles. `INVITE_DECLINE: 3` is
now routed on all three.

### 6.7 Known `enterError` omissions — recorded, NOT guessed

`enterError` is not covered by any dispatcher yaml or by
`packet-audit operations --check`, so these gaps are invisible to CI. Recording
them here so they are a known omission rather than a silent one. Every key
listed as absent resolves to **99** through `atlas_packet.ResolveCode`.

| template | enterError keys | note |
|---|---|---|
| gms_v48 | 18 | lacks `TRADE_NOT_ALLOWED_2`, `NOT_ENOUGH_MESOS`, `INCORRECT_PASSWORD`, `ITEM_EXPIRED` — the four keys gms_v61+ carry at 20/21/22/24. A v48 `TRADE_NOT_ALLOWED_2` resolves to 99. **The byte was not guessed**: v48's enter-error switch was not decompiled this pass, and the v48 mini-room enum is demonstrably shifted from v61's (see §6.1), so copying 20 across would be exactly the class of error §6.1 fixed. |
| gms_v92 | 0 | the v92 writer has **no** `enterError` table at all — every enter-error key resolves to 99 there. Same reasoning: not populated by copying the v83 column. |
| jms_v185 | 2 | `TRADE_NOT_ALLOWED` 7 / `TRADE_NOT_ALLOWED_2` 20 only, both proved by the shared StringPool id 0x1B2 in `OnEnterResultStatic` @0x6da234. The other twenty arms exist but are **non-positional** — `0xA`→SP 0xE37, `0xB`→0x1D7, `0xC`→0x1D4, `0x12`→0xE30, `0x19`→0x105 break any positional mapping to the gms key order — so mapping them by position would have produced wrong bytes. Left underived pending a StringPool read. |

Closing any of these needs a decompile of that version's enter-error switch plus
a StringPool read, which is a `CMiniRoomBaseDlg` enter-result job, not a trade one.
