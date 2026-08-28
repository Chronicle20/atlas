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

## Task 18: `npc-1063011.json` (Thief/Puppeteer password merge)

- Sources: `<cosmic>/scripts/npc/ThiefPassword.js` and
  `<cosmic>/scripts/npc/PupeteerPassword.js` (external Cosmic checkout, not
  available in this repository). The brief's state table (task-18-brief.md)
  was used as the sole authority; no direct read of the Cosmic source was
  needed or performed.
- **Two Cosmic scripts, one Atlas NPC.** Cosmic lets the *portal* name the
  script it opens: `<cosmic>/scripts/portal/thief_in1.js` calls
  `pi.openNpc(1063011, "ThiefPassword")` and
  `<cosmic>/scripts/portal/enterDollcave.js` calls
  `pi.openNpc(1063011, "PupeteerPassword")`. Atlas keys conversations by NPC
  id and `atlas-portal-actions` has no "open NPC conversation" operation, so
  both scripts had to merge into the single `npc-1063011.json`, sharing the
  same `askText` prompt with two `matches` (`"Open Sesame"` → the Thief gate,
  `"Francis is a genius Puppeteer!"` → the Puppeteer gate) and a single
  `wrongPassword` fallback. **Portal-side implication for any future
  content-authoring or portal-action work touching `thief_in1` /
  `enterDollcave`:** neither portal can be modeled as "open a dedicated NPC
  conversation" — both must resolve to opening NPC 1063011, and the
  distinction between "which script the portal meant" only survives inside
  that NPC's `askText` matches, not at the portal layer. If Atlas ever gains
  a portal action that opens an NPC by id, it will point both portals at the
  same id and rely on this merge; it cannot recover which original Cosmic
  script fired.
- **Doll-cave map id derivation (brief Step 1).** WZ `Map.wz` is not checked
  into this repository and neither portal is seeded under
  `deploy/seed/*/portal-actions/portals/`, so the id could not come from
  static repo contents. It was derived instead from the live `atlas-data`
  service in the `atlas-main` Kubernetes namespace (tenant `GMS v83`,
  region `GMS`, majorVersion `83`, minorVersion `1`) via
  `GET /api/data/npcs/1063011/maps`, which returned NPC 1063011 spawned on
  map `105070300` ("The Cave of Evil Eye III", street name "Dungeon").
  `GET /api/data/maps/105070300/portals` confirmed portal id `3` (`in00`)
  carries `scriptName: "enterDollcave"`, matching the brief's citation
  verbatim. `puppeteerPreCheck`'s first outcome therefore gates on
  `{"type": "mapId", "operator": "=", "value": "105070300"}` AND
  `questStatus(21728) = 2`, per the brief's Step 2 table.
- `puppeteerGate1` and `puppeteerGate2` are two sequential states (not one
  `OR`ed condition list) because a `genericAction`'s condition list is an
  `AND`; Cosmic's gate is an `OR` of two quest conditions. Chained on
  failure, per the brief and consistent with Task 17's identical reasoning
  (`design.md` §9).
- `send_message` with `messageType: "PINK_TEXT"` reproduces Cosmic's
  `playerMessage(5, …)` (`operation_executor.go:2042`), per the brief.
- `enterDollcave.js`'s quest-completed pre-branch (20730 or 21734 completed
  → warp `105040201` portal `2`) belongs to the **portal**, not this NPC, and
  is out of scope here — the same exclusion applied to `secretDoor.js` in
  Task 16 and to the per-NPC branches in Task 17.
- The ten seed files (`gms/{48,61,72,79,83,84,87,92,95}_1` and `jms/185_1`)
  are byte-identical (verified via `md5sum`); one file was authored and
  copied to the other nine paths.
- Validated with `go run ./tools/catalog-lint deploy/seed` — no errors.

## Task 19: `npc-2091009.json` (Sealed Shrine entrance)

- Source: `<cosmic>/scripts/npc/2091009.js` (external Cosmic checkout, not
  available in this repository). The brief's state table (task-19-brief.md)
  was used as the sole authority; no direct read of the Cosmic source was
  needed or performed.
- **Deliberate ordering deviation.** `2091009.js` checks map occupancy
  *before* comparing the entered password. Atlas evaluates `matches` on the
  `askText` state itself (`askPassword`), so the password comparison
  necessarily happens first, with the occupancy check (`checkOccupancy`,
  `mapCapacity` condition on map `925040100`) only reachable after a correct
  password. The only observable behavioural difference: a player who types
  the wrong password into an already-occupied shrine now sees `"#rWrong!"`
  instead of `"Someone is already attending the Sealed Shrine."` — the warp
  itself is gated identically under both orderings, since no path reaches
  `warpIn` without both the correct password and an unoccupied,
  quest-eligible shrine.
- `mapCapacity` (condition type `MapCapacityCondition` in
  `libs/atlas-saga/validation.go:33`) takes the target map id in
  `referenceId` and the capacity threshold in `value`; verified against the
  brief's cited existing usage shape (`{"operator": ">=", "referenceId":
  "910220004", "type": "mapCapacity", "value": "5"}`).
- `questProgress`'s `step` key (`RestConditionModel.Step`,
  `conversation/rest.go:110`, backed by `ValidationConditionInput.Step` in
  `libs/atlas-saga/validation.go:67`) is a real, supported field — confirmed
  before use per the controller's flag on this task.
- `PINK_TEXT` is a real `messageType` value for `send_message`; confirmed
  present in already-seeded templates (e.g.
  `deploy/seed/gms/83_1/npc-conversations/npc/npc-1063011.json`).
- The ten seed files (`gms/{48,61,72,79,83,84,87,92,95}_1` and `jms/185_1`)
  are byte-identical (verified via `md5sum`); one file was authored under
  `gms/83_1` and copied to the other nine paths.
- Validated with `go run ./tools/catalog-lint deploy/seed` — no errors.

## Task 20: `npc-1092019.json` (Nautilus seagull quiz)

- Source: `<cosmic>/scripts/npc/1092019.js` (external Cosmic checkout, not
  available in this repository). The brief's state table (task-20-brief.md)
  is a transcription of the script made during planning and was used as the
  sole authority; no direct read of the Cosmic source was needed or
  performed.
- Progress lives on quest `6400`, info number `1`, read via `questProgress`
  (`referenceId: "6400"`, `step: "1"`) and written via `set_quest_progress`
  (`questId: "6400"`, `infoNumber: "1"`).
- **The `seagullProgress == 1` arm is omitted, not stubbed.** That branch
  routes into `cm.getEventManager("4jaerial").startInstance(…)`, the
  nine-Barts instance. Atlas has no instance or event-manager capability
  reachable from a conversation — this is an external blocker, not a
  prerequisite this task can produce. `branchProgress`'s outcome list
  therefore only covers progress `0` (→ `questionIntro`) and progress `2`
  (→ `finalPraise`); the default outcome (`[]`) falls through to `null`
  (dispose), silently covering the omitted `progress == 1` case with no
  placeholder dialogue or "coming soon" text. Also flagged for Task 21's
  research-doc update.
- The quiz has exactly one question/answer pair (`seagullQuestion[0]` /
  `seagullAnswer[0]`), so `seagullIdx` is always `0` and no randomisation
  state was needed — `askAnswer`'s single `matches` entry (`"72"` →
  `correct`) is sufficient.
- The three commented-out lines in the script's final branch (`cm.gainExp`,
  `cm.teachSkill`, `cm.forceCompleteQuest`) are commented out in Cosmic
  itself and are not converted.
- Transcribed strings (including the source misspelling `"intellingence"`
  in `correct` and the `\r\n\r\n` plus leading double-space in `finalHint`)
  were copied verbatim from the brief's Step 1 table, which is itself the
  byte-for-byte transcription of `1092019.js`.
- The ten seed files (`gms/{48,61,72,79,83,84,87,92,95}_1` and `jms/185_1`)
  are byte-identical (verified via `md5sum`); one file was authored and
  copied to the other nine paths.
- Validated with `go run ./tools/catalog-lint deploy/seed` — no errors.

## Quests referenced but not seeded as quest conversations

Tasks 16–20 reference the following quest ids through `questStatus` /
`questProgress` conditions and `set_quest_progress` / `local:get_quest_progress`
operations. None of these conversions requires the referenced quest to exist
as a seeded quest conversation — quest state is read and written entirely
through the condition/operation types listed, independent of whether a
`quest-conversation` seed file exists for that id.

- `3360` (Task 16, Magatia lab door)
- `3339` (Task 17, `gate` state's `questStatus` check)
- `23339` (Task 17, Magatia lab pipes progress quest)
- `3925` (Task 18, Thief gate)
- `20730` (Task 18, Puppeteer gate 1)
- `21728` (Task 18, Thief/Puppeteer password merge, `puppeteerPreCheck`)
- `21731` (Task 18, Puppeteer gate 2)
- `21747` (Task 19, Sealed Shrine entrance)
- `6400` (Task 20, Nautilus seagull quiz)

## Engine fix: genericAction outcome conditions now use AND semantics

Task 19's reviewer found that `processGenericActionState` evaluated only
`outcome.Conditions()[0]` and ignored the rest, with a bare `// TODO` marking
the missing loop
(`services/atlas-npc-conversations/atlas.com/npc/conversation/processor.go`).
The user approved landing this engine fix on this branch. The loop now
evaluates every condition on an outcome in slice order, short-circuiting on
the first `false` (conditions can perform remote calls, so this is a
behavioural requirement), and reports the specific failing condition — not
`Conditions()[0]` — if evaluation errors. Covered by five new tests in
`processor_condition_and_test.go`, including a counting-fake-evaluator test
that fails if the short-circuit is removed.

This tightens every multi-condition outcome already seeded, including
Tasks 18 and 19's two-condition gates (e.g. `askPassword`'s password match
plus `checkOccupancy`'s `mapCapacity` check), which previously enforced only
their first condition and now enforce both.
