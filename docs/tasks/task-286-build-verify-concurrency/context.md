# task-286 — context

## Phase inputs

No `prd.md`. `design.md` self-documents as "approved in brainstorming,
2026-08-28", which CLAUDE.md permits in place of a PRD for work that does not
warrant one. The plan was written from `design.md` alone.

## Key files

| Path | Why it matters |
|---|---|
| `tools/verify.sh` (746 lines) | Touched by five of the nine tasks. The serial Go loop is 252-259, the per-target bake loop is 336-338, the fan-out predicate is 188-236, the `--facts` block is 666-727. |
| `tools/verify_test.sh` (199 lines) | Its central claim is that `--facts` IS the selection logic with the work removed. Every `verify.sh` change must keep the agreement loop (115-127) and the structural assertions (43-58) passing. |
| `docker-bake.hcl` | 67 go-service targets from a matrix, plus `atlas-ui` (root context), `atlas-pr-bootstrap` and `atlas-kafka-precreate` (own contexts). Source of the C1 correction. |
| `Dockerfile` | `COPY`s only `libs/*/go.mod`, `services/${SERVICE}/`, `libs/*/`. Synthesizes its own `go.work`, so nothing at the repo root outside those two trees is needed. |
| `.dockerignore` | 9 lines today; excludes `node_modules`, `.next`, `*.log`, `.git`, `.github` — and nothing else, so `tmp/` and `.worktrees/` ship on every solve. |
| `docs/verification.md` (368 lines) | Every task edits it. New sections: `## Host tuning (WSL2)` before `## The Go layer` (line 100), with a `### Build slots` subsection. |
| `tools/lib/analyzer-guard.sh` + `_test.sh` | The precedent for a sourceable `tools/lib/` library with its own suite — the shape Tasks 6 and 9 follow. |
| `tools/doc-slice_test.sh:1-22` | The scaffold every new `_test.sh` in this plan copies. |

## Decisions and deviations from `design.md`

**C1 — `services/atlas-ui` stays in the build context.** The design's Layer 1
claims it is "never a bake target"; `docker-bake.hcl:120-124` defines it as one
with `context = "."`. Excluding it would break `bake atlas-ui`, `all-services`
and the default group. Its 1.2 GiB is `node_modules`, already excluded; the
tracked tree measures 8.7 MiB. The allowlist therefore re-includes `services/`
whole.

**C2 — no tracked `.envrc`.** The main checkout carries an untracked personal
`.envrc` that is not gitignored, so a tracked one would make `git checkout` of
this branch fail there. `TMPDIR` is documented as host state and *detected* by
the Task 7 preflight instead.

**Layer 3 is split three ways** (Tasks 6, 7, 8) rather than landing as one
unit, because the broker, its adoption in `verify.sh`, and the `GOMODCACHE`
lock have independent tests and independent blast radii.

**The broker is a library plus a CLI, not just a CLI.** The design specifies
`tools/with-build-slot.sh <label> -- <command...>`, but Task 7 needs to hold a
slot around `launch_go_layers`, a shell function that cannot be `exec`'d across
a process boundary. `tools/lib/build-slot.sh` holds `acquire_build_slot` /
`release_build_slot`; the CLI is a thin wrapper over it and keeps the design's
contract for external callers.

**Rollout order preserved, with Layer 1 first.** The design's rollout list puts
Layer 0 first as a no-repo-change step; the plan puts Layer 1 (`.dockerignore`)
first because it is the largest single win and the smallest diff, and Layer 0's
repo-side work (the sweeper) is independent of it. Layer 4 still lands before
Layer 3 wiring, and Layer 5 still lands last.

## Deliberately large task

**Task 5 (governed BuildKit builder) touches 6 files** — the F4 sizing warning
threshold. It is not split because the four code files are one atomic change:
creating the `docker-container` builder without simultaneously giving
`tools/build-services.sh` its `--load` flag leaves the repo in a state where
image-producing builds silently stop producing images, which is exactly the
failure mode the design's risk table names. The two doc/config files carry no
implementation risk.

Every other task is at or under 4 files and one service surface.

## Dependencies between tasks

- Task 5 modifies the bake `step` that Task 3 introduces → Task 3 before Task 5.
- Task 7 wraps both the bake step (Tasks 3, 5) and `launch_go_layers` (Task 4),
  and consumes `tools/lib/build-slot.sh` (Task 6) → Tasks 3–6 before Task 7.
- Task 5 before Task 7: `max-parallelism = 8` is a precondition for the
  per-slot budget arithmetic.
- Task 9 is last and requires its own before/after validation against
  `ATLAS_LIBS_FANOUT=all` before the closure becomes the default.
- Tasks 1, 2 and 8 are independent of everything else.

## External blockers / operator actions

Three items in the plan are host state the repo cannot apply: the `.wslconfig`
`[wsl2]` section, the `/etc/fstab` `/tmp` pin, and the `TMPDIR` export plus the
systemd user timer. They are documented in `docs/verification.md` and detected
by the Task 7 preflight. If they are not applied at implementation time, Task 2
Step 4 and Task 7's acceptance require recording that fact rather than a
fabricated after-figure.

## Measurements

All acceptance evidence lands in
`docs/tasks/task-286-build-verify-concurrency/measurements.md`, created in
Task 1 and appended by Tasks 2, 4, 5, 7 and 9 — one heading per layer. Criteria
1–5 of `design.md` are measurements; criterion 6 is `tools/verify_test.sh`;
criterion 7 is a flagless `tools/verify.sh` exiting 0 on the branch.
