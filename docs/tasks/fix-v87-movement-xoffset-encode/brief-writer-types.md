# Brief — give the movement writers their `types` table

## Why

Commit `bd3a09003` made `SummonMove.Encode` / `DragonMove.Encode` re-serialize
the client's move-path blob so a GMS v87 observer is not sent the per-element
`XOffset`/`YOffset` pair it never reads (root cause in `diagnosis.md`). That fix
is currently INERT in production: `model.ReserializeMovePath` needs the tenant's
movement `types` table, taken from the `options` map the writer is registered
with, to classify a fragment as NORMAL. The `SummonMove` and `DragonMove`
**writer** entries carry no `options.types`, so every fragment falls back to the
bare `model.Element`, the guard in `ReserializeMovePath` trips, and the blob goes
out verbatim exactly as before the fix.

"Writers don't get the table" is NOT the convention, whatever
`docs/packets/TEMPLATE_CONVENTIONS.md` currently says. The `CharacterMovement`
and `NPCAction` writer entries already carry it, and `PetMovement`'s writer
carries it on 7 of the 11 templates. The gaps below are omissions.

The `PetMovement` gap is its own live defect, independent of summon/dragon:
without `types`, `NormalElement.Encode`'s `FALL_DOWN` name check never fires, so
`fhFallStart` is silently dropped from outbound pet movement on those templates.

## Exact inventory

Templates live in `services/atlas-configurations/seed-data/templates/`. Measured
state — `W` = writer entry, `H` = handler entry, number = size of `options.types`:

| template | needs `types` added to writer | copy from (same file) |
|---|---|---|
| `template_gms_12_1.json` | `SummonMove` (0) | `SummonMoveHandle` (9) |
| `template_gms_48_1.json` | `SummonMove` (0) | `SummonMoveHandle` (23) |
| `template_gms_61_1.json` | `SummonMove` (0) | `SummonMoveHandle` (23) |
| `template_gms_72_1.json` | `SummonMove` (0) | `SummonMoveHandle` (23) |
| `template_gms_79_1.json` | `SummonMove` (0) | `SummonMoveHandle` (23) |
| `template_gms_83_1.json` | `SummonMove` (0), `DragonMove` (0) | respective `*Handle` (23) |
| `template_gms_84_1.json` | `SummonMove` (0), `DragonMove` (0) | respective `*Handle` (24) |
| `template_gms_87_1.json` | `SummonMove` (0), `DragonMove` (0), `PetMovement` (0) | respective `*Handle` (25) |
| `template_gms_92_1.json` | `SummonMove` (0), `DragonMove` (0), `PetMovement` (0) | respective `*Handle` (37) |
| `template_gms_95_1.json` | `SummonMove` (0), `DragonMove` (0) | respective `*Handle` (37) |
| `template_jms_185_1.json` | `SummonMove` (0), `DragonMove` (0), `PetMovement` (0) | respective `*Handle` (33) |

`PetMovement`'s writer already has the table on 48/61/72/79/83/84/95 — leave
those alone. `gms_12_1` has no dragon and no pet move entries at all; only
`SummonMove` applies there.

The table to copy is ALWAYS the matching handler entry IN THE SAME FILE.
`CMovePath::Decode` is one shared client function, so the outbound table is by
construction the same table that parsed the blob inbound. Do NOT copy across
templates, do NOT hand-author a table, and do NOT normalise or reorder entries —
index position IS the attr code.

## The guard

`tools/template-movement-types-guard.sh` covers handler entries only. Extend it
so a movement WRITER entry that needs the table cannot silently lose it again —
this whole class of bug is "the table is absent and everything still appears to
work, just wrong on the wire." Read the script first and follow its existing
structure and failure-message style rather than bolting on a second mechanism.

The guard must be precise about which writers require the table. `SummonMove`,
`DragonMove`, `PetMovement`, `CharacterMovement`, `NPCAction` all encode a
`model.Movement`. Derive the list from the repo rather than hardcoding this
sentence if there is a reliable way to; if you hardcode, say so in a comment and
explain how to extend it.

## Also update

`docs/packets/TEMPLATE_CONVENTIONS.md` — it documents a handler-only rule that
the data contradicts. Correct it to state that any entry whose codec encodes or
decodes a `model.Movement` carries the table, and why (the fragment attr → shape
resolution is needed in BOTH directions).

## Verification

- `cd libs/atlas-packet && go test ./...` passes.
- `tools/template-movement-types-guard.sh` exits 0, AND fails when you
  temporarily delete one of the writer tables you just added. Show both.
- The other template guards still pass: `tools/template-opcode-order-guard.sh`,
  `tools/template-duplicate-binding-guard.sh`, `tools/template-symbol-check.sh`.
- From the repo root: `go run ./tools/packet-audit matrix --check`,
  `fname-doc --check`, `operations --check`, `dispatcher-lint` — all exit 0.
- Prove the summon/dragon fix is now LIVE, not just present: add or extend a test
  that drives the v87 `SummonMove`/`DragonMove` writer with the template's real
  options and asserts the emitted blob has dropped the pair. This is the whole
  point of the change — without it we have only re-proved that
  `ReserializeMovePath` works when handed a table by hand.
- Do NOT run `tools/verify.sh`; the controller dispatches that separately.

## Scope

Do not touch the gates in `libs/atlas-packet/model/movement.go`, the codecs
committed in `540929015` / `56e9858c5` / `bd3a09003`, or the Kafka contract.
Templates, the guard script, the conventions doc, and tests only.

Worktree `.worktrees/fix-v87-movement-xoffset-encode`, branch
`fix-v87-movement-xoffset-encode`. Commit your work; leave the tree clean.
