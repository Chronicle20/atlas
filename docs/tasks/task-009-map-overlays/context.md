# Task-009 Map Image Overlays — Context

**Last Updated: 2026-04-18**

---

## Source Documents

| Doc | Purpose |
|---|---|
| `docs/tasks/task-009-map-overlays/prd.md` | Canonical requirements — overlay layer, marker visual language, hover-coordination rules, acceptance criteria. |
| `docs/tasks/task-009-map-overlays/ux-flow.md` | Hover scenarios, interaction-rule table, edge cases, visual states. |
| `docs/tasks/task-009-map-overlays/plan.md` | Implementation phases A–G, task list, risk table. |
| `docs/tasks/task-008-map-detail-redesign/prd.md` | Predecessor task — defined `MapDetailPage` layout this task builds on. |
| `docs/tasks/task-008-map-detail-redesign/render-pipeline.md` | Confirms `render.png` dims === world rect dims (the cross-service invariant we exploit). |

---

## Key Discovery (read this first)

**`mapArea` already exists on the wire.** PRD §4.1 specifies a new `bounds` attribute and migration. During planning we found that `services/atlas-data/atlas.com/data/map/rest.go:27` already emits `mapArea: { x, y, width, height }`, populated by `reader.go:179-216` using the same VR → miniMap precedence the renderer uses. **Use `mapArea`, not a new field.** The plan converts `MapArea` from a value type to a pointer so the synthetic `1 << 18` fallback (`reader.go:198-204`) becomes `null` in the JSON — the UI's signal for "no real bounds, no overlay."

Save: a migration, a backfill, four nullable columns, and a transformer change. Cost: one struct field type change + one `getMapArea()` return type change.

---

## Key Files (existing, to read before editing)

### Backend — `services/atlas-data/atlas.com/data/map/`

| File | Why it matters |
|---|---|
| `rest.go:27` | `RestModel.MapArea` field. Change from value to pointer. |
| `rest.go:269` | `RectangleRestModel` shape — unchanged. |
| `reader.go:179-216` | `getMapArea()` bounds-resolution precedence. Phase A.2 changes return type to `*RectangleRestModel` and the `1<<18` branch returns `nil`. |
| `model.go` | Domain model + builder. Field becomes nilable per immutable-model pattern. |

### Frontend — `services/atlas-ui/src/`

| File | Why it matters |
|---|---|
| `pages/MapDetailPage.tsx` | Hosts the page composition. Wrap children below `MapHeader` in `<HoverHighlightProvider>`; pass entity arrays into `MapImagePanel` so the overlay can render. |
| `components/features/maps/MapImagePanel.tsx` | Container refactor target. When `mapArea` is present and state is `"render"`, switch from `object-contain max-h-[320px]` to a wrapper with `style={{ aspectRatio }}` + `object-cover`, and mount `<MapImageOverlay>` as a sibling. All other states preserved. Dialog gets the same treatment for the natural-size view. |
| `components/features/maps/MapEntitySummary.tsx` | Add row-level `onPointerEnter/Leave` handlers wired to `useHoverHighlight`. Apply highlight styles. |
| `components/features/maps/MapDetailTabs.tsx` | Same as summary, but per-row in each of the three tab tables (portals/monsters/reactors). |
| `services/api/maps.service.ts:6-9` | Extend `MapAttributes` with `mapArea?: { x, y, width, height } | null`. Wire payload already has it. |
| `services/api/map-entities.service.ts` | Entity types (`MapPortalData`, `MapNpcData`, `MapMonsterData`, `MapReactorData`) — already include `x` and `y`, no changes needed. |
| `lib/utils/asset-url.ts` | No change — overlay URLs not introduced. |

### New files

| Path | Purpose |
|---|---|
| `services/atlas-ui/src/lib/utils/map-overlay.ts` | `worldToOverlayPercent` pure helper + `MapBounds` type. |
| `services/atlas-ui/src/lib/utils/__tests__/map-overlay.test.ts` | Helper unit tests (positive/negative origin, edge cases). |
| `services/atlas-ui/src/components/features/maps/HoverHighlightContext.tsx` | Context, provider, `useHoverHighlight` hook. Discriminated `HoverTarget` union per PRD §4.5; matching rules per PRD §4.6. |
| `services/atlas-ui/src/components/features/maps/__tests__/hover-highlight.test.tsx` | Context match-rule tests. |
| `services/atlas-ui/src/components/features/maps/MapImageOverlay.tsx` | Marker layer (`absolute inset-0 pointer-events-none` + per-marker `<button>` with `Tooltip`). |

---

## Key Decisions (from PRD + planning)

1. **Reuse `mapArea` instead of new `bounds` field.** See "Key Discovery" above. Smaller backend change, no migration.
2. **Pointer-ize `MapArea` → null sentinel for "no bounds."** Replaces ambiguous `1<<18` huge-rectangle fallback.
3. **Overlay only when `mapArea` non-null AND panel state is `"render"`.** Minimap fallback hides overlay (PRD §2 non-goal). Placeholder hides overlay.
4. **Container refactor is conditional.** Only the `"render"` + bounds-present path uses the new `aspectRatio` wrapper. All other states keep today's `object-contain max-h-[320px]` so non-render fallbacks look identical.
5. **Hover state is transient and context-local.** No URL encoding, no React Query side-effect, no persistent selection. Cleared on `Dialog` close.
6. **Per-template highlight for monsters/NPCs.** Hovering one marker highlights all sibling markers (same template) + the deduped summary row + matching detail-tab rows. Hovering the per-template summary row highlights all sibling markers.
7. **Marker shapes/colors fixed.** Portal = emerald diamond, NPC = sky circle, monster = rose dot, reactor = amber square. White border + yellow ring on highlight.
8. **All portals rendered.** Including `targetMapId === 999999999` and any portal type. (Differs from `ConnectedMapsRow` which filters NONE — that's a different concern.)
9. **Touch shows markers but skips hover.** No tap-to-highlight; markers visible so the "where things are" cue isn't lost.
10. **Dialog gets overlay + hover, no persistent selection.** Closing the dialog clears any active hover.
11. **No new dependencies.** Pure additive React work + one Go field type change.

---

## Cross-Service Invariant

For any map where `atlas-data` returns non-null `mapArea`, the corresponding `render.png` produced by `atlas-wz-extractor` MUST be `mapArea.width × mapArea.height` pixels.

This holds by construction because both services use the same VR → miniMap precedence. Phase G.1 verifies on Henesys, Perion, Ellinia. If it fails, fall back to using `img.naturalWidth/Height` for the percentage denominator instead of `mapArea.width/height` — but expect this to pass.

---

## Dependencies & Integration Points

### Upstream (unchanged, relied on)
- `atlas-data` `GET /api/data/maps/{id}` — already includes `mapArea`; only nullification is added.
- `atlas-data` `.../portals`, `.../npcs`, `.../monsters`, `.../reactors` — entity coordinates unchanged.
- `atlas-wz-extractor` `render.png` output — unchanged; relied on for the cross-service invariant.
- `TenantProvider` in `atlas-ui` — unchanged.
- shadcn primitives: `Tooltip`, `Card`, `Dialog`, `Button` — all already imported by the page.

### Downstream (consumers)
- Operators on `/maps/:id`. No other consumer.

---

## Open Questions (rolled forward from PRD §9)

| Question | Resolution path |
|---|---|
| Backfill strategy for `mapArea` | N/A — `mapArea` already exists; pointer-izing is a behavior change, no backfill needed. Existing maps re-resolve on next read. |
| NPC tooltip name resolution | `MapNpcData.attributes.name` already on the wire — just consume it. |
| Marker collision (stacked spawns) | Accepted v1 limitation. Future task can add cluster-spread or count badge. |
| Portal type-specific shapes | Accepted v1: one diamond shape regardless of `type`. Future task. |

---

## Test & Build Commands

### atlas-data
```bash
cd services/atlas-data/atlas.com/data
go test ./...
go build ./...
docker compose build atlas-data    # from repo root
```

### atlas-ui
```bash
cd services/atlas-ui
pnpm install
pnpm typecheck
pnpm test           # vitest run
pnpm build
pnpm dev            # for manual browser walkthrough
```

### Repo-root build sanity
```bash
docker compose build atlas-data atlas-ui
```

---

## Verification Map Set

For G.1 (cross-service invariant) and G.2 (manual walkthrough), use:

| Map ID | Name | Why |
|---|---|---|
| 100000000 | Henesys | Town with portals, NPCs, varied layout. |
| 102000000 | Perion | Town, vertical layout. |
| 101000000 | Ellinia | Tall vertical map (5146px) — stress-tests the aspect-ratio wrapper. |
| 100000200 | Henesys Hunting Ground I | Many monster spawns — stress-tests dense markers. |
| Any PQ map (e.g., 103000800 KPQ) | — | Reactors present (verify reactor markers). |

---

## Follow-Ups (explicitly deferred)

- **Marker cluster-spread or count badge** for stacked spawns at identical coords. Open if visual clutter becomes a complaint.
- **Per-portal-type shape variants** (spawn point vs hidden vs script). Open if operators want type at a glance.
- **Overlay on minimap fallback** with `mag` scale. Out of scope per user decision.
- **Tap-to-highlight on touch.** Out of scope per user decision.

---

## Guardrails

- **CLAUDE.md workflow rule**: do not start implementing until explicitly approved. This plan is the "plan" phase.
- **Don't modify `getAssetIconUrl` or any task-008 component contracts unnecessarily.** Phase D's container refactor is the only structural change to a task-008 component, and it's behind a conditional.
- **No new dependencies** — both projects already include everything needed.
- After backend change, build atlas-data docker image. After frontend change, run `pnpm typecheck` + `pnpm build`.
- When updating tracking docs, locate them via Glob/Grep — don't assume paths.
