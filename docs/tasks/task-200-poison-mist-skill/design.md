# Poison Mist (2111003) — Design

Version: v1
Status: Approved
Created: 2026-08-07
PRD: [`prd.md`](prd.md)

---

## 0. Executive summary

The PRD's shape is right — a generic, target/effect-descriptor-driven mist that
`atlas-channel` produces and `atlas-maps` ticks — but two of its load-bearing
assumptions do not survive contact with the data and the client:

1. **FR-1's `dot` / `dotInterval` / `dotTime` WZ nodes do not exist on any
   provisioned version.** Verified below (§2.1). They first appear in v1.17-era
   `Skill.wz`, and even there they live in the `common` formula block that
   atlas-data's reader does not parse. Implementing FR-1 as written yields three
   fields that are zero for every skill on every tenant, and FR-3.4's magnitude /
   per-target-duration / tick-interval would all resolve to `0` — which
   FR-6.1/6.2 would then correctly reject, producing a skill that still does
   nothing. The parse is still implemented (additive, forward-compatible), but
   the mist's parameters are sourced from WZ nodes that actually exist.

2. **`nType` is not what selects the poison-mist rendering — `nSkillID` is.**
   The client's `nType` has exactly one meaningful value on this packet (`3` =
   area-buff *item*), and every other value falls through to a dispatch keyed on
   `nSkillID`, where `2111003` has its own arm. So `nType = 0` is not a guess or
   a placeholder; it is the only correct value (§3).

Everything else in the PRD is implemented as specified.

---

## 1. Component map

```
atlas-channel                       atlas-maps                      atlas-monsters
─────────────                       ──────────                      ──────────────
processAttack (character_attack_common.go)
  └─ attackCastTryApply
       └─ LookupAttackCast(FirePoisonMagicianPoisonMist)
       └─ skill/handler/poisonmist
            emits CREATE ──────►  COMMAND_TOPIC_MIST
                                    └─ mist.Processor.Create
                                         └─ registry.Add (tenant-scoped)
                                         └─ emits MIST_CREATED
kafka/consumer/mist  ◄──────────────── EVENT_TOPIC_MIST
  └─ AffectedAreaCreated → all sessions in field

                                    tasks.MistTick (1 Hz)
                                      ├─ targetKind=CHARACTER → COMMAND_TOPIC_CHARACTER_BUFF (unchanged)
                                      └─ targetKind=MONSTER
                                           ├─ monster.GetInMapRect (REST) ──► atlas-monsters
                                           └─ APPLY_STATUS ──► COMMAND_TOPIC_MONSTER ──► StatusExpirationTask DoT
                                      └─ Expired() → Destroy → MIST_DESTROYED
kafka/consumer/mist  ◄──────────────── EVENT_TOPIC_MIST
  └─ AffectedAreaRemoved → all sessions in field
```

> **Correction (live test, 2026-08-08) — the `GetInMapRect` leg never worked.**
>
> `monster.GetInMapRect` built its URL with a `&limit=%d` query param
> (`atlas-maps/monster/requests.go`), and the server rejects **any** request
> carrying a `limit` param with 400 — `paginate.ParseParams`
> (`libs/atlas-rest/server/paginate/params.go`) enforces the task-117 repo-wide
> rule that paging is expressed only via `page[number]`/`page[size]`. The URL is
> drained through `requests.DrainProvider`, which appends exactly those params,
> so every rect lookup 400'd.
>
> Live on `atlas-pr-1255`, once per second per mist:
> `MistTick: monster rect lookup failed for mist [...]; skipping this mist's
> tick. error: "bad request"` — no monster was ever poisoned. Reproduced
> directly: the same URL returns 200 with the expected monster once `&limit=0`
> is removed.
>
> Reusing this design's rect endpoint hid a **pre-existing** bug of the same
> shape: `atlas-channel/monster/requests.go` has the identical template, so
> `skill/handler/common.go`'s AoE mob-select helper had been 400ing on every
> call too.
>
> Fixed by renaming the cap `limit` → `max` on `/monsters/in-rect`, in both
> client templates and the server's `parseUint32QueryOrDefault`. Renamed rather
> than dropped because atlas-channel genuinely uses the cap for `mobCount`;
> removing it would silently uncap AoE target selection.

> **Correction (live test, 2026-08-08) — the DoT never ticked in the ephemeral
> env, for a reason outside this design.**
>
> With the rect lookup fixed, `atlas-maps` logged
> `MistTick: mist [...] applied [POISON] to 1 of 1 monsters in rect` every ~3s
> and the monsters REST showed the effect landing
> (`{sourceSkillId: 2111003, sourceSkillLevel: 30, statuses: {POISON: 0}}`) —
> but their HP never moved.
>
> `atlas-monsters` registers all sweep tasks, `StatusExpirationTask` among
> them, only inside its leader-election callback (`main.go` `registerSweepTasks`),
> and leader election is on by default. The lease key was
> `atlas:lock:<name>` — no deployment component — while every namespace shares
> one Redis (`REDIS_URL` = `redis.home:6379` in both `atlas-main` and each
> `atlas-pr-NNNN`, no DB separation). `atlas-main`'s pod held
> `atlas:lock:monsters-sweep` (`"Acquired leader for [monsters-sweep]."` +
> `"Initializing status expiration task to run every 1000ms."`); the
> `atlas-pr-1255` pod logged neither and had never registered the task.
>
> Fixed in `libs/atlas-lock` by scoping the key to `ATLAS_ENV`:
> `atlas:lock:<env>:<name>`. Not a task-200 defect — it disabled every
> leader-gated sweep in every ephemeral namespace (`atlas-monsters`' drop
> timers, aggro decay, skill-picker sweep, recovery and hidden reconciliation
> alongside the DoT tick, plus `atlas-rankings`, `atlas-summons`,
> `atlas-doors` and `atlas-world`). Landed here because it is what blocked
> verifying this feature at all.

No new service, no new topic, no new REST endpoint. The four seams that change
are: the skill-handler registry (one new subpackage), the mist command/event
bodies (additive fields), the mist tick (one new branch), and the atlas-maps
monster client (one new method).

---

## 2. Data: what the WZ actually contains

### 2.1 Finding — `dot`/`dotInterval`/`dotTime` are absent (OQ-3, resolved)

Against the v83-era `Skill.wz` corpus:

```
$ grep -l 'name="dot"' Skill.wz/*.img.xml | wc -l
0
$ grep -l 'dotInterval\|dotTime' Skill.wz/*.img.xml
(no output)
```

Zero files. Not "zero for `2111003`" — **zero across the entire skill corpus**.
Against a v1.17 `Skill.wz` the same grep matches 37 files, and `2111003`'s
entry there is:

```xml
<imgdir name="common">
  <int    name="maxLevel"    value="20"/>
  <string name="time"        value="4+3*d(x/10)"/>
  <string name="dot"         value="100+5*x"/>
  <string name="dotTime"     value="4+d(x/10)"/>
  <string name="dotInterval" value="1"/>
  ...
</imgdir>
```

Two independent blockers: the nodes postdate every provisioned version, and even
where they exist they are `common`-block *formula strings*, which atlas-data's
reader does not parse at all (known defect
[[bug_v95_skill_common_formula_nodes_unparsed]] — the reader only walks `level`).

**What `2111003` actually carries per level on the provisioned corpus:**

| level | mpCon | mad | time (s) | prop (%) | lt | rb |
|---|---|---|---|---|---|---|
| 1 | 21 | 32 | 4 | 41 | (-110,-82) | (110,83) |
| 4 | 24 | 38 | 8 | 44 | (-120,-90) | (120,90) |
| 7 | 27 | 44 | 12 | 47 | (-130,-97) | (130,98) |
| 19–21 | 39–41 | 68–72 | 28 | 59–61 | (-170,-127) | (170,128) |
| 30 (max) | 50 | 90 | 40 | 70 | (-200,-150) | (200,150) |

No `dot`, no `dotTime`, no `dotInterval`, no `x`, no `y`.

### 2.2 Decision D1 — source the mist parameters from nodes that exist

| Mist parameter | PRD source | Actual source | Rationale |
|---|---|---|---|
| mist lifetime | `time` | `time` (s→ms at reader, already parsed, `reader.go:195-198`) | unchanged; this is also what the client uses for its own `tEnd` (§3.2) |
| rectangle | `lt`/`rb` | `lt`/`rb` (already parsed, `reader.go:232-236`) | unchanged |
| per-target poison duration | `dotTime` | mist lifetime (`time`) | see D1a |
| tick interval | `dotInterval` | constant `PlayerMistTickIntervalMs = 1000` | see D1b |
| poison magnitude | `dot` | `0` — magnitude is unused for POISON | see D1c |

**D1a — per-target duration = mist lifetime.** With no `dotTime`, the two
defensible values are "the mist's lifetime" (poison persists after the monster
leaves the cloud) and "one tick interval" (poison stops the moment it leaves).
We take the former: it matches the observable v83 behavior of the skill, and it
degrades gracefully — because atlas-monsters *replaces* a same-type effect on
re-apply (§4.3), a monster standing in the cloud simply has its expiry pushed
forward each tick rather than accumulating parallel effects.

**D1b — tick interval is a constant, not WZ.** `PlayerMistTickIntervalMs = 1000`,
declared in the new `poisonmist` package. This is not an invented number: the
monster AREA_POISON producer already hard-codes `TickIntervalMs: 1000`
(`atlas-monsters/monster/processor.go` `buildMistCreateBody`), and
atlas-monsters' `APPLY_STATUS` consumer independently defaults a POISON/VENOM
tick to `1000ms` when the command omits one
(`kafka/consumer/monster/consumer.go:113-122`). 1 Hz is already the de-facto
DoT cadence on both ends of this contract; the constant makes it explicit rather
than relying on the consumer's fallback.

**D1c — magnitude is `0` and that is correct, not a shortcut.** atlas-monsters
does **not** read the `POISON` magnitude out of the statuses map. Its DoT tick
computes

```go
// atlas-monsters/monster/status_task.go — calculatePoisonDamage
divisor := int32(70) - int32(se.SourceSkillLevel())
return m.MaxHp() / uint32(divisor)
```

i.e. `maxHP / (70 - skillLevel)` — the reference v83 poison formula, driven
entirely by `sourceSkillLevel`, which the mist already carries. (`VENOM` is the
status that *does* consume its magnitude; `POISON` does not.) Sending a non-zero
`dot` here would be dead payload. We send `POISON: 0` and the existing formula
does the work. **This resolves OQ-4 without needing to revisit it**: the damage
is neither raw `dot` nor magic-attack-scaled — it is HP-proportional, which is
the behavior the skill is expected to have.

> A consequence worth stating plainly: `dot`, `dotInterval`, and `dotTime` will
> be `0` on every skill on every provisioned tenant after this task. That is the
> honest state of the data, not a defect introduced here. FR-1.5's "report it as
> a data defect" branch is the branch we are on, and this section is the report.

### 2.3 FR-1 is still implemented

`skill/reader.go` gains the three reads (`dot` raw; `dotInterval`, `dotTime`
seconds→ms at the reader, matching the `time` treatment at `reader.go:195-198`),
the effect model and REST model gain the three fields, and `atlas-channel`'s
`effect.Model` gains `Dot()`, `DotInterval()`, `DotTime()`. All default to `0`,
all are additive, no existing consumer changes shape (FR-1.3). Cost is ~40 lines
and it means a future WZ re-ingest that *does* carry the nodes needs no
plumbing change — only a switch in the handler from the D1 defaults to the
parsed values, gated on the parsed value being non-zero.

### 2.4 Deliberate non-goal — `prop`

`2111003` carries `prop` 41–70%, the per-monster poison application chance. We
do **not** apply it. Two reasons: the PRD does not ask for it, and under D1a's
refresh-on-reapply semantics a per-tick roll is close to a no-op (a failed roll
merely skips one expiry extension of an already-active effect), so implementing
it would add a seam and a stochastic behavior for near-zero observable effect.
Recorded here so the omission is a decision rather than an oversight; if in-game
tuning later wants it, the lever is a single roll in the `MONSTER` tick branch.

---

## 3. Client: what the wire values must be (FR-5.1–5.4, OQ-1/OQ-2 resolved)

All addresses below are from live IDBs: `GMS_v95.0_U_DEVM.exe` (PDB-backed) and
`MapleStory_dump.exe` (v83).

### 3.1 `nType` — must be `0` (any value except `3`)

`CAffectedAreaPool::OnAffectedAreaCreated` (v95 `@0x437ec0`) reads the body, then:

```
v4 = Decode4()                    // nType, stored AFFECTEDAREA+0x4  @0x437f12
...
if (v4 == 3)                      //                                 @0x437ff8
    → CItemInfo::GetAreaBuffItem(nSkillID)                           @0x43800c
      tEnd = tStart + 250 * pInfo->tTime                             @0x43822a
      MakeLayer_Fog(...)                                             @0x438255
else
    → CAffectedAreaPool::AffectedAreaAnimationCreated(pa, /*bResetEndTime=*/1)
                                                                     @0x43829f-0x4382be
```

`nType == 3` is the **area-buff-item** arm — it treats `nSkillID` as an *item*
id and looks up `AREABUFFITEM`. A skill mist must never take it.

Everything else routes to `AffectedAreaAnimationCreated` (v95 `@0x4372c0`),
which dispatches **entirely on `nSkillID`** and never reads `nType` again:

| `nSkillID` | v95 arm | behavior |
|---|---|---|
| `130` | `@0x4374a6` | mob AREA_POISON; `nDamage = mobskill.nX`; `MakeLayer_FootHold`; **no `tEnd`** |
| `131` | `@0x43736d` | mob AREA_POISON variant; `nDamage`; `MakeLayer_Fog`; **no `tEnd`** |
| **`2111003`** | **`@0x437515` → `@0x437b40`** | **`SKILLENTRY::GetTileUOL`; `MakeLayer_Fog` `@0x437cd3`; `tEnd = tStart + 1000 * SKILLLEVELDATA::tTime` `@0x437c6b-0x437c9f`** |
| `14111006` | shares the `2111003` arm (`LABEL_16`) | Night Walker Poison Bomb |
| `22161003` | `@0x4378f2` | `GetTileUOL`; `MakeLayer_Fog_OneTile`; `tEnd` set |
| `32121006` | `@0x437579` | own arm; `tEnd` set |

v83 is the same code with `AffectedAreaAnimationCreated` inlined into
`OnAffectedAreaCreated` (`@0x431e30`):

```
nType == 3   @0x431b66  → area-buff-item arm
nSkillID == 130         @0x4321cb   (mob)
nSkillID == 131         @0x43206d   (mob)
nSkillID == 2111003     @0x431d50   → arm @0x431f09:
                                       CSkillInfo::GetSkill               @0x431f09
                                       GetTileUOL                         @0x431f3d
                                       tEnd = tStart + 1000*tTime         @0x43200f
                                       MakeLayer_Fog                      @0x43203e
nSkillID == 4221006     @0x431d56  → Smokescreen, shares the generic skill arm
```

**Decision D2: `nType = 0`.** It is the value the already-verified monster path
sends (`mist.Mist.Type()` defaults to `0`, never set by
`buildMistCreateBody`), it is not `3`, and it is not read anywhere on the
rendering path. There is no per-skill `nType` to discover — the field's only
in-packet semantics is the `== 3` test.

For completeness, the other `nType` values the client *does* test, elsewhere in
the pool (not on this packet's create path):

- `nType == 2` → `IsSmokeAreaByPoint` (v95 `@0x434f40`, `p->nType == 2`), the
  Smokescreen miss-chance area. Reserved; **do not** use for Poison Mist.

> **Correction (live test, 2026-08-08) — D2 was WRONG. `nType = 1`.**
>
> This section's sweep of the pool's `nType` readers **missed
> `CAffectedAreaPool::GetAffectedAreaByPoint`** (v83 `sub_431783` `@0x4317b6`,
> v95 `@0x434cc0`, PDB-named). Its predicate is:
>
> ```c
> if ( !p->nType && tCur - p->tStart >= 0 && PtInRect(&p->rcArea, ptUser) )
>     *dwDiseaseData = p->nSLV | (p->nSkillID << 8);   // a MOB-skill descriptor
> ```
>
> `CUserLocal::Update` calls it every frame for the **local user** (v83
> `@0x94b7ba`) and, on a hit, computes
> `AFFECTEDAREA.nDamage (+0x34) * (100 - resist) / 100` (`@0x94b801`) and
> damages them. So `nType == 0` does not mean "unused default" — it is the
> client's *mob disease cloud* marker, and it is the **only** `nType` test that
> can fire for a mist standing over a player.
>
> `nDamage` compounds it. `CAffectedAreaPool::AffectedAreaAnimationCreated`
> (v95 `@0x4372c0`) writes `pa.p->nDamage = a[nSLV-1].nX` **only** under
> `nSkillID == 130` and `nSkillID == 131`; the `nSkillID == 2111003` arm sets
> `tEnd`/`bEnd` and builds fog layers and never touches `nDamage`. A
> character-owned mist sent as `nType 0` therefore bills the caster whatever was
> left in the freshly-allocated `AFFECTEDAREA`.
>
> Observed live on `atlas-pr-1255` (GMS 83.1, map 240011000): ~0.9 s after every
> cast the client sent `CharacterDamageHandle nAttackIdx [-4] damage [1434803]`,
> clamped by the channel to 999999 — instant death, three casts in a row.
>
> The claim in §2 item 2 that "`nType` has exactly one meaningful value on this
> packet (`3`)" is true of `OnAffectedAreaCreated` in isolation and false of the
> pool as a whole. Read-order analysis of the decoder is not sufficient to
> choose a field's value; the field's *readers* have to be enumerated too.
>
> **Corrected decision: `nType = 1` for a character-owned mist, `0` for a
> monster-owned one**, derived from `ownerType` in `atlas-maps`
> (`mist.AffectedAreaTypeFor`) rather than carried on `COMMAND_TOPIC_MIST` —
> no producer should need the client's value table. Keeping `0` for the mob path
> is required, not cosmetic: it is what makes a mob's cloud apply to players
> standing in it (the pre-task-200 AREA_POISON behaviour).
>
> `1` specifically because the complete set of values the client reads is
> {`0` disease cloud, `2` smoke screen, `3` area-buff item}; the per-skill
> construction path dispatches on `nSkillID` alone and the user/party aura
> lookups (`GetAffectAreaByPoint`, `GetAr01AreaPAD`/`MAD`) ignore `nType`
> entirely. `1` is inert. Note the GMS v95 PDB carries **no enum symbol** for
> `AFFECTEDAREA::nType` (it is a bare `int`), so "1 = user skill" is our name
> for an unnamed value, not a client-attested one — the verified property is
> "not 0, not 2, not 3".

**OQ-2 resolved.** The client's own reaction to `nSkillID == 2111003` is purely
cosmetic: fetch the skill's `tile` UOL, build the fog layers, and compute its
own `tEnd` from its own `Skill.wz` `time`. It applies no damage and no status of
its own — all gameplay effect is server-side. Nothing to suppress or complement.
(The v12 `@0x4167cc` comparison the PRD cites is the same cosmetic dispatch on an
older client.)

### 3.2 Corollary — no server-side lifetime clamp (confirms FR-3.5)

Because `bResetEndTime` is `1` on the create path (v95 `@0x43829f`), the client
computes `tEnd = tStart + 1000 * SKILLLEVELDATA::tTime` from **its own WZ**
(v95 `@0x437c95`, v83 `@0x43200f`). If the server clamped the mist's lifetime,
the cloud would keep rendering after the server stopped ticking it, or vanish
while still ticking. The 60 s `MistDurationCapMs` used by the monster path
(which is safe there — arms 130/131 set no `tEnd` at all and are removal-driven)
**must not** be applied. Confirmed as designed.

### 3.3 `dwOwnerId` — the casting character id; attribution only (FR-5.2)

`OnAffectedAreaCreated` stores it at `AFFECTEDAREA+0x8` (v95 `@0x437fb0`,
v83 `@0x431b16`) and never reads it in the create path. Its only readers are the
*ally-area* queries:

- `IsSmokeAreaByPoint` (`@0x434f40`): gated on `p->nType == 2`; compares
  `dwOwnerID` against the caller and their party list.
- `GetAffectAreaByPoint` (`@0x4350f0`): gated on `p->nSkillID == nSkillID` for a
  caller-supplied skill id; same owner/party comparison.
- `GetAr01AreaPAD` / `GetAr01AreaMAD`: same shape, for booster areas.

A Poison Mist (`nType = 0`, `nSkillID = 2111003`) enters none of them: the first
requires `nType == 2`, and the others are only ever called with the *querying*
skill's own id. **There is no owner-exclusion for Poison Mist** — the owner is
not skipped, and there is no client-side effect keyed on ownership.

On the id-collision question the PRD raises: `dwOwnerId` currently carries a
monster *unique* id on the mob path and will carry a *character* id on the
player path. This is safe because the only comparisons made against it
(above) are (a) unreachable for both mist kinds, and (b) scoped by `nType`/
`nSkillID` first, so a numeric coincidence between a monster unique id and a
character id can never route into an ally check.

**Decision D3: `dwOwnerId` = the casting character id.**

### 3.4 `skillDelay = 0` and `nElemAttr = 0` (FR-5.3, FR-5.4)

`skillDelay` feeds `tStart = get_update_time() + 100 * skillDelay` (v95
`@0x437fa3`, v83 `@0x431b50`), and `CAffectedAreaPool::Update` gates the mist's
first draw on `tStart`. It is a draw delay in units of 100 ms; non-zero hides the
mist. Atlas has no per-mist cast delay to express. **`0`** — unchanged from the
existing const `mistSkillDelay` in `kafka/consumer/mist/consumer.go:92`.

`nElemAttr` is the first trailing `Decode4`, stored raw at `AFFECTEDAREA+0x30`
(v95 `@0x437fd9`, v83 `@0x431b3b`). It is written on the create path and not
read by any `CAffectedAreaPool` member we traced (`AffectedAreaAnimationCreated`,
`Update`, `FindAndDraw`, `GetAffectAreaByPoint`, `IsSmokeAreaByPoint`,
`ShelterUpdate` all key on `nType`/`nSkillID`/`tStart`/`nPhase`). `2111003`
carries `elemAttr = "s"` in WZ, but the client reads that from its own
`Skill.wz` via `GetSkill`, not from this packet. **`0`** — unchanged from
`mistElemAttr`.

`nPhase` (GMS v92+, `AFFECTEDAREA+0x48`, v95 `@0x437fde`): compared for equality
against a caller-supplied phase in `IsSmokeAreaByPoint` / `GetAffectAreaByPoint`
— both unreachable here. **`0`** — unchanged from `mistPhase`.

### 3.5 Consequence for FR-2.4

All three values are `0` for both mist kinds. FR-2.4 asks that `CreatedBody`
carry them rather than the channel hard-coding literals, so the *plumbing* is
built (three additive fields on `CreatedBody`, populated from the `Mist` model,
consumed by `handleMistCreated`), with the atlas-maps side defaulting them to
the same constants and the channel's `mistSkillDelay` / `mistElemAttr` /
`mistPhase` consts retained as the documented rationale for those defaults. The
**wire bytes do not change for any mist**, which is what keeps FR-5.5 true: the
existing `SPAWN_MIST` fixtures assert byte-for-byte encodings and must continue
to pass unmodified.

---

## 4. Architecture

### 4.1 Contract generalization (FR-2)

`atlas-maps/kafka/message/mist/kafka.go`:

```go
const (
    TargetKindCharacter = "CHARACTER"
    TargetKindMonster   = "MONSTER"

    EffectKindDisease        = "DISEASE"
    EffectKindDamageOverTime = "DAMAGE_OVER_TIME"
)

type CreateCommandBody struct {
    ...                              // unchanged
    TargetKind string `json:"targetKind"`   // "" ⇒ CHARACTER
    EffectKind string `json:"effectKind"`   // "" ⇒ DISEASE
}

type CreatedBody struct {
    ...                              // unchanged
    NType      int32 `json:"nType"`
    ElemAttr   int32 `json:"elemAttr"`
    SkillDelay int16 `json:"skillDelay"`
}
```

**Decision D4: `Disease`/`DiseaseValue`/`DiseaseDuration`/`TickIntervalMs` keep
their JSON keys** (FR-2.2's rename escape hatch is declined). They are already
the generic status-name/magnitude/duration/interval quadruple; renaming buys a
better name at the cost of touching a live contract that the monster path
publishes today, for zero behavioral gain. The Go-side field names are left
alone too, so the diff stays reviewable; the `Mist` model's doc comment is
updated to say "status" rather than "disease".

Normalization happens **once**, in `mist.ProcessorImpl.Create`: empty
`TargetKind` → `CHARACTER`, empty `EffectKind` → `DISEASE`. That gives FR-2.3's
byte-for-byte compatibility for the existing `atlas-monsters` producer for free,
and it means the `Mist` model's getters are never empty-valued — the tick branch
can `switch m.TargetKind()` without a nil-ish case. `atlas-monsters`'
`buildMistCreateBody` is updated to set both explicitly (`CHARACTER`/`DISEASE`),
which is a no-op change guarded by its existing tests.

`mist.Mist` gains `targetKind`/`effectKind` (plus the three render fields),
with `Builder.SetKinds(target, effect string)` and
`Builder.SetRender(nType, elemAttr int32, skillDelay int16)` — grouped setters,
matching the existing `SetOwner` / `SetSource` / `SetDisease` style rather than
five single-field setters.

### 4.2 Cast path (FR-3)

New package `atlas-channel/skill/handler/poisonmist`, blank-imported from
`skill/handler/registrations`, registered in `init()` as
`channelhandler.RegisterAttackCast(skill2.FirePoisonMagicianPoisonMist, Apply)`.

> **Correction (live test, 2026-08-08).** This originally read
> `channelhandler.Register(...)` — the use-skill registry — and that was wrong.
> Poison Mist carries `damage: 100`, `attackCount: 1`, `mobCount: 1`,
> `prop: 0.41` in Skill.wz (verified live: `GET /api/data/skills/2111003` on the
> GMS 83.1 tenant), so the client delivers it on the **magic-attack** packet
> (opcode `0x2E`, `CharacterMagicAttackHandle`) and never on USE_SKILL. The
> use-skill registry is read by `processAttack` but only to decide whether to
> skip the generic HP/MP cost block — it never invokes the handler. Net effect
> in-game: the direct magic-attack damage landed, no mist was ever created
> (no `SPAWN_MIST`, no poison DoT), no log line was emitted, and the skill
> additionally cast for free because the cost gate saw it as "registered."
>
> The fix adds a second, opt-in registry — `RegisterAttackCast` /
> `LookupAttackCast` — invoked from `attackCastTryApply` in
> `character_attack_common.go`, in the post-broadcast fire-and-forget region
> beside the projectile and meso-explosion emits. It is deliberately separate
> from the use-skill registry rather than a merge of the two: Heal is genuinely
> dual-packet (magic attack for the undead damage, use-skill for the party
> heal), so a single shared registry read from both dispatch sites would fire
> Heal's handler twice per cast. It also takes the skill id and level directly
> instead of a `packetmodel.SkillUsageInfo`, because `processAttack` holds an
> `AttackInfo` and synthesizing a `SkillUsageInfo` would hand handlers
> zero-valued `AffectedMobIds` / `AffectedPartyMemberBitmap` fields that look
> real.

Dispatch is identity-keyed on both paths: `processAttack` resolves the incoming
wire id via `constants.For(region, major, minor).Skill.Resolve(...)` and calls
`LookupAttackCast(attackId)`. Verified that
`FirePoisonMagicianPoisonMist ↔ 2111003` exists in every provisioned version's
generated map (v12/v48/v61/v72/v79/v83/v84/v87/v92/v95/jms). No raw `2111003`
compare anywhere — which also keeps `tools/skill-job-id-guard.sh` green.

Structure mirrors `mysticdoor` (the closest precedent — a handler that emits an
external command using the caster's position):

```go
var loadCaster = func(l, ctx, characterId) (int16, int16, error)   // seam
var emitCreate = func(l, ctx, body mistKafka.CreateCommandBody) error // seam

func Apply(l) func(ctx) func(wp, f, characterId, info, e) error
```

Order of operations inside `Apply`:

1. Resolve `duration := e.Duration()` (ms), `lt := e.LT()`, `rb := e.RB()`.
2. Validation gates, each logging a **distinct** reason and returning `nil`
   (FR-6.1–6.4, NFR-4):
   - `duration <= 0` → "no lifetime"
   - `PlayerMistTickIntervalMs <= 0` → structurally impossible with a const, so
     this is asserted at package level via a compile-time-adjacent guard rather
     than a runtime branch; the runtime gate instead rejects
     `duration < PlayerMistTickIntervalMs` ("lifetime shorter than one tick"),
     which is the real-world form of FR-6.2's invisible-no-op mist
   - `rb.X() <= lt.X() || rb.Y() <= lt.Y()` → "degenerate rectangle"
   - `duration > MaxPlayerMistDurationMs` → "implausible lifetime"
3. `loadCaster` → on error, log + return `nil` (FR-3.3: no mist, no client error).
4. Build the body and `emitCreate` exactly once (FR-3.6: no dedup, no supersede —
   each cast is a distinct mist with its own uuid).

**Decision D5: `MaxPlayerMistDurationMs = 300_000` (5 min).** Fixed against the
observed data: the largest `time` for `2111003` across the corpus is **40 s** at
level 30 (§2.1), so the ceiling is 7.5× the largest legitimate value — it can
only ever fire on corrupt or mis-scaled data (e.g. a seconds/ms unit inversion,
which would turn 40 into 40 000). It is explicitly *not* the monster path's 60 s
clamp: this rejects, it does not truncate (§3.2).

Body fields, per FR-3.4 and D1:

| field | value |
|---|---|
| `OwnerType` / `OwnerId` | `"CHARACTER"` / `characterId` |
| `TargetKind` / `EffectKind` | `"MONSTER"` / `"DAMAGE_OVER_TIME"` |
| `OriginX/Y` | caster position from `loadCaster` |
| `LtX/LtY/RbX/RbY` | `e.LT()` / `e.RB()` |
| `Disease` / `DiseaseValue` | `"POISON"` / `0` (D1c) |
| `DiseaseDuration` | `duration` (D1a) |
| `TickIntervalMs` | `PlayerMistTickIntervalMs` (D1b) |
| `Duration` | `duration` |
| `SourceSkillId` / `SourceSkillLevel` | `uint32(info.SkillId())` / `uint32(info.SkillLevel())` |

`SourceSkillId` is the **wire** id (`info.SkillId()`, straight off the cast
packet), not the identity — the client compares it against its own WZ (§3.1), so
it must be the value that version binds. This is the one place a wire id is
correct, and the handler comments say so.

The handler runs after cost/cooldown/buff and returns `nil` on every rejection,
so no MP or cooldown rollback path is introduced (FR-3.2, FR-6.5). NFR-6 holds
by construction: rectangle, lifetime, and origin all come from server-side skill
data and the character service — nothing from the cast packet except the skill
id and level, which `UseSkill` has already validated.

### 4.3 Effect path (FR-4)

`tasks/mist_tick.go` `processTenant` gains a branch after the `Expired()` /
`ShouldTick()` gates. Rather than inlining a second body, the per-mist work is
extracted:

```go
switch m.TargetKind() {
case mistKafka.TargetKindMonster:
    r.tickMonsters(tctx, prov, t, m)
default:
    r.tickCharacters(tctx, prov, t, m)   // existing body, moved verbatim
}
r.registry.UpdateLastTick(t, m.Id(), time.Now())
```

`tickCharacters` is a pure extraction — same `charsInField` → `posLookup` →
`Contains` → single-`Emit` batch, same `applyDiseaseCommandProvider`. Behavior
is bit-identical, which is what keeps `mist_tick_test.go` and NFR-5 intact.

`tickMonsters`:

1. Resolve the absolute rect from the mist: `(originX+ltX, originY+ltY)` to
   `(originX+rbX, originY+rbY)` — the same arithmetic `Mist.Contains` already
   uses, factored into a `Mist.Rect() (x1, y1, x2, y2 int16)` getter so the two
   cannot drift.
2. `r.monstersInRect(tctx, m)` — the new injectable seam (FR-4.4), defaulting to
   `monster.NewProcessor(l, tctx).GetInMapRect(m.Field(), x1, y1, x2, y2, 0)`.
   `limit = 0` means "no cap" in the atlas-monsters rect endpoint.
3. On error: log at `Error` with the mist id and `return` — this mist's tick is
   abandoned, the loop continues to the next mist and the next tenant is on its
   own goroutine (FR-4.6, NFR-2). Note the `UpdateLastTick` above still runs, so
   a persistently failing lookup does not spin.
4. One `message.Emit(prov)` batch, one `buf.Put(EnvCommandTopicMonster, ...)`
   per monster (FR-4.5).

The command body mirrors `atlas-channel`'s `ApplyStatusCommandBody` exactly:

```go
type applyStatusBody struct {
    SourceType        string           `json:"sourceType"`        // "PLAYER_SKILL"
    SourceCharacterId uint32           `json:"sourceCharacterId"` // m.OwnerId()
    SourceSkillId     uint32           `json:"sourceSkillId"`
    SourceSkillLevel  uint32           `json:"sourceSkillLevel"`
    Statuses          map[string]int32 `json:"statuses"`          // {"POISON": 0}
    Duration          uint32           `json:"duration"`          // ms
    TickInterval      uint32           `json:"tickInterval"`      // ms
}
```

wrapped in the `COMMAND_TOPIC_MONSTER` envelope with `monsterId` and
`type: "APPLY_STATUS"`, keyed on the monster unique id.

> **Contract hazard, called out deliberately.** `COMMAND_TOPIC_MONSTER` is a
> shared topic: every registered handler unmarshals every message, and a sibling
> command body with a same-named-but-narrower field causes decode errors on
> unrelated handlers ([[bug_monster_command_topic_shared_handler_unmarshal_collision]]
> — `UseSkillCommandBody.SkillId` is a `byte` while `ApplyStatusCommandBody.SourceSkillId`
> is a `uint32`). Reusing the **exact** existing key set, with no added or
> renamed keys, is what avoids re-triggering it. The design does not introduce a
> new command type, and the local struct is a byte-compatible mirror — a test
> asserts the marshalled JSON key set matches atlas-channel's.

`atlas-maps/monster` gains `GetInMapRect(f field.Model, x1, y1, x2, y2 int16, limit uint32) ([]RestModel, error)` +
`inMapRectUrl`, copied from `atlas-channel/monster/requests.go:12`. Its
`RestModel` already carries `X`/`Y`; no model change. `requests.DrainProvider`
is used as with `CountInMap`, so multi-page results are fully drained.

Expiry is untouched (FR-4.7): the `Expired()` → `Destroy(..., ReasonExpired)`
path already emits `MIST_DESTROYED` → `AffectedAreaRemoved`. FR-4.8 (mist
outlives caster departure/death/logout) is satisfied by *not building anything*:
nothing in atlas-maps observes character lifecycle, and `CANCEL` is only sent by
an explicit producer, of which there is none for player mists.

### 4.4 Refresh vs stack (OQ-5, resolved)

Verified in `atlas-monsters/monster/builder.go:137-163`:

```go
// AddStatusEffect adds a status effect, replacing any existing effect with
// overlapping status types. Exception: VENOM stacks up to 3 times.
func (b *ModelBuilder) AddStatusEffect(effect StatusEffect) *ModelBuilder {
    for statusType := range effect.Statuses() {
        if statusType == "VENOM" { /* ...cap at 3, evict earliest... */ }
        else { b.RemoveStatusEffectByType(statusType) }
    }
    b.statusEffects = append(b.statusEffects, effect)
```

`POISON` takes the `else` branch: same-type replace. **Re-applying every tick
refreshes**, exactly as the PRD intends — no reconciliation between tick
interval and `dotTime` is needed, and D1a's "per-target duration = mist
lifetime" is safe (the effect's expiry is simply pushed forward while the
monster stays inside).

One real consequence: each refresh mints a **new** `StatusEffect` with a fresh
`lastTick`, so the DoT tick timer restarts each second. With
`PlayerMistTickIntervalMs == 1000` and atlas-monsters' DoT cadence also at 1 s,
this is a race that can under-count damage ticks. Mitigation is D1b's choice of
exactly 1000 ms plus the `StatusExpirationTask`'s own interval; the acceptance
criterion is behavioral ("monsters lose HP periodically for the mist's
duration"), not a tick-count assertion. If observed damage is visibly starved,
the lever is raising `PlayerMistTickIntervalMs` above the DoT cadence — a
one-constant change, and flagged here so it is a known tuning point rather than
a surprise.

Also inherited, and correct: `ApplyStatusEffect` applies elemental and boss
immunity gates for `SourceTypePlayerSkill` (`processor.go:1374-1395`), so a
poison-immune monster (`IsImmuneToElement("P")`) standing in the cloud takes
nothing. That is desired behavior, obtained for free.

---

## 5. Non-functional

**NFR-1 (tick cost).** One rect query per *active monster-targeting mist* per
second — not per monster, not per player. `2111003`'s lifetime is 4–40 s, so a
single caster sustains at most one in-flight mist for ≤40 s per cast. Worst case
on a busy map: assume 10 FP mages spamming with no cooldown at ~1 cast/s and 40 s
lifetime → ≤400 concurrent mists → 400 rect QPS against atlas-monsters. That is
well inside the endpoint's budget (`atlas-channel` already issues one per AoE
skill cast per attacking player, at higher burst rates). The mitigation lever if
it ever isn't, per NFR-1, is raising `PlayerMistTickIntervalMs` — never dropping
mists.

**NFR-2 (isolation).** Unchanged: `runOnce` already fans out one goroutine per
tenant via `routine.Go` and `WaitGroup`. Per-mist error containment (§4.3 step 3)
means a slow atlas-monsters degrades one mist, then one tenant's serial loop —
never another tenant's. No shared blocking call is introduced.

**NFR-3 (multi-tenancy).** `tickMonsters` receives `tctx` (already
`tenant.WithContext`-decorated at `mist_tick.go:163`) and passes it to both the
REST call and the producer, matching the character branch.

**NFR-4 (observability).**
- creation: one `Info` per accepted mist in the handler (character, level, rect,
  lifetime) and one distinct `Warn` per rejection reason (§4.2 step 2)
- tick: one `Debug` per mist per tick with the monster count applied; the count
  makes "the mist did nothing" a one-log-line diagnosis (0 monsters found vs.
  N found but no damage)
- failure: `Error` on rect-lookup failure and on emit failure, both carrying the
  mist id

**NFR-5 (no regression).** The monster AREA_POISON path changes in exactly one
way: `buildMistCreateBody` sets `TargetKind`/`EffectKind` explicitly. Its
existing tests (`processor_test.go:1162+`) get two added field assertions;
`mist_tick_test.go` is untouched except that its mists now normalize to
`CHARACTER`/`DISEASE` in `Create` — which is the default path, so no test edits
are expected.

---

## 6. Packet work (FR-5.6, FR-5.7)

`SPAWN_MIST × gms_v92` is `❌` (opcode `0x140`) and `REMOVE_MIST × gms_v92` is
`🟡ᶠ` (`0x141`) — `STATUS.md:337,340`. The encoder already models v92 explicitly
(`hasPhase = IsRegion("GMS") && MajorAtLeast(92)`,
`affected_area_created.go:141`), and v92's `OnAffectedAreaCreated` `@0x4392a0` is
documented as identical to v95's. Expectation: **verification pass, not a codec
change.** Both cells go through the standard single-cell procedure
([`VERIFYING_A_PACKET.md`](../../packets/audits/VERIFYING_A_PACKET.md)) — byte
fixture with a `packet-audit:verify` marker, pinned evidence record, regenerated
matrix, all three artifacts committed together.

If the v92 read order turns out to diverge from v95, that is a codec change and
it is in scope for this task — not a follow-up. The v92 IDB is open and named.

`packet-audit matrix --check`, `fname-doc --check`, and `operations --check`
must all exit 0, with the matrix regenerated **after** merging main
([[bug_packet_matrix_toolsha_reads_git_head]]).

No template change is needed: `AffectedAreaCreated` / `AffectedAreaRemoved` are
registered in all eleven seed templates as of `ae3341511` (#1226, task-165).
This is verified, not re-added.

---

## 7. Testing

**atlas-data** — table test over the three new reads: node present, node absent,
node zero; asserts `dotInterval`/`dotTime` are ×1000 at the reader and `dot` is
raw. Plus a REST→`atlas-channel` round-trip test asserting the ms values survive
hydration (FR-1.4).

**atlas-channel `poisonmist`** — against a recording `emitCreate` seam and a stub
`loadCaster`:
- happy path asserts every field of the emitted body (the FR-3.4 table above),
  including `targetKind=MONSTER` / `effectKind=DAMAGE_OVER_TIME` and exactly one
  emitted command
- one test per FR-6.1–6.4 rejection: zero emissions, and the expected distinct
  log reason (asserted via `logrus/hooks/test`)
- caster-load failure: zero emissions, `nil` returned
- `Lookup(skill2.FirePoisonMagicianPoisonMist)` resolves after importing
  `registrations`

**atlas-maps `mist`** — `Create` normalizes empty kinds to `CHARACTER`/`DISEASE`;
`Create` with explicit `MONSTER`/`DAMAGE_OVER_TIME` round-trips onto the model;
`Mist.Rect()` agrees with `Mist.Contains()` on boundary coordinates.

**atlas-maps `tasks`** — against injected monster positions and a recording
producer:
- one `APPLY_STATUS` per monster inside the rect, none for monsters outside
  (the seam returns a rect-query result, so the "outside" case is exercised by
  returning an out-of-rect monster and asserting `Contains`-style filtering is
  *not* re-applied — the endpoint is authoritative; see note below)
- emitted body's JSON key set is byte-identical to atlas-channel's
  `ApplyStatusCommandBody`
- a `targetKind=CHARACTER` mist still emits the old disease body unchanged
- a monster-lookup failure on mist A does not prevent mist B from ticking
- expiry emits `MIST_DESTROYED` exactly once and stops ticking

> On filtering: `tickMonsters` trusts the `in-rect` endpoint and does **not**
> re-filter with `Contains`. Double-filtering would silently mask an endpoint
> bug and would diverge if the two rect conventions (inclusive vs exclusive
> edges) ever differ. One authority per question.

**atlas-monsters** — existing `buildMistCreateBody` tests gain assertions for the
two new fields.

**Verification gates** (CLAUDE.md §Build & Verification): `go test -race`,
`go vet`, `go build` in atlas-data / atlas-channel / atlas-maps / atlas-monsters;
`tools/lint.sh --check`; `tools/redis-key-guard.sh`; `tools/goroutine-guard.sh`;
`tools/buff-duration-guard.sh` (the character branch's `duration` field is
untouched and stays ms); `tools/skill-job-id-guard.sh`. No `go.mod` changes are
expected, so no `docker buildx bake` — if one appears, the bake becomes mandatory.

---

## 8. Deviations from the PRD

| PRD | Design | Why |
|---|---|---|
| FR-1.5 — effect model exposes non-zero `Dot()`/`DotInterval()`/`DotTime()` for `2111003` | They are `0` on every provisioned version | WZ nodes do not exist pre-v1.17 (§2.1). FR-1.5's data-defect branch; §2 is the report. |
| FR-3.4 — magnitude = `Dot()`, per-target duration = `DotTime()`, tick interval = `DotInterval()` | magnitude `0`; duration = mist lifetime; interval = `PlayerMistTickIntervalMs` const | D1a–D1c. POISON damage is `maxHP/(70-level)` server-side; the magnitude field is unread. |
| OQ-4 — "raw `dot` vs magic-attack-scaled, revisit if wrong" | Neither; HP-proportional via the existing formula | Resolved, not deferred (§2.2 D1c). |
| FR-2.2 — renaming the disease fields permitted | Declined; JSON keys and Go names retained | Live contract, zero behavioral gain (§4.1 D4). |
| — | `prop` (41–70% apply chance) not implemented | Not in the PRD; near-no-op under refresh semantics (§2.4). Recorded as a decision. |

---

## 9. Resolved open questions

| | Resolution |
|---|---|
| **OQ-1** `nType` value and whether it drives rendering / owner-exclusion | `0`. Its only in-packet semantics is `== 3` (area-buff item); rendering is dispatched on `nSkillID`, and owner-exclusion does not exist for this mist. §3.1, §3.3 |
| **OQ-2** Client-side effect for `nSkillID == 2111003` | Cosmetic only — tile UOL, fog layers, and its own `tEnd` from its own WZ. Nothing to suppress. §3.1 |
| **OQ-3** Are `dot`/`dotInterval`/`dotTime` populated per version | No — absent from the entire pre-v1.17 corpus, and `common`-block formulas where they exist. §2.1 |
| **OQ-4** Raw `dot` vs magic-attack-scaled magnitude | Neither; `maxHP/(70-skillLevel)`, already implemented in atlas-monsters. §2.2 D1c |
| **OQ-5** Does re-applying POISON stack or refresh | Refresh — `AddStatusEffect` replaces same-type effects (VENOM is the only stacking exception). §4.4 |
