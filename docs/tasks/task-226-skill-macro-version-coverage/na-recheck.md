# v48 / v61 n-a re-check (task-226, FR-2)

Sessions: gms_v48 = `93cc947e` (`GMS_v48_1_DEVM.exe`), gms_v61 = `415bf585`
(`GMS_v61.1_U_DEVM.exe`), re-verified live via `idb_list` immediately before
use (per CLAUDE.md IDA-lookup discipline).

## SKILL_MACRO × gms_v48 — CONFIRMED-NA

**Step 1 (sibling cross-check).** gms_v48 has no `OnMacroSysDataInit`
anywhere: `func_query name_regex "MacroSysMan|OnMacroSysDataInit|
FlushToSvr|MACROSYSDATA|SINGLEMACRO"` over the whole image → 0 hits (also
`find_regex "macro"` → 0 hits). The receive side does not exist to cross
against, so this is not "receive proves send" pressure — it is corroborating
absence on both sides at once.

**Step 2 (invariant search, exhaustive).** Located the v48
`COutPacket::COutPacket(&pkt, int)` ctor (`??0COutPacket@@QAE@J@Z`,
`0x57b77e`) and enumerated **all 317** code xrefs to it via `xref_query`
(direction=`to`, paginated in 10-record pages via `offset`, confirmed
`total: 317` on every page, all 32 pages read to `next_offset: null`). Every
call site's containing function was inspected:

- Named callers (the large majority) were read from their mangled/demangled
  symbol; none is `CMacroSysMan`-class. The only `FuncKeyMappedMan` sender
  present is `SaveFuncKeyMap` (`0x4e5fae`) — the unrelated `CHANGE_KEYMAP`
  op, not a macro list.
- Every unnamed (`sub_XXXXXX`) caller — **44 in total** — was individually
  decompiled and checked against the confirmed send shape (`Encode1(count
  clamped <= 5)` then a per-entry loop of
  `EncodeStr(name)+Encode1(shout)+3x Encode4(skillId)`, established on
  gms_v72/v79/v83/v84/v87/v92/v95/jms_v185 and re-confirmed on gms_v61
  below). None matches. The four largest unnamed candidates —
  `sub_69CC07` (opcode 41, quest-portal/UI dispatch), `sub_6A0528` (opcode
  36, melee-attack `CLOSE_RANGE_ATTACK`-family sender), `sub_6A228C`
  (opcode 37, ranged-attack sender), `sub_6A3AC7` (opcode 38, magic-attack
  sender) — are all large skill-attack senders unrelated to macros, verified
  by reading their full decompile and the literal opcode each pushes into
  its `COutPacket` ctor.

No candidate anywhere in the 317-site enumeration constructs a packet with
`Encode1(count<=5)` followed by a string+bool+3×uint32 per-entry loop.

**Verdict:** absence is bilateral (no send site, no receive site, no
`CMacroSysMan`/`MACROSYSDATA`/`SINGLEMACRO` symbol or string anywhere),
exhaustively searched by invariant (not name), and consistent with v48
being the oldest binary in the corpus and predating the skill-macro
feature entirely. Evidence recorded in `feature-na-evidence.yaml`.

## SKILL_MACRO × gms_v61 — CORRECTED

**Step 1 (sibling cross-check).** gms_v61's receive side is real and fully
resolved: `CWvsContext::OnMacroSysDataInit` (`0x849bce`) →
`CMacroSysMan::SetMacro` (`0x59744b`, already named) →
`MACROSYSDATA::Decode` (`0x4b796d`) → `SINGLEMACRO::Decode` (`0x4b78d5`),
decoding a `Decode1`-clamped-to-5 count and, per entry,
`DecodeStr`+`Decode1`+3×`Decode4` — i.e. it decodes and stores macro state.
Per "the receive side proves the send side," this was positive pressure to
keep looking rather than accept the prior "NOT FOUND" framing.

**Step 2 (invariant search — the technique the harvest log named as the
concrete next step).** Located the v61 `COutPacket::COutPacket(&pkt, int)`
ctor (`??0COutPacket@@QAE@J@Z`, `0x5ffc4f`) and enumerated its code xrefs
via `xref_query` (direction=`to`, paginated, `total: 391` reported on every
page). The harvest log's own hint — "a v61 `MACROSYSDATA::Encode` /
`SINGLEMACRO::Encode`, if present, is likely adjacent in the binary" —
paid off immediately: `CMacroSysMan::SetMacro` sits at `0x59744b`, and the
xref enumeration surfaces an unnamed function `sub_59746C` at `0x59746c`
(0x21 bytes later, well inside the same translation unit) as a caller of
the `COutPacket` ctor. Decompiling it:

```
int sub_59746C() {
  COutPacket::COutPacket((COutPacket *)v1, 101);
  sub_4B7928(v1);
  CClientSocket::SendPacket(g_pClientSocketInstance, v1);
  return ZArray<unsigned char>::RemoveAll(v2);
}
```

— an exact match to the confirmed `FlushToSvr` shape (construct → delegate
to a struct's `Encode` → `SendPacket` → `RemoveAll`), pushing opcode **101**
(0x65). Following the delegate:

```
sub_4B7928(this, a2):            // MACROSYSDATA::Encode
  v3 = this[3] ? clamp(*(this[3]-4), 5) : 0;
  COutPacket::Encode1(a2, v3);
  loop v3 times (stride 29): sub_4B7884(a2)

sub_4B7884(this, a2):            // SINGLEMACRO::Encode
  COutPacket::EncodeStr(a2, name);
  COutPacket::Encode1(a2, field25 != 0);   // shout, raw-store/!=0, same
                                            // polarity as every other version
  loop 3: COutPacket::Encode4(a2, skillId);
```

This is byte-for-byte the confirmed cross-version send shape — count
clamped to 5, `EncodeStr`+`Encode1`+3×`Encode4` per entry, 29-byte stride
(matching `SINGLEMACRO`'s per-entry size implied by `MACROSYSDATA::Decode`'s
own stride) — and `sub_4B7884` sits immediately before the already-resolved
`SINGLEMACRO::Decode` (`0x4b78d5`), the adjacency the harvest log predicted.

All three functions were renamed (`mcp__ida-pro__rename`) and the IDB saved
(`idb_save`, confirmed `ok: true`):

| old name | new name | address |
|---|---|---|
| `sub_59746C` | `CMacroSysMan::FlushToSvr` | `0x59746c` |
| `sub_4B7928` | `MACROSYSDATA::Encode` | `0x4b7928` |
| `sub_4B7884` | `SINGLEMACRO::Encode` | `0x4b7884` |

No serverbound opcode-101 collision exists in `gms_v61.yaml` (101 was only
used clientbound, for `WHISPER`).

**Verdict:** the send site exists and was located, not absent. Registered
in `docs/packets/registry/gms_v61.yaml` at its opcode-sorted position
(between `QUEST_ACTION`=98 and `SPOUSE_CHAT`=109) with `provenance:
ida-discovered`. `layout-derivation.md`'s gms_v61 row updated to carry both
directions; the "no divergence, no version gate" conclusion is unaffected
(v61's encode shape matches every other populated version exactly). gms_v61
is now in scope for Tasks 6-12 like any other populated version — it needs
a template binding, a fixture, and verification evidence, which are those
tasks' job per the plan, not deferred here.

## MACRO_SYS_DATA_INIT × gms_v48 — CONFIRMED-NA

**Step 1 (sibling cross-check).** The send side (`SKILL_MACRO`) is
confirmed absent above (317/317 `COutPacket` call sites enumerated, no
match) — no positive pressure from that direction either.

**Step 2 (invariant search).** `CWvsContext::OnPacket` (`0x70d215`) is the
receive-side dispatcher this op would need a case in. It is a **fully
compiled, fully-named switch** over cases 25 through 70 (`default: return`)
— every single case dispatches to a real, named handler (`OnInventoryOperation`
… `OnGivePopularityResult` … `OnPartyResult` … through `sub_7215EA` at case
70), each individually legible by name and, for the four unnamed stub cases
(58, 62, 68, 70), by decompiled body — none decodes a macro list (`Decode1`
count clamped to 5, then a `DecodeStr`+`Decode1`+3×`Decode4` per-entry loop).
There is no unresolved/default case in the 25-70 range that could hide an
unnamed macro-init handler, and no case number in that range is unaccounted
for by a real handler. `func_query name_regex
"MacroSysMan|OnMacroSysDataInit|MACROSYSDATA|SINGLEMACRO"` over the whole
image returns 0 hits.

**Verdict:** absence is bilateral, exhaustively bounded on the receive side
by the fully-enumerated compiled switch (not a name search) and on the send
side by the 317-site ctor-xref enumeration above. Consistent with v48
predating the skill-macro feature entirely. Evidence recorded in
`feature-na-evidence.yaml`.

## Summary

| cell | verdict | search extent |
|---|---|---|
| `SKILL_MACRO` × `gms_v48` | CONFIRMED-NA | all 317 `COutPacket::COutPacket(long)` call sites scanned (317/317 named-or-decompiled); 0 macro-shaped matches |
| `SKILL_MACRO` × `gms_v61` | CORRECTED | send site located at `CMacroSysMan::FlushToSvr` (`0x59746c`), opcode 101, within the 391-site `COutPacket`-ctor xref enumeration |
| `MACRO_SYS_DATA_INIT` × `gms_v48` | CONFIRMED-NA | `CWvsContext::OnPacket`'s full compiled switch (cases 25-70, 46 cases) exhaustively enumerated; 0 macro-shaped matches |

Downstream: `SKILL_MACRO` × `gms_v61` is corrected into scope for Tasks
6-12 (template binding, fixture, verification) exactly like any other
populated version. The two `gms_v48` cells remain `n-a` and are out of
scope for those tasks.
