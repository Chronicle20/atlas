# bug: the corrected Maple Life seed data never reaches an already-seeded database

Task: task-246-maple-life-character-creation
PR: atlas-pr-1466
Namespace under test: `atlas-pr-1466`
Client/tenant: GMS 83.1, tenant `947e7bf0-8835-4c42-8b0a-3e052cecdc45`

Follow-up to `bug-ap-sp-and-starting-equipment.md`, whose fix (`47dc7bf00`) is
present and correct on the branch but had no effect in live testing.

## Reproduced

User created "Ralph", a Warrior (ordinal 0), on `atlas-pr-1466` after the fix
was deployed. Observed **114 unused AP** (expected 123), **77 unused SP** with
10 spent on Improved MaxHP (expected 51 = 61 − 10), and the pre-fix starting
equipment (Beginner apparel + a level-10 sword).

Every observed number is exactly the **pre-fix** seed value.

## Observed

The deployed image ships the corrected file — inside
`atlas-configurations-6c6c744bb7-8qbkq`:

```
$ grep -o '"ap"[^,]*' /seed-data/templates/template_gms_83_1.json | sort | uniq -c
      2 "ap": 123
      4 "ap": 133
      4 "ap": 138
```

But both database rows still hold the pre-fix document.

`GET /api/configurations/templates` — GMS 83/87/92/95 all show
`(ordinal 0, gender 0, ap 114)`.

`GET /api/configurations/tenants` — tenant `947e7bf0…`, all 10 classes:

```
ord 0 g 0 ap 114 sp 87,0,...  [1040002, 1060002, 1072001, 1302077]
ord 1 g 0 ap 129 sp 87,0,...  [1040002, 1060002, 1072001, 1372043]
ord 2 g 0 ap 124 sp 87,0,...  [1040002, 1060002, 1072001, 1452051]
ord 3 g 0 ap 124 sp 87,0,...  [1040002, 1060002, 1072001, 1332063]
ord 4 g 0 ap 129 sp 87,0,...  [1040002, 1060002, 1072001, 1482000, 1492000]
```

`ap 114` for ordinal 0 and `1302077` (level-10 sword) match the user's report
exactly.

## Expected

A character created after the fix deployed should receive the corrected
`ap`/`sp`/`equipment`, i.e. Warrior 123 AP / 61 SP / `1040021, 1060016,
1072039, 1302008, 1442001, 1422001, 1312005`.

## Root cause

**Not a code defect on the branch. A data-propagation gap, by design.**

1. The template seeder is **create-if-absent, deliberately**
   (`services/atlas-configurations/atlas.com/configurations/seeder/seeder.go:140-148`,
   FR-4.1): an existing region/version key is `skipped` regardless of whether
   the shipped file differs. Reconciling on boot would discard operator edits,
   so drift correction is an explicit operator action —
   `POST /configurations/templates/{id}/reseed`
   (`templates/resource.go:32`, `templates/processor.go:248`). The
   `atlas-pr-1466` template rows were seeded from an **earlier commit of this
   same branch**, when `mapleLife` first landed with the pre-fix numbers, so
   redeploying `47dc7bf00` changed nothing.

2. Tenant configurations are a **separate table with their own copy** of the
   `mapleLife` block; `atlas-configurations` has no template→tenant
   re-derivation path (no reference to templates anywhere in
   `configurations/tenants/`). `atlas-character-factory` reads the **tenant**
   document (`configuration/tenant/rest.go:21`,
   `factory/processor.go:377-405`), so even a reseeded template would not have
   changed the created character. The tenant row must be updated with
   `PATCH /configurations/tenants/{tenantId}` (a full-document replace,
   `tenants/resource.go:78`).

This applies to **every** already-seeded environment, `atlas-main` included:
merging this PR will not by itself put the corrected — or any — `mapleLife`
block into a live tenant. It is a deploy step, not a code change.

## Fix

Operational, performed against `atlas-pr-1466` from this session; no source
file changes.

1. `POST /api/configurations/templates/{id}/reseed` for GMS 83.1
   (`5ceee7ed-8159-46fc-9f9d-75cfe4ffd5e9`), 87.1 (`ad86ff79-de17-4776-aa4c-606b6f35b117`),
   92.1 (`042cb561-32c6-4c88-bb5f-3fde73c39d32`), 95.1 (`a6b17474-df6d-48f1-b136-7039feaf06b3`).
2. `GET` tenant `947e7bf0…`, replace its `mapleLife` block with the reseeded
   GMS 83.1 template's `mapleLife`, `PATCH` it back, and re-read to confirm
   `ap 123 / sp 61 / 1040021…` on ordinal 0.
3. Live re-test by the user.

### Files

None. The repository is already correct as of `47dc7bf00`.

## Not yet answered

- Whether the PR should carry a documented deploy/migration note so the
  `mapleLife` block reaches `atlas-main`'s tenant after merge. Raised with the
  user; not decided here.

## Resolution

**Remediation performed against `atlas-pr-1466` on 2026-08-27, from this
session. No commit fixes this — the code was never wrong.**

1. Reseeded all four templates — `POST .../templates/{id}/reseed` returned
   **204** for GMS 83.1, 87.1, 92.1, 95.1. Note the `ENVIRONMENT` header must
   be **omitted**: this deployment sets no `ATLAS_ENVIRONMENT`, so it is a
   legacy (`""`) caller and `ENVIRONMENT: pr-1466` is rejected **400**.
2. Re-read GMS 83.1 — `ap 123 / sp "61,0,…" / [1040021, 1060016, 1072039,
   1302008, 1442001, 1422001, 1312005]` on ordinal 0.
3. PATCHed tenant `947e7bf0…` with that template's `mapleLife` block spliced
   into its own document — **204**. Re-read confirms all 10 rows now match the
   `## Fix` tables of `bug-ap-sp-and-starting-equipment.md`
   (123/138/133/133/138, `sp 61`, HeavenMS/Cosmic equipment, Thief `2070000
   ×500`, Pirate `2330000 ×800`). A field-by-field diff of the tenant document
   before vs. after shows **no non-`mapleLife` attribute changed**.
4. `atlas-character-factory` consumed the resulting
   `EVENT_TOPIC_CONFIGURATION_TENANT_STATUS` message at `10:28:20.912Z`; the
   payload in its log carries `"ap":123 ×2, 133 ×4, 138 ×4`. The running
   projection is current — no pod restart needed.

**Live re-test — owed by the user.** Create a Warrior (expect 123 unused AP,
61 SP before spending, HeavenMS starting kit) and ideally a Bowman or Thief
(expect 133) on `atlas-pr-1466`.

**Open for the PR:** `atlas-main` will be in exactly this state after merge —
its template rows already exist, so the seeder will skip them and no tenant
will ever receive a `mapleLife` block. Steps 1–3 above are the deploy
procedure and should be written into the PR description or a task note.
