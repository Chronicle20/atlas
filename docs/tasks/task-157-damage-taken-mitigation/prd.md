# Damage-Taken Mitigation/Reaction Skills — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-07-10
---

## 1. Overview

When a character takes damage, the v83+ client sends a damage-taken packet that atlas-channel decodes (`libs/atlas-packet/model/damage_taken_info.go`) and applies verbatim: `character_damage.go:43` calls `ChangeHP(..., -damage)` unconditionally. The entire family of damage-taken mitigation and reaction skills is therefore inert on the server even though the buffs apply and the client renders their icons: Magic Guard drains HP instead of splitting to MP, Power Guard reflects nothing, Meso Guard neither halves damage nor spends mesos, and Mana Reflection, Achilles, Combo Barrier, Body Pressure, Divine Shield, and High Defense do nothing. The handler carries nine TODOs (`character_damage.go:22-30`) enumerating exactly this gap.

This task implements the damage-taken pipeline: on each damage event, atlas-channel looks up the character's active buffs (`character/buff` processor) and passive skill levels (character `skills` via `SkillModelDecorator`), computes per-skill mitigation and reactions from skill-effect data (atlas-data `data/skill/effect`), and applies the adjusted deltas — HP, MP (Magic Guard), mesos (Meso Guard), and reflected mob damage (Power Guard, Mana Reflection, Body Pressure) — instead of the raw client value.

Two principles govern the work. First, **server authority with anti-cheat clamping**: the server recomputes all mitigation from its own buff/skill state, treating the client damage value as raw pre-mitigation input and the client's `bPowerGuard`/`powerGuard` flags as hints only — the historical Power Guard reflect exploit (client-inflated reflect values) must be structurally impossible. Second, **per-skill IDA verification**: for each skill, the design phase must verify against the client binary what the client pre-applies to the damage value before sending, so the server neither double-applies nor skips a mitigation. Cosmic's `TakeDamageHandler` is the reference implementation, but its assumptions are verified, not trusted.

## 2. Goals

Primary goals:
- Every skill in the roster (§4.1) mitigates/reacts server-side according to its IDA-verified per-version semantics.
- Damage application is server-authoritative: mitigation is recomputed from server-side buff/skill state; client-supplied reflect flags are never trusted for amounts.
- Anti-cheat clamps reject or bound absurd client damage values and reflect claims (including the classic Power Guard exploit).
- Behavior is version-gated across all supported tenant versions; post-Big Bang skills (Divine Shield) activate only on post-BB tenants (v95+).
- The nine mitigation TODOs in `character_damage.go` are removed (battleship TODO excluded — owned by task-153).

Non-goals:
- Corsair Battleship HP decrease (`// TODO decrease battleship hp`) — in-flight task-153-corsair-battleship.
- Dark Knight Berserk HP-threshold interplay — task-154 owns Berserk; this task only guarantees HP changes flow through the existing HP-change events Berserk listens to.
- Attack-side passives (MP Eater, drain) — task-147 family.
- Guardian/Blocking-type skills that cancel the hit entirely client-side, unless IDA verification shows a server obligation (recorded as an open question if discovered).
- Any UI work.

## 3. User Stories

- As a Magician with Magic Guard active, I want damage split to MP per the skill's percentage so that my HP survives hits the way the client displays them.
- As a Magician whose MP runs out mid-hit, I want the un-absorbable remainder applied to HP so that Magic Guard cannot make me unkillable.
- As a Fighter/Page with Power Guard active, I want a percentage of physical touch damage reflected to the attacking mob so that the skill functions as designed.
- As a Chief Bandit with Meso Guard active, I want damage halved and the guarding cost deducted from my mesos, falling back to full damage when I cannot pay.
- As an Arch Magician/Bishop with Mana Reflection, I want magic attacks reflected back at the caster per the skill's proc chance and formula.
- As a Hero/Paladin/Dark Knight, I want my passive Achilles reduction applied to every damage event without any buff being active.
- As an Aran, I want Combo Barrier's reduction, High Defense's effect, and Body Pressure's on-touch reflection honored.
- As a post-BB Paladin (v95+ tenant), I want Divine Shield's hit-blocking honored, and as a v83 player I want no post-BB code path to activate.
- As a server operator, I want client-supplied damage/reflect values clamped and logged when out of bounds so that the Power Guard reflect exploit and damage-packet forgery are neutralized.

## 4. Functional Requirements

### 4.1 Skill roster and classification

Buff-driven (detected via active buff temporary-stat types from `character/buff` processor; stat type constants already exist in `libs/atlas-constants/character/temporary_stat.go`):

| Skill | IDs | Stat type | Reference behavior (Cosmic — verify per §4.3) |
|---|---|---|---|
| Magic Guard | 2001002 (Magician), 12001001 (Blaze Wizard), 22111001 (Evan) | `MAGIC_GUARD` | x% of damage paid from MP; MP shortfall spills back to HP |
| Power Guard | 1101007 (Fighter), 1201007 (Page) | `POWER_GUARD` | x% of physical touch damage reflected to attacker; player damage reduced by reflected amount; reflect capped vs mob HP |
| Meso Guard | 4211005 (Chief Bandit) | `MESO_GUARD` | ~50% of damage guarded; guarded portion costs mesos (rate from skill effect); insufficient mesos → no guard |
| Mana Reflection | 2121002 (F/P), 2221002 (I/L), 2321002 (Bishop) | `MANA_REFLECTION` | on magic damage, proc chance reflects portion of damage to caster, capped vs mob max HP |
| Combo Barrier | 21120007 (Aran) | `COMBO_BARRIER` | % damage reduction while active |
| Body Pressure | 21101003 (Aran) | `BODY_PRESSURE` | on mob-touch damage, proc chance deals reflect damage to the touching mob |

Passive (no buff; detected from character skill levels via `SkillModelDecorator`):

| Skill | IDs | Reference behavior (verify) |
|---|---|---|
| Achilles | 1120004 (Hero), 1220005 (Paladin), 1320005 (Dark Knight) | flat % damage reduction on all damage taken |
| High Defense | 21120004 (Aran) | semantics TBD — may be stat-only (already reflected in defense stats) with no damage-pipeline hook; IDA verification decides (§4.3); if stat-only, document and remove the TODO with no code path |
| Divine Shield | Paladin, post-BB — **ID unverified**, no `libs/atlas-constants` entry yet | on-hit block/shield mechanic; exact trigger and server obligation TBD from v95 IDA |

All skill IDs above except Divine Shield already exist in `libs/atlas-constants/skill/constants.go`; Divine Shield's constant is added once its ID is verified from v95 WZ/IDA (never from memory).

### 4.2 Damage pipeline

- FR-2.1: `CharacterDamageHandleFunc` classifies the event by `nAttackIdx` (`DamageTypePhysical`, `DamageTypeMagic`, `DamageTypeCounter`, `DamageTypeObstacle`, `DamageTypeStat`) and routes it through an ordered mitigation chain. Which mitigations apply to which damage types is IDA-verified (e.g., Power Guard applies to touch damage only; Mana Reflection to magic only).
- FR-2.2: The chain's application **order** (e.g., Achilles before/after Magic Guard split; Meso Guard vs Magic Guard stacking) is determined by IDA verification of the client's own damage computation, and documented in design.md. No order is assumed from memory.
- FR-2.3: Each mitigation step is a pure function `(damageContext) -> (adjusted damageContext, side effects)` testable in isolation; side effects (MP change, meso change, mob damage) are emitted after the chain completes, batched via `message.Buffer` where the pattern applies.
- FR-2.4: When no relevant buff/passive exists, behavior is byte-identical to today: full damage to HP, single announce to other sessions.
- FR-2.5: The foreign-session `CharacterDamageWriter` announce continues to broadcast the client-reported event (including its power-guard flags) so observers render correctly; the announce is not blocked on mitigation computation.

### 4.3 Per-skill IDA verification (mandatory)

- FR-3.1: For each roster skill × supported version family, the design phase decompiles the client's damage-taken path (via ida-pro MCP, `func_query` with `name_regex`, correct `select_instance` per version) and records: (a) whether the client pre-applies the mitigation to the `damage` field before sending, (b) the exact formula and rounding, (c) which `nAttackIdx` values the mitigation applies to, (d) any client-side proc roll (and whether the server must re-roll or trust the packet).
- FR-3.2: Findings are committed as a verification matrix in the task folder (design phase artifact). A skill may not be implemented ahead of its verification row.
- FR-3.3: **[Revised post-`main`-merge — see design §2a/§3, plan finding §7.]** The original scope assumed IDBs for v83/v87/v95/jms only, with v92 inheriting the nearest verified version. After merging `main`'s legacy bring-up (which wired `CharacterDamageHandle` for gms_48/61/72/79/84), the verified column set is **v48, v61, v72, v79, v83, v84, v87, v92, v95, jms_185** — live per-version IDB sessions exist for all of them, including **v92, which is verified independently and does NOT inherit v87**. Each column's `CUserLocal::SetDamaged` is decompiled and cited in design §3.
- FR-3.4: Where Cosmic's server-side re-application disagrees with the client binary, the client binary wins; the discrepancy is recorded.

### 4.4 Magic Guard

- FR-4.1: With `MAGIC_GUARD` active, x% (from the buff's statup value / skill effect) of incoming damage is deducted from MP via `ChangeMP`, remainder from HP.
- FR-4.2: If current MP < the MP share, the shortfall is applied to HP (no free absorption). Requires reading current MP from the character model.
- FR-4.3: Applies to the damage types verified in §4.3 (reference: both physical and magic mob damage).

### 4.5 Power Guard

- FR-5.1: With `POWER_GUARD` active and the event being an eligible touch/physical hit with a valid attacking mob (`monsterId`/`monsterTemplateId` present, mob alive in the character's field), the server computes reflect = verified formula from server-side buff value and server-side damage input — never from client-supplied amounts.
- FR-5.2: Reflect is capped per the verified rule (reference: percentage of the mob's current/max HP; boss handling per verification).
- FR-5.3: Reflected damage is applied via the existing `monster.Damage` command (attribution to the character is acceptable — matches Cosmic). `EmitDamageReflected` is used where the observer-facing status event is required.
- FR-5.4: The client's `bPowerGuard`/`powerGuard` booleans and `monsterId2` are cross-checks only. Mismatch between client claim and server state (flag set but no Power Guard buff; reflect target not the attacking mob) is logged at warn and the client claim ignored.

### 4.6 Meso Guard

- FR-6.1: With `MESO_GUARD` active, the guarded share of damage (verified %, reference ~50%) is absorbed and its meso cost (rate from skill effect) deducted by emitting the existing atlas-character command `REQUEST_CHANGE_MESO` (already consumed by atlas-character; channel needs only a producer).
- FR-6.2: If the character's meso balance cannot cover the cost, the guard does not apply (full damage to HP) — resolved against the server-side meso balance before emitting, not by fire-and-forget relying on `NOT_ENOUGH_MESO`.
- FR-6.3: Meso deduction and HP change for one damage event are emitted together (single buffered emission) to avoid partial application on failure paths.

### 4.7 Mana Reflection

- FR-7.1: With `MANA_REFLECTION` active and the event magic-type, the verified proc rule applies (whether the server rolls `Prop()` or honors a client indication is a §4.3 finding); on proc, reflect damage per the verified formula, capped per the verified rule, applied via `monster.Damage`.

### 4.8 Passives: Achilles, High Defense

- FR-8.1: Achilles reduction applies whenever the character has the skill at level > 0, using the level's effect data — no buff required.
- FR-8.2: High Defense is implemented only if §4.3 verification shows a damage-pipeline obligation; if it is stat-only, the TODO is removed with a code comment-free, documented no-op decision in design.md.

### 4.9 Combo Barrier, Body Pressure, Divine Shield

- FR-9.1: Combo Barrier's % reduction applies while the buff is active.
- FR-9.2: Body Pressure's on-touch reaction applies per its verified proc/formula on mob-touch damage events.
- FR-9.3: Divine Shield is implemented for post-BB tenants only (v95+), version-gated, with ID/semantics from v95 verification. On pre-BB tenants the code path is unreachable.

### 4.10 Anti-cheat clamping

- FR-10.1: `damage` is validated: negative values (other than legitimate sentinel semantics found in §4.3) and values exceeding a sane bound are clamped/rejected and logged with character, map, mob template, and raw packet fields.
- FR-10.2: The existing `-int16(p.Damage())` truncation bug (int32 damage silently wraps in int16) is fixed as part of the pipeline — HP deltas are range-checked before conversion.
- FR-10.3: All reflect amounts are server-computed and capped (FR-5.1/5.2, FR-7.1); a client cannot influence reflect magnitude beyond supplying the raw damage input, which is itself clamped.
- FR-10.4: Clamp events are observable (structured warn logs; counter metric if the service already has a metrics pattern — do not introduce a new metrics stack for this).

### 4.11 Version gating

- FR-11.1: All version-dependent behavior gates on tenant region + major version via `tenant.MustFromContext(ctx)`, consistent with existing patterns (cf. the v95 `bGuard` field gate already in the decoder).
- FR-11.2: Skill availability gates: Evan Magic Guard (v84+ per version timeline — verify), Blaze Wizard/Aran per their introduction versions, Divine Shield post-BB (v95+). Gating is by skill existence in that version's data — a character on a version without the skill simply never has it — plus explicit gates only where formulas differ by version.
- FR-11.3: If §4.3 finds the damage packet layout itself differs by version beyond the known v95 `bGuard` field, `damage_taken_info.go` decode/encode is extended with the same gating style and byte-fixture tests.

## 5. API Surface

No new REST endpoints.

Kafka:
- atlas-channel produces the existing atlas-character command `REQUEST_CHANGE_MESO` (new producer + command constant in atlas-channel's `kafka/message/character`; body shape copied from atlas-character's consumer contract, verified against `services/atlas-character/atlas.com/character/kafka/message/character/kafka.go`).
- Existing commands/events reused: `CHANGE_HP`, `CHANGE_MP` (atlas-character), monster `Damage` command and `DAMAGE_REFLECTED` status event (atlas-monsters topics, producers already in atlas-channel).
- No new topics.

Error cases: meso-insufficient resolves server-side pre-emission (FR-6.2); mob-not-found on reflect drops the reflect (logged debug) but still applies the character-side mitigation.

## 6. Data Model

No new persistent entities and no migrations. All inputs come from existing sources:
- Active buffs: `character/buff` processor (`GetByCharacterId`) — statup types + values.
- Passive skill levels: character model `skills` (requires `SkillModelDecorator` on the damage-path character fetch).
- Skill effect data (x, prop, etc. per level): atlas-data via `data/skill/effect`.
- Meso balance / current MP: character model.

All lookups are tenant-scoped through context as today.

## 7. Service Impact

- **atlas-channel** (primary): rework `socket/handler/character_damage.go` into the mitigation pipeline; new mitigation package with per-skill pure functions; buff + skill + effect lookups; `REQUEST_CHANGE_MESO` producer; anti-cheat clamps; unit tests (Builder pattern per project convention — no `*_testhelpers.go`).
- **atlas-character**: expected no change (already consumes `REQUEST_CHANGE_MESO`, `CHANGE_HP`, `CHANGE_MP`); contract verified, any gap surfaced in design.
- **atlas-monsters**: expected no change (existing damage command / reflected status event).
- **libs/atlas-constants**: add `PaladinDivineShieldId` (+ Skill var/map entries) once IDA/WZ-verified; add any missing temporary-stat constants (roster's stat types already exist).
- **libs/atlas-packet**: only if §4.3 reveals version-dependent decode differences (FR-11.3); byte-fixture tests mandatory for any change.

## 8. Non-Functional Requirements

- **Multi-tenancy**: all processing derives tenant from context; version gates from tenant model; no cross-tenant state.
- **Performance**: damage events are high-frequency. The pipeline adds buff/skill/effect lookups per event; effect data lookups should use whatever caching the `data/skill/effect` path already provides, and the character fetch must not add decorators beyond what the pipeline needs. No new synchronous cross-service calls beyond those already on this path plus the buff lookup; if the buff lookup is REST-per-event, design must assess and state the cost.
- **Correctness under concurrency**: HP/MP/meso deltas for one event are emitted atomically-per-event via the message buffer pattern; no read-modify-write races introduced in the handler (registry/local state, if any, guarded per project patterns).
- **Observability**: debug log of the mitigation breakdown per event (pre/post damage, per-skill contributions); warn logs for clamp/mismatch events (FR-10.4).
- **Security**: server authority throughout; client packet fields are inputs to validation, never to authorization.

## 9. Open Questions

All are design-phase items answered by IDA verification (§4.3), not blockers to starting design:
1. Which mitigations does the v83 client pre-apply to the sent `damage` value (the central caveat)? Per-skill, per-version.
2. Exact formulas, rounding, caps, and mitigation ordering per skill per version.
3. Divine Shield's skill ID and precise server obligation on v95.
4. High Defense: damage-pipeline hook or stat-only?
5. Proc-chance authority for Mana Reflection / Body Pressure: server re-roll vs client indication.
6. Whether any block-type skill (e.g., Guardian) creates a server obligation discovered during verification — if so, it becomes a scoped follow-up, not silent scope creep.

## 10. Acceptance Criteria

- [ ] Verification matrix committed covering every roster skill × {v48, v61, v72, v79, v83, v84, v87, v92, v95, jms} (post-merge scope, FR-3.3 as revised), each row citing decompiled client evidence; the v48 divergent wire layout and the v92-independent verification are documented in design §2a/§3.
- [ ] `CharacterDamageHandle` wired in all ten templates (gms_48/61/72/79/83/84/87/92/95/jms); `tools/template-opcode-order-guard.sh` clean.
- [ ] All nine mitigation TODOs removed from `character_damage.go`; battleship TODO left for task-153.
- [ ] Unit tests for every mitigation function: no-buff passthrough, standard case, boundary cases (MP shortfall, meso shortfall, reflect cap, dead/missing mob, clamp triggers), using the project Builder pattern.
- [ ] Anti-cheat: forged oversized damage and forged Power Guard reflect claims are demonstrably clamped/ignored in tests; int16 truncation fixed.
- [ ] Version gating tests: pre-BB tenant never enters Divine Shield path; per-version formula differences covered where verification found them.
- [ ] `go test -race ./...`, `go vet ./...`, `go build ./...` clean in every changed module; `docker buildx bake atlas-channel` (and any other touched service) succeeds from repo root; `tools/redis-key-guard.sh` clean.
- [ ] Code review (three-reviewer pattern via `superpowers:requesting-code-review`) run before PR.
