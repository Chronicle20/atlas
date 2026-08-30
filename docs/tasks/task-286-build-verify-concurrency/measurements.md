# task-286 measurements

Before/after evidence for each layer of the build & verify concurrency work.
Each layer appends its own heading below; do not renumber existing headings.

## Layer 1 — build context

Command (per the brief):

```sh
docker buildx bake --progress=plain --set '*.output=type=cacheonly' atlas-ban 2>&1 | grep -i 'transferring context'
```

### Measured figures

All runs below were taken from this task's worktree
(`.worktrees/task-286-build-verify-concurrency/`), with `docker buildx prune -a -f`
run immediately before each measurement to eliminate BuildKit's local-source
transfer cache as a confound (BuildKit dedupes unchanged context bytes across
builds; without pruning, a "before" run immediately followed by an "after" run
under-reports because most bytes were already known to the daemon).

| Run | `.dockerignore` | `transferring context` (build-context step) |
|---|---|---|
| Before (clean cache) | original (excludes only `node_modules`, `.next`, `*.log`, `.git`, `.github`) | `11.30MB` |
| After (clean cache)  | allowlist (`*` then `!libs`, `!services`)                                    | `11.30MB` |

Raw log excerpts:

```
# before (docker buildx prune -a -f; then bake atlas-ban)
#7 transferring context: 320B done
#10 transferring context: 11.30MB 0.4s done

# after (docker buildx prune -a -f; then bake atlas-ban)
#7 transferring context: 1.14kB done
#11 transferring context: 11.30MB 0.3s done
```

(The `#7`/`.dockerignore` transfer step differs only because the new file is
larger — 320B vs 1.14kB — than the old one; that is the `.dockerignore` file
itself, not the build context.)

### What this does and does not show

**Not measured / no material difference observed for a single-service build
in this worktree.** The brief's acceptance rationale (`transferring context`
drops from several GiB to under 1 GiB) assumes a context that contains the
gitignored `tmp/` (~8 GiB of WZ assets) and `.worktrees/` (~26 GiB of sibling
checkouts) at the repo root. Neither directory exists inside a `git worktree`
checkout — `git worktree add` does not duplicate a parent worktree's
gitignored content, `tmp/` is never populated here, and `.worktrees/` (this
worktree's own path) has no meaning inside itself. `du -sh tmp .worktrees`
inside this worktree returns "No such file or directory" for both.

Independently of that, this BuildKit (`v0.32.2`, `docker` driver) performs a
lazy local-context transfer: it sends only the files the invoked Dockerfile
stages actually `COPY`, not the full directory tree admitted by
`.dockerignore`. For the `atlas-ban` target that COPY set is `libs/*` and
`services/atlas-ban/`, which is why both the old and new `.dockerignore`
report the identical `11.30MB` figure above — the old file already permitted
those same paths through implicitly (it excluded only `node_modules`, `.next`,
`*.log`, `.git`, `.github`, nothing under `libs/` or `services/`).

The allowlist's real effect — reducing the directory walk BuildKit performs to
build its context digest before deciding what to request — is not visible in
the `transferring context` byte count on a machine/worktree where `tmp/` and
`.worktrees/` are absent. It would need to be measured from the main repo
checkout, which is out of scope for work done inside this task's worktree per
this task's constraints (never edit files in the main repo when a task
worktree exists for the work). No number for that scenario is fabricated
here; acceptance criterion 1 (after-figure under 1 GiB) is trivially met by
the measured `11.30MB`, but readers should not conclude a GiB-scale reduction
was demonstrated by this evidence — it was not measurable from this location.

### Step 3 — `all-services` bake (no file lost)

Command:

```sh
docker buildx bake --set '*.output=type=cacheonly' all-services
```

Run from this worktree (`.worktrees/task-286-build-verify-concurrency/`)
with the new allowlist `.dockerignore` in place.

Result: exit `0`. Wall time: `1829` seconds (30m29s) — start
`2026-08-28T17:11:32-04:00`, end `2026-08-28T17:42:01-04:00`. This builds
all 67 Go services plus `atlas-ui`, `atlas-pr-bootstrap` and
`atlas-kafka-precreate`; no target failed, so the allowlist did not drop any
file any bake target needed.

`grep -icE 'error|failed'` over the full build log returns `215` lines.
Every one of those 215 lines is the literal string `ERROR:` appearing
inside the shared Dockerfile's own `go.work`-synthesis `RUN` command text,
as echoed verbatim by BuildKit's step header for that command (not a build
failure) — the process's exit code (`0`) is the acceptance signal, not the
grep count.

Full log retained at `.superpowers/sdd/plan/logs/task1-all-services-bake.log`
(git-ignored scratch; not committed).

## Layer 0 — scratch

Commands (per the brief): `df -h /tmp` and `ls /tmp | wc -l` before, then
after the operator applies the host tuning (`docs/verification.md` ->
`## Host tuning (WSL2)`) and runs `tools/scratch-sweep.sh --now --root /tmp`.

### Before

```
$ df -h /tmp
Filesystem      Size  Used Avail Use% Mounted on
tmpfs            16G  5.1G   11G  33% /tmp

$ ls /tmp | wc -l
2661
```

`free -h` at the same moment:

```
               total        used        free      shared  buff/cache   available
Mem:            31Gi        13Gi       4.6Gi       3.8Gi        17Gi        17Gi
Swap:          8.0Gi       3.3Gi       4.7Gi
```

`/etc/fstab` at the same moment carries no `/tmp` line (unconfigured base
system fstab), confirming `/tmp` is currently sized by WSL2's 50%-of-RAM
default, not the pinned `size=4G` line this task documents. `Mem: 31Gi total`
matches the "VM currently gets 31 GiB by the 50% default" figure the
`## Host tuning (WSL2)` section cites.

### After

Recorded 2026-08-30, after the operator applied the host tuning
(`.wslconfig` `memory=52GB, processors=24, swap=16GB`, the `/etc/fstab`
`/tmp` pin, `TMPDIR=/var/tmp/atlas/scratch` in `~/.zshrc`, the
`atlas-scratch-sweep.timer` user unit) and restarted the VM with
`wsl --shutdown`.

```
$ df -h /tmp
Filesystem      Size  Used Avail Use% Mounted on
tmpfs           4.0G     0  4.0G   0% /tmp

$ ls /tmp | wc -l
2
```

`free -h` and `nproc` at the same moment confirm the VM resize took:

```
               total        used        free      shared  buff/cache   available
Mem:            50Gi       2.6Gi        47Gi       3.0Mi       1.7Gi        48Gi
Swap:           16Gi          0B        16Gi

$ nproc
24
```

`/etc/fstab` now carries the pinned line
`tmpfs /tmp tmpfs rw,nosuid,nodev,size=4G,nr_inodes=1048576 0 0`
(duplicated — the operator added it twice; the duplicate is harmless, `df`
shows a single 4.0G mount, but one copy should be removed).

Two deviations from the brief's literal command, both disclosed rather than
papered over:

- `tools/scratch-sweep.sh --now --root /tmp` exits 2 by design —
  `/tmp` is on the script's dangerous-root refusal list (as the guard tests
  require). The brief's command predates that guard. The sweep was instead
  run against its default root: `tools/scratch-sweep.sh --now` over
  `/var/tmp/atlas/scratch`, leaving the scratch root at 32K.
- The `/tmp` after-figures above therefore reflect the restart (a fresh
  4G tmpfs), not a sweep of `/tmp`; that is the steady state the tuning
  targets — 4G pinned vs the old 16G/33%-used/2661-entry tmpfs.

Operational note: `tools/scratch-sweep.sh --now` deletes *live* session
scratch under `$TMPDIR` (it removed this session's own Claude Code task
output mid-run). `--now` should only be run with no active sessions; the
timer's age-based sweep does not have this hazard.

## Layer 2 — parallelism (Go half)

### What was measured

The brief for task 4 asks for the Go layer's before/after wall time on a
`libs/` fan-out. Running that through the full, flagless `tools/verify.sh`
was out of scope for this session (Contract 2 — module-local checks only; a
full `--all` run also re-executes every other gate, which would swamp the Go
layer's own number in noise). Instead, the narrowest real Go run that
isolates the same work `go_layer`/`launch_go_layers` perform was used
directly: for every one of this repo's **91** Go modules under
`services/` and `libs/` (`find services libs -name go.mod | xargs -n1
dirname | sort -u | wc -l`), `cd "$mod" && go build ./... && go vet ./...` —
exactly what `go_layer` runs under `--quick` (the `go test -race` pass is
excluded from this timing, since it is unchanged by this task and would
dominate the number with per-module test time rather than pool overhead).
This is the scenario a `go.work`/`libs/` change fans out to today, run twice
each way (serial, then pooled, then serial again) on a 24-thread host
(`nproc` = 24) to separate a genuine pool speedup from Go's build-cache
warming between runs.

Before (serial, one module at a time — the loop this task replaces):

```sh
while IFS= read -r mod; do (cd "$mod" && go build ./... && go vet ./...); done < mods.txt
```

After (pooled, `GO_JOBS=4` — `launch_go_layers`'s bounded-parallelism shape):

```sh
for mod in "${MODULES[@]}"; do
    while [ "$(jobs -rp | wc -l)" -ge "$GO_JOBS" ]; do wait -n; done
    ( cd "$mod" && go build ./... && go vet ./... ) &
done
wait
```

### Measured figures

| Run | Wall time |
|---|---|
| Serial, cold-ish cache (first run this session) | 156.4s (2m36s) |
| Pooled, `GO_JOBS=4`, cache warmed by the serial run above | 34.4s |
| Serial again, same warm cache (fair comparison — both sides warm) | 115.2s |
| Pooled again, `GO_JOBS=4`, same warm cache | 35.3s |

The first serial/pooled pair is confounded by Go's build cache — the serial
run pays whatever cold-cache cost exists, and the pooled run that follows it
inherits a warm cache. The second serial/pooled pair controls for that: both
run against the same already-warm cache, and the pool still finishes in
roughly **a third of the serial wall time** (35.3s vs 115.2s) at 4 concurrent
workers on 24 threads — consistent with the module set being dominated by
small, largely I/O- and vet-bound packages rather than CPU-bound compilation,
so the pool's benefit comes from overlapping many small modules' wall time
rather than from raw core count.

### What this does and does not show

Measured: the wall-time effect of bounding module builds to `GO_JOBS`
concurrent workers instead of one at a time, using the exact `cd "$mod" &&
go build ./... && go vet ./...` command `go_layer` runs, over the real,
complete module set a `libs/`/`go.work` fan-out selects today.

Not measured: `go test -race ./...` (excluded, see above — unaffected by
this task, and race-detector runtime would dominate the number); the
`step()`/log-replay overhead `launch_go_layers`/`replay_go_layer` add on top
of the raw build/vet loop (a `mktemp -d`, per-module log/rc files, and a
`cat` per module) — negligible next to a `go build` invocation, but not
separately isolated here; and any figure from the full flagless
`tools/verify.sh`, which was not run for this measurement per Contract 2.

## Layer 3 — concurrency

Task 7 wires the Task 6 build-slot broker and a capacity preflight into
`tools/verify.sh`. Two of the three acceptance measurements below were run
directly; the third (four concurrent `--quick` runs from four different
worktrees) is deferred — see "What this does and does not show".

### Preflight fails closed, and is not permanently on

```sh
env -u ATLAS_MIN_FREE_MB -u ATLAS_MIN_TMP_MB tools/verify.sh --base HEAD --no-ui --no-docker
```

Exit `0`. Immediately after, same command with the override:

```sh
ATLAS_MIN_FREE_MB=99999999 tools/verify.sh --base HEAD --no-ui --no-docker
```

Exit non-zero (`1`), with:

```
verify.sh: insufficient free RAM — 19778 MiB available, 99999999 MiB required.
✗ preflight (capacity) FAILED
...
✗ preflight (capacity)
2 check(s) FAILED — the branch is not ready.
```

This is the pair acceptance criterion 2 asks for: the override makes the
gate fail closed, and the same command without it exits 0 — the preflight
is a real gate, not a permanent block.

### `--quick --base HEAD` exits 0

```sh
tools/verify.sh --quick --base HEAD
```

Exit `0`. Summary ends `All checks passed, but docker bake was skipped — not
a pre-PR pass.` — the preflight step does not even appear in the summary
(skipped, per `[ "$QUICK" -eq 0 ]`), confirming it is off the `--quick` path.

### What this does and does not show

**The four-concurrent-worktree run (acceptance criterion 3) is deferred, not
run.** At the time this task was implemented, `.worktrees/` contained only
this task's own worktree (`.worktrees/task-286-build-verify-concurrency/`) —
`ls -d .worktrees/*/` returns exactly one entry. The acceptance criterion
needs three *additional* worktrees running `--quick` concurrently alongside
this one; creating three worktrees solely to produce this measurement was
explicitly out of scope for this dispatch (per the controller's resolution:
"attempt only if three other worktrees already exist; if they do not, do NOT
create them and do NOT fabricate a result"). No `dmesg`/OOM or `go` cache
error evidence is recorded here because the run that would produce it did
not happen. This measurement belongs to the branch-end pass, once whatever
worktrees exist at that point (or are deliberately created for the purpose)
make a genuine 4-way concurrent run possible.

What *was* exercised concurrency-wise: the build-slot broker's own test
suites (`tools/lib/build-slot_test.sh`, `tools/with-build-slot_test.sh`,
both from Task 6) already cover slot contention, timeout, and release
behavior with real concurrent `flock` holders — see those suites' own
passing output for that evidence. This session did not re-derive it.

## Layer 4 — builder

Task 5 acceptance criterion 4: capture `docker system df` before and after a
real bake and establish whether the 60 GB builder ceiling binds.

### How this was obtained, and how it differs from a gated run

The branch-end flagless `tools/verify.sh` does **not** bake. task-286 changes
only `tools/*.sh`, `tools/lib/*`, and docs — no `go.mod` — so the run reports
`- docker buildx bake (no go.mod touched)` and skips the whole Go layer too.
`docker system df` taken around that run is byte-for-byte identical before and
after, which is recorded here as the positive evidence that no bake ran:

```
# immediately before AND immediately after the passing flagless run
Images          18   15   1.475GB   465.2MB (31%)
Containers      15    1   241.7kB   184.3kB (76%)
Local Volumes   35    1   1.578GB   48.38MB (3%)
Build Cache    256  256   2.644GB   0B
```

The measurement below therefore comes from a bake forced **out of band**, by
explicit user decision, rather than from the gate. It mirrors
`tools/verify.sh:567-569` — same builder, same `*.output=type=cacheonly`, same
`tools/with-build-slot.sh` wrapper — over the full `all-go-services` group:

```
./tools/with-build-slot.sh "bake" -- \
  docker buildx bake --builder atlas --progress=plain \
  --set '*.output=type=cacheonly' all-go-services
```

Two deliberate deviations, disclosed so the number is not read as something it
is not:

1. `--progress=plain` was added so the solve output could be inspected for the
   layer-sharing question below. It changes log verbosity only, not the build.
2. `all-go-services` is the whole group. A gated bake builds only the targets
   whose `go.mod` changed, so this is an upper bound on one run's cost, not a
   typical one.

Result: `rc=0`, wall clock 3m45s (18:45:04 -> 18:48:49). Near-cold for these
targets — 73 `CACHED` vertices out of ~4231 total.

### The df delta, and why the "Build Cache" row is the wrong place to look

```
                    BEFORE                          AFTER
Images          18  15   1.475GB   465.2MB    18  15   1.475GB   465.2MB
Containers      15   1   241.7kB   184.3kB    15   1   249.9kB   184.3kB
Local Volumes   35   1   1.578GB   48.38MB    35   1   11.75GB   48.38MB
Build Cache    256 256   2.644GB   0B        256 256   2.644GB   0B
```

Two rows that did **not** move are the point:

- **Images is flat**, as designed. verify.sh bakes with
  `*.output=type=cacheonly` (`tools/verify.sh:512`), which never writes the
  image store. Any write-up that presented an Images delta as bake output
  would be wrong.
- **`Build Cache` is flat at 2.644GB / 256 entries across a bake that consumed
  ~10 GB.** That row describes the default docker driver's cache. The `atlas`
  builder is a *container-driver* builder, and its cache lives in a Docker
  volume, so `docker system df`'s Build Cache row is blind to it.

The growth landed entirely in **Local Volumes: 1.578GB -> 11.75GB (+10.17GB)**.
Attributed directly:

```
$ docker volume ls --format '{{.Name}}' | grep buildx
buildx_buildkit_atlas0_state
$ docker system df -v   # after
buildx_buildkit_atlas0_state    1    11.72GB
```

Honest limit on this number: only the Local Volumes **total** was captured
before the bake, not that volume individually. Since the post-bake total is
11.75GB with the buildx volume at 11.72GB, the other 34 volumes account for
~0.03GB, which puts the volume's pre-bake size at ~1.55GB. The +10.17GB delta
is measured; the ~1.55GB starting point is inferred from the totals.

### Does the 60 GB ceiling bind?

One near-cold full `all-go-services` bake adds ~10.2 GB of builder state. The
ceiling therefore binds after roughly five to six such bakes without a prune —
it is a real bound, not a theoretical one, but it is not reached by any single
gated run. A gated bake builds only changed targets and so costs a fraction of
this. This is the measured basis for the ceiling; it was not exercised to
exhaustion, and no run in this branch drove the builder to 60 GB.

### Cross-target layer sharing — the claim does NOT hold as written

`docs/verification.md` claimed BuildKit shares the `libs/` mod-only and source
layers across bake targets within one solve. This solve's output does not
support that, so the doc has been softened to what was observed.

Every target got its **own** vertex for byte-identical steps — 67 distinct
vertex ids for one identical `COPY`, one per service target:

```
$ grep -E 'COPY libs/atlas-constants   libs/atlas-constants' bake.log \
    | grep -oE '^#[0-9]+' | sort -u | wc -l
67
#152  [atlas-ban build-env 28/53] COPY libs/atlas-constants   libs/atlas-constants
#1817 [atlas-drops build-env 28/53] COPY libs/atlas-constants   libs/atlas-constants
#1820 [atlas-character build-env 28/53] COPY libs/atlas-constants   libs/atlas-constants
```

Same for the go-build step: 67 distinct vertices. With only 73 `CACHED`
vertices out of ~4231, there was no meaningful cross-target reuse either.

The mechanism is structural in the shared root `Dockerfile`, and it is not
BuildKit failing to dedup:

- `Dockerfile:27-28` declares `ARG SERVICE` and then runs
  `RUN test -n "${SERVICE}" || ...`. That step's digest differs per target and
  it sits **above every `libs/` COPY**, so every downstream layer inherits a
  per-target parent digest. Cross-target sharing is impossible from that line
  onward.
- `Dockerfile:64` copies `services/${SERVICE}/` **before** the full-source
  `libs/` COPYs at `Dockerfile:68-89` — an independent second defeater for the
  source layers specifically.

Follow-up worth its own task, deliberately NOT done here (it is a build
restructuring, out of task-286's scope): hoisting the `SERVICE` validation
below the `libs/` mod-only COPYs, and moving the `services/${SERVICE}/` COPY
below the `libs/` source COPYs, would let the mod-only and libs-source layers
be shared across all 67 targets in one solve. This measurement is the evidence
that the sharing does not happen today; it is not a measurement of what the
reordering would save, which remains unmeasured.

## Layer 5 — fan-out

Task 9 narrows `changed_modules()`'s `libs/` branch from "any `libs/` change
reaches every module" to the transitive reverse-dependency closure over the
workspace `require` graph (`tools/lib/module-graph.sh`), with
`ATLAS_LIBS_FANOUT=all` as a one-variable escape hatch back to the old
behaviour, and Correction B's fix to `go.work.sum` (see below).

### Closure vs. all-modules, on a real `libs/` change

Change set: one untracked file under `libs/atlas-tenant/` (a real `libs/`
consumer with wide reach — 10 other `libs/` modules and multiple services
name it directly in their own `go.mod`).

```sh
ATLAS_LIBS_FANOUT=all tools/verify.sh --facts --quick --base HEAD
tools/verify.sh --facts --quick --base HEAD
```

| Run | `fanout_reason` | `modules_selected` |
|---|---|---|
| `ATLAS_LIBS_FANOUT=all` (old behaviour) | `shared-lib:libs/atlas-tenant/<probe-file>` | `91` |
| closure (default)                       | `shared-lib-closure:libs/atlas-tenant (77 consumers)` | `77` |

Total workspace module count at measurement time (`find services libs -name
go.mod -not -path '*/node_modules/*' \| wc -l`): `91`. The closure's `77` is a
strict subset of the all-modules `91` — 14 modules dropped.

### Spot-check: three dropped modules confirmed as real non-consumers

The 14 dropped modules were computed directly (source
`tools/lib/module-graph.sh`, diff `module_consumers "$ROOT"
"$ROOT/libs/atlas-tenant"` against the full `services`+`libs` module set).
Three were spot-checked by grepping their `go.mod` for the changed lib's
module path (`github.com/Chronicle20/atlas/libs/atlas-tenant`), confirming
each does not name it, directly or as a `replace` target:

```sh
grep -n "atlas-tenant" libs/atlas-opcodes/go.mod libs/atlas-env/go.mod services/atlas-kafka-precreate/go.mod
# (no output — exit 1: none of the three reference atlas-tenant)
```

- `libs/atlas-opcodes` — requires only `atlas-socket`, `atlas-routine`, and
  external deps; no `atlas-tenant`.
- `libs/atlas-env` — requires only `atlas-model` and `atlas-routine`
  (indirect); no `atlas-tenant`.
- `services/atlas-kafka-precreate` — a standalone service with no
  workspace-lib requires at all (only `kafka-go`/`logrus` and their
  transitive deps); no `atlas-tenant`.

### RED/GREEN: the assertions are load-bearing, not vacuously true

Per the controller's instruction, the closure logic was broken in place
(`tools/lib/module-graph.sh`'s `module_consumers`, final `printf` changed to
emit every discovered module directory instead of just the BFS result — i.e.
simulating a bug that always selects everything) and the same `--facts`
invocation re-run, then the file was restored.

| State | `fanout_reason` | `modules_selected` |
|---|---|---|
| GREEN (real closure) | `shared-lib-closure:libs/atlas-tenant (77 consumers)` | `77` |
| RED (closure broken to select everything) | `shared-lib-closure:libs/atlas-tenant (91 consumers)` | `91` |
| GREEN (restored) | `shared-lib-closure:libs/atlas-tenant (77 consumers)` | `77` |

With the closure broken, `modules_selected` (`91`) equals `total_modules`
(`91`) — `tools/verify_test.sh`'s "a real libs/ change no longer selects
every module" assertion (`[ "$closure_selected" -lt "$total_modules" ]`)
would fail under this break, confirming the assertion is load-bearing rather
than passing regardless of the implementation.

### Correction B: a dirty `go.work.sum` no longer forces the full fan-out

Decision recorded (see also the comment at `tools/verify.sh`'s
`gowork_changed()`): `go.work.sum` is **excluded** from the `go.work`
fan-out trigger — `gowork_changed()` matches `^go\.work$`, anchored at both
ends, so it no longer matches `go.work.sum`. Rejected alternative: giving
`go.work.sum` its own branch that still fans out (e.g. to `all_modules` or
to the closure) — rejected because `go.work.sum` is a checksum artifact of
resolving the workspace, not a `require`-graph edit; an ordinary local
`go build`/`go mod tidy` dirties it with zero graph change, so treating it as
a fan-out trigger at all (full or closure) would reintroduce exactly the
"everyday op sends every subsequent run to a full/wide build" trap this task
exists to remove. A real `require`-graph edit that happens to also dirty
`go.work.sum` is still caught on its own merits, via the `go.work` or
`libs/`/`services/` path that actually changed.

Measured on a clean tree (no other changed paths), dirtying only
`go.work.sum` with a synthetic appended line:

```sh
git checkout -- go.work.sum   # start clean
tools/verify.sh --facts --quick --base HEAD   # before
printf '// dirty line\n' >> go.work.sum
tools/verify.sh --facts --quick --base HEAD   # after
git checkout -- go.work.sum   # restore
```

| State | `fanout_reason` | `modules_selected` |
|---|---|---|
| clean `go.work.sum` | `none` | `0` |
| dirty `go.work.sum` (no other change) | `none` | `0` |

Confirms Correction B's fix holds: dirtying `go.work.sum` alone — the
everyday-local-`go build` case — no longer routes to `all_modules`.
`tools/verify_test.sh` covers this same condition under its own per-process
`flock`-protected probe (see "a dirty go.work.sum alone does not fan out" /
"...selects zero modules" in that suite's output).

### What this does and does not show

Measured: the closure vs. all-modules module counts on one real `libs/`
change (`libs/atlas-tenant`, a wide-reach lib), the RED/GREEN load-bearing
check on that same change, and the dirty-`go.work.sum` condition on a clean
tree. Not measured: wall-clock time saved (the brief asks for module-count
evidence, not a timed before/after build); the closure's behavior on a lib
with very few consumers (not needed to establish the win — `atlas-tenant`
was chosen because it is a stress case, not a favorable one); a `go.work`
change (the `go.work` branch is unchanged by Task 9 and is covered
structurally in `tools/verify_test.sh`, not re-measured here).


## Task 7 acceptance 4 — WAIVED BY USER DECISION (not measured, not fabricated)

Task 7's acceptance criterion 4 called for four concurrent `tools/verify.sh --quick`
runs launched from four different worktrees, to demonstrate the machine-wide build-slot
broker serialising real contention.

**This was never run, and is consciously waived — it is NOT recorded as passed.**

Reason: only one worktree for this branch exists on this host. Producing the measurement
would have required creating three additional throwaway worktrees and running four
concurrent builds on a shared machine. Session 2 declined to do that and recorded the
criterion as deferred rather than fabricating a result; the user was asked directly at
branch end and chose to drop it explicitly rather than run it.

What the build-slot broker's correctness therefore rests on instead:
- `tools/lib/build-slot_test.sh` — 7/7, including a deliberate-breakage check by the
  Task 6 reviewer confirming the assertions are load-bearing.
- Static assertions in `tools/verify_test.sh` that the bake layer and the Go pool each
  hold exactly one slot reference, and that no slot is acquired on the guard, lint,
  `--facts`, or summary paths.
- Incidental live concurrency observed during the Task 8b review: a genuinely separate
  top-level `verify_test.sh` instance ran concurrently with the reviewer's own runs and
  both completed exit 0 with a clean tree.

The 4-way saturation case named by the criterion remains **unproven by direct
measurement**. Anyone relying on the broker under heavy multi-worktree contention should
treat that as an open question.

## Criterion 2 — flagless before/after on a `libs/` fan-out (branch-end, 2026-08-30)

The design's headline measurement, run at branch end: the same fan-out change
timed through the old `verify.sh` (main) and the new one (this branch),
back-to-back on the same host.

### Method

- Change set: one untracked probe file under `libs/atlas-tenant/` (the same
  wide-reach stress lib as the Layer 5 measurement), identical on both sides.
- "Before": a detached-`main` worktree at `/var/tmp/atlas/measure-286-before`
  (created for this measurement, outside the repo so it never enters a build
  context). `--facts` confirms the old selection: `shared-lib`, **92 modules**,
  serial, no bake targets.
- "After": this branch's worktree. `--facts` confirms: `shared-lib-closure:
  libs/atlas-tenant (77 consumers)`, **77 modules**, pooled (`GO_JOBS=4`),
  no bake targets.
- Both runs flagless `tools/verify.sh --base HEAD` — full `go build`/`go vet`/
  `go test -race` plus the lint & format guard over the selected modules.
  Neither side bakes (no `go.mod` changed) and neither runs the UI layer, so
  the comparison isolates the fan-out path the design targets.
- Cache control: compile caches (plain and `-race`) warmed once before either
  timed run; `go clean -testcache` immediately before **each** timed run, so
  both pay real `-race` test execution and neither inherits the other's cached
  test results. Without this the second run would trivially win on cached
  `go test` verdicts.
- Host state at measurement time: **untuned** — VM at 31 GiB / 24 threads,
  `/tmp` a 16 GiB tmpfs at 48% use. (The `## Host tuning (WSL2)` section had
  been applied on the Windows side but the WSL restart that activates it was
  deliberately deferred until after this measurement, so both runs share the
  same untuned conditions.)

### Measured figures

| Run | verify.sh | Modules | Wall time | Exit |
|---|---|---|---|---|
| Before | main (`3b03c2905`) | 92, serial | **2846s (47m26s)** | 0 |
| After | this branch | 77, pooled `GO_JOBS=4` | **1349s (22m29s)** | 1 (see below) |

**2.11× faster wall-clock on the fan-out case.** The delta comes from both
levers at once, as designed: 15 fewer modules selected (closure) and 4-way
overlap of the remainder (pool).

### The after-run's exit 1 is a pre-existing flaky data race, not a branch defect

The after run completed all 77 modules; one module failed:
`libs/atlas-kafka` — `TestAddConsumerNoWarnForHealthyDefaults` hit
`WARNING: DATA RACE`: a consumer goroutine spawned via
`consumer.(*Manager).AddConsumer` (manager.go:202 → engine.go:64, via
`atlas-routine.Go`) outlives its test and races with `testing.tRunner`
returning. This branch touches no Go code; the identical code passed in the
serial before-run, and 30 stress iterations of the full consumer suite on the
main checkout (`go test -race -count=30 ./consumer/`, 273s) all passed —
the race needs the pooled run's machine load to fire. It is a latent
goroutine-leak bug in atlas-kafka's tests (or the consumer shutdown path)
that the parallel Go layer is better at exposing than the serial one was.
Filed as follow-up work; the wall-time figure is unaffected (the module ran
to completion, failing only its race check).

### What this does and does not show

Measured: the full flagless wall time on the design's headline scenario,
before vs after, same change, same host, controlled caches. Not measured: a
"typical single-service change" timing (the design's single-digit-minutes
goal for that case; on this branch such a change selects one module and is
dominated by that module's own `-race` time); the tuned-host figures (both
runs predate the WSL restart by design). Note the fan-out case lands at
22m29s — materially faster, as the goal asks, but a `libs/` fan-out with
full `-race` is still not a single-digit-minutes operation on this host.
