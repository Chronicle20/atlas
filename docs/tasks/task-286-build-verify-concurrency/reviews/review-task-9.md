# Review: Task 9 — narrow `libs/` fan-out to reverse-dependency closure

Commit under review: `2f8847fee` (range `24ddd77fc..2f8847fee`), 6 files, +663/-33.
Briefs: `.superpowers/sdd/plan/task-9-brief.md` (incl. `## CONTROLLER CORRECTIONS`),
`.superpowers/sdd/plan/task-9-brief-cont.md`. Implementer's report:
`.superpowers/sdd/plan/task-9-report.md`.

## Verdict

APPROVED. No blocking findings. All five falsifiable questions the controller
asked me to settle came back clean, independently re-derived against the real
repo graph rather than trusting the report's numbers.

## Provenance / concurrency check

The commit shows no sign of two agents' half-edits: no duplicate function
definitions (`gowork_changed`, `libs_paths`, `libs_changed_module_dirs`,
`changed_modules`, `all_modules` each defined exactly once — `grep -c` on each),
no orphaned references to the old `fanout_paths()` name anywhere under
`tools/` or `docs/` (only pre-existing, unrelated docs mention it), and the
diff reads as a single coherent narrative (comments match the code they sit
above, the stderr-warning text was updated consistently in both the `go.work`
and `libs/` branches and in the `--facts` block). `go.work.sum` was correctly
excluded from the commit (`git show 2f8847fee --stat` lists exactly the 6
files named in the brief).

## Question 1 — is the closure CORRECT, not merely green?

PASS, with a real two-hop chain independently found and confirmed on the live
repo graph (the report's own validation used only wide, mostly single-hop-
flattened libs, see note below).

- `bash tools/lib/module-graph_test.sh`: 10/10 pass, reproduced myself.
- Built the real requires-graph via `tools/lib/module-graph.sh` sourced
  directly (no `go build`, no docker) and computed `module_consumers` for two
  real libs:
  - `libs/atlas-tenant`: raw closure (whole `$ROOT`) = 78 dirs. Diffing
    against `find services libs -name go.mod` (91 total) confirms the only
    non-services/libs member is `tools/catalog-lint` — matches `verify.sh`'s
    documented intersection-with-mods_set behavior, giving 77, matching
    `measurements.md`.
  - `libs/atlas-routine`: raw closure = 84 dirs (incl. self + `tools/catalog-
    lint`). A grep-based spot check first looked like 6 false negatives
    (`libs/atlas-constants`, `atlas-retry`, `atlas-saga`, `atlas-script-core`,
    `atlas-tracing`, `atlas-wz`) but each of those six `go.mod` files has only
    a `replace github.com/.../atlas-routine => ../atlas-routine` line with **no
    corresponding `require`** — confirmed no `.go` source in
    `libs/atlas-retry` imports atlas-routine at all. This is exactly the
    "restrict edges to actual workspace `require` entries, not path-prefix
    guessing" behavior the brief demanded (`_module_graph_requires` only
    parses `require` lines) — the closure is right and my grep was naive.
  - **Genuine 2-hop transitivity, proven on the real graph**: built the full
    direct-require edge table and searched for `A -> B -> C` where `A` does
    not list `C` directly. Found `services/atlas-map-actions` (requires
    `libs/atlas-service`, which requires `libs/atlas-env`) — confirmed by
    `grep -n atlas-env services/atlas-map-actions/atlas.com/map-actions/go.mod`
    (no match) — and `module_consumers "$ROOT" "$ROOT/libs/atlas-env"`
    correctly includes `services/atlas-map-actions/atlas.com/map-actions` in
    its output. This is real transitivity, not the flattened-`go.mod`
    single-hop case that the report's own atlas-tenant/atlas-routine
    measurements happen to reduce to (Go's `go mod tidy` flattens most
    transitive deps into direct `require ... // indirect` lines, so most pairs
    in this repo don't exercise BFS depth > 1 — this one does).
  - `// indirect` counted as a real edge on the real graph too:
    `services/atlas-events/atlas.com/events/go.mod` requires
    `atlas-tenant // indirect`, and it is present in the `atlas-tenant`
    closure.

## Question 2 — `go.work.sum` handling

PASS, both directions confirmed live.

- Read `gowork_changed()` (`tools/verify.sh`): `grep -E '^go\.work$'`, anchored
  at both ends — cannot match `go.work.sum`.
- Live check on the actual worktree (which currently has a dirty
  `go.work.sum` left by an unrelated in-flight background gate):
  `tools/verify.sh --facts --quick --base HEAD` → `fanout_reason=none`,
  `modules_selected=0` — a dirty `go.work.sum` alone does not fan out, exactly
  as claimed.
- The other direction (a real `go.work` edit still fans out to everything) is
  covered structurally by `tools/verify_test.sh`'s "the go.work branch of
  `changed_modules` still calls `all_modules`" assertion (line-range `awk`
  over the function body), which is appropriate since `go.work` is a fixed
  tracked path that can't be probed with an untracked file — matches the
  brief's own explanation for why this case is structural rather than a live
  run.

## Question 3 — are the new `verify_test.sh` assertions load-bearing?

PASS. Ran `bash tools/verify_test.sh` myself, alone, in the background (no
concurrent copies of the suite from me): all 65 assertions pass, exit 0,
including all 8 Task 9 assertions (`a real libs/ change no longer selects
every module`, `the fan-out reason names the closure`, `the escape hatch
restores the old behaviour (module count)`, `the escape hatch's fan-out
reason carries the shared-lib: prefix`, `no libs change, no fan-out`, `a
dirty go.work.sum alone does not fan out`, `...selects zero modules`, `the
go.work branch of changed_modules still calls all_modules`).

Independently reproduced the RED/GREEN check the report describes, without
re-running the (slow) full suite: copied `tools/lib/module-graph.sh` to a
scratch path, changed the final `printf` to emit `${!path_of_dir[@]}` (every
discovered module) instead of the BFS `result_dirs`, sourced the broken copy,
and called `module_consumers "$ROOT" "$ROOT/libs/atlas-tenant"` — got back
111 dirs (every `go.mod` under `$ROOT`, not just services+libs). Since
`verify.sh` intersects with the services+libs set (91), this break would
force `modules_selected=91=total_modules`, which fails
`verify_test.sh`'s `[ "$closure_selected" -lt "$total_modules" ]` assertion —
confirming the assertion is genuinely load-bearing, not vacuously true. This
matches the report's own RED/GREEN numbers (77 → 91).

## Question 4 — stdout/stderr rule

PASS. `awk` over the `changed_modules()` and `libs_changed_module_dirs()`
function bodies: every `echo`/`printf` line is either explicitly `>&2` or is
the function's one designed stdout output (the final `result`/`seen` keys at
`tools/verify.sh:99` inside `changed_modules`, and the module-dir list at the
end of `libs_changed_module_dirs`). No stray diagnostic reaches stdout.

## Question 5 — measurement honesty

PASS, independently re-derived, not merely re-stated.

- Total workspace module count: `find services libs -name go.mod -not -path
  '*/node_modules/*' | wc -l` → `91`, matches `measurements.md`.
- Closure for `libs/atlas-tenant` intersected with services+libs → `77`,
  matches (see Question 1 above for the derivation, including why the raw
  closure is 78 before excluding `tools/catalog-lint`).
- `comm -23` between the full module set and the closure set produces exactly
  14 dropped modules, matching "14 modules dropped" in `measurements.md`, and
  the three named spot-checks (`libs/atlas-opcodes`, `libs/atlas-env`,
  `services/atlas-kafka-precreate`) are all present in that 14-item diff.
  `grep -n atlas-tenant` on those three `go.mod` files returns nothing (exit
  1), confirming none reference the changed lib — matches the report's claim.
- Live re-run of `tools/verify.sh --facts --quick --base HEAD` on the current
  (already-dirty-`go.work.sum`, no other changes) worktree reproduces the
  `fanout_reason=none` / `modules_selected=0` Correction-B measurement
  in-session.

## Non-blocking observations

- `tools/verify.sh`'s per-lib-change stderr summary
  (`"verify.sh: shared-lib change (${first_lib}) fans out to its
  ${#result[@]}-module reverse-dependency closure..."`) is printed before the
  final direct-changed-modules loop runs, so `${#result[@]}` in that one line
  is the closure size only, not the final total if the same run also has
  unrelated directly-changed services in the diff. Cosmetic — it's stderr,
  informational only, and does not affect `modules_selected`/`MODULES` used
  everywhere else. Not blocking.
- `go.work.sum` is currently dirty in the worktree (`M go.work.sum`) and a
  background `bash` process (PID confirmed live via `ps -p`) is still running
  — this predates my review session (present in `git status` before I ran
  anything) and is consistent with the controller's note that a
  `tools/verify.sh --quick --base 24ddd77fc` gate is already running in the
  background against this range. I did not touch it, per the review
  instructions, and it is not part of the reviewed commit (the commit itself
  correctly excludes `go.work.sum` — confirmed via `git show 2f8847fee
  --stat`).
- The report's own Step-4 spot-check used `libs/atlas-tenant` and
  `libs/atlas-routine`, both of which happen to be almost entirely
  single-hop in the current, fully-`go mod tidy`'d repo state (every real
  consumer already lists the lib directly, with `// indirect` where
  applicable) — so the report's own validation does not by itself demonstrate
  BFS depth > 1 on the real graph, only on the synthetic fixture. I consider
  this closed because I found and confirmed a genuine real-repo two-hop case
  myself (`atlas-map-actions` → `atlas-service` → `atlas-env`, Question 1
  above), but a future reader of `measurements.md` alone would not see that
  evidence recorded there. Worth a follow-up note in `measurements.md` if the
  team wants that specific proof preserved, not a blocker for this review.

## Not evaluable

None. Everything in the diff's scope was directly runnable or independently
re-derivable without a docker bake or a full non-`--quick` build.
