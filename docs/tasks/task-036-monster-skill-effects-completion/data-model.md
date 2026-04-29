# Data Model — task-036

Companion to `prd.md`. Lays out new and modified types so the plan phase can write tests first without re-deriving structure.

---

## 1. atlas-monsters — `monster.StatusEffect` extension

```go
type StatusEffect struct {
    // existing
    effectId           uuid.UUID
    sourceType         string
    sourceCharacterId  uint32
    sourceSkillId      uint32
    sourceSkillLevel   uint32
    statuses           map[string]int32
    duration           time.Duration
    tickInterval       time.Duration
    lastTick           time.Time
    createdAt          time.Time
    expiresAt          time.Time

    // NEW — zero values for non-reflect statuses.
    reflectKind        string // "" | "PHYSICAL" | "MAGICAL"
    reflectPercent     int32
    reflectRange       int32
    reflectMaxDamage   int32
}

func (s StatusEffect) ReflectKind() string       { return s.reflectKind }
func (s StatusEffect) ReflectPercent() int32     { return s.reflectPercent }
func (s StatusEffect) ReflectRange() int32       { return s.reflectRange }
func (s StatusEffect) ReflectMaxDamage() int32   { return s.reflectMaxDamage }
func (s StatusEffect) IsReflect() bool           { return s.reflectKind != "" }
```

### Constructor surface

- Existing `NewStatusEffect(...)` keeps its signature; new fields default to zero. All existing call sites continue to work.
- New `NewReflectStatusEffect(sourceType, srcCharId, skillId, skillLevel, statuses, duration, kind, percent, range, maxDamage)` for reflect call sites.
- Builder-style would also work but keeps the existing constructor pattern of the file. Plan phase picks one and is consistent across both helpers.

### `WithLastTick` extension

No change — existing copy-by-value pattern propagates the new fields automatically.

---

## 2. atlas-monsters — venom slot status names

```go
// libs/atlas-constants/monster/skill.go (or status.go)
const (
    StatusVenom1 = "VENOM_1"
    StatusVenom2 = "VENOM_2"
    StatusVenom3 = "VENOM_3"
    StatusVenom  = "VENOM" // wire-side alias

    StatusVenomLuckSuffix   = "_LUCK"
    StatusVenomMatkSuffix   = "_MATK"
    StatusVenomSourceSuffix = "_SOURCE"
)

func IsVenomSlot(name string) bool {
    return name == StatusVenom1 || name == StatusVenom2 || name == StatusVenom3
}

func VenomWireName() string { return StatusVenom }
```

### Status map shape per slot

For slot N, the `statuses` map carries:

```
{
  "VENOM_N":         <snapshotDamagePerTick>,
  "VENOM_N_LUCK":    <attackerLuck>,
  "VENOM_N_MATK":    <attackerMagicAttack>,
  "VENOM_N_SOURCE":  <attackerCharacterId>,
}
```

The auxiliary keys are NOT broadcast to the client — atlas-channel's `VENOM_N → VENOM` translator drops them.

---

## 3. atlas-monsters — venom slot allocator

Pseudocode for the slot-selection helper (lives in `monster/processor.go` next to `executeStatBuff`):

```go
func (p *ProcessorImpl) allocateVenomSlot(m Model) string {
    active := m.StatusEffects()
    var occupied [3]*StatusEffect
    for i, slot := range []string{StatusVenom1, StatusVenom2, StatusVenom3} {
        for _, se := range active {
            if se.HasStatus(slot) {
                occupied[i] = &se
                break
            }
        }
    }
    // Prefer first empty slot.
    for i, slot := range []string{StatusVenom1, StatusVenom2, StatusVenom3} {
        if occupied[i] == nil { return slot }
    }
    // All occupied — replace oldest by ExpiresAt.
    oldest := 0
    for i := 1; i < 3; i++ {
        if occupied[i].ExpiresAt().Before(occupied[oldest].ExpiresAt()) {
            oldest = i
        }
    }
    return []string{StatusVenom1, StatusVenom2, StatusVenom3}[oldest]
}
```

When a slot is being **replaced**, the existing status MUST be cancelled (regular cancel flow with event emission) before the new apply.

---

## 4. atlas-monsters — DoT tick aggregation

Update `processDoTTick` (`status_task.go:61`):

```go
if se.HasStatus("POISON")          { totalDamage += poisonDamage(m, se) }
for _, slot := range venomSlots {
    if se.HasStatus(slot) { totalDamage += venomDamage(se, slot) }
}
```

Where `venomDamage(se, slot)` reads `se.statuses[slot]` directly — already the snapshot DPT.

---

## 5. atlas-channel — `monster.StatusMirror`

```go
type ReflectInfo struct {
    Kind             string
    Percent          int32
    Range            int32
    MaxDamage        int32
    ExpiresAt        time.Time
}

type StatusEntry struct {
    Statuses     map[string]int32
    Reflect      *ReflectInfo // nil for non-reflect entries
    ExpiresAt    time.Time
}

type StatusMirror struct {
    mu       sync.RWMutex
    perTenant map[string]map[uint32]map[string]StatusEntry
}

func GetStatusMirror() *StatusMirror // singleton via sync.Once
func (m *StatusMirror) OnApplied(t tenant.Model, uniqueId uint32, body StatusEffectAppliedBody)
func (m *StatusMirror) OnExpired(t tenant.Model, uniqueId uint32, statuses map[string]int32)
func (m *StatusMirror) OnCancelled(t tenant.Model, uniqueId uint32, statuses map[string]int32)
func (m *StatusMirror) OnMonsterGone(t tenant.Model, uniqueId uint32)
func (m *StatusMirror) GetReflect(t tenant.Model, uniqueId uint32, kind string) (ReflectInfo, bool)
func (m *StatusMirror) VenomSlotCount(t tenant.Model, uniqueId uint32) int
```

The mirror's tenant key follows whichever pattern atlas-channel already uses (string-cast UUID is the common idiom). No persistence.

---

## 6. atlas-maps — `mist.Mist`

```go
type Point struct { X, Y int16 }

type Mist struct {
    id              uuid.UUID
    field           field.Model
    ownerType       string
    ownerId         uint32
    origin          Point
    ltX, ltY        int16
    rbX, rbY        int16
    disease         string
    diseaseValue    int32
    diseaseDuration time.Duration
    duration        time.Duration
    tickInterval    time.Duration
    createdAt       time.Time
    expiresAt       time.Time
    lastTick        time.Time
}

// Standard immutable getters (one per private field).
// Builder follows the existing convention used elsewhere in atlas-maps for
// immutable models.

func (m Mist) Contains(x, y int16) bool {
    minX, maxX := m.origin.X+m.ltX, m.origin.X+m.rbX
    minY, maxY := m.origin.Y+m.ltY, m.origin.Y+m.rbY
    return x >= minX && x <= maxX && y >= minY && y <= maxY
}

func (m Mist) Expired() bool { return time.Now().After(m.expiresAt) }

func (m Mist) ShouldTick() bool {
    return m.tickInterval > 0 && time.Since(m.lastTick) >= m.tickInterval
}
```

### Registry

```go
type MistRegistry struct {
    mu       sync.RWMutex
    perTenant map[string]map[uuid.UUID]Mist
    byField  map[string]map[fieldKey][]uuid.UUID
}

func GetMistRegistry() *MistRegistry
func (r *MistRegistry) Add(t tenant.Model, m Mist) error
func (r *MistRegistry) Remove(t tenant.Model, id uuid.UUID) (Mist, error)
func (r *MistRegistry) GetByField(t tenant.Model, f field.Model) []Mist
func (r *MistRegistry) UpdateLastTick(t tenant.Model, id uuid.UUID, at time.Time)
```

---

## 7. atlas-maps — `MistTickTask`

```go
type MistTickTask struct {
    l        logrus.FieldLogger
    ctx      context.Context
    interval time.Duration
}

func NewMistTickTask(l logrus.FieldLogger, ctx context.Context, interval time.Duration) *MistTickTask

func (t *MistTickTask) Run() {
    // for each tenant:
    //   for each mist in registry:
    //     if expired: remove + emit MIST_DESTROYED(EXPIRED)
    //     else if ShouldTick: list characters in field, filter Contains,
    //                         produce apply-disease command per character,
    //                         UpdateLastTick(now)
}

func (t *MistTickTask) SleepTime() time.Duration { return t.interval }
```

---

## 8. atlas-buffs — `tasks.PoisonTick`

```go
type PoisonTick struct {
    l        logrus.FieldLogger
    interval int // ms
}

func NewPoisonTick(l logrus.FieldLogger, interval int) *PoisonTick

func (r *PoisonTick) Run() {
    // for each tenant context:
    //   entries := character.GetRegistry().GetPoisonCharacters(ctx)
    //   now := time.Now()
    //   for each e in entries:
    //     last, ok := character.GetRegistry().GetLastPoisonTick(ctx, e.CharacterId)
    //     if ok && now.Sub(last) < tickInterval { continue }
    //     produce CHARACTER_DAMAGE command for (e.CharacterId, e.Amount, POISON)
    //     character.GetRegistry().UpdatePoisonTick(ctx, e.CharacterId, now)
}

func (r *PoisonTick) SleepTime() time.Duration { return time.Duration(r.interval) * time.Millisecond }
```

Wired in atlas-buffs `main.go` next to the existing Expiration task; default 1000 ms.

---

## 9. State transitions — venom slot

```
                applyVenom()
                    │
                    ▼
            [find slot]──── empty? ───────► insert into slot N
                    │
                    └─── all occupied ────► cancel slot with earliest ExpiresAt
                                              │
                                              ▼
                                          insert into freed slot

                expireOrCancel(slot N)
                    │
                    ▼
           [count remaining slots]
                    │
        ┌───────────┼───────────┐
        ▼                       ▼
   slots > 0               slots == 0
   wire: no-op             wire: MonsterStatReset(VENOM)
```

---

## 10. State transitions — reflect mirror

```
APPLIED(reflect):  Mirror.OnApplied → entry written, ReflectInfo populated
EXPIRED(reflect):  Mirror.OnExpired → entry removed
CANCELLED(reflect): same as EXPIRED
DESTROYED/KILLED(monster): Mirror.OnMonsterGone → all entries for uniqueId removed

attack(damageEntry, attackerKind):
    info, ok := Mirror.GetReflect(monsterId, kindFor(attackerKind))
    if !ok: continue
    if dx := |attacker.X - monster.X|; dx > info.Range: continue
    reflected := min(damageEntry.Damage * info.Percent / 100, info.MaxDamage)
    produce DAMAGE_REFLECTED(monsterId, attackerId, reflected)
    damageEntry.Damage = 0
```
