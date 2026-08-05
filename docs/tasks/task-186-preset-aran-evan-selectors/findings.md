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

### Issue 2b — atlas-data 404 window (investigation result)

Traced all three stages of the operator workflow. The destructive step is
**restore (apply-baseline), not publish or ingest**:

| Stage | File | DB behavior | 404 window? |
|---|---|---|---|
| Ingest (re-process WZ) | `document/db_storage.go:144` | UPSERT (ON CONFLICT DO UPDATE) | No |
| Publish baseline | `baseline/publish.go:39-116` | read-only COPY-OUT → MinIO | No |
| **Restore baseline** | `baseline/restore.go:87-102` | **DELETE-then-COPY** | **Yes** |

**Root cause — a non-atomic DELETE+COPY.** `restoreOneTable`
(`baseline/restore.go:87-102`) wraps `DELETE FROM documents WHERE tenant_id = ?`
plus the repopulating COPY in a `db.Transaction(...)` — *intending* an atomic
swap. But `copyInBinary` (`restore.go:337-359`) calls `tx.DB()` →
`sqlDB.Conn(ctx)`, which checks out a **different pooled connection** than the
transaction. GORM's `DB.DB()` (`gorm.io/gorm/gorm.go:426-433`) reflects the pool
`*sql.DB` back out of the `*sql.Tx` — it does **not** hand back the tx's own
connection. So the `DELETE` runs in transaction *T* on one connection while the
COPY runs on an independent autocommit connection. The two are not one MVCC unit,
so a reader can observe the target tenant's rows **deleted-but-not-yet-
repopulated** → `GET /api/data/skills/{id}` 404s (`skill/resource.go:105-108`).
This also makes each table's restore self-conflict on the unique key
(`idx_documents_tenant_type_docid`), consistent with the observed multi-minute
window rather than a sub-second flip.

- **Origin:** `DELETE` at `restore.go:93`, visible independently of the
  out-of-transaction COPY at `restore.go:359`.
- **Scope:** per-tenant, per-table (`runRestoreTables` loops table-by-table,
  each in its own transaction, `restore.go:64-85`). The whole restore is also
  not one transaction, and `cleanupAfterFailure` (`restore.go:116-124`) can
  DELETE all tenant rows on failure — a second, longer 404 path.

**Recommended minimal fix (backend, atlas-data):** run the per-table DELETE and
COPY on the **same** connection so MVCC hides the swap — check out one
`*sql.Conn` and run `BEGIN; DELETE …; COPY … FROM STDIN; COMMIT` on it, instead
of `tx.DB()` + `sqlDB.Conn(ctx)`. Equivalent: COPY into a staging table on the
tx connection then `DELETE`+`INSERT … SELECT` within one transaction.

**Caveat:** the atomicity defect is proven from source (restore.go + gorm.go),
but the exact live interleaving that produced the specific ~9-min duration was
inferred from code, not reproduced against live Postgres. Verify against a live
restore before/while landing the backend fix.

**Status: FIXED on this branch.** `baseline/restore.go` — `restoreOneTable` no
longer wraps the DELETE in a gorm `db.Transaction` that lets the COPY escape onto
a second pooled connection. New `replaceTableBinary` checks out **one** pooled
connection and runs `BEGIN; DELETE …; COPY … FROM STDIN (FORMAT binary); COMMIT`
on it via the pgx conn, so the row replacement is a single MVCC unit — readers
never see the gap, and the re-apply self-conflict on
`idx_documents_tenant_type_docid` is gone (the COPY now sees the DELETE). Added
`TestRestoreTableSwapIsAtomic` (a source-structure guard, matching the file's
existing test style, since the pgx binary-COPY path only runs against real
Postgres, not the sqlite unit-test DB).

Verified: `go build`/`go vet`/`gofmt` clean; `go test -race ./...` — 33 packages
pass; `go.mod` untouched (pgx already a dep, no docker-bake needed);
goroutine-guard + `tools/lint.sh --check` clean.

**Runtime caveat still stands:** the atomicity defect and fix are reasoned from
source; the pgx COPY-in-transaction path is not exercised by unit tests (sqlite),
so confirm against a live baseline re-apply that the 404 window is gone before
relying on it in production.

## Verification

- `npx vitest run` — 1358 passed (189 files).
- `npm run build` — clean.
- ESLint + Prettier — clean on all changed/new files.
