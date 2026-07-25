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

## Round-loop completion (2026-07-17)

The items below were previously parked; they were completed this session after
live end-to-end testing in `atlas-pr-933` exposed that the round could not
actually begin/complete without them. All are wired end-to-end and covered by
tests (see the round-loop commits `3745d5c`..`b775bc0` + `78b6caf`):

- **START_SELECT (mode 9)** — the serverbound `START(0)`→`BEGIN`
  command→`ROUND_STARTED` event→clientbound `START_SELECT` frame handshake that
  enables the client's R/P/S buttons (and re-arms them after `CONTINUE`). Byte-
  fixtures + a verified matrix cell on all 9 versions (see the packet-audit).
- **Retry (mode 5) restart-with-fee** — `atlas-channel` now emits a `RETRY`
  command; `atlas-rps` re-charges the full `entryCostMeso` (blocking — a
  saga-submit failure aborts the restart, no free re-roll) and reopens a fresh
  round at rung 0 via `START_SELECT`.
- **Loss consolation prize** — `consolationMeso` (default 500, matching
  `SP_3681`) is granted on a **rung-0** loss (never won this game), deferred to
  the leave action (Exit/Retry) so the meso effect lands after the client
  renders the loss. A loss at rung ≥ 1 pays nothing. See `reward-ladder.md`.
- **Reward-ladder item content** — the ladder now ships the 10 streak
  certificates `4031332`–`4031341` (WZ- and live-verified; see
  `reward-ladder.md`), replacing the meso-only placeholder.
- **Fee visibility** — entry and retry fee deductions now set `ShowEffect`.

## Remaining parked follow-ups

None block merge; recorded for visibility.

1. **v92 support** — parked, needs a v92 IDB to verify RPS packet opcodes/body
   layout before it can be implemented. Mirrors the existing
   `project_v92_mount_food_parked` precedent (same blocker: no v92 IDB).
2. **Balance-gated Retry / `FAIL_NOT_ENOUGH_MESO` (mode 6)** — Retry blocks on
   a fee-saga *submit* failure, but does not pre-check the player's balance or
   route the client's mode-6 "not enough mesos" frame when the deduction fails
   *downstream* (insufficient funds). Feature-sized (needs a balance query or a
   saga-failure feedback loop); the client currently just restarts and the
   AwardMesos step fails server-side. Deferred.
3. **Consolation on pure abandonment** — the consolation requires the player to
   acknowledge the loss (Exit/Retry). A player who loses at rung 0 and simply
   disconnects/walks away forfeits it: the TTL sweeper (`game/task.go`) is
   intentionally payout-free (no ladder/saga plumbing) and reaps the session
   without awarding. Accepted tradeoff, not a silent gap.
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
