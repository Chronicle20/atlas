# Review: fix/packet-matrix-cache-invalidation (commit 2846074af)

Scope: commit `2846074af1ea0714861093f5b7991e6ac5dc6c8b` on branch
`fix/packet-matrix-cache-invalidation`, base `main@8d875caec`. Requirement:
`bug-cashshopentry-0x2b-not-in-matrix.md` Round 3 / "Root cause (final)" /
"Fix (final)" (Defect A only — Defect B, `highestVersionKey`, is explicitly
out of scope). Implementer's own report:
`fix-report-cache-invalidation.md`.

## Verdict

APPROVED_WITH_FINDINGS

## What was reviewed

`git show --stat 2846074af` and full hunks for all six changed files:
`socketKeys.ts` (new), `useSocketObjects.ts`, `useTemplates.ts`,
`useTenants.ts`, and the two new test files. Ran the new and full
`src/lib/hooks/api` suite (270/270 pass) and `tsc --noEmit` (clean) inside
the worktree. Additionally reverted the three source files to their
pre-commit (`2846074af~1`) state and reran the two new test files to confirm
they fail without the fix (test-honesty check, not in the implementer's own
verification).

## Findings

### 1. Completeness of invalidation — `useDeleteTenant` gap (non-blocking, correctly scoped out but should be tracked)

`useDeleteTenant` (`services/atlas-ui/src/lib/hooks/api/useTenants.ts:165-201`)
deletes a `TenantBasic` and only invalidates `tenantKeys.basicLists()`
(`useTenants.ts:199-201`). It does not invalidate `socketKeys.all`.

Deleting a tenant removes a column from the tenant packet matrix
(`useSocketMatrixTenants`, which reads `socketKeys.tenantMatrix()` via
`socketKeys.all`). This is a real gap in the mutation-hook sense: if the
tenant matrix were rendered anywhere, a delete would leave a stale column
until reload, exactly the class of bug this commit fixes elsewhere.

However: confirmed via `grep -rn "useSocketMatrixTenants" src` that the hook
has zero callers outside `PacketMatrixPage.test.tsx`'s own mock — the tenant
matrix is unrendered (matches the background note and the bug file's own
"packet-matrix is templates-only" finding). With no live read path, there is
no observable defect today, so leaving `useDeleteTenant` untouched does not
violate the "every write path that can change a template's or tenant's
socket document" requirement in a way that has any current user-visible
effect. Judgment: **correctly out of scope for this commit**, but it is a
latent trap — the day `useSocketMatrixTenants` gets a caller, this hook
becomes a real bug with no test flagging it. Worth a one-line follow-up note
or a task-tracker entry so it isn't rediscovered the hard way; not blocking
this fix.

There is no separate "delete tenant configuration" mutation hook in
`useTenants.ts` (`grep -n "^export function" useTenants.ts` shows only
`useCreateTenantConfiguration`/`useUpdateTenantConfiguration` for configs) —
config deletion apparently rides on tenant deletion, so this is one gap, not
two.

### 2. Import-cycle rationale — verified sound

`socketKeys.ts` (new, `services/atlas-ui/src/lib/hooks/api/socketKeys.ts`)
imports nothing from `useTemplates.ts`, `useTenants.ts`, or
`useSocketObjects.ts`; it is a leaf module. `useTemplates.ts:22` and
`useTenants.ts:25` import `socketKeys` from the new module directly.
`useSocketObjects.ts:29` also imports from the new module and re-exports it
(`export { socketKeys }`) with a comment explaining why. `grep -rln
socketKeys src` shows all other importers (`dialogs.test.tsx`,
`DefinitionGridPage.test.tsx`, `CopyFromAncestorFlow.test.tsx`,
`useSocketObjects.test.tsx`) still reference `useSocketObjects.ts` (or, in
`templates.service.ts` and `PacketMatrixPage.test.tsx`, only a comment/mock,
not an import) — none broken by the re-export. `npx tsc --noEmit` is clean
and the full `src/lib/hooks/api` suite (270/270) and the two socket dialog/
grid test files pass, which exercises these import paths at runtime. No
cycle.

### 3. onSuccess/onSettled placement — correct, `useReseedTemplate` preserved as `onSuccess`-only

Verified each hook's existing success/settle choice was preserved and the
invalidation was added into the *same* callback, not a new one:
- `useCreateTemplate` — `onSuccess` (`useTemplates.ts:159-162`)
- `useUpdateTemplate` — `onSettled` (`useTemplates.ts:250-253`)
- `usePatchTemplate` — `onSuccess` (`useTemplates.ts:290-292`)
- `useDeleteTemplate` — `onSettled` (`useTemplates.ts:341-343`)
- `useReseedTemplate` — `onSuccess` only (`useTemplates.ts:370`); no
  `onError`/`onSettled` was added, so a failed reseed invalidates nothing.
  Confirmed live: reverting to pre-fix source and running the new
  `"useReseedTemplate invalidates nothing on failure"` test passed even in
  the *reverted* state (expected — that assertion doesn't depend on the fix)
  while the other 9 template tests and both tenant tests failed, which
  independently confirms `onSuccess`-only placement satisfies FR-5.6 without
  over-invalidating on error.
- `useCreateTemplatesBatch` — `onSuccess` (`useTemplates.ts:396-397`)
- `useUpdateTemplatesBatch` — `onSuccess` (`useTemplates.ts:434-435`)
- `useDeleteTemplatesBatch` — `onSettled` (`useTemplates.ts:491-492`)
- `useCreateTenantConfiguration` — `onSuccess` (`useTenants.ts:270-272`)
- `useUpdateTenantConfiguration` — `onSettled` (`useTenants.ts:339-341`)

No hook's callback was moved between `onSuccess` and `onSettled`; diffs show
only additive lines inside existing blocks.

### 4. Test honesty and style — pass

Both new test files (`useTemplates.socketInvalidation.test.tsx`,
`useTenants.socketInvalidation.test.tsx`) follow the existing
`renderHook`+`QueryClientProvider`+`vi.spyOn(qc, "invalidateQueries")`
pattern used by `useReseedTemplate.test.tsx` and `useSocketObjects.test.tsx`
in the same `__tests__/` directory — same wrapper helper, same assertion
style (`invalidate.mock.calls.map(...).toContainEqual(...)`).

Test-honesty check performed directly (not just trusting the implementer's
report): checked out the pre-fix versions of `useTemplates.ts`,
`useTenants.ts`, `useSocketObjects.ts` from `2846074af~1` into the worktree
and reran the two new test files. Result: **10 of 11 new tests failed**
against the pre-fix code (the 11th, the reseed-failure negative case, is
expected to pass either way since it asserts absence of invalidation). This
confirms the tests actually pin the new behavior rather than passing
vacuously. Worktree was restored to the commit's actual state afterward
(`git checkout HEAD -- <3 files>`).

### 5. Over-invalidation — acceptable at this scale

`socketKeys.all` (`["socket"]`) is the parent of both `socketKeys.matrix()`
and `socketKeys.tenantMatrix()`, so every template mutation also invalidates
the (currently unrendered) tenant-matrix key and vice versa for tenant
mutations. Given there are only two consumers of `socketKeys.all`
(`useSocketMatrixTemplates`, `useSocketMatrixTenants`), both cheap sparse
reads with a 30s `staleTime`, and the tenant one has no active subscriber
today, the blast radius of over-invalidation is negligible — not worth a
finer-grained split for this fix.

## Requirement-by-requirement (Fix (final) checklist)

- `useTemplates.ts` — invalidate `socketKeys.all` in all eight mutation
  hooks: DONE, verified above.
- `useTenants.ts` — same for the tenant-configuration hooks that write
  `socket`: DONE for the two hooks that write `TenantConfigAttributes`
  (`useCreateTenantConfiguration`, `useUpdateTenantConfiguration`); see
  Finding 1 for `useDeleteTenant`.
- Import-cycle avoidance via a new `socketKeys.ts` module: DONE, verified.
- Tests asserting each mutation invalidates `socketKeys.all`: DONE, verified
  to actually pin the new behavior.
- Defect B (`highestVersionKey`) untouched: DONE —
  `git diff 8d875caec 2846074af -- services/atlas-ui/src/pages/PacketMatrixPage.tsx`
  is empty, and `git diff --stat` for the range shows only the six files
  listed in the commit's own stat.

## Not evaluable

- Runtime/E2E confirmation that a hard-reload is no longer needed against a
  live cluster was not performed (out of scope for a unit-level code
  review; the bug file's own confirmation in Round 3 already established the
  root cause via live reload, and this review verified the code-level fix
  against that root cause).
