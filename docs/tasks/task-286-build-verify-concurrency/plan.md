# task-286 — Build & verify concurrency — implementation plan

Spec: `docs/tasks/task-286-build-verify-concurrency/design.md`.
Context notes: `docs/tasks/task-286-build-verify-concurrency/context.md`.

Nine tasks, ordered by the design's rollout order with one correction (see
"Design corrections" below). Each task is independently landable and carries
its own measurement or test.

Module root: this branch touches no Go module. Every `go build`/`go test` in
this plan is a *verification* of unrelated modules, not a build of changed Go
code. The verification command for every task is
`tools/shell-guard.sh --require-shellcheck` plus the task's own `_test.sh`.

## Design corrections (apply these, not the design's literal text)

**C1 — `services/atlas-ui` IS a bake target.** `design.md` Layer 1 asserts
"not a Go service, never a bake target, 1.2 GiB" and puts `services/atlas-ui`
in the `.dockerignore` allowlist's exclusion block. That is wrong on both
counts:

- `docker-bake.hcl:120-124` defines `target "atlas-ui"` with `context = "."`
  and `dockerfile = "services/atlas-ui/Dockerfile"`. It is a member of the
  `all-services` group (`docker-bake.hcl:146`) and of the `default` group
  (`docker-bake.hcl:152`). `services/atlas-ui/Dockerfile:5,8,13` `COPY`s
  `services/atlas-ui/package.json`, `services/atlas-ui/`, and
  `services/atlas-ui/nginx.conf` from that root context.
- The 1.2 GiB figure is `node_modules`, which the existing `.dockerignore`
  already excludes via `**/node_modules`. Measured in this worktree:
  `du -sh services/atlas-ui` → `8.7M`.

So excluding it would break `docker buildx bake atlas-ui`, `bake
all-services` and the default group, for no space saving. Task 1 keeps
`services/` allowlisted whole.

**C2 — no tracked `.envrc`.** `design.md` Layer 0 says `TMPDIR` is exported
"from the repo `.envrc`". The main repo already carries an untracked personal
`.envrc` (a single `dotenv` line pointing at a file under the user's config
directory), and `.envrc` is not in `.gitignore` — committing a repo `.envrc`
would make `git checkout` of this branch fail there with "untracked working
tree file would be overwritten". `TMPDIR` is therefore documented as host
state in Task 2 alongside `.wslconfig` and `/etc/fstab`, and detected (not
set) by the Task 7 preflight.

**C3 — `atlas-pr-bootstrap` and `atlas-kafka-precreate` are unaffected by the
root `.dockerignore`.** Both set `context` to their own service directory
(`docker-bake.hcl:131,139`), and Docker resolves `.dockerignore` relative to
the context root. No change needed for them.

---

## Task 1: Starve the docker build context (Layer 1)

Rewrite the repo-root `.dockerignore` as an allowlist so the context shipped
to BuildKit contains only `libs/` and `services/`, dropping the gitignored
repo-root `tmp/` (~8 GiB of WZ assets), `.worktrees/` (~26 GiB), `docs/`,
`deploy/`, and everything else at the root.

### Files

- `.dockerignore` — rewrite as an allowlist
- `docs/tasks/task-286-build-verify-concurrency/measurements.md` — **new file**;
  the before/after context-transfer record required by acceptance criterion 1
- `docs/verification.md` — add a short paragraph under `## The docker layer`
  (line 112) stating that the context is an allowlist and what that implies
  when a new root-context Dockerfile is added

Patterns to copy: the existing `.dockerignore` comment style (a "why this
exists" header per block) — see `tools/shell-guard.sh:1-25` for the repo's
prevailing tone in tooling files.

### Steps

- [ ] **Step 1: Capture the baseline, before editing anything.**

Run, and record the `transferring context:` line and the total wall time into
`measurements.md` under a `## Layer 1 — build context` heading:

```sh
docker buildx bake --progress=plain --set '*.output=type=cacheonly' atlas-ban 2>&1 | grep -i 'transferring context'
```

The baseline must be captured with the *current* `.dockerignore` in place. If
this step is run after Step 2 the measurement is worthless; do it first.

- [ ] **Step 2: Rewrite `.dockerignore`.**

Exact content:

```
# Allowlist. Every root-context image build in docker-bake.hcl COPYs only from
# libs/ and services/<name>/ — the shared go-service Dockerfile (COPY
# libs/*/go.mod, COPY services/${SERVICE}/, COPY libs/*/) and
# services/atlas-ui/Dockerfile (COPY services/atlas-ui/). Everything else at
# the repo root is host-only: the gitignored tmp/ holds ~8 GiB of WZ assets and
# .worktrees/ holds ~26 GiB of sibling checkouts, and both were being tarred and
# uploaded on every solve.
#
# services/atlas-ui is deliberately NOT excluded. docker-bake.hcl's "atlas-ui"
# target uses context "." and is a member of the all-services and default
# groups, so excluding it breaks that build. Its bulk is node_modules, already
# excluded below; the tracked tree is ~9 MiB.
#
# Adding a new root-context Dockerfile means checking that everything it COPYs
# lives under libs/ or services/ — or adding another `!` line here.
*
!libs
!services

# Host-only artifacts inside the allowlisted trees. Last match wins, so these
# re-exclude after the `!services` re-inclusion above.
**/node_modules
**/.next
**/*.log
```

Do NOT add `**/dist` or any other new exclusion: the original file did not
have one, and this task's claim is "same files reach the context, less of
everything else."

- [ ] **Step 3: Prove no bake target lost a file.**

The shared Dockerfile `COPY services/${SERVICE}/` wholesale and then copies
auxiliary runtime dirs (the `for src in seed-data drops data scripts
conversations shops party-quests configurations` loop near the end of the
build stage). A full bake is the only proof the allowlist did not drop one:

```sh
docker buildx bake --set '*.output=type=cacheonly' all-services
```

Must exit 0. This builds all 67 go-services plus `atlas-ui`,
`atlas-pr-bootstrap` and `atlas-kafka-precreate`; it is slow and that is
expected — it is the acceptance evidence, not an inner-loop command. Record
the wall time in `measurements.md`.

- [ ] **Step 4: Re-measure and record.**

Re-run the Step 1 command and append the after-figure to `measurements.md`.
Acceptance criterion 1 requires the after-figure to be under 1 GiB.

### Acceptance

- `measurements.md` carries a before and an after `transferring context` figure
  for the same command, and the after-figure is under 1 GiB.
- `docker buildx bake --set '*.output=type=cacheonly' all-services` exits 0.
- `tools/shell-guard.sh --require-shellcheck` exits 0.

---

## Task 2: Disk-backed scratch root and its sweeper (Layer 0)

`/tmp` is a 16 GiB tmpfs — RAM — holding 2,635 stale agent scratch files while
1.1 GiB of RAM is free and 2.2 GiB is swapped. Move scratch to `/` (831 GiB
free) and give it a sweeper. The host-side changes (`.wslconfig`,
`/etc/fstab`, the `TMPDIR` export) are host state, not repo state, so they are
documented here and *detected* by Task 7's preflight rather than applied by
this repo.

### Files

- `tools/scratch-sweep.sh` — **new file**; the sweeper
- `tools/scratch-sweep_test.sh` — **new file**; its suite
- `docs/verification.md` — new `## Host tuning (WSL2)` section, inserted before
  `## The Go layer` (line 100)
- `docs/tasks/task-286-build-verify-concurrency/measurements.md` — created in
  Task 1; append the Layer 0 figures

Patterns to copy: `tools/doc-slice.sh` (a small self-contained tools script
with `--help` produced by a `sed -n` of its own header) and
`tools/doc-slice_test.sh:1-22` (the assert-helper trio and the `mktemp -d` +
`trap` fixture shape).

### Steps

- [ ] **Step 1: Write the failing test.**

`tools/scratch-sweep_test.sh`, structured like `tools/doc-slice_test.sh:1-22`
(shebang, `set -uo pipefail`, `HERE`/target resolution, `assert_eq` /
`assert_has` / `assert_lacks`, `tmp="$(mktemp -d)"` with
`trap 'rm -rf "$tmp"' EXIT`). Every case sets
`ATLAS_SCRATCH_ROOT="$tmp/scratch"` so nothing touches the real root.

Fixture per case: create `$ATLAS_SCRATCH_ROOT`, then populate with `touch -d`
to set mtimes.

| case | fixture | invocation | expect exit | expect after |
|---|---|---|---|---|
| creates a missing root | root does not exist | `scratch-sweep.sh` | 0 | root exists, mode `700` |
| removes an entry older than the default age | `old.txt`, `touch -d '10 days ago'` | `scratch-sweep.sh` | 0 | `old.txt` gone |
| keeps an entry inside the default age | `new.txt`, `touch -d '2 days ago'` | `scratch-sweep.sh` | 0 | `new.txt` present |
| `--age-days 1` removes the 2-day entry | `new.txt` at 2 days | `scratch-sweep.sh --age-days 1` | 0 | `new.txt` gone |
| `--now` removes everything | `new.txt` at 2 days, `dir/` at 2 days | `scratch-sweep.sh --now` | 0 | root empty, root still exists |
| removes a stale directory, not just files | `dir/a`, `touch -d '10 days ago' dir` | `scratch-sweep.sh` | 0 | `dir` gone |
| `--dry-run` removes nothing | `old.txt` at 10 days | `scratch-sweep.sh --dry-run` | 0 | `old.txt` present; stdout contains `old.txt` |
| refuses a dangerous root — `/` | — | `ATLAS_SCRATCH_ROOT=/ scratch-sweep.sh --now` | 2 | stderr contains `refusing` |
| refuses a dangerous root — `/tmp` | — | `ATLAS_SCRATCH_ROOT=/tmp scratch-sweep.sh --now` | 2 | stderr contains `refusing` |
| refuses a dangerous root — `/var/tmp` | — | `ATLAS_SCRATCH_ROOT=/var/tmp scratch-sweep.sh --now` | 2 | stderr contains `refusing` |
| refuses a dangerous root — the home dir | — | `ATLAS_SCRATCH_ROOT="$HOME" scratch-sweep.sh --now` | 2 | stderr contains `refusing` |
| unknown option | — | `scratch-sweep.sh --nope` | 2 | stderr contains `unknown option` |
| `--root` overrides the env var | `$tmp/other` with a 10-day file | `scratch-sweep.sh --root "$tmp/other"` | 0 | file gone |
| summary names the count and the root | two 10-day files | `scratch-sweep.sh` | 0 | stdout matches `removed 2 entr` and contains the root path |
| `-h` prints usage and exits 0 | — | `scratch-sweep.sh -h` | 0 | stdout contains `usage:` |

The dangerous-root cases must be written before the implementation, and must
not be allowed to fall through to a `find -delete`: a wrong guard should be
caught by a failing assertion, not by deleting `/tmp`.

- [ ] **Step 2: Implement `tools/scratch-sweep.sh`.**

Contract:

```
usage: tools/scratch-sweep.sh [--root <dir>] [--age-days <n>] [--now] [--dry-run]

  --root <dir>     scratch root (default: $ATLAS_SCRATCH_ROOT, else
                   /var/tmp/atlas/scratch)
  --age-days <n>   remove entries older than n days (default: 7)
  --now            equivalent to --age-days 0 — remove everything
  --dry-run        print what would be removed; remove nothing
  -h, --help       this message
```

Behaviour:

- `set -euo pipefail`; header comment explains why the root is on disk rather
  than tmpfs (WSL2 sizes `/tmp` at 50% of VM RAM, so scratch there is RAM
  taken from the compilers).
- Resolve the root, `mkdir -p` it with mode `700`.
- Refuse, with exit 2 and a message containing `refusing`, when the resolved
  root is `/`, `/tmp`, `/var/tmp`, the home directory, or has fewer than two
  path components. This guard runs *before* any deletion.
- Sweep top-level entries only (`find "$root" -mindepth 1 -maxdepth 1`), by
  mtime, with `-mtime +$((age-1))` for age ≥ 1 and no mtime predicate for
  `--now`. Deleting a whole stale directory counts as one entry, not N.
- `--dry-run` prints each candidate path, deletes nothing, exits 0.
- On success print `scratch-sweep: removed <n> entr(y|ies) from <root>`.
- Exit 0 when the root is empty, or was absent and has just been created.

- [ ] **Step 3: Document the host state.**

Add `## Host tuning (WSL2)` to `docs/verification.md`, immediately before
`## The Go layer` (currently line 100). It must carry:

- The `.wslconfig` block verbatim (`memory=52GB`, `processors=24`,
  `swap=16GB`), the file's location written as
  `C:\Users\<windows-user>\.wslconfig` — a placeholder, never a literal home
  path, which is enforced under `docs/` — and `wsl --shutdown` as the apply
  step, with the measured justification (host has ~64 GiB and 24 logical CPUs;
  the VM currently gets 31 GiB by the 50% default).
- The `/etc/fstab` line verbatim:
  `tmpfs /tmp tmpfs rw,nosuid,nodev,size=4G,nr_inodes=1048576 0 0`, with the
  reason it is needed *after* the memory bump (the default sizing rule would
  make `/tmp` 26 GiB, worse than today's 16 GiB).
- `export TMPDIR=/var/tmp/atlas/scratch` in the user shell profile, and an
  explicit note that the repo deliberately does **not** ship an `.envrc` for
  this (an untracked personal `.envrc` already exists in the main checkout and
  a tracked one would break `git checkout` there). `CLAUDE_JOB_DIR` relocation
  is best-effort where the harness permits it; `TMPDIR` is the load-bearing
  control.
- The systemd **user** timer for the sweeper: the two unit files to drop in
  the user systemd directory (`atlas-scratch-sweep.service` running
  `<repo-root>/tools/scratch-sweep.sh` — written with that placeholder — and
  `atlas-scratch-sweep.timer` with `OnCalendar=daily`), plus
  `systemctl --user enable --now atlas-scratch-sweep.timer`.
- A one-line pointer that Task 7's preflight reports the un-tuned condition
  rather than assuming this section was applied.

- [ ] **Step 4: One-time rollout sweep.**

Not a repo change; record it in `measurements.md` under
`## Layer 0 — scratch`: `df -h /tmp` and `ls /tmp | wc -l` before, then after
the operator applies the host tuning and runs
`tools/scratch-sweep.sh --now --root /tmp`. If the host tuning has not been
applied at implementation time, record that fact explicitly instead of a
fabricated after-figure.

### Acceptance

- `tools/scratch-sweep_test.sh` exits 0, all assertions passing.
- `tools/shell-guard.sh --require-shellcheck` exits 0.
- `docs/verification.md` contains a `## Host tuning (WSL2)` section with no
  literal home path.

---

## Task 3: One bake solve instead of one per target (Layer 2, docker half)

`tools/verify.sh:336-338` runs `docker buildx bake` inside
`for t in "${TARGETS[@]}"`, re-transferring the whole context per target. A
`libs/` or `go.work` touch selects all 67 go-services, so the worst case
solves 67 times. Replace the loop with a single invocation and let BuildKit
schedule the graph.

### Files

- `tools/verify.sh` — replace the loop at lines 336-338; extend the block
  comment at lines 261-284
- `tools/verify_test.sh` — add the bake-selection assertions
- `docs/verification.md` — update `## The docker layer` (line 112)

### Steps

- [ ] **Step 1: Write the failing test.**

Append to `tools/verify_test.sh`, after the module-selection block at lines
155-161, using the file's own helpers (`assert_eq` / `assert_true` at lines
28-35, `facts_selected` / `facts_key` at lines 86-87).

The behavioural assertion needs a change set that selects more than one bake
target. `bake_targets` (`tools/verify.sh:299-308`) matches a changed path
against each service's `path` from `.github/config/services.json` (e.g.
`services/atlas-ban`) and requires the path to end in `go.mod`; `CHANGED`
includes `git ls-files --others` (`tools/verify.sh:137`). So two untracked
probe files select exactly two targets.

| case | fixture | invocation | expect |
|---|---|---|---|
| two changed go.mods select two bake targets | untracked empty `services/atlas-ban/zz-verify-probe/go.mod` and `services/atlas-account/zz-verify-probe/go.mod` | `--facts --base HEAD` | `bake_targets` is `atlas-account,atlas-ban` |
| …but produce exactly ONE bake gate | same fixture | `--facts --base HEAD` | exactly 1 line of `facts_selected` starts with `docker buildx bake` |
| the gate names the target count | same fixture | `--facts --base HEAD` | that line is `docker buildx bake (2 target(s))` |
| no probes, no bake gate | probes removed | `--facts --base HEAD` | 0 lines start with `docker buildx bake` |
| structural: no per-target bake loop remains | — | read `tools/verify.sh` | `grep -c 'for t in .\{0,4\}TARGETS' tools/verify.sh` is `0` |

Probe files follow the fixed-name convention at `tools/verify_test.sh:99-105`:
name them `probe_bake_ban` and `probe_bake_account`, remove them (and their
`zz-verify-probe/` parent directories) from the same `cleanup()` at line 101,
extending the existing `trap`. Do not use `$$` in the names.

These probe invocations must NOT pass `--quick`: `--quick` sets `NO_DOCKER=1`
(`tools/verify.sh:64`) and skips bake selection entirely. Use
`--facts --base HEAD` alone — `--facts` executes nothing, so no bake runs.

- [ ] **Step 2: Replace the loop.**

`tools/verify.sh:336-338` becomes:

```sh
        step "docker buildx bake (${#TARGETS[@]} target(s))" \
            docker buildx bake --set "$BAKE_OUTPUT" "${TARGETS[@]}"
```

Nothing else in the block changes: `BAKE_OUTPUT` (line 285), the fail-closed
`bake_targets` handling (lines 320-332) and the zero-target skip (line 333)
all stay as they are.

- [ ] **Step 3: Update the comment and the doc.**

Extend the block comment above `BAKE_OUTPUT` (`tools/verify.sh:261-284`) with
the failure-attribution note: one solve means BuildKit's own output names the
failing target and step, so a failure is recorded as a single `FAILED` entry
with the full solve output printed, rather than re-derived by re-running
targets individually.

In `docs/verification.md` `## The docker layer`, change "`docker buildx bake
atlas-<svc>` for every service whose `go.mod` changed" to state that all
selected targets go into one solve, and say why (one context transfer;
BuildKit shares the `libs/` mod-only and source layers across targets within
the solve).

### Acceptance

- `tools/verify_test.sh` exits 0.
- `tools/verify.sh --facts --all` reports `bake_targets=all-go-services` and
  exactly one `gate=docker buildx bake` line.
- A real gate with a bake still passes: `tools/verify.sh --base HEAD --no-ui`
  exits 0.

---

## Task 4: Parallel Go layer with ordered reporting (Layer 2, Go half)

`tools/verify.sh:252-259` iterates modules serially, one
`go build`/`vet`/`test -race` at a time — up to 89 of them on a `libs/`
fan-out. Run them through a bounded worker pool while preserving `step()`'s
pass/fail bookkeeping and per-module output order.

### Files

- `tools/verify.sh` — `go_layer` (lines 241-250) and the driver loop (lines
  252-259); a new `GO_JOBS` default near lines 26-31
- `tools/verify_test.sh` — pool assertions
- `docs/verification.md` — update `## The Go layer` (line 100)

### Steps

- [ ] **Step 1: Write the failing test.**

Append to `tools/verify_test.sh`. The claim is that concurrency changes
nothing observable except wall time, so the assertions compare runs at
different job counts. `--base HEAD` scopes the change set to the working tree,
as every existing probe in that file does (`tools/verify_test.sh:89-98`).

| case | invocation | expect |
|---|---|---|
| gate labels are job-count invariant | `ATLAS_VERIFY_GO_JOBS=1` vs `=4`, both `real_selected --quick --base HEAD` | identical sorted label sets |
| module count is job-count invariant | same two, `facts_key modules_selected --quick --base HEAD` | identical |
| an invalid job count is rejected | `ATLAS_VERIFY_GO_JOBS=0 tools/verify.sh --quick --base HEAD` | exit 2, stderr contains `ATLAS_VERIFY_GO_JOBS` |
| a non-numeric job count is rejected | `ATLAS_VERIFY_GO_JOBS=x tools/verify.sh --quick --base HEAD` | exit 2 |
| structural: the module log dir is created under TMPDIR and cleaned up | read `tools/verify.sh` | contains `mktemp -d "${TMPDIR:-/tmp}/verify-go.XXXXXX"` and an EXIT trap that removes it |

The existing `structural()` block (`tools/verify_test.sh:43-58`) already
asserts exactly one `SELECTED+=` site inside `step()`. The pool must not add a
second one; re-running the suite is the check, no new assertion needed.

- [ ] **Step 2: Add the pool.**

Near the other option defaults (`tools/verify.sh:26-31`):

```sh
# Bounded parallelism for the Go layer. The default is the per-slot CPU budget
# from the build broker (tools/lib/build-slot.sh, K=4 slots on 24 threads).
GO_JOBS="${ATLAS_VERIFY_GO_JOBS:-4}"
case "$GO_JOBS" in
    ''|*[!0-9]*|0) echo "verify.sh: ATLAS_VERIFY_GO_JOBS must be a positive integer (got '$GO_JOBS')" >&2; exit 2 ;;
esac
```

Replace the driver at lines 252-259 with a two-phase shape. Phase A launches
the work; Phase B replays it in module order through the unchanged `step()`,
so `PASSED`/`FAILED`/`SELECTED` bookkeeping and output ordering are exactly
what they are today.

```sh
GO_LOG_DIR=""
launch_go_layers() {
    GO_LOG_DIR="$(mktemp -d "${TMPDIR:-/tmp}/verify-go.XXXXXX")"
    trap 'rm -rf "$GO_LOG_DIR"' EXIT
    local i=0
    for mod in "${MODULES[@]}"; do
        while [ "$(jobs -rp | wc -l)" -ge "$GO_JOBS" ]; do wait -n; done
        (
            # `cmd; rc=$?` would abort the subshell under set -e before the rc
            # file is written, and the replay would then read an empty rc and
            # report a pass. The `if` is load-bearing.
            if go_layer "$mod" >"$GO_LOG_DIR/$i.log" 2>&1; then
                echo 0 >"$GO_LOG_DIR/$i.rc"
            else
                echo $? >"$GO_LOG_DIR/$i.rc"
            fi
        ) &
        i=$((i + 1))
    done
    wait
}

replay_go_layer() {
    cat "$GO_LOG_DIR/$1.log"
    return "$(cat "$GO_LOG_DIR/$1.rc")"
}
```

Driver:

```sh
if [ "${#MODULES[@]}" -eq 0 ]; then
    skip "go build/vet/test (no Go module changed)"
else
    info "verify.sh: ${#MODULES[@]} changed Go module(s), ${GO_JOBS} job(s)"
    # Under --facts step() executes nothing, so launching the work would be
    # pure waste and would break the "--facts runs no check" contract that
    # verify_test.sh:168-177 times.
    if [ "$FACTS" -eq 0 ]; then
        launch_go_layers
    fi
    i=0
    for mod in "${MODULES[@]}"; do
        step "go build/vet$([ "$QUICK" -eq 0 ] && echo '/test -race')  ${mod#"$ROOT"/}" \
            replay_go_layer "$i"
        i=$((i + 1))
    done
fi
```

Two hazards to get right:

- Write the `FACTS` guard as the `if` block above, not as
  `[ "$FACTS" -eq 0 ] && launch_go_layers`. The bare `&&` list is false when
  `FACTS` is 1 and aborts the script under `set -e`; the same trap is already
  documented at `tools/verify.sh:718`.
- `trap ... EXIT` inside `launch_go_layers` replaces any existing EXIT trap.
  `verify.sh` sets none today — confirm with `grep -n 'trap ' tools/verify.sh`
  before relying on that, and if one has appeared, chain rather than replace.

`go_layer` (lines 241-250) is otherwise unchanged by this task; Task 7 adds
its resource budgets.

- [ ] **Step 3: Update the doc.**

In `docs/verification.md` `## The Go layer`, add a paragraph: modules are built
in parallel with `ATLAS_VERIFY_GO_JOBS` workers (default 4); each worker's
output is captured and flushed in module order so a failure still reads as one
labelled block.

### Acceptance

- `tools/verify_test.sh` exits 0, including the existing `--facts` agreement
  and `--facts --all` timing assertions (lines 115-127, 168-177).
- `tools/verify.sh --quick --base HEAD` exits 0.
- A deliberately broken module still reports as a `FAILED` gate naming that
  module: introduce a syntax error in one module, run
  `tools/verify.sh --quick --base HEAD`, confirm the summary names it, revert.
- Record before/after wall time for the Go layer on a `libs/` fan-out in
  `measurements.md` under `## Layer 2 — parallelism` (acceptance criterion 2).

---

## Task 5: A governed BuildKit builder (Layer 4)

The default `docker` driver's build cache is 1.8 GB with 0 B reclaimable and
no policy bounding it, and its parallelism is ungoverned. Move to a
`docker-container` builder pinned to a checked-in config.

Land this **before** Task 7: `max-parallelism` is a precondition for the
broker's budget math.

### Files

- `deploy/buildkit/buildkitd.toml` — **new file**; the pinned config
- `tools/buildx-bootstrap.sh` — **new file**; idempotent create/select/check
- `tools/buildx-bootstrap_test.sh` — **new file**; its suite
- `tools/build-services.sh` — pass `--builder` and `--load`
- `tools/verify.sh` — assert the builder before the bake step added in Task 3
- `docs/verification.md` — `## The docker layer`

Six files, at the sizing limit; see `context.md` for why it is not split.

### Steps

- [ ] **Step 1: Write the config.**

`deploy/buildkit/buildkitd.toml`, verbatim from the design:

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

- [ ] **Step 2: Write the failing test.**

`tools/buildx-bootstrap_test.sh`, same shape as `tools/doc-slice_test.sh:1-22`.
Cases that would touch docker are gated on `command -v docker >/dev/null`.

| case | invocation | expect |
|---|---|---|
| `-h` prints usage | `buildx-bootstrap.sh -h` | exit 0, stdout contains `usage:` |
| unknown option | `buildx-bootstrap.sh --nope` | exit 2, stderr contains `unknown option` |
| `--check` fails with the remedy when the builder is absent | `ATLAS_BUILDER=zz-atlas-absent buildx-bootstrap.sh --check` | non-zero, stderr contains `tools/buildx-bootstrap.sh` |
| the config path the script names resolves | read `tools/buildx-bootstrap.sh` | the path it names is `deploy/buildkit/buildkitd.toml` and that file exists |
| config declares the parallelism budget | read `deploy/buildkit/buildkitd.toml` | contains `max-parallelism = 8` |
| config declares a hard ceiling | read `deploy/buildkit/buildkitd.toml` | contains `all = true` and `keepBytes = 60000000000` |
| `--check` succeeds after a bootstrap (docker only) | `buildx-bootstrap.sh` then `buildx-bootstrap.sh --check` | both exit 0 |
| bootstrap is idempotent (docker only) | run `buildx-bootstrap.sh` twice | both exit 0; `docker buildx ls` lists `atlas` exactly once |

- [ ] **Step 3: Implement `tools/buildx-bootstrap.sh`.**

Contract:

```
usage: tools/buildx-bootstrap.sh [--check] [--force]

  --check   exit 0 if the builder exists; otherwise exit 1 with the command
            that creates it. Makes no changes.
  --force   remove and recreate the builder. Required to pick up a change to
            deploy/buildkit/buildkitd.toml — buildx cannot update the config
            of an existing builder in place.
  -h        this message
```

Behaviour:

- `set -euo pipefail`; `ROOT="$(cd "$(dirname "$0")/.." && pwd)"; cd "$ROOT"`,
  matching `tools/verify.sh:23-24`.
- Builder name `${ATLAS_BUILDER:-atlas}`; config
  `deploy/buildkit/buildkitd.toml`.
- Existence probe: `docker buildx inspect "$name" >/dev/null 2>&1`.
- `--check`: exit 0 if it exists; else print
  `buildx-bootstrap: builder '<name>' does not exist — run tools/buildx-bootstrap.sh`
  to stderr and exit 1.
- `--force`: `docker buildx rm "$name" || true`, then create.
- Create: `docker buildx create --name "$name" --driver docker-container
  --config "$config" --bootstrap --use`.
- Already exists, no `--force`: `docker buildx use "$name"` and report
  `buildx-bootstrap: builder '<name>' already exists (use --force to recreate from <config>)`.
- Header comment must carry the two consequences the design calls out: the
  `docker-container` driver does not write the local image store by default,
  and switching drivers means a one-time cold cache including the
  `/go/pkg/mod` and `/root/.cache/go-build` mounts.

- [ ] **Step 4: Wire the two callers.**

`tools/build-services.sh` — its entire purpose is producing runnable
`<svc>:local` images, which the container driver will not do without `--load`:

```sh
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
./tools/buildx-bootstrap.sh --check
exec docker buildx bake --builder "${ATLAS_BUILDER:-atlas}" --load "$@"
```

Update its header comment to say `--load` is mandatory under the container
driver and why.

`tools/verify.sh` — in the docker block, before the `step` added in Task 3:

```sh
        step "buildx builder" ./tools/buildx-bootstrap.sh --check
        step "docker buildx bake (${#TARGETS[@]} target(s))" \
            docker buildx bake --builder "${ATLAS_BUILDER:-atlas}" --set "$BAKE_OUTPUT" "${TARGETS[@]}"
```

`verify.sh` keeps `*.output=type=cacheonly` and therefore needs no `--load`.
`verify.sh` is not invoked from `.github/workflows/` — the only matches there
are two comments, `pr-validation.yml:102` and `pr-validation.yml:570` — so
this does not change CI.

The new `buildx builder` gate is selected whenever the bake is. Task 3's
assertions count only lines starting with `docker buildx bake`, so they still
hold, but re-run `tools/verify_test.sh` to confirm the `--facts`/real
agreement set (lines 115-127) still matches.

- [ ] **Step 5: Doc and rollout note.**

In `docs/verification.md` `## The docker layer`, add a builder subsection:
what `tools/buildx-bootstrap.sh` does, that `--force` is needed after editing
`deploy/buildkit/buildkitd.toml`, the `--load` requirement for
`tools/build-services.sh`, and — explicitly — that the first build after the
switch is a cold cache and that this is expected, not a regression.

- [ ] **Step 6: Prove an image still runs.**

```sh
tools/build-services.sh atlas-ban
docker image inspect atlas-ban:local
```

Both must exit 0. Record `docker system df` before and after a full
`all-go-services` bake in `measurements.md` under `## Layer 4 — builder`
(acceptance criterion 4).

### Acceptance

- `tools/buildx-bootstrap_test.sh` exits 0.
- `tools/shell-guard.sh --require-shellcheck` exits 0.
- `docker buildx ls` shows an `atlas` builder on the `docker-container` driver.
- `docker image inspect atlas-ban:local` exits 0 after
  `tools/build-services.sh atlas-ban`.
- `docker system df` build cache after a full bake is under the 60 GB ceiling.

---

## Task 6: The build slot broker (Layer 3, broker only)

A machine-wide counting semaphore so N concurrent sessions cannot run N heavy
gates at once. Standalone here; `verify.sh` adopts it in Task 7.

The acquisition logic goes in a sourceable library, because Task 7 needs to
hold a slot around a shell *function* (`launch_go_layers`) that cannot cross a
process boundary. The CLI is a thin wrapper for external callers.

### Files

- `tools/lib/build-slot.sh` — **new file**; `acquire_build_slot` /
  `release_build_slot`
- `tools/lib/build-slot_test.sh` — **new file**; the function-level suite
- `tools/with-build-slot.sh` — **new file**; the CLI wrapper
- `tools/with-build-slot_test.sh` — **new file**; the CLI suite
- `docs/verification.md` — new `### Build slots` subsection under
  `## Host tuning (WSL2)` (added in Task 2)

Patterns to copy: `tools/lib/analyzer-guard.sh` + `tools/lib/analyzer-guard_test.sh`
(the repo's existing sourceable-library-with-suite pair);
`tools/doc-slice_test.sh:1-22` for the suite scaffold. `flock(1)` is present on
this host; the script must still fail with a clear message when it is not.

### Steps

- [ ] **Step 1: Write the failing tests.**

`tools/lib/build-slot_test.sh` — sources the library and drives the functions
directly. Every case sets `ATLAS_SLOT_DIR="$tmp/slots"`.

| case | fixture | assertion |
|---|---|---|
| acquire succeeds and reports a slot | none | `acquire_build_slot t` returns 0 and sets `BUILD_SLOT` to a value in `1..4` |
| the slot dir and files are created | dir absent | after acquire, `$ATLAS_SLOT_DIR` exists with 4 `slot.N` files |
| release frees the slot | after acquire | `release_build_slot` returns 0; a `flock -n` on that slot file from a subshell then succeeds |
| a second acquire past K fails under a timeout | `ATLAS_BUILD_SLOTS=1`, one acquire already held in a background holder | `ATLAS_BUILD_SLOT_TIMEOUT=1 acquire_build_slot t` returns 75 |
| an invalid slot count is rejected | `ATLAS_BUILD_SLOTS=0` | acquire returns 2 |

`tools/with-build-slot_test.sh` — drives the CLI.

| case | fixture | invocation | expect |
|---|---|---|---|
| runs the command and passes stdout through | none | `with-build-slot.sh t -- echo hello` | exit 0, stdout `hello` |
| propagates a non-zero exit status | none | `with-build-slot.sh t -- sh -c 'exit 7'` | exit 7 |
| reports the acquired slot on stderr | none | `with-build-slot.sh t -- true` | stderr matches `acquired slot [1-4]` |
| a free slot is taken without waiting | none | `with-build-slot.sh t -- true` | stderr matches `after 0s` |
| all slots busy + `--timeout` fails cleanly | `ATLAS_BUILD_SLOTS=2`, two background holders each `-- sleep 5` | `ATLAS_BUILD_SLOTS=2 with-build-slot.sh t --timeout 1 -- true` | exit 75, stderr contains `no build capacity` |
| a released slot is reacquired | `ATLAS_BUILD_SLOTS=1`, one holder `-- sleep 1` | `ATLAS_BUILD_SLOTS=1 with-build-slot.sh t --timeout 5 -- true` | exit 0 |
| `--slots` overrides the env var | `ATLAS_BUILD_SLOTS=1`, one holder | `with-build-slot.sh t --slots 4 -- true` | exit 0, no wait |
| missing `--` separator | none | `with-build-slot.sh t true` | exit 2, stderr mentions `--` |
| missing command after `--` | none | `with-build-slot.sh t --` | exit 2 |
| invalid slot count | none | `with-build-slot.sh t --slots 0 -- true` | exit 2 |
| `-h` prints usage | none | `with-build-slot.sh -h` | exit 0, stdout contains `usage:` |

Background holders start as `with-build-slot.sh hold -- sleep 5 &` with PIDs
recorded and `kill`ed in the `trap`; give them `sleep 0.3` to acquire before
the assertion runs. Exit 75 is `EX_TEMPFAIL` from `sysexits.h`, chosen so it
cannot be confused with a command's own failure status.

- [ ] **Step 2: Implement `tools/lib/build-slot.sh`.**

Sourceable — no `set -e` at file scope (`tools/verify.sh` sets its own).
Header comment must state why this blocks in `flock` rather than polling:
CLAUDE.md forbids spending inference turns waiting on a process, and a
blocking `flock` costs no turns.

- Slot dir `${ATLAS_SLOT_DIR:-/var/tmp/atlas/slots}`, `mkdir -p`. Machine-global
  on purpose — every worktree and session shares it.
- Slot count `${ATLAS_BUILD_SLOTS:-4}`; reject non-positive-integer values with
  return 2 and a message naming the variable.
- Slot files `slot.1` … `slot.N`, created with `: >`.
- Return 2 with a clear message if `flock` is not on `PATH`.
- `acquire_build_slot <label>`: try each slot `1..N` with `flock -n` on a
  numbered fd. If all are held, pick a deterministic slot
  (`$(( $$ % N + 1 ))`) and block on it — with `flock -w
  "$ATLAS_BUILD_SLOT_TIMEOUT"` when that variable is set, else a plain
  blocking `flock`. On timeout print
  `build-slot: no build capacity for '<label>' after <sec>s` to stderr and
  return 75. On success set `BUILD_SLOT` and print
  `build-slot: '<label>' acquired slot <n> after <d>s` to stderr, `<d>`
  measured with `SECONDS`.
- `release_build_slot`: `flock -u` the held fd and close it.
- The lock lives on a file descriptor, so the kernel releases it when the
  process dies. Do not write a stale-lock cleanup path; there is nothing to
  clean.

Use a fixed numbered fd rather than `exec {fd}>` if `shellcheck -S error`
objects — `tools/shell-guard.sh` runs at `error` severity
(`tools/shell-guard.sh:14-18`). Pick fds outside 0-2 and outside fd 9, which
Task 8 uses for the module-cache lock.

- [ ] **Step 3: Implement `tools/with-build-slot.sh`.**

```
usage: tools/with-build-slot.sh [--slots N] [--timeout SEC] <label> -- <command...>

  --slots N       number of machine-wide slots (default: $ATLAS_BUILD_SLOTS,
                  else 4)
  --timeout SEC   give up after SEC seconds waiting for a slot and exit 75
                  (EX_TEMPFAIL) instead of blocking forever
  <label>         what this slot is for; appears in the stderr diagnostics
  -h, --help      this message
```

`set -euo pipefail`; parse args, `source tools/lib/build-slot.sh`, map
`--slots`/`--timeout` onto `ATLAS_BUILD_SLOTS`/`ATLAS_BUILD_SLOT_TIMEOUT`,
`acquire_build_slot "$label"` (propagating 75), then run `"$@"` and exit with
its status while the lock is still held.

- [ ] **Step 4: Document it.**

`### Build slots` under `## Host tuning (WSL2)`: what the broker is, the K=4
default and the arithmetic behind it (24 threads, 52 GiB after the memory
bump), the per-slot budget table from the design (`GOMAXPROCS` 6,
`go build -p` 6, `go test -p` 2, BuildKit `max-parallelism` 8), and the
exit-75 contract.

### Acceptance

- `tools/lib/build-slot_test.sh` and `tools/with-build-slot_test.sh` both exit 0.
- `tools/shell-guard.sh --require-shellcheck` exits 0.

---

## Task 7: verify.sh adopts slots, budgets and a preflight (Layer 3, wiring)

### Files

- `tools/verify.sh` — slot wrapping, resource budgets, preflight
- `tools/verify_test.sh` — preflight and budget assertions
- `docs/verification.md` — `## The Go layer`, `## The docker layer`, and the
  `### Build slots` subsection from Task 6

### Steps

- [ ] **Step 1: Write the failing test.**

| case | invocation | expect |
|---|---|---|
| preflight is selected on a full run | `--facts --base HEAD --no-ui` | `facts_selected` contains `preflight (capacity)` |
| preflight is NOT selected under `--quick` | `--facts --quick --base HEAD` | that label absent |
| `--facts`/real agreement still holds | the existing loop at `tools/verify_test.sh:115-127` | unchanged, passing |
| a starved run fails rather than proceeding | `ATLAS_MIN_FREE_MB=99999999 tools/verify.sh --base HEAD --no-ui --no-docker` | non-zero exit; summary contains `✗` and the text `preflight` |
| the preflight message names the shortfall | same | output matches `free RAM` and a MiB figure |
| an un-tuned host is reported, not assumed | `TMPDIR=/tmp ATLAS_MIN_FREE_MB=1 ATLAS_MIN_TMP_MB=1 tools/verify.sh --base HEAD --no-docker --no-ui` | output contains `Host tuning` and `docs/verification.md` |
| budgets are applied in the Go layer | read `tools/verify.sh` | `go_layer` exports `GOMAXPROCS` and passes a `-p` to `go build` and a distinct `-p` to `go test` |
| the heavy phases are slotted | read `tools/verify.sh` | `acquire_build_slot` appears exactly twice — once for the bake, once for the Go pool |
| cheap phases are not slotted | read `tools/verify.sh` | no slot acquisition on the guard, lint, or `--facts` paths |

The starved-run case is the important one: it must prove the gate *fails*
rather than proceeding, which is the whole point of the preflight.

- [ ] **Step 2: Add the preflight.**

```sh
MIN_FREE_MB="${ATLAS_MIN_FREE_MB:-4096}"
MIN_TMP_MB="${ATLAS_MIN_TMP_MB:-8192}"

preflight() {
    local free_mb tmp_mb tmpdir="${TMPDIR:-/tmp}" rc=0
    free_mb="$(free -m | awk '/^Mem:/ {print $NF}')"
    tmp_mb="$(df -Pm "$tmpdir" | awk 'NR==2 {print $4}')"

    # Not a failure — a report. The host tuning in docs/verification.md is host
    # state, not repo state, so the gate detects the un-tuned condition rather
    # than assuming it was applied.
    case "$tmpdir" in
        /tmp|/tmp/*)
            echo "verify.sh: TMPDIR is ${tmpdir} — on this WSL2 host /tmp is tmpfs (RAM)." >&2
            echo "           See docs/verification.md, 'Host tuning (WSL2)'." >&2
            ;;
    esac

    if [ "$free_mb" -lt "$MIN_FREE_MB" ]; then
        echo "verify.sh: insufficient free RAM — ${free_mb} MiB available, ${MIN_FREE_MB} MiB required." >&2
        rc=1
    fi
    if [ "$tmp_mb" -lt "$MIN_TMP_MB" ]; then
        echo "verify.sh: insufficient free space under TMPDIR (${tmpdir}) — ${tmp_mb} MiB free, ${MIN_TMP_MB} MiB required." >&2
        rc=1
    fi
    return "$rc"
}
```

Select it with `step "preflight (capacity)" preflight` immediately before the
Go-module block (`tools/verify.sh:252`), gated on `[ "$QUICK" -eq 0 ]` so a
`--quick` inner loop is never blocked by it.

Before committing the `awk` field index, run `free -m` and confirm that the
"available" column is the last field on the `Mem:` line on this host. Do not
assume it.

- [ ] **Step 3: Apply the per-slot budgets.**

In `go_layer` (`tools/verify.sh:241-250`), and only there:

```sh
go_layer() {
    local mod="$1"
    (
        cd "$mod"
        export GOMAXPROCS="${ATLAS_GOMAXPROCS:-6}"
        go build -p "${ATLAS_GO_P:-6}" ./... && go vet ./... || exit 1
        if [ "$QUICK" -eq 0 ]; then
            go test -p "${ATLAS_GO_TEST_P:-2}" -race ./... || exit 1
        fi
    )
}
```

`go vet` takes no `-p`; do not add one. The `rel` local at line 242 is unused
today — leave it or remove it, but do not invent a use for it.

- [ ] **Step 4: Slot the two heavy phases.**

`source "$ROOT/tools/lib/build-slot.sh"` near the top of `verify.sh`, beside
the other setup at lines 23-24.

Bake step (from Tasks 3 and 5) — the CLI wrapper is the right tool here,
because the command is an external process:

```sh
        step "docker buildx bake (${#TARGETS[@]} target(s))" \
            ./tools/with-build-slot.sh "bake" -- \
            docker buildx bake --builder "${ATLAS_BUILDER:-atlas}" --set "$BAKE_OUTPUT" "${TARGETS[@]}"
```

Go pool (from Task 4) — the library functions, because `launch_go_layers` is a
shell function and cannot be `exec`'d across a process boundary. One slot
covers the whole parallel phase, and only when `-race` is actually running:

```sh
    if [ "$FACTS" -eq 0 ]; then
        if [ "$QUICK" -eq 0 ]; then
            if acquire_build_slot "go test -race"; then
                launch_go_layers
                release_build_slot
            else
                FAILED+=("build slot (go test -race)")
            fi
        else
            launch_go_layers
        fi
    fi
```

Do not wrap each worker subshell in its own slot: that is the wrong
granularity and multiplies broker invocations by module count.

- [ ] **Step 5: Doc.**

Update `### Build slots` to name which two phases are slotted and which are
deliberately not (vet, guards, lint, `--facts`), and add the preflight's
thresholds and env overrides (`ATLAS_MIN_FREE_MB`, `ATLAS_MIN_TMP_MB`) to
`docs/verification.md`.

### Acceptance

- `tools/verify_test.sh`, `tools/with-build-slot_test.sh` and
  `tools/lib/build-slot_test.sh` all exit 0.
- `ATLAS_MIN_FREE_MB=99999999 tools/verify.sh --base HEAD --no-docker --no-ui`
  exits non-zero with a `preflight` failure, and the same command without the
  override exits 0 — proving the preflight fails closed but is not
  permanently on.
- `tools/verify.sh --quick --base HEAD` exits 0 (the preflight is off the
  quick path).
- Four concurrent `tools/verify.sh --quick` runs from four different worktrees
  all exit 0, with no OOM kill in `dmesg -T | tail -50` and no `go` cache
  error. Record in `measurements.md` under `## Layer 3 — concurrency`
  (acceptance criterion 3).

---

## Task 8: Serialise the module-cache writers (Layer 3, GOMODCACHE lock)

`tools/tidy-all-go.sh` runs `go mod tidy && go mod download` across 89 modules
with no coordination, and `GOMODCACHE` is machine-global. This is the one
genuinely unsafe concurrency in the system.

### Files

- `tools/tidy-all-go.sh` — take an exclusive lock around the whole sweep
- `tools/tidy-all-go_test.sh` — **new file**
- `docs/verification.md` — `### Build slots` subsection

### Steps

- [ ] **Step 1: Write the failing test.**

The suite must not actually tidy 89 modules. The observable claim is that the
script produces no work while the lock is held.

| case | fixture | invocation | expect |
|---|---|---|---|
| blocks while the lock is held | `ATLAS_GOMODCACHE_LOCK="$tmp/lock"`; hold it with `flock -x "$tmp/lock" sleep 5 &` | `timeout 3 tools/tidy-all-go.sh` | output contains no `==> ` line |
| proceeds when the lock is free | `ATLAS_GOMODCACHE_LOCK="$tmp/lock"`, nothing holding it | `timeout 10 tools/tidy-all-go.sh` | output contains at least one `==> ` line |
| creates the lock's parent dir | `ATLAS_GOMODCACHE_LOCK="$tmp/nested/deep/lock"` | `timeout 10 tools/tidy-all-go.sh` | `$tmp/nested/deep` exists |
| the lock is distinct from the build slots | read `tools/tidy-all-go.sh` | the default path is `/var/tmp/atlas/gomodcache.lock`, not under `slots/` |

The "no `==> ` line" assertion is what discriminates a real lock from a
missing one: without the lock the script starts tidying immediately and prints
its first `==>` within milliseconds. Asserting `timeout` exit codes would NOT
discriminate — both the locked and the unlocked case are killed with 124.

- [ ] **Step 2: Take the lock.**

`tools/tidy-all-go.sh` becomes:

```sh
#!/usr/bin/env bash
# tools/tidy-all-go.sh — `go mod tidy && go mod download` across every
# workspace module.
#
# These mutate GOMODCACHE, which is machine-global while worktrees are not.
# Concurrent sessions running this against the same cache is the one genuinely
# unsafe concurrency in the build system, so the whole sweep takes an exclusive
# lock — distinct from the counting build slots in tools/lib/build-slot.sh,
# which bound CPU and RAM rather than protecting a shared mutable store.
set -euo pipefail

LOCK="${ATLAS_GOMODCACHE_LOCK:-/var/tmp/atlas/gomodcache.lock}"
mkdir -p "$(dirname "$LOCK")"
exec 9>"$LOCK"
flock 9

mods=$(
  find ./services ./libs -name go.mod -print0 \
    | xargs -0 -n1 dirname \
    | sort -u
)

while IFS= read -r d; do
  echo "==> $d"
  (cd "$d" && go mod tidy && go mod download)
done <<< "$mods"
```

Note the added `set -euo pipefail` — the current script has none, so a failing
`go mod tidy` in module 3 of 89 is silently ignored today. That is a real fix,
not scope creep; call it out in the commit message.

`grep -rn "go work sync" tools/ .github/` returns only a comment in
`tools/lint.sh:15`, so there is no second writer to lock.

- [ ] **Step 3: Doc.**

One paragraph under `### Build slots`: the module-cache lock exists, what it
protects, why it is not one of the counting slots, and the env override.

### Acceptance

- `tools/tidy-all-go_test.sh` exits 0.
- `tools/shell-guard.sh --require-shellcheck` exits 0.

---

## Task 9: Narrow the `libs/` fan-out to the reverse-dependency closure (Layer 5)

`changed_modules()` (`tools/verify.sh:192-236`) returns *all* 89 modules
whenever any path under `libs/` or `go.work` appears in the change set. Since
`CHANGED` is the whole-branch diff, one `libs/` commit makes every subsequent
flagless run on that branch a full 89-module `-race` build for the life of the
branch.

Land this **last**, and validate the closure against the current all-modules
result before it becomes the default.

### Files

- `tools/lib/module-graph.sh` — **new file**; the workspace require graph
- `tools/lib/module-graph_test.sh` — **new file**
- `tools/verify.sh` — `changed_modules` (192-236) and the `--facts`
  `fanout_reason` block (691-697)
- `tools/verify_test.sh` — closure assertions
- `docs/verification.md` — `## The Go layer`

Precedent for a `tools/lib/` script with its own suite:
`tools/lib/analyzer-guard.sh` + `tools/lib/analyzer-guard_test.sh`.
`changed_tool_suites` (`tools/verify.sh:159-177`) matches `^tools/.*\.sh$` and
derives `${f%.sh}_test.sh`, so the new suite is picked up automatically.

### Steps

- [ ] **Step 1: Write the failing test.**

`tools/lib/module-graph_test.sh` — a unit test over a synthetic workspace in
`mktemp -d`, so it does not depend on the repo's real 89-module graph and does
not get slower as the repo grows.

Fixture, written by the test:

| file | module directive | requires |
|---|---|---|
| `$tmp/libs/a/go.mod` | `example.test/libs/a` | — |
| `$tmp/libs/b/go.mod` | `example.test/libs/b` | `example.test/libs/a` |
| `$tmp/svc/one/go.mod` | `example.test/svc/one` | `example.test/libs/b` |
| `$tmp/svc/two/go.mod` | `example.test/svc/two` | `example.test/libs/a` |
| `$tmp/svc/three/go.mod` | `example.test/svc/three` | — |

Function under test: `module_consumers <root> <changed-dir>...`, printing one
module directory per line, sorted, absolute.

| case | changed dirs | expect (sorted, relative to `$tmp`) |
|---|---|---|
| direct consumer only | `libs/b` | `libs/b`, `svc/one` |
| transitive closure | `libs/a` | `libs/a`, `libs/b`, `svc/one`, `svc/two` |
| unrelated module never selected | `libs/a` | output does not contain `svc/three` |
| two changed libs union | `libs/a`, `libs/b` | `libs/a`, `libs/b`, `svc/one`, `svc/two` |
| a changed dir with no consumers | `svc/three` | `svc/three` |
| a changed dir that is not a module | `$tmp/nope` | exit 0, `$tmp/nope` absent from the output |
| a `require` block, not a single line | rewrite `svc/one/go.mod` to use `require ( … )` | same as the direct-consumer case |
| an `// indirect` require is still an edge | mark `svc/two`'s require indirect | `svc/two` still selected for `libs/a` |
| cycle-safe | add `example.test/libs/b` to `libs/a`'s requires | terminates; output is `libs/a`, `libs/b`, `svc/one`, `svc/two` |

Then, in `tools/verify_test.sh`:

| case | fixture | invocation | expect |
|---|---|---|---|
| a real `libs/` change no longer selects every module | untracked `libs/atlas-tenant/zz-verify-probe.go` | `--facts --quick --base HEAD` | `modules_selected` is less than `find services libs -name go.mod \| wc -l` |
| the fan-out reason names the closure | same | `--facts --quick --base HEAD` | `fanout_reason` starts `shared-lib-closure:libs/atlas-tenant` and carries a consumer count |
| the escape hatch restores the old behaviour | same | `ATLAS_LIBS_FANOUT=all … --facts --quick --base HEAD` | `modules_selected` equals the total module count and `fanout_reason` starts `shared-lib:` |
| no libs change, no fan-out | probe removed | `--facts --quick --base HEAD` | `fanout_reason=none` |
| structural: `go.work` still fans out to everything | — | read `tools/verify.sh` | the `go.work` branch of `changed_modules` still calls `all_modules` |

`go.work` is a fixed tracked path and cannot be probed with an untracked
file, which is why that branch is covered structurally plus by the
`ATLAS_LIBS_FANOUT=all` case.

Add the probe file to `cleanup()` at `tools/verify_test.sh:101` and its
`trap`, following the fixed-name convention there.

- [ ] **Step 2: Implement `tools/lib/module-graph.sh`.**

A sourceable library — no `set -e` at file scope, since `tools/verify.sh` sets
its own — exposing:

- `module_path_of <dir>` — the `module` directive from `<dir>/go.mod`.
- `module_consumers <root> <dir>...` — the transitive reverse-dependency
  closure of the given module directories over the workspace graph, unioned
  with the given directories themselves; one absolute directory per line,
  sorted and unique.

Build the graph by reading each workspace `go.mod` once: the `module` line is
the node, and every `require` entry — both the single-line and the block form
— whose target is another workspace module path is an edge. An `// indirect`
require is still a real edge and must be kept. Restrict edges to module paths
present in the workspace; external deps are irrelevant here.

The libs are addressable module paths and every consumer names them in its own
`go.mod`, so the edge set is mechanical — do not add heuristics or path-prefix
guessing. Read `libs/atlas-tenant/go.mod` and confirm the actual module-path
prefix before writing any matcher; do not assume it.

BFS from the changed set over reverse edges with a visited set, so a cycle
terminates.

- [ ] **Step 3: Wire it into `changed_modules`.**

Split the current single fan-out predicate in two:

- `go.work` in the change set → `all_modules`, unchanged, with the existing
  warning text.
- `libs/` paths only → map each changed `libs/<name>/…` path to its module
  directory, call `module_consumers`, and union with the directly-changed
  service modules from the existing loop at lines 227-235.
- `ATLAS_LIBS_FANOUT=all` → keep the old all-modules behaviour, so Step 4's
  validation and any future doubt have a one-variable escape hatch.

The warning at lines 210-223 currently explains a full fan-out. Replace it
with the equivalent for the closure: which lib changed, how many consumers
were selected, and that `ATLAS_LIBS_FANOUT=all` restores the old behaviour.
Keep it on **stderr** — the comment at lines 218-221 explains exactly why a
stdout line here becomes a phantom module and fails the gate.

In the `--facts` block (lines 691-697) emit
`fanout_reason=shared-lib-closure:<first-lib-path> (<n> consumers)` for the
new branch, keeping the `shared-lib:` prefix for the `ATLAS_LIBS_FANOUT=all`
and `go.work` cases so existing callers grepping that prefix still work.

- [ ] **Step 4: Validate the closure before trusting it.**

Required, and recorded in `measurements.md` under `## Layer 5 — fan-out`. On a
change set containing a real `libs/` edit, run the gate's fact block both ways:

```sh
ATLAS_LIBS_FANOUT=all tools/verify.sh --facts --quick --base HEAD
tools/verify.sh --facts --quick --base HEAD
```

The closure's `modules_selected` must be a strict subset of the all-modules
result, and every module the closure drops must be one whose `go.mod` does not
name the changed lib. Spot-check three of the dropped modules by `grep`ing
their `go.mod`, and record which three.

- [ ] **Step 5: Doc.**

Update `## The Go layer` in `docs/verification.md`: the fan-out is now the
transitive reverse-dependency closure over the workspace require graph, a
`go.work` change still reaches everything, and `ATLAS_LIBS_FANOUT=all`
restores the old behaviour.

### Acceptance

- `tools/lib/module-graph_test.sh` and `tools/verify_test.sh` both exit 0.
- On a `libs/` change set, `modules_selected` is strictly less than the total
  module count, and Step 4's validation is recorded.
- `tools/shell-guard.sh --require-shellcheck` exits 0.

---

## Final gate

After Task 9, on this branch:

```sh
tools/verify.sh
```

must exit 0 (flagless — acceptance criterion 7). `--quick` does not count.

`measurements.md` must by then carry all five recorded measurements
(criteria 1–5); criterion 6 is `tools/verify_test.sh`, which every task from
Task 3 onward extends.
