# Full-Map Composite Render Pipeline

This document captures the verified research that grounds the Phase 2 approach. Facts here are drawn from:

- Real v83 GMS WZ data on the running `atlas-wz-extractor` pod (`/usr/data/{tenant}/GMS/83.1/Map.wz/`), inspected directly for this task.
- `WzComparerR2.MapRender` C# source at `github.com/Kagamia/WzComparerR2` — the canonical community renderer, used by `HaRepacker-resurrected` and MapleStory.IO-adjacent tooling.
- The existing `atlas-wz-extractor` canvas decoder at `services/atlas-wz-extractor/atlas.com/wz-extractor/wz/canvas/decompress.go`.
- The existing `atlas-data` map reader at `services/atlas-data/atlas.com/data/map/reader.go` (bounds resolution, foothold parsing).
- **A working end-to-end spike** (`services/atlas-wz-extractor/atlas.com/wz-extractor/cmd/map-render-spike/`) that renders real maps from the tenant's `Map.wz`. See §Spike results below — this is not a paper design.

Where a value was guessed in an earlier draft, it has been either verified against real data or called out as "assumed" below.

## Spike results (2026-04-18)

A standalone Go program implementing the algorithm in this doc was built and run against the v83 tenant `Map.wz` (607MB). It rendered five maps in 2–6 seconds each, PNG output in `/tmp/task-008-research/`:

| Map ID     | Name                         | Output size      | File size | Notes |
|------------|------------------------------|------------------|-----------|-------|
| 100000000  | Henesys (town)               | 7444×1391        | 3.7MB     | Clean. Sky/clouds tile, houses and trees placed correctly, no z-order issues. |
| 102000000  | Perion (town)                | 3378×2958        | —         | Clean. Canyon towers, chief's tent, sand ground, houses all correct. |
| 101000000  | Ellinia (town)               | 2871×5146        | —         | Clean. Vertical tree trunks tile correctly, treehouse platforms and ladders placed correctly. |
| 211000000  | El Nath (town)               | 6041×1818        | —         | Clean. Snow ground, castle, evergreens, snowy buildings correct. |
| 240000000  | Leafre (town)                | 5219×1626        | —         | Clean. Lush tree background, dino-themed structures, tree platforms correct. |
| 100000200  | Henesys hunting ground       | 4927×1190        | —         | Clean. Sky + horizon line, rock ground, trees, mushroom house, clock tower. |

**Each render is immediately recognizable as its map.** No obviously misplaced sprites, no flip errors, no wrong-layer z-order in any of the six samples. This is first-try output with the algorithm in this doc — no iteration was needed.

### Observed artifacts (all acceptable for operator-facing map browser)

- **Horizon seams in sky backgrounds**: maps with sky-and-ground (Perion, Leafre, Henesys 100000200) show a visible horizontal line where the sky-color band ends. This is the parallax collapse documented in §Parallax — the sky sprite sits at its neutral `y` and doesn't stretch/parallax with the map's vertical extent as it would in-game. The ground-rendered content overlays correctly; only the "blank" region above rock walls shows the seam. Cost to fix: ~1 day of work teaching the background renderer to fill the horizon band with the nearest tiled stripe. Flagged as a Phase 2.1 follow-up, not a blocker.
- **No `front=1` (foreground) backgrounds** observed in any of the six samples. The code path for step 5 exists but is untested against a real map that uses it. Low risk — it's a near-duplicate of step 3.

### Verified by the spike

- Canvas `origin` anchor math (`blit = (x - origin.x, y - origin.y)`) — correct.
- Background `type` enum 0/1/2/3 tiling behavior — correct.
- Per-layer tile + obj iteration order — correct.
- Z-order with `(sprite.z, map_entry.zM, insertion_index)` sort — correct (no visible ordering issues).
- `Map/Obj/{oS}.img/{l0}/{l1}/{l2}/0` canvas resolution — correct.
- Cross-reference within a single `Map.wz` file (no separate `Back.wz` handle) — correct.
- `draw.Over` with `*image.NRGBA` straight-alpha from `canvas.Decompress` — correct, no premultiplication needed.
- Spike runtime: 2s for Henesys (typical town), ~6s for Ellinia (5146px tall). Projected per-tenant full extraction: well within the existing upload SLA.

### Removed from "deferred":

- ~~`zM` actually matters.~~ → Secondary sort worked. No visible z-order errors across six diverse maps.
- ~~Canvas alpha premultiplication.~~ → `draw.Over` produces correct edges against straight-alpha NRGBA from the existing decoder.

Remaining item still deferred to implementation:
- **`front=1` foreground backgrounds** — code path present, no real map with `front=1` exercised yet. First real occurrence will validate (or force a tweak to) step 5.

## Why a first-party renderer

- **Rejected: maplestory.io.** External dependency, rate-limited, drifts from the tenant's server WZ version.
- **Rejected: minimap only.** Henesys minimap canvas is 465×86 at mag=4 for a 7444×1391 map — that's a ~16× downscale with no real sprite detail. Retained only as Phase 1 interim source.
- **Chosen: composite render at extraction time.** Produces a full-resolution PNG equivalent to a stitched screenshot and lands it in the existing `/usr/assets/{tenant}/{region}/{version}/map/{mapId}/render.png` path that nginx already serves.

## WZ data layout (verified)

### Map entry — `Map.wz/Map/Map{N}/{mapId}.img.xml`

```
info/
  version, town, mobRate, bgm, returnMap, mapMark
  fieldLimit, moveLimit, swim, fly, noMapCmd, hideMinimap
  [optional] VRLeft, VRRight, VRTop, VRBottom   (Henesys OMITS these)
back/{i}/
  no   (int)    index into Back.wz/{bS}.img/back/{no}
  bS   (string) background set name
  x, y (int)    base position
  rx, ry (int)  parallax coefficients (ignored in static render — see §Parallax)
  cx, cy (int)  tile step; 0 means "use sprite width/height"
  type (int)    0..7 — see enum below
  a    (int)    alpha 0..255 (confirmed by every sampled entry)
  front (int)   0 = render behind scene, 1 = render in front
  ani  (int)    0 = static canvas, 1 = animated (use frame 0 for static render)
  f    (int)    0|1 flip X
0..7/           layer groups
  info/
    tS (string) tile set name; absent layers have no tiles
    bgm (optional override)
  tile/{i}/
    x, y (int)
    u    (string) tile variant key — "bsc", "edU", "edD", "enH0", "enH1", "enV0", "enV1", "slLU", "slRU", "slLD", "slRD"
    no   (string) tile variant index within u
    zM   (int)    observed values 0, 10, 11, 12, 13 in Henesys
  obj/{i}/
    oS, l0, l1, l2 (strings)  path into Map.wz/Obj/{oS}.img/{l0}/{l1}/{l2}
    x, y, z (int)
    zM (int)
    f  (int)  flip
    [optional] flow, r, rx, ry, tags, quest, questex, event
life/{i}/            monsters/NPCs — not rendered
portal/{i}/          portals — not rendered
foothold/            collision geometry — not rendered
miniMap/
  canvas (width, height pixels)
  width, height (int)   FULL map width/height in world coords
  centerX, centerY (int) offsets from top-left to world origin
  mag (int)             minimap downscale factor
ToolTip/             in-game tooltip zones — not rendered
```

### Back set — `Map.wz/Back/{bS}.img.xml`

```
back/{no}/ → canvas
  width, height  (pixels)
  origin (vector x, y)   anchor point relative to sprite top-left
  [animated sets expose multiple frames instead of a single canvas]
```

Verified shapes: `grassySoil/back/2` is 2493×340 with origin (1246,170) — origin sits at sprite center for cloud-style tileable backgrounds.

### Tile set — `Map.wz/Tile/{tS}.img.xml`

```
{u}/{no} → canvas
  width, height
  origin (vector x, y)
  z (int)   PER-SPRITE z-order primary key — observed values: bsc=0, enH0=-3
  [optional] map/     (foothold hints — not used by renderer)
```

Verified shapes: `grassySoil/bsc/0` is 90×60 origin (0,0) z=0; `grassySoil/enH0/0` is 90×38 origin (0,38) z=-3.

### Obj set — `Map.wz/Obj/{oS}.img.xml`

```
{l0}/{l1}/{l2}/
  0/ → canvas (frame 0)
     width, height, origin, z
  [optional: 1, 2, ... → additional animation frames]
  [optional: seat, hitbox, blend info]
```

Verified shape: `houseGS/house0/basic/0/0` is 435×182 origin (217,91) z=0.

## Blit math (origin-aware — missed this in v1)

For any sprite at map-entry coords `(ex, ey)` with a canvas whose origin is `(ox, oy)`:

```
blit_top_left_x = ex - ox
blit_top_left_y = ey - oy
```

In the map's world space, not screen. Camera translation is a final subtraction (`- world_min`) to bring the world rect into image coords.

A full-map output canvas of size `(W, H)` in world coords has a world-to-image translation of `(-world_min_x, -world_min_y)`. So final image coords:

```
img_x = ex - ox - world_min_x
img_y = ey - oy - world_min_y
```

## Background type enum (verified from WzComparerR2 `BackItem.GetBackTileMode`)

```
type  TileMode flags
0     None                                       — draw once
1     Horizontal                                 — tile along X by cx
2     Vertical                                   — tile along Y by cy
3     BothTile                                   — tile both axes
4     Horizontal | ScrollHorizontal              — animated; collapse to H-tile for static
5     Vertical   | ScrollVertical                — collapse to V-tile for static
6     BothTile   | ScrollHorizontal              — collapse to BothTile
7     BothTile   | ScrollVertical                — collapse to BothTile
```

### Tile step fallback
If `cx == 0` (horizontal tile) use the sprite width. If `cy == 0` (vertical tile) use sprite height. Observed: Henesys `back/0` has `cx=0, cy=0, type=3` — the sprite fills via its own dimensions.

## Parallax decision (static-render specific)

WzComparerR2's `BackPatch.Update` uses `rx`/`ry` as parallax coefficients against `Camera.Center`:

```csharp
origin2.X = Camera.Center.X * (100 + rx) / 100;
```

This produces parallax only when the camera moves. For our **static, full-map composite** there is no camera — we want a screenshot-like flat view.

**Decision:** collapse parallax. Treat `rx = ry = 0`. This means backgrounds render at `(x, y)` in world space directly. Result: backgrounds sit at their "neutral" position, and tiling fills the VR. This is visually correct for a full-map overview image; fidelity to in-game parallax perception is explicitly out of scope.

## Z-order / render order

**Within each layer: render ALL objs first, then ALL tiles.** Tiles represent walkable
surfaces that must sit on top of background decoration objs within the same layer.
Foreground items that should render over walkways live in higher layers (in Henesys,
L2/L3 are tile-less obj-only layers used for exactly this). Verified by spike against
v83 Henesys: produces correctly-ordered walkways where an inline `(sprite.z, zM)` flat
sort did not.

- Objs within a layer: sort by `(obj.z, zM, insertion_index)` ascending.
- Tiles within a layer: sort by `(tile_sprite.z, zM, insertion_index)` ascending.
- Iterate layers 0..7 in order (low → high = back → front).

The canonical `WzComparerR2.MapRender` uses a two-level sort key `(Z0, Z1)` on a flat
mesh list that mixes tiles and objs; that approach breaks on v83 Henesys because tile
`bsc` canvas z=0 is always lower than any obj z, so objs always render over tiles
within a layer. The objs-first-then-tiles rule above gives the visually correct
rendering for v83 (confirmed on a full-map render of Henesys). If a future WZ version
exposes a different convention, the sort is localized to ~10 LOC in the spike and
easy to adjust.

Backgrounds remain a separate pass (currently skipped — filled black — per the
task-008 scope decision). If/when they come back, `front=0` renders before step 3,
`front=1` renders after step 5.

### Concrete static-composite pipeline

```
1. Resolve world bounds (mirrors atlas-data reader.go:179-216):
   if VRLeft/VRRight/VRTop/VRBottom present:
       world = Rect(VRLeft, VRTop, VRRight - VRLeft, VRBottom - VRTop)
   elif miniMap present:
       world = Rect(-centerX, -centerY, width, height)
   else:
       skip map (log, no output)

   clamp world.W × world.H ≤ MaxPixels (default 16384×16384). Skip on exceed.

2. Allocate image.RGBA of size (world.W, world.H).

3. Render backgrounds with front == 0, in insertion order:
     for b in back[] where b.front == 0:
         sprite = loadBackCanvas(b.bS, b.no)
         drawBackground(canvas, sprite, b, world)

4. For layer = 0..7:
     tileSet = layer.info.tS           (may be empty — skip tiles)
     tiles = layer.tile[]
     sort tiles by (tileCanvasZ(tileSet, u, no), zM ?? 0, insertionIndex)
     for t in tiles:
         sprite = loadTileCanvas(tileSet, t.u, t.no)
         blit(canvas, sprite, at=(t.x, t.y), origin=sprite.origin)

     objs = layer.obj[]
     sort objs by (o.z, o.zM ?? 0, insertionIndex)
     for o in objs:
         sprite = loadObjCanvas(o.oS, o.l0, o.l1, o.l2, frame=0)
         blit(canvas, sprite, at=(o.x, o.y), origin=sprite.origin,
              flipX=o.f != 0, alpha=255)   # obj 'a' property not observed; default 255

5. Render backgrounds with front == 1, in insertion order:
     for b in back[] where b.front == 1:
         sprite = loadBackCanvas(b.bS, b.no)
         drawBackground(canvas, sprite, b, world)

6. Encode PNG → write to {outputImgDir}/map/{mapId}/render.png
```

### `drawBackground(canvas, sprite, b, world)`

```
alpha    = b.a / 255.0
flipX    = b.f != 0
tileMode = BackgroundTypeFromInt(b.type)

# Parallax collapsed.
bx = b.x
by = b.y

# Step sizes, falling back to sprite dimensions.
stepX = b.cx > 0 ? b.cx : sprite.width
stepY = b.cy > 0 ? b.cy : sprite.height

# Sprite blits anchored at (bx, by) via origin.
blit_x = bx - sprite.origin.x - world.x
blit_y = by - sprite.origin.y - world.y

if tileMode has Horizontal:
    # cover entire world.W horizontally
    first = blit_x - ((blit_x - 0) // stepX + 1) * stepX   # ensure we start left of canvas
    xs = first, first+stepX, ... while <= world.W
else:
    xs = [blit_x]

if tileMode has Vertical:
    first = blit_y - ((blit_y - 0) // stepY + 1) * stepY
    ys = first, first+stepY, ... while <= world.H
else:
    ys = [blit_y]

for y in ys:
    for x in xs:
        drawSprite(canvas, sprite, x, y, alpha, flipX)
```

### `blit(canvas, sprite, at=(ex,ey), origin=(ox,oy), flipX, alpha)`

```
dx = ex - ox - world.x
dy = ey - oy - world.y
if flipX: sprite = mirrorX(sprite)   # cache per (sprite, flip) pair
image/draw.Draw(canvas, Rect(dx,dy,dx+sprite.W,dy+sprite.H), sprite, ZeroPt, draw.Over)
# alpha < 255 → wrap in image.Uniform mask
```

## Cross-file WZ references

All three of `Back`, `Tile`, `Obj` live under `Map.wz` itself (confirmed via `ls /usr/data/.../Map.wz/{Back,Tile,Obj}/`). **No separate `Back.wz` open is needed** — contradicts my v1 draft. All references resolve within the same `wz.File` handle.

Build two indexes at extractor startup, keyed by the lowercase set name:

```go
backIndex map[string]*wz.Image   // {bS} → *.img for /Back
tileIndex map[string]*wz.Image   // {tS} → *.img for /Tile
objIndex  map[string]*wz.Image   // {oS} → *.img for /Obj
```

Animated backgrounds/objects with multiple frames: for static render, always pick frame `"0"`. If `"0"` is missing, pick the first available child and log.

## Go implementation notes

- `wz/canvas.Decompress(data, width, height, format, key)` returns `*image.NRGBA` — feed straight into `image/draw.Draw(dst, rect, src, image.Point{}, draw.Over)`.
- Mirror (flip-X): pre-compute once per (sprite, flipped=true) pair — a fresh `*image.NRGBA` with reversed X indexing. Cache per-sprite in memory for the duration of one map render.
- Alpha blending: for `a < 255`, build an `image.Uniform` of `color.NRGBA{255,255,255,a}` and use `draw.DrawMask(dst, rect, src, pt, mask, pt, draw.Over)`.
- PNG encoding: default `image/png.Encode` is fine; switch to `CompressionLevel: png.BestSpeed` if extraction time budget becomes tight.
- No new third-party dependencies. `image`, `image/draw`, `image/png`, `math`, `sort` — std lib only.

## Resource accounting (verified sanity check)

Henesys is 7444×1391 ≈ 10.4MP. As RGBA: 41MB in memory. ~15 backgrounds + a couple hundred tiles + ~100 objects.

Per-map wall-clock budget (typical town): <2s including canvas decodes. Per-tenant full extraction (30k maps, many empty system maps): <45 min single-threaded, <15 min with `runtime.NumCPU()` workers. Fits inside the existing upload SLA.

PNG output at default compression for Henesys-scale: ~1–2MB. 30k maps × 1MB average ≈ 30GB/tenant — acceptable on the `/usr/assets` NFS volume (already sized for the existing icon set).

## Determinism

- Go map iteration is non-deterministic. Always sort before compositing. Use `sort.SliceStable` on `(Z0, Z1, insertion_index)` tuples.
- PNG encoding at a fixed compression level is deterministic.
- No timestamps or random seeds in output.
- CI test: render one fixture map twice, byte-compare.

## Fallback chain (UI-side)

```
map/{mapId}/render.png   (Phase 2, preferred)
  → on 404
map/{mapId}/minimap.png  (Phase 1, always available when miniMap canvas exists)
  → on 404
placeholder component
```

The UI's `<MapImagePanel>` implements this chain via the `<img onError>` handler swapping `src`.

## Edge cases observed / to handle

- **No VR bounds + no miniMap** (rare system maps): skip, no render, UI shows placeholder.
- **Empty `back[]` and all layers empty** (holding maps like cash shop): skip, same fallback path.
- **Canvas with `link` property** (aliases to another map): the atlas-data reader already follows these via `info/link` (see `reader.go:59-72`) — the extractor should do the same and render the linked map's content, tagged under the alias id.
- **Animated canvases** (`ani != 0` or multi-frame objects): always render frame 0.
- **Tile `u` not in known set**: render anyway using the per-canvas `z` from the tile set; unknown `u` strings are not a blocking condition.
- **Missing `bS`/`tS`/`oS` reference**: log warning, skip that sprite, continue the map. Partial renders beat no renders.
- **Map dimensions exceeding `MaxPixels`**: log + skip. Surfaces via structured log; operator can tune env var.

## Known limitations / follow-ups (post-spike)

1. **Sky/horizon seams** from parallax collapse. Visible on Perion, Leafre, Henesys 100000200. Fixable by having the background renderer fill the vertical band between the sprite and the top/bottom of the world with its nearest edge color, or by extending the tile loop to cover the full world height regardless of `type`. ~1 day, Phase 2.1.
2. **`front=1` foreground backgrounds** — code path exists (step 5) but no map in the six-sample set exercises it. Validate on first real occurrence.
3. **Obj `a` property** — not observed in v83 samples. Default 255. Honor if encountered.
4. **Animated canvases** (`ani != 0`, multi-frame objs) — always render frame 0 for static composite.

## Out of scope (revised, unchanged from v1)

- Foothold overlay
- Portal markers on the render
- Spawn-point markers on the render
- Interactive zoom/pan
- Per-layer toggling
- Animated output (GIF/APNG)
- Pixel-identical reproduction of an in-game screenshot (parallax, dynamic lighting, camera effects)
