# bug: CashShopEntryHandle @ 0x2B not visible after reseeding the GMS 95.1 template

Task: task-146-v95-packet-verification-batch (merged to `main` as 845a5c9;
overlay bumped in 8d875caec). Environment: cluster context `bee`, namespace
`atlas-main`.

## Reproduced

Partially. The reported symptom (Packet Matrix does not show
`CashShopEntryHandle` at `0x2B`) could NOT be reproduced from data: every layer
the Packet Matrix reads carries the route. A different, real drift WAS found
one layer down (tenant configuration), which is recorded below.

## Observed

Live checks, all against `atlas-main`:

- Deployed image: `atlas-configurations:main-845a5c9` (the task-146 merge
  commit); `atlas-channel:main-845a5c9`; `atlas-ui:main-99fcbfe`.
- Seed file inside the running configurations pod contains the route
  (`grep -c CashShopEntryHandle .../template_gms_95_1.json` → 1).
- Stored template `3f15459e-def5-4220-a1b5-8270e552ed6b` (GMS 95.1) returns
  `{"opCode":"0x2B","validator":"LoggedInValidator","handler":"CashShopEntryHandle",
  "fname":"CWvsContext::SendMigrateToShopRequest","services":["channel"]}`,
  162 handlers, `seedDrift: false`, `storedRevision == shippedRevision`
  (`725a22cb…`). The reseed did apply.
- The sparse read the matrix actually uses
  (`GET /api/configurations/templates?fields[templates]=region,majorVersion,minorVersion,socket`)
  returns the same entry — 11 objects, 162 handlers on the v95 column.
- Client render path reviewed and found sound for this case:
  `lib/socket/normalize.ts` (keys bindings by handler name, no dedupe),
  `lib/socket/matrix.ts` `buildRows`/`withOpcodeGaps` (0x2B is bound by exactly
  one definition in the v95 baseline, so it is `occupied`, not a gap), and
  `PacketGridCell` (renders `lowestOpCodeValue`). Nothing in that path suppresses
  the row.
- `/packet-matrix` is templates-only: `useSocketMatrixTenants` is exported but
  has no caller (`grep` across `services/atlas-ui/src`).

## Expected

The GMS v95.1 column of the `CashShopEntryHandle` row shows `0x2B`.

## Root cause

NOT established for the matrix-display claim. Ruled out: stale seed file, stale
configurations image, an unapplied reseed, the sparse fieldset projection,
normalize/buildRows/gap-row/cell-render logic, and a tenant column leaking into
the templates-only matrix.

Remaining unruled-out causes for the display claim are all client-side/session
level (a stale React Query cache — `staleTime: 30_000`, no persistence — an
active toolbar filter, a non-v95 `baseline`/`cols` URL param, or the browser
pointed at a different environment; namespace `atlas-pr-1534` also exists).

**Separate, confirmed drift:** the GMS 95.1 **tenant** configuration
`c794c706-aea3-4882-90a6-a3b7ee314f52` carries only 132 handlers and is missing
31 handlers and 23 writers that the shipped template now has — including
`0x2B CashShopEntryHandle`, plus `0x29 MapChangeHandle`, `0x3F
NPCStartConversationHandle`, `0x43 StorageOperationHandle`, `0x63
CharacterAutoDistributeApHandle`, `0x70 PortalScriptHandle`, `0x8C
CharacterMultiChatHandle`, `0x92 PartyInviteRejectHandle`, `0x19 PongHandle`,
and writers `0x16B NPCConversation`, `0x1B PicResult`, `0x8F CashShopOpen`.
No opcode disagrees between the two; the tenant is strictly behind.

This is the configuration `atlas-channel` actually routes with, so cash-shop
entry (and the other seven routes) is unrouted at runtime on that tenant
regardless of what the template says. `POST /configurations/templates/{id}/reseed`
is template-only (`templates/resource.go:32`); there is no tenant equivalent, so
resetting the template never propagates to a tenant.

## Fix

No code fix identified yet — the actionable item is operational, and the
display claim needs one more observation from the reporter (which page, which
URL params, and whether a hard refresh clears it).

Files that would be in scope if a code fix is chosen:

- `services/atlas-configurations/atlas.com/configurations/templates/resource.go`
  and `.../processor.go` (`ReseedById`) — if a tenant-side "sync from template"
  is wanted.
- `services/atlas-ui/src/services/api/tenants.service.ts`,
  `services/atlas-ui/src/lib/hooks/api/useSocketObjects.ts` — if the tenant
  matrix (`useSocketMatrixTenants`, currently unused) should be surfaced so this
  drift is visible.

## Not yet answered

1. Which surface showed the missing entry — `/packet-matrix`, a template
   definition grid, or a **tenant** definition grid? If it was tenant-scoped,
   the tenant drift above is the whole explanation and there is no UI defect.
2. Does a hard refresh (or `/packet-matrix` with no `cols`/`baseline`/filter
   params) show the row?
3. Is the drifted v95 tenant supposed to track the shipped template? If so,
   should that be a manual copy, or a new tenant-side sync endpoint?

## Resolution

Unresolved. No commit yet.

---

## Round 2 — reproduction against the live payload

Ran the real client pipeline (`normalize.fromTemplate` -> `matrix.buildRows` ->
`sortRows` -> `withOpcodeGaps`) over the live sparse response, under vitest in
`services/atlas-ui` (scratch test, removed after the run).

**Finding A — the CashShopEntryHandle row is fully populated.** Every column is
`defined`, GMS v95.1 included:

```
GMS v12.1 0x15   GMS v48.1 0x20   GMS v61.1 0x25   GMS v72.1 0x27
GMS v79.1 0x26   GMS v83.1 0x28   GMS v84.1 0x28   GMS v87.1 0x2A
GMS v92.1 0x2D   GMS v95.1 0x2B   JMS v185.1 0x1F
```

The reported "GMS v95.1 cell is empty" is NOT reproducible from the data the
server currently serves.

**Finding B — separate real defect: the default baseline is JMS v185.1.**
`PacketMatrixPage.highestVersionKey` (`services/atlas-ui/src/pages/PacketMatrixPage.tsx:38-46`)
reduces on `(majorVersion, minorVersion)` only, ignoring `region`. JMS 185 > GMS
95, so with no `baseline` URL param the matrix silently baselines on JMS v185.1.
Consequences observed in the same run:

- the row is ORDERED at `0x1F` (JMS's opcode), between `ChannelChangeHandle@0x1E`
  and `CharacterMoveHandle@0x20` — not near 0x2B;
- `withOpcodeGaps` emits `{"gap":true,"opCodeValue":43}` — 0x2B renders as a
  blank "No Definition" gap row, because the JMS baseline does not bind 43.

So scrolling to 0x2B under the default baseline shows an empty gap row, and the
definition is 12 rows up. This is a genuine display defect independent of the
data.

**Transport ruled out.** The exact URL the browser issues, through the public
ingress (`http://dev.atlas.home/api/configurations/templates?fields[templates]=
region,majorVersion,minorVersion,socket`), returns 691323 bytes, 162 v95
handlers, including the 0x2B entry. Response carries no `Cache-Control`,
`ETag`, or `Last-Modified`.

## Root cause (updated)

- For "the row sits at the wrong opcode / 0x2B is a blank row": **Finding B**,
  the region-blind default baseline.
- For "the GMS v95.1 cell is empty": still unexplained by any server-side or
  render-path fact. Every layer checked serves the entry. The remaining
  candidate is the reporter's browser session state — React Query holds
  `socketKeys.matrix()` with `staleTime: 30_000` and the default 5-minute
  `gcTime`, and a within-SPA navigation reuses it, so a tab opened BEFORE the
  reseed can keep showing the pre-reseed (drifted) template. Needs a hard
  reload to confirm or refute.

## Fix (updated)

- `services/atlas-ui/src/pages/PacketMatrixPage.tsx` — `highestVersionKey` must
  not compare versions across regions.
- `services/atlas-ui/src/pages/__tests__/PacketMatrixPage.test.tsx` — regression
  test asserting the default baseline with a JMS template present.

## Not yet answered (updated)

1. Does a hard reload (Ctrl+Shift+R) on `/packet-matrix` make the GMS v95.1
   cell show `0x2B`? If yes, there is no second defect and only Finding B is
   actionable.
2. What should the default baseline be — highest version within a default
   region, or an explicit/persisted choice?

---

## Round 3 — root cause CONFIRMED

Reporter confirmed: a hard reload of `/packet-matrix` makes the GMS v95.1 cell
show `0x2B`. That settles it as a client cache-invalidation defect, and rules
out every server-side and render-path candidate above.

**Defect A (confirmed root cause) — template mutations never invalidate the
Packet Matrix's query key.**

`useSocketObjects.ts` deliberately keys the matrix's sparse reads under
`socketKeys.matrix()` / `socketKeys.tenantMatrix()`, separate from
`templateKeys.detail` / `templateKeys.lists`, so a sparse document can never
reach a write path. Correct in itself — but nothing outside the matrix page
ever invalidates that key:

- `useReseedTemplate` (`useTemplates.ts:341-354`) invalidates
  `templateKeys.detail(id)` and `templateKeys.lists()` only. Its own docstring
  claims "the drift badge clears without a manual reload" — true for the badge,
  false for the matrix.
- `grep -n socketKeys services/atlas-ui/src/lib/hooks/api/useTemplates.ts
  services/atlas-ui/src/lib/hooks/api/useTenants.ts` → no matches. NONE of the
  eight template mutation hooks touch it: `useCreateTemplate` (:144),
  `useUpdateTemplate` (:196), `usePatchTemplate` (:265), `useDeleteTemplate`
  (:293), `useReseedTemplate` (:341), `useCreateTemplatesBatch` (:364),
  `useUpdateTemplatesBatch` (:392), `useDeleteTemplatesBatch` (:423). Same for
  the tenant-config hooks.
- `useSocketMutation` (`useSocketObjects.ts:onSuccess`) DOES invalidate
  `socketKeys.all` — which is why edits made from inside the matrix appear
  immediately and edits made anywhere else do not.

Blast radius: `socketKeys.matrix()` carries `staleTime: 30_000` and React
Query's default 5-minute `gcTime`, so after ANY template write outside the
matrix page the grid serves the pre-write document until a hard reload or a
5-minute idle. The reseed path is simply the loudest instance — it rewrites the
whole document at once.

**Defect B (separate, still open) — region-blind default baseline.** As written
up in Round 2, Finding B. Unaffected by the reload; needs a product decision on
what the default should be.

## Root cause (final)

Defect A. `useReseedTemplate` — and every other template/tenant mutation hook —
omits `queryClient.invalidateQueries({ queryKey: socketKeys.all })`.

## Fix (final)

- `services/atlas-ui/src/lib/hooks/api/useTemplates.ts` — invalidate
  `socketKeys.all` in the `onSuccess`/`onSettled` of all eight mutation hooks.
- `services/atlas-ui/src/lib/hooks/api/useTenants.ts` — same for the tenant
  configuration mutation hooks that write `socket`.
- Watch for an import cycle: `useSocketObjects.ts` already imports
  `templateKeys` from `useTemplates.ts`, so importing `socketKeys` back the
  other way closes a loop. Extract `socketKeys` into its own module (e.g.
  `lib/hooks/api/socketKeys.ts`) rather than cross-importing.
- Tests: `services/atlas-ui/src/lib/hooks/api/__tests__/` — assert each
  mutation invalidates `socketKeys.all`.
- Defect B, if approved separately:
  `services/atlas-ui/src/pages/PacketMatrixPage.tsx` `highestVersionKey`
  + `services/atlas-ui/src/pages/__tests__/PacketMatrixPage.test.tsx`.

## Resolution

Root cause confirmed; fix not yet implemented. No commit yet.

---

## Resolution (final)

Fixed by **2846074af** on branch `fix/packet-matrix-cache-invalidation`
(worktree `.worktrees/fix-packet-matrix-cache-invalidation`, off main @ 8d875caec).

Defect A only; Defect B (region-blind default baseline in
`PacketMatrixPage.highestVersionKey`) is deliberately untouched and remains open.

Changes: new `services/atlas-ui/src/lib/hooks/api/socketKeys.ts` (leaf module, so
`useTemplates.ts`/`useTenants.ts` can import it without closing a cycle through
`useSocketObjects.ts`, which re-exports it for existing importers);
`socketKeys.all` invalidation added to all eight template mutation hooks and to
`useCreateTenantConfiguration` / `useUpdateTenantConfiguration`;
`useReseedTemplate`'s stale docstring corrected; two new test files under
`lib/hooks/api/__tests__/`.

Gates:
- `tools/verify.sh --quick --base 8d875caec` → exit 0 (8 changed paths, atlas-ui
  lint & format guard passed).
- Flagless `tools/verify.sh` → exit 0, "All checks passed." Executed: atlas-ui
  lint & format guard, atlas-ui tests + build. Every Go/bake/guard step reported
  itself not-applicable ("no Go module changed", "no go.mod touched", etc.) —
  the script's own applicability logic, not a flag-based skip, so this run
  counts as the real gate.
- `npx vitest run src/lib/hooks/api` → 270/270; `npx tsc --noEmit` clean.
- `task-reviewer` (sonnet) → APPROVED_WITH_FINDINGS, 0 blocking, 1 non-blocking.
  Artifact: `review-cache-invalidation.md`. It confirmed test honesty by
  reverting the three source files to `2846074af~1` and observing 10 of 11 new
  assertions fail (the 11th is the reseed-failure negative case, expected to pass
  either way).

Live confirmation: the ORIGINAL symptom was confirmed live (hard reload cleared
it, Round 3). The FIX itself has NOT been re-tested against the cluster — the
atlas-ui image would need rebuilding and deploying first. Not yet verified live.

### Follow-ups (not done here)

1. **Defect B** — `PacketMatrixPage.highestVersionKey`
   (`services/atlas-ui/src/pages/PacketMatrixPage.tsx:38-46`) compares
   `(majorVersion, minorVersion)` across regions, so JMS 185 > GMS 95 and the
   default baseline is JMS v185.1. Under that baseline the
   `CashShopEntryHandle` row is ordered at `0x1F` and `0x2B` renders as a blank
   "No Definition" gap row. Needs a product decision on the default rule.
2. **`useDeleteTenant`** (`useTenants.ts:165-201`) does not invalidate
   `socketKeys.all`. Inert today — `useSocketMatrixTenants` has no non-test
   caller — but a latent trap once the tenant matrix is surfaced. Reviewer
   judged it correctly out of scope for this commit.
3. **Stale GMS 95.1 tenant config** `c794c706-aea3-4882-90a6-a3b7ee314f52` is 31
   handlers and 23 writers behind the shipped template (see Round 1). Unrelated
   to this fix; still unaddressed. Template reseed does not propagate to
   tenants, and there is no tenant-side reseed endpoint.
