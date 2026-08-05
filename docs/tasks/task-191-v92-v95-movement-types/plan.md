# v92/v95 Movement `types` Configuration — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make character/monster/pet/summon movement decode correctly on GMS v92 and v95 tenants by
deriving the per-index movement `types` table from both clients, fixing the v88+ movement header in
`libs/atlas-packet/model/movement.go`, correcting the mis-registered v92 summon opcode block, adding
the two missing move-handler entries, closing the v48 Summon leftover, and adding a permanent guard
so the defect class cannot recur.

**Architecture:** Movement decode in Atlas is generic: `model.Movement.Decode` reads a header, then
per fragment reads a one-byte element type and looks it up **as an array index** in the tenant socket
handler's `options.types`. The looked-up `Type` selects the concrete element decoder. So the table is
tenant configuration (seed templates), not Go code. The one Go change is the packet **header**: the
v88+ clients write four `int16` (`x,y,vx,vy`) before the element count where v83–v87 and JMS write
two — a version gate paired exactly like the existing `XOffset`/`YOffset` gate in the same file.

**Tech Stack:** Go 1.x (`libs/atlas-packet`), JSON seed templates
(`services/atlas-configurations/seed-data/templates/`), bash + `python3` guard scripts, GitHub
Actions, IDA Pro via the MCP session server for derivation.

## Global Constraints

- **Worktree.** All work happens in `.worktrees/task-191-v92-v95-movement-types` on branch
  `task-191-v92-v95-movement-types`. Never edit the main checkout.
- **Grounding (CLAUDE.md).** Every derived value — every index's `Name`/`Type`, every opcode — cites
  a client function and address. No value is carried over from a neighbouring template, from general
  MapleStory knowledge, or from this plan's Appendix A without independent re-derivation. An index
  that cannot be resolved from the client is recorded as unresolved and **escalated**, never guessed
  (PRD FR-1.6).
- **Appendix A is a design-phase derivation, not an oracle.** Task 1 re-derives the table
  independently. Where Task 1's reading differs from Appendix A, **Task 1's reading wins** and the
  divergence is recorded in `movement-types-derivation.md`.
- **No file under `services/atlas-channel/` is modified.** It is the runtime beneficiary only.
- **Templates other than `gms_48_1`, `gms_92_1`, `gms_95_1` are byte-unchanged.**
- **`CharacterInventoryMoveHandle` is untouched in every template** — it is inventory item movement,
  not map movement, and correctly carries no `types` (PRD FR-2.4).
- **Every added handler entry has `"validator": "LoggedInValidator"` and `"services": ["channel"]`.**
  A socket handler entry with a missing validator is silently dropped at load time (PRD FR-3.3).
- **Added entries go at their sorted `opCode` position**, never appended next to a semantically
  related entry (PRD FR-3.4, `docs/packets/TEMPLATE_CONVENTIONS.md`).
- **Line endings.** All three templates are LF. `template_gms_92_1.json` has **no trailing newline** —
  do not add one; it inflates the diff.
- **`Type` values.** Exactly seven are recognized by the decoder: `NORMAL`, `JUMP`, `TELEPORT`,
  `START_FALL_DOWN`, `FLYING_BLOCK`, `STAT_CHANGE`, `DEFAULT`. A typo'd value degrades that one index
  to the 3-byte generic decode **silently**.
- **`FALL_DOWN` is the only load-bearing `Name`** — it triggers the extra `FhFallStart` int16 in
  `NormalElement` (`movement.go:126-128`, mirrored at `:218-220`). At most one per array.
- **Encode/Decode symmetry.** Any version gate added to `movement.go` must be *textually identical*
  in `Decode` and `Encode`, or Atlas corrupts its own outbound movement packets.
- **Commit granularity.** Commit at the end of every task. Never leave a `// TODO`, stub, or
  half-applied template edit in a commit.

### Scope amendments to the PRD (approved during design + planning)

| # | Amendment | Source |
|---|---|---|
| 1 | `libs/atlas-packet/model/movement.go` **is** modified (v88+ `StartVx`/`StartVy` header), against PRD §2's non-goal. Without it the `types` arrays cannot take effect. | design §4 |
| 2 | v92 `SummonMoveHandle` opCode corrected `0xC8` → `0xCC`. | design §5.3 |
| 3 | **v92 `SummonAttackHandle` and `SummonDamageHandle` opcodes are derived and corrected too** — the whole 3-handler summon block is suspected stale, not just the move entry. | user decision, planning phase |
| 4 | A **permanent** guard script + CI job (not a one-off), registered in `pr-validation.yml`, CLAUDE.md, and `TEMPLATE_CONVENTIONS.md`. | user decision, planning phase |
| 5 | `docs/packets/registry/gms_v95.yaml` `MOVE_PLAYER` row: **fname promotion only**, opcode 44 (`0x2C`) is correct and stays. | design §5.1 |

---

## File Structure

**Created**

| Path | Responsibility |
|---|---|
| `docs/tasks/task-191-v92-v95-movement-types/movement-types-derivation.md` | The FR-1.4 evidence document: per version, per index — client function, address, observed field sequence, resulting `Name`/`Type`; plus the v83 control run, the header delta, and every opcode derivation. |
| `docs/tasks/task-191-v92-v95-movement-types/reconcile.md` | FR-6.3: the live-tenant reconcile procedure, environments/tenants patched, and quoted read-back output. |
| `tools/template-movement-types-guard.sh` | Permanent mechanical check of the FR-5 invariants over every `template_*.json`. |

**Modified**

| Path | Change |
|---|---|
| `libs/atlas-packet/model/movement.go` | `Movement` gains `StartVx`/`StartVy`; both gated `IsRegion("GMS") && MajorAtLeast(88)` in `Decode` (:33-64) and `Encode` (:186-198). |
| `libs/atlas-packet/model/movement_test.go` | New header-boundary test (byte-pinned) + v92/v95 round-trip. |
| `services/atlas-configurations/seed-data/templates/template_gms_92_1.json` | `types` on 4 move handlers; new `PetMovementHandle` `0xC4`; summon block `0xC8/0xC9/0xCA` → `0xCC/0xCD/0xCE`. |
| `services/atlas-configurations/seed-data/templates/template_gms_95_1.json` | `types` on 3 existing move handlers; new `CharacterMoveHandle` `0x2C`. |
| `services/atlas-configurations/seed-data/templates/template_gms_48_1.json` | `types` on `SummonMoveHandle` `0x78` (copied from that template's own `CharacterMoveHandle`). |
| `.github/workflows/pr-validation.yml` | New `template-movement-types-guard` job (:~313 area), added to `pr-validation-complete`'s `needs:` (:713) and the result rollup (:734 area). |
| `CLAUDE.md` | Build & Verification item 11 for the new guard. |
| `docs/packets/TEMPLATE_CONVENTIONS.md` | New "Rule: move handlers carry a `types` table" section + guard reference. |
| `docs/packets/registry/gms_v95.yaml` | `MOVE_PLAYER` serverbound: promote `CVecCtrlUser::EndUpdateActive` to `fname`, demote `CUserLocal::OnKey` to `fname_alts`. Opcode unchanged. |

---

## Task 1: Derive the movement tables, header delta, and opcodes from the v92/v95 clients

This task produces **evidence only** — no code, no template edits. Everything downstream depends on
its output, so it is the one task that may legitimately halt the plan (FR-1.6 escalation).

**Files:**
- Create: `docs/tasks/task-191-v92-v95-movement-types/movement-types-derivation.md`

**Interfaces:**
- Produces: the authoritative 37-entry (expected) `Name`/`Type` array for v92 and v95; the v92
  opcodes for `PetMovementHandle`, `SummonMoveHandle`, `SummonAttackHandle`, `SummonDamageHandle`;
  the v95 opcode for `CharacterMoveHandle`; the confirmed v88+ header field list. Tasks 2, 5, 6, 8
  consume these.

### Reaching IDA

The IDA-MCP **session** server is at `http://192.168.20.3:8745/mcp` (confirmed reachable from this
WSL host, HTTP 200 on `initialize`). The older `:13337/mcp` server rejects the `database` argument —
do not use it.

Handshake: POST `initialize` (capture the `Mcp-Session-Id` **response header**), POST
`notifications/initialized` as a notification (no `id`; expect 202 with an empty body — do not
JSON-parse it), then `tools/call` with `{"name": ..., "arguments": {..., "database": <session_id>}}`.

- [ ] **Step 1: Enumerate the IDBs and match by filename**

Session ids rotate; **never** hardcode one. Call `idb_list` and match on `filename`/`input_path`:

| Version | Expected filename |
|---|---|
| GMS v83 (control) | `MapleStory_dump.exe.i64` |
| GMS v87 (header control) | `GMSv87_4GB.exe.i64` |
| GMS v92 | `GMS_v92_1_DEVM.exe.i64` |
| GMS v95 | `GMS_v95.0_U_DEVM.exe.i64` |
| JMS v185 (header control) | `MapleStory_dump_SCY.exe.i64` |

Record the session id ↔ filename mapping at the top of `movement-types-derivation.md`.

**Trap (project memory):** IDA stores MSVC symbols **mangled**. `lookup_funcs("CMovePath::Encode")`
returns "Not found" every time. Use
`func_query {queries:[{name_regex:"Encode@CMovePath"}]}` → take `result[0].data[].addr` → operate by
address. `name`/`name_contains`/`name_substr` keys are silently ignored by `func_query`.

**Trap:** server responses truncate silently. A short page from `func_query`/`func_profile` is *not*
end-of-data. `xrefs_to`/`xref_query` cap around 10 results with rich output — page them.

- [ ] **Step 2: Run the v83 control to validate the method**

Decompile `CMovePath::Encode` in the v83 IDB (design cites `0x68a563`; confirm the address you get
from `func_query` matches before trusting the read). Read the `switch (nAttr)` and group each `case`
by the `Encode2`/`Encode1` sequence it emits.

Expected (design §2.2) — this is the *check*, and reproducing it is what licenses the v92/v95 read:

```
NORMAL          (x,y,vx,vy,fh; +fhFallStart when nAttr==15) : 0, 5, 15, 17
JUMP            (vx,vy)                                     : 1, 2, 6, 12, 13, 16, 18, 19, 20, 22
TELEPORT        (x,y,fh)                                    : 3, 4, 7, 8, 9, 11
STAT_CHANGE     (bStat only, no bMoveAction/tElapse)        : 10
START_FALL_DOWN (vx,vy,fhFallStart)                         : 14
DEFAULT         (nothing extra)                             : 21, and anything unlisted
```

Compare index-for-index against `template_gms_83_1.json`'s 23-entry
`CharacterMoveHandle.options.types` (read it with `python3 -c` — do not eyeball).

**Gate:** if the control does **not** reproduce v83's committed array index-for-index, STOP. The
method is not validated and no v92/v95 derivation may proceed. Report the mismatch.

- [ ] **Step 3: Derive the v95 element table from both codec halves**

Decompile `CMovePath::Encode` (design: `0x666e20`) **and** `CMovePath::Decode` (design: `0x667920`)
in the v95 IDB. `Encode` is the serverbound authority (what Atlas decodes); `Decode` is the
clientbound authority (what Atlas encodes). They must agree case-for-case — that agreement is a free
consistency check, so read both and say so in the evidence.

For every `case` value from 0 to the maximum (design says max case is `0x24`, i.e. length 37), record:
the case value, the exact `Decode2`/`Decode1`/`Encode2`/`Encode1` sequence, and the resulting `Type`
by this mapping (PRD FR-1.2):

| `Type` | Fields after the type byte |
|---|---|
| `NORMAL` | X, Y, Vx, Vy, Fh, [FhFallStart iff the index is the `FALL_DOWN` one], XOffset, YOffset |
| `TELEPORT` | X, Y, Fh |
| `START_FALL_DOWN` | Vx, Vy, FhFallStart |
| `FLYING_BLOCK` | X, Y, Vx, Vy |
| `JUMP` | Vx, Vy |
| `STAT_CHANGE` | BStat only — **no** trailing BMoveAction/TElapse |
| `DEFAULT` | nothing extra (BMoveAction + TElapse only) |

Naming policy (design §2.4 — the table was renumbered, so v83-era semantic names cannot be carried
across by index and the client exposes no name strings; inventing one would be fabrication):

| Index | `Name` | Only if justified by |
|---|---|---|
| the base/first NORMAL-group member | `NORMAL` | first NORMAL arm |
| the sole `bStat`-only arm | `STAT_CHANGE` | that arm |
| the sole `vx,vy,fhFallStart` arm | `START_FALL_DOWN` | that arm |
| the NORMAL-group arm with the inner `fhFallStart` branch | `FALL_DOWN` | **load-bearing** |
| the sole `x,y,vx,vy` arm | `FLYING_BLOCK` | that arm |
| everything else | `UNKNOWN` | — |

- [ ] **Step 4: Derive the v92 element table independently**

Same procedure against the v92 IDB (design: `Encode` `0x65a260`, `Decode` `0x65ad60`). Do **not**
assume it equals v95 — derive it, then state whether it matched. Design found both `Encode` bodies
`0x552` bytes and both `Decode` bodies `0x31e` bytes with case-for-case identical groups; if your
read disagrees, your read wins and the divergence is recorded.

- [ ] **Step 5: Confirm the v88+ header delta**

For each of v83, v87, JMS v185, v92, v95, record the header write sequence at the top of
`CMovePath::Encode` and the matching read sequence at the top of `CMovePath::Decode`:

| Version | Expected header | Design's cited addresses |
|---|---|---|
| GMS v83 | `Encode2(x)`, `Encode2(y)`, `Encode1(count)` | `0x68a563` → `0x68a57c`, `0x68a592`, `0x68a5c3` |
| GMS v87 | 2 × `Encode2` + `Encode1` | `0x6c70fe` → `0x6c7118`, `0x6c712e`, `0x6c715f` |
| JMS v185 | 2 × `Encode2` + `Encode1` | `0x70b6c4` → `0x70b6de`, `0x70b6f4`, `0x70b725` |
| GMS v92 | `Encode2(x)`, `Encode2(y)`, **`Encode2(vx)`**, **`Encode2(vy)`**, `Encode1(count)` | `0x65a260` → `0x65a284`, `0x65a29f`, `0x65a2ba`, `0x65a2d5`, `0x65a306` |
| GMS v95 | same 4 + count | `0x666e20` → `0x666e44`, `0x666e5f`, `0x666e7a`, `0x666e95`, `0x666ec6` |

The **JMS read is not optional** — Task 2's gate excludes JMS, and that exclusion rests on this
evidence. If JMS turns out to write the four-field header, Task 2's predicate changes to
`MajorAtLeast(88)` without the region clause; report it and adjust.

- [ ] **Step 6: Derive every opcode by walking xrefs into `CMovePath::Flush`**

v95 `Flush` is at `0x668160`, v92 at `0x65b5a0` (per design; confirm). Six senders per client (mob,
npc, dragon, pet, summoned, user). For each, read the `COutPacket(<imm>)` constructor immediate at
the sender and record address + immediate.

v95 (named symbols):

| Role | v95 function | Expected opcode |
|---|---|---|
| Mob | `CMob::GenerateMovePath` `0x651100` | 227 = `0xE3` (matches template) |
| Pet | `CVecCtrlPet::EndUpdateActive` `0x99f5a0` | 199 = `0xC7` (matches template) |
| Summoned | `CVecCtrlSummoned::EndUpdateActive` `0x9a0700` | 207 = `0xCF` (matches template) |
| **User** | `CVecCtrlUser::EndUpdateActive` `0x9a0d20`, `COutPacket(44)` @`0x9a0ee3` | **44 = `0x2C`** ← the FR-3.1 answer |

Three of those four already match `template_gms_95_1.json`, which cross-validates the walk before it
is used to derive the new value.

v92 (senders are `sub_*`; resolve by positional correspondence against the v95 named set, then
**confirm structurally** — positional order alone is not evidence):

| Role | v92 function (design) | Expected opcode | Structural confirmation |
|---|---|---|---|
| Mob | `sub_6447A0` | `0xDC` (matches template) | — |
| Npc | `sub_664DC0` | `0xEA` | not routed by Atlas |
| Dragon | `sub_96F190` | `0xD3` | neither v92 nor v95 encodes a prefix |
| **Pet** | `sub_9781A0` | **`0xC4`** | both `EncodeBuffer(petLockerSN, 8)` before `Flush` |
| **Summoned** | `sub_9792D0` @`0x97932d` | **`0xCC`** | both `Encode4(dwSummonedID)` before `Flush` |
| User | `sub_9798F0` | `0x2E` (matches template) | both write the 8-field anti-cheat header |

- [ ] **Step 7: Derive the rest of the v92 summon block and settle what owns `0xC8`**

`template_gms_92_1.json` currently registers `SummonMoveHandle 0xC8`, `SummonAttackHandle 0xC9`,
`SummonDamageHandle 0xCA`. If the summon *move* opcode is really `0xCC`, the neighbouring two are
suspect too. In `template_gms_95_1.json` the same three are `0xCF`/`0xD0`/`0xD1`, and v95's
`0xC8`/`0xC9`/`0xCA` are `PetChatHandle`/`PetCommandHandle`/`PetDropPickUpHandle` — i.e. under a
v95−3 correspondence, v92's `0xC8`/`0xC9` would be `PetItemUseHandle`/`PetItemExcludeHandle`.
**That is a hypothesis to test against the client, not a conclusion.**

Derive from the v92 client:

1. The v92 summon **attack** send site (v95 counterpart: `CSummoned::TryDoingAttackManual` /
   equivalent — locate via `func_query {queries:[{name_regex:"Summoned"}]}` in v95, then position-map
   to v92 and confirm structurally) → record its `COutPacket` immediate.
2. The v92 summon **damage/hit** send site, same way → record its immediate.
3. What, if anything, the v92 client sends on `0xC8` (and `0xC9`, `0xCA`). Search the v92 `.text`
   for `COutPacket` constructions with those immediates and identify the enclosing function.

Record all three findings with addresses. If a real, Atlas-routed v92 handler legitimately collides
at `0xC8`/`0xC9`/`0xCA`, **STOP and escalate** (design §5.3) rather than silently overwriting.

- [ ] **Step 8: Name the v92 `sub_*` functions in the IDB**

Per CLAUDE.md's RE discipline, rename each v92 `sub_*` resolved in Steps 6–7 to its v95 mangled
symbol (`rename` tool with the session's `database`; `dir:"vibe"` in the output means it took), so
the addresses cited in the evidence document resolve to names on re-read. List every rename applied.

- [ ] **Step 9: Write `movement-types-derivation.md`**

Required sections:

1. **IDB session map** — session id ↔ filename, and the date. Note that session ids rotate.
2. **Method + v83 control** (Step 2) — the reproduction, index-for-index, against the committed v83
   array.
3. **v95 element table** — a row per index 0..N: index, client function + address, observed field
   sequence, resulting `Name`, resulting `Type`.
4. **v92 element table** — same, derived independently, plus an explicit statement of whether it
   equals the v95 table.
5. **Header delta** (Step 5) — the five-version table with addresses, including the JMS read that
   licenses the region clause in Task 2's gate.
6. **Opcodes** (Steps 6–7) — every opcode cited to a `COutPacket` constructor address, including the
   v92 summon-block findings and the `0xC8` ownership answer.
7. **IDB renames applied** (Step 8).
8. **Unresolved indices** — explicitly "none" if none. Any entry here is an FR-1.6 escalation.
9. **Divergences from this plan's Appendix A** — explicitly "none" if none.
10. **Matrix note** — the existing `packet-audit:verify … version=gms_v95` markers on the movement
    fixtures pinned a header layout the v95 client does not read (the known "round-trip fixture ≠
    client-validated" failure mode). This task does **not** open a fixture campaign (PRD §2 non-goal);
    record the correction here so a later matrix pass has the evidence.
11. **Client-option coupling** — v92/v95 read two extra `int16` (`usRandCnt`, `usActualRandCnt`) per
    element when `CClientOptMan::GetOpt(2)` is truthy (v95 `0x667a57`). `GetOpt` @`0x4ac700` returns 0
    for any key absent from `m_mOpt`, and `m_mOpt` is populated only by `CClientOptMan::DecodeOpt`
    @`0x4acb30` from a server-sent option list. Atlas writes a zero option count
    (`libs/atlas-packet/.../set_field.go` — `w.WriteShort(0)`), so the map stays empty and those
    fields are never on the wire. Confirm the `GetOpt`/`DecodeOpt` reads and record the coupling: if
    Atlas ever sends a non-empty client-option list including type 2, `movement.go` must gain those
    two reads.

- [ ] **Step 10: Commit**

```bash
git add docs/tasks/task-191-v92-v95-movement-types/movement-types-derivation.md
git commit -m "docs(task-191): derive v92/v95 movement types, header delta, and move opcodes from client"
```

---

## Task 2: Add the v88+ `StartVx`/`StartVy` movement header

**Files:**
- Modify: `libs/atlas-packet/model/movement.go:27-31` (struct), `:33-64` (`Decode`), `:186-198` (`Encode`)
- Test: `libs/atlas-packet/model/movement_test.go`

**Interfaces:**
- Consumes: Task 1's header delta (Step 5) — the four-field header at v92/v95, the two-field header
  at v83/v87/JMS.
- Produces: `model.Movement{StartX, StartY, StartVx, StartVy int16; Elements []MovementCodec}`. Field
  names `StartVx`/`StartVy` are referenced by nothing else in this plan, but the struct is consumed
  by `services/atlas-channel` (pass-through only — it re-encodes the decoded value, so no channel
  change is needed).

**Gate:** do not start until `movement-types-derivation.md` §5 confirms the delta *and* the JMS
exclusion. If JMS writes the four-field header, use `MajorAtLeast(88)` alone and say so in the commit.

- [ ] **Step 1: Write the failing test**

Append to `libs/atlas-packet/model/movement_test.go`. Note this file currently imports
`"github.com/Chronicle20/atlas/libs/atlas-packet/test"` unaliased plus `logrus` and
`atlas-socket/request`; add `"bytes"` to the import block.

```go
// TestMovementHeaderVersionBoundary pins the CMovePath header across the v88
// rework. v83/v84/v87 and JMS write x,y,count; v92/v95 write x,y,vx,vy,count.
//
// Evidence (movement-types-derivation.md §5):
//   v83 CMovePath::Encode@0x68a563 — Encode2 @0x68a57c, @0x68a592, Encode1 @0x68a5c3
//   v87 @0x6c70fe                  — Encode2 @0x6c7118, @0x6c712e, Encode1 @0x6c715f
//   jms @0x70b6c4                  — Encode2 @0x70b6de, @0x70b6f4, Encode1 @0x70b725
//   v92 @0x65a260                  — Encode2 @0x65a284,@0x65a29f,@0x65a2ba,@0x65a2d5, Encode1 @0x65a306
//   v95 @0x666e20                  — Encode2 @0x666e44,@0x666e5f,@0x666e7a,@0x666e95, Encode1 @0x666ec6
//
// The four header bytes precede the element count, so with them unread numElems
// is parsed out of the low byte of vx and every subsequent read is garbage —
// no matter how correct the configured `types` table is.
func TestMovementHeaderVersionBoundary(t *testing.T) {
	build := func() *Movement {
		return &Movement{StartX: 100, StartY: 200, StartVx: 5, StartVy: -3}
	}
	encode := func(region string, major uint16) []byte {
		ctx := test.CreateContext(region, major, 1)
		return test.Encode(t, ctx, build().Encode, nil)
	}

	// Pre-rework: x(2) + y(2) + count(1) = 5 bytes. vx/vy are NOT on the wire.
	wantShort := []byte{0x64, 0x00, 0xC8, 0x00, 0x00}
	for _, v := range []struct {
		region string
		major  uint16
	}{
		{"GMS", 83}, {"GMS", 84}, {"GMS", 87}, {"JMS", 185},
	} {
		if got := encode(v.region, v.major); !bytes.Equal(got, wantShort) {
			t.Errorf("%s v%d header = % x, want % x (2-field header)", v.region, v.major, got, wantShort)
		}
	}

	// v88+: x(2) + y(2) + vx(2) + vy(2) + count(1) = 9 bytes, little-endian.
	wantLong := []byte{0x64, 0x00, 0xC8, 0x00, 0x05, 0x00, 0xFD, 0xFF, 0x00}
	for _, major := range []uint16{92, 95} {
		if got := encode("GMS", major); !bytes.Equal(got, wantLong) {
			t.Errorf("GMS v%d header = % x, want % x (4-field header)", major, got, wantLong)
		}
	}
}

// TestMovementHeaderRoundTrip proves Decode mirrors Encode exactly on both sides
// of the gate. A one-sided gate silently corrupts Atlas's own outbound packets.
func TestMovementHeaderRoundTrip(t *testing.T) {
	for _, v := range []struct {
		name   string
		region string
		major  uint16
		wantVx int16
		wantVy int16
	}{
		{"GMS v87 drops vx/vy", "GMS", 87, 0, 0},
		{"JMS v185 drops vx/vy", "JMS", 185, 0, 0},
		{"GMS v92 carries vx/vy", "GMS", 92, 5, -3},
		{"GMS v95 carries vx/vy", "GMS", 95, 5, -3},
	} {
		t.Run(v.name, func(t *testing.T) {
			ctx := test.CreateContext(v.region, v.major, 1)
			in := &Movement{StartX: 100, StartY: 200, StartVx: 5, StartVy: -3}
			out := &Movement{}
			// No unconsumed bytes proves the header is sized identically both ways.
			test.RoundTrip(t, ctx, in.Encode, out.Decode, nil)
			if out.StartX != 100 || out.StartY != 200 {
				t.Errorf("start position lost: x=%d y=%d", out.StartX, out.StartY)
			}
			if out.StartVx != v.wantVx || out.StartVy != v.wantVy {
				t.Errorf("start velocity = (%d,%d), want (%d,%d)", out.StartVx, out.StartVy, v.wantVx, v.wantVy)
			}
		})
	}
}
```

- [ ] **Step 2: Run the tests and verify they fail**

```bash
go test ./model/ -run 'TestMovementHeader' -v
```
Working directory: `libs/atlas-packet`.
Expected: **compile failure** — `unknown field StartVx in struct literal of type Movement`.

- [ ] **Step 3: Add the struct fields**

In `libs/atlas-packet/model/movement.go`, replace the `Movement` struct (currently `:27-31`):

```go
type Movement struct {
	StartX int16
	StartY int16
	// StartVx/StartVy are GMS v88+ only — see the gate in Decode/Encode.
	StartVx  int16
	StartVy  int16
	Elements []MovementCodec
}
```

- [ ] **Step 4: Gate the reads in `Decode`**

In `Movement.Decode`, immediately after `m.StartY = r.ReadInt16()` and before `numElems := r.ReadByte()`:

```go
		// StartVx/StartVy are GMS v88+ — the same client movement rework that
		// added XOffset/YOffset to NormalElement (see the gate at NormalElement
		// .Decode). v83/v84/v87 and JMS write x,y,count only:
		//   v83 CMovePath::Encode@0x68a563, v87 @0x6c70fe, jms @0x70b6c4 — 2 Encode2 + Encode1.
		//   v92 @0x65a260, v95 @0x666e20 — 4 Encode2 + Encode1.
		// NOTE the predicate shape: this is IsRegion("GMS") && MajorAtLeast(88),
		// NOT the !IsRegion("GMS") || MajorAtLeast(88) shape used for
		// XOffset/YOffset. JMS v185 was checked directly (@0x70b6c4) and writes
		// the TWO-field header, so JMS is EXCLUDED here even though it is
		// INCLUDED by the XOffset gate. Reusing that predicate by reflex breaks
		// JMS movement.
		//
		// Boundary 88 vs 92: observed v87 no, v92 yes; v88..v91 have no IDB.
		// 88 is chosen for consistency with the adjacent XOffset/YOffset gate,
		// which pins the same rework — and deploy/k8s/base/versions.json ships
		// no GMS version between 87 and 92, so the two are behaviourally
		// indistinguishable for every tenant Atlas can serve.
		//
		// MUST stay textually identical to Encode.
		if t.IsRegion("GMS") && t.MajorAtLeast(88) {
			m.StartVx = r.ReadInt16()
			m.StartVy = r.ReadInt16()
		}
```

`Movement.Decode` does not currently resolve a tenant. Add it at the top of the method, matching
`NormalElement.Decode`'s existing shape (`:118-120`) — outside the returned closure:

```go
func (m *Movement) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
```

- [ ] **Step 5: Gate the writes in `Encode`**

In `Movement.Encode`, immediately after `w.WriteInt16(m.StartY)` and before
`w.WriteByte(byte(len(m.Elements)))`:

```go
		// StartVx/StartVy are GMS v88+. Paired with the Decode boundary; the two
		// MUST stay textually identical. JMS is EXCLUDED (jms CMovePath::Encode
		// @0x70b6c4 writes the two-field header) — do not reuse the
		// XOffset/YOffset predicate shape here.
		if t.IsRegion("GMS") && t.MajorAtLeast(88) {
			w.WriteInt16(m.StartVx)
			w.WriteInt16(m.StartVy)
		}
```

`Movement.Encode` also needs the tenant; add it beside the existing writer construction (`:186-188`):

```go
func (m *Movement) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
```

- [ ] **Step 6: Run the new tests and verify they pass**

```bash
go test ./model/ -run 'TestMovementHeader' -v
```
Expected: PASS for all cases, including the four `GMS v87 drops vx/vy` / `JMS v185 drops vx/vy`
sub-tests.

- [ ] **Step 7: Run the full library suite — the regression net**

```bash
go test -race ./... && go vet ./...
```
Working directory: `libs/atlas-packet`.

Pay particular attention to these, which exercise movement across every tenant variant (and now
include GMS v92 and v95 via `test.Variants`):

- `model/movement_test.go`, `model/version_bounds_test.go`
- `character/clientbound/movement_test.go`, `character/serverbound/move_test.go`
- `monster/clientbound/movement_test.go`, `monster/serverbound/movement_test.go`
- `pet/clientbound/movement_test.go`, `pet/serverbound/movement_test.go`

These wrapper fixtures assert the move-path blob **equals** `model.Movement.Encode`'s output rather
than pinning literal bytes, so they should stay green without edits — but run them, do not assume.
The one literal-byte movement fixture (`TestMonsterMovementBytesV79`) is at v79, below the gate.

Expected: all PASS, `go vet` silent. If a v92/v95 sub-test fails, the gate is one-sided — recheck
Steps 4 and 5 are textually identical.

- [ ] **Step 8: Commit**

```bash
git add libs/atlas-packet/model/movement.go libs/atlas-packet/model/movement_test.go
git commit -m "fix(packet): read/write the GMS v88+ movement header vx/vy (task-191)"
```

---

## Task 3: Add the movement-`types` invariant guard, demonstrated RED

The guard is written and proven to **fail on the current tree** before any template is fixed. A guard
that never demonstrated a failure is not a verified guard. CI wiring lands in Task 7, after the
templates are green — so this commit does not break the branch's CI.

**Files:**
- Create: `tools/template-movement-types-guard.sh`

**Interfaces:**
- Produces: `tools/template-movement-types-guard.sh` — run from the repo root, exits 0 on success,
  non-zero with per-violation diagnostics otherwise. Tasks 4, 5, 6, 7, 9 run it.

- [ ] **Step 1: Write the guard**

Create `tools/template-movement-types-guard.sh`. Modelled on `tools/template-opcode-order-guard.sh`
(bash preamble + inlined `python3` heredoc, no Go toolchain, run from the repo root, non-empty
diagnostics → non-zero exit):

```bash
#!/usr/bin/env bash
# template-movement-types-guard.sh — enforces that every tenant socket-config
# template gives every MOVE handler a well-formed movement `types` table.
#
# Rationale: libs/atlas-packet/model/movement.go decodes a movement fragment by
# reading a one-byte element type and looking it up AS AN ARRAY INDEX in the
# handler's options.types. The entry's `Type` selects the concrete element
# decoder. When `types` is missing the lookup returns ("NOT_FOUND", "DEFAULT"),
# no decoder branch matches DEFAULT, and every fragment falls through to the
# bare 3-byte Element decoder — desyncing the reader against a fragment that is
# 9-15 bytes wide. When a single `Type` value is TYPO'D the same thing happens
# for that one index, silently and with no log line.
#
# This defect has now shipped twice (task-179 fixed v48/61/72/79 but missed
# v48's SummonMoveHandle; the v92/v95 templates were seeded with no types at
# all), which is why this is a permanent guard and not a one-off check.
#
# CharacterInventoryMoveHandle is deliberately NOT a move handler — it is
# inventory item movement and correctly carries no types.
#
# See docs/packets/TEMPLATE_CONVENTIONS.md. Pure shell + python3, no Go setup.
# Run from the repo root; non-empty diagnostics → non-zero exit.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
TEMPLATE_DIR="$ROOT/services/atlas-configurations/seed-data/templates"

python3 - "$TEMPLATE_DIR" <<'PY'
import glob, json, os, sys

tmpl_dir = sys.argv[1]

MOVE_HANDLERS = {
    "CharacterMoveHandle",
    "MonsterMovementHandle",
    "PetMovementHandle",
    "SummonMoveHandle",
}
VALID_TYPES = {
    "NORMAL", "JUMP", "TELEPORT", "START_FALL_DOWN",
    "FLYING_BLOCK", "STAT_CHANGE", "DEFAULT",
}

bad = 0
checked = 0

paths = sorted(glob.glob(os.path.join(tmpl_dir, "template_*.json")))

# Floor check: a wrong/empty template dir would otherwise make this guard pass
# VACUOUSLY (0 files -> 0 violations -> exit 0), which is the worst outcome for
# a guard. There are 11 templates today; the floor only has to catch "the glob
# found nothing/almost nothing", so it does not need bumping per new template.
if len(paths) < 5:
    print("FAIL: found only %d template_*.json under %s — expected the full set."
          " The template directory moved or the glob is wrong; refusing to pass"
          " vacuously." % (len(paths), tmpl_dir))
    sys.exit(1)

for path in paths:
    name = os.path.basename(path)
    try:
        d = json.load(open(path))
    except Exception as e:
        print("PARSE ERROR: %s: %s" % (name, e))
        bad += 1
        continue

    arrays = {}  # handler -> serialized types, for the intra-template equality check
    for e in d.get("socket", {}).get("handlers", []) or []:
        if not isinstance(e, dict):
            continue
        h = e.get("handler")
        if h not in MOVE_HANDLERS:
            continue
        checked += 1
        types = (e.get("options") or {}).get("types")

        # (1) present and non-empty
        if not isinstance(types, list) or len(types) == 0:
            print("MISSING TYPES: %s %s (opCode %s) has no non-empty options.types"
                  % (name, h, e.get("opCode")))
            bad += 1
            continue

        # (2) well-formed entries with a recognized Type
        fall_down = 0
        for i, entry in enumerate(types):
            if not isinstance(entry, dict):
                print("BAD ENTRY: %s %s index %d is %s, want an object"
                      % (name, h, i, type(entry).__name__))
                bad += 1
                continue
            n, ty = entry.get("Name"), entry.get("Type")
            if not isinstance(n, str) or not isinstance(ty, str):
                print("BAD ENTRY: %s %s index %d needs string Name and Type, got %r/%r"
                      % (name, h, i, n, ty))
                bad += 1
                continue
            if ty not in VALID_TYPES:
                print("BAD TYPE: %s %s index %d has Type %r, not one of %s"
                      % (name, h, i, ty, sorted(VALID_TYPES)))
                bad += 1
            if n == "FALL_DOWN":
                fall_down += 1

        # (3) at most one FALL_DOWN (the only Name the decoder branches on)
        if fall_down > 1:
            print("DUPLICATE FALL_DOWN: %s %s has %d entries named FALL_DOWN, want at most 1"
                  % (name, h, fall_down))
            bad += 1

        arrays[h] = json.dumps(types, sort_keys=True)

    # (4) all move-handler arrays within one template are identical
    if len(set(arrays.values())) > 1:
        print("DIVERGENT TYPES: %s move handlers disagree: %s"
              % (name, ", ".join("%s(%d entries)" % (h, len(json.loads(v)))
                                 for h, v in sorted(arrays.items()))))
        bad += 1

if bad:
    print("")
    print("FAIL: %d movement-types violation(s). Every move handler needs a well-formed,"
          " template-consistent options.types. See docs/packets/TEMPLATE_CONVENTIONS.md." % bad)
    sys.exit(1)
print("OK: %d move handlers across %d templates carry a valid movement types table."
      % (checked, len(paths)))
PY
```

```bash
chmod +x tools/template-movement-types-guard.sh
```

- [ ] **Step 2: Run it and verify it FAILS, naming exactly the known-bad cells**

```bash
tools/template-movement-types-guard.sh; echo "exit=$?"
```
Working directory: repo root.

Expected — this exact output, verified by dry-running the guard logic against the current tree during
planning:

```
MISSING TYPES: template_gms_48_1.json SummonMoveHandle (opCode 0x78) has no non-empty options.types
MISSING TYPES: template_gms_92_1.json CharacterMoveHandle (opCode 0x2E) has no non-empty options.types
MISSING TYPES: template_gms_92_1.json SummonMoveHandle (opCode 0xC8) has no non-empty options.types
MISSING TYPES: template_gms_92_1.json MonsterMovementHandle (opCode 0xDC) has no non-empty options.types
MISSING TYPES: template_gms_95_1.json PetMovementHandle (opCode 0xC7) has no non-empty options.types
MISSING TYPES: template_gms_95_1.json SummonMoveHandle (opCode 0xCF) has no non-empty options.types
MISSING TYPES: template_gms_95_1.json MonsterMovementHandle (opCode 0xE3) has no non-empty options.types

FAIL: 7 movement-types violation(s). Every move handler needs a well-formed, template-consistent options.types. See docs/packets/TEMPLATE_CONVENTIONS.md.
exit=1
```

Seven lines, and **no** `DIVERGENT TYPES` lines: a handler that fails `MISSING TYPES` hits `continue`
before it is added to `arrays`, so the surviving arrays in each template are self-consistent.
`template_gms_92_1.json`'s `PetMovementHandle` and `template_gms_95_1.json`'s `CharacterMoveHandle` do
not appear because those entries do not exist yet — they are added in Tasks 5 and 6.

**Gate:** if the guard reports anything *other* than these seven lines, stop and investigate — either
the guard is wrong or `current-state.md`'s baseline survey was.

- [ ] **Step 2b: Prove the floor check works (no vacuous pass)**

```bash
mkdir -p /tmp/claude-1000/task-191-empty
python3 - /tmp/claude-1000/task-191-empty <<'PY'
import subprocess, sys, re
src = open("tools/template-movement-types-guard.sh").read()
body = src.split("<<'PY'", 1)[1].rsplit("PY", 1)[0]
r = subprocess.run([sys.executable, "-c", body, sys.argv[1]], capture_output=True, text=True)
print(r.stdout, r.stderr, "rc=%d" % r.returncode)
PY
```
Expected: `FAIL: found only 0 template_*.json under …` and `rc=1`. Without the floor check this would
print `OK: 0 move handlers …` and exit 0 — a guard that passes because it looked nowhere.

- [ ] **Step 3: Verify it does not flag `CharacterInventoryMoveHandle`**

```bash
tools/template-movement-types-guard.sh 2>&1 | grep -c CharacterInventoryMoveHandle
```
Expected: `0`. That handler appears in every template with no `types` and is correctly excluded.

- [ ] **Step 4: Commit**

```bash
git add tools/template-movement-types-guard.sh
git commit -m "test(tools): add movement-types template guard (currently RED, task-191)"
```

---

## Task 4: Close the v48 `SummonMoveHandle` leftover (FR-4)

The smallest template fix and the only one needing no derivation: v48 has a valid in-template source.

**Files:**
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_48_1.json`

**Interfaces:**
- Consumes: `tools/template-movement-types-guard.sh` from Task 3.

- [ ] **Step 1: Read the source array**

```bash
sed -n '140,241p' services/atlas-configurations/seed-data/templates/template_gms_48_1.json
```

That is the complete `CharacterMoveHandle` entry (opCode `0x21`). Lines 145–240 are its
`"options": { "types": [ … ] }` block — 23 entries, indentation: `"options"` at 8 spaces, `"types"`
at 10, each entry object at 12, entry keys at 14.

- [ ] **Step 2: Read the target entry**

```bash
sed -n '732,738p' services/atlas-configurations/seed-data/templates/template_gms_48_1.json
```

Expected:
```json
      {
        "opCode": "0x78",
        "validator": "LoggedInValidator",
        "handler": "SummonMoveHandle",
        "services": ["channel"]
      },
```

- [ ] **Step 3: Splice the array in**

Use Edit. Change `"services": ["channel"]` → `"services": ["channel"],` on the `SummonMoveHandle`
entry and append the `"options": { … }` block copied **verbatim** from lines 145–240, at the same
indentation. The `old_string` must include the `"handler": "SummonMoveHandle",` line so the match is
unique.

Do not retype the 23 entries by hand — copy the exact text read in Step 1.

- [ ] **Step 4: Verify the two arrays are byte-identical**

```bash
python3 -c "
import json
d=json.load(open('services/atlas-configurations/seed-data/templates/template_gms_48_1.json'))
h={e['handler']:(e.get('options') or {}).get('types') for e in d['socket']['handlers'] if e.get('handler','').endswith(('MoveHandle','MovementHandle'))}
for k,v in h.items(): print(k, len(v) if v else None)
ref=h['CharacterMoveHandle']
print('all identical:', all(h[k]==ref for k in ('CharacterMoveHandle','PetMovementHandle','SummonMoveHandle','MonsterMovementHandle')))
"
```
Expected: `CharacterMoveHandle 23`, `PetMovementHandle 23`, `SummonMoveHandle 23`,
`MonsterMovementHandle 23`, `CharacterInventoryMoveHandle None`, and `all identical: True`.

- [ ] **Step 5: Confirm the guard no longer flags v48**

```bash
tools/template-movement-types-guard.sh 2>&1 | grep gms_48 || echo "v48 clean"
```
Expected: `v48 clean`. (The guard still exits 1 overall — v92/v95 are fixed in Tasks 5 and 6.)

- [ ] **Step 6: Confirm the diff is confined to that one entry**

```bash
git diff --stat services/atlas-configurations/seed-data/templates/
```
Expected: only `template_gms_48_1.json`, ~96 insertions, 1 deletion (the `"services"` line gaining
its comma).

- [ ] **Step 7: Commit**

```bash
git add services/atlas-configurations/seed-data/templates/template_gms_48_1.json
git commit -m "fix(templates): give v48 SummonMoveHandle its movement types (task-179 leftover, task-191)"
```

---

## Task 5: Fix `template_gms_92_1.json`

**Files:**
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_92_1.json`

**Interfaces:**
- Consumes: Task 1's derived v92 table (Steps 3–4) and v92 opcodes (Steps 6–7); the guard from Task 3.

**Gate:** use the table and opcodes from `movement-types-derivation.md`. Appendix A below is the
design-phase derivation — if the evidence document differs, the evidence document wins.

Target end state:

| Handler | opCode | Change |
|---|---|---|
| `CharacterMoveHandle` | `0x2E` (confirmed) | add `types` |
| `PetMovementHandle` | `0xC4` (**new entry**) | add whole entry with `types` |
| `SummonMoveHandle` | `0xC8` → **`0xCC`** | correct opCode + add `types` |
| `SummonAttackHandle` | `0xC9` → **derived** (design-phase hypothesis: `0xCD`) | correct opCode |
| `SummonDamageHandle` | `0xCA` → **derived** (design-phase hypothesis: `0xCE`) | correct opCode |
| `MonsterMovementHandle` | `0xDC` (confirmed) | add `types` |

- [ ] **Step 1: Re-read the current handler block**

```bash
python3 -c "
import json
d=json.load(open('services/atlas-configurations/seed-data/templates/template_gms_92_1.json'))
for i,e in enumerate(d['socket']['handlers']):
    print(i, e['opCode'], e.get('handler'))
" | sed -n '25,46p'
```
Expected around indices 40–45: `0x8D CharacterInteractionHandle`, `0xC8 SummonMoveHandle`,
`0xC9 SummonAttackHandle`, `0xCA SummonDamageHandle`, `0xDC MonsterMovementHandle`,
`0xEA NPCActionHandle`.

- [ ] **Step 2: Correct the three summon opcodes**

Edit the three `opCode` values in place — `0xC8` → `0xCC`, `0xC9` → `0xCD`, `0xCA` → `0xCE` (using
the derived values, not these if they differ). Ascending order is preserved: `0x8D` < `0xCC` <
`0xCD` < `0xCE` < `0xDC`. Nothing needs to move.

Add a note to the commit body recording that `0xC8` was v92's *stale* summon-move opcode and what
the derivation found actually owns it.

- [ ] **Step 3: Add the `PetMovementHandle` entry at its sorted position**

`0xC4` sorts after `0x8D CharacterInteractionHandle` and before `0xCC SummonMoveHandle`. Insert with
Edit, anchoring on the `CharacterInteractionHandle` entry's closing `},`:

```json
      {
        "opCode": "0xC4",
        "validator": "LoggedInValidator",
        "handler": "PetMovementHandle",
        "services": ["channel"],
        "options": {
          "types": [
```
… Appendix A verbatim …
```json
          ]
        }
      },
```

- [ ] **Step 4: Add `types` to the other three move handlers**

For each of `CharacterMoveHandle` (`0x2E`), `SummonMoveHandle` (now `0xCC`), and
`MonsterMovementHandle` (`0xDC`): change that entry's `"services": ["channel"]` →
`"services": ["channel"],` and append the `"options": { "types": [ … ] }` block with **Appendix A
verbatim** at the same indentation used in Task 4 (`"options"` at 8 spaces, `"types"` at 10, entry
objects at 12, entry keys at 14). Anchor each `old_string` on its `"handler": "…"` line for
uniqueness.

- [ ] **Step 5: Verify structurally**

```bash
python3 -c "
import json
p='services/atlas-configurations/seed-data/templates/template_gms_92_1.json'
d=json.load(open(p))
h=d['socket']['handlers']
mv={e['handler']:(e.get('options') or {}).get('types') for e in h if e.get('handler') in
    ('CharacterMoveHandle','MonsterMovementHandle','PetMovementHandle','SummonMoveHandle')}
for k in sorted(mv): print(k, len(mv[k]) if mv[k] else None)
ref=mv['CharacterMoveHandle']
print('identical:', all(v==ref for v in mv.values()))
print('validators ok:', all(e.get('validator')=='LoggedInValidator' and e.get('services')==['channel']
      for e in h if e.get('handler') in mv))
codes=[int(e['opCode'],16) for e in h]
print('ascending:', codes==sorted(codes))
print('summon block:', [(e['opCode'],e['handler']) for e in h if 'Summon' in e.get('handler','')])
print('inventory untouched:', [(e['opCode'],(e.get('options') or {}).get('types')) for e in h if e.get('handler')=='CharacterInventoryMoveHandle'])
"
```
Expected: all four move handlers report the derived length (37 per Appendix A), `identical: True`,
`validators ok: True`, `ascending: True`, the summon block at the derived opcodes, and
`inventory untouched: [('0x4E', None)]`.

- [ ] **Step 6: Confirm no trailing newline was introduced**

```bash
python3 -c "
b=open('services/atlas-configurations/seed-data/templates/template_gms_92_1.json','rb').read()
print('trailing newline:', b.endswith(b'\n'), '(must be False)')
print('CRLF present:', b'\r\n' in b, '(must be False)')
"
```
Expected: `trailing newline: False`, `CRLF present: False`.

- [ ] **Step 7: Run both template guards**

```bash
tools/template-opcode-order-guard.sh; echo "order exit=$?"
tools/template-movement-types-guard.sh 2>&1 | grep gms_92 || echo "v92 clean"
```
Expected: `order exit=0`, `v92 clean`. (The movement guard still exits 1 — v95 remains.)

- [ ] **Step 8: Commit**

```bash
git add services/atlas-configurations/seed-data/templates/template_gms_92_1.json
git commit -m "fix(templates): v92 movement types, PetMovementHandle 0xC4, and corrected summon opcode block (task-191)"
```

---

## Task 6: Fix `template_gms_95_1.json`

**Files:**
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_95_1.json`

**Interfaces:**
- Consumes: Task 1's derived v95 table (Step 3) and the v95 `CharacterMoveHandle` opcode (Step 6);
  the guard from Task 3.

Target end state:

| Handler | opCode | Change |
|---|---|---|
| `CharacterMoveHandle` | `0x2C` (**new entry**) | add whole entry with `types` |
| `PetMovementHandle` | `0xC7` (confirmed) | add `types` |
| `SummonMoveHandle` | `0xCF` (confirmed) | add `types` |
| `MonsterMovementHandle` | `0xE3` (confirmed) | add `types` |

- [ ] **Step 1: Confirm `0x2C` is free and find the insertion point**

```bash
python3 -c "
import json
d=json.load(open('services/atlas-configurations/seed-data/templates/template_gms_95_1.json'))
for e in d['socket']['handlers']:
    c=int(e['opCode'],16)
    if 0x28<=c<=0x30: print(e['opCode'], e.get('handler'))
"
```
Expected: `0x2A ChannelChangeHandle`, `0x2D CharacterChairInteractionHandle`,
`0x2E CharacterChairPortableHandle`, `0x2F CharacterMeleeAttackHandle`, `0x30 CharacterRangedAttackHandle`.
`0x2C` is free; the new entry goes between `0x2A` and `0x2D`.

If `0x2C` is **not** free, stop — that contradicts Task 1's derivation and needs escalation.

- [ ] **Step 2: Add the `CharacterMoveHandle` entry at its sorted position**

Insert with Edit, anchoring on the `ChannelChangeHandle` entry's closing `},`:

```json
      {
        "opCode": "0x2C",
        "validator": "LoggedInValidator",
        "handler": "CharacterMoveHandle",
        "services": ["channel"],
        "options": {
          "types": [
```
… Appendix A verbatim …
```json
          ]
        }
      },
```

- [ ] **Step 3: Add `types` to the three existing move handlers**

For `PetMovementHandle` (`0xC7`), `SummonMoveHandle` (`0xCF`), and `MonsterMovementHandle` (`0xE3`):
change `"services": ["channel"]` → `"services": ["channel"],` and append the
`"options": { "types": [ … ] }` block with **Appendix A verbatim**, same indentation as Task 4.
Anchor each `old_string` on its `"handler": "…"` line.

**Watch out:** v95's `PetMovementHandle` (`0xC7`) sits directly above `PetChatHandle` (`0xC8`),
`PetCommandHandle` (`0xC9`), `PetDropPickUpHandle` (`0xCA`), `PetItemUseHandle` (`0xCB`),
`PetItemExcludeHandle` (`0xCC`) — several near-identical five-line entries. Include the
`"handler": "PetMovementHandle",` line in the match so the Edit cannot land on a neighbour.

- [ ] **Step 4: Verify structurally**

```bash
python3 -c "
import json
p='services/atlas-configurations/seed-data/templates/template_gms_95_1.json'
d=json.load(open(p))
h=d['socket']['handlers']
mv={e['handler']:(e.get('options') or {}).get('types') for e in h if e.get('handler') in
    ('CharacterMoveHandle','MonsterMovementHandle','PetMovementHandle','SummonMoveHandle')}
for k in sorted(mv): print(k, len(mv[k]) if mv[k] else None)
ref=mv['CharacterMoveHandle']
print('identical:', all(v==ref for v in mv.values()))
print('validators ok:', all(e.get('validator')=='LoggedInValidator' and e.get('services')==['channel']
      for e in h if e.get('handler') in mv))
codes=[int(e['opCode'],16) for e in h]
print('ascending:', codes==sorted(codes))
print('inventory untouched:', [(e['opCode'],(e.get('options') or {}).get('types')) for e in h if e.get('handler')=='CharacterInventoryMoveHandle'])
print('pet block:', [(e['opCode'],e['handler']) for e in h if e.get('handler','').startswith('Pet')])
"
```
Expected: four move handlers at the derived length (37), `identical: True`, `validators ok: True`,
`ascending: True`, `inventory untouched: [('0x4D', None)]`, and the pet block unchanged
(`0xC7 PetMovementHandle`, `0xC8 PetChatHandle`, `0xC9 PetCommandHandle`, `0xCA PetDropPickUpHandle`,
`0xCB PetItemUseHandle`, `0xCC PetItemExcludeHandle`, plus `PetFoodHandle 0x52`, `PetSpawnHandle 0x6E`).

- [ ] **Step 5: Cross-template check — v92 and v95 tables agree (or the divergence is recorded)**

```bash
python3 -c "
import json
a=json.load(open('services/atlas-configurations/seed-data/templates/template_gms_92_1.json'))
b=json.load(open('services/atlas-configurations/seed-data/templates/template_gms_95_1.json'))
g=lambda d: next((e.get('options') or {}).get('types') for e in d['socket']['handlers'] if e.get('handler')=='CharacterMoveHandle')
print('v92 == v95 types:', g(a)==g(b))
"
```
Expected: `True`, if and only if `movement-types-derivation.md` §4 says the two derived tables are
identical. If the evidence recorded a divergence, this prints `False` and that is correct — do not
"fix" it to match.

- [ ] **Step 6: Run both template guards — both must now be GREEN**

```bash
tools/template-opcode-order-guard.sh; echo "order exit=$?"
tools/template-movement-types-guard.sh; echo "movement exit=$?"
```
Expected: `order exit=0`, `movement exit=0`, and the movement guard prints

```
OK: 43 move handlers across 11 templates carry a valid movement types table.
```

43 = 41 today (counted during planning: 3 in `gms_12_1` — it has no pet handler — 4 in each of
`gms_48/61/72/79/83/84/87/jms_185`, 3 in `gms_92_1`, 3 in `gms_95_1`) plus the two entries added in
Tasks 5 and 6. A different total means a handler was added or missed.

This is the RED→GREEN transition for the guard written in Task 3.

- [ ] **Step 7: Confirm no other template changed**

```bash
git diff --stat main -- services/atlas-configurations/seed-data/templates/
```
Expected: exactly three files — `template_gms_48_1.json`, `template_gms_92_1.json`,
`template_gms_95_1.json`.

- [ ] **Step 8: Commit**

```bash
git add services/atlas-configurations/seed-data/templates/template_gms_95_1.json
git commit -m "fix(templates): v95 movement types and CharacterMoveHandle 0x2C (task-191)"
```

---

## Task 7: Wire the guard into CI and the docs

Now that the guard is green, register it so the defect class cannot recur.

**Files:**
- Modify: `.github/workflows/pr-validation.yml`, `CLAUDE.md`, `docs/packets/TEMPLATE_CONVENTIONS.md`

**Interfaces:**
- Consumes: `tools/template-movement-types-guard.sh` (Task 3), green as of Task 6.

- [ ] **Step 1: Add the CI job**

In `.github/workflows/pr-validation.yml`, insert a new job immediately after the
`template-opcode-order-guard` job (which ends at the `run: ./tools/template-opcode-order-guard.sh`
line, ~:322), matching its exact shape:

```yaml
  # ============================================
  # Enforces that every MOVE handler in every
  # tenant socket-config template carries a
  # well-formed, template-consistent movement
  # `types` table. A missing or typo'd entry
  # silently degrades that element to a 3-byte
  # decode and desyncs the packet. See
  # docs/packets/TEMPLATE_CONVENTIONS.md.
  # Pure shell + python3, no Go setup.
  # ============================================
  template-movement-types-guard:
    name: Template Movement Types Guard
    runs-on: ubuntu-latest

    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: template movement types guard
        run: ./tools/template-movement-types-guard.sh
```

- [ ] **Step 2: Add it to the summary job's `needs:` list**

At `.github/workflows/pr-validation.yml:713`, add `template-movement-types-guard` immediately after
`template-opcode-order-guard`:

```yaml
    needs: [detect-changes, test-go-libraries, test-go-services, test-ui, build-docker, update-pr-overlay, redis-key-guard, outbox-guard, goroutine-guard, gen-lb-ports, service-registration-guard, template-opcode-order-guard, template-movement-types-guard, skill-job-id-guard, atlas-constants-drift-guard, lint-go, lint-ui]
```

- [ ] **Step 3: Add it to the result rollup**

In the same job's `Check results` step, after the `TMPL_ORDER_GUARD_RESULT` line (~:734):

```bash
          TMPL_MOVE_GUARD_RESULT="${{ needs.template-movement-types-guard.result }}"
```

and after the `| Template Opcode Order Guard |` summary row:

```bash
          echo "| Template Movement Types Guard | $TMPL_MOVE_GUARD_RESULT |" >> $GITHUB_STEP_SUMMARY
```

- [ ] **Step 3b: Add it to the pass/fail assertion**

At the end of the same step (~:761) there is one long `if` chaining every `*_RESULT` variable. Add the
new variable immediately after `$TMPL_ORDER_GUARD_RESULT`:

```bash
          if [ "$LIBS_RESULT" == "failure" ] || [ "$SERVICES_RESULT" == "failure" ] || [ "$UI_RESULT" == "failure" ] || [ "$DOCKER_RESULT" == "failure" ] || [ "$OVERLAY_RESULT" == "failure" ] || [ "$GUARD_RESULT" == "failure" ] || [ "$OUTBOX_GUARD_RESULT" == "failure" ] || [ "$GOROUTINE_GUARD_RESULT" == "failure" ] || [ "$LBPORTS_RESULT" == "failure" ] || [ "$SVC_REG_GUARD_RESULT" == "failure" ] || [ "$TMPL_ORDER_GUARD_RESULT" == "failure" ] || [ "$TMPL_MOVE_GUARD_RESULT" == "failure" ] || [ "$SKILL_JOB_ID_GUARD_RESULT" == "failure" ] || [ "$CONSTANTS_DRIFT_GUARD_RESULT" == "failure" ] || [ "$LINT_GO_RESULT" == "failure" ] || [ "$LINT_UI_RESULT" == "failure" ]; then
            echo ""
            echo "One or more jobs failed!"
            exit 1
          fi
```

**Missing this step is the silent failure mode**: without it the job would appear in the summary table
but never fail the check run.

- [ ] **Step 4: Add CLAUDE.md Build & Verification item 11**

After item 10 (`tools/skill-job-id-guard.sh`):

```markdown
11. **`tools/template-movement-types-guard.sh` clean from the repo root** whenever
    a tenant socket-config template under
    `services/atlas-configurations/seed-data/templates/` changed. Every move
    handler (`CharacterMoveHandle`, `MonsterMovementHandle`, `PetMovementHandle`,
    `SummonMoveHandle`) must carry a non-empty `options.types`; all such arrays
    within one template must be byte-identical; every `Type` must be one of the
    seven the decoder recognizes; at most one entry may be named `FALL_DOWN`.
    A missing table makes every movement fragment decode as a 3-byte stub (loud:
    "Code [N] not configured for use in movement"); a typo'd `Type` does the same
    for one index, silently. See
    [`docs/packets/TEMPLATE_CONVENTIONS.md`](docs/packets/TEMPLATE_CONVENTIONS.md).
```

**Also update the worktree's `CLAUDE.md`, not the main checkout's.**

- [ ] **Step 5: Document the rule in `docs/packets/TEMPLATE_CONVENTIONS.md`**

Add a new section after "Rule: ascending opcode order (enforced)" and before the existing "Guard"
section — then extend that "Guard" section to name both scripts:

```markdown
## Rule: move handlers carry a `types` table (enforced)

The four **move** handlers — `CharacterMoveHandle`, `MonsterMovementHandle`,
`PetMovementHandle`, `SummonMoveHandle` — MUST each carry a non-empty
`options.types` array, and all such arrays within one template MUST be
byte-identical.

`libs/atlas-packet/model/movement.go` decodes a movement fragment by reading a
one-byte element type and looking it up **as an array index** in that handler's
`options.types`. The entry's `Type` selects the concrete element decoder
(`NORMAL`, `JUMP`, `TELEPORT`, `START_FALL_DOWN`, `FLYING_BLOCK`, `STAT_CHANGE`,
`DEFAULT` — those seven and no others). `Name` is cosmetic except for the
reserved `FALL_DOWN`, which triggers an extra `FhFallStart` int16 in
`NormalElement`; at most one entry per array may use it.

The array is **positional — the index IS the wire value**. Entries are never
reordered, and gaps are filled with `{"Name": "UNKNOWN", "Type": "<derived>"}`
rather than omitted. Its length and contents are version-specific and MUST be
derived from that version's client (`CMovePath::Encode`/`::Decode`), never
copied from a neighbouring template — the table is renumbered between versions.

Failure modes, both of which have shipped:

- **Table absent** → the lookup returns `("NOT_FOUND", "DEFAULT")`, no decoder
  branch matches, and every fragment takes the bare 3-byte `Element` decoder
  against a fragment 9–15 bytes wide. Loud: one error log line per fragment.
- **One `Type` typo'd** → the same 3-byte degradation for that one index, with
  **no** log line at all.

`CharacterInventoryMoveHandle` is not a move handler despite the name — it is
inventory item movement and correctly carries no `types`.
```

- [ ] **Step 6: Validate the workflow YAML parses**

```bash
python3 -c "
import sys
try:
    import yaml
except ImportError:
    sys.exit('PyYAML not installed — validate with: docker run --rm -v \$PWD:/w -w /w cytopia/yamllint .github/workflows/pr-validation.yml')
d=yaml.safe_load(open('.github/workflows/pr-validation.yml'))
jobs=d['jobs']
print('job present:', 'template-movement-types-guard' in jobs)
print('in needs:', 'template-movement-types-guard' in jobs['pr-validation-complete']['needs'])
"
```
Expected: `job present: True`, `in needs: True`. If PyYAML is unavailable, use the printed fallback
or `gh workflow view` — do not skip the check.

- [ ] **Step 7: Run the guard once more from a clean shell**

```bash
tools/template-movement-types-guard.sh; echo "exit=$?"
```
Expected: `exit=0`.

- [ ] **Step 8: Commit**

```bash
git add .github/workflows/pr-validation.yml CLAUDE.md docs/packets/TEMPLATE_CONVENTIONS.md
git commit -m "ci(templates): enforce movement types guard in PR validation (task-191)"
```

---

## Task 8: Correct the v95 registry `MOVE_PLAYER` fname

The PRD anticipated a possible **opcode** correction; Task 1's derivation says the opcode 44
(`0x2C`) is right and the `fname` is the misleading part — the true sender is already recorded in
that row's `fname_alts`.

**Files:**
- Modify: `docs/packets/registry/gms_v95.yaml:2290-2294`

**Interfaces:**
- Consumes: Task 1 Step 6's v95 user-sender derivation (`CVecCtrlUser::EndUpdateActive` `0x9a0d20`,
  `COutPacket(44)` @`0x9a0ee3`).

**Gate:** apply only if Task 1 confirmed opcode 44. If the derivation found a different opcode, the
change is an opcode correction instead and must be recorded as such.

- [ ] **Step 1: Read the current row**

```bash
sed -n '2290,2295p' docs/packets/registry/gms_v95.yaml
```
Expected:
```yaml
- op: MOVE_PLAYER
  direction: serverbound
  opcode: 44
  fname: CUserLocal::OnKey
  fname_alts:
    - CVecCtrlUser::EndUpdateActive
  provenance: csv-import
```

- [ ] **Step 2: Swap fname and fname_alts**

```yaml
- op: MOVE_PLAYER
  direction: serverbound
  opcode: 44
  fname: CVecCtrlUser::EndUpdateActive
  fname_alts:
    - CUserLocal::OnKey
  provenance: csv-import
```

Opcode unchanged at 44. `CVecCtrlUser::EndUpdateActive` @`0x9a0d20` is the function that builds
`COutPacket(44)` @`0x9a0ee3` and calls `CMovePath::Flush`; `CUserLocal::OnKey` is an input handler,
not the sender.

- [ ] **Step 3: Verify the YAML still parses and no other row moved**

```bash
python3 -c "
import yaml
d=yaml.safe_load(open('docs/packets/registry/gms_v95.yaml'))
rows=[r for r in (d if isinstance(d,list) else d.get('packets',d)) if isinstance(r,dict) and r.get('op')=='MOVE_PLAYER']
print(rows)
"
git diff --stat docs/packets/registry/gms_v95.yaml
```
Expected: the `MOVE_PLAYER` serverbound row shows `fname: CVecCtrlUser::EndUpdateActive`, and the
diffstat shows exactly 2 insertions / 2 deletions in that one file. If the top-level YAML shape
differs from the guess above, adapt the one-liner — the point is that the file parses and the row
reads correctly.

- [ ] **Step 4: Commit**

```bash
git add docs/packets/registry/gms_v95.yaml
git commit -m "docs(registry): v95 MOVE_PLAYER fname is CVecCtrlUser::EndUpdateActive (task-191)"
```

---

## Task 9: Full verification sweep

Every claim in the acceptance criteria gets a command and quoted output. No "should pass."

**Files:** none modified (unless a check fails).

- [ ] **Step 1: Go tests and vet in the one changed module**

```bash
cd libs/atlas-packet && go test -race ./... && go vet ./...
```
Expected: all packages `ok` or `no test files`; `go vet` silent. `libs/atlas-packet` is the only Go
module touched, so no other module needs this and no `docker buildx bake` is required (no `go.mod`
changed — confirm with `git diff --name-only main | grep go.mod || echo "no go.mod touched"`).

- [ ] **Step 2: Both template guards**

```bash
cd "$(git rev-parse --show-toplevel)"
tools/template-opcode-order-guard.sh; echo "order exit=$?"
tools/template-movement-types-guard.sh; echo "movement exit=$?"
```
Expected: both `exit=0`.

- [ ] **Step 3: Lint and format**

```bash
tools/lint.sh --check; echo "exit=$?"
```
Expected: `exit=0`. If it reports formatting drift, run `tools/lint.sh` (no flags) to fix in place,
re-run `--check`, and amend the relevant commit. Watch for the known `goimports` trap with the
unaliased `atlas-tenant` import: the package is `tenant` but the directory is `atlas-tenant`, and
goimports can duplicate the import. `movement.go` already imports it as
`tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"` — leave that alias in place.

- [ ] **Step 4: Re-run the FR-5 invariants across all 11 templates, output shown**

```bash
python3 -c "
import glob, json
MV={'CharacterMoveHandle','MonsterMovementHandle','PetMovementHandle','SummonMoveHandle'}
VALID={'NORMAL','JUMP','TELEPORT','START_FALL_DOWN','FLYING_BLOCK','STAT_CHANGE','DEFAULT'}
for p in sorted(glob.glob('services/atlas-configurations/seed-data/templates/template_*.json')):
    d=json.load(open(p)); arrs={}
    for e in d['socket']['handlers']:
        if e.get('handler') in MV: arrs[e['handler']]=(e.get('options') or {}).get('types')
    lens={k:(len(v) if v else None) for k,v in arrs.items()}
    same=len({json.dumps(v,sort_keys=True) for v in arrs.values()})==1
    bad=[t for v in arrs.values() if v for t in {x.get('Type') for x in v} if t not in VALID]
    fd=max((sum(1 for x in v if x.get('Name')=='FALL_DOWN') for v in arrs.values() if v), default=0)
    print('%-28s %s identical=%s bad_types=%s max_fall_down=%d' % (p.split('/')[-1], lens, same, bad or 'none', fd))
"
```
Expected, for every one of the 11 templates: no `None` lengths, `identical=True`,
`bad_types=none`, `max_fall_down` ≤ 1.

- [ ] **Step 5: Scope containment against `main`**

```bash
git diff --stat main
```
Expected — exactly this set and nothing else:

```
.github/workflows/pr-validation.yml
CLAUDE.md
docs/packets/TEMPLATE_CONVENTIONS.md
docs/packets/registry/gms_v95.yaml
docs/tasks/task-191-v92-v95-movement-types/…            (prd/design/current-state/plan/context/derivation/reconcile)
libs/atlas-packet/model/movement.go
libs/atlas-packet/model/movement_test.go
services/atlas-configurations/seed-data/templates/template_gms_48_1.json
services/atlas-configurations/seed-data/templates/template_gms_92_1.json
services/atlas-configurations/seed-data/templates/template_gms_95_1.json
tools/template-movement-types-guard.sh
```

```bash
git diff --name-only main | grep -E '^services/atlas-channel/' && echo "VIOLATION: atlas-channel modified" || echo "atlas-channel untouched (ok)"
```
Expected: `atlas-channel untouched (ok)`.

- [ ] **Step 6: Confirm the worktree and branch are still correct**

```bash
git rev-parse --show-toplevel   # must end with /.worktrees/task-191-v92-v95-movement-types
git branch --show-current       # must be task-191-v92-v95-movement-types
git status --short              # must be clean
```

- [ ] **Step 7: Commit (only if Step 3 required formatting fixes)**

```bash
git add -A && git commit -m "chore(task-191): lint fixes"
```
If nothing changed, skip — do not create an empty commit.

---

## Task 10: Reconcile the live v92 and v95 tenants (FR-6)

Seed templates apply **only at tenant provisioning**. Existing v92/v95 tenants keep their stored
socket configuration and will not pick the fix up without a data operation.

**Files:**
- Create: `docs/tasks/task-191-v92-v95-movement-types/reconcile.md`

**Interfaces:**
- Consumes: the final state of `template_gms_92_1.json` and `template_gms_95_1.json` (Tasks 5, 6).

- [ ] **Step 1: Enumerate the configuration tenants**

atlas-configurations is reachable externally through the ingress; no tenant headers are needed for
this bootstrap resource:

```bash
curl -s --resolve dev.atlas.home:80:192.168.23.230 \
  http://dev.atlas.home/api/configurations/tenants | python3 -m json.tool | head -60
```

If that host/IP no longer resolves, find the current ingress and LB IP first:
```bash
kubectl -n atlas-main get ingress,svc -o wide
```

Select the tenants whose region/major version is GMS 92 or GMS 95. Project memory records these from
2026-07-30 (**verify, do not trust**): v92 `db1dbfb3…` (parked), v95 `c794c706…`, in namespace
`atlas-main`. Record the full ids you actually observe.

- [ ] **Step 2: Snapshot each tenant's current configuration**

```bash
mkdir -p /tmp/claude-1000/task-191
for T in <v92-tenant-id> <v95-tenant-id>; do
  curl -s --resolve dev.atlas.home:80:192.168.23.230 \
    "http://dev.atlas.home/api/configurations/tenants/$T" > /tmp/claude-1000/task-191/$T.before.json
done
```

Record, per tenant, the **before** shape of the four move handlers:

```bash
python3 -c "
import json,sys
d=json.load(open(sys.argv[1]))
for e in d['data']['attributes']['socket']['handlers']:
    if e.get('handler','').endswith(('MoveHandle','MovementHandle')) or 'Summon' in e.get('handler',''):
        t=(e.get('options') or {}).get('types')
        print(e['opCode'], e['handler'], 'validator=%s' % e.get('validator'), 'types=%s' % (len(t) if t else None))
" /tmp/claude-1000/task-191/<tenant-id>.before.json
```

- [ ] **Step 3: Build the reconciled configuration — surgical, not a template push**

**A wholesale `socket` swap is forbidden.** PATCH is a **full replace** of the tenant configuration
JSON, so the body must carry complete attributes — but the *content* must be the live config with
only the move-handler entries changed. Swapping in the template's whole `socket` block would also
apply unrelated opcode relocations and rewrite `operations` mode tables (client wire values needing
their own IDA provenance) and silently revert tenant-specific customization.

Per tenant, apply exactly these edits to `attributes.socket.handlers` and leave every other handler,
writer, and the `worlds`/`characters`/`npcs`/`cashShop`/`usesPin` blocks byte-identical:

*v92 tenant:*
1. Add `options.types` (the derived array) to `CharacterMoveHandle`, `MonsterMovementHandle`,
   `SummonMoveHandle`.
2. Add a new `PetMovementHandle` entry at the derived opCode with
   `"validator": "LoggedInValidator"`, `"services": ["channel"]`, and the derived `types`.
3. Correct the three summon opcodes to the derived values.
4. Re-sort `handlers` by ascending `opCode`.

*v95 tenant:*
1. Add `options.types` to `PetMovementHandle`, `MonsterMovementHandle`, `SummonMoveHandle`.
2. Add a new `CharacterMoveHandle` entry at the derived opCode with
   `"validator": "LoggedInValidator"`, `"services": ["channel"]`, and the derived `types`.
3. Re-sort `handlers` by ascending `opCode`.

Before PATCHing, diff the built body against the snapshot to prove nothing else moved:

```bash
python3 -c "
import json,sys
a=json.load(open(sys.argv[1]))['data']['attributes']
b=json.load(open(sys.argv[2]))['data']['attributes']
ka,kb=set(a),set(b)
print('top-level keys added/removed:', kb-ka, ka-kb)
for k in sorted(ka&kb):
    if k!='socket' and a[k]!=b[k]: print('CHANGED OUTSIDE SOCKET:', k)
sa={e['handler']:e for e in a['socket']['handlers']}
sb={e['handler']:e for e in b['socket']['handlers']}
print('handlers added:', sorted(set(sb)-set(sa)))
print('handlers removed:', sorted(set(sa)-set(sb)))
for h in sorted(set(sa)&set(sb)):
    if sa[h]!=sb[h]: print('handler changed:', h, sa[h].get('opCode'),'->',sb[h].get('opCode'))
print('writers unchanged:', a['socket'].get('writers')==b['socket'].get('writers'))
" /tmp/claude-1000/task-191/<tenant>.before.json /tmp/claude-1000/task-191/<tenant>.after.json
```

Expected: no keys added/removed, nothing changed outside `socket`, `writers unchanged: True`, exactly
one handler added, zero removed, and only the intended handlers changed.

- [ ] **Step 4: PATCH**

```bash
curl -s -X PATCH --resolve dev.atlas.home:80:192.168.23.230 \
  -H 'Content-Type: application/vnd.api+json' \
  --data @/tmp/claude-1000/task-191/<tenant>.after.json \
  "http://dev.atlas.home/api/configurations/tenants/<tenant-id>" -w '\nHTTP %{http_code}\n'
```

The body is the JSON:API envelope `{"data":{"type":"tenants","id":"…","attributes":{…}}}`.
Expected: HTTP 200.

- [ ] **Step 5: Read back and verify — the PATCH response is NOT evidence**

A handler entry missing its `validator` is accepted at the transport layer and then silently dropped
at load time, so verification must be a fresh `GET`:

```bash
curl -s --resolve dev.atlas.home:80:192.168.23.230 \
  "http://dev.atlas.home/api/configurations/tenants/<tenant-id>" > /tmp/claude-1000/task-191/<tenant>.readback.json

python3 -c "
import json,sys
d=json.load(open(sys.argv[1]))
hs=d['data']['attributes']['socket']['handlers']
mv={'CharacterMoveHandle','MonsterMovementHandle','PetMovementHandle','SummonMoveHandle'}
found={}
for e in hs:
    if e.get('handler') in mv:
        t=(e.get('options') or {}).get('types')
        found[e['handler']]=(e['opCode'], e.get('validator'), len(t) if t else None)
for k in sorted(mv): print(k, found.get(k, 'ABSENT'))
codes=[int(e['opCode'],16) for e in hs]
print('ascending:', codes==sorted(codes))
print('all four present with non-empty types:', len(found)==4 and all(v[2] for v in found.values()))
print('all validators set:', all(v[1] for v in found.values()))
" /tmp/claude-1000/task-191/<tenant>.readback.json
```

Expected per tenant: all four handlers present at the derived opcodes, each with
`validator=LoggedInValidator` and the derived `types` length; `ascending: True`;
`all four present with non-empty types: True`; `all validators set: True`.

**Quote the actual output in `reconcile.md`.** A `False` on any line means the PATCH did not take —
do not proceed.

- [ ] **Step 6: Restart atlas-channel**

Handler/writer maps are built at listener-creation time and the configuration projection's
`ListenerConfig` diff **excludes** handlers/writers, so a handlers-only change does **not** hot-reload.

```bash
kubectl -n atlas-main rollout restart deployment/atlas-channel
kubectl -n atlas-main rollout status deployment/atlas-channel
```

Confirm the rebuild from the startup logs — look for `Configuring opcode [0x…] … handler [...]` lines
covering the new/moved move opcodes, and `listener.added`:

```bash
kubectl -n atlas-main logs deployment/atlas-channel --tail=400 | grep -iE 'Configuring opcode|listener.added' | head -40
```

- [ ] **Step 7: Confirm the negative signal — the error line is gone**

The primary post-reconcile signal is negative. `movementPathAttrFromOptions` logs
`"Code [%d] not configured for use in movement…"` at error level per fragment (`movement.go:289`).
After the reconcile those lines must be absent for v92/v95 channels:

```bash
kubectl -n atlas-main logs deployment/atlas-channel --since=10m | grep -c "not configured for use in movement" || echo 0
```
Expected: `0`. Continued presence means the reconcile did not take.

- [ ] **Step 8: Write `reconcile.md`**

Required content (FR-6.3 — the operation must be repeatable for any environment not covered here):

1. Environment(s) and namespace, the ingress host/IP used, and the date.
2. The tenant ids patched, with their region/version.
3. The exact `curl` commands (GET / diff / PATCH / read-back), with tenant ids filled in.
4. The before/after handler shape per tenant.
5. The **quoted** read-back output from Step 5.
6. The atlas-channel restart and the quoted `Configuring opcode` / `listener.added` evidence.
7. The Step 7 log-grep result.
8. Any environment deliberately **not** reconciled, and why.
9. The explicit warning that PATCH is a full replace and that a wholesale `socket` swap is forbidden.

- [ ] **Step 9: Commit**

```bash
git add docs/tasks/task-191-v92-v95-movement-types/reconcile.md
git commit -m "docs(task-191): record live v92/v95 tenant socket-config reconcile"
```

---

## Task 11: Code review before PR

CLAUDE.md: *"Always run the code-review step before opening a PR. Do not skip even when the task plan
looks complete."*

- [ ] **Step 1: Invoke the review skill**

Use `superpowers:requesting-code-review`. Go files changed (`libs/atlas-packet/model/movement.go`),
so it dispatches `backend-guidelines-reviewer` and `plan-adherence-reviewer`; no atlas-ui TypeScript
changed, so no frontend reviewer.

Per CLAUDE.md's model preference, pin the reviewer subagents to Sonnet/Haiku — do not use an
expensive model for review workflows.

Every reviewer prompt must `cd` into `.worktrees/task-191-v92-v95-movement-types` and verify
`git branch --show-current` before doing anything, and write artifacts only inside that worktree.

- [ ] **Step 2: Verify the tree is clean after the subagent runs**

```bash
git status --short
git rev-parse --show-toplevel
```
Expected: clean, and the toplevel ends with `/.worktrees/task-191-v92-v95-movement-types`. If a
reviewer wrote into the main checkout, move the file and clean up.

- [ ] **Step 3: Address findings**

Use `superpowers:receiving-code-review` — verify each finding technically before implementing it;
do not implement blindly, and do not agree performatively with a finding that is wrong.

- [ ] **Step 4: Re-run the verification sweep**

Repeat Task 9 Steps 1–5 after any review fix. Commit the fixes.

- [ ] **Step 5: Commit the audit artifacts**

```bash
git add docs/tasks/task-191-v92-v95-movement-types/
git commit -m "docs(task-191): code review findings"
```

---

## Appendix A — canonical v92/v95 `types` block

**This is the design-phase derivation (design §3), reproduced here so Tasks 5 and 6 have literal text
to insert. It is NOT an oracle.** Task 1 re-derives the table independently against both IDBs; where
Task 1's reading differs, Task 1 wins and this block is edited before use, with the divergence
recorded in `movement-types-derivation.md`.

Length 37 (indices 0–36). Group membership per design §3:

| Group | Indices | Count |
|---|---|---|
| `NORMAL` | 0, 5, 12, 14, 35, 36 | 6 |
| `JUMP` | 1, 2, 13, 16, 18, 31, 32, 33, 34 | 9 |
| `TELEPORT` | 3, 4, 6, 7, 8, 10 | 6 |
| `STAT_CHANGE` | 9 | 1 |
| `START_FALL_DOWN` | 11 | 1 |
| `FLYING_BLOCK` | 17 | 1 |
| `DEFAULT` | 15, 19, 20–30 | 13 |
| **total** | | **37** |

Only five indices carry a non-`UNKNOWN` `Name`, each justified by client behaviour (design §2.4):
0 `NORMAL`, 9 `STAT_CHANGE`, 11 `START_FALL_DOWN`, 12 `FALL_DOWN` (**load-bearing**),
17 `FLYING_BLOCK`.

Insert verbatim between `"types": [` and `]`, at 12-space indentation for each entry object:

```json
            {
              "Name": "NORMAL",
              "Type": "NORMAL"
            },
            {
              "Name": "UNKNOWN",
              "Type": "JUMP"
            },
            {
              "Name": "UNKNOWN",
              "Type": "JUMP"
            },
            {
              "Name": "UNKNOWN",
              "Type": "TELEPORT"
            },
            {
              "Name": "UNKNOWN",
              "Type": "TELEPORT"
            },
            {
              "Name": "UNKNOWN",
              "Type": "NORMAL"
            },
            {
              "Name": "UNKNOWN",
              "Type": "TELEPORT"
            },
            {
              "Name": "UNKNOWN",
              "Type": "TELEPORT"
            },
            {
              "Name": "UNKNOWN",
              "Type": "TELEPORT"
            },
            {
              "Name": "STAT_CHANGE",
              "Type": "STAT_CHANGE"
            },
            {
              "Name": "UNKNOWN",
              "Type": "TELEPORT"
            },
            {
              "Name": "START_FALL_DOWN",
              "Type": "START_FALL_DOWN"
            },
            {
              "Name": "FALL_DOWN",
              "Type": "NORMAL"
            },
            {
              "Name": "UNKNOWN",
              "Type": "JUMP"
            },
            {
              "Name": "UNKNOWN",
              "Type": "NORMAL"
            },
            {
              "Name": "UNKNOWN",
              "Type": "DEFAULT"
            },
            {
              "Name": "UNKNOWN",
              "Type": "JUMP"
            },
            {
              "Name": "FLYING_BLOCK",
              "Type": "FLYING_BLOCK"
            },
            {
              "Name": "UNKNOWN",
              "Type": "JUMP"
            },
            {
              "Name": "UNKNOWN",
              "Type": "DEFAULT"
            },
            {
              "Name": "UNKNOWN",
              "Type": "DEFAULT"
            },
            {
              "Name": "UNKNOWN",
              "Type": "DEFAULT"
            },
            {
              "Name": "UNKNOWN",
              "Type": "DEFAULT"
            },
            {
              "Name": "UNKNOWN",
              "Type": "DEFAULT"
            },
            {
              "Name": "UNKNOWN",
              "Type": "DEFAULT"
            },
            {
              "Name": "UNKNOWN",
              "Type": "DEFAULT"
            },
            {
              "Name": "UNKNOWN",
              "Type": "DEFAULT"
            },
            {
              "Name": "UNKNOWN",
              "Type": "DEFAULT"
            },
            {
              "Name": "UNKNOWN",
              "Type": "DEFAULT"
            },
            {
              "Name": "UNKNOWN",
              "Type": "DEFAULT"
            },
            {
              "Name": "UNKNOWN",
              "Type": "DEFAULT"
            },
            {
              "Name": "UNKNOWN",
              "Type": "JUMP"
            },
            {
              "Name": "UNKNOWN",
              "Type": "JUMP"
            },
            {
              "Name": "UNKNOWN",
              "Type": "JUMP"
            },
            {
              "Name": "UNKNOWN",
              "Type": "JUMP"
            },
            {
              "Name": "UNKNOWN",
              "Type": "NORMAL"
            },
            {
              "Name": "UNKNOWN",
              "Type": "NORMAL"
            }
```

Sanity check after inserting (run per template, per handler):

```bash
python3 -c "
import json,sys,collections
d=json.load(open(sys.argv[1]))
for e in d['socket']['handlers']:
    t=(e.get('options') or {}).get('types')
    if not t: continue
    if e['handler'] not in ('CharacterMoveHandle','MonsterMovementHandle','PetMovementHandle','SummonMoveHandle'): continue
    c=collections.Counter(x['Type'] for x in t)
    print(e['handler'], 'len=%d'%len(t), dict(c),
          'FALL_DOWN@%s' % [i for i,x in enumerate(t) if x['Name']=='FALL_DOWN'])
" services/atlas-configurations/seed-data/templates/template_gms_92_1.json
```
Expected per handler: `len=37`,
`{'NORMAL': 6, 'JUMP': 9, 'TELEPORT': 6, 'STAT_CHANGE': 1, 'START_FALL_DOWN': 1, 'DEFAULT': 13, 'FLYING_BLOCK': 1}`,
`FALL_DOWN@[12]`.
