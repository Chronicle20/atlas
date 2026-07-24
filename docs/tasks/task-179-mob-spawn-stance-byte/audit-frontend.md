# Frontend Audit — task-179-mob-spawn-stance-byte (atlas-ui fix commit)

- **Audit Scope:** `git diff main...HEAD -- 'services/atlas-ui/**'` — single commit `a298daad7 fix(atlas-ui): send X-Atlas-Operator on baseline restore`. Two files changed:
  - `services/atlas-ui/src/services/api/baseline.service.ts`
  - `services/atlas-ui/src/services/api/__tests__/baseline.service.test.ts`
- **Guidelines Source:** frontend-dev-guidelines skill (SKILL.md + resources/*)
- **Date:** 2026-07-21
- **Build:** PASS
- **Tests:** 1121 passed, 0 failed (154 files)
- **Overall:** PASS

## Build & Test Results

```
$ npm run build
✓ built in 1.16s   (tsc -b + vite build, no errors; only a pre-existing >500kB chunk-size
  warning on ConversationEditorPanel-*.js, unrelated to this diff)

$ npm test
 Test Files  154 passed (154)
      Tests  1121 passed (1121)
```

Additionally ran targeted lint/format checks on the two changed files (not part of the
mandated gate but relevant since this is a header-formatting change):

```
$ npx eslint src/services/api/baseline.service.ts src/services/api/__tests__/baseline.service.test.ts
(no output — clean)

$ npx prettier --check src/services/api/baseline.service.ts src/services/api/__tests__/baseline.service.test.ts
All matched files use Prettier code style!
```

## File Inventory

- **Service** — `services/atlas-ui/src/services/api/baseline.service.ts` (`BaselineService.restore`, lines 56-64 touched)
- **Test** — `services/atlas-ui/src/services/api/__tests__/baseline.service.test.ts` (restore test block, lines 26, 51-52 touched)

No hooks, components, pages, schemas, or types were touched by this commit.

## Anti-Pattern Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-01 | No `any` type | PASS | `grep -n ': any\|as any' services/atlas-ui/src/services/api/baseline.service.ts services/atlas-ui/src/services/api/__tests__/baseline.service.test.ts` → zero matches in the diff |
| FE-02 | No manual class concatenation | N/A | No JSX/className in either file |
| FE-03 | No direct API client calls in components | N/A | This is the service layer itself, not a component; `fetch()` at `baseline.service.ts:65` is the documented direct-client pattern used by every method in this file (`publish` at line 89, `listBaselines` at line 115) |
| FE-04 | No inline Zod schemas in components | N/A | No `z.object`/`z.string` in either file |
| FE-05 | No spinners for content loading | N/A | No UI in scope |
| FE-06 | No hardcoded colors | N/A | No UI in scope |
| FE-07 | No state mutation | N/A | No component state in scope |
| FE-08 | No default exports for components | N/A | `baseline.service.ts` is not a component; `export class BaselineService` at line 55 is a named export, consistent with convention |
| FE-09 | Tenant guard in hooks | N/A (no hook changed) | The consuming hook `useRestoreBaseline` (`services/atlas-ui/src/lib/hooks/api/useBaseline.ts:9-23`, unchanged by this commit) already guards on `tenant` before calling `baselineService.restore(tenant, body)` at line 16 |
| FE-10 | Tenant ID in query keys | N/A | No query key factory touched |
| FE-11 | Error handling with `createErrorFromUnknown` | PRE-EXISTING, out of scope | `baseline.service.ts:76-81` uses a bespoke `decodeErrorMessage()` + `throw new Error(message)` instead of `createErrorFromUnknown()`. This pattern is unchanged by the diff — confirmed via `git log main..HEAD -- services/atlas-ui/src/services/api/baseline.service.ts` (only commit `a298daad7`, which does not touch lines 75-81) and predates it (introduced in `b39311782`, task-134). Not a new violation introduced by this fix; flagged non-blocking since it sits in the file being touched. |
| FE-12 | JSON:API model shape | N/A | No model files touched. (Pre-existing: `BaselineRestoreInput`/`Baseline` at lines 8-22 are plain attribute interfaces, not `{id, attributes}}` envelopes — consistent with them being wire-payload/DTO types wrapped in the JSON:API envelope inline at call sites, e.g. `attributes: body` at line 71. Out of scope.) |

## Architecture Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-13 | Service extends `BaseService` (when applicable) | PASS (direct-client pattern) | No `base.service.ts` exists anywhere in `services/atlas-ui/src` (`find ... -iname "*BaseService*"` → zero results) — the codebase's documented alternate is the direct-`fetch()` pattern, which `BaselineService.restore` already used before this commit and continues to use after it (`baseline.service.ts:65`, `fetch("/api/data/baseline/restore", ...)`). Consistent with sibling methods `publish` (line 89) and `listBaselines` (line 115) in the same class. |
| FE-14 | Query key factory uses `as const` | N/A | No query key factory touched |
| FE-15 | Forms use `react-hook-form` + `zodResolver` | N/A | No form component touched |
| FE-16 | Schema in `lib/schemas/` with inferred type | N/A | No schema touched |
| **Header/tenant-operator consistency (ad hoc, per audit brief)** | | | |
| — | `restore()` carries correct tenant headers | PASS | `baseline.service.ts:62` — `const headers = tenantHeaders(tenant);` builds `TENANT_ID`/`REGION`/`MAJOR_VERSION`/`MINOR_VERSION` from the tenant argument (`lib/headers.tsx:3-10`), required because restore targets a specific `tenant` (see `BaselineRestoreInput.tenantId`, `baseline.service.ts:12`) — it cannot use `canonicalHeaders`, which hardcodes the synthetic `CANONICAL_TENANT_ID` (`lib/headers.tsx:19,34`) and would send the wrong tenant scope. |
| — | `restore()` now sends `X-Atlas-Operator` | PASS | `baseline.service.ts:63` — `headers.set("X-Atlas-Operator", "1");` added; matches the value baked into `canonicalHeaders` at `lib/headers.tsx:38` (`headers.set("X-Atlas-Operator", "1")`) used by sibling methods `publish` (line 87) and `listBaselines` (line 113) |
| — | Rationale documented at point of use | PASS | `baseline.service.ts:57-61` comment explains the atlas-data 403 gate and why `tenantHeaders` alone is insufficient, referencing `baseline/handler.go` |
| WARN | Operator-flag value `"1"` duplicated as a bare literal instead of a shared constant | WARN (non-blocking) | `lib/headers.tsx:38` and `baseline.service.ts:63` both hardcode the literal `"1"` for `X-Atlas-Operator`. `canonicalHeaders`'s own doc comment (`lib/headers.tsx:27-31`) states its purpose is "one construction path, no drift" for the operator header — that guarantee is now only true for canonical-scope calls; `restore()` is a second, independent construction path that must keep the literal in sync by hand. No pre-existing helper covered "tenant-scoped + operator-gated" (checked all other tenant-scoped calls in `seed.service.ts:156,206,210,216,223` — none set `X-Atlas-Operator`), so this isn't a regression against an established pattern, but a named export (e.g. `OPERATOR_HEADER_VALUE` or a `tenantOperatorHeaders(tenant)` helper in `lib/headers.tsx`) would remove the duplication for the next tenant-scoped+operator-gated call. |

## Testing Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-17 | Tests exist for changed components | PASS | `baseline.service.test.ts:26-66` — the existing `restore` test block was updated in place (title changed line 26; assertion flipped from `toBeNull()` to `toBe("1")` at line 52) to cover the new header. It asserts the header is present, and continues to assert all four tenant headers (lines 46-49), `Content-Type` (line 50), and the full request body (lines 53-65) — full coverage of `restore()`'s header-construction branch, not just the one line changed. |
| — | Test actually exercises the changed code path (not vacuously) | PASS | Ran `npm test` — `1121 passed (1121)`, and the specific assertion `expect(headers.get("X-Atlas-Operator")).toBe("1")` at `baseline.service.test.ts:52` would fail if `baseline.service.ts:63` were reverted (manually verifiable: removing that `headers.set` call makes `tenantHeaders` return a `Headers` object with no `X-Atlas-Operator` key, so `.get()` returns `null`, failing `toBe("1")`) |
| — | Test doesn't merely assert the mock's own behavior | PASS | `fetchMock` (line 16-19) only stubs the HTTP response; the header assertions read the real `Headers` object built by `baselineService.restore()` and passed to `fetch`, not a hand-constructed fixture |
| FE-18 | Mocks updated when services changed | N/A | No `__mocks__/` directory exists in `services/atlas-ui/src` (`find ... -iname "__mocks__"` → zero results); no consumer mocks `baselineService` — grep for `baseline.service|BaselineService` outside `__tests__/` found only `useBaseline.ts` and `useCanonicalData.ts`, both of which call the real service through the hook layer, not a jest/vitest mock |

## Summary

### Blocking (must fix)
- None.

### Non-Blocking (should fix)
- Operator-flag literal `"1"` is duplicated across `lib/headers.tsx:38` and `baseline.service.ts:63` with no shared constant/helper (see WARN row above). Low risk today (both values verified equal), but the "one construction path, no drift" guarantee `canonicalHeaders` documents for itself no longer holds project-wide once a second tenant-scoped+operator-gated call is added elsewhere. Consider extracting a named export before the next such call is written.
- FE-11 (`createErrorFromUnknown` not used in `baseline.service.ts`'s error path) is a pre-existing gap dating to task-134 (`b39311782`), untouched by this commit — flagged for visibility only, not blocking this PR.
