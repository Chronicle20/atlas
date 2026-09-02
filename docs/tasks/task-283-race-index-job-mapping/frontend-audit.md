# Frontend Audit — task-283-race-index-job-mapping (atlas-ui half)

- **Audit Scope:** atlas-ui files changed by commits `a821f0c43` and `b3c301eec` (per `git diff --name-only 9cd1ec5af..HEAD -- services/atlas-ui`)
- **Guidelines Source:** frontend-dev-guidelines skill, overridden where it conflicts with `services/atlas-ui/CLAUDE.md` (Vite/React Router/Vitest SPA, not Next.js/Jest — the skill's Next.js/Jest references do not apply to this codebase and are noted, not enforced, below)
- **Date:** 2026-08-28
- **Build:** PASS
- **Tests:** 2356 passed, 0 failed (280 test files)
- **Overall:** PASS

## Build & Test Results

```
$ cd services/atlas-ui && npm run build
...
✓ built in 6.89s
(only warning: ConversationEditorPanel chunk >500kB — pre-existing, unrelated to this change)

$ cd services/atlas-ui && npm test
 Test Files  280 passed (280)
      Tests  2356 passed (2356)
```

## File Inventory

- `services/atlas-ui/src/components/features/characters/templates/jobNames.ts` — **Other** (pure data/logic module: per-version race carousel tables + label helpers, no React)
- `services/atlas-ui/src/components/features/characters/templates/IdentitySection.tsx` — **Component** (feature component)
- `services/atlas-ui/src/components/features/characters/templates/CharacterTemplatesEditor.tsx` — **Component** (feature component)
- `services/atlas-ui/src/components/features/characters/templates/TemplateSelector.tsx` — **Component** (feature component)
- `services/atlas-ui/src/components/features/characters/templates/__tests__/CharacterTemplatesEditor.test.tsx` — **Other** (test)
- `services/atlas-ui/src/components/features/characters/templates/__tests__/IdentitySection.test.tsx` — **Other** (test)
- `services/atlas-ui/src/components/features/characters/templates/__tests__/TemplateSelector.test.tsx` — **Other** (test)
- `services/atlas-ui/src/components/features/characters/templates/__tests__/jobNames.test.ts` — **Other** (test)
- `services/atlas-ui/src/components/features/characters/templates/__tests__/raceCarousels.parity.test.ts` — **Other** (test, cross-language fixture parity check)

No hooks (`lib/hooks/api/*`), services (`services/api/*`), schemas (`lib/schemas/*`), or `types/models/*` files are in this diff.

## Anti-Pattern Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-01 | No `any` type | PASS | `grep -n ': any\|as any'` over all 9 changed files returns zero matches |
| FE-02 | No manual class concatenation | PASS | `grep -n 'className={"'` over all 4 source files returns zero matches; no `cn()`-bypassing concatenation introduced |
| FE-03 | No direct API client calls in components | PASS (N/A) | `grep -n '@/lib/api/client'` returns zero matches; diff adds no API calls, only `useTenant()` context reads (`IdentitySection.tsx:16`, `CharacterTemplatesEditor.tsx` diff hunk, `TemplateSelector.tsx:6`) |
| FE-04 | No inline Zod schemas in components | PASS (N/A) | `grep -n 'z\.object\|z\.string('` over the 3 `.tsx` files returns zero matches; no forms/validation touched by this diff |
| FE-05 | No spinners for content loading | PASS (N/A) | `grep -n 'animate-spin'` over the 3 `.tsx` files returns zero matches; diff adds no loading states |
| FE-06 | No hardcoded colors | PASS | `grep -nE 'bg-(white\|black\|gray-[0-9]\|red-[0-9])'` over the 3 `.tsx` files returns zero matches |
| FE-07 | No state mutation | PASS | `grep -n '\.push(\|\.splice(\|\.sort('` over all 4 source files returns zero matches |
| FE-08 | No default exports for components | PASS | `IdentitySection.tsx:25` `export function IdentitySection`, `TemplateSelector.tsx:19` `export function TemplateSelector`, `CharacterTemplatesEditor.tsx` uses named export (unchanged by diff); `grep -n 'export default function'` over all 4 source files returns zero matches |
| FE-09 | Tenant guard in hooks | PASS (N/A) | No `lib/hooks/api/*` files in this diff. The three consumers read tenant via `useTenant()` directly (not a query hook) — `IdentitySection.tsx:31`, `TemplateSelector.tsx:24`, `CharacterTemplatesEditor.tsx` (diff hunk `const { activeTenant } = useTenant();`) — and pass `region`/`majorVersion` straight into pure functions that already handle `undefined` (`jobNames.ts:97` `if (region === undefined \|\| majorVersion === undefined) return [];`), so no `enabled` guard is applicable |
| FE-10 | Tenant ID in query keys | N/A | No query key factories touched by this diff |
| FE-11 | Error handling with `createErrorFromUnknown` | N/A | No `.catch(` / async operations introduced by this diff |

## Architecture Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-12 | JSON:API model shape | N/A | No `types/models/` files changed. `CharacterTemplate` (imported, not modified) is used unchanged |
| FE-13 | Service extends `BaseService` | N/A | No `services/api/` files changed |
| FE-14 | Query key factory uses `as const` | N/A | No query key factories touched |
| FE-15 | Forms use `react-hook-form` + `zodResolver` | N/A | No form touched; `IdentitySection.tsx`'s `<Select>`/`<Input>` are directly-controlled editor fields (pre-existing pattern, unchanged), not a `react-hook-form` form |
| FE-16 | Schema in `lib/schemas/` with inferred type | N/A | No Zod schema in this diff |
| ARCH-1 | Version-aware lookup handles missing tenant | PASS | `jobNames.ts:93-102` `classesForVersion` returns `[]` when `region`/`majorVersion` is `undefined`, and `worldNameFromJobIndex` falls back to `` `Job ${jobIndex}` `` (`jobNames.ts:147`), covered by `jobNames.test.ts:34-36` (`falls back to Job N when no tenant is selected`) |
| ARCH-2 | Tenant fields read from correct nested shape | PASS | `IdentitySection.tsx:32-33` reads `activeTenant?.attributes.region` / `activeTenant?.attributes.majorVersion` (optional-chained only at the `activeTenant` level, matching the `Tenant \| null` / `{ id, attributes }` JSON:API shape); `tsc -b` (via `npm run build`) type-checks this access under `strict`+`exactOptionalPropertyTypes` and passed, which is objective evidence the accessor matches the real `Tenant` type. Same pattern at `CharacterTemplatesEditor.tsx` diff hunk (`activeTenant?.attributes.region`, `activeTenant?.attributes.majorVersion`) and `TemplateSelector.tsx:27-28` |
| ARCH-3 | `raceCarousels.parity.test.ts` fixture path resolution | PASS | `raceCarousels.parity.test.ts:6-16` builds `FIXTURE` via `path.resolve(__dirname, ..8x "..", "docs", "packets", "race-carousels.json")`; verified by direct filesystem check that this resolves to `docs/packets/race-carousels.json` at the worktree root and the file exists there |
| ARCH-4 | Parity test failure mode when fixture missing | PASS (acceptable) | `readFileSync(FIXTURE, "utf-8")` at module scope (`raceCarousels.parity.test.ts:37`) throws synchronously if the fixture is absent, which Vitest surfaces as a hard suite-load failure for this file rather than a silently-skipped or falsely-green test — an acceptably loud failure mode for a parity gate, no try/catch swallowing found |
| ARCH-5 | No duplicate `(jobIndex, subJobIndex)` keys per version table | PASS | Manual inspection of all 8 tables in `jobNames.ts` (lines 16-91) plus the fixture (`docs/packets/race-carousels.json`, scripted duplicate check) shows no duplicate `jobIndex.subJobIndex` pair within any single table/version — the prior duplicate-`SelectItem`-key bug does not recur |
| ARCH-6 | Both `templateLabels` call sites updated for the new signature | PASS | `grep -rn 'templateLabels('` (excluding tests) shows exactly two call sites, `TemplateSelector.tsx:26` and `CharacterTemplatesEditor.tsx:184`, both passing `(templates, region, majorVersion)`; no stale two-argument caller left behind |

## Testing Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-17 | Tests exist for changed components | PASS | `jobNames.ts` → `__tests__/jobNames.test.ts` (96 lines, exercises `classesForVersion`/`worldNameFromJobIndex`/`templateLabels` incl. the undefined-tenant fallback and the 80-82 interpolation band) and `__tests__/raceCarousels.parity.test.ts` (fixture parity); `IdentitySection.tsx` → `__tests__/IdentitySection.test.tsx` (mocks `useTenant` with `region: "GMS", majorVersion: 83`, asserts class-select behavior incl. the "unknown class combo" display path); `TemplateSelector.tsx` → `__tests__/TemplateSelector.test.tsx` (mocks `useTenant` with `majorVersion: 95`); `CharacterTemplatesEditor.tsx` → `__tests__/CharacterTemplatesEditor.test.tsx` (mocks `useTenant`, per `grep` at line 11-12) |
| FE-18 | Mocks updated when services changed | N/A | No `services/api/*` changed in this diff, so no service-facing mocks require updates. Component-level `useTenant()` mocks were added/updated in all three affected test files (`IdentitySection.test.tsx:17-23`, `TemplateSelector.test.tsx:6-12`, `CharacterTemplatesEditor.test.tsx:11-12`) to supply the new `region`/`majorVersion` fields |

## Not evaluable from the diff

- FE-10/FE-14 (query key `as const` / tenant-id-in-key): not evaluable — this diff touches no query key factory file; would need to read `lib/hooks/api/*` query-key definitions to know if any *pre-existing* factory near this feature is non-compliant, but that is out of scope for this change.
- ARCH-2 exact `Tenant`/`TenantAttributes` interface shape: not read directly (no `type Tenant = ...` definition found via targeted grep in `services/atlas-ui/src/context/tenant-context.tsx` or `types/`); compliance was inferred from `tsc -b` passing under `noUncheckedIndexedAccess`/`exactOptionalPropertyTypes`, which is strong but indirect evidence rather than a direct read of the type definition.

## Summary

### Blocking (must fix)
- None.

### Non-Blocking (should fix)
- None identified.
