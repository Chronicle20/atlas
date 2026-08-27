# Frontend Audit — task-269-ring-pair-behavior (atlas-ui diff 32d55cb21..e5f7cf0)

- **Audit Scope:** `git diff --stat 32d55cb21..e5f7cf0 -- services/atlas-ui` → exactly 2 files:
  - `services/atlas-ui/src/services/api/accounts.service.ts` (5 lines changed)
  - `services/atlas-ui/src/services/api/__tests__/accounts.service.test.ts` (new file, 31 lines)
- **Guidelines Source:** frontend-dev-guidelines skill
- **Date:** 2026-08-27
- **Build:** NOT RUN — explicitly withheld per task instructions (a flagless `tools/verify.sh` was running concurrently against this exact worktree at task start; its atlas-ui build/lint/test stage already passed at an earlier commit on this branch and was re-running). No independent build/test evidence was collected in this session.
- **Tests:** NOT RUN — same reason as above.
- **Overall:** NEEDS-WORK (build/test not independently verified in this session — cannot be claimed PASS by default-FAIL rule; checklist below also has WARN-level pre-existing findings visible in the in-scope file)

## Build & Test Results

Not executed. Per the dispatch instructions: "Do not run `tools/verify.sh` yourself; its atlas-ui lint/test/build stage already passed at an earlier commit on this branch and is re-running now." `npm run build` / `npm test` were likewise withheld to keep the worktree read-only while the concurrent process reads it. This audit is therefore Phase 3 (mechanical/architecture/testing checklist) only; Phase 1's objective gate is unconfirmed from this session and must not be represented as a PASS.

## File Inventory

- `services/atlas-ui/src/services/api/accounts.service.ts` — **Service**. Diff touches only `updateAccountBirthDate` (lines 118-135 in the current file; the actual changed lines are the import at line 15 and lines 129/134). Rest of the file (transformAccount, sortAccounts, buildAccountQuery, accountExists, createAccount, terminateMultipleSessions, etc.) is byte-identical to `32d55cb21` — confirmed via `diff` against the pre-change blob, which shows only the import line and the `patch`/unwrap lines changed.
- `services/atlas-ui/src/services/api/__tests__/accounts.service.test.ts` — **Service test** (new file). Covers `getAccountsPage`, `getAllAccounts`, and a regression test for `updateAccountBirthDate`'s envelope-unwrap bug.

Note on scope vs. task framing: the task description frames this branch as "ring/couple-ring pair behavior," but the actual atlas-ui diff for `32d55cb21..e5f7cf0` touches only `accounts.service.ts`/its test — a JSON:API envelope-unwrap bug fix for account birth-date updates, unrelated to ring pairing on its face. Scope was derived mechanically from `git diff --stat`; no ring-specific atlas-ui files appear in this diff range.

## Anti-Pattern Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-01 | No `any` type | PASS | `grep -n ': any\|as any'` on both in-scope files returns zero matches. |
| FE-02 | No manual class concatenation | N/A | Neither file contains JSX/`className`. |
| FE-03 | No direct API client calls in components | N/A | Both files are service/test layer, not components; `accounts.service.ts:1` imports `@/lib/api/client` as expected for a service module. |
| FE-04 | No inline Zod schemas in components | N/A | No `z.object`/`z.string` in either file. |
| FE-05 | No spinners for content loading | N/A | No `animate-spin` in either file. |
| FE-06 | No hardcoded colors | N/A | No className/color tokens in either file. |
| FE-07 | No state mutation | FAIL (pre-existing, not touched by this diff) | `services/atlas-ui/src/services/api/accounts.service.ts:44` — `sortAccounts` does `return accounts.sort((a, b) => ...)`, which sorts the array **in place** (`Array.prototype.sort` mutates its receiver and returns the same reference), matching the exact anti-pattern the checklist forbids (`.sort(` before/around a return). Confirmed via `diff` against `32d55cb21` that this line is unchanged by the current diff — flagged for visibility, not attributable to this change. Low practical risk since callers always pass a freshly-`.map()`'d array, but it is a literal checklist violation present in the file under review. |
| FE-08 | No default exports for components | PASS | `grep -n 'export default function'` on `accounts.service.ts` returns zero matches; service is exported as a named `export const accountsService = {...}` (line 65). |
| FE-09 | Tenant guard in hooks | N/A | Neither file is a hook under `lib/hooks/api/`. |
| FE-10 | Tenant ID in query keys | N/A | No query key factory in either file. |
| FE-11 | Error handling with `createErrorFromUnknown` | FAIL (pre-existing, not touched by this diff) | `accounts.service.ts:137-152` (`accountExists`) and `:247-282` (`terminateMultipleSessions`) use raw `try/catch` with manual `error.message`/`error.status` extraction instead of `createErrorFromUnknown()` (defined in `src/lib/api/errors.ts:22`). Neither call site is part of the diff hunk (confirmed via `diff` against `32d55cb21`), so this predates the branch under review — flagged for visibility only. |
| FE-12 | JSON:API model shape | Not evaluable from the diff | `types/models/account.ts` was not touched and is out of the review surface; `Account`/`AccountAttributes` are only imported (`accounts.service.ts:13`), not defined, in scope. |
| FE-13 | Service extends `BaseService` (or documented direct-client alternative) | PASS (pre-existing pattern) | `accounts.service.ts` uses the documented "Direct API Client Pattern" (`patterns-service-layer.md` §2) — a plain object composing `api.getOne`/`api.patch`/`api.delete` (lines 101-104, 129-134, 169-181), not extending `BaseService`. This is an accepted alternative for simple resources. Note: unlike the doc's example signature (`tenant` as explicit first parameter on every tenant-scoped method), most methods here take no `tenant` param at all and rely on the `TenantProvider`-driven singleton `api.setTenant` side channel documented in `services/atlas-ui/CLAUDE.md` ("Tenant contract"/"redundant per-call `api.setTenant` calls... legacy from the Next.js era"). This deviation is pre-existing and not introduced by the diff under review. |
| FE-14 | Query key factory uses `as const` | N/A | No query key factory in either file. |
| FE-15 | Forms use `react-hook-form` + `zodResolver` | N/A | No form code in either file. |
| FE-16 | Schema in `lib/schemas/` with inferred type | N/A | No Zod schema in either file. |

## Architecture Checklist

(Merged into Anti-Pattern table above where overlapping; remaining items below.)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-12 | JSON:API model shape | Not evaluable from the diff | See above — `types/models/account.ts` not in scope. |
| FE-13 | Service extends `BaseService` | PASS with note | See above. |
| FE-14 | Query key factory `as const` | N/A | No query keys in scope. |
| FE-15 | react-hook-form + zodResolver | N/A | No forms in scope. |
| FE-16 | Schema + inferred type | N/A | No schemas in scope. |
| Core diff correctness | `updateAccountBirthDate` envelope unwrap | PASS | `accounts.service.ts:129-134` now calls `api.patch<ApiSingleResponse<Account>>(...)` and returns `transformAccount(response.data)`. Verified against `src/lib/api/client.ts:388-392` (`api.patch` returns the raw parsed JSON body — the JSON:API envelope — with no `.data` unwrap, unlike `api.getOne` at `client.ts:374-375` which does unwrap). Before this diff, `api.patch<Account>(...)` typed the envelope itself as `Account` and passed it straight to `transformAccount`, whose first field access (`data.attributes.loggedIn`) would have thrown on the real envelope shape `{data: {...}}`. The fix is correct and directly addresses a real runtime bug. |

## Testing Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-17 | Tests exist for changed components | PASS | `services/atlas-ui/src/services/api/__tests__/accounts.service.test.ts:99-119` adds a regression test for the exact bug fixed in this diff (envelope unwrap in `updateAccountBirthDate`), asserting `result.attributes.birthDate` and `result.attributes.loggedIn` against a mocked `{data: {...}}` envelope response from `patchMock`. `getAccountsPage`/`getAllAccounts` tests (lines 39-90) are pre-existing coverage carried into the same file, not new to this diff (they don't touch the changed lines) but are present. |
| FE-18 | Mocks updated when services changed | N/A | No `__mocks__/` directory convention in use anywhere in `services/atlas-ui/src` (`find ... -iname '*mock*'` under services returns nothing); this codebase uses inline `vi.mock()` per test file instead, as seen at `accounts.service.test.ts:5-13`, which correctly mocks only the `get`/`patch` surface actually exercised. |

## Not evaluable from the diff

- FE-12 (JSON:API model shape) — would need `services/atlas-ui/src/types/models/account.ts`, which is untouched and outside the review surface.
- FE-09/FE-10 (tenant guard / tenant-scoped query keys) — no hooks or query key factories appear in this diff; genuinely not applicable rather than unevaluated, but noted since `accountsService` methods lack explicit `tenant` parameters and this audit did not trace every call site that invokes `accountsService.updateAccountBirthDate` to confirm the ambient `api.setTenant` singleton is always correctly primed before each call — that would require reading callers outside the diff (e.g. hooks/pages consuming this service), which is out of scope per the Scope rules for a diff-scoped review.

## Summary

### Blocking (must fix)
- None introduced by this diff. The two-line functional change (`accounts.service.ts:129-134` + the `ApiSingleResponse` import at line 15) is correct, tested, and matches the documented envelope-handling contract (`client.ts:374-375` vs `:388-392`).

### Non-Blocking (should fix)
- FE-07 — `accounts.service.ts:44` (`sortAccounts`) mutates its input array via `.sort()` rather than sorting a copy; pre-existing, not part of this diff, but a literal checklist violation visible in the file.
- FE-11 — `accounts.service.ts:137-152` and `:247-282` use manual `try/catch` + `error.message`/`error.status` narrowing instead of `createErrorFromUnknown()`; pre-existing, not part of this diff.
- FE-13 note — most `accountsService` methods omit the explicit `tenant` parameter the service-layer pattern doc shows, relying instead on the `TenantProvider`-driven `api.setTenant` singleton; this is an acknowledged, pre-existing legacy pattern per `services/atlas-ui/CLAUDE.md`, not introduced here.
