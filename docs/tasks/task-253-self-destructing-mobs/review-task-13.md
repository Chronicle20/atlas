# Review: Task 13 — pin `ActorId = 0` (no-killer) behaviour in atlas-monster-death

Commit range: `79b5f3bbf..77fc9bc64` (single commit `77fc9bc64`)
Brief: `.superpowers/sdd/plan/task-13-brief.md`
Report: `.superpowers/sdd/plan/task-13-report.md`

## Scope confirmed

`git diff --stat 79b5f3bbf..77fc9bc64` shows exactly one file:

```
.../atlas.com/monster/monster/actor_zero_test.go | 221 +++++++++++++++++++++
1 file changed, 221 insertions(+)
```

`git diff --name-only` confirms the same single path. No production file
under `services/atlas-monster-death` (or anywhere else) changed. The commit
is genuinely test-only, matching the brief and the report's claim.

## Findings

### 1. Test-only, no production change — PASS

`git diff --name-only 79b5f3bbf..77fc9bc64` → only
`services/atlas-monster-death/atlas.com/monster/monster/actor_zero_test.go`.
Verified directly, not taken on the report's word.

### 2. `TestFilterByQuestStateExcludesQuestDropsForNoKiller` exercises the real production path — PASS

`actor_zero_test.go:96` constructs `&ProcessorImpl{l: l, ctx: ctx}` and calls
`.filterByQuestState(0, tc.drops)` — the actual unexported method on the real
processor type, not a reimplementation. Cross-checked against
`monster/processor.go:90-124`: `filterByQuestState` only calls the quest
service when at least one drop has `QuestId() != 0` (`processor.go:93-101`),
excludes quest drops the character hasn't started, and on a quest-service
error clears `startedQuests` and excludes every quest drop
(`processor.go:107-112`). The three subtests (500 error, empty JSON:API
page, and no server contact when no drop needs it — enforced by
`t.Fatal` inside the handler at `actor_zero_test.go:87`) map 1:1 onto that
logic. This test would fail if `filterByQuestState` stopped excluding
undropped quest items or started hitting the quest server unconditionally.

### 3. `TestCreateDropsWithNoKillerSpawnsUnownedDrop` exercises the real spawn path — PASS

Calls `NewProcessor(l, ctx).CreateDrops(f, 500, 8500003, 10, 20, 0)`
(`actor_zero_test.go:150`) — the real constructor and the real
`CreateDrops` (`monster/processor.go:41-71`), which fetches drops over REST
(`DROPS_INFORMATION_SERVICE_URL`), filters by quest state, resolves rates
(`RATES_SERVICE_URL` 404 → `rates.Default()`, confirmed at
`rates/provider.go:22-31`), resolves party (`PARTIES_SERVICE_URL` 404 →
`party.GetByMemberId` errors, `ownerPartyId` stays the zero value —
confirmed at `party/processor.go:30-33` and `monster/processor.go:61-64`),
and then calls `drop.NewProcessor(...).Create(...)`, which for a
non-zero-item drop reaches `SpawnItem` → `SpawnDrop` →
`producer.ProviderImpl(...)(EnvCommandTopic)(cp)`
(`monster/drop/processor.go:75-91`). `EnvCommandTopic` is
`"COMMAND_TOPIC_DROP"` (`monster/drop/kafka.go:13`), the same topic the
test reads via `emitted.Messages(drop.EnvCommandTopic)`
(`actor_zero_test.go:159`). The captured message is unmarshalled and
asserted `ownerId == 0`, `ownerPartyId == 0`, `itemId == 1000` — genuinely
derived from the production call chain, not from a self-constructed
fixture. Ran the test directly: `PASS`.

The chance-999999 drop reliably succeeds (`evaluateSuccess`,
`monster/processor.go:80-84`: `rand.Int31n(999999) < 999999*1.0` is always
true), so the test is deterministic, not merely usually-passing.

### 4. `TestDistributeExperienceWithEmptyEntriesIsNoOp` exercises the real no-op path — PASS

Calls the real `DistributeExperience(f, 8500003, nil)`
(`monster/processor.go:126-141`), which builds a `DamageDistributionModel`
via `produceDistribution` with `damageEntries == nil` → `soloDistribution`
stays empty → `d.Solo()` is empty → the `for k, v := range d.Solo()` loop
body (which is the only place `character.NewProcessor(...).GetById` and
therefore any character-service HTTP call is made) never executes. The
character stub server calls `t.Fatal` if hit at all
(`actor_zero_test.go:213`), so the assertion is on absence of a call, not a
constructed fixture. `DistributeExperience` unconditionally returns `nil`
(`monster/processor.go:140`) regardless of loop execution, so the `err ==
nil` half of the assertion is trivially true either way — the load-bearing
assertion is the "character server never called" one, and it is correctly
enforced via `t.Fatal` in the handler rather than a post-hoc request
counter.

### 5. `TestCalculateExperienceStandardDeviationThresholdEmptyIsNaN` — known defect pinned, and documented as pinned, not silently blessed — PASS

`calculateExperienceStandardDeviationThreshold([]float64{}, 0)`
(`monster/processor.go:207-220`) computes `0.0 / float64(0)` twice (NaN via
0/0), matching the test's `math.IsNaN` assertion. The test's doc comment
(`actor_zero_test.go:184-190`) explicitly labels this "a known adjacent
defect (design §8.2)" and states it is "pinned here, not changed" with the
harmlessness rationale (the only caller then iterates an empty map). This
satisfies the brief's requirement (task-13-brief.md:119-124) that the
defect be pinned with an explicit comment, not silently treated as
intended behaviour. Not a blocking finding — the brief explicitly directs
pinning this defect rather than fixing it in this task.

### 6. httptest servers and `t.Setenv` scoping — PASS, no cross-test leakage

Every httptest server in every test/subtest is `defer srv.Close()`'d
immediately after creation, and every `t.Setenv` call is scoped to the
enclosing `t.Run` closure or top-level test function (Go's `t.Setenv`
restores the prior value automatically at test/subtest cleanup). The only
package-level shared state is `emitted *producertest.Capture`, installed
once in `TestMain` (`actor_zero_test.go:31-35`); it is used by exactly one
test (`TestCreateDropsWithNoKillerSpawnsUnownedDrop`), which calls
`emitted.Reset()` before invoking `CreateDrops` (`actor_zero_test.go:149`).
Checked the rest of package `monster`'s test files
(`builder_test.go`, `characterization_test.go`, `processor_test.go`) —
none reference `producer.` or the capturing producer, so installing a
capturing producer package-wide cannot desynchronize another test's own
producer expectations. No other `TestMain` exists in the package (the
brief's premise), so there is no conflict.

### 7. Tests actually run and pass — PASS

Ran directly (not taken on the report's word):

```
go test ./monster/ -run 'NoKiller|EmptyEntries|EmptyIsNaN|FilterByQuestState' -v
```
→ all 4 tests / 7 subtests `PASS`.

```
go build ./... && go test ./...
```
→ full module suite `PASS`, no other package affected.

## Not evaluable

None. The unit is small and self-contained (one new test file, no
production change), and every assertion could be traced to the real
production code path it claims to exercise within this review's scope.

## Report accuracy

The implementer's report (`task-13-report.md`) accurately describes the
change, the test rationale, and the "no defect required a fix" outcome.
Independent verification confirms every claim in it: the file diff, the
production code paths each test exercises, the NaN defect and its pinning
comment, and both test-command results.

## Verdict rationale

All four tests exercise genuine production code paths (not
self-constructed fixtures), would fail under the behavioural regressions
they claim to guard against, are properly isolated from each other, and
the one known defect pinned is explicitly documented as such per the
brief's own instruction. No blocking issues found.
