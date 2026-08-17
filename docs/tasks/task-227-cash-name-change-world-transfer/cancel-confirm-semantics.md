# Cancel-confirm semantics: dismiss-path send behavior and v83 confirm-byte value

Read-only IDA derivation, follow-up to `cancel-entry-point.md`. Sessions used:
gms_v83 `41f13e0d` (`E:\...\GMS\v83_Me\MapleStory_dump.exe.i64`), gms_v95
`79906a1e` (`E:\...\GMS\v95_0\GMS_v95.0_U_DEVM.exe.i64`).

## Answers up front

1. **CONFIRMED, both versions: dismissing either confirmation dialog sends NO
   packet at all.** The dismiss path jumps around the `SendPacket` call
   entirely and lands directly in cleanup/epilogue code. `Encode1` and
   `SendPacket` are both skipped together — there is no tail-less packet on
   the wire to worry about. A received packet of this opcode carrying the
   full name-change/world-transfer tail (any tail byte) IS necessarily the
   confirmed-cancel case; the tail-less form the design flagged as a risk
   does not exist on the wire.

2. **CONFIRMED: v83 encodes `Encode1(1)` on confirm — the same value as v95,
   not `0`.** `edi` is set to literal `1` once at function entry
   (`push 1 / pop edi` at `@0xa0a6b4`) and is never reassigned before either
   `Encode1(edi)` call site. Both versions send `1` for "confirmed cancel."
   No version divergence on this byte.

## Method note

Both functions share one epilogue (single `retn` per function: v83
`@0xa0eac3`→`retn`, v95 `@0x9eb6a7`→`retn`). Both the confirm path and the
dismiss path funnel into that same epilogue, which is why a superficial read
of "where do the branches end up" can look like they converge — the
distinguishing question is whether `SendPacket` sits *between* the branch
point and that shared epilogue on each path. It does only on the confirm
side, on both versions.

---

## v83 — case 52 (name change, `0xa0b1b4`–`0xa0b294`) and case 53 (world
transfer, `0xa0b294`–`0xa0b386`)

### Sentinel setup (function entry, applies to every switch arm)

```
a0a651  xor ebx, ebx                     ; ebx = 0, never reassigned again
...
a0a6ad  call COutPacket::COutPacket(0x4F)
a0a6b2  push 1
a0a6b4  pop edi                          ; edi = 1, never reassigned before
                                          ; either Encode1 call site below
```

`edi` is the MFC `IDOK` sentinel used by every `cmp eax, edi` / `jnz` after a
`DoModal` call in this function — confirmed by direct disassembly, not
inferred. It is not touched again anywhere between `0xa0a6b4` and the
`Encode1` calls at `0xa0b26a` / `0xa0b34b`.

### Case 52 dialog-chain trace (case 53 is structurally identical, see raw dump below)

```
a0b1c2  call Alloc(...)                  ; dialog 1
a0b1d5  call sub_998874(this, edi)       ; construct dialog 1 (edi=1 => nType path)
a0b1fc  call sub_A154EA(...)
a0b208  call DoModal                     ; eax = result
a0b20d  cmp eax, edi
a0b20f  jnz short loc_A0B282             ; DISMISS dialog 1 -> loc_A0B282
a0b211  call Alloc(...)                  ; dialog 2
a0b22a  call sub_998874(this, ebx)       ; construct dialog 2
a0b251  call sub_A154EA(...)
a0b25d  call DoModal                     ; eax = result
a0b262  cmp eax, edi
a0b264  jnz short loc_A0B271             ; DISMISS dialog 2 -> loc_A0B271
a0b266  push edi                         ; a2
a0b267  lea ecx, [ebp+var_38]            ; this = &oPacket
a0b26a  call Encode1(edi)                ; CONFIRM: Encode1(1)
a0b26f  jmp short loc_A0B274
loc_A0B271:
a0b271  mov [ebp+a2], edi                ; a2 (local "aborted" flag) = 1
loc_A0B282 (dialog-1-dismiss target):
a0b282  mov [ebp+a2], edi                ; a2 = 1
```

Both dismiss sub-paths, and the confirm path, converge at `loc_A0B285`
(shared with case 53's tail at `0xa0b366`):

```
a0b285/a0b366  cmp [ebp+a2], ebx         ; a2 == 0 ?
a0b369  mov byte ptr [ebp+var_4], 1
a0b36d  lea ecx, [ebp+var_1C]
a0b370  jz short loc_A0B37C             ; a2==0 (CONFIRM, Encode1 ran) -> A0B37C
a0b372  call sub_A15507                  ; a2!=0 (DISMISS) -> falls through
a0b377  jmp loc_A0B876                   ; DISMISS destination
a0b37c  call sub_A15507                  ; CONFIRM destination
a0b381  jmp loc_A0A8F1                   ; jumptable "cases 33,69" shared tail
```

`a2` is 0 only when neither dialog was dismissed (i.e. `Encode1` actually
ran); it is set to `edi`(1) the instant either dialog is dismissed, and
that dismissal branch never touches `a2` back to 0. So the `jz`/fallthrough
at `0xa0b370` is exactly "did we confirm."

### The two destinations, traced to completion

**`loc_A0A8F1`** (confirm destination — cases 33/69 shared tail):

```
a0a8f1  mov ecx, [ebp+var_54]
a0a8f5  push 1F4h
a0a8fa  call CanSendExclRequest
a0a8ff  test eax, eax
a0a901  jnz loc_A0EA53                    ; normal path
...
a0ea53  call get_update_time
a0ea5c  call Encode4                      ; append update-time
a0ea64  push eax
a0ea65  call SendPacket                   ; *** THE PACKET IS TRANSMITTED ***
a0ea6f  call SetExclRequestSent(edi)      ; edi=1
...
a0eac3  loc_A0EAC3:                       ; shared function epilogue
        pop edi / pop esi / restore fs:0 / pop ebx / leave / retn 10h
```

**`loc_A0B876`** (dismiss destination):

```
a0b876  lea ecx, [ebp+var_34]
a0b87c  call RemoveAll                    ; discard the COutPacket's array buffer
a0b881  mov eax, [ebp+arg_C]
a0b884  or [ebp+var_4], 0FFFFFFFFh
a0b888  cmp eax, ebx
a0b88a  jz loc_A0EAC3                     ; -> same shared epilogue, NO SendPacket call
a0b890  add eax, 0FFFFFFF4h
a0b893  push eax
a0b894  call _Release                     ; string cleanup
a0b89a  jmp loc_A0EAC3                    ; -> same shared epilogue, NO SendPacket call
```

**CONFIRMED (v83):** `SendPacket` (`?SendPacket@@YAXABVCOutPacket@@@Z`,
`@0xa0ea65`) is called exactly once in this function's dialog-gated
control flow, only on the branch reached from `loc_A0A8F1`, which is only
reached when `a2==0`, which only happens when both dialogs were confirmed
and `Encode1` ran. The dismiss branch (`loc_A0B876`) runs `RemoveAll` /
`_Release` (destroying the already-built `COutPacket`'s buffers) and jumps
straight to the shared epilogue — `SendPacket` is never reached. Case 53
(world transfer) shares this exact tail (`jmp loc_A0B876` at `0xa0b377`,
`jmp loc_A0A8F1` at `0xa0b381` — see raw dump above), so the same
conclusion holds for both arms verbatim, not by analogy.

Raw disasm queried directly from IDA (`insn_query`, ranges
`0xa0b1b4`-`0xa0b294` and `0xa0b294`-`0xa0b386`) backs every address cited
above; the `SendPacket`/`RemoveAll` tail was independently queried at
`0xa0a8e0`-`0xa0a940`, `0xa0ea40`-`0xa0eae0`, and `0xa0b860`-`0xa0b8b0`.

---

## v95 — case 53 (name change, `0x9ec299`–`0x9ec384`) and case 54 (world
transfer, `0x9ec384`–`0x9ec446`)

### Dialog-chain trace (case 53; case 54 is byte-for-byte structurally identical with `nType=2`)

```
9ec29e  xor esi, esi                      ; esi = 0 at function entry
...
9ec2bc  call CUICancelCharacterCouponRequests(nType=1)   ; dialog 1
9ec2dd  call DoModal
9ec2e2  cmp eax, 1
9ec2e5  jnz short loc_9EC358              ; DISMISS dialog 1 -> loc_9EC358 (esi still 0)
9ec307  call CUICancelCharacterCouponRequests(nType=0)   ; dialog 2
9ec328  call DoModal
9ec32d  cmp eax, 1
9ec330  jnz short loc_9EC33E              ; DISMISS dialog 2
9ec337  call Encode1(1)                   ; CONFIRM: Encode1(1)
9ec33c  jmp short loc_9EC343
loc_9EC33E:
9ec33e  mov esi, 1                        ; DISMISS-dialog-2 bookkeeping flag
loc_9EC343 (both confirm and dismiss-dialog-2 land here):
9ec343  ...
9ec34b  call ~CUICancelCharacterCouponRequests (destruct dialog 2 object)
9ec354  cmp esi, ebx                      ; esi == 0 ?
9ec356  jz short loc_9EC36E               ; esi==0 (CONFIRM, Encode1 ran) -> 9EC36E
loc_9EC358 (dialog-1-dismiss lands here directly, and dismiss-dialog-2 falls through here):
9ec358  call ~CUICancelCharacterCouponRequests (destruct dialog 1 object)
9ec369  jmp $LN2090_1                     ; jumptable "case 28" -> DISMISS destination
loc_9EC36E:
9ec36e  call ~CUICancelCharacterCouponRequests (destruct dialog 1 object)
9ec37f  jmp $LN232_14                     ; jumptable "cases 33,72,73" -> CONFIRM destination
```

So `esi` is set to `1` only on the dialog-2-dismiss sub-case, purely to pick
the correct one of two nearly-identical cleanup sequences (both destruct
dialog 1, but only the dialog-2-dismiss path also needed to have already
destructed dialog 2 first) before both dismiss sub-cases converge on the
same final target, `$LN2090_1`. **This refutes the prior derivation's
framing of `esi` as "an abort flag" checked broadly** — it is local
bookkeeping for object-destructor sequencing, not a flag read anywhere
else. What actually gates send-vs-no-send is *which label* control reaches
(`$LN232_14` vs `$LN2090_1`), and `esi` only ever steers case 53/54's
internal path into the correct predecessor of `$LN2090_1`. Net effect:
INFERRED as functionally an abort signal in this narrow sense (it does
result in the no-send path), but CONFIRMED it is not a generically-checked
abort flag.

### `$LN232_14` (confirm destination) traced to completion — proof it is the send path, independent of the coupon arms

`$LN232_14` is also the destination for case 62 (an ordinary item, no
dialog at all), which proves this label is the "finish the item request"
tail generically, not something specific to the coupon cancel-confirm UI:

```
9eb511  jumptable case 62:
9eb511  mov eax, [nEPOS]
9eb51d  call Encode2                       ; append per-type payload, no dialog
9eb522  jmp $LN232_14                      ; straight fall-through to shared finish
```

`$LN232_14` itself, resolved to `0x9f063c`:

```
9f063c  mov esi, [sLog]                    ; jumptable "cases 33,72,73" (= $LN232_14)
9f0648  call CanSendExclRequest
9f064d  test eax, eax
9f064f  jnz short loc_9F0666
...
9f0666  lea ecx, [oPacket]
9f066a  push ecx
9f066b  call SendPacket                    ; *** THE PACKET IS TRANSMITTED ***
9f0677  call SetExclRequestSent(1)
9f0686  ...play_game_sound...
9f06be  lea ecx, [var_208]                 ; falls into shared final cleanup
9f06c9  call RemoveAll
9f06e0  call ~ZXString(sDefaultValue)
9f06e5  jmp loc_9EB68E                     ; -> shared epilogue
```

### `$LN2090_1` (dismiss destination) traced to completion — proof it skips `SendPacket` entirely

`$LN2090_1` resolves to `0x9f06be` — the *same* final-cleanup block the
confirm path falls into *after* `SendPacket`, but the dismiss path jumps
straight there, skipping the entire `0x9f063c`–`0x9f06b9` block that
contains `CanSendExclRequest`/`SendPacket`/`SetExclRequestSent`:

```
9f0600  ...destruct pItemInfo / sDefaultValue-adjacent locals...
9f061e  call ~_com_ptr_t (pItemInfo)
9f0623  jmp $LN2090_1                      ; jumps FORWARD, over the entire
                                            ; SendPacket block, straight to:
9f06be  lea ecx, [var_208]                 ; jumptable "case 28" (= $LN2090_1)
9f06c9  call RemoveAll
9f06e0  call ~ZXString(sDefaultValue)
9f06e5  jmp loc_9EB68E                     ; -> shared epilogue
```

**CONFIRMED (v95):** `xrefs_to` the `SendPacket` symbol
(`?SendPacket@@YAXABVCOutPacket@@@Z` `@0x429b80`) inside
`SendConsumeCashItemUseRequest` returns exactly one call site,
`@0x9f066b`, physically inside the `$LN232_14` block. The dismiss path's
only route out (`$LN2090_1`, `@0x9f06be`) is physically located *after*
that call, and is reached by a direct `jmp` that never executes any
instruction in the `0x9f063c`–`0x9f06b9` range. Both destinations rejoin
the same single-`retn` epilogue (`@0x9eb6a7` via `loc_9EB68E`), but only
the confirm path passes through the `SendPacket` call to get there. Case
54 (world transfer) is structurally identical (same `$LN2090_1` /
`$LN232_14` labels reused, confirmed via the raw `0x9ec384`–`0x9ec446`
dump above), so this holds for both arms.

---

## Impact on the server-side contract

- No tail-less / partial packet variant exists on the wire for this opcode's
  name-change and world-transfer arms — the client either sends the full
  confirmed packet (opcode + slot + item id + `Encode1(1)`) or sends
  literally nothing. The server does not need to distinguish "tail byte
  present vs. absent" as a signal; any received packet of this shape,
  landing at the server at all, is the double-confirmed cancel-request.
- v83 and v95 agree on the confirm-byte value: both send `1`. There is no
  version-gated interpretation needed for this byte — a codec that treats
  `1` as "confirmed" is correct on both versions traced here (`v83`, `v95`).
  This was not independently verified on v87/v92/jms/etc. in this pass;
  only the two sessions named in the task were traced.

## One-line answers

1. Dismissing either dialog aborts the function before it ever calls `SendPacket` — no packet, tail-less or otherwise, is transmitted on either v83 or v95 (both CONFIRMED by direct disassembly to the single `SendPacket` call site and back).
2. v83 encodes `Encode1(edi)` with `edi=1` (set once at function entry, unmodified before the call) — identical to v95's literal `Encode1(1)`, not `0` (CONFIRMED).
