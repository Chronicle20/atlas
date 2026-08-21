# Review: Task 12 — `atlas-monster-death` rewrite `DistributeExperience` as resolve → plan → award

**Range reviewed:** `b99352738..5218f8aba` (single commit `5218f8aba`)
**Brief:** `.superpowers/sdd/plan/task-12-brief.md`
**Report:** `.superpowers/sdd/plan/task-12-report.md`

## Scope confirmed

The diff touches exactly what the commit message and report claim:

```
monster/builder.go            |  63 -------
monster/builder_test.go       |  89 ---------
monster/experience.go         |  24 +++
monster/experience_test.go    |  54 ++++++
monster/model.go              |  24 +--
monster/processor.go          | 206 +++++++++++++--------
monster/processor_test.go     |  27 ---
```

`processor.go` and `model.go` are named in the brief's Files inventory. `experience.go`/`experience_test.go` additions are the one pure, testable piece (`aggregateDamageEntries`) the brief calls for. `builder.go`/`builder_test.go` and the one deletion in `processor_test.go` are outside the brief's Files list — addressed as a targeted item below. `experience.go`'s non-additive contents (`planDistribution`, `computeAward`, the config/rate types) were not touched by this diff and were read only as call-site contracts for `DistributeExperience`, per the brief's "read-only" note.

Module builds and tests green, re-run locally (module-local, read-only):
```
$ go build ./...    → clean
$ go vet ./...       → clean
$ go test ./...      → all packages ok
```
Required greps both empty, re-confirmed:
```
grep -rn 'TODO parties\|TODO account for healing' .   → nothing
grep -n 'totalPartyLevel byte' monster/processor.go    → nothing
grep -rn 'produceDistribution|distributeCharacterExperience' .  → nothing (repo-wide)
grep -rn 'DamageDistributionModel|DamageDistributionBuilder' .  → nothing (repo-wide)
```

## Targeted item 1 — scope expansion: `builder.go` / `builder_test.go` deletion

Verified independently, not taken on the report's word:

- `builder.go` (pre-diff, read via `git show b99352738:...builder.go`) defines only `DamageDistributionBuilder`, whose every setter and its `Build()` method construct/return `DamageDistributionModel`. It has no other purpose and no other consumer.
- `builder_test.go` contains five tests, all against `DamageDistributionBuilder`.
- `processor_test.go`'s deleted `TestDamageDistributionModel_Accessors` is the only test in that file that referenced `DamageDistributionModel`; the diff confirms the rest of that file (drop/quest/threshold/white-gain tests) is untouched.
- Repo-wide `grep -rln "DamageDistributionModel\|DamageDistributionBuilder"` after the commit returns nothing — no dangling reference anywhere, including outside the module (checked from repo root, not just the module directory).
- No test lost its only coverage as a side effect: the deleted tests exercised exclusively the deleted builder/model pair; no other symbol's test coverage passed through those two files.

**Verdict on item 1: the expansion is correct and should be accepted, not rejected.** It is inseparable from the brief's own instruction ("delete `produceDistribution` and `distributeCharacterExperience` entirely... delete `DamageDistributionModel` and its accessors") — leaving a builder and five tests for a deleted type would itself have been a stub/dead-code violation of the "No stubs" global constraint. The brief's Files inventory not naming `builder.go` is an omission in the brief, not license to leave dead code; the implementer's report documents the additional grep and reasoning inline, which is the right way to make an out-of-brief deletion auditable.

## Targeted item 2 — behavior preservation on the surviving path

Traced end to end:

- **`AWARD_EXPERIENCE` contract** (`character/producer.go`): `awardExperienceCommandProvider(characterId, ch, white, amount, party)` still emits no `transactionId`/`showEffect` and still appends the `PARTY` distribution unconditionally (`Amount: party`, which is `0` for solo since `computeAward` returns `bonus=0` when `PartyBonusMod=0`). Untouched by this diff — confirmed via `character/producer.go` (not in the changed-file list) and the call site `p.cp.AwardExperience(f.Channel(), r.CharacterId, r.White, personal, bonus)` in `processor.go:191`, whose argument order (`ch, characterId, white, amount, party`) matches `character.Processor.AwardExperience`'s signature exactly.
- **Kafka consumer seam** (`kafka/consumer/monster/consumer.go:64-66`, not in the diff): still calls `monster.NewDamageEntryModel(m.CharacterId, m.Damage)` with the same two-arg shape; `DamageEntryModel`/`NewDamageEntryModel` are unchanged in `model.go` (only two accessors were *added*, not altered).
- **Solo-share preservation**: old `distributeCharacterExperience` hard-coded `hightestPartyDamage=true` and `totalPartyLevel=level` for a solo character, so `characterExperience = (0.8*level/level) + 0.2 = 1.0` — a solo character always got 100% of the pooled figure. New `planDistribution` (Task 10 territory, read-only here) builds the solo `Recipient` with `TotalPartyLevel: uint32(s.Level)` and `IsMvp: true` (`experience.go:238-248`), which drives the identical `share == 1.0` result through `computeAward`. Confirmed by reading, not assumed.
- **Monster HP → aggregated damage**: `totalDamage` is now `Σ damages` from `aggregateDamageEntries` rather than `mi.Hp()` — this is the FR-3.1/normalisation change the brief calls out explicitly as intentional, not a regression.
- **Party resolution / NFR-1 memoisation**: `partyOf[m.Id()] = pt.Id()` is set for every member of a fetched party (`processor.go:148-150`), so a later damager in the same party short-circuits on `partyOf[characterId]` and never re-fetches — matches the brief's one-lookup-per-party design.
- **D12/D13**: out-of-field damagers are filtered by `inField[characterId]` before any party lookup (`processor.go:116-121`) and never enter `solos`/`partyInputs`, matching D12. MVP selection lives in `planDistribution` (`expMembers`, non-damagers counted as 0), Task 10 territory, not re-derived here — correctly out of this task's surface.
- **Hints run strictly after all awards** (`processor.go` step 5, after the full award loop) so a `ShowHint` publish failure cannot roll back or block an award — matches FR-6.10 and the brief.

No consumer of an emitted event lost a field. This unit crosses the service seam correctly for the parts it owns; the seam-defence itself (`aggregateDamageEntries`, FR-1.4) is proven by `TestAggregateDamageEntries`, which is a genuine red→green test — it references a function (`aggregateDamageEntries`) that did not exist before this commit, and the "accumulates, does not assign" case directly encodes the PRD acceptance bullet about the old `soloDistribution[de.characterId] = de.damage` assignment bug.

## Targeted item 3 — stubs / dead code / orphaned helpers

- No `// TODO` markers remain (`TODO parties`, `TODO account for healing` both grepped clean, matching the brief's required check).
- No commented-out code blocks in the diff.
- `soloInputFor` is a new helper, used from two call sites in the party-resolution loop (`processor.go:137,141`) — not orphaned.
- `calculateExperienceStandardDeviationThreshold` and `isWhiteExperienceGain` are retained; both are still called from `planDistribution` in `experience.go` (same package, untouched by this diff) and covered by pre-existing `processor_test.go` tests — confirmed still present in the module's test suite that stayed green.
- No stub handler or placeholder return introduced.

## Non-blocking findings

1. **`processor.go:132-138` — WARN-level log fires on every normal (non-outage) solo kill, not just a party-service outage.** `p.pp.GetByMemberId` (`party/processor.go`) resolves to `model.FirstProvider` over a `SliceProvider`; when a character has no party the REST call legitimately returns an empty result set, and `FirstProvider` turns that into `ErrEmptySlice` (`libs/atlas-model/model/processor.go:550-552`) — a non-nil `error`, indistinguishable at this call site from an actual party-service outage. Since the majority of kills involve at least one unpartied character, this means the new code path will emit `p.l.WithError(err).Warnf("Unable to locate party for character [%d]; treating as solo.", ...)` on essentially every kill in normal operation, not only during a genuine outage. Functionally harmless — `soloInputFor` still computes the correct level and the award is correct — but it defeats the purpose of a WARN log (signalling an abnormal condition) and will produce constant log noise that makes a genuine party-service outage harder to spot. This is not an implementer deviation: the brief's own Step 3 pseudocode specifies exactly this "on error, log at warn, treat as solo" handling without distinguishing "no party" from "service down," so the root cause is a gap in the brief/interface contract of `party.Processor.GetByMemberId`, not something Task 12 introduced independently. Flagging for awareness; not blocking this unit.
2. **`processor.go:140-143` — the `pt.Id() == 0` branch is effectively unreachable given `party.Processor.GetByMemberId`'s current implementation** (as traced above, `FirstProvider` never returns a zero-value `Model` with a `nil` error — the empty case is always routed through the `err != nil` branch). Harmless defensive redundancy, not a defect; noted only because targeted item 2 asked for a full trace of the surviving path.

## Not evaluable

- **Mock-driven behavioural tests of `DistributeExperience` itself** are explicitly deferred to Task 13 per both the brief ("The mock-driven tests for this behaviour are Task 13. This task's own gate is that the package still builds and every pre-existing test stays green.") and the report. This diff's own gate (build + existing-tests-green + the one pure `aggregateDamageEntries` test) is met, but the resolve→plan→award orchestration logic in `DistributeExperience` (concurrency, party-degrade-to-solo paths, award-loop continue-on-error, hint throttling) has no test asserting the new contract *in this commit*. This is consistent with the plan's task split and is not treated as a defect of Task 12, but it means item 2's "check that a test asserts the new contract" is only partially satisfiable within this unit's surface — the FR-1.4 seam defence is tested, the rest is not yet.
- `planDistribution`/`computeAward` internals (Task 9/10 territory) were read only as call-site contracts, not re-audited for correctness — out of this task's surface per the brief's "read-only" note.

## Summary

The rewrite does what the brief asked: `DistributeExperience` now follows resolve → plan → award → hints, `produceDistribution`/`distributeCharacterExperience`/`DamageDistributionModel` are fully retired with no dangling reference anywhere in the repo, the `totalPartyLevel byte` overflow (D6) and both stale TODOs are gone, and the out-of-brief `builder.go`/`builder_test.go` deletion is independently verified as correct and complete rather than a scope violation. The `AWARD_EXPERIENCE` Kafka contract and the `DamageEntryModel` consumer seam are both confirmed unchanged. No blocking defects found.

---

```text
verdict: APPROVED_WITH_FINDINGS
artifact: docs/tasks/task-254-party-experience-sharing/review-task-12.md
scope_confirmed: reviewed the full diff of b99352738..5218f8aba (processor.go, model.go, experience.go, experience_test.go, processor_test.go, builder.go, builder_test.go); traced call-site contracts into party/processor.go, party/model.go, character/producer.go, character/processor.go, system_message/processor.go, system_message/throttle.go, monster/information/model.go, kafka/consumer/monster/consumer.go, and libs/atlas-model/model/processor.go (FirstProvider) as read-only dependencies; did not re-audit planDistribution/computeAward internals (Task 9/10 territory, read-only per brief)
blocking: 0
non_blocking: 2
  - services/atlas-monster-death/atlas.com/monster/monster/processor.go:132-138 — party-lookup-empty (no party) is indistinguishable from a real outage, so WARN fires on essentially every solo kill in normal operation, not just during an outage; root cause is the brief's own error-handling spec plus party.Processor.GetByMemberId's contract, not an implementer deviation
  - services/atlas-monster-death/atlas.com/monster/monster/processor.go:140-143 — the `pt.Id() == 0` solo-degrade branch is unreachable given GetByMemberId's current behaviour; harmless dead defensive code
not_evaluable: 1
  - DistributeExperience's own orchestration behavior (concurrency, degrade-to-solo, award-loop continue-on-error, hint throttling) has no asserting test within this commit; explicitly deferred to Task 13 by the plan, not a Task 12 defect
```
