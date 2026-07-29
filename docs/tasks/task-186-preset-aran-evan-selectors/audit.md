# Frontend Guidelines Audit — task-186 (preset Aran/Evan selectors)

Branch: `task-186-preset-aran-evan-selectors`
Diff scope: `git diff main...HEAD -- services/atlas-ui`
Mindset: FAIL-until-proven. Each item below is evidenced with file:line.

## Verdict: PASS

No blocking (HIGH/MEDIUM) findings. One cosmetic (LOW) doc-drift item.

## Verification performed

- `npx vitest run` targeted at every changed/new test file (24 files, 159
  tests) — all pass:
  `src/components/features/characters/presets/**`,
  `src/lib/hooks/__tests__/usePresetJobOptions.test.tsx`,
  `src/lib/jobs/__tests__/job-advancement-tree.test.ts`,
  `src/lib/hooks/api/__tests__/useSkillDefinition.test.tsx`,
  `src/lib/hooks/api/__tests__/useJobSkillDefinitions.test.tsx`,
  `src/components/features/rankings/__tests__/LeaderboardRow.test.tsx`.
- `npm run build` (tsc -b + vite build) — clean, no type errors.
- `npx eslint` on all core changed source files — clean, zero warnings.
- `grep -rn "presetJobs"` / `"jobLabel"` across `src/` — no dangling
  references to the deleted module or its old export name (except one test
  comment, see Finding 1).
- `git status --short` after all of the above — tree unchanged (build
  artifacts are gitignored).

## Findings

### Finding 1 — LOW — stale comment references the deleted `jobLabel` name

`services/atlas-ui/src/components/features/rankings/__tests__/LeaderboardRow.test.tsx:103`

```ts
// jobId 110 -> "Fighter" via jobLabel; the raw id must not be shown.
```

The component under test was switched from `jobLabel` (deleted
`presetJobs.ts`) to `jobName` (`job-advancement-tree.ts`) at
`LeaderboardRow.tsx:7`, but this pre-existing test comment wasn't updated. No
functional impact — the assertion itself (`getByText("Fighter")`) is correct
— but it's stale documentation that will confuse the next reader. Not part of
the `docs/tasks/.../findings.md` claims (which don't mention this file's test
comment), so it slipped through review. Cosmetic only; does not block.

## Areas checked, no issues found

**React Query / multi-tenancy (`usePresetJobOptions.ts`)**
- `useJobs(activeTenant)` is the tenant-scoped source (`lib/hooks/api/useJobs.ts:22-30`); its query key includes `tenant?.id` (`useJobs.ts:8-9`), so cache entries can't cross tenants and `TenantProvider`'s `queryClient.clear()` on tenant switch correctly invalidates it.
- `usePresetJobOptions` composes `useTenant()` + `useJobs()` rather than accepting an explicit tenant param — appropriate since it's a derivation hook under `lib/hooks/` (not a `lib/hooks/api/` resource hook), matching the file-responsibility convention in `services/atlas-ui/CLAUDE.md` ("plus data-derivation hooks").
- Both `JobCombobox` and `JobSkillsAddButton` call `usePresetJobOptions()` independently; because the underlying `useJobs` query key is identical for the same tenant, React Query dedupes to one shared cache entry — no redundant fetch.
- Pending/error fallback (`usePresetJobOptions.ts:27-31`): `!jobsQuery.isSuccess` returns the full `JOB_LIST` rather than an empty array. `isSuccess` is `false` for both pending *and* error React Query states, so both are correctly treated as "unknown" per the documented intent (`usePresetJobOptions.ts:17-21`) and covered by `usePresetJobOptions.test.tsx:45-53`. Verified this matches the existing precedent in `JobsPage.tsx` (`visibleRoots` docstring at `job-advancement-tree.ts:154-161` calls out the same isSuccess gate), so it's consistent with an established pattern, not a one-off.
- On `isSuccess` with a genuinely empty tenant job set, the picker would show zero rows — same behavior as the pre-existing `JobsPage`/`visibleRoots` gate, and the manual-numeric-id escape hatch (`JobCombobox.tsx:99-115`, `JobSkillsAddButton.tsx:107-128`) stays available regardless, so a picker is never fully blocked. Not a regression introduced by this branch.

**TypeScript strictness**
- No `any` in any changed/new source file (`job-advancement-tree.ts`, `usePresetJobOptions.ts`, `JobCombobox.tsx`, `JobSkillsAddButton.tsx`, `PresetCard.tsx`, `PresetEditor.tsx`, `LeaderboardRow.tsx`, `useSkillDefinition.ts`) — grepped for `: any`, `<any>`, `as any`, `any[]`.
- `tsc -b` (via `npm run build`) is clean.

**`presetJobs.ts` deletion — no dangling references**
- `grep -rn "presetJobs"` and `grep -rn "jobLabel"` across `services/atlas-ui/src` return zero hits outside the one stale test comment (Finding 1). All five consumers (`JobCombobox.tsx:11`, `JobSkillsAddButton.tsx` — via `usePresetJobOptions`, `PresetCard.tsx:7`, `PresetEditor.tsx:5`, `LeaderboardRow.tsx:7`) were switched to `jobName`/`usePresetJobOptions` from `job-advancement-tree.ts`. `ClassAppearanceSection.tsx:18` imports `JobCombobox` only (no direct `presetJobs`/`jobLabel` reference).

**Bounded 404 retry (`skillDefinitionRetry`, `useSkillDefinition.ts:52-61`)**
- Both consumers verified: `useSkillDefinition` (`useSkillDefinition.ts:76`, single query, preset row) and `useJobSkillDefinitions` (`useJobSkillDefinitions.ts:32`, `useQueries` batch, Jobs page skill list).
- Non-404 errors keep the original 3-attempt budget (`useSkillDefinition.ts:60`) — unchanged behavior, no regression.
- 404s now retry up to `SKILL_DEFINITION_404_MAX_RETRIES = 2` (was 0) with the query client's default exponential backoff (`min(1000*2^n, 30000)`, `lib/query-client.ts:18-19`) — adds at most ~1s+2s≈3s before giving up on a genuinely-invalid id.
- For the batch consumer (`useJobSkillDefinitions`), `skillIds` in its one real caller (`JobsPage.tsx:86-87`) comes from `useJobSkills(activeTenant, jobId).data` — i.e., the job's own `/jobs/{id}/skills` list, so per-skill 404s should be rare in steady state; the retries exist for the re-ingest-blip window the task's `findings.md` documents. Because `useQueries` runs the per-skill queries in parallel, the added retry latency is bounded to ~3s regardless of how many skills are in the batch — it does not multiply with skill count.
- `useJobSkillDefinitions`'s existing `isError` semantics (`results.every((r) => r.isError)`, `useJobSkillDefinitions.ts:44`, unchanged by this branch) mean the aggregate error state still only fires when *all* skills fail — consistent pre/post-change.
- Test coverage (`useSkillDefinition.test.tsx:60-87`) exercises the boundary (`MAX_RETRIES - 1` → true, `MAX_RETRIES` → false) for both the `"404"` and `"not found"` message paths, plus the untouched non-404 3-attempt path.

**`job-advancement-tree.ts` additions (`jobName`, `JOB_LIST`)**
- `jobName` falls back to `` `Job ${id}` `` for unmapped ids (`job-advancement-tree.ts:130-132`), preserving the "backend is the validator of record" contract the manual-id escape hatch depends on.
- `JOB_LIST` is derived from the existing `JOB_GRAPH` (`job-advancement-tree.ts:119-121`), not a new hand-maintained array — avoids reintroducing the exact staleness bug (`PRESET_JOBS` stopping at Super GM) this task fixes.
- Covered by `job-advancement-tree.test.ts` (JOB_LIST length/ordering/Aran-Evan-Cygnus membership).

**Test quality**
- `usePresetJobOptions.test.tsx` covers all three branches: tenant has Aran/Evan, tenant lacks them, and the pending/unknown permissive fallback.
- `JobCombobox.test.tsx` and `JobSkillsAddButton.test.tsx` mock `usePresetJobOptions` directly (isolating gating logic, which is tested separately) and cover: name display, unmapped-id fallback, name search + pick, Aran search + pick, tenant-without-Aran hiding, and the numeric manual-id escape hatch — both the "matches a listed id" and "unlisted id" paths.
- Tests use the project's established `vi.mock` + `renderHook`/`render` + `userEvent` patterns consistent with the rest of the atlas-ui test suite; no ad hoc test-only constructors.

## Scope note

This audit covers the files named in the task brief plus their direct
dependents/consumers (`ClassAppearanceSection.tsx`, `useJobs.ts`, `JobsPage.tsx`,
`query-client.ts`) to trace tenant-scoping and retry behavior end to end. One
unrelated pre-existing file, `query-provider.tsx`, still carries a stale
`"use client"` directive that `services/atlas-ui/CLAUDE.md` says should be
purged — it is **not** part of this branch's diff (confirmed via
`git diff main...HEAD`) and is out of scope for this audit.
