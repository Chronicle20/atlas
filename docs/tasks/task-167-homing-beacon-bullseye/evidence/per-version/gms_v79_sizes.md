# GMS v79 — SecondaryStat two-state group: per-member block sizes

Session `1438cecd`, binary `GMS_v79_1_DEVM.exe.i64` (confirmed via `idb_list`).

## Method used

1. Confirmed the 7-member decode loop in `SecondaryStat::DecodeForRemote`
   (`0x701539`, size `0x760`) and `SecondaryStat::DecodeForLocal` (`0x6fbcba`,
   size `0x44fc`). Both end in the identical pattern:

   ```c
   v56 = 0;
   v57 = this + 725;              // byte offset 2900 (0xB54) into SecondaryStat
   do {
     v58 = sub_7099E5(mask, v56);
     if (UINT128::compareTo(mask & v58))
       (*(vtable+0x18))(*v57, packet);   // *(_DWORD*)*v57 = vtable ptr
     ++v56;
     v57 += 2;                    // stride = 2 DWORDs = 8 bytes
   } while (v56 < 7);
   ```

   Confirmed at the disasm level (`0x701c41`-`0x701c7c`):
   `add esi, 0B54h` (base) ... `mov ecx,[esi]` / `mov eax,[ecx]` /
   `call dword ptr [eax+18h]` ... `add esi, 8` ... `cmp ebx, 7`.

   This gives the 7 slot byte-offsets into `SecondaryStat`:
   slot0=0xB54(2900), slot1=0xB5C(2908), slot2=0xB64(2916), slot3=0xB6C(2924),
   slot4=0xB74(2932), slot5=0xB7C(2940), slot6=0xB84(2948).

2. Attempted to resolve each slot's concrete vtable (and hence its
   `DecodeForClient` address at vtable+0x18) by tracing the member pointer
   stored at each slot back to its construction site, per the task's
   prescribed technique ("trace the member-pointer array to the vtables, or
   find the vtables by their DecodeForClient/EncodeForClient slot contents").
   This required finding where `*(this+offset)` is first **written** (not
   read) with `new`+vtable-set, since the call at `[eax+0x18]` is a pure
   runtime indirect call with no static resolution possible otherwise (no
   comment/xref annotation on the call site — verified via `disasm`).

3. Exhausted every static-analysis avenue available without a constructor
   symbol:
   - `func_query` for `SecondaryStat`/`TemporaryStat_*` constructors,
     `??0SecondaryStat`, `??0CUserRemote`, `??0CharacterData`, `??_7...6B@`
     (vtable symbols), `??__E...` (dynamic global initializers) — **all
     empty**. This binary has no RTTI (`entity_query kind=strings` for
     "GuidedBullet"/"TemporaryStat" → 0 hits) and no local struct type for
     `SecondaryStat` (`search_structs` → 0 hits), so Hex-Rays never
     propagated a concrete type onto the `this` pointer in
     `DecodeForLocal`/`DecodeForRemote` (both show `_DWORD *this`).
   - `find type=immediate` for each of the 7 slot byte-offsets
     (2900/2908/2916/2924/2932/2940/2948) across the **whole binary** —
     every hit not already inside the known Decode/Reset loops was a false
     positive from an unrelated struct that happens to reuse the same small
     integer offset (a `CWnd`-derived UI class at `0x7d85fc`, chat-log
     drawing code at `0x83a6f1`, unrelated `CharacterData`/mob fields at
     `0x8229b0`, etc. — checked and ruled out by decompiling each).
   - `search_text` for the hex literal `0B7Ch` (slot 5's offset) restricted
     to `0x6e0000-0x900000` in ranged chunks — only hit was the already-known
     **read** site in `sub_706287` (`mov esi, [eax+0B7Ch]`).
   - `search_text` for `operator new` inside the `SecondaryStat` translation
     unit (`0x6e0000-0x710000`, bounded to avoid timeout) — **zero calls**;
     allocation of the 7 members (if heap-based) does not happen in this TU.
   - Located and ruled out an **unconditional** full-reset function
     `sub_6F736F` (`0x6f736f`, immediately follows the sibling `ForcedStat`
     class's TU) which also walks the same 7-slot array — but it only calls
     the shared non-virtual `sub_725A9C` (a common base-class field reset,
     not type-specific), so it presupposes the pointers already exist; it is
     not the constructor either.
   - Identified 6 candidate "GetMaxXxx"-style helper functions between
     `sub_6F736F` and `Reset` (`0x6f8643`, `0x6f86d3`, `0x6f8769`, `0x6f87ff`,
     `0x6f8947`, `0x6f8a4a`) that also dereference slot 0 (`this+2900`) via
     `vtbl+4` — useful confirmation of the offset map, but they are
     game-logic getters (skill-bonus computations), not constructors, and
     reveal no type/vtable identity.

4. **One slot's identity WAS resolved** via a non-virtual (hence
   statically-linked) call rather than the vtable: `sub_706287` (`0x706287`,
   a guided-bullet damage-bonus helper referenced from
   `?OnAfterImage@…`-adjacent skill code) contains:

   ```c
   v6 = *(TemporaryStat_GuidedBullet **)(a2 + 2940);   /*0x7062b1*/
   if (v6 && (*(vtbl+4))(v6))                           /*0x7062bf*/
     if (TemporaryStat_GuidedBullet::GetMobID(v6) == a3) /*0x7062c8, direct call to 0x631fe7*/
       ...
   ```

   Offset `2940` = `0xB7C` = slot 5 (0-indexed) in the loop above. This
   confirms **slot 5 = `TemporaryStat_GuidedBullet`**, matching the
   pre-95 reference's 17-byte member (base 13 + 4-byte `dwMobId`) by class
   identity — but the actual `DecodeForClient` byte count for this slot
   still could not be measured (see below): `GetMobID` is a plain accessor,
   not the decoder, and the class has no vtable symbol to read slot+0x18 from.

## Per-member table

| Slot (0-idx) | Byte offset | Vtable addr | DecodeForClient addr | Size | Evidence |
|---|---|---|---|---|---|
| 0 | 0xB54 (2900) | UNVERIFIED | UNVERIFIED | UNVERIFIED | No write site found for this slot's pointer; only reads (`Reset`, `DecodeForRemote`/`DecodeForLocal` loops, `sub_6F8643`'s `vtbl+4` check at `this+2900`) |
| 1 | 0xB5C (2908) | UNVERIFIED | UNVERIFIED | UNVERIFIED | No write site found |
| 2 | 0xB64 (2916) | UNVERIFIED | UNVERIFIED | UNVERIFIED | No write site found |
| 3 | 0xB6C (2924) | UNVERIFIED | UNVERIFIED | UNVERIFIED | No write site found |
| 4 | 0xB74 (2932) | UNVERIFIED | UNVERIFIED | UNVERIFIED | No write site found |
| 5 | 0xB7C (2940) | UNVERIFIED | UNVERIFIED | UNVERIFIED | Class identity confirmed = `TemporaryStat_GuidedBullet` via non-virtual `GetMobID` call in `sub_706287` (`0x7062c8`); no vtable symbol / constructor to read slot+0x18 from |
| 6 | 0xB84 (2948) | UNVERIFIED | UNVERIFIED | UNVERIFIED | No write site found |

## Subtotal / verdict

`PARTIAL: 0 of 7 measured, subtotal 0`

**Blocker (applies to all 7 slots):** the `DecodeForClient` call at
`vtable+0x18` (confirmed by disasm at `0x701c72`:
`call dword ptr [eax+18h]`) is a pure runtime indirect call. Resolving it to
a concrete function address requires knowing the vtable address written into
each member object at construction time — but this binary has **no
constructor symbol** for `SecondaryStat` or any `TemporaryStat_*` class, **no
RTTI** (no `.?AV...` type-descriptor strings), **no vtable symbols**
(`??_7...6B@` pattern), and **no local struct type** for `SecondaryStat`
(Hex-Rays never got field-level type info, hence `this` stays `_DWORD *`
throughout). Every static-analysis avenue used successfully in other
versions of this campaign (name search, immediate-value search bounded to
plausible ranges, `operator new` text search, RTTI/vtable-symbol search,
struct-type search) was tried here and came up empty except for the one
non-virtual `GetMobID` call that identified slot 5's class. This matches
and confirms the prior pass's conclusion that v79 lacks the anchor needed
to measure per-member block sizes — no fabricated total is provided.
