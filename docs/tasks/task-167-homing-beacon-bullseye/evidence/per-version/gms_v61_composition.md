# GMS v61 — two-state group FIELD COMPOSITION (task-167)

Binary: `GMS_v61.1_U_DEVM.exe.i64` · session: `415bf585` · identity confirmed via `mcp__ida-pro__idb_list`
(returned `filename: "GMS_v61.1_U_DEVM.exe.i64"`, exact match).

## Method

Built directly on the already-established (two-independent-pass) findings in `gms_v61_recheck.md` /
`gms_v61.md`: 6 members, sizes 14/14/14/12/18/16=88, GuidedBullet=i5/shift64, per-member mask-gated reads.
This pass decompiles each member's `DecodeForClient` override (all statically resolved via literal
vtable-pointer stores in the constructor, per the prior pass) to record the ordered field reads, not
just the totals. I then additionally scoped `CInPacket::Decode1` (`0x424516`) call sites across the
whole `SecondaryStat::DecodeForLocal` body (`insn_query`, `op0=0x424516`, `func=0x663665`) to test
the "5-byte-vs-4-byte time field" half of the hypothesis directly, rather than inferring it from byte
totals alone, and inspected three ordinary (non-two-state) stat blocks at the top of the same function
for comparison.

## Per-member field tables

### Shared base — `sub_66E9B6` (called by every one of the 6, directly or via wrapper `sub_66EB3D`)

```c
int __thiscall sub_66E9B6(_DWORD *this, CInPacket *a2)
{
  ...
  CInPacket::DecodeBuffer((int)a2, this + 1, 4u);   /*0x66e9df*/
  CInPacket::DecodeBuffer((int)a2, this + 2, 4u);   /*0x66e9ed*/
  result = CInPacket::Decode4(a2);                  /*0x66e9f5*/
  this[3] = result;                                 /*0x66e9fc*/
  ...
}
```

| field | width | evidence |
|---|---|---|
| field A (`this+1`, byte offset +4) | 4 | `CInPacket::DecodeBuffer((int)a2, this + 1, 4u); /*0x66e9df*/` |
| field B (`this+2`, byte offset +8) | 4 | `CInPacket::DecodeBuffer((int)a2, this + 2, 4u); /*0x66e9ed*/` |
| field C (`this[3]`, byte offset +12) | 4 | `result = CInPacket::Decode4(a2); /*0x66e9f5*/ this[3] = result; /*0x66e9fc*/` |
| **running total** | **12** | matches known base size (12) exactly — 4+4+4 |

No `CInPacket::Decode1` call anywhere in this function, and no combination with a local clock value
(unlike the ordinary-stat pattern below) — field C is a plain 4-byte value, not evidenced to be "time"
in any sense beyond being a bare `Decode4`.

### i=0 EnergyCharge / i=1,2 DashSpeed,DashJump (positional) — `sub_66EC19` / `sub_66ED94` (byte-identical shape)

```c
__int16 __thiscall sub_66EC19(int this, CInPacket *a2)
{
  ...
  sub_66E9B6((_DWORD *)this, a2);        /*0x66ec3e*/
  result = CInPacket::Decode2(a2);       /*0x66ec46*/
  *(_WORD *)(this + 36) = result;        /*0x66ec4d*/
  ...
}
```
(`sub_66ED94` is byte-identical, only the stored offset differs, `this+36` in both — confirms i=1,i=2
truly share one function, per the prior pass.)

| field | width | evidence | running total |
|---|---|---|---|
| base (A+B+C) | 12 | `sub_66E9B6((_DWORD *)this, a2); /*0x66ec3e*/` | 12 |
| extra int16 @ `this+36` | 2 | `result = CInPacket::Decode2(a2); /*0x66ec46*/ *(_WORD *)(this + 36) = result; /*0x66ec4d*/` | **14** |

Matches known size (14) exactly. No `Decode1`/time pattern.

### i=3 RideVehicle — `sub_66EB3D` (base only, no override fields)

```c
int __thiscall sub_66EB3D(_DWORD *this, int a2)
{
  ...
  result = sub_66E9B6(a2);   /*0x66eb62*/
  ...
}
```

| field | width | evidence | running total |
|---|---|---|---|
| base (A+B+C) | 12 | `result = sub_66E9B6(a2); /*0x66eb62*/` (no other packet call in this wrapper) | **12** |

Matches known size (12) exactly. RideVehicle adds zero fields on top of base.

### i=4 SpeedInfusion — `sub_66E8EF`

```c
__int16 __thiscall sub_66E8EF(int this, CInPacket *a2)
{
  ...
  sub_66E9B6((_DWORD *)this, a2);                     /*0x66e914*/
  *(_DWORD *)(this + 32) = CInPacket::Decode4(a2);     /*0x66e924*/
  result = CInPacket::Decode2(a2);                     /*0x66e927*/
  *(_WORD *)(this + 40) = result;                      /*0x66e92e*/
  ...
}
```

| field | width | evidence | running total |
|---|---|---|---|
| base (A+B+C) | 12 | `sub_66E9B6((_DWORD *)this, a2); /*0x66e914*/` | 12 |
| extra dword @ `this+32` | 4 | `*(_DWORD *)(this + 32) = CInPacket::Decode4(a2); /*0x66e924*/` | 16 |
| extra int16 @ `this+40` | 2 | `result = CInPacket::Decode2(a2); /*0x66e927*/ *(_WORD *)(this + 40) = result; /*0x66e92e*/` | **18** |

Matches known size (18) exactly. The two extra fields are a **plain Decode4 + a plain Decode2** — two
separate ordinary fields, **not** a single 5-byte (or 4-byte) compound "time" field. There is no
`Decode1` call in this function.

### i=5 GuidedBullet — `sub_65F840` (wraps `sub_66EB3D`, i.e. base via RideVehicle's wrapper)

```c
int __thiscall sub_65F840(_DWORD *this, CInPacket *a2)
{
  ...
  sub_66EB3D(this, (int)a2);   /*0x65f865*/
  result = CInPacket::Decode4(a2);  /*0x65f86d*/
  this[7] = result;                 /*0x65f874*/
  ...
}
```

| field | width | evidence | running total |
|---|---|---|---|
| base (via `sub_66EB3D` → `sub_66E9B6`) | 12 | `sub_66EB3D(this, (int)a2); /*0x65f865*/` | 12 |
| `dwMobId` @ `this[7]` (byte offset +28) | 4 | `result = CInPacket::Decode4(a2); /*0x65f86d*/ this[7] = result; /*0x65f874*/` | **16** |

Matches known size (16) exactly. `this[7]` at offset +28 is exactly `TemporaryStat_GuidedBullet`'s
`dwMobId`, consistent with the prior pass's independent confirmation via `TemporaryStat_GuidedBullet::GetMobID`.

## Summary table (all 6, in slot order)

| i | member (confidence) | fields (order) | total | matches known size? |
|---|---|---|---|---|
| 0 | EnergyCharge (positional) | base(12) + int16(2) | 14 | yes |
| 1 | DashSpeed (positional, Dash pair) | base(12) + int16(2) | 14 | yes |
| 2 | DashJump (positional, Dash pair) | base(12) + int16(2) | 14 | yes |
| 3 | RideVehicle (independently confirmed) | base(12) only | 12 | yes |
| 4 | SpeedInfusion (positional) | base(12) + dword(4) + int16(2) | 18 | yes |
| 5 | GuidedBullet (independently confirmed) | base(12) + dwMobId dword(4) | 16 | yes |

Sum: 14+14+14+12+18+16 = **88 bytes**, matching the prior pass.

## The base-block composition

`sub_66E9B6` = **3 plain 4-byte fields** (`DecodeBuffer(4)`, `DecodeBuffer(4)`, `Decode4()`→int) = **12
bytes**, with **zero `CInPacket::Decode1` calls** and **zero combination with a local clock value**
anywhere in the function. There is no evidence this function reads a "time" field in any sense distinct
from its other two fields — all three are structurally identical 4-byte reads.

## The time-field-width finding

**Directly tested, not inferred.** I scoped every `CInPacket::Decode1` (`0x424516`) call site inside
`SecondaryStat::DecodeForLocal` (`0x663665`, size `0x3871`, i.e. the whole function body including the
ordinary-stat section, the two standalone flag bytes, and the two-state trailer loop) via
`insn_query(mnem="call", op0=0x424516, func=0x663665)`:

```
matches: [{"addr":"0x666d48", ...}, {"addr":"0x666d58", ...}]
```

**Only 2 hits in the entire function, and neither is inside any of the 6 two-state `DecodeForClient`
overrides** (all 6 were individually decompiled above/previously; none contain a `Decode1` call). Both
hits sit immediately *before* the two-state trailer loop starts (`0x666d66`: `and [ebp+var_14],0`, the
loop's `i=0` init):

```
666d46  mov ecx, edi
666d48  call ?Decode1@CInPacket@@QAEEXZ    ; CInPacket::Decode1(void)
666d4d  mov dl, al
666d4f  mov ecx, esi
666d51  call sub_667A49                    ; setter, stores at this+2208/2212
666d56  mov ecx, edi
666d58  call ?Decode1@CInPacket@@QAEEXZ    ; CInPacket::Decode1(void)
666d5d  mov dl, al
666d5f  mov ecx, esi
666d61  call sub_667AC1                    ; setter, stores at this+2252/2256
666d66  and [ebp+var_14], 0                ; two-state loop i=0 starts here
```

These are **two standalone single-byte flags**, decoded once each (not per-member, not looped), and
stored well outside the two-state array's own storage (`this[598..603]`, byte offsets 0x958-0x96C) — at
byte offsets 2208-2212 and 2252-2256 instead. They are **not part of any individual two-state member's
block** and not evidence of a "time" field inside the group.

**Conclusion: there is no byte-prefixed ("Decode1 + Decode4", i.e. 5-byte) time pattern anywhere in
v61's two-state group** — not in the shared base, not in any of the 6 per-member overrides. The base's
third field is a bare `Decode4` (4 bytes) with no clock involvement; SpeedInfusion's extra `Decode4`
(offset +32) is likewise a bare 4-byte value with no clock involvement. If "the time field" is meant to
denote *any* wire-decoded duration/timestamp-shaped value in this group, **no such field exists at
all** in v61's two-state group — the hypothesis's premise (a time field that "shrank" from 5 to 4 bytes)
does not match the mechanism found, even though the base's raw byte count (12) happens to arithmetically
equal 13-1.

### Comparison to non-two-state (ordinary) stat blocks

For contrast, I decompiled/disassembled the first three ordinary (non-two-state) stat blocks at the top
of the same `DecodeForLocal` function (offsets `esi+0Ch..+2Ch`, `esi+3Ch..+5Ch`, `esi+6Ch..+8Ch`). All
three follow an identical pattern:

```
663690  call ds:__imp_timeGetTime           ; local clock snapshot -> ebx (once, function-level, not per-field)
...
6636bd  call ?Decode2@CInPacket@@QAEGXZ      ; value, 2 bytes                     -> [esi+0Ch]
6636d2  call ?Decode4@CInPacket@@QAEKXZ      ; reason, 4 bytes                    -> [esi+14h]
6636e9  call ?Decode4@CInPacket@@QAEKXZ      ; duration, 4 bytes
6636ee  lea ecx, [ebx+eax]                   ; duration + local-now = absolute deadline -> [esi+20h]
```

i.e. each active ordinary stat = `Decode2`(value,2) + `Decode4`(reason,4) + `Decode4`(duration,4) = **10
bytes**, where the third field IS combined with a local `timeGetTime()` snapshot to compute an absolute
expiry — this is the closest thing to a "time field" in this whole function, and **it too is a plain
4-byte `Decode4`, never preceded by a `Decode1` byte**. This pattern repeats identically for at least the
first 3 ordinary stats (quoted above) and is structurally distinct from the two-state base (which never
touches `timeGetTime`).

**Answer to "does the 5-vs-4-byte time difference also appear in non-two-state blocks":** There is no
5-byte (byte-prefixed) time encoding anywhere in v61's `SecondaryStat::DecodeForLocal` — two-state or
ordinary. Every duration/value-shaped wire field found, in both the two-state group and the ordinary
stats, is a bare 2- or 4-byte `Decode2`/`Decode4`, with `Decode1` used only for the two unrelated
standalone flags noted above. **Confidence: high** for this specific claim (directly evidenced by an
exhaustive `Decode1` call-site scope over the entire function, not a sample) — this reads as a
version-wide absence of the byte-prefixed time encoding in this function, not a two-state-only trait,
though I did not exhaustively decompile all ~100+ ordinary stat blocks (only the first 3, which are
identical in shape) so cannot rule out an outlier later in the sequence.

## Slot → stat mapping

Same positional/independently-confirmed basis as the prior pass (`gms_v61_recheck.md`,
`gms_v61.md`) — this pass did not re-derive it, only cross-checked that it's still consistent with the
now fully-decomposed field lists:

| i | v83 order name | v61 evidence |
|---|---|---|
| 0 | EnergyCharge | positional (1st slot, base+int16) |
| 1 | DashSpeed | positional (Dash pair, byte-identical fn to i=2) |
| 2 | DashJump | positional (Dash pair, byte-identical fn to i=1) |
| 3 | MonsterRiding (RideVehicle) | **independently confirmed** — `OnTemporaryStatReset`/`Set` index 601 (=598+3) drives `CUser::ShowRideVehicleEffect`; mask `unk_97C300 = sub_66EDE6(3)` = shift 62, matching i=3's own per-member mask test |
| 4 | SpeedInfusion | positional (base+dword+int16, matches v83's "extra fields" shape) |
| 5 | GuidedBullet (HomingBeacon) | **independently confirmed** — index 603 (=598+5) is typed `TemporaryStat_GuidedBullet` with named `GetMobID`/`GetReason` accessors; `unk_97C2E0 = sub_66EDE6(5)` = shift 64 |

**Missing from v83's 7:** **Undead**. The loop bound is hard-coded `6` in three independent places
(`DecodeForRemote`'s trailer loop, `DecodeForLocal`'s trailer loop, and the constructor `sub_65F66F`'s
allocation loop) — there is no 7th slot/allocation/vtable in v61's `SecondaryStat` two-state array at
all. (Established in the prior pass; not re-derived here, only relied upon.)

## Hypothesis test result

**PARTIAL.**

- **Base block size (12 bytes on v61, vs 13 on v83): CONFIRMED** at the byte-count level — `sub_66E9B6`
  reads exactly 3 fields of 4 bytes each (`DecodeBuffer(4)`, `DecodeBuffer(4)`, `Decode4()`), summing to
  12, with no `Decode1` call anywhere in the function.
- **"The time field is 4 bytes on v61 (vs 5 on v83)": REFUTED as a mechanism claim, though the arithmetic
  happens to land on the predicted number.** No field in the two-state group's base or any of the 6
  per-member overrides is time-shaped (byte-prefixed, or combined with a local clock) — the base's third
  field is a bare, un-prefixed `Decode4` structurally identical to its other two fields, and none of the
  5 non-RideVehicle overrides' extra fields (`Decode2`, `Decode4`, `Decode2`, `Decode4`) touch
  `timeGetTime` either. The one place in `DecodeForLocal` that DOES combine a wire `Decode4` with a local
  `timeGetTime()` snapshot (the closest real analog of "time") is in the **ordinary, non-two-state**
  stat blocks — a different code region entirely, also 4 bytes, also never byte-prefixed. So: the byte
  count "4" is correct by coincidence of the base's total (12 = 4+4+4), not because a 5-byte time pattern
  present elsewhere shrank to 4 bytes in this group — no 5-byte (or byte-prefixed) time pattern exists
  anywhere in this version's `SecondaryStat` decode tree, two-state or ordinary, as far as scoped.
- **Per-member composition (dynamic=14, RideVehicle=12, GuidedBullet=16, SpeedInfusion=18): CONFIRMED**
  at both the byte-count and field-order level — every member's extra fields were individually
  decompiled and quoted above, and every running total matches its previously-established known size.
