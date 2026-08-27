# bug: Maple Life ops unregistered on gms_v84 (derivation.md §2.0 "VERSION-ABSENT" is wrong)

Branch: `task-246-bug-maple-life-v84-registration`, worktree
`.worktrees/task-246-bug-maple-life-v84-registration`, off `main` @ `903537fe8`.

Reported as "task-246 forgot to register `MapleLifeCheckNameHandle`,
`MapleLifeResult`, and `MapleLifeError` for GMS v84.1 and JMS 185.1."

The two halves of that report resolve differently. **gms_v84 is confirmed and
fully derived below.** **jms_v185 remains unresolved** and is NOT part of this
fix — see `## Not yet answered`.

## Reproduced

Static, against the seed data and the live IDA sessions. Not a runtime repro.

Only 4 of 11 seed templates in `services/atlas-configurations/seed-data/templates/`
carry the three entries:

| template | `MapleLifeCheckNameHandle` | `MapleLifeResult` / `MapleLifeError` | `mapleLife` block |
|---|---|---|---|
| gms_83_1 | 0x100 | 0x15D / 0x15E | present |
| gms_87_1 | 0x10E | 0x172 / 0x173 | present |
| gms_92_1 | 0x12D | 0x194 / 0x195 | present |
| gms_95_1 | 0x137 | 0x19D / 0x19E | present |
| **gms_84_1** | **absent** | **absent** | **absent** |
| jms_185_1 | absent | absent | absent |

`docs/packets/registry/gms_v84.yaml` likewise has no `MAPLELIFE_*` row, and
`docs/packets/evidence/gms_v84/` has no `maplelife.*` record.

## Observed

A GMS v84.1 tenant cannot use the Maple Life (`Cash/0543`) in-game character
creation dialog: the client's name probe reaches `atlas-channel` with no
registered handler, and the server has no writer to answer with.

## Expected

gms_v84 registers the same three ops as its neighbours, at its own opcodes.

## Root cause

`docs/tasks/task-246-maple-life-character-creation/derivation.md` §2.0 concluded
"`Cash/0543` in-game character creation via `CUICharacterSaleDlg` is
**VERSION-ABSENT on gms_v84**", and every downstream task honoured that. **The
conclusion is wrong.** It rests on three findings; two are artifacts and the
third is not load-bearing.

Session `46c2a2eb` = `E:\Programs\Nexon\IDBs_v9\GMS\v84_1\GMS_v84.1_U_DEVM.i64`
(`is_analyzing: false`). Control session `c0829805` = v87 (`GMSv87_4GB.exe.i64`).

### Why §2.0 finding #1 (`func_query` → zero matches) is an artifact

It is a symbol-coverage artifact, not an absence. `func_query name_regex=CUI`
returns **24** functions on v84 versus **~280** on v87. The v87 names are full
mangled PDB symbols (`?SendCheckDuplicateIDPacket@CUICharacterSaleDlg@@...`);
several of v84's 24 are hand-made RE artifacts from earlier tasks
(`CUIGuildBBS__OnRegister_send_0x9F`). This IDB carries almost no class symbols,
so "no `CUICharacterSaleDlg` symbol" carries no information about the class.

### Why §2.0 finding #2 (RTTI string absent) is void

§2.0 asserted the string `CharacterSaleDlg` is "present as a plain string
wherever the class exists." It is not. `find_regex` for `CharacterSale` on the
**v87** session — where the class provably exists, with 15 named methods —
returns **zero matches**. The test cannot distinguish presence from absence on
any build. §6.3 of the same document already said so ("no RTTI string on this
build, matching every GMS build checked too — expected, not informative either
way"), contradicting §2.0.

### Why the `CField` symbols misled the pass

v84's IDB symbol `?OnCharacterSale@CField@@...` at `0x5443af` is **misapplied**.
It forwards through `this[135]` (CField+0x21C), which task-129 already annotated
as `CUIItemUpgrade` (the Vicious Hammer path). v87's real
`CField::OnCharacterSale` (`0x55fa2c`) forwards through `this[136]`
(CField+0x220). The v84 Maple Life route is the *other* member — the IDB-named
`CField::OnItemUpgrade` at `0x544395` (`this[134]`), whose task-129 comment
records that it "routes 359/360". The two `CField` symbols are swapped on v84,
which is what sent §2.0 down the wrong path.

### Positive derivation (this is the load-bearing evidence)

`CUICharacterSaleDlg` **exists on gms_v84**. Located by structural fingerprint,
not by symbol. All four functions renamed in session `46c2a2eb`:

| v84 addr | new IDB name | v87 counterpart | size |
|---|---|---|---|
| `0x7fd86a` | `CUICharacterSaleDlg__SendCheckDuplicateIDPacket_send_0x107` | `0x82e04d` | `0xbf` both |
| `0x7fd845` | `CUICharacterSaleDlg__OnPacket_recv_0x167_0x168` | (virtual, vtable+0x3C) | `0x25` |
| `0x7fd949` | `CUICharacterSaleDlg__OnCheckDuplicatedIDResult_recv` | `0x82e12c` | — |
| `0x7fda6f` | `CUICharacterSaleDlg__OnCreateNewCharacterResult_recv` | `0x82e252` | — |
| `0x7c4771` | `is_valid_character_name` | `0x7f1238` | — |
| `0x9d389c` | `CUtilDlg__Notice` | `0xa195ca` | — |

The sender is instruction-for-instruction identical to v87's, which is why the
size matches exactly at `0xbf`:

| step | v87 `0x82e04d` | v84 `0x7fd86a` |
|---|---|---|
| busy guard | `!this[184]` | `!this[180]` |
| name validation | `is_valid_character_name(a2, v3 == 0)` | same, `sub_7C4771` |
| **opcode** | `COutPacket::COutPacket(v12, 0x10E)` = 270 | `COutPacket::COutPacket(v12, 263)` = **0x107** |
| payload | `COutPacket::EncodeStr(name)` | `COutPacket::EncodeStr(name)` |
| send helper | `sub_82E10C` | `sub_7FD929` |
| failure path | `GetString(5057)` → `Notice` → `(*this+32)(this, 1001)` | `GetString(5051)` → `Notice` → `(*this+32)(this, 1001)` |

**Clientbound opcodes are read directly off the dispatcher**, not inferred.
`CUICharacterSaleDlg__OnPacket` at `0x7fd845` decompiles to exactly:

```c
if ( a2 == 359 )      CUICharacterSaleDlg__OnCheckDuplicatedIDResult_recv(this, a3);
else if ( a2 == 360 ) sub_7FDA6F(a3);   // OnCreateNewCharacterResult
```

So on gms_v84: **`MAPLELIFE_RESULT` = 359 (0x167)**, **`MAPLELIFE_ERROR` = 360
(0x168)**, **`MAPLELIFE_CHECK_NAME` = 263 (0x107)**.

> Note for whoever extends this: I first *predicted* 349/350 from a
> registry-gap argument (on v83 and v87 both, `MAPLELIFE_RESULT` sits at
> `MTS_OPERATION+1`, and v84 has a matching two-slot hole at 349/350). **That
> prediction was wrong** — the dispatcher says 359/360. The gap argument is a
> lead, never a value. Do not resurrect it.

Field orders, read off the two receivers:

- `MapleLifeResult` (359) = `DecodeStr(name)` then `Decode1(result)`. Branches:
  `0` → success, vtable `(*this+32)(this, 1000)`; `> 0` → `GetString(5052)`
  ("name taken") → `1001`; negative → `GetString(5757)` formatted → `1001`.
  Matches the `AVAILABLE: 0, TAKEN: 1, UNKNOWN_ERROR: 255` table used by
  v83/v87/v92/v95 (255 as a signed char is the negative arm).
- `MapleLifeError` (360) = `Decode1(result)` then `Decode4(u32)`. Branches:
  `52` → success path; `54` → `GetString(5051)` name-taken notice; else →
  `GetString(5757)` formatted with the u32. So the operations table is
  **`SUCCESS: 52, NAME_TAKEN_AT_SUBMIT: 54, UNKNOWN_ERROR: 255`** — identical to
  gms_83_1's, as expected for an adjacent build.

Corroborating data-side check (per the user's note that the cash items exist):
`Item.wz/Cash/0543.img` is present in the v82-era pack at
`/mnt/d/Source/thepack_82_docker/wz/`, containing item `05430000`. The feature's
data exists across this version range; v84 is bracketed by v83 and v87, both of
which ship the dialog.

No opcode collisions: `docs/packets/registry/gms_v84.yaml` has 359, 360
clientbound and 263 serverbound all unoccupied.

## Fix

All paths relative to the worktree root.

- `services/atlas-configurations/seed-data/templates/template_gms_84_1.json` —
  add to `socket.handlers`:
  `{"opCode":"0x107","validator":"LoggedInValidator","handler":"MapleLifeCheckNameHandle","fname":"CUICharacterSaleDlg::SendCheckDuplicateIDPacket","services":["channel"]}`.
  Add to `socket.writers` the `MapleLifeResult` entry at `0x167` and the
  `MapleLifeError` entry at `0x168`, with `fname`s
  `CUICharacterSaleDlg::OnCheckDuplicatedIDResult` and
  `CUICharacterSaleDlg::OnCreateNewCharacterResult`, and the `options.operations`
  tables given above. Match the exact key order and formatting of the
  corresponding entries in `template_gms_83_1.json`.
- Same file — add the `mapleLife` block. gms_84_1 currently has none, and
  `supportsMapleLife` in the UI keys off the handler's presence
  (`services/atlas-ui/src/components/features/characters/maple-life/mapleLifeSupport.ts`),
  so registering the handler without the block would surface an editor over a
  null config. Model it on `template_gms_83_1.json`'s block; v84 is the adjacent
  build. **If the two templates' character/job data differ in a way that makes a
  straight copy wrong, stop and report rather than inventing entries.**
- `docs/packets/registry/gms_v84.yaml` — add `MAPLELIFE_CHECK_NAME` (serverbound
  263), `MAPLELIFE_RESULT` (clientbound 359), `MAPLELIFE_ERROR` (clientbound
  360), each `provenance: ida-discovered` with the `ida.address` values from the
  table above (`0x7fd86a`, `0x7fd949`, `0x7fda6f`).
- `docs/tasks/task-246-maple-life-character-creation/derivation.md` — append a
  §2.0-CORRECTION subsection recording that §2.0's VERSION-ABSENT finding is
  retracted and why. **Do not renumber or rewrite existing sections**; the file's
  own header forbids it.
- `libs/atlas-packet/maplelife/**` — verify whether any version gate uses a
  `MajorAtLeast` boundary that currently skips v84. If a gate excludes v84,
  correct it; if the codecs are version-agnostic, change nothing.
- Evidence + matrix: add `docs/packets/evidence/gms_v84/maplelife.*.yaml` and
  promote the three gms_v84 cells, per
  `docs/packets/audits/VERIFYING_A_PACKET.md`. This is `packet-verifier` work —
  one cell per agent — not the implementer's.

## Not yet answered

- **jms_v185 is NOT resolved and is excluded from this fix.** derivation.md §6.3
  left opcode 271 unidentified, and my pass did not close it either: a
  `push 271` scan across JMS `.text` returned 13 hits, none matching the sender
  fingerprint (size `0xbf`, `COutPacket` + `EncodeStr`). JMS's
  `CField::OnCharacterSale` (`0x57528c`) forwards through `this[133]`
  (CField+0x214) and **that member is still unidentified** — `insn_query op_any`
  matches immediates, not displacements, so it could not find the member write.
  That is the concrete next lead: identify what is stored at CField+0x214 on
  session `a977912e`, then read its vtable+0x3C dispatcher the same way
  `0x7fd845` was read on v84. Registering JMS today would mean inventing three
  opcodes. Note the standing caution: the user observes the `Cash/0543` items
  exist, which does not by itself prove the client routes them.
- Whether the v84 `mapleLife` block can be copied from gms_83_1 verbatim, or
  whether its job/appearance pools diverge. Not checked.
- No live re-test. The fix is derived from the binary and seed data only; a
  v84.1 client has not exercised the dialog against a patched tenant.

## Resolution

_Unresolved at time of writing. Record the fixing commit, the gate verdict, and
whether live testing confirmed it, before closing._
