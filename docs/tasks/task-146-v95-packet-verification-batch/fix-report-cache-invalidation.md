# Fix report: Packet Matrix cache invalidation (Defect A)

Branch: `fix/packet-matrix-cache-invalidation`
Commit: `2846074af` — fix(atlas-ui): invalidate Packet Matrix cache on template/tenant socket writes

## What I implemented

Root cause (from the bug file, Round 3 / "Root cause (final)"): none of the
template or tenant-configuration mutation hooks invalidated
`socketKeys.all`, so the Packet Matrix's sparse read
(`socketKeys.matrix()` / `socketKeys.tenantMatrix()`, `staleTime: 30_000`,
default 5-minute `gcTime`) kept serving the pre-write document until a hard
reload or a 5-minute idle.

1. **Extracted `socketKeys`** out of `useSocketObjects.ts` into a new module
   `services/atlas-ui/src/lib/hooks/api/socketKeys.ts`, carrying the original
   docstring (updated to describe the new home). `useSocketObjects.ts` now
   imports it from there and re-exports it (`export { socketKeys }`), so
   every existing importer (`dialogs.test.tsx`, `CopyFromAncestorFlow.test.tsx`,
   `DefinitionGridPage.test.tsx`, `PacketMatrixPage.test.tsx`,
   `templates.service.ts`, `useSocketObjects.test.tsx`) keeps working
   unchanged — confirmed via `grep -rln "socketKeys" src`, all seven hits
   import from `useSocketObjects.ts` or are the file itself.
   `useSocketMutation`'s `onSuccess` (already correct — untouched) now
   references the imported binding.

2. **`useTemplates.ts`** — added
   `queryClient.invalidateQueries({ queryKey: socketKeys.all })` to all eight
   mutation hooks, each in its existing `onSuccess`/`onSettled` block,
   preserving each hook's existing choice:
   - `useCreateTemplate` — `onSuccess`
   - `useUpdateTemplate` — `onSettled`
   - `usePatchTemplate` — `onSuccess`
   - `useDeleteTemplate` — `onSettled`
   - `useReseedTemplate` — `onSuccess` only (FR-5.6 preserved: a failed
     reseed changes nothing server-side, so no invalidation on error)
   - `useCreateTemplatesBatch` — `onSuccess`
   - `useUpdateTemplatesBatch` — `onSuccess`
   - `useDeleteTemplatesBatch` — `onSettled`

   Also fixed the stale `useReseedTemplate` docstring, which claimed "the
   drift badge clears without a manual reload" — it now says the Packet
   Matrix's sparse read is invalidated too, and no longer implies reload was
   ever needed for correctness.

3. **`useTenants.ts`** — read the whole file first, per the ruling. Only
   `useCreateTenantConfiguration` and `useUpdateTenantConfiguration` write a
   `TenantConfigAttributes` document (which includes `socket` —
   confirmed via `tenants.service.ts` sort-handlers-and-writers code at
   `tenants.service.ts:179-189`, which reads `config.attributes.socket`).
   Added the same `socketKeys.all` invalidation to their existing
   `onSuccess` / `onSettled` blocks. Left `useTenants`, `useTenant`,
   `useCreateTenant`, `useUpdateTenant`, `useDeleteTenant` (all `TenantBasic`,
   no socket document) untouched.

Did **not** touch `PacketMatrixPage.tsx#highestVersionKey` (Defect B) —
explicitly out of scope for this branch.

## Tests added

Following the existing hook-test harness pattern (see
`useReseedTemplate.test.tsx`, `useSocketObjects.test.tsx`): `renderHook` +
`QueryClientProvider` + `vi.spyOn(qc, "invalidateQueries")`, asserting
`socketKeys.all` is among the invalidated query keys.

- `services/atlas-ui/src/lib/hooks/api/__tests__/useTemplates.socketInvalidation.test.tsx`
  — one test per mutation hook (8 hooks, 9 tests including a negative case
  for `useReseedTemplate`'s failure path, which must invalidate nothing).
- `services/atlas-ui/src/lib/hooks/api/__tests__/useTenants.socketInvalidation.test.tsx`
  — `useCreateTenantConfiguration` and `useUpdateTenantConfiguration`.

## Verification (module-local, per Contract 2)

```
cd services/atlas-ui && npx vitest run src/lib/hooks/api
```
```
 Test Files  36 passed (36)
      Tests  270 passed (270)
   Start at  18:47:50
   Duration  4.26s (transform 8.17s, setup 5.81s, import 14.45s, tests 9.28s, environment 46.16s)
```

```
cd services/atlas-ui && npx tsc --noEmit
```
Empty output — clean.

(Note: `services/atlas-ui/node_modules` was absent in the worktree; ran
`npm install` first — 611 packages, no code changes, matches the committed
`package-lock.json`.)

## Files changed

- `services/atlas-ui/src/lib/hooks/api/socketKeys.ts` (new)
- `services/atlas-ui/src/lib/hooks/api/useSocketObjects.ts`
- `services/atlas-ui/src/lib/hooks/api/useTemplates.ts`
- `services/atlas-ui/src/lib/hooks/api/useTenants.ts`
- `services/atlas-ui/src/lib/hooks/api/__tests__/useTemplates.socketInvalidation.test.tsx` (new)
- `services/atlas-ui/src/lib/hooks/api/__tests__/useTenants.socketInvalidation.test.tsx` (new)

## Self-review

- Every hook named in the ruling has the invalidation, placed alongside the
  existing `templateKeys.lists()` / `tenantKeys.configLists()` call, in the
  same `onSuccess`/`onSettled` callback the hook already used (no hook's
  success/settled choice was changed).
- `useSocketMutation` in `useSocketObjects.ts` was not modified — confirmed
  by diff (only the `socketKeys` declaration moved out; the mutation body is
  byte-identical).
- No import cycle: `useSocketObjects.ts`, `useTemplates.ts`, `useTenants.ts`
  all import `socketKeys` from the new leaf module `socketKeys.ts`, which
  imports nothing from any of them.
- `grep -n socketKeys useTemplates.ts` / `useTenants.ts` confirms all
  intended call sites; `grep -rln socketKeys src` confirms no import-cycle
  regression for existing consumers.
- `PacketMatrixPage.tsx` untouched (`git diff --stat` on the commit shows
  only the six files above).

## Concerns

None. `npm install` was required because `node_modules` didn't exist in this
worktree; this is an environment setup step, not a code change, and
`package-lock.json` was not modified (`git status` shows no lockfile diff).
