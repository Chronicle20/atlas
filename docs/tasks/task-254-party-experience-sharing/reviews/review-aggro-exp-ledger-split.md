# Review — split aggro table from the experience ledger (ee77b949c)

**Brief:** `docs/tasks/task-254-party-experience-sharing/bug-aggro-decay-corrupts-exp-ledger.md`
**Commit:** `ee77b949c` (parent `7225787da`), service `atlas-monsters`
**Scope:** `services/atlas-monsters/atlas.com/monsters/monster/{builder,model,model_test,registry,rest,experience_ledger_test}.go`, `services/atlas-monsters/docs/{domain,rest}.md`

## Verdict

APPROVED_WITH_FINDINGS — the fix is correct and complete against the brief; one
cosmetic `gofmt` violation is the only defect found.

## 1. Sweep for missed writers/mutators of DamageEntries / damageEntries

Full-module grep of `DamageEntries|damageEntries` across every non-test `.go`
file in `atlas.com/monsters` (builder.go, model.go, registry.go, aggro_task.go,
producer.go, kafka.go, processor.go, rest.go):

- **ApplyDamage** (`registry.go:578,584`) — writes both `cur.DamageEntries`
  (via `creditEntry`) and `cur.ExperienceEntries` (via `creditEntry`) with the
  identical clamped `actual` figure. Correct.
- **ModelBuilder.AddDamageEntry** (`builder.go:155-158`) — credits
  `b.damageEntries` and `b.experienceEntries` via `creditModelEntry`. Correct,
  and `Clone` (`builder.go:34-35`) carries both fields forward.
- **DecayDamageEntries** (`registry.go:864-909`) — mutates only
  `cur.DamageEntries`; `cur.ExperienceEntries` is never referenced in the
  closure. Correct (this is the bug's root cause, now fixed).
- **ClearDamageEntries** (`registry.go:940-956`) — mutates only
  `cur.DamageEntries`. Correct (the other bug half, now fixed).
- **SelfDestruct** (`registry.go:621-629`) — touches neither ledger (sets
  `cur.Hp = 0` only), matching its own doc comment ("Damage entries are left
  untouched: a detonation is not damage"). Pre-existing test
  `TestRegistrySelfDestructLeavesDamageEntries` (`self_destruct_registry_test.go:50`)
  still passes and now exercises `DamageSummary()`/`DamageLeader()` against the
  corrected ledger — confirmed by `go test` run below.
- **ApplyRecovery** (`registry.go:657-702`, HP/MP regen) — touches `cur.Hp` /
  `cur.Mp` only, never damage entries. Unaffected, correctly untouched.
- **SetAggro** (`processor.go`, exercised by `set_aggro_test.go:366`
  `TestSetAggro_LeavesDamageEntriesUntouched`) — auto-aggro claims never call
  `ApplyDamage` or any entry mutator; test still passes.
- **NewMonster** (`model.go:110-111`) — initializes both `damageEntries` and
  `experienceEntries` as independent empty slices (`make([]entry, 0)` called
  twice, not shared/aliased backing arrays). Correct.
- No remaining Lua scripts (`decayDamageEntriesScript`, `applyDamageScript`,
  `applyRecoveryScript` are all named only in comments as historical
  references; `grep -n "Script\|\.lua"` finds no executable script bodies).

No missed writer found. The sweep is exhaustive over the module (not a
spot-check): every reference to `DamageEntries`/`damageEntries` in every
non-test file was enumerated and individually classified above.

## 2. Round-tripping (toStored/fromStored/creditEntry/toStoredEntries/fromStoredEntries)

- `creditEntry` (`registry.go:112-125`) is shared by both ledgers' write path
  in `ApplyDamage`, so aggro and EXP entries can never structurally diverge in
  representation, only in mutation history (decay/clear).
- `toStoredEntries`/`fromStoredEntries` (`registry.go:135-166`) are generic
  projectors reused for both `DamageEntries` and `ExperienceEntries` in
  `toStored`/`fromStored` (`registry.go:174-175`, `246-247`). Symmetric,
  correct.
- Legacy row (no `experienceEntries` key in the Redis JSON blob): Go's
  `encoding/json` leaves `storedMonster.ExperienceEntries` at its zero value
  (`nil` `damageEntryList`) when the key is absent — the field's custom
  `UnmarshalJSON` is only invoked when the key is present, so a missing key is
  not an error path at all. `fromStoredEntries(nil)` then returns
  `make([]entry, 0, 0)` (empty, non-nil) via the `order`-loop, never an error.
  Confirmed by reading `damageEntryList.UnmarshalJSON`
  (`registry.go:63-67`) and `fromStoredEntries` (`registry.go:141-166`).
- `ExperienceEntries` has no `omitempty` tag, matching `DamageEntries`'s
  existing convention — consistent.

## 3. Aggro-ledger vs experience-ledger read sites

- `Model.DamageEntries()` (`model.go:178-180`) still returns `m.damageEntries`
  — unchanged, still the aggro view.
- `aggro_task.go:84` (`entries := m.DamageEntries()`, feeding `IsAggroIdle`)
  and `aggro_task.go:112` (`DecayDamageEntries`) are untouched by this commit
  (`git diff` confirms `aggro_task.go` has zero hunks) and still read/mutate
  the aggro table exclusively.
- `Model.DamageSummary()` (`model.go:190-192`) now returns
  `m.experienceEntries`, with an explicit doc comment explaining the switch
  and the regression it fixes.
- `Model.DamageLeader()` (`model.go:251-264`) now iterates
  `m.experienceEntries` instead of `m.damageEntries`.
- No call site was found reading the wrong ledger in either direction.

## 4. model_test.go `makeModelWithEntries` change

```go
func makeModelWithEntries(entries []entry) Model {
	return Model{damageEntries: entries, experienceEntries: entries}
}
```

This seeds both ledgers with the *same* slice, so `TestDamageSummaryPassthrough`
and `TestDamageLeaderOverAggregatedEntries` in `model_test.go` no longer
distinguish which ledger `DamageSummary()`/`DamageLeader()` read from — both
tests would still pass even if those methods had been left reading
`damageEntries`. Taken alone this would be "coverage that papers over a
regression."

However, the new `experience_ledger_test.go` closes that gap explicitly:
`TestDecayLeavesExperienceLedgerIntact`, `TestClearAggroLeavesExperienceLedgerIntact`,
and `TestDamageLeaderUsesExperienceLedger` all construct a divergence between
the two ledgers (via real `DecayDamageEntries`/`ClearDamageEntries` calls
through the registry) and assert on the EXP-ledger-specific value. These would
fail against the pre-fix code (confirmed structurally: pre-fix
`DamageSummary`/`DamageLeader` read `m.damageEntries`, which the decay/clear
calls in these tests mutate down to 0 or a flipped leader). So the *unit's*
test suite as a whole does pin the new contract; `model_test.go`'s change is a
non-regressive simplification of an existing fixture, not the load-bearing
regression test.

Non-blocking observation: `TestDamageSummaryPassthrough` /
`TestDamageLeaderOverAggregatedEntries` would read more honestly if seeded
with *different* aggro/experience entries so they independently pin
"DamageSummary reads experienceEntries," rather than relying on
`experience_ledger_test.go` to carry that assertion alone. Not blocking —
the assertion exists and is real, just carried by a different file.

## 5. Cross-service seam — DAMAGED/KILLED wire shape

`producer.go`, `kafka.go`, and `processor.go` have **zero diff hunks** in this
commit (confirmed via `git diff ee77b949c~1 ee77b949c -- producer.go kafka.go
processor.go` — empty output). All producer call sites that assemble
DAMAGED/KILLED events route through `m.DamageSummary()` /
`last.Monster.DamageSummary()` / `s.Monster.DamageLeader()`
(`recovery_task.go:70`, `processor.go:633,694,731,817,828,1280,1983`,
`status_task.go:103`) — same `[]entry` → `[]damageEntry` JSON shape as before,
only the *values* now come from the corrected ledger. The wire contract
(`json:"damageEntries"` field name and structure in `kafka.go`) is unchanged.

Checked the six consumers for `damageEntries`/`DamageEntries` references:
`atlas-quest/.../consumer/monster/consumer.go:52` (`for _, entry := range
e.Body.DamageEntries`) and `atlas-monster-death/.../monster/processor.go:213`
→ `experience.go:21` (`aggregateDamageEntries`) both consume the event's
`DamageEntries` list generically — neither hardcodes or special-cases the old
buggy numbers, so both are fixed by the corrected producer values with no
consumer-side change required, matching the brief's claim. `atlas-channel`,
`atlas-consumables`, `atlas-maps`, `atlas-party-quests` references are message
struct definitions / REST passthroughs only, not decision logic keyed on the
now-corrected damage figures.

## Backend-dev-guidelines / repo conventions

- Builder pattern used correctly for both the production `ModelBuilder` and
  the test constructions (via `r.CreateMonster` / `Clone(...).Build()` in the
  new tests) — no `*_testhelpers.go`-style bypass introduced.
- **Finding (non-blocking):** `gofmt -l` flags `monster/model.go`. Inserting
  the `experienceEntries []entry` field (longer name than several existing
  fields) shifted the struct's aligned column but the surrounding
  `statusEffects`/`nextSkillDecision`/`lastDamageTakenMs` field lines were not
  re-aligned to match. `gofmt -d monster/model.go` shows only whitespace
  realignment, no semantic change. `tools/verify.sh` does not run a `gofmt`
  check itself (grepped, no hit), so this will not fail the flagless gate, but
  it is a real convention violation and a one-line `gofmt -w` fixes it.
- Doc comments (`model.go:64-68`, `registry.go` for `DecayDamageEntries` /
  `ClearDamageEntries` / `creditEntry` / `toStoredEntries` /
  `fromStoredEntries`) are accurate and explain *why*, matching repo style.
- `docs/domain.md` and `docs/rest.md` were updated in the same commit to
  describe the new field and its semantics — consistent with the code.

## Verification run

```
$ go build ./...            # clean
$ go vet ./...               # clean
$ go test ./... (atlas-monsters/atlas.com/monsters)   # all packages ok
$ go test ./monster/... -run 'Damage|Aggro|Experience|Model' -v
    ... all PASS, including TestDecayLeavesExperienceLedgerIntact,
    TestClearAggroLeavesExperienceLedgerIntact,
    TestDamageLeaderUsesExperienceLedger,
    TestApplyDamageClampWritesBothLedgers,
    TestBuilderAddDamageEntryFeedsExperienceLedger,
    TestRegistrySelfDestructLeavesDamageEntries,
    TestSetAggro_LeavesDamageEntriesUntouched
```

(Note: `go test` for this package spins up an in-process Redis-backed
registry via `testContext`; no external services were reached beyond the
expected/pre-existing `data/monsters/...` HTTP lookup retries visible in the
`SelfDestruct` test's logs, which are unrelated pre-existing noise from an
`information` HTTP client with an empty base URL in the test environment, not
caused by this commit.)

## Not evaluable

None. The unit is self-contained to `atlas-monsters`; the claimed absence of
required consumer-side changes was verified by inspection of the consumer
call sites listed above (grep + read), which is within the review's
"trace the event into its consumers by hand" mandate and did not require
running those services.

## Summary

- Blocking: 0
- Non-blocking: 2 (gofmt alignment in `model.go`; `model_test.go`'s
  `makeModelWithEntries` fixture no longer independently distinguishes the two
  ledgers, though `experience_ledger_test.go` carries that assertion)
- Not evaluable: 0
