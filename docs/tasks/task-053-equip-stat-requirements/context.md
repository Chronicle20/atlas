# task-053 — Context Reference

Quick lookup for the executing agent. Source of truth is `prd.md` and `design.md`; this file just collects the breadcrumbs.

## Service under change

`services/atlas-effective-stats/atlas.com/effective-stats/` — Go module `atlas-effective-stats` (short, **not** the full path).

Only this service is modified. `atlas-data`, `atlas-inventory`, `atlas-character`, `atlas-channel`, `atlas-ui` are untouched (PRD §7).

## Files: read first

| Path | Why it matters |
|------|----------------|
| `character/model.go` | Immutable `Model` with `bonuses []stat.Bonus`, builder methods, `ComputeEffectiveStats`, `Recompute`, `MaxHpMpCap = 30000` clamp, JSON marshal/unmarshal. Substantially extended in this task. |
| `character/processor.go` | `Processor` interface + `ProcessorImpl`. `AddEquipmentBonuses`/`RemoveEquipmentBonuses`/`SetBaseStats` exist; signatures change. `checkAndPublishClampCommands` already publishes `CLAMP_HP`/`CLAMP_MP` on cap decrease — reused as-is. |
| `character/registry.go` | Redis-backed via `atlas.TenantRegistry[uint32, Model]`; mutations call `m.Recompute()` then `Put`. New helpers added here. |
| `character/initializer.go` | Lazy init: fetch base stats → equipment → buffs → passives → `Recompute`. Equipment fetch path replaced. |
| `character/resource.go` | REST GET handler — calls `model.Bonuses()` and `model.Computed()`. **Signature unchanged.** |
| `kafka/consumer/character/consumer.go` | `handleStatChanged` filters on `MAX_HP/MAX_MP/STRENGTH/DEXTERITY/INTELLIGENCE/LUCK`; merges values onto base; calls `SetBaseStats`. Filter is widened and a new branch is added for `LEVEL`/`JOB`. |
| `kafka/consumer/asset/consumer.go` | `handleAssetMoved` → `handleItemEquipped`/`handleItemUnequipped`; `handleAssetDeleted`. Calls `AddEquipmentBonuses` / `RemoveEquipmentBonuses`. Updated to pass `e.TemplateId`. |
| `external/character/rest.go` | `RestModel` for atlas-character lookup. Add `JobId job.Id` here. |
| `external/inventory/rest.go` | `AssetRestModel` + `EquipableRestData`. `Slot < 0` → equipped. Already has `TemplateId`. Unchanged. |
| `external/data/skill/{rest,requests}.go` | The shape the new equipment data client mirrors. |

## New files

```
external/data/equipment/rest.go          # RestModel mirroring atlas-data equipment response (only the six req fields)
external/data/equipment/requests.go      # RequestById builder using requests.RootUrl("DATA")
external/data/equipment/cache.go         # Per-tenant cache + Provider closure factory
external/data/equipment/cache_test.go    # Cache hit/miss/tenant-isolation tests
character/qualification.go               # meetsRequirements, AppliedStats, wearerClassMask, QualifiedEquipment
character/qualification_test.go          # Unit tests for qualification engine
```

The asset consumer test file `kafka/consumer/asset/consumer_test.go` may not exist yet — create it for the integration coverage in Phase E.

## Key design decisions (cross-reference design.md §1)

| # | Decision | Implication for the implementer |
|---|----------|-----|
| 1 | Equipment is the source of truth for `m.equipped` (snapshot map); `m.bonuses[]` carries **only** `buff:*` and `passive:*` entries after this task. | `AddBuffBonuses`/`AddPassiveBonuses` keep using `bonuses[]`. `AddEquipmentBonuses` writes to `equipped`. Don't double-count. |
| 2 | Snapshots store `[]stat.Bonus` pre-extracted (not raw `EquipableRestData`). | `extractEquipmentBonuses` (already in `kafka/consumer/asset/consumer.go`) is the canonical extractor. The initializer's duplicated logic can call it too. |
| 3 | Equipment-data cache is a package singleton with `sync.Once`; keyed by `(tenant.Id, templateId)` behind `sync.RWMutex`. | Standard project pattern. Get tenant via `tenant.MustFromContext(ctx)`. |
| 4 | Qualification logic is a pure method on `Model` taking a `Provider`. | Unit-testable without I/O stubs; integration tests inject a stub provider. |
| 5 | Re-evaluation funnels through one entry point: `Processor.RecomputeEquipmentBonuses`. | `SetBaseStats` and `SetWearerProfile` call it internally. Asset add/remove call it after mutating the snapshot map. |
| 6 | Wearer level + jobId go in a new `WearerProfile`; `stat.Base` is **not** widened. | Don't touch `libs/atlas-constants/stat/`. |
| 7 | STAT_CHANGED with `TypeLevel` / `TypeJob` carries `values=nil`; consumer must refetch wearer record. | Branch the consumer; the existing numeric path stays. |
| 8 | A 2-equip mutual cycle resolves to "neither qualifies" — `qualified` starts empty and only grows when an item passes under the prior round's stats. | No special-case code; locked in by a unit test. |
| 9 | atlas-data fetch failure with cold cache → asset dropped from current evaluation, WARN log, no retry. | Provider returns `(_, false)`; caller treats as "doesn't qualify". |
| 10 | Tenant scoping: `map[tenant.Id]map[uint32]EquipmentRequirements` behind one `sync.RWMutex`. | Standard pattern. |

## reqJob bitmask gotcha (NOT in design.md but required)

Design §4.1 line 220 writes `uint16(jobId) & r.ReqJob`. **This is wrong as written** — atlas internal `job.Id` is not a v83-client bitmask. Internal jobIds: Beginner=0, Warrior branch=100/110/111/112/120/121/122, Magician branch=200/210-232, Bowman=300/310-322, Thief=400/410-422, Pirate=500/510-522 (plus Cygnus 1000s, Aran 2100-2112, Evan 2200-2218 for non-v83 tenants).

A wearer with `jobId=200` (Magician) and an item with `reqJob=2` would compute `200 & 2 = 0` and fail.

**Plan introduces** `wearerClassMask(jobId job.Id) uint16` that maps internal jobId → v83 bitmask:

| Internal branch (`jobId/100`) | v83 mask |
|------|------|
| `0` (Beginner / Noblesse 1000 / Legend 2000) | `0` |
| `1, 11, 12` (Warrior, DawnWarrior, Aran*) | `1` |
| `2, 22` (Magician, Evan) | `2` |
| `3, 13` (Bowman, WindArcher) | `4` |
| `4, 14` (Thief, NightWalker) | `8` |
| `5, 15` (Pirate, ThunderBreaker) | `16` |

(*Aran sits in Warrior class branch; jobId/100 = 21 — the helper handles `21` explicitly.)

The diagnosis case (`reqLuk` only) doesn't depend on this, but the unit test for cross-class items (e.g. `reqJob = 1|2`) does.

## Useful upstream constants / types

- `stat.TypeLevel = "LEVEL"`, `stat.TypeJob = "JOB"` (`libs/atlas-constants/stat/constants.go:10-11`).
- `stat.Type` is a string type — `Updates []stat.Type` JSON-decodes from string array directly.
- `job.Id` is `uint16` (`libs/atlas-constants/job/constants.go:7`).
- `tenant.MustFromContext(ctx)` returns the tenant model; `t.Id()` returns a UUID-like `tenant.Id`.
- Existing `requests.RootUrl("DATA")` resolves the atlas-data base URL; tenant header propagation is automatic.
- `MaxHpMpCap uint32 = 30000` clamp lives in `character/model.go`. New compute path preserves it.

## Test scaffolding pattern

Existing tests use `setupProcessorTest(t)` which boots `miniredis` and a fresh `Registry`. Reuse the helper. Tenant context is created via `tenant.Create(uuid.New(), "GMS", 83, 1)` and `tenant.WithContext(ctx, t)`.

## PRD §4.1 reproduction parameters (for integration tests)

- characterId: arbitrary (use a fixed test value, e.g. `12345`).
- assetId: arbitrary (e.g. `42`).
- templateId: `1052095` (an overall in v83).
- assetRest: `Slot=-5, Mp=50` (only MP populated; rest zero).
- equipmentReqs: `ReqLuk=40` (rest zero).
- wearer.luck=39 → asset disqualifies → MaxMp == base.MaxMp; no `equipment:<assetId>` entry in `Bonuses()`.
- wearer.luck=40 → asset qualifies → MaxMp == base.MaxMp + 50; entry present.

## Build & test commands

```bash
cd services/atlas-effective-stats/atlas.com/effective-stats
go build ./...
go test ./...
```

After all tasks:

```bash
docker build services/atlas-effective-stats   # confirms image still builds
```

## Files NOT to touch

- `libs/atlas-constants/**` — no new shared types needed.
- `services/atlas-data/**` — atlas-data already exposes the six req fields.
- `services/atlas-character/**` — already emits `STAT_CHANGED` with `TypeLevel`/`TypeJob`.
- `services/atlas-channel/**` — display path is correct once `Computed.maxMp` agrees with the v83 client.
- `services/atlas-ui/**` — no frontend changes (PRD §10).
