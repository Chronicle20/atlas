# Task-008 Map Detail Redesign — Context

**Last Updated: 2026-04-18**

---

## Source Documents

| Doc | Purpose |
|---|---|
| `docs/tasks/task-008-map-detail-redesign/prd.md` | Canonical requirements — layout, phases, acceptance criteria. Read in full before implementing. |
| `docs/tasks/task-008-map-detail-redesign/ux-flow.md` | Visual ASCII layout, interaction notes, dedup snippets, progressive-render timeline, error table. |
| `docs/tasks/task-008-map-detail-redesign/render-pipeline.md` | Phase 2 algorithm, verified against a working spike. Z-order rule, blit math, parallax decision. |

## Existing Spike

`services/atlas-wz-extractor/atlas.com/wz-extractor/cmd/map-render-spike/` — standalone Go program that rendered Henesys, Perion, Ellinia, El Nath, Leafre, and Henesys hunting ground correctly from real v83 `Map.wz`. First-try output, no iteration needed. Promote into `mapimage/` during Phase 2A.

---

## Key Files (existing, to read before editing)

### Frontend — `services/atlas-ui/src/`

| File | Why it matters |
|---|---|
| `pages/MapDetailPage.tsx` | Current page to be rewritten. Contains the existing hook wiring + tab tables that the new page keeps (portals / monsters / reactors). |
| `components/map-cell.tsx:43-45` | Canonical `TooltipContent copyable` usage — reuse exactly in `MapHeader`. Also the name-resolution path (with `mapNameCache`) that `ConnectedMapsRow` leverages. |
| `lib/utils/asset-url.ts` | `getAssetIconUrl` contract — extend by adding `'map'` to the category union and a sibling `getMapImageUrl`. Don't fork `getAssetIconUrl`. |
| `components/providers/*` or similar | `TenantProvider` — `MapImagePanel` reads tenant/region/version from here, same as icon callers. |
| `pages/NpcDetailPage.tsx`, `pages/MonsterDetailPage.tsx` | Confirm route params for the summary-row links. PRD §9 flags `/monsters/:id` vs `/mobs/:id` as an open question. |
| `components/ui/{Card,Badge,Tabs,Tooltip,Skeleton}.tsx` | shadcn primitives used throughout the new components. |

### Extractor — `services/atlas-wz-extractor/atlas.com/wz-extractor/`

| File | Why it matters |
|---|---|
| `image/extract.go` | Shape reference for the Phase 1A minimap extractor. Mirror how entity-icon extraction walks the WZ tree and writes PNGs. |
| `extraction/processor.go` | Wire point for both Phase 1A (minimap) and Phase 2B (composite render). |
| `wz/canvas/decompress.go` | `Decompress(data, w, h, format, key) *image.NRGBA` — used by every sprite decode. No wrapping needed for `draw.Over`. |
| `cmd/map-render-spike/` | Source material for Phase 2A. Algorithm is verified; promote into `mapimage/` subpackage. |

### Data reference (read-only, do not modify)

| File | Why it matters |
|---|---|
| `services/atlas-data/atlas.com/data/map/reader.go:179-216` | Canonical bounds resolution (`VR*` precedence, `miniMap` fallback). Phase 2 `mapimage/bounds.go` mirrors this. |

---

## Key Decisions (from PRD + companion docs)

1. **Two-phase ship.** Phase 1 redesigns the UI and extracts minimaps only (low risk, small extractor change). Phase 2 adds the full-map composite renderer and swaps the image source. Both phases live under task-008.

2. **No maplestory.io.** External dependency rejected. All map images come from the tenant's own `Map.wz` via `atlas-wz-extractor` into the atlas-assets shared volume.

3. **Minimap as interim + fallback.** Phase 1 uses `minimap.png`. Phase 2 prefers `render.png` but falls back to `minimap.png` on 404, then to a placeholder.

4. **Static render, parallax collapsed.** Backgrounds render at `(x, y)` in world space (`rx/ry` ignored). Acceptable "horizon seam" artifact documented; fix deferred to Phase 2.1.

5. **Z-order: objs-first-then-tiles per layer.** Verified against v83 Henesys by the spike. Deviates from WzComparerR2's flat mesh sort, but correct for this data.

6. **No new dependencies.** Std lib only in Go (`image`, `image/draw`, `image/png`, `math`, `sort`). No new npm packages in `atlas-ui`.

7. **Renders produced at extraction time.** No on-demand HTTP rendering. Written as PNG into `{outputImgDir}/map/{mapId}/render.png` (or `minimap.png`) and served by existing atlas-assets nginx.

8. **Deterministic output.** Sorted sprite lists, no random seeds, fixed PNG compression — re-uploads do not churn unchanged maps. CI test byte-compares two renders of the same fixture.

9. **Safety cap.** Maps exceeding `MaxPixels` (default 16384×16384) are skipped + logged, not rendered.

10. **UI progressive render.** Header + image + tabs shell mount before any entity query resolves. Each summary sub-section and tab renders independently.

---

## Dependencies & Integration Points

### Upstream (unchanged, relied on)
- `atlas-data` REST endpoints: `GET /api/data/maps/{id}`, `.../portals`, `.../npcs`, `.../monsters`, `.../reactors`. No changes.
- atlas-assets nginx serving `/api/assets/{tenantId}/{region}/{version}/...`. No changes.
- `TenantProvider` in `atlas-ui` — existing multi-tenant context.

### Downstream (consumers)
- Operators browsing `/maps/:id` — primary user.
- No other services consume the `map/{mapId}/*.png` assets.

### Asset path contract
- Phase 1: `/api/assets/{tenantId}/{region}/{version}/map/{mapId}/minimap.png`
- Phase 2: `/api/assets/{tenantId}/{region}/{version}/map/{mapId}/render.png`
- Tenant UUID is part of the path — no cross-tenant leakage possible.

---

## Open Questions (from PRD §9)

| Question | Resolution path |
|---|---|
| Monster detail route — `/monsters/:id` or `/mobs/:id`? | Inspect `App.tsx` during task 1C.3. Match whatever exists. |
| Connected-map widget visual density | Ship "name only" per PRD. If thin in practice, open a follow-up to add an inline minimap thumbnail. Not blocking. |
| Render determinism guarantee | Task 2A.2 golden-PNG byte-compare test. Confirmed by sort + fixed PNG compression. |
| Global (`uuid.Nil`) tenant fallback for map images | Leaning yes for consistency. Confirm during Phase 2 wiring; document final choice in `render-pipeline.md` if diverging. |

---

## Test & Build Commands

### Extractor
```bash
cd services/atlas-wz-extractor/atlas.com/wz-extractor
go test ./...
go build ./...
docker compose build atlas-wz-extractor   # from repo root
```

### UI
```bash
cd services/atlas-ui
pnpm install
pnpm typecheck
pnpm build
pnpm dev   # for manual browser walkthrough
```

### Repo root docker check
```bash
docker compose build atlas-wz-extractor atlas-ui
```

---

## Follow-Ups (explicitly deferred)

- **Phase 2.1 horizon seam fix** — ~1 day, described in `render-pipeline.md` §"Known limitations". Open as separate task once Phase 2 ships.
- **WebP output** — optional, env-gated. Skip if it requires a new Go dependency.
- **`front=1` foreground validation** — local fix when first real map triggers the code path.
- **Tenant-global (`uuid.Nil`) fallback for `map/` assets** — decide during 2B; update docs if implemented.

---

## Guardrails

- CLAUDE.md workflow rule: **do not start implementing until explicitly approved.** This plan is the "plan" phase. Wait for go-ahead before editing code.
- After any shared-library change, build **all** affected services' docker images.
- Keep abstractions clean — do not have `atlas-ui` reach into extractor internals; go through the asset URL contract only.
- When updating tracking docs, locate them via Glob/Grep — do not assume paths.
