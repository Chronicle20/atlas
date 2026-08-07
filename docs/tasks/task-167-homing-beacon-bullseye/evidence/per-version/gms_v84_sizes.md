# GMS v84 — SecondaryStat two-state member block sizes

Session: `5881cf84` (confirmed via `idb_list`: `input_path` ends
`GMS\v84_1\GMS_v84.1_U_DEVM.i64`, matching the requested
`GMS_v84.1_U_DEVM.i64` binary).

## Method used

1. Confirmed the 7-member decode loop in `sub_7AC409` (local decode) via
   disasm: base `esi += 0xCEC` (3308 decimal), loop body `add esi, 8`,
   `cmp ebx, 7` — i.e. 7 members at stride 8 bytes starting at relative
   offset `0xCEC`, dispatched through `call dword ptr [eax+18h]`
   (vtable slot `+0x18`) on the pointer stored in each slot. Confirmed
   identical field-offset table (`dword_C49638`, `dword_C49628`, …) and
   identical loop shape in the sibling function `sub_7A5D2B` and in the
   "clear" function `sub_7A4758` and destructor `sub_79F3A1`.
2. Determined which raw bit each loop index `i` (0..6) tests: the mask
   lookup helper `sub_7B0D46(a1, a2)` computes `sub_89F235(a2 + 84)` —
   i.e. loop index `i` maps to **raw bit `84 + i`**. This gives
   `i=5 → raw 89`, matching the already-established GuidedBullet raw
   shift of 89. **GuidedBullet is confirmed at loop slot index 5**
   (0-based).
3. Determined the member array's true base address: `CUserRemote::
   OnResetTemporaryStat` (`0x9c3d2d`, disasm) does
   `lea ecx, [esi+2C24h]` immediately before `call sub_7A4758` — i.e.
   the SecondaryStat-like sub-object that owns this 7-member array is
   embedded at `CUserRemote + 0x2C24`, not at object offset 0. The
   7 member-pointer slots therefore live at absolute `CUserRemote`
   offsets `0x2C24 + 0xCEC + 8*i` for `i = 0..6`
   (`0x3910, 0x3918, 0x3920, 0x3928, 0x3930, 0x3938, 0x3940`).
4. Attempted to find each member's real (derived-class) vtable/
   `DecodeForClient` address by locating the allocation site
   (`new ClassX(...)` assigning a vtable into one of the 7 slots), since
   no `CTemporaryStat`/`SecondaryStat` constructor symbol exists (per
   the prior pass). This is the blocking step and is where the
   investigation stalled — see "What was ruled out" below.

## What was ruled out (evidence, not speculation)

- **`sub_9045FF`** (found via immediate-search hits on the *relative*
  byte offsets 3308/3316/3324/3332/3340/3348 inside `sub_7AC409`'s own
  address range) is **not** the constructor for this member array. It
  zeroes relative offsets `0xCEC`–`0xD10` individually (`mov [esi+X],
  ebx` for `X` = `0xCF0..0xD10`, i.e. plain NULL-init of unrelated
  fields) and then runs
  `` `eh vector constructor iterator'(esi+0xD14, size=4, count=4, ctor=sub_4B7D8C, dtor=sub_415087)`` —
  disasm at `0x9047a0`–`0x9047e7`. Chasing that array's consumer
  (`sub_915410`, found via immediate-search on absolute offset `0xD14`)
  showed a COM/`VARIANTARG`-based generic property accessor
  (`sub_4463E3`, `sub_429206`, global `pvargSrc` at `0xC4F9D8`) —
  unrelated to packet codecs. The relative-offset match was coincidental:
  `sub_9045FF` operates on a **different object** than `CUserRemote`'s
  embedded SecondaryStat (confirmed in step 3 above, which showed the
  real sub-object lives at `CUserRemote+0x2C24`, so the true absolute
  slot addresses are `CUserRemote+0x3910..0x3940`, not `+0xCEC..+0xD1C`).
  An immediate-search on the correct absolute offsets (14608/14616/
  14624/14632/14640/14648/14656 decimal) returned only one incidental
  hit (`0xa6a1ab`, inside `CWvsContext::OnInventoryOperation`, an
  unrelated inventory-slot count literal — confirmed by decompiling it).
- `CUserRemote::OnSetTemporaryStat` (`0x9c3bfb`), `CUserRemote::
  OnResetTemporaryStat` (`0x9c3cbf`), and `CUserRemote::
  OnAvatarModified` (`0x9c3a1c`) were decompiled in full; none allocates
  (`new`) any of the 7 members — they only decode raw bytes into the
  pre-existing bitmask/value fields and dispatch through the existing
  (already-allocated-elsewhere-or-null) member pointers.
- `sub_79F3A1` (destructor) and the reset path in `sub_7A4758` both
  destroy the 7 members through a **shared, generic deleter**
  (`sub_7D71F1`, called on `*slot`), which is type-erased and reveals no
  per-member vtable.
- No RTTI/type-descriptor strings exist for these classes
  (`find_regex` for `GuidedBullet|Beacon|Bullseye|Homing|TemporaryStat`
  and for `\.\?AV.{0,40}Stat` both returned 0 matches).
- `find data_ref` on `sub_7AC409`, `sub_7A5D2B`, `sub_7A4758`,
  `sub_79F3A1` returned no hits — these container-class methods are
  themselves non-virtual (called directly), consistent with the 7
  *members* (not the container) being the polymorphic objects, but this
  means the container's own vtable can't be used to bootstrap the
  member vtables either.
- The block of tiny (`0x1e`/`0x2b`/`0x5`-byte) functions immediately
  following `sub_7A5D2B` in the address space (`sub_7AABD6` onward,
  `sub_7B0xxx`/`sub_7B1xxx`/`sub_7B2xxx`) were sampled by decompile and
  are trivial `GetX`/`SetX` one-liners for the ~40 *simple* (non-vtable)
  mask-gated fields, not the 7 two-state members.
- The two largest nearby functions (`sub_7B0636`, 0x583 bytes;
  `sub_7B0ECC`, 0x8ca bytes) were decompiled and are unrelated
  (a bulk field-reset function for simple fields, and an EXP-table
  generator, respectively).

## Table

| Slot (i) | Raw bit (84+i) | Vtable addr | DecodeForClient addr | Size (bytes) | Evidence |
|---|---|---|---|---|---|
| 0 | 84 | UNVERIFIED | UNVERIFIED | UNVERIFIED | allocation site not found (see above) |
| 1 | 85 | UNVERIFIED | UNVERIFIED | UNVERIFIED | allocation site not found |
| 2 | 86 | UNVERIFIED | UNVERIFIED | UNVERIFIED | allocation site not found |
| 3 | 87 | UNVERIFIED | UNVERIFIED | UNVERIFIED | allocation site not found |
| 4 | 88 | UNVERIFIED | UNVERIFIED | UNVERIFIED | allocation site not found |
| 5 (GuidedBullet, raw 89) | 89 | UNVERIFIED | UNVERIFIED | UNVERIFIED | slot identity confirmed (sub_7B0D46 index math); allocation site not found |
| 6 | 90 | UNVERIFIED | UNVERIFIED | UNVERIFIED | allocation site not found |

Sum: **UNVERIFIED** (0 of 7 members' block sizes measured).

## Verdict

**PARTIAL: 0 of 7 members measured, subtotal 0**

Structural facts that ARE verified for v84: 7-member loop base
`CUserRemote+0x2C24+0xCEC`, stride 8 bytes, vtable dispatch at slot
`+0x18`, index-to-raw-bit mapping `raw = 84 + i`, and slot 5 = raw bit
89 = GuidedBullet (consistent with the already-established raw-shift-89
fact). None of the 7 derived-class vtables/`DecodeForClient` addresses
could be located — the allocation sites that would stamp a real vtable
into each of the 7 slots were not found despite checking: the local/
remote decode functions, the reset/clear function, the destructor, the
avatar-modify handler, RTTI strings, and every nearby moderately-sized
function in the same source-file address cluster. This blocks the
per-member byte-count measurement entirely; no sizes are reported to
avoid inventing values.
