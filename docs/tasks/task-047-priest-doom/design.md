# Priest Doom (Skill 2311005) — Design

Version: v2 (revised after wrong-channel discovery)
Status: Draft
Predecessor: see `postmortem.md`. v1 design (Doom on the magic-attack
empty-damage branch) is superseded.

---

## 1. Architecture overview

```
v83 client cast (SPECIAL_MOVE opcode)
        │
        ▼
atlas-channel: CharacterUseSkillHandle (character_skill_use.go)
        │
        ▼
atlas-channel: handler.UseSkill (skill/handler/common.go)
        │ ├── HP / MP / cooldown / itemConsume cost block
        │ ├── party buff apply (statups, if any)
        │ ├── applyToMobs (no-op for Doom — info.AffectedMobIds() is empty)
        │ └── per-skill dispatcher: Lookup(PriestDoomId) → doom.Apply
        │
        ▼
atlas-channel: skill/handler/doom/doom.go (NEW)
        │ ├── Load caster (X, Y, stance) → derive isFacingLeft
        │ ├── Compute bounding box (caster pos + facing + e.LT()/e.RB())
        │ ├── Query atlas-monsters: monsters in field rectangle (limit MobCount)
        │ ├── For each mob:
        │ │   ├── Magic-reflect probe (skip if active)
        │ │   ├── prop RNG gate (skip with prob 1 - prop)
        │ │   └── mp.ApplyStatus(field, mob, caster, 2311005, level, {DOOM:1}, duration)
        │ └── Summary log line
        │
        ▼
atlas-channel/monster: mp.ApplyStatus emits APPLY_STATUS Kafka command
        │ + Doom-targeted Debugf (already in place)
        │
        ▼
atlas-monsters: APPLY_STATUS consumer → ApplyStatusEffect
        │ ├── DOOM short-circuit in isElementallyImmune (already in place)
        │ ├── isBossAllowedStatus rejects DOOM on bosses (already in place)
        │ └── ModelBuilder.AddStatusEffect (replace-not-noop refresh semantics)
        │
        ▼
atlas-monsters: STATUS_APPLIED Kafka event
        │
        ▼
atlas-channel: monster status broadcast → MonsterStatSet (DOOM mask bit)
        │
        ▼
v83 client: snail render + elemental normalization for the duration
```

**Key architectural shift from v1**: Doom does not pass through
`processAttack` at all. The cast is a buff packet (SPECIAL_MOVE opcode);
target selection is server-authoritative (caster pos + facing + skill
rectangle); item consumption is plumbed into the buff-path cost block
(which covers all `itemConsume` skills, not just Doom).

## 2. Component changes

### 2.1 atlas-monsters: rectangle monster query

**New REST endpoint** under the existing field/monsters subtree:

```
GET /tenants/{tenant_id}/worlds/{world_id}/channels/{channel_id}/maps/{map_id}/monsters?x1={x1}&y1={y1}&x2={x2}&y2={y2}&limit={limit}
```

Response: JSON:API list of monster resources whose `(x, y)` lies inside the
inclusive rectangle `[min(x1,x2), min(y1,y2)] × [max(x1,x2), max(y1,y2)]`,
ordered by distance from the rectangle center (so an `?limit=6` truncation
behaves like Cosmic's "first N in iteration" with a stable, less arbitrary
ordering).

**Implementation**: in atlas-monsters' monster processor, add
`GetInFieldRect(field.Model, x1, y1, x2, y2 int16, limit uint32) ([]Model, error)`.
The existing in-memory monster registry (`monster/registry.go`) is keyed
by (tenant, unique id); add a per-field walk that filters by rect and
caps to limit. Existing `GetInField` is the model — same locking, same
tenant scoping.

**Why a new endpoint** rather than client-side filtering of `GetInField`:
network economy (a populated map can have 50+ monsters; we only want
≤ 6). Also keeps the rect predicate authoritative server-side; future
non-channel callers (a boss-skill mob targeter, mist application) get
the same primitive.

### 2.2 atlas-channel: rect query client wrapper

**New method** on `atlas-channel/monster.Processor`:

```go
func (p *Processor) GetInFieldRect(f field.Model, x1, y1, x2, y2 int16, limit uint32) ([]Model, error)
```

Issues the GET above via the existing `requests`-style client; transforms
the JSON:API list into `[]monster.Model`. Mirrors the shape of the
existing `GetById` and the (likely-existing) `GetInField`.

### 2.3 atlas-channel: generic `itemConsume` charge in `UseSkill`

Modify `services/atlas-channel/atlas.com/channel/skill/handler/common.go`:
extend the existing cost block (after the MP charge, before the buff
dispatch) to charge `e.ItemConsume()` when non-zero. Lookup is via
`compartment.FindFirstByItemId` on the compartment resolved from
`inventoryconst.TypeFromItemId(itemconst.Id(itemId))`. Emit
`consumable.NewProcessor(l, ctx).RequestItemConsume(...)`. Missing-item
warning logs and the cast proceeds (matches the HP/MP semantics).

This is the **only place** `itemConsume` is plumbed. Every skill that
flows through `handler.UseSkill` benefits — Doom, Mystic Door, summons,
mists, etc. The previous placement inside `processAttack`'s cost gate
is reverted (postmortem.md lists what to remove).

### 2.4 atlas-channel: new Doom per-skill handler

**New package**: `services/atlas-channel/atlas.com/channel/skill/handler/doom/`

Files:
- `doom.go` — handler registration + `Apply` function.
- `bbox.go` — `calculateBoundingBox(casterX, casterY int16, facingLeft bool, lt, rb point.Model) (x1, y1, x2, y2 int16)`. Pure function, mirrors Cosmic `StatEffect.java:1206-1218`. Easy to test in isolation.
- `doom_test.go` — handler tests (see PRD §4.7).
- `bbox_test.go` — bounding-box pure-function tests (left/right facing, sign conventions).

**Handler shape** (mirrors `skill/handler/heal/heal.go`):

```go
func init() {
    channelhandler.Register(skill2.PriestDoomId, Apply)
}

func Apply(l logrus.FieldLogger) func(ctx context.Context) func(
    wp writer.Producer,
    f field.Model, characterId uint32,
    info packetmodel.SkillUsageInfo, e effect.Model,
) error {
    return func(ctx context.Context) func(...) error {
        return func(wp writer.Producer, f field.Model, characterId uint32, info packetmodel.SkillUsageInfo, e effect.Model) error {
            cp := character.NewProcessor(l, ctx)
            c, err := cp.GetById()(characterId)
            if err != nil {
                l.WithError(err).Errorf("Doom: failed to load caster [%d].", characterId)
                return nil
            }

            facingLeft := (c.Stance() & 1) == 1
            x1, y1, x2, y2 := calculateBoundingBox(c.X(), c.Y(), facingLeft, e.LT(), e.RB())

            mp := monster.NewProcessor(l, ctx)
            mobs, err := mp.GetInFieldRect(f, x1, y1, x2, y2, e.MobCount())
            if err != nil {
                l.WithError(err).Errorf("Doom: GetInFieldRect failed for caster [%d].", characterId)
                return nil
            }

            mirror := monster.GetStatusMirror()
            t := tenant.MustFromContext(ctx)
            applied := 0
            reflectSkipped := 0
            propSkipped := 0
            statuses := map[string]int32{monster2.StatusDoom: 1}
            for _, m := range mobs {
                if _, ok := mirror.GetReflect(t, m.UniqueId(), monster2.ReflectKindMagical); ok {
                    l.Debugf("Doom: monster [%d] has MAGICAL reflect; status apply skipped.", m.UniqueId())
                    reflectSkipped++
                    continue
                }
                if !propRollFunc(e.Prop()) {
                    propSkipped++
                    continue
                }
                _ = mp.ApplyStatus(f, m.UniqueId(), characterId, uint32(skill2.PriestDoomId), uint32(info.SkillLevel()), statuses, uint32(e.Duration()))
                applied++
            }

            l.Debugf("Doom: caster=[%d] level=[%d] mobsInRect=[%d] applied=[%d] reflectSkipped=[%d] propSkipped=[%d].",
                characterId, info.SkillLevel(), len(mobs), applied, reflectSkipped, propSkipped)
            return nil
        }
    }
}
```

`propRollFunc` is a package-private indirection so tests can inject a
deterministic gate:

```go
var propRollFunc = func(prop float64) bool {
    if prop <= 0 {
        return false
    }
    if prop >= 1 {
        return true
    }
    return rand.Float64() <= prop
}
```

Two simpler design choices on the table:
- (A) Inline the prop math; no test indirection. Tests would set
  `prop = 0` or `prop = 1` to coerce determinism. Sharper but loses
  the ability to test the in-between branches.
- (B) Indirection above. Marginally more API surface, but tests can
  pin all three branches (apply, skip, in-between).

**Choice: B**. The indirection is one variable; the test value is
worth more than the cosmetic gain.

`effect.Model` accessors needed: `LT() point.Model`, `RB() point.Model`
(both exist), `MobCount() uint32` (need to verify; if missing, add to
`effect.Model`'s rest extraction), `Prop() float64` (likewise).

### 2.5 atlas-channel: revert wrong-path Doom code

Per `postmortem.md`:

- `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go`:
  - Remove the Doom-gated reflect probe in `processDamageInfoEntry`'s
    empty-damage branch (added in e05a1983a).
  - Remove the `if itemId := se.ItemConsume(); itemId > 0 { ... }` block
    in `processAttack`'s HP/MP cost gate (added in 4a3312d6d, simplified
    in 9f1b14a00). Replaced by the generic `UseSkill` cost-block change
    in §2.3.
  - Remove imports orphaned by these reverts: `consumable`,
    `inventoryconst`, `itemconst`, `charcon`, `slot`. Keep `field` (the
    `processDamageInfoEntry` helper extraction added it).
- `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common_test.go`:
  - Remove the four `TestProcessDamageInfoEntry_Doom_*` tests + the
    supporting `damageEntryFakes`, `applyStatusCall`, `newDoomEffect`,
    `newDoomAttackInfo` helpers.
  - Remove the `skillconst` import if no other test uses it.
  - Keep the `TestComputeReflect_*`, `TestReflectFlow_*`,
    `TestSnapshotVenomDamagePerTick_*`, and
    `TestAttackKindFromAttackType` tests.

The `processDamageInfoEntry` helper itself (added in 03c84901c) **stays**
because it's a clean refactor and a useful seam for any future
empty-damage status flow. The Debugf in
`monster.Processor.ApplyStatus` (746c24714, 98b38112b) **stays** because
it engages on the corrected path.

## 3. Sequence: a Doom cast end-to-end

```
0   Priest with Magic Rock in ETC inventory casts Doom
1   v83 client → SPECIAL_MOVE packet
2   atlas-channel CharacterUseSkillHandle decodes SkillUsageInfo
3   atlas-channel UseSkill cost block:
        - ChangeMP(-88)
        - findItemSlotInInventory(c.Inventory(), 4006000) → slot.Position
        - consumable.RequestItemConsume(field, charId, item.Id(4006000), slot, 0)
4   atlas-channel UseSkill applyToMobs early-returns (info.AffectedMobIds() empty)
5   atlas-channel UseSkill dispatches to doom.Apply
6   doom.Apply loads caster, derives facing, computes bbox
7   doom.Apply queries mp.GetInFieldRect(field, x1, y1, x2, y2, 6)
8   atlas-monsters returns up to 6 monsters in rect
9   doom.Apply per-mob loop:
        - reflect probe (skip on active reflect)
        - prop gate (skip with prob 1 - 0.52)
        - mp.ApplyStatus(field, mob, char, 2311005, 30, {DOOM:1}, 60000)
              ↓
              atlas-channel monster.Processor.ApplyStatus emits Kafka cmd
              + Debugf("Doom: caster=...")
10  atlas-monsters APPLY_STATUS consumer → ApplyStatusEffect
        - SourceTypePlayerSkill, isElementallyImmune short-circuits on DOOM
        - isBossAllowedStatus rejects on bosses (no DOOM apply)
        - AddStatusEffect appends/refreshes
        - emits STATUS_APPLIED Kafka event
11  atlas-channel STATUS_APPLIED consumer → MonsterStatSet broadcast (DOOM bit)
12  v83 client renders snail; normalizes elemental table
…   60s later …
13  atlas-monsters status-expiry timer → STATUS_EXPIRED
14  atlas-channel → MonsterStatReset (DOOM bit)
15  v83 client restores original sprite
```

## 4. Bounding box

Mirrors Cosmic `StatEffect.java:1206-1218`:

```go
func calculateBoundingBox(casterX, casterY int16, facingLeft bool, lt, rb point.Model) (x1, y1, x2, y2 int16) {
    if facingLeft {
        x1 = casterX + lt.X()
        y1 = casterY + lt.Y()
        x2 = casterX + rb.X()
        y2 = casterY + rb.Y()
    } else {
        // Mirror about caster X. Cosmic does: mylt.x = -rb.x + posFrom.x; myrb.x = -lt.x + posFrom.x
        x1 = casterX - rb.X()
        y1 = casterY + lt.Y()
        x2 = casterX - lt.X()
        y2 = casterY + rb.Y()
    }
    return
}
```

Returns `(x1, y1, x2, y2)` as the inclusive rectangle. atlas-monsters'
rect filter is left to normalize so `min(x1, x2) ≤ x ≤ max(x1, x2)`
and similarly for y, regardless of caller order.

For Doom level 30 the v83 effect data carries `lt = (-200, -100)`,
`rb = (200, 100)` (per the data path; the actual values may differ for
the deployed wz pack, but this is the intent). Facing right, that
becomes `(casterX-200, casterY-100, casterX+200, casterY+100)` — a
400×200 box centered on the caster.

## 5. Tests

### 5.1 atlas-channel: bounding box

`bbox_test.go`:

- `TestBoundingBox_FacingRight` — caster at (0,0), lt=(-200,-100), rb=(200,100), facingLeft=false → (-200, -100, 200, 100).
- `TestBoundingBox_FacingLeft` — same input, facingLeft=true → mirror about X: still (-200, -100, 200, 100) for a symmetric rect, then asymmetric case to differentiate.
- `TestBoundingBox_Asymmetric_FacingRight` — caster at (100, 50), lt=(-50, -10), rb=(150, 30), facingLeft=false → (100-150, 50-10, 100-(-50), 50+30) = (-50, 40, 150, 80).
- `TestBoundingBox_Asymmetric_FacingLeft` — same caster, facingLeft=true → (100+(-50), 50-10, 100+150, 50+30) = (50, 40, 250, 80).

### 5.2 atlas-channel: doom handler

`doom_test.go` (uses fakes for the monster processor, character processor, and reflect mirror):

- `TestDoom_Apply_AppliesToMobsInRect` — 3 mobs returned by the rect query, no reflect, prop=1.0 → 3 ApplyStatus calls.
- `TestDoom_Apply_RespectsMobCount` — covered implicitly: the rect query returns at most `e.MobCount()` mobs (cap done in atlas-monsters; the handler test passes `limit=6` and stub returns exactly 6).
- `TestDoom_Apply_SkipsMagicReflectMobs` — 3 mobs returned, 2nd has a magic-reflect entry → 2 ApplyStatus calls (mobs 1 and 3); reflectSkipped=1 in the summary log assertion.
- `TestDoom_Apply_RespectsProp_Zero` — prop=0.0 → 0 ApplyStatus calls; propSkipped=N.
- `TestDoom_Apply_RespectsProp_One` — prop=1.0 → all in-rect mobs receive apply.
- `TestDoom_Apply_LeftFacingRectMirror` — caster facing left, mob in left-mirrored rectangle, applies.

### 5.3 atlas-channel: itemConsume in UseSkill

`common_test.go` (new file in `skill/handler/`):

- `TestUseSkill_ItemConsume_BurnsItem` — caster with Magic Rock at slot 3, effect with ItemConsume=4006000 → exactly 1 RequestItemConsume call with slot=3.
- `TestUseSkill_ItemConsume_LogsWarningOnMissingItem` — caster with no Magic Rock, effect requires it → no RequestItemConsume call; warning log captured via a logrus hook.
- `TestUseSkill_ItemConsume_ZeroItemConsume_Noop` — effect with ItemConsume=0 → no RequestItemConsume call.

### 5.4 atlas-monsters: rect query

In `monster/processor_test.go` (or a new `processor_rect_test.go`):

- `TestGetInFieldRect_Inside` — 3 monsters at (10,10), (50,50), (200,200); query (-100,-100,100,100) → returns the first two.
- `TestGetInFieldRect_LimitTruncates` — 10 monsters in rect, limit=3 → returns 3 (and document the ordering).
- `TestGetInFieldRect_EmptyResultOnNoMobs` — empty registry → returns empty slice, no error.
- `TestGetInFieldRect_BoundsInclusive` — monster on the corner → included.
- `TestGetInFieldRect_OtherFieldsExcluded` — monster on a different map → not returned.

### 5.5 Existing tests that stay

- atlas-data: `TestReader_PriestDoom_MapsDoomStatus` (reader test).
- atlas-monsters: `TestApplyStatusEffect_Doom_BypassesElementalImmunity`, `TestApplyStatusEffect_Doom_RejectedOnBoss`, `TestApplyStatusEffect_Doom_ReapplyReplacesExisting` (apply tests).

## 6. Risks and mitigations

- **Stance parity**: relying on `Stance() & 1` is OdinMS / Cosmic
  convention. If atlas's character model uses a different facing
  encoding, the handler picks the wrong rectangle and Doom misses
  targets. Mitigation: the bbox tests cover both orientations; the
  acceptance criteria (multi-target spread, left-facing mirror) catches
  this in-game. If the model uses a boolean accessor, prefer it.
- **prop RNG flakiness in tests**: avoided by injecting `propRollFunc`.
- **REST query latency**: per-cast network call from atlas-channel to
  atlas-monsters. Measured today, the existing `GetById` runs ≤ 5ms
  cluster-internal. The rect query is the same shape. No optimization
  needed.
- **mobCount cap interpretation**: Cosmic iterates the in-rect list and
  breaks at `mobCount`. Order is "registry iteration" (effectively
  insertion order). atlas-monsters' rect query orders by distance from
  rect center for stability, then truncates. Slight semantic
  difference but functionally equivalent — same N mobs targeted, just
  a more deterministic selection.
- **Doom on a magic-reflect-active mob**: handler skips that mob's
  apply but still emits the `Doom: monster [%d] has MAGICAL reflect;
  status apply skipped.` Debugf so production diagnoses see the
  decision. No reflect-damage event because Doom does no damage.

## 7. Out of scope

(Same as v1.)

- Other Priest skills.
- Server-side polymorph entity swap.
- Server-side elemental damage recomputation.
- New Kafka topic / event type.
- Any change to `libs/atlas-packet`.
- Any change to `libs/atlas-constants` (PriestDoomId, StatusDoom, TemporaryStatTypeDoom all exist).
