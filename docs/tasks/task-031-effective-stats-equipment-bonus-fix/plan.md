# atlas-effective-stats: Equipment Bonus Hydration Fix — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `atlas-effective-stats` populate `bonuses[]` with equipped-item stats by hydrating the `included` JSON:API assets returned from atlas-inventory.

**Architecture:** Add the api2go-fork hydration hooks (`GetReferences`, `GetReferencedIDs`, `GetReferencedStructs`, `SetToOneReferenceID`, `SetReferencedStructs`) to `CompartmentRestModel` so that `jsonapi.Unmarshal` walks the `included` array after creating the ID-only stubs and replaces each stub with a fully-hydrated `AssetRestModel`. The downstream `fetchEquipmentBonuses` loop already handles the populated structs correctly — it simply gets accurate input.

**Tech Stack:** Go, `github.com/jtumidanski/api2go/jsonapi` (api2go fork), `net/http/httptest`, `github.com/sirupsen/logrus/hooks/test`, `github.com/Chronicle20/atlas/libs/atlas-tenant`.

---

## File Structure

| Path | Action | Responsibility |
|---|---|---|
| `services/atlas-effective-stats/atlas.com/effective-stats/external/inventory/rest.go` | **Modify** | Add 5 new methods on `CompartmentRestModel`; import `github.com/jtumidanski/api2go/jsonapi`. |
| `services/atlas-effective-stats/atlas.com/effective-stats/external/inventory/rest_test.go` | **Create** | Unit test that hand-rolls a JSON:API document with one compartment + one asset in `included`, calls `jsonapi.Unmarshal`, asserts every numeric attribute on the asset is populated. |
| `services/atlas-effective-stats/atlas.com/effective-stats/character/initializer_test.go` | **Create** | Integration test that stands up an `httptest.NewServer`, points `INVENTORY_SERVICE_URL` at it, calls `fetchEquipmentBonuses`, asserts the returned `[]stat.Bonus` includes `equipment:42` entries for HP/MP. |

No other source files change. The reference implementation for the hydration methods is `services/atlas-npc-shops/atlas.com/npc/compartment/rest.go:37-100`.

---

## Task 1: Add JSON:API hydration hooks to `CompartmentRestModel`

**Files:**
- Test: `services/atlas-effective-stats/atlas.com/effective-stats/external/inventory/rest_test.go` (create)
- Modify: `services/atlas-effective-stats/atlas.com/effective-stats/external/inventory/rest.go` (add 5 methods + 1 import)

**Reference:** `services/atlas-npc-shops/atlas.com/npc/compartment/rest.go:37-100` shows the same pattern working today against `asset.BaseRestModel`. Mirror it, substituting `AssetRestModel`.

- [ ] **Step 1: Write the failing unit test**

Create `services/atlas-effective-stats/atlas.com/effective-stats/external/inventory/rest_test.go` with the following content:

```go
package inventory

import (
	"testing"

	"github.com/jtumidanski/api2go/jsonapi"
)

// TestCompartmentRestModel_HydratesIncludedAssets feeds a JSON:API document
// with one compartment whose single asset is denormalised in `included`,
// and asserts that every numeric attribute on the asset is populated after
// jsonapi.Unmarshal. This is the regression net for the bug where stub
// assets had Slot==0/Hp==0/Mp==0 and were silently skipped by IsEquipped().
func TestCompartmentRestModel_HydratesIncludedAssets(t *testing.T) {
	doc := []byte(`{
      "data": {
        "type": "compartments",
        "id": "00000000-0000-0000-0000-000000000001",
        "attributes": {"type": 1, "capacity": 24},
        "relationships": {
          "assets": { "data": [{"type": "assets", "id": "42"}] }
        }
      },
      "included": [{
        "type": "assets",
        "id": "42",
        "attributes": {
          "slot": -49,
          "templateId": 1142107,
          "expiration": "0001-01-01T00:00:00Z",
          "ownerId": 0,
          "strength": 1,
          "dexterity": 2,
          "intelligence": 3,
          "luck": 4,
          "hp": 47,
          "mp": 50,
          "weaponAttack": 5,
          "magicAttack": 6,
          "weaponDefense": 7,
          "magicDefense": 8,
          "accuracy": 9,
          "avoidability": 10,
          "hands": 11,
          "speed": 12,
          "jump": 13
        }
      }]
    }`)

	var c CompartmentRestModel
	if err := jsonapi.Unmarshal(doc, &c); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if len(c.Assets) != 1 {
		t.Fatalf("len(Assets) = %d, want 1", len(c.Assets))
	}
	a := c.Assets[0]
	if a.Id != 42 {
		t.Errorf("Id = %d, want 42", a.Id)
	}
	if a.Slot != -49 {
		t.Errorf("Slot = %d, want -49", a.Slot)
	}
	if a.TemplateId != 1142107 {
		t.Errorf("TemplateId = %d, want 1142107", a.TemplateId)
	}
	if a.Hp != 47 {
		t.Errorf("Hp = %d, want 47", a.Hp)
	}
	if a.Mp != 50 {
		t.Errorf("Mp = %d, want 50", a.Mp)
	}
	if a.Strength != 1 {
		t.Errorf("Strength = %d, want 1", a.Strength)
	}
	if a.Dexterity != 2 {
		t.Errorf("Dexterity = %d, want 2", a.Dexterity)
	}
	if a.Intelligence != 3 {
		t.Errorf("Intelligence = %d, want 3", a.Intelligence)
	}
	if a.Luck != 4 {
		t.Errorf("Luck = %d, want 4", a.Luck)
	}
	if a.WeaponAttack != 5 {
		t.Errorf("WeaponAttack = %d, want 5", a.WeaponAttack)
	}
	if a.MagicAttack != 6 {
		t.Errorf("MagicAttack = %d, want 6", a.MagicAttack)
	}
	if a.WeaponDefense != 7 {
		t.Errorf("WeaponDefense = %d, want 7", a.WeaponDefense)
	}
	if a.MagicDefense != 8 {
		t.Errorf("MagicDefense = %d, want 8", a.MagicDefense)
	}
	if a.Accuracy != 9 {
		t.Errorf("Accuracy = %d, want 9", a.Accuracy)
	}
	if a.Avoidability != 10 {
		t.Errorf("Avoidability = %d, want 10", a.Avoidability)
	}
	if a.Hands != 11 {
		t.Errorf("Hands = %d, want 11", a.Hands)
	}
	if a.Speed != 12 {
		t.Errorf("Speed = %d, want 12", a.Speed)
	}
	if a.Jump != 13 {
		t.Errorf("Jump = %d, want 13", a.Jump)
	}

	// Sanity: IsEquipped() and GetEquipableData() must now succeed,
	// confirming the downstream gate in fetchEquipmentBonuses will
	// no longer starve.
	if !a.IsEquipped() {
		t.Errorf("IsEquipped() = false, want true (Slot=%d)", a.Slot)
	}
	eq, ok := a.GetEquipableData()
	if !ok {
		t.Fatalf("GetEquipableData() ok = false, want true")
	}
	if eq.Hp != 47 {
		t.Errorf("eq.Hp = %d, want 47", eq.Hp)
	}
	if eq.Mp != 50 {
		t.Errorf("eq.Mp = %d, want 50", eq.Mp)
	}
}
```

- [ ] **Step 2: Run the test and confirm it fails**

Run:

```bash
cd services/atlas-effective-stats/atlas.com/effective-stats && \
  go test ./external/inventory/ -run TestCompartmentRestModel_HydratesIncludedAssets -v
```

Expected: FAIL. The test should report `Slot = 0, want -49` (and similar zeros for `Hp`, `Mp`, `TemplateId`, etc.) because `SetToManyReferenceIDs` only stores the asset ID — every other field defaults to zero.

If the test fails for a different reason (compilation error, panic), stop and investigate before proceeding.

- [ ] **Step 3: Add the import line and the five hydration methods to `rest.go`**

Modify `services/atlas-effective-stats/atlas.com/effective-stats/external/inventory/rest.go`.

3a. Update the import block at the top of the file (currently `import ( "strconv"; "time" )`) to:

```go
import (
	"strconv"
	"time"

	"github.com/jtumidanski/api2go/jsonapi"
)
```

3b. Insert the five new methods **immediately after** the existing `SetToManyReferenceIDs` method (which currently ends at line 40) and **before** the `// AssetRestModel represents…` comment that starts the asset block. Add this block verbatim:

```go
// GetReferences declares the to-many relationship list so api2go can wire
// `included` resources back to this compartment.
func (r CompartmentRestModel) GetReferences() []jsonapi.Reference {
	return []jsonapi.Reference{
		{
			Type: "assets",
			Name: "assets",
		},
	}
}

// GetReferencedIDs lists the IDs api2go should expect in `included` for the
// declared references. Required by the api2go fork interface even though this
// service only consumes (not produces) compartment documents.
func (r CompartmentRestModel) GetReferencedIDs() []jsonapi.ReferenceID {
	var result []jsonapi.ReferenceID
	for _, v := range r.Assets {
		result = append(result, jsonapi.ReferenceID{
			ID:   v.GetID(),
			Type: v.GetName(),
			Name: v.GetName(),
		})
	}
	return result
}

// GetReferencedStructs returns the embedded asset structs for outbound
// marshalling. Implemented to mirror the in-repo template
// (atlas-npc-shops/.../compartment/rest.go) and avoid surprises if this type
// ever needs to round-trip.
func (r CompartmentRestModel) GetReferencedStructs() []jsonapi.MarshalIdentifier {
	var result []jsonapi.MarshalIdentifier
	for key := range r.Assets {
		result = append(result, r.Assets[key])
	}
	return result
}

// SetToOneReferenceID is a no-op satisfier — CompartmentRestModel has no
// to-one relationships, but the api2go fork interface requires the method.
func (r *CompartmentRestModel) SetToOneReferenceID(_, _ string) error {
	return nil
}

// SetReferencedStructs walks the stub assets that SetToManyReferenceIDs
// populated and replaces each with a fully-hydrated copy from `included`,
// using jsonapi.ProcessIncludeData to populate flat attribute fields
// (slot, hp, mp, strength, …) from the included resource's attributes object.
//
// Without this method, every AssetRestModel in r.Assets retains its zero-value
// Slot/Hp/Mp, causing the downstream IsEquipped() (Slot < 0) gate in
// character/initializer.go::fetchEquipmentBonuses to skip every equipped
// asset and silently drop all equipment bonuses.
func (r *CompartmentRestModel) SetReferencedStructs(references map[string]map[string]jsonapi.Data) error {
	if refMap, ok := references["assets"]; ok {
		assets := make([]AssetRestModel, 0)
		for _, ri := range r.Assets {
			if ref, ok := refMap[ri.GetID()]; ok {
				wip := ri
				err := jsonapi.ProcessIncludeData(&wip, ref, references)
				if err != nil {
					return err
				}
				assets = append(assets, wip)
			}
		}
		r.Assets = assets
	}
	return nil
}
```

Do **not** modify `SetToManyReferenceIDs`, `AssetRestModel`, `EquipableRestData`, `IsEquipped`, or `GetEquipableData`. Their bodies are correct.

- [ ] **Step 4: Run the test and confirm it passes**

Run:

```bash
cd services/atlas-effective-stats/atlas.com/effective-stats && \
  go test ./external/inventory/ -run TestCompartmentRestModel_HydratesIncludedAssets -v
```

Expected: PASS. All `Errorf` lines stay quiet; `IsEquipped()` returns true; `GetEquipableData()` returns `(eq, true)` with `eq.Hp == 47` and `eq.Mp == 50`.

If any assertion still fails on a numeric field, the most likely cause is the JSON tag on `AssetRestModel` not matching the `attributes` key in the test document — verify both spellings (e.g. `templateId` not `template_id`).

- [ ] **Step 5: Verify the rest of the package still compiles and tests still pass**

Run:

```bash
cd services/atlas-effective-stats/atlas.com/effective-stats && go build ./... && go test ./external/inventory/ -v
```

Expected: build succeeds; `TestCompartmentRestModel_HydratesIncludedAssets` is the only test in this package and it passes.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-effective-stats/atlas.com/effective-stats/external/inventory/rest.go \
        services/atlas-effective-stats/atlas.com/effective-stats/external/inventory/rest_test.go
git commit -m "fix(effective-stats): hydrate included assets in compartment unmarshal

CompartmentRestModel only implemented SetToManyReferenceIDs, which created
ID-only AssetRestModel stubs with Slot==0. The downstream IsEquipped() gate
(Slot < 0) skipped every asset, dropping all equipment bonuses silently.

Add the api2go-fork hydration hooks (GetReferences, GetReferencedIDs,
GetReferencedStructs, SetToOneReferenceID, SetReferencedStructs) so
included assets are walked and merged into r.Assets via ProcessIncludeData.
Mirrors the working pattern in atlas-npc-shops/.../compartment/rest.go.

Adds a regression unit test that round-trips a JSON:API doc with one
compartment + one included asset and asserts every numeric attribute is
populated."
```

---

## Task 2: Integration test — `fetchEquipmentBonuses` against an `httptest` stub

**Files:**
- Test: `services/atlas-effective-stats/atlas.com/effective-stats/character/initializer_test.go` (create)

Task 1 fixes the bug. This task adds an end-to-end (within-service) test that exercises the full `fetchEquipmentBonuses` chain — HTTP fetch → `jsonapi.Unmarshal` → loop body → `[]stat.Bonus` — using an `httptest.NewServer` as the stub atlas-inventory. It will pass on first run (Task 1 already made the unit test pass), but the regression value is in covering the integration seam where the bug actually manifested.

- [ ] **Step 1: Create the integration test file**

Create `services/atlas-effective-stats/atlas.com/effective-stats/character/initializer_test.go` with the following content:

```go
package character

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"atlas-effective-stats/stat"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
)

// TestFetchEquipmentBonuses_HydratesEquipmentStats stubs atlas-inventory with
// a JSON:API document containing one equipped Medal-style asset (slot -49,
// templateId 1142107, hp 47, mp 50). It then calls fetchEquipmentBonuses and
// asserts the returned []stat.Bonus includes the expected equipment:42
// entries for TypeMaxHp and TypeMaxMp.
//
// This is the integration-level regression net for the bug where the
// CompartmentRestModel only kept asset IDs, leaving Slot==0 and starving
// the IsEquipped() gate.
func TestFetchEquipmentBonuses_HydratesEquipmentStats(t *testing.T) {
	const doc = `{
      "data": {
        "type": "compartments",
        "id": "00000000-0000-0000-0000-000000000001",
        "attributes": {"type": 1, "capacity": 24},
        "relationships": {
          "assets": { "data": [{"type": "assets", "id": "42"}] }
        }
      },
      "included": [{
        "type": "assets",
        "id": "42",
        "attributes": {
          "slot": -49,
          "templateId": 1142107,
          "expiration": "0001-01-01T00:00:00Z",
          "ownerId": 0,
          "strength": 0,
          "dexterity": 0,
          "intelligence": 0,
          "luck": 0,
          "hp": 47,
          "mp": 50,
          "weaponAttack": 0,
          "magicAttack": 0,
          "weaponDefense": 0,
          "magicDefense": 0,
          "accuracy": 0,
          "avoidability": 0,
          "hands": 0,
          "speed": 0,
          "jump": 0
        }
      }]
    }`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(doc))
	}))
	defer srv.Close()

	// RootUrl("INVENTORY") reads INVENTORY_SERVICE_URL and concatenates the
	// path template directly. The trailing slash keeps the resulting URL valid
	// (httptest.Server.URL has no trailing slash by default).
	t.Setenv("INVENTORY_SERVICE_URL", srv.URL+"/")

	l, _ := test.NewNullLogger()
	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create() error = %v", err)
	}
	ctx := tenant.WithContext(context.Background(), ten)

	bonuses, err := fetchEquipmentBonuses(l, ctx, 12)
	if err != nil {
		t.Fatalf("fetchEquipmentBonuses() error = %v", err)
	}

	var sawHp, sawMp bool
	for _, b := range bonuses {
		if b.Source() != "equipment:42" {
			t.Errorf("bonus source = %q, want %q", b.Source(), "equipment:42")
		}
		if b.StatType() == stat.TypeMaxHp && b.Amount() == 47 {
			sawHp = true
		}
		if b.StatType() == stat.TypeMaxMp && b.Amount() == 50 {
			sawMp = true
		}
	}
	if !sawHp {
		t.Errorf("missing TypeMaxHp+47 bonus; got %d bonuses: %#v", len(bonuses), bonuses)
	}
	if !sawMp {
		t.Errorf("missing TypeMaxMp+50 bonus; got %d bonuses: %#v", len(bonuses), bonuses)
	}
}
```

Notes:
- The handler ignores the request path — `fetchEquipmentBonuses` makes exactly one HTTP call, so any path returns the canned doc. URL contract is verified by the post-deploy smoke test in Task 3.
- This file does not call `setupTestRegistry` (used by `processor_test.go`) because `fetchEquipmentBonuses` does not touch the registry.
- The `tenant.Create` and `tenant.WithContext` calls match the pattern in `processor_test.go::createTestContext` (lines 25-30).

- [ ] **Step 2: Run the integration test**

Run:

```bash
cd services/atlas-effective-stats/atlas.com/effective-stats && \
  go test ./character/ -run TestFetchEquipmentBonuses_HydratesEquipmentStats -v
```

Expected: PASS. The handler returns the canned doc; `jsonapi.Unmarshal` (with Task 1's hooks) populates `compartment.Assets[0]` with `Slot=-49, Hp=47, Mp=50`; `fetchEquipmentBonuses` emits `[stat.Bonus{source:"equipment:42", statType:TypeMaxHp, amount:47}, stat.Bonus{source:"equipment:42", statType:TypeMaxMp, amount:50}]`; both `sawHp` and `sawMp` flip true.

If the test fails with "tenant headers required" or similar, double-check that `tenant.WithContext(context.Background(), ten)` was called and that `ctx` was passed to `fetchEquipmentBonuses`.

- [ ] **Step 3: Run the full character package test suite to ensure no collateral breakage**

Run:

```bash
cd services/atlas-effective-stats/atlas.com/effective-stats && go test ./character/ -v
```

Expected: every test in the package passes (existing `TestProcessor_*` and `TestModel_*` plus the new `TestFetchEquipmentBonuses_HydratesEquipmentStats`).

- [ ] **Step 4: Commit**

```bash
git add services/atlas-effective-stats/atlas.com/effective-stats/character/initializer_test.go
git commit -m "test(effective-stats): integration cover fetchEquipmentBonuses with httptest stub

Stand up an httptest.NewServer returning a JSON:API compartment + one
included Medal-style asset (slot -49, hp 47, mp 50), point
INVENTORY_SERVICE_URL at it, and assert fetchEquipmentBonuses emits
equipment:42 bonuses for TypeMaxHp+47 and TypeMaxMp+50.

Covers the integration seam where the included-hydration bug manifested
(rest.go fix landed in prior commit) so a future regression would be
caught here even if the unit-level rest_test.go is bypassed."
```

---

## Task 3: Service-wide verification + smoke test

**Files:** none modified.

This task is two checks: (a) the full service builds and tests cleanly, and (b) the post-deploy curl in `prd.md` §4.1 returns the expected payload.

- [ ] **Step 1: Run the full service build and test suite**

Run:

```bash
cd services/atlas-effective-stats/atlas.com/effective-stats && go build ./... && go test ./...
```

Expected: build succeeds; all tests pass (this includes the two new tests plus every pre-existing test in `character/` and `stat/`).

If any test outside the two new files fails, do not paper over it — investigate. The fix is parsing-only and should have zero behavioural impact on bonus computation, registry, processor, or producer code paths.

- [ ] **Step 2: Post-deploy smoke verification (manual, after the change ships to dev)**

Pick a character known to have HP-bonus equipment. The PRD identifies character `12` on the dev cluster as having a Medal at slot -49 (templateId 1142107) granting `hp +47, mp +50`.

From a kubectl-attached pod in the `atlas` namespace (or from the local laptop with port-forwarding), run:

```bash
curl -s 'http://atlas-ingress/api/worlds/0/channels/0/characters/12/stats' \
  -H 'TENANT_ID: <tenant-uuid>' \
  -H 'REGION: GMS' \
  -H 'MAJOR_VERSION: 83' \
  -H 'MINOR_VERSION: 1' \
  | jq '.data.attributes | {maxHP, maxMP, bonuses}'
```

Expected response shape:

```json
{
  "maxHP": <base + 47 + any other contributions>,
  "maxMP": <base + 50 + any other contributions>,
  "bonuses": [
    { "source": "equipment:<assetId>", "statType": "max_hp", "amount": 47 },
    { "source": "equipment:<assetId>", "statType": "max_mp", "amount": 50 },
    ...
  ]
}
```

Verification checklist:
- [ ] `bonuses` contains at least one entry whose `source` matches `^equipment:\d+$`.
- [ ] `maxHP > characters/12.attributes.maxHp` (compare against `curl http://atlas-ingress/api/characters/12 | jq '.data.attributes.maxHp'`).
- [ ] If the character has equipment granting STR/DEX/INT/LUK/WeaponAttack/etc., corresponding `bonuses` entries appear with the matching `statType`.

If the response still shows only the `passive:1002` speed entry, re-check the deploy: the rest.go change must be in the running image. `kubectl exec` into a pod and `grep -c GetReferencedIDs /atlas/effective-stats || true` (or equivalent) is one quick sanity check.

- [ ] **Step 3: No further commit required**

This task contains no code changes — it is verification only. If both Step 1 and Step 2 pass, the acceptance criteria in `prd.md` §10 are satisfied:

- [x] Unit test for JSON:API round-trip (Task 1)
- [x] Integration test for `fetchEquipmentBonuses` (Task 2)
- [x] `go build ./...` and `go test ./...` pass (Step 1 above)
- [x] Smoke verification on dev cluster (Step 2 above)
- [x] No frontend changes (atlas-ui already renders `+bonus` when present — see PRD §7)

---

## Out of Scope (do not do as part of this plan)

Per `prd.md` §2 non-goals and §9 open questions, the following are explicitly **not** part of this task:

- Auditing other services for the same `SetToManyReferenceIDs`-only pattern. (PRD §9 lists this as a follow-up.)
- Any change to `requests.go`, `character/initializer.go`, or any non-test file outside `external/inventory/rest.go`.
- Any change to atlas-inventory, atlas-character, or atlas-ui.
- Any new endpoints, schema changes, multipliers, stat-curve changes, buff path changes, or passive-skill path changes.

If during implementation you find code that "looks broken" in any of these areas, file a follow-up — do not bundle it into this fix.
