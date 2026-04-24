# Task 022 — Seed Counts Context Pack

Quick reference for agents executing `plan.md`. Pair with `prd.md`, `design.md`, `api-contracts.md`.

---

## Feature in one sentence

Add eight tenant-scoped `GET .../seed/status` JSON:API endpoints across seven Go services, then surface their counts next to the eight Seed buttons on `/setup`, polled every 5 s.

---

## Reference implementation

`services/atlas-data/atlas.com/data/data/status.go` is the canonical template for every new status handler. Re-read it before starting any backend task:

- `StatusRestModel` (lines 17–58) — JSON:API model implementing `GetName()/GetID()/SetID()/GetReferences()/…`.
- `queryStatus` (lines 60–91) — `COUNT(*)` then `MAX(updated_at)` via `Select("MAX(updated_at)").Row().Scan(&sql.NullString)`.
- `parseDBTime` (lines 93–108) — 6-format tolerant parser, returns zero-time on total failure.
- `handleGetStatus` (lines 110–135) — reads tenant, formats `*string` RFC3339, marshals via `server.MarshalResponse`.

---

## Tenant scoping

Every service with a sub-resource `entity` registers `database.RegisterTenantCallbacks`, which means GORM `Query` / `Row` / `Update` / `Delete` callbacks auto-inject `WHERE tenant_id = <ctx.tenant.Id()>`. In the new `Count` methods we rely on this implicit filter: just do `db.WithContext(p.ctx).Model(&entity{}).Count(&count)`. Tenant is read from ctx — never passed explicitly.

Source: `libs/atlas-database/tenant_scope.go:39`.

---

## Per-service inventory

All file paths are relative to repo root.

### atlas-drop-information (`services/atlas-drop-information/atlas.com/dis/`)

| Path | Notes |
|---|---|
| `monster/drop/entity.go` | `entity`, table `monster_drops`. No `updated_at`. |
| `monster/drop/processor.go` | `Processor` interface already has `GetAll`, `GetForMonster`, `GetForItem`. Add `Count`. |
| `monster/drop/processor_test.go` | Existing test harness: `testDatabase(t)`, `testTenant()`, `seedTestData(t,db,tenantId,monsterId,items)`. |
| `continent/drop/entity.go` | `entity`, table `continent_drops`. No `updated_at`. |
| `continent/drop/processor.go` + test | Mirror monster pattern. Seed test uses direct `INSERT` with `(tenant_id, continent_id, item_id, ...)`. |
| `reactor/drop/entity.go` | `entity`, table `reactor_drops`. No `updated_at`. |
| `reactor/drop/processor.go` + test | Same. |
| `seed/resource.go` | `InitResource` registers `POST /drops/seed`. Add `GET /drops/seed/status` alongside. |
| `seed/processor.go` | Uses `errgroup.WithContext` + `sync.Mutex` for the three sub-seeds (good reference for the status handler pattern). |

### atlas-gachapons (`services/atlas-gachapons/atlas.com/gachapons/`)

| Path | Notes |
|---|---|
| `gachapon/entity.go` | `entity`, table `gachapons`. PK `ID` is string. **No `updated_at`.** |
| `gachapon/processor.go` | `Processor` has `GetAll/GetById/Create/Update/Delete`. Add `Count`. |
| `item/entity.go`, `item/processor.go` | Gachapon items. No `updated_at`. |
| `global/entity.go`, `global/processor.go` | Global items. No `updated_at`. |
| `seed/resource.go` | Register `GET /gachapons/seed/status`. |
| `seed/processor.go` | Sequential seed (not errgroup). Status handler is still parallel. |

### atlas-npc-conversations (`services/atlas-npc-conversations/atlas.com/`)

| Path | Notes |
|---|---|
| `npc/conversation/npc/entity.go` | `Entity`, table `conversations`. Has explicit `UpdatedAt`. |
| `npc/conversation/npc/processor.go` | `Processor` has Create/Update/Delete/AllProvider/ByIdProvider/AllByNpcIdProvider/Seed/DeleteAllForTenant. Add `Count`. |
| `npc/conversation/npc/resource.go` | `InitResource` registers routes including `POST /npcs/conversations/seed` (line 33). Add `GET /npcs/conversations/seed/status`. |
| `npc/conversation/quest/entity.go` | `Entity`, table `quest_conversations`. Has explicit `UpdatedAt`. |
| `npc/conversation/quest/processor.go` | Same interface shape. Add `Count`. |
| `npc/conversation/quest/resource.go` | Mirror. Register `GET /quests/conversations/seed/status`. |
| `npc/conversation/quest/status/` | ⚠️ Pre-existing package named `status` holding remote `quest-status` models from atlas-quest — **unrelated** to seed status. Keep new seed-status files in the parent package (file name `seed_status.go`). |

### atlas-npc-shops (`services/atlas-npc-shops/atlas.com/`)

| Path | Notes |
|---|---|
| `npc/shops/entity.go` | `Entity` embeds `gorm.Model`, so `UpdatedAt` exists. Table `shops`. |
| `npc/shops/processor.go` | `Processor` has decorators + Create/Update/AddCommodity/GetAllShops/etc. Add `Count`. |
| `npc/commodities/entity.go` | `Entity` embeds `gorm.Model`. Table `commodities`. |
| `npc/commodities/processor.go` | Add `Count`. |
| `npc/seed/resource.go` | Register `GET /shops/seed/status`. |

### atlas-portal-actions (`services/atlas-portal-actions/atlas.com/portal/script/`)

| Path | Notes |
|---|---|
| `entity.go` | `Entity`, table `portal_scripts`. Explicit `CreatedAt`/`UpdatedAt`. |
| `processor.go` | `ScriptProcessor` interface. Add `Count`. |
| `resource.go` | Registers `POST /portals/scripts/seed`. Add `GET /portals/scripts/seed/status`. |

### atlas-reactor-actions (`services/atlas-reactor-actions/atlas.com/reactor/script/`)

| Path | Notes |
|---|---|
| `entity.go` | Table `reactor_scripts`. Explicit `UpdatedAt`. |
| `processor.go` | `ScriptProcessor`. Add `Count`. |
| `resource.go` | Register `GET /reactors/actions/seed/status`. |

### atlas-map-actions (`services/atlas-map-actions/atlas.com/map-actions/script/`)

| Path | Notes |
|---|---|
| `entity.go` | Table `map_scripts`. Explicit `UpdatedAt`. |
| `processor.go` | `ScriptProcessor`. Add `Count`. |
| `resource.go` | Register `GET /maps/actions/seed/status`. |

---

## Ingress

Two files must stay in sync. Only **one** route needs editing — the drops `$`-anchored block:

- `deploy/shared/routes.conf:151` — change `^/api/drops/seed$` → `^/api/drops/seed(/.*)?$`.
- `deploy/k8s/ingress.yaml:168` — same edit.

All other new paths already fall under a catch-all `(/.*)?$` block. Verify manually with `curl` after deploy.

---

## Frontend (`services/atlas-ui/`)

### Files touched

| File | Change |
|---|---|
| `src/components/features/setup/SetupRow.tsx` | **New.** Houses `SetupRow` (extracted from inline `GameDataRow` in SetupPage), plus the `formatCount` and `pluralize` helpers (moved out of SetupPage). |
| `src/services/api/seed.service.ts` | Add 8 status interfaces + 8 `fetchJsonApi<A>` getters. Existing `fetchJsonApi` helper (lines 29–46) is reused verbatim. |
| `src/lib/hooks/api/useSeed.ts` | Add 8 `useXxxSeedStatus` query hooks (mirror `useWzInputStatus` at lines 98–107). Extend each of the 8 existing `useSeedXxx` mutations (lines 20–58) with an `onSuccess` that invalidates the matching status key. |
| `src/pages/SetupPage.tsx` | Delete inline `GameDataRow` (lines 89–118), `SeedButton` (lines 39–64), `formatCount` / `pluralize` (lines 81–87). Import from `SetupRow.tsx`. Replace Seed Data grid (lines 368–384) with 8 `<SetupRow>` rows driven by the new hooks. |

### Existing scaffolding to match

- Hook keys pattern: `const wzInputStatusKey = (tenantId: string) => ['wzInputStatus', tenantId] as const;` (useSeed.ts:16).
- Query polling shape (useSeed.ts:98–107):
  ```ts
  useQuery({
    queryKey: activeTenant ? <key>(activeTenant.id) : ['<type>', 'none'],
    queryFn: () => seedService.getXxx(activeTenant!),
    enabled: !!activeTenant,
    staleTime: 0,
    refetchInterval: 5000,
  });
  ```
- Invalidation pattern (useSeed.ts:60–71, `useUploadWzFiles`): `onSuccess` opens with `if (!activeTenant) return;` then `queryClient.invalidateQueries({ queryKey: <key>(activeTenant.id) })`.

### Vitest setup

- Testing libs already installed: vitest + `@testing-library/react` + `jsdom`.
- No existing `src/lib/hooks/api/__tests__/useSeed.test.ts`. Create it fresh.
- Other hook tests in the repo (e.g., `src/context/__tests__/tenant-context.test.tsx`) wrap renders in a `QueryClientProvider` + `TenantProvider` stack; mirror that.

---

## Test harness cheat sheet

Backend sub-resource processor tests (`processor_test.go`) follow this pattern:

```go
func testDatabase(t *testing.T) *gorm.DB {
    l, _ := test.NewNullLogger()
    db, _ := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
    database.RegisterTenantCallbacks(l, db)
    _ = drop.Migration(db)
    return db
}

func testTenant() tenant.Model {
    t, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
    return t
}
```

Tests then call `drop.NewProcessor(l, ctx, db)` where `ctx = tenant.WithContext(context.Background(), te)`. For `Count` tests:

1. Empty → `(0, nil, nil)`.
2. Seeded → `(N, <non-nil if table has updated_at>, nil)`.
3. Two-tenant isolation → count under tenant A excludes tenant B rows.

Handler tests use the extraction/status pattern in `services/atlas-wz-extractor/atlas.com/wz-extractor/extraction/status_test.go` — set all four tenant headers (`TENANT_ID`, `REGION`, `MAJOR_VERSION`, `MINOR_VERSION`), hit the handler via `httptest.NewRecorder`, assert envelope shape.

---

## Build & verify

Each Go service has its own `go.mod`. From repo root:

```bash
cd services/atlas-drop-information/atlas.com/dis && go build ./... && go test ./...
```

Repeat per service touched. Docker build is not needed at the task level but is the final smoke.

Frontend:

```bash
cd services/atlas-ui && npm run lint && npm run test && npm run build
```

---

## Checked-in working tree

Branch: `main`. Untracked task folder `docs/tasks/task-022-seed-counts/` holds the spec artifacts. Create a feature branch off `main` for implementation (branch protection blocks direct pushes to `main`).
