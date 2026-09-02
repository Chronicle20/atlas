# Tenant socket-template conventions

The tenant seed templates under
`services/atlas-configurations/seed-data/templates/template_<region>_<major>_<minor>.json`
drive per-version packet routing. Two arrays live under `socket`:

- `handlers` — serverbound: `opCode` → `handler` (+ `validator`, optional
  `services`, `options`).
- `writers` — clientbound: `opCode` → `writer` (+ optional `services`,
  `options`).

## Rule: ascending opcode order (enforced)

**Both `handlers` and `writers` MUST be listed in strictly ascending `opCode`
order within each template.** When you add a handler or writer, insert it at its
sorted position — never append it next to a semantically-related entry (e.g. do
not drop the portable-chair handler right after the heal-over-time handler just
because they are both recovery packets; place it by its opcode).

Why this is a rule and not just a nicety:

- The arrays are loaded into an **opcode-keyed dispatch map**, so order is
  functionally irrelevant to the running server. That is exactly why it drifts —
  nothing at runtime notices when it is wrong.
- Sorted arrays are **diffable and mergeable**: a new entry shows up as a single
  localized insertion, and two branches adding different opcodes do not fight
  over the same append point.
- Sorted arrays are **auditable by eye**: "is opcode `0xNN` already routed on
  this version?" is answerable by scanning, and `verify-serverbound` / template
  cross-checks read cleanly.

## Rule: move entries carry a `types` table (enforced)

**Every entry whose codec encodes or decodes a `model.Movement` carries the
table — writers as well as handlers.** The attr→shape resolution the table
provides is needed in BOTH directions: `CMovePath` is one shared client
function, so the table that parsed a blob inbound is by construction the table
that must serialize it outbound.

The six **move** handlers — `CharacterMoveHandle`, `MonsterMovementHandle`,
`PetMovementHandle`, `SummonMoveHandle`, `DragonMoveHandle`, `NPCActionHandle` —
and the six **move** writers — `CharacterMovement`, `MoveMonster`,
`PetMovement`, `SummonMove`, `DragonMove`, `NPCAction` — MUST each carry a
non-empty `options.types` array, and all such arrays within one template MUST
be byte-identical. (A template legitimately omits an *entry* for a packet its
version does not have — `template_gms_12_1.json` has no dragon or pet movement
at all — but an entry that exists carries the table.)

`libs/atlas-packet/model/movement.go` decodes a movement fragment by reading a
one-byte element type and looking it up **as an array index** in that entry's
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

Failure modes, all of which have shipped:

- **Table absent on a handler** → the lookup returns `("NOT_FOUND", "DEFAULT")`,
  no decoder branch matches, and every fragment takes the bare 3-byte `Element`
  decoder against a fragment 9–15 bytes wide. Loud: one error log line per
  fragment.
- **One `Type` typo'd** → the same 3-byte degradation for that one index, with
  **no** log line at all.
- **Table absent on a writer** → silent and invisible server-side; the packet
  still goes out, just wrong on the wire. `model.ReserializeMovePath` (summon
  and dragon movement) cannot classify a fragment, bails out, and reships the
  GMS v87 `XOffset`/`YOffset` pair that v87's `CMovePath::Decode` never reads —
  the observing client's element loop desyncs and the summon teleports. And
  `NormalElement.Encode`'s reserved `FALL_DOWN` name check never fires, so
  `fhFallStart` is dropped from the outbound fragment.

Two name-based traps, in opposite directions:

- `CharacterInventoryMoveHandle` is **not** a move handler despite the name —
  it is inventory item movement and correctly carries no `types`.
- `NPCActionHandle` **is** a move handler despite the name — it decodes
  `model.Movement` through the same `options.types` map
  (`libs/atlas-packet/npc/serverbound/action.go`) because an NPC-action packet
  carries an optional embedded movement payload.

## Guard

Two scripts enforce this file's rules, both run in CI and listed in the
CLAUDE.md Build & Verification checklist:

- `tools/template-opcode-order-guard.sh` (repo root) checks every template's
  `handlers` and `writers` for ascending `opCode` order and exits non-zero on
  any descent. It runs in CI as the **Template Opcode Order Guard** job in
  `.github/workflows/pr-validation.yml`.
- `tools/template-movement-types-guard.sh` (repo root) checks every template's
  move handlers **and move writers** for a non-empty, byte-identical,
  well-formed `options.types` table and exits non-zero on any violation. The
  writer carrier set is derived from `libs/atlas-packet` (a clientbound codec
  holding a `model.Movement` field or calling `model.ReserializeMovePath`), so a
  new movement-carrying writer is covered the day it is added. It runs in CI as
  the **Template Movement Types Guard** job in
  `.github/workflows/pr-validation.yml`.

Run them locally before committing template edits:

```sh
tools/template-opcode-order-guard.sh
tools/template-movement-types-guard.sh
```

If `template-opcode-order-guard.sh` fails, move the offending entry to its
sorted position (the guard prints the exact `0xNN (handler) follows 0xMM
(handler)` pair). Re-sorting is safe — it changes only array order, never
behavior.

If `template-movement-types-guard.sh` fails, fix the reported entry's
`options.types` array in the reported template — either add the missing
table, correct the typo'd `Type`, or make it byte-identical to the other move
entries' tables in that same template. A writer's table is always copied from
the matching handler **in the same file** (`SummonMove` ← `SummonMoveHandle`,
and so on); never hand-author one and never copy across templates, since the
index position IS the attr code and the numbering is version-specific.

## Character presets must carry ids

Every entry in a seed template's `characters.presets` array must have a
non-empty `id`. The preset validator assigns a UUID only to presets that lack
one (`templates/characters/preset/validator.go`), and it runs on the PATCH
path - so an id-less preset in a seed file means the stored row diverges from
the file the moment the template is edited through the UI, and the "Differs
from image" badge (task-201) lights up for a reason unrelated to what the
operator changed.

Not machine-checked: the consequence is a spuriously-lit badge on one
template, not a gameplay failure. As of task-201, all eleven shipped templates
satisfy it - six carry presets, all with ids; five carry none.
