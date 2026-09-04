# Experience Gain Distribution Mapping — Test Coverage — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-09-04
Source: [Chronicle20/atlas#1630](https://github.com/Chronicle20/atlas/issues/1630)
---

## 1. Overview

`announceExperienceGain` in `services/atlas-channel/atlas.com/channel/kafka/consumer/character/consumer.go:362-420` is the seam where an `EXPERIENCE_CHANGED` Kafka event becomes what the player actually reads on screen. It walks the event's `[]ExperienceDistributions` and folds each entry into a `model.IncreaseExperienceConfig`, whose 19 fields are then splatted, positionally, into `CharacterStatusMessageOperationIncreaseExperienceBody`. Each distribution type name is a short, plausible-sounding string (`ITEM`, `PARTY`, `CHAT`) bound to a non-obvious client render line. There is no test over this mapping — `consumer_test.go` exercises the snapshot handlers but not a single distribution shape — so the entire correspondence is held by code reading alone.

This is not a hypothetical gap. Task-277's stored-EXP redeem tagged its distribution `ITEM` on the reasonable assumption that `ITEM` meant "EXP that came from an item". It does not: `ITEM` populates `c.ItemBonusEXP`, the client's *Equip Item Bonus EXP* modifier line, and only `WHITE` / `YELLOW` / `CHAT` set the primary `c.Amount`. On a live GMS 83.1 client the player saw `You have gained experience (+0)` followed by `Equip Item Bonus EXP (+100000)`. The EXP was awarded correctly, so nothing failed loudly; the wrong choice survived spec, design, plan, implementation, a full green gate, and code review, and was caught only by a human looking at the client. It was fixed in `e82d77c42`, with the diagnosis in `docs/tasks/task-277-stored-exp-items/bug-redeem-renders-as-item-bonus.md`.

This task closes the seam. It extracts the mapping loop into a pure function, converts the 14-arm `if/else` chain to a `switch` whose arms are annotated with the client line each one renders, and adds a table-driven test asserting the exact `IncreaseExperienceConfig` field every distribution type populates — plus an exhaustiveness check that fails when a new distribution type is introduced without a corresponding test case. The mapping's observable behavior does not change.

## 2. Goals

Primary goals:

- Every `ExperienceDistributionType` has a test asserting which `IncreaseExperienceConfig` field(s) it populates and with what value.
- The mapping is a pure, directly-callable function requiring no session, writer, or Kafka harness to test.
- Adding a new `ExperienceDistributionType` constant without covering it fails the test suite.
- A future caller choosing a distribution type can read, at the point of choice, which client line it renders — so the task-277 name-plausibility mistake is not available.
- The multi-distribution shape that every visible award now emits (`WHITE` + `CHAT`, appended by `AwardExperience` in `atlas-character/.../processor.go:787-794`) is pinned.

Non-goals:

- Changing any distribution → field mapping. Current behavior is locked in and asserted, not corrected.
- Testing the `session.Announce` / writer path, the 17-argument call into `CharacterStatusMessageOperationIncreaseExperienceBody`, or its argument order.
- Any change to the `atlas-character` producer side, or an audit of which distribution types existing award sites emit.
- Packet-level or wire-level verification, coverage-matrix promotion, or version gating of the v95+ fields.
- Any new API endpoint, database entity, migration, or Kafka contract change.

## 3. User Stories

- As a service developer adding a new EXP award path, I want each distribution type to state which client line it renders, so that I do not pick one by the plausibility of its name.
- As a service developer, I want a test that fails when I introduce a new `ExperienceDistributionType` without mapping it, so that a silently-unrendered distribution cannot ship.
- As a reviewer of an EXP-related change, I want the distribution mapping asserted in a table I can read in one screen, so that a wrong choice is visible in the diff rather than only on a live client.
- As a maintainer refactoring the consumer, I want the mapping callable as a pure function, so that I can test it without constructing a session, a tenant, or a writer producer.

## 4. Functional Requirements

### 4.1 Extract the mapping into a pure function

- FR-1. A new unexported function is added to `services/atlas-channel/atlas.com/channel/kafka/consumer/character/consumer.go`:

  ```go
  func buildIncreaseExperienceConfig(ds []character2.ExperienceDistributions) model2.IncreaseExperienceConfig
  ```

- FR-2. The function is pure: no logger, no context, no session, no writer, no package-level state. Given the same input slice it returns the same value, and it mutates nothing the caller owns.
- FR-3. `announceExperienceGain` calls it in place of the inline loop. Its signature, its currying shape, its `session.Announce` call, its argument order, and its error handling and log message are unchanged.
- FR-4. A `nil` or empty distribution slice returns the zero `IncreaseExperienceConfig`, matching today's behavior.

### 4.2 Convert the chain to an annotated switch

- FR-5. The 14-arm `if/else` chain becomes a `switch d.ExperienceType { ... }` over the same constants, producing byte-identical field assignments.
- FR-6. An unrecognized `ExperienceType` value falls through with no assignment — matching today's behavior, where an unmatched string reaches the end of the chain untouched. No error, no log, no panic.
- FR-7. Each `case` carries a comment naming the client line that field renders. The `ITEM` arm additionally records the task-277 trap explicitly — that it is the *Equip Item Bonus EXP* modifier line, not "EXP that came from an item".

### 4.3 Preserve current behavior exactly

The following are current behaviors. They are asserted by the new tests and deliberately not changed:

- FR-8. `YELLOW` assigns `White = false` and `Amount`. On a zero-value config the `White = false` write is a no-op; it is meaningful only when a prior `WHITE` in the same slice set it true.
- FR-9. The primary amount is last-wins. Given `WHITE` followed by `YELLOW` in one slice, `Amount` holds the `YELLOW` value and `White` is `false`. The function does not detect, reject, or accumulate multiple primary-amount distributions.
- FR-10. `MONSTER_EVENT` and `PLAY_TIME` both write `MobEventBonusPercentage`; `PLAY_TIME` additionally writes `PlayTimeHour` from `Attr1`. Both overwrite each other last-wins when present together.
- FR-11. Narrowing conversions are unchanged: `int32(d.Amount)` for the `int32` fields, `byte(d.Amount)` / `byte(d.Attr1)` for the `byte` fields. Overflow behavior on a value exceeding the target width is Go's defined truncation and is not guarded.
- FR-12. `PartyBonusPercentage` and `QuestBonusRemainCount` are populated by no distribution type today and remain unpopulated. The tests assert they stay zero.

### 4.4 The complete mapping under test

Every case asserts the full resulting `IncreaseExperienceConfig` — the populated field(s) at the expected value and every other field at zero — so a stray write to an unrelated field fails the case.

| Distribution type | Field(s) populated | Source | Client line rendered |
|---|---|---|---|
| `WHITE` | `Amount`, `White = true` | `Amount` | "You have gained experience", white text |
| `YELLOW` | `Amount`, `White = false` | `Amount` | "You have gained experience", yellow text |
| `CHAT` | `Amount`, `InChat = true` | `Amount` | Chat-window experience line |
| `MONSTER_BOOK` | `MonsterBookBonus` | `Amount` | Right side, yellow. Bonus Event EXP |
| `MONSTER_EVENT` | `MobEventBonusPercentage` | `byte(Amount)` | In chat, pink. Bonus EXP per 3rd monster |
| `PLAY_TIME` | `MobEventBonusPercentage`, `PlayTimeHour` | `byte(Amount)`, `byte(Attr1)` | Right side, yellow. Bonus EXP for hunting over N hrs |
| `WEDDING` | `WeddingBonusEXP` | `Amount` | Right side, yellow. Bonus Wedding EXP |
| `SPIRIT_WEEK` | `QuestBonusRate` | `byte(Amount)` | Earned 'Spirit Week Event' bonus EXP |
| `PARTY` | `PartyBonusExp`, `PartyBonusEventRate` | `Amount`, `byte(Attr1)` | Right side, yellow. Bonus Event Party EXP |
| `ITEM` | `ItemBonusEXP` | `Amount` | Right side, yellow. **Equip Item Bonus EXP** |
| `INTERNET_CAFE` | `PremiumIPExp` | `Amount` | Right side, yellow. Internet Cafe EXP Bonus |
| `RAINBOW_WEEK` | `RainbowWeekEventEXP` | `Amount` | Right side, yellow. Rainbow Week Bonus Event EXP |
| `PARTY_RING` | `PartyEXPRingEXP` | `Amount` | v95+ only |
| `CAKE_PIE` | `CakePieEventBonus` | `Amount` | v95+ only |

- FR-13. All 14 types are covered, including `PARTY_RING` and `CAKE_PIE`. The mapping is version-independent — the test asserts the config field, not the wire — so these are covered identically to the rest. Their case names note the v95+ render availability.
- FR-14. Cases with an `Attr1`-sourced field (`PLAY_TIME`, `PARTY`) use an `Attr1` value distinct from the case's `Amount`, so a mixup between the two sources fails the case.
- FR-15. Every case uses a distinct, non-zero, easily-identifiable `Amount`, so a mapping that writes the correct value into the wrong field is caught by the zero-check on the other fields.

### 4.5 Multi-distribution cases

- FR-16. `WHITE` + `CHAT` — the pair `AwardExperience` appends for a `showEffect` award (`atlas-character/.../processor.go:787-794`), and the shape every visible award now emits. Asserts `Amount` set once, `White = true`, `InChat = true`.
- FR-17. A primary + bonus combination (e.g. `WHITE` + `PARTY` + `ITEM`) asserting that independent fields accumulate without interfering.
- FR-18. `WHITE` followed by `YELLOW` — pins the last-wins overwrite of FR-9 as documented behavior.
- FR-19. An empty slice returns the zero config (FR-4).
- FR-20. A slice containing an unrecognized type string alongside a valid one — asserts the unknown entry is ignored and the valid entry still applies (FR-6).

### 4.6 Exhaustiveness enforcement

- FR-21. A package-level slice enumerating every distribution type is added to `services/atlas-channel/atlas.com/channel/kafka/message/character/kafka.go`, adjacent to the constants:

  ```go
  var AllExperienceDistributionTypes = []string{ /* all 14, in declaration order */ }
  ```

- FR-22. The test iterates `AllExperienceDistributionTypes` and fails, naming the offending type, if any entry has no case in the mapping table. A new constant added to the block but not to the slice, or to the slice but not to the test table, fails.
- FR-23. The test also fails if the table contains a case for a type not present in `AllExperienceDistributionTypes`, catching a removed or renamed constant.

### 4.7 Documentation of the mapping at the point of choice

- FR-24. A comment block above the `ExperienceDistributionType*` constants in `kafka.go` records, per type, the `IncreaseExperienceConfig` field it populates and the client line it renders — matching the FR-4.4 table. This is where a producer-side developer picks a value, so it is where the trap must be visible.
- FR-25. The comment explicitly records that only `WHITE`, `YELLOW`, and `CHAT` set the primary "You have gained experience" amount, and that every other type is a secondary modifier line.

## 5. API Surface

None. No HTTP endpoint, JSON:API resource, Kafka topic, message schema, or event contract is added, removed, or modified. `ExperienceDistributions` and `ExperienceChangedStatusEventBody` keep their exact wire shape; `AllExperienceDistributionTypes` is a Go-side enumeration of already-published string constants and changes nothing observable to any producer or consumer.

## 6. Data Model

None. No entity, table, column, index, or migration. No persisted state is read or written by the code under test — `buildIncreaseExperienceConfig` is a pure in-memory transform. No `tenant_id` scoping applies because nothing is persisted or queried.

## 7. Service Impact

### atlas-channel (only affected service)

| File | Change |
|---|---|
| `atlas.com/channel/kafka/consumer/character/consumer.go` | Extract `buildIncreaseExperienceConfig`; convert the chain to an annotated `switch`; `announceExperienceGain` calls the new function. |
| `atlas.com/channel/kafka/message/character/kafka.go` | Add `AllExperienceDistributionTypes`; add the mapping doc comment above the distribution constants. |
| `atlas.com/channel/kafka/consumer/character/consumer_test.go` | Add the table-driven mapping test, the multi-distribution cases, and the exhaustiveness check. |

No other service compiles against any of these symbols. `atlas-character` produces the `EXPERIENCE_CHANGED` event and is untouched — its distribution choices are out of scope (§2 non-goals). `services/atlas-channel/atlas.com/channel/skill/handler/heal/heal.go:224` and its test reference `ExperienceDistributionTypeWhite` but are unaffected by an added `var` and comments.

## 8. Non-Functional Requirements

- **Performance.** The extracted function is called once per `EXPERIENCE_CHANGED` event on the same code path as today. A `switch` over string constants is no slower than the equivalent `if/else` chain. No new allocation is introduced — the config is a value type returned by copy, as it is built today.
- **Behavioral risk.** Zero intended change to what the client renders. The refactor is behavior-preserving by construction (FR-3, FR-5) and the new tests are written to pass against the pre-refactor mapping, so a divergence introduced during extraction fails the suite.
- **Multi-tenancy.** Not applicable — the function is tenant-agnostic and touches no tenant-scoped state. The existing handler's `sc.Is(tenant.MustFromContext(ctx), ...)` guard is upstream of the extraction and unchanged.
- **Observability.** No new logging. The existing `l.WithError(...).Errorf("Unable to announce experience gain to character [%d].", ...)` stays in `announceExperienceGain`, outside the pure function.
- **Security.** No new input surface. The function already accepts attacker-influenceable amounts only insofar as the upstream event does; the narrowing conversions of FR-11 are unchanged and out of scope.
- **Test conventions.** Table-driven per DOM-20, matching the existing `TestSnapshotHandlers` style in the same file. Use the project Builder pattern for any model construction; no `*_testhelpers.go` test-only constructors.
- **Determinism.** The tests are pure and require no Docker, Kafka, database, session, or network. They run under `go test ./...` for the module and under `tools/verify.sh --quick`.

## 9. Open Questions

None blocking. Every interview question was resolved:

- Extraction shape → unexported pure func in the consumer package (FR-1).
- Chain style → convert to `switch` (FR-5).
- Questionable behaviors (`YELLOW`'s `White = false`; last-wins `Amount`) → locked in and asserted, not fixed (FR-8, FR-9).
- Recurrence guard → both per-arm client-line comments (FR-7) and a doc note at the constants (FR-24).
- Exhaustiveness → registry slice + iteration check (FR-21, FR-22).
- Test scope → mapping only; no writer/session fake (§2 non-goals).
- v95+ types → covered identically to the rest (FR-13).

Deferred, not in scope, and worth a separate issue if anyone wants them:

- Whether the last-wins primary-amount overwrite (FR-9) should be an error. Deciding needs live-client evidence for the multi-primary case, which no current caller emits.
- Whether `PartyBonusPercentage` and `QuestBonusRemainCount` (FR-12) should have distribution types at all — they are reachable in the config struct and the packet body but by no event.
- An audit of the `atlas-character` producer side to confirm every existing award site picks the distribution type it means.

## 10. Acceptance Criteria

- [ ] `buildIncreaseExperienceConfig(ds []character2.ExperienceDistributions) model2.IncreaseExperienceConfig` exists in `consumer.go`, is pure, and takes no logger, context, session, or writer.
- [ ] `announceExperienceGain` calls it; its signature, currying, `session.Announce` call and argument order, and error handling are unchanged.
- [ ] The mapping is a `switch`, not an `if/else` chain, and produces identical field assignments for all 14 types.
- [ ] Every `switch` arm has a comment naming the client line it renders; the `ITEM` arm names the task-277 trap explicitly.
- [ ] `AllExperienceDistributionTypes` exists in `kafka.go` and lists all 14 constants.
- [ ] A doc comment above the distribution constants records the per-type field and client line, and states that only `WHITE`/`YELLOW`/`CHAT` set the primary amount.
- [ ] A table-driven test covers all 14 types, each asserting the full resulting config — populated fields at expected values, all others zero.
- [ ] `PLAY_TIME` and `PARTY` cases use an `Attr1` value distinct from their `Amount`.
- [ ] Multi-distribution cases cover `WHITE`+`CHAT`, a primary+bonus combination, `WHITE`→`YELLOW` last-wins, the empty slice, and an unknown type alongside a valid one.
- [ ] The test fails, naming the type, when a constant is added to `AllExperienceDistributionTypes` without a table case, and fails when the table has a case not in the slice. Verified by temporarily introducing each condition and observing the failure.
- [ ] Assigning `ITEM` to `c.Amount` instead of `c.ItemBonusEXP` fails the suite — i.e. the test would have caught the task-277 bug. Verified by temporarily making that edit and observing the failure.
- [ ] No mapping behavior changed: the new tests pass against the pre-refactor implementation as well as the post-refactor one.
- [ ] `go build ./...` and `go test ./...` pass in `services/atlas-channel`.
- [ ] Flagless `tools/verify.sh` exits 0.
- [ ] Code review passes before the PR opens.
