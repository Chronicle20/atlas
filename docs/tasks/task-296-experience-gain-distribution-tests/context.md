# task-296 — Implementation Context

## What this task is

Lock the EXPERIENCE_CHANGED distribution-type → `IncreaseExperienceConfig`
mapping under test, so the task-277 class of bug (picking `ITEM` for an
item-sourced EXP award, which renders "You have gained experience (+0)") fails
in CI instead of on a live client. Three production files, one service, no wire
change, no cross-service seam.

## Key files

| Path | Role |
|---|---|
| `services/atlas-channel/atlas.com/channel/kafka/consumer/character/consumer.go` | `announceExperienceGain` at line 362; the 14-arm `if/else` chain at 367-405; the 17-arg packet splat at 407-412 |
| `services/atlas-channel/atlas.com/channel/kafka/message/character/kafka.go` | the 14 `ExperienceDistributionType*` constants at 110-123; `ExperienceDistributions` at 173-177 |
| `services/atlas-channel/atlas.com/channel/socket/model/experience_status.go` | `IncreaseExperienceConfig`, **17** fields, all `bool`/`int32`/`byte` — comparable, which the whole-struct assertion depends on; its field comments are the authoritative client-line wording |
| `services/atlas-channel/atlas.com/channel/kafka/consumer/character/consumer_test.go` | `TestSnapshotHandlers` at 54-200 — the table-driven precedent; `package character`, imports at 1-19 |

Module root for every build/test: `services/atlas-channel/atlas.com/channel`.

## Decisions carried from design.md

- Pure `buildIncreaseExperienceConfig` lives **unexported in the consumer
  package**, not in `socket/model` (that would point a socket package at a
  Kafka message package) and not in a new file (its only caller is 15 lines
  below).
- `switch`, not an applier map — the map defeats the "byte-identical
  assignments" review property and moves nothing off the registry.
- Exhaustiveness via a hand-maintained `AllExperienceDistributionTypes` slice.
  The `exhaustive` linter needs a defined type, which would be a wire-adjacent
  refactor the PRD forecloses; reflection over a `const` block is impossible.
  **Accepted limit:** adding a constant while updating neither the slice nor the
  table is unguarded. Mitigation is proximity plus the doc comment.
- Whole-struct equality (`got != tc.want`), not `reflect.DeepEqual` and not
  field-by-field. Consequence: adding a field to `IncreaseExperienceConfig`
  does not break these tests — correct, since the suite asserts the mapping,
  not the struct's shape.
- No `default:` in the switch. Silent fall-through on an unknown type is the
  current behavior (FR-6), and a logging default would break purity.

## Ordering dependency — do not collapse

Task 2 (extract, chain verbatim) → Task 3 (tests green against the chain) →
Task 4 (convert to switch, same tests green). The two green runs on either side
of the conversion are the only evidence that the 14-arm hand-transcription
preserved behavior. Merging Task 2 and Task 4 destroys that evidence. Task 4's
`git diff` on `consumer_test.go` must be empty.

## Discovery that changed the design

Design §4.4 proposed a hypothetical `"EQUIP_ITEM"` for the unknown-type case.
The repo has a real one: `atlas-character` declares
`ExperienceDistributionTypeDeath = "DEATH"`
(`services/atlas-character/atlas.com/character/kafka/message/character/kafka.go:59`)
and emits it (`.../character/processor.go:840`), while atlas-channel has no
constant and no arm for it — so it is silently dropped today. The plan uses
`"DEATH"` instead: same intent, backed by evidence rather than invention.

`WhiteAndChat_PrimaryAwardShape`'s same-amount pairing is likewise pinned to
the producer: `services/atlas-character/atlas.com/character/character/processor.go:791-792`.

## Out of scope, deliberately

- `PartyBonusPercentage` and `QuestBonusRemainCount` have no distribution type.
  Asserted to stay zero; whether they *should* have one is a PRD open question.
- The last-wins primary-amount overwrite (`WHITE` then `YELLOW`) is pinned as
  documented behavior, not fixed. Deciding needs live-client evidence.
- FR-11 truncation on an `Amount` > 255 into a `byte` field is not exercised —
  it would document Go's conversion rules, not this mapping.
- `atlas-parties` and `atlas-monster-death` carry their own copies of the same
  14 constants. Not touched; this task is scoped to the atlas-channel consumer.

## Task sizing

Six tasks, each touching one or two files, all inside one service. Nothing was
left deliberately large. Task 3 is the biggest (a 19-case table) but is a single
new block in a single file with no discovery.
