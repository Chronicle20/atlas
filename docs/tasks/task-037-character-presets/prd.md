# Character Presets — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-04-30
---

## 1. Overview

The Atlas UI today can browse accounts and characters but cannot create characters; characters are produced exclusively through the player-facing creation flow (atlas-login → atlas-character-factory) which validates user picks against the existing tenant `characters/templates` option-list. There is no operator path for producing a fully-specified, ready-to-play character for testing or content validation.

This feature introduces **Character Presets**: the deterministic dual of the existing player-creation template. A preset fully specifies a character — appearance, level, stats, meso, GM status, equipment (with deterministic stat rolls), starting inventory (with quantities), and a skill book (with levels) — so that a single UI action instantiates the character without operator picks. Presets are stored as a **sibling array under the existing `characters` configuration resource** (alongside the existing `templates` array), not as a parallel top-level resource. This keeps the URL surface, atlas-tenants storage row, and tenant/template clone semantics shared between the two — see Section 11 for the rationale. Presets follow the same template-then-tenant-clone ownership model as every other Atlas configuration.

Two operator workflows are supported on top of the preset catalog: **(a)** a one-shot **Admin Bootstrap** wizard that creates a fresh account and instantiates one character per preset matching a chosen tag (e.g. `4th-job`), so that a content engineer can spin up a play-testing rig in seconds; and **(b)** a per-account **Apply Preset** action so that an existing account can have a single play-test character grafted onto it without disturbing its other characters.

## 2. Goals

Primary goals:

- Add a `presets` array as a sibling of the existing `templates` array within the `characters` configuration resource, at template scope, cloned to tenant scope, and editable per-tenant via the Atlas UI. Storage row, REST URL, and clone semantics are shared with `templates`; only the in-document attribute is new.
- Extend `atlas-character-factory` so a character can be instantiated from a preset against `(accountId, worldId, name)` — emitting the existing `CharacterCreation` saga but with the preset's full specification, including deterministic equipment stat rolls, multi-slot equipment, meso, GM, and skill-with-level entries.
- Bypass the equipment stat-variance roll in `atlas-inventory` when the saga payload requests deterministic ("average") stats, so equipment created from a preset has predictable values matching the atlas-data default.
- Surface two UI flows: **Apply Preset to Existing Account** (from `AccountDetailPage`) and **Admin Bootstrap** (a wizard that creates a new account via the standard atlas-account flow and applies a tag-filtered preset subset).
- Pre-curate a set of 4th-job presets at template scope so the Admin Bootstrap wizard has something to apply on day one.

Non-goals:

- Modifying or removing the existing `characters/templates` resource or the player-facing creation flow in atlas-login.
- Authentication, RBAC, or audit logging for preset application — atlas-ui has no auth today and this feature does not change that.
- Editing existing characters. Presets only produce *new* characters.
- Per-equipment-item explicit stat overrides. The preset declares "deterministic from atlas-data defaults"; finer-grained control is not in scope.
- A bulk-import format for preset definitions. Presets are edited in the UI (or seeded once into atlas-tenants by ops).
- Presets that span multiple worlds or accounts. A single application targets one `(account, world)`.

## 3. User Stories

- As a content engineer, I want to spin up a fresh **Admin** account with one fully-equipped 4th-job character per class so I can play-test class balance changes without manually leveling characters.
- As a QA engineer, I want to add a single play-test character to an existing test account so I can reproduce a bug against a known character configuration without polluting the account's other characters.
- As a tenant operator, I want to edit the catalog of character presets for my tenant — adjust starting items, levels, GM flag — so I can tailor presets to my tenant's needs without forking code.
- As a platform engineer, I want preset definitions to live at template scope and clone to tenants on tenant creation, so that adding a new tenant inherits a curated baseline.
- As a developer reading the code, I want preset application to reuse the existing `CharacterCreation` saga rather than introduce a parallel creation path, so the same compensation, validation, and event surface applies.

## 4. Functional Requirements

### 4.1 Preset definition (storage)

- **FR-1**: The existing `characters` configuration resource (atlas-tenants) gains a sibling array `presets` next to its existing `templates` array. The resource URL, JSON:API document type, and storage row in the atlas-tenants `configurations` table are unchanged. Document shape becomes:

  ```jsonc
  {
    "characters": {
      "templates": [ /* unchanged player-creation option-lists */ ],
      "presets":   [ /* new admin/playtest concrete instances   */ ]
    }
  }
  ```

  Each preset entry is `{ "id": "<uuid>", "attributes": <PresetAttributes> }` per FR-4.
- **FR-2**: The same `characters.presets` array exists at template scope (atlas-tenants template-level configurations). Cloning a template into a tenant copies the entire `characters` document verbatim — including both `templates` and `presets`, with each preset's `id` preserved. Subsequent edits at tenant scope do not propagate back to the template, matching the existing behavior for `templates`.
- **FR-3**: Each preset has a stable `id` (UUID, generated at creation time) and is identified to the UI and to atlas-character-factory by that id.
- **FR-4**: Each preset has the following user-editable attributes:

  | Field          | Type       | Notes                                                                  |
  |----------------|------------|------------------------------------------------------------------------|
  | name           | string     | Human label (e.g. "Hero — 4th job"); 1–64 chars, free text.           |
  | description    | string     | Optional long-form description; ≤ 512 chars.                           |
  | tags           | []string   | Free-text tags; used by Admin Bootstrap to filter (e.g. `4th-job`).    |
  | jobId          | uint32     | Concrete `job.Id` (e.g. `412` for Hermit). Replaces job/sub-job indexing. |
  | gender         | byte       | 0 or 1.                                                                |
  | face           | uint32     | Single value (encodes eye style + color in WZ data).                   |
  | hair           | uint32     | Single value.                                                          |
  | hairColor      | uint32     | Single value.                                                          |
  | skinColor      | byte       | Single value.                                                          |
  | mapId          | uint32     | Starting map.                                                          |
  | level          | byte       | 1–250.                                                                 |
  | meso           | uint32     | Starting meso.                                                         |
  | gm             | int        | 0 = normal; non-zero = GM. Stored as `int` to match `atlas-character`. |
  | stats          | StatBlock  | `{ str, dex, int, luk, hp, mp }`, all uint16.                          |
  | defaultName    | string     | Optional default character name to suggest at apply-time.              |
  | equipment      | []Equip    | `{ templateId: uint32, useAverageStats: bool }`. Slot derived by factory. |
  | inventory      | []Item     | `{ templateId: uint32, quantity: uint32 }`.                            |
  | skills         | []Skill    | `{ skillId: uint32, level: uint8 }`. Master level set to skill max.    |

- **FR-5**: Within a single preset, `equipment` may not contain two entries that resolve to the same equip slot (per atlas-data `equipSlots`). Validation occurs at preset-save time and at preset-apply time.

### 4.2 Preset application (factory)

- **FR-6**: atlas-character-factory exposes a new endpoint:
  `POST /factory/characters/from-preset` with body `{ presetId, accountId, worldId, name }` (see `api-contracts.md`).
- **FR-7**: The endpoint resolves the preset by id from the active tenant's `characters.presets` array (404 if not found), validates the target name (FR-19), and emits a single `CharacterCreation` saga. The response is the same `CreateCharacterResponse { transactionId }` shape the existing `POST /factory/characters` returns.
- **FR-8**: The saga's `create_character` step uses the preset's full attribute block: name from the request, jobId/gender/face/hair/skinColor/mapId/level/stats/meso/gm from the preset, hair-color folded into hair (per the existing `Hair + HairColor` convention).
- **FR-9**: For each preset `inventory` entry, an `award_asset` step is emitted with `{ templateId, quantity }`.
- **FR-10**: For each preset `equipment` entry, a `create_and_equip_asset` step is emitted carrying a new `useAverageStats` boolean (default true for presets). When `useAverageStats=true`, the asset-creation path in atlas-inventory uses the atlas-data default value for every stat (no variance roll).
- **FR-11**: For each preset `skills` entry, a `create_skill` step is emitted with `{ skillId, level, masterLevel }` where `masterLevel` is derived by the factory by querying atlas-data for the skill's max level (factory falls back to `level` if atlas-data lookup fails, with a logged warning).
- **FR-12**: Saga step ordering matches the existing factory convention: `create_character` first, then `award_asset_*`, then `create_and_equip_asset_*`, then `create_skill_*`. `characterId=0` sentinel forwarding by the saga orchestrator (`saga/processor.go:1415-1450`) handles late character-id injection unchanged.
- **FR-13**: Preset application reuses the existing saga compensation surface — if any step fails, the saga rolls back via existing compensations. No new compensation actions are introduced.
- **FR-14**: The factory does **not** invent a parallel route or a parallel saga type; it composes the same `CharacterCreation` saga. The existing player-creation `POST /factory/characters` endpoint is unchanged.

### 4.3 Equipment stat determinism (atlas-inventory)

- **FR-15**: The shared saga payload `CreateAndEquipAssetPayload` gains a `UseAverageStats bool` field (omitempty; defaults to false to preserve existing player-creation behavior).
- **FR-16**: When the asset-status consumer in atlas-inventory processes a `create_and_equip_asset` whose payload sets `UseAverageStats=true`, `services/atlas-inventory/atlas.com/inventory/asset/processor.go` skips `getRandomStat(...)` and writes each equipable stat as the atlas-data default value verbatim. `slots` continues to use `ea.Slots()` as today.
- **FR-17**: The `award_asset` path is unaffected. Non-equip items (use/setup/etc/cash) have no stat variance and need no flag.

### 4.4 Admin Bootstrap (UI + atlas-account)

- **FR-18**: The Atlas UI exposes a new **Admin Bootstrap** entry point (location: AccountsPage header action, button labeled "Bootstrap Admin Account"). The wizard collects:
  1. Account name + password (forwarded to standard `POST /accounts/` create flow).
  2. Target world (drop-down of worlds for the active tenant).
  3. Tag filter (drop-down or chip selector populated from the union of preset `tags`; defaults to `4th-job`).
  4. A preview list of presets matching the filter, with toggles to deselect individual presets.
  5. A name override per selected preset (defaults to `defaultName`; required if `defaultName` is empty).
- **FR-19**: When the wizard submits, the UI orchestrates:
  1. `POST /accounts/` (atlas-account, returns 202 Accepted; account creation is async via Kafka).
  2. Poll `GET /accounts/?name=<name>` until the account materializes (timeout: 30 s; 1 s poll interval). Surface a clear error if the account does not appear in time.
  3. For each selected preset, in sequence: `POST /factory/characters/from-preset` with `{ presetId, accountId, worldId, name }`.
  4. Display per-preset success/failure inline, with the option to retry individual failures.
- **FR-20**: A name validity helper is exposed by atlas-character-factory at `GET /factory/characters/name-validity?name=<n>&worldId=<w>` returning `{ valid: bool, reason?: string }`. Reasons cover regex (`[A-Za-z0-9...]{3,12}`), uniqueness within world, and blocked-name list. The UI calls this helper before submitting each preset to surface validation errors immediately and before launching the wizard's apply phase.

### 4.5 Apply Preset to Existing Account (UI)

- **FR-21**: `AccountDetailPage` gains an **Add character from preset** button (visible whenever the active tenant has at least one preset). The button opens a dialog with: preset picker (searchable, shows preset name + tags + jobId), world picker, and a name input pre-filled with the preset's `defaultName`.
- **FR-22**: On submit, the UI calls `POST /factory/characters/from-preset`. On 202 Accepted, the dialog closes and a toast indicates the character is being created. The page invalidates `characterKeys.list(tenant)` so the new character appears once the saga completes.
- **FR-23**: The button is hidden if the active tenant has zero presets configured.

### 4.6 Tenant/Template UI

- **FR-24**: The UI gains two new pages following the existing pattern (existing precedent: `/templates/:id/character/templates`, `/tenants/:id/character/templates`):
  - `TemplatesCharacterPresetsPage` (route `/templates/:id/character/presets`) — edit the template-scoped catalog.
  - `TenantsCharacterPresetsPage` (route `/tenants/:id/character/presets`) — edit the tenant-scoped catalog.
- **FR-25**: The forms support: list view with create/edit/delete; per-preset detail view exposing every FR-4 field; tag chip editor; equipment list with item-template lookup (reusing the existing item search component if available, free-text uint32 input otherwise); inventory list (templateId + quantity); skills list (skillId + level).
- **FR-26**: Both pages link off the `TemplateDetailPage` and `TenantDetailPage` sidebars next to the existing **Character Templates** entries, so the operator can see at a glance that "templates" (player-creation) and "presets" (admin/playtest) are siblings.

### 4.7 Seeding

- **FR-27**: A canonical 4th-job preset catalog is committed to the repo and applied at template-scope creation time as part of the `characters` configuration document (next to the existing seeded `templates`), so a fresh template has a 4th-job preset for each explorer 4th-job class as a default. Preset list scope: explorer 4th-job classes (Hero, Paladin, Dark Knight, Fire/Poison ArchMage, Ice/Lightning ArchMage, Bishop, Bowmaster, Marksman, Night Lord, Shadower, Buccaneer, Corsair). Cygnus/Aran/Resistance/Legend 4th-job presets may be added but are not a launch requirement.
- **FR-28**: Preset seeding does not overwrite a non-empty `characters.presets` array on a clone source — the clone is the source of truth, matching how `templates` seeding behaves today.

## 5. API Surface

See `api-contracts.md` for full request/response shapes. Summary:

| Method | Path                                                | Service                  | Purpose                                                              |
|--------|-----------------------------------------------------|--------------------------|----------------------------------------------------------------------|
| GET    | `/configurations/characters` (tenant header)        | atlas-tenants            | **Existing endpoint.** Now returns both `templates` and `presets`.   |
| PUT    | `/configurations/characters`                        | atlas-tenants            | **Existing endpoint.** Body now also accepts `presets`.              |
| GET    | `/templates/{templateId}/configurations/characters` | atlas-tenants            | **Existing endpoint.** Same change at template scope.                |
| PUT    | `/templates/{templateId}/configurations/characters` | atlas-tenants            | **Existing endpoint.** Same change at template scope.                |
| POST   | `/factory/characters/from-preset`                   | atlas-character-factory  | **New.** Instantiate character from preset.                          |
| GET    | `/factory/characters/name-validity`                 | atlas-character-factory  | **New.** Check character-name validity.                              |

No new atlas-tenants endpoints are added. Existing endpoints unchanged: `POST /factory/characters` (player-creation flow) and `POST /accounts/` (account creation).

## 6. Data Model

See `data-model.md` for full schema, validation rules, and clone semantics. Storage summary:

- atlas-tenants `configurations` table (existing JSONB storage), `resource_name = "characters"` (existing row, not a new resource). The `resource_data` JSONB document gains a `presets` array sibling to the existing `templates` array. Each preset is `{ "id": "<uuid>", "attributes": {...} }`.
- No new SQL migrations are required in atlas-tenants — the JSONB document gains a new attribute, no schema change. Seed data wiring extends the existing template-creation path.
- atlas-character-factory and atlas-inventory introduce no new tables. The shared `CreateAndEquipAssetPayload` Go type (in the saga shared library) gains a single boolean field; payload is JSON-serialized into the saga store, so existing in-flight sagas remain decodable (Go's `omitempty` on a missing field defaults to false).

## 7. Service Impact

- **atlas-tenants**: Extend the existing `characters` configuration: REST handlers, provider/processor, JSON validators, mock processor, Kafka events, REST model, and seed loader gain a `presets` field next to `templates`. No new routes, no new resource type, no new Kafka topic. Validation on PUT is broadened to cover the new field per Section 4.1 / `data-model.md`.
- **atlas-character-factory**: New endpoint and handler `POST /factory/characters/from-preset`. New endpoint and handler `GET /factory/characters/name-validity`. New factory processor method that resolves a preset from atlas-tenants, looks up skill master levels via atlas-data, builds the `CharacterCreation` saga, and emits it. Existing factory code paths unchanged.
- **atlas-inventory**: Asset processor `Create` honors `UseAverageStats` from the saga-driven create path (the asset processor receives the flag via the create-and-equip event payload propagated by the saga orchestrator). Bypass `getRandomStat` when set.
- **atlas-saga-orchestrator**: `CreateAndEquipAssetPayload` (shared library) gains `UseAverageStats`. Character-id injection and compensation logic untouched.
- **atlas-character**: No code changes required; existing `Gm` and `Meso` builder methods are already plumbed by the `create_character` saga step.
- **atlas-account**: No code changes required; the wizard reuses `POST /accounts/` and `GET /accounts/?name=`.
- **atlas-data**: No code changes required; factory queries the existing skill information endpoint to derive `masterLevel`. Equipment defaults are already exposed.
- **atlas-ui**: Two new pages (`TemplatesCharacterPresetsPage`, `TenantsCharacterPresetsPage`) reusing the existing `useTemplate` / `useTenant` configuration hooks (since presets ride on the same `characters` resource the templates pages already use). One new dialog (Apply Preset on `AccountDetailPage`), one new wizard (Admin Bootstrap on `AccountsPage`). Helper hooks for preset selection and apply may live under `lib/hooks/api/useCharacterPresets.ts`, but resource fetch goes through the existing `useTemplates`/`useTenants` paths. Routes added in `App.tsx`.

See `ux-flow.md` for the wizard step layout.

## 8. Non-Functional Requirements

- **Multi-tenancy**: All new endpoints honor the four tenant headers (`TENANT_ID`, `REGION`, `MAJOR_VERSION`, `MINOR_VERSION`); preset lookup is tenant-scoped via `tenant.MustFromContext(ctx)` in the same way every other configuration resource is. Preset application validates that the target account belongs to the active tenant.
- **Idempotency**: Preset *application* is not idempotent (each call creates a new character); the wizard de-dupes by tracking which presets have already succeeded for a given Bootstrap session in the UI. Preset *editing* (PUT replacement of the resource list) is idempotent by design.
- **Performance**: Admin Bootstrap is sequential per preset to keep saga error reporting simple — typical 12-class catalog completes well under 60 s in local dev given existing saga latencies. No parallelism requirement.
- **Observability**: Factory logs each preset-driven saga emission with `preset_id`, `account_id`, `world_id`, `tenant_id` fields (existing logrus structured-field convention). Saga events emitted to Kafka are unchanged, so existing dashboards keep working.
- **Backwards compatibility**: `UseAverageStats` defaults to false on the saga payload, so existing `create_and_equip_asset` calls from the player-creation flow retain today's stat-variance behavior.
- **Security**: No new auth surface. Preset application can produce GM characters, which is dangerous in a non-admin environment; this is acceptable because (a) atlas-ui has no auth today and (b) the same risk already exists via direct DB or Kafka access. The risk is documented but not mitigated in this task.
- **Validation**: Preset-save server-side validation (atlas-tenants) covers: jobId is a known `job.Id`, gender ∈ {0,1}, level ∈ [1,250], no two equipment entries claim the same slot, all skill ids resolve in atlas-data, all item ids resolve in atlas-data. Failures return `400` with a JSON:API `errors` body.

## 9. Open Questions

- **OQ-1**: Should the Admin Bootstrap wizard support resuming a partially-completed bootstrap (account created, only some presets applied)? Current proposal: the wizard surfaces per-preset retry inline but does not persist progress server-side; closing the wizard mid-bootstrap loses the apply-state but the account and any successful characters remain. Confirm.
- **OQ-2**: Should preset edits at tenant scope ever be re-synced from template scope ("revert to template default")? Current proposal: no — clone-once, then tenant owns. Confirm.
- **OQ-3**: For `defaultName`, what's the disambiguation rule if two presets share the same `defaultName` and the operator runs both in one Bootstrap pass? Current proposal: surface the conflict in the wizard preview step and require operator override. Confirm.
- **OQ-4**: Should `useAverageStats` be a per-preset toggle (carried on the preset definition) or a per-equipment-entry toggle (per-item, as written in FR-4)? Current proposal: per-entry, defaulting to true. Confirm.
- **OQ-5**: Is there any concern with the GM-flag preset producing real-game-affecting GM characters in production tenants? If so, should preset application short-circuit GM > 0 in non-dev tenants? Current proposal: no gating — operator responsibility. Confirm.

## 10. Acceptance Criteria

A reviewer can verify the feature is complete by:

1. Loading `/templates/<id>/character-presets` for a fresh template and seeing the seeded 4th-job explorer preset list (FR-27).
2. Editing a tenant's preset catalog (`/tenants/<id>/character-presets`), changing a preset's level, and confirming via `GET /configurations/character-presets` that the change persisted at tenant scope and did not affect the parent template.
3. Opening `AccountDetailPage` for any tenant account and using **Add character from preset** to create a new character; the character appears in the account's character list within saga-completion latency, with appearance/level/stats matching the preset, equipment with deterministic stats from atlas-data, inventory items with the configured quantities, and skills at the configured levels with master = skill max.
4. Running the **Admin Bootstrap** wizard end-to-end with `tag = 4th-job`: a new account is created via the standard atlas-account flow, the wizard waits for the account, then applies all 4th-job presets sequentially, with per-preset success indicated. Each character lands on the account in the chosen world.
5. Forcing a name collision (running Bootstrap twice with the same `defaultName`s) produces clear per-character validation errors via the name-validity helper before saga emission, never producing a saga that fails server-side for the same reason.
6. Forcing a preset-apply failure mid-saga (e.g. invalid skill id) produces saga compensation that rolls back inventory/equipment/skills and leaves no partial character. Existing saga-orchestrator tests for this path continue to pass.
7. The existing player-creation flow (`POST /factory/characters`) and atlas-login character creation continue to work unchanged; equipment created via that flow continues to roll stats with variance (regression check on `getRandomStat`).
8. All new and impacted services build and tests pass: `atlas-tenants`, `atlas-character-factory`, `atlas-inventory`, `atlas-saga-orchestrator`, `atlas-ui`.

## 11. Design Rationale: Why Not Unify with `templates`

A natural reaction to this PRD is "we already have `characters.templates`; why introduce a second concept instead of extending the first?" This section records the rationale so future readers don't relitigate it.

### What the two model

- `characters.templates` is a **constraint set**. Each entry carries pick-lists (`faces[]`, `hairs[]`, `tops[]`, …) plus `jobIndex` + `subJobIndex`. atlas-login reads it during the player-facing creation flow to validate that the picks the player submitted are within the allowed set. A template entry is never, by itself, a character — it is the menu the player chooses from.
- `characters.presets` is a **concrete instance**. Each entry carries single values for every appearance field, plus `level`, `meso`, `gm`, full stat block, deterministic equipment, inventory with quantities, and skills with levels. A preset entry, when applied, produces exactly one fully-realized character with no further input than the target name.

These are two genuinely different intents on the same domain (a character's starting state). They share field *names* (face, hair, mapId, …) but the *shape* of those fields differs: a template carries `faces []uint32` because the player picks one; a preset carries `face uint32` because it's already decided.

### Options considered

1. **One unified entry shape with optional fields** — every entry has both pick-lists and concrete instance fields, consumers ignore what they don't need. Rejected because the semantic split moves from the resource boundary into every consumer (atlas-login has to ignore preset fields; the factory's apply path has to handle "what if this entry is options-only?"), validation rules become disjunctive (`len(faces)≥1 OR face is set`), and the editor UX becomes ambiguous (am I editing options or an instance?).
2. **Replace `templates` with `presets`** — rejected because atlas-login is currently stable, has no defect motivating the change, and the player-creation flow legitimately needs the constraint-set model.
3. **Make presets a special-case template (single-element lists)** — the apply path could "pick the only option" from a list. Rejected because it conflates two intents under one name and forces level/meso/gm/stats/etc. to live somewhere awkward (either as new fields on every template or in a side document keyed by template id), reintroducing option (1)'s problems.
4. **Adopted: sibling array under the same resource** — `characters.templates` stays as-is (atlas-login is unaffected), `characters.presets` is added next to it. Storage row, REST URL, JSON:API resource type, atlas-tenants Kafka events, and tenant/template clone semantics are all shared between them. The only duplication is at the in-document schema layer, where the duplication is justified by genuine semantic divergence.

### What this buys us

- **No new endpoints in atlas-tenants.** The `characters` resource handlers are extended in place.
- **No new storage row or migration.** A new attribute is added to an existing JSONB document.
- **Shared clone semantics.** Whatever atlas-tenants does today on tenant creation for `characters` automatically carries presets along.
- **atlas-login is untouched.** It reads `characters.templates`; the new attribute is invisible to it.
- **UI scaffolding can be shared.** Both editor pages fetch the same `characters` configuration document via the same React Query hooks and write back through the same mutation, only differing in which sub-array they render.

### When to revisit

If atlas-login ever loses the player-creation flow (e.g. deleted in favor of operator-only character provisioning), `templates` becomes vestigial and should be removed; presets would then be the only inhabitant of the `characters` resource and could optionally be promoted to a top-level resource name. That's a separate, larger decision than this task.
