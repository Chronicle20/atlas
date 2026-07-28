# Damage-Taken Mitigation/Reaction Skills — Design

Task: task-157-damage-taken-mitigation
Status: Approved design (Phase 2 artifact)
Inputs: `prd.md` (approved), IDA verification of CUserLocal::SetDamaged across gms_v83 / gms_v87 / gms_v95 / jms_v185, extended (post-`main`-merge) to the legacy columns gms_v48 / v61 / v72 / v79 / v84 and to gms_v92 independently — see §3 "Legacy & v92 verification (post-merge extension)".

---

## 1. The central verified fact

The design phase's mandated IDA verification (§3 below) produced one governing result that shapes the whole architecture:

**On every supported version, the client sends the *raw* (pre-mitigation) damage value.** `CUserLocal::SetDamaged` computes the incoming damage from mob stats, then locally subtracts Magic Guard, Meso Guard, Power Guard reflect, Achilles/High Defense, Combo Barrier, and Magic Shield when applying HP — but the value written to the wire (`Encode4(damage)`) is taken *before* all of those subtractions. The only reductions pre-applied to the sent value are the element-matched `DefenseAtt` potion stat and post-BB extras (equip potentials, Mechanic safety summon) that do not exist pre-BB.

Consequently the server does not need to "undo" anything: it re-derives every roster mitigation from its own buff/skill state and applies the adjusted deltas. This matches the PRD's server-authority principle exactly, and it means the anti-cheat posture is simple — client flags are cross-checks, amounts are always server-computed.

The verification also produced three scope corrections to the PRD (all per FR-3.4, client binary wins):

1. **Body Pressure is not a damage-taken mitigation.** The client never reads the BodyPressure stat in `SetDamaged`. Aran's on-touch damage is a *separate serverbound attack packet* sent by `CUserLocal::TryDoingBodyAttack` (v83 opcode 0x2F) — which Atlas already decodes as `AttackTouchRequest` (`libs/atlas-packet/character/serverbound/attack_request.go:95`, routed to `CharacterTouchAttackHandle` → `processAttack`). Body Pressure's server work is a validation concern on the existing touch-attack path, not a branch in `character_damage.go`. Its TODO is removed with this documented redirect.
2. **"Paladin Divine Shield" resolves to the `GUARD` temporary stat, and its only damage-handler obligation is suppressing Power Guard reflect.** On v95, `SetDamaged` reads `nGuard_` exactly once: `if (nPowerGuard_ <= 0 && nGuard_ > 0) nPowerGuardDamage = 0` (0x936119). The skill id is not hardcoded in the client (no immediate 1220013 exists in the v95 binary); the stat is granted by the server-side buff system when the skill is cast, which is the buff domain's job, not this handler's. The v95-only wire byte the current decoder calls `bGuard` is **not** Divine Shield — it is Mechanic Perfect Armor (skill 35101007, verified at v95 pseudocode: prop roll → `nDamage = 0; bGuard = 1`).
3. **Meso Guard degrades partially, not all-or-nothing.** `CalcDamage` (v83 `sub_792FA8`, v95 `CalcDamage::GetMesoGuardReduce`) guards `damage/2`; if mesos cannot cover the cost the guarded amount is scaled down to `100*mesos/x` — it does not drop to zero. FR-6.2's "insufficient mesos → no guard" is replaced by the verified partial-guard formula.

## 2. Wire format findings (drives a mandatory decoder fix)

Verified encode order of the damage-taken packet (client → server), per version:

| Field | v83 (op 0x30) | v87 (op 0x32) | v95 GMS (op 0x34) | jms v185 (op 0x27) |
|---|---|---|---|---|
| updateTime u32 | ✓ | ✓ | ✓ | ✓ |
| nAttackIdx i8 | ✓ | ✓ | ✓ | ✓ |
| nMagicElemAttr i8 | ✓ | ✓ | ✓ | ✓ |
| damage i32 (RAW) | ✓ | ✓ | ✓ | ✓ |
| — mob branch (attackIdx ≥ −2) — | | | | |
| mobTemplateId u32 | ✓ | ✓ | ✓ | ✓ |
| mobId u32 | ✓ | ✓ | ✓ | ✓ |
| left u8 | ✓ | ✓ | ✓ | ✓ |
| reflect byte u8 (currently named `nX`) | ✓ | ✓ | ✓ | ✓ |
| **bGuard u8 (Mechanic Perfect Armor block)** | — | — | ✓ | — |
| block byte u8 (0 / 1 / 2; currently named `relativeDir`) | ✓ | ✓ | ✓ | ✓ |
| **reflect extension — PRESENT ONLY IF `reflect != 0 \|\| blockByte source flag`** | cond. | cond. | cond. | cond. |
| ├ isPowerGuard u8 (currently `bPowerGuard`) | ✓ | ✓ | ✓ | ✓ |
| ├ reflect target mobId u32 (currently `monsterId2`) | ✓ | ✓ | ✓ | ✓ |
| ├ hitAction u8 (currently misnamed `powerGuard` bool) | ✓ | ✓ | ✓ | ✓ |
| ├ hitX i16, hitY i16, charX i16, charY i16 | ✓ | ✓ | ✓ | ✓ |
| stance byte u8 (currently `expression`; v95/jms: bit0 stance, bit1 WH stance skill) | ✓ | ✓ | ✓ | ✓ |
| — non-mob branch (−3 obstacle / −4 stat) — | i8, u8(0), damage i32, obstacleData i16, u8(0) | same | same | same |

Evidence: v83 `CUserLocal::SetDamaged` 0x9581a9 (encode block `if (v171 || *v185)`); v87 0x9da6f2 (same, op 0x32); v95 0x9343c0 encode disasm at 0x93624a–0x93644a (block gated on `bKnockback/nX`, `bGuard` encoded unconditionally at 0x93633e); jms 0xa228f8 encode at 0xa24171–0xa2432b (no bGuard byte; block gated at 0xa24270).

**Bug: the current decoder reads the reflect extension unconditionally** (`libs/atlas-packet/model/damage_taken_info.go:107-113`). On every plain hit (no reflect, no block) it consumes the block byte as `bPowerGuard`, misreads the stance byte into `monsterId2`, and over-reads 12+ bytes past the packet end. Damage/mob ids decode correctly (they precede the block), which is why the handler "works" today; everything after `left` is garbage. Fix in this task (FR-11.3):

- Make the extension conditional: read `nX` (reflect) and the block byte, then read the extension only when `nX != 0 || blockByte != 0`. This matches all four verified clients. (The client's exact condition uses internal flags whose observable wire correlate is `nX != 0 || blockByte != 0`: the block byte is non-zero exactly when the guardian/blocking flag that gates the extension is set, and `nX != 0` covers the reflect case.)
- Keep the v95-GMS-only `bGuard` byte gate as-is (`Region()=="GMS" && MajorVersion()>=95`) — verified correct: present in GMS v95, absent in v83/v87/jms185.
- Rename/document fields to their verified meanings (reflect, blockByte, isPowerGuard, reflectTargetMobId, hitAction, stanceFlags) — straightforward move per project convention, no aliases. Byte-fixture tests per version (both branches, with/without extension) are mandatory.

Damage sentinel semantics (FR-10.1): `damage == -1` is a legitimate "blocked/missed" value produced by Guardian (1120005/1220006 + shield), Fake/Shadow Shifter (jobs 412/422), post-BB block procs, and the smoke-screen/field checks. The client applies 0 HP loss for it. Today's handler would apply `ChangeHP(-int16(-1))` = **+1 HP** — the sentinel must short-circuit to zero damage. (The `1120005/1220006` shield-block proc is a *separate* skill from Achilles `1120004/1220005/1320005`; the legacy re-verification confirmed the two are distinct and that the block proc produces the `-1` sentinel — i.e. no server obligation beyond the sentinel handling above. Do not conflate them.)

### 2a. Legacy versions (v48/61/72/79/84) and v92 — wire layout (post-merge extension)

`main`'s legacy bring-up wired `CharacterDamageHandle` into the gms_48/61/72/79/84 templates (it was only in gms_83/84 when this design was first written), so the decoder now runs on those columns too. Per-version serverbound `TAKE_DAMAGE` opcodes and mob-hit layout, all IDA-verified against the live legacy IDBs:

| Version | Op | Mob-hit layout vs the v83 table above | Send function |
|---|---|---|---|
| gms_v48 | 0x27 | **DISTINCT** — see below | `CUserLocal::SetDamaged` @0x6A598B (standalone) |
| gms_v61 | 0x2D | **= v83, byte-identical** (14-byte extension, same `reflect!=0 \|\| block!=0` gate, no bGuard) | `CUserLocal::SetDamaged` @0x7AA748 |
| gms_v72 | 0x2F | **= v83, byte-identical** | `CUserLocal::SetDamaged` @0x864B92 (virtual, via `CMob::ProcessAttack`) |
| gms_v79 | 0x2E | **= v83, byte-identical** | `CUserLocal::SetDamaged` @0x8B0277 |
| gms_v84 | 0x30 | **= v83, byte-identical** | `CUserLocal::SetDamaged` @0x99634D |
| gms_v92 | 0x35 | **= v83, byte-identical — NO trailing `bGuard` byte** (verified independently, NOT inherited from v87) | `CUserLocal::SetDamaged` @0x913BB0 |

**v48 is the only divergent layout.** Three verified differences from the v83 encode order (evidence @0x6A66AB–0x6A6799):
- **No `nMagicElemAttr` byte** — v48 never encodes the second `Encode1` after `attackIdx`; the `damage` i32 follows `attackIdx` directly.
- **Reflect extension is 10 bytes, not 14** — `isPowerGuard(u8) + reflectTargetMobId(u32) + hitAction(u8) + hitX(i16) + hitY(i16)`; the trailing `charX`/`charY` i16 pair is absent.
- **`stanceFlags` is encoded inside the mob branch only** (v83 encodes it once after the mob/non-mob merge).

Legacy **non-mob/obstacle** branch (v48/61/72/79 — verified): the obstacle path (`0xFE` sentinel + zero byte + `damage` i32 + `obstacleData` u16) has **no trailing `stanceFlags` byte** — one byte shorter than v83's, which emits `stanceFlags` unconditionally after the merge. The decoder must not over-read a stance byte on legacy non-mob events. This path carries no mob, so no roster mitigation applies to it regardless.

Decoder consequence (Task 1): one legacy branch gated on `Region()=="GMS" && MajorVersion() < 61` produces the v48 layout (skip `nMagicElemAttr`, 10-byte extension, mob-branch-only stance); v61 through v92 already decode as v83; the v95-GMS `bGuard` byte stays gated at `>= 95`; jms takes the no-`bGuard` branch. Byte-fixture tests add a v48 mob-hit + v48 obstacle case and one representative legacy (v79) = v83 case.

## 3. Verification matrix (FR-3.1/3.2)

Legend: "sent value" = the i32 damage on the wire. All formulas integer math as decompiled. Function evidence per version: v83 `CUserLocal::SetDamaged` 0x9581a9, helpers 0x95CDC3 (Achilles/High Defense), 0x95FA8B (elemental DefenseAtt), 0x79309F/0x79345E + 0x792FA8 (CalcDamage + MesoGuard); v87 0x9da6f2 (+ analogous helpers, verified inline); v95 0x9343c0 + `CUserLocal::GetAchillesReduce` 0x908c40 + `CalcDamage::PDamage` 0x72e720; jms 0xa228f8 (tail verified; mitigation set structurally identical to v83/v87).

| Skill | Pre-applied to sent value? | Server formula (verified) | Applies to | Proc authority | v83 | v87 | v95 | jms185 |
|---|---|---|---|---|---|---|---|---|
| Magic Guard (2001002/12001001/22111001) | **No** | portion = damage×x/100; MP loss = min(portion, curMP); HP absorbed = portion capped at curMP unless **Infinity** buff active (then full portion absorbed, no MP cost) | mob touch, mob attack, obstacle, stat | none | ✓ cookie[672]=MagicGuard, [1908]=Infinity, MP cap = CharacterData MP | ✓ cookie[680]/[1964] | ✓ `nMagicGuard_`/`nInfinity_`/`nMP` (callees) | ✓ same slot layout in TryConsumePetHP call |
| Power Guard (1101007/1201007) | **No** (reflect subtracted only locally) | reflect = pg%×damage/100; cap mob MaxHP/10; **÷2 for boss** (v83 `sub_66918F`/v87 `sub_6A3AFA`); if template fixedDamage>0 → min(reflect, fixedDamage); 0 if mob invincible or immune to skill 1101007/1201007 (`IsVulnerableTo` v95); client also rolls an accuracy check vs mob avoid; self HP loss reduced by reflect | mob-sourced hits where caller passes pg% (touch verified; wire `isPowerGuard`+`nX` signal which hits the client mitigated) | client evade roll; server recomputes amount, honors wire signal as cross-check | ✓ | ✓ | ✓ (`nPowerGuardDamage`; suppressed if `nGuard_`>0 & no PG) | ✓ |
| Meso Guard (4211005) | **No** | guarded = damage/2; cost = x%×guarded/100; if mesos < cost → guarded = 100×mesos/x (partial); HP loss reduced by guarded | mob touch + mob attack (computed inside CalcDamage P/MDamage) | none | ✓ 0x792FA8 | ✓ (7E427E/7E3E70 out-param) | ✓ `GetMesoGuardReduce` in PDamage 0x72e720 | ✓ (slot in final subtraction) |
| Mana Reflection (2121002/2221002/2321002) | **No** (does not reduce own damage at all) | on magic mob attacks (AttackInfo magic flag): roll prop% ; reflect = damage×x/100, cap mob MaxHP/20; accuracy + immunity checks; **no self-damage reduction** | mob *skill* attacks flagged magic only | **client rolls prop** (v83 0x9581a9 MR block; v95 0x9354c3 `rand%100 < nProp`); outcome visible on wire (`nX`>0 & !isPowerGuard); server validates + recomputes amount | ✓ | ✓ cookie[1844] | ✓ 0x9354a9–0x935542 | ✓ (structurally) |
| Achilles (1120004/1220005/1320005) | **No** | reduce = damage×(1000−x)/1000, x per-mille from passive level data; job-gated 112/122/132 | all damage types incl. obstacle/stat | none | ✓ 0x95CDC3 | ✓ 0x9DF4AA | ✓ `GetAchillesReduce` 0x908c40 | ✓ |
| High Defense (21120004) | **No** | **same function as Achilles** — job 2112 selects 21120004; identical per-mille formula. Answer to PRD open Q4: it IS a damage-pipeline hook, not stat-only | all damage types | none | ✓ 0x95CDC3 job 2112 | ✓ | ✓ 0x908c40 job 2112 | ✓ |
| Combo Barrier (21120007) | **No** | reduce = (damage − achillesReduce)×(1000−x)/1000, x per-mille from buff | all damage types | none | ✓ cookie[3028] = SecondaryStat ComboBarrier (offset-verified via `SecondaryStat::Reset` 0x7807bf, base delta +240) | ✓ cookie[3084] | ✓ `nComboBarrier_` | ✓ |
| Body Pressure (21101003) | n/a | **not in the damage-taken path** — separate serverbound touch-attack packet (`TryDoingBodyAttack`, v83 op 0x2F) already handled by `CharacterTouchAttackHandle`; server work = damage validation on that path | — | client rolls; per-mob damage arrives in the attack packet | ✓ 0x95f135 | — (same by structure) | — | — |
| Divine Shield (post-BB Paladin) | n/a | maps to `GUARD` temporary stat; damage-handler obligation = **suppress PG reflect when GUARD active and POWER_GUARD absent** (v95 0x936119). Block mechanics = buff-domain (charge bookkeeping), no wire signal; skill id not client-hardcoded — resolve from v95 skill WZ during plan phase before adding the constant | v95+ only | server (buff state) | n/a | n/a | ✓ | not observed |
| Magic Shield (Evan 22131001, `EvanStage5MagicShieldId` — **discovered**, PRD open Q6) | **No** | reduce = damage×x/100 from buff (v87: applied to damage−MG portion; v83: full damage — use per-version gate or v83 form, see §6) | all damage types | none | ✓ cookie[3352] = MagicShield (offset-verified) | ✓ cookie[3408] | ✓ `nMagicShieldConv` | ✓ (slot present) |
| Elemental DefenseAtt potion | **Yes — pre-applied to sent value** | server must NOT re-apply; listed so nobody adds it | — | — | ✓ 0x95FA8B | ✓ 9E2C53 | ✓ `CalcBuffDefenseAttr` | ✓ |
| Chakra (4211001) cast-state | **Yes** (multiplier applied before send) | damage×x/100 while charging Chakra — pre-applied; server does nothing | — | — | ✓ | ✓ | ✓ 0x9353c3 | ✓ |
| Mechanic Perfect Armor (35101007) | v95 GMS: block → damage 0, `bGuard=1` on wire | pre-BB tenants unaffected; v95 server treats `bGuard=1` as informational (damage already 0) | — | client rolls prop | n/a | n/a | ✓ 0x935397+ | no wire byte |

### Legacy & v92 verification (post-merge extension)

The four-column matrix above was extended to gms_v48/v61/v72/v79/v84 and gms_v92 by decompiling each version's `CUserLocal::SetDamaged` against the v83 reference (session-per-version IDBs; see §2a for send-function addresses). **v92 was verified independently and does NOT inherit v87** (overriding the original FR-3.3 no-IDB assumption — the v92 IDB exists). The governing result holds on every legacy column: `SetDamaged` writes the raw damage to the wire and subtracts every roster mitigation only into the local `TryConsumePetHP` HP-delta, so the server re-derives everything — identical to §1.

Per-skill result relative to the v83 row (verified, not assumed):

| Skill | v48 | v61 | v72 | v79 | v84 | v92 |
|---|---|---|---|---|---|---|
| Magic Guard | absent¹ | absent¹ | = v83² | = v83 (cached-slot MP/HP split, offsets +113/+117) | = v83² | = v83² |
| Power Guard | = v83 **minus** fixedDamage clamp and immune-zero³ | = v83 | = v83 | = v83 | = v83 | = v83 (cap **/10** @0x914464, fixedDamage **min** @0x9144e0 — pre-BB, not v95's /2 + replace) |
| Meso Guard | = v83 (partial-scale not re-derived) | = v83 (100×mesos/x confirmed) | = v83 | = v83 (100×mesos/x confirmed) | = v83 | = v83 |
| Mana Reflection | = v83 **minus** MaxHP/20 cap and immune-gate³ | = v83 | = v83 | = v83 | = v83 | = v83 (cap /20) |
| Achilles / High Defense | dead-code⁴ | dead-code⁴ | = v83² | = v83 (functional, `dmg×(1000−x)/1000`) | = v83 | = v83² |
| Combo Barrier (Aran) | n/a (no Aran) | n/a | n/a | present (Aran ≈v76) | present | present² |
| Magic Shield (Evan) | n/a (no Evan) | n/a | n/a | n/a | present (Evan v84)⁵ | present² |
| GUARD-suppress-PG / Mechanic Perfect Armor | absent | absent | absent | absent | absent | **absent** (v95-only — confirmed 0 hits in v92) |

¹ v48/v61: no MP/HP-split logic located in `SetDamaged` and no `2001002` reference. **Immaterial to the server** — the pipeline is data-driven, so a version whose data never grants the Magic Guard buff simply no-ops; no server gate is needed.
² Stat-cookie-driven skills (Magic Guard, Magic Shield, Combo Barrier, and Achilles' passive read) are resolved by temporary-stat offset, not a skill-id immediate, so a per-function immediate search cannot prove presence/absence; "= v83" here means the same stat-read structure was found or the same non-presence pattern holds in both v83 and the target. The server applies these from buff/skill data regardless of the client's local read.
³ v48 omits the fixedDamage clamp, the Power-Guard invincibility zero, and the Mana-Reflection MaxHP/20 cap that later versions have. The server applies all three universally: they only ever bound a reflect **downward**, so applying them on v48 is safe and anti-exploit-consistent — no v48-specific formula gate is warranted.
⁴ v48/v61 are **DEVM (developer) builds** in which `GetAchillesReduce` computes the reduction then unconditionally returns 0. This is a client-local artifact and does not affect a server-authoritative, data-driven implementation: the server applies Achilles from the character's skill level, and its authoritative `CHANGE_HP` is what the client renders. (The v83/v87/v95 IDBs the original design used are also DEVM builds; the wire format — what the decoder actually gates on — is structural and reliable.)
⁵ Magic Shield's v83-full-damage vs v87-reduced-damage form (§6) is **stat-cookie-driven** and could not be re-corroborated by the legacy immediate-search pass. The original design-phase cookie-offset finding is retained: the `MajorVersion() >= 87` gate stands, so v84 uses the v83 (full-damage) form. This is the single roster gate the legacy re-verification could not independently confirm; it affects one Evan-only skill on a narrow version band.

**Net effect on the implementation:** the only structural code change the legacy columns force is the v48 decoder branch (§2a). Every mitigation *formula* gate already in the plan (`pgCapDivisor`, `pgFixedDamageOverride`, `magicShieldOnReducedDamage`, the v95 GUARD rule) is verified correct across all ten columns — the pre-BB legacy versions fall into the pre-BB branch of each gate, exactly as the version-number conditions already express. Per-version skill *availability* needs no explicit gates because the design is data-driven and server-authoritative.

Attack-type applicability (FR-2.1): obstacle (−3) and stat (−4) events skip CalcDamage (no mob), so Meso Guard, Power Guard, and Mana Reflection cannot apply; Achilles/High Defense, Combo Barrier, Magic Guard, and Magic Shield are computed in the shared tail and DO apply to them (verified v83 control flow: non-mob path jumps to LABEL_215 but the reduction block at LABEL_241 runs for both).

## 4. Approaches considered

**A. Pure mitigation functions + deps-injected orchestrator in the handler package (recommended).** Mirror the proven `character_attack_common.go` pattern (`damageInfoEntryDeps`, `processDamageInfoEntry`): a `damageMitigationDeps` struct injects `getBuffs`, `getCharacter`, `getMonster`, `getEffect`, `applyHP/MP`, `requestMesoChange`, `damageMonster`; the mitigation math is pure functions over a small immutable context struct, unit-testable without REST/Kafka. Fits existing code shape, smallest review surface.

**B. Kafka-projected buff mirror (StatusMirror pattern) + local registry.** Would eliminate the per-hit REST buff lookup (`character/buff` is REST-per-call, `requests.go:9-18`). Rejected for v1: it duplicates buff state into a second source of truth, needs its own consumers/lifecycle, and the damage path already performs a REST character fetch per event today — one more call is a measured, bounded cost. The deps seam makes swapping the lookup for a mirror later a one-line change. Recorded as the designated optimization if profiling shows the buff call dominating.

**C. Full server recompute of incoming damage from mob stats (ignore client damage).** Maximum authority, but requires porting the entire `CalcDamage` PDamage/MDamage formula chain (mob PAD, level scaling, PDD curves, per-version doubles) — a much larger, riskier task, and the PRD scopes the client damage value as clamped input, not recomputed. Rejected; the clamp (§8) bounds the abuse surface.

Choice: **A**, with one addition — a tenant-scoped read-through cache for skill-effect data (immutable per tenant) using the project registry pattern (`sync.Once` + `RWMutex`), because `data/skill.GetEffect` is an uncached REST call today and effects are static.

## 5. Architecture

New/changed units, all in existing services:

```
libs/atlas-packet/model/damage_taken_info.go     — conditional decode + field renames + fixtures (§2)
services/atlas-channel/.../socket/handler/
  character_damage.go                            — thin orchestrator (decode → announce → mitigate → emit)
  character_damage_mitigation.go                 — pure functions + deps struct + context/result types
  character_damage_mitigation_test.go            — unit tests (Builder pattern)
services/atlas-channel/.../kafka/message/character/kafka.go
                                                 — add CommandRequestChangeMeso + RequestChangeMesoBody
services/atlas-channel/.../character/processor.go — RequestChangeMeso producer (mirror RequestDropMeso)
services/atlas-channel/.../data/skill/            — effect cache (tenant+skill+level keyed, read-through)
libs/atlas-constants/                             — Divine Shield skill id constant only, added after v95 WZ
                                                    verification (plan phase). GUARD and MAGIC_SHIELD temp-stat
                                                    constants already exist (character/temporary_stat.go:104,85)
```

### Data flow per damage event

1. Decode `DamageTakenInfo` (fixed decoder). Classify: sentinel (−1) / obstacle-stat / mob-sourced.
2. Announce to foreign sessions immediately with the client-reported fields (unchanged, FR-2.5) — never blocked on mitigation.
3. Fetch inputs: character (`GetById`, already fetched today — add `SkillModelDecorator` **only when** the job is Hero/Paladin/DK/Aran lines, resolvable from the base model's `JobId()` without the decorator); active buffs (`buff.GetByCharacterId`, one REST call); monster (`monster.GetById`) only when a reflect will actually be computed.
4. Run the pure chain (§6) → `mitigationResult{hpLoss, mpLoss, mesoCost, reflect{mobId, amount, kind}, blocked, breakdown}`.
5. Emit side effects: `ChangeHP(−hpLoss)`, `ChangeMP(−mpLoss)` (existing), `RequestChangeMeso(−mesoCost)` (new producer), `monster.Damage(f, mobId, charId, []uint32{reflect}, attackType)` for PG/MR reflect. Debug-log the breakdown; warn-log clamps/mismatches (FR-10.4).

Buff amounts come from the buff itself (`stat.Model{Type, Amount}`) for MAGIC_GUARD/POWER_GUARD/MESO_GUARD/COMBO_BARRIER/MAGIC_SHIELD — no effect fetch needed on the hot path. Effect fetches are needed only for: Mana Reflection (prop + x via `SourceId()`+`Level()` → `GetEffect`, cached) and the warrior/Aran passive (skill level from decorated model → `GetEffect`, cached).

Note on `message.Buffer` (FR-6.3): socket handlers in atlas-channel do not currently use the buffer pattern — side effects go through processor emit methods (per the attack-path precedent). Meso Guard's invariant is enforced by *deciding* mesoCost and hpLoss together in the pure chain from the server-side meso balance (no fire-and-forget dependence on `NOT_ENOUGH_MESO`), then emitting both commands in the same handler invocation. If the plan phase finds a practical way to route both through one buffered emission it may, but the correctness requirement is the joint decision, which this design satisfies.

## 6. Mitigation chain (server mirror of verified client math)

Order and formulas, applied to `raw` = wire damage (int32, already clamped per §8):

```
if raw == -1 (block sentinel): hpLoss = 0; validate a block source exists (Guardian/Fake/
    Shadow Shifter passive, GUARD buff, or v95 bGuard) else warn-log; done.

achillesReduce   = raw*(1000-ax)/1000        ax: passive X per-mille (job-selected; Aran→High Defense)
comboBarrier     = (raw-achillesReduce)*(1000-cbx)/1000       cbx: COMBO_BARRIER buff amount
magicShield      = raw*msx/100                                msx: MAGIC_SHIELD buff amount
magicGuardPortion= raw*mgx/100                                mgx: MAGIC_GUARD buff amount
    mpLoss = min(magicGuardPortion, currentMP); absorbed = mpLoss
    if INFINITY buff active: absorbed = magicGuardPortion; mpLoss = 0
mesoGuarded      = raw/2; mesoCost = mgx2*mesoGuarded/100     mgx2: MESO_GUARD buff amount
    if meso < mesoCost: mesoGuarded = 100*meso/mgx2; mesoCost recomputed (≈ meso)
    (mob-sourced events only)
pgReflect        = pgx*raw/100                                pgx: POWER_GUARD buff amount
    cap: mob.MaxHp()/10, halved for boss; min vs template fixedDamage when >0; 0 if mob
    dead/missing; 0 on v95+ when GUARD buff active and no POWER_GUARD
    (mob-sourced physical events only; wire isPowerGuard+nX are cross-checks, never inputs)
hpLoss = max(0, raw - achillesReduce - comboBarrier - magicShield - absorbed
                 - mesoGuarded - pgReflect)
manaReflect (magic mob attacks, MANA_REFLECTION buff, wire signals reflect without isPowerGuard):
    amount = raw*x/100 capped mob.MaxHp()/20 → monster.Damage; no hpLoss change
```

Per-version notes: v87 computes Magic Shield on `(raw − magicGuardPortion)`; v83 on `raw`. Magic Shield is Evan-only (v84+); use the v87 form for `MajorVersion() >= 87` tenants and the v83 form otherwise — a one-line version gate, recorded in the matrix. Boss-halving and fixedDamage caps use monster/template data already exposed by the atlas-channel monster model (`Hp()/MaxHp()`); if the boss flag or fixedDamage is not exposed, the plan phase adds the getter to the monster/data model rather than skipping the cap.

Mana Reflection proc authority (PRD open Q5, verified): the client rolls `prop`. The wire tells the server the outcome (`nX > 0` with `isPowerGuard == false` on a magic mob attack). The server honors the roll *signal* only when its own MANA_REFLECTION buff is active and the event is a mob skill attack, and always recomputes/caps the amount itself. A forged signal without the buff is ignored and warn-logged. (Re-rolling server-side would desync the client's already-displayed reflect; honoring a validated, amount-recomputed signal cannot be inflated by the client.)

Power Guard applicability nuance (recorded honestly): `SetDamaged` mitigates PG whenever its caller passes `nPowerGuard > 0`; the caller is virtual-dispatched and was not decompiled. The wire signal says exactly which hits the client mitigated, so the server gate is: server-side POWER_GUARD buff active AND mob-sourced physical event AND wire signals `isPowerGuard` — amounts always server-computed. Mismatches (flag without buff, reflect target ≠ attacking mob) → ignore claim, warn-log (FR-5.4).

## 7. Kafka / API surface

- **New producer only**: atlas-channel emits `REQUEST_CHANGE_MESO` (`RequestChangeMesoBody{ActorId, ActorType, Amount, ShowEffect}`) to `COMMAND_TOPIC_CHARACTER` — contract verified against the consumer at `services/atlas-character/atlas.com/character/kafka/message/character/kafka.go:22,127-132`. Pattern-copy of the existing `RequestDropMeso` producer (`character/processor.go:267`).
- Reused: `CHANGE_HP`, `CHANGE_MP` (atlas-character), monster `Damage` command (atlas-monsters). PG/MR reflect goes through `monster.Damage` (attribution to the character, matching Cosmic); `EmitDamageReflected`/`DAMAGE_REFLECTED` is **not** used for Power Guard — its existing consumer applies character HP loss (`consumer.go:526-539`), i.e. it models mob-side reflect, the opposite direction.
- No new topics, REST endpoints, or migrations. atlas-character and atlas-monsters unchanged.
- Error cases: meso shortfall resolved in-chain from the model's `Meso()` (partial guard, §6); reflect target missing/dead → drop reflect (debug log) but keep character-side mitigation.

## 8. Anti-cheat clamping (FR-10)

- `damage` accepted range: `-1` (sentinel) or `[0, 999999]` — 999999 is the client's own hard cap in every version's CalcDamage (`dbl_AFE8A0`, v95 PDamage clamp). Values outside → clamp to bounds, warn-log with character/map/mob/raw fields.
- Reflect amounts are never read from the wire: `nX` is an echo (PG: buff percent; MR: truncated amount) used only for cross-checking. All reflect magnitudes derive from the server buff value × clamped damage, capped by mob HP rules. The classic PG exploit (inflated reflect) is structurally impossible.
- `-int16(p.Damage())` truncation fixed: hpLoss is computed in int32 and clamped to the `ChangeHP` int16 contract bounds before conversion (pre-BB max HP ≤ 30000 so no legitimate loss exceeds int16; the CHANGE_HP body's int16 width is an existing cross-service contract, out of scope to widen — the clamp plus a debug log satisfies FR-10.2 and is recorded here as the boundary).
- Sentinel `-1` accepted only with a plausible block source (§6); positive HP from negative damage is impossible by construction (hpLoss floored at 0).
- Observability: structured warn logs for every clamp/mismatch; no new metrics stack (none exists in atlas-channel today).

## 9. Version gating (FR-11)

- Packet layout: three gates — the v95-GMS-only `bGuard` byte (already present, verified correct), the new conditional reflect extension (identical condition on all versions, no gate needed), and the **v48-era layout** gate `Region()=="GMS" && MajorVersion() < 61` (no `nMagicElemAttr`, 10-byte extension, mob-branch-only stance; §2a). v61 through v92 decode as v83.
- Skill availability: by data — a character on a version without the skill never has the buff/skill, so no explicit per-skill gates except: Magic Shield formula variant (`>=87`, §6) and the GUARD-suppresses-PG rule (harmless everywhere, only reachable on tenants whose buff system grants GUARD — effectively v95+). The legacy verification (§3) confirms this holds for the pre-BB columns: absent skills (Magic Guard pre-v72, Aran pre-≈v76, Evan pre-v84) and client-dead-code (Achilles on the v48/v61 DEVM builds) all resolve to server no-ops via the data-driven, server-authoritative path — no per-version formula gates required.
- Divine Shield (FR-9.3): pre-BB tenants cannot reach the code path because GUARD is never granted. The skill-id constant is added to `libs/atlas-constants` only after v95 WZ verification in the plan phase (never from memory).
- v92: **verified independently against its own IDB — NOT inherited from v87** (§3). It is byte-identical to v83 (no `bGuard`, Power-Guard cap `/10`, fixedDamage `min`, no GUARD/Mechanic rule), i.e. it falls into the pre-BB branch of every existing gate.
- Handler wiring: `main` wired `CharacterDamageHandle` into gms_48/61/72/79/84 (was gms_83/84 only). gms_87/92/95/jms still lack the handler entirely (pre-existing gap — the packet is never routed there), so this task also wires those four templates (serverbound opcodes v87=0x32, v92=0x35, v95=0x34, jms=0x27; all free of handler collision) so the verified v87/v92/v95/jms behavior is actually reachable. See plan Task 7.

## 10. Testing

- **Packet fixtures** (libs/atlas-packet): per version × {plain hit, hit with PG reflect, hit with block, obstacle, stat, v95 bGuard set}, byte-exact decode/encode round-trips. These encode the §2 matrix. Add the **v48-era layout** cases (mob-hit with the 10-byte extension and no `nMagicElemAttr`; obstacle with no trailing stance byte — §2a) and a representative legacy `= v83` case (v79); the tenant-driven decode gate is exercised by both branches.
- **Mitigation unit tests** (pure functions, Builder pattern per project convention, no `*_testhelpers.go`): no-buff passthrough (FR-2.4, byte-identical deltas), Magic Guard standard + MP shortfall + Infinity, Meso Guard standard + partial-afford + zero-meso, Power Guard cap/boss/fixedDamage/dead-mob/GUARD-suppression, Achilles + High Defense job selection, Combo Barrier stacking order (post-Achilles), Magic Shield v83 vs ≥87 form, sentinel −1, forged oversized damage clamp, forged reflect claim ignored, int16 boundary.
- **Handler tests**: deps-struct fakes verifying emission set per scenario (HP only; HP+MP; HP+meso; HP+mob damage) and that announce always fires.
- Verification commands per CLAUDE.md: `go test -race`, `go vet`, `go build` in atlas-channel + libs/atlas-packet + (if touched) libs/atlas-constants; `docker buildx bake atlas-channel`; `tools/redis-key-guard.sh`.

## 11. PRD open questions — resolved

1. Client pre-applies to sent damage: only elemental DefenseAtt potions and Chakra state (plus post-BB equip potential/Mechanic extras). All roster mitigations are server's to apply. (§1, §3)
2. Formulas/rounding/caps/order: §3 matrix + §6 chain, integer math as decompiled.
3. Divine Shield: GUARD stat; damage-handler obligation = PG-reflect suppression; skill id from v95 WZ in plan phase; `bGuard` wire byte is Mechanic Perfect Armor, not Divine Shield. (§1, §3)
4. High Defense: damage-pipeline hook, same function as Achilles (job 2112 → 21120004). (§3)
5. Proc authority: MR/Perfect Armor rolled client-side, signaled on wire; server validates buff + recomputes amounts. Body Pressure rolls client-side inside its own attack packet. (§6)
6. Block-type skills: Guardian/Fake produce the −1 sentinel (server: zero damage, validate source); Magic Shield discovered and included in the chain; Mechanic Perfect Armor documented, no pre-BB obligation. (§2, §3)

## 12. Out of scope (unchanged from PRD)

Battleship HP (task-153, TODO stays), Berserk interplay (task-154 — HP flows through existing CHANGE_HP events), attack-side passives (task-147), UI. Post-BB-only stats beyond the roster (SwallowDefence, BlueAura/SuperBody, equip potentials, Safety summon) are recorded in §3 but not implemented — they require buff-system support that doesn't exist for those stats and affect only v95+ tenants; any future post-BB skill work slots into the same chain.
