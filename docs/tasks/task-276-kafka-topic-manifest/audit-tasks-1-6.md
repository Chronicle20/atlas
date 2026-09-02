# Plan Audit — task-276-kafka-topic-manifest (Tasks 1-6)

**Plan Path:** docs/tasks/task-276-kafka-topic-manifest/plan.md
**Audit Date:** 2026-08-30
**Branch:** task-276-kafka-topic-manifest
**Base Branch:** main (merge-base dd00cd7a5)
**Scope:** Tasks 1-6 only (Task 7-12 audited separately in `audit-tasks-7-12.md`)

## Executive Summary

Tasks 1-6 are fully and faithfully implemented. `topic.Token` and the fatal
`EnvProvider` contract (Task 1), the `tools/topicmod` codemod's type rules
R1/R2 (Task 2) and error rules R3/R4 (Task 3) are all present with passing
table-driven tests. Task 4's repository-wide sweep was executed correctly and
**uniformly** — no remaining bare `TOPIC`-shaped string constant, no
remaining `NewConfig(token string)` wrapper, and no remaining raw
`os.Getenv("...TOPIC...")` call outside the three deliberately-optional
`libs/atlas-service` sites, all verified by direct repository grep rather
than sampling. The codemod's own `-check` gate reports exit 0 (idempotent,
no residue). Task 5's `libs/atlas-kafka/gen` scanner produces exactly the
159-entry manifest the design predicted, with the three `compact`-policy
tokens correct and `_test.go`-only tokens structurally excluded. Task 6's
`env-configmap.yaml`/`kafka-topics-configmap.yaml` rendering is exact: the
comment-preserving hand-key block, the marker splice, the 17-orphan-removed
+2-`STATUS_*`-added arithmetic (174 → 159), and `kustomization.yaml`'s
alphabetically-placed new resource all check out byte-for-byte against
git history. All directly-verified module builds and tests pass
(`libs/atlas-kafka`, `tools/topicmod`, `libs/atlas-kafka/gen`,
`libs/atlas-outbox`, `libs/atlas-service`, plus a partial full-workspace
`tools/test-all-go.sh` run and a build-only spot-check of `atlas-channel`,
`atlas-saga-orchestrator`, `atlas-merchant`, `atlas-marriages`,
`atlas-maps`). No blocking findings in this range.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | `topic.Token` type + `EnvProvider` fatal contract | DONE | `libs/atlas-kafka/topic/token.go:12` (`type Token string`); `libs/atlas-kafka/topic/provider.go:17-20` returns `fmt.Errorf("topic token [%s] has no value in the environment", token)` on unset/empty; `provider_test.go` `TestEnvProvider` passes all 3 subtests (`go test ./topic/... -v` → PASS) |
| 2 | `tools/topicmod` type rules R1, R2 | DONE | `tools/topicmod/rewrite.go`, `testdata/r1_decl/`, `testdata/r2_buffer/`; `GOWORK=off go test ./...` → `TestRewrite/r1_decl` and `/r2_buffer` PASS |
| 3 | `tools/topicmod` error rules R3, R4 | DONE | `testdata/r3_propagate*/`, `testdata/r4_newconfig*/` present and passing; also extended with `TestRewriteR3Residue`, `TestRewriteR4Residue`, `TestRunCheckReportsUnmigratedSite` (all PASS) beyond the plan's minimum — legitimate hardening, not scope creep affecting the required behavior |
| 4 | Run the codemod across the repository | DONE | See "Task 4 sweep verification" below — verified uniform, not sampled |
| 5 | `libs/atlas-kafka/gen` scanner, `topics.yaml`, `policies.yaml` | DONE | `libs/atlas-kafka/gen/scan.go`, `manifest.go`, `policies.yaml`; `topics.yaml` has exactly 159 `- token:` entries (`grep -c '^  - token:'`); `TestScan`, `TestStalePolicyIsAnError`, `TestDrift` all PASS (`GOWORK=off go test ./... -v`, 16.8s) |
| 6 | Render `env-configmap.yaml` + `kafka-topics-configmap.yaml` | DONE | `libs/atlas-kafka/gen/splice.go`, `render_configmap.go`; `deploy/k8s/base/env-configmap.yaml` restructured with hand keys above `# BEGIN generated:topics`/`# END generated:topics` markers, `ATLAS_ENVIRONMENT`'s 12-line rationale comment byte-identical to merge-base; `deploy/k8s/base/kafka-topics-configmap.yaml` new, no `packages:` key, `sync-wave: "-1"` annotation present; `kustomization.yaml:77-78` lists both in alphabetical position |

**Completion Rate:** 6/6 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

### Task 4 sweep verification (uniform, not spot-checked)

The instructions flagged Task 4 as the task most likely to be partially
applied. I verified it with repository-wide greps rather than sampling
individual files:

- `grep -rnE '^\s*[A-Za-z0-9_]+\s*=\s*"[A-Z0-9_]*TOPIC[A-Z0-9_]*"\s*$' --include="*.go" services libs | grep -v _test.go` → **0 matches**. No bare-string topic constant declaration survives outside test files anywhere in `services/` or `libs/`.
- `grep -rln 'func NewConfig' services --include="*.go" | xargs grep -l 'token string' | grep -v _test.go` → **0 matches** (63 `NewConfig` functions found; all retyped). Independently, `grep -rln 'token topic.Token' services --include='*.go' | grep -v _test.go` → 66 files.
- `grep -rnE 'os\.(Getenv|LookupEnv)\("[A-Z0-9_]*TOPIC[A-Z0-9_]*"' services libs --include="*.go" | grep -v _test.go` → only `services/atlas-kafka-precreate/main.go:61` (`KAFKA_TOPIC_MANIFEST_PATH`, a config path variable name matching the regex coincidentally, not a topic token — a Task 10/11 concern, out of this range, already allowlisted per commit `4cf425bbc`).
- Building the codemod fresh from the current tree and running `topicmod -check ./services ./libs` from the repo root exits **0** — no un-migrated site remains, confirming the codemod's own idempotency claim (plan Task 4 Step 5) still holds against the final tree.
- The four named hand-edit seams verified directly: `libs/atlas-kafka/producer/provider.go:10` (`Provider func(token topic.Token) MessageProducer`), `libs/atlas-kafka/producer/manager.go:66,133` (`Writer(l, token topic.Token)`, `ManagerWriterProvider`; `m.writers` map keys confirmed still `string`), `libs/atlas-outbox/bridge.go:21` (`contents map[topic.Token][]kafka.Message]`; `Message.Topic` at `Enqueue(...)` call confirmed still `string`), `services/atlas-marriages/.../consumer.go` bare literal replaced by `marriageMsg.EnvEventTopicStatus` (grep confirms `"EVENT_TOPIC_MARRIAGE_STATUS"` now appears only in `_test.go`/`integration_test.go` files), `services/atlas-merchant/.../notification_task.go:52` residue site now assigns to a named `err` and warns/returns rather than discarding, `libs/atlas-service/topics.go` new file with the 3 `topic.Token` constants, `envregistry.go:52,59` and `projection.go:67,68` all converted to `os.Getenv(string(EnvEventTopicConfiguration...))`.
- `agent-ledger.tsv` shows Task 4 was executed across ~9 large iterative batches (`Task 4d batch 1` through `batch 7`, plus fix rounds `Task 4 fix1 cont`, `Task 4c`) each independently reviewed (`task-reviewer`) and gated (`task-verifier`/`gate` entries), with the SA9004 mixed-const-group gap caught by staticcheck and closed by an added R5 rule (`d50ba7b03`, with its own passing test `TestRewriteR5PreservesIota`) rather than a hand-patch. All batches terminated `DONE`/`APPROVED` or `APPROVED_WITH_FINDINGS` with `caused_fix` resolved by a subsequent commit.
- Orphan/addition arithmetic cross-checked directly against merge-base: `comm -23`/`comm -13` between the merge-base's 174 `env-configmap.yaml` topic keys and the current 159 shows exactly the 17 orphans design.md predicted removed (`COMMAND_TOPIC_ACCOUNT_LOGOUT`, `COMMAND_TOPIC_CHARACTER_GENERAL_CHAT`, `COMMAND_TOPIC_DROP_ITEM`, `COMMAND_TOPIC_EQUIP_ITEM`, `COMMAND_TOPIC_INVENTORY`, `COMMAND_TOPIC_MONSTER_DAMAGE`, `COMMAND_TOPIC_MOVE_ITEM`, `COMMAND_TOPIC_UNEQUIP_ITEM`, `COMMAND_TOPIC_WZ_EXTRACTION`, `EVENT_TOPIC_CHARACTER_GENERAL_CHAT`, `EVENT_TOPIC_CHARACTER_MOVEMENT`, `EVENT_TOPIC_EQUIP_CHANGED`, `EVENT_TOPIC_INVENTORY_CHANGED`, `EVENT_TOPIC_ITEM_GAIN`, `EVENT_TOPIC_MONSTER_MOVEMENT`, `EVENT_TOPIC_PET_MOVEMENT`, `EVENT_TOPIC_SKILL_MACRO`) and exactly the 2 additions (`STATUS_EVENT_TOPIC_SKILL_MACRO`, `STATUS_TOPIC_CASH_ITEM`) FR-6.1 requires. 174 − 17 + 2 = 159, matching the scanner's actual count.

Task 4's Step 6 instruction to land the retype as "two commits" was not
followed literally — the mechanical retype landed as one commit
(`e0fb2ac52`) followed by a long series of separate fix/residue commits
(`c54cd6111`, `0c4b0280f`, `b50dd51bb`, `d50ba7b03`, and the ~30
`fix(atlas-*): restore ... topic env token(s) for tests` commits). The plan
explicitly permits this ("If the working tree cannot be cleanly split... commit
once... and record in `context.md`"), but `context.md` does not carry that
specific record — it documents the general sizing rationale for Task 4
being one large task, not the single-vs-two-commit split decision. This is a
**minor documentation gap**, not a functional gap: the net result (one
retype commit plus reviewed fix commits, all landing before Task 5 begins,
per the ledger) is behaviourally equivalent and every commit is independently
inspectable in git history. Non-blocking.

## Skipped / Deferred Tasks

None. All 6 tasks in range are `DONE`.

## Build & Test Results

| Module / Service | Build | Tests | Notes |
|---|---|---|---|
| `libs/atlas-kafka` | PASS | PASS | `go test ./topic/... -v` — `TestEnvProvider` and subtests all PASS |
| `tools/topicmod` (`GOWORK=off`) | PASS | PASS | `go test ./... -v` — 8 `TestRewrite` subcases + 6 supporting tests, all PASS |
| `libs/atlas-kafka/gen` (`GOWORK=off`) | PASS | PASS | `go test ./... -v` — `TestScan` (7.6s), `TestDrift` (8.9s), `TestSplice` (6 subtests), `TestEmitEnvConfigMapBlock`, `TestEmitTopicsConfigMap`, plus Task-7-scope `TestEmitOverlayBlock*`/`TestEmitComposeBlock`, all PASS |
| `libs/atlas-outbox` | PASS | PASS | `go test ./... -count=1` |
| `libs/atlas-service` | PASS | PASS | `go test ./... -count=1` |
| `services/atlas-channel` | PASS | not run | build-only spot-check (largest declaration count, 70) |
| `services/atlas-saga-orchestrator` | PASS | not run | build-only spot-check (36 declarations) |
| `services/atlas-merchant` | PASS | not run | build-only spot-check (the one true residue site) |
| `services/atlas-marriages` | PASS | not run | build-only spot-check (the one bare-literal site) |
| `services/atlas-maps` | PASS | not run | build-only spot-check (12 declarations) |
| Full workspace (`tools/test-all-go.sh`, background run) | PASS through `atlas-merchant` (alphabetical cutoff, see notes); `libs/atlas-kafka/gen` FAILs by design of the script | PASS through `atlas-merchant`; `libs/atlas-kafka/gen` FAILs by design of the script | Ran under a 590s timeout and reached through `services/atlas-merchant` alphabetically (all `libs/*` and services `atlas-account` … `atlas-merchant` PASS, no other failures observed in that span) before the timeout truncated the run — it did not reach the remaining ~50 services, so this is corroborating, not exhaustive, evidence. The one real `FAIL` (`libs/atlas-kafka/gen`: `pattern ./...: directory prefix . does not contain modules listed in go.work`) is expected and non-blocking: `tools/test-all-go.sh` is a plain `find ./services ./libs -name go.mod \| go test ./...` convenience sweep that has not been updated to skip or `GOWORK=off` the module Task 5 deliberately keeps out of `go.work`. The actual merge gate, `tools/verify.sh:613-615`, already handles this correctly — it runs `cd libs/atlas-kafka/gen && GOWORK=off go test ./...` as its own dedicated step whenever `libs/atlas-kafka/gen/` (or a downstream generated artifact) is touched, with a comment at `tools/verify.sh:183-184` explicitly crediting this design decision. So the authoritative gate is unaffected by this; only the informal helper script is stale. |

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE (for this range; overall branch recommendation depends on the Task 7-12 shard's findings and the concurrently-running repo-wide `tools/verify.sh`)

## Action Items

1. (Non-blocking, documentation only) Add a line to `context.md` recording that Task 4's retype landed as one commit (`e0fb2ac52`) followed by reviewed fix/residue commits rather than a clean two-commit split, per the plan's own escape clause.
2. (Non-blocking, low priority) `tools/test-all-go.sh` was never updated to account for `libs/atlas-kafka/gen` living outside `go.work` (Task 5), so it now always fails one module by design mismatch. Consider adding a `GOWORK=off`/skip case there too, purely so the helper script's output stays legible — `tools/verify.sh` (the actual gate) is unaffected.
3. Confirm the concurrently-running repo-wide `tools/verify.sh` completes green before merge.
