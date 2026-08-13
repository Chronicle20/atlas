# Energy Charge — Design

Task: task-216-energy-charge
PRD: [`prd.md`](prd.md) (approved)
Status: Draft for review
Created: 2026-08-12

---

## 1. What changed relative to the PRD

Three PRD assumptions did not survive contact with the code. Each is stated here
with its evidence, because each moves work into a different service than the PRD
anticipated.

### 1.1 The `ENERGY_CHARGE` wire block is currently encoded as zeros (FR-4.5 is wrong)

The PRD's FR-4.5 asserts "No new opcode, writer, or template routing is
introduced… the `ENERGY_CHARGE` two-state dynamic CTS entry already exists". The
entry exists; **its value does not reach the wire.**

`ENERGY_CHARGE` is a two-state *base* stat
(`libs/atlas-packet/model/character_temporary_stat.go:1130`,
`:1192`), so it is never written as a per-stat value block — it is written as a
trailing base block by `getBaseTemporaryStats`. For `twoStateDynamic` members
that function currently writes:

```go
default: // twoStateDynamic
    list = append(list, NewCharacterTemporaryStatBase(true, narrow)) // 15 (14 on GMS v61)
```

(`libs/atlas-packet/model/character_temporary_stat.go:1396-1397`)

`NewCharacterTemporaryStatBase` leaves `nOption` and `rOption` at zero
(`:420-430`); only `twoStateMonsterRiding` / `twoStateGuidedBullet` use the
`…WithOptions` constructor that carries `s.Value()` / `s.SourceId()`
(`:1374`, `:1382`). So a `GIVE_BUFF` carrying `ENERGY_CHARGE = 4998` puts
`nOption = 0` on the wire.

The client reads `nOption` as the bar reading. Verified on the GMS v83 IDB
(`MapleStory_dump.exe.i64`, session `41f13e0d`):

- `sub_95F065` (the local `OnTemporaryStatChanged` fan-out) tests the received
  mask against `dword_BF1298`, and on a hit opens UI window 20 (`CUIEnergyBar`,
  `CWvsContext::UI_Open(v7, 20, -1)`), then calls
  `sub_7F9BAD(pEnergyBarWnd, *ZFatalSection::Lock(v11), <virtual getter>)` —
  i.e. it passes the two-state entry's **first field** (`nOption`) and a second
  scalar.
- Inside `sub_7F9BAD` (`0x7F9BAD`) those two arguments are stored at
  `this[364]` / `this[365]` and the fill is computed as
  `a2 = sub_A62018(this[364] / this[365] * flt_B38988)` — value ÷ max × bar
  width. The first argument is the bar reading.

Consequence: **`libs/atlas-packet` must change.** The `twoStateDynamic` branch
must emit `NewCharacterTemporaryStatBaseWithOptions(true, s.Value(),
s.SourceId(), narrow)`.

Scope of that change (deliberately narrow):

- The block **shape and size are unchanged** — only the two leading int32s stop
  being zero. No mask, order, or length changes, on any version.
- `DASH_SPEED`, `DASH_JUMP` and `UNDEAD` are the group's other `twoStateDynamic`
  members and are almost certainly under-encoded for the same reason. They are
  **out of scope**: no evidence was gathered for what their clients read, and
  this task must not make an unverified wire change to already-verified cells.
  The change is therefore applied per-stat, keyed on the stat being
  `ENERGY_CHARGE`, not by flipping the shared `default:` arm.
- `GIVE_BUFF` / `GIVE_FOREIGN_BUFF` / `CANCEL_BUFF` matrix cells
  (`docs/packets/audits/STATUS.md:63`, `:266`, and the CANCEL_BUFF row) are ✅ on
  most versions today. No existing fixture pins an `ENERGY_CHARGE` base block
  (`grep ENERGY_CHARGE libs/atlas-packet/**/*_test.go` finds only comments about
  the group's mask shifts), so no verified cell asserts the zeros we are
  replacing. The implementer must re-run those fixtures and confirm that
  claim before landing, and add a fixture that pins a non-zero
  `ENERGY_CHARGE` block (see §7, AC-9).

### 1.2 Energy Blast is an attack skill, not a `USE_SKILL` cast (FR-6.2 needs relocating)

FR-6.2 says to model the gate on the Enrage precedent in
`services/atlas-channel/atlas.com/channel/socket/handler/character_skill_use.go:114-132`.
A gate there would never fire.

WZ (`tmp/<tenant>/GMS/83.1/Skill.wz/511.img.xml`, `1510.img.xml`) shows both
Energy Blast entries carry `damage` / `mobCount` / `lt` / `rb` and **no `time`**:

| Skill | level 1 | notes |
|---|---|---|
| `5111002` Marauder Energy Blast | `damage 246, mobCount 2, lt/rb` | no `mpCon` |
| `15101005` TB Energy Blast | `damage 246, mobCount 2, lt/rb` | `req: {15100004: 1}` |

They arrive on a melee ATTACK packet and land in `processAttack`
(`character_attack_common.go:739`), not `CharacterUseSkillHandleFunc`. The
correct precedent is `battleshipAttackPermitted`
(`character_attack_common.go:785-791`, `:1070`): a soft rejection *before* any
cost, damage, or broadcast, returning `nil` rather than destroying the session.

That precedent also dictates the data source: the battleship gate reads a
**pod-local mirror** precisely so the attack path stays free of REST (FR-6.2 of
task-183). The energy gate does the same (§4.4).

### 1.3 The charged buff cannot be created by `UPDATE_STAT_VALUE` (§6 of the PRD is incomplete)

`Registry.UpdateStatValue` returns `(Model{}, false, nil)` when the buff is
missing (`services/atlas-buffs/atlas.com/buffs/character/registry.go:373-375`)
— by design, and Combo relies on it. The PRD's data model has the accumulating
buff created "on first qualifying hit with no existing buff", but the channel
cannot know whether that buff exists without a REST read on the attack path,
which NFR-1 forbids. §4.2 resolves this.

Two PRD facts *were* confirmed and are load-bearing:

- WZ level tables (`Skill.wz/511.img.xml` `5110001`, `1510.img.xml` `15100004`):
  `time` 31 s at L1 → 40 s at L20 (Marauder's table runs to L40/50 s);
  `pad` 0 at L1–3, 11 at L4–5, 15 at L20 for `5110001`. The Cygnus table
  differs — `15100004` has `pad 11` from **L2**, and 20 levels total. Nothing
  may hard-code one table for both.
- `MarauderEnergyCharge` is present in every provisioned version's set from
  gms_v61 up; `ThunderBreakerStage2EnergyCharge` is **absent from gms_v61**
  (`libs/atlas-constants/skill/version_gms_61_1_gen.go` contains `5110001` and
  not `15100004`).

---

## 2. Architecture

Energy Charge is one buff row in atlas-buffs, one stat on it, and four
reactions to that stat's lifecycle. No new service, no new topic, no new
opcode.

```
ATTACK packet (melee | touch/energy | ranged+SharkWave)
  └─ processAttack (atlas-channel)
       ├─ [FR-6] energyBlastPermitted(mirror)  ── reject before any side effect
       └─ …damage / broadcast…
           └─ energyChargeTryUpdate(...)                 1 Kafka emit, fire-and-forget
                 UPDATE_STAT_VALUE {INCREMENT, 102×mobs, Cap 10000, CreateIfMissing}
                      ↓
                 atlas-buffs                              atomic get-modify-put
                      ├─ buff absent → create NoExpiry buff, ENERGY_CHARGE=amount → APPLIED
                      └─ buff present → clamp to cap                          → STAT_UPDATED
                      ↓  EVENT_TOPIC_CHARACTER_BUFF_STATUS
                 atlas-channel buff consumer
                      ├─ (existing) GIVE_BUFF to owner + GIVE_FOREIGN_BUFF to map   [FR-4.1/4.3]
                      ├─ CharacterEffect / CharacterEffectForeign (skill-use)       [FR-4.2]
                      ├─ energy mirror ← authoritative value                        [FR-6]
                      └─ amount == 10000 → APPLY {duration = effect.Duration(),
                                                  ENERGY_CHARGE = 15000}            [FR-3.1]
                      ↓
                 atlas-buffs timer → EXPIRED
                      ↓
                 atlas-channel buff consumer (existing)
                      └─ CANCEL_BUFF + CANCEL_FOREIGN_BUFF, mirror cleared          [FR-3.3/4.4]

GET /characters/{id}/effective-stats (atlas-effective-stats)
  └─ fetchBuffBonuses: ENERGY_CHARGE amount == 15000
        → skill effect at buff.Level → weapon_attack += pad                          [FR-5]
```

The single buff row is reused across both phases, keyed by `srcKey(sourceId)`
in atlas-buffs, so the charged `APPLY` **replaces** the accumulating buff
in place (`registry.go` buff map is `map[srcKey]buff.Model`). There is never a
moment with two Energy Charge buffs.

---

## 3. Resolved open questions

| PRD OQ | Resolution |
|---|---|
| OQ-1 cast-gate drift | **(b) reject and re-announce.** On rejection the channel does a single `GET /characters/{id}/buffs` (off the hot path — only on a rejected cast) and re-announces the authoritative `ENERGY_CHARGE` value to the owner via `CharacterBuffGive`. A dead-input bug becomes self-healing. The gate additionally **fails open** when the mirror has no entry at all (§4.4). |
| OQ-2 who owns 10000→15000 | **Channel-driven**, reacting to `STAT_UPDATED` with amount 10000. atlas-buffs stays skill-agnostic and the transition is exactly-once by construction (§4.3). Buff *creation*, however, moves into atlas-buffs as a generic upsert (§4.2) — that is a separate concern from the phase transition and does not encode any skill rule. |
| OQ-3 charged-skill set | **Energy Blast only** (`5111002`, `15101005`). No client-side charge gate was found in the v83 IDB: an `insn_query` for `cmp` against `15000` over `0x401000–0xA07E7B` returned five hits, all in `CDropPool::TryPickUpDrop` neighbours and `CWndMan::s_Update` — none in the attack path. One `cmp eax, 2710h` (10000) sits inside `CUserLocal::Update` (`0x94BC91`); whether that is energy-related is **unverified** and is not relied upon. Shockwave (`5111006`) carries `mpCon 18` and Shark Wave (`15111007`) `mpCon 15`, i.e. they are MP-costed skills; Energy Blast carries no `mpCon` at any level. That asymmetry is the only evidence, and it supports gating Energy Blast alone. |
| OQ-4 reset broadcast fidelity | **No extra emit needed.** `handleStatusEventExpired` (`kafka/consumer/buff/consumer.go:216-262`) already sends `CharacterBuffCancel` to the owner and `CharacterBuffCancelForeign` to the map, and `EncodeMask` claims exactly the stats the CTS holds (task-190), so the reset names `ENERGY_CHARGE` and nothing else. Cosmic's "give 0" and Atlas's "reset that bit" are the same instruction to the client. The implementer confirms the bar visibly zeroes during the live pass (AC-6). |
| OQ-5 task number | Accepted as given; 213–215 are in-flight worktrees (`.worktrees/task-214-*`, `task-215-*`) so no gap in practice. No action. |

---

## 4. Design decisions

### 4.1 Eligibility resolves from the owned skill through the version-aware set

`comboSkillIds` resolves a line by scanning the character's skill book for
known wire ids with a `// version-stable per task-187 audit` annotation
(`character_attack_combo.go:31-63`). Energy Charge cannot copy that verbatim:
the Cygnus skill does not exist on gms_v61, and the guard/resolver discipline
prefers identities.

```go
type energyLine struct {
    skillId skill.Id  // resolved WIRE id for this tenant
    level   byte
}

func energyChargeLine(set skill.Set, skills []characterskill.Model) (energyLine, bool)
```

It walks `[skill.MarauderEnergyCharge, skill.ThunderBreakerStage2EnergyCharge]`
as **identities**, calls `set.Wire(identity)` (`libs/atlas-constants/skill/identity.go:45`)
to get the tenant's wire id — which returns `false` on gms_v61 for the Cygnus
identity, satisfying AC-10's "no-op rather than a bogus id" — and returns the
first one the character owns at level > 0.

The PRD's FR-1.2 job check (`job.Marauder` and above / `job.ThunderBreakerStage2`
and above) is **dropped as redundant**: owning `5110001` at level > 0 already
implies the Marauder line, and Cosmic's `isCygnus()` exists only to pick which
id to look up — which `set.Wire` now does explicitly. Fewer inputs, same
semantics, one less thing to keep in sync with job tables.

### 4.2 Accumulation: one emit, buffs-side create-or-increment

`UpdateStatValueCommandBody` gains two optional fields (owned by atlas-buffs,
mirrored in the channel's local copy):

```go
type UpdateStatValueCommandBody struct {
    SourceId  int32  `json:"sourceId"`
    StatType  string `json:"statType"`
    Operation string `json:"operation"`
    Amount    int32  `json:"amount"`
    Cap       int32  `json:"cap"`
    // CreateIfMissing turns INCREMENT into an upsert: when the character has no
    // buff for SourceId, one is created with NoExpiry and a single StatType
    // change of min(Amount, Cap), and APPLIED (not STAT_UPDATED) is emitted.
    // Level is the source skill level stamped on that created buff.
    CreateIfMissing bool `json:"createIfMissing"`
    Level           byte `json:"level"`
}
```

Semantics stay generic — "accumulator upsert" is the same family as the
`INCREMENT`+`Cap` the command already owns, and carries no skill knowledge.
Omitted/false leaves every existing caller (Combo, Enrage) byte-identical.

Why not the alternatives:

- **Channel-side existence mirror, APPLY-then-INCREMENT.** Rejected. The mirror
  is only fed by buff-status events (the beacon mirror precedent,
  `character/buff/beacon.go`) and nothing seeds it on session enter — the
  channel makes no buff fetch at login (`grep GetByCharacterId` finds only the
  Enrage gate). The first attack after a channel change would see an empty
  mirror, emit `APPLY`, and **reset a full bar to 102** — breaking the PRD's
  "energy state survives channel change" goal outright. It also races two
  in-flight attacks into two `APPLY`s.
- **Read the buff on the attack path to decide.** Rejected by NFR-1 (zero
  blocking REST on the attack path).

The gain amount is computed once per attack, never per monster (NFR-1):
`102 × len(ai.DamageInfo())`, `Cap 10000`. FR-2.5 ("no gain while charged") is
structural, not a guard: at 15000 the registry's `current >= capValue` test
(`registry.go:391-394`) makes the increment a no-op and emits nothing.

### 4.3 The phase transition is channel-driven and exactly-once

`handleStatusEventStatUpdated` gains an Energy Charge arm: when the event
carries an `ENERGY_CHARGE` change whose amount is exactly **10000** and the
character's session is present on this channel, the channel emits

```
APPLY { sourceId = <same skill id>, level = e.Body.Level,
        duration = effect.Duration(),         // already MILLISECONDS
        statups  = [ENERGY_CHARGE = 15000] }
```

Exactly-once holds because atlas-buffs emits `STAT_UPDATED` only when the value
actually changed (`processor.go:179-186`) and clamps at the cap — so exactly one
event in the bar's life carries 10000. (Kafka redelivery of that one event would
refresh the charged window; the same at-least-once posture every other buff
emit in this codebase carries. Not mitigated.)

`effect.Duration()` is fetched with `skill.NewProcessor(l, ctx).GetEffect(sourceId, level)`
inside the consumer — a REST call off the attack path — and passed through
**unscaled**: `data/skill/effect/model.go:80-85` already returns milliseconds and
`ApplyCommandBody.Duration` is milliseconds (`tools/buff-duration-guard.sh`
enforces the absence of a `×1000`).

FR-4.2's skill-use effect (`CharacterEffect` to the owner,
`CharacterEffectForeign` to the map, via the existing `AnnounceSkillUse` /
`AnnounceForeignSkillUse` in `socket/handler/effects.go:20`,`:32`) is emitted from
this same consumer arm, on `APPLIED` and `STAT_UPDATED` for the stat. Emitting
it here rather than at the attack site is what keeps it honest: it fires once
per *actual value change*, so a hit against a full bar produces no packet.

### 4.4 The Energy Blast gate reads a pod-local mirror and fails open

New `character/buff/energy.go` mirror, shaped exactly like `BeaconMirror`
(`character/buff/beacon.go:60`): `Set(tenant, characterId, value)` from
`APPLIED` and `STAT_UPDATED` for `ENERGY_CHARGE`, `Clear` from `EXPIRED`.

In `processAttack`, immediately beside the battleship gate:

```go
if !energyBlastPermitted(l, ctx, s.CharacterId(), attackId, attackIdOk) {
    // Debug log, re-announce authoritative bar, return nil
}
```

- Not an Energy Blast identity → permitted.
- Mirror has an entry and it is `15000` → permitted.
- Mirror has an entry below 15000 → **rejected**, then one
  `GET /characters/{id}/buffs` and a `CharacterBuffGive` re-announce of the
  authoritative value to the owner (OQ-1 (b)).
- Mirror has **no** entry (fresh channel, pod restart) → **permitted**. A
  missing mirror entry means "unknown", and an unknown must never eat a
  legitimate cast.

Zero REST on the permitted path; one REST only on a rejection.

### 4.5 The charged weapon-attack bonus stays server-side, in effective-stats

`fetchBuffBonuses` (`services/atlas-effective-stats/atlas.com/effective-stats/character/initializer.go:174-197`)
gains one special case ahead of the generic `BonusesForBuffChange` dispatch:

```
change.Type == "ENERGY_CHARGE":
    if change.Amount != 15000 { skip }                      // FR-5.3
    skill := skilldata.RequestById(buff.SourceId)
    eff   := skill.GetEffectForLevel(buff.Level)
    if eff.WeaponAttack > 0 { bonus(buff:<sourceId>, weapon_attack, eff.WeaponAttack) }
```

`BonusesForBuffChange` has no `ENERGY_CHARGE` arm today
(`stat/model.go:442-500`), so the five-digit-stat hazard of FR-5.3 is already
avoided by default; the special case is additive, not corrective.

One contract gap: `BuffRestModel` in effective-stats has no `Level`
(`external/buffs/rest.go:9-16`) while atlas-buffs serves one
(`services/atlas-buffs/atlas.com/buffs/buff/rest.go:13`). Add the field — it is
a pure widening of an existing payload.

**Rejected alternative:** put `pad`/`acc`/`eva` statups on the charged `APPLY`
itself, which would need no effective-stats change at all because
`BonusesForBuffChange` already maps `WEAPON_ATTACK`. Rejected because those
statups are CTS bits: the client would light up `PAD`/`ACC`/`EVA` buff icons
that neither Cosmic nor the real client shows for Energy Charge. The stat is a
server-side combat value, not a displayed buff.

### 4.6 FR-7's touch-refresh guard is already structural

Cosmic needed an explicit "don't re-apply Energy Charge on its own touch
damage" guard because its `AbstractDealDamageHandler` applies the attacking
skill's effect. Atlas's attack path applies **no** skill statups: effect
application lives in `handler.UseSkill` (`skill/handler/common.go:101`), reached
only from the `USE_SKILL` packet, and the attack path's only per-skill hook is
the separate `attackCastTryApply` registry — which Energy Charge is not, and
must not be, registered in.

So FR-7.1 requires **no new code**. It requires a *test* pinning the invariant
(AC-12) and a comment at the registry saying why Energy Charge must never be
registered as an attack-cast handler. FR-7.2 (touch attacks still grant energy)
is satisfied by including `AttackTypeEnergy` in the call-site gate.

### 4.7 Call-site gate

`comboOrbTryUpdate` is gated on `ai.AttackType() == AttackTypeMelee`
(`character_attack_common.go:980`). Energy Charge's gate is wider and lives
right beside it:

```go
if ai.AttackType() == packetmodel.AttackTypeMelee ||
   ai.AttackType() == packetmodel.AttackTypeEnergy ||
   (ai.AttackType() == packetmodel.AttackTypeRanged && isSharkWave(attackId, attackIdOk)) {
    energyChargeTryUpdate(l, c, ai, energyChargeProductionDeps(l, ctx, s.Field(), s.CharacterId()))
}
```

`isSharkWave` resolves `skill.ThunderBreakerStage3SharkWave` through the
identity set (wire `15111007`,
`libs/atlas-constants/skill/constants.go:3295`), never a raw compare —
`tools/skill-job-id-guard.sh` territory, and correct regardless.

---

## 5. Components and files

| Module | File | Change |
|---|---|---|
| libs/atlas-packet | `model/character_temporary_stat.go` | `getBaseTemporaryStats`: `ENERGY_CHARGE` emits `…WithOptions(true, s.Value(), s.SourceId(), narrow)`. Other dynamic members untouched (§1.1). |
| libs/atlas-packet | `model/character_temporary_stat_test.go` | Fixture pinning a non-zero `ENERGY_CHARGE` base block on v83 and on v61 (narrow shape). |
| atlas-buffs | `kafka/message/character/kafka.go` | `UpdateStatValueCommandBody`: `CreateIfMissing`, `Level`. |
| atlas-buffs | `character/registry.go` | `UpdateStatValue`: upsert branch when the buff is missing and `CreateIfMissing`. Returns a `created` signal alongside `changed`. |
| atlas-buffs | `character/processor.go` | Emit `APPLIED` for a created buff, `STAT_UPDATED` otherwise. |
| atlas-channel | `kafka/message/buff/kafka.go` + `character/buff/producer.go` | Mirror the two new fields; `UpdateStatValue` signature gains them (or an options variant, implementer's call — Combo's call site must stay readable). |
| atlas-channel | `socket/handler/character_attack_energy_charge.go` (new) | `energyLine` resolution, gain computation, `energyChargeDeps`, `energyChargeTryUpdate`, `energyBlastPermitted`, `isSharkWave`. All pure but the deps struct. |
| atlas-channel | `socket/handler/character_attack_common.go` | Blast gate beside the battleship gate; accumulation call beside `comboOrbTryUpdate`. |
| atlas-channel | `character/buff/energy.go` (new) | `EnergyMirror` (Set/Get/Clear), same shape as `BeaconMirror`. |
| atlas-channel | `kafka/consumer/buff/consumer.go` | Mirror maintenance on APPLIED/STAT_UPDATED/EXPIRED; 10000→15000 transition; FR-4.2 effect announce. |
| atlas-channel | `data/skill/effect/model.go` | `WeaponAttack()` getter (field already parsed, `rest.go:11`). Needed only if the channel resolves `pad`; not needed for the transition itself — add only if a call site materialises. |
| atlas-effective-stats | `external/buffs/rest.go` | `Level byte` field. |
| atlas-effective-stats | `character/initializer.go` | `ENERGY_CHARGE` special case in `fetchBuffBonuses`. |

No template, opcode, docker-bake, or k8s change. `go.mod` is untouched in every
module, so `docker buildx bake` is not required (CLAUDE.md item 4).

---

## 6. Error handling

- **Attack path (FR-2.6 / NFR-2).** `energyChargeTryUpdate` returns nothing and
  logs every failure at Error with character id and skill line, exactly as
  `comboOrbTryUpdate` does (`character_attack_combo.go:175, 188, 195`). No
  branch can return an error to `processAttack`.
- **Gate.** A mirror miss permits the cast; a re-announce failure after a
  rejection logs at Error and is swallowed — the rejection itself already
  happened.
- **Consumer.** Effect lookup failure at the transition logs at Error and skips
  the promotion; the bar stays at 10000 and the next hit re-emits nothing (at
  cap), so the character sits at a full-but-uncharged bar until the buff is
  cancelled. Documented, not papered over: a silent retry loop here would be
  worse than a visible stall.
- **atlas-buffs.** Upsert failures propagate as today (message.Emit rolls the
  buffer back).

---

## 7. Testing

Unit (channel, table-driven with injected deps, Builder-pattern models — no
`*_testhelpers.go`):

- `energyChargeLine`: adventurer owned; Cygnus owned; both owned (adventurer
  wins); neither; Cygnus identity unavailable in the set (gms_v61 → no-op).
- gain amount: 0 monsters → no emit; N monsters → single emit of `102×N` with
  `Cap 10000`; already charged → registry-level no-op asserted in atlas-buffs
  tests, and the channel still emits (documented, intentional).
- `energyBlastPermitted`: not-blast → true; mirror 15000 → true; mirror 4998 →
  false + re-announce called; mirror absent → true, no re-announce.
- every dep forced to fail → no error escapes, one Error log each (AC-13).
- FR-7: an `AttackTypeEnergy` attack whose skill id is Energy Charge emits a
  gain and calls no effect-application path (AC-12).

atlas-buffs: upsert creates with `NoExpiry` and `min(Amount, Cap)`; emits
`APPLIED` not `STAT_UPDATED`; `CreateIfMissing=false` preserves today's
missing-buff no-op (Combo regression).

atlas-packet: byte fixtures for the `ENERGY_CHARGE` base block, v83 (15-byte,
bool-prefixed time) and v61 (14-byte, narrow), asserting `nOption` = bar value
and `rOption` = skill id. This is the AC-9 artifact; if any in-scope version's
cell is not already ✅ for `GIVE_BUFF`/`GIVE_FOREIGN_BUFF`/`CANCEL_BUFF`, record
it as out of scope with the reason rather than claiming it.

atlas-effective-stats: `ENERGY_CHARGE` at 15000 → `weapon_attack += pad`;
at 4998 → no bonus; at 15000 with `pad = 0` (levels 1–3) → no bonus.

Live pass (v83 tenant, Marauder): bar fills 102/mob, charges at 10000, aura
visible to a second client in the map, expires and the bar visibly zeroes,
effective-stats weapon attack rises and falls with the window, Energy Blast is
refused below full and accepted at full without consuming the bar.

---

## 8. Risks

1. **The v83 evidence is one client.** The bar-reads-`nOption` finding is
   verified on GMS v83 only. v61's narrower base block shares the field order
   (`sub_66E9B6`, task-167) so the same fix applies, but the *reader* was not
   re-derived per version. Mitigation: the packet fixture is written per
   version, and any version that cannot be verified is recorded as out of scope
   (AC-9), never assumed.
2. **`CreateIfMissing` widens a shared contract.** Two services and one
   duplicated struct. Mitigation: default-false, existing callers unchanged, and
   a regression test that a `false` upsert still no-ops.
3. **Charged-window refresh on Kafka redelivery.** Accepted; consistent with
   every other buff emit in the codebase.
4. **No client-side charge gate was found**, so FR-6 remains a deliberate
   server-side divergence. The fail-open + re-announce design bounds the damage
   to "one cast allowed that Cosmic would also have allowed".
