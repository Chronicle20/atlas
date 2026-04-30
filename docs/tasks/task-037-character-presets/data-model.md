# Character Presets — Data Model

Companion to `prd.md`. Describes the on-disk preset shape, how it lives at template vs tenant scope, and how it threads through the saga.

---

## Storage location

Presets live **inside the existing `characters` row** in atlas-tenants' `configurations` table — no new resource, no new row. The JSONB document for that row gains a `presets` array sibling to the existing `templates` array.

| Column          | Notes                                                          |
|-----------------|----------------------------------------------------------------|
| tenant_id       | UUID; null when row is template-scoped.                        |
| template_id     | UUID; null when row is tenant-scoped.                          |
| resource_name   | `"characters"` (existing)                                      |
| resource_data   | JSONB; shape `{ "templates": [...], "presets": [...] }`        |

Exactly one of `tenant_id` / `template_id` is non-null per row, matching the existing convention.

A row whose `resource_data` predates this change (i.e. has no `presets` key) is read as `presets: []` by readers — no migration required.

---

## Resource shape (JSONB)

```jsonc
{
  "templates": [
    // Existing player-creation option-lists. Unchanged.
  ],
  "presets": [
    {
      "id": "<uuid>",
      "attributes": {
        "name":        "<string, 1..64>",
        "description": "<string, 0..512>",
        "tags":        ["<string>", ...],

        "jobId":       <uint32>,
        "gender":      <0|1>,
        "face":        <uint32>,
        "hair":        <uint32>,
        "hairColor":   <uint32>,
        "skinColor":   <byte>,
        "mapId":       <uint32>,

        "level":       <byte, 1..250>,
        "meso":        <uint32>,
        "gm":          <int>,

        "stats": {
          "str": <uint16>,
          "dex": <uint16>,
          "int": <uint16>,
          "luk": <uint16>,
          "hp":  <uint16>,
          "mp":  <uint16>
        },

        "defaultName": "<string|null>",

        "equipment": [
          { "templateId": <uint32>, "useAverageStats": <bool> }
        ],

        "inventory": [
          { "templateId": <uint32>, "quantity": <uint32, ≥1> }
        ],

        "skills": [
          { "skillId": <uint32>, "level": <uint8, ≥1> }
        ]
      }
    }
  ]
}
```

---

## Field semantics

| Field             | Source of truth                                                              |
|-------------------|------------------------------------------------------------------------------|
| `id`              | Server-generated UUID at first save. Stable across template→tenant clones.   |
| `name`/`description`/`tags` | Free text; `tags` powers Admin Bootstrap filter.                  |
| `jobId`           | Concrete `job.Id`. Replaces the legacy `jobIndex`/`subJobIndex` pair, which was a creation-flow concept and not meaningful for already-decided presets. |
| `face`            | Single uint32 encoding eye style + color in WZ data. Confirmed during scope: one field is sufficient. |
| `hair` + `hairColor` | The factory folds hair-color into hair (`Hair + HairColor`) when emitting the saga, matching the existing convention in `services/atlas-character-factory/atlas.com/character-factory/factory/processor.go`. |
| `level`           | Used directly. The `create_character` saga payload already accepts `Level`. |
| `meso`            | Persisted on the character via the existing `Meso` builder field; no new code required. |
| `gm`              | Persisted via the existing `Gm` builder field. Stored as `int` to match `services/atlas-character/atlas.com/character/character/model.go:43`. |
| `stats.hp/mp`     | Map to the character's *max* HP/MP; stat re-derivation logic in atlas-character is unchanged. |
| `defaultName`     | Pre-fills the apply-time name input. Unique-per-world check still happens at apply time; preset never carries a guaranteed-unique name. |
| `equipment[].useAverageStats` | When true, the saga payload's `UseAverageStats` is set; atlas-inventory uses atlas-data defaults verbatim (no variance). |
| `inventory[].quantity` | Awarded to the appropriate inventory tab via `award_asset` saga step (existing logic infers tab from item id). |
| `skills[].level`  | Skill master level is computed by the factory at apply time (atlas-data lookup → max level for that skill); not stored in the preset. |

---

## Template ↔ tenant semantics

- A template owns the canonical `characters` document, including both `templates` and `presets`.
- When a tenant is created from a template, the entire `characters` document is cloned verbatim — both arrays, with each preset's `id` preserved — into a new `tenant_id`-scoped row. This is the existing clone behavior; no new code path is introduced.
- After clone, the template and tenant rows are independent. Edits at either scope do not propagate.
- The repo ships a curated default preset list (FR-27); that default applies during template *creation* and is merged into the seeded `characters` document next to whatever templates are seeded today. A template whose `characters.presets` is already non-empty keeps it.

---

## Saga payload changes

The factory emits the same `CharacterCreation` saga used by player creation. Two payload changes:

### `CreateAndEquipAssetPayload` (shared library, atlas-saga)

Add one boolean field:

```go
type CreateAndEquipAssetPayload struct {
    CharacterId     uint32      `json:"characterId"`
    Item            ItemPayload `json:"item"`
    UseAverageStats bool        `json:"useAverageStats,omitempty"`
}
```

`omitempty` keeps existing emitters (player creation, change-job, etc.) wire-compatible. Existing in-flight saga rows decode unchanged (Go zero-value boolean is false).

### `CreateSkillPayload` (shared library, atlas-saga)

No shape change. The factory computes `MasterLevel` at saga-build time using atlas-data (skill max level), then sets it on the existing field — no new field required.

---

## Validation summary

Preset-save validation lives in atlas-tenants and runs on every `PUT`. Preset-apply validation lives in atlas-character-factory and runs on every `POST /factory/characters/from-preset`. The factory does not blindly trust the stored preset — at apply time it re-validates equipment and skill ids against atlas-data so a stale preset (referencing an item template that was removed) fails with a clear `400` rather than half-emitting a saga that compensates immediately.

| Rule                                                | Save (atlas-tenants) | Apply (atlas-character-factory) |
|-----------------------------------------------------|:--------------------:|:-------------------------------:|
| `jobId` resolves                                    |          ✓           |                ✓                |
| `level ∈ [1,250]`                                   |          ✓           |                ✓                |
| Equipment slots unique                              |          ✓           |                ✓                |
| Equipment / inventory / skill ids resolve in atlas-data | ✓                |                ✓                |
| Name regex / blocked / duplicate                    |          —           |                ✓                |
| Tenant owns the target account                      |          —           |                ✓                |
