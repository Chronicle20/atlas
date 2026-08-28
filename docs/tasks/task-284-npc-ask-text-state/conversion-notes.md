# Conversion Notes

Deviations and dependencies discovered while converting Cosmic NPC scripts to
`npc-conversation` seed JSON for the `askText` content tasks (16–20). Each
section below covers exactly one task; do not add sections for other tasks
from this file.

## Task 16: `npc-2111024.json` (Magatia lab door)

- Source: `<cosmic>/scripts/npc/MagatiaPassword.js` (external Cosmic checkout,
  not available in this repository). The brief's state table (task-16-brief.md)
  was used as the sole authority; no direct read of the Cosmic source was
  needed or performed.
- `cm.getQuestProgress(3360)` and `cm.setQuestProgress(3360, 1)` are the
  one/two-argument overloads that delegate to info number `0`
  (`AbstractPlayerInteraction.java:402-403,414-415`). Encoded as
  `infoNumber: "0"` on both the `local:get_quest_progress` operation and the
  `set_quest_progress` operation.
- Used `local:get_quest_progress` (the `local:`-prefixed operation landed by
  Task 11) rather than the un-prefixed name still shown in prd.md §4.5 and
  design.md §8 — those docs are corrected by Task 21.
- The `sp_jenu` / `sp_alca` portal-name split reproduces
  `"sp_" + ((cm.getMapId() == 261010000) ? "jenu" : "alca")` via a `mapId`
  condition (`operator: "="`, `value: "261010000"`) on the `openDoor` state's
  first outcome, falling through to the unconditional `warpAlca` outcome.
- `secretDoor.js`'s quest-complete and quest-unstarted branches are
  deliberately excluded — they belong to the portal object, not this NPC, and
  are out of scope for this task (`context.md` D5).
- `deploy/seed/gms/12_1/` was excluded per the brief (that template registers
  no NPC conversation handlers).
- The ten seed files (`gms/{48,61,72,79,83,84,87,92,95}_1` and `jms/185_1`)
  are byte-identical (verified via `md5sum`); one file was authored and
  copied to the other nine paths.
- Validated with `go run ./tools/catalog-lint deploy/seed` — no errors.
