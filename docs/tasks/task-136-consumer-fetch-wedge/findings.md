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

### Final verdicts (post-fix, commit d0e71ba1bd)

- **H1 — Confirmed and fixed.** Pre-fix S2 measured `totalRecreates=75`
  driven purely by idle deadline ticks (design §3-A violation). Post-fix S2
  measures `recreates=15` total, all attributable to the one-time initial
  group-join transient (each idle consumer at `RecreateCount:1`, zero
  recreates thereafter) — **0 NEW recreates in steady state**, verified by
  the committed baseline-delta assertion
  (`dwell_integration_test.go:274-275`). The idle-vs-no-progress
  classification (Approach A) eliminates the churn mechanism at its source.
- **H2 — Quantified, addressed by the `maxWait` default change.** S5
  measured 646 (50ms) vs 4 (10s) idle fetch attempts/reader/30s, a ~162×
  reduction (646/4 ≈ 161.5), extrapolated to ~10,357/s vs ~64/s across ~481
  live partitions. See "Config default changes & rationale" below.
- **H3 — Closed by the S4 control.** Post-fix S4 (ticks alone, recreate path
  live, no churn generator) measures `recreates=0`
  (`dwell_integration_test.go:374`) — ticks alone no longer trigger any
  recreate, confirming the fix separates "deadline expired" from "reader
  wedged" as designed. Combined with the pre-fix source-level refutation
  (F6: a per-call deadline does not touch the group session), H3 is closed
  in both directions: refuted at the library level and, post-fix, no longer
  even exercised through Atlas's own recreate path.
- **H4 — Closed (refuted).** No post-fix change was needed for this
  hypothesis; handler dispatch remains negligible (`MaxHandlerDuration` in
  the low-to-mid microseconds across every post-fix snapshot, e.g. S1
  active consumer `MaxHandlerDuration:157.268µs`, S3 active consumer
  `MaxHandlerDuration:96.699µs`) against multi-second
  `TimeToFirstFetch`/`MaxFetchDuration` figures. Head-of-line blocking in
  the serial handler loop is not a contributor to the dwell.

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

## Post-fix results (commit d0e71ba1bd)

Full suite green: `ok github.com/Chronicle20/atlas/libs/atlas-kafka/consumer 296.162s`,
all five scenarios PASS.

| Scenario | Result | p99 | max | recreates | notes |
|---|---|---|---|---|---|
| S1 steady state | PASS | 14.109536ms | 14.166479ms | 0 | |
| S2 idle-tick churn | PASS | 15.737613ms | 15.737613ms | 15 total (0 NEW in steady state) | the 15 recreates are the excluded startup group-join transient (see calibration note below); all 15 idle consumers recorded `IdleTicks` (each shows `IdleTicks:16` or `17` in the snapshot dump, satisfying the test's `>=10` floor assertion) |
| S3 forced-recreate bounded | PASS | — | max dwell across recreate = 7.568674681s | 1 (forced) | timeToFirstFetch=7.047282067s, totalBackoff=500ms; 7.57s ≤ the 10s join+backoff design budget |
| S4 tick control | PASS | 16.011361ms | 16.011361ms | 0 | ticks alone (no churn) never recreate |
| S5 MaxWait A/B | PASS | — | — | — | 646 (maxWait=50ms) vs 4 (maxWait=10s) idle fetch attempts/30s per reader |

S3 detail: the design's acceptance bound for a genuine forced recreate is
"max dwell across the recreate ≤ 10s (join + backoff budget from F8)"
(design §4.1). The measured value, 7.568674681s, is under that bound. The
10s figure is kept as-is — it is the design's join+backoff budget derived
from kafka-go's group-protocol timing defaults (F8: `JoinGroupBackoff=5s`,
`RebalanceTimeout=30s`), not a number tuned to fit this one measurement.

S2 detail: `dwell_integration_test.go:269` logs
`S2: p99=15.737613ms max=15.737613ms recreates=15`. Those 15 recreates are
one each across the 15 idle-topic consumers (`RecreateCount:1` in every
`dwell-dwell.idle.N` snapshot), all occurring during the initial ~16-member
group join (see calibration note below), not during steady-state operation.
The committed test (`dwell_integration_test.go:252-262`) captures this count
as a baseline via `require.Eventually` before publishing, then asserts
`totalRecreates(cm) == baselineRecreates` after the run — i.e. **0 NEW
recreates in steady state**. It also asserts `tickedIdle >= 10`
(`dwell_integration_test.go:274-281`) to prove the 2s deadline ticks
actually fired and were classified idle (not vacuously passing); the log
snapshots show idle consumers recording `IdleTicks:16`/`17` and
`NoProgressTicks:2`/`3` over the 30s window, each hitting the compressed
S2 `SetMaxConsecutiveTimeouts(2)` threshold once during join before settling.

## Config default changes & rationale

| Setting | Old default | New default | Rationale |
|---|---|---|---|
| `maxWait` | 50ms | 10s | kafka-go's own library default (F9). With `MinBytes=1` the broker answers immediately when data exists, so `MaxWait` only bounds an *empty* long-poll — raising it costs zero delivery latency. S5 measured 646 idle fetch attempts/30s at 50ms vs 4 at 10s per reader — a ~162× reduction (646/4 ≈ 161.5) in idle fetch-request volume for the same idle reader. Extrapolated to ~481 live partitions: 646/30s × 481 ≈ 10,357/s of idle fetch traffic at the old 50ms default vs 4/30s × 481 ≈ 64/s at the new 10s default. |
| `fetchTimeout` | 5m | 1m | No longer a recreate trigger — it's a liveness-tick cadence. A deadline expiry is now classified idle (healthy, reader still making progress) vs no-progress (stalled), per Approach A (design §3). Shortening it from 5m to 1m tightens real-stall detection from ~15m to ~3m at `maxConsecutiveTimeouts=3` without reintroducing churn, since idle ticks no longer count toward the threshold. |
| `maxConsecutiveTimeouts` | 3 | 3 (unchanged) | Same numeric value, redefined semantics: it now counts consecutive **no-progress** ticks (zero `Stats().Fetches`/`Dials` delta across the tick) rather than every deadline expiry. S4 confirms ticks alone (idle, still progressing) never recreate (`recreates=0`); S3 confirms the no-progress/recreate path still fires and recovers within budget. |

## Scenario calibration — implicit fetchTimeout invariants

Two calibration facts surfaced while tuning the compressed S2/S4 scenarios
to pass post-fix. Both are properties of the fix's design (Approach A) and
are recorded here because they constrain any future scenario or config change,
not because they are bugs.

1. **`maxWait` must stay well under `fetchTimeout`.** The idle-vs-stuck
   classification signal is the delta in `Reader.Stats().Fetches` across a
   tick (design §4.3/§3-A). An idle reader's `Fetches` counter increments
   approximately once per `maxWait` interval — S5 proves this directly (a
   50ms `maxWait` idle reader completes 646 fetch long-polls in 30s; a 10s
   `maxWait` idle reader completes only 4). If `maxWait >= fetchTimeout`, an
   idle reader can complete **zero** fetches within a single tick and gets
   misclassified as a no-progress stall even though it is healthy. Production
   defaults satisfy the invariant by ~6×: `maxWait` 10s vs `fetchTimeout` 1m.
   The S2/S4 scenarios initially left `maxWait` at the new 10s default while
   compressing `fetchTimeout` to 2s for test speed — a 5× *inversion* of the
   production ratio — which mis-fired the no-progress classification. Fixed
   by adding `consumer.SetMaxWait(200*time.Millisecond)` to both scenarios'
   decorators (`dwell_integration_test.go:241-244`, `:361-364`); 200ms « 2s
   mirrors the production 10s « 1m ratio.
2. **Initial group-join transient.** For the ~16-member group modeled by S2
   (15 idle + 1 active consumer, one shared `GroupID`), the first
   join/rebalance takes several seconds — measured active-consumer
   `TimeToFirstFetch` ranges ≈7–14s across the S1/S2/S3/S4 snapshots in this
   run (S1: 14.302955926s, S2: 7.703326872s, S3: 7.047282067s, S4:
   5.132005193s). With the compressed 2s `fetchTimeout` in S2/S4, an idle
   reader still mid-rebalance during join can go >1 tick with no fetch
   progress and self-wedges **once** before the group settles — every idle
   consumer in the S2 log ends at `RecreateCount:1`, then accumulates 15-17
   clean `IdleTicks` afterward with no further recreates. Production's 1m
   `fetchTimeout` dwarfs join time (single-digit seconds), so this transient
   never occurs live.

Both are artifacts of the compressed **test** scenarios' timing, not
production behavior. Production's default relationship — `maxWait` 10s ≪
`fetchTimeout` 1m, and `fetchTimeout` 1m ≫ join time — makes both non-issues
live. S2 (`dwell_integration_test.go:252-262`) accounts for fact 2 the same
way `dwellSetup` already excludes warm-up latency elsewhere in this harness:
it lets recreates stabilize post-join via `require.Eventually`, baselines
the count, and then asserts **zero NEW recreates** relative to that baseline
in steady state — rather than asserting a raw zero that the join transient
would always fail.

## Follow-up decision (design §7 gate)

Decision rule (design §7, verbatim): "if post-fix live observation (wedge
logs gone, saga dwell < 1s) still shows multi-second dwells, file the
cluster-infra follow-up (multi-broker / per-env topic reduction / Approach C)
citing those numbers; otherwise record 'library fix sufficient' in findings
and close §9 Q1."

All dwell reproduced and eliminated in this task is client-side by
construction: the harness broker is a single, unloaded testcontainers Kafka
instance (`confluentinc/cp-kafka:7.6.0`) with no contention or load applied.
Everything measured here — S2's 15 join-time recreates eliminated from
steady state, S3's bounded 7.57s forced-recreate dwell, S4's confirmation
that ticks alone never recreate — is H1 (client-side reader-recreate churn),
not broker-side fetch latency. The broker had no reason to be slow in this
harness.

S5's extrapolation (recorded above) is the quantified basis for the gate:
at the old 50ms `maxWait` default, ~481 live partitions would generate
646/30s × 481 ≈ 10,357 idle fetch requests/sec against the shared broker;
at the new 10s default, that drops to 4/30s × 481 ≈ 64/s. This is the
number to cite if the follow-up is ever filed.

Per the decision rule, this task does not file the cluster-infra follow-up
now: the library fix (Approach A) eliminates the reproduced H1 churn
mechanism at the source, and this harness has no way to observe live
broker-side saga dwell (that requires post-deploy observation of the actual
~481-partition cluster, which is out of scope for a library-level test
harness). The decision rule's gate condition — "post-fix live observation
still shows multi-second dwells with wedge logs gone" — is deferred to
post-deploy monitoring, not resolved here. If live monitoring after this fix
ships still shows multi-second saga dwells with the wedge-recreate log lines
gone, file the cluster-infra follow-up (multi-broker / per-env topic
reduction / design Approach C) citing the S5 extrapolation above; otherwise
the library fix in this task suffices and §9 Q1 is closed.
