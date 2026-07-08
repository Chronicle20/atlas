# task-136 findings — Kafka consumer dwell root cause

## Reproduction environment

Harness: `libs/atlas-kafka/consumer/dwell_integration_test.go` (build tag
`integration`), single-broker testcontainers Kafka (`confluentinc/cp-kafka:7.6.0`),
one group with 15 idle topics + 1 active topic (models atlas-saga-orchestrator's
fan-out) plus a 4-consumer second group. Latency = publish→handler, stamped
in-message.

## Pre-fix baseline (commit 30ac51339a)

| Scenario | Result | p99 | max | total recreates | notes |
|---|---|---|---|---|---|
| S1 steady state | PASS | 22.042368ms | 87.094377ms | 0 | ticks never fire at 5m defaults |
| S2 idle-tick churn | FAIL (expected) | 13.431397ms | 13.431397ms | 75 | wedge cadence compressed to 2s×2 |
| S4 tick control | FAIL (expected) | 14.149919ms | 14.149919ms | 4 | |
| S5 fetch rate 50ms vs 10s | 648 vs 4 attempts/30s | — | — | — | |

Phase attribution (S2 snapshot dump, active-topic consumer `dwell-active`,
`Topic:dwell.active`):
- TimeToFirstFetch: 7.60173024s (join/assignment cost per recreate; up from
  5.187035634s in the S1 baseline snapshot for the same consumer)
- TotalBackoff: 0s (the active consumer itself never recreated —
  `RecreateCount:0` — so it carries no backoff; the 75 recreates and their
  backoff are on the idle consumers. Representative idle snapshot
  `dwell-dwell.idle.0`, `Topic:dwell.idle.0`: `RecreateCount:5`,
  `ConsecutiveTimeouts:1`, `TotalBackoff:15.5s`, and this pattern repeats
  identically across all 15 idle-topic consumers in the dump.)
- MaxFetchDuration: 7.60171275s (active consumer). Idle consumer
  `dwell-dwell.idle.0` shows `MaxFetchDuration:2.001073099s` per wedged fetch
  attempt (the 2s fetch-wait ceiling being hit repeatedly before recreate).
- MaxHandlerDuration: 82.719µs (active consumer; H4 check — negligible as
  expected)

## Hypothesis verdicts

- **H1 (wedge-recreate churn → group-wide rebalance storms):** Confirmed.
  S2 measured `totalRecreates=75` across the 15 idle-topic consumers (each
  showing `RecreateCount:5`, `ConsecutiveTimeouts:1`, `TotalBackoff:15.5s` in
  its snapshot — e.g. `dwell-dwell.idle.0`, `dwell-dwell.idle.1`,
  `dwell-dwell.idle.3`, all identical). This is a hard FAIL on the
  `totalRecreates == 0` assertion at `dwell_integration_test.go:252`
  ("S2: idle deadline ticks must not recreate readers (design §3-A)").
  Attribution: the active-topic consumer's `TimeToFirstFetch` grew from
  5.187035634s (S1, no churn) to 7.60173024s (S2, churn active) for the same
  consumer/topic, evidencing group-wide join-cost inflation coincident with
  the idle-consumer recreate storm, even though the active consumer never
  recreated itself (`RecreateCount:0` in both snapshots).
- **H2 (50ms MaxWait idle-spin):** S5 measured 648 vs 4 fetch attempts per
  idle reader per 30s (`dwell_integration_test.go:374`). Extrapolated to
  ~481 live partitions: 648/30s × 481 ≈ 10,390/s vs 4/30s × 481 ≈ 64/s — a
  ~162x amplification in idle-poll fetch-request volume from the 50ms
  default `maxWait` versus a 10s `maxWait`.
- **H3 (deadline drops group session — refuted in source):** S4 control
  (ticks alone, no full churn cadence) still produced `recreates=4`
  (`dwell_integration_test.go:340`), failing the "ticks alone must not
  recreate" assertion at `dwell_integration_test.go:342`. This shows the
  tick-driven recreate path itself fires even in the reduced S4 scenario —
  consistent with H1's recreate mechanism as the actual driver. H3's
  specific claim (deadline directly drops the *group* session, as opposed
  to the per-reader recreate path) remains refuted at the source level;
  S4 does not measure a group-session drop, only that ticks alone are
  sufficient to trigger reader recreation.
- **H4 (head-of-line blocking):** handler dispatch time
  `MaxHandlerDuration:82.719µs` (S2, active consumer) — negligible relative
  to the multi-second `TimeToFirstFetch`/`MaxFetchDuration` figures above.
  Refuted: handler dispatch is not a contributor to the measured dwell.

## Client-side vs broker-side split

All dwell reproduced in this harness is client-side by construction: the
harness broker is a single, unloaded testcontainers Kafka instance
(`confluentinc/cp-kafka:7.6.0`) with no contention or load applied to it.
The measured dwell (S2 `recreates=75`, per-idle-consumer
`TotalBackoff:15.5s`, active-consumer `TimeToFirstFetch` growing from
~5.19s to ~7.60s) is driven entirely by the reader-recreate churn on the
client side (idle-topic consumers self-wedging on the deadline tick and
recreating their readers), not by broker-side fetch latency — the broker
had no reason to be slow. This confirms the dwell mechanism under
investigation is a client-side reader-recreate defect, not a broker
capacity/latency issue.

## Post-fix results

(Completed in the post-fix run task.)

## Config default changes & rationale

(Completed in the post-fix run task.)

## Follow-up decision (design §7 gate)

(Completed in the post-fix run task.)
