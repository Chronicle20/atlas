# Homing Beacon / Bullseye (Outlaw 5211006, Corsair 5220011) — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-07-10
---

## 1. Overview

Homing Beacon (Outlaw, skill 5211006) and Bullseye (Corsair, skill 5220011) are pirate
lock-on skills. The player fires the skill at a monster as a ranged attack; on hit, the
server grants the attacker a `HOMING_BEACON` temporary stat whose value is the struck
monster's object id. The client uses that stat to home subsequent shots onto the locked
target. The lock persists until the player changes maps or re-casts the skill on a
different monster — it has no natural duration.

Atlas currently does nothing at the point where these casts arrive: the ranged-attack
pipeline processes the damage normally, but the buff application is an unimplemented
TODO (`services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go:413`,
`// TODO Homing Beacon / Bullseye`). The skill ids and the `HOMING_BEACON` temporary-stat
type already exist in shared libraries, and the packet library already models the
GuidedBullet wire block — but three concrete gaps (listed in §4) prevent the feature
from working end to end.

Reference behavior is Cosmic (verified from source, not memory):
- `src/main/java/net/server/channel/handlers/AbstractDealDamageHandler.java:346-348` —
  when `attack.skill` is `Outlaw.HOMING_BEACON` or `Corsair.BULLSEYE`, apply the beacon
  buff with the damaged monster's object id.
- `src/main/java/server/StatEffect.java:1254-1260` (`applyBeaconBuff`) — the buff is
  registered with `Long.MAX_VALUE` expiry (never expires on its own) and sent only to
  the casting player.
- `src/main/java/net/server/channel/handlers/PlayerMapTransitionHandler.java:46-52` —
  on map-transition complete, the beacon buff is canceled and a clearing stat update
  (value 0) is sent to the client.
- `src/main/java/server/StatEffect.java:676-679` — the statup list carries
  `BuffStat.HOMING_BEACON` (single stat, no other statups).

## 2. Goals

Primary goals:
- Applying Homing Beacon / Bullseye via a ranged attack grants the attacker a
  `HOMING_BEACON` buff carrying the struck monster's object id.
- The buff is explicitly non-expiring: atlas-buffs gains first-class, explicit
  no-expiry buff support (owner decision — no magic large durations, no sentinel abuse
  hidden behind a huge finite number).
- The buff is canceled on map change via the existing `MAP_CHANGED` character status
  event, and replaced when the skill is re-cast on another monster.
- The `HOMING_BEACON` stat reaches the client correctly (GuidedBullet block carrying
  `dwMobId`) on **all supported tenant versions** — v83, v84, v87, JMS, and v95. v95
  requires new IDA verification work (§4 gap 4).
- All client-interpreted values (skill effect data, costs, wire bytes) are verified
  from WZ data / atlas-data / IDA during design — nothing assumed from general
  MapleStory knowledge (owner decision).

Non-goals:
- Server-side cancel when the locked target dies. Cosmic does not do this; the client
  handles the visual drop. Parity is kept deliberately (owner decision: "leave it,
  we'll see if that causes bugs").
- Broadcasting the beacon to other players. Cosmic sends the buff only to the caster
  (`applyBeaconBuff` uses `applyto.sendPacket`, never a map broadcast); `HOMING_BEACON`
  is skipped by the foreign CTS encode path in atlas-packet.
- Server-side homing/aim logic. Homing is client behavior driven by the stat.
- Any of the other TODOs at the same handler site (Flame Thrower, Snow Charge,
  Hamstring, etc.).

## 3. User Stories

- As an Outlaw, I want casting Homing Beacon at a monster to lock my shots onto it so
  that my subsequent attacks home in, matching classic v83 behavior.
- As a Corsair, I want Bullseye to behave identically (higher-level variant of the same
  mechanic).
- As a player, I want the lock to clear when I change maps so that a stale lock does
  not auto-flag monsters on the new map (the exact bug Cosmic's map-transition cancel
  exists to prevent — see the comment at `StatEffect.java:1254`).
- As a player, I want re-casting the skill on a different monster to move the lock to
  the new target.
- As an operator, I want the buff never to be reaped by the expiration ticker, and I
  want no-expiry to be an explicit, auditable concept in atlas-buffs rather than a
  disguised large duration.

## 4. Current State and Gaps

### Already in place (verified)

| What | Where |
|---|---|
| Skill ids `OutlawHomingBeaconId = 5211006`, `CorsairBullseyeId = 5220011` | `libs/atlas-constants/skill/constants.go:3225,3236` |
| `TemporaryStatTypeHomingBeacon = "HOMING_BEACON"` | `libs/atlas-constants/character/temporary_stat.go:121` |
| `GuidedBulletTemporaryStat` 17-byte wire block ending in `dwMobId uint32` | `libs/atlas-packet/model/character_temporary_stat.go:473-506` |
| Two-state base-stat group includes the HomingBeacon/GuidedBullet slot for v83/v84/v87/JMS | `libs/atlas-packet/model/character_temporary_stat.go:717-731` (`twoStateBaseStats`) |
| atlas-buffs command consumers: APPLY, CANCEL, CANCEL_ALL, CANCEL_BY_TYPES | `services/atlas-buffs/atlas.com/buffs/kafka/consumer/character/consumer.go:30-39` |
| `MAP_CHANGED` character status event (emitted by atlas-maps, consumed by many services) | `services/atlas-maps/atlas.com/maps/kafka/message/character/kafka.go:16,50` |
| Channel-side buff apply/cancel processor: `Apply(f, fromId, sourceId, level, duration, statups)`, `Cancel(f, characterId, sourceId)` | `services/atlas-channel/atlas.com/channel/character/buff/processor.go:45,52` |
| BuffGive writers build a CTS via `AddStat(type, sourceId, amount, level, expiresAt)` | `services/atlas-channel/atlas.com/channel/socket/writer/character_buff_give.go` |

### Gaps this task must close

1. **Attack-handler hook missing.** Nothing at
   `character_attack_common.go:413` applies the buff when `ai.SkillId()` is 5211006 or
   5220011.
2. **CTS encoder never emits an active beacon.** `getBaseTemporaryStats` always appends
   an empty `NewGuidedBulletTemporaryStat()` (`dwMobId = 0`) regardless of stat-map
   contents (`libs/atlas-packet/model/character_temporary_stat.go:843-844`), so a mob
   id can never reach the wire today.
3. **No non-expiring buffs.** `NewBuff` rejects `duration <= 0`
   (`services/atlas-buffs/atlas.com/buffs/buff/model.go:99`) and the expiration task
   reaps strictly by `expiresAt` (`services/atlas-buffs/atlas.com/buffs/tasks/expiration.go`).
4. **v95 GuidedBullet slot unverified.** For GMS ≥ 95, `twoStateBaseStats` truncates
   the group after MonsterRiding because the PartyBooster/GuidedBullet base-stat wire
   sizes were never IDA-verified (comment at
   `libs/atlas-packet/model/character_temporary_stat.go:707-715`, "Task 41b"). v95 is
   in scope (owner decision), so this verification and encoder extension is part of
   this task.
5. **No buff-cancel-on-map-change hook exists anywhere.**

## 5. Functional Requirements

### FR-1: Beacon buff application on attack

- FR-1.1: When the ranged-attack pipeline in atlas-channel processes an attack whose
  skill id is `OutlawHomingBeaconId` (5211006) or `CorsairBullseyeId` (5220011), and at
  least one monster was struck, the handler MUST apply a buff to the **attacker** with:
  - `sourceId` = the cast skill id,
  - a single statup of type `HOMING_BEACON`,
  - statup amount = the struck monster's object id (Atlas monster unique id),
  - level = the character's skill level,
  - explicit no-expiry duration (FR-2).
- FR-1.2: The buff MUST be applied through the existing channel → atlas-buffs Kafka
  command flow (`buff.Processor` / APPLY command), not by writing packets directly from
  the attack handler.
- FR-1.3: If the attack's damage entries reference multiple monsters, the design phase
  MUST determine the correct target selection against Cosmic (Cosmic applies per
  damaged monster inside the per-monster loop, so the last-processed monster wins) and
  document the chosen rule. The skill's WZ data (`mobCount`) must be checked rather
  than assumed to be 1.
- FR-1.4: Applying the buff while a beacon from either skill is already active MUST
  replace the previous lock (re-cast on a new monster moves the lock). If the two skill
  ids produce distinct buff identities in atlas-buffs, the design must ensure the old
  one is canceled — a Corsair with SP in both skills must never hold two beacons.
- FR-1.5: A beacon attack that strikes no monster (all-miss / no damage entries) MUST
  NOT apply or move the lock. (Cosmic's hook sits inside the per-damaged-monster loop,
  so a whiff changes nothing.)
- FR-1.6: The attack itself (damage, projectile/bullet consumption, MP cost) flows
  through the existing attack pipeline unchanged. No special-case cost handling.
  Effect values consumed by this feature (duration source, `x` value, mob count, costs)
  MUST be read from atlas-data / WZ during design — not assumed (owner decision).

### FR-2: Explicit no-expiry buff support in atlas-buffs

- FR-2.1: atlas-buffs MUST gain a first-class, explicit representation of a
  non-expiring buff (owner decision: "should be explicit"). The exact shape (e.g. a
  `noExpiry` flag on the model + REST/Kafka contract, with `duration`/`expiresAt`
  semantics defined for that case) is a design-phase decision, but a bare magic number
  (e.g. `duration = MaxInt32`) is explicitly rejected.
- FR-2.2: `NewBuff` validation MUST accept the no-expiry form while continuing to
  reject nonsensical finite durations (`<= 0`).
- FR-2.3: The expiration task MUST never reap a no-expiry buff.
- FR-2.4: `Expired()` MUST return false for a no-expiry buff.
- FR-2.5: The REST projection of buffs (`buff/rest.go`) and the channel-side mirror
  model MUST represent no-expiry buffs coherently (a reader must be able to tell the
  buff does not expire; the channel writer must not encode a bogus remaining-time).
  Client-visible time encoding for the beacon stat is defined by the packet contract
  (FR-4), verified per version.
- FR-2.6: Existing finite-duration buff behavior MUST be unchanged (regression tests).

### FR-3: Cancel semantics

- FR-3.1: atlas-buffs MUST consume the existing `MAP_CHANGED` character status event
  and cancel any active `HOMING_BEACON` stat for that character (CancelByTypes flow),
  mirroring Cosmic's `PlayerMapTransitionHandler`. This is the service-boundary-correct
  placement (owner decision: skill semantics live in atlas-buffs, not channel).
- FR-3.2: The cancel MUST result in the client's beacon state being cleared. Cosmic
  sends a clearing stat update (`HOMING_BEACON` value 0 — `PlayerMapTransitionHandler.java:50-51`);
  the design MUST verify whether Atlas's existing buff-cancel packet path
  (mask-based BuffCancel) clears the v83 client's lock, or whether a value-0 give is
  required, and implement whichever the client actually honors (IDA/live verification,
  not assumption).
- FR-3.3: The lock is NOT canceled when the target monster dies (owner decision —
  Cosmic parity; revisit only if it causes observable bugs).
- FR-3.4: Existing whole-character cancel paths (death/respawn CancelAll, logout) MUST
  also clear the beacon — expected to fall out of the generic flows, but must be
  covered by a test rather than assumed.

### FR-4: Wire encoding (all versions in scope)

- FR-4.1: When a character's temporary stats include an active `HOMING_BEACON`, the
  CTS encoder MUST emit the GuidedBullet two-state block populated with the locked
  monster's object id (`dwMobId`) instead of the current unconditional empty block, for
  every tenant version whose client reads that block.
- FR-4.2: The BuffGive packet for the beacon apply MUST be byte-verified against the
  v83 client's `OnTemporaryStatSet` read order via IDA (and against each other
  supported version's client), including how the two-state mask bit and block interact
  with a per-stat HOMING_BEACON mask bit. Cosmic's `giveBuff` treats HOMING_BEACON as
  "special" with trailing pad bytes (`PacketCreator.java:2842-2856`) — the Atlas
  encoding must come from IDA, not from Cosmic's packet shape.
- FR-4.3: **v95:** the truncated two-state group in `twoStateBaseStats` must be
  extended: IDA-verify the v95 PartyBooster and GuidedBullet base-stat wire blocks
  (the "Task 41b" gap), then emit the full v95 two-state group so the beacon works on
  v95 tenants. If IDA verification concludes the v95 client's group genuinely differs,
  the verified truth wins and is documented with decompilation evidence.
- FR-4.4: Character re-entry mid-lock (the `SetField` full-CTS encode) must also carry
  the beacon correctly — relevant within the same map (e.g. cash shop round-trip is a
  map change and cancels; but any flow that re-sends full CTS without a map change must
  not drop or corrupt the stat). Design phase enumerates which flows re-send full CTS.
- FR-4.5: Foreign CTS encode continues to skip `HOMING_BEACON` (already the case —
  `baseStatNames` skip in `DecodeForeign`/foreign encode). No foreign beacon packet is
  introduced.

### FR-5: Multi-skill correctness

- FR-5.1: Both skill ids MUST be handled identically apart from their effect values
  (levels, costs — read from WZ). Job gating (Outlaw vs Corsair) is the client's and
  skill-assignment's concern; the handler keys on skill id only, consistent with the
  existing pipeline.

## 6. API Surface

No new REST endpoints. Changes are to existing contracts:

- **atlas-buffs APPLY command / buff REST model:** extended with the explicit
  no-expiry representation (FR-2). Exact field shape decided in design; must remain
  backward compatible with existing producers (atlas-channel skill-use path,
  atlas-consumables, sagas) that always send finite durations.
- **atlas-buffs new consumer:** `MAP_CHANGED` character status event (existing topic,
  existing event schema — new consumer group registration in atlas-buffs).
- **No new Kafka topics.**
- Buff status events emitted by atlas-buffs (consumed by channel to write BuffGive /
  BuffCancel packets) carry the statup `amount` = mob object id; the existing
  `stat.Model{statType, amount int32}` shape suffices (monster object ids fit int32 in
  practice; design confirms the id-generation range in atlas-monsters and documents the
  cast).

## 7. Data Model

- atlas-buffs `buff.Model`: add the explicit no-expiry concept (field + JSON
  marshalling + validation). Storage is the existing per-character registry — no
  database migration.
- No new entities, no schema changes in any Postgres-backed service.
- Multi-tenancy: unchanged — buffs are already tenant-scoped via context; the new
  consumer parses tenant headers like every other atlas-buffs consumer.

## 8. Service Impact

| Service / lib | Change |
|---|---|
| `services/atlas-channel` | Attack-handler hook at the `character_attack_common.go` TODO site: detect skill ids 5211006/5220011, resolve struck monster object id, apply no-expiry HOMING_BEACON buff via existing buff processor. Channel-side buff mirror model updated for no-expiry. |
| `services/atlas-buffs` | Explicit no-expiry support (model, validation, expiration task, REST). New `MAP_CHANGED` consumer issuing CancelByTypes(HOMING_BEACON). Mock updates if the processor interface changes. |
| `libs/atlas-packet` | `getBaseTemporaryStats` emits populated GuidedBullet block from an active HOMING_BEACON stat; v95 two-state group extension after IDA verification; byte-fixture tests per version. |
| `libs/atlas-constants` | No changes expected (both skill ids and the stat type exist). |

Dockerfile / go.work: no new libs, so no COPY-line or go.work changes expected. The
standard verification matrix still applies (see §10).

## 9. Non-Functional Requirements

- **Performance:** the hook adds at most one Kafka command per beacon cast — no
  per-damage-line work beyond reading the already-parsed damage entries.
- **Observability:** buff apply/cancel paths keep existing logging; the map-change
  cancel logs at debug level like sibling consumers.
- **Multi-tenancy / versioning:** all wire-level behavior is version-gated exactly as
  the packet lib already gates the two-state group; no client-interpreted byte may be
  hard-coded where a tenant-resolved table exists (DOM-25).
- **Honesty of verification:** every client-facing byte claim in design.md must carry
  IDA or WZ evidence per `docs/packets/audits/VERIFYING_A_PACKET.md` norms. A
  mode/mask enumeration without byte fixtures is not verification.

## 10. Open Questions

1. Exact shape of the explicit no-expiry contract (flag vs. documented sentinel at the
   API boundary with an internal flag) — design phase, with a survey of existing
   producers so none breaks.
2. Whether the v83 client clears the lock on mask-based BuffCancel alone or requires
   the value-0 give Cosmic sends (FR-3.2) — IDA/live verification.
3. v95 GuidedBullet/PartyBooster base-stat block layout (FR-4.3) — IDA verification
   against the v95 IDB.
4. Target-selection rule when a beacon cast reports multiple damaged monsters (FR-1.3)
   — WZ `mobCount` + Cosmic loop semantics.
5. Whether the buff clears the client's UI icon state correctly given the beacon has no
   duration bar (client rendering detail, verified live on v83).

## 11. Acceptance Criteria

- [ ] Casting Homing Beacon (5211006) as an Outlaw on a v83 tenant strikes a monster
      and the attacker receives a HOMING_BEACON buff whose amount is that monster's
      object id; subsequent shots visibly home in on the live client.
- [ ] Casting Bullseye (5220011) as a Corsair behaves identically.
- [ ] Re-casting on a different monster moves the lock; the old lock is gone (single
      active beacon per character at all times, across both skill ids).
- [ ] A beacon cast that hits nothing leaves the current lock state unchanged.
- [ ] Changing maps cancels the buff server-side and clears the lock on the client;
      returning to the previous map does not resurrect it.
- [ ] The locked target dying does NOT cancel the buff server-side (parity behavior
      documented).
- [ ] atlas-buffs no-expiry buffs: never reaped by the expiration ticker, `Expired()`
      false, explicit in the model/contract, finite-duration behavior regression-tested.
- [ ] Death/respawn and logout clear the beacon via existing cancel-all flows (test
      evidence, not assumption).
- [ ] Byte-fixture tests cover the BuffGive encode with an active beacon (populated
      GuidedBullet block) for v83, v84, v87, JMS; v95 covered after its IDA
      verification lands, with decompilation evidence recorded.
- [ ] All effect values used (mob count, costs, any duration/x reads) cite WZ /
      atlas-data sources in design.md — zero values taken from memory.
- [ ] `go test -race ./...`, `go vet ./...`, `go build ./...` clean in every changed
      module; `docker buildx bake` for atlas-channel and atlas-buffs; and
      `tools/redis-key-guard.sh` clean from repo root.
