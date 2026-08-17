# Is the Cash/0540 item-use arm the client's cancel entry point?

Read-only IDA derivation. Sessions used: gms_v83 `41f13e0d`
(`E:\...\GMS\v83_Me\MapleStory_dump.exe.i64`), gms_v95 `79906a1e`
(`E:\...\GMS\v95_0\GMS_v95.0_U_DEVM.exe.i64`).

## Verdict up front

**CONFIRMED: yes.** `CWvsContext::SendConsumeCashItemUseRequest` case 52/53
(v83) and case 53/54 (v95) — the name-change-coupon and
world-transfer-coupon arms of the cash item-use opcode — construct a
client-side "ask continue or cancel" dialog chain and, only if the player
confirms through both dialogs, append a single `Encode1` byte to the
*same* outgoing packet used for consuming any cash item. There is no
separately-named `SendCancel*` function; the cancel request rides on the
generic item-use send path. This directly matches the prior implementer's
finding.

## 1. What is the dialog class, and is it a cancel-request dialog?

Two distinct, unrelated C++ classes matter here — conflating them is the
trap:

### `CUICancelCharacterCouponResults` (v83 `@0x998c81`) — NOT called from the item-use arm

Full ctor decompile (v83):

```
void __thiscall CUICancelCharacterCouponResults::CUICancelCharacterCouponResults(CDialog ***this, CDialog **a2)
{
  ...
  if ( v3 ) {
    if ( v3 == 1 )      GetStringW(..., SP_4830_..._CANCELNAMECHANGE_FAIL);
    else if ( v3 == 2 ) GetStringW(..., SP_4834_..._CANCELCHARACTERTRANSFER_SUC);
    else if ( v3 == 3 ) GetStringW(..., SP_4835_..._CANCELCHARACTERTRANSFER_FAIL);
  } else                GetStringW(..., SP_4829_..._CANCELNAMECHANGE_SUC);
  ...
}
```

`xrefs_to(0x998c81)` on v83 returns **exactly two callers**, both result
receivers, never the item-use arm:

- `CWvsContext::OnCancelNameChangeResult` (`@0xa2a677`) — calls it at
  `0xa2a6b1` and `0xa2a723`
- `CWvsContext::OnCancelTransferWorldResult` (`@0xa2a82d`) — calls it at
  `0xa2a868` and `0xa2a8da`

This class is the **result display** dialog (success/fail message) shown
*after* the server replies with `CancelNameChangeResult` /
`CancelTransferWorldResult`. It is unrelated to sending anything. CONFIRMED.

### `CUICancelCharacterCouponRequests` (v95 `@0x96a2d0`) — the actual class built by the item-use arm

Note the naming: v83's decompiler could not recover a name for the
equivalent object (`sub_998874`, unnamed/local), but v95 has full RTTI
naming and demangles it explicitly as `CUICancelCharacterCouponRequests`
— "Requests", not "Results". This is a **different class** from the one
above; the prior implementer's name is correct for v95.

`xrefs_to(0x96a2d0)` on v95 returns **exactly four hits, all inside
`SendConsumeCashItemUseRequest` (`@0x9eb3e0`)**: `0x9ec2bc`, `0x9ec307`
(case 53, name change) and `0x9ec3a7`, `0x9ec3f2` (case 54, world
transfer). Not called from anywhere else in the binary. CONFIRMED.

v83's unnamed equivalent (`sub_998874` `@0x998874`) has the same
three-branch structure, keyed on a mode byte, over the **same string-pool
group name** (`CANCELREQUESTS`):

```
if ( !v3 )      GetStringW(..., SP_4823_..._CANCELREQUESTS_ASKCONTINUECANCEL_BACKGRND);
if ( v3 == 1 )  GetStringW(..., SP_4826_..._CANCELREQUESTS_CANCELNAMECHANGE_BACKGRND);
if ( v3 == 2 )  GetStringW(..., SP_4831_..._CANCELREQUESTS_CANCELCHARACTERTRANSFER_BACKGRND);
```

So on both versions the item-use arm builds a dialog whose message strings
live under the `CANCELREQUESTS_*` resource namespace — "ask continue or
cancel", "cancel name change", "cancel character transfer" — i.e. this is
the confirmation UI for **cancelling a pending coupon-driven change**, not
a generic "use item" confirmation. This matches point 1 of the question:
it is a cancellation-request dialog, not a use/list/confirm dialog for
consuming the coupon itself. CONFIRMED (string names + call-site
exclusivity), the object's exact runtime behavior beyond DoModal/string
selection was not further decompiled (no controls/buttons were traced) —
that residual detail is unverified but doesn't change the conclusion.

## 2. What opcode does the item-use arm send, and does that plausibly mean "cancel"?

v83, top of `SendConsumeCashItemUseRequest` (`@0xa0a63f`):

```
a0a6a8  push 4Fh ; int              ; opcode = 0x4F
a0a6ad  call ??0COutPacket@@QAE@J@Z ; COutPacket::COutPacket(0x4F)
a0a6bf  call ?Encode2@COutPacket@@QAEXG@Z   ; item slot/pos
a0a6cb  call ?Encode4@COutPacket@@QAEXK@Z   ; item unique id (nItemID)
a0a6d1  call ?get_consume_cash_item_type@@YAJJ@Z   ; dispatch key
...     switch(get_consume_cash_item_type(itemId)) { case 52: ...; case 53: ...; ... }
```

Opcode `0x4F` (79) is **the single generic "consume cash item" client→server
op for every cash-item type** — the switch on `get_consume_cash_item_type`
happens *inside* the same function, after the opcode/slot/id have already
been encoded, and just picks which extra payload bytes to append per item
category (pet name tag, megaphone, name-change coupon, world-transfer
coupon, etc.). CONFIRMED: there is no distinct "cancel" opcode; case
52/53 (v83) and 53/54 (v95) share the opcode with every other cash-item
consumption.

Case 52 (v83, name change) full control flow, decompiled from the raw
disassembly at `0xa0b1b4`–`0xa0b294`:

```
a0b1b4  loc_A0B1B4:                       ; jumptable case 52
a0b1c2    call Alloc(...)                 ; allocate dialog 1
a0b1d5    call sub_998874(this, edi)      ; construct CANCELREQUESTS dialog #1
a0b1fc    call sub_A154EA(...)            ; (generic ZRef/vtable helper)
a0b208    call DoModal                    ; show dialog #1, get result in eax
a0b20f    jnz  loc_A0B282                 ; if result != edi -> skip dialog 2, no Encode1
a0b217    call Alloc(...)                 ; allocate dialog 2
a0b22a    call sub_998874(this, ebx)      ; construct CANCELREQUESTS dialog #2
a0b251    call sub_A154EA(...)
a0b25d    call DoModal                    ; show dialog #2
a0b264    jnz  loc_A0B271                 ; if result != edi -> no Encode1
a0b26a    call Encode1(edi)               ; ***APPEND CANCEL FLAG BYTE TO THE PACKET***
```

Case 53 (v83, world transfer, `0xa0b294`–`0xa0b386`) is **structurally
identical**: `Alloc` → `sub_998874(this, 2)` (note the literal `push 2`
selecting the `CANCELCHARACTERTRANSFER_BACKGRND` string) → `DoModal` →
`Alloc` → `sub_998874` → `DoModal` → conditional `Encode1` at `0xa0b34b`.

**Correction to evidence given in the task**: the prior claim "v83 case 53
has ZERO `Encode*` calls in range" does not hold under this direct
disassembly — case 53 (`0xa0b294`–`0xa0b386`) contains an `Encode1` call at
`0xa0b34b`, gated behind the same double-`DoModal` confirmation as case
52. It is possible the earlier note used different case-number boundaries
or a different address window; regardless, what is decompiled here for
`0xa0b294`–`0xa0b386` is what IDA reports for jump-table case 53 on this
session, and it does contain a conditional `Encode1`.

v95 case 53 (name change, `0x9ec299`–`0x9ec384`) is even more explicit,
with full symbol names:

```
9ec2bc  call CUICancelCharacterCouponRequests::CUICancelCharacterCouponRequests(this, nType=1)
9ec2dd  call DoModal                 ; eax = result
9ec2e2  cmp eax, 1
9ec2e5  jnz loc_9EC358               ; not confirmed -> skip 2nd dialog, no Encode1
9ec307  call CUICancelCharacterCouponRequests::CUICancelCharacterCouponRequests(this, nType=0)
9ec328  call DoModal
9ec32d  cmp eax, 1
9ec330  jnz loc_9EC33E               ; not confirmed -> esi=1 (abort flag), no Encode1
9ec337  call Encode1(1)              ; ***APPEND CANCEL-CONFIRM BYTE = 1***
```

v95 case 54 (world transfer, `0x9ec384`–`0x9ec446`) mirrors this exactly,
with `nType=2` on the first dialog (`CANCELCHARACTERTRANSFER` string
group) instead of `nType=1`.

So on both versions: **the byte encoded is not the generic
"you dismissed dialog A vs dialog B" flag described in the prior finding's
paraphrase** — it's a single boolean confirmation flag (v95: literal `1`;
v83: whatever `edi`/`ebx` hold, most likely `0` given they're zeroed at
function entry, though that register's value was not traced end-to-end)
sent **only when the player clicks through both confirmation dialogs**,
and **omitted entirely (no `Encode1` call at all) if the player dismisses
either dialog**. Either way, the packet carries a real signal distinguishing
"confirmed" from "not confirmed," gated by the CANCELREQUESTS dialog chain.

## 3. Any other client send path for cancellation?

`xrefs_to` the v95 `CUICancelCharacterCouponRequests` constructor returns
exactly the 4 call sites already covered (all inside
`SendConsumeCashItemUseRequest`), and `xrefs_to` the v83
`CUICancelCharacterCouponResults` (result-display) constructor returns
exactly the 2 `OnCancel*Result` receivers. No other function in either
binary constructs either dialog class. Search for the demangled name
`CUICancelCharacterCouponRequests` in the v83 IDB (`func_query
name_regex`) returns **zero hits** — v83's copy of this class exists but
without a recovered/exported symbol (compiled as `sub_998874`); it is not
a separate, differently-named class hiding elsewhere. **CONFIRMED** there
is no other send path: the only place the client builds this
cancel-confirmation UI and conditionally emits the flag byte is inside
`SendConsumeCashItemUseRequest`'s name-change and world-transfer arms.

## 4. Does the client have a cancel request path, and what carries it?

**Yes, CONFIRMED.** The client's cancel-a-pending-coupon-change request is
not a distinct packet/opcode and not a distinct named `Send` function. It
is a conditional tail appended to the ordinary "consume cash item" request
(op `0x4F` on v83; same generic item-use op on v95 — not independently
re-verified numerically on v95 in this pass, but the encode sequence
(`Encode2`/`Encode4` slot+id, then per-type dispatch) is structurally the
same function). When the player uses a name-change or world-transfer
coupon item that has a pending change outstanding, the client shows the
`CANCELREQUESTS_*` dialog chain (ask-continue-or-cancel → type-specific
confirm), and only on double confirmation appends one `Encode1` byte to
that same outgoing item-use packet, which the server presumably
interprets (given no other opcode carries this signal) as the cancel
request.

## Impact on design.md §4.2.1 / plan.md Task 28

**Design §4.2.1's literal claim — "no `SendCancel*` of any kind on any
version" — is technically true** (no function of that name exists), but
**the substantive claim it was used to support — "the cancel path is
operator-only REST" — does not hold.** The client has a genuine cancel
request path; it's just multiplexed onto the generic cash item-use opcode
via a type-specific dialog-confirmed tail byte, not a separate opcode or
named send function. §4.2.1 needs revision to describe this correctly
rather than asserting no client-initiated cancel exists at all. Task 28's
`cancel-path-guard.sh`, if it enforces "no client send op maps to a
CANCEL_* receiver," would be machine-enforcing a **false** property — it
should instead account for (or explicitly except) the cash item-use op's
conditional cancel-flag tail.

**One-line verdict: design.md §4.2.1 does NOT stand as written and needs
revision — the client has a real cancel-request path, riding on the cash
item-use opcode rather than a dedicated `SendCancel*` function.**
