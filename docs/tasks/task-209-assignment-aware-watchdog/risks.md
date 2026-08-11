# task-209 — Risks & Rollback

This task rewrites the fetch/commit core of a library consumed by every Go service.
Unlike a service-scoped change, a defect lands everywhere at once. This document is
the explicit risk and rollback story the PRD's FR-5 requires.

## Why the risk is real

`libs/atlas-kafka/consumer/manager.go` is 735 lines pinned by 1436 lines of
`manager_test.go`, plus `dwell_integration_test.go` (411), `idle_stuck_test.go` (233)
and `debug_test.go` (181). Migrating from `*kafka.Reader` to `ConsumerGroup` moves
work that kafka-go currently owns and has hardened into our tree:

| Responsibility | Today | After |
|---|---|---|
| Partition→reader lifecycle | `*kafka.Reader` | Ours (per generation) |
| Offset commit batching | `*kafka.Reader` | Ours (`Generation.CommitOffsets`) |
| Rebalance teardown/startup | `*kafka.Reader` | Ours (`ConsumerGroup.Next` loop) |
| Offset reset / `startOffset` | `*kafka.Reader` | Ours |
| Partition-count watch | `ReaderConfig.WatchPartitionChanges` | `ConsumerGroupConfig` equivalent (PRD §9.5) |
| Dial retry / backoff | `*kafka.Reader` | Ours |

## Ranked risks

| # | Risk | Impact | Mitigation |
|---|---|---|---|
| R1 | Offset-commit regression → message loss | **Critical.** Silent gameplay data loss (lost drops, lost EXP) | FR-1.3/FR-1.6. Never advance the cursor on commit failure. Dedicated tests for commit-failure and mid-generation shutdown. Loss is *worse* than duplication — on ambiguity, prefer redelivery |
| R2 | Duplicate delivery amplified during generation churn | High. task-208's idempotency guard covers atlas-inventory, atlas-cashshop and atlas-storage — but **only those three**; other consumers of moved-semantics topics are unprotected | Prefer landing 208 first so the compartment writes are guarded before delivery semantics move. No longer blocking (R5 resolved), but it is the cheapest available safety net |
| R3 | `maxInFlight` prefix-commit cursor mis-ported | High. Out-of-order commit → skipped messages under concurrency | FR-1.5. Default stays 1 (serial). Port the cursor per-partition and test at maxInFlight > 1 explicitly |
| R4 | Partition-count increase no longer detected | Medium. A future repartition would silently strand partitions | PRD §9.5 — confirm the `ConsumerGroupConfig` watch equivalent before merge |
| ~~R5~~ | ~~Merge conflict with task-208 Part 3~~ | **RESOLVED 2026-08-10** — 208 reverted its `libs/atlas-kafka` changes; verified that branch no longer touches the module. 209 is sole owner of `manager.go` | — |
| R6 | Latency regression vs `*kafka.Reader` | Medium | Gate on task-136 S1: p99 ≤ 22 ms, max ≤ 87 ms |
| R7 | New engine is *itself* wedge-prone in an unforeseen way | Medium | FR-5.1 env-var rollback; stage the rollout |
| R8 | **No interim mitigation in flight.** With 208's Part 3 reverted, churn continues at baseline (19–246 recreates/hour/service) until 209 ships — and 209 is a migration, not a patch | Medium/schedule. Live gameplay stalls of 4–9 s persist for the duration | Accepted deliberately (option C chosen over the detection patch). If the window proves too long, the FR-2 assignment check is the part that delivers the fix — it could ship on the legacy engine first under FR-5.1 |

## Rollback

`KAFKA_CONSUMER_ENGINE` (FR-5.1) selects the engine at process start:

- `consumergroup` (default) — new path
- `reader` — legacy `*kafka.Reader` path, retained for one release

Both engines use identical group IDs and offset-commit semantics (FR-5.3), so
rollback is **a pod restart with one env var flipped**. No topic, offset, consumer-group
or database migration is involved, and no state is written that the legacy path cannot
read. Rollback is therefore reversible in both directions and safe to exercise
mid-incident.

Rollback trigger conditions — any one is sufficient:

- Consumer lag on any hot topic sustained > 5 s for more than 2 minutes
- Any evidence of message loss (committed offset advancing past an unhandled message)
- Recreate rate materially *worse* than the 19–246/hour/service baseline
- Delivery p99 above the task-136 S1 ceiling

## Staged rollout

The library ships to ~45 services simultaneously, so stage by blast radius rather than
all at once:

1. **Ephemeral/PR environment** — full dwell suite + a manual play session.
2. **One low-traffic service** on `atlas-main` (e.g. `atlas-chalkboards`) with the new
   engine while everything else stays on `reader`. Both engines coexist in one cluster
   because group and offset semantics are identical.
3. **One hot-path service** — `atlas-monsters`, the service whose stall was traced.
4. **Remainder**, once 2–3 have run clean through several rebalance windows.

## Verification beyond unit tests

Unit tests cannot prove the rebalance behaviour. Required evidence before calling this
done:

- Task-136's dwell harness (testcontainers, single-broker) with a new S6 scenario:
  more group members than partitions, asserting `totalRecreates == 0`.
- A live `atlas-main` measurement after deploy: recreate count/hour per service, and
  an attack→drop-visible trace with no multi-second gap.

Baselines to compare against are recorded in the PRD §10 acceptance criteria.
