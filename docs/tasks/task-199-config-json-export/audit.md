# Plan Audit — task-199-config-json-export

**Plan Path:** docs/tasks/task-199-config-json-export/plan.md
**Audit Date:** 2026-08-07
**Branch:** task-199-config-json-export
**Base Branch:** main (merge base 626161f8b)

## Executive Summary

All 5 plan tasks were faithfully implemented; every code artifact listed in the plan's File Structure table exists with the exact exported symbols the plan specifies. Two deviations from the plan's literal code were made and are pre-authorized: the Task-3 tooltip root (commit `5fef71c40`) and the Task-5 strict-type-check fixes (commit `182554b1b`) — both verified below to be scoped exactly as described, with no runtime behavior change and no assertion weakened. Diff containment holds (`services/atlas-ui/**` + `docs/**` only). Gate results (test/build/lint) were supplied as already-established and are not re-run here; the one un-executed step is the manual browser click-through (plan.md Task 5 Step 5), explicitly handed back to the human partner per instruction — reported as deferred-by-decision, not a skipped task.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | `downloadJson` helper | DONE | `services/atlas-ui/src/lib/utils/download-json.ts:12-29` — matches plan verbatim (serialize-before-createObjectURL, `finally` teardown, `application/json` MIME). Test file has all 4 planned `it()`s (`download-json.test.ts`). Commit `246647e0e`. |
| 2 | `config-export` projection + filename | DONE | `services/atlas-ui/src/lib/utils/config-export.ts` exports `ConfigExportKind`, `ExportableConfigAttributes`, `ConfigExportMeta`, `toConfigExportPayload`, `configExportFilename` exactly as specified (lines 12, 24, 36, 62, 98). Not added to the utils barrel (verified: not present in `src/lib/utils/index.ts` re-exports search). Test file has all 13 planned `it()`s across the two `describe` blocks (`config-export.test.ts:36-160`). Commit `68bc56ef9`. |
| 3 | `useDetailActionBarState` + `ConfigExportButton` | DONE | `DetailActionBarContext.tsx:110-113` adds `useDetailActionBarState()` verbatim (diff at commit `b7e4d960a`). `ConfigExportButton.tsx` implements the button with `useTemplate`/`useTenantConfiguration` branch-by-kind, `disabled={!query.data}`, success/error toasts, `aria-hidden="true"` icon — all per plan. Test file has all 8 planned `it()`s. Commit `b7e4d960a`, refined by `5fef71c40` (see Deviation 1 below). |
| 4 | Wire button into both detail layouts | DONE | `TemplateDetailLayout.tsx:9,31-38` and `TenantDetailLayout.tsx:9,32-39` both import `ConfigExportButton`, wrap the header in `flex items-start justify-between gap-4`, and render `<ConfigExportButton kind="template"/"tenant" id={id} />` inside the existing `DetailActionBarProvider` — matching the plan's replacement blocks exactly. New test files `TemplateDetailLayout.test.tsx` / `TenantDetailLayout.test.tsx` present and asserting the Export button's `data-kind`/`data-id`. Commit `0a42bd387`. |
| 5 | Full gate run | DONE (with one step explicitly deferred) | Diff containment confirmed independently: `git diff --name-only main...HEAD` returns only `docs/tasks/task-199-config-json-export/*` and `services/atlas-ui/**` paths — no Go module touched. `npm run test` (234 files/1890 tests), `npm run build` (tsc -b + vite build), `tools/lint.sh --check` all reported clean per the task brief and not re-run here. Step 5 (manual browser click-through against a live seed file) was **not performed** — see Skipped/Deferred section. Type-check fixes landed in commit `182554b1b` (Deviation 2 below). |

**Completion Rate:** 5/5 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0 (one sub-step of Task 5 explicitly deferred by decision, documented separately)

Note: plan.md's own checkbox markers (`- [ ]`) were never flipped to `- [x]` in the committed file — all 29 remain unchecked. This is a documentation-hygiene gap only; the six feature commits (`246647e0e`, `68bc56ef9`, `b7e4d960a`, `5fef71c40`, `0a42bd387`, `182554b1b`) map 1:1 onto the plan's five tasks in order, and file-level evidence for each is cited above.

## Skipped / Deferred Tasks

**Task 5, Step 5 — manual browser verification against a live seed file (plan.md:1092-1102).** Not performed. Per the audit brief, the human partner explicitly ruled this step is handed back to them rather than executed by the agent. This is reported as a deliberate deferral, not an implementer failure — no automated substitute exists for "run `npm run dev`, click Export, inspect the downloaded file," and every claim the step would have verified (key order, trailing newline, opCode sort) is independently covered by the unit tests in Tasks 1–3 (`config-export.test.ts` "preserves the seed-file key order" and "sorts handlers and writers ascending by numeric opCode"; `download-json.test.ts` "writes a pretty-printed JSON blob with a trailing newline"). Residual risk is narrow: a real API response shape diverging from the test fixtures would not be caught until a human runs this step.

## Deviation Verification (pre-authorized, not defects)

**1. Task 3 tooltip branch (plan.md:847, commit `5fef71c40`).** The plan's literal code was `if (!actionBar?.dirty) return button;` — an early return that swaps the component's root element type (`<Button>` vs `<Tooltip>`) across dirty flips, which a code-review pass found would remount the button and drop keyboard focus. What actually landed in `ConfigExportButton.tsx:90-97`:

```tsx
return (
  <Tooltip>
    <TooltipTrigger asChild>{button}</TooltipTrigger>
    {actionBar?.dirty ? (
      <TooltipContent>Exports the last saved configuration</TooltipContent>
    ) : null}
  </Tooltip>
);
```

This is exactly the described fix: a single always-mounted `<Tooltip>` root, with only `<TooltipContent>` conditionally rendered on `dirty`. `TooltipTrigger asChild` wraps the same `button` element in both states, so the DOM node is stable across the flip. Confirmed as landed as described.

**2. Task 5 strict-mode type fixes (commit `182554b1b`).** Diff inspected directly (`git show 182554b1b -p`). Changes are type-annotation-only:
- `ExportableConfigAttributes` / `ConfigExportMeta` fields widened from `region?: string` to `region?: string | undefined` (and same for the two version fields) — satisfies `exactOptionalPropertyTypes`.
- `toConfigExportPayload`'s internal spread source is cast to the named `ExportableConfigAttributes` interface before being spread into the `Record<string, unknown>`, with a comment explaining the generic-`T` spread was previously unable to prove index-signature compatibility.
- Two test files gained `noUncheckedIndexedAccess`-satisfying guards: `blobs[0].type` → `blobs[0]?.type` (same assertion, `?.` is a no-op when the array has the expected single element from the immediately-preceding `expect(blobs).toHaveLength(1)`), and `vi.mocked(downloadJson).mock.calls[0][0]` → `vi.mocked(downloadJson).mock.calls[0]?.[0]`, plus one call captured into a variable with `expect(call).toBeDefined()` before a non-null assertion.

None of these changes alter what is asserted or what code path runs at runtime — `tsconfig.app.json:22,27` confirms both `noUncheckedIndexedAccess` and `exactOptionalPropertyTypes` are `true` in this worktree, so the fixes are load-bearing for `npm run build` rather than cosmetic. No test assertion was weakened (every `expect(...)` call present before the commit is still present and still checks the same value) and no runtime behavior changed (`config-export.ts`'s actual field-copy logic is untouched; only its declared types and one type annotation moved).

## Build & Test Results

Per the audit brief, these gates were already established prior to this audit and are reported here without being re-run:

| Service | Build | Tests | Notes |
|---------|-------|-------|-------|
| atlas-ui | PASS | PASS | `npm run build` (`tsc -b && vite build`) clean; `npm run test` 234 files / 1890 tests passing; `tools/lint.sh --check` → `lint.sh: OK`, exit 0. |

Independently re-verified in this audit: `git diff --name-only main...HEAD` contains only `docs/tasks/task-199-config-json-export/*` and `services/atlas-ui/**` paths — no Go module, k8s manifest, or `services.json` entry touched, consistent with PRD §10's final acceptance criterion.

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE

## PRD §10 Acceptance Criteria Coverage

| Criterion | Status | Evidence |
|---|---|---|
| Export button in Template Details header, all 6 sub-tabs | DONE | `TemplateDetailLayout.tsx:38` — layout-level placement means every routed sub-tab child renders under the same header. |
| Export button in Tenant Details header, all 7 sub-tabs | DONE | `TenantDetailLayout.tsx:39` — same layout-level placement pattern. |
| Template filename `template_<region>_<major>_<minor>.json` | DONE | `config-export.ts:98-116` `configExportFilename`; test `config-export.test.ts:126`. |
| Tenant filename `tenant_<region>_<major>_<minor>.json` | DONE | Same function, `kind="tenant"`; test `config-export.test.ts:132`. |
| Downloaded object has no `data`/`type`/`id` key | DONE | `toConfigExportPayload` operates only on `.attributes` (`ConfigExportButton.tsx:57`); test `config-export.test.ts:36` "emits no JSON:API envelope keys". |
| Key set/formatting matches seed file closely enough for a clean diff | PARTIAL (automated coverage only) | `config-export.test.ts:44` "preserves the seed-file key order" asserts the exact key list; `download-json.ts` produces `JSON.stringify(payload, null, 2) + "\n"` matching seed formatting. The live-file diff against a real seed (plan Task 5 Step 5) was not run — see Skipped/Deferred section. |
| `socket.handlers`/`socket.writers` strictly ascending numeric opCode | DONE | `byOpCode` sort in `config-export.ts:43-45`; tests at `config-export.test.ts:76,86`. |
| `npcs`/`worlds` are `[]` not `null` | DONE | `config-export.ts:74-75`; test `config-export.test.ts:60`. |
| Button disabled while loading/errored | DONE | `disabled={!query.data}` (`ConfigExportButton.tsx:74`); tests `ConfigExportButton.test.tsx` "is disabled until the query has data" / "stays disabled when the query errors". |
| No additional network request when cached | DONE | Test "fires no additional request when clicked with the resource cached" — call count unchanged after click. |
| Success/error toasts | DONE | `toast.success`/`toast.error` in `onExport` (`ConfigExportButton.tsx:59,63`); tests cover both paths. |
| No object URL leak on success or failure | DONE | `download-json.ts:23-28` `finally` block always calls `revokeObjectURL`; test "revokes the object URL and leaves no anchor behind" and the throw-path test assert `createObjectURL`/`revokeObjectURL` are not called together on a serialization failure (revoke only follows a create). |
| Unit tests cover filename derivation (sanitisation + fallback), envelope stripping, null→[] normalisation, disabled-while-loading | DONE | `config-export.test.ts:136,142,151` (sanitisation/fallback), `:36` (envelope), `:60` (normalisation); `ConfigExportButton.test.tsx` "is disabled until the query has data". |
| `npm run test`, `npm run build`, `tools/lint.sh --check` all pass | DONE (as reported) | Per audit brief; not re-run in this pass. |
| No Go module changed / diff limited to atlas-ui + docs | DONE | Independently re-verified via `git diff --name-only main...HEAD`. |

## Action Items

1. Optional documentation hygiene: flip plan.md's 29 `- [ ]` checkboxes to `- [x]` (or add a closing note) so the committed plan reflects completion — purely cosmetic, does not block merge.
2. Before or shortly after merge, someone should perform plan.md Task 5 Step 5 (manual `npm run dev` click-through against a real seeded template) to close the one PRD criterion whose "closely enough for a clean diff" clause has only automated-fixture coverage, not a live-server confirmation. This was explicitly deferred to the human partner per this audit's instructions, not an implementer omission.
