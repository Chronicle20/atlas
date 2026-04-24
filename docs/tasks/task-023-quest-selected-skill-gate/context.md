# Context — Task 023: Quest `selectedSkillID` Start Gate

Quick reference for executing agents. Companion to `plan.md` and `design.md`.

## What this fixes

Skill-tutorial quests on shared job trees (Thief in this case) have identical start requirements in `Quest.wz/Check.img.xml`. The WZ attribute `QuestInfo.img/<id>/selectedSkillID` is the intended mutual-exclusion gate. Atlas drops that attribute at every layer. This task wires it through.

Concrete reproducer: a Bandit (job 420) who completed quest 2413 can currently start both quest 2418 ("Using Double Stab", selectedSkillID 4001334) and 2420 ("Using Lucky Seven", selectedSkillID 4001344). After this change, the server refuses to start whichever quest references a skill the character does not possess.

## Scope

- **Forward-looking only.** No migration, no cleanup sweep, no forfeiture of already-started duplicates.
- **Start phase only.** `ValidateEndRequirements` is untouched; once a quest is started it may still be completed.
- **`skipValidation = true` still bypasses.** The gate lives inside `ValidateStartRequirements`, so force-start paths continue to override it. This is intentional and consistent with existing force semantics.

## Services

| Service | Module | Root dir |
|---|---|---|
| atlas-data | `atlas-data` | `services/atlas-data/atlas.com/data/` |
| atlas-quest | `atlas-quest` | `services/atlas-quest/atlas.com/quest/` |
| atlas-query-aggregator | (not modified) | `services/atlas-query-aggregator/atlas.com/query-aggregator/` |

Run `go build`/`go test` from the **service root dir** (where `go.mod` lives), not from the repo root.

## Files touched

**atlas-data**
- `services/atlas-data/atlas.com/data/quest/rest.go` — add `SelectedSkillId uint32` field
- `services/atlas-data/atlas.com/data/quest/reader.go` — parse `selectedSkillID` in `ReadQuestInfo`
- `services/atlas-data/atlas.com/data/quest/reader_test.go` — extend existing test fixture + assertion

**atlas-quest**
- `services/atlas-quest/atlas.com/quest/data/quest/rest.go` — mirror `SelectedSkillId uint32` field
- `services/atlas-quest/atlas.com/quest/data/validation/model.go` — add `SkillCondition = "skillLevel"` constant
- `services/atlas-quest/atlas.com/quest/data/validation/processor.go` — extract `buildStartConditions`, add skill emission
- `services/atlas-quest/atlas.com/quest/data/validation/processor_test.go` — new file, unit tests for the helper

## Pre-verified invariants (do not re-verify)

- **Query-aggregator fully implements `skillLevel` already.** No changes needed there.
  - `SkillLevelCondition` constant: `services/atlas-query-aggregator/atlas.com/query-aggregator/validation/model.go:47`
  - Evaluator path: same file, lines 795-799, delegates to `ValidationContext.GetSkillLevel(skillId)`
  - Fetcher: `services/atlas-query-aggregator/atlas.com/query-aggregator/validation/context.go:103-115` (calls atlas-skills)
  - Wire docs: `services/atlas-query-aggregator/docs/rest.md:126`
- **Wire string is exactly `"skillLevel"`.** Case-sensitive. Must match for the condition to be accepted by query-aggregator's `SetType` dispatch (`validation/model.go:123-129`).
- **`CheckAutoStart` honors validation.** Already calls `startWithDefinition(..., skipValidation=false, ...)` at `services/atlas-quest/atlas.com/quest/quest/processor.go:787`. Auto-started quests will pick up the new gate for free.
- **`MockValidationProcessor` at `services/atlas-quest/atlas.com/quest/test/mocks.go:67-100` short-circuits the real validator.** Existing `quest` package tests pass through the mock, so they do **not** exercise the refactored `ValidateStartRequirements` body. This is why we write unit tests directly in `data/validation/`.

## Key design decisions (not negotiable)

1. **`SelectedSkillId` lives on top-level `RestModel`, not on `RequirementsRestModel`.** Mirrors WZ source layout (`QuestInfo.img`, not `Check.img`), and matches existing top-level flags like `AutoStart`, `SelectedMob`.
2. **Emitted condition is `>= 1`.** "Character has the skill" = level ≥ 1. Specific skill levels are irrelevant to the gate.
3. **New helper `buildStartConditions` is internal (lowercase).** Tests live in `package validation` (same package) so they can call it directly. No need to export.
4. **Item requirements with `Count <= 0` (removal semantics) continue to emit nothing at start time.** Preserved from the pre-refactor code; tests cover this (`TestBuildStartConditions_FameMesoItem`).

## Testing conventions followed

- `logrus/hooks/test.NewNullLogger()` for loggers in tests (see `services/atlas-data/atlas.com/data/quest/reader_test.go:7`).
- `reflect.DeepEqual` for slice comparisons (most Go teams' default).
- Table-free explicit tests when behavior-under-test is small and readable inline.
- Go 1.21+ features are OK; the project uses `slices`, generics, etc.

## Commit message style

Recent commits (see `git log`) use Conventional Commits:
- `feat(<service>): <summary>`
- `fix(<service>): <summary>`
- `refactor(<service>): <summary>`
- Multi-service changes use `feat(<service1>,<service2>): <summary>`.

Never skip hooks (no `--no-verify`). Never force-push. Each task's commit should be independent — Task 3 (refactor) must land cleanly on its own so a later bisect can distinguish refactor from feature.

## Execution environment

- Working in the isolated worktree at `.worktrees/task-023-quest-selected-skill-gate/` on branch `feature/task-023-quest-selected-skill-gate`.
- Plan artifacts (`design.md`, `plan.md`, `context.md`) are already committed on the branch (design was committed during Phase 2).

## Out of scope — do not do these things

- Do not touch atlas-query-aggregator.
- Do not touch atlas-skills.
- Do not change `ValidateEndRequirements`.
- Do not change the `CheckAutoStart` or `StartChained` paths — they already flow through `startWithDefinition` which calls the refactored `ValidateStartRequirements`.
- Do not add new Kafka topics, producers, or consumers.
- Do not add new REST endpoints.
- Do not write migration SQL or alter database schemas.
- Do not clean up already-corrupted character state (explicitly deferred in design).
