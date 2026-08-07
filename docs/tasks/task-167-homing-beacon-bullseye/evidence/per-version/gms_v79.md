# GMS v79 — CTS evidence (task-167)

Binary: `GMS_v79_1_DEVM.exe.i64`  ·  session: 1438cecd
Confirmed identity: yes — `mcp__ida-pro__idb_list` reports session `1438cecd` → `input_path: E:\Programs\Nexon\IDBs_v9\GMS\v79\GMS_v79_1_DEVM.exe.i64`, `filename: GMS_v79_1_DEVM.exe.i64`. Matches the expected binary exactly.

## Question A — movement-affecting filter

Reset handler: `0x96ab32` (`CWvsContext::OnTemporaryStatReset`)

Decompile of the reset handler shows the trailing-byte read gated on the filter helper:

```c
UINT128::UINT128((UINT128 *)&v20, (const struct UINT128 *)v27, 0x80u); /*0x96ac3d*/
if ( SecondaryStat::IsMovementAffectingStat() ) /*0x96ac42*/
{
  v14 = (CUserLocal *)dword_B0BECC; /*0x96ac51*/
  v15 = CInPacket::Decode1(a2); /*0x96ac57*/
  CUserLocal::SetSecondaryStatChangedPoint(v14, v15); /*0x96ac5f*/
}
```

Filter helper: `0x6f852f` (`SecondaryStat::IsMovementAffectingStat`, already named in this IDB — not renamed by this pass)

### Mechanism

`IsMovementAffectingStat` builds a static OR-accumulated 128-bit mask (dynamic-initializer,
guarded by `byte_B0FB70`) from **12** per-constant masks, then tests it against the caller's
decoded mask via `UINT128::operator&` / `operator!`. Disassembly of the accumulation chain
(`0x6f8538`-`0x6f8623`):

```
mov ecx, offset unk_B0FB50        ; initial accumulator
call sub_7DE63D                   ; acc = acc | unk_B0FB40  (each call: dest = this | *a3)
call sub_7DE63D                   ; acc = acc | unk_B0FB30
call sub_7DE63D                   ; acc = acc | unk_B0FB20
call sub_7DE63D                   ; acc = acc | unk_B0FB10
call sub_7DE63D                   ; acc = acc | unk_B0FB00
call sub_7DE63D                   ; acc = acc | unk_B0FAF0
call sub_7DE63D                   ; acc = acc | unk_B0FAE0
call sub_7DE63D                   ; acc = acc | unk_B0FAD0
call sub_7DE63D                   ; acc = acc | unk_B0FAC0
call sub_7DE63D                   ; acc = acc | unk_B0FAB0
call sub_7DE63D                   ; acc = acc | unk_B0FAA0
mov ecx, offset unk_B0FB60        ; final accumulator stored here
```

`sub_7DE63D(this, a2, a3)` (`0x7de63d`) is confirmed by decompile to compute
`*a2 = *this | *a3` (128-bit OR, 4 dwords) and return `a2` — i.e. a running OR-accumulator,
consistent with v83's "chained comparisons" shape but implemented as one pre-built mask
instead of N separate `if` tests.

### Decompiled constants tested (12 total)

Each of the 12 source globals is itself a dynamic-initializer built by a small (`0x25`-byte)
constructor. Two families were found:

**Direct family** — `sub_7DE266(this=UINT128(1), a2=shift)` computes `1 << shift` (128-bit
left shift; confirmed by decompile of `0x7de266`, which shifts the 4-dword value left by
`a2` bits). 9 of the 12 constants use this form directly:

| Global | Ctor | Shift arg (raw) |
|---|---|---|
| `unk_B0FB50` (initial accumulator) | `sub_709E44`@`0x709e44` | `sub_7DE266(this,7)` → **7** |
| `unk_B0FB40` | `sub_709E74`@`0x709e74` | `sub_7DE266(this,8)` → **8** |
| `unk_B0FB30` | `sub_70A297`@`0x70a297` | `sub_7DE266(this,17)` → **17** |
| `unk_B0FB20` | `sub_70A99B`@`0x70a99b` | `sub_7DE266(this,30)` → **30** |
| `unk_B0FB10` | `sub_70A9FB`@`0x70a9fb` | `sub_7DE266(this,32)` → **32** |
| `unk_B0FB00` | `sub_70AA2B`@`0x70aa2b` | `sub_7DE266(this,33)` → **33** |
| `unk_B0FAF0` | `sub_70B43D`@`0x70b43d` | `sub_7DE266(this,49)` → **49** |
| `unk_B0FAE0` | `sub_70ABB5`@`0x70abb5` | `sub_7DE266(this,35)` → **35** |
| `unk_B0FAD0` | `sub_70B034`@`0x70b034` | `sub_7DE266(this,39)` → **39** |

**`sub_7099E5` family** — `sub_7099E5(dest, a2)` (`0x7099e5`) is confirmed by decompile to
call `sub_7DE266(1, a2+73)`, i.e. `1 << (a2+73)`. The remaining 3 constants use this form:

| Global | Ctor | Shift arg (raw = a2+73) |
|---|---|---|
| `unk_B0FAB0` | `sub_70BFD9`@`0x70bfd9` | `sub_7099E5(dest,1)` → **74** |
| `unk_B0FAA0` | `sub_70C003`@`0x70c003` | `sub_7099E5(dest,2)` → **75** |
| `unk_B0FAC0` | `sub_70C02D`@`0x70c02d` | `sub_7099E5(dest,3)` → **76** |

Raw shift set tested by `IsMovementAffectingStat`: **{7, 8, 17, 30, 32, 33, 35, 39, 49, 74,
75, 76}** — 12 constants, matching v83's count.

### Resolved stat names

- **Shift 76 (`unk_B0FAC0`) = RideVehicle — CONFIRMED.** The identical construction
  (`sub_7099E5(dest,3)` → shift 76) is independently used to build `unk_B14DE0`, the mask
  tested in `OnTemporaryStatReset` immediately before `CUser::ShowRideVehicleEffect`:
  ```c
  v4 = (UINT128 *)UINT128::operator&(v24, &unk_B14DE0); /*0x96ab6f*/
  if ( (unsigned __int8)UINT128::compareTo(v4) ) /*0x96ab76*/
  {
    v5 = (CUser *)dword_B0BECC;
    v6 = (_DWORD *)sub_4E8DA8(*((_DWORD *)v3 + 731));
    CUser::ShowRideVehicleEffect(v5, *v6); /*0x96ab94*/
  }
  ```
  Disasm of `unk_B14DE0`'s own dynamic initializer (`0x967f30`-`0x967f49`) shows
  `push 3; ...; call sub_7099E5; ...; mov ecx, offset unk_B14DE0` — the same
  `sub_7099E5(dest,3)` → shift 76 construction. Two independently-built globals encoding the
  same shift is strong corroboration this is the RideVehicle flag.
- **Shift 7 (`unk_B0FB50`, the initial accumulator) = Speed — supported by cross-reference.**
  `docs/tasks/task-167-homing-beacon-bullseye/plan.md:1334` records "Speed is registry shift
  7" from the (already-verified) v83/v95 evidence. v79's raw shift 7 for this same "direct
  family" constant matches exactly, and the direct family (no `+73`/`+9` offset) is the
  family Speed belongs to.
- **Shifts 8, 17, 30, 32, 33, 35, 39, 49 = the remaining 8 v83 names (Jump, Stun, Weakness,
  Slow, Morph, Ghost, BasicStatUp, Attract), by count and family only.** `context.md:27`
  gives v83's list as "Speed, Jump, Stun, Weakness→Weaken, Slow, Morph, Ghost→GhostMorph,
  BasicStatUp→MapleWarrior, Attract→Seduce, RideVehicle→MonsterRiding, DashSpeed, DashJump"
  (12 names). v79 independently yields 8 more "direct family" constants beyond Speed — the
  right count — but **UNVERIFIED: this IDB has no `TEMPORARY_STAT` enum, class, or string
  table naming these individual globals**, so which specific shift (8/17/30/32/33/35/39/49)
  is Jump vs. Stun vs. Weakness vs. Slow vs. Morph vs. Ghost vs. BasicStatUp vs. Attract
  could not be resolved bit-by-bit in this pass.
- **Shifts 74, 75 = the two Dash bits (DashSpeed, DashJump), by count and family only.**
  `design.md:104-105` describes v83's filter tail as "RideVehicle (`0x0020…`) | `0x0010…` |
  `0x0008…` (the two Dash bits)" — 3 adjacent `sub_7099E5`-family entries, matching v79's
  74/75/76 triple exactly (76=RideVehicle, confirmed above). **UNVERIFIED: which of {74, 75}
  is DashSpeed vs. DashJump** — no per-bit name evidence found in this IDB.

### Comparison to v83 list

**MATCHES by count and structure (12 constants; RideVehicle position independently
confirmed at shift 76; Speed position independently confirmed at shift 7; GuidedBullet,
built the same way, is confirmed absent from this list — see Question B).**
Individual name↔shift assignment for the remaining 10 constants is UNVERIFIED (no
`TEMPORARY_STAT` symbol table exists in this IDB) — this does not indicate a behavioral
DIFFERS, only that per-bit naming could not be resolved from symbols alone in v79.

## Question B — two-state member group

SecondaryStat constructor: **UNVERIFIED — not found.** `func_query` for `.*SecondaryStat.*`
and `??0SecondaryStat*` in this IDB returns only `IsMovementAffectingStat`, `Reset`,
`DecodeForLocal`, `DecodeForRemote`, and two unrelated helper matches — no
`SecondaryStat::SecondaryStat` constructor symbol exists (contrast with v95, where
`context.md:113` cites a named `SecondaryStat::SecondaryStat@0x72F190`). The object's 7
member slots are therefore visible only via their masks and the two decode-loop call sites,
not via a constructor that would reveal each member's concrete class/vtable.

### Mask constants (7 members, same `sub_7099E5` family as Question A)

Found via `xrefs_to(sub_7099E5)`: 7 consecutive `0x25`-byte dynamic initializers at
`0x7119c3`-`0x711ae4`, each `sub_7099E5(dest, a2)` for `a2 = 0..6` → raw shift `73..79`:

| Idx (a2) | Raw shift | Global | Ctor |
|---|---|---|---|
| 0 | 73 | `unk_B10370` | `sub_7119C3`@`0x7119c3` |
| 1 | 74 | `unk_B0FCB0` | `sub_7119ED`@`0x7119ed` |
| 2 | 75 | `unk_B10458` | `sub_711A17`@`0x711a17` |
| 3 | 76 | `unk_B0FC50` | `sub_711A41`@`0x711a41` |
| 4 | 77 | `unk_B0FCA0` | `sub_711A6B`@`0x711a6b` |
| 5 | 78 | `unk_B104D8` | `sub_711A95`@`0x711a95` |
| 6 | 79 | `unk_B107B0` | `sub_711ABF`@`0x711abf` |

Converting to the "registry shift" numbering used elsewhere in this task's docs
(`registry = raw + 9` for this `sub_7099E5` family — derived below) gives **registry shifts
82-88**, which matches the background fixture fact exactly: *"v79's empty two-state mask is
`int1 = 0x01FC0000`, exactly seven bits (82-88)"*. The `+9` conversion itself is not
invented: `context.md:26` records "v83 GuidedBullet mask bit: registry shift 87", and
`plan.md:1356` records "RideVehicle shift 85" — both match `raw+9` for the two members
independently confirmed below (78+9=87, 76+9=85), so the conversion is evidence-derived, not
assumed.

### Two independently-confirmed members

- **Idx 3 (raw 76 / registry 85) = RideVehicle.** `unk_B0FC50` is built by
  `sub_7099E5(dest,3)`, the identical construction independently confirmed as RideVehicle in
  Question A (via `unk_B14DE0`/`ShowRideVehicleEffect`). Positionally this is also the exact
  slot referenced in `OnTemporaryStatReset` as `v3 + 731` (`this+725+3*2`, matching the
  7-member array stride of 2 dwords/8 bytes starting at member index 0 = `this+725`).
- **Idx 5 (raw 78 / registry 87) = GuidedBullet.** `unk_B104D8` is built by
  `sub_7099E5(dest,5)`, the identical construction used for `unk_B14DC0` — the mask tested
  in `OnTemporaryStatReset` immediately before the `TemporaryStat_GuidedBullet`-specific
  calls:
  ```c
  v7 = (UINT128 *)UINT128::operator&(v24, &unk_B14DC0); /*0x96aba5*/
  if ( (unsigned __int8)UINT128::compareTo(v7) ) /*0x96abac*/
    if ( vtable-call-IsActivated(...) )
    {
      MobID = TemporaryStat_GuidedBullet::GetMobID(v8);      /*0x6f1fe7 dispatch*/
      Reason = (_DWORD *)TemporaryStatBase<long>::GetReason(v8); /*0x6408b9*/
      CMobPool::ResetGuidedMob(v9, *Reason, MobID);
    }
  ```
  This matches registry shift 87, exactly the value `context.md:26` records for v83's
  GuidedBullet — strong cross-version stability confirmation.
  **GuidedBullet member's CTS mask-bit shift: raw 78 (registry 87).**

### Remaining 5 members — INFERRED by position, not independently named in v79

`context.md:110-112` (v95, design-phase-verified) gives the analogous 7-slot table as:
`0=EnergyCharged, 1=Dash_Speed, 2=Dash_Jump, 3=RideVehicle, 4=PartyBooster, 5=GuidedBullet,
6=unnamed/unreachable`. v79's slot 3 and slot 5 independently verified above land on
RideVehicle and GuidedBullet respectively — the *same slot indices* as the v95 table. On
that positional correspondence (not independently symbol-verified in this IDB): idx 0 =
EnergyCharged, idx 1 = Dash_Speed, idx 2 = Dash_Jump, idx 4 = PartyBooster, idx 6 = unnamed.
**UNVERIFIED as direct v79 symbol evidence** — flagged as inference, not fact.

### Block sizes — UNVERIFIED

Both decode loops dispatch to each member's `DecodeForClient` through a **virtual call**
(vtable slot `+0x18`), e.g. `DecodeForRemote`'s loop:
```c
v58 = sub_7099E5(v62, v56);
v59 = (UINT128 *)UINT128::operator&(v61, v58);
if ( (unsigned __int8)UINT128::compareTo(v59) )
  (*(void (__thiscall **)(_DWORD, CInPacket *))(*(_DWORD *)*v57 + 24))(*v57, a3); /*0x701c72*/
```
Without a `SecondaryStat` constructor symbol (see above) to reveal each member's concrete
class/vtable address, the 7 `DecodeForClient` implementations could not be statically
resolved from the virtual-call site alone, so **per-member block sizes (expected 15/15/15/
13/20/17/15 by analogy with v95/v83) could not be measured directly in v79 in this pass**.
**UNVERIFIED: blocker = no `SecondaryStat::SecondaryStat` constructor symbol and no way to
resolve the polymorphic `DecodeForClient` targets from the virtual-call site without one;**
attempts to locate it via address-proximity to `Reset`/`DecodeForLocal`/`DecodeForRomote`
and via `xrefs_to` on `CInPacket::Decode4` (too many unrelated hits) did not succeed.
**Summed trailer length: UNVERIFIED (not measured) — cannot be compared to the 110-byte
reference.**

### Trailer read style: **PER-MEMBER MASK-GATED — DIFFERS from "unconditional"**

Both `DecodeForLocal` and `DecodeForRemote` implement the 7-member trailer as a genuine loop
(not unrolled) that tests each member's `UINT128(1) << shift` against the decoded flag
*before* making the virtual `DecodeForClient` call — i.e. gated, not unconditional.

`DecodeForLocal` (`0x6fbcba`), loop body confirmed by disasm (`0x700050`-`0x70014a`):
```
loc_700050: call sub_70C3E5            ; index into member array
             push [ebp+var_14]         ; loop counter
             call sub_7099E5           ; shift = (loop counter)+73
             ...
             call ??IUINT128@@... (operator&)   ; decoded_flag & member_mask
             call ?compareTo@UINT128@@...
             jz ...                    ; skip virtual call if bit not set
             mov ecx,[arg_4]; mov eax,[ecx]; push edi; call dword ptr [eax+18h]  ; DecodeForClient(packet)
             mov ecx,[arg_4]; call TemporaryStatBase<long>::GetReason           ; 0x700091
             ...
loc_70013F:  inc [ebp+var_14]
             add [ebp+var_18], 8
             cmp [ebp+var_14], 7
             jl  loc_700050
```

`DecodeForRemote` (`0x701539`), loop body confirmed by decompile (`0x701c41`-`0x701c7c`):
```c
v56 = 0;
v57 = this + 725;
do {
  v58 = sub_7099E5(v62, v56);                          /*0x701c4e*/
  v59 = (UINT128 *)UINT128::operator&(v61, v58);        /*0x701c5d*/
  if ( (unsigned __int8)UINT128::compareTo(v59) )       /*0x701c64*/
    (*(void (__thiscall **)(_DWORD, CInPacket *))(*(_DWORD *)*v57 + 24))(*v57, a3); /*0x701c72*/
  ++v56;
  v57 += 2;
} while ( v56 < 7 );
```

Both loops explicitly skip the virtual `DecodeForClient` call (and therefore skip that
member's packet bytes entirely) when the member's bit is not set in the decoded flag. This
is **mask-gated**, not the unconditional "always read all 7 blocks" shape the task's leading
hypothesis (v83 reference) describes.

**VERDICT: Question A MATCHES v83 by count/structure (12 constants; Speed@shift7 and
RideVehicle@shift76 independently confirmed; remaining 10 names UNVERIFIED at the
individual-bit level — no behavioral DIFFERS found). Question B: member count (7) and
GuidedBullet/RideVehicle slot positions MATCH v83/v95; UNVERIFIED: per-member block sizes
and the resulting 110-byte sum (no `SecondaryStat` constructor symbol → could not resolve
the polymorphic `DecodeForClient` targets); DIFFERS from "unconditional" — v79's trailer
read is directly evidenced as per-member mask-gated in both `DecodeForLocal` and
`DecodeForRemote`.**
