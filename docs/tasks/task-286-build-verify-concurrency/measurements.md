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
