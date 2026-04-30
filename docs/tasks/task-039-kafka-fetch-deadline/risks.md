# Risks — task-039-kafka-fetch-deadline

## R1 — Reverses task-016's "no staleness heuristics" non-goal

Task-016's PRD §2 explicitly rejected staleness heuristics: *"No alert rules, no staleness-based 'silent consumer' detectors (explicitly rejected — there are legitimately idle topics in a test system with no user activity)."* The 2026-04-30 atlas-maps / atlas-monsters wedge incident is the case for revisiting that decision.

**Why this task does not violate the spirit of that constraint.** The rejected design was an *external watchdog* comparing `lastFetchAt` against wall-clock and force-killing readers heuristically. That approach has the cited problem: in a dev cluster, idle topics look identical to wedged topics from outside, so the watchdog fires false positives.

This task instead adds a per-call *fetch deadline*. Idle topics generate `DeadlineExceeded` and the loop ticks back with a fresh deadline — same reader, same partition assignment, no recreate. Wedged topics generate the same `DeadlineExceeded`, but because the wedge is in `FetchMessage` itself, no successful fetch occurs in between, and the consecutive-timeout counter escalates. The signal that distinguishes idle from wedged comes for free from the loop structure, not from a heuristic.

The 5-minute default and 3-strike escalation are conservative enough that the false-positive concern does not apply: a topic that legitimately produces one message every 14 minutes ticks twice and resets on the third attempt. No existing topic in the monorepo has cadence that slow.

**Mitigation.** No reversal of task-016's design choices is required. The new behavior is additive and addresses a failure mode task-016's PRD did not anticipate. Document the relationship in PRD §1 so future contributors understand why both decisions are correct in their respective contexts.

## R2 — kafka-go's FetchMessage cancellation behavior

The plan assumes `reader.FetchMessage(ctx)` returns promptly when `ctx` is cancelled or its deadline expires. segmentio/kafka-go honors context cancellation — that is the contract — but historically there have been edge cases where a long-poll already in flight to a broker did not unblock immediately on cancellation, particularly when the broker's TCP read was wedged.

**Worst case if the assumption fails.** A wedge that resists ctx cancellation would mean `FetchMessage` doesn't return on deadline either. The new code would still be no worse than the current code (which already blocks indefinitely on the same wedge), but the recovery path wouldn't fire as designed.

**Mitigation.** During implementation, add a goroutine-leak check around the wedge-recreate test: after `runFetchLoop` returns `errFetchWedged`, verify the `FetchMessage` goroutine actually exited. If it didn't (cancellation didn't unblock it), this entire approach has a hole and we need to escalate — likely by also force-closing the reader from the outer loop and accepting that the leaked goroutine eventually unblocks when the OS times out the underlying TCP connection. Flag for design re-discussion if observed.

## R3 — Reader-recreate storm on a flapping broker

If a broker is recovering from a partial outage and accepts connections but then drops them mid-fetch, the new code's escalation could combine with the existing outer backoff to produce more aggressive recreates than the old code. Concretely: old code recreates on the kafka-go transport error directly; new code potentially recreates on the deadline first, then again on each subsequent transport error.

**Mitigation.** This case is bounded by task-016's existing 10s outer-backoff cap. The cap also resets only on a successful fetch — which means a flapping broker never amortizes back to fast recreates, which is the correct behavior. No change required, but call it out in the design phase to confirm the existing cap is the right knob.

## R4 — Test fake behavior change cascading into existing tests

Task-016 introduced a `KafkaReader` test fake. If its `FetchMessage` returns immediately (e.g., from a pre-loaded queue) regardless of ctx, the new deadline never fires in test, and existing tests pass unchanged. If it blocks (e.g., on a channel read) and respects ctx, the new code's deadline fires during existing tests that don't intend to exercise the wedge path.

**Mitigation.** During implementation, audit every existing test in `manager_test.go`. Tests that don't exercise the wedge path get `SetFetchTimeout(1*time.Hour)` (or similar) added to their config so the deadline never fires. Tests that exercise the wedge path use `50*time.Millisecond`. This is a mechanical change; the audit cost is low.

## R5 — Operator misinterpretation of `consecutiveTimeouts > 0` on idle topics

An operator inspecting `/api/debug/consumers` during a quiet period may see `consecutiveTimeouts: 1` or `2` on legitimately idle topics and assume something is wrong. This is a documentation problem, not a behavior problem.

**Mitigation.** Add a one-paragraph explainer to the task-016 debug-route docs (if one exists) or to a new `docs/runbooks/kafka-consumer-debug.md` describing the three states:

- `consecutiveTimeouts == 0`, `lastFetchAt` recent → healthy and active.
- `consecutiveTimeouts == 0`, `lastFetchAt == "0001-01-01T00:00:00Z"`, `aliveSince` recent → just started, hasn't fetched yet (normal for first 5 minutes).
- `consecutiveTimeouts > 0`, `lastFetchAt == "0001-01-01..."`, `aliveSince` old → in progress of detecting a possible wedge; will escalate if it reaches `maxConsecutiveTimeouts`.
- `recreateCount > 0`, `lastError` contains `"consumer fetch wedged"` → wedge was detected and recovered.

This is follow-up documentation, not a blocker for the code change.

## R6 — Default fetch timeout collides with future low-cadence topics

The default `5 * time.Minute` is right for every topic in the monorepo today. If someone introduces a topic with cadence slower than 5 minutes (e.g., a daily aggregator event) and forgets to override, that consumer would tick continuously and burn one cancelled syscall every 5 minutes — harmless but noisy.

**Mitigation.** None required at this scale. If/when low-cadence topics arrive, override at the consumer registration site. Documenting `SetFetchTimeout` in the task-016 debug-route docs (per R5) covers the discoverability aspect.
