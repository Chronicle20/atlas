# atlas-effective-stats: equipment stat requirements — Design

Version: v1
Status: Approved
Created: 2026-05-03
Inputs: `docs/tasks/task-053-equip-stat-requirements/prd.md`

---

## 1. Architectural Decisions

The following choices were settled during the design phase. The PRD's §9 Open Questions are resolved here.

| # | Decision | Rationale |
|---|----------|-----------|
| 1 | The equipped-asset snapshot map is the **source of truth for equipment**; `Model.bonuses[]` carries only `buff:*` and `passive:*` entries. | Cleaner separation of concerns. Eliminates the dual-source-of-truth risk of also writing equipment entries into `bonuses[]`. The compute path becomes "iterate qualifying snapshots + iterate `bonuses[]`." |
| 2 | Snapshot shape is `[]stat.Bonus` (pre-extracted), not raw `EquipableRestData`. | Extraction logic was already duplicated between `initializer.go` and `asset/consumer.go`. Pre-extracting denormalises that into one place at write time and keeps the read path arithmetic-only. |
| 3 | Equipment-template requirement cache is a **package-level singleton** in `external/data/equipment/`, initialised via `sync.Once`, keyed by `(tenant.Id, templateId)` behind a `sync.RWMutex`. | Matches the existing project registry/cache convention. Tenant scoping handles the version-variation case (GMS/83 vs GMS/95 etc.). |
| 4 | Fixed-point qualification logic lives on `character.Model` as a pure method that accepts a `Provider` function, not in `initializer.go` or `processor.go`. | Pure methods on the immutable model are unit-testable without I/O stubs. The `Provider` injects the cache lookup. |
| 5 | Re-evaluation is triggered through a **single new entry point**: `Processor.RecomputeEquipmentBonuses(ch, characterId)`. All three call sites (STAT_CHANGED consumer, asset MOVED, asset DELETED) invoke it after their own state mutations. `SetBaseStats` and `SetWearerProfile` fold the call into themselves so the consumer never has to remember both. | Single-responsibility entry point, mirrors the existing `Add/RemoveBonusesBySource` pattern. |
| 6 | Wearer level + jobId go into a **new `WearerProfile` struct** held alongside `baseStats` in `character.Model`. `stat.Base` is **not** extended (it remains the six numeric stats only). | `stat.Base` is shared with the computed-stats math; widening it bloats both paths. A sibling struct keeps concerns separate. |
| 7 | STAT_CHANGED events with `TypeLevel` or `TypeJob` (which carry `values=nil` per atlas-character producer) are handled in a **dedicated branch** that re-fetches the wearer record from atlas-character and calls `SetWearerProfile`. The numeric-stat branch keeps using `mergeBaseStats`. | The two paths have fundamentally different inputs (event values vs REST refetch); branching is clearer than overloading `mergeBaseStats`. |
| 8 | A 2-equip cycle where A and B mutually require each other's bonuses converges to "neither qualifies" by construction (the iteration starts with empty `qualified` and only adds items that pass under the prior round's stats — self-bootstrapping is impossible). | Matches v83 client behaviour. No special-case code needed; locked in by a unit test. |
| 9 | On atlas-data fetch error with cold cache, the asset is **dropped from the qualifying set for the current evaluation** (WARN log). No retry is scheduled; the next re-evaluation trigger naturally retries. | The cache only stores successful fetches, so subsequent attempts re-hit atlas-data. Active retry would add complexity for no behavioural gain. |
| 10 | Tenant scoping uses `map[tenant.Id]map[uint32]EquipmentRequirements` behind a single `sync.RWMutex`. | Standard Go pattern. Memory bounded by the per-tenant template-id population (~5–10K entries per tenant). |

## 2. Data Model

### 2.1 `character.Model` extensions

```go
type Model struct {
    // existing fields, unchanged
    tenant      tenant.Model
    ch          channel.Model
    characterId uint32
    baseStats   stat.Base                  // STR/DEX/INT/LUK + MaxHp/MaxMp; unchanged
    bonuses     []stat.Bonus               // CHANGED: now holds only buff:* and passive:* entries
    computed    stat.Computed
    lastUpdated time.Time
    initialized bool

    // NEW
    wearer            WearerProfile           // Level + JobId; inputs to reqLevel / reqJob
    equipped          map[uint32]EquippedAsset // assetId -> snapshot; source of truth for equipment
    qualifiedSnapshot map[uint32]bool         // cached qualifying-asset set, populated by RecomputeWith
}

type WearerProfile struct {
    level byte
    jobId job.Id
}

func (p WearerProfile) Level() byte    { return p.level }
func (p WearerProfile) JobId() job.Id  { return p.jobId }
func NewWearerProfile(level byte, jobId job.Id) WearerProfile

type EquippedAsset struct {
    assetId    uint32
    templateId uint32
    bonuses    []stat.Bonus  // pre-extracted, sourced as "equipment:<assetId>"
}

func (a EquippedAsset) AssetId() uint32         { return a.assetId }
func (a EquippedAsset) TemplateId() uint32      { return a.templateId }
func (a EquippedAsset) Bonuses() []stat.Bonus   { /* defensive copy */ }
func NewEquippedAsset(assetId, templateId uint32, bonuses []stat.Bonus) EquippedAsset
```

Builder methods follow the existing immutable-builder pattern:

```go
func (m Model) WithWearer(p WearerProfile) Model
func (m Model) WithEquippedAsset(a EquippedAsset) Model        // overwrite by assetId
func (m Model) WithoutEquippedAsset(assetId uint32) Model
```

JSON marshal/unmarshal additions:

```go
type modelJSON struct {
    // existing fields...
    Wearer   WearerProfileJSON           `json:"wearer"`
    Equipped map[uint32]EquippedAssetJSON `json:"equipped"`
}
```

so the registry serialises losslessly. `qualifiedSnapshot` is **not** serialised (it's a derived cache, recomputed on load).

### 2.2 `EquipmentRequirements`

```go
// In external/data/equipment/cache.go (or a sibling file).
type EquipmentRequirements struct {
    ReqLevel byte
    ReqJob   uint16   // v83 bitmask; 0 = no restriction
    ReqStr   uint16
    ReqDex   uint16
    ReqInt   uint16
    ReqLuk   uint16
}
```

### 2.3 `Provider`

```go
// In external/data/equipment/cache.go.
type Provider func(ctx context.Context, templateId uint32) (EquipmentRequirements, bool)
```

Returns `(_, false)` when the cold-cache fetch failed. The caller treats `false` as "this asset does not qualify" and the package logs WARN at the failure site.

### 2.4 No persistence changes

atlas-effective-stats remains process-local. The new fields and caches share the existing in-memory volatility (rebuild on lazy-init).

## 3. Equipment-Data Client (`external/data/equipment/`)

Three new files modeled on `external/data/skill/`:

### 3.1 `rest.go`

```go
package equipment

type RestModel struct {
    Id       uint32 `json:"-"`
    ReqLevel byte   `json:"reqLevel"`
    ReqJob   uint16 `json:"reqJob"`
    ReqStr   uint16 `json:"reqStr"`
    ReqDex   uint16 `json:"reqDex"`
    ReqInt   uint16 `json:"reqInt"`
    ReqLuk   uint16 `json:"reqLuk"`
}

func (r RestModel) GetName() string  { return "equipment" }
func (r RestModel) GetID() string    { /* itoa(Id) */ }
func (r *RestModel) SetID(id string) error { /* parse */ }
```

Only the six requirement fields are mirrored. atlas-data's full equipment response carries many more fields, but atlas-effective-stats only needs requirements (the per-asset stats already arrive via atlas-inventory).

### 3.2 `requests.go`

```go
package equipment

const (
    Resource     = "data/equipment"
    EquipmentById = Resource + "/%d"
)

func getBaseRequest() string { return requests.RootUrl("DATA") }

func RequestById(templateId uint32) requests.Request[RestModel] {
    return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+"/"+EquipmentById, templateId))
}
```

Tenant header propagation comes for free via the requests decorator chain.

### 3.3 `cache.go`

```go
package equipment

type cache struct {
    mu    sync.RWMutex
    store map[tenant.Id]map[uint32]EquipmentRequirements
}

var (
    once   sync.Once
    inst   *cache
)

func getCache() *cache {
    once.Do(func() { inst = &cache{store: map[tenant.Id]map[uint32]EquipmentRequirements{}} })
    return inst
}

// GetProvider returns a Provider closure bound to the given logger.
// On cache hit: returns the cached value.
// On cache miss: fetches from atlas-data; caches success; returns (_, false) on failure with a WARN log.
func GetProvider(l logrus.FieldLogger) Provider {
    return func(ctx context.Context, templateId uint32) (EquipmentRequirements, bool) {
        t := tenant.MustFromContext(ctx)
        if r, ok := getCache().get(t.Id(), templateId); ok {
            return r, true
        }
        rm, err := RequestById(templateId)(l, ctx)
        if err != nil {
            l.WithError(err).Warnf("equipment template [%d] fetch failed; treating dependent assets as unqualified", templateId)
            return EquipmentRequirements{}, false
        }
        r := EquipmentRequirements{ReqLevel: rm.ReqLevel, ReqJob: rm.ReqJob, ReqStr: rm.ReqStr, ReqDex: rm.ReqDex, ReqInt: rm.ReqInt, ReqLuk: rm.ReqLuk}
        getCache().put(t.Id(), templateId, r)
        return r, true
    }
}
```

## 4. Qualification Engine (pure, on `Model`)

### 4.1 `meetsRequirements`

```go
// AppliedStats is the wearer's per-evaluation snapshot used to test
// requirements. It is the sum of base stats + always-on (buff/passive)
// bonuses + bonuses from the currently-qualifying equipment subset.
type AppliedStats struct {
    Strength     uint32
    Dexterity    uint32
    Intelligence uint32
    Luck         uint32
}

func meetsRequirements(r EquipmentRequirements, s AppliedStats, level byte, jobId job.Id) bool {
    if r.ReqLevel > 0 && level < r.ReqLevel       { return false }
    if r.ReqJob   > 0 && uint16(jobId) & r.ReqJob == 0 { return false }
    if r.ReqStr   > 0 && s.Strength     < uint32(r.ReqStr) { return false }
    if r.ReqDex   > 0 && s.Dexterity    < uint32(r.ReqDex) { return false }
    if r.ReqInt   > 0 && s.Intelligence < uint32(r.ReqInt) { return false }
    if r.ReqLuk   > 0 && s.Luck         < uint32(r.ReqLuk) { return false }
    return true
}
```

Notes:
- `r.ReqLevel == 0` means "no level requirement" (per v83 WZ data convention).
- `r.ReqJob == 0` means "no class restriction"; otherwise it is a bitmask (Beginner=0, Warrior=1, Magician=2, Bowman=4, Thief=8, Pirate=16, OR'd for cross-class items). The Beginner=0 convention means a Beginner-only item would have to be modelled differently if it ever existed in v83 (it doesn't — Beginners can equip anything restricted to level 0 with no class flag).
- The level / job comparison helpers do not exist on the actual jobId today — `jobId` is converted to its underlying uint16 to mask against `ReqJob`.

### 4.2 `Model.QualifiedEquipment`

```go
func (m Model) QualifiedEquipment(reqProvider Provider, ctx context.Context) map[uint32]bool {
    qualified := make(map[uint32]bool, len(m.equipped))
    if len(m.equipped) == 0 {
        return qualified
    }

    // Always-on baseline = base + sum(non-equipment bonuses).
    //   bonuses[] contains only buff:* and passive:* by invariant.
    flatNonEquip := sumFlatBonuses(m.bonuses) // map[stat.Type]int32

    // computeApplied returns AppliedStats = base + non-equipment flat
    // contributions + flat contributions from the equipment subset in `qualified`.
    computeApplied := func(qualified map[uint32]bool) AppliedStats {
        s := AppliedStats{
            Strength:     uint32(m.baseStats.Strength())     + uint32max0(flatNonEquip[stat.TypeStrength]),
            Dexterity:    uint32(m.baseStats.Dexterity())    + uint32max0(flatNonEquip[stat.TypeDexterity]),
            Intelligence: uint32(m.baseStats.Intelligence()) + uint32max0(flatNonEquip[stat.TypeIntelligence]),
            Luck:         uint32(m.baseStats.Luck())         + uint32max0(flatNonEquip[stat.TypeLuck]),
        }
        for assetId, snap := range m.equipped {
            if !qualified[assetId] { continue }
            for _, b := range snap.bonuses {
                switch b.StatType() {
                case stat.TypeStrength:     s.Strength     += uint32max0(b.Amount())
                case stat.TypeDexterity:    s.Dexterity    += uint32max0(b.Amount())
                case stat.TypeIntelligence: s.Intelligence += uint32max0(b.Amount())
                case stat.TypeLuck:         s.Luck         += uint32max0(b.Amount())
                }
            }
        }
        return s
    }

    for {
        applied := computeApplied(qualified)
        added := false
        for assetId, snap := range m.equipped {
            if qualified[assetId] {
                continue
            }
            req, ok := reqProvider(ctx, snap.templateId)
            if !ok {
                continue // permanent disqualification for this evaluation
            }
            if meetsRequirements(req, applied, m.wearer.level, m.wearer.jobId) {
                qualified[assetId] = true
                added = true
            }
        }
        if !added {
            return qualified
        }
    }
}
```

Properties:
- **Monotonic.** Each round can only grow `qualified` because adding items only adds positive flat bonuses to `applied`, and every requirement is a `>=` test.
- **Bounded.** Convergence in at most `len(m.equipped)` iterations (typically ≤12 in v83). Worst case `O(n²)` requirement checks with constant-time provider lookup after warm-up.
- **No self-bootstrap.** An asset's own bonuses cannot help it qualify, because it is only added to `qualified` after the round in which `applied` excluded it. A 2-equip mutual-dependency cycle therefore stays empty.

### 4.3 `ComputeEffectiveStats` reshape

```go
func (m Model) ComputeEffectiveStats(reqProvider Provider, ctx context.Context) (stat.Computed, map[uint32]bool) {
    qualified := m.QualifiedEquipment(reqProvider, ctx)
    flat       := initFlatFromBase(m.baseStats)         // map[stat.Type]int32
    multipliers := initEmptyMultipliers()               // map[stat.Type]float64

    // Buff + passive contributions (also the only multiplier source).
    for _, b := range m.bonuses {
        flat[b.StatType()]        += b.Amount()
        multipliers[b.StatType()] += b.Multiplier()
    }
    // Equipment contributions, only for the qualifying subset.
    for assetId, snap := range m.equipped {
        if !qualified[assetId] { continue }
        for _, b := range snap.bonuses {
            flat[b.StatType()] += b.Amount()
        }
    }
    // Existing computeEffective(...) closure runs as before.
    return buildComputed(flat, multipliers), qualified
}
```

`Recompute` becomes:

```go
func (m Model) RecomputeWith(reqProvider Provider, ctx context.Context) Model {
    computed, qualified := m.ComputeEffectiveStats(reqProvider, ctx)
    next := m.WithComputed(computed)
    next.qualifiedSnapshot = qualified
    return next
}
```

The `qualifiedSnapshot` cache lets `Bonuses()` reuse the already-computed qualifying set (Section 5) without re-running the iteration on every read.

## 5. REST Surface — `Bonuses()`

The wire shape stays identical (per PRD §5). `Bonuses()` reconstructs the full list by merging the qualifying equipment snapshots with the buff/passive entries already in `m.bonuses`.

```go
func (m Model) Bonuses() []stat.Bonus {
    out := make([]stat.Bonus, 0, len(m.bonuses)+len(m.equipped)*4)
    out = append(out, m.bonuses...)
    for assetId, snap := range m.equipped {
        if m.qualifiedSnapshot[assetId] {
            out = append(out, snap.bonuses...)
        }
    }
    return out
}
```

`Bonuses()` does **not** take the provider — it relies on `qualifiedSnapshot` having been populated by the most recent `RecomputeWith`. Every state mutation funnels through `RecomputeWith` (via `RecomputeEquipmentBonuses`, Section 6), so the cache is always current when REST reads it.

The REST handler in `character/resource.go` is therefore unchanged in signature: it continues to call `model.Bonuses()` and `model.Computed()` directly.

## 6. Processor

### 6.1 New / changed methods

```go
type Processor interface {
    // existing methods unchanged...

    // NEW
    SetWearerProfile(ch channel.Model, characterId uint32, p WearerProfile) error
    RecomputeEquipmentBonuses(ch channel.Model, characterId uint32) error

    // CHANGED signatures
    AddEquipmentBonuses(ch channel.Model, characterId, equipmentId, templateId uint32, bonuses []stat.Bonus) error
    RemoveEquipmentBonuses(characterId, equipmentId uint32) error
    // SetBaseStats unchanged in signature; folds in RecomputeEquipmentBonuses internally
    SetBaseStats(ch channel.Model, characterId uint32, base stat.Base) error
}
```

### 6.2 Re-evaluation flow

```go
func (p *ProcessorImpl) RecomputeEquipmentBonuses(ch channel.Model, characterId uint32) error {
    oldModel, err := GetRegistry().Get(p.ctx, characterId)
    if err != nil { return err }
    oldComputed := oldModel.Computed()

    newModel := oldModel.RecomputeWith(equipment.GetProvider(p.l), p.ctx)
    GetRegistry().Update(p.ctx, newModel)

    p.checkAndPublishClampCommands(newModel, oldComputed, newModel.Computed())
    return nil
}
```

The diff-and-clamp helper is the existing `checkAndPublishClampCommands` — re-gating that drops a `+MaxHp` item out must publish `CLAMP_HP` so atlas-character snaps current HP down to the new cap.

### 6.3 `AddEquipmentBonuses` reshape

The registry helpers return the updated Model so the processor does not need a separate `Get` round-trip:

```go
func (p *ProcessorImpl) AddEquipmentBonuses(ch channel.Model, characterId, equipmentId, templateId uint32, bonuses []stat.Bonus) error {
    snap := character.NewEquippedAsset(equipmentId, templateId, bonuses)
    _ = GetRegistry().PutEquippedAsset(p.ctx, ch, characterId, snap)
    return p.RecomputeEquipmentBonuses(ch, characterId)
}

func (p *ProcessorImpl) RemoveEquipmentBonuses(characterId, equipmentId uint32) error {
    m, err := GetRegistry().RemoveEquippedAsset(p.ctx, characterId, equipmentId)
    if err != nil {
        // No model yet for this character — nothing to recompute.
        return nil
    }
    return p.RecomputeEquipmentBonuses(m.Channel(), characterId)
}
```

`RemoveEquippedAsset` returns `(Model, error)`; the error is non-nil only when the character is not in the registry, in which case there is nothing to re-gate.

`SetBaseStats` and `SetWearerProfile` likewise call `RecomputeEquipmentBonuses` after writing the new state, so a single consumer-side call is enough to re-gate.

### 6.4 Registry helpers

```go
// In character/registry.go
func (r *Registry) PutEquippedAsset(ctx context.Context, ch channel.Model, characterId uint32, a EquippedAsset) Model
func (r *Registry) RemoveEquippedAsset(ctx context.Context, characterId uint32, assetId uint32) (Model, error)
func (r *Registry) SetWearerProfile(ctx context.Context, ch channel.Model, characterId uint32, p WearerProfile) Model
```

`PutEquippedAsset` and `SetWearerProfile` mirror the existing `AddBonus` shape — they get-or-create and always succeed. `RemoveEquippedAsset` returns an error when the character is not in the registry (mirroring `RemoveBonus`).

These mirror the existing `AddBonus` / `RemoveBonusesBySource` pattern (atomic copy-on-write with the registry mutex).

## 7. Consumer Changes

### 7.1 `kafka/consumer/character/consumer.go::handleStatChanged`

```go
func handleStatChanged(l logrus.FieldLogger, ctx context.Context, e ...) {
    if e.Type != character2.StatusEventTypeStatChanged { return }

    relevantNumeric := false   // STRENGTH/DEXTERITY/INTELLIGENCE/LUCK/MAX_HP/MAX_MP
    relevantProfile := false   // LEVEL/JOB
    for _, u := range e.Body.Updates {
        switch u {
        case "MAX_HP", "MAX_MP", "STRENGTH", "DEXTERITY", "INTELLIGENCE", "LUCK":
            relevantNumeric = true
        case "LEVEL", "JOB":
            relevantProfile = true
        }
    }
    if !relevantNumeric && !relevantProfile { return }

    p  := character.NewProcessor(l, ctx)
    ch := channel.NewModel(e.WorldId, e.Body.ChannelId)

    if relevantNumeric && e.Body.Values != nil && len(e.Body.Values) > 0 {
        currentBase := lookupCurrentBase(ctx, l, ch, e.CharacterId)
        merged := mergeBaseStats(currentBase, e.Body.Values)
        if err := p.SetBaseStats(ch, e.CharacterId, merged); err != nil {
            l.WithError(err).Errorf("Unable to set base stats for character [%d].", e.CharacterId)
            // continue to profile branch — independent
        }
    }

    if relevantProfile {
        // values is nil for TypeLevel / TypeJob events; refetch wearer record.
        cm, err := externalcharacter.RequestById(e.CharacterId)(l, ctx)
        if err != nil {
            l.WithError(err).Warnf("Unable to refetch wearer profile for character [%d]; skipping re-gate.", e.CharacterId)
            return
        }
        wp := character.NewWearerProfile(cm.Level, cm.JobId)
        if err := p.SetWearerProfile(ch, e.CharacterId, wp); err != nil {
            l.WithError(err).Errorf("Unable to set wearer profile for character [%d].", e.CharacterId)
        }
    }
}
```

Both branches fold `RecomputeEquipmentBonuses` into the processor methods they call, so the consumer is not responsible for sequencing two side-effects.

### 7.2 `kafka/consumer/asset/consumer.go::handleItemEquipped`

`AddEquipmentBonuses` now takes `templateId`:

```go
if err := character.NewProcessor(l, ctx).AddEquipmentBonuses(ch, e.CharacterId, e.AssetId, e.TemplateId, bonuses); err != nil {
    ...
}
```

`e.TemplateId` is already on the moved-status event. The processor handles re-gate internally.

`handleItemUnequipped` and `handleAssetDeleted` are unchanged in surface — they call `RemoveEquipmentBonuses(characterId, assetId)`, which now also clears the snapshot and re-gates.

### 7.3 `external/character/rest.go`

Add the missing field:

```go
type RestModel struct {
    // existing...
    Level        byte     `json:"level"`
    JobId        job.Id   `json:"jobId"`   // NEW
    Strength     uint16   `json:"strength"`
    // ...
}
```

atlas-character already serialises `jobId`; this mirrors it on the consumer side.

### 7.4 `character/initializer.go::InitializeCharacter`

Restructured fetch order:

1. Fetch wearer record from atlas-character (single call carries Level + JobId + base stats).
2. `m = m.WithBaseStats(...).WithWearer(NewWearerProfile(charData.Level, charData.JobId))`.
3. Fetch buffs → `m.WithBonuses(buffBonuses)` (now only buff:* entries).
4. Fetch passives → `m.WithBonuses(passiveBonuses)` (now only passive:* entries).
5. Fetch equipment via the new `fetchEquippedSnapshots` helper, which iterates the equip compartment and returns `[]EquippedAsset` (one per `Slot < 0` asset, with bonuses pre-extracted via the existing extraction logic). For each: `m = m.WithEquippedAsset(snap)`.
6. `m = m.RecomputeWith(equipment.GetProvider(l), ctx).WithInitialized()`.
7. `GetRegistry().Update(ctx, m)`.

The existing per-source `Add*Bonuses` helpers stay; they just operate on `bonuses[]` (buff/passive only) or on `m.equipped` (equipment).

## 8. Failure Modes & Observability

| Scenario | Behaviour |
|----------|-----------|
| Cold cache + atlas-data unreachable | Provider returns `(_, false)`. Asset dropped from qualifying set. WARN log: `equipment template [%d] fetch failed; treating dependent assets as unqualified`. No retry; next re-eval trigger naturally re-attempts. |
| Warm cache + atlas-data unreachable | Cached value used (cache only stores successful fetches). Transparent. |
| Wearer-profile re-fetch fails (TypeLevel/TypeJob branch) | WARN log; re-gate is skipped. Next event on this character re-attempts. |
| Asset has no template lookup possible (template id 0 / corrupt) | `RequestById(0)` will 4xx; same path as cold-cache failure. |

Logging additions (DEBUG unless noted):

- `Recomputed qualifying equipment for character [%d]: %d/%d items qualify.`
- `Asset [%d] template [%d] does not qualify (failed: <which req>).`
- `Asset [%d] template [%d] qualifies.`
- WARN at the Provider failure site as above.

## 9. Test Plan

### 9.1 Unit

`character/model_test.go` (or a new `character/qualification_test.go`):

- **`meetsRequirements`** — six predicates × `{zero-skip, req-1, req, req+1}`.
  - reqJob bitmask: `{Beginner=0, Warrior=1, Magician=2, Bowman=4, Thief=8, Pirate=16}` plus OR'd combos (e.g. `1|2`); confirm Beginner (jobId=0) does NOT match `reqJob=1`; confirm jobId=2 matches `reqJob=1|2`.
- **`Model.QualifiedEquipment`** — synthesised models with stub Provider:
  - Empty equipped set → empty result.
  - Single qualifying asset, single non-qualifying asset → only the first qualifies.
  - Chain: A grants +5 STR no req; B requires STR≥(base+5); C requires STR≥(base+5+B.str). Base STR meets only A → after iteration {A,B,C} all qualify.
  - Cycle: A requires B's bonus; B requires A's bonus; neither base-qualifies → result empty.
  - Provider returning `false` for one asset → that asset excluded; others continue.
- **`Model.ComputeEffectiveStats`** — flat sum from base + buffs/passives + qualifying equipment; multipliers from `bonuses[]` only; MaxHpMpCap clamp behaviour preserved.

### 9.2 Integration

`character/initializer_test.go`, `kafka/consumer/character/consumer_test.go`, plus a new `kafka/consumer/asset/consumer_test.go` if missing:

- **PRD §4.1 reproduction** — stub atlas-inventory returning one compartment with asset slot=-5, mp=50, templateId=1052095; stub atlas-data returning `{reqLuk: 40}`; stub atlas-character returning Level=30 / JobId=200 / Luk=39. `GetEffectiveStats` returns `MaxMp == base.MaxMp` and `Bonuses()` excludes `equipment:<assetId>`.
- Same stubs but `Luk=40` → asset qualifies; `MaxMp == base + 50`; entry present.
- Dispatch STAT_CHANGED with `TypeLuck`, `values={"luck": 40}` → qualification flips; verify the inventory stub was NOT called a second time (snapshot preserved across the mutation).
- Dispatch STAT_CHANGED with `TypeJob`, `values=nil` → consumer re-fetches wearer; qualification flips for a `reqJob`-restricted asset.
- Dispatch STAT_CHANGED with `TypeLevel`, `values=nil` → consumer re-fetches wearer; qualification flips for a `reqLevel`-restricted asset.
- Equip a +STR cape (no requirements) that pushes effective STR over a previously-unqualifying weapon's `reqStr` → after `MOVED` is processed, both items appear in `Bonuses()`.

### 9.3 Cache

`external/data/equipment/cache_test.go`:

- Two consecutive `Provider(t, 1234)` → exactly one HTTP fetch (verified via stub counter).
- Cold-cache `Provider(t, 1234)` → atlas-data returns 5xx → `(_, false)`; cache empty; second call retries (also fails, also `(_, false)`).
- Tenant isolation: `Provider(tenantA, 1234)` then `Provider(tenantB, 1234)` → two separate HTTP fetches.

### 9.4 Smoke

Post-deploy on dev cluster:

- Re-run PRD §4.1 reproduction against the diagnosis character.
- Confirm `GET /api/worlds/0/channels/0/characters/<id>/stats` returns `MaxMp` without the +50 from the unqualified overall.
- Distribute `+1 LUK` AP; observe MaxMp jump by 50 and the `equipment:<assetId>` entry appearing in `bonuses[]`.

## 10. File-Level Diff Summary

### New
- `services/atlas-effective-stats/atlas.com/effective-stats/external/data/equipment/rest.go`
- `services/atlas-effective-stats/atlas.com/effective-stats/external/data/equipment/requests.go`
- `services/atlas-effective-stats/atlas.com/effective-stats/external/data/equipment/cache.go`
- `services/atlas-effective-stats/atlas.com/effective-stats/external/data/equipment/cache_test.go`
- `services/atlas-effective-stats/atlas.com/effective-stats/character/qualification_test.go` (or extend `model_test.go`)
- `services/atlas-effective-stats/atlas.com/effective-stats/kafka/consumer/asset/consumer_test.go` (if missing today)

### Modified
- `services/atlas-effective-stats/atlas.com/effective-stats/external/character/rest.go` — add `JobId job.Id`.
- `services/atlas-effective-stats/atlas.com/effective-stats/character/model.go` — add `wearer`, `equipped`, `qualifiedSnapshot`; builder methods; JSON marshal/unmarshal; `meetsRequirements`; `QualifiedEquipment`; reshape `ComputeEffectiveStats` and `Recompute` into `RecomputeWith`; `Bonuses()` reads qualifying snapshots.
- `services/atlas-effective-stats/atlas.com/effective-stats/character/initializer.go` — fetch wearer profile alongside base stats; replace `fetchEquipmentBonuses` with `fetchEquippedSnapshots`; final `RecomputeWith`.
- `services/atlas-effective-stats/atlas.com/effective-stats/character/processor.go` — add `SetWearerProfile`, `RecomputeEquipmentBonuses`; reshape `AddEquipmentBonuses` (templateId param, snapshot write, re-gate); `RemoveEquipmentBonuses` (snapshot clear, re-gate); fold re-gate into `SetBaseStats`.
- `services/atlas-effective-stats/atlas.com/effective-stats/character/registry.go` — add `PutEquippedAsset`, `RemoveEquippedAsset`, `SetWearerProfile`.
- `services/atlas-effective-stats/atlas.com/effective-stats/character/resource.go` — no signature changes; verifies `Bonuses()` is called only on a model whose `qualifiedSnapshot` has been populated.
- `services/atlas-effective-stats/atlas.com/effective-stats/kafka/consumer/character/consumer.go` — split `handleStatChanged` into numeric vs profile branches; expand `hasRelevantStats` to include `LEVEL` / `JOB`.
- `services/atlas-effective-stats/atlas.com/effective-stats/kafka/consumer/asset/consumer.go` — pass `e.TemplateId` into `AddEquipmentBonuses`.

### Unchanged
- atlas-data, atlas-inventory, atlas-character, atlas-channel, atlas-ui (per PRD §7).
