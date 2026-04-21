# Atlas-Character Service Remediation Plan

**Service:** `atlas-character`
**Service Path:** `services/atlas-character/atlas.com/character`
**Audit Reference:** `dev/audits/atlas-character/audit.md`
**Last Updated:** 2026-01-13
**Status:** Planned

---

## 1. Executive Summary

The `atlas-character` service audit identified **2 blocking issues** and **2 non-blocking issues** that need remediation. The service currently achieves an 87.5% pass rate (14/16 checks) against the Atlas backend architecture guidelines.

### Blocking Issues (P0)
1. **ARCH-008a**: REST `Transform` function directly accesses private model fields instead of using accessor methods
2. **ARCH-008b**: REST `Extract` function directly creates Model struct bypassing the builder pattern

### Non-Blocking Issues (P2)
1. **ARCH-003**: Dual builder implementations (`Builder` and `modelBuilder`) may cause confusion - needs documentation
2. Custom REST handler abstraction (functionally equivalent, no action required)

---

## 2. Current State Analysis

### Problem 1: Transform Function (rest.go:65-101)

The `Transform` function converts a domain `Model` to a `RestModel` for JSON:API serialization. Currently it accesses private fields directly:

```go
// CURRENT - Violates encapsulation
func Transform(m Model) (RestModel, error) {
    td := GetTemporalRegistry().GetById(m.Id())
    rm := RestModel{
        Id:         m.id,        // Direct private field access
        AccountId:  m.accountId, // Direct private field access
        // ... 25+ more fields
    }
    return rm, nil
}
```

### Problem 2: Extract Function (rest.go:103-134)

The `Extract` function converts a `RestModel` back to a domain `Model`. Currently it directly instantiates the struct:

```go
// CURRENT - Bypasses builder pattern
func Extract(m RestModel) (Model, error) {
    return Model{
        id:        m.Id,        // Direct struct instantiation
        accountId: m.AccountId, // Bypasses builder
        // ... 25+ more fields
    }, nil
}
```

### Available Resources

The `model.go` file already provides:
- **30+ accessor methods**: `Id()`, `AccountId()`, `WorldId()`, `Name()`, `Level()`, etc.
- **`modelBuilder`**: Fluent builder with `SetId()`, `SetAccountId()`, etc.
- **`NewModelBuilder()`**: Factory function returning empty builder

---

## 3. Proposed Future State

### Solution 1: Transform Using Accessors

```go
func Transform(m Model) (RestModel, error) {
    td := GetTemporalRegistry().GetById(m.Id())
    rm := RestModel{
        Id:                 m.Id(),
        AccountId:          m.AccountId(),
        WorldId:            m.WorldId(),
        Name:               m.Name(),
        Level:              m.Level(),
        Experience:         m.Experience(),
        GachaponExperience: m.GachaponExperience(),
        Strength:           m.Strength(),
        Dexterity:          m.Dexterity(),
        Intelligence:       m.Intelligence(),
        Luck:               m.Luck(),
        Hp:                 m.HP(),
        MaxHp:              m.MaxHP(),
        Mp:                 m.MP(),
        MaxMp:              m.MaxMP(),
        Meso:               m.Meso(),
        HpMpUsed:           m.HPMPUsed(),
        JobId:              m.JobId(),
        SkinColor:          m.SkinColor(),
        Gender:             m.Gender(),
        Fame:               m.Fame(),
        Hair:               m.Hair(),
        Face:               m.Face(),
        Ap:                 m.AP(),
        Sp:                 m.SPString(),
        MapId:              m.MapId(),
        SpawnPoint:         m.SpawnPoint(),
        Gm:                 m.GM(),
        X:                  td.X(),
        Y:                  td.Y(),
        Stance:             td.Stance(),
    }
    return rm, nil
}
```

### Solution 2: Extract Using Builder

```go
func Extract(m RestModel) (Model, error) {
    return NewModelBuilder().
        SetId(m.Id).
        SetAccountId(m.AccountId).
        SetWorldId(m.WorldId).
        SetName(m.Name).
        SetLevel(m.Level).
        SetExperience(m.Experience).
        SetGachaponExperience(m.GachaponExperience).
        SetStrength(m.Strength).
        SetDexterity(m.Dexterity).
        SetIntelligence(m.Intelligence).
        SetLuck(m.Luck).
        SetHp(m.Hp).
        SetMaxHp(m.MaxHp).
        SetMp(m.Mp).
        SetMaxMp(m.MaxMp).
        SetMeso(m.Meso).
        SetHpMpUsed(m.HpMpUsed).
        SetJobId(m.JobId).
        SetSkinColor(m.SkinColor).
        SetGender(m.Gender).
        SetFame(m.Fame).
        SetHair(m.Hair).
        SetFace(m.Face).
        SetAp(m.Ap).
        SetSp(m.Sp).
        SetMapId(m.MapId).
        SetSpawnPoint(m.SpawnPoint).
        SetGm(m.Gm).
        Build(), nil
}
```

---

## 4. Implementation Phases

### Phase 1: Blocking Issues (P0)
**Priority:** Critical - Must complete before merge

| # | Task | File | Effort | Dependencies |
|---|------|------|--------|--------------|
| 1.1 | Update Transform to use accessor methods | `character/rest.go` | S | None |
| 1.2 | Update Extract to use modelBuilder | `character/rest.go` | S | None |
| 1.3 | Run existing tests to verify behavior | - | S | 1.1, 1.2 |
| 1.4 | Update rest_test.go if needed | `character/rest_test.go` | S | 1.3 |

### Phase 2: Documentation (P2)
**Priority:** Low - Can be done separately

| # | Task | File | Effort | Dependencies |
|---|------|------|--------|--------------|
| 2.1 | Document Builder vs modelBuilder purpose in code comments | `character/builder.go`, `character/model.go` | S | Phase 1 |

---

## 5. Detailed Tasks

### Task 1.1: Update Transform Function

**File:** `services/atlas-character/atlas.com/character/character/rest.go`
**Lines:** 65-101

**Acceptance Criteria:**
- [ ] All direct field accesses (`m.id`, `m.accountId`, etc.) replaced with accessor calls (`m.Id()`, `m.AccountId()`, etc.)
- [ ] Temporal data fields (X, Y, Stance) remain unchanged (already using accessors via `td.X()`, etc.)
- [ ] Function signature unchanged
- [ ] Return type unchanged

**Field Mapping:**
| Private Field | Accessor Method |
|---------------|-----------------|
| `m.id` | `m.Id()` |
| `m.accountId` | `m.AccountId()` |
| `m.worldId` | `m.WorldId()` |
| `m.name` | `m.Name()` |
| `m.level` | `m.Level()` |
| `m.experience` | `m.Experience()` |
| `m.gachaponExperience` | `m.GachaponExperience()` |
| `m.strength` | `m.Strength()` |
| `m.dexterity` | `m.Dexterity()` |
| `m.intelligence` | `m.Intelligence()` |
| `m.luck` | `m.Luck()` |
| `m.hp` | `m.HP()` |
| `m.maxHp` | `m.MaxHP()` |
| `m.mp` | `m.MP()` |
| `m.maxMp` | `m.MaxMP()` |
| `m.meso` | `m.Meso()` |
| `m.hpMpUsed` | `m.HPMPUsed()` |
| `m.jobId` | `m.JobId()` |
| `m.skinColor` | `m.SkinColor()` |
| `m.gender` | `m.Gender()` |
| `m.fame` | `m.Fame()` |
| `m.hair` | `m.Hair()` |
| `m.face` | `m.Face()` |
| `m.ap` | `m.AP()` |
| `m.sp` | `m.SPString()` |
| `m.mapId` | `m.MapId()` |
| `m.spawnPoint` | `m.SpawnPoint()` |
| `m.gm` | `m.GM()` |

### Task 1.2: Update Extract Function

**File:** `services/atlas-character/atlas.com/character/character/rest.go`
**Lines:** 103-134

**Acceptance Criteria:**
- [ ] Direct Model struct instantiation replaced with builder pattern
- [ ] Uses `NewModelBuilder()` to create builder instance
- [ ] All fields set via fluent `Set*()` methods
- [ ] Ends with `.Build()` to produce Model
- [ ] Function signature unchanged
- [ ] Return type unchanged

### Task 1.3: Verify Tests Pass

**Acceptance Criteria:**
- [ ] `go test ./character/...` passes
- [ ] Existing `rest_test.go` tests pass without modification
- [ ] No regression in Transform/Extract behavior

### Task 1.4: Update Tests If Needed

**File:** `services/atlas-character/atlas.com/character/character/rest_test.go`

**Acceptance Criteria:**
- [ ] If tests fail, update to match new implementation
- [ ] Test coverage for Transform function maintained
- [ ] Test coverage for Extract function maintained

### Task 2.1: Document Builder Variants (Optional)

**Files:**
- `services/atlas-character/atlas.com/character/character/builder.go`
- `services/atlas-character/atlas.com/character/character/model.go`

**Acceptance Criteria:**
- [ ] Add doc comment to `Builder` struct explaining it's for character creation with game config
- [ ] Add doc comment to `modelBuilder` struct explaining it's for internal entity-to-model conversion
- [ ] Comments explain why two builders exist and when to use each

---

## 6. Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Transform accessor name mismatch | Low | Medium | Field mapping table above; verify each accessor exists in model.go |
| Extract builder method missing | Low | Medium | Verify all Set* methods exist in modelBuilder |
| Test failures | Low | Low | Run tests after each change; existing tests should pass |
| Behavioral regression | Very Low | High | The changes are purely structural; no logic changes |

---

## 7. Success Metrics

| Metric | Target |
|--------|--------|
| ARCH-008 audit status | Pass |
| Test pass rate | 100% |
| Overall audit pass rate | 100% (16/16) |

---

## 8. Required Resources and Dependencies

### Prerequisites
- Go development environment
- Access to `services/atlas-character` codebase
- Understanding of model.go accessor methods

### Dependencies
- No external dependencies
- No service restarts required
- No database migrations

### Estimated Total Effort
- Phase 1 (Blocking): ~30 minutes
- Phase 2 (Documentation): ~10 minutes
