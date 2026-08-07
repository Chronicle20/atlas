# GMS v92 — SecondaryStat two-state 7-member array: slot mapping + sizes (task-167 follow-up)

Binary: `GMS_v92_1_DEVM.exe.i64` · session: `acdfccff` · identity confirmed via `idb_list` (`input_path` = `E:\Programs\Nexon\IDBs_v9\GMS\V92_1\GMS_v92_1_DEVM.exe.i64`, `filename` = `GMS_v92_1_DEVM.exe.i64`).

This closes the two gaps left open by the prior v92 pass (`gms_v92.md` Question B): which raw-bit slot is `TemporaryStat_GuidedBullet`, and what the 7th member's vtable/size actually is. Both are resolved by reading the **constructor**, `sub_7129F0` (called from `SecondaryStat`'s own ctor path), which the prior pass could not find (`func_query` on `SecondaryStat` name patterns only surfaces `Reset`/`DecodeForRemote`/`DecodeForLocal`, all in-class methods, not the ctor helper — the ctor itself is unnamed).

## Method

1. Confirmed session identity (`idb_list`).
2. `xrefs_to("0x70d260")` (known `TemporaryStat_GuidedBullet::DecodeForClient`) → exactly one data xref: address `0xb362f4`. Read 40 bytes at `0xb362dc` (= `0xb362f4 - 0x18`) and confirmed byte offset `+0x18` (bytes 24–27: `0x60 0xd2 0x70 0x00` LE = `0x0070d260`) holds `0x70d260`. So `0xb362dc` is the **vtable base**, `0xb362f4` is its `+0x18` slot (this resolves the notation ambiguity in the prior pass: the six addresses it called "vtable `0xbXXXXX`" — `0xb36140`, `0xb361f4`, `0xb3617c`, `0xb362f4`, `0xb360f0`, `0xb361b8` — are each a `+0x18` **slot address**, i.e. `base+0x18`, not the base itself).
3. `search_text("11FCh", code_only, 0x700000-0x730000)` located every function touching offset `0x11FCh` (=4604 = `this`+1151 DWORDs, the array `SecondaryStat::DecodeForRemote` loops over). Among the hits, `sub_7129F0` uses `mov esi, [ebp+ecx*8+11FCh]` / `lea eax, [ebp+ecx*8+11FCh]` — an **indexed** access, i.e. the per-slot constructor/assignment loop.
4. Decompiled + disassembled `sub_7129F0` in full. It opens with:
   `` `eh vector constructor iterator'((void*)(this+4600), 8u, 7, sub_7065B0, sub_712100); ``
   then `do { switch(v2) { case 0: ...; case 1: case 2: case 6: ...; case 3: ...; case 4: ...; case 5: ...; } v13=++v2; } while (v2<7);` — a **7-iteration switch, one arm per array index 0..6**, each either constructing a fresh policy object (embedding a specific vtable pointer via `*obj = &off_BXXXXX`) or (cases 1/2/6) sharing one allocator function (`sub_7097E0`).
5. For each case, disassembled the `lea ecx, [ebp+0xNNNN]` immediately before its `call sub_7126F0` (the smart-pointer "assign into slot" helper — confirmed via its own decompile: `this[1] = a2` with interlocked refcounting). The array base is `ebp+0x11F8` (matches the `eh vector constructor iterator` call's `this+4600`); `sub_7126F0` writes into `this[1]` = `ecx+4`, so slot index = `(ecx_offset - 0x11F8) / 8`. This offset arithmetic is **direct, load-bearing evidence** for which case maps to which raw-bit index — not inferred from ordering.
6. Cross-checked every resulting vtable base's `+0x18` slot against the prior pass's six measured function addresses/sizes.

## Vtable → +0x18 slot → DecodeForClient mapping, all 7 slots

| idx (0-based) | switch case | ctor evidence (quoted) | slot addr (`ecx` at `sub_7126F0` call) | index arithmetic | vtable base | `+0x18` slot addr | `DecodeForClient` | size |
|---|---|---|---|---|---|---|---|---|
| 0 | `case 0` | `sub_709720` sets `mov dword ptr [esi], offset off_B36164` @ 0x709751 | `lea ecx, [ebp+11F8h]` @ 0x712a78 | (0x11F8−0x11F8)/8 = **0** | `0xb36164` | `0xb3617c` | `sub_712180` @ 0x712180 | **15** |
| 1 | `case 1` (shared) | `sub_7097E0` sets `mov dword ptr [esi], offset off_B361DC` @ 0x709811 | `mov esi,[ebp+ecx*8+11FCh]` / `lea eax,[ebp+ecx*8+11FCh]` @ 0x712bbc/0x712bc3, ecx=1 at this iteration | (index passed as loop var `ecx`=1) → **1** | `0xb361dc` | `0xb361f4` | `sub_70CDF0` @ 0x70cdf0 | **15** |
| 2 | `case 2` (shared, same code as case 1) | same as above, ecx=2 | same instrs, ecx=2 | **2** | `0xb361dc` | `0xb361f4` | `sub_70CDF0` @ 0x70cdf0 (same instance-type as idx1) | **15** |
| 3 | `case 3` | decompile: `*v6 = &off_B36128; /*0x712b5d*/` | `lea ecx, [ebp+1210h]` @ 0x712b6f | (0x1210−0x11F8)/8 = **3** | `0xb36128` | `0xb36140` | `sub_70CD00` @ 0x70cd00 (`TwoStateTemporaryStat<...NoExpire...>::DecodeForClient`) | **13** |
| 4 | `case 4` | decompile: `*v6 = &off_B361A0; /*0x712aae*/` | `lea ecx, [ebp+1218h]` @ 0x712ace | (0x1218−0x11F8)/8 = **4** | `0xb361a0` | `0xb361b8` | `sub_712040` @ 0x712040 (`TwoStateTemporaryStat<...Expire<BaseOnCurrentTime,DynamicTermSet>...>::DecodeForClient`) | **20** |
| **5** | **`case 5`** | decompile: `*v6 = &off_B362DC; /*0x712b0b*/` | `lea ecx, [ebp+1220h]` @ 0x712b20 | (0x1220−0x11F8)/8 = **5** | `0xb362dc` | `0xb362f4` | **`sub_70D260` @ 0x70d260 = `TemporaryStat_GuidedBullet::DecodeForClient`** | **17** |
| 6 | `case 6` (shared, same code as case 1/2) | same as case 1/2, ecx=6 | same instrs, ecx=6 | **6** | `0xb361dc` | `0xb361f4` | `sub_70CDF0` @ 0x70cdf0 (same instance-type as idx1/idx2) | **15** |

Re-decompiled `0x70cdf0` and `0x712180` directly to confirm the 15-byte size independently of the prior pass's table: both are byte-identical in shape — `TemporaryStatBase<long>::DecodeForClient(this,a2)` (13) + `CInPacket::Decode2(a2)` (2) = **15**. `0x70cd00`, `0x712040`, `0x70d260` sizes are inherited unchanged from the prior pass's table (all five re-confirmed by exact address match against the ctor's `off_BXXXXX` targets — no re-guessing).

## Gap 1 — GuidedBullet's slot index and raw shift

**Slot index = 5 (0-based). Raw shift = 115 + 5 = 120.**

Confirmed **two independent ways**, exactly as the task anticipated:

1. `xrefs_to(0x70d260)` → the single data xref is `0xb362f4`, which is vtable base `0xb362dc`'s `+0x18` slot (bytes at `0xb362dc+0x18` = `0x60 0xd2 0x70 0x00` LE = `0x0070d260`).
2. The constructor's `case 5` arm sets `*v6 = &off_B362DC` (i.e. `0xb362dc`, the same base) and then calls `sub_7126F0` with `ecx = ebp+0x1220`. `(0x1220 − 0x11F8) / 8 = 5`, and `DecodeForRemote`'s own read address for slot 5 is `this+0x11FC+8*5 = this+0x1224`, which equals `sub_7126F0`'s write target `ecx+4 = 0x1220+4 = 0x1224` — exact match.

The task's speculative shortcut ("check whether `0xb362f4`'s `+0x18` slot is `0x70d260`") turned out to rest on a notational ambiguity: `0xb362f4` **is itself** the `+0x18` slot address (not a vtable base needing a further `+0x18`), and it **does** hold `0x70d260` directly. Both checks converge on the same answer, so this is not a contradiction — just a correction to how "the vtable" was labeled in the prior pass's shorthand.

**Atlas-registry equivalent:** none exists yet. `libs/atlas-packet/model/character_temporary_stat.go` only encodes the **v95+** two-state group (`twoStateGuidedBullet` at raw bit 127, in the 8-member/6-effective-member v95 shape — see `character_temporary_stat.go:784-996`). There is no v92-specific (raw-bit-115-based) two-state table in the Go codebase to cross-reference; this pass's raw shift (120) is client-basis only, unmapped to any existing atlas registry constant.

## Gap 2 — the 7th member's vtable and size

**There is no independent "7th vtable."** The prior pass's assumption that six *distinct* vtables (13,15,15,17,20,20) accounted for 6 of 7 slots, leaving one truly novel vtable to find, was wrong in one specific place: **`0xb360f0` (its second "20-byte" candidate) does not belong to this array at all.** The constructor's `case 4` arm is the *only* size-20 member, and it targets vtable base `0xb361a0` (slot `0xb361b8`), not `0xb360d8`/`0xb360f0`. Read 40 bytes at `0xb360f0` directly: its own `+0` DWORD is `0x40 0x20 0x71 0x00` = `0x712040` — the **same** `Expire<BaseOnCurrentTime,...>` function as `0xb361b8`'s slot, just a **different, unreferenced vtable instance** of identical shape (consistent with the prior pass's own note that `0xb75d70` — a same-code vtable for the base type — belongs to an unrelated consumer like `CMob`'s temp-stat system; `0xb360f0` is evidently a sibling static instantiation of the same template that this particular `SecondaryStat` array simply never points at).

The actual 7th slot (index 6, raw bit 121) is a **third instance of the 15-byte type already used for indices 1 and 2** (`sub_7097E0` → vtable `0xb361dc` → slot `0xb361f4` → `sub_70CDF0`, 15 bytes). The constructor's `case 1: case 2: case 6:` block is one shared switch arm — the decompiler merged them because they run byte-identical code (same allocator, same `sub_7097E0` initializer, same final vtable) — meaning indices 1, 2, and 6 are three separate freshly-allocated instances of the *same* class, not three different classes.

So: **the 7th member's vtable is `0xb361dc` (slot `0xb361f4`, `sub_70CDF0`), size 15** — not a new/undiscovered vtable, and not `0xb360f0`/20.

## Full 7-member total

15 (idx0) + 15 (idx1) + 15 (idx2) + 13 (idx3) + 20 (idx4) + 17 (idx5, GuidedBullet) + 15 (idx6) = **110 bytes**.

This number is **not** obtained by rounding toward the pre-95 reference — it falls out directly from the ctor's index arithmetic and the five distinct vtable bases it actually uses (one of which, `0xb361dc`, is instantiated three times: indices 1, 2, 6). The prior pass's "100 subtotal across 6 slots" was inflated by wrongly including `0xb360f0` (unreferenced) instead of recognizing the true 7th slot as a repeat of `0xb361f4`.

## Verdict

**GuidedBullet = raw-bit slot index 5, raw shift 120 — confirmed two independent ways (direct `xrefs_to` on `0x70d260` landing on vtable-base-`0xb362dc`'s `+0x18` slot; and the constructor's `case 5` arm writing that same vtable base into array slot 5 via verified offset arithmetic).** The 7-member array is **not** six distinct types + one unknown; it is **five distinct vtables**, one of which (`0xb361dc`/`sub_70CDF0`, 15 bytes) is instantiated three times (indices 1, 2, 6). `0xb360f0` (previously treated as the "missing 20-byte 7th") is a **false lead** — a same-shape, same-code, but unreferenced sibling vtable not used by this array. Full 7-member `DecodeForClient` trailer total = **110 bytes**. No `UNVERIFIED` items remain for either gap — both are resolved with `xrefs_to` + ctor-disassembly evidence, not inference.
