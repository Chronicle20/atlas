# Skill Cast Consumption Fidelity (itemConNo + cast-time bulletConsume) — Product Requirements Document

Version: v1
Status: Descoped (see note)
Created: 2026-07-10
---

> **Scope note (2026-07-27):** `task-158` (PR #1003, "Shadow Stars") landed on
> main and independently implemented the Shadow Stars half of this PRD — the
> cast-time `bulletConsume` cost, the SHADOW_CLAW star-id encoding, and the claw
> attack-path skip (**FR-2, FR-3, and the resolved open question §9.1**). Those
> requirements are DONE on main and are NOT re-implemented here. The remaining
> in-scope work is **FR-1 only (`itemConNo` quantity plumbing)**. The FR-2/FR-3
> sections below are retained for historical context; `plan.md` and `context.md`
> reflect the actual (FR-1-only) delivery and list where each superseded item
> now lives.

## 1. Overview

Skill casts in `atlas-channel` under-implement two WZ-driven consumption attributes, so casting costs are wrong relative to the client's expectation.

**Gap 1 — `itemConNo` (item consume amount) is ignored.** When a skill declares an item cost (`itemCon` + `itemConNo` in Skill.wz), `UseSkill` requests consumption of the item but the quantity is hardcoded to `1` inside atlas-channel's consumable processor (`services/atlas-channel/atlas.com/channel/consumable/processor.go:30` passes a literal `1` to `RequestItemConsumeCommandProvider`). The effect model already exposes the amount as `ItemConsumeAmount()` (`services/atlas-channel/atlas.com/channel/data/skill/effect/model.go:93`, backed by `itemConNo`), atlas-data already serves it (`services/atlas-data/atlas.com/data/skill/reader.go:217`), the Kafka contract already carries `Quantity int16` (`RequestItemConsumeBody`), and atlas-consumables already honors it end-to-end (`services/atlas-consumables/atlas.com/consumables/kafka/consumer/consumable/consumer.go:52` → `processor.go:221` reserve with quantity). Only the atlas-channel plumbing drops it. Note: the trailing `0` at `skill/handler/common.go:84` (the original backlog citation) is `updateTime`, not quantity — the hardcode is one layer down.

Verified impact: in the v83 WZ dump every `itemConNo` is `1`, so v83 tenants are currently unaffected by Gap 1; later-version data (verified in a v117-era dump: `itemConNo=2` skills, e.g. Echo of Hero consuming 2× Magic Rock 4006000) does exercise it. The fix is cross-version correctness plumbing.

**Gap 2 — cast-time `bulletConsume` is unimplemented.** Shadow Stars (4121006, Night Lord) declares `bulletConsume=200` in v83 Skill.wz with no `itemCon` — the 200-star cost is paid at cast time. `UseSkill` never reads `BulletConsume()`, so the cast is free. Worse, the projectile-attack path (`socket/handler/character_attack_projectile.go`) has no SHADOW_CLAW skip (only Soul Arrow for bow/crossbow at line 107), so attacks made *during* the buff keep consuming stars. Current behavior is exactly inverted from the client's model: the cast should cost 200 stars and subsequent throws should be free while the buff lasts. Reference implementation confirming both halves: Cosmic `StatEffect.java` (`isShadowClaw()` cast-time consume, first USE slot with ≥ `bulletConsume` matching stars, cast fails if none) and `RangedAttackHandler.java:182-190` (skip per-attack consume while `SHADOW_CLAW` buffed).

In v83 Skill.wz, `bulletConsume` appears on exactly four skills: 4121006 Shadow Stars (buff, 200), 4111005 (attack, 3), 14111002 (attack, 3), 5201001 (attack, 1). The attack skills are already handled by the projectile path's `computeCount` (`character_attack_projectile.go:227`). Only the buff-cast case (Shadow Stars) is missing.

## 2. Goals

Primary goals:
- Skill casts consume the WZ-declared item quantity (`itemConNo`), not a hardcoded 1, across all tenant versions.
- Buff casts that declare `bulletConsume` (Shadow Stars) consume that many matching projectiles at cast time, with Cosmic-parity slot selection and failure semantics.
- Projectile attacks made while SHADOW_CLAW is active consume no stars (parity with the existing Soul Arrow skip).
- No Kafka schema or atlas-consumables behavior changes — the existing `Quantity` contract is used as-is.

Non-goals:
- Passive no-consume mechanics (Mortal Blow, Claw Mastery roll-to-preserve, etc.) — pre-existing `TODO(task-007)` at `character_attack_projectile.go:118`, unchanged.
- Changing the itemCon shortfall stance: today a cast whose required item is missing is still permitted (defense-in-depth gate only, `skill/handler/common.go:86`). That stance is preserved for `itemCon`; only Shadow Stars gets the stricter Cosmic-parity gate (see FR-2.3 and §9).
- SHADOW_CLAW statup amount fidelity (currently applied with amount 0 by `atlas-data/skill/reader.go:298`) — whether the v83 client needs the consumed star's item id encoded in the buff is an open question (§9), not a requirement of this task.
- Multi-slot spread draws for `itemCon` consumption (single-slot-with-enough was chosen; the projectile path's multi-slot `resolvePlan` remains attack-only).
- `updateTime` plumbing through the skill-cast consume request (remains 0; it is log-only today).

## 3. User Stories

- As a Night Lord player, I want casting Shadow Stars to deduct 200 throwing stars from my inventory so that the buff has its intended cost.
- As a Night Lord player with Shadow Stars active, I want my attacks to stop consuming stars so that the buff delivers its intended benefit.
- As a player on a later-version tenant casting a skill with `itemConNo > 1`, I want the correct number of items consumed so that my inventory matches what the client shows.
- As a server operator, I want consumption quantities driven by per-tenant WZ data (no hardcoded skill/version branches) so that all tenant versions behave correctly.

## 4. Functional Requirements

### FR-1 — `itemConNo` quantity plumbing (atlas-channel)

- **FR-1.1** `UseSkill` (`skill/handler/common.go`) must request consumption of `e.ItemConsumeAmount()` units of the `itemCon` item. An amount of `0` (attribute absent in WZ; reader default at `atlas-data/skill/reader.go:217`) must be treated as `1`. Note real v83 data contains explicit `itemConNo=1` entries; some later dumps carry string-typed `"0"` values, hence the floor.
- **FR-1.2** `consumable.Processor.RequestItemConsume` (atlas-channel) must accept a quantity and pass it to `RequestItemConsumeCommandProvider` instead of the literal `1`. All existing call sites (`character_item_use.go` ×3, `character_cash_item_use.go`, `pet_food.go`, `pet_item_use.go`) pass `1` — their behavior is unchanged. Whether this is a signature change or a new method is a design decision.
- **FR-1.3** Slot selection: choose the lowest-index slot in the item's compartment whose quantity ≥ the required amount (replacing the bare `FindFirstByItemId` at `common.go:83` for this path). If no single slot qualifies, log a warning, skip consumption, and permit the cast (unchanged defense-in-depth stance).
- **FR-1.4** No changes to the Kafka message schema (`RequestItemConsumeBody.Quantity` already exists) or to atlas-consumables (it already reserves with the received quantity, `consumables/consumable/processor.go:221`). A test must pin that the emitted command carries the effect's quantity.

### FR-2 — Cast-time `bulletConsume` (Shadow Stars)

- **FR-2.1** In `UseSkill`, when `e.BulletConsume() > 0`, consume that many projectiles at cast time. The trigger must be data-driven (`BulletConsume()` on the effect model), not keyed to skill id 4121006, so any version's buff-cast with `bulletConsume` is covered. (Attack skills with `bulletConsume` never flow through `UseSkill`; they are already handled in the projectile-attack path.)
- **FR-2.2** Projectile matching and slot selection: lowest-index slot in the USE compartment whose item classification matches the caster's equipped weapon's projectile type (claw → throwing star, per `requiredClassification` at `character_attack_projectile.go:211`) and whose quantity ≥ `bulletConsume`. Single-slot draw only (Cosmic parity, and the user-selected rule).
- **FR-2.3** If no qualifying slot exists, the cast must not apply the buff and nothing is consumed (Cosmic parity: `StatEffect.java` returns false). Log at warn. This is deliberately stricter than the `itemCon` stance — the 200-star cost is the skill's defining cost, and silently granting a free buff is a material economy/fidelity break.
- **FR-2.4** The 200-star decrement must go through the existing reserve→consume consumable flow (quantity from FR-1 plumbing); no new direct-inventory mutation path.
- **FR-2.5** Consumption ordering: the bullet cost is settled (qualifying slot found, consume requested) before the buff apply for the same cast.

### FR-3 — Attack-path SHADOW_CLAW skip

- **FR-3.1** In the projectile consumption gate (`character_attack_projectile.go`, alongside the Soul Arrow skip at line 107), when the caster's weapon is a claw and the `SHADOW_CLAW` temporary stat (`libs/atlas-constants/character/temporary_stat.go:46`) is active, skip projectile consumption entirely for the attack. The Shadow Partner ×2 doubling must not resurrect a consume in this case (the skip precedes `computeCount`).
- **FR-3.2** The existing buff-lookup-failure fallback (treat as no buffs, over-consume rather than break the attack path, `character_attack_projectile.go:97-105`) applies to this skip as well.

## 5. API Surface

- **REST:** no changes.
- **Kafka:** no schema changes. `CONSUMABLE` command topic's `REQUEST_ITEM_CONSUME` body already carries `quantity` (`atlas-channel/kafka/message/consumable/kafka.go:33`, mirrored in atlas-consumables). The only observable change is that skill-cast-originated commands may now carry `quantity > 1`.
- **Internal (atlas-channel):** `consumable.Processor.RequestItemConsume` gains a quantity parameter (or a quantity-bearing sibling method — design decision). `UseSkill` gains the cast-time bullet-consume step; the projectile attack handler gains the SHADOW_CLAW skip.

## 6. Data Model

No new entities, fields, or migrations. All required data already flows: WZ `itemConNo` → atlas-data `itemConsumeAmount` REST attribute → atlas-channel `effect.Model.ItemConsumeAmount()`; WZ `bulletConsume` → `effect.Model.BulletConsume()`.

## 7. Service Impact

- **atlas-channel** — all code changes: consumable processor quantity plumbing, `UseSkill` itemConNo amount + slot selection, cast-time bulletConsume step with buff gate, projectile-attack SHADOW_CLAW skip. Tests for each (Builder-pattern test setup per project convention; the existing seams in `skill/handler/common.go` and the projectile handler's plan tests are the model).
- **atlas-consumables** — no behavior change expected. Optionally a pinning test that a `quantity > 1` command reserves that quantity through `ConsumeBare` (the fallback branch skill items hit, `consumables/consumable/processor.go:206-216`).
- **atlas-data** — no changes (already serves both attributes).

## 8. Non-Functional Requirements

- **Multi-tenancy / versions:** behavior must be entirely WZ-data-driven per tenant. No version branches, no hardcoded skill ids for the consume triggers (skill-id use is acceptable only where atlas-data already keys statups, e.g. SHADOW_CLAW statup emission — unchanged).
- **Atomicity/ordering:** consumption uses the existing async reserve→consume transaction (`RequestReserve` + one-time status consumer). No new failure modes beyond current behavior; a failed reservation must not crash the cast path.
- **Observability:** shortfall skips (FR-1.3) and blocked casts (FR-2.3) log at warn with characterId, skillId, itemId/classification, required amount.
- **Verification:** per CLAUDE.md — `go test -race`, `go vet`, `go build` in changed modules; `docker buildx bake atlas-channel` (and `atlas-consumables` if its `go.mod`-scoped code is touched); `tools/redis-key-guard.sh`.

## 9. Open Questions

1. **SHADOW_CLAW buff amount encoding:** atlas-data applies SHADOW_CLAW with amount 0. The v83 client may expect the buff's option to encode the consumed star item id (it must render a projectile for star-free throws; Cosmic separately resolves a `visProjectile`). Needs IDA verification of the v83 client's SHADOW_CLAW handling during design. If required, that becomes a small atlas-data/buff change; if not, no action.
2. **Does the Shadow Stars `UseSkill` packet carry a slot hint?** The projectile-attack decode has `ProperBulletPosition()`; if the skill-use packet for 4121006 also carries a position, slot selection should prefer it (like `resolvePlan` step 1). Design phase to check the packet decode / IDA.
3. **Shortfall race:** between the inventory read (slot pick) and the async reservation, the stack can change. Current code already has this window for quantity 1; with 200 the window is more visible. Accepted risk unless design finds a cheap fix (the reservation itself is the authoritative gate — a failed reserve consumes nothing).

## 10. Acceptance Criteria

- [ ] Casting a skill whose effect has `ItemConsumeAmount() == N` (N > 1) emits a `REQUEST_ITEM_CONSUME` command with `quantity == N`; `N == 0` emits `quantity == 1`. Pinned by test.
- [ ] The consume slot chosen is the lowest-index slot holding ≥ N of the item; when no slot qualifies, no command is emitted, a warning is logged, and the cast still proceeds. Pinned by test.
- [ ] All pre-existing `RequestItemConsume` call sites still emit `quantity == 1` (compile-verified via signature change + spot test).
- [ ] Casting Shadow Stars (effect with `BulletConsume() == 200`, claw weapon) with a qualifying star stack: emits a consume for exactly 200 of that stack's item and applies the SHADOW_CLAW buff. Pinned by test.
- [ ] Casting Shadow Stars with no single stack ≥ 200: no consumption, no buff applied, warning logged. Pinned by test.
- [ ] A claw projectile attack while SHADOW_CLAW is active consumes zero projectiles (including with Shadow Partner also active); without the buff, consumption is unchanged. Pinned by test.
- [ ] Soul Arrow skip, pet food, item-use, and cash-item-use paths behave exactly as before (regression tests pass).
- [ ] `go test -race ./...`, `go vet ./...`, `go build ./...` clean in atlas-channel (and atlas-consumables if touched); `docker buildx bake atlas-channel` succeeds; `tools/redis-key-guard.sh` clean.
- [ ] Open question §9.1 (SHADOW_CLAW amount) resolved during design with an IDA-verified answer, and implemented if the client requires it.
