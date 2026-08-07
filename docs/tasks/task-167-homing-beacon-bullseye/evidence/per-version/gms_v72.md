# GMS v72 — CTS evidence (task-167)

Binary: `GMS_v72.1_U_DEVM.exe.i64`  ·  session: c8acae95
Confirmed identity: yes — `idb_list` shows session `c8acae95` → `input_path`
`E:\Programs\Nexon\IDBs_v9\GMS\v72\GMS_v72.1_U_DEVM.exe.i64`, filename
`GMS_v72.1_U_DEVM.exe.i64`, matching the expected binary exactly.

## Question A — movement-affecting filter

Reset handler: `0x918f3c` (`CWvsContext::OnTemporaryStatReset`, pre-verified anchor).

Decompile of `0x918f3c` shows the mask decoded via `CInPacket::DecodeBuffer(v28, 16)`
then passed to the filter:
```
UINT128::UINT128((UINT128 *)&v20, (const struct UINT128 *)v28, 0x80u); /*0x919047*/
if ( SecondaryStat::IsMovementAffectingStat() )                        /*0x91904c*/
{
  v15 = CInPacket::Decode1(a2);                                         /*0x919061*/
  CUserLocal::SetSecondaryStatChangedPoint(v14, v15);                   /*0x919069*/
}
```
Disasm confirms the call is by-value (`UINT128` copy-ctor builds the 16-byte
arg directly on the stack at `0x919047`, `call` at `0x91904c`, caller cleans
up `add esp,10h` at `0x919051` — matches the mangled name
`?IsMovementAffectingStat@SecondaryStat@@SAHVUINT128@@@Z`, static, takes
`UINT128` by value).

Filter helper: `0x6c87b6` (`SecondaryStat::IsMovementAffectingStat`, already named
in this IDB — v83-equivalent of `sub_77DC78`).

### Decompiled constants tested

The helper lazily builds a combined "movement-affecting" mask once
(guard `byte_AA7298`) by OR-chaining **12** per-stat UINT128 constants, then
tests `(incoming_mask & combined) != 0`:

```
mov ecx, offset unk_AA7278                    /*0x6c8844 — this = AA7278 (no OR, chain seed)*/
call SecondaryStat__TwoStateBitMask (ecx=this) /*0x6c8849*/
... (10 more chained calls each OR-ing one more constant) ...
call SecondaryStat__TwoStateBitMask            /*0x6c888f — last: OR unk_AA71C8*/
push eax
mov ecx, offset unk_AA7288
call UINT128::UINT128(UINT128 const&, uint)    /*0x6c889a — cache combined mask*/
...
push offset unk_AA7288
lea eax, [ebp+var_10]
push eax
lea ecx, [ebp+arg_0]                            /*arg_0 = incoming mask, by value*/
call UINT128::operator&(UINT128 const &)        /*0x6c88b6*/
mov ecx, eax
call UINT128::operator!(void)                   /*0x6c88bd*/
neg al / sbb eax,eax / inc eax                  /*bool result*/
```

The 12 constant addresses (in push/chain order) and their bit shifts, each
resolved by decompiling the tiny per-constant initializer that the address
xrefs to (each does `1 << shift`, either directly via `sub_7998DE(shift)`
["`UINT128(1)<<shift`" idiom] or via the two-state helper
`SecondaryStat__TwoStateBitMask(a1, idx)` = `1 << (idx + 67)`):

| global | init fn | shift expr | **bit** |
|---|---|---|---|
| `unk_AA7278` | `0x6d7b72` | `SecondaryStat__TwoStateBitMask(0)`→7998DE(7) direct: `sub_7998DE(v0,7)` | **7** |
| `unk_AA7268` | `0x6d7ba2` | `sub_7998DE(v0,8)` | **8** |
| `unk_AA7258` | `0x6d7d52` | `sub_7998DE(v0,17)` | **17** |
| `unk_AA7248` | `0x6d7fc2` | `sub_7998DE(v0,30)` | **30** |
| `unk_AA7238` | `0x6d8022` | `sub_7998DE(v0,32)` | **32** |
| `unk_AA7228` | `0x6d8052` | `sub_7998DE(v0,33)` | **33** |
| `unk_AA7208` | `0x6d80b2` | `sub_7998DE(v0,35)` | **35** |
| `unk_AA71F8` | `0x6d8172` | `sub_7998DE(39)` (direct, no `+67` wrapper) | **39** |
| `unk_AA7218` | `0x6d8352` | `sub_7998DE(v0,49)` | **49** |
| `unk_AA71D8` | `0x6d85f2` | `SecondaryStat__TwoStateBitMask(idx=1)` → `sub_7998DE(a2+67)` = `1+67` | **68** |
| `unk_AA71C8` | `0x6d861c` | `SecondaryStat__TwoStateBitMask(idx=2)` = `2+67` | **69** |
| `unk_AA71E8` | `0x6d8646` | `SecondaryStat__TwoStateBitMask(idx=3)` = `3+67` | **70** |

Note on `sub_7998DE`: decompiled as `UINT128 *__thiscall(UINT128 *this, signed int a2)`,
body is `this << a2` bit-for-bit (guards `a2>0x7F` → zero), i.e. it is
`UINT128::operator<<`. The 9 "direct" constants call it with `this=1`
pre-built (`UINT128::UINT128(v,1u)` then `sub_7998DE(v,shift)`), the 3
"two-state" constants go through `SecondaryStat__TwoStateBitMask` (renamed
from `sub_6D79EC`), which does the identical `1 << (a2+67)` — this is the
**same +67 base** independently confirmed in `SecondaryStat::Reset` (0x6ca91a,
tail 7-iteration loop `sub_6D79EC(v71, v65)` for `v65=0..6`) and in both
`DecodeForLocal`/`DecodeForRemote`'s two-state trailer loops (Question B).

### Resolved stat identity

- Bit **7 / 8 / 17** = **Speed / Jump / Stun** — high confidence: these are
  the first 9-18 fixed slots of the CTS layout (WeaponAttack..Hands, then
  Speed=7, Jump=8, ..., Stun=17), foundational stats present in every known
  GMS client era; v72's own extraction lands on exactly 7/8/17, matching.
- Bits **68/69/70** = **DashSpeed / DashJump / RideVehicle** (in that order)
  — confirmed two independent ways: (1) `TemporaryStat_GuidedBullet`'s
  pointer offset in `OnTemporaryStatReset` (`v3+2724`) lands on two-state
  array slot **5** (`2680 + 8*5 + 4`), and (2) the per-member block-size
  order derived in Question B (15/15/15/13/20/17/15) exactly matches v83's
  named order EnergyCharge(0)/DashSpeed(1)/DashJump(2)/RideVehicle(3)/
  SpeedInfusion(4)/GuidedBullet(5)/Undead(6) — placing DashSpeed=67+1=68,
  DashJump=67+2=69, RideVehicle=67+3=70.
- Bits **30, 32, 33, 35, 39, 49** — **UNVERIFIED name**. v72's IDB has no
  `CTS_*`/`dynamic_initializer_for_*` symbol names (`func_query`/`list_globals`
  regex sweep for `CTS_`, `dynamic_initializer`, `TemporaryStat_*` returned
  nothing beyond the already-named `TemporaryStat_GuidedBullet`), unlike the
  v95 IDB (which task-086's evidence says *does* carry those names). A naive
  "same index number = same name as atlas's v83-derived registry" lookup
  (`libs/atlas-packet/model/character_temporary_stat.go`) was spot-checked
  and falsified: registry index 35 = `MapleWarrior` (`NoOpForeignValueWriter`,
  i.e. expected **zero** extra decode bytes), but v72's bit-35 site
  (`0x6cbf40`, guarded by `unk_AA7208`) decodes `Decode2()` (2B) then
  `Decode4()` (4B) — not the NoOp shape. Since v72's two-state group starts
  15 bits earlier than v83's (67 vs 82 — see Question B), v72 evidently has
  ~15 fewer main-sequence stats than v83, and the missing ones are not
  necessarily a contiguous suffix, so index-for-index name lookup against
  the v83-tuned registry is unsafe. Not resolved further within this pass.

### Comparison to v83 list

**Structural count MATCHES**: 12 constants total, split 9 main-sequence +
3 two-state — identical to the v83 breakdown (Speed, Jump, Stun, Weakness,
Slow, Morph, Ghost, BasicStatUp, Attract = 9; RideVehicle, DashSpeed,
DashJump = 3).

**Absolute bit positions DIFFER** for the two-state trio: v72 has
DashSpeed/DashJump/RideVehicle at bits 68/69/70 (two-state group starts at
bit 67), vs v83's registry-documented 83/84/85 (group starts at bit 82).
This is expected/mechanical — v72 is an earlier client with fewer
main-sequence CTS slots before the two-state block — not a behavioral
divergence.

**Name-for-name identity of the 6 middle main-sequence bits is UNVERIFIED**
(see above) — cannot confirm or deny they are Weakness/Slow/Morph/Ghost/
BasicStatUp/Attract specifically, only that 9 main-sequence bits are tested,
matching the count.

## Question B — two-state member group

SecondaryStat constructor: `0x6c70e9` (renamed `SecondaryStat__ctor` in this
pass). Confirmed via:
```
`eh vector constructor iterator'((void *)(this + 2680), 8u, 7, sub_6D89F7, sub_6D89E1); /*0x6c7113*/
for ( i = 0; i < 7; ++i ) { ... switch(v2) { case 0: ...; case 1/2: ...; case 3: ...; case 4: ...; case 5: ...; case 6: goto LABEL_20(=case1/2 path); } }
```
— an `eh vector constructor iterator` over 7 elements of stride 8 bytes
starting at offset **2680 (0xA78)**, immediately followed by a 7-way
per-index placement-construct switch. This offset (2680) and stride (8)
match the array base seen in both `DecodeForLocal`'s (`lea eax,[esi+0A78h]`)
and `DecodeForRemote`'s (`add esi, 0A7Ch`) trailer loops.

### Ordered members (name, DecodeForClient address, block size)

Each member's concrete vtable was read from its ctor's final `*this = &off_X`
assignment (multi-inheritance vtable-fixup pattern); vtable slot `+0x18` is
the `DecodeForClient`-equivalent virtual (confirmed by its use at
`CInPacket::Decode…`-calling sites in `DecodeForLocal`/`DecodeForRemote`).
All 7 members share one common base decoder,
`SecondaryStat__TwoStateBase__DecodeForClient` (renamed from `sub_6D87D4`):
`CInPacket::DecodeBuffer(this+3,4)` (4B) + `CInPacket::DecodeBuffer(this+4,4)`
(4B) + `sub_6C6B28` (`CInPacket::Decode1`+`CInPacket::Decode4` = 1B+4B) =
**13 bytes base**, common to every member.

| # | member (identity basis) | ctor | vtable | DecodeForClient | extra bytes | **block size** |
|---|---|---|---|---|---|---|
| 0 | EnergyCharge (order/size match) | `SecondaryStat__EnergyCharge__ctor` (0x6c73f8) | 0x9d4bdc | `SecondaryStat__EnergyCharge__DecodeForClient` (0x6d8ad5) | base(13) + `Decode2`(2) | **15** |
| 1 | DashSpeed (order/size + bit-68 filter match) | `SecondaryStat__DashDashUndead__ctor` (0x6c74b3) | 0x9d4c7c | `SecondaryStat__DashDashUndead__DecodeForClient` (0x6d8c50) | base(13) + `Decode2`(2) | **15** |
| 2 | DashJump (order/size + bit-69 filter match) | same as #1 | same as #1 | same as #1 | base(13) + `Decode2`(2) | **15** |
| 3 | RideVehicle/MonsterRiding (pointer-offset + bit-70 filter match) | `SecondaryStat__RideVehicle__ctor` (0x6c73a0) | 0x9d4ba0 | `SecondaryStat__RideVehicle__DecodeForClient` (0x6d897d) | base(13) + 0 | **13** |
| 4 | SpeedInfusion (order/size match) | `SecondaryStat__SpeedInfusion__ctor` (0x6c748c) | 0x9d4c40 | `SecondaryStat__SpeedInfusion__DecodeForClient` (0x6d870a) | base(13) + `sub_6C6B28`(5: Decode1+Decode4) + `Decode2`(2) | **20** |
| 5 | GuidedBullet/HomingBeacon (pointer-offset-confirmed, `TemporaryStat_GuidedBullet` at `v3+2724` = slot 5) | reuses `SecondaryStat__RideVehicle__ctor`, then vtable stomped to 0x9d4b4c (case 5: `*v4=&off_9D4B4C; v4[8]=&off_9D4B30`) | 0x9d4b4c | `SecondaryStat__GuidedBullet__DecodeForClient` (0x6c72ea) | calls `SecondaryStat__RideVehicle__DecodeForClient` (=base only, 13) + `Decode4`(4) | **17** |
| 6 | Undead (order/size match; ctor `case 6: goto LABEL_20` = same path as #1/#2) | same as #1 | same as #1 | same as #1 | base(13) + `Decode2`(2) | **15** |

Summed trailer length: **15+15+15+13+20+17+15 = 110 bytes** (v83 reference = 110 — **MATCH**).

GuidedBullet mask-bit shift: **72** (`67 + 5`, slot index 5 of the 7-element
two-state array) — confirmed by two independent methods: (a) pointer
arithmetic (`TemporaryStat_GuidedBullet` accessed at `v3+2724` in
`OnTemporaryStatReset` = `2680 + 8*5 + 4`, the object-pointer half of
slot 5's 8-byte `ZRef`), and (b) block-size-order match against v83's
named 15/15/15/13/20/17/15 sequence, where index 5 is GuidedBullet (17B).

### Trailer read style: **per-member mask-gated** — DIFFERS from the v83 "unconditional" baseline

Both `SecondaryStat::DecodeForLocal` (0x6cb87b) and `SecondaryStat::DecodeForRemote`
(0x6cfe78) implement the trailer as a literal `for(i=0;i<7;i++)` loop that
computes `1<<(i+67)` via `SecondaryStat__TwoStateBitMask` **every iteration**
and explicitly gates the virtual `DecodeForClient` call on it:

`DecodeForLocal` (0x6cef89–0x6cf083):
```
call SecondaryStat__TwoStateBitMask   /*0x6cef9b — mask = 1<<(i+67)*/
call UINT128::operator&(UINT128 const &)   /*0x6cefaa*/
call UINT128::compareTo(ulong)             /*0x6cefb1*/
test al, al
jz loc_6CF078                              /*skip virtual call if bit unset*/
mov ecx, [ebp+arg_4]
mov eax, [ecx]
call dword ptr [eax+18h]                   /*0x6cefc4 — DecodeForClient, GATED*/
...
loc_6CF078: inc [ebp+var_14]; cmp [ebp+var_14],7; jl loc_6CEF89   /*loop back*/
```

`DecodeForRemote` (0x6d0588–0x6d05bb) is structurally identical:
```
call SecondaryStat__TwoStateBitMask    /*0x6d058d*/
call UINT128::operator&
call UINT128::compareTo
test al, al
jz loc_6D05B4
mov ecx, [esi]; mov eax, [ecx]
call dword ptr [eax+18h]               /*0x6d05b1 — DecodeForClient, GATED*/
loc_6D05B4: inc ebx; add esi,8; cmp ebx,7; jl loc_6D0588
```

Both functions skip a member's `DecodeForClient` call entirely when its bit
is not set in the packet's decoded mask — this is **per-member mask-gated**,
not the "every member's block always read" unconditional pattern the v83
baseline describes.

**VERDICT: block sizes/count MATCH v83 (7 members / 110 bytes, same
15/15/15/13/20/17/15 shape and same relative member order) — but trailer
READ STYLE DIFFERS: v72 is per-member mask-gated in both `DecodeForLocal`
and `DecodeForRemote` (bit test + conditional virtual-call skip), not
unconditional. Question A: bit positions and structural count (9
main-sequence + 3 two-state = 12) MATCH v83; absolute two-state bit
positions differ mechanically (67-based vs v83's 82-based, from v72 having
~15 fewer main-sequence CTS slots); 6 of 9 main-sequence stat NAMES are
UNVERIFIED — v72's IDB carries no CTS_*/dynamic-initializer symbol names,
and a wire-shape spot-check against the v83-derived atlas registry falsified
the naive same-index assumption at bit 35.**
