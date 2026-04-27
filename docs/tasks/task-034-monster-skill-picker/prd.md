# Server-Side Mob Skill Picker & Movement Skill Injection — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-04-26
---

## 1. Overview

Atlas today only fires `UseSkill` when the inbound `MoveLife` packet from a monster's controller already contains a non-zero `skillId`. The controller's client decides whether the monster should cast a skill, by walking the WZ-derived skill list and rolling each skill's `prop` locally. This means the server is dependent on (and trusts) the client to roll skill choices. Modern v83 clients increasingly trim that logic, and behavior varies across clients, so monsters that *should* be casting (Iron Hog's WATK buff, Stirge's DARKNESS, Mushmom's stat buffs, Snowman's heal, etc.) frequently never do.

This feature moves skill selection to the server. atlas-monsters runs a per-monster picker that walks the skill list, filters by the eligibility gates the executor already enforces (cooldown, MP, HP% threshold, SEAL, reflect/immunity already-active), and rolls each candidate skill's `prop` independently per tick to choose at most one skill to fire on the next controller tick. atlas-monsters is the authoritative decision-maker and publishes its decision as a `NEXT_SKILL_DECIDED` event on `EVENT_TOPIC_MONSTER_STATUS`. atlas-channel mirrors the latest decision into a per-monster in-memory cache and writes the chosen skill bytes into the next outbound `MoveMonsterAck`. The controller's client receives that ack and dutifully sends the skill back in its next `MoveLife`, at which point the existing `UseSkill` Kafka path validates eligibility one more time and applies the effect.

The companion change is removing the redundant `prop` post-roll inside `UseSkill` (the picker already consumed `prop` per skill per tick; rolling again would dilute high-`prop` skills). Defense-in-depth in `UseSkill` becomes the eligibility re-check (cooldown, MP, HP%, status), not a probability re-roll. A latent bug in the animation-delay path (`m.Alive()` is not re-checked after the sleep, so dead monsters can still apply heals/banishes) is fixed in the same code path.

## 2. Goals

Primary goals:
- Make atlas-monsters the authoritative chooser of which mob skill, if any, fires next, independent of any client-side skill rolling.
- Inject the chosen skill into the `MoveMonsterAck` returned to the controller so the controller's client casts it on the next tick.
- Mirror v83 / classic-MapleStory mob skill cadence by rolling each skill's `prop` independently per movement tick (the per-skill-independent interpretation, not Cosmic's uniform-pick-then-roll).
- Re-pick reactively on every state change that could flip the decision (spawn, post-`UseSkill`, post-damage, picker-relevant status apply/expire, controller change) and on a periodic 1500ms sweep that catches cooldown expiry.
- Fix the dead-monster-applies-effect bug in the existing animation-delay path.

Non-goals:
- Implementing missing executors (mist / `AREA_POISON`, DoT mechanics, reflect tick mechanics). Deferred to Spec-Task 3.
- Boss multi-skill phase rotations, revive sequencing, or HP-band-driven skill scripting. Deferred to Spec-Task 4. The picker treats all monsters identically; bosses are not special-cased here.
- AI movement decisions. The controller's client still computes movement; the server only injects the skill field into the ack.
- New skills not present in atlas-data's monster information. The picker walks whatever skill list atlas-data already returns.
- Tenant-configurable picker tuning knobs. The picker has no tenant-tunable parameters beyond what is already in atlas-data's mob skill data.

## 3. User Stories

- As a player fighting Iron Hog, I want it to occasionally cast its WATK buff so I see the buff icon on its HP bar and feel the increased damage — without having to rely on my client to roll the skill.
- As a player walking near a Stirge, I want it to cast DARKNESS on me within ~10 seconds, so the encounter matches the v83 game's actual difficulty.
- As a boss raider fighting Mushmom, I want her stat-buff and disease skills to fire at roughly the cadence the WZ data describes (i.e. with `prop` honored as a per-tick fire chance), not at a Cosmic-flat rate.
- As an operator, I want to see in logs which skill the picker chose for a monster and why a re-pick was triggered, so I can debug "monster never casts" complaints.
- As a tenant operator running a non-v83 client variant that strips client-side mob-skill rolling, I want monsters to still cast their skills, because the server now drives the choice.

## 4. Functional Requirements

### 4.1 Picker algorithm

- **FR-1.** A new `pickNextSkill(m Model) (Decision, time.Time)` function lives in atlas-monsters' `monster` package. Inputs are the monster's current in-memory `Model` (HP, MP, status effects, controller, monster ID) plus access to the cooldown registry. The function reads atlas-data through the existing `information.GetById` and `mobskill.GetByIdAndLevel` REST clients.
- **FR-2.** The picker iterates the monster's skill list (`information.Model.Skills() []information.Skill`). For each `(skillId, level)` it fetches the corresponding `mobskill.Model` and runs the eligibility gates in this order:
  1. `skillId <= 255`. If not, log a warning and skip the skill (defensive guard against malformed atlas-data; never panic, never overflow).
  2. Cooldown not active: `!GetCooldownRegistry().IsOnCooldown(ctx, t, uniqueId, skillId)`.
  3. HP threshold satisfied: `sd.Hp() == 0 || m.HpPercentage() <= sd.Hp()`. Note: `sd.Hp()` is the **maximum** HP% at which the skill becomes eligible, mirroring existing behavior at `processor.go:486`.
  4. MP available: `m.Mp() >= sd.MpCon()`.
  5. Not sealed: `!m.HasStatusEffect("SEAL")`. (If sealed, the picker emits "no skill" — single global gate, no need to re-evaluate per-skill.)
  6. For reflect/immunity skills: not already-active. Mirrors `processor.go:519-527`.
  7. Skill category is **not** `AREA_POISON` (mist). Until the mist executor lands in Spec-Task 3, the picker excludes `AREA_POISON` from the eligible set. The exclusion lives as a single guarded condition with a TODO comment naming Spec-Task 3.
- **FR-3.** For each surviving candidate, the picker rolls `prop` independently: `rand.Intn(100) < int(sd.Prop())`. The first candidate whose roll succeeds becomes the decision. Iteration order is the order returned by `information.Model.Skills()` (insertion order from atlas-data, which is the WZ data order). Skills further down the list have lower a-priori chance of being chosen because earlier hits short-circuit; this matches the per-skill-independent interpretation of v83 mob skill semantics.
- **FR-4.** If no candidate's `prop` roll succeeds, the decision is "no skill" (sentinel `Decision{SkillId: 0, SkillLevel: 0}`).
- **FR-5.** The picker also computes `nextEligibleRepickAtMs`: the minimum cooldown expiry timestamp across all skills that are **currently gated only by cooldown** (i.e. would have been eligible if their cooldown were clear). If no skill is cooldown-gated, the value is `0` (sentinel: no scheduled re-pick needed). This timestamp drives the periodic sweep in §4.4.
- **FR-6.** The picker is pure with respect to monster state: it must not deduct MP, set cooldowns, or mutate the monster. All mutations happen later inside `UseSkill` if the controller's client follows through.

### 4.2 Decision storage and event emission

- **FR-7.** A new field `nextSkillDecision` is added to atlas-monsters' in-memory monster `Model`, holding `(skillId byte, skillLevel byte, decidedAtMs int64, nextEligibleRepickAtMs int64)`. Initial value on monster creation is the sentinel "no skill" decision; the spawn trigger runs the picker immediately and replaces it.
- **FR-8.** Whenever the picker runs, atlas-monsters writes the new decision to the monster registry and emits a `NEXT_SKILL_DECIDED` event on `EVENT_TOPIC_MONSTER_STATUS` with the body shape defined in §5. The event is **always** emitted on a picker run, even if the new decision is "no skill" or identical to the previous one. (Always-emit avoids cache-coherence bugs on atlas-channel: a stale cached decision can never outlive a picker run.)
- **FR-9.** The event is partition-keyed by `uniqueId` so per-monster ordering is preserved across atlas-channel consumers.
- **FR-10.** Decision events for a destroyed monster are not emitted; the decision is discarded along with the rest of the monster state. atlas-channel removes its cache entry on receipt of the existing `MONSTER_DESTROYED` event (no new wiring needed).

### 4.3 Re-pick triggers

The picker runs in atlas-monsters in response to **all** of the following events:

- **FR-11.** **Monster spawn.** Initial picker run happens during the existing monster-create flow, after the monster is committed to the registry. The first decision event is emitted before any `START_CONTROL` event, so the controller has a decision available the instant it takes ownership.
- **FR-12.** **Post-`UseSkill` completion.** After the executor's category dispatch returns (or after the animation-delayed effect runs, whichever is later in the code path), atlas-monsters re-runs the picker. The just-fired skill is now on cooldown, so the new decision typically excludes it.
- **FR-13.** **Post-damage application.** Within the existing damage handler (the path that mutates HP), after HP is updated, atlas-monsters re-runs the picker if the HP% bucket actually changed in a way that could un-gate a skill. To keep this cheap, the trigger fires whenever `oldHpPercentage != newHpPercentage`. The picker run itself is fast enough (no network calls in the steady state once atlas-data responses are cached) that conservative re-running is acceptable.
- **FR-14.** **Status effect apply/expire**, but only when the status touches the picker's gates: `SEAL`, `WEAPON_REFLECT`, `MAGIC_REFLECT`, `WEAPON_IMMUNITY`, `MAGIC_IMMUNITY`, `SEAL_SKILL`. Other statuses do not re-trigger the picker.
- **FR-15.** **Controller change.** When `START_CONTROL` is emitted (initial assignment or task-033's controller-switch path), the picker re-runs and emits a fresh decision so the new controller's atlas-channel cache is primed.
- **FR-16.** **Periodic sweep.** A new `MonsterSkillPickerSweepTask` runs every 1500ms across all monsters in all tenants, mirroring the cadence and scan pattern of `StatusExpirationTask` and `MonsterAggroDecayTask`. For each monster with `nextEligibleRepickAtMs > 0 && nextEligibleRepickAtMs <= now`, the sweep re-runs the picker. Monsters with sentinel `0` are skipped. The sweep does not iterate monsters with empty skill lists (cheap precondition check via `information.Model.Skills()`).

### 4.4 atlas-channel-side cache and packet injection

- **FR-17.** atlas-channel maintains an in-memory map `nextSkillCache: map[(tenantId, uniqueId)] → Decision`, keyed per-tenant for multi-tenancy isolation. The cache is per-channel-process and **not** persisted; on channel restart it re-hydrates from atlas-monsters' next emission cycle.
- **FR-18.** On receipt of a `NEXT_SKILL_DECIDED` event, atlas-channel writes (or replaces) the cache entry. Last-writer-wins keyed by `(tenantId, uniqueId)`; partition keying on `uniqueId` ensures ordering is correct.
- **FR-19.** On receipt of `MONSTER_DESTROYED` (existing event), atlas-channel removes the cache entry.
- **FR-20.** When atlas-channel handles a `MoveLife` packet in `movement/processor.go:ForMonster`, it reads the cache entry for the monster. If present and non-sentinel, it writes the bytes into the `MoveMonsterAck`:
  - `useSkills = true`
  - `skillId = decision.SkillId` (already byte-wide)
  - `skillLevel = decision.SkillLevel`
- **FR-21.** After writing the bytes into the ack, atlas-channel **clears** the cache entry (single-use prediction). The next `NEXT_SKILL_DECIDED` event from atlas-monsters re-populates it. This prevents the same prediction from being re-served across multiple ticks if the controller's client doesn't follow through on a tick.
- **FR-22.** If the cache entry is missing or sentinel ("no skill"), atlas-channel writes the existing default ack: `useSkills = false, skillId = 0, skillLevel = 0`.
- **FR-23.** The outbound broadcast `MoveMonster` packet (sent to all non-controller clients in the field) continues to forward the inbound `skillId/skillLevel` from the serverbound MoveLife verbatim, **not** the predicted skill. The predicted skill only goes into the ack to the controller; the controller's *next* MoveLife will carry that skill in its inbound shape, which is when the broadcast naturally picks it up.

### 4.5 `UseSkill` simplification and animation-delay fix

- **FR-24.** The post-pick `prop` re-roll inside `UseSkill` (`processor.go:512-517`) is **removed**. The picker is now the sole authority for `prop`; re-rolling here would dilute high-`prop` skills.
- **FR-25.** The eligibility re-check in `UseSkill` is **retained**: cooldown, MP cost, HP%, SEAL, reflect/immunity already-active. These act as defense-in-depth against picker/executor divergence (e.g. a stale cached decision served by atlas-channel for a monster whose state has since changed). On any failed re-check, `UseSkill` returns silently without consuming MP or setting cooldown.
- **FR-26.** The animation-delay goroutine at `processor.go:553-560` adds an `m.Alive()` re-check after the `time.Sleep`. Specifically, the goroutine re-fetches the monster from the registry by `uniqueId` and skips `executeEffect()` if the monster is no longer present or `!Alive()`. Mirrors Cosmic's `MobSkill.java:181-184`. A unit test covers both branches.
- **FR-27.** The integer types of `Processor.UseSkill`, `Processor.UseSkillGM`, and the cooldown registry's signatures change from `uint16` → `byte` for `skillId` and `skillLevel`. See §7 for the boundary types.

### 4.6 Picker exclusions and observability

- **FR-28.** Skills whose category is `AREA_POISON` (mist) are excluded from the eligible set. The exclusion is a named, single-line condition in the picker with a TODO comment naming Spec-Task 3. When the mist executor lands, this exclusion is removed in a single line.
- **FR-29.** The picker logs at debug level on every run: monster `uniqueId`, candidate count, gates each candidate failed (one log line per candidate at debug level), and the chosen skill (or "none"). The cooldown registry logs the skill ID and remaining cooldown when a candidate is filtered by cooldown.
- **FR-30.** atlas-monsters logs at info level when the picker emits a non-sentinel decision after previously holding a sentinel decision (i.e. the monster transitioned from "nothing to cast" to "casting X"), and vice versa. This makes it easy to grep production logs for "did this mob ever decide to cast anything?"
- **FR-31.** No new metrics are required; existing Kafka producer/consumer metrics (event volume, lag) cover observability of the new event topic body type.

## 5. API Surface

### 5.1 New Kafka event body — `EVENT_TOPIC_MONSTER_STATUS`, type `NEXT_SKILL_DECIDED`

Producer: atlas-monsters. Consumer: atlas-channel (new consumer handler), plus any future service that wants to observe picker decisions.

```go
const StatusEventTypeNextSkillDecided = "NEXT_SKILL_DECIDED"

type statusEventNextSkillDecidedBody struct {
    SkillId               byte  `json:"skillId"`
    SkillLevel            byte  `json:"skillLevel"`
    DecidedAtMs           int64 `json:"decidedAtMs"`
    NextEligibleRepickAtMs int64 `json:"nextEligibleRepickAtMs"`
}
```

- `SkillId == 0` and `SkillLevel == 0` is the sentinel "no skill chosen" (decision: do not write a predicted skill into the next ack).
- `DecidedAtMs` is `time.Now().UnixMilli()` at picker run time.
- `NextEligibleRepickAtMs == 0` means no cooldown-gated skills; sweep skips this monster. Otherwise it's the earliest cooldown expiry to re-evaluate.

The event uses the existing `EVENT_TOPIC_MONSTER_STATUS` envelope (already used by `damaged`, `apply_status`, `start_control`, `stop_control`, etc.). Partition key is the monster's `uniqueId`.

### 5.2 Modified Kafka command body — `monster.UseSkillCommandBody`

```go
type UseSkillCommandBody struct {
    CharacterId uint32 `json:"characterId"`
    SkillId     byte   `json:"skillId"`     // narrowed from uint16
    SkillLevel  byte   `json:"skillLevel"`  // narrowed from uint16
}
```

The producer (atlas-channel) narrows from the inbound MoveLife's `int16` with a guard: if `skillId < 0 || skillId > 255 || skillLevel < 0 || skillLevel > 255`, log a warning and **drop** the command without forwarding (treat as malformed input).

### 5.3 Modified Kafka command body — `monster.UseSkillFieldCommandBody` (GM)

The GM command body (`USE_SKILL_FIELD`) gets the same `byte` narrowing for consistency, applied in atlas-monsters' command consumer entry point. atlas-channel and any GM tooling that produces this command must narrow at the producer side; the handler logs and drops on overflow.

### 5.4 No HTTP/REST API changes

The picker is entirely Kafka-driven. There is no new REST endpoint, no new query parameter, and no change to the `monster` REST surface served by atlas-monsters.

## 6. Data Model

The picker introduces no database schema changes — all picker state is in atlas-monsters' in-memory monster registry, alongside HP/MP/status/controller. No Redis schema changes either; the cooldown registry is read-only by the picker and remains as-is.

The new field on `monster.Model`:

```go
type nextSkillDecision struct {
    skillId                byte
    skillLevel             byte
    decidedAtMs            int64
    nextEligibleRepickAtMs int64
}
```

Stored as part of the `Model`'s private fields with builder methods following the existing immutable model pattern (private field + getter + Builder). Initial value on construction: zero-valued struct (sentinel "no skill, no scheduled re-pick").

`storedMonster` (Redis representation) is **not** extended to carry the decision. The decision is per-process in-memory state on atlas-monsters; if atlas-monsters restarts, the picker re-runs as part of the existing monster-rehydration flow (the spawn-trigger equivalent on rehydration). Persisting the decision would be redundant given that the picker runs sub-millisecond once atlas-data responses are warm.

## 7. Service Impact

### 7.1 atlas-monsters

Changes:

- Add `pickNextSkill` in `monster/processor.go` (or a new `monster/picker.go` for clarity).
- Add `MonsterSkillPickerSweepTask` in `monster/picker_task.go`, registered alongside the existing `StatusExpirationTask` and `MonsterAggroDecayTask` task wiring.
- Wire re-pick triggers into the existing damage, status, controller, and `UseSkill` code paths.
- Extend `monster.Model` and `Builder` with `nextSkillDecision`.
- Add `statusEventNextSkillDecidedBody` and the corresponding producer in `monster/producer.go` / `monster/kafka.go`.
- Narrow `Processor.UseSkill`, `Processor.UseSkillGM`, and `cooldown.go` signatures from `uint16` → `byte`. Update tests.
- Add `m.Alive()` re-check guard on the animation-delay goroutine (`processor.go:553-560`).
- Remove the post-pick `prop` re-roll from `UseSkill` (`processor.go:512-517`).
- Picker entry-point widens `byte → uint16` when calling `mobskill.GetByIdAndLevel(uint16, uint16)` so the REST client surface stays untouched.
- Picker entry-point narrows `uint32 → byte` when reading `information.Skill{Id, Level}`, with the defensive `id > 255 || level > 255 → skip + warn` guard.

Surfaces unchanged:
- `mobskill.Model.SkillId() uint16`, `mobskill.RestModel`, `information.RestModel` — all REST-facing types remain as they are, since the schema is owned by atlas-data.

### 7.2 atlas-channel

Changes:

- New consumer handler for `NEXT_SKILL_DECIDED` events on `EVENT_TOPIC_MONSTER_STATUS` (extend the existing monster-status consumer at `services/atlas-channel/atlas.com/channel/kafka/consumer/monster/`).
- New in-memory cache map `nextSkillCache: map[(tenantId, uniqueId)] → Decision` with sync-protected reads/writes, scoped per-channel process.
- Extend `movement/processor.go:ForMonster` to read the cache and write into `MonsterMovementAck`'s `useSkills/skillId/skillLevel` fields. Clear the cache entry on serve.
- Cache invalidation on `MONSTER_DESTROYED` (extend the existing destroyed-event handler).
- Narrow `monster.Processor.UseSkill` signature and `monster.UseSkillCommandProvider` body type from `uint16` → `byte`. Add the `int16 → byte` guard with bounds-check at the producer site (the MoveLife handler path).

### 7.3 libs/atlas-packet

No packet shape changes. The existing `MovementAck.skillId byte` and `MovementAck.skillLevel byte` fields are already present and were always being written as `0, 0` placeholders. They start carrying real values once atlas-channel populates them.

The serverbound `MovementRequest.SkillId() int16` and `SkillLevel() int16` getters remain unchanged; the bounds-check happens in atlas-channel's MoveLife handler when narrowing to `byte` for the Kafka command body.

### 7.4 libs/atlas-constants

No changes. The skill ID enum / category mapping (`monster2.SkillCategory`, `monster2.SkillTypeToStatusName`, etc.) is already byte-wide in semantics and continues to work after narrowing.

### 7.5 Other services

No other service consumes mob skill data directly. atlas-data continues to serve `mobskill` and `information` REST resources unchanged.

## 8. Non-Functional Requirements

### 8.1 Performance

- Picker runs are O(N skills × constant) where N is typically 1–4 per monster. The cooldown lookup is a single Redis `EXISTS` per skill; HP%, MP, and status are local memory reads. With atlas-data responses cached client-side (existing behavior of the `requests` library), the picker takes well under 1ms per call in the steady state.
- Periodic sweep at 1500ms interval iterates all monsters in all tenants. With timestamp-prefiltering (`nextEligibleRepickAtMs <= now` and `len(skills) > 0`), the per-tick cost scales with the number of mobs currently in cooldown-gated states, not the full monster population.
- Trigger 3 (post-damage) is the highest-volume re-pick path. Conservative re-running on every HP% change is acceptable because the picker is sub-millisecond. If profiling shows hotspots later, debounce strategies are additive and don't change the contract.
- `NEXT_SKILL_DECIDED` event volume: roughly 1 event per significant state change per monster. For 100 actively-fought monsters, expect on the order of 100–1000 events/sec aggregate. Comparable to existing `damaged` event volume; well within Kafka throughput headroom.

### 8.2 Multi-tenancy

- Picker state and decisions are scoped per-tenant via the existing `tenant.Model` propagation through `ProcessorImpl(p.l)(p.ctx)` and registry keys. atlas-channel's `nextSkillCache` is keyed on `(tenantId, uniqueId)` — never mix tenants in lookups.
- `NEXT_SKILL_DECIDED` events carry the tenant header through the existing message envelope.
- The 1500ms sweep iterates monsters per-tenant using the existing per-tenant scan pattern from `StatusExpirationTask`; each tenant's monsters are processed in their own context.

### 8.3 Observability

- Debug-level picker logs (FR-29) are sufficient for diagnosing "this mob never picks a skill" complaints. Operators can grep for `pickNextSkill` lines and see candidate count, per-skill gate failures, and the chosen result.
- Info-level state-transition logs (FR-30) make picker activity visible without enabling debug.
- Existing Kafka observability (consumer lag, producer throughput, error counts on `EVENT_TOPIC_MONSTER_STATUS`) covers the new event body without additional instrumentation.

### 8.4 Security

- The picker runs entirely inside atlas-monsters' trust boundary. Inbound `USE_SKILL` Kafka commands continue to flow only from atlas-channel.
- The new MoveLife `int16 → byte` bounds-check at atlas-channel's producer prevents a misbehaving or hostile client packet from overflowing the Kafka body. Rejected commands are logged and dropped.
- No new authentication or authorization surface.

### 8.5 Failure modes

- **atlas-monsters down**: no decision events emitted. atlas-channel's cache eventually empties as monsters are destroyed; new monsters spawn with no decision. Fallback is the pre-feature behavior: monsters cast skills only when the controller's client picks them. Not a regression vs. today.
- **atlas-channel restarted**: cache is cleared. Monsters miss skill predictions for one re-pick cycle (≤1500ms in the worst case via the sweep, immediate on the next damage/status event in practice). Recovers without operator intervention.
- **Stale cached decision** (e.g. cooldown elapsed between emission and serve): `UseSkill`'s eligibility re-check catches it. The skill silently no-ops; no MP or cooldown is consumed. The picker re-runs after the next state change and emits a fresh decision.
- **Malformed atlas-data skill ID** (`Id > 255`): picker logs a warning and skips that skill. Other skills in the list still work.

## 9. Open Questions

None blocking implementation. Items deferred to design phase:

- Whether to share the 1500ms sweep tick goroutine with `MonsterAggroDecayTask` (single timer dispatching to multiple sweepers) or keep a dedicated timer for the picker. Either is acceptable; the design phase can pick whichever is cleaner with the existing task package's API.
- Exact placement of the `pickNextSkill` function: in `monster/processor.go` next to `UseSkill`, or in a new `monster/picker.go` for separation. Lean toward `picker.go` for testability and code-locality, but defer to design.
- Whether a "no skill, no candidates eligible" decision is logged at info level vs debug. Affects log volume; design phase can tune.

## 10. Acceptance Criteria

### 10.1 Behavioral acceptance — manual/operator verification

Tested by logging into a tenant configured with v83 GMS data, finding the named mob, and observing in-game:

- **Iron Hog (mob ID 4090000)** — within ~30 seconds of engagement, the WEAPON_ATTACK_UP buff icon appears on its HP bar; subsequent monster auto-attacks deal noticeably more damage.
- **Stirge** — within ~10 seconds of engagement, your character receives the DARKNESS disease; the screen visibly darkens and your accuracy drops.
- **Drumming Bunny / Toy Trojan** — DEFENSE_UP buff icon appears on the mob's HP bar; your damage drops accordingly.
- **Snowman** — Snowman's HP visibly refills mid-fight (heal skill).
- **Mushmom (boss)** — across a single fight, multiple distinct skills fire (e.g. stat buff plus a disease application). At least two distinct skill types should be observed in one engagement.
- **Rurumo** — SEAL disease applied to player; player's skill bar grays out.
- **Mob with `AREA_POISON` skill** (e.g. Big Spider) — the picker does **not** select the mist skill; logs confirm the exclusion is hit. Other skills on the same mob (if any) still fire.

### 10.2 Code-level acceptance — automated tests

- Unit tests for `pickNextSkill` covering: empty skill list, single-skill HP-gated, single-skill cooldown-gated, single-skill MP-gated, sealed monster, reflect-already-active, `AREA_POISON` exclusion, byte-overflow defensive guard, `prop` roll determinism (with a mocked RNG).
- Unit test for `nextEligibleRepickAtMs` computation: zero when no cooldown gates, minimum-of-cooldowns when multiple skills are cooldown-gated.
- Unit test for `m.Alive()` guard: dead monster's animation-delayed effect does not run.
- Unit test for `UseSkill` post-pick `prop` removal: with `prop=50` and a fixed-seed RNG that would have failed, the executor still runs (because the picker is the only `prop` gate now).
- Unit test for atlas-channel cache: serve-and-clear semantics, cache miss on missing entry, cache eviction on `MONSTER_DESTROYED`.
- Unit test for atlas-channel's `int16 → byte` overflow guard on the MoveLife producer path: out-of-range inbound skill values are dropped without forwarding.
- Sweep task test: monster with `nextEligibleRepickAtMs <= now` triggers a re-pick; monster with sentinel zero is skipped.

### 10.3 Integration acceptance

- All affected services build (`go build ./...` in each service directory).
- All affected services test (`go test ./...` in each service directory).
- Docker builds succeed for `atlas-monsters`, `atlas-channel`, `libs/atlas-packet` (smoke build).
- A Bruno or equivalent end-to-end smoke run: spawn a mob with skills via the GM `USE_SKILL_FIELD` command, observe a `NEXT_SKILL_DECIDED` event flow through Kafka after engagement, observe the byte payload in the next ack.

### 10.4 Definition of done

- All 10.1 manual mobs verified by the implementer (or a tester) on a running tenant.
- All 10.2 unit tests pass.
- All 10.3 build/test gates pass.
- The redundant `prop` re-roll in `UseSkill` is gone; the `m.Alive()` guard is in place; the picker exclusion list is documented in code with the Spec-Task 3 TODO.
- Logs at info level confirm picker activity on at least one of the test mobs in 10.1 during verification.
