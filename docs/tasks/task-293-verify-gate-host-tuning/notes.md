# task-293 — verify.sh host tuning for a 12-core box under parallel sessions

Trivial-fix tier: no PRD/design/plan. This note is the brainstorming record.

## Diagnosis

`tools/verify.sh --quick` runs after every plan task in every session. On the
dev host (AMD 5900X: 12 physical cores, 24 SMT threads; WSL2 VM ~50 GiB) with
two or three sessions running, the box bogged down. Reading the gate:

- The slot broker's K=4 was sized from `nproc` (24), but Go compilation scales
  with physical cores (12). Four slots × 6 threads was ~2x oversubscribed.
- Inside each slot the Go pool ran 4 workers at `go build -p 6` — 24 threads
  in a slot budgeted for 6.
- `--quick` was exempt from the slot broker entirely. N sessions × 4 workers ×
  6 threads with nothing arbitrating.
- Each changed module was type-checked three or four times per `--quick`:
  `go build`, full-module `go vet`, `golangci-lint run` in workspace mode, and
  the analyzer guards through `unitchecker`. Only the first is cached.
- The gate never ran under `nice`, so a background gate competed equally with
  the interactive sessions above it.
- Nothing measured per-step wall time, so all of the above was inference.
- Two "Known footguns" in `docs/verification.md` (golangci-lint lock
  contention, nvm on PATH) were fixed by #1413 and `tools/lib/node-env.sh`
  but the doc still told readers to work around them.

## Changes

1. **Per-step timing.** `step()` records wall seconds; the Go pool records
   per-module build time and its own wall time; the summary prints
   `✓ label  (Ns)`, the pool total, the worker/slot counts, and the run total.
   `verify_test.sh`'s label normaliser strips the suffix.
2. **Slots derived from physical cores.** `tools/lib/build-slot.sh` defaults
   K to `physical_cores / 6` floored at 1 (via `lscpu`, fallback `nproc/2`).
   12 cores → 2. `ATLAS_BUILD_SLOTS` still overrides. `ATLAS_VERIFY_GO_JOBS`
   default drops from 4 to 2 so one gate is one slot's worth of threads.
3. **`--quick` takes a slot.** The Go pool acquires a slot on every run; only
   the label differs.
4. **Nice.** `verify.sh` renices itself to `ATLAS_VERIFY_NICE` (default 10)
   and sets best-effort/lowest I/O priority; `0` disables.
5. **`--quick` runs formatters only.** `verify.sh` passes `--fmt` to
   `lint.sh` under `--quick`; the golangci linters run on the flagless gate.
6. **Gate per range.** `/execute-task` Step 4c now launches a gate when two
   or more tasks have landed, the task touched `libs/`/`go.work`, a handoff
   or plan end is next, or the implementer reported `DONE_WITH_CONCERNS`.
   Otherwise commits join the next range.
7. **Doc fixes.** Build-slots section rewritten around the derivation; stale
   footguns moved to a "resolved" list.

Not done here (larger change): folding the Atlas analyzers into golangci-lint
as a v2 plugin so vet + lint + guards are one type-check pass.
