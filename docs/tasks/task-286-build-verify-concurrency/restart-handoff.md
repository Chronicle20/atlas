# task-286 — post-WSL-restart handoff (written 2026-08-30, pre-restart)

Context for the session that resumes after `wsl --shutdown`. Everything below
is resumable from repo state plus this file; no conversation history needed.
Delete this file before the PR.

## Where things stand

The branch's implementation and audits were already complete. A critical
review against the design's acceptance criteria found the gaps; this session
closed the biggest one and applied the host tuning. Current criterion status
(see `design.md` "Acceptance criteria" and `measurements.md`):

| # | Criterion | Status |
|---|---|---|
| 1 | context < 1 GiB | met (trivially; disclosed in measurements) |
| 2 | flagless before/after timing, fan-out headline | **DONE this session** — 2846s → 1349s (2.11×), committed `d8ff038`, full method in `measurements.md` "## Criterion 2" |
| 3 | 4-way concurrent `--quick` | still WAIVED BY USER; baseline worktree kept at `/var/tmp/atlas/measure-286-before` if ever wanted |
| 4 | cache under ceiling | partial (out-of-band bake, ~10.2 GB/bake, ceiling never driven to bind) |
| 5 | /tmp unchanged after gate run | **OPEN — this is the resume task, see below** |
| 6 | verify_test.sh | met |
| 7 | flagless exit 0 on branch | met (recorded before this session) |

## Host tuning state at restart

Applied before the restart:

- `C:\Users\<windows-user>\.wslconfig` now carries `[wsl2] memory=52GB,
  processors=24, swap=16GB` (host confirmed 64 GiB). Activates on the restart.
- `~/.zshrc` exports `TMPDIR=/var/tmp/atlas/scratch`; the directory exists.
- systemd user timer `atlas-scratch-sweep.timer` installed and enabled
  (units in `~/.config/systemd/user/`), daily, ExecStart points at the main
  checkout's `tools/scratch-sweep.sh`.

Possibly NOT applied — verify first: the `/etc/fstab` `/tmp` pin
(`tmpfs /tmp tmpfs rw,nosuid,nodev,size=4G,nr_inodes=1048576 0 0`). The
session could not sudo; the user was given the command to run themselves.
Check `cat /etc/fstab` — if the line is absent, ask the user to add it and
restart again before recording after-figures.

## Resume task: close criterion 5 (Layer 0 after-figures)

In the worktree `.worktrees/task-286-build-verify-concurrency/`:

1. Confirm tuning took: `free -h` (expect ~52 Gi total), `nproc` (24),
   `df -h /tmp` (expect 4.0G if the fstab pin is in), `echo $TMPDIR`
   (expect `/var/tmp/atlas/scratch`), `cat /etc/fstab`.
2. Run `tools/scratch-sweep.sh --now --root /tmp` (the brief's command), then
   capture `df -h /tmp` and `ls /tmp | wc -l`.
3. Append the after-figures under `measurements.md` "## Layer 0 — scratch"
   "### After" — replacing the "Not applied at implementation time" paragraph
   with the real figures and the date, keeping the before-figures intact
   (before: 16G tmpfs, 33% used, 2661 entries, 31Gi VM).
4. Commit as `docs(task-286): record Layer 0 after-figures post host tuning`.

## Known follow-up (separate task, not this branch)

Flaky data race in `libs/atlas-kafka`, exposed by the pooled Go layer under
load, pre-existing on main: `TestAddConsumerNoWarnForHealthyDefaults` leaks
the consumer goroutine from `consumer.(*Manager).AddConsumer`
(`consumer/manager.go:202` → `consumer/engine.go:64`, via `atlas-routine.Go`)
past test end; it races with `testing.tRunner` returning. 30× `-race` stress
on main passed — needs machine load to fire. Full race log:
`/var/tmp/atlas/measure-286/after.log` (lines 14–80). Fix shape: make the
test (or manager shutdown) await consumer-goroutine exit before returning.
This is Go code, out of task-286's tools-only scope — file it as its own
task; it will otherwise flake the parallel gate rarely.

## Remaining before PR

- Criterion 5 record (above).
- Decide whether criterion 3 stays waived (13 worktrees now exist under
  `.worktrees/`, so the original "only one worktree" blocker is gone).
- Optionally re-run flagless `tools/verify.sh` on the tuned host.
- Delete this file; then the usual pre-PR review gates
  (`docs/review-protocol.md`).
