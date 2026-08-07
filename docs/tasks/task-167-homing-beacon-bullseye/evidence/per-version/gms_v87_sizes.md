# GMS v87 — SecondaryStat two-state per-member block sizes (task-167 follow-up)

Binary: `GMSv87_4GB.exe.i64` · session: `d51ecbd3` · identity confirmed via `idb_list`:
session `d51ecbd3` → `input_path: E:\Programs\Nexon\IDBs_v9\GMS\v87\GMSv87_4GB.exe.i64`,
`filename: GMSv87_4GB.exe.i64`. Matches exactly.

## Method used

The prior pass (`gms_v87.md`, Question B) searched for a `SecondaryStat::SecondaryStat`
symbol via `func_query`/`list_globals`/`search_structs` and found none, then stopped. This
pass does **not** repeat that search. Instead it works forward from the mask-gated trailer
loop in `SecondaryStat::DecodeForRemote` @`0x7d8533`:

```c
v70 = 0;                    /*0x7d8e3c*/
v71 = this + 848;           /*0x7d8e3e*/
do {
  v72 = sub_7DCED7(v76, v70);                        /*0x7d8e49*/
  v73 = UINT128::operator&(&v77, v75, v72);           /*0x7d8e58*/
  if ( UINT128::operator bool(v73) )                  /*0x7d8e5f*/
    (*(**v71 + 24))(*v71, iPacket);                   /*0x7d8e6d — vtable+0x18, DecodeForClient*/
  ++v70; v71 += 2;
} while ( v70 < 7 );
```

Steps actually taken:

1. Found the shared base decoder by searching `xrefs_to(CInPacket::DecodeBuffer @0x43374e)`
   (139 total callers) for a *small* function calling `DecodeBuffer` exactly twice — the
   signature `TemporaryStatBase<T>::DecodeForClient` has on every prior version audited
   (v72/JMS: `DecodeBuffer(4)+DecodeBuffer(4)+"time"(Decode1+Decode4=5)=13 bytes`). Exactly one
   candidate matched: `sub_7E4D25` (size `0x64`), confirmed by decompile to be exactly that
   13-byte base.
2. Found `xrefs_to(sub_7E4D25)` — 4 code callers (`sub_7E4C5B`, `sub_7E4EB0`, `sub_7E4FEA`,
   `sub_7E5165`) plus 1 unrelated data xref. Decompiled each: they are per-member
   `DecodeForClient` wrappers that call the base and optionally add bytes.
3. Found `xrefs_to(sub_7E4EB0)` (the base-only, 13-byte wrapper) → `sub_7CAAB5`, which calls
   `sub_7E4EB0` then `CInPacket::Decode4` — a 17-byte shape (base+4), the GuidedBullet
   candidate size from the task brief.
4. **Found the actual member-array constructor**, not by name search but by reading the raw
   vtable-cluster bytes and walking backward: `get_bytes` at the six `DecodeForClient` data
   xrefs (each function's own "who points at me" query) gave six candidate vtable-start
   addresses (`slot_addr − 0x18`). `xrefs_to` on those six vtable-start addresses returned six
   distinct small constructor functions (`sub_7CABC3`, `sub_7CAB6B`, `sub_7CAC57`,
   `sub_7CAC7E`, plus GuidedBullet's `sub_7CA8B4`/`sub_7CAD0D` — the last being an unrelated
   ICF-folded sibling, see below). `sub_7CA8B4` — called directly from
   `CWvsContext::CWvsContext` at `0xa96d4f` (`sub_7CA8B4(&this[1].m_Cookie.szCookie[248])`) —
   turned out to be the actual 7-member array constructor: it opens with
   `` `eh vector constructor iterator'((this + 3388), 8u, 7, sub_7E4F0C, sub_7E4EF6) `` (7
   elements, stride 8, exactly matching the `DecodeForRemote` loop's `this+848` dword base ×
   4 = 3392 ≈ 3388-ish region) followed by an explicit `for(i=0;i<7;++i)` **7-way switch that
   allocates and placement-constructs each member with a distinct concrete class**. This *is*
   `CWvsContext`'s inlined equivalent of `SecondaryStat::SecondaryStat` for this array — it
   never carried a demangled `SecondaryStat::` name because it's a file-local helper the
   compiler emitted for the member-initializer list, not an out-of-line class method.

Full quoted switch (`sub_7CA8B4` @ `0x7ca8b4`, confirmed by decompile):

```c
`eh vector constructor iterator'((this + 3388), 8u, 7, sub_7E4F0C, sub_7E4EF6); /*0x7ca8de*/
for ( i = 0; i < 7; ++i ) {
  if ( !v2 ) {                                    // idx 0
    ...Alloc(0x30)...
    v5 = sub_7CABC3();                             /*0x7caa13*/
    goto LABEL_26;
  }
  if ( v2 <= 0 ) goto LABEL_27;
  if ( v2 <= 2 ) {                                 // idx 1, idx 2
LABEL_20:
    ...Alloc(0x30)...
    v5 = sub_7CAC7E();                             /*0x7ca9e4*/
LABEL_26:
    sub_7E4F17(v5);
    goto LABEL_27;
  }
  switch ( v2 ) {
    case 3:                                        // idx 3
      ...Alloc(0x24)...
      v5 = (sub_7CAB6B)();                         /*0x7ca9b9*/
      goto LABEL_26;
    case 4:                                        // idx 4
      ...Alloc(0x34)...
      v5 = sub_7CAC57();                           /*0x7ca98b*/
      goto LABEL_26;
    case 5:                                        // idx 5 — GuidedBullet
      v3 = ZAllocEx<...>::Alloc(..., 0x28u);       /*0x7ca92e*/
      v4 = v3;
      if ( v3 ) {
        sub_7CAB6B(v3);                            /*0x7ca942 — base-construct as RideVehicle-type*/
        v4[9] = 0;
        *v4 = &off_BA2834;                         /*0x7ca94a — STOMP vtable to GuidedBullet's own*/
        v4[8] = &off_BA2818;                       /*0x7ca950 — secondary (MI) vtable*/
      }
      sub_7E4F17(v4);
      break;
    case 6:                                        // idx 6
      goto LABEL_20;                                /*0x7ca91f — shares idx1/idx2's path*/
  }
}
```

This is byte-for-byte the same shape task-167's other pre-95 evidence files describe for
v72/JMS: GuidedBullet base-constructs as the RideVehicle-type object then stomps its vtable
to its own, and one class (`sub_7CAC7E`) is reused for three indices (1, 2, 6).

## Vtable → DecodeForClient resolution (per index)

For each per-index ctor function, `xrefs_to` on the six candidate vtable-start addresses
(`DecodeForClient_data_xref_addr − 0x18`) directly named the constructor that writes that
vtable pointer, closing the loop back to the switch above:

| Ctor called (switch) | Sets vtable @ | `xrefs_to(vtable_start)` confirms | Vtable `+0x18` slot addr | `DecodeForClient` |
|---|---|---|---|---|
| `sub_7CABC3`@`0x7cabc3` | `0xba28c4` | `sub_7CABC3` (fn `0x7cabc3`) | `0xba28dc` | `sub_7E4FEA`@`0x7e4fea` |
| `sub_7CAC7E`@`0x7cac7e` | `0xba2964` | `sub_7CAC7E` (fn `0x7cac7e`) | `0xba297c` | `sub_7E5165`@`0x7e5165` |
| `sub_7CAB6B`@`0x7cab6b` | `0xba2888` | `sub_7CAB6B` (fn `0x7cab6b`) | `0xba28a0` | `sub_7E4EB0`@`0x7e4eb0` |
| `sub_7CAC57`@`0x7cac57` | `0xba2928` | `sub_7CAC57` (fn `0x7cac57`) | `0xba2940` | `sub_7E4C5B`@`0x7e4c5b` |
| GuidedBullet (`&off_BA2834` stomp, `0x7ca94a`) | `0xba2834` | `sub_7CA8B4` (fn `0x7ca8b4`, the switch itself, quoted above) | `0xba284c` | `sub_7CAAB5`@`0x7caab5` |

(`0xba2988` → `sub_7CAD0D` was also found in the initial cluster scan but is **not** referenced
by the 7-way switch above — it is an unrelated class, byte-identical-code-folded onto the same
`sub_7E4C5B` body by the linker's ICF, elsewhere in the binary. Excluded from the 7-member
count.)

## Per-member block sizes (bytes read from `CInPacket`)

Shared base, confirmed by decompile:

- `sub_7E4D25`@`0x7e4d25`: `CInPacket::DecodeBuffer(a2, this+3, 4u)` (4) +
  `CInPacket::DecodeBuffer(a2, this+4, 4u)` (4) + `sub_7CA264(a2)` (5 — confirmed:
  `CInPacket::Decode1(a1)` + `CInPacket::Decode4(a1)`, the bool+int32 "time" pattern) =
  **13 bytes base**, common to every member.

| Idx | `DecodeForClient` | Quoted extra decode | Extra bytes | **Block size** |
|---|---|---|---|---|
| 0 | `sub_7E4FEA`@`0x7e4fea` | `sub_7E4D25(this,...)` (13) + `CInPacket::Decode2(a2)` (2) | 2 | **15** |
| 1 | `sub_7E5165`@`0x7e5165` | `sub_7E4D25(this,...)` (13) + `CInPacket::Decode2(a2)` (2) | 2 | **15** |
| 2 | `sub_7E5165` (same fn as idx 1) | same | 2 | **15** |
| 3 | `sub_7E4EB0`@`0x7e4eb0` | `sub_7E4D25(this,a2)` only, no extra call | 0 | **13** |
| 4 | `sub_7E4C5B`@`0x7e4c5b` | `sub_7E4D25(this,...)` (13) + `sub_7CA264(a2)` (5, second "time" read) + `CInPacket::Decode2(a2)` (2) | 7 | **20** |
| 5 | `sub_7CAAB5`@`0x7caab5` | `sub_7E4EB0(this,...)` (13, = idx-3's decoder reused as base) + `CInPacket::Decode4(a2)` (4) | 4 | **17** |
| 6 | `sub_7E5165` (same fn as idx 1/2) | same | 2 | **15** |

Quoted decompiles backing each row:

```c
// idx 0 — sub_7E4FEA
sub_7E4D25(this, &a2->m_bLoopback);      /*0x7e500f*/
result = CInPacket::Decode2(a2);          /*0x7e5017*/
*(this + 44) = result;                    /*0x7e501e*/
```
```c
// idx 1/2/6 — sub_7E5165
sub_7E4D25(this, &a2->m_bLoopback);      /*0x7e518a*/
result = CInPacket::Decode2(a2);          /*0x7e5192*/
*(this + 44) = result;                    /*0x7e5199*/
```
```c
// idx 3 — sub_7E4EB0 (base only)
result = sub_7E4D25(this, a2);            /*0x7e4ed5*/
```
```c
// idx 4 — sub_7E4C5B
sub_7E4D25(this, &a2->m_bLoopback);      /*0x7e4c80*/
*(this + 40) = sub_7CA264(a2);            /*0x7e4c91*/
result = CInPacket::Decode2(a2);          /*0x7e4c94*/
*(this + 48) = result;                    /*0x7e4c9b*/
```
```c
// idx 5 — sub_7CAAB5 (GuidedBullet)
sub_7E4EB0(this, &a2->m_bLoopback);      /*0x7caada*/
result = CInPacket::Decode4(a2);          /*0x7caae2*/
this[9] = result;                         /*0x7caae9*/
```

## Sum and verdict

15 + 15 + 15 + 13 + 20 + 17 + 15 = **110 bytes**.

**MATCHES 110** — identical total, identical per-index shape (15/15/15/13/20/17/15), and
identical relative order to the pre-95 v72/JMS reference.

## GuidedBullet slot — confirmed, not assumed

**Idx 5 = GuidedBullet**, confirmed two independent ways:

1. **Construction pattern**: idx 5's switch arm (`case 5`) explicitly base-constructs via
   `sub_7CAB6B` (the *same* ctor idx 3/RideVehicle uses) and then stomps the vtable pointer to
   `&off_BA2834` — exactly the "GuidedBullet reuses RideVehicle's ctor, then vtable-swapped"
   pattern independently documented for v72 and JMS in this same task's evidence set.
2. **Block shape**: idx 5's `DecodeForClient` (`sub_7CAAB5`) calls idx 3's own decoder
   (`sub_7E4EB0`, the 13-byte RideVehicle base) as its base, then reads one more `Decode4` (4
   bytes) — the exact "base(13) + 4-byte `dwMobId`" shape the task brief specified, landing on
   the distinctive **17-byte** size.
3. Cross-check against the existing `gms_v87.md` evidence: that file independently confirmed
   (via `OnTemporaryStatReset`'s `*(v3+858)` dereference gated on `dword_CA82B8`, raw shift
   `86+5=91`) that **member index 5** is GuidedBullet/HomingBeacon. Index 5 is exactly the row
   this pass measured at 17 bytes. Both derivations (behavioral, from `OnTemporaryStatReset`;
   and structural, from the constructor switch + vtable chase) agree on the same index.

**Idx 3 = RideVehicle/MonsterRiding** is likewise doubly confirmed: `gms_v87.md`'s
`OnTemporaryStatReset` cross-reference (`*(v3+854)`, raw shift 89) and this pass's constructor
switch (`case 3` → `sub_7CAB6B` → 13-byte base-only decoder, the "NoExpire" shape) agree.

## Note on the "PER-MEMBER MASK-GATED" trailer-read finding

Unaffected by this pass — already independently confirmed in `gms_v87.md` via the disassembled
`DecodeForRemote`/`DecodeForLocal` tail loops (bit-test + conditional-skip of the vtable+0x18
call). Not re-derived here.
