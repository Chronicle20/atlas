# Energy Charge — Implementation Context

Companion to [`plan.md`](plan.md). Everything here was read out of the tree at
plan time; line numbers are anchors, not guarantees — re-grep if a hunk has
moved.

---

## 1. What exists today

`grep -rE "EnergyCharge|ENERGY_CHARGE"` finds declarations only, never
behaviour:

| Thing | Where | State |
|---|---|---|
| `TemporaryStatTypeEnergyCharge = "ENERGY_CHARGE"` | `libs/atlas-constants/character/temporary_stat.go:116` | exists |
| Two-state base-stat membership | `libs/atlas-packet/model/character_temporary_stat.go:1130` (`baseStatNames`), `:1192` (`twoStateBaseStats` slot 1, kind `twoStateDynamic`) | exists, **encoded as zeros** |
| `CharacterAttackEnergy` writer | `libs/atlas-packet/character/clientbound/attack.go:19` | unrelated — broadcasts an energy-*type attack*, not the bar |
| Accumulation / charged state / reset | — | **absent everywhere** |

Skill/job constants all exist and are version-stable:

| Name | Identity + wire id | File |
|---|---|---|
| `MarauderEnergyCharge` | 5110001 | `libs/atlas-constants/skill/identities_gen.go:338`, `constants.go:3203` |
| `MarauderEnergyBlast` | 5111002 | `identities_gen.go:339`, `constants.go:3204` |
| `ThunderBreakerStage2EnergyCharge` | 15100004 | `identities_gen.go:497`, `constants.go:3284` |
| `ThunderBreakerStage2EnergyBlast` | 15101005 | `identities_gen.go:501`, `constants.go:3285` |
| `ThunderBreakerStage3SharkWave` | 15111007 | `identities_gen.go:510`, `constants.go:3295` |

`libs/atlas-constants/skill/version_gms_61_1_gen.go` binds `5110001` and does
**not** bind `15100004` — that is the mechanism AC-10's "gms_v61 Cygnus branch
is a no-op" relies on, and `energyChargeLine` exercises it via `set.Wire`.

---

## 2. The three design corrections that move work

Restated from `design.md` §1 because they are the load-bearing surprises:

1. **The wire block is zeros.** `getBaseTemporaryStats`'s `default:`
   (`twoStateDynamic`) arm calls `NewCharacterTemporaryStatBase(true, narrow)`
   (`character_temporary_stat.go:1396-1397`), which leaves `nOption`/`rOption`
   at zero (`:420-430`). Only `twoStateMonsterRiding` (`:1374`) and
   `twoStateGuidedBullet` (`:1382`) use the `…WithOptions` constructor. The v83
   client reads `nOption` as the bar reading (`sub_7F9BAD`:
   `this[364] / this[365] * flt_B38988`). So **libs/atlas-packet must change** —
   PRD FR-4.5 ("no packet change") is wrong.
2. **Energy Blast is an attack skill.** WZ shows `damage 246 / mobCount 2 /
   lt / rb` and **no `time`** for both `5111002` and `15101005`, so they arrive
   on an ATTACK packet and land in `processAttack`
   (`character_attack_common.go:739`), never in `CharacterUseSkillHandleFunc`.
   The Enrage precedent PRD FR-6.2 names would never fire; the right precedent
   is `battleshipAttackPermitted` (`character_attack_common.go:785-791`,
   `:1070`).
3. **`UPDATE_STAT_VALUE` cannot create the buff.** `Registry.UpdateStatValue`
   returns `(Model{}, false, nil)` on a missing buff
   (`services/atlas-buffs/atlas.com/buffs/character/registry.go:373-375`) — by
   design, and Combo depends on it. Hence the opt-in `CreateIfMissing` upsert.

---

## 3. Key files, by module

### libs/atlas-packet

- `model/character_temporary_stat.go`
  - `getBaseTemporaryStats` `:1362-1410` — the one function Task 1 edits.
  - `NewCharacterTemporaryStatBaseWithOptions` `:429` — the constructor to switch to.
  - `decodeBaseTemporaryStats` `:1321-1350` — **shape-only**; it consumes the
    block without storing values, so populating `nOption`/`rOption` cannot
    break round-trip symmetry. `pt.RoundTrip` only asserts the reader is fully
    drained (`libs/atlas-packet/test/roundtrip.go:22-34`), not value fidelity.
  - Block widths: dynamic base is **15 bytes** on every in-scope version,
    **14 on GMS v61** (`narrowTimeField` — v61 drops the bool-prefixed time
    byte; IDA-verified `sub_66E9B6`, task-167).
- `model/character_temporary_stat_test.go` — `TestCTSHomingBeaconPre95PopulatedBlock`
  `:275` and `TestCTSHomingBeaconV61PopulatedBlock` `:408` are the templates to
  copy for the new ENERGY_CHARGE fixtures (same `pt.CreateContext` /
  `tenant.Create` / `AddStat` / `bytes.Index` shape).
- Fixture byte math: bar `4998` = `0x1386` → LE `86 13 00 00`; skill
  `5110001` = `0x004DF8F1` → LE `F1 F8 4D 00`.

### atlas-buffs

- `character/registry.go`
  - `Apply` `:70-122` — how a Model is constructed when the character has none
    (needs `worldId`/`channelId`); `srcKey(sourceId)` is the whole-source key.
  - `UpdateStatValue` `:360-420` — the method Task 2 rewrites.
- `buff/model.go:165-179` — `NewNoExpiryBuff(sourceId, level, changes)`;
  errors with `ErrEmptyChanges` on an empty change set.
- `character/processor.go`
  - interface `:25`, `UpdateStatValue` `:168-189`, `Apply` `:49-76`.
  - `appliedStatusEventProvider` / `statUpdatedStatusEventProvider` are the two
    emit providers; `Apply` shows the full `appliedStatusEventProvider`
    argument order.
- `kafka/message/character/kafka.go`
  - `UpdateStatValueCommandBody` `:94-100`, `Command[E]` envelope `:33-41`
    (carries `ChannelId`, which the consumer currently drops).
  - `ApplyCommandBody.Duration` `:56` — **the** authoritative statement that
    the buff-command duration unit is milliseconds.
- `kafka/consumer/character/consumer.go:103-110` — `handleUpdateStatValue`.
- `buff/rest.go:13` — atlas-buffs already serves `level` over REST.
- Test scaffolding: `character/registry_test.go:20-45`
  (`setupTestRegistry` via miniredis, `setupTestTenant`, `setupTestContext`),
  `character/processor_test.go:19-34` (`setupProcessorTest`),
  `character/testmain_test.go` (`producertest.InstallNoop()` — so emits are
  swallowed in tests). Assertions use `testify/assert`.

### atlas-channel

- `socket/handler/character_attack_common.go`
  - `processAttack` `:739` — the single funnel every attack type reaches
    (`character_attack_melee.go:21`, `_magic.go:21`, `_ranged.go:21`,
    `_touch.go:18-21`, the last building an `AttackTypeEnergy` AttackInfo).
  - `t` / `set` / `attackId` / `attackIdOk` resolved once at `:756-758`.
  - `battleshipAttackPermitted` gate `:785-791`, function `:1070` — the
    soft-rejection precedent for Task 8.
  - `comboOrbTryUpdate` call site `:980-982` — where Task 7's block goes.
  - `LookupAttackCast` call `:730` — the attack path's ONLY per-skill hook,
    which is why FR-7 needs no code.
- `socket/handler/character_attack_combo.go` — the structural template for
  `character_attack_energy_charge.go`: `comboLine` `:23-29`, `comboSkillIds`
  `:36-62`, `comboOrbDeps` `:137-141`, `comboOrbProductionDeps` `:145-156`,
  `comboOrbTryUpdate` `:166-198`, and its swallow-and-log error contract
  `:175, 188, 195`.
- `socket/handler/character_skill_use.go:114-140` — the Enrage gate (fails
  OPEN on a buff-read error); `:171` is the Enrage orb consume that Task 4
  updates; `:176-178` shows the `AnnounceSkillUse` / `AnnounceForeignSkillUse`
  call shape with `c.Level()`.
- `socket/handler/effects.go:20`, `:32` — `AnnounceSkillUse` /
  `AnnounceForeignSkillUse`.
- `character/buff/beacon.go` — the mirror template for `energy.go`
  (singleton + `sync.Once` + `sync.RWMutex` + per-tenant map);
  `beacon_test.go` shows the `energyMirrorOnce = sync.Once{}` reset idiom.
- `character/buff/processor.go` — `UpdateStatValue` `:93-96`, `Apply` `:74-80`
  (`model.Operator[uint32]`, so it is called as `Apply(...)(characterId)`),
  `GetByCharacterId` `:70`.
- `character/buff/producer.go` — `ApplyCommandProvider` `:14-40`,
  `UpdateStatValueCommandProvider` `:123-141`.
- `character/buff/model.go` — `Model.Changes() []stat.Model`, `SourceId()`,
  `Level()`, `Expired()`; `character/buff/stat/model.go` — `Type()`/`Amount()`.
- `kafka/consumer/buff/consumer.go`
  - `announceBuffGive` `:88-115` (owner GIVE_BUFF + foreign fan-out).
  - `handleStatusEventApplied` `:117-193`, `handleStatusEventStatUpdated`
    `:196-215`, `handleStatusEventExpired` `:217-263`.
  - Helper style to copy: `beaconChange` `:422`, `isBeaconOnly` `:434`,
    `isBattleshipRide` `:459`, `newBattleshipProcessor` seam `:476`.
  - **OQ-4 resolution lives here**: `handleStatusEventExpired` already sends
    `CharacterBuffCancel` to the owner and `CharacterBuffCancelForeign` to the
    map, so the reset needs no extra emit.
- `data/skill/effect/model.go:80-85` — `Duration()` returns **milliseconds**,
  `-1` is the no-duration sentinel. `weaponAttack` is already parsed into the
  model (`rest.go:11`) but has **no getter**; add one only if a call site
  materialises — the plan's channel-side code never needs `pad`.
- `data/skill/effect/statup/model.go` — `statup.NewModel(buffType, amount)`,
  used to synthesize the charged `ENERGY_CHARGE = 15000` statup.
- `kafka/message/buff/kafka.go` — `UpdateStatValueCommandBody` mirror `:74-83`,
  `StatUpdatedStatusEventBody` `:124-131` (carries `Level`, which the promotion
  needs), `AppliedStatusEventBody` `:99-108`, `ExpiredStatusEventBody` `:111-119`.
- `character/processor.go:71` — `GetById(decorators ...)(characterId)`;
  no caching, so each call is a REST hit.
- `libs/atlas-packet/model/attack_info.go:16-21` — `AttackTypeMelee` 0,
  `AttackTypeRanged` 1, `AttackTypeMagic` 2, `AttackTypeEnergy` 3. There is no
  separate "touch" type: touch attacks ARE `AttackTypeEnergy`.

### atlas-effective-stats

- `character/initializer.go`
  - `fetchBuffBonuses` `:175-198` — the function Task 10 edits.
  - `fetchPassiveBonuses` `:201-270` — shows the
    `skilldata.RequestById(id)(l, ctx)` → `GetEffectForLevel(level)` →
    `effect.WeaponAttack` path the energy bonus reuses.
- `stat/model.go:442-500` — `BonusesForBuffChange`. It has **no**
  `ENERGY_CHARGE` arm, so the five-digit-stat hazard of FR-5.3 is avoided by
  default; the special case is additive, not corrective. `"WEAPON_ATTACK"` /
  `"PAD"` map to `TypeWeaponAttack` (`:15`); `NewBonus` at `:75`.
- `external/buffs/rest.go:9-16` — `BuffRestModel`, **missing `Level`**.
- `external/data/skill/rest.go:39-46` — `GetEffectForLevel` returns
  `*EffectModel` or nil; `EffectModel.WeaponAttack` is `int16` (`:50`).
- `character/stubs_test.go` — five httptest servers + `PointEnv` for the
  integration-style initializer tests. The new energy tests are pure-helper
  tests and do NOT need this harness.

---

## 4. Verified game-data values

Do not re-derive these from memory; they came from WZ and are recorded in
`design.md` §1.3.

| Skill | `time` (charged window) | `pad` (weapon attack) | levels |
|---|---|---|---|
| `5110001` Marauder Energy Charge | 31 s @ L1 → 40 s @ L20 (table runs to L40 / 50 s) | 0 @ L1–3, 11 @ L4–5, 15 @ L20 | 40 |
| `15100004` TB Stage2 Energy Charge | same 31 s @ L1 → 40 s @ L20 shape | **11 from L2** — a different table | 20 |

The WZ `time` node is **seconds**; `effect.Model.Duration()` already returns
**milliseconds**, and `ApplyCommandBody.Duration` is milliseconds. No scaling
at any emit site — `tools/buff-duration-guard.sh` fails CI on one.

MP-cost asymmetry (the only evidence behind OQ-3's "Energy Blast only"):
`5111002` and `15101005` carry **no `mpCon` at any level**; Shockwave
`5111006` carries `mpCon 18`, Shark Wave `15111007` carries `mpCon 15`.

---

## 5. Version scope

| Version | `5110001` | `15100004` | Note |
|---|---|---|---|
| gms_v12, gms_v48 | absent | absent | **n-a** — Pirates postdate both |
| gms_v61 | present | **absent** | Cygnus branch must be a no-op (AC-10) |
| gms_v72 … gms_v95, jms_v185 | present | present | full support |

---

## 6. AC-9 — ENERGY_CHARGE encoding coverage

Task 1 changes how one base block serializes inside `GIVE_BUFF` /
`GIVE_FOREIGN_BUFF` / `CANCEL_BUFF`. Matrix state at plan time
(`docs/packets/audits/STATUS.md`):

| Row | line | v48 | v61 | v72 | v79 | v83 | v84 | v87 | v92 | v95 | JMS185 |
|---|---|---|---|---|---|---|---|---|---|---|---|
| `GIVE_BUFF` | `:63` | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | **❌** | ✅ | ✅ |
| `CANCEL_BUFF` | `:65` | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | **❌** | ✅ | ✅ |
| `GIVE_FOREIGN_BUFF` | `:266` | ⬜ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | **❌** | ✅ | ✅ |

No existing fixture pins an `ENERGY_CHARGE` base block, so no verified cell
asserted the zeros being replaced — **Task 11 Step 1 confirms this by running
the suite rather than repeating the claim.** v92 is already ❌ on all three rows
independently of this task; it is out of scope, and the reason (pre-existing
❌; task-216 changes a value, not a wire shape) is what Task 11 Step 2 records.

### Observed result (Task 11 Step 1, run on the task branch)

`grep -rn "ENERGY_CHARGE" libs/atlas-packet/ | grep -i test` returns four hits,
all in `libs/atlas-packet/model/character_temporary_stat_test.go` and all added
by this task: `TestCTSEnergyChargePre95PopulatedBlock` (`:931`) with its two
assertions (`:968`, `:991`) and `TestCTSDashSpeedStaysZeroed` (`:1011`). No
pre-existing fixture pins an `ENERGY_CHARGE` base block, so no verified matrix
cell asserted the zeros Task 1 replaced.

`cd libs/atlas-packet && go test ./...` is fully green — every package reports
`ok` or `no test files`, zero `FAIL` lines. The pre-existing `GIVE_BUFF` /
`GIVE_FOREIGN_BUFF` / `CANCEL_BUFF` fixtures stayed green across the whole
supported set, confirming the claim above by execution rather than assertion.

**No matrix cell moved, and none is claimed to have moved.** Task 1 changes a
*value* inside an already-verified wire shape (the base block's `nOption` /
`rOption` pair, previously emitted as zeros for this stat); it adds no field,
removes none, and shifts no offset. `docs/packets/audits/STATUS.md` is therefore
unchanged by this task.

Per-version outcome for the §5 supported set:

| Version | `GIVE_BUFF` | `GIVE_FOREIGN_BUFF` | `CANCEL_BUFF` | Task-216 status |
|---|---|---|---|---|
| gms_v48 | ✅ | ⬜ | ✅ | n-a — Pirates postdate v48 (§5); no ENERGY_CHARGE traffic |
| gms_v61 | ✅ | ✅ | ✅ | in scope, covered by the new v61 fixture; cell already ✅, unchanged |
| gms_v72 | ✅ | ✅ | ✅ | in scope; cell already ✅, unchanged |
| gms_v79 | ✅ | ✅ | ✅ | in scope; cell already ✅, unchanged |
| gms_v83 | ✅ | ✅ | ✅ | in scope, the version the bar-reads-`nOption` finding was derived on; cell already ✅, unchanged |
| gms_v84 | ✅ | ✅ | ✅ | in scope; cell already ✅, unchanged |
| gms_v87 | ✅ | ✅ | ✅ | in scope; cell already ✅, unchanged |
| gms_v92 | ❌ | ❌ | ❌ | **out of scope** — the row was already ❌ on all three ops before this task; task-216 makes no wire-shape change, only a value change, so there is nothing here to promote or regress |
| gms_v95 | ✅ | ✅ | ✅ | in scope; cell already ✅, unchanged |
| jms_v185 | ✅ | ✅ | ✅ | in scope; cell already ✅, unchanged |

The v48 `GIVE_FOREIGN_BUFF` ⬜ is likewise untouched: it was unimplemented before
this task and remains so, and v48 carries no Energy Charge skill to encode.

---

## 7. Dependency order

```
Task 1  (packet)          ── independent
Task 2  (buffs registry)  ──▶ Task 3 (buffs processor + consumer)
Task 3                    ──▶ Task 4 (channel command mirror)
Task 4                    ──▶ Task 6 (channel helpers)
Task 5  (energy mirror)   ──▶ Task 6, Task 8, Task 9
Task 6                    ──▶ Task 7 (attack call site), Task 8 (blast gate)
Task 9  (buff consumer)   ── needs Task 5 only
Task 10 (effective-stats) ── independent of every channel task
Task 11 (verification)    ── last
```

Tasks 2 and 3 leave atlas-buffs un-buildable between them (the processor still
calls the old registry signature). Do their implementation steps back-to-back
if a green build is wanted at every commit.

---

## 8. Guards that apply

| Guard | Why it is in scope |
|---|---|
| `tools/buff-duration-guard.sh` | Task 9 emits an `APPLY` with a `duration` field. Pass `se.Duration()` through unscaled. |
| `tools/skill-job-id-guard.sh` | Tasks 6 and 8 compare skill identities. Always via `set.Wire` / `set.Resolve` + `skill3.IsIdentity`, never a raw `==` on a `…Id` constant. |
| `tools/redis-key-guard.sh` | Task 2 touches an atlas-buffs registry backed by Redis — but only through the existing `r.characters` store, so no new keyed command is introduced. |
| `tools/goroutine-guard.sh` | No new goroutines are planned. If one becomes necessary, it must go through `routine.Go`. |
| `tools/lint.sh --check` | Every task. Run fix mode (`tools/lint.sh`) before committing. |

Not in scope: the four `tools/template-*-guard.sh` scripts (no tenant socket
template changes), `tools/service-registration-guard.sh` (no service
registration lists touched), `tools/trade-contract-mirror-guard.sh` (no trade
contract touched).

---

## 9. Things deliberately NOT done

- **`DASH_SPEED` / `DASH_JUMP` / `UNDEAD` stay zeroed.** They share the
  `twoStateDynamic` kind and are almost certainly under-encoded the same way,
  but no evidence was gathered for what their clients read, and this task must
  not make an unverified wire change to already-verified cells. Task 1's
  `TestCTSDashSpeedStaysZeroed` pins that boundary.
- **No `pad` / `acc` / `eva` statups on the charged `APPLY`.** They would need
  no effective-stats change (`BonusesForBuffChange` already maps
  `WEAPON_ATTACK`), but statups are CTS bits: the client would light up buff
  icons neither Cosmic nor the real client shows for Energy Charge. The stat
  is a server-side combat value.
- **The bar is not consumed by a charged cast** (FR-6.3). Only the charged
  window's own timer resets it.
- **No job-line check.** PRD FR-1.2's `job.Marauder`-and-above /
  `job.ThunderBreakerStage2`-and-above test is dropped as redundant: owning the
  skill at level > 0 already implies the line, and `set.Wire` does explicitly
  what Cosmic's `isCygnus()` was implicitly deciding.
- **`calcDmgMax`, Buccaneer Energy Orb, Energy Drain, Body Pressure** — PRD §2
  non-goals, unchanged.
