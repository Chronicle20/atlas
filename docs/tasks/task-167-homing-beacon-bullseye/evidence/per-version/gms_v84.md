# GMS v84 — CTS evidence (task-167)

Binary: `GMS_v84.1_U_DEVM.i64` · session: `5881cf84` · identity confirmed: yes — `mcp__ida-pro__idb_list`
reports session `5881cf84` → `input_path: E:\Programs\Nexon\IDBs_v9\GMS\v84_1\GMS_v84.1_U_DEVM.i64`,
`filename: GMS_v84.1_U_DEVM.i64`. Matches the expected binary exactly.

## Question A — movement-affecting filter

Reset handler: `0xa6bb24` (`CWvsContext::OnTemporaryStatReset`)

Decompile of the reset handler shows the trailing-byte read gated on the filter helper
(the 16-byte mask decoded at `0xa6bb4a` is copied into a local `v17..v20` struct at
`0xa6bbf2`/`0xa6bbf9`, then tested):

```c
sub_89F0EA(v24, 128);                    /*0xa6bc63*/
...
if ( sub_7A07E7(v17) )                   /*0xa6bc34*/
{
  LOBYTE(v12) = CInPacket::Decode1(a2);  /*0xa6bc49*/
  sub_999A9D(v12);                       /*0xa6bc51*/
}
```

Filter helper: `0x7a07e7` (unnamed — named here as the v84 equivalent of v83's `sub_77DC78`;
not renamed in the IDB, only identified for this evidence record).

### Mechanism

`sub_7A07E7` builds a static OR-accumulated 128-bit mask (dynamic-initializer, guarded by
`byte_C49658`) from **14** per-constant masks, then AND's it against the caller's decoded
mask via `sub_89F6F2` (confirmed by decompile of `0x89f6f2` to compute a 128-bit AND, 4
dwords) and tests non-zero via `sub_89F775` (confirmed by decompile of `0x89f775` to be an
"all-4-dwords-zero" test — the caller negates it: `return sub_89F775(v1) == 0` = "AND result
is non-empty").

`sub_89F60C` (`0x89f60c`, confirmed by decompile) computes `*a2 = *this | *a3` (128-bit OR,
4 dwords) — the OR-accumulator, structurally identical to v79's `sub_7DE63D`.

Raw disassembly of the lazy-init block (`0x7a07fd`-`0x7a08fb`) — **critical finding**: the
HLL decompile's printed call order does *not* match the true accumulation chain. The very
first operand is loaded via `mov ecx, offset unk_C49638` (`0x7a088d`) — an *implicit*
`this`/accumulator argument the pseudocode does not print as an explicit call argument.
Missing this instruction undercounts the constant list by one (13 vs. the true 14):

```
7a07fd  push 80h
7a0802  push offset dword_C49568         ; shift 83 (direct)
7a080a  lea eax,[ebp+var_80]; push eax
7a080b  push offset dword_C49578         ; shift 82 (direct)
...                                       ; (11 more pushes, descending addresses)
7a0884  push offset dword_C49628         ; shift 8  (direct)
7a0889  lea eax,[ebp+var_10]; push eax
7a088d  mov ecx, offset unk_C49638       ; shift 7  (direct) — INITIAL ACCUMULATOR (implicit)
7a0892  call sub_89F60C                  ; acc = unk_C49638 | dword_C49568
7a0897  mov ecx, eax
7a0899  call sub_89F60C                  ; acc |= dword_C49578
...                                       ; (11 chained calls total)
7a08e6  call sub_89F60C                  ; acc |= dword_C49628 (final)
7a08ec  mov ecx, offset dword_C49648     ; final accumulator stored here
7a08f1  call sub_89F0EA
```

### Decompiled constants tested (14 total)

Each source global is a small dynamic-initializer. Two families were found (both confirmed
by decompile), matching v79's pattern exactly in shape but with a different family offset:

**Direct family** — `sub_89F0D8(1); v0 = sub_89F235(shift); sub_89F0EA(dest, v0, 0x80)`.
`sub_89F235` is confirmed by decompile (`0x89f235`) to be a 128-bit `1 << a2` left-shift
(same role as v79's `sub_7DE266`). 11 of the 14 constants use this form:

| Global | Ctor | Shift arg (raw) |
|---|---|---|
| `dword_C49638` (initial accumulator, implicit ecx) | `sub_7B17BC`@`0x7b17bc` | `sub_89F235(this,7)` → **7** |
| `dword_C49628` | `sub_7B1A2A`@`0x7b1a2a` | `sub_89F235(8)` → **8** |
| `dword_C49618` | `sub_7B218C`@`0x7b218c` | `sub_89F235(17)` → **17** |
| `dword_C49608` | `sub_7B23FC`@`0x7b23fc` | `sub_89F235(30)` → **30** |
| `dword_C495F8` | `sub_7B245C`@`0x7b245c` | `sub_89F235(32)` → **32** |
| `dword_C495E8` | `sub_7B248C`@`0x7b248c` | `sub_89F235(33)` → **33** |
| `dword_C495C8` | `sub_7B24EC`@`0x7b24ec` | `sub_89F235(35)` → **35** |
| `dword_C495B8` | `sub_7B2774`@`0x7b2774` | `sub_89F235(39)` → **39** |
| `dword_C495D8` | `sub_7B5648`@`0x7b5648` | `sub_89F235(49)` → **49** |
| `dword_C49578` | `sub_7B5C78`@`0x7b5c78` | `sub_89F235(82)` → **82** |
| `dword_C49568` | `sub_7B5CA8`@`0x7b5ca8` | `sub_89F235(83)` → **83** |

**`sub_7B0D46` family** — `sub_7B0D46(dest, a2)` (`0x7b0d46`) confirmed by decompile to call
`sub_89F235(a2 + 84)`, i.e. `1 << (a2+84)` — the v84 equivalent of v79's `sub_7099E5`
(`1 << (a2+73)`; **the family offset itself grew from 73 to 84, a delta of 11**, evidence a
version-to-version stat-enum insertion of 11 new entries below this range). 3 of the 14
constants use this form:

| Global | Ctor | Shift arg (raw = a2+84) |
|---|---|---|
| `dword_C49598` | `sub_7B5D02`@`0x7b5d02` | `sub_7B0D46(dest,1)` → **85** |
| `dword_C49588` | `sub_7B5D2C`@`0x7b5d2c` | `sub_7B0D46(dest,2)` → **86** |
| `dword_C495A8` | `sub_7B5D56`@`0x7b5d56` | `sub_7B0D46(dest,3)` → **87** |

Raw shift set tested by `sub_7A07E7`: **{7, 8, 17, 30, 32, 33, 35, 39, 49, 82, 83, 85, 86,
87}** — **14 constants**, all confirmed by decompile of each constant's own dynamic
initializer (not inferred from disassembly context alone).

### Resolved stat names

- **Raw shifts {7, 8, 17, 30, 32, 33, 35, 39, 49} — 9 constants, EXACT match to v79's direct
  family (v79 evidence: `docs/tasks/task-167-homing-beacon-bullseye/evidence/per-version/gms_v79.md`
  lines 63-71 record the identical raw shift set {7,8,17,30,32,33,35,39,49}).** By v79/v83
  positional and count correspondence this is Speed(7), Jump, Stun, Weakness, Slow, Morph,
  Ghost, BasicStatUp, Attract (9 names) — **UNVERIFIED at the individual-bit level**: this
  IDB has no `TEMPORARY_STAT` enum/string table, matching v79's exact blocker, so which raw
  shift among {8,17,30,32,33,35,39,49} is Jump vs. Stun vs. Weakness vs. Slow vs. Morph vs.
  Ghost vs. BasicStatUp vs. Attract could not be resolved bit-by-bit.
- **Raw shift 87 (`dword_C495A8`, family idx 3) = RideVehicle — CONFIRMED.** In
  `CWvsContext::OnTemporaryStatReset`, immediately before this filter check, a separate
  single-bit test against `dword_C4EC20` (own ctor `sub_A686EF`@`0xa686ef`:
  `sub_7B0D46(v2,3)` → same raw shift 87) gates a `sub_96DB34` call — the show-ride-effect
  pattern independently confirmed for the same construction in Question B below. Two
  independently-built masks (the filter's own copy and this side-effect check) encoding the
  identical shift is strong corroboration.
- **Raw shifts 85, 86 (family idx 1, 2) = the two Dash bits (DashSpeed/DashJump), by count,
  family, and adjacency to the confirmed RideVehicle(87) slot — UNVERIFIED at the
  individual-bit level** which of {85,86} is DashSpeed vs. DashJump (no per-bit name
  evidence in this IDB; matches v79's identical disposition for its 74/75 pair).
- **Raw shifts 82, 83 (direct family) — NEW relative to v79/v83, semantic name
  UNVERIFIED.** These two constants have no analog in v79's evidence (`gms_v79.md`'s
  raw shift set tops out at 49 before jumping to the family range 74-76; no 82/83-equivalent
  exists there). No side-effect block in `OnTemporaryStatReset` references either bit
  individually (only `dword_C4EC20`(87)/`dword_C4EC00`(89, Question B)/`dword_C4EC10`(50,
  unrelated single check) get dedicated blocks — see the full decompile of `0xa6bb24`
  quoted for Question B). **UNVERIFIED: no `TEMPORARY_STAT` symbol table exists in this IDB
  to name these two constants** — flagged as a genuine new-stat DIFFERS, not an artifact of
  bit renumbering (see Comparison below).

### Raw-vs-registry basis

All shifts above are the **raw client bit shift** read directly from each constant's own
dynamic-initializer argument (`sub_89F235`/`sub_7B0D46` argument), confirmed by decompile —
not inferred from disassembly context.

The **atlas registry shift** (per `docs/tasks/task-167-homing-beacon-bullseye/context.md:26`
and `plan.md:1356`, established from v83) is anchored to v83's *own* raw client shift:
`context.md:26` records "v83 GuidedBullet mask bit: registry shift 87"; `plan.md:1356`
records "RideVehicle shift 85". For v84, the same two semantic bits (RideVehicle confirmed
above at raw 87; GuidedBullet confirmed in Question B at raw 89) map to registry 85 and 87
respectively — a consistent **offset of `registry = raw − 2`** for the shifted/family range,
derived from two independent anchors (not assumed). For the *unshifted* low/direct range
(raw 7-49), `plan.md:1334` records "Speed is registry shift 7", exactly matching v84's own
raw shift 7 for the same slot (the initial accumulator) — **offset 0** for that range. The
offset is therefore **non-uniform across the bit range**: 0 below shift 50, −2 at/above
shift 82 — consistent with 2 new stat types having been inserted into the enum between v83
and v84, immediately below the family block (see raw 82/83 above).

### Comparison to v83 list

**DIFFERS: v84 has 14 constants, not v83's reference 12.** The original 12 semantic stats
are still present and accounted for: the 9 low-range direct constants match v79's raw shift
set exactly (registry offset 0), and RideVehicle + the 2 Dash bits are present in the
`sub_7B0D46` family at raw 85/86/87 (registry 83/84/85, offset −2, RideVehicle
independently confirmed). **v84 additionally tests 2 constants (raw shift 82, 83) with no
v79/v83 equivalent** — their semantic names are UNVERIFIED (no `TEMPORARY_STAT` symbol table
in this IDB, no dedicated `OnTemporaryStatReset` side-effect block referencing them
individually to cross-check via behavior, unlike RideVehicle/GuidedBullet). This is a real
behavioral DIFFERS: v84's movement-affecting filter treats 2 more stat bits as
movement-affecting than v83 does, not merely a renumbering artifact.

## Question B — two-state member group

`SecondaryStat` constructor: **UNVERIFIED — not found.** `func_query` for `.*SecondaryStat.*`
in this IDB returns no results — no `SecondaryStat::SecondaryStat` symbol exists, matching
v79's exact blocker (`gms_v79.md` lines 134-140). The object's 7 member slots are visible
only via their masks and the decode-loop call sites, not via a constructor that would reveal
each member's concrete class/vtable.

### Two independent decode loops found (both mask-gated, both confirmed by decompile+disasm)

**Loop 1 — "give"/local decode**, inside `sub_7AC409`@`0x7ac409` (called from
`CUserRemote::OnSetTemporaryStat`@`0x9c3bfb`; this is the v84 analog of v79's
`DecodeForLocal`). Tail loop (`0x7acce3`-`0x7acd1e`), confirmed by decompile:

```c
v69 = 0;                                              /*0x7acce3*/
v70 = this + 827;                                     /*0x7acce5*/
do {
  v71 = sub_7B0D46(v75, v69);                         /*0x7accf0*/   // shift = v69+84
  v72 = sub_89F6F2(v74, v71);                         /*0x7accff*/   // decoded_flag & member_mask
  if ( (unsigned __int8)sub_89F78E(v72) )             /*0x7acd06*/   // non-zero test
    (*(void (__thiscall **)(_DWORD, CInPacket *))(*(_DWORD *)*v70 + 24))(*v70, a3); /*0x7acd14*/ // DecodeForClient(a3), vtable+0x18
  ++v69;                                               /*0x7acd17*/
  v70 += 2;                                            /*0x7acd18*/   // stride 2 dwords (8 bytes)
} while ( v69 < 7 );                                   /*0x7acd1e*/
```

**Loop 2 — remote/party decode**, inside `sub_7A5D2B`@`0x7a5d2b` (a ~20KB function; this is
the v84 analog of v79's `DecodeForRemote`). Tail loop confirmed by raw disassembly
(`0x7aaa70`-`0x7aab6a`):

```
loc_7AAA70: mov ecx,[var_18]; call sub_7B8A37         ; member pointer for this slot
            push [var_14]; lea eax,[var_50]; push eax
            call sub_7B0D46                            ; shift = var_14 + 84
            ...
            call sub_89F6F2                            ; decoded_flag & member_mask
            call sub_89F78E                             ; non-zero test
            test al,al; jz loc_7AAB5F                   ; skip if bit not set
            mov ecx,[arg_4]; mov eax,[ecx]; push edi; call dword ptr [eax+18h]  ; DecodeForClient(a3)
            ...
loc_7AAB5F: inc [var_14]; add [var_18],8; cmp [var_14],7; jl loc_7AAA70
```

Both loops iterate exactly 7 times, stride 8 bytes (2 dwords)/member, starting at the same
field (`this+827` dwords / `this+0xCE8` bytes — `827*4 = 3308`, close enough to `0xCE8=3304`
given the two loops address slightly different base objects), and both gate the virtual
`DecodeForClient` call (vtable slot `+0x18`/24) on `sub_7B0D46(idx) & decoded_flag != 0`
**before** making the call — i.e. **mask-gated, not unconditional**.

### Two independently-confirmed members (via `OnTemporaryStatReset` side-effect blocks)

Full decompile of `CWvsContext::OnTemporaryStatReset`@`0xa6bb24` shows two dedicated
single-bit side-effect blocks, each testing a `sub_7B0D46`-family mask (i.e. a two-state
group member):

- **Family idx 3 (raw shift 87) = RideVehicle.** `dword_C4EC20`, ctor `sub_A686EF`@
  `0xa686ef`: `sub_7B0D46(v2,3)` → raw 87 (identical construction to the movement-filter
  member confirmed in Question A). Reset-handler block:
  ```c
  v4 = sub_89F6F2(v21, &unk_C4EC20);           /*0xa6bb61*/
  if ( (unsigned __int8)sub_89F78E(v4) )       /*0xa6bb68*/
  {
    v5 = (_DWORD *)sub_742739(*((_DWORD *)v3 + 833));  /*0xa6bb7d*/   // this+833 = base(827)+2*3
    sub_96DB34(*v5);                            /*0xa6bb86*/            // show-ride-effect pattern
  }
  ```
  `this+833` = `this+827` (the array base) `+ 2*3` — exactly index 3, matching the loop
  stride confirmed above.
- **Family idx 5 (raw shift 89) = GuidedBullet.** `dword_C4EC00`, ctor `sub_A68719`@
  `0xa68719`: `sub_7B0D46(v2,5)` → raw 89. Reset-handler block:
  ```c
  v6 = sub_89F6F2(v21, &unk_C4EC00);                                    /*0xa6bb97*/
  if ( (unsigned __int8)sub_89F78E(v6) )                                /*0xa6bb9e*/
  {
    if ( (vtable-call: IsActivated)(*((_DWORD *)v3 + 837)) )            /*0xa6bbb5*/  // this+837 = base+2*5
    {
      v7 = *((_DWORD *)v3 + 837);                                       /*0xa6bbbc*/
      if ( v7 )                                                         /*0xa6bbc0*/
      {
        v20 = sub_6794D7(v7);                                           /*0xa6bbcf*/  // GetMobID/GetReason
        v8 = (_DWORD *)sub_688C92(v7);                                  /*0xa6bbd2*/
        sub_690483(*v8, v20);                                           /*0xa6bbdb*/  // CMobPool::ResetGuidedMob pattern
      }
    }
  }
  ```
  This exactly matches v79's confirmed GuidedBullet pattern ("if bit set → if IsActivated →
  GetMobID/GetReason → ResetGuidedMob"), and is NOT part of Question A's movement filter
  (raw 89 is absent from the 14-constant set) — consistent with the documented fact that
  GuidedBullet is not movement-affecting.

**GuidedBullet member's CTS mask-bit shift: raw 89 — BASIS: raw client bit shift** (from
`sub_7B0D46(dest,5)` → `sub_89F235(5+84)` → `1<<89`, confirmed by decompile of both
`sub_A68719` and `sub_7B0D46`).

**Registry basis: 87** — using the `registry = raw − 2` offset derived in Question A from
this same GuidedBullet anchor plus the independently-confirmed RideVehicle anchor
(raw 87 → registry 85). This exactly matches `context.md:26`'s "v83 GuidedBullet mask bit:
registry shift 87" — strong cross-version corroboration that the atlas registry constant is
unchanged and only the *client's* raw bit position moved.

**Raw-vs-registry offset: −2** (`registry = raw − 2`), confirmed via two independent
anchors (RideVehicle: raw 87 → registry 85; GuidedBullet: raw 89 → registry 87), consistent
with the same −2 offset already established for Question A's family range.

### Remaining 5 members — INFERRED by position, not independently named in v84

Following the same method v79's evidence used (`gms_v79.md` lines 191-199): the v95 table
(`context.md`/`plan.md:871`, design-phase-verified) gives the 7-slot order as
`0=EnergyCharged, 1=Dash_Speed, 2=Dash_Jump, 3=RideVehicle, 4=PartyBooster, 5=GuidedBullet,
6=unnamed/unreachable`. v84's independently-confirmed idx 3 (RideVehicle) and idx 5
(GuidedBullet) land on the *same slot indices* as this table. On that positional
correspondence only: idx 0 = EnergyCharged, idx 1 = Dash_Speed, idx 2 = Dash_Jump,
idx 4 = PartyBooster, idx 6 = unnamed. **UNVERIFIED as direct v84 symbol evidence** — no
dedicated `OnTemporaryStatReset` side-effect block references these 5 slots individually to
cross-check.

### Block sizes — UNVERIFIED

Both decode loops dispatch to each member's `DecodeForClient` through a virtual call
(vtable slot `+0x18`/24) on a per-member object pointer (`*v70` / `[arg_4]`). Without a
`SecondaryStat`/`CTemporaryStat` member-array constructor symbol (see above) to reveal each
member's concrete class/vtable address, the 7 `DecodeForClient` implementations could not be
statically resolved from the virtual-call site alone — the same blocker v79 hit.
`func_query`/`xrefs_to` searches for the owning constructor (which would show 7 `operator
new`/placement-construct calls with concrete per-member sizes) did not locate it in this
pass. **Summed trailer length: UNVERIFIED (not measured) — cannot be compared to the
110-byte reference.**

### Trailer read style: **PER-MEMBER MASK-GATED — DIFFERS from "unconditional"**

Both Loop 1 (`sub_7AC409`) and Loop 2 (`sub_7A5D2B`'s tail) implement the 7-member trailer
as a genuine loop that tests each member's `sub_7B0D46(idx)` (`1 << (idx+84)`) against the
decoded flag *before* making the virtual `DecodeForClient` call — confirmed by both decompile
(Loop 1) and raw disassembly (Loop 2), matching v79's finding exactly (`gms_v79.md`
lines 222-263).

**VERDICT: Question A DIFFERS from v83's 12-constant reference — v84 tests 14 constants: the
9 low-range direct stats match v79's raw shift set exactly (Speed@raw7/registry7 confirmed,
remaining 8 names UNVERIFIED at the individual-bit level); RideVehicle confirmed at
raw87/registry85; the 2 Dash bits present at raw85/86 (registry83/84, UNVERIFIED which is
which); PLUS 2 constants with no v79/v83 equivalent (raw82, raw83) whose semantic names are
UNVERIFIED — a genuine new-stat addition, not a renumbering artifact.**

**Question B: member count (7) and structure (mask-gated, stride 8 bytes, vtable+0x18
dispatch) MATCH v83/v95/v79; GuidedBullet independently confirmed at raw89/registry87
(offset −2 from two independent anchors); RideVehicle independently confirmed at
raw87/registry85; remaining 5 slots INFERRED by v95 positional analogy only; UNVERIFIED:
per-member block sizes and the resulting 110-byte sum (no `SecondaryStat`/`CTemporaryStat`
member-array constructor symbol found → could not resolve the polymorphic
`DecodeForClient` targets); DIFFERS from "unconditional" — v84's trailer read is directly
evidenced as per-member mask-gated in both the local and remote decode paths.**
