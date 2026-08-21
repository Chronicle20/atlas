# Mob Skill AoE Target Selection Fidelity — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-08-21
---

## 1. Overview

`atlas-monsters` applies mob-skill diseases, banish, and dispel to players through a single
shared target selector, `ProcessorImpl.getDiseaseTargets`
(`services/atlas-monsters/atlas.com/monsters/monster/processor.go:1282–1304`). That selector
currently uses the WZ bounding box (`lt`/`rb`) only as a *boolean switch*: if the skill declares
any non-zero box coordinate, the selector fetches **every character in the field** via
`_map.CharacterIdsInFieldProvider` and never compares a character's position against the box.
The box coordinates are read correctly elsewhere in the same file — the ally-heal AoE at
`processor.go:1200–1210` translates `LtX/LtY/RbX/RbY` to the caster's position and tests each
monster against it — so the data is present and simply is not applied to player targeting.

The consequence is that any mob skill with a bounding box hits the entire map instead of the
rectangle the WZ data describes. A Horntail seduce, a Zakum-arm dispel, or a banish that should
catch the players standing in front of the mob currently catches everyone on the field,
regardless of distance. Two smaller divergences compound it: the `count` cap is applied to
*every* AoE disease (the reference server caps only SEDUCE, letting other box diseases hit
everyone inside the rectangle), and the selector randomly shuffles candidates before truncating,
which makes the sample non-deterministic and untestable. A third, narrower defect: the
single-target branch is gated on `!HasBoundingBox() && Count() <= 1`, so a boxless skill with
`count > 1` falls through to the field-wide AoE branch instead of hitting only the controller.

This task fixes the shared selector for all of its callers — debuff/disease, banish, and dispel —
using seduce (mob skill 128) as the motivating case. Fidelity is defined against the local
reference implementation (Cosmic `src/main/java/server/life/MobSkill.java:384–406, 452–457`),
which is stated as reference-server parity and **not** as an IDA/WZ-verified reproduction of
GMS v83 behavior. What GMS itself did for seduce target *ordering* remains unverified; this PRD
does not claim to resolve that and records it as an open question.

## 2. Goals

Primary goals:

- Select AoE mob-skill targets from the WZ `lt`/`rb` rectangle translated to the casting mob's
  position, instead of from the whole field.
- Apply the `count` cap with reference-server semantics: SEDUCE only.
- Make target selection deterministic and unit-testable (no `rand.Shuffle` in the selection path).
- Route boxless skills to the single-target (controller) path regardless of `count`.
- Give `atlas-monsters` a character-position lookup, following the existing `atlas-maps`
  mist-tick precedent, without making an AoE cast pathologically expensive.
- Apply the fix once, in the shared selector, so debuff, banish, and dispel all inherit it.

Non-goals:

- Changing what any disease *does* once applied (that is `atlas-buffs`' side of the pipeline).
- The Holy Shield immunity-list gap (`atlas-buffs/.../character/immunity.go` omits `STOP_MOTION`
  and `FEAR`) — tracked separately in `docs/research/missing-features/monsters-and-bosses.md` §6.
- Horntail-specific or any other boss-specific scripted encounter logic. This task adds no
  per-mob special cases.
- The client-initiated `MOB_BANISH_PLAYER` path (mob *attack*-borne banish), which is
  decode-and-log only in `atlas-channel` — research doc §2.
- Proving GMS-canonical seduce ordering from the client binary or WZ data. That evidence pass is
  explicitly deferred (see §9).
- Mirroring the bounding box when the mob faces left. The reference implementation does not flip
  the rectangle by facing, and whether the v83 client did is unverified; this task matches the
  reference and records the question.

## 3. User Stories

- As a player fighting a boss with a seduce skill, I want to be seduced only when I am standing
  inside the skill's area, so that positioning is a meaningful defensive choice rather than
  irrelevant.
- As a player on the far side of a large map, I want a mob's AoE debuff not to reach me, so that
  distance from the mob protects me the way the game data says it should.
- As a party fighting a mob whose skill declares `count: 2`, I want exactly two of us seduced and
  the rest of the party's non-seduce debuffs applied to everyone in range, so that the encounter
  behaves the way the WZ data describes.
- As a developer, I want target selection to be deterministic given a fixed set of characters and
  positions, so that I can write a fixture test asserting exactly who was targeted.

## 4. Functional Requirements

### 4.1 Bounding-box target selection

- **FR-1.1** When a mob skill declares a bounding box (`HasBoundingBox()` true), the selector
  MUST build the rectangle by translating the skill's `LtX/LtY/RbX/RbY` by the casting mob's
  `X()`/`Y()`, matching the translation already used for ally-heal AoE at
  `processor.go:1200–1210` and the reference `calculateBoundingBox`
  (Cosmic `MobSkill.java:452–457`).
- **FR-1.2** A character is a candidate iff its `(x, y)` falls inside that rectangle, using the
  same inclusive comparison form as the existing ally-heal check
  (`dx >= LtX && dx <= RbX && dy >= LtY && dy <= RbY`, where `dx`/`dy` are the character's
  coordinates minus the mob's).
- **FR-1.3** The rectangle MUST NOT be mirrored based on the mob's facing direction. (Reference
  parity; see §9.)
- **FR-1.4** Characters whose position cannot be resolved MUST be excluded from the candidate
  set, and the failure logged at warn level with the character id and the mob's unique id. A
  single unresolvable character MUST NOT abort the cast for the others.

### 4.2 Single-target path

- **FR-2.1** When a mob skill declares **no** bounding box, the selector MUST return the mob's
  controlling character only — regardless of the skill's `count`. This replaces the current
  `!HasBoundingBox() && Count() <= 1` condition, which leaks boxless multi-count skills into the
  field-wide branch.
- **FR-2.2** If the mob has no controller (`ControlCharacterId() == 0`), the selector MUST return
  an empty set and the caller MUST make no Kafka emission.
- **FR-2.3** The single-target path MUST make no character-position REST call.

### 4.3 Count semantics

- **FR-3.1** The `count` cap MUST be applied **only** when the skill's type is SEDUCE
  (`monster.SkillTypeSeduce`, 128). All other AoE diseases, plus banish and dispel, apply to
  every candidate inside the rectangle with no cap. This matches
  Cosmic `MobSkill.applyDisease` (`MobSkill.java:384–402`), where the `i < count` guard sits
  inside the `disease.equals(Disease.SEDUCE)` branch only.
- **FR-3.2** When the skill is SEDUCE and `count` is 0, no cap is applied (all in-box candidates
  are targeted). A `count` of 0 in the WZ data means "unspecified," not "target nobody."
- **FR-3.3** When the number of in-box candidates is at or below `count`, all of them are
  targeted and no truncation logic runs.

### 4.4 Ordering determinism

- **FR-4.1** `rand.Shuffle` MUST be removed from the selection path. Selection order MUST be the
  stable order in which candidate character ids are returned by the field listing, filtered by
  the box test, matching the reference server's "first `count` in iteration order."
- **FR-4.2** The selector's output MUST be a pure function of (mob position, skill data, the
  ordered candidate list with positions). Given identical inputs, two calls return identical
  output.

### 4.5 Character position lookup

- **FR-5.1** `atlas-monsters` gains a read-only client for character position, following the
  shape already established by `services/atlas-maps/atlas.com/maps/character` (a minimal
  `RestModel` projecting only the consumed fields, a `requestById` against
  `requests.RootUrlFor(ctx, "CHARACTERS")`, and a `Processor` interface with a mock).
- **FR-5.2** The projection MUST declare only the fields consumed. `x` and `y` are required.
  `hp` is included only if a liveness filter is adopted (see FR-5.5).
- **FR-5.3** Position lookups for one cast MUST be issued with bounded concurrency rather than
  strictly serially, so that a crowded field does not serialize N round-trips into the mob's
  skill execution path. The bound is a design decision, not fixed here.
- **FR-5.4** A lookup failure for one character MUST NOT fail the cast (see FR-1.4). A failure to
  list the field's characters at all MUST result in an empty target set and an error log, matching
  today's behavior at `processor.go:1294–1298`.
- **FR-5.5** Whether dead characters (`hp == 0`) are excluded from AoE candidates is a design
  decision. The current selector does not filter them; the reference server's
  `getPlayersInRange` does not either. If no positive evidence is found, preserve current
  behavior and do not filter.

### 4.6 Caller behavior (unchanged contracts)

- **FR-6.1** `executeDebuff`, `executeBanish`, and `executeDispel` continue to call the shared
  selector and continue to emit exactly one Kafka command per returned character id. Their
  emission logic, topics, and payloads are unchanged by this task.
- **FR-6.2** The selector MUST receive enough information to apply FR-3.1 — that is, the mob
  skill's type/id, which it does not receive today. Threading that through is a design decision;
  the callers already hold `skillId`.

## 5. API Surface

No new externally-facing endpoints. One new outbound REST dependency:

- `atlas-monsters` → `atlas-character`: `GET {CHARACTERS}/characters/{characterId}`, consuming
  the JSON:API `characters` resource and projecting `x`, `y` (and `hp` only if FR-5.5 adopts the
  liveness filter). Identical in shape to the call `atlas-maps` already makes
  (`services/atlas-maps/atlas.com/maps/character/requests.go`).

If the design instead introduces a positional/bulk query on `atlas-maps` or `atlas-character`,
that endpoint's contract must be specified in the design document; this PRD does not mandate one.
Any such endpoint follows JSON:API conventions and is tenant-scoped through the standard context
propagation.

## 6. Data Model

No persisted schema changes. No new entities, no migrations.

In-memory only: the selector gains a candidate representation carrying `(characterId, x, y)` for
the duration of one cast. Mob-skill data (`mobskill.Model`) already exposes everything needed —
`Count()`, `HasBoundingBox()`, `LtX/LtY/RbX/RbY`
(`services/atlas-monsters/atlas.com/monsters/monster/mobskill/model.go:59–86`) — and is
unchanged.

## 7. Service Impact

| Service | Change |
|---|---|
| `atlas-monsters` | Primary. `getDiseaseTargets` rewritten per §4.1–§4.4; new read-only character-position client package per §4.5; callers (`executeDebuff`/`executeBanish`/`executeDispel`) adjusted only to pass the skill type through. |
| `atlas-character` | Consumed only. No change expected — it already serves `x`/`y` on the `characters` resource (`services/atlas-character/atlas.com/character/character/rest.go:49–50`). |
| `atlas-maps` | Consumed only, unchanged. Still supplies the field's character id list. Its `character` package is the pattern being followed, not modified. |
| `atlas-buffs` | Untouched. Receives the same `apply disease` commands, only for a corrected target set. |
| `libs/atlas-constants` | Read-only. `monster.SkillTypeSeduce` (`monster/skill.go:40`) already exists; reuse it rather than introducing a literal 128. |

## 8. Non-Functional Requirements

- **Performance.** An AoE cast on a crowded field must not serialize one REST round-trip per
  character into the skill execution path. Bounded concurrency (FR-5.3) is required; a
  short-lived position cache is permitted if the design justifies its staleness window against
  the fact that positions change continuously during combat.
- **Correctness under failure.** Partial position-resolution failure degrades to a smaller target
  set, never to a crash or a field-wide fallback. Silently falling back to "everyone in the
  field" when positions are unavailable is explicitly forbidden — that is the bug being fixed.
- **Determinism.** No `math/rand` in the selection path (FR-4.1), so tests can assert an exact
  target list.
- **Multi-tenancy.** All REST calls are made through the tenant-scoped context already carried on
  `ProcessorImpl` (`p.ctx`), consistent with every other outbound call in the file.
- **Observability.** Log at debug the mob unique id, skill id, candidate count, and final target
  count per cast, so a live encounter can be diagnosed without a repro harness. Log at warn on
  per-character position-resolution failure.
- **Test builders.** Test setup uses the project's Builder pattern; no `*_testhelpers.go`
  test-only constructors.

## 9. Open Questions

- **GMS-canonical seduce ordering (unverified).** Whether v83 GMS ordered seduce targets by
  damage rank, proximity, party slot, or map-object insertion order is not established. This task
  implements reference-server parity (stable iteration order) as a documented approximation, not
  as a verified reproduction. Resolving it needs an IDA/WZ sweep — carried forward from
  `docs/research/missing-features/monsters-and-bosses.md` §8 and its "Unverified / needs deeper
  data" entry.
- **Facing-direction mirroring (unverified).** Real MapleStory skill rectangles are commonly
  mirrored when the caster faces left. The reference implementation does not mirror. Whether the
  v83 mob-skill path did is unknown; FR-1.3 chooses reference parity pending evidence.
- **Dead-character filtering (FR-5.5).** Neither the current code nor the reference filters by
  HP. Left as a design decision, defaulting to no filter.
- **Position freshness.** `atlas-character`'s `x`/`y` reflects the last movement update it
  processed. How stale that can be during active combat has not been measured, and it bounds the
  achievable fidelity of box targeting regardless of implementation quality.
- **Bulk vs. per-character lookup.** This PRD permits either. If per-character N-calls prove
  unacceptable in the design phase, introducing a positional query is in scope for the design,
  not a follow-up task.

## 10. Acceptance Criteria

- [ ] `getDiseaseTargets` no longer calls `rand.Shuffle`; `math/rand` is removed from the file if
      it has no other use.
- [ ] A skill with a bounding box targets only characters whose position falls inside the
      mob-translated rectangle. Covered by a unit test with at least one character inside the box
      and one outside, asserting the exact returned id list.
- [ ] A skill with no bounding box and `count > 1` returns exactly the controlling character.
      Covered by a unit test.
- [ ] A mob with no controller and a boxless skill returns an empty target set and emits no
      Kafka command. Covered by a unit test.
- [ ] A non-SEDUCE AoE disease with `count: 2` and four in-box candidates targets all four.
      Covered by a unit test.
- [ ] A SEDUCE skill with `count: 2` and four in-box candidates targets exactly two, in stable
      candidate order (the same two on repeated invocation with identical input). Covered by a
      unit test.
- [ ] A SEDUCE skill with `count: 0` and three in-box candidates targets all three.
- [ ] `executeBanish` and `executeDispel` inherit box-scoped targeting with no cap; covered by at
      least one test each asserting an out-of-box character is not targeted.
- [ ] A position-lookup failure for one character excludes only that character; the remaining
      in-box characters are still targeted. Covered by a unit test using the mock position
      processor.
- [ ] The SEDUCE branch uses `monster.SkillTypeSeduce` from `libs/atlas-constants`, not a literal.
- [ ] The new character-position client has a `mock/` implementation consistent with the other
      `mock/` packages in `atlas-monsters`.
- [ ] `docs/research/missing-features/monsters-and-bosses.md` §8 and the "Unverified" bullet are
      updated to reflect what this task fixed and what remains unverified.
- [ ] Flagless `tools/verify.sh` exits 0.
- [ ] Code review completed before the PR is opened.
