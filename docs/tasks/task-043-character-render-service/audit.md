# Plan Audit — task-043-character-render-service

**Plan Path:** docs/tasks/task-043-character-render-service/plan.md
**Audit Date:** 2026-05-01
**Branch:** task-043-character-render-service (worktree)
**Base Branch:** main (commit 284b1e64eb4d47942deec0672b30bf1b854d0f77)
**Head:** 1dccee14dcbf9fcd362e54307975ec16015d61d6 (34 commits)

## Executive Summary

Plan implementation is faithful and substantively complete. All 8 phases are implemented across 34 commits — every numbered `Task X.Y` in the plan has corresponding code, tests, and a commit. Five documented deviations (5.8 zero-stripped bodyTemplateId, 5.9 hat fixture id + Origin subtraction, 6.6 hair/face synthetic sprites, post-8.4 TS fix) are technical refinements aligning code with the actual on-disk extraction layout, not skipped work. The only unverified item is Task 8.5, which is explicitly manual / out of autonomous scope. Backend `go build ./...` and `go test ./... -count=1` PASS for both `libs/atlas-constants` and `services/atlas-wz-extractor`. atlas-ui `tsc -b` PASS; `vitest run` could not run in this environment due to a Windows-side `@rolldown/binding-win32-x64-msvc` mismatch (environmental, not a plan defect).

## Task Completion

The plan numbers tasks `X.Y` with sub-steps; each `Task X.Y` is treated as one row.

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1.1 | `item.IsTwoHanded` helper + test | DONE | `libs/atlas-constants/item/constants.go:183-203`, test `constants_test.go:39-65` (commit `0a311feed`) |
| 2.1 | Cross-language fixture file | DONE | `services/atlas-wz-extractor/.../characterrender/testdata/loadout-hashes.json` and copy at `services/atlas-ui/src/services/api/__tests__/loadout-hashes.json` (commit `91a2a1ce3`) |
| 2.2 | `CanonicalLoadoutString` (Go) | DONE | `characterrender/hash.go:13-35`; sort-invariance test `hash_test.go:80-87` (commit `5e3147f9c`) |
| 2.3 | `LoadoutHash` matches fixture | DONE | `hash.go:38-41`; fixture-driven tests `hash_test.go:43-78` (commit `573050603`) |
| 3.1 | Define metadata sidecar JSON shapes | DONE | `services/atlas-wz-extractor/.../image/zmap.go:15-94` defines `extractCharacterMaps`, `writeZmap`, `writeSmap` and writes `character-meta/{zmap,smap}.json` (commits `0b56097bd`, `3c5daa57b`) |
| 3.2 | Wire `extractCharacterMaps` into dispatch | DONE | `image/extract.go:37-38` (case `"base"` → `extractCharacterMaps`); test `image/zmap_test.go` exercises helpers without WZ binding |
| 4.1 | Sprite metadata sidecar struct + JSON writer | DONE | `image/character_parts.go:18-30` (`partSidecar`, `vec`, `templateInfo`); commit `fd95516d0` |
| 4.2 | Extract `info` block to `info.json` | DONE | `image/character_parts.go:54-94` (`extractInfoBlock` w/ Int + Short variants); test `character_parts_test.go` (commit `cbf14166f`) |
| 4.3 | Extract sprite + sidecar for one canvas | DONE | sidecar emission helpers in `character_parts.go` (commit `81eb21bab`) |
| 4.4 | Walk one img's stance/frame/part tree | DONE | `character_parts.go` walker w/ `stancesInScope` allow-list (commit `9e477a5ce`) |
| 4.5 | Wire `extractCharacterParts` into dispatch | DONE | `image/extract.go:33-37` combines icons + parts under the `"character"` case (commit `e2b9d86d1`) |
| 5.1 | `characterimage` package skeleton + errors | DONE | `characterimage/doc.go`, `errors.go` (sentinels: `ErrUnknownTemplateId`, `ErrInvalidStance`, `ErrFrameOutOfRange`, `ErrAssetsMissing`, `ErrCompositorInternal`) (commit `b0f19284b`) |
| 5.2 | Sidecar loaders + meta cache | DONE | `meta.go` (`LoadZmap`, `LoadSmap`, `LoadInfo`, `LoadPartMeta`), `meta_cache.go`, `meta_test.go` (commit `f6e464859`) |
| 5.3 | Joint resolution | DONE | `joints.go:18-24` `ResolveAnchor`, `joints_test.go` (commit `e09d55e28`) |
| 5.4 | Stance / frame validation | DONE | `stance.go` `ValidateStance`/`ValidateFrame`, `SupportedStances` (commit `d556bfc8c`) |
| 5.5 | Skin mapping + slot filtering | DONE | `skin.go` `MapInternalSkin` (0..10 → 2000..2013); `filter.go` `FilterEquipment` w/ pet/mount/cash drops (commit `430431bab`) |
| 5.6 | Two-handed override | DONE | `two_handed.go` `ResolveStance` consumes `item.IsTwoHanded` (commit `822cbdab6`) |
| 5.7 | Nearest-neighbor scaling | DONE | `scale.go:6-22` `NearestNeighborUpscale` (commit `b3895e31f`) |
| 5.8 | Compositor request + bare-body render | DONE (deviation 1) | `compositor.go:23-118`; `bodyTemplateId` strips zeros to mirror `normalizeId` extraction layout, `compositor.go:120-130` (commit `96861d8d3`) |
| 5.9 | Equipment blitting via joint tree | DONE (deviations 2,3) | `compositor.go:253-329` (`blitEquipment`); subtracts `meta.Origin` from anchor at `compositor.go:283-284`; test fixture `writeSyntheticHat` uses `"10000"` `compositor_test.go:102` (commit `ab79a35b1`) |
| 6.1 | JSON:API error body writer | DONE | `characterrender/error.go:27-41` `WriteError` (`application/vnd.api+json`); test `error_test.go` (commit `60761de14`) |
| 6.2 | Path parser | DONE | `path.go:24-54` `ParseRenderPath`; `path_test.go` (commit `85cac16ad`) |
| 6.3 | Query parser | DONE | `query.go:23-95` `ParseRenderQuery` w/ defaults stance=stand1, frame=0, resize=2 (commit `5e1194bcb`) |
| 6.4 | Atomic write | DONE | `write.go:13-44` `AtomicWritePNG` (mkdir → temp → sync → close → rename); `write_test.go` (commit `0183eefb2`) |
| 6.5 | Observability span + counters | DONE | `otel.go:8-39` (`character_render_total`, `character_render_errors_total`, `character_render_duration_ms`) (commit `5e00e19a5`) |
| 6.6 | Handler | DONE (deviation 4) | `handler.go:20-219`; `handler_test.go:1-162` adds hair/face synthetic sprites for compositor (commit `8e3541741`) |
| 6.7 | Route registration | DONE | `resource.go:13-30` registers `/wz/character/render/{tenant}/{region}/{version}/{hash}.png`; `main.go:8-66` constructs `Compositor` + `Handler` and adds initialiser (commit `9c8d2464f`) |
| 7.1 | nginx `try_files` block | DONE | `services/atlas-assets/nginx.conf:18-31` adds `^/(?<ctenant>...)/(?<chash>[a-f0-9]{16})\.png$` location with `@character_render` proxy to atlas-wz-extractor (commit `e230bf20e`) |
| 7.2 | Wipe rendered cache before extraction | DONE | `extraction/processor.go:55-58` (call), `extraction/processor.go:90-99` (helper); test `extraction/processor_test.go:106-122` `TestRunExtractionWipesCharacterCache` (commit `3c63b2b95`) |
| 7.3 | Whole-service build + test | DONE | `go build ./...` and `go test ./... -count=1` PASS for both modules (see Build & Test Results) |
| 8.1 | Add `js-sha256` dependency | DONE | `services/atlas-ui/package.json` adds `"js-sha256": "^0.11.1"`; `package-lock.json` updated (commit `4d20cdccd`) |
| 8.2 | New `characterRender.service.ts` | DONE | `services/atlas-ui/src/services/api/characterRender.service.ts:1-115`; tests `__tests__/characterRender.service.test.ts:1-85`; fixture `__tests__/loadout-hashes.json` (commit `cc7620167`) |
| 8.3 | Update `useCharacterImage` and `CharacterRenderer` | DONE | `useCharacterImage.ts:14` imports from `characterRender.service`; `CharacterRenderer.tsx:6` uses `characterToLoadout`; `OptimizedCharacterRenderer.tsx:8` likewise (commit `802145590`) |
| 8.4 | Purge `maplestory.service.ts` and `maplestory.io` references | DONE (post-fix 5) | `maplestory.service.ts` and its test deleted (`-1085` lines); plan-specified greps return zero hits in `services/atlas-ui/src/` and `public/`; follow-up TS fix added `compactRenderOptions` helper at `useCharacterImage.ts:20` and made `characterToMapleStoryData` accept tenant params with defaults at `maplestory.ts:351` (commits `82ce3a846`, `1dccee14d`) |
| 8.5 | Manual verification | DEFERRED | Explicit out-of-scope — requires running the full dev stack |

**Completion Rate:** 34/35 implementable tasks (97%); 1 deferred (Task 8.5 manual smoke).
**Skipped without approval:** 0
**Partial implementations:** 0

## Skipped / Deferred Tasks

- **Task 8.5 (Manual verification)** — explicitly deferred to humans operating the running stack: confirm no `maplestory.io` requests, cache hit on second load, slot dropping, and error states. No code consequence; the implementation paths Task 8.5 would observe are exercised by the Go and TS test suites.

## Build & Test Results

| Service / Module | Build | Tests | Notes |
|------------------|-------|-------|-------|
| `libs/atlas-constants` | PASS | PASS | `go test ./item/ -count=1` → `ok ... 0.002s` (covers `TestIsTwoHanded`) |
| `services/atlas-wz-extractor/atlas.com/wz-extractor` | PASS | PASS | `go test ./... -count=1` → `characterimage`, `characterrender`, `extraction`, `image`, `mapimage`, `wz/*`, `xml` all `ok`; no failures, no skips. |
| `services/atlas-ui` | PASS (`tsc -b`) | UNVERIFIED | `tsc -b` exits 0. `vitest run` aborted at startup with `Cannot find native binding ... @rolldown/binding-win32-x64-msvc` — Windows-side npm in WSL placed the wrong native binding for the Linux runtime. Environmental; reproducing locally requires `rm -rf node_modules package-lock.json && npm i` from a Linux node binary. Plan-affected files (`characterRender.service.test.ts`, `CharacterRenderer.test.tsx`, etc.) compile cleanly under `tsc`. |

## Documented Deviations (verified)

1. **Task 5.8 — `bodyTemplateId` zero-stripping** — `compositor.go:120-130` mirrors `normalizeId` so `wzSkin=2000` resolves to `"2000"`, matching how `extractEquipmentIcons`/`extractCharacterParts` write directories. Plan's verbatim form would have produced `"00002000"` and 404'd. Verified.
2. **Task 5.9 — `writeSyntheticHat` fixture id** — `compositor_test.go:102,139` writes hat under `"10000"` rather than `"00010000"`. Same stripping rationale. Verified.
3. **Task 5.9 — `meta.Origin` subtraction** — `compositor.go:283-284` and the analogous body branch at `compositor.go:164-165` subtract `meta.Origin` from the resolved anchor before drawing, so `drawPart` lands the sprite top-left where the joint origin should sit. Verified by `compositor_test.go` synthetic-hat-over-body test passing.
4. **Task 6.6 — synthetic hair/face fixtures** — `handler_test.go` writes hair/face PNGs that the compositor pipeline requires; the verbatim plan omitted them. Verified by `characterrender` test pass.
5. **Post-8.4 TS fix** — `useCharacterImage.ts:20` `compactRenderOptions` helper plus `maplestory.ts:351` defaulted tenant params; resolved 6 TS errors introduced by Task 8.3 making `MapleStoryCharacterData` tenant fields required. Verified by `tsc -b` passing.

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE (pending Task 8.5 manual smoke and a clean re-run of `npm run test` from a Linux-native node install)

## Action Items

1. From a Linux-native node toolchain (i.e. not the Windows-side npm reachable via WSL), run `cd services/atlas-ui && rm -rf node_modules package-lock.json && npm i && npm run test` to close the JS test loop the audit could not run.
2. Execute Task 8.5 manual smoke per its checklist (no `maplestory.io` requests; cache hit on second load; slot dropping; error states).
3. Out-of-scope follow-up: residual `maplestory.io` references remain in `services/atlas-ui/CLAUDE.md`, `services/atlas-ui/docs/api-integration-patterns.md`, and `services/atlas-ui/tests/integration/components/features/CharacterRendering.integration.test.tsx`. Plan Task 8.4 only required `src/` and `public/` to be clean (both are), so this is non-blocking, but a docs/integration-test cleanup PR would close out the migration narrative.

---

# Frontend Audit — task-043-character-render-service (atlas-ui)

- **Audit Scope:** atlas-ui changes between `284b1e64eb4d47942deec0672b30bf1b854d0f77` and `1dccee14dcbf9fcd362e54307975ec16015d61d6`
- **Guidelines Source:** `.claude/skills/frontend-dev-guidelines/`
- **Date:** 2026-05-01
- **Build:** PASS (`tsc -b && vite build`, "built in 1.28s")
- **Tests:** 683 passed, 0 failed (71 test files, vitest run)
- **Overall:** NEEDS-WORK

## Build & Test Results

```
$ npm run build         # tsc -b && vite build
... built in 1.28s

$ npm test              # vitest run
 Test Files  71 passed (71)
      Tests  683 passed (683)
   Duration  9.81s
```

(Run via `PATH=/home/tumidanski/.nvm/versions/node/v22.22.2/bin`; the default `npm` on PATH is the Windows `/mnt/c/Program Files/nodejs/npm`, which cannot execute against the WSL UNC path. Closes Action Item 1 from the plan-adherence audit above — both build and test pass cleanly when invoked from a Linux-native node toolchain.)

## Acceptance Gate

`grep -rn "maplestory.io" services/atlas-ui/src/ services/atlas-ui/public/` returns zero matches. PASS.

## File Inventory

| File | Classification | Change |
|------|----------------|--------|
| `services/atlas-ui/src/services/api/characterRender.service.ts` | Service (pure URL/hash) | added |
| `services/atlas-ui/src/services/api/__tests__/characterRender.service.test.ts` | Test | added |
| `services/atlas-ui/src/services/api/__tests__/loadout-hashes.json` | Test fixture | added |
| `services/atlas-ui/src/services/api/index.ts` | Service barrel | modified |
| `services/atlas-ui/src/services/api/maplestory.service.ts` | Service | deleted |
| `services/atlas-ui/src/services/api/__tests__/maplestory.service.test.ts` | Test | deleted |
| `services/atlas-ui/src/types/models/maplestory.ts` | Type | modified |
| `services/atlas-ui/src/lib/hooks/useCharacterImage.ts` | Hook | modified |
| `services/atlas-ui/src/components/features/characters/CharacterRenderer.tsx` | Component | modified |
| `services/atlas-ui/src/components/features/characters/OptimizedCharacterRenderer.tsx` | Component | modified |
| `services/atlas-ui/src/components/features/characters/__tests__/CharacterRenderer.test.tsx` | Test | modified |
| `services/atlas-ui/src/lib/utils/character-cache-sw.ts` | Utility | modified (only `generateCharacterImageUrls` body) |
| `services/atlas-ui/src/lib/utils/maplestory.ts` | Utility | modified (signature widened) |
| `services/atlas-ui/src/lib/utils/__tests__/maplestory.test.ts` | Test | modified |
| `services/atlas-ui/public/sw-character-cache.js` | Service worker (JS) | modified |
| `services/atlas-ui/package.json`, `package-lock.json` | Manifest | modified (`js-sha256@^0.11.0`) |

## Anti-Pattern Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-01 | No `any` type | FAIL | `services/atlas-ui/src/services/api/__tests__/characterRender.service.test.ts:32` uses `row.stance as any`. The guideline forbids `as any` outright; cast through `Stance` (e.g. `row.stance as Stance`) instead. |
| FE-02 | No manual class concatenation | PASS (in scope) | New/modified component code uses `cn()` (e.g. `CharacterRenderer.tsx:252,280,318,326`). Pre-existing template-literal `className={\`...\`}` at `OptimizedCharacterRenderer.tsx:97,316` was not touched by this branch. |
| FE-03 | No direct API client calls in components | PASS | Components import only from `@/services/api/characterRender.service` (`CharacterRenderer.tsx:6`, `OptimizedCharacterRenderer.tsx:8`); the only `@/lib/api/client` reference is inside a Vitest mock at `__tests__/CharacterRenderer.test.tsx:15`. |
| FE-04 | No inline Zod schemas in components | N/A | No Zod schemas in any in-scope file. |
| FE-05 | No spinners for content loading | PASS | `grep -nE 'animate-spin'` over the in-scope files returns zero results; loading uses `<CharacterRendererDetailSkeleton>` (`CharacterRenderer.tsx:220,232`). |
| FE-06 | No hardcoded colors | FAIL (pre-existing, not introduced) | `CharacterRenderer.tsx:252,255,259,295,347` (`bg-gray-100`, `border-gray-300`, `text-gray-400`, `text-gray-500`, `bg-green-500 text-white`) and `OptimizedCharacterRenderer.tsx:97,98,103,108,342` (`border-red-300 bg-red-50`, `text-red-600`, `bg-red-100 hover:bg-red-200`, `text-red-500`, `text-gray-600`). Branch diff did not add or remove these lines. |
| FE-07 | No state mutation | PASS | `characterRender.service.ts:52` builds a sorted copy via `[...items].sort(...)` rather than mutating the argument. |
| FE-08 | No default exports for components | FAIL (pre-existing) | `OptimizedCharacterRenderer.tsx:350` (`export default OptimizedCharacterRenderer;`) and `character-cache-sw.ts:276` (`export default characterCacheManager;`) — both untouched by this branch. |
| FE-09 | Tenant guard in hooks | PASS | `useCharacterImage` exposes `enabled` (default `true`) so the consumer guards. The renderer wires `enabled: !!activeTenant && (priority || !lazy || shouldLoad)` (`CharacterRenderer.tsx:114`) and short-circuits with a skeleton when `activeTenant` is null (`CharacterRenderer.tsx:217`). The preloader/cache helpers are gated upstream: `OptimizedCharacterRenderer.tsx:157,289` only call `preloadImages` when `activeTenant` is truthy. |
| FE-10 | Tenant ID in query keys | PASS | `generateQueryKey` (`useCharacterImage.ts:66-90`) bakes `character.tenant`, `character.region`, `character.majorVersion`, `character.minorVersion` into the canonical loadout string before hashing, so two tenants cannot collide. The four tenant fields are part of `MapleStoryCharacterData` (`types/models/maplestory.ts:108-115`) and are populated from `useTenant()` (`CharacterRenderer.tsx:89-92`). |
| FE-11 | Error handling with `createErrorFromUnknown` | PASS (in scope) | The new service has no `.catch(`. The hook surfaces errors through React Query (`useCharacterImage.ts:202-206`), and the renderer classifies `queryError` via a typed helper (`CharacterRenderer.tsx:150-190`). |

## Architecture Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-12 | JSON:API model shape | PASS | `MapleStoryCharacterData` is an internal DTO (not a JSON:API model). Consumers like `CharacterRenderer.tsx:79-93` are still fed by `Character` (JSON:API `{ id, attributes }`) and `Asset` (JSON:API). The new service consumes `Character` via `character.attributes.skinColor` etc. (`characterRender.service.ts:107-109`). |
| FE-13 | Service extends `BaseService` (when applicable) | N/A | `characterRender.service.ts` is a pure URL/hash module with no HTTP calls. The deleted `maplestory.service.ts` is gone. The barrel re-exports the helpers via `services/api/index.ts:99` (`export * from './characterRender.service';`). |
| FE-14 | Query key factory uses `as const` | WARN | `useCharacterImage.ts` does not export a hierarchical `characterImageKeys` factory; it uses an ad-hoc `['character-image', loadoutHash(canonical)]` array (`useCharacterImage.ts:89`). Cache invalidation in `invalidate` and `clearCache` keys off `['character-image']` plus a stringified `character.id` (`useCharacterImage.ts:269-272,386-389`) — but actual cache entries are keyed by hash, so those invalidation paths cannot match an existing entry. Pre-existing bug that the rename did not fix. |
| FE-15 | Forms use `react-hook-form` + `zodResolver` | N/A | No forms in scope. |
| FE-16 | Schema in `lib/schemas/` with inferred type | N/A | No Zod schemas in scope. |

## Testing Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-17 | Tests exist for changed components | PASS | `services/atlas-ui/src/services/api/__tests__/characterRender.service.test.ts` covers `canonicalLoadoutString`, `loadoutHash`, `generateCharacterUrl`, `filterEquipment` against an externalized fixture (`loadout-hashes.json`) shared with `atlas-wz-extractor`. `services/atlas-ui/src/components/features/characters/__tests__/CharacterRenderer.test.tsx` covers the renderer end-to-end with 21 cases. |
| FE-18 | Mocks updated when services changed | PASS | `__tests__/CharacterRenderer.test.tsx:15-19,22-39` add new mocks for `@/lib/api/client` (because `TenantProvider` now lives in the wrapper) and `@/services/api` (so `tenantsService.getAllTenants` resolves with a real tenant). The old `__tests__/maplestory.service.test.ts` was removed alongside the deleted service module. |

## Spot-checks Requested by Reviewer

- **`exactOptionalPropertyTypes` discipline.** `compactRenderOptions` (`useCharacterImage.ts:20-26`) builds the `RenderOptions` object only with keys whose values are not `undefined`, which is the correct pattern for `exactOptionalPropertyTypes`. The hook also avoids assigning `undefined` to `mergedOptions.stance` etc. by using guarded `if` blocks (`useCharacterImage.ts:166-170,336-340,432-436`). Note: `tsconfig.app.json` has `exactOptionalPropertyTypes` OFF per `services/atlas-ui/CLAUDE.md`, so the build does not enforce this; the helper is forward-compatible defensive programming. PASS.
- **`js-sha256` synchronous import.** `import { sha256 } from 'js-sha256'` (`characterRender.service.ts:1`) is sync, so each render that calls `generateCharacterUrl` runs SHA-256 over the canonical string. The hash is computed inside the React Query `queryFn` (`useCharacterImage.ts:139-185`), so it runs at most once per cache miss; subsequent renders read the cached `imageUrl`. `generateQueryKey` (`useCharacterImage.ts:66-90`) ALSO calls `loadoutHash` on every render to build the cache key — that is the real hot path. For typical loadouts (~20 items, short strings) this is sub-millisecond, but on a render-heavy page like the gallery (`OptimizedCharacterRenderer.tsx:275`) it runs once per character per render. Acceptable; flag as follow-up if the gallery profiles hot.
- **Test wrapper hygiene.** The mocks at `__tests__/CharacterRenderer.test.tsx:15-45` stub: (1) `@/lib/api/client` so `api.setTenant` does not crash inside `TenantProvider`'s effect; (2) `@/services/api` so `tenantsService.getAllTenants` resolves with a synthetic tenant; (3) `@/lib/hooks/useCharacterImage` so the renderer never exercises the real query path; (4) `useLazyLoad` so visibility gating returns true. The shape of the synthetic tenant matches `Tenant` (`id`, `type`, `attributes.{region,majorVersion,minorVersion,name}`) so `useTenant()` returns a populated value and the renderer's `!activeTenant` guard short-circuits to the success branch. The mocks DO hide the real `generateCharacterUrl` plumbing, but parity is enforced by the parallel fixture-based test in `characterRender.service.test.ts` against `loadout-hashes.json`, so coverage is not lost. PASS.
- **No reintroduction of `useState/useEffect` for server state.** All new server state lives behind React Query in `useCharacterImage`. The `useState` calls in the renderer (`CharacterRenderer.tsx:63-65`) hold local UI state (fallback-image flag, `imageLoaded`, manual retry counter) — not server data. PASS.
- **Multi-tenancy: `useTenant()` is the single source of truth.** Tenant fields on `MapleStoryCharacterData` are populated only from `activeTenant` (`CharacterRenderer.tsx:89-92`, `OptimizedCharacterRenderer.tsx:172-175,303-306`). No hardcoded tenant/region/version anywhere in the in-scope source. The `region` / `majorVersion` props that previously existed on `CharacterRenderer` were removed (diff hunk at lines 60-61); they remain as ignored props on `OptimizedCharacterRenderer`'s API for back-compat (`OptimizedCharacterRenderer.tsx:67-68,137-138`) but are NOT consulted for tenant resolution. PASS.

## Frontend Summary

### Blocking (must fix)
- **FE-01** — `services/atlas-ui/src/services/api/__tests__/characterRender.service.test.ts:32` uses `row.stance as any`. Replace with `row.stance as Stance` (or tighten the fixture row type to constrain `stance`).

### Non-Blocking (should fix)
- **FE-06** — `CharacterRenderer.tsx:252,255,259,295,347` and `OptimizedCharacterRenderer.tsx:97,98,103,108,342` hardcode Tailwind palette colors (`bg-gray-100`, `text-gray-500`, `bg-green-500`, `text-red-600`, etc.). Pre-existing; out of scope for this branch but worth a follow-up since the renderer is the active surface area.
- **FE-08** — `OptimizedCharacterRenderer.tsx:350` and `character-cache-sw.ts:276` use `export default`. Pre-existing.
- **FE-02** — `OptimizedCharacterRenderer.tsx:97,316` uses template-literal `className` concatenation instead of `cn()`. Pre-existing.
- **FE-14** — `useCharacterImage.ts` lacks a `characterImageKeys` factory. The `invalidate` (`useCharacterImage.ts:269-272`) and `clearCache(characterId)` (`useCharacterImage.ts:386-389`) helpers key off `['character-image', character.id.toString()]`, but cache entries are keyed by `['character-image', loadoutHash]`, so those invalidation calls cannot hit any entry. Pre-existing bug that this branch's rename did not address.
