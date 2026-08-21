# Review: fix-sparse-overlay-argo-path (25503bcf3..b296200e8)

## Scope

Commit range `25503bcf3..b296200e8` on branch `fix-sparse-overlay-argo-path`:

- `8cd44133b` — `fix(ci): publish the sparse PR overlay at the path Argo actually reads`
  (`.github/workflows/pr-validation.yml`, +47/-8)
- `b296200e8` — `docs(task): record the sparse-overlay Argo path diagnosis`
  (`docs/tasks/fix-sparse-overlay-argo-path/bug-sparse-overlay-argo-path.md`, new file)

Scope matches the brief exactly: a single workflow-step change plus its
diagnosis doc. No code outside `pr-validation.yml`'s `update-pr-overlay` job
was touched.

## Findings

### 1. Step ordering — PASS

Traced the full `update-pr-overlay` job in commit order:

1. `Substitute placeholders in PR overlay` (`.github/workflows/pr-validation.yml:926-1046`) —
   operates on `OVERLAY_DIR` (`pr-sparse` or `pr` per mode), computed fresh
   each run; unaffected by the later move since it runs first.
2. `Stamp CATALOG_REVISION (PR)` (`:1048-1054`) — writes into `deploy/seed/*/*/`,
   unrelated to the overlay directories.
3. `Bump image tags for built services` (`:1056-1100`) — recomputes
   `MAIN_OVERLAY` from `$MODE` (`:1066-1070`), so in sparse mode it correctly
   bumps `deploy/k8s/overlays/pr-sparse/kustomization.yaml`, which the later
   move carries into the published `pr/`. Order is correct — this must run
   before the move, and it does.
4. `Regenerate cluster-infra coordination ConfigMap artifact` (`:1102-1114`) —
   runs `deploy/k8s/overlays/pr/scripts/gen-cleanup-env.sh`. Read the script
   (`deploy/k8s/overlays/pr/scripts/gen-cleanup-env.sh`): it derives
   `ATLAS_DB_NAMES` from `deploy/k8s/overlays/pr/kustomization.yaml`'s static
   `ATLAS_DB_NAMES=` literal and `ATLAS_SERVICES` from
   `.github/config/services.json` — both mode-independent, static content
   that exists in `pr/` regardless of mode and regardless of whether `pr/`
   has been substituted yet. This step must run before the move (`pr/` still
   has `scripts/` at this point in the job); it does, per the diff and the
   inline comment at `:1143-1146`.
5. `Force-push bot/pr-<N>-resolved` (`:1116-1188`) — the move
   (`rm -rf .../pr && mv .../pr-sparse .../pr`, `:1147-1150`) runs here, after
   all four steps above and before the second leak guard and `git add -A`.

No step between substitution and the move reads `pr/` or `pr-sparse/` in a
way that would observe the wrong directory. `MAIN_OVERLAY` in the image-tag
step is recomputed from `$MODE`, not carried from the substitution step's
`$OVERLAY_DIR`, so there is no stale-variable risk either.

### 2. Non-sparse regression from `git add -A` — PASS

In isolated/full mode the move block (`if [ "$MODE" = "sparse" ]`) is
skipped, so `deploy/k8s/overlays/pr-sparse/` is never touched by any step in
the job — substitution used `OVERLAY_DIR=deploy/k8s/overlays/pr` (`:941`),
image-tag bump used `MAIN_OVERLAY=deploy/k8s/overlays/pr/kustomization.yaml`
(`:1069`). Verified with `git status --porcelain deploy/k8s/overlays` and
`.gitignore` (no ignore rules under `deploy/k8s/overlays`, no untracked
cruft) that there is nothing under `pr-sparse/` for `-A` to pick up
unexpectedly in non-sparse mode. `git add -A deploy/k8s/overlays deploy/seed`
is a strict superset of the old `git add <explicit paths>` only in the sense
of also covering deletions/renames — in non-sparse mode it stages exactly
what the old command staged, since `pr-sparse/` has zero working-tree diff.

### 3. `git add -A` correctness in sparse mode — PASS

`rm -rf deploy/k8s/overlays/pr` deletes tracked files without staging (plain
`rm`, not `git rm`); `mv` is a plain filesystem move, not `git mv`. `git add
-A deploy/k8s/overlays` afterward correctly stages: deletions of the old
`pr/*` paths that no longer exist, the moved content now present at `pr/*`
(as adds/modifies), and deletions of `pr-sparse/*` (now missing from the
working tree). This is standard `git add -A` semantics after an out-of-git
directory move and is correct. The `git diff --cached --quiet` early-exit at
`:1172` operates on the correctly-staged index either way — no false
positive/negative introduced.

### 4. Does published `pr/` lose anything Argo needs at sync time? — PASS (with the flagged design note already called out of scope)

`pr-sparse/kustomization.yaml:69` has an explicit inline comment: "wave0-create-dbs.yaml
(overlays/pr) is intentionally NOT included" — confirming the omission is a
deliberate task-232 design decision, not an oversight of this fix. `scripts/`
(containing `gen-cleanup-env.sh`) is a CI/dev-time generator, read only by
the `Regenerate cluster-infra coordination ConfigMap artifact` step (source
checkout, before the move) — never read from the bot branch by Argo at sync
time. Confirmed no other reference to `overlays/pr/scripts` exists anywhere
else in the workflow (`grep -n "overlays/pr/" .github/workflows/pr-validation.yml`
shows only the two pre-move references already covered above). `README.md`
also moves from `pr-sparse/` into published `pr/` (no collision — `pr/` has
no README.md of its own) but is inert prose, not read by kustomize/Argo.

### 5. New leak guard — PASS

`:1157-1164` re-scans `deploy/k8s/overlays/pr deploy/k8s/overlays/pr-cleanup`
*after* the move — i.e., the actual published path — for `*.yaml`/`*.yml`
files containing `PLACEHOLDER_`, and does `exit 1` with an `::error::` on any
match, inside a `set -euo pipefail` step. This is exactly the check that
would have caught the original bug (the first guard at `:1038-1045` only
ever scanned `$OVERLAY_DIR`, i.e. `pr-sparse/` in sparse mode, and never
`pr/`). Scope is correct: same two directories the first guard already
covers, applied post-move.

### 6. Shell correctness — PASS

- `set -euo pipefail` is present at the top of the `Force-push` step
  (`:1122`).
- `LEAKS=$(find ... -exec grep -n 'PLACEHOLDER_' {} + || true)` — GNU `find`
  propagates a non-zero exit from an `-exec ... +` invocation (which `grep`
  produces on "no match"), so the `|| true` is required and present, matching
  the pre-existing guard's style at `:1038-1040`. Correct under `pipefail`
  (this is a command substitution assignment with an explicit `|| true`, not
  a pipeline, so `pipefail` doesn't independently affect it).
- `find ... -exec grep ... {} +` batches invocations correctly; no unquoted
  expansion of variables inside the `find`/`grep` invocation (paths are
  literals).
- `rm -rf deploy/k8s/overlays/pr` is safe under `set -e` even if the path
  doesn't exist (rm -rf on a non-existent path is not an error).
- `mv deploy/k8s/overlays/pr-sparse deploy/k8s/overlays/pr` — if
  `pr-sparse/` were somehow missing this would legitimately fail the job
  under `set -e`, which is the right fail-fast behavior.

## Not evaluable

- Whether Argo's `ApplicationSet(atlas-pr)` really does resolve
  `path: deploy/k8s/overlays/pr` against `bot/pr-<N>-resolved` post-fix
  (i.e., end-to-end confirmation on a live re-run of PR #1411) — that object
  lives in cluster-infra, out of this repo's scope and this review's reach.
  The diagnosis doc states this was confirmed live pre-fix via `kubectl`;
  post-fix live confirmation is not part of this diff and wasn't
  re-verified here.
- The `pr-sparse mirror drift` and `mode select decision table` verify.sh
  guards mentioned in the diagnosis doc as "unchanged/skip" for this diff —
  not exercised as part of this review since the diff doesn't touch the
  files those guards select on.

## Verdict

No blocking findings. The fix is narrowly scoped, the step ordering is
verified correct by tracing every step between substitution and the move,
the `git add -A` change is a safe superset of the prior explicit-path add in
both modes, the new leak guard is correctly scoped and fails the job on a
match, and the wave0/scripts omission from published `pr/` in sparse mode is
confirmed to be a pre-existing, intentional task-232 decision (documented
inline at `pr-sparse/kustomization.yaml:69`) rather than something this fix
introduces or breaks.
