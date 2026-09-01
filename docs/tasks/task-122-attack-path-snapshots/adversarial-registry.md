# Adversarial review — task-122 snapshot registry (concurrency, lifecycle, correctness)

Scope: `services/atlas-channel/atlas.com/channel/character/snapshot/{registry.go,processor.go,shadow.go,metrics.go}`,
plus session-destroy hooks (`session/processor.go`), tenant-drain wiring (`main.go`), event-handler call sites
(`kafka/consumer/{character,skill,asset,compartment,buff}/consumer.go`), the movement position feed
(`movement/processor.go`), and the one cross-service producer change (`atlas-character/character/processor.go`)
that the generation-guard correctness claim depends on. Diff base: `main...HEAD`.

## 1. Generation-guard races

**PASS** — `backfill` (registry.go:204-222) re-checks `genOK(e)` after the REST fetch under exclusive lock;
`Get`/`GetBuffs` (processor.go:55-176) capture `v.<Component>Gen` from `View` *before* issuing REST, and REST
calls run with no lock held (View's Lock/Unlock at registry.go:98-99 brackets only the read; the REST calls at
processor.go:78-131 are outside any lock; Backfill re-locks at registry.go:205). No I/O under the mutex —
confirmed by reading every call site, not by "looks correct."

- Invalidate-during-backfill: every `mutate`-based invalidator (`InvalidateCore/Skills/Inventory/Buffs`,
  registry.go:391-397, 443-449, 587-593, 636-642) bumps the component's gen before returning, so a backfill
  keyed to the pre-invalidation gen is discarded (registry.go:214-217) and the discard is counted
  (`kindBackfillDiscarded`). This exact path is asserted, not just exercised, by
  `TestRegistry_StaleBackfillDiscarded` (registry_test.go:128-152).
- Apply-during-backfill: in-place event mutators (`ApplyStatChanged`, `UpsertSkill`, `UpsertAsset`, …) also bump
  gen unconditionally before applying (e.g. registry.go:329, 401, 493), so an event that arrives mid-fetch
  invalidates the in-flight backfill even though it "successfully" mutated the component in the interim.
- Two concurrent reads both triggering refetch: both capture the same pre-fetch gen, both fetch, both backfill
  successfully (gen unchanged between them) — redundant REST work, not corruption; last writer's REST body wins,
  which is no worse than sequential re-reads. Not a bug.
- Position has no generation counter, and none is needed: per design §4.2 there is no REST-specific backfill
  path for position — `SetPosition`/`InvalidatePosition` (registry.go:646-659) are the only writers, and
  `Get`'s position overlay falls back to the base REST core's X/Y when `!PosValid` (processor.go:135-137),
  exactly today's source. Verified: covers the position write path correctly.
- Buffs use the same generation-guard machinery as the other three components (`BuffsGen`, `BackfillBuffs`,
  registry.go:197-202) — no separate/weaker path.

## 2. Lifecycle

**PASS**, with one non-blocking gap.

- `session.Processor.Destroy` (session/processor.go:409-420) is the single teardown funnel: `Evict(t,
  s.CharacterId())` runs unconditionally, and the comment there explicitly documents "logout, disconnect, and
  channel change all funnel here." `CharacterId can be 0 for pre-login sessions; Evict no-ops then" — confirmed:
  `Evict` (registry.go:663-674) is a plain map lookup+delete, safe on a characterId that was never populated
  (character-selection back-out is covered — no entry to leak).
- Tenant-wide drain: `listener.RegisterEvictor` in main.go:307-319 calls `snapshot.GetRegistry().EvictTenant(tid)`
  alongside every other per-pod tenant-scoped cache (account, monster mirrors, skill-data cache) — consistent
  wiring, no snapshot-specific gap.
- Unbounded growth: entries are created only by `View` (registry.go:97-100, comment at :93-96 states this
  explicitly and is honored — no `mutate`-based creation anywhere; every mutator early-returns via
  `entryLocked(..., false)` returning nil, registry.go:235-238). Bounded by session count as designed, assuming
  every session reaches `Destroy` exactly once.
- **Non-blocking gap (lifecycle race, not confirmed exploitable in this diff's scope):** `View` is the only
  entry-creating call, and it is invoked from the attack-handling goroutine, not from session lifecycle code.
  If a packet-handling goroutine for a session is still in flight (mid-`processAttack`) when that session's
  `Destroy`/`Evict` runs (disconnect racing an in-flight packet), `View` can recreate an entry for a character
  id whose session has already been torn down — Evict already ran and will not run again for that instance,
  so the resurrected entry survives until (a) the character reconnects on the same pod and reuses it, or
  (b) EvictTenant fires on tenant drain. Whether this is reachable depends on the socket dispatch model
  (goroutine-per-packet vs. sequential per-connection read loop), which lives outside this diff's files
  (`socket/` read-loop internals were not touched by this unit and were not read for this review) — reported
  under "Not evaluable" rather than asserted as a confirmed defect.

## 3. Tenant scoping

**PASS.** `Registry.perTenant` is `map[uuid.UUID]map[uint32]*entry` (registry.go:54), and every mutator/reader
takes `t tenant.Model` and indexes via `t.Id()` (registry.go:112). Every Kafka handler wired to the snapshot
derives `t := tenant.MustFromContext(ctx)` per-message (verified at kafka/consumer/character/consumer.go:561,
574, 587, 605 and the equivalent lines in asset/compartment/skill/buff consumers) — no global/package-level
tenant variable is used. `sc.IsWorld`/`sc.Is` filters gate on world/channel match but tenant isolation is
enforced independently at the data-structure level regardless of that filter, so a filter bug could not leak
data cross-tenant.

## 4. Redelivery/ordering

**PASS**, for the property this unit controls.

- In-place appliers only ever apply absolute values: `ApplyStatChanged` applies `Values` keyed absolute floats
  (registry.go:271-311); `SetLevel`/`SetExperience` set `Current` absolute (registry.go:369-389);
  `SetAssetQuantity` sets absolute quantity (registry.go:521-538); `SetAssetSlot` sets absolute slot
  (registry.go:540-557); `UpsertAsset`/`UpsertSkill`/`UpsertBuff` replace-by-key with the event's full payload
  (registry.go:399-424, 491-519, 595-617). No delta arithmetic exists anywhere in registry.go — grepped and
  read every mutator; confirmed idempotent under at-least-once redelivery.
- Cross-service verification: the one producer change this design depends on
  (`services/atlas-character/atlas.com/character/character/processor.go`) was checked in the diff — every
  STAT_CHANGED site that previously emitted `nil` `Values` now emits the absolute post-mutation value (e.g.
  `RequestChangeMeso` at processor.go:906-921, `RequestChangeFame` at :1031-1047, the AP-distribution loop at
  :1080-1150). `plan.md:2003` records that MESO/FAME_CHANGED (delta-only events, kafka.go:173-182) deliberately
  get **no** dedicated snapshot handler because every meso/fame mutation site also emits a sibling
  STAT_CHANGED with the absolute value — this contradicts design.md's §5 table ("MESO/FAME_CHANGED (deltas) →
  InvalidateCore") but the deviation is explicitly recorded and verified in plan.md, not silently dropped.
  Not a finding.
- Out-of-order across partitions/topics for the same character (e.g., a skill event delivered after a
  character event that logically preceded it): the generation counter only protects REST-backfill-vs-event
  ordering *within a single component*; it does not order two in-place *event* applications against each other.
  If Kafka delivers, e.g., two STAT_CHANGED events for the same character out of causal order (broker-level
  reordering across partitions, or a redelivery after a partition rebalance), the second-applied one wins
  unconditionally — there is no per-event sequence/version field compared against the *previous event's*
  value, only against the component's REST-backfill generation. This is inherent to Kafka at-least-once
  without per-key ordering guarantees and is not unique to this unit; whether the producer topics are
  partitioned by characterId (which would make this a non-issue for same-character events) was **not**
  verified — the partition-key wiring lives in shared producer/kafka library code outside this diff's file set.
  Reported under "Not evaluable."

## 5. Locking

**PASS.** Read end-to-end:

- `View` (registry.go:97-108): `Lock`/`defer Unlock`, value-copies every field into `ComponentView`, no I/O
  under the lock.
- `ComposedIfValid` (registry.go:135-160): double-checked locking — `RLock`, check validity+cache, `RUnlock`;
  on stale-cache, re-acquire `Lock`, re-validate (handles a concurrent `Evict` deleting the entry between the
  two lock acquisitions — `entryLocked(..., false)` returns nil, handled at :152-154), rebuild, `Unlock`.
  Correct pattern, no I/O under either lock.
- `backfill` (registry.go:204-222) and `mutate` (registry.go:232-241): `Lock`/`defer Unlock`, pure in-memory
  work, no I/O.
- `Get`/`GetBuffs` (processor.go): the three-plus-one REST fetches (`coreFetchFn`, `inventoryFetchFn`,
  `skillsFetchFn`, `buffsFetchFn`) all run *outside* any registry lock — confirmed no deadlock/latency-under-
  lock risk.
- Copy-on-read vs. aliasing: `character.Model.SetInventory`/`SetSkills` (character/model.go:313-357) go through
  `CloneModel(m)...MustBuild()` — an immutable-builder pattern, not in-place field mutation — so `composeLocked`
  building on `e.core` never mutates the entry's stored model. Slice-typed components (`e.skills`, and
  compartment/asset slices inside `e.inv`) are never mutated in place by any event mutator: `UpsertSkill`,
  `RemoveSkill`, `mutateAssetInInventory`, `UpsertAsset`, `SetAssetQuantity`, `SetAssetSlot`, `RemoveAsset` all
  build a fresh `out := make([]T, 0, …)` slice and reassign `e.skills`/`e.inv` rather than mutating the existing
  backing array (registry.go:405-421, 470-486, 503-518, 527-536, 546-556, 565-577) — a slice header copied out
  via `View`/`ComposedIfValid` to a reader cannot be corrupted by a later event. `replaceCompartment`
  (registry.go:455-465) explicitly documents and avoids the `inventory.CloneModel`-shares-the-map hazard by
  using a fresh `inventory.NewBuilder` instead of cloning — and this exact hazard is regression-tested by
  `TestRegistry_AssetMutationDoesNotAliasPriorReads` (registry_test.go:393-417).

## 6. Failure path

**PASS**, per design's own stated intent, with a rare degraded-composition edge noted as non-blocking.

- Core REST fallback failure: `Get` returns the error immediately (processor.go:79-83) — no zero-value model,
  no stale snapshot served. Matches design §4.2 ("A REST fallback failure propagates exactly as today's error
  path... it never serves a stale component in place of a failed refetch").
- Inventory/skills REST fallback failure: matches the pre-existing decorator-chain behavior by design —
  `invOk`/`skillsOk` stay false and the corresponding `SetInventory`/`SetSkills` call is skipped
  (processor.go:144-149), leaving those parts of the composed model at the base/zero value rather than
  crashing. Non-blocking edge case (not exercised by a specific test in this diff, and not independently
  re-derived here): if core *also* needed a refetch (freshly fetched with no inventory ever set) *and*
  inventory's fallback fails on the same call, the served model's `Inventory()` is `character.Model`'s zero
  value rather than an initialized-but-empty inventory. Iterating a nil `Compartments()`/`Assets()` slice does
  not panic, so this degrades to "character appears to carry no items this swing" rather than crashing — an
  acceptable-looking outcome but not independently verified against the pre-existing decorator chain's exact
  zero-state.

## 7. Test honesty

**PASS.** `TestRegistry_StaleBackfillDiscarded` (registry_test.go:128-152) asserts the actual generation-guard
semantics deterministically (not merely under `-race`): it forces the exact interleaving (View → concurrent
invalidate → stale backfill) and asserts both that the backfill call returns `false` and that the component is
left invalid — a test that would fail without the gen-check in `backfill`. `TestRegistry_ConcurrentAccess`
(registry_test.go:533-565) additionally runs 8 goroutines × 200 iterations of View/Backfill/ApplyStatChanged/
SetPosition/ComposedIfValid/Evict against shared characterIds — a genuine `-race`-detectable concurrent-access
test, not a sequential unit test dressed up as one. `TestRegistry_AssetMutationDoesNotAliasPriorReads`
(registry_test.go:393-417) is a real regression test for the aliasing hazard documented in registry.go:451-454.

## Non-blocking findings

1. **registry.go:138, processor.go:57** — `ComposedIfValid`'s fast path additionally requires `e.buffsValid`,
   but `Get()` (the attack path's only call) never populates buffs — buffs are populated solely by the separate
   `GetBuffs`/`BuffsProvider`, which per design §2.4 is only invoked once per **ranged** swing. For melee/magic
   swings, `buffsValid` never transitions to true (event mutators for buffs no-op while invalid, registry.go:
   598, 622), so `ComposedIfValid` never returns `true` for those sessions and the composed-model cache is dead
   code for the majority of attack types. This does not cause any additional REST calls (Get's slow path still
   only refetches genuinely-invalid components), so it is a missed-caching/perf nit against design §4.2's
   stated "fast path," not a correctness defect.
2. **session lifecycle (see §2)** — a theoretically possible Evict-then-resurrect race between an in-flight
   attack-handling goroutine and a concurrent `Destroy`/`Evict` was not confirmed exploitable because it
   depends on socket dispatch code outside this diff.

## Not evaluable

1. Whether the underlying Kafka topics this snapshot consumes (character status, skill, asset, compartment,
   buff) are partitioned by `characterId`, which would make same-character cross-partition reordering a
   non-issue — the partition-key wiring lives in shared producer/kafka library code not touched by this diff.
2. The socket read-loop's dispatch model (goroutine-per-packet vs. sequential-per-connection), needed to
   confirm or rule out the Evict-then-resurrect lifecycle race noted above — outside this diff's file set.
3. The pre-existing (undiffed) `character.ProcessorImpl.InventoryDecorator`/`SkillModelDecorator` failure
   behavior, cited by processor.go's comments as the parity target for degraded-model composition — not
   independently re-read to confirm the exact zero-state equivalence claimed in the comments.

## Verdict rationale

No blocking defect was located with a concrete `file:line` reproduction. The generation-guard, locking
discipline, tenant isolation, and idempotent-absolute-value invariants are all implemented as designed and are
backed by tests that would fail without the guard (not merely passing incidentally). The two non-blocking items
are a dead fast-path optimization and an unconfirmed (not ruled in or out) lifecycle race that requires reading
files outside this unit's scope to resolve.
