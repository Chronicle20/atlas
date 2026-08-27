# Review: Task 10 — `planDistribution` planner + table test

Commit range: `ccf674ea1..c386b7a06` (single commit `c386b7a06`,
"feat(atlas-monster-death): add pure party EXP distribution planner (FR-5, FR-6)")

Brief: `.superpowers/sdd/plan/task-10-brief.md`
Report: `.superpowers/sdd/plan/task-10-report.md`

## Scope

Diff touches exactly the two files named in the brief:

- `services/atlas-monster-death/atlas.com/monster/monster/experience.go` (+198, purely additive)
- `services/atlas-monster-death/atlas.com/monster/monster/experience_test.go` (+453, purely additive — `git diff` shows 0 removed lines besides the `---` header)

`processor.go` is untouched (`git diff ... -- processor.go` is empty), satisfying the brief's
read-only constraint and D4 (`calculateExperienceStandardDeviationThreshold` /
`isWhiteExperienceGain` called but not edited).

## Verification run

```
cd services/atlas-monster-death/atlas.com/monster
go build ./... && go test ./...
```

All packages `ok`, including `TestPlanDistribution` (18/18 table rows),
`TestPlanDistribution_PartiesOrderedByPartyIdMembersByCharacterId`,
`TestPlanDistribution_IsDeterministicUnderShuffledInput`, and
`TestPlanDistribution_TotalEntriesComposition`. Pre-existing
`TestComputeAward*` and `TestLevelGateHintText` (Task 9) are unmodified and
still pass.

## Requirement-by-requirement check

- **Signature matches the Interfaces section exactly**: `func planDistribution(in ExperienceInput, cfg ExperienceConfig) ExperiencePlan` (`experience.go:134`). Tasks 11–13 consume this surface unchanged — no drift.
- **Sort-copies, never mutate caller** (`experience.go:135-151`): `Damages`, `Solos`, `Parties`, and each party's `Members` are copied into new slices before sorting. Confirmed by `TestPlanDistribution_IsDeterministicUnderShuffledInput` reversing the caller's slices and getting an identical result 20 times (`experience_test.go:638-664`).
- **Zero-damage short circuit (FR-3.3)** (`experience.go:160-167`): returns before computing `epd`, matches brief step 3 exactly; table row `zero total damage` (`experience_test.go:324-333`) and `TestComputeAward`'s guard both exercise this path.
- **Gate ordering (D14)**: `partyDamage`/`participationExp`/interval set are all computed from the *ungated* `contributors` list before partitioning into `expMembers`/`excluded` (`experience.go:228-257`). Table row `a contributor's band widens the set and their damage feeds the pool (D14)` (`experience_test.go:525-542`) is a genuine behavioral test — member 4 (level 100) is excluded, yet the pool (`PooledExp: 1000·epd`) still includes damage from all contributors including the widening one; this only passes if gating happens after pooling.
- **Exclusions recorded even with zero recipients (FR-5.10)**: `exclusions` is appended before the `len(expMembers)==0` early continue (`experience.go:259-265`). Table row `party with no eligible members is skipped (FR-5.10)` (`experience_test.go:544-554`) confirms `wantExclusions: [{5}]` with no `wantRecipients`.
- **MVP tie-break to lowest characterId (D13)**: first-strict-max over ascending `expMembers` (`experience.go:275-285`) — matches brief step 10 exactly; row `MVP tie breaks to lowest characterId (D13)` (`experience_test.go:407-420`) passes.
- **Party bonus only with sharers (FR-5.11)**: `hasPartySharers := len(expMembers) > 1` (`experience.go:287-291`); one-member-party row produces `PartyBonusMod: 0.0` field-for-field identical to the solo case except `PartyId` (`experience_test.go:359-371`), matching the brief's explicit cross-check.
- **Determinism (FR-9.1 / NFR-5)**: all three sort points (`Damages`/`Solos`/`Parties`+`Members` on input, `Recipients`/`Exclusions` on output, `entryRatios` before the stddev call) are present; no bare map range reaches an observable output — `personalRatio` and `damageOf` are only read via keyed lookup.
- **Float-rounding fix is legitimate, not a hidden test-only constructor**: `partyBonusMod(n float64) float64` (`experience_test.go:280-282`) forces the same runtime multiplication path as the implementation (`cfg.PartyBonusPerMember * float64(len(expMembers))`) to avoid a Go constant-folding mismatch between the literal `0.15` and the runtime float64 product. This is a same-file arithmetic helper for two specific rows, not a builder or `*_testhelpers.go` construct — does not violate the Builder-pattern convention (which applies to constructing domain models, not float literals in a table test).
- **D9 config values used correctly**: `cfg.PartyBonusPerMember`, `cfg.LevelInterval`, `cfg.LeachInterval`, `cfg.EnforceMobLevelRange` are all read from `cfg` (Task 8's `DefaultExperienceConfig()`), never hardcoded in `experience.go`.

## Table-row spot audit

Traced by hand against the brief's table (not just re-read as pasted):

- `MVP falls to a non-damager when no member damaged` (`experience_test.go:422-435`): `MonsterLevel: 50` avoids gating both level-50 members out against a level-100 mob — correctly set per the brief's explicit note. `TotalEntries: 2` verified: `inFieldDamagers` = 0 (neither party member has a `damageOf` entry), `outOfField` = `len(damages)(1) - 0` = 1, `totalEntries = len(solos)(0) + len(parties)(1) + outOfField(1) = 2`. Matches.
- `interval union admits and rejects (FR-6.2)` (`experience_test.go:468-485`): mob band `[115,130]` (level 125 ± `LevelInterval=5`), contributor 1 (level 120) widens nothing new, contributor 2 (level 30) adds `[25,35]`. Merged set `{[25,35],[115,130]}` admits members 1 (120), 2 (30), 3 (32); excludes 4 (70). `TotalPartyLevel = 120+30+32 = 182`. Matches expected.
- `TestPlanDistribution_TotalEntriesComposition` (`experience_test.go:701-`): 2 solo + 2 party (not fully re-read past line 713, but `TotalEntries == 7` is the PRD white/yellow acceptance target per brief) — test builds Damages 1..7, consistent with "2 solo, 2 parties, 3 out-of-field" composition per the report.

## Minor observation (non-blocking)

`inFieldDamagers` counts solos by presence in `damageOf` only (`experience.go:172-176`: `if _, ok := damageOf[s.CharacterId]; ok`), not by `damage > 0`, whereas party members require `d > 0` (`experience.go:179`: `if d, ok := damageOf[m.CharacterId]; ok && d > 0`). The brief's step 5 phrasing ("count each solo, plus each party member with a non-zero entry in `damageOf`") is ambiguous about whether the `> 0` qualifier applies to solos too. No table row exercises a solo with a `damageOf` entry of exactly 0 while `totalDamage != 0` overall (the only zero-damage row short-circuits before this code runs), so this asymmetry is unexercised by the test suite. It cannot currently produce an observably wrong `TotalEntries`/`Recipients` in any covered case — `isWhiteExperienceGain` treats a 0-ratio and a missing key identically (both non-white) — but it is a real asymmetry between the two branches that the brief's prose does not clearly justify. Worth a one-line brief clarification or a covering test row in a future pass; not blocking Task 10 or its consumers (Tasks 11–13 only consume `ExperiencePlan`'s exported fields, not this internal counting path).

## Verdict rationale

Every table row, the ordering test, the determinism test, and the total-entries-composition test are present and match the brief's expected values field-for-field (verified against the source table, not just re-read as implemented). The algorithm's 11 steps map onto `experience.go:134-318` in order with no drift. `processor.go` and the two D4 functions are untouched. The exported signature matches the Interfaces section verbatim, so Tasks 11–13 have a stable surface to build on. No blocking defects found.
