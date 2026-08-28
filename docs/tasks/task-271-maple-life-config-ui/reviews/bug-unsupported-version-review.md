# Review: bug-unsupported-version-node-and-derivation-status

**Range:** `3e0fd6fbc..dc8b47630` (single commit `dc8b47630`)
**Scope:** `services/atlas-ui/` only, matches the frontend-only scope stated in the bug doc.
**Requirement:** `docs/tasks/task-271-maple-life-config-ui/bug-unsupported-version-node-and-derivation-status.md`, `## Fix` inventory.

## Verdict

APPROVED

## Findings

### PASS — `supportsMapleLife` keys off the handler implementation name, never opcode/version

`services/atlas-ui/src/components/features/characters/maple-life/mapleLifeSupport.ts:14-27`

```ts
export const MAPLE_LIFE_HANDLER = "MapleLifeCheckNameHandle";
export function supportsMapleLife(socket: SocketConfig | undefined): boolean {
  return (
    socket?.handlers.some((h) => h.handler === MAPLE_LIFE_HANDLER) ?? false
  );
}
```

No opcode or majorVersion literal anywhere in the predicate. Doc comment cites the seed-data cross-check and explicitly calls out `gms_84_1` as the counter-example to a version-cutoff rule, matching the bug doc's warning verbatim. `__tests__/mapleLifeSupport.test.ts:24-37` asserts `true` across all four opcodes (`0x100/0x10E/0x12D/0x137`) with only the handler name held constant — this is the correct axis to test and it is tested.

### PASS — null-array crash fixed in all four call sites, independent of the nav change

- `mapleLifeEditorState.ts:150` — `config?.classes?.find(...)` (was `config?.classes.find`)
- `mapleLifeEditorState.ts:178` — `config?.looks?.find(...)` (was `config?.looks.find`)
- `mapleLifeEditorState.ts:314` — `isEmptyConfig`: `config === undefined || (config.classes?.length ?? 0) === 0` (was `config.classes.length`)
- `SeedFromTemplateDialog.tsx:68-69` — `mapleLife?.looks?.length ?? 0` / `mapleLife?.classes?.length ?? 0` (was `mapleLife?.looks.length`)

`__tests__/mapleLifeEditorState.test.ts:139-162` loads `{ looks: null, classes: null }` cast through the wire shape and asserts no throw, 10 absent drafts, and `isEmptyConfig === true`. This is a real regression test: reverting any one of the four `?.` insertions above throws a `TypeError` under the old code, confirmed by tracing the exact null-deref paths the bug doc names (`?.` guards `config`, not the nested array). The fix holds independently of `supportsMapleLife`/nav change — a deep link to an unsupported route still hits `buildDrafts`/`isEmptyConfig` via `mapleLifeReducer`'s `load` action before the layout's `supportsMapleLife` gate could ever suppress it, so this null-safety layer is load-bearing on its own, not merely redundant with Part 1.

### PASS — both detail layouts hide the node without an extra request or loading flicker

`TemplateDetailLayout.tsx:20` calls `useTemplate(String(id ?? ""))` — identical call signature to `TemplatesMapleLifePage.tsx:13` (`useTemplate(String(id ?? ""))`), so the React Query cache key matches and no second network round-trip is issued. Same check for tenants: `TenantDetailLayout.tsx:21` and `TenantsMapleLifePage.tsx:16` both call `useTenantConfiguration(id ?? "")` verbatim.

Nav item is built via a conditional spread (`...(supportsMapleLife(...) ? [...] : [])`), so before the query resolves (`socket` undefined) the predicate returns `false` and the item is simply absent until data arrives — no flicker of a stale/broken link, and once data arrives the array re-renders with the item present if supported. `TemplateDetailLayout.test.tsx` and `TenantDetailLayout.test.tsx` each add a "hidden when absent" and "shown when present" test asserting on `getByRole("link", { name: "Maple Life" })` / `queryByRole(...)`.

Page-level gate (`TemplatesMapleLifePage.tsx:17-30`, `TenantsMapleLifePage.tsx` mirror) renders a short notice instead of `<MapleLifeEditor>` only once `!isLoading && !error && data present`, so the existing loading/error paths are preserved unchanged as required ("Keep the loading/error paths as they are").

### PASS — derivation-status UI fully removed, no dead code

- `ClassSelector.tsx`: `Badge` import, `badgeText` const, and `<Badge>` render all removed (`git diff` shows exactly those 3 lines deleted, nothing else touched). `not configured` marker untouched.
- `IdentitySection.tsx`: `isDerived` const and the entire ternary block (muted note + `role="note"` warning box) removed; doc comment updated to drop the FR-5.1..5.5/FR-11.8 provenance language.
- `mapleLifeWarnings.ts`: `WARN.unconfirmedOrdinal` and the `draft.ordinal >= 2` push removed; `absentRow` and `unknownSpSkill` untouched.
- Repo-wide grep for `unconfirmedOrdinal|badgeText|isDerived` after the change returns zero hits — no orphaned references.
- `npx tsc -b --noEmit` and `npx eslint` on every touched file are clean (no unused-import warnings), confirming the `Badge` import removal didn't leave anything else dangling.

### PASS — tests are real assertions, not just deletions

- `IdentitySection.test.tsx` keeps "the job field stays editable for ordinal 2" but adds `expect(screen.queryByRole("note")).not.toBeInTheDocument()` — a positive assertion that the warning box is gone, not merely a deleted test.
- `mapleLifeWarnings.test.ts`'s remaining `warningMap` test changes its final assertion from `toContain(WARN.unconfirmedOrdinal)` to `toEqual([])`, positively pinning the new (empty) warning set for that fixture rather than silently losing coverage.
- `mapleLifeSupport.test.ts` and both `*DetailLayout.test.tsx` files are net-new, asserting both the true and false branch of the predicate with mocked query hooks.
- `mapleLifeEditorState.test.ts`'s new null-load test is a genuine failing-without-the-fix case (see above).

### Verification performed

- `npx tsc -b --noEmit` — clean, no errors.
- `npx eslint` on all touched source files — clean, no warnings.
- `npx vitest run` scoped to the maple-life directory plus both detail-layout test files — 15 files / 148 tests, all passing.
- `git log --oneline 3e0fd6fbc..dc8b47630` confirms a single commit in range, matching the stated scope.

## Not evaluable

- Live-browser re-test against a running `atlas-configurations` pod is explicitly out of scope per the bug doc's "Not yet answered" section and was not attempted here (static review only, matching the doc's own caveat).
- Backend response-shape correctness (`services/atlas-configurations/.../rest.go` non-pointer `MapleLife` field) is out of this diff's scope — the frontend fix correctly treats it as an established constraint rather than attempting to change it, per the bug doc's Part-1 framing.

## Scope confirmation

Reviewed exactly the 16 non-doc files changed by commit `dc8b47630` (`git diff --stat 3e0fd6fbc..dc8b47630`), all under `services/atlas-ui/`. No out-of-scope files touched. No scope mismatch.
