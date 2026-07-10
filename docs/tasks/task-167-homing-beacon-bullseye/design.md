# Homing Beacon / Bullseye (task-167) — Design

Version: v1
Status: Ready for planning
PRD: `docs/tasks/task-167-homing-beacon-bullseye/prd.md`

---

## 1. Summary

Casting Homing Beacon (5211006) or Bullseye (5220011) as a ranged attack that strikes a
monster grants the attacker a non-expiring `HOMING_BEACON` buff whose statup amount is the
struck monster's object id. The client homes subsequent shots onto that monster. The lock
is replaced on re-cast and canceled on map change; it never expires on its own.

Four subsystems change:

| Where | What |
|---|---|
| `services/atlas-channel` | Attack-handler hook (cancel-then-apply), beacon mirror registry, local-give merge, foreign-announcement suppression for beacon-only events, no-expiry mirror support |
| `services/atlas-buffs` | First-class no-expiry buffs; new `MAP_CHANGED` consumer issuing CancelByTypes(HOMING_BEACON) |
| `libs/atlas-packet` | Populated GuidedBullet block from the active stat; accurate cancel masks (new); conditional movement byte on cancel (new); v95 two-state group extension (Task 41b closed) |
| Contracts | `noExpiry` field on buff APPLY command, buff status events, buff REST model |

Two of those items — accurate cancel masks and the local-give merge — were not in the PRD.
IDA verification during design (§3) proved the feature cannot work without them: under the
current encoder, any unrelated buff give or cancel silently destroys the client's lock.

## 2. Verified inputs (WZ / Cosmic / IDA)

Per the PRD's honesty requirement, every client-facing claim below carries its source.
IDA instances: v83 = `MapleStory_dump.exe` (v83_Me IDB, port 13342), v95 =
`GMS_v95.0_U_DEVM.exe` (port 13341).

### 2.1 WZ effect data (source: Cosmic v83 `wz/Skill.wz/521.img.xml`, `522.img.xml`)

- **5211006 Homing Beacon**: 30 levels. Per level: `mpCon` 20 (lv1-10) / 25 (lv11-20) / 30
  (lv21-30), `damage` 148→380, `range` 350. **No `time` node, no `mobCount` node, no `x`
  node.** Action `homing`, has `ball` node (projectile).
- **5220011 Bullseye**: 20 levels. Per level: `mpCon` 35 (lv1-10) / 40 (lv11-20), `damage`
  390→580, `x` = level (1..20), `range` 350. `masterLevel` 10, `req` 5211006 level 30.
  **No `time`, no `mobCount`.**

Consequences: no natural duration exists (no-expiry is data-correct, not just Cosmic
parity); `mobCount` absent means single-target (mobCount=1 default) — the multi-monster
rule (§5.2) is a defensive rule, not an expected path. Bullseye's `x` is not consumed by
this design (Cosmic seeds the statup with `x` and then overwrites it with the object id;
we skip the intermediate step).

### 2.2 Cosmic reference behavior (read from source, not memory)

- `AbstractDealDamageHandler.java:346-348` — hook inside the per-damaged-monster loop:
  `applyBeaconBuff(player, monster.getObjectId())`. Whiff → loop body never runs.
- `StatEffect.java:1254-1260` — `applyBeaconBuff` registers the effect with
  `Long.MAX_VALUE` expiry and sends the packet only to the caster.
- `PlayerMapTransitionHandler.java:46-52` — on transition complete: cancel the buff stat
  and send a value-0 give (`giveBuff(1, beaconid, {HOMING_BEACON: 0})`).
- `PacketCreator.java:2837-2856` — Cosmic's `giveBuff` special-cases
  `HOMING_BEACON`/`MONSTER_RIDING` with 3 trailing pad bytes. Per PRD FR-4.2 we do NOT
  copy this shape; the canonical encoding comes from IDA (§2.3) and the existing
  fixture-verified atlas-packet model.

### 2.3 IDA — v83 client (all addresses in `MapleStory_dump.exe`, v83_Me)

- **GuidedBullet mask bit**: constant at `0xBF5528` = 16 bytes whose low qword (LE) is
  `0x0080000000000000` — matches Cosmic `BuffStat.HOMING_BEACON = 0x80000000000000L`.
- **Wire block (17 bytes)**: the GuidedBullet stat object's `DecodeForClient` is the
  unnamed `sub_77C442`: calls
  `TwoStateTemporaryStat<long,not_equal<long,0>,NoExpire,...>::DecodeForClient` (@`0x79407D`,
  which via `TemporaryStatBase<long>::DecodeForClient` @`0x793EF2` reads nValue int32 +
  rValue int32 + DecodeTime [byte + int32]) then `Decode4 → this[9] = dwMobId`. Total
  4+4+5+4 = **17 bytes — exactly `GuidedBulletTemporaryStat`'s existing layout**
  (`libs/atlas-packet/model/character_temporary_stat.go:473-505`).
- **Set path** — `CWvsContext::OnTemporaryStatSet` @`0xA202BE`: after
  `SecondaryStat::DecodeForLocal`, if the decoded flag contains the GuidedBullet bit AND
  the stored stat is activated (vtable `IsActivated`, i.e. `nValue != 0`), it looks up
  `CMobPool::GetMob(stored dwMobId)` and calls `CMob::SetGuided(mob, rValue, 0)`
  (@`0x671121`). **Constraints: nValue must be nonzero; rValue is passed to SetGuided as
  the skill/reason; dwMobId selects the mob.**
- **Reset path** — `CWvsContext::OnTemporaryStatReset` @`0xA2071F`: decodes the 16-byte
  mask; if it contains the GuidedBullet bit and a guided stat is active, clears the mob's
  guided entry (`sub_671166` via CMobPool), then `SecondaryStat::Reset(mask)` and
  `CTemporaryStatView::ResetTemporary(mask)` (icon removal). **A mask-based
  TemporaryStatReset fully clears the lock — no value-0 give is needed** (Cosmic's value-0
  give is an alternative mechanism through the set path; we use the reset path Atlas
  already has). This answers PRD Open Question 2.
- **Movement byte** — both set and reset handlers read one trailing byte only when
  `sub_77DC78(mask)` is true. Filter (@`0x77DC78`):
  Speed | Jump | Stun | Weakness | Slow | Morph | Ghost | BasicStatUp | Attract |
  RideVehicle (`0x0020…`) | `0x0010…` | `0x0008…` (the two Dash bits). **GuidedBullet is
  NOT movement-affecting.**

### 2.4 IDA — v95 client (`GMS_v95.0_U_DEVM.exe`) — closes the "Task 41b" gap (FR-4.3)

- **Two-state mask bits** (from `CTS_*` dynamic initializers): EnergyCharged = bit 122,
  Dash_Speed = 123, Dash_Jump = 124, RideVehicle = 125, **PartyBooster = 126,
  GuidedBullet = 127**.
- **Group membership and block sizes** (from `SecondaryStat::SecondaryStat` @`0x72F190`,
  which builds `aTemporaryStat[7]`, and each variant's `DecodeForClient`):

  | Slot | Stat | Template variant | DecodeForClient | Block size |
  |---|---|---|---|---|
  | 0 | EnergyCharged | `greater_equal<10000>`, `Expire<BaseOnLastUpdatedTime,DynamicTermSet>`, `Decrease<200,10000>` | `0x72C740` | base 13 + expireTerm 2 = **15** |
  | 1 | Dash_Speed | `not_equal<0>`, `Expire<BaseOnLastUpdatedTime,DynamicTermSet>` | `0x726BA0` | **15** |
  | 2 | Dash_Jump | same as slot 1 | `0x726BA0` | **15** |
  | 3 | RideVehicle | `not_equal<0>`, `NoExpire` | `0x726AB0` | base only = **13** |
  | 4 | PartyBooster | `TemporaryStat_PartyBooster` over `not_equal<0>`, `Expire<BaseOnCurrentTime,DynamicTermSet>` | `0x72C600` | base 13 + tCurrentTime 5 + expireTerm 2 = **20** |
  | 5 | GuidedBullet | `TemporaryStat_GuidedBullet` (`NoExpire` + `m_dwMobID`) | `0x727180` | base 13 + dwMobId 4 = **17** |
  | 6 | (unnamed) | `not_equal<0>`, `Expire<BaseOnLastUpdatedTime,DynamicTermSet>` | — | 15, **no CTS mask constant exists → unreachable on the wire** (this is the "Undead overflows the mask" slot the lib's comment predicted) |

  `DecodeTime` (@`0x725430`) = Decode1 + Decode4 = 5 bytes, same as v83.
- **The v95 two-state trailer read is mask-gated per member**: `DecodeForLocal`
  (@`0x7350E0`), tail loop at `0x73DBA0-0x73DBF2`, builds `UINT128(1) << shift`, tests it
  against the decoded flag, and only on a hit virtual-calls that member's
  `DecodeForClient`. This is why the lib's current truncated 4-block v95 encode (which
  sets only those 4 bits) is byte-consistent today — the existing fixture total
  `16+2+58` matches slots 0-3 exactly (15+15+15+13 = 58).
- **Set path** — `CWvsContext::OnTemporaryStatSet` @`0xA02FC0`: explicit
  `flag & CTS_GuidedBullet && aTemporaryStat[5]->IsActivated()` →
  `CMob::SetGuided(CMobPool::GetMob(GetMobID()), GetReason(), 0)`. Same constraints as
  v83. (`IsActivated` for the NoExpire variant @`0x726A80` is `m_value != 0`.)
- **Reset path** — `CWvsContext::OnTemporaryStatReset` @`0x9F2AB0`: mask-based; on the
  GuidedBullet bit with an active stat calls `CMobPool::ResetGuidedMob(m_reason, mobId)`
  (@`0x6572E0`) then `SecondaryStat::Reset(mask)`. Mask-based cancel clears the lock on
  v95 too.
- **Movement filter** — `SecondaryStat::IsMovementAffectingStat` @`0x7208C0`: Speed, Jump,
  Stun, Weakness, Slow, Morph, Ghost, BasicStatUp, Attract, RideVehicle, Dash_Speed,
  Dash_Jump, Flying, Frozen, YellowAura. GuidedBullet is not in it.
- v95 additionally uses the stat for damage (`ApplyGuidedBulletDamage` @`0x7265E0`) —
  client-side, no server work.

### 2.5 Codebase facts the design builds on (file:line, worktree-relative)

- `NewBuff` rejects `duration <= 0`; `expiresAt = now + duration·ms`; `Expired()` is
  `expiresAt.Before(now)` — `services/atlas-buffs/atlas.com/buffs/buff/model.go:98-114,29-31`.
- Expiration reap: `Registry.GetExpired` uses `Expired()`;
  `ProcessorImpl.ExpireBuffs` emits EXPIRED per reaped buff —
  `services/atlas-buffs/atlas.com/buffs/character/registry.go:184-208`,
  `character/processor.go:130-143`.
- Buffs are stored per character keyed by `sourceId` (or `sourceId:statType` in
  accumulate mode) — `character/model.go:12-24`, `registry.go:46-56`. Two different
  sourceIds → two coexisting buffs; FR-1.4 replace does not fall out for free.
- Kafka: `ApplyCommandBody{FromId, SourceId, Level, Duration, Changes, Accumulate}`;
  status events APPLIED/EXPIRED carry `Duration, CreatedAt, ExpiresAt` —
  `services/atlas-buffs/atlas.com/buffs/kafka/message/character/kafka.go:30-42,73-90`.
- **Every existing APPLY producer sends a finite positive duration** (atlas-channel skill/
  mount/mysticdoor, atlas-consumables, atlas-summons Beholder, atlas-messages `@buff`,
  atlas-maps mist tick, atlas-monsters disease; atlas-saga-orchestrator only produces
  CANCEL_ALL). A `noExpiry` extension is backward-compatible with all of them.
- CancelByTypes flow exists end-to-end (producer shape in atlas-consumables; atlas-buffs
  `CancelByStatTypes` emits one EXPIRED per removed buff; atlas-channel writes
  BuffCancel + foreign) — `consumer.go:81-89`, `processor.go:103-128`,
  atlas-channel `kafka/consumer/buff/consumer.go:93-123`.
- `MAP_CHANGED` character status event: envelope `{TransactionId, WorldId, CharacterId,
  Type}` + body `{ChannelId, OldMapId, OldInstance, TargetMapId, TargetInstance, ...}` —
  `services/atlas-maps/atlas.com/maps/kafka/message/character/kafka.go:11-60`. Consumer
  pattern to mirror: atlas-summons `kafka/consumer/character/{kafka,consumer}.go`.
- Monster object ids: allocated from `libs/atlas-object-id` with `MinId = 1,000,000`,
  `MaxId = 2,147,483,647 (0x7FFFFFFF)` — always nonzero and fits `int32` exactly
  (`allocator.go:30-42`). Damage entries expose `MonsterId() uint32`
  (`libs/atlas-packet/model/damage_info.go:84`).
- Attack handler: TODO site at
  `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go:413`
  has in scope: `ai` (skill id, damage info), `sk skill.Model` (owned skill + level,
  resolved at `:277-290`), `c character.Model`, `s.Field()`, `s.CharacterId()`, monster
  processor `mp`. The MP Eater proc (`:206-264`, wired at `:351-355`) is the established
  skill-conditional side-effect pattern (errors swallowed).
- CTS encoder: `EncodeMask` unconditionally ORs every `twoStateBaseStats(t)` member's bit
  (`libs/atlas-packet/model/character_temporary_stat.go:558-583`); `getBaseTemporaryStats`
  appends an **empty** `NewGuidedBulletTemporaryStat()` regardless of the stat map
  (`:843-844`); MonsterRiding (`:833-840`) is the only member populated from the map —
  the pattern to extend. `BuffCancel` encodes `EncodeMask` + one trailing byte
  (`libs/atlas-packet/character/clientbound/buff_cancel.go:27-35`).
- Only the foreign spawn re-sends a full CTS (`CharacterSpawn.Encode` →
  `cts.EncodeForeign`, `libs/atlas-packet/character/clientbound/spawn.go:75`); there is no
  local full-CTS re-send flow (answers FR-4.4: nothing to protect beyond gives/cancels;
  map entry re-issues nothing locally and the beacon cancels on map change anyway).
  `HOMING_BEACON` is in `baseStatNames` (`:682`) so the foreign per-stat path already
  skips it (FR-4.5).

## 3. Constraints discovered during design (design-driving findings)

These three findings reshape the PRD's plan and are the core of this design:

**F1 — Any buff cancel currently clears every two-state stat on the client.**
`BuffCancel` reuses `EncodeMask`, which always sets all two-state group bits. Both v83
(@`0xA2071F`) and v95 (@`0x9F2AB0`) reset handlers clear *every masked stat*
(`SecondaryStat::Reset(mask)`) and, for GuidedBullet specifically, unflag the guided mob.
So the moment any other buff expires or is dispelled, an active beacon would be destroyed
client-side. **The cancel path must encode an accurate mask** (only the stats actually
canceled). This also fixes the same latent hazard for MonsterRiding/Dash/EnergyCharge on
every version.

**F2 — Any local buff give currently zeroes the client's stored beacon (pre-v95).**
Pre-v95, every local give writes the full 7-block trailer with all group bits set, and the
client's `DecodeForLocal` overwrites the stored GuidedBullet stat from the wire. An
unrelated give while locked (e.g. drinking a speed potion) would therefore overwrite
`nValue/dwMobId` with zeros and kill homing. **While a beacon is active, every local give
must carry the populated beacon stat.** On v95 this problem does not exist if we only set
bit 127 when the beacon is active (the mask-gated reader then never touches slot 5 on
unrelated gives).

**F3 — The v95 group is 6 members with verified sizes, and the read is mask-gated.**
The lib's truncation was correct as far as it went (slots 0-3 match exactly); the
extension is now fully specified: PartyBooster = 20 bytes (note: *not* the same 15-byte
shape as the pre-95 members — it has an extra 5-byte `tCurrentTime`), GuidedBullet = 17
bytes, bits 126/127. Undead genuinely has no v95 wire slot (mask overflow) — the verified
truth per FR-4.3.

## 4. Alternatives considered

### 4.1 No-expiry contract shape (FR-2, PRD Open Question 1)

- **(a) Explicit `noExpiry bool` field through the whole contract (CHOSEN).** Model field
  + dedicated constructor, `omitempty` on the wire so every existing producer/consumer is
  untouched. Readers can tell "does not expire" without interpreting sentinels.
- (b) Sentinel `duration = -1` at the API boundary mapped to an internal flag. Rejected:
  the PRD owner decision explicitly rejects magic values at the boundary; also
  `mist_tick.go`'s existing seconds-vs-ms confusion shows how sentinel semantics rot.
- (c) `expiresAt = nil` (pointer) as the signal. Rejected: turns every `ExpiresAt()`
  caller into a nil-check; a zero `time.Time` already means "immediately expired" in
  `Expired()`, so overloading the zero value invites the exact bug FR-2.4 forbids.

### 4.2 Single-active-beacon semantics (FR-1.4)

- **(a) Channel emits `CANCEL_BY_TYPES(HOMING_BEACON)` followed by `APPLY` on every
  beacon cast (CHOSEN).** Uses two existing commands; idempotent (cancel of nothing is a
  no-op); covers the cross-skill case (5211006 vs 5220011 have different `sourceId` keys
  in the registry, so plain APPLY would let a Corsair hold two beacons); uniform for
  same-skill re-cast (old mob's client-side flag is explicitly cleared before the new
  lock — plain same-key overwrite would leave the old mob flagged until map change).
  Ordering is guaranteed because both commands go to the same topic keyed by character.
- (b) Teach atlas-buffs `Apply` to evict any existing buff sharing a stat type. Rejected:
  a general behavioral change to buff replacement semantics far beyond this feature's
  needs; risks changing stacking behavior for unrelated buffs.
- (c) An APPLY option `replaceTypes []string`. Rejected: new contract surface for
  something two existing commands already express; YAGNI.

### 4.3 Keeping the give path from destroying the lock (F2)

- **(a) Channel-side beacon mirror registry + merge into local gives (CHOSEN).** The
  channel's buff status consumer already sees every APPLIED/EXPIRED event; a small
  tenant-aware in-memory registry (characterId → active beacon statup {sourceId, level,
  mobId}) is maintained from those events, and the local-give handler appends the active
  beacon to the buff list it passes to `CharacterBuffGiveBody`. Zero extra I/O per give,
  idiomatic (channel keeps many such registries), and pre-95 byte streams for non-locked
  characters are unchanged (the always-written GuidedBullet block simply stays empty).
- (b) REST `GetByCharacterId` against atlas-buffs inside every APPLIED handler. Rejected:
  a synchronous REST hop per buff give server-wide to serve a rare stat.
- (c) Make the GuidedBullet block/bit conditional on all versions (then unrelated gives
  never touch it). Rejected: requires proving the pre-v95 trailer read is mask-gated per
  member (not established for v83 — its `DecodeForLocal` is too inlined to settle
  cheaply), and it changes every existing pre-95 give fixture. Option (a) is correct
  under either read semantics.
- (d) atlas-buffs includes the character's full active buff set in every APPLIED event.
  Rejected: contract bloat for all consumers to serve one stat.

### 4.4 v95 group extension shape (F3)

- **(a) Extend `twoStateBaseStats` for GMS≥95 with PartyBooster + GuidedBullet as
  *conditional* members: block written and mask bit set only when the stat is active
  (CHOSEN).** Matches the client's verified mask-gated read; non-beacon v95 traffic stays
  byte-identical to today (regression-safe); beacon gives add bit 127 + one 17-byte
  populated block.
- (b) Always-write all 6 v95 blocks with bits always set (uniform with pre-95). Rejected:
  changes every v95 buff packet at once (large regression surface), and always-setting
  bit 127 would re-introduce F1 on the cancel path if any cancel encoder ever reuses the
  give mask.
- The existing 4 members stay always-written on v95 (status-quo, fixture-locked).
  The asymmetry (4 unconditional + 2 conditional) is deliberate and documented in code.

## 5. Design

### 5.1 End-to-end flows

**Cast → lock**
1. `processAttack` (atlas-channel) sees `ai.SkillId() ∈ {5211006, 5220011}` on the ranged
   path and a valid struck monster (§5.2).
2. Channel emits, in order, on the buff command topic (same key = character):
   `CANCEL_BY_TYPES{Types:["HOMING_BEACON"]}` then
   `APPLY{SourceId: skillId, Level: sk.Level(), NoExpiry: true, Duration: 0,
   Changes: [{Type:"HOMING_BEACON", Amount: int32(monsterObjectId)}]}`.
3. atlas-buffs cancels any prior beacon (emits EXPIRED if one existed), stores the new
   no-expiry buff, emits APPLIED.
4. Channel consumer: on EXPIRED-with-HOMING_BEACON → updates beacon registry (clear),
   sends BuffCancel with accurate mask (beacon bit only) → client `ResetGuidedMob`.
   On APPLIED-with-HOMING_BEACON → updates beacon registry (set), sends local BuffGive;
   the CTS stat map contains HOMING_BEACON, so the GuidedBullet block is populated
   (nOption = mobId ≠ 0, rOption = skillId, dwMobId = mobId) → client `SetGuided`.
   No foreign give is sent for beacon-only events (§5.5).

**Unrelated buff give while locked** — the APPLIED handler merges the registry's beacon
statup into the CTS → pre-95 trailer carries the populated block again → client re-applies
the same lock (idempotent `SetGuided`). On v95 the merge also sets bit 127 and emits the
populated block; equally idempotent. (Merging on v95 is not strictly required by F2 but
keeping one code path for all versions is simpler and harmless.)

**Unrelated buff cancel while locked** — accurate cancel mask no longer contains the
beacon bit → client keeps the lock. (Fixes F1.)

**Map change** — atlas-maps emits `MAP_CHANGED`; new atlas-buffs consumer calls
`CancelByStatTypes(worldId, characterId, ["HOMING_BEACON"])`; EXPIRED flows as above; the
client's lock and icon clear via the mask-based reset (IDA-verified both versions).
Death/respawn and logout keep flowing through the existing CANCEL_ALL paths, which remove
all buffs regardless of expiry (FR-3.4 covered by tests, not new code).

**Target death** — nothing (owner decision; Cosmic parity).

### 5.2 Attack-handler hook (atlas-channel)

At the `character_attack_common.go` TODO site (after the damage loop; `ai`, `sk`, `mp`,
`s` in scope), mirroring the MP Eater "errors swallowed, pipeline unaffected" pattern:

- Gate: ranged path already reached; `ai.SkillId()` equals
  `skill.OutlawHomingBeaconId` or `skill.CorsairBullseyeId`
  (`libs/atlas-constants/skill/constants.go:3225,3236`).
- Target selection (FR-1.3): iterate `ai.DamageInfo()`; candidate = the **last** entry
  whose `MonsterId() != 0` and for which `mp.GetById` confirms the monster exists in the
  field registry (Cosmic's `monster != null` inside the loop; last-wins matches Cosmic's
  loop order). WZ says mobCount is absent (=1) so in practice there is at most one entry;
  the rule exists for malformed/multi-entry packets.
- Whiff (FR-1.5): no valid candidate → return without emitting anything; the prior lock
  is untouched.
- Emit `CancelByTypes` then `Apply` as in §5.1. Failures are logged and swallowed —
  damage, projectile consumption, and broadcasts are already done and must not be
  affected (FR-1.6: no changes to cost/projectile handling; costs come from the client's
  normal skill-use flow and the WZ `mpCon` values above are informational).
- The channel `character/buff` package gains a `CancelByTypes` provider/processor method
  (shape copied from atlas-consumables' producer) and a no-expiry-capable apply (§5.3).

### 5.3 No-expiry buffs (atlas-buffs + contracts)

**Domain model** (`buff/model.go`):
- Add private field `noExpiry bool`, accessor `NoExpiry() bool`.
- New constructor `NewNoExpiryBuff(sourceId int32, level byte, changes []stat.Model)
  (Model, error)` — validates non-empty changes; sets `duration = 0`, `createdAt = now`,
  `expiresAt = time.Time{}` (zero), `noExpiry = true`. `NewBuff` keeps rejecting
  `duration <= 0` (FR-2.2).
- `Expired()` returns `false` when `noExpiry` is set, before any time comparison
  (FR-2.4). The zero `expiresAt` is therefore never consulted.
- JSON marshal/unmarshal round-trips `noExpiry` (Redis-persisted registry); absent field
  unmarshals to `false`, so previously stored buffs are unaffected.

**Registry / processor**: `Registry.Apply` and `Processor.Apply` gain a `noExpiry bool`
parameter (no parallel method — one code path; the four consumer handlers and the fan-out
funcs are the only callers, and there is no cross-service mock to update). `GetExpired` needs no change beyond
`Expired()` returning false (FR-2.3) — plus an explicit regression test that a no-expiry
buff survives a reap pass while an expired finite buff beside it is removed.

**Kafka contract** (`kafka/message/character/kafka.go`, mirrored in every producer's local
copy that needs it — only atlas-channel's):
- `ApplyCommandBody` += `NoExpiry bool \`json:"noExpiry,omitempty"\``. `Duration` MUST be
  0 when `NoExpiry` is true (validated in the consumer handler; a nonzero duration with
  the flag is rejected as a malformed command and logged).
- `AppliedStatusEventBody` and `ExpiredStatusEventBody` += `NoExpiry bool
  \`json:"noExpiry,omitempty"\``; `ExpiresAt` is the zero time for no-expiry buffs.

**REST** (`buff/rest.go`): `RestModel` += `NoExpiry bool \`json:"noExpiry"\``; `ExpiresAt`
zero-valued for no-expiry (FR-2.5 — a reader can tell the buff does not expire).

**atlas-channel mirror** (`character/buff/model.go`): += `noExpiry` field populated from
the status event; its `Expired()` respects the flag. The channel writer never encodes a
bogus remaining time for the beacon because `HOMING_BEACON` is in `baseStatNames` and thus
never hits the per-stat relative-ms encode; the GuidedBullet block ignores `expiresAt`
entirely (NoExpire type client-side — §2.3/2.4).

### 5.4 MAP_CHANGED consumer (atlas-buffs)

New consumer package mirroring atlas-summons' character-status consumer: local
re-declaration of `EVENT_TOPIC_CHARACTER_STATUS`, `MAP_CHANGED` type, envelope
`{TransactionId, WorldId, CharacterId, Type}` and a `MapChangedBody` subset (only fields
needed for logging). Handler guards on type and calls
`Processor.CancelByStatTypes(worldId, characterId, ["HOMING_BEACON"])` — the existing
method (`processor.go:103-128`); debug-level logging like sibling consumers. Registration
follows the `InitConsumers(l)(cmf)(groupId)` / `InitHandlers` idiom with tenant + span
header parsers. Deployment: atlas-buffs gains the `EVENT_TOPIC_CHARACTER_STATUS` env var
in its kustomize base (same literal name every other consumer of this topic uses).

Only `HOMING_BEACON` is canceled on map change. No other stat gains map-change semantics
in this task.

### 5.5 Packet library changes (libs/atlas-packet)

1. **Populated GuidedBullet block (FR-4.1).** `getBaseTemporaryStats`'s
   `twoStateGuidedBullet` case follows the MonsterRiding pattern: if
   `m.stats[HOMING_BEACON]` exists → `NewGuidedBulletTemporaryStatWithOptions(value =
   s.Value(), reason = s.SourceId(), dwMobId = uint32(s.Value()))`; else the existing
   empty block. Field mapping (IDA constraints, §2.3/2.4): `nOption` = monster object id
   (guaranteed nonzero by the allocator range — satisfies `IsActivated`), `rOption` =
   skill id (client passes it to `SetGuided` as the reason and uses it for the icon),
   `dwMobId` = monster object id. The true GMS server's `nOption` content is unknowable
   from the client (only `!= 0` is checked); using the object id is our documented
   choice. `decodeBaseTemporaryStats` mirrors the read so round-trip tests hold.
2. **Accurate cancel masks (F1).** New `EncodeCancelMask` on `CharacterTemporaryStat`
   that ORs **only** `m.stats` (no unconditional two-state group bits);
   `BuffCancel`/`BuffCancelForeign` switch to it. Give-path `EncodeMask` is untouched.
3. **Conditional movement byte on cancel (consequence of #2).** The trailing byte the
   cancel writer emits (`buff_cancel.go:32`, currently mislabeled `tSwallowBuffTime`) is
   in truth the movement byte the client reads only when
   `IsMovementAffectingStat(mask)` — today's always-set RideVehicle/Dash bits made it
   unconditionally required, which is why the unconditional write worked. With accurate
   masks the writer must emit it **iff** the canceled mask intersects the
   movement-affecting filter. The filter is a version-gated table in the lib seeded from
   IDA: v83 and v95 lists in §2.3/2.4; v84/v87/JMS lists to be extracted from their IDBs
   during execution (expected identical to v83's; verified, not assumed, per the
   packet-audit norms). The give path keeps its existing unconditional byte because give
   masks always contain RideVehicle/Dash group bits pre-95 and on v95 (4 always-set
   members) — always movement-affecting, byte always read.
4. **v95 group extension (F3, FR-4.3).** `twoStateKind` gains `twoStatePartyBooster`
   (base + tCurrentTime + usExpireTerm = 20B). `twoStateBaseStats` for GMS≥95 returns the
   verified 6-member group with PartyBooster and GuidedBullet marked **conditional**:
   `getBaseTemporaryStats` emits a conditional member's block only when its stat is
   active; `EncodeMask` ORs unconditional members always (status quo for slots 0-3) while
   conditional members' bits arrive via the active-stats loop. Registry alignment: on
   v95 the group base must place EnergyCharge at bit 122 so HomingBeacon (base+5) lands
   on 127 and PartyBooster (base+4) on 126 — pinned by byte fixtures. PartyBooster gets
   no producer in this task; its entry completes the verified group (Task 41b) and its
   encode path is exercised by lib round-trip tests only.
5. **Foreign path (FR-4.5).** `HOMING_BEACON` stays skipped in foreign per-stat encode
   (already in `baseStatNames`). Additionally `EncodeForeign`'s mask and conditional
   trailer must exclude the beacon on v95 (never set bit 127, never write the block —
   the remote-user reader's handling of it is unverified and the stat is caster-only).
   Pre-95 foreign encodes keep their status-quo always-written empty GuidedBullet block —
   byte-identical to today. The channel additionally suppresses foreign
   BuffGive/BuffCancel announcements when an event's changes contain only
   `HOMING_BEACON` (nothing to show other players; avoids exercising unverified remote
   read paths).

### 5.6 Contract summary

| Contract | Change | Compat |
|---|---|---|
| `COMMAND_TOPIC_CHARACTER_BUFF` APPLY | `noExpiry` bool, omitempty; Duration must be 0 when set | Existing producers unaffected |
| `EVENT_TOPIC_CHARACTER_BUFF_STATUS` APPLIED/EXPIRED | `noExpiry` bool, omitempty; zero `expiresAt` when set | Existing consumers (channel) tolerate absent field |
| atlas-buffs REST buffs | `noExpiry` on RestModel | Additive |
| `EVENT_TOPIC_CHARACTER_STATUS` | no schema change; new consumer group in atlas-buffs | — |
| Statup amount | monster object id as `int32` | Allocator max = MaxInt32; no truncation possible (§2.5) |

No new topics, no DB migrations, no new libs (no Dockerfile/go.work changes).

### 5.7 Error handling

- Attack hook: every failure (monster lookup, producer emit) logs and returns; the attack
  pipeline result is never altered.
- atlas-buffs APPLY with `NoExpiry && Duration != 0`: rejected + warn log (malformed).
- MAP_CHANGED handler: cancel errors log-and-continue (next map change or logout/death
  cancel-all is the safety net).
- Channel beacon registry is process-local; on channel restart it repopulates only from
  subsequent events. Consequence: after a channel restart, an unrelated give to a
  still-locked character could send an empty GuidedBullet block (pre-95) and drop the
  client's lock visual — degraded, not broken (re-cast restores; map change was going to
  clear anyway). Accepted; noted in code. (The alternative — REST backfill on miss — is
  available later without contract changes.)

### 5.8 Multi-tenancy / versioning

All wire behavior is version-gated inside atlas-packet via the tenant model exactly like
the existing two-state handling; no client-interpreted byte is hard-coded in a service
(DOM-25: the beacon writer resolves through the existing writer/CTS registry paths; no
new tenant-config tables are needed because the CTS layout is not one of the
config-resolved operations tables). The new movement-filter table is version-gated in the
lib alongside `twoStateBaseStats`.

## 6. Testing strategy

**atlas-buffs (unit)** — no-expiry: `NewNoExpiryBuff` validation; `Expired()` false; JSON
round-trip incl. absent-field default; reap pass keeps no-expiry while removing expired
finite buffs; finite-duration behavior regression (existing tests must stay green);
CancelByTypes removes a no-expiry beacon; CancelAll removes it (FR-3.4 evidence);
MAP_CHANGED handler guards type and calls CancelByStatTypes.

**atlas-channel (unit)** — target selection: single hit, whiff (no emit), multi-entry
last-valid-wins, monster-not-found (no emit); command emission order
(CancelByTypes before Apply); beacon registry set/clear from APPLIED/EXPIRED; local-give
merge adds the beacon statup; foreign suppression for beacon-only events; mirror model
`Expired()` with flag.

**libs/atlas-packet (byte fixtures, per version)** — following the existing
`TestCTSMonsterRiding*` shape (`character_temporary_stat_test.go:169-246`):
- v83/v84/v87/JMS: BuffGive with active beacon — mask bit present, populated 17-byte
  GuidedBullet block (nOption=mobId, rOption=skillId, dwMobId), all other trailer bytes
  unchanged from the current fixtures; give with NO beacon — byte-identical to today.
- v95: give with active beacon — bit 127 set, trailer = 58 status-quo bytes + 17-byte
  populated GuidedBullet block; give without — byte-identical to today (16+2+58);
  PartyBooster-active round-trip (20-byte block, bit 126).
- Cancel: beacon-only cancel mask contains exactly the beacon bit and no movement byte;
  a Speed/Jump-stat cancel carries the movement byte; a mount cancel carries the riding
  bit; regression: no cancel mask ever contains inactive two-state bits again.
- Round-trip (`pt.RoundTrip`) for the populated GuidedBullet and PartyBooster decodes.
- Evidence records per the packet-audit norms for the newly pinned v95 facts (§2.4) and
  the v84/v87/JMS movement filters once extracted.

**Live verification (acceptance, v83 tenant)** — cast → shots home; re-cast other
monster → lock moves; whiff → unchanged; unrelated buff gained/expired mid-lock → lock
survives (F1/F2 proof); map change → lock and icon clear; locked target killed → no
server cancel; icon rendering with no duration bar (PRD Open Question 5 — only
observable live).

**Standard gates** — `go test -race`, `go vet`, `go build` per changed module;
`docker buildx bake atlas-channel atlas-buffs`; `tools/redis-key-guard.sh`.

## 7. Resolved PRD open questions

1. No-expiry shape → explicit `noExpiry` flag end-to-end (§4.1, §5.3).
2. Mask-based cancel vs value-0 give → mask-based reset fully clears (IDA v83 `0xA2071F`,
   v95 `0x9F2AB0`); no value-0 give needed (§2.3).
3. v95 layout → verified 6-member group, sizes 15/15/15/13/20/17, bits 122-127, mask-gated
   reads; Undead has no v95 slot (§2.4).
4. Multi-monster rule → WZ has no `mobCount` (single-target); defensive last-valid-entry
   rule, Cosmic loop parity (§5.2).
5. Icon behavior with no duration bar → client renders via `CTemporaryStatView` with the
   skill id from rOption; final confirmation is a live-test acceptance item (§6).

## 8. Out of scope

- The other TODOs at the attack-handler site (Flame Thrower, Snow Charge, Hamstring, …).
- Server-side cancel on target death (owner decision; revisit only on observed bugs).
- PartyBooster gameplay semantics (only its verified wire slot is added).
- The pre-existing seconds-vs-ms duration mismatch in atlas-maps/atlas-monsters disease
  producers (documented in §2.5 survey; untouched by this task).
- Backfilling the channel beacon registry after restart (§5.7 accepted degradation).
