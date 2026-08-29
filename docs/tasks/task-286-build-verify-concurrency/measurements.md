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

**Not applied at implementation time.** The `.wslconfig` memory/CPU bump and
the `/etc/fstab` `/tmp` pin are host state outside this repo and outside this
task's scope to apply (`wsl --shutdown` and an `/etc/fstab` edit are operator
actions on the Windows host and the WSL2 VM, not repo changes); doing either
from an implementer session would also invalidate the "before" figures above
for any later comparison. No after-figure is recorded here — recording one
without having actually applied the tuning and rerun `tools/scratch-sweep.sh
--now --root /tmp` would be a fabricated number. Task 7's preflight is the
mechanism that later detects whether an operator has applied this section on
a given host; that detection output, or a fresh `df -h /tmp` / `ls /tmp | wc
-l` pair taken after the operator applies the tuning, is the legitimate
source for an after-figure.

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
