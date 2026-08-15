# Task 34 review — pending-changes panel, confirm dialog, page wiring

Commit reviewed: `f184386cb` (1 commit, +383 lines, 4 files: `PendingChangesPanel.tsx`,
`CancelPendingChangeDialog.tsx`, `PendingChangesPanel.test.tsx`,
`CharacterDetailPage.tsx`). Read-only review; no edits made.

## Verdict: PASS, with one Moderate defect (spec compliance) / PASS, with two non-blocking notes (FE-* code quality)

## 1. Scope fence (FR-2.10: read + cancel only) — PASS

`PendingChangesPanel.tsx` renders one row per `PendingChange` with a `Cancel`
button gated on `change.status === "PENDING"`
(`PendingChangesPanel.tsx:96-107`); no `create`/`add`/`edit`/`new` control
anywhere in either new component. Grepped both files for `button`/`onClick` —
the only interactive elements are the per-row `Cancel` button and the
dialog's `Keep Request`/`Cancel Request` actions
(`CancelPendingChangeDialog.tsx:75-96`). Test 5
(`PendingChangesPanel.test.tsx:127-134`, "exposes no create or edit
affordance") asserts `queryByRole("button", { name: /new|create|add|edit/i
})` is null and is load-bearing — adding any such button would fail it.

## 2. PRE-IDENTIFIED FINDING — `CancelPendingChangeDialog` names the character by ID, not by name — CONFIRMED, rate Moderate (should fix, not blocking)

Verified independently, not accepted on the implementer's framing.

- The dialog's props interface **does** include `characterName: string` as a
  required prop (`CancelPendingChangeDialog.tsx:29`), matching the brief's
  Produces line exactly. The dialog correctly renders it:
  `CancelPendingChangeDialog.tsx:60-62` — `<strong>{characterName}</strong>`.
  So the *component* is not missing the capability the brief specified.
- The break is one level up. `PendingChangesPanel.tsx:41-42` declares
  `characterName?: string` as **optional**, and when it is absent falls back
  to `` `Character ${characterId}` `` at `PendingChangesPanel.tsx:114`
  (`characterName={characterName ?? \`Character ${characterId}\`}`).
- `CharacterDetailPage.tsx:281` — the only call site —
  passes `<PendingChangesPanel characterId={String(id)} />` with **no**
  `characterName` prop. So in the shipped app the fallback always fires;
  the dialog will always read "Character 1", never the operator-recognizable
  name.
- This is not a missing capability — it's an unthreaded one. The page
  already resolves the real name: `character = characterQuery.data ?? null`
  (`CharacterDetailPage.tsx:64`), and `CharacterPageHeader.tsx:22` in the same
  page reads it as `character.attributes.name`. The one-line fix is
  `<PendingChangesPanel characterId={String(id)} characterName={character?.attributes.name} />`.
- Assessment against the brief's stated rationale ("so an operator cannot
  cancel the wrong record by muscle memory," Step 4): a numeric character ID
  *does* uniquely identify the record — an operator working from the ID in
  the URL/list would not be misled. But the rationale is about operators
  recognizing the character they intend to act on, which in this UI is
  normally done by name, not ID (see `CharacterPageHeader` rendering the name
  as the page's title, not the ID). "Character 1" does not let an operator
  confirm by the identifier they actually use day to day, so the stated
  safety purpose is only partially met. I rate this **Moderate, not
  Critical**: it is a real, one-line-fix omission at the wiring step (Step 5
  in the brief) that undercuts an explicitly-stated requirement, but it does
  not create a data-integrity risk (IDs are unique, so no wrong record can
  actually be cancelled) and every other part of the interface (types,
  dialog rendering, prop threading inside the two new components) is
  correct. Should be fixed before merge; not a shipped-bug-in-production
  severity issue.
- The implementer's own "Concerns" section in the report correctly
  identifies this and offers the exact fix — this review reaches the same
  conclusion independently from source, not from the self-report.

## 3. The six tests are load-bearing — PASS with one gap noted

Read `PendingChangesPanel.test.tsx` in full (150 lines, matches diff).

| Test | Behaviour | Load-bearing? |
|---|---|---|
| 1, `:56-66` | Lists pending name change w/ requested value + expiry | Yes — asserts `"Name Change"`, `"Zulu"`, `/PENDING/`; removing `TYPE_LABELS` or `requestedValue()` rendering fails it. |
| 2, `:68-75` | Shows rejection reason on resolved record | Yes — asserts `/REJECTED/` and `/name.taken/i`; removing the `reason` line at `PendingChangesPanel.tsx:90-93` fails it. |
| 3, `:77-84` | Cancel only on PENDING | Yes — asserts exactly 1 button; adding a Cancel button on the `appliedNameChange` row fails it. |
| 4, `:86-105` | Names character + requested value in dialog; no mutate before confirm | **Partial.** Only asserts `dialog).toHaveTextContent("Zulu")` (the requested value) and the `mutateAsync` call-order/args. It never asserts on character-identifying text in the dialog. The test's own name claims it covers "names the character," but the assertion doesn't cover that half — it would still pass with the `characterName` prop entirely deleted from `CancelPendingChangeDialog`, since the fallback text (`Character 1`) still renders *some* text and nothing in the test checks for it. This is exactly the gap in finding #2: a test that could have caught the missing wiring, but doesn't, because it doesn't assert on the character-naming half of its own stated behaviour. |
| 5, `:107-116` | No create/edit affordance | Yes (see §1). |
| 6, `:118-124` | Empty state | Yes — asserts `/no pending changes/i`; matches `PendingChangesPanel.tsx:70`. |

Five of six are fully load-bearing; the fourth is load-bearing for the
"confirm gates the mutation" half but not for the "names the character"
half its own title promises. Not "assert nothing" (the historical pattern
this branch was warned about) — a real, partial gap.

## 4. Test harness correctness — PASS

`PendingChangesPanel.test.tsx:6-16` mocks `@/context/tenant-context` and
`@/lib/hooks/api/usePendingChanges` at module scope, following
`TeleportRockListCard.test.tsx:6-24` exactly (hook-level mock, not
`vi.spyOn` on the service — the service can never be reached once the hook
is mocked, correctly avoiding the invented-service-spy pattern flagged in
the brief).

Verified the `Vars` shape against the real hook,
`usePendingChanges.ts:33-37`:
```ts
interface CancelVars {
  tenant: Tenant | null | undefined;
  characterId: string;
  id: string;
}
```
and the mutationFn at `usePendingChanges.ts:44-45`:
`pendingChangesService.cancel(v.characterId, v.id)`. The test's assertion
(`PendingChangesPanel.test.tsx:102-105`):
```ts
expect(mutateAsync).toHaveBeenCalledWith({
  tenant: { id: "t" }, characterId: "1", id: "pc-1",
});
```
matches the real field names (`tenant`, `characterId`, `id`) and the mocked
`useTenant()` return (`{ activeTenant: { id: "t" } }`,
`PendingChangesPanel.test.tsx:6-8`) exactly. This is correct, not
transcribed from the brief's invented `pendingChangesService.cancel("1",
"pc-1")` service-spy assertion, which — confirmed — could never fire under
this repo's mock-the-hook pattern.

## 5. Reference-component parity — PASS, one minor divergence noted

Matches `TeleportRockListCard.tsx` structurally: `Card`/`CardHeader`/
`CardTitle`/`CardContent` (`PendingChangesPanel.tsx:53-56`), `Badge` per
status (`STATUS_VARIANTS`, `:22-30`), `toast` from `sonner`
(`CancelPendingChangeDialog.tsx:3`), tenant via `useTenant().activeTenant`
(`PendingChangesPanel.tsx:49`, `CancelPendingChangeDialog.tsx:41`, never
hard-coded), errors via `createErrorFromUnknown`
(`PendingChangesPanel.tsx:59-62`, `CancelPendingChangeDialog.tsx:63`).
`AlertDialog` shape matches the destructive-confirm pattern (`buttonVariants({
variant: "destructive" })` at `CancelPendingChangeDialog.tsx:88`, matching
`DeleteServiceDialog.tsx`'s pattern per the brief).

Minor, non-blocking divergence: the panel's own-data-fetch loading state is
plain text — `<p ...>Loading...</p>` (`PendingChangesPanel.tsx:66-68`) —
rather than a `Skeleton`, unlike every other sibling component in this same
directory that owns its own query (`SkillWidget.tsx:27`,
`SkillsSection.tsx:26-31`, `AttributesPanel.tsx:174`,
`MonsterBookWidget.tsx:59`, all use `Skeleton`). `TeleportRockListCard`
itself isn't a counterexample — it receives `maps` as already-loaded props
from its parent and has no query/loading state of its own, so it wasn't a
usable precedent for this specific case. Not an FE-05 violation (no
`animate-spin` on content), but a real, fixable parity gap against the
directory's actual own-fetch convention.

## 6. Page wiring — PASS

`CharacterDetailPage.tsx:36` — import added directly beside the other
`@/components/features/characters/*` imports, next to `TeleportRockListCard`
(`:35`). `CharacterDetailPage.tsx:281` — `<PendingChangesPanel
characterId={String(id)} />` renders immediately after the
`teleportRocksQuery.data && (...)` grid block (`:265-278`) and before
`<Toaster>` (`:283`) — exactly the placement the controller's inventory pass
specified.

## 7. Import convention — PASS

`PendingChangesPanel.tsx:2-8` and `CancelPendingChangeDialog.tsx:1-16` import
`usePendingChanges`/`useCancelPendingChange` from
`@/lib/hooks/api/usePendingChanges` and the `PendingChange` type from
`@/services/api/pending-changes.service` directly — not the barrel. Correct
per the Task 33 ruling (barrel exports pending-changes types only,
deliberately).

## FE-* checklist (code quality)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-01 | No `any` | PASS | Grepped `: any`/`as any` across all 3 new files — zero matches |
| FE-02 | No manual class concat | PASS | Only `cn(buttonVariants(...))` at `CancelPendingChangeDialog.tsx:91`; no string-concat `className={"..." + ...}` anywhere |
| FE-03 | No direct API client calls in components | PASS | Grepped both components for `@/lib/api/client` — zero matches; both go through hooks |
| FE-04 | No inline Zod in components | PASS | No `z.object`/zod import in either file |
| FE-05 | No spinners for content loading | PASS (with note) | `animate-spin` appears once, on the dialog's submit button (`CancelPendingChangeDialog.tsx:100`) — allowed per the rule. Content loading uses plain text, not `Skeleton` — see §5 parity note (non-blocking, not a literal FE-05 violation) |
| FE-06 | No hardcoded colors | PASS | Only semantic classes (`text-destructive`, `text-muted-foreground`) — grepped for `bg-(white\|black\|gray-\d\|red-\d...)`, zero matches |
| FE-07 | No state mutation | PASS | `useState<PendingChange \| null>` replaced wholesale via `setCancelling`; no `.push`/`.splice`/`.sort` in either file |
| FE-08 | No default exports | PASS | `export function PendingChangesPanel` / `export function CancelPendingChangeDialog` — both named |
| FE-09 | Tenant guard in hooks | PASS (inherited) | `usePendingChanges`/`useCancelPendingChange` (Task 33) already gate via `enabled: !!tenant?.id && !!characterId`; not redefined here |
| FE-10 | Tenant ID in query keys | N/A | No new query key factory in this diff |
| FE-11 | Error handling via `createErrorFromUnknown` | PASS | `PendingChangesPanel.tsx:60`, `CancelPendingChangeDialog.tsx:63` both use it in the query-error and mutation-catch paths respectively |
| FE-14 | Query key factory `as const` | N/A | No new key factory in this diff |
| FE-16 | Schema paired with inferred type | N/A | No Zod schema — read + cancel only, no form |

## Testing checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-17 | Tests exist for changed components | PASS | `PendingChangesPanel.test.tsx` covers both new components (the dialog only through the panel, which is acceptable since it has no standalone public entry point outside the panel) |
| FE-18 | Mocks updated when services changed | N/A | No service interface changed in this task; only hook mocks used, consistent with Task 33's shipped `Vars` shape |

## Summary

### Blocking (must fix)
- None.

### Should fix before merge
- **Finding #2**: Thread the real character name into the confirm dialog.
  `CharacterDetailPage.tsx:281` should pass
  `characterName={character?.attributes.name}` to `<PendingChangesPanel>`,
  and `PendingChangesPanel.tsx:110-117` should forward it to
  `CancelPendingChangeDialog` unchanged (it already does, once given a real
  value). One line at the call site; no interface changes needed. Rated
  Moderate, not Critical — no data-integrity risk since IDs are unique, but
  it undercuts an explicitly-stated brief requirement and is nearly free to
  fix.
- **Finding #3 (test 4 gap)**: `PendingChangesPanel.test.tsx:86-105` should
  additionally assert the dialog contains character-identifying text (e.g.
  pass a `characterName` into `renderPanel()`/the component and assert
  `dialog.toHaveTextContent(<that name>)`), so the test actually pins the
  "names the character" half of its own stated behaviour and would have
  caught Finding #2.

### Non-blocking (should fix opportunistically)
- §5: `PendingChangesPanel`'s own loading state uses plain text instead of
  `Skeleton`, diverging from every other self-fetching sibling component in
  `components/features/characters/` (`SkillWidget`, `SkillsSection`,
  `AttributesPanel`, `MonsterBookWidget`).

## Fix round 1 re-review

Scope: fix diff `f184386cb..d636d6598` (1 commit, `d636d6598`, 3 files,
+9/-5). Read-only; no edits made.

### Finding 1 (Moderate — `characterName` never threaded) — ADDRESSED

- `CharacterDetailPage.tsx:284-287` now passes
  `<PendingChangesPanel characterId={String(id)} characterName={character?.attributes.name} />`
  — exactly the one-line fix the original review specified.
- Threading traced end to end:
  - `PendingChangesPanel.tsx:41` receives `characterName?: string | undefined`
    and forwards it unchanged at `PendingChangesPanel.tsx:116`:
    `characterName={characterName ?? \`Character ${characterId}\`}`.
  - `CancelPendingChangeDialog.tsx:29` declares `characterName: string`
    (required, unchanged from round 1) and genuinely renders it in the
    dialog's visible text: `CancelPendingChangeDialog.tsx:76`,
    `<strong>{characterName}</strong>`. This is not a prop pass-through that
    dead-ends — it lands in the rendered DOM.
- **Undefined-at-render-time check.** At the only call site
  (`CharacterDetailPage.tsx`), `PendingChangesPanel` is reachable only after
  the early-return guard at `CharacterDetailPage.tsx:109-117`
  (`if (loading) return <CharacterDetailSkeleton />; if (error || !character
  || ...) return <ErrorDisplay ... />;`). `character` is declared once
  (`CharacterDetailPage.tsx:64`, `const character = characterQuery.data ??
  null`) and never reassigned, so TypeScript control-flow narrows it to
  non-null for the remainder of the function body, including the JSX at
  line 285 — there is no in-between render where `character` is null but
  the panel is still mounted. `attributes.name` on the `Character` model is
  a required `string` (`services/atlas-ui/src/types/models/character.ts:12`),
  not optional. So `character?.attributes.name` is never actually `undefined`
  at this call site; the `?.` is a defensive habit, not a real fallback path
  in the shipped app. The `Character ${characterId}` fallback inside
  `PendingChangesPanel.tsx:116` is dead code at this call site today — not a
  bug, just belt-and-suspenders for any future caller that doesn't have the
  same guard.
- **Optional widening (`characterName?: string` → `characterName?: string |
  undefined`) — sound, not masking anything.** This change is required only
  because `exactOptionalPropertyTypes` rejects an explicitly-passed
  `undefined` against a bare `?: string` prop type, and the test's
  `renderPanel(characterName?: string)` helper (test file, same diff) passes
  `characterName={characterName}` where `characterName` is typed `string |
  undefined`. It is a type-signature accommodation for the test call site,
  not a runtime-relevant relaxation — the production call site never
  actually supplies `undefined` per the narrowing argument above.
- Verdict: **ADDRESSED**, threaded correctly page → panel → dialog, and the
  dialog's rendered output genuinely reflects the character's real name.

### Finding 2 (Important — naming test didn't pin the behaviour) — ADDRESSED

- `PendingChangesPanel.test.tsx:105` now calls `renderPanel("Bravestar")`
  (was `renderPanel()`), and `PendingChangesPanel.test.tsx:114` adds
  `expect(dialog).toHaveTextContent("Bravestar")` before the pre-existing
  `expect(dialog).toHaveTextContent("Zulu")` assertion.
- **Not circular.** Neither `CancelPendingChangeDialog` nor
  `PendingChangesPanel` is mocked in this test file — only
  `@/context/tenant-context` and `@/lib/hooks/api/usePendingChanges` are
  module-mocked (`PendingChangesPanel.test.tsx:8-18`), and neither of those
  mocks touches `characterName` or dialog rendering. The chain from
  `renderPanel("Bravestar")` to the asserted DOM text is: test helper prop
  → `PendingChangesPanel` prop → real `CancelPendingChangeDialog` render →
  `<strong>{characterName}</strong>` (`CancelPendingChangeDialog.tsx:76`).
  The assertion reaches actual rendered output produced by real component
  code, not an echo of a value the test injected through a mock.
- **Mutation-testing claim verified by structural read (not re-run, per
  scope constraints on this pass).** Manually reverting the test line to
  `renderPanel()` (no name) traces to: `PendingChangesPanel` receives
  `characterName={undefined}`, its fallback at `PendingChangesPanel.tsx:116`
  computes `` `Character ${characterId}` `` = `"Character 1"`, and the
  dialog renders `<strong>Character 1</strong>` instead of
  `<strong>Bravestar</strong>`. `expect(dialog).toHaveTextContent("Bravestar")`
  would genuinely fail against that output — the implementer's reported
  mutation-test result (`Expected: Bravestar / Received: ...for Character
  1...`) is consistent with tracing the code by hand; nothing in the diff
  contradicts it.
- The pre-existing `expect(dialog).toHaveTextContent("Zulu")` assertion is
  retained unchanged immediately after, so the test still covers both halves
  of its own title ("names the character and the requested value").
- Verdict: **ADDRESSED**. The test now pins the exact behaviour it is
  titled for, non-circularly, and would fail if `characterName` regressed
  to unthreaded.

### New breakage introduced by this diff

None found. Checked the diff's three hunks against the FE-* checklist and
for regressions:

- No new `any`, no manual class concatenation, no direct API client import,
  no inline Zod, no hardcoded color classes, no state mutation, no default
  export introduced by this diff — the only production change is the prop
  type widening and the two-line JSX addition in `CharacterDetailPage.tsx`,
  neither of which touches those categories.
- `CancelPendingChangeDialog.tsx` is untouched by this diff (not in the
  changed-files list) — its Finding-1 rendering behaviour, quoted above, was
  already correct in round 1 and is unaffected here.
- Test file change is additive (one new render arg, one new assertion) and
  does not remove or weaken any existing assertion — `toHaveTextContent("Zulu")`
  and the `mutateAsync` call-order/args checks from round 1 are untouched.

Deferred per instruction, not re-raised: `PendingChangesPanel`'s loading
state still uses plain text rather than `Skeleton` — unchanged by this diff,
already logged as non-blocking in the original review.

### Overall verdict: both open findings ADDRESSED. No new blocking issues from this fix diff.
