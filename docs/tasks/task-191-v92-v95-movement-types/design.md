# v92/v95 Movement `types` Configuration — Design

Version: v1
Status: Proposed
Created: 2026-08-04
PRD: [`prd.md`](prd.md) · Current state: [`current-state.md`](current-state.md)

---

## 1. Executive summary

The PRD frames this as a pure configuration task: derive the per-index movement `types` arrays for
GMS v92 and v95, drop them into the two seed templates, add the two missing handler entries, close
the v48 Summon leftover, and reconcile live tenants. That framing is correct as far as it goes, and
the derivation is done — §3 carries the full 37-entry array, derived from
`CMovePath::Encode`/`CMovePath::Decode` in both clients.

Design-phase reverse engineering surfaced **two defects the PRD did not anticipate**, both of which
sit upstream of the `types` fix. Neither is optional: shipping the `types` arrays alone leaves v92
and v95 movement just as broken, only failing silently instead of loudly.

| # | Finding | PRD status | Consequence if omitted |
|---|---|---|---|
| **A** | The v92/v95 movement **header** carries four `int16` (`x`, `y`, `vx`, `vy`) before the element count, not two. `libs/atlas-packet/model/movement.go` reads two. | PRD §2 explicitly non-goals any `movement.go` change | Every v92/v95 movement packet desyncs by 4 bytes **before** the first element is read. The `types` arrays never get a chance to matter. |
| **B** | `template_gms_92_1.json`'s `SummonMoveHandle` is registered at `0xC8`. The v92 client sends summon movement on **`0xCC`**; `0xC8` is v87's *monster* opcode — a stale copy. | PRD assumes existing v92 opcodes are correct and only asks for the two *missing* entries | v92 summon movement stays unrouted, and `0xC8` decodes an unrelated packet as a summon move. |

Both are proposed as scope amendments (§4, §5). The rest of the design follows the PRD as written.

Two of the PRD's four open questions are also resolved here from the clients: the v95
`CharacterMoveHandle` opCode is **`0x2C` — the registry value was right** (§5.1), and the v92
`PetMovementHandle` opCode is **`0xC4`** (§5.2). Array length is **37** for both versions, not 25.

---

## 2. Derivation method

### 2.1 Source of truth

The movement element table is not a data file in the client; it is the `switch (nAttr)` in the two
halves of the `CMovePath` codec. Both halves are present and named in every IDB in the set:

| Version | IDB (match by filename; session ids rotate) | `CMovePath::Encode` | `CMovePath::Decode` |
|---|---|---|---|
| GMS v83 | `MapleStory_dump.exe.i64` | `0x68a563` | `0x68a33c` |
| GMS v87 | `GMSv87_4GB.exe.i64` | `0x6c70fe` | `0x6c6e86` |
| **GMS v92** | `GMS_v92_1_DEVM.exe.i64` | **`0x65a260`** | **`0x65ad60`** |
| **GMS v95** | `GMS_v95.0_U_DEVM.exe.i64` | **`0x666e20`** | **`0x667920`** |
| JMS v185 | `MapleStory_dump_SCY.exe.i64` | `0x70b6c4` | — |

`Encode` is the serverbound authority (what Atlas decodes); `Decode` is the clientbound authority
(what Atlas must encode). Both were read for v92 and v95, and they agree case-for-case — which is
the expected invariant, and a free consistency check on the derivation.

Read the switch, and each `case` group's `Encode2`/`Encode1` sequence maps 1:1 onto one of the seven
`Type` values `libs/atlas-packet/model/movement.go` branches on. No inference is required: the field
set *is* the type.

### 2.2 Why this method is trustworthy — the v83 control

Before deriving anything new, the method was run against a version whose template is already known
good. v83's `CMovePath::Encode` @`0x68a563` yields:

- NORMAL (`x,y,vx,vy,fh`, plus `fhFallStart` when `nAttr == 15`): `0, 5, 15, 17`
- JUMP (`vx,vy`): `1, 2, 6, 12, 13, 16, 18, 19, 20, 22`
- TELEPORT (`x,y,fh`): `3, 4, 7, 8, 9, 11`
- STAT_CHANGE (`bStat`, and **no** trailing `bMoveAction`/`tElapse`): `10`
- START_FALL_DOWN (`vx,vy,fhFallStart`): `14`
- DEFAULT (no extra fields): `21`, and anything unlisted

That reproduces `template_gms_83_1.json`'s 23-entry array **index-for-index, including the
`FALL_DOWN` position at 15 and the lone `DEFAULT` at 21**. The method is validated; the same
procedure applied to v92/v95 is the derivation, not a guess.

### 2.3 Why neighbouring templates could not have been copied

The v83 control also proves FR-1.5's warning empirically. The v92/v95 table is **not** a
right-extension of v87's — it is renumbered:

| Semantic | v83/v84/v87 index | v92/v95 index |
|---|---|---|
| `STAT_CHANGE` | 10 | **9** |
| `START_FALL_DOWN` | 14 | **11** |
| `FALL_DOWN` | 15 | **12** |
| `FLYING_BLOCK` | 23 (v84+) | **17** |

Had the v87 array been copied into v92, index 15 (`FALL_DOWN`/NORMAL there) would have decoded as a
15-byte NORMAL element what the v92 client sends as a 3-byte default element — a worse failure than
the current one, because it is silent. This is the precise shortcut the tracking issue rules out,
and the numbers show why.

### 2.4 Naming policy

`Name` is load-bearing in exactly one place: `NormalElement` reads an extra `FhFallStart` when
`Name == "FALL_DOWN"` (`movement.go:126-128`, mirrored at `:218-220`). Every other name is cosmetic.

Because the table was renumbered, the v83-era semantic names (`IMPACT`, `ASSAULTER`,
`ASSASSINATION`, `RUSH`, `SIT_DOWN`, `START_WINGS`, `ARAN_ADJUST`, `MOB_TOSS`, `DASH_SLIDE`) cannot
be carried across by index, and the client exposes no name strings for them — the attribute is a
bare `int` throughout `CVecCtrl::SetMovePathAttribute` and the `CMovePath::ELEM::nAttr` field.
Inventing a semantic name per index would be exactly the fabrication CLAUDE.md's grounding rule
forbids.

**Decision:** name only what the client's own behaviour establishes, and use `UNKNOWN` everywhere
else:

| Index | `Name` | Justification |
|---|---|---|
| 0 | `NORMAL` | the base walk/normal element, first NORMAL-group member |
| 9 | `STAT_CHANGE` | sole `bStat`-only arm |
| 11 | `START_FALL_DOWN` | sole `vx,vy,fhFallStart` arm |
| 12 | `FALL_DOWN` | **load-bearing** — the `nAttr == 12` inner branch that reads `fhFallStart` inside the NORMAL group |
| 17 | `FLYING_BLOCK` | sole `x,y,vx,vy` arm |
| all others | `UNKNOWN` | no client-side evidence for a semantic name |

This matches how the existing templates already treat unresolved indices (`UNKNOWN`) and keeps every
committed value traceable to a decompile line.

*Rejected alternative:* recover names from a client `MPA_*` enum. The v95 IDB is PDB-backed, but a
PDB carries no enumerator names for a value passed as a plain `int`; there is nothing to recover.
Attempting it would burn execution time for a cosmetic field.

---

## 3. Derived movement `types` — GMS v92 and GMS v95

**The v92 and v95 tables are identical.** Both `CMovePath::Encode` bodies are `0x552` bytes and both
`Decode` bodies are `0x31e` bytes, with case-for-case identical switch groups. Length is **37**
(indices 0–36; max case is `0x24`).

Evidence — v95 `Decode`@`0x667920` / `Encode`@`0x666e20`, v92 `Decode`@`0x65ad60` /
`Encode`@`0x65a260`:

| Group | Fields after the type byte | Indices | `Type` |
|---|---|---|---|
| NORMAL | `x,y,vx,vy,fh`, `fhFallStart` iff idx==12, `xOffset,yOffset` | 0, 5, 12, 14, 35, 36 | `NORMAL` |
| JUMP | `vx,vy` (x/y inherited from the previous element) | 1, 2, 13, 16, 18, 31, 32, 33, 34 | `JUMP` |
| TELEPORT | `x,y,fh` | 3, 4, 6, 7, 8, 10 | `TELEPORT` |
| STAT_CHANGE | `bStat` only, **no** `bMoveAction`/`tElapse` | 9 | `STAT_CHANGE` |
| START_FALL_DOWN | `vx,vy,fhFallStart` | 11 | `START_FALL_DOWN` |
| FLYING_BLOCK | `x,y,vx,vy` | 17 | `FLYING_BLOCK` |
| default | nothing extra (`bMoveAction`+`tElapse` only) | 15, 19, 20–30 | `DEFAULT` |

Indices 20–30 are an explicit `case` group that reads nothing and inherits `x/y/vx/vy` from the
running header state; 15 and 19 fall to the `default:` arm. Both shapes are 3 bytes on the wire and
both map onto Atlas's bare `Element` decoder — i.e. `Type: "DEFAULT"`.

Flat array to be written into both templates (index → `Name`/`Type`):

```
 0 NORMAL/NORMAL            13 UNKNOWN/JUMP             26 UNKNOWN/DEFAULT
 1 UNKNOWN/JUMP             14 UNKNOWN/NORMAL           27 UNKNOWN/DEFAULT
 2 UNKNOWN/JUMP             15 UNKNOWN/DEFAULT          28 UNKNOWN/DEFAULT
 3 UNKNOWN/TELEPORT         16 UNKNOWN/JUMP             29 UNKNOWN/DEFAULT
 4 UNKNOWN/TELEPORT         17 FLYING_BLOCK/FLYING_BLOCK 30 UNKNOWN/DEFAULT
 5 UNKNOWN/NORMAL           18 UNKNOWN/JUMP             31 UNKNOWN/JUMP
 6 UNKNOWN/TELEPORT         19 UNKNOWN/DEFAULT          32 UNKNOWN/JUMP
 7 UNKNOWN/TELEPORT         20 UNKNOWN/DEFAULT          33 UNKNOWN/JUMP
 8 UNKNOWN/TELEPORT         21 UNKNOWN/DEFAULT          34 UNKNOWN/JUMP
 9 STAT_CHANGE/STAT_CHANGE  22 UNKNOWN/DEFAULT          35 UNKNOWN/NORMAL
10 UNKNOWN/TELEPORT         23 UNKNOWN/DEFAULT          36 UNKNOWN/NORMAL
11 START_FALL_DOWN/START_FALL_DOWN
                            24 UNKNOWN/DEFAULT
12 FALL_DOWN/NORMAL         25 UNKNOWN/DEFAULT
```

Execution re-derives this table independently against both IDBs and records it per FR-1.4 in
`movement-types-derivation.md`; the table above is the design-phase derivation that establishes
feasibility and shape, not a substitute for that evidence document. Where execution's reading
differs from this table, execution's reading wins and the divergence is recorded.

---

## 4. Amendment A — the movement header gained `vx`/`vy` (FR-0)

### 4.1 The finding

`Movement.Decode` (`movement.go:33-64`) reads `StartX` (int16), `StartY` (int16), then the element
count. That matches v83, v87, and JMS. It does **not** match v92 or v95.

| Version | `CMovePath::Encode` header | Evidence |
|---|---|---|
| GMS v83 | `Encode2(x)`, `Encode2(y)`, `Encode1(count)` | `0x68a563` → `0x68a57c`, `0x68a592`, `0x68a5c3` |
| GMS v87 | `Encode2`, `Encode2`, `Encode1(count)` | `0x6c70fe` → `0x6c7118`, `0x6c712e`, `0x6c715f` |
| JMS v185 | `Encode2`, `Encode2`, `Encode1(count)` | `0x70b6c4` → `0x70b6de`, `0x70b6f4`, `0x70b725` |
| **GMS v92** | `Encode2(x)`, `Encode2(y)`, **`Encode2(vx)`**, **`Encode2(vy)`**, `Encode1(count)` | `0x65a260` → `0x65a284`, `0x65a29f`, `0x65a2ba`, `0x65a2d5`, `0x65a306` |
| **GMS v95** | `Encode2(x)`, `Encode2(y)`, **`Encode2(vx)`**, **`Encode2(vy)`**, `Encode1(count)` | `0x666e20` → `0x666e44`, `0x666e5f`, `0x666e7a`, `0x666e95`, `0x666ec6` |

The decode side confirms it symmetrically: v95 `Decode`@`0x667920` reads four `Decode2` (`0x667938`,
`0x667951`, `0x667970`, `0x66797c`) before `Decode1(count)` at `0x667987`; v92 `Decode`@`0x65ad60`
does the same at `0x65ad78`, `0x65ad91`, `0x65adb0`, `0x65adbc`, `0x65adc7`.

The two new fields are not decoration — they seed the running velocity that the index-20–30 arms
inherit (`v9->vx = b; v9->vy = ovy;` at v95 `0x667b15`/`0x667b19`). They are part of the same v88
movement rework that added `xOffset`/`yOffset` to the NORMAL element, which `movement.go:129-137`
already gates.

### 4.2 Why this must be in scope

The four header bytes precede the element count. With them unread, `numElems` is parsed from the
low byte of `vx`, and every subsequent read is garbage — regardless of how correct `types` is. The
PRD's user stories ("my character's movement is received correctly", "monsters hold their
foothold") are unreachable without this. Per CLAUDE.md's *No Deferring Producible Work*, this is a
prerequisite that can be produced right now, on this branch, and should be.

### 4.3 Design

Add two fields to `model.Movement` and gate them in both directions, keeping `Encode` and `Decode`
textually identical the way the existing `XOffset`/`YOffset` pair is:

```go
type Movement struct {
    StartX   int16
    StartY   int16
    StartVx  int16   // GMS v88+ only
    StartVy  int16   // GMS v88+ only
    Elements []MovementCodec
}
```

```go
// Decode, after StartY:
// StartVx/StartVy are GMS v88+ (the same movement rework that added
// XOffset/YOffset to NormalElement). v87 and JMS write x,y,count only:
//   v83 CMovePath::Encode@0x68a563, v87 @0x6c70fe, jms @0x70b6c4 — 2 Encode2 + Encode1.
//   v92 @0x65a260, v95 @0x666e20 — 4 Encode2 + Encode1.
// MUST stay textually identical to Encode.
if t.IsRegion("GMS") && t.MajorAtLeast(88) {
    m.StartVx = r.ReadInt16()
    m.StartVy = r.ReadInt16()
}
```

**Predicate shape.** Note this is `IsRegion("GMS") && MajorAtLeast(88)`, *not* the
`!IsRegion("GMS") || MajorAtLeast(88)` shape used for `XOffset`/`YOffset`. JMS v185 was checked
directly (`0x70b6c4`) and writes the two-field header, so JMS must be **excluded** here even though
it is *included* by the `XOffset` gate. Reusing the `XOffset` predicate by reflex would break JMS
movement — a live regression on a currently-working version.

**Boundary choice: 88 vs 92.** Observed: v87 no, v92 yes. v88–v91 have no IDB in the set, so the
exact boundary is unobservable from binaries. Two options:

- **(a) `MajorAtLeast(88)` — recommended.** Consistent with the `XOffset`/`YOffset` gate already in
  the file, which pins the same client rework at 88. `deploy/k8s/base/versions.json` ships no GMS
  version between 87 and 92, so 88 vs 92 is behaviourally indistinguishable for every tenant Atlas
  can actually serve; consistency with the adjacent gate is then the only tiebreaker, and it also
  keeps the two boundaries readable as one rework rather than two unrelated numbers.
- (b) `MajorAtLeast(92)` — asserts only what was observed. Rejected: it splits one rework across two
  different constants for no observable benefit, and a future v88–v91 bring-up would have to
  reconcile them anyway.

**Round-trip impact.** Atlas decodes a client movement and re-encodes it outbound; `StartVx`/
`StartVy` ride through unchanged. Server-synthesized movement (no inbound path to copy) writes
`0,0`, which is what the client itself sends at rest.

**Test impact.** `libs/atlas-packet/model/movement_test.go` is the declared byte oracle for the
move-path blob; it gains v92 and v95 header cases. The wrapper fixtures
(`character/clientbound/movement_test.go`, `monster/…`, `pet/…`) assert the blob *equals*
`model.Movement.Encode`'s output rather than pinning literal bytes, so they stay green without
edits — but they must be run, not assumed.

**Matrix note.** The existing `packet-audit:verify … version=gms_v95` markers on those movement
fixtures pinned a header layout the v95 client does not read. That is an instance of the known
"round-trip fixture ≠ client-validated" failure mode. This design does **not** open a fixture
campaign (PRD §2 non-goal stands); it records the correction in the derivation document so a later
matrix pass has the evidence.

---

## 5. Opcode derivation and Amendment B

All opcodes below come from the packet constructor immediate at the sender, reached by walking
xrefs into `CMovePath::Flush` (v95 `0x668160`, v92 `0x65b5a0`) — six senders per client:
mob, npc, dragon, pet, summoned, user.

### 5.1 GMS v95 — `CharacterMoveHandle` = `0x2C` (registry was right)

`CVecCtrlUser::EndUpdateActive` @`0x9a0d20` builds `COutPacket(44)` at `0x9a0ee3`, then
`Encode4(~drInfo[0])`, `Encode4(~drInfo[1])`, `Encode1(FieldKey)`, `Encode4(~drInfo[2])`,
`Encode4(~drInfo[3])`, `Encode4(Crc)`, `Encode4(dwKey)`, `Encode4(Crc32)`, then
`CMovePath::Flush`.

So `docs/packets/registry/gms_v95.yaml:2290-2294`'s **opcode 44 (`0x2C`) is correct**, and the
PRD's monotonic-drift suspicion is a false alarm — the serverbound opcode table simply is not
monotonic across versions. The `fname` (`CUserLocal::OnKey`) is the misleading part; the true sender
is already recorded in that row's `fname_alts` as `CVecCtrlUser::EndUpdateActive`. Proposed
registry change is therefore narrow: **promote `CVecCtrlUser::EndUpdateActive` to `fname` and demote
`CUserLocal::OnKey` to `fname_alts`; leave the opcode alone.**

This decompile also independently confirms the `MajorAtLeast(84)` gate on the
`dr0/dr1/dr2/dr3/dwKey/crc32` header in `character/serverbound/move.go` holds at v95.

### 5.2 GMS v92 — `PetMovementHandle` = `0xC4`

v92's senders are unnamed (`sub_*`). They were resolved by positional correspondence against v95's
named set — the same technique that produced the v92 IDB's existing names — and then confirmed
structurally, so the mapping does not rest on address ordering alone:

| Role | v95 (named) | v95 opcode | v92 (unnamed) | v92 opcode | Structural confirmation |
|---|---|---|---|---|---|
| Mob | `CMob::GenerateMovePath` `0x651100` | 227 = `0xE3` ✓template | `sub_6447A0` | `0xDC` ✓template | — |
| Npc | `CNpc::GenerateMovePath` `0x671590` | — | `sub_664DC0` | `0xEA` | not routed by Atlas |
| Dragon | `CVecCtrlDragon::EndUpdateActive` `0x996570` | 214 = `0xD6` | `sub_96F190` | `0xD3` | neither encodes a prefix |
| **Pet** | `CVecCtrlPet::EndUpdateActive` `0x99f5a0` | 199 = `0xC7` ✓template | `sub_9781A0` | **`0xC4`** | both `EncodeBuffer(petLockerSN, 8)` before `Flush` |
| **Summoned** | `CVecCtrlSummoned::EndUpdateActive` `0x9a0700` | 207 = `0xCF` ✓template | `sub_9792D0` | **`0xCC`** | both `Encode4(dwSummonedID)` before `Flush` |
| User | `CVecCtrlUser::EndUpdateActive` `0x9a0d20` | 44 = `0x2C` | `sub_9798F0` | `0x2E` ✓template | both write the 8-field anti-cheat header |

Three v95 opcodes in that table (`0xE3`, `0xC7`, `0xCF`) and one v92 opcode (`0x2E`, `0xDC`) match
what the templates already carry, which cross-validates the whole mapping before it is used to
derive anything new.

**v92 `PetMovementHandle` = `0xC4`**, and the pet move packet carries an 8-byte pet-locker SN before
the move path — matching v95's shape, so `libs/atlas-packet/pet/serverbound` needs no change.

### 5.3 Amendment B — v92 `SummonMoveHandle` is at the wrong opcode

`template_gms_92_1.json` registers `SummonMoveHandle` at **`0xC8`**. The derived v92 summon-move
opcode is **`0xCC`** (`sub_9792D0` @`0x97932d`).

`0xC8` is `template_gms_87_1.json`'s `MonsterMovementHandle` opcode — the fingerprint of a
copy-paste when the v92 template was seeded. The consequence is two-sided: v92 summon movement is
unrouted, and whatever the v92 client actually sends on `0xC8` is fed to the summon-move decoder.

Correcting this is a one-line change to the same file the task already edits, and leaving it would
mean shipping a `types` array onto a handler wired to the wrong packet. **Proposed: change
`SummonMoveHandle`'s `opCode` from `0xC8` to `0xCC`,** and re-sort (`0xCC` moves after `0xC4`, both
still ahead of `0xDC` — `tools/template-opcode-order-guard.sh` covers this).

Execution should also confirm from the v92 client what, if anything, legitimately owns `0xC8`, and
record it. If a real v92 handler collides there, that is a stop-and-ask.

### 5.4 IDB hygiene

Per CLAUDE.md's RE discipline, execution names the five v92 `sub_*` functions above with their v95
mangled symbols before harvesting evidence, so the addresses cited in the evidence document resolve
to names on re-read.

---

## 6. Template changes

`services/atlas-configurations/seed-data/templates/template_gms_92_1.json`:

| Handler | opCode | Change |
|---|---|---|
| `CharacterMoveHandle` | `0x2E` (confirmed) | add derived 37-entry `types` |
| `PetMovementHandle` | `0xC4` (**new entry**) | add entry: `validator: "LoggedInValidator"`, `services: ["channel"]`, derived `types` |
| `SummonMoveHandle` | `0xC8` → **`0xCC`** | correct opCode (§5.3) + add derived `types` |
| `MonsterMovementHandle` | `0xDC` (confirmed) | add derived 37-entry `types` |

`services/atlas-configurations/seed-data/templates/template_gms_95_1.json`:

| Handler | opCode | Change |
|---|---|---|
| `CharacterMoveHandle` | `0x2C` (**new entry**) | add entry: `validator: "LoggedInValidator"`, `services: ["channel"]`, derived `types` |
| `PetMovementHandle` | `0xC7` (confirmed) | add derived 37-entry `types` |
| `SummonMoveHandle` | `0xCF` (confirmed) | add derived 37-entry `types` |
| `MonsterMovementHandle` | `0xE3` (confirmed) | add derived 37-entry `types` |

`services/atlas-configurations/seed-data/templates/template_gms_48_1.json`:

| Handler | opCode | Change |
|---|---|---|
| `SummonMoveHandle` | `0x78` | copy that template's own `CharacterMoveHandle.types` (23 entries) verbatim — FR-4.1, no derivation |

Both new entries go at their sorted position in `handlers` (FR-3.4). `CharacterInventoryMoveHandle`
is untouched everywhere (FR-2.4). No other template changes (FR-4.2).

---

## 7. Enforcing the invariants (FR-5)

FR-5 asks for a mechanical check across all 11 templates. Three ways to satisfy it:

- **(a) A permanent guard script + CI job — recommended.**
  `tools/template-movement-types-guard.sh`, modelled directly on
  `tools/template-opcode-order-guard.sh` (bash preamble + inlined `python3` heredoc, no Go
  toolchain, run from repo root, non-empty diagnostics → non-zero exit), registered in
  `.github/workflows/pr-validation.yml` alongside `template-opcode-order-guard` (job at :313, plus
  the `needs:` list at :713 and the result rollup at :734), and listed in CLAUDE.md's Build &
  Verification section as item 11.
- (b) A one-off script under the task folder, run once and cited in the PR.
- (c) Manual inspection of the three touched files.

(c) fails the PRD outright ("checked mechanically over all 11 templates, not by inspection of the
three touched files"). (b) satisfies the letter of FR-5 but not its spirit: **this exact defect has
now shipped twice** — task-179 fixed v48/61/72/79 and missed v48's Summon handler, and the v92/v95
templates were seeded with no `types` at all. A check that only runs once cannot catch the third
occurrence. The marginal cost of (a) over (b) is the CI wiring, which is a known, four-touchpoint
pattern in this repo.

Checks the guard performs, over every `template_*.json`:

1. Every handler named `CharacterMoveHandle`, `MonsterMovementHandle`, `PetMovementHandle`, or
   `SummonMoveHandle` has a non-empty `options.types` array. (FR-5.1)
2. Within one template, all such arrays are byte-identical. (FR-5.1)
3. Every entry is an object with string `Name` and string `Type`; every `Type` is one of `NORMAL`,
   `JUMP`, `TELEPORT`, `START_FALL_DOWN`, `FLYING_BLOCK`, `STAT_CHANGE`, `DEFAULT`. (FR-5.2)
4. At most one entry per array is named `FALL_DOWN`. (FR-5.3)
5. `CharacterInventoryMoveHandle` is explicitly excluded from (1) — it is inventory item movement,
   not map movement, and correctly carries no `types`.

Check 3's `Type` allowlist is the one that catches the silent failure mode: a typo'd `Type` degrades
that one index to the 3-byte generic decode with no log line, which is the original bug in
miniature.

The guard is written and made to pass on the *current* tree state first (i.e. it must fail on the
pre-fix tree, naming exactly the eight known-bad cells from `current-state.md`), then the templates
are fixed and it goes green. A guard that never demonstrated a failure is not a verified guard.

---

## 8. Live tenant reconcile (FR-6)

Seed templates apply only at provisioning. Existing v92/v95 tenants keep their stored socket
configuration, so the fix does not reach them without a data operation. This is the known
"tenant-config seed endpoints never invoked" / "new opcodes missing from live tenant config" class.

Procedure, per environment, per v92/v95 tenant:

1. `GET /configurations/tenants` — enumerate; select tenants whose `region`/`majorVersion` is
   GMS 92 or GMS 95.
2. `GET /configurations/tenants/{tenantId}` — read the stored configuration. **Reconcile against
   this document, not against a from-scratch template push**: swap only the four move-handler
   entries (adding the two that are absent, correcting v92's Summon opCode), leaving every other
   handler, writer, and world/character block exactly as stored. A wholesale template overwrite
   would silently revert tenant-specific customization.
3. `PATCH /configurations/tenants/{tenantId}` with the JSON:API envelope
   `{"data":{"type":"tenants","id":"…","attributes":{…}}}` and the standard tenant headers.
4. **Read back** with a fresh `GET` and assert all four move handlers are present, each with a
   `validator` and a non-empty 37-entry `types`, and that v92's Summon sits at `0xCC`. Quote the
   actual response. The PATCH response is not evidence — a handler entry missing its `validator` is
   accepted at the transport layer and then silently dropped at load time.
5. Restart / roll the affected `atlas-channel` pods only if the service caches socket configuration
   at startup; confirm from the code path rather than assuming either way.

Recorded in `reconcile.md` in the task folder: the environments and tenant ids patched, the
before/after handler shape, and the read-back output — so the operation is repeatable for any
environment not covered during this task (FR-6.3).

The primary post-reconcile signal is negative: `movementPathAttrFromOptions`'s
`"Code [%d] not configured for use in movement…"` error line (`movement.go:289`) must disappear from
v92/v95 channel logs. Its continued presence means the reconcile did not take.

---

## 9. Risks and resolved unknowns

**Resolved during design — no longer risks:**

- *Are the rand-count fields on the wire?* v92/v95 read two extra `int16`
  (`usRandCnt`, `usActualRandCnt`) per element when `CClientOptMan::GetOpt(2)` is truthy (v95
  `0x667a57`). `CClientOptMan::GetOpt` @`0x4ac700` returns 0 for any key absent from `m_mOpt`, and
  `m_mOpt` is populated **only** by `CClientOptMan::DecodeOpt` @`0x4acb30` from a server-sent
  option list. Atlas writes a zero option count (`set_field.go:49-51`, `w.WriteShort(0)`), so the
  map stays empty and the fields are never on the wire. **Coupling to record:** if Atlas ever sends
  a non-empty client-option list including type 2, `movement.go` must gain those two reads.
- *v95 `CharacterMoveHandle` opCode.* `0x2C`, confirmed (§5.1). Registry opcode stands.
- *v92 `PetMovementHandle` opCode.* `0xC4`, confirmed (§5.2).
- *Array length.* 37 for both v92 and v95 — neither 25 (v87) nor a further-grown jms-like table.

**Open risks:**

| Risk | Mitigation |
|---|---|
| The v88–v91 header boundary is unobservable (no IDB). | Behaviourally moot — no such tenant version ships (`versions.json`). Rationale recorded in the code comment so a future bring-up knows the constant was chosen, not measured. |
| `movement.go` is shared by every version; a mistake in the gate breaks working versions. | The gate is region+version explicit and JMS-excluding by direct evidence (§4.3). Full `go test ./...` in `libs/atlas-packet` plus the existing v61/v72/v83/v84/v87/jms fixtures is the regression net. |
| Something legitimate already owns v92 `0xC8`. | Execution confirms from the client before removing the handler from that opcode; a real collision is a stop-and-ask, not a silent overwrite. |
| Reconcile overwrites tenant-specific config. | Surgical swap of four entries, never a template push (§8 step 2), with read-back proof. |
| IDA session ids rotate between sessions. | Always `idb_list` and match by **filename**, then cross-check the decompiled address against the one cited here before trusting a read. |

---

## 10. Verification plan

Derivation:

- `movement-types-derivation.md` in the task folder: per version, per index — client function, address,
  observed field sequence, resulting `Name`/`Type`. Includes the v83 control run (§2.2) as the
  method's calibration, and the v87→v92 header delta (§4.1).
- Every opcode in §5 cited to a `COutPacket` constructor address.

Code and templates:

- `go test -race ./...` and `go vet ./...` clean in `libs/atlas-packet`.
- New v92/v95 header cases in `libs/atlas-packet/model/movement_test.go`; existing v61–v87/jms cases
  unchanged and green (proving the gate excludes them).
- `tools/template-movement-types-guard.sh` — demonstrated failing on the pre-fix tree, green after.
- `tools/template-opcode-order-guard.sh` exits 0 (both new entries plus the v92 Summon opCode move).
- `tools/lint.sh --check` exits 0.
- `git diff --stat` vs `main` confined to: the three templates, `libs/atlas-packet/model/movement.go`
  + its test, the new guard script, the CI workflow, CLAUDE.md's verification list,
  `docs/packets/registry/gms_v95.yaml` (fname promotion only), and the task docs.

Live:

- Post-PATCH read-back per tenant, response quoted (§8 step 4).
- v92/v95 channel logs free of the `"not configured for use in movement"` error.

Out of band, not gating: the requester play-tests v92 and v95.

---

## 11. Scope delta vs the PRD — decisions requested

Everything in §2, §3, §6, §7, §8 implements the PRD as written. Four items change its stated scope;
all four are on the same branch and worktree, and none is deferrable to a follow-up task without
leaving this one non-functional.

1. **`libs/atlas-packet/model/movement.go` is modified** (§4), against PRD §2's non-goal. Without
   it the `types` arrays cannot take effect on v92/v95. Also touches `movement_test.go`.
2. **`template_gms_92_1.json`'s `SummonMoveHandle` opCode is corrected** `0xC8` → `0xCC` (§5.3),
   beyond the PRD's "add the missing entries" framing.
3. **A permanent guard script and CI job are added** (§7), where the PRD only required a mechanical
   check. Recommended over a one-off because this defect class has now recurred.
4. **`docs/packets/registry/gms_v95.yaml` changes only its `fname`**, not the opcode (§5.1) — the
   PRD anticipated a possible opcode correction; the derivation says the opcode was right.

Items 1 and 2 are load-bearing: dropping either makes the delivered task non-functional on the
versions it targets. Item 3 is a judgment call the requester can decline in favour of a one-off
script without affecting correctness. Item 4 is a two-line documentation fix.
