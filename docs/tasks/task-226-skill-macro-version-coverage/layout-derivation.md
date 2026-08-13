# Skill macro — per-version layout derivation (task-226)

Sources: `docs/packets/ida-exports/` (spliced in Task 2) and live decompiles via
ida-pro-mcp (this task, addresses taken from `harvest-log.md`, independently
re-decompiled at every address before being relied on — per the controller
ruling that cited `file:line`/address tables can be stale). Addresses and
binary names: see `harvest-log.md`.

Both wire functions on every populated version delegate the entire
read/write to `MACROSYSDATA::Encode`/`Decode`, which in turn delegates the
per-entry body to `SINGLEMACRO::Encode`/`Decode`. All field-shape and
capacity findings below come from decompiling that bottom layer directly —
the layer that actually issues `COutPacket::EncodeN`/`CInPacket::DecodeN`
calls — not from the top-level `FlushToSvr`/`OnMacroSysDataInit` functions,
which do no wire I/O of their own (confirmed structural fact from Task 2,
re-confirmed here by decompile on gms_v83; see "Top-level delegation" below).

## Verdicts

- **Shout polarity: INVERTED** (wire byte `0` = shout on, wire byte `1` =
  shout off). Settled on gms_v83 two independent ways, then confirmed
  structurally identical (raw-copy-on-decode / compare-nonzero-on-encode,
  zero deviation) on all nine populated versions:
  1. `CMacroSysMan::IsShoutMacro` (gms_v83, `0x631d19`) —
     `return (a2 & 0x80000000) != 0 || (v2 = this[11]) == 0 || a2 >= *(v2 - 4) || *(29 * a2 + v2 + 25) == 0;`
     — for a valid index this reduces to `field25 == 0`, i.e. `IsShoutMacro`
     is **true when the decoded byte is 0**. Its two callers,
     `CUIMacroSysEx::OnSelected` (`0x8b8e63`) and `sub_8A984D` (`0x8a984d`),
     both do `CCtrlCheckBox::SetChecked(v5, IsShoutMacro)` — the return
     value directly drives the shout checkbox's checked state, so
     `IsShoutMacro==true` means "checkbox ticked" means "shout on".
  2. gms_v95 has richer symbol/type info than v83 for this struct: the field
     the other versions store as an anonymous `this+25` byte is literally
     named `bMute` in v95's decompile (`SINGLEMACRO::Decode`, `0x4f97f0`:
     `this->bMute = CInPacket::Decode1(v2);`; `SINGLEMACRO::Encode`,
     `0x4f9710`: `COutPacket::Encode1(v6, v2->bMute != 0);`). `bMute` (muted)
     is semantically the *negation* of "shout" — a muted macro does not
     shout. `bMute` is stored as the raw decoded byte with no inversion, so
     wire `1` = muted = not-shout, wire `0` = not-muted = shout.
  3. The write side (gms_v83): `CMacroSysMan`'s macro-editor commit path
     (`sub_631D45`, `0x631d45`, called from the checkbox's `OnItemClicked`
     handler `sub_8B9747`/`0x8b9747`) does
     `*(v6 + *(this + 44) + 25) = a4 == 0;` where `a4` is the shout
     checkbox's raw checked-state value read via
     `*(*(this + 120) + 72)`. So when the checkbox is ticked (`a4 != 0`),
     the stored field is set to `0`; `SINGLEMACRO::Encode` then writes
     `field25 != 0` → `0`. Ticked checkbox → wire byte `0`. Consistent with
     (1) and (2).
  - Both (1)+(3) are on gms_v83; (2)'s field-name evidence is independent
    (gms_v95, different binary, different symbol source). No version's
    `SINGLEMACRO::Encode`/`Decode` pair deviates from "decode stores the raw
    byte; encode writes `(stored != 0)`" (verified below, per version) — so
    the polarity generalizes to every populated version without a gate.
  - **Consequence for production** (per `baseline-bytes.md`): the shipped
    clientbound encoder (`libs/atlas-packet/model/macros.go:53`,
    `w.WriteBool(m.shout)`, upright) writes the **wrong** polarity — it
    should write `!m.shout`. The shipped serverbound decoder
    (`libs/atlas-packet/character/skill_macro.go:62`,
    `shout := !r.ReadBool()`, inverted) is **correct** as-is. This matches
    the task brief's framing that "one of the two live versions is
    corrupting saved macros today" — it is the clientbound encoder.

- **Capacity: 5** — a compile-time constant, identical on every populated
  version. Quoted from `MACROSYSDATA::Decode`, one line per version (the
  clamp immediately following the `Decode1` count read):
  - gms_v61 (`0x4b796d`, decode side): `if ( (unsigned int)v3 > 5 ) v3 = 5;` — matching encode-side clamp now confirmed at `0x4b7928` (task-226 FR-2.2): `if ( v3 > 5 ) v3 = 5;`
  - gms_v72 (`0x4d36b4`): `if ( (unsigned int)v1 > 5 ) v1 = 5;`
  - gms_v79 (`0x4db984`): `if ( (unsigned int)v1 > 5 ) v1 = 5;`
  - gms_v83 (`0x4e77b0`): `if ( v3 > 5 ) v3 = 5;`
  - gms_v84 (`0x4efc70`): `if ( (unsigned int)v1 > 5 ) v1 = 5;`
  - gms_v87 (`0x50858f`): `if ( v3 > 5 ) v3 = 5;`
  - gms_v92 (`0x4f4c50`): `if ( (unsigned int)v4 > 5 ) v4 = 5;`
  - gms_v95 (`0x4f98b0`): `if ( v4 > 5 ) v4 = 5;`
  - jms_v185 (`0x515496`): `if ( v3 > 5 ) v3 = 5;`
  - The matching send-side clamp (`MACROSYSDATA::Encode`) is byte-identical
    in shape on every version that has a send side (e.g. gms_v83,
    `0x4e776b`: `if ( v4 > 5 ) v4 = 5;`) — the client will never construct
    or transmit more than 5 entries either.
  - gms_v48: no row (function absent — see "gms_v48" below).

- **Count width: byte (`Decode1`)** — every version's leading count read is
  `CInPacket::Decode1(...)`, several with an explicit `(unsigned __int8)`
  cast on the result (e.g. gms_v61 `0x4b796d`:
  `v3 = (unsigned __int8)CInPacket::Decode1(a2);`; gms_v92 `0x4f4c50`:
  `v4 = (unsigned __int8)CInPacket::Decode1(a2);`), confirming a 1-byte wire
  field, not `Decode2`/`Decode4`.

## Field table

Every function address below was independently re-decompiled in this task
(not taken on trust from `harvest-log.md`'s table, per the controller
ruling on stale citations) via `mcp__ida-pro__decompile` against the
session ids in `harvest-log.md`'s "IDA sessions" table.

| version | fn (clientbound, addr) | fn (serverbound, addr) | count | name | shout | skillId1..3 |
|---|---|---|---|---|---|---|
| gms_v48 | NOT FOUND | NOT FOUND | — no row; both functions absent from the binary (binary-wide `func_query`/`find_regex` zero hits, no registry entries — see harvest-log.md "v48 / v61: absence, searched not assumed"). Task 4 owns the n-a decision. |
| gms_v61 | `CWvsContext::OnMacroSysDataInit` `0x849bce` → `CMacroSysMan::SetMacro` `0x59744b` → `MACROSYSDATA::Decode` `0x4b796d` → `SINGLEMACRO::Decode` `0x4b78d5` | `CMacroSysMan::FlushToSvr` `0x59746c` (renamed from `sub_59746C`, task-226 FR-2.2) → `MACROSYSDATA::Encode` `0x4b7928` (renamed from `sub_4B7928`) → `SINGLEMACRO::Encode` `0x4b7884` (renamed from `sub_4B7884`); COutPacket(101) | `Decode1`, cap 5 (`0x4b796d`: `if ((unsigned int)v3 > 5) v3 = 5;`); encode side `0x4b7928`: `Encode1(a2, v3)` where `v3` is clamped `if (v3 > 5) v3 = 5;` | `DecodeStr` (`0x4b78d5`: `CInPacket::DecodeStr(a2, &v6)`) / `EncodeStr` (`0x4b7884`: `COutPacket::EncodeStr(a2, v5[0])`) | `Decode1` raw store (`0x4b78d5`: `*(_DWORD *)(this + 25) = (unsigned __int8)CInPacket::Decode1(a2);`) / `Encode1(a2, *(unsigned int *)((char *)v2 + 25) != 0)` (`0x4b7884`) — same raw-store/`!=0` shape as every other populated version; INVERTED per the cross-version generalization above (now independently confirmed on v61's own encode side, not just the decode shape) | 3× `Decode4` (`0x4b78d5`, `do{...Decode4...}while(v4)` loop, `v4=3` init) / 3× `Encode4` (`0x4b7884`, `do{... COutPacket::Encode4 ...}while(v3)` loop, `v3=3` init) |
| gms_v72 | `CWvsContext::OnMacroSysDataInit` `0x92126b` → `CMacroSysMan::SetMacro` `0x5e39bf` → `MACROSYSDATA::Decode` `0x4d36b4` → `SINGLEMACRO::Decode` `0x4d361c` | `CMacroSysMan::FlushToSvr` `0x5e39e0` → `MACROSYSDATA::Encode` `0x4d366f` → `SINGLEMACRO::Encode` `0x4d35cb` | `Decode1`, cap 5 (`0x4d36b4`: `if ((unsigned int)v1 > 5) v1 = 5;`) | `DecodeStr`/`EncodeStr` | `Decode1`/`Encode1`, same raw-store/`!=0` shape (`0x4d361c`: `*(_DWORD *)((char *)this + 25) = (unsigned __int8)CInPacket::Decode1(a2);`; `0x4d35cb`: `COutPacket::Encode1(a2, *(unsigned int *)((char *)v2 + 25) != 0);`) | 3× `Decode4`/`Encode4` |
| gms_v79 | `CWvsContext::OnMacroSysDataInit` `0x97311a` → `CMacroSysMan::SetMacro` `0x6022ba` → `MACROSYSDATA::Decode` `0x4db984` → `SINGLEMACRO::Decode` `0x4db8ec` | `CMacroSysMan::FlushToSvr` `0x6022db` → `MACROSYSDATA::Encode` `0x4db93f` → `SINGLEMACRO::Encode` `0x4db89b` | `Decode1`, cap 5 (`0x4db984`) | `DecodeStr`/`EncodeStr` | `Decode1`/`Encode1`, same shape (`0x4db8ec`, `0x4db89b`) | 3× `Decode4`/`Encode4` |
| gms_v83 | `CWvsContext::OnMacroSysDataInit` `0xa290f8` → `CMacroSysMan::SetMacro` `0x6318f8` → `MACROSYSDATA::Decode` `0x4e77b0` → `SINGLEMACRO::Decode` `0x4e7718` | `CMacroSysMan::FlushToSvr` `0x631919` → `MACROSYSDATA::Encode` `0x4e776b` → `SINGLEMACRO::Encode` `0x4e76c7` | `Decode1`, cap 5 (`0x4e77b0`: `if (v3 > 5) v3 = 5;`) | `DecodeStr`/`EncodeStr` | `Decode1`/`Encode1`, shape as quoted in Verdicts §1 | 3× `Decode4`/`Encode4` (`do{...}while(v3)`, `v3=3` init) |
| gms_v84 | `CWvsContext::OnMacroSysDataInit` `0xa748bb` → `CMacroSysMan::SetMacro` `0x646e5e` → `MACROSYSDATA::Decode` `0x4efc70` → `SINGLEMACRO::Decode` `0x4efbd6` | `CMacroSysMan::FlushToSvr` `0x646e7f` → `MACROSYSDATA::Encode` `0x4efc2b` → `SINGLEMACRO::Encode` `0x4efb85` | `Decode1`, cap 5 (`0x4efc70`) | `DecodeStr`/`EncodeStr` (name copy uses `lstrcpynA(this+12, v3, 13)` — 12-char client-side buffer truncation; the *wire* field is still length-prefixed `DecodeStr`, not fixed-width — see "Unresolved"/name-encoding note below) | `Decode1`/`Encode1`, same shape (`0x4efbd6`, `0x4efb85`) | 3× `Decode4`/`Encode4` |
| gms_v87 | `CWvsContext::OnMacroSysDataInit` `0xac0d6e` → `CMacroSysMan::SetMacro` `0x66a4e4` → `MACROSYSDATA::Decode` `0x50858f` → `SINGLEMACRO::Decode` `0x5084f5` | `CMacroSysMan::FlushToSvr` `0x66a505` → `MACROSYSDATA::Encode` `0x50854a` → `SINGLEMACRO::Encode` `0x5084a4` | `Decode1`, cap 5 (`0x50858f`) | `DecodeStr`/`EncodeStr` (`lstrcpynA(this+12, v3->_m_pStr, 13)`) | `Decode1`/`Encode1`, same shape (`0x5084f5`, `0x5084a4`) | 3× `Decode4`/`Encode4` |
| gms_v92 | `CWvsContext::OnMacroSysDataInit` `0x9c5390` → `MACROSYSDATA::Decode` `0x4f4c50` (no `SetMacro` — inlined away, this task's one confirmed structural difference; see harvest-log.md) → `SINGLEMACRO::Decode` `0x4f4b90` | `CMacroSysMan::FlushToSvr` `0x602ed0` → `MACROSYSDATA::Encode` `0x4f4c00` → `SINGLEMACRO::Encode` `0x4f4ab0` | `Decode1`, cap 5 (`0x4f4c50`: `if ((unsigned int)v4 > 5) v4 = 5;`) | `DecodeStr`/`EncodeStr` (`lstrcpynA(this+12, *v4, 13)`) | `Decode1`/`Encode1`, same shape (`0x4f4b90`: `*(_DWORD *)((char *)this + 25) = (unsigned __int8)CInPacket::Decode1((int)v2);`; `0x4f4ab0`: `COutPacket::Encode1(v6, *(_DWORD *)((char *)v2 + 25) != 0);`) | 3× `Decode4`/`Encode4` (loop-unrolled with buffer-growth logic in `SINGLEMACRO::Encode`/`Decode` — `sub_4EF7F0` in `MACROSYSDATA::Decode` is a `ZArray` resize helper only, no wire reads, per harvest-log.md) |
| gms_v95 | `CWvsContext::OnMacroSysDataInit` `0x9f0c70` → `CMacroSysMan::SetMacro` `0x60e580` → `MACROSYSDATA::Decode` `0x4f98b0` → `SINGLEMACRO::Decode` `0x4f97f0` | `CMacroSysMan::FlushToSvr` `0x60ea20` → `MACROSYSDATA::Encode` `0x4f9860` → `SINGLEMACRO::Encode` `0x4f9710` | `Decode1`, cap 5 (`0x4f98b0`: `if (v4 > 5) v4 = 5;`) | `DecodeStr`/`EncodeStr` (`this->sName`, `lstrcpynA(this->sName, v4->_m_pStr, 13)`) | `Decode1`/`Encode1`, field named `bMute` (`0x4f97f0`: `this->bMute = CInPacket::Decode1(v2);`; `0x4f9710`: `COutPacket::Encode1(v6, v2->bMute != 0);`) | 3× `Decode4`/`Encode4`, field named `aSkill[3]` (`0x4f97f0`: `this->aSkill[i] = CInPacket::Decode4(v2);` for `i` in `[0,3)`) |
| jms_v185 | `CWvsContext::OnMacroSysDataInit` `0xb10384` → `CMacroSysMan::SetMacro` `0x6a3445` → `MACROSYSDATA::Decode` `0x515496` → `SINGLEMACRO::Decode` `0x5153fc` | `CMacroSysMan::FlushToSvr` `0x6a3466` → `MACROSYSDATA::Encode` `0x515451` → `SINGLEMACRO::Encode` `0x5153ab` | `Decode1`, cap 5 (`0x515496`: `if (v3 > 5) v3 = 5;`) | `DecodeStr`/`EncodeStr` | `Decode1`/`Encode1`, same shape (`0x5153fc`, `0x5153ab`) | 3× `Decode4`/`Encode4` |

### Top-level delegation (re-confirmed, gms_v83)

- `CMacroSysMan::FlushToSvr` (`0x631919`): constructs the `COutPacket` with
  opcode `0x6E`, calls `MACROSYSDATA::Encode(this + 8, v2)`, then
  `CClientSocket::SendPacket`. No direct `EncodeN` call of its own.
- `CWvsContext::OnMacroSysDataInit` (`0xa290f8`): `CMacroSysMan::Reset(...)`
  then `CMacroSysMan::SetMacro(instance, a2)`. No direct `DecodeN` call.
- `CMacroSysMan::SetMacro` (`0x6318f8`): `MACROSYSDATA::Decode(a2);` — a
  pure one-line forward, no direct `DecodeN` call.

This matches Task 2's finding exactly; both the address and the delegation
shape were re-verified by live decompile in this task rather than trusted
from the prior log.

## Divergences requiring a gate

**No divergence found across gms_v61..jms_v185; the layout is uniform and
the codec carries no version gate.** Every populated version (all nine, both
directions — gms_v61's send side located and confirmed task-226 FR-2.2) has
byte-identical wire shape:

```
count:     Decode1 (byte), clamped client-side to a hardcoded 5
per entry × count:
  name:      DecodeStr (length-prefixed)
  shout:     Decode1 (byte), INVERTED — wire 0 = shout on
  skillId1:  Decode4 (uint32)
  skillId2:  Decode4 (uint32)
  skillId3:  Decode4 (uint32)
```

gms_v48 is not a divergence in this sense — it has no wire format at all
(the feature doesn't exist in that client), which is a presence/absence
question for Task 4, not a field-shape gate for Task 6/7/10/11 to encode.

No fixed-width name was found on any version (a structural divergence the
task brief specifically flagged to watch for) — every version uses
`DecodeStr`/`EncodeStr` on the wire. The 12/13-byte `lstrcpynA` truncation
visible in gms_v84/v87/v92/v95's `SINGLEMACRO::Decode` is client-side
receive-buffer sizing (the client's own storage cap for a macro name), not
a wire-format difference — the wire bytes are still length-prefixed on
every version, including the ones with no visible truncation call in their
decompile (gms_v61/v72/v79/v83/jms_v185 use `lstrcpyA`/direct assignment
instead of the length-capped `lstrcpynA`, but the *wire* read is
`DecodeStr` in all ten cases without exception).

## Unresolved

None. Every field in every populated version's row above is backed by a
quoted decompiled line at a re-verified address. gms_v48's absence is a
confirmed absence (see harvest-log.md's binary-wide search evidence and
Task 4's na-recheck.md); v61's send side, previously missing, was located
and confirmed by Task 4 (na-recheck.md, feature-na-evidence.yaml supersedes
the prior "NOT FOUND" framing) — gms_v61 is now a fully-populated row, in
scope for Tasks 6-12 like every other populated version.
