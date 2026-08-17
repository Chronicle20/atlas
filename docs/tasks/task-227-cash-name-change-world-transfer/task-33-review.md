# Task 33 review — atlas-ui pending-changes service and hooks

Commit reviewed: `4dfd4df1d` (1 commit, +181 lines, 4 files). Read-only review; no edits made.

## Verdict: PASS (spec compliance) / PASS (FE-* code quality)

Both halves checked independently against source (Go producer, `client.ts`, and
the sibling reference files), not against the implementer's report claims.

## 1. Scope fence (FR-2.10: read + cancel only) — PASS

`pending-changes.service.ts:41-59` exposes exactly two methods:
`getByCharacterId` (GET) and `cancel` (DELETE). Grepped the whole diff for
`api.post`, `api.put`, `api.patch`, `create` — zero matches. No method can
originate or edit a requested value. The Go side (`pending_change/resource.go:26-29`)
has `create_pending_change` (POST) and `resolve_pending_change` (POST) routes
that this diff correctly does not touch.

## 2. Optional vs nullable fields — PASS

`pending-changes.service.ts:11-21`:
```ts
requestedName?: string;
reason?: string;
resolvedAt?: string;
```
Matches `pending_change/rest.go:17-24`'s `omitempty` tags exactly (`RequestedName`,
`Reason` both `,omitempty` `string`; `ResolvedAt` `*time.Time` `,omitempty`). No
`| null` anywhere in the file — correctly modeled as absent-key-when-empty, not
nullable. `PendingChangeAttributes` (`:23-33`) duplicates the same three fields
as optional, so the two interfaces don't disagree with each other either.

## 3. GET returns an array → `api.getList` — PASS

`pending-changes.service.ts:42-47`:
```ts
const resources = await api.getList<PendingChangeResource>(
  `${BASE_PATH}/${characterId}/pending-changes`,
);
return resources.map(flatten);
```
`api.getList<T>` (`client.ts:357-358`) returns `Promise<T[]>` via
`apiClient.get<ApiListResponse<T>>(url).then(r => r.data)` — matches
`handleGetPendingChanges` marshaling `[]RestModel`
(`pending_change/resource.go:41-60`). Correctly does NOT copy
`teleport-rocks.service.ts`'s `api.getOne` call, which is right only for that
service's single-resource endpoint.

## 4. DELETE returns 204, no body — PASS

`pending-changes.service.ts:57-59`: `await api.delete<void>(...)`, method
signature `cancel(...): Promise<void>`. Matches `handleCancelPendingChange`
(`pending_change/resource.go:139-183`), which ends `w.WriteHeader(http.StatusNoContent)`
with no marshaled body on the success path.
`usePendingChanges.ts:44-52` uses `qc.invalidateQueries({ queryKey:
pendingChangeKeys.detail(v.tenant?.id, v.characterId) })` in `onSuccess` — no
`setQueryData` anywhere in the diff. Correct: there is no response body to
cache, so invalidate-and-refetch is the only valid strategy.

The implementer also correctly declined to add `teleport-rocks.service.ts`'s
`unwrap()` helper (`teleport-rocks.service.ts:20-29`) — there's no write path
here that returns a body needing envelope normalization, and Correction 4 of
the brief explicitly says not to add a dead helper. Confirmed no `unwrap` in
`pending-changes.service.ts`.

## 5. Test harness and assertion strength — PASS

`usePendingChanges.test.tsx` uses the real repo pattern, verified line-by-line
against `useCoupons.test.tsx:1-40`:
- Module-scope `vi.mock("@/services/api/pending-changes.service", () => ({...}))`
  at `test.tsx:11-16` (not `vi.spyOn` on a real import — the brief's invented
  snippet used `vi.spyOn`, correctly discarded).
- Locally defined `wrapper(qc)` at `test.tsx:18-22`, matching
  `useCoupons.test.tsx`'s `wrapper(qc)` shape exactly.
- A fresh `new QueryClient({ defaultOptions: { queries: { retry: false } } })`
  per test (`test.tsx:29-31`, `:41-43`), not a shared module-level client.

Both assertions are real, not vacuous:
- **Tenant gate** (`test.tsx:32-40`): asserts
  `pendingChangesService.getByCharacterId` was never called AND
  `result.current.isPending === true`. Removing the `enabled: !!tenant?.id &&
  !!characterId` gate in `usePendingChanges.ts:27` would fire the query and
  fail the `not.toHaveBeenCalled()` assertion — this is a real regression
  trap, not a tautology.
- **Cancel → invalidate** (`test.tsx:42-60`): awaits `mutateAsync(...)`, then
  `waitFor` asserts `invalidate` was called with the exact computed key
  `pendingChangeKeys.detail(tenant.id, "1")`. Removing the
  `qc.invalidateQueries(...)` call in `usePendingChanges.ts:47-50`, or keying
  it on the wrong tenant/character, would fail this assertion. Both tests
  would fail if the described behavior were removed — satisfies the brief's
  bar.

RED/GREEN evidence in the report (module-not-found failure, then 2/2 pass) is
plausible and consistent with a genuinely new test file plus new source files
landing in the same commit; nothing in the diff contradicts it.

## 6. Shape parity with `useTeleportRocks.ts` / `teleport-rocks.service.ts` — PASS, one flagged (non-blocking) divergence

- `pendingChangeKeys.{all,detail}` (`usePendingChanges.ts:14-18`) — same
  shape as `teleportRockKeys` (`useTeleportRocks.ts:14-18`), including the
  `tenantId ?? "no-tenant"` fallback (FE-10 compliant).
- `enabled: !!tenant?.id && !!characterId`, `staleTime: 60 * 1000`,
  `gcTime: 5 * 60 * 1000` (`usePendingChanges.ts:29-31`) — byte-identical to
  `useTeleportRocks.ts:27-29`.
- `CancelVars` interface for the mutation (`usePendingChanges.ts:34-38`) plays
  the same role as `AddVars`/`RemoveVars` in the reference file.

Flagged divergence (style only, not a defect): `flatten()`
(`pending-changes.service.ts:36-39`) spreads `r.attributes` into the return
value and relies on `PendingChangeAttributes` (`:23-33`) being kept
structurally identical to `PendingChange` minus `id` by hand, whereas
`teleport-rocks.service.ts:31-36` builds the flattened object field-by-field
from `TeleportRockLists` (reused directly, not duplicated). The duplication
here is a real maintenance hazard — a future field added to `PendingChange`
but not to `PendingChangeAttributes` would silently type-check as valid data
loss on read (the field would just be `undefined`), since `exactOptionalPropertyTypes`
doesn't catch a *missing* interface member, only a wrongly-typed one. `Omit<PendingChange, "id">` would remove that hazard entirely.
Not blocking — no live bug today, both interfaces are currently in sync — but
worth fixing before another field is added to either one.

## 7. Barrel export (`index.ts`) — PASS against the authoritative brief, flagged as worth revisiting

`index.ts:207-212` adds only
`export type { PendingChange, PendingChangeType, PendingChangeStatus } from "./pending-changes.service";`
— no `export { pendingChangesService }` line. This is literally what the
controller inventory's Files section specifies ("MODIFY. Export the service's
TYPES only"), so it is compliant with the authoritative half of the brief.

However: checked every other service export in the same file
(`index.ts:79-217`) — `tenantsService`, `charactersService`, `couponsService`,
`bansService`, `monsterBookService`, etc. — and every single one that has a
type-export block *also* has an instance-export line beside it. There is no
existing precedent in this barrel for "types only, no instance." (`teleportRocksService`
itself isn't in the barrel at all, so it's not a counter-example either —
it's simply absent.) So while the implementer followed the letter of the
controller correction, the correction itself creates a barrel entry with no
precedent in the file, and the next task (the panel component that actually
calls `.cancel()`/`.getByCharacterId()`) will have to import
`pendingChangesService` directly from `@/services/api/pending-changes.service`
rather than the barrel — inconsistent with how every other consuming
component in this codebase imports its service. Not a defect in this task
(the instruction was followed correctly and the omission causes no compile or
runtime issue), but flag it for whoever picks up the panel task: either add
the instance export then, or confirm direct-import is intentionally the new
convention for read/cancel-only services.

## FE-* checklist (code quality)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-01 | No `any` | PASS | grep for `: any`/`as any` across all 4 changed files — zero matches |
| FE-02 | No manual class concat | PASS | N/A — no JSX/className in these files |
| FE-03 | No direct API client import in components | PASS | These are service/hook files, not components; `pending-changes.service.ts:1` imports `api` from `@/lib/api/client`, which is exactly where the service layer is supposed to do that |
| FE-04 | No inline Zod in components | PASS | No `z.object`/zod usage at all in this diff |
| FE-05 | No spinners for content loading | PASS | No `animate-spin` in diff |
| FE-06 | No hardcoded colors | PASS | No Tailwind color classes in diff (no JSX) |
| FE-07 | No state mutation | PASS | `flatten` and key factories are pure; no `.push`/`.splice`/`.sort` |
| FE-08 | No default exports | PASS | All exports in the 3 new files are named (`export const`, `export function`, `export interface`, `export type`) |
| FE-09 | Tenant guard in hooks | PASS | `usePendingChanges.ts:24` takes `tenant: Tenant \| null \| undefined` param; `:29` gates `enabled: !!tenant?.id && !!characterId` |
| FE-10 | Tenant ID in query keys | PASS | `pendingChangeKeys.detail` (`usePendingChanges.ts:16-18`) includes `tenantId ?? "no-tenant"` |
| FE-11 | Error handling via `createErrorFromUnknown` | PASS (N/A pattern) | No `.catch(` in the diff; matches the reference hook `useTeleportRocks.ts`, which also has no catch/createErrorFromUnknown — errors propagate to React Query's `.error`/`isError`, the correct hook-layer pattern; `createErrorFromUnknown` is a component-layer concern per the reference file |
| FE-14 | Query key factory uses `as const` | PASS | `usePendingChanges.ts:15,18` — both `all` and `detail`'s return are `as const` |
| FE-16 | Schema paired with inferred type | N/A | No Zod schema in this task (read + cancel only, no form) |

## Self-flagged items — independent conclusions

- **`PendingChangeAttributes`/`PendingChange` duplication instead of `Omit<>`:**
  agree it's non-blocking, but disagree it's purely stylistic — see §6 above,
  it's a real (if currently dormant) drift hazard. Recommend fixing opportunistically,
  not blocking this task on it.
- **Barrel exports types only, not the service instance:** the implementer's
  own justification (matching brief-intent) is correct as far as it goes, but
  see §7 — the deeper point is that this creates a barrel with no existing
  precedent shape in the file. Non-blocking for this task; worth a decision
  before the panel-component task lands.

## Summary

### Blocking (must fix)
- None.

### Non-blocking (should fix opportunistically)
- `PendingChangeAttributes`/`PendingChange` duplication in
  `pending-changes.service.ts:11-33` — replace with `Omit<PendingChange, "id">`
  to remove the manual-sync hazard (§6).
- Confirm before the panel-component task whether `pendingChangesService`
  should be added to the barrel (`index.ts`) alongside its types, since every
  other barrel-exported service follows that shape and this one currently
  doesn't (§7).
