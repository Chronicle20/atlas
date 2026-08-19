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

## The Go layer

Per changed module: `go build ./...`, `go vet ./...`, `go test -race ./...`.

A change to `go.work` or a shared lib fans out to every module, and the script
expands the set accordingly: services consume libs through the workspace, so a
lib edit can break a service with no changed file of its own.

`go vet` runs full-module here on purpose. The lint guard's `govet` is
diff-gated (`--new-from-rev`), so it will not see a pre-existing vet failure in
a file you happened to touch.

## The docker layer

`docker buildx bake atlas-<svc>` for every service whose `go.mod` changed.
**This is mandatory, not optional.**

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
