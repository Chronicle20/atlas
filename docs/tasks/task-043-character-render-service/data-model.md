# Character render — data model

Source of truth for sprite layout, joint conventions, and the metadata schema the render service consumes. Verify against the live `tmp/wz-input/Character.wz` extract before implementation; some details below are inferred from a single XML walk and may need adjustment.

## Character.wz file layout (v83 GMS)

```
Character.wz/
├── 00002000.img             body, skin 2000 (light, female)
├── 00002001.img             body, skin 2001 (ashen, female)
│   ... 00002009-00002013    other female skin colors
├── 00012000.img             body, skin 2000 (light, male)
│   ... 00012001-00012013    male skin colors
├── Cap/                     hats (templateId 1002xxx)
│   └── 0100XXXX.img
├── Coat/                    tops (1040xxx)
├── Longcoat/                overalls (1050xxx)  — overrides Coat slot when present
├── Pants/                   bottoms (1060xxx)
├── Shoes/                   shoes (1070xxx)
├── Glove/                   gloves (1080xxx)
├── Cape/                    capes (1100xxx-1140xxx)
├── Shield/                  shields (1090xxx)
├── Weapon/                  weapons (130xxxx-159xxxx)
├── Hair/                    hairs (30000-49999)
├── Face/                    faces (20000-29999)
├── Accessory/               face/eye/earring accs (1010xxx, 1020xxx, 1030xxx)
├── Ring/                    rings (1110xxx)            — out of scope (no visual)
├── Pendant/                 pendants (1120xxx)         — out of scope (no visual)
├── PetEquip/                pet items                  — out of scope
├── TamingMob/               mounts                     — out of scope
├── Dragon/                  evan dragon parts          — out of scope (post-evan content)
├── Afterimage/              weapon swing afterimages   — out of scope (animation only)
```

Body skin imgs use the pattern `0000{skin}` for female and `0001{skin}` for male. The `{skin}` portion is the WZ skin id (2000–2013, non-contiguous: 2000, 2001, 2002, 2003, 2004, 2005, 2009, 2010, 2011 currently shipped per `tmp/` extract).

## Per-img structure

Every `.img` has a top-level `info/` block followed by stance/frame canvases:

```
{templateId}.img/
├── info/
│   ├── islot (string)         "Hp" / "Bd" / "Cp" — equipment slot category
│   ├── vslot (string)         "Hp" / "Bd*Hb" — visual slot (with overrides)
│   ├── cash (int)             0 = regular, 1 = cash item
│   └── (other info: defense stats, requirements, etc. — not needed for rendering)
├── stand1/
│   ├── 0/                    frame 0
│   │   ├── {part}            canvas (e.g. body, arm, head, hairOverHead)
│   │   │   ├── origin (vector)         anchor point on this sprite
│   │   │   ├── map/                    joint dictionary
│   │   │   │   ├── neck (vector)
│   │   │   │   ├── navel (vector)
│   │   │   │   └── hand (vector)
│   │   │   ├── z (string)              layer name (resolved via smap)
│   │   │   └── group (string)          "skin" / "head" / "weapon" / etc.
│   │   ├── delay (int)                 frame duration ms
│   │   └── face (short)                whether face renders this frame (1/0)
│   └── 1/, 2/, ...                     additional frames
├── stand2/                  alt stance (typically two-handed)
├── walk1/                   walk animation
├── alert/                   alert pose
├── jump/                    jump pose
└── (other stances — fly, prone, swing, etc., not in scope)
```

Equipment imgs follow the same shape; the part names differ (e.g. a weapon img has `weapon` and `weaponOverHand` parts, a hair img has `hair`/`hairOverHead`/`hairBelowBody` parts).

## Joint system

Sprites compose by mapping each child's `origin` to its parent's joint coordinate, then translating in canvas space.

| Joint | Owner | Used by |
|---|---|---|
| `neck` | body | head, hair, face |
| `navel` | body | top, overall, cape |
| `hand` | arm | weapon, shield |
| `head` | head/hair | hat, face accessory |
| `brow` | head | eye accessory |
| `lEar` / `rEar` | head | earrings |
| `body` | body | overall (alternate to navel for full-body suits) |

A render walks the joint tree starting from body. Body's `origin` lands at a fixed canvas position (we choose: foot row 124, horizontal center). All other parts join transitively.

## Z-order (smap)

Each sprite carries a string `z` (e.g. `"body"`, `"arm"`, `"hairBelowBody"`, `"weapon"`, `"weaponOverGlove"`). The render order is determined by a smap — a global `string → int` mapping that sorts these strings into draw order.

**Open question:** the smap's location in WZ files is not yet confirmed. Hypothesis: `Base.wz/smap.img` or `Character.wz/smap.img`. The design phase must verify and document. Falls under the data-model open questions.

If the smap is absent or unparseable, fallback: ship a hardcoded smap derived from a community reference (e.g. HaRepacker), document its source.

## Sprite metadata sidecar

Each extracted sprite gets a JSON sidecar at `{templateId}/{stance}/{frame}/{partName}.json`:

```json
{
  "origin": { "x": 19, "y": 32 },
  "map": {
    "neck": { "x": -4, "y": -32 },
    "navel": { "x": -6, "y": -20 }
  },
  "z": "body",
  "group": "skin",
  "delay": 180,
  "face": 1
}
```

Plus one `{templateId}/info.json`:

```json
{
  "islot": "Bd",
  "vslot": "Bd",
  "cash": 0
}
```

The compositor loads these sidecars on first render of a loadout (then cached in-process) rather than re-parsing WZ.

## Skin color mapping

atlas-ui currently maps internal skin IDs 0..10 to WZ ids 2000..2013 via `SKIN_COLOR_MAPPING` (in `services/api/maplestory.service.ts`). The new render endpoint accepts the *internal* 0..10 value (not the WZ id) and applies the mapping server-side. atlas-ui's mapping table is removed; the server is the single source of truth.

Mapping (verified against existing constants):

| Internal | WZ id | Name |
|---|---|---|
| 0 | 2000 | Light |
| 1 | 2001 | Ashen |
| 2 | 2002 | Pale Pink |
| 3 | 2003 | Clay |
| 4 | 2004 | Mercedes |
| 5 | 2005 | Alabaster |
| 6 | 2009 | Ghostly |
| 7 | 2010 | Pale |
| 8 | 2011 | Green |
| 9 | 2012 | Skeleton |
| 10 | 2013 | Blue |

## Loadout hash

The cache key is a stable hash of the rendered tuple. Inputs in canonical order:

```
{tenant}|{region}|{version}|{skin}|{hair}|{face}|{stance}|{frame}|{resize}|{itemIds-sorted-comma-separated}
```

Hash function: SHA-256 truncated to 16 hex chars (64 bits). Output filename: `character/{hash}.png`. 64-bit collision space against ~10⁹ unique loadouts gives a negligible collision probability.

**Note:** the *order* of equipment templateIds is normalized (sorted ascending) before hashing so `1002000,1040002` and `1040002,1002000` collapse to one cache entry.
