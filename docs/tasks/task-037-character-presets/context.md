# Character Presets — Execution Context

Quick reference for the agent(s) implementing `plan.md`. Read this first; for the full *why* behind each decision, see `design.md` (and `prd.md` for requirements).

---

## Goal in one paragraph

Add a sibling array `presets` to the existing `characters` configuration document in **atlas-configurations** (template + tenant scope). Each preset fully specifies a character (appearance, level, stats, meso, GM, equipment, inventory, skills). Add a new factory endpoint `POST /factory/characters/from-preset` in **atlas-character-factory** that materializes a preset into a real character by emitting the existing `CharacterCreation` saga. Equipment created from a preset must use deterministic atlas-data default stats (no variance roll); this is plumbed via a new `UseAverageStats` boolean on the saga `CreateAndEquipAssetPayload`. Surface two operator workflows in **atlas-ui**: an Apply Preset dialog on `AccountDetailPage` and a multi-step Admin Bootstrap wizard on `AccountsPage`.

## Service ownership and boundaries

| Service | Role in this feature |
|---|---|
| `services/atlas-configurations` | Owns `characters.presets` storage (JSONB on existing row), validation, seed. |
| `services/atlas-data` | New `MaxLevel` field on skill rest model; new `ids=` filter on skill search. |
| `services/atlas-character` | New `GET /characters/name-validity?name=&worldId=`; `handleCreateCharacter` sets Gm + Meso from command body. |
| `services/atlas-character-factory` | New `POST /factory/characters/from-preset`; passthrough `GET /factory/characters/name-validity`. |
| `services/atlas-saga-orchestrator` | Threads `UseAverageStats` through compartment commands; threads `Gm`/`Meso` through character-create command. |
| `services/atlas-inventory` | `asset.Create` refactored to options struct; honors `UseAverageStats`. |
| `libs/atlas-saga` | `CreateAndEquipAssetPayload.UseAverageStats`; `CharacterCreatePayload.Gm`/`Meso`. |
| `services/atlas-ui` | Two new pages, one new dialog, one new wizard, three new hooks, one new service. |

## Critical design corrections (from PRD → design)

These supersede language in `prd.md` / `data-model.md` / `api-contracts.md`. The plan implements them, not the PRD's original wording:

1. **PRD says "atlas-tenants"; reality is "atlas-configurations".** No new endpoints; the existing `PATCH /api/configurations/{tenants,templates}/:id` partial-update path carries the new `presets` field.
2. **atlas-data does not expose skill MaxLevel today.** Add `MaxLevel uint8` to the skill rest model and populate it during load.
3. **`Gm` and `Meso` are NOT plumbed through `create_character` today.** Add to shared payload, orchestrator command, atlas-character handler.
4. **`create_and_equip_asset` already auto-equips** via the saga orchestrator's asset consumer dynamically injecting an `EquipAsset` step (`saga-orchestrator/kafka/consumer/asset/consumer.go:138-209`). The plan does **not** introduce a parallel saga step.

## Cross-cutting decisions

- **D-1 / D-2:** `UseAverageStats` lives on `CreateAndEquipAssetPayload` (per-equipment-entry). Default false preserves player-creation/shop/drop behaviour. Per-preset entry defaults to true.
- **D-3:** `asset.Create` refactored to `(...slot int16, opts CreateOptions) (Model, error)` — single mechanical refactor across the package.
- **D-5:** atlas-data skill search gains `?ids=<csv>` (or repeated `id=`); when supplied, the 10-result substring cap is bypassed.
- **D-6:** atlas-character is the authority on names. `IsValidName` is widened with `worldId`. The factory's name-validity endpoint is a passthrough.
- **D-8:** Preset validation lives in `tenants/characters/preset/validator.go` (and templates mirror), owns its own atlas-data client.
- **D-9:** Configurations API stays whole-document replace per existing PATCH semantics.
- **D-10 / R-4:** Canonical 4th-job preset catalog only ships into `template_gms_83_1.json`; other region/version `template_*.json` files get `characters.presets: []`.
- **D-11 — D-13:** UI hooks reuse the existing template/tenant configuration hooks; wizard state via `useReducer`; account-materialization via `useQuery` poll + watchdog.
- **R-1:** Server generates UUIDs for missing preset ids.
- **R-2:** No factory-side tenant/account ownership pre-check.
- **R-3:** Validation errors use `meta.path = "presets[<presetId>].<field>"`.

## Key files to reference

| Concern | File |
|---|---|
| Existing `characters` rest model (tenant) | `services/atlas-configurations/atlas.com/configurations/tenants/characters/rest.go` |
| Existing `characters` rest model (template) | `services/atlas-configurations/atlas.com/configurations/templates/characters/rest.go` |
| Configurations PATCH handler (tenant) | `services/atlas-configurations/atlas.com/configurations/tenants/resource.go:64-76` |
| Configurations PATCH handler (template) | `services/atlas-configurations/atlas.com/configurations/templates/resource.go` |
| Seed data sample | `services/atlas-configurations/seed-data/templates/template_gms_83_1.json:2388` |
| Shared saga payloads | `libs/atlas-saga/payloads.go:127-130` (CreateAndEquip), `:588-610` (CharacterCreate) |
| Saga orchestrator character producer | `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/character/processor.go:208-211` |
| Saga orchestrator character producer wire | `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/character/producer.go:231-238` |
| Saga orchestrator CreateCharacterCommandBody | `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/message/character/kafka.go:177-194` |
| Saga orchestrator handler call site | `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/handler.go:1394` |
| Saga orchestrator compartment producer | `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/compartment/producer.go:15-32` |
| Saga orchestrator compartment processor | `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/compartment/processor.go:54-104` |
| Saga orchestrator CreateAssetCommandBody | `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/message/compartment/kafka.go:119` |
| Saga orchestrator forwardCharacterCreationResult | `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/processor.go:1412-1465` |
| atlas-character CreateCharacterCommandBody | `services/atlas-character/atlas.com/character/kafka/message/character/kafka.go:62-79` |
| atlas-character handleCreateCharacter | `services/atlas-character/atlas.com/character/kafka/consumer/character/consumer.go:333-374` |
| atlas-character builder Gm/Meso setters | `services/atlas-character/atlas.com/character/character/builder.go:62` (`SetGm`), `services/atlas-character/atlas.com/character/character/model.go:421` (`SetMeso`) |
| atlas-character IsValidName | `services/atlas-character/atlas.com/character/character/processor.go:196-218` |
| atlas-character resource registration | `services/atlas-character/atlas.com/character/character/resource.go` |
| atlas-data skill rest model | `services/atlas-data/atlas.com/data/skill/rest.go` |
| atlas-data skill search handler | `services/atlas-data/atlas.com/data/skill/resource.go:28-61` |
| atlas-data skill loader | `services/atlas-data/atlas.com/data/skill/processor.go` |
| atlas-character-factory entrypoint | `services/atlas-character-factory/atlas.com/character-factory/factory/processor.go:138-211` (`buildCharacterCreationSaga`) |
| atlas-character-factory routes | `services/atlas-character-factory/atlas.com/character-factory/factory/resource.go` |
| atlas-character-factory configuration client | `services/atlas-character-factory/atlas.com/character-factory/configuration/requests.go` |
| atlas-inventory asset.Create | `services/atlas-inventory/atlas.com/inventory/asset/processor.go:276-355` |
| atlas-inventory create-asset consumer | `services/atlas-inventory/atlas.com/inventory/kafka/consumer/compartment/consumer.go:207-213` |
| atlas-inventory CreateAssetCommandBody | `services/atlas-inventory/atlas.com/inventory/kafka/message/compartment/kafka.go:99` |
| atlas-inventory compartment Create call sites | `services/atlas-inventory/atlas.com/inventory/compartment/processor.go:948,1018,1030,1084,1191,1207` (~14 sites) |
| atlas-ui templates form precedent | `services/atlas-ui/src/pages/templates-character-templates-form.tsx` |
| atlas-ui tenants form precedent | `services/atlas-ui/src/pages/tenants-character-templates-form.tsx` |
| atlas-ui App.tsx routes | `services/atlas-ui/src/App.tsx` |
| atlas-ui breadcrumb registry | `services/atlas-ui/src/lib/breadcrumbs/routes.ts` |

## Build / test commands

Each service has its own `go.mod`. Per affected service:

```bash
cd services/<service>/atlas.com/<service> && go test ./... -count=1
```

For atlas-ui:

```bash
cd services/atlas-ui && npm test
```

Frontend dev server (used for browser sanity check on the wizard):

```bash
cd services/atlas-ui && npm run dev
```

When commits cross multiple services, run `go test ./... -count=1` in each affected service before declaring a task complete.

## Out-of-scope (explicit)

- Per-equipment explicit stat overrides (preset only carries `useAverageStats` flag, no per-stat numbers).
- Bulk preset import format.
- Saga `transactionId` polling in the wizard (toast + list invalidation only).
- Item / skill picker components (free-text uint32 + lookup link suffices for FR-25).
- RBAC / auth on preset application.
- Cygnus / Aran / Resistance / Legend 4th-job presets at launch (explorer 4th job only).
