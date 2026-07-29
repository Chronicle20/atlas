# task-186 — Character Preset UI: Aran/Evan selectors + transient skill 404s

Two independent issues reported against the Character Preset web UI.

## Issue 1 — Aran/Evan cannot be selected (Class selector + Bulk Job Skill add)

### Root cause

Both selectors were driven by a hand-maintained curated array,
`PRESET_JOBS` in
`services/atlas-ui/src/components/features/characters/presets/presetJobs.ts`,
that stopped at `910 Super GM`. It contained **no Aran (2000, 2100, 2110–2112),
no Evan (2001, 2200–2218), and no Cygnus (1000–1512)** entries. Both
`JobCombobox` (Class) and `JobSkillsAddButton` (Bulk Job Skill add) filter that
array **by name**, so typing "Aran"/"Evan" matched nothing → "No matches" →
unselectable. Only a pure-digit manual-id escape hatch could reach them, and the
chosen job then rendered label-less as `Job 2100`.

The complete, version-aware data already existed in
`services/atlas-ui/src/lib/jobs/job-advancement-tree.ts` (`JOB_GRAPH`, with Aran/
Evan/Cygnus and the `visibleRoots`/`available` gating the Jobs page uses) — the
preset selectors just didn't use it.

### Fix (chosen direction: drive off JOB_GRAPH / GET /api/data/jobs)

- Added `jobName(id)` and `JOB_LIST` to `job-advancement-tree.ts` — the canonical
  id→name / full-entry list (covers Aran/Evan/Cygnus).
- Added `src/lib/hooks/usePresetJobOptions.ts`: returns the `JOB_GRAPH` entries
  gated by the tenant's ingested job set (`useJobs` → `GET /api/data/jobs`) — the
  same signal `JobsPage` uses. Aran/Evan appear on versions that ship them and
  are hidden on versions that don't. Pending/error → permissive full list (a
  picker must never be blank; the backend validates the chosen id).
- Rewrote `JobCombobox` and `JobSkillsAddButton` to source rows from
  `usePresetJobOptions()` and label via `jobName()`.
- Replaced `jobLabel` with `jobName` in `PresetEditor`, `PresetCard`,
  `LeaderboardRow` — Aran/Evan/Cygnus now render by name everywhere (previously
  `Job 2100`). Deleted the dead `presetJobs.ts` + its test.

### Live verification (atlas-main)

`GET /api/data/jobs` for GMS **v61 does not include** roots `2000/2001/2100`
(Aran/Evan absent); **v95 includes** `2000, 2001, 2100, 2110–2112, 2200–2218`.
So gating `JOB_GRAPH` by the tenant's job set yields Aran/Evan exactly on the
versions that support them.

## Issue 2 — Preset skills 404 from atlas-data after a baseline re-publish

### Root cause (verified — NOT stale data, NOT a missing baseline)

The 404s were **transient, during the baseline re-ingest**. Evidence from the
atlas-main ingress access log:

- All skill 404s came from one page: `.../tenants/ec876921…/character/presets`
  (tenant = **GMS v83**), for **core explorer skills** (1000000, 1000001,
  1100001, 1110001, 1120005 …) that unquestionably exist in v83.
- The 404s occur **only in a ~9-minute window (14:01–14:10 UTC, 2026-07-29)** then
  stop completely — a single count-over-time spike, then flat zero.
- Direct live queries afterward: **every stored preset skillId across every
  version (v48/61/72/79/83/84) resolves 200**. Nothing is persistently missing.
  (v87/v92 have no presets; v95 has an empty presets array.)

So while atlas-data replaced v83's SKILL documents during the re-publish, GETs
for those skills briefly 404'd; once ingest finished they resolved.

Two secondary facts:

1. **Why the Jobs page looked fine:** `useJobSkillDefinitions` reports an error
   only if *every* skill fails (`useJobSkillDefinitions.ts:44`), silently
   swallowing a few transient 404s. The preset page renders each skill in its
   own `useSkillData` row and surfaces every 404 individually.
2. **The re-publish has a 404 window at all** — "publish baseline" makes existing
   skill documents momentarily un-fetchable rather than swapping atomically.
   Backend concern; see Issue 2b.

### Fix (Issue 2a — UI tolerates transient 404s)

`skillDefinitionRetry` (shared by `useSkillDefinition` and
`useJobSkillDefinitions`) treated a 404 as terminal (never retried), so a
transient re-ingest 404 was cached and the row stayed "Unknown skill" until a
manual reload. Changed it to retry a 404 a **bounded** number of times
(`SKILL_DEFINITION_404_MAX_RETRIES = 2`, default exponential backoff): a
re-ingest blip self-heals without a reload, while a genuinely-invalid id still
gives up quickly and does not hammer the backend. Non-404 errors keep the
original three-attempt budget. The row's graceful "Unknown skill" + Sparkles
fallback (no crash/error card) is unchanged.

### Issue 2b — atlas-data publish gap (investigation)

Separate backend investigation into why "publish baseline" leaves a 404 window
(delete-then-insert vs atomic swap vs cache-invalidation gap) and the minimal
fix. Findings appended below / tracked as follow-up.

## Verification

- `npx vitest run` — 1358 passed (189 files).
- `npm run build` — clean.
- ESLint + Prettier — clean on all changed/new files.
