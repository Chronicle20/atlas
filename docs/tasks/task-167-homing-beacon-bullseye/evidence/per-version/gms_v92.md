# GMS v92 — CTS evidence (task-167)

Binary: `GMS_v92_1_DEVM.exe.i64` (path `E:\Programs\Nexon\IDBs_v9\GMS\V92_1\GMS_v92_1_DEVM.exe.i64`) · session: acdfccff · identity confirmed: `idb_list` returned session `acdfccff` mapped to exactly this `input_path`/`filename`, matching the expected `GMS_v92_1_DEVM.exe.i64`.

## Anchor discovery

Reset handler: **`CWvsContext::OnTemporaryStatReset` @ 0x9c7800** (size 0x26d)
HOW FOUND: `func_query` with `name_regex: "OnTemporaryStatReset|TemporaryStatReset|OnStatReset"` returned the exact demangled symbol `?OnTemporaryStatReset@CWvsContext@@QAEXAAVCInPacket@@@Z` directly by name — the mangled C++ name was still present in this IDB, no signature/heuristic search was needed.
CONFIDENCE: **high**. Beyond the exact name match, the decompiled body is self-consistent with the known CTS shape (`CInPacket::DecodeBuffer(a2, buf, 0x10u)` — a 16-byte mask read — followed by per-stat mask-gated resets, a `SecondaryStat::Reset` call, and a conditional `CInPacket::Decode1` trailer). The function also carries pre-existing inline comments from a prior corroboration pass (task-013) pinning `CWvsContext::m_secondaryStat @ 0x2148` and cross-referencing v95 (`@0x9F2B1A`) and v87 (`@0xAB7E0E`) member offsets for an adjacent field (`m_forcedStat`), independently corroborating this is the right function/class in this IDB.

## Question A — movement-affecting filter

Filter helper: **`sub_705080` @ 0x705080**

Call site in the reset handler (0x9c797c): `if ( sub_705080(v16, v17) ) { v12 = CInPacket::Decode1((int)a2); sub_8EACA0(v12); }` — `v16..v19` is the 4×DWORD (128-bit) mask returned by the preceding `SecondaryStat::Reset(v16, v17, v18, v19)` call, i.e. the same "decoded 16-byte mask" the packet carries. `sub_705080` decompiles as a parameterless-looking `BOOL sub_705080()` because hexrays folded the true (`this`=address of the 128-bit mask) argument into an implicit `ecx`; functionally it computes a composite OR of 13 fixed bit-constants (once, guarded by `dword_C38930`) and returns whether that composite intersects the mask (`sub_7F50A0`=UINT128 AND, `sub_7F4F00`=IsZero, so the function returns `!IsZero(mask & composite)`).

Confirmed primitive semantics (decompiled):
- `sub_7F5010(dest, src)` → `dest = *this | *src` (UINT128 OR)
- `sub_7F50A0(dest, src)` → `dest = *this & *src` (UINT128 AND)
- `sub_7F4F00(x)` → `return (x==0)` (IsZero, all 4 dwords tested)
- `sub_7F5100(x)` → wraps `sub_7F4DF0(0) != 0` (nonzero test used pervasively as the packet-mask "if (mask & FLAG)" gate)

Constants tested — quoted from `sub_705080` (0x705080), each `unk_Cxxxxx` global is a 16-byte (UINT128) constant OR'd into the composite filter mask:

```
sub_7F5010(v3,  &unk_C37918);   /*0x70516c*/
sub_7F5010(v13, &unk_C37928);   /*0x705173*/
sub_7F5010(v11, &unk_C37938);   /*0x70517a*/
sub_7F5010(v9,  &unk_C37948);   /*0x705181*/
sub_7F5010(v7,  &unk_C37958);   /*0x705188*/
sub_7F5010(v5,  &unk_C37968);   /*0x70518f*/
sub_7F5010(v4,  &unk_C37978);   /*0x705196*/
sub_7F5010(v12, &unk_C37988);   /*0x70519d*/
sub_7F5010(v8,  &unk_C37998);   /*0x7051a4*/
sub_7F5010(v14, &unk_C379A8);   /*0x7051ab*/
sub_7F5010(v6,  &unk_C379B8);   /*0x7051b2*/
sub_7F5010(v15, &unk_C379C8);   /*0x7051b9*/
v0 = sub_7F5010(v10, &unk_C379D8); /*0x7051c0*/
```

Each `unk_Cxxxxx` global is itself lazily constructed as `UINT128(1) << N` by a tiny dedicated one-shot initializer function (`sub_7F4F30(1)` = `UINT128(1)`, `sub_7F4E20(N)` = `<<= N`). Resolved shift, BASIS = **raw client bit shift** (the literal `N` passed to `sub_7F4E20`, read directly from each initializer's decompile — NOT the atlas registry basis):

| global | initializer | shift (raw) |
|---|---|---|
| unk_C37918 | sub_AD8B90 (`sub_7F4E20(8)`) | **8** |
| unk_C37928 | sub_AD8D40 (`sub_7F4E20(17)`) | **17** |
| unk_C37938 | sub_AD8FB0 (`sub_7F4E20(30)`) | **30** |
| unk_C37948 | sub_AD9010 (`sub_7F4E20(32)`) | **32** |
| unk_C37958 | sub_AD9040 (`sub_7F4E20(33)`) | **33** |
| unk_C37968 | sub_AD9340 (`sub_7F4E20(49)`) | **49** |
| unk_C37978 | sub_AD90A0 (`sub_7F4E20(35)`) | **35** |
| unk_C37988 | sub_AD9160 (`sub_7F4E20(39)`) | **39** |
| unk_C37998 | sub_ADA060 (`sub_7F4E20(118)`) | **118** |
| unk_C379A8 | sub_AD9FE0 (`sub_7F4E20(116)`) | **116** |
| unk_C379B8 | sub_ADA020 (`sub_7F4E20(117)`) | **117** |
| unk_C379C8 | sub_AD9970 (`sub_7F4E20(82)`) | **82** |
| unk_C379D8 | sub_AD99A0 (`sub_7F4E20(83)`) | **83** |

So the 13 raw bit shifts tested are: **8, 17, 30, 32, 33, 35, 39, 49, 82, 83, 116, 117, 118**.

Resolved stat names: **UNVERIFIED for most bits.** This binary carries no per-bit textual/RTTI label for the plain scalar `long` stats (they are decoded as raw `Decode1/Decode2/Decode4` reads gated only by a bit-shift constant, with no distinguishing class name) — unlike bits 115–121 (Question B's two-state group), where at least one member (`TemporaryStat_GuidedBullet`) does carry a positively-identifying class name. Bits 116, 117 and 118 fall inside the Question B two-state member's bit range (115–121, confirmed below), meaning at least 3 of these 13 movement-affecting stats are two-state/pointer-dispatched members rather than plain scalars — plausibly Mount/RideVehicle/GuidedBullet-family flags — but I could not conclusively bind any specific one of {116,117,118} to `TemporaryStat_GuidedBullet` specifically (see Question B blocker) or to any of the reference names (Attract, RideVehicle, DashSpeed, DashJump, Flying, Frozen, YellowAura, etc.) without a name-bearing symbol or cross-binary bit-value corroboration, which was explicitly out of scope per the anchor-independence instruction for this pass.

Comparison: **DIFFERS from both reference lists by count** — 13 movement-affecting bits found, vs. v83's 12 and v95's 15. The raw bit *values* (8,17,30,32,33,35,39,49,82,83,116,117,118) also don't look like a simple "v83 ∪ 1 stat" or "v95 − 2 stats" pattern by inspection, but without name resolution I cannot state which named stats were added/dropped relative to either reference list. **UNVERIFIED: exact stat-name membership** — only the raw shift set and count are confirmed; do not infer the v83/v95 name lists apply.

## Question B — two-state member group

SecondaryStat constructor: **UNVERIFIED — no distinct constructor symbol found.** `func_query` for `^\?\?0SecondaryStat` and broader `SecondaryStat` filters returned only `SecondaryStat::Reset` (0x70f320), `SecondaryStat::DecodeForRemote` (0x711240), and `SecondaryStat::DecodeForLocal` (0x71a9f0) — no `??0SecondaryStat` constructor exists as a standalone symbol (likely inlined into `CWvsContext`'s/`CUserRemote`'s own constructor, or the member vtables are statically baked into the linked image rather than constructed at runtime — see blocker below). The **member group and its per-member dispatch was instead recovered from `SecondaryStat::DecodeForRemote` (0x711240)**, which ends in an explicit fixed loop over the two-state group:

```c
v86 = 0;                              /*0x711e47*/
v87 = this + 1151;                    /*0x711e49*/
do {
  sub_7F4F30(1);                      /*0x711e5f*/
  v88 = sub_7F4E20(v86 + 115);        /*0x711e66*/   // raw shift = 115 + v86
  sub_7F4F50(v88, 128);               /*0x711e70*/
  v89 = sub_7F50A0(v93, v92);         /*0x711e83*/    // AND against decoded packet mask
  if ( (unsigned __int8)sub_7F5100(v89) )             /*0x711e8a*/
    (*(void (__thiscall **)(_DWORD, int))(*(_DWORD *)*v87 + 24))(*v87, a3); /*0x711e9b*/  // virtual DecodeForClient dispatch
  ++v86;                              /*0x711e9d*/
  v87 += 2;                           /*0x711e9e*/
} while ( v86 < 7 );                  /*0x711ea4*/
```

This confirms **7 members**, gated on raw bit shifts **115, 116, 117, 118, 119, 120, 121** (one per loop iteration, `v86`=0..6), each dispatched through a per-member virtual `DecodeForClient` call (vtable offset +24) only if `mask & (1<<(115+i))` is nonzero.

Ordered members (name, DecodeForClient addr, block size) — **order within the 7-slot array (`this+1151`, stride 8 bytes/slot) is UNVERIFIED**; I could not resolve which of the 7 pointer slots holds which class instance (see blocker), but I positively identified the **distinct decode implementations** reachable via this dispatch pattern (vtable slot @ virtual-offset +24, called with a `TemporaryStatBase<long>`-derived object):

| class / function | addr | reads (quoted) | block size |
|---|---|---|---|
| `TemporaryStatBase<long>::DecodeForClient` (base) | 0x70cc10 | `DecodeBuffer(a2,this+3,4)` + `DecodeBuffer(a2,this+4,4)` + `DecodeTime(a2)` [=`Decode1`(1)+`Decode4`(4)=5] → 4+4+5 | **13** |
| `TwoStateTemporaryStat<long,not_equal<long,0>,NoExpire,...>::DecodeForClient` | 0x70cd00 | calls base only, no extra reads | **13** |
| unnamed variant A (`sub_70CDF0`) | 0x70cdf0 | base(13) + `CInPacket::Decode2(a2)` | **15** |
| unnamed variant B (`sub_712180`) | 0x712180 | base(13) + `CInPacket::Decode2(a2)` (code-identical to variant A but a distinct symbol/vtable) | **15** |
| `TemporaryStat_GuidedBullet::DecodeForClient` | 0x70d260 | calls NoExpire-variant (13) + `CInPacket::Decode4(a2)` | **17** |
| `TwoStateTemporaryStat<long,not_equal<long,0>,Expire<BaseOnCurrentTime,DynamicTermSet>,...>::DecodeForClient` | 0x712040 | base(13) + `DecodeTime(a2)`(5, but decompiled as a direct `` `anonymous namespace'::DecodeTime`` call rather than base's inline copy — see note) + `CInPacket::Decode2(a2)`(2) → 13+5+2 | **20** |

Note on the Expire-variant's arithmetic: `0x712040`'s own decompile is `TemporaryStatBase<long>::DecodeForClient(a2)` [call with no explicit `this`, i.e. reuses the *shared* base-read code inline] + `DecodeTime(a2)`→4 bytes stored at `this+40` + `Decode2`→2 bytes at `this+48`. Taking the base's own internal reads (4+4+5=13) plus these two extra fields (4[DecodeTime's return, stored as a raw 4-byte field — the *call* to DecodeTime here reads 5 bytes off the wire per DecodeTime's own body] +2) gives 13+5+2=20, matching the pre-95 reference's "20" entry exactly.

A 7th distinct vtable/data address (`0xb361b8`) was found to also target `0x712040` (i.e., a *second* class instantiation reusing the identical Expire<BaseOnCurrentTime,...> code, same 20-byte size) — consistent with two of the seven slots sharing this shape. This gives 6 of 7 slots positively enumerated by size (13, 13or15, 15, 15, 17, 20, 20 — one slot short of 7; the two "13-byte" candidates `0x70cc10`/`0x70cd00` may represent the *same* logical slot rather than two, since `0x70cc10` is the abstract base and its own vtable hit (`0xb75d70`) was found in a memory region (`0xb75xxx`) far outside the tight `0xb36xxx` cluster all the other 6 confirmed slot-vtables occupy, suggesting `0xb75d70` belongs to an unrelated consumer of the same template, e.g. `CMob`'s own temp-stat system).

Summed trailer length: **UNVERIFIED — approximately 110 (if the 7th slot is 15) to 113–115 depending on which candidate fills the ambiguous slot.** Confirmed distinct sizes found: {13, 15, 15, 17, 20, 20} across 6 positively-clustered vtables (`0xb36140`→13, `0xb361f4`→15, `0xb3617c`→15, `0xb362f4`→17/GuidedBullet, `0xb360f0`→20, `0xb361b8`→20) = subtotal **100** for 6 slots; a 7th slot is required by the loop's `while (v86<7)` bound but its distinguishing vtable/size was not conclusively isolated (candidates found in the same cluster — `0xb360a4`, `0xb36160`, `0xb361d8` — resolve to a *different*, smaller "policy" vtable shape for the `Expire<BaseOnLastUpdatedTime,DynamicTermSet>` variant whose slot-6 entry did not land at the expected function per manual byte inspection, so I did not force a size onto it). **This is closer to the pre-95 reference shape (7 members / 110 bytes) than to the v95 shape (6 members / 95 bytes) by member COUNT (7, confirmed) and by the four now-plausible individual sizes matching the pre-95 list {13,15,15,17,20} exactly, but the precise 7th-member size and full order remain UNVERIFIED.**

GuidedBullet mask-bit shift: **UNVERIFIED — which of {115,116,117,118,119,120,121} (raw basis) corresponds to `TemporaryStat_GuidedBullet` specifically.** `TemporaryStat_GuidedBullet::DecodeForClient`/`EncodeForClient` (0x70d260 / 0x7167c0) contain no embedded mask-shift constant (the mask test lives entirely in the generic loop shown above, not in the per-member class), and I could not resolve the `this+1151` array's per-slot class assignment (no code cross-reference sets these pointers — they are presumably populated at static-initialization/link time by a bulk `memcpy` of a prototype block rather than per-field vtable stores, which left no traceable "constructor" for `xrefs_to` to find). BASIS: n/a (unresolved). Blocker: no live/heap instance available in a static IDB to inspect `this+1151`'s actual stored pointers, and no constructor code sets them individually.

Raw-vs-registry offset: **UNVERIFIED** (no atlas registry shift values were consulted for this pass per the task's anchor-independence framing; all shifts reported above are raw-client-basis only).

Trailer read style: **per-member mask-gated**, strong evidence — every single field decode in both `SecondaryStat::DecodeForRemote` (0x711240) and the `OnTemporaryStatReset`/`OnTemporaryStatSet` (0x9c7160) callers is wrapped in `if (sub_7F5100(sub_7F50A0(mask, &unk_Cxxx)))` before decoding, including the 7-member loop itself (`1<<(115+i)` tested against the mask before each virtual `DecodeForClient` call). No unconditional trailer reads were observed anywhere in these three functions.

Conditional members? All 7 loop members are individually mask-gated (see loop above) — this differs from the v95 reference's specific claim that only the *last two* of 6 members are conditional; here **all 7** are conditional, uniformly, via the same per-iteration `1<<(115+i)` test. This is itself evidence against the v95 6-member/last-two-conditional shape and toward the pre-95 7-member shape (where, per the reference, the 7-member group's whole 110-byte block was described as a single unconditional 110-byte trailer in that era's client — but v92's client clearly mask-gates each of the 7 individually, which does **not** match either given reference exactly; flagging as a genuine third data point rather than forcing a match).

**VERDICT: MATCHES pre-95 shape by member COUNT (7, confirmed via the `while(v86<7)` loop bound and raw bit range 115–121) and by 5-of-7 individual block sizes exactly matching the pre-95 list's value set {13,15,15,17,20} (with a 6th slot at 20 also found, one short of the full 7-slot enumeration and the 7th slot unresolved) — but DIFFERS from the pre-95 reference's implied "single unconditional 110-byte trailer" in that v92 mask-gates each of the 7 members individually (confirmed from the decompiled loop). UNVERIFIED: exact member order, the 7th slot's size/identity, and specifically which slot is GuidedBullet.**
