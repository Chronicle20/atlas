# Shadow Stars (Night Lord 4121006) — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-07-10
---

## 1. Overview

Shadow Stars (skill id `4121006`) is a Night Lord 4th-job self-buff. On cast, the
player picks one throwing-star type from inventory; for the buff duration the
Night Lord throws that star with **no per-attack consumption**, at a one-time
cast cost of a large batch of that same star (WZ `bulletCount`, 200 in the
reference data). The chosen star is communicated to every observer via the
`SHADOW_CLAW` temporary stat, whose value is the star's item id, so remote
clients render the correct projectile.

Today the buff *applies* but none of the star handling works. The client's
chosen star id is decoded off the wire and then dropped; the `SHADOW_CLAW`
statup is emitted with amount `0` (so the client is never told which star to
throw); and the per-attack projectile-consumption gate has no `SHADOW_CLAW`
carve-out for claws, so stars are still consumed on every ranged attack. The net
effect is a buff that lights up in the UI but changes no behavior — and if the
per-attack gate is fixed in isolation without a cast cost, the skill would
degrade into "infinite free stars, zero cost," which is worse than the current
state.

This task makes Shadow Stars behave correctly end to end: plumb the runtime star
id from the client packet into the `SHADOW_CLAW` statup, stop consuming stars per
attack while the buff is active, and charge the one-time cast cost of the chosen
star.

Cosmic reference for the intended behavior: `RangedAttackHandler.java:182-184`
(ranged attack consults the Shadow Stars buff to decide the projectile item and
skip normal consumption).

## 2. Goals

Primary goals:

- The `SHADOW_CLAW` temporary stat sent to clients carries the **client-chosen
  star item id**, not `0`, so both the caster and observers render/throw the
  correct star.
- While the Shadow Stars buff is active, ranged attacks with a **claw** do **not**
  consume throwing stars per attack (mirrors the existing `SOUL_ARROW` carve-out
  for bow/crossbow).
- The one-time cast cost — `bulletCount` (200) of the chosen star type — is
  consumed from inventory at cast, so the buff is not free.
- The chosen star id is validated as a throwing-star the player actually owns
  before it is trusted (it now drives both a client-visible throw and a bulk
  consume).

Non-goals:

- No change to how any *other* buff (Soul Arrow, Shadow Partner, etc.) is
  produced or gated. Shadow Partner interaction with Shadow Stars is limited to
  the existing `computeCount` doubling behavior, which becomes moot once the
  claw+`SHADOW_CLAW` gate short-circuits consumption.
- No new generic multi-item / batch-consume framework. If P8 introduces a
  generic consume path, P7's cast-cost consume can later be re-homed onto it;
  P7 ships a working, self-contained consume so it never ships the free-stars
  exploit.
- No changes to Night Walker's analogous mechanics or to any non-4121006 skill.
- No change to `reader.go`'s static WZ→statup mapping other than what is required
  to carry a placeholder the channel layer overwrites (atlas-data cannot know the
  runtime star id).

## 3. User Stories

- As a Night Lord, when I cast Shadow Stars and pick a star, I want that star to
  be thrown automatically for the buff's duration without draining my inventory
  per attack, so the skill behaves as designed.
- As a Night Lord, I want the one-time star cost deducted at cast, so the buff
  has the intended cost and my star count reflects it.
- As another player in the map, I want to see the Night Lord throwing the correct
  star (the one they chose), so remote rendering matches the caster's client.
- As a server operator, I want a client that sends a bogus or unowned star id to
  be rejected rather than have it injected into a buff or trigger consumption of
  an unintended item.

## 4. Functional Requirements

### FR-1 — Expose the decoded star id

- `SkillUsageInfo` MUST expose the decoded `spiritJavelinItemId` via a public
  getter (e.g. `SpiritJavelinItemId() uint32`), so the channel skill-use
  orchestrator can read it. The field is already decoded at
  `libs/atlas-packet/model/skill_usage_info.go:32-34` and settable via the
  builder (`SetSpiritJavelinItemId`), but has no getter today.

### FR-2 — Carry the chosen star id in SHADOW_CLAW

- When Shadow Stars (`4121006`) is cast, the `SHADOW_CLAW` statup delivered to
  the client MUST have its value/amount equal to the **chosen star item id**
  (`SpiritJavelinItemId()`), not `0`.
- `SHADOW_CLAW` is wire-encoded as an int foreign value
  (`libs/atlas-packet/model/character_temporary_stat.go:124`,
  `ValueAsIntForeignValueWriter`), so the statup amount is exactly the value the
  client reads as the star id.
- atlas-data's `reader.go:298` MAY continue to emit `SHADOW_CLAW` with a
  placeholder amount of `0` (it has no access to the runtime choice); the
  channel layer is responsible for supplying the real value before the buff is
  applied. The exact injection mechanism (statup rewrite before `buff.Apply`
  vs. an explicit parameter) is a **design-phase decision** (see Open Questions).

### FR-3 — Suppress per-attack star consumption while buffed

- In the projectile-consumption gate
  (`services/atlas-channel/.../socket/handler/character_attack_projectile.go`),
  when the caster's equipped weapon is a **claw** and the caster has an active
  `SHADOW_CLAW` (`TemporaryStatTypeShadowClaw`) buff, projectile consumption MUST
  be skipped (return no plan / not-consuming), analogous to the existing
  `SOUL_ARROW` skip for bow/crossbow at line 107.
- This gate MUST NOT affect claw attacks when `SHADOW_CLAW` is inactive, nor
  bow/crossbow/gun attacks.

### FR-4 — Charge the one-time cast cost

- On casting Shadow Stars, the server MUST consume `bulletCount` (WZ-defined;
  200 in reference data) throwing stars of the **chosen star type**
  (`SpiritJavelinItemId()`) from the caster's USE inventory.
- The consume MUST target the chosen star's item id (not an arbitrary/first
  star), drawing across slots if a single slot is insufficient (reuse the
  existing reservation→consume machinery where practical).
- If the player lacks enough of the chosen star to pay the cast cost, the
  behavior MUST be defined (see Open Questions — reject cast vs. consume-what's-
  available). Default assumption for v1: the client already gates on having ≥
  the required count (the same star it chose), so the server consumes what is
  present and logs a shortfall, consistent with the existing projectile
  shortfall posture (`character_attack_projectile.go:139-146`).

### FR-5 — Validate the chosen star

- Before the star id is injected into `SHADOW_CLAW` (FR-2) or used to compute the
  cast cost (FR-4), the server MUST verify that `SpiritJavelinItemId()` is:
  1. a throwing-star classification (`item.ClassificationConsumableThrowingStar`),
     and
  2. present in the caster's USE inventory.
- On validation failure, the server MUST NOT inject the bogus id or consume an
  unintended item. The failure handling (drop the whole cast vs. apply buff with
  no star / no consume) is a **design-phase decision**; the requirement is that
  no unowned/mistyped id reaches the client or the consume path. A warn-level log
  MUST record the rejected id, consistent with the existing defense-in-depth
  logging in `common.go UseSkill`.

## 5. API Surface

No new REST endpoints. Changes are confined to:

- **Packet model (libs/atlas-packet):** new getter `SkillUsageInfo.SpiritJavelinItemId()`.
  No wire-format change — the field is already decoded.
- **Kafka:** no new topics. Buff application and item consume reuse existing
  buff-apply and compartment reserve/consume messages. If the injection design
  requires threading the star id through an existing buff-apply command, that
  command's payload may gain a field (design-phase; note multi-tenancy/versioning
  if so).

## 6. Data Model

No persistent schema changes. The chosen star id is transient (per-cast, carried
in the `SHADOW_CLAW` temporary-stat value on the live buff). No new entities,
columns, or migrations.

## 7. Service Impact

- **libs/atlas-packet** — add `SpiritJavelinItemId()` getter to `SkillUsageInfo`
  (`model/skill_usage_info.go`).
- **services/atlas-channel** —
  - `skill/handler/common.go` (`UseSkill`): inject the chosen star id into the
    `SHADOW_CLAW` statup before `buff.Apply`; validate the star (FR-5); charge the
    cast cost (FR-4).
  - `socket/handler/character_attack_projectile.go` (`Plan`/gate): add the
    claw + `SHADOW_CLAW` consumption skip (FR-3).
- **services/atlas-data** — no functional change expected;
  `skill/reader.go:298` continues to emit the `SHADOW_CLAW` placeholder. Touch
  only if design chooses to remove/relocate the placeholder.

Per CLAUDE.md build rules, any service whose `go.mod` is touched must pass
`go test -race`, `go vet`, `go build`, and `docker buildx bake atlas-<svc>`, and
`tools/redis-key-guard.sh` must be clean from the repo root.

## 8. Non-Functional Requirements

- **Performance:** validation (FR-5) and cast cost (FR-4) run once per cast, off
  the attack hot path. The per-attack gate change (FR-3) adds one buff check for
  claws, matching the existing `SOUL_ARROW` check cost — negligible.
- **Security / anti-cheat:** FR-5 prevents a crafted client from injecting an
  arbitrary item id into a buff or triggering consumption of an unintended item.
- **Multi-tenancy:** all buff/inventory operations run under the existing
  `tenant.MustFromContext(ctx)` context; no tenant-specific values are hardcoded.
- **Observability:** rejected star ids (FR-5) and cast-cost shortfalls (FR-4) are
  logged at warn level with `characterId`, `skillId`, and the offending item id.
- **Version stability:** the skill id and star mechanic are version-stable across
  the supported GMS versions that have Night Lord; no per-version opcode work is
  anticipated. Confirm during design that the `SHADOW_CLAW` stat and the
  `spiritJavelinItemId` decode gate apply for all target versions.

## 9. Open Questions

1. **Cast-cost ownership (P7 vs P8).** The originating note says the 200-star cost
   "falls into the generic single-item consume (see P8)." This PRD scopes a
   working consume into P7 (FR-4) so the skill never ships the free-stars
   exploit. Confirm: implement in P7 (default), or land P7 gated behind P8 so
   both merge together? *(Interview question 1 — unanswered; default chosen.)*
2. **Star validation strictness.** FR-5 requires throwing-star-classification +
   ownership validation because the id now drives a consume. Confirm this over
   the lighter "trust the client / cast permitted" posture used elsewhere in
   `UseSkill`. *(Interview question 2 — unanswered; validation chosen.)*
3. **Injection mechanism.** Rewrite the `SHADOW_CLAW` statup amount in
   `common.go` before `buff.Apply`, versus adding an explicit parameter to the
   buff-apply path. Deferred to `/design-task`. *(Interview question 3 —
   unanswered; deferred to design.)*
4. **Shortfall behavior on cast cost (FR-4).** Reject the cast vs. consume-what's-
   available-and-log. Default: consume available + log, matching the existing
   projectile shortfall posture. Confirm during design.
5. **Interaction with Shadow Partner.** Once FR-3 skips claw consumption under
   `SHADOW_CLAW`, the `computeCount` Shadow-Partner doubling no longer applies to
   Shadow-Stars attacks. Confirm no other Shadow-Partner interaction is expected
   for the cast cost.

## 10. Acceptance Criteria

- [ ] `SkillUsageInfo.SpiritJavelinItemId()` getter exists and returns the decoded
      value; covered by a byte-fixture/decode test.
- [ ] Casting Shadow Stars with a chosen star results in a `SHADOW_CLAW` statup
      whose amount equals that star's item id (verified in a unit/integration
      test asserting the applied buff's statup value, not `0`).
- [ ] With Shadow Stars active and a claw equipped, a ranged attack produces **no**
      projectile-consume plan (test asserts the gate returns not-consuming for
      claw + `SHADOW_CLAW`).
- [ ] With Shadow Stars **inactive**, claw ranged attacks still consume stars
      (regression test — the new carve-out is correctly scoped).
- [ ] Casting Shadow Stars consumes `bulletCount` of the chosen star from the USE
      inventory (test asserts the reserve/consume targets the chosen item id and
      quantity).
- [ ] A cast with a non-throwing-star or unowned `spiritJavelinItemId` does not
      inject the id into the buff and does not consume an unintended item; a
      warn log is emitted (test asserts rejection).
- [ ] `go test -race ./...`, `go vet ./...`, `go build ./...` clean in every
      changed module; `docker buildx bake atlas-channel` (and any other touched
      service) succeeds; `tools/redis-key-guard.sh` clean from repo root.
