# Context — task-031: atlas-effective-stats Equipment Bonus Hydration Fix

Quick-reference companion to `plan.md`. Read this once before starting; refer back as needed.

## The Bug in One Sentence

`CompartmentRestModel` only implements `SetToManyReferenceIDs`, so `jsonapi.Unmarshal` creates ID-only `AssetRestModel` stubs (every field zero except `Id`). The `IsEquipped()` gate (`Slot < 0`) skips every stub, so `fetchEquipmentBonuses` always returns an empty `[]stat.Bonus`.

## The Fix in One Sentence

Add the missing api2go-fork hydration interface (`GetReferences`, `GetReferencedIDs`, `GetReferencedStructs`, `SetToOneReferenceID`, `SetReferencedStructs`) to `CompartmentRestModel`, mirroring the working pattern in `services/atlas-npc-shops/atlas.com/npc/compartment/rest.go:37-100`.

## Key Files

| File | Why |
|---|---|
| `services/atlas-effective-stats/atlas.com/effective-stats/external/inventory/rest.go` | The only source file that changes. Add 5 methods + 1 import. |
| `services/atlas-effective-stats/atlas.com/effective-stats/external/inventory/requests.go` | Read-only reference. Confirms the URL pattern (`characters/{id}/inventory/compartments?type=1&include=assets`) and that `RequestEquipCompartment` returns `requests.Request[CompartmentRestModel]`. No changes. |
| `services/atlas-effective-stats/atlas.com/effective-stats/character/initializer.go` | Read-only reference. The downstream `fetchEquipmentBonuses` (lines 97-168) is correct as written — it just gets accurate input after the fix. No changes. |
| `services/atlas-npc-shops/atlas.com/npc/compartment/rest.go` | **Reference template.** Mirror the 5 hydration methods here verbatim, substituting `AssetRestModel` for `asset.BaseRestModel`. |
| `libs/atlas-rest/requests/response.go` | Confirms the consumer path: `requests.GetRequest` → `unmarshalResponse[A]` → `jsonapi.Unmarshal(body, &result)`. So adding hooks on `*CompartmentRestModel` is all that's needed; no producer/configurator changes. |
| `libs/atlas-rest/requests/url.go` | Confirms `RootUrl("INVENTORY")` reads env `INVENTORY_SERVICE_URL` (falling back to `BASE_SERVICE_URL`). The integration test sets `INVENTORY_SERVICE_URL` via `t.Setenv`. |

## Key Decisions (from design.md)

- **Option A** (full hydration via api2go hooks) was chosen over Option B (per-asset N+1 fetches) and Option C (denormalised flat-fetch). Option A keeps the consumer pattern consistent with the rest of the monorepo and adds zero HTTP overhead.
- **No changes to `AssetRestModel`** — its existing `GetName`/`GetID`/`SetID` trio is the full leaf-resource interface in the api2go fork. The flat JSON tags (`hp`, `mp`, `slot`, `templateId`, …) are how `jsonapi.ProcessIncludeData` populates fields from the included resource's `attributes` object.
- **No changes to `requests.go`** — the URL is correct, the headers are correct (tenant headers propagate via `TenantHeaderDecorator`), and the response type stays the same.
- **No changes to `fetchEquipmentBonuses`** — the loop body, the `IsEquipped()` gate, and the per-stat `if x > 0` checks are all correct. They were starving on stub input.

## Conventions in Play

- **api2go fork**: `github.com/jtumidanski/api2go v1.0.4`. The fork exposes `jsonapi.Reference`, `jsonapi.ReferenceID`, `jsonapi.MarshalIdentifier`, `jsonapi.Data`, and `jsonapi.ProcessIncludeData(target, ref, refs)` — all used in the npc-shops template.
- **Test framework**: standard `testing` plus `github.com/sirupsen/logrus/hooks/test` for null loggers and `github.com/Chronicle20/atlas/libs/atlas-tenant` for tenant-context fixtures. See `services/atlas-effective-stats/atlas.com/effective-stats/character/processor_test.go:25-30` for the canonical `createTestContext` pattern.
- **Multi-tenancy**: `tenant.Create(uuid.New(), "GMS", 83, 1)` then `tenant.WithContext(context.Background(), ten)`. `fetchEquipmentBonuses` requires this context — without it the request fails because tenant headers are not stamped.
- **Service URL**: env var `INVENTORY_SERVICE_URL` is concatenated directly with the path template (no implicit slash), so `t.Setenv("INVENTORY_SERVICE_URL", srv.URL+"/")` is the safe form for an `httptest.Server` URL (which has no trailing slash).
- **Stat type identifiers**: `stat.TypeMaxHp = "max_hp"`, `stat.TypeMaxMp = "max_mp"`, `stat.TypeStrength = "strength"`, etc. (see `services/atlas-effective-stats/atlas.com/effective-stats/stat/model.go`). The wire-format identifiers are the snake_case strings — JSON serialization emits them via the `MarshalJSON` shim around line 100-110.
- **Bonus accessors**: `Bonus.Source() string`, `Bonus.StatType() Type`, `Bonus.Amount() int32`. Constructor: `stat.NewBonus(source, statType, amount)`.

## Verification Surface

After the fix:
1. `external/inventory/rest_test.go` (new) — JSON:API round-trip; fails on `main`, passes after.
2. `character/initializer_test.go` (new) — `httptest`-stubbed integration; passes once the unit fix lands.
3. `go build ./... && go test ./...` from `services/atlas-effective-stats/atlas.com/effective-stats` — clean.
4. Manual: `curl /api/worlds/0/channels/0/characters/12/stats` against the dev cluster — `bonuses[]` now contains `equipment:<id>` entries; `maxHP` strictly greater than the character's base `maxHp`.

## Things That Look Tempting But Are Out of Scope

- **Auditing other services** for the same `SetToManyReferenceIDs`-only pattern (PRD §9 — separate follow-up).
- **Refactoring `CompartmentRestModel`** to share types with atlas-npc-shops or other services. Each service maintains its own consumer-side rest model by convention; do not introduce a shared lib here.
- **Touching the buff or passive-skill bonus paths** in `initializer.go`. They work today; do not refactor them while you're here.
- **Adding cash-equip or account-bonus support.** PRD §3 lists this as a future-agent benefit, not a deliverable for this task.
- **Making `fetchEquipmentBonuses` log additional context** ("with N equipped assets"). The existing debug log at line 166 is sufficient; the bonus count alone signals whether the fix is taking effect.

## Background References (skim before starting if you want extra context)

- `prd.md` (this folder) — full diagnosis chain in §11; reproduction steps in §4.1.
- `design.md` (this folder) — alternative options considered in §1; risks and rollback in §5.
- `services/atlas-npc-shops/atlas.com/npc/compartment/rest.go` — the working reference. Notice the structural symmetry: same `compartments` resource type, same `assets` to-many relationship, same flat-attribute leaf resource.
