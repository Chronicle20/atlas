# Dark Knight Berserk (1320006) — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-07-10
---

## 1. Overview

Berserk (Dark Knight, skill id 1320006) is a passive that activates while the character's HP is below a per-skill-level percentage threshold of max HP. The client renders a persistent aura on the character while active and computes the damage bonus itself; the server's job is to (a) continuously evaluate HP against the threshold and (b) broadcast the on/off state to the character and to other players in the map so the aura renders everywhere.

The reference behavior is Cosmic `Character.checkBerserk` (verified against the local Cosmic checkout, `src/main/java/client/Character.java:1843-1870`): whenever HP changes, a berserk-flagged stat effect registers, or the player logs in, the server recomputes `hp * 100 / currentMaxHp < effect.getX()` (strict less-than, buff-inclusive max HP, threshold from the skill effect's per-level `x`) and (re)schedules a repeating broadcast — 5s initial delay, 3s period — of the skill-use effect packet carrying the skill level and the captured berserk bool, to the character itself and to everyone else in the map. The WZ `berserk` field on the effect only marks the effect as berserk-type (`StatEffect.isBerserk`); the numeric threshold comes from `x`.

Atlas has the entire packet layer already in place: `libs/atlas-packet/character/clientbound.EffectSkillUse` encodes the Berserk flag byte (`berserkDarkForce`), `libs/atlas-constants/skill.DarkKnightBerserkId` exists, and atlas-channel has `CharacterEffectWriter`/`CharacterEffectForeignWriter` with `CharacterSkillUseEffectBody`/`...ForeignBody` helpers (`services/atlas-channel/.../socket/handler/effects.go`). What is missing is the domain logic — nothing tracks HP against the threshold, holds per-character berserk state, or drives the periodic broadcast. Per the scope interview, that logic lives in **atlas-buffs**.

## 2. Goals

Primary goals:
- A Dark Knight with Berserk leveled sees the berserk aura activate when HP drops below the skill's `x`% threshold and deactivate when HP rises to/above it, matching Cosmic semantics (strict `<`, buff-inclusive max HP).
- Other players in the same map see the aura state on the Dark Knight, including players who enter the map while the state is active (covered by the periodic re-broadcast).
- State is re-evaluated on every trigger Cosmic honors, mapped to Atlas equivalents: login, HP change, max-HP-affecting buff apply/expire/cancel, and Berserk skill level change; plus channel transfer and logout for registry lifecycle.
- Broadcast cadence is Cosmic parity: repeating effect broadcast with 5s initial delay and 3s period per tracked character; each re-evaluation replaces the schedule with freshly computed state.
- The threshold percentage comes from the tenant's skill-effect data (`x` at the character's Berserk level) — no hard-coded thresholds or levels (DOM-25 posture: data-resolved, not literal).

Non-goals:
- Server-side damage amplification or damage validation for Berserk — the client computes damage; nothing server-side consumes the berserk state beyond the effect broadcast.
- GM-hide handling (Cosmic broadcasts to GMs only when the Dark Knight is hidden). Atlas will grow a hide concept eventually; tracked as an open question / follow-up, not implemented here.
- Packet or writer changes — `EffectSkillUse` and its Berserk flag are already implemented and fixtured in `libs/atlas-packet`.
- Other Dark Knight mechanics (Beholder healing/buff schedules, Dragon Blood drain — the latter is backlog item 25).
- Evan Dragon Fury (1211011/`EvanStage8DragonFuryId`) — same packet flag family, different mechanic, different task.

## 3. User Stories

- As a Dark Knight, I want the Berserk aura to appear on my character as soon as my HP falls below the skill's threshold so that I know the damage bonus is active.
- As a Dark Knight, I want the aura to disappear when I heal above the threshold so that the displayed state matches the actual mechanic.
- As a player in the same map as a Dark Knight, I want to see their berserk aura (including when I enter the map while it is already active) so that their state renders consistently for everyone.
- As a Dark Knight relogging or changing channels while below threshold, I want the aura restored without re-triggering an HP change so that state survives session transitions.
- As a server operator, I want the threshold to come from the tenant's skill data so that different game versions resolve their own values.

## 4. Functional Requirements

### FR-1 — State computation

Berserk state for a character is:

```
active := skillLevel > 0
       && hp * 100 / effectiveMaxHp < x(skillLevel)
```

- `skillLevel` is the character's level in skill 1320006 (`skill.DarkKnightBerserkId` from `libs/atlas-constants/skill`; no numeric literals — DOM-21).
- `hp` is the character's current HP.
- `effectiveMaxHp` is the **buff-inclusive** max HP from atlas-effective-stats (so Hyper Body apply/expire can flip the state), mirroring Cosmic's `getCurrentMaxHp()`.
- `x(skillLevel)` is the `x` field of the skill effect at that level, from the tenant's skill-effect data pipeline. The `berserk` WZ field is a type marker only; it MUST NOT be used as the threshold. (Per-version `x` values are data-resolved at runtime; design phase verifies the v83 WZ values locally per the grounding rule.)
- Comparison is strict less-than (Cosmic parity: `Character.java:1852`).
- Characters with `skillLevel == 0` (which includes all non-Dark-Knights) are not tracked at all — no registry entry, no ticker, no broadcasts.

### FR-2 — Re-evaluation triggers

Berserk state MUST be recomputed, and the broadcast schedule replaced, on:

| Trigger | Cosmic analogue | Atlas signal |
|---|---|---|
| Login | `PlayerLoggedinHandler.java:365` | character status event `LOGIN` |
| HP change (any source: damage, heal, potions, HP cost skills) | `hpChangeAction`, `Character.java:8900` | character status event `STAT_CHANGED` where the updates include HP |
| Max-HP-affecting buff apply/expire/cancel (e.g. Hyper Body) | `registerEffect`, `Character.java:4446` + buff-inclusive max HP | atlas-buffs' own apply/expire/cancel flow for buffs whose stat-ups affect max HP |
| Berserk skill level change (SP allocation) | implicit via next check | atlas-skills status event `UPDATED` for skill 1320006 |
| Channel/map transfer | login handler re-runs on transfer | character status events already consumed for lifecycle (`CHANNEL_CHANGED` / `MAP_CHANGED`) — update registry world/channel so broadcasts route correctly |
| Logout / disconnect | schedule cancelled with session | character status event `LOGOUT` — remove registry entry, stop ticker |

Death handling: Cosmic skips `checkBerserk` when the HP change is a death (`hpChangeAction` branches to `playerDead()`). Atlas MUST stop broadcasting for a dead character; recomputation on the revive/respawn HP change naturally restores correct state (at full-HP revive the state is off).

### FR-3 — Broadcast behavior (Cosmic parity)

- While a character is tracked (skillLevel > 0 and logged in), the server broadcasts the berserk effect on a repeating schedule: **5s initial delay, 3s period** (`Character.java:1867`).
- Each broadcast tick sends, with the state captured at the last re-evaluation (not recomputed per tick — Cosmic parity):
  - To the character: `CharacterEffectWriter` with `CharacterSkillUseEffectBody(1320006, characterLevel, skillLevel, berserkActive, false, false)` — the `darkForceEffect` bool is the on/off flag the encoder writes for the Berserk skill id.
  - To other sessions in the character's map: `CharacterEffectForeignWriter` with `CharacterSkillUseEffectForeignBody(...)`, same flag.
- Broadcasts are sent both when active and when inactive (Cosmic sends the packet with `berserk=false` too — this is what clears the aura client-side and keeps late-joining observers consistent).
- Every re-evaluation (FR-2) cancels the existing schedule and starts a new one with the fresh state, resetting the 5s initial delay (Cosmic parity: `checkBerserk` head cancels `berserkSchedule`).
- Ownership split: **atlas-buffs owns state + tick timing** (per scope interview decision); **atlas-channel owns packet emission**, consuming a Kafka status event from atlas-buffs and writing both packets. Whether each 3s tick is one Kafka event from the buffs ticker, or buffs emits only state changes and channel re-broadcasts on its own 3s timer, is a design-phase decision — the on-the-wire client behavior above is the requirement either way.

### FR-4 — State tracking in atlas-buffs

- atlas-buffs maintains an in-memory, tenant-scoped registry of tracked Dark Knights: character id → world id, channel id, character level, Berserk skill level, current berserk-active bool, and schedule handle. Registry follows the established singleton pattern (`sync.Once` + `sync.RWMutex`), storing world/channel per character because tickers have no event context (same reasoning as the existing poison registry entries, `services/atlas-buffs/.../character/registry.go:285-286`).
- Entries are created at login (or at first re-evaluation trigger post-login) only when the character has Berserk level > 0; removed at logout; world/channel updated on transfer events.
- No persistence: state is fully reconstructable from HP + skill level + effect data, and Cosmic likewise keeps it transient. Service restart recovers state lazily via the next trigger per character (or login).

### FR-5 — Data access

To evaluate FR-1, atlas-buffs needs (all read-only, REST unless already event-carried):

- **Berserk skill level**: atlas-skills (`GET` character's skill 1320006), refreshed on skill `UPDATED` events.
- **Skill effect `x` at level**: the skill-effect data pipeline backed by atlas-data (atlas-buffs does not currently mirror the `data/skill/effect` model — it gains a read client/mirror per the established pattern used by atlas-channel/atlas-character/atlas-messages).
- **Effective (buff-inclusive) max HP**: atlas-effective-stats REST.
- **Current HP**: from the `STAT_CHANGED` event body where carried (`Values` map), else atlas-character REST as fallback. Design phase confirms what the event carries for HP updates.
- **Character level** (for the effect packet body): from atlas-character (event or REST).

All lookups are tenant-scoped via `tenant.MustFromContext(ctx)`; failures are logged and skip the re-evaluation (never crash the consumer or leave a stale schedule broadcasting wrong state — on lookup failure the existing schedule keeps its last-known state, matching "recompute on next trigger" semantics).

## 5. API Surface

No new REST endpoints.

Kafka changes:

- **New**: a berserk status signal from atlas-buffs on its existing buff status topic (or a sibling event type), carrying at minimum: transaction id, world id, channel id, character id, character level, skill level, berserk-active bool. Exact event name/shape and whether it fires per state change or per 3s tick is decided in design (FR-3). Emission follows the `message.Buffer` / `message.Emit` producer pattern.
- **New consumers in atlas-buffs**: character status events (`LOGIN`, `LOGOUT`, `STAT_CHANGED`, `CHANNEL_CHANGED`/`MAP_CHANGED` as needed) on `EVENT_TOPIC_CHARACTER_STATUS`, and skill status events (`UPDATED`) on the atlas-skills status topic. atlas-buffs currently consumes only its own buff command topic (`services/atlas-buffs/.../kafka/consumer/character/consumer.go`), so these are new consumer registrations following the curried `InitConsumers(l)(cmf)(groupId)` pattern.
- **New handler in atlas-channel**: on the buffs-emitted event, write `CharacterEffectWriter` to the character's session and `CharacterEffectForeignWriter` to other sessions in the map (template: `handleStatusEventLevelChanged` + `ForOtherSessionsInMap`, `services/atlas-channel/.../kafka/consumer/character/consumer.go:437-457`; existing buff consumer at `kafka/consumer/buff/consumer.go` is the wiring model).

No packet registry, writer, or tenant-template changes: `CharacterEffectWriter`/`CharacterEffectForeignWriter` and the `SKILL_USE` operations-table mode are already wired for every tenant version (the encoder resolves the mode via `WithResolvedCode("operations", "SKILL_USE")`).

## 6. Data Model

No database changes. All state is the in-memory registry described in FR-4 (tenant-scoped keys, no cross-tenant leakage). No new shared-lib types are expected; if design finds a need for one, `libs/atlas-constants` is checked first per DOM-21 before defining anything new.

## 7. Service Impact

| Service | Change |
|---|---|
| **atlas-buffs** (primary) | New berserk domain package: registry, re-evaluation logic, 5s/3s scheduler (following the existing `tasks/` ticker pattern, cf. `tasks/poison.go`); new consumers for character + skill status events; new status-event producer; new read clients for skills, effective-stats, character, and skill-effect data. |
| **atlas-channel** | New consumer handler translating the berserk event into own + foreign `EffectSkillUse` packets. No writer/packet work. |
| **atlas-skills** | None (existing `UPDATED` status events and REST reads suffice). |
| **atlas-effective-stats** | None (existing REST read). |
| **atlas-character** | None (existing status events and REST reads suffice). |
| **atlas-data** | None (effect `x`/`berserk` fields already parsed). |
| **libs** | None expected (packet + constants already complete). |

Deployment: no new env topics beyond what the chosen event shape requires; if a new topic env var is introduced, it must be added to the affected services' k8s manifests (both base yaml and any env overlays) — flagged here because missing live-config wiring has bitten new-opcode/new-topic work before.

## 8. Non-Functional Requirements

- **Multi-tenancy**: registry keys and all lookups tenant-scoped; consumers parse tenant headers per the standard pattern.
- **Overhead**: tracking is limited to logged-in characters with Berserk level > 0 (a small subset). Steady-state cost is one schedule per tracked character firing every 3s; with the buffs-emits-per-tick design that is one small Kafka event per tick per Dark Knight — acceptable at expected populations, but the design phase should confirm the tick-vs-state-change split with this cost in mind.
- **Resilience**: consumer handlers never panic on missing data; failed lookups log and skip (FR-5). Ticker goroutines follow the project's safe-goroutine conventions.
- **Ordering**: re-evaluations for one character must not interleave destructively (cancel-then-reschedule must be atomic under the registry lock).
- **Observability**: state transitions (on↔off, track/untrack) logged at debug/info with character id and tenant.

## 9. Open Questions

1. **GM-hide branch** (user-acknowledged follow-up): Cosmic broadcasts only to GMs when the Dark Knight is hidden (`Character.java:1861-1865`). Atlas has no hide concept yet; when it grows one, the foreign broadcast here must honor it. Record as a follow-up item when hide lands — do not block this task.
2. **Tick emission split** (design phase): buffs emits per-3s-tick events vs. buffs emits state changes + channel owns the 3s re-broadcast timer. Client-visible behavior is fixed by FR-3.
3. **STAT_CHANGED payload** (design phase): confirm whether the `Values` map carries the new HP value for HP updates, or a character REST read is needed per re-evaluation.
4. **v83 WZ `x` values** (design phase): verify skill 1320006's per-level `x` from local WZ data per the grounding rule (the formula and field are Cosmic-verified; the values are data-resolved at runtime regardless).

## 10. Acceptance Criteria

- [ ] A Dark Knight with Berserk leveled who drops below `x`% of effective max HP starts receiving the self skill-use effect with the berserk flag **on**, within one broadcast tick; other players in the map receive the foreign variant.
- [ ] Healing to/above the threshold flips the broadcast flag **off** (boundary: `hp*100/effectiveMaxHp == x` is **inactive** — strict less-than).
- [ ] Hyper Body applying/expiring re-evaluates the state (max HP change alone can toggle berserk with HP constant).
- [ ] Allocating SP into Berserk (level 0→1 while below the new threshold) starts tracking without requiring a relog; level changes re-resolve `x`.
- [ ] Login below threshold restores the active state; logout stops all broadcasts for that character; channel transfer keeps broadcasts routed to the correct channel/map.
- [ ] A player entering the map of an active-berserk Dark Knight sees the aura within one broadcast period (≤3s) without any HP event occurring.
- [ ] Characters without the skill (level 0) generate no registry entries, no tickers, and no events.
- [ ] Broadcast cadence matches Cosmic: 5s initial delay, 3s period, schedule replaced on every re-evaluation.
- [ ] Threshold, skill id, and mode byte are all data/constant-resolved: no numeric skill-id literals outside `atlas-constants`, no hard-coded `x` values, no hard-coded mode bytes.
- [ ] Death stops the broadcast; revive re-establishes correct state on the revive HP change.
- [ ] All state and lookups are tenant-scoped; two tenants' Dark Knights never cross-talk.
- [ ] `go test -race ./...`, `go vet ./...`, `go build ./...` clean in atlas-buffs and atlas-channel; `docker buildx bake atlas-buffs atlas-channel` clean; `tools/redis-key-guard.sh` clean.
- [ ] Tests use the project Builder pattern for setup (no `*_testhelpers.go` constructors); registry/scheduler logic covered including the strict-`<` boundary and cancel-reschedule race.
