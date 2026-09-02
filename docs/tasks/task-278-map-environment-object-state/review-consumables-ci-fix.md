# Review: 3cd6d12ad — repair potion-lock test build and Kafka fixtures

**Branch:** task-278-map-environment-object-state
**Commit under review:** 3cd6d12ad3cfe53e17634407885457d94bf81d8d
**Scope note:** This commit's content (STOP_PORTION potion-lock test fixtures,
task-280 FRs) has nothing to do with task-278 ("map environment object
state"). Per the task brief supplied to this review, that is expected: PR
#1566 (task-278) was failing CI because origin/main itself was broken by PR
#1557, and this commit was landed on the task-278 branch specifically to
repair the inherited breakage so task-278's own CI can go green. Confirmed via
`git log`: 3cd6d12ad sits directly on top of `591a0e103 Merge ... origin/main`,
which pulled in `869cd523d feat(consumables): enforce STOP_PORTION consume
lock server-side (#1557)`. Scope for this review is exactly the two files this
commit touches; no other task-278 work was inspected.

## Files touched

- `services/atlas-consumables/atlas.com/consumables/consumable/processor_potion_lock_test.go` (+4/-4)
- `services/atlas-consumables/atlas.com/consumables/consumable/testmain_test.go` (+47/-1)

## Verdict

APPROVED

## What I checked and how

1. Read the full diff (`git show 3cd6d12ad`) and both files in full.
2. Read the production code path the tests exercise: `RequestItemConsume`,
   `resolvePotionLocked`, `rejectPotionLocked`, `usesStandardConsumer` in
   `consumable/processor.go`.
3. Read `consumer.Manager` (`GetManager`, `AddConsumer`, `RegisterHandler`)
   and the reader-engine `start`/`startReaderEngine` lifecycle in the vendored
   `atlas-kafka` module used by this go.mod
   (`v0.0.0-20260829214709-9e47d56c0727`, resolved from local module cache)
   to verify the idle-reader/TestMain wiring is genuinely leak-free rather
   than just "looks fine."
4. Ran the actual gates this commit exists to fix, in this worktree:
   - `go build ./...` — clean.
   - `go vet ./...` — clean.
   - `go test ./consumable/... -run "TestRequestItemConsume|TestResolvePotionLocked" -v` — all pass, and I inspected the log output line-by-line against the sub-test's assertions.
   - `go test ./consumable/... -race` — clean, 22.3s, no leaked-goroutine/data-race report.
   - `go test ./...` (whole atlas-consumables module) — all packages pass.
   - `tools/lint.sh --check --go services/atlas-consumables` — `0 issues.` / `lint.sh: OK`.
   - `./tools/go-analyzer-guards.sh` scoped to `services/atlas-consumables/atlas.com/consumables` — `go-analyzer-guards: PASS` (rediskeyguard/outboxguard/goroutineguard/buffdurationguard/scopeguard/topicguard all ran clean).

## Finding 1 — Messages() cast (processor_potion_lock_test.go:89)

`emitted.Messages(string(consumable.EnvEventTopic))` matches
`producertest.Capture.Messages(topicName string)`'s real signature (found in
the `atlas-kafka` module's `producer/producertest` package) and matches the
established convention in the same package (`processor_catch_test.go:102,108,114`
all cast `topic.Token` to `string` at the call site). PASS — this is a
mechanical, correct fix, not a new pattern.

## Finding 2 — setEnv(compartmentmsg.EnvEventTopicStatus) (testmain_test.go:77)

`RequestItemConsume` (`processor.go:328`) resolves
`EVENT_TOPIC_COMPARTMENT_STATUS` via `topic.EnvProvider` unconditionally, at
the very top of the function, before the potion-lock gate at line 345 and
before any classification branch. Without the env var set, every call in the
suite returns that resolution error before ever reaching
`resolvePotionLocked`/`RequestReserve` — meaning every assertion the four
potion-lock tests make would have been trivially true for the wrong reason
(an early env error, not the actual gate). Setting the env var in `TestMain`
is necessary and sufficient to let control flow actually reach the
task-280 logic. Confirmed empirically: with the fix applied, the standard
`-v` test run I captured shows the reserve-mock/buff-mock branches firing (log
lines "Created kafka writer...", the buff-read Warn, "Enrichment degraded"),
which would not appear if the function had bailed at line 328-331.

## Finding 3 — registerCompartmentStatusConsumer / idleReader (testmain_test.go:33-67, 85)

Traced this against `consumer.Manager`'s real implementation rather than
trusting the docstring:

- `RegisterHandler` (`manager.go:206`) hard-errors `"no consumer found for
  topic"` unless `m.consumers[topic]` already has an entry. `AddConsumer`
  (`manager.go:136`) inserts that map entry **synchronously**, under `m.mu`,
  before it spawns the fetch-loop goroutine via `routine.Go`. So calling
  `AddConsumer(...)(config)` once in `TestMain`, before `m.Run()`, is
  sufficient for every later `RegisterHandler` call inside a test to succeed
  — there's no ordering race between registration and the first test that
  needs it.
- Only 3 of the 4 potion-lock tests actually need this: the locked-rejects
  path (`RequestItemConsume_LockedInScopeRejects`) returns at
  `processor.go:346`, before `RegisterHandler` is ever called
  (`processor.go:396`), so it doesn't depend on the consumer being
  registered. The out-of-scope, unlocked, and fail-open tests all reach
  `RegisterHandler`. This matches the commit message's claim ("so the
  unlocked/fail-open cases reach RequestReserve") and I confirmed it by
  tracing the actual control flow, not just running the tests.
- Leak-safety: `ctx, cancel := context.WithCancel(...); cancel()` runs before
  `AddConsumer` is invoked, so `c.engine == EngineReader`'s
  `startReaderEngine` (`engine_reader.go:34`) does `wg.Add(1); defer
  wg.Done()` and its very first statement is `if ctx.Err() != nil { ...
  return }` — the goroutine returns immediately without ever calling `c.rp`
  (i.e. never touches `idleReader.FetchMessage`, which itself blocks on
  `<-ctx.Done()` and would also return instantly since ctx is already
  cancelled). `wg.Wait()` in `TestMain` after `m.Run()` therefore drains
  cleanly. The `go test -race` run (22.3s, all packages) reported no
  goroutine leak and no race — empirical confirmation, not just code reading.
- `sync.Once` note (flagged explicitly in the review brief): `GetManager`'s
  configurators only apply on the **first** call in the process for the
  whole `consumable` test binary. I checked for any other `consumer.GetManager()`
  call earlier in package-load or `TestMain` order: the only other call is
  inside `RequestItemConsume` itself (`processor.go:396`), which fires from
  inside a test function, always after `TestMain`'s
  `registerCompartmentStatusConsumer` has already called `GetManager` at
  package-test-startup. There is no init()-time or other TestMain-preceding
  `consumer.GetManager()` call in this package that would grab the `Once`
  first with a different (default, broker-dialing) `rp`. Verified by `grep -rn
  "consumer.GetManager"` across the `consumable` package — the only call
  sites are in `processor.go` (production code path) and the new
  `testmain_test.go` helper.
- `t.Fatalf`/env-var failure path: not applicable here — `AddConsumer` never
  returns an error to check; the only failure mode would be a topic-name
  collision with a config already registered under the same topic string
  (`manager.go:146` — silently returns, logging `Infof`, not fatal), which
  cannot occur since this is the only consumer this package registers for
  `EVENT_TOPIC_COMPARTMENT_STATUS`.

## Finding 4 — comment correction (processor_potion_lock_test.go:180-182)

The old comment claimed the extraneous Warn was caused by "unset
EVENT_TOPIC_COMPARTMENT_STATUS env var in this test environment" — which was
literally true pre-fix (that's the bug this commit fixes) but would have gone
stale and misleading the moment the env var got set. The new wording
("consumer registration, topic resolution") is generic enough to not need
another edit if the log surface shifts again, and the filter logic below it
(`strings.Contains(e.Message, "Unable to read buffs")`) is unchanged and
still isolates the correct Warn — confirmed by the passing
`TestRequestItemConsume_BuffReadErrorFailsOpen` run showing exactly one
`warnEntries` match containing "555". Non-blocking, but worth noting the
comment update is honest self-correction, not just rewording.

## Convention checks (CLAUDE.md / backend-dev-guidelines)

- No test-only constructor added; `lockedBuffsFixture` (unchanged by this
  commit) already uses the package's own exported `buff.NewBuff`. This
  commit adds no new constructors of any kind.
- `idleReader` is a legitimate hand-written test double implementing the
  library's `consumer.KafkaReader` interface (`FetchMessage`,
  `CommitMessages`, `Close`) — not a production-facing type, scoped to
  `_test.go`. No convention violation.
- The `sync.WaitGroup` drain-before-exit pattern in `TestMain` is the correct
  shape for a package that registers a real `consumer.Manager` consumer in
  tests; nothing here violates the "no bare `go` outside
  `libs/atlas-routine`" rule — the goroutine is spawned by the library's own
  `routine.Go`, inside `AddConsumer`, not by test code directly. Confirmed by
  `goroutineguard` passing clean in the analyzer-guards run above.
- No CRLF/LF concerns — diff is a plain unified diff of Go source, in-place
  edits only.

## Not evaluable

- I did not re-run the exact GitHub Actions workflow (`pr-validation.yml`)
  end-to-end; I ran the equivalent local commands (`tools/lint.sh --check`,
  `tools/go-analyzer-guards.sh`, `go test ./...`, `-race`) that back those
  three named CI jobs, scoped to `services/atlas-consumables`. If CI's
  toolchain/version pins differ from what's on this machine that would not be
  caught here — but `tools/toolchain.versions` is what both paths read from,
  so this is a low-risk gap, not a live unknown.
- I did not review the rest of PR #1557's diff (the STOP_PORTION feature
  itself) beyond the two functions this fix commit's tests exercise
  (`RequestItemConsume`, `resolvePotionLocked`) — that is PR #1557's own
  surface, out of scope for a review of the CI-repair commit.

## Summary

All three named CI failures (Go Analyzer Guards, Lint & Format Guard (Go),
Test Service - atlas-consumables) are fixed by this commit for a real reason,
not by weakening the fixtures: the topic-cast fix restores compilation, the
`setEnv` fix restores the tests' ability to actually reach the potion-lock
gate instead of failing early on an unrelated env lookup, and the idle
Kafka-reader registration restores the tests' ability to reach
`RequestReserve` instead of hard-erroring on `RegisterHandler`. Traced by hand
against the real `consumer.Manager` implementation and confirmed empirically
with `-race`, full-module `go test`, `lint.sh --check`, and
`go-analyzer-guards.sh` — all green. No fixture was widened, no assertion was
weakened, and the locked/out-of-scope/unlocked/fail-open FRs (task-280 FR-2,
FR-4, FR-5) are exercised for real, not vacuously.
