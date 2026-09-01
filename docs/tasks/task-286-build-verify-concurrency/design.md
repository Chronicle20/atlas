# task-286 — Build & verify concurrency

**Status:** design (approved in brainstorming, 2026-08-28)
**Branch:** `task-286-build-verify-concurrency`

## Problem

Multiple concurrent Claude Code sessions, each driving `/execute-task`, run
`tools/verify.sh` (go build/vet/test + `docker buildx bake`) against their own
worktree on one machine. Four symptoms were reported:

1. Go build-cache corruption.
2. `/tmp` filling up.
3. Docker image/build cache growing without bound.
4. Flagless `tools/verify.sh` taking 30+ minutes.

## Measured baseline

All figures measured on the development host, 2026-08-28.

| Fact | Value | How measured |
|---|---|---|
| Windows host RAM / logical CPUs | 68,630,040,576 B (~64 GiB) / 24 | `Win32_ComputerSystem.TotalPhysicalMemory`, `Win32_Processor.NumberOfLogicalProcessors` |
| WSL2 VM RAM / CPUs | 31 GiB / 24 | `free -g`, `nproc` |
| `[wsl2]` section in `.wslconfig` | absent (only `[environment]`) | `cat` of the Windows-side `.wslconfig` |
| `/tmp` | tmpfs, 16 GiB, 1.7 GiB used, **2,635 entries** | `df -h /tmp`, `mount`, `ls /tmp \| wc -l` |
| Swap in use | 2.2 GiB of 8.0 GiB | `free -h` |
| Free RAM | 1.1 GiB | `free -h` |
| Disk free on `/` | 831 GiB of 1007 GiB | `df -h /` |
| `GOCACHE` size | 7.8 GiB | `du -sh "$(go env GOCACHE)"` |
| `GOMODCACHE` size | 11 GiB | `du -sh "$(go env GOMODCACHE)"` |
| `GOMAXPROCS`, `GOFLAGS`, `GOTMPDIR` | all unset | `go env` |
| Active worktrees | 13, ~26 GiB total | `ls .worktrees`, `du -sh .worktrees` |
| Go modules in repo | 72 services + 22 libs | `ls services \| wc -l`, `ls libs \| wc -l` |
| Repo-root `tmp/` (gitignored) | 8.0 GiB (WZ assets) | `du -sh ./tmp` |
| `services/atlas-ui` | 1.2 GiB, no `go.mod` | `du -sh`, `find -name go.mod` |
| Docker builder driver | `docker` (default), BuildKit v0.32.2 | `docker buildx ls` |
| Docker build cache | 1.8 GB, 0 B reclaimable | `docker system df` |

### Root causes

**A. The docker build context is enormous and is shipped once per target.**
`Dockerfile` `COPY`s only `libs/` and `services/${SERVICE}/`. `.dockerignore`
excludes only `**/node_modules`, `**/.next`, `**/*.log`, `.git`, `.github`. It
does **not** exclude the gitignored repo-root `tmp/` (8.0 GiB), `.worktrees/`
(26 GiB), `docs/` (148 MiB), `deploy/` (166 MiB), or `services/atlas-ui`
(1.2 GiB, not a Go service and never a bake target). From the main repo the
context is therefore ~36 GiB.

`tools/verify.sh:337` runs `docker buildx bake --set … "$t"` **inside a
`for t in "${TARGETS[@]}"` loop** — one solve per target, each re-transferring
that context. A `libs/` or `go.work` touch selects all 72 go-services, so the
worst case tars and uploads the context 72 times. This is the dominant term in
the 30-minute figure.

**B. `/tmp` is RAM.** WSL2 with no `[wsl2] memory=` uses 50% of host RAM
(31 GiB) and sizes the default `/tmp` tmpfs at 50% of that (16 GiB). The 2,635
stale agent scratch files resident there are RAM subtracted from the compilers,
against 1.1 GiB free and 2.2 GiB already swapped.

**C. Nothing bounds parallelism.** With `GOMAXPROCS` unset, every `go build` and
`go test -race` fans out to 24. N concurrent sessions × 24 on 24 cores, with
`-race` at roughly 5–10× the memory of a normal test binary, is the OOM path.
An OOM-killed process mid-write is the most plausible origin of the reported
build-cache corruption — see "Rejected alternatives" for why cache *sharing* is
not the cause.

**D. `verify.sh` has its parallelism inverted.** The Go layer
(`tools/verify.sh:255`) iterates modules **serially**, one `go build`/`vet`/
`test -race` at a time. The docker layer serializes bake targets that BuildKit
would schedule as a single parallel graph. Each layer does the opposite of what
it should.

**E. The `libs/` fan-out is a ratchet.** `changed_modules()`
(`tools/verify.sh:192`) returns *all* modules whenever any path under `libs/` or
`go.work` appears in the change set. Because `CHANGED` is the whole-branch diff
against the merge base, one `libs/` commit makes every subsequent flagless run
on that branch a full 91-module `-race` build for the life of the branch. The
script already documents the `--base` remedy, but it is opt-in and does not
apply to the mandatory pre-PR flagless run.

## Goals

- Flagless `tools/verify.sh` completes in single-digit minutes on a typical
  change, and materially faster on a `libs/` fan-out.
- Four concurrent sessions can run gates without OOM, cache corruption, or
  mutual interference.
- Docker build cache is bounded by explicit policy, not by manual pruning.
- Agent scratch never competes with compilers for RAM.

## Non-goals

- Changing what `verify.sh` checks. Coverage is unchanged; only scheduling,
  context, and resource governance change.
- Remote/shared build cache across machines.
- Reducing the number of services or worktrees.

## Design

Six layers, independently landable, ordered by payoff per unit of risk.

### Layer 0 — Reclaim machine capacity

**`.wslconfig`** (Windows side, `C:\Users\<windows-user>\.wslconfig`) gains a
`[wsl2]` section alongside the existing `[environment]`:

```ini
[wsl2]
memory=52GB
processors=24
swap=16GB
```

Applied with `wsl --shutdown`. Net +21 GiB of RAM inside the VM, from memory
that is currently reserved for Windows and unused.

**Bound `/tmp` explicitly.** After the memory bump, the default tmpfs sizing
rule would make `/tmp` 26 GiB — worse, not better. Pin it in `/etc/fstab`:

```
tmpfs /tmp tmpfs rw,nosuid,nodev,size=4G,nr_inodes=1048576 0 0
```

**Move scratch off tmpfs.** Introduce a disk-backed scratch root on `/`
(831 GiB free):

- `TMPDIR=/var/tmp/atlas/scratch`, exported from the user shell profile and
  from the repo `.envrc`, so every `go`/`docker`/agent child process inherits it.
- Where the harness permits it, `CLAUDE_JOB_DIR` is pointed under the same root.
  `CLAUDE_JOB_DIR` is set per-session by the Claude Code harness (unset in a
  plain shell), so `TMPDIR` — not `CLAUDE_JOB_DIR` — is the load-bearing
  control here; job-dir relocation is best-effort.
- `tools/scratch-sweep.sh`: deletes entries under the scratch root older than a
  configurable age (default 7 days), plus a `--now` mode. Run from a systemd
  user timer, and invoked opportunistically by `verify.sh`'s preflight.

The existing 2,635 files in `/tmp` are swept once as part of rollout.

### Layer 1 — Starve the build context

Rewrite `.dockerignore` as an allowlist. The `Dockerfile` needs exactly `libs/`
and `services/<name>/`; nothing else in the tree is ever `COPY`ed.

```
# Allowlist: the shared Dockerfile COPYs only libs/ and services/<name>/.
# Everything else in the repo root is host-only and must not enter the context.
*
!libs
!services

# Not a Go service, never a bake target, 1.2 GiB.
services/atlas-ui

# Host-only artifacts inside the allowlisted trees.
**/node_modules
**/.next
**/*.log
```

Expected context: ~36 GiB → ~600 MiB (`services/` minus `atlas-ui` is ~600 MiB;
`libs/` is 16 MiB).

**Verification requirement:** every go-service bake target must still build.
`services/<name>/` is copied wholesale by the Dockerfile, including auxiliary
runtime data (`seed-data/`, `drops/`, `scripts/`, `conversations/`, `shops/`,
`party-quests/`, `configurations/`), so the allowlist must be proven not to drop
any of them — asserted by a full `docker buildx bake all-go-services` under the
new file, not by inspection.

### Layer 2 — Invert `verify.sh`'s parallelism

**Docker layer.** Replace the per-target loop at `tools/verify.sh:337` with a
single invocation:

```sh
docker buildx bake --set "$BAKE_OUTPUT" "${TARGETS[@]}"
```

One context transfer; BuildKit schedules the target graph itself and shares the
`libs/` mod-only and source layers across targets within the solve. Failure
attribution comes from BuildKit's own output, which names the failing target and
step — recorded as a single `FAILED` entry with the solve output printed, rather
than being re-derived by re-running targets individually.

**Go layer.** Replace the serial `for mod in "${MODULES[@]}"` at
`tools/verify.sh:255` with a bounded worker pool (`wait -n` over a job slot
count). Each worker's stdout/stderr is captured to a file under `TMPDIR` and
flushed in module order on completion, so `step()`'s existing pass/fail
reporting and output ordering are preserved. Concurrency defaults to the
per-slot CPU budget from Layer 3.

### Layer 3 — A machine-wide build broker

`tools/with-build-slot.sh <label> -- <command...>`:

- Slot files `1..K` (K=4) under `/var/tmp/atlas/slots/`, a path that is
  machine-global and therefore shared by every worktree and session.
- Try each slot with `flock -n`; if all are held, block on `flock` against a
  deterministically chosen slot. Blocking in `flock` — not polling — satisfies
  the CLAUDE.md prohibition on spending turns waiting on a process.
- `--timeout <sec>` with a non-zero exit and an explicit "no build capacity"
  message, so a wedged holder surfaces rather than hanging a session forever.
- Emits the acquired slot number and wait duration to stderr for diagnosis.

`verify.sh` wraps its two heavy phases — the bake step and the `go test -race`
portion of the Go layer — in a slot. Cheap phases (vet, guards, lint, `--facts`)
run unslotted so a `--quick` inner loop is never queued behind a full gate.

**Per-slot budgets**, derived from 24 threads / 52 GiB and K=4:

| Knob | Value |
|---|---|
| `GOMAXPROCS` | 6 |
| `go build -p` / `go vet -p` | 6 |
| `go test -p` | 2 |
| BuildKit `max-parallelism` | 8 (daemon-global, see Layer 4) |

**`GOMODCACHE` writers get an exclusive lock.** `go mod tidy` and
`go mod download` mutate the shared module cache; `tools/tidy-all-go.sh` runs
them across 91 modules with no coordination. That script — and any `go work
sync` — acquires a single exclusive `flock` on `/var/tmp/atlas/gomodcache.lock`,
distinct from the counting slots. This is the one genuinely unsafe concurrency
in the system.

**Preflight in `verify.sh`.** Before the first heavy phase, hard-fail when free
RAM or free space under `TMPDIR` is below a threshold, with a message naming the
shortfall. A starved run should report "no capacity" rather than proceed and
poison a cache.

### Layer 4 — A governed BuildKit

Replace the default `docker`-driver builder with a `docker-container` builder
pinned to a checked-in config.

`deploy/buildkit/buildkitd.toml`:

```toml
[worker.oci]
  max-parallelism = 8

# Evict aggressively-reclaimable cache first, then enforce a hard ceiling.
[[worker.oci.gcpolicy]]
  keepBytes = 40000000000
  keepDuration = 604800
  filters = ["type==source.local", "type==exec.cachemount", "type==source.git.checkout"]

[[worker.oci.gcpolicy]]
  all = true
  keepBytes = 60000000000
```

`tools/buildx-bootstrap.sh` — idempotent — creates or updates the `atlas`
builder from that config and selects it. `verify.sh` and
`tools/build-services.sh` assert the builder exists and fail with the bootstrap
command if it does not.

Two consequences to handle explicitly:

- The `docker-container` driver does not write to the local image store by
  default. `verify.sh` is unaffected (it already uses
  `*.output=type=cacheonly`), but `tools/build-services.sh` — whose entire
  purpose is producing runnable `<svc>:local` images — must pass `--load`.
- Switching drivers moves BuildKit state into the builder container, so the
  first run after the switch is a cold cache, including the `/go/pkg/mod` and
  `/root/.cache/go-build` cache mounts. This is a one-time cost and must be
  called out in rollout, not discovered.

The Dockerfile's Go cache mounts keep BuildKit's default `sharing=shared`.
Parallel reuse is the point of those mounts, and Go's build cache is designed
for concurrent access; if corruption persists after Layers 0 and 3, revisiting
`sharing=locked` is the fallback, not the opening move.

### Layer 5 — Narrow the `libs/` fan-out

Replace the "any `libs/` path ⇒ all modules" rule in `changed_modules()` with
the transitive reverse-dependency closure over the workspace module graph.

Libs are addressable module paths (`github.com/Chronicle20/atlas/libs/<name>`),
and every consumer names them in its own `go.mod`, so the edge set is derivable
mechanically from the `go.mod` files — no heuristics. The algorithm:

1. Map each changed `libs/<name>/` path to its module path.
2. Build the require graph across all 91 workspace modules.
3. Take the transitive closure of consumers of the changed set.
4. Union with the directly-changed service modules.

`go.work` changes keep the existing full fan-out — a workspace edit can alter
resolution for any module, and there is no cheaper sound answer.

The `--facts` output must name the new reason (which lib, how many consumers)
with the same clarity as the current warning, so a large selection is still
explicable.

## Rejected alternatives

**Per-worktree `GOCACHE`/`GOMODCACHE` isolation.** Considered and rejected. The
Go build cache is content-addressed and explicitly safe for concurrent readers
and writers; sharing it across 13 worktrees is what makes the 2nd through 13th
tree fast. Sharding would turn 7.8 GiB into an estimated ~100 GiB with near-zero
reuse, and would not address the actual failure mode. The corruption evidence
points at memory exhaustion (1.1 GiB free, 2.2 GiB swapped, a 16 GiB RAM-backed
`/tmp`), which Layers 0 and 3 remove. The preflight in Layer 3 is the guard that
converts a future capacity shortfall into a clean failure instead of a corrupt
cache.

**Serializing gates to K=1.** Rejected: it makes the interference problem
disappear by making the machine single-user, and the measured capacity (24
threads, 52 GiB after Layer 0) supports four bounded gates.

**Skipping the bake in the pre-PR gate.** Rejected: `go build` against the
workspace cannot catch a missing `COPY` in the shared Dockerfile, which is
exactly what the bake exists to check. `verify.sh` already documents this.

## Risks

| Risk | Mitigation |
|---|---|
| Allowlist `.dockerignore` silently drops a file some service needs at build time | Full `docker buildx bake all-go-services` must pass under the new file before the change lands |
| Single-bake failure attribution is less precise than per-target | BuildKit output names the failing target and step; the full solve output is printed on failure |
| Parallel Go layer interleaves output and obscures which module failed | Per-module output captured to files and flushed in module order; `step()` semantics preserved |
| Reverse-dependency closure under-selects and misses a real break | Land Layer 5 last; validate the closure against the current all-modules result on a `libs/` branch before switching the default |
| `docker-container` driver breaks image-producing workflows | `tools/build-services.sh` gains `--load`; verified by building and running a service image |
| Broker deadlock wedges every session | `--timeout` with explicit non-zero exit; slot files are `flock`-based and released on process death |
| `.wslconfig` / `/etc/fstab` are host state, not repo state | Documented in `docs/verification.md` with exact contents; `verify.sh` preflight detects and reports the un-tuned condition rather than assuming it |

## Acceptance criteria

Each is a measurement, not an assertion.

1. Build context transferred to BuildKit for a full bake is under 1 GiB
   (measured from the solve's context-transfer step, compared against a
   pre-change baseline capture).
2. Flagless `tools/verify.sh` on a representative single-service change, and on
   a `libs/` fan-out change, are both timed before and after; the fan-out case
   is the headline number.
3. Four concurrent `tools/verify.sh --quick` runs from four different worktrees
   complete successfully with no OOM kill in `dmesg` and no `go` cache error.
4. `docker system df` build cache stays under the configured ceiling across a
   full `all-go-services` bake plus the concurrent-run test.
5. `/tmp` usage after a full gate run is unchanged from before it — scratch
   lands under the disk-backed root instead.
6. `tools/verify_test.sh` passes, extended with cases for the parallel Go pool,
   the single-bake path, the slot broker, and the reverse-dependency closure.
7. The flagless `tools/verify.sh` exits 0 on this branch.

## Rollout order

Layers are independently landable and should land in payoff order, each with
its own measurement recorded:

0. Layer 0 (host tuning + scratch relocation) — no repo behavior change.
1. Layer 1 (`.dockerignore`) — largest single win, smallest diff.
2. Layer 2 (single bake + parallel Go pool).
3. Layer 4 (governed builder) — before Layer 3, because `max-parallelism` is a
   precondition for the broker's budget math to hold.
4. Layer 3 (broker, budgets, preflight, `GOMODCACHE` lock).
5. Layer 5 (reverse-dependency fan-out) — last, validated against the
   all-modules result before becoming the default.
