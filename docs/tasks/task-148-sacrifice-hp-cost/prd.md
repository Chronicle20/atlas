# Sacrifice Self-HP Cost (Dragon Knight 1311005) — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-07-09
---

## 1. Overview

Dragon Knight's Sacrifice (skill 1311005) is a single-target melee attack that trades the attacker's HP for high damage that ignores enemy weapon defense. The attack itself already works in Atlas — damage is decoded, applied to the monster, and broadcast — but the defining self-HP cost is unimplemented, marked by the TODO at `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go:405` (`// TODO decrease HP from DragonKnight Sacrifice`). Today a Dragon Knight casts Sacrifice for free, which is a meaningful balance deviation for the job.

Reference behavior was verified directly from Cosmic source (not memory):

- `Cosmic/src/main/java/net/server/channel/handlers/CloseRangeDamageHandler.java:142-149` — after a melee attack with skill `1311005` that hit at least one target, the attacker loses HP equal to `firstDamageLine × effect.X / 100`, where `firstDamageLine` is the first damage line of the (single) target and `X` comes from the skill effect data for the caster's skill level.
- `Cosmic/src/main/java/client/AbstractCharacterObject.java:488-495` (`safeAddHP`) — the loss is clamped so the character is left with at least 1 HP. **Sacrifice can never kill the caster.**

This task implements that behavior in atlas-channel's shared attack post-processing, using effect data served per-tenant by atlas-data (no hardcoded percentages), and removes the TODO.

## 2. Goals

Primary goals:
- Apply the self-HP cost after a successful Sacrifice attack: `X%` of the first damage line dealt, per the caster's skill level effect data.
- Clamp the cost channel-side so the caster always survives with at least 1 HP (Cosmic parity).
- Keep the attack pipeline resilient: cost application failures are logged and swallowed, never abort the attack.
- Remove the TODO at `character_attack_common.go:405`.

Non-goals:
- The neighboring attack-side TODOs, including HP-gain skills (Drain / Energy Drain / Vampire — research item #7), Pick Pocket, Meso Explosion, on-hit debuff procs.
- Brawler MP Recovery (5101005) — research item #6, a buff-side `SpecialMoveHandler` skill, not part of this attack path.
- Any change to atlas-character or its `CHANGE_HP` command semantics (interview decision: clamp channel-side).
- Per-skill dispatcher registration for Sacrifice — it is a plain attack skill, not a dual-packet skill; the generic HP/MP *cast cost* block (`character_attack_common.go:303-310`, driven by `hpCon`/`mpCon`) continues to apply independently of this damage-based cost.

## 3. User Stories

- As a Dragon Knight player, I want Sacrifice to deduct HP proportional to the damage I dealt so that the skill has its authentic risk/reward tradeoff.
- As a Dragon Knight player, I want Sacrifice to never kill my character (leaving me at 1 HP at worst) so that the skill behaves like the reference client expects.
- As an operator, I want the cost percentage to come from the tenant's skill effect data so that behavior is correct on every supported version without code changes.

## 4. Functional Requirements

All changes are in `processAttack` in `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go` (and a small extracted helper). Interview decisions are baked in below.

- **FR-1 (trigger):** After the per-monster damage processing loop, when `ai.SkillId()` equals `skill.DragonKnightSacrificeId` (use the existing constant from `libs/atlas-constants/skill/constants.go:2997`; never a literal `1311005`) and the attack has at least one damage entry, compute and apply the self-HP cost exactly once per attack packet.
- **FR-2 (damage basis):** The cost basis is **only the first damage line of the first damage entry**: `ai.DamageInfo()[0].Damages()[0]` (getters at `libs/atlas-packet/model/damage_info.go:80`). Do not sum additional lines or targets — Sacrifice must never cost more than the first line (interview decision #2, matches Cosmic `damageLines().getFirst()`). If there are no damage entries, the first entry has no damage lines, or the first line is 0 (miss), apply no cost.
- **FR-3 (formula):** `cost = firstLine × X / 100` using integer arithmetic (truncating division, matching Cosmic's Java `int` math), where `X` is `se.X()` from the effect model already fetched in `processAttack` (`skill2.NewProcessor(l, ctx).GetEffect(ai.SkillId(), sk.Level())`, line 292). If `X ≤ 0`, apply no cost.
- **FR-4 (survival clamp, channel-side):** Clamp against the caster's *current* HP from the already-fetched character model `c` (`c.Hp()`, `services/atlas-channel/atlas.com/channel/character/model.go:131`): if `cost ≥ c.Hp()`, reduce it to `c.Hp() − 1`; if `c.Hp() ≤ 1`, the cost is 0 and no HP change is emitted. Effective stats are **not** consulted — the clamp needs current HP, not derived maxima (interview decision #3). The slight staleness of `c` relative to concurrent damage is accepted; Cosmic has the same read-then-write shape.
- **FR-5 (application):** Apply via the existing `cp.ChangeHP(s.Field(), s.CharacterId(), -int16(cost))` (`services/atlas-channel/atlas.com/channel/character/processor.go:271`), the same mechanism the generic cast-cost block uses. The FR-4 clamp bounds `cost ≤ c.Hp() − 1 < 32767`, so the `int16` narrowing cannot overflow.
- **FR-6 (resilience):** Any error from `ChangeHP` is logged (`Errorf`) and swallowed; the attack pipeline (broadcast, projectile consumption) is unaffected. Follow the MP Eater conventions at `character_attack_common.go:202-264`.
- **FR-7 (testable core):** Extract the pure computation (first-line selection, percentage, clamp) into a helper function alongside `mpEaterAbsorbAmount` (`character_attack_common.go:195`) so it is unit-testable without socket/session scaffolding. Tests use the project Builder pattern; no `*_testhelpers.go` files.
- **FR-8 (cleanup):** Delete the TODO comment at line 405. The other TODOs in that block remain untouched.
- **FR-9 (interaction with cast cost):** The generic `hpCon`/`mpCon` deduction (lines 303-310) is a separate, pre-existing cast cost and must continue to apply. This task adds the damage-proportional cost only; do not gate or merge the two.

## 5. API Surface

None. No new or modified REST endpoints, Kafka topics, or packet writers. The HP change rides the existing atlas-channel → atlas-character `ChangeHP` command path, which already emits the stat-update event/packet flow to the client.

## 6. Data Model

None. The `X` percentage is read at runtime from the tenant-scoped skill effect served by atlas-data (`effect.Model.X()`), the same field MP Eater consumes. No migrations, no seed/template changes.

## 7. Service Impact

- **atlas-channel** — the only changed service. One helper function + one call site in `socket/handler/character_attack_common.go`, plus unit tests.
- atlas-character, atlas-data — consumed as-is; no changes.

## 8. Non-Functional Requirements

- **No additional network fetches:** the implementation reuses the character model `c` and effect `se` already fetched by `processAttack`. Zero extra REST/Kafka round-trips on the hot attack path for non-Sacrifice attacks (a single skill-id comparison).
- **Multi-tenancy:** version-agnostic by construction — `X` resolves per-tenant from atlas-data. Interview decision #4: validate on v83 and v95 tenants; no version branching expected or permitted (no `MajorVersion()` gates).
- **Observability:** log the applied cost at `Debugf` (caster id, skill id, first line, X, clamped cost), errors at `Errorf`, mirroring MP Eater logging.
- **Concurrency:** no new shared state; the helper is pure.

## 9. Open Questions

None. All interview questions were resolved:
1. HP floor: clamp to leave ≥ 1 HP (never kill) — confirmed.
2. Damage basis: first damage line only; never more — confirmed.
3. Clamp location: channel-side against current HP; effective stats not used — confirmed.
4. Validation targets: v83 and v95; implementation version-agnostic — confirmed.

## 10. Acceptance Criteria

- [ ] Attacking with Sacrifice (1311005) deducts `firstDamageLine × X / 100` HP from the caster, where `X` is the level-appropriate effect value from atlas-data.
- [ ] A Sacrifice that would reduce HP to 0 or below leaves the caster at exactly 1 HP; a caster already at 1 HP loses nothing.
- [ ] A missed Sacrifice (first damage line 0, or no damage entries) costs nothing beyond the normal `hpCon`/`mpCon` cast cost.
- [ ] The generic cast-cost block still applies unchanged for Sacrifice and all other skills.
- [ ] Non-Sacrifice attacks are behaviorally unchanged.
- [ ] Unit tests cover: normal computation, truncating division, `X ≤ 0`, zero/missing damage lines, clamp to `Hp−1`, `Hp ≤ 1` no-op.
- [ ] `// TODO decrease HP from DragonKnight Sacrifice` is removed.
- [ ] `go test -race ./...`, `go vet ./...`, `go build ./...` clean in `services/atlas-channel/atlas.com/channel`; `tools/redis-key-guard.sh` clean from repo root; `docker buildx bake atlas-channel` if `go.mod` is touched (not expected).
- [ ] Manual validation on v83 and v95 tenants: HP drop matches the formula in-game and the caster cannot die to it.
