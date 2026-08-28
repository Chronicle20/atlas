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

## Task 17: `npc-2111017.json`, `npc-2111018.json`, `npc-2111019.json` (Magatia lab pipes)

- Sources: `<cosmic>/scripts/npc/2111017.js`, `2111018.js`, `2111019.js`
  (external Cosmic checkout, not available in this repository). The brief's
  state/outcome tables (task-17-brief.md) were used as the sole authority; no
  direct read of the Cosmic source was needed or performed.
- Modeled `cm.getQuestProgress(23339, 1)` / `cm.setQuestProgress(23339, 1, n)`
  as `questProgress` conditions and `set_quest_progress` operations carrying
  `referenceId: "23339"` (quest id) and `step: "1"` / `infoNumber: "1"`
  (info number), per `conversation/rest.go:110`.
- `questStatus` values follow `docs/npc_conversation_conversion_spec.md:426-437`
  (`1` not-started, `2` started, `3` completed). The `gate` state's outcome
  order — `questStatus = 2` (started) routes into `checkProgress`,
  `questStatus = 3` (completed) routes to `warpIn`, and the default (`[]`, not
  started) falls to `null` (do nothing) — is taken verbatim from the brief's
  Step 1 table.
- The three files are otherwise identical apart from the one `checkProgress`
  branch and its target state, per the brief:
  - `2111017`: progress `= 0` → `advanceTo1` (sets progress `1`), disposes
    (`nextState: null`).
  - `2111019`: progress `= 1` → `advanceTo2` (sets progress `2`), disposes.
  - `2111018`: progress `= 2` → `advanceTo3` (sets progress `3`) **and**
    routes to `askPassword` in the same turn — the one asymmetry called out
    in the brief; `2111017` and `2111019` dispose instead.
- All three share the password match `"my love Phyllia"` → `unlock`, which
  sets progress to `4` and warps to map `261000001` portal `1`; a wrong
  answer falls through to a `dialogue` state (`"#rWrong!"`, `Ok`/`Exit`
  choices, both dispose).
- The ten seed files per NPC (`gms/{48,61,72,79,83,84,87,92,95}_1` and
  `jms/185_1`) are byte-identical (verified via `md5sum`); one file per NPC
  was authored under `gms/83_1` and copied to the other nine paths.
- Used `set_quest_progress`/`questProgress` (not `local:get_quest_progress`)
  since these three NPCs only ever write and re-read the info-number-1
  progress they own; no cross-NPC quest-progress read was needed here.
- Validated with `go run ./tools/catalog-lint deploy/seed` — no errors. Every
  `nextState` resolves within its own file or is `null`.
