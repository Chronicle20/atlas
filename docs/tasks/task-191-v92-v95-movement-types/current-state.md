# Current State — movement `types` across all seed templates

Surveyed 2026-08-04 against `services/atlas-configurations/seed-data/templates/` at `main`
(`999e48a2a`). This is observed repo state, not derivation. It exists so the design phase does not
re-survey, and so the FR-5 invariant check has a known baseline.

## Move handler coverage

Counts are `len(options.types)`. `None` = the handler entry exists but has no `types` key.
`—` = no handler entry of that name in the template.

| template | Character | Monster | Pet | Summon |
|---|---|---|---|---|
| `template_gms_12_1.json` | 9 (`0x16`) | 9 (`0x58`) | — | 9 (`0xAF`) |
| `template_gms_48_1.json` | 23 (`0x21`) | 23 (`0x81`) | 23 (`0x71`) | **None** (`0x78`) |
| `template_gms_61_1.json` | 23 (`0x26`) | 23 (`0x9B`) | 23 (`0x8A`) | 23 (`0x92`) |
| `template_gms_72_1.json` | 23 (`0x28`) | 23 (`0xB2`) | 23 (`0xA1`) | 23 (`0xA9`) |
| `template_gms_79_1.json` | 23 (`0x27`) | 23 (`0xB4`) | 23 (`0xA3`) | 23 (`0xAB`) |
| `template_gms_83_1.json` | 23 (`0x29`) | 23 (`0xBC`) | 23 (`0xA7`) | 23 (`0xAF`) |
| `template_gms_84_1.json` | 24 (`0x29`) | 24 (`0xC1`) | 24 (`0xAC`) | 24 (`0xB2`) |
| `template_gms_87_1.json` | 25 (`0x2B`) | 25 (`0xC8`) | 25 (`0xB3`) | 25 (`0xBB`) |
| `template_gms_92_1.json` | **None** (`0x2E`) | **None** (`0xDC`) | **—** | **None** (`0xC8`) |
| `template_gms_95_1.json` | **—** | **None** (`0xE3`) | **None** (`0xC7`) | **None** (`0xCF`) |
| `template_jms_185_1.json` | 33 (`0x20`) | 33 (`0xC2`) | 33 (`0xAA`) | 33 (`0xB2`) |

`CharacterInventoryMoveHandle` appears in every template with no `types` and is **not** a movement
handler — it is inventory item movement. It is excluded from this survey's scope and from FR-5.1.

Gaps this task addresses: the seven `None`/`—` cells in the `gms_92_1` and `gms_95_1` rows, plus the
one `None` in `gms_48_1` (a task-179 / PR #1036 leftover — that task claimed Monster/Pet/Summon
coverage but this entry was missed).

## Cross-version `Name`/`Type` prefix

`CharacterMoveHandle.types` compared index-by-index across the four highest-numbered templates that
have one. **This is a continuity check for FR-1.5, not a source to copy from.**

| idx | v83 | v84 | v87 | jms_185 |
|---|---|---|---|---|
| 0 | `NORMAL`/NORMAL | same | same | same |
| 1 | `JUMP`/JUMP | same | same | same |
| 2 | `IMPACT`/JUMP | same | same | same |
| 3 | `IMMEDIATE`/TELEPORT | same | same | same |
| 4 | `TELEPORT`/TELEPORT | same | same | same |
| 5 | `HANG_ON_BACK`/NORMAL | same | same | same |
| 6 | `UNKNOWN`/JUMP | same | same | same |
| 7 | `ASSAULTER`/TELEPORT | same | same | same |
| 8 | `ASSASSINATION`/TELEPORT | same | same | same |
| 9 | `RUSH`/TELEPORT | same | same | same |
| 10 | `STAT_CHANGE`/STAT_CHANGE | same | same | same |
| 11 | `SIT_DOWN`/TELEPORT | same | same | same |
| 12 | `UNKNOWN`/JUMP | same | same | same |
| 13 | `UNKNOWN`/JUMP | same | same | same |
| 14 | `START_FALL_DOWN`/START_FALL_DOWN | same | same | same |
| 15 | `FALL_DOWN`/NORMAL | same | same | same |
| 16 | `START_WINGS`/JUMP | same | same | same |
| 17 | `WINGS`/NORMAL | same | same | same |
| 18 | `ARAN_ADJUST`/JUMP | same | same | same |
| 19 | `MOB_TOSS`/JUMP | same | same | same |
| 20 | `DASH_SLIDE`/JUMP | same | same | same |
| 21 | `UNKNOWN`/DEFAULT | same | same | same |
| 22 | `UNKNOWN`/JUMP | same | same | `UNKNOWN`/DEFAULT |
| 23 | — | `FLYING_BLOCK`/FLYING_BLOCK | `FLYING_BLOCK`/FLYING_BLOCK | `UNKNOWN`/JUMP |
| 24 | — | — | `UNKNOWN`/JUMP | `FLYING_BLOCK`/FLYING_BLOCK |
| 25–30 | — | — | — | `UNKNOWN`/JUMP ×5, `UNKNOWN`/DEFAULT at 26 |
| 31 | — | — | — | `MOB_ATK_RUSH`/NORMAL |
| 32 | — | — | — | `MOB_ATK_RUSH_STOP`/NORMAL |

Observations for the design phase to test against the v92/v95 clients, **not** to assume:

- Indices 0–21 are identical across all four GMS-lineage templates listed. jms_185 diverges from
  index 22 onward, so jms is not a useful neighbour for a GMS derivation.
- `FLYING_BLOCK` appears at index 23 from v84 onward, absent at v83.
- `FALL_DOWN` occupies index 15 in every template above. Its name is load-bearing — it is the only
  `Name` the decoder branches on (`libs/atlas-packet/model/movement.go:126-128`, mirrored in
  `Encode`), triggering the extra `FhFallStart` int16.
- Length trend 23 (v48–v83) → 24 (v84) → 25 (v87) → 33 (jms_185). Where v92 and v95 land is an
  output of the derivation.

## Decoder contract

`libs/atlas-packet/model/movement.go`:

- `movementPathAttrFromOptions` (:284-312) — indexes `options["types"]` by the element-type byte.
  Returns `("NOT_FOUND", "DEFAULT")` and logs at error level when `types` is missing, is not an
  array, is empty, the index is out of range, or the entry is not an object.
- `Movement.Decode` (:33-65) — branches on the returned `Type` through `NORMAL` → `TELEPORT` →
  `START_FALL_DOWN` → `FLYING_BLOCK` → `JUMP` → `STAT_CHANGE`, falling back to the bare `Element`.
  `DEFAULT` matches no branch, so it takes the fallback: `BMoveAction` (byte) + `TElapse` (int16),
  three bytes, versus 9–15 bytes for a real fragment. That desync is the bug.
- `NormalElement.Decode` (:118-138) — reads XOffset/YOffset only when
  `!t.IsRegion("GMS") || t.MajorAtLeast(88)`. This gate already covers v92 and v95 correctly and is
  out of scope.
