# Character Presets Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a `presets` array to the existing `characters` configuration document and an admin-facing flow that materializes a preset into a fully-realized character via the existing `CharacterCreation` saga, with deterministic equipment stats.

**Architecture:** Preset storage rides on the existing atlas-configurations `characters` resource (sibling array next to `templates`). The factory exposes `POST /factory/characters/from-preset` that resolves the preset, validates name/ids, then emits the same saga used by player creation — augmented with new payload fields `UseAverageStats` (per-equipment determinism), `Gm`, and `Meso`. atlas-inventory honors `UseAverageStats` by writing atlas-data defaults verbatim. The UI surfaces an Apply Preset dialog and an Admin Bootstrap wizard.

**Tech Stack:** Go (Gorilla mux, GORM, Kafka via libs/atlas-kafka, JSON:API via api2go/jsonapi), TypeScript (React 19, TanStack React Query, react-hook-form + Zod, shadcn/ui).

**Conventions:**
- All Go file paths are absolute from repo root: `services/<service>/atlas.com/<service>/...`.
- Each service has its own `go.mod`; build/test inside that directory.
- TDD discipline: write the failing test, run it, write minimal impl, run it green, commit. Skip a unit test only when explicitly noted (e.g., wiring-only commits).
- Commits are bite-sized — one task = one commit unless noted.

**Read first:** `docs/tasks/task-037-character-presets/context.md` (key files and decision summary), then `design.md` for the *why*.

---

## Phase 1 — Shared saga library

Foundation. Done first because every downstream change keys off these struct fields.

### Task 1: Add `UseAverageStats` to `CreateAndEquipAssetPayload`

**Files:**
- Modify: `libs/atlas-saga/payloads.go:126-130`
- Test: `libs/atlas-saga/unmarshal_test.go`

- [ ] **Step 1: Write failing decode test**

Append to `libs/atlas-saga/unmarshal_test.go`:

```go
func TestCreateAndEquipAssetPayload_UseAverageStats_RoundTrip(t *testing.T) {
	in := CreateAndEquipAssetPayload{
		CharacterId:     42,
		Item:            ItemPayload{TemplateId: 1002357, Quantity: 1},
		UseAverageStats: true,
	}
	bs, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(bs), `"useAverageStats":true`) {
		t.Fatalf("expected useAverageStats:true in payload, got %s", string(bs))
	}
	var out CreateAndEquipAssetPayload
	if err := json.Unmarshal(bs, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !out.UseAverageStats {
		t.Fatalf("expected UseAverageStats=true after round-trip, got false")
	}

	// Backwards-compat: missing field decodes to false.
	var legacy CreateAndEquipAssetPayload
	if err := json.Unmarshal([]byte(`{"characterId":7,"item":{"templateId":1,"quantity":1}}`), &legacy); err != nil {
		t.Fatalf("legacy unmarshal: %v", err)
	}
	if legacy.UseAverageStats {
		t.Fatalf("expected legacy payload to default UseAverageStats=false")
	}
}
```

If `strings` or `json` aren't already imported in this test file, add them.

- [ ] **Step 2: Run the test to confirm it fails**

```bash
cd libs/atlas-saga && go test -run TestCreateAndEquipAssetPayload_UseAverageStats_RoundTrip -count=1 ./...
```

Expected: FAIL — `unknown field UseAverageStats` (or `field UseAverageStats not found`).

- [ ] **Step 3: Add the field**

In `libs/atlas-saga/payloads.go`, replace the struct:

```go
// CreateAndEquipAssetPayload represents the payload required to create and equip an asset.
type CreateAndEquipAssetPayload struct {
	CharacterId     uint32      `json:"characterId"`               // CharacterId associated with the action
	Item            ItemPayload `json:"item"`                      // Item to create and equip
	UseAverageStats bool        `json:"useAverageStats,omitempty"` // When true, atlas-inventory writes atlas-data defaults verbatim (no variance roll)
}
```

- [ ] **Step 4: Run the test to confirm it passes**

```bash
cd libs/atlas-saga && go test -run TestCreateAndEquipAssetPayload_UseAverageStats_RoundTrip -count=1 ./...
```

Expected: PASS.

- [ ] **Step 5: Run the full lib test suite to confirm no regressions**

```bash
cd libs/atlas-saga && go test ./... -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add libs/atlas-saga/payloads.go libs/atlas-saga/unmarshal_test.go
git commit -m "feat(atlas-saga): add UseAverageStats to CreateAndEquipAssetPayload

Per task-037 design §2.4 / D-1. Default false preserves existing
player-creation/shop/drop behaviour; presets opt in per-equipment-entry."
```

---

### Task 2: Add `Gm` and `Meso` to `CharacterCreatePayload`

**Files:**
- Modify: `libs/atlas-saga/payloads.go:588-610`
- Test: `libs/atlas-saga/unmarshal_test.go`

- [ ] **Step 1: Write failing decode test**

Append to `libs/atlas-saga/unmarshal_test.go`:

```go
func TestCharacterCreatePayload_GmAndMeso_RoundTrip(t *testing.T) {
	in := CharacterCreatePayload{
		AccountId: 1,
		Name:      "AdminHero",
		Gm:        2,
		Meso:      100_000_000,
	}
	bs, _ := json.Marshal(in)
	if !strings.Contains(string(bs), `"gm":2`) || !strings.Contains(string(bs), `"meso":100000000`) {
		t.Fatalf("expected gm/meso in payload, got %s", string(bs))
	}
	var out CharacterCreatePayload
	if err := json.Unmarshal(bs, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Gm != 2 || out.Meso != 100_000_000 {
		t.Fatalf("expected gm=2 meso=1e8, got gm=%d meso=%d", out.Gm, out.Meso)
	}

	// Backwards-compat: legacy payload defaults both to zero.
	var legacy CharacterCreatePayload
	if err := json.Unmarshal([]byte(`{"accountId":1,"name":"Foo"}`), &legacy); err != nil {
		t.Fatalf("legacy: %v", err)
	}
	if legacy.Gm != 0 || legacy.Meso != 0 {
		t.Fatalf("expected gm=0 meso=0 from legacy payload")
	}
}
```

- [ ] **Step 2: Run the test to confirm it fails**

```bash
cd libs/atlas-saga && go test -run TestCharacterCreatePayload_GmAndMeso_RoundTrip -count=1 ./...
```

Expected: FAIL.

- [ ] **Step 3: Add the fields**

In `libs/atlas-saga/payloads.go`, after `MapId`:

```go
type CharacterCreatePayload struct {
	AccountId    uint32   `json:"accountId"`
	WorldId      world.Id `json:"worldId"`
	Name         string   `json:"name"`
	Gender       byte     `json:"gender"`
	Level        byte     `json:"level"`
	Strength     uint16   `json:"strength"`
	Dexterity    uint16   `json:"dexterity"`
	Intelligence uint16   `json:"intelligence"`
	Luck         uint16   `json:"luck"`
	JobId        job.Id   `json:"jobId"`
	Hp           uint16   `json:"hp"`
	Mp           uint16   `json:"mp"`
	Face         uint32   `json:"face"`
	Hair         uint32   `json:"hair"`
	Skin         byte     `json:"skin"`
	Top          uint32   `json:"top"`
	Bottom       uint32   `json:"bottom"`
	Shoes        uint32   `json:"shoes"`
	Weapon       uint32   `json:"weapon"`
	MapId        _map.Id  `json:"mapId"`
	Gm           int      `json:"gm,omitempty"`
	Meso         uint32   `json:"meso,omitempty"`
}
```

- [ ] **Step 4: Run the test to confirm it passes**

```bash
cd libs/atlas-saga && go test ./... -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-saga/payloads.go libs/atlas-saga/unmarshal_test.go
git commit -m "feat(atlas-saga): add Gm and Meso to CharacterCreatePayload

Per task-037 design §2.3. Plumbed through orchestrator + atlas-character
in later tasks. omitempty keeps existing emitters wire-compatible."
```

---

## Phase 2 — atlas-data: skill MaxLevel + ids filter

### Task 3: Add `MaxLevel` field to skill REST model and populate it

**Files:**
- Modify: `services/atlas-data/atlas.com/data/skill/rest.go`
- Modify: `services/atlas-data/atlas.com/data/skill/processor.go` (loader)
- Test: `services/atlas-data/atlas.com/data/skill/rest_test.go`
- Test: `services/atlas-data/atlas.com/data/skill/reader_test.go` (or processor test if MaxLevel set in loader)

- [ ] **Step 1: Read the skill loader**

```bash
grep -n "MaxLevel\|maxLevel\|len(.*Effects)\|func.*Skill\b" services/atlas-data/atlas.com/data/skill/processor.go services/atlas-data/atlas.com/data/skill/reader.go | head
```

Identify where the per-skill `Effects[]` slice is populated; that count is the natural max level. Use that location to set `MaxLevel`.

- [ ] **Step 2: Write a failing rest serialization test**

Append to `services/atlas-data/atlas.com/data/skill/rest_test.go`:

```go
func TestRestModel_MaxLevel_Serialization(t *testing.T) {
	rm := RestModel{Id: 1121008, Name: "Hero's Will", MaxLevel: 5}
	bs, err := json.Marshal(rm)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(bs), `"maxLevel":5`) {
		t.Fatalf("expected maxLevel:5 in JSON, got %s", string(bs))
	}
}
```

(Add `encoding/json` and `strings` imports if missing.)

- [ ] **Step 3: Run to confirm fail**

```bash
cd services/atlas-data/atlas.com/data && go test -run TestRestModel_MaxLevel_Serialization ./skill -count=1
```

Expected: FAIL — `unknown field MaxLevel`.

- [ ] **Step 4: Add the field to the rest model**

Replace the struct in `services/atlas-data/atlas.com/data/skill/rest.go`:

```go
type RestModel struct {
	Id            uint32             `json:"-"`
	Name          string             `json:"name"`
	Description   string             `json:"description"`
	Action        bool               `json:"action"`
	Element       string             `json:"element"`
	AnimationTime uint32             `json:"animationTime"`
	MaxLevel      uint8              `json:"maxLevel"`
	Effects       []effect.RestModel `json:"effects"`
}
```

- [ ] **Step 5: Populate `MaxLevel` in the loader**

In whichever skill loader builds `RestModel` from WZ/JSON, set `MaxLevel` from the count of per-level effect entries. Sample (adjust to actual loader signature):

```go
rm.MaxLevel = uint8(len(rm.Effects))
```

If `len(Effects)` exceeds 255 (it shouldn't for any real skill), clamp:

```go
if n := len(rm.Effects); n > 255 {
	rm.MaxLevel = 255
} else {
	rm.MaxLevel = uint8(n)
}
```

- [ ] **Step 6: Add a loader test confirming MaxLevel is set**

Append a focused test exercising the loader path. If the loader is private, test through the public `Storage.GetById` path with a fixture skill known to have N effect entries. Sample in `services/atlas-data/atlas.com/data/skill/processor_test.go` (create if absent):

```go
func TestSkillLoader_PopulatesMaxLevel(t *testing.T) {
	// Arrange a fixture skill with 30 effect entries; load via the loader
	// helper used by the existing reader tests. Skim reader_test.go for
	// the established fixture style and reuse it.
	// Then assert: skill.MaxLevel == 30.
	t.Skip("fill in once loader fixture style is known; see reader_test.go for pattern")
}
```

If the loader test is awkward to write quickly, leave the `t.Skip` in place but file a TODO comment referring to task-037; the rest serialization test plus the live-loaded test in the next phase suffice for confidence. **Do not** ship without verifying `MaxLevel` is non-zero for a real skill in step 7.

- [ ] **Step 7: Run all skill tests**

```bash
cd services/atlas-data/atlas.com/data && go test ./skill/... -count=1
```

Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add services/atlas-data/atlas.com/data/skill/rest.go services/atlas-data/atlas.com/data/skill/processor.go services/atlas-data/atlas.com/data/skill/rest_test.go
git commit -m "feat(atlas-data): expose MaxLevel on skill rest model

Loader sets MaxLevel from the count of per-level effect entries.
Required by task-037 character presets so the factory can derive a
correct masterLevel without an undocumented len(Effects) convention."
```

---

### Task 4: Add `ids=` filter to `GET /data/skills`

**Files:**
- Modify: `services/atlas-data/atlas.com/data/skill/resource.go:28-61`
- Test: `services/atlas-data/atlas.com/data/skill/resource_test.go`

- [ ] **Step 1: Write failing handler test**

Append to `services/atlas-data/atlas.com/data/skill/resource_test.go`. Reuse whatever harness `resource_test.go` already uses to seed skills and call the handler. Pseudocode:

```go
func TestHandleSearchSkillsRequest_IdsFilter(t *testing.T) {
	// Seed three skills with ids 1121008, 1121009, 9999999.
	// GET /data/skills?ids=1121008,9999999 → 200 with both ids returned.
	// GET /data/skills?ids=1111111 → 200 with empty list.
	// GET /data/skills?ids=1121008&ids=9999999 → 200 with both ids.
	// GET /data/skills?ids=1121008&name=ignored → 200 with id 1121008
	//   (ids filter takes precedence; name=  is ignored).
	t.Skip("fill in using the existing resource_test.go harness pattern")
}
```

If `resource_test.go` doesn't yet exist or the test infra is missing, the equivalent assertions can live in `rest_test.go` against an in-memory skill list — what matters is that we verify all three branches: comma-separated ids, repeated `ids=` params, and ids+name combo behaviour.

- [ ] **Step 2: Run to confirm fail**

```bash
cd services/atlas-data/atlas.com/data && go test ./skill/... -count=1 -run IdsFilter
```

Expected: FAIL.

- [ ] **Step 3: Implement the filter**

Replace the body of `handleSearchSkillsRequest` in `services/atlas-data/atlas.com/data/skill/resource.go`:

```go
func handleSearchSkillsRequest(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			query := r.URL.Query()
			idParams := query["ids"]
			nameQuery := query.Get("name")

			if len(idParams) == 0 && nameQuery == "" {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			s := NewStorage(d.Logger(), db)
			allSkills, err := s.GetAll(d.Context())
			if err != nil {
				d.Logger().WithError(err).Debugf("Unable to get all skills.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			results := make([]RestModel, 0)

			if len(idParams) > 0 {
				idSet := make(map[uint32]struct{})
				for _, raw := range idParams {
					for _, part := range strings.Split(raw, ",") {
						part = strings.TrimSpace(part)
						if part == "" {
							continue
						}
						id, err := strconv.ParseUint(part, 10, 32)
						if err != nil {
							w.WriteHeader(http.StatusBadRequest)
							return
						}
						idSet[uint32(id)] = struct{}{}
					}
				}
				for _, sk := range allSkills {
					if _, ok := idSet[sk.Id]; ok {
						results = append(results, sk)
					}
				}
			} else {
				nameQueryLower := strings.ToLower(nameQuery)
				for _, sk := range allSkills {
					if strings.Contains(strings.ToLower(sk.Name), nameQueryLower) {
						results = append(results, sk)
						if len(results) >= 10 {
							break
						}
					}
				}
			}

			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(results)
		}
	}
}
```

Notes:
- `ids=` takes precedence over `name=` (per design D-5).
- The 10-result cap is *only* applied to the `name=` substring path (its original purpose). `ids=` returns all matches uncapped.
- Malformed `ids` element ⇒ `400`.

- [ ] **Step 4: Run to confirm pass**

```bash
cd services/atlas-data/atlas.com/data && go test ./skill/... -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-data/atlas.com/data/skill/resource.go services/atlas-data/atlas.com/data/skill/resource_test.go
git commit -m "feat(atlas-data): add ids= filter to GET /data/skills

Per task-037 design D-5. ids= takes precedence over name=, lifts the
10-result cap, returns 400 on malformed ids."
```

---

## Phase 3 — atlas-character

### Task 5: Add `Gm` and `Meso` to atlas-character `CreateCharacterCommandBody`

**Files:**
- Modify: `services/atlas-character/atlas.com/character/kafka/message/character/kafka.go:62-79`
- Test: extend the closest decode test (search for `CreateCharacterCommandBody` under that service's test files; if none, add a focused test next to `kafka.go`).

- [ ] **Step 1: Write failing test**

Add `services/atlas-character/atlas.com/character/kafka/message/character/kafka_test.go` (create if absent):

```go
package character

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCreateCharacterCommandBody_GmMeso_RoundTrip(t *testing.T) {
	in := CreateCharacterCommandBody{Gm: 2, Meso: 12345}
	bs, _ := json.Marshal(in)
	if !strings.Contains(string(bs), `"gm":2`) || !strings.Contains(string(bs), `"meso":12345`) {
		t.Fatalf("expected gm/meso in JSON, got %s", string(bs))
	}
	var out CreateCharacterCommandBody
	if err := json.Unmarshal(bs, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Gm != 2 || out.Meso != 12345 {
		t.Fatalf("expected gm=2 meso=12345, got gm=%d meso=%d", out.Gm, out.Meso)
	}
}
```

- [ ] **Step 2: Run to confirm fail**

```bash
cd services/atlas-character/atlas.com/character && go test ./kafka/message/character/... -count=1
```

Expected: FAIL.

- [ ] **Step 3: Add the fields**

In `kafka.go`, replace `CreateCharacterCommandBody`:

```go
type CreateCharacterCommandBody struct {
	AccountId    uint32   `json:"accountId"`
	WorldId      world.Id `json:"worldId"`
	Name         string   `json:"name"`
	Level        byte     `json:"level"`
	Strength     uint16   `json:"strength"`
	Dexterity    uint16   `json:"dexterity"`
	Intelligence uint16   `json:"intelligence"`
	Luck         uint16   `json:"luck"`
	MaxHp        uint16   `json:"maxHp"`
	MaxMp        uint16   `json:"maxMp"`
	JobId        job.Id   `json:"jobId"`
	Gender       byte     `json:"gender"`
	Hair         uint32   `json:"hair"`
	Face         uint32   `json:"face"`
	SkinColor    byte     `json:"skinColor"`
	MapId        _map.Id  `json:"mapId"`
	Gm           int      `json:"gm,omitempty"`
	Meso         uint32   `json:"meso,omitempty"`
}
```

- [ ] **Step 4: Run to confirm pass**

```bash
cd services/atlas-character/atlas.com/character && go test ./kafka/message/character/... -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-character/atlas.com/character/kafka/message/character/
git commit -m "feat(atlas-character): add Gm and Meso to CreateCharacterCommandBody

Per task-037 design §2.3. Consumer wiring follows in next task."
```

---

### Task 6: `handleCreateCharacter` calls `SetGm` + `SetMeso`

**Files:**
- Modify: `services/atlas-character/atlas.com/character/kafka/consumer/character/consumer.go:333-374`
- Test: `services/atlas-character/atlas.com/character/kafka/consumer/character/consumer_test.go` (or wherever `handleCreateCharacter` is unit-tested)

- [ ] **Step 1: Locate or create the consumer test**

```bash
grep -rn "handleCreateCharacter" services/atlas-character/atlas.com/character/ | head
```

If a test exists, augment it; if not, write a focused test using the in-memory test harness already used elsewhere in atlas-character (look for `kafka_integration_test.go` patterns).

- [ ] **Step 2: Write failing test asserting Gm and Meso land on the model**

Skeleton (adapt to the actual harness in `character/processor_test.go` + `kafka_integration_test.go`):

```go
func TestHandleCreateCharacter_PersistsGmAndMeso(t *testing.T) {
	// Set up an in-memory db + bus per existing pattern.
	// Build a Command[CreateCharacterCommandBody] with Gm=2, Meso=1_000_000.
	// Invoke handleCreateCharacter(db).Handle(...).
	// Read back the persisted character; assert Gm()==2 and Meso()==1_000_000.
	t.Skip("fill in using existing test harness")
}
```

The Skip is acceptable only if no in-package harness exists; in that case a black-box assertion at processor level (`Create` honors a model with Gm/Meso set) plus a manual review against `consumer.go` change is enough. Prefer wiring an actual test if patterns exist.

- [ ] **Step 3: Run to confirm fail (or skip)**

```bash
cd services/atlas-character/atlas.com/character && go test ./kafka/consumer/character/... -count=1
```

- [ ] **Step 4: Add the builder calls**

In `consumer.go:339-356`, insert `SetGm` and `SetMeso` chain calls. Replace the model construction block:

```go
		model := character.NewModelBuilder().
			SetAccountId(c.Body.AccountId).
			SetWorldId(c.Body.WorldId).
			SetName(c.Body.Name).
			SetLevel(c.Body.Level).
			SetStrength(c.Body.Strength).
			SetDexterity(c.Body.Dexterity).
			SetIntelligence(c.Body.Intelligence).
			SetLuck(c.Body.Luck).
			SetMaxHp(c.Body.MaxHp).SetHp(c.Body.MaxHp).
			SetMaxMp(c.Body.MaxMp).SetMp(c.Body.MaxMp).
			SetJobId(c.Body.JobId).
			SetGender(c.Body.Gender).
			SetHair(c.Body.Hair).
			SetFace(c.Body.Face).
			SetSkinColor(c.Body.SkinColor).
			SetMapId(c.Body.MapId).
			SetGm(c.Body.Gm).
			SetMeso(c.Body.Meso).
			Build()
```

- [ ] **Step 5: Run tests to confirm pass**

```bash
cd services/atlas-character/atlas.com/character && go test ./... -count=1
```

Expected: PASS (existing player-creation traffic continues to work because both fields default to zero).

- [ ] **Step 6: Commit**

```bash
git add services/atlas-character/atlas.com/character/kafka/consumer/character/
git commit -m "feat(atlas-character): persist Gm and Meso from CreateCharacterCommandBody

Per task-037 design §2.3. Zero defaults preserve current player-creation
behaviour; preset traffic supplies non-zero values."
```

---

### Task 7: New `GET /characters/name-validity` endpoint with worldId scope

**Files:**
- Modify: `services/atlas-character/atlas.com/character/character/processor.go:55, :196-218`
- Modify: `services/atlas-character/atlas.com/character/character/resource.go`
- Create: `services/atlas-character/atlas.com/character/character/name_validity_resource.go`
- Test: `services/atlas-character/atlas.com/character/character/name_validity_resource_test.go`

- [ ] **Step 1: Widen `IsValidName` to take a `worldId` argument and a structured reason**

Replace the relevant method on `ProcessorImpl` (lines around 196-218 and the interface entry at 55) with:

```go
type NameValidityResult struct {
	Valid  bool
	Reason string // empty when Valid; otherwise one of "regex","length","blocked","duplicate"
	Detail string
}

// In the Processor interface (around line 55), replace IsValidName with:
//   IsValidName(name string) (bool, error)
//   CheckNameValidity(name string, worldId world.Id) (NameValidityResult, error)
//
// Existing internal callers continue to use IsValidName.

func (p *ProcessorImpl) IsValidName(name string) (bool, error) {
	res, err := p.CheckNameValidity(name, 0)
	if err != nil {
		return false, err
	}
	// Tenant-scoped duplicate check is acceptable for legacy callers.
	if !res.Valid && res.Reason == "duplicate" {
		// fall through; existing tenant-only duplicate rule
	}
	return res.Valid, nil
}

func (p *ProcessorImpl) CheckNameValidity(name string, worldId world.Id) (NameValidityResult, error) {
	if len(name) < 3 || len(name) > 12 {
		return NameValidityResult{Valid: false, Reason: "length", Detail: "Name must be 3-12 characters."}, nil
	}
	m, err := regexp.MatchString("[A-Za-z0-9぀-ゟ゠-ヿ一-龯]{3,12}", name)
	if err != nil {
		return NameValidityResult{}, err
	}
	if !m {
		return NameValidityResult{Valid: false, Reason: "regex", Detail: "Name contains invalid characters."}, nil
	}

	cs, err := p.GetForName()(name)
	if err != nil {
		return NameValidityResult{}, err
	}
	if len(cs) > 0 {
		// World-scoped duplicate check when worldId != 0; otherwise any-tenant match counts.
		// (worldId=0 path keeps legacy IsValidName behaviour.)
		for _, c := range cs {
			if worldId == 0 || c.WorldId() == worldId {
				return NameValidityResult{Valid: false, Reason: "duplicate", Detail: "Name already taken."}, nil
			}
		}
	}

	// TODO blocked-name list (existing TODO at processor.go:210-214 stays a TODO).
	return NameValidityResult{Valid: true}, nil
}
```

If the existing `IsValidName` interface contract requires a strict tenant-scoped duplicate (no world filter), preserve it by returning `false` for the legacy method whenever `len(cs) > 0`. The world-scoped behaviour applies only to `CheckNameValidity` callers.

- [ ] **Step 2: Add the HTTP handler**

Create `services/atlas-character/atlas.com/character/character/name_validity_resource.go`:

```go
package character

import (
	"atlas-character/rest"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type NameValidityResponse struct {
	Valid  bool   `json:"valid"`
	Reason string `json:"reason,omitempty"`
	Detail string `json:"detail,omitempty"`
}

func handleGetNameValidity(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			q := r.URL.Query()
			name := q.Get("name")
			widRaw := q.Get("worldId")
			if name == "" || widRaw == "" {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			wid, err := strconv.ParseUint(widRaw, 10, 8)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			res, err := NewProcessor(d.Logger(), d.Context(), db).CheckNameValidity(name, world.Id(wid))
			if err != nil {
				d.Logger().WithError(err).Error("name-validity check failed")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(NameValidityResponse{
				Valid:  res.Valid,
				Reason: res.Reason,
				Detail: res.Detail,
			})
			_ = jsonapi.ParseQueryFields(&q) // satisfies the unused-import path
			_ = server.SafeMarshalResponse  // confirm rest util is wired (remove if not used)
		}
	}
}
```

(Trim unused imports/refs above as you copy. The body above intentionally writes plain JSON — the response is not a JSON:API resource collection, just a small object.)

- [ ] **Step 3: Register the route**

In `services/atlas-character/atlas.com/character/character/resource.go`, in the same `InitResource` registration that already binds character routes, add:

```go
r.HandleFunc("/characters/name-validity", rest.RegisterHandler(l)(si)("get_name_validity", handleGetNameValidity(db))).Methods(http.MethodGet)
```

(Use the existing pattern in this file as the template — match indent, helper names, and ordering.)

- [ ] **Step 4: Write handler tests**

Create `services/atlas-character/atlas.com/character/character/name_validity_resource_test.go`. Cover:

```go
// 1. valid name → 200 {"valid":true}
// 2. too short → 200 {"valid":false,"reason":"length"}
// 3. invalid char → 200 {"valid":false,"reason":"regex"}
// 4. duplicate in target world → 200 {"valid":false,"reason":"duplicate"}
// 5. duplicate only in *other* world → 200 {"valid":true}  (world-scoped)
// 6. missing name or worldId → 400
// 7. malformed worldId → 400
```

Use whatever harness `processor_test.go` / `rest_test.go` use to seed in-memory characters; replicate.

- [ ] **Step 5: Run all atlas-character tests**

```bash
cd services/atlas-character/atlas.com/character && go test ./... -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-character/atlas.com/character/character/processor.go services/atlas-character/atlas.com/character/character/resource.go services/atlas-character/atlas.com/character/character/name_validity_resource.go services/atlas-character/atlas.com/character/character/name_validity_resource_test.go
git commit -m "feat(atlas-character): add GET /characters/name-validity

Per task-037 design D-6. Widens duplicate-name check with worldId scope
and exposes a structured reason (regex/length/duplicate). Existing
IsValidName callers retain tenant-scoped behaviour."
```

---

## Phase 4 — atlas-saga-orchestrator: thread Gm/Meso and UseAverageStats

### Task 8: Add `Gm` + `Meso` to orchestrator's `CreateCharacterCommandBody` and propagate

**Files:**
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/message/character/kafka.go:177-194`
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/character/processor.go:44, :208-211`
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/character/producer.go:231-238`
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/character/mock/processor.go:247-249` (mock signature)
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/handler.go:1394` (handler call)
- Test: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/character/processor_test.go` if present, else extend `producer_test.go`

- [ ] **Step 1: Write failing producer test**

Append a test asserting the produced kafka message body includes `"gm":N` and `"meso":N`. Pattern: similar to existing producer assertions in this package (search `RequestCreateCharacterProvider` callers in `_test.go`).

- [ ] **Step 2: Run to confirm fail**

```bash
cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go test ./character/... -count=1
```

- [ ] **Step 3: Add fields to `CreateCharacterCommandBody`**

In `kafka/message/character/kafka.go:177-194`:

```go
type CreateCharacterCommandBody struct {
	AccountId    uint32   `json:"accountId"`
	WorldId      world.Id `json:"worldId"`
	Name         string   `json:"name"`
	Level        byte     `json:"level"`
	Strength     uint16   `json:"strength"`
	Dexterity    uint16   `json:"dexterity"`
	Intelligence uint16   `json:"intelligence"`
	Luck         uint16   `json:"luck"`
	MaxHp        uint16   `json:"maxHp"`
	MaxMp        uint16   `json:"maxMp"`
	JobId        job.Id   `json:"jobId"`
	Gender       byte     `json:"gender"`
	Hair         uint32   `json:"hair"`
	Face         uint32   `json:"face"`
	SkinColor    byte     `json:"skinColor"`
	MapId        _map.Id  `json:"mapId"`
	Gm           int      `json:"gm,omitempty"`
	Meso         uint32   `json:"meso,omitempty"`
}
```

- [ ] **Step 4: Extend producer signature**

In `character/producer.go:231-238` (`RequestCreateCharacterProvider`), add `gm int, meso uint32` parameters at the end and set the fields on `CreateCharacterCommandBody`.

- [ ] **Step 5: Extend processor interface and impl**

In `character/processor.go`:

- Interface (line 44): append `gm int, meso uint32` to `RequestCreateCharacter`.
- Impl (line 208): same; pass through to provider.

- [ ] **Step 6: Update mock**

In `character/mock/processor.go:247-249`: same signature change.

- [ ] **Step 7: Update handler call site**

In `saga/handler.go:1394`, pass `payload.Gm, payload.Meso`:

```go
err := h.charP.RequestCreateCharacter(s.TransactionId(), payload.AccountId, payload.WorldId, payload.Name, payload.Level, payload.Strength, payload.Dexterity, payload.Intelligence, payload.Luck, payload.Hp, payload.Mp, payload.JobId, payload.Gender, payload.Face, payload.Hair, payload.Skin, payload.MapId, payload.Gm, payload.Meso)
```

- [ ] **Step 8: Run all orchestrator tests**

```bash
cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go test ./... -count=1
```

Expected: PASS. Existing player-creation flow propagates `Gm=0, Meso=0` (zero values), preserving current behaviour.

- [ ] **Step 9: Commit**

```bash
git add services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/character/ services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/message/character/ services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/handler.go
git commit -m "feat(atlas-saga-orchestrator): thread Gm and Meso through create_character

Per task-037 design §2.3. Producer + processor + handler + mock all
gain the two fields; player-creation traffic continues to pass zero."
```

---

### Task 9: Add `UseAverageStats` to orchestrator `CreateAssetCommandBody` and split create-item APIs

**Files:**
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/message/compartment/kafka.go:119` (`CreateAssetCommandBody`)
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/compartment/producer.go:15-32`
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/compartment/processor.go:30-104`
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/compartment/mock/processor.go` (mock, find via grep)
- Test: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/compartment/processor_test.go` (if present) and producer test

- [ ] **Step 1: Write failing test**

Add a test asserting that `RequestCreateAndEquipAsset` with `payload.UseAverageStats=true` produces a kafka command body whose JSON contains `"useAverageStats":true`. Use the existing producer-test pattern.

- [ ] **Step 2: Run to confirm fail**

```bash
cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go test ./compartment/... -count=1
```

- [ ] **Step 3: Add the field on the command body**

In `kafka/message/compartment/kafka.go:119`, replace:

```go
type CreateAssetCommandBody struct {
	TemplateId      uint32    `json:"templateId"`
	Quantity        uint32    `json:"quantity"`
	Expiration      time.Time `json:"expiration"`
	OwnerId         uint32    `json:"ownerId"`
	Flag            uint16    `json:"flag"`
	Rechargeable    uint64    `json:"rechargeable"`
	UseAverageStats bool      `json:"useAverageStats,omitempty"`
}
```

- [ ] **Step 4: Extend the producer**

In `compartment/producer.go:15-32`, add `useAverageStats bool` to `RequestCreateAssetCommandProvider` and forward it onto the command body:

```go
func RequestCreateAssetCommandProvider(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, templateId uint32, quantity uint32, expiration time.Time, useAverageStats bool) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &compartment.Command[compartment.CreateAssetCommandBody]{
		TransactionId: transactionId,
		CharacterId:   characterId,
		InventoryType: byte(inventoryType),
		Type:          compartment.CommandCreateAsset,
		Body: compartment.CreateAssetCommandBody{
			TemplateId:      templateId,
			Quantity:        quantity,
			Expiration:      expiration,
			OwnerId:         0,
			Flag:            0,
			Rechargeable:    0,
			UseAverageStats: useAverageStats,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
```

- [ ] **Step 5: Split create-item APIs**

In `compartment/processor.go`, leave `RequestCreateItem` signature unchanged (back-compat) and add a new method that takes the flag:

```go
type Processor interface {
	RequestCreateItem(transactionId uuid.UUID, characterId uint32, templateId uint32, quantity uint32, expiration time.Time) error
	RequestCreateItemWithStats(transactionId uuid.UUID, characterId uint32, templateId uint32, quantity uint32, expiration time.Time, useAverageStats bool) error
	// ... unchanged ...
}

func (p *ProcessorImpl) RequestCreateItem(transactionId uuid.UUID, characterId uint32, templateId uint32, quantity uint32, expiration time.Time) error {
	return p.RequestCreateItemWithStats(transactionId, characterId, templateId, quantity, expiration, false)
}

func (p *ProcessorImpl) RequestCreateItemWithStats(transactionId uuid.UUID, characterId uint32, templateId uint32, quantity uint32, expiration time.Time, useAverageStats bool) error {
	inventoryType, ok := inventory.TypeFromItemId(item.Id(templateId))
	if !ok {
		return errors.New("invalid templateId")
	}
	return producer.ProviderImpl(p.l)(p.ctx)(compartment.EnvCommandTopic)(RequestCreateAssetCommandProvider(transactionId, characterId, inventoryType, templateId, quantity, expiration, useAverageStats))
}

func (p *ProcessorImpl) RequestCreateAndEquipAsset(transactionId uuid.UUID, payload CreateAndEquipAssetPayload) error {
	return p.RequestCreateItemWithStats(transactionId, payload.CharacterId, payload.Item.TemplateId, payload.Item.Quantity, payload.Item.Expiration, payload.UseAverageStats)
}
```

Also add `UseAverageStats bool` to the local `CreateAndEquipAssetPayload` struct (`compartment/processor.go:25-28`):

```go
type CreateAndEquipAssetPayload struct {
	CharacterId     uint32      `json:"characterId"`
	Item            ItemPayload `json:"item"`
	UseAverageStats bool        `json:"useAverageStats,omitempty"`
}
```

- [ ] **Step 6: Update the saga handler that constructs `CreateAndEquipAssetPayload`**

Search for where the orchestrator unmarshals the saga step payload `CreateAndEquipAssetPayload` (likely `saga/handler.go`) and ensure `UseAverageStats` propagates from the wire payload to the local payload struct passed to `RequestCreateAndEquipAsset`. The saga step payload is the shared `saga.CreateAndEquipAssetPayload` from libs/atlas-saga; the handler converts it to the orchestrator's local `compartment.CreateAndEquipAssetPayload` — copy the new field across.

```bash
grep -n "RequestCreateAndEquipAsset\|CreateAndEquipAssetPayload" services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/handler.go | head
```

Wherever the local payload is built, add:

```go
compartment.CreateAndEquipAssetPayload{
	CharacterId:     payload.CharacterId,
	Item:            compartment.ItemPayload{TemplateId: payload.Item.TemplateId, Quantity: payload.Item.Quantity, Expiration: payload.Item.Expiration},
	UseAverageStats: payload.UseAverageStats,
}
```

- [ ] **Step 7: Update the mock**

`services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/compartment/mock/processor.go` — add `RequestCreateItemWithStats` mock method matching the interface.

- [ ] **Step 8: Run tests**

```bash
cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go test ./... -count=1
```

Expected: PASS. Player-creation traffic continues to use the false default.

- [ ] **Step 9: Commit**

```bash
git add services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/
git commit -m "feat(atlas-saga-orchestrator): thread UseAverageStats through create_and_equip_asset

Per task-037 design D-1. Splits RequestCreateItem to preserve existing
callers; orchestrator handler copies the new field from the saga payload."
```

---

## Phase 5 — atlas-inventory: refactor + variance bypass

### Task 10: Refactor `asset.Create` to options struct (D-3)

**Files:**
- Modify: `services/atlas-inventory/atlas.com/inventory/asset/processor.go:276-355`
- Modify: `services/atlas-inventory/atlas.com/inventory/compartment/processor.go` (cascade through call sites)
- Test: `services/atlas-inventory/atlas.com/inventory/compartment/processor_test.go` (~14 sites — update construction syntax)

This is a mechanical refactor. No behaviour change yet; pure signature reshuffle. Test pass = success.

- [ ] **Step 1: Define `CreateOptions` and rewrite `Create` signature**

In `services/atlas-inventory/atlas.com/inventory/asset/processor.go`, replace the `Create` method with:

```go
type CreateOptions struct {
	Quantity        uint32
	Expiration      time.Time
	OwnerId         uint32
	Flag            uint16
	Rechargeable    uint64
	UseAverageStats bool
}

func (p *Processor) Create(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, compartmentId uuid.UUID, templateId uint32, slot int16, opts CreateOptions) (Model, error) {
	return func(transactionId uuid.UUID, characterId uint32, compartmentId uuid.UUID, templateId uint32, slot int16, opts CreateOptions) (Model, error) {
		p.l.Debugf("Character [%d] attempting to create [%d] item(s) [%d] in slot [%d] of compartment [%s].", characterId, opts.Quantity, templateId, slot, compartmentId.String())
		var a Model
		txErr := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
			inventoryType, ok := inventory.TypeFromItemId(item.Id(templateId))
			if !ok {
				return errors.New("unknown item type")
			}

			b := NewBuilder(compartmentId, templateId).
				SetSlot(slot).
				SetExpiration(opts.Expiration).
				SetCreatedAt(time.Now())

			switch inventoryType {
			case inventory.TypeValueEquip:
				ea, err := p.statProcessor.GetById(templateId)
				if err != nil {
					if errors.Is(err, requests.ErrNotFound) {
						p.l.WithError(err).Errorf("Equipment template [%d] not present in atlas-data; seed data is likely missing.", templateId)
					} else {
						p.l.WithError(err).Errorf("Unable to get equipment stats for item [%d].", templateId)
					}
					return err
				}
				b.SetStrength(getRandomStat(ea.Strength(), 5)).
					SetDexterity(getRandomStat(ea.Dexterity(), 5)).
					SetIntelligence(getRandomStat(ea.Intelligence(), 5)).
					SetLuck(getRandomStat(ea.Luck(), 5)).
					SetHp(getRandomStat(ea.Hp(), 10)).
					SetMp(getRandomStat(ea.Mp(), 10)).
					SetWeaponAttack(getRandomStat(ea.WeaponAttack(), 5)).
					SetMagicAttack(getRandomStat(ea.MagicAttack(), 5)).
					SetWeaponDefense(getRandomStat(ea.WeaponDefense(), 10)).
					SetMagicDefense(getRandomStat(ea.MagicDefense(), 10)).
					SetAccuracy(getRandomStat(ea.Accuracy(), 5)).
					SetAvoidability(getRandomStat(ea.Avoidability(), 5)).
					SetHands(getRandomStat(ea.Hands(), 5)).
					SetSpeed(getRandomStat(ea.Speed(), 5)).
					SetJump(getRandomStat(ea.Jump(), 5)).
					SetSlots(ea.Slots())
			case inventory.TypeValueUse, inventory.TypeValueSetup, inventory.TypeValueETC:
				b.SetQuantity(opts.Quantity).
					SetOwnerId(opts.OwnerId).
					SetFlag(opts.Flag).
					SetRechargeable(opts.Rechargeable)
			case inventory.TypeValueCash:
				if item.GetClassification(item.Id(templateId)) == item.ClassificationPet {
					pe, err := p.petProcessor.Create(characterId, templateId)
					if err != nil {
						return err
					}
					b.SetPetId(pe.Id()).
						SetCashId(pe.CashId()).
						SetOwnerId(pe.OwnerId()).
						SetFlag(pe.Flag()).
						SetExpiration(pe.Expiration()).
						SetPurchaseBy(pe.PurchaseBy())
				} else {
					b.SetQuantity(opts.Quantity).
						SetOwnerId(opts.OwnerId).
						SetFlag(opts.Flag)
				}
			}

			var err error
			a, err = create(tx, p.t.Id(), b.Build())
			if err != nil {
				return err
			}
			return mb.Put(asset.EnvEventTopicStatus, CreatedEventStatusProvider(transactionId, characterId, a))
		})
		if txErr != nil {
			return Model{}, txErr
		}
		return a, nil
	}
}
```

(This intermediate version keeps the variance branch unchanged. Variance bypass is the next task — TDD'd separately.)

- [ ] **Step 2: Update every internal caller**

```bash
grep -rn "assetProcessor.*\.Create(mb)\|p.Create(mb)" services/atlas-inventory/atlas.com/inventory/ | grep -v _test.go
```

Expect ~6 production call sites under `compartment/processor.go` (lines around `:948, :1018, :1030`, plus a few helpers). For each, rewrite from positional args:

```go
// Before
a, err = p.assetProcessor.WithTransaction(tx).Create(mb)(transactionId, characterId, c.Id(), templateId, nfs, quantity, expiration, ownerId, flag, rechargeable)
```

to:

```go
// After
a, err = p.assetProcessor.WithTransaction(tx).Create(mb)(transactionId, characterId, c.Id(), templateId, nfs, asset.CreateOptions{
	Quantity:     quantity,
	Expiration:   expiration,
	OwnerId:      ownerId,
	Flag:         flag,
	Rechargeable: rechargeable,
})
```

`UseAverageStats` is omitted (defaults to false).

Add `asset.` import alias if not already in scope.

- [ ] **Step 3: Update tests**

```bash
grep -rn "assetProcessor.*\.Create(mb)\|asset\.Create\b" services/atlas-inventory/atlas.com/inventory/ | grep _test.go
```

For every test calling `Create(mb)(...)` with positional args, switch to the options struct.

- [ ] **Step 4: Build + run inventory tests**

```bash
cd services/atlas-inventory/atlas.com/inventory && go build ./... && go test ./... -count=1
```

Expected: PASS, with no behaviour change. (Compile errors caught here pinpoint missed call sites.)

- [ ] **Step 5: Commit**

```bash
git add services/atlas-inventory/atlas.com/inventory/
git commit -m "refactor(atlas-inventory): asset.Create takes CreateOptions struct

Per task-037 design D-3. Mechanical signature change; behaviour unchanged.
Sets up the upcoming UseAverageStats branch and the long-pending UNTRADEABLE
flag (TODO at processor.go:293)."
```

---

### Task 11: Variance bypass when `UseAverageStats=true`

**Files:**
- Modify: `services/atlas-inventory/atlas.com/inventory/asset/processor.go` (the equip branch of `Create`)
- Test: `services/atlas-inventory/atlas.com/inventory/asset/processor_test.go` (or wherever an equip-creation unit test lives; if absent, create one)

- [ ] **Step 1: Write failing test**

Locate or create `services/atlas-inventory/atlas.com/inventory/asset/processor_test.go`. The test should:

1. Set up a fake `statProcessor` returning a known `EquipableArrangement` (e.g., Strength=10, Dexterity=10, ...).
2. Invoke `Create(mb)(... CreateOptions{UseAverageStats: true})` on an equip-classified template id.
3. Assert the persisted asset's stats *exactly* match the input EA, with no variance.
4. Repeat with `UseAverageStats: false`; assert stats fall in the variance window `[base - δ, base + δ]` for at least one of the rolled stats (or, simpler, that *not all* stats equal base).

Skeleton:

```go
func TestAssetCreate_UseAverageStats_WritesAtlasDataDefaultsVerbatim(t *testing.T) {
	// Arrange: an equip template (e.g., 1002357) with EA{Strength:10, Slots:7, ...}
	// Act: Create with UseAverageStats=true.
	// Assert: persisted asset has Strength==10, Dexterity==Dex, ..., Slots==7.
	t.Skip("fill in using existing inventory test harness")
}

func TestAssetCreate_UseAverageStats_FalseRetainsVariance(t *testing.T) {
	t.Skip("fill in using existing inventory test harness")
}
```

If the package lacks a clean unit-test harness today, use the integration-style harness already used by `compartment/processor_test.go` and assert through that path.

- [ ] **Step 2: Run to confirm fail**

```bash
cd services/atlas-inventory/atlas.com/inventory && go test ./asset/... -count=1
```

- [ ] **Step 3: Add the bypass branch**

In the equip case of `Create` (the one rewritten in Task 10), wrap the variance-rolled `b.SetXxx(getRandomStat(...))` chain with an `if opts.UseAverageStats { ... } else { ... }`:

```go
case inventory.TypeValueEquip:
	ea, err := p.statProcessor.GetById(templateId)
	if err != nil {
		if errors.Is(err, requests.ErrNotFound) {
			p.l.WithError(err).Errorf("Equipment template [%d] not present in atlas-data; seed data is likely missing.", templateId)
		} else {
			p.l.WithError(err).Errorf("Unable to get equipment stats for item [%d].", templateId)
		}
		return err
	}
	if opts.UseAverageStats {
		b.SetStrength(ea.Strength()).
			SetDexterity(ea.Dexterity()).
			SetIntelligence(ea.Intelligence()).
			SetLuck(ea.Luck()).
			SetHp(ea.Hp()).
			SetMp(ea.Mp()).
			SetWeaponAttack(ea.WeaponAttack()).
			SetMagicAttack(ea.MagicAttack()).
			SetWeaponDefense(ea.WeaponDefense()).
			SetMagicDefense(ea.MagicDefense()).
			SetAccuracy(ea.Accuracy()).
			SetAvoidability(ea.Avoidability()).
			SetHands(ea.Hands()).
			SetSpeed(ea.Speed()).
			SetJump(ea.Jump()).
			SetSlots(ea.Slots())
	} else {
		b.SetStrength(getRandomStat(ea.Strength(), 5)).
			SetDexterity(getRandomStat(ea.Dexterity(), 5)).
			SetIntelligence(getRandomStat(ea.Intelligence(), 5)).
			SetLuck(getRandomStat(ea.Luck(), 5)).
			SetHp(getRandomStat(ea.Hp(), 10)).
			SetMp(getRandomStat(ea.Mp(), 10)).
			SetWeaponAttack(getRandomStat(ea.WeaponAttack(), 5)).
			SetMagicAttack(getRandomStat(ea.MagicAttack(), 5)).
			SetWeaponDefense(getRandomStat(ea.WeaponDefense(), 10)).
			SetMagicDefense(getRandomStat(ea.MagicDefense(), 10)).
			SetAccuracy(getRandomStat(ea.Accuracy(), 5)).
			SetAvoidability(getRandomStat(ea.Avoidability(), 5)).
			SetHands(getRandomStat(ea.Hands(), 5)).
			SetSpeed(getRandomStat(ea.Speed(), 5)).
			SetJump(getRandomStat(ea.Jump(), 5)).
			SetSlots(ea.Slots())
	}
```

- [ ] **Step 4: Run tests**

```bash
cd services/atlas-inventory/atlas.com/inventory && go test ./... -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-inventory/atlas.com/inventory/asset/
git commit -m "feat(atlas-inventory): bypass equip stat variance when UseAverageStats=true

Per task-037 design FR-16/D-1. Atlas-data defaults written verbatim;
variance branch unchanged. Required for deterministic preset equipment."
```

---

### Task 12: Wire `UseAverageStats` from the create-asset consumer to `Create`

**Files:**
- Modify: `services/atlas-inventory/atlas.com/inventory/kafka/message/compartment/kafka.go:99` (`CreateAssetCommandBody`)
- Modify: `services/atlas-inventory/atlas.com/inventory/kafka/consumer/compartment/consumer.go:207-213` (`handleCreateAssetCommand`)
- Modify: `services/atlas-inventory/atlas.com/inventory/compartment/processor.go` (`CreateAsset`/`CreateAssetAndEmit` to forward the flag)
- Test: extend the closest consumer/processor test that already exercises this path.

- [ ] **Step 1: Add `UseAverageStats` to atlas-inventory's `CreateAssetCommandBody`**

(Mirror of the orchestrator-side change.)

```go
type CreateAssetCommandBody struct {
	TemplateId      uint32    `json:"templateId"`
	Quantity        uint32    `json:"quantity"`
	Expiration      time.Time `json:"expiration"`
	OwnerId         uint32    `json:"ownerId"`
	Flag            uint16    `json:"flag"`
	Rechargeable    uint64    `json:"rechargeable"`
	UseAverageStats bool      `json:"useAverageStats,omitempty"`
}
```

- [ ] **Step 2: Forward from consumer**

In `kafka/consumer/compartment/consumer.go:207-213`, locate the `CreateAssetAndEmit` (or equivalent) call. Add `c.Body.UseAverageStats` to the chain:

```bash
grep -n "CreateAssetAndEmit\|CreateAssetAndLock\|CreateAsset\b" services/atlas-inventory/atlas.com/inventory/kafka/consumer/compartment/consumer.go
```

Forward the flag through `compartment.Processor.CreateAssetAndEmit` (whatever it's called). The `CreateOptions` already carry the field; thread it from the consumer through `CreateAsset` → `CreateAssetAndLock` → the asset.Create call.

- [ ] **Step 3: Write a failing consumer test**

A unit/integration test that:
1. Sends a `CreateAssetCommand` with `UseAverageStats: true`.
2. Asserts the persisted equip asset's stats match atlas-data defaults verbatim.
3. Sends the same command with the flag false; asserts variance applied (or at minimum, stats vary across invocations).

Use the existing harness in `compartment/processor_test.go` or a similar consumer integration test; follow the in-memory bus + db pattern already present.

- [ ] **Step 4: Run tests**

```bash
cd services/atlas-inventory/atlas.com/inventory && go test ./... -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-inventory/atlas.com/inventory/
git commit -m "feat(atlas-inventory): forward UseAverageStats from consumer to asset.Create

Per task-037 design §4.7. The flag now travels end-to-end through the
saga path: orchestrator command body → consumer → CreateAssetAndEmit →
asset.Create CreateOptions."
```

---

## Phase 6 — atlas-configurations: Presets storage + validation + seed

### Task 13: Define preset RestModel (template + tenant scopes)

**Files:**
- Create: `services/atlas-configurations/atlas.com/configurations/templates/characters/preset/rest.go`
- Create: `services/atlas-configurations/atlas.com/configurations/tenants/characters/preset/rest.go`
- Test: `services/atlas-configurations/atlas.com/configurations/tenants/characters/preset/rest_test.go`

Both files have identical content; the package mirrors how the existing `template` package mirrors itself across scopes.

- [ ] **Step 1: Write the rest model**

`tenants/characters/preset/rest.go` (and templates mirror):

```go
package preset

type StatBlock struct {
	Str uint16 `json:"str"`
	Dex uint16 `json:"dex"`
	Int uint16 `json:"int"`
	Luk uint16 `json:"luk"`
	Hp  uint16 `json:"hp"`
	Mp  uint16 `json:"mp"`
}

type EquipmentEntry struct {
	TemplateId      uint32 `json:"templateId"`
	UseAverageStats bool   `json:"useAverageStats"`
}

type InventoryEntry struct {
	TemplateId uint32 `json:"templateId"`
	Quantity   uint32 `json:"quantity"`
}

type SkillEntry struct {
	SkillId uint32 `json:"skillId"`
	Level   uint8  `json:"level"`
}

type Attributes struct {
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Tags        []string         `json:"tags"`
	JobId       uint32           `json:"jobId"`
	Gender      byte             `json:"gender"`
	Face        uint32           `json:"face"`
	Hair        uint32           `json:"hair"`
	HairColor   uint32           `json:"hairColor"`
	SkinColor   byte             `json:"skinColor"`
	MapId       uint32           `json:"mapId"`
	Level       byte             `json:"level"`
	Meso        uint32           `json:"meso"`
	Gm          int              `json:"gm"`
	Stats       StatBlock        `json:"stats"`
	DefaultName string           `json:"defaultName"`
	Equipment   []EquipmentEntry `json:"equipment"`
	Inventory   []InventoryEntry `json:"inventory"`
	Skills      []SkillEntry     `json:"skills"`
}

type RestModel struct {
	Id         string     `json:"id"`
	Attributes Attributes `json:"attributes"`
}
```

- [ ] **Step 2: Write a round-trip test**

```go
package preset

import (
	"encoding/json"
	"testing"
)

func TestRestModel_RoundTrip(t *testing.T) {
	in := RestModel{
		Id: "5e1c0b6e-8a52-4c33-9f4a-6c2c1bc9c1d7",
		Attributes: Attributes{
			Name:      "Hero — 4th job",
			JobId:     112,
			Gender:    0,
			Level:     200,
			Stats:     StatBlock{Str: 999, Hp: 30000, Mp: 6000},
			Equipment: []EquipmentEntry{{TemplateId: 1002357, UseAverageStats: true}},
			Inventory: []InventoryEntry{{TemplateId: 2000000, Quantity: 200}},
			Skills:    []SkillEntry{{SkillId: 1121008, Level: 30}},
		},
	}
	bs, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out RestModel
	if err := json.Unmarshal(bs, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Attributes.Equipment[0].TemplateId != 1002357 {
		t.Fatalf("equipment templateId did not survive round trip")
	}
	if !out.Attributes.Equipment[0].UseAverageStats {
		t.Fatalf("UseAverageStats flag lost")
	}
}
```

- [ ] **Step 3: Run the new package tests**

```bash
cd services/atlas-configurations/atlas.com/configurations && go test ./tenants/characters/preset/... ./templates/characters/preset/... -count=1
```

Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-configurations/atlas.com/configurations/tenants/characters/preset/ services/atlas-configurations/atlas.com/configurations/templates/characters/preset/
git commit -m "feat(atlas-configurations): preset RestModel for both scopes

Per task-037 design §4.1 and data-model.md. Mirror packages under
templates/characters/preset and tenants/characters/preset to match the
existing template/tenant split."
```

---

### Task 14: Add `Presets` field to `characters.RestModel` (both scopes)

**Files:**
- Modify: `services/atlas-configurations/atlas.com/configurations/tenants/characters/rest.go`
- Modify: `services/atlas-configurations/atlas.com/configurations/templates/characters/rest.go`
- Test: a focused decode test under `tenants/` that confirms both old (no `presets`) and new payloads decode correctly.

- [ ] **Step 1: Write failing decode test**

Add `services/atlas-configurations/atlas.com/configurations/tenants/characters/rest_test.go`:

```go
package characters

import (
	"encoding/json"
	"testing"
)

func TestRestModel_BackwardsCompat_NoPresets(t *testing.T) {
	in := []byte(`{"templates":[]}`)
	var out RestModel
	if err := json.Unmarshal(in, &out); err != nil {
		t.Fatalf("unmarshal legacy: %v", err)
	}
	if out.Presets == nil {
		// Acceptable for the field to be nil; the orchestrator coerces nil → []
		// at read-time. Just confirm no panic.
	}
}

func TestRestModel_PresetsRoundTrip(t *testing.T) {
	in := []byte(`{"templates":[],"presets":[{"id":"abc","attributes":{"name":"x","jobId":112}}]}`)
	var out RestModel
	if err := json.Unmarshal(in, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(out.Presets) != 1 || out.Presets[0].Id != "abc" || out.Presets[0].Attributes.JobId != 112 {
		t.Fatalf("preset did not decode: %+v", out)
	}
}
```

- [ ] **Step 2: Run to confirm fail**

```bash
cd services/atlas-configurations/atlas.com/configurations && go test ./tenants/characters/... -count=1
```

Expected: FAIL — `unknown field Presets`.

- [ ] **Step 3: Add the field on both rest models**

`tenants/characters/rest.go`:

```go
package characters

import (
	"atlas-configurations/templates/characters/template"
	"atlas-configurations/tenants/characters/preset"
)

type RestModel struct {
	Templates []template.RestModel `json:"templates"`
	Presets   []preset.RestModel   `json:"presets"`
}
```

`templates/characters/rest.go`: same shape, importing `atlas-configurations/templates/characters/preset` instead.

- [ ] **Step 4: Run tests**

```bash
cd services/atlas-configurations/atlas.com/configurations && go test ./... -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-configurations/atlas.com/configurations/tenants/characters/ services/atlas-configurations/atlas.com/configurations/templates/characters/
git commit -m "feat(atlas-configurations): add Presets to characters rest model

Per task-037 design §4.1. Sibling array next to Templates at both
template and tenant scope. Legacy documents without 'presets' decode
to nil and are treated as empty by readers."
```

---

### Task 15: atlas-data client package for preset validation

**Files:**
- Create: `services/atlas-configurations/atlas.com/configurations/data/skill_requests.go`
- Create: `services/atlas-configurations/atlas.com/configurations/data/item_requests.go`
- Create: `services/atlas-configurations/atlas.com/configurations/data/mock/processor.go`
- Test: `services/atlas-configurations/atlas.com/configurations/data/requests_test.go`

The validator needs to look up item templates (for slot conflict + existence) and skill templates (for `MaxLevel` ceiling and existence). Build a small client.

- [ ] **Step 1: Inspect how atlas-character-factory currently reaches atlas-data**

```bash
ls services/atlas-character-factory/atlas.com/character-factory/configuration/
grep -rn "atlas-data\|/data/" services/atlas-character-factory/atlas.com/character-factory/ | head
```

Reuse the `atlas-rest` client wiring already used by atlas-character-factory's `configuration/requests.go` as a template. No need to invent a new pattern.

- [ ] **Step 2: Define an interface and implementation**

`services/atlas-configurations/atlas.com/configurations/data/skill_requests.go`:

```go
package data

import "context"

type SkillInfo struct {
	Id       uint32
	Name     string
	MaxLevel uint8
}

type ItemInfo struct {
	Id   uint32
	Slot int16 // signed equip slot (negative); 0 for non-equip items
}

type Client interface {
	GetSkillsByIds(ctx context.Context, ids []uint32) ([]SkillInfo, error)
	GetItemById(ctx context.Context, id uint32) (ItemInfo, error)
}
```

The implementation hits the atlas-data endpoints via the shared `atlas-rest` client. Mirror the construction pattern in `configuration/registry.go` if there is one.

- [ ] **Step 3: Implement and add a fake/mock for tests**

Create `services/atlas-configurations/atlas.com/configurations/data/mock/processor.go` with a fake client driven by maps:

```go
package mock

import (
	"atlas-configurations/data"
	"context"
	"errors"
)

type FakeClient struct {
	Skills map[uint32]data.SkillInfo
	Items  map[uint32]data.ItemInfo
}

func (f *FakeClient) GetSkillsByIds(_ context.Context, ids []uint32) ([]data.SkillInfo, error) {
	out := make([]data.SkillInfo, 0, len(ids))
	for _, id := range ids {
		if sk, ok := f.Skills[id]; ok {
			out = append(out, sk)
		}
	}
	return out, nil
}

func (f *FakeClient) GetItemById(_ context.Context, id uint32) (data.ItemInfo, error) {
	if it, ok := f.Items[id]; ok {
		return it, nil
	}
	return data.ItemInfo{}, errors.New("not found")
}
```

- [ ] **Step 4: Tests**

A small `requests_test.go` exercising the real client against an httptest server is ideal but optional; the validator tests in Task 16 will exercise the interface contract via the FakeClient. Skip a roundtrip test if it's not idiomatic in this repo (look at how `configuration/requests.go` is tested in atlas-character-factory; replicate or skip).

- [ ] **Step 5: Run tests**

```bash
cd services/atlas-configurations/atlas.com/configurations && go test ./data/... -count=1
```

- [ ] **Step 6: Commit**

```bash
git add services/atlas-configurations/atlas.com/configurations/data/
git commit -m "feat(atlas-configurations): add atlas-data client for preset validation

Per task-037 design §4.1 / D-8. Exposes skill MaxLevel + item slot
lookups; mock package powers the validator unit tests."
```

---

### Task 16: Preset validator

**Files:**
- Create: `services/atlas-configurations/atlas.com/configurations/tenants/characters/preset/validator.go`
- Create: `services/atlas-configurations/atlas.com/configurations/templates/characters/preset/validator.go` (mirror — re-export or duplicate)
- Test: `services/atlas-configurations/atlas.com/configurations/tenants/characters/preset/validator_test.go`

The validator runs at PATCH time (before persistence). Returns a slice of `ValidationError{ PresetId, Field, Message }`; the caller maps to JSON:API errors.

- [ ] **Step 1: Define the validator interface**

`tenants/characters/preset/validator.go`:

```go
package preset

import (
	"atlas-configurations/data"
	"context"
	"strconv"

	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	"github.com/google/uuid"
)

type ValidationError struct {
	PresetId string `json:"presetId"`
	Field    string `json:"field"`
	Message  string `json:"message"`
}

type Validator struct {
	client data.Client
}

func NewValidator(client data.Client) *Validator {
	return &Validator{client: client}
}

// Validate inspects the supplied preset list and returns a slice of validation
// errors. The list is mutated in place to assign UUIDs to entries with empty
// Id fields per R-1 (called *before* validation so error rows always carry an id).
func (v *Validator) Validate(ctx context.Context, presets []RestModel) ([]RestModel, []ValidationError) {
	for i := range presets {
		if presets[i].Id == "" {
			presets[i].Id = uuid.New().String()
		}
	}

	var out []ValidationError
	for _, p := range presets {
		out = append(out, v.validateOne(ctx, p)...)
	}
	return presets, out
}

func (v *Validator) validateOne(ctx context.Context, p RestModel) []ValidationError {
	var errs []ValidationError
	add := func(field, msg string) {
		errs = append(errs, ValidationError{PresetId: p.Id, Field: field, Message: msg})
	}

	// Identity
	if l := len(p.Attributes.Name); l < 1 || l > 64 {
		add("name", "must be 1..64 characters")
	}
	if len(p.Attributes.Description) > 512 {
		add("description", "must be ≤512 characters")
	}

	// Job / character
	if !job.IsValid(job.Id(p.Attributes.JobId)) {
		add("jobId", "unknown job id")
	}
	if p.Attributes.Gender > 1 {
		add("gender", "must be 0 or 1")
	}
	if p.Attributes.Level < 1 || p.Attributes.Level > 250 {
		add("level", "must be in [1,250]")
	}

	// Equipment: existence + slot uniqueness
	seenSlots := map[int16]string{}
	for i, eq := range p.Attributes.Equipment {
		info, err := v.client.GetItemById(ctx, eq.TemplateId)
		if err != nil {
			add(fieldPath("equipment", i, "templateId"), "item not found in atlas-data")
			continue
		}
		if info.Slot >= 0 {
			add(fieldPath("equipment", i, "templateId"), "item is not equippable")
			continue
		}
		if other, exists := seenSlots[info.Slot]; exists {
			add(fieldPath("equipment", i, "templateId"), "equipment slot collision with "+other)
		} else {
			seenSlots[info.Slot] = strconv.FormatUint(uint64(eq.TemplateId), 10)
		}
	}

	// Inventory
	for i, it := range p.Attributes.Inventory {
		if _, err := v.client.GetItemById(ctx, it.TemplateId); err != nil {
			add(fieldPath("inventory", i, "templateId"), "item not found in atlas-data")
		}
		if it.Quantity < 1 {
			add(fieldPath("inventory", i, "quantity"), "must be ≥1")
		}
	}

	// Skills (batched; single round-trip for the whole preset's skill list)
	if len(p.Attributes.Skills) > 0 {
		ids := make([]uint32, 0, len(p.Attributes.Skills))
		for _, s := range p.Attributes.Skills {
			ids = append(ids, s.SkillId)
		}
		got, err := v.client.GetSkillsByIds(ctx, ids)
		if err != nil {
			add("skills", "atlas-data lookup failed: "+err.Error())
		} else {
			byId := map[uint32]data.SkillInfo{}
			for _, sk := range got {
				byId[sk.Id] = sk
			}
			for i, s := range p.Attributes.Skills {
				sk, ok := byId[s.SkillId]
				if !ok {
					add(fieldPath("skills", i, "skillId"), "skill not found in atlas-data")
					continue
				}
				if s.Level < 1 || s.Level > sk.MaxLevel {
					add(fieldPath("skills", i, "level"), "must be in [1,maxLevel]")
				}
			}
		}
	}

	return errs
}

func fieldPath(arr string, i int, name string) string {
	return arr + "[" + strconv.Itoa(i) + "]." + name
}
```

- [ ] **Step 2: Mirror at template scope**

`templates/characters/preset/validator.go` re-exports the same logic, importing the local `preset` package's types. If duplication feels heavy, factor a shared helper into `services/atlas-configurations/atlas.com/configurations/internal/presetvalidate/` and call it from both. Mirroring is the simplest path that matches the existing template/tenant split.

- [ ] **Step 3: Write table-driven tests**

`tenants/characters/preset/validator_test.go` covering at minimum:

- All-good preset → no errors.
- Missing Id → server fills in via `uuid.New()`, downstream errors carry that id.
- Invalid jobId → field "jobId".
- Gender 5 → field "gender".
- Level 0 or 251 → field "level".
- Equipment with non-existent templateId → field "equipment[0].templateId".
- Equipment slot collision (two helmets) → second entry errors with collision message.
- Inventory quantity 0 → field "inventory[0].quantity".
- Skill with level > maxLevel → field "skills[0].level".
- Skill with id not in atlas-data → field "skills[0].skillId".

Each row uses `mock.FakeClient` with a hand-built skill+item map.

- [ ] **Step 4: Run tests**

```bash
cd services/atlas-configurations/atlas.com/configurations && go test ./tenants/characters/preset/... ./templates/characters/preset/... -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-configurations/atlas.com/configurations/tenants/characters/preset/validator.go services/atlas-configurations/atlas.com/configurations/templates/characters/preset/validator.go services/atlas-configurations/atlas.com/configurations/tenants/characters/preset/validator_test.go services/atlas-configurations/atlas.com/configurations/templates/characters/preset/validator_test.go
git commit -m "feat(atlas-configurations): preset validator (12 rules)

Per task-037 design D-8 / data-model.md §Validation summary. Generates
missing UUIDs (R-1), uses preset id in error paths (R-3), batches skill
lookups via the new atlas-data ids= filter."
```

---

### Task 17: Wire validator into PATCH handlers

**Files:**
- Modify: `services/atlas-configurations/atlas.com/configurations/tenants/resource.go:64-76` (`handleUpdateConfigurationTenant`) and `tenants/processor.go:65-77` (`UpdateById`)
- Modify: `services/atlas-configurations/atlas.com/configurations/templates/resource.go` (mirror)
- Test: extend `tenants/processor_test.go` to assert that an invalid preset returns an error and that an empty `id` gets a generated UUID assigned.

- [ ] **Step 1: Inject the validator into the processor**

In `tenants/processor.go`, change `NewProcessor` to accept (or lazily construct) a `*preset.Validator`. The existing `Processor` struct can grow a `validator *preset.Validator` field; the http resource constructs it (e.g., in `InitResource`) using the data client.

```go
func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB, validator *preset.Validator) *Processor {
	return &Processor{l: l, ctx: ctx, db: db, validator: validator}
}
```

Adjust call sites in `resource.go` accordingly.

- [ ] **Step 2: Validate in `UpdateById`**

Before marshal:

```go
func (p *Processor) UpdateById(tenantId uuid.UUID, input RestModel) error {
	if p.validator != nil {
		assigned, errs := p.validator.Validate(p.ctx, input.Characters.Presets)
		input.Characters.Presets = assigned
		if len(errs) > 0 {
			return validationFailureError(errs)
		}
	}
	// ... existing marshal + persist ...
}
```

`validationFailureError` is a typed error wrapping `[]preset.ValidationError`. The HTTP handler renders these as JSON:API `errors[]` with `meta.path = "presets[<presetId>].<field>"`.

- [ ] **Step 3: Render the error in the handler**

In `handleUpdateConfigurationTenant`, replace the simple `500 InternalServerError` path with:

```go
err := NewProcessor(...).UpdateById(tenantId, input)
if err != nil {
	var ve *validationFailureError
	if errors.As(err, &ve) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"errors": ve.AsJSONAPIErrors(),
		})
		return
	}
	d.Logger().WithError(err).Errorf("Unable to update configuration tenant.")
	w.WriteHeader(http.StatusInternalServerError)
	return
}
```

`AsJSONAPIErrors()` returns one entry per validation error:

```go
type jsonapiError struct {
	Status string         `json:"status"`
	Title  string         `json:"title"`
	Detail string         `json:"detail"`
	Meta   map[string]any `json:"meta"`
}

func (e *validationFailureError) AsJSONAPIErrors() []jsonapiError {
	out := make([]jsonapiError, 0, len(e.errors))
	for _, ve := range e.errors {
		out = append(out, jsonapiError{
			Status: "400",
			Title:  "validation failed",
			Detail: ve.Message,
			Meta:   map[string]any{"path": "presets[" + ve.PresetId + "]." + ve.Field},
		})
	}
	return out
}
```

- [ ] **Step 4: Mirror at template scope**

Same pattern in `templates/resource.go` and `templates/processor.go`.

- [ ] **Step 5: Tests**

Extend (or add) `tenants/processor_test.go`:

```go
func TestUpdateById_AssignsMissingPresetIdAndPersists(t *testing.T) { ... }
func TestUpdateById_ReturnsValidationErrorForInvalidPreset(t *testing.T) { ... }
```

Use the fake atlas-data client.

Then add a handler-level test that sends a malformed preset PATCH and asserts the JSON:API errors body shape.

- [ ] **Step 6: Run all atlas-configurations tests**

```bash
cd services/atlas-configurations/atlas.com/configurations && go test ./... -count=1
```

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-configurations/atlas.com/configurations/tenants/ services/atlas-configurations/atlas.com/configurations/templates/
git commit -m "feat(atlas-configurations): wire preset validator into PATCH handlers

Per task-037 design §4.1. Generates UUIDs for empty preset ids (R-1),
returns JSON:API errors with meta.path rooted at presets[<id>].<field>
on validation failure (R-3)."
```

---

### Task 18: Seed canonical 4th-job presets into `template_gms_83_1.json`; empty arrays elsewhere

**Files:**
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_83_1.json` (insert canonical preset list)
- Modify: every other `services/atlas-configurations/seed-data/templates/template_*.json` (add `characters.presets: []`)
- Optional: Update the seeder unit/integration test to verify the new field is loaded.

- [ ] **Step 1: List the seed files**

```bash
ls services/atlas-configurations/seed-data/templates/
```

- [ ] **Step 2: Add `presets: []` to every non-GMS-v83 template file**

For each `template_*.json` *other than* `template_gms_83_1.json`, locate the `"characters": { "templates": [...] }` block and append:

```jsonc
"characters": {
  "templates": [ /* unchanged */ ],
  "presets": []
}
```

- [ ] **Step 3: Add the canonical 4th-job preset list to `template_gms_83_1.json`**

Replace the `"characters"` block with `templates` unchanged plus a `presets` list. The list contains one preset per explorer 4th-job class (FR-27): Hero, Paladin, Dark Knight, F/P ArchMage, I/L ArchMage, Bishop, Bowmaster, Marksman, Night Lord, Shadower, Buccaneer, Corsair.

For each preset, fill in:
- `id`: a fresh UUID (generate with `uuidgen`).
- `attributes.name`: `"<Class> — 4th job"`
- `attributes.description`: short flavour (e.g., `"Full 4th-job Hero with Combat Orders, Stance, etc."`).
- `attributes.tags`: `["4th-job", "<branch>", "explorer"]` (branch ∈ warrior/magician/bowman/thief/pirate).
- `attributes.jobId`: 112 (Hero), 122 (Paladin), 132 (Dark Knight), 212 (F/P ArchMage), 222 (I/L ArchMage), 232 (Bishop), 312 (Bowmaster), 322 (Marksman), 412 (Night Lord), 422 (Shadower), 512 (Buccaneer), 522 (Corsair).
- `attributes.gender`: 0.
- `attributes.face`: a stock face id from atlas-data (e.g., 20000).
- `attributes.hair`: a stock hair id (e.g., 30030); `hairColor`: 0.
- `attributes.skinColor`: 0.
- `attributes.mapId`: a sensible 4th-job map (e.g., 240000000 for Leafre).
- `attributes.level`: 200.
- `attributes.meso`: 100000000.
- `attributes.gm`: 0.
- `attributes.stats`: `{ str/dex/int/luk: numbers appropriate for the class; hp: 30000; mp: 6000 }`.
- `attributes.defaultName`: `"Admin<Class>"` (e.g., `"AdminHero"`).
- `attributes.equipment`: a small canonical 4th-job loadout per class — typically 4-6 entries (helm, top OR overall, gloves, shoes, weapon, [shield]). Use atlas-data-known templateIds; `useAverageStats: true` on each.
- `attributes.inventory`: starter consumables (Red Potions x200, Mana Elixirs x200, etc).
- `attributes.skills`: skill IDs of the class's signature 4th-job skills + ultimate (e.g., for Hero: 1121008 Hero's Will, 1121011 Enrage, 1121002 Achilles, 1121006 Stance, 1121000 Brandish, 1121010 Combat Orders). Levels at master.

If the implementer cannot determine skill/item ids confidently for every class on day one, ship the list with whichever of the 12 classes are confidently filled in and leave the others as `[]` entries to be added in a follow-up content commit (see follow-up TODO #6 in design). PRD acceptance §10 #1 still passes as long as at least the seeded classes appear.

- [ ] **Step 4: Run the seeder test (if it exists)**

```bash
cd services/atlas-configurations/atlas.com/configurations && go test ./seeder/... -count=1
```

If the seeder loads and round-trips templates, pass implies the JSON parses cleanly. If a parse error surfaces, fix the offending file.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-configurations/seed-data/templates/
git commit -m "feat(atlas-configurations): seed 4th-job presets in GMS v83 template

Per task-037 design D-10 / R-4. Other region/version template files
gain an empty 'presets: []' for forward compatibility; their canonical
preset lists are owned by their content engineers."
```

---

## Phase 7 — atlas-character-factory: preset endpoint and name-validity passthrough

### Task 19: New external clients in atlas-character-factory

**Files:**
- Create: `services/atlas-character-factory/atlas.com/character-factory/data/skill_requests.go`
- Create: `services/atlas-character-factory/atlas.com/character-factory/data/item_requests.go`
- Create: `services/atlas-character-factory/atlas.com/character-factory/configuration/preset_requests.go` (alongside existing `requests.go`)
- Create: `services/atlas-character-factory/atlas.com/character-factory/character/name_validity_requests.go`

Mirror the existing `configuration/requests.go` style (`atlas-rest` client, JSON:API decode).

- [ ] **Step 1: skill_requests.go**

```go
package data

import "context"

type SkillInfo struct {
	Id       uint32
	MaxLevel uint8
}

type SkillClient interface {
	GetByIds(ctx context.Context, ids []uint32) ([]SkillInfo, error)
}
```

Implementation hits `GET /data/skills?ids=1,2,3` (the new endpoint from Task 4) and decodes JSON:API.

- [ ] **Step 2: item_requests.go**

```go
package data

type ItemInfo struct {
	Id   uint32
	Slot int16 // negative for equip; 0 for non-equip
}

type ItemClient interface {
	GetById(ctx context.Context, id uint32) (ItemInfo, error)
}
```

Implementation hits the existing `GET /data/items/:id` (already used elsewhere by the factory; reuse the same builder).

- [ ] **Step 3: configuration/preset_requests.go**

```go
package configuration

import (
	"context"
	"github.com/google/uuid"
)

type Preset struct {
	Id         string
	Attributes PresetAttributes
}

type PresetAttributes struct { /* mirror the rest model from atlas-configurations */ }

type PresetClient interface {
	GetById(ctx context.Context, presetId uuid.UUID) (Preset, error)
}
```

Implementation hits `GET /api/configurations/tenants/:id` (the active-tenant endpoint, already used by the factory) and filters the returned `characters.presets` array client-side. Returns a typed `ErrNotFound` when the id is missing.

- [ ] **Step 4: character/name_validity_requests.go**

```go
package character

import "context"

type NameValidityResult struct {
	Valid  bool   `json:"valid"`
	Reason string `json:"reason,omitempty"`
	Detail string `json:"detail,omitempty"`
}

type NameValidityClient interface {
	Check(ctx context.Context, name string, worldId byte) (NameValidityResult, error)
}
```

Implementation hits `GET /characters/name-validity` from Task 7.

- [ ] **Step 5: Tests**

A single `requests_test.go` per file using `httptest.NewServer` covers both happy and error paths.

- [ ] **Step 6: Build the factory**

```bash
cd services/atlas-character-factory/atlas.com/character-factory && go build ./...
```

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-character-factory/atlas.com/character-factory/
git commit -m "feat(atlas-character-factory): clients for skill ids, item, preset, name-validity

Scaffolding for the upcoming POST /factory/characters/from-preset path.
Each client follows the existing atlas-rest client convention."
```

---

### Task 20: `CreateFromPreset` processor method + `buildPresetCharacterCreationSaga`

**Files:**
- Modify: `services/atlas-character-factory/atlas.com/character-factory/factory/processor.go`
- Test: `services/atlas-character-factory/atlas.com/character-factory/factory/processor_test.go`

- [ ] **Step 1: Define the input rest model**

Create `services/atlas-character-factory/atlas.com/character-factory/factory/preset_rest.go`:

```go
package factory

type PresetCreateRestModel struct {
	PresetId  string `json:"presetId"`
	AccountId uint32 `json:"accountId"`
	WorldId   byte   `json:"worldId"`
	Name      string `json:"name"`
}

func (r PresetCreateRestModel) GetName() string { return "preset-create" }
func (r PresetCreateRestModel) GetID() string   { return "" }
func (r *PresetCreateRestModel) SetID(string) error { return nil }
```

- [ ] **Step 2: Add `CreateFromPreset` to the processor**

In `factory/processor.go`, append a new method that:

1. Resolves the preset via `configuration.PresetClient.GetById`. On not-found, return a typed error mapped to 404 by the handler.
2. Calls `character.NameValidityClient.Check`. On invalid, return a typed error carrying `Reason` ("regex"/"length") for 400 or "duplicate" for 409.
3. Validates equipment / inventory / skill ids against atlas-data:
   - For each equipment entry, `data.ItemClient.GetById(eq.TemplateId)` — must exist and Slot < 0.
   - Equipment slots must be unique across the preset.
   - For each inventory entry, `data.ItemClient.GetById(inv.TemplateId)` — must exist.
   - Batch-fetch all skill `MaxLevel`s via `data.SkillClient.GetByIds(ids)`. On any miss → 502 (per design — reject the PRD's "fall back" because that would silently produce wrong characters). On atlas-data unreachable → 502.
4. Builds the saga via a new private helper `buildPresetCharacterCreationSaga`.
5. Emits via the existing `saga.NewProcessor(p.l, ctx).Create(...)`.
6. Returns the transactionId.

```go
func (p *ProcessorImpl) CreateFromPreset(ctx context.Context, in PresetCreateRestModel) (string, error) {
	presetId, err := uuid.Parse(in.PresetId)
	if err != nil {
		return "", errInvalidPresetId
	}

	preset, err := p.presetClient.GetById(ctx, presetId)
	if err != nil {
		return "", errPresetNotFound
	}

	nv, err := p.nameClient.Check(ctx, in.Name, in.WorldId)
	if err != nil {
		return "", err
	}
	if !nv.Valid {
		return "", &nameInvalidError{Reason: nv.Reason, Detail: nv.Detail}
	}

	if err := p.validatePresetIds(ctx, preset); err != nil {
		return "", err
	}

	skillsById, err := p.fetchSkillMaxLevels(ctx, preset.Attributes.Skills)
	if err != nil {
		return "", errAtlasDataUnreachable
	}

	transactionId := uuid.New()
	sag := buildPresetCharacterCreationSaga(transactionId, in, preset, skillsById)
	if err := saga.NewProcessor(p.l, ctx).Create(sag); err != nil {
		return "", err
	}
	return transactionId.String(), nil
}
```

(`validatePresetIds`, `fetchSkillMaxLevels`, the typed errors, and the registry wiring for the four clients are auxiliary — implement alongside.)

- [ ] **Step 3: Implement `buildPresetCharacterCreationSaga`**

Add next to `buildCharacterCreationSaga`:

```go
func buildPresetCharacterCreationSaga(transactionId uuid.UUID, in PresetCreateRestModel, preset configuration.Preset, skillsById map[uint32]data.SkillInfo) saga.Saga {
	a := preset.Attributes
	builder := saga.NewBuilder().
		SetTransactionId(transactionId).
		SetSagaType(saga.CharacterCreation).
		SetInitiatedBy(fmt.Sprintf("account_%d", in.AccountId)).
		SetTimeout(10 * time.Second)

	builder.AddStep("create_character", saga.Pending, saga.CreateCharacter, saga.CharacterCreatePayload{
		AccountId:    in.AccountId,
		WorldId:      world.Id(in.WorldId),
		Name:         in.Name,
		Gender:       a.Gender,
		Level:        a.Level,
		Strength:     a.Stats.Str,
		Dexterity:    a.Stats.Dex,
		Intelligence: a.Stats.Int,
		Luck:         a.Stats.Luk,
		JobId:        job.Id(a.JobId),
		Hp:           a.Stats.Hp,
		Mp:           a.Stats.Mp,
		Face:         a.Face,
		Hair:         a.Hair + a.HairColor,
		Skin:         a.SkinColor,
		Top:          0,
		Bottom:       0,
		Shoes:        0,
		Weapon:       0,
		MapId:        _map.Id(a.MapId),
		Gm:           a.Gm,
		Meso:         a.Meso,
	})

	for i, inv := range a.Inventory {
		builder.AddStep(fmt.Sprintf("award_asset_%d", i), saga.Pending, saga.AwardAsset, saga.AwardItemActionPayload{
			CharacterId: 0,
			Item:        saga.ItemPayload{TemplateId: inv.TemplateId, Quantity: inv.Quantity},
		})
	}

	for i, eq := range a.Equipment {
		builder.AddStep(fmt.Sprintf("create_and_equip_asset_%d", i), saga.Pending, saga.CreateAndEquipAsset, saga.CreateAndEquipAssetPayload{
			CharacterId:     0,
			Item:            saga.ItemPayload{TemplateId: eq.TemplateId, Quantity: 1},
			UseAverageStats: eq.UseAverageStats,
		})
	}

	for i, sk := range a.Skills {
		master := uint32(skillsById[sk.SkillId].MaxLevel)
		builder.AddStep(fmt.Sprintf("create_skill_%d", i), saga.Pending, saga.CreateSkill, saga.CreateSkillPayload{
			CharacterId: 0,
			SkillId:     sk.SkillId,
			Level:       uint32(sk.Level),
			MasterLevel: master,
			Expiration:  time.Time{},
		})
	}

	return builder.Build()
}
```

(Adjust types — `saga.CreateSkillPayload.MasterLevel` may be a different concrete type; match the existing definition.)

- [ ] **Step 4: Tests for `CreateFromPreset`**

In `factory/processor_test.go`:

- Preset not found → returns errPresetNotFound (404 mapped by handler).
- Name invalid → returns nameInvalidError with the right reason.
- Skill id not in atlas-data → returns 502 (errAtlasDataUnreachable or distinct typed error — be explicit).
- Equipment templateId not in atlas-data → 400.
- Equipment slot collision detected at apply time even if save-time validator missed it → 400.
- Happy path: produces a `saga.Saga` with N+M+K+1 steps in the documented order; payload fields match preset attributes.

Use `httptest.Server`-backed client mocks per the existing factory test style, or hand-rolled in-memory implementations of the four client interfaces.

- [ ] **Step 5: Run factory tests**

```bash
cd services/atlas-character-factory/atlas.com/character-factory && go test ./factory/... -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-character-factory/atlas.com/character-factory/factory/
git commit -m "feat(atlas-character-factory): CreateFromPreset processor method

Per task-037 design §4.4. Resolves preset, validates name, re-validates
ids against atlas-data (R-3 path), batches MaxLevel lookups, builds and
emits the existing CharacterCreation saga with UseAverageStats and
Gm/Meso plumbed through."
```

---

### Task 21: Register `POST /factory/characters/from-preset` and `GET /factory/characters/name-validity`

**Files:**
- Modify: `services/atlas-character-factory/atlas.com/character-factory/factory/resource.go`
- Test: extend `factory/resource_test.go` (or wherever the existing factory routes are HTTP-tested) to assert 202 for happy path, 404 for missing preset, 400/409 for invalid/duplicate names, 502 for atlas-data fail.

- [ ] **Step 1: Register the routes**

Add inside the existing `InitResource` registration:

```go
r.HandleFunc("/factory/characters/from-preset", rest.RegisterInputHandler[PresetCreateRestModel](l)(si)("create_from_preset", handleCreateFromPreset(...))).Methods(http.MethodPost)
r.HandleFunc("/factory/characters/name-validity", rest.RegisterHandler(l)(si)("get_name_validity", handleNameValidity(...))).Methods(http.MethodGet)
```

- [ ] **Step 2: Implement the handlers**

```go
func handleCreateFromPreset(p factory.Processor) rest.InputHandler[PresetCreateRestModel] {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext, in PresetCreateRestModel) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			tx, err := p.CreateFromPreset(d.Context(), in)
			if err != nil {
				switch {
				case errors.Is(err, factory.ErrPresetNotFound):
					w.WriteHeader(http.StatusNotFound)
				case isInvalidName(err):
					w.WriteHeader(http.StatusBadRequest)
					_ = json.NewEncoder(w).Encode(jsonAPIErrorFromName(err))
				case errors.Is(err, factory.ErrNameDuplicate):
					w.WriteHeader(http.StatusConflict)
				case errors.Is(err, factory.ErrAtlasDataUnreachable):
					w.WriteHeader(http.StatusBadGateway)
				case errors.Is(err, factory.ErrPresetValidation):
					w.WriteHeader(http.StatusBadRequest)
					_ = json.NewEncoder(w).Encode(jsonAPIErrorsFromValidation(err))
				default:
					d.Logger().WithError(err).Error("CreateFromPreset failed")
					w.WriteHeader(http.StatusInternalServerError)
				}
				return
			}
			w.WriteHeader(http.StatusAccepted)
			_ = json.NewEncoder(w).Encode(map[string]string{"transactionId": tx})
		}
	}
}

func handleNameValidity(client character.NameValidityClient) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			q := r.URL.Query()
			name := q.Get("name")
			widRaw := q.Get("worldId")
			if name == "" || widRaw == "" {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			wid64, err := strconv.ParseUint(widRaw, 10, 8)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			res, err := client.Check(d.Context(), name, byte(wid64))
			if err != nil {
				w.WriteHeader(http.StatusBadGateway)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(res)
		}
	}
}
```

- [ ] **Step 3: Tests**

Run a small `httptest`-style suite against the registered router. Use the same harness the existing factory tests use.

- [ ] **Step 4: Update README + routes.conf**

`services/atlas-character-factory/README.md`'s REST table gains the two new endpoints; check if `routes.conf` (in the gateway) needs an entry — search for existing factory entries and mirror.

- [ ] **Step 5: Run tests**

```bash
cd services/atlas-character-factory/atlas.com/character-factory && go test ./... -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-character-factory/
git commit -m "feat(atlas-character-factory): expose POST /factory/characters/from-preset and GET /factory/characters/name-validity

Per task-037 design §4.4. Maps typed processor errors to 400/404/409/502
JSON:API responses; name-validity is a thin passthrough to atlas-character."
```

---

## Phase 8 — atlas-ui

### Task 22: `factory.service.ts`

**Files:**
- Create: `services/atlas-ui/src/services/api/factory.service.ts`
- Optional: Modify `services/atlas-ui/src/services/api/accounts.service.ts` to add `getAll({name})` filter if absent.

- [ ] **Step 1: Inspect the existing service module pattern**

```bash
ls services/atlas-ui/src/services/api/
grep -n "buildAuthHeaders\|tenantHeaders\|class .*Service\|export const" services/atlas-ui/src/services/api/accounts.service.ts | head
```

- [ ] **Step 2: Implement the service**

```ts
// services/atlas-ui/src/services/api/factory.service.ts
import { apiClient } from "@/lib/api/client"; // adjust to actual import per repo

export interface CreateFromPresetPayload {
  presetId: string;
  accountId: number;
  worldId: number;
  name: string;
}

export interface CreateFromPresetResponse {
  transactionId: string;
}

export interface NameValidityResponse {
  valid: boolean;
  reason?: "regex" | "length" | "blocked" | "duplicate";
  detail?: string;
}

export const factoryService = {
  async createFromPreset(payload: CreateFromPresetPayload): Promise<CreateFromPresetResponse> {
    const res = await apiClient.post("/factory/characters/from-preset", payload);
    return res.data;
  },

  async checkNameValidity(name: string, worldId: number): Promise<NameValidityResponse> {
    const res = await apiClient.get("/factory/characters/name-validity", {
      params: { name, worldId },
    });
    return res.data;
  },
};
```

(Match the actual import path / client wrapper used by sibling services. Don't invent a new HTTP client.)

- [ ] **Step 3: Tests**

Add `services/atlas-ui/src/services/api/__tests__/factory.service.test.ts` mocking the api client per the existing test pattern.

- [ ] **Step 4: Run vitest**

```bash
cd services/atlas-ui && npm test -- factory.service
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-ui/src/services/api/
git commit -m "feat(atlas-ui): factory.service for createFromPreset + name-validity"
```

---

### Task 23: React Query hooks (`useCreateCharacterFromPreset`, `useNameValidity`, `useAccountByName`)

**Files:**
- Create: `services/atlas-ui/src/lib/hooks/api/useCharacterFromPresetMutation.ts`
- Create: `services/atlas-ui/src/lib/hooks/api/useNameValidity.ts`
- Create: `services/atlas-ui/src/lib/hooks/api/useAccountByName.ts`
- Test: corresponding `__tests__/*.test.tsx`

- [ ] **Step 1: useCharacterFromPresetMutation**

```ts
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { factoryService, type CreateFromPresetPayload } from "@/services/api/factory.service";
import { characterKeys } from "@/lib/hooks/api/useCharacters"; // adjust

export function useCreateCharacterFromPreset() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (payload: CreateFromPresetPayload) => factoryService.createFromPreset(payload),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: characterKeys.lists() });
    },
  });
}
```

- [ ] **Step 2: useNameValidity**

```ts
import { useQuery } from "@tanstack/react-query";
import { factoryService } from "@/services/api/factory.service";
import { useDebouncedValue } from "@/lib/hooks/useDebouncedValue"; // existing

export function useNameValidity(name: string, worldId: number, opts?: { enabled?: boolean; debounceMs?: number }) {
  const debounced = useDebouncedValue(name, opts?.debounceMs ?? 300);
  return useQuery({
    queryKey: ["name-validity", worldId, debounced],
    queryFn: () => factoryService.checkNameValidity(debounced, worldId),
    enabled: (opts?.enabled ?? true) && debounced.length >= 3,
    staleTime: 0,
  });
}
```

If `useDebouncedValue` doesn't exist yet, add a tiny implementation under `lib/hooks/`.

- [ ] **Step 3: useAccountByName**

```ts
import { useEffect, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { accountsService } from "@/services/api/accounts.service";

export interface UseAccountByNameOptions {
  pollUntilFound?: boolean;
  timeoutMs?: number;
  intervalMs?: number;
}

export function useAccountByName(name: string, opts: UseAccountByNameOptions = {}) {
  const interval = opts.intervalMs ?? 1000;
  const timeout = opts.timeoutMs ?? 30000;
  const [timedOut, setTimedOut] = useState(false);

  const query = useQuery({
    queryKey: ["account", "by-name", name],
    queryFn: () => accountsService.getAll({ name }),
    enabled: !!name && !timedOut,
    refetchInterval: ({ state }) => {
      const found = Array.isArray(state.data) && state.data.length > 0;
      if (found || timedOut || !opts.pollUntilFound) return false;
      return interval;
    },
  });

  useEffect(() => {
    if (!opts.pollUntilFound) return;
    const t = setTimeout(() => setTimedOut(true), timeout);
    return () => clearTimeout(t);
  }, [opts.pollUntilFound, timeout, name]);

  return { ...query, timedOut };
}
```

- [ ] **Step 4: Tests**

Each hook gets a focused test in `__tests__/`. Mock the service module; use `@testing-library/react`'s `renderHook`. Cover:

- `useNameValidity`: returns valid for happy path; debounce works; disabled below 3 chars.
- `useAccountByName`: success (found on first poll), success-after-poll, timeout. Use Jest fake timers.
- `useCharacterFromPresetMutation`: invalidates character list on success.

- [ ] **Step 5: Run tests**

```bash
cd services/atlas-ui && npm test -- useCharacterFromPreset useNameValidity useAccountByName
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-ui/src/lib/hooks/api/
git commit -m "feat(atlas-ui): hooks for createFromPreset, name-validity, account-by-name poll

Per task-037 design D-11/D-13. useNameValidity debounces by 300ms by
default; useAccountByName polls every 1s up to 30s with a watchdog
timeout; useCreateCharacterFromPreset invalidates the character list."
```

---

### Task 24: TemplatesCharacterPresetsPage + TenantsCharacterPresetsPage

**Files:**
- Create: `services/atlas-ui/src/pages/templates-character-presets-form.tsx`
- Create: `services/atlas-ui/src/pages/tenants-character-presets-form.tsx`
- Reference precedent: `services/atlas-ui/src/pages/templates-character-templates-form.tsx`, `tenants-character-templates-form.tsx`

The two pages share most of their structure; factor a `<CharacterPresetsForm scope="template"|"tenant" />` component if duplication is heavy. Otherwise mirror the templates form layout with the per-card structure from `ux-flow.md` §C:

- Top bar: `<Button>Add preset</Button>`
- Per preset: an accordion item with sections **Identity / Character / Stats / Equipment / Inventory / Skills**
- Bottom: `<Button type="submit">Save</Button>`

For state management:

- Use `useTemplate(id)` (or `useTenantConfiguration(id)`) to fetch.
- Use react-hook-form with a Zod schema mirroring the `Attributes` rest model. Top-level `presets` is a field array (`useFieldArray`).
- On submit, spread the existing `characters` document with the edited presets:

```ts
updateTemplate.mutate({
  id,
  updates: {
    characters: { ...template.attributes.characters, presets },
  },
});
```

For server-side validation errors (JSON:API `errors[]`), map `meta.path = "presets[<id>].<field>"` to react-hook-form's per-field error.

- [ ] **Step 1: Define the Zod schema**

`services/atlas-ui/src/pages/character-presets-schema.ts` (shared by both pages):

```ts
import { z } from "zod";

const equipmentEntry = z.object({
  templateId: z.number().int().nonnegative(),
  useAverageStats: z.boolean().default(true),
});

const inventoryEntry = z.object({
  templateId: z.number().int().nonnegative(),
  quantity: z.number().int().min(1),
});

const skillEntry = z.object({
  skillId: z.number().int().nonnegative(),
  level: z.number().int().min(1),
});

const stats = z.object({
  str: z.number().int().nonnegative(),
  dex: z.number().int().nonnegative(),
  int: z.number().int().nonnegative(),
  luk: z.number().int().nonnegative(),
  hp:  z.number().int().nonnegative(),
  mp:  z.number().int().nonnegative(),
});

export const presetSchema = z.object({
  id: z.string().optional(),
  attributes: z.object({
    name: z.string().min(1).max(64),
    description: z.string().max(512).optional().default(""),
    tags: z.array(z.string()).default([]),
    jobId: z.number().int().nonnegative(),
    gender: z.union([z.literal(0), z.literal(1)]),
    face: z.number().int().nonnegative(),
    hair: z.number().int().nonnegative(),
    hairColor: z.number().int().nonnegative(),
    skinColor: z.number().int().nonnegative(),
    mapId: z.number().int().nonnegative(),
    level: z.number().int().min(1).max(250),
    meso: z.number().int().nonnegative(),
    gm: z.number().int().min(0),
    stats,
    defaultName: z.string().optional().default(""),
    equipment: z.array(equipmentEntry).default([]),
    inventory: z.array(inventoryEntry).default([]),
    skills: z.array(skillEntry).default([]),
  }),
});

export const presetsFormSchema = z.object({
  presets: z.array(presetSchema),
});

export type PresetsFormValues = z.infer<typeof presetsFormSchema>;
```

- [ ] **Step 2: Implement the templates form**

`services/atlas-ui/src/pages/templates-character-presets-form.tsx` — mirror the structure of `templates-character-templates-form.tsx` (header, sidebar, breadcrumbs, form wrapper) but render the preset card list. Use `useFieldArray` for `presets`. Each preset card uses additional `useFieldArray`s for `equipment`, `inventory`, `skills`.

For free-text uint32 inputs, use shadcn `<Input type="number">` bound to react-hook-form, with `lookup` links pointing at `/items/:id` and `/skills/:id` (existing detail pages).

- [ ] **Step 3: Implement the tenants form**

`services/atlas-ui/src/pages/tenants-character-presets-form.tsx` — same as above against `useTenantConfiguration` / `useUpdateTenantConfiguration`.

If you factored a shared component, both pages are thin wrappers selecting the right hook.

- [ ] **Step 4: Tests**

`services/atlas-ui/src/pages/__tests__/templates-character-presets-form.test.tsx` exercises:

- Loads existing presets (mocked via MSW or service mock).
- Add preset → empty preset card appears.
- Edit field → state updates.
- Submit → mutation called with the merged `characters` document.
- Server-side validation error for `presets[<id>].name` shows up under that preset's name input.

- [ ] **Step 5: Run tests**

```bash
cd services/atlas-ui && npm test -- character-presets-form
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-ui/src/pages/
git commit -m "feat(atlas-ui): preset catalog editor pages (template + tenant scope)

Per task-037 design §4.8 / ux-flow.md §C. Reuses the existing
characters-document hooks; renders the new 'presets' sub-array next to
the existing templates editor."
```

---

### Task 25: Wire routes, breadcrumbs, sidebar entries

**Files:**
- Modify: `services/atlas-ui/src/App.tsx`
- Modify: `services/atlas-ui/src/lib/breadcrumbs/routes.ts`
- Modify: `services/atlas-ui/src/pages/TemplateDetailPage.tsx` (sidebar entry)
- Modify: `services/atlas-ui/src/pages/TenantDetailPage.tsx` (sidebar entry)

- [ ] **Step 1: Add routes**

In `App.tsx`:

```tsx
<Route path="/templates/:id/character/presets" element={<TemplatesCharacterPresetsPage />} />
<Route path="/tenants/:id/character/presets" element={<TenantsCharacterPresetsPage />} />
```

- [ ] **Step 2: Breadcrumbs**

In `lib/breadcrumbs/routes.ts`, add:

```ts
{ path: "/templates/:id/character/presets", label: "Character Presets" },
{ path: "/tenants/:id/character/presets",   label: "Character Presets" },
```

(Match existing entry shape — likely a registry pattern.)

- [ ] **Step 3: Sidebar entries**

In `TemplateDetailPage.tsx` and `TenantDetailPage.tsx`, find the existing **Character Templates** sidebar/nav entry and add a sibling **Character Presets** entry pointing at the new route. Use the same icon family.

- [ ] **Step 4: Manual smoke**

```bash
cd services/atlas-ui && npm run dev
```

Open both pages; confirm:
- Sidebar entries appear.
- Breadcrumbs render with `Templates / <name> / Character Presets`.
- Empty state renders when `characters.presets` is empty/undefined.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-ui/src/App.tsx services/atlas-ui/src/lib/breadcrumbs/ services/atlas-ui/src/pages/
git commit -m "feat(atlas-ui): routes, breadcrumbs, sidebar for character presets pages"
```

---

### Task 26: ApplyPresetDialog on AccountDetailPage

**Files:**
- Create: `services/atlas-ui/src/components/features/characters/ApplyPresetDialog.tsx`
- Modify: `services/atlas-ui/src/pages/AccountDetailPage.tsx` (header action)
- Test: `services/atlas-ui/src/components/features/characters/__tests__/ApplyPresetDialog.test.tsx`

- [ ] **Step 1: Implement the dialog**

```tsx
// ApplyPresetDialog.tsx — sketch
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { useTenantConfiguration } from "@/lib/hooks/api/useTenants";
import { useNameValidity } from "@/lib/hooks/api/useNameValidity";
import { useCreateCharacterFromPreset } from "@/lib/hooks/api/useCharacterFromPresetMutation";

const schema = z.object({
  presetId: z.string().min(1),
  worldId: z.number().int().min(0),
  name: z.string().min(3).max(12),
});

export function ApplyPresetDialog({ accountId, tenantId, open, onOpenChange }: Props) {
  const { data: tenant } = useTenantConfiguration(tenantId);
  const presets = tenant?.attributes?.characters?.presets ?? [];

  const form = useForm({ resolver: zodResolver(schema), defaultValues: { worldId: 0 } });
  const name = form.watch("name");
  const worldId = form.watch("worldId");
  const validity = useNameValidity(name, worldId, { enabled: !!name && name.length >= 3 });

  const mutation = useCreateCharacterFromPreset();

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      {/* combobox for preset, world picker, name input bound to form, submit button */}
    </Dialog>
  );
}
```

- [ ] **Step 2: Add header button + dialog state to AccountDetailPage**

In `AccountDetailPage.tsx`, add an **Add character from preset** button to the page header. The button is conditional on `presets.length > 0` (FR-23):

```tsx
const { data: tenant } = useTenantConfiguration(activeTenant.id);
const hasPresets = (tenant?.attributes?.characters?.presets ?? []).length > 0;

{hasPresets && (
  <Button onClick={() => setApplyOpen(true)}>Add character from preset</Button>
)}
<ApplyPresetDialog
  accountId={account.id}
  tenantId={activeTenant.id}
  open={applyOpen}
  onOpenChange={setApplyOpen}
/>
```

- [ ] **Step 3: Test the dialog**

`__tests__/ApplyPresetDialog.test.tsx` covering:

- Renders preset list from `useTenantConfiguration`.
- Submit disabled until name validity is true.
- On submit success: dialog closes, toast fires, character list invalidated.
- 400 / 409 from mutation: dialog stays open with inline error.

- [ ] **Step 4: Run tests**

```bash
cd services/atlas-ui && npm test -- ApplyPresetDialog
```

Expected: PASS.

- [ ] **Step 5: Manual sanity**

```bash
cd services/atlas-ui && npm run dev
```

Open an `AccountDetailPage` for a tenant with at least one preset. Click the button; confirm the dialog opens, name validity checks debounce, submit fires.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-ui/src/components/features/characters/ services/atlas-ui/src/pages/AccountDetailPage.tsx
git commit -m "feat(atlas-ui): ApplyPresetDialog on AccountDetailPage

Per task-037 design §4.8 / ux-flow.md §A. Visible only when active
tenant has at least one preset (FR-23)."
```

---

### Task 27: AdminBootstrapWizard on AccountsPage

**Files:**
- Create: `services/atlas-ui/src/components/features/accounts/AdminBootstrapWizard.tsx`
- Modify: `services/atlas-ui/src/pages/AccountsPage.tsx` (header action)
- Test: `services/atlas-ui/src/components/features/accounts/__tests__/AdminBootstrapWizard.test.tsx`

The wizard is the largest UI surface in the task. Per design D-12 it uses `useReducer` for top-level state and react-hook-form for per-step sub-forms.

- [ ] **Step 1: Implement reducer + state types**

Sketch:

```tsx
type RowStatus = "pending" | "applying" | "success" | "failed";
type Row = { presetId: string; name: string; validity: NameValidityResponse | null; applyStatus: RowStatus; error?: string };
type WizardState = {
  step: 1 | 2 | 3 | 4;
  account: { name: string; password: string };
  worldId: number;
  tagFilter: string[];
  rows: Record<string, Row>;
  accountId?: number;
  error?: string;
};

type WizardAction =
  | { type: "SET_ACCOUNT"; account: { name: string; password: string } }
  | { type: "SET_WORLD"; worldId: number }
  | { type: "SET_TAG_FILTER"; tags: string[] }
  | { type: "TOGGLE_PRESET"; presetId: string }
  | { type: "SET_NAME"; presetId: string; name: string }
  | { type: "SET_VALIDITY"; presetId: string; validity: NameValidityResponse }
  | { type: "SET_ROW_STATUS"; presetId: string; status: RowStatus; error?: string }
  | { type: "ACCOUNT_CREATED"; accountId: number }
  | { type: "GOTO"; step: 1 | 2 | 3 | 4 }
  | { type: "RESET" };

function wizardReducer(state: WizardState, action: WizardAction): WizardState { ... }
```

- [ ] **Step 2: Implement the four steps**

Step 1 (`AccountCredentialsStep.tsx`): react-hook-form with name+password.
Step 2 (`WorldAndTagStep.tsx`): world picker + tag chip selector + live preview list of presets matching filter.
Step 3 (`NameOverridesStep.tsx`): table of selected presets with name input bound to `useNameValidity`. Wizard-local duplicate detection: walk `rows`, mark rows whose name appears more than once as `validity: { valid: false, reason: 'duplicate', detail: 'Duplicate within wizard' }`.
Step 4 (`ApplyStep.tsx`): the orchestration logic.

Step 4 logic:

```tsx
async function runApply() {
  // 1. Create account
  const created = await accountsService.create({ name: state.account.name, password: state.account.password });
  // 2. Wait for account to materialize
  const accountByName = await waitForAccountByName(state.account.name, { timeoutMs: 30000, intervalMs: 1000 });
  if (!accountByName) {
    dispatch({ type: 'GOTO', step: 4 });
    setError('Account did not materialize within 30s');
    return;
  }
  dispatch({ type: 'ACCOUNT_CREATED', accountId: accountByName.id });

  // 3. Apply presets sequentially
  for (const row of selectedRows) {
    dispatch({ type: 'SET_ROW_STATUS', presetId: row.presetId, status: 'applying' });
    try {
      await mutation.mutateAsync({ presetId: row.presetId, accountId: accountByName.id, worldId: state.worldId, name: row.name });
      dispatch({ type: 'SET_ROW_STATUS', presetId: row.presetId, status: 'success' });
    } catch (e) {
      dispatch({ type: 'SET_ROW_STATUS', presetId: row.presetId, status: 'failed', error: extractErrorDetail(e) });
    }
  }
}
```

`waitForAccountByName` is implemented inline using `useAccountByName` with `pollUntilFound:true,timeoutMs:30000`. Promise-ify the hook by wrapping with `useEffect` + state — or just call `accountsService.getAll({name})` directly in a manual poll loop here, which is simpler in step 4 since we're outside the React render cycle.

After the loop, surface a **Done** button which closes the dialog and navigates to `/accounts/<accountId>`.

- [ ] **Step 3: Add header button to AccountsPage**

```tsx
<Button onClick={() => setBootstrapOpen(true)}>Bootstrap Admin Account</Button>
<AdminBootstrapWizard open={bootstrapOpen} onOpenChange={setBootstrapOpen} />
```

- [ ] **Step 4: Tests**

`__tests__/AdminBootstrapWizard.test.tsx`:

- Reducer tests (pure, no DOM): each action type produces the expected next state.
- Component tests:
  - Step gating: Next disabled until invariant holds.
  - Tag filter narrows the preview list.
  - Wizard-internal duplicate name detection flags both rows.
  - Step 4 happy path: runs sequentially, marks rows success.
  - Step 4 partial failure: account created but second preset fails; first row success, second failed, retry available.
  - Account materialization timeout: surfaces error + retry button without invalidating prior state.

Use Jest fake timers for the poll-timeout test.

- [ ] **Step 5: Run tests**

```bash
cd services/atlas-ui && npm test -- AdminBootstrapWizard
```

Expected: PASS.

- [ ] **Step 6: Browser smoke**

```bash
cd services/atlas-ui && npm run dev
```

Drive the wizard end-to-end against a dev backend. Confirm at minimum: account is created, materializes, at least one preset applies and the resulting character appears under the new account.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-ui/src/components/features/accounts/ services/atlas-ui/src/pages/AccountsPage.tsx
git commit -m "feat(atlas-ui): AdminBootstrapWizard on AccountsPage

Per task-037 design §4.8 / ux-flow.md §B. Four-step wizard with
useReducer state, sequential preset application, per-row retry, and
account-materialization watchdog (30s, 1s poll)."
```

---

## Phase 9 — Integration & follow-ups

### Task 28: Saga compensation integration test for preset failure

**Files:**
- Test: a new integration test under `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/` (mirror existing compensation integration tests; search `compensation` in the saga test files).

The PRD's acceptance criterion §10 #6 is "forcing a preset-apply failure mid-saga produces compensation that rolls back inventory/equipment/skills". With `create_and_equip_asset`, `award_asset`, and `create_skill` already covered by existing compensation tests, this task adds a single end-to-end case that builds a preset-shaped saga (multiple inventory items + multiple equipment items + multiple skills) and forces a failure on the second equipment step. Assert: previously-created inventory items and the first equipment are removed; the character is deleted; saga ends in compensated state.

- [ ] **Step 1: Locate the existing compensation integration test pattern**

```bash
grep -rn "compensation\b\|Compensated\b" services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/ | head
```

- [ ] **Step 2: Add the test**

Mirror the closest existing compensation integration test; modify to build a preset-shaped saga from `factory.buildPresetCharacterCreationSaga` (or a hand-built saga with the same step sequence) and inject a forced failure in the second `create_and_equip_asset` step.

- [ ] **Step 3: Run**

```bash
cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go test ./saga/... -count=1 -run Preset
```

Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/
git commit -m "test(atlas-saga-orchestrator): preset-driven compensation integration

Per task-037 PRD acceptance §10 #6. End-to-end test that a failing
equipment step rolls back the entire preset application."
```

---

### Task 29: TODO entries for follow-ups

**Files:**
- Modify: `docs/TODO.md`

Per design §7, append the following entries (use the existing TODO.md format — find it via `Glob "docs/TODO.md"`):

1. **Migrate atlas-npc-shops to deterministic stats.** `services/atlas-npc-shops/atlas.com/npc/compartment/producer.go:13-19` — set `UseAverageStats=true` after task-037 ships.
2. **Migrate atlas-character-factory player-creation flow to deterministic stats.** `factory/processor.go:138-211` `buildCharacterCreationSaga` — set `UseAverageStats=true` for the four equip steps.
3. **Saga `transactionId` polling for the Admin Bootstrap wizard.** Drive per-row UI flips success → failure when compensation fires.
4. **Reusable `<ItemPicker>` / `<SkillPicker>` components.** Replace the free-text uint32 inputs in the preset editor.
5. **Cygnus / Aran / Resistance / Legend 4th-job presets.** Content task: extend `template_gms_83_1.json` with non-explorer 4th-job presets.

- [ ] **Step 1: Append entries**

Match the existing TODO.md format/section grouping.

- [ ] **Step 2: Commit**

```bash
git add docs/TODO.md
git commit -m "docs: log follow-ups from task-037 character presets

5 follow-ups: shops + player-creation determinism migrations,
transactionId polling, picker components, non-explorer 4th-job content."
```

---

## Self-Review

After completing the tasks above, the implementer should re-skim the spec and verify spec coverage. The plan covers:

- FR-1..3: Tasks 13–17 (storage), Task 17 (UUID generation per R-1).
- FR-4: Task 13 (rest model fields).
- FR-5: Task 16 (slot uniqueness rule).
- FR-6..14: Tasks 19–21 (factory endpoint, saga build), Task 28 (compensation test).
- FR-15..17: Tasks 1, 9, 11, 12 (UseAverageStats).
- FR-18..20: Task 27 (Admin Bootstrap wizard) + Task 7 (name-validity helper) + Task 21 (factory passthrough).
- FR-21..23: Task 26 (ApplyPresetDialog) + Task 23 (hooks).
- FR-24..26: Tasks 24, 25 (pages + nav).
- FR-27..28: Task 18 (seeded GMS v83 list, empty arrays elsewhere; existing seeder behaviour preserves non-empty rows).
- §2.3 corrections (Gm/Meso plumb): Tasks 2, 5, 6, 8.
- §2.2 (skill MaxLevel): Task 3.
- D-5 (skill ids filter): Task 4.
- D-6 (name authority): Task 7.
- D-3 (asset.Create options struct): Task 10.
- D-10 / R-4 (seed only GMS v83): Task 18.
- §10 #6 (compensation): Task 28.
- §7 follow-ups: Task 29.

If a spec requirement is missing from this list, add a task. If a placeholder (`TODO`, `TBD`, `Similar to Task N`) appears in any code block above, replace with concrete content.
