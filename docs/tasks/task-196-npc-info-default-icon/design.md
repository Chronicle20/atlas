# task-196 — NPC icon extraction: prefer `info/default` over a placeholder `stand/0`

**Status:** design approved, not yet implemented
**Branch / worktree:** `task-196-npc-info-default-icon`
**Date:** 2026-08-06

## Problem

`icons.ExtractNpcIcon` picks the wrong canvas for a subset of NPCs. It resolves
the likeness through `findStandCanvas`, whose precedence is
`stand/*` → `move/*` → first canvas under any top-level sub-property. A number
of NPCs ship a **1-pixel-wide placeholder** `stand/0` and carry their real
likeness at `info/default`. The placeholder wins, so those NPCs get a sliver
`icon.png` uploaded to the assets bucket and rendered in the atlas-ui NPC
header.

Verified against the GMS 83.1 extracted dump,
`Npc.wz/1101000.img.xml`:

```xml
<imgdir name="1101000.img">
  <imgdir name="info">
    <canvas name="default" width="129" height="86">   <!-- the real likeness -->
  </imgdir>
  <imgdir name="stand">
    <canvas name="0" width="1" height="60">           <!-- placeholder; wins today -->
  </imgdir>
</imgdir>
```

`1101001.img` has the same shape — `info/default` 131×145 against a `stand/0`
of 1×121.

Note that the generic "first canvas under any sub" fallback *would* find
`info/default`, but it is unreachable whenever a `stand` dir exists at all,
which is precisely the broken case.

## Scope, measured

Census of all 1620 `.img` entries in GMS 83.1 `Npc.wz`. "Real `stand/0`" means
both dimensions greater than 8px.

| `info/default` | real `stand/0` | `info/link` | count | today |
|---|---|---|---|---|
| no | yes | no | 1211 | correct |
| no | no | no | 302 | no likeness anywhere — intentionally invisible `hideName` 1×1 stubs (`9901xxx` rank boards, `1063015`) |
| yes | yes | no | 41 | works, but resolves to the field sprite rather than the portrait |
| no | no | yes | 33 | correct — `info/link` resolves, since these carry no `stand` of their own |
| yes | no | no | 31 | **broken** |
| yes | no | yes | 2 | **broken** |

So 33 NPCs are visibly broken and 41 change appearance under this design.

Two findings that constrain the fix:

- **`Mob.wz` contains zero `info/default` nodes** (0 of 1564). This is an
  NPC-only concern, so the shared finder must not change for mobs.
- **`1209003.img` has a *top-level* `default` imgdir** — a 14-frame 192×84
  animation — that is *not* `info/default`. Any rule keyed on the name
  `default` alone would misfire here. That NPC has a healthy 192×84 `stand/0`
  and must keep it.

## Decision

Approach **A + C**, chosen over a size-threshold alternative:

- **A — `info/default` always wins when present.** One structural rule, no
  pixel heuristics.
- **C — NPCs get their own finder;** the shared `findStandCanvas` is left
  alone so mobs and reactors are unaffected by construction rather than by
  observation.

The rejected alternative was "use `info/default` only when `stand/0` looks
degenerate." It preserves the 41 unchanged, but the degeneracy threshold is
load-bearing and already wrong at the edges: at ≤2px it catches 13 NPCs, at
≤8px it catches 31, and `2012023` sits in the gap with a 4×4 `stand/0` beside a
real 161×86 `info/default`. An NPC that ships a 1×60 stand is telling us the
stand is not the likeness; that signal is structural, and encoding it as a
pixel count invents a boundary the data does not have.

Accepted cost: 41 already-working icons change from the full-body field sprite
to the `info/default` portrait crop on the next ingest. These were not
rendered and compared before the decision.

## Design

### Change 1 — NPC-specific canvas finder

`libs/atlas-wz/icons/extract.go`. `ExtractNpcIcon` stops sharing
`findStandCanvas` with mobs:

```go
func ExtractNpcIcon(f *wz.File, id uint32) (image.Image, error) {
	return extractEntityIcon(f, id, findNpcCanvas)
}

// findNpcCanvas prefers info/default — the static likeness carried by NPCs
// whose stand animation is a 1-px placeholder — then falls back to the
// shared stand/move ordering.
func findNpcCanvas(props []property.Property) *property.CanvasProperty {
	if info := findSub(props, "info"); info != nil {
		if cp := findSubCanvas(info.Children(), "default"); cp != nil {
			return cp
		}
	}
	return findStandCanvas(props)
}
```

No new helpers. `findSub` and `findSubCanvas` already exist, and
`findSubCanvas` matches only a `*property.CanvasProperty` — which is exactly
what excludes the `1209003` top-level sub-dir case.

`ExtractMobIcon` and `ExtractReactorIcon` are untouched, and `findStandCanvas`
keeps its current body.

`extractEntityIcon` already threads the finder through to
`resolveLinkedCanvas`, so linked NPCs inherit the same precedence with no
further change. The 2 NPCs carrying both `info/default` and `info/link`
resolve against their own `info/default` first — the link is a fallback, not
an override.

### Change 2 — none

`services/atlas-data/atlas.com/data/data/workers/npc.go` calls
`ExtractNpcIcon` and overwrites `<prefix>/npc/<id>/icon.png`. No worker,
storage, REST, or atlas-ui change.

### Data flow and rollout

Icons are baked at WZ-ingest into the assets bucket under
`<scope>/regions/<region>/versions/<major>.<minor>/npc/<id>/icon.png`
(`minioAssetPrefix`, `workers/runtime.go`). `baseline/publish.go` tars only the
database into `BucketCanonical`; icons are not carried in the baseline dump.

Consequence: **the code change alone fixes nothing already deployed.** Existing
tenants keep their stale sliver icons until `Npc.wz` is re-ingested, per tenant
per version. Re-ingest is the deploy step.

### Error handling

Unchanged. A nil return from `findNpcCanvas` still falls through to
`resolveLinkedCanvas` and then `ErrNotFound`, and the worker's existing
`if err != nil || icon == nil { continue }` skips the ~302 NPCs with no
likeness anywhere. Those remain 1×1 — they are intentionally invisible in
game, and this design does not attempt to give them an appearance.

## Testing

Fixture tests in `libs/atlas-wz/icons/extract_test.go`, built with
`wztest.NewBuilder()`:

1. **The fix** — img with `info/default` (payload A) and `stand/0` (payload B);
   `ExtractNpcIcon` returns A.
2. **No regression** — img with `stand/0` only; returns B.
3. **Mobs unaffected** — the same both-canvases tree through `ExtractMobIcon`;
   returns B.
4. **Sub-dir exclusion** — a top-level `default` imgdir alongside a real
   `stand/0`; returns the stand canvas. Guards the `1209003` shape.
5. **Link precedence** — `info/default` wins over `info/link`.

`wztest.Canvas` hardcodes 1×1 / format 2, so the two canvases cannot be told
apart by dimensions. The tests distinguish them by decoded pixel value, which
requires the payloads to be zlib-compressed 4-byte BGRA. That is a test-local
helper, not a change to the `wztest` builder.

The existing `TestPublicSurfaceExists` and `TestNormalizeId` stay as they are.

## Verification

Per `CLAUDE.md`. `libs/atlas-wz` is the only module with edited files, but
`services/atlas-data` consumes it through `go.work` and so is verified too:

- `go test -race ./...`
- `go vet ./...`
- `go build ./...`
- `tools/lint.sh --check`
- `tools/redis-key-guard.sh`, `tools/goroutine-guard.sh`

No `go.mod` changes are expected, so `docker buildx bake` is not triggered by
the mandatory-bake rule. If a `go.mod` does end up touched,
`docker buildx bake atlas-data` becomes required.

## Open items

- The 41 both-present NPCs change appearance and have not been visually
  compared. If any regress noticeably in the UI header, the fallback is to
  narrow the rule rather than reintroduce a pixel threshold.
- Only GMS 83.1 was censused. Other version columns are assumed to share the
  structure but were not measured. **Accepted by decision (2026-08-06)** — the
  design proceeds on this assumption rather than blocking on a second version's
  WZ. The rule is purely structural, so a version that lacks `info/default`
  simply falls through to the existing `stand`/`move` behavior; the risk is a
  missed fix on another version, not a regression.
