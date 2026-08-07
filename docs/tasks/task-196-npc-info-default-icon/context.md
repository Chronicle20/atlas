# task-196 — Context

Background an implementer needs before touching `libs/atlas-wz/icons`.

## What the code does today

`icons.ExtractNpcIcon(f *wz.File, id uint32)` resolves an NPC's likeness out of
a parsed `Npc.wz` and returns a decoded `image.Image`. It delegates to
`extractEntityIcon`, which:

1. Indexes every root-level image by `normalizeId(img.Name())` (strips `.img`
   and leading zeros).
2. Finds the image whose normalized name equals the decimal id.
3. Calls a `canvasFinder` on that image's properties.
4. If the finder returns nil, calls `resolveLinkedCanvas` to follow
   `info/link` (up to 5 hops) and re-runs the same finder on the linked image.
5. Decodes the chosen canvas via `decodeCanvas`.

`ExtractNpcIcon` and `ExtractMobIcon` currently share one finder,
`findStandCanvas`, whose precedence is:

```
stand/*  →  move/*  →  first canvas under any top-level sub-property
```

## Why that is wrong for some NPCs

A set of NPCs ship a **1-pixel placeholder** `stand/0` and carry the real
likeness at `info/default`. Because `stand` exists, step 3 returns the
placeholder and the generic third-tier fallback — which *would* find
`info/default` — is never reached.

`Npc.wz/1101000.img` (GMS 83.1), the canonical example:

```xml
<imgdir name="info">
  <canvas name="default" width="129" height="86">   <!-- real likeness -->
<imgdir name="stand">
  <canvas name="0" width="1" height="60">           <!-- placeholder; wins -->
```

## Two structural traps

**1. `Mob.wz` has zero `info/default` nodes** (0 of 1564 imgs). The fix must
not alter mob behavior. This is why the design gives NPCs their own finder
rather than editing `findStandCanvas` in place — mobs stay correct by
construction, not by coincidence.

**2. `1209003.img` has a *top-level* `default` imgdir** — a 14-frame 192×84
animation — which is **not** `info/default`:

```
imgdir '1209003.img'
  imgdir 'info'   → int 'hideName'          (no default canvas here)
  imgdir 'stand'  → canvas '0' 192x84       (healthy; must keep winning)
  imgdir 'default'→ canvas '0'..'13' 192x84 (a top-level animation)
```

A rule that matches the name `default` anywhere would break this NPC. The fix
matches only a `*property.CanvasProperty` named `default` among **`info`'s
children**, which excludes this shape structurally.

## Verified fixture mechanics

Both of these were confirmed by running a throwaway probe against the real lib
(the probe was deleted; the plan re-derives it as permanent tests):

- `wztest.NewBuilder().SetVersion(83).SetEncryption(crypto.EncryptionNone)`
  builds an archive that `wz.Open(logrus.StandardLogger(), path)` parses. Log
  line on success: `Detected version 83 (hash=1876) with encryption=None`.
- `wztest.Canvas(name, payload)` hardcodes **1×1, format 2 (`FormatBGRA8888`)**.
  The payload is raw bytes handed to `canvas.Decompress`, which tries zlib
  first (`isZlibHeader` wants `0x78` followed by `0x9C`/`0xDA`/`0x01`/`0x5E`).
  A `zlib.NewWriter` default-compression stream over 4 bytes satisfies this.
- Byte order is **B, G, R, A**, so payload `{0x11,0x22,0x33,0xFF}` decodes to
  `color.NRGBA{R:0x33, G:0x22, B:0x11, A:0xFF}`.

Because every fixture canvas is 1×1, canvases **cannot be told apart by
dimensions**. Tests distinguish them by decoded pixel value.

**Probe results, for reference:**

- Both-canvases NPC → returned `{R:102 G:85 B:68}` = the *stand* marker.
  Reproduces the bug.
- `info/link` NPC with no canvases of its own → returned the link target's
  `info/default`. Link resolution already works today (via the generic
  third-tier fallback) and must not regress.

## Where the icons go

`services/atlas-data/atlas.com/data/data/workers/npc.go` iterates
`file.Root().Images()`, calls `ExtractNpcIcon`, and PUTs
`<minioAssetPrefix>/npc/<id>/icon.png`. `minioAssetPrefix` is
`<scope>/regions/<region>/versions/<major>.<minor>` (`workers/runtime.go`).

`baseline/publish.go` tars only the database into `BucketCanonical` — icons are
**not** in the baseline dump. So the code change fixes nothing already
deployed; existing tenants need an `Npc.wz` re-ingest.

## Conventions that apply

- `libs/atlas-wz` is its own Go module, wired through `go.work`.
- No `go.mod` change is expected, so the mandatory `docker buildx bake` rule is
  not triggered. If a `go.mod` does get touched, `docker buildx bake atlas-data`
  becomes required.
- Existing tests in `icons/extract_test.go` (`TestPublicSurfaceExists`,
  `TestNormalizeId`) stay untouched.
