# Design: Quest Start Gate by `selectedSkillID`

## Problem

A character was able to start both quest 2418 "Using Double Stab" and quest 2420 "Using Lucky Seven" simultaneously. These are intended to be mutually exclusive skill-tutorial quests on the Thief job tree: 2418 targets Bandits (skill `4001334`, Double Stab mastery) and 2420 targets Assassins (skill `4001344`, Lucky Seven mastery). Neither job possesses both passives, so no single character should be eligible for both tutorials.

## Root Cause

The WZ quest data (`Quest.wz/Check.img.xml`) defines the same start requirements for both quests: `job ∈ {400, 410, 420, 411, 421, 412, 422}` (all Thief-family jobs) plus prerequisite quest `2413` completed. On the job/prereq axis alone the two quests are identical.

The differentiator in the source data lives on `QuestInfo.img.xml`:

- `2418`: `selectedSkillID = 4001334`
- `2420`: `selectedSkillID = 4001344`

In canonical MapleStory mechanics, `selectedSkillID` acts as an implicit start gate — the character must possess the indicated skill (level ≥ 1) to be eligible. Atlas currently drops this field at every layer:

1. `services/atlas-data/atlas.com/data/quest/reader.go:28-43` — `ReadQuestInfo` populates `RestModel` fields but never reads the `selectedSkillID` attribute.
2. `services/atlas-quest/atlas.com/quest/data/quest/rest.go:10-29` — the quest `RestModel` has no field to hold it.
3. `services/atlas-quest/atlas.com/quest/data/validation/processor.go:53-186` — `ValidateStartRequirements` has no code path that would emit a skill-possession check.
4. `services/atlas-quest/atlas.com/quest/data/validation/model.go:16-23` — the atlas-quest validator's condition-type constants do not include `skillLevel`.

Result: a Thief-family character who completed quest `2413` passes every check the server performs for both `2418` and `2420`, and can start either or both.

## Scope Decision

This design fixes the forward path only. Characters who have **already** started both quests are left in place — no migration, no forfeit sweep. Cleanup has enough edge cases (partial progress, rewards already dispensed, ambiguous "should have") that it doesn't belong in the primary bug fix.

## Approaches Considered

### A. Emit a `skillLevel` validation condition (selected)

Parse `selectedSkillID` from `QuestInfo.img` into the quest `RestModel`. In atlas-quest's `ValidateStartRequirements`, append a `SkillLevelCondition` with `>= 1` whenever the quest has a non-zero `SelectedSkillId`. Query-aggregator already implements the `skillLevel` condition end-to-end.

### B. Direct skills lookup from atlas-quest (rejected)

Have atlas-quest call atlas-skills directly to check skill level. Duplicates plumbing query-aggregator already owns. Splits validation across two call paths. Harder to audit. No advantage.

### C. Client-side filter only (rejected)

Assume the client hides ineligible quests in its UI and trust that. This is the status quo and it's exactly what produced the bug. Rejected.

## Infrastructure Already in Place

Query-aggregator fully implements the `skillLevel` condition and does not need to change:

- Constant: `SkillLevelCondition` at `services/atlas-query-aggregator/atlas.com/query-aggregator/validation/model.go:47`.
- Parser validation (requires non-zero `referenceId`) at `services/atlas-query-aggregator/atlas.com/query-aggregator/validation/rest.go:275-277`.
- Evaluator at `services/atlas-query-aggregator/atlas.com/query-aggregator/validation/model.go:795-799` (calls `ctx.GetSkillLevel`).
- Fetcher at `services/atlas-query-aggregator/atlas.com/query-aggregator/validation/context.go:103-115` (delegates to `skill/processor.go`'s `GetSkillLevel`).
- REST contract documented at `services/atlas-query-aggregator/docs/rest.md:126`.

## Architecture & Scope

| Service | Change |
|---|---|
| `atlas-data` | `ReadQuestInfo` parses a new `selectedSkillID` attribute into `RestModel`. |
| `atlas-quest` — data model | New `SelectedSkillId uint32` field on the top-level quest `RestModel`. |
| `atlas-quest` — validation | Add `SkillCondition = "skillLevel"` constant. `ValidateStartRequirements` emits a single `ConditionInput` of type `skillLevel`, operator `>=`, value `1`, referenceId = `SelectedSkillId`, whenever that field is non-zero. |
| `atlas-query-aggregator` | No changes. |

No new Kafka topics. No new consumers. No new HTTP endpoints. No database migrations. No REST contract breaks. The gate flows through the existing `validations` request atlas-quest already makes against query-aggregator.

## Data Model Changes

### `services/atlas-data/atlas.com/data/quest/rest.go`

Add one field on the top-level `RestModel`:

```go
type RestModel struct {
    // ... existing fields ...
    SelectedSkillId uint32 `json:"selectedSkillId,omitempty"`
    // ... rest ...
}
```

Placement rationale:

- WZ source lives on `QuestInfo.img/<questId>/selectedSkillID`, not on `Check.img/<questId>/0/...`. Keeping the field at the top level of `RestModel` mirrors the source layout.
- It is one value per quest; it does not vary between start and end phases, so `RequirementsRestModel` is the wrong home.
- Existing top-level flags sourced from `QuestInfo.img` (`AutoStart`, `SelectedMob`, etc.) already live on `RestModel`.

### `services/atlas-data/atlas.com/data/quest/reader.go`

`ReadQuestInfo` gains one line in the `RestModel` literal:

```go
SelectedSkillId: uint32(questNode.GetIntegerWithDefault("selectedSkillID", 0)),
```

### `services/atlas-quest/atlas.com/quest/data/quest/rest.go`

Mirror the same field on atlas-quest's copy of the rest model. The quest service deserializes the atlas-data response into this type.

### `services/atlas-quest/atlas.com/quest/data/validation/model.go`

Add the condition-type constant whose value matches query-aggregator's wire string:

```go
SkillCondition = "skillLevel"
```

No new fields on `ConditionInput` — `ReferenceId` carries the skill ID, `Value` carries `1`, `Operator` carries `">="`.

## Validation Flow

### Where the condition is emitted

In `services/atlas-quest/atlas.com/quest/data/validation/processor.go`, `ValidateStartRequirements` currently appends conditions sourced from `questDef.StartRequirements`. The new check reads from the top-level `questDef.SelectedSkillId` instead and only appends when non-zero:

```go
if questDef.SelectedSkillId > 0 {
    conditions = append(conditions, ConditionInput{
        Type:        SkillCondition,
        Operator:    ">=",
        Value:       1,
        ReferenceId: questDef.SelectedSkillId,
    })
}
```

Location: after the existing `req.Quests` loop, before the `len(conditions) == 0` short-circuit. Ordering is irrelevant to correctness — query-aggregator evaluates all conditions independently and returns per-condition pass/fail.

### End requirements

The gate is **start-only**. `ValidateEndRequirements` does not emit a `skillLevel` condition. Rationale:

- Once a quest reaches `StateStarted`, the character already passed the start check. Re-gating at end would let a skill loss (SP reset, job change edge case) retroactively lock them out of work already accepted.
- `selectedSkillID` is a per-quest identity attribute in WZ, not a phase-scoped requirement.

### `skipValidation` interaction

The gate lives inside `ValidateStartRequirements`, so it is naturally bypassed when a caller passes `skipValidation = true`. This preserves existing force-start semantics:

- Kafka consumer: `Body.Force` → `skipValidation = true`.
- Saga-initiated starts that pre-validated elsewhere.
- Any admin/GM override path.

Force-start already means "skip all start-time requirements." The new check belongs in that same bucket.

### `StartChained`

`StartChained` routes through `startWithDefinition` with `skipValidation = false`, so chained starts will honor the new gate. This is correct: a chained quest still needs the right job and level; the same logic extends to skill possession.

### `CheckAutoStart`

Auto-start also routes through `startWithDefinition`. The design requires auto-started quests to honor the gate. A concrete verification task in the implementation plan: confirm `CheckAutoStart` invokes `startWithDefinition` with `skipValidation = false`. Adjust if necessary.

### Failure surfacing

On failure, `ValidateStartRequirements` returns `"skillLevel"` in the `failedConditions` slice. Callers (NPC conversation engine, REST handler, saga handler) already propagate the slice back to the originator. No new error string or error variant is needed.

## Testing Strategy

### atlas-data

Add parser tests in `reader_test.go`:

- QuestInfo XML containing `<int name="selectedSkillID" value="4001334" />` → `RestModel.SelectedSkillId == 4001334`.
- QuestInfo XML with the attribute absent → `RestModel.SelectedSkillId == 0`.

### atlas-quest — validation processor

Add unit tests in `data/validation/processor_test.go` that assert the shape of the emitted `ConditionInput` slice:

- `SelectedSkillId = 4001334` → slice contains `ConditionInput{Type: "skillLevel", Operator: ">=", Value: 1, ReferenceId: 4001334}`.
- `SelectedSkillId = 0` → slice contains no `skillLevel` entry.
- `SelectedSkillId` set **and** job/level requirements set → all expected conditions are emitted; each is independent.

These are pure assembly tests; they verify the payload leaving the processor, not the answer query-aggregator returns.

### atlas-quest — Start-path integration shape

One test that drives the full `Start` path against a stubbed query-aggregator:

- Stub returns `{Type: "skillLevel", Passed: false}`.
- Call `Start(transactionId, characterId, questId, f, false, nil)`.
- Assert the return is `ErrStartRequirementsNotMet` and `"skillLevel"` appears in the returned failed-conditions slice.

### Regression verification (manual, documented in plan)

1. Bandit (`jobId = 420`) who has skill `4001334` (Double Stab mastery) but not `4001344` (Lucky Seven mastery) and has completed quest `2413`:
   - Start quest `2418` → succeeds.
   - Start quest `2420` → fails; failed-conditions contains `"skillLevel"`.
2. Same character, now job-advanced to Assassin (`jobId = 410`) with `4001344` but not `4001334`:
   - Start quest `2420` → succeeds.
   - Start quest `2418` → fails; failed-conditions contains `"skillLevel"`.

### Negative: no-skill-gate quests unchanged

Sanity test — any quest with `SelectedSkillId == 0` (e.g., any ordinary storyline quest) validates with the same behavior as before the change. Guards against accidentally emitting a gate when the field is absent.

### Out of scope for this change's tests

- Query-aggregator's `skillLevel` evaluator (owned by query-aggregator's own suite).
- atlas-skills storage (owned by atlas-skills' own suite).
- Per-quest content of Quest.wz (data, not logic).

## Rollout

### Deploy order (preferred, not required)

1. **atlas-data** first. Older atlas-quest instances deserialize the response and harmlessly ignore the new `selectedSkillId` field.
2. **atlas-quest** second. Once it rolls, the gate enforces. If atlas-data has not yet rolled in some tenant, `SelectedSkillId` is `0` everywhere and no gate is emitted — behavior identical to today.

### Edge Cases

- **Character not found in atlas-skills.** `GetSkillLevel` returns `0`. `skillLevel >= 1` fails → start is refused. Consistent with `ValidateStartRequirements`' existing fail-closed policy (`processor.go:178`).
- **`selectedSkillID` references a skill that does not exist.** Same path — level resolves to `0`, start is refused. A data error does not silently open the gate.
- **Character has skill level > 1.** Passes `>= 1`. We deliberately do not care about specific levels, only possession.
- **`selectedSkillID = 0` in WZ.** Treated as "no gate" (consistent with WZ's `0`-as-null convention). `omitempty` drops it from JSON. No condition emitted.
- **`CheckAutoStart` path.** Verified in the plan to honor the gate via `skipValidation = false`.

### Observability

`ValidateStartRequirements` already logs failed conditions at Debug level (`processor.go:182`). A failed `"skillLevel"` condition surfaces through that existing log line. No new metrics. If a surge in `"skillLevel"` failures becomes interesting, that is a separate follow-up.

### No schema migration

Zero database rows changed. Zero Kafka message shapes changed. Zero REST contract breaks. Pure data-flow addition.

## Summary

One field added to `atlas-data`'s `RestModel` and its parser. One field mirrored on `atlas-quest`'s `RestModel`. One constant added to `atlas-quest`'s validation model. One conditional append in `ValidateStartRequirements`. The existing `skillLevel` condition handled by query-aggregator does the rest.
