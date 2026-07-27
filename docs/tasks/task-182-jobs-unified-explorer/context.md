# Jobs Unified Explorer — Execution Context

Task: task-182-jobs-unified-explorer · Worktree: `.worktrees/task-182-jobs-unified-explorer` · Branch: `task-182-jobs-unified-explorer`

Companion to [`plan.md`](plan.md). Spec: [`design.md`](design.md); requirements: [`prd.md`](prd.md); visual contract: [`ux-mock.html`](ux-mock.html) (open it in a browser — its `<script>` is the reference implementation for the grid math, rail groups, and gating behavior).

## What this task does

Replaces two pages in `services/atlas-ui` with one three-pane explorer at `/jobs` + `/jobs/:jobId`: branches rail · tier-aligned advancement flow + filterable skill list · skill detail with level slider. Also reparents GM (900) under Beginner (0) and Super GM (910) under GM in the display graph. Frontend-only; no Go, deploy, or endpoint changes.

## Key files

| Area | Path |
|---|---|
| Display graph + new helpers | `src/lib/jobs/job-advancement-tree.ts` |
| Rail groups + accents map | `src/components/features/jobs/rail-groups.ts` (new) |
| Panes | `src/components/features/jobs/{branch-rail,advancement-flow,skill-list,skill-detail,skill-icon}.tsx` (new) |
| Page (router + queries only) | `src/pages/JobsPage.tsx` (rewritten) |
| Deleted | `src/pages/JobDetailPage.tsx` + `src/pages/__tests__/JobDetailPage.test.tsx` |
| Routing | `src/App.tsx` (drop `JobDetailPage` lazy import; `/jobs/:jobId` → `JobsPage`) |
| Accent tokens | `src/index.css` (`--c-*` + `--acc-fg` in `:root` AND `.dark`) |
| Viewport hook | `src/hooks/use-media-query.ts` (new; `use-mobile.tsx` untouched) |
| Breadcrumbs | `src/lib/breadcrumbs/routes.ts` — **no change needed**; test pin only |

Existing hooks consumed as-is: `useJobSkills(tenant, jobId)` → `UseQueryResult<number[]>`; `useJobSkillDefinitions(tenant, ids)` → `{ definitions, isLoading, isError }` (per-skill `useQueries`, 30 min staleTime — switching jobs must not refetch cached defs). Skill display helpers: `deriveSkillType`, `resolveSkillName`, `formatSkillDescription`, `buildLevelTable` (all existing, unchanged).

## Locked decisions (from design.md)

- **D1 — URL is the single source of truth.** Job = route param, skill = `?skill=`. Selections `navigate()`/`setSearchParams()` push; every normalization (unknown/version-hidden jobId, stale `?skill=`, tenant switch) uses `{ replace: true }`. Filter text and slider level are local state, not URL.
- **D2 — Tier alignment via CSS Grid explicit placement.** Anchors (ancestors + entry from `jobTreePath`) span all rows at columns 1..n; chain node k of path r at column `anchors+1+k`, row `r+1`. No measurement code, no layout lib.
- **D3 — One `SkillDetail`, two hosts.** Wide (≥1150px, `useMediaQuery`): persistent third column. Narrow: `Sheet side="right"`; `onOpenChange(false)` clears `?skill=`. Slider reset via `key={def.id}` remount.
- **D4 — Accents are theme tokens.** Components receive the token *name* (`"--c-warrior"`) and scope it: `style={{ "--acc": "var(--c-warrior)" }}`; CSS uses `hsl(var(--acc)/…)`. Never hard-code colors per component.
- **D5 — Graph edits stay in `job-advancement-tree.ts`.** New pure exports: `advancementChains(entryId, major): number[][]` (chains **exclude** the entry — design §6's `[[900,910]]` note is a typo; D2's "below the entry" text and the mock's `chainsFrom` govern), `tierLabel(id)`, `subtreeCount(entryId, major)`.

## Gotchas / environment facts (verified during planning)

- `src/test/setup.ts` globally stubs `window.matchMedia` with `matches: false` → components default to the **narrow** layout in tests. The page test mocks `@/hooks/use-media-query` explicitly; the hook's own test installs a controllable stub.
- Tailwind 4 ships the `aria-pressed:` variant (precedent: `src/pages/ItemsPage.tsx`); selection styling relies on it.
- Branch path counts: Warrior/Magician have 3 advancement paths (10 nodes); Bowman/Rogue/Pirate have 2 (7 nodes); GM line is 1 path (2 nodes). Beginner chains at v83 = 13, at v12 = 11 (Pirate floor 62); `subtreeCount(0,12)=37`, `(0,62)=44`.
- After reparenting, `JOB_ROOTS === [0, 800, 1000, 2000, 2001]` and `floorOf(900) === floorOf(910) === 1` (inherited from Beginner). `BRANCH_FLOORS` loses its 900/910 keys.
- `branchEntryOf(0)` falls back to the Warrior entry (Beginner isn't a rail entry) but the *job* selection stays 0 — `useJobSkills(tenant, 0)` works (hook guards `jobId >= 0`, definitions guard `skillId > 0`).
- Verbatim state strings are load-bearing (parity with the old `JobDetailPage`); see plan Global Constraints.
- `getJobNameById(110) === "Fighter"` (`src/lib/jobs.ts`) — used by the breadcrumb label-resolver pin.
- shadcn `TableRow` forwards `data-*` to the `<tr>`; React renders `data-selected={bool}` as `"true"`/`"false"` strings — the highlight tests rely on that.
- Old Jest-era test files are excluded from `tsc -b`, but **new** tests are type-checked by `npm run build` (see `reference_atlas_ui_build_typechecks_tests`). Keep new tests strict-clean.
- `npm` may need `source ~/.nvm/nvm.sh && nvm use 22` in this environment.

## Task order & dependencies

1 (graph reparent) → 2 (helpers) → 3 (tokens + rail-groups) → 4 (useMediaQuery) → 5 (SkillIcon) → 6 (BranchRail) → 7 (AdvancementFlow) → 8 (SkillList) → 9 (SkillDetail) → 10 (page rewrite + routing + deletion + breadcrumb pin) → 11 (gates).

Tasks 4–9 only depend on 1–3 (and 5 for 8/9); they could interleave, but run 10 strictly last — it deletes `JobDetailPage` and rewires `App.tsx`. Expect the old `JobsPage.test.tsx` to stay green through Tasks 1–9 (it doesn't assert on 900/910 as roots) and be replaced wholesale in Task 10.

## Verification gates (all must pass before finishing)

```
cd services/atlas-ui && npm run test && npm run lint && npm run build
tools/lint.sh --check       # repo root
```

Then `superpowers:requesting-code-review` (frontend-guidelines-reviewer + plan-adherence-reviewer) before any PR — see repo CLAUDE.md.
