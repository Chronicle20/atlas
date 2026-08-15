# task-227 — resume state

**Rewritten end of controller session 4.** The previous version of this file was
written at the end of session 3 and had gone stale — it said "resume at Task 26"
long after Tasks 26, 27 and 28 had landed. Everything below is current as of
session 4's last commit.

`.superpowers/sdd/plan/progress.md` (git-ignored, but it has survived three
`/clear`s) remains the authority on per-task state. This file is the narrative.

## Where the plan stands

**Done:** plan Tasks 1–30, 32, and 40. Task 41 was folded into Task 26.
**Skipped by user ruling:** Task 31 — see below.
**Remaining:** 33, 34 (Phase G, atlas-ui), 37, 38, 39 (Phase H, the purchase path),
then 35 (flagless gate) and 36 (branch review).

### Landed in session 4

| Commit | What |
|---|---|
| `085b52068` | docs — Task 26 + Task 27 review reports (written in session 3, never committed) |
| `41a2d7e98` | **Task 29** — atlas-guilds consumes `NAME_CHANGED` |
| `e05fafd17` | docs — Task 29 review |
| `4586d78b2` | **Task 30** — atlas-buddies consumes `NAME_CHANGED`, emits `BUDDY_UPDATED` |
| `25eb4d3bd` | docs — Task 30 review |
| `8e7ad6c3d` | Task 30 fix round — pin "a redelivered event emits no second `BUDDY_UPDATED`" |
| `6f00b9014` | docs — record Task 31 as skipped |
| `526a3b50b` | **Task 32** — atlas-mts consumes `NAME_CHANGED` |

Every one of Tasks 29, 30 and 32 passed `tools/verify.sh --quick` (exit 0) and a
read-only review with no blocking findings. **Phase F is complete.**

## Next action

Start **Task 33** (atlas-ui service layer + React Query hooks), then Task 34 (the
panel, confirm dialog and page wiring). Phase G is scoped by FR-2.10 to **read +
cancel only** — the operator console must not be able to create a rename or transfer
request, nor edit a requested value.

Generate the brief with `tools/task-brief.sh docs/tasks/task-227-cash-name-change-world-transfer/plan.md 33`.
**Check it for a `### Files` section and expect it to be wrong or absent** — see the
standing rule below. Phase G is the first frontend work on this branch, so the
`frontend-dev-guidelines` skill and the `frontend-guidelines-reviewer` agent apply
rather than the backend ones.

## Rulings made in session 4 — do not relitigate

- **Task 31 (atlas-rankings) is SKIPPED**, by explicit user choice from three
  presented options. `plan.md` Task 31 and `design.md:89` both carry the full
  reasoning. Short version: `ranking/processor.go:123` +
  `ranking/administrator.go:33-41` already restamp `name` on every recompute cycle,
  and the service has no Kafka stack at all (no `kafka/` tree, no consumer manager,
  `atlas-kafka` only a `go.mod` replace line). Accepted residual: the rankings table
  is stale for at most one recompute tick (~1 min) after a rename. If that window
  ever becomes unacceptable, the fix is Task 31 as originally written.
- **Emit-or-not is decided per service, and came out differently in all three.**
  atlas-guilds emits nothing (no suitable event type exists). atlas-buddies emits
  `BUDDY_UPDATED` (the event, its body carrying `CharacterName`, and a live
  registered consumer in atlas-channel all already existed). atlas-mts emits nothing
  (request/response; no push path). Do not generalise from any one of them.
- **atlas-merchant is not a missing Phase F task.** `design.md`'s table lists five
  services but `plan.md:2602` already excludes it by decision (design §3.8, §10):
  `blacklists.name` / `merchant_visits.name` are name-**keyed** rows, and rewriting a
  blacklist entry on rename is a moderation product question.
- **atlas-mts renames every listing row for the seller regardless of `State`.**
  Verified that nothing anywhere treats `seller_name` as a point-in-time record.

## Corrections to earlier sessions' notes

- **The turn-budget hook bug is FIXED** (main-repo commit `c17f8ccad` — the counter
  is now keyed on `agent_id`, not `session_id`). The old instruction in this file to
  hand-zero `/tmp/claude-turn-budget/<session>` before every dispatch is **obsolete;
  do not do it.** That bug is what produced session 3's two spurious Task 26
  PARTIALs.
- **Explicit `tenant_id` predicates are not required.** An earlier session note
  called atlas-guilds' `updateStatus`/`updateTitle` filtering on `character_id`
  alone "a pre-existing gap". That was wrong: `libs/atlas-database` registers an
  automatic tenant scope on any `*gorm.DB` carrying a tenant context (see
  `libs/atlas-database/tenant_scope_test.go`, and `list/provider_test.go:58-70`
  which uses `db.Unscoped()` specifically to bypass it). Task 29's explicit
  predicate is harmless belt-and-braces; Task 30's and Task 32's absence of one is
  correct. Do not "fix" either.

## Open items carried forward

- **Commit `4a5d9ff65` (the client cancel path) remains UNREVIEWED**, by earlier user
  ruling, deferred to the branch-end whole-branch review (Task 36). Its implementer
  was stopped before writing a report, so the commit message is the only account of
  it. It claims a cross-character red run — expected 404, actual 204 before the fix.
  **Re-verify that claim** at Task 36; it has never been independently checked.
- **Flagless `tools/verify.sh`** (docker bake + `-race`) is still owed at branch end
  (Task 35). Only the flagless run counts as verified; every gate run so far on this
  branch has been `--quick`.
- **A pre-existing defect in atlas-buddies, surfaced but deliberately not fixed:**
  `buddy/entity.go:24` makes `character_id` the sole primary key of the `buddies`
  table, so two owners cannot both hold the same buddy. The service's own
  `list/processor.go:618` `UpdateBuddyChannel` loops over multiple owners as though
  they could. Out of scope for task-227 (it is a schema migration on a shared table)
  and reported to the user. Task 30's `updateBuddyName` queries the buddy's own
  `character_id` column, so it stays correct if the defect is ever fixed.

## Standing rules earned on this branch — do not relearn these

1. **The plan's `Files:` blocks are unreliable; verify every one against source
   before dispatching.** Session 4 hit this on all three of its tasks: Task 29's
   block named a file to *create* that already existed, named a `kafka.go` that does
   not exist, and said to modify a `main.go` that needed no change; Tasks 30 and 32
   were prose-only with no `### Files` block at all; and every one of the three test
   snippets invented a `newTestDB` helper the service does not have. One controller
   inventory pass costs a fraction of the same discovery inside a large implementer.
2. **Never two `atlas-verifier`s at once** — concurrent golangci-lint produces
   `parallel golangci-lint is running` and phantom failures in unrelated modules.
3. **Dispatch the verifier with an explicit background+Monitor instruction.** A
   foreground Bash timeout killed one gate at 10 minutes (exit 137) and it reported a
   false ERROR. Per `/execute-task` Step 4c, ERROR is never PASS — re-run it.
4. **Verifier must report EVERY failing module**, not just the first block. A second
   failure has hidden behind the first on this branch.
5. **Serialize implementers** — one at a time in this worktree. Do not run an
   implementer while a gate is running either; a mid-run edit produces phantom
   build failures you then have to disambiguate.
6. **Treat briefs and reports as claims, not authority.** Twenty-two instances on
   this branch where prose contradicted source. Several were introduced by the
   controller, not the implementers — including two in session 4 (a wrong
   `updateBuddyName` signature, caught by the implementer; and the tenant-scoping
   claim corrected above). What catches it: deriving from source and demanding
   red/green evidence.
7. **Writer registration follows whoever EMITS.**
8. `pendingchange`'s `assetId` param **is an item TEMPLATE id** — passing
   `com.ItemId` is CORRECT. Do not "fix" it.
