# Verification

`tools/verify.sh` is the pre-PR gate. It must exit 0 before a branch is called
"done," "ready for PR," or handed to `superpowers:finishing-a-development-branch`.

```sh
tools/verify.sh              # full gate — what you run before opening a PR
tools/verify.sh --quick      # inner loop: build/vet/guards, no docker, no -race
tools/verify.sh --all        # ignore change detection, run everything
tools/verify.sh --no-docker  # everything except the bake

tools/verify.sh --quick --base <rev>   # iteration gate: only the increment
```

Only the **flagless** invocation counts as verified. `--quick`, `--no-docker`,
and `--all` also exit 0 — the first two print a caveat and skip the bake and
`-race` — so "verify.sh exited 0" is not a pass unless it ran with no flags.
Never claim verified from a subset.

The script mirrors the jobs in `.github/workflows/pr-validation.yml`. **CI is the
authority**: if the two ever disagree, the script is the bug. Path-gated CI jobs
are path-gated here too, against the merge base with `origin/main`.

This document explains *why* each check exists and what breaks when it is
skipped. The list of checks lives in the script, not here — do not maintain a
second copy of it in `CLAUDE.md` or anywhere else.

---

## Asking the gate what it selected — `--facts`

Before investigating why a run was broad, slow, or skipped something, ask it:

```sh
tools/verify.sh --facts --quick --base <sha>
```

```text
base=8c3736a
changed_paths=41
changed_services=atlas-cashshop,atlas-channel
changed_libs=none
go_changed=true
ui_changed=false
fanout_reason=none
modules_selected=4
modules=services/atlas-cashshop/…,services/atlas-channel/…
guard_suites=none
bake_targets=none
gates_selected=3
gate=go build/vet (4 modules)
gate=go analyzer guards
gate=lint & format guard (4 module(s))
gates_skipped=11
```

Pass the same flags you would really run — the answer reflects them, so
`--facts --quick --base X` reports what `--quick --base X` would do. Every
informational line goes to stderr; stdout is `key=value` only.

**It cannot drift from a real run.** `--facts` does not re-implement the
selection logic: it runs the script's real body and neuters `step()`, so each
gate records its label instead of executing. `tools/verify_test.sh` asserts both
the behavioural agreement (selected and skipped label sets match a real run over
the same change set) and the structural invariant that a gate label can only
originate inside `step()`.

This exists because the alternative was measured: ~30 turns at 170–290k context
spent reverse-engineering the change-detection, module selection, and guard
gating from this script's source — ≈6.9M tokens, about a quarter of one
controller session — to learn facts the script had already computed. If you find
yourself running `git diff` against `tools/verify.sh`, or grepping it to see how
it invokes guards, run `--facts` instead.

## The iteration gate

The default change base is the merge base with `origin/main` — the **whole
branch**. That is right for the pre-PR gate and wrong for a per-task gate run
twenty times on the same branch, because a single `libs/` commit fans every
later run out to all 86 modules (see below). On task-227 that made each
`--quick` run ~11 minutes instead of ~1, for 24 hours, silently.

For a gate you run per task, scope it to the increment:

```sh
tools/verify.sh --quick --base <last-commit-you-already-gated>
```

Launch the gate in the background and keep working; never idle waiting on it.

Measured on the task-227 branch: `--quick` resolved **86 changed Go modules**;
`--quick --base HEAD~1` resolved **2**. The script now prints a warning when
the fan-out happens on an un-narrowed base, so this can no longer be silent.

The narrowing is safe only because every commit in the range gets gated by
*some* run — the increment's base must be the last commit that actually passed,
not blindly `HEAD~1`. The flagless pre-PR run always uses the merge base and
covers the branch as a whole regardless.

## Host tuning (WSL2)

`/tmp` is a tmpfs — RAM, not disk — and WSL2 sizes it at 50% of the VM's RAM
by default. On a host measured with ~64 GiB and 24 logical CPUs, the VM
currently gets 31 GiB by that default, and every stale scratch file left in
`/tmp` is RAM taken from the compilers doing the build. This section is host
state, not repo state — none of it is applied by checking out this branch.
Task 7's preflight *detects* whether it has been applied and reports the
un-tuned condition; it does not assume this section was followed.

### `.wslconfig` — give the VM the memory and CPU the host actually has

Location: `C:\Users\<windows-user>\.wslconfig` (a placeholder — never a
literal home path).

```ini
[wsl2]
memory=52GB
processors=24
swap=16GB
```

Apply with `wsl --shutdown` from a Windows shell, then restart the WSL2
session.

### `/etc/fstab` — pin `/tmp` after the memory bump

WSL2's default `/tmp` sizing rule is 50% of VM RAM. Applied *after* the
`.wslconfig` bump above, that default would make `/tmp` 26 GiB — worse than
the 16 GiB it is today. Pin it explicitly instead:

```
tmpfs /tmp tmpfs rw,nosuid,nodev,size=4G,nr_inodes=1048576 0 0
```

### `TMPDIR` — move scratch off tmpfs

```sh
export TMPDIR=/var/tmp/atlas/scratch
```

Set in the user's shell profile (e.g. `~/.bashrc` or `~/.zshrc`). The repo
deliberately does **not** ship a tracked `.envrc` for this: an untracked
personal `.envrc` already exists in the main checkout and is not gitignored,
so a tracked one would break `git checkout` of this branch there. Relocating
`CLAUDE_JOB_DIR` is best-effort where the harness permits it; `TMPDIR` is the
load-bearing control — it is what `tools/scratch-sweep.sh`'s default root
(`/var/tmp/atlas/scratch`) is meant to align with.

### Sweeper — systemd user timer

`tools/scratch-sweep.sh` ages entries out of the scratch root; give it a
daily systemd **user** timer rather than relying on manual runs. Drop these
two unit files in the user systemd directory
(`~/.config/systemd/user/`, again a placeholder, never a literal home path):

`atlas-scratch-sweep.service`:

```ini
[Unit]
Description=Sweep atlas scratch root

[Service]
Type=oneshot
ExecStart=<repo-root>/tools/scratch-sweep.sh
```

`atlas-scratch-sweep.timer`:

```ini
[Unit]
Description=Daily atlas scratch sweep

[Timer]
OnCalendar=daily
Persistent=true

[Install]
WantedBy=timers.target
```

Enable with:

```sh
systemctl --user enable --now atlas-scratch-sweep.timer
```

### Build slots

`tools/lib/build-slot.sh` is a machine-wide counting semaphore: K slots,
shared by every worktree and every session on the host, so N concurrent
sessions cannot all run a heavy build gate at once and thrash the box. Task 7
wires it into `tools/verify.sh` itself; it also works standalone through the
CLI wrapper, `tools/with-build-slot.sh`.

**Why K=4.** The host measured earlier in this section has 24 logical CPUs
and, after the `.wslconfig` bump above, 52 GiB of VM memory. The per-slot
budget one heavy gate run actually uses is:

| resource | value |
|---|---|
| `GOMAXPROCS` | 6 |
| `go build -p` | 6 |
| `go test -p` | 2 |
| BuildKit `max-parallelism` | 8 (`deploy/buildkit/buildkitd.toml`) |

Four slots at 6 threads apiece cover the 24-thread budget without
oversubscribing it, and each slot's peak memory footprint (a `go build`, a
`go test`, and a BuildKit solve) fits inside 52 GiB / 4. A fifth concurrent
gate would either wait for a slot or blow through both budgets at once —
that's the failure mode this broker exists to prevent.

**What's slotted.** Exactly two phases of `verify.sh` acquire a machine-wide
slot: the bake (`docker buildx bake`, through the CLI wrapper, since it runs
as an external process) and the Go pool (`launch_go_layers`, held around the
function itself, and only when `-race` is actually running — a `--quick` pass
never competes for a slot). Everything else in `verify.sh` is deliberately
NOT slotted: `go vet` (part of the unslotted half of `go_layer`, cheap next to
`go build`/`go test -race`), the analyzer/lint/format guards, and the
`--facts` path, which executes no check at all.

**Usage.** A caller that runs the guarded work as a subprocess should use the
CLI wrapper:

```sh
tools/with-build-slot.sh verify -- tools/verify.sh
```

A caller that needs to hold the slot around a shell *function* — as
`tools/verify.sh` does around `launch_go_layers` — cannot use the wrapper (a
slot held by a subprocess releases the instant that subprocess exits, before
the guarded work even starts) and instead sources the library directly:

```sh
. tools/lib/build-slot.sh
acquire_build_slot "verify" || exit $?
...
release_build_slot
```

**The exit-75 contract.** Both the library and the CLI use exit code 75
(`EX_TEMPFAIL` from `sysexits.h`) to mean "gave up waiting for a slot," never
"the guarded command itself failed." Pass `--timeout SEC` (CLI) or set
`ATLAS_BUILD_SLOT_TIMEOUT` (library) to bound the wait; unset/absent blocks
until a slot frees. The wait itself is a blocking `flock`, not a polling
loop, so it costs no inference turns while it waits.

**The GOMODCACHE lock.** `tools/tidy-all-go.sh` takes an exclusive `flock` on
`/var/tmp/atlas/gomodcache.lock` (override with `ATLAS_GOMODCACHE_LOCK`)
around its whole sweep, before it walks a single module. This is a different
mechanism from the build slots above: the slots are a *counting* semaphore
bounding CPU/RAM across concurrent heavy gates, while this is an *exclusive*
mutex protecting a shared mutable store — `GOMODCACHE` is machine-global
while worktrees are not, so two sessions running `go mod tidy`/`go mod
download` against it at once is the one genuinely unsafe concurrency in the
build system. Nothing else acquires this lock; `tools/verify.sh` and the
build slots never touch it.

### Capacity preflight

Before the Go-module block, a non-`--quick` run gates on a `preflight
(capacity)` check: free RAM and free space under `TMPDIR` must clear a floor
before the heavy phases are allowed to start. This fails CLOSED — a starved
host fails the gate rather than proceeding into a slow, possibly-flaky build
— and it is off the `--quick` path entirely, so the fast inner loop is never
blocked by it.

| threshold | env override | default |
|---|---|---|
| free RAM | `ATLAS_MIN_FREE_MB` | 4096 (MiB) |
| free space under `TMPDIR` | `ATLAS_MIN_TMP_MB` | 8192 (MiB) |

The preflight also reports — never assumes — the un-tuned WSL2 condition from
"Host tuning (WSL2)" above: when `TMPDIR` resolves under `/tmp`, it prints a
pointer back to that section, because `/tmp` being tmpfs on this host is host
state that checking out this branch cannot apply.

## The Go layer

Per changed module: `go build ./...`, `go vet ./...`, `go test -race ./...`.
The build and test steps carry the per-slot budgets from "Build slots" above
— `GOMAXPROCS=6`, `go build -p 6`, `go test -p 2` (each overridable via
`ATLAS_GOMAXPROCS`, `ATLAS_GO_P`, `ATLAS_GO_TEST_P`) — so a module's own
internal parallelism stays inside the slot it is running in rather than
oversubscribing the host. `go vet` takes no `-p`; it is not a valid flag for
that subcommand.

A change to `go.work` fans out to every module: it is the workspace
membership list itself, not a require edge, so there is nothing narrower to
compute from it.

A change under `libs/` fans out to its transitive reverse-dependency closure
over the workspace `require` graph (Layer 5, `tools/lib/module-graph.sh`),
not to every module: services consume libs through the workspace, so a lib
edit can break a consumer with no changed file of its own, but it cannot
break a module that never named the lib, directly or transitively. The graph
is built once per run by reading every workspace `go.mod`'s `module` line and
`require` entries — both the single-line and block forms, and `// indirect`
requires count as real edges — then BFS'd from the changed lib(s) over
reverse edges. `ATLAS_LIBS_FANOUT=all` is the one-variable escape hatch back
to the old "any `libs/` change reaches every module" behaviour, for whenever
the closure is in doubt.

`go.work.sum` is deliberately **not** treated as a fan-out trigger: it is a
checksum artifact of resolving the workspace, and an ordinary local
`go build`/`go mod tidy` dirties it with no `require`-graph edge actually
changing. Before Layer 5 this mattered less because any `libs/` or `go.work`
change fanned out fully anyway; narrowing the `libs/` case to a closure while
leaving `go.work.sum` unanchored in the `go.work` match would have left a
second, silent path back to the full 89+-module fan-out — the common case in
daily use, not an edge case. A real `require`-graph edit that happens to also
dirty `go.work.sum` is still caught on its own merits, either as a `go.work`
change or as the `libs/`/`services/` path that actually changed.

`go vet` runs full-module here on purpose. The lint guard's `govet` is
diff-gated (`--new-from-rev`), so it will not see a pre-existing vet failure in
a file you happened to touch.

Modules build through a bounded worker pool: `ATLAS_VERIFY_GO_JOBS` workers run
concurrently (default `4`), not one `go build`/`vet`/`test -race` at a time —
significant on a `libs/`/`go.work` fan-out, where the change set can be every
module in the workspace. Each worker's output is captured to a per-module log
and replayed through the same `step()` bookkeeping in module order once the
pool drains, so a failure still reads as one labelled block naming its module,
and the summary's `PASSED`/`FAILED` counts are exactly what a serial run would
report — concurrency changes wall time only, never what gets reported or in
what order. Override the worker count with `ATLAS_VERIFY_GO_JOBS=<n>`; `0` or
a non-numeric value is rejected at startup.

## The docker layer

The repo-root `.dockerignore` is an allowlist (`*` then `!libs`, `!services`),
not a blocklist: only `libs/` and `services/` reach any root-context build.
Every existing root-context Dockerfile only `COPY`s from those two trees, so
today nothing is lost. Adding a new root-context Dockerfile (or a `COPY` line
that reaches outside `libs/`/`services/` in an existing one) means it silently
loses that content unless the `.dockerignore` gets another `!` line first —
check this before adding one.

`docker buildx bake` over every target whose `go.mod` changed — all selected
targets in a **single** invocation, not one bake per target: one context
transfer instead of one per target, and BuildKit shares the `libs/` mod-only
and source layers across targets within the solve instead of resolving them
once per invocation. A failure is BuildKit's own solve output naming the
failing target and step. **This is mandatory, not optional.**

The shared root `Dockerfile` is parameterized by `ARG SERVICE`; `docker-bake.hcl`
enumerates one target per Go service, driven by `.github/config/services.json`
(the single source of truth). `go build`/`go test` run against the workspace
`go.work` and will **not** catch a missing `COPY libs/...` line in the shared
Dockerfile — only the bake will. CI catches it, but each round-trip wastes a CI
cycle and turns "verified" into a lie.

Build everything: `docker buildx bake all-go-services` (or `tools/build-services.sh`,
a thin wrapper).

`verify.sh` runs its bake as a **build check only** — it passes
`--set '*.output=type=cacheonly'`, so it never writes to the docker image store.
That keeps a run in one worktree from replacing the `<svc>:local` image another
tree built, since `docker-bake.hcl` tags every target `<svc>:${ATLAS_IMAGE_TAG}`
(default `local`) — the same tag `deploy/compose/docker-compose.*.yml` runs, and
the image store is machine-global while worktrees are not. A broken build still
fails; only the export is dropped. To actually **produce** runnable
`<svc>:local` images, run `tools/build-services.sh` — `verify.sh` will not.

Adding a new shared lib requires two `COPY` lines in the root `Dockerfile` (one
in the mod-only block, one in the source block) and one `./libs/<name>` line in
`go.work`. That is all — no per-service edits.

For large refactors expect several fix-and-rebuild cycles. Do not shortcut the
bake step.

### The builder

`docker buildx bake` runs against a pinned `atlas` builder
(`docker-container` driver), not the default `docker` driver — the default
driver's build cache is unbounded and its parallelism is ungoverned.
`tools/buildx-bootstrap.sh` creates and selects it, from
`deploy/buildkit/buildkitd.toml`, which caps solve parallelism at 8 and
enforces a two-tier GC policy: aggressively-reclaimable cache (local
sources, cache mounts, git checkouts) is evicted first at a 40 GB/7-day
threshold, then a hard 60 GB ceiling applies regardless of type. `verify.sh`
asserts the builder exists (`tools/buildx-bootstrap.sh --check`) before its
bake step and fails closed if it does not.

Editing `deploy/buildkit/buildkitd.toml` requires
`tools/buildx-bootstrap.sh --force` to take effect — buildx cannot update an
existing builder's config in place; `--force` removes and recreates it.

The `docker-container` driver does not write to the local image store by
default, unlike `docker`. `tools/build-services.sh` — whose entire purpose is
producing runnable `<svc>:local` images — therefore always passes `--load`.
`verify.sh`'s own bake stays `--set '*.output=type=cacheonly'` and needs no
`--load`, since it never intends to produce an image.

Switching to the `atlas` builder means a brand-new BuildKit instance with an
empty cache. The first bake after `tools/buildx-bootstrap.sh` runs — for
either `verify.sh` or `tools/build-services.sh` — is a cold cache, including
the `/go/pkg/mod` and `/root/.cache/go-build` cache mounts the Dockerfile
relies on. That is expected, not a regression; subsequent bakes warm the new
builder's cache same as before.

## Adding a new service

Follow [`docs/adding-a-new-service.md`](adding-a-new-service.md) in full. It
enumerates every hand-maintained list a service must be registered in (CI,
docker-bake, go.work, k8s base, **both** kustomize overlays, databases, ingress)
and the silent-failure traps: unpinned `:latest` image, `behavior: replace`
configmap key drops, unsuffixed Kafka topic fallback.

`tools/service-registration-guard.sh` machine-checks those lists and runs as
part of the gate.

---

## Guards

Analyzer- and script-backed invariants. Each exists because the failure it
catches is silent — no build error, no test failure, just wrong behavior at
runtime.

### Go analyzer guards

Gated on "a Go module was affected." Locally that is `tools/verify.sh`'s
`.go`-touched check; in CI it is the affected-module matrices, plus a
`tools/`-changed signal so that editing an analyzer still forces a full sweep of
code that did not change. An analyzer cannot find a new violation in a diff that
contains no Go.

CI runs all four through a single job (`go-analyzer-guards`) driving
`tools/go-analyzer-guards.sh`, which links every analyzer into one `unitchecker`
binary. They were four jobs until a six-file docs-only PR spent its entire
7-minute run on them: four runners each cold-compiling the same dependency graph
to analyze the same 64 modules, because type-checking — not analysis — is where
the time goes. What each guard detects and what each guard scans are unchanged;
`tools/atlasguards/guards.go` holds the services-only vs services+libs split.

The per-guard `tools/<name>-guard.sh` entry points below remain the local
iteration path and the single-guard escape hatch.

| Guard | Invariant | Silent failure it prevents |
|---|---|---|
| `redis-key-guard.sh` | No keyed Redis commands on the raw `go-redis` client outside `libs/atlas-redis` | Tenant key-prefix bypass (FR-1.5, task-045) |
| `goroutine-guard.sh` | No bare `go` statements outside `libs/atlas-routine` | Unsupervised goroutine, lost panic (RR-6, task-115). Escape hatch: `//goroutine-guard:allow` |
| `outbox-guard.sh` | Outbox write discipline | Event published outside the transaction |
| `buff-duration-guard.sh` | No seconds→ms scaling in `COMMAND_TOPIC_CHARACTER_BUFF` `duration` fields | The unit contract has been flipped three times in prose alone. Owned by `services/atlas-buffs/.../kafka/message/character/kafka.go` (`ApplyCommandBody.Duration` — **milliseconds**). Fingerprints json tag sets, not type names, because the body struct is duplicated under seven local names. Escape hatch: `//buffdurationguard:allow <justification>` (task-190 FR-3.2) |

### Always run

`skill-job-id-guard.sh` — no raw `==`/`!=`/`case`/`Is(`/`IsA(` against
version-divergent job/skill `…Id` constants. `job.GmId` (500) means Gm at v0.48
but Pirate at v0.61+; use the version-aware resolver
`constants.For(region,major,minor).Job.Resolve` / `.Skill.Resolve`. The banned
list is derived from
`docs/tasks/task-187-version-aware-id-semantics/audit/divergences.csv`, so it
grows as future audit passes add divergent ids. Pure shell + python3 — it greps
rather than type-checks, so it is seconds, not minutes, and is not worth gating.

### Path-gated

**Registration lists** — `service-registration-guard.sh`, when `services.json`,
`deploy/k8s/`, `docker-bake.hcl`, `go.work`, or `tools/db-bootstrap.sh` changed.

**Toolchain pins** — `toolchain-pin-guard.sh`, when a `go.mod`, `go.work`,
`Dockerfile`, `docker-bake.hcl`, anything under `.github/`, `README.md`,
`tools/toolchain.versions`, or the guard's own source changed. Asserts every Go
/ Alpine / golangci-lint pin site agrees with `tools/toolchain.versions`.
Silent failure it prevents: a partially landed toolchain bump leaves the tree
building against two Go versions at once with nothing failing — which is how the
1.24/1.25/1.26 spread task-261 collapsed came to exist. The `tools/cideps`
fixtures are exempt by path prefix (FR-7); the synthesized workspace directive
in `Dockerfile` is derived from `ARG GO_VERSION`, not checked.

**Tenant socket-config templates** (`services/atlas-configurations/seed-data/templates/`) —
see [`docs/packets/TEMPLATE_CONVENTIONS.md`](packets/TEMPLATE_CONVENTIONS.md):

- `template-opcode-order-guard.sh` — strictly ascending `opCode` in both the
  `handlers` and `writers` arrays. New entries go at their sorted position,
  never appended next to a semantically-related entry.
- `template-duplicate-binding-guard.sh` — bans binding the same
  (implementation name, numeric opCode) pair twice. The leading-zero-padding
  duplicate (`0xB8` and `0x0B8`) made the dispatch map's last-write-wins
  behavior decide which entry's options survived (task-194). One name bound to
  several *distinct* opcodes is legitimate and untouched.
- `template-movement-types-guard.sh` — every move handler
  (`CharacterMoveHandle`, `MonsterMovementHandle`, `PetMovementHandle`,
  `SummonMoveHandle`, `NPCActionHandle`) must carry a non-empty `options.types`;
  all such arrays within one template must be byte-identical; every `Type` must
  be one the decoder recognizes; at most one entry may be named `FALL_DOWN`.
  A missing table makes every movement fragment decode as a 3-byte stub (loud:
  "Code [N] not configured for use in movement"); a typo'd `Type` does the same
  for one index, **silently**.

**Kafka contract mirrors** — one service owns the contract, others carry a copy
in a separate Go module. A field name or json tag changed in one copy and not
the others fails no build; it decodes into a zero-valued body at runtime,
silently. Each guard diffs the copies from the `package` clause onward; only the
leading doc comment, which names the mirror direction, may differ.

| Guard | Owner | Mirrors |
|---|---|---|
| `trade-contract-mirror-guard.sh` | atlas-trades `kafka/message/trade/kafka.go` | atlas-channel |
| `mist-contract-mirror-guard.sh` | atlas-maps `kafka/message/mist/kafka.go` | atlas-channel — a drifted mirror yields a mist with no bounds, no lifetime, no recovery magnitude and no party scope (task-218) |
| `npc-shop-contract-mirror-guard.sh` | atlas-npc-shops `kafka/message/shops/kafka.go` | atlas-channel, atlas-saga-orchestrator |
| `npc-conversation-contract-mirror-guard.sh` | atlas-npc-conversations `kafka/message/npc/kafka.go` | atlas-saga-orchestrator — a drifted mirror yields a conversation start with no item id, no avatar, or no transactionId, so the awaiting saga step never completes (task-230) |

**Operator cancel path** — path-gated on
`services/atlas-channel/atlas.com/channel/socket/handler/*.go` or a tenant
socket-config template changing. The property was narrowed from an earlier,
disproven "no client cancel path exists" assertion — see
`docs/tasks/task-227-cash-name-change-world-transfer/cancel-entry-point.md`
and `cancel-confirm-semantics.md`. The legitimate self-scoped route (`POST
.../pending-changes/cancel`, reason `"player_cancelled"`, task-227
client-cancel addendum) is unaffected by this guard.

| Guard | What it bans | Why the failure is silent otherwise |
|---|---|---|
| `operator-cancel-path-guard.sh` | A socket handler file referencing the reason string `"operator_cancelled"` or combining an HTTP DELETE call with the `pending-changes` resource path; a template binding the clientbound writer `CashShopCancelNameChangeResult` or `CashShopCancelTransferWorldResult` as a `handler` | Either would make the id-based, operator-only pending-change cancel route (`pending_change/resource.go:164`) reachable from a game-client packet — the route's own ownership check only verifies the id belongs to the calling `{characterId}` path segment, not that the caller is an operator, so a socket handler that reached it would work, just with the wrong actor |

**Deploy / versions** — `gen-lb-ports.sh --check` and `check-version-coverage.sh`,
when `deploy/`, `tools/gen-lb-ports.sh`, or a `versions.json` changed. A new
client version needs LB socket ports; without them the version is unreachable
with no error anywhere.

**Shell tooling** — `shell-guard.sh --require-shellcheck` plus the test suite of
each changed script, when any `tools/**/*.sh` changed.

The gate was previously hardcoded to `^tools/task-(resolve|brief)(_test)?\.sh$`,
so every other script in `tools/` was ungated — including `plan-lint.sh`, which
*executes* commands extracted from a plan file. A branch adding three `tools/`
scripts produced a flagless run in which all 14 checks skipped and the gate
still exited 0. A green gate that ran nothing is the failure mode this closes.

- `shell-guard.sh` parse-checks each script with the interpreter its shebang
  names, then runs `shellcheck -S error`. Severity `error` is deliberate: it is
  clean across the tree today, so the guard landed with zero legacy debt and any
  failure is a real regression. `-S warning` currently reports 19 pre-existing
  findings; raising the bar means fixing those first.
- `--require-shellcheck` fails when shellcheck is absent rather than degrading
  to a syntax-only pass. A guard that silently weakens into one that always
  passes is the same bug in a different place.
- Test suites are discovered by convention: a changed `tools/foo.sh` runs
  `tools/foo_test.sh` when it exists, and a changed `tools/foo_test.sh` runs
  itself. Adding a suite is enough to gate it — no edit to `verify.sh`.

CI mirror: the `shell-tooling-guard` job, ungated (the whole sweep is ~1s, so
path filtering would cost more than it saves).

---

**atlas-pr-bootstrap bats suite** — `bats services/atlas-pr-bootstrap/test`, when
anything under `services/atlas-pr-bootstrap/` changed.

That service's shell *is* the ephemeral-environment control plane:
`bootstrap.sh`, `cleanup.sh`, `sweep-orphans.sh` and their helpers create and
reclaim every PR namespace. It has shipped a substantial suite for a long time
and **nothing ran it** — the shell-tooling gate above reaches only `tools/` and
discovers only `*_test.sh`, while these are `*.bats`; `bats` appears nowhere in
`.github/workflows`; and `tools/task-facts.sh` merely probes whether the binary
is installed. The suite was therefore advisory, which is how the sparse
`SERVICE_ID` defect (a missing `uuidgen` silently yielding an empty id, then an
empty env var, crash-looping atlas-channel and atlas-login) reached a live
cluster past a test file well placed to have caught it.

- A missing `bats` is a hard failure, not a skip — the same reasoning as
  `--require-shellcheck` above. A silent skip is precisely how this suite went
  unrun for so long.
- The whole suite runs, not a per-file mapping: these files do not follow the
  `foo.sh` → `foo_test.sh` convention, and the suite is fast enough that
  selecting within it would buy nothing.

CI mirror: **none.** See "Known drift" below.

---

## Lint & format

`tools/lint.sh --check` (task-171). golangci-lint v2 formatters (gofumpt +
goimports, tree-wide) and `standard` linters (rev-gated to new code) across every
Go module, plus Prettier + ESLint for atlas-ui.

Fix mode is `tools/lint.sh` with no flags — it rewrites files in place. Run it
before committing.

Known footguns:

- `--check` false-fails without nvm on PATH (atlas-ui layer). Use `--go` to skip
  the UI layer when that is not what you are verifying.
- Cross-worktree golangci-lint lock contention: two worktrees linting at once
  will block. Serialize them.
- Merging `main` into a branch that predates task-171 requires a `tools/lint.sh`
  fix pass afterward.

---

## Known drift between this gate and CI

Tracked here so it is visible rather than folklore. Neither side is a strict
superset of the other; CI is the authority for everything it does run.

- CI has **no** job for `trade-contract-mirror-guard.sh`,
  `mist-contract-mirror-guard.sh`, `template-duplicate-binding-guard.sh`, or
  `operator-cancel-path-guard.sh`. The gate runs all four. A drifted trade or
  mist contract, or a socket handler that regains the operator cancel path,
  will pass CI today.
- CI's `atlas-constants-drift-guard` (generator output drift) has no local
  equivalent in the gate; `go test` in `libs/atlas-constants` covers most of it.
- CI runs **no** `bats` anywhere, so `services/atlas-pr-bootstrap`'s suite is
  gate-only. CI does not run `tools/verify.sh` either, so a PR touching that
  service passes CI without its suite ever executing — the local gate is the
  only thing standing behind it. Anyone wiring a CI job for it should mirror
  the hard-fail-on-missing-`bats` behaviour rather than skipping.
- Neither side covers the Go modules under `tools/` — CI's `detect-changes`
  matrix enumerates `services/` and `libs/` only, and the gate mirrors that.
  The analyzer guards' own sources are compiled as a side effect of building
  each guard, and `goroutineguard`/`buffdurationguard` self-test on every run;
  `rediskeyguard` and `outboxguard` have no test pass anywhere.
- `.github/config/services.json` lists **9 of the 22 Go modules under `libs/`**
  (it covers `services/` exactly, 64 of 64). Everything CI drives off the
  libraries matrix — `test-go-libraries`, `lint-go` — therefore does not see the
  other 13 at all. The gate's `changed_modules()` discovers modules with `find`
  and does cover them, so this is a CI-side hole, not a local one. It is also
  why `go-analyzer-guards` scopes its `services/` pass to the matrix but always
  sweeps all of `libs/`: scoping that pass to the matrix would have quietly
  narrowed `goroutineguard` and `buffdurationguard` coverage by 13 modules.

Closing the first gap means adding three CI jobs; until then, running the gate
locally is the only thing standing between a drifted mirror and production.
