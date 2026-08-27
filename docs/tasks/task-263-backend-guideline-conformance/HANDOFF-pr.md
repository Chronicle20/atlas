# Handoff — task-263, remaining work: gate the merge, then push and open the PR

State at handoff: branch `task-263-backend-guideline-conformance`, HEAD **`4d724271e`**, tree clean.

## Diagnosis / what just happened

All 27 plan tasks are implemented, audited, and gated. The user chose "push and create a PR". The
push could not proceed directly: `origin/main` had advanced 24 commits past the fork point and the
branch **conflicted**, and per `docs/git-workflow.md` a conflicting push does not trigger the PR
workflows or the `atlas-pr-<N>` rollout. That doc's stated exception applies — merge `origin/main`,
resolve, push the merge commit. That merge is `4d724271e`.

Ten files conflicted (12 hunks). All were the same underlying collision: **this branch relocated and
renamed builders while `main` edited or extended those same builders.** Resolutions, all verified by
`go vet ./...` per module:

| File | Resolution |
|---|---|
| `atlas-channel/channel/account/model.go` | Took HEAD (builder lives in `builder.go` now); applied main's removal of the flat `characterSlots` to `builder.go`, `rest.go` `Transform`, and `rest_test.go` |
| `atlas-login/login/account/model.go` | Same pattern; `characterSlots` dropped from `builder.go` |
| `atlas-monsters/monsters/monster/builder.go` | Took **main's** new aggregating `AddDamageEntry` + `creditModelEntry` semantics, with our `ModelBuilder`→`Builder` rename applied |
| `atlas-channel/channel/skill/handler/recipients.go` (+2 tests) | Took HEAD; carried main's new `SetLevel` into `builder.go`; kept main's `SetLevel(70)` in both tests |
| `atlas-consumables/consumable/processor_test.go` | Kept main's new `computeEffectPlan(..., false)` 4th arg with our `NewBuilder` rename |
| `atlas-monster-death/monster/party/rest.go` | Add/add — **kept both sides** (our `Transform`, main's `ExtractMember`/`MemberRestModel`) |
| `atlas-monster-death/monster/party/rest_test.go` | Add/add — **kept both**: main's full suite plus our `TestTransformRoundTrip` |
| `atlas-monster-death/monster/monster/information/rest_test.go` | Add/add — **kept both**, same shape |

### The non-obvious hazard, and why a full gate is mandatory

Auto-merge silently accepted `main`-added code that calls **pre-rename** builder names. Those
compile-fail but produce no conflict marker, so `git merge` reported success on them. Found and
fixed:

- `atlas-monsters/.../mobskill/builder.go` — main's new `SetCount` had receiver `*ModelBuilder`
- `atlas-channel/.../skill/handler/heal/heal_apply_test.go`, `atlas-consumables/.../processor_test.go`,
  `atlas-monsters/.../disease_callers_test.go`, `disease_targets_test.go`,
  `disease_targets_shell_test.go` — calls to `NewModelBuilder()`
- `atlas-character/.../character/processor_test.go` — mapped to **`NewEmptyBuilder()`**, not
  `NewBuilder()`: this package is the Task 5 exception whose `NewBuilder` takes 8 required args

Sweep now returns **0** call sites (`grep -rn 'NewModelBuilder()' services libs`) and **0**
definitions, so acceptance criterion 1 still holds post-merge. `gofmt -l services libs` is clean.

Only the modules named above were vetted individually. **The other ~85 modules have not been
compiled against merged `main`**, and the same silent-stale-reference class could exist anywhere
main touched a renamed builder. That is exactly what the full gate is for.

## Next steps, in order

1. **Run the flagless gate on the merged tree.** Required — no code-touching claim may be made
   without it, and it has never run on `4d724271e`.

   ```sh
   tools/verify.sh > /tmp/task263-gatelogs/gate-merge.log 2>&1
   ```

   Launch with `run_in_background: true` and reconcile on the completion notification. **Do NOT
   dispatch `task-verifier` for it** — the full bake plus `-race` over 91 modules runs long enough
   that the agent restarts it, and concurrent runs writing one log voided a gate on this branch once
   already. Nothing may edit the working tree while it runs.

   On failure the likely cause is another stale pre-rename reference in an unvetted module; fix it
   the same way (`NewBuilder`, or `NewEmptyBuilder` for `atlas-character/character`).

2. **Push**, then open the PR against `main`:

   ```sh
   git push -u origin task-263-backend-guideline-conformance
   env -u GH_TOKEN -u GITHUB_TOKEN gh pr create --base main --title "..." --body-file ...
   ```

   `gh` must have the token env cleared (`docs/git-workflow.md`) or it 401s.

3. Do **not** re-run the guideline reviewers or `plan-adherence-reviewer`. Both already returned on
   this work (`reviews/audit.md`; `adherence-tasks-*.md`, 0 blocking across all 27 tasks). The merge
   is the only thing they have not seen, and the gate covers it.

## Carry into the PR description

- Scope: backend-guidelines conformance sweep — DOM-01, DOM-04, FILE-05, FR-14/15/17/18 — across
  ~53 services and 3 libs. Exemptions with `file:line` justification are in `exemptions.md`.
- Verification: flagless `tools/verify.sh` green at `ea15a4477`, `f8de7249c`, and `87e11d9fe`
  (91 modules, `-race`, all guards). Plan adherence APPROVED, 0 blocking, 3 shards.
- **Three pre-existing `Extract` defects were found and deliberately NOT fixed** — they predate this
  branch and fixing them would be an out-of-scope behavior change:
  `atlas-channel/character` and `atlas-cashshop/character` drop `spawnPoint`;
  `atlas-npc-shops/character` drops `x`/`y`/`stance`.
- This branch merges `origin/main` (`4d724271e`) to resolve conflicts; see the table above.
