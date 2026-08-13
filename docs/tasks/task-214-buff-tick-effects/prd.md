# Player-Buff Periodic Tick Effects — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-08-12
---

## 1. Overview

atlas-buffs stores every character buff as a set of temporary-stat changes (`buff/stat.Model`) and emits `APPLIED` / `EXPIRED` status events so the channel can paint the client's buff icons. For the overwhelming majority of statups that is the whole story — the stat sits there until it expires, and atlas-effective-stats folds it into combat math.

A small number of buffs are different: they carry an ongoing *periodic side effect* on the character's HP or MP for as long as the buff is active. Today atlas-buffs implements exactly one of them — POISON — via a bespoke path: `character/registry.go` `GetPoisonCharacters` scans every stored buff for a change whose `Type() == "POISON"`, `tasks/poison.go` runs a 1 s tick task, `character/processor.go` `ProcessPoisonTicks` throttles per-character with a dedicated `poisonTicks` Redis registry and emits `COMMAND_TOPIC_CHARACTER` / `CHANGE_HP`.

Every other periodic statup is applied but inert. Dragon Blood (`1311008`) publishes a `DRAGON_BLOOD` statup (`services/atlas-data/atlas.com/data/skill/reader.go:341-342`) and the client draws the icon, but the caster never pays the HP cost. Recovery (`BeginnerRecoveryId` / `NoblesseRecoveryId` / `LegendRecoveryId` / `EvanRecoveryId`) publishes a `RECOVERY` statup (`reader.go:317-318`) and heals nothing. The buffs are cosmetic.

This task replaces the POISON-only special case with a **periodic-effect table**: a declarative registry of `statType -> {interval, resource, direction, floor}` rows that one generic tick task drives. POISON becomes a row rather than a hard-coded string; DRAGON_BLOOD and RECOVERY become rows; and the task includes an explicit audit sweep over every statup atlas-data can emit so that any *other* statup with a periodic schedule is either wired in the same change or recorded as deliberately excluded with a reason.

## 2. Goals

Primary goals:

- Replace the hard-coded `c.Type() == "POISON"` scan and the `poisonTicks` single-purpose store with a generic periodic-effect mechanism keyed by `(characterId, statType)`.
- Make Dragon Blood drain the caster's HP on its own cadence, without ever killing the caster.
- Make Recovery heal the character on its own cadence.
- Complete an audit sweep of every temporary-stat type atlas-data emits, wiring the ones with a genuine periodic schedule and documenting the exclusions.
- Preserve POISON's observable behavior exactly — same cadence, same damage, same command shape.

Non-goals:

- Changing how buffs are applied, stored, expired, or how `APPLIED`/`EXPIRED` events are shaped.
- Berserk. `berserk/` is a reactive HP-*threshold* re-evaluation (does the character's HP fraction cross the skill's `x`?), not a periodic resource change; it keeps its own tick task and cache.
- Combo Drain — HP recovery proportional to damage dealt, driven by attack events, not a timer. Owned by task-166 (`.worktrees/task-166-combo-drain`).
- Any client-side / packet work. The tick emits an existing command on an existing topic; the HP/MP bar update is already handled by atlas-character's `STAT_CHANGED` event.
- Making the intervals tenant- or version-configurable.

## 3. User Stories

- As a Dragon Knight, when I cast Dragon Blood I want my HP to visibly drain on a fixed cadence so that the skill's power carries its intended cost.
- As a Dragon Knight, I want Dragon Blood's self-drain never to be the thing that kills me, so that a buff I cast on myself cannot be a suicide button.
- As a Beginner (or Noblesse / Legend / Evan) with Recovery active, I want to regain HP on a fixed cadence so the skill does what its description says.
- As a service maintainer, I want a new periodic buff effect to be a table row plus a test, not a new tick task, a new Redis registry, and a new registry scan.
- As a service maintainer, I want a written record of which statups were evaluated for periodic behavior and why each one was included or excluded, so the next person does not have to redo the sweep.

## 4. Functional Requirements

### FR-1 — Periodic-effect table

**FR-1.1** Define a declarative table in atlas-buffs mapping a temporary-stat type to its periodic behavior. Each row carries at minimum:

| Field | Meaning |
|---|---|
| `statType` | the `character.TemporaryStatType` string stored on the buff change (e.g. `"POISON"`, `"DRAGON_BLOOD"`, `"RECOVERY"`) |
| `interval` | the cadence between ticks for this effect |
| `resource` | `HP` or `MP` |
| `direction` | drain (negative) or restore (positive) |
| `floor` | whether the effect may reduce the resource to 0 (and thus emit `DIED`), or must clamp at 1 |

**FR-1.2** The table is the only place a stat type is named. No tick-path code may compare a stat type to a string literal outside the table.

**FR-1.3** Intervals are compile-time constants in atlas-buffs, not configuration and not fetched per tick.

**FR-1.4** The per-tick magnitude is the `Amount()` already carried on the stored buff change (the WZ `x` value that `atlas-data`'s `produceBuffStatAmount` put there). The tick path makes **no** call to atlas-data.

**FR-1.5** A row whose per-tick magnitude resolves to 0 is skipped (no command emitted). This preserves today's poison guard (`amount >= 0 { continue }`).

### FR-2 — Generic tick task

**FR-2.1** Replace `Registry.GetPoisonCharacters` with a generic scan that returns, per tenant, every `(character, statType, amount, worldId, channelId)` tuple whose stat type appears in the periodic-effect table and whose buff is not expired.

**FR-2.2** Replace the `poisonTicks` Redis registry with a last-tick store keyed by `(characterId, statType)` so two periodic effects on the same character throttle independently.

**FR-2.3** A single tick task drives all rows. Its sleep interval must be fine enough to honor the shortest row's cadence; each row is emitted only when `now - lastTick >= row.interval`.

**FR-2.4** POISON's existing 1 s cadence and negative-`Amount`→`CHANGE_HP` semantics are preserved byte-for-byte in the emitted command.

**FR-2.5** The tick path stays multi-tenant: iterate `GetTenants`, run per-tenant work under `tenant.WithContext`, spawn via `routine.Go` (never a bare `go`), exactly as `character.ProcessPoisonTicks` does today.

**FR-2.6** Ticks are emitted through `message.Emit` / `message.Buffer` as today, keyed by `characterId`.

### FR-3 — Dragon Blood (`1311008`)

**FR-3.1** `DRAGON_BLOOD` is a table row: HP, drain direction, floor-at-1.

**FR-3.2** The cadence is 4 s.

**FR-3.3** The per-tick drain magnitude is the `DRAGON_BLOOD` statup amount stored on the buff. The design phase MUST confirm against WZ data (`Skill.wz` effect node for `1311008`) that the value `reader.go:342` stores (`e.X()`) is the HP cost per tick and not the weapon-attack bonus; if it is the attack bonus, the drain magnitude must come from the correct effect field and `reader.go` must carry whatever additional value the tick needs. **Do not implement against the assumption — verify.**

**FR-3.4** The drain MUST NOT kill the caster. atlas-character's `ChangeHP` emits a `DIED` status event whenever the adjusted HP lands on 0 (`services/atlas-character/atlas.com/character/character/processor.go:1312-1322`), so a raw negative `CHANGE_HP` is unsafe. atlas-buffs must read the character's current HP and emit a reduced amount such that HP floors at 1; when current HP is already 1, no command is emitted for that tick.

**FR-3.5** The buff itself is unaffected by hitting the floor — it stays applied and keeps ticking (no-op ticks) until it expires or is cancelled.

**FR-3.6** Current HP is read from the existing `external/character` client (`RestModel.Hp`, already used by the berserk path). Fetch at most once per character per tick pass, not once per row.

### FR-4 — Recovery (`BeginnerRecoveryId`, `NoblesseRecoveryId`, `LegendRecoveryId`, `EvanRecoveryId`)

**FR-4.1** `RECOVERY` is a table row: HP, restore direction.

**FR-4.2** The cadence is 5 s.

**FR-4.3** The per-tick heal magnitude is the `RECOVERY` statup amount (`e.X()`, `reader.go:318`), confirmed against WZ data in the design phase.

**FR-4.4** The heal is a positive `CHANGE_HP`; atlas-character already clamps to effective MaxHP (`enforceBounds`, `processor.go:1305`), so atlas-buffs does not need to cap it.

**FR-4.5** No overheal command: if the design phase confirms atlas-character emits a `STAT_CHANGED` on every `CHANGE_HP` regardless of whether the value moved, evaluate skipping the emit when the character is already at effective MaxHP to avoid a 5 s heartbeat of no-op stat events per buffed character. If the value cannot be determined without an extra service call, emit unconditionally and record the decision.

### FR-5 — Audit sweep over all statups

**FR-5.1** Enumerate every `character.TemporaryStatType` that atlas-data can attach to a character buff — the authoritative list is the set of `TemporaryStatType*` constants referenced in `services/atlas-data/atlas.com/data/skill/reader.go` plus any emitted by consumable/item readers.

**FR-5.2** For each, record a verdict: **wired** (periodic → table row), **excluded** (not periodic), or **deferred** (periodic but owned elsewhere), each with a one-line reason and a source citation (WZ node, client decompile, or the owning Atlas service/task).

**FR-5.3** Candidates that the spec scan already flags as needing a verdict — not an assertion that they are periodic, a list of things to check:

| Stat type | Why it is on the list |
|---|---|
| `DRAGON_BLOOD` | in scope, FR-3 |
| `RECOVERY` | in scope, FR-4 |
| `POISON` | already ticking; becomes a table row |
| `INFINITY` | commonly implemented as a periodic HP/MP restore + escalating magic attack; must be checked against WZ and the client |
| `BODY_PRESSURE` | an ongoing aura; check whether its effect is periodic on the *caster's* resources or purely an outbound mob effect |
| `COMBO_DRAIN` | periodic-adjacent but damage-driven; expected verdict **deferred** to task-166 |
| `MESO_GUARD`, `MAGIC_GUARD`, `PICK_POCKET`, `HOMING_BEACON`, `PUPPET`, `SUMMON` | event-driven, not timer-driven; expected verdict **excluded**, confirm |

**FR-5.4** The sweep's output is a committed table in the task folder (`audit-statups.md` or a section of `design.md`). A statup with no verdict is a failed sweep.

**FR-5.5** Any statup the sweep finds to be genuinely periodic and unowned is wired in this task as an additional row — not filed as a follow-up. (Project rule: no deferring producible work.)

### FR-6 — Lifecycle

**FR-6.1** When a buff carrying a periodic stat type is cancelled or expires, its last-tick entry for that `(character, statType)` is removed. Today `ClearPoisonTick` exists but has **no callers** (`character/registry.go:344`) — the generic replacement must actually be wired into the cancel/expire paths.

**FR-6.2** A character with no active periodic buffs leaves no residual last-tick keys in Redis.

**FR-6.3** Channel/world changes do not reset a tick schedule; the entry is keyed by character, and the emitted command carries the buff's current `worldId`/`channelId` as stored on the character model.

## 5. API Surface

No new or modified REST endpoints.

No new Kafka topics or command types. The tick path uses the existing:

- `COMMAND_TOPIC_CHARACTER` / `CHANGE_HP` — `ChangeHPCommandBody{ChannelId, Amount int16}` (already produced by `character/producer.go:44`).
- `COMMAND_TOPIC_CHARACTER` / `CHANGE_MP` — `ChangeMPBody`, already consumed by atlas-character (`kafka/consumer/character/consumer.go:266`). atlas-buffs does not currently produce it; a mirror provider is added **only if** the FR-5 sweep wires an MP-resource row.

## 6. Data Model

No relational schema change. atlas-buffs is Redis-backed.

Redis key change: the `buffs-poison` `TenantRegistry[uint32, time.Time]` is replaced by a periodic-tick store whose key includes the stat type. The design phase chooses between a new prefix with a composite key encoder (e.g. `buffs-tick` keyed `"<characterId>:<statType>"`) and one registry per stat type; either way the change must respect `tools/redis-key-guard.sh` (all keyed access through `libs/atlas-redis`).

Migration: none required. Tick entries are ephemeral throttle state — a stale `buffs-poison:*` key set left behind by the deploy is harmless and self-evidently orphaned. If the design prefers cleanliness, note the abandoned prefix in the plan rather than writing a migration.

## 7. Service Impact

| Service | Change |
|---|---|
| **atlas-buffs** | All implementation. New periodic-effect table; generic tick scan replacing `GetPoisonCharacters`; composite-keyed last-tick store replacing `poisonTicks`; single tick task replacing `tasks/poison.go`; HP read for the Dragon Blood floor; cancel/expire wiring for FR-6.1. `main.go` registers one periodic task where it registers `NewPoisonTick` today. |
| **atlas-data** | Read-only unless FR-3.3 / FR-4.3 finds that `reader.go` stores the wrong effect field for the tick magnitude, in which case the statup mapping is corrected there. |
| **atlas-character** | None. Existing `CHANGE_HP` / `CHANGE_MP` consumers absorb the traffic unchanged. |
| **atlas-channel** | None. HP/MP bar updates already flow from atlas-character's `STAT_CHANGED`. |
| **atlas-effective-stats** | None. |

## 8. Non-Functional Requirements

- **Load.** The tick pass scans every stored buff of every online character per tenant, as it does today. Adding rows must not add a scan pass: one traversal yields all rows for all effects. The Dragon Blood HP read is the only new outbound call and must be bounded to at most one per *affected* character per pass — characters with no floor-sensitive row make no call.
- **Multi-tenancy.** Every registry access resolves the tenant from context via `tenant.MustFromContext`; no cross-tenant leakage of tick state.
- **Concurrency.** Per-tenant work spawns through `routine.Go` (`tools/goroutine-guard.sh` bans bare `go`). Ticks for one character are emitted under one `characterId` partition key, preserving ordering with other buff commands.
- **Observability.** Each emitted tick logs at debug with character, stat type, and amount (mirroring today's poison log line). A tick suppressed by the HP floor logs at debug with the reason.
- **Correctness under redelivery.** The last-tick store is the throttle; a duplicated tick pass within one interval must not double-apply. Ticks are inherently non-idempotent HP mutations, so the throttle check and the store update must not straddle a failure window that can double-emit.
- **Verification.** `go test -race ./...`, `go vet ./...`, `tools/redis-key-guard.sh`, `tools/goroutine-guard.sh`, and `tools/lint.sh --check` clean. `go.mod` is not expected to change; if it does, `docker buildx bake atlas-buffs` is mandatory.

## 9. Open Questions

1. **Q (blocking design, not implementation):** Is Dragon Blood's `x` in the `1311008` WZ effect node the HP cost per tick or the weapon-attack bonus? `reader.go:342` stores `e.X()` as the `DRAGON_BLOOD` statup amount — which the client uses to render the buff, and which FR-3.3 wants to reuse as the drain. If those are different fields, the statup amount cannot serve both purposes. Resolve from WZ data before writing the row.
2. Same question for Recovery's `x` (`reader.go:318`) — heal amount per tick, confirmed against WZ.
3. Does the v83 client render anything on a Dragon Blood tick (a self-damage number, a flash) that requires a packet atlas-buffs does not send today? Expected answer: no — HP change alone — but confirm rather than assume.
4. Does `INFINITY` belong in the table (FR-5.3)? Its verdict decides whether an MP-resource row and a `CHANGE_MP` provider are in scope.
5. Should the floor-at-1 guard be generic (any drain row is floor-sensitive) or per-row? Current answer: per-row `floor` field, so a future genuinely-lethal drain is expressible; POISON keeps today's behavior of being allowed to reach 0.

## 10. Acceptance Criteria

- [ ] A periodic-effect table exists in atlas-buffs; no tick-path code compares a stat type against a string literal outside it.
- [ ] `GetPoisonCharacters` and the `poisonTicks` registry are gone, replaced by the generic scan and the `(characterId, statType)`-keyed store.
- [ ] Exactly one periodic tick task is registered in `main.go` (alongside expiration and berserk); `tasks/poison.go` no longer exists as a separate effect-specific task.
- [ ] POISON behavior is unchanged: same 1 s cadence, same `CHANGE_HP` amount, verified by the existing `tasks/poison_test.go` assertions (ported, not deleted).
- [ ] Casting Dragon Blood (`1311008`) drains the caster's HP every 4 s, at the magnitude confirmed from WZ in Q1.
- [ ] Dragon Blood's drain floors the caster at 1 HP; a caster at 1 HP with Dragon Blood active never receives a `DIED` event from the tick, verified by a unit test asserting the emitted `CHANGE_HP` amount.
- [ ] Recovery (all four beginner-class ids) heals the character every 5 s at the magnitude confirmed from WZ in Q2.
- [ ] Recovery does not push HP past effective MaxHP (relies on atlas-character's `enforceBounds`); verified by a test that the emitted amount is positive and unclamped by atlas-buffs.
- [ ] Two periodic effects active on one character tick independently on their own cadences, verified by a unit test.
- [ ] Cancelling or expiring a periodic buff removes its last-tick entry, verified by a test — `ClearPoisonTick`'s zero-caller state is not reproduced.
- [ ] The FR-5 sweep is committed with a verdict + citation for every statup atlas-data emits; every statup found periodic and unowned is wired in this branch.
- [ ] `go test -race ./...`, `go vet ./...`, `tools/redis-key-guard.sh`, `tools/goroutine-guard.sh`, `tools/lint.sh --check` all clean.
- [ ] No `// TODO`, stub, or deferred-work marker in the landed diff.
