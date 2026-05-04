# atlas-effective-stats: equipment stat requirements — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Gate per-asset equipment bonuses in atlas-effective-stats on the wearer's `reqLevel` / `reqJob` / `reqStr` / `reqDex` / `reqInt` / `reqLuk`, including cross-asset qualification via fixed-point iteration and dynamic re-gating on `STAT_CHANGED`/asset events, so `Computed.maxMp` agrees with the v83 client.

**Architecture:** Add an indefinite per-tenant cache of equipment-template requirements fetched from atlas-data; promote equipped items into a snapshot map on `Model` and let `bonuses[]` carry only `buff:*`/`passive:*` entries; introduce a pure `QualifiedEquipment` fixed-point iterator on `Model`; funnel every state change through a new `Processor.RecomputeEquipmentBonuses` so `Computed` and the eventual `Bonuses()` slice always reflect the qualifying subset.

**Tech Stack:** Go (`atlas-effective-stats` module), `sync.Once`/`sync.RWMutex` registry pattern, JSON:API requests via `requests.RootUrl("DATA")`, `miniredis` test-backed `atlas.TenantRegistry`, `stretchr/testify`-free table-driven tests with `*testing.T`, `logrus` for structured logging.

> **Read first:** `docs/tasks/task-053-equip-stat-requirements/context.md`. The reqJob bitmask gotcha there is required reading — the design's literal expression does not work for atlas internal job IDs.

> **Working directory** for every task: `services/atlas-effective-stats/atlas.com/effective-stats`. All paths in this plan are relative to that directory unless prefixed with `services/`, `libs/`, or `docs/`.

---

## Task 1: Add `JobId` to the external character REST model

**Why:** The wearer-profile branch (Task 19) needs `JobId` to call `SetWearerProfile`. atlas-character already serialises `jobId`; we just have to mirror it.

**Files:**
- Modify: `external/character/rest.go`

- [ ] **Step 1: Add the import and field**

Edit `external/character/rest.go`:

```go
package character

import (
	"strconv"

	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// RestModel represents a character from atlas-character service
type RestModel struct {
	Id           uint32   `json:"-"`
	AccountId    uint32   `json:"accountId"`
	WorldId      world.Id `json:"worldId"`
	Name         string   `json:"name"`
	Level        byte     `json:"level"`
	JobId        job.Id   `json:"jobId"`
	Strength     uint16   `json:"strength"`
	Dexterity    uint16   `json:"dexterity"`
	Intelligence uint16   `json:"intelligence"`
	Luck         uint16   `json:"luck"`
	Hp           uint16   `json:"hp"`
	MaxHp        uint16   `json:"maxHp"`
	Mp           uint16   `json:"mp"`
	MaxMp        uint16   `json:"maxMp"`
}
```

(Everything below `SetID` is unchanged.)

- [ ] **Step 2: Build to confirm the import resolves**

```bash
go build ./...
```
Expected: clean build.

- [ ] **Step 3: Commit**

```bash
git add external/character/rest.go
git commit -m "feat(atlas-effective-stats): mirror jobId on external character REST model"
```

---

## Task 2: Introduce `WearerProfile` value type

**Files:**
- Create: `character/wearer.go`
- Test: `character/wearer_test.go`

- [ ] **Step 1: Write the failing test**

Create `character/wearer_test.go`:

```go
package character

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
)

func TestWearerProfile_GettersReturnConstructorArgs(t *testing.T) {
	wp := NewWearerProfile(35, job.Id(200))
	if wp.Level() != 35 {
		t.Errorf("Level() = %d, want 35", wp.Level())
	}
	if wp.JobId() != job.Id(200) {
		t.Errorf("JobId() = %d, want 200", wp.JobId())
	}
}

func TestWearerProfile_ZeroValueIsBeginnerLevelZero(t *testing.T) {
	var wp WearerProfile
	if wp.Level() != 0 {
		t.Errorf("zero Level() = %d, want 0", wp.Level())
	}
	if wp.JobId() != job.Id(0) {
		t.Errorf("zero JobId() = %d, want 0", wp.JobId())
	}
}
```

- [ ] **Step 2: Run, expect FAIL (`undefined: NewWearerProfile`)**

```bash
go test ./character/ -run TestWearerProfile -v
```

- [ ] **Step 3: Implement**

Create `character/wearer.go`:

```go
package character

import "github.com/Chronicle20/atlas/libs/atlas-constants/job"

// WearerProfile carries the non-numeric inputs to equipment requirement checks.
// Lives alongside the numeric stat.Base inside the character Model.
type WearerProfile struct {
	level byte
	jobId job.Id
}

func NewWearerProfile(level byte, jobId job.Id) WearerProfile {
	return WearerProfile{level: level, jobId: jobId}
}

func (p WearerProfile) Level() byte    { return p.level }
func (p WearerProfile) JobId() job.Id  { return p.jobId }
```

- [ ] **Step 4: Run, expect PASS**

```bash
go test ./character/ -run TestWearerProfile -v
```

- [ ] **Step 5: Commit**

```bash
git add character/wearer.go character/wearer_test.go
git commit -m "feat(atlas-effective-stats): add WearerProfile value type"
```

---

## Task 3: Introduce `EquippedAsset` snapshot type

**Files:**
- Create: `character/equipped_asset.go`
- Test: `character/equipped_asset_test.go`

- [ ] **Step 1: Write the failing test**

Create `character/equipped_asset_test.go`:

```go
package character

import (
	"atlas-effective-stats/stat"
	"testing"
)

func TestEquippedAsset_Getters(t *testing.T) {
	bonuses := []stat.Bonus{
		stat.NewBonus("equipment:42", stat.TypeMaxMp, 50),
	}
	a := NewEquippedAsset(42, 1052095, bonuses)

	if a.AssetId() != 42 {
		t.Errorf("AssetId() = %d, want 42", a.AssetId())
	}
	if a.TemplateId() != 1052095 {
		t.Errorf("TemplateId() = %d, want 1052095", a.TemplateId())
	}
	got := a.Bonuses()
	if len(got) != 1 || got[0].Amount() != 50 {
		t.Errorf("Bonuses() = %+v, want one MaxMp=50", got)
	}
}

func TestEquippedAsset_BonusesIsDefensiveCopy(t *testing.T) {
	bonuses := []stat.Bonus{
		stat.NewBonus("equipment:42", stat.TypeMaxMp, 50),
	}
	a := NewEquippedAsset(42, 1052095, bonuses)

	bonuses[0] = stat.NewBonus("equipment:42", stat.TypeMaxMp, 9999)
	if a.Bonuses()[0].Amount() != 50 {
		t.Errorf("internal bonuses leaked through constructor; got %d", a.Bonuses()[0].Amount())
	}

	out := a.Bonuses()
	out[0] = stat.NewBonus("equipment:42", stat.TypeMaxMp, -1)
	if a.Bonuses()[0].Amount() != 50 {
		t.Errorf("internal bonuses leaked through Bonuses(); got %d", a.Bonuses()[0].Amount())
	}
}
```

- [ ] **Step 2: Run, expect FAIL**

```bash
go test ./character/ -run TestEquippedAsset -v
```

- [ ] **Step 3: Implement**

Create `character/equipped_asset.go`:

```go
package character

import "atlas-effective-stats/stat"

// EquippedAsset is the per-asset snapshot held on the character Model.
// It is the source of truth for equipment bonuses; m.bonuses[] holds only
// buff:* and passive:* entries.
type EquippedAsset struct {
	assetId    uint32
	templateId uint32
	bonuses    []stat.Bonus
}

// NewEquippedAsset takes a defensive copy of bonuses so callers can mutate
// their slice without affecting the snapshot.
func NewEquippedAsset(assetId, templateId uint32, bonuses []stat.Bonus) EquippedAsset {
	owned := make([]stat.Bonus, len(bonuses))
	copy(owned, bonuses)
	return EquippedAsset{
		assetId:    assetId,
		templateId: templateId,
		bonuses:    owned,
	}
}

func (a EquippedAsset) AssetId() uint32    { return a.assetId }
func (a EquippedAsset) TemplateId() uint32 { return a.templateId }

// Bonuses returns a defensive copy of the snapshot's flat bonuses.
func (a EquippedAsset) Bonuses() []stat.Bonus {
	out := make([]stat.Bonus, len(a.bonuses))
	copy(out, a.bonuses)
	return out
}
```

- [ ] **Step 4: Run, expect PASS**

```bash
go test ./character/ -run TestEquippedAsset -v
```

- [ ] **Step 5: Commit**

```bash
git add character/equipped_asset.go character/equipped_asset_test.go
git commit -m "feat(atlas-effective-stats): add EquippedAsset snapshot type"
```

---

## Task 4: Extend `Model` with wearer + equipped + qualifiedSnapshot fields and builders

**Why:** The new fields need to coexist with all existing builders. We add the storage, getters, and builders before any code reads them, so existing tests keep passing (the new fields default to zero/empty).

**Files:**
- Modify: `character/model.go`

- [ ] **Step 1: Add the fields and getters**

Replace the `Model` struct and its trivial getters in `character/model.go`. The full updated section (lines 22-92) reads:

```go
// Model holds all stat bonuses and computed effective stats for a character
type Model struct {
	tenant      tenant.Model
	ch          channel.Model
	characterId uint32

	// Base stats from character service
	baseStats stat.Base

	// Bonuses by source. After this task lands and the equipment migration in
	// Task 17 / 18 completes, this slice holds only buff:* and passive:*
	// entries; equipment lives in the equipped map below.
	bonuses []stat.Bonus

	// Wearer profile (level + jobId) — inputs to reqLevel / reqJob.
	wearer WearerProfile

	// Equipped-asset snapshot map keyed by assetId. Source of truth for
	// equipment bonuses.
	equipped map[uint32]EquippedAsset

	// Cached set of qualifying asset ids from the most recent
	// RecomputeWith. Read by Bonuses() to avoid re-running the iterator.
	// Always treated as read-only after construction.
	qualifiedSnapshot map[uint32]bool

	// Cached computed totals
	computed    stat.Computed
	lastUpdated time.Time
	initialized bool
}

func (m Model) Tenant() tenant.Model {
	return m.tenant
}

func (m Model) WorldId() world.Id {
	return m.ch.WorldId()
}

func (m Model) ChannelId() channel.Id {
	return m.ch.Id()
}

func (m Model) Channel() channel.Model {
	return m.ch
}

func (m Model) CharacterId() uint32 {
	return m.characterId
}

func (m Model) BaseStats() stat.Base {
	return m.baseStats
}

func (m Model) Wearer() WearerProfile {
	return m.wearer
}

// Equipped returns a copy of the equipped-asset snapshot map.
func (m Model) Equipped() map[uint32]EquippedAsset {
	out := make(map[uint32]EquippedAsset, len(m.equipped))
	for k, v := range m.equipped {
		out[k] = v
	}
	return out
}
```

- [ ] **Step 2: Update `NewModel` to initialise the maps**

```go
// NewModel creates a new character effective stats model
func NewModel(t tenant.Model, ch channel.Model, characterId uint32) Model {
	return Model{
		tenant:            t,
		ch:                ch,
		characterId:       characterId,
		bonuses:           make([]stat.Bonus, 0),
		equipped:          make(map[uint32]EquippedAsset),
		qualifiedSnapshot: make(map[uint32]bool),
		initialized:       false,
	}
}
```

- [ ] **Step 3: Add new builder methods (append at the end of the existing `With*` block, just before `ComputeEffectiveStats`)**

```go
// WithWearer returns a new model with an updated wearer profile.
func (m Model) WithWearer(p WearerProfile) Model {
	out := m.shallowCopy()
	out.wearer = p
	return out
}

// WithEquippedAsset overwrites (or inserts) the snapshot keyed by asset id.
func (m Model) WithEquippedAsset(a EquippedAsset) Model {
	out := m.shallowCopy()
	out.equipped = copyEquipped(m.equipped)
	out.equipped[a.AssetId()] = a
	return out
}

// WithoutEquippedAsset removes the snapshot for the given asset id.
func (m Model) WithoutEquippedAsset(assetId uint32) Model {
	out := m.shallowCopy()
	out.equipped = copyEquipped(m.equipped)
	delete(out.equipped, assetId)
	return out
}

// withQualifiedSnapshot is package-private — only RecomputeWith should call it.
func (m Model) withQualifiedSnapshot(q map[uint32]bool) Model {
	out := m.shallowCopy()
	out.qualifiedSnapshot = q
	return out
}

func (m Model) shallowCopy() Model {
	return Model{
		tenant:            m.tenant,
		ch:                m.ch,
		characterId:       m.characterId,
		baseStats:         m.baseStats,
		bonuses:           m.bonuses,
		wearer:            m.wearer,
		equipped:          m.equipped,
		qualifiedSnapshot: m.qualifiedSnapshot,
		computed:          m.computed,
		lastUpdated:       m.lastUpdated,
		initialized:       m.initialized,
	}
}

func copyEquipped(src map[uint32]EquippedAsset) map[uint32]EquippedAsset {
	out := make(map[uint32]EquippedAsset, len(src)+1)
	for k, v := range src {
		out[k] = v
	}
	return out
}
```

- [ ] **Step 4: Update every existing `With*` builder to preserve the new fields**

Search-and-replace the existing builder bodies in `character/model.go`. Each builder currently reconstructs `Model{...}` literally. Update each one to also carry the new fields. For example, the post-edit `WithBaseStats` is:

```go
// WithBaseStats returns a new model with updated base stats
func (m Model) WithBaseStats(base stat.Base) Model {
	out := m.shallowCopy()
	out.baseStats = base
	return out
}
```

Apply the same pattern to **`WithBonus`**, **`WithBonuses`**, **`WithoutBonus`**, **`WithoutBonusesBySource`**, **`WithComputed`**, and **`WithInitialized`** — each computes its mutation, then uses `shallowCopy` to inherit everything else. The post-edit forms:

```go
func (m Model) WithBonus(b stat.Bonus) Model {
	newBonuses := make([]stat.Bonus, 0, len(m.bonuses)+1)
	for _, existing := range m.bonuses {
		if existing.Source() != b.Source() || existing.StatType() != b.StatType() {
			newBonuses = append(newBonuses, existing)
		}
	}
	newBonuses = append(newBonuses, b)
	out := m.shallowCopy()
	out.bonuses = newBonuses
	return out
}

func (m Model) WithBonuses(bonuses []stat.Bonus) Model {
	result := m
	for _, b := range bonuses {
		result = result.WithBonus(b)
	}
	return result
}

func (m Model) WithoutBonus(source string, statType stat.Type) Model {
	newBonuses := make([]stat.Bonus, 0, len(m.bonuses))
	for _, existing := range m.bonuses {
		if existing.Source() != source || existing.StatType() != statType {
			newBonuses = append(newBonuses, existing)
		}
	}
	out := m.shallowCopy()
	out.bonuses = newBonuses
	return out
}

func (m Model) WithoutBonusesBySource(source string) Model {
	newBonuses := make([]stat.Bonus, 0, len(m.bonuses))
	for _, existing := range m.bonuses {
		if existing.Source() != source {
			newBonuses = append(newBonuses, existing)
		}
	}
	out := m.shallowCopy()
	out.bonuses = newBonuses
	return out
}

func (m Model) WithComputed(computed stat.Computed) Model {
	out := m.shallowCopy()
	out.computed = computed
	out.lastUpdated = time.Now()
	return out
}

func (m Model) WithInitialized() Model {
	out := m.shallowCopy()
	out.initialized = true
	return out
}
```

- [ ] **Step 5: Run all character tests; expect PASS (existing tests don't reference the new fields)**

```bash
go test ./character/ -v
```

If any existing test fails, the most likely cause is a builder dropping `bonuses`/`baseStats`/etc. Reread step 4 — every builder must call `shallowCopy()` first.

- [ ] **Step 6: Commit**

```bash
git add character/model.go
git commit -m "feat(atlas-effective-stats): add wearer + equipped fields and builders to Model"
```

---

## Task 5: JSON marshal/unmarshal for new Model fields

**Why:** The registry persists `Model` to Redis as JSON. Without round-tripping the new fields, every Get loses `wearer`/`equipped`/`qualifiedSnapshot`.

**Files:**
- Modify: `character/model.go`
- Test: `character/model_test.go` (extend)

- [ ] **Step 1: Write the failing round-trip test**

Append to `character/model_test.go`:

```go
func TestModel_JSONRoundTrip_PreservesWearerAndEquipped(t *testing.T) {
	tn, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ch := channel.NewModel(0, 0)
	m := NewModel(tn, ch, 12345).
		WithBaseStats(stat.NewBase(4, 25, 39, 4, 1430, 6330)).
		WithWearer(NewWearerProfile(35, job.Id(200))).
		WithEquippedAsset(NewEquippedAsset(42, 1052095, []stat.Bonus{
			stat.NewBonus("equipment:42", stat.TypeMaxMp, 50),
		})).
		withQualifiedSnapshot(map[uint32]bool{42: true})

	data, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var got Model
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if got.Wearer().Level() != 35 || got.Wearer().JobId() != job.Id(200) {
		t.Errorf("wearer not preserved: %+v", got.Wearer())
	}
	eq := got.Equipped()
	if len(eq) != 1 {
		t.Fatalf("equipped len = %d, want 1", len(eq))
	}
	asset := eq[42]
	if asset.TemplateId() != 1052095 {
		t.Errorf("template not preserved: %d", asset.TemplateId())
	}
	if len(asset.Bonuses()) != 1 || asset.Bonuses()[0].Amount() != 50 {
		t.Errorf("snapshot bonuses not preserved: %+v", asset.Bonuses())
	}
	if !got.qualifiedSnapshot[42] {
		t.Errorf("qualifiedSnapshot not preserved: %+v", got.qualifiedSnapshot)
	}
}
```

You will need to add these imports to `model_test.go` if not already present: `encoding/json`, `github.com/Chronicle20/atlas/libs/atlas-constants/channel`, `github.com/Chronicle20/atlas/libs/atlas-constants/job`, `github.com/Chronicle20/atlas/libs/atlas-tenant`, `github.com/google/uuid`.

- [ ] **Step 2: Run the test; expect FAIL**

```bash
go test ./character/ -run TestModel_JSONRoundTrip_PreservesWearerAndEquipped -v
```

The fields are dropped by the existing `MarshalJSON` / `UnmarshalJSON`.

- [ ] **Step 3: Implement marshal/unmarshal extensions**

Replace the entire `MarshalJSON` and `UnmarshalJSON` block at the bottom of `character/model.go`:

```go
type wearerJSON struct {
	Level byte   `json:"level"`
	JobId job.Id `json:"jobId"`
}

type equippedAssetJSON struct {
	AssetId    uint32       `json:"assetId"`
	TemplateId uint32       `json:"templateId"`
	Bonuses    []stat.Bonus `json:"bonuses"`
}

func (m Model) MarshalJSON() ([]byte, error) {
	eq := make(map[string]equippedAssetJSON, len(m.equipped))
	for id, snap := range m.equipped {
		eq[strconv.FormatUint(uint64(id), 10)] = equippedAssetJSON{
			AssetId:    snap.assetId,
			TemplateId: snap.templateId,
			Bonuses:    append([]stat.Bonus(nil), snap.bonuses...),
		}
	}
	qs := make(map[string]bool, len(m.qualifiedSnapshot))
	for id, ok := range m.qualifiedSnapshot {
		qs[strconv.FormatUint(uint64(id), 10)] = ok
	}
	return json.Marshal(struct {
		WorldId           world.Id                     `json:"worldId"`
		ChannelId         channel.Id                   `json:"channelId"`
		CharacterId       uint32                       `json:"characterId"`
		BaseStats         stat.Base                    `json:"baseStats"`
		Bonuses           []stat.Bonus                 `json:"bonuses"`
		Wearer            wearerJSON                   `json:"wearer"`
		Equipped          map[string]equippedAssetJSON `json:"equipped"`
		QualifiedSnapshot map[string]bool              `json:"qualifiedSnapshot"`
		Computed          stat.Computed                `json:"computed"`
		LastUpdated       time.Time                    `json:"lastUpdated"`
		Initialized       bool                         `json:"initialized"`
	}{
		WorldId:           m.ch.WorldId(),
		ChannelId:         m.ch.Id(),
		CharacterId:       m.characterId,
		BaseStats:         m.baseStats,
		Bonuses:           m.bonuses,
		Wearer:            wearerJSON{Level: m.wearer.level, JobId: m.wearer.jobId},
		Equipped:          eq,
		QualifiedSnapshot: qs,
		Computed:          m.computed,
		LastUpdated:       m.lastUpdated,
		Initialized:       m.initialized,
	})
}

func (m *Model) UnmarshalJSON(data []byte) error {
	var aux struct {
		WorldId           world.Id                     `json:"worldId"`
		ChannelId         channel.Id                   `json:"channelId"`
		CharacterId       uint32                       `json:"characterId"`
		BaseStats         stat.Base                    `json:"baseStats"`
		Bonuses           []stat.Bonus                 `json:"bonuses"`
		Wearer            wearerJSON                   `json:"wearer"`
		Equipped          map[string]equippedAssetJSON `json:"equipped"`
		QualifiedSnapshot map[string]bool              `json:"qualifiedSnapshot"`
		Computed          stat.Computed                `json:"computed"`
		LastUpdated       time.Time                    `json:"lastUpdated"`
		Initialized       bool                         `json:"initialized"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	m.ch = channel.NewModel(aux.WorldId, aux.ChannelId)
	m.characterId = aux.CharacterId
	m.baseStats = aux.BaseStats
	if aux.Bonuses == nil {
		m.bonuses = make([]stat.Bonus, 0)
	} else {
		m.bonuses = aux.Bonuses
	}
	m.wearer = WearerProfile{level: aux.Wearer.Level, jobId: aux.Wearer.JobId}
	m.equipped = make(map[uint32]EquippedAsset, len(aux.Equipped))
	for _, snap := range aux.Equipped {
		m.equipped[snap.AssetId] = EquippedAsset{
			assetId:    snap.AssetId,
			templateId: snap.TemplateId,
			bonuses:    append([]stat.Bonus(nil), snap.Bonuses...),
		}
	}
	m.qualifiedSnapshot = make(map[uint32]bool, len(aux.QualifiedSnapshot))
	for k, v := range aux.QualifiedSnapshot {
		id, err := strconv.ParseUint(k, 10, 32)
		if err != nil {
			continue
		}
		m.qualifiedSnapshot[uint32(id)] = v
	}
	m.computed = aux.Computed
	m.lastUpdated = aux.LastUpdated
	m.initialized = aux.Initialized
	return nil
}
```

Add `strconv` and `github.com/Chronicle20/atlas/libs/atlas-constants/job` to the import block at the top of `character/model.go`.

> **Why string-keyed maps:** Go's `encoding/json` requires `map[string]T` for objects; a `map[uint32]T` is rejected at marshal time.

- [ ] **Step 4: Run, expect PASS for the new test and all existing model tests**

```bash
go test ./character/ -v
```

- [ ] **Step 5: Commit**

```bash
git add character/model.go character/model_test.go
git commit -m "feat(atlas-effective-stats): persist wearer/equipped/qualifiedSnapshot through JSON round-trip"
```

---

## Task 6: Equipment data REST model

**Files:**
- Create: `external/data/equipment/rest.go`

- [ ] **Step 1: Write the failing test**

Create `external/data/equipment/rest_test.go`:

```go
package equipment

import (
	"encoding/json"
	"testing"
)

func TestRestModel_DecodesAtlasDataFields(t *testing.T) {
	body := []byte(`{"reqLevel":40,"reqJob":2,"reqStr":0,"reqDex":0,"reqInt":80,"reqLuk":40}`)
	var rm RestModel
	if err := json.Unmarshal(body, &rm); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if rm.ReqLevel != 40 || rm.ReqJob != 2 || rm.ReqInt != 80 || rm.ReqLuk != 40 {
		t.Errorf("decode mismatch: %+v", rm)
	}
}

func TestRestModel_GetNameIsEquipment(t *testing.T) {
	if (RestModel{}).GetName() != "equipment" {
		t.Errorf("GetName mismatch")
	}
}

func TestRestModel_IDRoundTrip(t *testing.T) {
	var rm RestModel
	if err := rm.SetID("1052095"); err != nil {
		t.Fatalf("SetID: %v", err)
	}
	if rm.GetID() != "1052095" {
		t.Errorf("GetID = %s", rm.GetID())
	}
}
```

- [ ] **Step 2: Run, expect FAIL (`undefined: RestModel`)**

```bash
go test ./external/data/equipment/ -v
```

- [ ] **Step 3: Implement**

Create `external/data/equipment/rest.go`:

```go
package equipment

import "strconv"

// RestModel mirrors the requirement subset of atlas-data's equipment endpoint.
// atlas-data exposes many more fields (per-stat bonuses, classification, etc.)
// but atlas-effective-stats only needs the requirement gate inputs — per-asset
// stats already arrive via atlas-inventory.
type RestModel struct {
	Id       uint32 `json:"-"`
	ReqLevel byte   `json:"reqLevel"`
	ReqJob   uint16 `json:"reqJob"`
	ReqStr   uint16 `json:"reqStr"`
	ReqDex   uint16 `json:"reqDex"`
	ReqInt   uint16 `json:"reqInt"`
	ReqLuk   uint16 `json:"reqLuk"`
}

func (r RestModel) GetName() string { return "equipment" }

func (r RestModel) GetID() string { return strconv.Itoa(int(r.Id)) }

func (r *RestModel) SetID(idStr string) error {
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}
```

- [ ] **Step 4: Run, expect PASS**

```bash
go test ./external/data/equipment/ -v
```

- [ ] **Step 5: Commit**

```bash
git add external/data/equipment/rest.go external/data/equipment/rest_test.go
git commit -m "feat(atlas-effective-stats): add equipment REST model for requirement fields"
```

---

## Task 7: Equipment data REST request builder

**Files:**
- Create: `external/data/equipment/requests.go`

- [ ] **Step 1: Implement**

Create `external/data/equipment/requests.go`:

```go
package equipment

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	Resource      = "data/equipment"
	EquipmentById = Resource + "/%d"
)

func getBaseRequest() string {
	return requests.RootUrl("DATA")
}

// RequestById returns a request to fetch equipment data by template ID from
// the atlas-data service. Tenant header propagation is handled by the request
// decorator chain.
func RequestById(templateId uint32) requests.Request[RestModel] {
	return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+"/"+EquipmentById, templateId))
}
```

(Mirrors `external/data/skill/requests.go`. No new test — exercised in Task 9 via the cache.)

- [ ] **Step 2: Build**

```bash
go build ./...
```
Expected: clean build.

- [ ] **Step 3: Commit**

```bash
git add external/data/equipment/requests.go
git commit -m "feat(atlas-effective-stats): add equipment REST request builder"
```

---

## Task 8: Equipment requirements cache + Provider closure

**Files:**
- Create: `external/data/equipment/cache.go`

- [ ] **Step 1: Implement**

Create `external/data/equipment/cache.go`:

```go
package equipment

import (
	"context"
	"sync"

	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
)

// EquipmentRequirements holds the six gating fields read from atlas-data.
type EquipmentRequirements struct {
	ReqLevel byte
	ReqJob   uint16 // v83 raw bitmask: 0=no restriction, 1=Warrior, 2=Mage, 4=Bowman, 8=Thief, 16=Pirate
	ReqStr   uint16
	ReqDex   uint16
	ReqInt   uint16
	ReqLuk   uint16
}

// Provider returns the requirements for a template id. The bool is false when
// the lookup failed and there is no cached value; callers MUST treat that as
// "this asset does not qualify for this evaluation".
type Provider func(ctx context.Context, templateId uint32) (EquipmentRequirements, bool)

// fetcher is the indirection point for tests (Task 9 swaps it).
type fetcher func(ctx context.Context, l logrus.FieldLogger, templateId uint32) (EquipmentRequirements, error)

var defaultFetcher fetcher = func(ctx context.Context, l logrus.FieldLogger, templateId uint32) (EquipmentRequirements, error) {
	rm, err := RequestById(templateId)(l, ctx)
	if err != nil {
		return EquipmentRequirements{}, err
	}
	return EquipmentRequirements{
		ReqLevel: rm.ReqLevel,
		ReqJob:   rm.ReqJob,
		ReqStr:   rm.ReqStr,
		ReqDex:   rm.ReqDex,
		ReqInt:   rm.ReqInt,
		ReqLuk:   rm.ReqLuk,
	}, nil
}

type cache struct {
	mu    sync.RWMutex
	store map[tenant.Id]map[uint32]EquipmentRequirements
}

var (
	once sync.Once
	inst *cache
)

func getCache() *cache {
	once.Do(func() {
		inst = &cache{store: make(map[tenant.Id]map[uint32]EquipmentRequirements)}
	})
	return inst
}

func (c *cache) get(tID tenant.Id, templateId uint32) (EquipmentRequirements, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	t, ok := c.store[tID]
	if !ok {
		return EquipmentRequirements{}, false
	}
	r, ok := t[templateId]
	return r, ok
}

func (c *cache) put(tID tenant.Id, templateId uint32, r EquipmentRequirements) {
	c.mu.Lock()
	defer c.mu.Unlock()
	t, ok := c.store[tID]
	if !ok {
		t = make(map[uint32]EquipmentRequirements)
		c.store[tID] = t
	}
	t[templateId] = r
}

// reset is exposed package-internally for tests; production callers never
// invoke it.
func (c *cache) reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.store = make(map[tenant.Id]map[uint32]EquipmentRequirements)
}

// GetProvider returns a Provider closure bound to the given logger. The
// closure consults the per-tenant cache first; on cold-cache miss it fetches
// from atlas-data, caches success, and logs WARN on failure. Returning
// (_, false) means "treat this asset as unqualified for this evaluation".
func GetProvider(l logrus.FieldLogger) Provider {
	return func(ctx context.Context, templateId uint32) (EquipmentRequirements, bool) {
		t := tenant.MustFromContext(ctx)
		if r, ok := getCache().get(t.Id(), templateId); ok {
			return r, true
		}
		r, err := defaultFetcher(ctx, l, templateId)
		if err != nil {
			l.WithError(err).Warnf("equipment template [%d] fetch failed; treating dependent assets as unqualified", templateId)
			return EquipmentRequirements{}, false
		}
		getCache().put(t.Id(), templateId, r)
		return r, true
	}
}
```

- [ ] **Step 2: Build**

```bash
go build ./...
```

- [ ] **Step 3: Commit**

```bash
git add external/data/equipment/cache.go
git commit -m "feat(atlas-effective-stats): add per-tenant equipment requirements cache"
```

---

## Task 9: Equipment cache tests (hit/miss/tenant isolation)

**Files:**
- Create: `external/data/equipment/cache_test.go`

- [ ] **Step 1: Write the failing tests**

Create `external/data/equipment/cache_test.go`:

```go
package equipment

import (
	"context"
	"errors"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
)

// withFetcher swaps the package-level fetcher for the duration of a single
// test. Returns the call counter so tests can assert how many fetches landed.
func withFetcher(t *testing.T, fn func(ctx context.Context, templateId uint32) (EquipmentRequirements, error)) *int {
	t.Helper()
	prev := defaultFetcher
	calls := 0
	defaultFetcher = func(_ context.Context, _ logger, id uint32) (EquipmentRequirements, error) {
		calls++
		return fn(context.Background(), id)
	}
	t.Cleanup(func() { defaultFetcher = prev; getCache().reset() })
	return &calls
}

// We need an alias for the logger param so the closure signature in
// withFetcher matches `fetcher`. Use the logrus.FieldLogger via a tiny
// adapter type.
//
//nolint:unused
type logger = interface {
	WithError(err error) interface{ Warnf(format string, args ...interface{}) }
	Warnf(format string, args ...interface{})
}
```

(Stop here — that adapter type approach won't compile because logrus uses concrete types. Replace the helper with a direct fetcher swap. Restart this file with the simpler form below.)

Replace the file contents with:

```go
package equipment

import (
	"context"
	"errors"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
)

func swapFetcher(t *testing.T, fn fetcher) *int {
	t.Helper()
	prev := defaultFetcher
	calls := 0
	defaultFetcher = func(ctx context.Context, l logrus.FieldLogger, id uint32) (EquipmentRequirements, error) {
		calls++
		return fn(ctx, l, id)
	}
	t.Cleanup(func() {
		defaultFetcher = prev
		getCache().reset()
	})
	return &calls
}

func tenantContext(t *testing.T, region string) (context.Context, tenant.Model) {
	t.Helper()
	tn, err := tenant.Create(uuid.New(), region, 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}
	return tenant.WithContext(context.Background(), tn), tn
}

func TestProvider_CacheHitOnSecondCall(t *testing.T) {
	l, _ := test.NewNullLogger()
	calls := swapFetcher(t, func(_ context.Context, _ logrus.FieldLogger, id uint32) (EquipmentRequirements, error) {
		return EquipmentRequirements{ReqLuk: 40}, nil
	})

	ctx, _ := tenantContext(t, "GMS")
	p := GetProvider(l)
	if r, ok := p(ctx, 1052095); !ok || r.ReqLuk != 40 {
		t.Fatalf("first call: ok=%v r=%+v", ok, r)
	}
	if r, ok := p(ctx, 1052095); !ok || r.ReqLuk != 40 {
		t.Fatalf("second call: ok=%v r=%+v", ok, r)
	}
	if *calls != 1 {
		t.Errorf("fetch count = %d, want 1", *calls)
	}
}

func TestProvider_ColdCacheFetchFailureReturnsFalse(t *testing.T) {
	l, _ := test.NewNullLogger()
	calls := swapFetcher(t, func(_ context.Context, _ logrus.FieldLogger, id uint32) (EquipmentRequirements, error) {
		return EquipmentRequirements{}, errors.New("boom")
	})

	ctx, _ := tenantContext(t, "GMS")
	p := GetProvider(l)
	if _, ok := p(ctx, 1052095); ok {
		t.Errorf("expected (_, false) on cold-cache fetch failure")
	}
	if _, ok := p(ctx, 1052095); ok {
		t.Errorf("expected (_, false) on second cold-cache fetch failure too")
	}
	if *calls != 2 {
		t.Errorf("fetch count = %d, want 2 (cache only stores success)", *calls)
	}
}

func TestProvider_TenantIsolation(t *testing.T) {
	l, _ := test.NewNullLogger()
	calls := swapFetcher(t, func(_ context.Context, _ logrus.FieldLogger, id uint32) (EquipmentRequirements, error) {
		return EquipmentRequirements{ReqLuk: 40}, nil
	})

	ctxA, _ := tenantContext(t, "GMS")
	ctxB, _ := tenantContext(t, "JMS")
	p := GetProvider(l)
	_, _ = p(ctxA, 1052095)
	_, _ = p(ctxB, 1052095)
	if *calls != 2 {
		t.Errorf("fetch count = %d, want 2 (per-tenant)", *calls)
	}
}
```

- [ ] **Step 2: Run, expect FAIL until cache.go exists from Task 8 — should PASS now**

```bash
go test ./external/data/equipment/ -v
```

- [ ] **Step 3: Commit**

```bash
git add external/data/equipment/cache_test.go
git commit -m "test(atlas-effective-stats): cache hit/miss/tenant-isolation tests for equipment provider"
```

---

## Task 10: `meetsRequirements` + `wearerClassMask` helpers

**Why:** Pure functions, isolated from the iteration logic; lets us tightly test the bitmask gotcha.

**Files:**
- Create: `character/qualification.go`
- Create: `character/qualification_test.go`

- [ ] **Step 1: Write the failing tests**

Create `character/qualification_test.go`:

```go
package character

import (
	"testing"

	"atlas-effective-stats/external/data/equipment"

	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
)

func TestWearerClassMask_StandardClasses(t *testing.T) {
	cases := []struct {
		name string
		id   job.Id
		want uint16
	}{
		{"Beginner", 0, 0},
		{"Warrior 1st", 100, 1},
		{"Fighter 2nd", 110, 1},
		{"Crusader 3rd", 111, 1},
		{"Hero 4th", 112, 1},
		{"Magician 1st", 200, 2},
		{"FP Wizard 2nd", 210, 2},
		{"Bowman 1st", 300, 4},
		{"Thief 1st", 400, 8},
		{"Pirate 1st", 500, 16},
		{"DawnWarrior 1st", 1100, 1},
		{"BlazeWizard 1st", 1200, 2},
		{"WindArcher 1st", 1300, 4},
		{"NightWalker 1st", 1400, 8},
		{"ThunderBreaker 1st", 1500, 16},
		{"Aran 1st (2100)", 2100, 1},
		{"Evan 2nd (2200)", 2200, 2},
		{"Noblesse beginner", 1000, 0},
		{"Legend beginner", 2000, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := wearerClassMask(c.id); got != c.want {
				t.Errorf("mask(%d) = %d, want %d", c.id, got, c.want)
			}
		})
	}
}

func TestMeetsRequirements_AllZerosAlwaysPass(t *testing.T) {
	r := equipment.EquipmentRequirements{}
	if !meetsRequirements(r, AppliedStats{}, 0, job.Id(0)) {
		t.Error("zero reqs should always pass, even with zero wearer")
	}
}

func TestMeetsRequirements_LevelGate(t *testing.T) {
	r := equipment.EquipmentRequirements{ReqLevel: 30}
	if meetsRequirements(r, AppliedStats{}, 29, job.Id(100)) {
		t.Error("level 29 should fail reqLevel=30")
	}
	if !meetsRequirements(r, AppliedStats{}, 30, job.Id(100)) {
		t.Error("level 30 should pass reqLevel=30")
	}
	if !meetsRequirements(r, AppliedStats{}, 31, job.Id(100)) {
		t.Error("level 31 should pass reqLevel=30")
	}
}

func TestMeetsRequirements_JobBitmask(t *testing.T) {
	// Magician-only item.
	r := equipment.EquipmentRequirements{ReqJob: 2}
	if meetsRequirements(r, AppliedStats{}, 1, job.Id(100)) {
		t.Error("Warrior should not pass Magician-only item")
	}
	if !meetsRequirements(r, AppliedStats{}, 1, job.Id(200)) {
		t.Error("Magician should pass Magician-only item")
	}
	if meetsRequirements(r, AppliedStats{}, 1, job.Id(0)) {
		t.Error("Beginner (mask 0) should not pass class-restricted item")
	}
	// Cross-class (Warrior | Magician).
	rCross := equipment.EquipmentRequirements{ReqJob: 1 | 2}
	if !meetsRequirements(rCross, AppliedStats{}, 1, job.Id(100)) {
		t.Error("Warrior should pass W|M cross-class item")
	}
	if !meetsRequirements(rCross, AppliedStats{}, 1, job.Id(200)) {
		t.Error("Magician should pass W|M cross-class item")
	}
	if meetsRequirements(rCross, AppliedStats{}, 1, job.Id(300)) {
		t.Error("Bowman should not pass W|M cross-class item")
	}
}

func TestMeetsRequirements_StatGates_OffByOne(t *testing.T) {
	r := equipment.EquipmentRequirements{ReqStr: 100, ReqDex: 50, ReqInt: 10, ReqLuk: 40}
	pass := AppliedStats{Strength: 100, Dexterity: 50, Intelligence: 10, Luck: 40}
	if !meetsRequirements(r, pass, 1, job.Id(100)) {
		t.Error("exact match should pass")
	}
	below := AppliedStats{Strength: 99, Dexterity: 50, Intelligence: 10, Luck: 40}
	if meetsRequirements(r, below, 1, job.Id(100)) {
		t.Error("STR-1 should fail")
	}
	below = AppliedStats{Strength: 100, Dexterity: 49, Intelligence: 10, Luck: 40}
	if meetsRequirements(r, below, 1, job.Id(100)) {
		t.Error("DEX-1 should fail")
	}
	below = AppliedStats{Strength: 100, Dexterity: 50, Intelligence: 9, Luck: 40}
	if meetsRequirements(r, below, 1, job.Id(100)) {
		t.Error("INT-1 should fail")
	}
	below = AppliedStats{Strength: 100, Dexterity: 50, Intelligence: 10, Luck: 39}
	if meetsRequirements(r, below, 1, job.Id(100)) {
		t.Error("LUK-1 should fail (the diagnosis case)")
	}
}
```

- [ ] **Step 2: Run, expect FAIL (`undefined: wearerClassMask`, etc.)**

```bash
go test ./character/ -run "TestWearerClassMask|TestMeetsRequirements" -v
```

- [ ] **Step 3: Implement**

Create `character/qualification.go`:

```go
package character

import (
	"atlas-effective-stats/external/data/equipment"

	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
)

// AppliedStats is the per-evaluation snapshot of wearer numeric stats used
// to test equipment requirements. It is the sum of base stats + always-on
// (buff/passive) flat bonuses + flat bonuses from the currently-qualifying
// equipment subset.
type AppliedStats struct {
	Strength     uint32
	Dexterity    uint32
	Intelligence uint32
	Luck         uint32
}

// wearerClassMask maps an internal atlas job id to the v83 reqJob bitmask.
// atlas internal jobIds are NOT raw v83 client bitmasks (Magician 1st = 200,
// not 2), so a direct AND would silently misqualify every class-restricted
// item. This helper centralises the mapping.
//
// v83 bits: Warrior=1, Magician=2, Bowman=4, Thief=8, Pirate=16. Beginner
// classes (no class restriction in v83 reqJob semantics) map to 0.
func wearerClassMask(id job.Id) uint16 {
	branch := uint16(id) / 100
	switch branch {
	case 0, 10, 20: // Beginner / Noblesse / Legend
		return 0
	case 1, 11, 12, 21: // Warrior, DawnWarrior, Aran (jobId/100 = 21)
		return 1
	case 2, 22: // Magician, Evan (jobId/100 = 22)
		return 2
	case 3, 13: // Bowman, WindArcher
		return 4
	case 4, 14: // Thief, NightWalker
		return 8
	case 5, 15: // Pirate, ThunderBreaker
		return 16
	default:
		return 0
	}
}

// meetsRequirements returns true when the wearer satisfies every populated
// requirement on the equipment template. A zero req is "no restriction".
func meetsRequirements(r equipment.EquipmentRequirements, s AppliedStats, level byte, jobId job.Id) bool {
	if r.ReqLevel > 0 && level < r.ReqLevel {
		return false
	}
	if r.ReqJob > 0 && wearerClassMask(jobId)&r.ReqJob == 0 {
		return false
	}
	if r.ReqStr > 0 && s.Strength < uint32(r.ReqStr) {
		return false
	}
	if r.ReqDex > 0 && s.Dexterity < uint32(r.ReqDex) {
		return false
	}
	if r.ReqInt > 0 && s.Intelligence < uint32(r.ReqInt) {
		return false
	}
	if r.ReqLuk > 0 && s.Luck < uint32(r.ReqLuk) {
		return false
	}
	return true
}
```

- [ ] **Step 4: Run, expect PASS**

```bash
go test ./character/ -run "TestWearerClassMask|TestMeetsRequirements" -v
```

- [ ] **Step 5: Commit**

```bash
git add character/qualification.go character/qualification_test.go
git commit -m "feat(atlas-effective-stats): add meetsRequirements + wearerClassMask helpers"
```

---

## Task 11: `Model.QualifiedEquipment` fixed-point iteration

**Files:**
- Modify: `character/qualification.go` (add method)
- Modify: `character/qualification_test.go` (add tests)

- [ ] **Step 1: Write the failing tests**

Append to `character/qualification_test.go`:

```go
import (
	"context"

	"atlas-effective-stats/stat"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
)
```

(If your import block already exists, merge these in rather than adding a second `import` block.)

Then append the tests:

```go
func newTestModel(t *testing.T, base stat.Base, wp WearerProfile, snaps ...EquippedAsset) Model {
	t.Helper()
	tn, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}
	m := NewModel(tn, channel.NewModel(0, 0), 12345).
		WithBaseStats(base).
		WithWearer(wp)
	for _, s := range snaps {
		m = m.WithEquippedAsset(s)
	}
	return m
}

// providerOf builds a stub Provider for the given templates. Missing entries
// return (_, false), simulating an atlas-data fetch failure.
func providerOf(reqs map[uint32]equipment.EquipmentRequirements) equipment.Provider {
	return func(_ context.Context, id uint32) (equipment.EquipmentRequirements, bool) {
		r, ok := reqs[id]
		return r, ok
	}
}

func TestQualifiedEquipment_EmptyEquippedReturnsEmpty(t *testing.T) {
	m := newTestModel(t, stat.NewBase(0, 0, 0, 0, 0, 0), NewWearerProfile(30, job.Id(100)))
	got := m.QualifiedEquipment(providerOf(nil), context.Background())
	if len(got) != 0 {
		t.Errorf("expected empty map, got %v", got)
	}
}

func TestQualifiedEquipment_DiagnosisCase_LukBelowReq(t *testing.T) {
	overall := NewEquippedAsset(42, 1052095, []stat.Bonus{
		stat.NewBonus("equipment:42", stat.TypeMaxMp, 50),
	})
	base := stat.NewBase(4, 25, 39 /*luk*/, 4, 1430, 6330)
	m := newTestModel(t, base, NewWearerProfile(30, job.Id(200)), overall)
	prov := providerOf(map[uint32]equipment.EquipmentRequirements{
		1052095: {ReqLuk: 40},
	})
	got := m.QualifiedEquipment(prov, context.Background())
	if got[42] {
		t.Error("LUK 39 should NOT qualify reqLuk=40 (diagnosis case)")
	}
}

func TestQualifiedEquipment_DiagnosisCase_LukAtReq(t *testing.T) {
	overall := NewEquippedAsset(42, 1052095, []stat.Bonus{
		stat.NewBonus("equipment:42", stat.TypeMaxMp, 50),
	})
	base := stat.NewBase(4, 25, 40 /*luk*/, 4, 1430, 6330)
	m := newTestModel(t, base, NewWearerProfile(30, job.Id(200)), overall)
	prov := providerOf(map[uint32]equipment.EquipmentRequirements{
		1052095: {ReqLuk: 40},
	})
	got := m.QualifiedEquipment(prov, context.Background())
	if !got[42] {
		t.Error("LUK 40 should qualify reqLuk=40")
	}
}

func TestQualifiedEquipment_ChainQualification(t *testing.T) {
	// A is no-req and grants +5 STR.
	// B requires STR>=base.STR+5, grants +5 STR.
	// C requires STR>=base.STR+10.
	a := NewEquippedAsset(1, 1001, []stat.Bonus{
		stat.NewBonus("equipment:1", stat.TypeStrength, 5),
	})
	b := NewEquippedAsset(2, 1002, []stat.Bonus{
		stat.NewBonus("equipment:2", stat.TypeStrength, 5),
	})
	c := NewEquippedAsset(3, 1003, nil)
	base := stat.NewBase(50, 0, 0, 0, 0, 0)
	m := newTestModel(t, base, NewWearerProfile(30, job.Id(100)), a, b, c)
	prov := providerOf(map[uint32]equipment.EquipmentRequirements{
		1001: {},
		1002: {ReqStr: 55},
		1003: {ReqStr: 60},
	})
	got := m.QualifiedEquipment(prov, context.Background())
	if !got[1] || !got[2] || !got[3] {
		t.Errorf("chain should converge to {1,2,3}; got %v", got)
	}
}

func TestQualifiedEquipment_MutualCycle_NeitherQualifies(t *testing.T) {
	// A grants +5 STR, requires DEX>=10.
	// B grants +5 DEX, requires STR>=55.
	// Base 50 STR / 5 DEX → only the OTHER's bonus would unlock either.
	// Per design, neither bootstraps.
	a := NewEquippedAsset(1, 1001, []stat.Bonus{
		stat.NewBonus("equipment:1", stat.TypeStrength, 5),
	})
	b := NewEquippedAsset(2, 1002, []stat.Bonus{
		stat.NewBonus("equipment:2", stat.TypeDexterity, 5),
	})
	base := stat.NewBase(50, 5, 0, 0, 0, 0)
	m := newTestModel(t, base, NewWearerProfile(30, job.Id(100)), a, b)
	prov := providerOf(map[uint32]equipment.EquipmentRequirements{
		1001: {ReqDex: 10},
		1002: {ReqStr: 55},
	})
	got := m.QualifiedEquipment(prov, context.Background())
	if got[1] || got[2] {
		t.Errorf("mutual cycle should leave both unqualified; got %v", got)
	}
}

func TestQualifiedEquipment_ProviderFailureExcludesAsset(t *testing.T) {
	a := NewEquippedAsset(1, 1001, nil)
	b := NewEquippedAsset(2, 1002, nil)
	base := stat.NewBase(0, 0, 0, 0, 0, 0)
	m := newTestModel(t, base, NewWearerProfile(30, job.Id(100)), a, b)
	prov := providerOf(map[uint32]equipment.EquipmentRequirements{
		1002: {}, // 1001 deliberately missing → provider returns (_, false)
	})
	got := m.QualifiedEquipment(prov, context.Background())
	if got[1] {
		t.Error("provider miss should exclude asset 1")
	}
	if !got[2] {
		t.Error("asset 2 should still qualify")
	}
}

func TestQualifiedEquipment_BuffsAndPassivesContributeToApplied(t *testing.T) {
	// Equipment A requires STR>=110. Base STR=100. A buff:* and a passive:*
	// each add +5 STR — together they push effective STR to 110, making A
	// qualify even though equipment alone wouldn't.
	a := NewEquippedAsset(1, 1001, nil)
	base := stat.NewBase(100, 0, 0, 0, 0, 0)
	m := newTestModel(t, base, NewWearerProfile(30, job.Id(100)), a).
		WithBonus(stat.NewBonus("buff:9001", stat.TypeStrength, 5)).
		WithBonus(stat.NewBonus("passive:9002", stat.TypeStrength, 5))
	prov := providerOf(map[uint32]equipment.EquipmentRequirements{
		1001: {ReqStr: 110},
	})
	got := m.QualifiedEquipment(prov, context.Background())
	if !got[1] {
		t.Error("buff+passive should help asset 1 qualify")
	}
}
```

- [ ] **Step 2: Run, expect FAIL (`Model.QualifiedEquipment` undefined)**

```bash
go test ./character/ -run TestQualifiedEquipment -v
```

- [ ] **Step 3: Implement `QualifiedEquipment` and the helper used by it**

Append to `character/qualification.go`:

```go
import (
	"context"

	"atlas-effective-stats/stat"
)
```

Merge that with the existing `import (...)` block so there is exactly one import block at the top of the file. Then append:

```go
// QualifiedEquipment runs the fixed-point iteration described in design §4.3
// and returns the set of asset ids whose template requirements are satisfied
// under the wearer's base stats + non-equipment bonuses + the qualifying
// equipment subset itself.
//
// Provider failures (cold cache + atlas-data unreachable) drop the asset
// from this evaluation; callers do NOT see a separate error path.
func (m Model) QualifiedEquipment(reqProvider equipment.Provider, ctx context.Context) map[uint32]bool {
	qualified := make(map[uint32]bool, len(m.equipped))
	if len(m.equipped) == 0 {
		return qualified
	}

	flatNonEquip := sumFlatNonEquipBonuses(m.bonuses)

	computeApplied := func() AppliedStats {
		s := AppliedStats{
			Strength:     uint32max0(int32(m.baseStats.Strength()) + flatNonEquip[stat.TypeStrength]),
			Dexterity:    uint32max0(int32(m.baseStats.Dexterity()) + flatNonEquip[stat.TypeDexterity]),
			Intelligence: uint32max0(int32(m.baseStats.Intelligence()) + flatNonEquip[stat.TypeIntelligence]),
			Luck:         uint32max0(int32(m.baseStats.Luck()) + flatNonEquip[stat.TypeLuck]),
		}
		for assetId, snap := range m.equipped {
			if !qualified[assetId] {
				continue
			}
			for _, b := range snap.bonuses {
				switch b.StatType() {
				case stat.TypeStrength:
					s.Strength = addClamp(s.Strength, b.Amount())
				case stat.TypeDexterity:
					s.Dexterity = addClamp(s.Dexterity, b.Amount())
				case stat.TypeIntelligence:
					s.Intelligence = addClamp(s.Intelligence, b.Amount())
				case stat.TypeLuck:
					s.Luck = addClamp(s.Luck, b.Amount())
				}
			}
		}
		return s
	}

	for {
		applied := computeApplied()
		added := false
		for assetId, snap := range m.equipped {
			if qualified[assetId] {
				continue
			}
			req, ok := reqProvider(ctx, snap.templateId)
			if !ok {
				continue
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

func sumFlatNonEquipBonuses(bs []stat.Bonus) map[stat.Type]int32 {
	out := make(map[stat.Type]int32, 8)
	for _, b := range bs {
		out[b.StatType()] += b.Amount()
	}
	return out
}

func uint32max0(v int32) uint32 {
	if v < 0 {
		return 0
	}
	return uint32(v)
}

func addClamp(s uint32, delta int32) uint32 {
	if delta >= 0 {
		return s + uint32(delta)
	}
	d := uint32(-delta)
	if d > s {
		return 0
	}
	return s - d
}
```

- [ ] **Step 4: Run, expect PASS**

```bash
go test ./character/ -run TestQualifiedEquipment -v
```

- [ ] **Step 5: Run the entire character suite to confirm nothing else regressed**

```bash
go test ./character/ -v
```

- [ ] **Step 6: Commit**

```bash
git add character/qualification.go character/qualification_test.go
git commit -m "feat(atlas-effective-stats): add Model.QualifiedEquipment fixed-point iteration"
```

---

## Task 12: Reshape `ComputeEffectiveStats` and `Recompute` into `RecomputeWith`

**Why:** The compute path now needs to consult the qualifying snapshot subset, not the legacy `bonuses[]` for equipment.

**Files:**
- Modify: `character/model.go`
- Modify: `character/model_test.go` (extend)

- [ ] **Step 1: Write the failing test**

Append to `character/model_test.go`:

```go
func TestRecomputeWith_DropsUnqualifiedEquipmentFromComputed(t *testing.T) {
	tn, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ch := channel.NewModel(0, 0)
	a := NewEquippedAsset(42, 1052095, []stat.Bonus{
		stat.NewBonus("equipment:42", stat.TypeMaxMp, 50),
	})
	m := NewModel(tn, ch, 12345).
		WithBaseStats(stat.NewBase(4, 25, 39, 4, 1430, 6330)).
		WithWearer(NewWearerProfile(30, job.Id(200))).
		WithEquippedAsset(a)
	prov := func(_ context.Context, id uint32) (equipment.EquipmentRequirements, bool) {
		return equipment.EquipmentRequirements{ReqLuk: 40}, true
	}
	m = m.RecomputeWith(prov, tenant.WithContext(context.Background(), tn))
	if m.Computed().MaxMp() != 6330 {
		t.Errorf("MaxMp = %d, want 6330 (unqualified item dropped)", m.Computed().MaxMp())
	}
}

func TestRecomputeWith_IncludesQualifiedEquipmentInComputed(t *testing.T) {
	tn, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ch := channel.NewModel(0, 0)
	a := NewEquippedAsset(42, 1052095, []stat.Bonus{
		stat.NewBonus("equipment:42", stat.TypeMaxMp, 50),
	})
	m := NewModel(tn, ch, 12345).
		WithBaseStats(stat.NewBase(4, 25, 40, 4, 1430, 6330)).
		WithWearer(NewWearerProfile(30, job.Id(200))).
		WithEquippedAsset(a)
	prov := func(_ context.Context, id uint32) (equipment.EquipmentRequirements, bool) {
		return equipment.EquipmentRequirements{ReqLuk: 40}, true
	}
	m = m.RecomputeWith(prov, tenant.WithContext(context.Background(), tn))
	if m.Computed().MaxMp() != 6380 {
		t.Errorf("MaxMp = %d, want 6380", m.Computed().MaxMp())
	}
}
```

Add `"atlas-effective-stats/external/data/equipment"` and `"context"` to the test file's imports if missing.

- [ ] **Step 2: Run, expect FAIL (`Model.RecomputeWith` undefined)**

```bash
go test ./character/ -run TestRecomputeWith -v
```

- [ ] **Step 3: Implement**

Update `character/model.go`. Replace the existing `ComputeEffectiveStats` and `Recompute` (lines ~210-289 in the current file) with:

```go
// ComputeEffectiveStats calculates effective stats from base + non-equipment
// bonuses + the qualifying-equipment subset described by `qualified`.
//
// `qualified` is the output of QualifiedEquipment for the same model; passing
// a stale or wrong-sized map silently produces a stale Computed.
func (m Model) ComputeEffectiveStats(qualified map[uint32]bool) stat.Computed {
	baseValues := map[stat.Type]int32{
		stat.TypeStrength:      int32(m.baseStats.Strength()),
		stat.TypeDexterity:     int32(m.baseStats.Dexterity()),
		stat.TypeLuck:          int32(m.baseStats.Luck()),
		stat.TypeIntelligence:  int32(m.baseStats.Intelligence()),
		stat.TypeMaxHp:         int32(m.baseStats.MaxHp()),
		stat.TypeMaxMp:         int32(m.baseStats.MaxMp()),
		stat.TypeWeaponAttack:  0,
		stat.TypeWeaponDefense: 0,
		stat.TypeMagicAttack:   0,
		stat.TypeMagicDefense:  0,
		stat.TypeAccuracy:      0,
		stat.TypeAvoidability:  0,
		stat.TypeSpeed:         0,
		stat.TypeJump:          0,
	}

	flatBonuses := make(map[stat.Type]int32)
	multipliers := make(map[stat.Type]float64)
	for _, statType := range stat.AllTypes() {
		flatBonuses[statType] = 0
		multipliers[statType] = 0.0
	}

	// Non-equipment bonuses contribute both flat and multiplier values.
	for _, b := range m.bonuses {
		flatBonuses[b.StatType()] += b.Amount()
		multipliers[b.StatType()] += b.Multiplier()
	}

	// Equipment bonuses contribute only flat values (existing semantics) and
	// only for assets that survived the qualification gate.
	for assetId, snap := range m.equipped {
		if !qualified[assetId] {
			continue
		}
		for _, b := range snap.bonuses {
			flatBonuses[b.StatType()] += b.Amount()
		}
	}

	computeEffective := func(statType stat.Type) uint32 {
		base := baseValues[statType]
		flat := flatBonuses[statType]
		mult := multipliers[statType]

		effective := float64(base+flat) * (1.0 + mult)
		if effective < 0 {
			return 0
		}
		v := uint32(math.Floor(effective))
		if statType == stat.TypeMaxHp || statType == stat.TypeMaxMp {
			if v > MaxHpMpCap {
				v = MaxHpMpCap
			}
		}
		return v
	}

	return stat.NewComputed(
		computeEffective(stat.TypeStrength),
		computeEffective(stat.TypeDexterity),
		computeEffective(stat.TypeLuck),
		computeEffective(stat.TypeIntelligence),
		computeEffective(stat.TypeMaxHp),
		computeEffective(stat.TypeMaxMp),
		computeEffective(stat.TypeWeaponAttack),
		computeEffective(stat.TypeWeaponDefense),
		computeEffective(stat.TypeMagicAttack),
		computeEffective(stat.TypeMagicDefense),
		computeEffective(stat.TypeAccuracy),
		computeEffective(stat.TypeAvoidability),
		computeEffective(stat.TypeSpeed),
		computeEffective(stat.TypeJump),
	)
}

// Recompute recomputes effective stats assuming every equipped item
// qualifies. Use it only in unit tests or paths that have already gated
// equipment via legacy bonuses[]; production code should use RecomputeWith.
func (m Model) Recompute() Model {
	qualified := make(map[uint32]bool, len(m.equipped))
	for id := range m.equipped {
		qualified[id] = true
	}
	return m.WithComputed(m.ComputeEffectiveStats(qualified)).withQualifiedSnapshot(qualified)
}

// RecomputeWith runs the fixed-point qualification iteration via reqProvider,
// caches the qualifying set on the returned model, and rebuilds Computed
// from the qualifying subset.
func (m Model) RecomputeWith(reqProvider equipment.Provider, ctx context.Context) Model {
	qualified := m.QualifiedEquipment(reqProvider, ctx)
	return m.WithComputed(m.ComputeEffectiveStats(qualified)).withQualifiedSnapshot(qualified)
}
```

Add `"atlas-effective-stats/external/data/equipment"` and `"context"` to the import block at the top of `character/model.go`.

- [ ] **Step 4: Run, expect PASS for the new tests AND every existing model/processor test**

```bash
go test ./character/ -v
```

If `TestProcessor_*` tests now fail because `Recompute` semantics changed for equipment-via-`AddBonuses`, that's expected — the next tasks rewire the processor. For now, only the suite must compile and the model+qualification tests must pass. **If a processor test fails because `m.bonuses[]` still contains `equipment:*` entries left over from older code, that's the migration not being complete yet — proceed to Task 13.**

- [ ] **Step 5: Commit**

```bash
git add character/model.go character/model_test.go
git commit -m "feat(atlas-effective-stats): split ComputeEffectiveStats over qualifying subset; add RecomputeWith"
```

---

## Task 13: `Bonuses()` reads qualifying snapshots merged with `bonuses[]`

**Why:** REST consumers expect `bonuses[]` to include `equipment:<assetId>` entries for qualifying items. We rebuild that view from `m.equipped` filtered by `qualifiedSnapshot`.

**Files:**
- Modify: `character/model.go`
- Modify: `character/model_test.go` (extend)

- [ ] **Step 1: Write the failing test**

Append to `character/model_test.go`:

```go
func TestBonuses_OmitsUnqualifiedEquipment(t *testing.T) {
	tn, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ch := channel.NewModel(0, 0)
	a := NewEquippedAsset(42, 1052095, []stat.Bonus{
		stat.NewBonus("equipment:42", stat.TypeMaxMp, 50),
	})
	m := NewModel(tn, ch, 12345).
		WithBaseStats(stat.NewBase(4, 25, 39, 4, 1430, 6330)).
		WithWearer(NewWearerProfile(30, job.Id(200))).
		WithEquippedAsset(a).
		WithBonus(stat.NewBonus("buff:7", stat.TypeStrength, 5))
	prov := func(_ context.Context, id uint32) (equipment.EquipmentRequirements, bool) {
		return equipment.EquipmentRequirements{ReqLuk: 40}, true
	}
	m = m.RecomputeWith(prov, tenant.WithContext(context.Background(), tn))

	got := m.Bonuses()
	if len(got) != 1 || got[0].Source() != "buff:7" {
		t.Errorf("Bonuses() = %+v, want only buff:7 entry", got)
	}
}

func TestBonuses_IncludesQualifiedEquipment(t *testing.T) {
	tn, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ch := channel.NewModel(0, 0)
	a := NewEquippedAsset(42, 1052095, []stat.Bonus{
		stat.NewBonus("equipment:42", stat.TypeMaxMp, 50),
	})
	m := NewModel(tn, ch, 12345).
		WithBaseStats(stat.NewBase(4, 25, 40, 4, 1430, 6330)).
		WithWearer(NewWearerProfile(30, job.Id(200))).
		WithEquippedAsset(a)
	prov := func(_ context.Context, id uint32) (equipment.EquipmentRequirements, bool) {
		return equipment.EquipmentRequirements{ReqLuk: 40}, true
	}
	m = m.RecomputeWith(prov, tenant.WithContext(context.Background(), tn))

	got := m.Bonuses()
	if len(got) != 1 || got[0].Source() != "equipment:42" || got[0].Amount() != 50 {
		t.Errorf("Bonuses() = %+v, want one equipment:42 MaxMp=50 entry", got)
	}
}
```

- [ ] **Step 2: Run, expect FAIL (current `Bonuses()` ignores `m.equipped`)**

```bash
go test ./character/ -run "TestBonuses_OmitsUnqualifiedEquipment|TestBonuses_IncludesQualifiedEquipment" -v
```

- [ ] **Step 3: Implement**

Replace `Bonuses()` in `character/model.go`:

```go
// Bonuses reconstructs the flat list consumed by REST clients: every
// non-equipment bonus from m.bonuses, plus the snapshot bonuses for every
// asset in the most-recent qualifying set.
//
// This relies on qualifiedSnapshot being current — every state mutation
// funnels through RecomputeWith (via Processor.RecomputeEquipmentBonuses),
// so the cache is always populated when REST reads it.
func (m Model) Bonuses() []stat.Bonus {
	out := make([]stat.Bonus, 0, len(m.bonuses)+len(m.equipped)*4)
	out = append(out, m.bonuses...)
	for assetId, snap := range m.equipped {
		if !m.qualifiedSnapshot[assetId] {
			continue
		}
		out = append(out, snap.bonuses...)
	}
	return out
}
```

- [ ] **Step 4: Run, expect PASS for the new tests**

```bash
go test ./character/ -run "TestBonuses_" -v
```

- [ ] **Step 5: Run the full character suite**

```bash
go test ./character/ -v
```

Existing tests that exercised `AddEquipmentBonuses → Bonuses` will still pass because the legacy path keeps writing to `m.bonuses[]` until Tasks 15-17. Once those land, the migration is complete.

- [ ] **Step 6: Commit**

```bash
git add character/model.go character/model_test.go
git commit -m "feat(atlas-effective-stats): Bonuses() merges qualifying equipment snapshots"
```

---

## Task 14: Registry helpers — `PutEquippedAsset`, `RemoveEquippedAsset`, `SetWearerProfile`

**Why:** The processor needs atomic copy-on-write helpers for the new map fields, mirroring `AddBonus`/`RemoveBonusesBySource`.

**Files:**
- Modify: `character/registry.go`
- Modify: `character/processor_test.go` (extend with registry-level test)

- [ ] **Step 1: Write the failing tests**

Append to `character/processor_test.go`:

```go
func TestRegistry_PutEquippedAsset_PersistsSnapshot(t *testing.T) {
	setupTestRegistry(t)
	_, ctx, ten := createTestContext()
	ch := channel.NewModel(1, 2)
	snap := NewEquippedAsset(42, 1052095, []stat.Bonus{
		stat.NewBonus("equipment:42", stat.TypeMaxMp, 50),
	})
	GetRegistry().PutEquippedAsset(ctx, ch, 12345, snap)
	m, err := GetRegistry().Get(ctx, 12345)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if eq := m.Equipped(); len(eq) != 1 || eq[42].TemplateId() != 1052095 {
		t.Errorf("equipped not persisted: %+v", eq)
	}
	_ = ten
}

func TestRegistry_RemoveEquippedAsset_ReturnsNilErrorOnAbsent(t *testing.T) {
	setupTestRegistry(t)
	_, ctx, _ := createTestContext()
	if _, err := GetRegistry().RemoveEquippedAsset(ctx, 12345, 42); err == nil {
		t.Error("expected ErrNotFound for absent character")
	}
}

func TestRegistry_RemoveEquippedAsset_ClearsSnapshot(t *testing.T) {
	setupTestRegistry(t)
	_, ctx, _ := createTestContext()
	ch := channel.NewModel(1, 2)
	snap := NewEquippedAsset(42, 1052095, nil)
	GetRegistry().PutEquippedAsset(ctx, ch, 12345, snap)
	if _, err := GetRegistry().RemoveEquippedAsset(ctx, 12345, 42); err != nil {
		t.Fatalf("RemoveEquippedAsset: %v", err)
	}
	m, _ := GetRegistry().Get(ctx, 12345)
	if len(m.Equipped()) != 0 {
		t.Errorf("expected empty equipped map, got %+v", m.Equipped())
	}
}

func TestRegistry_SetWearerProfile_PersistsLevelAndJob(t *testing.T) {
	setupTestRegistry(t)
	_, ctx, _ := createTestContext()
	ch := channel.NewModel(1, 2)
	GetRegistry().SetWearerProfile(ctx, ch, 12345, NewWearerProfile(35, job.Id(200)))
	m, err := GetRegistry().Get(ctx, 12345)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if m.Wearer().Level() != 35 || m.Wearer().JobId() != job.Id(200) {
		t.Errorf("wearer not persisted: %+v", m.Wearer())
	}
}
```

Add `"github.com/Chronicle20/atlas/libs/atlas-constants/job"` to the test file's imports if missing.

- [ ] **Step 2: Run, expect FAIL (`PutEquippedAsset`, etc. undefined)**

```bash
go test ./character/ -run "TestRegistry_PutEquippedAsset|TestRegistry_RemoveEquippedAsset|TestRegistry_SetWearerProfile" -v
```

- [ ] **Step 3: Implement**

Append to `character/registry.go`:

```go
// PutEquippedAsset writes the snapshot, get-or-create the model. It does NOT
// recompute Computed — callers must follow up with RecomputeWith via the
// Processor.RecomputeEquipmentBonuses entry point.
func (r *Registry) PutEquippedAsset(ctx context.Context, ch channel.Model, characterId uint32, a EquippedAsset) Model {
	t := tenant.MustFromContext(ctx)
	m, err := r.characters.Get(ctx, t, characterId)
	if err != nil {
		m = NewModel(t, ch, characterId)
	}
	m = m.WithEquippedAsset(a)
	_ = r.characters.Put(ctx, t, characterId, m)
	return m
}

// RemoveEquippedAsset clears the snapshot. ErrNotFound is returned if the
// character is not in the registry — there is nothing to recompute in that
// case.
func (r *Registry) RemoveEquippedAsset(ctx context.Context, characterId uint32, assetId uint32) (Model, error) {
	t := tenant.MustFromContext(ctx)
	m, err := r.characters.Get(ctx, t, characterId)
	if err != nil {
		return Model{}, ErrNotFound
	}
	m = m.WithoutEquippedAsset(assetId)
	_ = r.characters.Put(ctx, t, characterId, m)
	return m, nil
}

// SetWearerProfile writes the level/jobId. Like PutEquippedAsset, it does
// NOT recompute — the Processor wraps this with RecomputeEquipmentBonuses.
func (r *Registry) SetWearerProfile(ctx context.Context, ch channel.Model, characterId uint32, p WearerProfile) Model {
	t := tenant.MustFromContext(ctx)
	m, err := r.characters.Get(ctx, t, characterId)
	if err != nil {
		m = NewModel(t, ch, characterId)
	}
	m = m.WithWearer(p)
	_ = r.characters.Put(ctx, t, characterId, m)
	return m
}
```

- [ ] **Step 4: Run, expect PASS**

```bash
go test ./character/ -run "TestRegistry_PutEquippedAsset|TestRegistry_RemoveEquippedAsset|TestRegistry_SetWearerProfile" -v
```

- [ ] **Step 5: Commit**

```bash
git add character/registry.go character/processor_test.go
git commit -m "feat(atlas-effective-stats): registry helpers for equipped snapshots and wearer profile"
```

---

## Task 15: Processor — `RecomputeEquipmentBonuses`, reshape `AddEquipmentBonuses`/`RemoveEquipmentBonuses`, add `SetWearerProfile`

**Why:** Single re-evaluation entry point; equipment now flows through the snapshot map.

**Files:**
- Modify: `character/processor.go`

- [ ] **Step 1: Update the `Processor` interface and impl signatures**

Edit `character/processor.go` — replace the `Processor` interface block at lines 15-33:

```go
type Processor interface {
	GetEffectiveStats(ch channel.Model, characterId uint32) (stat.Computed, []stat.Bonus, error)
	AddBonus(ch channel.Model, characterId uint32, source string, statType stat.Type, amount int32) error
	AddMultiplierBonus(ch channel.Model, characterId uint32, source string, statType stat.Type, multiplier float64) error
	RemoveBonus(characterId uint32, source string, statType stat.Type) error
	RemoveBonusesBySource(characterId uint32, source string) error
	SetBaseStats(ch channel.Model, characterId uint32, base stat.Base) error
	SetWearerProfile(ch channel.Model, characterId uint32, p WearerProfile) error
	// Equipment bonus methods
	AddEquipmentBonuses(ch channel.Model, characterId uint32, equipmentId uint32, templateId uint32, bonuses []stat.Bonus) error
	RemoveEquipmentBonuses(characterId uint32, equipmentId uint32) error
	// Re-evaluation entry point — invoked internally by SetBaseStats /
	// SetWearerProfile / AddEquipmentBonuses / RemoveEquipmentBonuses.
	// Exposed so consumer-side handlers that mutate state in unusual ways
	// can still trigger re-gating.
	RecomputeEquipmentBonuses(ch channel.Model, characterId uint32) error
	// Buff bonus methods
	AddBuffBonuses(ch channel.Model, characterId uint32, buffSourceId int32, bonuses []stat.Bonus) error
	RemoveBuffBonuses(characterId uint32, buffSourceId int32) error
	// Passive skill bonus methods
	AddPassiveBonuses(ch channel.Model, characterId uint32, skillId uint32, bonuses []stat.Bonus) error
	RemovePassiveBonuses(characterId uint32, skillId uint32) error
	// Cleanup
	RemoveCharacter(characterId uint32)
}
```

- [ ] **Step 2: Add `RecomputeEquipmentBonuses` and `SetWearerProfile` to the impl**

Append to `character/processor.go` (after `SetBaseStats`):

```go
// SetWearerProfile updates level/jobId then re-runs the qualifying set.
func (p *ProcessorImpl) SetWearerProfile(ch channel.Model, characterId uint32, wp WearerProfile) error {
	GetRegistry().SetWearerProfile(p.ctx, ch, characterId, wp)
	p.l.Debugf("Set wearer profile for character [%d]: level=%d job=%d", characterId, wp.Level(), wp.JobId())
	return p.RecomputeEquipmentBonuses(ch, characterId)
}

// RecomputeEquipmentBonuses re-runs QualifiedEquipment, updates Computed,
// and emits clamp commands when MaxHp/MaxMp drops.
func (p *ProcessorImpl) RecomputeEquipmentBonuses(ch channel.Model, characterId uint32) error {
	oldModel, err := GetRegistry().Get(p.ctx, characterId)
	if err != nil {
		return err
	}
	oldComputed := oldModel.Computed()

	newModel := oldModel.RecomputeWith(equipment.GetProvider(p.l), p.ctx)
	GetRegistry().Update(p.ctx, newModel)

	totalEquipped := len(newModel.Equipped())
	totalQualified := 0
	for _, ok := range newModel.qualifiedSnapshot {
		if ok {
			totalQualified++
		}
	}
	p.l.Debugf("Recomputed qualifying equipment for character [%d]: %d/%d items qualify.", characterId, totalQualified, totalEquipped)
	p.logEffectiveStats(characterId, newModel.Computed())

	p.checkAndPublishClampCommands(newModel, oldComputed, newModel.Computed())
	return nil
}
```

Add the import:

```go
import (
	// existing imports...
	"atlas-effective-stats/external/data/equipment"
)
```

(Merge into the existing import block.)

- [ ] **Step 3: Replace `AddEquipmentBonuses` and `RemoveEquipmentBonuses` impls**

Replace the existing bodies (lines ~141-157):

```go
// AddEquipmentBonuses writes the asset snapshot and re-runs the qualifying
// set. The asset's bonuses are stored alongside its template id; whether
// they enter Computed depends on the requirement check.
func (p *ProcessorImpl) AddEquipmentBonuses(ch channel.Model, characterId uint32, equipmentId uint32, templateId uint32, bonuses []stat.Bonus) error {
	source := fmt.Sprintf("equipment:%d", equipmentId)
	sourcedBonuses := make([]stat.Bonus, 0, len(bonuses))
	for _, b := range bonuses {
		sourcedBonuses = append(sourcedBonuses, stat.NewFullBonus(source, b.StatType(), b.Amount(), b.Multiplier()))
	}
	snap := NewEquippedAsset(equipmentId, templateId, sourcedBonuses)
	GetRegistry().PutEquippedAsset(p.ctx, ch, characterId, snap)
	p.l.Debugf("Stored equipment [%d] (template %d) snapshot for character [%d]: %d stats", equipmentId, templateId, characterId, len(bonuses))
	return p.RecomputeEquipmentBonuses(ch, characterId)
}

// RemoveEquipmentBonuses clears the asset snapshot and re-runs the qualifying
// set. Returns nil (not ErrNotFound) when the character is not registered —
// there is nothing to re-gate.
func (p *ProcessorImpl) RemoveEquipmentBonuses(characterId uint32, equipmentId uint32) error {
	m, err := GetRegistry().RemoveEquippedAsset(p.ctx, characterId, equipmentId)
	if err != nil {
		return nil
	}
	p.l.Debugf("Removed equipment [%d] snapshot for character [%d]", equipmentId, characterId)
	return p.RecomputeEquipmentBonuses(m.Channel(), characterId)
}
```

- [ ] **Step 4: Update `SetBaseStats` to fold in re-gate**

Replace its body:

```go
// SetBaseStats sets base numeric stats for a character then re-runs the
// qualifying set so STR/DEX/INT/LUK/level changes flip equipment in or out
// of Computed.
func (p *ProcessorImpl) SetBaseStats(ch channel.Model, characterId uint32, base stat.Base) error {
	GetRegistry().SetBaseStats(p.ctx, ch, characterId, base)
	p.l.Debugf("Set base stats for character [%d]: STR=%d, DEX=%d, INT=%d, LUK=%d, MaxHP=%d, MaxMP=%d",
		characterId, base.Strength(), base.Dexterity(), base.Intelligence(), base.Luck(), base.MaxHp(), base.MaxMp())
	return p.RecomputeEquipmentBonuses(ch, characterId)
}
```

> **Note:** `Registry.SetBaseStats` currently calls `m.Recompute()` internally, which now treats every equipped snapshot as qualifying (per Task 12's permissive `Recompute`). That's fine — `RecomputeEquipmentBonuses` immediately runs `RecomputeWith` and overwrites the result. The double pass is a few microseconds; not worth a registry-helper rewrite.

- [ ] **Step 5: Run tests; expect existing processor tests to pass and the new ones from Task 14 to keep passing**

```bash
go test ./character/ -v
```

If `TestProcessor_AddEquipmentBonuses` (or similar) fails because the test calls the old 4-arg signature, update the test call site to pass a synthetic templateId (e.g. `1234`):

```go
err := p.AddEquipmentBonuses(ch, 12345, 42, 1234, []stat.Bonus{...})
```

Re-run.

- [ ] **Step 6: Commit**

```bash
git add character/processor.go character/processor_test.go
git commit -m "feat(atlas-effective-stats): processor routes equipment through snapshot map and re-gates on every state change"
```

---

## Task 16: Update initializer to populate `wearer` + `equipped` and call `RecomputeWith`

**Why:** Lazy init now writes to the new fields directly. Existing equipment-bonus extraction is reused via the consumer-side helper.

**Files:**
- Modify: `character/initializer.go`

- [ ] **Step 1: Replace `fetchBaseStats` to also return wearer profile and refactor `InitializeCharacter`**

Replace the entire file body of `character/initializer.go` with:

```go
package character

import (
	"atlas-effective-stats/external/buffs"
	"atlas-effective-stats/external/character"
	"atlas-effective-stats/external/data/equipment"
	skilldata "atlas-effective-stats/external/data/skill"
	"atlas-effective-stats/external/inventory"
	"atlas-effective-stats/external/skills"
	"atlas-effective-stats/stat"
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/sirupsen/logrus"
)

// IsInitialized checks if a character has been initialized
func IsInitialized(ctx context.Context, characterId uint32) bool {
	return GetRegistry().IsInitialized(ctx, characterId)
}

// InitializeCharacter performs lazy initialization of a character's effective
// stats. It fetches base stats + wearer profile from atlas-character, the
// equipped-asset snapshots from atlas-inventory, buffs from atlas-buffs,
// passive skill bonuses from atlas-data + atlas-skills, and finally runs the
// equipment qualification gate before publishing Computed to the registry.
func InitializeCharacter(l logrus.FieldLogger, ctx context.Context, characterId uint32, ch channel.Model) error {
	l.Debugf("Initializing effective stats for character [%d] on world [%d] channel [%d].", characterId, ch.WorldId(), ch.Id())

	m := GetRegistry().GetOrCreate(ctx, ch, characterId)
	if err := GetRegistry().MarkInitialized(ctx, characterId); err != nil {
		l.WithError(err).Warnf("Failed to mark character [%d] as initialized.", characterId)
	}

	baseStats, wp, err := fetchWearer(l, ctx, characterId)
	if err != nil {
		l.WithError(err).Warnf("Failed to fetch wearer record for character [%d], using defaults.", characterId)
		baseStats = stat.NewBase(0, 0, 0, 0, 0, 0)
		wp = WearerProfile{}
	}
	m = m.WithBaseStats(baseStats).WithWearer(wp)

	snapshots, err := fetchEquippedSnapshots(l, ctx, characterId)
	if err != nil {
		l.WithError(err).Warnf("Failed to fetch equipped snapshots for character [%d].", characterId)
	} else {
		for _, snap := range snapshots {
			m = m.WithEquippedAsset(snap)
		}
	}

	buffBonuses, err := fetchBuffBonuses(l, ctx, characterId)
	if err != nil {
		l.WithError(err).Warnf("Failed to fetch buff bonuses for character [%d].", characterId)
	} else {
		m = m.WithBonuses(buffBonuses)
	}

	passiveBonuses, err := fetchPassiveBonuses(l, ctx, characterId)
	if err != nil {
		l.WithError(err).Warnf("Failed to fetch passive skill bonuses for character [%d].", characterId)
	} else {
		m = m.WithBonuses(passiveBonuses)
	}

	m = m.RecomputeWith(equipment.GetProvider(l), ctx).WithInitialized()
	GetRegistry().Update(ctx, m)

	l.Debugf("Completed initialization for character [%d]. Effective stats: STR=%d, DEX=%d, INT=%d, LUK=%d, MaxHP=%d, MaxMP=%d",
		characterId, m.Computed().Strength(), m.Computed().Dexterity(), m.Computed().Intelligence(),
		m.Computed().Luck(), m.Computed().MaxHp(), m.Computed().MaxMp())

	return nil
}

// fetchWearer fetches the character record and returns base stats + wearer
// profile in a single call.
func fetchWearer(l logrus.FieldLogger, ctx context.Context, characterId uint32) (stat.Base, WearerProfile, error) {
	l.Debugf("Fetching base stats + wearer profile for character [%d] from character service.", characterId)

	charData, err := character.RequestById(characterId)(l, ctx)
	if err != nil {
		return stat.Base{}, WearerProfile{}, fmt.Errorf("failed to fetch character [%d]: %w", characterId, err)
	}

	base := stat.NewBase(
		charData.Strength,
		charData.Dexterity,
		charData.Luck,
		charData.Intelligence,
		charData.MaxHp,
		charData.MaxMp,
	)
	wp := NewWearerProfile(charData.Level, charData.JobId)
	return base, wp, nil
}

// fetchEquippedSnapshots iterates the equip compartment and returns a
// snapshot per equipped (Slot < 0) asset, with bonuses pre-extracted.
func fetchEquippedSnapshots(l logrus.FieldLogger, ctx context.Context, characterId uint32) ([]EquippedAsset, error) {
	l.Debugf("Fetching equipment snapshots for character [%d] from inventory service.", characterId)

	compartment, err := inventory.RequestEquipCompartment(characterId)(l, ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch equip compartment for character [%d]: %w", characterId, err)
	}

	out := make([]EquippedAsset, 0)
	for _, asset := range compartment.Assets {
		if !asset.IsEquipped() {
			continue
		}
		equipData, ok := asset.GetEquipableData()
		if !ok {
			continue
		}
		bonuses := extractAssetBonuses(asset.Id, equipData)
		out = append(out, NewEquippedAsset(asset.Id, asset.TemplateId, bonuses))
	}

	l.Debugf("Built %d equipment snapshots for character [%d].", len(out), characterId)
	return out, nil
}

// extractAssetBonuses converts atlas-inventory equipable stats into the flat
// stat.Bonus list keyed by source = "equipment:<assetId>".
func extractAssetBonuses(assetId uint32, equipData inventory.EquipableRestData) []stat.Bonus {
	bonuses := make([]stat.Bonus, 0)
	source := fmt.Sprintf("equipment:%d", assetId)

	if equipData.Strength > 0 {
		bonuses = append(bonuses, stat.NewBonus(source, stat.TypeStrength, int32(equipData.Strength)))
	}
	if equipData.Dexterity > 0 {
		bonuses = append(bonuses, stat.NewBonus(source, stat.TypeDexterity, int32(equipData.Dexterity)))
	}
	if equipData.Luck > 0 {
		bonuses = append(bonuses, stat.NewBonus(source, stat.TypeLuck, int32(equipData.Luck)))
	}
	if equipData.Intelligence > 0 {
		bonuses = append(bonuses, stat.NewBonus(source, stat.TypeIntelligence, int32(equipData.Intelligence)))
	}
	if equipData.Hp > 0 {
		bonuses = append(bonuses, stat.NewBonus(source, stat.TypeMaxHp, int32(equipData.Hp)))
	}
	if equipData.Mp > 0 {
		bonuses = append(bonuses, stat.NewBonus(source, stat.TypeMaxMp, int32(equipData.Mp)))
	}
	if equipData.WeaponAttack > 0 {
		bonuses = append(bonuses, stat.NewBonus(source, stat.TypeWeaponAttack, int32(equipData.WeaponAttack)))
	}
	if equipData.MagicAttack > 0 {
		bonuses = append(bonuses, stat.NewBonus(source, stat.TypeMagicAttack, int32(equipData.MagicAttack)))
	}
	if equipData.WeaponDefense > 0 {
		bonuses = append(bonuses, stat.NewBonus(source, stat.TypeWeaponDefense, int32(equipData.WeaponDefense)))
	}
	if equipData.MagicDefense > 0 {
		bonuses = append(bonuses, stat.NewBonus(source, stat.TypeMagicDefense, int32(equipData.MagicDefense)))
	}
	if equipData.Accuracy > 0 {
		bonuses = append(bonuses, stat.NewBonus(source, stat.TypeAccuracy, int32(equipData.Accuracy)))
	}
	if equipData.Avoidability > 0 {
		bonuses = append(bonuses, stat.NewBonus(source, stat.TypeAvoidability, int32(equipData.Avoidability)))
	}
	if equipData.Speed > 0 {
		bonuses = append(bonuses, stat.NewBonus(source, stat.TypeSpeed, int32(equipData.Speed)))
	}
	if equipData.Jump > 0 {
		bonuses = append(bonuses, stat.NewBonus(source, stat.TypeJump, int32(equipData.Jump)))
	}
	return bonuses
}

// fetchBuffBonuses fetches active buffs and their stat changes
func fetchBuffBonuses(l logrus.FieldLogger, ctx context.Context, characterId uint32) ([]stat.Bonus, error) {
	l.Debugf("Fetching buff bonuses for character [%d] from buffs service.", characterId)

	buffList, err := buffs.RequestCharacterBuffs(characterId)(l, ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch buffs for character [%d]: %w", characterId, err)
	}

	bonuses := make([]stat.Bonus, 0)
	for _, buff := range buffList {
		source := fmt.Sprintf("buff:%d", buff.SourceId)
		for _, change := range buff.Changes {
			statType, isMultiplier := stat.MapBuffStatType(change.Type)
			if statType == "" {
				l.Debugf("Unknown buff stat type: %s", change.Type)
				continue
			}
			if isMultiplier {
				multiplier := float64(change.Amount) / 100.0
				bonuses = append(bonuses, stat.NewMultiplierBonus(source, statType, multiplier))
			} else {
				bonuses = append(bonuses, stat.NewBonus(source, statType, change.Amount))
			}
		}
	}

	l.Debugf("Fetched %d buff bonuses for character [%d].", len(bonuses), characterId)
	return bonuses, nil
}

// fetchPassiveBonuses fetches passive skill bonuses from character skills
func fetchPassiveBonuses(l logrus.FieldLogger, ctx context.Context, characterId uint32) ([]stat.Bonus, error) {
	l.Debugf("Fetching passive skill bonuses for character [%d].", characterId)

	characterSkills, err := skills.RequestCharacterSkills(characterId)(l, ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch skills for character [%d]: %w", characterId, err)
	}

	bonuses := make([]stat.Bonus, 0)
	for _, charSkill := range characterSkills {
		if charSkill.Level == 0 {
			continue
		}
		skillInfo, err := skilldata.RequestById(charSkill.Id)(l, ctx)
		if err != nil {
			l.WithError(err).Debugf("Failed to fetch skill data for skill [%d], skipping.", charSkill.Id)
			continue
		}
		if !skillInfo.IsPassive() {
			continue
		}
		effect := skillInfo.GetEffectForLevel(charSkill.Level)
		if effect == nil {
			l.Debugf("No effect found for passive skill [%d] at level [%d].", charSkill.Id, charSkill.Level)
			continue
		}
		source := fmt.Sprintf("passive:%d", charSkill.Id)
		if effect.WeaponAttack != 0 {
			bonuses = append(bonuses, stat.NewBonus(source, stat.TypeWeaponAttack, int32(effect.WeaponAttack)))
		}
		if effect.MagicAttack != 0 {
			bonuses = append(bonuses, stat.NewBonus(source, stat.TypeMagicAttack, int32(effect.MagicAttack)))
		}
		if effect.WeaponDefense != 0 {
			bonuses = append(bonuses, stat.NewBonus(source, stat.TypeWeaponDefense, int32(effect.WeaponDefense)))
		}
		if effect.MagicDefense != 0 {
			bonuses = append(bonuses, stat.NewBonus(source, stat.TypeMagicDefense, int32(effect.MagicDefense)))
		}
		if effect.Accuracy != 0 {
			bonuses = append(bonuses, stat.NewBonus(source, stat.TypeAccuracy, int32(effect.Accuracy)))
		}
		if effect.Avoidability != 0 {
			bonuses = append(bonuses, stat.NewBonus(source, stat.TypeAvoidability, int32(effect.Avoidability)))
		}
		if effect.Speed != 0 {
			bonuses = append(bonuses, stat.NewBonus(source, stat.TypeSpeed, int32(effect.Speed)))
		}
		if effect.Jump != 0 {
			bonuses = append(bonuses, stat.NewBonus(source, stat.TypeJump, int32(effect.Jump)))
		}
		if effect.Hp > 0 {
			bonuses = append(bonuses, stat.NewBonus(source, stat.TypeMaxHp, int32(effect.Hp)))
		}
		if effect.Mp > 0 {
			bonuses = append(bonuses, stat.NewBonus(source, stat.TypeMaxMp, int32(effect.Mp)))
		}
		for _, statup := range effect.Statups {
			statType := stat.MapStatupType(statup.Type)
			if statType == "" {
				l.Debugf("Unknown passive stat type: %s for skill [%d].", statup.Type, charSkill.Id)
				continue
			}
			bonuses = append(bonuses, stat.NewBonus(source, statType, statup.Amount))
		}
	}

	l.Debugf("Fetched %d passive skill bonuses for character [%d].", len(bonuses), characterId)
	return bonuses, nil
}
```

- [ ] **Step 2: Run the existing initializer test suite to ensure nothing is broken**

```bash
go test ./character/ -v
```

If `initializer_test.go` references the removed `fetchBaseStats` symbol, update the call to `fetchWearer` and discard the second return value. Re-run.

- [ ] **Step 3: Commit**

```bash
git add character/initializer.go
git commit -m "feat(atlas-effective-stats): initializer populates wearer + equipped snapshots and runs RecomputeWith"
```

---

## Task 17: Asset consumer — pass `templateId`, route through snapshot map

**Files:**
- Modify: `kafka/consumer/asset/consumer.go`

- [ ] **Step 1: Update `handleItemEquipped` to pass templateId**

Edit `kafka/consumer/asset/consumer.go`. Replace the body of `handleItemEquipped` (lines ~64-101):

```go
func handleItemEquipped(l logrus.FieldLogger, ctx context.Context, e asset.StatusEvent[asset.MovedStatusEventBody]) {
	l.Debugf("Equipment [%d] (template %d) equipped by character [%d], fetching stats.", e.AssetId, e.TemplateId, e.CharacterId)

	compartment, err := inventory.RequestEquipCompartment(e.CharacterId)(l, ctx)
	if err != nil {
		l.WithError(err).Errorf("Failed to fetch equipment data for character [%d].", e.CharacterId)
		return
	}

	var equipData *inventory.EquipableRestData
	for _, a := range compartment.Assets {
		if a.Id == e.AssetId {
			data, ok := a.GetEquipableData()
			if ok {
				equipData = &data
			}
			break
		}
	}

	if equipData == nil {
		l.Warnf("Could not find equipment data for asset [%d].", e.AssetId)
		return
	}

	bonuses := extractEquipmentBonuses(e.AssetId, equipData)
	ch := channel.NewModel(0, 0)
	if err := character.NewProcessor(l, ctx).AddEquipmentBonuses(ch, e.CharacterId, e.AssetId, e.TemplateId, bonuses); err != nil {
		l.WithError(err).Errorf("Failed to add equipment bonuses for character [%d].", e.CharacterId)
	}
}
```

The `bonuses` slice is now allowed to be empty — an asset with no positive stats still needs to be tracked in the snapshot map so it participates in the qualification iteration. The `if len(bonuses) > 0` guard is dropped on purpose.

- [ ] **Step 2: Build**

```bash
go build ./...
```

The consumer now matches the new 4-arg `AddEquipmentBonuses` signature; build should succeed.

- [ ] **Step 3: Run tests**

```bash
go test ./...
```

- [ ] **Step 4: Commit**

```bash
git add kafka/consumer/asset/consumer.go
git commit -m "feat(atlas-effective-stats): asset consumer passes templateId and registers every equipped asset"
```

---

## Task 18: Character consumer — split stat-changed handler, add LEVEL/JOB branch

**Files:**
- Modify: `kafka/consumer/character/consumer.go`
- Modify: `kafka/consumer/character/consumer_test.go` (extend)

- [ ] **Step 1: Write the failing tests for the new branches**

Read the existing `consumer_test.go` first to keep helper-function reuse consistent. Append:

```go
func TestHandleStatChanged_JobUpdateRefetchesWearer(t *testing.T) {
	// Set up: registry pre-populated with character + initial wearer profile.
	// Dispatch STAT_CHANGED with Updates=[JOB], Values=nil.
	// Stub external/character.RequestById to return JobId=200.
	// Expect: registry's wearer.JobId is updated to 200.
	t.Skip("integration test placeholder; covered by Task 21 — keeps the consumer split honest at unit scope")
}

func TestHandleStatChanged_LevelUpdateRefetchesWearer(t *testing.T) {
	t.Skip("integration test placeholder; covered by Task 21")
}

func TestHandleStatChanged_NumericAndProfileBothInOneEvent(t *testing.T) {
	t.Skip("integration test placeholder; covered by Task 21")
}
```

> **Why skip placeholders?** The existing consumer_test.go uses miniredis but does not stub the `requests.RootUrl(...)` HTTP layer. Wiring an HTTP test server here would balloon Task 18; integration coverage in Task 21 (which runs with httptest.Server-backed stubs) gives stronger coverage at the right level.

- [ ] **Step 2: Replace `handleStatChanged` with the split form**

Replace the function in `kafka/consumer/character/consumer.go`:

```go
func handleStatChanged(l logrus.FieldLogger, ctx context.Context, e character2.StatusEvent[character2.StatusEventStatChangedBody]) {
	if e.Type != character2.StatusEventTypeStatChanged {
		return
	}

	relevantNumeric := false
	relevantProfile := false
	for _, update := range e.Body.Updates {
		switch update {
		case "MAX_HP", "MAX_MP", "STRENGTH", "DEXTERITY", "INTELLIGENCE", "LUCK":
			relevantNumeric = true
		case "LEVEL", "JOB":
			relevantProfile = true
		}
	}
	if !relevantNumeric && !relevantProfile {
		return
	}

	l.Debugf("Processing stat changed event for character [%d] updates=%v values=%v", e.CharacterId, e.Body.Updates, e.Body.Values)

	p := character.NewProcessor(l, ctx)
	ch := channel.NewModel(e.WorldId, e.Body.ChannelId)

	if relevantNumeric && len(e.Body.Values) > 0 {
		currentBase := lookupCurrentBase(ctx, l, ch, e.CharacterId)
		merged := mergeBaseStats(currentBase, e.Body.Values)
		if err := p.SetBaseStats(ch, e.CharacterId, merged); err != nil {
			l.WithError(err).Errorf("Unable to set base stats for character [%d].", e.CharacterId)
		}
	}

	if relevantProfile {
		// atlas-character emits TypeLevel / TypeJob STAT_CHANGED events with
		// Values=nil (see services/atlas-character/.../character/processor.go).
		// We must refetch the wearer record to pick up the new level/jobId.
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

Add the alias import at the top of the file (the package already imports `atlas-effective-stats/character`; we need a separate alias for the external client to avoid the name collision):

```go
import (
	"atlas-effective-stats/character"
	externalcharacter "atlas-effective-stats/external/character"
	consumer2 "atlas-effective-stats/kafka/consumer"
	character2 "atlas-effective-stats/kafka/message/character"
	"atlas-effective-stats/stat"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	// existing remainder...
)
```

- [ ] **Step 3: Build**

```bash
go build ./...
```

- [ ] **Step 4: Run tests**

```bash
go test ./...
```

- [ ] **Step 5: Commit**

```bash
git add kafka/consumer/character/consumer.go kafka/consumer/character/consumer_test.go
git commit -m "feat(atlas-effective-stats): consumer re-gates equipment on LEVEL/JOB STAT_CHANGED via wearer refetch"
```

---

## Task 19: Integration test — diagnosis case unqualified

**Why:** Lock in the PRD §4.1 reproduction end-to-end through the initializer.

**Files:**
- Modify or create: `character/initializer_test.go`

- [ ] **Step 1: Write the failing test**

Append to `character/initializer_test.go`:

```go
func TestInitializeCharacter_DropsUnqualifiedOverall_Diagnosis(t *testing.T) {
	setupTestRegistry(t)
	l, ctx, _ := createTestContext()

	stubServers := startInitializerStubs(t, stubConfig{
		character: stubCharacter{
			level: 30, jobId: 200,
			str: 4, dex: 25, intl: 4, luk: 39,
			maxHp: 1430, maxMp: 6330,
		},
		equipped: []stubEquipped{{
			assetId: 42, templateId: 1052095, slot: -5, mp: 50,
		}},
		equipmentReqs: map[uint32]equipmentReqs{
			1052095: {reqLuk: 40},
		},
	})
	t.Cleanup(stubServers.Close)
	stubServers.PointEnv(t)

	if err := InitializeCharacter(l, ctx, 12345, channel.NewModel(0, 0)); err != nil {
		t.Fatalf("InitializeCharacter: %v", err)
	}
	m, err := GetRegistry().Get(ctx, 12345)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if m.Computed().MaxMp() != 6330 {
		t.Errorf("MaxMp = %d, want 6330 (overall +50 should be dropped)", m.Computed().MaxMp())
	}
	for _, b := range m.Bonuses() {
		if b.Source() == "equipment:42" {
			t.Errorf("Bonuses() unexpectedly contains equipment:42 entry: %+v", b)
		}
	}
}

func TestInitializeCharacter_KeepsQualifiedOverall(t *testing.T) {
	setupTestRegistry(t)
	l, ctx, _ := createTestContext()

	stubServers := startInitializerStubs(t, stubConfig{
		character: stubCharacter{
			level: 30, jobId: 200,
			str: 4, dex: 25, intl: 4, luk: 40,
			maxHp: 1430, maxMp: 6330,
		},
		equipped: []stubEquipped{{
			assetId: 42, templateId: 1052095, slot: -5, mp: 50,
		}},
		equipmentReqs: map[uint32]equipmentReqs{
			1052095: {reqLuk: 40},
		},
	})
	t.Cleanup(stubServers.Close)
	stubServers.PointEnv(t)

	if err := InitializeCharacter(l, ctx, 12345, channel.NewModel(0, 0)); err != nil {
		t.Fatalf("InitializeCharacter: %v", err)
	}
	m, _ := GetRegistry().Get(ctx, 12345)
	if m.Computed().MaxMp() != 6380 {
		t.Errorf("MaxMp = %d, want 6380", m.Computed().MaxMp())
	}
}
```

- [ ] **Step 2: Implement the stub harness**

Create `character/stubs_test.go` (one file used by the integration tests):

```go
package character

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

type stubCharacter struct {
	level                       byte
	jobId                       uint16
	str, dex, intl, luk         uint16
	maxHp, maxMp                uint16
}

type stubEquipped struct {
	assetId    uint32
	templateId uint32
	slot       int16
	str, dex, intl, luk uint16
	hp, mp     uint16
	wAtk, mAtk uint16
}

type equipmentReqs struct {
	reqLevel byte
	reqJob   uint16
	reqStr, reqDex, reqInt, reqLuk uint16
}

type stubConfig struct {
	character     stubCharacter
	equipped      []stubEquipped
	equipmentReqs map[uint32]equipmentReqs
}

type stubServers struct {
	character *httptest.Server
	inventory *httptest.Server
	data      *httptest.Server
	buffs     *httptest.Server
	skills    *httptest.Server
}

func (s *stubServers) Close() {
	s.character.Close()
	s.inventory.Close()
	s.data.Close()
	s.buffs.Close()
	s.skills.Close()
}

// PointEnv sets the *_BASE_URL env vars consulted by requests.RootUrl so the
// initializer talks to our stubs instead of real services.
func (s *stubServers) PointEnv(t *testing.T) {
	t.Helper()
	t.Setenv("CHARACTER_BASE_URL", s.character.URL)
	t.Setenv("INVENTORY_BASE_URL", s.inventory.URL)
	t.Setenv("DATA_BASE_URL", s.data.URL)
	t.Setenv("BUFFS_BASE_URL", s.buffs.URL)
	t.Setenv("SKILLS_BASE_URL", s.skills.URL)
}

func startInitializerStubs(t *testing.T, cfg stubConfig) *stubServers {
	t.Helper()

	character := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// /api/characters/{id}
		idStr := lastSegment(r.URL.Path)
		id, _ := strconv.Atoi(idStr)
		writeJSONAPI(w, map[string]interface{}{
			"data": map[string]interface{}{
				"type": "characters",
				"id":   strconv.Itoa(id),
				"attributes": map[string]interface{}{
					"level":        cfg.character.level,
					"jobId":        cfg.character.jobId,
					"strength":     cfg.character.str,
					"dexterity":    cfg.character.dex,
					"intelligence": cfg.character.intl,
					"luck":         cfg.character.luk,
					"maxHp":        cfg.character.maxHp,
					"maxMp":        cfg.character.maxMp,
					"hp":           cfg.character.maxHp,
					"mp":           cfg.character.maxMp,
				},
			},
		})
	}))

	inventory := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// /api/characters/{id}/inventory/compartments?type=1&include=assets
		assets := make([]map[string]interface{}, 0, len(cfg.equipped))
		included := make([]map[string]interface{}, 0, len(cfg.equipped))
		for _, e := range cfg.equipped {
			idStr := strconv.FormatUint(uint64(e.assetId), 10)
			assets = append(assets, map[string]interface{}{
				"type": "assets",
				"id":   idStr,
			})
			included = append(included, map[string]interface{}{
				"type": "assets",
				"id":   idStr,
				"attributes": map[string]interface{}{
					"slot":          e.slot,
					"templateId":    e.templateId,
					"strength":      e.str,
					"dexterity":     e.dex,
					"intelligence":  e.intl,
					"luck":          e.luk,
					"hp":            e.hp,
					"mp":            e.mp,
					"weaponAttack":  e.wAtk,
					"magicAttack":   e.mAtk,
				},
			})
		}
		writeJSONAPI(w, map[string]interface{}{
			"data": map[string]interface{}{
				"type": "compartments",
				"id":   "1",
				"attributes": map[string]interface{}{
					"type":     1,
					"capacity": 24,
				},
				"relationships": map[string]interface{}{
					"assets": map[string]interface{}{"data": assets},
				},
			},
			"included": included,
		})
	}))

	data := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// /api/data/equipment/{id}
		idStr := lastSegment(r.URL.Path)
		id, _ := strconv.ParseUint(idStr, 10, 32)
		reqs, ok := cfg.equipmentReqs[uint32(id)]
		if !ok {
			http.NotFound(w, r)
			return
		}
		writeJSONAPI(w, map[string]interface{}{
			"data": map[string]interface{}{
				"type": "equipment",
				"id":   idStr,
				"attributes": map[string]interface{}{
					"reqLevel": reqs.reqLevel,
					"reqJob":   reqs.reqJob,
					"reqStr":   reqs.reqStr,
					"reqDex":   reqs.reqDex,
					"reqInt":   reqs.reqInt,
					"reqLuk":   reqs.reqLuk,
				},
			},
		})
	}))

	buffs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeJSONAPI(w, map[string]interface{}{"data": []interface{}{}})
	}))

	skills := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeJSONAPI(w, map[string]interface{}{"data": []interface{}{}})
	}))

	return &stubServers{
		character: character,
		inventory: inventory,
		data:      data,
		buffs:     buffs,
		skills:    skills,
	}
}

func lastSegment(path string) string {
	idx := strings.LastIndex(path, "/")
	if idx < 0 {
		return path
	}
	return path[idx+1:]
}

func writeJSONAPI(w http.ResponseWriter, body map[string]interface{}) {
	w.Header().Set("Content-Type", "application/vnd.api+json")
	_ = json.NewEncoder(w).Encode(body)
	_ = fmt.Sprintf // keep fmt referenced to avoid linter "imported and not used"
}
```

> **Important:** Verify the env-var names match what `requests.RootUrl(<key>)` actually reads. Run:
>
> ```bash
> grep -rn "RootUrl\|os.Getenv\|BASE_URL" libs/atlas-rest/requests/ services/atlas-effective-stats/atlas.com/effective-stats/external/ | head -20
> ```
>
> If the convention is `<KEY>_SERVICE_URL` instead of `<KEY>_BASE_URL`, update the `Setenv` calls in `PointEnv`. Likewise verify the URL **path templates**: `external/character/requests.go`, `external/inventory/requests.go`, and `external/data/skill/requests.go` show the exact route templates each client uses; the stub handlers must match.

- [ ] **Step 3: Run, expect PASS**

```bash
go test ./character/ -run "TestInitializeCharacter_DropsUnqualifiedOverall|TestInitializeCharacter_KeepsQualifiedOverall" -v
```

If a test fails because the stub handler returns 404 (route mismatch), fix the stub paths to match `external/<service>/requests.go`. Iterate until both tests pass.

- [ ] **Step 4: Run the full suite**

```bash
go test ./...
```

- [ ] **Step 5: Commit**

```bash
git add character/initializer_test.go character/stubs_test.go
git commit -m "test(atlas-effective-stats): integration test for PRD §4.1 unqualified-overall reproduction"
```

---

## Task 20: Integration test — re-gate after `STAT_CHANGED` (LUCK / JOB / LEVEL)

**Files:**
- Modify: `kafka/consumer/character/consumer_test.go`

- [ ] **Step 1: Write the failing tests**

Append to `kafka/consumer/character/consumer_test.go`. The existing tests in this file already establish a setup pattern; reuse it. (If the file's setup helper is private to that package, copy/adapt the stub harness from Task 19 — likely the cleanest move is to create a sibling `kafka/consumer/character/stubs_test.go` that mirrors `character/stubs_test.go`.)

```go
func TestHandleStatChanged_LuckRise_RegatesEquipment(t *testing.T) {
	setupCharacterTest(t) // miniredis + initial registry

	stubs := startInitializerStubs(t, stubConfig{
		character: stubCharacter{
			level: 30, jobId: 200,
			str: 4, dex: 25, intl: 4, luk: 39,
			maxHp: 1430, maxMp: 6330,
		},
		equipped: []stubEquipped{{
			assetId: 42, templateId: 1052095, slot: -5, mp: 50,
		}},
		equipmentReqs: map[uint32]equipmentReqs{1052095: {reqLuk: 40}},
	})
	t.Cleanup(stubs.Close)
	stubs.PointEnv(t)

	// Lazy-init the character (luk=39 → asset disqualified).
	l, ctx, _ := createTestContext()
	if _, _, err := character.NewProcessor(l, ctx).GetEffectiveStats(channel.NewModel(0, 0), 12345); err != nil {
		t.Fatalf("GetEffectiveStats: %v", err)
	}
	m, _ := character.GetRegistry().Get(ctx, 12345)
	if m.Computed().MaxMp() != 6330 {
		t.Fatalf("pre: MaxMp = %d, want 6330", m.Computed().MaxMp())
	}

	// Dispatch STAT_CHANGED with LUCK → 40.
	handleStatChanged(l, ctx, character2.StatusEvent[character2.StatusEventStatChangedBody]{
		WorldId:     0,
		CharacterId: 12345,
		Type:        character2.StatusEventTypeStatChanged,
		Body: character2.StatusEventStatChangedBody{
			ChannelId: 0,
			Updates:   []stat.Type{stat.TypeLuck},
			Values:    map[string]interface{}{"luck": 40},
		},
	})

	m, _ = character.GetRegistry().Get(ctx, 12345)
	if m.Computed().MaxMp() != 6380 {
		t.Errorf("post: MaxMp = %d, want 6380 (asset reactivated)", m.Computed().MaxMp())
	}
}

func TestHandleStatChanged_JobChange_RefetchesAndRegates(t *testing.T) {
	setupCharacterTest(t)

	// Wearer starts as Magician (200), the equip is Warrior-only (reqJob=1).
	cfg := stubConfig{
		character: stubCharacter{
			level: 30, jobId: 200, str: 4, dex: 25, intl: 4, luk: 50,
			maxHp: 1430, maxMp: 6330,
		},
		equipped: []stubEquipped{{
			assetId: 99, templateId: 1402000, slot: -10, str: 5,
		}},
		equipmentReqs: map[uint32]equipmentReqs{1402000: {reqJob: 1}},
	}
	stubs := startInitializerStubs(t, cfg)
	t.Cleanup(stubs.Close)
	stubs.PointEnv(t)

	l, ctx, _ := createTestContext()
	_, _, _ = character.NewProcessor(l, ctx).GetEffectiveStats(channel.NewModel(0, 0), 12345)
	m, _ := character.GetRegistry().Get(ctx, 12345)
	if m.Computed().Strength() != uint32(cfg.character.str) {
		t.Fatalf("pre: STR = %d, want %d (Magician should not get Warrior weapon)", m.Computed().Strength(), cfg.character.str)
	}

	// Now flip the wearer to a Warrior (the stub reflects new state on the
	// character endpoint). The handler refetches via the stub.
	cfg.character.jobId = 100
	stubs.character.Close()
	newStubs := startInitializerStubs(t, cfg)
	t.Cleanup(newStubs.Close)
	newStubs.PointEnv(t)

	handleStatChanged(l, ctx, character2.StatusEvent[character2.StatusEventStatChangedBody]{
		WorldId:     0,
		CharacterId: 12345,
		Type:        character2.StatusEventTypeStatChanged,
		Body: character2.StatusEventStatChangedBody{
			ChannelId: 0,
			Updates:   []stat.Type{stat.TypeJob},
			Values:    nil,
		},
	})

	m, _ = character.GetRegistry().Get(ctx, 12345)
	if m.Computed().Strength() != uint32(cfg.character.str+5) {
		t.Errorf("post-job-change: STR = %d, want %d (asset should now qualify)", m.Computed().Strength(), cfg.character.str+5)
	}
}

func TestHandleStatChanged_LevelRise_RefetchesAndRegates(t *testing.T) {
	setupCharacterTest(t)

	cfg := stubConfig{
		character: stubCharacter{
			level: 29, jobId: 100, str: 50, dex: 4, intl: 4, luk: 4,
			maxHp: 1430, maxMp: 6330,
		},
		equipped: []stubEquipped{{
			assetId: 7, templateId: 1302000, slot: -10, str: 5,
		}},
		equipmentReqs: map[uint32]equipmentReqs{1302000: {reqLevel: 30}},
	}
	stubs := startInitializerStubs(t, cfg)
	t.Cleanup(stubs.Close)
	stubs.PointEnv(t)

	l, ctx, _ := createTestContext()
	_, _, _ = character.NewProcessor(l, ctx).GetEffectiveStats(channel.NewModel(0, 0), 12345)
	m, _ := character.GetRegistry().Get(ctx, 12345)
	if m.Computed().Strength() != 50 {
		t.Fatalf("pre: STR = %d, want 50 (asset disqualifies at level 29)", m.Computed().Strength())
	}

	cfg.character.level = 30
	stubs.character.Close()
	newStubs := startInitializerStubs(t, cfg)
	t.Cleanup(newStubs.Close)
	newStubs.PointEnv(t)

	handleStatChanged(l, ctx, character2.StatusEvent[character2.StatusEventStatChangedBody]{
		WorldId:     0,
		CharacterId: 12345,
		Type:        character2.StatusEventTypeStatChanged,
		Body: character2.StatusEventStatChangedBody{
			ChannelId: 0,
			Updates:   []stat.Type{stat.TypeLevel},
			Values:    nil,
		},
	})

	m, _ = character.GetRegistry().Get(ctx, 12345)
	if m.Computed().Strength() != 55 {
		t.Errorf("post-level: STR = %d, want 55 (asset newly qualifies)", m.Computed().Strength())
	}
}
```

The helpers `setupCharacterTest`, `stubConfig`, `startInitializerStubs`, etc. live in this consumer package's test scope; they should be created in this task as `kafka/consumer/character/stubs_test.go` mirroring `character/stubs_test.go`.

> **Note on `setupCharacterTest`:** This helper sets up `miniredis` + `character.InitRegistry`, identical to `setupTestRegistry` in `character/processor_test.go`. Implement it in `kafka/consumer/character/stubs_test.go`:
>
> ```go
> func setupCharacterTest(t *testing.T) {
>     t.Helper()
>     mr := miniredis.RunT(t)
>     client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
>     character.InitRegistry(client)
> }
>
> func createTestContext() (logrus.FieldLogger, context.Context, tenant.Model) {
>     l, _ := test.NewNullLogger()
>     tn, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
>     ctx := tenant.WithContext(context.Background(), tn)
>     return l, ctx, tn
> }
> ```

- [ ] **Step 2: Verify env-var names again with the harness in place**

If `requests.RootUrl(<KEY>)` does not resolve to `<KEY>_BASE_URL`, the stubs won't be hit. Run:

```bash
grep -rn "RootUrl" libs/atlas-rest/requests/ | head -10
```

Adjust `PointEnv` accordingly.

- [ ] **Step 3: Run, expect PASS**

```bash
go test ./kafka/consumer/character/ -v
```

- [ ] **Step 4: Commit**

```bash
git add kafka/consumer/character/consumer_test.go kafka/consumer/character/stubs_test.go
git commit -m "test(atlas-effective-stats): re-gate equipment on LUCK/JOB/LEVEL STAT_CHANGED events"
```

---

## Task 21: Integration test — cross-asset qualification on equip event

**Files:**
- Create: `kafka/consumer/asset/consumer_test.go`

- [ ] **Step 1: Write the failing test**

Create `kafka/consumer/asset/consumer_test.go`:

```go
package asset

import (
	"testing"

	"atlas-effective-stats/character"
	"atlas-effective-stats/kafka/message/asset"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
)

func TestHandleAssetMoved_CapeUnlocksWeapon_CrossAssetQualification(t *testing.T) {
	setupAssetTest(t) // see stubs_test.go below

	// Step 1: character has a weapon equipped requiring STR>=110.
	// Base STR=100. Without the cape, weapon is disqualified.
	cfg := stubConfig{
		character: stubCharacter{
			level: 30, jobId: 100,
			str: 100, dex: 4, intl: 4, luk: 4, maxHp: 1430, maxMp: 1000,
		},
		equipped: []stubEquipped{
			{assetId: 1, templateId: 1402000, slot: -11, wAtk: 50},
		},
		equipmentReqs: map[uint32]equipmentReqs{
			1402000: {reqStr: 110},
			1102000: {reqStr: 0}, // cape is no-req
		},
	}
	stubs := startInitializerStubs(t, cfg)
	t.Cleanup(stubs.Close)
	stubs.PointEnv(t)

	l, ctx, _ := createTestContext()
	_, _, _ = character.NewProcessor(l, ctx).GetEffectiveStats(channel.NewModel(0, 0), 12345)
	m, _ := character.GetRegistry().Get(ctx, 12345)
	if hasEquipmentBonus(m, 1) {
		t.Fatalf("pre: weapon should be unqualified (base STR 100 < 110)")
	}

	// Step 2: dispatch MOVED for the cape (slot 100 → -9). Inventory stub
	// must now also expose the cape so the handler's compartment refetch
	// sees it.
	cfg.equipped = append(cfg.equipped, stubEquipped{
		assetId: 2, templateId: 1102000, slot: -9, str: 10,
	})
	stubs.inventory.Close()
	newStubs := startInitializerStubs(t, cfg)
	t.Cleanup(newStubs.Close)
	newStubs.PointEnv(t)

	handleAssetMoved(l, ctx, asset.StatusEvent[asset.MovedStatusEventBody]{
		WorldId:     0,
		CharacterId: 12345,
		AssetId:     2,
		TemplateId:  1102000,
		Slot:        -9,
		Type:        asset.StatusEventTypeMoved,
		Body:        asset.MovedStatusEventBody{ChannelId: 0, OldSlot: 100},
	})

	m, _ = character.GetRegistry().Get(ctx, 12345)
	if !hasEquipmentBonus(m, 1) {
		t.Errorf("post-equip: weapon should now qualify (effective STR 100+10=110 ≥ 110)")
	}
	if !hasEquipmentBonus(m, 2) {
		t.Errorf("post-equip: cape should be present in qualifying set")
	}
}

func hasEquipmentBonus(m character.Model, assetId uint32) bool {
	src := "equipment:" + strconv.FormatUint(uint64(assetId), 10)
	for _, b := range m.Bonuses() {
		if b.Source() == src {
			return true
		}
	}
	return false
}
```

(Add `"strconv"` to the imports.)

- [ ] **Step 2: Implement `kafka/consumer/asset/stubs_test.go`**

Mirror the harness from Task 19. The simplest move is to copy `character/stubs_test.go` into `kafka/consumer/asset/stubs_test.go` and adjust the package declaration to `package asset`. Reuse `setupAssetTest` / `createTestContext` from there.

- [ ] **Step 3: Run, expect PASS**

```bash
go test ./kafka/consumer/asset/ -v
```

- [ ] **Step 4: Commit**

```bash
git add kafka/consumer/asset/consumer_test.go kafka/consumer/asset/stubs_test.go
git commit -m "test(atlas-effective-stats): cross-asset qualification on equip event end-to-end"
```

---

## Task 22: Final build + full test sweep + smoke instructions

**Files:** none (verification only)

- [ ] **Step 1: Build everything**

```bash
go build ./...
```
Expected: clean.

- [ ] **Step 2: Run every test**

```bash
go test ./...
```
Expected: PASS.

- [ ] **Step 3: Vet**

```bash
go vet ./...
```
Expected: clean.

- [ ] **Step 4: Document smoke verification (no commit unless docs exist)**

The smoke procedure lives in PRD §4.1 / Acceptance Criteria. Reproduce in plain text here, for posterity in case the executing engineer needs to demonstrate the fix on the dev cluster:

```
1. Identify the diagnosis character (was wearing an overall granting +50 MP
   with reqLuk=40, while the wearer's LUK was 39).
2. GET /api/worlds/0/channels/0/characters/<id>/stats against atlas-effective-stats
   in the dev tenant. Expect Computed.maxMp == base.maxMp (no +50).
3. Confirm the bonuses[] array does NOT contain an `equipment:<assetId>` entry
   for that overall.
4. AP-distribute +1 LUK on the character. Wait for the STAT_CHANGED LUCK
   event to land.
5. Re-GET. Expect Computed.maxMp == base.maxMp + 50 and the
   `equipment:<assetId>` entry to appear in bonuses[].
6. In-game, the v83 client's MP bar cap should match Computed.maxMp; MP
   regen via ChangeMP should top up to that cap.
```

- [ ] **Step 5 (if all green): final no-op commit to checkpoint**

Skip if no working-tree changes remain.

---

## Self-review (executed at write time, not by the implementer)

**Spec coverage:**

| PRD requirement | Implementing task |
|---|---|
| §4.2 requirement-evaluation contract | Tasks 10–11 |
| §4.3 fixed-point cross-asset qualification | Task 11 |
| §4.4 re-evaluation triggers (STAT_CHANGED filter expansion + asset events) | Tasks 17–18 |
| §4.5 equipment template fetch + cache | Tasks 6–9 |
| §4.6 per-asset snapshot in registry | Tasks 3, 5, 14 |
| §4.7 unit + integration tests | Tasks 9, 10, 11, 19, 20, 21 |
| §6 in-process state (template cache + snapshot map) | Tasks 4, 5, 8, 14 |
| §7 file-level changes in atlas-effective-stats | Tasks 1, 4, 5, 6, 7, 8, 15, 16, 17, 18 |
| §8 failure modes & observability (WARN on cold-cache fail; DEBUG on gate decisions/recompute counts) | Tasks 8, 15 |
| §10 acceptance criteria | Tasks 19–22 |

**Placeholder scan:** No "TODO/TBD/implement later/similar to" placeholders remain. Two test placeholders in Task 18 are deliberate `t.Skip(...)` markers because the same coverage lands in Task 20 with proper HTTP stubs — that's noted inline.

**Type consistency check:** `Provider`, `EquipmentRequirements`, `EquippedAsset`, `WearerProfile`, `AppliedStats` all use the same names everywhere. `AddEquipmentBonuses` 4-arg signature is consistent across processor interface, impl, and consumer. `RecomputeWith` vs `RecomputeEquipmentBonuses`: the former is the pure model method, the latter is the processor entry point — both names appear consistently.

**Reqs/design gap closed:** The reqJob bitmask gotcha (design §4.1 line 220) is corrected via the `wearerClassMask` helper added in Task 10 and explicitly documented in `context.md`.

---

## Execution Handoff

Plan saved to `docs/tasks/task-053-equip-stat-requirements/plan.md` and context to `context.md`.

This is Phase 3 of the Atlas four-phase workflow; the user will run `/clear` then `/execute-task task-053` separately. Do NOT auto-invoke the execution skill from this session.
