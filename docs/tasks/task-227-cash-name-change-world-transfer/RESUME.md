# task-227 — resume state

**Rewritten end of controller session 5.** Session 4's version said "start Task 33";
Phase G is now done. Everything below is current as of session 5's last commit.

`.superpowers/sdd/plan/progress.md` (git-ignored, but it has survived four
`/clear`s) remains the authority on per-task state. This file is the narrative.

## Where the plan stands

**Done:** plan Tasks 1–30, 32, 33, 34, and 40. Task 41 was folded into Task 26.
**Skipped by user ruling:** Task 31 — see session 4's reasoning, preserved below.
**Remaining:** 37, 38, 39 (Phase H, the purchase path), then 35 (flagless gate)
and 36 (whole-branch review).

### Landed in session 5

| Commit | What |
|---|---|
| `4dfd4df1d` | **Task 33** — atlas-ui pending-changes service + React Query hooks |
| `f184386cb` | **Task 34** — operator pending-changes panel, confirm dialog, page wiring |
| `d636d6598` | Task 34 fix round 1 — thread `characterName` into the cancel dialog |

Both tasks passed `tools/verify.sh --quick` and a read-only review. **Phase G is
complete.**

## Next action

Start **Task 37** (Phase H, the purchase path). This is Go/Kafka backend work —
the `backend-dev-guidelines` skill and `backend-guidelines-reviewer` agent apply
again, not the frontend ones.

Generate the brief with `tools/task-brief.sh docs/tasks/task-227-cash-name-change-world-transfer/plan.md 37`.
**Check it for a `### Files` section and expect it to be wrong or absent** — see
standing rule 1. Session 5 hit this on both its tasks.

## Rulings made in session 5 — do not relitigate

- **The services barrel exports pending-changes TYPES ONLY; import the service and
  hooks directly.** The Task 33 reviewer flagged the types-only export as "a shape
  with no existing precedent," on the premise that every other barrel entry also
  exports an instance. That premise is false, and checking it settled the question:
  the barrel does **not** export `teleportRocksService` — the very reference file
  Task 33 was told to copy — and across `src/`, direct
  `@/services/api/<name>.service` imports outnumber barrel imports **237 to 13**.
  Cost if wrong: one export line plus a one-line import change.
- **The `unwrap` helper is conditional, not mandatory.** Task 33's brief said to
  reproduce `teleport-rocks.service.ts`'s `unwrap` verbatim. With only `getList` and
  a bodyless 204 `delete`, there is nothing to normalize; a dead helper added to
  satisfy prose is a review finding. Cost if wrong: three lines when a write path
  is added.
- **`CancelPendingChangeDialog`'s `Character ${characterId}` fallback is dead code
  at the shipped call site** and was deliberately left in place. `character` is
  narrowed non-null by the early-return guard at `CharacterDetailPage.tsx:109-117`,
  and `Character.attributes.name` (`types/models/character.ts:12`) is a required
  `string`. The `string | undefined` widening on the prop is an
  `exactOptionalPropertyTypes` accommodation for the test helper, not a real runtime
  gap. Do not "fix" either.

## Open items carried forward

- **Commit `4a5d9ff65` (the client cancel path) remains UNREVIEWED**, by earlier user
  ruling, deferred to the branch-end whole-branch review (Task 36). Its implementer
  was stopped before writing a report, so the commit message is the only account of
  it. It claims a cross-character red run — expected 404, actual 204 before the fix.
  **Re-verify that claim** at Task 36; it has never been independently checked.
- **Flagless `tools/verify.sh`** (docker bake + `-race`) is still owed at branch end
  (Task 35). Every gate run on this branch so far has been `--quick`, which prints
  "All checks passed, but docker bake was skipped — not a pre-PR pass."
- **Two deferred minors for Task 36 triage:**
  - Task 33 — `PendingChangeAttributes` / `PendingChange` are written out twice in
    `pending-changes.service.ts:11-33` rather than `Omit<PendingChange, "id">`. A
    dormant drift hazard: a field added to one and not the other still type-checks.
  - Task 34 — `PendingChangesPanel`'s loading state uses plain text where every
    self-fetching sibling in `components/features/characters/` (`SkillWidget`,
    `SkillsSection`, `AttributesPanel`, `MonsterBookWidget`) uses `Skeleton`.
- **A pre-existing defect in atlas-buddies, surfaced but deliberately not fixed:**
  `buddy/entity.go:24` makes `character_id` the sole primary key of the `buddies`
  table, so two owners cannot both hold the same buddy, while the service's own
  `list/processor.go:618` loops over multiple owners as though they could. Out of
  scope (schema migration on a shared table) and already reported to the user.

## Standing rules earned on this branch — do not relearn these

1. **The plan's `Files:` blocks and test snippets are unreliable; verify every one
   against source before dispatching.** Session 5 hit this on both tasks. Task 33's
   block named the wrong test extension (`.ts` for a JSX file), typed three
   `omitempty` fields as nullable rather than optional, and copied `api.getOne` for
   an endpoint that returns an array. Task 34's paths were all correct, but its test
   snippet invented five helpers **and** spied on a service that this repo's
   hook-mocking pattern never reaches — an assertion that could not have passed as
   written. One controller inventory pass costs a fraction of the same discovery
   inside a large implementer.
2. **Never two `atlas-verifier`s at once** — concurrent golangci-lint produces
   `parallel golangci-lint is running` and phantom failures in unrelated modules.
3. **Never run an implementer while a gate is running.** Session 5 made this mistake
   and had to kill a 10-minute gate rather than trust a verdict read off a tree being
   edited underneath it.
4. **Verifier must report EVERY failing module**, not just the first block.
5. **Serialize implementers** — one at a time in this worktree.
6. **Treat briefs and reports as claims, not authority.** Twenty-plus instances on
   this branch where prose contradicted source, several introduced by the controller.
   What catches it: deriving from source and demanding red/green evidence.
7. **Writer registration follows whoever EMITS.**
8. `pendingchange`'s `assetId` param **is an item TEMPLATE id** — passing
   `com.ItemId` is CORRECT. Do not "fix" it.
9. **The gate needs Node 22 sourced or atlas-ui lint false-FAILs.**
   `tools/verify.sh` DOES cover `services/atlas-ui`, but it does not source nvm and
   this machine defaults to Node 24. Any agent or shell running the gate must use:
   `export NVM_DIR="$HOME/.nvm" && . "$NVM_DIR/nvm.sh" && nvm use 22 && tools/verify.sh ...`
   in the SAME call. This cost session 5 a full false-FAIL cycle
   (`lint.sh: ERROR — node v24 found, need v22`, exit 1, with all 75 other checks
   passing). **It will hit Task 35's flagless run identically.**
10. **Run the gate as a bounded background Bash call from the controller, not via
    `atlas-verifier` on haiku.** Session 5's verifier returned four times with
    "still running" and no exit status, then finally the false FAIL above —
    ~1200s of wall clock and four round trips for a verdict the controller got in
    one call. If delegating, pin a model that can hold a single blocking wait.
11. **`pgrep -f <pattern>` self-matches the shell running it.** Session 5 twice
    read a phantom "process still alive" that was its own `pgrep` command line.
    Use a bracket class — `pgrep -f "golangci-lint-v[2]"` — or check `etime`;
    `00:00` means you are looking at yourself.

## Method note worth carrying into Phase H

Task 34's two findings are the shape to expect. Finding 1 was a one-line wiring
miss (the page never passed `characterName`). Finding 2 — found independently by
the reviewer — was that the test *titled* "names the character and the requested
value in the confirm dialog" asserted only the requested value, so it passed with
the prop deleted entirely. **That is why six green tests did not catch a real
wiring bug.**

On a branch with this much invented-test-helper history, the question that finds
real defects is "does this test fail if the behaviour is removed?" Demanding the
implementer actually run that mutation — revert, observe a genuine failure,
restore, confirm green — and then having the re-reviewer independently check the
assertion is not circular (i.e. that it reaches real rendered output rather than
echoing a prop the test itself supplied) is what closed it.
