# Review: Task 1 — Starve the docker build context (Layer 1)

Range reviewed: `1a95779a2..583be745e` (2 commits: `19fc7580c`
"chore(docker): starve build context to an allowlist (task-286 layer 1)",
`583be745e` "docs(task-286): record Layer 1 all-services bake measurements").

Files touched: `.dockerignore`, `docs/verification.md`,
`docs/tasks/task-286-build-verify-concurrency/measurements.md`.

## Requirement-by-requirement

### `.dockerignore` exact content (binding constraint)

PASS. `.dockerignore` (repo root) is byte-for-byte the block specified in
the brief's Step 2: `*` / `!libs` / `!services`, followed by the
`**/node_modules`, `**/.next`, `**/*.log` re-exclusions. No `**/dist` and no
exclusion the original file did not have. Verified with `cat` against the
brief text — comment wording, blank-line placement, and pattern order all
match.

### C1 — `services/atlas-ui` stays reachable

PASS. `docker-bake.hcl:129-132` (construct, not line-anchored to brief) shows
`target "atlas-ui" { context = "." ... }` and it is a member of `all-services`
via `docker-bake.hcl:156` (`targets = concat(go_services, ["atlas-ui",
"atlas-pr-bootstrap", "atlas-kafka-precreate"])`). `.dockerignore` allowlists
`services` whole (`!services`, no narrower `!services/<name>` list), so
`atlas-ui` is not excluded. `services/atlas-ui/Dockerfile` only `COPY`s from
`services/atlas-ui/package.json`, `services/atlas-ui/package-lock.json`, and
`services/atlas-ui/` — consistent with the allowlist. `docs/verification.md`'s
new paragraph states this correctly.

### C3 — `atlas-pr-bootstrap` / `atlas-kafka-precreate` need no change

PASS by construction: `docker-bake.hcl:135-148` sets
`context = "services/atlas-pr-bootstrap"` and
`context = "services/atlas-kafka-precreate"` respectively — the root
`.dockerignore` does not gate their build context at all. The diff correctly
makes no special-case for them.

### `go.work` / shared-lib COPY set stays inside the allowlist

PASS (verified, not assumed). The root `Dockerfile` (`ARG SERVICE`,
`Dockerfile:37-58,64,68-73…`) `COPY`s exclusively from `libs/*/go.mod`,
`services/${SERVICE}/`, and `libs/*` — nothing from repo root outside those
two trees. The repo-root `go.work` is not `COPY`'d; it is synthesized
in-container (`Dockerfile:91-111`, `} > go.work`). So the allowlist does not
starve anything the shared build stage needs.

### `measurements.md` — new file, before/after figure, extensible structure

PASS. `## Layer 1 — build context` is the top-level heading for this task;
Tasks 2/4/5/7/9 append further `##`-level "Layer N — …" headings with no
renumbering required, satisfying the brief's extensibility requirement. The
file records:
- Before: `11.30MB` (`.dockerignore` original).
- After: `11.30MB` (`.dockerignore` allowlist).
- The acceptance threshold ("after-figure under 1 GiB") is technically met
  (11.30MB ≪ 1GiB), but the file is honest that the GiB-scale reduction the
  brief's rationale predicts (`tmp/`, `.worktrees/` starvation) is **not
  demonstrable from a git-worktree checkout**, because neither directory
  exists inside `.worktrees/task-286-build-verify-concurrency/` and because
  this BuildKit transfers context lazily per actual `COPY`, not per
  `.dockerignore`-admitted tree. This matches the controller's ruling given
  in the task prompt, is stated without hedging, and does not fabricate a
  GiB number. Confirmed: `du -sh tmp .worktrees` inside this worktree would
  fail (no such directories) — consistent with the claim (not independently
  re-run here since it is explanatory text, not an acceptance figure).

### Step 3 — `all-services` bake proof (no file dropped)

PASS. Log at `.superpowers/sdd/plan/logs/task1-all-services-bake.log` (not
committed, git-ignored — correctly so) tail shows `exit=0`,
`wall_seconds=1829`, `end=2026-08-28T17:42:01-04:00`. Cross-checked
`17:42:01 − 17:11:32 = 30m29s = 1829s` — the arithmetic in `measurements.md`
is internally consistent with the raw log, not just asserted.

**Non-blocking finding** — `docs/tasks/task-286-build-verify-concurrency/measurements.md:94`:
"Every one of those 215 lines is the literal string `ERROR:`..." is not
quite accurate. `grep -inE 'error|failed'` over the log shows 214 lines
match literal `ERROR:` and one additional line
(`#539 72.00 dist/assets/ErrorDisplay-CsUNnf3R.js ...`, an `atlas-ui`
vite-build asset filename) matches only because of the case-insensitive
`error` substring in "ErrorDisplay" — it is not the string `ERROR:` at all.
The conclusion drawn (exit 0 is the acceptance signal, no target failed) is
still correct and the `ErrorDisplay` line is self-evidently a benign asset
name, not a failure — so this does not change the verdict — but the
sentence overstates what was actually checked ("every one... is the literal
string `ERROR:`" is falsifiable and false for 1 of 215 lines). Should read
something like "214 of 215 are the literal `ERROR:` ...; the remaining line
is an unrelated `ErrorDisplay-*.js` asset filename from the atlas-ui vite
build."

### `docs/verification.md` addition

PASS. New paragraph inserted directly under `## The docker layer`
(`docs/verification.md:112-119` post-edit), states the allowlist mechanism,
correctly cites that every existing root-context Dockerfile only `COPY`s
from `libs/`/`services/`, and gives the forward-looking obligation ("check
this before adding one") for a future root-context Dockerfile or an
out-of-tree `COPY`. Matches the brief's ask verbatim in substance.

### Scope discipline

PASS. `git diff --stat 1a95779a2..583be745e` touches exactly the three files
the brief names — `.dockerignore`, `docs/verification.md`, `measurements.md`
— nothing else. The second commit (`583be745e`) touches only the Step 3
section of `measurements.md` (`git diff 19fc7580c..583be745e`, 16
insertions / 3 deletions, all inside the "Step 3" block); it did not
re-touch the Step 1/2 figures, the caveat section, `.dockerignore`, or
`docs/verification.md`, matching the implementer report's self-description.

### `tools/shell-guard.sh --require-shellcheck`

PASS. Re-ran: `shell-guard: 74 script(s) OK (syntax + shellcheck -S error).`
Exit 0. Matches both the report's claim and the task prompt's stated
context.

## Not evaluable

- The "GiB-scale reduction" the design's acceptance rationale originally
  motivated cannot be measured from any git worktree (by construction, per
  the controller's ruling, which this review independently corroborates via
  the Dockerfile `COPY` analysis above) — `measurements.md` labels this
  correctly as not measured rather than fabricating a number, which is the
  right call given the constraint, but it means acceptance criterion 1's
  spirit (not just its literal "under 1 GiB" text) is unverified from this
  location. This is not a defect in the unit under review — the brief's own
  acceptance text ("under 1 GiB") is trivially satisfied — but a reader of
  `measurements.md` alone would correctly conclude the GiB-scale claim is
  unverified, which is what the file says.
- Did not independently re-run the `atlas-ban` before/after cacheonly bake
  or the full `all-services` bake (per explicit instruction not to). Relied
  on the committed log file and `measurements.md`'s transcription of it,
  cross-checked where arithmetic/grep counts could be verified against the
  log directly.

## Verdict rationale

All binding constraints (exact `.dockerignore` content, C1, C3, extensible
`measurements.md` structure, honest caveat) are met. The one finding
(`measurements.md:94`) is a minor overstatement in a non-acceptance-bearing
explanatory sentence — it does not change any acceptance criterion's
disposition and the underlying conclusion (no real build failures) remains
correct. Non-blocking.
