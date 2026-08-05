# Frontend Audit — task-187-version-aware-id-semantics (atlas-ui)

- **Audit Scope:** atlas-ui files changed on `task-187-version-aware-id-semantics` vs `main` (confirmed via `git diff --name-only main...HEAD | grep atlas-ui`):
  - `services/atlas-ui/src/services/api/availability.service.ts` (new)
  - `services/atlas-ui/src/services/api/__tests__/availability.service.test.ts` (new)
  - `services/atlas-ui/src/lib/hooks/api/useJobAvailability.ts` (new)
  - `services/atlas-ui/src/lib/hooks/api/useSkillAvailability.ts` (new)
  - `services/atlas-ui/src/lib/hooks/usePresetJobOptions.ts` (modified)
  - `services/atlas-ui/src/lib/hooks/__tests__/usePresetJobOptions.test.tsx` (modified)
  - `services/atlas-ui/src/components/features/characters/presets/JobCombobox.tsx` (modified, +7/-1 lines)

  **Scope correction:** `JobSkillsAddButton.tsx` was named in the audit brief but `git diff main...HEAD -- services/atlas-ui/src/components/features/characters/presets/JobSkillsAddButton.tsx` returns empty — it was NOT touched on this branch (its `usePresetJobOptions` usage predates task-187, landed in task-186 per commit `5edb35daf`). Not audited as in-scope; findings below do not cover it.
- **Guidelines Source:** frontend-dev-guidelines skill (`.claude/skills/frontend-dev-guidelines/`)
- **Date:** 2026-07-30
- **Build:** PASS (`nvm use 22 && npm run build` — `tsc -b && vite build` clean; default npm/node on PATH resolves to a Windows node.exe via `/mnt/c` and fails with UNC-path errors, nvm22 required)
- **Tests:** 1361 passed, 0 failed (190 test files; `nvm use 22 && npm test -- --run`)
- **Overall:** NEEDS-WORK

## Build & Test Results

```
tsc -b && vite build  →  built in 1.09s (chunk-size warning only, pre-existing, unrelated to this diff)

vitest run --run
 Test Files  190 passed (190)
      Tests  1361 passed (1361)
```

## File Inventory

- **Service:** `services/atlas-ui/src/services/api/availability.service.ts` — new JSON:API list fetch (`api.getListDocument`) + id/name mapping for job/skill availability.
- **Hook:** `services/atlas-ui/src/lib/hooks/api/useJobAvailability.ts` — new React Query hook, mirrors `useJobs.ts`.
- **Hook:** `services/atlas-ui/src/lib/hooks/api/useSkillAvailability.ts` — new React Query hook, mirrors `useJobAvailability.ts`.
- **Hook:** `services/atlas-ui/src/lib/hooks/usePresetJobOptions.ts` — reconciled to derive options from `useJobAvailability` instead of `JOB_LIST`/`useJobs`.
- **Component:** `services/atlas-ui/src/components/features/characters/presets/JobCombobox.tsx` — now prefers the availability-sourced name for the selected job.
- **Tests:** `services/atlas-ui/src/services/api/__tests__/availability.service.test.ts` (new), `services/atlas-ui/src/lib/hooks/__tests__/usePresetJobOptions.test.tsx` (rewritten for the new data source).

## Anti-Pattern Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-01 | No `any` type | PASS | `grep -n ': any\|as any'` over all 7 in-scope files returns zero matches. |
| FE-02 | No manual class concatenation | PASS | `JobCombobox.tsx` uses `cn()` at line 93 (`className={cn("flex cursor-pointer items-center gap-2 rounded px-2 py-1 hover:bg-accent focus-visible:bg-accent", j.id === value && "bg-accent/60")}`); no `className={"..." +` pattern anywhere in scope. |
| FE-03 | No direct API client calls in components | PASS | `grep -n '@/lib/api/client'` in `JobCombobox.tsx` returns nothing; it goes through `usePresetJobOptions` → `useJobAvailability` → `availabilityService`. |
| FE-04 | No inline Zod schemas in components | PASS | No `z.object(`/`z.string(` anywhere in the 7 files (no forms/schemas touched by this change). |
| FE-05 | No spinners for content loading | PASS | `grep -n animate-spin` on `JobCombobox.tsx` returns nothing; `usePresetJobOptions` falls back to the full `JOB_LIST` while pending rather than rendering a spinner or blank state (`lib/hooks/usePresetJobOptions.ts:34`). |
| FE-06 | No hardcoded colors | PASS | `grep -nE 'bg-(white|black|gray|red|green|blue|yellow)-[0-9]'` on `JobCombobox.tsx` returns nothing; only semantic classes (`bg-accent`, `text-muted-foreground`) are used. |
| FE-07 | No state mutation | PASS | `usePresetJobOptions.ts:35-37` chains `.map(...).sort(...)` on the fresh array `.map()` already produced — `.sort()` never touches `availabilityQuery.data.jobs` (the React Query cache array) in place. |
| FE-08 | No default exports for components | PASS | `JobCombobox.tsx:26` — `export function JobCombobox(...)`, named export only. |
| FE-09 | Tenant guard in hooks | PASS | `useJobAvailability.ts:37` and `useSkillAvailability.ts:36` both set `enabled: !!tenant?.id`; both take an explicit `tenant: Tenant \| null \| undefined` parameter (`useJobAvailability.ts:30`, `useSkillAvailability.ts:29`). |
| FE-10 | Tenant ID in query keys | PASS | `jobAvailabilityKeys.list` (`useJobAvailability.ts:15-16`) and `skillAvailabilityKeys.list` (`useSkillAvailability.ts:15-16`) both include `tenantId ?? "no-tenant"`. |
| FE-11 | Error handling with `createErrorFromUnknown` | PASS (query-hook exception applies) | No `.catch(` sites were introduced in scope — errors flow through React Query's built-in `isError`/`error` state on `useJobAvailability`/`useSkillAvailability`, consistent with the sibling `useJobs.ts` pattern (which also has no explicit `.catch`/toast). `usePresetJobOptions.ts:34` treats both pending *and* error as "unknown" and returns the safe `JOB_LIST` fallback rather than surfacing raw failures — acceptable for a non-blocking picker-population path, but note under Testing below that this fallback branch (the `isError` case specifically, as opposed to `isPending`) has no direct test. |

## Architecture Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-12 | JSON:API model shape | PASS | `JobAvailabilityResource`/`SkillAvailabilityResource` (`availability.service.ts:12-22`) are `{ id: string, type: string, attributes: {...} }`. The flattened `AvailabilityEntry { id: number; name: string }` (`availability.service.ts:24-27`) is a deliberate post-mapping DTO, not a JSON:API resource — same pattern as the sibling `jobsService.getSkillsByJobId` returning plain `number[]` (`jobs.service.ts:29-32`). Documented rationale at `availability.service.ts:4-11`. |
| FE-13 | Service extends `BaseService` (when applicable) | PASS | `availabilityService` uses the direct-client-object pattern (module-level object wrapping `api`, no mutation/validation needs) — matches the documented "Direct API Client Pattern" and mirrors the sibling `jobsService` object literal in `jobs.service.ts:28`. |
| FE-14 | Query key factory uses `as const` | PASS | `useJobAvailability.ts:13-17` and `useSkillAvailability.ts:13-17` both close every factory branch with `as const`. |
| FE-15 | Forms use `react-hook-form` + `zodResolver` | N/A | No forms in scope. |
| FE-16 | Schema in `lib/schemas/` with inferred type | N/A | No Zod schemas in scope. |

## Testing Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-17 | Tests exist for changed components/hooks | **FAIL** | `useJobAvailability.ts` and `useSkillAvailability.ts` have **no dedicated test file** — `find src/lib/hooks/api/__tests__ -iname "*JobAvailability*" -o -iname "*SkillAvailability*"` returns nothing. This is a real regression against established sibling convention, not a stylistic nit: the structurally-identical `useJobs.ts` (which `useJobAvailability.ts`'s own doc comment at line 21 explicitly contrasts itself against) has `useJobs.test.tsx` covering the disabled-without-tenant case, the success path, the error path, *and* a `jobsKeys.list(undefined)` "no-tenant" key-fallback regression test (`useJobs.test.tsx:28-74`). None of that — enabled-guard behavior, the `no-tenant` key fallback, or the success/error mapping into `{ jobs: [...] }` / `{ skills: [...] }` — is independently verified for the two new hooks; it is only exercised indirectly and partially through the mocked-hook unit tests in `usePresetJobOptions.test.tsx` (which mocks `useJobAvailability` itself, so it proves nothing about `useJobAvailability`'s own `enabled`/queryKey/mapping logic). `JobCombobox.tsx`'s new `selectedName` fallback logic (`JobCombobox.tsx:31-34`) is likewise untested — `find src -iname "*JobCombobox.test*"` shows `JobCombobox.test.tsx` exists but it predates this diff (not in the changed-files list) and was not updated to cover the new `jobName(value)` fallback-when-not-in-availability-set branch. |
| FE-18 | Mocks updated when services changed | PASS | `availability.service.test.ts` mocks `@/lib/api/client`'s `getListDocument` directly and asserts both the URL and the id→number/attributes.name mapping (lines 4-8, 16-31); `usePresetJobOptions.test.tsx` mocks the new `useJobAvailability` collaborator (lines 4-7) consistent with how sibling hook-composition tests mock their collaborators. |

## Dead Code Finding (flagged per audit brief)

`useSkillAvailability.ts` has **zero non-test consumers**: `grep -rln "useSkillAvailability" src | grep -v __tests__` returns only `src/lib/hooks/api/useSkillAvailability.ts` itself (its own definition). No page, component, or other hook imports it. It is not unreasonable as the client half of a two-endpoint backend contract (`/api/data/job-availability` + `/api/data/skill-availability`) landing together, and its doc comment (`useSkillAvailability.ts:19-27`) is substantive rather than a stub — but as shipped on this branch it is inert: nothing in the app calls it, so its `enabled`/queryKey/staleTime behavior has no runtime exercise outside its own (nonexistent) test. Given CLAUDE.md's "No Deferring Producible Work" stance and that a skill-availability consumer is presumably the next logical step (a skill picker analogous to `JobCombobox`), this reads as scope left on the table rather than infrastructure with a clear future caller already lined up in this branch. Flagged Important, not Critical — it is inert, not broken, and does not violate any FE-* checklist item by itself.

## Summary

### Blocking (must fix)
- **FE-17** — `useJobAvailability.ts` and `useSkillAvailability.ts` lack dedicated hook tests (queryKey `no-tenant` fallback, `enabled` guard, success/error mapping) that the sibling `useJobs.test.tsx` establishes as the convention for this exact hook shape. `JobCombobox.test.tsx`'s pre-existing suite was not extended to cover the new `selectedName` fallback (`JobCombobox.tsx:31-34`).

### Non-Blocking (should fix)
- **Dead code** — `useSkillAvailability.ts` has no production consumer on this branch (see Dead Code Finding above). Either wire a consumer in this branch or explicitly scope it as a deliberately-landed-ahead-of-its-caller API surface in the PR description.
- **FE-11 (minor)** — the `isError` branch of `usePresetJobOptions`'s pending/error → `JOB_LIST` fallback (`usePresetJobOptions.ts:34`) is covered by inference (mocked `isSuccess: false`) but not by a test that specifically simulates `isError: true, isSuccess: false` to confirm the fallback also holds on genuine fetch failure, not just the pending state. Low risk since the code path is identical (`if (!availabilityQuery.isSuccess) return JOB_LIST;` covers both), but worth a one-line test given the doc comment's explicit claim that pending *and* error both mean "unknown."
