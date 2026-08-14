# Live-tenant reconciliation — skill macros (task-226)

Post-merge input for the live socket-config PATCH. Seed templates changed on
this branch do NOT reach already-provisioned tenants
(bug_new_opcodes_not_in_live_tenant_config). Procedure:
reference_reconcile_live_tenant_socket_to_template.

## Entries to add per version

All entries below are `services: ["channel"]`. Opcodes and fnames are taken
from the seed templates landed on this branch
(`services/atlas-configurations/seed-data/templates/`), cross-checked against
the coverage-matrix registry and `layout-derivation.md`.

### gms_61 — `handlers`

Task 4's n-a re-check (`na-recheck.md`) corrected `SKILL_MACRO` × gms_v61 off
`⬜ n-a`: `CMacroSysMan::FlushToSvr` is present at `0x59746c`, sending opcode
101 (`0x65`). The template binding landed in this task's Task 9.

| opCode | handler | validator | fname | services |
|---|---|---|---|---|
| 0x65 | CharacterSkillMacroHandle | LoggedInValidator | CMacroSysMan::FlushToSvr | channel |

(gms_61's clientbound writer, `CharacterSkillMacro` @ `0x5B`, was already
bound before this task — no change needed there.)

### gms_87 — `handlers`

| opCode | handler | validator | fname | services |
|---|---|---|---|---|
| 0x71 | CharacterSkillMacroHandle | LoggedInValidator | CMacroSysMan::FlushToSvr | channel |

### gms_92 — `handlers`

| opCode | handler | validator | fname | services |
|---|---|---|---|---|
| 0x79 | CharacterSkillMacroHandle | LoggedInValidator | CMacroSysMan::FlushToSvr | channel |

### gms_92 — `writers`

| opCode | writer | fname | services |
|---|---|---|---|
| 0x8B | CharacterSkillMacro | CWvsContext::OnMacroSysDataInit | channel |

### gms_95 — `handlers`

| opCode | handler | validator | fname | services |
|---|---|---|---|---|
| 0x7A | CharacterSkillMacroHandle | LoggedInValidator | CMacroSysMan::FlushToSvr | channel |

### jms_185 — `handlers`

| opCode | handler | validator | fname | services |
|---|---|---|---|---|
| 0x69 | CharacterSkillMacroHandle | LoggedInValidator | CMacroSysMan::FlushToSvr | channel |

### gms_48 — no entries

Both `SKILL_MACRO` and `MACRO_SYS_DATA_INIT` are **CONFIRMED-NA** for gms_v48
(`na-recheck.md`): the client binary has no macro system at this version — a
full 317/317 `COutPacket::COutPacket(long)` call-site scan found zero
macro-shaped sends, and `CWvsContext::OnPacket`'s fully-enumerated compiled
switch (43 labelled cases across the 25-70 opcode range, all gaps confirmed
no-op `default:`) has zero macro-shaped receives. gms_48 needs no PATCH for
this feature; there is nothing to add.

## Shout-polarity warning — read before patching

`layout-derivation.md` (Verdicts §1) establishes that the wire's shout flag
is **INVERTED**: wire byte `0` means shout is ON, wire byte `1` means shout is
OFF. This was derived from `CMacroSysMan::IsShoutMacro` (gms_v83,
`0x631d19`), independently confirmed by gms_v95's field literally being named
`bMute` (muted = the negation of shout) in `SINGLEMACRO::Decode`/`Encode`
(`0x4f97f0`/`0x4f9710`), and by the gms_v83 macro-editor commit path
(`sub_631D45`, `0x631d45`).

Before this task, the shipped clientbound encoder
(`libs/atlas-packet/model/macros.go:53`, `w.WriteBool(m.shout)`) wrote the
polarity **upright** — the wrong way. This task fixed the encoder to write
`!m.shout`. **Consequence: on already-live gms_83 and gms_84 tenants, the
macro shout flag was being sent to clients backwards prior to this change**
(a macro saved with shout ON displayed as OFF in the client, and vice versa).
This is a **fix**, not a regression — but it is a visible behavior change on
tenants that were already receiving this packet, and testers should expect
shout-flag display to flip (correctly) once the patched build deploys. It is
not something to "roll back" if noticed.

The serverbound decoder (`skill_macro.go:62`, `shout := !r.ReadBool()`) was
already correct and is unchanged.

## Verification after the PATCH

Log in on each patched tenant, edit a macro in the Skill window, confirm, log
out and back in, and confirm the macro persisted with the shout flag in the
state it was set to. **Check the shout flag specifically**: toggle it to ON
in the client, save, reload, and confirm it still reads ON in the client UI
(not merely that the value round-trips server-side) — this is the flag whose
wire polarity this task corrected, per the warning above.
