# Meso Explosion — Exploded-Meso Destruction + Damage — Design

Version: v2
Status: Approved-pending-review
Created: 2026-07-10
Updated: 2026-07-26 (v2 — legacy version scope)
PRD: `docs/tasks/task-150-meso-explosion/prd.md`

---

## 0. v2 changelog — legacy matrix columns added to scope

This design was written (v1) before the legacy version bring-up landed on `main`.
The packet coverage matrix now tracks **9** client versions
(`PROCESS.md`): `gms_v48, v61, v72, v79, v83, v84, v87, v95, jms_v185`. v1
verified only `gms_v83/v84/v87/v95` + jms clientbound symmetry. Per owner
decision (2026-07-26) the four new legacy matrix columns `gms_v48/v61/v72/v79`
are **in scope with full IDA verification**; their meso senders were derived in
this v2 pass (§2). `gms_12`/`gms_92` (template-only, no matrix column, no IDB)
stay **unverified — follows the family branch** (§2.4), the same posture v1 gave
`gms_92`. The result (§2.1): the three meso deltas are byte-identical across ALL
nine IDA-verified versions, and every version difference is a base-layout field
already gated in main's codec — so the "thread a flag, no new version gates"
architecture holds unchanged; only the plan's stale `>=83` per-mob-CRC snippet
needed correcting to the current shared `>=61` gate.

## 1. Summary

Meso Explosion (skill 4211006) is sent by the client as a **variant of the CLOSE_RANGE_ATTACK packet**, written by a dedicated sender (`CUserLocal::DoActiveSkill_MesoExplosion`), not by the normal melee sender. The variant differs from the standard melee attack in exactly three places, and those three deltas are **byte-identical across every IDA-verified matrix version** (gms_v48, gms_v61, gms_v72, gms_v79, gms_v83, gms_v84, gms_v87, gms_v95 — see §2):

1. **Per-mob damage-line count**: in each damage entry, the 2-byte `delay` field is *replaced* by a 1-byte damage-line count, followed by that many 4-byte damage values (then the mob CRC as usual).
2. **Trailing exploded-drop list**: after `characterX`/`characterY`, a 1-byte drop count, then per drop a 4-byte drop object id + a 1-byte hit-mask (bitmask of which attacked-mob indices that drop's explosion damaged).
3. **Trailing 2-byte delay**: a final int16 action-delay value closes the packet.

Everything else (dr-block, skillId, crc pair, mask1/mask2, actionType/speed/time, per-mob header fields, charX/charY) follows the standard melee layout for that version, including all existing version gates. The design therefore threads a *variant flag* through the existing `AttackInfo`/`DamageInfo` codec rather than introducing a new packet or new version gates.

Server-side: atlas-channel validates the listed drops against the field's drops (one REST fetch), rejects the whole attack on any failure, applies damage through the existing pipeline, and emits one drop `CONSUME` command per validated drop. atlas-drops is untouched.

## 2. IDA findings (FR-2)

Serverbound sender per version. gms_v83/v84/v87/v95/jms addresses confirmed in the
live IDBs on 2026-07-10 (v1); the four legacy senders confirmed 2026-07-26 (v2).

| Version | Function | Address | Opcode | Packet-encode readable? |
|---|---|---|---|---|
| gms_v48 | `CUserLocal::DoActiveSkill_MesoExplosion` (named) | `0x6ae4d7` | 36 | ✅ full |
| gms_v61 | meso sender `sub_7B8A39` | `0x7b8a39` | 41 | ✅ full |
| gms_v72 | meso sender `sub_875828` | `0x875828` | 43 | ✅ full |
| gms_v79 | meso sender `0x8c22fd` (IDB label `TryDoingMeleeAttack` overload) | `0x8c22fd` | 42 | ✅ full |
| gms_v83 | `CUserLocal::DoActiveSkill_MesoExplosion` | `0x96b3fb` | | ✅ full |
| gms_v84 | meso sender (IDB mislabeled `CUserLocal::TryDoingMeleeAttack`) | `0x9aa379` | | ✅ full |
| gms_v87 | `CUserLocal::DoActiveSkill_MesoExplosion` | `0x9eee04` | | ✅ full |
| gms_v95 | `CUserLocal::DoActiveSkill_MesoExplosion` | `0x942200` | | ✅ full (typed IDB) |
| jms_v185 | meso sender `sub_A3AAB1` | `0xa3aab1` | | ⚠️ encode tail SCY-virtualized (see §2.3) |
| gms_v92 / gms_12 | — no IDB — | — | | ❌ follows family branch, documented unverified (§2.4) |

Dispatch confirmation: the v84 `CUserLocal::DoActiveSkill` dispatcher jumps to the sender on `skillId == 4211006` (`0x9a7398`), and the jms dispatcher (`CUserLocal::DoActiveSkill` @ `0xa35c3f`) calls `sub_A3AAB1` on switch case `4211002 + 4`. The v84 IDB's name on `0x9aa379` is wrong (it is not the standard melee sender, which is separately pinned at `0x989692`); structure and dispatch prove it is the meso sender.

**Legacy dispatch confirmation (v2):** each legacy meso sender is the sole callee
of the `case 4211006` (`0x40413E`) arm of that version's `DoActiveSkill`
dispatcher — v48 `sub_6ABFA4` (a literal `case 4211006:` calling the named
`DoActiveSkill_MesoExplosion`), v61 `sub_7B5977`, v72 `sub_871D35`, v79
`sub_8BE4FE` (the latter three compile the switch as a `sub 0x40413D`/`dec`/`jz`
chain — no immediate `cmp 4211006`, which is why an immediate scan misses it).
Each sender's encode tail was read verbatim and matches the three deltas below;
`xrefs_to` confirms a single caller for each. v79 reuses a `TryDoingMeleeAttack`
overload rather than a distinct function, but it is the sole `case 4211006`
target and encodes the full meso variant (drop list + trailing delay).

### 2.1 Verified write order

Using gms_v83 (`0x96b3fb`) as the base; version gates noted inline, now spanning
the full legacy range (v48↔jms). Standard-melee fields keep their existing,
fixture-verified semantics — **every gate below is a pre-existing gate in main's
`attack_info.go`/`damage_info.go`; the meso variant adds none** (§2.1a).

```
byte   fieldKey
[GMS >= 84]  int32 dr0, int32 dr1
byte   (mobCount << 4) | (nMaxAttackCount & 0xF)      // see §2.2 — low nibble is NOT a hit count
[GMS >= 84]  int32 dr2, int32 dr3
int32  skillId (4211006)
[GMS >= 95]  byte skillLevel (nCombatOrders)
[GMS >= 84]  int32 randomDr, int32 crc32
[GMS >= 72]  int32 skillDataCrc                       // legacyGmsNoSkillDataCrc: absent < 72 (v48/v61)
[GMS >= 79 / JMS]  int32 skillDataCrc2                // legacyGmsSingleCrc: single CRC on v72, two on v79+
       // no keydown int: 4211006 is not a keydown/charge skill
byte   mask1 (client writes 8*bShadowPartner; observed 0 in v83)
mask2 = (left << shift) | attackAction                // legacyGmsByteAction: BYTE (left<<7) on GMS < 79
                                                       //   (v48/61/72); int16 (left<<15) on GMS >= 79 / JMS
[GMS >= 95]  int32 anotherCrc
byte   attackActionType
byte   attackSpeed
int32  attackTime
[GMS >= 95]  int32 0 (battle-mage related; present in the meso sender too, v95 line 729)
per mob (mobCount entries):
    int32  monsterId
    byte   hitAction
    byte   forceAction | (isLeft << 7)
    byte   frameIdx
    byte   calcDamageStatIndex
    int16  hitX, int16 hitY, int16 prevX, int16 prevY
    byte   damageLineCount                             // ← REPLACES the standard int16 delay
    int32  damage × damageLineCount                    // ← variable, NOT the hits nibble
    [GMS >= 61]  int32 mobCrc                          // per-mob CRC: absent on v48 (< 61), present v61+
int16  characterX
int16  characterY
byte   dropCount
per drop (dropCount entries):
    int32  dropId                                      // CDrop field +32
    byte   dropHitMask                                 // bitmask of mob indices this drop damaged
int16  delay                                           // action-delay tail (v83: a2 = action-data time)
```

Evidence excerpts (decompiled write sites):

- v48 `0x6ae4d7` (named `DoActiveSkill_MesoExplosion`, opcode 36): BYTE action `Encode1((v103<<7)|(v112&0x7F))`; per-mob `Encode1(v61[20])` (count) then `Encode4(*v72)` damage loop and **no `Encode4(mobCrc)`** (v48 < 61); tail `Encode1(dropCount)`, loop `Encode4(*(drop+32)); Encode1(v95[4*j])`, then `Encode2(v120)`.
- v61 `0x7b8a39` (opcode 41): BYTE action @`0x7b91a2`; count byte `Encode1(v60[20])` @`0x7b92f7` + damage loop @`0x7b9311`, then per-mob CRC `Encode4(sub_5CF2AF(...))` @`0x7b932b`; drop list @`0x7b9378`/`0x7b9394`/`0x7b93a6`, tail `Encode2(v117)` @`0x7b93b4`.
- v72 `0x875828` (opcode 43): head single skill-data CRC `Encode4` @`0x875fb9` (v72+), BYTE action @`0x875fd9`; count byte @`0x876128` + damage loop @`0x87613a`, per-mob CRC `Encode4(sub_61F8A5(...))` @`0x876155`; drop list @`0x8761a9`/`0x8761c5`/`0x8761d7`, tail `Encode2(v134)` @`0x8761e5`.
- v79 `0x8c22fd` (opcode 42): TWO head CRCs `Encode4` @`0x8c2ab2` + @`0x8c2abb` (v79+ pair), SHORT action `Encode2((v116<<15)|(v65&0x7FFF))` @`0x8c2adc`; count byte @`0x8c2c2a` + damage loop @`0x8c2c3e`, per-mob CRC `Encode4(sub_640131(...))` @`0x8c2c57`; drop list @`0x8c2ca7`/`0x8c2cc3`/`0x8c2cd5`, tail `Encode2(v137)` @`0x8c2ce3`.
- v83 `0x96b3fb`: mob entry `Encode1(&v132, v71[20])` (count byte) then damage loop from offset 24 then `Encode4(CMob::GetCrc(...))`; tail `Encode1(dropListSize)`, loop `Encode4(*(drop + 32)); Encode1(v108[m])`, then `Encode2(a2)`.
- v84 `0x9aa379`: identical variant deltas at lines 515 (`Encode1(v128[20])`), 529/535–536 (drop list), 538 (`Encode2(v151)`), with the v84 dr-block at 425–429 and randomDr/crc32 at 442/444.
- v87 `0x9eee04`: identical (lines 510, 528, 534–535, 537), opcode arg 46.
- v95 `0x942200` (typed): `Encode1(&oPacket, nMaxAttackCount & 0xF | (16 * nMobCount))`, `Encode1(cd->nCombatOrders)`, `Encode4(v209)` (anotherCrc), `Encode4(0)` after `tAttackTime`, count byte at 763, drop list at 791/798–799 (`(*v77)->dwId` + `v231[i]`), tail `Encode2(v208)`.

### 2.1a Base-layout gates compose with the meso deltas — no new gates

The four legacy senders confirm the meso variant introduces the **same three
deltas at the same positions** as v83–v95, and that every version-to-version
difference is a *base-layout* field already gated in main's codec:

| Base-layout field | Boundary | Existing gate | v48 | v61 | v72 | v79+ |
|---|---|---|---|---|---|---|
| per-mob CRC | present ≥ v61 | `damage_info.go` `MajorVersion() >= 61` | — | ✓ | ✓ | ✓ |
| action width | byte < v79 | `legacyGmsByteAction` | byte | byte | byte | short |
| head skill-data CRC | present ≥ v72 | `legacyGmsNoSkillDataCrc` | — | — | 1 | — |
| 2nd head CRC | present ≥ v79 | `legacyGmsSingleCrc` | — | — | — | 2 |

Because the meso flag only ever touches the three delta positions (count byte,
trailing drop list, trailing delay) and leaves the head + per-mob CRC reads on
their existing gated paths, threading the flag through the shared codec yields
the correct bytes for all nine versions with **no new `Region()`/`MajorVersion()`
gate**. In particular v48's absent per-mob CRC falls out for free: the meso
`DamageInfo` mode keeps the CRC read on the shared `>= 61` gate, so v48 (< 61)
skips it exactly as its sender does.

### 2.2 The mask low nibble is unusable for the meso variant

The v95 typed IDB names the mask expression `nMaxAttackCount & 0xF | (16 * nMobCount)`. `nMaxAttackCount` is the skill's max detonatable-drop count (10–20 by level, §3), so for attackCount ≥ 16 the nibble wraps (16 → 0). The decoder MUST NOT use the `hits` nibble to size damage arrays in this variant — only the per-mob count byte. (The client also clamps mobCount to 15 and attackCount to 20 before encoding — v83 `0x96b3fb` lines 194–204 — so both the nibble and the count byte always fit.)

### 2.3 jms_v185 caveat (deviation from FR-2, surfaced for review)

The jms sender `sub_A3AAB1` is readable up to and including drop collection, mob search, and `CMob::AddDamageInfo`, but the packet-encode tail sits behind SCY code-flow virtualization (`JUMPOUT(0xD29D2D)` into mutation-VM junk). The serverbound write order for jms is therefore **not statically IDA-verifiable in the available dump**.

Compensating evidence, all IDA-verified in the jms IDB:

- The dispatcher routes 4211006 to this dedicated sender (same architecture as every GMS version).
- The **clientbound** `CUserRemote::OnAttack` meso branch (`0xa53999` region, lines 134–152) reads per-mob `Decode4(mobId)`, `Decode1(hitAction)`, then — iff skill == 4211006 — `Decode1(count)` + `count × Decode4(damage)`: byte-identical to the GMS clientbound variant and to Atlas's existing clientbound encoder.
- The variant deltas are identical across all four readable versions (v83/v84/v87/v95), spanning both sides of every other layout shift.

**Decision (mirrors the PRD's gms_v92 treatment):** implement jms using the invariant variant deltas on top of jms's already-verified standard melee base layout (`0xa122be`), and document the jms serverbound meso cell as *unverified — sender virtualized* in the audit artifacts. No verification marker is added for a read order we could not read; no evidence hash is fabricated. If a non-SCY jms build ever becomes available, the cell can be promoted then.

### 2.4 gms_92 / gms_12 caveat (template-only, no IDB)

`gms_12` and `gms_92` ship a seed template but are **not** coverage-matrix
columns and have no IDB/IDA export (`PROCESS.md` version set = 9 columns). Per
owner decision (2026-07-26) both are handled **unverified — follows the family
branch**: they decode the meso variant through the same version-gated codec path
their standard attacks already use (`gms_92` → the `GMS >= 87` base layout;
`gms_12` → the very-legacy `GMS < 48` base layout — byte action, no head CRC, no
per-mob CRC), inheriting whichever gates that major version selects. No
`packet-audit:verify` marker or evidence hash is fabricated for either. If an IDB
for either is ever added and it becomes a matrix column, its cell can be verified
then via `VERIFYING_A_PACKET.md`.

## 3. Skill data & drop predicate (FR-5 open questions, resolved)

- **Max detonatable drops** = WZ `attackCount` on skill 4211006 levels 1–30: 10/12/14/16/18/20 stepping every 5 levels (verified in `Skill.wz/421.img.xml`, Cosmic v83 dump; `mobCount` is fixed at 6). The full plumbing already exists: atlas-data reader parses `attackCount` (`services/atlas-data/atlas.com/data/skill/reader.go:212`), the REST model carries it, and atlas-channel's `effect.Model` stores it (`services/atlas-channel/atlas.com/channel/data/skill/effect/model.go:54`) — but **has no accessor**; add `AttackCount() uint32`.
- **Meso-drop predicate** = `Model.Meso() > 0` on the channel drop model (`services/atlas-channel/atlas.com/channel/drop/model.go:37`). Meso drops carry `itemId == 0`, `meso > 0`; the same predicate is already used by the pickup consumer (`kafka/consumer/drop/consumer.go:179`).

## 4. Alternatives considered

### 4.1 Where the variant decode lives

- **A (chosen): variant flag inside the existing `AttackInfo`/`DamageInfo` codec.** `AttackInfo.Decode` reads `skillId` before the damage entries, so the variant is self-detectable mid-decode. The three deltas compose orthogonally with every existing version gate (verified §2.1), so one flag covers all versions. Encode mirrors it for round-trip fixtures.
- **B: separate `MesoExplosionAttackInfo` codec.** Duplicates ~200 lines of version-gated decode that must be kept field-for-field in sync with the melee codec forever; the existing off-by-one history in this file (v84 dr-block) is exactly the drift this would invite. Rejected.
- **C: decode the tail in atlas-channel's handler.** Leaks wire-layout knowledge out of atlas-packet, unfixturable by the packet test harness, and the *mid-packet* delta (count byte replaces delay) can't be done from outside the codec anyway. Rejected.

### 4.2 Where validation runs

- **A (chosen): inside `processAttack`, immediately after the skill-effect load and before the HP/MP cost block.** FR-6 requires zero side effects on rejection; the cost deduction at `character_attack_common.go:303-310` is a side effect, and the effect model (source of `AttackCount()`) is already in hand at that point.
- **B: in the melee packet handler before `processAttack`.** Would need its own character + effect fetches (duplicated REST traffic) and still couldn't stop the cost block without a flag. Rejected.

### 4.3 How CONSUME is emitted

- **A (chosen): one buffered emission carrying N messages.** A `Consume`-style provider per drop id, batched through the existing `message.Buffer`/`producer.ProviderImpl` pattern — N Kafka messages (one per drop, keyed by dropId, as FR-8 requires) in a single produce call.
- **B: N independent `producer.ProviderImpl` calls.** N network round-trips, partial-failure windows between them. Rejected.

## 5. Component design

### 5.1 `libs/atlas-packet` — codec extension

`model/damage_info.go`:
- Add `mesoExplosion bool` to `DamageInfo`, set via new constructor `NewMesoExplosionDamageInfo()` (Builder-style, matching existing constructors).
- `Decode`: when `mesoExplosion`, read `count := r.ReadByte()` **instead of** the `delay` uint16, then `count` damages. `hits` is ignored in this mode.
- `Encode`: symmetric — write `byte(len(damages))` instead of `delay`.
- The per-mob CRC read/write stays on its **existing shared `MajorVersion() >= 61`
  gate** (`damage_info.go:57,83`) — do NOT move it inside the meso branch and do
  NOT re-gate it to `>= 83` (v1's plan snippet was written against the old `>= 83`
  gate and would drop the CRC for v61/v72/v79). Because it stays shared, v48
  (< 61) skips it for free, matching its sender (§2.1a).

`model/attack_info.go`:
- Variant detection: `isMesoExplosion := skill.Id(m.skillId) == skill.ChiefBanditMesoExplosionId`, evaluated right after `skillId` is decoded (and in Encode from the set skillId).
- Damage-entry loop: construct meso-mode `DamageInfo` when the variant is active.
- New model fields (kept faithful so fixtures round-trip byte-identically):
  - `explodedMesoDrops []ExplodedMesoDrop` where `ExplodedMesoDrop{dropId uint32, hitMask byte}` — the hit-mask byte is part of the wire format and must round-trip even though the server logic only uses ids.
  - `mesoDelay uint16` — the trailing delay.
- Decode order: after `characterX`/`characterY` (and only for the variant): count byte, entries, trailing delay. This is positionally *before* the grenade/spark/dragon specials, which can never co-occur with 4211006 (skill-id keyed), so no interaction.
- Accessors/builders: `ExplodedMesoDrops() []uint32` (ids only, empty for non-meso attacks — FR-3), plus internal-fidelity accessor for entries, `SetExplodedMesoDrops(...)`, `SetMesoDelay(...)` for fixture construction.
- No new version gates: the variant deltas are version-invariant; surrounding fields keep their existing `Region()/MajorVersion()` gates.

Tests (`model/attack_info_test.go`, `character/serverbound/attack_request_test.go`):
- Meso-explosion round-trip across all `pt.Variants`: skillId 4211006, ≥2 mobs with *different* damage-line counts (e.g. 3 and 1), ≥2 drop entries with non-zero hit masks, non-zero trailing delay. Note `pt.Variants` now spans **v28, v48, v61, v72, v79, v83, v84, v86, v87, v95, jms_v185** — the meso round-trip runs against all of them, so it exercises the byte-action, no-CRC, single-CRC and two-CRC base-layout gates in the same test (v28/v86 are test-harness boundary variants, not shipped versions — round-trip symmetry only, no IDA claim, like gms_12/gms_92).
- All existing attack fixtures unchanged and passing (FR-4).
- Verification markers: the melee cells for the verified versions are already pinned in `attack_request_test.go` (registry-primary fname `CUserLocal::TryDoingNormalAttack`); the new meso tests document the variant against the §2 sender addresses for all eight IDA-verified GMS cells (v48 `0x6ae4d7`, v61 `0x7b8a39`, v72 `0x875828`, v79 `0x8c22fd`, v83 `0x96b3fb`, v84 `0x9aa379`, v87 `0x9eee04`, v95 `0x942200`). No **new** `packet-audit:verify` marker is added on any cell (the meso senders are `fname_alts` absent from the IDA exports; a second marker would orphan under `matrix --check` — see plan Task 3). jms gets no verify pin for the variant tail (§2.3); gms_12/gms_92 stay unverified-follows-family (§2.4).

Clientbound (`character/clientbound/attack.go`): already encodes the variable count when `isMesoExplosion` (line 127-129) and the decode mirror exists; FR-11 is satisfied by adding a meso-explosion round-trip case to `attack_test.go` with per-mob counts differing from `hits`. The jms clientbound read order for the variant was IDA-confirmed this task (§2.3).

### 5.2 `services/atlas-channel` — validation, damage, destruction

`data/skill/effect/model.go`: add the missing `AttackCount() uint32` accessor.

New pure validation helper (unit-testable without processors), in `socket/handler/character_attack_common.go` or a sibling file:

```go
// validateMesoExplosion returns the offending drop id and false when the
// attack must be rejected; deps are the field's drops and the skill's
// attackCount bound.
validateMesoExplosion(dropIds []uint32, fieldDrops map[uint32]drop.Model, maxCount uint32) (uint32, bool)
```

Checks, in order (FR-5/FR-6/FR-7):
1. `len(dropIds) <= maxCount` (`se.AttackCount()`).
2. No duplicate ids.
3. Every id exists in `fieldDrops` (fetched once via the existing `drop.Processor.InMapModelProvider(s.Field())` — one REST call, FR-5/NFR-perf).
4. Every drop satisfies `Meso() > 0`.

Wiring in `processAttack` (inside the `ai.SkillId() > 0` block, after `se` is loaded, before the cost gate):
- If `skill.Is(skill.Id(ai.SkillId()), skill.ChiefBanditMesoExplosionId)`: fetch field drops, run the helper. On failure: `l.Warnf(...)` with character id, skill id, and offending drop id; return nil (attack fully skipped — no cost, no damage, no broadcast, no consume). On success: stash the validated ids for the post-broadcast side-effect step.
- Note: an empty drop list validates trivially (legal — the player can swing with nothing to detonate).
- Damage lines then flow through `processDamageInfoEntry` untouched (FR-10) — variable-length `di.Damages()` needs no pipeline change.
- Replace the `// TODO destroy Chief Bandit exploded mesos` line (FR-12) with the consume emission: after the broadcast, alongside the projectile emission, emit the buffered CONSUME batch.

`kafka/message/drop/kafka.go`: add `CommandTypeConsume = "CONSUME"` and `ConsumeCommandBody{DropId uint32}` (matches atlas-drops' `CommandConsumeBody`), plus a `TransactionId uuid.UUID` field on the channel's `Command[E]` envelope (atlas-drops already unmarshals it; today's channel envelope simply omits it, wire-compatible either way).

`drop/producer.go` + `drop/processor.go`: `ConsumeCommandProvider(transactionId, f, dropId)` and a processor method `ConsumeAll(f field.Model, dropIds []uint32) error` that buffers one message per drop (keyed by dropId, envelope carrying the attacker's field per FR-8) and emits once. One `uuid.New()` transaction id shared by the batch aids log correlation in atlas-drops.

Broadcast (FR-11): no writer change — `CharacterAttackMeleeBody` already sets `isMesoExplosion` from the skill id (`socket/writer/character_attack_melee.go:19`) and the clientbound encoder already writes per-mob counts from `len(di.Damages())`, which now carries the true variable counts from the serverbound decode.

### 5.3 `services/atlas-drops` — no change (FR-9)

`ConsumeAndEmit` already removes the drop from the registry and emits `CONSUMED`; atlas-channel's drop consumer already announces `DropDestroyWriter` with `DropDestroyTypeExplode` to every session in the field (`kafka/consumer/drop/consumer.go:144`). A meso-only guard inside atlas-drops' `Consume` would break the reactor caller (`services/atlas-reactors/.../item_reactor.go:122`), which consumes *item* drops; the meso-only restriction is an attack-validation concern and lives in atlas-channel (§5.2). Exploded mesos are removed via CONSUME, never PICKED_UP, so no character is credited (AC-4).

### 5.4 Packet audit artifacts

- The eight GMS meso senders (v48/v61/v72/v79/v83/v84/v87/v95) are all IDA-verified (§2, §2.1). No **new** `packet-audit:verify` marker or evidence record is added, however: the CLOSE_RANGE_ATTACK serverbound cells stay pinned to the registry-primary fname, and the meso senders — already registered as `fname_alts` on each version's registry row — are absent from the IDA exports, so a second marker per cell would orphan under `matrix --check` (identical rationale to v1's plan Task 3). The verification lives as the fixture + the §2 evidence excerpts; the audit MDs carry a documentation note. Regenerate `docs/packets/audits/STATUS.md`/`status.json` via the standard tooling only if the tool changes them.
- The jms_v185 melee cell keeps its existing standard-layout pin; the audit MD gains an explicit note: *meso-explosion variant tail unverifiable — sender `0xa3aab1` SCY-virtualized; implemented from v48–v95 invariants + jms clientbound symmetry* (§2.3). gms_v92 (`GMS >= 87` family branch) and gms_12 (very-legacy family branch) are documented unverified-follows-family per §2.4 / FR-2 — but they are template-only, not matrix columns, so they have no audit MD to annotate.
- The four legacy audit MDs (`docs/packets/audits/gms_v{48,61,72,79}/CharacterAttackMeleeRequest.md`) each get a task-150 note recording the meso sender address and the three verified deltas (mirrors the jms note format), so the legacy meso read order is documented at the cell even though no new verify marker is pinned.

## 6. Error handling & security invariants

- Reject-whole-attack on: unknown drop id, non-meso drop, drop outside the attacker's field/instance (the field-scoped fetch enforces this structurally), duplicate id, count above `AttackCount()` (NFR-safety). One warn log names character, skill, and the first offending drop id.
- Decode robustness: counts are wire bytes (≤255); the damage-line loop and drop-list loop are bounded by their explicit byte counts, mob entries by the 4-bit mob count — no attacker-controlled unbounded allocation beyond existing packet-reader semantics.
- Rejection does not `Destroy` the session (unlike unowned-skill use, which keeps its existing behavior): a stale drop id can occur legitimately when a pet/player picks a meso up in flight. Rejecting silently (warn log only) matches the no-side-effects contract.
- Kafka emission failures: log at error level; damage has already been applied at that point (same at-least-once posture as every other post-broadcast side effect in this handler).

## 7. Testing strategy

- **atlas-packet**: per-version round-trip fixtures for the serverbound variant (variable per-mob counts + drop list + trailing delay), clientbound meso round-trip, all existing fixtures green (FR-4). Fixture values chosen to catch nibble-misuse (per-mob counts ≠ hits nibble; attackCount ≥ 16 sample so the wrapped nibble would fail loudly if misused).
- **atlas-channel**: table-driven unit tests for `validateMesoExplosion` (happy path, each rejection class from AC-3); processor test for the batched consume emission using the project's Builder pattern (no `*_testhelpers.go`).
- **Verification gates** (AC-6): `go test -race ./...`, `go vet ./...`, `go build ./...` in `libs/atlas-packet` and `services/atlas-channel`; `tools/redis-key-guard.sh`; `docker buildx bake atlas-channel` (and `atlas-drops` only if it ends up touched, which this design avoids).
- **Integration (AC-2)**: manual in-game pass on a v83 tenant — drop mesos, detonate, observe explode animation on a second session and monster damage; negative test with a forged drop id via the validation warn log.

## 8. Risks & open items

- **The eight GMS cells (v48–v95) are IDA-verified** (§2.1). The deltas are identical across the full range — spanning the byte↔short action shift (v79), the per-mob-CRC boundary (v61), and both head-CRC boundaries (v72, v79) — so the flag-threading has no remaining GMS ambiguity.
- **jms_v185 serverbound variant is implemented, not proven** (§2.3). Risk bounded by: identical deltas across the eight verified GMS versions, verified jms dispatch + drop collection + clientbound symmetry. A wrong guess surfaces as a decode misframe warn on jms meso use, not a crash of other tenants.
- **gms_v92 / gms_12**: unverified by definition (no IDB, not matrix columns); each follows the family branch it already uses for standard attacks (`GMS >= 87` and `GMS < 48` respectively). Same containment as jms — a misframe warns on that tenant only.
- The per-drop hit-mask byte and trailing delay are decoded and retained but unused by server logic; they exist for wire fidelity. If a future task wants Cosmic-style staggered destruction, `mesoDelay`/hit masks are already available.
