# Frontend Audit — task-284-npc-ask-text-state

- **Audit Scope:** `services/atlas-ui` files changed in `9cd1ec5af..e6a1540cb` (commits `17d5b6361` "add askText state type" and `a4f2490ed` "add askText inspector panel with ordered matches editor")
- **Guidelines Source:** frontend-dev-guidelines skill
- **Date:** 2026-08-29
- **Build:** PASS (per branch's flagless `tools/verify.sh` gate against `e6a1540cb`, not re-run here per instructions)
- **Tests:** PASS (per branch's flagless verification gate; not re-run here per instructions)
- **Overall:** PASS

## Build & Test Results

Not re-executed in this audit session. Per the task instructions, `npm test` and the atlas-ui build already passed in the branch's flagless verification gate against `e6a1540cb`. Code was read directly instead.

## File Inventory

- `services/atlas-ui/src/components/features/npc/conversation/ConversationInspector.tsx` — **Component** (feature). Diff adds `askText` read-only `Section`, `AskTextForm`, `MatchesEditor`, `MatchRow` (+320 lines, no deletions).
- `services/atlas-ui/src/components/features/npc/conversation/editorOps.ts` — **Other** (pure editor-state transform module). Diff adds `askText` handling to `rewireStateRefs`, `emptyStateOfType`, and `setTransitionTarget` (now exported); +32/-1.
- `services/atlas-ui/src/components/features/npc/conversation/stateMeta.ts` — **Other** (state metadata/labels). Diff adds `askText` entry to `STATE_TYPE_META` and a `describeState` case; +13.
- `services/atlas-ui/src/components/features/npc/conversation/transitions.ts` — **Other** (transition/edge derivation). Diff adds `"match"`/`"fallback"` transition kinds and an `askText` case to `getTransitions`; +21.
- `services/atlas-ui/src/types/models/conversation.ts` — **Type**. Diff adds `AskTextMatch`, `AskTextState`, and wires `askText` into `ConversationStateType`/`ConversationState`; +18.
- `services/atlas-ui/src/components/features/npc/conversation/__tests__/ConversationInspector` tests:
  - `__tests__/AskTextForm.test.tsx` — new test file, RTL + user-event, covers add/remove/reorder/source-switch of matches.
  - `__tests__/editorOps.test.ts` — new test file, covers rename-rewire, delete-rewire, and `setTransitionTarget` addressing for `askText`.
  - `__tests__/transitions.test.ts` — new test file, covers `getTransitions` edge count/order/labels/no-dedup for `askText`.

No hooks (`lib/hooks/api/`), services (`services/api/`), or schemas (`lib/schemas/`) are in this diff — the feature is a local, in-memory editor-state transform with no server round trip in this change set, so several checklist items (tenant guards, query keys, service layer, Zod schemas) are not applicable to the changed files.

## Anti-Pattern Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-01 | No `any` type | PASS | `grep -n ": any\|as any"` across all 5 changed source files returns no matches. |
| FE-02 | No manual class concatenation | PASS | `grep -n 'className={"'` in `ConversationInspector.tsx` diff hunk returns no matches; all new JSX uses static `className="..."` strings, no concatenation. |
| FE-03 | No direct API client calls in components | PASS | `grep -n '@/lib/api/client'` in `ConversationInspector.tsx` returns no matches; the new code only calls `onUpdateState` (a prop callback), never an API client. |
| FE-04 | No inline Zod schemas in components | PASS | `grep -n 'z\.object\|z\.string'` in `ConversationInspector.tsx` returns no matches. `AskTextForm` performs no validation at all (see Not-evaluable note below) but does not construct an inline schema. |
| FE-05 | No spinners for content loading | PASS | `grep -n 'animate-spin'` in all changed files returns no matches. |
| FE-06 | No hardcoded colors | FAIL (pre-existing pattern, not a new deviation) | `services/atlas-ui/src/components/features/npc/conversation/stateMeta.ts:37-39` adds `accent: "bg-rose-500/15 text-rose-700 dark:text-rose-300 border-rose-500/30"` for the new `askText` entry — literal Tailwind color scale, not a semantic token (`bg-background`, etc.), which is a literal match on the FE-06 anti-pattern regex. However every other entry in the same `STATE_TYPE_META` record (`stateMeta.ts:14,18-19,23-24,28-29,33-34,43-44,48-49,53-54,58-59,63-64,68-69`) uses the identical raw-Tailwind-color idiom, so the new line is a mechanical continuation of an established, unchanged convention rather than a new violation introduced by this diff. |
| FE-07 | No state mutation | PASS | `MatchesEditor` (`ConversationInspector.tsx`, new `moveUp`/`moveDown`/`removeMatch`/`addMatch`/`updateMatch` functions) all copy via `const copy = [...matches]` / `const next = [...matches]` before mutating the copy, then call `onChange(copy/next)` — never mutates the `matches` prop array in place. `editorOps.ts`'s `askText` handling in `rewireStateRefs` (lines 59-70) builds `next` from `JSON.parse(JSON.stringify(state))` (line 33) before mutating, and `setTransitionTarget`'s new `"match"`/`"fallback"` cases (lines 599-606) mutate a `clone`, consistent with the rest of that function. |
| FE-08 | No default exports for components | PASS | `grep -n 'export default function'` in `ConversationInspector.tsx` returns no matches; `AskTextForm`, `MatchesEditor`, `MatchRow` are all `function X(...)` declarations used only within the same file (not exported), consistent with sibling `AskNumberForm`/`AskStyleForm`. |
| FE-09 | Tenant guard in hooks | N/A | No files under `lib/hooks/api/` are in this diff. |
| FE-10 | Tenant ID in query keys | N/A | No query key factories are in this diff. |
| FE-11 | Error handling with `createErrorFromUnknown` | N/A | No `.catch(` / async operations are introduced by this diff; all new code is synchronous local state transforms (`onUpdateState`, array copies). `grep -n '\.catch('` across changed files returns no matches. |

## Architecture Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-12 | JSON:API model shape | N/A / PASS | `AskTextState`/`AskTextMatch` (`types/models/conversation.ts:76-92`) are nested value objects inside the existing `ConversationState` model, not top-level JSON:API resources — no `id`/`attributes` shape is expected for them, consistent with sibling `AskNumberState`/`AskStyleState` in the same file. |
| FE-13 | Service extends `BaseService` | N/A | No `services/api/` files changed. |
| FE-14 | Query key factory uses `as const` | N/A | No query key factories changed. |
| FE-15 | Forms use `react-hook-form` + `zodResolver` | N/A (pre-existing pattern) | `AskTextForm` (`ConversationInspector.tsx`, new code) is a direct-manipulation editor panel that calls `onUpdateState` on every keystroke/click — it does not use `react-hook-form` at all. This matches the pre-existing pattern for every other `Ask*Form`/`*Form` panel in the same file (e.g. `AskNumberForm` at `ConversationInspector.tsx:1334` in the base commit), none of which use `react-hook-form`/Zod — this is a graph-editor surface, not a submit-driven CRUD form, so the guideline's form pattern does not apply here and the new code is consistent with the file's existing convention. |
| FE-16 | Schema in `lib/schemas/` with inferred type | N/A | No Zod schema is added or needed; see FE-15. |

## Testing Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-17 | Tests exist for changed components | PASS | `services/atlas-ui/src/components/features/npc/conversation/__tests__/AskTextForm.test.tsx` (197 lines) exercises the new `AskTextForm`/`MatchesEditor`/`MatchRow` UI end-to-end via `ConversationInspector`, including add/remove/move-up/move-down/source-switch. `__tests__/editorOps.test.ts` (96 lines) and `__tests__/transitions.test.ts` (60 lines) cover the non-component logic changes. |
| FE-18 | Mocks updated when services changed | N/A | No `services/api/` files changed, so no `__mocks__/` update is required. |

## Ordering / stale-id review (focus areas)

**Ordering preservation:** The `matches` array's index order is the single source of truth for match priority end-to-end, and it is exercised at every layer:
- `transitions.ts:111-128` (`getTransitions`, `askText` case) emits one edge per match in `matches.forEach` array order, then the fallback edge last — `__tests__/transitions.test.ts` asserts this exact order (`edges.map(e => e.target)` equals `["sa", "sb", "sa", "fallback"]`) and explicitly asserts non-deduplication of two matches pointing at the same target (`transitions.test.ts:47-51`), which matters because first-match-wins semantics require duplicate targets to remain distinct, ordered edges.
- `ConversationInspector.tsx`'s new `MatchesEditor` `moveUp`/`moveDown` (diff, `MatchesEditor` function) swap adjacent array elements via `const next = [...matches]; next[i-1] = next[i]; next[i] = tmp;`, preserving all other rows' relative order — `__tests__/AskTextForm.test.tsx:127-159` ("moves an entry down..." / "moves an entry back up...") asserts this against the underlying `matches` state, not just the DOM.
- The read-only `Section` view (`ConversationInspector.tsx`, new `askText` case ~line 508) renders `(state.askText.matches ?? []).map((m, i) => ...)` in array order with no re-sort, so the order the editor produces is the order displayed.
- `setTransitionTarget`'s new `"match"` case (`editorOps.ts:599-602`) retargets `clone.askText.matches[ordinal]` by index, and `ordinal` is assigned by `getTransitions` in the same array-order pass (`transitions.ts:121`), so index-addressed edits stay aligned with display order — `__tests__/editorOps.test.ts` ("retargets match index 1 without touching matches 0, 2, or the fallback") confirms this.

No gap found: ordering is preserved through create, reorder, remove, retarget, and both the read-only and edit views, and each of those paths has a corresponding test that inspects the underlying data (not just rendered text).

**Stale-id self-fallback ruling (`editorOps.ts:59-70`, deferred from a prior review):**

Verdict: **not a new defect introduced by this diff — it is a faithful, tested continuation of a pre-existing repo-wide idiom, not a regression to block on.**

- The pattern in question is `next.askText.matches = next.askText.matches.map((m) => ({ ...m, nextState: rewire(m.nextState) ?? m.nextState }))` (`editorOps.ts:65-68`) and the analogous single-value line `next.askText.nextState = rewire(next.askText.nextState) ?? next.askText.nextState` (`editorOps.ts:60-62`).
- When `rewire` is `deleteState`'s `rewireNull` (`editorOps.ts:247-248`: `id && toRemove.has(id) ? null : (id ?? null)`), a reference to a state being deleted evaluates to `null`, and `null ?? m.nextState` returns `m.nextState` — i.e. the **pre-delete, now-dangling id is left in place** rather than cleared. This is a real latent behavior (a deleted state's id can survive in `matches[].nextState`), but:
  - The identical idiom already exists at the base commit for `askNumber.nextState` — confirmed via `git show 9cd1ec5af:services/atlas-ui/src/components/features/npc/conversation/editorOps.ts` lines 55-58 (`next.askNumber.nextState = rewire(next.askNumber.nextState) ?? next.askNumber.nextState;`), i.e. before either commit in this diff's range. The new `askText` code is not introducing a new anti-pattern; it is applying the file's existing convention to a new field.
  - Both `AskTextMatch.nextState` (`types/models/conversation.ts:76-79`) and `AskNumberState.nextState` (`types/models/conversation.ts:67-74`) are typed as required `string` (no `| null`), so the `?? null` idiom used elsewhere in the same function for optional/choice-array fields (e.g. `dialogue.choices[].nextState: rewire(c.nextState) ?? null`, `editorOps.ts:34-37`) is not type-compatible here — the existing convention specifically falls back to the pre-delete value for required-string fields, in a way that is deliberate rather than an oversight.
  - The new test `__tests__/editorOps.test.ts:60-73` ("rewires every matches[].nextState referencing the deleted state, matching askNumber's delete behaviour") names this behavior explicitly and documents it in a comment: *"Matches the existing askNumber pattern: required string fields are left with their pre-delete value rather than nulled, since the type does not allow null."* This is a conscious, tested decision to mirror existing behavior, not a silently-introduced or masked bug.
- Net effect: deleting a state that a `askText` match points at leaves a dangling id in `matches[].nextState`, exactly as deleting a state that an `askNumber.nextState` points at already does today. This is pre-existing latent tech debt in the editor (arguably `AskTextMatch.nextState`/`AskNumberState.nextState` should be nullable, or the delete path should special-case required-string fields), but it predates this branch, is not worsened or newly introduced by it, and is intentionally tested rather than hidden. **Non-blocking** for this diff; flagged here as a repo-wide item worth its own follow-up rather than a gate on this change.

## Not evaluable from the diff

- FE-04 exception scope (whether `AskTextForm`'s min/max length inputs should have cross-field `.refine()`-style validation, e.g. enforcing `minLength <= maxLength`): the diff performs no validation on these fields at all (`Number(e.target.value)` is written straight through with no bounds check). This mirrors the pre-existing `AskNumberForm` pattern (not itself in the diff), so judging whether the lack of validation is a real gap vs. established convention would require reading `AskNumberForm`'s current behavior outside the diff; not read, per scope.
- Whether `ConversationCanvas.tsx` (unchanged, not in diff) renders the new `"match"`/`"fallback"` transition kinds with any special edge styling (e.g. distinguishing them visually from `"choice"` edges) was not checked — the file is outside the diff and `getTransitions`'s generic `Transition` shape means it should render generically, but this was not confirmed by reading `ConversationCanvas.tsx`.

## Summary

### Blocking (must fix)
- none

### Non-Blocking (should fix)
- FE-06 — `stateMeta.ts:37-39` uses a raw Tailwind color (`bg-rose-500/15` etc.) instead of a semantic token for the new `askText` accent; matches the existing convention for all other state types in the same file, so fixing it in isolation would create inconsistency — better addressed as a repo-wide `STATE_TYPE_META` theming pass, not a blocker on this diff.
- Stale-id self-fallback in `editorOps.ts:59-70` (and the pre-existing `askNumber` equivalent it mirrors) leaves dangling `nextState` references after `deleteState` for required-string fields — pre-existing behavior, intentionally tested and documented in this diff, worth a follow-up task to either make these fields nullable or special-case them in `deleteState`, but not a regression introduced here.
