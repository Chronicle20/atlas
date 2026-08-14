# Verification

`tools/verify.sh` is the pre-PR gate. It must exit 0 before a branch is called
"done," "ready for PR," or handed to `superpowers:finishing-a-development-branch`.

```sh
tools/verify.sh              # full gate — what you run before opening a PR
tools/verify.sh --quick      # inner loop: build/vet/guards, no docker, no -race
tools/verify.sh --all        # ignore change detection, run everything
tools/verify.sh --no-docker  # everything except the bake
```

The script mirrors the jobs in `.github/workflows/pr-validation.yml`. **CI is the
authority**: if the two ever disagree, the script is the bug. Path-gated CI jobs
are path-gated here too, against the merge base with `origin/main`.

This document explains *why* each check exists and what breaks when it is
skipped. The list of checks lives in the script, not here — do not maintain a
second copy of it in `CLAUDE.md` or anywhere else.

---

## The Go layer

Per changed module: `go build ./...`, `go vet ./...`, `go test -race ./...`.

A change to `go.work` or a shared lib fans out to every module, and the script
expands the set accordingly.

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

### Always run

| Guard | Invariant | Silent failure it prevents |
|---|---|---|
| `redis-key-guard.sh` | No keyed Redis commands on the raw `go-redis` client outside `libs/atlas-redis` | Tenant key-prefix bypass (FR-1.5, task-045) |
| `goroutine-guard.sh` | No bare `go` statements outside `libs/atlas-routine` | Unsupervised goroutine, lost panic (RR-6, task-115). Escape hatch: `//goroutine-guard:allow` |
| `outbox-guard.sh` | Outbox write discipline | Event published outside the transaction |
| `buff-duration-guard.sh` | No seconds→ms scaling in `COMMAND_TOPIC_CHARACTER_BUFF` `duration` fields | The unit contract has been flipped three times in prose alone. Owned by `services/atlas-buffs/.../kafka/message/character/kafka.go` (`ApplyCommandBody.Duration` — **milliseconds**). Fingerprints json tag sets, not type names, because the body struct is duplicated under seven local names. Escape hatch: `//buffdurationguard:allow <justification>` (task-190 FR-3.2) |
| `skill-job-id-guard.sh` | No raw `==`/`!=`/`case`/`Is(`/`IsA(` against version-divergent job/skill `…Id` constants | `job.GmId` (500) means Gm at v0.48 but Pirate at v0.61+. Use the version-aware resolver `constants.For(region,major,minor).Job.Resolve` / `.Skill.Resolve`. The banned list is derived from `docs/tasks/task-187-version-aware-id-semantics/audit/divergences.csv`, so it grows as future audit passes add divergent ids |

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

**Deploy / versions** — `gen-lb-ports.sh --check` and `check-version-coverage.sh`,
when `deploy/`, `tools/gen-lb-ports.sh`, or a `versions.json` changed. A new
client version needs LB socket ports; without them the version is unreachable
with no error anywhere.

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
  `mist-contract-mirror-guard.sh`, or `template-duplicate-binding-guard.sh`.
  The gate runs all three. A drifted trade or mist contract will pass CI today.
- CI's `atlas-constants-drift-guard` (generator output drift) has no local
  equivalent in the gate; `go test` in `libs/atlas-constants` covers most of it.
- Neither side covers the Go modules under `tools/` — CI's `detect-changes`
  matrix enumerates `services/` and `libs/` only, and the gate mirrors that.
  The analyzer guards' own sources are compiled as a side effect of building
  each guard, and `goroutineguard`/`buffdurationguard` self-test on every run;
  `rediskeyguard` and `outboxguard` have no test pass anywhere.

Closing the first gap means adding three CI jobs; until then, running the gate
locally is the only thing standing between a drifted mirror and production.
