## Character Render

### Responsibility

Parses and validates character render requests, resolves an equipped-item
loadout to a canonical hash, and composites a character sprite from body,
head, equipment, hair, and face part atlases into a single image.

### Core Models

- `RenderQuery` — parsed/validated render request parameters: `Skin`,
  `Hair`, `Face`, `Stance`, `Frame`, `Resize`, `Items`, `Gender`.
- `placement` — a positioned, ready-to-draw sprite: `templateID`,
  `partClass`, `sprite`, `atlasImage`, `anchor`.
- `anchorPoint` — a canvas-space `(X, Y)` coordinate.
- `vslotOwner` / `ownerKind` — records a template's claimed vslot codes and
  its occlusion-precedence class. Precedence order, highest first:
  `ownerEquipment`, `ownerHair`, `ownerFace`, `ownerHead`, `ownerBody`.
- `ErrorBody` / `wireError` — JSON:API-shaped error payload.

### Invariants

- The compositing canvas is fixed at 96×128 (`CanvasWidth` × `CanvasHeight`);
  the body skin anchors at `(CanvasWidth/2, FootRow-4)` where `FootRow = 124`.
- Internal skin values `0..5` and `6..10` map to WZ skin ids `2000..2005`
  and `2009..2013` respectively (`MapInternalSkin`); any other value yields
  `ErrUnknownSkin`.
- Supported stances are exactly `stand1`, `stand2`, `walk1`, `alert`,
  `jump` (`ValidateStance`, `SupportedStances`); any other value yields
  `ErrInvalidStance`.
- Equipment slots `-14`, `-18..-30` are dropped before compositing
  (`FilterEquipment`).
- Equipment slots `-101..-114` (cash/pet/mount) are dropped before
  compositing.
- Equipped item ids are mapped to slot integers (`slotForItemID`) by item
  classification: `-1` Hat, `-2` FaceAccessory, `-3` EyeAccessory, `-4`
  Earring, `-5` Top/Overall, `-6` Bottom, `-7` Shoes, `-8` Gloves, `-9`
  Cape/Ring/Pendant/Belt/Medal, `-10` Shield, `-11` any item with a known
  weapon type.
- `partClassFor` maps the same item classifications to asset category
  labels used to look up character part atlases: `Cap`, `FaceAccessory`,
  `EyeAccessory`, `Earrings`, `Coat`, `Longcoat`, `Pants`, `Shoes`, `Glove`,
  `Shield`, `Cape`, `Weapon`.
- An empty top slot (`-5`) is filled with the gender's default coat
  (`DefaultCoatMale = 1040036`, `DefaultCoatFemale = 1041046`) unless an
  Overall is equipped in that slot; an empty bottom slot (`-6`) is filled
  with the gender's default pants (`DefaultPantsMale = 1060026`,
  `DefaultPantsFemale = 1061039`).
- `ResolveGender`: an explicit gender parameter of `0` (male) or `1`
  (female) is authoritative; otherwise a face id where
  `(face/1000)%10 == 1` resolves female, all other face ids (including
  non-positive) resolve male. Idempotent for explicit `0`/`1` inputs.
- A two-handed weapon in the weapon slot (`-11`) forces `stance = stand2`
  when that weapon's atlas manifest ships a `stand2` sprite, overriding the
  requested stance.
- `CanonicalLoadoutString` sorts equipment item ids ascending before
  hashing, so item input order does not affect the resulting hash.
- `LoadoutHash` is the first 16 hex characters of SHA-256 of the canonical
  string.
- The head template always renders at stance `front`, frame `0`.
- Sprite anchoring (`solveViaSharedJoint`) walks already-placed parts
  most-recent-first for a matching joint name. Parts with no resolvable
  joint fall back to the body anchor for body-atlas parts and for
  non-body parts placed with `requireParent = false`; non-body parts
  placed with `requireParent = true` are dropped instead.
- Non-body atlas stance/frame resolution (`resolveTemplateStance`) falls
  back in order: requested stance/frame → `default/0` → a stance-specific
  fallback list (`stand2` → `stand1, walk1, alert`; `stand1` → `stand2,
  walk1, alert`; any other stance → `stand1, stand2`).
- vslot/smap occlusion precedence is equipment > hair > face > head >
  body; a part is suppressed only if every two-character slot code its
  z-layer occupies is claimed by a different template.
- Final draw order sorts by descending zmap index of each sprite's Z
  label (back-most first); a Z label absent from zmap sorts last (index =
  `len(zmap)`); an empty zmap collapses the sort to insertion order.
- `NearestNeighborUpscale` expands each source pixel into an N×N block for
  integer resize factors ≥ 1; a resize value `< 1` is treated as `1`.
- `ParseRenderQuery` defaults: `stance = stand1`, `frame = 0`, `resize =
  2`. `frame` must be `>= 0`; `resize` must be in `1..4`; `gender`, if
  present, must be `0` or `1`; `skin`, `hair`, and `face` are required.

### State Transitions

Not applicable — each render request is handled statelessly.

### Processors

- `ParseRenderQuery` — parses and validates query parameters into a
  `RenderQuery`.
- `ResolveGender` — derives the effective gender from the query gender
  selector and face id.
- `CanonicalLoadoutString` / `LoadoutHash` — derive the canonical
  cache-key hash for a loadout.
- `FilterEquipment` / `ItemsToSlotMap` / `slotForItemID` /
  `partClassFor` — normalize the requested equipped-item list into a slot
  map and resolve each item's compositing asset category.
- `MapInternalSkin` / `ValidateStance` — validate the skin id and stance.
- `Composite` — orchestrates the full render: resolves stance and skin,
  applies default clothing, fetches body/head/equipment/hair/face atlases,
  applies vslot occlusion, z-orders parts, and blits them onto the output
  canvas.
- `applyVslotOcclusion` / `claimSlots` / `isPartVisible` — resolve
  cross-template equipment/hair/face occlusion.
- `NearestNeighborUpscale` — post-composite integer upscaling.

## Map Render

### Responsibility

Composites a per-map image from a parsed Map.wz archive's layer data,
stacked in the map layout's z-order.

### Core Models

None defined locally; operates on `maplayout.Layout` and WZ archive types
supplied by the `atlas-wz` library.

### Invariants

- The map `.img` lookup key is the map id zero-padded to 9 digits; a
  non-padded exact-match fallback is also attempted.
- Layer stacking order is `layout.ZMap` when non-empty; otherwise it falls
  back to the declaration order of `layout.Layers`.
- A layer listed in the layout but absent from the extracted layer set is
  skipped.
- The composited canvas is sized to `layout.Bounds`
  (`Left`/`Top`/`Right`/`Bottom`); empty bounds is an error.
- Map background layers (`Map.img` `back[]`) are not composited; only the
  foreground world layers are drawn, onto a transparent canvas.

### State Transitions

Not applicable.

### Processors

- `CompositeFromWZ` — builds a Map.wz index, resolves the requested map's
  `.img`, extracts its layers, and stacks them into the output canvas.
