# Corsair Battleship (5221006) — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-07-10
---

## 1. Overview

The Corsair Battleship (skill 5221006) is the 4th-job Corsair skill-mount. Casting it
puts the character on a battleship vehicle; the ship has its own HP pool separate from
the character's. While riding, damage the character takes also drains the ship's HP.
When ship HP reaches zero the ship "breaks": the character is dismounted and the skill
goes on cooldown. Uniquely among cooldown skills, the cooldown is applied **when the
ship breaks, not when the skill is cast** — a rider whose ship never breaks can dismount
and remount freely.

Atlas deliberately excluded Battleship from the task-086 mount system
(`libs/atlas-constants/skill/mount_test.go:16` marks 5221006 out of scope;
`services/atlas-channel/atlas.com/channel/socket/handler/character_damage.go:31` carries
`// TODO decrease battleship hp`). This task closes that gap: mounting, the HP pool,
break/cooldown semantics, client HP-gauge reporting, and server-side gating of the two
battleship-dependent attack skills — Cannon (5221007) and Torpedo (5221008).

The reference behavior is Cosmic (v83-era), verified directly against Cosmic source and
WZ data (§4.1). The WZ skill data carries **no ship-HP field**, so the HP formula is
Cosmic-parity by decision (interview Q2).

## 2. Goals

Primary goals:
- Battleship (5221006) is a working rideable skill-mount on all provisioned tenant versions.
- Ship HP pool: initialized on mount from the Cosmic formula, drained by damage taken, reported to the client.
- Break semantics: HP ≤ 0 → dismount + skill cooldown (duration from skill-effect `cooltime`).
- Cooldown carve-out: no cooldown on cast; no cooldown on manual dismount; cooldown only on break.
- Server-side gating of Cannon (5221007) and Torpedo (5221008): rejected when the caster is not riding the battleship (exploit prevention).

Non-goals:
- Mount progression/tiredness (atlas-mounts territory; battleship has none).
- Tamed-mount or other skill-only-mount changes (task-086 landed those).
- Persisting ship HP across rides, logins, or channel changes (interview Q3: reset).
- Big-Bang-era battleship behavior changes.
- New visual packet families — battleship renders through the existing MONSTER_RIDING plumbing.

## 3. User Stories

- As a Corsair, I want to cast Battleship and ride the ship so that I can use Cannon and Torpedo.
- As a Corsair, I want the ship to absorb a parallel HP pool and show its remaining HP so that I know when it is about to break.
- As a Corsair, I want the skill to go on cooldown only when the ship breaks so that I can freely dismount and remount an intact ship.
- As a server operator, I want Cannon/Torpedo casts rejected when the player is not on the battleship so that packet-editing clients cannot use them on foot.

## 4. Functional Requirements

### 4.1 Verified reference behavior (Cosmic + WZ)

All values below were verified on 2026-07-10 against the local Cosmic checkout and its
v83-era WZ data. Cosmic file paths are relative to the Cosmic repo `src/main/java/`.

| Fact | Source |
|------|--------|
| Ship HP formula: `400 × skillLevel + max(characterLevel − 120, 0) × 200` | Cosmic `client/Character.java` `resetBattleshipHp()` |
| Rider takes the damage **and** the ship drains by the same amount (parallel pool, not an absorber) | Cosmic `net/server/channel/handlers/TakeDamageHandler.java:275-278` |
| On HP ≤ 0: send cooldown packet for 5221006, register server cooldown (duration = effect `cooltime`), cancel MONSTER_RIDING (dismount) | Cosmic `client/Character.java` `decreaseBattleshipHp()` |
| On drain without break: report remaining ship HP to the client via the skill-cooldown packet with pseudo-skill id **5221999** (client renders it as the ship HP gauge) | Cosmic `client/Character.java` `announceBattleshipHp()` |
| Cast-time cooldown application explicitly skips BATTLE_SHIP (the carve-out) | Cosmic `net/server/channel/handlers/SpecialMoveHandler.java:86` |
| Vehicle id is item **1932000** (BATTLESHIP); buff is MONSTER_RIDING | Cosmic `server/StatEffect.java:1294`, `constants/id/ItemId.java:376` |
| WZ 5221006: 10 levels; `cooltime=90` at **every** level; buff duration `time=2100000` ms (35 min); per-level `mpCon`/`pdd`/`mdd`; **no ship-HP field** | Cosmic `wz/Skill.wz/522.img.xml` |
| Riding check = active MONSTER_RIDING buff sourced from 5221006 | Cosmic `client/Character.java` `isRidingBattleship()` |

Known divergence (accepted, interview Q3): Cosmic persists damaged ship HP across
remounts and even relogs (it smuggles the value through the cooldown table under id
5221999). Atlas instead **resets** — ship HP exists only for the lifetime of one active
ride and every fresh mount starts at full formula HP. Consequence: a rider can manually
dismount/remount to "repair" the ship. Accepted as a simplification; the break cooldown
still prevents remounting a broken ship for 90s.

### 4.2 Mounting (cast path)

- FR-2.1: Casting 5221006 applies the MONSTER_RIDING buff with vehicle id 1932000,
  through the existing skill-only mount cast path
  (`services/atlas-channel/atlas.com/channel/skill/handler/mount.go`). Battleship must be
  classified as a valid mount skill in `libs/atlas-constants/skill/mount.go` (today it is
  excluded; `mount_test.go` expectations flip from out-of-scope to in-scope).
- FR-2.2: On mount, ship HP is initialized to `400 × skillLevel + max(characterLevel − 120, 0) × 200`
  using the caster's current level in 5221006 and character level. Always full — never a
  carried-over value.
- FR-2.3: No cooldown is applied on cast (the carve-out). The generic "apply `cooltime`
  on skill use" path, if any exists or is added later, must exempt 5221006.
- FR-2.4: Casting 5221006 while it is on cooldown (post-break) is rejected server-side —
  the client greys the icon, but the server must not trust that.
- FR-2.5: Toggle behavior matches the other skill-only mounts: casting while already
  riding the battleship dismounts (existing mount.go dismount arm), which ends the ride
  and clears ship HP state (FR-4).

### 4.3 HP pool and damage drain

- FR-3.1: In the character-damage handler
  (`services/atlas-channel/atlas.com/channel/socket/handler/character_damage.go`), when
  the damaged character is riding the battleship (active MONSTER_RIDING sourced from
  5221006), decrement ship HP by the damage amount. The character's own HP change is
  unaffected (parallel pool — the existing `ChangeHP` stays as-is).
- FR-3.2: Ship HP state lives in Redis, accessed exclusively through `libs/atlas-redis`
  types (redis-key-guard clean), keyed by tenant + character id. Decrement must be atomic
  (e.g. DECRBY), since damage handling must not lose concurrent drains.
- FR-3.3: If a rider has an active battleship ride but no Redis entry (state lost —
  e.g. Redis restart, or a transition that cleared it), lazily re-initialize to full
  formula HP on first access. Missing state is never an error and never a stuck ship.
- FR-3.4: After a drain that does not break the ship, report remaining ship HP to that
  client via the skill-cooldown packet carrying pseudo-skill id 5221999 and the remaining
  HP as the "cooldown" value (the client's ship HP gauge — §4.1). This is a direct
  session announce in the damage handler, reusing the existing skill-cooldown writer
  (`libs/atlas-packet/character/clientbound/skill_cooldown.go`).

### 4.4 Break

- FR-4.1: When ship HP reaches ≤ 0: cancel the MONSTER_RIDING buff (dismount, with the
  normal foreign-buff cancel broadcast so other players see the ship disappear), apply a
  cooldown to 5221006 with duration = the caster's skill-effect `cooltime` (90s at every
  level in v83 data, but read from effect data, not hardcoded), and delete the ship HP
  state.
- FR-4.2: The cooldown is applied through the existing atlas-skills cooldown subsystem
  (`SetCooldownAndEmit` — Redis-backed registry + `cooldown applied` status event), so
  the existing atlas-channel consumer
  (`kafka/consumer/skill/consumer.go` `handleCooldownApplied`) announces the client
  cooldown packet with no new packet work.
- FR-4.3: Break is the **only** trigger for the 5221006 cooldown. Manual dismount, buff
  expiry (35 min), logout, and map/channel transitions never apply it.

### 4.5 Ride-end cleanup (reset semantics)

- FR-5.1: Whenever the battleship ride ends — break, manual dismount/toggle, buff
  cancel, buff expiry, or logout — the ship HP state for that character is deleted.
- FR-5.2: The Redis entry carries a TTL of the buff duration (35 min from effect data)
  as a safety net so orphaned entries self-expire even if a cleanup path is missed.
- FR-5.3: Combined with FR-3.3 (lazy re-init), the invariant is: ship HP state exists
  only while a ride is active, and any observation of "riding but no state" means full HP.

### 4.6 Cannon / Torpedo gating

- FR-6.1: Attack processing for skills 5221007 (Battleship Cannon) and 5221008
  (Battleship Torpedo) validates server-side that the caster currently has the active
  battleship MONSTER_RIDING buff. If not riding, the attack is rejected (no damage
  applied, no broadcast) and logged at debug — matching how a packet-editing client is
  otherwise able to use these skills on foot. (Cosmic does not gate this; Atlas does, per
  interview Q4 — this is an intentional hardening beyond parity.)
- FR-6.2: Gating must not add latency-sensitive external calls to the attack hot path;
  the riding check reads the same state the damage handler uses (design decides the
  exact source: session/buff registry vs. temp-stat lookup).

### 4.7 Version scope

- FR-7.1: The feature ships for **all currently provisioned tenant versions** (GMS v83,
  v84, v87, v92, v95, JMS v185). The server logic is version-independent; per-version
  work is limited to verifying the client-interpreted values (§8 / §9).
- FR-7.2: The pseudo-skill id 5221999 and vehicle item id 1932000 are client-interpreted
  wire values. Per DOM-25, they must be config-resolved from tenant configuration, not
  hardcoded in service code — "version-stable" does not exempt them. Design determines
  the exact table (writer options / operations-style lookup) and backfills live tenant
  configs (new template values do not reach existing tenants automatically).

## 5. API Surface

No new public REST endpoints and no new Kafka topics are expected.

- Reused: atlas-skills cooldown application (existing command/API surface +
  `cooldown applied` status event → existing channel consumer → existing
  `skill_cooldown` clientbound writer).
- Reused: buff cancel path (atlas-buffs) for the break dismount, including the foreign
  cancel broadcast.
- New (internal only): ship HP registry operations inside atlas-channel (Redis via
  `libs/atlas-redis`). Not exposed over REST.
- Clientbound: the existing skill-cooldown packet is sent with (a) pseudo-id 5221999 +
  remaining ship HP on drain, and (b) 5221006 + `cooltime` on break. No new packet
  structures.

Error cases:
- Cast while cooling → rejected, no state change.
- Cannon/Torpedo while not riding → attack rejected, no state change.
- Damage while riding with missing HP state → lazy re-init then drain (never an error).

## 6. Data Model

No PostgreSQL changes. One new Redis structure (illustrative; final key shape follows
`libs/atlas-redis` conventions and its KeyPrefix tenant isolation):

| Key | Value | TTL | Lifecycle |
|-----|-------|-----|-----------|
| `battleship:hp:{tenantId}:{characterId}` | remaining ship HP (int) | buff duration (35 min) | created on mount (FR-2.2) or lazily (FR-3.3); atomically decremented on drain; deleted on any ride end |

Multi-tenancy: tenant id is part of the key (and the standard KeyPrefix isolation
applies). No cross-tenant reads.

## 7. Service Impact

| Service / lib | Change |
|---------------|--------|
| `libs/atlas-constants` | Classify 5221006 as a skill-only mount with vehicle 1932000 (extend `skill/mount.go`); flip `mount_test.go` out-of-scope expectations. |
| `services/atlas-channel` | The bulk: mount cast path HP init + cooldown-cast rejection; `character_damage.go` drain/gauge/break (replaces the TODO at line 31); ship-HP Redis registry; Cannon/Torpedo gating in the attack path; ride-end cleanup hooks (dismount, buff cancel/expiry, logout). |
| `services/atlas-skills` | Expected no code change — consumed via existing `SetCooldownAndEmit` surface. Verify at design time that atlas-channel can reach it (command topic or REST) for a server-initiated cooldown; add the thin missing piece if not. |
| `services/atlas-buffs` | Expected no code change — existing cancel path performs the dismount. |
| `services/atlas-data` | Verify only: `skill/reader.go:469` already includes `CorsairBattleshipId` in the vehicle-id-emission band; confirm the emitted MONSTER_RIDING statup amount for battleship is 1932000 (or fix if it emits something else). |
| Tenant configuration / seed templates | Config-resolved wire values for 5221999 / 1932000 (FR-7.2), per-version tables + live-tenant backfill. |

## 8. Non-Functional Requirements

- **Multi-tenancy:** all state tenant-scoped (Redis key + KeyPrefix); wire values
  config-resolved per tenant version (DOM-25).
- **Redis discipline:** all Redis access through `libs/atlas-redis`;
  `tools/redis-key-guard.sh` clean.
- **Concurrency:** HP drain is atomic; break must fire exactly once per depletion (a
  double-break must not double-apply the cooldown or double-cancel the buff).
- **Hot path:** damage and attack handlers gain at most one Redis round-trip; no new
  synchronous cross-service REST calls in either path.
- **Observability:** drain, break, gating rejections, and lazy re-init logged at debug
  with character/tenant fields, consistent with surrounding handlers.
- **State home caveat (interview Q1):** Redis was chosen as "to start" — the registry
  must be encapsulated behind a processor interface so a future move (e.g. into a
  service-owned store) doesn't ripple through the handlers.

## 9. Open Questions

1. **Per-version client verification of the 5221999 gauge.** Cosmic targets v83; before
   trusting the gauge on v84/v87/v92/v95/jms, verify in each IDA database that the
   client's skill-cooldown handler treats 5221999 as the battleship gauge. If a version
   does not, the gauge is cosmetic there (break logic is unaffected) — but this must be
   verified, not assumed, at design time.
2. **Server-initiated cooldown transport.** Confirm the existing atlas-skills surface
   atlas-channel should call for FR-4.2 (Kafka command vs. REST) and whether a command
   topic already exists.
3. **Riding-state source for gating/drain.** Session temp-stat tracking vs. buff
   registry lookup — decide the cheapest reliable source at design time (FR-6.2).
4. **atlas-data vehicle emission.** Confirm what `reader.go:469` actually injects into
   the MONSTER_RIDING statup for 5221006 (the comment suggests the skill id band emits
   something other than a fixed item id).

## 10. Acceptance Criteria

- [ ] Casting 5221006 mounts the character on vehicle 1932000 (MONSTER_RIDING), visible to self and foreign observers, on every provisioned tenant version.
- [ ] No cooldown is applied on cast; casting while the post-break cooldown is active is rejected server-side.
- [ ] Ship HP initializes to `400 × skillLevel + max(charLevel − 120, 0) × 200` on every mount (fresh full pool each ride).
- [ ] Damage taken while riding drains ship HP by the damage amount AND still applies to character HP; the client receives the 5221999 gauge update on each non-breaking drain.
- [ ] Depleting ship HP dismounts the character (foreign broadcast included), applies the 5221006 cooldown with duration read from effect `cooltime`, and clears ship state — exactly once, even under concurrent damage.
- [ ] Manual dismount, buff expiry, logout: no cooldown, ship state cleared; next mount starts at full HP.
- [ ] Cannon (5221007) and Torpedo (5221008) attacks are rejected when the caster is not riding the battleship; work normally when riding.
- [ ] 5221999 / 1932000 wire values are config-resolved per tenant (no hardcoded literals in service code); live tenant configs backfilled.
- [ ] `mount_test.go` battleship expectations updated; new unit tests cover formula, drain, break-once, reset, and gating.
- [ ] `go test -race ./...`, `go vet ./...`, `go build ./...` clean in every changed module; `docker buildx bake` clean for every touched service; `tools/redis-key-guard.sh` clean.
