# task-167 Homing Beacon / Bullseye — Execution Context

Companion to `plan.md`. Everything here is verified against source (file:line) or IDA (design.md §2); nothing is from memory.

## What this task builds

Outlaw 5211006 / Corsair 5220011 lock-on: ranged attack strike → attacker gets a non-expiring `HOMING_BEACON` buff whose statup amount is the struck monster's object id → the client homes subsequent shots. Lock replaced on re-cast (cancel-then-apply), canceled on map change, never reaped by the expiration ticker.

## Key decisions (owner-ratified in design.md)

1. **Explicit `noExpiry` flag end-to-end** (model, Kafka APPLY + APPLIED/EXPIRED, REST, channel mirror). Magic sentinel durations rejected. `NewBuff` still rejects `duration <= 0`; a new `NewNoExpiryBuff` constructor owns the no-expiry case. `Expired()` short-circuits on the flag.
2. **Single active beacon via CANCEL_BY_TYPES then APPLY** from the attack handler — two existing commands, same character-keyed topic → ordered. Covers the cross-skill case (the two skill ids are distinct registry keys, so plain APPLY would let a Corsair hold two beacons).
3. **Map-change cancel lives in atlas-buffs** (new consumer of the existing `EVENT_TOPIC_CHARACTER_STATUS` / `MAP_CHANGED`), not in channel.
4. **Mask-based cancel clears the client lock** — no Cosmic-style value-0 give needed (IDA v83 reset `0xA2071F`, v95 `0x9F2AB0`). PRD Open Question 2 resolved.
5. **No cancel on target death** (Cosmic parity; revisit only on observed bugs).
6. **F1 (design §3): cancel packets must carry accurate masks.** Today `BuffCancel` reuses `EncodeMask`, whose always-set two-state bits make ANY cancel destroy an active mount/dash/beacon client-side. New `CancelMask`/`EncodeCancelMask` OR only active stats; the trailing byte becomes conditional on a version-gated movement filter (`MovementAffectingMask`).
7. **F2: pre-95 local gives overwrite the stored beacon** (full 7-block trailer, unconditional read). Channel keeps a process-local `BeaconMirror` (tenant → characterId → {sourceId, level, mobId}) fed by APPLIED/EXPIRED events and merges the beacon statup into every local give while locked. Accepted degradation: mirror empties on channel restart (design §5.7).
8. **v95 group verified (closes "Task 41b")**: 6 members, blocks 15/15/15/13/20/17, bits 122–127, trailer read mask-gated per member (IDA `0x73DBA0`). PartyBooster(126) and GuidedBullet(127) become *conditional* members — bit+block only when active — so non-beacon v95 traffic stays byte-identical. Undead has no v95 wire slot (bit 128 overflows the mask). PartyBooster's 20-byte block reuses `SpeedInfusionTemporaryStat` (identical wire shape).
9. **Foreign path**: `HOMING_BEACON` already skipped per-stat (`baseStatNames`); channel additionally suppresses foreign BuffGive/BuffCancel for beacon-only events. Foreign v95 never sets bit 127 by construction.
10. **Channel processor gets separate `ApplyNoExpiry`/`CancelByTypes` methods** (deliberate deviation from atlas-buffs' one-code-path rule: channel `Apply` has many callers — skill common path, mounts, mystic door — and no mock exists to update).

## Wire facts the encodes must satisfy (IDA, design §2.3–2.4)

- GuidedBullet block (17B): nOption int32 | rOption int32 | time (bool+int32, 5B) | dwMobId uint32.
- Client set path requires `nOption != 0` (IsActivated) → we send nOption = mobId (allocator floor 1,000,000 guarantees nonzero, max 0x7FFFFFFF fits int32 — `libs/atlas-object-id/allocator.go:30-42`); rOption = skill id (SetGuided reason + icon); dwMobId = mobId.
- v83 GuidedBullet mask bit: registry shift 87 → wire dword[1] `0x00800000`. v95: shift 127 → wire dword[0] `0x80000000`.
- Movement filter (trailing-byte gate): v83 `sub_77DC78` = Speed, Jump, Stun, Weakness→`Weaken`, Slow, Morph, Ghost→`GhostMorph`, BasicStatUp→`MapleWarrior`, Attract→`Seduce`, RideVehicle→`MonsterRiding`, DashSpeed, DashJump. v95 (`0x7208C0`) adds Flying, Frozen, YellowAura. GuidedBullet is NOT movement-affecting. v84/v87/JMS lists must be IDA-extracted (plan Task 7), expected identical to v83 — verify, don't assume.

## WZ facts (Cosmic v83 Skill.wz, design §2.1)

- 5211006: 30 levels, mpCon 20/25/30, no `time`, no `mobCount`, no `x`.
- 5220011: 20 levels, mpCon 35/40, `x` = level (unused by us), no `time`/`mobCount`.
- No natural duration and single-target: no-expiry is data-correct; multi-monster rule (last-valid-wins) is defensive only.

## Key files

| Area | File | Notes |
|---|---|---|
| Buff model | `services/atlas-buffs/atlas.com/buffs/buff/model.go` | `NewBuff` rejects `<=0` at :99; `Expired()` :30 |
| Buff registry | `services/atlas-buffs/atlas.com/buffs/character/registry.go` | `Apply` :68 (srcKey/statKey keying), `GetExpired` :184, `CancelByStatTypes` :231 |
| Buff processor | `services/atlas-buffs/atlas.com/buffs/character/processor.go` | 5 event-provider call sites gain `b.NoExpiry()` |
| Buff Kafka msgs | `services/atlas-buffs/atlas.com/buffs/kafka/message/character/kafka.go` | authoritative contract; channel mirrors it |
| Buff cmd consumer | `services/atlas-buffs/atlas.com/buffs/kafka/consumer/character/consumer.go` | `handleApply` gains NoExpiry validation |
| New consumer | `services/atlas-buffs/atlas.com/buffs/kafka/consumer/charstatus/` | mirrors `atlas-summons/.../kafka/consumer/character/` |
| CTS encoder | `libs/atlas-packet/model/character_temporary_stat.go` | registry builder :61; `EncodeMask` :558; `twoStateBaseStats` :716; `getBaseTemporaryStats` :829; GuidedBullet struct :473 |
| Cancel writer | `libs/atlas-packet/character/clientbound/buff_cancel.go` | byte at :32 mislabeled `tSwallowBuffTime` — it's the movement flag |
| Fixtures | `libs/atlas-packet/model/character_temporary_stat_test.go` | `TestCTSMonsterRidingV95MaskAndLayout` pins today's 16+2+58 truncation |
| Attack handler | `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go` | TODO at :413; `damageInfoEntryDeps`/MP Eater = the deps + swallow-errors pattern |
| Channel buff pkg | `services/atlas-channel/atlas.com/channel/character/buff/` | processor :45 `Apply`, producer, mirror model, rest |
| Channel buff consumer | `services/atlas-channel/atlas.com/channel/kafka/consumer/buff/consumer.go` | APPLIED :57 / EXPIRED :93 handlers |
| Channel registry pattern | `services/atlas-channel/atlas.com/channel/monster/status_mirror.go` | sync.Once singleton keyed by `t.Id()` |
| CancelByTypes producer shape | `services/atlas-consumables/atlas.com/consumables/character/buff/producer.go:57-72` | copy shape |

## Corrections to design.md discovered while planning

- **No k8s change needed for the MAP_CHANGED consumer** (design §5.4 said atlas-buffs' base needed the env var): atlas-buffs consumes the shared `atlas-env` ConfigMap via `envFrom` (`deploy/k8s/base/atlas-buffs.yaml`) and `EVENT_TOPIC_CHARACTER_STATUS` is already in it (`deploy/k8s/base/env-configmap.yaml:94`; per-env values via overlay configMapGenerator).
- `character.Model.Buffs()` (atlas-buffs) returns `map[string]buff.Model`, not a slice — tests iterate.
- atlas-buffs already appends v95 PartyBooster(126)/HomingBeacon(127) in the CTS registry builder — only the *encoder group* (`twoStateBaseStats`) was truncated; no registry change needed.

## Dependencies / environment

- IDA-MCP instances: v83 `MapleStory_dump.exe` port 13342, v95 `GMS_v95.0_U_DEVM.exe` port 13341, JMS port 13340 (`*_U_DEVM` build, not SMC); v84/v87 via `list_instances`. Confirm binary identity before reading. Use `func_query` with `name_regex`.
- Known reset-handler addresses (from `buff_cancel_test.go` packet-audit markers): v84 `0xa6bb24`, v87 `0xab7dc1`, JMS `0xb07628` — entry points for the movement-filter extraction.
- Tests: atlas-buffs uses miniredis + `producertest.InstallNoop()`; packet lib uses `pt.Variants` / `pt.RoundTrip` (`libs/atlas-packet/test/context.go`).
- Gates: `go test -race`, `go vet`, `go build` per module; `docker buildx bake atlas-buffs atlas-channel`; `tools/redis-key-guard.sh` (no global `GOWORK=off`).

## Out of scope (do not touch)

Other attack-handler TODOs; target-death cancel; PartyBooster gameplay semantics (wire slot only); the pre-existing seconds-vs-ms disease-duration mismatch; beacon-mirror restart backfill.
