# Review: Task 12 — Cache population and `RING_PURCHASED` invalidation

Range: `9f65d8ac0..efd355942` (1 commit, 8 files).
Module: `services/atlas-channel/atlas.com/channel`.

## Summary

The invalidation half of this task (`RING_PURCHASED` → buyer + partner) is solid,
tenant-correct, and genuinely tested (RED/GREEN evidence quoted in the report,
independently reproduced via fault injection below). The Controller addendum
items A1–A3 all landed correctly. The population wiring at
`kafka/consumer/session/consumer.go` (the judgment call) is placed correctly
relative to the first encode of the entering character's own ring data.

However, the PRD's FR-4 has **two** invalidation triggers — `RING_PURCHASED`
*and* "character map/channel transfer" — and only the first is implemented.
`plan.md`'s own Task 12 Step 3 lists "Map/channel transfer drops the entry" as
part of what to implement. Nothing in this diff does that, the report does not
mention it as a deliberate deferral (unlike the Populate-callsite decision,
which *is* flagged), and no test exercises it. This is a silently dropped
requirement, not a judgment call.

## Priority 1 — the population call site

**Verified by reading, not just trusted:** `kafka/consumer/session/consumer.go`
`processStateReturn` (session/consumer.go:207-236) calls
`ring.NewProcessor(l, ctx).Populate(c.Id())` at line ~217, synchronously
*before* `session.Announce(...)(writer.SetFieldBody(s.ChannelId(), c, bl))(s)`
at line ~236, which is the call that produces `cd.Rings` via
`BuildCharacterData` (`socket/writer/character_data.go:60`). `SpawnForSelf`
(spawns other characters *to* the entering player) runs after that,
synchronously. So for the entering character's own record block (site A) the
ordering is correct: population always lands before the first encode that
needs it, for both login and channel-enter (this consumer is fed by
`account_session_status_event`, the same session-bootstrap path independently
documented elsewhere in this codebase — see `session/processor.go:412-421`'s
`Destroy` comments — as covering "logout, disconnect, timeout, and channel
change").

I also traced the reverse direction — the entering character's own halves
being spawned *to other players already on the map* (site B,
`kafka/consumer/map/consumer.go:393-424` `enterMap`, which calls
`spawnCharacterForSession(self, g, true)` → `writer.CharacterSpawnBody(self,
...)` → `GetRingSet(self.Id(), ...)`). `enterMap` is driven by a separate Kafka
event (`CharacterEnter` map status event) that can only be produced after
`SpawnForSelf`/the map-entry command that follows `Populate` in the same
synchronous session-bootstrap call, so per-partition Kafka ordering keeps
`Populate` ahead of this read too. No path found where a character is encoded
with an unpopulated cache on its own load.

**Idempotency vs. invalidation — the interaction the brief specifically asked
about:** `Populate` (`ring/processor.go:101-113`) short-circuits on any cache
hit, including a hit that resulted from `Invalidate` never having run since a
stale prior session (see Priority-1-adjacent finding below on FR-4). This
correctly satisfies the "second load in the same presence does not re-fetch"
criterion (proven by `TestRingCachePopulatedOnCharacterLoad`,
`ring/processor_test.go:412-449`, and reproduced independently — see
"Independent verification" below). It does **not** by itself cause a
never-refreshing entry within one full task-12 flow, because `Invalidate` is
called unconditionally (not session-gated) from every channel process that
consumes the `RING_PURCHASED` topic (`kafka/consumer/cashshop/consumer.go`
registers its own consumer group per channel process —
`kafka/consumer/cashshop/consumer.go:57-59` — so every channel instance sees
every purchase event and drops its own local copy of both halves regardless of
where the buyer/partner currently is).

**Gap found:** the wiring itself (`session/consumer.go:217`) has zero test
coverage — `kafka/consumer/session/consumer_test.go` only tests
`claimEnableAnnouncer` extracted from this same function, per that file's own
documented practicality boundary, and the new `ring/processor_test.go` tests
only prove `Processor.Populate`'s idempotency in isolation, not that
`processStateReturn` actually calls it. Deleting the one-line
`_ = ring.NewProcessor(l, ctx).Populate(c.Id())` from `session/consumer.go`
would not fail any test in this suite. This mirrors the pre-existing pattern
in this file (most of `processStateReturn`'s other side effects — guild fetch,
buddy list emit — are similarly untested at the wiring level), so I am not
blocking on it, but it is worth naming: the single most load-bearing line in
this task is asserted only by inspection.

## Priority 2 — A1, the evictor move

Confirmed **exactly one** registration:
- `ring/cache.go`'s `init()`/`listener.RegisterEvictor` call is deleted, along
  with the now-unused `listener` and `tenant` imports (`ring/cache.go:1-9`
  diff).
- `main.go:307` adds `ring.EvictTenant(tid)` inside the existing central
  closure (`main.go:299-312`), alongside `monsterinfo.EvictTenant(tid)`.

`go build ./...` is clean (no leftover unused-import or duplicate-registration
compile issue). `EvictTenant` itself is tested directly at
`ring/cache_test.go` (`TestRingCacheTenantIsolation` — pre-existing from Task
10, asserts `EvictTenant(tenantA)` drops only tenant A's entries) — the same
level of test coverage every other line in that `main.go` closure gets (none
of the sibling registrations, e.g. `monsterinfo.EvictTenant`, have a
`main.go`-level test either; the underlying `EvictTenant` function is what's
tested). No double-registration risk found.

## Priority 3 — multi-tenancy in the invalidation path

`Processor.Invalidate` (`ring/processor.go:116-119`) derives the tenant from
`p.ctx` via `tenant.MustFromContext`, never a default. Fault-injected a
dropped tenant id (`getRingCache().invalidate(uuid.Nil, characterId)` in place
of the real tenant lookup) and reran the suite:

```
--- FAIL: TestProcessorInvalidate (0.00s)
    processor_test.go:477: lookup(tid, 100) = true after Processor.Invalidate, want false
--- FAIL: TestRingPurchasedInvalidatesCache/buyer_invalidated
--- FAIL: TestRingPurchasedInvalidatesCache/partner_invalidated_when_present
--- FAIL: TestRingPurchasedInvalidatesCache/partner_absent_is_not_an_error
--- PASS: TestRingPurchasedInvalidatesCache/wrong_tenant_untouched   (still passes, since a nil-tenant
    invalidation also fails to touch env.tenant's own entry — the OTHER three subtests are what catch it)
```
Reverted by hand (`git status --porcelain` confirmed clean afterward). This
confirms a mis-threaded/defaulted tenant id in the invalidation path is caught
— just not by the specific "wrong tenant untouched" subtest alone; it's the
combination of that subtest with the other three that pins correct tenant
threading. That combination is adequate.

## Priority 4 — getters stay I/O-free

`GetRingSet` (`ring/processor.go:121-132`) and `GetRingRecords`
(`ring/processor.go:134-161`) are both pure `getRingCache().lookup(...)` reads
with no fallback fetch. `upstreamFn` is called from exactly one place,
`Populate` (`ring/processor.go:106`). Confirmed by reading the full bodies of
both getters — no lazy-fetch crept in via the idempotency change (idempotency
lives entirely inside `Populate`).

## Priority 5 — A2 and A3

**A2** — `TestProcessorInvalidate` (`ring/processor_test.go:451-477`) is a
genuine Processor-level test: it populates through `Processor.Populate`,
confirms the entry landed under the *test's own* tenant id (not a
default/zero id) via `getRingCache().lookup(tid, ...)`, invalidates through
`Processor.Invalidate`, and confirms the miss. Confirmed non-tautological by
fault injection above (Priority 3).

**A3** — `TestBuildCharacterData_Rings`
(`socket/writer/character_data_test.go:63-108`) seeds the cache via
`ring.Processor.Populate` against a real `httptest` `/rings` fixture, calls
the real `BuildCharacterData`, and asserts on `cd.Rings.Couple[0]`'s
`PairCharacterName`, `PairCharacterId`, `OwnSN`/`PairSN`. Cross-checked against
`libs/atlas-packet/model/ring.go`: these four fields belong to
`packetmodel.CoupleRecord` (the RECORD block written by site A /
`CharacterData`) — `PairCharacterId`/`PairCharacterName`/`OwnSN`/`PairSN` do
not exist on `PairRing` (the AVATAR block used by the other three encoder
sites, which instead has `OwnSN`/`PartnerSN`/`ItemId`, no character id or
name field at all). The test genuinely pins the RECORD-block shape, including
`partnerName`, at the writer call site. Not tautological.

## Blocking

- **`services/atlas-channel/atlas.com/channel/kafka/consumer/session/consumer.go`
  (no invalidation added) and `ring/cache.go` — FR-4's second invalidation
  trigger, "character map/channel transfer," is unimplemented.** PRD FR-4
  (`docs/tasks/task-269-ring-pair-behavior/prd.md:93-95`) reads: "The cache is
  invalidated on the `RING_PURCHASED` status event... and on character
  map/channel transfer." `plan.md`'s own Task 12 Step 3
  (`docs/tasks/task-269-ring-pair-behavior/plan.md:1009`, "Map/channel
  transfer drops the entry") and `design.md §4.3`
  (`docs/tasks/task-269-ring-pair-behavior/design.md:344`, identical wording)
  both name it explicitly as in-scope work for this task. No code in this
  diff evicts a character's ring cache entry on session end. The established
  idiom for exactly this class of per-character, session-scoped state exists
  in this same file and was not reused: `session.ProcessorImpl.Destroy`
  (`session/processor.go:408-434`) already calls
  `clearBattleshipOnDestroy`, `clearAranComboOnDestroy`, and
  `clearLastPositionOnDestroy` — each documented with "logout, disconnect,
  timeout, and channel change all funnel here" — but no analogous
  `clearRingCacheOnDestroy` was added. Practical impact today is bounded (every
  channel process independently consumes the full `RING_PURCHASED` topic via
  its own consumer group, so a stale entry left behind by a channel-transfer
  is still corrected the next time that character's rings actually change),
  but it is an unbounded per-character memory leak in every channel process
  (an entry is created on every load and never removed except at
  tenant-wide drain), it is an explicitly named PRD requirement and an
  explicitly named plan.md implementation step, and the implementer's report
  does not flag it as a deferral — unlike the Populate-callsite deviation,
  which the report does flag. This reads as a dropped requirement, not a
  scoped-out one. No test exists for it because no code exists for it.

## Non-blocking

- `kafka/consumer/session/consumer.go:217` — the `Populate` wiring itself
  (as opposed to `Processor.Populate`'s own behavior) has no test coverage;
  deleting the call would not fail the suite. Matches the existing,
  documented practicality boundary in this file for the rest of
  `processStateReturn`'s side effects, so not blocking, but worth naming
  since the report's own framing calls this "the judgment call" the reviewer
  should weigh most heavily.
- `docs/tasks/task-269-ring-pair-behavior/plan.md:1009` step 3's exact phrase
  "Map/channel transfer drops the entry" is easy to read as descriptive
  ("this is *how* the architecture already behaves") rather than
  imperative ("*implement* this"). Given FR-4's phrasing in the PRD is
  unambiguous and imperative, I read it as a requirement above, but the
  planning artifacts could have been clearer that this needed its own
  explicit code, not just the login-populate wiring.

## Not evaluable

- End-to-end ordering between `Populate` and the async `CharacterEnter`
  map-status broadcast (`enterMap`) is verified by Kafka per-partition
  ordering reasoning and code reading, not by an integration test — no such
  test exists in this diff or is practical to add without a live broker,
  consistent with this codebase's stated boundary for `processStateReturn`.

## Verification performed

- `go build ./...` clean (module-local).
- `go test ./ring/... ./kafka/consumer/cashshop/... ./kafka/consumer/session/... ./socket/writer/...` — all pass.
- Fault-injected a dropped tenant id in `Processor.Invalidate` (`ring/processor.go`);
  confirmed `TestProcessorInvalidate` and 3 of 4 `TestRingPurchasedInvalidatesCache`
  subtests fail as expected; reverted by hand; confirmed `git status --porcelain`
  clean before and after.
- No `tools/verify.sh`/`tools/lint.sh` run (per instructions; a concurrent gate
  run was already in progress).
