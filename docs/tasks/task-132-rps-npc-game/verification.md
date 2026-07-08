# task-132 RPS NPC Minigame — Full Verification Gate

Date: 2026-07-08
Worktree: `.worktrees/task-132-rps-npc-game` (branch `task-132-rps-npc-game`)
Scope: verification only — no source changes were made during this pass.

## Overall status

**ALL GREEN**, with one known-environmental caveat: `tools/redis-key-guard.sh`
reports a non-zero exit because of a pre-existing, unrelated `atlas-data`
`storage/minio` missing-go.sum-entry failure (task-071, confirmed absent from
this branch's diff — see the Redis Key Guard section below). No raw-keyed-redis
finding exists in `atlas-rps` or any other task-132 file.

`go.work.sum` never drifted during any gate in this run — no revert was needed.

## Module gates

Each module was gated with `go test -race -count=1 ./...`, `go vet ./...`,
`go build ./...` from the module's own directory. All `go.work.sum` checks
after every command showed no diff.

| Module | test -race | vet | build |
|---|---|---|---|
| `libs/atlas-saga` | PASS | PASS | PASS |
| `libs/atlas-packet` | PASS | PASS | PASS |
| `tools/packet-audit` | PASS | PASS | PASS |
| `services/atlas-rps/atlas.com/rps` | PASS | PASS | PASS |
| `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator` | PASS | PASS | PASS |
| `services/atlas-channel/atlas.com/channel` | PASS | PASS | PASS |
| `services/atlas-tenants/atlas.com/tenants` | PASS | PASS | PASS |
| `services/atlas-npc-conversations/atlas.com/npc` | PASS | PASS | PASS |

No `FAIL` lines were observed in any `go test -race -count=1 ./...` run
(confirmed via `grep -c FAIL` returning `0` for atlas-channel, the largest
module gated). All `go vet` and `go build` runs completed with exit 0 and no
diagnostic output.

## Docker bakes

Run from the worktree root: `docker buildx bake <target>`. All targets
completed with exit 0 and produced a `naming to docker.io/library/<image>:local`
success line. Verified twice: once as part of a combined 5-target bake, once
individually confirmed against `docker images`.

| Bake target | Result |
|---|---|
| `atlas-rps` | SUCCESS — `atlas-rps:local` built and tagged |
| `atlas-saga-orchestrator` | SUCCESS — `atlas-saga-orchestrator:local` built and tagged |
| `atlas-channel` | SUCCESS — `atlas-channel:local` built and tagged |
| `atlas-tenants` | SUCCESS — `atlas-tenants:local` built and tagged |
| `atlas-npc-conversations` | SUCCESS — `atlas-npc-conversations:local` built and tagged |
| `atlas-configurations` (seed-JSON only, no go.mod touch; baked as a safety net) | SUCCESS — `atlas-configurations:local` built and tagged |

All 6 images are present in `docker images` with matching digests to the bake
output's `naming to docker.io/library/...` lines. No `error:` / `failed to`
build-step failures — the only lines matching `ERROR` in the log are literal
defensive shell strings embedded in Dockerfile `RUN` steps (e.g. `"ERROR:
build arg SERVICE is required"`), not actual failures; each of those steps
`DONE`d successfully.

## Repo-root gates (packet-audit)

Built once: `go build -o /tmp/pa27 ./tools/packet-audit` (exit 0), then each
gate run sequentially against the pre-built binary (no concurrent `go run`).

| Gate | Result |
|---|---|
| `dispatcher-lint` | PASS (exit 0) — `dispatcher-lint: clean` |
| `matrix --check` | PASS (exit 0) — no output (clean) |
| `fname-doc --check` | PASS (exit 0) — `fname-doc check OK (219 structs without an audit report carry no fname)` |
| `operations --check` | PASS (exit 0) — `operations check OK (0 absent-writer note(s))` |

## Redis Key Guard

`GOWORK=off tools/redis-key-guard.sh` — **exit 1**, but the failure is the
known pre-existing environmental issue, not a task-132 regression:

- The tool's static analysis errors out while type-checking
  `services/atlas-data/atlas.com/data/storage/minio/client.go` with:
  ```
  storage/minio/client.go:9:2: missing go.sum entry for module providing package github.com/minio/minio-go/v7 (imported by atlas-data/storage/minio)
  storage/minio/client.go:10:2: missing go.sum entry for module providing package github.com/minio/minio-go/v7/pkg/credentials (imported by atlas-data/storage/minio)
  ```
  This cascades into ~23 "analysis skipped due to errors in package" lines for
  unrelated packages loaded in the same `go vet`-style analysis pass.
- Confirmed unrelated to this branch: `git diff 38d4d0ba22 HEAD -- services/atlas-data`
  is empty — task-132 touches no file under `services/atlas-data`.
- Grepped the full guard output for any `.go:<line>:` finding: the **only**
  matches are the four `atlas-data/storage/minio/client.go` lines above. No
  `.go:` finding references `atlas-rps`, `atlas-npc-conversations`,
  `atlas-channel`, `atlas-tenants`, `atlas-saga-orchestrator`, or any other
  task-132 path.
- The guard's per-module progress line for `atlas-rps`
  (`rediskeyguard: .../services/atlas-rps/atlas.com/rps`) is just the scan
  announcement, not a violation — it completed with no raw-keyed-redis finding
  reported against it.
- Conclusion: **no new raw-keyed-redis violation introduced by task-132**; the
  guard's non-zero exit is caused entirely by the pre-existing, out-of-branch
  `atlas-data` minio go.sum gap. Recorded here as-is per instructions (not
  silently treated as a pass, but not attributable to this branch either).

## Kustomize (manifest parse)

| Overlay | Result |
|---|---|
| `kustomize build deploy/k8s/overlays/pr` | PASS (exit 0), 6076 lines rendered. Only a pre-existing deprecation warning (`'commonLabels' is deprecated`), no error. |
| `kustomize build deploy/k8s/overlays/main` | PASS (exit 0), 5248 lines rendered. Same pre-existing deprecation warning only. |

`kustomize` (standalone binary, `~/.local/bin/kustomize`) was available; no
fallback to `kubectl kustomize` was needed.

## Parked follow-ups

These are known, previously-scoped gaps in task-132's delivered feature set.
None block this verification pass; they are recorded here for visibility per
the task's own design/plan docs.

1. **v92 support** — parked, needs a v92 IDB to verify RPS packet opcodes/body
   layout before it can be implemented. Mirrors the existing
   `project_v92_mount_food_parked` precedent (task-086 mount-food handler):
   same class of blocker (no v92 IDB available), same resolution path (unblocks
   when a v92 IDB exists).
2. **Retry(5) restart-with-fee** — `atlas-channel` currently log-drops the
   `OnBtRetry` sub-op case; the "restart the minigame paying the fee again"
   flow is deferred. In the interim the player re-talks to the NPC to start a
   fresh round — functionally equivalent from the player's perspective, just
   not wired to the in-UI retry button.
3. **Loss consolation prize** — the client's RPS UI shows a "500 meso
   consolation" string (`SP_3681`) on a loss, but the server does not grant
   any meso on loss. This is an explicit design default (see `design.md` /
   `context.md`): only an explicit collect (win) pays out; the client string
   is cosmetic/decorative in this implementation and not backed by a server
   grant.
4. **Reward-ladder item content** — the shipped `rps-rewards` default config
   is meso-only (no item rewards seeded). Per `reward-ladder.md`, neither an
   authentic Cosmic reward-ladder source nor an in-environment WZ/atlas-data
   item-id verification path was available, and CLAUDE.md's "do not ship an
   unverified item id" rule is binding — so item rewards were intentionally
   left out of the seed rather than guessed. Operators can add item rewards
   per tenant via the `rps-rewards` configuration resource (see
   `live-config-patch.md`).
5. **Blocked-pending-IDB versions from Tasks 14/16** — **none.** All 5
   supported tenant versions (v83/v84/v87/v95/jms per the task's version
   matrix) were IDA-verified for the RPS packet family; there is no
   version left in a blocked-pending-IDB state coming out of Tasks 14/16.
   (v92, item 1 above, is a separate, always-out-of-scope-for-this-task
   version, not a Task 14/16 leftover.)

---

## Code review results (Task 27 Step 5 — whole-branch)

Dispatched in parallel per the project code-review pattern (no atlas-ui TypeScript
changed, so no frontend reviewer).

- **plan-adherence-reviewer** (`audit-plan-adherence.md`): **PASS.** All 27 plan tasks
  plus amendments 12b and 17b implemented with file:line evidence. All six
  controller-approved IDA-grounded deviations present and correct. All five supported
  versions IDA-verified; no blocked-pending-IDB cell. No stubs/TODOs/501s. The only gap
  it noted (missing `verification.md`) is this file, now produced.

- **backend-guidelines-reviewer** (`audit-backend.md`): every code-level DOM-*/SUB-*/SEC-*
  check **PASS** (immutable model+Builder, processor Interface+Impl + buffered/AndEmit,
  `tenant.MustFromContext`, Redis discipline via `libs/atlas-redis` lib types, DOM-24
  producertest/header parsers, money-path retry-safety, concurrency). One **BLOCKING
  DOM-23** finding — `COMMAND_TOPIC_RPS`/`EVENT_TOPIC_RPS` were absent from
  `deploy/k8s/base/env-configmap.yaml` (source of truth for `atlas-kafka-precreate`
  topic precreation and `gen-topic-config.sh` per-env suffixing) → **FIXED** (commit
  `fa0c28e6e7`; both keys added alphabetically; verified `atlas-kafka-precreate` iterates
  all `COMMAND/EVENT_TOPIC_*` env vars and `gen-topic-config.sh` yq-selects them, so the
  two-line configmap edit fully wires precreation + per-env isolation; kustomize both
  overlays re-parse clean). Non-blocking notes (accepted, not fixed): `characterId`/`npcId`
  as bare `uint32` (consistent with peer services), some `game` tests per-case vs
  table-driven, and the RETRY sub-op no-op (a tracked parked follow-up, listed above).

**Overall: GREEN.** All module gates, bakes, packet-audit gates, and kustomize pass; the
one blocking code-review finding (DOM-23) is fixed; the redis-key-guard failure is the
known pre-existing unrelated `atlas-data` minio go.sum issue (not a task-132 defect).
