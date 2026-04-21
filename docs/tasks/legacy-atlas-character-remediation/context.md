# Atlas-Character Remediation - Context

**Last Updated:** 2026-01-13

---

## Key Files

### Primary Files to Modify

| File | Purpose | Changes Needed |
|------|---------|----------------|
| `services/atlas-character/atlas.com/character/character/rest.go` | REST Transform/Extract | Update both functions |

### Reference Files (Read-Only)

| File | Purpose | Why Referenced |
|------|---------|----------------|
| `services/atlas-character/atlas.com/character/character/model.go` | Domain model + accessors + modelBuilder | Source of accessor methods and builder pattern |
| `services/atlas-character/atlas.com/character/character/rest_test.go` | REST tests | Verify tests pass |
| `docs/audits/atlas-character/audit.md` | Audit findings | Original issue documentation |
| `docs/audits/atlas-character/audit.json` | Machine-readable audit | Issue metadata |

### Example Compliant Files

| File | Pattern Demonstrated |
|------|---------------------|
| `services/atlas-chalkboards/atlas.com/chalkboards/chalkboard/rest.go` | Transform using accessors: `m.Id()`, `m.Message()` |

---

## Key Decisions

### Decision 1: Use Existing modelBuilder for Extract

**Context:** The character model has two builder implementations:
1. `Builder` (builder.go) - For character creation with game-specific config initialization
2. `modelBuilder` (model.go) - For internal entity-to-model conversion

**Decision:** Use `modelBuilder` for the `Extract` function because:
- It's already designed for reconstructing models from external data
- It has all required `Set*` methods
- The `Builder` requires game config and is meant for initial character creation

**Alternative Considered:** Creating a third builder specific to REST extraction
**Why Rejected:** Unnecessary complexity; `modelBuilder` serves this purpose

### Decision 2: No Validation in Extract

**Context:** The `modelBuilder.Build()` method does not perform validation.

**Decision:** This is acceptable for the Extract function because:
- REST input is validated at the handler level before Extract is called
- The original code also didn't validate
- Adding validation would change behavior (potential breaking change)

### Decision 3: Accessor Method Naming

**Context:** Some accessor methods have different naming conventions than the private fields:
- `hp` -> `HP()` (uppercase)
- `maxHp` -> `MaxHP()` (uppercase)
- `sp` -> `SPString()` (different name)
- `hpMpUsed` -> `HPMPUsed()` (uppercase)
- `gm` -> `GM()` (uppercase)
- `ap` -> `AP()` (uppercase)

**Decision:** Use the exact accessor method names as defined in model.go.

---

## Dependencies

### Internal Dependencies

```
rest.go
  └── imports
      ├── model.go (Model, NewModelBuilder, modelBuilder)
      └── temporal_data.go (GetTemporalRegistry)
```

### External Dependencies

```
rest.go
  └── imports
      ├── github.com/Chronicle20/atlas-constants/job
      ├── github.com/Chronicle20/atlas-constants/map
      └── github.com/Chronicle20/atlas-constants/world
```

---

## Model Field to Accessor Mapping

Complete mapping for Transform function:

```go
// Private Field    ->  Accessor Method
m.id                ->  m.Id()
m.accountId         ->  m.AccountId()
m.worldId           ->  m.WorldId()
m.name              ->  m.Name()
m.level             ->  m.Level()
m.experience        ->  m.Experience()
m.gachaponExperience ->  m.GachaponExperience()
m.strength          ->  m.Strength()
m.dexterity         ->  m.Dexterity()
m.intelligence      ->  m.Intelligence()
m.luck              ->  m.Luck()
m.hp                ->  m.HP()           // Note: uppercase
m.maxHp             ->  m.MaxHP()        // Note: uppercase
m.mp                ->  m.MP()           // Note: uppercase
m.maxMp             ->  m.MaxMP()        // Note: uppercase
m.meso              ->  m.Meso()
m.hpMpUsed          ->  m.HPMPUsed()     // Note: uppercase
m.jobId             ->  m.JobId()
m.skinColor         ->  m.SkinColor()
m.gender            ->  m.Gender()
m.fame              ->  m.Fame()
m.hair              ->  m.Hair()
m.face              ->  m.Face()
m.ap                ->  m.AP()           // Note: uppercase
m.sp                ->  m.SPString()     // Note: different name
m.mapId             ->  m.MapId()
m.spawnPoint        ->  m.SpawnPoint()
m.gm                ->  m.GM()           // Note: uppercase
```

---

## Builder Method Mapping

Complete mapping for Extract function:

```go
// RestModel Field  ->  Builder Method
m.Id                ->  SetId()
m.AccountId         ->  SetAccountId()
m.WorldId           ->  SetWorldId()
m.Name              ->  SetName()
m.Level             ->  SetLevel()
m.Experience        ->  SetExperience()
m.GachaponExperience ->  SetGachaponExperience()
m.Strength          ->  SetStrength()
m.Dexterity         ->  SetDexterity()
m.Intelligence      ->  SetIntelligence()
m.Luck              ->  SetLuck()
m.Hp                ->  SetHp()
m.MaxHp             ->  SetMaxHp()
m.Mp                ->  SetMp()
m.MaxMp             ->  SetMaxMp()
m.Meso              ->  SetMeso()
m.HpMpUsed          ->  SetHpMpUsed()
m.JobId             ->  SetJobId()
m.SkinColor         ->  SetSkinColor()
m.Gender            ->  SetGender()
m.Fame              ->  SetFame()
m.Hair              ->  SetHair()
m.Face              ->  SetFace()
m.Ap                ->  SetAp()
m.Sp                ->  SetSp()
m.MapId             ->  SetMapId()
m.SpawnPoint        ->  SetSpawnPoint()
m.Gm                ->  SetGm()
```

---

## Testing Commands

```bash
# Run all character package tests
cd services/atlas-character && go test ./atlas.com/character/character/...

# Run with verbose output
cd services/atlas-character && go test -v ./atlas.com/character/character/...

# Run specific test file
cd services/atlas-character && go test -v ./atlas.com/character/character/ -run TestTransform
```

---

## Related Audit Checks

| Check ID | Description | Current Status | After Remediation |
|----------|-------------|----------------|-------------------|
| ARCH-008 | REST JSON:API Pattern | `fail` | `pass` |
| ARCH-003 | Builder Pattern | `warn` | `warn` (documentation only) |
