# Character Presets — Design

Version: v1
Status: Draft
Created: 2026-04-30
Companion to: `prd.md`, `api-contracts.md`, `data-model.md`, `ux-flow.md`

---

## 1. Scope of this document

The PRD pins the *what*. This design pins the *how*: which service owns what, where new code lives, how data flows end-to-end, and what existing assumptions had to be revised. Everything below is the result of a brainstorming pass against the PRD plus targeted code reads; references to file paths and line numbers reflect the state of `main` at the time of writing.

PRD discrepancies surfaced during exploration are listed in §2 and supersede the corresponding PRD/data-model/api-contracts text.

---

## 2. PRD discrepancies (corrections)

### 2.1 The owning service is `atlas-configurations`, not `atlas-tenants`.

The PRD repeatedly refers to "atlas-tenants `characters` configuration resource", a `configurations` table row keyed `resource_name="characters"`, and endpoints like `PUT /configurations/characters`. None of that exists.

- `atlas-tenants` only manages `routes`, `vessels`, `instance-routes` (per-resource rows in its `configurations` table). It has **no** reference to `characters` anywhere.
- `atlas-configurations` (separate service, own repo path `services/atlas-configurations/atlas.com/configurations/`) owns `/api/configurations/tenants/:id` and `/api/configurations/templates/:id`. The tenant/template configuration is **one JSON document** with `characters`, `npcs`, `socket`, `worlds`, `cashShop` as top-level attributes — not row-per-resource.
- The existing `characters.templates` model is at `services/atlas-configurations/atlas.com/configurations/{tenants,templates}/characters/rest.go`.
- Atlas-ui already uses this: `tenantsService.updateTenantConfiguration()` does `PATCH /api/configurations/tenants/:id` with a partial-attributes body, merging client-side first.

**Effect on the PRD:** every "atlas-tenants" reference in §4.1 / §5 / §7 / data-model / api-contracts is retargeted to `atlas-configurations`. The `presets` array is added as a sibling field on the existing `characters` sub-document at `services/atlas-configurations/atlas.com/configurations/{tenants,templates}/characters/`. There is no `PUT /configurations/characters` endpoint and there shouldn't be one — that fights the whole-document partial-update model already in production. The endpoints stay `PATCH /api/configurations/{tenants,templates}/:id` with the body now also accepting `characters.presets`.

### 2.2 atlas-data does not expose skill master/max level.

PRD FR-11 plans for the factory to derive `MasterLevel` from atlas-data's existing skill information endpoint. The current skill REST model (`services/atlas-data/atlas.com/data/skill/rest.go:8-16`) exposes `Id, Name, Description, Action, Element, AnimationTime, Effects[]` — no max-level field.

**Decision:** extend atlas-data's skill REST model with an explicit `MaxLevel uint8` field (sourced from the WZ skill data the loader already ingests, where the count of per-level entries gives the max). This is a small, well-bounded atlas-data change and beats relying on an undocumented `len(Effects)` convention.

### 2.3 Gm and Meso are NOT plumbed through `create_character` today.

The PRD §7 claims "atlas-character: No code changes required; existing `Gm` and `Meso` builder methods are already plumbed by the `create_character` saga step." That is incorrect.

- `libs/atlas-saga/payloads.go` `CharacterCreatePayload` has **no** `Gm` or `Meso` fields (verified by grep — only AwardMesosPayload, RequestChangeMesoCommand, etc. exist as separate post-creation actions).
- The orchestrator's `RequestCreateCharacter` (`services/atlas-saga-orchestrator/.../character/processor.go:208-211`) does not pass them.
- atlas-character's `handleCreateCharacter` consumer (`services/atlas-character/atlas.com/character/kafka/consumer/character/consumer.go:333-373`) never calls `SetGm` and does not set initial meso.

**Effect on the design:** both `Gm` and `Meso` are extended onto the existing chain (`CharacterCreatePayload` → orchestrator's `CreateCharacterCommandBody` and `RequestCreateCharacter` → atlas-character's `handleCreateCharacter` consumer):

- `saga.CharacterCreatePayload` gains `Gm int \`json:"gm,omitempty"\`` and `Meso uint32 \`json:"meso,omitempty"\``.
- The orchestrator's `CreateCharacterCommandBody` (`services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/message/character/kafka.go:177`) gains the same. `RequestCreateCharacter` (`character/processor.go:208-211`) and `RequestCreateCharacterProvider` (`character/producer.go:231-238`) propagate.
- atlas-character's `handleCreateCharacter` consumer (`character/kafka/consumer/character/consumer.go:333-373`) calls `SetGm(c.Body.Gm)` and `SetMeso(c.Body.Meso)` on the builder. atlas-character's `Model` builder already has both setters (`character/model.go:421` for `SetMeso`; `character/builder.go:62` for `SetGm`).

This places both fields where they belong — atomic with character creation, on the same row — and avoids dragging the existing `AwardMesosPayload` (which requires `WorldId`/`ChannelId`/`ActorId`/`ActorType` for an online-character wallet flow) into an offline post-creation context where it doesn't fit.

`omitempty` on both fields keeps existing emitters (player-creation flow, change-job, etc.) wire-compatible — they pass the zero value and behaviour is unchanged.

### 2.4 Existing `create_and_equip_asset` already auto-equips.

A preliminary read suggested `RequestCreateAndEquipAsset` (`services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/compartment/processor.go:102-104`) only put items in the inventory tab and never equipped them. The full trace shows this is wrong.

The actual flow:

1. `create_and_equip_asset` step → atlas-inventory creates the asset in `NextFreeSlot()` (positive equip-tab slot) → emits `StatusEventTypeCreated`.
2. The saga orchestrator's asset consumer (`services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/consumer/asset/consumer.go:138-209`) sees the CREATED event for a `CreateAndEquipAsset` step and **dynamically injects an `EquipAsset` step** via `sagaProcessor.AddStepAfterCurrent(...)` with `Source = e.Slot`, `Destination = -1` as a hint.
3. The injected step runs → atlas-inventory's `EquipItem` (`services/atlas-inventory/atlas.com/inventory/compartment/processor.go:237`) → `equipmentProcessor.DestinationSlotProvider(destination)(templateId)` resolves the actual equip slot from atlas-data and moves the asset there.

**Effect on this design:** preset equipment uses the existing `create_and_equip_asset` saga step unchanged. No new saga step, no slot threading, no `Destination` field on the create command body.

---

## 3. Cross-cutting decisions

The following decisions apply across services and are referenced from §4 onward.

| # | Decision | Rationale |
|---|----------|-----------|
| D-1 | Determinism via a payload flag, not a separate path. `CreateAndEquipAssetPayload` gains `UseAverageStats bool` (omitempty); when true, atlas-inventory writes atlas-data defaults verbatim instead of `getRandomStat`. Default false preserves existing player-creation/shop/drop behaviour. | Smallest change that delivers the PRD's "predictable test rig" goal without a parallel command type. (PRD/Q3) |
| D-2 | Per-equipment-entry `useAverageStats` toggle, defaulting to true on preset equipment entries. Preset author can opt back into variance for any single item. | Flexibility is free in the data model. (PRD OQ-4 / Q4) |
| D-3 | `asset.Create` in atlas-inventory (`services/atlas-inventory/atlas.com/inventory/asset/processor.go:276`) refactored to take an options struct. The signature is already 10 positional parameters and a TODO at line 293 wants to add an 11th flag (`UNTRADEABLE`). The refactor is in scope for this task. | The next flag becomes a one-line addition; mechanical caller updates only (single package). (Q4) |
| D-4 | Equipment stat variance is migration-target. Add a TODO to migrate atlas-npc-shops and the player-creation flow to set `UseAverageStats=true` once preset application is shipped. Game mechanics expect both paths to be deterministic; variance was historically applied indiscriminately. | Avoid orphaning the new flag; document the future cleanup. (User, this round) |
| D-5 | Master-level derivation runs at apply time via a batch atlas-data fetch. atlas-data's existing `GET /data/skills?name=` handler (`services/atlas-data/atlas.com/data/skill/resource.go:28-61`) is extended with an `ids=` filter (or `id=` repeated). With `ids` supplied, the 10-result cap is lifted; the response includes the new `MaxLevel` field per skill. | One round-trip beats N parallel calls; no new endpoint required. (Q5) |
| D-6 | atlas-character exposes `GET /characters/name-validity?name=&worldId=` as a thin HTTP wrapper around its existing internal `IsValidName` (`character/processor.go:196-218`), with the uniqueness check broadened from tenant-scoped to (tenant, world)-scoped. atlas-character-factory's new `GET /factory/characters/name-validity` is a passthrough. | atlas-character is the authority on names. (Q6) |
| D-7 | `POST /factory/characters/from-preset` lives as a new method on the existing factory processor (`services/atlas-character-factory/atlas.com/character-factory/factory/processor.go`), reusing the existing `buildCharacterCreationSaga` helper via a new `buildPresetCharacterCreationSaga` neighbor. No new processor file, no new package. | The saga emit is the natural seam; one new entry point doesn't justify a new processor. (Q7) |
| D-8 | atlas-configurations preset validation lives in a dedicated `tenants/characters/preset/validator.go` package (and its `templates/characters/preset/validator.go` mirror at template scope). The validator owns the atlas-data client (new). Handler calls `validator.ValidatePresets(ctx, presets) []ValidationError`. | Validation is non-trivial (~12 rules including atlas-data lookups) and warrants its own seam. atlas-configurations doesn't have an atlas-data client today; the validator is the only caller. (Q8) |
| D-9 | The configurations API stays whole-document replace per the existing PATCH semantics. The PRD's "PUT may omit `presets` to leave the existing array unchanged" is dropped; the UI already sends the full attributes block. | Existing pattern; no nullability dance. (Q9) |
| D-10 | The canonical 4th-job preset catalog ships into the **active dev region/version file only** (currently GMS v83, per `.bruno/MapleStory Dev/environments/GMS v83.yml`). Other region/version `template_*.json` files get `characters.presets: []` and are populated by their content engineers when they're ready. | Content (skill ids, item ids, cosmetics) diverges by region/version; templates already follow this pattern. (Q10) |
| D-11 | UI hooks: presets pages reuse `useTenantConfiguration` / `useTemplate` directly (mirroring `tenants-character-templates-form.tsx`), spreading `{...characters, presets: edited}` before mutating. New hooks are added only for genuinely new endpoints: `useCreateCharacterFromPreset` mutation, `useNameValidity` debounced query, `useAccountByName(name, {pollUntilFound, timeoutMs})`. | The templates form is the precedent and works. (Q11) |
| D-12 | Admin Bootstrap wizard state lives in a single `useReducer` inside the wizard component. Per-step sub-forms use `react-hook-form` + Zod locally where they make sense (step 1 credentials, step 3 name table). | Wizards are inherently stateful; one source of truth. (Q12) |
| D-13 | The 30s account-materialization poll is wrapped as `useAccountByName(name, { pollUntilFound: true, timeoutMs: 30000 })`, internally a React Query `useQuery` with `refetchInterval: 1000` and an external timeout watchdog. | Idiomatic, automatic cleanup, intent-revealing name. (Q13) |

PRD open questions OQ-1, OQ-2, OQ-3, OQ-5 are confirmed as the PRD's proposed defaults (no server-side resume; no revert-to-template-default; client-side dedupe of `defaultName` collisions; no GM-flag gating). See PRD §9.

---

## 4. Architecture by service

Diagram of the call graph for a single preset application:

```
atlas-ui                         atlas-character-factory                            atlas-configurations           atlas-data         atlas-character     atlas-saga-orchestrator                          atlas-inventory
  │                                       │                                                  │                          │                  │                          │                                            │
  ├── POST /factory/characters/from-preset ─►                                                 │                          │                  │                          │                                            │
  │                                       ├── GET /api/configurations/tenants/:id ──────────►│                          │                  │                          │                                            │
  │                                       │◄── characters.presets[i] ─────────────────────────                          │                  │                          │                                            │
  │                                       ├── GET /data/skills?ids=… ───────────────────────────────────────────────────►                  │                          │                                            │
  │                                       │◄── [{id,maxLevel}, …] ──────────────────────────────────────────────────────                   │                          │                                            │
  │                                       ├── GET /data/items/:id (per equip+inventory) ─────────────────────────────────►                  │                          │                                            │
  │                                       │◄── item info (validate slots, ids) ─────────────────────────────────────────                   │                          │                                            │
  │                                       ├── GET /characters/name-validity?name=&worldId= ──────────────────────────────────────────────►│                          │                                            │
  │                                       │◄── {valid,reason?} ──────────────────────────────────────────────────────────────────────────  │                          │                                            │
  │                                       ├── emit saga (CharacterCreation) ─────────────────────────────────────────────────────────────────────────────────────────►│                                            │
  │◄── 202 {transactionId} ───────────────                                                                                                                              │                                            │
  │                                                                                                                                                                    ├── create_character ───────────────────────►(atlas-character)
  │                                                                                                                                                                    ├── award_asset_… (per inventory) ──────────►│
  │                                                                                                                                                                    ├── create_and_equip_asset_… (per equip) ──►│
  │                                                                                                                                                                    │     ▲ asset-CREATED → auto-inject equip ─────│
  │                                                                                                                                                                    ├── create_skill_… (per skill) ─────────────►(atlas-character)
```

### 4.1 atlas-configurations

**Files added:**

- `services/atlas-configurations/atlas.com/configurations/{tenants,templates}/characters/preset/`
  - `rest.go` — `RestModel` for a single preset matching the data-model.md shape; `Transform`/`Extract` between domain model and JSON.
  - `model.go` — immutable `Preset` model with accessors.
  - `builder.go` — fluent builder enforcing invariants (FR-4 ranges, no duplicate equip slots).
  - `validator.go` — `ValidatePresets(ctx context.Context, dataClient SkillItemClient, presets []RestModel) []ValidationError`. Owns atlas-data lookups for templateId/skillId resolution and equip-slot derivation (used for the no-duplicate-slot rule).
  - `validator_test.go` — table-driven cases per rule.

**Files modified:**

- `tenants/characters/rest.go` (and the templates mirror): add `Presets []preset.RestModel \`json:"presets"\``. Existing `Templates` field unchanged.
- `tenants/rest.go` and `templates/rest.go`: no change — `Characters` field already wraps the sub-document.
- `tenants/resource.go` PATCH handler: assign UUIDs to any preset entries with empty `id` (per R-1) and then invoke `preset.ValidatePresets(...)`. Map returned errors to JSON:API `errors[]` with `meta.path = "presets[<presetId>].<field>"` (per R-3).
- `templates/resource.go`: same treatment for the template-scoped PATCH.

**New atlas-data client:**

- `services/atlas-configurations/atlas.com/configurations/data/` (new package)
  - `requests.go` — `GetItemById(uint32) (item.RestModel, error)`, `GetSkillByIds([]uint32) ([]skill.RestModel, error)` using the shared `atlas-rest` client.
  - `mock/processor.go` — for validator tests.

**Validation rules implemented (per data-model.md §"Validation summary"):**

- `id` valid UUID or absent (validator generates one via `uuid.New()` per R-1).
- `name` 1..64; `description` ≤ 512.
- `tags` element type string; no length cap.
- `jobId` resolves via `libs/atlas-constants/job` (existing package, no atlas-data round-trip).
- `gender ∈ {0,1}`; `level ∈ [1,250]`.
- `face`, `hair`, `hairColor`, `skinColor` non-negative.
- `equipment[*].templateId` exists in atlas-data (per-id GET); equipment slots unique within the preset (atlas-data lookup of each equip's natural slot).
- `inventory[*].templateId` exists in atlas-data; `quantity ≥ 1`.
- `skills[*].skillId` exists in atlas-data (batched via `ids=`); `level ≥ 1` and `level ≤ MaxLevel` from atlas-data.

**Seeding:**

The seed loader currently reads region/version JSON files at `services/atlas-configurations/seed-data/templates/template_*.json` at template-creation time. Per D-10:

- `template_gms_83_1.json` gains the canonical 4th-job explorer preset list inside `characters.presets`.
- Other `template_*.json` files gain `characters.presets: []` (idempotent — empty array won't be overwritten on re-seed per FR-28).
- The seed merge is the existing one — no code change to the seeder, only data edits.

**Kafka events:**

The existing tenants/templates configuration update events emit when the document is patched. The new `presets` field rides inside the existing payload. No new event topic, no new event type.

### 4.2 atlas-data

**Files modified:**

- `services/atlas-data/atlas.com/data/skill/rest.go` — add `MaxLevel uint8 \`json:"maxLevel"\`` to `RestModel`.
- `services/atlas-data/atlas.com/data/skill/processor.go` (or wherever the skill model is built from WZ data) — populate `MaxLevel` from the count of per-level effect entries during load.
- `services/atlas-data/atlas.com/data/skill/resource.go` `handleSearchSkillsRequest` — accept an `ids` query parameter (comma-separated `uint32` list, or repeated `id=`). When `ids` is supplied:
  - Bypass the `name=` filter.
  - Filter skills by id membership.
  - Return all matches (no 10-result cap; the cap exists to bound substring searches, not explicit id lists).
- `services/atlas-data/atlas.com/data/skill/rest_test.go` — extend cases for `ids=` and for the `MaxLevel` field.

**Compatibility:** existing callers using `name=` see the new `MaxLevel` field but can ignore it. Existing JSON consumers continue to decode (Go zero-value when absent on inputs).

### 4.3 atlas-character

**Files added:**

- `services/atlas-character/atlas.com/character/character/name_validity_resource.go` (new) — handler for `GET /characters/name-validity`.

**Files modified:**

- `services/atlas-character/atlas.com/character/character/resource.go` — register the new route.
- `services/atlas-character/atlas.com/character/character/processor.go` — overload (or rename) `IsValidName` to take a `worldId` parameter. The existing internal callers (`Update` line 1626, `CreateAndEmit` flow line 236) keep tenant-scoped behaviour; the new endpoint passes the supplied `worldId` and the processor adds a world filter to the `GetForName` lookup. Returns a structured reason: `regex` / `length` / `blocked` / `duplicate` so the factory passthrough can render JSON:API meaningfully.
- `services/atlas-character/atlas.com/character/kafka/consumer/character/consumer.go:333-373` (`handleCreateCharacter`) — call `SetGm(c.Body.Gm)` and `SetMeso(c.Body.Meso)` on the builder per §2.3. Existing zero-value defaults preserve current behaviour for player-creation traffic.

**REST shape:**

```jsonc
GET /characters/name-validity?name=AdminHero&worldId=0
200 OK
{ "valid": false, "reason": "duplicate", "detail": "Name already taken in world 0." }
```

`reason` is one of `regex`, `length`, `blocked` (currently always false; the existing TODO at processor.go:210-214 about blocked-name lookups stays a TODO), `duplicate`. `400 Bad Request` for missing/malformed parameters.

**README + ingress:** routes.conf already has `atlas-character`; no change. Service README's REST table gains the new endpoint.

### 4.4 atlas-character-factory

**Files added:**

- `services/atlas-character-factory/atlas.com/character-factory/factory/preset_rest.go` — `PresetCreateRestModel { presetId string; accountId uint32; worldId byte; name string }` plus JSON:API helpers.
- `services/atlas-character-factory/atlas.com/character-factory/data/skill_requests.go` — `GetSkillsByIds(ids []uint32) ([]skill.RestModel, error)` against atlas-data's extended `GET /data/skills?ids=...`.
- `services/atlas-character-factory/atlas.com/character-factory/data/item_requests.go` — `GetItemById(id uint32) (item.RestModel, error)` for equip-slot derivation and item-id validation at apply time.
- `services/atlas-character-factory/atlas.com/character-factory/configurations/preset_requests.go` — `GetPresetById(presetId uuid.UUID) (preset.RestModel, error)` against `GET /api/configurations/tenants/:id` (filtering the returned `characters.presets` array client-side; no new atlas-configurations endpoint).
- `services/atlas-character-factory/atlas.com/character-factory/character/name_validity_requests.go` — `CheckNameValidity(name string, worldId byte) (NameValidityResponse, error)` against atlas-character's new endpoint.

**Files modified:**

- `factory/processor.go` — new method `CreateFromPreset(input PresetCreateRestModel) (transactionId string, err error)`:
  1. Resolve preset via `configurations.GetPresetById(...)`. 404 if missing. (No factory-side tenant-account ownership check per R-2; cross-tenant misuse is caught by atlas-account's own tenant scoping when the saga runs.)
  2. Pre-emptively call `character.CheckNameValidity(name, worldId)`. Map non-`valid` to a 400 (invalid-name) or 409 (duplicate) before saga emission.
  3. Validate equipment/inventory/skill ids against atlas-data (apply-time re-validation per data-model.md). Any miss → 400 with `meta.path = "presets[<presetId>].<field>"` per R-3.
  4. Batch-fetch skill `MaxLevel`s via `data.GetSkillsByIds(...)`. Hard-fail with 502 if atlas-data is unreachable or any id misses (reject the PRD's "fallback to `level` with logged warning" — that produces silently-wrong characters).
  5. Build the saga via a new private helper `buildPresetCharacterCreationSaga(transactionId, preset, target)` mirroring `buildCharacterCreationSaga` but consuming the preset's full attribute block. Emit.
  6. Return `transactionId`.
- `factory/resource.go` — register `POST /factory/characters/from-preset` and `GET /factory/characters/name-validity`. The latter is a passthrough that calls `character.CheckNameValidity`.

**Saga build details (`buildPresetCharacterCreationSaga`):**

| Step | Action | Payload | Notes |
|------|--------|---------|-------|
| 1 | `create_character` | `CharacterCreatePayload` from preset | Sets `Name` from request, `JobId` directly (not `JobFromIndex`), `Hair = preset.Hair + preset.HairColor` matching existing convention, `Top/Bottom/Shoes/Weapon = 0` (preset gear handled by create_and_equip_asset), `Gm = preset.gm` and `Meso = preset.meso` (per §2.3 fix), `Hp = preset.Stats.Hp`, `Mp = preset.Stats.Mp`. |
| 2..N | `award_asset_<i>` | `AwardItemActionPayload{CharacterId: 0, Item{TemplateId, Quantity}}` | One per `preset.inventory[i]`. `CharacterId=0` sentinel forwarded by orchestrator (`saga/processor.go:1417-1465`). |
| N+1..N+M | `create_and_equip_asset_<i>` | `CreateAndEquipAssetPayload{CharacterId: 0, Item{TemplateId, Quantity:1}, UseAverageStats: preset.equipment[i].useAverageStats}` | Auto-equip is handled by the saga orchestrator's asset consumer (§2.4). |
| N+M+1..end | `create_skill_<i>` | `CreateSkillPayload{CharacterId: 0, SkillId, Level, MasterLevel, Expiration: zero}` | `MasterLevel` populated from the batch atlas-data fetch in step 4. |

The factory's existing saga builder already supports `MasterLevel` (`payloads.go:225-231`) — no shared-lib change.

**`CharacterCreatePayload.{Top,Bottom,Shoes,Weapon} = 0`** — preset paths bypass the existing player-creation starter-gear plumbing and use uniform `create_and_equip_asset` for all equipment. Setting these to 0 is safe because the orchestrator's `RequestCreateCharacter` (`services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/character/processor.go:208-211`) does not pass them through to atlas-character anyway — they're computed locally by the existing factory only to build the create_and_equip_asset steps.

**Saga-orchestrator sentinel:** the existing `forwardCharacterCreationResult` at `saga/processor.go:1436-1464` already type-switches on `AwardItemActionPayload`, `CreateAndEquipAssetPayload`, `CreateSkillPayload`. No orchestrator change.

### 4.5 atlas-saga (shared lib)

**Files modified:**

- `libs/atlas-saga/payloads.go:127-130` — `CreateAndEquipAssetPayload` gains `UseAverageStats bool \`json:"useAverageStats,omitempty"\``.
- `libs/atlas-saga/payloads.go` `CharacterCreatePayload` (around line 589-610) — gain `Gm int \`json:"gm,omitempty"\`` and `Meso uint32 \`json:"meso,omitempty"\`` per §2.3. `omitempty` keeps existing player-creation emitters wire-compatible.
- `libs/atlas-saga/unmarshal.go:96-101` (the `CreateAndEquipAsset` arm of the saga step `UnmarshalJSON`) — no code change, but verified that the new field is decoded automatically. Same for the `CreateCharacter` arm.

**Backwards compatibility:** existing in-flight saga rows in PostgreSQL JSON columns decode to the same struct with `UseAverageStats == false`, `Gm == 0`, `Meso == 0`. `omitempty` keeps existing emitters wire-compatible. Verified by inspecting `payloads.go` and `unmarshal.go`.

### 4.6 atlas-saga-orchestrator

**Files modified:**

- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/compartment/producer.go:15-32` (`RequestCreateAssetCommandProvider`) — propagate the new `useAverageStats` flag from the saga payload into `CreateAssetCommandBody`. The provider gains a parameter; the caller chain is `RequestCreateAndEquipAsset` (line 102) → `RequestCreateItem` (line 54). `RequestCreateItem` is a generic "create an item" path used by both the saga step and other internal callers, so the cleanest split is:
  - Add `RequestCreateItemWithStats(transactionId, characterId, templateId, quantity, expiration, useAverageStats)`.
  - Have the existing `RequestCreateItem` keep its current signature and forward to the new method with `useAverageStats=false`.
  - Have `RequestCreateAndEquipAsset` (line 102) pass `payload.UseAverageStats` through.
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/message/compartment/kafka.go:99` (`CreateAssetCommandBody`) — gain `UseAverageStats bool \`json:"useAverageStats,omitempty"\``.
- Mock processor in `compartment/mock/processor.go` — update `RequestCreateAndEquipAssetFunc` test fixture sites to keep behaviour.
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/character/processor.go:208-211` (`RequestCreateCharacter`) and `kafka/message/character/kafka.go:177` (`CreateCharacterCommandBody`) — both gain `Gm int` and `Meso uint32` per §2.3. The producer (`character/producer.go:231-238`) propagates the fields. (`forwardCharacterCreationResult` at `saga/processor.go:1436-1464` already covers all the saga step types used by the preset saga: `AwardItemActionPayload`, `CreateAndEquipAssetPayload`, `CreateSkillPayload`. No change needed since meso is now part of the create_character payload.)

### 4.7 atlas-inventory

**Files modified:**

- `services/atlas-inventory/atlas.com/inventory/kafka/message/compartment/kafka.go:99` (`CreateAssetCommandBody`) — gain `UseAverageStats bool \`json:"useAverageStats,omitempty"\`` (mirror of the orchestrator-side schema).
- `services/atlas-inventory/atlas.com/inventory/kafka/consumer/compartment/consumer.go:207-213` (`handleCreateAssetCommand`) — propagate `c.Body.UseAverageStats` into `compartment.CreateAssetAndEmit(...)`.
- `services/atlas-inventory/atlas.com/inventory/compartment/processor.go:42-43, 63-65, 980-1051` — `CreateAsset`, `CreateAssetAndLock`, `CreateAssetAndEmit` get the new flag in their signatures (per D-3 they are converted to options-struct form; see below).
- `services/atlas-inventory/atlas.com/inventory/asset/processor.go:276-355` (`Create`) — D-3 refactor:

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
          // ... unchanged setup ...
          switch inventoryType {
          case inventory.TypeValueEquip:
              ea, err := p.statProcessor.GetById(templateId)
              if err != nil { /* unchanged */ }
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
                      // ... existing variance logic ...
                      SetSlots(ea.Slots())
              }
          // ... unchanged Use/Setup/ETC/Cash branches ...
          }
          // ... unchanged persist + emit ...
      }
  }
  ```

  Caller updates inside the package are mechanical; tests in `compartment/processor_test.go` (~14 sites) construct `CreateOptions{Quantity: ..., Expiration: ...}` instead of passing positional args.
- `services/atlas-inventory/atlas.com/inventory/compartment/processor.go:1014, 1018, 1026, 1030, 1084, 1191, 1207, …` — every internal call to `assetProcessor.Create(mb)(...)` updated to construct a `CreateOptions{...}`. `UseAverageStats: false` everywhere except the create-asset-command consumer path which forwards it from the command body.

### 4.8 atlas-ui

**Routes added in `App.tsx`:**

- `/templates/:id/character/presets` → `<TemplatesCharacterPresetsPage />`
- `/tenants/:id/character/presets` → `<TenantsCharacterPresetsPage />`

**Pages added (`services/atlas-ui/src/pages/`):**

- `templates-character-presets-form.tsx` — list/edit/delete presets at template scope. Uses `useTemplate(id)` + `useUpdateTemplate()`. Form spreads `{...template.attributes.characters, presets: editedPresets}` before mutating.
- `tenants-character-presets-form.tsx` — same at tenant scope, against `useTenantConfiguration` / `useUpdateTenantConfiguration`.
- The form structure is per ux-flow.md §C: top "Add preset" button; below, an accordion of preset cards each with identity / character / stats / equipment / inventory / skills sections; bottom Save.
- Equipment/inventory/skill rows are uint32 free-text inputs with optional "lookup" links to the existing item/skill detail pages. Building a new searchable picker is out of scope; FR-25 is satisfied by free-text + link.

**Hooks added (`services/atlas-ui/src/lib/hooks/api/`):**

- `useCharacterFromPresetMutation.ts` — `useCreateCharacterFromPreset()` calling `factoryService.createFromPreset({presetId, accountId, worldId, name})`. On success, invalidates `characterKeys.list(tenant)`.
- `useNameValidity.ts` — `useNameValidity(name, worldId, {enabled, debounceMs})`. Internally `useQuery({queryKey: ["name-validity", worldId, name], queryFn: () => factoryService.checkNameValidity(name, worldId), enabled: enabled && name.length >= 3, staleTime: 0})`. Debounce applied at the consumer level via `useDebouncedValue(name, debounceMs)`.
- `useAccountByName.ts` — `useAccountByName(name, {pollUntilFound, timeoutMs})`. Internally a `useQuery` with `refetchInterval: 1000` while `enabled && !found && !timedOut`; a `useEffect` watchdog flips `timedOut = true` after `timeoutMs`.

**Service modules added/extended (`services/atlas-ui/src/services/api/`):**

- `factory.service.ts` (new) — `createFromPreset(payload)`, `checkNameValidity(name, worldId)`.
- `accounts.service.ts` (existing) — extend `getAll` to support `?name=` filter if not already plumbed.

**Apply Preset dialog** (atlas-ui §B in ux-flow.md):

- Component: `services/atlas-ui/src/components/features/characters/ApplyPresetDialog.tsx`.
- Triggered from `AccountDetailPage` header; visible only when `useTenantConfiguration(activeTenantId).characters.presets.length > 0` (FR-23).
- Local form state via `react-hook-form` + Zod schema `{presetId, worldId, name}`. Name field bound to `useNameValidity` for live feedback. Submit calls `useCreateCharacterFromPreset`.

**Admin Bootstrap wizard** (atlas-ui §A in ux-flow.md):

- Component: `services/atlas-ui/src/components/features/accounts/AdminBootstrapWizard.tsx`.
- State: `useReducer<WizardState, WizardAction>` per D-12.
  - `WizardState`: `{ step, account: {name, password}, worldId, tagFilter[], selected: Map<presetId, {name, validity, applyStatus}>, error? }`.
  - `WizardAction`: `SET_ACCOUNT`, `SET_WORLD`, `SET_TAG_FILTER`, `TOGGLE_PRESET`, `SET_NAME`, `SET_VALIDITY`, `START_APPLY`, `SET_ROW_STATUS`, `RETRY_ROW`, `RESET`.
- Step 4 apply runs sequentially per row:
  ```ts
  for (const row of selectedRows) {
    dispatch({type: 'SET_ROW_STATUS', presetId: row.presetId, status: 'applying'});
    try {
      await createFromPreset({presetId: row.presetId, accountId, worldId, name: row.name});
      dispatch({type: 'SET_ROW_STATUS', presetId: row.presetId, status: 'success'});
    } catch (e) {
      dispatch({type: 'SET_ROW_STATUS', presetId: row.presetId, status: 'failed', error: e.message});
    }
  }
  ```
  No abort/cancel mid-loop per ux-flow.md cancellation semantics.
- Account-materialization wait between Step-1 submit and Step-4 starts uses `useAccountByName(account.name, {pollUntilFound: true, timeoutMs: 30000})`.

**Breadcrumb registry** (`services/atlas-ui/src/lib/breadcrumbs/routes.ts`):

- Add entries for `/templates/:id/character/presets` and `/tenants/:id/character/presets` mirroring the existing template/tenant character/templates entries.

**Sidebar / TemplateDetailPage / TenantDetailPage:**

- Each gains a "Character Presets" entry next to the existing "Character Templates" link, per FR-26.

---

## 5. End-to-end flows

### 5.1 Apply Preset to existing account (FR-21..23)

1. Operator opens `AccountDetailPage`.
2. Header action **Add character from preset** opens `<ApplyPresetDialog>`. Render guard: `useTenantConfiguration(activeTenant.id).data?.attributes.characters.presets?.length > 0`.
3. Operator selects preset, world, types name. Each keystroke debounced (300ms) into `useNameValidity` against `GET /factory/characters/name-validity`. Submit button enabled iff `validity.valid && name.length >= 3`.
4. Submit → `useCreateCharacterFromPreset({presetId, accountId, worldId, name})` → factory's `POST /factory/characters/from-preset`.
5. On 202: dialog closes, toast "Creating character… this may take a moment.", `queryClient.invalidateQueries(characterKeys.list(activeTenant))`.
6. On 4xx: dialog stays open with inline error from JSON:API `errors[]`.

### 5.2 Admin Bootstrap wizard (FR-18..20)

(Full sequence in ux-flow.md §B; this section captures the engineering invariants only.)

- Step transitions are state-machine-driven; "Next" disabled until the local validation invariants hold.
- Step 4's pipeline tolerates per-row failure but not transport failure for `POST /accounts/`. If account creation fails (transport or non-202 response), the wizard surfaces the error and offers Retry without invalidating prior state.
- Account materialization uses the new `useAccountByName` poll. Failure (timeout) is recoverable via Retry.
- Preset apply is sequential by design (PRD §8): the wizard awaits each `POST /factory/characters/from-preset` 202 before starting the next, so per-row UI status reflects synchronous transport result. Saga-time compensation is invisible to the wizard; the `transactionId` polling mentioned in PRD §B step 4 stays out of scope.
- Cancellation in steps 1-3 is free; in step 4 the operator can close the wizard but in-flight rows continue server-side. Re-entry of the wizard with the missing presets re-applies them.

### 5.3 Saga compensation

The saga step list for preset application is a strict superset of the existing player-creation step list (more `award_asset` and `create_and_equip_asset` and `create_skill` steps). Compensations (`Saga.AddCompensation`) are inherited unchanged — atlas-saga-orchestrator's existing compensation surface for these step types covers preset application. No new compensation actions introduced (FR-13).

A failure mid-saga (e.g. equipment templateId rejected by atlas-inventory) triggers compensation across the inventory items already created, the equipment already equipped, and any skills already added, and finally the character itself. PRD §10 acceptance criterion 6 is satisfied by this inheritance plus a new test in atlas-saga-orchestrator's compensation integration tests covering a preset-driven failure.

---

## 6. Testing strategy

| Layer | What is tested | Where |
|-------|----------------|-------|
| Validator | All FR-4 / data-model rules (jobId, gender, level, slot uniqueness, atlas-data id resolution, MaxLevel ceiling). Table-driven. Mock atlas-data client. | `configurations/.../characters/preset/validator_test.go` |
| atlas-data skill ids filter | `?ids=…` returns the correct subset; cap is lifted; `MaxLevel` populated. | `data/skill/resource_test.go`, `data/skill/rest_test.go` |
| atlas-character name-validity | regex / length / duplicate (per-world) cases; missing parameters → 400. | new `name_validity_resource_test.go` |
| Factory `CreateFromPreset` | preset resolution / not-found, name validity short-circuit, atlas-data validation, saga emission shape. Mock atlas-configurations / atlas-data / atlas-character / saga-orchestrator clients. | `factory/processor_test.go` (extended) |
| Saga orchestrator | `RequestCreateItemWithStats` / `RequestCreateAndEquipAsset` propagate `UseAverageStats`. Existing sentinel forwarding still works with the new payload field. | `compartment/processor_test.go`, `compartment/producer_test.go` |
| atlas-inventory asset processor | `useAverageStats=true` writes atlas-data defaults; `useAverageStats=false` retains variance. Options-struct refactor covered in existing tests. | `asset/processor_test.go` (new cases), `compartment/processor_test.go` (refactor adjustments) |
| atlas-ui | Form validation, mutation hooks, wizard reducer transitions, account-poll hook timeout/success, name-validity debounce. Vitest + React Testing Library. | `pages/__tests__/templates-character-presets-form.test.tsx`, `lib/hooks/api/__tests__/useCharacterFromPresetMutation.test.tsx`, `lib/hooks/api/__tests__/useNameValidity.test.tsx`, `lib/hooks/api/__tests__/useAccountByName.test.tsx`, `components/features/accounts/__tests__/AdminBootstrapWizard.test.tsx` |
| Saga compensation | Preset-driven failure rolls back through every successfully-applied step. | new integration test in `services/atlas-saga-orchestrator/.../saga/` |

Build verification per backend-dev-guidelines: `go test ./... -count=1` must pass per affected service.

---

## 7. Out-of-scope follow-ups (TODO entries)

These are recorded here so they show up in `docs/TODO.md` alongside this task's deliverables:

1. **Migrate atlas-npc-shops to deterministic stats.** atlas-npc-shops emits `CreateAssetCommandBody` for every shop purchase (`services/atlas-npc-shops/atlas.com/npc/compartment/producer.go:13-19`). After this task ships, set `UseAverageStats=true` in that producer. Game mechanics expect shop-purchased equips to be deterministic, not stat-rolled (D-4).
2. **Migrate atlas-character-factory player-creation flow to deterministic stats.** The existing `buildCharacterCreationSaga` (`factory/processor.go:138-211`) emits `CreateAndEquipAssetPayload` for Top/Bottom/Shoes/Weapon. Set `UseAverageStats=true` for those steps so starter gear matches WZ defaults exactly (D-4).
3. **Variance retains for monster drops.** No migration; variance there is a deliberate game-design property.
4. **Saga `transactionId` polling for the Admin Bootstrap wizard.** PRD §B step 4 notes that saga-time compensation is invisible to the wizard. A follow-up could add a poll of saga status by `transactionId` so per-row UI flips success → failure when compensation fires. Not blocking.
5. **Item / skill picker components.** FR-25 is satisfied by free-text uint32 inputs plus "lookup" links. A reusable `<ItemPicker>` / `<SkillPicker>` would improve the editor UX but is independently scoped.
6. **Cygnus / Aran / Resistance / Legend 4th-job presets.** Launch ships explorer 4th-job only (FR-27). Other class presets are a content task.

---

## 8. Resolved during design

Items in earlier drafts of this section have been resolved during the brainstorming pass. They are captured here as binding implementation rules:

- **R-1 — Preset id generation.** The atlas-configurations PATCH handler / preset validator generates a UUID via `uuid.New()` for any incoming preset entry whose `id` is missing or empty. The mutation response carries the assigned id back. The UI may submit a fresh entry with `id: null` and trust the server. (Other Atlas resources follow the same convention; UUIDv4 collision is not a real risk.)
- **R-2 — No factory-side tenant-account check.** atlas-character-factory does **not** call `accountsService.GetById(accountId)` to verify the account belongs to the active tenant before emitting the preset saga. atlas-account's own tenant-header scoping already returns 404 for cross-tenant reads, and downstream saga steps that touch the account will fail clearly with that boundary intact. The PRD §8 "validates that the target account belongs to the active tenant" requirement is satisfied by inheritance from existing tenant scoping; an explicit pre-check would add latency without defending against a real attack vector (atlas-ui has no auth, so the tenant header is whatever the operator's session sets in the first place).
- **R-3 — Validation error path uses preset id, not array index.** Both save-time (atlas-configurations validator) and apply-time (atlas-character-factory) JSON:API `errors[]` use `meta.path = "presets[<presetId>].<field>"`. Because the validator generates ids per R-1 before running rule checks, every preset has an id by the time errors are emitted. The UI renders both flavours through one code path.
- **R-4 — Canonical seed is GMS v83 only; non-GMS-v83 template files keep `characters.presets: []`.** The implementer does **not** stub region-appropriate presets into other `template_*.json` files; populating those is a content-engineering task per region/version owner. Acceptance criterion §10 #1 of the PRD is verified only against a template seeded from `template_gms_83_1.json` — see §9 below for the explicit narrowing.

---

## 9. Acceptance criteria delta

PRD §10 stands. Two notes:

- §10 #1 ("Loading `/templates/<id>/character/presets` for a fresh template and seeing the seeded 4th-job explorer preset list") applies only to a template seeded from `template_gms_83_1.json` per D-10 / R-4. Other region/version templates show an empty preset list as expected; populating their presets is content engineering owned per region.
- §10 #6 ("Forcing a preset-apply failure mid-saga … produces saga compensation that rolls back inventory/equipment/skills") is exercised by the new integration test in §6.

---

## 10. Files touched, summarized

| Service | New files | Modified files |
|---------|-----------|----------------|
| atlas-configurations | `tenants/characters/preset/{rest,model,builder,validator,validator_test}.go`; `templates/characters/preset/{...}` mirror; `data/{requests,mock/processor}.go` | `tenants/characters/rest.go`; `templates/characters/rest.go`; `tenants/resource.go` PATCH; `templates/resource.go` PATCH; `seed-data/templates/template_gms_83_1.json` (data); other `template_*.json` (data, empty array) |
| atlas-data | — | `skill/rest.go`; `skill/processor.go` (or loader); `skill/resource.go` `handleSearchSkillsRequest`; tests |
| atlas-character | `character/name_validity_resource.go`, tests | `character/processor.go` `IsValidName` widened; `character/resource.go` route registration; `kafka/consumer/character/consumer.go` `handleCreateCharacter` set Gm; README; routes.conf (already covered) |
| atlas-character-factory | `factory/preset_rest.go`; `data/{skill_requests,item_requests}.go`; `configurations/preset_requests.go`; `character/name_validity_requests.go`; tests | `factory/processor.go` `CreateFromPreset` + `buildPresetCharacterCreationSaga`; `factory/resource.go` two new routes; README; routes.conf |
| atlas-saga (lib) | — | `payloads.go` `CreateAndEquipAssetPayload.UseAverageStats`, `CharacterCreatePayload.Gm` |
| atlas-saga-orchestrator | — | `compartment/processor.go` add `RequestCreateItemWithStats`, propagate `UseAverageStats`; `compartment/producer.go` `RequestCreateAssetCommandProvider`; `kafka/message/compartment/kafka.go` `CreateAssetCommandBody.UseAverageStats`; `character/processor.go` + `character/producer.go` + `kafka/message/character/kafka.go` add `Gm`; `saga/processor.go` `forwardCharacterCreationResult` extend type-switch for `AwardMesosPayload` if missing; `compartment/mock/processor.go`; `character/mock/processor.go`; tests |
| atlas-inventory | — | `asset/processor.go` `Create` options-struct refactor + variance bypass; `compartment/processor.go` cascade of `Create` callers; `kafka/message/compartment/kafka.go` `CreateAssetCommandBody.UseAverageStats`; `kafka/consumer/compartment/consumer.go` `handleCreateAssetCommand`; tests |
| atlas-ui | `pages/templates-character-presets-form.tsx`; `pages/tenants-character-presets-form.tsx`; `services/api/factory.service.ts`; `lib/hooks/api/{useCharacterFromPresetMutation,useNameValidity,useAccountByName}.ts`; `components/features/characters/ApplyPresetDialog.tsx`; `components/features/accounts/AdminBootstrapWizard.tsx`; tests | `App.tsx` two routes; `lib/breadcrumbs/routes.ts`; `pages/AccountDetailPage.tsx` header action; `pages/AccountsPage.tsx` header action; sidebar entries on `TemplateDetailPage` / `TenantDetailPage` |
| `docs/TODO.md` | — | New entries per §7 |

---

## 11. References

- PRD: `docs/tasks/task-037-character-presets/prd.md`
- API contracts: `docs/tasks/task-037-character-presets/api-contracts.md`
- Data model: `docs/tasks/task-037-character-presets/data-model.md`
- UX flow: `docs/tasks/task-037-character-presets/ux-flow.md`
- Backend developer guidelines: `.claude/skills/backend-dev-guidelines/`
- Frontend developer guidelines: atlas-ui CLAUDE.md (loaded during this session)
- Project conventions: `CLAUDE.md`
