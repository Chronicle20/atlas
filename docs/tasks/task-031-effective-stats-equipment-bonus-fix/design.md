# atlas-effective-stats: equipment bonuses silently dropped — Design

Version: v1
Status: Approved
Created: 2026-04-26
Companion to: `prd.md` (this folder)
---

## 1. Decision

Implement **Option A** from `prd.md` §4.4: add the missing api2go-fork hydration hooks to `CompartmentRestModel` so `included` assets are fully populated during `jsonapi.Unmarshal`.

### Why this option

- The api2go fork (`github.com/jtumidanski/api2go`) supports full struct hydration via `SetReferencedStructs(map[string]map[string]jsonapi.Data) error` plus `jsonapi.ProcessIncludeData(target, ref, refs)`. This is the established pattern for compartment-with-assets in this monorepo.
- Reference implementation already exists at `services/atlas-npc-shops/atlas.com/npc/compartment/rest.go:37-100` — same compartment+assets shape, working today.
- Option B (per-asset N+1 fetches) trades a one-shot fetch for `1 + N` HTTP calls per character with no offsetting benefit.
- Option C (switch to the denormalized `/api/characters/{id}/inventory` endpoint) over-fetches every inventory type when only the equip compartment is needed, and weakens the consumer contract by binding to a coarser response shape.
- The bug is in how this service parses the response, not in the response shape. Fix the consumer.

## 2. Architecture

The fix is contained to one file in one service: `services/atlas-effective-stats/atlas.com/effective-stats/external/inventory/rest.go`.

```
┌────────────────────────────────────┐
│ atlas-inventory                    │
│ GET /api/characters/{id}/inventory │
│   /compartments?type=1&            │
│   include=assets                   │
└──────────────┬─────────────────────┘
               │ JSON:API
               │ { data: [{type:compartments,...,
               │           relationships: {assets:{data:[...]}}}],
               │   included: [{type:assets,id,attributes:{slot,hp,mp,...}}, ...] }
               ▼
┌────────────────────────────────────┐
│ external/inventory/requests.go     │
│  RequestEquipCompartment(id)       │
│  → requests.GetRequest[Compartment │
│        RestModel](url)             │
└──────────────┬─────────────────────┘
               │ jsonapi.Unmarshal()
               │   ① top-level data → CompartmentRestModel fields
               │   ② SetID()
               │   ③ SetToManyReferenceIDs("assets", [...])  ← stubs created
               │   ④ SetReferencedStructs(included)         ← NEW: hydrates stubs
               ▼
┌────────────────────────────────────┐
│ character/initializer.go            │
│  fetchEquipmentBonuses               │
│  for asset in compartment.Assets:    │
│    if !asset.IsEquipped() continue   │ ← Slot now populated, gate works
│    equipData, _ := asset.GetEquipa…  │ ← stats populated
│    bonuses = append(bonuses, ...)    │
└──────────────┬─────────────────────┘
               │
               ▼  populated []stat.Bonus
        Computed.maxHp, maxMp, ... reflect equipment
```

`character/initializer.go::fetchEquipmentBonuses` is unchanged — its loop body is correct, it just gets accurate input.

## 3. Implementation

### 3.1 `CompartmentRestModel` interface set

After this change, `CompartmentRestModel` will implement (in addition to today's `GetName`, `GetID`, `SetID`, `SetToManyReferenceIDs`):

```go
// Existing (unchanged):
func (r CompartmentRestModel) GetName() string
func (r CompartmentRestModel) GetID() string
func (r *CompartmentRestModel) SetID(strId string) error
func (r *CompartmentRestModel) SetToManyReferenceIDs(name string, IDs []string) error

// New:
func (r CompartmentRestModel) GetReferences() []jsonapi.Reference
func (r CompartmentRestModel) GetReferencedIDs() []jsonapi.ReferenceID
func (r CompartmentRestModel) GetReferencedStructs() []jsonapi.MarshalIdentifier
func (r *CompartmentRestModel) SetToOneReferenceID(name, ID string) error
func (r *CompartmentRestModel) SetReferencedStructs(refs map[string]map[string]jsonapi.Data) error
```

The body of each new method follows `services/atlas-npc-shops/atlas.com/npc/compartment/rest.go` verbatim, substituting `AssetRestModel` for `asset.BaseRestModel`. Notably:

- `GetReferences()` returns one entry: `{Type: "assets", Name: "assets"}`.
- `GetReferencedIDs()` and `GetReferencedStructs()` are required for *outgoing* marshalling (api2go uses them when serialising). atlas-effective-stats does not currently marshal a `CompartmentRestModel` outbound, but the api2go-fork interface set treats them as part of the contract — implement them to mirror the reference template and avoid future surprises if the service ever needs to round-trip the type.
- `SetToOneReferenceID(_, _ string) error` is a no-op satisfier — the model has no to-one relationships but the interface requires the method.
- `SetReferencedStructs` iterates the stub assets that `SetToManyReferenceIDs` already populated, looks up each by ID in `refs["assets"]`, and calls `jsonapi.ProcessIncludeData(&wip, ref, refs)` to hydrate the full struct, then replaces `r.Assets` with the hydrated slice.

`SetToManyReferenceIDs` is left as-is. The two methods are complementary: the first creates ID-only stubs during the relationships pass, the second walks the `included` array and replaces them with fully hydrated copies.

### 3.2 `AssetRestModel`: leaf, no changes

`AssetRestModel` already implements `GetName()`, `GetID()`, and `SetID()`, which is the full set required for an api2go-fork *leaf* (no further references of its own). The flat JSON tags on its fields — `slot`, `hp`, `mp`, `strength`, etc. — are how `ProcessIncludeData` populates it from the included resource's `attributes` object. No changes required.

The reference template's leaf (`services/atlas-npc-shops/atlas.com/npc/asset/rest.go::BaseRestModel`) implements the same minimal trio — confirming the leaf pattern.

### 3.3 No other code changes

- `requests.go` is unchanged.
- `character/initializer.go::fetchEquipmentBonuses` is unchanged. The `IsEquipped()` and `GetEquipableData()` gates were always correct; they were starving on stub input.
- No producer, consumer, or registry impact. No Kafka events. No persistence.

## 4. Test Strategy

Two new tests, scoped narrowly to the fix.

### 4.1 Unit test — `external/inventory/rest_test.go` (new file)

Hand-rolled JSON:API document fed through `jsonapi.Unmarshal` into a `CompartmentRestModel`. Asserts that `compartment.Assets[0]` has the full attribute set populated from `included`.

```go
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
          "slot": -49, "templateId": 1142107,
          "hp": 47, "mp": 50,
          "strength": 0, "dexterity": 0, "intelligence": 0, "luck": 0,
          "weaponAttack": 0, "magicAttack": 0,
          "weaponDefense": 0, "magicDefense": 0,
          "accuracy": 0, "avoidability": 0,
          "hands": 0, "speed": 0, "jump": 0,
          "ownerId": 0, "expiration": "0001-01-01T00:00:00Z"
        }
      }]
    }`)

    var c CompartmentRestModel
    if err := jsonapi.Unmarshal(doc, &c); err != nil { t.Fatal(err) }

    if len(c.Assets) != 1 { t.Fatalf("got %d assets, want 1", len(c.Assets)) }
    a := c.Assets[0]
    if a.Id != 42        { t.Errorf("Id = %d, want 42", a.Id) }
    if a.Slot != -49     { t.Errorf("Slot = %d, want -49", a.Slot) }
    if a.Hp != 47        { t.Errorf("Hp = %d, want 47", a.Hp) }
    if a.Mp != 50        { t.Errorf("Mp = %d, want 50", a.Mp) }
    if a.TemplateId != 1142107 { t.Errorf("TemplateId = %d, want 1142107", a.TemplateId) }
}
```

This test fails on `main` today (`Slot == 0`, `Hp == 0`, `Mp == 0`) and passes after the fix. It is the minimum-viable regression net for this class of bug.

### 4.2 Integration test — `character/initializer_test.go` (new file)

Stand up an `httptest.NewServer` that serves a fixed JSON:API document for the equip-compartment URL. Point `INVENTORY_SERVICE_URL` at it via `t.Setenv`. Call `fetchEquipmentBonuses(l, ctx, characterId)` and assert the returned `[]stat.Bonus` contains the expected `equipment:42` entries.

Narrow scope: the test stubs only atlas-inventory and exercises only `fetchEquipmentBonuses` — not the full `InitializeCharacter` chain. The wider-flow test (stubbing character + inventory + buffs + skills + data) is out of scope for this task; the dev-cluster smoke test in `prd.md` §4.1 covers the end-to-end path.

The harness:

```go
func TestFetchEquipmentBonuses_HydratesEquipmentStats(t *testing.T) {
    doc := `{ "data": {"type":"compartments", "id":"...", "attributes":{...},
              "relationships":{"assets":{"data":[{"type":"assets","id":"42"}]}}},
              "included":[{"type":"assets","id":"42",
                "attributes":{"slot":-49,"templateId":1142107,"hp":47,"mp":50, ...}}] }`

    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/vnd.api+json")
        _, _ = w.Write([]byte(doc))
    }))
    defer srv.Close()
    t.Setenv("INVENTORY_SERVICE_URL", srv.URL)

    l, _ := test.NewNullLogger()
    ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
    ctx := tenant.WithContext(context.Background(), ten)

    bonuses, err := fetchEquipmentBonuses(l, ctx, 12)
    if err != nil { t.Fatal(err) }

    // Expect TypeMaxHp 47 and TypeMaxMp 50, both sourced "equipment:42".
    var sawHp, sawMp bool
    for _, b := range bonuses {
        if b.Source() != "equipment:42" { t.Errorf("bonus source = %q, want equipment:42", b.Source()) }
        if b.StatType() == stat.TypeMaxHp && b.Amount() == 47 { sawHp = true }
        if b.StatType() == stat.TypeMaxMp && b.Amount() == 50 { sawMp = true }
    }
    if !sawHp { t.Error("missing TypeMaxHp+47 bonus") }
    if !sawMp { t.Error("missing TypeMaxMp+50 bonus") }
}
```

Implementation notes:

- This is the first httptest stub in `atlas-effective-stats`; existing tests under `character/` use `miniredis` only. The new test does not need miniredis (it does not exercise the registry).
- `t.Setenv` is the standard `testing` API for env-scoped overrides; it auto-restores at test teardown.
- The handler does not bother route-matching — `fetchEquipmentBonuses` only issues one HTTP call, so any inbound request can return the canned doc.

### 4.3 Smoke verification (post-deploy, manual)

Per `prd.md` §4.1 / §10:

```sh
curl -s 'http://atlas-ingress/api/worlds/0/channels/0/characters/12/stats' \
  -H 'TENANT_ID: <tenant>' -H 'REGION: GMS' \
  -H 'MAJOR_VERSION: 83' -H 'MINOR_VERSION: 1' \
  | jq '.data.attributes.bonuses'
```

Expect at least one `{source:"equipment:<id>", statType:"TypeMaxHp", amount:47}` entry plus its `TypeMaxMp:50` sibling, and `data.attributes.maxHP > characters/12.attributes.maxHp`.

## 5. Risks & Rollback

- **Risk: `ProcessIncludeData` semantics differ from the npc-shops template.** Mitigation: the unit test in §4.1 catches any divergence; if `ProcessIncludeData` does not in fact populate flat fields from `attributes`, the test fails locally before the change ships.
- **Risk: future api2go-fork upgrade changes the interface set.** Mitigation: matching the existing in-repo template means any breaking upgrade affects atlas-npc-shops first, surfacing the issue in a service that already has integration coverage.
- **Risk: stubbing one endpoint with a permissive handler hides URL drift.** Acceptable: this is a fix-test, not a contract test. The dev-cluster smoke verification in §4.3 is the URL contract check.
- **Rollback:** revert the single commit touching `external/inventory/rest.go` (and the two new test files). No data, no schema, no Kafka — pure parsing logic. Rollback restores today's behaviour exactly: empty equipment bonuses, no other side effects.

## 6. Acceptance Criteria (from PRD, restated)

- `CompartmentRestModel` JSON:API round-trip unit test exists and passes.
- `fetchEquipmentBonuses` integration test against an `httptest` stub exists and passes.
- `go build ./...` and `go test ./...` pass in `services/atlas-effective-stats/atlas.com/effective-stats`.
- Dev-cluster smoke (`character 12`) shows `equipment:<id>` bonuses with `+47 HP / +50 MP` and `Computed.maxHp > Base.maxHp`.
- No frontend changes (atlas-ui already renders `+bonus` when present — see `prd.md` §7).
